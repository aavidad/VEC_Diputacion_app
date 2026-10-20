"""Base de revocación de una CA nueva para el recorrido F1-131."""

from datetime import datetime, timedelta, timezone
import hashlib
import os
from pathlib import Path
import re
import stat
import tempfile


OPENSSL = "/usr/bin/openssl"
SID = re.compile(r"[1-9][0-9]{14,19}\Z")
NUMERO_INICIAL = "1000\n"


def _exigir(condicion, motivo):
    if not condicion:
        raise ValueError(motivo)


def _directorio(ruta):
    ruta = Path(ruta)
    st = ruta.lstat()
    _exigir(stat.S_ISDIR(st.st_mode) and not ruta.is_symlink()
            and st.st_uid == os.getuid() and not stat.S_IMODE(st.st_mode) & 0o077,
            "directorio privado de certificados inválido")
    return ruta


def _archivo(ruta, *, vacio=False):
    ruta = Path(ruta)
    st = ruta.lstat()
    _exigir(stat.S_ISREG(st.st_mode) and not ruta.is_symlink()
            and st.st_uid == os.getuid() and stat.S_IMODE(st.st_mode) == 0o600
            and st.st_size <= 1_048_576 and (vacio or st.st_size > 0),
            "archivo privado de certificados inválido")
    return ruta


def _sha(ruta):
    return hashlib.sha256(_archivo(ruta, vacio=True).read_bytes()).hexdigest()


def _run(c, nombre, *argv, ambos=False):
    anterior = os.umask(0o077)
    try:
        resultado = c.ejecutar((OPENSSL, *map(str, argv)), nombre)
    finally:
        os.umask(anterior)
    _exigir(resultado.returncode == 0, "OpenSSL rechazó material de la copia")
    texto = resultado.stdout
    if ambos:
        texto += resultado.stderr
    return texto.decode("utf-8", "strict")


def _fecha(texto, etiqueta):
    lineas = texto.splitlines()
    _exigir(len(lineas) == 2 and lineas[0].startswith(etiqueta[0] + "=")
            and lineas[1].startswith(etiqueta[1] + "="), "fechas de certificado o CRL inválidas")
    try:
        return tuple(datetime.strptime(linea.split("=", 1)[1].strip(), "%b %d %H:%M:%S %Y %Z")
                     .replace(tzinfo=timezone.utc) for linea in lineas)
    except ValueError as error:
        raise ValueError("fechas de certificado o CRL inválidas") from error


def _ca(c, ca, clave, ahora):
    _archivo(ca)
    _archivo(clave)
    _exigir(_run(c, "cert-base-ca-pub.log", "x509", "-in", ca, "-pubkey", "-noout") ==
            _run(c, "cert-base-key-pub.log", "pkey", "-in", clave, "-pubout"),
            "clave y certificado de CA distintos")
    _run(c, "cert-base-ca-verify.log", "verify", "-CAfile", ca, ca)
    _run(c, "cert-base-ca-checkend.log", "x509", "-in", ca, "-checkend", str(398 * 86400), "-noout")
    desde, hasta = _fecha(_run(c, "cert-base-ca-fechas.log", "x509", "-in", ca,
                            "-noout", "-dates"), ("notBefore", "notAfter"))
    _exigir(desde <= ahora < hasta and hasta - ahora > timedelta(days=398),
            "CA no cubre los certificados F1-131")
    propositos = _run(c, "cert-base-ca-proposito.log", "x509", "-in", ca, "-purpose", "-noout")
    _exigir("SSL client CA : Yes" in propositos, "certificado sin autoridad de CA")


def _indice(ruta):
    datos = _archivo(ruta, vacio=True).read_bytes()
    try:
        lineas = datos.decode("ascii").splitlines()
    except UnicodeDecodeError as error:
        raise ValueError("índice de CA ilegible") from error
    revocados = set()
    for linea in lineas:
        campos = linea.split("\t")
        _exigir(len(campos) >= 4 and campos[0] in ("V", "R", "E")
                and re.fullmatch(r"[0-9A-Fa-f]+", campos[3]), "índice de CA inválido")
        if campos[0] == "R":
            revocados.add(int(campos[3], 16))
    return revocados


def _crl(c, ca, crl, ahora, revocados):
    _archivo(crl)
    salida = _run(c, "cert-base-crl-verify.log", "crl", "-in", crl,
                  "-verify", "-CAfile", ca, "-noout", ambos=True)
    _exigir(any(x.strip() == "verify OK" for x in salida.splitlines()), "firma CRL inválida")
    desde, hasta = _fecha(_run(c, "cert-base-crl-fechas.log", "crl", "-in", crl,
                            "-noout", "-lastupdate", "-nextupdate"),
                          ("lastUpdate", "nextUpdate"))
    _exigir(desde <= ahora + timedelta(minutes=1) < hasta
            and timedelta(days=90) - timedelta(minutes=1) <= hasta - desde <= timedelta(days=90, minutes=1),
            "CRL fuera de 90 días")
    numero = _run(c, "cert-base-crl-numero.log", "crl", "-in", crl,
                  "-noout", "-crlnumber").strip()
    _exigir(re.fullmatch(r"crlNumber=0x[0-9A-Fa-f]+", numero)
            and int(numero.split("0x", 1)[1], 16) >= 0x1000, "número CRL inválido")
    texto = _run(c, "cert-base-crl-texto.log", "crl", "-in", crl, "-noout", "-text")
    presentes = {int(x.replace(":", ""), 16) for x in re.findall(
        r"Serial Number:\s*([0-9A-Fa-f:]+)", texto)}
    _exigir(revocados.issubset(presentes), "CRL no conserva las revocaciones del índice")


def preparar(c):
    """Crear una sola vez index y CRL de 90 días para la CA nueva de la copia."""
    estado = c.copia.leer()
    sid = estado.get("sid_copia")
    _exigir(estado.get("recorrido_completo") is True and estado.get("material_modo") == "nuevo"
            and isinstance(sid, str) and SID.fullmatch(sid)
            and c.copia.raiz == c.raiz and c.copia.material == c.material
            and c.material == c.raiz / "material"
            and c.consultar("SELECT system_identifier::text FROM pg_catalog.pg_control_system();").strip() == sid,
            "certificados F1-131 requieren copia completa nueva y SID propio")
    material = _directorio(c.material)
    ca_dir = _directorio(material / "ca")
    identidad = _directorio(material / "identidad")
    ca, clave = ca_dir / "ca.crt", ca_dir / "ca.key"
    indice, numero = ca_dir / "index.txt", ca_dir / "crlnumber"
    crl = identidad / "clientes.crl"
    ahora = c.ahora().astimezone(timezone.utc).replace(microsecond=0)
    _ca(c, ca, clave, ahora)
    trabajo = c.carpeta / "certificados-base"
    trabajo.mkdir(mode=0o700, exist_ok=True)
    _directorio(trabajo)
    manifiesto = trabajo / "metadata.json"
    presentes = tuple(p.exists() or p.is_symlink() for p in (indice, numero, crl))
    _exigir(presentes in ((False, False, False), (True, True, True)),
            "revocación parcial: conservar archivos y revisar, sin reiniciar índice")
    if all(presentes):
        _archivo(numero)
        revocados = _indice(indice)
        _crl(c, ca, crl, ahora, revocados)
        meta = c.leer(manifiesto)
        _exigir(meta.get("version") == 1 and meta.get("alcance") == "f131-copia-nueva"
                and meta.get("copia_ref") == c.copia.nombre and meta.get("sid_copia") == sid
                and meta.get("ca_sha256") == _sha(ca)
                and meta.get("ca_clave_sha256") == _sha(clave)
                and meta.get("index_sha256") == _sha(indice)
                and meta.get("crlnumber_sha256") == _sha(numero)
                and meta.get("crl_sha256") == _sha(crl),
                "manifiesto de revocación divergente; no regenerar")
        return meta
    _exigir(not manifiesto.exists(), "manifiesto previo sin archivos de revocación")
    c.guardar(indice, b"")
    c.guardar(numero, NUMERO_INICIAL.encode())
    with tempfile.TemporaryDirectory(prefix="crl-", dir=trabajo) as temporal:
        temporal = Path(temporal)
        config = temporal / "ca.cnf"
        salida = temporal / "clientes.crl"
        c.guardar(config, ("[ ca ]\ndefault_ca=desarrollo\n[ desarrollo ]\n"
                           f"database={indice}\ncrlnumber={numero}\nprivate_key={clave}\n"
                           f"certificate={ca}\ndefault_md=sha256\ndefault_crl_days=90\n").encode())
        _run(c, "cert-base-crl-generar.log", "ca", "-gencrl", "-config", config, "-out", salida)
        salida.chmod(0o600)
        _crl(c, ca, salida, ahora, set())
        c.guardar(crl, _archivo(salida).read_bytes())
    _archivo(numero)
    revocados = _indice(indice)
    _crl(c, ca, crl, ahora, revocados)
    meta = {"version": 1, "alcance": "f131-copia-nueva", "copia_ref": c.copia.nombre,
            "sid_copia": sid, "ca_sha256": _sha(ca), "ca_clave_sha256": _sha(clave),
            "index_sha256": _sha(indice), "crlnumber_sha256": _sha(numero),
            "crl_sha256": _sha(crl), "index": str(indice), "crl": str(crl)}
    c.guardar(manifiesto, meta)
    return meta
