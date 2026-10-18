#!/usr/bin/env python3
"""Ejecuta todas las reglas de GoSec con su adaptador oficial de paquetes."""
import argparse
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile

HERRAMIENTAS = Path(__file__).resolve().parent


def resultado(informe):
    """El modo JSON de go/analysis devuelve cero incluso si encuentra avisos."""
    if not isinstance(informe, dict):
        raise ValueError('El informe de GoSec no es un objeto')
    avisos = 0
    for analizadores in informe.values():
        if not isinstance(analizadores, dict) or set(analizadores) != {'gosec'}:
            raise ValueError('El informe no corresponde al analizador GoSec')
        diagnosticos = analizadores['gosec']
        if not isinstance(diagnosticos, list):
            raise ValueError('GoSec no pudo completar el análisis')
        for diagnostico in diagnosticos:
            if (not isinstance(diagnostico, dict) or not diagnostico.get('posn')
                    or not diagnostico.get('message')):
                raise ValueError('GoSec devolvió un diagnóstico incompleto')
        avisos += len(diagnosticos)
    return 1 if avisos else 0


def main():
    os.umask(0o077)
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--informe', type=Path, help='Archivo JSON nuevo; permisos privados')
    parser.add_argument('paquetes', nargs='+', help='Directorios, por ejemplo ./internal/app/bootstrap')
    args = parser.parse_args()
    for paquete in args.paquetes:
        directorio = Path(paquete.removesuffix('/...'))
        if paquete.startswith('-') or not directorio.is_dir():
            parser.error('Indique directorios de paquetes; no archivos Go sueltos')
    go = subprocess.check_output([HERRAMIENTAS / 'seleccionar_toolchain_go_local.sh'],
                                 text=True).strip()
    entorno = {**os.environ, 'GOTOOLCHAIN': 'local'}
    entorno['PATH'] = str(Path(go).parent) + os.pathsep + entorno['PATH']
    entorno['GOWORK'] = 'off'
    with tempfile.TemporaryDirectory(prefix='vec-gosec-') as temporal:
        binario = Path(temporal) / 'vec-gosec'
        subprocess.run([go, '-C', HERRAMIENTAS.parent / 'tools/vecgosec',
                        'build', '-mod=readonly', '-p', '6', '-o', binario, '.'],
                       env=entorno, check=True, timeout=300)
        # Este binario solo incorpora el adaptador local. No usa el CLI de
        # autofix, proveedores remotos, claves de API ni exclusiones de reglas.
        proceso = subprocess.run([binario, '-test=false', '-exclude-generated=false',
                                  '-json', *args.paquetes], env=entorno,
                                 stdout=subprocess.PIPE, check=False, timeout=300)
    if proceso.returncode != 0:
        raise ValueError('GoSec no pudo cargar o analizar todos los paquetes')
    informe = json.loads(proceso.stdout)
    estado = resultado(informe)
    if args.informe:
        with args.informe.open('xb') as salida:
            salida.write(proceso.stdout)
    else:
        sys.stdout.buffer.write(proceso.stdout)
    return estado


if __name__ == '__main__':
    try:
        sys.exit(main())
    except (ValueError, OSError, subprocess.SubprocessError) as error:
        print(str(error) if isinstance(error, ValueError) else
              'No se completó GoSec; revise el error anterior.', file=sys.stderr)
        sys.exit(2)
