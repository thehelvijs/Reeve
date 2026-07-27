# Graph Report - Reeve  (2026-07-28)

## Corpus Check
- 298 files · ~186,694 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 2723 nodes · 6995 edges · 166 communities (140 shown, 26 thin omitted)
- Extraction: 81% EXTRACTED · 19% INFERRED · 0% AMBIGUOUS · INFERRED: 1315 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `e93fb1ee`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- publishAgent
- release.py
- HostInventory.tsx
- signing.go
- Install
- contracts.go
- HostMetrics.tsx
- DB
- newCLIEnv
- signup
- google_test.go
- adminClient
- Hosts.tsx
- app
- controllableHost
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
- .evaluateHostThresholds
- samplePush
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
- Push
- Send
- countRows
- decodeJSON
- host_handlers_test.go
- DB
- AdminAlerts.tsx
- zzz-visual.spec.ts
- .Sample
- .handlePutSettings
- writeError
- DB
- NewID
- api.ts
- app
- DB
- .handleLogin
- devDependencies
- HandlerFunc
- TestLoadIngest
- T
- Open
- users_cli.go
- app
- testServer
- DB
- dependencies
- app
- mustVerify
- .handleRestoreUpload
- .handleCreateCommand
- .loadEndpoint
- .handleCreateCredential
- DB
- collect_test.go
- test_build_config.py
- ScanLogErrors
- downloadBackup
- lookupMDNS
- SSH push install
- newPusher
- app
- newDownloadApp
- .handleCreateWebhook
- .handleGetHostThresholds
- DB
- DB
- @axe-core/playwright
- ParseMounts
- replaceHostProcesses
- rbac.go
- writeJSON
- AccessRequest
- processes_test.go
- DB
- .checkSchemaDrift
- Button
- .handleTestEmail
- AdminSettings.tsx
- mustUser
- zz-agent-updates.spec.ts
- RevealModal.tsx
- TestHostEventsEndpoint
- live_ssh_install_test.sh
- .uiHandler
- ssh_install_handlers_test.go
- scripts
- pusher
- docker.go
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
- replaceHostDisks
- group_handlers.go
- tool_visibility_handlers.go
- .handleSetHostAutoUpdate
- @types/react
- confirmText.ts
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
- eslint
- NewHostSampler
- @typescript-eslint/eslint-plugin
- postcss
- @typescript-eslint/parser

## God Nodes (most connected - your core abstractions)
1. `newTestServer()` - 226 edges
2. `signup()` - 129 edges
3. `writeError()` - 122 edges
4. `adminClient()` - 84 edges
5. `openTemp()` - 80 edges
6. `writeJSON()` - 70 edges
7. `New()` - 53 edges
8. `FromContext()` - 51 edges
9. `testServer` - 43 edges
10. `createTool()` - 41 edges

## Surprising Connections (you probably didn't know these)
- `Checksum decides currency, not version` --references--> `signedVersion()`  [INFERRED]
  deploy/README.md → agent/update.go
- `server users CLI` --references--> `VerifyPassword()`  [INFERRED]
  deploy/README.md → server/internal/auth/auth.go
- `Updates only move forward` --rationale_for--> `signedVersion()`  [EXTRACTED]
  SECURITY.md → agent/update.go
- `Push monitoring` --references--> `Push`  [EXTRACTED]
  README.md → contracts/contracts.go
- `The agent distrusts its own server` --rationale_for--> `CommandTargetKind`  [INFERRED]
  SECURITY.md → contracts/contracts.go

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Agent trust boundary** — security_signed_releases, security_downgrade_protection, security_agent_local_veto, security_agent_target_validation, security_agent_runs_as_root [EXTRACTED 0.95]
- **Fleet update rollout** — contracts_api_update_state, contracts_api_rollout_pause, contracts_api_forced_slot, deploy_readme_checksum_not_version, security_agent_local_veto [EXTRACTED 0.95]
- **Not leaking what you cannot see** — contracts_api_404_not_403, contracts_api_visibility, contracts_api_slug, security_single_tenant [INFERRED 0.85]

## Communities (166 total, 26 thin omitted)

### Community 0 - "publishAgent"
Cohesion: 0.08
Nodes (50): Forced update sits outside the rollout, The host's local veto is absolute, effectiveAutoUpdate(), Duration, Host, app, Time, fetchRollup() (+42 more)

### Community 1 - "release.py"
Cohesion: 0.05
Nodes (49): CI gate job, CI grants contents:read only, Rolling edge prerelease, Signing key scoped to steps, not the workflow, Tag-triggered versioned release, web-build and server-assets precede build, make gate / web-check / pytest before commit, Two-second pre-commit gate (+41 more)

### Community 2 - "HostInventory.tsx"
Cohesion: 0.10
Nodes (33): AccessRequest, EmptyState(), EventHistory(), ACTION_LABEL, HostControls(), STATUS_TONE, unavailableReason(), Card() (+25 more)

### Community 3 - "signing.go"
Cohesion: 0.08
Nodes (58): Actions pinned by commit, not tag, releasePublicKey(), fetchBody(), config, Client, isSHA256Hex(), needsUpdate(), olderThan() (+50 more)

### Community 4 - "Install"
Cohesion: 0.10
Nodes (45): Builder, Channel, ClientConfig, clientConfig(), connect(), dial(), dialTCP(), envAssignments() (+37 more)

### Community 5 - "contracts.go"
Cohesion: 0.06
Nodes (41): argvFor(), Mutex, newController(), fakeRunner(), T, TestArgvForEveryAction(), TestArgvForRefusesADangerousTarget(), TestArgvForRejectsAnUnknownAction() (+33 more)

### Community 6 - "HostMetrics.tsx"
Cohesion: 0.06
Nodes (45): Colorblind-validated chart palette, Light-only theme, Four-radius vocabulary, Two colors do two jobs, uplot, Blocking theme-boot script, uplot, Chart() (+37 more)

### Community 7 - "DB"
Cohesion: 0.06
Nodes (29): Stable slug and /go redirect, DCO sign-off, no CLA, Binds every interface by default, Docker NAT sits in front of ufw, Host networking for mDNS, Catalog of services, LAN-only, no external backend, Push monitoring (+21 more)

### Community 8 - "newCLIEnv"
Cohesion: 0.10
Nodes (36): server users CLI, cliEnv, Request, ResponseWriter, app, hashResetToken(), newResetToken(), reachableFromOtherHosts() (+28 more)

### Community 9 - "signup"
Cohesion: 0.11
Nodes (38): T, TestServerInfoAdminOnly(), TestUpdateUserRoleAndActive(), fromIP(), Client, Response, T, login() (+30 more)

### Community 10 - "google_test.go"
Cohesion: 0.09
Nodes (27): Config, Profile, Config, Request, ResponseWriter, app, User, DomainAllowed() (+19 more)

### Community 11 - "adminClient"
Cohesion: 0.15
Nodes (42): TestAuditListsAndFilters(), createCred(), Client, T, newHost(), revealSecret(), TestAccessRequestOnAnUnknownHost(), TestAdminCanRevealAndAudited() (+34 more)

### Community 12 - "Hosts.tsx"
Cohesion: 0.11
Nodes (17): AgentUpdateRollup, AutoUpdatePolicy, UpdateState, POLICY_LABEL, showsVersionPill(), UPDATE_LABEL, UPDATE_TONE, hostRowActions() (+9 more)

### Community 13 - "app"
Cohesion: 0.07
Nodes (40): clock, Throttle, throttleEntry, Int64, Once, Login throttle that cannot be held shut, REEVE_TRUST_PROXY is opt-in, DB (+32 more)

### Community 14 - "controllableHost"
Cohesion: 0.45
Nodes (13): controllableHost(), Client, T, queue(), TestAResultFromTheWrongHostIsIgnored(), TestCommandsAreAdminOnly(), TestQueueAndDeliverACommand(), TestQueuePowerActionsNeedNoTarget() (+5 more)

### Community 15 - "Dashboard.tsx"
Cohesion: 0.09
Nodes (27): Say what the state means, AlertEvent, ToolStatus, DiskList(), MetricBar(), ServiceDots(), ListSkeleton(), Skeleton() (+19 more)

### Community 16 - "Portal.tsx"
Cohesion: 0.08
Nodes (34): Test every trigger of a modal, CollectionRef, endpointString(), goURL(), Host, HostInventory, InventoryItem, Tool (+26 more)

### Community 17 - "schema.sql"
Cohesion: 0.09
Nodes (36): One schema file, re-executed on open, No protocol version to negotiate, Everything but secrets is plaintext, access_requests, alert_events, alert_state, alert_thresholds, collection_editors (+28 more)

### Community 18 - "run"
Cohesion: 0.10
Nodes (23): compressible(), ResponseWriter, Writer, compressWriter, Conn, envOr(), every(), Context (+15 more)

### Community 19 - "Principal"
Cohesion: 0.17
Nodes (18): Principal, CollectionRef, assetURL(), agentMonitored(), checkEndpointFields(), Host, Request, ResponseWriter (+10 more)

### Community 20 - "DB"
Cohesion: 0.10
Nodes (6): collectionVisibleArgs(), DB, Time, VisibilityGrant, scanCollection(), Collection

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
Cohesion: 0.20
Nodes (23): newTestServer(), T, TestHostMetricsEndpointCarriesDisks(), TestHostMetricsEndpointContainers(), TestHostMetricsEndpointDisksEmptyForSilentHost(), TestHostProcessUsageEndpoint(), TestHostProcessUsageUnknownHost(), TestHostUptimeEndpoint() (+15 more)

### Community 26 - "procs_test.go"
Cohesion: 0.17
Nodes (24): ProcessSample, Time, NewProcSampler(), ParseCmdline(), ParsePasswd(), ParseProcStat(), ParseProcStatus(), T (+16 more)

### Community 27 - "gather"
Cohesion: 0.20
Nodes (20): durationArg(), gather(), gatherCron(), gatherDockerLogErrors(), gatherLogErrors(), config, Duration, Time (+12 more)

### Community 28 - ".evaluateHostThresholds"
Cohesion: 0.15
Nodes (14): fmtRate(), Duration, Host, app, Time, Webhook, hostMetricValues(), severityOrError() (+6 more)

### Community 29 - "samplePush"
Cohesion: 0.18
Nodes (23): TestUpdateNowRefusesAVetoedHost(), bearer(), T, TestAPIErrorEnvelope(), TestPublicEndpointsNoAuthExcludeRestricted(), TestPublicHostsNoAuth(), enrollHost(), Client (+15 more)

### Community 30 - "newUser"
Cohesion: 0.27
Nodes (16): containsKey(), containsName(), Client, T, hasKey(), hasName(), jsonString(), newUser() (+8 more)

### Community 31 - "app"
Cohesion: 0.18
Nodes (10): AccessRequest, Request, ResponseWriter, app, validPrincipalType(), accessGrantView, requestView, Request (+2 more)

### Community 32 - "Layout.tsx"
Cohesion: 0.12
Nodes (13): adminNav, Layout(), primaryNav, IconName, paths, Wordmark(), cache, Entry (+5 more)

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
Cohesion: 0.19
Nodes (21): Client, Response, T, pngHeader(), TestEveryImageSlot(), TestHostIconVisibility(), TestToolIconVisibilityAndLifecycle(), uploadIcon() (+13 more)

### Community 37 - "store/agent_update_test.go"
Cohesion: 0.20
Nodes (22): DB, T, Time, pacedSlot(), TestApplyPushRecordsVeto(), TestApplyPushWithAVetoReleasesTheSlot(), TestClearStalledUpdatesSweepsIneligibleHostsToo(), TestFleetDefaultOffMakesADefaultPolicyHostIneligible() (+14 more)

### Community 38 - "seedHost"
Cohesion: 0.18
Nodes (21): T, TestApplyPushStoresDisks(), TestDeleteHostMetricsDropsDisks(), TestHostDisksNilStoresEmptyList(), TestHostDisksRoundTripAndReplace(), TestLatestHostDisksUnknownHost(), T, TestApplyPushKeepsLastKnownChecksum() (+13 more)

### Community 39 - "New"
Cohesion: 0.21
Nodes (19): AEAD, Cipher, Compose refuses to start without the master key, Backup carries ciphertext, not the key, Credentials encrypted before they touch disk, Master-key custody, New(), NewFromEnv() (+11 more)

### Community 40 - "runOnce"
Cohesion: 0.22
Nodes (19): commandReport(), config, Duration, Time, loadConfig(), main(), runOnce(), runSelfUpdate() (+11 more)

### Community 41 - "compilerOptions"
Cohesion: 0.09
Nodes (21): DOM, DOM.Iterable, ES2021, src, compilerOptions, allowImportingTsExtensions, isolatedModules, jsx (+13 more)

### Community 42 - "Push"
Cohesion: 0.11
Nodes (19): ParseSystemctl(), ContainerState, HostMetrics, ProcessSample, Time, CronState, LogEvent, Push (+11 more)

### Community 43 - "Send"
Cohesion: 0.21
Nodes (18): Config, Message, stubSMTP, BuildMessage(), Time, hasCRLF(), Send(), atoi() (+10 more)

### Community 44 - "countRows"
Cohesion: 0.35
Nodes (13): countRows(), enrollHostWin(), Client, T, TestAgentOfflineAlert(), TestDispatchSendsAndRetries(), TestDownAlertDebounceFireResolve(), TestFlapSuppressed() (+5 more)

### Community 45 - "decodeJSON"
Cohesion: 0.16
Nodes (10): errNoAgentBuild, errUnsupportedArch, decodeJSON(), Request, ResponseWriter, app, Request, ResponseWriter (+2 more)

### Community 46 - "host_handlers_test.go"
Cohesion: 0.33
Nodes (10): T, TestAgentCommandsFallBackToRequestHost(), TestAgentInstallCommandShape(), TestClearHostMetricsAdminOnly(), TestDeleteHostLeavesLinkedToolsListable(), TestDeleteHostTakesItsTelemetryWithIt(), TestListHostsIncludesLatestMetric(), TestNonAdminCannotSeeWhatAHostRuns() (+2 more)

### Community 47 - "DB"
Cohesion: 0.20
Nodes (7): channelAccepts(), filterBySeverity(), DB, Time, severityRank(), Delivery, Webhook

### Community 48 - "AdminAlerts.tsx"
Cohesion: 0.17
Nodes (17): Delivery, ThresholdMetric, ThresholdsPayload, EMPTY_ROW, METRIC_KEYS, METRICS, ThresholdRow, ThresholdRows (+9 more)

### Community 49 - "zzz-visual.spec.ts"
Cohesion: 0.10
Nodes (8): CI Playwright e2e job, Frontend changes are reviewed in a browser, axe against WCAG 2 A/AA, Manual screenshot sweep after any frontend change, Pixel baselines for every page, TAGS, THEMES, THEMES

### Community 50 - ".Sample"
Cohesion: 0.19
Nodes (16): TestParseCPUStatAndPercent(), TestParseUptimeAndNetDev(), diskUsage(), DiskUsage, HostMetrics, sampleDisks(), sampleGPU(), CPUPercent() (+8 more)

### Community 51 - ".handlePutSettings"
Cohesion: 0.15
Nodes (14): Agent update state machine, Checksum decides currency, not version, Retention, agentUpdateView, googleAuthView, googleInput, retentionView, Request (+6 more)

### Community 52 - "writeError"
Cohesion: 0.16
Nodes (13): ReadCloser, Request, ResponseWriter, app, writeError(), Request, ResponseWriter, readImageUpload() (+5 more)

### Community 53 - "DB"
Cohesion: 0.21
Nodes (8): Rows, DB, Time, scanAlertEvents(), Time, parseNullableTime(), AlertEvent, AlertState

### Community 54 - "NewID"
Cohesion: 0.13
Nodes (4): DB, Time, NewID(), Host

### Community 55 - "api.ts"
Cohesion: 0.07
Nodes (36): AgentUpdateSettings, api, ChannelKind, Collection, CollectionDetail, CommandStatus, Credential, CREDENTIAL_FIELDS (+28 more)

### Community 56 - "app"
Cohesion: 0.28
Nodes (11): alwaysEditable(), Request, ResponseWriter, app, hostCanSee(), hostTarget(), toolCanEdit(), toolCanSee() (+3 more)

### Community 57 - "DB"
Cohesion: 0.22
Nodes (10): ContainerSample, HostMetrics, DB, Time, insertContainerStats(), insertHostMetric(), intCols(), scanLatestMetric() (+2 more)

### Community 58 - ".handleLogin"
Cohesion: 0.40
Nodes (4): Request, ResponseWriter, app, User

### Community 59 - "devDependencies"
Cohesion: 0.12
Nodes (17): autoprefixer, eslint-plugin-react-hooks, @playwright/test, tailwindcss, @types/leaflet, @types/react-dom, typescript, vitest (+9 more)

### Community 60 - "HandlerFunc"
Cohesion: 0.26
Nodes (15): HandlerFunc, Handler, Handler, gzipResponses(), Handler, Response, T, gzipRequest() (+7 more)

### Community 61 - "TestLoadIngest"
Cohesion: 0.67
Nodes (3): T, realisticPush(), TestLoadIngest()

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

### Community 66 - "testServer"
Cohesion: 0.28
Nodes (15): app, Server, getSettings(), Client, Response, T, putSettings(), TestSealedSettingRoundTrip() (+7 more)

### Community 67 - "DB"
Cohesion: 0.18
Nodes (5): DB, Time, scanCredentialMeta(), AccessGrant, Credential

### Community 68 - "dependencies"
Cohesion: 0.15
Nodes (13): @fontsource-variable/inter, leaflet, react, react-dom, react-router-dom, dependencies, @fontsource-variable/inter, leaflet (+5 more)

### Community 69 - "app"
Cohesion: 0.23
Nodes (7): Writes must come from this origin, Handler, Request, app, User, User, PrincipalFromUser()

### Community 70 - "mustVerify"
Cohesion: 0.45
Nodes (10): chainedDB(), DB, T, mustVerify(), TestAuditChainCatchesADeletedRow(), TestAuditChainCatchesAnEditedRow(), TestAuditChainVerifiesWhenUntouched(), TestAuditChainWithoutAKeyReportsItCannotCheck() (+2 more)

### Community 71 - ".handleRestoreUpload"
Cohesion: 0.46
Nodes (4): Request, ResponseWriter, app, StagedRestorePath()

### Community 72 - ".handleCreateCommand"
Cohesion: 0.29
Nodes (6): HostCommand, Request, ResponseWriter, app, commandInput, commandView

### Community 73 - ".loadEndpoint"
Cohesion: 0.27
Nodes (9): 404 instead of 403 for invisible records, Follow-the-host endpoint resolution, Host, Request, ResponseWriter, app, Tool, resolveEndpoint() (+1 more)

### Community 74 - ".handleCreateCredential"
Cohesion: 0.33
Nodes (6): Credential, credentialAAD(), Host, Request, ResponseWriter, app

### Community 75 - "DB"
Cohesion: 0.22
Nodes (8): auditLink(), auditWhere(), DB, Time, AuditChainResult, AuditFilter, GrantAudit, RevealAudit

### Community 76 - "collect_test.go"
Cohesion: 0.25
Nodes (12): T, TestParseCrontabSystemForm(), TestParseCrontabUserForm(), TestParseDockerPS(), TestParseDockerStats(), TestParseLoadAvg(), TestParseMemInfo(), TestParseNvidiaSMI() (+4 more)

### Community 77 - "test_build_config.py"
Cohesion: 0.23
Nodes (8): parametrize, dry_run(), job_commands(), test_actions_are_pinned_to_a_commit(), test_npm_target_installs_its_own_deps(), test_signing_key_is_never_workflow_or_job_scoped(), test_workflow_jobs_install_web_deps_before_running_npm_scripts(), test_workflows_grant_no_blanket_permissions()

### Community 78 - "ScanLogErrors"
Cohesion: 0.21
Nodes (13): Time, ScanLogErrors(), basename(), maskInline(), namesSecret(), redactSecrets(), T, TestParseCmdlineRedacts() (+5 more)

### Community 79 - "downloadBackup"
Cohesion: 0.41
Nodes (12): downloadBackup(), Client, Response, T, TestBackupAndRestoreAreAdminOnly(), TestBackupDownloadIsAUsableDatabase(), TestBackupKeepsCredentialsEncrypted(), TestRestoreCancelDiscardsStaged() (+4 more)

### Community 80 - "lookupMDNS"
Cohesion: 0.32
Nodes (11): Context, lookupMDNS(), mdnsAnswer(), mdnsQuery(), T, response(), TestMDNSAnswerIgnoresOtherHosts(), TestMDNSAnswerPicksTheRequestedName() (+3 more)

### Community 81 - "SSH push install"
Cohesion: 0.67
Nodes (3): Host enrollment token, SSH push install, Per-host credentials

### Community 82 - "newPusher"
Cohesion: 0.39
Nodes (11): config, newPusher(), T, seedBuffer(), TestFlushBufferDropsAPermanentlyRejectedBody(), TestFlushBufferKeepsEverythingOn401(), TestFlushBufferKeepsEverythingOnServerError(), TestMalformedAckIsNotContact() (+3 more)

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
Cohesion: 0.27
Nodes (4): blockingSlot(), DB, Time, StalledHost

### Community 88 - "DB"
Cohesion: 0.20
Nodes (3): DB, Time, Group

### Community 90 - "ParseMounts"
Cohesion: 0.32
Nodes (10): ParseMounts(), T, TestParseMountsDecodesOctalEscapes(), TestParseMountsDedupesByDevice(), TestParseMountsIgnoresGarbage(), TestParseMountsKeepsRealFilesystems(), TestUnescapeMountLeavesMalformedEscapes(), unescapeMount() (+2 more)

### Community 91 - "replaceHostProcesses"
Cohesion: 0.33
Nodes (8): accumulateProcessUsage(), ProcessSample, DB, Time, replaceHostProcesses(), execer, HostProcesses, ProcessUsage

### Community 92 - "rbac.go"
Cohesion: 0.27
Nodes (10): Session cookie auth, ctxKey, Single-tenant, admin sees everything, Context, Handler, ResponseWriter, RequireAdmin(), RequireAuth() (+2 more)

### Community 93 - "writeJSON"
Cohesion: 0.15
Nodes (13): One Go binary serves UI and API, dynamicRequest(), ResponseWriter, writeJSON(), Request, ResponseWriter, app, Request (+5 more)

### Community 94 - "AccessRequest"
Cohesion: 0.27
Nodes (5): DB, Time, placeholders(), scanRequest(), AccessRequest

### Community 95 - "processes_test.go"
Cohesion: 0.33
Nodes (10): T, TestDeleteHostMetricsDropsProcesses(), TestDeleteHostMetricsDropsProcessUsage(), TestHostProcessesNilStoresEmptyList(), TestHostProcessesRoundTripAndOverwrite(), TestLatestHostProcessesUnknownHost(), TestProcessUsageAveragesAndPeaks(), TestProcessUsageKeepsTopByMemory() (+2 more)

### Community 96 - "DB"
Cohesion: 0.36
Nodes (4): Duration, DB, Time, Retention

### Community 97 - ".checkSchemaDrift"
Cohesion: 0.25
Nodes (6): declaredColumns(), DB, T, TestDeclaredColumnsReadsTheSchema(), TestOpenAcceptsACurrentDatabase(), TestOpenRefusesADatabaseMissingAColumn()

### Community 98 - "Button"
Cohesion: 0.08
Nodes (36): Enter confirms every form, AdminUser, ApiError, SSHTarget, uploadAvatar(), User, App(), Protected() (+28 more)

### Community 99 - ".handleTestEmail"
Cohesion: 0.22
Nodes (6): Config, Request, ResponseWriter, app, smtpInput, smtpView

### Community 100 - "AdminSettings.tsx"
Cohesion: 0.09
Nodes (15): GoogleInput, Settings, SettingsInput, SMTPInput, BackLink(), AgentUpdateSection(), EmailSection(), numOrEmpty() (+7 more)

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

### Community 107 - "ssh_install_handlers_test.go"
Cohesion: 0.39
Nodes (11): createHostFor(), Client, T, sshBody(), TestAgentBinaryForUname(), TestReachableFromOtherHosts(), TestSSHEndpointsAreAdminOnlyAndScopedToAHost(), TestSSHInstallRejectsIncompleteTargets() (+3 more)

### Community 108 - "scripts"
Cohesion: 0.25
Nodes (8): scripts, build, dev, e2e, lint, preview, test, typecheck

### Community 109 - "pusher"
Cohesion: 0.21
Nodes (8): Client, permanentReject(), randSuffix(), pusher, statusError, Ingest push contract, Commands ride the push ack, PushAck

### Community 110 - "docker.go"
Cohesion: 0.33
Nodes (8): healthFromStatus(), parseBytes(), ParseDockerPS(), ParseDockerStats(), parseMemUsage(), parsePercent(), dockerPSLine, dockerStatsLine

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

### Community 130 - "replaceHostDisks"
Cohesion: 0.43
Nodes (5): DiskUsage, DB, Time, replaceHostDisks(), HostDisks

### Community 133 - ".handleSetHostAutoUpdate"
Cohesion: 0.26
Nodes (7): A stalled host pauses the whole rollout, Request, ResponseWriter, app, StalledHost, agentUpdateRollup, ValidAutoUpdatePolicy()

### Community 135 - "confirmText.ts"
Cohesion: 0.48
Nodes (5): Confirmation, NEEDS_CONFIRM, needsConfirm(), rowConfirmation(), RowControls()

### Community 163 - "NewHostSampler"
Cohesion: 0.38
Nodes (5): NewHostSampler(), T, TestHostSamplerMeasuresCPUAcrossCalls(), TestSampleHostMetricsDoesNotPanicAndReportsMemory(), HostMetrics

## Ambiguous Edges - Review These
- `LAN-only, no external backend` → `DCO sign-off, no CLA`  [AMBIGUOUS]
  CONTRIBUTING.md · relation: conceptually_related_to
- `Light-only theme` → `Blocking theme-boot script`  [AMBIGUOUS]
  DESIGN.md · relation: conceptually_related_to

## Knowledge Gaps
- **153 isolated node(s):** `dockerPSLine`, `dockerStatsLine`, `config`, `DiskUsage`, `ProcessSample` (+148 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **26 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **What is the exact relationship between `LAN-only, no external backend` and `DCO sign-off, no CLA`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **What is the exact relationship between `Light-only theme` and `Blocking theme-boot script`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **Why does `LAN-only, no external backend` connect `DB` to `SSH push install`, `contracts.go`?**
  _High betweenness centrality (0.281) - this node is a cross-community bridge._
- **Why does `Public portal front door` connect `DB` to `Portal.tsx`?**
  _High betweenness centrality (0.279) - this node is a cross-community bridge._
- **Why does `New()` connect `New` to `publishAgent`, `signing.go`, `Install`, `contracts.go`, `newCLIEnv`, `google_test.go`, `app`, `google_auth_test.go`, `Send`, `decodeJSON`, `.handlePutSettings`, `writeError`, `TestLoadIngest`, `T`, `Open`, `users_cli.go`, `app`, `mustVerify`, `newPusher`, `app`, `.handleTestEmail`, `.dispatchDue`?**
  _High betweenness centrality (0.275) - this node is a cross-community bridge._
- **Are the 220 inferred relationships involving `newTestServer()` (e.g. with `TestAuditListsAndFilters()` and `TestServerInfoAdminOnly()`) actually correct?**
  _`newTestServer()` has 220 INFERRED edges - model-reasoned connections that need verification._
- **Are the 110 inferred relationships involving `signup()` (e.g. with `TestAuditListsAndFilters()` and `TestServerInfoAdminOnly()`) actually correct?**
  _`signup()` has 110 INFERRED edges - model-reasoned connections that need verification._