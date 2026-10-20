#!/usr/bin/env python3
"""Comprueba la derivación mediante la sesión propia de una copia local."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import ssl
import urllib.error
import urllib.parse
import urllib.request


class SinRedireccion(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def main():
    p = argparse.ArgumentParser()
    for campo in ('origen', 'ca', 'certificado', 'clave', 'entrada', 'salida'):
        p.add_argument('--' + campo, required=True)
    p.add_argument('--recuperar-desde')
    a = p.parse_args()
    os.umask(0o077)
    u = urllib.parse.urlsplit(a.origen)
    if (u.scheme != 'https' or u.hostname not in ('localhost', '127.0.0.1')
            or not u.port or u.username is not None or u.password is not None
            or u.path or u.query or u.fragment):
        raise ValueError('origen_local_invalido')
    entrada = Path(a.entrada).read_bytes()
    if len(entrada) > 4096:
        raise ValueError('entrada_demasiado_grande')
    datos = json.loads(entrada)
    if set(datos) != set(('padre_ref', 'version_padre', 'oferta_ref', 'numero_plaza',
                         'secuencia_plaza', 'participacion_ref', 'recibo_adjudicacion_ref',
                         'clave_idempotencia')):
        raise ValueError('campos_entrada_invalidos')
    salida = Path(a.salida)
    salida.mkdir(mode=0o700, parents=False, exist_ok=False)
    contexto = ssl.create_default_context(cafile=a.ca)
    contexto.load_cert_chain(a.certificado, a.clave)
    http = urllib.request.build_opener(urllib.request.ProxyHandler({}), SinRedireccion(),
                                      urllib.request.HTTPSHandler(context=contexto))

    def guardar(nombre, contenido):
        with (salida / nombre).open('xb') as f:
            f.write(contenido)

    def post(nombre, ruta, cuerpo, esperado):
        req = urllib.request.Request(a.origen + ruta, data=cuerpo,
                                     headers={'Content-Type': 'application/json',
                                              'Accept': 'application/json'}, method='POST')
        try:
            with http.open(req, timeout=30) as respuesta:
                codigo, contenido = respuesta.status, respuesta.read(2097153)
        except urllib.error.HTTPError as err:
            codigo, contenido = err.code, err.read(2097153)
            err.close()
        guardar(nombre + '.json', contenido)
        guardar(nombre + '-http.txt', str(codigo).encode())
        if codigo != esperado or len(contenido) > 2097152:
            raise ValueError('respuesta_inesperada_' + nombre + '_' + str(codigo))
        return json.loads(contenido)['data']

    ruta = '/api/vec/contratacion-temporal/nombramientos/derivaciones'
    if a.recuperar_desde:
        base = Path(a.recuperar_desde)
        snapshot = base / 'origen.json'
        if not snapshot.exists():
            snapshot = base / 'resultado.json'
        anterior = json.loads(snapshot.read_bytes())
        if anterior['entrada_sha256'] != hashlib.sha256(entrada).hexdigest():
            raise ValueError('entrada_distinta_de_la_original')
        original = anterior['derivacion']
    else:
        original = post('crear', ruta, entrada, 201)
    snapshot = json.dumps({'entrada_sha256': hashlib.sha256(entrada).hexdigest(),
                           'derivacion': original}, separators=(',', ':')).encode()
    guardar('origen.json', snapshot)
    replay = post('replay', ruta, entrada, 200)
    if original['estado'] != 'confirmado' or replay['estado'] != 'replay_confirmado':
        raise ValueError('estado_no_confirmado')
    for clave in ('hijo_ref', 'recibo_ref', 'numero_visible', 'numero_plaza', 'total_personas',
                  'version', 'padre_ref', 'aceptacion_ref', 'confirmada_en'):
        if not original.get(clave) or replay.get(clave) != original[clave]:
            raise ValueError('replay_distinto_' + clave)
    if original['padre_ref'] != datos['padre_ref'] or original['numero_plaza'] != datos['numero_plaza']:
        raise ValueError('derivacion_ajena')
    consulta = json.dumps({'expediente_ref': original['hijo_ref'],
                           'version_observada': 0}, separators=(',', ':')).encode()
    detalle = post('detalle', '/api/vec/contratacion-temporal/expedientes/consultas', consulta, 200)
    instruccion = detalle.get('instruccion_compartida', {})
    hitos = detalle.get('hitos', [])
    if (detalle['resumen']['expediente_ref'] != original['hijo_ref']
            or detalle['resumen']['version'] != 6
            or instruccion.get('padre_ref') != original['padre_ref']
            or instruccion.get('recibo_derivacion_ref') != original['recibo_ref']
            or instruccion.get('oferta_ref') != datos['oferta_ref']
            or instruccion.get('numero_plaza') != datos['numero_plaza']
            or len(instruccion.get('hitos_padre', [])) < 6 or len(hitos) != 1
            or hitos[0].get('accion_clave') != 'contratacion_temporal.nombramiento.derivar'):
        raise ValueError('detalle_derivado_incompleto')
    guardar('resultado.json', snapshot)
    print(json.dumps({'estado': 'derivacion_y_lectura_confirmadas', 'replay': 200,
                      'detalle': 200, 'recuperacion': bool(a.recuperar_desde)}))


if __name__ == '__main__':
    main()
