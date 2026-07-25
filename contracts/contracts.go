// Package contracts holds the wire types shared by the server and the agent.
// Both sides import these so the push payload and API DTOs never drift.
package contracts

import (
	"time"
)

// PushProtocolVersion is bumped when the agent->server payload shape changes.
const PushProtocolVersion = 1

// Push is one telemetry batch an agent sends to the server on its interval.
type Push struct {
	ProtocolVersion int               `json:"protocol_version"`
	AgentVersion    string            `json:"agent_version"`
	SentAt          time.Time         `json:"sent_at"`
	Services        []ServiceState    `json:"services"`
	Containers      []ContainerState  `json:"containers"`
	CronJobs        []CronState       `json:"cron_jobs"`
	Metrics         HostMetrics       `json:"metrics"`
	ContainerStats  []ContainerSample `json:"container_stats"`
	LogEvents       []LogEvent        `json:"log_events"`
}

// ServiceState is a systemd unit's current state.
type ServiceState struct {
	Unit        string `json:"unit"`
	ActiveState string `json:"active_state"`
	SubState    string `json:"sub_state"`
}

// ContainerState is a Docker container's current state and health.
type ContainerState struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Image  string `json:"image"`
	State  string `json:"state"`
	Health string `json:"health"`
}

// CronState is a cron job with a best-effort last-run heuristic.
type CronState struct {
	Name      string     `json:"name"`
	Schedule  string     `json:"schedule"`
	LastRunAt *time.Time `json:"last_run_at,omitempty"`
	LastExit  *int       `json:"last_exit,omitempty"`
}

// HostMetrics is a single host-level metric sample.
type HostMetrics struct {
	CPUPct      float64            `json:"cpu_pct"`
	MemUsed     uint64             `json:"mem_used"`
	MemTotal    uint64             `json:"mem_total"`
	DiskUsed    uint64             `json:"disk_used"`
	DiskTotal   uint64             `json:"disk_total"`
	DiskRead    uint64             `json:"disk_read"`
	DiskWrite   uint64             `json:"disk_write"`
	NetRx       uint64             `json:"net_rx"`
	NetTx       uint64             `json:"net_tx"`
	Load1       float64            `json:"load1"`
	Load5       float64            `json:"load5"`
	Load15      float64            `json:"load15"`
	Temps       map[string]float64 `json:"temps,omitempty"`
	UptimeSecs  uint64             `json:"uptime_secs"`
	GPUUtil     float64            `json:"gpu_util"`
	GPUMemUsed  uint64             `json:"gpu_mem_used"`
	GPUMemTotal uint64             `json:"gpu_mem_total"`
}

// ContainerSample is a per-container resource sample.
type ContainerSample struct {
	ContainerID string  `json:"container_id"`
	CPUPct      float64 `json:"cpu_pct"`
	MemUsed     uint64  `json:"mem_used"`
	MemLimit    uint64  `json:"mem_limit"`
}

// LogEvent is a single error-level log line the agent flagged.
type LogEvent struct {
	Source  string    `json:"source"`
	Level   string    `json:"level"`
	Message string    `json:"message"`
	At      time.Time `json:"at"`
}

// ToolStatus is the derived health of a tool.
type ToolStatus string

const (
	StatusUp           ToolStatus = "up"
	StatusDown         ToolStatus = "down"
	StatusAgentOffline ToolStatus = "agent_offline"
	StatusUnknown      ToolStatus = "unknown"
)

// CollectionRef is the compact collection shape embedded in a tool payload.
type CollectionRef struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	IconURL string `json:"icon_url"`
}

// ToolDTO is the catalog representation returned by the public API. It never
// carries credentials.
type ToolDTO struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Description      string          `json:"description"`
	Collections      []CollectionRef `json:"collections"`
	Tags             []string        `json:"tags"`
	Scheme           string          `json:"scheme"`
	Address          string          `json:"address"`
	Port             int             `json:"port,omitempty"`
	URL              string          `json:"url,omitempty"`
	PhysicalLocation string          `json:"physical_location,omitempty"`
	HostID           string          `json:"host_id,omitempty"`
	SourceType       string          `json:"source_type"`
	Status           ToolStatus      `json:"status"`
}

// ErrorResponse is the consistent error envelope for every API failure.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
