# Graph Report - Reeve  (2026-07-28)

## Corpus Check
- 304 files · ~191,323 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 2762 nodes · 7076 edges · 169 communities (142 shown, 27 thin omitted)
- Extraction: 81% EXTRACTED · 19% INFERRED · 0% AMBIGUOUS · INFERRED: 1333 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `795ee8c1`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- publishAgent
- release.py
- secretFields.ts
- signing.go
- Install
- CollectionDetail.tsx
- app
- HostMetrics.tsx
- Hosts.tsx
- signup
- google_test.go
- adminClient
- Catalog.tsx
- app
- ui.tsx
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
- testServer
- newUser
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
- decodeJSON
- control_test.go
- DB
- HostInventory.tsx
- zzz-visual.spec.ts
- collect_test.go
- .handlePutSettings
- writeError
- DB
- DB
- api.ts
- dbWithUser
- DB
- crypto_test.go
- devDependencies
- HandlerFunc
- shutdown_test.go
- T
- Open
- users_cli.go
- app
- putSettings
- DB
- dependencies
- app
- .handleIngest
- contracts.go
- .handleCreateCommand
- .loadEndpoint
- .handleCreateCredential
- NewID
- newCLIEnv
- test_build_config.py
- ScanLogErrors
- downloadBackup
- lookupMDNS
- .handleSetHostAutoUpdate
- newPusher
- app
- newDownloadApp
- .handleCreateWebhook
- .handleGetHostThresholds
- DB
- DB
- TestPublicHostsNoAuth
- .handleCreateGroup
- .handleGeocode
- rbac.go
- writeJSON
- AccessRequest
- processes_test.go
- DB
- .checkSchemaDrift
- MapPicker.tsx
- .sendMail
- App.tsx
- mustUser
- zz-agent-updates.spec.ts
- zzzz-location-autosave.spec.ts
- TestHostEventsEndpoint
- live_ssh_install_test.sh
- .uiHandler
- ssh_install_handlers_test.go
- scripts
- @playwright/test
- effectiveThreshold
- theme/tailwind.config.js
- contracts_test.go
- .dispatchDue
- webhooks_test.go
- package.json
- DB
- render
- tailwindcss
- principal_handlers.go
- TestHostNetRate
- zzz-button-alignment.spec.ts
- agent/ must not import server/
- uninstall.sh
- Every column has a name
- mustVerify
- confirmText.ts
- group_handlers.go
- tool_visibility_handlers.go
- postWebhook
- @types/react
- adminResetPassword
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
- Push
- @types/react-dom
- typescript
- postcss
- TestContainerStatsNameFallback
- compressWriter
- setPassword

## God Nodes (most connected - your core abstractions)
1. `newTestServer()` - 231 edges
2. `signup()` - 131 edges
3. `writeError()` - 123 edges
4. `adminClient()` - 88 edges
5. `openTemp()` - 80 edges
6. `writeJSON()` - 71 edges
7. `New()` - 53 edges
8. `FromContext()` - 51 edges
9. `testServer` - 43 edges
10. `createTool()` - 41 edges

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

## Communities (169 total, 27 thin omitted)

### Community 0 - "publishAgent"
Cohesion: 0.08
Nodes (50): Forced update sits outside the rollout, The host's local veto is absolute, effectiveAutoUpdate(), Duration, Host, app, Time, fetchRollup() (+42 more)

### Community 1 - "release.py"
Cohesion: 0.05
Nodes (51): CI gate job, CI grants contents:read only, Actions pinned by commit, not tag, Rolling edge prerelease, Signing key scoped to steps, not the workflow, Tag-triggered versioned release, web-build and server-assets precede build, make gate / web-check / pytest before commit (+43 more)

### Community 2 - "secretFields.ts"
Cohesion: 0.39
Nodes (6): RevealModal(), SecretField(), isSensitiveField(), ORDER, orderedSecretFields(), SENSITIVE

### Community 3 - "signing.go"
Cohesion: 0.08
Nodes (56): releasePublicKey(), fetchBody(), config, Client, isSHA256Hex(), needsUpdate(), olderThan(), selfArch() (+48 more)

### Community 4 - "Install"
Cohesion: 0.10
Nodes (45): Builder, Channel, ClientConfig, clientConfig(), connect(), dial(), dialTCP(), envAssignments() (+37 more)

### Community 5 - "CollectionDetail.tsx"
Cohesion: 0.14
Nodes (14): HostInventory, InventoryItem, AddServiceModal(), Chevron(), DiskList(), DiskUsage, sortDisks(), usedPct() (+6 more)

### Community 6 - "app"
Cohesion: 0.28
Nodes (11): alwaysEditable(), Request, ResponseWriter, app, hostCanSee(), hostTarget(), toolCanEdit(), toolCanSee() (+3 more)

### Community 7 - "HostMetrics.tsx"
Cohesion: 0.10
Nodes (32): Colorblind-validated chart palette, Chart(), Series, ContainerPoint, fmtPct(), HostMetrics(), HostPoint, labelFor() (+24 more)

### Community 8 - "Hosts.tsx"
Cohesion: 0.12
Nodes (13): AgentUpdateRollup, UpdateState, Tabs(), showsVersionPill(), hostRowActions(), RowActions, HOST_FILTERS, HostFilter (+5 more)

### Community 9 - "signup"
Cohesion: 0.13
Nodes (36): T, TestServerInfoAdminOnly(), TestUpdateUserRoleAndActive(), fromIP(), Client, Response, T, login() (+28 more)

### Community 10 - "google_test.go"
Cohesion: 0.09
Nodes (27): Config, Profile, Config, Request, ResponseWriter, app, User, DomainAllowed() (+19 more)

### Community 11 - "adminClient"
Cohesion: 0.15
Nodes (42): TestAuditListsAndFilters(), createCred(), Client, T, newHost(), revealSecret(), TestAccessRequestOnAnUnknownHost(), TestAdminCanRevealAndAudited() (+34 more)

### Community 12 - "Catalog.tsx"
Cohesion: 0.17
Nodes (12): GrantAuditEntry, RevealAuditEntry, EmptyState(), Pill(), matchesQuery(), AdminAudit(), AdminGroups(), AdminWebhooks() (+4 more)

### Community 13 - "app"
Cohesion: 0.09
Nodes (31): clock, Throttle, throttleEntry, Int64, Once, Login throttle that cannot be held shut, REEVE_TRUST_PROXY is opt-in, DB (+23 more)

### Community 14 - "ui.tsx"
Cohesion: 0.10
Nodes (33): Enter confirms every form, api, Credential, CREDENTIAL_LABEL, Group, RevealedCredential, SSHTarget, uploadAvatar() (+25 more)

### Community 15 - "Dashboard.tsx"
Cohesion: 0.12
Nodes (22): Say what the state means, AccessRequest, AlertEvent, ToolStatus, MetricBar(), ServiceDots(), ListSkeleton(), Skeleton() (+14 more)

### Community 16 - "Portal.tsx"
Cohesion: 0.08
Nodes (31): Test every trigger of a modal, CollectionRef, endpointString(), goURL(), Host, Tool, CollectionInfoModal(), HostInfoModal() (+23 more)

### Community 17 - "schema.sql"
Cohesion: 0.09
Nodes (36): One schema file, re-executed on open, No protocol version to negotiate, Everything but secrets is plaintext, access_requests, alert_events, alert_state, alert_thresholds, collection_editors (+28 more)

### Community 18 - "run"
Cohesion: 0.23
Nodes (11): envOr(), every(), Context, Duration, Server, app, main(), probeHealth() (+3 more)

### Community 19 - "Principal"
Cohesion: 0.17
Nodes (18): Principal, CollectionRef, assetURL(), agentMonitored(), checkEndpointFields(), Host, Request, ResponseWriter (+10 more)

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
Cohesion: 0.17
Nodes (29): Client, T, noRedirect(), pushIP(), TestEmptyReportedAddressKeepsTheLastKnownOne(), TestEndpointJSON(), TestGoHidesToolsTheCallerCannotSee(), TestGoRedirectFollowsTheHostAddress() (+21 more)

### Community 24 - "hostToView"
Cohesion: 0.19
Nodes (13): Host, Request, ResponseWriter, app, Time, hostStatus(), hostToView(), hostMetricsView (+5 more)

### Community 25 - "newTestServer"
Cohesion: 0.18
Nodes (23): T, TestGeocodeProxiesAndLabels(), TestGeocodeRequiresAdmin(), TestGeocodeShortQuerySkipsUpstream(), TestGeocodeUpstreamFailureIsBadGateway(), newTestServer(), T, TestHostMetricsEndpointCarriesDisks() (+15 more)

### Community 26 - "procs_test.go"
Cohesion: 0.17
Nodes (24): ProcessSample, Time, NewProcSampler(), ParseCmdline(), ParsePasswd(), ParseProcStat(), ParseProcStatus(), T (+16 more)

### Community 27 - "gather"
Cohesion: 0.19
Nodes (21): durationArg(), gather(), gatherCron(), gatherDockerLogErrors(), gatherLogErrors(), config, Duration, Time (+13 more)

### Community 28 - "DB"
Cohesion: 0.07
Nodes (21): fmtRate(), Duration, Host, app, Time, Webhook, hostMetricValues(), severityOrError() (+13 more)

### Community 29 - "testServer"
Cohesion: 0.15
Nodes (34): TestUpdateNowRefusesAVetoedHost(), controllableHost(), Client, T, queue(), TestAResultFromTheWrongHostIsIgnored(), TestCommandsAreAdminOnly(), TestQueueAndDeliverACommand() (+26 more)

### Community 30 - "newUser"
Cohesion: 0.17
Nodes (24): containsKey(), containsName(), Client, T, hasKey(), hasName(), jsonString(), newUser() (+16 more)

### Community 31 - "app"
Cohesion: 0.18
Nodes (10): AccessRequest, Request, ResponseWriter, app, validPrincipalType(), accessGrantView, requestView, Request (+2 more)

### Community 32 - "Layout.tsx"
Cohesion: 0.08
Nodes (22): UptimeSummary, Avatar(), initials(), adminNav, Layout(), NavRecord, primaryNav, IconName (+14 more)

### Community 33 - "T"
Cohesion: 0.26
Nodes (19): configureRelay(), doReset(), forgot(), Client, Response, T, newRelay(), resetTokenFrom() (+11 more)

### Community 34 - "google_auth_test.go"
Cohesion: 0.40
Nodes (23): assertLoginError(), callback(), enableGoogle(), Client, Response, Server, T, noFollow() (+15 more)

### Community 35 - "openTemp"
Cohesion: 0.20
Nodes (26): ApplyStagedRestore(), DB, T, openTemp(), TestApplyStagedRestore(), TestApplyStagedRestoreNoOp(), TestBackupTo(), TestBackupToRefusesExistingDest() (+18 more)

### Community 36 - "profile_handlers_test.go"
Cohesion: 0.30
Nodes (13): Client, Response, T, pngBytes(), readBody(), TestAdminDeleteUser(), TestAvatarRejectsNonImageAndOversize(), TestAvatarUploadServeDelete() (+5 more)

### Community 37 - "store/agent_update_test.go"
Cohesion: 0.20
Nodes (22): DB, T, Time, pacedSlot(), TestApplyPushRecordsVeto(), TestApplyPushWithAVetoReleasesTheSlot(), TestClearStalledUpdatesSweepsIneligibleHostsToo(), TestFleetDefaultOffMakesADefaultPolicyHostIneligible() (+14 more)

### Community 38 - "seedHost"
Cohesion: 0.18
Nodes (21): T, TestApplyPushStoresDisks(), TestDeleteHostMetricsDropsDisks(), TestHostDisksNilStoresEmptyList(), TestHostDisksRoundTripAndReplace(), TestLatestHostDisksUnknownHost(), T, TestApplyPushKeepsLastKnownChecksum() (+13 more)

### Community 39 - "New"
Cohesion: 0.29
Nodes (8): AEAD, Cipher, Compose refuses to start without the master key, Backup carries ciphertext, not the key, Credentials encrypted before they touch disk, Master-key custody, New(), NewFromEnv()

### Community 40 - "runOnce"
Cohesion: 0.22
Nodes (19): commandReport(), config, Duration, Time, loadConfig(), main(), runOnce(), runSelfUpdate() (+11 more)

### Community 41 - "compilerOptions"
Cohesion: 0.09
Nodes (21): DOM, DOM.Iterable, ES2021, src, compilerOptions, allowImportingTsExtensions, isolatedModules, jsx (+13 more)

### Community 42 - ".ApplyPush"
Cohesion: 0.06
Nodes (34): ContainerSample, ContainerState, DiskUsage, DB, Time, replaceHostDisks(), HostMetrics, DB (+26 more)

### Community 43 - "Send"
Cohesion: 0.21
Nodes (18): Config, Message, stubSMTP, BuildMessage(), Time, hasCRLF(), Send(), atoi() (+10 more)

### Community 44 - "countRows"
Cohesion: 0.18
Nodes (22): countRows(), enrollHostWin(), Client, T, TestAgentOfflineAlert(), TestDispatchSendsAndRetries(), TestDownAlertDebounceFireResolve(), TestFlapSuppressed() (+14 more)

### Community 45 - "decodeJSON"
Cohesion: 0.11
Nodes (16): errNoAgentBuild, errUnsupportedArch, Request, ResponseWriter, app, hashResetToken(), newResetToken(), reachableFromOtherHosts() (+8 more)

### Community 46 - "control_test.go"
Cohesion: 0.29
Nodes (15): argvFor(), newController(), fakeRunner(), T, TestArgvForEveryAction(), TestArgvForRefusesADangerousTarget(), TestArgvForRejectsAnUnknownAction(), TestControlDefaultsOn() (+7 more)

### Community 47 - "DB"
Cohesion: 0.20
Nodes (7): channelAccepts(), filterBySeverity(), DB, Time, severityRank(), Delivery, Webhook

### Community 48 - "HostInventory.tsx"
Cohesion: 0.10
Nodes (29): AutoUpdatePolicy, Delivery, ThresholdMetric, ThresholdsPayload, HostControls(), unavailableReason(), EMPTY_ROW, METRIC_KEYS (+21 more)

### Community 49 - "zzz-visual.spec.ts"
Cohesion: 0.09
Nodes (9): CI Playwright e2e job, Frontend changes are reviewed in a browser, axe against WCAG 2 A/AA, Manual screenshot sweep after any frontend change, Pixel baselines for every page, TAGS, THEMES, GREY_TILE (+1 more)

### Community 50 - "collect_test.go"
Cohesion: 0.06
Nodes (50): T, TestParseCPUStatAndPercent(), TestParseCrontabSystemForm(), TestParseCrontabUserForm(), TestParseDockerPS(), TestParseDockerStats(), TestParseLoadAvg(), TestParseMemInfo() (+42 more)

### Community 51 - ".handlePutSettings"
Cohesion: 0.15
Nodes (14): Agent update state machine, Checksum decides currency, not version, Retention, agentUpdateView, googleAuthView, googleInput, retentionView, Request (+6 more)

### Community 52 - "writeError"
Cohesion: 0.15
Nodes (14): ReadCloser, Request, ResponseWriter, app, writeError(), Request, ResponseWriter, readImageUpload() (+6 more)

### Community 53 - "DB"
Cohesion: 0.21
Nodes (8): Rows, DB, Time, scanAlertEvents(), Time, parseNullableTime(), AlertEvent, AlertState

### Community 54 - "DB"
Cohesion: 0.15
Nodes (3): DB, Time, Host

### Community 55 - "api.ts"
Cohesion: 0.07
Nodes (29): AgentUpdateSettings, ChannelKind, Collection, CollectionDetail, CommandStatus, CREDENTIAL_FIELDS, CredentialType, GoogleInput (+21 more)

### Community 56 - "dbWithUser"
Cohesion: 0.33
Nodes (11): Stable slug and /go redirect, Slugify(), dbWithUser(), DB, T, TestCreateToolFallsBackWhenTheNameHasNoSlug(), TestGetToolBySlug(), TestSlugify() (+3 more)

### Community 57 - "DB"
Cohesion: 0.23
Nodes (5): NullString, DB, Time, parseNullTime(), HostCommand

### Community 58 - "crypto_test.go"
Cohesion: 0.45
Nodes (11): T, mustKey(), testKey(), TestNewFromEnvMissingKeyFails(), TestNewFromEnvValid(), TestNewFromEnvWrongLengthFails(), TestOpenRejectsAnotherRowsAAD(), TestSealOpenRoundTrip() (+3 more)

### Community 59 - "devDependencies"
Cohesion: 0.12
Nodes (17): autoprefixer, @axe-core/playwright, eslint, eslint-plugin-react-hooks, @types/leaflet, @typescript-eslint/eslint-plugin, @typescript-eslint/parser, vitest (+9 more)

### Community 60 - "HandlerFunc"
Cohesion: 0.26
Nodes (15): HandlerFunc, Handler, Handler, gzipResponses(), Handler, Response, T, gzipRequest() (+7 more)

### Community 61 - "shutdown_test.go"
Cohesion: 0.33
Nodes (8): Conn, T, TestEveryStopsOnCancel(), TestProbeHealth(), TestProbeHealthFailsOnNon200(), TestServeDrainsInFlightRequest(), TestServeReturnsListenError(), waitForListener()

### Community 62 - "T"
Cohesion: 0.23
Nodes (14): captureLog(), Buffer, Client, Response, T, newTestServerKey(), TestClientIPForwardedForOnlyWhenProxyTrusted(), TestHealthz() (+6 more)

### Community 63 - "Open"
Cohesion: 0.22
Nodes (5): DB, Open(), TestValidateBackupRejectsAForeignDatabaseWithoutTouchingIt(), ValidateBackup(), Tx

### Community 64 - "users_cli.go"
Cohesion: 0.43
Nodes (16): deletionHeir(), firstArg(), DB, User, Writer, guardLastActiveAdmin(), lookup(), normalizeEmail() (+8 more)

### Community 65 - "app"
Cohesion: 0.27
Nodes (6): Every reveal is audited, Request, ResponseWriter, app, User, adminUserView

### Community 66 - "putSettings"
Cohesion: 0.37
Nodes (12): getSettings(), Client, Response, T, putSettings(), TestSealedSettingRoundTrip(), TestSettingsAdminOnly(), TestSettingsDefaults() (+4 more)

### Community 67 - "DB"
Cohesion: 0.18
Nodes (5): DB, Time, scanCredentialMeta(), AccessGrant, Credential

### Community 68 - "dependencies"
Cohesion: 0.13
Nodes (15): @fontsource-variable/inter, leaflet, react, react-dom, react-router-dom, uplot, dependencies, @fontsource-variable/inter (+7 more)

### Community 69 - "app"
Cohesion: 0.22
Nodes (8): Writes must come from this origin, dynamicRequest(), Handler, Request, app, User, User, PrincipalFromUser()

### Community 70 - ".handleIngest"
Cohesion: 0.12
Nodes (15): Host enrollment token, DCO sign-off, no CLA, Binds every interface by default, Docker NAT sits in front of ufw, Host networking for mDNS, SSH push install, Catalog of services, Host and service controls (+7 more)

### Community 71 - "contracts.go"
Cohesion: 0.11
Nodes (23): Mutex, controller, Fixed action allowlist, no shell, Collections replace free-text category, Single error envelope, Ingest push contract, Commands ride the push ack, CollectionRef (+15 more)

### Community 72 - ".handleCreateCommand"
Cohesion: 0.29
Nodes (6): HostCommand, Request, ResponseWriter, app, commandInput, commandView

### Community 73 - ".loadEndpoint"
Cohesion: 0.27
Nodes (9): 404 instead of 403 for invisible records, Follow-the-host endpoint resolution, Host, Request, ResponseWriter, app, Tool, resolveEndpoint() (+1 more)

### Community 74 - ".handleCreateCredential"
Cohesion: 0.33
Nodes (6): Credential, credentialAAD(), Host, Request, ResponseWriter, app

### Community 75 - "NewID"
Cohesion: 0.19
Nodes (9): auditLink(), auditWhere(), DB, Time, NewID(), AuditChainResult, AuditFilter, GrantAudit (+1 more)

### Community 76 - "newCLIEnv"
Cohesion: 0.12
Nodes (30): server users CLI, Request, ResponseWriter, app, User, cliEnv, HashPassword(), HashToken() (+22 more)

### Community 77 - "test_build_config.py"
Cohesion: 0.23
Nodes (8): parametrize, dry_run(), job_commands(), test_actions_are_pinned_to_a_commit(), test_npm_target_installs_its_own_deps(), test_signing_key_is_never_workflow_or_job_scoped(), test_workflow_jobs_install_web_deps_before_running_npm_scripts(), test_workflows_grant_no_blanket_permissions()

### Community 78 - "ScanLogErrors"
Cohesion: 0.16
Nodes (16): TestScanLogErrors(), isEnvAssignment(), ParseCrontab(), Time, ScanLogErrors(), basename(), maskInline(), namesSecret() (+8 more)

### Community 79 - "downloadBackup"
Cohesion: 0.41
Nodes (12): downloadBackup(), Client, Response, T, TestBackupAndRestoreAreAdminOnly(), TestBackupDownloadIsAUsableDatabase(), TestBackupKeepsCredentialsEncrypted(), TestRestoreCancelDiscardsStaged() (+4 more)

### Community 80 - "lookupMDNS"
Cohesion: 0.32
Nodes (11): Context, lookupMDNS(), mdnsAnswer(), mdnsQuery(), T, response(), TestMDNSAnswerIgnoresOtherHosts(), TestMDNSAnswerPicksTheRequestedName() (+3 more)

### Community 81 - ".handleSetHostAutoUpdate"
Cohesion: 0.29
Nodes (6): A stalled host pauses the whole rollout, Request, ResponseWriter, app, StalledHost, agentUpdateRollup

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

### Community 86 - ".handleGetHostThresholds"
Cohesion: 0.33
Nodes (5): Request, ResponseWriter, app, thresholdDTO, thresholdsPayload

### Community 87 - "DB"
Cohesion: 0.24
Nodes (5): blockingSlot(), DB, Time, ValidAutoUpdatePolicy(), StalledHost

### Community 88 - "DB"
Cohesion: 0.20
Nodes (3): DB, Time, Group

### Community 89 - "TestPublicHostsNoAuth"
Cohesion: 0.53
Nodes (5): bearer(), T, TestAPIErrorEnvelope(), TestPublicEndpointsNoAuthExcludeRestricted(), TestPublicHostsNoAuth()

### Community 90 - ".handleCreateGroup"
Cohesion: 0.50
Nodes (3): Request, ResponseWriter, app

### Community 91 - ".handleGeocode"
Cohesion: 0.31
Nodes (8): Request, ResponseWriter, app, photonHits(), photonLabel(), geocodeHit, photonFeature, photonResponse

### Community 92 - "rbac.go"
Cohesion: 0.27
Nodes (10): Session cookie auth, ctxKey, Single-tenant, admin sees everything, Context, Handler, ResponseWriter, RequireAdmin(), RequireAuth() (+2 more)

### Community 93 - "writeJSON"
Cohesion: 0.14
Nodes (14): One Go binary serves UI and API, ResponseWriter, writeJSON(), Request, ResponseWriter, Request, ResponseWriter, app (+6 more)

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

### Community 98 - "MapPicker.tsx"
Cohesion: 0.17
Nodes (18): LeafletMap(), MapPoint, tileURL(), MapPicker(), Saved, cities, City, cityMatches() (+10 more)

### Community 99 - ".sendMail"
Cohesion: 0.29
Nodes (4): Config, app, smtpInput, smtpView

### Community 100 - "App.tsx"
Cohesion: 0.09
Nodes (24): AdminUser, ApiError, uploadIcon(), User, App(), Protected(), AuthContext, AuthProvider() (+16 more)

### Community 101 - "mustUser"
Cohesion: 0.50
Nodes (8): DB, T, mustUser(), TestCollectionCRUD(), TestCollectionEditRights(), TestCollectionMembership(), TestCollectionVisibility(), TestDeleteGroupClearsCollectionGrants()

### Community 104 - "TestHostEventsEndpoint"
Cohesion: 0.36
Nodes (7): AlertEvent, T, Time, nowUTC(), storeEvent(), TestHostEventsEndpoint(), TestToolEventsHiddenForOutsider()

### Community 105 - "live_ssh_install_test.sh"
Cohesion: 0.43
Nodes (6): api(), bad(), cleanup_host(), ok(), remote(), live_ssh_install_test.sh script

### Community 106 - ".uiHandler"
Cohesion: 0.29
Nodes (5): agentDistFS(), assetCacheControl(), FS, Handler, app

### Community 107 - "ssh_install_handlers_test.go"
Cohesion: 0.39
Nodes (11): createHostFor(), Client, T, sshBody(), TestAgentBinaryForUname(), TestReachableFromOtherHosts(), TestSSHEndpointsAreAdminOnlyAndScopedToAHost(), TestSSHInstallRejectsIncompleteTargets() (+3 more)

### Community 108 - "scripts"
Cohesion: 0.25
Nodes (8): scripts, build, dev, e2e, lint, preview, test, typecheck

### Community 110 - "effectiveThreshold"
Cohesion: 0.62
Nodes (6): effectiveThreshold(), T, TestHostThresholdsAdminGetSet(), TestHostThresholdsReset(), TestHostThresholdsUnknownHost(), TestThresholdsAdminGetSet()

### Community 111 - "theme/tailwind.config.js"
Cohesion: 0.33
Nodes (4): Light-only theme, Four-radius vocabulary, Two colors do two jobs, Blocking theme-boot script

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

### Community 120 - "principal_handlers.go"
Cohesion: 0.83
Nodes (3): principalGroup, principalsView, principalUser

### Community 129 - "mustVerify"
Cohesion: 0.45
Nodes (10): chainedDB(), DB, T, mustVerify(), TestAuditChainCatchesADeletedRow(), TestAuditChainCatchesAnEditedRow(), TestAuditChainVerifiesWhenUntouched(), TestAuditChainWithoutAKeyReportsItCannotCheck() (+2 more)

### Community 130 - "confirmText.ts"
Cohesion: 0.48
Nodes (5): Confirmation, NEEDS_CONFIRM, needsConfirm(), rowConfirmation(), RowControls()

### Community 133 - "postWebhook"
Cohesion: 0.29
Nodes (9): postWebhook(), redactConfig(), T, TestDispatchFailsChannelWithNoURL(), TestDispatchMarksSentAndFailed(), TestDueDeliveriesReturnsKindConfig(), TestPostWebhookBearerToken(), TestPostWebhookSendsPayload() (+1 more)

### Community 135 - "adminResetPassword"
Cohesion: 0.35
Nodes (10): adminResetPassword(), Client, Response, T, TestAdminResetPasswordRejects(), TestAdminResetPasswordSetsItAndSignsTargetOut(), TestChangePasswordDropsOtherSessions(), TestEveryPasswordPathEnforcesTheMinimum() (+2 more)

### Community 162 - "Push"
Cohesion: 0.24
Nodes (8): HostMetrics, ProcessSample, Time, CronState, Push, T, realisticPush(), TestLoadIngest()

### Community 166 - "TestContainerStatsNameFallback"
Cohesion: 0.67
Nodes (3): T, TestContainerStatsNameFallback(), TestContainerStatsRoundTrip()

### Community 167 - "compressWriter"
Cohesion: 0.31
Nodes (4): compressible(), ResponseWriter, Writer, compressWriter

### Community 168 - "setPassword"
Cohesion: 0.67
Nodes (3): DB, resetPasswordCLI(), setPassword()

## Ambiguous Edges - Review These
- `LAN-only, no external backend` → `DCO sign-off, no CLA`  [AMBIGUOUS]
  CONTRIBUTING.md · relation: conceptually_related_to
- `Light-only theme` → `Blocking theme-boot script`  [AMBIGUOUS]
  DESIGN.md · relation: conceptually_related_to

## Knowledge Gaps
- **160 isolated node(s):** `dockerPSLine`, `dockerStatsLine`, `config`, `DiskUsage`, `ProcessSample` (+155 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **27 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **What is the exact relationship between `LAN-only, no external backend` and `DCO sign-off, no CLA`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **What is the exact relationship between `Light-only theme` and `Blocking theme-boot script`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **Why does `New()` connect `New` to `publishAgent`, `mustVerify`, `signing.go`, `Install`, `postWebhook`, `google_test.go`, `google_auth_test.go`, `Push`, `setPassword`, `Send`, `decodeJSON`, `control_test.go`, `.handlePutSettings`, `writeError`, `crypto_test.go`, `T`, `Open`, `users_cli.go`, `app`, `newPusher`, `app`, `.sendMail`, `.dispatchDue`?**
  _High betweenness centrality (0.296) - this node is a cross-community bridge._
- **Why does `Public portal front door` connect `.handleIngest` to `Portal.tsx`?**
  _High betweenness centrality (0.288) - this node is a cross-community bridge._
- **Are the 225 inferred relationships involving `newTestServer()` (e.g. with `TestAuditListsAndFilters()` and `TestServerInfoAdminOnly()`) actually correct?**
  _`newTestServer()` has 225 INFERRED edges - model-reasoned connections that need verification._
- **Are the 112 inferred relationships involving `signup()` (e.g. with `TestAuditListsAndFilters()` and `TestServerInfoAdminOnly()`) actually correct?**
  _`signup()` has 112 INFERRED edges - model-reasoned connections that need verification._
- **Are the 119 inferred relationships involving `writeError()` (e.g. with `validPrincipalType()` and `.applyCollectionIDs()`) actually correct?**
  _`writeError()` has 119 INFERRED edges - model-reasoned connections that need verification._