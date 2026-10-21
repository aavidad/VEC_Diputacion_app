"""Prepara y publica el plan nominal de cinco pasos con dos ADMIN propios."""

import base64
import json
import hashlib
import re
import secrets
import uuid
from pathlib import Path

from .actos import perfiles_admin, _recibo as recibo_acto
from .materiales_actos import PASOS, canon, json_propio, verificar_materiales

RUTA = '/api/admin/perfiles/v1/plan-firma/gobierno'
CATALOGO = 'ct.plan.firma.hz39.operativa'
HEX = re.compile(r'[0-9a-f]{64}\Z')


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


def _correspondencia(c, org):
    operativa = c.parametros['organizacion_operativa_ref']
    if not re.fullmatch(r'[a-z][a-z0-9:._-]{2,127}', operativa) or not re.fullmatch(r'org_[0-9a-f]{32}', org):
        raise ValueError('Organización de firma no cotejable')
    consulta = ("SELECT h.organizacion_canonica_ref||'|'||h.correspondencia_ref||'|'||h.version "
                "FROM vec_contexto_actor_v1.correspondencia_organizacion_actual_v1 a "
                "JOIN vec_contexto_actor_v1.correspondencia_organizacion_versiones_v1 h "
                "USING(organizacion_operativa_ref,version) WHERE a.organizacion_operativa_ref='" + operativa + "';")
    salida = c.consultar(consulta).strip()
    if not salida:
        plantilla = c.fuente / 'scripts/operacion/firma_ct_tres_pasos/publicar_correspondencia.sql'
        script = ("\\set organizacion_operativa_ref '" + operativa + "'\n"
                  "\\set organizacion_canonica_ref '" + org + "'\n").encode() + plantilla.read_bytes()
        c.sql(script, 'plan-correspondencia-ca45.log')
        salida = c.consultar(consulta).strip()
    filas = salida.splitlines()
    if len(filas) != 1:
        raise ValueError('Correspondencia CA45 ausente o ambigua')
    canonica, fuente, version = filas[0].split('|')
    if canonica != org or not fuente or not version.isdecimal() or int(version) < 1:
        raise ValueError('Correspondencia CA45 ajena al órgano de los actos')
    return fuente, version


def _material(c, raiz, op):
    archivo = raiz / (op + '-material.json')
    huella = (raiz / (op + '-material.sha256')).read_text().strip()
    if not HEX.fullmatch(huella) or sha(archivo.read_bytes()) != huella:
        raise ValueError('Material canónico del plan modificado')
    resultado = c.ejecutar([str(c.cli('vec-plan-firma-validar')), str(archivo), huella],
                           f'plan-validar-{op}.log')
    if resultado.returncode or json.loads(resultado.stdout)['estado'] != 'material_validado_sin_autorizacion':
        raise ValueError('El validador Go rechazó el material del plan')
    return json.loads(archivo.read_bytes()), huella


def _cotejar_par(crear, publicar, meta):
    a = json.loads(base64.b64decode(crear['catalogo_canonico_base64'], validate=True))
    b = json.loads(base64.b64decode(publicar['catalogo_canonico_base64'], validate=True))
    claves = [p[0] for p in PASOS]
    if (crear.get('operacion') != 'crear' or publicar.get('operacion') != 'publicar'
            or a.get('id') != CATALOGO or b.get('id') != CATALOGO
            or a.get('version') != 1 or b.get('version') != 1
            or a.get('creado_por') != meta['admin_personas'][0]
            or b.get('publicado_por') != meta['admin_personas'][1]
            or a.get('entradas') != b.get('entradas')
            or [e['clave'] for e in b['entradas']] != claves
            or publicar.get('huella_esperada') != crear.get('catalogo_sha256')
            or any(e['atributos'].get('circuito_ref') != 'vec.contratacion_temporal.circuito_firma:2'
                   for e in b['entradas'])):
        raise ValueError('Los materiales crear/publicar del plan no corresponden')
    return b


def _recibo(respuesta, op):
    if not isinstance(respuesta, dict) or respuesta.get('error'):
        raise ValueError('Respuesta del gobierno del plan inválida')
    r = respuesta.get('recibo')
    estado = 'borrador' if op == 'crear' else 'publicado'
    if (not isinstance(r, dict) or r.get('catalogo_ref') != CATALOGO + ':1'
            or r.get('estado') != estado or not r.get('recibo_ref')
            or not r.get('auditoria_ref') or not r.get('consumo_auditoria_ref')):
        raise ValueError('Recibo del plan ausente o ajeno')
    return r


def preparar_plan(c):
    actos = json_propio(c, c.carpeta / 'actos/metadata.json')
    m = json_propio(c, c.carpeta / 'actos/acciones.json')
    verificar_materiales(c, m)
    if actos.get('actos_confirmados') != 14 or len(actos.get('recibos', [])) != 14 or actos.get('gobierno_tecnico_roles') is not False:
        raise ValueError('Faltan recibos de los catorce actos; AUT53 no es gobierno técnico')
    for i, a in enumerate(m['acciones']):
        r = json_propio(c, c.carpeta / 'actos/recibos' / (a['id'] + '.json'))
        if recibo_acto(a, r)['recibo_ref'] != actos['recibos'][i]:
            raise ValueError('Recibo ADMIN del acto ajeno al expediente')
    if actos['admin_personas'][0] == actos['admin_personas'][1]:
        raise ValueError('Un solo ADMIN para el plan')
    perfiles = perfiles_admin(c)
    if perfiles != actos['admin_perfiles']:
        raise ValueError('Perfil ADMIN distinto del que publicó los actos')
    org = m['pasos'][0]['organizacion_canonica_ref']
    fuente_mapeo, version_mapeo = _correspondencia(c, org)
    carpeta = c.carpeta / 'plan'
    circuito_origen = c.fuente / 'data/catalogos/contratacion-temporal/circuito_firma.v2.json'
    circuito = carpeta / 'circuito_firma.v2.json'
    raw_circuito = circuito_origen.read_bytes()
    c.guardar(circuito, raw_circuito)
    paquete_circuito = c.leer(circuito)
    cc = paquete_circuito.get('catalogo', {})
    if (paquete_circuito.get('fuente', {}).get('demostracion') is not False
            or cc.get('id') != 'vec.contratacion_temporal.circuito_firma'
            or cc.get('version') != 2 or cc.get('estado') != 'publicado'
            or not cc.get('publicado_en') or not cc.get('aprobacion_ref')):
        raise ValueError('Circuito v2 sin publicación acreditable')
    base = json.loads((Path(__file__).parent / 'datos/plan_entradas.json').read_bytes())
    if len(base) != 5 or len(m['pasos']) != 5:
        raise ValueError('Cinco pasos del plan ausentes')
    entradas = []
    for t, paso in zip(base, m['pasos']):
        if (t['clave'] != paso['clave'] or paso['organizacion_canonica_ref'] != org
                or paso['organizacion_operativa_ref'] != c.parametros['organizacion_operativa_ref']):
            raise ValueError('Paso del plan ajeno a los actos')
        slug = paso['cargo'] + '-' + paso['documento']
        cargo = c.leer(c.carpeta / 'actos/materiales' / (slug + '-cargo.json'))
        if cargo.get('objeto_ref') != paso['cargo_ref']:
            raise ValueError('Cargo material divergente del paso')
        e = json.loads(json.dumps(t))
        e['vigente_desde'] = cargo['datos']['vigente_desde']
        e['vigente_hasta'] = cargo['datos']['vigente_hasta']
        e['atributos'].update({'cargo_ref': paso['cargo_ref'], 'rol_id': paso['rol_id'],
                               'perfil_esperado_ref': paso['perfil_esperado_ref'],
                               'mapeo_fuente_ref': fuente_mapeo, 'mapeo_version': version_mapeo,
                               'organizacion_ref': paso['organizacion_operativa_ref'],
                               'unidad_ref': paso['unidad_ref']})
        for clave in ('circuito_ref', 'circuito_version', 'circuito_sha256', 'paso_ref'):
            e['atributos'].pop(clave, None)
        entradas.append(e)
    entradas_archivo = carpeta / 'entradas.json'
    c.guardar(entradas_archivo, entradas)
    config_archivo = carpeta / 'config.json'
    if config_archivo.exists():
        cfg = c.leer(config_archivo)
    else:
        token = secrets.token_hex(12)
        referencia = str(uuid.uuid4())
        copia_ref = m['copia_ref']
        cfg = {'version': 1, 'copia_ref': copia_ref, 'circuito_archivo': str(circuito),
               'circuito_archivo_sha256': sha(raw_circuito), 'entradas_archivo': str(entradas_archivo),
               'entradas_archivo_sha256': sha(entradas_archivo.read_bytes()),
               'fuente_ref': 'fuente:plan:' + referencia,
               'creado_por': actos['admin_personas'][0], 'creado_perfil_ref': perfiles[0],
               'publicado_por': actos['admin_personas'][1], 'publicado_perfil_ref': perfiles[1],
               'aprobacion_ref': 'aprobacion:plan:' + referencia,
               'preparacion_ref': 'preparacion:plan:' + referencia,
               'clave_operacion_crear': 'plan-firma-hz39-v1-' + copia_ref + '-crear-' + token,
               'clave_operacion_publicar': 'plan-firma-hz39-v1-' + copia_ref + '-publicar-' + token,
               'correlacion_crear': 'correlacion:plan-firma-hz39-v1:' + token + ':crear',
               'correlacion_publicar': 'correlacion:plan-firma-hz39-v1:' + token + ':publicar'}
        c.guardar(config_archivo, cfg)
    if (cfg['entradas_archivo_sha256'] != sha(entradas_archivo.read_bytes())
            or cfg['circuito_archivo_sha256'] != sha(raw_circuito)
            or cfg['creado_por'] != actos['admin_personas'][0]
            or cfg['publicado_por'] != actos['admin_personas'][1]
            or cfg['creado_perfil_ref'] != perfiles[0] or cfg['publicado_perfil_ref'] != perfiles[1]):
        raise ValueError('Configuración anterior del plan no coincide con los actos')
    material = carpeta / 'material'
    if not material.exists():
        r = c.ejecutar([str(c.cli('vec-plan-firma-validar')), '--config', str(config_archivo),
                       '--salida', str(material)], 'plan-preparador-go.log')
        if r.returncode or json.loads(r.stdout)['estado'] != 'preparado_sin_autorizacion':
            raise ValueError('El preparador Go rechazó el plan')
    crear, _ = _material(c, material, 'crear')
    publicar, _ = _material(c, material, 'publicar')
    catalogo = _cotejar_par(crear, publicar, actos)
    recibos = []
    for numero, op, mat in ((1, 'crear', crear), (2, 'publicar', publicar)):
        confirmado = carpeta / 'recibos' / (op + '.json')
        incierto = carpeta / 'recibos' / (op + '.sin-confirmar.json')
        if confirmado.exists():
            r = _recibo(c.leer(confirmado), op)
        else:
            if incierto.exists():
                raise ValueError('Plan ' + op + ' incierto: conciliar la clave original')
            c.guardar(incierto, {'operacion': op, 'material_sha256': sha((material / (op + '-material.json')).read_bytes()),
                                 'estado': 'pendiente_de_conciliacion'})
            peticion = {'material_base64': base64.b64encode((material / (op + '-material.json')).read_bytes()).decode()}
            try:
                estado, respuesta = c.admin(numero, 'POST', RUTA, peticion)
            except Exception as error:
                raise ValueError('Plan ' + op + ' incierto: consultar la operación original') from error
            if estado != 200:
                raise ValueError('Plan ' + op + ' sin confirmación HTTP; conciliar la operación original')
            r = _recibo(respuesta, op)
            c.guardar(confirmado, respuesta)
        recibos.append(r['recibo_ref'])
    paquete = {'version_esquema': 1,
               'fuente': {'revision': 'plan-firma-hz39-f131-v1', 'actualizada_en': catalogo['publicado_en'],
                          'demostracion': False, 'aviso': 'Copia local sintética; publicación acreditada por recibos ADMIN.',
                          'origen_sha256': sha((material / 'publicar-catalogo.json').read_bytes())},
               'catalogo': catalogo}
    destino = c.material / 'identidad/firma-recursos/plan-firma-publicado.json'
    c.guardar(destino, paquete)
    meta = {'version': 1, 'copia_ref': m['copia_ref'], 'catalogo_ref': CATALOGO + ':1',
            'catalogo_sha256': sha((material / 'publicar-catalogo.json').read_bytes()),
            'paquete_sha256': sha(destino.read_bytes()), 'recibos': recibos,
            'admin_personas': actos['admin_personas'], 'admin_perfiles': perfiles,
            'gobierno_tecnico_roles': False, 'plan_publicado': True}
    c.guardar(carpeta / 'metadata.json', meta)
    return meta
