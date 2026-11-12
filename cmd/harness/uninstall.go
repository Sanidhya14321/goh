package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Sanidhya14321/goh/internal/core"
)

// runUninstall implements `goh uninstall`, like `opencode uninstall`. Without
// --yes it only lists what would be removed. It removes this executable (when
// it is named goh) and, unless --keep-data, the GoHarness config folder (with
// saved keys and the ChatGPT sign-in, plus the sealing key in the OS
// credential store when one is used) and state folder (sessions). It never
// touches shell configuration or anything outside those named folders.
func runUninstall(args []string, w io.Writer) error {
	f := flag.NewFlagSet("uninstall", flag.ContinueOnError)
	yes := f.Bool("yes", false, "remove the listed files (default: only list them)")
	keep := f.Bool("keep-data", false, "keep config, saved credentials and sessions")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if f.NArg() != 0 {
		return errors.New("usage: goh uninstall [--keep-data] [--yes]")
	}
	exe, err := os.Executable()
	if err == nil {
		exe, err = filepath.EvalSymlinks(exe)
	}
	if err != nil {
		return err
	}
	targets := uninstallTargets(exe, *keep)
	forgetKey := !*keep && core.UsesCredentialKey()
	if len(targets) == 0 && !forgetKey {
		fmt.Fprintln(w, "Nothing to remove.")
		return nil
	}
	if !*yes {
		fmt.Fprintln(w, "goh uninstall would remove:")
		for _, t := range targets {
			fmt.Fprintln(w, "  "+t)
		}
		if forgetKey {
			fmt.Fprintln(w, "  the GoHarness credential key in the "+core.CredentialStoreName())
		}
		fmt.Fprintln(w, "Run goh uninstall --yes to remove them (add --keep-data to keep sessions, config and sign-in).")
		return nil
	}
	var failed error
	if forgetKey {
		// Before the folder goes, while the store choice is still readable.
		if removed, err := core.ForgetCredentialKey(); err != nil {
			failed = errors.Join(failed, err)
		} else if removed {
			fmt.Fprintln(w, "removed the GoHarness credential key from the "+core.CredentialStoreName())
		}
	}
	for _, t := range targets {
		if t == exe && runtime.GOOS == "windows" {
			fmt.Fprintln(w, "Windows cannot delete a running program; delete it after exit: "+t)
			continue
		}
		if err := os.RemoveAll(t); err != nil {
			failed = errors.Join(failed, err)
			continue
		}
		fmt.Fprintln(w, "removed "+t)
	}
	return failed
}

// uninstallTargets lists existing paths to remove. Data folders are included
// only when their last element is GoHarness's own folder name.
func uninstallTargets(exe string, keepData bool) []string {
	var out []string
	add := func(p string) {
		if _, err := os.Lstat(p); err == nil {
			out = append(out, p)
		}
	}
	if base := filepath.Base(exe); base == "goh" || base == "goh.exe" {
		add(exe)
		if link := filepath.Join(filepath.Dir(exe), "harness"); isLinkTo(link, exe) {
			add(link)
		}
	}
	if keepData {
		return out
	}
	if cfg, err := core.UserConfigPath(); err == nil && filepath.Base(filepath.Dir(cfg)) == "independent-harness" {
		add(filepath.Dir(cfg))
	}
	if state := filepath.Dir(core.Defaults().SessionDir); filepath.Base(state) == "independent-harness" {
		add(state)
	}
	return out
}

func isLinkTo(link, target string) bool {
	info, err := os.Lstat(link)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		return false
	}
	resolved, err := filepath.EvalSymlinks(link)
	if err != nil {
		return false
	}
	if t, err := filepath.EvalSymlinks(target); err == nil {
		target = t
	}
	return resolved == target
}
