package backend

import (
	"encoding/json"
	"testing"
)

func TestRegisterResponse_deviceIdJSON(t *testing.T) {
	const in = `{"deviceId":"550e8400-e29b-41d4-a716-446655440000"}`
	var got RegisterResponse
	if err := json.Unmarshal([]byte(in), &got); err != nil {
		t.Fatal(err)
	}
	if got.DeviceID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("DeviceID: got %q", got.DeviceID)
	}
}

func TestElevationPostResponse_JSON(t *testing.T) {
	const in = `{"id":"b008d3ab-c3d4-42f5-897c-14dad70b6c1a","status":"pending"}`
	var got ElevationPostResponse
	if err := json.Unmarshal([]byte(in), &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != "b008d3ab-c3d4-42f5-897c-14dad70b6c1a" || got.Status != "pending" {
		t.Fatalf("got %+v", got)
	}
}
