"""Manifiesto privado de firma generado desde las fuentes de la candidata."""

import hashlib
import json
import os
from pathlib import Path
import shlex
import shutil
import stat
import sys

from copia_local_firma import privado


ETAPAS = ('administracion', 'montajes', 'firmantes', 'actos', 'plan',
          'admisiones', 'configuracion')


def retirar(copia, directorio=None):
    """Retira únicamente los lanzadores generados para esta copia."""
    ruta = Path(directorio) if directorio is not None else (
        Path.home() / '.local/state' / ('vec-firma-' + copia.nombre))
    if not ruta.exists() and not ruta.is_symlink():
        return
    privado(ruta, directorio=True, padres=True)
    meta = json.loads(privado(ruta / 'fuentes.json').read_text())
    if meta['raiz'] != str(copia.raiz) or meta['nombre'] != copia.nombre:
        raise ValueError('firma_lanzadores_de_otra_copia')
    for entrada in ruta.rglob('*'):
        privado(entrada, directorio=entrada.is_dir())
    shutil.rmtree(ruta)


def _conservar(ruta, datos, modo=0o600):
    if ruta.exists() or ruta.is_symlink():
        privado(ruta)
        if ruta.read_bytes() != datos or stat.S_IMODE(ruta.stat().st_mode) != modo:
            raise ValueError('firma_fuente_generada_divergente')
        return
    fd = os.open(ruta, os.O_WRONLY | os.O_CREAT | os.O_EXCL, modo)
    with os.fdopen(fd, 'wb') as archivo:
        archivo.write(datos)


def generar(copia, *, repo, verificador=None, puerto=None, directorio=None):
    """Conserva las fuentes exactas y reutiliza el contrato de manifiesto v1.

    El manifiesto no acredita publicación: las etapas conservan los recibos
    de los productores nominales y se ejecutan mediante el conductor existente.
    """
    estado = copia.leer()
    origen = Path(repo) / 'deploy/principal/firma_copia'
    entrada = origen / 'ejecutar.py'
    if origen.is_symlink() or not entrada.is_file() or entrada.is_symlink():
        raise ValueError('firma_fuentes_versionadas_ausentes')
    for nombre in ('comun', 'administracion', 'montajes', 'firmantes',
                   'publicaciones', 'plan', 'configuracion'):
        fuente = origen / (nombre + '.py')
        if fuente.is_symlink() or not fuente.is_file():
            raise ValueError('firma_fuente_versionada_ausente:' + nombre)
    # El ejecutor exige ancestros sin escritura de grupo. La carpeta común
    # de memoria puede ser compartida; los lanzadores públicos van al estado
    # privado del usuario. Las claves y recibos se generan en la propia copia.
    directorio = Path(directorio) if directorio is not None else (
        Path.home() / '.local/state' / ('vec-firma-' + copia.nombre))
    directorio.mkdir(mode=0o700, exist_ok=True)
    privado(directorio, directorio=True, padres=True)
    fuentes = directorio / 'fuentes'
    fuentes.mkdir(mode=0o700, exist_ok=True)
    hashes = {}
    for fuente in sorted(origen.rglob('*')):
        if fuente.is_symlink():
            raise ValueError('firma_fuente_versionada_enlazada')
        if not fuente.is_file() or fuente.suffix not in ('.py', '.json', '.sql'):
            continue
        relativa = fuente.relative_to(origen)
        destino = fuentes / relativa
        destino.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
        privado(destino.parent, directorio=True)
        datos = fuente.read_bytes()
        _conservar(destino, datos)
        hashes[str(relativa)] = hashlib.sha256(datos).hexdigest()
    puerto = puerto if puerto is not None else estado['puerto_app'] + 40000
    if type(puerto) is not int or not 1024 <= puerto <= 65535:
        raise ValueError('firma_puerto_verificador_invalido')
    _conservar(directorio / 'fuentes.json',
               (json.dumps({'revision': estado['revision'], 'raiz': str(copia.raiz),
                            'nombre': copia.nombre, 'fuente': str(Path(repo)),
                            'puerto_admin': estado['puerto_app'] + 1000, 'archivos': hashes},
                           sort_keys=True, separators=(',', ':')) + '\n').encode())
    etapas = []
    for etapa in ETAPAS:
        lanzador = directorio / etapa
        orden = [sys.executable, '-I', str(fuentes / 'ejecutar.py'), etapa]
        _conservar(lanzador,
                   ('#!/bin/sh\nset -eu\nexec ' + shlex.join(orden) + '\n').encode(), 0o700)
        etapas.append({'nombre': etapa, 'orden': [str(lanzador)]})
    herramienta = verificador or shutil.which('grxfirma')
    if not herramienta:
        raise ValueError('firma_verificador_instalado_ausente')
    herramienta = Path(herramienta)
    if not herramienta.is_absolute() or herramienta.is_symlink() or not herramienta.is_file():
        raise ValueError('firma_verificador_instalado_invalido')
    binario = directorio / 'grxfirma-verificador'
    _conservar(binario, herramienta.read_bytes(), 0o700)
    manifiesto = {'version': 1, 'etapas': etapas,
                  'verificador': {'binario': str(binario),
                                  'binario_sha256': hashlib.sha256(binario.read_bytes()).hexdigest(),
                                  'puerto': puerto}}
    ruta = directorio / 'manifiesto.json'
    _conservar(ruta, (json.dumps(manifiesto, sort_keys=True) + '\n').encode())
    return ruta
