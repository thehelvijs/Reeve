# Graph Report - Reeve  (2026-07-28)

## Corpus Check
- 316 files · ~200,772 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 2854 nodes · 7318 edges · 168 communities (142 shown, 26 thin omitted)
- Extraction: 81% EXTRACTED · 19% INFERRED · 0% AMBIGUOUS · INFERRED: 1407 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `636d9349`
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
- collect/metrics.go
- .evaluateHostThresholds
- .handlePutSettings
- DB
- DB
- api.ts
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
- AdminSettings.tsx
- DB
- dependencies
- app
- MapPicker.tsx
- Push
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
- .handleLogin
- newDownloadApp
- .handleCreateWebhook
- decodeJSON
- DB
- New
- TestPublicHostsNoAuth
- writeError
- .handleGeocode
- rbac.go
- writeJSON
- AccessRequest
- processes_test.go
- DB
- .checkSchemaDrift
- theme.ts
- .handleRestoreUpload
- app
- mustUser
- zz-agent-updates.spec.ts
- zzzz-location-autosave.spec.ts
- TestHostEventsEndpoint
- live_ssh_install_test.sh
- .uiHandler
- ssh_install_handlers_test.go
- scripts
- crypto_test.go
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
- agent/ must not import server/
- uninstall.sh
- Every column has a name
- mustVerify
- postWebhook
- newGroup
- tool_visibility_handlers.go
- .sendMail
- @types/react
- TestWebhookProbeReportsBothOutcomes
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
- principal_handlers.go
- @types/react-dom
- typescript
- postcss
- TestContainerStatsNameFallback
- @playwright/test

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

## Communities (168 total, 26 thin omitted)

### Community 0 - "publishAgent"
Cohesion: 0.13
Nodes (27): Forced update sits outside the rollout, The host's local veto is absolute, effectiveAutoUpdate(), Duration, Host, app, Time, app (+19 more)

### Community 1 - "release.py"
Cohesion: 0.05
Nodes (51): CI gate job, CI grants contents:read only, Actions pinned by commit, not tag, Rolling edge prerelease, Signing key scoped to steps, not the workflow, Tag-triggered versioned release, web-build and server-assets precede build, make gate / web-check / pytest before commit (+43 more)

### Community 2 - "RevealModal.tsx"
Cohesion: 0.38
Nodes (7): RevealedCredential, RevealModal(), SecretField(), isSensitiveField(), ORDER, orderedSecretFields(), SENSITIVE

### Community 3 - "signing.go"
Cohesion: 0.08
Nodes (55): releasePublicKey(), fetchBody(), config, Client, isSHA256Hex(), needsUpdate(), olderThan(), selfArch() (+47 more)

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
Cohesion: 0.13
Nodes (36): T, TestServerInfoAdminOnly(), TestUpdateUserRoleAndActive(), fromIP(), Client, Response, T, login() (+28 more)

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
Cohesion: 0.14
Nodes (28): T, TestGeocodeProxiesAndLabels(), TestGeocodeRequiresAdmin(), TestGeocodeShortQuerySkipsUpstream(), TestGeocodeUpstreamFailureIsBadGateway(), newTestServer(), T, TestHostMetricsEndpointCarriesDisks() (+20 more)

### Community 26 - "procs_test.go"
Cohesion: 0.07
Nodes (52): BenchmarkHostSamplerSample(), BenchmarkParseCrontab(), BenchmarkParseMounts(), BenchmarkParsePasswd(), BenchmarkProcSamplerSample(), BenchmarkTopProcs(), benchProcs(), benchProcTree() (+44 more)

### Community 27 - "gather"
Cohesion: 0.19
Nodes (21): durationArg(), gather(), gatherCron(), gatherDockerLogErrors(), gatherLogErrors(), config, Duration, Time (+13 more)

### Community 28 - "DB"
Cohesion: 0.06
Nodes (33): Host enrollment token, Stable slug and /go redirect, DCO sign-off, no CLA, Binds every interface by default, Docker NAT sits in front of ufw, Host networking for mDNS, SSH push install, Catalog of services (+25 more)

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
Nodes (21): UptimeSummary, Avatar(), initials(), adminNav, Layout(), NavRecord, primaryNav, IconName (+13 more)

### Community 33 - "T"
Cohesion: 0.06
Nodes (48): compressible(), ResponseWriter, Writer, compressWriter, Request, ResponseWriter, app, hashResetToken() (+40 more)

### Community 34 - "google_auth_test.go"
Cohesion: 0.21
Nodes (35): assertLoginError(), callback(), enableGoogle(), Client, Response, Server, T, noFollow() (+27 more)

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
Cohesion: 0.21
Nodes (18): Config, Message, stubSMTP, BuildMessage(), Time, hasCRLF(), Send(), atoi() (+10 more)

### Community 44 - "countRows"
Cohesion: 0.18
Nodes (22): countRows(), enrollHostWin(), Client, T, TestAgentOfflineAlert(), TestDispatchSendsAndRetries(), TestDownAlertDebounceFireResolve(), TestFlapSuppressed() (+14 more)

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
Nodes (9): CI Playwright e2e job, Frontend changes are reviewed in a browser, axe against WCAG 2 A/AA, Manual screenshot sweep after any frontend change, Pixel baselines for every page, TAGS, THEMES, GREY_TILE (+1 more)

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

### Community 55 - "api.ts"
Cohesion: 0.05
Nodes (39): AgentUpdateSettings, api, ChannelKind, Collection, CollectionDetail, CommandStatus, Credential, CREDENTIAL_FIELDS (+31 more)

### Community 56 - "users_cli.go"
Cohesion: 0.43
Nodes (16): deletionHeir(), firstArg(), DB, User, Writer, guardLastActiveAdmin(), lookup(), normalizeEmail() (+8 more)

### Community 57 - "DB"
Cohesion: 0.23
Nodes (5): NullString, DB, Time, parseNullTime(), HostCommand

### Community 58 - "NewHostSampler"
Cohesion: 0.22
Nodes (15): fakeNvidiaSMI(), T, TestSampleGPUKeepsAnIdleCard(), TestSampleGPUReadsAReportingCard(), TestSampleGPURecoversFromOneBadReading(), TestSampleGPUToleratesAFailingProbe(), TestSampleGPUWithoutABinary(), NewHostSampler() (+7 more)

### Community 59 - "devDependencies"
Cohesion: 0.12
Nodes (17): autoprefixer, @axe-core/playwright, eslint, eslint-plugin-react-hooks, @types/leaflet, @typescript-eslint/eslint-plugin, @typescript-eslint/parser, vitest (+9 more)

### Community 60 - "HandlerFunc"
Cohesion: 0.28
Nodes (14): HandlerFunc, Handler, Handler, gzipResponses(), Handler, Response, T, gzipRequest() (+6 more)

### Community 61 - "DB"
Cohesion: 0.16
Nodes (4): DB, Time, Group, GroupMember

### Community 62 - "T"
Cohesion: 0.26
Nodes (12): captureLog(), Buffer, Client, Response, T, TestClientIPForwardedForOnlyWhenProxyTrusted(), TestHealthz(), TestLoginSetsTheReeveSessionCookie() (+4 more)

### Community 63 - "Open"
Cohesion: 0.22
Nodes (5): DB, Open(), TestValidateBackupRejectsAForeignDatabaseWithoutTouchingIt(), ValidateBackup(), Tx

### Community 64 - "collect_test.go"
Cohesion: 0.11
Nodes (24): T, TestParseCrontabSystemForm(), TestParseCrontabUserForm(), TestParseDockerPS(), TestParseDockerStats(), TestParseLoadAvg(), TestParseMemInfo(), TestParseNvidiaSMI() (+16 more)

### Community 65 - "app"
Cohesion: 0.23
Nodes (7): Every reveal is audited, Request, ResponseWriter, app, User, adminUserView, inviteView

### Community 66 - "AdminSettings.tsx"
Cohesion: 0.16
Nodes (10): GoogleInput, Settings, SettingsInput, SMTPInput, AgentUpdateSection(), EmailSection(), numOrEmpty(), RETENTION_CHOICES (+2 more)

### Community 67 - "DB"
Cohesion: 0.19
Nodes (5): DB, Time, scanCredentialMeta(), AccessGrant, Credential

### Community 68 - "dependencies"
Cohesion: 0.13
Nodes (15): @fontsource-variable/inter, leaflet, react, react-dom, react-router-dom, uplot, dependencies, @fontsource-variable/inter (+7 more)

### Community 69 - "app"
Cohesion: 0.18
Nodes (10): Writes must come from this origin, Handler, Request, app, User, securityHeaders(), TestSecurityHeaders(), User (+2 more)

### Community 70 - "MapPicker.tsx"
Cohesion: 0.27
Nodes (12): MapPicker(), Saved, cities, City, cityMatches(), locationKey(), lookupAddress(), mergePlaces() (+4 more)

### Community 71 - "Push"
Cohesion: 0.09
Nodes (27): Mutex, controller, Collections replace free-text category, Single error envelope, Ingest push contract, Commands ride the push ack, CollectionRef, CommandResult (+19 more)

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
Cohesion: 0.09
Nodes (42): server users CLI, cliEnv, HashPassword(), HashToken(), NewAgentToken(), NewSessionToken(), newToken(), T (+34 more)

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

### Community 85 - ".handleCreateWebhook"
Cohesion: 0.31
Nodes (5): channelView, Request, ResponseWriter, app, Webhook

### Community 86 - "decodeJSON"
Cohesion: 0.15
Nodes (12): decodeJSON(), Request, ResponseWriter, app, Request, ResponseWriter, app, thresholdDTO (+4 more)

### Community 87 - "DB"
Cohesion: 0.24
Nodes (5): blockingSlot(), DB, Time, ValidAutoUpdatePolicy(), StalledHost

### Community 88 - "New"
Cohesion: 0.29
Nodes (8): AEAD, Cipher, Compose refuses to start without the master key, Backup carries ciphertext, not the key, Credentials encrypted before they touch disk, Master-key custody, New(), NewFromEnv()

### Community 89 - "TestPublicHostsNoAuth"
Cohesion: 0.53
Nodes (5): bearer(), T, TestAPIErrorEnvelope(), TestPublicEndpointsNoAuthExcludeRestricted(), TestPublicHostsNoAuth()

### Community 90 - "writeError"
Cohesion: 0.44
Nodes (5): Request, ResponseWriter, app, groupRole(), writeError()

### Community 91 - ".handleGeocode"
Cohesion: 0.31
Nodes (8): Request, ResponseWriter, app, photonHits(), photonLabel(), geocodeHit, photonFeature, photonResponse

### Community 92 - "rbac.go"
Cohesion: 0.29
Nodes (9): Session cookie auth, ctxKey, Single-tenant, admin sees everything, Context, Handler, ResponseWriter, RequireAdmin(), RequireAuth() (+1 more)

### Community 93 - "writeJSON"
Cohesion: 0.13
Nodes (15): One Go binary serves UI and API, dynamicRequest(), ResponseWriter, writeJSON(), Request, ResponseWriter, Request, ResponseWriter (+7 more)

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

### Community 99 - ".handleRestoreUpload"
Cohesion: 0.29
Nodes (6): ReadCloser, Request, ResponseWriter, app, StagedRestorePath(), formFile()

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

### Community 109 - "crypto_test.go"
Cohesion: 0.45
Nodes (11): T, mustKey(), testKey(), TestNewFromEnvMissingKeyFails(), TestNewFromEnvValid(), TestNewFromEnvWrongLengthFails(), TestOpenRejectsAnotherRowsAAD(), TestSealOpenRoundTrip() (+3 more)

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

### Community 129 - "mustVerify"
Cohesion: 0.45
Nodes (10): chainedDB(), DB, T, mustVerify(), TestAuditChainCatchesADeletedRow(), TestAuditChainCatchesAnEditedRow(), TestAuditChainVerifiesWhenUntouched(), TestAuditChainWithoutAKeyReportsItCannotCheck() (+2 more)

### Community 130 - "postWebhook"
Cohesion: 0.29
Nodes (9): postWebhook(), redactConfig(), T, TestDispatchFailsChannelWithNoURL(), TestDispatchMarksSentAndFailed(), TestDueDeliveriesReturnsKindConfig(), TestPostWebhookBearerToken(), TestPostWebhookSendsPayload() (+1 more)

### Community 131 - "newGroup"
Cohesion: 0.38
Nodes (10): Client, T, groupsFor(), newGroup(), TestMembershipSpansGroupsAndKeepsRoles(), TestModeratorCannotReachAccounts(), TestModeratorCannotStandThemselvesDown(), TestModeratorManagesOnlyTheirOwnGroup() (+2 more)

### Community 133 - ".sendMail"
Cohesion: 0.29
Nodes (4): Config, app, smtpInput, smtpView

### Community 135 - "TestWebhookProbeReportsBothOutcomes"
Cohesion: 0.60
Nodes (5): T, mustSeal(), TestWebhookProbeIsAdminOnly(), TestWebhookProbeRejectsABadTarget(), TestWebhookProbeReportsBothOutcomes()

### Community 162 - "principal_handlers.go"
Cohesion: 0.83
Nodes (3): principalGroup, principalsView, principalUser

### Community 166 - "TestContainerStatsNameFallback"
Cohesion: 0.67
Nodes (3): T, TestContainerStatsNameFallback(), TestContainerStatsRoundTrip()

## Ambiguous Edges - Review These
- `LAN-only, no external backend` → `DCO sign-off, no CLA`  [AMBIGUOUS]
  CONTRIBUTING.md · relation: conceptually_related_to
- `Light-only theme` → `Blocking theme-boot script`  [AMBIGUOUS]
  DESIGN.md · relation: conceptually_related_to

## Knowledge Gaps
- **160 isolated node(s):** `dockerPSLine`, `dockerStatsLine`, `config`, `DiskUsage`, `ProcessSample` (+155 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **26 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **What is the exact relationship between `LAN-only, no external backend` and `DCO sign-off, no CLA`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **What is the exact relationship between `Light-only theme` and `Blocking theme-boot script`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **Why does `Public portal front door` connect `DB` to `Portal.tsx`?**
  _High betweenness centrality (0.280) - this node is a cross-community bridge._
- **Why does `New()` connect `New` to `publishAgent`, `mustVerify`, `postWebhook`, `signing.go`, `Install`, `.sendMail`, `TestWebhookProbeReportsBothOutcomes`, `google_test.go`, `google_auth_test.go`, `Send`, `control_test.go`, `.handlePutSettings`, `users_cli.go`, `T`, `Open`, `app`, `Push`, `newCLIEnv`, `newPusher`, `decodeJSON`, `.handleRestoreUpload`, `app`, `.uiHandler`, `crypto_test.go`, `.dispatchDue`?**
  _High betweenness centrality (0.238) - this node is a cross-community bridge._
- **Are the 236 inferred relationships involving `newTestServer()` (e.g. with `TestAuditListsAndFilters()` and `TestServerInfoAdminOnly()`) actually correct?**
  _`newTestServer()` has 236 INFERRED edges - model-reasoned connections that need verification._
- **Are the 118 inferred relationships involving `signup()` (e.g. with `TestAuditListsAndFilters()` and `TestServerInfoAdminOnly()`) actually correct?**
  _`signup()` has 118 INFERRED edges - model-reasoned connections that need verification._
- **Are the 125 inferred relationships involving `writeError()` (e.g. with `validPrincipalType()` and `.applyCollectionIDs()`) actually correct?**
  _`writeError()` has 125 INFERRED edges - model-reasoned connections that need verification._