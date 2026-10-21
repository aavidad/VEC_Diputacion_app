"""Actos de identidad documental CT en una copia ficticia ya preparada.

Solo se invoca después de catálogo, gobierno de roles y fuentes de Gestión.
Conserva las marcas de resultado incierto y usa siempre el acto original.
"""

import hashlib
import json
from pathlib import Path
import re
import secrets
import ssl
from urllib.parse import quote, urlsplit

from admin_copia import catalogo_acciones as canon
from . import catalogo, roles_admin


FUENTE = 'bolsa_identidad_documental_ct_rrhh_v1.json'
FUENTE_SHA256 = 'e427c10a08387328c52e09316bb5bad12049052652c831aa7c6ef94f4f07e131'
ACCION = 'bolsa.aceptacion_ct.identidad_documento.consultar'
ROL = 'identidad_documental_ct_gestion_dev'
ROL_REF = 'rol:' + ROL + ':v1'
RUTA_LOTE = '/api/admin/perfiles/v1/lotes-ordinarios'
HEX = re.compile(r'[0-9a-f]{64}\Z')
OPACA = re.compile(r'[A-Za-z0-9][A-Za-z0-9._:/-]{2,191}\Z')
DSN = {'bolsa': 'VEC_BOLSA_LLAMAMIENTOS_DATABASE_URL',
       'fuente_autorizacion': 'VEC_AUTORIZACION_FUENTE_DATABASE_URL',
       'motivos_autorizacion': 'VEC_AUTORIZACION_MOTIVOS_EVALUADOR_DATABASE_URL'}


def exigir(ok, mensaje):
    if not ok:
        raise ValueError(mensaje)


def _fuente(c):
    ruta = c.fuente / 'deploy/principal/fuentes' / FUENTE
    exigir(ruta.is_file() and not ruta.is_symlink() and c.sha(ruta) == FUENTE_SHA256,
           'Fuente documental de Git distinta de la revisada')
    f = json.loads(ruta.read_bytes())
    exigir(f.get('modulo_id') == 'bolsa' and f.get('version') == 1
           and len(f.get('entradas', [])) == 1
           and HEX.fullmatch(f.get('huella_sha256', ''))
           and hashlib.sha256(f['entradas_canon'].encode()).hexdigest() == f['huella_sha256']
           and f['entradas'][0]['concesion']['accion'] == ACCION,
           'Fuente documental incompatible')
    # La validación canónica es la misma que usa el preparador AUT58.
    canon.fuente_nueva(ruta.parent, ruta.name, FUENTE_SHA256)
    return f


def _admitir_fuente(c, f):
    anteriores = roles_admin._admisiones(c)
    exigir(len(anteriores['actos']) == 2 and anteriores['actos'][-1]['nombre'] == 'origen_ct',
           'Gobierno AUT58 previo no confirmado')
    _, sid, backup, fuentes = catalogo.copia_y_fuentes(c)
    m = catalogo.acto(c, c.cli('vec-catalogo-acciones-admitir'),
                      (13, 'fuentes', FUENTE, FUENTE_SHA256), sid, backup, fuentes,
                      None, encargo='F1-27')
    exigir(m['copia_ref'] == c.copia.nombre and m['sid_copia'] == sid
           and m['recibo']['catalogo_sha256'] == m['catalogo_sha256'],
           'Tercer acto AUT58 sin recibo propio')
    return m


def _crear_rol(c, f, admision):
    cabeza, publicado = roles_admin._cabeza(c, admision)
    seleccion = roles_admin._entrada(publicado, f, ACCION)
    previo = roles_admin._rol(c, ROL)
    carpeta = c.carpeta / 'identidad_documental/rol'
    if previo is None:
        exigir(not carpeta.exists(), 'Gobierno del rol incierto: conciliar propuesta original')
        carpeta.mkdir(parents=True, mode=0o700)
        entrada = c.leer(c.raiz / 'admin/h10/entrada-vec-admin.json')
        cuerpo = {'operacion_ref': 'propuesta_admin:' + secrets.token_hex(16),
                  'catalogo_ref': catalogo.CATALOGO,
                  'catalogo_version': cabeza['version'],
                  'catalogo_huella_sha256': admision['catalogo_sha256'],
                  **seleccion, 'rol_id': ROL,
                  'nombre': 'Redacción de documentos de contratación temporal',
                  'motivo': entrada['motivo']}
        roles_admin._proponer_y_cerrar(c, carpeta, 'roles', cuerpo)
        previo = roles_admin._rol(c, ROL)
    esperado = canon.concesion(f['entradas'][0]['concesion'])
    exigir(previo and previo['ref'] == ROL_REF and previo['version'] == 1
           and previo['control_estado'] == 'habilitada'
           and len(previo['documento']['concesiones']) == 1
           and canon.concesion(previo['documento']['concesiones'][0]) == esperado,
           'Rol documental sin concesión única publicada')
    return previo


def _gestion(c):
    identidad = c.leer(c.material / 'identidad/identidad.json')
    bolsa = c.leer(c.material / 'identidad/bolsa-bback.json')
    tramitacion = c.leer(c.carpeta / 'documentos/tramitacion-fuente.json')
    firmantes = c.leer(c.carpeta / 'firmantes/semilla-final/firmantes-datos.json')
    cert = c.material / 'mtls/cliente.crt'
    exigir(cert.is_file() and not cert.is_symlink() and cert.stat().st_mode & 0o077 == 0,
           'Certificado de Gestión inseguro')
    try:
        huella = hashlib.sha256(ssl.PEM_cert_to_DER_cert(cert.read_text())).hexdigest()
    except (OSError, ValueError) as error:
        raise ValueError('Certificado de Gestión ilegible') from error
    refs = tramitacion['referencias']
    exigir(HEX.fullmatch(huella) and identidad.get('certificate_sha256') == huella
           and identidad.get('subject') == bolsa.get('sujeto') == tramitacion.get('sujeto')
           and bolsa.get('certificado_sha256') == tramitacion.get('certificado_sha256') == huella
           and refs.get('persona_ref') and refs.get('cuenta_ref')
           and firmantes['referencias']['tecnico']['certificado_sha256'] != huella,
           'Gestión no es el certificado original o coincide con firmante técnico')
    unidad = c.leer(c.raiz / 'admin/h10/unidad/acuse-primero.json')['recibo']['unidad']['unidad_ref']
    exigir(OPACA.fullmatch(unidad) and unidad == firmantes.get('unidad_ref'),
           'Unidad de Gestión ajena al acto ADMIN')
    return {'persona_ref': refs['persona_ref'], 'cuenta_ref': refs['cuenta_ref'],
            'certificado_sha256': huella, 'unidad_ref': unidad}


def _preparacion_lote(c, gestion):
    ruta = ('/api/admin/perfiles/v1/personas/' + quote(gestion['persona_ref'], safe='')
            + '/preparacion-lote?unidad_ref=' + quote(gestion['unidad_ref'], safe=''))
    estado, datos = c.admin(1, 'GET', ruta)
    p = datos.get('preparacion', {}) if isinstance(datos, dict) else {}
    altas = [x for x in p.get('altas', []) if x.get('rol_version_ref') == ROL_REF]
    motivos = [x for x in p.get('motivos', []) if x.get('clave_i18n') == 'alta_funciones']
    exigir(estado == 200 and p.get('persona_ref') == gestion['persona_ref']
           and p.get('cuenta_ref') == gestion['cuenta_ref']
           and p.get('unidad_ref') == gestion['unidad_ref']
           and p.get('truncado') is False and len(altas) == 1 and len(motivos) == 1,
           'Preparación ordinaria de Gestión incompleta')
    return p, altas[0], motivos[0]


def _asignar(c, gestion):
    base = c.carpeta / 'identidad_documental'
    cuerpo_ruta, acuse_ruta = base / 'lote-cuerpo.json', base / 'lote-acuse.json'
    pendiente = base / 'lote-pendiente.json'
    if acuse_ruta.exists():
        cuerpo, datos = c.leer(cuerpo_ruta), c.leer(acuse_ruta)
    else:
        exigir(not pendiente.exists() and not cuerpo_ruta.exists(),
               'Lote ADMIN incierto: conciliar operación original')
        prep, alta, motivo = _preparacion_lote(c, gestion)
        exigir(OPACA.fullmatch(alta.get('perfil_ref', ''))
               and OPACA.fullmatch(alta.get('vinculo_ref', ''))
               and HEX.fullmatch(alta.get('huella_sha256', '')),
               'Alta documental sin preimagen')
        objetivo = {k: prep[k] for k in ('unidad_ref', 'cuenta_ref', 'cuenta_version',
            'persona_ref', 'persona_version', 'procedencia_ref', 'procedencia_version',
            'procedencia_huella_sha256')}
        objetivo.update(perfil_ref=alta['perfil_ref'], perfil_version=0,
                        vinculo_ref=alta['vinculo_ref'], vinculo_version=0,
                        huella_sha256=alta['huella_sha256'], revision_continuidad=0,
                        vigente_hasta=alta['vigente_hasta_maxima'])
        cuerpo = {'operacion_ref': 'acto_admin:' + secrets.token_hex(16),
                  'cambios': [{'operacion': 'otorgar', 'inicio_vigencia': 'inmediato',
                               'rol_version_ref': ROL_REF, 'objetivo': objetivo}],
                  'motivo': {k: v for k, v in motivo.items() if k != 'clave_i18n'}}
        c.guardar(cuerpo_ruta, cuerpo)
        c.guardar(pendiente, {'operacion_ref': cuerpo['operacion_ref'],
                             'cuerpo_sha256': c.sha(cuerpo_ruta), 'estado': 'pendiente_de_conciliacion'})
        estado, datos = c.admin(1, 'POST', RUTA_LOTE, cuerpo)
        exigir(estado == 200, 'Lote ADMIN sin respuesta: conciliar operación original')
        c.guardar(acuse_ruta, datos)
    r = datos.get('recibo', {}) if isinstance(datos, dict) else {}
    exigir(r.get('operacion_ref') == cuerpo['operacion_ref']
           and OPACA.fullmatch(r.get('acto_ref', ''))
           and OPACA.fullmatch(r.get('recibo_ref', ''))
           and OPACA.fullmatch(r.get('auditoria_ref', ''))
           and HEX.fullmatch(r.get('huella_solicitud_sha256', ''))
           and len(r.get('cambios', [])) == 1,
           'Recibo del lote documental ausente o ajeno')
    objetivo = cuerpo['cambios'][0]['objetivo']
    exigir(objetivo['persona_ref'] == gestion['persona_ref']
           and objetivo['cuenta_ref'] == gestion['cuenta_ref']
           and objetivo['unidad_ref'] == gestion['unidad_ref']
           and cuerpo['cambios'][0]['rol_version_ref'] == ROL_REF,
           'Lote dirigido a otra identidad')
    return cuerpo, r


def _motivo_publicado(c, motivo):
    exigir(isinstance(motivo, dict) and set(motivo) == {
           'catalogo_id', 'catalogo_version', 'catalogo_huella_sha256', 'entrada_clave'}
           and isinstance(motivo['catalogo_version'], int)
           and motivo['catalogo_version'] > 0
           and HEX.fullmatch(motivo['catalogo_huella_sha256'])
           and OPACA.fullmatch(motivo['catalogo_id']) and OPACA.fullmatch(motivo['entrada_clave']),
           'Motivo documental privado inválido')
    q = roles_admin.literal
    consulta = ("SELECT count(*) FROM vec_autorizacion.motivo_v2_catalogo_publicado c "
        "JOIN vec_autorizacion.motivo_v2_entrada e USING(catalogo_id,catalogo_version) "
        f"WHERE c.catalogo_id={q(motivo['catalogo_id'])} "
        f"AND c.catalogo_version={motivo['catalogo_version']} "
        f"AND c.catalogo_huella_publicada_sha256={q(motivo['catalogo_huella_sha256'])} "
        f"AND e.entrada_clave={q(motivo['entrada_clave'])} "
        "AND e.vigente_desde<=clock_timestamp() "
        "AND (e.vigente_hasta IS NULL OR clock_timestamp()<e.vigente_hasta) "
        "AND NOT EXISTS(SELECT 1 FROM vec_autorizacion.motivo_v2_retirada r "
        "WHERE r.catalogo_id=c.catalogo_id AND r.catalogo_version=c.catalogo_version);")
    exigir(c.consultar(consulta).strip() == '1', 'Motivo documental no publicado')


def _configuracion(c, gestion, cuerpo, recibo, motivo):
    _motivo_publicado(c, motivo)
    entorno = c.copia.entorno_modulos()
    base = c.carpeta / 'identidad_documental'
    nombres = {}
    usuarios = set()
    for clave, variable in DSN.items():
        dsn = entorno.get(variable)
        usuario = urlsplit(dsn).username if isinstance(dsn, str) else None
        exigir(usuario and usuario not in usuarios and dsn.startswith('postgresql://'),
               'DSN documental no segregado o ausente')
        usuarios.add(usuario)
        nombre = clave + '.dsn'
        c.guardar(base / nombre, (dsn + '\n').encode())
        nombres[clave] = nombre
    objetivo = cuerpo['cambios'][0]['objetivo']
    config = {'esquema': 'vec.contratacion-temporal.identidad-documental-servidor.v1',
              'perfil_activo_ref': objetivo['perfil_ref'], 'version_rol_ref': ROL_REF,
              'acto_asignacion_ref': recibo['acto_ref'],
              'principal_id': gestion['persona_ref'], 'cuenta_ref': gestion['cuenta_ref'],
              'certificado_sha256': gestion['certificado_sha256'],
              'unidad_ref': gestion['unidad_ref'], 'motivo': motivo, 'dsn_files': nombres}
    ruta = base / 'identidad-documental.json'
    c.guardar(ruta, config)
    return ruta


def preparar(c, motivo_documental=None):
    """Prepara actos autorizados; la activación espera montaje y motivo real."""
    f = _fuente(c)
    admision = _admitir_fuente(c, f)
    rol = _crear_rol(c, f, admision)
    gestion = _gestion(c)
    cuerpo, recibo = _asignar(c, gestion)
    resultado = {'copia_ref': c.copia.nombre, 'sid_copia': c.copia.leer()['sid_copia'],
                 'fuente_sha256': FUENTE_SHA256, 'catalogo_sha256': admision['catalogo_sha256'],
                 'rol_ref': rol['ref'], 'acto_asignacion_ref': recibo['acto_ref'],
                 'recibo_lote_ref': recibo['recibo_ref'], 'estado': 'pendiente_motivo_y_montaje'}
    if motivo_documental is not None:
        ruta = _configuracion(c, gestion, cuerpo, recibo, motivo_documental)
        resultado['configuracion_ref'] = str(ruta)
        resultado['configuracion_sha256'] = c.sha(ruta)
        resultado['estado'] = 'preparado_sin_activar'
    c.guardar(c.carpeta / 'identidad_documental/resultado.json', resultado)
    return resultado
