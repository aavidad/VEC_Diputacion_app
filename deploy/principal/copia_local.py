#!/usr/bin/env python3
"""Copia desechable local; nunca restaura credenciales globales."""
import argparse
import fcntl
import hashlib
import http.client
import json
import os
from pathlib import Path
import re
import secrets
import shutil
import socket
import ssl
import stat
import subprocess
import sys
import tempfile
import time

REPO = Path(__file__).resolve().parents[2]
BASE = Path('/dev/shm/vec/pg')
IMAGEN = 'postgres:18.4'
REFERENCIA = '44218865a'
FUENTES = {
    'principal-hz37-20261014.dump': '7ae0613fa174aebc8aa36c079a6c4b268b771df41c9310129ee9dfae9650cc45',
    'acl-base-20261014.sql': 'ab21b92e6f81a838c8c502d431de66454e8c47e7d7fca0becc01882d6d3584aa',
    'acl-tipos-20261014.sql': 'e058a00b9e0c8c9f06c2fc76d33882637ab7fb715618282305237c1ea959ea71',
}


def sha256(ruta):
    with ruta.open('rb') as fuente:
        return hashlib.file_digest(fuente, 'sha256').hexdigest()


def ejecutar(args, *, entrada=None, salida=None, entorno=None, cwd=None, descriptores=()):
    return subprocess.run([str(a) for a in args], input=entrada, stdout=salida,
                          stderr=subprocess.STDOUT if salida else subprocess.PIPE,
                          env=entorno, cwd=cwd, pass_fds=descriptores, check=True, timeout=600)


def guardar(ruta, valor):
    descriptor, nombre = tempfile.mkstemp(dir=ruta.parent, prefix='.estado-')
    temporal = Path(nombre)
    try:
        with os.fdopen(descriptor, 'w') as salida:
            salida.write(json.dumps(valor, ensure_ascii=False, indent=2) + '\n')
            salida.flush()
            os.fsync(salida.fileno())
        os.replace(temporal, ruta)
    finally:
        temporal.unlink(missing_ok=True)


def memoria():
    disponible = int(re.search(r'MemAvailable:\s+(\d+)', Path('/proc/meminfo').read_text())[1])
    if disponible < 34 * 1024**2:
        raise ValueError('Menos de 34 GiB disponibles (32 para el sistema y 2 para la copia); retire solo sus copias sin uso y avise en el canal')


def puerto_libre(puerto):
    # Un puerto en TIME-WAIT tras parar la aplicación está libre; solo falla si alguien escucha.
    with socket.socket() as s:
        s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        s.bind(('127.0.0.1', puerto))


class Copia:
    def __init__(self, equipo, instancia):
        self.nombre = f'vec-copia-{equipo}-{instancia}'
        self.raiz = BASE / f'{equipo}-{instancia}'
        if self.raiz.resolve() != self.raiz or self.raiz.is_symlink():
            raise ValueError('Destino enlazado o fuera de la raíz de copias')
        self.estado = self.raiz / 'estado.json'
        self.material = self.raiz / 'material'

    def leer(self):
        estado = json.loads(self.estado.read_text())
        if (estado.get('nombre') != self.nombre or estado.get('raiz') != str(self.raiz)
                or estado.get('fuente') != str(REPO)):
            raise ValueError('La marca de propiedad no corresponde a esta copia')
        return estado

    def docker(self, *args, **kwargs):
        return ejecutar(['docker', *args], **kwargs)

    def existe(self, entorno=None):
        r = subprocess.run(['docker', 'container', 'inspect', self.nombre],
                           capture_output=True, check=False, env=entorno)
        if r.returncode:
            # Diferenciar ausencia de contenedor de un motor inaccesible.
            self.docker('info', salida=subprocess.DEVNULL, entorno=entorno)
            return False
        etiqueta = (json.loads(r.stdout)[0]['Config']['Labels'] or {}).get('vec.copia')
        if etiqueta != str(self.raiz):
            raise ValueError('Contenedor ajeno con el mismo nombre')
        return True

    def pg(self):
        estado = self.leer()
        self.validar_pgdata()
        if self.existe():
            return
        puerto_libre(estado['puerto_pg'])
        self.docker('run', '--pull=never', '-d', '--rm', '--restart=no', '--memory=2g', '--cpus=2',
                    '--pids-limit=128', '--name', self.nombre,
                    '--label', f'vec.copia={self.raiz}',
                    '-p', f"127.0.0.1:{estado['puerto_pg']}:5432",
                    '-v', f'{self.raiz}/pgdata:/var/lib/postgresql',
                    '-v', f'{self.raiz}/postgres.password:/run/vec-password',
                    '-e', 'POSTGRES_PASSWORD_FILE=/run/vec-password',
                    '--entrypoint', 'sh', IMAGEN, '-ec',
                    'chown postgres:postgres /run/vec-password; '
                    'chmod 600 /run/vec-password; exec docker-entrypoint.sh postgres',
                    salida=subprocess.DEVNULL)
        for _ in range(120):
            if not self.existe():
                raise ValueError('El contenedor terminó antes de estar disponible')
            r = subprocess.run(['docker', 'exec', '-u', 'postgres', self.nombre,
                                'pg_isready', '-U', 'postgres'], capture_output=True)
            if r.returncode == 0:
                return
            time.sleep(.5)
        raise ValueError('PostgreSQL no respondió en 60 segundos')

    def validar_pgdata(self):
        ruta = self.raiz / 'pgdata'
        if ruta.resolve() != ruta or not ruta.is_dir():
            raise ValueError('PGDATA enlazado o ausente; se conserva la copia')

    def sql(self, fichero, log):
        self.sql_bytes(Path(fichero).read_bytes(), log)

    def sql_bytes(self, contenido, log):
        with (self.raiz / log).open('ab') as destino:
            subprocess.run(['docker', 'exec', '-i', '-u', 'postgres', self.nombre,
                            'psql', '-X', '-v', 'ON_ERROR_STOP=1', '-U', 'postgres',
                            '-d', 'postgres'], input=contenido, stdout=destino,
                           stderr=subprocess.STDOUT, check=True, timeout=300)

    def registrar_sid(self):
        """Conserva el identificador PostgreSQL comprobado de esta copia propia."""
        entorno = {'PATH': '/usr/bin:/bin', 'LANG': 'C',
                   'DOCKER_HOST': 'unix:///var/run/docker.sock'}
        if not self.existe(entorno=entorno):
            raise ValueError('El contenedor propio no está disponible para cotejar el SID')
        resultado = subprocess.run(
            ['docker', 'exec', '-i', '-u', 'postgres', self.nombre,
             'psql', '-X', '-At', '-v', 'ON_ERROR_STOP=1', '-U', 'postgres',
             '-d', 'postgres', '-c',
             'SELECT system_identifier::text FROM pg_catalog.pg_control_system();'],
            capture_output=True, text=True, check=True, timeout=30, env=entorno)
        sid = resultado.stdout.strip()
        if not re.fullmatch(r'[0-9]{15,20}', sid) or int(sid) == 0:
            raise ValueError('SID PostgreSQL inválido en la copia propia')
        estado = self.leer()
        anterior = estado.get('sid_copia')
        if anterior is not None:
            if anterior != sid:
                raise ValueError('SID PostgreSQL distinto del registrado para la copia')
            return sid
        estado['sid_copia'] = sid
        guardar(self.estado, estado)
        return sid

    def _preparar_software(self, origen, sha_esperada, carpeta_nombre, binario_nombre, validar):
        """Copia bytes declarados a un archivo propio antes de sondear el ejecutable."""
        from recorrido_copia.verificador_fuente import LIMITE_BINARIO
        origen = Path(origen)
        if sha_esperada is not None and not re.fullmatch(r'[0-9a-f]{64}', sha_esperada):
            raise ValueError('Huella declarada del software inválida')
        if not origen.is_absolute() or origen.is_symlink() or origen.resolve(strict=True) != origen:
            raise ValueError('Fuente del software ausente o enlazada')
        estado = origen.lstat()
        if (not stat.S_ISREG(estado.st_mode) or estado.st_uid not in (0, os.getuid())
                or estado.st_mode & 0o002 or not os.access(origen, os.X_OK)
                or estado.st_nlink != 1 or not 0 < estado.st_size <= LIMITE_BINARIO
                or (estado.st_mode & 0o020 and sha_esperada is None)):
            raise ValueError('Fuente del software insegura o sin huella declarada')
        carpeta = self.raiz / carpeta_nombre
        carpeta.mkdir(mode=0o700, exist_ok=True)
        directorio = carpeta.lstat()
        if (not stat.S_ISDIR(directorio.st_mode) or carpeta.is_symlink()
                or directorio.st_uid != os.getuid()
                or stat.S_IMODE(directorio.st_mode) != 0o700):
            raise ValueError('Carpeta privada del software insegura')
        temporal, destino = carpeta / (binario_nombre + '.nuevo'), carpeta / binario_nombre
        if destino.exists() or destino.is_symlink():
            raise ValueError('Software propio ya preparado')
        fd_origen = os.open(origen, os.O_RDONLY | os.O_NOFOLLOW | os.O_CLOEXEC)
        try:
            with os.fdopen(fd_origen, 'rb') as entrada:
                abierto = os.fstat(entrada.fileno())
                if (abierto.st_dev, abierto.st_ino, abierto.st_size) != (estado.st_dev, estado.st_ino, estado.st_size):
                    raise ValueError('Fuente del software cambió durante la apertura')
                fd_salida = os.open(temporal, os.O_CREAT | os.O_EXCL | os.O_WRONLY | os.O_NOFOLLOW, 0o700)
                huella = hashlib.sha256()
                total = 0
                with os.fdopen(fd_salida, 'wb') as salida:
                    while bloque := entrada.read(1_048_576):
                        total += len(bloque)
                        if total > LIMITE_BINARIO:
                            raise ValueError('Fuente del software excesiva')
                        huella.update(bloque)
                        salida.write(bloque)
                    salida.flush()
                    os.fsync(salida.fileno())
            despues = origen.lstat()
            if (total != estado.st_size or
                    (despues.st_dev, despues.st_ino, despues.st_size, despues.st_mtime_ns)
                    != (estado.st_dev, estado.st_ino, estado.st_size, estado.st_mtime_ns)):
                raise ValueError('Fuente del software cambió durante la copia')
            if sha_esperada is not None and huella.hexdigest() != sha_esperada:
                raise ValueError('Huella declarada del software divergente')
            resultado = validar(temporal, sha_esperada)
            if resultado['binario_sha256'] != huella.hexdigest():
                raise ValueError('Copia del software divergente')
            temporal.rename(destino)
            return {'binario': str(destino), 'binario_sha256': huella.hexdigest(),
                    'version': resultado['version']}
        except Exception:
            temporal.unlink(missing_ok=True)
            raise

    def preparar_verificador(self, origen, sha_esperada):
        from recorrido_copia.verificador_fuente import validar
        return self._preparar_software(origen, sha_esperada,
                                      'software-verificador', 'grxfirma', validar)

    def preparar_afirmauri(self, origen, sha_esperada):
        from recorrido_copia.afirmauri_fuente import validar
        return self._preparar_software(origen, sha_esperada,
                                      'software-afirmauri', 'grxfirma-afirmauri', validar)

    def crear(self, args):
        verificar_fuente()
        if args.rrhh_original:
            from copia_local_rrhh import validar
            validar(args.rrhh_original, REPO, BASE)
        memoria()
        puerto_libre(args.puerto_pg)
        puerto_libre(args.puerto_app)
        if args.modulos:
            puerto_libre(args.puerto_smtp)
        if getattr(args, 'recorrido_completo', False):
            puerto_libre(args.puerto_verificador)
        completo = getattr(args, 'recorrido_completo', False)
        datos = validar_ejecutor_privado('VEC_COPIA_DATOS_SCRIPT') if args.modulos and not completo else None
        roles = validar_ejecutor_privado('VEC_COPIA_ROLES_SCRIPT') if args.modulos and not completo else None
        portal = REPO / 'deploy/principal/portal_copia' / (
            'preparar_clasificacion_externa.py' if args.sin_inscripcion else 'preparar_portal.py')
        admin = validar_ejecutor_privado('VEC_COPIA_ADMIN_SCRIPT') if args.administracion and not completo else None
        if admin or completo:
            puerto_libre(args.puerto_admin)
        origen = Path(os.environ['VEC_COPIA_ORIGEN']).resolve(strict=True)
        dump = origen / 'principal-hz37-20261014.dump'
        acl = [origen / 'acl-base-20261014.sql', origen / 'acl-tipos-20261014.sql']
        for ruta in [dump, *acl]:
            if not ruta.is_file():
                raise ValueError('Falta el volcado o sus permisos de base y tipos')
            if sha256(ruta) != FUENTES[ruta.name]:
                raise ValueError('El volcado y los anexos deben corresponder a la misma referencia HZ37')
        listas = migraciones(args.lista_sql)
        self.raiz.mkdir(mode=0o700, parents=True, exist_ok=False)
        software = (self.preparar_verificador(args.binario_verificador, args.sha_verificador)
                    if completo else None)
        afirmauri_origen = getattr(args, 'binario_afirmauri', None)
        afirmauri = (self.preparar_afirmauri(afirmauri_origen, getattr(args, 'sha_afirmauri', None))
                     if completo and afirmauri_origen is not None else None)
        estado_inicial = dict(nombre=self.nombre, raiz=str(self.raiz),
                puerto_pg=args.puerto_pg, puerto_app=args.puerto_app,
                puerto_smtp=args.puerto_smtp if args.modulos else None,
                fuente=str(REPO), origen=str(origen),
                preparada=False, revision=revision(), creada_en=time.time(),
                material_modo='rrhh_original' if args.rrhh_original else 'nuevo')
        if software is not None:
            estado_inicial['software_verificador'] = software
        if completo:
            estado_inicial['software_afirmauri'] = afirmauri
        guardar(self.estado, estado_inicial)
        guardar(self.raiz / 'fuentes.json', FUENTES)
        self.pg_password()
        self.pg()
        with dump.open('rb') as fuente, (self.raiz / 'esquema.sql').open('wb') as destino:
            subprocess.run(['docker', 'exec', '-i', '-u', 'postgres', self.nombre,
                            'pg_restore', '--schema-only', '--file=-'], stdin=fuente, stdout=destino,
                           stderr=subprocess.PIPE, check=True, timeout=120)
        fuentes_roles = sorted((REPO / 'deploy/postgresql').glob('*/roles*_up.sql'))
        preparar_roles = [sys.executable, REPO / 'deploy/principal/copia_local_roles.py',
                          self.raiz / 'esquema.sql', *acl, '--roles-b119']
        for fuente in fuentes_roles:
            preparar_roles.extend(('--fuente-roles', fuente))
        for ruta, _, _ in listas:
            preparar_roles.extend(('--migracion', ruta))
        with (self.raiz / 'roles.sql').open('wb') as destino:
            ejecutar(preparar_roles, salida=destino)
        self.sql(self.raiz / 'roles.sql', 'restauracion.log')
        with dump.open('rb') as fuente, (self.raiz / 'restauracion.log').open('ab') as destino:
            subprocess.run(['docker', 'exec', '-i', '-u', 'postgres', self.nombre,
                            'pg_restore', '-U', 'postgres', '-d', 'postgres',
                            '--exit-on-error'], stdin=fuente, stdout=destino,
                           stderr=subprocess.STDOUT, check=True, timeout=300)
        for ruta in acl:
            self.sql(ruta, 'restauracion.log')
        self.comprobar_acl(acl, permitir_nuevas=False)
        self.registrar_sid()
        estado = self.leer()
        estado['recorrido_completo'] = completo
        guardar(self.estado, estado)
        if completo:
            from recorrido_copia import preparar_roles as roles_versionados
            roles_versionados(self)
        elif args.modulos:
            with (self.raiz / 'roles-preparacion.log').open('ab') as log:
                ejecutar([sys.executable, roles, '--raiz', self.raiz], salida=log)
        diario = []
        for ruta, huella, contenido in listas:
            self.sql_bytes(contenido, 'migraciones.log')
            diario.append(dict(ruta=str(ruta.relative_to(REPO)), sha256=huella))
            guardar(self.raiz / 'sql_aplicado.json', diario)
        self.comprobar_acl(acl, permitir_nuevas=True)
        if args.rrhh_original:
            from copia_local_rrhh import preparar
            preparar(args.rrhh_original, self.material, REPO, BASE)
        else:
            with (self.raiz / 'material.log').open('wb') as log:
                ejecutar([REPO / 'scripts/generar_credenciales_desarrollo.sh', self.material],
                          salida=log, entorno={**os.environ, 'VEC_DESARROLLO_VIGENCIA_DIAS': '400' if completo else '90'})
        from copia_local_ambito_bolsas import declarar as declarar_ambito_bolsas
        declarar_ambito_bolsas(self)
        self.tls_pg()
        if args.modulos:
            from copia_local_conexiones import preparar
            entorno_modulos = preparar(self)
            guardar(self.raiz / 'entorno_modulos.json', entorno_modulos)
            estado = self.leer()
            estado['entorno_modulos_sha256'] = sha256(self.raiz / 'entorno_modulos.json')
            guardar(self.estado, estado)
            if completo:
                from recorrido_copia import preparar_datos
                preparar_datos(self)
            else:
                with (self.raiz / 'datos-preparacion.log').open('ab') as log:
                    ejecutar([sys.executable, datos, '--raiz', self.raiz], salida=log)
            from copia_local_cancelacion import encender as encender_cancelacion
            encender_cancelacion(self, guardar, sha256)
            from copia_local_verificacion import comprobar as comprobar_conexiones
            comprobar_conexiones(self)
            if not args.portal_externo:
                # F1-99: datos del caso 4.0 e identidad de Mi Bolsa de Antonio.
                import copia_local_caso40 as caso40
                caso40.aplicar_guiones(self, REPO)
                entorno_modulos = json.loads((self.raiz / 'entorno_modulos.json').read_text())
                entorno_modulos = caso40.entorno_portal(entorno_modulos, caso40.preparar_antonio(self, REPO))
                guardar(self.raiz / 'entorno_modulos.json', entorno_modulos)
                estado = self.leer()
                estado['entorno_modulos_sha256'] = sha256(self.raiz / 'entorno_modulos.json')
                guardar(self.estado, estado)
                # F1-113: Mis preferencias y Mi imagen, ya con Antonio como aspirante.
                from copia_local_usuarios import preparar as preparar_usuarios
                preparar_usuarios(self, args.rrhh_original, guardar, sha256)
        if args.segundo_rrhh:
            from copia_local_rrhh_adicional import preparar_copia
            huella_semilla = preparar_copia(self)
            estado = self.leer()
            estado['rrhh_adicional_semilla_sha256'] = huella_semilla
            guardar(self.estado, estado)
        if completo:
            from recorrido_copia.centros import preparar as preparar_centros
            centros = preparar_centros(self)
            archivo = self.raiz / 'entorno_modulos.json'
            entorno = json.loads(archivo.read_bytes())
            entorno.update(centros['entorno'])
            guardar(archivo, entorno)
            estado = self.leer()
            estado['entorno_modulos_sha256'] = sha256(archivo)
            guardar(self.estado, estado)
        memoria()
        verificar_fuente(self.leer()['revision'])
        go = subprocess.check_output([REPO / 'scripts/seleccionar_toolchain_go_local.sh'], text=True).strip()
        entorno_go = {**os.environ, 'GOTOOLCHAIN': 'local', 'GOPROXY': 'off',
                      'GOSUMDB': 'off', 'GOENV': 'off', 'GOCACHE': '/dev/shm/vec/go-build',
                      'GOMAXPROCS': '2', 'GOMEMLIMIT': '512MiB'}
        modcache = subprocess.check_output([go, 'env', 'GOMODCACHE'],
                                          env=entorno_go, text=True).strip()
        guardar(self.raiz / 'compilacion.json', dict(go=go, gomodcache=modcache,
                gocache=entorno_go['GOCACHE'], revision=revision()))
        with (self.raiz / 'compilacion.log').open('wb') as log:
            ejecutar([go, 'build', '-p', '2', '-o', self.raiz / 'vec-server', './cmd/vec-server'],
                      cwd=REPO, salida=log, entorno=entorno_go)
        estado = self.leer()
        verificar_fuente(estado['revision'])
        estado['binario_sha256'] = sha256(self.raiz / 'vec-server')
        estado['preparada'] = True
        guardar(self.estado, estado)
        self.arrancar()
        if args.modulos and not args.portal_externo:
            caso40.registrar_contacto(self, REPO)
        if admin or completo:
            from copia_local_administracion import ejecutar as ejecutar_admin, huella_ejecutor
            estado = self.leer()
            estado['administracion'] = (dict(versionado=True, puerto=args.puerto_admin,
                catalogo_entrada_unidad='administracion_f131_' + hashlib.sha256(self.nombre.encode()).hexdigest()[:16])
                if completo else dict(ejecutor=str(admin), sha256=huella_ejecutor(admin), puerto=args.puerto_admin))
            guardar(self.estado, estado)
            ejecutar_admin(self, 'preparar')
            # Solo una preparación confirmada habilita el arranque en cada reinicio.
            estado = self.leer()
            estado['admin_preparada'] = True
            guardar(self.estado, estado)
            ejecutar_admin(self, 'arrancar')
            ejecutar_admin(self, 'comprobar')
        if args.portal_externo:
            # La app mantiene el sellado real mientras el productor provisiona
            # identidad, contexto y confianza. El reinicio carga su entorno.
            privado_portal = self.raiz / 'portal'
            privado_portal.mkdir(mode=0o700)
            verificar_fuente(self.leer()['revision'])
            with (self.raiz / 'portal-preparacion.log').open('ab') as log:
                ejecutar([sys.executable, portal, '--raiz', self.raiz,
                          '--candado-fd', str(self.candado)], salida=log,
                         descriptores=(self.candado,), entorno={**os.environ,
                         'VEC_COPIA_PORTAL_ESTADO': str(privado_portal)})
            comprobar_conexiones(self)
            # F1-113: Mis preferencias y Mi imagen, ya con el aspirante del portal.
            from copia_local_usuarios import preparar as preparar_usuarios
            preparar_usuarios(self, args.rrhh_original, guardar, sha256)
            estado = self.leer()
            estado['portal_externo_preparado'] = True
            guardar(self.estado, estado)
            self.parar()
            self.arrancar()
        if completo:
            from recorrido_copia.conductor import preparar as preparar_recorrido
            preparar_recorrido(self, args)
        if args.modulos:
            estado = self.leer()
            estado['comprobar_modulos'] = True
            guardar(self.estado, estado)
            self.comprobar()
        estado = self.leer()
        estado['segundos_creacion'] = round(time.time() - estado['creada_en'], 2)
        guardar(self.estado, estado)
        print(f"Copia disponible en https://localhost:{args.puerto_app}/portal-empleado/ ({estado['segundos_creacion']} s)")

    def comprobar_acl(self, anexos, permitir_nuevas):
        comprobacion = self.raiz / 'comprobar_acl.sql'
        args = [sys.executable, REPO / 'deploy/principal/copia_local_acl.py', *anexos]
        if permitir_nuevas:
            args.append('--permitir-nuevas-concesiones')
        with comprobacion.open('wb') as salida:
            ejecutar(args, salida=salida)
        self.sql(comprobacion, 'comprobar_acl.log')

    def pg_password(self):
        # postgres (UID del contenedor) debe atravesar el punto de montaje;
        # la raíz de instancia 0700 impide acceso desde otras cuentas del host.
        (self.raiz / 'pgdata').mkdir(mode=0o755)
        (self.raiz / 'pgdata').chmod(0o755)
        password = self.raiz / 'postgres.password'
        password.write_text(secrets.token_hex(32) + '\n')
        # El entrypoint asigna el fichero al usuario postgres antes de leerlo.
        password.chmod(0o600)

    def tls_pg(self):
        for origen, nombre in [('tls/servidor.crt', 'server.crt'), ('tls/servidor.key', 'server.key')]:
            self.docker('cp', str(self.material / origen), f'{self.nombre}:/var/lib/postgresql/18/docker/{nombre}', salida=subprocess.DEVNULL)
        self.docker('exec', '-u', '0', self.nombre, 'chown', 'postgres:postgres',
                    '/var/lib/postgresql/18/docker/server.crt', '/var/lib/postgresql/18/docker/server.key')
        self.docker('exec', '-u', '0', self.nombre, 'chmod', '600', '/var/lib/postgresql/18/docker/server.key')
        # Persistido en PGDATA: lo conserva la recreación del contenedor --rm.
        config = self.raiz / 'tls.sql'
        config.write_text("ALTER SYSTEM SET ssl='on';\nALTER SYSTEM SET ssl_cert_file='server.crt';\nALTER SYSTEM SET ssl_key_file='server.key';\nSELECT pg_reload_conf();\n")
        self.sql(config, 'tls.log')

    def arrancar(self, comprobar_modulos=True):
        estado = self.leer()
        if not estado['preparada'] or estado['revision'] != revision():
            raise ValueError('Copia incompleta o revisión distinta; use una instancia nueva')
        verificar_fuente(estado['revision'])
        self.validar_binario(estado)
        self.pg()
        # BC14: en cada arranque, ya con la lista y el kit de datos aplicados,
        # declara los plazos de inscripción que se hayan abierto. Idempotente.
        from copia_local_publicacion_bases import declarar as declarar_publicacion_bases
        declarar_publicacion_bases(self)
        if self.proceso(estado):
            raise ValueError('La aplicación propia ya está activa')
        puerto_libre(estado['puerto_app'])
        entorno = dict(PATH=os.environ['PATH'], LANG='C.UTF-8',
            VEC_EXECUTION_PROFILE='desarrollo', VEC_AUTH_MODE='desarrollo',
            VEC_DEVELOPMENT_GUARD='ACEPTO_CREDENCIALES_NO_AUTORITATIVAS_SOLO_DESARROLLO',
            VEC_DEVELOPMENT_MATERIAL_DIR=str(self.material),
            VEC_TLS_CERT_FILE=str(self.material / 'tls/servidor.crt'),
            VEC_TLS_KEY_FILE=str(self.material / 'tls/servidor.key'),
            VEC_GOBIERNO_ADMIN_VALIDEZ_HORAS='2136',
            VEC_HTTP_ADDR=f"127.0.0.1:{estado['puerto_app']}",
            VEC_BOLSA_DATA_DIR=str(self.raiz / 'datos'))
        entorno.update(self.entorno_modulos())
        from copia_local_correo import arrancar as arrancar_correo
        entorno.update(arrancar_correo(self))
        from copia_local_firma import arrancar_verificador
        arrancar_verificador(self, guardar, puerto_libre)
        estado = self.leer()
        with (self.raiz / 'app.log').open('ab') as log:
            proceso = subprocess.Popen([self.raiz / 'vec-server'], cwd=REPO, env=entorno,
                        stdout=log, stderr=subprocess.STDOUT, start_new_session=True)
        estado['pid'] = proceso.pid
        estado['inicio_proceso'] = Path(f'/proc/{proceso.pid}/stat').read_text().split()[21]
        guardar(self.estado, estado)
        for _ in range(120):
            if proceso.poll() is not None:
                raise ValueError('La aplicación terminó; consulte app.log en su copia')
            try:
                self.comprobar(comprobar_admin=False, comprobar_modulos=comprobar_modulos)
            except (OSError, ValueError):
                time.sleep(.5)
                continue
            if estado.get('admin_preparada'):
                from copia_local_administracion import ejecutar as ejecutar_admin
                ejecutar_admin(self, 'arrancar')
                ejecutar_admin(self, 'comprobar')
            return
        raise ValueError('La aplicación no respondió en 60 segundos')

    def entorno_modulos(self):
        estado = self.leer()
        huella = estado.get('entorno_modulos_sha256')
        if not huella:
            return {}
        ruta = self.raiz / 'entorno_modulos.json'
        info = ruta.lstat()
        if (not stat.S_ISREG(info.st_mode) or info.st_uid != os.getuid()
                or info.st_mode & 0o077 or info.st_nlink != 1
                or sha256(ruta) != huella):
            raise ValueError('La configuración de módulos cambió; se conserva la copia parada')
        entorno = json.loads(ruta.read_text())
        if (not isinstance(entorno, dict) or not entorno
                or any(not re.fullmatch('VEC_[A-Z0-9_]+', k)
                       or not isinstance(v, str) or '\x00' in v
                       for k, v in entorno.items())):
            raise ValueError('Configuración de módulos inválida')
        return entorno

    def validar_binario(self, estado):
        ruta = self.raiz / 'vec-server'
        datos = ruta.lstat()
        if (not stat.S_ISREG(datos.st_mode) or datos.st_uid != os.getuid()
                or datos.st_mode & 0o077 or datos.st_nlink != 1
                or sha256(ruta) != estado.get('binario_sha256')):
            raise ValueError('El binario no coincide con el compilado para esta copia')


    def proceso(self, estado):
        pid = estado.get('pid')
        if not pid:
            return False
        proc = Path(f'/proc/{pid}')
        try:
            return (proc / 'exe').resolve() == self.raiz / 'vec-server' and (proc / 'stat').read_text().split()[21] == estado['inicio_proceso']
        except FileNotFoundError:
            return False

    def parar(self):
        estado = self.leer()
        from copia_local_administracion import parar as parar_admin
        parar_admin(self)
        if self.proceso(estado):
            os.kill(estado['pid'], 15)
            for _ in range(100):
                if not self.proceso(estado):
                    break
                time.sleep(.1)
            else:
                raise ValueError('La aplicación sigue activa; se conserva su copia')
        if self.existe():
            self.docker('stop', '--time', '20', self.nombre, salida=subprocess.DEVNULL)
        from copia_local_correo import parar as parar_correo
        parar_correo(self)
        from copia_local_firma import parar_verificador
        parar_verificador(self)

    def personal(self, configuracion):
        from copia_local_personal import preparar
        estado = self.leer()
        verificar_fuente(estado['revision'])
        memoria()
        go = subprocess.check_output([REPO / 'scripts/seleccionar_toolchain_go_local.sh'], text=True).strip()
        entorno_go = {**os.environ, 'GOTOOLCHAIN': 'local', 'GOPROXY': 'off',
                      'GOSUMDB': 'off', 'GOENV': 'off', 'GOCACHE': '/dev/shm/vec/go-build',
                      'GOMAXPROCS': '2', 'GOMEMLIMIT': '512MiB'}
        preparar(self, configuracion, repo=REPO, go=go, entorno_go=entorno_go, guardar=guardar)

    def firma(self, manifiesto):
        """Compone la firma de una copia con módulos y la reinicia con el verificador."""
        from copia_local_firma import preparar
        estado = self.leer()
        if not estado.get('entorno_modulos_sha256'):
            raise ValueError('La firma necesita una copia creada con --modulos')
        verificar_fuente(estado['revision'])
        # Las etapas privadas pueden necesitar la aplicación; la v4 solo PostgreSQL.
        self.pg()
        memoria()
        go = subprocess.check_output([REPO / 'scripts/seleccionar_toolchain_go_local.sh'], text=True).strip()
        entorno_go = {**os.environ, 'GOTOOLCHAIN': 'local', 'GOPROXY': 'off',
                      'GOSUMDB': 'off', 'GOENV': 'off', 'GOCACHE': '/dev/shm/vec/go-build',
                      'GOMAXPROCS': '2', 'GOMEMLIMIT': '512MiB'}
        preparar(self, manifiesto, repo=REPO, go=go, entorno_go=entorno_go, guardar=guardar)
        self.parar()
        self.arrancar()

    def rol_aspirante(self):
        """Publica el rol del aspirante v3 por actos de Administración (F1-115)."""
        from copia_local_rol_aspirante import publicar
        estado = self.leer()
        verificar_fuente(estado['revision'])
        memoria()
        go = subprocess.check_output([REPO / 'scripts/seleccionar_toolchain_go_local.sh'], text=True).strip()
        entorno_go = {**os.environ, 'GOTOOLCHAIN': 'local', 'GOPROXY': 'off',
                      'GOSUMDB': 'off', 'GOENV': 'off', 'GOCACHE': '/dev/shm/vec/go-build',
                      'GOMAXPROCS': '2', 'GOMEMLIMIT': '512MiB'}
        publicar(self, repo=REPO, go=go, entorno_go=entorno_go)

    def comprobar(self, comprobar_admin=True, comprobar_modulos=True):
        estado = self.leer()
        contexto = ssl.create_default_context(cafile=str(self.material / 'ca/ca.crt'))
        contexto.load_cert_chain(str(self.material / 'mtls/cliente.crt'), str(self.material / 'mtls/cliente.key'))
        rutas = [('/livez', None), ('/portal-empleado/', None)]
        if comprobar_modulos and estado.get('comprobar_modulos'):
            cuadro = dict(filtros=dict(texto='', estado_clave='', fase_clave=''),
                          paginacion=dict(limite=6, cursor=''))
            rutas.extend([('/api/vec/contratacion-temporal/cuadro/consultas', json.dumps(cuadro).encode()),
                          ('/api/vec/bolsa/bolsas', None)])
        origen = f"https://localhost:{estado['puerto_app']}"
        for ruta, datos in rutas:
            # urllib añade Connection: close, prohibida por la frontera de CT.
            conexion = http.client.HTTPSConnection('localhost', estado['puerto_app'],
                                                   context=contexto, timeout=5)
            try:
                conexion.request('POST' if datos is not None else 'GET', ruta, body=datos,
                                 headers={'Origin': origen, 'Content-Type': 'application/json',
                                          'Accept': 'application/json', 'Sec-Fetch-Site': 'same-origin',
                                          'Sec-Fetch-Mode': 'cors', 'Sec-Fetch-Dest': 'empty'})
                r = conexion.getresponse()
                r.read()
                if r.status != 200:
                    raise ValueError(f'Respuesta HTTP {r.status} en {ruta}')
            finally:
                conexion.close()
        if comprobar_admin and estado.get('admin_preparada'):
            from copia_local_administracion import ejecutar as ejecutar_admin
            ejecutar_admin(self, 'comprobar')

    def exportar(self, archivo):
        destino = Path(archivo)
        estado = self.leer()
        if (not destino.is_absolute() or destino.resolve() != destino or destino.exists()
                or destino.is_relative_to(REPO) or destino.is_relative_to(self.raiz)
                or destino.is_relative_to(Path(estado['origen']))):
            raise ValueError('El clon debe quedar fuera del repositorio y de la instancia, sin enlaces')
        self.validar_pgdata()
        self.parar()
        descriptor, nombre = tempfile.mkstemp(dir=destino.parent, prefix='.vec-clon-')
        temporal = Path(nombre)
        try:
            with os.fdopen(descriptor, 'wb') as salida:
                self.docker('run', '--pull=never', '--rm', '--network=none', '--memory=256m', '--cpus=2',
                            '--pids-limit=32', '-v', f'{self.raiz}/pgdata:/copia:ro',
                            '--entrypoint', 'tar', IMAGEN, '--owner=999', '--group=999',
                            '-C', '/copia', '-czf', '-', '.', salida=salida)
            # Publicar solo al terminar; link rechaza un destino creado concurrentemente.
            os.link(temporal, destino)
        finally:
            temporal.unlink(missing_ok=True)
        print('Clon frío exportado; la instancia queda parada. Use VEC_CLON_ESTADO para el ensayo.')


def revision():
    return subprocess.check_output(['git', '-C', str(REPO), 'rev-parse', 'HEAD'], text=True).strip()


def verificar_fuente(esperada=None):
    estado = subprocess.check_output(['git', '-C', str(REPO), 'status', '--porcelain',
                                     '--untracked-files=all'])
    if estado or (esperada is not None and revision() != esperada):
        raise ValueError('La compilación requiere la revisión indicada y el árbol limpio')


def validar_ejecutor_privado(variable):
    valor = os.environ.get(variable)
    if not valor:
        raise ValueError(f'Falta {variable}: preparación de datos de su copia')
    ruta = Path(valor)
    dato = ruta.lstat()
    if (not ruta.is_absolute() or ruta.resolve(strict=True) != ruta
            or ruta.is_relative_to(REPO) or ruta.is_relative_to(BASE)
            or not stat.S_ISREG(dato.st_mode) or dato.st_uid != os.getuid()
            or dato.st_nlink != 1 or dato.st_mode & 0o077):
        raise ValueError('El preparador de datos debe ser un fichero privado fuera de Git y de las copias')
    return ruta


def migraciones(listas):
    resultado, vistas = [], set()
    nuevas = set()
    if listas:
        nuevas = set(subprocess.check_output(['git', '-C', str(REPO), 'diff', '--name-only',
                     '--diff-filter=A', REFERENCIA, 'HEAD', '--', 'deploy/postgresql'], text=True).splitlines())
    for lista in listas:
        for linea in Path(lista).read_text().splitlines():
            campos = linea.split('#', 1)[0].split()
            if not campos:
                continue
            ruta = (REPO / campos[0]).resolve(strict=True)
            if len(campos) > 2 or not ruta.is_relative_to(REPO / 'deploy/postgresql') or not ruta.name.endswith('.up.sql'):
                raise ValueError('Lista SQL inválida: solo migraciones UP dentro del repositorio')
            if ruta in vistas:
                raise ValueError('Migración repetida entre listas; no se aplicó ninguna')
            if str(ruta.relative_to(REPO)) not in nuevas:
                raise ValueError('La SQL ya existía en el corte de referencia o no está versionada en esta cabeza')
            vistas.add(ruta)
            contenido = ruta.read_bytes()
            versionado = subprocess.check_output(['git', '-C', str(REPO), 'show',
                            f'HEAD:{ruta.relative_to(REPO)}'])
            if contenido != versionado:
                raise ValueError('La SQL tiene cambios sin confirmar respecto a esta cabeza')
            huella = hashlib.sha256(contenido).hexdigest()
            if len(campos) == 2 and campos[1] != huella:
                raise ValueError('La huella de una migración no coincide')
            resultado.append((ruta, huella, contenido))
    return resultado


def main():
    os.umask(0o077)
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('orden', choices=['crear', 'arrancar', 'parar', 'reiniciar', 'comprobar', 'borrar', 'exportar-clon', 'firma',
                                     'personal', 'organizacion-historica', 'rol-aspirante'])
    p.add_argument('--equipo', required=True)
    p.add_argument('--instancia', required=True)
    p.add_argument('--puerto-pg', type=int, default=55481)
    p.add_argument('--puerto-app', type=int, default=18481)
    p.add_argument('--puerto-smtp', type=int, default=11081)
    p.add_argument('--lista-sql', action='append', default=[])
    p.add_argument('--recorrido-completo', action='store_true', help='Crear el recorrido completo con productores versionados')
    p.add_argument('--puerto-verificador', type=int, default=19881)
    p.add_argument('--binario-verificador', type=Path,
                   help='Ejecutable GrxFirma declarado para el recorrido completo')
    p.add_argument('--sha-verificador', help='SHA-256 del GrxFirma declarado')
    p.add_argument('--binario-afirmauri', type=Path,
                   help='Ejecutable público afirmauri declarado para el navegador')
    p.add_argument('--sha-afirmauri', help='SHA-256 del afirmauri declarado')
    p.add_argument('--modulos', action='store_true', help='Preparar conexiones nuevas de Bolsa y Contratación')
    p.add_argument('--rrhh-original', type=Path, help='Conservar el conjunto privado original de RRHH en una copia nueva')
    p.add_argument('--portal-externo', action='store_true', help='Preparar también el área del aspirante')
    p.add_argument('--administracion', action='store_true', help='Preparar también Administración con su ejecutor privado')
    p.add_argument('--segundo-rrhh', action='store_true', help='Preparar una segunda persona técnica de RRHH')
    p.add_argument('--puerto-admin', type=int, default=19481)
    p.add_argument('--sin-inscripcion', action='store_true', help='Preparar el portal sin activar inscripción')
    p.add_argument('--archivo-clon')
    p.add_argument('--manifiesto-firma', type=Path, help='Manifiesto privado de la orden firma (etapas y verificador)')
    p.add_argument('--config-personal', type=Path, help='Configuración B2 aprobada de referencia; solo se conservan referencias y motivos')
    args = p.parse_args()
    if args.recorrido_completo:
        if args.orden != 'crear' or args.rrhh_original:
            p.error('--recorrido-completo requiere crear con material nuevo')
        args.modulos = args.administracion = args.segundo_rrhh = True
        if not 1024 <= args.puerto_verificador <= 65535 or args.puerto_verificador in (
                args.puerto_pg, args.puerto_app, args.puerto_smtp, args.puerto_admin):
            p.error('El puerto del verificador debe ser distinto y estar entre 1024 y 65535')
        if args.binario_verificador is None or not args.binario_verificador.is_absolute():
            p.error('--recorrido-completo requiere --binario-verificador absoluto')
        if (args.binario_afirmauri is None) != (args.sha_afirmauri is None):
            p.error('--binario-afirmauri y --sha-afirmauri deben indicarse juntos')
        if args.binario_afirmauri is not None:
            if not args.binario_afirmauri.is_absolute():
                p.error('--binario-afirmauri requiere una ruta absoluta')
            if not re.fullmatch(r'[0-9a-f]{64}', args.sha_afirmauri):
                p.error('--sha-afirmauri debe ser una huella SHA-256 válida')
        if not args.lista_sql:
            args.lista_sql = [str(REPO / 'deploy/principal/lista_sql_main_tras_hz37.txt')]
    elif any(value is not None for value in (args.binario_verificador, args.sha_verificador,
                                            args.binario_afirmauri, args.sha_afirmauri)):
        p.error('El software declarado solo se admite con crear --recorrido-completo')
    if args.portal_externo and not args.modulos:
        p.error('--portal-externo requiere --modulos')
    if args.administracion and (args.orden != 'crear' or not args.modulos):
        p.error('--administracion solo se admite en crear --modulos')
    if args.segundo_rrhh and (args.orden != 'crear' or not args.modulos):
        p.error('--segundo-rrhh solo se admite en crear --modulos')
    if args.administracion and (not 1024 <= args.puerto_admin <= 65535
            or args.puerto_admin in (args.puerto_pg, args.puerto_app, args.puerto_smtp)):
        p.error('El puerto de Administración debe ser distinto y estar entre 1024 y 65535')
    if args.portal_externo and args.orden != 'crear':
        p.error('--portal-externo solo se admite en crear')
    if args.sin_inscripcion and not args.portal_externo:
        p.error('--sin-inscripcion requiere --portal-externo')
    if args.rrhh_original and (args.orden != 'crear' or not args.modulos):
        p.error('--rrhh-original solo se admite en crear --modulos')
    if not re.fullmatch('[a-z]', args.equipo) or not re.fullmatch('[a-z0-9][a-z0-9-]{0,40}', args.instancia):
        p.error('Equipo: una letra minúscula; instancia: letras, cifras y guiones')
    if not 1024 <= args.puerto_pg <= 65535 or not 1024 <= args.puerto_app <= 65535 or args.puerto_pg == args.puerto_app:
        p.error('Puertos distintos entre 1024 y 65535')
    if args.modulos and (not 1024 <= args.puerto_smtp <= 65535
                        or args.puerto_smtp in (args.puerto_pg, args.puerto_app)):
        p.error('El puerto SMTP debe ser distinto y estar entre 1024 y 65535')
    copia = Copia(args.equipo, args.instancia)
    for directorio in (BASE.parent, BASE):
        if not directorio.exists() and not directorio.is_symlink():
            directorio.mkdir(mode=0o700)
        datos = directorio.lstat()
        if (not stat.S_ISDIR(datos.st_mode) or datos.st_uid != os.getuid()
                or stat.S_IMODE(datos.st_mode) not in (0o700, 0o755)):
            raise ValueError('Directorio de copias local inseguro')
    # Candado por instancia fuera de PGDATA; borrar no permite otro escritor concurrente.
    with (BASE / f'.{copia.nombre}.lock').open('a') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        copia.candado = lock.fileno()
        if args.orden == 'crear':
            copia.crear(args)
        elif args.orden == 'reiniciar':
            copia.parar()
            copia.arrancar()
        elif args.orden == 'borrar':
            copia.parar()
            copia.validar_pgdata()
            copia.docker('run', '--pull=never', '--rm', '--network=none', '--memory=128m', '--pids-limit=32',
                         '-v', f'{copia.raiz}/pgdata:/copia', '--entrypoint', 'sh', IMAGEN,
                         '-c', 'find /copia -mindepth 1 -delete', salida=subprocess.DEVNULL)
            shutil.rmtree(copia.raiz)
        elif args.orden == 'organizacion-historica':
            from copia_local_organizacion_historica_activacion import activar
            activar(copia, REPO)
        elif args.orden == 'personal':
            if not args.config_personal or not args.config_personal.is_absolute():
                p.error('personal requiere --config-personal con ruta absoluta')
            copia.personal(args.config_personal)
        elif args.orden == 'firma':
            if not args.manifiesto_firma or not args.manifiesto_firma.is_absolute():
                p.error('firma requiere --manifiesto-firma con ruta absoluta')
            copia.firma(args.manifiesto_firma)
        elif args.orden == 'rol-aspirante':
            copia.rol_aspirante()
        elif args.orden == 'exportar-clon':
            if not args.archivo_clon:
                p.error('exportar-clon requiere --archivo-clon absoluto y nuevo')
            copia.exportar(args.archivo_clon)
        else:
            getattr(copia, args.orden)()


if __name__ == '__main__':
    try:
        main()
    except (ValueError, KeyError, OSError, subprocess.SubprocessError) as e:
        # Las excepciones de subprocess pueden contener órdenes con secretos: no imprimirlas.
        print(f'No se completó la copia: {e}' if isinstance(e, ValueError) else
              'No se completó la copia; consulte los registros de su instancia.', file=sys.stderr)
        sys.exit(1)
