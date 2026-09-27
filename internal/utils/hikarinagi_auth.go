package utils

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Hikarinagi 公共客户端授权（授权码 + PKCE 流程）
// 文档: https://www.hikarinagi.org/developers/guide/user-flow
const (
	hikarinagiAuthURL  = "https://id.hikarinagi.org/oidc/auth"
	hikarinagiScope    = "catalog:full offline_access"
	hikarinagiRedirect = "http://127.0.0.1:28282" // 须与控制台登记完全一致
)

// HikarinagiAuthResult OAuth 授权结果
type HikarinagiAuthResult struct {
	Code  string
	State string
	Error string
}

// 全局授权回调通道
var (
	hikarinagiAuthChan chan HikarinagiAuthResult
	hikarinagiServer   *http.Server
	hikarinagiAuthMu   sync.Mutex
)

// GeneratePKCE 生成 PKCE code_verifier 与 S256 code_challenge
// Hikarinagi ID 仅接受 S256
func GeneratePKCE() (verifier string, challenge string, err error) {
	bytes := make([]byte, 32)
	if _, err = rand.Read(bytes); err != nil {
		return "", "", fmt.Errorf("生成 PKCE 随机数失败: %w", err)
	}
	verifier = base64.RawURLEncoding.EncodeToString(bytes)
	digest := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(digest[:])
	return verifier, challenge, nil
}

// GetHikarinagiAuthURL 构建 Hikarinagi ID 授权 URL
func GetHikarinagiAuthURL(verifier string) (string, error) {
	challenge, err := pkceChallenge(verifier)
	if err != nil {
		return "", err
	}

	params := url.Values{
		"response_type":         {"code"},
		"client_id":             {hikarinagiClientID},
		"redirect_uri":          {hikarinagiRedirect},
		"scope":                 {hikarinagiScope},
		"prompt":                {"consent"}, // offline_access 必须与 prompt=consent 同时出现才会签发 refresh token
		"state":                 {verifier},  // 复用 verifier 随机串作为 state，回调时逐字比对
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}
	return hikarinagiAuthURL + "?" + params.Encode(), nil
}

func pkceChallenge(verifier string) (string, error) {
	digest := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(digest[:]), nil
}

// StartHikarinagiAuthServer 启动本地回调服务器，等待授权回调并校验 state
func StartHikarinagiAuthServer(ctx context.Context, timeout time.Duration, expectState string) (string, error) {
	hikarinagiAuthMu.Lock()
	defer hikarinagiAuthMu.Unlock()

	hikarinagiAuthChan = make(chan HikarinagiAuthResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleHikarinagiCallback)

	listener, err := net.Listen("tcp", "127.0.0.1:28282")
	if err != nil {
		return "", fmt.Errorf("无法启动本地回调服务器: %w", err)
	}

	hikarinagiServer = &http.Server{Handler: mux}

	go func() {
		if err := hikarinagiServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			hikarinagiAuthChan <- HikarinagiAuthResult{Error: err.Error()}
		}
	}()

	select {
	case result := <-hikarinagiAuthChan:
		hikarinagiServer.Shutdown(context.Background())
		if result.Error != "" {
			return "", fmt.Errorf("授权失败: %s", result.Error)
		}
		if result.State != expectState {
			return "", fmt.Errorf("state 校验失败，疑似 CSRF，已中止")
		}
		return result.Code, nil
	case <-time.After(timeout):
		hikarinagiServer.Shutdown(context.Background())
		return "", fmt.Errorf("授权超时")
	case <-ctx.Done():
		hikarinagiServer.Shutdown(context.Background())
		return "", ctx.Err()
	}
}

// handleHikarinagiCallback 处理 OAuth 回调
func handleHikarinagiCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errorMsg := r.URL.Query().Get("error")
	errorDesc := r.URL.Query().Get("error_description")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if errorMsg != "" {
		hikarinagiAuthChan <- HikarinagiAuthResult{Error: fmt.Sprintf("%s: %s", errorMsg, errorDesc)}
		fmt.Fprintf(w, `<!DOCTYPE html><html><head><title>授权失败</title></head><body>
			<h1>授权失败</h1><p>%s: %s</p><p>您可以关闭此窗口。</p>
			</body></html>`, errorMsg, errorDesc)
		return
	}

	if code == "" {
		hikarinagiAuthChan <- HikarinagiAuthResult{Error: "未收到授权码"}
		fmt.Fprint(w, `<!DOCTYPE html><html><head><title>授权失败</title></head><body>
			<h1>授权失败</h1><p>未收到授权码</p><p>您可以关闭此窗口。</p>
			</body></html>`)
		return
	}

	hikarinagiAuthChan <- HikarinagiAuthResult{Code: code, State: state}
	fmt.Fprint(w, `<!DOCTYPE html><html><head><title>授权成功</title></head><body>
		<h1>授权成功！</h1><p>您可以关闭此窗口并返回应用。</p>
		<script>window.close();</script>
		</body></html>`)
}

type HikarinagiTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	Error        string `json:"error,omitempty"`
	ErrorDesc    string `json:"error_description,omitempty"`
}

// ExchangeHikarinagiCode 用授权码换取令牌（公共客户端，仅传 client_id，不持有密钥）
func ExchangeHikarinagiCode(ctx context.Context, code, verifier string) (*HikarinagiTokenResponse, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {hikarinagiClientID},
		"code":          {code},
		"redirect_uri":  {hikarinagiRedirect},
		"code_verifier": {verifier},
	}
	return hikarinagiTokenRequest(ctx, data)
}

// RefreshHikarinagiToken 用刷新令牌续期（每次刷新均会轮换，调用方须覆盖保存新值）
func RefreshHikarinagiToken(ctx context.Context, refreshToken string) (*HikarinagiTokenResponse, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {hikarinagiClientID},
		"refresh_token": {refreshToken},
	}
	return hikarinagiTokenRequest(ctx, data)
}

func hikarinagiTokenRequest(ctx context.Context, data url.Values) (*HikarinagiTokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", hikarinagiTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Saramanda9988/LunaBox/1.4.0 (desktop) (https://github.com/Saramanda9988/LunaBox)")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("Error closing hikarinagi token response body: %v\n", err)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tokenResp HikarinagiTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("解析令牌响应失败: %w, body: %s", err, string(body))
	}

	if tokenResp.Error != "" {
		return nil, fmt.Errorf("OAuth 错误 %s: %s", tokenResp.Error, tokenResp.ErrorDesc)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hikarinagi token API returned status: %d, body: %s", resp.StatusCode, string(body))
	}

	return &tokenResp, nil
}
