"""Composición del ejecutor ADMIN privado con el ciclo de la copia propia."""

import fcntl
import hashlib
import os
from pathlib import Path
import signal
import stat
import subprocess
import sys
import time


def huella_ejecutor(ruta):
    ruta = Path(ruta)
    dato = ruta.lstat()
    if (not ruta.is_absolute() or ruta.resolve(strict=True) != ruta
            or not stat.S_ISREG(dato.st_mode) or dato.st_uid != os.getuid()
            or dato.st_mode & 0o077 or dato.st_nlink != 1):
        raise ValueError('El ejecutor ADMIN debe ser un fichero privado propio sin enlaces')
    return hashlib.sha256(ruta.read_bytes()).hexdigest()


def ejecutar(copia, fase):
    estado = copia.leer()
    control = estado.get('administracion')
    if control is None:
        return
    if fase not in ('preparar', 'arrancar', 'comprobar'):
        raise ValueError('Fase ADMIN desconocida')
    ruta = Path(control['ejecutor'])
    if huella_ejecutor(ruta) != control['sha256']:
        raise ValueError('El ejecutor ADMIN cambió; no se ejecutará sobre esta copia')
    fd = copia.candado
    dato = os.fstat(fd)
    esperado = (copia.raiz.parent / f'.{copia.nombre}.lock').stat()
    if (not stat.S_ISREG(dato.st_mode)
            or (dato.st_dev, dato.st_ino) != (esperado.st_dev, esperado.st_ino)):
        raise ValueError('El candado no pertenece a esta copia')
    # El hijo recibe la misma descripción abierta: no abre un segundo candado
    # incompatible con el que mantiene el conductor durante toda la creación.
    fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
    with (copia.raiz / 'administracion.log').open('ab') as log:
        subprocess.run([sys.executable, str(ruta), '--raiz', str(copia.raiz),
                        '--fase', fase, '--candado-fd', str(fd),
                        '--puerto-admin', str(control['puerto'])],
                       stdin=subprocess.DEVNULL, stdout=log, stderr=subprocess.STDOUT,
                       pass_fds=(fd,), check=True, timeout=600)


def activo(copia, estado):
    pid = estado.get('pid_admin')
    if type(pid) is not int or pid <= 0:
        return False
    proc = Path('/proc') / str(pid)
    try:
        return ((proc / 'exe').resolve() == copia.raiz / 'vec-admin'
                and (proc / 'stat').read_text().split()[21]
                == estado.get('inicio_proceso_admin'))
    except FileNotFoundError:
        return False


def mismo_proceso(estado):
    pid = estado.get('pid_admin')
    if type(pid) is not int or pid <= 0:
        return False
    try:
        return (Path('/proc') / str(pid) / 'stat').read_text().split()[21] == estado.get('inicio_proceso_admin')
    except FileNotFoundError:
        return False


def parar(copia):
    # La parada no ejecuta fuente privada que pudiera haber cambiado desde el
    # arranque. PID, ejecutable y comienzo del proceso deben seguir coincidiendo.
    estado = copia.leer()
    if not activo(copia, estado):
        if mismo_proceso(estado):
            # Mismo PID y comienzo con otro ejecutable: se recompiló o borró
            # vec-admin mientras corría. No se señala ni se sigue borrando.
            raise ValueError('ADMIN sigue activo con su binario cambiado; se conserva la copia')
        return
    os.kill(estado['pid_admin'], signal.SIGTERM)
    for _ in range(100):
        if not activo(copia, estado):
            return
        time.sleep(.1)
    raise ValueError('ADMIN sigue activo; se conserva la copia y PostgreSQL')
