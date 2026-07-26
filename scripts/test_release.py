import hashlib
import os
import shutil
import subprocess
import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parent))
import release


def test_arch_matrix_covers_expected():
    names = {a["name"] for a in release.ARCHES}
    assert names == {"amd64", "arm64", "armv7", "armv6", "386", "riscv64"}


def test_output_name():
    assert release.output_name({"name": "amd64"}) == "agent-linux-amd64"
    assert release.output_name({"name": "armv7"}) == "agent-linux-armv7"


def test_go_env_maps_arm_variants():
    base = {"PATH": "/usr/bin"}
    amd = release.go_env({"name": "amd64", "goarch": "amd64", "goarm": ""}, base)
    assert amd["GOOS"] == "linux" and amd["GOARCH"] == "amd64"
    assert "GOARM" not in amd or amd["GOARM"] == ""
    v7 = release.go_env({"name": "armv7", "goarch": "arm", "goarm": "7"}, base)
    assert v7["GOARCH"] == "arm" and v7["GOARM"] == "7"


def test_ldflags_contains_version():
    assert "-X main.version=1.2.3" in release.ldflags("1.2.3")


def test_sha256_file(tmp_path):
    p = tmp_path / "blob"
    p.write_bytes(b"hello")
    expected = hashlib.sha256(b"hello").hexdigest()
    assert release.sha256_file(str(p)) == expected


def test_sha256_line_format(tmp_path):
    p = tmp_path / "agent-linux-amd64"
    p.write_bytes(b"x")
    line = release.sha256_line(str(p))
    assert line.endswith("  agent-linux-amd64")
    assert line.split()[0] == hashlib.sha256(b"x").hexdigest()


@pytest.mark.skipif(shutil.which("go") is None, reason="go toolchain not available")
def test_release_smoke_single_arch(tmp_path, monkeypatch):
    repo = Path(__file__).resolve().parent.parent
    monkeypatch.chdir(repo)
    dist = tmp_path / "dist"
    release.build_release(version="0.0.0-test", commit="deadbee",
                          arches=[a for a in release.ARCHES if a["name"] == "amd64"],
                          dist_dir=str(dist))
    bin_path = dist / "agent-linux-amd64"
    assert bin_path.exists() and bin_path.stat().st_size > 0
    assert (dist / "agent-linux-amd64.sha256").exists()
    assert (dist / "SHA256SUMS").exists()
    assert (dist / "install.sh").exists()
    assert (dist / "uninstall.sh").exists()


def test_agent_ldflags_stamp_the_published_arch():
    """An armv6 build must know it is armv6: GOARM is not readable at runtime,
    so without the stamp it self-updates onto the armv7 binary and bricks."""
    flags = release.ldflags("1.2.3", "armv6")
    assert "-X main.version=1.2.3" in flags
    assert "-X main.buildArch=armv6" in flags


def test_server_ldflags_carry_no_arch_stamp():
    flags = release.ldflags("1.2.3")
    assert "-X main.version=1.2.3" in flags
    assert "buildArch" not in flags
