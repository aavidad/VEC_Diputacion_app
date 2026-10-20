"""Recursos del recorrido, ligados al candado y a los datos de una copia."""

import datetime as dt
import fcntl
import hashlib
import http.client
import json
import os
from pathlib import Path
import re
import socket
import ssl
import subprocess

from admin_copia.arranque import objeto_sin_duplicados, privado


class Contexto:
    def __init__(self, copia, parametros):
        from copia_local import REPO, verificar_fuente
        estado = copia.leer()
        verificar_fuente(estado['revision'])
        self.revision = estado['revision']
        self.copia, self.fuente = copia, REPO
        self.raiz, self.material = copia.raiz, copia.material
        descriptor = copia.candado
        actual = os.fstat(descriptor)
        esperado = (self.raiz.parent / f'.{copia.nombre}.lock').stat()
        if (actual.st_dev, actual.st_ino) != (esperado.st_dev, esperado.st_ino):
            raise ValueError('recorrido_candado_ajeno')
        fcntl.flock(descriptor, fcntl.LOCK_EX | fcntl.LOCK_NB)
        self.carpeta = self.raiz / 'recorrido'
        self.carpeta.mkdir(mode=0o700, exist_ok=True)
        if self.carpeta.resolve() != self.carpeta:
            raise ValueError('recorrido_salida_enlazada')
        self.bin = self.carpeta / 'bin'
        self.bin.mkdir(mode=0o700, exist_ok=True)
        self.parametros = dict(parametros)
        self.puerto_admin = estado.get('administracion', {}).get('puerto')

    @staticmethod
    def leer(ruta):
        return json.loads(privado(ruta, 2_097_152), object_pairs_hook=objeto_sin_duplicados)

    @staticmethod
    def guardar(ruta, datos):
        ruta = Path(ruta)
        raw = datos if isinstance(datos, bytes) else (
            json.dumps(datos, ensure_ascii=False, allow_nan=False, separators=(',', ':')) + '\n').encode()
        ruta.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
        if ruta.resolve() != ruta:
            raise ValueError('recorrido_salida_enlazada')
        if ruta.exists():
            if privado(ruta, 2_097_152) != raw:
                raise ValueError('recorrido_material_divergente')
            return False
        descriptor = os.open(ruta, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
        with os.fdopen(descriptor, 'wb') as salida:
            salida.write(raw)
            salida.flush()
            os.fsync(salida.fileno())
        return True

    @staticmethod
    def ahora():
        return dt.datetime.now(dt.timezone.utc).replace(microsecond=0)

    @staticmethod
    def sha(ruta):
        with Path(ruta).open('rb') as entrada:
            return hashlib.file_digest(entrada, 'sha256').hexdigest()

    def comprobar_fuente(self):
        from copia_local import verificar_fuente
        verificar_fuente(self.revision)

    def sql(self, datos, nombre_log):
        self.comprobar_fuente()
        self.copia.sql_bytes(datos, nombre_log)
        self.comprobar_fuente()

    def consultar(self, consulta):
        resultado = self.ejecutar([
            '/usr/bin/docker', 'exec', '-i', '-u', 'postgres', self.copia.nombre,
            'psql', '-XAtq', '-v', 'ON_ERROR_STOP=1', '-U', 'postgres', '-d', 'postgres',
            '-c', consulta], 'consultas.log')
        return resultado.stdout.decode().strip()

    def ejecutar(self, argv, nombre_log, cwd=None, env=None):
        if not re.fullmatch(r'[a-zA-Z0-9_.-]{1,100}', nombre_log):
            raise ValueError('recorrido_registro_invalido')
        self.comprobar_fuente()
        entorno = {'PATH': '/usr/bin:/bin', 'HOME': os.environ['HOME'], 'LANG': 'C.UTF-8',
                   'DOCKER_HOST': 'unix:///var/run/docker.sock'}
        entorno.update(env or {})
        resultado = subprocess.run([str(a) for a in argv], cwd=cwd, env=entorno,
                                   stdin=subprocess.DEVNULL, capture_output=True,
                                   timeout=600, check=False)
        destino = self.carpeta / nombre_log
        descriptor = os.open(destino, os.O_WRONLY | os.O_CREAT | os.O_APPEND | os.O_NOFOLLOW, 0o600)
        os.fchmod(descriptor, 0o600)
        with os.fdopen(descriptor, 'ab') as salida:
            salida.write(resultado.stdout)
            salida.write(resultado.stderr)
        self.comprobar_fuente()
        if resultado.returncode:
            raise ValueError('recorrido_operacion_fallida_consulte_registro')
        return resultado

    def cli(self, nombre):
        if not re.fullmatch(r'vec-[a-z0-9-]{1,70}', nombre) or not (self.fuente / 'cmd' / nombre).is_dir():
            raise ValueError('recorrido_cli_invalida')
        self.comprobar_fuente()
        destino, marca = self.bin / nombre, self.bin / (nombre + '.sha256')
        if destino.exists():
            if privado(marca).decode().strip() != self.sha(destino):
                raise ValueError('recorrido_cli_divergente')
            return destino
        from copia_local import memoria
        memoria()
        compilacion = self.leer(self.raiz / 'compilacion.json')
        self.ejecutar([compilacion['go'], 'build', '-trimpath', '-p', '2', '-o', destino,
                      './cmd/' + nombre], nombre + '-compilacion.log', cwd=self.fuente,
                     env={'GOTOOLCHAIN': 'local', 'GOPROXY': 'off', 'GOSUMDB': 'off',
                          'GOENV': 'off', 'GOMAXPROCS': '2', 'GOMEMLIMIT': '512MiB',
                          'GOCACHE': compilacion['gocache'], 'GOMODCACHE': compilacion['gomodcache']})
        destino.chmod(0o700)
        self.guardar(marca, (self.sha(destino) + '\n').encode())
        return destino

    def admin(self, numero, metodo, ruta, cuerpo=None):
        if (numero not in (1, 2) or metodo not in ('GET', 'POST')
                or not ruta.startswith('/api/admin/') or '\\' in ruta or '#' in ruta
                or any(p == '..' for p in ruta.split('/'))
                or type(self.puerto_admin) is not int):
            raise ValueError('recorrido_peticion_admin_invalida')
        self.comprobar_fuente()
        base = self.raiz / 'admin/h10'
        contexto = ssl.create_default_context(cafile=str(base / 'vec-admin/tls/srv-ca.pem'))
        contexto.load_cert_chain(str(base / f'fuentes/cliente-{numero}.pem'),
                                str(base / f'fuentes/cliente-{numero}.key'))
        conexion = http.client.HTTPSConnection('admin.localhost', self.puerto_admin,
                                             context=contexto, timeout=30)
        try:
            conexion.sock = contexto.wrap_socket(
                socket.create_connection(('127.0.0.1', self.puerto_admin), timeout=30),
                server_hostname='admin.localhost')
            cabeceras = {'Accept': 'application/json',
                         'Origin': f'https://admin.localhost:{self.puerto_admin}',
                         'Sec-Fetch-Site': 'same-origin', 'Sec-Fetch-Mode': 'cors',
                         'Sec-Fetch-Dest': 'empty'}
            datos = None
            if cuerpo is not None:
                datos = json.dumps(cuerpo, ensure_ascii=False, allow_nan=False,
                                   separators=(',', ':')).encode()
                if len(datos) > 2_097_152:
                    raise ValueError('recorrido_peticion_excesiva')
                cabeceras['Content-Type'] = 'application/json'
            conexion.request(metodo, ruta, body=datos, headers=cabeceras)
            respuesta = conexion.getresponse()
            raw = respuesta.read(2_097_153)
            if len(raw) > 2_097_152:
                raise ValueError('recorrido_respuesta_excesiva')
            datos = json.loads(raw or b'{}', object_pairs_hook=objeto_sin_duplicados)
            self.comprobar_fuente()
            return respuesta.status, datos
        finally:
            conexion.close()
