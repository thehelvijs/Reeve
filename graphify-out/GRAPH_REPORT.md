# Graph Report - Reeve  (2026-07-29)

## Corpus Check
- 315 files · ~200,951 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 2854 nodes · 7328 edges · 165 communities (140 shown, 25 thin omitted)
- Extraction: 81% EXTRACTED · 19% INFERRED · 0% AMBIGUOUS · INFERRED: 1408 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `650df658`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- publishAgent
- release.py
- RevealModal.tsx
- signing.go
- Install
- Catalog.tsx
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
- testServer
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
- .handleSSHInstall
- control_test.go
- DB
- HostInventory.tsx
- zzz-visual.spec.ts
- NewHostSampler
- .evaluateHostThresholds
- controllableHost
- DB
- NewID
- api.ts
- .handleRestoreUpload
- mustVerify
- TestPublicHostsNoAuth
- devDependencies
- HandlerFunc
- DB
- T
- Open
- docker.go
- app
- AdminSettings.tsx
- DB
- dependencies
- app
- MapPicker.tsx
- Push
- .handleCreateCommand
- .loadEndpoint
- .handleCreateCredential
- DB
- New
- test_build_config.py
- ScanLogErrors
- downloadBackup
- lookupMDNS
- .handleSetHostAutoUpdate
- newPusher
- .handleLogin
- newDownloadApp
- TestWebhookProbeReportsBothOutcomes
- .handleGetHostThresholds
- DB
- TestContainerStatsNameFallback
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
- scripts
- run
- contracts_test.go
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
- newGroup
- tool_visibility_handlers.go
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
- principal_handlers.go
- @types/react-dom
- typescript
- postcss
- CommandResult
- @playwright/test
- compressWriter
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

## Communities (165 total, 25 thin omitted)

### Community 0 - "publishAgent"
Cohesion: 0.13
Nodes (27): Forced update sits outside the rollout, The host's local veto is absolute, effectiveAutoUpdate(), Duration, Host, app, Time, app (+19 more)

### Community 1 - "release.py"
Cohesion: 0.06
Nodes (47): CI gate job, CI grants contents:read only, Actions pinned by commit, not tag, Rolling edge prerelease, Signing key scoped to steps, not the workflow, Tag-triggered versioned release, web-build and server-assets precede build, Pull override for a published image (+39 more)

### Community 2 - "RevealModal.tsx"
Cohesion: 0.38
Nodes (7): RevealedCredential, RevealModal(), SecretField(), isSensitiveField(), ORDER, orderedSecretFields(), SENSITIVE

### Community 3 - "signing.go"
Cohesion: 0.08
Nodes (57): releasePublicKey(), fetchBody(), config, Client, isSHA256Hex(), needsUpdate(), olderThan(), selfArch() (+49 more)

### Community 4 - "Install"
Cohesion: 0.10
Nodes (45): Builder, Channel, ClientConfig, clientConfig(), connect(), dial(), dialTCP(), envAssignments() (+37 more)

### Community 5 - "Catalog.tsx"
Cohesion: 0.12
Nodes (18): AccessRequest, AlertEvent, HostCommand, EmptyState(), EventHistory(), ACTION_LABEL, HostControls(), STATUS_TONE (+10 more)

### Community 6 - "app"
Cohesion: 0.19
Nodes (15): alwaysEditable(), Request, ResponseWriter, app, hostCanSee(), hostTarget(), toolCanEdit(), toolCanSee() (+7 more)

### Community 7 - "HostMetrics.tsx"
Cohesion: 0.09
Nodes (31): Colorblind-validated chart palette, DiskList(), ContainerPoint, fmtPct(), HostMetrics(), HostPoint, labelFor(), latest() (+23 more)

### Community 8 - "Hosts.tsx"
Cohesion: 0.13
Nodes (12): AgentUpdateRollup, UpdateState, showsVersionPill(), hostRowActions(), RowActions, HOST_FILTERS, HostFilter, hostMatchesFilter() (+4 more)

### Community 9 - "signup"
Cohesion: 0.11
Nodes (36): T, TestServerInfoAdminOnly(), TestUpdateUserRoleAndActive(), signup(), T, TestAgentCommandsFallBackToRequestHost(), TestAgentInstallCommandShape(), TestClearHostMetricsAdminOnly() (+28 more)

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
Cohesion: 0.06
Nodes (47): clock, Throttle, throttleEntry, Int64, Once, Login throttle that cannot be held shut, REEVE_TRUST_PROXY is opt-in, DB (+39 more)

### Community 14 - "ui.tsx"
Cohesion: 0.09
Nodes (38): Enter confirms every form, AdminUser, ApiError, goURL(), Group, uploadAvatar(), User, App() (+30 more)

### Community 15 - "Dashboard.tsx"
Cohesion: 0.13
Nodes (21): Say what the state means, ToolStatus, HostNode(), nodeDot(), ServiceDots(), ListSkeleton(), Skeleton(), Eyebrow() (+13 more)

### Community 16 - "Portal.tsx"
Cohesion: 0.08
Nodes (35): Test every trigger of a modal, CollectionRef, endpointString(), Host, HostInventory, InventoryItem, Tool, AddServiceModal() (+27 more)

### Community 17 - "schema.sql"
Cohesion: 0.09
Nodes (36): One schema file, re-executed on open, No protocol version to negotiate, Everything but secrets is plaintext, access_requests, alert_events, alert_state, alert_thresholds, collection_editors (+28 more)

### Community 18 - "testServer"
Cohesion: 0.18
Nodes (26): fetchRollup(), Client, StalledHost, T, Time, hostFromList(), reportBuild(), TestAgentUpdateEndpointsAreAdminOnly() (+18 more)

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
Cohesion: 0.10
Nodes (42): fromIP(), Client, Response, T, login(), loginWith(), TestAuthStatusSetupFlag(), TestFirstUserIsAdmin() (+34 more)

### Community 26 - "procs_test.go"
Cohesion: 0.07
Nodes (52): BenchmarkHostSamplerSample(), BenchmarkParseCrontab(), BenchmarkParseMounts(), BenchmarkParsePasswd(), BenchmarkProcSamplerSample(), BenchmarkTopProcs(), benchProcs(), benchProcTree() (+44 more)

### Community 27 - "gather"
Cohesion: 0.21
Nodes (19): durationArg(), gather(), gatherDockerLogErrors(), gatherLogErrors(), config, Duration, Time, localIPFor() (+11 more)

### Community 28 - "DB"
Cohesion: 0.06
Nodes (32): Host enrollment token, Stable slug and /go redirect, Binds every interface by default, Docker NAT sits in front of ufw, Host networking for mDNS, SSH push install, Catalog of services, Host and service controls (+24 more)

### Community 29 - "samplePush"
Cohesion: 0.14
Nodes (31): TestUpdateNowRefusesAVetoedHost(), enrollHost(), Client, T, samplePush(), TestCreateHostAdminOnly(), TestIngestBadTokenRejected(), TestIngestRejectsAnOversizedPush() (+23 more)

### Community 30 - "newUser"
Cohesion: 0.27
Nodes (16): containsKey(), containsName(), Client, T, hasKey(), hasName(), jsonString(), newUser() (+8 more)

### Community 31 - "app"
Cohesion: 0.26
Nodes (7): AccessRequest, Request, ResponseWriter, app, validPrincipalType(), accessGrantView, requestView

### Community 32 - "Layout.tsx"
Cohesion: 0.08
Nodes (21): UptimeSummary, Avatar(), initials(), adminNav, Layout(), NavRecord, primaryNav, IconName (+13 more)

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

### Community 39 - "DB"
Cohesion: 0.22
Nodes (10): ContainerSample, HostMetrics, DB, Time, insertContainerStats(), insertHostMetric(), intCols(), scanLatestMetric() (+2 more)

### Community 40 - "runOnce"
Cohesion: 0.20
Nodes (20): commandReport(), config, Duration, Time, loadConfig(), main(), runOnce(), runSelfUpdate() (+12 more)

### Community 41 - "compilerOptions"
Cohesion: 0.09
Nodes (21): DOM, DOM.Iterable, ES2021, src, compilerOptions, allowImportingTsExtensions, isolatedModules, jsx (+13 more)

### Community 42 - ".ApplyPush"
Cohesion: 0.17
Nodes (11): ContainerState, DB, Time, insertLogEvents(), replaceContainerStatus(), replaceCronJobs(), replaceServiceStatus(), InventoryContainer (+3 more)

### Community 43 - "Send"
Cohesion: 0.07
Nodes (36): Config, Message, stubSMTP, Retention, agentUpdateView, googleAuthView, googleInput, BuildMessage() (+28 more)

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
Cohesion: 0.09
Nodes (32): AutoUpdatePolicy, Delivery, ThresholdMetric, ThresholdsPayload, EMPTY_ROW, METRIC_KEYS, METRICS, ThresholdRow (+24 more)

### Community 49 - "zzz-visual.spec.ts"
Cohesion: 0.09
Nodes (8): CI Playwright e2e job, axe against WCAG 2 A/AA, Manual screenshot sweep after any frontend change, Pixel baselines for every page, TAGS, THEMES, GREY_TILE, THEMES

### Community 50 - "NewHostSampler"
Cohesion: 0.05
Nodes (55): T, TestParseCPUStatAndPercent(), TestParseCrontabSystemForm(), TestParseCrontabUserForm(), TestParseDockerPS(), TestParseDockerStats(), TestParseLoadAvg(), TestParseMemInfo() (+47 more)

### Community 51 - ".evaluateHostThresholds"
Cohesion: 0.15
Nodes (14): fmtRate(), Duration, Host, app, Time, Webhook, hostMetricValues(), severityOrError() (+6 more)

### Community 52 - "controllableHost"
Cohesion: 0.45
Nodes (13): controllableHost(), Client, T, queue(), TestAResultFromTheWrongHostIsIgnored(), TestCommandsAreAdminOnly(), TestQueueAndDeliverACommand(), TestQueuePowerActionsNeedNoTarget() (+5 more)

### Community 53 - "DB"
Cohesion: 0.21
Nodes (8): Rows, DB, Time, scanAlertEvents(), Time, parseNullableTime(), AlertEvent, AlertState

### Community 54 - "NewID"
Cohesion: 0.13
Nodes (4): DB, Time, NewID(), Host

### Community 55 - "api.ts"
Cohesion: 0.05
Nodes (39): AgentUpdateSettings, api, ChannelKind, Collection, CollectionDetail, CommandStatus, Credential, CREDENTIAL_FIELDS (+31 more)

### Community 56 - ".handleRestoreUpload"
Cohesion: 0.29
Nodes (6): ReadCloser, Request, ResponseWriter, app, StagedRestorePath(), formFile()

### Community 57 - "mustVerify"
Cohesion: 0.45
Nodes (10): chainedDB(), DB, T, mustVerify(), TestAuditChainCatchesADeletedRow(), TestAuditChainCatchesAnEditedRow(), TestAuditChainVerifiesWhenUntouched(), TestAuditChainWithoutAKeyReportsItCannotCheck() (+2 more)

### Community 58 - "TestPublicHostsNoAuth"
Cohesion: 0.53
Nodes (5): bearer(), T, TestAPIErrorEnvelope(), TestPublicEndpointsNoAuthExcludeRestricted(), TestPublicHostsNoAuth()

### Community 59 - "devDependencies"
Cohesion: 0.12
Nodes (17): autoprefixer, @axe-core/playwright, eslint, eslint-plugin-react-hooks, @types/leaflet, @typescript-eslint/eslint-plugin, @typescript-eslint/parser, vitest (+9 more)

### Community 60 - "HandlerFunc"
Cohesion: 0.26
Nodes (15): HandlerFunc, Handler, Handler, gzipResponses(), Handler, Response, T, gzipRequest() (+7 more)

### Community 61 - "DB"
Cohesion: 0.16
Nodes (4): DB, Time, Group, GroupMember

### Community 62 - "T"
Cohesion: 0.23
Nodes (14): captureLog(), Buffer, Client, Response, T, newTestServerKey(), TestClientIPForwardedForOnlyWhenProxyTrusted(), TestHealthz() (+6 more)

### Community 63 - "Open"
Cohesion: 0.22
Nodes (5): DB, Open(), TestValidateBackupRejectsAForeignDatabaseWithoutTouchingIt(), ValidateBackup(), Tx

### Community 64 - "docker.go"
Cohesion: 0.33
Nodes (8): healthFromStatus(), parseBytes(), ParseDockerPS(), ParseDockerStats(), parseMemUsage(), parsePercent(), dockerPSLine, dockerStatsLine

### Community 65 - "app"
Cohesion: 0.23
Nodes (7): Every reveal is audited, Request, ResponseWriter, app, User, adminUserView, inviteView

### Community 66 - "AdminSettings.tsx"
Cohesion: 0.16
Nodes (10): GoogleInput, Settings, SettingsInput, SMTPInput, AgentUpdateSection(), EmailSection(), numOrEmpty(), RETENTION_CHOICES (+2 more)

### Community 67 - "DB"
Cohesion: 0.18
Nodes (5): DB, Time, scanCredentialMeta(), AccessGrant, Credential

### Community 68 - "dependencies"
Cohesion: 0.13
Nodes (15): @fontsource-variable/inter, leaflet, react, react-dom, react-router-dom, uplot, dependencies, @fontsource-variable/inter (+7 more)

### Community 69 - "app"
Cohesion: 0.29
Nodes (5): Writes must come from this origin, dynamicRequest(), Handler, app, User

### Community 70 - "MapPicker.tsx"
Cohesion: 0.27
Nodes (12): MapPicker(), Saved, cities, City, cityMatches(), locationKey(), lookupAddress(), mergePlaces() (+4 more)

### Community 71 - "Push"
Cohesion: 0.18
Nodes (12): isEnvAssignment(), ParseCrontab(), gatherCron(), HostMetrics, ProcessSample, Time, CronState, LogEvent (+4 more)

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

### Community 76 - "New"
Cohesion: 0.06
Nodes (71): AEAD, Cipher, Compose refuses to start without the master key, Backup carries ciphertext, not the key, server users CLI, Credentials encrypted before they touch disk, Master-key custody, cliEnv (+63 more)

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

### Community 81 - ".handleSetHostAutoUpdate"
Cohesion: 0.29
Nodes (6): A stalled host pauses the whole rollout, Request, ResponseWriter, app, StalledHost, agentUpdateRollup

### Community 82 - "newPusher"
Cohesion: 0.18
Nodes (16): Client, config, newPusher(), permanentReject(), randSuffix(), T, seedBuffer(), TestFlushBufferDropsAPermanentlyRejectedBody() (+8 more)

### Community 83 - ".handleLogin"
Cohesion: 0.21
Nodes (10): Request, ResponseWriter, app, User, User, HashToken(), NewAgentToken(), NewSessionToken() (+2 more)

### Community 84 - "newDownloadApp"
Cohesion: 0.36
Nodes (11): MapFS, app, T, newDownloadApp(), TestRoutesDownloadDispatch(), TestServeAgentBinaryKnownArch(), TestServeAgentBinaryUnknownArch404(), TestServeAgentChecksumMatchesBytes() (+3 more)

### Community 85 - "TestWebhookProbeReportsBothOutcomes"
Cohesion: 0.60
Nodes (5): T, mustSeal(), TestWebhookProbeIsAdminOnly(), TestWebhookProbeRejectsABadTarget(), TestWebhookProbeReportsBothOutcomes()

### Community 86 - ".handleGetHostThresholds"
Cohesion: 0.33
Nodes (5): Request, ResponseWriter, app, thresholdDTO, thresholdsPayload

### Community 87 - "DB"
Cohesion: 0.24
Nodes (5): blockingSlot(), DB, Time, ValidAutoUpdatePolicy(), StalledHost

### Community 88 - "TestContainerStatsNameFallback"
Cohesion: 0.67
Nodes (3): T, TestContainerStatsNameFallback(), TestContainerStatsRoundTrip()

### Community 89 - "contracts.go"
Cohesion: 0.12
Nodes (17): ParseSystemctl(), Collections replace free-text category, Single error envelope, Ingest push contract, Commands ride the push ack, CollectionRef, DiskUsage, DiskUsage (+9 more)

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
Cohesion: 0.11
Nodes (18): One Go binary serves UI and API, channelView, Request, ResponseWriter, writeJSON(), Request, ResponseWriter, app (+10 more)

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

### Community 98 - "theme.ts"
Cohesion: 0.11
Nodes (23): Light-only theme, Four-radius vocabulary, Two colors do two jobs, Blocking theme-boot script, autoAxisSize(), Chart(), Series, LeafletMap() (+15 more)

### Community 99 - "writeError"
Cohesion: 0.27
Nodes (7): writeError(), Request, ResponseWriter, app, Request, ResponseWriter, app

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
Cohesion: 0.29
Nodes (5): agentDistFS(), assetCacheControl(), FS, Handler, app

### Community 108 - "scripts"
Cohesion: 0.25
Nodes (8): scripts, build, dev, e2e, lint, preview, test, typecheck

### Community 109 - "run"
Cohesion: 0.25
Nodes (10): envOr(), every(), Context, Duration, Server, app, main(), run() (+2 more)

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

### Community 131 - "newGroup"
Cohesion: 0.38
Nodes (10): Client, T, groupsFor(), newGroup(), TestMembershipSpansGroupsAndKeepsRoles(), TestModeratorCannotReachAccounts(), TestModeratorCannotStandThemselvesDown(), TestModeratorManagesOnlyTheirOwnGroup() (+2 more)

### Community 135 - "putSettings"
Cohesion: 0.37
Nodes (12): getSettings(), Client, Response, T, putSettings(), TestSealedSettingRoundTrip(), TestSettingsAdminOnly(), TestSettingsDefaults() (+4 more)

### Community 144 - "shutdown_test.go"
Cohesion: 0.30
Nodes (9): Conn, probeHealth(), T, TestEveryStopsOnCancel(), TestProbeHealth(), TestProbeHealthFailsOnNon200(), TestServeDrainsInFlightRequest(), TestServeReturnsListenError() (+1 more)

### Community 162 - "principal_handlers.go"
Cohesion: 0.83
Nodes (3): principalGroup, principalsView, principalUser

### Community 166 - "CommandResult"
Cohesion: 0.40
Nodes (4): Mutex, controller, CommandResult, IsPowerAction()

### Community 168 - "compressWriter"
Cohesion: 0.31
Nodes (4): compressible(), ResponseWriter, Writer, compressWriter

## Ambiguous Edges - Review These
- `Light-only theme` → `Blocking theme-boot script`  [AMBIGUOUS]
  DESIGN.md · relation: conceptually_related_to

## Knowledge Gaps
- **158 isolated node(s):** `dockerPSLine`, `dockerStatsLine`, `config`, `DiskUsage`, `ProcessSample` (+153 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **25 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **What is the exact relationship between `Light-only theme` and `Blocking theme-boot script`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **Why does `Public portal front door` connect `DB` to `Portal.tsx`?**
  _High betweenness centrality (0.274) - this node is a cross-community bridge._
- **Why does `New()` connect `New` to `publishAgent`, `google_auth_test.go`, `signing.go`, `app`, `app`, `Install`, `Push`, `google_test.go`, `Send`, `app`, `control_test.go`, `newPusher`, `TestWebhookProbeReportsBothOutcomes`, `.handleRestoreUpload`, `mustVerify`, `decodeJSON`, `T`, `Open`?**
  _High betweenness centrality (0.230) - this node is a cross-community bridge._
- **Are the 236 inferred relationships involving `newTestServer()` (e.g. with `TestAuditListsAndFilters()` and `TestServerInfoAdminOnly()`) actually correct?**
  _`newTestServer()` has 236 INFERRED edges - model-reasoned connections that need verification._
- **Are the 118 inferred relationships involving `signup()` (e.g. with `TestAuditListsAndFilters()` and `TestServerInfoAdminOnly()`) actually correct?**
  _`signup()` has 118 INFERRED edges - model-reasoned connections that need verification._
- **Are the 125 inferred relationships involving `writeError()` (e.g. with `validPrincipalType()` and `.applyCollectionIDs()`) actually correct?**
  _`writeError()` has 125 INFERRED edges - model-reasoned connections that need verification._
- **Are the 90 inferred relationships involving `adminClient()` (e.g. with `TestAnIneligibleHostReleasesItsSlot()` and `TestUpdateNowRefusesOnlyWhenTheServerShipsNothing()`) actually correct?**
  _`adminClient()` has 90 INFERRED edges - model-reasoned connections that need verification._