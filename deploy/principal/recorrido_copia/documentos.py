"""Conexiones documentales propias, cuando existe su configuración de datos."""

from datetime import datetime, timedelta, timezone
import json
import os
import secrets
from urllib.parse import parse_qs, quote, urlencode, urlsplit


REUTILIZA = {
    "dsn_registro_identidad": "VEC_CT_REGISTRO_IDENTIDAD_DATABASE_URL",
    "dsn_revalidacion_identidad": "VEC_CT_REVALIDACION_IDENTIDAD_DATABASE_URL",
    "dsn_contexto": "VEC_CT_CONTEXTO_ACTOR_DATABASE_URL",
    "dsn_fuente_autorizacion": "VEC_AUTORIZACION_FUENTE_DATABASE_URL",
    "dsn_registro_autorizacion": "VEC_CT_REGISTRO_AUTORIZACION_DATABASE_URL",
    "dsn_motivos": "VEC_AUTORIZACION_MOTIVOS_EVALUADOR_DATABASE_URL",
}
NUEVOS = {
    "dsn_documentos": ("vec_documentos_rrhh_ejecutor_desarrollo", "vec_documentos_ejecutor"),
    "dsn_documentos_auditor": ("vec_documentos_rrhh_auditor_desarrollo", "vec_documentos_auditor"),
}


def preparar_documentos(copia, base: dict, contexto=None) -> None:
    """Completa una configuración propia recibida del generador de la copia.

    ``base`` solo contiene cuentas, motivos, registro externo y custodia
    propios ya cotejados; nunca se abre un material de otra instalación.
    """
    estado = copia.leer()
    if not copia.existe():
        raise ValueError("La copia no está activa")
    if estado.get("preparada"):
        from .comun import Contexto
        if (not estado.get("recorrido_completo") or not isinstance(contexto, Contexto)
                or contexto.copia is not copia or contexto.raiz != copia.raiz
                or contexto.material != copia.material
                or contexto.consultar('SELECT system_identifier::text FROM pg_catalog.pg_control_system();')
                   != estado.get('sid_copia')):
            raise ValueError("La copia preparada no pertenece al recorrido completo")
        entorno_ruta = copia.raiz / "entorno_modulos.json"
        if (not entorno_ruta.is_file() or entorno_ruta.is_symlink()
                or contexto.sha(entorno_ruta) != estado.get('entorno_modulos_sha256')):
            raise ValueError("Las conexiones vigentes no corresponden a la copia")
        conexiones = contexto.leer(entorno_ruta)
        for variable in REUTILIZA.values():
            dsn = conexiones.get(variable)
            if not isinstance(dsn, str):
                raise ValueError("Faltan conexiones documentales vigentes")
            u = urlsplit(dsn)
            parametros = parse_qs(u.query, strict_parsing=True)
            if (u.scheme != 'postgresql' or not u.username or not u.password
                    or u.hostname != 'localhost' or u.port != estado.get('puerto_pg')
                    or u.path != '/postgres' or parametros.get('sslmode') != ['verify-full']
                    or parametros.get('sslrootcert') != [str(copia.material / 'ca/ca.crt')]):
                raise ValueError("Conexión documental ajena o incompleta")
    destino = copia.material / "identidad/documentos.json"
    if destino.exists() or not isinstance(base, dict) or base.get("version") != 1 or \
            base.get("autoridad") != "no_autoritativo" or not base.get("cuentas") or \
            not base.get("motivos", {}).get("listar"):
        raise ValueError("Falta configuración documental propia completa")
    entorno = json.loads((copia.raiz / "entorno_modulos.json").read_text())
    config = {k: v for k, v in base.items() if not k.startswith("dsn_")}
    for campo, variable in REUTILIZA.items():
        config[campo] = entorno[variable]
    sql = ["BEGIN;", "SET LOCAL lock_timeout='5s';"]
    vence = (datetime.now(timezone.utc) + timedelta(days=90)).strftime("%Y-%m-%dT%H:%M:%SZ")
    for campo, (login, grupo) in NUEVOS.items():
        secreto = secrets.token_hex(32)
        sql.append("CREATE ROLE " + login + " LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE "
                   "NOREPLICATION NOBYPASSRLS PASSWORD '" + secreto + "' VALID UNTIL '" + vence + "';")
        sql.append("GRANT " + grupo + " TO " + login +
                   " WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;")
        config[campo] = ("postgresql://" + quote(login) + ":" + quote(secreto) +
                         "@localhost:" + str(estado["puerto_pg"]) + "/postgres?" + urlencode({
                             "sslmode": "verify-full", "sslrootcert": str(copia.material / "ca/ca.crt")}))
    sql.append("COMMIT;")
    almacen = copia.raiz / "documentos-almacen"
    almacen.mkdir(mode=0o700, exist_ok=False)
    config["almacen"] = {**config.get("almacen", {}), "directorio": str(almacen)}
    copia.sql_bytes(("\n".join(sql) + "\n").encode(), "documentos-sql.log")
    with open(destino, "x", encoding="utf-8", opener=lambda p, f: os.open(p, f, 0o600)) as salida:
        json.dump(config, salida, ensure_ascii=False, indent=2)
        salida.write("\n")
