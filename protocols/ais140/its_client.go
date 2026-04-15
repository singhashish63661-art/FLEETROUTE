package ais140

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ITSClient forwards emergency packets to India's ITS backend portal.
// Integration is stubbed — configure via VLTD certificate and portal endpoint.
type ITSClient struct {
	endpoint string
	apiKey   string
	client   *http.Client
}

func NewITSClient(endpoint, apiKey string) *ITSClient {
	return &ITSClient{
		endpoint: endpoint,
		apiKey:   apiKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ForwardEmergency forwards a raw AIS140 emergency packet to the ITS portal.
// TODO: Implement full ITS portal integration with VLTD certificate credentials.
// Reference: https://its.example.gov.in/api/emergency (replace with actual endpoint)
func (c *ITSClient) ForwardEmergency(rawPacket string) error {
	if c.endpoint == "" || c.endpoint == "https://its.example.gov.in/api/emergency" {
		// Stub: log the emergency but don't forward
		fmt.Printf("[AIS140] STUB: emergency alert would be forwarded: %s\n", rawPacket)
		return nil
	}

	body := strings.NewReader(rawPacket)
	req, err := http.NewRequest(http.MethodPost, c.endpoint, body)
	if err != nil {
		return fmt.Errorf("its forward request: %w", err)
	}
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Content-Type", "text/plain")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("its forward: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("its forward status: %d", resp.StatusCode)
	}
	return nil
}
