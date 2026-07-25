package contracts

// ToolStatus is the derived health of a tool.
type ToolStatus string

const (
	StatusUp           ToolStatus = "up"
	StatusDown         ToolStatus = "down"
	StatusAgentOffline ToolStatus = "agent_offline"
	StatusUnknown      ToolStatus = "unknown"
)

// ToolDTO is the catalog representation returned by the public API. It never
// carries credentials.
type ToolDTO struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Description      string     `json:"description"`
	Category         string     `json:"category"`
	Tags             []string   `json:"tags"`
	Scheme           string     `json:"scheme"`
	Address          string     `json:"address"`
	Port             int        `json:"port,omitempty"`
	URL              string     `json:"url,omitempty"`
	PhysicalLocation string     `json:"physical_location,omitempty"`
	HostID           string     `json:"host_id,omitempty"`
	SourceType       string     `json:"source_type"`
	Status           ToolStatus `json:"status"`
}

// ErrorResponse is the consistent error envelope for every API failure.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
