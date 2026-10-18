"""Cotejo del estado nominal tras preparar los datos ficticios de copia."""
import re
import subprocess
from urllib.parse import parse_qs, urlsplit

from copia_local_conexiones import GRUPOS, GOBIERNO, membresias

# Excepción de arranque hasta retirar el gobierno por propietarios (F0-08).
# No se aplica a los LOGIN que atienden peticiones.
ARCOS_GOBIERNO = {
    'vec_autorizacion_atestada_v3_migrador': (True, True),
    'vec_contratacion_temporal_gobernador': (True, False),
    'vec_identidad_sesiones_v1_propietario': (False, True),
    'vec_contexto_actor_v1_propietario': (False, True),
    'vec_autorizacion_propietario': (False, True),
    'vec_autorizacion_motivos_proyector': (False, True),
}
MOTIVOS = 'VEC_CT_MOTIVOS_RRHH_DATABASE_URL'


def consulta(copia):
    entorno = copia.entorno_modulos()
    estado = copia.leer()
    arcos, logins = [], []
    vistos = set()
    for variable, grupo in GRUPOS.items():
        conexion = urlsplit(entorno[variable])
        opciones = parse_qs(conexion.query)
        login = conexion.username or ''
        if (not re.fullmatch('[a-z][a-z0-9_]{2,62}', login) or login in vistos
                or conexion.hostname != 'localhost' or conexion.port != estado['puerto_pg']
                or conexion.path != '/postgres' or not conexion.password or conexion.fragment
                or opciones != {'sslmode': ['verify-full'],
                                'sslrootcert': [str(copia.material / 'ca/ca.crt')]}):
            raise ValueError('Conexión nominal final incompatible con la copia')
        vistos.add(login)
        # El resolutor instalado exige caducidad NULL; no se modifica su guarda.
        logins.append(f"('{login}',{'true' if variable == MOTIVOS else 'false'})")
        permitidos = ARCOS_GOBIERNO if variable == GOBIERNO else membresias(variable, grupo)
        for rol, (hereda, cambia) in permitidos.items():
            arcos.append(f"('{login}','{rol}',{str(hereda).lower()},{str(cambia).lower()})")
    return ("WITH logins(login,sin_caducidad) AS (VALUES " + ','.join(logins) + "), "
            "esperados(login,grupo,hereda,cambia) AS (VALUES " + ','.join(arcos) + "), "
            "reales AS (SELECT u.rolname::text login,g.rolname::text grupo,"
            "m.inherit_option hereda,m.set_option cambia,m.admin_option administra "
            "FROM pg_catalog.pg_auth_members m JOIN pg_catalog.pg_roles u ON u.oid=m.member "
            "JOIN pg_catalog.pg_roles g ON g.oid=m.roleid WHERE u.rolname IN (SELECT login FROM logins)) "
            "SELECT NOT EXISTS(SELECT 1 FROM logins l LEFT JOIN pg_catalog.pg_roles r ON r.rolname=l.login "
            "WHERE r.oid IS NULL OR NOT r.rolcanlogin OR NOT r.rolinherit OR r.rolsuper "
            "OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls "
            "OR r.rolconfig IS NOT NULL OR (CASE WHEN l.sin_caducidad THEN r.rolvaliduntil IS NOT NULL "
            "ELSE r.rolvaliduntil IS NULL OR r.rolvaliduntil<clock_timestamp()+interval '60 days' "
            "OR r.rolvaliduntil>clock_timestamp()+interval '90 days' END)) "
            "AND NOT EXISTS((SELECT login,grupo,hereda,cambia FROM esperados EXCEPT "
            "SELECT login,grupo,hereda,cambia FROM reales) UNION ALL "
            "(SELECT login,grupo,hereda,cambia FROM reales EXCEPT "
            "SELECT login,grupo,hereda,cambia FROM esperados)) "
            "AND NOT EXISTS(SELECT 1 FROM logins l LEFT JOIN reales r ON r.login=l.login "
            "GROUP BY l.login HAVING count(r.login)<>"
            "(SELECT count(*) FROM esperados e WHERE e.login=l.login)) "
            "AND NOT EXISTS(SELECT 1 FROM reales WHERE administra) "
            "AND NOT EXISTS(SELECT 1 FROM esperados e JOIN pg_catalog.pg_roles g ON g.rolname=e.grupo "
            "WHERE g.rolcanlogin OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole "
            "OR g.rolreplication OR g.rolbypassrls) "
            "AND NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1 c "
            "WHERE c.canal_permitido='interna_corporativa' AND "
            "((c.audiencia_consumo='vec_contratacion_temporal.vinculo_emision_bolsa.v1' "
            "AND c.operacion='contratacion_temporal.bolsa.vincular' AND c.login_nombre<> '" +
            urlsplit(entorno['VEC_CT_DATABASE_URL']).username + "') OR "
            "(c.audiencia_consumo='vec_bolsa_llamamientos.llamamiento.emitir.v1' "
            "AND c.operacion='llamamiento.emitir.v1' AND c.login_nombre<> '" +
            urlsplit(entorno['VEC_BOLSA_LLAMAMIENTOS_DATABASE_URL']).username + "'))) "
            "AND (SELECT count(*) FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1 "
            "WHERE canal_permitido='interna_corporativa' AND "
            "((audiencia_consumo='vec_contratacion_temporal.vinculo_emision_bolsa.v1' "
            "AND operacion='contratacion_temporal.bolsa.vincular') OR "
            "(audiencia_consumo='vec_bolsa_llamamientos.llamamiento.emitir.v1' "
            "AND operacion='llamamiento.emitir.v1')))=2;\n")


def comprobar(copia):
    sql = "SET statement_timeout='5s';\n" + consulta(copia)
    with (copia.raiz / 'comprobar-conexiones.log').open('ab') as errores:
        resultado = subprocess.run(['docker', 'exec', '-i', '-u', 'postgres', copia.nombre,
                    'psql', '-XAtq', '-v', 'ON_ERROR_STOP=1', '-U', 'postgres', '-d', 'postgres'],
                    input=sql.encode(), stdout=subprocess.PIPE, stderr=errores,
                    check=True, timeout=15)
    if resultado.stdout != b't\n':
        raise ValueError('Los roles finales no corresponden al contrato de la copia')
