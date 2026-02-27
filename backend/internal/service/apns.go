package service

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kickwatch/backend/internal/config"
)

type APNsClient struct {
	cfg        *config.Config
	httpClient *http.Client
	mu         sync.Mutex
	token      string
	tokenExpAt time.Time
	privKey    *ecdsa.PrivateKey
}

func NewAPNsClient(cfg *config.Config) (*APNsClient, error) {
	keyData, err := os.ReadFile(cfg.APNSKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read apns key: %w", err)
	}
	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("invalid pem block in apns key")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse apns key: %w", err)
	}
	ecKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("apns key is not ECDSA")
	}

	transport := &http.Transport{}
	_ = http.ProxyFromEnvironment

	return &APNsClient{
		cfg:        cfg,
		httpClient: &http.Client{Transport: transport, Timeout: 10 * time.Second},
		privKey:    ecKey,
	}, nil
}

func (a *APNsClient) bearerToken() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.token != "" && time.Now().Before(a.tokenExpAt) {
		return a.token, nil
	}

	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:   a.cfg.APNSTeamID,
		IssuedAt: jwt.NewNumericDate(now),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	t.Header["kid"] = a.cfg.APNSKeyID

	signed, err := t.SignedString(a.privKey)
	if err != nil {
		return "", fmt.Errorf("sign apns jwt: %w", err)
	}
	a.token = signed
	a.tokenExpAt = now.Add(45 * time.Minute)
	return signed, nil
}

type APNsPayload struct {
	APS struct {
		Alert struct {
			Title string `json:"title"`
			Body  string `json:"body"`
		} `json:"alert"`
		Badge int    `json:"badge,omitempty"`
		Sound string `json:"sound,omitempty"`
	} `json:"aps"`
	AlertID    string `json:"alert_id,omitempty"`
	MatchCount int    `json:"match_count,omitempty"`
}

func (a *APNsClient) Send(deviceToken string, payload APNsPayload) error {
	host := "https://api.push.apple.com"
	if a.cfg.APNSEnv == "sandbox" {
		host = "https://api.sandbox.push.apple.com"
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/3/device/%s", host, deviceToken)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	token, err := a.bearerToken()
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "bearer "+token)
	req.Header.Set("apns-topic", a.cfg.APNSBundleID)
	req.Header.Set("apns-push-type", "alert")
	req.Header.Set("Content-Type", "application/json")
	req.Body = http.NoBody
	req.ContentLength = int64(len(body))

	// Re-set body after NoBody assignment
	req2, _ := http.NewRequest("POST", url, jsonBody(body))
	req2.Header = req.Header.Clone()

	resp, err := a.httpClient.Do(req2)
	if err != nil {
		return fmt.Errorf("apns send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusGone {
		return fmt.Errorf("apns: device token invalid (410)")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("apns status %d", resp.StatusCode)
	}
	log.Printf("APNs sent to %s...", deviceToken[:min(8, len(deviceToken))])
	return nil
}

type byteReader struct {
	data   []byte
	offset int
}

func jsonBody(data []byte) *byteReader { return &byteReader{data: data} }

func (r *byteReader) Read(p []byte) (int, error) {
	if r.offset >= len(r.data) {
		return 0, fmt.Errorf("EOF")
	}
	n := copy(p, r.data[r.offset:])
	r.offset += n
	return n, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
