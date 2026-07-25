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
