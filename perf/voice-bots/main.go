package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
)

func main() {
	baseURL := flag.String("base-url", "http://localhost:3270", "voice gateway base url")
	chatBaseURL := flag.String("chat-base-url", "http://localhost:3220", "chat gateway base url")
	tenantID := flag.String("tenant-id", "11111111-1111-1111-1111-111111111111", "tenant id")
	userID := flag.String("user-id", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1", "user id")
	agentID := flag.String("agent-id", "71000000-0000-0000-0000-000000000003", "agent id")
	concurrency := flag.Int("concurrency", 5, "concurrent sessions")
	flag.Parse()

	var waitGroup sync.WaitGroup
	for index := 0; index < *concurrency; index++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			conversationID, err := createConversation(*chatBaseURL, *userID, *agentID)
			if err != nil {
				log.Printf("conversation create failed: %v", err)
				return
			}

			payload, _ := json.Marshal(map[string]string{
				"tenant_id":       *tenantID,
				"user_id":         *userID,
				"conversation_id": conversationID,
				"agent_id":        *agentID,
			})
			response, err := http.Post(*baseURL+"/api/v1/voice/sessions", "application/json", bytes.NewReader(payload))
			if err != nil {
				log.Printf("voice session failed: %v", err)
				return
			}
			defer response.Body.Close()
			log.Printf("voice session status=%d", response.StatusCode)
		}()
	}
	waitGroup.Wait()
}

func createConversation(chatBaseURL string, userID string, agentID string) (string, error) {
	payload, _ := json.Marshal(map[string]string{
		"user_id":  userID,
		"agent_id": agentID,
		"title":    "voice-bot synthetic conversation",
	})
	response, err := http.Post(chatBaseURL+"/api/v1/conversations", "application/json", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		body, _ := io.ReadAll(response.Body)
		return "", fmt.Errorf("conversation create status=%d body=%s", response.StatusCode, string(body))
	}
	var decoded struct {
		ConversationID string `json:"conversation_id"`
	}
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return "", err
	}
	return decoded.ConversationID, nil
}
