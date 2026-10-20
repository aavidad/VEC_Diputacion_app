"""«Mis preferencias» y «Mi imagen» en una copia local con módulos (F1-113).

Sin este paso, `/api/vec/usuarios/mis-preferencias` y `/mi-imagen` no se
montaban en la copia y respondían 404: la copia no encendía sus selectores
(`VEC_USUARIOS_PREFERENCIAS_ENABLED`, `VEC_USUARIOS_IMAGEN_ENABLED`) y los dos
ficheros de configuración de Usuarios que trae el material original de RRHH
(`identidad/usuarios-preferencias-interna.json` y `-externa.json`) apuntaban a
conexiones de la principal que no existen en la copia.

El material de la copia no traslada conexiones de la principal
(copia_local_rrhh.py). De esos ficheros originales solo se toman las cuentas,
la versión, la superficie y los motivos; las conexiones se crean nuevas.

Los permisos ya están publicados en el volcado de referencia (roles
`usuarios_preferencias_interna_h3` y `_externa_h3`, asignados a las cuentas de
esos dos ficheros). Aquí no se escribe ninguna asignación ni versión de rol:
solo se crean LOGIN nominales propios de la copia, uno por conexión y con una
única membresía (INHERIT TRUE, SET FALSE), y se reescriben las conexiones de
los dos ficheros. Las cuentas, perfiles y motivos se conservan tal cual. El
ejecutor de cada superficie se llama como el LOGIN de su fila de origen de
consumos ya restaurada (`configuracion_origen_consumos_v1`), igual que el de
Documentos: sin ese nombre la base rechaza el consumo («origen de consumo no
acreditado») y la consulta respondía 503. La fila no se toca.

La cuenta interna (la técnica de RRHH del material original) se conserva y
debe ser la del certificado `mtls/cliente.crt` de la copia. La cuenta externa
de la principal no existe en la copia y el arranque rechaza una cuenta que no
reconoce (dejaría sin Contratación temporal todo el proceso). Por eso la
configuración externa usa la persona aspirante de la copia
(`identidad/bolsa-candidato.json`): Antonio en la copia con módulos o el
aspirante del área personal con `--portal-externo`. Esa persona no tiene
publicado el rol de preferencias de Usuarios, así que su consulta no se
atiende (la pantalla avisa de que no está disponible) hasta que Administración
le asigne ese rol.

Solo se aplica cuando el material original trae los dos ficheros
(`--rrhh-original`) y la copia ya tiene su aspirante. Con material nuevo no hay
cuentas con esos permisos y la copia sigue sin ellos.

F1-130 aporta la fuente de preferencias e imagen propias y el enlace de cada
identidad interna posterior. La orden que publica su cuenta y perfil llama a
vincular_cuenta_interna antes del reinicio. El enlace conserva las conexiones
y las cuentas existentes; no sustituye la publicación ni la asignación.
"""
from __future__ import annotations

from datetime import datetime, timedelta, timezone
import hashlib
import json
import re
from pathlib import Path
import secrets
import ssl
import subprocess
from urllib.parse import quote, urlencode

from copia_local_conexiones import _nombre_instancia, _privado, _publicar, _rol
from preparar_rrhh_adicional_copia import leer_json

SUPERFICIES = ("interna", "externa")
FICHERO = "identidad/usuarios-preferencias-{}.json"
# Clave de cada conexión en el fichero y grupo que exige el montaje
# (usuarios_preferencias_preflight.go y usuarios_preferencias_postgresql.go).
CONEXIONES = {
    "dsn_registro_identidad": "vec_identidad_sesiones_v1_registrador",
    "dsn_revalidacion_identidad": "vec_identidad_sesiones_v1_revalidador",
    "dsn_contexto": "vec_contexto_actor_v1_runtime",
    "dsn_fuente_autorizacion": "vec_autorizacion_fuente",
    "dsn_registro_autorizacion": "vec_autorizacion_registro",
    "dsn_motivos": "vec_autorizacion_motivos_evaluador",
    "dsn_usuarios": "vec_usuarios_ejecutor_{}",
    "dsn_usuarios_frontera": "vec_usuarios_registrador_frontera_{}",
}
SELECTORES = {"VEC_USUARIOS_PREFERENCIAS_ENABLED": "true", "VEC_USUARIOS_IMAGEN_ENABLED": "true"}
# Los índices 0..len(GRUPOS)-1 son de copia_local_conexiones; estos no chocan.
PRIMER_INDICE = 60
RESULTADO = "usuarios.json"


CONSERVADAS = ("version", "autoridad", "superficie", "cuentas", "motivo_consulta", "motivo_actualizacion")


def disponible(origen) -> bool:
    """El material original trae las dos configuraciones de Usuarios."""
    return origen is not None and all((Path(origen) / FICHERO.format(s)).is_file() for s in SUPERFICIES)


def huella_certificado(ruta: Path) -> str:
    """SHA-256 del certificado en DER, como lo calcula el resolvedor."""
    return hashlib.sha256(ssl.PEM_cert_to_DER_cert(ruta.read_text())).hexdigest()


def cuenta_aspirante(material: Path) -> dict:
    """Cuenta externa: la persona aspirante de la copia, no la de la principal."""
    datos = json.loads((material / "identidad/bolsa-candidato.json").read_text())
    if datos.get("certificado") != "mtls/candidato.crt" or not all(
            isinstance(datos.get(c), str) and datos[c] for c in ("sujeto", "cuenta_ref", "perfil_ref")):
        raise ValueError("Aspirante de la copia incompleto")
    return {"certificado_sha256": huella_certificado(material / "mtls/candidato.crt"),
            "sujeto": datos["sujeto"], "cuenta_ref": datos["cuenta_ref"], "perfil_ref": datos["perfil_ref"]}


CANAL = {"interna": "interna_corporativa", "externa": "externa_personal"}


def consulta_login_origen(superficie: str) -> bytes:
    """SQL de solo lectura: LOGIN de la fila de origen de la consulta de preferencias."""
    return ("SELECT login_nombre FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1 "
            "WHERE audiencia_consumo='vec_usuarios.preferencias.consultar." + CANAL[superficie] +
            ".v1' AND operacion='vec.preferencias.consultar' AND proceso='vec-server' "
            "AND canal_permitido='" + CANAL[superficie] + "';\n").encode()


def login_origen(copia, superficie: str) -> str:
    salida = subprocess.run(["docker", "exec", "-i", "-u", "postgres", copia.nombre, "psql", "-X", "-At",
                             "-v", "ON_ERROR_STOP=1", "-U", "postgres", "-d", "postgres"],
                            input=consulta_login_origen(superficie), capture_output=True,
                            check=True, timeout=60).stdout.decode().split()
    if len(salida) != 1 or not re.fullmatch(r"[a-z_][a-z0-9_]{0,62}", salida[0]):
        raise ValueError("Fila de origen de Usuarios ausente o repetida")
    return salida[0]


def plan() -> list[tuple[str, str, str]]:
    """(superficie, clave del fichero, grupo) en orden fijo: 16 LOGIN distintos."""
    return [(s, clave, grupo.format("interno" if s == "interna" else "externo"))
            for s in SUPERFICIES for clave, grupo in CONEXIONES.items()]


def sql(logins: list[str], claves: list[str], grupos: list[str], vence: str) -> bytes:
    """Altas en una transacción: comprueba grupos y que ningún LOGIN exista."""
    cadena = lambda valores: ",".join("'" + v + "'" for v in valores)
    pares = ",".join("('" + l + "','" + g + "')" for l, g in zip(logins, grupos))
    lineas = [
        "BEGIN;",
        "SET LOCAL search_path=pg_catalog;",
        "SET LOCAL lock_timeout='3s';",
        "SET LOCAL statement_timeout='30s';",
        "DO $pre$ BEGIN",
        "IF pg_catalog.current_database()<>'postgres' THEN RAISE EXCEPTION 'Base de copia inesperada'; END IF;",
        "IF EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname IN (" + cadena(logins) + ")) "
        "THEN RAISE EXCEPTION 'Un LOGIN de Usuarios ya existe'; END IF;",
        "IF EXISTS (SELECT 1 FROM (VALUES " + pares + ") e(login,grupo) "
        "LEFT JOIN pg_catalog.pg_roles r ON r.rolname=e.grupo "
        "WHERE r.oid IS NULL OR r.rolcanlogin OR r.rolsuper OR r.rolcreatedb OR r.rolcreaterole "
        "OR r.rolreplication OR r.rolbypassrls "
        "OR NOT pg_catalog.has_database_privilege(r.oid,'postgres','CONNECT') "
        "OR EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid)) "
        "THEN RAISE EXCEPTION 'Grupo de Usuarios ausente o no aislado'; END IF;",
        "END $pre$;",
    ]
    for login, clave, grupo in zip(logins, claves, grupos):
        lineas.append("CREATE ROLE " + login + " LOGIN INHERIT NOSUPERUSER NOCREATEDB "
                      "NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD '" + clave +
                      "' VALID UNTIL '" + vence + "';")
        lineas.append("GRANT " + grupo + " TO " + login + " WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;")
    lineas.extend((
        "DO $post$ BEGIN IF EXISTS (SELECT 1 FROM (VALUES " + pares + ") e(login,grupo) "
        "LEFT JOIN pg_catalog.pg_roles r ON r.rolname=e.login "
        "WHERE r.oid IS NULL OR NOT r.rolcanlogin OR NOT r.rolinherit "
        "OR (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid)<>1 "
        "OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m JOIN pg_catalog.pg_roles g ON g.oid=m.roleid "
        "WHERE m.member=r.oid AND g.rolname=e.grupo AND m.inherit_option AND NOT m.set_option "
        "AND NOT m.admin_option)) THEN RAISE EXCEPTION 'LOGIN de Usuarios no aislado'; END IF; END $post$;",
        "COMMIT;",
    ))
    return ("\n".join(lineas) + "\n").encode()


def configuracion(original: dict, dsns: dict[str, str], superficie: str) -> dict:
    """Cuentas y motivos originales con las conexiones de la copia.

    Ninguna conexión de la principal pasa a la copia: solo se conservan los
    campos de CONSERVADAS y el resto se rechaza si no es una conexión conocida.
    """
    if (not isinstance(original, dict) or set(original) != set(CONSERVADAS) | set(CONEXIONES)
            or set(dsns) != set(CONEXIONES)
            or original.get("superficie") != superficie + ("_corporativa" if superficie == "interna" else "_personal")):
        raise ValueError("Configuración de Usuarios original inesperada")
    nuevo = {clave: original[clave] for clave in CONSERVADAS}
    nuevo.update(dsns)
    return nuevo


def entorno_usuarios(entorno: dict) -> dict:
    """Enciende los dos selectores; un «false» puesto por el kit se respeta.

    La imagen depende de las preferencias: si estas se apagan, la imagen no se
    enciende sola, porque sin ellas el arranque apaga Contratación temporal."""
    nuevo = dict(entorno)
    if str(nuevo.get("VEC_USUARIOS_PREFERENCIAS_ENABLED", "")).lower() == "false":
        if not nuevo.get("VEC_USUARIOS_IMAGEN_ENABLED"):
            nuevo["VEC_USUARIOS_IMAGEN_ENABLED"] = "false"
    for variable, valor in SELECTORES.items():
        if not nuevo.get(variable):
            nuevo[variable] = valor
    return nuevo


def preparar(copia, origen, guardar, sha256) -> bool:
    """Provisiona Usuarios en una copia nueva. Devuelve si se ha encendido."""
    if not disponible(origen) or not (Path(copia.material) / "identidad/bolsa-candidato.json").is_file():
        return False
    nombre = _nombre_instancia(copia)
    raiz, material = Path(copia.raiz), Path(copia.material)
    _privado(raiz, directorio=True)
    _privado(material, directorio=True)
    destino = raiz / RESULTADO
    if destino.exists() or destino.is_symlink():
        raise ValueError("Las conexiones de Usuarios de la copia ya existen")
    estado = copia.leer()
    puerto = estado["puerto_pg"]
    ca = material / "ca" / "ca.crt"
    filas = plan()
    ejecutores = {s: login_origen(copia, s) for s in SUPERFICIES}
    logins = [ejecutores[s] if clave == "dsn_usuarios" else _rol(nombre, PRIMER_INDICE + i)
              for i, (s, clave, _) in enumerate(filas)]
    claves = [secrets.token_hex(32) for _ in filas]
    vence = (datetime.now(timezone.utc) + timedelta(days=90)).strftime("%Y-%m-%dT%H:%M:%SZ")
    dsn = lambda login, clave: ("postgresql://" + quote(login) + ":" + quote(clave) + "@localhost:" +
                                str(puerto) + "/postgres?" +
                                urlencode({"sslmode": "verify-full", "sslrootcert": str(ca)}))
    nuevas = {}
    for superficie in SUPERFICIES:
        ruta = material / FICHERO.format(superficie)
        if ruta.exists() or ruta.is_symlink():
            raise ValueError("La configuración de Usuarios de la copia ya existe")
        dsns = {clave: dsn(login, secreto) for (s, clave, _), login, secreto
                in zip(filas, logins, claves) if s == superficie}
        original = json.loads((Path(origen) / FICHERO.format(superficie)).read_text())
        nuevas[ruta] = configuracion(original, dsns, superficie)
    rrhh = huella_certificado(material / "mtls/cliente.crt")
    interna = material / FICHERO.format("interna")
    if [c.get("certificado_sha256") for c in nuevas[interna]["cuentas"]] != [rrhh]:
        raise ValueError("La cuenta interna de Usuarios no es la técnica de RRHH de la copia")
    nuevas[material / FICHERO.format("externa")]["cuentas"] = [cuenta_aspirante(material)]
    # La contraseña va por stdin del contenedor, nunca por argv.
    copia.sql_bytes(sql(logins, claves, [g for _, _, g in filas], vence), "usuarios.log")
    for ruta, contenido in nuevas.items():
        _publicar(ruta, json.dumps(contenido, ensure_ascii=False, indent=2) + "\n")
    _publicar(destino, json.dumps(sorted(logins)) + "\n")
    entorno = raiz / "entorno_modulos.json"
    guardar(entorno, entorno_usuarios(copia.entorno_modulos()))
    estado = copia.leer()
    estado["entorno_modulos_sha256"] = sha256(entorno)
    guardar(copia.estado, estado)
    return True


def vincular_cuenta_interna(material, identidad, certificado, cuenta_ref, perfil_ref) -> bool:
    """Enlaza una identidad ya publicada, antes del reinicio de su copia.

    La orden de identidad llama con su cuenta y perfil confirmados por el
    gobierno existente, bajo el candado de la instancia. Este enlace no
    publica roles, asignaciones ni permisos; Usuarios los revalida al consultar.
    Las dos rutas son relativas al material, en identidad/ y mtls/.
    Devuelve False si el vínculo exacto ya existe, sin reescribir el fichero.
    """
    from copia_local import guardar

    material = Path(material)
    _privado(material, directorio=True)
    for carpeta in (material, *material.parents):
        if (carpeta / '.git').exists():
            raise ValueError('El material de Usuarios debe quedar fuera de Git')
    identidad = _ruta_identidad_usuarios(material, identidad, 'identidad')
    certificado = _ruta_identidad_usuarios(material, certificado, 'mtls')
    principal = leer_json(identidad)
    huella = huella_certificado(certificado)
    sujeto = principal.get('subject')
    if (principal.get('version') != 1 or principal.get('autoridad') != 'no_autoritativo'
            or principal.get('certificate_sha256') != huella
            or not isinstance(sujeto, str) or not sujeto or any(ord(c) < 32 for c in sujeto)
            or not isinstance(cuenta_ref, str) or not re.fullmatch(r'cta_[A-Za-z0-9_-]{22,123}', cuenta_ref)
            or not isinstance(perfil_ref, str) or not re.fullmatch(r'prf_[A-Za-z0-9_-]{22,123}', perfil_ref)):
        raise ValueError('La cuenta de Usuarios no coincide con su identidad certificada')
    nueva = dict(certificado_sha256=huella, sujeto=sujeto, cuenta_ref=cuenta_ref, perfil_ref=perfil_ref)
    interna = material / FICHERO.format('interna')
    externa = material / FICHERO.format('externa')
    for ruta in (interna, externa):
        _privado(ruta, directorio=False)
        if ruta.stat().st_size > 128 * 1024:
            raise ValueError('La configuración de Usuarios supera el límite del servidor')
    configuracion_interna, configuracion_externa = leer_json(interna), leer_json(externa)
    if (any(set(c) != set(CONSERVADAS) | set(CONEXIONES) or c.get('version') != 1
            or c.get('autoridad') != 'no_autoritativo' for c in (configuracion_interna, configuracion_externa))
            or configuracion_interna.get('superficie') != CANAL['interna']
            or configuracion_externa.get('superficie') != CANAL['externa']):
        raise ValueError('Las configuraciones de Usuarios no separan sus superficies')
    internas = _cuentas_usuarios(configuracion_interna)
    externas = _cuentas_usuarios(configuracion_externa)
    claves = ('certificado_sha256', 'cuenta_ref', 'perfil_ref')
    for cuenta in externas:
        if any(cuenta[k] == nueva[k] for k in claves):
            raise ValueError('Una cuenta externa no puede enlazarse como interna')
    for cuenta in internas:
        if cuenta == nueva:
            return False
        if any(cuenta[k] == nueva[k] for k in claves):
            raise ValueError('La cuenta interna ya tiene otro vínculo; se conserva el original')
    if len(internas) >= 64:
        raise ValueError('La configuración interna alcanza el límite del servidor')
    configuracion_interna['cuentas'] = [*internas, nueva]
    contenido = json.dumps(configuracion_interna, ensure_ascii=False, indent=2) + '\n'
    if len(contenido.encode()) > 128 * 1024:
        raise ValueError('La configuración de Usuarios supera el límite del servidor')
    guardar(interna, configuracion_interna)
    return True


def _ruta_identidad_usuarios(material, relativa, carpeta):
    relativa = Path(relativa)
    if relativa.is_absolute() or '..' in relativa.parts or len(relativa.parts) != 2 or relativa.parts[0] != carpeta:
        raise ValueError('La identidad de Usuarios debe pertenecer al material de su copia')
    ruta = material / relativa
    _privado(ruta, directorio=False)
    return ruta


def _cuentas_usuarios(configuracion):
    cuentas = configuracion.get('cuentas')
    claves = {'certificado_sha256', 'sujeto', 'cuenta_ref', 'perfil_ref'}
    if not isinstance(cuentas, list) or not 1 <= len(cuentas) <= 64:
        raise ValueError('Las cuentas de Usuarios no corresponden al contrato')
    vistas = {k: set() for k in claves - {'sujeto'}}
    for cuenta in cuentas:
        if (not isinstance(cuenta, dict) or set(cuenta) != claves
                or any(not isinstance(v, str) or not v for v in cuenta.values())
                or not re.fullmatch(r'[0-9a-f]{64}', cuenta['certificado_sha256'])
                or any(cuenta[k] in valores for k, valores in vistas.items())):
            raise ValueError('Las cuentas de Usuarios están incompletas o repetidas')
        for clave, valores in vistas.items():
            valores.add(cuenta[clave])
    return cuentas
