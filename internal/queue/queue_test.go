package queue

import (
	"encoding/json"
	"testing"
)

func TestFingerprintTask(t *testing.T) {
	task, err := NewFingerprintTask("track")
	if err != nil || task.Type() != TypeFingerprintTrack {
		t.Fatalf("%v %v", task, err)
	}
	var got FingerprintTrackPayload
	if err := json.Unmarshal(task.Payload(), &got); err != nil || got.TrackID != "track" {
		t.Fatalf("%+v %v", got, err)
	}
}
