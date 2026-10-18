"""Conexiones nominales de CT y Bolsa para una copia local recién creada.

F0-08 sigue pendiente: el pool de gobierno usa SET ROLE al propietario AD3.
Durante el alta solo se añade SET TRUE en su arco al migrador AD3. El volcado carece de
membresías globales; se restaura solo el arco migrador→propietario autorizado
por roles_up.sql. El productor de datos completa los arcos de gobierno que
necesita el arranque; copia_local_verificacion coteja el contrato final.
El importador CONVOCA conserva sus dos grupos H1 con SET TRUE.
Los demás LOGIN reciben SET FALSE y una única membresía.
"""

from __future__ import annotations

import os
import json
from datetime import datetime, timedelta, timezone
from pathlib import Path
import re
import secrets
import stat
from urllib.parse import quote, urlencode


# Contratos vigentes de config y de los grupos comprobados por abrirPool.
# Mapa plano de la conexión y su grupo principal; la excepción H1 figura abajo.
GRUPOS = {
    "VEC_CT_DATABASE_URL": "vec_contratacion_temporal_ejecutor",
    "VEC_CT_GOBIERNO_DATABASE_URL": "vec_autorizacion_atestada_v3_migrador",
    "VEC_CT_REGISTRO_AUTORIZACION_DATABASE_URL": "vec_autorizacion_registro",
    "VEC_CT_CONFIRMADOR_DATABASE_URL": "vec_contratacion_temporal_confirmador_cobertura",
    "VEC_CT_LECTOR_RESULTADO_DATABASE_URL": "vec_contratacion_temporal_lector_resultado_cobertura",
    "VEC_BOLSA_LLAMAMIENTOS_DATABASE_URL": "vec_bolsa_llamamientos_ejecutor",
    "VEC_CT_CONSULTAS_RRHH_DATABASE_URL": "vec_contratacion_temporal_consultor_rrhh",
    "VEC_CT_MOTIVOS_RRHH_DATABASE_URL": "vec_autorizacion_motivos_rrhh_resolutor",
    "VEC_CT_REGISTRO_IDENTIDAD_DATABASE_URL": "vec_identidad_sesiones_v1_registrador",
    "VEC_CT_REVALIDACION_IDENTIDAD_DATABASE_URL": "vec_identidad_sesiones_v1_revalidador",
    "VEC_CT_CONTEXTO_ACTOR_DATABASE_URL": "vec_contexto_actor_v1_runtime",
    "VEC_CT_AUDITORIA_FRONTERA_DATABASE_URL": "vec_contratacion_temporal_registrador_frontera",
    "VEC_AUTORIZACION_FUENTE_DATABASE_URL": "vec_autorizacion_fuente",
    "VEC_AUTORIZACION_MOTIVOS_EVALUADOR_DATABASE_URL": "vec_autorizacion_motivos_evaluador",
    "VEC_BOLSA_AUDITORIA_FRONTERA_DATABASE_URL": "vec_bolsa_llamamientos_registrador_frontera",
    "VEC_BOLSA_POLITICA_OFERTAS_CALCULADOR_DATABASE_URL": "vec_bolsa_llamamientos_calculador_politica",
    "VEC_AUDITORIA_SELLADO_DATABASE_URL": "vec_auditoria_encadenador",
    "VEC_BOLSA_BORRADORES_EJECUTOR_CONSULTA_DATABASE_URL": "vec_bolsa_convocatorias_ejecutor_consulta",
    "VEC_BOLSA_BORRADORES_PROYECTOR_GOBIERNO_DATABASE_URL": "vec_bolsa_convocatorias_proyector_gobierno",
    "VEC_BOLSA_BORRADORES_VERIFICADOR_RECIBO_DATABASE_URL": "vec_bolsa_convocatorias_verificador_recibo",
    "VEC_BOLSA_IMPORTACION_CONVOCA_DATABASE_URL": "vec_bolsa_importacion_convoca_ejecutor",
    "VEC_BOLSA_B103_FUENTE_AUTORIZACION_DATABASE_URL": "vec_autorizacion_fuente",
}

RESULTADO = "conexiones.json"
GOBIERNO = "VEC_CT_GOBIERNO_DATABASE_URL"
CONVOCA = "VEC_BOLSA_IMPORTACION_CONVOCA_DATABASE_URL"
RECUPERADOR_CONVOCA = "vec_bolsa_importacion_convoca_recuperador"
MIGRADOR = "vec_autorizacion_atestada_v3_migrador"
PROPIETARIO_AD3 = "vec_autorizacion_atestada_v3_propietario"


def membresias(variable: str, grupo: str) -> dict[str, tuple[bool, bool]]:
    """Grupos directos y opciones INHERIT/SET del LOGIN de esta conexión."""
    if variable == CONVOCA:
        return {grupo: (True, True), RECUPERADOR_CONVOCA: (True, True)}
    return {grupo: (True, variable == GOBIERNO)}


def _privado(ruta: Path, *, directorio: bool) -> None:
    if ruta.is_symlink() or ruta.resolve(strict=True) != ruta:
        raise ValueError("Ruta de la copia enlazada")
    datos = ruta.stat()
    tipo = stat.S_ISDIR if directorio else stat.S_ISREG
    if not tipo(datos.st_mode) or datos.st_uid != os.getuid() or datos.st_mode & 0o077:
        raise ValueError("Material de copia con permisos inseguros")


def _nombre_instancia(copia) -> str:
    nombre = copia.nombre
    if not re.fullmatch(r"vec-copia-[a-z]-[a-z0-9][a-z0-9-]{0,40}", nombre):
        raise ValueError("Nombre de instancia inválido")
    return nombre


def _rol(nombre: str, indice: int) -> str:
    # 63 bytes como máximo en PostgreSQL; el sufijo evita truncamientos y
    # distingue incluso instancias cuyo prefijo sea igual.
    import hashlib
    etiqueta = nombre.removeprefix("vec-copia-").replace("-", "_")[:12]
    huella = hashlib.sha256(nombre.encode()).hexdigest()[:12]
    return f"vcl_{etiqueta}_{huella}_{indice:02d}"


def _sql(nombres: list[str], contrasenas: list[str], grupos: list[str], vence: str) -> bytes:
    """Plan completo: valida la preimagen antes de aplicar cualquier alta."""
    cadenas = lambda valores: ",".join("'" + valor + "'" for valor in valores)
    arcos = [(login, rol, hereda, cambia)
             for variable, login, grupo in zip(GRUPOS, nombres, grupos)
             for rol, (hereda, cambia) in membresias(variable, grupo).items()]
    pares = ",".join("('" + login + "','" + rol + "')" for login, rol, _, _ in arcos)
    pares_post = ",".join("('" + login + "','" + rol + "'," +
                          str(hereda).lower() + "," + str(cambia).lower() + ")"
                          for login, rol, hereda, cambia in arcos)
    lineas = [
        "BEGIN;",
        "SET LOCAL search_path=pg_catalog;",
        "SET LOCAL lock_timeout='3s';",
        "SET LOCAL statement_timeout='30s';",
        "SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:copia:conexiones:v1',0));",
        "DO $pre$ DECLARE v_fallos integer; BEGIN",
        "IF pg_catalog.current_database()<>'postgres' THEN RAISE EXCEPTION 'Base de copia inesperada'; END IF;",
        "IF EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname IN (" + cadenas(nombres) + ")) "
        "THEN RAISE EXCEPTION 'Un LOGIN nominal ya existe'; END IF;",
        "WITH esperados(login,grupo) AS (VALUES " + pares + ") "
        "SELECT count(*) INTO v_fallos FROM esperados e LEFT JOIN pg_catalog.pg_roles r ON r.rolname=e.grupo "
        "WHERE r.oid IS NULL OR r.rolcanlogin OR r.rolsuper OR r.rolcreatedb OR r.rolcreaterole "
        "OR r.rolreplication OR r.rolbypassrls OR r.rolconfig IS NOT NULL "
        "OR (r.rolname<>'vec_auditoria_encadenador' AND "
        "NOT pg_catalog.has_database_privilege(r.oid,'postgres','CONNECT')) "
        "OR (r.rolname<>'" + MIGRADOR + "' AND EXISTS "
        "(SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid));",
        "IF v_fallos<>0 THEN RAISE EXCEPTION 'Grupo nominal ausente o sin CONNECT seguro'; END IF;",
        # El primer corte reconstruyó ambos roles AD3 con INHERIT por defecto.
        # roles_up.sql:70-75 exige NOINHERIT. Un atributo divergente solo se
        # corrige sobre el rol todavía aislado, sin miembros ni arcos salientes.
        "IF EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m "
        "JOIN pg_catalog.pg_roles r ON r.oid=m.roleid "
        "WHERE r.rolname='" + MIGRADOR + "') "
        "THEN RAISE EXCEPTION 'Migrador AD3 ya tiene miembros'; END IF;",
        "IF EXISTS (SELECT 1 FROM pg_catalog.pg_roles r "
        "WHERE r.rolname IN ('" + MIGRADOR + "','" + PROPIETARIO_AD3 + "') "
        "AND r.rolinherit AND EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m "
        "WHERE m.member=r.oid OR m.roleid=r.oid)) "
        "THEN RAISE EXCEPTION 'Rol AD3 INHERIT con membresías'; END IF;",
        # Excepción heredada F0-08. Fuente:
        # deploy/postgresql/autorizacion_atestada_v3/roles_up.sql:83-85.
        "IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles r "
        "WHERE r.rolname='" + PROPIETARIO_AD3 + "' AND NOT r.rolcanlogin "
        "AND NOT r.rolsuper AND NOT r.rolcreatedb "
        "AND NOT r.rolcreaterole AND NOT r.rolreplication AND NOT r.rolbypassrls "
        "AND r.rolconfig IS NULL) "
        "THEN RAISE EXCEPTION 'Propietario AD3 distinto'; END IF;",
        "IF EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m "
        "JOIN pg_catalog.pg_roles r ON r.oid=m.member "
        "WHERE r.rolname='" + PROPIETARIO_AD3 + "') "
        "THEN RAISE EXCEPTION 'Propietario AD3 con membresía'; END IF;",
        "IF EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m "
        "JOIN pg_catalog.pg_roles g ON g.oid=m.roleid "
        "LEFT JOIN pg_catalog.pg_roles r ON r.oid=m.member "
        "WHERE g.rolname='" + PROPIETARIO_AD3 + "' "
        "AND (r.rolname IS DISTINCT FROM '" + MIGRADOR + "' OR "
        "m.admin_option OR m.inherit_option OR NOT m.set_option)) "
        "THEN RAISE EXCEPTION 'Propietario AD3 con miembro ajeno'; END IF;",
        "IF EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m "
        "JOIN pg_catalog.pg_roles r ON r.oid=m.member "
        "LEFT JOIN pg_catalog.pg_roles g ON g.oid=m.roleid "
        "WHERE r.rolname='" + MIGRADOR + "' "
        "AND (g.rolname IS DISTINCT FROM '" + PROPIETARIO_AD3 + "' "
        "OR m.admin_option OR m.inherit_option OR NOT m.set_option)) "
        "THEN RAISE EXCEPTION 'Arco AD3 heredado distinto'; END IF;",
        "IF (SELECT count(*) FROM pg_catalog.pg_auth_members m "
        "JOIN pg_catalog.pg_roles r ON r.oid=m.member "
        "WHERE r.rolname='" + MIGRADOR + "')>1 "
        "THEN RAISE EXCEPTION 'Arcos AD3 duplicados'; END IF;",
        "END $pre$;",
        "DO $atributo$ BEGIN "
        "IF EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname='" + PROPIETARIO_AD3 + "' AND rolinherit) "
        "THEN EXECUTE 'ALTER ROLE " + PROPIETARIO_AD3 + " NOINHERIT'; END IF; "
        "IF EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname='" + MIGRADOR + "' AND rolinherit) "
        "THEN EXECUTE 'ALTER ROLE " + MIGRADOR + " NOINHERIT'; END IF; END $atributo$;",
        "DO $arco$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m "
        "JOIN pg_catalog.pg_roles r ON r.oid=m.member "
        "JOIN pg_catalog.pg_roles g ON g.oid=m.roleid "
        "WHERE r.rolname='" + MIGRADOR + "' AND g.rolname='" + PROPIETARIO_AD3 + "') "
        "THEN EXECUTE 'GRANT " + PROPIETARIO_AD3 + " TO " + MIGRADOR +
        " WITH ADMIN FALSE, INHERIT FALSE, SET TRUE'; END IF; END $arco$;",
    ]
    for variable, login, clave, grupo in zip(GRUPOS, nombres, contrasenas, grupos):
        lineas.append("CREATE ROLE " + login + " LOGIN INHERIT NOSUPERUSER NOCREATEDB "
                      "NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD '" + clave +
                      "' VALID UNTIL '" + vence + "';")
        if grupo == "vec_auditoria_encadenador":
            # El anexo de base no concede CONNECT al sellador. Se otorga
            # únicamente al nuevo LOGIN, sin alterar el grupo instalado.
            lineas.append("GRANT CONNECT ON DATABASE postgres TO " + login + ";")
        for rol, (hereda, cambia) in membresias(variable, grupo).items():
            lineas.append("GRANT " + rol + " TO " + login +
                          " WITH ADMIN FALSE, INHERIT " + ("TRUE" if hereda else "FALSE") +
                          ", SET " + ("TRUE" if cambia else "FALSE") + ";")
    lineas.extend((
        "DO $post$ DECLARE v_fallos integer; BEGIN "
        "WITH esperados(login,grupo,hereda,cambia) AS (VALUES " + pares_post + ") "
        "SELECT count(*) INTO v_fallos FROM esperados e "
        "LEFT JOIN pg_catalog.pg_roles r ON r.rolname=e.login "
        "LEFT JOIN pg_catalog.pg_roles g ON g.rolname=e.grupo "
        "WHERE r.oid IS NULL OR NOT r.rolcanlogin OR NOT r.rolinherit OR "
        "r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR "
        "r.rolbypassrls OR r.rolconfig IS NOT NULL OR "
        "r.rolvaliduntil IS DISTINCT FROM '" + vence + "'::pg_catalog.timestamptz OR "
        "(SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid)<>"
        "(SELECT count(*) FROM esperados x WHERE x.login=e.login) OR "
        "NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m "
        "WHERE m.member=r.oid AND m.roleid=g.oid AND m.inherit_option=e.hereda "
        "AND m.set_option=e.cambia AND NOT m.admin_option); "
        "IF v_fallos<>0 THEN RAISE EXCEPTION 'LOGIN nominal no aislado'; END IF; "
        "IF (SELECT count(*) FROM pg_catalog.pg_roles r "
        "WHERE r.rolname IN ('" + MIGRADOR + "','" + PROPIETARIO_AD3 + "') "
        "AND NOT r.rolinherit)<>2 "
        "THEN RAISE EXCEPTION 'Roles AD3 conservan INHERIT'; END IF; "
        "IF (SELECT count(*) FROM pg_catalog.pg_auth_members m "
        "JOIN pg_catalog.pg_roles r ON r.oid=m.member "
        "JOIN pg_catalog.pg_roles g ON g.oid=m.roleid "
        "WHERE r.rolname='" + MIGRADOR + "' AND g.rolname='" + PROPIETARIO_AD3 + "' "
        "AND NOT m.admin_option AND NOT m.inherit_option AND m.set_option)<>1 "
        "THEN RAISE EXCEPTION 'Arco AD3 no restaurado'; END IF; END $post$;",
        "COMMIT;",
    ))
    return ("\n".join(lineas) + "\n").encode()


def _publicar(ruta: Path, contenido: str) -> None:
    # El nombre final se reserva con hardlink exclusivo al temporal completo.
    temporal = ruta.parent / ("." + ruta.name + "." + secrets.token_hex(12))
    descriptor = os.open(temporal, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    try:
        with os.fdopen(descriptor, "w", encoding="utf-8") as salida:
            salida.write(contenido)
            salida.flush()
            os.fsync(salida.fileno())
        os.link(temporal, ruta, follow_symlinks=False)
        padre = os.open(ruta.parent, os.O_RDONLY | os.O_DIRECTORY)
        try:
            os.fsync(padre)
        finally:
            os.close(padre)
    finally:
        temporal.unlink(missing_ok=True)


def preparar(copia) -> dict[str, str]:
    """Provisiona una instancia nueva; un fallo exige descartar esa instancia."""
    nombre = _nombre_instancia(copia)
    raiz = Path(copia.raiz)
    material = Path(copia.material)
    _privado(raiz, directorio=True)
    _privado(material, directorio=True)
    ca = material / "ca" / "ca.crt"
    if ca.is_symlink() or not ca.is_file() or ca.resolve(strict=True) != ca:
        raise ValueError("CA propia ausente o enlazada")
    estado = copia.leer()
    puerto = estado.get("puerto_pg")
    if (estado.get("nombre") != nombre or estado.get("revision") is None or
            estado.get("preparada") or not isinstance(puerto, int) or not 1024 <= puerto <= 65535):
        raise ValueError("La instancia no es nueva o su estado es inválido")
    destino = raiz / RESULTADO
    if destino.exists() or destino.is_symlink():
        raise ValueError("Las conexiones de la copia ya existen")
    nombres = [_rol(nombre, i) for i in range(len(GRUPOS))]
    claves = [secrets.token_hex(32) for _ in nombres]
    vence = (datetime.now(timezone.utc) + timedelta(days=90)).strftime("%Y-%m-%dT%H:%M:%SZ")
    grupos = list(GRUPOS.values())
    env = {variable: "postgresql://" + quote(login) + ":" + quote(clave) +
           "@localhost:" + str(puerto) + "/postgres?" + urlencode(
               {"sslmode": "verify-full", "sslrootcert": str(ca)})
           for variable, login, clave in zip(GRUPOS, nombres, claves)}
    # sql_bytes usa stdin del contenedor; la contraseña nunca pasa por argv.
    # El log es privado de la instancia y psql no imprime sentencias válidas.
    copia.sql_bytes(_sql(nombres, claves, grupos, vence), "conexiones.log")
    _publicar(destino, json.dumps(env, ensure_ascii=False, sort_keys=True) + "\n")
    return env
