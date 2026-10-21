"""Fuente de necesidades para copias de desarrollo, sin modificar sus actos."""

import argparse
import fcntl
import json
from pathlib import Path
import re
import sys

VARIABLE = 'VEC_CT_NECESIDADES_ALTA_SOURCE_PATH'
FUENTE = 'internal/modules/contrataciontemporal/adapters/catalogoalta/necesidades_v1.ejemplo.json'


def entorno_necesidades(repo):
    ruta = Path(repo) / FUENTE
    datos = json.loads(ruta.read_text())
    if (datos.get('esquema') != 'vec.ct.necesidades_alta.v1'
            or datos.get('referencia') != 'catalogo:ct:necesidades_alta:ejemplo'
            or datos.get('version') != 3 or datos.get('es_ejemplo') is not True):
        raise ValueError('La fuente versionada de necesidades no corresponde al recorrido')
    return {VARIABLE: str(ruta)}


def activar(copia, nativo):
    """Reutiliza el gestor de la fuente congelada, incluido su reinicio y sus guardas."""
    estado = copia.leer()
    if not estado.get('preparada') or not estado.get('recorrido_completo'):
        raise ValueError('Se requiere una copia preparada con recorrido completo')
    nativo.verificar_fuente(estado['revision'])
    copia.validar_binario(estado)
    anterior = copia.entorno_modulos()
    nuevas = entorno_necesidades(nativo.REPO)
    if VARIABLE in anterior:
        if anterior[VARIABLE] != nuevas[VARIABLE]:
            raise ValueError('Ya existe otra fuente de necesidades; se conserva')
        copia.comprobar()
        return
    archivo = copia.raiz / 'entorno_modulos.json'

    def publicar(entorno):
        nativo.guardar(archivo, entorno)
        actual = copia.leer()
        actual['entorno_modulos_sha256'] = nativo.sha256(archivo)
        nativo.guardar(copia.estado, actual)

    # La copia queda parada antes de cambiar un fichero leído al arrancar.
    copia.parar()
    try:
        publicar({**anterior, **nuevas})
        copia.arrancar()
        copia.comprobar()
    except BaseException:
        copia.parar()
        publicar(anterior)
        copia.arrancar()
        copia.comprobar()
        raise


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--fuente-copia', required=True, type=Path)
    parser.add_argument('--equipo', required=True)
    parser.add_argument('--instancia', required=True)
    args = parser.parse_args()
    if (not re.fullmatch('[a-z]', args.equipo)
            or not re.fullmatch('[a-z0-9][a-z0-9-]{0,40}', args.instancia)):
        parser.error('Equipo: una letra minúscula; instancia: letras, cifras y guiones')
    fuente = args.fuente_copia
    if not fuente.is_absolute() or fuente.resolve(strict=True) != fuente:
        parser.error('La fuente de la copia debe ser una ruta absoluta sin enlaces')
    # Solo carga el gestor del repositorio indicado; su leer() coteja la fuente
    # persistida y arrancar() conserva revisión limpia y huella del binario.
    sys.path.insert(0, str(fuente / 'deploy/principal'))
    import copia_local as nativo
    copia = nativo.Copia(args.equipo, args.instancia)
    with (nativo.BASE / f'.{copia.nombre}.lock').open('a') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        copia.candado = lock.fileno()
        activar(copia, nativo)
    print('Fuente de necesidades activada; copia comprobada tras el reinicio')


if __name__ == '__main__':
    main()
