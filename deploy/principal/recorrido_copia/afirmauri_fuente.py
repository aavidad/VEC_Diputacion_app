"""Prevalida la identidad pública de GrxFirma afirmauri sin abrir una URI."""

from pathlib import Path
import re
import tempfile

from .verificador_fuente import HUELLA, LIMITE_SALIDA, _exigir, _huella, _sondar


VERSION = re.compile(
    r'grxfirma-afirmauri ((?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)|dev)\Z')


def validar(binario: Path, sha_esperada: str | None) -> dict:
    """Acepta únicamente el programa afirmauri declarado por ruta y huella."""
    ruta = Path(binario)
    _exigir(sha_esperada is None or
            (isinstance(sha_esperada, str) and HUELLA.fullmatch(sha_esperada)),
            'afirmauri_huella_declarada_invalida')
    digest, identidad = _huella(ruta)
    _exigir(sha_esperada is None or digest == sha_esperada,
            'afirmauri_huella_distinta')
    try:
        temporal_privado = tempfile.TemporaryDirectory(
            prefix='afirmauri-preflight-', dir='/dev/shm/vec')
    except OSError:
        raise ValueError('afirmauri_sonda_no_disponible') from None
    with temporal_privado as temporal:
        stdout, stderr = _sondar(ruta, ['--version'], Path(temporal))
        _exigir(not stdout and 0 < len(stderr) <= LIMITE_SALIDA,
                'afirmauri_version_invalida')
        try:
            linea = stderr.decode('utf-8').strip()
        except UnicodeDecodeError:
            raise ValueError('afirmauri_version_invalida') from None
        coincidencia = VERSION.fullmatch(linea)
        _exigir(coincidencia is not None, 'afirmauri_identidad_invalida')
        version = coincidencia.group(1)
        _exigir(len(version) <= 32, 'afirmauri_version_invalida')
        if version == 'dev':
            _exigir(sha_esperada is not None, 'afirmauri_dev_sin_huella')
    posterior, identidad_posterior = _huella(ruta)
    _exigir(posterior == digest and identidad_posterior == identidad,
            'afirmauri_archivo_cambiado')
    return {'binario': ruta.name, 'binario_sha256': digest, 'version': version}
