#!/usr/bin/env python3
"""Copia TLS de desarrollo a ruta de sistema root-owned para vec-interno.

Se ejecuta como root en una instancia desechable o por el operador en cidonia.
Nunca modifica el material de origen ni arranca el servicio.
"""

from __future__ import annotations

import argparse
import grp
import os
from pathlib import Path
import stat
import subprocess
import sys
import tempfile


def fail(message: str) -> None:
    raise RuntimeError(message)


def root_chain(path: Path) -> None:
    if not path.is_absolute() or path == Path("/") or path != Path(os.path.normpath(path)):
        fail("destino TLS no absoluto/limpio")
    current = Path("/")
    for name in path.parts[1:]:
        current /= name
        if current.is_symlink():
            fail("enlace simbólico en ruta TLS")
        if not current.exists():
            current.mkdir(mode=0o755)
        info = current.stat()
        if not current.is_dir() or info.st_uid != 0 or stat.S_IMODE(info.st_mode) & 0o022:
            fail("directorio TLS no root-owned o escribible por terceros")


def source_file(path: Path) -> bytes:
    if path.is_symlink() or not path.is_file() or stat.S_IMODE(path.stat().st_mode) != 0o600:
        fail("fuente TLS privada ausente o insegura")
    data = path.read_bytes()
    if not data or len(data) > 1 << 20:
        fail("fuente TLS vacía o excesiva")
    return data


def install_file(target: Path, data: bytes, mode: int, gid: int) -> None:
    if target.is_symlink():
        fail("destino TLS enlazado")
    if target.exists():
        info = target.stat()
        if not target.is_file() or info.st_uid != 0 or info.st_gid != gid or \
           stat.S_IMODE(info.st_mode) != mode or target.read_bytes() != data:
            fail("TLS root-owned existente diferente; rotación requiere acto separado")
        return
    fd, temp = tempfile.mkstemp(prefix=".vec-tls-", dir=target.parent)
    try:
        with os.fdopen(fd, "wb") as file:
            file.write(data)
            file.flush()
            os.fsync(file.fileno())
        os.chown(temp, 0, gid)
        os.chmod(temp, mode)
        os.link(temp, target)
    finally:
        os.unlink(temp)


def main() -> None:
    parser = argparse.ArgumentParser(description="Instalación TLS root-owned separada de claves de identidad")
    parser.add_argument("--material-dir", required=True)
    parser.add_argument("--dest-root", required=True)
    parser.add_argument("--service-group", required=True)
    args = parser.parse_args()
    if os.geteuid() != 0:
        fail("instalar-tls requiere operador root; la app corre sin root")
    source = Path(args.material_dir).absolute()
    if source.is_symlink() or not source.is_dir() or stat.S_IMODE(source.stat().st_mode) != 0o700:
        fail("material TLS origen inseguro")
    git = subprocess.run(["git", "-C", str(source), "rev-parse", "--show-toplevel"],
                         stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, check=False)
    if git.returncode == 0 and git.stdout.strip():
        fail("fuente TLS dentro de Git")
    target = Path(args.dest_root)
    if str(target).startswith(("/tmp/", "/home/", "/workspace/")):
        fail("TLS productivo no puede residir en directorio de usuario o temporal")
    gid = grp.getgrnam(args.service_group).gr_gid
    cert = source_file(source / "tls" / "servidor.crt")
    key = source_file(source / "tls" / "servidor.key")
    ca = source_file(source / "ca" / "ac.crt")
    if cert.count(b"-----BEGIN CERTIFICATE-----") < 2 or ca.count(b"-----BEGIN CERTIFICATE-----") != 1:
        fail("certificado servidor sin fullchain o CA incompatible")
    with tempfile.TemporaryDirectory(prefix="vec-tls-validar-") as temporary:
        cert_file = Path(temporary) / "cert.pem"
        ca_file = Path(temporary) / "ca.pem"
        cert_file.write_bytes(cert)
        ca_file.write_bytes(ca)
        verified = subprocess.run(["openssl", "verify", "-purpose", "sslserver",
                                   "-CAfile", str(ca_file), str(cert_file)],
                                  stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, check=False)
        if verified.returncode:
            fail("cadena servidor no verifica con CA")
    root_chain(target)
    install_file(target / "servidor.crt", cert, 0o644, 0)
    install_file(target / "servidor.key", key, 0o440, gid)
    install_file(target / "clientes-ca.crt", ca, 0o644, 0)
    print("TLS root-owned instalado; servicio no-root puede leer solo su grupo")


if __name__ == "__main__":
    try:
        main()
    except (OSError, KeyError, RuntimeError) as error:
        print(f"vec-interno TLS: {error}", file=sys.stderr)
        sys.exit(1)
