#!/usr/bin/env python3
"""Compose one H6 archive attempt. Real execution lacks definitive authority.

The operational gate is deliberately absent. CLI pins cannot supply authority;
only isolated unit tests replace that gate with synthetic approval. Preparation
and restart remain blocked in the main orchestrator. Failed attempts retain all
snapshots, Docker resources and publication evidence; no retry or cleanup occurs.
"""
from __future__ import annotations

import argparse
from dataclasses import dataclass
import fcntl
import json
import os
from pathlib import Path
import stat
import sys
import tarfile
import subprocess

import clon_h6_archive as archive
import clon_h6_canario as validation
import clon_h6_docker_archive_driver as transport

ATTEMPT = "canario-archivo-intento.json"
WORK = "canario-archivo-evidencia"
CANONICAL = (validation.PENDING, "plan-conexiones.json", "plan-canonico-clon.json")
# No file, argument, environment variable or legacy receipt can open this gate.
DEFINITIVE_AUTHORITY = None


@dataclass(frozen=True)
class Request:
    state: Path
    pgid: str
    pg_image: str
    image: str
    paths: tuple[tuple[str, Path], ...]
    approved: tuple[tuple[str, str], ...]


@dataclass(frozen=True)
class Authority:
    """Internal contract only; no production provider is accredited or installed."""
    request_sha256: str
    trees_sha256: tuple[tuple[str, str], ...]
    pg_mounts_sha256: str


def request_bytes(request):
    archive.require(isinstance(request, Request), "request_contract")
    paths, approved = dict(request.paths), dict(request.approved)
    archive.require(len(paths) == len(request.paths) and len(approved) == len(request.approved)
                    and set(paths) == set(validation.FILES + validation.TREES)
                    and set(approved) == set(validation.FILES), "input_set")
    for path in (request.state, *paths.values()):
        archive.require(isinstance(path, Path) and path.is_absolute() and ".." not in path.parts
                        and path != Path("/"), "input_path")
    archive.require(len(set(paths.values())) == len(paths), "input_paths_duplicate")
    archive.require(archive.HEX.fullmatch(request.pgid) and archive.IMAGE.fullmatch(request.pg_image)
                    and archive.IMAGE.fullmatch(request.image)
                    and all(archive.HEX.fullmatch(pin) for pin in approved.values()), "input_pins")
    return validation.canonical({"state": str(request.state), "pgid": request.pgid,
        "pg_image": request.pg_image, "image": request.image,
        "paths": {name: str(path) for name, path in paths.items()}, "approved": approved})


def require_authority(request):
    # This check must precede even parsing/inspection of state and lock creation.
    authority = DEFINITIVE_AUTHORITY
    archive.require(isinstance(authority, Authority), "fresh_definitive_material_authority_missing")
    archive.require(archive.HEX.fullmatch(authority.request_sha256)
                    and archive.HEX.fullmatch(authority.pg_mounts_sha256)
                    and validation.sha(request_bytes(request)) == authority.request_sha256,
                    "definitive_authority_binding")
    trees = dict(authority.trees_sha256)
    archive.require(len(trees) == len(authority.trees_sha256)
                    and set(trees) == set(validation.TREES)
                    and all(archive.HEX.fullmatch(pin) for pin in trees.values()), "tree_authority")
    return authority


def config_for(request):
    return validation.Config(request.state, request.state / WORK / "export",
        request.pgid, request.image, dict(request.paths), dict(request.approved))


def tree_digests(snapshot):
    return {name: validation.sha(validation.canonical({relative: None if relative.endswith("/")
        else value[1] for relative, value in snapshot[name].items()})) for name in validation.TREES}


def freeze_inputs(config, authority):
    """Bounded verified byte snapshots before any state or Docker effect."""
    before, contents = validation.snapshot(config)
    archive.require(tree_digests(before) == dict(authority.trees_sha256), "tree_inventory_changed")
    entries, directories, total = [], [], 0
    for name in ("inspector", "connections", "resolver", "arr", "reference", "ca"):
        entries.append(archive.ApprovedInput("files/" + name, config.paths[name], config.approved[name],
            executable=name in ("inspector", "resolver")))
    for name in validation.TREES:
        prefix = "trees/" + name
        directories.append(prefix)
        for relative, value in before[name].items():
            if relative == "/":
                continue
            if relative.endswith("/"):
                directories.append(prefix + "/" + relative[:-1])
            else:
                entries.append(archive.ApprovedInput(prefix + "/" + relative,
                    config.paths[name] / relative, value[1]))
    archive.require(len(entries) + len(directories) <= archive.MEMBER_LIMIT, "input_count")
    frozen = []
    for entry in entries:
        archive.closed_name(entry.name)
        data = archive.approved_bytes(entry)
        total += len(data)
        archive.require(total <= archive.INPUT_LIMIT, "input_limit")
        frozen.append((entry.name, data, entry.executable))
    archive.require(validation.snapshot(config)[0] == before, "inputs_changed")
    return before, contents, tuple(frozen), tuple(directories)


def write_at(directory, name, data):
    fd = os.open(name, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW | os.O_CLOEXEC,
                 0o600, dir_fd=directory)
    try:
        remaining = memoryview(data)
        while remaining:
            written = os.write(fd, remaining)
            archive.require(written > 0, "evidence_write")
            remaining = remaining[written:]
        os.fsync(fd)
    finally:
        os.close(fd)
    os.fsync(directory)


def state_identity(state, retained):
    with validation.directory(state) as fd:
        info = os.fstat(fd)
        archive.require((info.st_dev, info.st_ino) == retained
                        and info.st_uid == os.getuid() and stat.S_IMODE(info.st_mode) == 0o700,
                        "state_replaced")


def private_work(state_fd, state, frozen):
    os.mkdir(WORK, 0o700, dir_fd=state_fd)
    os.fsync(state_fd)
    work = state / WORK
    with validation.directory(work) as fd:
        for name in ("snapshots", "helpers", "source", "destination", "export", "docker-config"):
            os.mkdir(name, 0o700, dir_fd=fd)
        os.fsync(fd)
    # Flat host snapshots are not mounted. Archive names preserve approved layout.
    entries = []
    with validation.directory(work / "snapshots") as fd:
        for index, (name, data, executable) in enumerate(frozen):
            leaf = str(index)
            write_at(fd, leaf, data)
            entries.append(archive.ApprovedInput(name, work / "snapshots" / leaf,
                                               validation.sha(data), executable=executable))
    return work, tuple(entries)


def pg_identity(config, request, authority, cli):
    inspected = []
    def runner(argv):
        archive.require(argv[:4] == ["docker", "inspect", "--type", "container"], "pg_inspect_only")
        payload = cli.execute(argv[1:])
        inspected.extend(transport.decode(payload))
        return payload
    original = validation.pg_identity(config, runner)
    data = inspected[0]
    mounts = data.get("Mounts")
    archive.require(data.get("Id") == request.pgid and data.get("Image") == request.pg_image
                    and data.get("Config", {}).get("Image") == request.pg_image
                    and isinstance(mounts, list) and mounts
                    and all(m.get("Type") in ("bind", "volume") and type(m.get("RW")) is bool
                            and isinstance(m.get("Source"), str) and m["Source"].startswith("/")
                            for m in mounts)
                    and validation.sha(validation.canonical(mounts)) == authority.pg_mounts_sha256,
                    "pg_physical_identity")
    archive.require(not data.get("HostConfig", {}).get("PortBindings"), "pg_host_ports")
    return original, validation.sha(validation.canonical(mounts))


def volume_mounts(config, session):
    result = []
    for source, destination, writable in validation.mounts(config):
        if writable:
            subpath = "output"
        else:
            name = next(key for key, path in config.paths.items() if path == source)
            subpath = ("trees/" if name in validation.TREES else "files/") + name
            subpath = "input/" + subpath
        result.append(archive.VolumeMount(session.volume, destination, writable, subpath))
    return tuple(result)


def trusted_pair(config, work, contents, plan, digest):
    parsed = validation.decode(plan)
    archive.require(isinstance(parsed, dict) and set(parsed) == {"conexiones", "huellas"}
                    and isinstance(parsed["conexiones"], list) and isinstance(parsed["huellas"], dict)
                    and parsed["huellas"], "output_json")
    scripts, source, destination = (work / name for name in ("helpers", "source", "destination"))
    for key in ("connections", "promoter", "receipt_helper"):
        validation.write(scripts / validation.HELPERS[key], contents[key])
    validation.write(work / "lock", contents["lock"])
    validation.write(source / "plan.json", plan)
    validation.run(validation.helper_command(scripts, source, destination, work / "lock",
        validation.HELPERS["promoter"], ["/source", "/destination"]))
    archive.require(os.listdir(source) == [] and os.listdir(destination) == ["plan-conexiones.json"],
                    "promotion_entries")
    promoted, promoted_sha, _ = validation.read(destination / "plan-conexiones.json", 131072,
                                               owner=os.getuid(), modes=(0o600,))
    archive.require(promoted == plan and promoted_sha == digest, "promotion_changed")
    validation.run(validation.helper_command(scripts, source, destination, work / "lock",
        validation.HELPERS["receipt_helper"], ["clon", "/destination/plan-conexiones.json", digest,
            config.pgid, config.image, config.approved["arr"], "/release.lock",
            "/destination/plan-canonico-clon.json"]))
    archive.require(sorted(os.listdir(destination)) == ["plan-canonico-clon.json", "plan-conexiones.json"],
                    "receipt_entries")
    receipt, _, _ = validation.read(destination / "plan-canonico-clon.json", 8192,
                                    owner=os.getuid(), modes=(0o600,))
    expected = {"version": 1, "kind": "clon", "package_sha256": config.approved["package"],
        "source_commit": validation.SOURCE, "pg_container_id": config.pgid, "plan_sha256": digest,
        "material_inventory_sha256": validation.sha(validation.canonical(parsed["huellas"])),
        "canary_image_id": config.image, "arranque_sha256": config.approved["arr"],
        "canary_output_sha256": digest}
    archive.require(receipt == validation.canonical(expected), "receipt_binding")
    return receipt


def canario_archivo(request):
    authority = require_authority(request)
    config = config_for(request)
    before, contents, frozen, directories = freeze_inputs(config, authority)
    with validation.directory(request.state) as state_fd:
        info = os.fstat(state_fd)
        archive.require(info.st_uid == os.getuid() and stat.S_IMODE(info.st_mode) == 0o700,
                        "state_private")
        archive.require(not any((parent / ".git").exists()
                        for parent in (request.state, *request.state.parents)), "state_in_git")
        retained = (info.st_dev, info.st_ino)
        fcntl.flock(state_fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        archive.require(not set((ATTEMPT, WORK, *CANONICAL)) & set(os.listdir(state_fd)),
                        "archive_attempt_exists_new_state_required")
        # An incomplete/fsync-failed attempt is never removed or replayed.
        write_at(state_fd, ATTEMPT, validation.canonical({"version": 1, "requires_new_state": True,
            "authority_sha256": authority.request_sha256,
            "inputs_sha256": validation.sha(validation.canonical([
                [name, validation.sha(data), executable] for name, data, executable in frozen])),
            "pgid": request.pgid, "pg_image": request.pg_image, "canary_image": request.image}))
        state_identity(request.state, retained)
        work, entries = private_work(state_fd, request.state, frozen)
        cli = transport.LocalDockerCLI(work / "docker-config")
        pg_before = pg_identity(config, request, authority, cli)
        state_identity(request.state, retained)
        session = transport.ArchiveSession(request.state, request.image, cli=cli)
        session.create_staging()
        archive.require(session.staging.id != request.pgid, "pg_staging_collision")
        session.stage(entries, directories=directories)
        session.create_canary(volume_mounts(config, session))
        archive.require(session.canary.id != request.pgid, "pg_canary_collision")
        session.start_canary()
        digest = session.export_output(work / "export")
        plan, observed, metadata = validation.read(work / "export" / "plan.json", 131072,
                                                  owner=os.getuid(), modes=(0o600,))
        archive.require(digest == observed, "export_changed")
        receipt = trusted_pair(config, work, contents, plan, digest)
        archive.require(require_authority(request) == authority
                        and validation.snapshot(config)[0] == before, "inputs_changed")
        archive.require(pg_identity(config, request, authority, cli) == pg_before, "pg_changed")
        session.driver.validate_canary(session.canary, exited=True)
        final_plan, final_sha, final_metadata = validation.read(work / "export" / "plan.json", 131072,
                                                               owner=os.getuid(), modes=(0o600,))
        archive.require((final_plan, final_sha, final_metadata) == (plan, digest, metadata), "export_changed")
        state_identity(request.state, retained)
        validation.publish_pair(state_fd, plan, receipt)
        # Retain containers/volume even on success: retirement is a separate authority.
        return {"kind": "clon", "canary_exit_code": 0, "plan_sha256": digest,
                "receipt_sha256": validation.sha(receipt), "pg_container_id": request.pgid,
                "canary_image_id": request.image}


class ClosedParser(argparse.ArgumentParser):
    def error(self, message):
        raise archive.Refused("arguments_invalid")


def arguments(argv):
    parser = ClosedParser(allow_abbrev=False)
    parser.add_argument("--state", required=True, type=Path)
    for name in ("pgid", "pg-image", "image"):
        parser.add_argument("--" + name, required=True)
    for name in validation.FILES + validation.TREES:
        parser.add_argument("--" + name.replace("_", "-"), required=True, type=Path)
    for name in validation.FILES:
        parser.add_argument("--" + name.replace("_", "-") + "-sha256", required=True)
    # argparse normally accepts repeated options with last-one-wins semantics.
    options = [value.split("=", 1)[0] for value in argv if value.startswith("--")]
    archive.require(len(options) == len(set(options)), "arguments_duplicate")
    args = parser.parse_args(argv)
    return Request(args.state, args.pgid, args.pg_image, args.image,
        tuple((name, getattr(args, name)) for name in validation.FILES + validation.TREES),
        tuple((name, getattr(args, name + "_sha256")) for name in validation.FILES))


def main(argv=None):
    try:
        result = canario_archivo(arguments(sys.argv[1:] if argv is None else argv))
        print(json.dumps(result, sort_keys=True))
        return 0
    except (archive.Refused, validation.Refused, OSError, ValueError, KeyError, TypeError,
            tarfile.TarError, subprocess.SubprocessError) as error:
        code = "fresh_definitive_material_authority_missing" if (
            isinstance(error, archive.Refused) and str(error) ==
            "fresh_definitive_material_authority_missing") else "h6_archive_canary_unverified"
        print(json.dumps({"status": "refused", "code": code}), file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
