package main

import (
	"io/fs"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

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
	notifiers      map[string]Notifier
	throttleOnce   sync.Once
	throttle       *auth.Throttle
	// googleEndpoints redirects the OAuth legs at a stub provider in tests.
	googleEndpoints *oauthEndpoints
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
	mux.HandleFunc("GET /api/v1/auth/status", a.handleAuthStatus)
	mux.HandleFunc("POST /api/v1/auth/signup", a.handleSignup)
	mux.HandleFunc("POST /api/v1/auth/login", a.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/logout", a.handleLogout)
	mux.HandleFunc("POST /api/v1/auth/forgot", a.handleForgotPassword)
	mux.HandleFunc("POST /api/v1/auth/reset", a.handleResetPassword)
	mux.HandleFunc("GET /api/v1/auth/google/start", a.handleGoogleStart)
	mux.HandleFunc("GET "+googleCallbackPath, a.handleGoogleCallback)
	mux.HandleFunc("GET /api/v1/public/tools", a.handleListPublicTools)
	mux.HandleFunc("GET /api/v1/public/hosts", a.handleListPublicHosts)
	mux.HandleFunc("GET /install.sh", a.handleInstallScript)
	mux.HandleFunc("GET /uninstall.sh", a.handleUninstallScript)
	mux.HandleFunc("GET /dl/{filename}", a.handleAgentDownload)

	// Agent ingest (authenticated by host enrollment token, not a user session).
	mux.HandleFunc("POST /api/v1/ingest", a.handleIngest)

	// Authenticated.
	authed := rbac.RequireAuth
	mux.Handle("GET /api/v1/me", authed(http.HandlerFunc(a.handleMe)))
	mux.Handle("PATCH /api/v1/me", authed(http.HandlerFunc(a.handleUpdateMe)))
	mux.Handle("DELETE /api/v1/me", authed(http.HandlerFunc(a.handleDeleteMe)))
	mux.Handle("POST /api/v1/me/password", authed(http.HandlerFunc(a.handleChangePassword)))
	mux.Handle("POST /api/v1/me/avatar", authed(http.HandlerFunc(a.handleUploadAvatar)))
	mux.Handle("DELETE /api/v1/me/avatar", authed(http.HandlerFunc(a.handleDeleteAvatar)))
	mux.Handle("GET /api/v1/users/{id}/avatar", authed(http.HandlerFunc(a.handleGetAvatar)))

	// Catalog.
	mux.Handle("GET /api/v1/tools", authed(http.HandlerFunc(a.handleListTools)))
	mux.Handle("POST /api/v1/tools", authed(http.HandlerFunc(a.handleCreateTool)))
	mux.Handle("GET /api/v1/tools/{id}", authed(http.HandlerFunc(a.handleGetTool)))
	mux.HandleFunc("GET /api/v1/tools/{id}/icon", a.handleServeToolIcon)
	mux.Handle("POST /api/v1/tools/{id}/icon", authed(http.HandlerFunc(a.handleUploadToolIcon)))
	mux.Handle("DELETE /api/v1/tools/{id}/icon", authed(http.HandlerFunc(a.handleDeleteToolIcon)))
	mux.HandleFunc("GET /api/v1/tools/{id}/thumbnail", a.handleServeToolThumbnail)
	mux.Handle("POST /api/v1/tools/{id}/thumbnail", authed(http.HandlerFunc(a.handleUploadToolThumbnail)))
	mux.Handle("DELETE /api/v1/tools/{id}/thumbnail", authed(http.HandlerFunc(a.handleDeleteToolThumbnail)))
	mux.Handle("PATCH /api/v1/tools/{id}", authed(http.HandlerFunc(a.handleUpdateTool)))
	mux.Handle("DELETE /api/v1/tools/{id}", authed(http.HandlerFunc(a.handleDeleteTool)))
	mux.Handle("GET /api/v1/tools/{id}/visibility", authed(http.HandlerFunc(a.handleListToolVisibility)))
	mux.Handle("PUT /api/v1/tools/{id}/visibility/{ptype}/{pid}", authed(http.HandlerFunc(a.handleAddToolVisibility)))
	mux.Handle("DELETE /api/v1/tools/{id}/visibility/{ptype}/{pid}", authed(http.HandlerFunc(a.handleRemoveToolVisibility)))

	// Credentials.
	mux.Handle("GET /api/v1/tools/{id}/credentials", authed(http.HandlerFunc(a.handleListCredentials)))
	mux.Handle("POST /api/v1/tools/{id}/credentials", authed(http.HandlerFunc(a.handleCreateCredential)))
	mux.Handle("PATCH /api/v1/credentials/{cid}", authed(http.HandlerFunc(a.handleUpdateCredential)))
	mux.Handle("DELETE /api/v1/credentials/{cid}", authed(http.HandlerFunc(a.handleDeleteCredential)))
	mux.Handle("POST /api/v1/credentials/{cid}/reveal", authed(http.HandlerFunc(a.handleRevealCredential)))

	// Access requests and standing grants.
	mux.Handle("POST /api/v1/tools/{id}/access-requests", authed(http.HandlerFunc(a.handleCreateAccessRequest)))
	mux.Handle("GET /api/v1/access-requests", authed(http.HandlerFunc(a.handleListAccessRequests)))
	mux.Handle("POST /api/v1/access-requests/{rid}/approve", authed(http.HandlerFunc(a.handleApproveAccessRequest)))
	mux.Handle("POST /api/v1/access-requests/{rid}/deny", authed(http.HandlerFunc(a.handleDenyAccessRequest)))
	mux.Handle("GET /api/v1/tools/{id}/access", authed(http.HandlerFunc(a.handleListToolAccess)))
	mux.Handle("PUT /api/v1/tools/{id}/access/{ptype}/{pid}", authed(http.HandlerFunc(a.handleGrantToolAccess)))
	mux.Handle("DELETE /api/v1/tools/{id}/access/{ptype}/{pid}", authed(http.HandlerFunc(a.handleRevokeToolAccess)))

	// Hosts (list + inventory for any user; create/delete admin-only).
	mux.Handle("GET /api/v1/hosts", authed(http.HandlerFunc(a.handleListHosts)))
	mux.HandleFunc("GET /api/v1/hosts/{id}/icon", a.handleServeHostIcon)
	mux.HandleFunc("GET /api/v1/hosts/{id}/thumbnail", a.handleServeHostThumbnail)
	mux.Handle("GET /api/v1/hosts/{id}/inventory", authed(http.HandlerFunc(a.handleHostInventory)))
	mux.Handle("GET /api/v1/hosts/{id}/metrics", authed(http.HandlerFunc(a.handleHostMetrics)))
	mux.Handle("GET /api/v1/hosts/{id}/events", authed(http.HandlerFunc(a.handleHostEvents)))
	mux.Handle("GET /api/v1/hosts/{id}/uptime", authed(http.HandlerFunc(a.handleHostUptime)))
	mux.Handle("GET /api/v1/tools/{id}/events", authed(http.HandlerFunc(a.handleToolEvents)))
	mux.Handle("GET /api/v1/tools/{id}/uptime", authed(http.HandlerFunc(a.handleToolUptime)))

	// Admin only.
	admin := rbac.RequireAdmin
	mux.Handle("GET /api/v1/admin/users", admin(http.HandlerFunc(a.handleListUsers)))
	mux.Handle("PATCH /api/v1/admin/users/{id}", admin(http.HandlerFunc(a.handleUpdateUser)))
	mux.Handle("DELETE /api/v1/admin/users/{id}", admin(http.HandlerFunc(a.handleDeleteUser)))
	mux.Handle("POST /api/v1/admin/users/{id}/password", admin(http.HandlerFunc(a.handleAdminResetPassword)))
	mux.Handle("GET /api/v1/admin/groups", admin(http.HandlerFunc(a.handleListGroups)))
	mux.Handle("POST /api/v1/admin/groups", admin(http.HandlerFunc(a.handleCreateGroup)))
	mux.Handle("PATCH /api/v1/admin/groups/{id}", admin(http.HandlerFunc(a.handleRenameGroup)))
	mux.Handle("DELETE /api/v1/admin/groups/{id}", admin(http.HandlerFunc(a.handleDeleteGroup)))
	mux.Handle("PUT /api/v1/admin/groups/{id}/members/{userId}", admin(http.HandlerFunc(a.handleAddGroupMember)))
	mux.Handle("DELETE /api/v1/admin/groups/{id}/members/{userId}", admin(http.HandlerFunc(a.handleRemoveGroupMember)))
	mux.Handle("GET /api/v1/admin/audit/reveals", admin(http.HandlerFunc(a.handleRevealAudit)))
	mux.Handle("GET /api/v1/admin/audit/grants", admin(http.HandlerFunc(a.handleGrantAudit)))
	mux.Handle("GET /api/v1/admin/settings", admin(http.HandlerFunc(a.handleGetSettings)))
	mux.Handle("PUT /api/v1/admin/settings", admin(http.HandlerFunc(a.handlePutSettings)))
	mux.Handle("POST /api/v1/admin/settings/test-email", admin(http.HandlerFunc(a.handleTestEmail)))
	mux.Handle("GET /api/v1/admin/backup", admin(http.HandlerFunc(a.handleBackupDownload)))
	mux.Handle("POST /api/v1/admin/restore", admin(http.HandlerFunc(a.handleRestoreUpload)))
	mux.Handle("DELETE /api/v1/admin/restore", admin(http.HandlerFunc(a.handleRestoreCancel)))
	mux.Handle("GET /api/v1/admin/server-info", admin(http.HandlerFunc(a.handleServerInfo)))
	mux.Handle("GET /api/v1/admin/server-metrics", admin(http.HandlerFunc(a.handleServerMetrics)))
	// Embedded SPA: least-specific pattern, so all API routes above win.
	mux.Handle("/", a.uiHandler())
	mux.Handle("POST /api/v1/admin/hosts", admin(http.HandlerFunc(a.handleCreateHost)))
	mux.Handle("PATCH /api/v1/admin/hosts/{id}", admin(http.HandlerFunc(a.handleUpdateHostLocation)))
	mux.Handle("DELETE /api/v1/admin/hosts/{id}", admin(http.HandlerFunc(a.handleDeleteHost)))
	mux.Handle("POST /api/v1/admin/hosts/{id}/icon", admin(http.HandlerFunc(a.handleUploadHostIcon)))
	mux.Handle("DELETE /api/v1/admin/hosts/{id}/icon", admin(http.HandlerFunc(a.handleDeleteHostIcon)))
	mux.Handle("POST /api/v1/admin/hosts/{id}/thumbnail", admin(http.HandlerFunc(a.handleUploadHostThumbnail)))
	mux.Handle("DELETE /api/v1/admin/hosts/{id}/thumbnail", admin(http.HandlerFunc(a.handleDeleteHostThumbnail)))
	mux.Handle("POST /api/v1/admin/ssh-probe", admin(http.HandlerFunc(a.handleSSHProbe)))
	mux.Handle("POST /api/v1/admin/hosts/{id}/ssh-install", admin(http.HandlerFunc(a.handleSSHInstall)))
	mux.Handle("POST /api/v1/admin/hosts/{id}/ssh-uninstall", admin(http.HandlerFunc(a.handleSSHUninstall)))
	mux.Handle("DELETE /api/v1/admin/hosts/{id}/metrics", admin(http.HandlerFunc(a.handleClearHostMetrics)))
	mux.Handle("GET /api/v1/admin/webhooks", admin(http.HandlerFunc(a.handleListWebhooks)))
	mux.Handle("POST /api/v1/admin/webhooks", admin(http.HandlerFunc(a.handleCreateWebhook)))
	mux.Handle("DELETE /api/v1/admin/webhooks/{id}", admin(http.HandlerFunc(a.handleDeleteWebhook)))
	mux.Handle("GET /api/v1/admin/alerts", admin(http.HandlerFunc(a.handleListAlertEvents)))
	mux.Handle("GET /api/v1/admin/deliveries", admin(http.HandlerFunc(a.handleListDeliveries)))
	mux.Handle("GET /api/v1/admin/thresholds", admin(http.HandlerFunc(a.handleGetThresholds)))
	mux.Handle("PUT /api/v1/admin/thresholds", admin(http.HandlerFunc(a.handlePutThresholds)))
	mux.Handle("GET /api/v1/admin/hosts/{id}/thresholds", admin(http.HandlerFunc(a.handleGetHostThresholds)))
	mux.Handle("PUT /api/v1/admin/hosts/{id}/thresholds", admin(http.HandlerFunc(a.handlePutHostThresholds)))
	mux.Handle("DELETE /api/v1/admin/hosts/{id}/thresholds", admin(http.HandlerFunc(a.handleResetHostThresholds)))

	return a.resolvePrincipal(a.logRequests(mux))
}
