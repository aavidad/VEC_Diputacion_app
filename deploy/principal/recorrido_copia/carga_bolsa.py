"""Constituye la bolsa Administrativo mediante la carga ordinaria de CONVOCA."""

import base64
import hashlib
import http.client
import json
from pathlib import Path
import re
import ssl

from admin_copia.arranque import objeto_sin_duplicados


FUENTE = Path('data/demo/bolsa/convoca/administrativo/resumen.xls')
CATEGORIA = 'administrativo'
CATEGORIA_REF = 'categoria:rpt:administrativo'
VISTA = '/api/vec/bolsa/cargas-convoca/vista-previa'
CONFIRMAR = '/api/vec/bolsa/cargas-convoca'
HEX = re.compile(r'[0-9a-f]{64}\Z')
MAX_RESPUESTA = 65_536
MAX_FICHERO = 1_048_576


def _exigir(condicion, motivo):
    if not condicion:
        raise ValueError(motivo)


def _peticion(c, puerto, ruta, cuerpo):
    c.comprobar_fuente()
    contexto = ssl.create_default_context(cafile=str(c.raiz / 'material/ca/ca.crt'))
    contexto.load_cert_chain(str(c.raiz / 'material/mtls/cliente.crt'),
                            str(c.raiz / 'material/mtls/cliente.key'))
    conexion = http.client.HTTPSConnection('localhost', puerto, context=contexto, timeout=30)
    try:
        conexion.request('POST', ruta, body=cuerpo, headers={
            'Origin': f'https://localhost:{puerto}', 'Content-Type': 'application/json',
            'Accept': 'application/json', 'Sec-Fetch-Site': 'same-origin',
            'Sec-Fetch-Mode': 'cors', 'Sec-Fetch-Dest': 'empty'})
        respuesta = conexion.getresponse()
        raw = respuesta.read(MAX_RESPUESTA + 1)
        _exigir(len(raw) <= MAX_RESPUESTA, 'Respuesta CONVOCA excesiva')
        estado = respuesta.status
    finally:
        conexion.close()
    c.comprobar_fuente()
    if estado not in (200, 201):
        raise ValueError(f'Carga CONVOCA rechazada (HTTP {estado})')
    try:
        datos = json.loads(raw, object_pairs_hook=objeto_sin_duplicados)
    except (TypeError, UnicodeDecodeError, ValueError) as error:
        raise ValueError('Respuesta CONVOCA inválida') from error
    _exigir(isinstance(datos, dict) and isinstance(datos.get('data'), dict),
            'Respuesta CONVOCA sin datos')
    return estado, datos['data']


def _acta(huella):
    contexto = (huella + '\x1f' + CATEGORIA_REF).encode()
    return 'acta:importacion-convoca:' + hashlib.sha256(contexto).hexdigest()


def _recibo(datos, huella, estado):
    _exigir(datos.get('esquema') == 'vec.bolsa.rrhh.carga_convoca.recibo.v1'
            and datos.get('huella_sha256') == huella
            and datos.get('acta_ref') == _acta(huella),
            'Recibo CONVOCA discordante con fichero o categoría')
    _exigir(isinstance(datos.get('bolsa_ref'), str) and datos['bolsa_ref'].startswith('bolsa:')
            and type(datos.get('version_bolsa')) is int and datos['version_bolsa'] > 0
            and type(datos.get('reutilizada')) is bool
            and type(datos.get('acta_reutilizada')) is bool
            and type(datos.get('filas_cargadas')) is int and datos['filas_cargadas'] > 0
            and datos.get('filas_excluidas') == 0
            and isinstance(datos.get('auditoria_ref'), str) and datos['auditoria_ref']
            and isinstance(datos.get('confirmada_en'), str) and datos['confirmada_en']
            and ((estado == 200) == datos['reutilizada']),
            'Recibo CONVOCA incompleto')
    return {'bolsa_ref': datos['bolsa_ref'], 'version': datos['version_bolsa'],
            'acta_ref': datos['acta_ref'], 'huella': huella}


def _aspirante(vista):
    filas = vista.get('filas')
    _exigir(isinstance(filas, list), 'Vista previa CONVOCA sin filas')
    primeras = [fila for fila in filas if isinstance(fila, dict)
                and type(fila.get('posicion')) is int and fila['posicion'] == 1]
    _exigir(len(primeras) == 1, 'Primera posición CONVOCA ausente o duplicada')
    fila = primeras[0]
    avisos = fila.get('avisos')
    _exigir(fila.get('estado') == 'aceptada'
            and type(fila.get('numero')) is int and fila['numero'] > 0
            and isinstance(avisos, list) and all(isinstance(a, str) for a in avisos)
            and 'identidad_ambigua' not in avisos
            and fila.get('errores') == [],
            'Primera posición CONVOCA inválida o ambigua')
    partes = [fila.get('nombre'), fila.get('primer_apellido'),
              fila.get('segundo_apellido', '')]
    _exigir(all(isinstance(parte, str) and parte.isprintable() for parte in partes)
            and partes[0].strip() and partes[1].strip(),
            'Nombre de primera posición CONVOCA inválido')
    nombre = ' '.join(parte.strip() for parte in partes if parte.strip())
    _exigir(len(nombre) <= 120 and nombre.isprintable(),
            'Nombre de primera posición CONVOCA fuera de límite')
    return {'fila_numero': fila['numero'], 'nombre': nombre}


def preparar(c):
    """Devuelve el recibo verificado y la primera posición de la vista previa."""
    c.comprobar_fuente()
    estado = c.copia.leer()
    puerto = estado.get('puerto_app')
    _exigir(type(puerto) is int and 1 <= puerto <= 65535,
            'Puerto de la aplicación ausente')
    fuente = c.fuente / FUENTE
    _exigir(fuente.is_file() and not fuente.is_symlink(), 'XLS CONVOCA ausente')
    contenido = fuente.read_bytes()
    _exigir(0 < len(contenido) <= MAX_FICHERO, 'XLS CONVOCA fuera de límite')
    huella = c.sha(fuente)
    _exigir(HEX.fullmatch(huella) and hashlib.sha256(contenido).hexdigest() == huella,
            'Huella XLS CONVOCA discordante')
    base = {'nombre_fichero': FUENTE.name,
            'contenido_base64': base64.b64encode(contenido).decode('ascii'),
            'categoria': CATEGORIA}
    ruta = c.carpeta / 'carga-bolsa'
    anterior = ruta / 'recibo.json'
    if anterior.exists():
        guardado = c.leer(anterior)
        _exigir(isinstance(guardado, dict), 'Recibo CONVOCA anterior inválido')
        previo = _recibo(guardado, huella, 200 if guardado.get('reutilizada') else 201)
    else:
        previo = None

    vista_cuerpo = json.dumps(base, separators=(',', ':')).encode()
    codigo, vista = _peticion(c, puerto, VISTA, vista_cuerpo)
    _exigir(codigo == 200
            and vista.get('esquema') == 'vec.bolsa.rrhh.carga_convoca.vista_previa.v1'
            and vista.get('huella_sha256') == huella
            and vista.get('nombre_fichero') == FUENTE.name
            and type(vista.get('filas_leidas')) is int and vista['filas_leidas'] > 0
            and type(vista.get('aceptadas')) is int and vista['aceptadas'] > 0
            and vista.get('rechazadas') == 0 and vista.get('bloqueo') == ''
            and vista['aceptadas'] == vista['filas_leidas'],
            'Vista previa CONVOCA inválida o con filas rechazadas')
    aspirante = _aspirante(vista)
    c.guardar(ruta / 'vista-previa.json', {
        'huella_sha256': huella, 'categoria': CATEGORIA,
        'nombre_fichero': FUENTE.name, 'filas_leidas': vista['filas_leidas'],
        'aceptadas': vista['aceptadas'], 'rechazadas': vista['rechazadas'],
        'aspirante': aspirante})

    confirmar = dict(base, excluir_filas_con_errores=False)
    cuerpo = json.dumps(confirmar, separators=(',', ':')).encode()
    codigo, recibo = _peticion(c, puerto, CONFIRMAR, cuerpo)
    resultado = _recibo(recibo, huella, codigo)
    _exigir(recibo['filas_cargadas'] == vista['aceptadas'],
            'Recibo CONVOCA no coincide con vista previa')
    _exigir(previo is None or previo == resultado,
            'Replay CONVOCA distinto del recibo anterior')
    if previo is None:
        c.guardar(anterior, recibo)
    else:
        _exigir(codigo == 200, 'Replay CONVOCA no reutilizado')
        c.guardar(ruta / 'replay.json', {'recibo': resultado, 'reutilizada': True})
    return dict(resultado, aspirante=aspirante)
