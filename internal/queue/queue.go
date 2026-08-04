// Package queue contains Lyra's small, explicit Asynq task contract.
package queue

import (
	"encoding/json"
	"fmt"
	"github.com/hibiken/asynq"
)

const TypeFingerprintTrack = "fingerprint_track"

type FingerprintTrackPayload struct {
	TrackID string `json:"track_id"`
}

func NewFingerprintTask(trackID string) (*asynq.Task, error) {
	body, err := json.Marshal(FingerprintTrackPayload{TrackID: trackID})
	if err != nil {
		return nil, fmt.Errorf("encode fingerprint task: %w", err)
	}
	return asynq.NewTask(TypeFingerprintTrack, body), nil
}

type Client struct{ client *asynq.Client }

func NewClient(redisAddr string) *Client {
	return &Client{client: asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})}
}
func (c *Client) Enqueue(trackID string) error {
	task, err := NewFingerprintTask(trackID)
	if err != nil {
		return err
	}
	_, err = c.client.Enqueue(task, asynq.MaxRetry(5))
	return err
}
func (c *Client) Close() error { return c.client.Close() }
