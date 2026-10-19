#!/usr/bin/env python3
"""Prepara clasificación externa sin activar inscripción."""
import argparse
import os
from pathlib import Path
import subprocess
import sys

os.umask(0o077)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--raiz', required=True, type=Path)
    parser.add_argument('--candado-fd', type=int)
    args = parser.parse_args()
    orden = [sys.executable, str(Path(__file__).with_name('preparar_portal.py')),
             '--raiz', str(args.raiz), '--sin-inscripcion']
    heredados = ()
    if args.candado_fd is not None:
        if args.candado_fd < 0:
            raise ValueError('Descriptor de candado inválido')
        orden.extend(('--candado-fd', str(args.candado_fd)))
        heredados = (args.candado_fd,)
    subprocess.run(orden, check=True, pass_fds=heredados)
    print('Material externo preparado para clasificación. La inscripción sigue apagada.')


if __name__ == '__main__':
    main()
