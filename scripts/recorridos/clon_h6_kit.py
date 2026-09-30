"""Proveedor H6 para una restauración H1 fresca, sin escrituras PostgreSQL.

Request procede del orquestador revisado: las huellas se reciben desde fuera.
El recibo de restauración lo crea exclusivamente ese orquestador después de
observar RESTORE, con creación exclusiva y fsync; este módulo nunca lo emite.
Una huella de dump no es una huella SQL: se cotejan los dumps normalizados
mediante los métodos fijos de ReadOnlyDB. No se admite reanudar ninguna UP.
Los 62 exit0 del journal acreditan ejecución observada, no 62 postimágenes.
verify exige además el recibo AD132 original revalidado contra el clon vivo.
No modifica LIVE_KIT ni publica READY, que pertenece al orquestador.
"""

from dataclasses import dataclass
import json
import os
from pathlib import Path
import re
import stat

try:
    from . import clon_sql, clon_ad132_recibo
except ImportError:
    import clon_sql
    import clon_ad132_recibo

Refused = clon_sql.Refused
HEX64 = re.compile(r"[0-9a-f]{64}\Z")
IDENTITY_FIELDS = {"system_identifier", "database_name", "database_oid",
                   "pg_container_id", "pg_image", "pg_image_id", "pg_volume"}
RESTORE_FIELDS = IDENTITY_FIELDS | {"version", "kind", "estado_h1_sha",
                                    "schema_sha", "roles_sha", "datacl_sha"}


@dataclass(frozen=True)
class Request:
    package_tar: Path
    release_lock: Path
    approved_package_sha256: str
    approved_lock_sha256: str
    h1_state_file: Path
    approved_h1_sha256: str
    git_repo: Path
    restore_receipt: Path
    approved_restore_receipt_sha256: str
    normalizer_path: Path
    approved_normalizer_sha256: str
    guiones_manifest: Path
    approved_guiones_sha256: str
    ad132_request: clon_ad132_recibo.Request | None = None


def canonical(value):
    return (json.dumps(value, sort_keys=True, ensure_ascii=False,
                       separators=(",", ":")) + "\n").encode("utf-8")


def unique(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise Refused("JSON con clave duplicada")
        result[key] = value
    return result


def read_restore(request):
    """Lee bytes y metadatos del mismo FD bajo directorios confiables retenidos."""
    path = Path(request.restore_receipt).absolute()
    if (not path.name or ".." in path.parts
            or not isinstance(request.approved_restore_receipt_sha256, str)
            or not HEX64.fullmatch(request.approved_restore_receipt_sha256)):
        raise Refused("ruta o huella del recibo de restauración incompatible")
    directories, snapshots = [], []
    fd = None
    try:
        directories.append(os.open("/", os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW))
        for position in range(len(path.parts) - 1):
            current = directories[-1]
            status = os.fstat(current)
            if (not stat.S_ISDIR(status.st_mode) or status.st_uid not in {0, os.getuid()}
                    or status.st_mode & 0o022):
                raise Refused("ancestro del recibo no confiable")
            try:
                os.stat(".git", dir_fd=current, follow_symlinks=False)
            except FileNotFoundError:
                pass
            else:
                raise Refused("recibo de restauración debe permanecer fuera de Git")
            snapshots.append(status)
            if position < len(path.parts) - 2:
                directories.append(os.open(path.parts[position + 1],
                    os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW, dir_fd=current))
        parent = snapshots[-1]
        if parent.st_uid != os.getuid() or stat.S_IMODE(parent.st_mode) != 0o700:
            raise Refused("directorio del recibo requiere propietario actual y modo 0700")
        fd = os.open(path.name, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK,
                     dir_fd=directories[-1])
        before = os.fstat(fd)
        if (not stat.S_ISREG(before.st_mode) or before.st_nlink != 1
                or before.st_uid != os.getuid() or stat.S_IMODE(before.st_mode) != 0o600
                or before.st_size > 16384):
            raise Refused("recibo requiere fichero propio regular de modo 0600")
        chunks, size = [], 0
        while chunk := os.read(fd, 16385):
            chunks.append(chunk)
            size += len(chunk)
            if size > 16384:
                raise Refused("recibo de restauración excede límites")
        data = b"".join(chunks)
        after = os.fstat(fd)
        stable = lambda s: (s.st_dev, s.st_ino, s.st_uid, s.st_mode, s.st_nlink,
                            s.st_size, s.st_mtime_ns, s.st_ctime_ns)
        # También rechaza una sustitución del nombre mientras estaba abierto.
        named = os.stat(path.name, dir_fd=directories[-1], follow_symlinks=False)
        if (stable(before) != stable(after) or stable(after) != stable(named)
                or size != before.st_size
                or clon_sql.sha(data) != request.approved_restore_receipt_sha256):
            raise Refused("recibo cambió durante lectura o difiere de la huella aprobada")
        for directory, initial in zip(directories, snapshots, strict=True):
            now = os.fstat(directory)
            if ((initial.st_dev, initial.st_ino, initial.st_uid, initial.st_mode)
                    != (now.st_dev, now.st_ino, now.st_uid, now.st_mode)):
                raise Refused("ancestro cambió durante lectura del recibo")
        return data
    except OSError as error:
        raise Refused("ruta del recibo de restauración inválida") from error
    finally:
        if fd is not None:
            os.close(fd)
        for directory in reversed(directories):
            os.close(directory)


def restore_receipt(request):
    """Lee sólo el sello propio fijado por la restauración del orquestador."""
    data = read_restore(request)
    try:
        value = json.loads(data, object_pairs_hook=unique)
        if not isinstance(value, dict) or set(value) != RESTORE_FIELDS or data != canonical(value):
            raise ValueError()
        if (type(value["version"]) is not int or value["version"] != 1
                or value["kind"] != "h1_restore_confirmed"
                or value["estado_h1_sha"] != request.approved_h1_sha256
                or value["database_name"] != "postgres"
                or type(value["database_oid"]) is not int or value["database_oid"] <= 0
                or not isinstance(value["system_identifier"], str)
                or not re.fullmatch(r"[1-9][0-9]{0,19}", value["system_identifier"])
                or not isinstance(value["pg_image_id"], str)
                or not re.fullmatch(r"sha256:[0-9a-f]{64}", value["pg_image_id"])
                or value["pg_image"] != "postgres:18.4"):
            raise ValueError()
        for key in ("estado_h1_sha", "pg_container_id", "schema_sha", "roles_sha", "datacl_sha"):
            if not isinstance(value[key], str) or not HEX64.fullmatch(value[key]) or value[key] == "0" * 64:
                raise ValueError()
        volume = Path(value["pg_volume"])
        if (volume.parent != Path("/dev/shm")
                or not re.fullmatch(r"vec-recorridos-[A-Za-z0-9_-]+", volume.name)):
            raise ValueError()
    except (ValueError, TypeError, KeyError, UnicodeError) as error:
        raise Refused("recibo de RESTORE fresco incompatible") from error
    return value


class H6Kit:
    def __init__(self, request):
        if not isinstance(request, Request):
            raise Refused("el proveedor requiere Request del orquestador")
        self.request = request
        self._plan = None

    def validate(self, plan, context):
        """Preflight de artefactos y sello externo, antes de inspeccionar el clon."""
        self._plan = None
        req = self.request
        context = clon_sql.validate_context(context)
        expected, _ = clon_sql.preflight_h6_package(
            req.package_tar, req.release_lock, req.approved_package_sha256,
            req.approved_lock_sha256, req.h1_state_file, req.approved_h1_sha256,
            source_ref=clon_sql.H6_FIRMA_FINAL_REF, git_repo=req.git_repo)
        if plan != expected or any(context.get(key) != expected.get(key)
                for key in (*clon_sql.CONTEXT_KEYS[1:], "lock_sha")):
            raise Refused("plan/contexto distinto de H1, paquete y lock aprobados")
        receipt = restore_receipt(req)
        if context["identidad_clon"] != req.approved_restore_receipt_sha256:
            raise Refused("identidad del clon distinta del sello RESTORE aprobado")
        # El normalizador D no tiene clave propia en release.lock: pin externo.
        clon_sql.approved_file(req.normalizer_path, req.approved_normalizer_sha256, 64000)
        scripts = clon_sql.approved_file(req.guiones_manifest, req.approved_guiones_sha256, 64000)
        lock = clon_sql.approved_file(req.release_lock, req.approved_lock_sha256, 1024 * 1024)
        lock_values = dict(line.split() for line in lock.decode("ascii").splitlines())
        if lock_values.get("KIT_GUIONES_SHA256") != req.approved_guiones_sha256:
            raise Refused("guiones distintos del lock aprobado")
        listed = {}
        for line in scripts.decode("ascii").splitlines():
            pair = line.split()
            if len(pair) != 2 or pair[1] in listed or not HEX64.fullmatch(pair[0]):
                raise Refused("manifiesto de guiones incompatible")
            listed[pair[1]] = pair[0]
        if (req.normalizer_path.name != "h6_normalizar_pg_dump.py"
                or listed.get(req.normalizer_path.name) != req.approved_normalizer_sha256):
            raise Refused("normalizador distinto de los guiones aprobados")
        if req.ad132_request is not None:
            self._bind_ad132(req.ad132_request, expected, receipt)
        self._plan = expected
        return True

    def _ready(self, db, context):
        if self._plan is None or not isinstance(db, clon_sql.ReadOnlyDB):
            raise Refused("falta preflight o frontera ReadOnlyDB")
        context = clon_sql.validate_context(context)
        if (context["identidad_clon"] != self.request.approved_restore_receipt_sha256
                or any(context.get(key) != self._plan.get(key)
                       for key in (*clon_sql.CONTEXT_KEYS[1:], "lock_sha"))):
            raise Refused("contexto cambiado después del preflight")
        return restore_receipt(self.request)

    def identity(self, db, context):
        receipt = self._ready(db, context)
        observed = db.system_identity()
        expected = {key: receipt[key] for key in IDENTITY_FIELDS}
        if observed != expected:
            raise Refused("identidad viva distinta de la restauración H1")
        return self.request.approved_restore_receipt_sha256

    def confirm(self, db, last_entry, completed, context):
        """Sólo preimagen inicial: cualquier interrupción exige clon nuevo."""
        if type(completed) is not int or completed != 0 or last_entry is not None:
            raise Refused("familia62 no admite reanudación SQL; reconstruir un clon nuevo")
        receipt = self._ready(db, context)
        self.identity(db, context)
        req = self.request
        if (db.schema_digest(req.normalizer_path, req.approved_normalizer_sha256) != receipt["schema_sha"]
                or db.roles_digest(req.normalizer_path, req.approved_normalizer_sha256) != receipt["roles_sha"]
                or db.database_acl_digest() != receipt["datacl_sha"]):
            raise Refused("preimagen viva de esquema, roles o ACL distinta de RESTORE H1")
        return True

    @staticmethod
    def _bind_ad132(request, plan, receipt):
        if not isinstance(request, clon_ad132_recibo.Request):
            raise Refused("AD132 exige Request original, sin callbacks")
        if (request.approved_package_sha256 != plan["package_sha"]
                or request.approved_lock_sha256 != plan["lock_sha"]
                or request.approved_release_sha256 != plan["release_sha"]
                or request.source_commit != plan["source_ref"]
                or request.container != receipt["pg_container_id"]):
            raise Refused("AD132 de otra restauración, paquete, lock o release")

    def verify(self, db, record, context):
        """Journal62 completo más postimagen/anclas originales AD132 vivas."""
        receipt = self._ready(db, context)
        self.identity(db, context)
        clon_sql.validate_record(record, self._plan, context)
        clon_sql.validate_receipts(record.get("installed"), self._plan)
        if record.get("phase") != "ad132_confirmed":
            raise Refused("falta cierre AD132 independiente")
        request = self.request.ad132_request
        if request is None:
            raise Refused("falta Request de revalidación AD132 original")
        self._bind_ad132(request, self._plan, receipt)
        try:
            confirmed = clon_ad132_recibo.revalidar(request)
        except Exception as error:
            raise Refused("postimagen o recibo AD132 no revalidado") from error
        if (confirmed.get("kind") != "ad132_apply_confirmed"
                or confirmed.get("package_sha256") != self._plan["package_sha"]
                or confirmed.get("lock_sha256") != self._plan["lock_sha"]
                or confirmed.get("release_sha256") != self._plan["release_sha"]
                or confirmed.get("source_commit") != self._plan["source_ref"]
                or confirmed.get("pg_container_id") != receipt["pg_container_id"]):
            raise Refused("recibo AD132 revalidado pertenece a otro contexto")
        return True
