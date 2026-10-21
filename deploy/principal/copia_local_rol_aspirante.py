"""Publica en una copia local la versión del rol del aspirante por actos de Administración.

Tres pasos, todos por las vías gobernadas y sin escribir permisos por SQL:

1. Admisión (AUT58) de la versión siguiente del catálogo
   ``catalogo:bolsa:carga_convoca:b1`` con la fuente de inscripción v2 y la de
   preferencias v2 en el sitio de sus versiones anteriores.
2. Propuesta del administrador 1 y cierre del administrador 2 con las cinco
   acciones de inscripción: el rol pasa de v1 a v2.
3. Otra propuesta y otro cierre con consultar y ordenar las bolsas propias,
   ya con la consulta de modalidades: el rol pasa de v2 a v3.

La única escritura SQL es la del DBA de la copia: un LOGIN técnico del grupo
``vec_admin_catalogo_acciones_ejecutor`` y su aprobación de un día en
``config_catalogo_acciones_admin_v1``, como pide el README de
``cmd/vec-catalogo-acciones-admitir``. Todo el material privado (contraseña,
plan, aprobación, cuerpos y recibos) queda en ``<copia>/admin/rol-aspirante``
y desaparece con ``borrar``. Repetir la orden reanuda con las mismas
referencias y bytes; un paso ya confirmado no se repite.
"""

import base64
import hashlib
import http.client
import json
import os
from pathlib import Path
import re
import secrets
import socket
import ssl
import subprocess
import time

CATALOGO = 'catalogo:bolsa:carga_convoca:b1'
ROL = 'candidato_bolsa_portal_historial_propio_desarrollo'
FUENTES = ('deploy/principal/fuentes/inscripcion_bolsa_actos_v2.json',
           'deploy/principal/fuentes/bolsa_preferencias_bolsas_propias_v2.json')
GRUPO_CATALOGO = 'vec_admin_catalogo_acciones_ejecutor'
AUDIENCIA = 'vec_autorizacion.versionar_rol_bolsa_externa.{}.v1'
RUTA_ACTOS = '/api/admin/perfiles/v1/gobierno-version-bolsa-externa/'
RUTA_SELECTOR = '/api/admin/seleccion-perfil/v1/'
# Cada acto deja el rol con estas concesiones; la acción marca que ya se hizo.
CAMPOS_MODALIDADES = ('modalidades_guardadas', 'modalidades_disponibles')
ACTOS = (('inscripcion', 'bolsa.inscripcion.presentar', 12),
         ('preferencias', 'bolsa.preferencias_bolsas.ordenar_propias', 14))
REFERENCIA = re.compile(r'[A-Za-z0-9][A-Za-z0-9._:-]{0,255}')
LOGIN = re.compile(r'vec_adm_cat_rolasp_[0-9a-f]{10}')
CLAVE = re.compile(r'[A-Za-z0-9_-]{32}')
INSTANTE = re.compile(r'\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ')


def exigir(condicion, motivo):
    if not condicion:
        raise ValueError(motivo)


def escribir(ruta, valor):
    """Escritura atómica 0600 de un JSON privado de la copia."""
    temporal = ruta.with_name('.' + ruta.name + '.tmp')
    descriptor = os.open(temporal, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600)
    with os.fdopen(descriptor, 'w') as salida:
        salida.write(json.dumps(valor, ensure_ascii=False, indent=1) + '\n')
        salida.flush()
        os.fsync(salida.fileno())
    os.replace(temporal, ruta)


def leer(ruta):
    return json.loads(ruta.read_text()) if ruta.exists() else None


def instante_micro():
    """Ahora con microsegundos: el control recién cerrado puede ser del mismo segundo."""
    t = time.time()
    return time.strftime('%Y-%m-%dT%H:%M:%S', time.gmtime(t)) + '.%06dZ' % int(t % 1 * 1e6)


def ahora():
    return time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime())


def consulta(copia, sql):
    """Lectura con transacción de solo lectura impuesta por el servidor."""
    r = subprocess.run(['docker', 'exec', '-i', '-u', 'postgres', '-e',
                        'PGOPTIONS=-c default_transaction_read_only=on', copia.nombre,
                        'psql', '-X', '-At', '-v', 'ON_ERROR_STOP=1', '-d', 'postgres'],
                       input=sql.encode(), capture_output=True, check=True, timeout=120)
    return r.stdout.decode().strip()


def literal(valor):
    exigir(REFERENCIA.fullmatch(valor), 'Referencia con caracteres no admitidos')
    return "'" + valor + "'"


def cabeza_catalogo(copia):
    fila = consulta(copia, f"""SELECT json_build_object('version',h.version,'huella',h.huella_sha256,
 'canon',encode(r.canon,'base64'),'paquete',encode(r.paquete_canon,'base64'))
 FROM vec_autorizacion.cabeza_catalogo_acciones_admin_v1 h
 JOIN vec_autorizacion.registro_catalogo_acciones_admin_v1 r
  ON r.catalogo_ref=h.catalogo_ref AND r.version=h.version
 WHERE h.catalogo_ref={literal(CATALOGO)}""")
    exigir(fila, 'El catálogo de acciones no está publicado en esta copia')
    c = json.loads(fila)
    for clave in ('canon', 'paquete'):
        c[clave] = base64.b64decode(c[clave]).decode()
    exigir(hashlib.sha256(c['canon'].encode()).hexdigest() == c['huella'], 'Cabeza del catálogo alterada')
    return c


def censo_perfiles(copia):
    return json.loads(consulta(copia, """SELECT json_agg(json_build_object(
 'rol',vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(v.documento),'control',c.documento)
 ORDER BY v.rol_id)
 FROM (SELECT DISTINCT ON (rol_id) * FROM vec_autorizacion.version_rol ORDER BY rol_id,version DESC) v
 JOIN vec_autorizacion.control_vigencia_version_rol_actual a USING(version_rol_ref)
 JOIN vec_autorizacion.control_vigencia_version_rol c
  ON c.version_rol_ref=a.version_rol_ref AND c.revision=a.revision"""))


def cabeza_rol(copia):
    fila = json.loads(consulta(copia, f"""SELECT json_build_object('ref',v.version_rol_ref,
 'version',v.version,'rol',vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(v.documento),
 'rol_huella',v.huella_sha256,'control',c.documento,'control_huella',c.huella_sha256)
 FROM (SELECT * FROM vec_autorizacion.version_rol WHERE rol_id={literal(ROL)}
  ORDER BY version DESC LIMIT 1) v
 JOIN vec_autorizacion.control_vigencia_version_rol_actual a USING(version_rol_ref)
 JOIN vec_autorizacion.control_vigencia_version_rol c
  ON c.version_rol_ref=a.version_rol_ref AND c.revision=a.revision"""))
    asignaciones = json.loads(consulta(copia, f"""SELECT coalesce(json_agg(json_build_object(
 'asignacion_ref',e.asignacion_ref,'huella_sha256',e.huella_sha256,'documento',e.documento)
 ORDER BY e.asignacion_ref),'[]')
 FROM vec_autorizacion.asignacion_perfil_actual_externa a
 JOIN vec_autorizacion.asignacion_perfil_externa e USING(perfil_activo_ref,asignacion_ref)
 WHERE e.version_rol_ref={literal(fila['ref'])} AND e.documento->>'estado'='activa'"""))
    exigir(1 <= len(asignaciones) <= 16, 'El rol del aspirante necesita entre 1 y 16 asignaciones activas')
    fila['asignaciones'] = asignaciones
    return fila


def herramienta(binario, orden, entrada):
    r = subprocess.run([str(binario), orden], input=json.dumps(entrada).encode(),
                       capture_output=True, timeout=120)
    exigir(r.returncode == 0, 'Preparación rechazada: ' + r.stderr.decode(errors='replace').strip()[-300:])
    return json.loads(r.stdout)


def compilar(carpeta, repo, go, entorno_go):
    bin_ = carpeta / 'bin'
    bin_.mkdir(mode=0o700, exist_ok=True)
    with (carpeta / 'compilacion.log').open('ab') as log:
        for nombre, paquete in (('rol_aspirante_copia', './deploy/principal/rol_aspirante_copia'),
                                ('admitir', './cmd/vec-catalogo-acciones-admitir')):
            subprocess.run([go, 'build', '-p', '6', '-trimpath', '-o', str(bin_ / nombre), paquete],
                           cwd=repo, env={**entorno_go, 'GOFLAGS': '-buildvcs=false'},
                           stdout=log, stderr=subprocess.STDOUT, check=True, timeout=600)
    return bin_ / 'rol_aspirante_copia', bin_ / 'admitir'


# --- 1. Catálogo ----------------------------------------------------------

def dba_catalogo(copia, carpeta, plan, aprobacion):
    """LOGIN técnico propio de la copia y su aprobación de un día (DBA local)."""
    acceso = carpeta / 'catalogo-acceso.json'
    datos = leer(acceso)
    if datos is None:
        sufijo = hashlib.sha256(copia.nombre.encode()).hexdigest()[:10]
        datos = dict(login=f'vec_adm_cat_rolasp_{sufijo}', clave=secrets.token_urlsafe(24))
        escribir(acceso, datos)
    p = json.loads(plan['plan'])
    # Se releen de ficheros privados al reanudar: no entran en el SQL sin su forma exacta.
    exigir(LOGIN.fullmatch(datos['login']) and CLAVE.fullmatch(datos['clave'])
           and INSTANTE.fullmatch(p['preparado_en']) and INSTANTE.fullmatch(p['caduca_en']),
           'Material privado del catálogo con forma inesperada')
    estado = copia.leer()
    escribir(carpeta / 'catalogo-config.json', dict(tiempo_segundos=40, dsn=(
        f"host=127.0.0.1 port={estado['puerto_pg']} dbname=postgres user={datos['login']} "
        f"password={datos['clave']} sslmode=verify-full sslrootcert={copia.material / 'ca/ca.crt'}")))
    login = datos['login']
    opciones = (f"LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS "
                f"PASSWORD '{datos['clave']}' VALID UNTIL '{p['caduca_en']}'")
    # La contraseña no llega al registro de PostgreSQL ni aunque la sentencia
    # falle; el LOGIN se crea o se actualiza, así que un corte no lo atasca.
    lineas = ['BEGIN;', "SET LOCAL log_statement='none';", "SET LOCAL log_min_error_statement='panic';",
              'SET LOCAL log_min_duration_statement=-1;',
              f"DO $$BEGIN IF EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname='{login}') "
              f"THEN ALTER ROLE {login} {opciones}; ELSE CREATE ROLE {login} {opciones}; "
              "END IF; END$$;",
              f'GRANT {GRUPO_CATALOGO} TO {login} WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;',
              'SET LOCAL ROLE vec_autorizacion_propietario;',
               f"DELETE FROM vec_autorizacion.config_catalogo_acciones_admin_v1 WHERE login_nombre='{login}';",
               'INSERT INTO vec_autorizacion.config_catalogo_acciones_admin_v1(login_nombre,plan_sha256,'
               'aprobacion_ref,aprobacion_sha256,aprobador_ref,paquete_ref,paquete_version,paquete_sha256,'
               'destino_ref,entorno,estado,vigente_desde,vigente_hasta) VALUES('
               + ','.join([f"'{login}'", literal(plan['plan_sha256']), literal(aprobacion['ref']),
                           literal(aprobacion['sha256']), literal('dba:copia-local:' + copia.nombre),
                           literal(p['paquete_ref']), str(int(p['paquete_version'])),
                           literal(p['paquete_sha256']), literal(CATALOGO), "'desarrollo'", "'admitida'",
                           f"timestamptz '{p['preparado_en']}'", f"timestamptz '{p['caduca_en']}'"]) + ');',
               'COMMIT;']
    copia.sql_bytes(('\n'.join(lineas) + '\n').encode(), 'rol-aspirante-dba.log')


def revocar_aprobacion(copia, carpeta):
    """Tras confirmar el recibo, la aprobación de un día deja de servir."""
    login = json.loads((carpeta / 'catalogo-acceso.json').read_text())['login']
    exigir(LOGIN.fullmatch(login), 'Material privado del catálogo con forma inesperada')
    copia.sql_bytes((f"BEGIN;\nALTER ROLE {login} NOLOGIN;\nSET LOCAL ROLE vec_autorizacion_propietario;\n"
                     "UPDATE vec_autorizacion.config_catalogo_acciones_admin_v1 SET estado='revocada' "
                     f"WHERE login_nombre='{login}';\nCOMMIT;\n").encode(), 'rol-aspirante-dba.log')


def admitir(binario, carpeta, plan):
    ruta = carpeta / 'catalogo-plan.json'
    ruta.write_bytes(plan['plan'].encode())
    ruta.chmod(0o600)
    r = subprocess.run([str(binario), '--fase', 'aplicar', '--config', str(carpeta / 'catalogo-config.json'),
                        '--plan', str(ruta), '--sha', plan['plan_sha256'], '--idioma', 'es'],
                       capture_output=True, timeout=120)
    try:
        salida = json.loads(r.stdout)
    except ValueError:
        salida = {}
    exigir(r.returncode == 0 and salida.get('recibo'),
           'Admisión del catálogo no confirmada: ' + str(salida.get('codigo') or r.returncode))
    return salida


def catalogo(copia, carpeta, repo, rac, binario_admitir):
    registro = carpeta / 'catalogo.json'
    hecho = leer(registro) or {}
    cabeza = cabeza_catalogo(copia)
    if hecho.get('recibo') and cabeza['huella'] == hecho['catalogo_sha256']:
        revocar_aprobacion(copia, carpeta)
        return hecho
    exigir(not hecho.get('recibo'), 'La cabeza del catálogo cambió después de admitirlo; revise la copia')
    plan = hecho.get('plan')
    vigente = plan and json.loads(plan['plan'])['caduca_en'] > time.strftime(
        '%Y-%m-%dT%H:%M:%SZ', time.gmtime(time.time() + 600))
    if plan and cabeza['huella'] == plan['catalogo_sha256'] and not vigente:
        # Admitido, sin recibo guardado y con la ventana cerrada: el recibo
        # se lee del registro, sin volver a presentar el plan.
        recibo = consulta(copia, f"""SELECT recibo FROM vec_autorizacion.registro_catalogo_acciones_admin_v1
 WHERE catalogo_ref={literal(CATALOGO)} AND huella_sha256={literal(plan['catalogo_sha256'])}""")
        hecho.update(recibo=json.loads(recibo))
        escribir(registro, hecho)
        revocar_aprobacion(copia, carpeta)
        return hecho
    if plan and cabeza['huella'] == plan['catalogo_sha256']:
        pass  # Admitido sin recibo guardado: se recupera con el mismo plan.
    elif vigente:
        # Plan guardado y aún dentro de su ventana: la aprobación se reescribe
        # igual (por si se interrumpió antes) y se presenta el mismo plan.
        dba_catalogo(copia, carpeta, plan, hecho['aprobacion'])
    else:
        fuentes = [json.loads((repo / f).read_text()) for f in FUENTES]
        aprobacion_doc = dict(tipo='aprobacion_tecnica_copia_local', copia=copia.nombre,
                              catalogo_ref=CATALOGO, esperado_version=cabeza['version'],
                              esperado_sha256=cabeza['huella'], preparado_en=ahora(),
                              fuentes={f: hashlib.sha256((repo / f).read_bytes()).hexdigest() for f in FUENTES})
        bytes_aprobacion = json.dumps(aprobacion_doc, sort_keys=True).encode()
        aprobacion = dict(documento=aprobacion_doc, sha256=hashlib.sha256(bytes_aprobacion).hexdigest(),
                          ref='aprobacion:copia-local:rol-aspirante:' + secrets.token_hex(8))
        preparado = time.time() - 60
        plan = herramienta(rac, 'catalogo', dict(
            paquete_canon=cabeza['paquete'], catalogo_canon=cabeza['canon'], catalogo_sha256=cabeza['huella'],
            fuentes=fuentes, perfiles=censo_perfiles(copia), operacion_ref='caa_' + secrets.token_hex(16),
            aprobacion_ref=aprobacion['ref'], aprobacion_sha256=aprobacion['sha256'], entorno='desarrollo',
            preparado_en=time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime(preparado)),
            caduca_en=time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime(preparado + 22 * 3600))))
        hecho = dict(plan=plan, aprobacion=aprobacion, catalogo_sha256=plan['catalogo_sha256'])
        escribir(registro, hecho)
        dba_catalogo(copia, carpeta, plan, aprobacion)
    primero = admitir(binario_admitir, carpeta, plan)
    repeticion = admitir(binario_admitir, carpeta, plan)
    exigir(repeticion['recibo'] == primero['recibo'], 'La repetición de la admisión devolvió otro recibo')
    exigir(cabeza_catalogo(copia)['huella'] == plan['catalogo_sha256'], 'La cabeza no quedó en el catálogo nuevo')
    hecho.update(recibo=primero['recibo'], repeticion_igual=True)
    escribir(registro, hecho)
    revocar_aprobacion(copia, carpeta)
    return hecho


# --- 2 y 3. Actos de Administración -----------------------------------------

class Administrador:
    """Cliente mTLS de vec-admin con las cabeceras que exige su frontera."""

    def __init__(self, run, numero, puerto):
        self.puerto, self.host = puerto, 'admin.localhost'
        self.ctx = ssl.create_default_context(cafile=str(run / 'vec-admin/tls/srv-ca.pem'))
        self.ctx.load_cert_chain(str(run / f'fuentes/cliente-{numero}.pem'),
                                 str(run / f'fuentes/cliente-{numero}.key'))

    def pedir(self, metodo, ruta, cuerpo=None):
        origen = f'https://{self.host}:{self.puerto}'
        cabeceras = {'Host': f'{self.host}:{self.puerto}', 'Accept': 'application/json',
                     'Sec-Fetch-Site': 'same-origin', 'Sec-Fetch-Mode': 'cors', 'Sec-Fetch-Dest': 'empty'}
        datos = None
        if cuerpo is not None:
            datos = json.dumps(cuerpo).encode()
            cabeceras.update({'Origin': origen, 'Content-Type': 'application/json'})
        con = http.client.HTTPSConnection(self.host, self.puerto, context=self.ctx, timeout=30)
        con.sock = self.ctx.wrap_socket(socket.create_connection(('127.0.0.1', self.puerto), timeout=30),
                                        server_hostname=self.host)
        try:
            con.request(metodo, ruta, body=datos, headers=cabeceras)
            r = con.getresponse()
            texto = r.read()
        finally:
            con.close()
        try:
            return r.status, json.loads(texto) if texto else {}
        except ValueError:
            return r.status, {}

    def perfil_aplicacion(self):
        estado, propios = self.pedir('GET', RUTA_SELECTOR + 'propios')
        exigir(estado == 200, f'Selector ADMIN respondió {estado}')
        elegidos = [p for p in propios.get('perfiles', []) if p.get('categoria_admin') == 'aplicacion']
        exigir(len(elegidos) == 1, 'El administrador no tiene un único perfil de Administración de la aplicación')
        if propios.get('perfil_activo_ref') != elegidos[0]['perfil_ref']:
            estado, _ = self.pedir('POST', RUTA_SELECTOR + 'seleccion', dict(
                perfil_ref=elegidos[0]['perfil_ref'], revision_esperada=propios['revision']))
            exigir(estado == 200, f'Selección de perfil ADMIN respondió {estado}')


def publicado_completo(concesiones, modo, esperado):
    if modo == 'inscripcion':
        return len(concesiones) >= esperado
    consulta = [c for c in concesiones if c['accion'] == 'bolsa.preferencias_bolsas.consultar_propias']
    return (len(concesiones) == esperado and len(consulta) == 1
            and set(CAMPOS_MODALIDADES) <= set(consulta[0].get('campos_permitidos') or []))


def acto(copia, carpeta, rac, modo, accion_marca, concesiones, admin1, admin2, motivos):
    registro = carpeta / f'acto-{modo}.json'
    hecho = leer(registro) or {}
    if hecho.get('cierre'):
        return hecho
    if not hecho.get('cuerpo'):
        rol = cabeza_rol(copia)
        acciones = [c['accion'] for c in rol['rol']['concesiones']]
        if accion_marca in acciones:
            # Solo se da por hecho el estado exacto que deja este acto, no uno parecido.
            exigir(publicado_completo(rol['rol']['concesiones'], modo, concesiones),
                   f'El rol tiene {modo} en un estado distinto del que publica esta orden; revise la copia')
            hecho = dict(ya_publicado=rol['ref'], concesiones=len(acciones))
            escribir(registro, hecho)
            return hecho
        cabeza = cabeza_catalogo(copia)
        preparada = herramienta(rac, 'propuesta', dict(
            modo=modo, operacion_ref='propuesta_admin:' + secrets.token_hex(16), catalogo_canon=cabeza['canon'],
            rol=rol['rol'], rol_huella_sha256=rol['rol_huella'], control=rol['control'],
            control_huella_sha256=rol['control_huella'], asignaciones=rol['asignaciones'],
            motivo=motivos['propuesta'],
            ahora=instante_micro()))
        exigir(preparada['concesiones'] == concesiones, 'El plan no deja el número de concesiones esperado')
        # Las referencias se guardan antes del primer envío: una respuesta
        # perdida se recupera con los mismos bytes, nunca con otra propuesta.
        hecho = dict(cuerpo=preparada['cuerpo'], objetivo=preparada['version_rol_objetivo_ref'],
                     cierre_ref='cierre_admin:' + secrets.token_hex(16))
        escribir(registro, hecho)
    if not hecho.get('propuesta'):
        admin1.perfil_aplicacion()
        estado, propuesta = admin1.pedir('POST', RUTA_ACTOS + 'propuestas', hecho['cuerpo'])
        exigir(estado in (200, 201), f'Propuesta {modo} respondió {estado}; si la cabeza del rol cambió '
               f'o la propuesta caducó, retire admin/rol-aspirante/{registro.name} y repita')
        estado2, repetida = admin1.pedir('POST', RUTA_ACTOS + 'propuestas', hecho['cuerpo'])
        exigir(estado2 == 200 and repetida.get('propuesta_ref') == propuesta['propuesta_ref']
               and repetida.get('replay') is True, 'La repetición de la propuesta no devolvió la misma')
        exigir(propuesta['version_rol_objetivo_ref'] == hecho['objetivo'], 'La propuesta apunta a otra versión')
        hecho.update(propuesta=propuesta, propuesta_estado=estado)
        escribir(registro, hecho)
    cuerpo = dict(operacion_ref=hecho['cierre_ref'], propuesta_ref=hecho['propuesta']['propuesta_ref'],
                  propuesta_huella_sha256=hecho['propuesta']['huella_sha256'], motivo=motivos['cierre'])
    admin2.perfil_aplicacion()
    estado, cierre = admin2.pedir('POST', RUTA_ACTOS + 'cierres', cuerpo)
    exigir(estado == 200 and cierre.get('recibo_ref'), f'Cierre {modo} respondió {estado}; si la propuesta '
           f'caducó, retire admin/rol-aspirante/{registro.name} y repita')
    estado2, repetido = admin2.pedir('POST', RUTA_ACTOS + 'cierres', cuerpo)
    exigir(estado2 == 200 and repetido.get('recibo_ref') == cierre['recibo_ref']
           and repetido.get('confirmado_en') == cierre['confirmado_en'],
           'La repetición del cierre no devolvió el mismo recibo')
    rol = cabeza_rol(copia)
    exigir(rol['ref'] == hecho['objetivo'] and len(rol['rol']['concesiones']) == concesiones,
           'El rol no quedó en la versión cerrada')
    hecho.update(cierre=cierre, concesiones=concesiones)
    escribir(registro, hecho)
    return hecho


def publicar(copia, *, repo, go, entorno_go):
    """Orden ``rol-aspirante``: requiere la copia creada con --administracion."""
    from copia_local_administracion import activo
    estado = copia.leer()
    exigir(estado.get('administracion') and estado.get('admin_preparada') and activo(copia, estado),
           'La copia necesita Administración preparada y en marcha (crear --administracion)')
    prep = json.loads((copia.raiz / 'admin_preparacion.json').read_text())
    run = copia.raiz / 'admin/h10'
    exigir(Path(prep['run']) == run and Path(prep['externa']).is_relative_to(run),
           'Preparación ADMIN fuera de la copia')
    externa = json.loads(Path(prep['externa']).read_text())
    motivos = {fase: externa['motivos'][AUDIENCIA.format(fase)] for fase in ('propuesta', 'cierre')}
    carpeta = copia.raiz / 'admin/rol-aspirante'
    carpeta.mkdir(mode=0o700, exist_ok=True)
    rac, binario_admitir = compilar(carpeta, repo, go, entorno_go)
    resumen = dict(catalogo=catalogo(copia, carpeta, repo, rac, binario_admitir))
    puerto = estado['administracion']['puerto']
    admin1, admin2 = Administrador(run, 1, puerto), Administrador(run, 2, puerto)
    for modo, marca, concesiones in ACTOS:
        resumen[modo] = acto(copia, carpeta, rac, modo, marca, concesiones, admin1, admin2, motivos)
    rol = cabeza_rol(copia)
    print(json.dumps(dict(
        catalogo_version=json.loads(resumen['catalogo']['plan']['plan'])['catalogo_version'],
        catalogo_operacion=resumen['catalogo']['recibo'].get('operacion_ref'),
        actos={m: (r.get('cierre') or {}).get('recibo_ref') or r.get('ya_publicado') for m, r in resumen.items()
               if m != 'catalogo'},
        rol=rol['ref'], concesiones=len(rol['rol']['concesiones'])), ensure_ascii=False, indent=1))
