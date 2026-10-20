"""Registradores nominales de intentos en los dos canales de la copia."""

from datetime import datetime, timedelta, timezone
import hashlib
import json
import os
import secrets
from urllib.parse import quote, urlencode


def preparar_intentos(copia) -> None:
    estado = copia.leer()
    if estado.get("preparada") or not copia.existe():
        raise ValueError("La copia no es nueva o no está activa")
    material = copia.material
    vence = (datetime.now(timezone.utc) + timedelta(days=90)).strftime("%Y-%m-%dT%H:%M:%SZ")
    sql = ["BEGIN;"]
    salidas = []
    for canal, sufijo in (("interna_corporativa", ""), ("externa_personal", "-externa")):
        login = "vcl_int_" + hashlib.sha256((copia.nombre + canal).encode()).hexdigest()[:20]
        clave = secrets.token_hex(32)
        proceso = "vec-copia-externa" if sufijo else "vec-rrhh"
        sql.extend(("CREATE ROLE " + login + " LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE "
                    "NOREPLICATION NOBYPASSRLS PASSWORD '" + clave + "' VALID UNTIL '" + vence + "';",
                    "GRANT vec_autorizacion_atestada_v3_registrador_intentos TO " + login +
                    " WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;",
                    "INSERT INTO vec_autorizacion_atestada_v3.configuracion_runtime_intentos"
                    "(login_nombre,proceso,canal) VALUES('" + login + "','" + proceso + "','" + canal + "');"))
        dsn = ("postgresql://" + quote(login) + ":" + quote(clave) + "@localhost:" +
               str(estado["puerto_pg"]) + "/postgres?" + urlencode({
                   "sslmode": "verify-full", "sslrootcert": str(material / "ca/ca.crt")}))
        salidas.append((sufijo, canal, proceso, dsn))
    sql.append("COMMIT;")
    copia.sql_bytes(("\n".join(sql) + "\n").encode(), "datos-intentos-sql.log")
    for sufijo, canal, proceso, dsn in salidas:
        archivo = "auditoria-intentos" + sufijo + ".dsn"
        with open(material / archivo, "x", opener=lambda p, f: os.open(p, f, 0o600)) as salida:
            salida.write(dsn + "\n")
        with open(material / ("auditoria-intentos" + sufijo + ".json"),
                "x", opener=lambda p, f: os.open(p, f, 0o600)) as salida:
            json.dump({"esquema": "vec.auditoria.intentos.servidor.v1", "dsn_file": archivo,
                       "proceso": proceso, "canal": canal, "limite_segundos": 10}, salida)
            salida.write("\n")
