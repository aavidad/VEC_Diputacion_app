#!/usr/bin/env python3
"""Copia un binario aprobado y sólo los recursos del manifiesto público."""
import argparse
import hashlib
import json
from pathlib import Path
import shutil
import subprocess

ROOT = Path(__file__).resolve().parents[2]


def revision(repository, reference="HEAD"):
    return subprocess.check_output(["git", "-C", str(repository), "rev-parse", "--verify",
                                    reference + "^{commit}"], text=True).strip()


def checked_blob(repository, commit, name):
    expected = subprocess.check_output(["git", "-C", str(repository), "show", commit + ":" + name])
    source = repository / name
    if source.is_symlink() or not source.is_file() or source.read_bytes() != expected:
        raise ValueError("source_worktree_changed")
    return expected


def package(binary, destination, web_source=None, web_commit="HEAD"):
    if destination.exists() or not destination.is_absolute() or binary.is_symlink():
        raise ValueError("destination")
    web_source = web_source or ROOT
    web_revision = revision(web_source, web_commit)
    launcher_revision = revision(ROOT)
    manifest = checked_blob(web_source, web_revision, "web/publico.manifest")
    names = manifest.decode().splitlines()
    if not names or len(set(names)) != len(names) or any(
            not n.startswith("static/") or ".." in n or "\\" in n for n in names):
        raise ValueError("public manifest")
    inventory = {"produccion.manifest": hashlib.sha256(manifest).hexdigest()}
    files = {"web/produccion.manifest": manifest}
    for name in names:
        data = checked_blob(web_source, web_revision, "web/" + name)
        files["web/" + name] = data
        inventory[name] = hashlib.sha256(data).hexdigest()
    scripts = {}
    for name in ("runtime.py", "aprovisionar.py", "arrancar.sh", "comprobar.sh", "vec-publico.service"):
        data = checked_blob(ROOT, launcher_revision, "deploy/publico/" + name)
        files["deploy/publico/" + name] = data
        scripts[name] = hashlib.sha256(data).hexdigest()
    destination.mkdir(mode=0o755)
    for name, data in files.items():
        target = destination / name
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes(data)
    shutil.copyfile(binary, destination / "vec-publico")
    (destination / "vec-publico").chmod(0o755)
    with (destination / "vec-publico").open("rb") as stream:
        binary_hash = hashlib.file_digest(stream, "sha256").hexdigest()
    (destination / "artefacto.json").write_text(json.dumps({
        "web_source_commit": web_revision, "launcher_source_commit": launcher_revision,
        "binary_sha256": binary_hash, "web_sha256": inventory, "launcher_sha256": scripts,
    }, indent=2) + "\n")


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--binary", required=True, type=Path)
    parser.add_argument("--destination", required=True, type=Path)
    parser.add_argument("--web-source", type=Path)
    parser.add_argument("--web-commit", default="HEAD")
    args = parser.parse_args()
    package(args.binary, args.destination, args.web_source, args.web_commit)
