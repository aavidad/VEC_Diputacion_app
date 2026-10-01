"""Lecturas CAS cerradas para el helper nominal; no abre conexiones ni escribe."""
from datetime import datetime, timedelta
from decimal import Decimal
import hashlib
import json
import re

class Refused(RuntimeError):
    """Solo códigos nominales, sin datos del proveedor."""


def require(ok, code):
    if not ok:
        raise Refused(code)


def keys(value, expected, code):
    require(type(value) is dict and set(value) == set(expected), code)


def fingerprint(value):
    require(type(value) is str and re.fullmatch(r'[a-f0-9]{64}', value), 'cas_hash_invalid')
    return value


def number(value, maximum=2**63, minimum=0):
    require(type(value) is int and minimum <= value < maximum, 'cas_number_invalid')
    return value


def instant(value):
    require(type(value) is str and len(value) <= 40, 'cas_time_invalid')
    try:
        result = datetime.fromisoformat(value.replace('Z', '+00:00'))
    except ValueError:
        raise Refused('cas_time_invalid') from None
    require(result.tzinfo is not None and result.utcoffset() == timedelta(0), 'cas_time_invalid')
    return result


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':'), allow_nan=False).encode()


def digest(value):
    return hashlib.sha256(value).hexdigest()

ROLE_ID = 'candidato_bolsa_portal_historial_propio_desarrollo'
ROLE_REF = 'rol:' + ROLE_ID + ':v1'
CATALOGS = {'motivos_mi_bolsa_desarrollo', 'motivos_historial_mi_bolsa_desarrollo',
            'motivos_portal_mi_bolsa_desarrollo'}
ACCOUNT_SQL = '''SELECT c.cuenta_ref,a.revision,e.estado
 FROM vec_identidad_externa_v1.cuenta c
 LEFT JOIN vec_identidad_externa_v1.estado_actual a USING(cuenta_ref)
 LEFT JOIN vec_identidad_externa_v1.estado_cuenta e
 ON e.cuenta_ref=a.cuenta_ref AND e.revision=a.revision WHERE c.cuenta_ref=%s'''
ALIAS_SQL = '''SELECT cuenta_ref,esquema_hmac,dominio_hmac_ref,clave_hmac_id,
 clave_hmac_version,encode(cuenta_id_hmac,'hex'),encode(sujeto_id_hmac,'hex')
 FROM vec_identidad_externa_v1.alias_cuenta WHERE cuenta_ref=%s ORDER BY alias_ref'''
CONTEXT_SQL = '''SELECT version,huella_sha256
 FROM vec_contexto_actor_v1.preimagen_snapshot_contexto_externo_v1(%s)'''
AUTH_SQL = 'SELECT vec_autorizacion.obtener_preimagen_candidato_externo_ro_v1(%s,%s,%s,%s)'
MOTIVES_SQL = 'SELECT vec_autorizacion.obtener_checkpoint_motivos_candidato_externo_ro_v1()'
CAS_FIELDS = {'revision_control_rol', 'huella_control_rol', 'version_asignacion',
              'huella_asignacion', 'version_contexto', 'huella_contexto', 'secuencia_motivos'}
CONTEXT_FIELDS = {'version_contexto', 'huella_contexto'}


def rows(cursor, sql, parameters=(), maximum=1):
    try:
        cursor.execute(sql, parameters)
        result = cursor.fetchmany(maximum + 1)
    except Exception:
        raise Refused('cas_query_not_accredited') from None
    require(type(result) in (tuple, list) and len(result) <= maximum, 'cas_query_ambiguous')
    require(all(type(row) in (tuple, list) for row in result), 'cas_row_invalid')
    return [tuple(row) for row in result]


def context(cursor, provision):
    result = rows(cursor, CONTEXT_SQL, (provision,))
    if not result:
        return {'version_contexto': 0, 'huella_contexto': ''}
    require(len(result[0]) == 2, 'cas_context_invalid')
    version, seal = result[0]
    if type(version) is Decimal:
        require(version.is_finite() and version == version.to_integral_value(), 'cas_context_invalid')
        version = int(version)
    return {'version_contexto': number(version, 2**64, 1), 'huella_contexto': fingerprint(seal)}


def identity(cursor, target, approved_alias):
    account = rows(cursor, ACCOUNT_SQL, (target['cuenta_ref'],))
    alias = rows(cursor, ALIAS_SQL, (target['cuenta_ref'],))
    require(bool(account) == bool(alias), 'cas_identity_partial')
    if not account:
        return {'account': None, 'alias': None}
    require(len(account[0]) == 3 and account[0][0] == target['cuenta_ref'] and
            account[0][2] == 'activa', 'cas_account_inconsistent')
    number(account[0][1], minimum=1)
    names = ('cuenta_ref', 'esquema', 'dominio_ref', 'clave_id', 'clave_version',
             'cuenta_id_hmac', 'sujeto_id_hmac')
    require(len(alias[0]) == len(names), 'cas_alias_inconsistent')
    value = dict(zip(names, alias[0]))
    require(value == approved_alias and type(value['clave_version']) is int,
            'cas_alias_changed')
    return {'account': account[0], 'alias': value}


def pair(state, value_name, hash_name, absent_fields, valid_state):
    require(state.get('estado_observacion') in ('ausente', valid_state), 'cas_state_rejected')
    if state['estado_observacion'] == 'ausente':
        require(all(state.get(name) is None for name in absent_fields), 'cas_absence_inconsistent')
        return 0, ''
    require(state.get('estado') == valid_state, 'cas_state_rejected')
    return number(state[value_name], minimum=1), fingerprint(state[hash_name])


def authorization(value, target):
    keys(value, ('contrato', 'observada_en', 'rol_id', 'version_rol_ref', 'principal_id',
                 'perfil_activo_ref', 'rol', 'control_rol', 'asignacion'), 'cas_auth_schema')
    require(value['contrato'] == 'preimagen_candidato_externo_ro_v1' and
            value['rol_id'] == ROLE_ID and value['version_rol_ref'] == ROLE_REF and
            value['principal_id'] == target['persona_ref'] and
            value['perfil_activo_ref'] == target['perfil_ref'], 'cas_auth_target')
    now = instant(value['observada_en'])
    role, control, assignment = value['rol'], value['control_rol'], value['asignacion']
    keys(role, ('estado_observacion', 'version', 'huella_sha256', 'estado', 'versiones', 'version_maxima'), 'cas_role_schema')
    keys(control, ('estado_observacion', 'revision', 'huella_sha256', 'estado', 'revisiones', 'revision_maxima'), 'cas_control_schema')
    keys(assignment, ('estado_observacion', 'versiones', 'version_maxima', 'asignacion_ref', 'version',
                     'huella_sha256', 'estado', 'vigente_desde', 'vigente_hasta'), 'cas_assignment_schema')
    role_version, _ = pair(role, 'version', 'huella_sha256', ('version', 'huella_sha256', 'estado'), 'publicada')
    revision, seal = pair(control, 'revision', 'huella_sha256', ('revision', 'huella_sha256', 'estado'), 'habilitada')
    version, assignment_seal = pair(assignment, 'version', 'huella_sha256',
        ('version', 'huella_sha256', 'estado', 'asignacion_ref', 'vigente_desde', 'vigente_hasta'), 'activa')
    require((role_version == 0) == (revision == 0) and (not version or role_version), 'cas_auth_partial')
    for state, n, maximum, actual in ((role, 'versiones', 'version_maxima', role_version),
             (control, 'revisiones', 'revision_maxima', revision),
             (assignment, 'versiones', 'version_maxima', version)):
        count = number(state[n])
        require((actual == 0 and count == 0 and state[maximum] is None) or
                (actual > 0 and count > 0 and state[maximum] == actual and type(state[maximum]) is int),
                'cas_auth_history_inconsistent')
    require(not role_version or role_version == 1, 'cas_role_version_changed')
    if version:
        require(type(assignment['asignacion_ref']) is str and 0 < len(assignment['asignacion_ref']) <= 512 and
                instant(assignment['vigente_desde']) <= now < instant(assignment['vigente_hasta']),
                'cas_assignment_not_current')
    return {'revision_control_rol': revision, 'huella_control_rol': seal,
            'version_asignacion': version, 'huella_asignacion': assignment_seal}


def motives(value):
    keys(value, ('contrato', 'observada_en', 'checkpoint', 'catalogos'), 'cas_motives_schema')
    require(value['contrato'] == 'checkpoint_motivos_candidato_externo_ro_v1', 'cas_motives_contract')
    now = instant(value['observada_en'])
    cp = value['checkpoint']
    keys(cp, ('controles', 'ultima_secuencia', 'ultimo_evento_ref', 'ultima_huella_evento_sha256',
              'actualizado_en', 'eventos', 'secuencia_maxima', 'coherente'), 'cas_checkpoint_schema')
    sequence = number(cp['ultima_secuencia'], 2**62)
    require(type(cp['controles']) is int and cp['controles'] == 1 and cp['coherente'] is True and
            type(cp['eventos']) is int and type(cp['secuencia_maxima']) is int and
            cp['eventos'] == sequence == cp['secuencia_maxima'], 'cas_checkpoint_inconsistent')
    require(instant(cp['actualizado_en']) <= now, 'cas_checkpoint_time')
    if sequence:
        fingerprint(cp['ultima_huella_evento_sha256'])
        require(type(cp['ultimo_evento_ref']) is str and 0 < len(cp['ultimo_evento_ref']) <= 512, 'cas_checkpoint_event')
    else:
        require(cp['ultimo_evento_ref'] is None and cp['ultima_huella_evento_sha256'] is None, 'cas_checkpoint_event')
    cats = value['catalogos']
    require(type(cats) is list and len(cats) == 3 and
            {c.get('catalogo_id') for c in cats if type(c) is dict} == CATALOGS, 'cas_catalog_set')
    for cat in cats:
        keys(cat, ('catalogo_id', 'versiones', 'observaciones'), 'cas_catalog_schema')
        observations = cat['observaciones']
        require(type(observations) is list and number(cat['versiones'], 4097) == len(observations), 'cas_catalog_count')
        seen = set()
        for obs in observations:
            keys(obs, ('version', 'huella_publicada_sha256', 'publicado_en', 'evento_publicacion_ref',
              'secuencia_publicacion', 'huella_evento_publicacion_sha256', 'entradas', 'entradas_vigentes',
              'retirada', 'huella_retirada_sha256', 'retirado_en', 'evento_retirada_ref',
              'secuencia_retirada', 'huella_evento_retirada_sha256', 'coherente'), 'cas_catalog_observation_schema')
            ver = number(obs['version'], minimum=1)
            require(ver not in seen and obs['coherente'] is True and obs['retirada'] is False, 'cas_catalog_rejected')
            seen.add(ver)
            fingerprint(obs['huella_publicada_sha256']); fingerprint(obs['huella_evento_publicacion_sha256'])
            require(instant(obs['publicado_en']) <= now and
                    0 < number(obs['secuencia_publicacion'], 2**62) <= sequence and
                    type(obs['evento_publicacion_ref']) is str and 0 < len(obs['evento_publicacion_ref']) <= 512 and
                    number(obs['entradas'], 65537, 1) == number(obs['entradas_vigentes'], 65537, 1), 'cas_catalog_not_current')
            require(all(obs[k] is None for k in ('huella_retirada_sha256', 'retirado_en',
                      'evento_retirada_ref', 'secuencia_retirada', 'huella_evento_retirada_sha256')), 'cas_catalog_partial')
    return sequence


def scan(cursors, targets, aliases):
    result = {'identidad': {}, 'contexto': {}}
    for population in ('candidato', 'usuarios'):
        target = targets[population]
        result['identidad'][population] = identity(cursors['identidad'], target, aliases[target['cuenta_ref']])
        result['contexto'][population] = context(cursors['contexto'], target['provision_ref'])
    auth = rows(cursors['autorizacion'], AUTH_SQL,
                (ROLE_ID, ROLE_REF, targets['candidato']['persona_ref'], targets['candidato']['perfil_ref']))
    mot = rows(cursors['autorizacion'], MOTIVES_SQL)
    require(len(auth) == len(mot) == 1 and len(auth[0]) == len(mot[0]) == 1, 'cas_auth_missing')
    result['autorizacion'], result['motivos'] = auth[0][0], mot[0][0]
    candidate = authorization(result['autorizacion'], targets['candidato'])
    sequence = motives(result['motivos'])
    result['preimages'] = {'candidato': {**candidate, **result['contexto']['candidato'], 'secuencia_motivos': sequence},
                          'usuarios': result['contexto']['usuarios']}
    return result


def semantic(reading):
    # Quitar únicamente los instantes de observación de las dos funciones AUT26.
    result = dict(reading)
    for channel in ('autorizacion', 'motivos'):
        result[channel] = dict(result[channel])
        del result[channel]['observada_en']
    return result
