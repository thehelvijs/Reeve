package main

import (
	"io/fs"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/thehelvijs/Reeve/agent/collect"
	"github.com/thehelvijs/Reeve/contracts"
	"github.com/thehelvijs/Reeve/server/internal/auth"
	"github.com/thehelvijs/Reeve/server/internal/crypto"
	"github.com/thehelvijs/Reeve/server/internal/rbac"
	"github.com/thehelvijs/Reeve/server/internal/store"
)

// config holds the server's runtime settings.
type config struct {
	Addr         string
	PublicURL    string
	DBPath       string
	AvatarDir    string
	IconDir      string
	SessionTTL   time.Duration
	CookieSecure bool
	Version      string
	LogRequests  bool
	// TrustProxyHeaders lets X-Forwarded-For set the client IP. Off unless a
	// reverse proxy the operator controls is the only way in: otherwise any
	// caller could forge the throttle key and the audited reveal source.
	TrustProxyHeaders bool
	// AllowedOrigins are extra origins the same-origin check accepts, for a
	// dev UI served from a different port than the API.
	AllowedOrigins []string
}

// app wires the store, cipher, and config for the HTTP handlers.
type app struct {
	db             *store.DB
	cipher         *crypto.Cipher
	cfg            config
	startedAt      time.Time
	ingestRejected atomic.Int64
	agentFS        fs.FS
	scriptFS       fs.FS
	sumsOnce       sync.Once
	sums           map[string]bool
	// hostSampler measures this machine's CPU across the gap between reads, so
	// nothing sleeps in a request or a loop to create a window of its own. Both
	// the sample loop and the admin endpoint read it, hence the lock.
	hostMu      sync.Mutex
	hostSampler *collect.HostSampler
	// send delivers one webhook payload; nil uses the real HTTP transport. A
	// field so tests can substitute a fake.
	send         func(notifyChannel, string) error
	throttleOnce sync.Once
	throttle     *auth.Throttle
	// googleEndpoints redirects the OAuth legs at a stub provider in tests.
	googleEndpoints *oauthEndpoints
}

// sampleHost reads this machine's metrics, measuring CPU against the previous
// read rather than a sleep of its own.
func (a *app) sampleHost() contracts.HostMetrics {
	a.hostMu.Lock()
	defer a.hostMu.Unlock()
	if a.hostSampler == nil {
		a.hostSampler = collect.NewHostSampler()
	}
	return a.hostSampler.Sample()
}

// oauthEndpoints is the provider's three URLs.
type oauthEndpoints struct {
	Auth     string
	Token    string
	UserInfo string
}

// loginThrottle returns the shared login failure throttle, built on first use
// so a zero-value app (as tests construct) still has one.
func (a *app) loginThrottle() *auth.Throttle {
	a.throttleOnce.Do(func() { a.throttle = auth.NewLoginThrottle() })
	return a.throttle
}

// routes builds the full HTTP handler: public auth routes, then authenticated
// and admin-gated groups. Every request passes through resolvePrincipal so a
// valid session or token populates the context.
func (a *app) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", a.handleHealthz)

	// Public (no auth required).
	mux.HandleFunc("GET /api/auth/status", a.handleAuthStatus)
	mux.HandleFunc("POST /api/auth/signup", a.handleSignup)
	mux.HandleFunc("POST /api/auth/login", a.handleLogin)
	mux.HandleFunc("POST /api/auth/logout", a.handleLogout)
	mux.HandleFunc("POST /api/auth/forgot", a.handleForgotPassword)
	mux.HandleFunc("POST /api/auth/reset", a.handleResetPassword)
	mux.HandleFunc("GET /api/auth/google/start", a.handleGoogleStart)
	mux.HandleFunc("GET "+googleCallbackPath, a.handleGoogleCallback)
	mux.HandleFunc("GET /api/public/tools", a.handleListPublicTools)
	mux.HandleFunc("GET /api/public/hosts", a.handleListPublicHosts)
	mux.HandleFunc("GET /api/public/collections", a.handleListPublicCollections)
	mux.HandleFunc("GET /api/collections/{id}/icon", a.serveImage(collectionIcon))
	// Short, durable URLs for a tool wherever it currently is. Public tools
	// answer anonymously; the handlers gate the rest.
	mux.HandleFunc("GET /go/{slug}", a.handleGoToTool)
	mux.HandleFunc("GET /api/endpoints/{slug}", a.handleGetEndpoint)
	mux.HandleFunc("GET /install.sh", a.handleInstallScript)
	mux.HandleFunc("GET /uninstall.sh", a.handleUninstallScript)
	mux.HandleFunc("GET /dl/{filename}", a.handleAgentDownload)

	// Agent ingest (authenticated by host enrollment token, not a user session).
	mux.HandleFunc("POST /api/ingest", a.handleIngest)

	// Authenticated.
	authed := rbac.RequireAuth
	admin := rbac.RequireAdmin
	mux.Handle("GET /api/me", authed(http.HandlerFunc(a.handleMe)))
	mux.Handle("PATCH /api/me", authed(http.HandlerFunc(a.handleUpdateMe)))
	mux.Handle("DELETE /api/me", authed(http.HandlerFunc(a.handleDeleteMe)))
	mux.Handle("POST /api/me/password", authed(http.HandlerFunc(a.handleChangePassword)))
	mux.Handle("POST /api/me/avatar", authed(http.HandlerFunc(a.handleUploadAvatar)))
	mux.Handle("DELETE /api/me/avatar", authed(http.HandlerFunc(a.handleDeleteAvatar)))
	mux.Handle("GET /api/users/{id}/avatar", authed(http.HandlerFunc(a.handleGetAvatar)))

	// Catalog.
	mux.Handle("GET /api/tools", authed(http.HandlerFunc(a.handleListTools)))
	mux.Handle("POST /api/tools", authed(http.HandlerFunc(a.handleCreateTool)))
	mux.Handle("GET /api/tools/{id}", authed(http.HandlerFunc(a.handleGetTool)))
	mux.HandleFunc("GET /api/tools/{id}/icon", a.serveImage(toolIcon))
	mux.Handle("POST /api/tools/{id}/icon", authed(a.uploadImage(toolIcon)))
	mux.Handle("DELETE /api/tools/{id}/icon", authed(a.deleteImage(toolIcon)))
	mux.HandleFunc("GET /api/tools/{id}/thumbnail", a.serveImage(toolThumbnail))
	mux.Handle("POST /api/tools/{id}/thumbnail", authed(a.uploadImage(toolThumbnail)))
	mux.Handle("DELETE /api/tools/{id}/thumbnail", authed(a.deleteImage(toolThumbnail)))
	mux.Handle("PATCH /api/tools/{id}", authed(http.HandlerFunc(a.handleUpdateTool)))
	mux.Handle("DELETE /api/tools/{id}", authed(http.HandlerFunc(a.handleDeleteTool)))
	mux.Handle("GET /api/tools/{id}/visibility", authed(http.HandlerFunc(a.handleListToolVisibility)))
	mux.Handle("PUT /api/tools/{id}/visibility/{ptype}/{pid}", authed(http.HandlerFunc(a.handleAddToolVisibility)))
	mux.Handle("DELETE /api/tools/{id}/visibility/{ptype}/{pid}", authed(http.HandlerFunc(a.handleRemoveToolVisibility)))

	// Collections.
	mux.Handle("GET /api/collections", authed(http.HandlerFunc(a.handleListCollections)))
	mux.Handle("POST /api/collections", authed(http.HandlerFunc(a.handleCreateCollection)))
	mux.Handle("GET /api/collections/{id}", authed(http.HandlerFunc(a.handleGetCollection)))
	mux.Handle("PATCH /api/collections/{id}", authed(http.HandlerFunc(a.handleUpdateCollection)))
	mux.Handle("DELETE /api/collections/{id}", authed(http.HandlerFunc(a.handleDeleteCollection)))
	mux.Handle("POST /api/collections/{id}/icon", authed(a.uploadImage(collectionIcon)))
	mux.Handle("DELETE /api/collections/{id}/icon", authed(a.deleteImage(collectionIcon)))
	mux.Handle("PUT /api/collections/{id}/tools/{toolId}", authed(http.HandlerFunc(a.handleAddCollectionTool)))
	mux.Handle("DELETE /api/collections/{id}/tools/{toolId}", authed(http.HandlerFunc(a.handleRemoveCollectionTool)))
	mux.Handle("GET /api/collections/{id}/editors", authed(http.HandlerFunc(a.handleListCollectionEditors)))
	mux.Handle("PUT /api/collections/{id}/editors/{ptype}/{pid}", authed(http.HandlerFunc(a.handleAddCollectionEditor)))
	mux.Handle("DELETE /api/collections/{id}/editors/{ptype}/{pid}", authed(http.HandlerFunc(a.handleRemoveCollectionEditor)))
	mux.Handle("GET /api/collections/{id}/visibility", authed(http.HandlerFunc(a.handleListCollectionVisibility)))
	mux.Handle("PUT /api/collections/{id}/visibility/{ptype}/{pid}", authed(http.HandlerFunc(a.handleAddCollectionVisibility)))
	mux.Handle("DELETE /api/collections/{id}/visibility/{ptype}/{pid}", authed(http.HandlerFunc(a.handleRemoveCollectionVisibility)))
	mux.Handle("GET /api/principals", authed(http.HandlerFunc(a.handleListPrincipals)))

	// Groups a moderator manages. The list is scoped to the caller, and every
	// membership write is gated on admin-or-moderator-of-that-group inside the
	// handler, which is why these sit outside /api/admin.
	mux.Handle("GET /api/groups", authed(http.HandlerFunc(a.handleListGroups)))
	mux.Handle("POST /api/groups/{id}/members", authed(http.HandlerFunc(a.handleAddGroupMemberByEmail)))
	mux.Handle("PATCH /api/groups/{id}/members/{userId}", authed(http.HandlerFunc(a.handleSetGroupMemberRole)))
	mux.Handle("DELETE /api/groups/{id}/members/{userId}", authed(http.HandlerFunc(a.handleRemoveGroupMember)))

	// Credentials.
	// Credentials belong to hosts: anyone signed in can see one exists and ask
	// for it, only an admin can add, rotate or delete it.
	mux.Handle("GET /api/hosts/{id}/credentials", authed(http.HandlerFunc(a.handleListCredentials)))
	mux.Handle("POST /api/admin/hosts/{id}/credentials", admin(http.HandlerFunc(a.handleCreateCredential)))
	mux.Handle("PATCH /api/admin/credentials/{cid}", admin(http.HandlerFunc(a.handleUpdateCredential)))
	mux.Handle("DELETE /api/admin/credentials/{cid}", admin(http.HandlerFunc(a.handleDeleteCredential)))
	mux.Handle("POST /api/credentials/{cid}/reveal", authed(http.HandlerFunc(a.handleRevealCredential)))

	// Access requests and standing grants.
	mux.Handle("POST /api/hosts/{id}/access-requests", authed(http.HandlerFunc(a.handleCreateAccessRequest)))
	mux.Handle("GET /api/access-requests", authed(http.HandlerFunc(a.handleListAccessRequests)))
	mux.Handle("POST /api/access-requests/{rid}/approve", authed(http.HandlerFunc(a.handleApproveAccessRequest)))
	mux.Handle("POST /api/access-requests/{rid}/deny", authed(http.HandlerFunc(a.handleDenyAccessRequest)))
	mux.Handle("GET /api/admin/hosts/{id}/access", admin(http.HandlerFunc(a.handleListHostAccess)))
	mux.Handle("PUT /api/admin/hosts/{id}/access/{ptype}/{pid}", admin(http.HandlerFunc(a.handleGrantHostAccess)))
	mux.Handle("DELETE /api/admin/hosts/{id}/access/{ptype}/{pid}", admin(http.HandlerFunc(a.handleRevokeHostAccess)))

	// Hosts (list + inventory for any user; create/delete admin-only).
	mux.Handle("GET /api/hosts", authed(http.HandlerFunc(a.handleListHosts)))
	mux.HandleFunc("GET /api/hosts/{id}/icon", a.serveImage(hostIcon))
	mux.HandleFunc("GET /api/hosts/{id}/thumbnail", a.serveImage(hostThumbnail))
	// What a machine runs is admin-only. Status, location and metrics stay open
	// to every account because the dashboard is built on them, but the unit,
	// container, cron and process lists name attack surface and sometimes carry
	// an argument nobody meant to publish.
	mux.Handle("GET /api/hosts/{id}/inventory", admin(http.HandlerFunc(a.handleHostInventory)))
	mux.Handle("GET /api/hosts/{id}/metrics", authed(http.HandlerFunc(a.handleHostMetrics)))
	mux.Handle("GET /api/hosts/{id}/process-usage", admin(http.HandlerFunc(a.handleHostProcessUsage)))
	mux.Handle("GET /api/hosts/{id}/events", authed(http.HandlerFunc(a.handleHostEvents)))
	mux.Handle("GET /api/hosts/{id}/uptime", authed(http.HandlerFunc(a.handleHostUptime)))
	mux.Handle("GET /api/tools/{id}/events", authed(http.HandlerFunc(a.handleToolEvents)))
	mux.Handle("GET /api/tools/{id}/uptime", authed(http.HandlerFunc(a.handleToolUptime)))

	// Admin only.
	mux.Handle("GET /api/admin/users", admin(http.HandlerFunc(a.handleListUsers)))
	mux.Handle("PATCH /api/admin/users/{id}", admin(http.HandlerFunc(a.handleUpdateUser)))
	mux.Handle("DELETE /api/admin/users/{id}", admin(http.HandlerFunc(a.handleDeleteUser)))
	mux.Handle("POST /api/admin/users/{id}/password", admin(http.HandlerFunc(a.handleAdminResetPassword)))
	mux.Handle("POST /api/admin/users", admin(http.HandlerFunc(a.handleInviteUser)))
	mux.Handle("GET /api/admin/groups", admin(http.HandlerFunc(a.handleListGroups)))
	mux.Handle("POST /api/admin/groups", admin(http.HandlerFunc(a.handleCreateGroup)))
	mux.Handle("PATCH /api/admin/groups/{id}", admin(http.HandlerFunc(a.handleRenameGroup)))
	mux.Handle("DELETE /api/admin/groups/{id}", admin(http.HandlerFunc(a.handleDeleteGroup)))
	mux.Handle("PUT /api/admin/groups/{id}/members/{userId}", admin(http.HandlerFunc(a.handleAddGroupMember)))
	mux.Handle("DELETE /api/admin/groups/{id}/members/{userId}", admin(http.HandlerFunc(a.handleRemoveGroupMember)))
	mux.Handle("GET /api/admin/audit/reveals", admin(http.HandlerFunc(a.handleRevealAudit)))
	mux.Handle("GET /api/admin/audit/grants", admin(http.HandlerFunc(a.handleGrantAudit)))
	mux.Handle("GET /api/admin/audit/verify", admin(http.HandlerFunc(a.handleVerifyAudit)))
	mux.Handle("GET /api/admin/settings", admin(http.HandlerFunc(a.handleGetSettings)))
	mux.Handle("PUT /api/admin/settings", admin(http.HandlerFunc(a.handlePutSettings)))
	mux.Handle("POST /api/admin/settings/test-email", admin(http.HandlerFunc(a.handleTestEmail)))
	mux.Handle("GET /api/admin/backup", admin(http.HandlerFunc(a.handleBackupDownload)))
	mux.Handle("POST /api/admin/restore", admin(http.HandlerFunc(a.handleRestoreUpload)))
	mux.Handle("DELETE /api/admin/restore", admin(http.HandlerFunc(a.handleRestoreCancel)))
	mux.Handle("GET /api/admin/server-info", admin(http.HandlerFunc(a.handleServerInfo)))
	mux.Handle("GET /api/admin/server-metrics", admin(http.HandlerFunc(a.handleServerMetrics)))
	// Embedded SPA: least-specific pattern, so all API routes above win.
	mux.Handle("/", a.uiHandler())
	mux.Handle("POST /api/admin/hosts", admin(http.HandlerFunc(a.handleCreateHost)))
	mux.Handle("PATCH /api/admin/hosts/{id}", admin(http.HandlerFunc(a.handleUpdateHostLocation)))
	mux.Handle("DELETE /api/admin/hosts/{id}", admin(http.HandlerFunc(a.handleDeleteHost)))
	mux.Handle("POST /api/admin/hosts/{id}/icon", admin(a.uploadImage(hostIcon)))
	mux.Handle("DELETE /api/admin/hosts/{id}/icon", admin(a.deleteImage(hostIcon)))
	mux.Handle("POST /api/admin/hosts/{id}/thumbnail", admin(a.uploadImage(hostThumbnail)))
	mux.Handle("DELETE /api/admin/hosts/{id}/thumbnail", admin(a.deleteImage(hostThumbnail)))
	mux.Handle("GET /api/admin/geocode", admin(http.HandlerFunc(a.handleGeocode)))
	mux.Handle("POST /api/admin/ssh-probe", admin(http.HandlerFunc(a.handleSSHProbe)))
	mux.Handle("POST /api/admin/hosts/{id}/ssh-install", admin(http.HandlerFunc(a.handleSSHInstall)))
	mux.Handle("POST /api/admin/hosts/{id}/ssh-uninstall", admin(http.HandlerFunc(a.handleSSHUninstall)))
	mux.Handle("DELETE /api/admin/hosts/{id}/metrics", admin(http.HandlerFunc(a.handleClearHostMetrics)))
	mux.Handle("GET /api/admin/webhooks", admin(http.HandlerFunc(a.handleListWebhooks)))
	mux.Handle("POST /api/admin/webhooks", admin(http.HandlerFunc(a.handleCreateWebhook)))
	mux.Handle("DELETE /api/admin/webhooks/{id}", admin(http.HandlerFunc(a.handleDeleteWebhook)))
	mux.Handle("GET /api/admin/alerts", admin(http.HandlerFunc(a.handleListAlertEvents)))
	mux.Handle("GET /api/admin/deliveries", admin(http.HandlerFunc(a.handleListDeliveries)))
	mux.Handle("GET /api/admin/thresholds", admin(http.HandlerFunc(a.handleGetThresholds)))
	mux.Handle("PUT /api/admin/thresholds", admin(http.HandlerFunc(a.handlePutThresholds)))
	mux.Handle("GET /api/admin/hosts/{id}/thresholds", admin(http.HandlerFunc(a.handleGetHostThresholds)))
	mux.Handle("PUT /api/admin/hosts/{id}/thresholds", admin(http.HandlerFunc(a.handlePutHostThresholds)))
	mux.Handle("DELETE /api/admin/hosts/{id}/thresholds", admin(http.HandlerFunc(a.handleResetHostThresholds)))
	mux.Handle("PUT /api/admin/hosts/{id}/auto-update", admin(http.HandlerFunc(a.handleSetHostAutoUpdate)))
	mux.Handle("POST /api/admin/hosts/{id}/commands", admin(http.HandlerFunc(a.handleCreateCommand)))
	mux.Handle("GET /api/admin/hosts/{id}/commands", admin(http.HandlerFunc(a.handleListCommands)))
	mux.Handle("POST /api/admin/hosts/{id}/update-now", admin(http.HandlerFunc(a.handleHostUpdateNow)))
	mux.Handle("GET /api/admin/agent-updates", admin(http.HandlerFunc(a.handleGetAgentUpdates)))
	mux.Handle("POST /api/admin/agent-updates/resume", admin(http.HandlerFunc(a.handleResumeAgentUpdates)))

	return securityHeaders(a.resolvePrincipal(a.requireSameOrigin(a.logRequests(gzipResponses(mux)))))
}
