# Graph Report - Reeve  (2026-07-27)

## Corpus Check
- 297 files · ~180,813 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 2676 nodes · 6804 edges · 159 communities (133 shown, 26 thin omitted)
- Extraction: 81% EXTRACTED · 19% INFERRED · 0% AMBIGUOUS · INFERRED: 1288 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `a0cba48e`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- publishAgent
- release.py
- ui.tsx
- signing.go
- Install
- control_test.go
- HostMetrics.tsx
- .handleIngest
- newCLIEnv
- signup
- google_test.go
- adminClient
- api.ts
- app
- hostFilter.ts
- Dashboard.tsx
- Portal.tsx
- schema.sql
- run
- Principal
- DB
- DB
- FromContext
- createTool
- hostToView
- newTestServer
- procs_test.go
- gather
- DB
- samplePush
- AdminServerInfo.tsx
- app
- Layout.tsx
- T
- google_auth_test.go
- openTemp
- profile_handlers_test.go
- store/agent_update_test.go
- seedHost
- New
- runOnce
- compilerOptions
- .ApplyPush
- Send
- countRows
- .handleSSHInstall
- DB
- DB
- HostInventory.tsx
- zzz-visual.spec.ts
- collect_test.go
- .handlePutSettings
- writeError
- DB
- DB
- ToolFormPage.tsx
- app
- newUser
- .handleLogin
- devDependencies
- HandlerFunc
- Push
- T
- Open
- users_cli.go
- app
- testServer
- DB
- dependencies
- app
- dbWithUser
- .handleSetHostAutoUpdate
- .handleCreateCommand
- .loadEndpoint
- .handleCreateCredential
- NewID
- shutdown_test.go
- test_build_config.py
- adminResetPassword
- downloadBackup
- lookupMDNS
- compressWriter
- newPusher
- app
- newDownloadApp
- .handleCreateWebhook
- decodeJSON
- DB
- DB
- TestPublicHostsNoAuth
- vitest
- rbac.go
- writeJSON
- AccessRequest
- processes_test.go
- DB
- .checkSchemaDrift
- useAuth
- .sendMail
- tailwindcss
- mustUser
- zz-agent-updates.spec.ts
- RevealModal.tsx
- TestHostEventsEndpoint
- live_ssh_install_test.sh
- .uiHandler
- scripts
- effectiveThreshold
- contracts_test.go
- .dispatchDue
- webhooks_test.go
- package.json
- DB
- render
- TestContainerStatsNameFallback
- principal_handlers.go
- TestHostNetRate
- zzz-button-alignment.spec.ts
- agent/ must not import server/
- uninstall.sh
- Every column has a name
- @playwright/test
- group_handlers.go
- tool_visibility_handlers.go
- @types/react-dom
- @types/react
- vite
- @vitejs/plugin-react
- e2e pins REEVE_DEV_PORT=8099
- vite-env.d.ts
- develop integrates, main is production
- Structure over shadow
- github.com/thehelvijs/Reeve
- Audit tables are append-only by convention
- Password change signs out everywhere
- zz-header-search.spec.ts
- typescript
- postWebhook
- .handleRestoreUpload

## God Nodes (most connected - your core abstractions)
1. `newTestServer()` - 221 edges
2. `signup()` - 127 edges
3. `writeError()` - 120 edges
4. `adminClient()` - 79 edges
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

## Communities (159 total, 26 thin omitted)

### Community 0 - "publishAgent"
Cohesion: 0.07
Nodes (55): Forced update sits outside the rollout, The host's local veto is absolute, effectiveAutoUpdate(), Duration, Host, app, Time, fetchRollup() (+47 more)

### Community 1 - "release.py"
Cohesion: 0.05
Nodes (51): CI gate job, CI grants contents:read only, Actions pinned by commit, not tag, Rolling edge prerelease, Signing key scoped to steps, not the workflow, Tag-triggered versioned release, web-build and server-assets precede build, make gate / web-check / pytest before commit (+43 more)

### Community 2 - "ui.tsx"
Cohesion: 0.10
Nodes (35): Enter confirms every form, AdminUser, AlertEvent, api, uploadAvatar(), ConfirmModal(), EmptyState(), EventHistory() (+27 more)

### Community 3 - "signing.go"
Cohesion: 0.08
Nodes (55): releasePublicKey(), fetchBody(), config, Client, isSHA256Hex(), needsUpdate(), olderThan(), selfArch() (+47 more)

### Community 4 - "Install"
Cohesion: 0.10
Nodes (45): Builder, Channel, ClientConfig, clientConfig(), connect(), dial(), dialTCP(), envAssignments() (+37 more)

### Community 5 - "control_test.go"
Cohesion: 0.29
Nodes (15): argvFor(), newController(), fakeRunner(), T, TestArgvForEveryAction(), TestArgvForRefusesADangerousTarget(), TestArgvForRejectsAnUnknownAction(), TestControlDefaultsOn() (+7 more)

### Community 6 - "HostMetrics.tsx"
Cohesion: 0.07
Nodes (41): Colorblind-validated chart palette, Light-only theme, Four-radius vocabulary, Two colors do two jobs, Blocking theme-boot script, Chart(), Series, ContainerPoint (+33 more)

### Community 7 - ".handleIngest"
Cohesion: 0.12
Nodes (15): Host enrollment token, DCO sign-off, no CLA, Binds every interface by default, Docker NAT sits in front of ufw, Host networking for mDNS, SSH push install, Catalog of services, Host and service controls (+7 more)

### Community 8 - "newCLIEnv"
Cohesion: 0.08
Nodes (47): server users CLI, cliEnv, Request, ResponseWriter, app, hashResetToken(), newResetToken(), reachableFromOtherHosts() (+39 more)

### Community 9 - "signup"
Cohesion: 0.12
Nodes (37): T, TestServerInfoAdminOnly(), TestUpdateUserRoleAndActive(), fromIP(), Client, Response, T, login() (+29 more)

### Community 10 - "google_test.go"
Cohesion: 0.09
Nodes (27): Config, Profile, Config, Request, ResponseWriter, app, User, DomainAllowed() (+19 more)

### Community 11 - "adminClient"
Cohesion: 0.15
Nodes (41): TestAuditListsAndFilters(), createCred(), Client, T, newHost(), revealSecret(), TestAccessRequestOnAnUnknownHost(), TestAdminCanRevealAndAudited() (+33 more)

### Community 12 - "api.ts"
Cohesion: 0.06
Nodes (34): AccessRequest, AgentUpdateRollup, AgentUpdateSettings, ChannelKind, Collection, CollectionDetail, CommandStatus, Credential (+26 more)

### Community 13 - "app"
Cohesion: 0.09
Nodes (31): clock, Throttle, throttleEntry, Int64, Once, Login throttle that cannot be held shut, REEVE_TRUST_PROXY is opt-in, DB (+23 more)

### Community 14 - "hostFilter.ts"
Cohesion: 0.32
Nodes (5): showsVersionPill(), HOST_FILTERS, HostFilter, hostMatchesFilter(), Hosts()

### Community 15 - "Dashboard.tsx"
Cohesion: 0.14
Nodes (20): Say what the state means, ToolStatus, HostData, HostNode(), nodeDot(), nodeTypes, SvcData, ServiceDots() (+12 more)

### Community 16 - "Portal.tsx"
Cohesion: 0.08
Nodes (30): Test every trigger of a modal, CollectionRef, endpointString(), goURL(), Group, Host, HostInventory, InventoryItem (+22 more)

### Community 17 - "schema.sql"
Cohesion: 0.09
Nodes (36): One schema file, re-executed on open, No protocol version to negotiate, Everything but secrets is plaintext, access_requests, alert_events, alert_state, alert_thresholds, collection_editors (+28 more)

### Community 18 - "run"
Cohesion: 0.23
Nodes (11): envOr(), every(), Context, Duration, Server, app, main(), probeHealth() (+3 more)

### Community 19 - "Principal"
Cohesion: 0.15
Nodes (19): Principal, CollectionRef, ToolDTO, ToolStatus, assetURL(), publicToolResponse, agentMonitored(), Host (+11 more)

### Community 20 - "DB"
Cohesion: 0.10
Nodes (7): placeholders(), collectionVisibleArgs(), DB, Time, VisibilityGrant, scanCollection(), Collection

### Community 21 - "DB"
Cohesion: 0.12
Nodes (8): Duration, DB, Time, scanUserRow(), PasswordReset, scanner, Session, User

### Community 22 - "FromContext"
Cohesion: 0.21
Nodes (11): Collection, Public/restricted visibility model, decodeCollectionInput(), Request, ResponseWriter, app, VisibilityGrant, collectionDetailView (+3 more)

### Community 23 - "createTool"
Cohesion: 0.18
Nodes (27): Client, T, noRedirect(), pushIP(), TestEmptyReportedAddressKeepsTheLastKnownOne(), TestEndpointJSON(), TestGoHidesToolsTheCallerCannotSee(), TestGoRedirectFollowsTheHostAddress() (+19 more)

### Community 24 - "hostToView"
Cohesion: 0.28
Nodes (8): Host, Request, ResponseWriter, app, Time, hostStatus(), hostToView(), updateContext

### Community 25 - "newTestServer"
Cohesion: 0.20
Nodes (23): newTestServer(), T, TestHostMetricsEndpointCarriesDisks(), TestHostMetricsEndpointContainers(), TestHostMetricsEndpointDisksEmptyForSilentHost(), TestHostProcessUsageEndpoint(), TestHostProcessUsageUnknownHost(), TestHostUptimeEndpoint() (+15 more)

### Community 26 - "procs_test.go"
Cohesion: 0.11
Nodes (33): ProcessSample, Time, NewProcSampler(), ParseCmdline(), ParsePasswd(), ParseProcStat(), ParseProcStatus(), T (+25 more)

### Community 27 - "gather"
Cohesion: 0.15
Nodes (25): TestScanLogErrors(), Time, ScanLogErrors(), durationArg(), gather(), gatherCron(), gatherDockerLogErrors(), gatherLogErrors() (+17 more)

### Community 28 - "DB"
Cohesion: 0.07
Nodes (21): fmtRate(), Duration, Host, app, Time, Webhook, hostMetricValues(), severityOrError() (+13 more)

### Community 29 - "samplePush"
Cohesion: 0.17
Nodes (31): TestUpdateNowRefusesAVetoedHost(), controllableHost(), Client, T, queue(), TestAResultFromTheWrongHostIsIgnored(), TestCommandsAreAdminOnly(), TestQueueAndDeliverACommand() (+23 more)

### Community 30 - "AdminServerInfo.tsx"
Cohesion: 0.21
Nodes (10): DiskList(), MetricBar(), fmtBytes(), fmtCount(), fmtUptime(), AdminServerInfo(), ServerInfo, CollectionDetail() (+2 more)

### Community 31 - "app"
Cohesion: 0.18
Nodes (10): AccessRequest, Request, ResponseWriter, app, validPrincipalType(), accessGrantView, requestView, Request (+2 more)

### Community 32 - "Layout.tsx"
Cohesion: 0.09
Nodes (20): UptimeSummary, Avatar(), initials(), adminNav, Layout(), primaryNav, IconName, paths (+12 more)

### Community 33 - "T"
Cohesion: 0.26
Nodes (19): configureRelay(), doReset(), forgot(), Client, Response, T, newRelay(), resetTokenFrom() (+11 more)

### Community 34 - "google_auth_test.go"
Cohesion: 0.40
Nodes (23): assertLoginError(), callback(), enableGoogle(), Client, Response, Server, T, noFollow() (+15 more)

### Community 35 - "openTemp"
Cohesion: 0.23
Nodes (23): DB, T, openTemp(), TestBackupTo(), TestBackupToRefusesExistingDest(), TestCountToolsAndHosts(), TestDeliveryBackoff(), TestDowntimeSecs() (+15 more)

### Community 36 - "profile_handlers_test.go"
Cohesion: 0.19
Nodes (21): Client, Response, T, pngHeader(), TestEveryImageSlot(), TestHostIconVisibility(), TestToolIconVisibilityAndLifecycle(), uploadIcon() (+13 more)

### Community 37 - "store/agent_update_test.go"
Cohesion: 0.20
Nodes (22): DB, T, Time, pacedSlot(), TestApplyPushRecordsVeto(), TestApplyPushWithAVetoReleasesTheSlot(), TestClearStalledUpdatesSweepsIneligibleHostsToo(), TestFleetDefaultOffMakesADefaultPolicyHostIneligible() (+14 more)

### Community 38 - "seedHost"
Cohesion: 0.19
Nodes (20): T, TestApplyPushStoresDisks(), TestDeleteHostMetricsDropsDisks(), TestHostDisksNilStoresEmptyList(), TestHostDisksRoundTripAndReplace(), TestLatestHostDisksUnknownHost(), T, TestApplyPushKeepsLastKnownChecksum() (+12 more)

### Community 39 - "New"
Cohesion: 0.22
Nodes (18): AEAD, Cipher, Compose refuses to start without the master key, Backup carries ciphertext, not the key, Credentials encrypted before they touch disk, Master-key custody, New(), NewFromEnv() (+10 more)

### Community 40 - "runOnce"
Cohesion: 0.20
Nodes (20): commandReport(), config, Duration, Time, loadConfig(), main(), runOnce(), runSelfUpdate() (+12 more)

### Community 41 - "compilerOptions"
Cohesion: 0.09
Nodes (21): DOM, DOM.Iterable, ES2021, src, compilerOptions, allowImportingTsExtensions, isolatedModules, jsx (+13 more)

### Community 42 - ".ApplyPush"
Cohesion: 0.06
Nodes (33): ContainerSample, ContainerState, DiskUsage, DB, Time, replaceHostDisks(), HostMetrics, DB (+25 more)

### Community 43 - "Send"
Cohesion: 0.21
Nodes (18): Config, Message, stubSMTP, BuildMessage(), Time, hasCRLF(), Send(), atoi() (+10 more)

### Community 44 - "countRows"
Cohesion: 0.35
Nodes (13): countRows(), enrollHostWin(), Client, T, TestAgentOfflineAlert(), TestDispatchSendsAndRetries(), TestDownAlertDebounceFireResolve(), TestFlapSuppressed() (+5 more)

### Community 45 - ".handleSSHInstall"
Cohesion: 0.22
Nodes (6): errNoAgentBuild, errUnsupportedArch, Request, ResponseWriter, app, sshTargetInput

### Community 46 - "DB"
Cohesion: 0.23
Nodes (5): NullString, DB, Time, parseNullTime(), HostCommand

### Community 47 - "DB"
Cohesion: 0.20
Nodes (7): channelAccepts(), filterBySeverity(), DB, Time, severityRank(), Delivery, Webhook

### Community 48 - "HostInventory.tsx"
Cohesion: 0.07
Nodes (39): AutoUpdatePolicy, Delivery, HostCommand, ThresholdMetric, ThresholdsPayload, UpdateState, ACTION_LABEL, HostControls() (+31 more)

### Community 49 - "zzz-visual.spec.ts"
Cohesion: 0.10
Nodes (8): CI Playwright e2e job, Frontend changes are reviewed in a browser, axe against WCAG 2 A/AA, Manual screenshot sweep after any frontend change, Pixel baselines for every page, TAGS, THEMES, THEMES

### Community 50 - "collect_test.go"
Cohesion: 0.06
Nodes (52): T, TestParseCPUStatAndPercent(), TestParseCrontabSystemForm(), TestParseCrontabUserForm(), TestParseDockerPS(), TestParseDockerStats(), TestParseLoadAvg(), TestParseMemInfo() (+44 more)

### Community 51 - ".handlePutSettings"
Cohesion: 0.15
Nodes (13): Agent update state machine, Checksum decides currency, not version, Retention, agentUpdateView, googleAuthView, googleInput, retentionView, Request (+5 more)

### Community 52 - "writeError"
Cohesion: 0.25
Nodes (7): Request, ResponseWriter, app, writeError(), Request, ResponseWriter, app

### Community 53 - "DB"
Cohesion: 0.21
Nodes (8): Rows, DB, Time, scanAlertEvents(), Time, parseNullableTime(), AlertEvent, AlertState

### Community 54 - "DB"
Cohesion: 0.16
Nodes (3): DB, Time, Host

### Community 55 - "ToolFormPage.tsx"
Cohesion: 0.10
Nodes (15): ApiError, SSHTarget, uploadIcon(), BackLink(), IconUploader(), message(), EMPTY, Mode (+7 more)

### Community 56 - "app"
Cohesion: 0.19
Nodes (15): alwaysEditable(), Request, ResponseWriter, app, hostCanSee(), hostTarget(), toolCanEdit(), toolCanSee() (+7 more)

### Community 57 - "newUser"
Cohesion: 0.27
Nodes (16): containsKey(), containsName(), Client, T, hasKey(), hasName(), jsonString(), newUser() (+8 more)

### Community 58 - ".handleLogin"
Cohesion: 0.40
Nodes (4): Request, ResponseWriter, app, User

### Community 59 - "devDependencies"
Cohesion: 0.12
Nodes (17): autoprefixer, @axe-core/playwright, eslint, eslint-plugin-react-hooks, postcss, @types/leaflet, @typescript-eslint/eslint-plugin, @typescript-eslint/parser (+9 more)

### Community 60 - "HandlerFunc"
Cohesion: 0.26
Nodes (15): HandlerFunc, Handler, Handler, gzipResponses(), Handler, Response, T, gzipRequest() (+7 more)

### Community 61 - "Push"
Cohesion: 0.09
Nodes (27): Mutex, controller, Fixed action allowlist, no shell, Collections replace free-text category, Single error envelope, Ingest push contract, Commands ride the push ack, CollectionRef (+19 more)

### Community 62 - "T"
Cohesion: 0.23
Nodes (14): captureLog(), Buffer, Client, Response, T, newTestServerKey(), TestClientIPForwardedForOnlyWhenProxyTrusted(), TestHealthz() (+6 more)

### Community 63 - "Open"
Cohesion: 0.18
Nodes (8): ApplyStagedRestore(), DB, Open(), TestApplyStagedRestore(), TestApplyStagedRestoreNoOp(), TestValidateBackupRejectsAForeignDatabaseWithoutTouchingIt(), ValidateBackup(), Tx

### Community 64 - "users_cli.go"
Cohesion: 0.43
Nodes (16): deletionHeir(), firstArg(), DB, User, Writer, guardLastActiveAdmin(), lookup(), normalizeEmail() (+8 more)

### Community 65 - "app"
Cohesion: 0.28
Nodes (6): Every reveal is audited, Request, ResponseWriter, app, User, adminUserView

### Community 66 - "testServer"
Cohesion: 0.28
Nodes (15): app, Server, getSettings(), Client, Response, T, putSettings(), TestSealedSettingRoundTrip() (+7 more)

### Community 67 - "DB"
Cohesion: 0.18
Nodes (5): DB, Time, scanCredentialMeta(), AccessGrant, Credential

### Community 68 - "dependencies"
Cohesion: 0.13
Nodes (15): @fontsource-variable/inter, leaflet, react, react-dom, react-router-dom, uplot, dependencies, @fontsource-variable/inter (+7 more)

### Community 69 - "app"
Cohesion: 0.22
Nodes (8): Writes must come from this origin, dynamicRequest(), Handler, Request, app, User, User, PrincipalFromUser()

### Community 70 - "dbWithUser"
Cohesion: 0.33
Nodes (11): Stable slug and /go redirect, Slugify(), dbWithUser(), DB, T, TestCreateToolFallsBackWhenTheNameHasNoSlug(), TestGetToolBySlug(), TestSlugify() (+3 more)

### Community 71 - ".handleSetHostAutoUpdate"
Cohesion: 0.29
Nodes (6): A stalled host pauses the whole rollout, Request, ResponseWriter, app, StalledHost, agentUpdateRollup

### Community 72 - ".handleCreateCommand"
Cohesion: 0.29
Nodes (6): HostCommand, Request, ResponseWriter, app, commandInput, commandView

### Community 73 - ".loadEndpoint"
Cohesion: 0.27
Nodes (9): 404 instead of 403 for invisible records, Follow-the-host endpoint resolution, Host, Request, ResponseWriter, app, Tool, resolveEndpoint() (+1 more)

### Community 74 - ".handleCreateCredential"
Cohesion: 0.36
Nodes (5): Credential, Host, Request, ResponseWriter, app

### Community 75 - "NewID"
Cohesion: 0.23
Nodes (7): auditWhere(), DB, Time, NewID(), AuditFilter, GrantAudit, RevealAudit

### Community 76 - "shutdown_test.go"
Cohesion: 0.33
Nodes (8): Conn, T, TestEveryStopsOnCancel(), TestProbeHealth(), TestProbeHealthFailsOnNon200(), TestServeDrainsInFlightRequest(), TestServeReturnsListenError(), waitForListener()

### Community 77 - "test_build_config.py"
Cohesion: 0.23
Nodes (8): parametrize, dry_run(), job_commands(), test_actions_are_pinned_to_a_commit(), test_npm_target_installs_its_own_deps(), test_signing_key_is_never_workflow_or_job_scoped(), test_workflow_jobs_install_web_deps_before_running_npm_scripts(), test_workflows_grant_no_blanket_permissions()

### Community 78 - "adminResetPassword"
Cohesion: 0.38
Nodes (9): adminResetPassword(), Client, Response, T, TestAdminResetPasswordRejects(), TestAdminResetPasswordSetsItAndSignsTargetOut(), TestChangePasswordDropsOtherSessions(), TestEveryPasswordPathEnforcesTheMinimum() (+1 more)

### Community 79 - "downloadBackup"
Cohesion: 0.41
Nodes (12): downloadBackup(), Client, Response, T, TestBackupAndRestoreAreAdminOnly(), TestBackupDownloadIsAUsableDatabase(), TestBackupKeepsCredentialsEncrypted(), TestRestoreCancelDiscardsStaged() (+4 more)

### Community 80 - "lookupMDNS"
Cohesion: 0.32
Nodes (11): Context, lookupMDNS(), mdnsAnswer(), mdnsQuery(), T, response(), TestMDNSAnswerIgnoresOtherHosts(), TestMDNSAnswerPicksTheRequestedName() (+3 more)

### Community 81 - "compressWriter"
Cohesion: 0.31
Nodes (4): compressible(), ResponseWriter, Writer, compressWriter

### Community 82 - "newPusher"
Cohesion: 0.18
Nodes (16): Client, config, newPusher(), permanentReject(), randSuffix(), T, seedBuffer(), TestFlushBufferDropsAPermanentlyRejectedBody() (+8 more)

### Community 83 - "app"
Cohesion: 0.44
Nodes (4): File, Request, ResponseWriter, app

### Community 84 - "newDownloadApp"
Cohesion: 0.36
Nodes (11): MapFS, app, T, newDownloadApp(), TestRoutesDownloadDispatch(), TestServeAgentBinaryKnownArch(), TestServeAgentBinaryUnknownArch404(), TestServeAgentChecksumMatchesBytes() (+3 more)

### Community 85 - ".handleCreateWebhook"
Cohesion: 0.32
Nodes (5): channelView, Request, ResponseWriter, app, Webhook

### Community 86 - "decodeJSON"
Cohesion: 0.21
Nodes (9): decodeJSON(), Request, ResponseWriter, app, Request, ResponseWriter, app, thresholdDTO (+1 more)

### Community 87 - "DB"
Cohesion: 0.24
Nodes (5): blockingSlot(), DB, Time, ValidAutoUpdatePolicy(), StalledHost

### Community 88 - "DB"
Cohesion: 0.20
Nodes (3): DB, Time, Group

### Community 89 - "TestPublicHostsNoAuth"
Cohesion: 0.53
Nodes (5): bearer(), T, TestAPIErrorEnvelope(), TestPublicEndpointsNoAuthExcludeRestricted(), TestPublicHostsNoAuth()

### Community 92 - "rbac.go"
Cohesion: 0.27
Nodes (10): Session cookie auth, ctxKey, Single-tenant, admin sees everything, Context, Handler, ResponseWriter, RequireAdmin(), RequireAuth() (+2 more)

### Community 93 - "writeJSON"
Cohesion: 0.14
Nodes (13): One Go binary serves UI and API, ResponseWriter, writeJSON(), Request, ResponseWriter, Request, ResponseWriter, app (+5 more)

### Community 94 - "AccessRequest"
Cohesion: 0.35
Nodes (4): DB, Time, scanRequest(), AccessRequest

### Community 95 - "processes_test.go"
Cohesion: 0.33
Nodes (10): T, TestDeleteHostMetricsDropsProcesses(), TestDeleteHostMetricsDropsProcessUsage(), TestHostProcessesNilStoresEmptyList(), TestHostProcessesRoundTripAndOverwrite(), TestLatestHostProcessesUnknownHost(), TestProcessUsageAveragesAndPeaks(), TestProcessUsageKeepsTopByMemory() (+2 more)

### Community 96 - "DB"
Cohesion: 0.36
Nodes (4): Duration, DB, Time, Retention

### Community 97 - ".checkSchemaDrift"
Cohesion: 0.25
Nodes (6): declaredColumns(), DB, T, TestDeclaredColumnsReadsTheSchema(), TestOpenAcceptsACurrentDatabase(), TestOpenRefusesADatabaseMissingAColumn()

### Community 98 - "useAuth"
Cohesion: 0.17
Nodes (12): User, App(), Protected(), AuthContext, AuthProvider(), AuthState, AuthStatus, useAuth() (+4 more)

### Community 99 - ".sendMail"
Cohesion: 0.29
Nodes (4): Config, app, smtpInput, smtpView

### Community 101 - "mustUser"
Cohesion: 0.50
Nodes (8): DB, T, mustUser(), TestCollectionCRUD(), TestCollectionEditRights(), TestCollectionMembership(), TestCollectionVisibility(), TestDeleteGroupClearsCollectionGrants()

### Community 103 - "RevealModal.tsx"
Cohesion: 0.38
Nodes (7): RevealedCredential, RevealModal(), SecretField(), isSensitiveField(), ORDER, orderedSecretFields(), SENSITIVE

### Community 104 - "TestHostEventsEndpoint"
Cohesion: 0.36
Nodes (7): AlertEvent, T, Time, nowUTC(), storeEvent(), TestHostEventsEndpoint(), TestToolEventsHiddenForOutsider()

### Community 105 - "live_ssh_install_test.sh"
Cohesion: 0.43
Nodes (6): api(), bad(), cleanup_host(), ok(), remote(), live_ssh_install_test.sh script

### Community 106 - ".uiHandler"
Cohesion: 0.29
Nodes (5): agentDistFS(), assetCacheControl(), FS, Handler, app

### Community 108 - "scripts"
Cohesion: 0.25
Nodes (8): scripts, build, dev, e2e, lint, preview, test, typecheck

### Community 111 - "effectiveThreshold"
Cohesion: 0.62
Nodes (6): effectiveThreshold(), T, TestHostThresholdsAdminGetSet(), TestHostThresholdsReset(), TestHostThresholdsUnknownHost(), TestThresholdsAdminGetSet()

### Community 112 - "contracts_test.go"
Cohesion: 0.53
Nodes (5): T, TestPushAckRoundTrip(), TestPushCarriesAutoUpdateVeto(), TestPushRoundTrip(), TestPushTooLargeNamesTheOffendingSection()

### Community 114 - "webhooks_test.go"
Cohesion: 0.53
Nodes (5): T, TestChannelAcceptsSeverity(), TestChannelColumnsRoundTrip(), TestChannelDefaults(), TestGlobalChannelsFilterBySeverity()

### Community 115 - "package.json"
Cohesion: 0.33
Nodes (5): license, name, private, type, version

### Community 118 - "render"
Cohesion: 0.67
Nodes (3): Image, main(), render()

### Community 119 - "TestContainerStatsNameFallback"
Cohesion: 0.67
Nodes (3): T, TestContainerStatsNameFallback(), TestContainerStatsRoundTrip()

### Community 120 - "principal_handlers.go"
Cohesion: 0.83
Nodes (3): principalGroup, principalsView, principalUser

### Community 163 - "postWebhook"
Cohesion: 0.29
Nodes (9): postWebhook(), redactConfig(), T, TestDispatchFailsChannelWithNoURL(), TestDispatchMarksSentAndFailed(), TestDueDeliveriesReturnsKindConfig(), TestPostWebhookBearerToken(), TestPostWebhookSendsPayload() (+1 more)

### Community 164 - ".handleRestoreUpload"
Cohesion: 0.29
Nodes (6): ReadCloser, Request, ResponseWriter, app, StagedRestorePath(), formFile()

## Ambiguous Edges - Review These
- `LAN-only, no external backend` → `DCO sign-off, no CLA`  [AMBIGUOUS]
  CONTRIBUTING.md · relation: conceptually_related_to
- `Light-only theme` → `Blocking theme-boot script`  [AMBIGUOUS]
  DESIGN.md · relation: conceptually_related_to

## Knowledge Gaps
- **149 isolated node(s):** `dockerPSLine`, `dockerStatsLine`, `config`, `DiskUsage`, `ProcessSample` (+144 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **26 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **What is the exact relationship between `LAN-only, no external backend` and `DCO sign-off, no CLA`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **What is the exact relationship between `Light-only theme` and `Blocking theme-boot script`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **Why does `New()` connect `New` to `publishAgent`, `signing.go`, `Install`, `control_test.go`, `newCLIEnv`, `google_test.go`, `google_auth_test.go`, `postWebhook`, `.handleRestoreUpload`, `Send`, `.handlePutSettings`, `Push`, `T`, `Open`, `users_cli.go`, `app`, `newPusher`, `app`, `decodeJSON`, `.sendMail`, `.dispatchDue`?**
  _High betweenness centrality (0.285) - this node is a cross-community bridge._
- **Why does `Public portal front door` connect `.handleIngest` to `Portal.tsx`?**
  _High betweenness centrality (0.269) - this node is a cross-community bridge._
- **Are the 215 inferred relationships involving `newTestServer()` (e.g. with `TestAuditListsAndFilters()` and `TestServerInfoAdminOnly()`) actually correct?**
  _`newTestServer()` has 215 INFERRED edges - model-reasoned connections that need verification._
- **Are the 108 inferred relationships involving `signup()` (e.g. with `TestAuditListsAndFilters()` and `TestServerInfoAdminOnly()`) actually correct?**
  _`signup()` has 108 INFERRED edges - model-reasoned connections that need verification._
- **Are the 116 inferred relationships involving `writeError()` (e.g. with `validPrincipalType()` and `.applyCollectionIDs()`) actually correct?**
  _`writeError()` has 116 INFERRED edges - model-reasoned connections that need verification._