#!/usr/bin/env python3
"""Deterministic multi-arch builder for the Reeve agent and server.

Single source of build logic: the Makefile and CI call this, never reimplement
the go build invocation. --release produces dist/ for upload; --embed populates
server/agentdist/ for go:embed; --server adds server binaries to dist/ for
operators who do not run Docker.
"""
import argparse
import hashlib
import os
import shutil
import subprocess
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
VERSION_FILE = REPO / "VERSION"

# The server ships for the two architectures a LAN server realistically runs on;
# the agent covers every board someone might monitor.
SERVER_ARCHES = ["amd64", "arm64"]

ARCHES = [
    {"name": "amd64", "goarch": "amd64", "goarm": ""},
    {"name": "arm64", "goarch": "arm64", "goarm": ""},
    {"name": "armv7", "goarch": "arm", "goarm": "7"},
    {"name": "armv6", "goarch": "arm", "goarm": "6"},
    {"name": "386", "goarch": "386", "goarm": ""},
    {"name": "riscv64", "goarch": "riscv64", "goarm": ""},
]


def read_version():
    return VERSION_FILE.read_text().strip()


def output_name(arch):
    return "agent-linux-" + arch["name"]


def ldflags(version):
    return "-s -w -X main.version=" + version


def go_env(arch, base_env):
    env = dict(base_env)
    env["CGO_ENABLED"] = "0"
    env["GOOS"] = "linux"
    env["GOARCH"] = arch["goarch"]
    if arch["goarm"]:
        env["GOARM"] = arch["goarm"]
    else:
        env.pop("GOARM", None)
    return env


def sha256_file(path):
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(65536), b""):
            h.update(chunk)
    return h.hexdigest()


def sha256_line(path):
    return sha256_file(path) + "  " + os.path.basename(path)


def signing_key_available(key_file=None):
    """Report whether a release signing key is reachable."""
    if key_file and Path(key_file).exists():
        return True
    return bool(os.environ.get("REEVE_SIGNING_KEY"))


def sign_files(paths, comment, key_file=None):
    """Write a .minisig beside each path using the release signing key.

    Agents refuse to self-update to a binary they cannot verify, so an unsigned
    build is a loud warning rather than a silent downgrade in safety.
    """
    if not paths:
        return []
    if not signing_key_available(key_file):
        print("WARNING: no REEVE_SIGNING_KEY; artifacts are unsigned and "
              "agents will refuse to self-update to them", file=sys.stderr)
        return []
    cmd = ["go", "run", "./scripts/sign", "-comment", comment]
    if key_file:
        cmd += ["-key", str(key_file)]
    cmd += [str(p) for p in paths]
    subprocess.run(cmd, cwd=str(REPO), env=dict(os.environ), check=True)
    return [Path(str(p) + ".minisig") for p in paths]


def build_one(arch, version, out_dir):
    out = Path(out_dir) / output_name(arch)
    cmd = ["go", "build", "-ldflags", ldflags(version), "-o", str(out), "./agent"]
    subprocess.run(cmd, cwd=str(REPO), env=go_env(arch, os.environ), check=True)
    return out


def server_output_name(arch):
    return "server-linux-" + arch["name"]


def build_server_one(arch, version, out_dir):
    out = Path(out_dir) / server_output_name(arch)
    cmd = ["go", "build", "-ldflags", ldflags(version), "-o", str(out), "./server"]
    subprocess.run(cmd, cwd=str(REPO), env=go_env(arch, os.environ), check=True)
    return out


def build_servers(version, arches, dist_dir):
    """Build server binaries into dist_dir, appending their checksums.

    The server embeds the built web UI and the agent binaries, so those must be
    staged first (make web-build server-assets); a missing UI is a hard error
    rather than a binary that serves a blank page.
    """
    if not (REPO / "server" / "webdist" / "index.html").exists():
        raise SystemExit("server/webdist/index.html missing: run 'make web-build' before --server")
    dist = Path(dist_dir)
    dist.mkdir(parents=True, exist_ok=True)
    outs = []
    for arch in [a for a in arches if a["name"] in SERVER_ARCHES]:
        out = build_server_one(arch, version, dist)
        (dist / (server_output_name(arch) + ".sha256")).write_text(sha256_line(str(out)) + "\n")
        outs.append(out)
    sums_path = dist / "SHA256SUMS"
    existing = sums_path.read_text().rstrip("\n").split("\n") if sums_path.exists() else []
    existing = [line for line in existing if line]
    existing.extend(sha256_line(str(o)) for o in outs)
    sums_path.write_text("\n".join(existing) + "\n")
    return outs


def build_release(version, commit, arches, dist_dir, key_file=None):
    full = version
    if commit:
        full = version + "+" + commit
    dist = Path(dist_dir)
    if dist.exists():
        shutil.rmtree(dist)
    dist.mkdir(parents=True)
    sums = []
    built = []
    for arch in arches:
        out = build_one(arch, full, dist)
        (Path(dist) / (output_name(arch) + ".sha256")).write_text(sha256_line(str(out)) + "\n")
        sums.append(sha256_line(str(out)))
        built.append(out)
    (dist / "SHA256SUMS").write_text("\n".join(sums) + "\n")
    sign_files(built, "version:" + full, key_file)
    for script in ("install.sh", "uninstall.sh"):
        src = REPO / "deploy" / script
        if src.exists():
            shutil.copy2(src, dist / script)
    return dist


def build_embed(version, arches, embed_dir, key_file=None):
    """Build the agents the server embeds, signing them so hosts can verify.

    The server serves these bytes, so their signatures must be embedded with
    them: an agent asks the server for both.
    """
    embed = Path(embed_dir)
    embed.mkdir(parents=True, exist_ok=True)
    built = []
    for arch in arches:
        built.append(build_one(arch, version, embed))
    sign_files(built, "version:" + version, key_file)


def main(argv):
    ap = argparse.ArgumentParser()
    ap.add_argument("--release", action="store_true")
    ap.add_argument("--embed", action="store_true")
    ap.add_argument("--server", action="store_true", help="also build server binaries into dist/")
    ap.add_argument("--version", default=None)
    ap.add_argument("--commit", default="")
    ap.add_argument("--dist", default=str(REPO / "dist"))
    ap.add_argument("--embed-dir", default=str(REPO / "server" / "agentdist"))
    ap.add_argument("--key", default=None, help="release signing key file (else $REEVE_SIGNING_KEY)")
    args = ap.parse_args(argv)

    version = args.version or read_version()
    if args.release:
        build_release(version, args.commit, ARCHES, args.dist, args.key)
        print("release built:", args.dist)
    if args.embed:
        build_embed(version, ARCHES, args.embed_dir, args.key)
        print("embed built:", args.embed_dir)
    if args.server:
        full = version + ("+" + args.commit if args.commit else "")
        build_servers(full, ARCHES, args.dist)
        print("server built:", args.dist)
    if not args.release and not args.embed and not args.server:
        ap.error("specify --release, --embed, and/or --server")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
