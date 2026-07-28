# Graph Report - Reeve  (2026-07-28)

## Corpus Check
- 315 files · ~199,616 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 2845 nodes · 7289 edges · 165 communities (140 shown, 25 thin omitted)
- Extraction: 81% EXTRACTED · 19% INFERRED · 0% AMBIGUOUS · INFERRED: 1395 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `687c3626`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- publishAgent
- release.py
- RevealModal.tsx
- signing.go
- Install
- HostInventory.tsx
- app
- HostMetrics.tsx
- Hosts.tsx
- signup
- google_test.go
- adminClient
- replaceHostDisks
- app
- Button
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
- newUser
- app
- Layout.tsx
- T
- New
- openTemp
- profile_handlers_test.go
- store/agent_update_test.go
- seedHost
- DB
- runOnce
- compilerOptions
- .ApplyPush
- Send
- countRows
- decodeJSON
- control_test.go
- DB
- AdminAlerts.tsx
- zzz-visual.spec.ts
- collect_test.go
- .evaluateHostThresholds
- .handleUploadAvatar
- DB
- DB
- api.ts
- dbWithUser
- DB
- NewHostSampler
- devDependencies
- HandlerFunc
- shutdown_test.go
- T
- Open
- docker.go
- writeJSON
- putSettings
- DB
- dependencies
- app
- .handleIngest
- Push
- .handleCreateCommand
- .loadEndpoint
- .handleCreateCredential
- NewID
- newCLIEnv
- test_build_config.py
- ParseCrontab
- downloadBackup
- lookupMDNS
- .handleSetHostAutoUpdate
- newPusher
- ParseDiskStats
- newDownloadApp
- .handleCreateWebhook
- .handleGetHostThresholds
- DB
- ThresholdSet
- TestPublicHostsNoAuth
- writeError
- .handleGeocode
- rbac.go
- .handleListPrincipals
- AccessRequest
- processes_test.go
- DB
- .checkSchemaDrift
- MapPicker.tsx
- .handleRestoreUpload
- TestServerInfoAdminOnly
- mustUser
- zz-agent-updates.spec.ts
- zzzz-location-autosave.spec.ts
- TestHostEventsEndpoint
- live_ssh_install_test.sh
- .uiHandler
- ssh_install_handlers_test.go
- scripts
- vitest
- effectiveThreshold
- contracts_test.go
- pusher
- webhooks_test.go
- package.json
- DB
- render
- tailwindcss
- TestHostNetRate
- zzz-button-alignment.spec.ts
- agent/ must not import server/
- uninstall.sh
- Every column has a name
- mustVerify
- confirmText.ts
- testServer
- tool_visibility_handlers.go
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
- @types/react-dom
- typescript
- postcss
- TestContainerStatsNameFallback
- compressWriter

## God Nodes (most connected - your core abstractions)
1. `newTestServer()` - 239 edges
2. `signup()` - 136 edges
3. `writeError()` - 128 edges
4. `adminClient()` - 96 edges
5. `openTemp()` - 80 edges
6. `writeJSON()` - 72 edges
7. `FromContext()` - 55 edges
8. `New()` - 54 edges
9. `testServer` - 45 edges
10. `createTool()` - 41 edges

## Surprising Connections (you probably didn't know these)
- `Checksum decides currency, not version` --references--> `signedVersion()`  [INFERRED]
  deploy/README.md → agent/update.go
- `Host and service controls` --references--> `Command`  [EXTRACTED]
  README.md → contracts/contracts.go
- `server users CLI` --references--> `VerifyPassword()`  [INFERRED]
  deploy/README.md → server/internal/auth/auth.go
- `Updates only move forward` --rationale_for--> `signedVersion()`  [EXTRACTED]
  SECURITY.md → agent/update.go
- `Push monitoring` --references--> `Push`  [EXTRACTED]
  README.md → contracts/contracts.go

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Agent trust boundary** — security_signed_releases, security_downgrade_protection, security_agent_local_veto, security_agent_target_validation, security_agent_runs_as_root [EXTRACTED 0.95]
- **Fleet update rollout** — contracts_api_update_state, contracts_api_rollout_pause, contracts_api_forced_slot, deploy_readme_checksum_not_version, security_agent_local_veto [EXTRACTED 0.95]
- **Not leaking what you cannot see** — contracts_api_404_not_403, contracts_api_visibility, contracts_api_slug, security_single_tenant [INFERRED 0.85]

## Communities (165 total, 25 thin omitted)

### Community 0 - "publishAgent"
Cohesion: 0.08
Nodes (50): Forced update sits outside the rollout, The host's local veto is absolute, effectiveAutoUpdate(), Duration, Host, app, Time, fetchRollup() (+42 more)

### Community 1 - "release.py"
Cohesion: 0.05
Nodes (49): CI gate job, CI grants contents:read only, Rolling edge prerelease, Signing key scoped to steps, not the workflow, Tag-triggered versioned release, web-build and server-assets precede build, make gate / web-check / pytest before commit, Two-second pre-commit gate (+41 more)

### Community 2 - "RevealModal.tsx"
Cohesion: 0.38
Nodes (7): RevealedCredential, RevealModal(), SecretField(), isSensitiveField(), ORDER, orderedSecretFields(), SENSITIVE

### Community 3 - "signing.go"
Cohesion: 0.08
Nodes (58): Actions pinned by commit, not tag, releasePublicKey(), fetchBody(), config, Client, isSHA256Hex(), needsUpdate(), olderThan() (+50 more)

### Community 4 - "Install"
Cohesion: 0.10
Nodes (45): Builder, Channel, ClientConfig, clientConfig(), connect(), dial(), dialTCP(), envAssignments() (+37 more)

### Community 5 - "HostInventory.tsx"
Cohesion: 0.12
Nodes (24): HostCommand, HostInventory, InventoryItem, ThresholdsPayload, AddServiceModal(), Chevron(), ACTION_LABEL, HostControls() (+16 more)

### Community 6 - "app"
Cohesion: 0.28
Nodes (11): alwaysEditable(), Request, ResponseWriter, app, hostCanSee(), hostTarget(), toolCanEdit(), toolCanSee() (+3 more)

### Community 7 - "HostMetrics.tsx"
Cohesion: 0.09
Nodes (32): Colorblind-validated chart palette, DiskList(), ContainerPoint, fmtPct(), HostMetrics(), HostPoint, labelFor(), latest() (+24 more)

### Community 8 - "Hosts.tsx"
Cohesion: 0.10
Nodes (18): AgentUpdateRollup, AutoUpdatePolicy, UpdateState, ListSkeleton(), Skeleton(), POLICY_LABEL, showsVersionPill(), UPDATE_LABEL (+10 more)

### Community 9 - "signup"
Cohesion: 0.11
Nodes (42): fromIP(), Client, Response, T, login(), loginWith(), signup(), TestAuthStatusSetupFlag() (+34 more)

### Community 10 - "google_test.go"
Cohesion: 0.09
Nodes (29): Config, Profile, Config, Request, ResponseWriter, app, User, googleAuthView (+21 more)

### Community 11 - "adminClient"
Cohesion: 0.15
Nodes (42): TestAuditListsAndFilters(), createCred(), Client, T, newHost(), revealSecret(), TestAccessRequestOnAnUnknownHost(), TestAdminCanRevealAndAudited() (+34 more)

### Community 12 - "replaceHostDisks"
Cohesion: 0.18
Nodes (13): DiskUsage, DB, Time, replaceHostDisks(), accumulateProcessUsage(), ProcessSample, DB, Time (+5 more)

### Community 13 - "app"
Cohesion: 0.07
Nodes (41): clock, Throttle, throttleEntry, Int64, Once, Login throttle that cannot be held shut, REEVE_TRUST_PROXY is opt-in, DB (+33 more)

### Community 14 - "Button"
Cohesion: 0.07
Nodes (39): Enter confirms every form, AdminUser, ApiError, GroupMember, Invite, Severity, SSHTarget, uploadAvatar() (+31 more)

### Community 15 - "Dashboard.tsx"
Cohesion: 0.14
Nodes (20): Say what the state means, AccessRequest, AlertEvent, ToolStatus, ServiceDots(), Eyebrow(), ALERT_LABEL, alertLabel() (+12 more)

### Community 16 - "Portal.tsx"
Cohesion: 0.08
Nodes (34): Test every trigger of a modal, CollectionRef, endpointString(), goURL(), Host, Tool, CollectionInfoModal(), HostInfoModal() (+26 more)

### Community 17 - "schema.sql"
Cohesion: 0.09
Nodes (36): One schema file, re-executed on open, No protocol version to negotiate, Everything but secrets is plaintext, access_requests, alert_events, alert_state, alert_thresholds, collection_editors (+28 more)

### Community 18 - "run"
Cohesion: 0.23
Nodes (11): envOr(), every(), Context, Duration, Server, app, main(), probeHealth() (+3 more)

### Community 19 - "Principal"
Cohesion: 0.16
Nodes (20): Principal, CollectionRef, assetURL(), publicToolResponse, agentMonitored(), checkEndpointFields(), Host, Request (+12 more)

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
Cohesion: 0.11
Nodes (37): T, TestGeocodeProxiesAndLabels(), TestGeocodeRequiresAdmin(), TestGeocodeShortQuerySkipsUpstream(), TestGeocodeUpstreamFailureIsBadGateway(), newTestServer(), T, TestHostMetricsEndpointCarriesDisks() (+29 more)

### Community 26 - "procs_test.go"
Cohesion: 0.07
Nodes (52): BenchmarkHostSamplerSample(), BenchmarkParseCrontab(), BenchmarkParseMounts(), BenchmarkParsePasswd(), BenchmarkProcSamplerSample(), BenchmarkTopProcs(), benchProcs(), benchProcTree() (+44 more)

### Community 27 - "gather"
Cohesion: 0.15
Nodes (25): TestScanLogErrors(), Time, ScanLogErrors(), durationArg(), gather(), gatherCron(), gatherDockerLogErrors(), gatherLogErrors() (+17 more)

### Community 28 - "DB"
Cohesion: 0.14
Nodes (7): boolToInt(), DB, Time, nullable(), Tool, ToolFilter, VisibilityGrant

### Community 29 - "samplePush"
Cohesion: 0.17
Nodes (31): TestUpdateNowRefusesAVetoedHost(), controllableHost(), Client, T, queue(), TestAResultFromTheWrongHostIsIgnored(), TestCommandsAreAdminOnly(), TestQueueAndDeliverACommand() (+23 more)

### Community 30 - "newUser"
Cohesion: 0.27
Nodes (16): containsKey(), containsName(), Client, T, hasKey(), hasName(), jsonString(), newUser() (+8 more)

### Community 31 - "app"
Cohesion: 0.26
Nodes (7): AccessRequest, Request, ResponseWriter, app, validPrincipalType(), accessGrantView, requestView

### Community 32 - "Layout.tsx"
Cohesion: 0.08
Nodes (22): Group, UptimeSummary, EventHistory(), adminNav, Layout(), NavRecord, primaryNav, IconName (+14 more)

### Community 33 - "T"
Cohesion: 0.26
Nodes (19): configureRelay(), doReset(), forgot(), Client, Response, T, newRelay(), resetTokenFrom() (+11 more)

### Community 34 - "New"
Cohesion: 0.09
Nodes (62): AEAD, Cipher, Compose refuses to start without the master key, Backup carries ciphertext, not the key, File, Credentials encrypted before they touch disk, Master-key custody, Request (+54 more)

### Community 35 - "openTemp"
Cohesion: 0.20
Nodes (26): ApplyStagedRestore(), DB, T, openTemp(), TestApplyStagedRestore(), TestApplyStagedRestoreNoOp(), TestBackupTo(), TestBackupToRefusesExistingDest() (+18 more)

### Community 36 - "profile_handlers_test.go"
Cohesion: 0.19
Nodes (21): Client, Response, T, pngHeader(), TestEveryImageSlot(), TestHostIconVisibility(), TestToolIconVisibilityAndLifecycle(), uploadIcon() (+13 more)

### Community 37 - "store/agent_update_test.go"
Cohesion: 0.20
Nodes (22): DB, T, Time, pacedSlot(), TestApplyPushRecordsVeto(), TestApplyPushWithAVetoReleasesTheSlot(), TestClearStalledUpdatesSweepsIneligibleHostsToo(), TestFleetDefaultOffMakesADefaultPolicyHostIneligible() (+14 more)

### Community 38 - "seedHost"
Cohesion: 0.18
Nodes (21): T, TestApplyPushStoresDisks(), TestDeleteHostMetricsDropsDisks(), TestHostDisksNilStoresEmptyList(), TestHostDisksRoundTripAndReplace(), TestLatestHostDisksUnknownHost(), T, TestApplyPushKeepsLastKnownChecksum() (+13 more)

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
Cohesion: 0.07
Nodes (36): Agent update state machine, Checksum decides currency, not version, Config, Message, stubSMTP, Retention, agentUpdateView, BuildMessage() (+28 more)

### Community 44 - "countRows"
Cohesion: 0.35
Nodes (13): countRows(), enrollHostWin(), Client, T, TestAgentOfflineAlert(), TestDispatchSendsAndRetries(), TestDownAlertDebounceFireResolve(), TestFlapSuppressed() (+5 more)

### Community 45 - "decodeJSON"
Cohesion: 0.08
Nodes (21): adminUserView, errNoAgentBuild, errUnsupportedArch, Request, ResponseWriter, app, hashResetToken(), newResetToken() (+13 more)

### Community 46 - "control_test.go"
Cohesion: 0.29
Nodes (15): argvFor(), newController(), fakeRunner(), T, TestArgvForEveryAction(), TestArgvForRefusesADangerousTarget(), TestArgvForRejectsAnUnknownAction(), TestControlDefaultsOn() (+7 more)

### Community 47 - "DB"
Cohesion: 0.09
Nodes (11): DB, Time, channelAccepts(), filterBySeverity(), DB, Time, severityRank(), Delivery (+3 more)

### Community 48 - "AdminAlerts.tsx"
Cohesion: 0.11
Nodes (23): Delivery, GrantAuditEntry, RevealAuditEntry, ThresholdMetric, EmptyState(), EMPTY_ROW, METRIC_KEYS, METRICS (+15 more)

### Community 49 - "zzz-visual.spec.ts"
Cohesion: 0.09
Nodes (9): CI Playwright e2e job, Frontend changes are reviewed in a browser, axe against WCAG 2 A/AA, Manual screenshot sweep after any frontend change, Pixel baselines for every page, TAGS, THEMES, GREY_TILE (+1 more)

### Community 50 - "collect_test.go"
Cohesion: 0.14
Nodes (23): T, TestParseCPUStatAndPercent(), TestParseCrontabSystemForm(), TestParseCrontabUserForm(), TestParseLoadAvg(), TestParseMemInfo(), TestParseNvidiaSMI(), TestParseUptimeAndNetDev() (+15 more)

### Community 51 - ".evaluateHostThresholds"
Cohesion: 0.25
Nodes (10): fmtRate(), Duration, Host, app, Time, Webhook, hostMetricValues(), severityOrError() (+2 more)

### Community 52 - ".handleUploadAvatar"
Cohesion: 0.19
Nodes (9): ReadCloser, Request, ResponseWriter, readImageUpload(), writeImageFile(), formFile(), Request, ResponseWriter (+1 more)

### Community 53 - "DB"
Cohesion: 0.21
Nodes (8): Rows, DB, Time, scanAlertEvents(), Time, parseNullableTime(), AlertEvent, AlertState

### Community 54 - "DB"
Cohesion: 0.16
Nodes (3): DB, Time, Host

### Community 55 - "api.ts"
Cohesion: 0.05
Nodes (40): AgentUpdateSettings, api, ChannelKind, Collection, CollectionDetail, CommandStatus, Credential, CREDENTIAL_FIELDS (+32 more)

### Community 56 - "dbWithUser"
Cohesion: 0.33
Nodes (11): Stable slug and /go redirect, Slugify(), dbWithUser(), DB, T, TestCreateToolFallsBackWhenTheNameHasNoSlug(), TestGetToolBySlug(), TestSlugify() (+3 more)

### Community 57 - "DB"
Cohesion: 0.23
Nodes (5): NullString, DB, Time, parseNullTime(), HostCommand

### Community 58 - "NewHostSampler"
Cohesion: 0.32
Nodes (11): fakeNvidiaSMI(), T, TestSampleGPUKeepsAnIdleCard(), TestSampleGPUReadsAReportingCard(), TestSampleGPURecoversFromOneBadReading(), TestSampleGPUToleratesAFailingProbe(), TestSampleGPUWithoutABinary(), NewHostSampler() (+3 more)

### Community 59 - "devDependencies"
Cohesion: 0.12
Nodes (17): autoprefixer, @axe-core/playwright, eslint, eslint-plugin-react-hooks, @playwright/test, @types/leaflet, @typescript-eslint/eslint-plugin, @typescript-eslint/parser (+9 more)

### Community 60 - "HandlerFunc"
Cohesion: 0.24
Nodes (16): HandlerFunc, Handler, Handler, gzipResponses(), Handler, Response, T, gzipRequest() (+8 more)

### Community 61 - "shutdown_test.go"
Cohesion: 0.33
Nodes (8): Conn, T, TestEveryStopsOnCancel(), TestProbeHealth(), TestProbeHealthFailsOnNon200(), TestServeDrainsInFlightRequest(), TestServeReturnsListenError(), waitForListener()

### Community 62 - "T"
Cohesion: 0.26
Nodes (12): captureLog(), Buffer, Client, Response, T, TestClientIPForwardedForOnlyWhenProxyTrusted(), TestHealthz(), TestLoginSetsTheReeveSessionCookie() (+4 more)

### Community 63 - "Open"
Cohesion: 0.22
Nodes (5): DB, Open(), TestValidateBackupRejectsAForeignDatabaseWithoutTouchingIt(), ValidateBackup(), Tx

### Community 64 - "docker.go"
Cohesion: 0.25
Nodes (10): TestParseDockerPS(), TestParseDockerStats(), healthFromStatus(), parseBytes(), ParseDockerPS(), ParseDockerStats(), parseMemUsage(), parsePercent() (+2 more)

### Community 65 - "writeJSON"
Cohesion: 0.19
Nodes (10): Every reveal is audited, Request, ResponseWriter, app, User, writeJSON(), Request, ResponseWriter (+2 more)

### Community 66 - "putSettings"
Cohesion: 0.37
Nodes (12): getSettings(), Client, Response, T, putSettings(), TestSealedSettingRoundTrip(), TestSettingsAdminOnly(), TestSettingsDefaults() (+4 more)

### Community 67 - "DB"
Cohesion: 0.19
Nodes (5): DB, Time, scanCredentialMeta(), AccessGrant, Credential

### Community 68 - "dependencies"
Cohesion: 0.13
Nodes (15): @fontsource-variable/inter, leaflet, react, react-dom, react-router-dom, uplot, dependencies, @fontsource-variable/inter (+7 more)

### Community 69 - "app"
Cohesion: 0.14
Nodes (12): One Go binary serves UI and API, Writes must come from this origin, dynamicRequest(), Handler, Request, ResponseWriter, app, User (+4 more)

### Community 70 - ".handleIngest"
Cohesion: 0.12
Nodes (15): Host enrollment token, DCO sign-off, no CLA, Binds every interface by default, Docker NAT sits in front of ufw, Host networking for mDNS, SSH push install, Catalog of services, Host and service controls (+7 more)

### Community 71 - "Push"
Cohesion: 0.08
Nodes (32): TestParseSystemctl(), ParseSystemctl(), Mutex, controller, Fixed action allowlist, no shell, Collections replace free-text category, Single error envelope, Ingest push contract (+24 more)

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
Cohesion: 0.17
Nodes (9): auditLink(), auditWhere(), DB, Time, NewID(), AuditChainResult, AuditFilter, GrantAudit (+1 more)

### Community 76 - "newCLIEnv"
Cohesion: 0.11
Nodes (34): server users CLI, Request, ResponseWriter, app, User, cliEnv, HashPassword(), HashToken() (+26 more)

### Community 77 - "test_build_config.py"
Cohesion: 0.23
Nodes (8): parametrize, dry_run(), job_commands(), test_actions_are_pinned_to_a_commit(), test_npm_target_installs_its_own_deps(), test_signing_key_is_never_workflow_or_job_scoped(), test_workflow_jobs_install_web_deps_before_running_npm_scripts(), test_workflows_grant_no_blanket_permissions()

### Community 78 - "ParseCrontab"
Cohesion: 0.24
Nodes (12): isEnvAssignment(), ParseCrontab(), basename(), maskInline(), namesSecret(), redactSecrets(), T, TestParseCmdlineRedacts() (+4 more)

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
Cohesion: 0.39
Nodes (11): config, newPusher(), T, seedBuffer(), TestFlushBufferDropsAPermanentlyRejectedBody(), TestFlushBufferKeepsEverythingOn401(), TestFlushBufferKeepsEverythingOnServerError(), TestMalformedAckIsNotContact() (+3 more)

### Community 83 - "ParseDiskStats"
Cohesion: 0.47
Nodes (8): T, TestHostSamplerReportsDiskIO(), TestParseDiskStatsCountsTheWholeDiskOnce(), TestParseDiskStatsKeepsNVMeWholeDisks(), TestParseDiskStatsSkipsVirtualDevices(), TestParseDiskStatsSumsSeparateDisks(), TestParseDiskStatsToleratesJunk(), ParseDiskStats()

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

### Community 88 - "ThresholdSet"
Cohesion: 0.31
Nodes (4): DB, thresholdKey(), Threshold, ThresholdSet

### Community 89 - "TestPublicHostsNoAuth"
Cohesion: 0.53
Nodes (5): bearer(), T, TestAPIErrorEnvelope(), TestPublicEndpointsNoAuthExcludeRestricted(), TestPublicHostsNoAuth()

### Community 90 - "writeError"
Cohesion: 0.28
Nodes (8): Request, ResponseWriter, app, groupRole(), writeError(), Request, ResponseWriter, app

### Community 91 - ".handleGeocode"
Cohesion: 0.31
Nodes (8): Request, ResponseWriter, app, photonHits(), photonLabel(), geocodeHit, photonFeature, photonResponse

### Community 92 - "rbac.go"
Cohesion: 0.29
Nodes (9): Session cookie auth, ctxKey, Single-tenant, admin sees everything, Context, Handler, ResponseWriter, RequireAdmin(), RequireAuth() (+1 more)

### Community 93 - ".handleListPrincipals"
Cohesion: 0.28
Nodes (7): Request, ResponseWriter, app, localPart(), principalGroup, principalsView, principalUser

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
Cohesion: 0.08
Nodes (35): Light-only theme, Four-radius vocabulary, Two colors do two jobs, Blocking theme-boot script, autoAxisSize(), Chart(), Series, LeafletMap() (+27 more)

### Community 99 - ".handleRestoreUpload"
Cohesion: 0.46
Nodes (4): Request, ResponseWriter, app, StagedRestorePath()

### Community 100 - "TestServerInfoAdminOnly"
Cohesion: 0.67
Nodes (3): T, TestServerInfoAdminOnly(), TestUpdateUserRoleAndActive()

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

### Community 110 - "effectiveThreshold"
Cohesion: 0.62
Nodes (6): effectiveThreshold(), T, TestHostThresholdsAdminGetSet(), TestHostThresholdsReset(), TestHostThresholdsUnknownHost(), TestThresholdsAdminGetSet()

### Community 112 - "contracts_test.go"
Cohesion: 0.53
Nodes (5): T, TestPushAckRoundTrip(), TestPushCarriesAutoUpdateVeto(), TestPushRoundTrip(), TestPushTooLargeNamesTheOffendingSection()

### Community 113 - "pusher"
Cohesion: 0.17
Nodes (7): Client, permanentReject(), randSuffix(), pusher, statusError, app, Time

### Community 114 - "webhooks_test.go"
Cohesion: 0.53
Nodes (5): T, TestChannelAcceptsSeverity(), TestChannelColumnsRoundTrip(), TestChannelDefaults(), TestGlobalChannelsFilterBySeverity()

### Community 115 - "package.json"
Cohesion: 0.33
Nodes (5): license, name, private, type, version

### Community 118 - "render"
Cohesion: 0.67
Nodes (3): Image, main(), render()

### Community 129 - "mustVerify"
Cohesion: 0.45
Nodes (10): chainedDB(), DB, T, mustVerify(), TestAuditChainCatchesADeletedRow(), TestAuditChainCatchesAnEditedRow(), TestAuditChainVerifiesWhenUntouched(), TestAuditChainWithoutAKeyReportsItCannotCheck() (+2 more)

### Community 130 - "confirmText.ts"
Cohesion: 0.48
Nodes (5): Confirmation, NEEDS_CONFIRM, needsConfirm(), rowConfirmation(), RowControls()

### Community 131 - "testServer"
Cohesion: 0.28
Nodes (13): Client, T, groupsFor(), newGroup(), TestMembershipSpansGroupsAndKeepsRoles(), TestModeratorCannotReachAccounts(), TestModeratorCannotStandThemselvesDown(), TestModeratorManagesOnlyTheirOwnGroup() (+5 more)

### Community 166 - "TestContainerStatsNameFallback"
Cohesion: 0.67
Nodes (3): T, TestContainerStatsNameFallback(), TestContainerStatsRoundTrip()

### Community 167 - "compressWriter"
Cohesion: 0.31
Nodes (4): compressible(), ResponseWriter, Writer, compressWriter

## Ambiguous Edges - Review These
- `LAN-only, no external backend` → `DCO sign-off, no CLA`  [AMBIGUOUS]
  CONTRIBUTING.md · relation: conceptually_related_to
- `Light-only theme` → `Blocking theme-boot script`  [AMBIGUOUS]
  DESIGN.md · relation: conceptually_related_to

## Knowledge Gaps
- **160 isolated node(s):** `dockerPSLine`, `dockerStatsLine`, `config`, `DiskUsage`, `ProcessSample` (+155 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **25 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **What is the exact relationship between `LAN-only, no external backend` and `DCO sign-off, no CLA`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **What is the exact relationship between `Light-only theme` and `Blocking theme-boot script`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **Why does `Public portal front door` connect `.handleIngest` to `Portal.tsx`?**
  _High betweenness centrality (0.295) - this node is a cross-community bridge._
- **Why does `New()` connect `New` to `publishAgent`, `mustVerify`, `signing.go`, `Install`, `app`, `Push`, `google_test.go`, `Send`, `.uiHandler`, `decodeJSON`, `control_test.go`, `app`, `newCLIEnv`, `pusher`, `newPusher`, `.handleUploadAvatar`, `T`, `Open`?**
  _High betweenness centrality (0.229) - this node is a cross-community bridge._
- **Are the 233 inferred relationships involving `newTestServer()` (e.g. with `TestAuditListsAndFilters()` and `TestServerInfoAdminOnly()`) actually correct?**
  _`newTestServer()` has 233 INFERRED edges - model-reasoned connections that need verification._
- **Are the 117 inferred relationships involving `signup()` (e.g. with `TestAuditListsAndFilters()` and `TestServerInfoAdminOnly()`) actually correct?**
  _`signup()` has 117 INFERRED edges - model-reasoned connections that need verification._
- **Are the 124 inferred relationships involving `writeError()` (e.g. with `validPrincipalType()` and `.applyCollectionIDs()`) actually correct?**
  _`writeError()` has 124 INFERRED edges - model-reasoned connections that need verification._