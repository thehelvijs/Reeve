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
PULL_COMPOSE = ROOT / "deploy" / "docker-compose.pull.yml"

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


# The updater only touches containers carrying the label, so dropping the label
# or the flag leaves an instance sitting on the release it was installed with
# while the compose file still claims it updates itself.
def test_pull_override_scopes_the_updater_to_the_server():
    doc = yaml.safe_load(PULL_COMPOSE.read_text())
    label = "com.centurylinklabs.watchtower.enable"
    assert doc["services"]["server"]["labels"][label] == "true"
    assert doc["services"]["updater"]["environment"]["WATCHTOWER_LABEL_ENABLE"] == "true"


# A compose build with no signing key embeds unsigned agents, and every host
# then refuses to self-update, so the key has to reach the build without anyone
# remembering a --secret flag. It stays a build secret: the running server
# never needs it, and the image must not carry it.
def test_compose_passes_the_signing_key_to_the_build_only():
    doc = yaml.safe_load(COMPOSE.read_text())
    server = doc["services"]["server"]
    assert server["build"]["secrets"] == ["signing_key"]
    assert doc["secrets"]["signing_key"] == {"environment": "REEVE_SIGNING_KEY"}
    assert "secrets" not in server
    assert "REEVE_SIGNING_KEY" not in server["environment"]


# Compose mounts an env-sourced secret as an empty file when the variable is
# unset, so a -f test would export an empty key and claim it signed.
def test_dockerfile_treats_an_empty_signing_secret_as_absent():
    body = (ROOT / "deploy" / "Dockerfile.server").read_text()
    assert "[ -s /run/secrets/signing_key ]" in body
