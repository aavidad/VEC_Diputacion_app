#!/usr/bin/env python3
"""Copia desechable local; nunca restaura credenciales globales."""
import argparse
import fcntl
import hashlib
import json
import os
from pathlib import Path
import re
import secrets
import shutil
import socket
import ssl
import subprocess
import sys
import tempfile
import time
import urllib.request

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


def ejecutar(args, *, entrada=None, salida=None, entorno=None, cwd=None):
    return subprocess.run([str(a) for a in args], input=entrada, stdout=salida,
                          stderr=subprocess.STDOUT if salida else subprocess.PIPE,
                          env=entorno, cwd=cwd, check=True, timeout=600)


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
    with socket.socket() as s:
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

    def existe(self):
        r = subprocess.run(['docker', 'container', 'inspect', self.nombre],
                           capture_output=True, check=False)
        if r.returncode:
            # Diferenciar ausencia de contenedor de un motor inaccesible.
            self.docker('info', salida=subprocess.DEVNULL)
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
                    '-v', f'{self.raiz}/postgres.password:/run/vec-password:ro',
                    '-e', 'POSTGRES_PASSWORD_FILE=/run/vec-password',
                    IMAGEN, salida=subprocess.DEVNULL)
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

    def crear(self, args):
        memoria()
        puerto_libre(args.puerto_pg)
        puerto_libre(args.puerto_app)
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
        guardar(self.estado, dict(nombre=self.nombre, raiz=str(self.raiz),
                puerto_pg=args.puerto_pg, puerto_app=args.puerto_app,
                fuente=str(REPO), origen=str(origen),
                preparada=False, revision=revision(), creada_en=time.time()))
        guardar(self.raiz / 'fuentes.json', FUENTES)
        self.pg_password()
        self.pg()
        with dump.open('rb') as fuente, (self.raiz / 'esquema.sql').open('wb') as destino:
            subprocess.run(['docker', 'exec', '-i', '-u', 'postgres', self.nombre,
                            'pg_restore', '--schema-only', '--file=-'], stdin=fuente, stdout=destino,
                           stderr=subprocess.PIPE, check=True, timeout=120)
        with (self.raiz / 'roles.sql').open('wb') as destino:
            ejecutar([sys.executable, REPO / 'deploy/principal/copia_local_roles.py',
                      self.raiz / 'esquema.sql', *acl, '--roles-b119'], salida=destino)
        self.sql(self.raiz / 'roles.sql', 'restauracion.log')
        with dump.open('rb') as fuente, (self.raiz / 'restauracion.log').open('ab') as destino:
            subprocess.run(['docker', 'exec', '-i', '-u', 'postgres', self.nombre,
                            'pg_restore', '-U', 'postgres', '-d', 'postgres',
                            '--exit-on-error'], stdin=fuente, stdout=destino,
                           stderr=subprocess.STDOUT, check=True, timeout=300)
        for ruta in acl:
            self.sql(ruta, 'restauracion.log')
        self.comprobar_acl(acl, permitir_nuevas=False)
        diario = []
        for ruta, huella, contenido in listas:
            self.sql_bytes(contenido, 'migraciones.log')
            diario.append(dict(ruta=str(ruta.relative_to(REPO)), sha256=huella))
            guardar(self.raiz / 'sql_aplicado.json', diario)
        self.comprobar_acl(acl, permitir_nuevas=True)
        with (self.raiz / 'material.log').open('wb') as log:
            ejecutar([REPO / 'scripts/generar_credenciales_desarrollo.sh', self.material],
                      salida=log, entorno={**os.environ, 'VEC_DESARROLLO_VIGENCIA_DIAS': '90'})
        self.tls_pg()
        memoria()
        go = subprocess.check_output([REPO / 'scripts/seleccionar_toolchain_go_local.sh'], text=True).strip()
        with (self.raiz / 'compilacion.log').open('wb') as log:
            ejecutar([go, 'build', '-p', '6', '-o', self.raiz / 'vec-server', './cmd/vec-server'],
                      cwd=REPO, salida=log, entorno={**os.environ, 'GOTOOLCHAIN': 'local',
                          'GOPROXY': 'off', 'GOSUMDB': 'off', 'GOENV': 'off'})
        estado = self.leer()
        estado['preparada'] = True
        guardar(self.estado, estado)
        self.arrancar()
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
        # Solo accesible al contenedor a través de un padre 0700 en el host.
        password.chmod(0o644)

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

    def arrancar(self):
        estado = self.leer()
        if not estado['preparada'] or estado['revision'] != revision():
            raise ValueError('Copia incompleta o revisión distinta; use una instancia nueva')
        self.pg()
        if self.proceso(estado):
            raise ValueError('La aplicación propia ya está activa')
        puerto_libre(estado['puerto_app'])
        entorno = dict(PATH=os.environ['PATH'], LANG='C.UTF-8',
            VEC_EXECUTION_PROFILE='desarrollo', VEC_AUTH_MODE='desarrollo',
            VEC_DEVELOPMENT_GUARD='ACEPTO_CREDENCIALES_NO_AUTORITATIVAS_SOLO_DESARROLLO',
            VEC_DEVELOPMENT_MATERIAL_DIR=str(self.material),
            VEC_TLS_CERT_FILE=str(self.material / 'tls/servidor.crt'),
            VEC_TLS_KEY_FILE=str(self.material / 'tls/servidor.key'),
            VEC_HTTP_ADDR=f"127.0.0.1:{estado['puerto_app']}",
            VEC_BOLSA_DATA_DIR=str(self.raiz / 'datos'))
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
                self.comprobar()
                return
            except (OSError, ValueError):
                time.sleep(.5)
        raise ValueError('La aplicación no respondió en 60 segundos')

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

    def comprobar(self):
        estado = self.leer()
        contexto = ssl.create_default_context(cafile=str(self.material / 'ca/ca.crt'))
        contexto.load_cert_chain(str(self.material / 'mtls/cliente.crt'), str(self.material / 'mtls/cliente.key'))
        for ruta in ['/livez', '/portal-empleado/']:
            with urllib.request.urlopen(f"https://localhost:{estado['puerto_app']}{ruta}", context=contexto, timeout=3) as r:
                if r.status != 200:
                    raise ValueError('Respuesta HTTP distinta de 200')

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
    p.add_argument('orden', choices=['crear', 'arrancar', 'parar', 'reiniciar', 'comprobar', 'borrar', 'exportar-clon'])
    p.add_argument('--equipo', required=True)
    p.add_argument('--instancia', required=True)
    p.add_argument('--puerto-pg', type=int, default=55481)
    p.add_argument('--puerto-app', type=int, default=18481)
    p.add_argument('--lista-sql', action='append', default=[])
    p.add_argument('--archivo-clon')
    args = p.parse_args()
    if not re.fullmatch('[a-z]', args.equipo) or not re.fullmatch('[a-z0-9][a-z0-9-]{0,40}', args.instancia):
        p.error('Equipo: una letra minúscula; instancia: letras, cifras y guiones')
    if not 1024 <= args.puerto_pg <= 65535 or not 1024 <= args.puerto_app <= 65535 or args.puerto_pg == args.puerto_app:
        p.error('Puertos distintos entre 1024 y 65535')
    copia = Copia(args.equipo, args.instancia)
    BASE.mkdir(parents=True, exist_ok=True)
    # Candado por instancia fuera de PGDATA; borrar no permite otro escritor concurrente.
    with (BASE / f'.{copia.nombre}.lock').open('a') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
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
