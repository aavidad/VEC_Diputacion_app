#!/usr/bin/env python3
"""Restaura el esquema HZ37 y ensaya la lista SQL en PostgreSQL 18 aislado."""

import argparse
import gzip
import hashlib
import json
import os
from pathlib import Path
import secrets
import shutil
import subprocess
import sys
import time

RAIZ = Path(__file__).resolve().parents[3]
FASE = Path(__file__).resolve().parent
IMAGEN = 'postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296'
BASE = Path('/dev/shm')
D15_SQL = 'deploy/postgresql/documentos/migraciones/000015_rechazo_expediente_ct_cero.up.sql'
PRUEBAS_SQL = (
    'deploy/postgresql/bolsa_llamamientos/pruebas_sql/b118_criterios_sin_salto.sql',
    'deploy/postgresql/bolsa_llamamientos/pruebas_sql/b144_ambito_gobernado_bolsa.sql',
    'deploy/postgresql/documentos/pruebas_sql/000014_referencia_expediente_tipado.sql',
    'deploy/postgresql/documentos/pruebas_sql/000015_rechazo_expediente_ct_cero.sql',
    'deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/ad180_rechazo_sin_consumo.sql',
    'deploy/postgresql/identidad_sesiones_v1/pruebas_sql/000021_seleccion_perfil_bolsa_rrhh.sql',
    'deploy/postgresql/contexto_actor_v1/pruebas_sql/000046_perfiles_propios_bolsa_rrhh.sql',
    'deploy/postgresql/bolsa_convocatorias/pruebas_sql/bc12_rechazo_inscripcion_conjunta_v1.sql',
    'deploy/postgresql/bolsa_convocatorias/pruebas_sql/bc14_publicacion_oficial_bases.sql',
    'deploy/postgresql/bolsa_convocatorias/pruebas_sql/bc15_declaracion_publicacion_oficial_rrhh.sql',
    'deploy/postgresql/bolsa_llamamientos/pruebas_sql/b142_modalidad_inscripcion_aspirante.sql',
    # El último caso es el fixture de la concurrencia de contexto externo.
    'deploy/postgresql/contexto_actor_v1/pruebas_sql/contexto_externo_segregado_v2.sql',
)


def memoria_disponible():
    for linea in Path('/proc/meminfo').read_text().splitlines():
        if linea.startswith('MemAvailable:'):
            return int(linea.split()[1]) * 1024
    raise ValueError('No se pudo medir la memoria disponible')


def comprobar_destino(valor):
    destino = Path(valor)
    if (not destino.is_absolute() or not destino.is_dir()
            or destino.resolve(strict=True) != destino
            or not destino.is_relative_to(BASE)):
        raise ValueError('VEC_PG18_DATOS_DIR debe ser un directorio real bajo /dev/shm')
    if shutil.disk_usage(destino).free < 2 * 1024**3:
        raise ValueError('Faltan 2 GiB libres en la memoria de trabajo')
    en_ci = os.environ.get('GITHUB_ACTIONS') == 'true'
    minimo = (3 if en_ci else 34) * 1024**3
    if memoria_disponible() < minimo:
        raise ValueError(f'Se requieren {3 if en_ci else 34} GiB disponibles antes de restaurar')
    return destino


def preparar_esquema(destino):
    manifiesto = json.loads((FASE / 'esquema_hz37.json').read_text())
    comprimido = FASE / 'esquema_hz37.sql.gz'
    if (manifiesto.get('referencia') != 'HZ37' or manifiesto.get('postgresql') != '18'
            or manifiesto.get('solo_esquema') is not True
            or manifiesto.get('sin_datos') is not True):
        raise ValueError('El manifiesto HZ37 no describe un esquema sin datos')
    with comprimido.open('rb') as fuente:
        huella_gzip = hashlib.file_digest(fuente, 'sha256').hexdigest()
    if huella_gzip != manifiesto['sha256_gzip']:
        raise ValueError('La huella del esquema comprimido no coincide')
    salida = destino / 'esquema.sql'
    huella = hashlib.sha256()
    longitud = 0
    with gzip.open(comprimido, 'rb') as fuente, salida.open('xb') as fichero:
        while bloque := fuente.read(1024 * 1024):
            longitud += len(bloque)
            if longitud > manifiesto['bytes_sql']:
                raise ValueError('El esquema descomprimido supera el tamaño previsto')
            huella.update(bloque)
            fichero.write(bloque)
    if longitud != manifiesto['bytes_sql'] or huella.hexdigest() != manifiesto['sha256_sql']:
        raise ValueError('La huella del esquema SQL no coincide')
    return salida


def contenido_versionado(relativa):
    ruta = RAIZ / relativa
    if ruta.resolve(strict=True) != ruta or not ruta.is_file():
        raise ValueError(f'El archivo debe ser el regular previsto: {relativa}')
    contenido = ruta.read_bytes()
    versionado = subprocess.check_output([
        'git', '-C', str(RAIZ), 'show', f'HEAD:{relativa}'], timeout=30)
    if contenido != versionado:
        raise ValueError(f'El archivo difiere de la cabeza versionada: {relativa}')
    return contenido


def preparacion_versionada():
    return contenido_versionado('deploy/postgresql/fase1/preparacion_fase1.sql')


def base_hz37_versionada():
    lista = contenido_versionado('deploy/postgresql/fase1/lista_sql_base_hz37.txt')
    filas = lista.decode('ascii').splitlines()
    if len(filas) != 1 or len(filas[0].split()) != 2:
        raise ValueError('La lista base HZ37 debe contener solo D15 y su huella')
    relativa, huella = filas[0].split()
    if relativa != D15_SQL or len(huella) != 64:
        raise ValueError('La lista base HZ37 no identifica D15')
    contenido = contenido_versionado(D15_SQL)
    if hashlib.sha256(contenido).hexdigest() != huella:
        raise ValueError('La huella de D15 no coincide')
    return relativa, contenido


def pruebas_versionadas():
    pruebas = []
    for relativa in PRUEBAS_SQL:
        pruebas.append((relativa, contenido_versionado(relativa)))
    return pruebas


def comandos_docker(*argumentos, entrada=None, timeout=180):
    return subprocess.run(['docker', *argumentos], input=entrada, capture_output=True,
                          check=True, timeout=timeout)


def contenedor_propio(nombre, etiqueta):
    consulta = subprocess.run(['docker', 'container', 'inspect', nombre], capture_output=True)
    if consulta.returncode:
        comandos_docker('info')  # Distinguir ausencia del contenedor y motor inaccesible.
        return False
    datos = json.loads(consulta.stdout)[0]
    if datos['Config'].get('Labels', {}).get('vec.fase1') != etiqueta:
        raise ValueError('El nombre del contenedor pertenece a otra ejecución')
    return True


def limpiar_copia(nombre, etiqueta, propio):
    if contenedor_propio(nombre, etiqueta):
        comandos_docker('rm', '-f', nombre)
    datos = propio / 'data'
    if datos.exists():
        try:
            shutil.rmtree(datos)
        except PermissionError:
            # La imagen inicia como root y PostgreSQL deja PGDATA propiedad de
            # postgres (uid 999). Montar exclusivamente los datos propios.
            comandos_docker('run', '--pull=never', '--rm', '--restart=no',
                            '--memory=64m', '--network=none',
                            '--mount', f'type=bind,src={datos},dst=/datos',
                            '--entrypoint', '/bin/sh', IMAGEN, '-ec',
                            'rm -rf -- /datos/* /datos/.[!.]* /datos/..?*')
            # El directorio queda vacío pero sigue siendo de uid 999 y modo
            # 0700. Su padre es nuestro: rmdir no necesita abrirlo.
            datos.rmdir()
    shutil.rmtree(propio)


def sql(nombre, contenido, etapa):
    try:
        return comandos_docker('exec', '-i', '-u', 'postgres', nombre, 'psql', '-X', '-q', '-A', '-t',
                        '-v', 'ON_ERROR_STOP=1', '-U', 'postgres', '-d', 'postgres',
                        entrada=contenido, timeout=300)
    except subprocess.CalledProcessError as exc:
        raise ValueError(f'Falló {etapa}: {exc.stderr.decode(errors="replace")[-1500:]}') from exc


TABLAS_USUARIO = b"""FROM pg_class AS c
JOIN pg_namespace AS n ON n.oid = c.relnamespace
WHERE (n.nspname = 'public' OR left(n.nspname, 4) = 'vec_')
  AND c.relkind IN ('r', 'm')
  AND NOT EXISTS (
    SELECT 1 FROM pg_depend AS d
    WHERE d.classid = 'pg_class'::regclass AND d.objid = c.oid AND d.deptype = 'e'
  )"""


def verificar_tablas_vacias(nombre):
    comprobacion = b"""DO $$
DECLARE tabla record; filas bigint; cantidad integer := 0;
BEGIN
  FOR tabla IN SELECT n.nspname AS esquema, c.relname AS nombre """ + TABLAS_USUARIO + b""" LOOP
    EXECUTE format('SELECT count(*) FROM %I.%I', tabla.esquema, tabla.nombre) INTO filas;
    IF filas <> 0 THEN
      RAISE EXCEPTION 'La tabla %.% contiene % filas', tabla.esquema, tabla.nombre, filas;
    END IF;
    cantidad := cantidad + 1;
  END LOOP;
  IF cantidad = 0 THEN RAISE EXCEPTION 'No se encontraron tablas de usuario'; END IF;
END $$;"""
    sql(nombre, comprobacion, 'la comprobación de esquema sin datos')
    conteo = sql(nombre, b'SELECT count(*) ' + TABLAS_USUARIO + b';',
                 'el recuento de tablas verificadas')
    cantidad = int(conteo.stdout.strip())
    if cantidad == 0:
        raise ValueError('No se encontraron tablas de usuario')
    print(f'Tablas sin filas verificadas: {cantidad}')


def esperar_postgresql(nombre, etiqueta):
    for _ in range(120):
        if not contenedor_propio(nombre, etiqueta):
            raise ValueError('PostgreSQL terminó durante el arranque')
        listo = subprocess.run(['docker', 'exec', '-u', 'postgres', nombre,
                                'pg_isready', '-h', '127.0.0.1', '-U', 'postgres'],
                               capture_output=True)
        if listo.returncode == 0:
            break
        time.sleep(0.5)
    else:
        raise ValueError('PostgreSQL no respondió en 60 segundos')


def ejecutar(solo_base=False, lista=None, datos_dir=None, preparacion=False, pruebas=False):
    sys.path.insert(0, str(RAIZ))
    from deploy.principal.copia_local_roles import extract, ROLES_B119
    from deploy.principal.copia_local import migraciones

    base = comprobar_destino(datos_dir or os.environ.get('VEC_PG18_DATOS_DIR', '/dev/shm'))
    if solo_base and (preparacion or pruebas):
        raise ValueError('--preparacion y --pruebas no corresponden a --solo-base')
    fixture = preparacion_versionada() if preparacion or pruebas else None
    base_hz37 = base_hz37_versionada() if fixture is not None else None
    casos = pruebas_versionadas() if pruebas else ()
    deltas = [] if solo_base else migraciones([lista or RAIZ / 'deploy/principal/lista_sql_main_tras_hz37.txt'])
    nombre = 'vec-fase1-' + secrets.token_hex(6)
    propio = base / nombre
    propio.mkdir(mode=0o700)
    etiqueta = str(propio)
    try:
        esquema = preparar_esquema(propio)
        (propio / 'data').mkdir(mode=0o700)
        roles = sorted(set(extract([esquema])) | ROLES_B119)
        sentencias = ''.join(f'CREATE ROLE "{rol.replace(chr(34), chr(34) * 2)}" NOLOGIN;\n'
                            for rol in roles).encode()
        comandos_docker('run', '--pull=never', '-d', '--rm', '--restart=no',
                        '--memory=2g', '--network=none', '--name', nombre,
                        '--label', f'vec.fase1={etiqueta}', '-e', 'PGDATA=/datos',
                        '-e', 'POSTGRES_HOST_AUTH_METHOD=trust',
                        '-v', f'{propio / "data"}:/datos', IMAGEN)
        esperar_postgresql(nombre, etiqueta)
        sql(nombre, sentencias, 'la creación de roles sin LOGIN')
        sql(nombre, esquema.read_bytes(), 'la restauración del esquema HZ37')
        verificar_tablas_vacias(nombre)
        if fixture is not None:
            sql(nombre, fixture, 'la preparación versionada de la fase 1')
            relativa, contenido = base_hz37
            sql(nombre, contenido, f'la reconciliación HZ37 {relativa}')
        for ruta, _huella, contenido in deltas:
            sql(nombre, contenido, f'la migración {ruta.relative_to(RAIZ)}')
        for relativa, contenido in casos:
            sql(nombre, contenido, f'la prueba {relativa}')
        if pruebas:
            # Reiniciar el mismo PGDATA: no reinstalar SQL ni preparar datos.
            if not contenedor_propio(nombre, etiqueta):
                raise ValueError('El contenedor propio no está disponible para reiniciar')
            comandos_docker('restart', '--time', '10', nombre)
            esperar_postgresql(nombre, etiqueta)
            for relativa, contenido in casos:
                sql(nombre, contenido, f'la prueba tras reiniciar {relativa}')
            if not contenedor_propio(nombre, etiqueta):
                raise ValueError('El contenedor propio no está disponible para la concurrencia')
            from deploy.postgresql.fase1 import concurrencia_contexto
            fixture_externo = dict(casos)[PRUEBAS_SQL[-1]]
            concurrencia_contexto.ensayar(nombre, etiqueta, fixture_externo)
        print(f'Esquema HZ37 restaurado; migraciones ensayadas: {len(deltas)}')
        if base_hz37 is not None:
            print('Reconciliación de la copia HZ37: Documentos D15 instalada')
        if pruebas:
            print(f'Pruebas SQL superadas: {len(casos)} antes y {len(casos)} después del reinicio')
            print('Concurrencia CAS de contexto externo superada')
    finally:
        limpiar_copia(nombre, etiqueta, propio)


def main():
    os.umask(0o077)
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--solo-base', action='store_true')
    parser.add_argument('--lista', type=Path)
    parser.add_argument('--datos-dir', type=Path)
    parser.add_argument('--preparacion', action='store_true')
    parser.add_argument('--pruebas', action='store_true')
    args = parser.parse_args()
    try:
        ejecutar(args.solo_base, args.lista, args.datos_dir, args.preparacion, args.pruebas)
    except (OSError, ValueError, subprocess.SubprocessError, KeyError,
            gzip.BadGzipFile, json.JSONDecodeError) as exc:
        print(f'Ensayo detenido: {exc}', file=sys.stderr)
        return 1
    return 0


if __name__ == '__main__':
    sys.exit(main())
