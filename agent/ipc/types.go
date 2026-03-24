package ipc

// Request is the JSON body sent over the named pipe.
type Request struct {
	ExePath    string `json:"exe_path"`
	Args       string `json:"args"`
	WorkingDir string `json:"working_dir"`
	UserSID    string `json:"user_sid"`
	Username   string `json:"username"`
	ParentPID  int32  `json:"parent_pid"`
	ParentPath string `json:"parent_path"`
}

// Response is returned to the client after handling the request.
type Response struct {
	OK            bool   `json:"ok"`
	Error         string `json:"error,omitempty"`
	PID           uint32 `json:"pid,omitempty"`
	BackendStatus string `json:"backend_status,omitempty"`
	RequestID     string `json:"request_id,omitempty"`
}
