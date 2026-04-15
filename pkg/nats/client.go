// Package nats provides a JetStream client wrapper with at-least-once publish semantics.
package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Subject name constants used across all services.
const (
	SubjectRawAVL      = "gpsgo.avl.raw"      // ingestion → stream-processor
	SubjectEnrichedAVL = "gpsgo.avl.enriched"  // stream-processor → rule-engine / ws
	SubjectAlerts      = "gpsgo.alerts"         // rule-engine → notification / ws
	SubjectCommands    = "gpsgo.commands"        // api → ingestion (bidirectional cmds)
	SubjectDeadLetter  = "gpsgo.dlq"            // parse failures with raw payload

	StreamAVL    = "AVL"
	StreamAlerts = "ALERTS"
)

// Client wraps a NATS connection and JetStream context.
type Client struct {
	conn *nats.Conn
	js   jetstream.JetStream
}

// New connects to NATS and creates required JetStream streams if they don't exist.
func New(url string) (*Client, error) {
	nc, err := nats.Connect(url,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if err != nil {
				fmt.Printf("NATS disconnected: %v\n", err)
			}
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("nats jetstream: %w", err)
	}

	c := &Client{conn: nc, js: js}
	if err := c.ensureStreams(context.Background()); err != nil {
		nc.Close()
		return nil, fmt.Errorf("nats ensure streams: %w", err)
	}
	return c, nil
}

func (c *Client) ensureStreams(ctx context.Context) error {
	streams := []jetstream.StreamConfig{
		{
			Name:        StreamAVL,
			Subjects:    []string{SubjectRawAVL, SubjectEnrichedAVL, SubjectDeadLetter},
			Retention:   jetstream.LimitsPolicy,
			MaxAge:      24 * time.Hour,
			Storage:     jetstream.FileStorage,
			Replicas:    1,
			Compression: jetstream.S2Compression,
		},
		{
			Name:      StreamAlerts,
			Subjects:  []string{SubjectAlerts},
			Retention: jetstream.LimitsPolicy,
			MaxAge:    7 * 24 * time.Hour,
			Storage:   jetstream.FileStorage,
			Replicas:  1,
		},
	}

	for _, cfg := range streams {
		_, err := c.js.CreateOrUpdateStream(ctx, cfg)
		if err != nil {
			return fmt.Errorf("stream %s: %w", cfg.Name, err)
		}
	}
	return nil
}

// Publish sends a message to the given subject synchronously with an idempotency key.
func (c *Client) Publish(ctx context.Context, subject, msgID string, data []byte) error {
	_, err := c.js.Publish(ctx, subject, data,
		jetstream.WithMsgID(msgID),
	)
	return err
}

// JetStream returns the raw JetStream interface for consumers that need it.
func (c *Client) JetStream() jetstream.JetStream {
	return c.js
}

// Close closes the NATS connection.
func (c *Client) Close() {
	c.conn.Drain() //nolint:errcheck
}
