# Graph Report - Reeve  (2026-07-29)

## Corpus Check
- 315 files · ~200,378 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 2848 nodes · 7311 edges · 175 communities (148 shown, 27 thin omitted)
- Extraction: 81% EXTRACTED · 19% INFERRED · 0% AMBIGUOUS · INFERRED: 1403 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `9c7cc27d`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- publishAgent
- release.py
- secretFields.ts
- signing.go
- Install
- App.tsx
- app
- HostMetrics.tsx
- Hosts.tsx
- signup
- google_test.go
- adminClient
- replaceHostDisks
- app
- ui.tsx
- Dashboard.tsx
- Portal.tsx
- schema.sql
- agent_update_handlers_test.go
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
- newUser
- app
- Layout.tsx
- T
- google_auth_test.go
- store_test.go
- profile_handlers_test.go
- openTemp
- seedHost
- DB
- runOnce
- compilerOptions
- .ApplyPush
- Send
- countRows
- .handleSSHInstall
- control_test.go
- DB
- HostInventory.tsx
- zzz-visual.spec.ts
- collect/metrics.go
- .evaluateHostThresholds
- .handlePutSettings
- DB
- DB
- api
- users_cli.go
- DB
- NewHostSampler
- devDependencies
- HandlerFunc
- DB
- T
- Open
- collect_test.go
- app
- api.ts
- NewID
- dependencies
- app
- MapPicker.tsx
- Push
- .handleCreateCommand
- .loadEndpoint
- .handleCreateCredential
- mustVerify
- newCLIEnv
- test_build_config.py
- ScanLogErrors
- downloadBackup
- lookupMDNS
- .handleSetHostAutoUpdate
- newPusher
- .handleLogin
- newDownloadApp
- AdminServerInfo.tsx
- .handleGetHostThresholds
- DB
- New
- contracts.go
- decodeJSON
- .handleGeocode
- rbac.go
- writeJSON
- AccessRequest
- processes_test.go
- DB
- .checkSchemaDrift
- theme.ts
- writeError
- app
- mustUser
- zz-agent-updates.spec.ts
- zzzz-location-autosave.spec.ts
- TestHostEventsEndpoint
- live_ssh_install_test.sh
- .uiHandler
- ssh_install_handlers_test.go
- scripts
- run
- effectiveThreshold
- contracts_test.go
- .dispatchDue
- webhooks_test.go
- package.json
- DB
- render
- tailwindcss
- TestHostNetRate
- zzz-button-alignment.spec.ts
- The agent runs as root
- uninstall.sh
- Every column has a name
- .handleInviteUser
- postWebhook
- testServer
- tool_visibility_handlers.go
- .handleTestEmail
- @types/react
- putSettings
- vite
- @vitejs/plugin-react
- e2e pins REEVE_DEV_PORT=8099
- vite-env.d.ts
- shutdown_test.go
- Structure over shadow
- github.com/thehelvijs/Reeve
- Audit tables are append-only by convention
- Password change signs out everywhere
- zz-header-search.spec.ts
- .handleListPrincipals
- @types/react-dom
- typescript
- postcss
- CommandResult
- @playwright/test
- compressWriter
- LeafletMap.tsx
- disks_test.go
- disks.ts
- theme/tailwind.config.js
- cssVar
- pre-commit.sh

## God Nodes (most connected - your core abstractions)
1. `newTestServer()` - 242 edges
2. `signup()` - 137 edges
3. `writeError()` - 129 edges
4. `adminClient()` - 99 edges
5. `openTemp()` - 80 edges
6. `writeJSON()` - 73 edges
7. `New()` - 55 edges
8. `FromContext()` - 55 edges
9. `testServer` - 46 edges
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

## Communities (175 total, 27 thin omitted)

### Community 0 - "publishAgent"
Cohesion: 0.12
Nodes (29): Forced update sits outside the rollout, The host's local veto is absolute, effectiveAutoUpdate(), Duration, Host, app, Time, app (+21 more)

### Community 1 - "release.py"
Cohesion: 0.06
Nodes (47): CI gate job, CI grants contents:read only, Actions pinned by commit, not tag, Rolling edge prerelease, Signing key scoped to steps, not the workflow, Tag-triggered versioned release, web-build and server-assets precede build, Pull override for a published image (+39 more)

### Community 2 - "secretFields.ts"
Cohesion: 0.39
Nodes (6): RevealModal(), SecretField(), isSensitiveField(), ORDER, orderedSecretFields(), SENSITIVE

### Community 3 - "signing.go"
Cohesion: 0.08
Nodes (56): releasePublicKey(), fetchBody(), config, Client, isSHA256Hex(), needsUpdate(), olderThan(), selfArch() (+48 more)

### Community 4 - "Install"
Cohesion: 0.10
Nodes (45): Builder, Channel, ClientConfig, clientConfig(), connect(), dial(), dialTCP(), envAssignments() (+37 more)

### Community 5 - "App.tsx"
Cohesion: 0.11
Nodes (24): ApiError, SSHTarget, uploadAvatar(), User, App(), Protected(), AuthContext, AuthProvider() (+16 more)

### Community 6 - "app"
Cohesion: 0.28
Nodes (11): alwaysEditable(), Request, ResponseWriter, app, hostCanSee(), hostTarget(), toolCanEdit(), toolCanSee() (+3 more)

### Community 7 - "HostMetrics.tsx"
Cohesion: 0.14
Nodes (20): Colorblind-validated chart palette, ContainerPoint, fmtPct(), HostMetrics(), HostPoint, labelFor(), latest(), palette() (+12 more)

### Community 8 - "Hosts.tsx"
Cohesion: 0.09
Nodes (22): AgentUpdateRollup, AutoUpdatePolicy, Host, HostCommand, UpdateState, ACTION_LABEL, HostControls(), STATUS_TONE (+14 more)

### Community 9 - "signup"
Cohesion: 0.11
Nodes (40): T, TestServerInfoAdminOnly(), TestUpdateUserRoleAndActive(), fromIP(), Client, Response, T, login() (+32 more)

### Community 10 - "google_test.go"
Cohesion: 0.09
Nodes (27): Config, Profile, Config, Request, ResponseWriter, app, User, DomainAllowed() (+19 more)

### Community 11 - "adminClient"
Cohesion: 0.15
Nodes (42): TestAuditListsAndFilters(), createCred(), Client, T, newHost(), revealSecret(), TestAccessRequestOnAnUnknownHost(), TestAdminCanRevealAndAudited() (+34 more)

### Community 12 - "replaceHostDisks"
Cohesion: 0.18
Nodes (13): DiskUsage, DB, Time, replaceHostDisks(), accumulateProcessUsage(), ProcessSample, DB, Time (+5 more)

### Community 13 - "app"
Cohesion: 0.09
Nodes (32): clock, Throttle, throttleEntry, Int64, Once, Login throttle that cannot be held shut, REEVE_TRUST_PROXY is opt-in, DB (+24 more)

### Community 14 - "ui.tsx"
Cohesion: 0.08
Nodes (36): AccessRequest, AdminUser, AlertEvent, GrantAuditEntry, Group, HostInventory, InventoryItem, RevealAuditEntry (+28 more)

### Community 15 - "Dashboard.tsx"
Cohesion: 0.10
Nodes (27): Say what the state means, ToolStatus, EventHistory(), HostData, HostNode(), nodeDot(), nodeTypes, PortalGraph() (+19 more)

### Community 16 - "Portal.tsx"
Cohesion: 0.10
Nodes (22): Test every trigger of a modal, CollectionRef, endpointString(), goURL(), Tool, CollectionInfoModal(), HostInfoModal(), statusTone() (+14 more)

### Community 17 - "schema.sql"
Cohesion: 0.09
Nodes (36): One schema file, re-executed on open, No protocol version to negotiate, Everything but secrets is plaintext, access_requests, alert_events, alert_state, alert_thresholds, collection_editors (+28 more)

### Community 18 - "agent_update_handlers_test.go"
Cohesion: 0.23
Nodes (21): fetchRollup(), Client, StalledHost, T, Time, hostFromList(), reportBuild(), TestAgentUpdateEndpointsAreAdminOnly() (+13 more)

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
Cohesion: 0.13
Nodes (34): bearer(), T, TestAPIErrorEnvelope(), TestPublicEndpointsNoAuthExcludeRestricted(), TestPublicHostsNoAuth(), Client, T, noRedirect() (+26 more)

### Community 24 - "hostToView"
Cohesion: 0.19
Nodes (13): Host, Request, ResponseWriter, app, Time, hostStatus(), hostToView(), hostMetricsView (+5 more)

### Community 25 - "newTestServer"
Cohesion: 0.12
Nodes (33): T, TestGeocodeProxiesAndLabels(), TestGeocodeRequiresAdmin(), TestGeocodeShortQuerySkipsUpstream(), TestGeocodeUpstreamFailureIsBadGateway(), newTestServer(), T, TestHostMetricsEndpointCarriesDisks() (+25 more)

### Community 26 - "procs_test.go"
Cohesion: 0.07
Nodes (52): BenchmarkHostSamplerSample(), BenchmarkParseCrontab(), BenchmarkParseMounts(), BenchmarkParsePasswd(), BenchmarkProcSamplerSample(), BenchmarkTopProcs(), benchProcs(), benchProcTree() (+44 more)

### Community 27 - "gather"
Cohesion: 0.19
Nodes (21): durationArg(), gather(), gatherCron(), gatherDockerLogErrors(), gatherLogErrors(), config, Duration, Time (+13 more)

### Community 28 - "DB"
Cohesion: 0.09
Nodes (19): Stable slug and /go redirect, Catalog of services, Slugify(), dbWithUser(), DB, T, TestCreateToolFallsBackWhenTheNameHasNoSlug(), TestGetToolBySlug() (+11 more)

### Community 29 - "samplePush"
Cohesion: 0.17
Nodes (31): TestUpdateNowRefusesAVetoedHost(), controllableHost(), Client, T, queue(), TestAResultFromTheWrongHostIsIgnored(), TestCommandsAreAdminOnly(), TestQueueAndDeliverACommand() (+23 more)

### Community 30 - "newUser"
Cohesion: 0.27
Nodes (16): containsKey(), containsName(), Client, T, hasKey(), hasName(), jsonString(), newUser() (+8 more)

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

### Community 35 - "store_test.go"
Cohesion: 0.17
Nodes (21): DB, T, TestBackupToRefusesExistingDest(), TestCountToolsAndHosts(), TestDeliveryBackoff(), TestDowntimeSecs(), TestDueDeliveriesResolvesWebhook(), TestEffectiveRetentionDefaults() (+13 more)

### Community 36 - "profile_handlers_test.go"
Cohesion: 0.19
Nodes (21): Client, Response, T, pngHeader(), TestEveryImageSlot(), TestHostIconVisibility(), TestToolIconVisibilityAndLifecycle(), uploadIcon() (+13 more)

### Community 37 - "openTemp"
Cohesion: 0.25
Nodes (23): DB, T, Time, pacedSlot(), TestApplyPushRecordsVeto(), TestApplyPushWithAVetoReleasesTheSlot(), TestClearStalledUpdatesSweepsIneligibleHostsToo(), TestFleetDefaultOffMakesADefaultPolicyHostIneligible() (+15 more)

### Community 38 - "seedHost"
Cohesion: 0.20
Nodes (18): T, TestContainerStatsNameFallback(), TestContainerStatsRoundTrip(), T, TestApplyPushKeepsLastKnownChecksum(), TestApplyPushReplacesSnapshotAndRollsBack(), TestApplyPushStoresEverySection(), TestLatestHostMetricsBatchesEveryHost() (+10 more)

### Community 39 - "DB"
Cohesion: 0.22
Nodes (10): ContainerSample, HostMetrics, DB, Time, insertContainerStats(), insertHostMetric(), intCols(), scanLatestMetric() (+2 more)

### Community 40 - "runOnce"
Cohesion: 0.22
Nodes (19): commandReport(), config, Duration, Time, loadConfig(), main(), runOnce(), runSelfUpdate() (+11 more)

### Community 41 - "compilerOptions"
Cohesion: 0.09
Nodes (21): DOM, DOM.Iterable, ES2021, src, compilerOptions, allowImportingTsExtensions, isolatedModules, jsx (+13 more)

### Community 42 - ".ApplyPush"
Cohesion: 0.17
Nodes (11): ContainerState, DB, Time, insertLogEvents(), replaceContainerStatus(), replaceCronJobs(), replaceServiceStatus(), InventoryContainer (+3 more)

### Community 43 - "Send"
Cohesion: 0.21
Nodes (18): Config, Message, stubSMTP, BuildMessage(), Time, hasCRLF(), Send(), atoi() (+10 more)

### Community 44 - "countRows"
Cohesion: 0.35
Nodes (13): countRows(), enrollHostWin(), Client, T, TestAgentOfflineAlert(), TestDispatchSendsAndRetries(), TestDownAlertDebounceFireResolve(), TestFlapSuppressed() (+5 more)

### Community 45 - ".handleSSHInstall"
Cohesion: 0.22
Nodes (6): errNoAgentBuild, errUnsupportedArch, Request, ResponseWriter, app, sshTargetInput

### Community 46 - "control_test.go"
Cohesion: 0.29
Nodes (15): argvFor(), newController(), fakeRunner(), T, TestArgvForEveryAction(), TestArgvForRefusesADangerousTarget(), TestArgvForRejectsAnUnknownAction(), TestControlDefaultsOn() (+7 more)

### Community 47 - "DB"
Cohesion: 0.19
Nodes (7): channelAccepts(), filterBySeverity(), DB, Time, severityRank(), Delivery, Webhook

### Community 48 - "HostInventory.tsx"
Cohesion: 0.11
Nodes (28): Enter confirms every form, ThresholdsPayload, EMPTY_ROW, METRIC_KEYS, METRICS, ThresholdRow, ThresholdRows, thresholdToDisplay() (+20 more)

### Community 49 - "zzz-visual.spec.ts"
Cohesion: 0.09
Nodes (8): CI Playwright e2e job, axe against WCAG 2 A/AA, Manual screenshot sweep after any frontend change, Pixel baselines for every page, TAGS, THEMES, GREY_TILE, THEMES

### Community 50 - "collect/metrics.go"
Cohesion: 0.14
Nodes (22): TestParseCPUStatAndPercent(), TestParseUptimeAndNetDev(), T, TestHostSamplerReportsDiskIO(), TestParseDiskStatsCountsTheWholeDiskOnce(), TestParseDiskStatsKeepsNVMeWholeDisks(), TestParseDiskStatsSkipsVirtualDevices(), TestParseDiskStatsSumsSeparateDisks() (+14 more)

### Community 51 - ".evaluateHostThresholds"
Cohesion: 0.15
Nodes (14): fmtRate(), Duration, Host, app, Time, Webhook, hostMetricValues(), severityOrError() (+6 more)

### Community 52 - ".handlePutSettings"
Cohesion: 0.15
Nodes (14): Agent update state machine, Checksum decides currency, not version, Retention, agentUpdateView, googleAuthView, googleInput, retentionView, Request (+6 more)

### Community 53 - "DB"
Cohesion: 0.21
Nodes (8): Rows, DB, Time, scanAlertEvents(), Time, parseNullableTime(), AlertEvent, AlertState

### Community 54 - "DB"
Cohesion: 0.16
Nodes (3): DB, Time, Host

### Community 55 - "api"
Cohesion: 0.12
Nodes (15): api, Collection, CollectionDetail, Principals, uploadIcon(), VisibilityGrant, IconUploader(), message() (+7 more)

### Community 56 - "users_cli.go"
Cohesion: 0.43
Nodes (16): deletionHeir(), firstArg(), DB, User, Writer, guardLastActiveAdmin(), lookup(), normalizeEmail() (+8 more)

### Community 57 - "DB"
Cohesion: 0.23
Nodes (5): NullString, DB, Time, parseNullTime(), HostCommand

### Community 58 - "NewHostSampler"
Cohesion: 0.30
Nodes (12): fakeNvidiaSMI(), T, TestSampleGPUKeepsAnIdleCard(), TestSampleGPUReadsAReportingCard(), TestSampleGPURecoversFromOneBadReading(), TestSampleGPUToleratesAFailingProbe(), TestSampleGPUWithoutABinary(), NewHostSampler() (+4 more)

### Community 59 - "devDependencies"
Cohesion: 0.12
Nodes (17): autoprefixer, @axe-core/playwright, eslint, eslint-plugin-react-hooks, @types/leaflet, @typescript-eslint/eslint-plugin, @typescript-eslint/parser, vitest (+9 more)

### Community 60 - "HandlerFunc"
Cohesion: 0.24
Nodes (16): HandlerFunc, Handler, Handler, gzipResponses(), Handler, Response, T, gzipRequest() (+8 more)

### Community 61 - "DB"
Cohesion: 0.16
Nodes (4): DB, Time, Group, GroupMember

### Community 62 - "T"
Cohesion: 0.26
Nodes (12): captureLog(), Buffer, Client, Response, T, TestClientIPForwardedForOnlyWhenProxyTrusted(), TestHealthz(), TestLoginSetsTheReeveSessionCookie() (+4 more)

### Community 63 - "Open"
Cohesion: 0.13
Nodes (13): T, TestDeclaredColumnsReadsTheSchema(), TestOpenAcceptsACurrentDatabase(), TestOpenRefusesADatabaseMissingAColumn(), ApplyStagedRestore(), DB, Open(), TestApplyStagedRestore() (+5 more)

### Community 64 - "collect_test.go"
Cohesion: 0.11
Nodes (24): T, TestParseCrontabSystemForm(), TestParseCrontabUserForm(), TestParseDockerPS(), TestParseDockerStats(), TestParseLoadAvg(), TestParseMemInfo(), TestParseNvidiaSMI() (+16 more)

### Community 65 - "app"
Cohesion: 0.23
Nodes (7): Every reveal is audited, Request, ResponseWriter, app, User, adminUserView, inviteView

### Community 66 - "api.ts"
Cohesion: 0.07
Nodes (28): AgentUpdateSettings, ChannelKind, CommandStatus, Credential, CREDENTIAL_FIELDS, CREDENTIAL_LABEL, CredentialType, Delivery (+20 more)

### Community 67 - "NewID"
Cohesion: 0.15
Nodes (6): DB, Time, scanCredentialMeta(), NewID(), AccessGrant, Credential

### Community 68 - "dependencies"
Cohesion: 0.13
Nodes (15): @fontsource-variable/inter, leaflet, react, react-dom, react-router-dom, uplot, dependencies, @fontsource-variable/inter (+7 more)

### Community 69 - "app"
Cohesion: 0.23
Nodes (8): Writes must come from this origin, dynamicRequest(), Handler, Request, app, User, User, PrincipalFromUser()

### Community 70 - "MapPicker.tsx"
Cohesion: 0.27
Nodes (12): MapPicker(), Saved, cities, City, cityMatches(), locationKey(), lookupAddress(), mergePlaces() (+4 more)

### Community 71 - "Push"
Cohesion: 0.09
Nodes (24): Host enrollment token, Ingest push contract, Commands ride the push ack, HostMetrics, ProcessSample, Time, CronState, Push (+16 more)

### Community 72 - ".handleCreateCommand"
Cohesion: 0.29
Nodes (6): HostCommand, Request, ResponseWriter, app, commandInput, commandView

### Community 73 - ".loadEndpoint"
Cohesion: 0.27
Nodes (9): 404 instead of 403 for invisible records, Follow-the-host endpoint resolution, Host, Request, ResponseWriter, app, Tool, resolveEndpoint() (+1 more)

### Community 74 - ".handleCreateCredential"
Cohesion: 0.33
Nodes (6): Credential, credentialAAD(), Host, Request, ResponseWriter, app

### Community 75 - "mustVerify"
Cohesion: 0.15
Nodes (18): auditLink(), auditWhere(), DB, Time, chainedDB(), DB, T, mustVerify() (+10 more)

### Community 76 - "newCLIEnv"
Cohesion: 0.11
Nodes (39): server users CLI, cliEnv, HashPassword(), HashToken(), NewAgentToken(), NewSessionToken(), newToken(), T (+31 more)

### Community 77 - "test_build_config.py"
Cohesion: 0.23
Nodes (8): parametrize, dry_run(), job_commands(), test_actions_are_pinned_to_a_commit(), test_npm_target_installs_its_own_deps(), test_signing_key_is_never_workflow_or_job_scoped(), test_workflow_jobs_install_web_deps_before_running_npm_scripts(), test_workflows_grant_no_blanket_permissions()

### Community 78 - "ScanLogErrors"
Cohesion: 0.19
Nodes (14): TestScanLogErrors(), Time, ScanLogErrors(), basename(), maskInline(), namesSecret(), redactSecrets(), T (+6 more)

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

### Community 83 - ".handleLogin"
Cohesion: 0.40
Nodes (4): Request, ResponseWriter, app, User

### Community 84 - "newDownloadApp"
Cohesion: 0.36
Nodes (11): MapFS, app, T, newDownloadApp(), TestRoutesDownloadDispatch(), TestServeAgentBinaryKnownArch(), TestServeAgentBinaryUnknownArch404(), TestServeAgentChecksumMatchesBytes() (+3 more)

### Community 85 - "AdminServerInfo.tsx"
Cohesion: 0.21
Nodes (10): MetricBar(), fmtBytes(), fmtCount(), fmtRate(), fmtUptime(), AdminServerInfo(), ServerInfo, CollectionDetail() (+2 more)

### Community 86 - ".handleGetHostThresholds"
Cohesion: 0.33
Nodes (5): Request, ResponseWriter, app, thresholdDTO, thresholdsPayload

### Community 87 - "DB"
Cohesion: 0.24
Nodes (5): blockingSlot(), DB, Time, ValidAutoUpdatePolicy(), StalledHost

### Community 88 - "New"
Cohesion: 0.21
Nodes (19): AEAD, Cipher, Compose refuses to start without the master key, Backup carries ciphertext, not the key, Credentials encrypted before they touch disk, Master-key custody, New(), NewFromEnv() (+11 more)

### Community 89 - "contracts.go"
Cohesion: 0.14
Nodes (15): Fixed action allowlist, no shell, Collections replace free-text category, Single error envelope, CollectionRef, CommandTargetKind, DiskUsage, DiskUsage, ErrorResponse (+7 more)

### Community 90 - "decodeJSON"
Cohesion: 0.26
Nodes (8): Request, ResponseWriter, app, groupRole(), decodeJSON(), Request, ResponseWriter, app

### Community 91 - ".handleGeocode"
Cohesion: 0.31
Nodes (8): Request, ResponseWriter, app, photonHits(), photonLabel(), geocodeHit, photonFeature, photonResponse

### Community 92 - "rbac.go"
Cohesion: 0.27
Nodes (10): Session cookie auth, ctxKey, Single-tenant, admin sees everything, Context, Handler, ResponseWriter, RequireAdmin(), RequireAuth() (+2 more)

### Community 93 - "writeJSON"
Cohesion: 0.14
Nodes (13): One Go binary serves UI and API, channelView, ResponseWriter, writeJSON(), Request, ResponseWriter, app, statusRecorder (+5 more)

### Community 94 - "AccessRequest"
Cohesion: 0.35
Nodes (4): DB, Time, scanRequest(), AccessRequest

### Community 95 - "processes_test.go"
Cohesion: 0.33
Nodes (10): T, TestDeleteHostMetricsDropsProcesses(), TestDeleteHostMetricsDropsProcessUsage(), TestHostProcessesNilStoresEmptyList(), TestHostProcessesRoundTripAndOverwrite(), TestLatestHostProcessesUnknownHost(), TestProcessUsageAveragesAndPeaks(), TestProcessUsageKeepsTopByMemory() (+2 more)

### Community 96 - "DB"
Cohesion: 0.36
Nodes (4): Duration, DB, Time, Retention

### Community 98 - "theme.ts"
Cohesion: 0.31
Nodes (9): ThemeToggle(), canvasFor(), listeners, otherTheme(), readTheme(), setTheme(), snapshot(), subscribe() (+1 more)

### Community 99 - "writeError"
Cohesion: 0.15
Nodes (14): ReadCloser, Request, ResponseWriter, app, writeError(), Request, ResponseWriter, readImageUpload() (+6 more)

### Community 100 - "app"
Cohesion: 0.44
Nodes (4): File, Request, ResponseWriter, app

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
Cohesion: 0.25
Nodes (6): agentDistFS(), assetCacheControl(), FS, Handler, app, newTestServerKey()

### Community 107 - "ssh_install_handlers_test.go"
Cohesion: 0.39
Nodes (11): createHostFor(), Client, T, sshBody(), TestAgentBinaryForUname(), TestReachableFromOtherHosts(), TestSSHEndpointsAreAdminOnlyAndScopedToAHost(), TestSSHInstallRejectsIncompleteTargets() (+3 more)

### Community 108 - "scripts"
Cohesion: 0.25
Nodes (8): scripts, build, dev, e2e, lint, preview, test, typecheck

### Community 109 - "run"
Cohesion: 0.25
Nodes (10): envOr(), every(), Context, Duration, Server, app, main(), run() (+2 more)

### Community 110 - "effectiveThreshold"
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

### Community 129 - ".handleInviteUser"
Cohesion: 0.21
Nodes (9): Request, ResponseWriter, app, hashResetToken(), newResetToken(), reachableFromOtherHosts(), Request, ResponseWriter (+1 more)

### Community 130 - "postWebhook"
Cohesion: 0.29
Nodes (9): postWebhook(), redactConfig(), T, TestDispatchFailsChannelWithNoURL(), TestDispatchMarksSentAndFailed(), TestDueDeliveriesReturnsKindConfig(), TestPostWebhookBearerToken(), TestPostWebhookSendsPayload() (+1 more)

### Community 131 - "testServer"
Cohesion: 0.19
Nodes (18): Client, T, groupsFor(), newGroup(), TestMembershipSpansGroupsAndKeepsRoles(), TestModeratorCannotReachAccounts(), TestModeratorCannotStandThemselvesDown(), TestModeratorManagesOnlyTheirOwnGroup() (+10 more)

### Community 133 - ".handleTestEmail"
Cohesion: 0.22
Nodes (6): Config, Request, ResponseWriter, app, smtpInput, smtpView

### Community 135 - "putSettings"
Cohesion: 0.37
Nodes (12): getSettings(), Client, Response, T, putSettings(), TestSealedSettingRoundTrip(), TestSettingsAdminOnly(), TestSettingsDefaults() (+4 more)

### Community 144 - "shutdown_test.go"
Cohesion: 0.30
Nodes (9): Conn, probeHealth(), T, TestEveryStopsOnCancel(), TestProbeHealth(), TestProbeHealthFailsOnNon200(), TestServeDrainsInFlightRequest(), TestServeReturnsListenError() (+1 more)

### Community 162 - ".handleListPrincipals"
Cohesion: 0.28
Nodes (7): Request, ResponseWriter, app, localPart(), principalGroup, principalsView, principalUser

### Community 166 - "CommandResult"
Cohesion: 0.40
Nodes (4): Mutex, controller, CommandResult, IsPowerAction()

### Community 168 - "compressWriter"
Cohesion: 0.31
Nodes (4): compressible(), ResponseWriter, Writer, compressWriter

### Community 169 - "LeafletMap.tsx"
Cohesion: 0.39
Nodes (6): LeafletMap(), MapPoint, tileURL(), pinFill(), pinSVG(), Theme

### Community 170 - "disks_test.go"
Cohesion: 0.48
Nodes (6): T, TestApplyPushStoresDisks(), TestDeleteHostMetricsDropsDisks(), TestHostDisksNilStoresEmptyList(), TestHostDisksRoundTripAndReplace(), TestLatestHostDisksUnknownHost()

### Community 171 - "disks.ts"
Cohesion: 0.48
Nodes (4): DiskList(), DiskUsage, sortDisks(), usedPct()

### Community 172 - "theme/tailwind.config.js"
Cohesion: 0.33
Nodes (4): Light-only theme, Four-radius vocabulary, Two colors do two jobs, Blocking theme-boot script

### Community 173 - "cssVar"
Cohesion: 0.60
Nodes (4): autoAxisSize(), Chart(), Series, cssVar()

## Ambiguous Edges - Review These
- `Light-only theme` → `Blocking theme-boot script`  [AMBIGUOUS]
  DESIGN.md · relation: conceptually_related_to

## Knowledge Gaps
- **158 isolated node(s):** `dockerPSLine`, `dockerStatsLine`, `config`, `DiskUsage`, `ProcessSample` (+153 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **27 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **What is the exact relationship between `Light-only theme` and `Blocking theme-boot script`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **Why does `LAN-only, no external backend` connect `Push` to `DB`?**
  _High betweenness centrality (0.281) - this node is a cross-community bridge._
- **Why does `Public portal front door` connect `Push` to `Portal.tsx`?**
  _High betweenness centrality (0.280) - this node is a cross-community bridge._
- **Why does `New()` connect `New` to `publishAgent`, `postWebhook`, `signing.go`, `Install`, `.handleTestEmail`, `testServer`, `google_test.go`, `google_auth_test.go`, `Send`, `control_test.go`, `.handlePutSettings`, `users_cli.go`, `T`, `Open`, `app`, `Push`, `mustVerify`, `newCLIEnv`, `newPusher`, `decodeJSON`, `writeError`, `app`, `.uiHandler`, `.dispatchDue`?**
  _High betweenness centrality (0.237) - this node is a cross-community bridge._
- **Are the 236 inferred relationships involving `newTestServer()` (e.g. with `TestAuditListsAndFilters()` and `TestServerInfoAdminOnly()`) actually correct?**
  _`newTestServer()` has 236 INFERRED edges - model-reasoned connections that need verification._
- **Are the 118 inferred relationships involving `signup()` (e.g. with `TestAuditListsAndFilters()` and `TestServerInfoAdminOnly()`) actually correct?**
  _`signup()` has 118 INFERRED edges - model-reasoned connections that need verification._
- **Are the 125 inferred relationships involving `writeError()` (e.g. with `validPrincipalType()` and `.applyCollectionIDs()`) actually correct?**
  _`writeError()` has 125 INFERRED edges - model-reasoned connections that need verification._