"""Materiales privados de los catorce actos nominales de firma de la copia."""

import base64
from datetime import datetime, timezone
import json
import hashlib
import re
import secrets
from pathlib import Path

from .firmantes import _roles

PASOS = (
    ('informe_definitivo.p1', 'tecnico', 'informe_definitivo', 1, 'ct_cargo_tecnico_firma_dev', 'perfil:ct:tecnico_rrhh', 'cargo:ct:tecnico:informe'),
    ('informe_definitivo.p2', 'jefatura', 'informe_definitivo', 2, 'ct_cargo_jefatura_firma_dev', 'perfil:ct:jefatura_servicio_rrhh', 'cargo:ct:jefatura:informe_definitivo'),
    ('resolucion.p1', 'jefatura', 'resolucion', 1, 'ct_cargo_jefatura_firma_dev', 'perfil:ct:jefatura_servicio_rrhh', 'cargo:ct:jefatura:resolucion'),
    ('resolucion.p2', 'diputacion', 'resolucion', 2, 'ct_cargo_diputacion_firma_dev', 'perfil:ct:diputacion_delegada_rrhh', 'cargo:ct:diputacion:resolucion'),
    ('resolucion.p3', 'secretaria', 'resolucion', 3, 'ct_cargo_secretaria_firma_dev', 'perfil:ct:secretaria_general', 'cargo:ct:secretaria:resolucion'),
)
ROLES = ('tecnico', 'jefatura', 'diputacion', 'secretaria')
CARGOS = '/api/admin/perfiles/v1/personal/cargos-competenciales/publicacion'
CERTIFICADOS = '/api/admin/perfiles/v1/certificados-nominales/publicacion'
REF = re.compile(r'[A-Za-z0-9][A-Za-z0-9:._-]{2,127}\Z')
HEX = re.compile(r'[0-9a-f]{64}\Z')


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


def canon(data):
    return json.dumps(data, ensure_ascii=False, separators=(',', ':'), allow_nan=False).encode()


def fecha(s):
    if not isinstance(s, str) or not re.fullmatch(r'\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z', s):
        raise ValueError('Fecha de firma inválida')
    return datetime.fromisoformat(s.replace('Z', '+00:00'))


def ref(value):
    if not isinstance(value, str) or not REF.fullmatch(value):
        raise ValueError('Referencia nominal inválida')
    return value


def json_propio(c, path):
    path = Path(path)
    if path.is_symlink() or not path.is_file() or path.stat().st_mode & 0o077:
        raise ValueError('Archivo privado de la copia ausente')
    return c.leer(path)


def _identidad(c):
    h10 = c.raiz / 'admin/h10'
    entrada = json_propio(c, h10 / 'entrada-vec-admin.json')
    plan = json_propio(c, h10 / 'bootstrap/plan-v3.json')['plan']
    personas = plan['personas']
    if len(personas) != 2 or personas[0]['persona_ref'] == personas[1]['persona_ref']:
        raise ValueError('Se requieren dos personas ADMIN distintas')
    for i, p in enumerate(personas, 1):
        a = json_propio(c, h10 / f'bootstrap/acreditacion-certificado-{i}.json')
        if a.get('persona_ref') != p['persona_ref']:
            raise ValueError('Certificado ADMIN ajeno a su persona')
    if not entrada.get('motivo'):
        raise ValueError('Falta el motivo administrativo propio de H10')
    unidad = json_propio(c, h10 / 'unidad/acuse-primero.json')
    if (unidad.get('estado') != 'permitido' or not unidad.get('recibo', {}).get('auditoria_ref')
            or not isinstance(unidad['recibo'].get('unidad'), dict)):
        raise ValueError('Unidad de H10 sin recibo acreditado')
    return personas, entrada['motivo'], unidad['recibo']['unidad']


def preparar_materiales(c):
    """Fija referencias, claves y bytes una vez; en reentrada coteja todo."""
    destino = c.carpeta / 'actos'
    manifiesto = destino / 'acciones.json'
    firmantes = json_propio(c, c.carpeta / 'firmantes/semilla-final/firmantes-datos.json')
    prep = json_propio(c, c.carpeta / 'firmantes/semilla-final/preparacion.json')
    meta_f = json_propio(c, c.carpeta / 'firmantes/metadata.json')
    raiz_f = (c.carpeta / 'firmantes').resolve()
    esperado_semilla = 'f131-' + sha(c.copia.nombre.encode())[:12]
    if (meta_f.get('copia_ref') != c.copia.nombre or meta_f.get('semilla_copia_ref') != esperado_semilla
            or meta_f.get('firmantes_datos') != str(c.carpeta / 'firmantes/semilla-final/firmantes-datos.json')
            or meta_f.get('preparacion') != str(c.carpeta / 'firmantes/semilla-final/preparacion.json')
            or meta_f.get('roles_iniciales') != 16 or meta_f.get('sucesoras_consulta') != 4
            or 'sin aprobación de dos administradores' not in meta_f.get('alcance', '')):
        raise ValueError('Semilla técnica de firmantes ajena a la copia')
    if (sha((c.raiz / 'admin/h10/unidad/acuse-primero.json').read_bytes())
            != meta_f.get('unidad_h10_acuse_sha256')):
        raise ValueError('Recibo H10 de unidad alterado')
    for clave, huella in (('roles_acuse', 'roles_acuse_sha256'),
                          ('consulta_v2_acuse', 'consulta_v2_acuse_sha256')):
        ruta = Path(meta_f[clave])
        if ruta.resolve().parent != raiz_f or sha(ruta.read_bytes()) != meta_f[huella]:
            raise ValueError('Acuse AUT53 de otra copia o alterado')
    roles_v1 = _roles(meta_f['roles_acuse'], meta_f['roles_plan'], 16)
    _roles(meta_f['consulta_v2_acuse'], meta_f['consulta_v2_plan'], 4)
    if any('rol:' + p[4] + ':v1' not in roles_v1 for p in PASOS):
        raise ValueError('Un paso carece de rol AUT53 acreditado')
    if prep.get('actos_ejecutados', 0) != 0 or prep.get('sql_ejecutadas', 0) != 0:
        raise ValueError('La semilla de firmantes no es previa a los actos')
    if set(firmantes.get('referencias', {})) != set(ROLES):
        raise ValueError('Faltan las cuatro fuentes de firmantes')
    unidad = firmantes['unidad_ref']
    org = firmantes['organizacion_ca_ref']
    nodo = firmantes['nodo_ref']
    for item in (unidad, org, nodo):
        ref(item)
    personas, motivo, unidad_h10 = _identidad(c)
    if (unidad_h10.get('organismo_ref') != org or unidad_h10.get('unidad_ref') != unidad
            or unidad_h10.get('nodo_ref') != nodo or unidad_h10.get('revision') != 1):
        raise ValueError('Unidad de firmantes ajena al acto H10')
    ahora_comprobacion = c.ahora()
    if isinstance(ahora_comprobacion, str):
        ahora_comprobacion = fecha(ahora_comprobacion)
    fin_comprobacion = fecha(firmantes['retirada_en'])
    if (fin_comprobacion - ahora_comprobacion).days < 60:
        raise ValueError('Vigencia de certificados insuficiente')
    for rol in ROLES:
        f = firmantes['referencias'][rol]
        cert = Path(f['certificado'])
        if not cert.is_absolute() or cert.is_symlink() or not cert.is_file() or cert.stat().st_mode & 0o077:
            raise ValueError('Certificado firmante propio ausente')
        r = c.ejecutar(['/usr/bin/openssl', 'x509', '-in', str(cert), '-outform', 'DER'],
                       f'actos-cert-{rol}-der.log')
        if r.returncode or sha(r.stdout) != f['certificado_sha256']:
            raise ValueError('Certificado firmante modificado')
        segundos = int((fin_comprobacion - ahora_comprobacion).total_seconds())
        r = c.ejecutar(['/usr/bin/openssl', 'x509', '-in', str(cert), '-checkend', str(segundos), '-noout'],
                       f'actos-cert-{rol}-vigencia.log')
        if r.returncode:
            raise ValueError('Certificado firmante no cubre su vigencia')
    copia_ref = c.copia.nombre
    if not re.fullmatch(r'[a-z][a-z0-9_-]{2,40}', copia_ref):
        raise ValueError('Identificador de copia inválido')
    fuente = c.carpeta / 'firmantes/semilla-final/firmantes-datos.json'
    fuente_sha = sha(fuente.read_bytes())
    if manifiesto.exists():
        m = json_propio(c, manifiesto)
        if m.get('copia_ref') != copia_ref or m.get('fuente_sha256') != fuente_sha or len(m.get('acciones', [])) != 14:
            raise ValueError('Propuestas anteriores de otra copia o fuente')
        verificar_materiales(c, m)
        return m
    ahora = c.ahora()
    if isinstance(ahora, str):
        ahora = fecha(ahora)
    ahora = ahora.astimezone(timezone.utc).replace(microsecond=0)
    fin = fecha(firmantes['retirada_en'])
    if fin < ahora.replace(microsecond=0) or (fin - ahora).days < 60:
        raise ValueError('Vigencia de los firmantes insuficiente')
    desde = ahora.strftime('%Y-%m-%dT%H:%M:%SZ')
    hasta = fin.strftime('%Y-%m-%dT%H:%M:%SZ')
    desde_us = ahora.strftime('%Y-%m-%dT%H:%M:%S.000000Z')
    hasta_us = fin.strftime('%Y-%m-%dT%H:%M:%S.%fZ')
    stamp = ahora.strftime('%Y%m%d')
    fuente_ref = f'fuente:ensayo:{copia_ref}:cargos:{stamp}'
    acciones, pasos = [], []

    def agregar(nombre, tipo, material, objeto, cert_sha=''):
        raw = canon(material)
        peticion = canon({'material_base64': base64.b64encode(raw).decode()})
        p = destino / 'materiales' / (nombre + '.json')
        q = destino / 'peticiones' / (nombre + '.json')
        c.guardar(p, raw)
        c.guardar(q, peticion)
        acciones.append({'id': nombre, 'tipo': tipo, 'admin': 1,
                         'ruta': CERTIFICADOS if tipo == 'certificado' else CARGOS,
                         'material': str(p), 'material_sha256': sha(raw),
                         'peticion': str(q), 'peticion_sha256': sha(peticion),
                         'objeto_esperado': objeto, 'certificado_sha256': cert_sha,
                         'motivo': motivo})

    for clave, rol, documento, orden, rol_id, perfil, denominacion in PASOS:
        f = firmantes['referencias'][rol]
        persona = ref(f['persona_ref'])
        ref(f['perfil']['firma'])  # Perfil activo del firmante; el esperado del paso viene del circuito v2.
        slug = f'{rol}-{documento}'
        cargo = 'car_' + secrets.token_hex(16)
        enlace = 'enc_' + secrets.token_hex(16)
        acto = f'acto:ensayo:{copia_ref}:cargo:{slug}'
        fuente_doc = {'tipo': 'cargo_ficticio_de_copia', 'cargo': rol, 'documento': documento,
                      'persona_ref': persona, 'vigente_desde': desde, 'vigente_hasta': hasta,
                      'orden': 'recorrido_firma_' + stamp}
        fuente_raw = canon(fuente_doc)
        c.guardar(destino / 'fuentes' / (slug + '.json'), fuente_raw)
        huella = sha(fuente_raw)
        evidencia = {'acto_ref': acto, 'acto_version': 1, 'acto_huella_sha256': huella,
                     'fuente_ref': fuente_ref, 'fuente_version': 1, 'fuente_huella_sha256': huella}
        datos_cargo = {'cargo_ref': cargo, 'version': 1, 'organizacion_ref': org, 'unidad_ref': unidad,
                       'puesto_ref': None, 'puesto_revision': None, 'organo_ref': nodo,
                       'organo_revision': unidad_h10['revision'], 'denominacion_catalogo_ref': denominacion,
                       'estado': 'vigente', 'vigente_desde': desde, 'vigente_hasta': hasta, **evidencia}
        agregar(slug + '-cargo', 'cargo', {'esquema': 'vec.personal.cargo-competencial.publicacion.v1',
                'operacion': 'cargo', 'clave_idempotencia': secrets.token_hex(16), 'objeto_ref': cargo,
                'organizacion_ref': org, 'unidad_ref': unidad, 'version_esperada': 0,
                'huella_esperada': None, 'datos': datos_cargo}, cargo)
        datos_enlace = {'enlace_ref': enlace, 'version': 1, 'cargo_ref': cargo, 'cargo_version': 1,
                        'persona_ref': persona, 'clase': 'titular', 'titular_enlace_ref': None,
                        'titular_enlace_version': None, 'titular_enlace_sha256': None,
                        'delegante_persona_ref': None,
                        'accion_ref': f'contratacion_temporal.documento.{documento}.firmar',
                        'recurso_ref': f'documento_{documento}_contratacion_temporal',
                        'finalidad_ref': 'formalizar_documento_ct', 'estado': 'vigente',
                        'vigente_desde': desde, 'vigente_hasta': hasta, **evidencia,
                        'requiere_enlace_laboral': False, 'empleado_ref': None,
                        'ocupacion_ref': None, 'ocupacion_revision': None}
        agregar(slug + '-enlace', 'enlace', {'esquema': 'vec.personal.cargo-competencial.publicacion.v1',
                'operacion': 'enlace', 'clave_idempotencia': secrets.token_hex(16), 'objeto_ref': enlace,
                'organizacion_ref': org, 'unidad_ref': unidad, 'version_esperada': 0,
                'huella_esperada': None, 'datos': datos_enlace}, enlace)
        pasos.append({'clave': clave, 'cargo': rol, 'documento': documento, 'orden': orden,
                      'cargo_ref': cargo, 'enlace_ref': enlace, 'rol_id': rol_id,
                      'perfil_esperado_ref': perfil, 'organizacion_canonica_ref': org,
                      'organizacion_operativa_ref': c.parametros['organizacion_operativa_ref'],
                      'unidad_ref': unidad})
    for rol in ROLES:
        f = firmantes['referencias'][rol]
        for k in ('persona_ref', 'cuenta_ref'):
            ref(f[k])
        cert_sha = f['certificado_sha256']
        if not isinstance(cert_sha, str) or not HEX.fullmatch(cert_sha):
            raise ValueError('Huella del certificado firmante inválida')
        descriptor = {'esquema': 'vec.contexto-actor.certificado-firmante.publicacion.v2',
                      'clave': secrets.token_hex(16), 'vinculo_ref': 'vcc_' + secrets.token_hex(16),
                      'version': 1, 'certificado_der_sha256': cert_sha, 'cuenta_ref': f['cuenta_ref'],
                      'persona_ref': f['persona_ref'], 'vinculo_cuenta_persona_ref': f['vinculo']['firma'],
                      'organizacion_ref': org, 'estado': 'vigente', 'vigente_desde': desde_us,
                      'vigente_hasta': hasta_us, 'evidencia_ref': 'evi_' + secrets.token_hex(16),
                      'evidencia_sha256': cert_sha, 'preimagen_ref': '', 'preimagen_version': 0,
                      'preimagen_sha256': 'none'}
        agregar(rol + '-certificado', 'certificado', descriptor, descriptor['vinculo_ref'], cert_sha)
    m = {'version': 1, 'copia_ref': copia_ref, 'fuente_sha256': fuente_sha,
         'admin_personas': [personas[0]['persona_ref'], personas[1]['persona_ref']],
         'acciones': acciones, 'pasos': pasos}
    c.guardar(manifiesto, m)
    verificar_materiales(c, m)
    return m


def verificar_materiales(c, m):
    esperados = [(p[1] + '-' + p[2] + '-' + tipo, tipo)
                 for p in PASOS for tipo in ('cargo', 'enlace')] + [
                     (rol + '-certificado', 'certificado') for rol in ROLES]
    if (len(m['acciones']) != 14 or
            [(a.get('id'), a.get('tipo')) for a in m['acciones']] != esperados):
        raise ValueError('Matriz de actos incompleta o reordenada')
    base = (c.carpeta / 'actos').resolve()
    for a in m['acciones']:
        p, q = Path(a['material']), Path(a['peticion'])
        if (a.get('admin') != 1 or a.get('ruta') != (CERTIFICADOS if a['tipo'] == 'certificado' else CARGOS)
                or p.name != a['id'] + '.json' or q.name != a['id'] + '.json'
                or p.is_symlink() or q.is_symlink() or p.stat().st_mode & 0o077 or q.stat().st_mode & 0o077
                or p.resolve().parent != base / 'materiales' or q.resolve().parent != base / 'peticiones'):
            raise ValueError('Material fuera de la carpeta de la copia')
        raw, req = p.read_bytes(), q.read_bytes()
        if sha(raw) != a['material_sha256'] or sha(req) != a['peticion_sha256']:
            raise ValueError('Material de acto alterado')
        if base64.b64decode(json.loads(req)['material_base64'], validate=True) != raw:
            raise ValueError('Petición divergente del material')
        dato = json.loads(raw)
        if a['tipo'] == 'certificado':
            if dato.get('vinculo_ref') != a['objeto_esperado'] or dato.get('certificado_der_sha256') != a['certificado_sha256']:
                raise ValueError('Certificado ajeno al acto')
        elif dato.get('objeto_ref') != a['objeto_esperado'] or dato.get('operacion') != a['tipo']:
            raise ValueError('Objeto ajeno al acto')
