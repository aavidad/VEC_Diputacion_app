#!/usr/bin/env python3
"""Prepara el material privado de vec-admin (modo metadatos_v1) a partir del arranque 2+1.

No contiene secretos: los lee de los directorios privados indicados en su JSON de entrada.
Fases (cada una idempotente por O_EXCL; no sobrescribe nada):
  tls        CA del servidor + certificado de servidor del host ADMIN; CRL firmada por la CA ADMIN.
  activos    ensamblaje ADMIN desde web/static con la lista fija de internal/app/administracion/perfiles.go.
  pools      11 LOGIN H5 y 2 externos segregados: material 0600 y SQL del DBA.
  claves     extrae las 2 claves HMAC del gobierno del día a ficheros 0600 (bytes en bruto).
  base       VEC_ADMIN_PERFILES_CONFIG_FILE (fichero base heredado).
  overlay    VEC_ADMIN_USUARIOS_CONFIG_FILE (metadatos_v1) con el gobierno del día.
  runtime    VEC_ADMIN_RUNTIME_CONFIG_FILE (necesita el archivo de identificadores ya preparado).
  entorno    fichero env con las variables VEC_ADMIN_* para arrancar.
Uso: preparar_vec_admin.py --entrada entrada.json --fase <fase> [--dia AAAAMMDD]
"""
import argparse, base64, datetime as dt, hashlib, json, os, re, secrets, subprocess, sys
from pathlib import Path
from preparar_runtime import validar_copia, validar_entrada, validar_gobierno

def sha(b): return hashlib.sha256(b).hexdigest()
def crear(p, b, modo=0o600):
    p = Path(p)
    fd = os.open(p, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, modo)
    with os.fdopen(fd, 'wb') as f:
        f.write(b if isinstance(b, bytes) else b.encode()); f.flush(); os.fsync(f.fileno())
def dir700(p):
    p = Path(p); p.mkdir(parents=True, mode=0o700, exist_ok=True); os.chmod(p, 0o700); return p
def jdump(x): return json.dumps(x, ensure_ascii=False, indent=1) + '\n'
def osl(*a, cwd=None):
    r = subprocess.run(['/usr/bin/openssl', *map(str, a)], cwd=cwd, capture_output=True, text=True)
    if r.returncode: sys.exit('openssl falló: ' + ' '.join(map(str, a[:3])))

POOLS = [  # clave, grupo
    ('fuente_autorizacion', 'vec_autorizacion_fuente'),
    ('registro_autorizacion', 'vec_autorizacion_registro'),
    ('motivos', 'vec_autorizacion_motivos_evaluador'),
    ('registro_sesiones', 'vec_identidad_sesiones_v1_registrador'),
    ('revalidacion_sesiones', 'vec_identidad_sesiones_v1_revalidador'),
    ('cuentas_admin', 'vec_identidad_sesiones_v1_admin_perfiles_runtime'),
    ('lector', 'vec_admin_usuarios_lector'),
    ('intentos', 'vec_autorizacion_atestada_v3_registrador_intentos'),
    ('selector', 'vec_identidad_sesiones_v1_admin_preperfil'),
    ('frontera_tecnica', 'vec_admin_frontera_tecnica_ejecutor'),
    ('contexto', 'vec_contexto_actor_v1_admin_contexto'),
]
POOLS_EXTERNOS = [  # clave, nombre de archivo, único grupo
    ('version_bolsa_externa_ejecutor', 'version-bolsa-externa-ejecutor.json',
     'vec_admin_version_rol_bolsa_externa_ejecutor'),
    ('catalogo_acciones_lector_externa', 'catalogo-acciones-lector-externa.json',
     'vec_admin_catalogo_acciones_lector'),
]

def main():
    ap = argparse.ArgumentParser(); ap.add_argument('--entrada', required=True); ap.add_argument('--fase', required=True); ap.add_argument('--dia')
    a = ap.parse_args(); os.umask(0o077)
    e = json.load(open(a.entrada))
    R = Path(e['runner']); K = Path(e['kit_dir'])
    V = Path(e['directorio_vec_admin'])
    try:
        validar_copia(R, K, e['sid_copia'])
        validar_entrada(e, R, e['sid_copia'])
    except (ValueError, OSError, KeyError, json.JSONDecodeError) as exc:
        sys.exit(str(exc))
    if (not re.fullmatch(r'[a-z0-9_]{3,18}', e['sufijo_login'])
            or not re.fullmatch(r'[a-z0-9.-]{1,120}', e['pg_host'])
            or not re.fullmatch(r'[a-z][a-z0-9_]{0,62}', e['pg_base'])
            or not re.fullmatch(r'[a-z0-9.-]{3,120}', e['host'])
            or not re.fullmatch(r'[a-z0-9:./,_-]+', e['redes'])
            or not re.fullmatch(r'[a-z0-9:._/-]{3,120}', e['audiencia'])
            or not re.fullmatch(r'https://[a-z0-9.:/_-]{3,200}', e['espacio_identidad'])
            or not re.fullmatch(r'20[0-9]{2}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z', e['vigente_hasta_config'])
            or not re.fullmatch(r'20[0-9]{2}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z', e['retirada_en'])
            or any(not re.fullmatch(r'[a-z][a-z0-9._-]{1,79}', value) for value in e['procesos'].values())
            or e['directorio_vec_admin'] != str(R / 'vec-admin')
            or e['checkout'] != str(K / 'source')
            or e['gobierno_dir'] != str(R / 'gobierno')
            or e['identificadores'] != str(V / 'identificadores/identificadores.json')
            or e['emisor_identidad'] != e['espacio_identidad']
            or e['escucha'] != f"127.0.0.1:{json.load(open(K / 'contrato-copia-local-admin-v1.json'))['puerto_admin']}"):
        sys.exit('entrada de runtime inválida')
    if a.fase in ('claves', 'base', 'overlay', 'entorno') and a.dia != e.get('dia_gobierno'):
        sys.exit('día de gobierno ajeno a la entrada')
    V = dir700(V)
    F = R / 'fuentes'; U = R / 'unidad'
    if a.fase == 'tls':
        T = dir700(V / 'tls')
        crear(T / 'srv-ca.cnf', '[req]\ndistinguished_name=dn\nx509_extensions=v3\nprompt=no\n[dn]\nCN=VEC ADMIN servidor DEV\n[v3]\nbasicConstraints=critical,CA:TRUE,pathlen:0\nkeyUsage=critical,keyCertSign,cRLSign\nsubjectKeyIdentifier=hash\n')
        crear(T / 'srv.ext', f"basicConstraints=critical,CA:FALSE\nkeyUsage=critical,digitalSignature\nextendedKeyUsage=serverAuth\nsubjectAltName=DNS:{e['host']}\n")
        osl('genpkey', '-algorithm', 'EC', '-pkeyopt', 'ec_paramgen_curve:P-256', '-out', 'srv-ca.key', cwd=T)
        osl('req', '-new', '-x509', '-sha256', '-days', e['dias_tls_servidor'], '-key', 'srv-ca.key', '-config', 'srv-ca.cnf', '-extensions', 'v3', '-out', 'srv-ca.pem', cwd=T)
        osl('genpkey', '-algorithm', 'EC', '-pkeyopt', 'ec_paramgen_curve:P-256', '-out', 'servidor.key', cwd=T)
        osl('req', '-new', '-sha256', '-key', 'servidor.key', '-subj', '/CN=' + e['host'], '-out', 'servidor.csr', cwd=T)
        osl('x509', '-req', '-sha256', '-days', e['dias_tls_servidor'], '-in', 'servidor.csr', '-CA', 'srv-ca.pem', '-CAkey', 'srv-ca.key', '-set_serial', '0x' + secrets.token_hex(16), '-extfile', 'srv.ext', '-out', 'servidor.pem', cwd=T)
        for n in ('srv-ca.pem', 'servidor.pem', 'servidor.key', 'srv-ca.key'): os.chmod(T / n, 0o600)
        # CA cliente ADMIN = la CA del arranque; se copia aparte (VEC_ADMIN_CA_FILE).
        crear(T / 'ca-admin.pem', (F / 'ca.pem').read_bytes())
        emitir_crl(e, F, T)
        print('tls listo')
    elif a.fase == 'crl':
        emitir_crl(e, F, V / 'tls', renovar=True); print('crl renovada')
    elif a.fase == 'activos':
        W = Path(e['checkout']) / 'web' / 'static'; A = dir700(V / 'activos')
        rutas = ['admin/usuarios/index.html', 'admin/usuarios/entry.js', 'admin/usuarios/vista.js', 'admin/usuarios/render.js', 'admin/usuarios/contratos.js',
                 'admin/usuarios/metadatos.js', 'admin/usuarios/cliente.js', 'admin/usuarios/cambio-perfiles.js', 'admin/usuarios/cambio-contratos.js', 'admin/usuarios/lecturas-http.js', 'admin/usuarios/propuestas.js',
                 'admin/usuarios/propuestas-contratos.js', 'admin/usuarios/usuarios.css', 'administracion-perfiles/selector-perfil.js',
                 'administracion-perfiles/selector-perfil.css', 'favicon.svg', 'comun/idioma.js', 'comun/textos.js', 'comun/tema-vec.css',
                 'portal-empleado/portal.css', 'portal-empleado/portal-componentes.css', 'portal-empleado/portal-flujos.css',
                 'portal-empleado/portal-patrones.css', 'textos/idiomas.json']
        idiomas = json.load(open(W / 'textos/idiomas.json'))['idiomas']
        for i in idiomas:
            for c in ('admin-usuarios', 'admin-selector'): rutas.append(f"textos/{i['codigo']}/{c}.json")
        manifiesto = json.load(open(K / 'source-manifest.json'))['files']
        for r in rutas:
            if manifiesto.get('web/static/' + r) != sha((W / r).read_bytes()):
                sys.exit('activo ajeno al manifiesto')
            d = A / r; dir700(d.parent); crear(d, (W / r).read_bytes())
        print('activos', len(rutas))
    elif a.fase == 'pools':
        P = dir700(V / 'pools'); sufijo = e['sufijo_login']
        sid = str(e.get('sid_copia', ''))
        contrato = json.load(open(K / 'contrato-copia-local-admin-v1.json'))
        if (not re.fullmatch(r'[0-9]{15,20}', sid) or sid != contrato.get('sid_copia')
                or contrato.get('conjunto_capacidades') != 7 or contrato.get('version') != 2
                or type(e.get('pg_port')) is not int or not 1024 <= e['pg_port'] <= 65535):
            sys.exit('destino ajeno a la copia')
        nombres = {clave: f"vec_adm_{clave}_{sufijo}"[:63] for clave, _ in POOLS}
        nombres.update({clave: f"vec_adm_ext_{'version' if clave.startswith('version_') else 'catalogo'}_{sufijo}"[:63]
                        for clave, _, _ in POOLS_EXTERNOS})
        if len(set(nombres.values())) != len(nombres) or any(not re.fullmatch(r'[a-z][a-z0-9_]{2,62}', n) for n in nombres.values()):
            sys.exit('LOGIN técnico repetido o inválido')
        logins = {}
        externos = [nombres[clave] for clave, _, _ in POOLS_EXTERNOS]
        hasta_login = (dt.datetime.now(dt.timezone.utc) + dt.timedelta(days=90)).isoformat()
        sqls = ['BEGIN;', 'SET LOCAL search_path=pg_catalog,pg_temp;',
                "SET LOCAL timezone='UTC';", "SET LOCAL lock_timeout='5s';",
                "SET LOCAL statement_timeout='30s';",
                "SELECT pg_advisory_xact_lock(hashtextextended('vec:admin:pools-externos',0));",
                f"DO $pre$ BEGIN IF current_database()<>'{e['pg_base']}' OR current_user<>'postgres' "
                "OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper) OR "
                f"(SELECT system_identifier FROM pg_control_system())<>{sid}::bigint "
                f"OR EXISTS(SELECT 1 FROM pg_roles WHERE rolname IN ('{externos[0]}','{externos[1]}')) "
                "OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1 "
                "WHERE audiencia_consumo IN ('vec_autorizacion.versionar_rol_bolsa_externa.propuesta.v1',"
                "'vec_autorizacion.versionar_rol_bolsa_externa.cierre.v1') "
                f"AND login_nombre IN ('{externos[0]}','{externos[1]}')) "
                "THEN RAISE EXCEPTION 'destino o preimagen de pools externos divergente'; END IF; END $pre$;"]
        for clave, grupo in POOLS:
            login = nombres[clave]
            pw = secrets.token_urlsafe(24)
            logins[clave] = login
            dsn = f"host={e['pg_host']} port={e['pg_port']} dbname={e['pg_base']} user={login} password={pw} sslmode=verify-full sslrootcert={e['pg_ca']}"
            crear(P / f'{clave}.json', json.dumps({'dsn': dsn}))
            sqls.append(f"CREATE ROLE {login} LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD '{pw}';")
            sqls.append(f"GRANT {grupo} TO {login} WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;")
        for clave, archivo, grupo in POOLS_EXTERNOS:
            login = nombres[clave]
            pw = secrets.token_urlsafe(24)
            logins[clave] = login
            dsn = f"host={e['pg_host']} port={e['pg_port']} dbname={e['pg_base']} user={login} password={pw} sslmode=verify-full sslrootcert={e['pg_ca']}"
            crear(P / archivo, json.dumps({'dsn': dsn}))
            sqls.append(f"CREATE ROLE {login} LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD '{pw}' VALID UNTIL '{hasta_login}';")
            sqls.append(f"GRANT {grupo} TO {login} WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;")
        crear(V / 'logins.json', jdump(logins))
        hasta = e['vigente_hasta_config']; proc = e['procesos']
        # CONNECT del grupo lector: lo concede AUT47 (000047_connect_lector_usuarios_admin).
        sqls += [
            "SET LOCAL ROLE vec_identidad_sesiones_v1_propietario;",
            f"INSERT INTO vec_identidad_sesiones_v1.config_runtime_admin_perfiles_v1(identidad_login,proceso,entorno,host_admin,audiencia,espacio_identidad,vigente_hasta) VALUES('{logins['cuentas_admin']}','{proc['cuentas']}','{e['entorno']}','{e['host']}','{e['audiencia']}','{e['espacio_identidad']}','{hasta}');",
            f"INSERT INTO vec_identidad_sesiones_v1.config_runtime_admin_preperfil_v1(identidad_login,proceso,entorno,host_admin,audiencia,vigente_hasta) VALUES('{logins['selector']}','{proc['selector']}','{e['entorno']}','{e['host']}','{e['audiencia']}','{hasta}');",
            "RESET ROLE;",
            "SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;",
            f"INSERT INTO vec_autorizacion_atestada_v3.config_contexto_admin_pre_v2_v1(login_nombre,proceso,canal,vigente_desde,vigente_hasta) VALUES('{logins['cuentas_admin']}','{proc['cuentas']}','administracion_privilegiada',clock_timestamp()-interval '1 minute','{hasta}');",
            f"INSERT INTO vec_autorizacion_atestada_v3.config_contexto_admin_pre_v2_v1(login_nombre,proceso,canal,vigente_desde,vigente_hasta) VALUES('{logins['contexto']}','{proc['contexto']}','administracion_privilegiada',clock_timestamp()-interval '1 minute','{hasta}');",
            f"INSERT INTO vec_autorizacion_atestada_v3.configuracion_runtime_intentos(login_nombre,proceso,canal) VALUES('{logins['intentos']}','{proc['usuarios']}','administracion_privilegiada');",
            f"INSERT INTO vec_autorizacion_atestada_v3.config_frontera_admin_tecnica_v1(login_nombre,proceso,canal,vigente_desde,vigente_hasta) VALUES('{logins['frontera_tecnica']}','{proc['usuarios']}','administracion_privilegiada',clock_timestamp()-interval '1 minute','{hasta}');",
            f"INSERT INTO vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1(login_nombre,audiencia_consumo,operacion,proceso,canal_permitido) VALUES('{logins['lector']}','vec.admin.usuarios.listar.v1','administracion.usuarios.listar','{proc['usuarios']}','administracion_privilegiada');",
            f"INSERT INTO vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1(login_nombre,audiencia_consumo,operacion,proceso,canal_permitido) VALUES('{logins['lector']}','vec.admin.usuarios.consultar.v1','administracion.usuarios.consultar','{proc['usuarios']}','administracion_privilegiada');",
            f"INSERT INTO vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1(login_nombre,audiencia_consumo,operacion,proceso,canal_permitido) VALUES('{logins['version_bolsa_externa_ejecutor']}','vec_autorizacion.versionar_rol_bolsa_externa.propuesta.v1','administracion.perfiles.version_bolsa.proponer','vec-admin','administracion_privilegiada');",
            f"INSERT INTO vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1(login_nombre,audiencia_consumo,operacion,proceso,canal_permitido) VALUES('{logins['version_bolsa_externa_ejecutor']}','vec_autorizacion.versionar_rol_bolsa_externa.cierre.v1','administracion.perfiles.version_bolsa.aprobar','vec-admin','administracion_privilegiada');",
            "RESET ROLE;",
            "DO $post$ BEGIN IF (SELECT count(*) FROM pg_roles WHERE rolname IN "
            f"('{externos[0]}','{externos[1]}') AND rolcanlogin AND rolinherit "
            "AND NOT (rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls) "
            "AND rolconfig IS NULL AND rolconnlimit=-1 "
            f"AND rolvaliduntil='{hasta_login}'::timestamptz)=2 "
            f"AND (SELECT count(*) FROM pg_auth_members m WHERE m.member='{externos[0]}'::regrole "
            "AND m.roleid='vec_admin_version_rol_bolsa_externa_ejecutor'::regrole "
            "AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)=1 "
            f"AND (SELECT count(*) FROM pg_auth_members m WHERE m.member='{externos[1]}'::regrole "
            "AND m.roleid='vec_admin_catalogo_acciones_lector'::regrole "
            "AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)=1 "
            f"AND (SELECT count(*) FROM pg_auth_members WHERE member IN ('{externos[0]}'::regrole,'{externos[1]}'::regrole))=2 "
            f"AND NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE roleid IN ('{externos[0]}'::regrole,'{externos[1]}'::regrole)) "
            "AND (SELECT count(*) FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1 "
            f"WHERE login_nombre='{externos[0]}' AND proceso='vec-admin' AND canal_permitido='administracion_privilegiada' "
            "AND ((audiencia_consumo='vec_autorizacion.versionar_rol_bolsa_externa.propuesta.v1' "
            "AND operacion='administracion.perfiles.version_bolsa.proponer') OR "
            "(audiencia_consumo='vec_autorizacion.versionar_rol_bolsa_externa.cierre.v1' "
            "AND operacion='administracion.perfiles.version_bolsa.aprobar')))=2 "
            f"AND NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1 WHERE login_nombre='{externos[1]}') "
            "THEN NULL; ELSE RAISE EXCEPTION 'atributos, membresías u orígenes externos divergentes'; END IF; END $post$;",
            'COMMIT;']
        crear(V / 'dba-logins.sql', '\n'.join(sqls) + '\n')
        print('pools: SQL del DBA en dba-logins.sql (contiene contraseñas: 0600, no copiar)')
    elif a.fase == 'claves':
        dia = a.dia; G = Path(e['gobierno_dir']) / dia; K = dir700(V / 'claves' / dia)
        try: _, m, _ = validar_gobierno(R, dia, aplicado=True)
        except (ValueError, OSError, KeyError, json.JSONDecodeError) as exc: sys.exit(str(exc))
        for c in m['claves']:
            raw = base64.b64decode(c['secreto_hmac'])
            if sha(raw) != c['huella_secreto_sha256']: sys.exit('huella de secreto divergente')
            crear(K / (c['audiencia'] + '.bin'), raw)
        print('claves', dia)
    elif a.fase in ('base', 'overlay'):
        dia = a.dia; G = Path(e['gobierno_dir']) / dia
        try: _, m, cm = validar_gobierno(R, dia, aplicado=True)
        except (ValueError, OSError, KeyError, json.JSONDecodeError) as exc: sys.exit(str(exc))
        r, g = cm['Raiz'], cm['Gobierno']
        meta = {'cabecera': {'formato_version': 3, 'suite': 'VEC-AD-3-COSE-EDDSA-1', 'clave_id': r['ClaveID'], 'audiencia': r['Audiencia']},
                'raiz': {'clave_id': r['ClaveID'], 'audiencia': r['Audiencia'], 'version': r['Version'], 'publica_base64': r['Publica'], 'estado': 'activa',
                         'valida_desde': r['ValidaDesde'], 'valida_hasta': r['ValidaHasta'], 'revocada_en': '0001-01-01T00:00:00Z'},
                'gobierno': {'revision': g['Revision'], 'huella_sha256': g['HuellaSHA256'], 'secuencia': g['Secuencia'], 'publicada_en': g['PublicadaEn'], 'expira_en': g['ExpiraEn']}}
        motivo = e['motivo']
        if a.fase == 'base':
            # En metadatos_v1 las 11 capacidades heredadas sólo se validan en forma; no se leen.
            aud = ['vec_autorizacion.administracion_perfiles.ordinario.v1', 'vec_autorizacion.administracion_perfiles.propuesta.v1', 'vec_autorizacion.administracion_perfiles.cierre.v1',
                   'vec_autorizacion.administracion_perfiles.consulta.v1', 'vec_autorizacion.administracion_perfiles.lectura.consultar_capacidades.v1',
                   'vec_autorizacion.administracion_perfiles.lectura.buscar_personas.v1', 'vec_autorizacion.administracion_perfiles.lectura.consultar_persona.v1',
                   'vec_autorizacion.administracion_perfiles.lectura.listar_roles.v1', 'vec_autorizacion.administracion_perfiles.lectura.listar_propuestas.v1',
                   'vec_autorizacion.administracion_perfiles.lectura.consultar_propuesta.v1', 'vec_autorizacion.administracion_perfiles.lectura.consultar_recibo.v1']
            ent = [{'audiencia': x, 'clave_id': f'clave:capacidad:admin:perfiles:heredada:{i}', 'emisor_id': 'emisor:admin:perfiles:heredado', 'huella_gobierno': '0' * 63 + '1',
                    'version': 1, 'revision_gobierno': 1, 'material_archivo': str(V / 'no-usado' / f'{i}.bin'), 'estado': 'emision',
                    'valida_desde': '2026-01-01T00:00:00Z', 'valida_hasta': '2036-01-01T00:00:00Z', 'revocada_en': '0001-01-01T00:00:00Z'} for i, x in enumerate(aud)]
            P = V / 'pools'
            cfg = {'pools': {'fuente_autorizacion': str(P / 'fuente_autorizacion.json'), 'registro_autorizacion': str(P / 'registro_autorizacion.json'), 'motivos': str(P / 'motivos.json'),
                             'registro_sesiones': str(P / 'registro_sesiones.json'), 'revalidacion_sesiones': str(P / 'revalidacion_sesiones.json'), 'cuentas_admin': str(P / 'cuentas_admin.json'),
                             'actos_admin': '', 'auditoria_frontera': str(P / 'auditoria_frontera-no-usado.json')},
                   'confianza': {**meta, 'entradas_capacidad': ent},
                   'firmante': {'clave_id': r['ClaveID'], 'audiencia': r['Audiencia'], 'prefijo_evidencia': 'evidencia:firma:admin:usuarios:', 'clave_privada_archivo': e['semilla_raiz'], 'publica_esperada_base64': r['Publica']},
                   'identidad': {**e['identidad'], 'incluir_cuenta_ordinaria': True},
                   'motivos_lectura': {'administracion.usuarios.listar': motivo}, 'catalogo_motivos_id': motivo['catalogo_id'],
                   'vigencia_decision_segundos': 60, 'timeout_arranque_segundos': 30, 'activos_directorio': str(V / 'activos')}
            C = dir700(V / 'config'); crear(C / 'base.json', jdump(cfg)); print('base escrita')
        else:
            K = V / 'claves' / dia
            audiencias_usuarios = {'vec.admin.usuarios.listar.v1', 'vec.admin.usuarios.consultar.v1'}
            ent = [{'audiencia': c['audiencia'], 'clave_id': c['clave_id'], 'emisor_id': c['emisor_id'], 'huella_gobierno': c['huella_gobierno_sha256'], 'version': c['version'],
                    'revision_gobierno': c['revision_gobierno'], 'material_archivo': str(K / (c['audiencia'] + '.bin')), 'estado': 'emision',
                    'valida_desde': c['valida_desde'], 'valida_hasta': c['valida_hasta'], 'revocada_en': '0001-01-01T00:00:00Z'} for c in m['claves'] if c['audiencia'] in audiencias_usuarios]
            if len(ent) != 2: sys.exit('gobierno de usuarios incompleto')
            org = json.load(open(F / 'fuente.json'))['organizacion']['organizacion_ref']
            uni = json.load(open(U / 'fuente.json'))['unidad']['unidad_ref']
            conj = 'conjunto_admin:' + sha(b'vec.admin.conjunto-usuarios.v1\n' + json.dumps({'organizacion_ref': org, 'unidad_ref': uni}, separators=(',', ':')).encode())[:32]
            dest = {k: {'accion': 'administracion.perfiles.frontera.' + k, 'recurso_ref': 'administracion:perfiles:' + k, 'finalidad_ref': 'gestion_perfiles', 'tipo_recurso': 'fijo'}
                    for k in ('consultar', 'buscar_personas', 'consultar_persona', 'consultar_recibo', 'escribir', 'aplicar_ordinario', 'proponer', 'cerrar_propuesta', 'aplicar_lote_ordinario')}
            dest['buscar_personas'] = {'accion': 'administracion.usuarios.listar', 'recurso_ref': conj, 'finalidad_ref': 'gestion_usuarios', 'tipo_recurso': 'conjunto'}
            dest['consultar_persona'] = {'accion': 'administracion.usuarios.consultar', 'recurso_ref': '', 'finalidad_ref': 'gestion_usuarios', 'tipo_recurso': 'persona'}
            P = V / 'pools'
            cfg = {'modo': 'metadatos_v1', 'pool_lector': str(P / 'lector.json'), 'pool_intentos': str(P / 'intentos.json'), 'pool_frontera_tecnica': str(P / 'frontera_tecnica.json'),
                   'pool_selector': str(P / 'selector.json'), 'confianza': {**meta, 'entradas_capacidad': ent}, 'organizacion_ref': org, 'unidad_ref': uni,
                   'proceso': e['procesos']['usuarios'], 'canal': 'administracion_privilegiada',
                   'motivos_usuarios': {'vec.admin.usuarios.listar.v1': motivo, 'vec.admin.usuarios.consultar.v1': motivo},
                   'motivo_denegado': motivo, 'motivo_error': motivo, 'plazo_auditoria_ms': 2000, 'destinos': dest}
            C = dir700(V / 'config'); crear(C / f'usuarios-{dia}.json', jdump(cfg)); print('overlay escrito', dia)
    elif a.fase == 'runtime':
        I = Path(e['identificadores'])
        cfg = {'version': 1, 'pool_contexto': str(V / 'pools' / 'contexto.json'), 'fuente_identificadores_archivo': str(I),
               'fuente_identificadores_sha256': sha(I.read_bytes()), 'proceso_contexto': e['procesos']['contexto']}
        crear(V / 'config' / 'runtime.json', jdump(cfg)); print('runtime escrito')
    elif a.fase == 'entorno':
        dia = a.dia; T = V / 'tls'; C = V / 'config'
        env = {'VEC_ADMIN_ENTORNO': e['entorno'], 'VEC_ADMIN_ESCUCHA': e['escucha'], 'VEC_ADMIN_HOST': e['host'], 'VEC_ADMIN_AUDIENCIA': e['audiencia'],
               'VEC_ADMIN_EMISOR_IDENTIDAD': e['emisor_identidad'], 'VEC_ADMIN_TLS_CERT_FILE': str(T / 'servidor.pem'), 'VEC_ADMIN_TLS_KEY_FILE': str(T / 'servidor.key'),
               'VEC_ADMIN_CA_FILE': str(T / 'ca-admin.pem'), 'VEC_ADMIN_CRL_FILE': str(T / 'admin.crl.pem'), 'VEC_ADMIN_REDES_PERMITIDAS': e['redes'],
               'VEC_ADMIN_RETIRADA_EN': e['retirada_en'], 'VEC_ADMIN_PERFILES_CONFIG_FILE': str(C / 'base.json'),
               'VEC_ADMIN_USUARIOS_CONFIG_FILE': str(C / f'usuarios-{dia}.json'), 'VEC_ADMIN_RUNTIME_CONFIG_FILE': str(C / 'runtime.json')}
        crear(C / f'entorno-{dia}.env', ''.join(f'{k}={v}\n' for k, v in env.items())); print('entorno escrito', dia)
    else:
        sys.exit('fase desconocida')

def emitir_crl(e, F, T, renovar=False):
    """CRL vacía firmada por la CA ADMIN del arranque; nextUpdate = dias_crl."""
    W = dir700(T / 'crl-ca'); idx = W / 'index.txt'
    if not idx.exists(): crear(idx, b''); crear(W / 'crlnumber', b'1000\n')
    cnf = W / 'ca.cnf'
    if not cnf.exists():
        crear(cnf, f"[ca]\ndefault_ca=ca\n[ca]\ndatabase={idx}\ncrlnumber={W/'crlnumber'}\ndefault_md=sha256\ndefault_crl_days={e['dias_crl']}\nprivate_key={F/'ca.key'}\ncertificate={F/'ca.pem'}\n")
    tmp = T / 'admin.crl.pem.nueva'
    if tmp.exists(): tmp.unlink()
    osl('ca', '-gencrl', '-config', cnf, '-out', tmp)
    os.chmod(tmp, 0o600); os.replace(tmp, T / 'admin.crl.pem')

if __name__ == '__main__':
    main()
