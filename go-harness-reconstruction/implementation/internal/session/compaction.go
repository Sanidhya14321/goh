package session

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"example.com/independent-harness/internal/core"
)

// Compaction is the shared journal and portable representation of a working view.
type Compaction struct {
	Messages  []core.Message     `json:"messages"`
	Before    int                `json:"before_estimate"`
	After     int                `json:"after_estimate"`
	Estimator string             `json:"estimator"`
	Manual    bool               `json:"manual,omitempty"`
	Model     string             `json:"model,omitempty"`
	Profile   string             `json:"profile,omitempty"`
	Protocol  string             `json:"protocol,omitempty"`
	Receipt   *CompactionReceipt `json:"receipt,omitempty"`
}

type CompactionReceipt struct {
	Version      int                  `json:"version"`
	ViewSHA256   string               `json:"view_sha256"`
	ActiveSHA256 string               `json:"active_user_sha256,omitempty"`
	Evidence     []CompactionEvidence `json:"evidence,omitempty"`
}

type CompactionEvidence struct {
	Tool       string `json:"tool"`
	Path       string `json:"path,omitempty"`
	FileSHA256 string `json:"file_sha256,omitempty"`
	ExitCode   *int   `json:"exit_code,omitempty"`
}

func (c *Compaction) SealReceipt(evidence []CompactionEvidence) error {
	view, task, err := c.identities(evidence)
	if err != nil {
		return err
	}
	c.Receipt = &CompactionReceipt{Version: 1, ViewSHA256: view, ActiveSHA256: task, Evidence: evidence}
	return nil
}

func (c Compaction) VerifyReceipt() error {
	if c.Receipt == nil {
		return nil // Earlier journals have no receipt.
	}
	view, task, err := c.identities(c.Receipt.Evidence)
	if err != nil {
		return err
	}
	if c.Receipt.Version != 1 || c.Receipt.ViewSHA256 != view || c.Receipt.ActiveSHA256 != task {
		return errors.New("compaction receipt does not match the saved working view")
	}
	return nil
}

func (c Compaction) identities(evidence []CompactionEvidence) (view, task string, err error) {
	if len(c.Messages) == 0 || c.Messages[0].Role != "system" {
		return "", "", errors.New("compaction requires system framing")
	}
	// The system message is refreshed at resume from the current trusted config.
	data, err := json.Marshal(struct {
		Messages []core.Message       `json:"messages"`
		Evidence []CompactionEvidence `json:"evidence,omitempty"`
	}{c.Messages[1:], evidence})
	if err != nil {
		return "", "", err
	}
	for i := len(c.Messages) - 1; i > 0; i-- {
		if c.Messages[i].Role == "user" {
			hash := sha256.Sum256([]byte(c.Messages[i].Content))
			task = hex.EncodeToString(hash[:])
			break
		}
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), task, nil
}
