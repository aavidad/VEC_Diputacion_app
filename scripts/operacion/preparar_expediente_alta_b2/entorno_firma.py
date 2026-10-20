#!/usr/bin/env python3
"""Prepara cuatro entornos locales separados para firmar desde el navegador."""
import argparse
import hashlib
import json
import os
import re
import secrets
import select
import shutil
import stat
import subprocess
import sys
import tempfile
from pathlib import Path

from preparar import Corte, exigir_origen, guardar
from tramitar import ruta_privada


ROLES = ("tecnico", "jefatura", "diputacion", "secretaria")


def privado(path, root):
    if (not root in path.parents or any(p.is_symlink() for p in (path, *path.parents) if p != root)
            or path.is_symlink()):
        raise Corte("FALTA: fuente de firmante enlazada o ajena")
    try:
        info = path.stat()
    except OSError as exc:
        raise Corte("FALTA: fuente de firmante ausente") from exc
    if (not stat.S_ISREG(info.st_mode) or info.st_uid != os.getuid()
            or stat.S_IMODE(info.st_mode) != 0o600 or not 0 < info.st_size <= 1048576):
        raise Corte("FALTA: P12 o contraseña exige fichero propio 0600")
    return path


def fuentes(root):
    paths = {"tecnico": (root / "material/mtls/cliente.p12",
                         root / "material/mtls/cliente.p12.password")}
    for role in ROLES[1:]:
        folder = root / "recorrido/firmantes/certificados" / ("rrhh_" + role)
        paths[role] = (folder / "cliente.p12", folder / "pkcs12.password")
    for p12, password in paths.values():
        privado(p12, root)
        privado(password, root)
    if len({hashlib.sha256(p12.read_bytes()).digest() for p12, _ in paths.values()}) != 4:
        raise Corte("FALTA: cada firmante necesita un P12 distinto")
    return paths


def escribir(path, content, mode=0o600):
    raw = content.encode() if isinstance(content, str) else content
    fd = os.open(path, os.O_CREAT | os.O_EXCL | os.O_WRONLY | os.O_NOFOLLOW, mode)
    with os.fdopen(fd, "wb") as stream:
        stream.write(raw)
        stream.flush()
        os.fsync(stream.fileno())


def desktop_quote(path):
    text = str(path)
    if "\n" in text or "\r" in text:
        raise Corte("FALTA: ruta incompatible con entrada de escritorio")
    return '"' + ''.join('%%' if c == '%' else '\\' + c if c in '\\"$`' else c for c in text) + '"'


LAUNCHER = '''#!/usr/bin/env python3
import os, pathlib, sys
base = pathlib.Path(__file__).resolve().parent
role = sys.argv[1] if len(sys.argv) > 1 else ''
if role not in ('tecnico', 'jefatura', 'diputacion', 'secretaria'): sys.exit(2)
home = base / role
password_file = home / 'p12.password'
if password_file.is_symlink() or password_file.stat().st_mode & 0o777 != 0o600: sys.exit(2)
password = password_file.read_text().strip()
display = (base / 'display').read_text().strip()
env = {'PATH':'/usr/bin:/bin', 'DISPLAY':':' + display,
       'XAUTHORITY':str(base / 'Xauthority'),
       'HOME':str(home / 'home'), 'XDG_CONFIG_HOME':str(home / 'home/.config'),
       'XDG_DATA_HOME':str(home / 'xdg-data'), 'XDG_CACHE_HOME':str(home / 'cache'),
       'XDG_RUNTIME_DIR':str(home / 'xdg-runtime'), 'TMPDIR':str(home / 'tmp'),
       'GRXFIRMA_PKCS12_DIR':str(home / 'p12'), 'GRXFIRMA_DIRECTORIO_P12':str(home / 'p12'),
       'GRXFIRMA_PKCS12_PASSWORD':password,
       'GRXFIRMA_LEGACY_LAUNCH_SOCKET':str(home / 'xdg-runtime/afirmauri-control.sock'),
       'GRXFIRMA_LOG_FILE':str(home / 'grxfirma.log'), 'LIBGL_ALWAYS_SOFTWARE':'1'}
fd = os.open(home / 'grxfirma.out', os.O_CREAT | os.O_APPEND | os.O_WRONLY, 0o600)
os.dup2(fd, 1); os.dup2(fd, 2); os.close(fd)
exe = str(base / 'grxfirma-afirmauri')
os.execve(exe, [exe, *sys.argv[2:]], env)
'''


def autorizar_pantalla(auth, display, cookie):
    command = f"add :{display} MIT-MAGIC-COOKIE-1 {cookie}\n".encode()
    try:
        result = subprocess.run(["/usr/bin/xauth", "-q", "-f", str(auth), "-"],
                                input=command, capture_output=True, timeout=5,
                                env={"PATH": "/usr/bin:/bin", "HOME": str(auth.parent)})
    except (OSError, subprocess.TimeoutExpired) as exc:
        raise Corte("FALTA: autenticación privada de pantalla no disponible") from exc
    if result.returncode:
        raise Corte("FALTA: autenticación privada de pantalla rechazada")


def cadena_confiable(path):
    for part in (path, *path.parents):
        if not os.path.lexists(part):
            continue
        info = part.lstat()
        mode = stat.S_IMODE(info.st_mode)
        if (not stat.S_ISDIR(info.st_mode) or info.st_uid not in (0, os.getuid())
                or (mode & 0o022 and not mode & stat.S_ISVTX)):
            raise Corte("FALTA: ancestro ajeno o escribible sin protección")


def preparar_salida(root, fw):
    if fw is None or not fw.is_absolute():
        raise Corte("FALTA: --salida privada absoluta y separada de las fuentes")
    ruta_privada(fw, "--salida")
    fw = fw.resolve(strict=False)
    if root == fw or root in fw.parents:
        raise Corte("FALTA: --salida dentro de las fuentes")
    if os.path.lexists(fw):
        raise Corte("FALTA: salida ya existe; conservar su entorno")
    cadena_confiable(fw.parent)
    parent = fw.parent
    missing = []
    while not parent.exists():
        missing.append(parent)
        parent = parent.parent
    return fw, missing


def preparar(root, origin, binary, sha, salida=None):
    ruta_privada(root, "--raiz-instancia")
    root = root.resolve(strict=False)
    cadena_confiable(root)
    fw, missing = preparar_salida(root, salida)
    if not re.fullmatch(r"[0-9a-f]{64}", sha):
        raise Corte("FALTA: huella SHA256 del manejador inválida")
    if not binary.is_absolute() or binary.is_symlink() or not binary.is_file():
        raise Corte("FALTA: binario afirmauri absoluto y regular")
    info = binary.stat()
    if (not stat.S_ISREG(info.st_mode) or info.st_uid != os.getuid()
            or not stat.S_IMODE(info.st_mode) & 0o100
            or stat.S_IMODE(info.st_mode) & 0o022
            or not 0 < info.st_size <= 104857600):
        raise Corte("FALTA: binario afirmauri propio, ejecutable y sin escritura ajena requerido")
    if hashlib.sha256(binary.read_bytes()).hexdigest() != sha:
        raise Corte("FALTA: huella del binario afirmauri distinta")
    root_info = root.stat()
    if (not stat.S_ISDIR(root_info.st_mode) or root_info.st_uid != os.getuid()
            or stat.S_IMODE(root_info.st_mode) & 0o022):
        raise Corte("FALTA: raíz de instancia propia sin escritura ajena requerida")
    with tempfile.TemporaryDirectory(prefix="firma-version-", dir=root) as probe_home:
        verified = Path(probe_home) / "grxfirma-afirmauri"
        shutil.copyfile(binary, verified)
        if hashlib.sha256(verified.read_bytes()).hexdigest() != sha:
            raise Corte("FALTA: copia de comprobación con huella distinta")
        verified.chmod(0o700)
        try:
            probe = subprocess.run([str(verified), "--version"], capture_output=True,
                                   env={"HOME": probe_home, "PATH": "/usr/bin:/bin"}, timeout=5)
        except (OSError, subprocess.TimeoutExpired) as exc:
            raise Corte("FALTA: tipo de binario afirmauri no comprobable") from exc
        if probe.returncode or not re.search(rb"grxfirma[- ]afirmauri", probe.stdout + probe.stderr, re.I):
            raise Corte("FALTA: el binario no es grxfirma-afirmauri")
        verified_bytes = verified.read_bytes()
    paths = fuentes(root)
    for directory in reversed(missing):
        directory.mkdir(mode=0o700)
    cadena_confiable(fw.parent)
    created = False
    process = None
    try:
        fw.mkdir(mode=0o700)
        created = True
        escribir(fw / "grxfirma-afirmauri", verified_bytes, 0o700)
        if hashlib.sha256((fw / "grxfirma-afirmauri").read_bytes()).hexdigest() != sha:
            raise Corte("FALTA: binario copiado con huella distinta")
        escribir(fw / "lanzar-grxfirma.py", LAUNCHER, 0o700)
        for role, (p12, password) in paths.items():
            base = fw / role
            for directory in ("home/.config/grxfirma", "xdg-config", "xdg-data/applications",
                              "cache", "xdg-runtime", "tmp", "p12", "chrome/Default"):
                (base / directory).mkdir(mode=0o700, parents=True, exist_ok=True)
            shutil.copyfile(p12, base / "p12/firmante.p12")
            (base / "p12/firmante.p12").chmod(0o600)
            shutil.copyfile(password, base / "p12.password")
            (base / "p12.password").chmod(0o600)
            cfg = base / "home/.config/grxfirma"
            guardar(cfg / "config.json", {"websocket_habilitado": True, "rest_habilitado": False,
                "tofu_habilitado": True, "directorio_p12": str(base / "p12"),
                "dominios_de_confianza": [origin], "nivel_log": "info",
                "timeout_operacion_segundos": 120, "max_tamano_documento_bytes": 52428800})
            guardar(cfg / "settings.json", {"autoSelectSingleCertificate": True,
                "confirmToSign": True, "signVisibleSeal": False, "checkForUpdates": False})
            guardar(cfg / "trusted-domains.json", {origin: "allowed"})
            for marker, value in (("trusted-domains.seeded", "seeded-default-trusted-domains"),
                                  ("trusted-domains.compatibility-v1.seeded", "seeded"),
                                  ("trusted-domains.hardening-2026-09.migrated", "migrated")):
                escribir(cfg / marker, value)
            escribir(base / "xdg-data/applications/grxfirma-copia.desktop",
                "[Desktop Entry]\nType=Application\nName=GrxFirma copia\nExec="
                + desktop_quote(fw / "lanzar-grxfirma.py") + " " + role
                + " %u\nMimeType=x-scheme-handler/afirma;\nNoDisplay=true\nTerminal=false\n")
            for directory in ("xdg-data", "xdg-config"):
                escribir(base / directory / "mimeapps.list",
                        "[Default Applications]\nx-scheme-handler/afirma=grxfirma-copia.desktop;\n")
            guardar(base / "chrome/Default/Preferences",
                    {"protocol_handler": {"allowed_origin_protocol_pairs": {origin: {"afirma": True}}}})
        auth = fw / "Xauthority"
        escribir(auth, b"")
        cookie = secrets.token_hex(16)
        autorizar_pantalla(auth, "0", cookie)
        r, w = os.pipe()
        try:
            log_fd = os.open(fw / "xvfb.log", os.O_CREAT | os.O_WRONLY | os.O_APPEND, 0o600)
            try:
                process = subprocess.Popen(["Xvfb", "-displayfd", str(w), "-screen", "0",
                    "1440x900x24", "-nolisten", "tcp", "-auth", str(auth)], pass_fds=(w,), stdout=log_fd,
                    stderr=log_fd, start_new_session=True,
                    env={"PATH": "/usr/bin:/bin", "HOME": str(fw)})
            finally:
                os.close(log_fd)
                os.close(w)
            ready, _, _ = select.select([r], [], [], 10)
            display = os.read(r, 16).decode().strip() if ready else ""
            if not re.fullmatch(r"[0-9]{1,3}", display) or process.poll() is not None:
                raise Corte("FALTA: Xvfb propio no entregó pantalla")
        finally:
            os.close(r)
        autorizar_pantalla(auth, display, cookie)
        escribir(fw / "display", display + "\n")
        escribir(fw / "xvfb.pid", str(process.pid) + "\n")
        print("Entornos de firma preparados; abra las sesiones de cada firmante en su navegador")
    except BaseException:
        if process is not None:
            process.terminate()
            try:
                process.wait(timeout=2)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait(timeout=2)
        if created:
            shutil.rmtree(fw)
        raise


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__,
        epilog="--salida guarda firma-web en un directorio privado separado de las fuentes. Para Chrome use --salida/display y XAUTHORITY=--salida/Xauthority. La firma se realiza en el navegador.")
    parser.add_argument("--raiz-instancia", type=Path, required=True)
    parser.add_argument("--salida", type=Path, required=True, help="directorio privado absoluto nuevo para firma-web")
    parser.add_argument("--origen", required=True, help="origen HTTPS loopback de la copia")
    parser.add_argument("--binario-afirmauri", type=Path, required=True)
    parser.add_argument("--sha-afirmauri", required=True)
    args = parser.parse_args(argv)
    os.umask(0o077)
    root = ruta_privada(args.raiz_instancia, "--raiz-instancia")
    origin = exigir_origen(args.origen)
    preparar(root, origin, args.binario_afirmauri, args.sha_afirmauri, args.salida)
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except Corte as exc:
        print(str(exc), file=sys.stderr)
        sys.exit(2)
