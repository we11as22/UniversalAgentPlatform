package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

var client = &http.Client{Timeout: 30 * time.Second}

func GetJSON[T any](ctx context.Context, url string) (T, error) {
	var target T
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return target, err
	}
	response, err := client.Do(request)
	if err != nil {
		return target, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		body, _ := io.ReadAll(response.Body)
		return target, fmt.Errorf("request failed: %s %s", response.Status, string(body))
	}
	err = json.NewDecoder(response.Body).Decode(&target)
	return target, err
}

func PostJSON[TRequest any, TResponse any](ctx context.Context, url string, payload TRequest) (TResponse, error) {
	var target TResponse
	body, err := json.Marshal(payload)
	if err != nil {
		return target, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return target, err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return target, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		raw, _ := io.ReadAll(response.Body)
		return target, fmt.Errorf("request failed: %s %s", response.Status, string(raw))
	}
	err = json.NewDecoder(response.Body).Decode(&target)
	return target, err
}

