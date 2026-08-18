import subprocess
from pathlib import Path

import pytest
import yaml

ROOT = Path(__file__).resolve().parent.parent
WORKFLOWS = sorted((ROOT / ".github" / "workflows").glob("*.yml"))


def dry_run(target):
    out = subprocess.run(
        ["make", "-n", target, "-W", "web/package-lock.json"],
        cwd=ROOT,
        capture_output=True,
        text=True,
        check=True,
    )
    return out.stdout


@pytest.mark.parametrize("target", ["web-build", "web-check"])
def test_npm_target_installs_its_own_deps(target):
    lines = [ln.strip() for ln in dry_run(target).splitlines() if ln.strip()]
    assert "npm --prefix web ci" in lines
    run_at = next(i for i, ln in enumerate(lines) if " run " in ln)
    assert lines.index("npm --prefix web ci") < run_at


SELF_INSTALLING = ("make web-build", "make web-check", "make dist")


def job_commands(path):
    doc = yaml.safe_load(path.read_text())
    for name, job in doc["jobs"].items():
        lines = []
        for step in job.get("steps", []):
            lines += [ln.strip() for ln in step.get("run", "").splitlines() if ln.strip()]
        yield f"{path.name}:{name}", lines


@pytest.mark.parametrize("path", WORKFLOWS, ids=lambda p: p.name)
def test_workflow_jobs_install_web_deps_before_running_npm_scripts(path):
    for job, lines in job_commands(path):
        installed = False
        for line in lines:
            if "npm" in line and " ci" in line:
                installed = True
            if any(line.startswith(t) for t in SELF_INSTALLING):
                installed = True
            if "npm" in line and " run " in line:
                assert installed, f"{job} runs {line!r} with no web deps installed"


# The release signing key is what every deployed agent trusts. A third-party
# action running in the same job can read anything in the job's environment,
# so the key stays scoped to the steps that sign and actions are pinned to a
# commit rather than a tag that can be repointed.
@pytest.mark.parametrize("path", WORKFLOWS, ids=lambda p: p.name)
def test_signing_key_is_never_workflow_or_job_scoped(path):
    doc = yaml.safe_load(path.read_text())
    assert "REEVE_SIGNING_KEY" not in (doc.get("env") or {}), f"{path.name}: key is workflow-scoped"
    for name, job in doc["jobs"].items():
        assert "REEVE_SIGNING_KEY" not in (job.get("env") or {}), f"{path.name}:{name}: key is job-scoped"


SHA_LEN = 40


@pytest.mark.parametrize("path", WORKFLOWS, ids=lambda p: p.name)
def test_actions_are_pinned_to_a_commit(path):
    doc = yaml.safe_load(path.read_text())
    for name, job in doc["jobs"].items():
        for step in job.get("steps", []):
            uses = step.get("uses")
            if not uses:
                continue
            ref = uses.split("@", 1)[1]
            assert len(ref) == SHA_LEN and all(c in "0123456789abcdef" for c in ref), \
                f"{path.name}:{name} uses {uses}, which is not pinned to a commit sha"


@pytest.mark.parametrize("path", WORKFLOWS, ids=lambda p: p.name)
def test_workflows_grant_no_blanket_permissions(path):
    doc = yaml.safe_load(path.read_text())
    top = doc.get("permissions")
    assert top is not None, f"{path.name}: no top-level permissions block"
    assert top in ({}, None) or set(top) <= {"contents"}, \
        f"{path.name}: top-level permissions {top} should be narrowed per job"


COMPOSE = ROOT / "deploy" / "docker-compose.yml"
BUILD_COMPOSE = ROOT / "deploy" / "docker-compose.build.yml"

# Compose's !reset tag is not plain YAML, and what it nulls does not matter here.
yaml.SafeLoader.add_constructor("!reset", lambda loader, node: None)


# On the host network there is no port mapping, so REEVE_ADDR is the only thing
# deciding what the server is reachable on, and REEVE_BIND has to still narrow it.
def test_compose_bind_is_overridable():
    server = yaml.safe_load(COMPOSE.read_text())["services"]["server"]
    assert "ports" not in server
    assert server["environment"]["REEVE_ADDR"] == "${REEVE_BIND:-0.0.0.0}:${REEVE_PORT:-7338}"


# mDNS is why the server is not behind a bridge: multicast never leaves it.
def test_compose_uses_the_host_network():
    server = yaml.safe_load(COMPOSE.read_text())["services"]["server"]
    assert server["network_mode"] == "host"


# The updater recreates the server by re-running compose, so it has to land in
# the project it is already part of. A hardcoded or defaulted project name would
# start a second server on the same host port instead of replacing this one.
def test_updater_takes_the_project_name_from_its_own_labels():
    updater = yaml.safe_load(COMPOSE.read_text())["services"]["updater"]
    script = updater["entrypoint"][-1]
    assert 'com.docker.compose.project' in script
    assert "--project-name" in script
    assert "refusing to guess" in script


# The updater mounts the data volume read-only. It reads one file the server
# writes there; write access would put the database in reach of a container
# whose only job is to restart another one.
def test_updater_cannot_write_the_data_volume():
    updater = yaml.safe_load(COMPOSE.read_text())["services"]["updater"]
    data = [v for v in updater["volumes"] if v.startswith("reeve-data:")]
    assert data == ["reeve-data:/state:ro"]


# A compose build with no signing key embeds unsigned agents, and every host
# then refuses to self-update, so the key has to reach the build without anyone
# remembering a --secret flag. It stays a build secret: the running server
# never needs it, and the image must not carry it.
def test_compose_passes_the_signing_key_to_the_build_only():
    doc = yaml.safe_load(BUILD_COMPOSE.read_text())
    server = doc["services"]["server"]
    assert server["build"]["secrets"] == ["signing_key"]
    assert doc["secrets"]["signing_key"] == {"environment": "REEVE_SIGNING_KEY"}
    assert "secrets" not in server
    # The override may only say what the deploy itself changes; the base file owns
    # the rest of what the container runs with.
    assert set(server["environment"]) == {"REEVE_SELF_UPDATE"}


# Compose mounts an env-sourced secret as an empty file when the variable is
# unset, so a -f test would export an empty key and claim it signed.
def test_dockerfile_treats_an_empty_signing_secret_as_absent():
    body = (ROOT / "deploy" / "Dockerfile.server").read_text()
    assert "[ -s /run/secrets/signing_key ]" in body


# The updater recreates containers from the base file alone, so a source build
# left with an updater running would be replaced by a pulled image on the next
# poll — silently undoing the deploy the operator chose.
def test_the_build_override_leaves_the_updater_out():
    updater = yaml.safe_load(BUILD_COMPOSE.read_text())["services"]["updater"]
    assert updater["profiles"] == ["never"]


# Nothing may be pulled on the build path: the tag it builds exists nowhere to
# pull from, and `always` would fail the command trying.
def test_the_build_override_builds_the_server():
    server = yaml.safe_load(BUILD_COMPOSE.read_text())["services"]["server"]
    assert server["build"]["dockerfile"] == "deploy/Dockerfile.server"
    assert server["pull_policy"] == "build"


# An operator who runs `up -d` gets the server and the updater that keeps it
# current. The machine's own agent is installed on the machine: a containerised
# one sees no systemd, no host processes and no host filesystem.
def test_the_default_path_runs_the_published_server_and_the_updater():
    services = yaml.safe_load(COMPOSE.read_text())["services"]
    assert set(services) == {"server", "updater"}
    assert services["server"]["image"].startswith("${REEVE_IMAGE:-ghcr.io/")
    assert "build" not in services["server"]
    assert services["server"]["pull_policy"] == "always"
