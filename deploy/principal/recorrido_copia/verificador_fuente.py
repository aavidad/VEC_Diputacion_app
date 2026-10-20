"""Coteja el ejecutable GrxFirma antes de preparar la copia local."""

import hashlib
import os
from pathlib import Path
import re
import resource
import signal
import stat
import subprocess
import tempfile


HUELLA = re.compile(r'[0-9a-f]{64}\Z')
VERSION = re.compile(r'grxfirma ((?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)|dev)\Z')
LIMITE_BINARIO = 100_000_000
LIMITE_SALIDA = 65_536
TIEMPO_SONDA = 5


def _exigir(condicion, motivo):
    if not condicion:
        raise ValueError(motivo)


def _huella(ruta):
    """Lee sin seguir enlaces y conserva identidad del archivo verificado."""
    try:
        _exigir(ruta.is_absolute() and not ruta.is_symlink()
                and ruta.resolve(strict=True) == ruta,
                'grxfirma_ruta_insegura')
        previo = ruta.lstat()
        _exigir(stat.S_ISREG(previo.st_mode) and previo.st_uid in (0, os.getuid())
                and not previo.st_mode & 0o022 and os.access(ruta, os.X_OK)
                and previo.st_nlink == 1 and 0 < previo.st_size <= LIMITE_BINARIO,
                'grxfirma_archivo_inseguro')
        descriptor = os.open(ruta, os.O_RDONLY | os.O_NOFOLLOW | os.O_CLOEXEC)
        with os.fdopen(descriptor, 'rb') as fuente:
            abierto = os.fstat(fuente.fileno())
            _exigir((previo.st_dev, previo.st_ino, previo.st_size)
                    == (abierto.st_dev, abierto.st_ino, abierto.st_size),
                    'grxfirma_archivo_cambiado')
            digest = hashlib.file_digest(fuente, 'sha256').hexdigest()
        return digest, (previo.st_dev, previo.st_ino, previo.st_size)
    except (OSError, RuntimeError) as error:
        raise ValueError('grxfirma_archivo_no_disponible') from error


def _limitar_salida():
    resource.setrlimit(resource.RLIMIT_FSIZE, (LIMITE_SALIDA, LIMITE_SALIDA))
    resource.setrlimit(resource.RLIMIT_CPU, (TIEMPO_SONDA, TIEMPO_SONDA))


def _sondar(ruta, argumentos, carpeta):
    entorno = {'PATH': '/usr/bin:/bin', 'LANG': 'C', 'LC_ALL': 'C',
               'HOME': str(carpeta), 'XDG_CONFIG_HOME': str(carpeta),
               'XDG_CACHE_HOME': str(carpeta), 'TMPDIR': str(carpeta)}
    with tempfile.TemporaryFile(dir=carpeta) as salida, tempfile.TemporaryFile(dir=carpeta) as error:
        proceso = None
        try:
            proceso = subprocess.Popen([str(ruta), *argumentos], stdin=subprocess.DEVNULL,
                stdout=salida, stderr=error, env=entorno, start_new_session=True,
                preexec_fn=_limitar_salida)
            try:
                retorno = proceso.wait(timeout=TIEMPO_SONDA)
            except subprocess.TimeoutExpired as exc:
                os.killpg(proceso.pid, signal.SIGKILL)
                proceso.wait()
                raise ValueError('grxfirma_sonda_agotada') from exc
            _exigir(retorno == 0 and salida.tell() <= LIMITE_SALIDA
                    and error.tell() <= LIMITE_SALIDA,
                    'grxfirma_sonda_rechazada')
            salida.seek(0)
            error.seek(0)
            return salida.read(LIMITE_SALIDA + 1), error.read(LIMITE_SALIDA + 1)
        except (OSError, subprocess.SubprocessError) as exc:
            raise ValueError('grxfirma_sonda_no_disponible') from exc


def validar(binario: Path, sha_esperada: str | None) -> dict:
    """Exige versión apta y ayuda con el flag REST de solo verificación."""
    ruta = Path(binario)
    _exigir(sha_esperada is None or
            (isinstance(sha_esperada, str) and HUELLA.fullmatch(sha_esperada)),
            'grxfirma_huella_declarada_invalida')
    digest, identidad = _huella(ruta)
    _exigir(sha_esperada is None or digest == sha_esperada,
            'grxfirma_huella_distinta')
    try:
        temporal_privado = tempfile.TemporaryDirectory(
            prefix='grxfirma-preflight-', dir='/dev/shm/vec')
    except OSError:
        raise ValueError('grxfirma_sonda_no_disponible') from None
    with temporal_privado as temporal:
        carpeta = Path(temporal)
        version_out, version_err = _sondar(ruta, ['--version'], carpeta)
        _exigir(not version_err and len(version_out) <= LIMITE_SALIDA,
                'grxfirma_version_invalida')
        try:
            version_line = version_out.decode('ascii').strip()
        except UnicodeDecodeError as exc:
            raise ValueError('grxfirma_version_invalida') from exc
        match = VERSION.fullmatch(version_line)
        _exigir(match is not None, 'grxfirma_version_invalida')
        version = match.group(1)
        if version == 'dev':
            _exigir(sha_esperada is not None, 'grxfirma_dev_sin_huella')
        else:
            _exigir(len(version) <= 32 and
                    tuple(map(int, version.split('.'))) >= (0, 0, 106),
                    'grxfirma_version_antigua')
        ayuda_out, ayuda_err = _sondar(
            ruta, ['-rest-solo-verificacion', '--help'], carpeta)
        ayuda = ayuda_out + ayuda_err
        _exigir(len(ayuda) <= 2 * LIMITE_SALIDA and b'Uso:' in ayuda
                and b'grxfirma' in ayuda.lower(),
                'grxfirma_bandera_verificacion_ausente')
    posterior, identidad_posterior = _huella(ruta)
    _exigir(posterior == digest and identidad_posterior == identidad,
            'grxfirma_archivo_cambiado')
    return {'binario': ruta.name, 'binario_sha256': digest, 'version': version}
