#!/usr/bin/env python3
"""Ejecuta todas las reglas de GoSec con su adaptador oficial de paquetes."""
import argparse
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tempfile

HERRAMIENTAS = Path(__file__).resolve().parent


def seleccionar_go():
    declaradas = re.findall(r'^toolchain\s+(go\d+\.\d+\.\d+)\s*(?://.*)?$',
                            (HERRAMIENTAS.parent / 'go.mod').read_text(), re.MULTILINE)
    if len(declaradas) != 1:
        raise ValueError('El proyecto debe declarar una versión exacta de Go')
    candidato = shutil.which('go')
    if candidato:
        version = subprocess.check_output([candidato, 'version'], text=True, timeout=10,
                    env={**os.environ, 'GOENV': 'off', 'GOTOOLCHAIN': 'local',
                         'GOPROXY': 'off', 'GOSUMDB': 'off'})
        if re.fullmatch(r'go version ' + re.escape(declaradas[0]) + r' \S+/\S+\n?', version):
            return candidato
    return subprocess.check_output([HERRAMIENTAS / 'seleccionar_toolchain_go_local.sh'],
                                    text=True).strip()


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
    go = seleccionar_go()
    entorno = {**os.environ, 'GOTOOLCHAIN': 'local', 'GOENV': 'off'}
    entorno['PATH'] = str(Path(go).parent) + os.pathsep + entorno['PATH']
    entorno['GOWORK'] = 'off'
    coincidencias = subprocess.check_output([go, 'list', '-f',
                    '{{range .Match}}{{println .}}{{end}}', *args.paquetes],
                    text=True, env=entorno, timeout=60)
    if not set(args.paquetes).issubset(set(coincidencias.splitlines())):
        raise ValueError('Algún patrón no contiene paquetes; no se acepta un análisis parcial')
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
