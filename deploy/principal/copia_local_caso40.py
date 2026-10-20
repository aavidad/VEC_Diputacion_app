"""Datos ficticios del caso 4.0 en una copia local con módulos (F1-99).

Sin este paso, una copia nueva no deja recorrer los pasos 9 y 10:

- los guiones de datos del caso (`candidata_hz39/datos`) no estaban aplicados:
  faltaban el puesto 9/01 del centro 520, la disponibilidad de Antonio en la
  bolsa vigente y el cargo de Manuel;
- Antonio no tenía identidad propia en la copia, y la del material original
  apunta a su candidatura de la bolsa sustituida: su permiso de Mi Bolsa
  quedaba ligado a esa candidatura y la aplicación lo dejaba pendiente de
  provisión (403);
- su participación no tenía datos de contacto y el aviso del llamamiento no
  salía al buzón de la copia.

Todo se hace con las herramientas que ya existen: los guiones SQL de datos
del caso, el preparador de Mi Bolsa (`preparar_candidato_desarrollo.sh`), que
crea una identidad nueva ligada a la candidatura vigente, y la API de Bolsa
para registrar el contacto. El permiso de Mi Bolsa lo publica la propia
aplicación al arrancar; aquí no se escribe ninguna asignación. Solo vale para
copias con datos inventados.
"""
import http.client
import json
import os
import re
import ssl
import subprocess

DATOS = 'deploy/principal/candidata_hz39/datos'
GUIONES = ('organizacion_520_administrativo.sql', 'antonio_disponible.sql', 'cargo_manuel.sql')
PREPARADOR = 'deploy/principal/preparar_candidato_desarrollo.sh'
IDENTIDAD = 'antonio-reyes'
# Contacto ficticio. El buzón de la copia solo admite el dominio reservado
# example.test (copia_local_correo.py), así que el aviso nunca sale fuera.
CONTACTO = {'correo': 'antonio.reyes@example.test', 'telefono_1': '+34 612 345 678',
            'motivo': 'Contacto del aspirante para los avisos de llamamiento'}
CLAVE_CONTACTO = 'copia-local:caso-4-0:contacto-antonio:v1'
REFERENCIA = re.compile(r'[A-Za-z0-9:._-]{1,256}')


def seleccion(repo):
    """Referencias del caso: organismo, bolsa, candidatura y participación de Antonio."""
    datos = json.loads((repo / DATOS / 'seleccion.json').read_text())
    aspirante = datos['aspirante']
    valores = dict(organismo=datos['organismo_ref'], bolsa=datos['bolsa_ref'],
                   candidato=aspirante['candidato_ref'], participacion=aspirante['participacion_ref'],
                   nombre=aspirante['nombre'])
    for clave, valor in valores.items():
        if clave != 'nombre' and not (isinstance(valor, str) and REFERENCIA.fullmatch(valor)):
            raise ValueError('seleccion.json del caso 4.0 trae una referencia no válida')
    return valores


def aplicar_guiones(copia, repo):
    """Aplica los tres guiones de datos; cada uno es idempotente y transaccional."""
    sel = seleccion(repo)
    with (copia.raiz / 'caso40-datos.log').open('ab') as log:
        for nombre in GUIONES:
            contenido = (repo / DATOS / nombre).read_bytes()
            subprocess.run(['docker', 'exec', '-i', '-u', 'postgres', copia.nombre, 'psql', '-X', '-q',
                            '-v', 'ON_ERROR_STOP=1', '-v', 'organismo=' + sel['organismo'],
                            '-U', 'postgres', '-d', 'postgres'],
                           input=contenido, stdout=log, stderr=subprocess.STDOUT, check=True, timeout=120)


def preparar_antonio(copia, repo):
    """Identidad de Mi Bolsa de Antonio ligada a su candidatura de la bolsa vigente."""
    sel = seleccion(repo)
    estado = copia.leer()
    entorno = {**os.environ,
               'VEC_DEVELOPMENT_MATERIAL_DIR': str(copia.material),
               'VEC_PRINCIPAL_POSTGRES_CONTAINER': copia.nombre,
               'VEC_PRINCIPAL_APP_CONTAINER': copia.nombre,
               'VEC_CANDIDATE_BOLSA_REF': sel['bolsa'],
               'VEC_CANDIDATE_BASE_URL': f"https://localhost:{estado['puerto_app']}"}
    with (copia.raiz / 'caso40-antonio.log').open('ab') as log:
        subprocess.run(['bash', repo / PREPARADOR, '--motor', 'docker', '--identidad', IDENTIDAD,
                        '--nombre', sel['nombre'], '--candidato-ref', sel['candidato']],
                       env=entorno, stdout=log, stderr=subprocess.STDOUT, check=True, timeout=180)
    manifiesto = json.loads((copia.material / 'identidad/bolsa-candidato.json').read_text())
    if manifiesto.get('candidato_ref') != sel['candidato']:
        raise ValueError('La identidad de Antonio no quedó ligada a su candidatura vigente')
    # Con el portal del candidato, Antonio puede responder al llamamiento desde Mi Bolsa.
    return {'VEC_BOLSA_PORTAL_CANDIDATO_ENABLED': 'true'}


def peticion_contacto(sel):
    """Ruta y cuerpo del registro de contacto de la participación de Antonio."""
    ruta = (f"/api/vec/bolsa/bolsas/{sel['bolsa']}/candidatos/"
            f"{sel['participacion']}/datos-contacto")
    return ruta, json.dumps(CONTACTO, ensure_ascii=False).encode()


def registrar_contacto(copia, repo):
    """Registra con la técnica de RRHH el contacto de Antonio por la API de Bolsa."""
    estado = copia.leer()
    ruta, cuerpo = peticion_contacto(seleccion(repo))
    contexto = ssl.create_default_context(cafile=str(copia.material / 'ca/ca.crt'))
    contexto.load_cert_chain(str(copia.material / 'mtls/cliente.crt'), str(copia.material / 'mtls/cliente.key'))
    conexion = http.client.HTTPSConnection('localhost', estado['puerto_app'], context=contexto, timeout=30)
    try:
        conexion.request('POST', ruta, body=cuerpo, headers={
            'Origin': f"https://localhost:{estado['puerto_app']}", 'Content-Type': 'application/json',
            'Accept': 'application/json', 'Idempotency-Key': CLAVE_CONTACTO})
        respuesta = conexion.getresponse()
        respuesta.read()
    finally:
        conexion.close()
    # 201 la primera vez; 200 si la misma clave ya lo había registrado.
    if respuesta.status not in (200, 201):
        raise ValueError(f'El contacto de Antonio no se registró (HTTP {respuesta.status})')
    return respuesta.status
