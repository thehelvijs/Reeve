# Graph Report - .  (2026-07-27)

## Corpus Check
- 297 files · ~176,726 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 2649 nodes · 6731 edges · 161 communities (136 shown, 25 thin omitted)
- Extraction: 81% EXTRACTED · 19% INFERRED · 0% AMBIGUOUS · INFERRED: 1274 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- Agent Update Rollout
- Release Build & Signing
- Web App Shell & Auth
- Agent Self-Update & Signature Check
- SSH Push Install
- Host Command Control
- Design Tokens & Charts
- Catalog, Deploy & Product Surface
- Password Reset & Users CLI
- Auth Handler Tests
- Google OAuth Sign-In
- Credential Access Tests
- Web API Client Types
- Login Throttle
- Hosts Page & Update State
- Status Tone & Metric Widgets
- Portal Views & Grouping
- SQLite Schema
- Server Main & Background Loops
- Tool Handlers & Visibility
- Collections Store
- Users Store
- Collection Handlers
- Endpoint Redirect Tests
- Host Handlers & HTTP Core
- Admin & Metrics Handler Tests
- Process Sampling
- Log & Cron Gathering
- Alert Evaluation
- Public API Tests
- Inventory & Service Modals
- Access Requests & Grants
- Event History & Uptime UI
- Forgot-Password Tests
- Google Auth Tests
- Store Round-Trip Tests
- Icon & Avatar Upload Tests
- Update Slot Store Tests
- Disk & Rollup Store Tests
- Credential Encryption
- Agent Entrypoint
- TypeScript Config
- Telemetry Ingest Store
- SMTP Mail Sending
- Alert Dispatch Tests
- SSH Install Handlers
- Group & Backup Handlers
- Webhook Store
- Alert Threshold UI
- Visual & Accessibility Suites
- Host Metric Collection
- Settings Handlers
- Profile Handlers
- Alert Event Store
- Hosts Store
- Metrics Store
- Image Serving
- Collection Handler Tests
- App Chrome & Navigation
- Web Dev Dependencies
- Gzip & Route Table
- Command Handler Tests
- HTTP Middleware Tests
- Store Open & Restore
- Users CLI
- Admin Handlers & Audit
- Settings Handler Tests
- Credentials Store
- Web Runtime Dependencies
- Origin Check & Request Logging
- Admin Settings Page
- Cron & Docker Parsing
- Agent Push Loop
- Endpoint Resolution
- Credential Handlers
- Audit Store
- Push Payload Types
- Build Config Tests
- Auth Handlers
- Backup Handler Tests
- mDNS Resolution
- Mount Parsing
- Agent Push Tests
- Agent Download Handlers
- Download Handler Tests
- Webhook Handlers
- Threshold Handlers
- Update Slot Store
- Groups Store
- Process Usage Store
- SSH Install Handler Tests
- Agent Update Handlers
- RBAC Principals
- Metrics Handlers
- Access Request Store
- Process Store Tests
- Retention Rollup
- Schema Application
- Tool Form Page
- Mail Settings
- Docker Collection
- Collection Store Tests
- Agent Update E2E
- Secret Reveal UI
- Event History Tests
- Live SSH Install Script
- Embedded UI Assets
- Disk Store
- Web NPM Scripts
- Host Sampler Tests
- Command Handlers
- Threshold Handler Tests
- Contract Round-Trip Tests
- Notification Dispatch
- Webhook Store Tests
- Web Package Metadata
- Slug Store
- PWA Icon Generation
- Container Stats Tests
- Principal Handlers
- Network Rate Tests
- Button Alignment E2E
- Agent Dependency Boundary
- Agent Uninstall Script
- Table Legibility Rules
- Playwright Dependency
- PostCSS Dependency
- Group Views
- Tool Visibility Views
- Tailwind Dependency
- React Types
- React DOM Types
- TypeScript Dependency
- Vite Dependency
- Vite React Plugin
- E2E Port Isolation
- Vite Type Shims
- Branch Promotion Flow
- Elevation Rules
- Go Module Root
- Audit Chain Limit
- Session Invalidation

## God Nodes (most connected - your core abstractions)
1. `newTestServer()` - 220 edges
2. `signup()` - 127 edges
3. `writeError()` - 120 edges
4. `adminClient()` - 78 edges
5. `openTemp()` - 76 edges
6. `writeJSON()` - 69 edges
7. `New()` - 50 edges
8. `FromContext()` - 49 edges
9. `testServer` - 43 edges
10. `createTool()` - 40 edges

## Surprising Connections (you probably didn't know these)
- `Checksum decides currency, not version` --references--> `signedVersion()`  [INFERRED]
  deploy/README.md → agent/update.go
- `Host and service controls` --references--> `Command`  [EXTRACTED]
  README.md → contracts/contracts.go
- `server users CLI` --references--> `VerifyPassword()`  [INFERRED]
  deploy/README.md → server/internal/auth/auth.go
- `Signed agent releases` --references--> `releasePublicKey()`  [EXTRACTED]
  SECURITY.md → agent/release_key.go
- `Updates only move forward` --rationale_for--> `signedVersion()`  [EXTRACTED]
  SECURITY.md → agent/update.go

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Agent trust boundary** — security_signed_releases, security_downgrade_protection, security_agent_local_veto, security_agent_target_validation, security_agent_runs_as_root [EXTRACTED 0.95]
- **Fleet update rollout** — contracts_api_update_state, contracts_api_rollout_pause, contracts_api_forced_slot, deploy_readme_checksum_not_version, security_agent_local_veto [EXTRACTED 0.95]
- **Not leaking what you cannot see** — contracts_api_404_not_403, contracts_api_visibility, contracts_api_slug, security_single_tenant [INFERRED 0.85]

## Communities (161 total, 25 thin omitted)

### Community 0 - "Agent Update Rollout"
Cohesion: 0.07
Nodes (55): Forced update sits outside the rollout, The host's local veto is absolute, effectiveAutoUpdate(), Duration, Host, app, Time, fetchRollup() (+47 more)

### Community 1 - "Release Build & Signing"
Cohesion: 0.05
Nodes (51): CI gate job, CI grants contents:read only, Actions pinned by commit, not tag, Rolling edge prerelease, Signing key scoped to steps, not the workflow, Tag-triggered versioned release, web-build and server-assets precede build, make gate / web-check / pytest before commit (+43 more)

### Community 2 - "Web App Shell & Auth"
Cohesion: 0.08
Nodes (37): Enter confirms every form, AdminUser, ApiError, SSHTarget, uploadAvatar(), User, App(), Protected() (+29 more)

### Community 3 - "Agent Self-Update & Signature Check"
Cohesion: 0.08
Nodes (55): releasePublicKey(), fetchBody(), config, Client, isSHA256Hex(), needsUpdate(), olderThan(), selfArch() (+47 more)

### Community 4 - "SSH Push Install"
Cohesion: 0.10
Nodes (45): Builder, Channel, ClientConfig, clientConfig(), connect(), dial(), dialTCP(), envAssignments() (+37 more)

### Community 5 - "Host Command Control"
Cohesion: 0.07
Nodes (39): argvFor(), Mutex, newController(), fakeRunner(), T, TestArgvForEveryAction(), TestArgvForRefusesADangerousTarget(), TestArgvForRejectsAnUnknownAction() (+31 more)

### Community 6 - "Design Tokens & Charts"
Cohesion: 0.07
Nodes (40): Colorblind-validated chart palette, Light-only theme, Four-radius vocabulary, Two colors do two jobs, Blocking theme-boot script, Chart(), Series, ContainerPoint (+32 more)

### Community 7 - "Catalog, Deploy & Product Surface"
Cohesion: 0.06
Nodes (33): Host enrollment token, Stable slug and /go redirect, DCO sign-off, no CLA, Binds every interface by default, Docker NAT sits in front of ufw, Host networking for mDNS, SSH push install, Catalog of services (+25 more)

### Community 8 - "Password Reset & Users CLI"
Cohesion: 0.11
Nodes (34): server users CLI, cliEnv, Request, ResponseWriter, app, hashResetToken(), newResetToken(), reachableFromOtherHosts() (+26 more)

### Community 9 - "Auth Handler Tests"
Cohesion: 0.11
Nodes (42): fromIP(), Client, Response, T, login(), loginWith(), signup(), TestAuthStatusSetupFlag() (+34 more)

### Community 10 - "Google OAuth Sign-In"
Cohesion: 0.09
Nodes (27): Config, Profile, Config, Request, ResponseWriter, app, User, DomainAllowed() (+19 more)

### Community 11 - "Credential Access Tests"
Cohesion: 0.15
Nodes (41): TestAuditListsAndFilters(), createCred(), Client, T, newHost(), revealSecret(), TestAccessRequestOnAnUnknownHost(), TestAdminCanRevealAndAudited() (+33 more)

### Community 12 - "Web API Client Types"
Cohesion: 0.08
Nodes (32): AccessRequest, AgentUpdateSettings, api, ChannelKind, Collection, CollectionDetail, CommandStatus, Credential (+24 more)

### Community 13 - "Login Throttle"
Cohesion: 0.08
Nodes (34): clock, Throttle, throttleEntry, Int64, Once, Login throttle that cannot be held shut, REEVE_TRUST_PROXY is opt-in, DB (+26 more)

### Community 14 - "Hosts Page & Update State"
Cohesion: 0.08
Nodes (25): AgentUpdateRollup, AutoUpdatePolicy, Host, UpdateState, HostControls(), unavailableReason(), POLICY_LABEL, showsVersionPill() (+17 more)

### Community 15 - "Status Tone & Metric Widgets"
Cohesion: 0.10
Nodes (27): Say what the state means, ToolStatus, DiskList(), MetricBar(), HostData, HostNode(), nodeDot(), nodeTypes (+19 more)

### Community 16 - "Portal Views & Grouping"
Cohesion: 0.10
Nodes (22): Test every trigger of a modal, CollectionRef, endpointString(), Tool, CollectionInfoModal(), HostInfoModal(), statusTone(), PortalGraph() (+14 more)

### Community 17 - "SQLite Schema"
Cohesion: 0.09
Nodes (36): One schema file, re-executed on open, No protocol version to negotiate, Everything but secrets is plaintext, access_requests, alert_events, alert_state, alert_thresholds, collection_editors (+28 more)

### Community 18 - "Server Main & Background Loops"
Cohesion: 0.10
Nodes (22): compressible(), ResponseWriter, Writer, compressWriter, envOr(), every(), Context, Duration (+14 more)

### Community 19 - "Tool Handlers & Visibility"
Cohesion: 0.19
Nodes (15): Principal, CollectionRef, assetURL(), agentMonitored(), Host, Request, ResponseWriter, app (+7 more)

### Community 20 - "Collections Store"
Cohesion: 0.10
Nodes (7): placeholders(), collectionVisibleArgs(), DB, Time, VisibilityGrant, scanCollection(), Collection

### Community 21 - "Users Store"
Cohesion: 0.12
Nodes (8): Duration, DB, Time, scanUserRow(), PasswordReset, scanner, Session, User

### Community 22 - "Collection Handlers"
Cohesion: 0.21
Nodes (11): Collection, Public/restricted visibility model, decodeCollectionInput(), Request, ResponseWriter, app, VisibilityGrant, collectionDetailView (+3 more)

### Community 23 - "Endpoint Redirect Tests"
Cohesion: 0.17
Nodes (28): Client, T, noRedirect(), pushIP(), TestEmptyReportedAddressKeepsTheLastKnownOne(), TestEndpointJSON(), TestGoHidesToolsTheCallerCannotSee(), TestGoRedirectFollowsTheHostAddress() (+20 more)

### Community 24 - "Host Handlers & HTTP Core"
Cohesion: 0.15
Nodes (15): One Go binary serves UI and API, Host, Request, ResponseWriter, app, Time, hostStatus(), hostToView() (+7 more)

### Community 25 - "Admin & Metrics Handler Tests"
Cohesion: 0.16
Nodes (26): T, TestServerInfoAdminOnly(), TestUpdateUserRoleAndActive(), newTestServer(), T, TestHostMetricsEndpointCarriesDisks(), TestHostMetricsEndpointContainers(), TestHostMetricsEndpointDisksEmptyForSilentHost() (+18 more)

### Community 26 - "Process Sampling"
Cohesion: 0.17
Nodes (24): ProcessSample, Time, NewProcSampler(), ParseCmdline(), ParsePasswd(), ParseProcStat(), ParseProcStatus(), T (+16 more)

### Community 27 - "Log & Cron Gathering"
Cohesion: 0.16
Nodes (24): Time, ScanLogErrors(), durationArg(), gather(), gatherCron(), gatherDockerLogErrors(), gatherLogErrors(), config (+16 more)

### Community 28 - "Alert Evaluation"
Cohesion: 0.15
Nodes (14): fmtRate(), Duration, Host, app, Time, Webhook, hostMetricValues(), severityOrError() (+6 more)

### Community 29 - "Public API Tests"
Cohesion: 0.18
Nodes (23): TestUpdateNowRefusesAVetoedHost(), bearer(), T, TestAPIErrorEnvelope(), TestPublicEndpointsNoAuthExcludeRestricted(), TestPublicHostsNoAuth(), enrollHost(), Client (+15 more)

### Community 30 - "Inventory & Service Modals"
Cohesion: 0.15
Nodes (15): HostInventory, InventoryItem, AddServiceModal(), Chevron(), EmptyState(), Pill(), Table(), Tabs() (+7 more)

### Community 31 - "Access Requests & Grants"
Cohesion: 0.18
Nodes (10): AccessRequest, Request, ResponseWriter, app, validPrincipalType(), accessGrantView, requestView, Request (+2 more)

### Community 32 - "Event History & Uptime UI"
Cohesion: 0.12
Nodes (19): AlertEvent, goURL(), Group, UptimeSummary, EventHistory(), Layout(), UptimeSummary(), cache (+11 more)

### Community 33 - "Forgot-Password Tests"
Cohesion: 0.23
Nodes (20): configureRelay(), doReset(), forgot(), Client, Conn, Response, T, newRelay() (+12 more)

### Community 34 - "Google Auth Tests"
Cohesion: 0.40
Nodes (23): assertLoginError(), callback(), enableGoogle(), Client, Response, Server, T, noFollow() (+15 more)

### Community 35 - "Store Round-Trip Tests"
Cohesion: 0.23
Nodes (23): DB, T, openTemp(), TestBackupTo(), TestBackupToRefusesExistingDest(), TestCountToolsAndHosts(), TestDeliveryBackoff(), TestDowntimeSecs() (+15 more)

### Community 36 - "Icon & Avatar Upload Tests"
Cohesion: 0.19
Nodes (21): Client, Response, T, pngHeader(), TestEveryImageSlot(), TestHostIconVisibility(), TestToolIconVisibilityAndLifecycle(), uploadIcon() (+13 more)

### Community 37 - "Update Slot Store Tests"
Cohesion: 0.20
Nodes (22): DB, T, Time, pacedSlot(), TestApplyPushRecordsVeto(), TestApplyPushWithAVetoReleasesTheSlot(), TestClearStalledUpdatesSweepsIneligibleHostsToo(), TestFleetDefaultOffMakesADefaultPolicyHostIneligible() (+14 more)

### Community 38 - "Disk & Rollup Store Tests"
Cohesion: 0.19
Nodes (20): T, TestApplyPushStoresDisks(), TestDeleteHostMetricsDropsDisks(), TestHostDisksNilStoresEmptyList(), TestHostDisksRoundTripAndReplace(), TestLatestHostDisksUnknownHost(), T, TestApplyPushKeepsLastKnownChecksum() (+12 more)

### Community 39 - "Credential Encryption"
Cohesion: 0.22
Nodes (18): AEAD, Cipher, Compose refuses to start without the master key, Backup carries ciphertext, not the key, Credentials encrypted before they touch disk, Master-key custody, New(), NewFromEnv() (+10 more)

### Community 40 - "Agent Entrypoint"
Cohesion: 0.20
Nodes (20): commandReport(), config, Duration, Time, loadConfig(), main(), runOnce(), runSelfUpdate() (+12 more)

### Community 41 - "TypeScript Config"
Cohesion: 0.09
Nodes (21): DOM, DOM.Iterable, ES2021, src, compilerOptions, allowImportingTsExtensions, isolatedModules, jsx (+13 more)

### Community 42 - "Telemetry Ingest Store"
Cohesion: 0.17
Nodes (11): ContainerState, DB, Time, insertLogEvents(), replaceContainerStatus(), replaceCronJobs(), replaceServiceStatus(), InventoryContainer (+3 more)

### Community 43 - "SMTP Mail Sending"
Cohesion: 0.21
Nodes (18): Config, Message, stubSMTP, BuildMessage(), Time, hasCRLF(), Send(), atoi() (+10 more)

### Community 44 - "Alert Dispatch Tests"
Cohesion: 0.21
Nodes (19): countRows(), enrollHostWin(), Client, T, TestAgentOfflineAlert(), TestDispatchSendsAndRetries(), TestDownAlertDebounceFireResolve(), TestFlapSuppressed() (+11 more)

### Community 45 - "SSH Install Handlers"
Cohesion: 0.16
Nodes (10): errNoAgentBuild, errUnsupportedArch, decodeJSON(), Request, ResponseWriter, app, Request, ResponseWriter (+2 more)

### Community 46 - "Group & Backup Handlers"
Cohesion: 0.20
Nodes (10): Request, ResponseWriter, app, Request, ResponseWriter, app, writeError(), StagedRestorePath() (+2 more)

### Community 47 - "Webhook Store"
Cohesion: 0.20
Nodes (7): channelAccepts(), filterBySeverity(), DB, Time, severityRank(), Delivery, Webhook

### Community 48 - "Alert Threshold UI"
Cohesion: 0.17
Nodes (17): Delivery, ThresholdMetric, ThresholdsPayload, EMPTY_ROW, METRIC_KEYS, METRICS, ThresholdRow, ThresholdRows (+9 more)

### Community 49 - "Visual & Accessibility Suites"
Cohesion: 0.10
Nodes (8): CI Playwright e2e job, Frontend changes are reviewed in a browser, axe against WCAG 2 A/AA, Manual screenshot sweep after any frontend change, Pixel baselines for every page, TAGS, THEMES, THEMES

### Community 50 - "Host Metric Collection"
Cohesion: 0.19
Nodes (16): TestParseCPUStatAndPercent(), TestParseUptimeAndNetDev(), diskUsage(), DiskUsage, HostMetrics, sampleDisks(), sampleGPU(), CPUPercent() (+8 more)

### Community 51 - "Settings Handlers"
Cohesion: 0.15
Nodes (13): Agent update state machine, Checksum decides currency, not version, Retention, agentUpdateView, googleAuthView, googleInput, retentionView, Request (+5 more)

### Community 52 - "Profile Handlers"
Cohesion: 0.20
Nodes (8): ReadCloser, Request, ResponseWriter, readImageUpload(), formFile(), Request, ResponseWriter, app

### Community 53 - "Alert Event Store"
Cohesion: 0.21
Nodes (8): Rows, DB, Time, scanAlertEvents(), Time, parseNullableTime(), AlertEvent, AlertState

### Community 54 - "Hosts Store"
Cohesion: 0.16
Nodes (3): DB, Time, Host

### Community 55 - "Metrics Store"
Cohesion: 0.22
Nodes (9): ContainerSample, HostMetrics, DB, Time, insertContainerStats(), insertHostMetric(), scanLatestMetric(), ContainerPoint (+1 more)

### Community 56 - "Image Serving"
Cohesion: 0.25
Nodes (12): alwaysEditable(), Request, ResponseWriter, app, hostCanSee(), hostTarget(), toolCanEdit(), toolCanSee() (+4 more)

### Community 57 - "Collection Handler Tests"
Cohesion: 0.27
Nodes (16): containsKey(), containsName(), Client, T, hasKey(), hasName(), jsonString(), newUser() (+8 more)

### Community 58 - "App Chrome & Navigation"
Cohesion: 0.16
Nodes (8): Avatar(), initials(), adminNav, primaryNav, IconName, paths, SearchBar(), Wordmark()

### Community 59 - "Web Dev Dependencies"
Cohesion: 0.12
Nodes (17): autoprefixer, @axe-core/playwright, eslint, eslint-plugin-react-hooks, @types/leaflet, @typescript-eslint/eslint-plugin, @typescript-eslint/parser, vitest (+9 more)

### Community 60 - "Gzip & Route Table"
Cohesion: 0.26
Nodes (15): HandlerFunc, Handler, Handler, gzipResponses(), Handler, Response, T, gzipRequest() (+7 more)

### Community 61 - "Command Handler Tests"
Cohesion: 0.32
Nodes (15): controllableHost(), Client, T, queue(), TestAResultFromTheWrongHostIsIgnored(), TestCommandsAreAdminOnly(), TestQueueAndDeliverACommand(), TestQueuePowerActionsNeedNoTarget() (+7 more)

### Community 62 - "HTTP Middleware Tests"
Cohesion: 0.23
Nodes (14): captureLog(), Buffer, Client, Response, T, newTestServerKey(), TestClientIPForwardedForOnlyWhenProxyTrusted(), TestHealthz() (+6 more)

### Community 63 - "Store Open & Restore"
Cohesion: 0.18
Nodes (8): ApplyStagedRestore(), DB, Open(), TestApplyStagedRestore(), TestApplyStagedRestoreNoOp(), TestValidateBackupRejectsAForeignDatabaseWithoutTouchingIt(), ValidateBackup(), Tx

### Community 64 - "Users CLI"
Cohesion: 0.43
Nodes (16): deletionHeir(), firstArg(), DB, User, Writer, guardLastActiveAdmin(), lookup(), normalizeEmail() (+8 more)

### Community 65 - "Admin Handlers & Audit"
Cohesion: 0.28
Nodes (6): Every reveal is audited, Request, ResponseWriter, app, User, adminUserView

### Community 66 - "Settings Handler Tests"
Cohesion: 0.28
Nodes (15): app, Server, getSettings(), Client, Response, T, putSettings(), TestSealedSettingRoundTrip() (+7 more)

### Community 67 - "Credentials Store"
Cohesion: 0.18
Nodes (5): DB, Time, scanCredentialMeta(), AccessGrant, Credential

### Community 68 - "Web Runtime Dependencies"
Cohesion: 0.13
Nodes (15): @fontsource-variable/inter, leaflet, react, react-dom, react-router-dom, uplot, dependencies, @fontsource-variable/inter (+7 more)

### Community 69 - "Origin Check & Request Logging"
Cohesion: 0.23
Nodes (8): Writes must come from this origin, dynamicRequest(), Handler, Request, app, User, User, PrincipalFromUser()

### Community 70 - "Admin Settings Page"
Cohesion: 0.16
Nodes (10): GoogleInput, Settings, SettingsInput, SMTPInput, AgentUpdateSection(), EmailSection(), numOrEmpty(), RETENTION_CHOICES (+2 more)

### Community 71 - "Cron & Docker Parsing"
Cohesion: 0.25
Nodes (12): T, TestParseCrontabSystemForm(), TestParseCrontabUserForm(), TestParseDockerPS(), TestParseDockerStats(), TestParseLoadAvg(), TestParseMemInfo(), TestParseNvidiaSMI() (+4 more)

### Community 72 - "Agent Push Loop"
Cohesion: 0.21
Nodes (8): Client, permanentReject(), randSuffix(), pusher, statusError, Ingest push contract, Commands ride the push ack, PushAck

### Community 73 - "Endpoint Resolution"
Cohesion: 0.27
Nodes (9): 404 instead of 403 for invisible records, Follow-the-host endpoint resolution, Host, Request, ResponseWriter, app, Tool, resolveEndpoint() (+1 more)

### Community 74 - "Credential Handlers"
Cohesion: 0.36
Nodes (5): Credential, Host, Request, ResponseWriter, app

### Community 75 - "Audit Store"
Cohesion: 0.23
Nodes (7): auditWhere(), DB, Time, NewID(), AuditFilter, GrantAudit, RevealAudit

### Community 76 - "Push Payload Types"
Cohesion: 0.18
Nodes (10): ParseSystemctl(), HostMetrics, ProcessSample, Time, CronState, Push, ServiceState, T (+2 more)

### Community 77 - "Build Config Tests"
Cohesion: 0.23
Nodes (8): parametrize, dry_run(), job_commands(), test_actions_are_pinned_to_a_commit(), test_npm_target_installs_its_own_deps(), test_signing_key_is_never_workflow_or_job_scoped(), test_workflow_jobs_install_web_deps_before_running_npm_scripts(), test_workflows_grant_no_blanket_permissions()

### Community 78 - "Auth Handlers"
Cohesion: 0.40
Nodes (4): Request, ResponseWriter, app, User

### Community 79 - "Backup Handler Tests"
Cohesion: 0.41
Nodes (12): downloadBackup(), Client, Response, T, TestBackupAndRestoreAreAdminOnly(), TestBackupDownloadIsAUsableDatabase(), TestBackupKeepsCredentialsEncrypted(), TestRestoreCancelDiscardsStaged() (+4 more)

### Community 80 - "mDNS Resolution"
Cohesion: 0.32
Nodes (11): Context, lookupMDNS(), mdnsAnswer(), mdnsQuery(), T, response(), TestMDNSAnswerIgnoresOtherHosts(), TestMDNSAnswerPicksTheRequestedName() (+3 more)

### Community 81 - "Mount Parsing"
Cohesion: 0.32
Nodes (10): ParseMounts(), T, TestParseMountsDecodesOctalEscapes(), TestParseMountsDedupesByDevice(), TestParseMountsIgnoresGarbage(), TestParseMountsKeepsRealFilesystems(), TestUnescapeMountLeavesMalformedEscapes(), unescapeMount() (+2 more)

### Community 82 - "Agent Push Tests"
Cohesion: 0.39
Nodes (11): config, newPusher(), T, seedBuffer(), TestFlushBufferDropsAPermanentlyRejectedBody(), TestFlushBufferKeepsEverythingOn401(), TestFlushBufferKeepsEverythingOnServerError(), TestMalformedAckIsNotContact() (+3 more)

### Community 83 - "Agent Download Handlers"
Cohesion: 0.44
Nodes (4): File, Request, ResponseWriter, app

### Community 84 - "Download Handler Tests"
Cohesion: 0.36
Nodes (11): MapFS, app, T, newDownloadApp(), TestRoutesDownloadDispatch(), TestServeAgentBinaryKnownArch(), TestServeAgentBinaryUnknownArch404(), TestServeAgentChecksumMatchesBytes() (+3 more)

### Community 85 - "Webhook Handlers"
Cohesion: 0.32
Nodes (5): channelView, Request, ResponseWriter, app, Webhook

### Community 86 - "Threshold Handlers"
Cohesion: 0.33
Nodes (5): Request, ResponseWriter, app, thresholdDTO, thresholdsPayload

### Community 87 - "Update Slot Store"
Cohesion: 0.24
Nodes (5): blockingSlot(), DB, Time, ValidAutoUpdatePolicy(), StalledHost

### Community 88 - "Groups Store"
Cohesion: 0.20
Nodes (3): DB, Time, Group

### Community 89 - "Process Usage Store"
Cohesion: 0.33
Nodes (8): accumulateProcessUsage(), ProcessSample, DB, Time, replaceHostProcesses(), execer, HostProcesses, ProcessUsage

### Community 90 - "SSH Install Handler Tests"
Cohesion: 0.39
Nodes (11): createHostFor(), Client, T, sshBody(), TestAgentBinaryForUname(), TestReachableFromOtherHosts(), TestSSHEndpointsAreAdminOnlyAndScopedToAHost(), TestSSHInstallRejectsIncompleteTargets() (+3 more)

### Community 91 - "Agent Update Handlers"
Cohesion: 0.29
Nodes (6): A stalled host pauses the whole rollout, Request, ResponseWriter, app, StalledHost, agentUpdateRollup

### Community 92 - "RBAC Principals"
Cohesion: 0.27
Nodes (10): Session cookie auth, ctxKey, Single-tenant, admin sees everything, Context, Handler, ResponseWriter, RequireAdmin(), RequireAuth() (+2 more)

### Community 93 - "Metrics Handlers"
Cohesion: 0.38
Nodes (4): Request, ResponseWriter, app, uptimeResponse

### Community 94 - "Access Request Store"
Cohesion: 0.35
Nodes (4): DB, Time, scanRequest(), AccessRequest

### Community 95 - "Process Store Tests"
Cohesion: 0.33
Nodes (10): T, TestDeleteHostMetricsDropsProcesses(), TestDeleteHostMetricsDropsProcessUsage(), TestHostProcessesNilStoresEmptyList(), TestHostProcessesRoundTripAndOverwrite(), TestLatestHostProcessesUnknownHost(), TestProcessUsageAveragesAndPeaks(), TestProcessUsageKeepsTopByMemory() (+2 more)

### Community 96 - "Retention Rollup"
Cohesion: 0.36
Nodes (4): Duration, DB, Time, Retention

### Community 97 - "Schema Application"
Cohesion: 0.25
Nodes (6): declaredColumns(), DB, T, TestDeclaredColumnsReadsTheSchema(), TestOpenAcceptsACurrentDatabase(), TestOpenRefusesADatabaseMissingAColumn()

### Community 98 - "Tool Form Page"
Cohesion: 0.22
Nodes (5): BackLink(), EndpointMode, slugPreview(), SOURCE_TYPES, ToolFormPage()

### Community 99 - "Mail Settings"
Cohesion: 0.29
Nodes (4): Config, app, smtpInput, smtpView

### Community 100 - "Docker Collection"
Cohesion: 0.33
Nodes (8): healthFromStatus(), parseBytes(), ParseDockerPS(), ParseDockerStats(), parseMemUsage(), parsePercent(), dockerPSLine, dockerStatsLine

### Community 101 - "Collection Store Tests"
Cohesion: 0.50
Nodes (8): DB, T, mustUser(), TestCollectionCRUD(), TestCollectionEditRights(), TestCollectionMembership(), TestCollectionVisibility(), TestDeleteGroupClearsCollectionGrants()

### Community 103 - "Secret Reveal UI"
Cohesion: 0.44
Nodes (6): RevealModal(), SecretField(), isSensitiveField(), ORDER, orderedSecretFields(), SENSITIVE

### Community 104 - "Event History Tests"
Cohesion: 0.36
Nodes (7): AlertEvent, T, Time, nowUTC(), storeEvent(), TestHostEventsEndpoint(), TestToolEventsHiddenForOutsider()

### Community 105 - "Live SSH Install Script"
Cohesion: 0.43
Nodes (6): api(), bad(), cleanup_host(), ok(), remote(), live_ssh_install_test.sh script

### Community 106 - "Embedded UI Assets"
Cohesion: 0.29
Nodes (5): agentDistFS(), assetCacheControl(), FS, Handler, app

### Community 107 - "Disk Store"
Cohesion: 0.43
Nodes (5): DiskUsage, DB, Time, replaceHostDisks(), HostDisks

### Community 108 - "Web NPM Scripts"
Cohesion: 0.25
Nodes (8): scripts, build, dev, e2e, lint, preview, test, typecheck

### Community 109 - "Host Sampler Tests"
Cohesion: 0.38
Nodes (5): NewHostSampler(), T, TestHostSamplerMeasuresCPUAcrossCalls(), TestSampleHostMetricsDoesNotPanicAndReportsMemory(), HostMetrics

### Community 110 - "Command Handlers"
Cohesion: 0.48
Nodes (4): HostCommand, Request, ResponseWriter, app

### Community 111 - "Threshold Handler Tests"
Cohesion: 0.62
Nodes (6): effectiveThreshold(), T, TestHostThresholdsAdminGetSet(), TestHostThresholdsReset(), TestHostThresholdsUnknownHost(), TestThresholdsAdminGetSet()

### Community 112 - "Contract Round-Trip Tests"
Cohesion: 0.53
Nodes (5): T, TestPushAckRoundTrip(), TestPushCarriesAutoUpdateVeto(), TestPushRoundTrip(), TestPushTooLargeNamesTheOffendingSection()

### Community 114 - "Webhook Store Tests"
Cohesion: 0.53
Nodes (5): T, TestChannelAcceptsSeverity(), TestChannelColumnsRoundTrip(), TestChannelDefaults(), TestGlobalChannelsFilterBySeverity()

### Community 115 - "Web Package Metadata"
Cohesion: 0.33
Nodes (5): license, name, private, type, version

### Community 118 - "PWA Icon Generation"
Cohesion: 0.67
Nodes (3): Image, main(), render()

### Community 119 - "Container Stats Tests"
Cohesion: 0.67
Nodes (3): T, TestContainerStatsNameFallback(), TestContainerStatsRoundTrip()

### Community 120 - "Principal Handlers"
Cohesion: 0.83
Nodes (3): principalGroup, principalsView, principalUser

## Ambiguous Edges - Review These
- `LAN-only, no external backend` → `DCO sign-off, no CLA`  [AMBIGUOUS]
  CONTRIBUTING.md · relation: conceptually_related_to
- `Light-only theme` → `Blocking theme-boot script`  [AMBIGUOUS]
  DESIGN.md · relation: conceptually_related_to

## Knowledge Gaps
- **148 isolated node(s):** `dockerPSLine`, `dockerStatsLine`, `config`, `DiskUsage`, `ProcessSample` (+143 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **25 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **What is the exact relationship between `LAN-only, no external backend` and `DCO sign-off, no CLA`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **What is the exact relationship between `Light-only theme` and `Blocking theme-boot script`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **Why does `New()` connect `Credential Encryption` to `Agent Update Rollout`, `Agent Self-Update & Signature Check`, `SSH Push Install`, `Host Command Control`, `Password Reset & Users CLI`, `Google OAuth Sign-In`, `Login Throttle`, `Google Auth Tests`, `SMTP Mail Sending`, `Alert Dispatch Tests`, `SSH Install Handlers`, `Settings Handlers`, `Profile Handlers`, `HTTP Middleware Tests`, `Store Open & Restore`, `Users CLI`, `Origin Check & Request Logging`, `Push Payload Types`, `Agent Push Tests`, `Agent Download Handlers`, `Mail Settings`, `Notification Dispatch`?**
  _High betweenness centrality (0.293) - this node is a cross-community bridge._
- **Why does `Public portal front door` connect `Catalog, Deploy & Product Surface` to `Portal Views & Grouping`?**
  _High betweenness centrality (0.259) - this node is a cross-community bridge._
- **Are the 214 inferred relationships involving `newTestServer()` (e.g. with `TestAuditListsAndFilters()` and `TestServerInfoAdminOnly()`) actually correct?**
  _`newTestServer()` has 214 INFERRED edges - model-reasoned connections that need verification._
- **Are the 108 inferred relationships involving `signup()` (e.g. with `TestAuditListsAndFilters()` and `TestServerInfoAdminOnly()`) actually correct?**
  _`signup()` has 108 INFERRED edges - model-reasoned connections that need verification._
- **Are the 116 inferred relationships involving `writeError()` (e.g. with `validPrincipalType()` and `.applyCollectionIDs()`) actually correct?**
  _`writeError()` has 116 INFERRED edges - model-reasoned connections that need verification._