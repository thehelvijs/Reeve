// Package contracts holds the wire types shared by the server and the agent.
// Both sides import these so the push payload and API DTOs never drift.
package contracts

import (
	"time"
)

// Per-section ceilings on one push. A host runs as many services and
// containers as it runs, but an enrolled agent is a credential sitting on
// someone else's machine: these bound what one compromised host writes per
// tick, well above what a real machine reports.
const (
	MaxPushServices   = 2000
	MaxPushContainers = 2000
	MaxPushCronJobs   = 2000
	MaxPushLogEvents  = 1000
	MaxPushProcesses  = 100
)

// Push is one telemetry batch an agent sends to the server on its interval.
type Push struct {
	AgentVersion     string `json:"agent_version"`
	AutoUpdateVetoed bool   `json:"auto_update_vetoed"`
	// IPAddress is the host's address on the route to the server, so a tool
	// pinned to this host can be reached after its DHCP lease changes. Empty
	// when the agent could not work it out.
	IPAddress      string            `json:"ip_address"`
	SentAt         time.Time         `json:"sent_at"`
	Services       []ServiceState    `json:"services"`
	Containers     []ContainerState  `json:"containers"`
	CronJobs       []CronState       `json:"cron_jobs"`
	Metrics        HostMetrics       `json:"metrics"`
	ContainerStats []ContainerSample `json:"container_stats"`
	Processes      []ProcessSample   `json:"processes"`
	LogEvents      []LogEvent        `json:"log_events"`
}

// TooLarge names the first section of p that exceeds its ceiling, or "" when
// the push is within bounds.
func (p Push) TooLarge() string {
	for _, section := range []struct {
		name  string
		count int
		max   int
	}{
		{"services", len(p.Services), MaxPushServices},
		{"containers", len(p.Containers), MaxPushContainers},
		{"container_stats", len(p.ContainerStats), MaxPushContainers},
		{"cron_jobs", len(p.CronJobs), MaxPushCronJobs},
		{"processes", len(p.Processes), MaxPushProcesses},
		{"log_events", len(p.LogEvents), MaxPushLogEvents},
	} {
		if section.count > section.max {
			return section.name
		}
	}
	return ""
}

// PushAck is the server's reply to a push. CheckNow asks the agent to run its
// self-update now; the server withholds it to pace a fleet-wide rollout.
type PushAck struct {
	CheckNow bool `json:"check_now"`
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

// ProcessSample is one process's resource use at push time. CPUPct is percent
// of a single core, as top reports it, so a process saturating two reads 200.
type ProcessSample struct {
	PID     int     `json:"pid"`
	User    string  `json:"user"`
	Command string  `json:"command"`
	CPUPct  float64 `json:"cpu_pct"`
	MemRSS  uint64  `json:"mem_rss"`
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
