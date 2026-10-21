"""Retira el lote ficticio HZ37 que se volverá a cargar por la API normal."""

import hashlib
import json
from pathlib import Path
import re

from .carga_bolsa import FUENTE


HUELLA_XLS = 'b0e202a6e25187429c17997f870d276b0ca440da134bb8c1e084e5440055940d'


def preparar(copia):
    from .datos import HUELLA_HZ37
    estado = copia.leer()
    repo = Path(__file__).resolve().parents[3]
    sid = estado.get('sid_copia', '')
    if (estado.get('fuente') != str(repo) or estado.get('preparada')
            or estado.get('material_modo') != 'nuevo'
            or not estado.get('recorrido_completo') or not copia.existe()
            or not isinstance(sid, str) or not re.fullmatch(r'[0-9]{15,20}', sid)):
        raise ValueError('La limpieza requiere una copia nueva del recorrido completo')
    fuentes = json.loads((copia.raiz / 'fuentes.json').read_bytes())
    if (fuentes.get('principal-hz37-20261014.dump') != HUELLA_HZ37
            or hashlib.sha256((repo / FUENTE).read_bytes()).hexdigest() != HUELLA_XLS):
        raise ValueError('La limpieza no corresponde al lote versionado de HZ37')
    sql = (Path(__file__).parent / 'datos/limpiar_lote_versionado.sql').read_text()
    sql = sql.replace('__SID_COPIA__', sid).replace('__HUELLA_FICHERO__', HUELLA_XLS)
    copia.sql_bytes(sql.encode(), 'datos-lote-versionado.log')
