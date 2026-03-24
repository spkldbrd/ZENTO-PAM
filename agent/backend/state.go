package backend

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// AgentState is persisted after successful POST /agent/register (device_id from JSON deviceId).
type AgentState struct {
	DeviceID string `json:"device_id"`
}

func statePath(baseDir string) string {
	return filepath.Join(baseDir, "data", "agent_state.json")
}

func keysPath(baseDir string) string {
	return filepath.Join(baseDir, "data", "ed25519.key")
}

// LoadState reads saved registration; ok false if missing.
func LoadState(baseDir string) (st AgentState, ok bool, err error) {
	p := statePath(baseDir)
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return AgentState{}, false, nil
		}
		return AgentState{}, false, err
	}
	if err := json.Unmarshal(b, &st); err != nil {
		return AgentState{}, false, err
	}
	return st, true, nil
}

func SaveState(baseDir string, st *AgentState) error {
	dir := filepath.Join(baseDir, "data")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	tmp := statePath(baseDir) + ".tmp"
	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, statePath(baseDir))
}
