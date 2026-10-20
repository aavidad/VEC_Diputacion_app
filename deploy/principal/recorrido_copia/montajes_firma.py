"""Montajes ADMIN del lote, plan, cargos y certificados de una copia F1-131.

Se invoca después de firmantes.preparar y antes de los catorce actos.
El código lee únicamente el kit y los recibos propios; jamás fuentes privadas H.
"""

import hashlib
import json
import os
from pathlib import Path
import re
import secrets
from datetime import timedelta

from admin_copia.catalogo_acciones import q, scram
from admin_copia.ejecutor import privado
from admin_copia.preparar_runtime import validar_gobierno
from recorrido_copia.actos import perfiles_admin


AUDIENCIAS = {
    'lote': 'vec_autorizacion.administracion_perfiles.lote_ordinario.v1',
    'plan_firma': 'vec_catalogos_configurables.plan_nominal_firma.gobierno.v1',
    'cargos': 'vec_personal.cargo_competencial.publicar.v1',
    'certificados': 'vec_contexto_actor.certificado_nominal.publicar.v1',
}
GRUPOS = {
    'lote': 'vec_admin_perfiles_lote_ejecutor',
    'plan_firma': 'vec_plan_firma_gobierno_ejecutor',
    'cargos': 'vec_personal_ejecutor',
    'certificados': 'vec_autorizacion_certificado_nominal_ejecutor',
}
OPERACIONES = {
    'lote': ('administracion.perfiles.aplicar_lote_ordinario',),
    'plan_firma': tuple('vec.catalogos.' + x for x in ('crear', 'actualizar', 'publicar', 'retirar')),
    'cargos': ('personal.cargo_competencial.publicar',),
    'certificados': ('administracion.certificados.nominal.publicar',
                     'administracion.certificados.nominal.retirar'),
}
ENTORNO = {
    'VEC_ADMIN_USUARIOS_CONFIG_FILE': 'usuarios-firma.json',
    'VEC_ADMIN_LOTE_CONFIG_FILE': 'lote.json',
    'VEC_ADMIN_PLAN_FIRMA_CONFIG_FILE': 'plan-firma.json',
    'VEC_ADMIN_CARGOS_CONFIG_FILE': 'cargos.json',
    'VEC_ADMIN_CERTIFICADOS_CONFIG_FILE': 'certificados.json',
}


def exigir(ok, motivo):
    if not ok:
        raise ValueError(motivo)


def documento(c, ruta):
    return c.leer(privado(ruta))


def sha(ruta):
    with Path(ruta).open('rb') as entrada:
        return hashlib.file_digest(entrada, 'sha256').hexdigest()


def preflight(c):
    estado = c.copia.leer()
    sid = estado.get('sid_copia')
    exigir(estado.get('admin_preparada') is True and estado.get('recorrido_completo') is True
           and isinstance(sid, str) and re.fullmatch(r'[0-9]{15,20}', sid)
           and c.consultar('SELECT system_identifier::text FROM pg_catalog.pg_control_system();') == sid,
           'ADMIN, recorrido o SID de copia no confirmados')
    run = c.raiz / 'admin/h10'
    entrada = documento(c, run / 'entrada-vec-admin.json')
    dia = entrada.get('dia_gobierno')
    exigir(re.fullmatch(r'[0-9]{8}[a-z0-9]{1,12}', str(dia))
           and entrada.get('runner') == str(run) and entrada.get('sid_copia') == sid
           and entrada.get('pg_host') == 'localhost'
           and entrada.get('pg_base') == 'postgres'
           and entrada.get('pg_port') == estado['puerto_pg']
           and entrada.get('pg_ca') == str(c.material / 'ca/ca.crt'),
           'Entrada de gobierno ajena a la copia')
    privado(c.material / 'ca/ca.crt')
    acuses = {}
    for nombre in ('bootstrap', 'unidad'):
        ruta = run / nombre / 'acuse-primero.json'
        acuse = documento(c, ruta)
        exigir(acuse.get('estado') == 'permitido'
               and acuse.get('recibo', {}).get('auditoria_ref'),
               'Acuse H10 ausente: ' + nombre)
        acuses[nombre] = (acuse, sha(ruta))
    gobierno = run / 'gobierno' / dia
    acuse_g = documento(c, gobierno / 'acuse-aplicar.json')
    _, material, configuracion = validar_gobierno(run, dia, aplicado=True)
    exigir(acuse_g.get('estado') == 'permitido'
           and configuracion.get('ConjuntoVersion') == 7
           and set(AUDIENCIAS.values()).issubset(
               {x.get('audiencia') for x in material.get('claves', [])}),
           'Gobierno 7 sin las cuatro capacidades nominales')
    acuses['gobierno'] = (acuse_g, sha(gobierno / 'acuse-aplicar.json'))
    firmantes = documento(c, c.carpeta / 'firmantes/metadata.json')
    exigir(firmantes.get('copia_ref') == c.copia.nombre
           and firmantes.get('roles_iniciales') == 16
           and firmantes.get('sucesoras_consulta') == 4,
           'Semilla de firmantes incompleta o de otra copia')
    perfiles = perfiles_admin(c)
    exigir(len(perfiles) == 2 and perfiles[0] != perfiles[1],
           'Se exigen dos ADMIN nominales distintos')
    base = documento(c, run / 'vec-admin/config/base.json')
    usuarios = documento(c, run / 'vec-admin/config' / ('usuarios-' + dia + '.json'))
    motivo = entrada.get('motivo')
    exigir(isinstance(motivo, dict) and set(motivo) == {
        'catalogo_id', 'catalogo_version', 'catalogo_huella_sha256', 'entrada_clave'}
        and motivo['catalogo_id'] == base.get('catalogo_motivos_id')
        and re.fullmatch(r'[0-9a-f]{64}', motivo['catalogo_huella_sha256'])
        and usuarios.get('organizacion_ref') == acuses['unidad'][0]['recibo']['unidad']['organismo_ref']
        and usuarios.get('unidad_ref') == acuses['unidad'][0]['recibo']['unidad']['unidad_ref'],
        'Motivo, usuarios o unidad H10 divergentes')
    return estado, run, dia, entrada, material, configuracion, usuarios, motivo, acuses, perfiles


def preparar_sql(c, estado, run, dia, entrada):
    """Cuatro LOGIN segregados y AD172, ligados al SID antes del primer efecto."""
    carpeta = c.carpeta / 'montajes_firma'
    carpeta.mkdir(mode=0o700, exist_ok=True)
    salida = carpeta / 'pools.sql'
    exigir(not salida.exists(), 'Pools de firma ya preparados; conservar resultados')
    puerto = estado['puerto_pg']
    ca = c.material / 'ca/ca.crt'
    hasta = (c.ahora() + timedelta(days=89)).strftime('%Y-%m-%dT%H:%M:%SZ')
    sid = estado['sid_copia']
    sql = ['BEGIN;', "DO $sid$ BEGIN IF current_database() IS DISTINCT FROM 'postgres' "
           'OR (SELECT system_identifier::text FROM pg_catalog.pg_control_system()) '
           f"IS DISTINCT FROM {q(sid)} THEN RAISE EXCEPTION 'montaje ADMIN en otra copia'; "
           'END IF; END $sid$;', "SET LOCAL password_encryption='scram-sha-256';"]
    logins = {}
    for nombre, audiencia in AUDIENCIAS.items():
        login = 'vec_adm_' + nombre + '_' + dia
        exigir(re.fullmatch(r'vec_[a-z0-9_]{3,62}', login), 'LOGIN de firma inválido')
        grupo = GRUPOS[nombre]
        clave = secrets.token_urlsafe(32)
        c.guardar(run / 'vec-admin/pools' / (nombre + '.json'), {
            'dsn': f'host=localhost port={puerto} dbname=postgres user={login} '
                   f'password={clave} sslmode=verify-full sslrootcert={ca}'})
        sql.extend((f'DO $g$ BEGIN IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname={q(login)}) '
                    f'OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname={q(grupo)}) '
                    "THEN RAISE EXCEPTION 'LOGIN de montaje ocupado o grupo ausente'; END IF; END $g$;",
                    f'CREATE ROLE {login} LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE '
                    f'NOREPLICATION NOBYPASSRLS VALID UNTIL {q(hasta)} PASSWORD {q(scram(clave))};',
                    f'GRANT {grupo} TO {login} WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;',
                    f'GRANT CONNECT ON DATABASE postgres TO {grupo};'))
        logins[nombre] = login
    sql.append('SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;')
    for nombre, audiencia in AUDIENCIAS.items():
        proceso = entrada['procesos']['usuarios'] if nombre == 'lote' else 'vec-admin'
        exigir(re.fullmatch(r'[a-z][a-z0-9._-]{1,79}', proceso), 'Proceso ADMIN inválido')
        for operacion in OPERACIONES[nombre]:
            sql.append('INSERT INTO vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1'
                       '(login_nombre,audiencia_consumo,operacion,proceso,canal_permitido) '
                       f'VALUES({q(logins[nombre])},{q(audiencia)},{q(operacion)},'
                       f"{q(proceso)},'administracion_privilegiada');")
    sql.extend(('RESET ROLE;', 'COMMIT;'))
    c.guardar(salida, ('\n'.join(sql) + '\n').encode())
    c.sql(privado(salida).read_bytes(), 'montajes-firma-pools.log')
    return carpeta, logins


def preparar_configuraciones(c, run, dia, entrada, material, cm, usuarios, motivo, acuses):
    """Metadatos por audiencia desde las claves realmente publicadas de Gov7."""
    conf = run / 'vec-admin/config'
    fuentes = {x['audiencia']: x for x in material['claves']}
    base = usuarios['confianza']
    raiz, gob = cm['Raiz'], cm['Gobierno']
    exigir(base['raiz']['clave_id'] == raiz['ClaveID']
           and base['raiz']['publica_base64'] == raiz['Publica']
           and base['gobierno']['revision'] == gob['Revision'],
           'Confianza de usuarios y Gov7 divergente')
    cabecera = {k: base[k] for k in ('cabecera', 'raiz', 'gobierno')}
    descriptores = []
    for nombre, audiencia in AUDIENCIAS.items():
        fuente = fuentes.get(audiencia)
        exigir(fuente is not None, 'Capacidad Gov7 ausente: ' + nombre)
        clave = run / 'vec-admin/claves' / dia / (audiencia + '.bin')
        exigir(sha(privado(clave)) == fuente['huella_secreto_sha256'],
               'Clave Gov7 de otra audiencia')
        ent = {'audiencia': audiencia, 'clave_id': fuente['clave_id'],
               'emisor_id': fuente['emisor_id'],
               'huella_gobierno': fuente['huella_gobierno_sha256'],
               'version': fuente['version'],
               'revision_gobierno': fuente['revision_gobierno'],
               'material_archivo': str(clave), 'estado': 'emision',
               'valida_desde': fuente['valida_desde'],
               'valida_hasta': fuente['valida_hasta'],
               'revocada_en': '0001-01-01T00:00:00Z'}
        descriptores.append(ent)
    exigir(len({x['material_archivo'] for x in descriptores}) == 4,
           'Capacidades comparten material')
    configs = {}
    for nombre, ent in zip(AUDIENCIAS, descriptores):
        configs[nombre] = {'pool': str(run / 'vec-admin/pools' / (nombre + '.json')),
                           'confianza': {**cabecera, 'entradas_capacidad': [ent]},
                           'motivo': motivo}
    persona = documento(c, run / 'bootstrap/plan-v3.json')['plan']['personas'][0]
    ambitos = persona['ambitos']
    exigir(len(ambitos) == 2
           and [x['dimension'] for x in ambitos] == ['organizacion_ref', 'unidad_ref'],
           'Ámbitos H10 inesperados')
    unidad = {'unidad_ref': ambitos[1]['valores'][0],
              'fuente_organizacion': ambitos[0]['fuente'],
              'fuente_unidad': ambitos[1]['fuente']}
    configs['lote'] = {'modo': 'lote_v1', 'pool_lote': configs['lote']['pool'],
                       'confianza': configs['lote']['confianza'],
                       'motivo_lote': motivo, 'unidades': [unidad],
                       'motivos_cambio': [{'motivo': motivo, 'clave_i18n': 'alta_funciones'}]}
    usuarios_firma = dict(usuarios)
    destinos = dict(usuarios.get('destinos', {}))
    esperado = {'accion': 'administracion.perfiles.frontera.preparar_lote_ordinario',
                'recurso_ref': 'administracion:perfiles:preparar_lote_ordinario',
                'finalidad_ref': 'gestion_perfiles', 'tipo_recurso': 'fijo'}
    exigir(destinos.get('preparar_lote_ordinario') in (None, esperado),
           'Destino de lote anterior divergente')
    destinos['preparar_lote_ordinario'] = esperado
    usuarios_firma['destinos'] = destinos
    c.guardar(conf / ENTORNO['VEC_ADMIN_USUARIOS_CONFIG_FILE'], usuarios_firma)
    for variable, nombre in ENTORNO.items():
        if variable != 'VEC_ADMIN_USUARIOS_CONFIG_FILE':
            clave = {'VEC_ADMIN_LOTE_CONFIG_FILE': 'lote',
                     'VEC_ADMIN_PLAN_FIRMA_CONFIG_FILE': 'plan_firma',
                     'VEC_ADMIN_CARGOS_CONFIG_FILE': 'cargos',
                     'VEC_ADMIN_CERTIFICADOS_CONFIG_FILE': 'certificados'}[variable]
            c.guardar(conf / nombre, configs[clave])
    resultado = {'version': 1, 'copia_ref': c.copia.nombre,
                 'sid_copia': c.copia.leer()['sid_copia'], 'dia_gobierno': dia,
                 'fuentes_sha256': {k: x[1] for k, x in acuses.items()},
                 'archivos_sha256': {k: sha(conf / v) for k, v in ENTORNO.items()}}
    return resultado


def activar(c, resultado):
    """Reinicio vigilado: si ADMIN rechaza el overlay, restaura el anterior."""
    from copia_local import guardar
    from copia_local_administracion import ejecutar as ejecutar_admin, parar as parar_admin

    preparacion_ruta = c.raiz / 'admin_preparacion.json'
    anterior = documento(c, preparacion_ruta)
    exigir('admin_firma_env_sha256' not in anterior,
           'Montaje de firma ya iniciado; conciliar antes de repetir')
    config = c.raiz / 'admin/h10/vec-admin/config'
    env_ruta = c.raiz / 'admin_firma_env.json'
    c.guardar(env_ruta, {k: str(config / v) for k, v in ENTORNO.items()})
    parar_admin(c.copia)
    guardar(preparacion_ruta, {**anterior, 'admin_firma_env_sha256': sha(env_ruta),
                              'admin_firma_archivos_sha256': resultado['archivos_sha256']})
    try:
        ejecutar_admin(c.copia, 'arrancar')
        ejecutar_admin(c.copia, 'comprobar')
    except Exception as error:
        parar_admin(c.copia)
        env_ruta.rename(c.carpeta / 'montajes_firma/admin-firma-env-fallido.json')
        guardar(preparacion_ruta, anterior)
        ejecutar_admin(c.copia, 'arrancar')
        ejecutar_admin(c.copia, 'comprobar')
        raise ValueError('Montaje de firma rechazado; Administración recuperada') from error


def preparar(c):
    datos = preflight(c)
    estado, run, dia, entrada, material, cm, usuarios, motivo, acuses, perfiles = datos
    carpeta, logins = preparar_sql(c, estado, run, dia, entrada)
    resultado = preparar_configuraciones(c, run, dia, entrada, material, cm,
                                         usuarios, motivo, acuses)
    resultado['perfiles_admin'] = perfiles
    resultado['logins'] = logins
    c.guardar(carpeta / 'metadata.json', resultado)
    activar(c, resultado)
    return resultado
