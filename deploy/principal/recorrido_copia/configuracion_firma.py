"""Compone firma-vec con el plan y las cuatro admisiones verificadas."""

from datetime import datetime, timedelta, timezone
import hashlib
import json
from pathlib import Path
import re

ROLES = ('tecnico', 'jefatura', 'diputacion', 'secretaria')
HEX = re.compile(r'[0-9a-f]{64}\Z')
PLAN = 'ct.plan.firma.hz39.operativa'


def _sha(raw):
    return hashlib.sha256(raw).hexdigest()


def _canon(obj):
    return json.dumps(obj, ensure_ascii=False, separators=(',', ':'), allow_nan=False).encode()


def _fuente(c, ruta):
    ruta = Path(ruta)
    if (not ruta.is_absolute() or ruta.is_symlink() or not ruta.is_file()
            or ruta.stat().st_mode & 0o077
            or not (ruta.resolve().is_relative_to(c.carpeta.resolve())
                    or ruta.resolve().is_relative_to(c.material.resolve()))):
        raise ValueError('Fuente de configuración ajena a la copia')
    return c.leer(ruta)


def _vigencia(c, retirada):
    try:
        fin = datetime.fromisoformat(retirada.replace('Z', '+00:00'))
    except (AttributeError, ValueError) as error:
        raise ValueError('Retirada de firmantes inválida') from error
    ahora = c.ahora()
    if fin.tzinfo is None or not timedelta(days=60) <= fin - ahora <= timedelta(days=90):
        raise ValueError('La autoridad de firma debe cubrir 60 a 90 días')


def preparar(c):
    semilla_ruta = c.carpeta / 'firmantes/semilla-final/firmantes-datos.json'
    semilla = _fuente(c, semilla_ruta)
    firmantes_meta = _fuente(c, c.carpeta / 'firmantes/metadata.json')
    actos = _fuente(c, c.carpeta / 'actos/metadata.json')
    adm_meta = _fuente(c, c.carpeta / 'admisiones/metadata.json')
    adm_ruta = c.carpeta / 'admisiones/admisiones-bd.json'
    adm = _fuente(c, adm_ruta)
    plan_meta = _fuente(c, c.carpeta / 'plan/metadata.json')
    plan_ruta = c.material / 'identidad/firma-recursos/plan-firma-publicado.json'
    paquete = _fuente(c, plan_ruta)
    recibo_ruta = c.carpeta / 'plan/recibos/publicar.json'
    recibo = _fuente(c, recibo_ruta).get('recibo', {})
    if (firmantes_meta.get('copia_ref') != c.copia.nombre or actos.get('copia_ref') != c.copia.nombre
            or adm_meta.get('copia_ref') != c.copia.nombre or plan_meta.get('copia_ref') != c.copia.nombre
            or actos.get('actos_confirmados') != 14 or plan_meta.get('plan_publicado') is not True
            or adm_meta.get('filas') != 4 or c.sha(adm_ruta) != adm_meta.get('admisiones_sha256')
            or c.sha(plan_ruta) != plan_meta.get('paquete_sha256')
            or len(adm) != 4 or len(semilla.get('firmantes', [])) != 4
            or set(semilla.get('referencias', {})) != set(ROLES)):
        raise ValueError('Fuentes de firma incompletas o de otra copia')
    _vigencia(c, semilla.get('retirada_en'))
    if semilla['retirada_en'] != firmantes_meta.get('autoridad_hasta'):
        raise ValueError('Retirada de firmantes divergente de su fuente')
    catalogo = paquete.get('catalogo', {})
    sha_catalogo = plan_meta.get('catalogo_sha256')
    if (paquete.get('fuente', {}).get('demostracion') is not False
            or paquete['fuente'].get('origen_sha256') != sha_catalogo
            or catalogo.get('id') != PLAN or catalogo.get('version') != 1
            or catalogo.get('revision') != 1 or catalogo.get('estado') != 'publicado'
            or not catalogo.get('publicado_en') or not catalogo.get('aprobacion_ref')
            or len(catalogo.get('entradas', [])) != 5
            or any(e.get('atributos', {}).get('organizacion_ref') != c.parametros['organizacion_operativa_ref']
                   or e['atributos'].get('unidad_ref') != semilla.get('unidad_ref')
                   for e in catalogo['entradas'])
            or recibo.get('accion') != 'vec.catalogos.publicar'
            or recibo.get('catalogo_ref') != PLAN + ':1'
            or recibo.get('estado') != 'publicado' or recibo.get('revision') != 1
            or recibo.get('publicacion_sha256') != sha_catalogo
            or not recibo.get('recibo_ref') or not recibo.get('auditoria_ref')
            or recibo['recibo_ref'] != plan_meta['recibos'][1]):
        raise ValueError('Plan sin publicación y recibo coincidentes')
    perfiles = {x.get('perfil_firma_ref'): x for x in semilla['firmantes']}
    if len(perfiles) != 4:
        raise ValueError('Perfiles físicos de firmantes repetidos')
    documentos, admisiones, huellas, perfiles_usados = [], [], [], []
    if not HEX.fullmatch(semilla.get('politica_sha256', '')):
        raise ValueError('Política de autenticación sin huella propia')
    for rol in ROLES:
        ref = semilla['referencias'][rol]
        perfil = ref['perfil']['firma']
        firmante = perfiles.get(perfil)
        filas = [a for a in adm if a.get('perfil_ref') == perfil]
        respuesta_ruta = c.carpeta / 'actos/recibos' / (rol + '-certificado.json')
        respuesta = _fuente(c, respuesta_ruta).get('recibo', {})
        if (not firmante or len(filas) != 1
                or firmante.get('persona_ref') != ref['persona_ref']
                or firmante.get('cuenta_ref') != ref['cuenta_ref']
                or firmante.get('certificado_sha256') != ref['certificado_sha256']
                or not HEX.fullmatch(ref['certificado_sha256'])):
            raise ValueError('Firmante físico sin fuente propia')
        fila = filas[0]
        if (fila.get('rol_version_ref') != 'rol:ct_cargo_' + rol + '_firma_dev:v1'
                or fila.get('asignacion_version') != 1
                or fila.get('persona_ref') != ref['persona_ref']
                or fila.get('cuenta_ref') != ref['cuenta_ref']
                or fila.get('certificado_sha256') != ref['certificado_sha256']
                or not all(HEX.fullmatch(fila.get(k, '')) for k in
                           ('asignacion_sha256', 'rol_sha256', 'control_sha256'))
                or respuesta.get('estado') != 'vigente' or respuesta.get('version') != 1
                or respuesta.get('certificado_der_sha256') != ref['certificado_sha256']
                or respuesta.get('organizacion_destino', {}).get('organizacion_ref') != semilla['organizacion_ca_ref']
                or any(respuesta.get(k) != fila.get(k) for k in
                       ('recibo_ref', 'decision_ref', 'auditoria_ref'))
                or respuesta.get('vinculo_ref') != fila.get('vinculo_certificado_ref')):
            raise ValueError('Admisión y recibo nominal divergentes')
        slots = [perfil] + [firmante.get(clave) for clave in
                            ('perfil_consulta_ct_ref', 'perfil_documentos_ref', 'perfil_consulta_circuito_ref')]
        if any(not isinstance(x, str) or not x.startswith('prf_') for x in slots):
            raise ValueError('Falta un perfil separado de consulta o Documentos')
        perfiles_usados.extend(slots)
        politica = {'esquema': 'vec.ct.firma-vec.admision-desarrollo.v1',
                    'referencia': 'politica:ct:firma-vec:copia-local:' + rol + ':v1',
                    'version': 1, 'perfil_firma_ref': perfil,
                    'asignacion_ref': fila['asignacion_ref'],
                    'asignacion_version': fila['asignacion_version'],
                    'asignacion_sha256': fila['asignacion_sha256'],
                    'rol_version_ref': fila['rol_version_ref'],
                    'rol_sha256': fila['rol_sha256'],
                    'control_revision': fila['control_revision'],
                    'control_sha256': fila['control_sha256'],
                    'politica_autenticacion_ref': semilla['politica_ref'],
                    'politica_autenticacion_sha256': semilla['politica_sha256'],
                    'retirada_en': semilla['retirada_en']}
        destino = c.carpeta / 'configuracion' / ('admision-' + rol + '-v1.json')
        c.guardar(destino, _canon(politica))
        admisiones.append({'perfil_ref': perfil, 'referencia': politica['referencia'],
                           'version': 1, 'huella_sha256': _sha(_canon(politica)),
                           'rol_version_ref': fila['rol_version_ref'],
                           'rol_sha256': fila['rol_sha256'],
                           'control_revision': fila['control_revision'],
                           'control_sha256': fila['control_sha256']})
        documentos.append(firmante['perfil_documentos_ref'])
        huellas.append({'rol': rol, 'recibo_ref': respuesta['recibo_ref'],
                        'respuesta_sha256': c.sha(respuesta_ruta)})
    if len(set(perfiles_usados)) != 16 or len({a['asignacion_ref'] for a in adm}) != 4:
        raise ValueError('Asignaciones o perfiles de Documentos compartidos')
    crl_origen = Path(firmantes_meta['certificados_crl'])
    if (not crl_origen.is_absolute() or crl_origen.is_symlink() or not crl_origen.is_file()
            or crl_origen.stat().st_mode & 0o077
            or not crl_origen.resolve().is_relative_to(c.carpeta.resolve())):
        raise ValueError('Lista de revocación de otra copia')
    crl = c.material / 'identidad/firma-recursos/clientes.crl'
    c.guardar(crl, crl_origen.read_bytes())
    configuracion = {'version': 1, 'autoridad': 'no_autoritativo',
                     'politica_autenticacion_ref': semilla['politica_ref'],
                     'politica_autenticacion_sha256': semilla['politica_sha256'],
                     'retirada_en': semilla['retirada_en'], 'crl_path': str(crl),
                     'plan': {'archivo': str(plan_ruta), 'referencia': PLAN,
                              'version': 1, 'huella_sha256': sha_catalogo},
                     'firmantes': semilla['firmantes'], 'admisiones': admisiones}
    destino_config = c.material / 'identidad/firma-vec.json'
    c.guardar(destino_config, configuracion)
    meta = {'version': 1, 'copia_ref': c.copia.nombre,
            'estado': 'preparado_sin_activar', 'firmantes': 4,
            'admisiones_cotejadas': 4, 'plan_recibo_ref': recibo['recibo_ref'],
            'plan_recibo_sha256': c.sha(recibo_ruta),
            'firma_vec_sha256': c.sha(destino_config), 'crl_sha256': c.sha(crl),
            'certificados': huellas, 'gobierno_tecnico_roles': False,
            'cuentas_documentos': 'perfiles_de_semilla; configuracion_documentos_ajena_a_este_corte'}
    c.guardar(c.carpeta / 'configuracion/metadata.json', meta)
    return meta
