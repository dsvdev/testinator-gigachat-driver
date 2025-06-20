package llm

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

type GigachatDriver struct {
	token       string
	accessToken string
	client      *http.Client
}

func NewGigachatDriver() *GigachatDriver {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client := &http.Client{Transport: tr}

	d := &GigachatDriver{
		token:  os.Getenv("GIGACHAT_TOKEN"),
		client: client,
	}
	d.updateToken()
	go func() {
		timer := time.NewTicker(15 * time.Minute)
		for range timer.C {
			d.updateToken()
		}
	}()
	return d
}

func (d *GigachatDriver) SendRequest(prompt string) (string, error) {
	return d.sendRequest(prompt)
}

func (d *GigachatDriver) updateToken() {
	reqId := uuid.New().String()
	s, err := getToken(d.token, reqId)
	if err != nil {
		log.Printf("Failed to get token with reqId %s: %v", reqId, err)
	}
	d.accessToken = s
}

func getToken(authKey string, reqUID string) (string, error) {
	endpoint := "https://ngw.devices.sberbank.ru:9443/api/v2/oauth"

	form := url.Values{}
	form.Add("scope", "GIGACHAT_API_PERS")

	req, err := http.NewRequest("POST", endpoint, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("RqUID", reqUID)
	req.Header.Set("Authorization", "Basic "+authKey)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("non-200 response: %d, body: %s", resp.StatusCode, string(body))
	}

	// вытащим access_token из тела, если оно JSON:
	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	return tokenResp.AccessToken, nil
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model             string    `json:"model"`
	Messages          []Message `json:"messages"`
	N                 int       `json:"n"`
	Stream            bool      `json:"stream"`
	MaxTokens         int       `json:"max_tokens"`
	RepetitionPenalty float64   `json:"repetition_penalty"`
	UpdateInterval    int       `json:"update_interval"`
}

type Choice struct {
	Message Message `json:"message"`
}

type ChatResponse struct {
	Choices []Choice `json:"choices"`
}

func (d *GigachatDriver) sendRequest(prompt string) (string, error) {
	baseURL := "https://gigachat.devices.sberbank.ru/api/v1/chat/completions"

	payload := ChatRequest{
		Model: "GigaChat",
		Messages: []Message{
			{Role: "user", Content: prompt},
		},
		N:                 1,
		RepetitionPenalty: 1,
		MaxTokens:         512,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", baseURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+d.accessToken)

	resp, err := d.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("non-200 response: %d, body: %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("failed to parse chat response: %w", err)
	}

	return chatResp.Choices[0].Message.Content, nil
}
