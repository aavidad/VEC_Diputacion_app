#!/usr/bin/env python3
"""Recibo AD3-132 del clon Docker con las autoridades originales del paquete D.

Contrato del orquestador: Request fija tar/raíz extraída, SHA aprobados de
paquete, lock, release, fuente, plan y aprobación privada previa, contenedor,
plan/recibo canónico del canario y destinos privados de pending/salida/recibo.
La raíz debe conservar el árbol y los modos del tar aprobado. No se extrae aquí.

apply_and_confirm(request) es la única confirmación. El adaptador crea y sincroniza
su propio pending canónico antes de ejecutar la CLI apply Docker mediante
subprocess.run fijo. Observa directamente su código de retorno y stdout,
conserva su stdout exacto y emite el recibo D tras cotejar postimagen/anclas.
El pending se conserva también tras éxito; su presencia siempre impide reaplicar.
Revalidar exige esa evidencia y el recibo. Ningún callback o runner forma parte
de la API. Los dobles de tests se instalan con monkeypatch en el entorno aislado.

La reapertura de CLI por el helper D se protege validando UID actual y ausencia
de escritura ajena en todos los archivos/directorios del paquete. Los ancestros
hasta / pertenecen al UID actual o root y tampoco admiten escritura ajena.
El UID operativo y root son autoridades confiables; un UID ajeno no puede
sustituir los bytes ni renombrar sus rutas entre verificación y ejecución.

revalidar(request) sólo lee el recibo y consulta inventario/anclas originales.
La CLI expone únicamente esta lectura. Una postimagen compatible sin recibo
confirmado no acredita aplicación. No se añaden consultas ni SQL propios.
"""

import argparse
from dataclasses import dataclass
import hashlib
import io
import json
import os
from pathlib import Path
import re
import stat
import subprocess
import sys
import tarfile
import types

try:
    from . import clon_sql
except ImportError:
    import clon_sql

HEX64 = re.compile(r"[0-9a-f]{64}\Z")
HEX40 = re.compile(r"[0-9a-f]{40}\Z")
HELPER_REL = "h6_recibo_ad132.py"
CLI_REL = "deploy/postgresql/autorizacion_atestada_v3/aplicar_000132_temp_public.py"
SUCCESS = b"AD3-132-OK: PUBLIC TEMP retirado; LOGIN sin TEMP efectivo\n"
CANARY_PENDING = "canario-publicacion-pendiente.json"


class Refused(RuntimeError):
    pass


@dataclass(frozen=True)
class Request:
    package_tar: Path
    package_root: Path
    approved_package_sha256: str
    release_lock: Path
    approved_lock_sha256: str
    source_commit: str
    approved_release_sha256: str
    plan: Path
    plan_receipt: Path
    approved_plan_sha256: str
    approval: Path
    approval_sha256: str
    container: str
    receipt: Path
    apply_output: Path
    pending_path: Path


def digest(data):
    return hashlib.sha256(data).hexdigest()


def package_tree(data):
    """Inventario cerrado de bytes y modos; jamás extrae el tar."""
    files, directories, seen, size = {}, {}, set(), 0
    with tarfile.open(fileobj=io.BytesIO(data), mode="r:gz") as archive:
        for position, member in enumerate(archive, 1):
            name = member.name[2:] if member.name.startswith("./") else member.name
            if name in ("", ".") and member.isdir():
                if "" in directories or member.mode & 0o7022:
                    raise Refused("modo o entrada raíz del tar incompatible")
                directories[""] = member.mode & 0o777
                continue
            name = name.rstrip("/") if member.isdir() else name
            if (position > 20000 or not name or name.startswith("/") or "\\" in name
                    or any(part in ("", ".", "..") for part in name.split("/"))
                    or name in seen or not (member.isfile() or member.isdir())
                    or member.mode & 0o7022):
                raise Refused("árbol del paquete incompatible")
            seen.add(name)
            for parent in Path(name).parents:
                if str(parent) != ".":
                    directories.setdefault(str(parent), None)
            if member.isdir():
                directories[name] = member.mode & 0o777
                continue
            size += member.size
            if member.size > 128 * 1024 * 1024 or size > 512 * 1024 * 1024:
                raise Refused("paquete excede límites")
            with archive.extractfile(member) as source:
                content = source.read(member.size + 1)
            if len(content) != member.size:
                raise Refused("paquete incompleto")
            files[name] = (digest(content), member.mode & 0o777)
    if not files or files.keys() & directories.keys():
        raise Refused("árbol del paquete ambiguo")
    return files, directories


def trusted_directory(path, owner=None, mode=None):
    status = path.lstat()
    owners = {0, os.getuid()} if owner is None else {owner}
    if (not stat.S_ISDIR(status.st_mode) or status.st_uid not in owners
            or status.st_mode & 0o7022
            or (mode is not None and stat.S_IMODE(status.st_mode) != mode)):
        raise Refused("directorio con propietario, modo o tipo no confiable")


def trusted_ancestors(path):
    for parent in path.parents:
        trusted_directory(parent)


def verify_tree(root, files, directories):
    root = clon_sql.validate_original_path(root)
    trusted_ancestors(root)
    trusted_directory(root, os.getuid(), directories.get(""))
    found, found_dirs = set(), set()
    for directory, dirs, names in os.walk(root, followlinks=False):
        for name in dirs:
            path = Path(directory) / name
            relative = path.relative_to(root).as_posix()
            if relative not in directories:
                raise Refused("directorio ajeno al paquete aprobado")
            trusted_directory(path, os.getuid(), directories[relative])
            found_dirs.add(relative)
        for name in names:
            path = Path(directory) / name
            relative = path.relative_to(root).as_posix()
            if relative not in files:
                raise Refused("fichero ajeno al paquete aprobado")
            expected, mode = files[relative]
            clon_sql.approved_file(path, expected, 128 * 1024 * 1024, retain=False)
            status = path.lstat()
            if status.st_uid != os.getuid() or stat.S_IMODE(status.st_mode) != mode:
                raise Refused("dueño o modo extraído distinto del paquete")
            found.add(relative)
    if found != files.keys() or found_dirs != directories.keys() - {""}:
        raise Refused("árbol extraído incompleto o ajeno")
    return root


def private_destination(path):
    path = clon_sql.validate_original_path(path)
    trusted_ancestors(path)
    parent = path.parent.stat()
    if (not stat.S_ISDIR(parent.st_mode) or parent.st_uid != os.getuid()
            or parent.st_mode & 0o077
            or any((p / ".git").exists() for p in path.parents)):
        raise Refused("destino de evidencia fuera de directorio privado externo a Git")
    return path


def require_committed_canary(request):
    """Un par completo no confirma el canario mientras conserve su marcador."""
    plan = private_destination(request.plan)
    receipt = private_destination(request.plan_receipt)
    if (plan.name != "plan-conexiones.json" or receipt.name != "plan-canonico-clon.json"
            or plan.parent != receipt.parent):
        raise Refused("plan y recibo canónicos requieren el mismo directorio privado")
    pending = plan.parent / CANARY_PENDING
    try:
        pending.lstat()
    except FileNotFoundError:
        return
    except OSError as error:
        raise Refused("no se puede comprobar la publicación del canario") from error
    # lstat también detecta enlaces rotos, directorios y otros tipos inválidos.
    raise Refused("publicación del canario pendiente; conservar evidencia")


def load_adapter(request):
    """Preflight sin Docker: carga sólo el helper original fijado externamente."""
    require_committed_canary(request)
    for path in (request.package_tar, request.release_lock, request.plan,
                 request.plan_receipt, request.approval):
        trusted_ancestors(clon_sql.validate_original_path(path))
    for value in (request.approved_package_sha256, request.approved_lock_sha256,
                  request.approved_release_sha256, request.approved_plan_sha256,
                  request.approval_sha256, request.container):
        if not isinstance(value, str) or not HEX64.fullmatch(value):
            raise Refused("falta identidad o huella aprobada externa")
    if not isinstance(request.source_commit, str) or not HEX40.fullmatch(request.source_commit):
        raise Refused("fuente sin commit canónico")
    lock_data = clon_sql.approved_file(request.release_lock, request.approved_lock_sha256,
                                      64_000)
    lock = {}
    for line in lock_data.decode("ascii").splitlines():
        pair = line.split()
        if len(pair) != 2 or pair[0] in lock:
            raise Refused("lock inválido o duplicado")
        lock[pair[0]] = pair[1]
    for key, expected in (("PAQUETE_SHA256", request.approved_package_sha256),
                          ("COMMIT", request.source_commit),
                          ("SQL_RELEASE_SHA256", request.approved_release_sha256)):
        if lock.get(key) != expected:
            raise Refused("lock distinto de aprobación externa")
    for key in ("AD132_RECIBO_SHA256", "AD132_CLI_SHA256"):
        if not HEX64.fullmatch(lock.get(key, "")):
            raise Refused("helper o CLI sin pin aprobado")
    package = clon_sql.approved_file(request.package_tar, request.approved_package_sha256,
                                    256 * 1024 * 1024)
    files, directories = package_tree(package)
    root = verify_tree(request.package_root, files, directories)
    helper_path = root / HELPER_REL
    helper_bytes = clon_sql.approved_file(helper_path, lock["AD132_RECIBO_SHA256"], 4_000_000)
    clon_sql.approved_file(root / CLI_REL, lock["AD132_CLI_SHA256"], 4_000_000)
    sys.dont_write_bytecode = True
    helper = types.ModuleType("ad132_recibo_original")
    helper.__file__ = str(helper_path)
    # Se ejecutan los bytes ya verificados, sin reabrir el helper ni usar pyc.
    exec(compile(helper_bytes, str(helper_path), "exec"), helper.__dict__)
    args = argparse.Namespace(engine="docker", container=request.container,
                              plan=clon_sql.validate_original_path(request.plan),
                              plan_receipt=clon_sql.validate_original_path(request.plan_receipt),
                              release_manifest=root / "h6-sql-release.json",
                              release_lock=clon_sql.validate_original_path(request.release_lock),
                              approval=clon_sql.validate_original_path(request.approval),
                              approval_sha256=request.approval_sha256,
                              receipt=private_destination(request.receipt),
                              apply_output=private_destination(request.apply_output),
                              pending_path=private_destination(request.pending_path))
    destinations = (args.receipt, args.apply_output, args.pending_path)
    if len(set(destinations)) != len(destinations):
        raise Refused("pending, salida y recibo requieren destinos distintos")
    if any(path.is_relative_to(root) for path in destinations):
        raise Refused("evidencia privada fuera del árbol inmutable del paquete")
    mod, release, approval, common = helper.context(args)
    expected = {"package_sha256": request.approved_package_sha256,
                "source_commit": request.source_commit,
                "release_sha256": request.approved_release_sha256,
                "lock_sha256": request.approved_lock_sha256,
                "plan_sha256": request.approved_plan_sha256,
                "approval_sha256": request.approval_sha256,
                "pg_container_id": request.container,
                "cli_sha256": lock["AD132_CLI_SHA256"],
                "apply_stdout_sha256": digest(SUCCESS)}
    if helper.SUCCESS != SUCCESS or any(common.get(k) != v for k, v in expected.items()):
        raise Refused("contexto D distinto de aprobaciones previas")
    return helper, mod, release, approval, common, args


def cli_command(args):
    return ["/usr/bin/python3", "-I", "-B", "-S", str(args.release_manifest.parent / CLI_REL),
            "apply", "--engine", "docker", "--container", args.container,
            "--plan", str(args.plan), "--plan-receipt", str(args.plan_receipt),
            "--release-manifest", str(args.release_manifest),
            "--release-lock", str(args.release_lock), "--approval", str(args.approval)]


def pending_record(common, args):
    return {"version": 1, "kind": "ad132_apply_pending", "context": common,
            "cli_command_sha256": digest((json.dumps(
                cli_command(args), ensure_ascii=False, separators=(",", ":")) + "\n").encode())}


def require_pending(helper, common, args):
    expected = helper.canonical(pending_record(common, args))
    if helper.read_regular(args.pending_path, 64_000, private=True) != expected:
        raise Refused("pending distinto de contexto, aprobación o CLI")


def apply_and_confirm(request):
    """Sólo un exit0 observado aquí puede crear el recibo canónico original D."""
    helper, mod, release, approval, common, args = load_adapter(request)
    if args.receipt.exists() or args.apply_output.exists() or args.pending_path.exists():
        raise Refused("evidencia previa presente; revalidar sin reaplicar")
    helper.atomic_receipt(args.pending_path, helper.canonical(pending_record(common, args)))
    # La marca se sincroniza antes de la CLI y nunca se elimina al recuperar.
    helper, mod, release, approval, current_common, args = load_adapter(request)
    if current_common != common:
        raise Refused("contexto cambió después de pending")
    require_pending(helper, common, args)
    if args.receipt.exists() or args.apply_output.exists():
        raise Refused("evidencia apareció después de pending; no reaplicar")
    result = subprocess.run(cli_command(args), stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                    check=False, timeout=360, env={"PATH": "/usr/bin:/bin",
                    "HOME": "/nonexistent", "LC_ALL": "C", "PYTHONDONTWRITEBYTECODE": "1"})
    if result.returncode != 0 or result.stdout != SUCCESS:
        raise Refused("apply no terminó con exit0 y stdout exacto; conservar pending")
    helper, mod, release, approval, current_common, args = load_adapter(request)
    if current_common != common:
        raise Refused("contexto cambió durante apply; conservar pending")
    require_pending(helper, common, args)
    current = mod.current_inventory(args, release)
    anchors = mod.current_anchors(args, release)
    if (not mod.postimage_compatible(approval["inventory"], current)
            or anchors != release["anchors_expected"]):
        raise Refused("postimagen o anclas divergentes; conservar pending")
    receipt = {**common, "postimage": helper.stable_inventory(current),
               "installed_anchors": anchors}
    data = helper.canonical(receipt)
    if len(data) > 4_000_000:
        raise Refused("recibo excede límite privado")
    helper.atomic_receipt(args.apply_output, result.stdout)
    helper.atomic_receipt(args.receipt, data)
    return receipt


def revalidar(request):
    helper, mod, release, approval, common, args = load_adapter(request)
    require_pending(helper, common, args)
    data = helper.read_regular(args.receipt, 4_000_000, private=True)
    receipt = helper.json_object(data)
    if (set(receipt) != helper.FIELDS or data != helper.canonical(receipt)
            or any(receipt.get(k) != v for k, v in common.items())
            or receipt.get("installed_anchors") != release["anchors_expected"]
            or not mod.postimage_compatible(approval["inventory"], receipt.get("postimage", {}))
            or helper.read_regular(args.apply_output, 256, private=True) != SUCCESS):
        raise Refused("recibo ajeno o sin confirmación original")
    current = mod.current_inventory(args, release)
    anchors = mod.current_anchors(args, release)
    if (helper.stable_inventory(current) != receipt["postimage"]
            or anchors != receipt["installed_anchors"]):
        raise Refused("postimagen o anclas cambiaron desde apply")
    return receipt


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("mode", choices=("revalidar",))
    for name in Request.__dataclass_fields__:
        is_path = name in {"package_tar", "package_root", "release_lock", "plan",
                          "plan_receipt", "approval", "receipt", "apply_output", "pending_path"}
        parser.add_argument("--" + name.replace("_", "-"), required=True,
                            type=Path if is_path else str)
    values = vars(parser.parse_args(argv))
    values.pop("mode")
    revalidar(Request(**values))
    print("AD3-132-REVALIDADO-OK")


if __name__ == "__main__":
    try:
        main()
    except Exception as error:
        # Ni mensajes del proveedor ni stdout/stderr privados pasan al terminal.
        print(f"AD132-NO-GO {type(error).__name__}", file=sys.stderr)
        raise SystemExit(1)
