#!/usr/bin/env python3
"""Copia un binario aprobado y sólo los recursos del manifiesto público."""
import argparse
import hashlib
import json
from pathlib import Path
import shutil
import subprocess

ROOT = Path(__file__).resolve().parents[2]


def package(binary, destination):
    if destination.exists() or not destination.is_absolute() or binary.is_symlink():
        raise ValueError("destination")
    manifest = (ROOT / "web/publico.manifest").read_bytes()
    names = manifest.decode().splitlines()
    if not names or len(set(names)) != len(names) or any(
            not n.startswith("static/") or ".." in n or "\\" in n for n in names):
        raise ValueError("public manifest")
    destination.mkdir(mode=0o755)
    web = destination / "web"
    web.mkdir()
    inventory = {"produccion.manifest": hashlib.sha256(manifest).hexdigest()}
    (web / "produccion.manifest").write_bytes(manifest)
    for name in names:
        source = ROOT / "web" / name
        if source.is_symlink() or not source.is_file():
            raise ValueError("public resource")
        target = web / name
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes(source.read_bytes())
        inventory[name] = hashlib.sha256(target.read_bytes()).hexdigest()
    shutil.copyfile(binary, destination / "vec-publico")
    (destination / "vec-publico").chmod(0o755)
    launcher = destination / "deploy/publico"
    launcher.mkdir(parents=True)
    scripts = {}
    for name in ("runtime.py", "aprovisionar.py", "arrancar.sh", "comprobar.sh", "vec-publico.service"):
        data = (ROOT / "deploy/publico" / name).read_bytes()
        (launcher / name).write_bytes(data)
        scripts[name] = hashlib.sha256(data).hexdigest()
    with (destination / "vec-publico").open("rb") as stream:
        binary_hash = hashlib.file_digest(stream, "sha256").hexdigest()
    commit = subprocess.check_output(["git", "-C", str(ROOT), "rev-parse", "HEAD"], text=True).strip()
    (destination / "artefacto.json").write_text(json.dumps({
        "web_source_commit": commit, "binary_sha256": binary_hash,
        "web_sha256": inventory, "launcher_sha256": scripts,
    }, indent=2) + "\n")


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--binary", required=True, type=Path)
    parser.add_argument("--destination", required=True, type=Path)
    args = parser.parse_args()
    package(args.binary, args.destination)
