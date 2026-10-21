#!/usr/bin/env python3
"""Enciende Cronos de la persona empleada en una copia completa ya creada.

Tarea F3-01, corte C1. Se ejecuta después de `copia_local.py crear
--recorrido-completo` con la misma revisión limpia; prepara la etapa de
`recorrido_copia/cronos.py` y reinicia la aplicación de la copia. Repetirla
coteja lo ya preparado y no crea otras cuentas ni otra asignación.
"""

import argparse
import fcntl
import re

from copia_local import BASE, Copia


def main():
    p = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    p.add_argument('--equipo', required=True)
    p.add_argument('--instancia', required=True)
    args = p.parse_args()
    if not re.fullmatch('[a-z]', args.equipo) or not re.fullmatch('[a-z0-9][a-z0-9-]{0,40}', args.instancia):
        p.error('Equipo: una letra minúscula; instancia: letras, cifras y guiones')
    copia = Copia(args.equipo, args.instancia)
    with (BASE / f'.{copia.nombre}.lock').open('a') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        copia.candado = lock.fileno()
        estado = copia.leer()
        if not estado.get('preparada') or estado.get('recorrido_completo') is not True:
            raise SystemExit('Cronos requiere una copia completa ya preparada')
        from recorrido_copia import cronos
        from recorrido_copia.comun import Contexto
        from recorrido_copia.conductor import _parametros
        resultado = cronos.preparar(Contexto(copia, _parametros(copia)))
        copia.parar()
        copia.arrancar()
        print('Cronos preparado: ' + resultado['perfil_ref'])


if __name__ == '__main__':
    main()
