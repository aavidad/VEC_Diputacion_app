#!/usr/bin/env python3
"""Preparación AUT58 de fuentes versionadas para una copia local sintética.

La aplicación y la consulta del acto corresponden a vec-catalogo-acciones-admitir.
"""
import argparse
import base64
import hashlib
import hmac
import json
import os
import re
import secrets
import stat
import subprocess
import sys
import uuid
from collections import OrderedDict
from datetime import datetime, timedelta, timezone
from pathlib import Path

# Al invocarlo como CLI, Python solo añade admin_copia/ a sys.path.
sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import copia_local
from admin_copia import arranque as admin_arranque

CATALOGO_REF = 'catalogo:bolsa:carga_convoca:b1'
GRUPO_ACTO = 'vec_admin_catalogo_acciones_ejecutor'
CFG_ACTO = 'config_catalogo_acciones_admin_v1'
TOPE_VENTANA_MIN = 1440
ENCARGO_REF = 'F1-131'
APROBADOR_REF = 'fabrica:copias:desarrollo'

def paro(motivo):
    print('PARO ' + motivo)
    sys.exit(1)


def copia_destino():
    raiz = Path(os.environ.get('VEC_COPIA_RAIZ', ''))
    if not raiz.is_absolute() or raiz.parent != copia_local.BASE:
        paro('copia-raiz-invalida')
    partes = re.fullmatch(r'([a-z])-([a-z0-9][a-z0-9-]{0,40})', raiz.name)
    if not partes:
        paro('copia-nombre-invalido')
    try:
        copia = copia_local.Copia(*partes.groups())
        if copia.raiz != raiz:
            paro('copia-raiz-invalida')
        copia.leer()
        # El socket y el ejecutable no proceden del entorno del proceso.
        if not copia.existe(entorno={'PATH': '/usr/bin:/bin', 'LANG': 'C',
                                    'DOCKER_HOST': 'unix:///var/run/docker.sock'}):
            paro('copia-contenedor-ausente')
    except (OSError, ValueError, KeyError, json.JSONDecodeError):
        paro('copia-no-pertenece-al-destino')
    return copia


def psql(sql):
    if sql not in (SQL_CABEZA, SQL_CENSO, SQL_SID):
        paro('consulta-no-autorizada')
    copia = copia_destino()
    orden = ['/usr/bin/docker', 'exec', '-i', '-u', 'postgres', copia.nombre,
             'psql', '-X', '-At', '-q', '-v', 'ON_ERROR_STOP=1', '-U', 'postgres', '-d', 'postgres']
    r = subprocess.run(orden, input=sql, text=True, capture_output=True, timeout=30,
                       env={'PATH': '/usr/bin:/bin', 'LANG': 'C'})
    if r.returncode:
        lineas = [x for x in r.stderr.splitlines() if 'PARO' in x or x.startswith('ERROR')]
        raise RuntimeError((lineas[-1] if lineas else 'psql')[:300])
    return r.stdout


def una_fila(salida):
    filas = [x for x in salida.splitlines() if x.strip()]
    if len(filas) != 1:
        raise RuntimeError('se esperaba una sola fila (%d)' % len(filas))
    return filas[0]


def sha(b):
    return hashlib.sha256(b).hexdigest()


def guardar(ruta, contenido):
    if isinstance(contenido, str):
        contenido = contenido.encode('utf-8')
    fd = os.open(ruta, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, 'wb') as f:
        f.write(contenido)
        f.flush()
        os.fsync(f.fileno())


def leer_privado(ruta):
    try:
        fd = os.open(ruta, os.O_RDONLY | os.O_NOFOLLOW)
    except OSError:
        paro('fichero-privado-inseguro ' + os.path.basename(ruta))
    with os.fdopen(fd, 'rb') as f:
        st = os.fstat(f.fileno())
        if (st.st_mode & 0o077) or st.st_uid != os.getuid() or not stat.S_ISREG(st.st_mode) or st.st_size > 32_000_000:
            paro('fichero-privado-inseguro ' + os.path.basename(ruta))
        return f.read()


def leer_json(ruta):
    try:
        return json.loads(leer_privado(ruta), object_pairs_hook=admin_arranque.objeto_sin_duplicados)
    except (admin_arranque.Paro, ValueError, UnicodeDecodeError):
        paro('json-privado-invalido')


def carpeta(d):
    if not os.path.isabs(d):
        paro('carpeta-no-absoluta')
    st = os.lstat(d)
    if not stat.S_ISDIR(st.st_mode) or (st.st_mode & 0o777) != 0o700 or st.st_uid != os.getuid() or os.path.islink(d):
        paro('carpeta-insegura ' + os.path.basename(d))


def ahora():
    return datetime.now(timezone.utc)


def instante(t):
    return t.strftime('%Y-%m-%dT%H:%M:%SZ')


def q(v):
    v = str(v)
    if "'" in v or '\\' in v:
        raise ValueError('literal')
    return "'" + v + "'"


def ident(v):
    if not re.fullmatch(r'[a-z][a-z0-9_]{5,62}', v):
        paro('login-no-admitido')
    return v


def scram(clave):
    sal = secrets.token_bytes(16)
    salted = hashlib.pbkdf2_hmac('sha256', clave.encode(), sal, 4096)
    ck = hmac.new(salted, b'Client Key', hashlib.sha256).digest()
    sk = hmac.new(salted, b'Server Key', hashlib.sha256).digest()
    b = lambda x: base64.b64encode(x).decode()
    v = f'SCRAM-SHA-256$4096:{b(sal)}${b(hashlib.sha256(ck).digest())}:{b(sk)}'
    if not re.fullmatch(r'SCRAM-SHA-256\$4096:[A-Za-z0-9+/=]+\$[A-Za-z0-9+/=]+:[A-Za-z0-9+/=]+', v):
        paro('verificador')
    return v


# ---------- canon Go (igual que hzb1adm.py) ----------

def od(*pares):
    return OrderedDict(pares)


def instante_go(s):
    if s.endswith('Z') and '.' in s:
        a, b = s[:-1].split('.', 1)
        b = b.rstrip('0')
        return a + ('.' + b if b else '') + 'Z'
    return s


def go_json(v):
    s = json.dumps(v, ensure_ascii=False, separators=(',', ':'))
    return (s.replace('&', '\\u0026').replace('<', '\\u003c').replace('>', '\\u003e')
            .replace('\u2028', '\\u2028').replace('\u2029', '\\u2029'))


def concesion(c):
    x = od(('accion', c['accion']), ('modulo_id', c['modulo_id']), ('tipo_recurso', c['tipo_recurso']),
           ('finalidades', c['finalidades']), ('garantia_minima', c['garantia_minima']))
    if c.get('campos_permitidos'):
        x['campos_permitidos'] = c['campos_permitidos']
    if c.get('obligaciones'):
        x['obligaciones'] = c['obligaciones']
    return x


def rol_go(r, completar=True):
    x = od(('rol_id', r['rol_id']), ('version', r['version']), ('nombre', r['nombre']), ('estado', r['estado']),
           ('concesiones', [concesion(c) for c in r['concesiones']]), ('publicada_por', r['publicada_por']),
           ('publicada_en', instante_go(r['publicada_en'])))
    if r.get('retirada_por'):
        x['retirada_por'] = r['retirada_por']
    # F1-103: la huella Go del rol lleva siempre retirada_en (instante cero); la normalización SQL del censo
    # solo la trae si el documento la guarda (los roles de cargo de la firma no la guardan).
    if completar or 'retirada_en' in r:
        x['retirada_en'] = instante_go(r.get('retirada_en', '0001-01-01T00:00:00Z'))
    if r.get('retirada_ref'):
        x['retirada_ref'] = r['retirada_ref']
    if r.get('motivo_retirada_codigo'):
        x['motivo_retirada_codigo'] = r['motivo_retirada_codigo']
    return x


def control_go(c):
    x = od(('version_rol_ref', c['version_rol_ref']), ('revision', c['revision']), ('estado', c['estado']),
           ('actualizado_por', c['actualizado_por']), ('actualizado_en', instante_go(c['actualizado_en'])))
    if c.get('acto_ref'):
        x['acto_ref'] = c['acto_ref']
    if c.get('motivo_codigo'):
        x['motivo_codigo'] = c['motivo_codigo']
    return x


def entrada_go(e):
    return od(('referencia', e['referencia']), ('version', e['version']), ('fuente_ref', e['fuente_ref']),
              ('fuente_version', e['fuente_version']), ('fuente_huella_sha256', e['fuente_huella_sha256']),
              ('concesion', concesion(e['concesion'])), ('dimensiones_ambito', e['dimensiones_ambito']),
              ('clase_control', e['clase_control']), ('vigente_desde', instante_go(e['vigente_desde'])),
              ('vigente_hasta', instante_go(e.get('vigente_hasta', '0001-01-01T00:00:00Z'))))
SQL_CENSO = """BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;
WITH latest AS (SELECT DISTINCT ON (v.rol_id) v.* FROM vec_autorizacion.version_rol v ORDER BY v.rol_id,v.version DESC),
records AS (SELECT l.rol_id,l.huella_sha256,l.documento raw_role,
  vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(l.documento) rol,
  c.documento control_vigencia,
  CASE WHEN EXISTS(SELECT 1 FROM vec_autorizacion.version_rol h WHERE h.rol_id=l.rol_id
    AND (EXISTS(SELECT 1 FROM vec_autorizacion.perfil_fijo_categoria_nominal_v1 f WHERE f.version_rol_ref=h.version_rol_ref)
    OR EXISTS(SELECT 1 FROM vec_autorizacion.rol_sensible_exacto s WHERE s.version_rol_ref=h.version_rol_ref)))
    THEN 'fijo_sistema' ELSE 'administrable' END tipo_perfil
  FROM latest l JOIN vec_autorizacion.control_vigencia_version_rol_actual ca ON ca.version_rol_ref=l.version_rol_ref
  JOIN vec_autorizacion.control_vigencia_version_rol c ON c.version_rol_ref=ca.version_rol_ref AND c.revision=ca.revision)
SELECT COALESCE(jsonb_agg(to_jsonb(records) ORDER BY rol_id),'[]'::jsonb)::text FROM records;
ROLLBACK;"""

SQL_CABEZA = """BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;
SELECT jsonb_build_object('version',h.version,'huella',h.huella_sha256,'canon',convert_from(r.canon,'UTF8'),
 'paquete',convert_from(r.paquete_canon,'UTF8'),'paquete_ref',r.paquete_ref,'paquete_version',r.paquete_version)::text
FROM vec_autorizacion.cabeza_catalogo_acciones_admin_v1 h
JOIN vec_autorizacion.registro_catalogo_acciones_admin_v1 r USING(catalogo_ref,version)
WHERE h.catalogo_ref='catalogo:bolsa:carga_convoca:b1';
ROLLBACK;"""

SQL_SID = "SELECT system_identifier::text FROM pg_catalog.pg_control_system();"


def exigir_sha(valor, motivo):
    if not isinstance(valor, str) or not re.fullmatch(r'[0-9a-f]{64}', valor) or valor == '0' * 64:
        paro(motivo)
    return valor


def exigir_sid(valor):
    if not isinstance(valor, str) or not re.fullmatch(r'[0-9]{15,20}', valor) or int(valor) == 0:
        paro('sid-copia-invalido')
    return valor


def fuente_nueva(dir_fuentes, nombre, archivo_sha):
    if not re.fullmatch(r'[a-z0-9_]+_v[0-9]+\.json', nombre):
        paro('nombre-de-fuente-invalido')
    ruta = Path(dir_fuentes) / nombre
    if ruta.is_symlink() or not ruta.is_file():
        paro('fuente-no-incluida ' + nombre)
    raw = ruta.read_bytes()
    if not 0 < len(raw) <= 2_000_000:
        paro('archivo-de-fuente-tamano-invalido ' + nombre)
    if sha(raw) != exigir_sha(archivo_sha, 'archivo-sha-invalido'):
        paro('archivo-de-fuente-divergente ' + nombre)
    try:
        f = json.loads(raw, object_pairs_hook=OrderedDict)
        huella = exigir_sha(f['huella_sha256'], 'huella-de-fuente-invalida')
        entradas = f['entradas']
        proy = [od(('referencia', e['referencia']), ('version', e['version']), ('fuente_ref', e['fuente_ref']),
                   ('fuente_version', e['fuente_version']), ('concesion', concesion(e['concesion'])),
                   ('dimensiones_ambito', e['dimensiones_ambito']), ('clase_control', e['clase_control']),
                   ('vigente_desde', instante_go(e['vigente_desde'])),
                   ('vigente_hasta', instante_go(e['vigente_hasta']))) for e in entradas]
        if sha(f['entradas_canon'].encode()) != huella or go_json(proy) != f['entradas_canon']:
            paro('canon-de-fuente-divergente ' + nombre)
        if not entradas or any(e['fuente_ref'] != f['referencia'] or e['fuente_version'] != f['version']
                              or e['fuente_huella_sha256'] != huella or e['concesion']['modulo_id'] != f['modulo_id']
                              for e in entradas):
            paro('descriptor-de-fuente-divergente ' + nombre)
        return od(('modulo_id', f['modulo_id']), ('referencia', f['referencia']), ('version', f['version']),
                  ('huella_sha256', huella), ('entradas_canon', f['entradas_canon']),
                  ('entradas', [entrada_go(e) for e in entradas]))
    except (KeyError, ValueError, TypeError, UnicodeError):
        paro('fuente-invalida ' + nombre)


def preparar_catalogo(dir_fuentes, modo, especificaciones):
    if modo not in ('fuentes', 'sustituir') or not especificaciones or (modo == 'sustituir' and len(especificaciones) != 1):
        paro('modo-o-numero-de-fuentes-invalido')
    cab = json.loads(una_fila(psql(SQL_CABEZA)))
    if sha(cab['canon'].encode()) != cab['huella']:
        paro('cabeza-del-catalogo-no-cuadra-con-su-canon')
    prev_paq = json.loads(cab['paquete'], object_pairs_hook=OrderedDict)
    prev_cat = json.loads(cab['canon'], object_pairs_hook=OrderedDict)
    if go_json(prev_paq) != cab['paquete'] or go_json(prev_cat) != cab['canon']:
        paro('el-canon-vigente-no-se-reproduce')
    if (prev_paq['referencia'] != cab['paquete_ref'] or prev_paq['version'] != cab['paquete_version']
            or prev_cat['fuente_ref'] != cab['paquete_ref'] or prev_cat['version'] != cab['version']
            or prev_cat['fuente_huella_sha256'] != sha(cab['paquete'].encode())):
        paro('paquete-y-catalogo-vigentes-incoherentes')
    fuentes = list(prev_paq['fuentes'])
    plano = [e for f in fuentes for e in f['entradas']]
    if go_json(plano) != go_json(prev_cat['entradas']):
        paro('entradas-vigentes-no-salen-de-su-paquete')
    nuevos = [fuente_nueva(dir_fuentes, nombre, archivo_sha) for nombre, archivo_sha in especificaciones]
    if len({f['referencia'] for f in nuevos}) != len(nuevos):
        paro('fuentes-repetidas')
    acciones_vigentes = {e['concesion']['accion'] for e in plano}
    if modo == 'fuentes':
        refs = {f['referencia'] for f in fuentes}
        acciones_nuevas = [e['concesion']['accion'] for f in nuevos for e in f['entradas']]
        if any(f['referencia'] in refs for f in nuevos) or len(set(acciones_nuevas)) != len(acciones_nuevas) \
                or acciones_vigentes.intersection(acciones_nuevas):
            paro('fuente-o-accion-ya-admitida')
        fuentes.extend(nuevos)
    else:
        nueva = nuevos[0]
        posiciones = [i for i, f in enumerate(fuentes) if f['referencia'] == nueva['referencia']]
        if len(posiciones) != 1:
            paro('version-anterior-ausente')
        vieja = fuentes[posiciones[0]]
        if vieja['version'] + 1 != nueva['version'] or vieja['modulo_id'] != nueva['modulo_id']:
            paro('sucesion-de-fuente-invalida')
        antes = [e['concesion']['accion'] for e in vieja['entradas']]
        despues = [e['concesion']['accion'] for e in nueva['entradas']]
        if antes != despues or len(set(despues)) != len(despues):
            paro('sustitucion-cambia-acciones')
        fuentes[posiciones[0]] = nueva
    filas = json.loads(una_fila(psql(SQL_CENSO)))
    perfiles = []
    for f in filas:
        if sha(go_json(rol_go(f['raw_role'])).encode()) != f['huella_sha256']:
            paro('canon-de-rol-divergente ' + f['rol_id'])
        perfiles.append(od(('rol', rol_go(f['rol'], completar=False)),
                           ('control_vigencia', control_go(f['control_vigencia'])),
                           ('tipo_perfil', f['tipo_perfil'])))
    if not perfiles or len({p['rol']['rol_id'] for p in perfiles}) != len(perfiles):
        paro('censo-de-perfiles-invalido')
    pv = cab['paquete_version'] + 1
    paquete = od(('esquema', prev_paq['esquema']), ('referencia', cab['paquete_ref']), ('version', pv),
                 ('fuentes', fuentes), ('perfiles', perfiles))
    paquete_canon = go_json(paquete)
    entradas = [e for f in fuentes for e in f['entradas']]
    catalogo = od(('referencia', CATALOGO_REF), ('version', cab['version'] + 1),
                  ('fuente_ref', cab['paquete_ref']), ('fuente_version', pv),
                  ('fuente_huella_sha256', sha(paquete_canon.encode())),
                  ('vigente_desde', prev_cat['vigente_desde']), ('vigente_hasta', prev_cat['vigente_hasta']),
                  ('entradas', entradas), ('perfiles', perfiles))
    resumen = dict(prev_version=cab['version'], prev_huella=cab['huella'], paquete_ref=cab['paquete_ref'],
                   paquete_version=pv, entradas_antes=len(plano), entradas=len(entradas), perfiles=len(perfiles),
                   fijos=sum(p['tipo_perfil'] == 'fijo_sistema' for p in perfiles),
                   censo_preparado_sha256=sha(go_json(perfiles).encode()))
    return paquete_canon, go_json(catalogo), resumen


def especificaciones(o):
    specs = []
    for valor in o.suelta:
        nombre, sep, archivo_sha = valor.partition(':')
        if not sep:
            paro('suelta-sin-huella-de-archivo')
        specs.append((nombre, exigir_sha(archivo_sha, 'archivo-sha-invalido')))
    if len({n for n, _ in specs}) != len(specs):
        paro('suelta-repetida')
    return specs


def cmd_preparar(o):
    carpeta(o.dir)
    if any((Path(o.dir) / n).exists() for n in ('preparado.json', 'paquete.json', 'catalogo.json')):
        paro('acto-ya-preparado')
    sid = exigir_sid(o.sid_copia)
    if o.encargo_ref != ENCARGO_REF:
        paro('encargo-ref-invalido')
    specs = especificaciones(o)
    paquete, catalogo, r = preparar_catalogo(o.fuentes, o.modo, specs)
    for val, key in ((o.backup_ref, 'backup-ref'), (o.encargo_ref, 'encargo-ref')):
        if not re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9._:-]{1,127}', val):
            paro(key + '-invalido')
    exigir_sha(o.backup_sha, 'backup-sha-invalido')
    login = 'vec_caa_' + secrets.token_hex(8)
    meta = dict(acto=o.acto, modo=o.modo, sueltas=specs, sid_copia=sid, backup_ref=o.backup_ref,
                backup_sha256=o.backup_sha, encargo_ref=o.encargo_ref, login=login, grupo=GRUPO_ACTO,
                preparado=instante(ahora()), paquete_sha256=sha(paquete.encode()),
                catalogo_sha256=sha(catalogo.encode()), **r)
    meta['antes'] = '%d:%s' % (r['prev_version'], r['prev_huella'])
    meta['despues'] = '%d:%s' % (r['prev_version'] + 1, meta['catalogo_sha256'])
    guardar(os.path.join(o.dir, 'paquete.json'), paquete)
    guardar(os.path.join(o.dir, 'catalogo.json'), catalogo)
    guardar(os.path.join(o.dir, 'preparado.json'), json.dumps(meta, ensure_ascii=False, indent=1))
    print('PREPARADO catalogo v%d→v%d entradas %d→%d perfiles=%d catalogo_sha256=%s paquete_sha256=%s' % (
        r['prev_version'], r['prev_version'] + 1, r['entradas_antes'], r['entradas'], r['perfiles'],
        meta['catalogo_sha256'], meta['paquete_sha256']))


def revisar(o, meta):
    paquete, catalogo, _ = preparar_catalogo(o.fuentes, meta['modo'], meta['sueltas'])
    if sha(paquete.encode()) != meta['paquete_sha256'] or sha(catalogo.encode()) != meta['catalogo_sha256']:
        paro('cabeza-o-censo-cambiado-desde-preparar')
    if paquete.encode() != leer_privado(os.path.join(o.dir, 'paquete.json')) \
            or catalogo.encode() != leer_privado(os.path.join(o.dir, 'catalogo.json')):
        paro('material-preparado-alterado')


def cmd_revisar(o):
    carpeta(o.dir)
    meta = leer_json(os.path.join(o.dir, 'preparado.json'))
    if o.huella != meta['catalogo_sha256']:
        paro('huella-revisada-distinta')
    revisar(o, meta)
    print('REVISADO cabeza, censo y material iguales a los preparados')


def cmd_configurar(o):
    carpeta(o.dir)
    meta = leer_json(os.path.join(o.dir, 'preparado.json'))
    if (Path(o.dir) / 'plan.json').exists() or (Path(o.dir) / 'configurar.sql').exists():
        paro('acto-ya-configurado')
    revisar(o, meta)
    raw_aprobacion = leer_privado(o.aprobacion)
    try:
        aprobacion = json.loads(raw_aprobacion, object_pairs_hook=admin_arranque.objeto_sin_duplicados)
    except (admin_arranque.Paro, ValueError, UnicodeDecodeError):
        paro('aprobacion-externa-invalida')
    esperado = {'esquema': 'vec.admin.catalogo-acciones.aprobacion-tecnica-copia.v1',
                'catalogo_sha256': meta['catalogo_sha256'], 'paquete_sha256': meta['paquete_sha256'],
                'backup_ref': meta['backup_ref'], 'backup_sha256': meta['backup_sha256'],
                'encargo_ref': ENCARGO_REF, 'sid_copia': exigir_sid(meta['sid_copia']),
                'esperado_version': meta['prev_version'], 'esperado_sha256': meta['prev_huella'],
                'censo_preparado_sha256': meta['censo_preparado_sha256'],
                'aprobador_ref': APROBADOR_REF}
    if any(aprobacion.get(k) != v for k, v in esperado.items()):
        paro('aprobacion-no-ligada-a-la-copia-y-material')
    for k in ('aprobacion_ref', 'aprobador_ref'):
        if not isinstance(aprobacion.get(k), str) or not re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9._:-]{1,127}', aprobacion[k]):
            paro('aprobacion-ref-invalida')
    if aprobacion['aprobacion_ref'].upper().startswith('ENSAYO'):
        paro('aprobacion-ref-invalida')
    if meta['encargo_ref'] != ENCARGO_REF:
        paro('encargo-ref-invalido')
    sid_actual = exigir_sid(una_fila(psql(SQL_SID)))
    if sid_actual != meta['sid_copia']:
        paro('sid-de-copia-cambiado')
    copia = copia_destino()
    estado_copia = copia.leer()
    if estado_copia.get('sid_copia') != sid_actual:
        paro('sid-de-copia-cambiado')
    if not 30 <= o.ttl <= 1380 or not 1 <= o.pg_puerto <= 65535:
        paro('ventana-o-puerto-invalido')
    if not os.path.isabs(o.pg_ca) or not os.path.isfile(o.pg_ca) or os.path.islink(o.pg_ca):
        paro('ca-invalida')
    if not re.fullmatch(r'[A-Za-z0-9_./:-]+', o.pg_ca):
        paro('ca-ruta-invalida')
    if not re.fullmatch(r'[a-zA-Z0-9.-]+', o.pg_host) or not re.fullmatch(r'[a-z][a-z0-9_]{0,62}', o.pg_base):
        paro('destino-pg-invalido')
    if (o.pg_host != 'localhost' or o.pg_puerto != estado_copia.get('puerto_pg')
            or o.pg_base != 'postgres' or Path(o.pg_ca) != copia.material / 'ca/ca.crt'):
        paro('destino-pg-ajeno-a-la-copia')
    ca = Path(o.pg_ca)
    dato_ca = ca.lstat()
    if (ca.resolve(strict=True) != ca or not stat.S_ISREG(dato_ca.st_mode)
            or dato_ca.st_uid != os.getuid() or dato_ca.st_mode & 0o022):
        paro('ca-de-copia-insegura')
    t0 = ahora()
    cad = t0 + timedelta(minutes=o.ttl)
    paquete = leer_privado(os.path.join(o.dir, 'paquete.json')).decode()
    catalogo = leer_privado(os.path.join(o.dir, 'catalogo.json')).decode()
    plan = od(('esquema', 'vec.admin.catalogo-acciones.plan.v2'), ('operacion_ref', 'caa_' + uuid.uuid4().hex),
              ('aprobacion_ref', aprobacion['aprobacion_ref']), ('aprobacion_sha256', sha(raw_aprobacion)),
              ('paquete_canon', paquete), ('paquete_ref', meta['paquete_ref']),
              ('paquete_version', str(meta['paquete_version'])), ('paquete_sha256', meta['paquete_sha256']),
              ('catalogo_canon', catalogo), ('catalogo_ref', CATALOGO_REF),
              ('catalogo_version', str(meta['prev_version'] + 1)), ('catalogo_sha256', meta['catalogo_sha256']),
              ('esperado_version', str(meta['prev_version'])), ('esperado_sha256', meta['prev_huella']),
              ('preparado_en', instante(t0 - timedelta(seconds=5))), ('caduca_en', instante(cad)),
              ('entorno', 'desarrollo'))
    plan_canon = go_json(plan)
    plan_sha = sha(plan_canon.encode())
    desde = t0 - timedelta(minutes=1)
    hasta = min(cad + timedelta(minutes=1), desde + timedelta(minutes=TOPE_VENTANA_MIN))
    login = ident(meta['login'])
    clave = secrets.token_urlsafe(32)
    dsn = f'host={o.pg_host} port={o.pg_puerto} dbname={o.pg_base} user={login} password={clave} sslmode=verify-full sslrootcert={o.pg_ca} connect_timeout=10'
    verificador = scram(clave)
    guardar(os.path.join(o.dir, 'plan.json'), plan_canon)
    guardar(os.path.join(o.dir, 'conexion.json'), go_json(od(('dsn', dsn), ('tiempo_segundos', 30))))
    del clave, dsn
    v = f"{q(instante(desde))}::timestamptz,{q(instante(hasta))}::timestamptz"
    sql = ("\\set ON_ERROR_STOP on\nBEGIN;\n"
           "DO $sid$ BEGIN IF current_database() IS DISTINCT FROM 'postgres' "
           f"OR (SELECT system_identifier::text FROM pg_catalog.pg_control_system()) "
           f"IS DISTINCT FROM {q(sid_actual)} THEN "
           "RAISE EXCEPTION 'AUT58: SID o base de copia distintos'; END IF; END $sid$;\n"
           "SET LOCAL log_statement='none';\nSET LOCAL log_min_duration_statement=-1;\n"
           "SET LOCAL password_encryption='scram-sha-256';\n"
           f"DO $g$ BEGIN IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname={q(login)}) "
           f"OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname={q(GRUPO_ACTO)}) "
           f"OR EXISTS(SELECT 1 FROM vec_autorizacion.{CFG_ACTO} WHERE login_nombre={q(login)}) "
           "THEN RAISE EXCEPTION 'AUT58: destino ocupado o grupo ausente'; END IF; END $g$;\n"
           f"CREATE ROLE {login} LOGIN INHERIT NOSUPERUSER NOCREATEROLE NOCREATEDB NOREPLICATION NOBYPASSRLS "
           f"VALID UNTIL {q(instante(hasta))} PASSWORD {q(verificador)};\n"
           f"GRANT {GRUPO_ACTO} TO {login} WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;\n"
           "SET LOCAL ROLE vec_autorizacion_propietario;\n"
           f"INSERT INTO vec_autorizacion.{CFG_ACTO}(login_nombre,plan_sha256,aprobacion_ref,aprobacion_sha256,"
           "aprobador_ref,paquete_ref,paquete_version,paquete_sha256,destino_ref,entorno,estado,vigente_desde,vigente_hasta) "
           f"VALUES({q(login)},{q(plan_sha)},{q(aprobacion['aprobacion_ref'])},{q(sha(raw_aprobacion))},"
           f"{q(aprobacion['aprobador_ref'])},{q(meta['paquete_ref'])},{int(meta['paquete_version'])},"
           f"{q(meta['paquete_sha256'])},{q(CATALOGO_REF)},'desarrollo','admitida',{v});\n"
           "RESET ROLE;\n"
           f"DO $c$ BEGIN IF (SELECT count(*) FROM pg_auth_members m JOIN pg_roles l ON l.oid=m.member "
           f"WHERE l.rolname={q(login)})<>1 "
           f"OR EXISTS(SELECT 1 FROM pg_roles WHERE rolname={q(login)} "
           "AND (rolsuper OR rolcreaterole OR rolcreatedb OR rolreplication OR rolbypassrls "
           "OR NOT rolinherit OR rolconfig IS NOT NULL)) "
           f"OR has_database_privilege({q(login)},current_database(),'CREATE,TEMP') "
           f"OR has_schema_privilege({q(login)},'vec_autorizacion','CREATE') "
           "THEN RAISE EXCEPTION 'AUT58: LOGIN no minimo'; END IF; END $c$;\nCOMMIT;\n")
    guardar(os.path.join(o.dir, 'configurar.sql'), sql)
    guardar(os.path.join(o.dir, 'configurado.json'), json.dumps({
        'plan_sha256': plan_sha, 'operacion_ref': plan['operacion_ref'], 'caduca_en': plan['caduca_en'],
        'aprobacion_ref': aprobacion['aprobacion_ref'], 'aprobacion_sha256': sha(raw_aprobacion),
        'aprobador_ref': aprobacion['aprobador_ref'], 'login': login, 'sid_copia': sid_actual,
        'encargo_ref': ENCARGO_REF,
        'preimagen': {'catalogo_ref': CATALOGO_REF, 'version': meta['prev_version'],
                      'huella_sha256': meta['prev_huella'],
                      'censo_preparado_sha256': meta['censo_preparado_sha256']},
        'catalogo_sha256': meta['catalogo_sha256'], 'paquete_sha256': meta['paquete_sha256']
    }, ensure_ascii=False, indent=1))
    print('CONFIGURACION-LISTA login=%s plan_sha256=%s catalogo_sha256=%s' % (
        login, plan_sha, meta['catalogo_sha256']))


def cmd_acuses(o):
    a1 = leer_json(o.primero)
    a2 = leer_json(o.replay)
    r1, r2 = a1.get('recibo'), a2.get('recibo')
    if not r1 or r1 != r2:
        paro('recibos-distintos')
    if a1.get('replay') is True or a2.get('replay') is not True:
        paro('marcas-de-replay-invalidas')
    claves = [k for k in ('operacion_ref', 'auditoria_ref', 'auditoria_secuencia', 'confirmado_en',
                          'catalogo_version', 'catalogo_sha256', 'censo_sha256') if k in r1]
    print('RECIBO-IGUAL ' + json.dumps({k: r1[k] for k in claves}, ensure_ascii=False))


def main():
    os.umask(0o077)
    p = argparse.ArgumentParser()
    s = p.add_subparsers(dest='orden', required=True)
    x = s.add_parser('preparar')
    x.add_argument('--dir', required=True); x.add_argument('--fuentes', required=True)
    x.add_argument('--modo', choices=['fuentes', 'sustituir'], required=True)
    x.add_argument('--suelta', action='append', default=[], required=True, metavar='NOMBRE:SHA256_ARCHIVO')
    x.add_argument('--acto', required=True); x.add_argument('--encargo-ref', required=True)
    x.add_argument('--sid-copia', required=True)
    x.add_argument('--backup-ref', required=True); x.add_argument('--backup-sha', required=True)
    x = s.add_parser('revisar')
    x.add_argument('--dir', required=True); x.add_argument('--huella', required=True); x.add_argument('--fuentes', required=True)
    x = s.add_parser('configurar')
    x.add_argument('--dir', required=True); x.add_argument('--fuentes', required=True)
    x.add_argument('--aprobacion', required=True); x.add_argument('--pg-ca', required=True)
    x.add_argument('--pg-puerto', type=int, required=True); x.add_argument('--pg-host', default='localhost')
    x.add_argument('--pg-base', default='postgres'); x.add_argument('--ttl', type=int, default=120)
    x = s.add_parser('acuses')
    x.add_argument('--primero', required=True); x.add_argument('--replay', required=True)
    o = p.parse_args()
    try:
        {'preparar': cmd_preparar, 'revisar': cmd_revisar, 'configurar': cmd_configurar, 'acuses': cmd_acuses}[o.orden](o)
    except (KeyError, ValueError, TypeError, IndexError, OSError, json.JSONDecodeError) as e:
        paro('entrada-o-estado-invalido')
    except RuntimeError:
        paro('postgresql-rechazo')


if __name__ == '__main__':
    main()
