package core

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Sanidhya14321/goh/internal/platform"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	ChatGPTProtocol = "openai_chatgpt"
	ChatGPTEndpoint = "https://api.openai.com/v1/responses"
	chatGPTResource = "https://api.openai.com/v1"
	chatGPTIssuer   = "https://auth.openai.com"
	chatGPTScope    = "openid profile email offline_access resource.invoke chatgpt.tokens.use.direct"
)

var ErrChatGPTLogin = errors.New("ChatGPT sign-in required; use /connect or goh auth login")

type chatGPTTokens struct {
	Access   string `json:"access_token"`
	Refresh  string `json:"refresh_token"`
	Identity string `json:"id_token"`
	Type     string `json:"token_type"`
	Scope    string `json:"scope"`
	Lifetime int64  `json:"expires_in"`
}

type chatGPTAccount struct {
	Tokens   chatGPTTokens `json:"tokens"`
	ClientID string        `json:"client_id"`
	Subject  string        `json:"subject"`
	Expires  time.Time     `json:"expires"`
	Model    string        `json:"model"`
}

func chatGPTPath(name string) (string, error) {
	cfg, err := UserConfigPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(cfg), "credentials", name), nil
}

// ErrChatGPTDamaged reports a saved ChatGPT credential file that cannot be
// used. Sign-in replaces it and sign-out removes it.
var ErrChatGPTDamaged = errors.New("saved ChatGPT credentials are unusable")

const chatGPTAccountFile = "chatgpt-account.json"

func readChatGPTAccount() (chatGPTAccount, error) {
	var account chatGPTAccount
	path, err := chatGPTPath(chatGPTAccountFile)
	if err != nil {
		return account, err
	}
	var src io.Reader
	if inKeychain(path) {
		b, err := keychainSecret(path)
		if err != nil {
			return account, fmt.Errorf("%w: %v", ErrChatGPTDamaged, err)
		}
		src = strings.NewReader(string(b))
	} else {
		f, err := os.OpenFile(path, os.O_RDONLY|platform.NoFollow|platform.NonBlock, 0)
		if os.IsNotExist(err) {
			return account, ErrChatGPTLogin
		}
		if err != nil {
			return account, fmt.Errorf("%w: could not open %s; run `goh auth login` (or /connect) to replace it", ErrChatGPTDamaged, path)
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil || !info.Mode().IsRegular() || !platform.PrivateFile(info, 0600) || info.Size() > 1<<20 {
			return account, fmt.Errorf("%w: %s must be a private regular file (mode 0600); run `goh auth login` (or /connect) to replace it", ErrChatGPTDamaged, path)
		}
		src = f
	}
	if json.NewDecoder(io.LimitReader(src, 1<<20)).Decode(&account) != nil ||
		account.ClientID == "" || account.Subject == "" || !hasPlanScope(account.Tokens.Scope) {
		return account, fmt.Errorf("%w: invalid contents; sign in again with `goh auth login` or /connect", ErrChatGPTDamaged)
	}
	return account, nil
}

func saveChatGPTAccount(a chatGPTAccount) error {
	path, err := chatGPTPath(chatGPTAccountFile)
	if err != nil {
		return err
	}
	b, err := json.Marshal(a)
	if err != nil {
		return err
	}
	return saveSecret(path, b)
}

func ChatGPTModel() (string, error) {
	a, err := readChatGPTAccount()
	return a.Model, err
}

func chatGPTLock() (func(), error) {
	path, err := chatGPTPath("chatgpt.lock")
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|platform.NoFollow|platform.NonBlock, 0600)
	if err != nil {
		return nil, err
	}
	if err := platform.TryLock(f); err != nil {
		f.Close()
		return nil, errors.New("another GoHarness process is updating ChatGPT sign-in; try again shortly")
	}
	return func() { _ = f.Close() }, nil
}

func chatGPTRequest(ctx context.Context, method, endpoint string, form url.Values, key string, dst any) error {
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return errors.New("invalid ChatGPT request")
	}
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	client := http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("could not reach OpenAI's sign-in service")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("OpenAI account request returned HTTP %d", resp.StatusCode)
	}
	if dst == nil {
		return nil
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if err != nil || len(data) > 1<<20 || json.Unmarshal(data, dst) != nil {
		return errors.New("invalid response from OpenAI's sign-in service")
	}
	return nil
}

func hasPlanScope(scope string) bool {
	return strings.Contains(" "+strings.Join(strings.Fields(scope), " ")+" ", " chatgpt.tokens.use.direct ")
}

func ChatGPTAccessToken(ctx context.Context) (string, error) {
	a, err := readChatGPTAccount()
	if err != nil {
		return "", err
	}
	if time.Until(a.Expires) > 30*time.Second && a.Tokens.Access != "" {
		return a.Tokens.Access, nil
	}
	unlock, err := chatGPTLock()
	if err != nil {
		return "", err
	}
	defer unlock()
	a, err = readChatGPTAccount()
	if err != nil {
		return "", err
	}
	if time.Until(a.Expires) > 30*time.Second && a.Tokens.Access != "" {
		return a.Tokens.Access, nil
	}
	return renewChatGPTLocked(ctx, a)
}

// RenewChatGPTAccessToken renews after the service rejected the rejected
// token (HTTP 401) before its recorded expiry. If another process already
// renewed it, the newer saved token is returned without a second renewal.
func RenewChatGPTAccessToken(ctx context.Context, rejected string) (string, error) {
	unlock, err := chatGPTLock()
	if err != nil {
		return "", err
	}
	defer unlock()
	a, err := readChatGPTAccount()
	if err != nil {
		return "", err
	}
	if a.Tokens.Access != "" && a.Tokens.Access != rejected && time.Until(a.Expires) > 30*time.Second {
		return a.Tokens.Access, nil
	}
	return renewChatGPTLocked(ctx, a)
}

// renewChatGPTLocked exchanges the refresh token; the caller holds the lock.
func renewChatGPTLocked(ctx context.Context, a chatGPTAccount) (string, error) {
	if a.Tokens.Refresh == "" {
		return "", ErrChatGPTLogin
	}
	var next chatGPTTokens
	err := chatGPTRequest(ctx, http.MethodPost, chatGPTIssuer+"/api/accounts/oauth/token", url.Values{
		"grant_type": {"refresh_token"}, "client_id": {a.ClientID},
		"refresh_token": {a.Tokens.Refresh}, "resource": {chatGPTResource},
	}, "", &next)
	if err != nil {
		return "", fmt.Errorf("ChatGPT renewal failed; reconnect with /connect if it persists: %w", err)
	}
	if next.Scope == "" {
		next.Scope = a.Tokens.Scope
	}
	if next.Refresh == "" {
		next.Refresh = a.Tokens.Refresh
	}
	// Retain the previously verified identity token for later authorization.
	next.Identity = a.Tokens.Identity
	if !validChatGPTTokens(next) {
		return "", ErrChatGPTLogin
	}
	a.Tokens, a.Expires = next, time.Now().Add(time.Duration(next.Lifetime)*time.Second)
	if err := saveChatGPTAccount(a); err != nil {
		return "", err
	}
	return next.Access, nil
}

func validChatGPTTokens(t chatGPTTokens) bool {
	return strings.EqualFold(t.Type, "Bearer") && t.Access != "" && t.Lifetime > 0 &&
		t.Lifetime <= 30*24*60*60 && hasPlanScope(t.Scope)
}

func LogoutChatGPT(ctx context.Context) error {
	unlock, err := chatGPTLock()
	if err != nil {
		return err
	}
	defer unlock()
	a, err := readChatGPTAccount()
	if errors.Is(err, ErrChatGPTLogin) {
		return nil
	}
	if errors.Is(err, ErrChatGPTDamaged) {
		// Nothing readable to revoke; removing the file is all that is left.
		a, err = chatGPTAccount{}, nil
	}
	if err != nil {
		return err
	}
	if a.Tokens.Refresh != "" {
		if err := chatGPTRequest(ctx, http.MethodPost, chatGPTIssuer+"/api/accounts/oauth/revoke", url.Values{
			"token": {a.Tokens.Refresh}, "token_type_hint": {"refresh_token"}, "client_id": {a.ClientID},
		}, "", nil); err != nil {
			return err
		}
	}
	path, err := chatGPTPath(chatGPTAccountFile)
	if err != nil {
		return err
	}
	return removeSecret(path)
}

// LoginChatGPT runs the documented public-client loopback flow. open receives
// only the authorization URL. The callback and tokens stay inside the process.
func LoginChatGPT(ctx context.Context, open func(string) error) (string, error) {
	unlock, err := chatGPTLock()
	if err != nil {
		return "", err
	}
	defer unlock()
	previous, err := readChatGPTAccount()
	if errors.Is(err, ErrChatGPTDamaged) {
		previous = chatGPTAccount{} // replaced by this sign-in
	} else if err != nil && !errors.Is(err, ErrChatGPTLogin) {
		return "", err
	}
	hostPath, err := chatGPTPath("chatgpt-host")
	if err != nil {
		return "", err
	}
	host, err := readConfig(hostPath)
	if os.IsNotExist(err) {
		b := make([]byte, 16)
		if _, err := rand.Read(b); err != nil {
			return "", err
		}
		b[6], b[8] = b[6]&15|64, b[8]&63|128
		h := hex.EncodeToString(b)
		host = []byte("urn:uuid:" + h[:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:])
		err = writePrivate(hostPath, string(host))
	}
	if err != nil {
		return "", err
	}
	random := make([]byte, 96)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	enc := base64.RawURLEncoding
	state, nonce, verifier := enc.EncodeToString(random[:32]), enc.EncodeToString(random[32:64]), enc.EncodeToString(random[64:])
	challenge := sha256.Sum256([]byte(verifier))
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer listener.Close()
	redirect := "http://" + listener.Addr().String() + "/auth/callback"
	clientID := previous.ClientID
	if clientID == "" {
		clientID = "dynamic_agent_client"
	}
	params := url.Values{
		"client_id": {clientID}, "ext_agent_host_id": {string(host)}, "response_type": {"code"},
		"redirect_uri": {redirect}, "scope": {chatGPTScope}, "resource": {chatGPTResource},
		"state": {state}, "nonce": {nonce}, "code_challenge_method": {"S256"},
		"code_challenge": {enc.EncodeToString(challenge[:])},
	}
	if previous.ClientID == "" {
		params.Set("agent_name_hint", "GoHarness")
	}

	callback := make(chan url.Values, 1)
	server := &http.Server{ReadHeaderTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second, MaxHeaderBytes: 16 << 10,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			q := r.URL.Query()
			if r.Method != http.MethodGet || r.Host != listener.Addr().String() ||
				r.URL.Path != "/auth/callback" || subtle.ConstantTimeCompare([]byte(q.Get("state")), []byte(state)) != 1 {
				http.Error(w, "Invalid sign-in callback", http.StatusBadRequest)
				return
			}
			w.Header().Set("Cache-Control", "no-store")
			select {
			case callback <- q:
				_, _ = io.WriteString(w, "GoHarness received the sign-in response. Return to the terminal.")
			default:
				http.Error(w, "Sign-in callback already received", http.StatusConflict)
			}
		}),
	}
	defer server.Close()
	go func() { _ = server.Serve(listener) }()
	if err := open(chatGPTIssuer + "/api/accounts/authorize?" + params.Encode()); err != nil {
		return "", err
	}
	var q url.Values
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case q = <-callback:
	}
	if q.Get("error") != "" {
		return "", errors.New("ChatGPT authorization was declined or unavailable")
	}
	issued := q.Get("client_id")
	if previous.ClientID != "" {
		if issued != "" && issued != previous.ClientID {
			return "", errors.New("ChatGPT registration changed during sign-in")
		}
		issued = previous.ClientID
	}
	if !strings.HasPrefix(issued, "oaiapp_") || q.Get("code") == "" {
		return "", errors.New("ChatGPT registration did not return a client and code")
	}
	var tokens chatGPTTokens
	if err := chatGPTRequest(ctx, http.MethodPost, chatGPTIssuer+"/api/accounts/oauth/token", url.Values{
		"grant_type": {"authorization_code"}, "client_id": {issued}, "code": {q.Get("code")},
		"code_verifier": {verifier}, "redirect_uri": {redirect}, "resource": {chatGPTResource},
	}, "", &tokens); err != nil {
		return "", err
	}
	sub, err := verifyChatGPTIdentity(ctx, tokens.Identity, issued, nonce)
	if err != nil {
		return "", err
	}
	if previous.Subject != "" && sub != previous.Subject {
		return "", errors.New("ChatGPT account changed during reauthorization")
	}
	if !validChatGPTTokens(tokens) {
		return "", errors.New("ChatGPT plan inference was not granted")
	}
	a := chatGPTAccount{Tokens: tokens, ClientID: issued, Subject: sub,
		Expires: time.Now().Add(time.Duration(tokens.Lifetime) * time.Second), Model: previous.Model}
	if models, err := listChatGPTModels(ctx, tokens.Access); err == nil {
		a.Model = ""
		for _, model := range models {
			if a.Model == "" || model.Slug == previous.Model {
				a.Model = model.Slug
			}
		}
	}
	if err := saveChatGPTAccount(a); err != nil {
		return "", err
	}
	return a.Model, nil
}

type ChatGPTModelInfo struct {
	Slug       string `json:"slug"`
	Name       string `json:"display_name"`
	Visibility string `json:"visibility"`
	// ContextWindow is the model's usable context in tokens, as reported by
	// the account's model catalog (0 when not reported).
	ContextWindow int `json:"context_window"`
}

func ListChatGPTModels(ctx context.Context) ([]ChatGPTModelInfo, error) {
	token, err := ChatGPTAccessToken(ctx)
	if err != nil {
		return nil, err
	}
	return listChatGPTModels(ctx, token)
}

func listChatGPTModels(ctx context.Context, token string) ([]ChatGPTModelInfo, error) {
	var catalog struct{ Models []ChatGPTModelInfo }
	if err := chatGPTRequest(ctx, http.MethodGet, chatGPTResource+"/models", nil, token, &catalog); err != nil {
		return nil, err
	}
	var models []ChatGPTModelInfo
	seen := map[string]bool{}
	for _, model := range catalog.Models {
		if model.Visibility == "list" && model.Slug != "" && len(model.Slug) <= 256 && !seen[model.Slug] {
			models = append(models, model)
			seen[model.Slug] = true
		}
	}
	if len(models) == 0 {
		return nil, errors.New("the ChatGPT account returned no selectable models")
	}
	return models, nil
}

func verifyChatGPTIdentity(ctx context.Context, token, clientID, nonce string) (string, error) {
	invalid := errors.New("OpenAI ID token did not pass verification")
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", invalid
	}
	enc := base64.RawURLEncoding
	head, e1 := enc.DecodeString(parts[0])
	claims, e2 := enc.DecodeString(parts[1])
	signature, e3 := enc.DecodeString(parts[2])
	var header struct{ Alg, Kid string }
	if e1 != nil || e2 != nil || e3 != nil || json.Unmarshal(head, &header) != nil || header.Alg != "RS256" || header.Kid == "" {
		return "", invalid
	}
	var jwks struct {
		Keys []struct{ Kid, Kty, N, E string }
	}
	if err := chatGPTRequest(ctx, http.MethodGet, chatGPTIssuer+"/.well-known/jwks.json", nil, "", &jwks); err != nil {
		return "", err
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	verified := false
	for _, key := range jwks.Keys {
		if key.Kid != header.Kid || key.Kty != "RSA" {
			continue
		}
		n, nerr := enc.DecodeString(key.N)
		e, eerr := enc.DecodeString(key.E)
		exp := new(big.Int).SetBytes(e)
		if nerr != nil || eerr != nil || len(n) < 256 || len(n) > 1024 || exp.BitLen() > 31 || exp.Int64() < 3 {
			continue
		}
		pub := rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(exp.Int64())}
		if rsa.VerifyPKCS1v15(&pub, crypto.SHA256, digest[:], signature) == nil {
			verified = true
			break
		}
	}
	var payload struct {
		Iss, Sub, Nonce, Azp string
		Aud                  json.RawMessage
		Exp, Iat, Nbf        int64
	}
	if !verified || json.Unmarshal(claims, &payload) != nil {
		return "", invalid
	}
	var audience []string
	if json.Unmarshal(payload.Aud, &audience) != nil {
		var one string
		if json.Unmarshal(payload.Aud, &one) != nil {
			return "", invalid
		}
		audience = []string{one}
	}
	now := time.Now().Unix()
	if payload.Iss != chatGPTIssuer || payload.Sub == "" || payload.Nonce != nonce ||
		payload.Exp <= now-5 || payload.Iat <= 0 || payload.Iat > now+5 || payload.Nbf > now+5 ||
		(len(audience) != 1 && payload.Azp != clientID) ||
		(payload.Azp != "" && payload.Azp != clientID) {
		return "", invalid
	}
	for _, aud := range audience {
		if aud == clientID {
			return payload.Sub, nil
		}
	}
	return "", invalid
}
