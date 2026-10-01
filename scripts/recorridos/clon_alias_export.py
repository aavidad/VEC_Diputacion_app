#!/usr/bin/env python3
"""Export aliases once, offline, from pinned synthetic material.

An interrupted attempt is never resumed. A completed attempt is verified with
its externally recorded receipt digest; replay never launches a process.
Direction must preserve the receipt digest emitted by the first successful
invocation in a separate act/channel. Replay requires that observed digest,
never a digest recomputed from files in the output package.
The material and output locations are pinned for this specific approved cut.
This receipt proves only the local export, not provisioning or runtime startup.
"""
from __future__ import annotations

import argparse
from contextlib import ExitStack
from datetime import datetime, timezone
import fcntl
import hashlib
import hmac
import importlib.util
import json
import os
from pathlib import Path
import resource
import re
import selectors
import signal
import stat
import subprocess
import sys
import time

sys.dont_write_bytecode = True
_SPEC = importlib.util.spec_from_file_location("vec_alias_material", Path(__file__).with_name("clon_material_externo_offline.py"))
material = importlib.util.module_from_spec(_SPEC)
_SPEC.loader.exec_module(material)

BASE = "268125ac2601f3dfce4306030cb91ea1dbf1fb7c"
COMMIT = "460e120c2ec9c2d4953d1e98bdba011403fe17e2"
TREE = "f5ebd9e67e8a78509263c6069888ffbc8ceab140"
BINARY_SHA = "84f6ba90d77b00ab33b739b9778f8515a0317e8568941eabe568555870246136"
BINARY_BYTES = 53263525
BUILD_ACT_SHA = "a448fcbe4c57aa5a26dd467ad99ee188fb5560ce71ac5e470f63f0ad840056da"
ACK_SHA = "a5bb87b1cc25cbf9bfd21d6a6eaa9d0e22c3675c7c2b33b443125a1bfe4ce30d"
MATERIAL_SHA = "d2f7c6b5e7253f1c300f4c34428e05e4c8f7099526ecaa322c285d3768b653a4"
# SHA256 of the canonical absolute locations agreed with Direction. Paths are
# private; a copied material tree or another output location cannot start an attempt.
MATERIAL_PATH_SHA = "42ff20c733761ddc80ba2acf75bd8c520284bc01cf7fb6d0c851a03eb4732a18"
OUTPUT_PATH_SHA = "fbab4fe36295b88d680dcea99022777d365cf7f6294698c913aec0e74e58759c"
COMMAND = "exportar-seudonimos-portal-externo"
PENDING = "alias-export.pending.json"
OUTPUT = "seudonimos-portal-externo.json"
RECEIPT = "alias-export.receipt.json"
MAX_OUTPUT = 256 << 10
MAX_STDERR = 64 << 10
GUARD = "ACEPTO_CREDENCIALES_NO_AUTORITATIVAS_SOLO_DESARROLLO"
DOMAIN = "idh_" + hashlib.sha256(b"vec.ct.alta.desarrollo.v1\0https://localhost/vec/desarrollo/identidad").hexdigest()[:32]


class ExportError(RuntimeError):
    """Public errors contain only fixed nominal codes."""


def require(condition, code):
    if not condition:
        raise ExportError(code)


def git_pins(repo: Path):
    env = {"HOME": "/nonexistent", "PATH": "/usr/bin:/bin", "GIT_CONFIG_NOSYSTEM": "1", "GIT_CONFIG_GLOBAL": "/dev/null"}
    def git(*args):
        result = subprocess.run(["/usr/bin/git", "-c", "core.fsmonitor=false", "-c", "core.hooksPath=/dev/null", "-C", str(repo), *args],
                                env=env, stdin=subprocess.DEVNULL, stdout=subprocess.PIPE,
                                stderr=subprocess.DEVNULL, timeout=10, check=False)
        require(result.returncode == 0 and len(result.stdout) <= 1 << 20, "git_pin_unavailable")
        return result.stdout
    # The build and validators remain pinned to their original objects. Main
    # may advance while a completed export is replayed, but it must retain the
    # accredited source commit. Resolve once so ancestry checks use one snapshot.
    main = git("rev-parse", "--verify", "origin/main^{commit}").strip()
    require(re.fullmatch(rb"[0-9a-f]{40}", main) is not None, "git_main_pin_changed")
    git("merge-base", "--is-ancestor", COMMIT, main.decode("ascii"))
    require(git("rev-parse", COMMIT + "^{tree}").strip() == TREE.encode(), "git_tree_pin_changed")
    git("merge-base", "--is-ancestor", BASE, "HEAD")
    # The validators are trusted only as the exact reviewed siblings of this cut.
    for name in ("clon_material_externo_offline.py", "clon_fuente_acreditada.py"):
        path = Path(__file__).with_name(name)
        require(not path.is_symlink() and path.read_bytes() == git("show", BASE + ":scripts/recorridos/" + name), "validator_pin_changed")


def binary_bytes(path: Path):
    root, identities = material.open_root(path.parent)
    try:
        fd = os.open(path.name, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK, dir_fd=root)
        try:
            before = os.fstat(fd)
            require(stat.S_ISREG(before.st_mode) and stat.S_IMODE(before.st_mode) == 0o700 and
                    before.st_uid == os.getuid() and before.st_nlink == 1 and before.st_size == BINARY_BYTES,
                    "binary_file_invalid")
            chunks = []
            remaining = BINARY_BYTES + 1
            while remaining:
                data = os.read(fd, min(1 << 20, remaining))
                if not data:
                    break
                chunks.append(data)
                remaining -= len(data)
            data = b"".join(chunks)
            after = os.fstat(fd)
            fields = ("st_dev", "st_ino", "st_mode", "st_uid", "st_nlink", "st_size", "st_mtime_ns", "st_ctime_ns")
            require(all(getattr(before, field) == getattr(after, field) for field in fields) and
                    len(data) == BINARY_BYTES and material.digest(data) == BINARY_SHA, "binary_pin_changed")
            material.revalidate(path.parent, identities)
            return data
        finally:
            os.close(fd)
    finally:
        os.close(root)


def preflight(repo: Path, binary: Path, source: Path, ack: Path, root: Path, now: datetime):
    """Read-only: no output directory, random reservation or process is created."""
    git_pins(repo)
    executable = binary_bytes(binary)
    act = material.read_private(binary.with_name("ACTA.txt"))
    require(material.digest(act) == BUILD_ACT_SHA and
            ("Commit fuente: " + COMMIT).encode() in act and ("Binario SHA256: " + BINARY_SHA).encode() in act,
            "build_act_pin_changed")
    source_data, ack_data = material.read_private(source), material.read_private(ack)
    require(material.digest(ack_data) == ACK_SHA, "ack_pin_changed")
    bound = material.binding(source_data, ack_data, now)
    fd, identities = material.open_root(root)
    try:
        require(material.inventory(fd) == material.FILES | {material.PENDING, material.RECEIPT, material.LOCK}, "material_inventory_changed")
        lock_fd = os.open(material.LOCK, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK, dir_fd=fd)
        try:
            material.file_info(lock_fd, empty=True)
        finally:
            os.close(lock_fd)
        receipt_data = material.read_at(fd, material.RECEIPT)
        require(material.digest(receipt_data) == MATERIAL_SHA, "material_receipt_pin_changed")
        require(material.read_at(fd, material.PENDING) == material.encoded(bound), "material_pending_changed")
        files = {name: material.read_at(fd, name) for name in material.FILES}
        fingerprints = material.verify_material(files, bound, now)
        require(receipt_data == material.encoded(material.receipt_for(bound, files, fingerprints)), "material_bytes_changed")
        material.revalidate(root, identities)
    finally:
        os.close(fd)
    return executable, files, bound


def expected_account(files: dict, bound: dict):
    # Go SeudonimizarAlta uses the newest huella-solicitud key and these exact
    # NUL-separated preimages. This comparison cannot attest process execution.
    key = files["runtime-externo/idempotencia/g2-huella-solicitud.bin"]
    def mac(label, value):
        preimage = ("vec.identidad.desarrollo.hmac.v1\0https://localhost/vec/desarrollo/identidad\0" + label + "\0" + value).encode()
        return hmac.new(key, preimage, hashlib.sha256).hexdigest()
    return {"cuenta_ref": bound["refs"]["cuenta"], "esquema": "vec.identidad.hmac-sha256.v1", "dominio_ref": DOMAIN,
            "clave_id": "vec.identidad.desarrollo.externo.g2", "clave_version": 2,
            "cuenta_id_hmac": mac("cuenta", "desarrollo:" + bound["refs"]["cuenta"]),
            "sujeto_id_hmac": mac("sujeto", bound["sujeto"])}


def validate_stdout(data: bytes, files: dict, bound: dict):
    require(0 < len(data) <= MAX_OUTPUT, "stdout_size_invalid")
    value = material.decoded(data)
    require(set(value) == {"version", "cuentas"} and type(value["version"]) is int and value["version"] == 1 and
            isinstance(value["cuentas"], list) and len(value["cuentas"]) == 1, "stdout_schema_invalid")
    account = value["cuentas"][0]
    expected = expected_account(files, bound)
    require(isinstance(account, dict) and set(account) == set(expected) and
            type(account["clave_version"]) is int and account == expected and
            account["cuenta_id_hmac"] != account["sujeto_id_hmac"], "stdout_account_invalid")
    return account


def sealed(data: bytes, stack: ExitStack):
    fd = os.memfd_create("vec-offline-input", os.MFD_CLOEXEC | os.MFD_ALLOW_SEALING)
    stack.callback(os.close, fd)
    view = memoryview(data)
    while view:
        written = os.write(fd, view)
        require(written > 0, "snapshot_write_failed")
        view = view[written:]
    os.lseek(fd, 0, os.SEEK_SET)
    fcntl.fcntl(fd, fcntl.F_ADD_SEALS, fcntl.F_SEAL_WRITE | fcntl.F_SEAL_GROW | fcntl.F_SEAL_SHRINK | fcntl.F_SEAL_SEAL)
    return fd


def sandbox_command(executable: bytes, files: dict, runtime: Path, stack: ExitStack):
    require(Path("/usr/bin/bwrap").is_file(), "offline_sandbox_unavailable")
    fds = []
    argv = ["/usr/bin/bwrap", "--unshare-all", "--die-with-parent", "--new-session", "--cap-drop", "ALL"]
    for path in ("/usr", "/lib", "/lib64"):
        if Path(path).exists():
            argv += ["--ro-bind", path, path]
    binary_fd = sealed(executable, stack)
    fds.append(binary_fd)
    argv += ["--perms", "0700", "--ro-bind-data", str(binary_fd), "/vec-server", "--proc", "/proc", "--dev", "/dev"]
    directories = {runtime, *runtime.parents}
    for name in material.RUNTIME_FILES:
        directories.update((runtime / name).parents)
    for directory in sorted(directories - {Path("/")}, key=lambda p: (len(p.parts), str(p))):
        argv += ["--perms", "0700", "--dir", str(directory)]
    for name in sorted(material.RUNTIME_FILES):
        fd = sealed(files[material.RUNTIME + name], stack)
        fds.append(fd)
        argv += ["--perms", "0600", "--ro-bind-data", str(fd), str(runtime / name)]
    argv += ["--dir", "/home/vec", "--remount-ro", "/", "--size", "1048576", "--tmpfs", "/home/vec", "--chdir", "/home/vec"]
    env = {"HOME": "/home/vec", "PATH": "/usr/bin:/bin", "VEC_PORTAL_PROCESO": "externo",
           "VEC_EXECUTION_PROFILE": "desarrollo", "VEC_AUTH_MODE": "desarrollo",
           "VEC_DEVELOPMENT_GUARD": GUARD, "VEC_DEVELOPMENT_MATERIAL_DIR": str(runtime)}
    # Apply the process bound inside the new user/PID namespaces, so existing
    # host processes cannot prevent the trusted sandbox supervisor from starting.
    argv += ["/usr/bin/prlimit", "--nproc=64", "--fsize=" + str(MAX_OUTPUT), "--", "/usr/bin/env", "-i",
             *[key + "=" + value for key, value in env.items()], "/vec-server", COMMAND]
    return argv, tuple(fds), env


def process_limits():
    for limit, value in ((resource.RLIMIT_CORE, 0), (resource.RLIMIT_FSIZE, 64 << 20),
                         (resource.RLIMIT_CPU, 15), (resource.RLIMIT_AS, 2 << 30),
                         (resource.RLIMIT_NOFILE, 128)):
        resource.setrlimit(limit, (value, value))


def execute(argv: list, fds: tuple, output_fd: int, timeout: float):
    started = datetime.now(timezone.utc).isoformat()
    proc = subprocess.Popen(argv, env={}, stdin=subprocess.DEVNULL, stdout=output_fd, stderr=subprocess.PIPE,
                            pass_fds=fds, start_new_session=True, preexec_fn=process_limits)
    count = 0
    deadline = time.monotonic() + timeout
    selector = selectors.DefaultSelector()
    try:
        selector.register(proc.stderr, selectors.EVENT_READ)
        while selector.get_map():
            remaining = deadline - time.monotonic()
            require(remaining > 0, "export_timeout")
            for key, _ in selector.select(min(remaining, 0.1)):
                data = os.read(key.fileobj.fileno(), 4096)
                if not data:
                    selector.unregister(key.fileobj)
                else:
                    count += len(data)
                    require(count <= MAX_STDERR, "stderr_limit_exceeded")
        exit_code = proc.wait(timeout=max(0.001, deadline - time.monotonic()))
        require(exit_code == 0, "export_exit_nonzero")
        return {"pid": proc.pid, "exit_code": exit_code, "iniciado_en": started,
                "terminado_en": datetime.now(timezone.utc).isoformat(), "stderr_bytes": count, "stderr": "redacted"}
    finally:
        selector.close()
        proc.stderr.close()
        # Kill the whole attempt group even if the immediate process exited.
        try:
            os.killpg(proc.pid, signal.SIGKILL)
        except ProcessLookupError:
            pass
        proc.wait()


def validate_process(value: object):
    require(isinstance(value, dict) and set(value) == {
        "pid", "exit_code", "iniciado_en", "terminado_en", "stderr_bytes", "stderr"}, "replay_process_invalid")
    require(type(value["pid"]) is int and 0 < value["pid"] < 1 << 31 and
            type(value["exit_code"]) is int and value["exit_code"] == 0 and
            type(value["stderr_bytes"]) is int and 0 <= value["stderr_bytes"] <= MAX_STDERR and
            value["stderr"] == "redacted", "replay_process_invalid")
    dates = []
    for key in ("iniciado_en", "terminado_en"):
        stamp = value[key]
        require(isinstance(stamp, str) and re.fullmatch(
            r"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(?:\.[0-9]{6})?\+00:00", stamp) is not None,
            "replay_process_timestamp_invalid")
        try:
            dates.append(datetime.fromisoformat(stamp))
        except ValueError:
            raise ExportError("replay_process_timestamp_invalid") from None
    require(dates[0] <= dates[1], "replay_process_time_order_invalid")


def export(repo: Path, binary: Path, source: Path, ack: Path, root: Path, output: Path,
           *, receipt_sha: str | None = None, timeout: float = 30, now: datetime | None = None):
    now = now or datetime.now(timezone.utc)
    require(now.tzinfo == timezone.utc and type(timeout) in (int, float) and 0 < timeout <= 60, "execution_limits_invalid")
    require(all(p.is_absolute() and ".." not in p.parts for p in (repo, binary, source, ack, root, output)), "path_invalid")
    require(material.digest(str(root).encode()) == MATERIAL_PATH_SHA and
            str(root) == str(root.resolve(strict=True)), "material_route_pin_changed")
    require(material.digest(str(output).encode()) == OUTPUT_PATH_SHA and
            str(output) == str(output.resolve(strict=False)), "output_route_pin_changed")
    for protected in (repo, binary.parent, source.parent, ack.parent, root):
        require(output != protected and output not in protected.parents and protected not in output.parents, "output_overlaps_input")
    executable, files, bound = preflight(repo, binary, source, ack, root, now)
    binding = {"source_commit": COMMIT, "source_tree": TREE, "binary_sha256": BINARY_SHA, "binary_bytes": BINARY_BYTES,
               "build_act_sha256": BUILD_ACT_SHA, "fuente_sha256": material.SOURCE_SHA, "acuse_sha256": ACK_SHA,
               "material_acta_sha256": MATERIAL_SHA, "runtime": str(root / "runtime-externo"),
               "exportador": "cmd/vec-server " + COMMAND}
    with ExitStack() as stack:
        argv, fds, environment = sandbox_command(executable, files, root / "runtime-externo", stack)
        fd, identities = material.open_root(output, create=True)
        stack.callback(os.close, fd)
        material.revalidate(output, identities)
        names = set(os.listdir(fd))
        if names:
            require(names == {PENDING, OUTPUT, RECEIPT}, "attempt_incomplete_or_unexpected")
            require(isinstance(receipt_sha, str) and re.fullmatch(r"[0-9a-f]{64}", receipt_sha) is not None, "replay_receipt_pin_required")
            raw_receipt = material.read_at(fd, RECEIPT)
            require(material.digest(raw_receipt) == receipt_sha, "replay_receipt_changed")
            receipt = material.decoded(raw_receipt)
            require(set(receipt) == {"version", "kind", "binding", "pending_sha256", "salida_sha256", "salida_bytes", "proceso", "entorno"} and
                    type(receipt["version"]) is int and receipt["version"] == 1 and receipt["kind"] == "exportacion_seudonimos_externos_ejecutada_v1" and
                    receipt["binding"] == binding and receipt["entorno"] == environment, "replay_binding_changed")
            validate_process(receipt["proceso"])
            require(type(receipt["salida_bytes"]) is int and 0 < receipt["salida_bytes"] <= MAX_OUTPUT, "replay_stdout_size_invalid")
            pending_data = material.read_at(fd, PENDING)
            pending = material.decoded(pending_data)
            require(set(pending) == {"version", "kind", "binding", "nonce"} and type(pending["version"]) is int and pending["version"] == 1 and
                    isinstance(pending["nonce"], str) and re.fullmatch(r"[0-9a-f]{64}", pending["nonce"]) is not None and
                    pending["kind"] == "exportacion_seudonimos_externos_pending_v1" and pending["binding"] == binding and
                    material.digest(pending_data) == receipt["pending_sha256"], "replay_pending_changed")
            data = material.read_at(fd, OUTPUT)
            require(material.digest(data) == receipt["salida_sha256"] and len(data) == receipt["salida_bytes"], "replay_stdout_changed")
            validate_stdout(data, files, bound)
            material.revalidate(output, identities)
            return receipt
        require(receipt_sha is None, "replay_attempt_absent")
        pending = {"version": 1, "kind": "exportacion_seudonimos_externos_pending_v1", "binding": binding, "nonce": os.urandom(32).hex()}
        pending_data = material.encoded(pending)
        material.write_at(fd, PENDING, pending_data)
        material.revalidate(output, identities)
        out_fd = os.open(OUTPUT, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600, dir_fd=fd)
        stack.callback(os.close, out_fd)
        os.fchmod(out_fd, 0o600)
        os.fsync(fd)
        try:
            result = execute(argv, fds, out_fd, timeout)
            validate_process(result)
        finally:
            os.fsync(out_fd)
        material.revalidate(output, identities)
        stdout_info = material.file_info(out_fd)
        named_info = os.stat(OUTPUT, dir_fd=fd, follow_symlinks=False)
        require((stdout_info.st_dev, stdout_info.st_ino) == (named_info.st_dev, named_info.st_ino), "stdout_file_changed")
        data = material.read_at(fd, OUTPUT)
        validate_stdout(data, files, bound)
        # Recheck the original pins after execution; changes preserve the pending attempt.
        preflight(repo, binary, source, ack, root, datetime.now(timezone.utc))
        receipt = {"version": 1, "kind": "exportacion_seudonimos_externos_ejecutada_v1", "binding": binding,
                   "pending_sha256": material.digest(pending_data), "salida_sha256": material.digest(data),
                   "salida_bytes": len(data), "proceso": result, "entorno": environment}
        require(material.read_at(fd, PENDING) == pending_data and set(os.listdir(fd)) == {PENDING, OUTPUT}, "attempt_changed")
        material.write_at(fd, RECEIPT, material.encoded(receipt))
        material.revalidate(output, identities)
        require(material.read_at(fd, RECEIPT) == material.encoded(receipt) and material.read_at(fd, OUTPUT) == data, "receipt_postimage_changed")
        return receipt


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ("binario", "fuente", "acuse", "material", "salida"):
        parser.add_argument("--" + name, type=Path, required=True)
    parser.add_argument("--replay-receipt-sha256", help="Digest emitted on first success and preserved by Direction in a separate act/channel; never recompute from the package.")
    args = parser.parse_args()
    try:
        result = export(Path(__file__).resolve().parents[2], args.binario, args.fuente, args.acuse, args.material, args.salida,
                        receipt_sha=args.replay_receipt_sha256)
    except (ExportError, material.OfflineMaterialError, OSError, ValueError, TypeError, KeyError, RuntimeError, subprocess.SubprocessError):
        # Neither child stderr nor exceptions containing paths/inputs are emitted.
        print(json.dumps({"error": "alias_export_rejected", "stderr": "redacted"}), file=sys.stderr)
        return 2
    print(json.dumps({"estado": "exportacion_offline_verificada", "receipt_sha256": material.digest(material.encoded(result)),
                      "salida_sha256": result["salida_sha256"], "salida_bytes": result["salida_bytes"]}, sort_keys=True))
    return 0


if __name__ == "__main__":
    sys.exit(main())
