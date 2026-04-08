# Changelog

# [v1.57.0](https://github.com/agones-dev/agones/tree/v1.57.0) (2026-04-07)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.56.0...v1.57.0)

**Breaking changes**
- Upgrade supported Kubernetes versions to 1.33, 1.34, 1.35 by @Sivasankaran25 in https://github.com/agones-dev/agones/pull/4488

**Implemented enhancements**
- feat: support init sidecar containers by @nrwiersma in https://github.com/agones-dev/agones/pull/4375

**Fixed bugs**
- fix: duplicate GameServerSets not being cleaned up under a Fleet by @mnthe in https://github.com/agones-dev/agones/pull/4469
- Flaky Test: TestGameServerAllocationReturnLabels by @Sivasankaran25 in https://github.com/agones-dev/agones/pull/4435
- Fix: propagate request context on allocation by @lacroixthomas in https://github.com/agones-dev/agones/pull/4415
- Fix memory leak in fleet autoscaler webhook HTTP client by @markmandel in https://github.com/agones-dev/agones/pull/4493

**Security fixes**
- build(deps): bump minimatch in /sdks/nodejs by @dependabot[bot] in https://github.com/agones-dev/agones/pull/4463
- build(deps): bump gRPC and related dependencies by @markmandel in https://github.com/agones-dev/agones/pull/4485

**Other**
- Preparation for Release v1.57.0 by @Sivasankaran25 in https://github.com/agones-dev/agones/pull/4458
- docs: add CountsAndLists priorities documentation to scheduling page by @markmandel in https://github.com/agones-dev/agones/pull/4460
- Add agones-kotlin-sdk to list of Client SDKs by @scrayos in https://github.com/agones-dev/agones/pull/4462
- Update links from `googleforgames` to `agones-dev` by @markmandel in https://github.com/agones-dev/agones/pull/4475
- CNCF: update footers / logo / copyrights by @lacroixthomas in https://github.com/agones-dev/agones/pull/4470
- build(deps): bump github.com/cloudflare/circl from 1.6.1 to 1.6.3 in /build/scripts/example-version-checker by @dependabot[bot] in https://github.com/agones-dev/agones/pull/4459
- docs: add DCO sign-off instructions to contributing guide by @markmandel in https://github.com/agones-dev/agones/pull/4477
- fix: correct environment variable syntax in Dockerfile to fix LegacyKeyValueFormat warnings by @shihabhasan0161 in https://github.com/agones-dev/agones/pull/4484
- Fix/4447 upgrade grafana helm chart by @souben in https://github.com/agones-dev/agones/pull/4454
- docs: split feature gates table into Alpha, Beta, and Stable sections by @markmandel in https://github.com/agones-dev/agones/pull/4490
- Created performance test cluster for 1.34 by @Sivasankaran25 in https://github.com/agones-dev/agones/pull/4489
- docs: add Ubisoft KubeCon EU 2026 keynote to videos and presentations by @markmandel in https://github.com/agones-dev/agones/pull/4494
- build(deps): bump google.golang.org/grpc from 1.77.0 to 1.79.3 in /build/agones-bot by @dependabot[bot] in https://github.com/agones-dev/agones/pull/4486

**New Contributors**
- @mnthe made their first contribution in https://github.com/agones-dev/agones/pull/4469
- @shihabhasan0161 made their first contribution in https://github.com/agones-dev/agones/pull/4484
- @souben made their first contribution in https://github.com/agones-dev/agones/pull/4454

# [v1.56.0](https://github.com/agones-dev/agones/tree/v1.56.0) (2026-02-24)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.55.0...v1.56.0)

**Breaking changes**
- Feat: Move SidecarContainers feature gate to Beta by @AryanBagade in https://github.com/agones-dev/agones/pull/4364
- Fix Prometheus installation with helm chart v28.9.0. by @markmandel in https://github.com/agones-dev/agones/pull/4448

**Implemented enhancements**
- Build simple-game-server in CI for e2e tests by @IrvingMg in https://github.com/agones-dev/agones/pull/4401
- Upgrade Hugo and Docsy by @Sivasankaran25 in https://github.com/agones-dev/agones/pull/4404
- add optional agones.metrics.serviceMonitor.additonalLabels map to values.yaml in helm chart by @jessebot in https://github.com/agones-dev/agones/pull/4424
- Added dark mode to the website by @thegalacticlor in https://github.com/agones-dev/agones/pull/4439
- site: update homepage showcase and navbar styling by @markmandel in https://github.com/agones-dev/agones/pull/4438

**Fixed bugs**
- fix(sdk): add validation to SetAnnotation to prevent infinite retry loop by @indurireddy-TF in https://github.com/agones-dev/agones/pull/4417
- Fixes post_cloudbuild release by explicitly passing variables as Makefile arguments by @igooch in https://github.com/agones-dev/agones/pull/4423
- fix: Ensure the processclient have a correct timeout / fix flacky test by @lacroixthomas in https://github.com/agones-dev/agones/pull/4402
- docs: fix alert shortcode syntax to process Markdown content by @markmandel in https://github.com/agones-dev/agones/pull/4437

**Other**
- Preparation for Release v1.56.0 by @Sivasankaran25 in https://github.com/agones-dev/agones/pull/4422
- chore(sdk/nodejs): update dependencies and migrate to ESLint 9 config by @markmandel in https://github.com/agones-dev/agones/pull/4412
- chore(sdk/nodejs): migrate linting from ESLint to Biome by @smazmi in https://github.com/agones-dev/agones/pull/4426
- fix(docs): update external links in out-of-cluster dev server guide by @markmandel in https://github.com/agones-dev/agones/pull/4427
- test: standardize runtime.ParseFeatures error handling and mutex usage by @markmandel in https://github.com/agones-dev/agones/pull/4416
- Add Singularity 6 Palia architecture article to third party content by @markmandel in https://github.com/agones-dev/agones/pull/4430
- Add CEDIL slides link for Sega Sonic Rumble CEDEC talk by @markmandel in https://github.com/agones-dev/agones/pull/4432
- Add Google Doodle containerized games case study to third party content by @markmandel in https://github.com/agones-dev/agones/pull/4433
- docs: add blog entry for "Making and Scaling a Game Server in Kubernetes using Agones" by @noetarbouriech in https://github.com/agones-dev/agones/pull/4428
- Document Allocation batching by @markmandel in https://github.com/agones-dev/agones/pull/4440
- build(deps): bump github.com/go-git/go-git/v5 from 5.16.4 to 5.16.5 in /build/scripts/example-version-checker by @dependabot[bot] in https://github.com/agones-dev/agones/pull/4442
- test: fix flaky allocation metrics test server startup by @srpvpn in https://github.com/agones-dev/agones/pull/4445
- docs: convert alert shortcodes to format for markdown rendering by @markmandel in https://github.com/agones-dev/agones/pull/4446
- build: improve test-gen-api-docs failure message by @markmandel in https://github.com/agones-dev/agones/pull/4453

**New Contributors**
- @smazmi made their first contribution in https://github.com/agones-dev/agones/pull/4426
- @IrvingMg made their first contribution in https://github.com/agones-dev/agones/pull/4401
- @noetarbouriech made their first contribution in https://github.com/agones-dev/agones/pull/4428
- @jessebot made their first contribution in https://github.com/agones-dev/agones/pull/4424
- @AryanBagade made their first contribution in https://github.com/agones-dev/agones/pull/4364
- @thegalacticlor made their first contribution in https://github.com/agones-dev/agones/pull/4439
- @srpvpn made their first contribution in https://github.com/agones-dev/agones/pull/4445

# [v1.55.0](https://github.com/agones-dev/agones/tree/v1.55.0) (2026-01-13)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.54.0...v1.55.0)

**Implemented enhancements**
- feat: promote FleetAutoscaleRequestMetaData to beta by @sugar-cat7 in https://github.com/agones-dev/agones/pull/4365
- Add Wasm Fleet Autoscaler quickstart documentation by @markmandel in https://github.com/agones-dev/agones/pull/4382

**Fixed bugs**
- Fix sdkserver updatelist gameserver max items by @Sivasankaran25 in https://github.com/agones-dev/agones/pull/4326
- Fix Site Docsy Theme by @markmandel in https://github.com/agones-dev/agones/pull/4394
- fix(sdk): validate labels in SetLabel to prevent retry loops by @rajanarahul93 in https://github.com/agones-dev/agones/pull/4386

**Other**
- Release 1.54.0 (#4374) by @Sivasankaran25 in https://github.com/agones-dev/agones/pull/4376
- Feat: enable remote debugging with minikube by @lacroixthomas in https://github.com/agones-dev/agones/pull/4341
- Preparation for Release v1.55.0 by @Sivasankaran25 in https://github.com/agones-dev/agones/pull/4378
- Update caBundle Documentation. by @markmandel in https://github.com/agones-dev/agones/pull/4380
- test: deferred cleanup for GameServer in TestSuperTuxKartGameServerReady by @markmandel in https://github.com/agones-dev/agones/pull/4379
- Upgrade Golang 1.25 Versions by @Sivasankaran25 in https://github.com/agones-dev/agones/pull/4383
- Reorganize third-party content documentation and add content by @markmandel in https://github.com/agones-dev/agones/pull/4388
- Fix flaky TestControllerGameServerCount race condition by @markmandel in https://github.com/agones-dev/agones/pull/4392
- feat: refactored to use SumGameServerSets by @Ledovecek in https://github.com/agones-dev/agones/pull/4398
- test: fix flaky TestPortRangeAllocatorSyncDeleteGameServer by @markmandel in https://github.com/agones-dev/agones/pull/4399
- Refactor metrics controller tests to fix flaky behaviour by @markmandel in https://github.com/agones-dev/agones/pull/4406
- Fix(build): Upgrade Agones base container images from Debian 12 (Book… by @indurireddy-TF in https://github.com/agones-dev/agones/pull/4387
- Update third-party content documentation with new articles and podcasts by @markmandel in https://github.com/agones-dev/agones/pull/4410
- fix(rust-simple): upgrade rust toolchain to 1.92 to fix build failure by @indurireddy-TF in https://github.com/agones-dev/agones/pull/4414

**New Contributors**
- @sugar-cat7 made their first contribution in https://github.com/agones-dev/agones/pull/4365
- @Ledovecek made their first contribution in https://github.com/agones-dev/agones/pull/4398
- @rajanarahul93 made their first contribution in https://github.com/agones-dev/agones/pull/4386

# [v1.54.0](https://github.com/agones-dev/agones/tree/v1.54.0) (2025-12-02)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.53.0...v1.54.0)

**Breaking changes**
- Update supported Kubernetes versions to 1.32, 1.33, 1.34 by @Sivasankaran25 in https://github.com/agones-dev/agones/pull/4320
- [Unreal] Refactor agones component to subsystem by @GloryOfNight in https://github.com/agones-dev/agones/pull/4033
- Return GameServerAllocationUnAllocated when an game server update error occurs by @miai10 in https://github.com/agones-dev/agones/pull/4267
- feat(autoscaler)!: Remove caBundle requirement for HTTPS URLs by @markmandel in https://github.com/agones-dev/agones/pull/4332

**Implemented enhancements**
- [Unreal] Add counters support to status by @GloryOfNight in https://github.com/agones-dev/agones/pull/4333
- docs(examples): add working autoscaler-wasm example configuration by @markmandel in https://github.com/agones-dev/agones/pull/4345
- Graduate AutopilotPassthroughPort to Stable by @Sivasankaran25 in https://github.com/agones-dev/agones/pull/4335

**Fixed bugs**
- Populate Env Vars for GameServer init containers by @giornetta in https://github.com/agones-dev/agones/pull/4319
- Fix update counter to return correct values by @indurireddy-TF in https://github.com/agones-dev/agones/pull/4324
- Fix: ensure the uninstall wait to be properly done by @lacroixthomas in https://github.com/agones-dev/agones/pull/4355
- Fix race condition in PerNodeCounter by tracking processed events by @markmandel in https://github.com/agones-dev/agones/pull/4363

**Other**
- Preparation for Release v1.54.0 by @Sivasankaran25 in https://github.com/agones-dev/agones/pull/4313
- cleanup(e2e): Scale back autoscaler timeout. by @markmandel in https://github.com/agones-dev/agones/pull/4312
- Refactor FleetAutoscaler state from map to typed struct by @markmandel in https://github.com/agones-dev/agones/pull/4315
- Created performance test cluster for 1.33 by @Sivasankaran25 in https://github.com/agones-dev/agones/pull/4325
- docs: Add Wasm autoscaling documentation to FleetAutoscaler reference by @markmandel in https://github.com/agones-dev/agones/pull/4314
- Feat: add metallb on kind / minikube to run e2e locally by @lacroixthomas in https://github.com/agones-dev/agones/pull/4318
- build: upgrade MetalLB to v0.15.2 and use helm upgrade --install by @markmandel in https://github.com/agones-dev/agones/pull/4330
- test: simplify autoscaler e2e tests for minikube compatibility by @markmandel in https://github.com/agones-dev/agones/pull/4331
- cleanup(examples): Upgrade SuperTuxKart and increase timeout. by @markmandel in https://github.com/agones-dev/agones/pull/4338
- cleanup(ci): Remove 403 link for Win 10 and minikube by @markmandel in https://github.com/agones-dev/agones/pull/4349
- Remove assignees from update dependencies github issue creation by @igooch in https://github.com/agones-dev/agones/pull/4327
- test: improve TestMultiClusterAllocationFromLocal flakiness. by @markmandel in https://github.com/agones-dev/agones/pull/4350
- Cleanup on SuperTuxKart README by @markmandel in https://github.com/agones-dev/agones/pull/4344
- Exclude wasm from example image check by @igooch in https://github.com/agones-dev/agones/pull/4353
- docs: add section highlighting good first issue and help wanted labels by @markmandel in https://github.com/agones-dev/agones/pull/4362
- More fixes for SuperTuxKart example to attempt to fix flakiness. by @markmandel in https://github.com/agones-dev/agones/pull/4359
- cleanup(agones-bots): Update Agones Bot Deps. by @markmandel in https://github.com/agones-dev/agones/pull/4366
- Bumps SuperTuxKart image version by @igooch in https://github.com/agones-dev/agones/pull/4367
- feat: Bump golang.org/x/crypto to v0.45.0 by @indurireddy-TF in https://github.com/agones-dev/agones/pull/4370
- Adds the build environment image to the pre_cloudbuild pipeline by @igooch in https://github.com/agones-dev/agones/pull/4372

**New Contributors**
- @giornetta made their first contribution in https://github.com/agones-dev/agones/pull/4319
- @indurireddy-TF made their first contribution in https://github.com/agones-dev/agones/pull/4324

# [v1.53.0](https://github.com/agones-dev/agones/tree/v1.53.0) (2025-10-21)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.52.0...v1.53.0)

**Breaking changes**
- feat(autoscaling): CRDs for Wasm autoscaler policy by @markmandel in https://github.com/agones-dev/agones/pull/4281

**Implemented enhancements**
- feat: add processor proto  by @lacroixthomas in https://github.com/agones-dev/agones/pull/4227
- Feat: Add new binary processor by @lacroixthomas in https://github.com/agones-dev/agones/pull/4222
- Update Helm option for spec.strategy.type for controller and extensio… by @Sivasankaran25 in https://github.com/agones-dev/agones/pull/4244
- WasmAutoscaler feature gate and prep build image by @markmandel in https://github.com/agones-dev/agones/pull/4243
- feat: use gameservers.lists.maxItems instead of a hardcoded limit by @miai10 in https://github.com/agones-dev/agones/pull/4246
- Feat: update error fields from processor proto by @lacroixthomas in https://github.com/agones-dev/agones/pull/4266
- Wasm Autoscaler: Example by @markmandel in https://github.com/agones-dev/agones/pull/4260
- Feat: implement rust sdk (counter and list) by @lacroixthomas in https://github.com/agones-dev/agones/pull/4247
- Fleet autoscaler threads maintain state by @markmandel in https://github.com/agones-dev/agones/pull/4277
- Feat: implement processor client by @lacroixthomas in https://github.com/agones-dev/agones/pull/4265
- Add Wasm autoscaler policy support to FleetAutoscaler CRDs YAML by @markmandel in https://github.com/agones-dev/agones/pull/4298
- Implement Wasm autoscaler policy controller logic by @markmandel in https://github.com/agones-dev/agones/pull/4299
- Feat: integrate processor on allocator by @lacroixthomas in https://github.com/agones-dev/agones/pull/4302
- Feat: integrate processor on extensions by @lacroixthomas in https://github.com/agones-dev/agones/pull/4301

**Fixed bugs**
- Fix: patch flaky tests from submit-upgrade-test-cloud-build by @lacroixthomas in https://github.com/agones-dev/agones/pull/4236
- Fix: Add missing permission for helm uninstall in upgrade test cleanup by @lacroixthomas in https://github.com/agones-dev/agones/pull/4250
- Prepend sidecars to existing init containers by @markmandel in https://github.com/agones-dev/agones/pull/4278
- fix: broken websocket connection after upgrading github.com/grpc-ecosystem/grpc-gateway/v2 by @swermin in https://github.com/agones-dev/agones/pull/4270
- flakey: add resource requirements to SuperTuxKart e2e test by @markmandel in https://github.com/agones-dev/agones/pull/4280
- Fix: update link to quilkin by @lacroixthomas in https://github.com/agones-dev/agones/pull/4288
- Update Add and Remove List Value in SDK Server by @Sivasankaran25 in https://github.com/agones-dev/agones/pull/4279

**Other**
- Preparation for Release v1.52.0 by @Sivasankaran25 in https://github.com/agones-dev/agones/pull/4234
- Updates agones-bot dependencies by @igooch in https://github.com/agones-dev/agones/pull/4232
- Update all tests to use the latest Helm version by @Sivasankaran25 in https://github.com/agones-dev/agones/pull/4238
- Bump github.com/go-viper/mapstructure/v2 from 2.2.1 to 2.3.0 in /examples/crd-client by @dependabot[bot] in https://github.com/agones-dev/agones/pull/4229
- Handle missing upgrade-test-runner pod to avoid log collection errors by @0xaravindh in https://github.com/agones-dev/agones/pull/4224
- e2e: add webhook autoscaler test with fleet metadata  by @0xaravindh in https://github.com/agones-dev/agones/pull/4251
- Bump github.com/go-viper/mapstructure/v2 from 2.2.1 to 2.4.0 in /test/upgrade by @dependabot[bot] in https://github.com/agones-dev/agones/pull/4254
- Updates the upgrade test to print any fatal error messages to the job pod termination log by @igooch in https://github.com/agones-dev/agones/pull/4252
- Pause Single Cluster Upgrade work until stable. by @markmandel in https://github.com/agones-dev/agones/pull/4257
- Replace bitname/kubectl with alpine/kubectl by @markmandel in https://github.com/agones-dev/agones/pull/4268
- Upgrade Golang to 1.24.6 and update related dependencies by @0xaravindh in https://github.com/agones-dev/agones/pull/4262
- flaky: TestControllerAllocator by @markmandel in https://github.com/agones-dev/agones/pull/4269
- Release v1.52.0 by @Sivasankaran25 in https://github.com/agones-dev/agones/pull/4272
- Preparation for Release v1.53.0 by @Sivasankaran25 in https://github.com/agones-dev/agones/pull/4273
- Updates the build scripts to allow for a patch release by @igooch in https://github.com/agones-dev/agones/pull/4291
- Post Release-1.52.2 by @Sivasankaran25 in https://github.com/agones-dev/agones/pull/4296
- Add Patch Release Template by @Sivasankaran25 in https://github.com/agones-dev/agones/pull/4300
- fix(ci): Fix all the links in CI by @markmandel in https://github.com/agones-dev/agones/pull/4303
- docs: clarify FleetAutoscaler webhook response structure by @markmandel in https://github.com/agones-dev/agones/pull/4297
- Updates Rust version by @igooch in https://github.com/agones-dev/agones/pull/4306
- npm audit fix by @igooch in https://github.com/agones-dev/agones/pull/4305
- Updates release templates by @igooch in https://github.com/agones-dev/agones/pull/4307

## [v1.52.0](https://github.com/agones-dev/agones/tree/v1.52.0) (2025-09-09)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.51.0...v1.52.0)

**Implemented enhancements:**
- feat: add processor proto  by @lacroixthomas in https://github.com/agones-dev/agones/pull/4227
- Feat: Add new binary processor by @lacroixthomas in https://github.com/agones-dev/agones/pull/4222
- Update Helm option for spec.strategy.type for controller and extensio… by @Sivasankaran25 in https://github.com/agones-dev/agones/pull/4244
- WasmAutoscaler feature gate and prep build image by @markmandel in https://github.com/agones-dev/agones/pull/4243
- feat: use gameservers.lists.maxItems instead of a hardcoded limit by @miai10 in https://github.com/agones-dev/agones/pull/4246
- Feat: update error fields from processor proto by @lacroixthomas in https://github.com/agones-dev/agones/pull/4266
- Wasm Autoscaler: Example by @markmandel in https://github.com/agones-dev/agones/pull/4260

**Fixed bugs:**
- Fix: patch flaky tests from submit-upgrade-test-cloud-build by @lacroixthomas in https://github.com/agones-dev/agones/pull/4236
- Fix: Add missing permission for helm uninstall in upgrade test cleanup by @lacroixthomas in https://github.com/agones-dev/agones/pull/4250

**Other:**
- Preparation for Release v1.52.0 by @Sivasankaran25 in https://github.com/agones-dev/agones/pull/4234
- Updates agones-bot dependencies by @igooch in https://github.com/agones-dev/agones/pull/4232
- Update all tests to use the latest Helm version by @Sivasankaran25 in https://github.com/agones-dev/agones/pull/4238
- Bump github.com/go-viper/mapstructure/v2 from 2.2.1 to 2.3.0 in /examples/crd-client by @dependabot[bot] in https://github.com/agones-dev/agones/pull/4229
- Handle missing upgrade-test-runner pod to avoid log collection errors by @0xaravindh in https://github.com/agones-dev/agones/pull/4224
- e2e: add webhook autoscaler test with fleet metadata  by @0xaravindh in https://github.com/agones-dev/agones/pull/4251
- Bump github.com/go-viper/mapstructure/v2 from 2.2.1 to 2.4.0 in /test/upgrade by @dependabot[bot] in https://github.com/agones-dev/agones/pull/4254
- Updates the upgrade test to print any fatal error messages to the job pod termination log by @igooch in https://github.com/agones-dev/agones/pull/4252
- Pause Single Cluster Upgrade work until stable. by @markmandel in https://github.com/agones-dev/agones/pull/4257
- Replace bitname/kubectl with alpine/kubectl by @markmandel in https://github.com/agones-dev/agones/pull/4268
- Upgrade Golang to 1.24.6 and update related dependencies by @0xaravindh in https://github.com/agones-dev/agones/pull/4262
- flaky: TestControllerAllocator by @markmandel in https://github.com/agones-dev/agones/pull/4269

## [v1.51.0](https://github.com/agones-dev/agones/tree/v1.51.0) (2025-07-29)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.50.0...v1.51.0)

**Implemented enhancements:**
- Feat: Add dev feature flag for Processor Allocator by @lacroixthomas in https://github.com/agones-dev/agones/pull/4221
- feat: promote ScheduledAutoscaler to beta by @lacroixthomas in https://github.com/agones-dev/agones/pull/4226
- Adds support for lists in the Unreal SDK #4029 by @keith-miller in https://github.com/agones-dev/agones/pull/4216

**Fixed bugs:**
- Controller for Pod in Succeeded state. by @markmandel in https://github.com/agones-dev/agones/pull/4201
- Changed the sidecar requests rate limiter from exponential to a constant one by @miai10 in https://github.com/agones-dev/agones/pull/4186
- Mocked GCE metadata to fix the Stackdriver local test failure by @0xaravindh in https://github.com/agones-dev/agones/pull/4215
- Fix: Adding a retry mechanism in case the addMoreGameServers function call fails. by @txuna in https://github.com/agones-dev/agones/pull/4214
- Remove former agones collaborator from github action by @igooch in https://github.com/agones-dev/agones/pull/4228

**Other:**
- Preparation for Release v1.51.0 by @0xaravindh in https://github.com/agones-dev/agones/pull/4209
- Add tests for Prometheus metrics endpoint and validation by @0xaravindh in https://github.com/agones-dev/agones/pull/4185
- fleetautoscaler.md references metadata incorrectly by @KAllan357 in https://github.com/agones-dev/agones/pull/4217
- Add logs reporting to submit-upgrade-test-cloud-build for better visibility by @0xaravindh in https://github.com/agones-dev/agones/pull/4165
- Bump github.com/go-viper/mapstructure/v2 from 2.2.1 to 2.3.0 in /examples/custom-controller by @dependabot[bot] in https://github.com/agones-dev/agones/pull/4211
- Update region to asia-east1 for 1.33 cluster in E2E tests by @0xaravindh in https://github.com/agones-dev/agones/pull/4231

## [v1.50.0](https://github.com/agones-dev/agones/tree/v1.49.0) (2025-06-17)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.49.0...v1.50.0)

**Breaking changes:**
- Update supported Kubernetes versions to 1.31, 1.32, 1.33 by @0xaravindh in https://github.com/agones-dev/agones/pull/4199

**Implemented enhancements:**
- Feat: allow push-chart to custom helm registry by @lacroixthomas in https://github.com/agones-dev/agones/pull/4172
- Specify exit code in simple-game-server CRASH by @markmandel in https://github.com/agones-dev/agones/pull/4196
- Promote FeatureRollingUpdateFix to Beta by @0xaravindh in https://github.com/agones-dev/agones/pull/4205

**Fixed bugs:**
- Updated version mapping and post-release step by @kamaljeeti in https://github.com/agones-dev/agones/pull/4191

**Other:**
- Preparation for Release v1.50.0 by @kamaljeeti in https://github.com/agones-dev/agones/pull/4177
- Log Chain ID in Events When Applying ChainPolicy in FleetAutoscaler by @0xaravindh in https://github.com/agones-dev/agones/pull/4131
- Update release template by @kamaljeeti in https://github.com/agones-dev/agones/pull/4181
- Enhance logging and error handling in computeDesiredFleetSize, including Chain policies by @0xaravindh in https://github.com/agones-dev/agones/pull/4179
- Update best practices for multi-cluster allocation by @kamaljeeti in https://github.com/agones-dev/agones/pull/4157
- fix: exclude InactiveScheduleError for error logging by @indexjoseph in https://github.com/agones-dev/agones/pull/4183
- Updated goimports formatting by @markmandel in https://github.com/agones-dev/agones/pull/4195
- Upgrade Golang to 1.24.4 and update related dependencies and Dockerfiles by @0xaravindh in https://github.com/agones-dev/agones/pull/4204
- Created performance test cluster for 1.32 by @0xaravindh in https://github.com/agones-dev/agones/pull/4202

## [v1.49.0](https://github.com/agones-dev/agones/tree/v1.49.0) (2025-05-06)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.48.0...v1.49.0)

**Breaking changes:**
- Add AFP features and update documentation by @0xaravindh in https://github.com/agones-dev/agones/pull/4061
- Adoption of Sidecar Containers by @markmandel in https://github.com/agones-dev/agones/pull/4146

**Implemented enhancements:**
- Promote PortPolicyNone to Beta by @kamaljeeti in https://github.com/agones-dev/agones/pull/4144
- Promote FeatureDisableResyncOnSDKServer to Stable by @igooch in https://github.com/agones-dev/agones/pull/4138
- Promote PortRanges to Beta by @kamaljeeti in https://github.com/agones-dev/agones/pull/4147

**Fixed bugs:**
- Update Windows manifest handling in push-agones-sdk-manifest by @0xaravindh in https://github.com/agones-dev/agones/pull/4136
- Fix CRD API docs generation script by @0xaravindh in https://github.com/agones-dev/agones/pull/4152
- fix: ensure fleet autoscaler policy are namespaced by @lacroixthomas in https://github.com/agones-dev/agones/pull/4098
- Fix feature stages page to show expected content by @0xaravindh in https://github.com/agones-dev/agones/pull/4156
- Allocation: Re-cache allocated `GameServer` by @markmandel in https://github.com/agones-dev/agones/pull/4159

**Other:**
- Preparation for Release v1.49.0 by @0xaravindh in https://github.com/agones-dev/agones/pull/4134
- Updated doc for adding support of Extended Duration Pods on GKE Autopilot by @kamaljeeti in https://github.com/agones-dev/agones/pull/4139
- Various e2e test improvements by @markmandel in https://github.com/agones-dev/agones/pull/4143
- load test client make concurrent requests by @peterzhongyi in https://github.com/agones-dev/agones/pull/4149
- Adds permissions in the agones-system namespace for the upgrade tests by @igooch in https://github.com/agones-dev/agones/pull/4148
- Adds explicit permissions for github workflows by @igooch in https://github.com/agones-dev/agones/pull/4161
- chore: update Nitrado GameFabric branding by @nrwiersma in https://github.com/agones-dev/agones/pull/4164
- Adds instructions to update dependencies as part of upgrading Golang by @igooch in https://github.com/agones-dev/agones/pull/4155
- Fix: Remove Kubernetes 1.29 from Agones 1.39.0 compatibility matrix by @0xaravindh in https://github.com/agones-dev/agones/pull/4168
- Documentation for Sidecar Containers by @markmandel in https://github.com/agones-dev/agones/pull/4171
- Upgrade: Go to 1.23.8 and deps by @0xaravindh in https://github.com/agones-dev/agones/pull/4170
- Updates GKE Autopilot documentation to include Passthrough portPolicy by @igooch in https://github.com/agones-dev/agones/pull/4173

## [v1.48.0](https://github.com/agones-dev/agones/tree/v1.48.0) (2025-03-25)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.47.0...v1.48.0)

**Implemented enhancements**
- add metadata to agones webhook autoscaler request by @swermin in https://github.com/agones-dev/agones/pull/3957
- feat(helm): support dual-stack networking for load balancers by @bergemalm in https://github.com/agones-dev/agones/pull/4073

**Fixed bugs**
- fix: bump version of jsonpatch for lossy max int64 by @lacroixthomas in https://github.com/agones-dev/agones/pull/4090
- Fix JSON Schema validation for ServiceAccount annotations by @0xaravindh in https://github.com/agones-dev/agones/pull/4122
- Refactor image build and manifest push process by @0xaravindh in https://github.com/agones-dev/agones/pull/4118

**Other**
- Preparation for Release v1.48.0 by @0xaravindh in https://github.com/agones-dev/agones/pull/4109
- Increase wait time for upgrade test runner by @igooch in https://github.com/agones-dev/agones/pull/4113
- Update Helm Schema Validation for topologySpreadConstraints and customCertSecretPath  by @AliaksandrTsimokhau in https://github.com/agones-dev/agones/pull/4112
- Fix: Ensure Buildx Builders Are Created or Used for ARM64 and Windows by @0xaravindh in https://github.com/agones-dev/agones/pull/4115
- Update Supported Kubernetes to 1.30, 1.31, 1.32 by @kamaljeeti in https://github.com/agones-dev/agones/pull/4124
- helm: change type from object to array for controller.customCertSecre… by @Joseph-Irving in https://github.com/agones-dev/agones/pull/4120
- Created performance test cluster for 1.31 by @kamaljeeti in https://github.com/agones-dev/agones/pull/4125
- Add deprecation notice for older image versions in release template by @0xaravindh in https://github.com/agones-dev/agones/pull/4126
- Fix flaky test TestListAutoscalerAllocated by @igooch in https://github.com/agones-dev/agones/pull/4130

**New Contributors**
- @AliaksandrTsimokhau made their first contribution in https://github.com/agones-dev/agones/pull/4112
- @swermin made their first contribution in https://github.com/agones-dev/agones/pull/3957
- @bergemalm made their first contribution in https://github.com/agones-dev/agones/pull/4073
- @Joseph-Irving made their first contribution in https://github.com/agones-dev/agones/pull/4120

## [v1.47.0](https://github.com/agones-dev/agones/tree/v1.47.0) (2025-02-11)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.46.0...v1.47.0)

**Implemented enhancements:**
- Docs: Update Player Tracking to use Lists by @markmandel in https://github.com/agones-dev/agones/pull/4086
- Docs: Counters with High Density by @markmandel in https://github.com/agones-dev/agones/pull/4085
- Add ability to change externalTrafficPolicy for agones-ping services (http&udp) by @zifter in https://github.com/agones-dev/agones/pull/4083
- JSON Schema Validation for Helm by @igooch in https://github.com/agones-dev/agones/pull/4094
- Adds helm schema validation test to the test suite by @igooch in https://github.com/agones-dev/agones/pull/4101

**Fixed bugs:**
- Changes upgrade game server template to use safe-to-evict: Always by @igooch in https://github.com/agones-dev/agones/pull/4096

**Other:**
- Preparation for Release v1.47.0 by @0xaravindh in https://github.com/agones-dev/agones/pull/4079
- Update `.golangci.yml` config to resolve deprecation warnings by @paulinek13 in https://github.com/agones-dev/agones/pull/4082
- Initialise FCounterResponse members by @alexrudd in https://github.com/agones-dev/agones/pull/4084
- Bump golang.org/x/crypto from 0.21.0 to 0.31.0 in /build/agones-bot by @dependabot in https://github.com/agones-dev/agones/pull/4062
- Added OKE steps in K8S version upgrade template by @kamaljeeti in https://github.com/agones-dev/agones/pull/4091
- User and developer documentation for Helm json schema validation by @igooch in https://github.com/agones-dev/agones/pull/4100
- Update All Go Module Dependencies to Latest Patches by @0xaravindh in https://github.com/agones-dev/agones/pull/4104
- Bump github.com/go-git/go-git/v5 from 5.12.0 to 5.13.0 in /build/scripts/example-version-checker by @dependabot in https://github.com/agones-dev/agones/pull/4088

**New Contributors:**
- @paulinek13 made their first contribution in https://github.com/agones-dev/agones/pull/4082
- @alexrudd made their first contribution in https://github.com/agones-dev/agones/pull/4084

## [v1.46.0](https://github.com/agones-dev/agones/tree/v1.46.0) (2025-01-02)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.45.0...v1.46.0)

**Breaking changes:**
- Removed reflector metric usage by @vicentefb in https://github.com/agones-dev/agones/pull/4056

**Implemented enhancements:**
- Set externalTrafficPolicy as Local for agones-allocator by @osterante in https://github.com/agones-dev/agones/pull/4022
- Integrates upgrades tests into Cloud Build by @igooch in https://github.com/agones-dev/agones/pull/4037
- Delete List Value(s) on Game Server Allocation by @igooch in https://github.com/agones-dev/agones/pull/4054
- In place upgrades version update instructions by @igooch in https://github.com/agones-dev/agones/pull/4064

**Fixed bugs:**
- Correct CI check for examples and add a unit test by @wheatear-dev in https://github.com/agones-dev/agones/pull/4045
- Enable counter based autoscaler to scale from 0 replicas by @geopaulm in https://github.com/agones-dev/agones/pull/4049

**Other:**
- Preparation for Release v1.46.0 by @0xaravindh in https://github.com/agones-dev/agones/pull/4043
- Updates Kubernetes templates for cpp-simple image by @0xaravindh in https://github.com/agones-dev/agones/pull/4044
- Changes upgrades clusters to use only us based regions by @igooch in https://github.com/agones-dev/agones/pull/4046
- Clarify docs on GKE Autopilot and node pools by @danfairs in https://github.com/agones-dev/agones/pull/4048
- Updated typo's in multiple files by @nallave in https://github.com/agones-dev/agones/pull/4055
- Flake: e2e/TestScheduleAutoscaler by @markmandel in https://github.com/agones-dev/agones/pull/4058
- Add ability to specify additional labels for controller and extension pods by @R4oulDuk3 in https://github.com/agones-dev/agones/pull/4057
- Adds Documention for how to run an in-place Agones upgrade by @igooch in https://github.com/agones-dev/agones/pull/3904
- Fixes build error in push-upgrade-test by @igooch in https://github.com/agones-dev/agones/pull/4065
- Fix broken link by @0xaravindh in https://github.com/agones-dev/agones/pull/4070
- Link to Google Cloud Agones Support. by @markmandel in https://github.com/agones-dev/agones/pull/4071
- Upgrade Go to 1.23.4 and update example image tags by @0xaravindh in https://github.com/agones-dev/agones/pull/4072
- Unblocks Agones release PR by waiting for either the Agones dev version or release version by @igooch in https://github.com/agones-dev/agones/pull/4078

**New Contributors:**
- @danfairs made their first contribution in https://github.com/agones-dev/agones/pull/4048
- @osterante made their first contribution in https://github.com/agones-dev/agones/pull/4022
- @nallave made their first contribution in https://github.com/agones-dev/agones/pull/4055
- @R4oulDuk3 made their first contribution in https://github.com/agones-dev/agones/pull/4057

## [v1.45.0](https://github.com/agones-dev/agones/tree/v1.45.0) (2024-11-19)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.44.0...v1.45.0)

**Breaking changes:**
- Update Supported Kubernetes to 1.29, 1.30, 1.31 by @kamaljeeti in https://github.com/agones-dev/agones/pull/4024

**Implemented enhancements:**
- Dashboard for Agones GameServer State duration by @vicentefb in https://github.com/agones-dev/agones/pull/3947
- Add Shutdown Delay Seconds to the sdk-client-test containers by @igooch in https://github.com/agones-dev/agones/pull/4030
- Add a CI check to fail on change to an example without a new version by @wheatear-dev in https://github.com/agones-dev/agones/pull/3940

**Fixed bugs:**
- Allowing list based fleet autoscaler to scale up from 0 replicas by @geopaulm in https://github.com/agones-dev/agones/pull/4016

**Other:**
- Preparation for Release v1.45.0 by @0xaravindh in https://github.com/agones-dev/agones/pull/4014
- Update all Rust SDK dependencies to latest versions by @john-haven in https://github.com/agones-dev/agones/pull/4008
- Write Terraform scripts and docs to show how to create OKE cluster and install Agones by @ouxingning in https://github.com/agones-dev/agones/pull/4023
- Created performance cluster 1.30 by @kamaljeeti in https://github.com/agones-dev/agones/pull/4031
- Updates the upgrade terraform by @igooch in https://github.com/agones-dev/agones/pull/4036
- Adding Fleet Active GameServerSet Percentage Metrics by @0xaravindh in https://github.com/agones-dev/agones/pull/4021
- Introducing Agones Guru on Gurubase.io by @kursataktas in https://github.com/agones-dev/agones/pull/4028

**New Contributors:**
- @john-haven made their first contribution in https://github.com/agones-dev/agones/pull/4008
- @geopaulm made their first contribution in https://github.com/agones-dev/agones/pull/4016
- @ouxingning made their first contribution in https://github.com/agones-dev/agones/pull/4023
- @wheatear-dev made their first contribution in https://github.com/agones-dev/agones/pull/3940
- @kursataktas made their first contribution in https://github.com/agones-dev/agones/pull/4028

## [v1.44.0](https://github.com/agones-dev/agones/tree/v1.44.0) (2024-10-08)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.43.0...v1.44.0)

**Implemented enhancements:**
- Unreal SDK - Added counters to unreal sdk by @GloryOfNight in https://github.com/agones-dev/agones/pull/3935
- Unreal SDK - Add Support for GameServer Status Addresses by @KAllan357 in https://github.com/agones-dev/agones/pull/3932
- Updates upgrade test to install multiple versions of Agones on a cluster in succession by @igooch in https://github.com/agones-dev/agones/pull/3982
- Adds game server template with containerized sdk-client-test by @igooch in https://github.com/agones-dev/agones/pull/3987
- Adds clusters for the in place upgrades tests by @igooch in https://github.com/agones-dev/agones/pull/3990
- Test in place upgrades run tests by @igooch in https://github.com/agones-dev/agones/pull/3991
- Move Feature GKEAutopilotExtendedDurationPods To Beta by @kamaljeeti in https://github.com/agones-dev/agones/pull/4006

**Fixed bugs:**
- fix: remove bad character from metrics markdown by @code-eg in https://github.com/agones-dev/agones/pull/3981
- Updating UpdateList to update the values on a list by @chrisfoster121 in https://github.com/agones-dev/agones/pull/3899
- Cleanup Patch Sidecar Logging by @markmandel in https://github.com/agones-dev/agones/pull/3973
- Refactor metrics registry exporter by @kamaljeeti in https://github.com/agones-dev/agones/pull/3989
- Fix the build-e2e error by @gongmax in https://github.com/agones-dev/agones/pull/4009
- Add a flag to sdkserver to avoid a collision on port 8080 by @KAllan357 in https://github.com/agones-dev/agones/pull/4010

**Other:**
- Update the note at the top of the player tracking docs by @roberthbailey in https://github.com/agones-dev/agones/pull/3974
- Adds schedule and chain policy to fleetautoscaler documentation by @indexjoseph in https://github.com/agones-dev/agones/pull/3934
- Improve documentation to run performance script by @vicentefb in https://github.com/agones-dev/agones/pull/3948
- Preparation for Release v1.44.0 by @kamaljeeti in https://github.com/agones-dev/agones/pull/3975
- Add instructions for running Agones on Minikube with the Windows Docker driver by @brightestpixel in https://github.com/agones-dev/agones/pull/3965
- Use Markdown when use k8s-api-version variable by @peterzhongyi in https://github.com/agones-dev/agones/pull/3964
- Refactor Terraform by @kamaljeeti in https://github.com/agones-dev/agones/pull/3958
- Created performance cluster 1.29 by @ashutosji in https://github.com/agones-dev/agones/pull/3986
- Adding missing documentation about: add option for extensions components to use host network and configure ports by @Orza in https://github.com/agones-dev/agones/pull/3912
- fix: correct misspelled metric in docs by @antiphp in https://github.com/agones-dev/agones/pull/3999
- Add finalizer name change to create gameserver example by @indexjoseph in https://github.com/agones-dev/agones/pull/4005
- Formatting code with gofmt by @cuishuang in https://github.com/agones-dev/agones/pull/4000
- Add 'Trace' to LogLevel in GameServer.Spec.SdkServer by @0xaravindh in https://github.com/agones-dev/agones/pull/3995
- Upgrade to Golang Version 1.22.6 and Golangci lint version v1.61.0 by @0xaravindh in https://github.com/agones-dev/agones/pull/3988
- Update the go version upgrade template by @gongmax in https://github.com/agones-dev/agones/pull/4011

**New Contributors:**
- @GloryOfNight made their first contribution in https://github.com/agones-dev/agones/pull/3935
- @brightestpixel made their first contribution in https://github.com/agones-dev/agones/pull/3965
- @code-eg made their first contribution in https://github.com/agones-dev/agones/pull/3981
- @chrisfoster121 made their first contribution in https://github.com/agones-dev/agones/pull/3899
- @cuishuang made their first contribution in https://github.com/agones-dev/agones/pull/4000
- @0xaravindh made their first contribution in https://github.com/agones-dev/agones/pull/3995

## [v1.43.0](https://github.com/agones-dev/agones/tree/v1.43.0) (2024-08-27)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.42.0...v1.43.0)

**Implemented enhancements:**
- Add Option to Use Host Network and Configure Ports by @Orza in https://github.com/agones-dev/agones/pull/3895
- Graduate Passthrough Port Policy to Beta on Autopilot by @vicentefb in https://github.com/agones-dev/agones/pull/3916
- Agones Unity SDK development setup instructions + Agones Unity SDK Ready test by @aallbrig in https://github.com/agones-dev/agones/pull/3887
- feat: Add API Changes and Validation for FleetAutoscaler Schedule/Chain Policy by @indexjoseph in https://github.com/agones-dev/agones/pull/3893
- feat: Adds autoscaling logic for new Chain and Schedule policies by @indexjoseph in https://github.com/agones-dev/agones/pull/3929
- Adds basic framework for the in place Agones upgrades test controller by @igooch in https://github.com/agones-dev/agones/pull/3956
- [Performance] - Added a new metric inside the allocator to track the success retry rate inside the retry loop  by @vicentefb in https://github.com/agones-dev/agones/pull/3927
- Make the parameters that limits the number of GameServers to add configurable by @vicentefb in https://github.com/agones-dev/agones/pull/3950
- feat: Adds e2e tests for chain/schedule policy and bump ScheduledAutoscaler to Alpha by @indexjoseph in https://github.com/agones-dev/agones/pull/3946
- Implement CountsAndLists for Unity SDK + Tests by @ZeroParticle in https://github.com/agones-dev/agones/pull/3883

**Fixed bugs:**
- Resolves `make site-server` issue #3885 by @aallbrig in https://github.com/agones-dev/agones/pull/3914

**Other:**
- Preparation for Release v1.43.0 by @kamaljeeti in https://github.com/agones-dev/agones/pull/3910
- Introduce external resource(s) on multiplayer game programming to docs by @aallbrig in https://github.com/agones-dev/agones/pull/3884
- Added line of code to update failure count details inside runscenario by @vicentefb in https://github.com/agones-dev/agones/pull/3915
- updated golang upgrade template by @ashutosji in https://github.com/agones-dev/agones/pull/3902
- Changes for GitHub/Cloud Build app integration by @zmerlynn in https://github.com/agones-dev/agones/pull/3918
- Meta: Contributor role by @markmandel in https://github.com/agones-dev/agones/pull/3922
- Fix allocator metrics endpoint by @vicentefb in https://github.com/agones-dev/agones/pull/3921
- Meta: Contributor => Collaborator by @markmandel in https://github.com/agones-dev/agones/pull/3928
- Rewrite agones-bot, commit to Agones repo by @zmerlynn in https://github.com/agones-dev/agones/pull/3923
- Small cleanup of incorrect comment in features.go file by @igooch in https://github.com/agones-dev/agones/pull/3944
- Update Supported Kubernetes to 1.28, 1.29, 1.30 by @ashutosji in https://github.com/agones-dev/agones/pull/3933
- remove ctx within the condition func by @peterzhongyi in https://github.com/agones-dev/agones/pull/3959
- Reapply "Update Supported Kubernetes to 1.28, 1.29, 1.30 (#3933)" (#3… by @gongmax in https://github.com/agones-dev/agones/pull/3961
- change kubernetes API version to fix broken CI by @peterzhongyi in https://github.com/agones-dev/agones/pull/3962
- docs(godot): add Agones x Godot third party example by @andresromerodev in https://github.com/agones-dev/agones/pull/3938
- Link Unity Netcode for Gameobjects example in documentation by @mbychkowski in https://github.com/agones-dev/agones/pull/3937
- Docs: Use k8s-api-version for links by @markmandel in https://github.com/agones-dev/agones/pull/3963

**New Contributors:**
- @Orza made their first contribution in https://github.com/agones-dev/agones/pull/3895

## [v1.42.0](https://github.com/agones-dev/agones/tree/v1.42.0) (2024-07-16)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.41.0...v1.42.0)

**Breaking changes:**
- Update csharp.md to indicate ConnectAsync is deprecated by @aallbrig in https://github.com/agones-dev/agones/pull/3866

**Implemented enhancements:**
- Add security context to Agones containers by @peterzhongyi in https://github.com/agones-dev/agones/pull/3856
- Add Security Context to game server sidecar by @peterzhongyi in https://github.com/agones-dev/agones/pull/3869
- Drop CountsAndLists Data from the Fleet and Game Server Set When the Flag is False by @igooch in https://github.com/agones-dev/agones/pull/3881
- Adds tests to confirm that Fleet, Fleet Autoscaler, and Fleet Allocation apply defaults code is idempotent by @igooch in https://github.com/agones-dev/agones/pull/3888
- feat: Add CRD Changes and Feature Flag for chain policy by @indexjoseph in https://github.com/agones-dev/agones/pull/3880

**Fixed bugs:**
- sdk-server expects SDK_LOG_LEVEL by @KAllan357 in https://github.com/agones-dev/agones/pull/3858
- this will resolve From/layer extraction issue on ltsc2019 in examples by @ashutosji in https://github.com/agones-dev/agones/pull/3873
- featuregate: adds validation if PortPolicyNone is not enabled by @daniellee in https://github.com/agones-dev/agones/pull/3871
- added local as default for registry when registry is not specified by @kamaljeeti in https://github.com/agones-dev/agones/pull/3876
- Buffer Unity SDK ReceiveData when watching for configuration changes by @ZeroParticle in https://github.com/agones-dev/agones/pull/3872
- agones-{extensions,allocator}: Make servers context aware by @zmerlynn in https://github.com/agones-dev/agones/pull/3845
- added condition for distributed logic by @ashutosji in https://github.com/agones-dev/agones/pull/3877

**Security fixes:**
- Bump @grpc/grpc-js from 1.10.7 to 1.10.9 in /sdks/nodejs by @dependabot in https://github.com/agones-dev/agones/pull/3863

**Other:**
- Preparation for Release v1.42.0 by @ashutosji in https://github.com/agones-dev/agones/pull/3854
- Add helpful note to edit-first-gameserver-go by @peterzhongyi in https://github.com/agones-dev/agones/pull/3846
- Moved Passthrough feature description to the correct section in Feature Stages by @vicentefb in https://github.com/agones-dev/agones/pull/3861
- Updated Node.js Page to Reflect that Counters and Lists is Implemented by @ashutosji in https://github.com/agones-dev/agones/pull/3865
- Change Slack channel description from #developers to #development by @branhoff in https://github.com/agones-dev/agones/pull/3868
- updated UpdateList documentation for local sdk server and sdk server by @ashutosji in https://github.com/agones-dev/agones/pull/3878
- Add zio-agones to the list of third party client SDKs by @ghostdogpr in https://github.com/agones-dev/agones/pull/3875
- refactor simple game server by @ashutosji in https://github.com/agones-dev/agones/pull/3817
- Update Slack invite link by @markmandel in https://github.com/agones-dev/agones/pull/3896
- Added cleanup for app-engine services in cloudbuild script by @kamaljeeti in https://github.com/agones-dev/agones/pull/3890
- Adds a command to generate the zz_generated.deepcopy.go files for the apis by @igooch in https://github.com/agones-dev/agones/pull/3900
- update go version to 1.21.12 by @ashutosji in https://github.com/agones-dev/agones/pull/3894

**New Contributors:**
- @KAllan357 made their first contribution in https://github.com/agones-dev/agones/pull/3858
- @branhoff made their first contribution in https://github.com/agones-dev/agones/pull/3868
- @aallbrig made their first contribution in https://github.com/agones-dev/agones/pull/3866
- @ZeroParticle made their first contribution in https://github.com/agones-dev/agones/pull/3872
- @ghostdogpr made their first contribution in https://github.com/agones-dev/agones/pull/3875

## [v1.41.0](https://github.com/agones-dev/agones/tree/v1.41.0) (2024-06-04)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.40.0...v1.41.0)

**Implemented enhancements:**
- Configure Allocator Status Code by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3782
- Graduate Counters and Lists to Beta by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3801
- Passthrough autopilot - Adds an AutopilotPassthroughPort Feature Gate and new pod label by @vicentefb in https://github.com/agones-dev/agones/pull/3809
- CountsAndLists: Move to Beta Protobuf by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3806
- feat: support multiple port ranges by @nrwiersma in https://github.com/agones-dev/agones/pull/3747
- Changes `sdk-server` to Patch instead of Update by @igooch in https://github.com/agones-dev/agones/pull/3803
- Generate grpc for nodejs from alpha to beta by @lacroixthomas in https://github.com/agones-dev/agones/pull/3825
- Update CountsAndLists from Alpha to Beta by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3824
- feat(gameserver): New DirectToGameServer PortPolicy allows direct traffic to a GameServer by @daniellee in https://github.com/agones-dev/agones/pull/3807
- Passthrough autopilot - Adds mutating webhook by @vicentefb in https://github.com/agones-dev/agones/pull/3833
- Passthrough autopilot - added ports array case and updated unit tests by @vicentefb in https://github.com/agones-dev/agones/pull/3842
- Nodejs counters and lists by @steven-supersolid in https://github.com/agones-dev/agones/pull/3726
- Promote AutopilotPassthroughPort feature gate to Alpha by @vicentefb in https://github.com/agones-dev/agones/pull/3849

**Fixed bugs:**
- Helm Param Update: Default to agones.controller if agones.extensions is Missing by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3773
- fix: rollout strategy issues by @nrwiersma in https://github.com/agones-dev/agones/pull/3762
- Set Minimum Buffer Size to 1 by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3749
- Pin ltsc2019 to older SHA by @zmerlynn in https://github.com/agones-dev/agones/pull/3829
- TestGameServerAllocationDuringMultipleAllocationClients: Readdress flake by @zmerlynn in https://github.com/agones-dev/agones/pull/3831
- Refactor finalizer name to include valid domain name and path by @indexjoseph in https://github.com/agones-dev/agones/pull/3840
- agones-{extensions,allocator}: Be more defensive about draining by @zmerlynn in https://github.com/agones-dev/agones/pull/3839
- agones-{extensions,allocator}: Pause after cancelling context by @zmerlynn in https://github.com/agones-dev/agones/pull/3843
- Change the line to modify in Quickstart: Edit a Game Server by @peterzhongyi in https://github.com/agones-dev/agones/pull/3844

**Other:**
- Prep for Release v1.41.0 by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3800
- Update site documentation to reflect firewall prefix and default to Autopilot cluster creation for Agones by @vicentefb in https://github.com/agones-dev/agones/pull/3769
- Add a System Diagram and overview page by @zmerlynn in https://github.com/agones-dev/agones/pull/3792
- Update Side Menu: Preserve and Restore Scroll Position by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3805
- fix: typo by @skmpf in https://github.com/agones-dev/agones/pull/3808
- Helm Config: Add httpUnallocatedStatusCode in Allocator Service by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3802
- Update Docs: CountersAndLists to Beta by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3810
- Disable Dev feature FeatureAutopilotPassthroughPort by @vicentefb in https://github.com/agones-dev/agones/pull/3815
- Disable FeatureAutopilotPassthroughPort in features.go by @vicentefb in https://github.com/agones-dev/agones/pull/3816
- SDK proto compatibility guarantees and deprecation policies documentation by @igooch in https://github.com/agones-dev/agones/pull/3774
- Fix dangling "as of" by @zmerlynn in https://github.com/agones-dev/agones/pull/3827
- Steps to Promote SDK Features from Alpha to Beta by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3814
- Adds comment for help troubleshooting issues with terraform tfstate by @igooch in https://github.com/agones-dev/agones/pull/3822
- docs: improve counter and list example comments by @yonbh in https://github.com/agones-dev/agones/pull/3818
- Skip /tmp/ on yamllint by @zmerlynn in https://github.com/agones-dev/agones/pull/3838
- TestAllocatorAfterDeleteReplica: More logging by @zmerlynn in https://github.com/agones-dev/agones/pull/3837
- Instructions for upgrading golang version by @gongmax in https://github.com/agones-dev/agones/pull/3819
- Remove unused function FindGameServerContainer by @zmerlynn in https://github.com/agones-dev/agones/pull/3841
- Adds Unreal to the List of URL Links to Not Check by @igooch in https://github.com/agones-dev/agones/pull/3847
- docs: clarify virtualization setup for Windows versions by @andresromerodev in https://github.com/agones-dev/agones/pull/3850

**New Contributors:**
- @skmpf made their first contribution in https://github.com/agones-dev/agones/pull/3808
- @yonbh made their first contribution in https://github.com/agones-dev/agones/pull/3818
- @peterzhongyi made their first contribution in https://github.com/agones-dev/agones/pull/3844
- @andresromerodev made their first contribution in https://github.com/agones-dev/agones/pull/3850

## [v1.40.0](https://github.com/agones-dev/agones/tree/v1.40.0) (2024-04-23)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.39.0...v1.40.0)

**Breaking changes:**
- Counters and Lists: Remove Bool Returns  by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3738

**Implemented enhancements:**
- Leader Election in Custom Controller by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3696
- Migrating from generate-groups.sh to kube_codegen.sh by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3722
- Move GKEAutopilotExtendedDurationPods to Alpha in 1.28+ by @zmerlynn in https://github.com/agones-dev/agones/pull/3729
- Move DisableResyncOnSDKServer to Beta by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3732
- Counters & Lists landing page and doc improvements by @markmandel in https://github.com/agones-dev/agones/pull/3649
- Graduate FleetAllocationOverflow to Stable by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3733
- Adds Counters and Lists to CSharp SDK by @igooch in https://github.com/agones-dev/agones/pull/3581
- Feat/counter and list defaulting order to ascending by @lacroixthomas in https://github.com/agones-dev/agones/pull/3734
- Add handling for StatusAddresses in GameServerStatus for the Unity SDK by @charlesvien in https://github.com/agones-dev/agones/pull/3739
- Feat(gameservers): Shared pod IPs with GameServer Addresses by @lacroixthomas in https://github.com/agones-dev/agones/pull/3764
- Be prescriptive about rotating regions when updating Kubernetes versions by @zmerlynn in https://github.com/agones-dev/agones/pull/3716
- Fix ensure-e2e-infra-state-bucket by @zmerlynn in https://github.com/agones-dev/agones/pull/3719
- Create Performance Cluster 1.28 by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3720
- Optimise GameServer Sub-Controller Queues by @markmandel in https://github.com/agones-dev/agones/pull/3781

**Fixed bugs:**
- Counters & Lists: Consolidate `priorities` sorting by @markmandel in https://github.com/agones-dev/agones/pull/3690
- Fix(Counter & Lists): Add validation for `priorities` by @lacroixthomas in https://github.com/agones-dev/agones/pull/3714
- fix: #3607 Metrics data loss in K8S controller by @alvin-7 in https://github.com/agones-dev/agones/pull/3692
- Deflake GameServerAllocationDuringMultipleAllocationClients by allowing errors by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3750

**Security fixes:**
- Bump protobufjs from 7.2.4 to 7.2.6 in /sdks/nodejs by @dependabot in https://github.com/agones-dev/agones/pull/3755
- Bump golang.org/x/net from 0.19.0 to 0.23.0 by @zmerlynn in https://github.com/agones-dev/agones/pull/3793

**Other:**
- Flaky: TestGameServerCreationAfterDeletingOneExtensionsPod by @markmandel in https://github.com/agones-dev/agones/pull/3699
- Prep for release v1.40.0 by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3700
- Bumps cpp-simple Image and Refactoring Example Makefiles by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3695
- Upgrade Protobuf to 1.33.0 by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3711
- Modify Script for Makefile Version Updates in Examples Directory by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3712
- Adds simple genai server example documentation to the Agones site by @igooch in https://github.com/agones-dev/agones/pull/3713
- Update Supported Kubernetes to 1.27, 1.28, 1.29 by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3654
- fix: typo in docs by @qhyun2 in https://github.com/agones-dev/agones/pull/3723
- Tweak: Setting up the Game Server by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3717
- Docs: gke.md - spelling by @daniellee in https://github.com/agones-dev/agones/pull/3740
- Aesthetic rearrangement of cloudbuild.yaml by @zmerlynn in https://github.com/agones-dev/agones/pull/3741
- Docs: Make hitting <enter> on connection explicit by @markmandel in https://github.com/agones-dev/agones/pull/3743
- CI: Don't check Unreal Link by @markmandel in https://github.com/agones-dev/agones/pull/3745
- New recommendation for multi-cluster allocation by @markmandel in https://github.com/agones-dev/agones/pull/3744
- Custom Controller Example Page on Agones Website by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3725
- Add Nitrado logo by @towolf in https://github.com/agones-dev/agones/pull/3753
- Remove unnecessary args from e2e-test-cloudbuild by @zmerlynn in https://github.com/agones-dev/agones/pull/3754
- Update Allocation from Fleet Documentation by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3761
- Transform Lint Warnings into Errors by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3756
- Update Canary Testing Documentation by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3760
- Supertuxkart Example on Agones Site by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3728
- Xonotic Example on Agones Site by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3742
- nit documentation fix in kind cluster section when building Agones by @vicentefb in https://github.com/agones-dev/agones/pull/3770
- Merged steps inside documentation about webhook certificate creation by @vicentefb in https://github.com/agones-dev/agones/pull/3768
- Example Images: Increment Tags by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3796
- Update simple game server example documentation by @vicentefb in https://github.com/agones-dev/agones/pull/3776

**New Contributors:**
- @lacroixthomas made their first contribution in https://github.com/agones-dev/agones/pull/3714
- @daniellee made their first contribution in https://github.com/agones-dev/agones/pull/3740
- @charlesvien made their first contribution in https://github.com/agones-dev/agones/pull/3739
- @vicentefb made their first contribution in https://github.com/agones-dev/agones/pull/3770

## [v1.39.0](https://github.com/agones-dev/agones/tree/v1.39.0) (2024-03-12)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.38.0...v1.39.0)

**Breaking changes:**
- Breaking: Remove Cmake gRPC install when not found by @markmandel in https://github.com/agones-dev/agones/pull/3621
- by default disable agones-metrics nodepools by @ashutosji in https://github.com/agones-dev/agones/pull/3672

**Implemented enhancements:**
- More description on fleetautoscaler.md by @markmandel in https://github.com/agones-dev/agones/pull/3632
- Modify NewSDK(): Hardcode localhost by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3676
- Allow passing values to Helm release of the Agones Terraform module by @Pierca7 in https://github.com/agones-dev/agones/pull/3665
- Create Controller Example by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3680
- feat: allocation response with counters and lists data by @katsew in https://github.com/agones-dev/agones/pull/3681
- simple-genai-server 0.2: Make autonomous mode effective by @zmerlynn in https://github.com/agones-dev/agones/pull/3693

**Fixed bugs:**
- fix(SdkList):  fix list delete values panic by @GStones in https://github.com/agones-dev/agones/pull/3615
- Define SDKServer LogLevel early by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3631
- Fix the handling of removing disconnected streams to avoid a panic when multiple streams disconnect from the sdkserver by @roberthbailey in https://github.com/agones-dev/agones/pull/3668
- resolve flaky e2e test by @ashutosji in https://github.com/agones-dev/agones/pull/3616
- fix: cannot load extensions image on minikube node by @katsew in https://github.com/agones-dev/agones/pull/3682
- added mutex at right places by @ashutosji in https://github.com/agones-dev/agones/pull/3678
- correct path of gameserver for windows node by @ashutosji in https://github.com/agones-dev/agones/pull/3687

**Other:**
- Prep for release v1.39.0 by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3620
- Flake: List Add/Delete Unit Tests by @markmandel in https://github.com/agones-dev/agones/pull/3627
- Script to bump example images by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3626
- Linting: need `git ... --add safe.directory` by @markmandel in https://github.com/agones-dev/agones/pull/3638
- Migrate to https://github.com/gomodules/jsonpatch by @markmandel in https://github.com/agones-dev/agones/pull/3639
- Docs: Default Counter Capacity as 1000 by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3637
- Build: Replace godoc with pkgsite by @markmandel in https://github.com/agones-dev/agones/pull/3643
- fix: typo by @qhyun2 in https://github.com/agones-dev/agones/pull/3658
- Switch to debian:bookworm by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3657
- Document `Distributed` pod scheduling. by @markmandel in https://github.com/agones-dev/agones/pull/3662
- Downscale performance test cluster by @markmandel in https://github.com/agones-dev/agones/pull/3666
- Info log level on Performance tests by @markmandel in https://github.com/agones-dev/agones/pull/3667
- Adds simple game server for gen AI by @igooch in https://github.com/agones-dev/agones/pull/3628
- fix: minor typos for simple-genai-server endpoints and readme by @indexjoseph in https://github.com/agones-dev/agones/pull/3673
- Local SDK: Counters and Lists by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3660
- Adds Chat Message History and Connects to the NPC Chat API by @igooch in https://github.com/agones-dev/agones/pull/3679
- Adding build targets for the simple-genai-server example. by @roberthbailey in https://github.com/agones-dev/agones/pull/3689

**New Contributors:**
- @GStones made their first contribution in https://github.com/agones-dev/agones/pull/3615
- @indexjoseph made their first contribution in https://github.com/agones-dev/agones/pull/3673
- @Pierca7 made their first contribution in https://github.com/agones-dev/agones/pull/3665

## [v1.38.0](https://github.com/agones-dev/agones/tree/v1.38.0) (2024-01-30)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.37.0...v1.38.0)

**Breaking changes:**
- Nodepool upgrades on GKE Terraform apply by @markmandel in https://github.com/agones-dev/agones/pull/3612

**Implemented enhancements:**
- Add Feature Template for Issues Created from Agones Website by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3561
- controller refresh certificate by @ashutosji in https://github.com/agones-dev/agones/pull/3489
- Kubernetes Config Update: Prioritize InClusterConfig function by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3584
- Support topologySpreadConstraints by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3591

**Fixed bugs:**
- ci/cache project root cloudbuild.yaml fix by @markmandel in https://github.com/agones-dev/agones/pull/3566
- GKEAutopilotExtendedDurationPods: Fix embarassing typo preventing use by @zmerlynn in https://github.com/agones-dev/agones/pull/3596
- Prevent Int64 Overflow by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3605
- SDK server not clearing lists on update by @jlory in https://github.com/agones-dev/agones/pull/3606

**Other:**
- Prep for release v1.38.0 by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3558
- Bump golang.org/x/crypto from 0.14.0 to 0.17.0 in /examples/allocation-endpoint/client by @dependabot in https://github.com/agones-dev/agones/pull/3551
- Bump golang.org/x/crypto from 0.14.0 to 0.17.0 by @dependabot in https://github.com/agones-dev/agones/pull/3550
- Increase performance test cluster size by @gongmax in https://github.com/agones-dev/agones/pull/3559
- fix: typo by @qhyun2 in https://github.com/agones-dev/agones/pull/3562
- Docs: Link to SDK Service Account by @markmandel in https://github.com/agones-dev/agones/pull/3565
- Docs: gomod go 1.21 by @markmandel in https://github.com/agones-dev/agones/pull/3568
- Upgrade Docker to 24.0.6 by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3567
- Upgrade from Debian Bullseye to Bookworm for Rust by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3571
- Update /cmd: Switch from debian11 to debian12 by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3577
- Upgrade from Debian Bullseye to Bookworm for NodeJS by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3572
- Switch from debian11 to debian12 for crd-client image by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3573
- Update autoscaler-webhook: Switch from debian11 to debian12 by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3575
- Fix Lint Warning by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3585
- Update simple-game-server: Switch from debian11 to debian12 by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3576
- Bump simple-game-server to 0.24 by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3588
- Bump Example Images: Rust, Crd-client, NodeJS, Autoscaler-webhook by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3587
- Use Docker 24.0.6 for performanace test by @gongmax in https://github.com/agones-dev/agones/pull/3592
- Upgrade Docker to 24.0.6  by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3580
- Update Site Go Version by @markmandel in https://github.com/agones-dev/agones/pull/3595
- Docs: Lifecycle Management of Counters and Lists in REST by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3560
- Local SDK: Refactor List and Count keys for default GameServer by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3598
- Docs: Game Server Allocation Details  by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3583
- Counts and Lists: Improvements to SDK docs by @markmandel in https://github.com/agones-dev/agones/pull/3569
- Upgrade Golang Version to 1.21.6 by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3602
- Example Images with Updated Tags by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3613
- Simple Game Server: Add \n to Counters and Lists Response by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3589

**New Contributors:**
- @qhyun2 made their first contribution in https://github.com/agones-dev/agones/pull/3562
- @jlory made their first contribution in https://github.com/agones-dev/agones/pull/3606

## [v1.37.0](https://github.com/agones-dev/agones/tree/v1.37.0) (2023-12-19)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.36.0...v1.37.0)

**Implemented enhancements:**
- Adds Counter conformance test by @igooch in https://github.com/agones-dev/agones/pull/3488
- Adds List SDK methods to simple-game-server by @igooch in https://github.com/agones-dev/agones/pull/3500
- Support appProtocol by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3502
- Adds gameserver e2e test for Lists by @igooch in https://github.com/agones-dev/agones/pull/3507
- Adds fleet e2e test for lists by @igooch in https://github.com/agones-dev/agones/pull/3510
- Disable resync on SDK Server by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3508
- Move PodHostName to Stable by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3517
- Adds gameserverallocation e2e tests for Lists by @igooch in https://github.com/agones-dev/agones/pull/3516
- Move FleetAllocationOverflow to Beta by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3515
- Move ResetMetricsOnDelete to Stable by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3518
- Adds fleetauotscaler e2e test for Lists by @igooch in https://github.com/agones-dev/agones/pull/3519
- Another List fleet autoscaler e2e test by @igooch in https://github.com/agones-dev/agones/pull/3521
- Adds Go Conformance Tests for Lists by @igooch in https://github.com/agones-dev/agones/pull/3524
- Move CountsAndLists to Alpha by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3527
- Move SplitControllerAndExtensions to Stable by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3523
- Add clusterIP for agones-allocator in helm chart by @govargo in https://github.com/agones-dev/agones/pull/3526
- GKE Autopilot: Add support for Extended Duration pods by @zmerlynn in https://github.com/agones-dev/agones/pull/3387
- Counter and List Aggregate Fleet Metrics by @igooch in https://github.com/agones-dev/agones/pull/3528
- CountsAndLists: SDK Reference by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3537
- Adds Counters and Lists REST API Conformance Tests by @igooch in https://github.com/agones-dev/agones/pull/3546
- CountsAndLists: Yaml Examples And References by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3552

**Fixed bugs:**
- Xonotic: gLibc incompatibility by @markmandel in https://github.com/agones-dev/agones/pull/3495
- Fixes occasional data race flake with TestSDKServerAddListValue by @igooch in https://github.com/agones-dev/agones/pull/3505
- handle static port policy by @ashutosji in https://github.com/agones-dev/agones/pull/3375
- Prevent Redundant Windows SDK Builds by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3520
- CloudBuild: Fix for cache image rebuild by @markmandel in https://github.com/agones-dev/agones/pull/3535

**Other:**
- Prep for release v1.37.0 by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3493
- Test SuperTuxKart Image by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3496
- Test Rust Image by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3499
- Test cpp-simple image by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3497
- Add steps to update performance test clusters when upgrading k8s version by @gongmax in https://github.com/agones-dev/agones/pull/3501
- Test NodeJS image by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3498
- Bumps simple-game-server version to 0.22 by @igooch in https://github.com/agones-dev/agones/pull/3504
- xonotic image test by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3494
- Bump helm install timeout to 10m by @zmerlynn in https://github.com/agones-dev/agones/pull/3506
- Add Shulker to the Agones adopters list by @jeremylvln in https://github.com/agones-dev/agones/pull/3503
- Remove warning on C# SDK Docs by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3525
- Ensure ci/save_cache and ci/restore_cache images don't get deleted by cleanup policy by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3522
- GH Action: Size label for PRs by @markmandel in https://github.com/agones-dev/agones/pull/3532
- Flake: TestControllerWatchGameServers by @markmandel in https://github.com/agones-dev/agones/pull/3534
- Go CRD Comment Updates for Counters and Lists by @markmandel in https://github.com/agones-dev/agones/pull/3536
- Simple Game Server Example: Upgrade Docker to 24.0.6 by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3531
- CI: Fix 404 on CI link testing by @markmandel in https://github.com/agones-dev/agones/pull/3542
- Xonotic Example: Docker 24.0.6 by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3538
- Bumps simple-game-server to 0.23 by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3543
- Tweaks to Client SDK reference by @markmandel in https://github.com/agones-dev/agones/pull/3541
- Updates to Counter and List Alpha.proto Methods by @igooch in https://github.com/agones-dev/agones/pull/3544
- Docs: SDK implementation matrixes by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3545
- Disable deletion protection for Autopilot test clusters by @gongmax in https://github.com/agones-dev/agones/pull/3468

**New Contributors:**
- @jeremylvln made their first contribution in https://github.com/agones-dev/agones/pull/3503

## [v1.36.0](https://github.com/agones-dev/agones/tree/v1.36.0) (2023-11-07)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.35.0...v1.36.0)

**Breaking changes:**
- Update Supported Kubernetes to 1.26, 1.27, 1.28 by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3450
- Remove 1.25 supported K8s version from e2e cluster by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3451

**Implemented enhancements:**
- Adds CounterActions and ListActions to Allocation.proto by @igooch in https://github.com/agones-dev/agones/pull/3407
- Terraform template file for the performance test cluster by @gongmax in https://github.com/agones-dev/agones/pull/3409
- In the scenario test, submitting request in a fixed interval, exposing more error type by @gongmax in https://github.com/agones-dev/agones/pull/3414
- Adds GameServerAllocation e2e tests for Counters by @igooch in https://github.com/agones-dev/agones/pull/3400
- Adds Counter FleetAutoScaler e2e Test by @igooch in https://github.com/agones-dev/agones/pull/3418
- simple-game-server: Adds a graceful termination delay by @zmerlynn in https://github.com/agones-dev/agones/pull/3436
- add opt-out ttlSecondsAfterFinished setting for the pre-delete hook by @mikeseese in https://github.com/agones-dev/agones/pull/3442
- Add Cloudbuild step to run performance test by using the scenario test framework.  by @gongmax in https://github.com/agones-dev/agones/pull/3429
- Implements UpdateList, AddListValue, and RemoveListValue in the SDK Server by @igooch in https://github.com/agones-dev/agones/pull/3445
- Adds Go SDK Client List Functions by @igooch in https://github.com/agones-dev/agones/pull/3484
- Updates LocalSDK UpdateCounter method by @igooch in https://github.com/agones-dev/agones/pull/3487

**Fixed bugs:**
- Post release: use clone source and update release process by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3408
- Remove `stale` and `obsolete` from PR's on update by @markmandel in https://github.com/agones-dev/agones/pull/3431
- fix: delay deleting GameServers in Error state by @nrwiersma in https://github.com/agones-dev/agones/pull/3428
- Cmake: Ensure find_dependency is on rebuild by @markmandel in https://github.com/agones-dev/agones/pull/3477

**Security fixes:**
- Bump @babel/traverse from 7.20.1 to 7.23.2 in /sdks/nodejs by @dependabot in https://github.com/agones-dev/agones/pull/3433

**Other:**
- Prep for release v1.36.0 by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3406
- Change to use grpc-dotnet instead of Grpc.Core in C# SDK by @yoshd in https://github.com/agones-dev/agones/pull/3397
- Docs for running docker-compose locally with SDK and server by @mbychkowski in https://github.com/agones-dev/agones/pull/3390
- fix: Fixed broken include paths in Unreal Engine plugin. by @KiaArmani in https://github.com/agones-dev/agones/pull/3416
- Docsy Upgrade by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3417
- Bump golang.org/x/net from 0.15.0 to 0.17.0 by @dependabot in https://github.com/agones-dev/agones/pull/3422
- Update Nodejs Apt Repository to latest by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3434
- Update Nodejs Apt Repository to latest by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3435
- Remove NodeJs dependency from RestApi Dockerfile by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3437
- Bump simple-game-server references to 0.19 by @zmerlynn in https://github.com/agones-dev/agones/pull/3439
- Removes flaky TestCounterGameServerAllocationSorting by @igooch in https://github.com/agones-dev/agones/pull/3440
- Flake: TestGameServerAllocationValidate by @markmandel in https://github.com/agones-dev/agones/pull/3443
- Remove Terraform Tests by @markmandel in https://github.com/agones-dev/agones/pull/3441
- Convert shell script to Go by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3413
- Ignore build gcloud config in yamllint by @markmandel in https://github.com/agones-dev/agones/pull/3446
- Update fleet autoscaling limited signification(#2828) by @atgane in https://github.com/agones-dev/agones/pull/3448
- Build and push system image before performance tests by @gongmax in https://github.com/agones-dev/agones/pull/3454
- Update examples/autoscaler-webook dependencies by @markmandel in https://github.com/agones-dev/agones/pull/3447
- Bump examples/allocation-endpoint by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3465
- More cleanup of Terraform Tests by @markmandel in https://github.com/agones-dev/agones/pull/3444
- Fix Various Deprecation Warnings by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3453
- Bump Examples: supertuxkart and xonotic by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3463
- Bump examples/crd-client by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3464
- Bump examples/simple-game-server by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3466
- Golang Version to go1.20.10 by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3475
- Upgrade gRPC version by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3472
- Updates to gRPC generation by @markmandel in https://github.com/agones-dev/agones/pull/3483

**New Contributors:**
- @nrwiersma made their first contribution in https://github.com/agones-dev/agones/pull/3428
- @atgane made their first contribution in https://github.com/agones-dev/agones/pull/3448

## [v1.35.0](https://github.com/agones-dev/agones/tree/v1.35.0) (2023-09-26)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.34.0...v1.35.0)

**Implemented enhancements:**
- Cloud build script for simple-game-server by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3314
- feat: discard disconnected game server streams by @antiphp in https://github.com/agones-dev/agones/pull/3328
- Rust SDK on crates.io by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3332
- restapi generation: clean before generation so we don't leak files by @zmerlynn in https://github.com/agones-dev/agones/pull/3353
- Implements GetCounter and UpdateCounter on the SDK Server by @igooch in https://github.com/agones-dev/agones/pull/3322
- Adds Go SDK client Counter functions by @igooch in https://github.com/agones-dev/agones/pull/3372
- Update Go simple-game-server to have commands for Counter SDK methods by @igooch in https://github.com/agones-dev/agones/pull/3378
- Adds GameServer e2e tests for Counters by @igooch in https://github.com/agones-dev/agones/pull/3381
- Updates Fleet and GameServerSet CRDs by @igooch in https://github.com/agones-dev/agones/pull/3396
- Add conformance test implementation for C# SDK by @yoshd in https://github.com/agones-dev/agones/pull/3392
- Adds fleet e2e test for Counter by @igooch in https://github.com/agones-dev/agones/pull/3399

**Fixed bugs:**
- Added TF DNS config options to prevent Autopilot destroy / create on existing cluster by @abmarcum in https://github.com/agones-dev/agones/pull/3330
- Fix site-server target by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3335
- Do not refresh cache if no update by @gongmax in https://github.com/agones-dev/agones/pull/3343
- bump: joonix/log to NewFormater() by @jonsch318 in https://github.com/agones-dev/agones/pull/3342
- Fixes TestGameServerResourceValidation flake by @igooch in https://github.com/agones-dev/agones/pull/3373
- Get the gs state correctly in error message by @gongmax in https://github.com/agones-dev/agones/pull/3385
- Reduce controller memory footprint considerably by @markmandel in https://github.com/agones-dev/agones/pull/3394

**Other:**
- Preparation for v1.35.0 by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3326
- Update Agones release guide url by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3329
- Improve SDK Conformance error reporting by @markmandel in https://github.com/agones-dev/agones/pull/3331
- Catch up C++ SDK to `make gen-all-sdk-grpc` by @zmerlynn in https://github.com/agones-dev/agones/pull/3337
- SDK Conformance: Use -test consistently instead of -no-build by @zmerlynn in https://github.com/agones-dev/agones/pull/3340
- fix of helm installation command in doc by @ashutosji in https://github.com/agones-dev/agones/pull/3333
- Update release version on Agones website by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3338
- Generate certs on TestFleetAutoscalerTLSWebhook by @markmandel in https://github.com/agones-dev/agones/pull/3350
- Verify gen-all-sdk-grpc has run by @zmerlynn in https://github.com/agones-dev/agones/pull/3349
- Update Rust document by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3336
- Yaml linter by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3358
- Update release checklist by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3334
- Add Octops Fleet Garbage Collector project to third party docs by @danieloliveira079 in https://github.com/agones-dev/agones/pull/3359
- Updates to GKE Terraform docs by @joeholley in https://github.com/agones-dev/agones/pull/3360
- Fix unaccurate progress description of HA Agones by @gongmax in https://github.com/agones-dev/agones/pull/3364
- Bump GitHub workflow actions to latest versions by @deining in https://github.com/agones-dev/agones/pull/3355
- dependency: bump github.com/grpc-ecosystem/grpc-gateway/v2 from v2.15.0 to v2.17.1 by @aimuz in https://github.com/agones-dev/agones/pull/3366
- Update: Allocation Overflow Documentation by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3368
- Bumps simple-game-server version to 0.18 by @igooch in https://github.com/agones-dev/agones/pull/3379
- Upgrade Hugo by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3369
- Add more items to .gcloudignore by @markmandel in https://github.com/agones-dev/agones/pull/3383
- Don't log when a pod can't be found on startup by @markmandel in https://github.com/agones-dev/agones/pull/3393
- Fix typo in examples/simple-game-server/README.md by @markmandel in https://github.com/agones-dev/agones/pull/3398

**New Contributors:**
- @antiphp made their first contribution in https://github.com/agones-dev/agones/pull/3328
- @jonsch318 made their first contribution in https://github.com/agones-dev/agones/pull/3342
- @deining made their first contribution in https://github.com/agones-dev/agones/pull/3355

## [v1.34.0](https://github.com/agones-dev/agones/tree/v1.34.0) (2023-08-15)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.33.0...v1.34.0)

**Breaking changes:**
- refactor: Throwing error messages with field. by @aimuz in https://github.com/agones-dev/agones/pull/3239
- refactor: apihook ValidateGameServerSpec and ValidateScheduling use field.ErrorList by @aimuz in https://github.com/agones-dev/agones/pull/3255
- Update Supported Kubernetes to 1.25, 1.26, 1.27 by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3261
- refactor(allocation): Reimplement the Validate method using "field.ErrorList" by @aimuz in https://github.com/agones-dev/agones/pull/3259
- refactor: FleetAutoscaler Validate use field.ErrorList by @aimuz in https://github.com/agones-dev/agones/pull/3272

**Implemented enhancements:**
- Graduate CustomFasSyncInterval To Stable by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3235
- Alpine ➡️ Distroless as Agones base image by @markmandel in https://github.com/agones-dev/agones/pull/3270
- Adds Counts and Lists AutoScale Policies by @igooch in https://github.com/agones-dev/agones/pull/3211
- More Local Dev Server Support by @CauhxMilloy in https://github.com/agones-dev/agones/pull/3252
- GameServerAllocation to sort Priorities by Allocated Capacity by @igooch in https://github.com/agones-dev/agones/pull/3282
- Add Node.Status.Address to GameServer.Status in CRD and SDK by @zmerlynn in https://github.com/agones-dev/agones/pull/3299
- Add GameServer addresses to the allocation APIs by @zmerlynn in https://github.com/agones-dev/agones/pull/3307
- Cloud Build Script for supertuxkart by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3291
- Add "Choosing a GCP network" to GKE Cluster Creation by @zmerlynn in https://github.com/agones-dev/agones/pull/3311
- Cloud Build script for crd-client by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3290
- Cloud build script for rust-simple by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3313
- Cloudbuild script for autoscaler-webhook by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3298
- update xonotic example to 0.8.6 by @ashutosji in https://github.com/agones-dev/agones/pull/3273
- Cloud Build script for allocation-endpoint by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3289
- Cloud build script for nodejs-simple by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3312
- Cloud build script for Xonotic image by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3317
- Graduate StateAllocationFilter to Stable by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3308
- Sort by Priority for strategy Distributed by @igooch in https://github.com/agones-dev/agones/pull/3296
- Build Script for cpp-simple by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3320

**Fixed bugs:**
- fix: Enabling SplitControllerAndExtensions leads to incomplete metrics availability by @aimuz in https://github.com/agones-dev/agones/pull/3242
- Race Flake in TestControllerSyncFleetAutoscaler() by @markmandel in https://github.com/agones-dev/agones/pull/3260
- Use maintenance exclusion to prevent auto-upgrade, add 1.27 test clusters by @gongmax in https://github.com/agones-dev/agones/pull/3253
- SDK WatchGameServer logs error on shutdown by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3271
- APIService: Updates to handlers for 1.27.x by @markmandel in https://github.com/agones-dev/agones/pull/3297
- [Helm Chart] Only enable service monitor relabelings for prometheus scrape when prometheusServiceDiscovery is enabled by @ufou in https://github.com/agones-dev/agones/pull/3285
- Flaky: TestAllocatorAllocateOnGameServerUpdateError by @markmandel in https://github.com/agones-dev/agones/pull/3300
- Run `make gen-all-sdk-grpc` by @zmerlynn in https://github.com/agones-dev/agones/pull/3301
- Fix for scaling split allocated GameServerSets by @markmandel in https://github.com/agones-dev/agones/pull/3292
- Flaky: TestAllocatorAllocateOnGameServerUpdateError by @markmandel in https://github.com/agones-dev/agones/pull/3306
- Bugs and Improvements for CPP SDK and Example by @markmandel in https://github.com/agones-dev/agones/pull/3318

**Other:**
- Preparation for 1.34.0 by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3248
- fix: Label PR Warning by @aimuz in https://github.com/agones-dev/agones/pull/3241
- Put e2e Cloud Build logs in public bucket by @markmandel in https://github.com/agones-dev/agones/pull/3250
- cleanup: Add agones-extensions Image by @aimuz in https://github.com/agones-dev/agones/pull/3256
- Release Checklist Cleanup by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3258
- Cleanup Labeler by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3257
- fix: info to debug level by @aimuz in https://github.com/agones-dev/agones/pull/3265
- refactor: Switch Helm Cleanup job to use bitnami/kubectl image by @aimuz in https://github.com/agones-dev/agones/pull/3263
- Remove e2e cluster with oldest supported K8s version by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3267
- Flake: TestControllerAllocator by @markmandel in https://github.com/agones-dev/agones/pull/3264
- Upgrade Version of Rust by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3268
- Some copy edits to the most recent release blog post. by @roberthbailey in https://github.com/agones-dev/agones/pull/3275
- Fix Dependabot Vulnerability by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3269
- Upgrade site to Google Analytics 4 by @markmandel in https://github.com/agones-dev/agones/pull/3278
- Flaky: TestAllocatorAllocatePriority by @markmandel in https://github.com/agones-dev/agones/pull/3280
- Move simple-game-server to Distroless base by @markmandel in https://github.com/agones-dev/agones/pull/3279
- TestAllocatorAllocateOnGameServerUpdateError by @markmandel in https://github.com/agones-dev/agones/pull/3283
- Switching autoscaler-webhook to utilize distroless as base Image by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3276
- Distroless base image for crd-client by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3277
- Flake TestAllocatorAllocateOnGameServerUpdateError by @markmandel in https://github.com/agones-dev/agones/pull/3295
- Updates for Terraform by @markmandel in https://github.com/agones-dev/agones/pull/3293
- Bring Rust SDK dependencies up to date by @markmandel in https://github.com/agones-dev/agones/pull/3305
- Add note about which namespace is used for game serves deployed from fleets by @mikeseese in https://github.com/agones-dev/agones/pull/3288
- Condition check for no content in PR by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3304
- Update close.yaml by @geetachavan1 in https://github.com/agones-dev/agones/pull/3316
- Fix inaccurate parameter description by @gongmax in https://github.com/agones-dev/agones/pull/3321

**New Contributors:**
- @CauhxMilloy made their first contribution in https://github.com/agones-dev/agones/pull/3252
- @ufou made their first contribution in https://github.com/agones-dev/agones/pull/3285
- @mikeseese made their first contribution in https://github.com/agones-dev/agones/pull/3288

## [v1.33.0](https://github.com/agones-dev/agones/tree/v1.33.0) (2023-07-05)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.32.0...v1.33.0)

**Implemented enhancements:**
- Cloud Build config to trigger a build if no build is running by @zmerlynn in https://github.com/agones-dev/agones/pull/3174
- Add a helm flag to force Agones system components onto dedicated nodes by @gongmax in https://github.com/agones-dev/agones/pull/3161
- Counts and Lists Aggregate Values for Fleet Status and GameServerSet Status by @igooch in https://github.com/agones-dev/agones/pull/3180
- [Release Automation] Label PRs with GitHub Action by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3151
- Add make gen-crd-clients to the CI suite by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3196
- Adds Counters and Lists to FleetAutoScaler CRD by @igooch in https://github.com/agones-dev/agones/pull/3198
- Expose GameServer types by @MiniaczQ in https://github.com/agones-dev/agones/pull/3205
- Label PR by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3216
- Remove Feature Expiry Version Shortcodes by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3210
- Add labels and annotations to allocation response by @austin-space in https://github.com/agones-dev/agones/pull/3197
- Update Version in site/config.toml by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3227
- Move SDKGracefulTermination To Stable by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3231
- Delete data-proofer-ignore attribute by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3225
- GKE Autopilot: Add primary container annotation for game server container by @zmerlynn in https://github.com/agones-dev/agones/pull/3234
- Fix goclient request dashboard and add allocator to the drill down; Change goclient workqueue dashboard y axis unit by @gongmax in https://github.com/agones-dev/agones/pull/3240

**Fixed bugs:**
- Fix container name conflict when build windows image by @gongmax in https://github.com/agones-dev/agones/pull/3195
- Have leader election use namespace from env var by @chiayi in https://github.com/agones-dev/agones/pull/3209
- Make sdk-update-version by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3221
- Add label changes for service-monitor by @chiayi in https://github.com/agones-dev/agones/pull/3201

**Other:**
- Preparation for next release v1.33.0 by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3181
- Run e2e test on regional standard clusters by @gongmax in https://github.com/agones-dev/agones/pull/3182
- Remove zonal test clusters, and create regional clusters with release channel by @gongmax in https://github.com/agones-dev/agones/pull/3186
- Update GKE installation instructions now that `SplitControllerAndExtensions` has been enabled by default. by @roberthbailey in https://github.com/agones-dev/agones/pull/3191
- build: add ltsc2022 target for windows builds by @davidedmondsMPG in https://github.com/agones-dev/agones/pull/3187
- Remove Rolling Update on Ready warning in docs by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3192
- Add write permission to id-token by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3207
- remove old warning of conversion int64 to float64 by @ashutosji in https://github.com/agones-dev/agones/pull/3214
- Docs: Break up Helm configuration table by @markmandel in https://github.com/agones-dev/agones/pull/3215
- Change pre-release version to MAJOR.MINOR.PATCH-dev-HASH by @gongmax in https://github.com/agones-dev/agones/pull/3219
- Change the helm config field `agones.system.requireDedicatedNode` to `agones.requireDedicatedNodes` by @gongmax in https://github.com/agones-dev/agones/pull/3226
- Potential fix for TestAllocatorAllocate* flakiness by @markmandel in https://github.com/agones-dev/agones/pull/3232
- Fix Unreal Engine SDK page for UE5 information. by @oniku-2929 in https://github.com/agones-dev/agones/pull/3237

**New Contributors:**
- @davidedmondsMPG made their first contribution in https://github.com/agones-dev/agones/pull/3187
- @ashutosji made their first contribution in https://github.com/agones-dev/agones/pull/3214

## [v1.32.0](https://github.com/agones-dev/agones/tree/v1.32.0) (2023-05-23)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.31.0...v1.32.0)

**Implemented enhancements:**
- Release Automation: Push images on cloud by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3090
- Sort By Counters or Lists during GameServerAllocation 2716 by @igooch in https://github.com/agones-dev/agones/pull/3091
- Push-Chart to Helm Repo on GCS by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3114
- Allocated GameServers updated on Fleet update by @markmandel in https://github.com/agones-dev/agones/pull/3101
- require.NoError in fleet tests instead of continuing by @zmerlynn in https://github.com/agones-dev/agones/pull/3124
- Move PodHostName to Beta by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3118
- Creating a branch for release by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3127
- Documentation: Allocated GameServer Overflow by @markmandel in https://github.com/agones-dev/agones/pull/3131
- Move make release-deploy-site into pre-build-release by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3132
- Upgrade to Golang version 1.20.4 by @igooch in https://github.com/agones-dev/agones/pull/3137
- Added labels to the agones.allocator by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3125
- GameServerAllocation Actions for Counters and Lists by @igooch in https://github.com/agones-dev/agones/pull/3117
- Graduate SafeToEvict to GA by @zmerlynn in https://github.com/agones-dev/agones/pull/3146
- Move ResetMetricsOnDelete to Beta by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3154
- [Release Automation] Update Helm/SDK/Install Packages Version Numbers by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3149
- Allocation.proto Updates for Counts and Lists by @igooch in https://github.com/agones-dev/agones/pull/3150
- Add parametric host address by @MiniaczQ in https://github.com/agones-dev/agones/pull/3111
- Allow setting a static NodePort for the ping service by @towolf in https://github.com/agones-dev/agones/pull/3148
- Promote SplitControllerAndExtensions to beta by @chiayi in https://github.com/agones-dev/agones/pull/3165

**Fixed bugs:**
- Revert #3070, wait on networking a different way by @zmerlynn in https://github.com/agones-dev/agones/pull/3107
- Make migration controller more forgiving wrt Node/GameServer addresses by @luckyswede in https://github.com/agones-dev/agones/pull/3116
- Docs: Fix some bugs in the feature gate page by @markmandel in https://github.com/agones-dev/agones/pull/3136
- Fix an invalid xonotic-example image path by @gongmax in https://github.com/agones-dev/agones/pull/3139
- Add a more graceful termination to Allocator by @chiayi in https://github.com/agones-dev/agones/pull/3105
- GraceTermination when GameServer get deleted by @qizichao-dm in https://github.com/agones-dev/agones/pull/3141
- Update stale.yaml by @geetachavan1 in https://github.com/agones-dev/agones/pull/3147
- Ignore twitter link in html tests by @gongmax in https://github.com/agones-dev/agones/pull/3158
- sdkserver: When waitForConnection fails, container should restart quickly by @zmerlynn in https://github.com/agones-dev/agones/pull/3157
- Move back to FailureThreshold failures of /gshealthz by @zmerlynn in https://github.com/agones-dev/agones/pull/3160
- Add fix for one issue with TestFleetRecreateGameServers test by @chiayi in https://github.com/agones-dev/agones/pull/3163

**Other:**
- Preparation for 1.32.0 by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3086
- Add to SplitControllerAndExtensions documentation for leader election by @chiayi in https://github.com/agones-dev/agones/pull/3083
- Update docs for Stable Network ID by @markmandel in https://github.com/agones-dev/agones/pull/3088
- Drop log level of worker queue to Trace by @zmerlynn in https://github.com/agones-dev/agones/pull/3092
- refactor: type and constant definitions are in the same area. by @aimuz in https://github.com/agones-dev/agones/pull/3102
- Remove consul install by @zmerlynn in https://github.com/agones-dev/agones/pull/3104
- Specify the machine type for agones-metrics nodepool since  the default one doesn't meet resource requirement by @gongmax in https://github.com/agones-dev/agones/pull/3109
- Clone Agones for release targets by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3119
- Fix broken link by @gongmax in https://github.com/agones-dev/agones/pull/3123
- Move PushChart into releaseFile by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3120
- Refactor: Modify logger implementation and log level by @aimuz in https://github.com/agones-dev/agones/pull/3103
- Remove unused target for generating change log by @gongmax in https://github.com/agones-dev/agones/pull/3126
- Docs: Remove contributing warning about bug. by @markmandel in https://github.com/agones-dev/agones/pull/3130
- Quilkin added in third-party-content/examples by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3129
- Remove milestone steps from release by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3135
- Bump example image versions by @igooch in https://github.com/agones-dev/agones/pull/3138
- Add allocator readiness configurations doc by @chiayi in https://github.com/agones-dev/agones/pull/3142
- Update values yaml file for `SplitControllerAndExtensions` by @chiayi in https://github.com/agones-dev/agones/pull/3153
- Always pull development images when running `make install`. by @roberthbailey in https://github.com/agones-dev/agones/pull/3162
- Add Cloud Best Practices guide, add guide on Release Channels by @zmerlynn in https://github.com/agones-dev/agones/pull/3152
- Suppress full e2e logs so the per-configuration links are obvious by @zmerlynn in https://github.com/agones-dev/agones/pull/3164
- Strengthen the warning about reusing certificates in the yaml installation. by @roberthbailey in https://github.com/agones-dev/agones/pull/3167
- Add docs for #3148 by @zmerlynn in https://github.com/agones-dev/agones/pull/3173

**New Contributors:**
- @luckyswede made their first contribution in https://github.com/agones-dev/agones/pull/3116
- @qizichao-dm made their first contribution in https://github.com/agones-dev/agones/pull/3141
- @MiniaczQ made their first contribution in https://github.com/agones-dev/agones/pull/3111
- @towolf made their first contribution in https://github.com/agones-dev/agones/pull/3148

## [v1.31.0](https://github.com/agones-dev/agones/tree/v1.31.0) (2023-04-11)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.30.0...v1.31.0)

**Breaking changes:**
- Update Supported Kubernetes to 1.24 1.25 1.26 by @gongmax in https://github.com/agones-dev/agones/pull/3029

**Implemented enhancements:**
- Add automation to report on recent build flakes by @zmerlynn in https://github.com/agones-dev/agones/pull/3012
- Fix GKE Autopilot auto-detection for 1.26 by @zmerlynn in https://github.com/agones-dev/agones/pull/3032
- Adds Counter to SDK alpha.proto by @igooch in https://github.com/agones-dev/agones/pull/3002
- Add leader election feature to `agones-controller` by @chiayi in https://github.com/agones-dev/agones/pull/3025
- Adds List to SDK alpha.proto by @igooch in https://github.com/agones-dev/agones/pull/3039
- Link to Global Scale Demo from Agones Examples page by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3064
- Add timeout to SDK k8s client by @zmerlynn in https://github.com/agones-dev/agones/pull/3070
- Add helm setting for leader election by @chiayi in https://github.com/agones-dev/agones/pull/3051
- Have TestPlayerConnectWithCapacityZero use framework to wait by @zmerlynn in https://github.com/agones-dev/agones/pull/3062
- Retry build cancellation if it fails by @zmerlynn in https://github.com/agones-dev/agones/pull/3073
- GitHub action for stale issues by @geetachavan1 in https://github.com/agones-dev/agones/pull/3075
- GameServer Allocation Filtering for Counts and Lists by @igooch in https://github.com/agones-dev/agones/pull/3065
- Automation: Update Approved Auto-Merge PR's to latest main by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3066
- Add e2e test for leader election by @chiayi in https://github.com/agones-dev/agones/pull/3076

**Fixed bugs:**
- Ensure the state bucket exists before creating e2e test clusters by @gongmax in https://github.com/agones-dev/agones/pull/3014
- Add Sigterm handler and readiness probe to extensions by @chiayi in https://github.com/agones-dev/agones/pull/3011
- Use actually distinct webhook for Autopilot by @zmerlynn in https://github.com/agones-dev/agones/pull/3035
- Changes to resolve error in creating gcloud-e2e-test-cluster by @igooch in https://github.com/agones-dev/agones/pull/3040
- Replaces functionality and types to make plugin cross-compilable between UE4 and UE5 by @DevChagrins in https://github.com/agones-dev/agones/pull/3060
- Rework game server health initial delay handling by @zmerlynn in https://github.com/agones-dev/agones/pull/3046
- Fix simple-game-server to use context substitute for the infinite loop by @oniku-2929 in https://github.com/agones-dev/agones/pull/3050
- Added -buildvcs=false in build/Makefile by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3069
- Rework game server health initial delay handling by @zmerlynn in https://github.com/agones-dev/agones/pull/3072

**Other:**
- Prep for 1.31.0 release by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3013
- Clarify instructions for Managed Prometheus by @zmerlynn in https://github.com/agones-dev/agones/pull/3015
- Delete unused e2e test cluster by @gongmax in https://github.com/agones-dev/agones/pull/3017
- Add autopilot instructions to doc as Alpha by @shannonxtreme in https://github.com/agones-dev/agones/pull/3004
- Removing dzlier-gcp from approvers list. by @dzlier-gcp in https://github.com/agones-dev/agones/pull/3021
- Fix Dependabot vulnerabilites by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3027
- Update _index.md by @deibu in https://github.com/agones-dev/agones/pull/3045
- Fix doc for multiple k8s version support by @gongmax in https://github.com/agones-dev/agones/pull/3038
- Helm test instruction cleanup in Agones doc by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3052
- Add licence to cancelot.sh by @markmandel in https://github.com/agones-dev/agones/pull/3055
- Generate release notes and Changelog using Github by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3022
- Fixed example images by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3063
- Extend e2e queue timings / Disable testing on Autopilot 1.26 by @zmerlynn in https://github.com/agones-dev/agones/pull/3059
- Revert "Rework game server health initial delay handling (#3046)" by @zmerlynn in https://github.com/agones-dev/agones/pull/3068
- Document missing Allocation Service helm variables by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3053
- Remove unnecessary intermediate variables. by @roberthbailey in https://github.com/agones-dev/agones/pull/3056
- Add description on when to upgrade supported Kubernetes version by @gongmax in https://github.com/agones-dev/agones/pull/3049
- Fix release tag on Unity SDK installation document page (#2622) by @oniku-2929 in https://github.com/agones-dev/agones/pull/3071
- Compilation errors on simple-game-server by @markmandel in https://github.com/agones-dev/agones/pull/3054
- Add tags for cluster, location, commit to e2e-test builds by @zmerlynn in https://github.com/agones-dev/agones/pull/3074
- Update examples images to latest version on agones-images by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3077
- Tag CI build with commit, tag e2e with parent build ID by @zmerlynn in https://github.com/agones-dev/agones/pull/3080
- Renamed action-secret by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3081
- simple-game-server with latest version 0.15 by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3078
- Remove changelog generation from release/cloudbuild.yaml by @Kalaiselvi84 in https://github.com/agones-dev/agones/pull/3079
- Remove 1.23 e2e test cluster by @gongmax in https://github.com/agones-dev/agones/pull/3082

**New Contributors:**
- @shannonxtreme made their first contribution in https://github.com/agones-dev/agones/pull/3004
- @deibu made their first contribution in https://github.com/agones-dev/agones/pull/3045
- @DevChagrins made their first contribution in https://github.com/agones-dev/agones/pull/3060
- @oniku-2929 made their first contribution in https://github.com/agones-dev/agones/pull/3050
- @geetachavan1 made their first contribution in https://github.com/agones-dev/agones/pull/3075

## [v1.30.0](https://github.com/agones-dev/agones/tree/v1.30.0) (2023-03-01)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.29.0...v1.30.0)

**Breaking changes:**

- Add error callback to testSDKClient [\#2964](https://github.com/agones-dev/agones/pull/2964) ([steven-supersolid](https://github.com/steven-supersolid))

**Implemented enhancements:**

- TypeScript types for Node SDK [\#2937](https://github.com/agones-dev/agones/issues/2937)
- Graduate `SafeToEvict` to Beta [\#2931](https://github.com/agones-dev/agones/issues/2931)
- Upgrade windows node image on GKE [\#2582](https://github.com/agones-dev/agones/issues/2582)
- Player Tracking for each GameServer [\#1033](https://github.com/agones-dev/agones/issues/1033)
- Add extensions status dash and dash name change [\#3000](https://github.com/agones-dev/agones/pull/3000) ([chiayi](https://github.com/chiayi))
- Add variable to go-client request dashboard [\#2998](https://github.com/agones-dev/agones/pull/2998) ([chiayi](https://github.com/chiayi))
- Add more time and logging to extensions test [\#2996](https://github.com/agones-dev/agones/pull/2996) ([chiayi](https://github.com/chiayi))
- Add Extensions Resource Dashboard [\#2993](https://github.com/agones-dev/agones/pull/2993) ([chiayi](https://github.com/chiayi))
- Add integration guide for Google Cloud Managed Service for Prometheus [\#2990](https://github.com/agones-dev/agones/pull/2990) ([zmerlynn](https://github.com/zmerlynn))
- Added back metrics support for extensions [\#2988](https://github.com/agones-dev/agones/pull/2988) ([chiayi](https://github.com/chiayi))
- Adds default values for Counters and Lists fields [\#2983](https://github.com/agones-dev/agones/pull/2983) ([igooch](https://github.com/igooch))
- Kubernetes Update template, Release template, and site content change for multiple k8s version support [\#2980](https://github.com/agones-dev/agones/pull/2980) ([gongmax](https://github.com/gongmax))
- CRDs for AllocationOverflow [\#2979](https://github.com/agones-dev/agones/pull/2979) ([markmandel](https://github.com/markmandel))
- More "packed" behavior when reducing GameServerSet replicas. [\#2974](https://github.com/agones-dev/agones/pull/2974) ([castaneai](https://github.com/castaneai))
- Create e2e test clusters in different regions to mitigate quota limit issue [\#2969](https://github.com/agones-dev/agones/pull/2969) ([gongmax](https://github.com/gongmax))
- Run e2e tests on multiple clusters with different versions [\#2968](https://github.com/agones-dev/agones/pull/2968) ([gongmax](https://github.com/gongmax))
- Pre-Alpha Feature Gate: FleetAllocationOverflow [\#2967](https://github.com/agones-dev/agones/pull/2967) ([markmandel](https://github.com/markmandel))
- Updates counters lists schema status CRDs for gameservers fleets [\#2965](https://github.com/agones-dev/agones/pull/2965) ([igooch](https://github.com/igooch))
- Create e2e test clusters with multiple k8s versions [\#2962](https://github.com/agones-dev/agones/pull/2962) ([gongmax](https://github.com/gongmax))
- Add e2e test for Extensions [\#2947](https://github.com/agones-dev/agones/pull/2947) ([chiayi](https://github.com/chiayi))
- Arbitrary Counts and Lists Feature/CRD [\#2946](https://github.com/agones-dev/agones/pull/2946) ([igooch](https://github.com/igooch))
- Add types for nodejs sdk [\#2940](https://github.com/agones-dev/agones/pull/2940) ([vasily-polonsky](https://github.com/vasily-polonsky))
- Use GCS as the Terraform state backend [\#2938](https://github.com/agones-dev/agones/pull/2938) ([gongmax](https://github.com/gongmax))
- Disable consul locking if consul is not present [\#2934](https://github.com/agones-dev/agones/pull/2934) ([zmerlynn](https://github.com/zmerlynn))
- Allocation Endpoint: Fix Makefile to correctly call docker build  [\#2933](https://github.com/agones-dev/agones/pull/2933) ([abmarcum](https://github.com/abmarcum))
- Added GKE Workload Identity flag to GKE Terraform modules [\#2928](https://github.com/agones-dev/agones/pull/2928) ([abmarcum](https://github.com/abmarcum))
- Add doc for "Controlling Disruption", document `SafeToEvict` [\#2924](https://github.com/agones-dev/agones/pull/2924) ([zmerlynn](https://github.com/zmerlynn))
- Enable SplitControllerAndExtensions for e2e testing and changed valueâ€¦ [\#2923](https://github.com/agones-dev/agones/pull/2923) ([chiayi](https://github.com/chiayi))
- Allow 30m more to acquire consul lock [\#2922](https://github.com/agones-dev/agones/pull/2922) ([zmerlynn](https://github.com/zmerlynn))
- Add GKE Autopilot to e2e [\#2913](https://github.com/agones-dev/agones/pull/2913) ([zmerlynn](https://github.com/zmerlynn))
- GKE Autopilot: Add terraform module, users [\#2912](https://github.com/agones-dev/agones/pull/2912) ([zmerlynn](https://github.com/zmerlynn))
- e2e: Use gotestsum for CI e2e runs, change e2e ARGS handling [\#2904](https://github.com/agones-dev/agones/pull/2904) ([zmerlynn](https://github.com/zmerlynn))
- Wait on free ports on GKE Autopilot [\#2901](https://github.com/agones-dev/agones/pull/2901) ([zmerlynn](https://github.com/zmerlynn))
- controller/extensions: Add explicit requests/limits for ephemeral storage [\#2900](https://github.com/agones-dev/agones/pull/2900) ([zmerlynn](https://github.com/zmerlynn))
- Validate the .scheduling field in the fleet and gameserverset [\#2892](https://github.com/agones-dev/agones/pull/2892) ([zmerlynn](https://github.com/zmerlynn))
- Trim down `agones-extensions` and add flag to `agones-controller` [\#2891](https://github.com/agones-dev/agones/pull/2891) ([chiayi](https://github.com/chiayi))

**Fixed bugs:**

- Unreal Engine - non-ascii characters in annotations breaks websocket. [\#2976](https://github.com/agones-dev/agones/issues/2976)
- Node.js SDK test is flaky [\#2954](https://github.com/agones-dev/agones/issues/2954)
- Install Agones using YAML doesn't work [\#2935](https://github.com/agones-dev/agones/issues/2935)
- Omit namepace Helm install includes [\#2920](https://github.com/agones-dev/agones/issues/2920)
- README.md refers to katacoda.com is now closed by Oâ€™Reilly [\#2907](https://github.com/agones-dev/agones/issues/2907)
- The Documentation/Installation page on old releases shows an incorrect Kubernetes version [\#2279](https://github.com/agones-dev/agones/issues/2279)
- Fix HandleWatchMessage [\#2977](https://github.com/agones-dev/agones/pull/2977) ([tvandijck](https://github.com/tvandijck))
- Fix for `make gen-crd-client` [\#2971](https://github.com/agones-dev/agones/pull/2971) ([markmandel](https://github.com/markmandel))
- Fix the broken example yaml [\#2956](https://github.com/agones-dev/agones/pull/2956) ([gongmax](https://github.com/gongmax))
- Omit namespace from cluster scopped resources in helm install [\#2925](https://github.com/agones-dev/agones/pull/2925) ([mbychkowski](https://github.com/mbychkowski))
- Adds snapshot Hugo env to separate from default env [\#2914](https://github.com/agones-dev/agones/pull/2914) ([igooch](https://github.com/igooch))
- flaky/TestFleetRollingUpdate [\#2902](https://github.com/agones-dev/agones/pull/2902) ([markmandel](https://github.com/markmandel))

## [v1.29.0](https://github.com/agones-dev/agones/tree/v1.29.0) (2023-01-17)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.29.0-rc...v1.29.0)

**Closed issues:**

- Release 1.29.0-rc [\#2897](https://github.com/agones-dev/agones/issues/2897)

## [v1.29.0-rc](https://github.com/agones-dev/agones/tree/v1.29.0-rc) (2023-01-11)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.28.0...v1.29.0-rc)

**Breaking changes:**

- Update Kubernetes to 1.24 [\#2867](https://github.com/agones-dev/agones/issues/2867)
- Migrate from github.com/golang/protobuf to google.golang.org/protobuf [\#2786](https://github.com/agones-dev/agones/pull/2786) ([govargo](https://github.com/govargo))

**Implemented enhancements:**

- Graduate SDKGracefulTermination to beta [\#2831](https://github.com/agones-dev/agones/issues/2831)
- Set the hostName of the Pod to the name of the GameServer [\#2704](https://github.com/agones-dev/agones/issues/2704)
- Update from golang/protobuf to google.golang.org/protobuf [\#2462](https://github.com/agones-dev/agones/issues/2462)
- Release Automation: Add cloud build target for release builds [\#2460](https://github.com/agones-dev/agones/issues/2460)
- Release Automation: Generate version of website to push [\#2457](https://github.com/agones-dev/agones/issues/2457)
- Consider moving agones system images from gcr.io to GCP's artifact registry [\#2358](https://github.com/agones-dev/agones/issues/2358)
- CI builds should publish a multi-arch manifest for the agones-sdk image [\#2280](https://github.com/agones-dev/agones/issues/2280)
- Generate Changelog - Release Automation: Add cloud build target for release builds [\#2884](https://github.com/agones-dev/agones/pull/2884) ([mangalpalli](https://github.com/mangalpalli))
- GameServer Pod: Stable Network ID [\#2826](https://github.com/agones-dev/agones/pull/2826) ([markmandel](https://github.com/markmandel))
- Release Automation: Generate version of website to push [\#2808](https://github.com/agones-dev/agones/pull/2808) ([mangalpalli](https://github.com/mangalpalli))

**Fixed bugs:**

- Check linter settings for exported symbols [\#2873](https://github.com/agones-dev/agones/issues/2873)
- GameServerAllocation example yaml file has incorrect format for selectors [\#2853](https://github.com/agones-dev/agones/issues/2853)
- Invalid warnings when using multi-cluster allocation [\#2498](https://github.com/agones-dev/agones/issues/2498)
- Update metrics documentation [\#1851](https://github.com/agones-dev/agones/issues/1851)
- GameServerTemplate validation: no description when used big port values [\#1770](https://github.com/agones-dev/agones/issues/1770)
- Inline JSON: GameServerAllocation v1.LabelSelector [\#2877](https://github.com/agones-dev/agones/pull/2877) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- Collaborator Request [\#2861](https://github.com/agones-dev/agones/issues/2861)
- Release 1.28.0 [\#2851](https://github.com/agones-dev/agones/issues/2851)
- Docs: Rename "Stackdriver" to "Cloud Monitoring" [\#2850](https://github.com/agones-dev/agones/issues/2850)

**Merged pull requests:**

- Fix the json5 vulnerabilities [\#2896](https://github.com/agones-dev/agones/pull/2896) ([gongmax](https://github.com/gongmax))
- Update Kubernetes version to 1.24 [\#2895](https://github.com/agones-dev/agones/pull/2895) ([gongmax](https://github.com/gongmax))
- Update aws-sdk-go version to latest [\#2894](https://github.com/agones-dev/agones/pull/2894) ([gongmax](https://github.com/gongmax))
- e2e framework: Allow variable timing based on cloud product [\#2893](https://github.com/agones-dev/agones/pull/2893) ([zmerlynn](https://github.com/zmerlynn))
- Don't run cloud product GameServerSpec validation on development GameServers [\#2889](https://github.com/agones-dev/agones/pull/2889) ([zmerlynn](https://github.com/zmerlynn))
- e2e: Add --cloud-product flag, add SkipOnCloudProduct [\#2886](https://github.com/agones-dev/agones/pull/2886) ([zmerlynn](https://github.com/zmerlynn))
- Set seccompProfile of `Unconfined` on Autopilot unless overidden by user [\#2885](https://github.com/agones-dev/agones/pull/2885) ([zmerlynn](https://github.com/zmerlynn))
- Updates allocation load testing documentation [\#2883](https://github.com/agones-dev/agones/pull/2883) ([igooch](https://github.com/igooch))
- Revert workload separation for Autopilot [\#2876](https://github.com/agones-dev/agones/pull/2876) ([zmerlynn](https://github.com/zmerlynn))
- Move all actual Agones releases images to GAR [\#2875](https://github.com/agones-dev/agones/pull/2875) ([gongmax](https://github.com/gongmax))
- lint: Reenable `revive` [\#2874](https://github.com/agones-dev/agones/pull/2874) ([zmerlynn](https://github.com/zmerlynn))
- cleanup: clean up make\(map\[string\]string, 1\) [\#2872](https://github.com/agones-dev/agones/pull/2872) ([aimuz](https://github.com/aimuz))
- NewFilteredSharedInformerFactory use NewSharedInformerFactoryWithOptions instead [\#2871](https://github.com/agones-dev/agones/pull/2871) ([aimuz](https://github.com/aimuz))
- Update restapi conformance-test [\#2869](https://github.com/agones-dev/agones/pull/2869) ([govargo](https://github.com/govargo))
- cloudproduct: Register API hooks, move validation/mutation to API [\#2868](https://github.com/agones-dev/agones/pull/2868) ([zmerlynn](https://github.com/zmerlynn))
- Fork `agones-controller` binary and Add `agones-extensions` deployments [\#2866](https://github.com/agones-dev/agones/pull/2866) ([chiayi](https://github.com/chiayi))
- Skip validation errors in mutating webhooks [\#2865](https://github.com/agones-dev/agones/pull/2865) ([zmerlynn](https://github.com/zmerlynn))
- Return better error message when mutation webhook gets invalid JSON [\#2863](https://github.com/agones-dev/agones/pull/2863) ([zmerlynn](https://github.com/zmerlynn))
- Update metrics documentation for Cloud Monitoring/Stackdriver [\#2862](https://github.com/agones-dev/agones/pull/2862) ([junninho](https://github.com/junninho))
- Introduce the Source field in GameServerAllocationStatus to indicate the allocation source [\#2860](https://github.com/agones-dev/agones/pull/2860) ([gongmax](https://github.com/gongmax))
- Release final version updates [\#2858](https://github.com/agones-dev/agones/pull/2858) ([mangalpalli](https://github.com/mangalpalli))
- SafeToEvict: Implement Eviction API, add SetEviction cloud product hook [\#2857](https://github.com/agones-dev/agones/pull/2857) ([zmerlynn](https://github.com/zmerlynn))
- 1.28.0 release [\#2852](https://github.com/agones-dev/agones/pull/2852) ([mangalpalli](https://github.com/mangalpalli))
- Rename `LifecycleContract` feature gate to `SafeToEvict` [\#2849](https://github.com/agones-dev/agones/pull/2849) ([zmerlynn](https://github.com/zmerlynn))
- fix\(make\): current\_project will be executed only when the relevant command is executed [\#2848](https://github.com/agones-dev/agones/pull/2848) ([aimuz](https://github.com/aimuz))
- refactor: Implemented using the standard library [\#2847](https://github.com/agones-dev/agones/pull/2847) ([aimuz](https://github.com/aimuz))
- Fixed: vulnerabilities scanned with govulncheck [\#2841](https://github.com/agones-dev/agones/pull/2841) ([aimuz](https://github.com/aimuz))
- GKE Autopilot: Separate game server workloads [\#2840](https://github.com/agones-dev/agones/pull/2840) ([zmerlynn](https://github.com/zmerlynn))
- SDKGracefulTermination: Promote to beta [\#2836](https://github.com/agones-dev/agones/pull/2836) ([zmerlynn](https://github.com/zmerlynn))

## [v1.28.0](https://github.com/agones-dev/agones/tree/v1.28.0) (2022-12-06)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.28.0-rc...v1.28.0)

**Implemented enhancements:**

- Add a FAQ entry describing when you would use Agones vs. StatefulSets [\#2770](https://github.com/agones-dev/agones/issues/2770)
- Documentation: Kubernetes and Agones supported version matrix [\#2237](https://github.com/agones-dev/agones/issues/2237)

**Fixed bugs:**

- Player tracking malfunction in Unreal SDK due to wrong HTTP method for setting Player Capacity [\#2845](https://github.com/agones-dev/agones/issues/2845)
- Unreal Editor errors due to uninitialized properties [\#2844](https://github.com/agones-dev/agones/issues/2844)
- `agones.allocator.allocationBatchWaitTime` missing in Helm Configuration documentation [\#2837](https://github.com/agones-dev/agones/issues/2837)
- Unreal SDK fix for setting capacity for Player Tracking and Editor error messages [\#2846](https://github.com/agones-dev/agones/pull/2846) ([Titantompa](https://github.com/Titantompa))
- Docs: `agones.allocator.allocationBatchWaitTime` [\#2838](https://github.com/agones-dev/agones/pull/2838) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- Request for gongmax to become Approver [\#2834](https://github.com/agones-dev/agones/issues/2834)
- Request for zmerlynn to become Approver [\#2833](https://github.com/agones-dev/agones/issues/2833)
- Release 1.28.0-rc [\#2832](https://github.com/agones-dev/agones/issues/2832)

**Merged pull requests:**

- Remove trailing whitespace. [\#2839](https://github.com/agones-dev/agones/pull/2839) ([roberthbailey](https://github.com/roberthbailey))
- Validation and documentation for PodDiscriptionBudget change [\#2829](https://github.com/agones-dev/agones/pull/2829) ([chiayi](https://github.com/chiayi))
- FAQ: Why not use Deployment or StatefulSet? [\#2824](https://github.com/agones-dev/agones/pull/2824) ([markmandel](https://github.com/markmandel))
- Adds matrix of Agones versions to Kubernetes versions. [\#2819](https://github.com/agones-dev/agones/pull/2819) ([igooch](https://github.com/igooch))

## [v1.28.0-rc](https://github.com/agones-dev/agones/tree/v1.28.0-rc) (2022-11-30)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.27.0...v1.28.0-rc)

**Implemented enhancements:**

- Immutable replicas field would allow PodDisruptionBudget on selected GameServer Pods [\#2806](https://github.com/agones-dev/agones/issues/2806)
- Update example allocation yaml files to use selectors instead of required [\#2771](https://github.com/agones-dev/agones/issues/2771)
- Only refresh certificates if the fsnotify event is relevant [\#1816](https://github.com/agones-dev/agones/issues/1816)
- Terraform, GKE - add autoscaling Node Pools option [\#1467](https://github.com/agones-dev/agones/issues/1467)
- Terraform, GKE - Option to create a Regional Cluster [\#1441](https://github.com/agones-dev/agones/issues/1441)
- Adding AGONES\_SDK\_GRPC\_HOST to NewSDK [\#1183](https://github.com/agones-dev/agones/issues/1183)
- GameServer: Implement \(immutable\) scale subresource, add pdb [\#2807](https://github.com/agones-dev/agones/pull/2807) ([zmerlynn](https://github.com/zmerlynn))
- Sync Pod host ports back to GameServer in GCP [\#2782](https://github.com/agones-dev/agones/pull/2782) ([zmerlynn](https://github.com/zmerlynn))
- Players in-game metric for when PlayerTracking is enabled [\#2765](https://github.com/agones-dev/agones/pull/2765) ([estebangarcia](https://github.com/estebangarcia))
- Implemented PodDisruptionBudget on relevant deployments [\#2740](https://github.com/agones-dev/agones/pull/2740) ([valentintorikian](https://github.com/valentintorikian))

**Fixed bugs:**

- `test-gen-api-docs` always fail at the first run after the api docs have change [\#2810](https://github.com/agones-dev/agones/issues/2810)
- \[Flake\] Unit Test: TestControllerGameServerCount [\#2804](https://github.com/agones-dev/agones/issues/2804)
- No gameservers available when lots of requests in quick succession [\#2788](https://github.com/agones-dev/agones/issues/2788)
- Shows missing "/usr/local/bin/locust" after building container [\#2744](https://github.com/agones-dev/agones/issues/2744)
- Context has canceled bug Allocate will retry [\#2736](https://github.com/agones-dev/agones/issues/2736)
- Getting started, can't create gameserver  [\#2593](https://github.com/agones-dev/agones/issues/2593)
- Flaky: TestGameServerRestartBeforeReadyCrash [\#2445](https://github.com/agones-dev/agones/issues/2445)
- Upgrade build tools from debian buster to bullseye [\#2224](https://github.com/agones-dev/agones/issues/2224)
- Allocator gRPC doesn't work without TLS [\#1945](https://github.com/agones-dev/agones/issues/1945)
- Agones roles have insufficient permissions defined for clusters where OwnerReferencesPermissionEnforcement is enabled [\#1740](https://github.com/agones-dev/agones/issues/1740)
- TestGameServerRestartBeforeReadyCrash: Close race [\#2812](https://github.com/agones-dev/agones/pull/2812) ([zmerlynn](https://github.com/zmerlynn))
- Flake: TestControllerGameServerCount [\#2805](https://github.com/agones-dev/agones/pull/2805) ([markmandel](https://github.com/markmandel))
- Avoid retry from allocateFromLocalCluster under context kill. [\#2783](https://github.com/agones-dev/agones/pull/2783) ([mangalpalli](https://github.com/mangalpalli))

**Closed issues:**

- Release 1.27.0 [\#2774](https://github.com/agones-dev/agones/issues/2774)
- Conditionally enable fieldalignment linter in govet [\#2325](https://github.com/agones-dev/agones/issues/2325)

**Merged pull requests:**

- Release 1.28.0 rc [\#2835](https://github.com/agones-dev/agones/pull/2835) ([mangalpalli](https://github.com/mangalpalli))
- Docs: Allocation query cache [\#2825](https://github.com/agones-dev/agones/pull/2825) ([markmandel](https://github.com/markmandel))
- Move example images to Artifact Registry [\#2823](https://github.com/agones-dev/agones/pull/2823) ([gongmax](https://github.com/gongmax))
- fix: solve the native collection's memory leak detected by Unity 2021… [\#2822](https://github.com/agones-dev/agones/pull/2822) ([kingshijie](https://github.com/kingshijie))
- Allow controller service account to update finalizers [\#2816](https://github.com/agones-dev/agones/pull/2816) ([bostrt](https://github.com/bostrt))
- Update Node.js dependencies and package [\#2815](https://github.com/agones-dev/agones/pull/2815) ([steven-supersolid](https://github.com/steven-supersolid))
- Terraform, GKE - Option to create regional cluster as well as option to create autoscaling nodepool [\#2813](https://github.com/agones-dev/agones/pull/2813) ([chiayi](https://github.com/chiayi))
- Remove Windows FAQ Entry [\#2811](https://github.com/agones-dev/agones/pull/2811) ([markmandel](https://github.com/markmandel))
- Release: Note to switch away from `agones-images` [\#2809](https://github.com/agones-dev/agones/pull/2809) ([markmandel](https://github.com/markmandel))
- Enable fieldalignment linter, then mostly ignore it [\#2795](https://github.com/agones-dev/agones/pull/2795) ([zmerlynn](https://github.com/zmerlynn))
- Add fswatch library to watch/batch filesystem events, use in allocator [\#2792](https://github.com/agones-dev/agones/pull/2792) ([zmerlynn](https://github.com/zmerlynn))
- GameServerRestartBeforeReadyCrash: Run serially [\#2791](https://github.com/agones-dev/agones/pull/2791) ([zmerlynn](https://github.com/zmerlynn))
- Fix \(not really\) problems reported by VSCode [\#2790](https://github.com/agones-dev/agones/pull/2790) ([zmerlynn](https://github.com/zmerlynn))
- Split port allocators, implement Autopilot port allocation/policies [\#2789](https://github.com/agones-dev/agones/pull/2789) ([zmerlynn](https://github.com/zmerlynn))
- Update game server allocation yaml files to use selectors [\#2787](https://github.com/agones-dev/agones/pull/2787) ([chiayi](https://github.com/chiayi))
- Update health-checking.md [\#2785](https://github.com/agones-dev/agones/pull/2785) ([Amit-karn](https://github.com/Amit-karn))
- Cleanup of load tests [\#2784](https://github.com/agones-dev/agones/pull/2784) ([mangalpalli](https://github.com/mangalpalli))
- Show how to set graceful termination in a game server that is safe to evict [\#2780](https://github.com/agones-dev/agones/pull/2780) ([roberthbailey](https://github.com/roberthbailey))
- Version updates [\#2778](https://github.com/agones-dev/agones/pull/2778) ([mangalpalli](https://github.com/mangalpalli))
- Bring SDK base image to debian:bullseye [\#2769](https://github.com/agones-dev/agones/pull/2769) ([markmandel](https://github.com/markmandel))
- Remove generation for swagger Go code and Add static swagger codes for test [\#2757](https://github.com/agones-dev/agones/pull/2757) ([govargo](https://github.com/govargo))

## [v1.27.0](https://github.com/agones-dev/agones/tree/v1.27.0) (2022-10-26)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.27.0-rc...v1.27.0)

**Closed issues:**

-  Release 1.27.0-rc [\#2766](https://github.com/agones-dev/agones/issues/2766)

**Merged pull requests:**

- Release 1.27.0 [\#2776](https://github.com/agones-dev/agones/pull/2776) ([mangalpalli](https://github.com/mangalpalli))
- Update FAQ on ExternalDNS [\#2773](https://github.com/agones-dev/agones/pull/2773) ([markmandel](https://github.com/markmandel))
- Updates to release checklist. [\#2772](https://github.com/agones-dev/agones/pull/2772) ([markmandel](https://github.com/markmandel))

## [v1.27.0-rc](https://github.com/agones-dev/agones/tree/v1.27.0-rc) (2022-10-20)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.26.0...v1.27.0-rc)

**Implemented enhancements:**

- Allow cluster autoscaler to scale down game server pods [\#2747](https://github.com/agones-dev/agones/issues/2747)
- \[GKE\] - Should we enable image streaming everywhere? [\#2746](https://github.com/agones-dev/agones/issues/2746)
- Support Agones on ARM systems [\#2216](https://github.com/agones-dev/agones/issues/2216)
- Update example containers to fix security vulnerabilities [\#1154](https://github.com/agones-dev/agones/issues/1154)
- Upgrade Go version to 1.19.1 [\#2743](https://github.com/agones-dev/agones/pull/2743) ([gongmax](https://github.com/gongmax))

**Fixed bugs:**

- Flaky test: CPP SDK Conformance tests [\#2298](https://github.com/agones-dev/agones/issues/2298)

**Closed issues:**

- Replace uses of the io/ioutil package [\#2748](https://github.com/agones-dev/agones/issues/2748)
- Release 1.26.0 [\#2737](https://github.com/agones-dev/agones/issues/2737)
- Log cleanup: Verbose error log on pod not yet running [\#2665](https://github.com/agones-dev/agones/issues/2665)
- Upgrade gRPC version [\#1797](https://github.com/agones-dev/agones/issues/1797)

**Merged pull requests:**

- Release 1.27.0-rc [\#2768](https://github.com/agones-dev/agones/pull/2768) ([mangalpalli](https://github.com/mangalpalli))
- Add repository for hashicorp/consul, and disable the consul-consul-client [\#2764](https://github.com/agones-dev/agones/pull/2764) ([gongmax](https://github.com/gongmax))
- 2665 Log cleanup: Verbose error log on pod not yet running [\#2763](https://github.com/agones-dev/agones/pull/2763) ([mangalpalli](https://github.com/mangalpalli))
- Enable image streaming for e2e test cluster [\#2762](https://github.com/agones-dev/agones/pull/2762) ([gongmax](https://github.com/gongmax))
- Fix case of HTTP module reference [\#2760](https://github.com/agones-dev/agones/pull/2760) ([TroutZhang](https://github.com/TroutZhang))
- Add an example of setting autoscaler behavior in a Fleet. [\#2759](https://github.com/agones-dev/agones/pull/2759) ([roberthbailey](https://github.com/roberthbailey))
- Enable image streaming everywhere by default [\#2756](https://github.com/agones-dev/agones/pull/2756) ([gongmax](https://github.com/gongmax))
- Update the Kubernetes upgrade instructions to include instructions for upgrading gRPC [\#2755](https://github.com/agones-dev/agones/pull/2755) ([roberthbailey](https://github.com/roberthbailey))
- If the user has specified cluster autoscaling behavior for their gameserver then don't overwrite it [\#2754](https://github.com/agones-dev/agones/pull/2754) ([roberthbailey](https://github.com/roberthbailey))
- Replace uses of the io/ioutil package [\#2752](https://github.com/agones-dev/agones/pull/2752) ([gongmax](https://github.com/gongmax))
- Bump the example images version [\#2751](https://github.com/agones-dev/agones/pull/2751) ([gongmax](https://github.com/gongmax))
- Added a title to the 1.26.0 release [\#2742](https://github.com/agones-dev/agones/pull/2742) ([markmandel](https://github.com/markmandel))
- updates for upcoming release [\#2741](https://github.com/agones-dev/agones/pull/2741) ([SaitejaTamma](https://github.com/SaitejaTamma))
- Allocator Informer Event  optimize updateFunc [\#2731](https://github.com/agones-dev/agones/pull/2731) ([alvin-7](https://github.com/alvin-7))

## [v1.26.0](https://github.com/agones-dev/agones/tree/v1.26.0) (2022-09-14)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.26.0-rc...v1.26.0)

**Closed issues:**

- Release 1.26.0-rc [\#2732](https://github.com/agones-dev/agones/issues/2732)

**Merged pull requests:**

- release v1.26.0 [\#2738](https://github.com/agones-dev/agones/pull/2738) ([SaitejaTamma](https://github.com/SaitejaTamma))

## [v1.26.0-rc](https://github.com/agones-dev/agones/tree/v1.26.0-rc) (2022-09-07)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.25.0...v1.26.0-rc)

**Breaking changes:**

- Update Kubernetes to 1.23 [\#2642](https://github.com/agones-dev/agones/issues/2642)
- Upgrade Agones to Kubernetes 1.23 [\#2711](https://github.com/agones-dev/agones/pull/2711) ([roberthbailey](https://github.com/roberthbailey))

**Implemented enhancements:**

- Move StateAllocationFilter to Beta [\#2688](https://github.com/agones-dev/agones/issues/2688)
- Made `strategy` configurable on relevant deployments [\#2721](https://github.com/agones-dev/agones/pull/2721) ([valentintorikian](https://github.com/valentintorikian))
- Exposes metrics ports on pods in order to enable GCP Managed Prometheus [\#2712](https://github.com/agones-dev/agones/pull/2712) ([austin-space](https://github.com/austin-space))
- Move StateAllocationFilter to beta [\#2695](https://github.com/agones-dev/agones/pull/2695) ([katsew](https://github.com/katsew))

**Fixed bugs:**

- Default value on website for helm.installTests in helm chart setup value is incorrect [\#2720](https://github.com/agones-dev/agones/issues/2720)
- Helm Agones terraform module crds.CleanupOnDelete setting is missing agones prefix [\#2718](https://github.com/agones-dev/agones/issues/2718)
- Fleet specific prometheus metrics should stop being exported when the fleet is deleted [\#2478](https://github.com/agones-dev/agones/issues/2478)
- Remove \< Ready GameServers first when scaling down on the same node [\#2372](https://github.com/agones-dev/agones/issues/2372)
- FleetAutoscaler keeps alive all TLS connections permanently causing memory leak on webhook server [\#2278](https://github.com/agones-dev/agones/issues/2278)
- Fleet Autoscaler - Fleet name CRD Validation issue [\#1954](https://github.com/agones-dev/agones/issues/1954)
- Added missing env to flag mapping [\#2728](https://github.com/agones-dev/agones/pull/2728) ([valentintorikian](https://github.com/valentintorikian))
- Docs/Helm: Formatted table, fix typo [\#2724](https://github.com/agones-dev/agones/pull/2724) ([markmandel](https://github.com/markmandel))
- Add the agones prefix to the cleanupOnDelete variable name. [\#2723](https://github.com/agones-dev/agones/pull/2723) ([roberthbailey](https://github.com/roberthbailey))
- Bug: Passing arguments to the constructor results in an error [\#2714](https://github.com/agones-dev/agones/pull/2714) ([g2-ochiai-yuta](https://github.com/g2-ochiai-yuta))
- Fleet scale down: Remove \< Ready GameServers first [\#2702](https://github.com/agones-dev/agones/pull/2702) ([markmandel](https://github.com/markmandel))
- Clear metric labels on Fleet/Autoscaler delete [\#2701](https://github.com/agones-dev/agones/pull/2701) ([markmandel](https://github.com/markmandel))
- Add back CustomFasSyncInterval=false to e2e test [\#2700](https://github.com/agones-dev/agones/pull/2700) ([markmandel](https://github.com/markmandel))
- TLS-Memoryleak [\#2681](https://github.com/agones-dev/agones/pull/2681) ([SaitejaTamma](https://github.com/SaitejaTamma))
- fleet-autoscaler-validationfix [\#2674](https://github.com/agones-dev/agones/pull/2674) ([SaitejaTamma](https://github.com/SaitejaTamma))

**Closed issues:**

- Release 1.25.0 [\#2692](https://github.com/agones-dev/agones/issues/2692)

**Merged pull requests:**

- relase v1.26.0-rc [\#2733](https://github.com/agones-dev/agones/pull/2733) ([SaitejaTamma](https://github.com/SaitejaTamma))
- Regenerate gRPC clients for all SDKs. [\#2727](https://github.com/agones-dev/agones/pull/2727) ([markmandel](https://github.com/markmandel))
- Update gRPC SDK tooling to 1.36.1 [\#2725](https://github.com/agones-dev/agones/pull/2725) ([roberthbailey](https://github.com/roberthbailey))
- Fix the default value for `helm.installTests` in the documentation. [\#2722](https://github.com/agones-dev/agones/pull/2722) ([roberthbailey](https://github.com/roberthbailey))
- Pre release check for RC [\#2719](https://github.com/agones-dev/agones/pull/2719) ([SaitejaTamma](https://github.com/SaitejaTamma))
- Update the Kubernetes image used in the helm pre-delete-hook to 1.23. [\#2717](https://github.com/agones-dev/agones/pull/2717) ([roberthbailey](https://github.com/roberthbailey))
- updates for upcoming release [\#2699](https://github.com/agones-dev/agones/pull/2699) ([SaitejaTamma](https://github.com/SaitejaTamma))
- Fixed typo below a "examples" folder [\#2698](https://github.com/agones-dev/agones/pull/2698) ([FirstSS-Sub](https://github.com/FirstSS-Sub))
- release v1.25.0 [\#2694](https://github.com/agones-dev/agones/pull/2694) ([SaitejaTamma](https://github.com/SaitejaTamma))
- Cloud Build: No more need for Docker 20 tag [\#2691](https://github.com/agones-dev/agones/pull/2691) ([markmandel](https://github.com/markmandel))

## [v1.25.0](https://github.com/agones-dev/agones/tree/v1.25.0) (2022-08-03)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.25.0-rc...v1.25.0)

**Implemented enhancements:**

- Adds load balancer ip variable to terraform modules [\#2690](https://github.com/agones-dev/agones/pull/2690) ([austin-space](https://github.com/austin-space))

**Fixed bugs:**

- Presence of \>1 inactive GSS can cause a rolling update to become stuck until an allocation ends [\#2574](https://github.com/agones-dev/agones/issues/2574)
- helm\_agones module for Terraform doesn't support setting variables [\#2484](https://github.com/agones-dev/agones/issues/2484)
- Fix typo in release 1.25.0-rc blog post [\#2686](https://github.com/agones-dev/agones/pull/2686) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- Release 1.25.0-rc [\#2684](https://github.com/agones-dev/agones/issues/2684)
- CI: How to run tests with and without Feature Gates [\#1411](https://github.com/agones-dev/agones/issues/1411)

**Merged pull requests:**

- Fix feature flags on "High Density" documentation [\#2689](https://github.com/agones-dev/agones/pull/2689) ([markmandel](https://github.com/markmandel))

## [v1.25.0-rc](https://github.com/agones-dev/agones/tree/v1.25.0-rc) (2022-07-27)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.24.0...v1.25.0-rc)

**Implemented enhancements:**

- Upgrade Xonotic to 0.8.5 [\#2664](https://github.com/agones-dev/agones/issues/2664)
- End to end tests for SDKGracefulTermination [\#2647](https://github.com/agones-dev/agones/issues/2647)
- Move CustomFasSyncInterval to Beta [\#2646](https://github.com/agones-dev/agones/issues/2646)
- Move NodeExternalDNS to stable [\#2643](https://github.com/agones-dev/agones/issues/2643)
- Upgrade SuperTuxKart to 1.3 [\#2546](https://github.com/agones-dev/agones/issues/2546)
- Docs: How to do local container with sdk [\#2677](https://github.com/agones-dev/agones/pull/2677) ([markmandel](https://github.com/markmandel))
- upgrade xonotic version [\#2669](https://github.com/agones-dev/agones/pull/2669) ([mridulji](https://github.com/mridulji))
- NodeExternalDNS/stable [\#2660](https://github.com/agones-dev/agones/pull/2660) ([SaitejaTamma](https://github.com/SaitejaTamma))
- Move CustomFasSyncInterval to Beta [\#2654](https://github.com/agones-dev/agones/pull/2654) ([govargo](https://github.com/govargo))
- Support for Unity Alpha SDK [\#2600](https://github.com/agones-dev/agones/pull/2600) ([MaxHayman](https://github.com/MaxHayman))

**Fixed bugs:**

- Agones controller down when enabling CustomFasSyncInterval on an existing cluster [\#2675](https://github.com/agones-dev/agones/issues/2675)
- Upgrade GKE Terraform scripts to 4.x Google Provider [\#2630](https://github.com/agones-dev/agones/issues/2630)
- nodejs sdk example container fails to run [\#2625](https://github.com/agones-dev/agones/issues/2625)
- Can't use `make` to build on m1 mac / ARM [\#2517](https://github.com/agones-dev/agones/issues/2517)
- Flaky:  make run-sdk-conformance-test-cpp [\#2346](https://github.com/agones-dev/agones/issues/2346)
- Bug: `make install` uses local helm and jq [\#2672](https://github.com/agones-dev/agones/pull/2672) ([markmandel](https://github.com/markmandel))
- update grpc package version for arm64 Linux [\#2668](https://github.com/agones-dev/agones/pull/2668) ([JJhuk](https://github.com/JJhuk))
- Minikube: Fixes for Makefile, dev & usage docs [\#2667](https://github.com/agones-dev/agones/pull/2667) ([markmandel](https://github.com/markmandel))
- Building and pushing ARM64 images [\#2666](https://github.com/agones-dev/agones/pull/2666) ([mridulji](https://github.com/mridulji))
- Fix minikube dev tooling [\#2662](https://github.com/agones-dev/agones/pull/2662) ([markmandel](https://github.com/markmandel))
- e2e tests and bug fixes: SDKGracefulTermination [\#2661](https://github.com/agones-dev/agones/pull/2661) ([markmandel](https://github.com/markmandel))
- Delete Katacoda link\(404\) because Katacoda is closed [\#2640](https://github.com/agones-dev/agones/pull/2640) ([govargo](https://github.com/govargo))
- Fix nodejs-simple cannot find agones-sdk module and update server\_tag to 0.8 [\#2633](https://github.com/agones-dev/agones/pull/2633) ([govargo](https://github.com/govargo))
- Update hashicorp/google to 4.25.0 [\#2632](https://github.com/agones-dev/agones/pull/2632) ([govargo](https://github.com/govargo))

**Closed issues:**

- Release 1.24.0 [\#2636](https://github.com/agones-dev/agones/issues/2636)

**Merged pull requests:**

- release-v1.25.0-rc [\#2685](https://github.com/agones-dev/agones/pull/2685) ([SaitejaTamma](https://github.com/SaitejaTamma))
- Add Feature Flag section to troubleshooting [\#2679](https://github.com/agones-dev/agones/pull/2679) ([markmandel](https://github.com/markmandel))
- Docs: Note that Feature Gate changes are upgrades [\#2678](https://github.com/agones-dev/agones/pull/2678) ([markmandel](https://github.com/markmandel))
- Update `Contribute` docs feature code version [\#2676](https://github.com/agones-dev/agones/pull/2676) ([markmandel](https://github.com/markmandel))
- Specify the OS architecture tested for known working drivers for minikube. [\#2663](https://github.com/agones-dev/agones/pull/2663) ([roberthbailey](https://github.com/roberthbailey))
- Add agones4j to 3rd party libraries. [\#2656](https://github.com/agones-dev/agones/pull/2656) ([portlek](https://github.com/portlek))
- Added "common development flows" to dev guide [\#2655](https://github.com/agones-dev/agones/pull/2655) ([markmandel](https://github.com/markmandel))
- Upgrade Terraform test dependencies. [\#2653](https://github.com/agones-dev/agones/pull/2653) ([markmandel](https://github.com/markmandel))
- Update the c\# sdk protobuf version to address a few high severity CVEs. [\#2652](https://github.com/agones-dev/agones/pull/2652) ([roberthbailey](https://github.com/roberthbailey))
- Update protobufjs to address a high severity CVE. [\#2651](https://github.com/agones-dev/agones/pull/2651) ([roberthbailey](https://github.com/roberthbailey))
- Health check error message [\#2649](https://github.com/agones-dev/agones/pull/2649) ([XiaNi](https://github.com/XiaNi))
- updates for upcoming rlease [\#2641](https://github.com/agones-dev/agones/pull/2641) ([SaitejaTamma](https://github.com/SaitejaTamma))
- Move CI image builds to Artifact Registry [\#2634](https://github.com/agones-dev/agones/pull/2634) ([markmandel](https://github.com/markmandel))
- Build image: Use apt to install gcloud [\#2629](https://github.com/agones-dev/agones/pull/2629) ([markmandel](https://github.com/markmandel))
- supertuxcart version update-1.3 [\#2608](https://github.com/agones-dev/agones/pull/2608) ([SaitejaTamma](https://github.com/SaitejaTamma))

## [v1.24.0](https://github.com/agones-dev/agones/tree/v1.24.0) (2022-06-22)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.24.0-rc...v1.24.0)

**Closed issues:**

- Release 1.24.0-rc [\#2628](https://github.com/agones-dev/agones/issues/2628)
- gameserverselector documentation mismatch [\#2491](https://github.com/agones-dev/agones/issues/2491)

**Merged pull requests:**

- v1.24.0 [\#2637](https://github.com/agones-dev/agones/pull/2637) ([SaitejaTamma](https://github.com/SaitejaTamma))
- Update Rust SDK build image [\#2635](https://github.com/agones-dev/agones/pull/2635) ([markmandel](https://github.com/markmandel))

## [v1.24.0-rc](https://github.com/agones-dev/agones/tree/v1.24.0-rc) (2022-06-16)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.23.0...v1.24.0-rc)

**Implemented enhancements:**

- Add metric for number of reserved replicas in a fleet [\#2609](https://github.com/agones-dev/agones/issues/2609)
- Make batchWaitTime configurable in the Allocator [\#2586](https://github.com/agones-dev/agones/issues/2586)
- Document how to use Informers and Listers to query Agones [\#1260](https://github.com/agones-dev/agones/issues/1260)
- Add docs for reserved replicas metric [\#2611](https://github.com/agones-dev/agones/pull/2611) ([markmandel](https://github.com/markmandel))
- Add metric for number of reserved replicas [\#2610](https://github.com/agones-dev/agones/pull/2610) ([govargo](https://github.com/govargo))
- e2e tests for arm64 simple game server [\#2604](https://github.com/agones-dev/agones/pull/2604) ([Ludea](https://github.com/Ludea))
- Ping arm img [\#2591](https://github.com/agones-dev/agones/pull/2591) ([Ludea](https://github.com/Ludea))
- Make Allocator batchWaitTime configurable [\#2589](https://github.com/agones-dev/agones/pull/2589) ([valentintorikian](https://github.com/valentintorikian))
- Added Agones Category to all UPROPERTY macro [\#2587](https://github.com/agones-dev/agones/pull/2587) ([Dinhh1](https://github.com/Dinhh1))
- Add Document about Informers and Listers [\#2579](https://github.com/agones-dev/agones/pull/2579) ([govargo](https://github.com/govargo))

**Fixed bugs:**

- Unable to scale fleet if a game server is allocated from an older version [\#2617](https://github.com/agones-dev/agones/issues/2617)
- Agones Allocator's verifyClientCertificate method does not properly handle intermediate certificates [\#2602](https://github.com/agones-dev/agones/issues/2602)
- ARM64 agones-controller image seems to start agones-allocator? [\#2578](https://github.com/agones-dev/agones/issues/2578)
- Agones Controller attempts to scale to a negative integer instead of zero [\#2509](https://github.com/agones-dev/agones/issues/2509)
- If Fleet is allocated during a rolling update, the old GameServerSet may not be deleted [\#2432](https://github.com/agones-dev/agones/issues/2432)
- sdkServer.logLevel in Gameserver config doesn't seem to be used in local mode [\#2221](https://github.com/agones-dev/agones/issues/2221)
- Fix replicas miscalculation when fleet replicas equals zero [\#2623](https://github.com/agones-dev/agones/pull/2623) ([govargo](https://github.com/govargo))
- Fix cross-platform build for simple-game-server [\#2613](https://github.com/agones-dev/agones/pull/2613) ([markmandel](https://github.com/markmandel))
- Fix double build of SDK binary in Makefile [\#2612](https://github.com/agones-dev/agones/pull/2612) ([markmandel](https://github.com/markmandel))
- debian image update/Examples [\#2607](https://github.com/agones-dev/agones/pull/2607) ([SaitejaTamma](https://github.com/SaitejaTamma))
- Fix intermediate certificate handling \(\#2602\) [\#2605](https://github.com/agones-dev/agones/pull/2605) ([josiahp](https://github.com/josiahp))
- Fix build command: build-agones-sdk-image [\#2603](https://github.com/agones-dev/agones/pull/2603) ([govargo](https://github.com/govargo))
- Fix typo in allocator.yaml [\#2590](https://github.com/agones-dev/agones/pull/2590) ([jhowcrof](https://github.com/jhowcrof))
- Fix local-includes WITH\_ARM64=0 or WITH\_WINDOWS=0 [\#2588](https://github.com/agones-dev/agones/pull/2588) ([markmandel](https://github.com/markmandel))
- updateing debian image to bullseye/Part-1 [\#2584](https://github.com/agones-dev/agones/pull/2584) ([SaitejaTamma](https://github.com/SaitejaTamma))
- Fix sdkServer.logLevel nil in Gameserver in local-mode [\#2580](https://github.com/agones-dev/agones/pull/2580) ([govargo](https://github.com/govargo))

**Closed issues:**

- Possible confusion on allocator CLI flag config [\#2598](https://github.com/agones-dev/agones/issues/2598)
- Release 1.23.0 [\#2571](https://github.com/agones-dev/agones/issues/2571)
- The Player Capacity integration pattern example is still using the deprecated fields in the GameServerAllocation [\#2570](https://github.com/agones-dev/agones/issues/2570)

**Merged pull requests:**

- v1.24.0-rc [\#2631](https://github.com/agones-dev/agones/pull/2631) ([SaitejaTamma](https://github.com/SaitejaTamma))
- Open Match + Agones Examples [\#2627](https://github.com/agones-dev/agones/pull/2627) ([markmandel](https://github.com/markmandel))
- Fix deprecated command for make minikube-push [\#2624](https://github.com/agones-dev/agones/pull/2624) ([govargo](https://github.com/govargo))
- Install docs: Supported Container Architectures [\#2621](https://github.com/agones-dev/agones/pull/2621) ([markmandel](https://github.com/markmandel))
- Upgrade everything to simple-game-server:0.13 [\#2620](https://github.com/agones-dev/agones/pull/2620) ([markmandel](https://github.com/markmandel))
- Unblock CI: Fix site link 404 [\#2619](https://github.com/agones-dev/agones/pull/2619) ([markmandel](https://github.com/markmandel))
- Upgrade built-tools Prometheus and Grafana [\#2618](https://github.com/agones-dev/agones/pull/2618) ([markmandel](https://github.com/markmandel))
- Add an extra step to the kubernetes upgrade tasks. [\#2616](https://github.com/agones-dev/agones/pull/2616) ([roberthbailey](https://github.com/roberthbailey))
- Update the image used in the helm uninstall to work on ARM. [\#2615](https://github.com/agones-dev/agones/pull/2615) ([roberthbailey](https://github.com/roberthbailey))
- Minor fix to the build target in the instructions. [\#2614](https://github.com/agones-dev/agones/pull/2614) ([roberthbailey](https://github.com/roberthbailey))
- Issue template for Kubernetes Upgrades [\#2606](https://github.com/agones-dev/agones/pull/2606) ([markmandel](https://github.com/markmandel))
- Moved allocator flag parsing configuration away from `metrics.go` [\#2599](https://github.com/agones-dev/agones/pull/2599) ([valentintorikian](https://github.com/valentintorikian))
- Move "future" helm parameters that are part of the current release up to the visible table [\#2596](https://github.com/agones-dev/agones/pull/2596) ([roberthbailey](https://github.com/roberthbailey))
- Update development guide to current practices [\#2595](https://github.com/agones-dev/agones/pull/2595) ([markmandel](https://github.com/markmandel))
- Fix controller arm64 tag  [\#2581](https://github.com/agones-dev/agones/pull/2581) ([Ludea](https://github.com/Ludea))
- Change deprecated preferred & required to selectors in Player Capacity docs [\#2577](https://github.com/agones-dev/agones/pull/2577) ([JJhuk](https://github.com/JJhuk))
- Preparation for the 1.24.0 release [\#2576](https://github.com/agones-dev/agones/pull/2576) ([SaitejaTamma](https://github.com/SaitejaTamma))

## [v1.23.0](https://github.com/agones-dev/agones/tree/v1.23.0) (2022-05-11)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.23.0-rc...v1.23.0)

**Fixed bugs:**

- Panic in sidecar in 1.22.0 [\#2568](https://github.com/agones-dev/agones/issues/2568)
- ensure context is set before spawning goroutines using it [\#2569](https://github.com/agones-dev/agones/pull/2569) ([Hades32](https://github.com/Hades32))

**Closed issues:**

- Release 1.23.0-rc [\#2566](https://github.com/agones-dev/agones/issues/2566)

**Merged pull requests:**

- Remove broken links to unblock CI [\#2573](https://github.com/agones-dev/agones/pull/2573) ([roberthbailey](https://github.com/roberthbailey))
- release v1.23.0 [\#2572](https://github.com/agones-dev/agones/pull/2572) ([SaitejaTamma](https://github.com/SaitejaTamma))

## [v1.23.0-rc](https://github.com/agones-dev/agones/tree/v1.23.0-rc) (2022-05-04)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.22.0...v1.23.0-rc)

**Breaking changes:**

- Upgrade terraform to Kubernetes 1.22 [\#2551](https://github.com/agones-dev/agones/pull/2551) ([roberthbailey](https://github.com/roberthbailey))

**Implemented enhancements:**

- Upgrade to Kubernetes 1.22 [\#2494](https://github.com/agones-dev/agones/issues/2494)
- Update golang version used in App Engine  [\#2382](https://github.com/agones-dev/agones/issues/2382)
- Allocator controller arm64 img [\#2565](https://github.com/agones-dev/agones/pull/2565) ([Ludea](https://github.com/Ludea))
- sdk arm64 images [\#2521](https://github.com/agones-dev/agones/pull/2521) ([Ludea](https://github.com/Ludea))

**Fixed bugs:**

- Healthcontroller falsly marks healthy pods as unhealthy [\#2553](https://github.com/agones-dev/agones/issues/2553)
- how to allocate a Local Game Server? [\#2536](https://github.com/agones-dev/agones/issues/2536)
- We should automatically reject PR's that are not against `main` [\#2531](https://github.com/agones-dev/agones/issues/2531)
- Foreground deletion of a fleet managed by fleet autoscaler results in infinite pod recreation loop [\#2524](https://github.com/agones-dev/agones/issues/2524)
- Flaky:  TestFleetRecreateGameServers/gameserver\_shutdown [\#2479](https://github.com/agones-dev/agones/issues/2479)
- GameServer stucks in Shutdown state preventing a rolling update to complete [\#2360](https://github.com/agones-dev/agones/issues/2360)
- \[Go SDK\] PlayerConnect\(\) at capacity panics [\#1957](https://github.com/agones-dev/agones/issues/1957)
- Fix for health controller race condition marking healthy pods as unhealthy [\#2554](https://github.com/agones-dev/agones/pull/2554) ([Thiryn](https://github.com/Thiryn))
- Queue updated GameServers with DeletionTimestamp [\#2550](https://github.com/agones-dev/agones/pull/2550) ([markmandel](https://github.com/markmandel))
- Don't move Dev GameServer back to Ready always [\#2545](https://github.com/agones-dev/agones/pull/2545) ([markmandel](https://github.com/markmandel))
- Initial step for M1 with \(make build-images WITH\_WINDOWS=0 and make s… [\#2542](https://github.com/agones-dev/agones/pull/2542) ([SaitejaTamma](https://github.com/SaitejaTamma))
- Update unreal.md to fix \#2523 [\#2537](https://github.com/agones-dev/agones/pull/2537) ([goace](https://github.com/goace))
- Use source sdk in simple-game-server [\#2527](https://github.com/agones-dev/agones/pull/2527) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- Collaborator Request [\#2552](https://github.com/agones-dev/agones/issues/2552)
- Release 1.22.0 [\#2530](https://github.com/agones-dev/agones/issues/2530)
- find a bug in the doc for ue4 plugin [\#2523](https://github.com/agones-dev/agones/issues/2523)

**Merged pull requests:**

- Release v1.23.0-rc [\#2567](https://github.com/agones-dev/agones/pull/2567) ([SaitejaTamma](https://github.com/SaitejaTamma))
- Update dependency for the `push-agones-sdk-linux-image-amd64` target [\#2564](https://github.com/agones-dev/agones/pull/2564) ([roberthbailey](https://github.com/roberthbailey))
- Update the go runtime environment version for the website from 1.13 to 1.16 [\#2563](https://github.com/agones-dev/agones/pull/2563) ([roberthbailey](https://github.com/roberthbailey))
- Regenerate Kubernetes resource includes \(ObjectMeta, PodTemplateSpec\) for Kubernetes 1.22 [\#2562](https://github.com/agones-dev/agones/pull/2562) ([roberthbailey](https://github.com/roberthbailey))
- Add the new gke-gcloud-auth-plugin binary to the build image [\#2561](https://github.com/agones-dev/agones/pull/2561) ([roberthbailey](https://github.com/roberthbailey))
- Update CRD API reference for Kubernetes 1.22 [\#2560](https://github.com/agones-dev/agones/pull/2560) ([roberthbailey](https://github.com/roberthbailey))
- Upgrade client-go to v0.22.9 [\#2559](https://github.com/agones-dev/agones/pull/2559) ([roberthbailey](https://github.com/roberthbailey))
- Update Minikube and Kind dev tooling to Kubernetes 1.22 [\#2558](https://github.com/agones-dev/agones/pull/2558) ([roberthbailey](https://github.com/roberthbailey))
- Increase the e2e cluster size to match what was observed [\#2557](https://github.com/agones-dev/agones/pull/2557) ([roberthbailey](https://github.com/roberthbailey))
- Update the dev version of Kubernetes in the website to 1.22 [\#2556](https://github.com/agones-dev/agones/pull/2556) ([roberthbailey](https://github.com/roberthbailey))
- Upgrade kubectl to 1.22 in dev tooling [\#2555](https://github.com/agones-dev/agones/pull/2555) ([roberthbailey](https://github.com/roberthbailey))
- Unblock CI: Remove EKS Autoscaler terraform link [\#2548](https://github.com/agones-dev/agones/pull/2548) ([markmandel](https://github.com/markmandel))
- Move some Info logging to Debug [\#2547](https://github.com/agones-dev/agones/pull/2547) ([markmandel](https://github.com/markmandel))
- Update fleetautoscaler e2e certs [\#2541](https://github.com/agones-dev/agones/pull/2541) ([markmandel](https://github.com/markmandel))
- Update everything to simple-gameserver:0.12 [\#2540](https://github.com/agones-dev/agones/pull/2540) ([markmandel](https://github.com/markmandel))
- v2: Flaky TestFleetRecreateGameServers fix. [\#2539](https://github.com/agones-dev/agones/pull/2539) ([markmandel](https://github.com/markmandel))
- Fix for flaky TestFleetRecreateGameServers [\#2535](https://github.com/agones-dev/agones/pull/2535) ([markmandel](https://github.com/markmandel))
- UE4 docs broke links with UE5 release [\#2534](https://github.com/agones-dev/agones/pull/2534) ([markmandel](https://github.com/markmandel))
-  updates for the upcoming release [\#2533](https://github.com/agones-dev/agones/pull/2533) ([SaitejaTamma](https://github.com/SaitejaTamma))
- Check for DeletionTimestamp of fleet and gameserverset before scaling [\#2526](https://github.com/agones-dev/agones/pull/2526) ([estebangarcia](https://github.com/estebangarcia))
- Add e2e test for PlayerConnectWithCapacityZero [\#2503](https://github.com/agones-dev/agones/pull/2503) ([jiwonaid](https://github.com/jiwonaid))

## [v1.22.0](https://github.com/agones-dev/agones/tree/v1.22.0) (2022-04-05)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.22.0-rc...v1.22.0)

**Fixed bugs:**

- Example crd-client/go.mod can not be compiled [\#2061](https://github.com/agones-dev/agones/issues/2061)

**Closed issues:**

- Release 1.22.0-rc [\#2518](https://github.com/agones-dev/agones/issues/2518)

**Merged pull requests:**

- Release v1.22.0 [\#2532](https://github.com/agones-dev/agones/pull/2532) ([SaitejaTamma](https://github.com/SaitejaTamma))

## [v1.22.0-rc](https://github.com/agones-dev/agones/tree/v1.22.0-rc) (2022-03-24)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.21.0...v1.22.0-rc)

**Implemented enhancements:**

- Add a multi-cluster allocation example solution to leverage GCP resources to connect to multiple Agones clusters [\#2495](https://github.com/agones-dev/agones/issues/2495)
- Agones controller metrics becomes a huge amount of data over time [\#2424](https://github.com/agones-dev/agones/issues/2424)
- Allow specifying agones-allocator nodePort via Helm values [\#1699](https://github.com/agones-dev/agones/issues/1699)
- Optionally include a ServiceMonitor in the Helm chart [\#1605](https://github.com/agones-dev/agones/issues/1605)
- Integrate with cert-manager to manage secrets on the cluster [\#1461](https://github.com/agones-dev/agones/issues/1461)
- Load Testing Framework for internal and external usage [\#412](https://github.com/agones-dev/agones/issues/412)
- Add Allocation Endpoint GCP solution for multi-cluster allocation to Agones examples [\#2499](https://github.com/agones-dev/agones/pull/2499) ([pooneh-m](https://github.com/pooneh-m))
- Add a tool that can run variable allocation load scenarios [\#2493](https://github.com/agones-dev/agones/pull/2493) ([roberthbailey](https://github.com/roberthbailey))
- updates for next release [\#2482](https://github.com/agones-dev/agones/pull/2482) ([SaitejaTamma](https://github.com/SaitejaTamma))

**Fixed bugs:**

- The `allocation-endpoint` sample terraform is using `goolge-private` provider [\#2512](https://github.com/agones-dev/agones/issues/2512)
- Failed to build CPP SDK [\#2486](https://github.com/agones-dev/agones/issues/2486)
- Allocator Service Document Bug [\#2467](https://github.com/agones-dev/agones/issues/2467)
- GameServerAllocation is not working for "High Density GameServers" [\#2408](https://github.com/agones-dev/agones/issues/2408)
- Fleet RollingUpdate gets stuck when Fleet has high number of allocated GameServers [\#2397](https://github.com/agones-dev/agones/issues/2397)
- Fix panic when playertracking is false [\#2489](https://github.com/agones-dev/agones/pull/2489) ([jiwonaid](https://github.com/jiwonaid))
- Fix Rolling Update with Allocated GameServers [\#2420](https://github.com/agones-dev/agones/pull/2420) ([WVerlaek](https://github.com/WVerlaek))

**Closed issues:**

- Release 1.21.0 [\#2480](https://github.com/agones-dev/agones/issues/2480)

**Merged pull requests:**

- Add multi-cluster allocation demo to third party doc [\#2520](https://github.com/agones-dev/agones/pull/2520) ([tmokmss](https://github.com/tmokmss))
- Release v1.22.0 rc [\#2519](https://github.com/agones-dev/agones/pull/2519) ([SaitejaTamma](https://github.com/SaitejaTamma))
- e2e flow for High Density GameServers [\#2516](https://github.com/agones-dev/agones/pull/2516) ([markmandel](https://github.com/markmandel))
- Sidecar arm64 bin [\#2514](https://github.com/agones-dev/agones/pull/2514) ([Ludea](https://github.com/Ludea))
- Remove google-private [\#2513](https://github.com/agones-dev/agones/pull/2513) ([pooneh-m](https://github.com/pooneh-m))
- C\# Health and WatchGameServer fixes [\#2510](https://github.com/agones-dev/agones/pull/2510) ([mleenhardt](https://github.com/mleenhardt))
- Update the version of the simple game server in the load test samples to the current version [\#2508](https://github.com/agones-dev/agones/pull/2508) ([roberthbailey](https://github.com/roberthbailey))
- Update simple-game-server to 0.11 everywhere [\#2506](https://github.com/agones-dev/agones/pull/2506) ([markmandel](https://github.com/markmandel))
- Added a Development Tools section as requested [\#2502](https://github.com/agones-dev/agones/pull/2502) ([comerford](https://github.com/comerford))
- Update CPP gRPC to v1.27.1 [\#2500](https://github.com/agones-dev/agones/pull/2500) ([markmandel](https://github.com/markmandel))
- Fix allocator service document [\#2477](https://github.com/agones-dev/agones/pull/2477) ([jiwonaid](https://github.com/jiwonaid))
- Add NodePort to the helm template for agones-allocator service [\#2476](https://github.com/agones-dev/agones/pull/2476) ([jiwonaid](https://github.com/jiwonaid))
- Tweaking wording for Agones installation [\#2475](https://github.com/agones-dev/agones/pull/2475) ([karenarialin](https://github.com/karenarialin))
- Reorganizing sections for GKE installation [\#2474](https://github.com/agones-dev/agones/pull/2474) ([karenarialin](https://github.com/karenarialin))
- Update minor wording [\#2473](https://github.com/agones-dev/agones/pull/2473) ([karenarialin](https://github.com/karenarialin))
- docs: Support gRPC without TLS [\#2472](https://github.com/agones-dev/agones/pull/2472) ([SpringMT](https://github.com/SpringMT))
- Support cert-manager for controller tls [\#2453](https://github.com/agones-dev/agones/pull/2453) ([xxtanisxx](https://github.com/xxtanisxx))

## [v1.21.0](https://github.com/agones-dev/agones/tree/v1.21.0) (2022-02-15)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.21.0-rc...v1.21.0)

**Fixed bugs:**

- Flaky test: fleetautoscaler\_test.go [\#2296](https://github.com/agones-dev/agones/issues/2296)

**Closed issues:**

- Release 1.21.0-rc [\#2469](https://github.com/agones-dev/agones/issues/2469)

**Merged pull requests:**

- Release v1.21.0 [\#2481](https://github.com/agones-dev/agones/pull/2481) ([SaitejaTamma](https://github.com/SaitejaTamma))

## [v1.21.0-rc](https://github.com/agones-dev/agones/tree/v1.21.0-rc) (2022-02-10)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.20.0...v1.21.0-rc)

**Breaking changes:**

- Remove node\_name label from allocation metrics [\#2433](https://github.com/agones-dev/agones/pull/2433) ([yoshd](https://github.com/yoshd))

**Implemented enhancements:**

- Update to node 16 / npm 7 [\#2450](https://github.com/agones-dev/agones/issues/2450)
- Fix "kubectl explain" output for Agones CRDs [\#1194](https://github.com/agones-dev/agones/issues/1194)
- Adding AcceleratXR to companies using agones [\#2412](https://github.com/agones-dev/agones/pull/2412) ([acceleratxr](https://github.com/acceleratxr))

**Fixed bugs:**

- Initial GameServer state is not sent on watch with local SDK server [\#2437](https://github.com/agones-dev/agones/issues/2437)
- Flakiness: Autoscaler tests [\#2385](https://github.com/agones-dev/agones/issues/2385)
- panic in simple-game-server on second UNHEALTHY message [\#2366](https://github.com/agones-dev/agones/issues/2366)
- CI: Uninstall/rollback release if Helm stuck in pending upgrade [\#2356](https://github.com/agones-dev/agones/issues/2356)
- FleetAutoscaler has confusing ScalingLimited warning when scaling down [\#2297](https://github.com/agones-dev/agones/issues/2297)
- GameServerAllocation metadata isn't validated [\#2282](https://github.com/agones-dev/agones/issues/2282)
- Update the simple game server to avoid a race condition when transitioning from ready to allocated [\#2451](https://github.com/agones-dev/agones/pull/2451) ([roberthbailey](https://github.com/roberthbailey))
- Validate GameServerAllocation metadata [\#2449](https://github.com/agones-dev/agones/pull/2449) ([markmandel](https://github.com/markmandel))
- update on scaler limited to Max [\#2446](https://github.com/agones-dev/agones/pull/2446) ([SaitejaTamma](https://github.com/SaitejaTamma))
- Fix connection timeout on Rust SDK. [\#2444](https://github.com/agones-dev/agones/pull/2444) ([markmandel](https://github.com/markmandel))
- Send initial GameServer update in WatchGameServer [\#2442](https://github.com/agones-dev/agones/pull/2442) ([scrayos](https://github.com/scrayos))
- Simple Game Server: Don't panic on UNHEALTHY x 2 [\#2427](https://github.com/agones-dev/agones/pull/2427) ([markmandel](https://github.com/markmandel))
- Fleet Autoscaler custom sync: Race condition fix [\#2422](https://github.com/agones-dev/agones/pull/2422) ([markmandel](https://github.com/markmandel))

**Security fixes:**

- Update the node-forge dependency to address GHSA-5rrq-pxf6-6jx5 [\#2435](https://github.com/agones-dev/agones/pull/2435) ([roberthbailey](https://github.com/roberthbailey))

**Closed issues:**

- Release 1.20.0 [\#2430](https://github.com/agones-dev/agones/issues/2430)

**Merged pull requests:**

- v1.21.0-rc [\#2470](https://github.com/agones-dev/agones/pull/2470) ([SaitejaTamma](https://github.com/SaitejaTamma))
- Update to nodejs 16 and npm lockfile version 2 [\#2465](https://github.com/agones-dev/agones/pull/2465) ([steven-supersolid](https://github.com/steven-supersolid))
- Wait for the shutdown signal before having the simple game server exit [\#2463](https://github.com/agones-dev/agones/pull/2463) ([roberthbailey](https://github.com/roberthbailey))
- Remove trailing space in logging statement. [\#2448](https://github.com/agones-dev/agones/pull/2448) ([markmandel](https://github.com/markmandel))
- Regenerate package-lock.json for the two node.js projects [\#2447](https://github.com/agones-dev/agones/pull/2447) ([steven-supersolid](https://github.com/steven-supersolid))
- Move Allocator unit tests to better test module [\#2441](https://github.com/agones-dev/agones/pull/2441) ([markmandel](https://github.com/markmandel))
- Get the initial game server state before marking the simple game server ready [\#2440](https://github.com/agones-dev/agones/pull/2440) ([roberthbailey](https://github.com/roberthbailey))
- Unit test for an Allocation empty selector. [\#2439](https://github.com/agones-dev/agones/pull/2439) ([markmandel](https://github.com/markmandel))
- Remove curl|bash from dockerfile to address vulnerability issues [\#2438](https://github.com/agones-dev/agones/pull/2438) ([cindy52](https://github.com/cindy52))
- Next release updates [\#2434](https://github.com/agones-dev/agones/pull/2434) ([SaitejaTamma](https://github.com/SaitejaTamma))
- docs: typo grep command [\#2429](https://github.com/agones-dev/agones/pull/2429) ([JJhuk](https://github.com/JJhuk))
- CI: Uninstall Helm release if stuck [\#2426](https://github.com/agones-dev/agones/pull/2426) ([markmandel](https://github.com/markmandel))
- Fix "kubectl explain" output for Agones CRDs [\#2423](https://github.com/agones-dev/agones/pull/2423) ([jiwonaid](https://github.com/jiwonaid))
- Add a new prerequisite to the release checklist [\#2419](https://github.com/agones-dev/agones/pull/2419) ([roberthbailey](https://github.com/roberthbailey))

## [v1.20.0](https://github.com/agones-dev/agones/tree/v1.20.0) (2022-01-18)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.20.0-rc...v1.20.0)

**Security fixes:**

- Upgrade terraform test dependencies. [\#2425](https://github.com/agones-dev/agones/pull/2425) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- Release 1.20.0-rc [\#2418](https://github.com/agones-dev/agones/issues/2418)

**Merged pull requests:**

- release 1.20.0 [\#2431](https://github.com/agones-dev/agones/pull/2431) ([SaitejaTamma](https://github.com/SaitejaTamma))

## [v1.20.0-rc](https://github.com/agones-dev/agones/tree/v1.20.0-rc) (2022-01-07)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.19.0...v1.20.0-rc)

**Implemented enhancements:**

- Update the simple game server to move itself back to the Ready state after allocation [\#2409](https://github.com/agones-dev/agones/pull/2409) ([roberthbailey](https://github.com/roberthbailey))
- Add a flag to simple-game-server to shutdown after a specified number of seconds [\#2407](https://github.com/agones-dev/agones/pull/2407) ([roberthbailey](https://github.com/roberthbailey))

**Fixed bugs:**

- Unreal SDK misses "const" prefix for function parameters. [\#2411](https://github.com/agones-dev/agones/issues/2411)
- System.BadImageFormatException: Bad IL format. The format of the file '...\runtimes\win\native\grpc\_csharp\_ext.x64.dll' is invalid. [\#2403](https://github.com/agones-dev/agones/issues/2403)
- TLS handshake error with multi-cluster setup [\#2402](https://github.com/agones-dev/agones/issues/2402)
- Fix webhook image, and e2e.AssertFleetCondition [\#2386](https://github.com/agones-dev/agones/pull/2386) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- Proposal move 1.20.0 release by a week [\#2388](https://github.com/agones-dev/agones/issues/2388)
- Release 1.19.0 [\#2381](https://github.com/agones-dev/agones/issues/2381)

**Merged pull requests:**

- Release v1.20.0-rc  [\#2421](https://github.com/agones-dev/agones/pull/2421) ([SaitejaTamma](https://github.com/SaitejaTamma))
- Fixed missing "const" parameter for SetAnnotation & SetLabels. [\#2415](https://github.com/agones-dev/agones/pull/2415) ([KiaArmani](https://github.com/KiaArmani))
- Upgrade-gRPC-version-2 [\#2414](https://github.com/agones-dev/agones/pull/2414) ([SaitejaTamma](https://github.com/SaitejaTamma))
- Update the simple game server version to 0.6. [\#2413](https://github.com/agones-dev/agones/pull/2413) ([roberthbailey](https://github.com/roberthbailey))
- Fix broken links in the website. [\#2405](https://github.com/agones-dev/agones/pull/2405) ([roberthbailey](https://github.com/roberthbailey))
- Update the minikube driver flag name \(--vm-driver is deprecated\) [\#2399](https://github.com/agones-dev/agones/pull/2399) ([roberthbailey](https://github.com/roberthbailey))
- format cURL statement in an human readable form [\#2395](https://github.com/agones-dev/agones/pull/2395) ([freegroup](https://github.com/freegroup))
- pretty print pre-formatted code to "bash" [\#2393](https://github.com/agones-dev/agones/pull/2393) ([freegroup](https://github.com/freegroup))
- Update allocator-service.md [\#2392](https://github.com/agones-dev/agones/pull/2392) ([freegroup](https://github.com/freegroup))
- Preparation for the 1.20.0 release [\#2384](https://github.com/agones-dev/agones/pull/2384) ([SaitejaTamma](https://github.com/SaitejaTamma))

## [v1.19.0](https://github.com/agones-dev/agones/tree/v1.19.0) (2021-11-24)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.19.0-rc...v1.19.0)

**Closed issues:**

-  Release 1.19.0-rc [\#2376](https://github.com/agones-dev/agones/issues/2376)

**Merged pull requests:**

- release v-1.19.0 [\#2383](https://github.com/agones-dev/agones/pull/2383) ([SaitejaTamma](https://github.com/SaitejaTamma))
- Add additional Octops projects to Third Party docs [\#2380](https://github.com/agones-dev/agones/pull/2380) ([danieloliveira079](https://github.com/danieloliveira079))

## [v1.19.0-rc](https://github.com/agones-dev/agones/tree/v1.19.0-rc) (2021-11-17)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.18.0...v1.19.0-rc)

**Breaking changes:**

- Upgrade to Kubernetes 1.21 [\#2311](https://github.com/agones-dev/agones/issues/2311)
- Move NodeExternalDNS to Beta [\#2240](https://github.com/agones-dev/agones/issues/2240)
- Upgrade client-go to v0.21.5. [\#2333](https://github.com/agones-dev/agones/pull/2333) ([roberthbailey](https://github.com/roberthbailey))
- Upgrade terraform to Kubernetes 1.21. [\#2326](https://github.com/agones-dev/agones/pull/2326) ([roberthbailey](https://github.com/roberthbailey))

**Implemented enhancements:**

- allow passing certificates as values instead of files in the Helm chart [\#2364](https://github.com/agones-dev/agones/issues/2364)
- Move SDK sidecar to first position in container list [\#2355](https://github.com/agones-dev/agones/issues/2355)
- Unity package for Unity SDK [\#2338](https://github.com/agones-dev/agones/issues/2338)
- Prometheus metrics: Use ServiceMonitor instead of deprecated annotation mechanism [\#2262](https://github.com/agones-dev/agones/issues/2262)
- Sidecar REST endpoint should return 400 if healthcheck body is empty [\#2256](https://github.com/agones-dev/agones/issues/2256)
- Move SDKWatchSendOnExecute to Stable [\#2238](https://github.com/agones-dev/agones/issues/2238)
- Upgrade Terraform to 1.0 [\#2142](https://github.com/agones-dev/agones/issues/2142)
- NodeExternalDNS moved to beta [\#2369](https://github.com/agones-dev/agones/pull/2369) ([SaitejaTamma](https://github.com/SaitejaTamma))
- expose Helm chart values for custom certs [\#2367](https://github.com/agones-dev/agones/pull/2367) ([rahil-p](https://github.com/rahil-p))
- Move the agones sidecar containers to the beginning of the list of containers [\#2357](https://github.com/agones-dev/agones/pull/2357) ([roberthbailey](https://github.com/roberthbailey))
- SDKWatchSendOnExecute to Stable [\#2353](https://github.com/agones-dev/agones/pull/2353) ([SaitejaTamma](https://github.com/SaitejaTamma))
- Update alpine version to 3.14 [\#2345](https://github.com/agones-dev/agones/pull/2345) ([SaitejaTamma](https://github.com/SaitejaTamma))
- Support Unity Package Manager [\#2343](https://github.com/agones-dev/agones/pull/2343) ([aaronchapin-tocaboca](https://github.com/aaronchapin-tocaboca))
- Add a flag to the simple game server so that it can have a delay before marking itself ready [\#2340](https://github.com/agones-dev/agones/pull/2340) ([roberthbailey](https://github.com/roberthbailey))
- Add ability to specify annotations for the SDK service account [\#2317](https://github.com/agones-dev/agones/pull/2317) ([highlyunavailable](https://github.com/highlyunavailable))
- Add error callback to WatchGameServer in Node.js SDK  [\#2315](https://github.com/agones-dev/agones/pull/2315) ([steven-supersolid](https://github.com/steven-supersolid))
- Upgraded Terraform to \>=1.0 [\#2308](https://github.com/agones-dev/agones/pull/2308) ([zaratsian](https://github.com/zaratsian))
- Prometheus metrics: Use ServiceMonitor instead of deprecated annotation mechanism [\#2290](https://github.com/agones-dev/agones/pull/2290) ([zifter](https://github.com/zifter))

**Fixed bugs:**

- Flakey: TestUnhealthyGameServersWithoutFreePorts [\#2339](https://github.com/agones-dev/agones/issues/2339)
- agones.dev/last-allocated GameServer annotation not in parseable format [\#2331](https://github.com/agones-dev/agones/issues/2331)
- Uncatchable error in NodeJS Agones SDK when calling shutdown\(\) [\#2304](https://github.com/agones-dev/agones/issues/2304)
- Can't run gen-api-docs with Go 1.16 [\#2168](https://github.com/agones-dev/agones/issues/2168)
- Configure kubernetes provider for eks module [\#2352](https://github.com/agones-dev/agones/pull/2352) ([mvlabat](https://github.com/mvlabat))
- Fix for health check race condition [\#2351](https://github.com/agones-dev/agones/pull/2351) ([markmandel](https://github.com/markmandel))
- Fix bug in e2e/LogEvents [\#2350](https://github.com/agones-dev/agones/pull/2350) ([markmandel](https://github.com/markmandel))
- Better e2e udp send errors [\#2349](https://github.com/agones-dev/agones/pull/2349) ([markmandel](https://github.com/markmandel))
- Hope to reduce e2e flakiness [\#2348](https://github.com/agones-dev/agones/pull/2348) ([markmandel](https://github.com/markmandel))
- Upgrade `terraform-aws-eks` to `v17.22.0` [\#2344](https://github.com/agones-dev/agones/pull/2344) ([mvlabat](https://github.com/mvlabat))
- pre\_delete\_hook.yaml should support release namespace [\#2342](https://github.com/agones-dev/agones/pull/2342) ([rayterrill](https://github.com/rayterrill))
- Update format of last-allocated to RFC 3339, set in Agones SDK as well [\#2336](https://github.com/agones-dev/agones/pull/2336) ([WVerlaek](https://github.com/WVerlaek))
- Update Rust to fix CI [\#2313](https://github.com/agones-dev/agones/pull/2313) ([markmandel](https://github.com/markmandel))

**Security fixes:**

- Update ansi-regex to fix a moderate security vulnerability. [\#2321](https://github.com/agones-dev/agones/pull/2321) ([roberthbailey](https://github.com/roberthbailey))

**Closed issues:**

- Request Releaser role for Agones Repository [\#2368](https://github.com/agones-dev/agones/issues/2368)
- Migrate to use prow GitHub app instead of bot account [\#2347](https://github.com/agones-dev/agones/issues/2347)
- Release 1.18.0 [\#2306](https://github.com/agones-dev/agones/issues/2306)
- Build tools: Deprecated linters [\#2301](https://github.com/agones-dev/agones/issues/2301)

**Merged pull requests:**

- Updates for release 1.19.0-rc [\#2377](https://github.com/agones-dev/agones/pull/2377) ([SaitejaTamma](https://github.com/SaitejaTamma))
- Update the version of the simple-game-server to 0.5. [\#2374](https://github.com/agones-dev/agones/pull/2374) ([roberthbailey](https://github.com/roberthbailey))
- Remove mention of the SDKWatchSendOnExecute feature gate in a comment in sdkserver\_test.go [\#2373](https://github.com/agones-dev/agones/pull/2373) ([roberthbailey](https://github.com/roberthbailey))
- docs: fixes from friction log [\#2370](https://github.com/agones-dev/agones/pull/2370) ([irataxy](https://github.com/irataxy))
- Upgrading grpc client and server on same version [\#2362](https://github.com/agones-dev/agones/pull/2362) ([SaitejaTamma](https://github.com/SaitejaTamma))
- Rewrite TestUnhealthyGameServersWithoutFreePorts so that it is less flaky [\#2341](https://github.com/agones-dev/agones/pull/2341) ([markmandel](https://github.com/markmandel))
- Updated aks.md [\#2337](https://github.com/agones-dev/agones/pull/2337) ([AmieDD](https://github.com/AmieDD))
- Regenerate Kubernetes resource includes \(ObjectMeta, PodTemplateSpec\) for Kubernetes 1.21. [\#2335](https://github.com/agones-dev/agones/pull/2335) ([roberthbailey](https://github.com/roberthbailey))
- Update CRD API reference for Kubernetes 1.21. [\#2334](https://github.com/agones-dev/agones/pull/2334) ([roberthbailey](https://github.com/roberthbailey))
- alphabetizing the linters list [\#2330](https://github.com/agones-dev/agones/pull/2330) ([SealTV](https://github.com/SealTV))
- Upgrade kubectl to 1.21 in dev tooling. [\#2329](https://github.com/agones-dev/agones/pull/2329) ([roberthbailey](https://github.com/roberthbailey))
- Update Minikube and Kind dev tooling to Kubernetes 1.21. [\#2328](https://github.com/agones-dev/agones/pull/2328) ([roberthbailey](https://github.com/roberthbailey))
- Update the dev version of Kubernetes in the website to 1.21. [\#2327](https://github.com/agones-dev/agones/pull/2327) ([roberthbailey](https://github.com/roberthbailey))
- Update .golangci.yml [\#2323](https://github.com/agones-dev/agones/pull/2323) ([SealTV](https://github.com/SealTV))
- GKE installation wording tweaks [\#2322](https://github.com/agones-dev/agones/pull/2322) ([karenarialin](https://github.com/karenarialin))
- Adding Vizor Games to list 'Companies using Agones' [\#2320](https://github.com/agones-dev/agones/pull/2320) ([SealTV](https://github.com/SealTV))
- Upgrade to Go 1.17 [\#2319](https://github.com/agones-dev/agones/pull/2319) ([cindy52](https://github.com/cindy52))
- Add an example to the release template for a step that I always have to double check. [\#2318](https://github.com/agones-dev/agones/pull/2318) ([roberthbailey](https://github.com/roberthbailey))
- Add winterpixel's rocket bot royale to list of agones links [\#2316](https://github.com/agones-dev/agones/pull/2316) ([jordo](https://github.com/jordo))
- Minikube: more options for connectivity [\#2312](https://github.com/agones-dev/agones/pull/2312) ([markmandel](https://github.com/markmandel))
- Preparation for the 1.19.0 release [\#2310](https://github.com/agones-dev/agones/pull/2310) ([roberthbailey](https://github.com/roberthbailey))
- Added check for empty healthcheck post-body [\#2288](https://github.com/agones-dev/agones/pull/2288) ([sankalp-r](https://github.com/sankalp-r))

## [v1.18.0](https://github.com/agones-dev/agones/tree/v1.18.0) (2021-10-12)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.18.0-rc...v1.18.0)

**Closed issues:**

- Spelling Error [\#2293](https://github.com/agones-dev/agones/issues/2293)
- Release 1.18.0-rc [\#2291](https://github.com/agones-dev/agones/issues/2291)
- Update https://github.com/golangci/golangci-lint [\#2220](https://github.com/agones-dev/agones/issues/2220)

**Merged pull requests:**

- Release 1.18.0 [\#2309](https://github.com/agones-dev/agones/pull/2309) ([roberthbailey](https://github.com/roberthbailey))
- Add some extra emphasis on the breaking change in the helm parameters related to the allocator service [\#2305](https://github.com/agones-dev/agones/pull/2305) ([roberthbailey](https://github.com/roberthbailey))
- e2e: Add test name to Fleet check [\#2303](https://github.com/agones-dev/agones/pull/2303) ([markmandel](https://github.com/markmandel))
- Flaky TestGameServerUnhealthyAfterReadyCrash [\#2302](https://github.com/agones-dev/agones/pull/2302) ([markmandel](https://github.com/markmandel))
- fixed typo in readme of load testing [\#2300](https://github.com/agones-dev/agones/pull/2300) ([dzmitry-lahoda](https://github.com/dzmitry-lahoda))
- Update golang-ci lint version and fix lint errors [\#2295](https://github.com/agones-dev/agones/pull/2295) ([rajat-mangla](https://github.com/rajat-mangla))
- Corrected spelling error: Issue \#2293 [\#2294](https://github.com/agones-dev/agones/pull/2294) ([vaibhavp1964](https://github.com/vaibhavp1964))

## [v1.18.0-rc](https://github.com/agones-dev/agones/tree/v1.18.0-rc) (2021-10-05)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.17.0...v1.18.0-rc)

**Breaking changes:**

- Allow the ports for gRPC and REST to be configured for the allocator service [\#2272](https://github.com/agones-dev/agones/pull/2272) ([roberthbailey](https://github.com/roberthbailey))

**Implemented enhancements:**

- Skip volume mounts in the allocator pod if TLS on mTLS is disabled [\#2276](https://github.com/agones-dev/agones/issues/2276)
- Allow the allocator service to use the go http/2 server [\#2263](https://github.com/agones-dev/agones/issues/2263)
- Move RollingUpdateOnReady to Stable [\#2239](https://github.com/agones-dev/agones/issues/2239)
- Explain how namespace parameter works for allocations [\#2090](https://github.com/agones-dev/agones/issues/2090)
- Proposal: provide allocator-client.default secret in gameserver namespace [\#1686](https://github.com/agones-dev/agones/issues/1686)
- Game Server Allocation advanced filtering: player count, state, reallocation [\#1239](https://github.com/agones-dev/agones/issues/1239)
- Skip the secrets and volume mounts in the allocator pod when they aren't needed [\#2277](https://github.com/agones-dev/agones/pull/2277) ([roberthbailey](https://github.com/roberthbailey))
- Move RollingUpdateOnReady to stable [\#2271](https://github.com/agones-dev/agones/pull/2271) ([Jeffwan](https://github.com/Jeffwan))
- Docs: High Density Integration Pattern [\#2270](https://github.com/agones-dev/agones/pull/2270) ([markmandel](https://github.com/markmandel))
- Docs: Integration Pattern - Reusing GameServers [\#2251](https://github.com/agones-dev/agones/pull/2251) ([markmandel](https://github.com/markmandel))
- Support graceful termination [\#2205](https://github.com/agones-dev/agones/pull/2205) ([bennetty](https://github.com/bennetty))

**Fixed bugs:**

- Can't apply fleetautoscaler on 1.17.0 [\#2253](https://github.com/agones-dev/agones/issues/2253)
- Unblock CI: Ignore rolltable link for testing [\#2269](https://github.com/agones-dev/agones/pull/2269) ([markmandel](https://github.com/markmandel))
- Separate the Helm value for the allocator service name from its service account name [\#2268](https://github.com/agones-dev/agones/pull/2268) ([rcreasey](https://github.com/rcreasey))

**Closed issues:**

- Release 1.17.0 [\#2244](https://github.com/agones-dev/agones/issues/2244)

**Merged pull requests:**

- Release 1.18.0-rc  [\#2292](https://github.com/agones-dev/agones/pull/2292) ([roberthbailey](https://github.com/roberthbailey))
- Unblock CI: Ignore afterverse link for testing [\#2289](https://github.com/agones-dev/agones/pull/2289) ([roberthbailey](https://github.com/roberthbailey))
- Update the link to look for issues for the release. [\#2287](https://github.com/agones-dev/agones/pull/2287) ([roberthbailey](https://github.com/roberthbailey))
- docs: remove --node-ami auto for managed nodegroups [\#2285](https://github.com/agones-dev/agones/pull/2285) ([SpringMT](https://github.com/SpringMT))
- Ignore SDK CPP build directory [\#2284](https://github.com/agones-dev/agones/pull/2284) ([markmandel](https://github.com/markmandel))
- Test for extended allocation metadata characters [\#2283](https://github.com/agones-dev/agones/pull/2283) ([markmandel](https://github.com/markmandel))
- Minor cleanup on reserved diagram [\#2281](https://github.com/agones-dev/agones/pull/2281) ([markmandel](https://github.com/markmandel))
- Fix a link on the FAQ that was pointed at localhost [\#2275](https://github.com/agones-dev/agones/pull/2275) ([roberthbailey](https://github.com/roberthbailey))
- Update the helm template command to match what is used in `make gen-install`. [\#2265](https://github.com/agones-dev/agones/pull/2265) ([roberthbailey](https://github.com/roberthbailey))
- Fix typo [\#2260](https://github.com/agones-dev/agones/pull/2260) ([paulhkim80](https://github.com/paulhkim80))
- Remove GOPATH from development guide [\#2259](https://github.com/agones-dev/agones/pull/2259) ([markmandel](https://github.com/markmandel))
- Create examples.md [\#2258](https://github.com/agones-dev/agones/pull/2258) ([paulhkim80](https://github.com/paulhkim80))
- Explain how namespaces work with allocations [\#2257](https://github.com/agones-dev/agones/pull/2257) ([eminguyen](https://github.com/eminguyen))
- \[docs\] fix create gameserver args [\#2254](https://github.com/agones-dev/agones/pull/2254) ([karamaru-alpha](https://github.com/karamaru-alpha))
- Preparation for 1.18.0 [\#2250](https://github.com/agones-dev/agones/pull/2250) ([cindy52](https://github.com/cindy52))

## [v1.17.0](https://github.com/agones-dev/agones/tree/v1.17.0) (2021-09-01)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.17.0-rc...v1.17.0)

**Fixed bugs:**

- Add nil check to fleet autoscaler validation for sync field [\#2246](https://github.com/agones-dev/agones/pull/2246) ([lambertwang](https://github.com/lambertwang))
- Fix validation bug in FleetAutoscaler [\#2242](https://github.com/agones-dev/agones/pull/2242) ([yoshd](https://github.com/yoshd))

**Closed issues:**

- Release 1.17.0-rc [\#2231](https://github.com/agones-dev/agones/issues/2231)

**Merged pull requests:**

- Release/1.17.0 [\#2247](https://github.com/agones-dev/agones/pull/2247) ([cindy52](https://github.com/cindy52))
- release 1.17.0 [\#2245](https://github.com/agones-dev/agones/pull/2245) ([cindy52](https://github.com/cindy52))

## [v1.17.0-rc](https://github.com/agones-dev/agones/tree/v1.17.0-rc) (2021-08-26)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.16.0...v1.17.0-rc)

**Breaking changes:**

- Upgrade to Kubernetes 1.20 [\#2186](https://github.com/agones-dev/agones/issues/2186)
- Upgrade client-go to v0.20.9. [\#2194](https://github.com/agones-dev/agones/pull/2194) ([roberthbailey](https://github.com/roberthbailey))

**Implemented enhancements:**

- Configurable custom resync period for FleetAutoscaler [\#1955](https://github.com/agones-dev/agones/issues/1955)
- Docs: Player Capacity Integration Pattern [\#2229](https://github.com/agones-dev/agones/pull/2229) ([markmandel](https://github.com/markmandel))
- Docs: Canary Testing Integration Pattern [\#2227](https://github.com/agones-dev/agones/pull/2227) ([markmandel](https://github.com/markmandel))
- Create "Integration Patterns" section in docs [\#2215](https://github.com/agones-dev/agones/pull/2215) ([markmandel](https://github.com/markmandel))
- Update the GameServerAllocation Specification to remove required/pref… [\#2206](https://github.com/agones-dev/agones/pull/2206) ([cindy52](https://github.com/cindy52))
- Update proto and allocator for advanced allocation [\#2199](https://github.com/agones-dev/agones/pull/2199) ([markmandel](https://github.com/markmandel))
- GSA: Advanced Filtering via resource API [\#2188](https://github.com/agones-dev/agones/pull/2188) ([markmandel](https://github.com/markmandel))
- Upgrade terraform to Kubernetes 1.20. [\#2187](https://github.com/agones-dev/agones/pull/2187) ([roberthbailey](https://github.com/roberthbailey))
- Custom fleet autoscaler resync interval [\#2171](https://github.com/agones-dev/agones/pull/2171) ([jie-bao](https://github.com/jie-bao))
- GSA: Switch LabelSelector to GameServerSelector [\#2166](https://github.com/agones-dev/agones/pull/2166) ([markmandel](https://github.com/markmandel))

**Fixed bugs:**

- Errors in Unreal Engine SDK BuildAgonesRequest [\#2169](https://github.com/agones-dev/agones/issues/2169)
- The documentation for game server allocations is inconsistent [\#2136](https://github.com/agones-dev/agones/issues/2136)
- Unlock mutex before returning on error in SDKServer.updateState [\#2234](https://github.com/agones-dev/agones/pull/2234) ([skystar-p](https://github.com/skystar-p))
- Workaround for bullseye release CI blockage [\#2225](https://github.com/agones-dev/agones/pull/2225) ([markmandel](https://github.com/markmandel))
- Used array of FStringFormatArg to process FString::Format to fix erro… [\#2170](https://github.com/agones-dev/agones/pull/2170) ([WilSimpson](https://github.com/WilSimpson))

**Security fixes:**

- Update dev dependencies and audit fix security warning [\#2233](https://github.com/agones-dev/agones/pull/2233) ([steven-supersolid](https://github.com/steven-supersolid))
- Update github.com/gorilla/websocket to address CVE-2020-27813. [\#2195](https://github.com/agones-dev/agones/pull/2195) ([roberthbailey](https://github.com/roberthbailey))

**Closed issues:**

- Request Releaser role for Agones Repository [\#2232](https://github.com/agones-dev/agones/issues/2232)
- Collaborator Request [\#2210](https://github.com/agones-dev/agones/issues/2210)
- Release 1.16.0 [\#2183](https://github.com/agones-dev/agones/issues/2183)
- Proposal: Update the GameServerAllocation Specification to remove required/preferred [\#2146](https://github.com/agones-dev/agones/issues/2146)

**Merged pull requests:**

- Release 1.17.0-rc [\#2236](https://github.com/agones-dev/agones/pull/2236) ([cindy52](https://github.com/cindy52))
- remove allocated 2 times from dot file [\#2230](https://github.com/agones-dev/agones/pull/2230) ([dzmitry-lahoda](https://github.com/dzmitry-lahoda))
- Fix for tabbing in gameserverallocation.md [\#2228](https://github.com/agones-dev/agones/pull/2228) ([markmandel](https://github.com/markmandel))
- Add a missing word to clarify our policy for when to upgrade Kubernetes versions [\#2212](https://github.com/agones-dev/agones/pull/2212) ([roberthbailey](https://github.com/roberthbailey))
- \[docs\] examples allocator fixed link \(without / it did no resolved\) [\#2211](https://github.com/agones-dev/agones/pull/2211) ([dzmitry-lahoda](https://github.com/dzmitry-lahoda))
- Rollback \#2200. [\#2204](https://github.com/agones-dev/agones/pull/2204) ([roberthbailey](https://github.com/roberthbailey))
- Regenerate Kubernetes resource includes \(ObjectMeta, PodTemplateSpec\) for Kubernetes 1.20 [\#2203](https://github.com/agones-dev/agones/pull/2203) ([roberthbailey](https://github.com/roberthbailey))
- Set sidebar\_menu\_compact to true to make the side menu easier to navigate [\#2200](https://github.com/agones-dev/agones/pull/2200) ([roberthbailey](https://github.com/roberthbailey))
- Rename metapatch var in applyAllocationToGameServer [\#2198](https://github.com/agones-dev/agones/pull/2198) ([markmandel](https://github.com/markmandel))
- Update CRD API reference for Kubernetes 1.20. [\#2197](https://github.com/agones-dev/agones/pull/2197) ([roberthbailey](https://github.com/roberthbailey))
- Update Minikube and Kind dev tooling to Kubernetes 1.20 [\#2193](https://github.com/agones-dev/agones/pull/2193) ([roberthbailey](https://github.com/roberthbailey))
- Update the dev version of Kubernetes in the website to 1.20. [\#2192](https://github.com/agones-dev/agones/pull/2192) ([roberthbailey](https://github.com/roberthbailey))
- Update prost/prost-types [\#2190](https://github.com/agones-dev/agones/pull/2190) ([Jake-Shadle](https://github.com/Jake-Shadle))
- Upgrade kubectl to 1.20 in dev tooling. [\#2189](https://github.com/agones-dev/agones/pull/2189) ([roberthbailey](https://github.com/roberthbailey))
- Prep for the 1.17.0 release. [\#2185](https://github.com/agones-dev/agones/pull/2185) ([roberthbailey](https://github.com/roberthbailey))

## [v1.16.0](https://github.com/agones-dev/agones/tree/v1.16.0) (2021-07-20)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.16.0-rc...v1.16.0)

**Closed issues:**

- Release 1.16.0-rc [\#2179](https://github.com/agones-dev/agones/issues/2179)

**Merged pull requests:**

- Release 1.16.0 [\#2184](https://github.com/agones-dev/agones/pull/2184) ([roberthbailey](https://github.com/roberthbailey))
- documentation - add godot-sdk to third-party libraries-tools page [\#2182](https://github.com/agones-dev/agones/pull/2182) ([AndreMicheletti](https://github.com/AndreMicheletti))

## [v1.16.0-rc](https://github.com/agones-dev/agones/tree/v1.16.0-rc) (2021-07-14)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.15.0...v1.16.0-rc)

**Breaking changes:**

- Upgrade to Kubernetes 1.19 [\#2129](https://github.com/agones-dev/agones/issues/2129)
- Migrate to using SANs for webhook certificates for Go 1.15 [\#1899](https://github.com/agones-dev/agones/issues/1899)
- Review Rust gRPC ecosystem for Rust SDK [\#1300](https://github.com/agones-dev/agones/issues/1300)
- Upgrade/go 1.15 [\#2167](https://github.com/agones-dev/agones/pull/2167) ([cindy52](https://github.com/cindy52))
- Upgrade client-go to v0.19.12 [\#2155](https://github.com/agones-dev/agones/pull/2155) ([cindy52](https://github.com/cindy52))
- Update helm configuration to allow annotations to be added to service accounts [\#2134](https://github.com/agones-dev/agones/pull/2134) ([roberthbailey](https://github.com/roberthbailey))
- Replace grpcio with tonic [\#2112](https://github.com/agones-dev/agones/pull/2112) ([Jake-Shadle](https://github.com/Jake-Shadle))

**Implemented enhancements:**

- Provide an easier way to bring your own certificates via helm chart installation [\#2175](https://github.com/agones-dev/agones/issues/2175)
- Remove pre-1.0 documentation from the agones.dev website [\#2156](https://github.com/agones-dev/agones/issues/2156)
- It is not possible to configure Agones HELM with Stackdriver in GCloud when the cluster has Workload Identity. [\#2101](https://github.com/agones-dev/agones/issues/2101)
- Add "copy to clipboard" buttons to example commands on the website [\#2096](https://github.com/agones-dev/agones/issues/2096)
- Add memory and cpu recommendations to minikube starting documentation [\#1536](https://github.com/agones-dev/agones/issues/1536)
- Allow disabling of all allocator secrets in helm chart [\#2177](https://github.com/agones-dev/agones/pull/2177) ([sudermanjr](https://github.com/sudermanjr))
- add copy to clipboard function to code on website [\#2149](https://github.com/agones-dev/agones/pull/2149) ([cindy52](https://github.com/cindy52))
- Refactor ReadyGameServerCache to AllocationCache [\#2148](https://github.com/agones-dev/agones/pull/2148) ([markmandel](https://github.com/markmandel))
- Feature gates for advanced Allocation filtering [\#2143](https://github.com/agones-dev/agones/pull/2143) ([markmandel](https://github.com/markmandel))

**Fixed bugs:**

- export-openapi.sh doesn't work with Kubernetes 1.19 [\#2159](https://github.com/agones-dev/agones/issues/2159)
- SSL Cert expired on agones.dev [\#2133](https://github.com/agones-dev/agones/issues/2133)
- fleet-tcp.yaml miss the spec of `ports.protocol` [\#2113](https://github.com/agones-dev/agones/issues/2113)
- Can't run Allocation example with Go 1.16 [\#2024](https://github.com/agones-dev/agones/issues/2024)
- Cannot connect to a game server using Docker Desktop \(with integrated K8s cluster or Minikube\) [\#1990](https://github.com/agones-dev/agones/issues/1990)
- Fix Rust Sample Docker image [\#2180](https://github.com/agones-dev/agones/pull/2180) ([markmandel](https://github.com/markmandel))
- Allow FILE env variable for local SDK server [\#2174](https://github.com/agones-dev/agones/pull/2174) ([markmandel](https://github.com/markmandel))
- Fix for failing export-openapi.sh on K8s 1.19 [\#2160](https://github.com/agones-dev/agones/pull/2160) ([markmandel](https://github.com/markmandel))
- Fix shutdown problems in ping application. [\#2141](https://github.com/agones-dev/agones/pull/2141) ([s-shin](https://github.com/s-shin))

**Closed issues:**

- Release 1.15.0 [\#2126](https://github.com/agones-dev/agones/issues/2126)
- Limiting resources documentation typo [\#2100](https://github.com/agones-dev/agones/issues/2100)
- Upgrade Hugo + Docsy to latest versions [\#1819](https://github.com/agones-dev/agones/issues/1819)

**Merged pull requests:**

- Release 1.16.0-rc [\#2181](https://github.com/agones-dev/agones/pull/2181) ([roberthbailey](https://github.com/roberthbailey))
- Update AKS terraform install template [\#2165](https://github.com/agones-dev/agones/pull/2165) ([WeetA34](https://github.com/WeetA34))
- Fix sidecar tag in different make targets [\#2163](https://github.com/agones-dev/agones/pull/2163) ([aLekSer](https://github.com/aLekSer))
- terraform-init on gcloud-terraform-destroy-cluster [\#2161](https://github.com/agones-dev/agones/pull/2161) ([markmandel](https://github.com/markmandel))
- Update CRD API reference and regenerate CRD Kubernetes client libraries for Kubernetes 1.19. [\#2158](https://github.com/agones-dev/agones/pull/2158) ([roberthbailey](https://github.com/roberthbailey))
- Remove links to the pre-1.0 versions of the website from the navbar. [\#2157](https://github.com/agones-dev/agones/pull/2157) ([roberthbailey](https://github.com/roberthbailey))
- Update Minikube and Kind dev tooling to Kubernetes 1.19 [\#2152](https://github.com/agones-dev/agones/pull/2152) ([roberthbailey](https://github.com/roberthbailey))
- Update the dev version of Kubernetes in the website to 1.19. [\#2151](https://github.com/agones-dev/agones/pull/2151) ([roberthbailey](https://github.com/roberthbailey))
- Update the consul helm chart to use HashiCorp's Official Helm Chart. [\#2150](https://github.com/agones-dev/agones/pull/2150) ([roberthbailey](https://github.com/roberthbailey))
- Upgrade kubectl to 1.19 in dev tooling.  [\#2147](https://github.com/agones-dev/agones/pull/2147) ([roberthbailey](https://github.com/roberthbailey))
- Removal of getRandomlySelectedGS [\#2144](https://github.com/agones-dev/agones/pull/2144) ([markmandel](https://github.com/markmandel))
- Upgrade terraform to Kubernetes 1.19. [\#2138](https://github.com/agones-dev/agones/pull/2138) ([roberthbailey](https://github.com/roberthbailey))
- Updated Minikube documentation [\#2137](https://github.com/agones-dev/agones/pull/2137) ([markmandel](https://github.com/markmandel))
- Fix a typo in the documentation. [\#2135](https://github.com/agones-dev/agones/pull/2135) ([roberthbailey](https://github.com/roberthbailey))
- Prep for the 1.16.0 release [\#2130](https://github.com/agones-dev/agones/pull/2130) ([sawagh](https://github.com/sawagh))
- Fix Network Security Group Gameserver rule not applied on AKS cluster [\#2124](https://github.com/agones-dev/agones/pull/2124) ([WeetA34](https://github.com/WeetA34))
- Update the doscy theme [\#2123](https://github.com/agones-dev/agones/pull/2123) ([roberthbailey](https://github.com/roberthbailey))
- Adds the Missing TCP protocol spec to the example fleet [\#2122](https://github.com/agones-dev/agones/pull/2122) ([Rohansjamadagni](https://github.com/Rohansjamadagni))
- Change the way we generate new version numbers for nodejs [\#2120](https://github.com/agones-dev/agones/pull/2120) ([roberthbailey](https://github.com/roberthbailey))

## [v1.15.0](https://github.com/agones-dev/agones/tree/v1.15.0) (2021-06-08)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.15.0-rc...v1.15.0)

**Closed issues:**

- Release 1.15.0-rc [\#2119](https://github.com/agones-dev/agones/issues/2119)

**Merged pull requests:**

- Release 1.15.0 [\#2128](https://github.com/agones-dev/agones/pull/2128) ([sawagh](https://github.com/sawagh))

## [v1.15.0-rc](https://github.com/agones-dev/agones/tree/v1.15.0-rc) (2021-06-02)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.14.0...v1.15.0-rc)

**Implemented enhancements:**

- Azure AKS support for public IP per Node/VM [\#2083](https://github.com/agones-dev/agones/issues/2083)
- Unreal plugin WatchGameServer implementation [\#2064](https://github.com/agones-dev/agones/pull/2064) ([highlyunavailable](https://github.com/highlyunavailable))

**Fixed bugs:**

- Creating a GameServerAllocation returns a 200 Ok instead of a 201 Created [\#2108](https://github.com/agones-dev/agones/issues/2108)
- Nil Reference/massive log spam in Controller \[1.13\] [\#2086](https://github.com/agones-dev/agones/issues/2086)
- Cannot update Fleet and set replicas to 0 in same transaction [\#2084](https://github.com/agones-dev/agones/issues/2084)
- Flaky: Hugo occasionally fails: fatal error: concurrent map read and map write [\#1981](https://github.com/agones-dev/agones/issues/1981)
- Return HTTP 201 on GameServerAllocation [\#2110](https://github.com/agones-dev/agones/pull/2110) ([markmandel](https://github.com/markmandel))
- Update and audit fix Node.js dependencies [\#2099](https://github.com/agones-dev/agones/pull/2099) ([steven-supersolid](https://github.com/steven-supersolid))
- Clone Kubernetes objects in API Server before encoding them [\#2089](https://github.com/agones-dev/agones/pull/2089) ([highlyunavailable](https://github.com/highlyunavailable))

**Closed issues:**

- Request Releaser role for Agones Repository [\#2115](https://github.com/agones-dev/agones/issues/2115)
- Release 1.14.0 [\#2077](https://github.com/agones-dev/agones/issues/2077)
- Allocation endpoint: Deprecate `metaPatch` for `metadata` [\#2042](https://github.com/agones-dev/agones/issues/2042)

**Merged pull requests:**

- Release 1.15.0-rc [\#2121](https://github.com/agones-dev/agones/pull/2121) ([sawagh](https://github.com/sawagh))
- Update link to contributing guide from the membership template. [\#2118](https://github.com/agones-dev/agones/pull/2118) ([roberthbailey](https://github.com/roberthbailey))
- Update our community membership guidelines to add a Releaser role. [\#2111](https://github.com/agones-dev/agones/pull/2111) ([roberthbailey](https://github.com/roberthbailey))
- Respectful code cleanup No.2 [\#2109](https://github.com/agones-dev/agones/pull/2109) ([markmandel](https://github.com/markmandel))
- Respectful code cleanup No.1 [\#2107](https://github.com/agones-dev/agones/pull/2107) ([markmandel](https://github.com/markmandel))
- aks setup improvements [\#2103](https://github.com/agones-dev/agones/pull/2103) ([dzmitry-lahoda](https://github.com/dzmitry-lahoda))
- e2e test: Update Fleet replicas 0 with Spec change [\#2095](https://github.com/agones-dev/agones/pull/2095) ([markmandel](https://github.com/markmandel))
- Link Client SDK page to Third Party SDKs [\#2094](https://github.com/agones-dev/agones/pull/2094) ([markmandel](https://github.com/markmandel))
- Add Afterverse logo [\#2092](https://github.com/agones-dev/agones/pull/2092) ([jose-cieni-afterverse](https://github.com/jose-cieni-afterverse))
- Upgrade to Hugo 0.82.1 [\#2085](https://github.com/agones-dev/agones/pull/2085) ([markmandel](https://github.com/markmandel))
- Add rust SDK functionality table [\#2082](https://github.com/agones-dev/agones/pull/2082) ([domgreen](https://github.com/domgreen))
- Adding Functionality table for go SDK [\#2081](https://github.com/agones-dev/agones/pull/2081) ([domgreen](https://github.com/domgreen))
- Minor updates to the release checklist. [\#2080](https://github.com/agones-dev/agones/pull/2080) ([roberthbailey](https://github.com/roberthbailey))
- Prep for the 1.15.0 release. [\#2079](https://github.com/agones-dev/agones/pull/2079) ([roberthbailey](https://github.com/roberthbailey))
- Documenting unity SDK functionality [\#2076](https://github.com/agones-dev/agones/pull/2076) ([domgreen](https://github.com/domgreen))
- Rename MetaPatch to Metadata for AllocationRequest [\#2070](https://github.com/agones-dev/agones/pull/2070) ([lambertwang](https://github.com/lambertwang))

## [v1.14.0](https://github.com/agones-dev/agones/tree/v1.14.0) (2021-04-28)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.14.0-rc...v1.14.0)

**Implemented enhancements:**

- Migrate away from Pull Panda [\#1689](https://github.com/agones-dev/agones/issues/1689)
- Document the Security and Disclosure process for Agones [\#745](https://github.com/agones-dev/agones/issues/745)
- Easier to find out about Community Meetings [\#2069](https://github.com/agones-dev/agones/pull/2069) ([markmandel](https://github.com/markmandel))

**Fixed bugs:**

- build.sh is missing in go directory for SDK [\#1039](https://github.com/agones-dev/agones/issues/1039)

**Closed issues:**

- Release 1.14.0-rc [\#2066](https://github.com/agones-dev/agones/issues/2066)
- GKE: Update documentation + Automation to disable node automatic updates for gameserver node pools [\#1137](https://github.com/agones-dev/agones/issues/1137)

**Merged pull requests:**

- Release 1.14.0 [\#2078](https://github.com/agones-dev/agones/pull/2078) ([roberthbailey](https://github.com/roberthbailey))
- Add table for all implemented SDK for Unreal [\#2074](https://github.com/agones-dev/agones/pull/2074) ([domgreen](https://github.com/domgreen))
- Add Netspeak Games logo [\#2073](https://github.com/agones-dev/agones/pull/2073) ([domgreen](https://github.com/domgreen))
- Suppress the long shell command to test for a file existence so that [\#2072](https://github.com/agones-dev/agones/pull/2072) ([roberthbailey](https://github.com/roberthbailey))
- Updates to the release checklist, based on cutting my first release candidate. [\#2068](https://github.com/agones-dev/agones/pull/2068) ([roberthbailey](https://github.com/roberthbailey))

## [v1.14.0-rc](https://github.com/agones-dev/agones/tree/v1.14.0-rc) (2021-04-21)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.13.0...v1.14.0-rc)

**Breaking changes:**

- Move RollingUpdateOnReady to Beta [\#1970](https://github.com/agones-dev/agones/issues/1970)
- Update the machine type for GKE clusters in build scripts and terraform modules. [\#2063](https://github.com/agones-dev/agones/pull/2063) ([roberthbailey](https://github.com/roberthbailey))
- Move RollingUpdateOnReady to Beta [\#2033](https://github.com/agones-dev/agones/pull/2033) ([roberthbailey](https://github.com/roberthbailey))

**Implemented enhancements:**

- Update recommended machine type for GKE [\#2055](https://github.com/agones-dev/agones/issues/2055)
- Update simple-game-server to 0.3 [\#2045](https://github.com/agones-dev/agones/issues/2045)
- Update the simple game server image to 0.3. [\#2048](https://github.com/agones-dev/agones/pull/2048) ([roberthbailey](https://github.com/roberthbailey))
- Add Terraform config for Windows clusters [\#2046](https://github.com/agones-dev/agones/pull/2046) ([jeremyje](https://github.com/jeremyje))
- Build Agones Windows images by default. [\#2037](https://github.com/agones-dev/agones/pull/2037) ([jeremyje](https://github.com/jeremyje))

**Fixed bugs:**

- Use the correct feature flag name \(and guard it properly\). [\#2035](https://github.com/agones-dev/agones/pull/2035) ([roberthbailey](https://github.com/roberthbailey))
- Upgrade Rust language test version [\#2034](https://github.com/agones-dev/agones/pull/2034) ([markmandel](https://github.com/markmandel))
- Fix GameServerAllocation preferred documentation [\#2029](https://github.com/agones-dev/agones/pull/2029) ([markmandel](https://github.com/markmandel))

**Security fixes:**

- Update Node.js dependencies and remove bundled sub-dependency [\#2040](https://github.com/agones-dev/agones/pull/2040) ([steven-supersolid](https://github.com/steven-supersolid))

**Closed issues:**

- Update documentation to describe why the Agones sidecar uses the prefix `agones.dev/sdk-` [\#2053](https://github.com/agones-dev/agones/issues/2053)
- Release 1.13.0 [\#2025](https://github.com/agones-dev/agones/issues/2025)

**Merged pull requests:**

- Release 1.14.0-rc [\#2067](https://github.com/agones-dev/agones/pull/2067) ([roberthbailey](https://github.com/roberthbailey))
- Add docs for running windows game servers [\#2065](https://github.com/agones-dev/agones/pull/2065) ([roberthbailey](https://github.com/roberthbailey))
- Updating code documentation for Labels [\#2060](https://github.com/agones-dev/agones/pull/2060) ([domgreen](https://github.com/domgreen))
- Cleanup: Start ➡ Run for all components. [\#2058](https://github.com/agones-dev/agones/pull/2058) ([markmandel](https://github.com/markmandel))
- Explanation for SetLabel/Annotation prefixes [\#2057](https://github.com/agones-dev/agones/pull/2057) ([markmandel](https://github.com/markmandel))
- Update the recommended machine type to use when creating GKE clusters. [\#2056](https://github.com/agones-dev/agones/pull/2056) ([roberthbailey](https://github.com/roberthbailey))
- Update the website to use simple-game-server version 0.3. [\#2049](https://github.com/agones-dev/agones/pull/2049) ([roberthbailey](https://github.com/roberthbailey))
- Add a security policy that uses g.co/vulnz for intake [\#2044](https://github.com/agones-dev/agones/pull/2044) ([roberthbailey](https://github.com/roberthbailey))
- Where did the TCP e2e test go? [\#2043](https://github.com/agones-dev/agones/pull/2043) ([markmandel](https://github.com/markmandel))
- camelCase playerID in log statements [\#2041](https://github.com/agones-dev/agones/pull/2041) ([mthssdrbrg](https://github.com/mthssdrbrg))
- Update app labels to all use "agones.name" and not "agones.fullname" [\#2039](https://github.com/agones-dev/agones/pull/2039) ([roberthbailey](https://github.com/roberthbailey))
- Add Google AIP docs to CONTRIBUTING.md [\#2036](https://github.com/agones-dev/agones/pull/2036) ([markmandel](https://github.com/markmandel))
- Allocation: Drop stale GameServer on conflict [\#2032](https://github.com/agones-dev/agones/pull/2032) ([markmandel](https://github.com/markmandel))
- Remove erroneous comma in feature gate docs. [\#2030](https://github.com/agones-dev/agones/pull/2030) ([roberthbailey](https://github.com/roberthbailey))
- Preparation for 1.14.0 [\#2027](https://github.com/agones-dev/agones/pull/2027) ([markmandel](https://github.com/markmandel))

## [v1.13.0](https://github.com/agones-dev/agones/tree/v1.13.0) (2021-03-16)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.13.0-rc...v1.13.0)

**Closed issues:**

- Release 1.13.0-rc [\#2020](https://github.com/agones-dev/agones/issues/2020)

**Merged pull requests:**

- Release 1.13.0 [\#2026](https://github.com/agones-dev/agones/pull/2026) ([markmandel](https://github.com/markmandel))
- Add links to Allocator Service APIs [\#2023](https://github.com/agones-dev/agones/pull/2023) ([jhowcrof](https://github.com/jhowcrof))
- Tweaks to release checklist. [\#2022](https://github.com/agones-dev/agones/pull/2022) ([markmandel](https://github.com/markmandel))

## [v1.13.0-rc](https://github.com/agones-dev/agones/tree/v1.13.0-rc) (2021-03-10)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.12.0...v1.13.0-rc)

**Breaking changes:**

- Upgrade to Kubernetes 1.18 [\#1971](https://github.com/agones-dev/agones/issues/1971)
- Update website supported Kubernetes [\#2008](https://github.com/agones-dev/agones/pull/2008) ([markmandel](https://github.com/markmandel))
- Update client-go to support Kubernetes 1.18.0  [\#1998](https://github.com/agones-dev/agones/pull/1998) ([markmandel](https://github.com/markmandel))
- Remove simple-tcp and simple-udp [\#1992](https://github.com/agones-dev/agones/pull/1992) ([markmandel](https://github.com/markmandel))
- Upgrade dev tooling kubectl to 1.18 [\#1989](https://github.com/agones-dev/agones/pull/1989) ([markmandel](https://github.com/markmandel))
- Upgrade Terraform to 1.18 [\#1988](https://github.com/agones-dev/agones/pull/1988) ([markmandel](https://github.com/markmandel))

**Implemented enhancements:**

- Update default branch to `main` [\#1798](https://github.com/agones-dev/agones/issues/1798)
- Allow no ports for GameServer [\#749](https://github.com/agones-dev/agones/issues/749)
- Added Cubxity/AgonesKt to third party libraries [\#2013](https://github.com/agones-dev/agones/pull/2013) ([Cubxity](https://github.com/Cubxity))
- Update to PodTemplateSpec for 1.18 [\#2007](https://github.com/agones-dev/agones/pull/2007) ([markmandel](https://github.com/markmandel))
- Add support for empty ports [\#2006](https://github.com/agones-dev/agones/pull/2006) ([ItsKev](https://github.com/ItsKev))
- add Vela Games to companies that use Agones [\#2003](https://github.com/agones-dev/agones/pull/2003) ([comerford](https://github.com/comerford))
- Add RollTable to Companies that use Agones [\#2002](https://github.com/agones-dev/agones/pull/2002) ([NullSoldier](https://github.com/NullSoldier))
- Add Space Game logo to Agones site [\#2000](https://github.com/agones-dev/agones/pull/2000) ([NBoychev](https://github.com/NBoychev))
- Add WebSocket capability to WatchGameServer REST API [\#1999](https://github.com/agones-dev/agones/pull/1999) ([highlyunavailable](https://github.com/highlyunavailable))

**Fixed bugs:**

- Flaky: e2e TestGameServerReadyAllocateReady [\#2016](https://github.com/agones-dev/agones/issues/2016)
- GameServerAllocationPolicy with empty AllocationEndpoints errors on allocate [\#2011](https://github.com/agones-dev/agones/issues/2011)
- Example from Access Agones via Kubernetes API failing to compile [\#1982](https://github.com/agones-dev/agones/issues/1982)
- Unable to COMPILE after added agones plugin in UE4.26 [\#1940](https://github.com/agones-dev/agones/issues/1940)
- Reduce e2e test parrallelism from 64 to 32 [\#2019](https://github.com/agones-dev/agones/pull/2019) ([markmandel](https://github.com/markmandel))
- Whoops! Websocket documentation should be hidden [\#2015](https://github.com/agones-dev/agones/pull/2015) ([markmandel](https://github.com/markmandel))
- Return last result even if all multicluster allocations fail [\#2012](https://github.com/agones-dev/agones/pull/2012) ([highlyunavailable](https://github.com/highlyunavailable))
- Fix bug in webhook docs after example switch [\#1996](https://github.com/agones-dev/agones/pull/1996) ([markmandel](https://github.com/markmandel))
- Fixed: Multi namespace support for client secrets in helm template service/allocation.yaml [\#1984](https://github.com/agones-dev/agones/pull/1984) ([nagodon](https://github.com/nagodon))
- Move build tooling to helm upgrade --atomic [\#1980](https://github.com/agones-dev/agones/pull/1980) ([markmandel](https://github.com/markmandel))
- \[UrealSDK\] Creating requests should work in all versions of UE4 [\#1944](https://github.com/agones-dev/agones/pull/1944) ([domgreen](https://github.com/domgreen))

**Closed issues:**

- Release 1.12.0 [\#1977](https://github.com/agones-dev/agones/issues/1977)
- Deprecate and remove the udp-server and tcp-server images [\#1890](https://github.com/agones-dev/agones/issues/1890)

**Merged pull requests:**

- Release 1.13.0-rc [\#2021](https://github.com/agones-dev/agones/pull/2021) ([markmandel](https://github.com/markmandel))
- Fix a link to the website. [\#2017](https://github.com/agones-dev/agones/pull/2017) ([roberthbailey](https://github.com/roberthbailey))
- Stop logging Pods without Nodes yet as an Error [\#2014](https://github.com/agones-dev/agones/pull/2014) ([markmandel](https://github.com/markmandel))
- Update Dev Kind & Minikube to 1.18 [\#2010](https://github.com/agones-dev/agones/pull/2010) ([markmandel](https://github.com/markmandel))
- Update CRD API reference to Kubernetes 1.18 [\#2009](https://github.com/agones-dev/agones/pull/2009) ([markmandel](https://github.com/markmandel))
- Add extra resource to DGS prereq knowledge section [\#2004](https://github.com/agones-dev/agones/pull/2004) ([markmandel](https://github.com/markmandel))
- Minor grammar fix [\#1995](https://github.com/agones-dev/agones/pull/1995) ([bleakley](https://github.com/bleakley))
- Terraform helm module fix deprecated version declaration [\#1994](https://github.com/agones-dev/agones/pull/1994) ([comerford](https://github.com/comerford))
- Add "gke" prefix to TCP e2e firewall [\#1991](https://github.com/agones-dev/agones/pull/1991) ([markmandel](https://github.com/markmandel))
- Upgrade crd-client example [\#1986](https://github.com/agones-dev/agones/pull/1986) ([markmandel](https://github.com/markmandel))
- Ignore Rust SDK Target folder [\#1985](https://github.com/agones-dev/agones/pull/1985) ([markmandel](https://github.com/markmandel))
- Update all reference of `master` to `main` [\#1983](https://github.com/agones-dev/agones/pull/1983) ([markmandel](https://github.com/markmandel))
- Preparation for 1.13.0 [\#1979](https://github.com/agones-dev/agones/pull/1979) ([markmandel](https://github.com/markmandel))

## [v1.12.0](https://github.com/agones-dev/agones/tree/v1.12.0) (2021-02-02)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.12.0-rc...v1.12.0)

**Fixed bugs:**

- Minikube [\#1973](https://github.com/agones-dev/agones/issues/1973)
- gRPC allocate can't get some metrics [\#1855](https://github.com/agones-dev/agones/issues/1855)
- Fix badly formatted feature tags in GameServer specification docs [\#1975](https://github.com/agones-dev/agones/pull/1975) ([edmundlam](https://github.com/edmundlam))
- Fix Minikube \#1973 [\#1974](https://github.com/agones-dev/agones/pull/1974) ([rolfedh](https://github.com/rolfedh))
- Fixed getting latency metrics [\#1969](https://github.com/agones-dev/agones/pull/1969) ([8398a7](https://github.com/8398a7))

**Closed issues:**

- Collaborator Request [\#1972](https://github.com/agones-dev/agones/issues/1972)
- Release 1.12.0-rc [\#1966](https://github.com/agones-dev/agones/issues/1966)

**Merged pull requests:**

- Release 1.12.0 [\#1978](https://github.com/agones-dev/agones/pull/1978) ([markmandel](https://github.com/markmandel))
- Limit the disableTLS to only gRPC API [\#1968](https://github.com/agones-dev/agones/pull/1968) ([pooneh-m](https://github.com/pooneh-m))

## [v1.12.0-rc](https://github.com/agones-dev/agones/tree/v1.12.0-rc) (2021-01-26)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.11.0...v1.12.0-rc)

**Breaking changes:**

- Move SDKWatchSendOnExecute to Beta [\#1904](https://github.com/agones-dev/agones/issues/1904)
- Move `SDKWatchSendOnExecute` to Beta stage. [\#1960](https://github.com/agones-dev/agones/pull/1960) ([markmandel](https://github.com/markmandel))
- Utilize ExternalDNS as well as ExternalIP [\#1928](https://github.com/agones-dev/agones/pull/1928) ([nanasi880](https://github.com/nanasi880))

**Implemented enhancements:**

- Utilize ExternalDNS as well as ExternalIP [\#1921](https://github.com/agones-dev/agones/issues/1921)
- Move "Port Allocations to Multiple Containers" \> Stable [\#1773](https://github.com/agones-dev/agones/issues/1773)
- Move ContainerPortAllocation to Stable [\#1961](https://github.com/agones-dev/agones/pull/1961) ([markmandel](https://github.com/markmandel))
- CRD OpenAPI Spec for ObjectMeta & PodTemplateSpec [\#1956](https://github.com/agones-dev/agones/pull/1956) ([markmandel](https://github.com/markmandel))
- Add a "why" section for the Allocator Service documentation [\#1953](https://github.com/agones-dev/agones/pull/1953) ([markmandel](https://github.com/markmandel))
- Add nodeSelector property to Agones helm chart for Allocator [\#1946](https://github.com/agones-dev/agones/pull/1946) ([yeslayla](https://github.com/yeslayla))

**Fixed bugs:**

- error updating fleetautoscaler status when LastScaleTime is nil [\#1951](https://github.com/agones-dev/agones/issues/1951)
- Not sure how to do nc on windows [\#1943](https://github.com/agones-dev/agones/issues/1943)
- Error executing simple gameserver tutorial \(node.js\) [\#1562](https://github.com/agones-dev/agones/issues/1562)
- Fix data race in sdkserver.go [\#1965](https://github.com/agones-dev/agones/pull/1965) ([markmandel](https://github.com/markmandel))
- Refactored sdk functions to always return &alpha.Bool{} instead of nil [\#1958](https://github.com/agones-dev/agones/pull/1958) ([justjoeyuk](https://github.com/justjoeyuk))
- nullable lastScaleTime on FleetAutoScaler CRD [\#1952](https://github.com/agones-dev/agones/pull/1952) ([markmandel](https://github.com/markmandel))
- Fix Twitter linkcheck failure. [\#1947](https://github.com/agones-dev/agones/pull/1947) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- Release 1.11.0 [\#1936](https://github.com/agones-dev/agones/issues/1936)
- Documentation: GameServer updates are not supported. No pods are created after switching image for GameServer [\#1724](https://github.com/agones-dev/agones/issues/1724)

**Merged pull requests:**

- Release 1.12.0-rc [\#1967](https://github.com/agones-dev/agones/pull/1967) ([markmandel](https://github.com/markmandel))
- ObjectMeta should use additionalProperties [\#1949](https://github.com/agones-dev/agones/pull/1949) ([markmandel](https://github.com/markmandel))
- Add Windows note for netcat in getting started [\#1948](https://github.com/agones-dev/agones/pull/1948) ([markmandel](https://github.com/markmandel))
- Preparation for 1.12.0 release [\#1938](https://github.com/agones-dev/agones/pull/1938) ([markmandel](https://github.com/markmandel))
- Update documentation to note there is no GameServer update support [\#1935](https://github.com/agones-dev/agones/pull/1935) ([yeslayla](https://github.com/yeslayla))

## [v1.11.0](https://github.com/agones-dev/agones/tree/v1.11.0) (2020-12-22)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.11.0-rc...v1.11.0)

**Implemented enhancements:**

- Proposal: Provide a flag to disable mTLS for agones-allocator [\#1590](https://github.com/agones-dev/agones/issues/1590)

**Closed issues:**

- Release 1.11.0-rc [\#1931](https://github.com/agones-dev/agones/issues/1931)

**Merged pull requests:**

- 1.11.0 Release [\#1937](https://github.com/agones-dev/agones/pull/1937) ([markmandel](https://github.com/markmandel))

## [v1.11.0-rc](https://github.com/agones-dev/agones/tree/v1.11.0-rc) (2020-12-15)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.10.0...v1.11.0-rc)

**Breaking changes:**

- Upgrade to Kubernetes 1.17 [\#1824](https://github.com/agones-dev/agones/issues/1824)
- Update apiextensions.k8s.io/v1beta1 to v1 for all CRDs [\#1799](https://github.com/agones-dev/agones/issues/1799)
- Move apiregistration & admissionregistration to v1 [\#1918](https://github.com/agones-dev/agones/pull/1918) ([markmandel](https://github.com/markmandel))
- Updated terraform scripts to 1.17 [\#1916](https://github.com/agones-dev/agones/pull/1916) ([markmandel](https://github.com/markmandel))
- Update client-go to 0.17.14 [\#1913](https://github.com/agones-dev/agones/pull/1913) ([markmandel](https://github.com/markmandel))
- Adding SAN to fleet autoscaler certs and updating documentation [\#1910](https://github.com/agones-dev/agones/pull/1910) ([kdima](https://github.com/kdima))
- Move CRDs from v1beta1 to v1 [\#1909](https://github.com/agones-dev/agones/pull/1909) ([markmandel](https://github.com/markmandel))
- Upgrade terraform and test clusters to 1.17 [\#1901](https://github.com/agones-dev/agones/pull/1901) ([markmandel](https://github.com/markmandel))

**Implemented enhancements:**

- Remove support / docs for helm v2 [\#1853](https://github.com/agones-dev/agones/issues/1853)
- grpc-gateway powered REST API for MultiCluster Allocation [\#1495](https://github.com/agones-dev/agones/issues/1495)
- Support Agones sidecar Windows build [\#110](https://github.com/agones-dev/agones/issues/110)
- Tooling to review pprof heaps [\#1927](https://github.com/agones-dev/agones/pull/1927) ([markmandel](https://github.com/markmandel))
- Move supported site K8s version to shortcodes [\#1917](https://github.com/agones-dev/agones/pull/1917) ([markmandel](https://github.com/markmandel))
- Adding rest to allocation endpoint [\#1902](https://github.com/agones-dev/agones/pull/1902) ([kdima](https://github.com/kdima))
- \#54 Preliminary Windows Image Support [\#1894](https://github.com/agones-dev/agones/pull/1894) ([jeremyje](https://github.com/jeremyje))

**Fixed bugs:**

- validations.agones.dev and mutations.agones.dev don't declare side effects [\#1891](https://github.com/agones-dev/agones/issues/1891)
- Agones Game Server Client SDKs [\#1854](https://github.com/agones-dev/agones/issues/1854)
- Pin the postcss-cli version [\#1930](https://github.com/agones-dev/agones/pull/1930) ([markmandel](https://github.com/markmandel))
- Oops, k8s-api shortcode only pointed at one URL [\#1924](https://github.com/agones-dev/agones/pull/1924) ([markmandel](https://github.com/markmandel))

**Security fixes:**

- Updated Go to 1.14.12 [\#1900](https://github.com/agones-dev/agones/pull/1900) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- Release 1.10.0 [\#1892](https://github.com/agones-dev/agones/issues/1892)
- Proposal: Multi-Cluster Allocation Policies [\#597](https://github.com/agones-dev/agones/issues/597)

**Merged pull requests:**

- 1.11.0 Release Candidate [\#1933](https://github.com/agones-dev/agones/pull/1933) ([markmandel](https://github.com/markmandel))
- Add some logging to help determine which game server / namespace is causing this particular error [\#1929](https://github.com/agones-dev/agones/pull/1929) ([roberthbailey](https://github.com/roberthbailey))
- Upgrade prow to 1.17 [\#1926](https://github.com/agones-dev/agones/pull/1926) ([markmandel](https://github.com/markmandel))
- Update crd-doc-config.json to Kubernetes 1.17 [\#1925](https://github.com/agones-dev/agones/pull/1925) ([markmandel](https://github.com/markmandel))
- Remove K8s API reference links from yaml examples [\#1923](https://github.com/agones-dev/agones/pull/1923) ([markmandel](https://github.com/markmandel))
- Updated CRD libraries to v1 [\#1922](https://github.com/agones-dev/agones/pull/1922) ([markmandel](https://github.com/markmandel))
- Move Scaling resource to v1betav1 to v1 [\#1920](https://github.com/agones-dev/agones/pull/1920) ([markmandel](https://github.com/markmandel))
- Move admissionregistration to v1 [\#1919](https://github.com/agones-dev/agones/pull/1919) ([markmandel](https://github.com/markmandel))
- Update Kind dev tooling to 1.17.3 [\#1915](https://github.com/agones-dev/agones/pull/1915) ([markmandel](https://github.com/markmandel))
- Issue \#1854: Fix getplayercount\(\) description on index  [\#1912](https://github.com/agones-dev/agones/pull/1912) ([bnwhorton](https://github.com/bnwhorton))
- Merge allocation service annotation metadata [\#1908](https://github.com/agones-dev/agones/pull/1908) ([markmandel](https://github.com/markmandel))
- Update minikube dev tooling [\#1906](https://github.com/agones-dev/agones/pull/1906) ([markmandel](https://github.com/markmandel))
- csharp SDK: exposes the Alpha SDK via the main interface [\#1896](https://github.com/agones-dev/agones/pull/1896) ([rcreasey](https://github.com/rcreasey))
- Preparation for 1.11.0 [\#1895](https://github.com/agones-dev/agones/pull/1895) ([markmandel](https://github.com/markmandel))

## [v1.10.0](https://github.com/agones-dev/agones/tree/v1.10.0) (2020-11-10)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.10.0-rc...v1.10.0)

**Fixed bugs:**

- helm install fails since helm3 requires a name [\#1886](https://github.com/agones-dev/agones/issues/1886)
- Fix formatting on the 1.10 blog post. [\#1889](https://github.com/agones-dev/agones/pull/1889) ([roberthbailey](https://github.com/roberthbailey))
- Fix GameServer count per type graph back to pie [\#1888](https://github.com/agones-dev/agones/pull/1888) ([markmandel](https://github.com/markmandel))
- Updates to the release template [\#1887](https://github.com/agones-dev/agones/pull/1887) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- Release 1.10.0 [\#1884](https://github.com/agones-dev/agones/issues/1884)

**Merged pull requests:**

- Release 1.10.0 [\#1893](https://github.com/agones-dev/agones/pull/1893) ([markmandel](https://github.com/markmandel))

## [v1.10.0-rc](https://github.com/agones-dev/agones/tree/v1.10.0-rc) (2020-11-03)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.9.0...v1.10.0-rc)

**Breaking changes:**

- Remove the documentation for helm v2 [\#1859](https://github.com/agones-dev/agones/pull/1859) ([roberthbailey](https://github.com/roberthbailey))

**Implemented enhancements:**

- Adding allocator log level [\#1879](https://github.com/agones-dev/agones/issues/1879)
- Adding allocator resources [\#1873](https://github.com/agones-dev/agones/issues/1873)
- Add troubleshooting section to allocator and multicluster allocation [\#1866](https://github.com/agones-dev/agones/issues/1866)
- Helm setting the annotation of controller and allocator [\#1848](https://github.com/agones-dev/agones/issues/1848)
- Change the multi-cluster allocation feature to stable version [\#1780](https://github.com/agones-dev/agones/issues/1780)
- Updated C\# documentation to use NuGet package [\#1769](https://github.com/agones-dev/agones/issues/1769)
- Documented assumed prerequisite knowledge for the project [\#1759](https://github.com/agones-dev/agones/issues/1759)
- Multicluster: Add gRPC dial timeout [\#1700](https://github.com/agones-dev/agones/issues/1700)
- Add new projects to Third Party section of the site [\#1882](https://github.com/agones-dev/agones/pull/1882) ([danieloliveira079](https://github.com/danieloliveira079))
- Add log level setting in allocator [\#1880](https://github.com/agones-dev/agones/pull/1880) ([8398a7](https://github.com/8398a7))
- Add troubleshooting for allocation gRPC request [\#1878](https://github.com/agones-dev/agones/pull/1878) ([pooneh-m](https://github.com/pooneh-m))
- Add allocator resources [\#1874](https://github.com/agones-dev/agones/pull/1874) ([8398a7](https://github.com/8398a7))
- \[Unreal SDK\] Added a response code check to some functions [\#1870](https://github.com/agones-dev/agones/pull/1870) ([dotcom](https://github.com/dotcom))
- Built tools: Update install with Allocation certs [\#1869](https://github.com/agones-dev/agones/pull/1869) ([markmandel](https://github.com/markmandel))
- Add gRPC load test for allocation service [\#1867](https://github.com/agones-dev/agones/pull/1867) ([ilkercelikyilmaz](https://github.com/ilkercelikyilmaz))
- Add pod annotations [\#1849](https://github.com/agones-dev/agones/pull/1849) ([8398a7](https://github.com/8398a7))
- Useful Unreal links [\#1846](https://github.com/agones-dev/agones/pull/1846) ([domgreen](https://github.com/domgreen))
- Make the force\_update option configurable in Helm/Terraform [\#1844](https://github.com/agones-dev/agones/pull/1844) ([comerford](https://github.com/comerford))
- \[Doc\] Mark multicluster allocation feature as stable [\#1843](https://github.com/agones-dev/agones/pull/1843) ([pooneh-m](https://github.com/pooneh-m))
- Docs: Prerequisite Knowledge section [\#1821](https://github.com/agones-dev/agones/pull/1821) ([markmandel](https://github.com/markmandel))
- adding timeout to remote cluster allocate call and adding total timeout to allocate [\#1815](https://github.com/agones-dev/agones/pull/1815) ([kdima](https://github.com/kdima))
- Docs: Update C\# SDK docs page [\#1796](https://github.com/agones-dev/agones/pull/1796) ([Reousa](https://github.com/Reousa))

**Fixed bugs:**

- Allocating multicluster using GameServerAllocation API fails with missing Kind [\#1864](https://github.com/agones-dev/agones/issues/1864)
- Allocator throttled by default K8s Client requests per second [\#1852](https://github.com/agones-dev/agones/issues/1852)
- Upgrading from 1.7.0 to 1.8.0 using the helm module for terraform fails with force\_update=true [\#1767](https://github.com/agones-dev/agones/issues/1767)
- Update helm installation to include a step to update helm repo [\#1881](https://github.com/agones-dev/agones/pull/1881) ([pooneh-m](https://github.com/pooneh-m))
- Fix kind on GameServerAllocation converter [\#1876](https://github.com/agones-dev/agones/pull/1876) ([pooneh-m](https://github.com/pooneh-m))
- Fix memory leak in client-go/workqueue [\#1871](https://github.com/agones-dev/agones/pull/1871) ([markmandel](https://github.com/markmandel))
- Add TypeMeta to GameServerAllocation when doing convertion [\#1865](https://github.com/agones-dev/agones/pull/1865) ([pooneh-m](https://github.com/pooneh-m))
- Add QPS settings to Allocation endpoints [\#1863](https://github.com/agones-dev/agones/pull/1863) ([markmandel](https://github.com/markmandel))
- Add more more retries to htmltest [\#1861](https://github.com/agones-dev/agones/pull/1861) ([markmandel](https://github.com/markmandel))

**Security fixes:**

- update node.js dependencies [\#1868](https://github.com/agones-dev/agones/pull/1868) ([steven-supersolid](https://github.com/steven-supersolid))

**Closed issues:**

- Release 1.9.0 [\#1834](https://github.com/agones-dev/agones/issues/1834)
- Metrics: link to helm repository is deprecated, install command as well  [\#1829](https://github.com/agones-dev/agones/issues/1829)

**Merged pull requests:**

- Release 1.10.0-rc [\#1885](https://github.com/agones-dev/agones/pull/1885) ([markmandel](https://github.com/markmandel))
- Move the loadBalancerIP to helm installation as best effort [\#1877](https://github.com/agones-dev/agones/pull/1877) ([pooneh-m](https://github.com/pooneh-m))
- Fixed error message [\#1875](https://github.com/agones-dev/agones/pull/1875) ([pooneh-m](https://github.com/pooneh-m))
- MultiCluster Allocation: Cleanup on error logs [\#1862](https://github.com/agones-dev/agones/pull/1862) ([markmandel](https://github.com/markmandel))
- Remove Make commands from Metrics documentation [\#1858](https://github.com/agones-dev/agones/pull/1858) ([markmandel](https://github.com/markmandel))
- Build Tools: Update Prometheus and Grafana [\#1857](https://github.com/agones-dev/agones/pull/1857) ([markmandel](https://github.com/markmandel))
- Update prometheus and grafana [\#1850](https://github.com/agones-dev/agones/pull/1850) ([8398a7](https://github.com/8398a7))
- Expand feature freeze details during RC. [\#1847](https://github.com/agones-dev/agones/pull/1847) ([markmandel](https://github.com/markmandel))
- Revert "\[Doc\] Mark multicluster allocation feature as stable" [\#1842](https://github.com/agones-dev/agones/pull/1842) ([pooneh-m](https://github.com/pooneh-m))
- Preparation for 1.10.0 [\#1836](https://github.com/agones-dev/agones/pull/1836) ([markmandel](https://github.com/markmandel))
- \[Doc\] Mark multicluster allocation feature as stable [\#1831](https://github.com/agones-dev/agones/pull/1831) ([pooneh-m](https://github.com/pooneh-m))

## [v1.9.0](https://github.com/agones-dev/agones/tree/v1.9.0) (2020-09-29)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.9.0-rc...v1.9.0)

**Closed issues:**

- Release 1.9.0-rc [\#1827](https://github.com/agones-dev/agones/issues/1827)
- \[Docs\] Multi-cluster Allocation [\#1582](https://github.com/agones-dev/agones/issues/1582)

**Merged pull requests:**

- Release 1.9.0 [\#1835](https://github.com/agones-dev/agones/pull/1835) ([markmandel](https://github.com/markmandel))

## [v1.9.0-rc](https://github.com/agones-dev/agones/tree/v1.9.0-rc) (2020-09-23)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.8.0...v1.9.0-rc)

**Breaking changes:**

- Upgrade to Kubernetes 1.16 [\#1649](https://github.com/agones-dev/agones/issues/1649)
- Upgrade Kubectl to 1.16.15 [\#1806](https://github.com/agones-dev/agones/pull/1806) ([markmandel](https://github.com/markmandel))
- Upgrade client-go and apimachinery to 0.16.15 [\#1794](https://github.com/agones-dev/agones/pull/1794) ([markmandel](https://github.com/markmandel))
- Update GKE Terraform clusters to 1.16 [\#1772](https://github.com/agones-dev/agones/pull/1772) ([markmandel](https://github.com/markmandel))

**Implemented enhancements:**

- \[Doc\] add a caveat on setting expiration for cert-manager certificate resources [\#1781](https://github.com/agones-dev/agones/issues/1781)
- Helm chart: allow to specify loadBalancerIP [\#1709](https://github.com/agones-dev/agones/issues/1709)
- Terraform, Helm module install: Allow gameserver namespaces and port ranges to be specified in terraform [\#1692](https://github.com/agones-dev/agones/issues/1692)
- Support using the same port for both TCP/UDP forwarding [\#1523](https://github.com/agones-dev/agones/issues/1523)
- Write Tests for Terraform configs [\#1227](https://github.com/agones-dev/agones/issues/1227)
- Add player tracking and shutdown to the supertuxkart example server [\#1825](https://github.com/agones-dev/agones/pull/1825) ([sudermanjr](https://github.com/sudermanjr))
- Add logging for the client certificate verification [\#1812](https://github.com/agones-dev/agones/pull/1812) ([pooneh-m](https://github.com/pooneh-m))
- Troubleshooting - namespace stuck terminating [\#1795](https://github.com/agones-dev/agones/pull/1795) ([domgreen](https://github.com/domgreen))
- Add load balancer configuration for Helm options [\#1793](https://github.com/agones-dev/agones/pull/1793) ([yoshd](https://github.com/yoshd))
- Added option to hardcode load balancer IP for allocator. [\#1766](https://github.com/agones-dev/agones/pull/1766) ([devloop0](https://github.com/devloop0))
- Add TCPUDP protocol [\#1764](https://github.com/agones-dev/agones/pull/1764) ([Bmandk](https://github.com/Bmandk))
- Publish to NuGet for Csharp SDK [\#1753](https://github.com/agones-dev/agones/pull/1753) ([markmandel](https://github.com/markmandel))
- Add Terraform example for GKE custom VPC deployment [\#1697](https://github.com/agones-dev/agones/pull/1697) ([moesy](https://github.com/moesy))
- Fix Fleets RollingUpdate [\#1626](https://github.com/agones-dev/agones/pull/1626) ([aLekSer](https://github.com/aLekSer))

**Fixed bugs:**

- Wrong `Alpha: GetPlayerCount` description in the REST docs [\#1810](https://github.com/agones-dev/agones/issues/1810)
- Flaky SDK Conformance Tests [\#1779](https://github.com/agones-dev/agones/issues/1779)
- agones-system gets stuck in "Terminating" [\#1778](https://github.com/agones-dev/agones/issues/1778)
- Rolling updates should wait for batches to become healthy before iterating [\#1625](https://github.com/agones-dev/agones/issues/1625)
- Fix 404 in AWS/EKS documentation [\#1820](https://github.com/agones-dev/agones/pull/1820) ([markmandel](https://github.com/markmandel))
- Pin npm autoprefixer package for site generation [\#1818](https://github.com/agones-dev/agones/pull/1818) ([markmandel](https://github.com/markmandel))
- Docs: Fix rest `GetPlayerCount` description [\#1811](https://github.com/agones-dev/agones/pull/1811) ([Reousa](https://github.com/Reousa))
- Flaky: TestControllerGameServersNodeState [\#1805](https://github.com/agones-dev/agones/pull/1805) ([markmandel](https://github.com/markmandel))
- Flaky: TestControllerSyncUnhealthyGameServers [\#1803](https://github.com/agones-dev/agones/pull/1803) ([markmandel](https://github.com/markmandel))
- Make Unreal lambda bindings on the AgonesComponent safe [\#1775](https://github.com/agones-dev/agones/pull/1775) ([achynes](https://github.com/achynes))
- Pass port into autoscaler url from webhook policy [\#1765](https://github.com/agones-dev/agones/pull/1765) ([andrewgrundy](https://github.com/andrewgrundy))

**Closed issues:**

- Unity Game Server Client SDK [\#1809](https://github.com/agones-dev/agones/issues/1809)
- Release 1.8.0 [\#1758](https://github.com/agones-dev/agones/issues/1758)

**Merged pull requests:**

- Release 1.9.0-rc [\#1828](https://github.com/agones-dev/agones/pull/1828) ([markmandel](https://github.com/markmandel))
- Corrected gke docs 'release-release' typo [\#1826](https://github.com/agones-dev/agones/pull/1826) ([eddie-knight](https://github.com/eddie-knight))
- Remove the warning about the unity SDK not being feature complete. [\#1817](https://github.com/agones-dev/agones/pull/1817) ([roberthbailey](https://github.com/roberthbailey))
- Website: Update API docs to Kubernetes 1.16 [\#1814](https://github.com/agones-dev/agones/pull/1814) ([aLekSer](https://github.com/aLekSer))
- Website: Update properly to Kubernetes 1.16 [\#1813](https://github.com/agones-dev/agones/pull/1813) ([aLekSer](https://github.com/aLekSer))
- CI: Wait for tests step before sdk-conformance [\#1808](https://github.com/agones-dev/agones/pull/1808) ([aLekSer](https://github.com/aLekSer))
- Rerun Agones CRD client generation [\#1807](https://github.com/agones-dev/agones/pull/1807) ([markmandel](https://github.com/markmandel))
- DriveBy - fix link [\#1804](https://github.com/agones-dev/agones/pull/1804) ([domgreen](https://github.com/domgreen))
- \[Doc\] Update allocator service & multi-cluster allocation documentation [\#1802](https://github.com/agones-dev/agones/pull/1802) ([pooneh-m](https://github.com/pooneh-m))
- Add lock to to sdk-conformance compare\(\) [\#1801](https://github.com/agones-dev/agones/pull/1801) ([markmandel](https://github.com/markmandel))
- UE4 small nit about float to float comparison [\#1792](https://github.com/agones-dev/agones/pull/1792) ([domgreen](https://github.com/domgreen))
- Docs: updating GKE and AKS Kubernetes versions [\#1791](https://github.com/agones-dev/agones/pull/1791) ([aLekSer](https://github.com/aLekSer))
- Docs and TF: Update EKS Kubernetes version to use 1.16 [\#1790](https://github.com/agones-dev/agones/pull/1790) ([aLekSer](https://github.com/aLekSer))
- Docs: updated advised version of Kubernetes to use [\#1789](https://github.com/agones-dev/agones/pull/1789) ([aLekSer](https://github.com/aLekSer))
- Adding aLekSer to approvers list [\#1788](https://github.com/agones-dev/agones/pull/1788) ([aLekSer](https://github.com/aLekSer))
- Examples: update Kubernetes version [\#1787](https://github.com/agones-dev/agones/pull/1787) ([aLekSer](https://github.com/aLekSer))
- Docs: updating Minikube kubernetes version to 1.16 [\#1786](https://github.com/agones-dev/agones/pull/1786) ([aLekSer](https://github.com/aLekSer))
- GolangCI-lint updating version to 1.30 [\#1785](https://github.com/agones-dev/agones/pull/1785) ([aLekSer](https://github.com/aLekSer))
- Doc changes for TLS and loadBalancerIP changes [\#1784](https://github.com/agones-dev/agones/pull/1784) ([devloop0](https://github.com/devloop0))
- Add Uninstall instructions when using install.yaml [\#1783](https://github.com/agones-dev/agones/pull/1783) ([aLekSer](https://github.com/aLekSer))
- Added a new 'disableTLS' flag and changed 'disableMTLS' to only disab… [\#1777](https://github.com/agones-dev/agones/pull/1777) ([devloop0](https://github.com/devloop0))
- The footnote shouldn't be part of the table. [\#1774](https://github.com/agones-dev/agones/pull/1774) ([roberthbailey](https://github.com/roberthbailey))
- Added game-server example [\#1771](https://github.com/agones-dev/agones/pull/1771) ([Bmandk](https://github.com/Bmandk))
- Preparation for 1.9.0 [\#1762](https://github.com/agones-dev/agones/pull/1762) ([markmandel](https://github.com/markmandel))
- Add Terraform GKE and Helm modules tests with Terratest [\#1483](https://github.com/agones-dev/agones/pull/1483) ([aLekSer](https://github.com/aLekSer))

## [v1.8.0](https://github.com/agones-dev/agones/tree/v1.8.0) (2020-08-18)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.8.0-rc...v1.8.0)

**Fixed bugs:**

- Content-Type: application/json; charset=utf-8 results in "Could not find deserializer" [\#1748](https://github.com/agones-dev/agones/issues/1748)
- Fix parsing the media type in GameServerAllocation [\#1749](https://github.com/agones-dev/agones/pull/1749) ([aLekSer](https://github.com/aLekSer))

**Closed issues:**

- Release 1.8.0-rc [\#1745](https://github.com/agones-dev/agones/issues/1745)

**Merged pull requests:**

- Release 1.8.0 [\#1760](https://github.com/agones-dev/agones/pull/1760) ([markmandel](https://github.com/markmandel))
- Helm default values in docs \(related to controller limits\) match chart default values [\#1755](https://github.com/agones-dev/agones/pull/1755) ([pgilfillan](https://github.com/pgilfillan))
- Best practices for game server shutdown [\#1752](https://github.com/agones-dev/agones/pull/1752) ([markmandel](https://github.com/markmandel))
- Remove Deployment Manager from build/ [\#1750](https://github.com/agones-dev/agones/pull/1750) ([markmandel](https://github.com/markmandel))

## [v1.8.0-rc](https://github.com/agones-dev/agones/tree/v1.8.0-rc) (2020-08-11)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.7.0...v1.8.0-rc)

**Breaking changes:**

- \[Discussion\] Assimilate netspeakgames/UnrealAgonesSDK [\#1683](https://github.com/agones-dev/agones/issues/1683)
- Upgrade to Kubernetes 1.15 [\#1478](https://github.com/agones-dev/agones/issues/1478)
- move Netspeak Unreal SDK into Agones Unreal SDK [\#1739](https://github.com/agones-dev/agones/pull/1739) ([domgreen](https://github.com/domgreen))

**Implemented enhancements:**

- Support for async/await syntax in Rust SDK [\#1732](https://github.com/agones-dev/agones/issues/1732)
- Add firewall name variable to GKE Terraform [\#1741](https://github.com/agones-dev/agones/pull/1741) ([markmandel](https://github.com/markmandel))
- Add Rust SDK async/await syntax support and minor improvements [\#1733](https://github.com/agones-dev/agones/pull/1733) ([yoshd](https://github.com/yoshd))
- Add extra troubleshooting steps. [\#1721](https://github.com/agones-dev/agones/pull/1721) ([markmandel](https://github.com/markmandel))
- Supports Rust Alpha SDK [\#1717](https://github.com/agones-dev/agones/pull/1717) ([yoshd](https://github.com/yoshd))
- Supports C\# Alpha SDK [\#1705](https://github.com/agones-dev/agones/pull/1705) ([yoshd](https://github.com/yoshd))
- Added allocator-client.default secret. [\#1702](https://github.com/agones-dev/agones/pull/1702) ([devloop0](https://github.com/devloop0))
- Add Custom VPC support to Terraform GKE Module [\#1695](https://github.com/agones-dev/agones/pull/1695) ([moesy](https://github.com/moesy))
- add gameserver values as configurable in helm terraform modules [\#1693](https://github.com/agones-dev/agones/pull/1693) ([comerford](https://github.com/comerford))
- Adding Fairwinds agones-allocator-client to third-party tools [\#1684](https://github.com/agones-dev/agones/pull/1684) ([sudermanjr](https://github.com/sudermanjr))
- Added new gen-install-alpha command [\#1673](https://github.com/agones-dev/agones/pull/1673) ([akremsa](https://github.com/akremsa))

**Fixed bugs:**

- Quickstart: create webhook autoscaler not working [\#1734](https://github.com/agones-dev/agones/issues/1734)
- Helm installation documentation doesn't mention --namespace on upgrade [\#1728](https://github.com/agones-dev/agones/issues/1728)
- CI: htmltest with 404 status does not treated as a failure on `make hugo-test` step [\#1712](https://github.com/agones-dev/agones/issues/1712)
- /watch/gameserver doesn't start with returning the current state [\#1703](https://github.com/agones-dev/agones/issues/1703)
- CI: make test-gen-api-docs is failing quite often [\#1690](https://github.com/agones-dev/agones/issues/1690)
- Unable to Create GKE clusters in non-default VPC \(Terraform\) [\#1641](https://github.com/agones-dev/agones/issues/1641)
- Terraform: GKE module leftovers after apply and destroy [\#1403](https://github.com/agones-dev/agones/issues/1403)
- Gameservers using nodejs sdk die with GOAWAY ENHANCE\_YOUR\_CALM too\_many\_pings [\#1299](https://github.com/agones-dev/agones/issues/1299)
- Rust SDK conflicts with dependencies using openssl [\#1201](https://github.com/agones-dev/agones/issues/1201)
- Building the cpp-simple example prints a fatal message and hangs for a long time before finishing [\#1091](https://github.com/agones-dev/agones/issues/1091)
- Terraform Helm enforcing string for set values [\#1737](https://github.com/agones-dev/agones/pull/1737) ([markmandel](https://github.com/markmandel))
- Fix fleetautoscalers webhook TLS policy [\#1736](https://github.com/agones-dev/agones/pull/1736) ([aLekSer](https://github.com/aLekSer))
- Build Terraform: Use docker image project default [\#1730](https://github.com/agones-dev/agones/pull/1730) ([markmandel](https://github.com/markmandel))
- Helm installation docs fix for missing namespace [\#1729](https://github.com/agones-dev/agones/pull/1729) ([thoraxe](https://github.com/thoraxe))
- Fix findOpenPorts portAllocator function [\#1725](https://github.com/agones-dev/agones/pull/1725) ([aLekSer](https://github.com/aLekSer))

**Security fixes:**

- Bump lodash from 4.17.15 to 4.17.19 to fix a security vulnerability. [\#1707](https://github.com/agones-dev/agones/pull/1707) ([roberthbailey](https://github.com/roberthbailey))

**Closed issues:**

- Release 1.7.0 [\#1679](https://github.com/agones-dev/agones/issues/1679)
- Agones Documentation [\#1654](https://github.com/agones-dev/agones/issues/1654)
- Collaborator Request [\#1640](https://github.com/agones-dev/agones/issues/1640)

**Merged pull requests:**

- Release 1.8.0 Release Candidate [\#1746](https://github.com/agones-dev/agones/pull/1746) ([markmandel](https://github.com/markmandel))
- Add note on GKE cluster versions [\#1743](https://github.com/agones-dev/agones/pull/1743) ([markmandel](https://github.com/markmandel))
- Switch gcloud-test-cluster to Terraform [\#1738](https://github.com/agones-dev/agones/pull/1738) ([markmandel](https://github.com/markmandel))
- Added several tests for metrics package [\#1735](https://github.com/agones-dev/agones/pull/1735) ([akremsa](https://github.com/akremsa))
- Tests update: gameserversets-controllers added missing test cases [\#1727](https://github.com/agones-dev/agones/pull/1727) ([akremsa](https://github.com/akremsa))
- Tests update: migration, pernodcounter [\#1726](https://github.com/agones-dev/agones/pull/1726) ([akremsa](https://github.com/akremsa))
- Tests update: portallocator\_test [\#1723](https://github.com/agones-dev/agones/pull/1723) ([akremsa](https://github.com/akremsa))
- Fix small typo [\#1722](https://github.com/agones-dev/agones/pull/1722) ([Alfred-Mountfield](https://github.com/Alfred-Mountfield))
- CI: increase lock time to 1 hour [\#1720](https://github.com/agones-dev/agones/pull/1720) ([aLekSer](https://github.com/aLekSer))
- Tests update: gameservers - health\_test [\#1719](https://github.com/agones-dev/agones/pull/1719) ([akremsa](https://github.com/akremsa))
- Tests update: gameservers\_test  [\#1718](https://github.com/agones-dev/agones/pull/1718) ([akremsa](https://github.com/akremsa))
- Website fix - OpenSSL Switch from master branch to more specific tag v1.1.1 [\#1716](https://github.com/agones-dev/agones/pull/1716) ([aLekSer](https://github.com/aLekSer))
- CI: website htmltest fix returning error from the loop [\#1715](https://github.com/agones-dev/agones/pull/1715) ([aLekSer](https://github.com/aLekSer))
- Use the scheduling.k8s.io/v1 API since the beta API will no longer be served by default starting with Kubernetes 1.16 [\#1714](https://github.com/agones-dev/agones/pull/1714) ([roberthbailey](https://github.com/roberthbailey))
- Tests update: gameservers-controller tests update [\#1713](https://github.com/agones-dev/agones/pull/1713) ([akremsa](https://github.com/akremsa))
- Website minor fix link and formatting [\#1708](https://github.com/agones-dev/agones/pull/1708) ([aLekSer](https://github.com/aLekSer))
- Fix a couple of broken links on the website. [\#1704](https://github.com/agones-dev/agones/pull/1704) ([roberthbailey](https://github.com/roberthbailey))
- Supress fatal message in CPP example build [\#1701](https://github.com/agones-dev/agones/pull/1701) ([aLekSer](https://github.com/aLekSer))
- Tests update: Added require statements to gameserversets package [\#1696](https://github.com/agones-dev/agones/pull/1696) ([akremsa](https://github.com/akremsa))
- Website: Update CRD API docs only if sorted files different [\#1694](https://github.com/agones-dev/agones/pull/1694) ([aLekSer](https://github.com/aLekSer))
- Fix for `gcloud-terraform-cluster` make target [\#1688](https://github.com/agones-dev/agones/pull/1688) ([aLekSer](https://github.com/aLekSer))
- Fix typo in docs index page [\#1687](https://github.com/agones-dev/agones/pull/1687) ([edmundlam](https://github.com/edmundlam))
- Tests update: Use require package in fleets package [\#1685](https://github.com/agones-dev/agones/pull/1685) ([akremsa](https://github.com/akremsa))
- Preparation for 1.8.0 Release [\#1681](https://github.com/agones-dev/agones/pull/1681) ([markmandel](https://github.com/markmandel))
- Add links to relevant AWS EKS documentation [\#1675](https://github.com/agones-dev/agones/pull/1675) ([comerford](https://github.com/comerford))
- Move CloudBuild to N1\_HIGHCPU\_32 [\#1668](https://github.com/agones-dev/agones/pull/1668) ([markmandel](https://github.com/markmandel))
- Added missing FailNow calls to sdkserver unit tests [\#1659](https://github.com/agones-dev/agones/pull/1659) ([akremsa](https://github.com/akremsa))

## [v1.7.0](https://github.com/agones-dev/agones/tree/v1.7.0) (2020-07-07)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.7.0-rc...v1.7.0)

**Implemented enhancements:**

- Alpha SDK and example for Node.js \(Player tracking\) [\#1658](https://github.com/agones-dev/agones/pull/1658) ([steven-supersolid](https://github.com/steven-supersolid))

**Fixed bugs:**

- Go unit test timeout after 10m [\#1672](https://github.com/agones-dev/agones/issues/1672)
- Fix tests timeout, WatchGameServer recent feature [\#1674](https://github.com/agones-dev/agones/pull/1674) ([aLekSer](https://github.com/aLekSer))
- Flaky: e2e namespace deletion [\#1666](https://github.com/agones-dev/agones/pull/1666) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- Release 1.7.0-rc [\#1660](https://github.com/agones-dev/agones/issues/1660)

**Merged pull requests:**

- Release 1.7.0 [\#1680](https://github.com/agones-dev/agones/pull/1680) ([markmandel](https://github.com/markmandel))
- Add C\# sdk to list of supported SDKs in overview doc [\#1678](https://github.com/agones-dev/agones/pull/1678) ([edmundlam](https://github.com/edmundlam))
- Fix typo in gameserverallocations [\#1676](https://github.com/agones-dev/agones/pull/1676) ([aLekSer](https://github.com/aLekSer))
- Update Hugo and Docsy [\#1671](https://github.com/agones-dev/agones/pull/1671) ([markmandel](https://github.com/markmandel))
- Flaky: TestPerNodeCounterRun [\#1669](https://github.com/agones-dev/agones/pull/1669) ([markmandel](https://github.com/markmandel))
- Flaky: TestSDKServerWatchGameServerFeatureSDKWatchSendOnExecute [\#1667](https://github.com/agones-dev/agones/pull/1667) ([markmandel](https://github.com/markmandel))
- Flaky TestLocal [\#1665](https://github.com/agones-dev/agones/pull/1665) ([markmandel](https://github.com/markmandel))
- Add ensure-build-image to test-go Make target [\#1664](https://github.com/agones-dev/agones/pull/1664) ([markmandel](https://github.com/markmandel))
- $\(ALPHA\_FEATURE\_GATES\) on gcloud-terraform-install [\#1663](https://github.com/agones-dev/agones/pull/1663) ([markmandel](https://github.com/markmandel))

## [v1.7.0-rc](https://github.com/agones-dev/agones/tree/v1.7.0-rc) (2020-06-30)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.6.0...v1.7.0-rc)

**Implemented enhancements:**

- WatchGameServer should immediately provide the cached GameServer [\#1630](https://github.com/agones-dev/agones/issues/1630)
- Feature Request: Allow 'serverCa' to come from secret/configmap in GameServerAllocationPolicy CRD [\#1614](https://github.com/agones-dev/agones/issues/1614)
- Automatically refresh all allocator TLS certs, not just client CA cert [\#1599](https://github.com/agones-dev/agones/issues/1599)
- Move ContainerPortAllocation to beta [\#1563](https://github.com/agones-dev/agones/issues/1563)
- Add GameServer state duration metric [\#1013](https://github.com/agones-dev/agones/issues/1013)
- Expose GameServer state change metrics [\#831](https://github.com/agones-dev/agones/issues/831)
- Update developer tooling to Helm 3 [\#1647](https://github.com/agones-dev/agones/pull/1647) ([markmandel](https://github.com/markmandel))
- Update Terraform to Helm 3 [\#1646](https://github.com/agones-dev/agones/pull/1646) ([markmandel](https://github.com/markmandel))
- Conditionally enable mtls for the allocator. [\#1645](https://github.com/agones-dev/agones/pull/1645) ([devloop0](https://github.com/devloop0))
- New feature: SDK cached gameserver [\#1642](https://github.com/agones-dev/agones/pull/1642) ([akremsa](https://github.com/akremsa))
- Adding support for refreshing TLS certs in the allocator [\#1638](https://github.com/agones-dev/agones/pull/1638) ([devloop0](https://github.com/devloop0))
- Helm 3 Install Documentation [\#1627](https://github.com/agones-dev/agones/pull/1627) ([markmandel](https://github.com/markmandel))
- Add flags which allow to pass namespace to e2e tests [\#1623](https://github.com/agones-dev/agones/pull/1623) ([akremsa](https://github.com/akremsa))
- Update docs to explicitly allow specifying ca.crt in client secret instead of serverCa field for multi-cluster allocation [\#1619](https://github.com/agones-dev/agones/pull/1619) ([robbieheywood](https://github.com/robbieheywood))
- Add port flag to example allocator-client [\#1618](https://github.com/agones-dev/agones/pull/1618) ([robbieheywood](https://github.com/robbieheywood))
- Grafana - add namespace to autoscalers dashboard [\#1615](https://github.com/agones-dev/agones/pull/1615) ([akremsa](https://github.com/akremsa))
- CI: Adding E2E cluster name as a parameter for CloudBuild [\#1611](https://github.com/agones-dev/agones/pull/1611) ([aLekSer](https://github.com/aLekSer))
- Additional commands for prometheus and grafana [\#1601](https://github.com/agones-dev/agones/pull/1601) ([akremsa](https://github.com/akremsa))
- Grafana - add namespace to distinguish fleets with the same name [\#1597](https://github.com/agones-dev/agones/pull/1597) ([akremsa](https://github.com/akremsa))
- Adding AccelByte in Companies using Agones list [\#1593](https://github.com/agones-dev/agones/pull/1593) ([accelbyte-raymond](https://github.com/accelbyte-raymond))
- Metrics: add namespace to distinguish fleets with the same name [\#1585](https://github.com/agones-dev/agones/pull/1585) ([akremsa](https://github.com/akremsa))
- Move ContainerPortAllocation to beta [\#1577](https://github.com/agones-dev/agones/pull/1577) ([akremsa](https://github.com/akremsa))
- New metric - state duration [\#1468](https://github.com/agones-dev/agones/pull/1468) ([aLekSer](https://github.com/aLekSer))

**Fixed bugs:**

- Better cleanup of namespace on e2e test failure [\#1653](https://github.com/agones-dev/agones/issues/1653)
- C\# SDK build is flakey due to a race condition [\#1639](https://github.com/agones-dev/agones/issues/1639)
- Site: fix obsolete links of Kubernetes API v1.13 in Autogenerated Agones CRD API reference [\#1617](https://github.com/agones-dev/agones/issues/1617)
- Flaky: TestAllocatorCrossNamespace [\#1603](https://github.com/agones-dev/agones/issues/1603)
- Flaky: TestFleetAggregatedPlayerStatus [\#1592](https://github.com/agones-dev/agones/issues/1592)
- HealthCheckLoop Never invoked in C\# SDK [\#1583](https://github.com/agones-dev/agones/issues/1583)
- Metrics: add namespace to distinguish same name fleets [\#1501](https://github.com/agones-dev/agones/issues/1501)
- Flaky: Csharp SDK Test [\#1651](https://github.com/agones-dev/agones/pull/1651) ([markmandel](https://github.com/markmandel))
- Load test: Fix example yaml config [\#1634](https://github.com/agones-dev/agones/pull/1634) ([aLekSer](https://github.com/aLekSer))
- Fix replacement bug in gen-api-docs.sh [\#1622](https://github.com/agones-dev/agones/pull/1622) ([markmandel](https://github.com/markmandel))
- Flaky: TestFleetAggregatedPlayerStatus [\#1606](https://github.com/agones-dev/agones/pull/1606) ([markmandel](https://github.com/markmandel))
- Flaky: TestAllocatorCrossNamespace [\#1604](https://github.com/agones-dev/agones/pull/1604) ([markmandel](https://github.com/markmandel))
- Allow env var overrides for e2e tests [\#1566](https://github.com/agones-dev/agones/pull/1566) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- Release 1.6.0 [\#1587](https://github.com/agones-dev/agones/issues/1587)
- Cleanup: Remove obsolete example of FleetAllocator service [\#1584](https://github.com/agones-dev/agones/issues/1584)
- Outdated and incomplete instructions on installing Agones using Helm [\#1494](https://github.com/agones-dev/agones/issues/1494)
- Update build image Debian version [\#1488](https://github.com/agones-dev/agones/issues/1488)
- Move support from Helm 2 ➡ Helm 3 [\#1436](https://github.com/agones-dev/agones/issues/1436)
- E2E tests should use a randomly created Namespace for testing [\#1074](https://github.com/agones-dev/agones/issues/1074)
- Terraform scripts for Agones [\#657](https://github.com/agones-dev/agones/issues/657)

**Merged pull requests:**

- Release 1.7.0-rc [\#1662](https://github.com/agones-dev/agones/pull/1662) ([markmandel](https://github.com/markmandel))
- Update Node.js dependencies in sdk and example [\#1657](https://github.com/agones-dev/agones/pull/1657) ([steven-supersolid](https://github.com/steven-supersolid))
- Update Agones developer guide to point at faster targets [\#1656](https://github.com/agones-dev/agones/pull/1656) ([markmandel](https://github.com/markmandel))
- Cleanup e2e namespaces before test start [\#1655](https://github.com/agones-dev/agones/pull/1655) ([markmandel](https://github.com/markmandel))
- E2E: Invert ContainerPortAllocation FeatureGate [\#1652](https://github.com/agones-dev/agones/pull/1652) ([aLekSer](https://github.com/aLekSer))
- Website: Change to relative reference [\#1644](https://github.com/agones-dev/agones/pull/1644) ([aLekSer](https://github.com/aLekSer))
- Fixed some formatting issues in the allocator code. [\#1643](https://github.com/agones-dev/agones/pull/1643) ([devloop0](https://github.com/devloop0))
- Add a note to the contributing docs that joining the mailing list gives access to build logs [\#1637](https://github.com/agones-dev/agones/pull/1637) ([roberthbailey](https://github.com/roberthbailey))
- Fixed GameServer State Diagram [\#1635](https://github.com/agones-dev/agones/pull/1635) ([suecideTech](https://github.com/suecideTech))
- Fixed broken link to pass tests. [\#1633](https://github.com/agones-dev/agones/pull/1633) ([devloop0](https://github.com/devloop0))
- CreateNamespace: delete dangling namespaces [\#1632](https://github.com/agones-dev/agones/pull/1632) ([akremsa](https://github.com/akremsa))
- test-e2e - Added several Fatal calls [\#1629](https://github.com/agones-dev/agones/pull/1629) ([akremsa](https://github.com/akremsa))
- Removed obsolete allocator-service-go.md [\#1624](https://github.com/agones-dev/agones/pull/1624) ([akremsa](https://github.com/akremsa))
- Fix 404 in CRD generated documentation. [\#1621](https://github.com/agones-dev/agones/pull/1621) ([markmandel](https://github.com/markmandel))
- Remove obsolete example of FleetAllocator service [\#1620](https://github.com/agones-dev/agones/pull/1620) ([akremsa](https://github.com/akremsa))
- Site: fix minor issues, obsolete helm parameters [\#1616](https://github.com/agones-dev/agones/pull/1616) ([aLekSer](https://github.com/aLekSer))
- Update AccelByte logo [\#1613](https://github.com/agones-dev/agones/pull/1613) ([accelbyte-raymond](https://github.com/accelbyte-raymond))
- Update all images to Go 1.14.4 [\#1612](https://github.com/agones-dev/agones/pull/1612) ([markmandel](https://github.com/markmandel))
- Update Rust Debian image version [\#1610](https://github.com/agones-dev/agones/pull/1610) ([aLekSer](https://github.com/aLekSer))
- Update Debian version for CPP example [\#1609](https://github.com/agones-dev/agones/pull/1609) ([aLekSer](https://github.com/aLekSer))
- Update the build image to buster [\#1608](https://github.com/agones-dev/agones/pull/1608) ([markmandel](https://github.com/markmandel))
- Added some missing helm vars documentation for Agones intall with helm. [\#1607](https://github.com/agones-dev/agones/pull/1607) ([EricFortin](https://github.com/EricFortin))
- Adding parallel to more tests. [\#1602](https://github.com/agones-dev/agones/pull/1602) ([markmandel](https://github.com/markmandel))
- Update edit-first-gameserver-go.md [\#1595](https://github.com/agones-dev/agones/pull/1595) ([minho-comcom-ai](https://github.com/minho-comcom-ai))
- Preparation for 1.7.0 [\#1589](https://github.com/agones-dev/agones/pull/1589) ([markmandel](https://github.com/markmandel))
- Improved Fleets - controller tests [\#1547](https://github.com/agones-dev/agones/pull/1547) ([akremsa](https://github.com/akremsa))
- CI: Add one more E2E tests run with all feature gates disabled [\#1546](https://github.com/agones-dev/agones/pull/1546) ([aLekSer](https://github.com/aLekSer))
- Improved fleetautoscalers - fleetautoscalers\_test.go unit tests + applyWebhookPolicy refactoring [\#1531](https://github.com/agones-dev/agones/pull/1531) ([akremsa](https://github.com/akremsa))
- Update Debian image version for SDK base [\#1511](https://github.com/agones-dev/agones/pull/1511) ([aLekSer](https://github.com/aLekSer))

## [v1.6.0](https://github.com/agones-dev/agones/tree/v1.6.0) (2020-05-26)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.6.0-rc...v1.6.0)

**Implemented enhancements:**

- C\# SDK Cleanup & Nuget Package [\#1596](https://github.com/agones-dev/agones/pull/1596) ([rcreasey](https://github.com/rcreasey))

**Fixed bugs:**

- Fix the Unreal Plugin's GetGameServer [\#1581](https://github.com/agones-dev/agones/pull/1581) ([dotcom](https://github.com/dotcom))

**Closed issues:**

- Release v1.6.0-rc [\#1573](https://github.com/agones-dev/agones/issues/1573)

**Merged pull requests:**

- Remove redundant "helm test" pod from "install.yaml" [\#1591](https://github.com/agones-dev/agones/pull/1591) ([aLekSer](https://github.com/aLekSer))
- Release 1.6.0 [\#1588](https://github.com/agones-dev/agones/pull/1588) ([markmandel](https://github.com/markmandel))
- Fix flaky Local SDK test [\#1586](https://github.com/agones-dev/agones/pull/1586) ([aLekSer](https://github.com/aLekSer))
- Warning to release checklist. [\#1580](https://github.com/agones-dev/agones/pull/1580) ([markmandel](https://github.com/markmandel))

## [v1.6.0-rc](https://github.com/agones-dev/agones/tree/v1.6.0-rc) (2020-05-20)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.5.0...v1.6.0-rc)

**Breaking changes:**

- Rename `PostAllocate` to `Allocate` in Multi Cluster Allocation Service [\#1331](https://github.com/agones-dev/agones/issues/1331)
- Proposal: Allocator service to return 400+ http status for failure  [\#1040](https://github.com/agones-dev/agones/issues/1040)
- Change the multi-cluster allocation API version to stable [\#1540](https://github.com/agones-dev/agones/pull/1540) ([pooneh-m](https://github.com/pooneh-m))
- Switch Node.js SDK grpc dependency to grpc-js [\#1529](https://github.com/agones-dev/agones/pull/1529) ([steven-supersolid](https://github.com/steven-supersolid))
- Change allocator gRPC response state to gRPC error status [\#1516](https://github.com/agones-dev/agones/pull/1516) ([pooneh-m](https://github.com/pooneh-m))
- Change rpc method from PostAllocate to Allocate [\#1513](https://github.com/agones-dev/agones/pull/1513) ([pooneh-m](https://github.com/pooneh-m))
- Update developer tooling to Kubernetes 1.15 [\#1486](https://github.com/agones-dev/agones/pull/1486) ([roberthbailey](https://github.com/roberthbailey))
- Update documentation describing when we will change the version of Kubernetes  that we support. [\#1477](https://github.com/agones-dev/agones/pull/1477) ([roberthbailey](https://github.com/roberthbailey))

**Implemented enhancements:**

- Change the multi-cluster allocation API version to stable [\#1534](https://github.com/agones-dev/agones/issues/1534)
- Proposal: For multi-cluster allocation move remote server CA to GameServerAllocationPolicy [\#1517](https://github.com/agones-dev/agones/issues/1517)
- Support annotations for ping services in the Helm chart [\#1491](https://github.com/agones-dev/agones/issues/1491)
- Switch Node.js SDK grpc dependency to grpc-js [\#1489](https://github.com/agones-dev/agones/issues/1489)
- Update to opencensus v0.22 [\#892](https://github.com/agones-dev/agones/issues/892)
- Player Tracking: REST SDK Reference [\#1570](https://github.com/agones-dev/agones/pull/1570) ([markmandel](https://github.com/markmandel))
- Player Tracking guide, and GameServer reference. [\#1569](https://github.com/agones-dev/agones/pull/1569) ([markmandel](https://github.com/markmandel))
- Player Tracking SDK Reference [\#1564](https://github.com/agones-dev/agones/pull/1564) ([markmandel](https://github.com/markmandel))
- Fleet Aggregate Player Tracking Logic [\#1561](https://github.com/agones-dev/agones/pull/1561) ([markmandel](https://github.com/markmandel))
- Add Third Party \> Libraries and Tools section [\#1558](https://github.com/agones-dev/agones/pull/1558) ([danieloliveira079](https://github.com/danieloliveira079))
- CRD values for Aggregate Player Tracking [\#1551](https://github.com/agones-dev/agones/pull/1551) ([markmandel](https://github.com/markmandel))
- Upgrade kubectl for e2e tests [\#1550](https://github.com/agones-dev/agones/pull/1550) ([markmandel](https://github.com/markmandel))
- Use ServerCA from GameServerAllocationPolicy instead of client secret ca.crt [\#1545](https://github.com/agones-dev/agones/pull/1545) ([pooneh-m](https://github.com/pooneh-m))
- E2E Tests for GameServer Player Tracking [\#1541](https://github.com/agones-dev/agones/pull/1541) ([markmandel](https://github.com/markmandel))
- REST SDK Conformance Tests [\#1537](https://github.com/agones-dev/agones/pull/1537) ([markmandel](https://github.com/markmandel))
- Upgrade client-go and related to 1.15 [\#1532](https://github.com/agones-dev/agones/pull/1532) ([markmandel](https://github.com/markmandel))
- Player Tracking Go SDK Conformance Tests [\#1527](https://github.com/agones-dev/agones/pull/1527) ([markmandel](https://github.com/markmandel))
- Update EKS Kubernetes version to 1.15 [\#1522](https://github.com/agones-dev/agones/pull/1522) ([aLekSer](https://github.com/aLekSer))
- Helm: add ping HTTP and UDP annotations into chart [\#1520](https://github.com/agones-dev/agones/pull/1520) ([aLekSer](https://github.com/aLekSer))
- PlayerConnect/Disconnect & related Go SDK functions [\#1519](https://github.com/agones-dev/agones/pull/1519) ([markmandel](https://github.com/markmandel))
- SDKServer Player Tracking implementation [\#1507](https://github.com/agones-dev/agones/pull/1507) ([markmandel](https://github.com/markmandel))
- Add PlayerTracking IDs to SDK Convert function [\#1498](https://github.com/agones-dev/agones/pull/1498) ([markmandel](https://github.com/markmandel))
- Add Event to Player Capacity update [\#1497](https://github.com/agones-dev/agones/pull/1497) ([markmandel](https://github.com/markmandel))
- Implementation of Local SDK Server Player Tracking [\#1496](https://github.com/agones-dev/agones/pull/1496) ([markmandel](https://github.com/markmandel))
- Code Gen and Stubs for updated Player Tracking [\#1493](https://github.com/agones-dev/agones/pull/1493) ([markmandel](https://github.com/markmandel))
- Terraform: update GKE cluster version, use locals and `lookup` to set default values [\#1482](https://github.com/agones-dev/agones/pull/1482) ([aLekSer](https://github.com/aLekSer))
- Changes in proto for Player Tracking design update [\#1481](https://github.com/agones-dev/agones/pull/1481) ([markmandel](https://github.com/markmandel))
- GameServer CRD with Updated Player Tracking [\#1476](https://github.com/agones-dev/agones/pull/1476) ([markmandel](https://github.com/markmandel))
- Added a pull request template [\#1471](https://github.com/agones-dev/agones/pull/1471) ([akremsa](https://github.com/akremsa))
- Update the agones-allocator doc to recommend using cert-manager [\#1459](https://github.com/agones-dev/agones/pull/1459) ([pooneh-m](https://github.com/pooneh-m))
- Add a simple helm test [\#1449](https://github.com/agones-dev/agones/pull/1449) ([aLekSer](https://github.com/aLekSer))
- Pass FEATURE\_GATES flag to e2e tests [\#1445](https://github.com/agones-dev/agones/pull/1445) ([akremsa](https://github.com/akremsa))
- Add validation for CPU and Memory Resources for GameServers, Fleets and GameServerSets [\#1423](https://github.com/agones-dev/agones/pull/1423) ([aLekSer](https://github.com/aLekSer))

**Fixed bugs:**

- Flaky: TestGameServerReserve [\#1543](https://github.com/agones-dev/agones/issues/1543)
- SDK Server ignores custom GameServer configuration file in local mode [\#1508](https://github.com/agones-dev/agones/issues/1508)
- Helm delete doesn't support tolerations/affinities [\#1504](https://github.com/agones-dev/agones/issues/1504)
- Node.js minimist CVE [\#1490](https://github.com/agones-dev/agones/issues/1490)
- Flaky: SDK conformance tests [\#1452](https://github.com/agones-dev/agones/issues/1452)
- agones-allocator couldn't be connected via a C++ gRPC client [\#1421](https://github.com/agones-dev/agones/issues/1421)
- Flaky: TestUnhealthyGameServersWithoutFreePorts  [\#1376](https://github.com/agones-dev/agones/issues/1376)
- Metrics: Export to Stackdriver is not working [\#1330](https://github.com/agones-dev/agones/issues/1330)
- SDK package should be versioned [\#1043](https://github.com/agones-dev/agones/issues/1043)
- CPU/MEMORY leak in agones controller container [\#414](https://github.com/agones-dev/agones/issues/414)
- Site: Fix publish issue with date update [\#1568](https://github.com/agones-dev/agones/pull/1568) ([markmandel](https://github.com/markmandel))
- Flaky: TestGameServerReserve [\#1565](https://github.com/agones-dev/agones/pull/1565) ([markmandel](https://github.com/markmandel))
- Faq links point to wrong place [\#1549](https://github.com/agones-dev/agones/pull/1549) ([markmandel](https://github.com/markmandel))
- fixed Agones.Build.cs for \#1303 [\#1544](https://github.com/agones-dev/agones/pull/1544) ([dotcom](https://github.com/dotcom))
- Fix broken Fuzz Roundtrip tests in 1.15 [\#1530](https://github.com/agones-dev/agones/pull/1530) ([aLekSer](https://github.com/aLekSer))
- Fix allocator service tls auth for C\# client and add a C\# sample [\#1514](https://github.com/agones-dev/agones/pull/1514) ([pooneh-m](https://github.com/pooneh-m))
- Unity SDK: Fix SpecHealth parsing [\#1510](https://github.com/agones-dev/agones/pull/1510) ([cadfoot](https://github.com/cadfoot))
- Local SDK wasn't loading referenced file [\#1509](https://github.com/agones-dev/agones/pull/1509) ([markmandel](https://github.com/markmandel))
- Be able to run individual e2e tests in Intellij [\#1506](https://github.com/agones-dev/agones/pull/1506) ([markmandel](https://github.com/markmandel))
- Fix for flaky e2e: TestUnhealthyGameServersWithoutFreePorts [\#1480](https://github.com/agones-dev/agones/pull/1480) ([akremsa](https://github.com/akremsa))
- Monitoring: fix error on Stackdriver exporter [\#1479](https://github.com/agones-dev/agones/pull/1479) ([aLekSer](https://github.com/aLekSer))

**Closed issues:**

- Release v1.5.0 [\#1472](https://github.com/agones-dev/agones/issues/1472)
- Proposal: Change K8s version upgrade timing to be more flexible [\#1435](https://github.com/agones-dev/agones/issues/1435)
- Create a pull request template [\#608](https://github.com/agones-dev/agones/issues/608)

**Merged pull requests:**

- Release 1.6.0-rc [\#1574](https://github.com/agones-dev/agones/pull/1574) ([markmandel](https://github.com/markmandel))
- Fix Local SDK nil Players with test [\#1572](https://github.com/agones-dev/agones/pull/1572) ([aLekSer](https://github.com/aLekSer))
- Fixed a typo sercerCA -\> serverCa [\#1567](https://github.com/agones-dev/agones/pull/1567) ([pooneh-m](https://github.com/pooneh-m))
- Player Tracking Proto: Players =\> players [\#1560](https://github.com/agones-dev/agones/pull/1560) ([markmandel](https://github.com/markmandel))
- Player Tracking Proto: IDs =\> ids [\#1559](https://github.com/agones-dev/agones/pull/1559) ([markmandel](https://github.com/markmandel))
- Terraform: update AKS version [\#1556](https://github.com/agones-dev/agones/pull/1556) ([aLekSer](https://github.com/aLekSer))
- Website: update documents to use Kubernetes 1.15 [\#1555](https://github.com/agones-dev/agones/pull/1555) ([aLekSer](https://github.com/aLekSer))
- Update documentation links in examples and website pages [\#1554](https://github.com/agones-dev/agones/pull/1554) ([aLekSer](https://github.com/aLekSer))
- Player Tracking: Json "IDs" =\> "ids" [\#1552](https://github.com/agones-dev/agones/pull/1552) ([markmandel](https://github.com/markmandel))
- Fix small typo in comments [\#1548](https://github.com/agones-dev/agones/pull/1548) ([aLekSer](https://github.com/aLekSer))
- Fix: SDK conformance test. Update Rust version to fix cargo build [\#1542](https://github.com/agones-dev/agones/pull/1542) ([aLekSer](https://github.com/aLekSer))
- Flaky: TestControllerSyncGameServerCreatingState [\#1533](https://github.com/agones-dev/agones/pull/1533) ([markmandel](https://github.com/markmandel))
- Fixed a small typo spotted while reading documentation. [\#1528](https://github.com/agones-dev/agones/pull/1528) ([EricFortin](https://github.com/EricFortin))
- Small typo in test [\#1526](https://github.com/agones-dev/agones/pull/1526) ([aLekSer](https://github.com/aLekSer))
- Fix typo in workerqueue [\#1525](https://github.com/agones-dev/agones/pull/1525) ([markmandel](https://github.com/markmandel))
- Add tolerations for delete hook script [\#1521](https://github.com/agones-dev/agones/pull/1521) ([aLekSer](https://github.com/aLekSer))
- Cleanup Feature Gate Errors [\#1518](https://github.com/agones-dev/agones/pull/1518) ([markmandel](https://github.com/markmandel))
- Improved fleetautoscalers - controller unit tests [\#1515](https://github.com/agones-dev/agones/pull/1515) ([akremsa](https://github.com/akremsa))
- factored metrics logic out of the allocator main file [\#1512](https://github.com/agones-dev/agones/pull/1512) ([pooneh-m](https://github.com/pooneh-m))
- fix a typo in README.md [\#1505](https://github.com/agones-dev/agones/pull/1505) ([robinbraemer](https://github.com/robinbraemer))
- Added a couple of tests to gameserverallocationpolicy [\#1503](https://github.com/agones-dev/agones/pull/1503) ([akremsa](https://github.com/akremsa))
- FeatureFlag:PlayerTesting ➡ FeatureFlag:PlayerTracking [\#1502](https://github.com/agones-dev/agones/pull/1502) ([markmandel](https://github.com/markmandel))
- Fix link to client SDK documentation [\#1500](https://github.com/agones-dev/agones/pull/1500) ([ramonberrutti](https://github.com/ramonberrutti))
- Refactor err check in GS controller a bit. Add event if Pod was not created [\#1499](https://github.com/agones-dev/agones/pull/1499) ([aLekSer](https://github.com/aLekSer))
- Update testify package in gomod [\#1492](https://github.com/agones-dev/agones/pull/1492) ([akremsa](https://github.com/akremsa))
- Docs: add Grafana version explicitly [\#1487](https://github.com/agones-dev/agones/pull/1487) ([aLekSer](https://github.com/aLekSer))
- Improved gameserver unit tests [\#1485](https://github.com/agones-dev/agones/pull/1485) ([akremsa](https://github.com/akremsa))
- Allocator client tutorial: add steps for MacOS [\#1484](https://github.com/agones-dev/agones/pull/1484) ([aLekSer](https://github.com/aLekSer))
- Preparation for 1.6.0 Release. [\#1474](https://github.com/agones-dev/agones/pull/1474) ([markmandel](https://github.com/markmandel))
- Update Grafana to the  6.7 release. [\#1465](https://github.com/agones-dev/agones/pull/1465) ([cyriltovena](https://github.com/cyriltovena))
- Refactor of localsdk tests [\#1464](https://github.com/agones-dev/agones/pull/1464) ([markmandel](https://github.com/markmandel))
- SDK Conformance test - fix parallel recordRequests [\#1456](https://github.com/agones-dev/agones/pull/1456) ([aLekSer](https://github.com/aLekSer))
- Enhance Logs readability of  SDK Conformance Tests [\#1453](https://github.com/agones-dev/agones/pull/1453) ([aLekSer](https://github.com/aLekSer))
- Update to OpenCensus version 0.22.3 [\#1446](https://github.com/agones-dev/agones/pull/1446) ([aLekSer](https://github.com/aLekSer))

## [v1.5.0](https://github.com/agones-dev/agones/tree/v1.5.0) (2020-04-14)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.5.0-rc...v1.5.0)

**Implemented enhancements:**

- FAQ for Agones [\#1460](https://github.com/agones-dev/agones/pull/1460) ([markmandel](https://github.com/markmandel))

**Fixed bugs:**

- Flaky: TestGameServerWithPortsMappedToMultipleContainers [\#1450](https://github.com/agones-dev/agones/issues/1450)
- e2e image needs 1.14.10 kubectl [\#1470](https://github.com/agones-dev/agones/pull/1470) ([markmandel](https://github.com/markmandel))
- Working Node.js example gameserver.yaml [\#1469](https://github.com/agones-dev/agones/pull/1469) ([markmandel](https://github.com/markmandel))
- Fixed flaky TestGameServerWithPortsMappedToMultipleContainers [\#1458](https://github.com/agones-dev/agones/pull/1458) ([akremsa](https://github.com/akremsa))

**Closed issues:**

- Release 1.5.0-rc [\#1454](https://github.com/agones-dev/agones/issues/1454)
- Move /site to go.mod and Go 1.12/1.13 [\#1295](https://github.com/agones-dev/agones/issues/1295)

**Merged pull requests:**

- Release 1.5.0 [\#1473](https://github.com/agones-dev/agones/pull/1473) ([markmandel](https://github.com/markmandel))
- Website: A number of corrections in the docs [\#1466](https://github.com/agones-dev/agones/pull/1466) ([aLekSer](https://github.com/aLekSer))
- Website: Fix path in swagger command [\#1462](https://github.com/agones-dev/agones/pull/1462) ([aLekSer](https://github.com/aLekSer))
- Use go modules for a website and update go version [\#1457](https://github.com/agones-dev/agones/pull/1457) ([aLekSer](https://github.com/aLekSer))

## [v1.5.0-rc](https://github.com/agones-dev/agones/tree/v1.5.0-rc) (2020-04-07)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.4.0...v1.5.0-rc)

**Breaking changes:**

- Upgrade to Kubernetes 1.14 [\#1329](https://github.com/agones-dev/agones/issues/1329)

**Implemented enhancements:**

- GameServer remains "STATE:Creating" if not create serviceaccount [\#1370](https://github.com/agones-dev/agones/issues/1370)
- Site: Prioritise search results on agones.dev better [\#1327](https://github.com/agones-dev/agones/issues/1327)
- Create and document rules of thumb for log levels in agones code [\#1223](https://github.com/agones-dev/agones/issues/1223)
- Configurable Log Level for Agones controllers [\#1218](https://github.com/agones-dev/agones/issues/1218)
- Refactor Docker files for gRPC between SDK and allocation [\#1115](https://github.com/agones-dev/agones/issues/1115)
- C\# SDK [\#884](https://github.com/agones-dev/agones/issues/884)
- Feature Gates: EnableAllFeatures [\#1448](https://github.com/agones-dev/agones/pull/1448) ([markmandel](https://github.com/markmandel))
- Local implementation of Set/GetPlayerCapacity [\#1444](https://github.com/agones-dev/agones/pull/1444) ([markmandel](https://github.com/markmandel))
- Alpha GameServer attributes added to SDK [\#1440](https://github.com/agones-dev/agones/pull/1440) ([markmandel](https://github.com/markmandel))
- Added version to stress tests files [\#1433](https://github.com/agones-dev/agones/pull/1433) ([akremsa](https://github.com/akremsa))
- Terraform: Add FeatureGates into helm release [\#1431](https://github.com/agones-dev/agones/pull/1431) ([aLekSer](https://github.com/aLekSer))
- SuperTuxKart Game Server that allows AI connections [\#1424](https://github.com/agones-dev/agones/pull/1424) ([markmandel](https://github.com/markmandel))
- Fix wrong condition check for Memory limit [\#1418](https://github.com/agones-dev/agones/pull/1418) ([aLekSer](https://github.com/aLekSer))
- Applied allocation test [\#1417](https://github.com/agones-dev/agones/pull/1417) ([akremsa](https://github.com/akremsa))
- Add shutdown duration option to Node.js simple  [\#1413](https://github.com/agones-dev/agones/pull/1413) ([steven-supersolid](https://github.com/steven-supersolid))
- Add sidecar memory resources setting [\#1402](https://github.com/agones-dev/agones/pull/1402) ([suecideTech](https://github.com/suecideTech))
- Add ErrorHandling for failed to create pods because of forbidden [\#1400](https://github.com/agones-dev/agones/pull/1400) ([suecideTech](https://github.com/suecideTech))
- Alpha SDK.SetPlayerCapacity & GetPlayerCapacity [\#1399](https://github.com/agones-dev/agones/pull/1399) ([markmandel](https://github.com/markmandel))
- Add feature gate block to Make install [\#1397](https://github.com/agones-dev/agones/pull/1397) ([markmandel](https://github.com/markmandel))
- Allow ports to be added to any container in a GS pod [\#1396](https://github.com/agones-dev/agones/pull/1396) ([benclive](https://github.com/benclive))
- Adding the C\# gRPC SDK [\#1315](https://github.com/agones-dev/agones/pull/1315) ([Reousa](https://github.com/Reousa))

**Fixed bugs:**

- No validation when helm parameter `agones.image.sdk.cpuRequest` set less than `cpuLimit` [\#1419](https://github.com/agones-dev/agones/issues/1419)
- AKS labels are not supported in the Terraform provider, wrong Controller placement [\#1383](https://github.com/agones-dev/agones/issues/1383)
- sdk-server needs patch rbac on events [\#1304](https://github.com/agones-dev/agones/issues/1304)
- Flaky: TestGameServerReserve [\#1276](https://github.com/agones-dev/agones/issues/1276)
- Flaky: TestLocalSDKServerPlayerCapacity [\#1451](https://github.com/agones-dev/agones/pull/1451) ([markmandel](https://github.com/markmandel))
- Revert local sdk logging from Debug-\>Info [\#1443](https://github.com/agones-dev/agones/pull/1443) ([markmandel](https://github.com/markmandel))
- Retry logic in htmltest [\#1429](https://github.com/agones-dev/agones/pull/1429) ([markmandel](https://github.com/markmandel))
- Terraform fix AKS Node Pool Labels [\#1420](https://github.com/agones-dev/agones/pull/1420) ([aLekSer](https://github.com/aLekSer))
- Update Node.js dependencies and fix annoyances [\#1415](https://github.com/agones-dev/agones/pull/1415) ([steven-supersolid](https://github.com/steven-supersolid))
- Flaky: TestGameServerReserve [\#1414](https://github.com/agones-dev/agones/pull/1414) ([markmandel](https://github.com/markmandel))
- Fixed permission of sidecar serviceaccount [\#1408](https://github.com/agones-dev/agones/pull/1408) ([suecideTech](https://github.com/suecideTech))
- SdkServer: updateState does not do a DeepClone\(\) [\#1398](https://github.com/agones-dev/agones/pull/1398) ([markmandel](https://github.com/markmandel))
- Fix SDK conformance GRPC gateway test [\#1390](https://github.com/agones-dev/agones/pull/1390) ([aLekSer](https://github.com/aLekSer))
- Extra Debugging for TestGameServerReserve [\#1334](https://github.com/agones-dev/agones/pull/1334) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- Release 1.4.0 [\#1387](https://github.com/agones-dev/agones/issues/1387)
- C++ Game Server Client SDK documentation formatting [\#1379](https://github.com/agones-dev/agones/issues/1379)
- \[Deprecation\] Upgrade Build Node.js version to 12  [\#1272](https://github.com/agones-dev/agones/issues/1272)

**Merged pull requests:**

- Release 1.5.0-rc [\#1455](https://github.com/agones-dev/agones/pull/1455) ([markmandel](https://github.com/markmandel))
- Fix wrong function usage [\#1442](https://github.com/agones-dev/agones/pull/1442) ([aLekSer](https://github.com/aLekSer))
- Index out of range error in e2e TestFleetGSSpecValidation [\#1439](https://github.com/agones-dev/agones/pull/1439) ([akremsa](https://github.com/akremsa))
- Added log\_level parameter to Terraform deployment [\#1438](https://github.com/agones-dev/agones/pull/1438) ([akremsa](https://github.com/akremsa))
- update local.md [\#1437](https://github.com/agones-dev/agones/pull/1437) ([DemonsLu](https://github.com/DemonsLu))
- Add go.sum to SuperTuxKart example. [\#1434](https://github.com/agones-dev/agones/pull/1434) ([markmandel](https://github.com/markmandel))
- Terraform docs about terraform destroy  [\#1425](https://github.com/agones-dev/agones/pull/1425) ([aLekSer](https://github.com/aLekSer))
- Super Tux Kart -\> SuperTuxKart, and additional wording fixes [\#1410](https://github.com/agones-dev/agones/pull/1410) ([qwertychouskie](https://github.com/qwertychouskie))
- Docs: Add description for Kubernetes client metrics [\#1409](https://github.com/agones-dev/agones/pull/1409) ([aLekSer](https://github.com/aLekSer))
- Add missing allocator affinity and tolerations to helm docs [\#1406](https://github.com/agones-dev/agones/pull/1406) ([bburghaus](https://github.com/bburghaus))
- Upgrade Build Node.js version to 12 [\#1405](https://github.com/agones-dev/agones/pull/1405) ([akremsa](https://github.com/akremsa))
- update client-go for kubernetes-1.14.10 [\#1404](https://github.com/agones-dev/agones/pull/1404) ([heartrobotninja](https://github.com/heartrobotninja))
- Site search only agones.dev, not previous version domains. [\#1395](https://github.com/agones-dev/agones/pull/1395) ([markmandel](https://github.com/markmandel))
- Switch godoc.org links to pkg.go.dev [\#1394](https://github.com/agones-dev/agones/pull/1394) ([markmandel](https://github.com/markmandel))
- Fixed mangled lists in C++ guide  [\#1393](https://github.com/agones-dev/agones/pull/1393) ([akremsa](https://github.com/akremsa))
- Preparation for 1.5.0 Release [\#1391](https://github.com/agones-dev/agones/pull/1391) ([markmandel](https://github.com/markmandel))
- Update terraform EKS module to 1.14 version [\#1386](https://github.com/agones-dev/agones/pull/1386) ([aLekSer](https://github.com/aLekSer))
- AKS use supported version of Kubernetes 1.14 [\#1385](https://github.com/agones-dev/agones/pull/1385) ([aLekSer](https://github.com/aLekSer))
- Fix for AKS recent provider change [\#1380](https://github.com/agones-dev/agones/pull/1380) ([aLekSer](https://github.com/aLekSer))
- Terraform make targets: Switch from the plane structure to module terraform config [\#1375](https://github.com/agones-dev/agones/pull/1375) ([aLekSer](https://github.com/aLekSer))
- Separate e2e tests in build Makefile [\#1371](https://github.com/agones-dev/agones/pull/1371) ([drichardson](https://github.com/drichardson))
- UE4 readme to communicate development information [\#1360](https://github.com/agones-dev/agones/pull/1360) ([drichardson](https://github.com/drichardson))
- Updated log levels in pkg [\#1359](https://github.com/agones-dev/agones/pull/1359) ([akremsa](https://github.com/akremsa))

## [v1.4.0](https://github.com/agones-dev/agones/tree/v1.4.0) (2020-03-04)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.4.0-rc...v1.4.0)

**Breaking changes:**

- Fix for critical GKE/GCP Terraform Bugs [\#1373](https://github.com/agones-dev/agones/pull/1373) ([markmandel](https://github.com/markmandel))
- Updated documentation for multi-cluster allocation [\#1365](https://github.com/agones-dev/agones/pull/1365) ([pooneh-m](https://github.com/pooneh-m))

**Fixed bugs:**

- Terraform: clean up duplicate tf config files. [\#1372](https://github.com/agones-dev/agones/issues/1372)
- Documentation for gRPC Based Multicluster Allocator [\#1333](https://github.com/agones-dev/agones/issues/1333)

**Closed issues:**

- Release 1.4.0-rc [\#1366](https://github.com/agones-dev/agones/issues/1366)

**Merged pull requests:**

- Release 1.4.0 [\#1388](https://github.com/agones-dev/agones/pull/1388) ([markmandel](https://github.com/markmandel))
- Removed dockerfile from build-allocation-images [\#1382](https://github.com/agones-dev/agones/pull/1382) ([akremsa](https://github.com/akremsa))
- Fix the link to cert.sh [\#1381](https://github.com/agones-dev/agones/pull/1381) ([pooneh-m](https://github.com/pooneh-m))
- Add crd-client image to release template [\#1378](https://github.com/agones-dev/agones/pull/1378) ([aLekSer](https://github.com/aLekSer))
- Fix for 404 in OpenSSL Install link [\#1374](https://github.com/agones-dev/agones/pull/1374) ([markmandel](https://github.com/markmandel))
- Main page, gameserver lifecycle page - typos [\#1369](https://github.com/agones-dev/agones/pull/1369) ([burningalchemist](https://github.com/burningalchemist))
- Documented an approach of log levels usage [\#1368](https://github.com/agones-dev/agones/pull/1368) ([akremsa](https://github.com/akremsa))

## [v1.4.0-rc](https://github.com/agones-dev/agones/tree/v1.4.0-rc) (2020-02-26)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.3.0...v1.4.0-rc)

**Breaking changes:**

- Change GameServerAllocationPolicy CRD version to stable [\#1290](https://github.com/agones-dev/agones/issues/1290)
- Update documentation \(examples and website\) to use Kubernetes 1.14 [\#1364](https://github.com/agones-dev/agones/pull/1364) ([roberthbailey](https://github.com/roberthbailey))
- Update terraform configs to use Kubernetes 1.14 [\#1342](https://github.com/agones-dev/agones/pull/1342) ([roberthbailey](https://github.com/roberthbailey))
- Update developer tooling to Kubernetes 1.14 [\#1341](https://github.com/agones-dev/agones/pull/1341) ([roberthbailey](https://github.com/roberthbailey))
- Change the GameServerAllocationPolicy version to stable. [\#1332](https://github.com/agones-dev/agones/pull/1332) ([pooneh-m](https://github.com/pooneh-m))
- Changing the allocator API to gRPC [\#1314](https://github.com/agones-dev/agones/pull/1314) ([pooneh-m](https://github.com/pooneh-m))

**Implemented enhancements:**

- Better documentation for BufferPolicy in fleetautoscaler  [\#1104](https://github.com/agones-dev/agones/issues/1104)
- Add Fuzz Tests [\#1098](https://github.com/agones-dev/agones/issues/1098)
- agones-allocator service should scale up independent to gameserverallocation extension API server [\#1018](https://github.com/agones-dev/agones/issues/1018)
- Missing documentation/example for new matchmaker support \(Allocate and Reserve\) [\#976](https://github.com/agones-dev/agones/issues/976)
- Release NPM package, and Node update [\#1356](https://github.com/agones-dev/agones/pull/1356) ([markmandel](https://github.com/markmandel))
- Unreal SDK add Allocate + Reserve and changes to the plugin settings [\#1345](https://github.com/agones-dev/agones/pull/1345) ([WVerlaek](https://github.com/WVerlaek))
- Adding SuperTuxKart to the examples page [\#1336](https://github.com/agones-dev/agones/pull/1336) ([markmandel](https://github.com/markmandel))
- CRD implementation of alpha player tracking [\#1324](https://github.com/agones-dev/agones/pull/1324) ([markmandel](https://github.com/markmandel))
- Player Tracking Proto and Go stubs [\#1312](https://github.com/agones-dev/agones/pull/1312) ([markmandel](https://github.com/markmandel))
- Add fuzz tests missing vendor changes [\#1306](https://github.com/agones-dev/agones/pull/1306) ([pooneh-m](https://github.com/pooneh-m))
- Extend Agones Unreal SDK [\#1303](https://github.com/agones-dev/agones/pull/1303) ([WVerlaek](https://github.com/WVerlaek))
- Super Tux Kart Example [\#1302](https://github.com/agones-dev/agones/pull/1302) ([markmandel](https://github.com/markmandel))
- Stubs for SDK alpha/beta/stable functionality [\#1285](https://github.com/agones-dev/agones/pull/1285) ([markmandel](https://github.com/markmandel))

**Fixed bugs:**

- UE4 plugin stores configuration data in per-user rather than per-project config [\#1351](https://github.com/agones-dev/agones/issues/1351)
- Alpha field in the stable API should follow best practices [\#1347](https://github.com/agones-dev/agones/issues/1347)
- Flaky: TestGameServerAllocationDeletionOnUnAllocate [\#1326](https://github.com/agones-dev/agones/issues/1326)
- UE4 plugin fails to compile with default UE4 4.24 project [\#1318](https://github.com/agones-dev/agones/issues/1318)
- Running go mod tidy fails with error [\#1309](https://github.com/agones-dev/agones/issues/1309)
- Gameserver failed to start in the namespace with runAsNonRoot pod security context. [\#1287](https://github.com/agones-dev/agones/issues/1287)
- Not explicitly providing a fleet replacement strategy results in it being set to blank and redeployments failing [\#1286](https://github.com/agones-dev/agones/issues/1286)
- Agones controller shut down  [\#1170](https://github.com/agones-dev/agones/issues/1170)
- Swagger: WatchGameServer\(\) definition is not generated properly in sdk.swagger.json [\#1106](https://github.com/agones-dev/agones/issues/1106)
- Gameserver is not removed when node hosting gameserver pod is shutdown [\#1102](https://github.com/agones-dev/agones/issues/1102)
- Moving cluster to a new node pool doesn't recreate all fleets [\#398](https://github.com/agones-dev/agones/issues/398)
- Fix test failure due to v1alpha becoming v1. [\#1361](https://github.com/agones-dev/agones/pull/1361) ([drichardson](https://github.com/drichardson))
- Fix UE4 plugin compilation error in AgonesHook.h. [\#1358](https://github.com/agones-dev/agones/pull/1358) ([drichardson](https://github.com/drichardson))
- Save UE4 Plugin settings to per-project config file [\#1352](https://github.com/agones-dev/agones/pull/1352) ([drichardson](https://github.com/drichardson))
- Stackdriver - fix getMonitoredResource [\#1335](https://github.com/agones-dev/agones/pull/1335) ([aLekSer](https://github.com/aLekSer))
- Flakiness: TestGameServerAllocationDeletionOnUnAllocate [\#1328](https://github.com/agones-dev/agones/pull/1328) ([markmandel](https://github.com/markmandel))
- Fix for `go mod vendor` command [\#1322](https://github.com/agones-dev/agones/pull/1322) ([aLekSer](https://github.com/aLekSer))
- Support UE4 BuildSettingsVersion.V2 [\#1319](https://github.com/agones-dev/agones/pull/1319) ([drichardson](https://github.com/drichardson))
- Fleet: Add validation for Strategy Type [\#1316](https://github.com/agones-dev/agones/pull/1316) ([aLekSer](https://github.com/aLekSer))
- Workerqueue IsConflict needed to check error Cause [\#1310](https://github.com/agones-dev/agones/pull/1310) ([markmandel](https://github.com/markmandel))
- Fix deep copy for multi-cluster allocation policy CRD [\#1308](https://github.com/agones-dev/agones/pull/1308) ([pooneh-m](https://github.com/pooneh-m))
- Use a numeric User ID for the "agones" user in the SDK sidecar [\#1293](https://github.com/agones-dev/agones/pull/1293) ([TBBle](https://github.com/TBBle))
- Fix for Pod deletion during unavailable controller [\#1279](https://github.com/agones-dev/agones/pull/1279) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- Approver access for @aLekSer [\#1354](https://github.com/agones-dev/agones/issues/1354)
- Release 1.3.0 [\#1288](https://github.com/agones-dev/agones/issues/1288)
- Vendor tool dependencies [\#695](https://github.com/agones-dev/agones/issues/695)
- Create npm package for Node.js sdk [\#679](https://github.com/agones-dev/agones/issues/679)

**Merged pull requests:**

- Release 1.4.0 Release Candidate [\#1367](https://github.com/agones-dev/agones/pull/1367) ([markmandel](https://github.com/markmandel))
- Update Feature Stage: New CRD attributes section [\#1355](https://github.com/agones-dev/agones/pull/1355) ([markmandel](https://github.com/markmandel))
- Fix Kubernetes terraform provider version [\#1353](https://github.com/agones-dev/agones/pull/1353) ([aLekSer](https://github.com/aLekSer))
- Update the kind make targets to work with kind 0.6.0 and later. [\#1349](https://github.com/agones-dev/agones/pull/1349) ([roberthbailey](https://github.com/roberthbailey))
- Implement Alpha field best practices [\#1348](https://github.com/agones-dev/agones/pull/1348) ([markmandel](https://github.com/markmandel))
- Add more details on Allocate\(\) behaviour [\#1346](https://github.com/agones-dev/agones/pull/1346) ([aLekSer](https://github.com/aLekSer))
- Update images with latest everything \#1261 [\#1344](https://github.com/agones-dev/agones/pull/1344) ([akremsa](https://github.com/akremsa))
- Add missing CABundle field for FleetAutoScaler CRD [\#1339](https://github.com/agones-dev/agones/pull/1339) ([aLekSer](https://github.com/aLekSer))
- Unity SDK - Initialize in Awake\(\) [\#1338](https://github.com/agones-dev/agones/pull/1338) ([mollstam](https://github.com/mollstam))
- Use official github-changelog-generator in release [\#1337](https://github.com/agones-dev/agones/pull/1337) ([markmandel](https://github.com/markmandel))
- Remove the provider blocks from the gke submodules [\#1323](https://github.com/agones-dev/agones/pull/1323) ([chrisst](https://github.com/chrisst))
- Adding missing dependency to a vendor\_fixes [\#1321](https://github.com/agones-dev/agones/pull/1321) ([aLekSer](https://github.com/aLekSer))
- Removing Interpolation-only expressions [\#1317](https://github.com/agones-dev/agones/pull/1317) ([chrisst](https://github.com/chrisst))
- Move all "Synchronising" logs to debug [\#1311](https://github.com/agones-dev/agones/pull/1311) ([markmandel](https://github.com/markmandel))
- Do not remove replica field if set to zero [\#1305](https://github.com/agones-dev/agones/pull/1305) ([pooneh-m](https://github.com/pooneh-m))
- Fix broken agones.dev link [\#1297](https://github.com/agones-dev/agones/pull/1297) ([moesy](https://github.com/moesy))
- Adding a "Major Feature" to overview page [\#1294](https://github.com/agones-dev/agones/pull/1294) ([markmandel](https://github.com/markmandel))
- Show GameServer name in TestGameServerReserve flakiness fail [\#1292](https://github.com/agones-dev/agones/pull/1292) ([markmandel](https://github.com/markmandel))
- Preparation for 1.4.0 Sprint [\#1291](https://github.com/agones-dev/agones/pull/1291) ([markmandel](https://github.com/markmandel))
- CPP SDK example code: join threads [\#1283](https://github.com/agones-dev/agones/pull/1283) ([aLekSer](https://github.com/aLekSer))
- Conformance test for CPP SDK [\#1282](https://github.com/agones-dev/agones/pull/1282) ([aLekSer](https://github.com/aLekSer))

## [v1.3.0](https://github.com/agones-dev/agones/tree/v1.3.0) (2020-01-21)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.3.0-rc...v1.3.0)

**Implemented enhancements:**

- Site: Add video & podcasts to third party section [\#1281](https://github.com/agones-dev/agones/pull/1281) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- Release 1.3.0-rc [\#1273](https://github.com/agones-dev/agones/issues/1273)

**Merged pull requests:**

- Release 1.3.0 [\#1289](https://github.com/agones-dev/agones/pull/1289) ([markmandel](https://github.com/markmandel))
- Nodejs docs: require for npm should switch on 1.3.0 [\#1284](https://github.com/agones-dev/agones/pull/1284) ([markmandel](https://github.com/markmandel))
- Need to change to https url on swagger-codegen download [\#1278](https://github.com/agones-dev/agones/pull/1278) ([markmandel](https://github.com/markmandel))
- Release publishing of NPM module [\#1277](https://github.com/agones-dev/agones/pull/1277) ([markmandel](https://github.com/markmandel))
- Fix list formatting on the terraform install page. [\#1274](https://github.com/agones-dev/agones/pull/1274) ([roberthbailey](https://github.com/roberthbailey))

## [v1.3.0-rc](https://github.com/agones-dev/agones/tree/v1.3.0-rc) (2020-01-14)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.2.0...v1.3.0-rc)

**Breaking changes:**

- Node packaging [\#1264](https://github.com/agones-dev/agones/pull/1264) ([rorygarand](https://github.com/rorygarand))
- Update GRPC to v1.20.1 [\#1215](https://github.com/agones-dev/agones/pull/1215) ([markmandel](https://github.com/markmandel))

**Implemented enhancements:**

- Terraform support for EKS [\#966](https://github.com/agones-dev/agones/issues/966)
- Added Community Meetings to the community pages [\#1271](https://github.com/agones-dev/agones/pull/1271) ([markmandel](https://github.com/markmandel))
- Fuzz Roundtrip test for v1 Agones schemas [\#1269](https://github.com/agones-dev/agones/pull/1269) ([aLekSer](https://github.com/aLekSer))
- Add Annotations validation for Template ObjectMeta [\#1266](https://github.com/agones-dev/agones/pull/1266) ([aLekSer](https://github.com/aLekSer))
- Add validation for Labels on Fleet and GS Template [\#1257](https://github.com/agones-dev/agones/pull/1257) ([aLekSer](https://github.com/aLekSer))
- Feature Gate implementation [\#1256](https://github.com/agones-dev/agones/pull/1256) ([markmandel](https://github.com/markmandel))
- Add Embark logo to Agones site [\#1237](https://github.com/agones-dev/agones/pull/1237) ([luna-duclos](https://github.com/luna-duclos))
- Unity SDK - Watch GameServer Functionality [\#1234](https://github.com/agones-dev/agones/pull/1234) ([markmandel](https://github.com/markmandel))
- Better error message for Health Check Failure [\#1222](https://github.com/agones-dev/agones/pull/1222) ([markmandel](https://github.com/markmandel))
- Configurable Log Level for Agones Controller [\#1220](https://github.com/agones-dev/agones/pull/1220) ([aLekSer](https://github.com/aLekSer))
- Add Go Client example which could create GS [\#1213](https://github.com/agones-dev/agones/pull/1213) ([aLekSer](https://github.com/aLekSer))
- Automate confirming example images are on gcr.io [\#1207](https://github.com/agones-dev/agones/pull/1207) ([markmandel](https://github.com/markmandel))
- improve stackdriver metric type [\#1132](https://github.com/agones-dev/agones/pull/1132) ([cyriltovena](https://github.com/cyriltovena))
- Initial version of EKS terraform config [\#986](https://github.com/agones-dev/agones/pull/986) ([aLekSer](https://github.com/aLekSer))

**Fixed bugs:**

- GKE: Preemptible node pools make game server unreachable [\#1245](https://github.com/agones-dev/agones/issues/1245)
- Fleet/GameServerSet Validation: Able to create a Fleet with label and annotations length \> 63 symbols [\#1244](https://github.com/agones-dev/agones/issues/1244)
- Health Checking Documentation bug - doesn't restart before Ready [\#1229](https://github.com/agones-dev/agones/issues/1229)
- TestMultiClusterAllocationOnLocalCluster is flakey [\#1114](https://github.com/agones-dev/agones/issues/1114)
- Fixup Helm Configuration table [\#1255](https://github.com/agones-dev/agones/pull/1255) ([markmandel](https://github.com/markmandel))
- Handling of PVM shutdown/maintenance events [\#1254](https://github.com/agones-dev/agones/pull/1254) ([markmandel](https://github.com/markmandel))
- Fix cleanup Agones resources script [\#1240](https://github.com/agones-dev/agones/pull/1240) ([aLekSer](https://github.com/aLekSer))
- Fix documentation for multi-cluster allocation [\#1235](https://github.com/agones-dev/agones/pull/1235) ([pooneh-m](https://github.com/pooneh-m))
- Health Checking: Fix doc errors and expand [\#1233](https://github.com/agones-dev/agones/pull/1233) ([markmandel](https://github.com/markmandel))

**Security fixes:**

- Update Alpine image to 3.11 [\#1253](https://github.com/agones-dev/agones/pull/1253) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- typo in README.md [\#1259](https://github.com/agones-dev/agones/issues/1259)
- Release 1.2.0 [\#1225](https://github.com/agones-dev/agones/issues/1225)
- Updates GRPC to v1.20.1 [\#1214](https://github.com/agones-dev/agones/issues/1214)

**Merged pull requests:**

- Release 1.3.0-rc [\#1275](https://github.com/agones-dev/agones/pull/1275) ([markmandel](https://github.com/markmandel))
- Fix broken link to Helm installation [\#1270](https://github.com/agones-dev/agones/pull/1270) ([mdanzinger](https://github.com/mdanzinger))
- Update golangci-lint, add more linters [\#1267](https://github.com/agones-dev/agones/pull/1267) ([aLekSer](https://github.com/aLekSer))
- Fix a minor typo in the unity example's README [\#1265](https://github.com/agones-dev/agones/pull/1265) ([roberthbailey](https://github.com/roberthbailey))
- Update e2e Kubectl to 1.13.12 [\#1252](https://github.com/agones-dev/agones/pull/1252) ([markmandel](https://github.com/markmandel))
- Update to Go 1.13 [\#1251](https://github.com/agones-dev/agones/pull/1251) ([markmandel](https://github.com/markmandel))
- Events: Move logging to Debug [\#1250](https://github.com/agones-dev/agones/pull/1250) ([markmandel](https://github.com/markmandel))
- Webhooks: Move "running" logging to Debug [\#1249](https://github.com/agones-dev/agones/pull/1249) ([markmandel](https://github.com/markmandel))
- Workerqueue: Enqueuing & processing logs to Debug [\#1248](https://github.com/agones-dev/agones/pull/1248) ([markmandel](https://github.com/markmandel))
- Adding retry on GameServerAllocations.Create for multi-cluster testing [\#1243](https://github.com/agones-dev/agones/pull/1243) ([pooneh-m](https://github.com/pooneh-m))
- Half the width of the Embark Logo [\#1242](https://github.com/agones-dev/agones/pull/1242) ([luna-duclos](https://github.com/luna-duclos))
- Remove syncGameServerSet logging. [\#1241](https://github.com/agones-dev/agones/pull/1241) ([markmandel](https://github.com/markmandel))
- Changed the LabelSelector reference of allocator gRPC api [\#1236](https://github.com/agones-dev/agones/pull/1236) ([pooneh-m](https://github.com/pooneh-m))
- Move conflict messaged to Debug Logging [\#1232](https://github.com/agones-dev/agones/pull/1232) ([markmandel](https://github.com/markmandel))
- Preparation for 1.3.0 [\#1231](https://github.com/agones-dev/agones/pull/1231) ([markmandel](https://github.com/markmandel))
- Docs Fleet add section about RollingUpdate fields [\#1228](https://github.com/agones-dev/agones/pull/1228) ([aLekSer](https://github.com/aLekSer))
- Split installation guide into separate sections [\#1211](https://github.com/agones-dev/agones/pull/1211) ([markmandel](https://github.com/markmandel))

## [v1.2.0](https://github.com/agones-dev/agones/tree/v1.2.0) (2019-12-11)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.2.0-rc...v1.2.0)

**Implemented enhancements:**

- Document the default ports used by the sdkserver sidecar on the website [\#1210](https://github.com/agones-dev/agones/pull/1210) ([roberthbailey](https://github.com/roberthbailey))

**Fixed bugs:**

- `agones.allocator.http.expose` and `agones.allocator.http.response` are documented but not consumed by Helm [\#1216](https://github.com/agones-dev/agones/issues/1216)
- Revert: Make it possible to create a Fleet with 0 replicas [\#1226](https://github.com/agones-dev/agones/pull/1226) ([markmandel](https://github.com/markmandel))
- Fix documentation for allocator helm args [\#1221](https://github.com/agones-dev/agones/pull/1221) ([pooneh-m](https://github.com/pooneh-m))
- Setting Unreal plugin version to 3 [\#1209](https://github.com/agones-dev/agones/pull/1209) ([domgreen](https://github.com/domgreen))

**Closed issues:**

- Release 1.2.0-rc [\#1203](https://github.com/agones-dev/agones/issues/1203)

**Merged pull requests:**

- Release 1.2.0 [\#1230](https://github.com/agones-dev/agones/pull/1230) ([markmandel](https://github.com/markmandel))
- Docs: allocator service should have save-config [\#1224](https://github.com/agones-dev/agones/pull/1224) ([aLekSer](https://github.com/aLekSer))
- Add missing license headers [\#1219](https://github.com/agones-dev/agones/pull/1219) ([aLekSer](https://github.com/aLekSer))
- Wrong date on 1.2.0-rc blog post [\#1208](https://github.com/agones-dev/agones/pull/1208) ([markmandel](https://github.com/markmandel))
- Release 1.2.0-rc [\#1206](https://github.com/agones-dev/agones/pull/1206) ([markmandel](https://github.com/markmandel))

## [v1.2.0-rc](https://github.com/agones-dev/agones/tree/v1.2.0-rc) (2019-12-04)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.1.0...v1.2.0-rc)

**Breaking changes:**

- Update client-go to kubernetes-1.13.12 [\#1189](https://github.com/agones-dev/agones/pull/1189) ([heartrobotninja](https://github.com/heartrobotninja))
- Update the default gRPC and HTTP ports for the sdkserver for the 1.2.0 release [\#1180](https://github.com/agones-dev/agones/pull/1180) ([roberthbailey](https://github.com/roberthbailey))
- Update terraform configs to use Kubernetes 1.13 [\#1179](https://github.com/agones-dev/agones/pull/1179) ([roberthbailey](https://github.com/roberthbailey))
- Update documentation for the 1.2 release to use Kubernetes 1.13. [\#1178](https://github.com/agones-dev/agones/pull/1178) ([roberthbailey](https://github.com/roberthbailey))
- Update minikube and KIND developer tooling to Kubernetes 1.13 [\#1177](https://github.com/agones-dev/agones/pull/1177) ([roberthbailey](https://github.com/roberthbailey))
- Update the prow cluster to use Kubernetes 1.13 [\#1176](https://github.com/agones-dev/agones/pull/1176) ([roberthbailey](https://github.com/roberthbailey))
- Update GKE development tooling to Kubernetes 1.13 [\#1175](https://github.com/agones-dev/agones/pull/1175) ([roberthbailey](https://github.com/roberthbailey))

**Implemented enhancements:**

- Upgrade to Kubernetes 1.13 [\#1044](https://github.com/agones-dev/agones/issues/1044)
- Local SDK Server should update internal GameServer on Ready\(\), Allocate\(\) etc. [\#958](https://github.com/agones-dev/agones/issues/958)
- Error checking of attributes on feature shortcode [\#1205](https://github.com/agones-dev/agones/pull/1205) ([markmandel](https://github.com/markmandel))
- Implement Reserve for Unity [\#1193](https://github.com/agones-dev/agones/pull/1193) ([markmandel](https://github.com/markmandel))
- Add ImagePullSecrets for Allocator and Ping \(helm\) [\#1190](https://github.com/agones-dev/agones/pull/1190) ([aLekSer](https://github.com/aLekSer))
- Implementation of Unity SDK.Connect\(\) [\#1181](https://github.com/agones-dev/agones/pull/1181) ([markmandel](https://github.com/markmandel))
- Implementation of GameServer\(\) for Unity [\#1169](https://github.com/agones-dev/agones/pull/1169) ([markmandel](https://github.com/markmandel))
- Add Spec and Health data to default local sdk GameServer [\#1166](https://github.com/agones-dev/agones/pull/1166) ([markmandel](https://github.com/markmandel))
- Refresh client CA certificate if changed [\#1145](https://github.com/agones-dev/agones/pull/1145) ([pooneh-m](https://github.com/pooneh-m))

**Fixed bugs:**

- Undocumented dependencies for make run-sdk-conformance-local [\#1199](https://github.com/agones-dev/agones/issues/1199)
- Documentation: FleetAutoScaler BufferSize could contain both Ready and Reserved GameServers [\#1195](https://github.com/agones-dev/agones/issues/1195)
- \[helm\] Missing imagePullSecrets Option in agones-ping Deployment [\#1185](https://github.com/agones-dev/agones/issues/1185)
- Agones fails to start the pod after update cpu limits to 1000m [\#1184](https://github.com/agones-dev/agones/issues/1184)
- RollingUpdate shuts down all servers at once [\#1156](https://github.com/agones-dev/agones/issues/1156)
- CI: Build cache cpp-sdk-build contains unnecessary archives [\#1136](https://github.com/agones-dev/agones/issues/1136)
- Segfault with C++ SDK [\#999](https://github.com/agones-dev/agones/issues/999)
- Game server container crash before Ready, should restart, not move to Unhealthy [\#956](https://github.com/agones-dev/agones/issues/956)
- Sidecar occasionally fails to start up [\#851](https://github.com/agones-dev/agones/issues/851)
- Fleet Autoscaler spawn extra gs [\#443](https://github.com/agones-dev/agones/issues/443)
- Whoops - spelling mistake in feature tag. [\#1204](https://github.com/agones-dev/agones/pull/1204) ([markmandel](https://github.com/markmandel))
- Fix infinite creation of GameServerSets when 1000m CPU limit was used [\#1188](https://github.com/agones-dev/agones/pull/1188) ([aLekSer](https://github.com/aLekSer))
- Flaky: TestGameServerRestartBeforeReadyCrash [\#1174](https://github.com/agones-dev/agones/pull/1174) ([markmandel](https://github.com/markmandel))
- Allow fleet with empty Replicas [\#1168](https://github.com/agones-dev/agones/pull/1168) ([aLekSer](https://github.com/aLekSer))
- GameServer container restart before Ready, move to Unhealthy state After \(v2\) [\#1099](https://github.com/agones-dev/agones/pull/1099) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- Flaky: TestGameServerRestartBeforeReadyCrash [\#1173](https://github.com/agones-dev/agones/issues/1173)
- Release 1.1.0 [\#1160](https://github.com/agones-dev/agones/issues/1160)
- Quickstart: Create a Fleet Autoscaler with Webhook Policy [\#1151](https://github.com/agones-dev/agones/issues/1151)
- Move terraform modules from ./build to ./install [\#1150](https://github.com/agones-dev/agones/issues/1150)
- Speed up e2e tests [\#511](https://github.com/agones-dev/agones/issues/511)

**Merged pull requests:**

- Fix sdk-server image make target [\#1200](https://github.com/agones-dev/agones/pull/1200) ([aLekSer](https://github.com/aLekSer))
- FleetAutoScaler add reserved into consideration [\#1198](https://github.com/agones-dev/agones/pull/1198) ([aLekSer](https://github.com/aLekSer))
- Flaky: TestGameServerUnhealthyAfterReadyCrash [\#1192](https://github.com/agones-dev/agones/pull/1192) ([markmandel](https://github.com/markmandel))
- Fix MD files in the repo [\#1191](https://github.com/agones-dev/agones/pull/1191) ([aLekSer](https://github.com/aLekSer))
- Point Helm Upgrade link to v2 documentation [\#1187](https://github.com/agones-dev/agones/pull/1187) ([markmandel](https://github.com/markmandel))
- Change the "Read more ..." links to better text on the home page. [\#1186](https://github.com/agones-dev/agones/pull/1186) ([roberthbailey](https://github.com/roberthbailey))
- Extra debugging for TestGameServerRestartBeforeReadyCrash [\#1172](https://github.com/agones-dev/agones/pull/1172) ([markmandel](https://github.com/markmandel))
- Mitigate panic when no ports allocated in E2E tests [\#1171](https://github.com/agones-dev/agones/pull/1171) ([aLekSer](https://github.com/aLekSer))
- Move terraform modules from ./build to ./install [\#1167](https://github.com/agones-dev/agones/pull/1167) ([amitlevy21](https://github.com/amitlevy21))
- Docs: Fix using GOPATH in the guides, added shortcode ghrelease [\#1165](https://github.com/agones-dev/agones/pull/1165) ([aLekSer](https://github.com/aLekSer))
- Add top level notes on functionalitity on Unity & Unreal docs [\#1164](https://github.com/agones-dev/agones/pull/1164) ([markmandel](https://github.com/markmandel))
- Preperation for 1.2.0 sprint [\#1162](https://github.com/agones-dev/agones/pull/1162) ([markmandel](https://github.com/markmandel))
- Disable Auto Upgrade for Deployment Manager [\#1143](https://github.com/agones-dev/agones/pull/1143) ([aLekSer](https://github.com/aLekSer))
- Fix CPP SDK archive path - created separate one [\#1142](https://github.com/agones-dev/agones/pull/1142) ([aLekSer](https://github.com/aLekSer))
- Move Googlers no longer actively reviewing code to emeritus owners [\#1116](https://github.com/agones-dev/agones/pull/1116) ([roberthbailey](https://github.com/roberthbailey))

## [v1.1.0](https://github.com/agones-dev/agones/tree/v1.1.0) (2019-10-29)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.1.0-rc...v1.1.0)

**Implemented enhancements:**

- Document agones-allocator and multi-cluster allocation [\#1155](https://github.com/agones-dev/agones/pull/1155) ([pooneh-m](https://github.com/pooneh-m))
- Add SDK.GameServer\(\) to Matchmaking Registration Diagram [\#1149](https://github.com/agones-dev/agones/pull/1149) ([markmandel](https://github.com/markmandel))
- Update the instructions for installing a GKE cluster with terraform to disable automatic node upgrades [\#1141](https://github.com/agones-dev/agones/pull/1141) ([roberthbailey](https://github.com/roberthbailey))

**Fixed bugs:**

- TestFleetRollingUpdate/Use\_fleet\_Patch\_false\_10% test is flaky [\#1049](https://github.com/agones-dev/agones/issues/1049)
- Broke Allocator Status in Grafana Dashboard [\#1158](https://github.com/agones-dev/agones/pull/1158) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- Release 1.1.0-rc [\#1133](https://github.com/agones-dev/agones/issues/1133)
- Rebuild example images for the 1.1 release [\#1126](https://github.com/agones-dev/agones/issues/1126)
- Allow issues to be assigned to collaborators [\#1120](https://github.com/agones-dev/agones/issues/1120)

**Merged pull requests:**

- Release 1.1.0 [\#1161](https://github.com/agones-dev/agones/pull/1161) ([markmandel](https://github.com/markmandel))
- Update "Local Development" documents [\#1159](https://github.com/agones-dev/agones/pull/1159) ([aLekSer](https://github.com/aLekSer))
- Cloud Build: Add --output-sync=target to run-sdk-conformance-tests [\#1157](https://github.com/agones-dev/agones/pull/1157) ([markmandel](https://github.com/markmandel))
- Update the images for all examples to use the 1.1 SDKs [\#1153](https://github.com/agones-dev/agones/pull/1153) ([roberthbailey](https://github.com/roberthbailey))
- Replace `\>` quoted text with alert shortcodes [\#1152](https://github.com/agones-dev/agones/pull/1152) ([markmandel](https://github.com/markmandel))
- Fixed a typo [\#1147](https://github.com/agones-dev/agones/pull/1147) ([pooneh-m](https://github.com/pooneh-m))
- Update the instructions for manually creating a GKE cluster to disable automatic node upgrades [\#1140](https://github.com/agones-dev/agones/pull/1140) ([roberthbailey](https://github.com/roberthbailey))
- Replace tabs with spaces in the svg file so that it renders more nicely. [\#1139](https://github.com/agones-dev/agones/pull/1139) ([roberthbailey](https://github.com/roberthbailey))
- Remove the svg version of the old logo. [\#1138](https://github.com/agones-dev/agones/pull/1138) ([roberthbailey](https://github.com/roberthbailey))
- CI: Update Rust SDK conformance test cache version [\#1135](https://github.com/agones-dev/agones/pull/1135) ([aLekSer](https://github.com/aLekSer))
- Add owners files for the nodejs code [\#1119](https://github.com/agones-dev/agones/pull/1119) ([roberthbailey](https://github.com/roberthbailey))

## [v1.1.0-rc](https://github.com/agones-dev/agones/tree/v1.1.0-rc) (2019-10-22)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.0.0...v1.1.0-rc)

**Breaking changes:**

- Change the API for allocator to allocation.proto [\#1123](https://github.com/agones-dev/agones/pull/1123) ([pooneh-m](https://github.com/pooneh-m))

**Implemented enhancements:**

- Allocator service should log the Agones version [\#1042](https://github.com/agones-dev/agones/issues/1042)
- Allocation policy needs to expose namespace of the targeted cluster [\#980](https://github.com/agones-dev/agones/issues/980)
- Configurable log level for agones sidecar [\#971](https://github.com/agones-dev/agones/issues/971)
- Add dynamic configuration of the sidecar http port to the unreal sdk. [\#1131](https://github.com/agones-dev/agones/pull/1131) ([roberthbailey](https://github.com/roberthbailey))
- Retry on failures for multi-cluster allocation [\#1130](https://github.com/agones-dev/agones/pull/1130) ([pooneh-m](https://github.com/pooneh-m))
- Simplify the selection of the dynamic port in the nodejs sdk. [\#1128](https://github.com/agones-dev/agones/pull/1128) ([roberthbailey](https://github.com/roberthbailey))
- Simplify the selection of the dynamic port in the Go sdk. [\#1127](https://github.com/agones-dev/agones/pull/1127) ([roberthbailey](https://github.com/roberthbailey))
- Added dynamic configuration of http port in the rust sdk [\#1125](https://github.com/agones-dev/agones/pull/1125) ([roberthbailey](https://github.com/roberthbailey))
- Added dynamic configuration of http port in the unity sdk [\#1121](https://github.com/agones-dev/agones/pull/1121) ([roberthbailey](https://github.com/roberthbailey))
- Implement converters between the GameServerAllocation API and allocation.proto [\#1117](https://github.com/agones-dev/agones/pull/1117) ([pooneh-m](https://github.com/pooneh-m))
- Add AltaVR logo to companies using Agones [\#1103](https://github.com/agones-dev/agones/pull/1103) ([TimoSchmechel](https://github.com/TimoSchmechel))
- Use an environment variable to dynamically set the grpc port in the nodejs sdk [\#1093](https://github.com/agones-dev/agones/pull/1093) ([roberthbailey](https://github.com/roberthbailey))
- Set the port to connect to the sdkserver based on the AGONES\_SDK\_GRPC\_PORT environment variable [\#1092](https://github.com/agones-dev/agones/pull/1092) ([roberthbailey](https://github.com/roberthbailey))
- Use environment variables to dynamically set the grpc port in the golang sdk. [\#1086](https://github.com/agones-dev/agones/pull/1086) ([roberthbailey](https://github.com/roberthbailey))
- Add mcmahan.games to the companies using agones list. [\#1085](https://github.com/agones-dev/agones/pull/1085) ([benmcmahan](https://github.com/benmcmahan))
- Add missing "/reserve" endpoint description [\#1083](https://github.com/agones-dev/agones/pull/1083) ([aLekSer](https://github.com/aLekSer))
- Feature Stages Documentation [\#1080](https://github.com/agones-dev/agones/pull/1080) ([markmandel](https://github.com/markmandel))
- Add SDK server HTTP API test [\#1079](https://github.com/agones-dev/agones/pull/1079) ([aLekSer](https://github.com/aLekSer))
- Sdkserver port configuration [\#1078](https://github.com/agones-dev/agones/pull/1078) ([roberthbailey](https://github.com/roberthbailey))
- Fixes, more e2e tests and logging for multi-cluster allocation [\#1077](https://github.com/agones-dev/agones/pull/1077) ([pooneh-m](https://github.com/pooneh-m))
- Longer blog post for Agones 1.0.0 announcement [\#1076](https://github.com/agones-dev/agones/pull/1076) ([markmandel](https://github.com/markmandel))
- Add a delay flag to the sdkserver [\#1070](https://github.com/agones-dev/agones/pull/1070) ([roberthbailey](https://github.com/roberthbailey))
- Add Yager Logo to companies using Agones [\#1057](https://github.com/agones-dev/agones/pull/1057) ([topochan](https://github.com/topochan))
- Adding namespace for multi-cluster allocation policy [\#1052](https://github.com/agones-dev/agones/pull/1052) ([pooneh-m](https://github.com/pooneh-m))
- Logging Agones version and port on the startup. [\#1048](https://github.com/agones-dev/agones/pull/1048) ([pooneh-m](https://github.com/pooneh-m))
- Adding make file to generate allocation go from proto [\#1041](https://github.com/agones-dev/agones/pull/1041) ([pooneh-m](https://github.com/pooneh-m))
- Add Sidecar log level parameter to GS specification [\#1007](https://github.com/agones-dev/agones/pull/1007) ([aLekSer](https://github.com/aLekSer))

**Fixed bugs:**

- Please create a root OWNERS file for agones [\#1112](https://github.com/agones-dev/agones/issues/1112)
- In a busy cluster, fleet reaction becomes slower and slower over time due to exponential back-off on requeueing [\#1107](https://github.com/agones-dev/agones/issues/1107)
- YAML installation is broken due to sdkServer validation failure [\#1090](https://github.com/agones-dev/agones/issues/1090)
- `make stress-test-e2e` run detects a race condition in test framework.go [\#1055](https://github.com/agones-dev/agones/issues/1055)
- TestAllocator is flakey [\#1050](https://github.com/agones-dev/agones/issues/1050)
- GameServer status does not account for Evicted Pods [\#1028](https://github.com/agones-dev/agones/issues/1028)
- gameserver-allocator: helm chart is missing tolerations [\#901](https://github.com/agones-dev/agones/issues/901)
- sdk/cpp cmake build fails on Linux [\#718](https://github.com/agones-dev/agones/issues/718)
- Improve fleet controller response times in busy clusters. [\#1108](https://github.com/agones-dev/agones/pull/1108) ([jkowalski](https://github.com/jkowalski))
- Fix metrics bug for when a gameserver is not retrievable [\#1101](https://github.com/agones-dev/agones/pull/1101) ([pooneh-m](https://github.com/pooneh-m))
- Fix install.yaml [\#1094](https://github.com/agones-dev/agones/pull/1094) ([aLekSer](https://github.com/aLekSer))
- Slack invite link is no longer active [\#1082](https://github.com/agones-dev/agones/pull/1082) ([markmandel](https://github.com/markmandel))
- Marking Gameservers with Evicted backing Pods as Unhealthy [\#1056](https://github.com/agones-dev/agones/pull/1056) ([aLekSer](https://github.com/aLekSer))

**Security fixes:**

- Ran `npm audit fix` to update package dependencies. [\#1097](https://github.com/agones-dev/agones/pull/1097) ([roberthbailey](https://github.com/roberthbailey))
- Bump eslint-utils from 1.4.0 to 1.4.2 in /sdks/nodejs [\#1014](https://github.com/agones-dev/agones/pull/1014) ([dependabot[bot]](https://github.com/apps/dependabot))

**Closed issues:**

- Release 1.0.0 [\#1058](https://github.com/agones-dev/agones/issues/1058)
- SDK conformance harness [\#672](https://github.com/agones-dev/agones/issues/672)

**Merged pull requests:**

- Release 1.1.0-rc [\#1134](https://github.com/agones-dev/agones/pull/1134) ([markmandel](https://github.com/markmandel))
- Style fixes. [\#1129](https://github.com/agones-dev/agones/pull/1129) ([roberthbailey](https://github.com/roberthbailey))
- Drop the Extension API Server reference from agones-allocator [\#1124](https://github.com/agones-dev/agones/pull/1124) ([pooneh-m](https://github.com/pooneh-m))
- Faster subsequent Rust SDK conformance builds [\#1122](https://github.com/agones-dev/agones/pull/1122) ([aLekSer](https://github.com/aLekSer))
- Add owners files for the C++ code [\#1118](https://github.com/agones-dev/agones/pull/1118) ([roberthbailey](https://github.com/roberthbailey))
- Move the owners file to the root of the repository. [\#1113](https://github.com/agones-dev/agones/pull/1113) ([roberthbailey](https://github.com/roberthbailey))
- Run all SDK conformance tests in parallel [\#1111](https://github.com/agones-dev/agones/pull/1111) ([aLekSer](https://github.com/aLekSer))
- Move allocation proto to root level proto [\#1110](https://github.com/agones-dev/agones/pull/1110) ([pooneh-m](https://github.com/pooneh-m))
- Remove required multi-cluster allocation policy fields that are not required [\#1100](https://github.com/agones-dev/agones/pull/1100) ([pooneh-m](https://github.com/pooneh-m))
- Fix the log output when starting the grpc gateway in the sdkserver [\#1096](https://github.com/agones-dev/agones/pull/1096) ([roberthbailey](https://github.com/roberthbailey))
- Use non-ephemeral port for Go SDK conformance test [\#1095](https://github.com/agones-dev/agones/pull/1095) ([aLekSer](https://github.com/aLekSer))
- Remove the ugly gaps in the Fleet & GameServer reference [\#1081](https://github.com/agones-dev/agones/pull/1081) ([markmandel](https://github.com/markmandel))
- Update of Docsy and Hugo to latest versions. [\#1073](https://github.com/agones-dev/agones/pull/1073) ([markmandel](https://github.com/markmandel))
- Add a simple-tcp game server to use for testing. [\#1071](https://github.com/agones-dev/agones/pull/1071) ([roberthbailey](https://github.com/roberthbailey))
- Fix Access Api Example [\#1068](https://github.com/agones-dev/agones/pull/1068) ([ramonberrutti](https://github.com/ramonberrutti))
- add allocation policy namespace field to the CRD [\#1067](https://github.com/agones-dev/agones/pull/1067) ([pooneh-m](https://github.com/pooneh-m))
- Changed remote cluster to validate PEM certificate instead of DER [\#1066](https://github.com/agones-dev/agones/pull/1066) ([pooneh-m](https://github.com/pooneh-m))
- Add func to wait for fleet condition without Fatalf [\#1065](https://github.com/agones-dev/agones/pull/1065) ([aLekSer](https://github.com/aLekSer))
- Small refactoring of the simple udp server. [\#1062](https://github.com/agones-dev/agones/pull/1062) ([roberthbailey](https://github.com/roberthbailey))
- Preparation for 1.1.0 [\#1060](https://github.com/agones-dev/agones/pull/1060) ([markmandel](https://github.com/markmandel))
- Update golangci-lint to 1.18, add bodyclose check [\#1051](https://github.com/agones-dev/agones/pull/1051) ([aLekSer](https://github.com/aLekSer))
- E2E test for Unhealthy GameServer on process crash [\#1038](https://github.com/agones-dev/agones/pull/1038) ([markmandel](https://github.com/markmandel))
- Build examples make target [\#1019](https://github.com/agones-dev/agones/pull/1019) ([aLekSer](https://github.com/aLekSer))

## [v1.0.0](https://github.com/agones-dev/agones/tree/v1.0.0) (2019-09-17)

[Full Changelog](https://github.com/agones-dev/agones/compare/v1.0.0-rc...v1.0.0)

**Closed issues:**

- Release 1.0.0-rc [\#1053](https://github.com/agones-dev/agones/issues/1053)
- Top Level Plan: 1.0 Release! [\#732](https://github.com/agones-dev/agones/issues/732)

**Merged pull requests:**

- Release 1.0.0 [\#1059](https://github.com/agones-dev/agones/pull/1059) ([markmandel](https://github.com/markmandel))

## [v1.0.0-rc](https://github.com/agones-dev/agones/tree/v1.0.0-rc) (2019-09-10)

[Full Changelog](https://github.com/agones-dev/agones/compare/v0.12.0...v1.0.0-rc)

**Implemented enhancements:**

- JSON serialisation error reporting on Mutation/Validation webhooks [\#992](https://github.com/agones-dev/agones/issues/992)
- CRASH for simple-udp example [\#1032](https://github.com/agones-dev/agones/pull/1032) ([markmandel](https://github.com/markmandel))
- Rust SDK: Reserved [\#1030](https://github.com/agones-dev/agones/pull/1030) ([markmandel](https://github.com/markmandel))
- Define the proto definition for the allocator service [\#1025](https://github.com/agones-dev/agones/pull/1025) ([pooneh-m](https://github.com/pooneh-m))
- Documentation on Fleet Updates and Upgrades [\#1020](https://github.com/agones-dev/agones/pull/1020) ([markmandel](https://github.com/markmandel))
- Documentation on how to upgrade Agones and/or Kubernetes. [\#1008](https://github.com/agones-dev/agones/pull/1008) ([markmandel](https://github.com/markmandel))
- Output JSON unmarshal error as Admission response [\#1005](https://github.com/agones-dev/agones/pull/1005) ([aLekSer](https://github.com/aLekSer))
- Add GameServer troubleshooting guide [\#1003](https://github.com/agones-dev/agones/pull/1003) ([markmandel](https://github.com/markmandel))
- Local SDK Server: Add proper GS state handling [\#979](https://github.com/agones-dev/agones/pull/979) ([aLekSer](https://github.com/aLekSer))
- Add allocations metrics [\#963](https://github.com/agones-dev/agones/pull/963) ([cyriltovena](https://github.com/cyriltovena))

**Fixed bugs:**

- Fleet Image Edit causes an infinite create/destroy loop [\#975](https://github.com/agones-dev/agones/issues/975)
- Fix the required version of terraform. [\#1006](https://github.com/agones-dev/agones/pull/1006) ([roberthbailey](https://github.com/roberthbailey))

**Closed issues:**

- Release 0.12.0 [\#982](https://github.com/agones-dev/agones/issues/982)
- Document upgrading / managing Fleets [\#557](https://github.com/agones-dev/agones/issues/557)
- Document how to do upgrades of Agones [\#555](https://github.com/agones-dev/agones/issues/555)
- Statistics collection and display [\#144](https://github.com/agones-dev/agones/issues/144)

**Merged pull requests:**

- Release 1.0.0 Release Candidate [\#1054](https://github.com/agones-dev/agones/pull/1054) ([markmandel](https://github.com/markmandel))
- Change allocator's preferredGameServerSelector field name to plural [\#1047](https://github.com/agones-dev/agones/pull/1047) ([pooneh-m](https://github.com/pooneh-m))
- Fix a broken link in the node js client sdk docs. [\#1045](https://github.com/agones-dev/agones/pull/1045) ([roberthbailey](https://github.com/roberthbailey))
- Fix for git.apache.org being down. [\#1031](https://github.com/agones-dev/agones/pull/1031) ([markmandel](https://github.com/markmandel))
- Cpp SDK. Fixed regex for version detection. Fixed mingw build. [\#1029](https://github.com/agones-dev/agones/pull/1029) ([dsazonoff](https://github.com/dsazonoff))
- flaky/TestFleetScaleUpEditAndScaleDown [\#1024](https://github.com/agones-dev/agones/pull/1024) ([markmandel](https://github.com/markmandel))
- add vendor grpc third\_party to be referenced by protos [\#1023](https://github.com/agones-dev/agones/pull/1023) ([pooneh-m](https://github.com/pooneh-m))
- flaky/TestGameServerShutdown [\#1022](https://github.com/agones-dev/agones/pull/1022) ([markmandel](https://github.com/markmandel))
- Fix typo in Makefile [\#1021](https://github.com/agones-dev/agones/pull/1021) ([orthros](https://github.com/orthros))
- Add configuration for the prow build cluster along with a make target to create and delete it [\#1017](https://github.com/agones-dev/agones/pull/1017) ([roberthbailey](https://github.com/roberthbailey))
- Add an OWNERS file at the root of the repository [\#1016](https://github.com/agones-dev/agones/pull/1016) ([roberthbailey](https://github.com/roberthbailey))
- Refactor gameserverallocations to its components [\#1015](https://github.com/agones-dev/agones/pull/1015) ([pooneh-m](https://github.com/pooneh-m))
- added an allowing UDP traffic section to the EKS installation docs [\#1011](https://github.com/agones-dev/agones/pull/1011) ([daplho](https://github.com/daplho))
- Fix outdated links in comments [\#1009](https://github.com/agones-dev/agones/pull/1009) ([aLekSer](https://github.com/aLekSer))
- Add note about SDK Sidecar starting after gameserver binary [\#1004](https://github.com/agones-dev/agones/pull/1004) ([markmandel](https://github.com/markmandel))
- Flaky: TestControllerApplyGameServerAddressAndPort [\#1002](https://github.com/agones-dev/agones/pull/1002) ([markmandel](https://github.com/markmandel))
- Revert the change to promote the service for multicluster allocation to v1 [\#1001](https://github.com/agones-dev/agones/pull/1001) ([roberthbailey](https://github.com/roberthbailey))
- Remove Go SDK repeat connection attempt [\#998](https://github.com/agones-dev/agones/pull/998) ([markmandel](https://github.com/markmandel))
- Adding clean target for SDK conformance tests [\#997](https://github.com/agones-dev/agones/pull/997) ([aLekSer](https://github.com/aLekSer))
- Speed up CI Build [\#996](https://github.com/agones-dev/agones/pull/996) ([markmandel](https://github.com/markmandel))
- Flaky: Allocator bad tls certificate [\#995](https://github.com/agones-dev/agones/pull/995) ([markmandel](https://github.com/markmandel))
- Rename gameserver-allocator resources to agones-allocator [\#994](https://github.com/agones-dev/agones/pull/994) ([markmandel](https://github.com/markmandel))
- Fix instructions printed out after helm install [\#991](https://github.com/agones-dev/agones/pull/991) ([aLekSer](https://github.com/aLekSer))
- Capitalize "GitHub" correctly throughout the docs [\#990](https://github.com/agones-dev/agones/pull/990) ([hegemonic](https://github.com/hegemonic))
- Fix Hugo syntax - remove deprecation warning [\#988](https://github.com/agones-dev/agones/pull/988) ([aLekSer](https://github.com/aLekSer))
- Remove the instructions for using click to deploy on GCP [\#987](https://github.com/agones-dev/agones/pull/987) ([roberthbailey](https://github.com/roberthbailey))
- Preparation for the 1.0.0 next release [\#984](https://github.com/agones-dev/agones/pull/984) ([markmandel](https://github.com/markmandel))
- Docs Installation use cluster size the same as dev [\#981](https://github.com/agones-dev/agones/pull/981) ([aLekSer](https://github.com/aLekSer))
- Flaky: TestGameServerUnhealthyAfterDeletingPod [\#968](https://github.com/agones-dev/agones/pull/968) ([markmandel](https://github.com/markmandel))

## [v0.12.0](https://github.com/agones-dev/agones/tree/v0.12.0) (2019-08-07)

[Full Changelog](https://github.com/agones-dev/agones/compare/v0.12.0-rc...v0.12.0)

**Closed issues:**

- Release 0.12.0-rc [\#972](https://github.com/agones-dev/agones/issues/972)

**Merged pull requests:**

- Release 0.12.0 [\#983](https://github.com/agones-dev/agones/pull/983) ([markmandel](https://github.com/markmandel))
- Minor Fix 0.8.1 release on agones.dev Blog [\#978](https://github.com/agones-dev/agones/pull/978) ([aLekSer](https://github.com/aLekSer))
- Minor - Fix helm repo command in the governance template [\#977](https://github.com/agones-dev/agones/pull/977) ([aLekSer](https://github.com/aLekSer))
- Documentation updates to apply just prior to cutting the 0.12.0 release. [\#911](https://github.com/agones-dev/agones/pull/911) ([roberthbailey](https://github.com/roberthbailey))

## [v0.12.0-rc](https://github.com/agones-dev/agones/tree/v0.12.0-rc) (2019-08-01)

[Full Changelog](https://github.com/agones-dev/agones/compare/v0.11.0...v0.12.0-rc)

**Breaking changes:**

- C++ SDK: Use const-reference in WatchGameServer [\#941](https://github.com/agones-dev/agones/issues/941)
- Proposal: Split up the api group stable.agones.dev [\#703](https://github.com/agones-dev/agones/issues/703)
- Update the supported version of Kubernetes to 1.12. [\#967](https://github.com/agones-dev/agones/pull/967) ([roberthbailey](https://github.com/roberthbailey))
- Update the node affinity key to the new label name. [\#964](https://github.com/agones-dev/agones/pull/964) ([roberthbailey](https://github.com/roberthbailey))
- Implement block on connect with Rust+Node.js SDK [\#953](https://github.com/agones-dev/agones/pull/953) ([markmandel](https://github.com/markmandel))
- C++ SDK: Update the function signature of WatchGameServer to use a const-reference [\#951](https://github.com/agones-dev/agones/pull/951) ([roberthbailey](https://github.com/roberthbailey))
- Update GKE documentation to 1.12 [\#897](https://github.com/agones-dev/agones/pull/897) ([roberthbailey](https://github.com/roberthbailey))
- Move the stable api group and promote it to v1 [\#894](https://github.com/agones-dev/agones/pull/894) ([roberthbailey](https://github.com/roberthbailey))
- Promote allocation to v1 [\#881](https://github.com/agones-dev/agones/pull/881) ([roberthbailey](https://github.com/roberthbailey))
- Promote autoscaling to v1 [\#874](https://github.com/agones-dev/agones/pull/874) ([roberthbailey](https://github.com/roberthbailey))
- Remove / Expire FleetAllocation from documentation. [\#867](https://github.com/agones-dev/agones/pull/867) ([markmandel](https://github.com/markmandel))
- Remove FleetAllocation. [\#856](https://github.com/agones-dev/agones/pull/856) ([markmandel](https://github.com/markmandel))

**Implemented enhancements:**

- Make all yaml files in the examples directory use working configurations / images [\#969](https://github.com/agones-dev/agones/issues/969)
- Move nodejs example to a docker build [\#943](https://github.com/agones-dev/agones/issues/943)
- Upgrade to Kubernetes 1.12 [\#717](https://github.com/agones-dev/agones/issues/717)
- 1st Party / Distributed Matchmaker support [\#660](https://github.com/agones-dev/agones/issues/660)
- SDK Build/test pipeline separation from build-image [\#599](https://github.com/agones-dev/agones/issues/599)
- Move to using CRD Subresources for all Agones CRDs [\#329](https://github.com/agones-dev/agones/issues/329)
- Unity Plugin SDK [\#246](https://github.com/agones-dev/agones/issues/246)
- Add Reserve to Node.js SDK [\#955](https://github.com/agones-dev/agones/pull/955) ([steven-supersolid](https://github.com/steven-supersolid))
- Add the missing functions to the C++ SDK \(Allocated & Reserve\) [\#948](https://github.com/agones-dev/agones/pull/948) ([roberthbailey](https://github.com/roberthbailey))
- Update the nodejs example to build in a docker image [\#945](https://github.com/agones-dev/agones/pull/945) ([roberthbailey](https://github.com/roberthbailey))
- Updates to the C++ SDK along with the simple example that exercises it. [\#934](https://github.com/agones-dev/agones/pull/934) ([roberthbailey](https://github.com/roberthbailey))
- Update GameServer state diagram with Reserved [\#933](https://github.com/agones-dev/agones/pull/933) ([markmandel](https://github.com/markmandel))
- E2E tests for SDK.Reserve\(seconds\) [\#925](https://github.com/agones-dev/agones/pull/925) ([markmandel](https://github.com/markmandel))
- Add new GameServer lifecycle diagrams for Reserved [\#922](https://github.com/agones-dev/agones/pull/922) ([markmandel](https://github.com/markmandel))
- Compliance tests for Reserve\(seconds\). [\#920](https://github.com/agones-dev/agones/pull/920) ([markmandel](https://github.com/markmandel))
- Reserve\(\) SDK implementation [\#891](https://github.com/agones-dev/agones/pull/891) ([markmandel](https://github.com/markmandel))
- Update GKE development tooling to 1.12 [\#887](https://github.com/agones-dev/agones/pull/887) ([markmandel](https://github.com/markmandel))
- Fix Rust SDK, add allocate, add conformance test [\#879](https://github.com/agones-dev/agones/pull/879) ([aLekSer](https://github.com/aLekSer))
- Add instructions about taints and tolerations to the install instructions [\#870](https://github.com/agones-dev/agones/pull/870) ([roberthbailey](https://github.com/roberthbailey))
- Add events to SDK state change operations [\#866](https://github.com/agones-dev/agones/pull/866) ([markmandel](https://github.com/markmandel))
- Add ReserveUntil to GameServer.Status [\#865](https://github.com/agones-dev/agones/pull/865) ([markmandel](https://github.com/markmandel))
- add unity example [\#860](https://github.com/agones-dev/agones/pull/860) ([whisper0077](https://github.com/whisper0077))
- SDK Conformance testing [\#848](https://github.com/agones-dev/agones/pull/848) ([aLekSer](https://github.com/aLekSer))
- Reserve proto definition and generated code [\#820](https://github.com/agones-dev/agones/pull/820) ([markmandel](https://github.com/markmandel))
- Cpp prerequisities cmake [\#803](https://github.com/agones-dev/agones/pull/803) ([dsazonoff](https://github.com/dsazonoff))

**Fixed bugs:**

- Rust SDK does not wait for connection to be ready [\#938](https://github.com/agones-dev/agones/issues/938)
- Unable to build the rust-simple example [\#935](https://github.com/agones-dev/agones/issues/935)
- Building the rust sdk leaves untracked files [\#912](https://github.com/agones-dev/agones/issues/912)
- Fail to pass Health Check on Agones + UE4 plugin [\#861](https://github.com/agones-dev/agones/issues/861)
- Agones Getting Started Guide with Minikube uses wrong IP due to minikube bug  [\#751](https://github.com/agones-dev/agones/issues/751)
- Flaky - e2e cp: cannot stat [\#919](https://github.com/agones-dev/agones/pull/919) ([markmandel](https://github.com/markmandel))
- added affinity and tolerations to gameserver-allocator [\#910](https://github.com/agones-dev/agones/pull/910) ([daplho](https://github.com/daplho))
- fix: Fix gRCP context leaks. [\#904](https://github.com/agones-dev/agones/pull/904) ([devjgm](https://github.com/devjgm))
- Fix minikube developer experience [\#898](https://github.com/agones-dev/agones/pull/898) ([markmandel](https://github.com/markmandel))
- Fix timeout on Terraform Helm install agones step [\#890](https://github.com/agones-dev/agones/pull/890) ([aLekSer](https://github.com/aLekSer))
- Flaky: TestGameServerPassthroughPort [\#863](https://github.com/agones-dev/agones/pull/863) ([markmandel](https://github.com/markmandel))

**Security fixes:**

- Need to Bump js-yaml from 3.12.1 to 3.13.1 in /sdks/nodejs  [\#868](https://github.com/agones-dev/agones/issues/868)
- Update node.js coverage, dependencies and potential issue [\#954](https://github.com/agones-dev/agones/pull/954) ([steven-supersolid](https://github.com/steven-supersolid))

**Closed issues:**

- Approver access for @roberthbailey [\#914](https://github.com/agones-dev/agones/issues/914)
- Release 0.11.0 [\#849](https://github.com/agones-dev/agones/issues/849)
- NodeJS example needs a description in the README [\#728](https://github.com/agones-dev/agones/issues/728)
- C++ SDK should follow Google Style  [\#713](https://github.com/agones-dev/agones/issues/713)
- Write a guide for setting up Agones with taints and tolerations [\#491](https://github.com/agones-dev/agones/issues/491)

**Merged pull requests:**

- Typo in the changelog. [\#974](https://github.com/agones-dev/agones/pull/974) ([markmandel](https://github.com/markmandel))
- Release 0.12.0-rc [\#973](https://github.com/agones-dev/agones/pull/973) ([markmandel](https://github.com/markmandel))
- Fix Reference docs and sync it with Examples dir [\#970](https://github.com/agones-dev/agones/pull/970) ([aLekSer](https://github.com/aLekSer))
- Fix AWS EKS cluster creating docs [\#965](https://github.com/agones-dev/agones/pull/965) ([aLekSer](https://github.com/aLekSer))
- Fix Installation Docs and example GS configuration [\#962](https://github.com/agones-dev/agones/pull/962) ([aLekSer](https://github.com/aLekSer))
- \[Fix Breaking CI\] Site test failure due to 404s [\#961](https://github.com/agones-dev/agones/pull/961) ([markmandel](https://github.com/markmandel))
- Fix the capitalization for 'publishVersion' in the website docs. [\#960](https://github.com/agones-dev/agones/pull/960) ([roberthbailey](https://github.com/roberthbailey))
- Remove GameServer UpdateStatus\(\) since its not being used. [\#959](https://github.com/agones-dev/agones/pull/959) ([markmandel](https://github.com/markmandel))
- Update the container image version for the simple unity sdk. [\#952](https://github.com/agones-dev/agones/pull/952) ([roberthbailey](https://github.com/roberthbailey))
- Update the xonotic wrapper binary to the new Go SDK version. [\#950](https://github.com/agones-dev/agones/pull/950) ([roberthbailey](https://github.com/roberthbailey))
- Fix indentation of two of the build targets. [\#944](https://github.com/agones-dev/agones/pull/944) ([roberthbailey](https://github.com/roberthbailey))
- Updates to the rust simple example. [\#937](https://github.com/agones-dev/agones/pull/937) ([roberthbailey](https://github.com/roberthbailey))
- Update Dockerfile for Rust simple [\#936](https://github.com/agones-dev/agones/pull/936) ([aLekSer](https://github.com/aLekSer))
- With the extra whitespace the rendered output has undesirable new paragraphs. [\#932](https://github.com/agones-dev/agones/pull/932) ([roberthbailey](https://github.com/roberthbailey))
- Updates to the create webhook fleetautoscaler guide. [\#931](https://github.com/agones-dev/agones/pull/931) ([roberthbailey](https://github.com/roberthbailey))
- Updates to the allocator service tutorial. [\#930](https://github.com/agones-dev/agones/pull/930) ([roberthbailey](https://github.com/roberthbailey))
- Update the dev-gameserver example to use the current udp-server container image. [\#929](https://github.com/agones-dev/agones/pull/929) ([roberthbailey](https://github.com/roberthbailey))
- Update K8s Code Generation Tooling to 1.12 [\#928](https://github.com/agones-dev/agones/pull/928) ([markmandel](https://github.com/markmandel))
- Flaky: Make preview --no-promote to avoid promotion race conditions [\#926](https://github.com/agones-dev/agones/pull/926) ([markmandel](https://github.com/markmandel))
- Fix a small typo. [\#923](https://github.com/agones-dev/agones/pull/923) ([roberthbailey](https://github.com/roberthbailey))
- Fixup the formatting on Fleet Getting Started Guide [\#921](https://github.com/agones-dev/agones/pull/921) ([markmandel](https://github.com/markmandel))
- Update client-go to 9.0.0 and k8s api to 1.12.10 [\#918](https://github.com/agones-dev/agones/pull/918) ([heartrobotninja](https://github.com/heartrobotninja))
- Fix the command to teardown the test cluster in the GKE developer instructions. [\#916](https://github.com/agones-dev/agones/pull/916) ([roberthbailey](https://github.com/roberthbailey))
- Newline at the end of go.mod [\#915](https://github.com/agones-dev/agones/pull/915) ([roberthbailey](https://github.com/roberthbailey))
- Fix Rust SDK conformance flakiness, add clean step [\#913](https://github.com/agones-dev/agones/pull/913) ([aLekSer](https://github.com/aLekSer))
- Cmake. Removed unused script. [\#907](https://github.com/agones-dev/agones/pull/907) ([dsazonoff](https://github.com/dsazonoff))
- Changed cmake required version to 3.5 to work on more machines [\#903](https://github.com/agones-dev/agones/pull/903) ([devjgm](https://github.com/devjgm))
- Upgrading Terraform to 0.12.3 [\#899](https://github.com/agones-dev/agones/pull/899) ([aLekSer](https://github.com/aLekSer))
- Update kind dev tooling to 1.12 [\#896](https://github.com/agones-dev/agones/pull/896) ([roberthbailey](https://github.com/roberthbailey))
- Update minikube documentation and dev tooling to 1.12 [\#895](https://github.com/agones-dev/agones/pull/895) ([roberthbailey](https://github.com/roberthbailey))
- Explicitly disable creation of the client certificate on GKE [\#888](https://github.com/agones-dev/agones/pull/888) ([roberthbailey](https://github.com/roberthbailey))
- Document SDK - Allocated-\>Ready\(\) [\#886](https://github.com/agones-dev/agones/pull/886) ([markmandel](https://github.com/markmandel))
- Small improvements to create-fleet quickstart [\#885](https://github.com/agones-dev/agones/pull/885) ([markmandel](https://github.com/markmandel))
- Last touches to the install guide regarding taints and tolerations [\#882](https://github.com/agones-dev/agones/pull/882) ([roberthbailey](https://github.com/roberthbailey))
- Make GameServerAllocation reference document [\#880](https://github.com/agones-dev/agones/pull/880) ([markmandel](https://github.com/markmandel))
- Fix ensure SDK image [\#878](https://github.com/agones-dev/agones/pull/878) ([aLekSer](https://github.com/aLekSer))
- Fix for CRD API docs generator [\#877](https://github.com/agones-dev/agones/pull/877) ([aLekSer](https://github.com/aLekSer))
- Update gen-sdk-grpc make target name. [\#873](https://github.com/agones-dev/agones/pull/873) ([roberthbailey](https://github.com/roberthbailey))
- Add a note about the gameserver IP not being reachable when using minikube [\#871](https://github.com/agones-dev/agones/pull/871) ([roberthbailey](https://github.com/roberthbailey))
- Docs: Centralise udp-server tag [\#869](https://github.com/agones-dev/agones/pull/869) ([markmandel](https://github.com/markmandel))
- Unreal Engine plugin - Health Ping Enabled to true by default [\#864](https://github.com/agones-dev/agones/pull/864) ([edwardchuang](https://github.com/edwardchuang))
- Fix for podtemplatespec link [\#862](https://github.com/agones-dev/agones/pull/862) ([markmandel](https://github.com/markmandel))
- Update gke install instructions [\#857](https://github.com/agones-dev/agones/pull/857) ([roberthbailey](https://github.com/roberthbailey))
- Cpp clang-format [\#855](https://github.com/agones-dev/agones/pull/855) ([dsazonoff](https://github.com/dsazonoff))
- Preparation for 0.12.0 sprint [\#852](https://github.com/agones-dev/agones/pull/852) ([markmandel](https://github.com/markmandel))
- Update CPP Example Readme and yaml files [\#847](https://github.com/agones-dev/agones/pull/847) ([markmandel](https://github.com/markmandel))
- Add Unity to the SDK page. [\#846](https://github.com/agones-dev/agones/pull/846) ([markmandel](https://github.com/markmandel))

## [v0.11.0](https://github.com/agones-dev/agones/tree/v0.11.0) (2019-06-25)

[Full Changelog](https://github.com/agones-dev/agones/compare/v0.11.0-rc...v0.11.0)

**Fixed bugs:**

- Set secret namespace to agones-system for allocator service [\#843](https://github.com/agones-dev/agones/pull/843) ([pooneh-m](https://github.com/pooneh-m))

**Closed issues:**

- Release 0.11.0-rc [\#841](https://github.com/agones-dev/agones/issues/841)

**Merged pull requests:**

- Release 0.11.0 [\#850](https://github.com/agones-dev/agones/pull/850) ([markmandel](https://github.com/markmandel))
- Flaky: TestAutoscalerWebhook [\#844](https://github.com/agones-dev/agones/pull/844) ([aLekSer](https://github.com/aLekSer))

## [v0.11.0-rc](https://github.com/agones-dev/agones/tree/v0.11.0-rc) (2019-06-18)

[Full Changelog](https://github.com/agones-dev/agones/compare/v0.10.0...v0.11.0-rc)

**Breaking changes:**

- Move FleetAutoscaling to autoscaling.agones.dev group [\#829](https://github.com/agones-dev/agones/pull/829) ([markmandel](https://github.com/markmandel))
- Fixing SDK proto file according to style guide [\#776](https://github.com/agones-dev/agones/pull/776) ([aLekSer](https://github.com/aLekSer))

**Implemented enhancements:**

- Add Events for common errors with Webhook FleetAutoscaler configuration [\#792](https://github.com/agones-dev/agones/issues/792)
- Self allocation in Node.js is not supported [\#773](https://github.com/agones-dev/agones/issues/773)
- In case of dynamic port allocation, offer the option to set the container port to the same value as the host port [\#294](https://github.com/agones-dev/agones/issues/294)
- Implement EnqueueAfter on WorkerQueue [\#835](https://github.com/agones-dev/agones/pull/835) ([markmandel](https://github.com/markmandel))
- Changed AllocationEndpoint to array of endpoints [\#830](https://github.com/agones-dev/agones/pull/830) ([pooneh-m](https://github.com/pooneh-m))
- fix: check if NodeExternalIP is empty to fallback to NodeInternalIP [\#828](https://github.com/agones-dev/agones/pull/828) ([aarnaud](https://github.com/aarnaud))
- Rewrite Agones Overview [\#824](https://github.com/agones-dev/agones/pull/824) ([markmandel](https://github.com/markmandel))
- Add Unity SDK [\#818](https://github.com/agones-dev/agones/pull/818) ([whisper0077](https://github.com/whisper0077))
- PortPolicy of Passthrough - Same Port for Container and Host [\#817](https://github.com/agones-dev/agones/pull/817) ([markmandel](https://github.com/markmandel))
- Add Fleet RollingUpdate strategy params validation [\#808](https://github.com/agones-dev/agones/pull/808) ([aLekSer](https://github.com/aLekSer))
- Batched Packed and Distributed Allocations [\#804](https://github.com/agones-dev/agones/pull/804) ([markmandel](https://github.com/markmandel))
- Add Events on FleetAutoscaler connection errors [\#794](https://github.com/agones-dev/agones/pull/794) ([aLekSer](https://github.com/aLekSer))
- Expose allocate method in node sdk [\#774](https://github.com/agones-dev/agones/pull/774) ([rorygarand](https://github.com/rorygarand))
- Adding an allocator service that acts as a reverse proxy. [\#768](https://github.com/agones-dev/agones/pull/768) ([pooneh-m](https://github.com/pooneh-m))
- Add Reserved GameServer State [\#766](https://github.com/agones-dev/agones/pull/766) ([markmandel](https://github.com/markmandel))
- Add AKS, GKE and Helm terraform modules [\#756](https://github.com/agones-dev/agones/pull/756) ([aLekSer](https://github.com/aLekSer))
- Add close method to node client [\#748](https://github.com/agones-dev/agones/pull/748) ([BradfordMedeiros](https://github.com/BradfordMedeiros))

**Fixed bugs:**

- Allocator service needs to get the namespace from input and not environment. [\#809](https://github.com/agones-dev/agones/issues/809)
- apiserver role binding is referencing an invalid system account [\#805](https://github.com/agones-dev/agones/issues/805)
- Fleet scale down doesn't work after an update [\#800](https://github.com/agones-dev/agones/issues/800)
- Fleet Rolling Update doesn't seem to be rolling [\#799](https://github.com/agones-dev/agones/issues/799)
- Packed Allocation is very not packed [\#783](https://github.com/agones-dev/agones/issues/783)
- If GameServer webhook validation fails, it doesn't raise up to Fleet [\#765](https://github.com/agones-dev/agones/issues/765)
- Some Gameservers stays in Unhealthy state \(instead of being deleted\) [\#736](https://github.com/agones-dev/agones/issues/736)
- GS Shutdown sdk calls sometimes failed/timeout and leave Gameservers behind [\#624](https://github.com/agones-dev/agones/issues/624)
- Adding apiGroup to roleRef for gameservice-allocator [\#825](https://github.com/agones-dev/agones/pull/825) ([pooneh-m](https://github.com/pooneh-m))
- Add ShutdownReplicas count [\#810](https://github.com/agones-dev/agones/pull/810) ([aLekSer](https://github.com/aLekSer))
- Fix Down Scale on RollingUpdate [\#802](https://github.com/agones-dev/agones/pull/802) ([aLekSer](https://github.com/aLekSer))
- Fix publishDate on unreal docs [\#793](https://github.com/agones-dev/agones/pull/793) ([markmandel](https://github.com/markmandel))
- Flaky: TestAllocator [\#789](https://github.com/agones-dev/agones/pull/789) ([markmandel](https://github.com/markmandel))
- Prevent race conditions by syncing node cache on GameServer controller [\#782](https://github.com/agones-dev/agones/pull/782) ([markmandel](https://github.com/markmandel))
- Prevent race conditions by syncing cache on new Allocation elements [\#780](https://github.com/agones-dev/agones/pull/780) ([markmandel](https://github.com/markmandel))
- Fix for front link. Not sure what happened? [\#772](https://github.com/agones-dev/agones/pull/772) ([markmandel](https://github.com/markmandel))
- Add validation of the fleet underlying gameserver [\#771](https://github.com/agones-dev/agones/pull/771) ([aLekSer](https://github.com/aLekSer))

**Closed issues:**

- Request to become an Approver on Agones [\#796](https://github.com/agones-dev/agones/issues/796)
- Approver access for @pooneh-m [\#787](https://github.com/agones-dev/agones/issues/787)
- Release 0.10.0 [\#769](https://github.com/agones-dev/agones/issues/769)
- Use batching in GameServerAllocation controller to improve throughput. [\#536](https://github.com/agones-dev/agones/issues/536)
- Improve fleet scaling performance [\#483](https://github.com/agones-dev/agones/issues/483)
- End to End test [\#37](https://github.com/agones-dev/agones/issues/37)

**Merged pull requests:**

- Release 0.11.0-rc [\#842](https://github.com/agones-dev/agones/pull/842) ([markmandel](https://github.com/markmandel))
- Flaky: TestFleetRecreateGameServers [\#840](https://github.com/agones-dev/agones/pull/840) ([markmandel](https://github.com/markmandel))
- Flaky: TestAllocator [\#839](https://github.com/agones-dev/agones/pull/839) ([markmandel](https://github.com/markmandel))
- Flaky: TestFleetRollingUpdate [\#838](https://github.com/agones-dev/agones/pull/838) ([markmandel](https://github.com/markmandel))
- Flaky: TestSDKSetAnnotation [\#837](https://github.com/agones-dev/agones/pull/837) ([markmandel](https://github.com/markmandel))
- Move all to https://github.com/agones-dev/agones [\#836](https://github.com/agones-dev/agones/pull/836) ([markmandel](https://github.com/markmandel))
- E2E test for Ready-\>Allocated-\>Ready [\#834](https://github.com/agones-dev/agones/pull/834) ([markmandel](https://github.com/markmandel))
- Remove GSA Experimental warnings. [\#833](https://github.com/agones-dev/agones/pull/833) ([markmandel](https://github.com/markmandel))
- More Google Inc. -\> Google LLC Licence changes. [\#832](https://github.com/agones-dev/agones/pull/832) ([markmandel](https://github.com/markmandel))
- Flaky: TestControllerSyncFleetAutoscaler [\#827](https://github.com/agones-dev/agones/pull/827) ([markmandel](https://github.com/markmandel))
- Move lifecycle diagram to use GameServerAllocation [\#823](https://github.com/agones-dev/agones/pull/823) ([markmandel](https://github.com/markmandel))
- Fix instructions for getting pprof information from controller [\#822](https://github.com/agones-dev/agones/pull/822) ([markmandel](https://github.com/markmandel))
- Fix header alignment on Fleet Reference [\#821](https://github.com/agones-dev/agones/pull/821) ([markmandel](https://github.com/markmandel))
- --promote on development site [\#819](https://github.com/agones-dev/agones/pull/819) ([markmandel](https://github.com/markmandel))
- Flaky: TestHealthControllerRun [\#816](https://github.com/agones-dev/agones/pull/816) ([markmandel](https://github.com/markmandel))
- Handle delete events when caching Ready GameServers for Allocation [\#815](https://github.com/agones-dev/agones/pull/815) ([markmandel](https://github.com/markmandel))
- Remove env namespace dependency from allocator service. [\#814](https://github.com/agones-dev/agones/pull/814) ([pooneh-m](https://github.com/pooneh-m))
- Flaky: TestAutoscalerWebhook [\#812](https://github.com/agones-dev/agones/pull/812) ([markmandel](https://github.com/markmandel))
- Update Locust to 0.9 image and set cpu requests/limits [\#807](https://github.com/agones-dev/agones/pull/807) ([markmandel](https://github.com/markmandel))
- Correcting the reference to the service account role binding. \#805 [\#806](https://github.com/agones-dev/agones/pull/806) ([bbf](https://github.com/bbf))
- Add HandleError input parameter check [\#798](https://github.com/agones-dev/agones/pull/798) ([aLekSer](https://github.com/aLekSer))
- Fix locust tests [\#797](https://github.com/agones-dev/agones/pull/797) ([aLekSer](https://github.com/aLekSer))
- Fleetautoscaler: Add check to have minReplicas\>0 when use percents bufferSize [\#795](https://github.com/agones-dev/agones/pull/795) ([aLekSer](https://github.com/aLekSer))
- Add missing validation for GameServerSet, refactor [\#791](https://github.com/agones-dev/agones/pull/791) ([aLekSer](https://github.com/aLekSer))
- Add a link in the minikube install instructions to jump to creating the required cluster role binding [\#790](https://github.com/agones-dev/agones/pull/790) ([roberthbailey](https://github.com/roberthbailey))
- Docs: FleetAllocation example fix [\#788](https://github.com/agones-dev/agones/pull/788) ([rorygarand](https://github.com/rorygarand))
- Fix spelling in docs and comments [\#785](https://github.com/agones-dev/agones/pull/785) ([aLekSer](https://github.com/aLekSer))
- Flakey: TestControllerSyncGameServerDeletionTimestamp [\#781](https://github.com/agones-dev/agones/pull/781) ([markmandel](https://github.com/markmandel))
- Flaky: Race condition in TestControllerSyncGameServerStartingState [\#779](https://github.com/agones-dev/agones/pull/779) ([markmandel](https://github.com/markmandel))
- Update SDK page with full list of current SDKs. [\#778](https://github.com/agones-dev/agones/pull/778) ([markmandel](https://github.com/markmandel))
- Fix spelling mistake in grafana dashboard. [\#777](https://github.com/agones-dev/agones/pull/777) ([markmandel](https://github.com/markmandel))
- Preparation for 0.11.0 release! [\#775](https://github.com/agones-dev/agones/pull/775) ([markmandel](https://github.com/markmandel))
- Revert UnHealthy standalone GameServers to not be deleted [\#763](https://github.com/agones-dev/agones/pull/763) ([markmandel](https://github.com/markmandel))

## [v0.10.0](https://github.com/agones-dev/agones/tree/v0.10.0) (2019-05-16)

[Full Changelog](https://github.com/agones-dev/agones/compare/v0.10.0-rc...v0.10.0)

**Fixed bugs:**

- Add secret list and watch permissions to RBAC rules [\#762](https://github.com/agones-dev/agones/pull/762) ([pooneh-m](https://github.com/pooneh-m))

**Closed issues:**

- Release 0.10.0-rc [\#759](https://github.com/agones-dev/agones/issues/759)

**Merged pull requests:**

- Release 0.10.0 [\#770](https://github.com/agones-dev/agones/pull/770) ([markmandel](https://github.com/markmandel))
- Update examples list [\#767](https://github.com/agones-dev/agones/pull/767) ([markmandel](https://github.com/markmandel))
- Update Fleet autoscaling documentation [\#764](https://github.com/agones-dev/agones/pull/764) ([markmandel](https://github.com/markmandel))
- Bad copy paste on 0.10.0 rc release page [\#761](https://github.com/agones-dev/agones/pull/761) ([markmandel](https://github.com/markmandel))

## [v0.10.0-rc](https://github.com/agones-dev/agones/tree/v0.10.0-rc) (2019-05-08)

[Full Changelog](https://github.com/agones-dev/agones/compare/v0.9.0...v0.10.0-rc)

**Breaking changes:**

- Add status subresource for FleetAutoscaler [\#730](https://github.com/agones-dev/agones/pull/730) ([aLekSer](https://github.com/aLekSer))
- Implement GameServerAlocation as API Extension  [\#682](https://github.com/agones-dev/agones/pull/682) ([markmandel](https://github.com/markmandel))

**Implemented enhancements:**

- Implementing cross cluster allocation request [\#757](https://github.com/agones-dev/agones/pull/757) ([pooneh-m](https://github.com/pooneh-m))
- Rename APIServerEndpoint to AllocationEndpoint for multi-cluster allocation [\#755](https://github.com/agones-dev/agones/pull/755) ([pooneh-m](https://github.com/pooneh-m))
- Implement multicluster allocation for local cluster allocation. [\#753](https://github.com/agones-dev/agones/pull/753) ([pooneh-m](https://github.com/pooneh-m))
- Implementing cluster selector from multi-cluster allocation policies. [\#733](https://github.com/agones-dev/agones/pull/733) ([pooneh-m](https://github.com/pooneh-m))
- Added Supersolid logo to the homepage [\#727](https://github.com/agones-dev/agones/pull/727) ([KamiMay](https://github.com/KamiMay))
- Implementation of SDK.Allocate\(\) [\#721](https://github.com/agones-dev/agones/pull/721) ([markmandel](https://github.com/markmandel))
- Add allocation policy CRD and schema definition. [\#698](https://github.com/agones-dev/agones/pull/698) ([pooneh-m](https://github.com/pooneh-m))
- Helm support for Terraform [\#696](https://github.com/agones-dev/agones/pull/696) ([aLekSer](https://github.com/aLekSer))
- Implement lacking functions in Rust SDK [\#693](https://github.com/agones-dev/agones/pull/693) ([thara](https://github.com/thara))
- Terraform support to generate test cluster [\#670](https://github.com/agones-dev/agones/pull/670) ([aLekSer](https://github.com/aLekSer))
- Lightweight library for implementing APIServer extensions [\#659](https://github.com/agones-dev/agones/pull/659) ([markmandel](https://github.com/markmandel))
- Unreal Engine 4 Plugin [\#647](https://github.com/agones-dev/agones/pull/647) ([YannickLange](https://github.com/YannickLange))

**Fixed bugs:**

- Ensure memory leak fix in apimachinery wait.go fix does not get overwritten [\#734](https://github.com/agones-dev/agones/issues/734)
- Flaky Test: TestGameServerAllocationMetaDataPatch [\#725](https://github.com/agones-dev/agones/issues/725)
- gen-api-docs make target is not generating API docs for GameServerAllocation [\#705](https://github.com/agones-dev/agones/issues/705)
- Agones controller does not remove deleted pod from game server list [\#678](https://github.com/agones-dev/agones/issues/678)
- Flaky: Fix test for TestGameServerUnhealthyAfterDeletingPod [\#758](https://github.com/agones-dev/agones/pull/758) ([markmandel](https://github.com/markmandel))
- Updated the filtering condition on GameServerShutdown to include the undeleted Unhealthy GSs [\#740](https://github.com/agones-dev/agones/pull/740) ([ilkercelikyilmaz](https://github.com/ilkercelikyilmaz))
- Add back goimports 🔥 [\#714](https://github.com/agones-dev/agones/pull/714) ([markmandel](https://github.com/markmandel))
- Add proto directory and update tooling. [\#709](https://github.com/agones-dev/agones/pull/709) ([heartrobotninja](https://github.com/heartrobotninja))
- Add explicit local version of agones in go.mod [\#706](https://github.com/agones-dev/agones/pull/706) ([aLekSer](https://github.com/aLekSer))
- Move GameServer to Unheathy when Pod Deleted [\#694](https://github.com/agones-dev/agones/pull/694) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- Agones e2e tests are flakey [\#700](https://github.com/agones-dev/agones/issues/700)
- Release 0.9.0 [\#686](https://github.com/agones-dev/agones/issues/686)
- Integration with Unreal Engine [\#138](https://github.com/agones-dev/agones/issues/138)

**Merged pull requests:**

- Release 0.10.0-rc [\#760](https://github.com/agones-dev/agones/pull/760) ([markmandel](https://github.com/markmandel))
- Add tests for gameServerCacheEntry in GameServerAllocation controller [\#754](https://github.com/agones-dev/agones/pull/754) ([markmandel](https://github.com/markmandel))
- Fix instructions to create AKS cluster [\#752](https://github.com/agones-dev/agones/pull/752) ([aLekSer](https://github.com/aLekSer))
- Deprecate Fleet Allocation. [\#750](https://github.com/agones-dev/agones/pull/750) ([markmandel](https://github.com/markmandel))
- Cleanup - no longer need to list Pods for GameServers [\#747](https://github.com/agones-dev/agones/pull/747) ([markmandel](https://github.com/markmandel))
- Convert C++ Example to Docker Build Pattern [\#746](https://github.com/agones-dev/agones/pull/746) ([markmandel](https://github.com/markmandel))
- Switch to parrallel execution of SDK commands [\#742](https://github.com/agones-dev/agones/pull/742) ([aLekSer](https://github.com/aLekSer))
- Move terraform targets into a separate file [\#741](https://github.com/agones-dev/agones/pull/741) ([aLekSer](https://github.com/aLekSer))
- We don't need CMake in the base build image [\#739](https://github.com/agones-dev/agones/pull/739) ([markmandel](https://github.com/markmandel))
- CI Speedup: Cache Build SDK between builds [\#738](https://github.com/agones-dev/agones/pull/738) ([markmandel](https://github.com/markmandel))
- Intial tool vendoring commit. [\#737](https://github.com/agones-dev/agones/pull/737) ([heartrobotninja](https://github.com/heartrobotninja))
- Add vendor\_fixed directory with apimachinery module [\#735](https://github.com/agones-dev/agones/pull/735) ([heartrobotninja](https://github.com/heartrobotninja))
- Option for CMake silent output [\#731](https://github.com/agones-dev/agones/pull/731) ([dsazonoff](https://github.com/dsazonoff))
- Cache htmltest url checks for 2 weeks [\#729](https://github.com/agones-dev/agones/pull/729) ([markmandel](https://github.com/markmandel))
- Fix for flaky TestGameServerAllocationMetaDataPatch [\#726](https://github.com/agones-dev/agones/pull/726) ([markmandel](https://github.com/markmandel))
- Adds a .clang-format file making Google style the default [\#724](https://github.com/agones-dev/agones/pull/724) ([devjgm](https://github.com/devjgm))
- Group make test output in cloudbuild.yaml [\#723](https://github.com/agones-dev/agones/pull/723) ([markmandel](https://github.com/markmandel))
- Upgrade Hugo to 0.55.2 [\#722](https://github.com/agones-dev/agones/pull/722) ([markmandel](https://github.com/markmandel))
- Remove dependency to util/runtime from allocation/v1alpha1/register.go [\#720](https://github.com/agones-dev/agones/pull/720) ([pooneh-m](https://github.com/pooneh-m))
- Clang-formatted the C++ SDK files. [\#716](https://github.com/agones-dev/agones/pull/716) ([devjgm](https://github.com/devjgm))
- Abstract build image ensuring and building [\#715](https://github.com/agones-dev/agones/pull/715) ([markmandel](https://github.com/markmandel))
- Mount go mod cache [\#712](https://github.com/agones-dev/agones/pull/712) ([markmandel](https://github.com/markmandel))
- Move local-includes above others [\#711](https://github.com/agones-dev/agones/pull/711) ([markmandel](https://github.com/markmandel))
- We no longer need gen-grpc-cpp.sh [\#710](https://github.com/agones-dev/agones/pull/710) ([markmandel](https://github.com/markmandel))
- Update issue templates [\#708](https://github.com/agones-dev/agones/pull/708) ([thisisnotapril](https://github.com/thisisnotapril))
- Change  the website theme  and add Ubisoft logo [\#704](https://github.com/agones-dev/agones/pull/704) ([cyriltovena](https://github.com/cyriltovena))
- Fixed typo in URL [\#702](https://github.com/agones-dev/agones/pull/702) ([devjgm](https://github.com/devjgm))
- Fixed a minor typo [\#701](https://github.com/agones-dev/agones/pull/701) ([pooneh-m](https://github.com/pooneh-m))
- Change License from Google Inc. to Google LLC due to branding change in 2015 [\#699](https://github.com/agones-dev/agones/pull/699) ([pooneh-m](https://github.com/pooneh-m))
- Remove dependency to util/runtime from APIS package [\#697](https://github.com/agones-dev/agones/pull/697) ([pooneh-m](https://github.com/pooneh-m))
- Update Linter to 1.16.0 [\#692](https://github.com/agones-dev/agones/pull/692) ([markmandel](https://github.com/markmandel))
- Choose specific release version of gen-crd-api-reference-docs [\#691](https://github.com/agones-dev/agones/pull/691) ([aLekSer](https://github.com/aLekSer))
- Upgrade Testify to 1.3.0 [\#689](https://github.com/agones-dev/agones/pull/689) ([markmandel](https://github.com/markmandel))
- Setup for release 0.10.0 [\#688](https://github.com/agones-dev/agones/pull/688) ([markmandel](https://github.com/markmandel))
- End to end tests for GameServer Pod Deletion [\#684](https://github.com/agones-dev/agones/pull/684) ([markmandel](https://github.com/markmandel))
- Removes the sdk generation tooling from our main build image [\#630](https://github.com/agones-dev/agones/pull/630) ([cyriltovena](https://github.com/cyriltovena))

## [v0.9.0](https://github.com/agones-dev/agones/tree/v0.9.0) (2019-04-03)

[Full Changelog](https://github.com/agones-dev/agones/compare/v0.9.0-rc...v0.9.0)

**Fixed bugs:**

- Url pointing to install.yaml pointing to master [\#676](https://github.com/agones-dev/agones/pull/676) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- Release 0.9.0-rc [\#673](https://github.com/agones-dev/agones/issues/673)
- Move to go modules [\#625](https://github.com/agones-dev/agones/issues/625)
- Documentation for the extended Kubernetes API [\#409](https://github.com/agones-dev/agones/issues/409)

**Merged pull requests:**

- Changes for 0.9.0 release! [\#687](https://github.com/agones-dev/agones/pull/687) ([markmandel](https://github.com/markmandel))
- Add the GDC presentation to the website [\#685](https://github.com/agones-dev/agones/pull/685) ([markmandel](https://github.com/markmandel))
- Forgot to highlight the breaking change. [\#680](https://github.com/agones-dev/agones/pull/680) ([markmandel](https://github.com/markmandel))
- Minor tweaks to documentation [\#677](https://github.com/agones-dev/agones/pull/677) ([markmandel](https://github.com/markmandel))
- Update do-release to fully build images [\#675](https://github.com/agones-dev/agones/pull/675) ([markmandel](https://github.com/markmandel))

## [v0.9.0-rc](https://github.com/agones-dev/agones/tree/v0.9.0-rc) (2019-03-26)

[Full Changelog](https://github.com/agones-dev/agones/compare/v0.8.1...v0.9.0-rc)

**Breaking changes:**

- Consistency: Portpolicy static=\>Static & dynamic=\>Dynamic [\#617](https://github.com/agones-dev/agones/pull/617) ([markmandel](https://github.com/markmandel))

**Implemented enhancements:**

- Adding a section in the documentation about deploying Agones using GCP Marketplace. [\#664](https://github.com/agones-dev/agones/pull/664) ([bbf](https://github.com/bbf))
- Add Agones Kubernetes API docs generator [\#645](https://github.com/agones-dev/agones/pull/645) ([aLekSer](https://github.com/aLekSer))
- Added support for persisting logs in 'emptyDir' volume attached to agones controller. [\#620](https://github.com/agones-dev/agones/pull/620) ([jkowalski](https://github.com/jkowalski))
- Adding Locust tests - initial changes for \#412 [\#611](https://github.com/agones-dev/agones/pull/611) ([pm7h](https://github.com/pm7h))
- Emit stress test metrics in Fortio format. [\#586](https://github.com/agones-dev/agones/pull/586) ([jkowalski](https://github.com/jkowalski))
- Add Node.js SDK and example - closes \#538 [\#581](https://github.com/agones-dev/agones/pull/581) ([steven-supersolid](https://github.com/steven-supersolid))
- Cpp sdk cmake [\#464](https://github.com/agones-dev/agones/pull/464) ([dsazonoff](https://github.com/dsazonoff))

**Fixed bugs:**

- Feature shortcode does not behave correctly for versions \> "0.10.0" \(2 digit minor version\) [\#650](https://github.com/agones-dev/agones/issues/650)
- Labels referencing resources name can be too long [\#541](https://github.com/agones-dev/agones/issues/541)
- Fix feature shortcode for Hugo [\#655](https://github.com/agones-dev/agones/pull/655) ([aLekSer](https://github.com/aLekSer))
- \[Regression\] Fleet scale down didn't adhere to Packed Scheduling [\#638](https://github.com/agones-dev/agones/pull/638) ([markmandel](https://github.com/markmandel))
- Fixed gameserverset overshooting the number of GameServers [\#621](https://github.com/agones-dev/agones/pull/621) ([jkowalski](https://github.com/jkowalski))
- Update GameServerSet scheduling when Fleet scheduling is changed. [\#582](https://github.com/agones-dev/agones/pull/582) ([pooneh-m](https://github.com/pooneh-m))

**Security fixes:**

- Remove serviceaccount for game server container [\#640](https://github.com/agones-dev/agones/pull/640) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- gcloud-auth-cluster: Create unique name for cluster role binding [\#662](https://github.com/agones-dev/agones/issues/662)
- Hotfix 0.8.1 [\#652](https://github.com/agones-dev/agones/issues/652)
- Slow game servers deletion [\#540](https://github.com/agones-dev/agones/issues/540)

**Merged pull requests:**

- Release 0.9.0-rc [\#674](https://github.com/agones-dev/agones/pull/674) ([markmandel](https://github.com/markmandel))
- Fix typo on metrics documentation [\#671](https://github.com/agones-dev/agones/pull/671) ([fireflyfenix](https://github.com/fireflyfenix))
- Moved Azure link, breaking builds. [\#669](https://github.com/agones-dev/agones/pull/669) ([markmandel](https://github.com/markmandel))
- Add hash of account to clusterrolebinding name to support multiple accounts [\#666](https://github.com/agones-dev/agones/pull/666) ([aLekSer](https://github.com/aLekSer))
- Simplify homepage messaging [\#665](https://github.com/agones-dev/agones/pull/665) ([markmandel](https://github.com/markmandel))
- GKE installation n1-standard-1 -\> n1-standard-2 [\#663](https://github.com/agones-dev/agones/pull/663) ([markmandel](https://github.com/markmandel))
- Switch Agones to Go Modules [\#661](https://github.com/agones-dev/agones/pull/661) ([heartrobotninja](https://github.com/heartrobotninja))
-  Merge hotfix 0.8.1 back into master [\#658](https://github.com/agones-dev/agones/pull/658) ([markmandel](https://github.com/markmandel))
- Cleanup Xonotic image [\#654](https://github.com/agones-dev/agones/pull/654) ([markmandel](https://github.com/markmandel))
- E2E Cleanup: Implement SendGameServerUDP [\#644](https://github.com/agones-dev/agones/pull/644) ([markmandel](https://github.com/markmandel))
- Refactor https server into its own component [\#643](https://github.com/agones-dev/agones/pull/643) ([markmandel](https://github.com/markmandel))
- Add .gocache directory for WSL users [\#642](https://github.com/agones-dev/agones/pull/642) ([heartrobotninja](https://github.com/heartrobotninja))
- E2E test for Disabled Health checks. [\#641](https://github.com/agones-dev/agones/pull/641) ([markmandel](https://github.com/markmandel))
- Refactor AllocationCounter to GameServerCounter [\#639](https://github.com/agones-dev/agones/pull/639) ([markmandel](https://github.com/markmandel))
- Adding Kubernetes API server requests metrics [\#637](https://github.com/agones-dev/agones/pull/637) ([aLekSer](https://github.com/aLekSer))
- Partial revert "Emit stress test metrics in Fortio format." which accidentally overwrote our vendored fixes to wait.go [\#633](https://github.com/agones-dev/agones/pull/633) ([jkowalski](https://github.com/jkowalski))
- Switch to using default gke-cluster oauthScopes settings for clusters [\#632](https://github.com/agones-dev/agones/pull/632) ([aLekSer](https://github.com/aLekSer))
- Update docs Create Gameserver with current state [\#627](https://github.com/agones-dev/agones/pull/627) ([aLekSer](https://github.com/aLekSer))
- New logo for the website! [\#618](https://github.com/agones-dev/agones/pull/618) ([markmandel](https://github.com/markmandel))
- Unified logging of resource identifiers so that we can reliably get entire history of a resource in stack driver. [\#616](https://github.com/agones-dev/agones/pull/616) ([jkowalski](https://github.com/jkowalski))
- Organising Makefile into includes [\#615](https://github.com/agones-dev/agones/pull/615) ([markmandel](https://github.com/markmandel))
- Upgraed go-lint tooling. [\#612](https://github.com/agones-dev/agones/pull/612) ([markmandel](https://github.com/markmandel))
- Moving to 0.9.0! [\#610](https://github.com/agones-dev/agones/pull/610) ([markmandel](https://github.com/markmandel))
- Adding resources limits for E2E tests gameserver Spec [\#606](https://github.com/agones-dev/agones/pull/606) ([aLekSer](https://github.com/aLekSer))
- Add Fleet and Gameserverset Validation handler [\#598](https://github.com/agones-dev/agones/pull/598) ([aLekSer](https://github.com/aLekSer))
- Improve allocation performance [\#583](https://github.com/agones-dev/agones/pull/583) ([ilkercelikyilmaz](https://github.com/ilkercelikyilmaz))
- Add status subresource to fleet and Gameserverset [\#575](https://github.com/agones-dev/agones/pull/575) ([aLekSer](https://github.com/aLekSer))

## [v0.8.1](https://github.com/agones-dev/agones/tree/v0.8.1) (2019-03-15)

[Full Changelog](https://github.com/agones-dev/agones/compare/v0.8.0...v0.8.1)

**Implemented enhancements:**

- Create Node.js library [\#538](https://github.com/agones-dev/agones/issues/538)

**Fixed bugs:**

- CPU/Memory leak issue caused by go routines that never completes [\#636](https://github.com/agones-dev/agones/issues/636)
- Quickstart: Create a Game Server [\#609](https://github.com/agones-dev/agones/issues/609)
- Fleet status completely out-of-sync with GameServerSet status [\#570](https://github.com/agones-dev/agones/issues/570)
- GameServerSet sometimes creates more GameServers than necessary [\#569](https://github.com/agones-dev/agones/issues/569)
- If you modify the `Scheduling` on a Fleet, it does not flow down to the `GameServerSet`. [\#495](https://github.com/agones-dev/agones/issues/495)
- SDK Service Account was Hardcoded [\#629](https://github.com/agones-dev/agones/pull/629) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- GKE scopes in installation and testing are overkill [\#614](https://github.com/agones-dev/agones/issues/614)
- Release 0.8.0 [\#604](https://github.com/agones-dev/agones/issues/604)
- Grafana: add basic API Server graphs [\#546](https://github.com/agones-dev/agones/issues/546)
- Remove all the kubectl custom commands from the quickstarts [\#521](https://github.com/agones-dev/agones/issues/521)

**Merged pull requests:**

- Final release pieces for 0.8.1 hotfix. [\#653](https://github.com/agones-dev/agones/pull/653) ([markmandel](https://github.com/markmandel))
- Tarballing source into the images for dependencies that are required by their licenses. [\#634](https://github.com/agones-dev/agones/pull/634) ([bbf](https://github.com/bbf))
- 2 Hotfixes: Allow Helm to reference image digests and inject licenses [\#631](https://github.com/agones-dev/agones/pull/631) ([bbf](https://github.com/bbf))
- \[Hotfix\] Prep work for hotfix 0.8.1 [\#628](https://github.com/agones-dev/agones/pull/628) ([markmandel](https://github.com/markmandel))
- Add input parameters check on CRD loggers [\#626](https://github.com/agones-dev/agones/pull/626) ([aLekSer](https://github.com/aLekSer))

## [v0.8.0](https://github.com/agones-dev/agones/tree/v0.8.0) (2019-02-20)

[Full Changelog](https://github.com/agones-dev/agones/compare/v0.8.0-rc...v0.8.0)

**Implemented enhancements:**

- Register GameServers with local IP addresses [\#469](https://github.com/agones-dev/agones/issues/469)

**Fixed bugs:**

- agonessdk-0.8.0-\*-runtime-linux-arch\_64.tar.gz is growing unboundedly [\#589](https://github.com/agones-dev/agones/issues/589)
- Create a boolean to gate the creation of priority classes for controllers. [\#602](https://github.com/agones-dev/agones/pull/602) ([bbf](https://github.com/bbf))
- Exclude tar.gz and zip files from Runtime archive [\#596](https://github.com/agones-dev/agones/pull/596) ([aLekSer](https://github.com/aLekSer))
- Switch to htmltest link checker -- and fix issues [\#594](https://github.com/agones-dev/agones/pull/594) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- Release 0.8.0-rc [\#590](https://github.com/agones-dev/agones/issues/590)
- Help us pick a new project logo!  [\#577](https://github.com/agones-dev/agones/issues/577)

**Merged pull requests:**

- Release 0.8.0 [\#605](https://github.com/agones-dev/agones/pull/605) ([markmandel](https://github.com/markmandel))
- Remove deprecation from FleetAllocation [\#603](https://github.com/agones-dev/agones/pull/603) ([markmandel](https://github.com/markmandel))
- Remove -v from Go testing - becomes too noisy [\#595](https://github.com/agones-dev/agones/pull/595) ([markmandel](https://github.com/markmandel))
- Minor tweaks to release process. [\#592](https://github.com/agones-dev/agones/pull/592) ([markmandel](https://github.com/markmandel))

## [v0.8.0-rc](https://github.com/agones-dev/agones/tree/v0.8.0-rc) (2019-02-14)

[Full Changelog](https://github.com/agones-dev/agones/compare/v0.7.0...v0.8.0-rc)

**Implemented enhancements:**

- Allocation on GameServers rather than Fleets [\#436](https://github.com/agones-dev/agones/issues/436)
- Website that contains documentation [\#410](https://github.com/agones-dev/agones/issues/410)
- Node Affinity, Tolerations and Node selector support for helm chart [\#407](https://github.com/agones-dev/agones/issues/407)
- As game server, I want to get the Agones sidecar version [\#310](https://github.com/agones-dev/agones/issues/310)
- fix setAnnotation for simple-udp to use customized key & value [\#576](https://github.com/agones-dev/agones/pull/576) ([Yingxin-Jiang](https://github.com/Yingxin-Jiang))
- Adding Github link and version dropdown to the navigation bar [\#566](https://github.com/agones-dev/agones/pull/566) ([markmandel](https://github.com/markmandel))
- simple-udp: added support for customizing labels and annotations by the caller [\#564](https://github.com/agones-dev/agones/pull/564) ([jkowalski](https://github.com/jkowalski))
- Monitoring improvements [\#559](https://github.com/agones-dev/agones/pull/559) ([jkowalski](https://github.com/jkowalski))
- Add support to create a development gameserver. [\#558](https://github.com/agones-dev/agones/pull/558) ([jeremyje](https://github.com/jeremyje))
- Adds gameservers per node count and distribution [\#551](https://github.com/agones-dev/agones/pull/551) ([cyriltovena](https://github.com/cyriltovena))
- Add Scale Subresource into Fleet and Gameserverset CRDs [\#539](https://github.com/agones-dev/agones/pull/539) ([aLekSer](https://github.com/aLekSer))
- Continuous Deployment of Agones.dev [\#527](https://github.com/agones-dev/agones/pull/527) ([markmandel](https://github.com/markmandel))
- Makefile: allowed 'go test' to run without docker and optionally w/o race detector [\#509](https://github.com/agones-dev/agones/pull/509) ([jkowalski](https://github.com/jkowalski))
- add client-go metrics and grafana dashboards [\#505](https://github.com/agones-dev/agones/pull/505) ([cyriltovena](https://github.com/cyriltovena))
- Prometheus and grafana improvements based on load testing experience [\#501](https://github.com/agones-dev/agones/pull/501) ([jkowalski](https://github.com/jkowalski))
- improved isolation of Agones controllers using taints and priority [\#500](https://github.com/agones-dev/agones/pull/500) ([jkowalski](https://github.com/jkowalski))
- Add Agones version into Gameserver Annotation [\#498](https://github.com/agones-dev/agones/pull/498) ([aLekSer](https://github.com/aLekSer))
- controller: made QPS, burst QPS and number of workers externally configurable [\#497](https://github.com/agones-dev/agones/pull/497) ([jkowalski](https://github.com/jkowalski))
- Website for Agones [\#493](https://github.com/agones-dev/agones/pull/493) ([markmandel](https://github.com/markmandel))
- Add Stackdriver Exporter for Opencensus [\#492](https://github.com/agones-dev/agones/pull/492) ([aLekSer](https://github.com/aLekSer))
- Add TLS to Fleetautoscaler webhook service [\#476](https://github.com/agones-dev/agones/pull/476) ([aLekSer](https://github.com/aLekSer))
- Add pod tolerations, nodeSelector and affinity in helm [\#473](https://github.com/agones-dev/agones/pull/473) ([cyriltovena](https://github.com/cyriltovena))
- adding Prometheus+Grafana for metrics and visualizations [\#472](https://github.com/agones-dev/agones/pull/472) ([cyriltovena](https://github.com/cyriltovena))
- GameServerAllocation implementation [\#465](https://github.com/agones-dev/agones/pull/465) ([markmandel](https://github.com/markmandel))

**Fixed bugs:**

- Gameserver's that are not assigned to a node are left behind even after the scale was lowered [\#543](https://github.com/agones-dev/agones/issues/543)
- Investigate why increasing worker count/QPS causes E2E tests to fail [\#499](https://github.com/agones-dev/agones/issues/499)
- Investigate why we sometimes have multiple pods per gameserver [\#490](https://github.com/agones-dev/agones/issues/490)
- Assign higher priority to Agones system pods [\#489](https://github.com/agones-dev/agones/issues/489)
- e2e tests don't cleanup fleetautoscalers [\#471](https://github.com/agones-dev/agones/issues/471)
- Race condition in SDK.SetLabel and SDK.SetAnnotation [\#455](https://github.com/agones-dev/agones/issues/455)
- sdkserver: fix race condition in SDK.SetLabel and SDK.SetAnnotation \(issue \#455\) [\#588](https://github.com/agones-dev/agones/pull/588) ([Yingxin-Jiang](https://github.com/Yingxin-Jiang))
- Changed how GameServer POD names are generated [\#565](https://github.com/agones-dev/agones/pull/565) ([jkowalski](https://github.com/jkowalski))
- Fix stackdriver distribution without bucket bounds [\#554](https://github.com/agones-dev/agones/pull/554) ([aLekSer](https://github.com/aLekSer))
- Fix potential data race in allocation counter [\#525](https://github.com/agones-dev/agones/pull/525) ([markmandel](https://github.com/markmandel))
- Fix concurrency bug in port allocator. [\#514](https://github.com/agones-dev/agones/pull/514) ([markmandel](https://github.com/markmandel))
- Go download link has changed [\#494](https://github.com/agones-dev/agones/pull/494) ([markmandel](https://github.com/markmandel))
- Fix for the controller panic issue on metrics.enabled is false [\#486](https://github.com/agones-dev/agones/pull/486) ([aLekSer](https://github.com/aLekSer))

**Security fixes:**

- \[SECURITY\] Update Go to 1.11.5 [\#528](https://github.com/agones-dev/agones/pull/528) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- Grafana: add graph of nodes in cluster [\#547](https://github.com/agones-dev/agones/issues/547)
- Replace global allocation mutex with fine-grained concurrency controls. [\#535](https://github.com/agones-dev/agones/issues/535)
- Approver access for @jkowalski [\#526](https://github.com/agones-dev/agones/issues/526)
- Docker images layers not optimal [\#481](https://github.com/agones-dev/agones/issues/481)
- Release 0.7.0 [\#477](https://github.com/agones-dev/agones/issues/477)
- Improve build speed by refactoring Makefile [\#453](https://github.com/agones-dev/agones/issues/453)

**Merged pull requests:**

- Release 0.8.0-rc [\#591](https://github.com/agones-dev/agones/pull/591) ([markmandel](https://github.com/markmandel))
- typo [\#587](https://github.com/agones-dev/agones/pull/587) ([jkowalski](https://github.com/jkowalski))
- test: make e2e test logs more readable [\#585](https://github.com/agones-dev/agones/pull/585) ([jkowalski](https://github.com/jkowalski))
- Update godoc command to enable search [\#580](https://github.com/agones-dev/agones/pull/580) ([markmandel](https://github.com/markmandel))
- Removal of allocationMutex from controllers that don't need it. [\#579](https://github.com/agones-dev/agones/pull/579) ([markmandel](https://github.com/markmandel))
- Remove the mutex usage for Delete GS in both GS and GSS controllers [\#572](https://github.com/agones-dev/agones/pull/572) ([ilkercelikyilmaz](https://github.com/ilkercelikyilmaz))
- Added very simple stress test which scales fleets up/down and basic stress test harness [\#571](https://github.com/agones-dev/agones/pull/571) ([jkowalski](https://github.com/jkowalski))
- Fix of TestWorkQueueHealthCheck test [\#568](https://github.com/agones-dev/agones/pull/568) ([aLekSer](https://github.com/aLekSer))
- bump default qps to 400 w/burst to 500 and worker count to 100 [\#563](https://github.com/agones-dev/agones/pull/563) ([jkowalski](https://github.com/jkowalski))
- added fleet-loadtest.yaml for use in load testing [\#562](https://github.com/agones-dev/agones/pull/562) ([jkowalski](https://github.com/jkowalski))
- Fix prometheous installation on minikube [\#561](https://github.com/agones-dev/agones/pull/561) ([markmandel](https://github.com/markmandel))
- CloudBuild for a "development" subdomain [\#560](https://github.com/agones-dev/agones/pull/560) ([markmandel](https://github.com/markmandel))
- Remove the custom kubectl commands from quickstarts [\#556](https://github.com/agones-dev/agones/pull/556) ([hpandeycodeit](https://github.com/hpandeycodeit))
- e2e: fixed test-only race condition in TestAutoscalerBasicFunctions [\#552](https://github.com/agones-dev/agones/pull/552) ([jkowalski](https://github.com/jkowalski))
- e2e: improved logging and simplified waiting for fleet conditions [\#550](https://github.com/agones-dev/agones/pull/550) ([jkowalski](https://github.com/jkowalski))
- Typo: Docsy -\> Agones Blog. [\#549](https://github.com/agones-dev/agones/pull/549) ([markmandel](https://github.com/markmandel))
- GameServer Creation, Allocation and Shutdown Lifecycle [\#548](https://github.com/agones-dev/agones/pull/548) ([markmandel](https://github.com/markmandel))
- Changed kubeInformation to kubeInformer. [\#545](https://github.com/agones-dev/agones/pull/545) ([pooneh-m](https://github.com/pooneh-m))
- Changed kubeInformation to kubeInformer. [\#544](https://github.com/agones-dev/agones/pull/544) ([pooneh-m](https://github.com/pooneh-m))
- Speed up creation/deletion of game servers in a set [\#542](https://github.com/agones-dev/agones/pull/542) ([jkowalski](https://github.com/jkowalski))
- Adding tags to cloudbuilds [\#537](https://github.com/agones-dev/agones/pull/537) ([markmandel](https://github.com/markmandel))
- This is how you write shortcode in hugo [\#534](https://github.com/agones-dev/agones/pull/534) ([markmandel](https://github.com/markmandel))
- Add 2 new flags to control the Helm installation: [\#533](https://github.com/agones-dev/agones/pull/533) ([bbf](https://github.com/bbf))
- PortAllocator.Run\(\) is no longer blocking. [\#531](https://github.com/agones-dev/agones/pull/531) ([markmandel](https://github.com/markmandel))
- Move SDK local tooling into its own section [\#529](https://github.com/agones-dev/agones/pull/529) ([markmandel](https://github.com/markmandel))
- Put CI buiild logs in a public bucket. [\#524](https://github.com/agones-dev/agones/pull/524) ([markmandel](https://github.com/markmandel))
- fixed go\_build\_base\_path for LOCAL\_GO [\#523](https://github.com/agones-dev/agones/pull/523) ([jkowalski](https://github.com/jkowalski))
- Test using gcloud base for e2e works. [\#522](https://github.com/agones-dev/agones/pull/522) ([markmandel](https://github.com/markmandel))
- fixed gcloud-test-cluster setup problem caused by bad merge between \#500 and \#501 [\#520](https://github.com/agones-dev/agones/pull/520) ([jkowalski](https://github.com/jkowalski))
- Remove TOC from metrics page [\#519](https://github.com/agones-dev/agones/pull/519) ([markmandel](https://github.com/markmandel))
- Extend consul -try to 30m [\#518](https://github.com/agones-dev/agones/pull/518) ([markmandel](https://github.com/markmandel))
- fixes kind prometheus installation [\#517](https://github.com/agones-dev/agones/pull/517) ([cyriltovena](https://github.com/cyriltovena))
- Fix for flaky TestSDKSetAnnotation [\#516](https://github.com/agones-dev/agones/pull/516) ([markmandel](https://github.com/markmandel))
- minkube-setup-grafana =\> minikube-setup-grafana [\#515](https://github.com/agones-dev/agones/pull/515) ([markmandel](https://github.com/markmandel))
- Restructure cloudbuild.yaml [\#513](https://github.com/agones-dev/agones/pull/513) ([markmandel](https://github.com/markmandel))
- e2e: run cleanup before tests in addition to after [\#512](https://github.com/agones-dev/agones/pull/512) ([jkowalski](https://github.com/jkowalski))
- Prometheus installation docs tweak. [\#510](https://github.com/agones-dev/agones/pull/510) ([markmandel](https://github.com/markmandel))
- Add e2e test for updating gameserver configurations in fleet [\#508](https://github.com/agones-dev/agones/pull/508) ([Yingxin-Jiang](https://github.com/Yingxin-Jiang))
- Extend e2e lock to 30m [\#507](https://github.com/agones-dev/agones/pull/507) ([markmandel](https://github.com/markmandel))
- Speed up builds by using local go/zip instead of dockerized ones. [\#506](https://github.com/agones-dev/agones/pull/506) ([jkowalski](https://github.com/jkowalski))
- Fixes for flaky e2e tests. [\#504](https://github.com/agones-dev/agones/pull/504) ([markmandel](https://github.com/markmandel))
- Fix for Flaky TestControllerCreationMutationHandler [\#503](https://github.com/agones-dev/agones/pull/503) ([markmandel](https://github.com/markmandel))
- fixed e2e tests by using generated object names [\#502](https://github.com/agones-dev/agones/pull/502) ([jkowalski](https://github.com/jkowalski))
- added resource limits to gameserver.yaml and changed to generateName: [\#496](https://github.com/agones-dev/agones/pull/496) ([jkowalski](https://github.com/jkowalski))
- Remove reflect from controller. [\#488](https://github.com/agones-dev/agones/pull/488) ([markmandel](https://github.com/markmandel))
- specify resource limits on simple-udp/fleet.yaml [\#487](https://github.com/agones-dev/agones/pull/487) ([jkowalski](https://github.com/jkowalski))
- improve docker layers using COPY --chown [\#482](https://github.com/agones-dev/agones/pull/482) ([cyriltovena](https://github.com/cyriltovena))
- Update fleet\_spec.md [\#480](https://github.com/agones-dev/agones/pull/480) ([pm7h](https://github.com/pm7h))
- Post 0.7.0 changes [\#479](https://github.com/agones-dev/agones/pull/479) ([markmandel](https://github.com/markmandel))
- docs: added game server state diagram [\#475](https://github.com/agones-dev/agones/pull/475) ([jkowalski](https://github.com/jkowalski))
- fix autoscaler cleanup on tests failure [\#474](https://github.com/agones-dev/agones/pull/474) ([cyriltovena](https://github.com/cyriltovena))

## [v0.7.0](https://github.com/agones-dev/agones/tree/v0.7.0) (2019-01-08)

[Full Changelog](https://github.com/agones-dev/agones/compare/v0.7.0-rc...v0.7.0)

**Closed issues:**

- Release 0.7.0-rc [\#467](https://github.com/agones-dev/agones/issues/467)

**Merged pull requests:**

- Release 0.7.0 [\#478](https://github.com/agones-dev/agones/pull/478) ([markmandel](https://github.com/markmandel))
- Preparation for 0.7.0 [\#470](https://github.com/agones-dev/agones/pull/470) ([markmandel](https://github.com/markmandel))

## [v0.7.0-rc](https://github.com/agones-dev/agones/tree/v0.7.0-rc) (2019-01-02)

[Full Changelog](https://github.com/agones-dev/agones/compare/v0.6.0...v0.7.0-rc)

**Breaking changes:**

- Update to Kubernetes 1.11 [\#447](https://github.com/agones-dev/agones/pull/447) ([markmandel](https://github.com/markmandel))
- Remove crd-install hook, as it break CRD updates [\#441](https://github.com/agones-dev/agones/pull/441) ([markmandel](https://github.com/markmandel))

**Implemented enhancements:**

- Delete crds, and fleets, gameservers etc on deletion of Helm chart [\#426](https://github.com/agones-dev/agones/issues/426)
- `GameServers` should have the Fleet name in a label for easy retrieval [\#411](https://github.com/agones-dev/agones/issues/411)
- Horizontal Fleet Autoscaling [\#334](https://github.com/agones-dev/agones/issues/334)
- Add webhook functionality into FleetAutoscaler [\#460](https://github.com/agones-dev/agones/pull/460) ([aLekSer](https://github.com/aLekSer))
- Adds Kind local cluster support with documentation [\#458](https://github.com/agones-dev/agones/pull/458) ([cyriltovena](https://github.com/cyriltovena))
- Adds OpenCensus metrics integration. [\#457](https://github.com/agones-dev/agones/pull/457) ([cyriltovena](https://github.com/cyriltovena))
- added incremental build option to Makefile to speed up rebuilds [\#454](https://github.com/agones-dev/agones/pull/454) ([jkowalski](https://github.com/jkowalski))
- CRD: added additionalPrinterColumns to GameServer for kubectl [\#444](https://github.com/agones-dev/agones/pull/444) ([jkowalski](https://github.com/jkowalski))
- Adding explicit length of git revision in Makefile and E2E Can't Allocate test  [\#440](https://github.com/agones-dev/agones/pull/440) ([aLekSer](https://github.com/aLekSer))
- Pinger service for Multiple Cluster Latency Measurement. [\#434](https://github.com/agones-dev/agones/pull/434) ([markmandel](https://github.com/markmandel))

**Fixed bugs:**

- This should fail e2e in any command fails [\#462](https://github.com/agones-dev/agones/pull/462) ([markmandel](https://github.com/markmandel))
- Apply fix for goroutines leak [\#461](https://github.com/agones-dev/agones/pull/461) ([aLekSer](https://github.com/aLekSer))
- GameServerSets: DeleteFunc could receive a DeletedFinalStateUnknown [\#442](https://github.com/agones-dev/agones/pull/442) ([markmandel](https://github.com/markmandel))

**Security fixes:**

- \[Security\] Upgrade Go to 1.11.4 [\#446](https://github.com/agones-dev/agones/pull/446) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- Controller logging consistency  [\#456](https://github.com/agones-dev/agones/issues/456)
- Add Agones to helm hub [\#450](https://github.com/agones-dev/agones/issues/450)
- Add support for Kind cluster [\#448](https://github.com/agones-dev/agones/issues/448)
- Move SDK server code from pkg/gameservers to a separate package [\#445](https://github.com/agones-dev/agones/issues/445)
- Helm chart for 0.6.0 do not work on Helm v2.9.1 due crd-install hook [\#431](https://github.com/agones-dev/agones/issues/431)
- Release 0.6.0 [\#428](https://github.com/agones-dev/agones/issues/428)

**Merged pull requests:**

- Release 0.7.0-rc [\#468](https://github.com/agones-dev/agones/pull/468) ([markmandel](https://github.com/markmandel))
- Move the README.md into /agones/ so it's in the Helm Chart [\#466](https://github.com/agones-dev/agones/pull/466) ([markmandel](https://github.com/markmandel))
- Convert to using Fluentdformatter [\#463](https://github.com/agones-dev/agones/pull/463) ([markmandel](https://github.com/markmandel))
- Upgrade minikube to K8s 1.11 [\#459](https://github.com/agones-dev/agones/pull/459) ([markmandel](https://github.com/markmandel))
- pkg/sdkserver: added doc.go [\#452](https://github.com/agones-dev/agones/pull/452) ([jkowalski](https://github.com/jkowalski))
- Split pkg/gameservers into two  [\#451](https://github.com/agones-dev/agones/pull/451) ([jkowalski](https://github.com/jkowalski))
- Copy/paste fix [\#449](https://github.com/agones-dev/agones/pull/449) ([joeholley](https://github.com/joeholley))
- Delete crds, and fleets, gameservers etc on deletion of Helm chart [\#437](https://github.com/agones-dev/agones/pull/437) ([EricFortin](https://github.com/EricFortin))
- Update gRPC to v1.16.1 [\#435](https://github.com/agones-dev/agones/pull/435) ([markmandel](https://github.com/markmandel))
- adds minimun tiller version in chart and update doc [\#433](https://github.com/agones-dev/agones/pull/433) ([cyriltovena](https://github.com/cyriltovena))
- README: Autoscaler example link [\#432](https://github.com/agones-dev/agones/pull/432) ([markmandel](https://github.com/markmandel))
- Post 0.6.0 updates [\#430](https://github.com/agones-dev/agones/pull/430) ([markmandel](https://github.com/markmandel))
- add fleet name to gameservers labels [\#427](https://github.com/agones-dev/agones/pull/427) ([cyriltovena](https://github.com/cyriltovena))

## [v0.6.0](https://github.com/agones-dev/agones/tree/v0.6.0) (2018-11-28)

[Full Changelog](https://github.com/agones-dev/agones/compare/v0.6.0-rc...v0.6.0)

**Closed issues:**

- Release 0.6.0.rc [\#424](https://github.com/agones-dev/agones/issues/424)

**Merged pull requests:**

- Release 0.6.0 updates. [\#429](https://github.com/agones-dev/agones/pull/429) ([markmandel](https://github.com/markmandel))

## [v0.6.0-rc](https://github.com/agones-dev/agones/tree/v0.6.0-rc) (2018-11-21)

[Full Changelog](https://github.com/agones-dev/agones/compare/v0.5.0...v0.6.0-rc)

**Implemented enhancements:**

- Using the Cluster Autoscaler with Agones [\#368](https://github.com/agones-dev/agones/issues/368)
- Agones sdk-server sidecar should have cpu and memory limits [\#344](https://github.com/agones-dev/agones/issues/344)
- As developer, I want to emulate an allocation in local mode [\#314](https://github.com/agones-dev/agones/issues/314)
- Document how to configure maximum number of pods/node that can be allocated [\#295](https://github.com/agones-dev/agones/issues/295)
- Development tools to enable pprof [\#422](https://github.com/agones-dev/agones/pull/422) ([markmandel](https://github.com/markmandel))
- Changes to the GameServer configuration are reflected in the local sdk server [\#413](https://github.com/agones-dev/agones/pull/413) ([markmandel](https://github.com/markmandel))
- Mark GameServer Unhealthy if allocated HostPort isn't available [\#408](https://github.com/agones-dev/agones/pull/408) ([markmandel](https://github.com/markmandel))
- Cluster Autoscaling: safe-to-evict=false annotations for GameServer Pods [\#405](https://github.com/agones-dev/agones/pull/405) ([markmandel](https://github.com/markmandel))
- Packed: Fleet scaled down removes GameServers from least used Nodes [\#401](https://github.com/agones-dev/agones/pull/401) ([markmandel](https://github.com/markmandel))
- Packed: PreferredDuringSchedulingIgnoredDuringExecution PodAffinity with a HostName topology [\#397](https://github.com/agones-dev/agones/pull/397) ([markmandel](https://github.com/markmandel))
- Specify CPU Request for the SDK Server Sidecar [\#390](https://github.com/agones-dev/agones/pull/390) ([markmandel](https://github.com/markmandel))
- Mount point for helm config [\#383](https://github.com/agones-dev/agones/pull/383) ([markmandel](https://github.com/markmandel))
- Add crd-install helm hook to crds templates [\#375](https://github.com/agones-dev/agones/pull/375) ([smoya](https://github.com/smoya))

**Fixed bugs:**

- Admission webhook "mutations.stable.agones.dev" errors with Invalid FleetAutoscaler [\#406](https://github.com/agones-dev/agones/issues/406)
- Ports should always be allocated to a GameServer [\#415](https://github.com/agones-dev/agones/pull/415) ([markmandel](https://github.com/markmandel))
- Apparently patching events is a thing. [\#402](https://github.com/agones-dev/agones/pull/402) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- Release 0.5.0 [\#387](https://github.com/agones-dev/agones/issues/387)

**Merged pull requests:**

- Release 0.6.0-rc [\#425](https://github.com/agones-dev/agones/pull/425) ([markmandel](https://github.com/markmandel))
- More stringent linting rules \(and update linter\) [\#417](https://github.com/agones-dev/agones/pull/417) ([markmandel](https://github.com/markmandel))
- FleetAutoscaler can be targeted at Non Existent Fleets [\#416](https://github.com/agones-dev/agones/pull/416) ([markmandel](https://github.com/markmandel))
- Adding colour to the linter, because colours are pretty. [\#400](https://github.com/agones-dev/agones/pull/400) ([markmandel](https://github.com/markmandel))
- Process to become an reviewer/approver on Agones. [\#399](https://github.com/agones-dev/agones/pull/399) ([markmandel](https://github.com/markmandel))
- Update Helm to 2.11.0 [\#396](https://github.com/agones-dev/agones/pull/396) ([markmandel](https://github.com/markmandel))
- Make sure do-release always uses the release\_registry [\#394](https://github.com/agones-dev/agones/pull/394) ([markmandel](https://github.com/markmandel))
- Adding third part videos and presentations. [\#393](https://github.com/agones-dev/agones/pull/393) ([markmandel](https://github.com/markmandel))
- TOC for the SDK integration and tooling [\#392](https://github.com/agones-dev/agones/pull/392) ([markmandel](https://github.com/markmandel))
- Set test clusters to base version. GKE will work out the rest. [\#391](https://github.com/agones-dev/agones/pull/391) ([markmandel](https://github.com/markmandel))
- Post 0.5.0 Updates [\#389](https://github.com/agones-dev/agones/pull/389) ([markmandel](https://github.com/markmandel))
- Update to Go 1.11.1 [\#385](https://github.com/agones-dev/agones/pull/385) ([markmandel](https://github.com/markmandel))

## [v0.5.0](https://github.com/agones-dev/agones/tree/v0.5.0) (2018-10-16)

[Full Changelog](https://github.com/agones-dev/agones/compare/v0.5.0-rc...v0.5.0)

**Fixed bugs:**

- Tutorial fails @ Step 5 due to RBAC issues if you have capital letters in your gcloud account name [\#282](https://github.com/agones-dev/agones/issues/282)

**Closed issues:**

- Release 0.5.0.rc [\#378](https://github.com/agones-dev/agones/issues/378)

**Merged pull requests:**

- Change for the 0.5.0 release. [\#388](https://github.com/agones-dev/agones/pull/388) ([markmandel](https://github.com/markmandel))
- Troubleshooting guide for issues with Agones. [\#384](https://github.com/agones-dev/agones/pull/384) ([markmandel](https://github.com/markmandel))
- Spec docs for FleetAutoscaler [\#381](https://github.com/agones-dev/agones/pull/381) ([markmandel](https://github.com/markmandel))
- Post 0.5.0-rc updates [\#380](https://github.com/agones-dev/agones/pull/380) ([markmandel](https://github.com/markmandel))

## [v0.5.0-rc](https://github.com/agones-dev/agones/tree/v0.5.0-rc) (2018-10-09)

[Full Changelog](https://github.com/agones-dev/agones/compare/v0.4.0...v0.5.0-rc)

**Implemented enhancements:**

- Improve support for developing in custom environments [\#348](https://github.com/agones-dev/agones/issues/348)
- Agones helm repo [\#285](https://github.com/agones-dev/agones/issues/285)
- Add Amazon EKS Agones Setup Instructions [\#372](https://github.com/agones-dev/agones/pull/372) ([GabeBigBoxVR](https://github.com/GabeBigBoxVR))
- Prioritise Allocation from Nodes with Allocated/Ready GameServers [\#370](https://github.com/agones-dev/agones/pull/370) ([markmandel](https://github.com/markmandel))
- Agones stable helm repository [\#361](https://github.com/agones-dev/agones/pull/361) ([cyriltovena](https://github.com/cyriltovena))
- Improve support for custom dev environments [\#349](https://github.com/agones-dev/agones/pull/349) ([victor-prodan](https://github.com/victor-prodan))
- FleetAutoScaler v0 [\#340](https://github.com/agones-dev/agones/pull/340) ([victor-prodan](https://github.com/victor-prodan))
- Forces restart when using tls generation. [\#338](https://github.com/agones-dev/agones/pull/338) ([cyriltovena](https://github.com/cyriltovena))

**Fixed bugs:**

- Fix loophole in game server initialization [\#354](https://github.com/agones-dev/agones/issues/354)
- Health messages logged with wrong severity [\#335](https://github.com/agones-dev/agones/issues/335)
- Helm upgrade and SSL certificates [\#309](https://github.com/agones-dev/agones/issues/309)
- Fix for race condition: Allocation of Deleting GameServers Possible [\#367](https://github.com/agones-dev/agones/pull/367) ([markmandel](https://github.com/markmandel))
- Map level to severity for stackdriver [\#363](https://github.com/agones-dev/agones/pull/363) ([cyriltovena](https://github.com/cyriltovena))
- Add ReadTimeout for e2e tests, otherwise this can hang forever. [\#359](https://github.com/agones-dev/agones/pull/359) ([markmandel](https://github.com/markmandel))
- Fixes race condition bug with Pod not being scheduled before Ready\(\) [\#357](https://github.com/agones-dev/agones/pull/357) ([markmandel](https://github.com/markmandel))
- Allocation is broken when using the generated go client [\#347](https://github.com/agones-dev/agones/pull/347) ([markmandel](https://github.com/markmandel))

**Security fixes:**

- \[Vuln\] Update to Alpine 3.8.1 [\#355](https://github.com/agones-dev/agones/issues/355)
- Update Alpine version to 3.8.1 [\#364](https://github.com/agones-dev/agones/pull/364) ([fooock](https://github.com/fooock))

**Closed issues:**

- C++ SDK no destructor body [\#366](https://github.com/agones-dev/agones/issues/366)
- Release 0.4.0 [\#341](https://github.com/agones-dev/agones/issues/341)
- Update "Developing, Testing and Building Agones" tutorial with how to push updates to your test cluster [\#308](https://github.com/agones-dev/agones/issues/308)
- Use revive instead of gometalinter [\#237](https://github.com/agones-dev/agones/issues/237)
- Integrate a spell and/or grammar check into build system [\#187](https://github.com/agones-dev/agones/issues/187)
- Helm package CI [\#153](https://github.com/agones-dev/agones/issues/153)
- Use functional parameters in Controller creation [\#104](https://github.com/agones-dev/agones/issues/104)

**Merged pull requests:**

- Release 0.5.0.rc changes [\#379](https://github.com/agones-dev/agones/pull/379) ([markmandel](https://github.com/markmandel))
- Make WaitForFleetCondition take up to 5 minutes [\#377](https://github.com/agones-dev/agones/pull/377) ([markmandel](https://github.com/markmandel))
- Fix for flaky test TestControllerAddress [\#376](https://github.com/agones-dev/agones/pull/376) ([markmandel](https://github.com/markmandel))
- Fix typo [\#374](https://github.com/agones-dev/agones/pull/374) ([maxpain](https://github.com/maxpain))
- Update instructions for Minikube 0.29.0 [\#373](https://github.com/agones-dev/agones/pull/373) ([markmandel](https://github.com/markmandel))
- Update README.md [\#371](https://github.com/agones-dev/agones/pull/371) ([iamrare](https://github.com/iamrare))
- Remove c++ sdk destructor causing linker errors [\#369](https://github.com/agones-dev/agones/pull/369) ([nikibobi](https://github.com/nikibobi))
- Update README.md [\#362](https://github.com/agones-dev/agones/pull/362) ([iamrare](https://github.com/iamrare))
- Upgrade GKE version and increase test cluster size [\#360](https://github.com/agones-dev/agones/pull/360) ([markmandel](https://github.com/markmandel))
- Fix typo in sdk readme which said only two sdks [\#356](https://github.com/agones-dev/agones/pull/356) ([reductor](https://github.com/reductor))
- Add allocator service example and documentation [\#353](https://github.com/agones-dev/agones/pull/353) ([slartibaartfast](https://github.com/slartibaartfast))
- Adding goimports back into the build shell. [\#352](https://github.com/agones-dev/agones/pull/352) ([markmandel](https://github.com/markmandel))
- e2e tests for Fleet Scaling and Updates [\#351](https://github.com/agones-dev/agones/pull/351) ([markmandel](https://github.com/markmandel))
- Switch to golangci-lint [\#346](https://github.com/agones-dev/agones/pull/346) ([cyriltovena](https://github.com/cyriltovena))
- Prepare for next release - 0.5.0.rc [\#343](https://github.com/agones-dev/agones/pull/343) ([markmandel](https://github.com/markmandel))

## [v0.4.0](https://github.com/agones-dev/agones/tree/v0.4.0) (2018-09-04)

[Full Changelog](https://github.com/agones-dev/agones/compare/v0.4.0.rc...v0.4.0)

**Closed issues:**

- Release 0.4.0.rc [\#330](https://github.com/agones-dev/agones/issues/330)

**Merged pull requests:**

- Release 0.4.0 [\#342](https://github.com/agones-dev/agones/pull/342) ([markmandel](https://github.com/markmandel))
- Fix yaml file paths [\#339](https://github.com/agones-dev/agones/pull/339) ([oskoi](https://github.com/oskoi))
- Add Troubleshooting section to Build doc [\#337](https://github.com/agones-dev/agones/pull/337) ([victor-prodan](https://github.com/victor-prodan))
- Preparing for 0.4.0 release next week. [\#333](https://github.com/agones-dev/agones/pull/333) ([markmandel](https://github.com/markmandel))

## [v0.4.0.rc](https://github.com/agones-dev/agones/tree/v0.4.0.rc) (2018-08-28)

[Full Changelog](https://github.com/agones-dev/agones/compare/v0.3.0...v0.4.0.rc)

**Implemented enhancements:**

- When running the SDK sidecar in local mode, be able to specify the backing `GameServer` configuration [\#296](https://github.com/agones-dev/agones/issues/296)
- Move Status \> Address & Status \> Ports population to `Creating` state processing [\#293](https://github.com/agones-dev/agones/issues/293)
- Propagating game server process events to Agones system [\#279](https://github.com/agones-dev/agones/issues/279)
- Session data propagation to dedicated server [\#277](https://github.com/agones-dev/agones/issues/277)
- Ability to pass `GameServer` yaml/json to local sdk server [\#328](https://github.com/agones-dev/agones/pull/328) ([markmandel](https://github.com/markmandel))
- Move Status \> Address & Ports population to `Creating` state processing [\#326](https://github.com/agones-dev/agones/pull/326) ([markmandel](https://github.com/markmandel))
- Implement SDK SetLabel and SetAnnotation functionality [\#323](https://github.com/agones-dev/agones/pull/323) ([markmandel](https://github.com/markmandel))
- Implements SDK callback for GameServer updates [\#316](https://github.com/agones-dev/agones/pull/316) ([markmandel](https://github.com/markmandel))
- Features/e2e [\#315](https://github.com/agones-dev/agones/pull/315) ([cyriltovena](https://github.com/cyriltovena))
- Metadata propagation from fleet allocation to game server [\#312](https://github.com/agones-dev/agones/pull/312) ([victor-prodan](https://github.com/victor-prodan))

**Fixed bugs:**

- Fleet allocation request could not find fleet [\#324](https://github.com/agones-dev/agones/issues/324)
- Hotfix: Ensure multiple Pods don't get created for a GameServer [\#332](https://github.com/agones-dev/agones/pull/332) ([markmandel](https://github.com/markmandel))
- Fleet Allocation via REST was failing [\#325](https://github.com/agones-dev/agones/pull/325) ([markmandel](https://github.com/markmandel))
- Make sure the test-e2e ensures the build image. [\#322](https://github.com/agones-dev/agones/pull/322) ([markmandel](https://github.com/markmandel))
- Update getting started guides with kubectl custom columns [\#319](https://github.com/agones-dev/agones/pull/319) ([markmandel](https://github.com/markmandel))
- Fix bug: Disabled health checking not implemented [\#317](https://github.com/agones-dev/agones/pull/317) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- Release 0.3.0 [\#304](https://github.com/agones-dev/agones/issues/304)
- Change container builder steps to run concurrently [\#186](https://github.com/agones-dev/agones/issues/186)
- Move Deployment in install script out of v1beta1 [\#173](https://github.com/agones-dev/agones/issues/173)
- YAML packaging [\#101](https://github.com/agones-dev/agones/issues/101)

**Merged pull requests:**

- Changelog, and documentation changes for 0.4.0.rc [\#331](https://github.com/agones-dev/agones/pull/331) ([markmandel](https://github.com/markmandel))
- Added github.com/spf13/viper to dep toml [\#327](https://github.com/agones-dev/agones/pull/327) ([markmandel](https://github.com/markmandel))
- Add Minikube instructions [\#321](https://github.com/agones-dev/agones/pull/321) ([slartibaartfast](https://github.com/slartibaartfast))
- Convert Go example into multi-stage Docker build [\#320](https://github.com/agones-dev/agones/pull/320) ([markmandel](https://github.com/markmandel))
- Removal of the legacy port configuration [\#318](https://github.com/agones-dev/agones/pull/318) ([markmandel](https://github.com/markmandel))
- Fix flakiness with TestSidecarHTTPHealthCheck [\#313](https://github.com/agones-dev/agones/pull/313) ([markmandel](https://github.com/markmandel))
- Move linting into it's own serial step [\#311](https://github.com/agones-dev/agones/pull/311) ([markmandel](https://github.com/markmandel))
- Update to move from release to the next version \(0.4.0.rc\) [\#306](https://github.com/agones-dev/agones/pull/306) ([markmandel](https://github.com/markmandel))

## [v0.3.0](https://github.com/agones-dev/agones/tree/v0.3.0) (2018-07-26)

[Full Changelog](https://github.com/agones-dev/agones/compare/v0.3.0.rc...v0.3.0)

**Fixed bugs:**

- Missing `watch` permission in rbac for agones-sdk [\#300](https://github.com/agones-dev/agones/pull/300) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- Release 0.3.0.rc [\#290](https://github.com/agones-dev/agones/issues/290)

**Merged pull requests:**

- Changes for release  0.3.0 [\#305](https://github.com/agones-dev/agones/pull/305) ([markmandel](https://github.com/markmandel))
- Move back to 0.3.0 [\#292](https://github.com/agones-dev/agones/pull/292) ([markmandel](https://github.com/markmandel))

## [v0.3.0.rc](https://github.com/agones-dev/agones/tree/v0.3.0.rc) (2018-07-17)

[Full Changelog](https://github.com/agones-dev/agones/compare/v0.2.0...v0.3.0.rc)

**Breaking changes:**

- \[Breaking Change\] Multiple port support for `GameServer` [\#283](https://github.com/agones-dev/agones/pull/283) ([markmandel](https://github.com/markmandel))

**Implemented enhancements:**

- Expose SDK Sidecar GRPC Server as HTTP+JSON [\#240](https://github.com/agones-dev/agones/issues/240)
- supporting multiple ports [\#151](https://github.com/agones-dev/agones/issues/151)
- Support Cluster Node addition/deletion [\#60](https://github.com/agones-dev/agones/issues/60)
- SDK `GameServer\(\)` function for retrieving backing GameServer configuration [\#288](https://github.com/agones-dev/agones/pull/288) ([markmandel](https://github.com/markmandel))
- Move cluster node addition/removal out of "experimental" [\#271](https://github.com/agones-dev/agones/pull/271) ([markmandel](https://github.com/markmandel))
- added information about Agones running on Azure Kubernetes Service [\#269](https://github.com/agones-dev/agones/pull/269) ([dgkanatsios](https://github.com/dgkanatsios))
- Expose SDK-Server at HTTP+JSON [\#265](https://github.com/agones-dev/agones/pull/265) ([markmandel](https://github.com/markmandel))
- Support Rust SDK by gRPC-rs [\#230](https://github.com/agones-dev/agones/pull/230) ([thara](https://github.com/thara))

**Fixed bugs:**

- Minikube does not start with 0.26.x [\#192](https://github.com/agones-dev/agones/issues/192)
- Forgot to update the k8s client-go codegen. [\#281](https://github.com/agones-dev/agones/pull/281) ([markmandel](https://github.com/markmandel))
- Fix bug with hung GameServer resource on Kubernetes 1.10 [\#278](https://github.com/agones-dev/agones/pull/278) ([markmandel](https://github.com/markmandel))
- Fix Xonotic example race condition [\#266](https://github.com/agones-dev/agones/pull/266) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- Agones on Azure AKS [\#254](https://github.com/agones-dev/agones/issues/254)
- Release v0.2.0 [\#242](https://github.com/agones-dev/agones/issues/242)
- helm namespace [\#212](https://github.com/agones-dev/agones/issues/212)

**Merged pull requests:**

- Release 0.3.0.rc [\#291](https://github.com/agones-dev/agones/pull/291) ([markmandel](https://github.com/markmandel))
- Update README.md with information about Public IPs on AKS [\#289](https://github.com/agones-dev/agones/pull/289) ([dgkanatsios](https://github.com/dgkanatsios))
- fix yaml install link [\#286](https://github.com/agones-dev/agones/pull/286) ([nikibobi](https://github.com/nikibobi))
- install.yaml now installs by default in agones-system [\#284](https://github.com/agones-dev/agones/pull/284) ([cyriltovena](https://github.com/cyriltovena))
- Update GKE testing cluster to 1.10.5 [\#280](https://github.com/agones-dev/agones/pull/280) ([markmandel](https://github.com/markmandel))
- Update dependencies to support K8s 1.10.x [\#276](https://github.com/agones-dev/agones/pull/276) ([markmandel](https://github.com/markmandel))
- Remove line [\#274](https://github.com/agones-dev/agones/pull/274) ([markmandel](https://github.com/markmandel))
- Update minikube instructions to 0.28.0 [\#273](https://github.com/agones-dev/agones/pull/273) ([markmandel](https://github.com/markmandel))
- Added list of various libs used in code [\#272](https://github.com/agones-dev/agones/pull/272) ([meanmango](https://github.com/meanmango))
- More Docker and Kubernetes Getting Started Resources [\#270](https://github.com/agones-dev/agones/pull/270) ([markmandel](https://github.com/markmandel))
- Fixing Flaky test TestControllerSyncFleet [\#268](https://github.com/agones-dev/agones/pull/268) ([markmandel](https://github.com/markmandel))
- Update Helm App Version [\#267](https://github.com/agones-dev/agones/pull/267) ([markmandel](https://github.com/markmandel))
- Give linter 15 minutes. [\#264](https://github.com/agones-dev/agones/pull/264) ([markmandel](https://github.com/markmandel))
- Upgrade to Go 1.10.3 [\#263](https://github.com/agones-dev/agones/pull/263) ([markmandel](https://github.com/markmandel))
- Upgrade Helm for build tools [\#262](https://github.com/agones-dev/agones/pull/262) ([markmandel](https://github.com/markmandel))
- Fixed some links & typos [\#261](https://github.com/agones-dev/agones/pull/261) ([meanmango](https://github.com/meanmango))
- Flaky test fix: TestWorkQueueHealthCheck [\#260](https://github.com/agones-dev/agones/pull/260) ([markmandel](https://github.com/markmandel))
- Upgrade gRPC to 1.12.0 [\#259](https://github.com/agones-dev/agones/pull/259) ([markmandel](https://github.com/markmandel))
- Flakey test fix: TestControllerUpdateFleetStatus [\#257](https://github.com/agones-dev/agones/pull/257) ([markmandel](https://github.com/markmandel))
- Remove reference to internal console site. [\#256](https://github.com/agones-dev/agones/pull/256) ([dzlier-gcp](https://github.com/dzlier-gcp))
- Add Licences to Rust SDK & Examples [\#253](https://github.com/agones-dev/agones/pull/253) ([markmandel](https://github.com/markmandel))
- Clearer Helm installation instructions [\#252](https://github.com/agones-dev/agones/pull/252) ([markmandel](https://github.com/markmandel))
- Rust SDK Doc additions [\#251](https://github.com/agones-dev/agones/pull/251) ([markmandel](https://github.com/markmandel))
- use the helm --namespace convention  [\#250](https://github.com/agones-dev/agones/pull/250) ([cyriltovena](https://github.com/cyriltovena))
- fix podspec template broken link to documentation [\#247](https://github.com/agones-dev/agones/pull/247) ([cyriltovena](https://github.com/cyriltovena))
- Make Cloud Builder Faster [\#245](https://github.com/agones-dev/agones/pull/245) ([markmandel](https://github.com/markmandel))
- Increment base version [\#244](https://github.com/agones-dev/agones/pull/244) ([markmandel](https://github.com/markmandel))
- Lock protoc-gen-go to 1.0 release [\#241](https://github.com/agones-dev/agones/pull/241) ([markmandel](https://github.com/markmandel))

## [v0.2.0](https://github.com/agones-dev/agones/tree/v0.2.0) (2018-06-06)

[Full Changelog](https://github.com/agones-dev/agones/compare/v0.2.0.rc...v0.2.0)

**Closed issues:**

- Release v0.2.0.rc [\#231](https://github.com/agones-dev/agones/issues/231)

**Merged pull requests:**

- Release 0.2.0 [\#243](https://github.com/agones-dev/agones/pull/243) ([markmandel](https://github.com/markmandel))
- Adding my streaming development to contributing [\#239](https://github.com/agones-dev/agones/pull/239) ([markmandel](https://github.com/markmandel))
- Updates to release process [\#235](https://github.com/agones-dev/agones/pull/235) ([markmandel](https://github.com/markmandel))
- Adding a README.md file for the simple-udp to help developer to get start [\#234](https://github.com/agones-dev/agones/pull/234) ([g-ericso](https://github.com/g-ericso))
- Revert install configuration back to 0.2.0 [\#233](https://github.com/agones-dev/agones/pull/233) ([markmandel](https://github.com/markmandel))

## [v0.2.0.rc](https://github.com/agones-dev/agones/tree/v0.2.0.rc) (2018-05-30)

[Full Changelog](https://github.com/agones-dev/agones/compare/v0.1...v0.2.0.rc)

**Implemented enhancements:**

- Generate Certs for Mutation/Validatiion Webhooks [\#169](https://github.com/agones-dev/agones/issues/169)
- Add liveness check to `pkg/gameservers/controller`. [\#116](https://github.com/agones-dev/agones/issues/116)
- GameServer Fleets [\#70](https://github.com/agones-dev/agones/issues/70)
- Release steps of archiving installation resources and documentation [\#226](https://github.com/agones-dev/agones/pull/226) ([markmandel](https://github.com/markmandel))
- Lint timeout increase, and make configurable [\#221](https://github.com/agones-dev/agones/pull/221) ([markmandel](https://github.com/markmandel))
- add the ability to turn off RBAC in helm and customize gcp test-cluster [\#220](https://github.com/agones-dev/agones/pull/220) ([cyriltovena](https://github.com/cyriltovena))
- Target for generating a CHANGELOG from GitHub Milestones. [\#217](https://github.com/agones-dev/agones/pull/217) ([markmandel](https://github.com/markmandel))
- Generate Certs for Mutation/Validatiion Webhooks [\#214](https://github.com/agones-dev/agones/pull/214) ([cyriltovena](https://github.com/cyriltovena))
- Rolling updates for Fleets [\#213](https://github.com/agones-dev/agones/pull/213) ([markmandel](https://github.com/markmandel))
- helm namespaces [\#210](https://github.com/agones-dev/agones/pull/210) ([cyriltovena](https://github.com/cyriltovena))
- Fleet update strategy: Replace [\#199](https://github.com/agones-dev/agones/pull/199) ([markmandel](https://github.com/markmandel))
- Status \> AllocatedReplicas on Fleets & GameServers [\#196](https://github.com/agones-dev/agones/pull/196) ([markmandel](https://github.com/markmandel))
- Creating a FleetAllocation allocated a GameServer from a Fleet [\#193](https://github.com/agones-dev/agones/pull/193) ([markmandel](https://github.com/markmandel))
- Add nano as editor to the build image [\#179](https://github.com/agones-dev/agones/pull/179) ([markmandel](https://github.com/markmandel))
- Feature/gometalinter [\#176](https://github.com/agones-dev/agones/pull/176) ([EricFortin](https://github.com/EricFortin))
- Creating a Fleet creates a GameServerSet [\#174](https://github.com/agones-dev/agones/pull/174) ([markmandel](https://github.com/markmandel))
- Register liveness check in gameservers.Controller [\#160](https://github.com/agones-dev/agones/pull/160) ([enocom](https://github.com/enocom))
- GameServerSet Implementation [\#156](https://github.com/agones-dev/agones/pull/156) ([markmandel](https://github.com/markmandel))

**Fixed bugs:**

- gometalinter fails [\#181](https://github.com/agones-dev/agones/issues/181)
- Line endings in Windows make the project can't be compiled [\#180](https://github.com/agones-dev/agones/issues/180)
- Missing links in documentation [\#165](https://github.com/agones-dev/agones/issues/165)
- Cannot run GameServer in non-default namespace [\#146](https://github.com/agones-dev/agones/issues/146)
- Don't allow allocation of Deleted GameServers [\#198](https://github.com/agones-dev/agones/pull/198) ([markmandel](https://github.com/markmandel))
- Fixes for GKE issues with install/quickstart [\#197](https://github.com/agones-dev/agones/pull/197) ([markmandel](https://github.com/markmandel))
- `minikube-test-cluster` needed the `ensure-build-image` dependency [\#194](https://github.com/agones-dev/agones/pull/194) ([markmandel](https://github.com/markmandel))
- Update initialClusterVersion to 1.9.6.gke.1 [\#190](https://github.com/agones-dev/agones/pull/190) ([markmandel](https://github.com/markmandel))
- Point the install.yaml to the release-0.1 branch [\#189](https://github.com/agones-dev/agones/pull/189) ([markmandel](https://github.com/markmandel))
- Fixed missing links in documentation. [\#166](https://github.com/agones-dev/agones/pull/166) ([fooock](https://github.com/fooock))

**Security fixes:**

- RBAC: controller doesn't need fleet create [\#202](https://github.com/agones-dev/agones/pull/202) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- helm RBAC on/off [\#211](https://github.com/agones-dev/agones/issues/211)
- Release cycle [\#203](https://github.com/agones-dev/agones/issues/203)
- Fix cyclomatic complexity in examples/simple-udp/server/main.go [\#178](https://github.com/agones-dev/agones/issues/178)
- Fix cyclomatic complexity in cmd/controller/main.go [\#177](https://github.com/agones-dev/agones/issues/177)
- Add .helmignore to Helm chart [\#168](https://github.com/agones-dev/agones/issues/168)
- Add gometalinter to build [\#163](https://github.com/agones-dev/agones/issues/163)
- Google Bot is double posting [\#155](https://github.com/agones-dev/agones/issues/155)
- Add .editorconfig to ensure common formatting [\#97](https://github.com/agones-dev/agones/issues/97)

**Merged pull requests:**

- Release v0.2.0.rc [\#232](https://github.com/agones-dev/agones/pull/232) ([markmandel](https://github.com/markmandel))
- do-release release registry and upstream push [\#228](https://github.com/agones-dev/agones/pull/228) ([markmandel](https://github.com/markmandel))
- Archive C++ src on build and release [\#227](https://github.com/agones-dev/agones/pull/227) ([markmandel](https://github.com/markmandel))
- Update installing\_agones.md [\#225](https://github.com/agones-dev/agones/pull/225) ([g-ericso](https://github.com/g-ericso))
- Some missing tasks in the release [\#224](https://github.com/agones-dev/agones/pull/224) ([markmandel](https://github.com/markmandel))
- Move to proper semver [\#223](https://github.com/agones-dev/agones/pull/223) ([markmandel](https://github.com/markmandel))
- Update tools - vetshadow is more aggressive [\#222](https://github.com/agones-dev/agones/pull/222) ([markmandel](https://github.com/markmandel))
- add helm ignore file [\#219](https://github.com/agones-dev/agones/pull/219) ([cyriltovena](https://github.com/cyriltovena))
- Intercept Xonotic stdout for SDK Integration [\#218](https://github.com/agones-dev/agones/pull/218) ([markmandel](https://github.com/markmandel))
- Some more Extending Kubernetes resources [\#216](https://github.com/agones-dev/agones/pull/216) ([markmandel](https://github.com/markmandel))
- Release process documentation [\#215](https://github.com/agones-dev/agones/pull/215) ([markmandel](https://github.com/markmandel))
- Fix cyclomatic complexity in cmd/controller/main.go [\#209](https://github.com/agones-dev/agones/pull/209) ([enocom](https://github.com/enocom))
- Testing functions for resource events [\#208](https://github.com/agones-dev/agones/pull/208) ([markmandel](https://github.com/markmandel))
- Shrink main func to resolve gocyclo warning [\#207](https://github.com/agones-dev/agones/pull/207) ([enocom](https://github.com/enocom))
- Clearer docs on developing and building from source [\#206](https://github.com/agones-dev/agones/pull/206) ([markmandel](https://github.com/markmandel))
- Add formatting guidelines to CONTRIBUTING.md [\#205](https://github.com/agones-dev/agones/pull/205) ([enocom](https://github.com/enocom))
- Fleet docs: Some missing pieces. [\#204](https://github.com/agones-dev/agones/pull/204) ([markmandel](https://github.com/markmandel))
- Release version, and twitter badges. [\#201](https://github.com/agones-dev/agones/pull/201) ([markmandel](https://github.com/markmandel))
- Typo in GameServer json [\#200](https://github.com/agones-dev/agones/pull/200) ([markmandel](https://github.com/markmandel))
- Install docs: minikube 0.25.2 and k8s 1.9.4 [\#195](https://github.com/agones-dev/agones/pull/195) ([markmandel](https://github.com/markmandel))
- Update temporary auth against Google Container Registry [\#191](https://github.com/agones-dev/agones/pull/191) ([markmandel](https://github.com/markmandel))
- Make the development release warning more visible. [\#188](https://github.com/agones-dev/agones/pull/188) ([markmandel](https://github.com/markmandel))
- Solve rare flakiness on TestWorkerQueueHealthy [\#185](https://github.com/agones-dev/agones/pull/185) ([markmandel](https://github.com/markmandel))
- Adding additional resources for CRDs and Controllers [\#184](https://github.com/agones-dev/agones/pull/184) ([markmandel](https://github.com/markmandel))
- Reworked some Dockerfiles to improve cache usage. [\#183](https://github.com/agones-dev/agones/pull/183) ([EricFortin](https://github.com/EricFortin))
- Windows eol [\#182](https://github.com/agones-dev/agones/pull/182) ([fooock](https://github.com/fooock))
- Missed a couple of small things in last PR [\#175](https://github.com/agones-dev/agones/pull/175) ([markmandel](https://github.com/markmandel))
- Centralise utilities for testing controllers [\#172](https://github.com/agones-dev/agones/pull/172) ([markmandel](https://github.com/markmandel))
- Generate the install.yaml from `helm template` [\#171](https://github.com/agones-dev/agones/pull/171) ([markmandel](https://github.com/markmandel))
- Integrated Helm into the `build` and development system [\#170](https://github.com/agones-dev/agones/pull/170) ([markmandel](https://github.com/markmandel))
- Refactor of workerqueue health testing [\#164](https://github.com/agones-dev/agones/pull/164) ([markmandel](https://github.com/markmandel))
- Fix some Go Report Card warnings [\#162](https://github.com/agones-dev/agones/pull/162) ([dvrkps](https://github.com/dvrkps))
- fix typo found by report card [\#161](https://github.com/agones-dev/agones/pull/161) ([cyriltovena](https://github.com/cyriltovena))
- Why does this project exist? [\#159](https://github.com/agones-dev/agones/pull/159) ([markmandel](https://github.com/markmandel))
- Add GKE Comic to explain Kubernetes to newcomers [\#158](https://github.com/agones-dev/agones/pull/158) ([markmandel](https://github.com/markmandel))
- Adding a Go Report Card [\#157](https://github.com/agones-dev/agones/pull/157) ([markmandel](https://github.com/markmandel))
- Documentation on the CI build system. [\#152](https://github.com/agones-dev/agones/pull/152) ([markmandel](https://github.com/markmandel))
- Helm integration [\#149](https://github.com/agones-dev/agones/pull/149) ([fooock](https://github.com/fooock))
- Minor rewording [\#148](https://github.com/agones-dev/agones/pull/148) ([bransorem](https://github.com/bransorem))
- Move GameServer validation to a ValidatingAdmissionWebhook [\#147](https://github.com/agones-dev/agones/pull/147) ([markmandel](https://github.com/markmandel))
- Update create\_gameserver.md [\#143](https://github.com/agones-dev/agones/pull/143) ([royingantaginting](https://github.com/royingantaginting))
- Fixed spelling issues in gameserver\_spec.md [\#141](https://github.com/agones-dev/agones/pull/141) ([mattva01](https://github.com/mattva01))
- Handle stop signal in the SDK Server [\#140](https://github.com/agones-dev/agones/pull/140) ([markmandel](https://github.com/markmandel))
- go vet: 3 warnings, 2 of them are easy. [\#139](https://github.com/agones-dev/agones/pull/139) ([Deleplace](https://github.com/Deleplace))
- Update Go version to 1.10 [\#137](https://github.com/agones-dev/agones/pull/137) ([markmandel](https://github.com/markmandel))
- Cleanup of grpc go generation code [\#136](https://github.com/agones-dev/agones/pull/136) ([markmandel](https://github.com/markmandel))
- Update base version to 0.2 [\#133](https://github.com/agones-dev/agones/pull/133) ([markmandel](https://github.com/markmandel))
- Centralise the canonical import paths and more package docs [\#130](https://github.com/agones-dev/agones/pull/130) ([markmandel](https://github.com/markmandel))

## [v0.1](https://github.com/agones-dev/agones/tree/v0.1) (2018-03-06)

[Full Changelog](https://github.com/agones-dev/agones/compare/20f6ab798a49e3629d5f6651201504ff49ea251a...v0.1)

**Implemented enhancements:**

- The local mode of the agon sidecar listen to localhost only [\#62](https://github.com/agones-dev/agones/issues/62)
- Record Events for GameServer State Changes [\#32](https://github.com/agones-dev/agones/issues/32)
- Use a single install.yaml to install Agon [\#17](https://github.com/agones-dev/agones/issues/17)
- SDK + Sidecar implementation [\#16](https://github.com/agones-dev/agones/issues/16)
- Game Server health checking [\#15](https://github.com/agones-dev/agones/issues/15)
- Dynamic Port Allocation on Game Servers [\#14](https://github.com/agones-dev/agones/issues/14)
- Sidecar needs a healthcheck [\#12](https://github.com/agones-dev/agones/issues/12)
- Health Check for the Controller [\#11](https://github.com/agones-dev/agones/issues/11)
- GameServer definition validation [\#10](https://github.com/agones-dev/agones/issues/10)
- Default RestartPolicy should be Never on the GameServer container [\#9](https://github.com/agones-dev/agones/issues/9)
- Mac & Windows binaries for local development [\#8](https://github.com/agones-dev/agones/issues/8)
- `gcloud docker --authorize` make target and push targets [\#5](https://github.com/agones-dev/agones/issues/5)
- Do-release target to automate releases [\#121](https://github.com/agones-dev/agones/pull/121) ([markmandel](https://github.com/markmandel))
- Zip archive of sdk server server binaries for release [\#118](https://github.com/agones-dev/agones/pull/118) ([markmandel](https://github.com/markmandel))
- add hostPort and container validations to webhook [\#106](https://github.com/agones-dev/agones/pull/106) ([cyriltovena](https://github.com/cyriltovena))
- MutatingWebHookConfiguration for GameServer creation & Validation. [\#95](https://github.com/agones-dev/agones/pull/95) ([markmandel](https://github.com/markmandel))
- Address flag for the sidecar [\#73](https://github.com/agones-dev/agones/pull/73) ([markmandel](https://github.com/markmandel))
- Allow extra args to be passed into minikube-shell [\#71](https://github.com/agones-dev/agones/pull/71) ([markmandel](https://github.com/markmandel))
- Implementation of Health Checking [\#69](https://github.com/agones-dev/agones/pull/69) ([markmandel](https://github.com/markmandel))
- Develop and Build on Windows \(WSL\) with Minikube [\#59](https://github.com/agones-dev/agones/pull/59) ([markmandel](https://github.com/markmandel))
- Recording GameServers Kubernetes Events [\#56](https://github.com/agones-dev/agones/pull/56) ([markmandel](https://github.com/markmandel))
- Add health check for gameserver-sidecar. [\#44](https://github.com/agones-dev/agones/pull/44) ([dzlier-gcp](https://github.com/dzlier-gcp))
- Dynamic Port Allocation for GameServers [\#41](https://github.com/agones-dev/agones/pull/41) ([markmandel](https://github.com/markmandel))
- Finalizer for GameServer until backing Pods are Terminated [\#40](https://github.com/agones-dev/agones/pull/40) ([markmandel](https://github.com/markmandel))
- Continuous Integration with Container Builder [\#38](https://github.com/agones-dev/agones/pull/38) ([markmandel](https://github.com/markmandel))
- Windows and OSX builds of the sidecar [\#36](https://github.com/agones-dev/agones/pull/36) ([markmandel](https://github.com/markmandel))
- C++ SDK implementation, example and doc [\#35](https://github.com/agones-dev/agones/pull/35) ([markmandel](https://github.com/markmandel))
- Use a sha256 of Dockerfile for build-image [\#25](https://github.com/agones-dev/agones/pull/25) ([markmandel](https://github.com/markmandel))
- Utilises Xonotic.org to build and run an actual game on Agon. [\#23](https://github.com/agones-dev/agones/pull/23) ([markmandel](https://github.com/markmandel))
- Go SDK for integration with Game Servers. [\#20](https://github.com/agones-dev/agones/pull/20) ([markmandel](https://github.com/markmandel))

**Fixed bugs:**

- `make gcloud-auth-docker` fails on Windows [\#49](https://github.com/agones-dev/agones/issues/49)
- Convert `ENTRYPOINT foo` to `ENTRYPOINT \["/path/foo"\]` [\#39](https://github.com/agones-dev/agones/issues/39)
- Handle SIGTERM in Controller [\#33](https://github.com/agones-dev/agones/issues/33)
- Gopkg.toml should use tags not branches for k8s.io dependencies [\#1](https://github.com/agones-dev/agones/issues/1)
- fix liveness probe in the install.yaml [\#119](https://github.com/agones-dev/agones/pull/119) ([cyriltovena](https://github.com/cyriltovena))
- Make Port Allocator idempotent for GameServers and Node events [\#117](https://github.com/agones-dev/agones/pull/117) ([markmandel](https://github.com/markmandel))
- DeleteFunc could recieve a DeletedFinalStateUnknown [\#113](https://github.com/agones-dev/agones/pull/113) ([markmandel](https://github.com/markmandel))
- Goimports wasn't running on CRD generation [\#99](https://github.com/agones-dev/agones/pull/99) ([markmandel](https://github.com/markmandel))
- Fix a bug in HandleError [\#67](https://github.com/agones-dev/agones/pull/67) ([markmandel](https://github.com/markmandel))
- Minikube targts: make sure they are on the agon minikube profile [\#66](https://github.com/agones-dev/agones/pull/66) ([markmandel](https://github.com/markmandel))
- Header insert on gRPC code gen touched too many files [\#58](https://github.com/agones-dev/agones/pull/58) ([markmandel](https://github.com/markmandel))
- Fix for health check stability issues [\#55](https://github.com/agones-dev/agones/pull/55) ([markmandel](https://github.com/markmandel))
- `make gcloud-auth-docker` works on Windows [\#50](https://github.com/agones-dev/agones/pull/50) ([markmandel](https://github.com/markmandel))
- Use the preferred ENTRYPOINT format [\#43](https://github.com/agones-dev/agones/pull/43) ([markmandel](https://github.com/markmandel))
- Update Kubernetes dependencies to release branch [\#24](https://github.com/agones-dev/agones/pull/24) ([markmandel](https://github.com/markmandel))

**Security fixes:**

- Switch to RBAC [\#57](https://github.com/agones-dev/agones/issues/57)
- Upgrade to Go 1.9.4 [\#81](https://github.com/agones-dev/agones/pull/81) ([markmandel](https://github.com/markmandel))

**Closed issues:**

- `make do-release` target [\#115](https://github.com/agones-dev/agones/issues/115)
- Creating a Kubernetes Cluster quickstart [\#93](https://github.com/agones-dev/agones/issues/93)
- Namespace for Agones infrastructure [\#89](https://github.com/agones-dev/agones/issues/89)
- Health check should be moved out of `gameservers/controller.go` [\#88](https://github.com/agones-dev/agones/issues/88)
- Add archiving the sdk-server binaries into gcs into the cloudbuild.yaml [\#87](https://github.com/agones-dev/agones/issues/87)
- Upgrade to Go 1.9.3 [\#63](https://github.com/agones-dev/agones/issues/63)
- Building Agon on Windows [\#47](https://github.com/agones-dev/agones/issues/47)
- Building Agones on macOS [\#46](https://github.com/agones-dev/agones/issues/46)
- Write documentation for creating a GameServer [\#45](https://github.com/agones-dev/agones/issues/45)
- Agon should work on Minikube [\#30](https://github.com/agones-dev/agones/issues/30)
- Remove the entrypoint from the build-image [\#28](https://github.com/agones-dev/agones/issues/28)
- Base Go Version and Docker image tag on Git commit sha [\#21](https://github.com/agones-dev/agones/issues/21)
- Tag agon-build with hash of the Dockerfile [\#19](https://github.com/agones-dev/agones/issues/19)
- Example using Xonotic [\#18](https://github.com/agones-dev/agones/issues/18)
- Continuous Integration [\#13](https://github.com/agones-dev/agones/issues/13)
- C++ SDK [\#7](https://github.com/agones-dev/agones/issues/7)
- Upgrade to alpine 3.7 [\#4](https://github.com/agones-dev/agones/issues/4)
- Make controller SchemeGroupVersion a var [\#3](https://github.com/agones-dev/agones/issues/3)
- Consolidate `Version` into a single constant [\#2](https://github.com/agones-dev/agones/issues/2)

**Merged pull requests:**

- Godoc badge! [\#131](https://github.com/agones-dev/agones/pull/131) ([markmandel](https://github.com/markmandel))
- add missing link to git message documentation [\#129](https://github.com/agones-dev/agones/pull/129) ([cyriltovena](https://github.com/cyriltovena))
- Minor tweak to top line description of Agones. [\#127](https://github.com/agones-dev/agones/pull/127) ([markmandel](https://github.com/markmandel))
- Documentation for programmatically working with Agones. [\#126](https://github.com/agones-dev/agones/pull/126) ([markmandel](https://github.com/markmandel))
- Extend documentation for SDKs [\#125](https://github.com/agones-dev/agones/pull/125) ([markmandel](https://github.com/markmandel))
- Documentation quickstart and specification gameserver [\#124](https://github.com/agones-dev/agones/pull/124) ([cyriltovena](https://github.com/cyriltovena))
- Add MutatingAdmissionWebhook requirements to README [\#123](https://github.com/agones-dev/agones/pull/123) ([markmandel](https://github.com/markmandel))
- Add cluster creation Quickstart. [\#122](https://github.com/agones-dev/agones/pull/122) ([dzlier-gcp](https://github.com/dzlier-gcp))
- Fix typo introduced by refactor [\#120](https://github.com/agones-dev/agones/pull/120) ([markmandel](https://github.com/markmandel))
- Cleanup on GameServer Controller [\#114](https://github.com/agones-dev/agones/pull/114) ([markmandel](https://github.com/markmandel))
- Fixed some typos. [\#112](https://github.com/agones-dev/agones/pull/112) ([EricFortin](https://github.com/EricFortin))
- Added the source of the name to the Readme. [\#111](https://github.com/agones-dev/agones/pull/111) ([markmandel](https://github.com/markmandel))
- Add 'make' to minikube target commands [\#109](https://github.com/agones-dev/agones/pull/109) ([joeholley](https://github.com/joeholley))
- Move WaitForEstablishedCRD into central `crd` package [\#108](https://github.com/agones-dev/agones/pull/108) ([markmandel](https://github.com/markmandel))
- Centralise Controller Queue and Worker processing [\#107](https://github.com/agones-dev/agones/pull/107) ([markmandel](https://github.com/markmandel))
- Slack community! [\#105](https://github.com/agones-dev/agones/pull/105) ([markmandel](https://github.com/markmandel))
- Add an `source` to all log statements [\#103](https://github.com/agones-dev/agones/pull/103) ([markmandel](https://github.com/markmandel))
- Putting the code of conduct front of page. [\#102](https://github.com/agones-dev/agones/pull/102) ([markmandel](https://github.com/markmandel))
- Add CRD validation via OpenAPIv3 Schema [\#100](https://github.com/agones-dev/agones/pull/100) ([cyriltovena](https://github.com/cyriltovena))
- Use github.com/heptio/healthcheck [\#98](https://github.com/agones-dev/agones/pull/98) ([enocom](https://github.com/enocom))
- Adding CoC and Discuss mailing lists. [\#96](https://github.com/agones-dev/agones/pull/96) ([markmandel](https://github.com/markmandel))
- Standardize on LF line endings [\#92](https://github.com/agones-dev/agones/pull/92) ([enocom](https://github.com/enocom))
- Move Agones resources into a `agones-system` namespace. [\#91](https://github.com/agones-dev/agones/pull/91) ([markmandel](https://github.com/markmandel))
- Support builds on macOS [\#90](https://github.com/agones-dev/agones/pull/90) ([enocom](https://github.com/enocom))
- Enable RBAC [\#86](https://github.com/agones-dev/agones/pull/86) ([dzlier-gcp](https://github.com/dzlier-gcp))
- Update everything to be Kubernetes 1.9+ [\#85](https://github.com/agones-dev/agones/pull/85) ([markmandel](https://github.com/markmandel))
- Expand on contributing documentation. [\#84](https://github.com/agones-dev/agones/pull/84) ([markmandel](https://github.com/markmandel))
- Remove entrypoints in makefile. [\#82](https://github.com/agones-dev/agones/pull/82) ([cyriltovena](https://github.com/cyriltovena))
- Update to client-go release 1.6 [\#80](https://github.com/agones-dev/agones/pull/80) ([markmandel](https://github.com/markmandel))
- Setup for social/get involved section. [\#79](https://github.com/agones-dev/agones/pull/79) ([markmandel](https://github.com/markmandel))
- Changing name from Agon =\> Agones. [\#78](https://github.com/agones-dev/agones/pull/78) ([markmandel](https://github.com/markmandel))
- Refactor to centralise controller [\#77](https://github.com/agones-dev/agones/pull/77) ([markmandel](https://github.com/markmandel))
- Missing link to healthchecking. [\#76](https://github.com/agones-dev/agones/pull/76) ([markmandel](https://github.com/markmandel))
- Upgrade to Alpine 3.7 [\#75](https://github.com/agones-dev/agones/pull/75) ([markmandel](https://github.com/markmandel))
- Update to Go 1.9.3 [\#74](https://github.com/agones-dev/agones/pull/74) ([markmandel](https://github.com/markmandel))
- Update Xonotic demo to use dynamic ports [\#72](https://github.com/agones-dev/agones/pull/72) ([markmandel](https://github.com/markmandel))
- Basic structure for better documentation [\#68](https://github.com/agones-dev/agones/pull/68) ([markmandel](https://github.com/markmandel))
- Update gke-test-cluster admin password to new minimum length 16 chars. [\#65](https://github.com/agones-dev/agones/pull/65) ([dzlier-gcp](https://github.com/dzlier-gcp))
- Output the stack error as an actual array [\#61](https://github.com/agones-dev/agones/pull/61) ([markmandel](https://github.com/markmandel))
- Update documentation [\#53](https://github.com/agones-dev/agones/pull/53) ([cyriltovena](https://github.com/cyriltovena))
- Correct maximum parameter typo [\#52](https://github.com/agones-dev/agones/pull/52) ([cyriltovena](https://github.com/cyriltovena))
- Document building Agon on different platforms [\#51](https://github.com/agones-dev/agones/pull/51) ([markmandel](https://github.com/markmandel))
- Development and Deployment to Minikube [\#48](https://github.com/agones-dev/agones/pull/48) ([markmandel](https://github.com/markmandel))
- Fix typo for logrus gameserver field [\#42](https://github.com/agones-dev/agones/pull/42) ([alexandrem](https://github.com/alexandrem))
- Add health check. [\#34](https://github.com/agones-dev/agones/pull/34) ([dzlier-gcp](https://github.com/dzlier-gcp))
- Guide for developing and building Agon. [\#31](https://github.com/agones-dev/agones/pull/31) ([markmandel](https://github.com/markmandel))
- Implement Versioning for dev and release [\#29](https://github.com/agones-dev/agones/pull/29) ([markmandel](https://github.com/markmandel))
- Consolidate the Version constant [\#27](https://github.com/agones-dev/agones/pull/27) ([markmandel](https://github.com/markmandel))
- Make targets `gcloud docker --authorize-only` and `push` [\#26](https://github.com/agones-dev/agones/pull/26) ([markmandel](https://github.com/markmandel))
- zinstall.yaml to install Agon. [\#22](https://github.com/agones-dev/agones/pull/22) ([markmandel](https://github.com/markmandel))
- Disclaimer that Agon is alpha software. [\#6](https://github.com/agones-dev/agones/pull/6) ([markmandel](https://github.com/markmandel))



\* *This Changelog was automatically generated by [github_changelog_generator](https://github.com/github-changelog-generator/github-changelog-generator)*
