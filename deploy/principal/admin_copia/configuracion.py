#!/usr/bin/env python3
"""Prepara configuración privada H10; no ejecuta SQL ni actos."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import secrets
import shutil
import stat
import sys

BINARIOS = ('h-root-seed.test', 'vec-gobierno-usuarios-admin', 'vec-admin',
            'vec-preparar-fuentes-admin', 'vec-aplicar-fuentes-admin',
            'vec-preparar-unidad-admin', 'vec-preparar-admin-bootstrap',
            'vec-aplicar-admin-bootstrap', 'helper-hmac')
CATALOGOS = {
    'fuentes_preparar': ('vec-preparar-fuentes-admin', 'admin-fuentes-preparar.json'),
    'fuentes_aplicar': ('vec-aplicar-fuentes-admin', 'admin-fuentes-aplicar.json'),
    'unidad': ('vec-preparar-unidad-admin', 'admin-unidad-preparar.json'),
    'bootstrap_preparar': ('vec-preparar-admin-bootstrap', 'admin-comprobar-perfil.json'),
    'bootstrap_aplicar': ('vec-aplicar-admin-bootstrap', 'admin-bootstrap-aplicar.json'),
}


def exigir(ok, motivo):
    if not ok:
        raise ValueError(motivo)


def digest(path):
    with Path(path).open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()


def archivo(path, directorio=False, privado=True):
    path = Path(path)
    exigir(path.is_absolute() and not path.is_symlink() and path.resolve(strict=True) == path,
           'ruta privada inválida')
    st = path.lstat()
    exigir(st.st_uid in (os.getuid(), 0) and
           not st.st_mode & (0o077 if privado else 0o022) and
           (stat.S_ISDIR(st.st_mode) if directorio else
            stat.S_ISREG(st.st_mode) and st.st_nlink == 1), 'entrada privada insegura')
    return path


def leer_json(path):
    document = json.loads(archivo(path).read_text())
    exigir(isinstance(document, dict), 'documento JSON inválido')
    return document


def escribir(path, value):
    raw = value if isinstance(value, str) else json.dumps(value, ensure_ascii=False, indent=1) + '\n'
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, 'w') as stream:
        stream.write(raw)
        stream.flush()
        os.fsync(stream.fileno())


def citar_dsn(value):
    return "'" + str(value).replace('\\', '\\\\').replace("'", "\\'") + "'"


def validar_kit(kit, revision, sid, puerto_admin, catalogo):
    kit = archivo(kit, True)
    manifest = leer_json(kit / 'manifiesto-kit.json')
    build = leer_json(kit / 'build-config.json')
    contract = leer_json(kit / 'contrato-copia-local-admin-v1.json')
    exigir(manifest.get('estado') == 'compilado_sin_actos' and
           manifest.get('revision') == revision == build.get('repo_revision') == contract.get('revision_candidata') and
           manifest.get('sid_copia') == sid == build.get('sid_copia') == contract.get('sid_copia') and
           manifest.get('puerto_admin') == puerto_admin == contract.get('puerto_admin') and
           manifest.get('conjunto_capacidades') == contract.get('conjunto_capacidades') == 7 and
           contract.get('catalogo_entrada_unidad') == catalogo and
           manifest.get('contrato_sha256') == digest(kit / 'contrato-copia-local-admin-v1.json'),
           'kit, revisión, SID, puerto o conjunto divergentes')
    exigir(manifest.get('fuente_sha256') == digest(kit / 'source-manifest.json'),
           'manifiesto de fuentes alterado')
    for name, sha in manifest.get('textos_sha256', {}).items():
        exigir(name in ('arranque.py', 'configuracion.py', 'guardar_preimagen.py',
                       'preparar_fixture.py', 'plantilla-config.json',
                       'fixture-admin-hz37.sql.in', 'derivar_raiz_test.go') and
               digest(archivo(kit / name)) == sha, 'texto del kit divergente')
    exigir(len(manifest.get('textos_sha256', {})) == 7, 'textos del kit incompletos')
    for name in BINARIOS:
        relative = 'bin/' + name
        exigir(manifest.get('binarios_sha256', {}).get(relative) == digest(archivo(kit / relative)),
               'binario del kit divergente')
    for name, sha in contract.get('scripts_sha256', {}).items():
        exigir(name in ('arranque.py', 'guardar_preimagen.py', 'preparar_fixture.py') and
               digest(kit / name) == sha, 'script de contrato divergente')
    exigir(len(contract.get('scripts_sha256', {})) == 3, 'scripts de contrato incompletos')
    for name, sha in contract.get('bin_sha256', {}).items():
        exigir(name in ('bin/h-root-seed.test', 'bin/vec-gobierno-usuarios-admin',
                        'bin/vec-admin') and digest(kit / name) == sha,
               'binario de contrato divergente')
    exigir(len(contract.get('bin_sha256', {})) == 3, 'binarios de contrato incompletos')
    source_manifest = leer_json(kit / 'source-manifest.json')
    source = archivo(kit / 'source', True)
    expected = source_manifest.get('files')
    exigir(source_manifest.get('version') == 1 and isinstance(expected, dict) and
           len(expected) > 1000, 'manifiesto de fuentes inválido')
    actual = set()
    for path in source.rglob('*'):
        if path.is_dir() and not path.is_symlink():
            continue
        relative = path.relative_to(source).as_posix()
        actual.add(relative)
        exigir(relative in expected and digest(archivo(path)) == expected[relative],
               'fuente del kit divergente')
    exigir(actual == set(expected), 'fuentes del kit incompletas')
    return manifest, build, contract


def preparar(args):
    exigir(re.fullmatch(r'[a-z0-9_]{3,18}', args.instancia) and
           re.fullmatch(r'[a-z0-9-]{6,80}', args.contenedor), 'instancia o contenedor inválidos')
    exigir(1024 <= args.puerto_pg <= 65535 and 1024 <= args.puerto_admin <= 65535 and
           re.fullmatch(r'[a-z0-9_]{3,80}', args.catalogo_entrada_unidad),
           'puerto o catálogo inválidos')
    exigir(re.fullmatch(r'[0-9a-f]{40}', args.revision) and
           re.fullmatch(r'[0-9]{15,20}', args.sid_copia), 'revisión o SID inválidos')
    exigir(60 <= args.dias_perfiles < args.dias_personas <= args.dias_organizacion <= 90 and
           60 <= args.dias_unidad <= 90 and args.dias_ca in (397, 400),
           'vigencias incompatibles con H10')
    archivo(args.backup_admin)
    archivo(args.ca_pg, privado=False)
    material = archivo(args.material, True)
    archivo(material / 'idempotencia/configuracion.json')
    manifest, build, contract = validar_kit(args.kit, args.revision, args.sid_copia,
                                             args.puerto_admin, args.catalogo_entrada_unidad)
    output = args.salida
    exigir(output.is_absolute() and not output.exists() and not output.is_symlink() and
           not output.is_relative_to(args.kit) and output.parent.is_dir() and
           not output.parent.is_symlink() and output.parent.resolve() == output.parent and
           output.parent.stat().st_mode & 0o077 == 0,
           'salida debe ser carpeta absoluta, nueva y privada')
    output.mkdir(mode=0o700)
    for name in ('dba', 'aprobaciones'):
        (output / name).mkdir(mode=0o700)
    shutil.copytree(args.kit / 'source', output / 'source')
    (output / 'source').chmod(0o700)
    c = leer_json(args.kit / 'plantilla-config.json')
    c['backup'] = {'ref': 'preimagen:admin:' + args.instancia,
                   'sha256': digest(args.backup_admin)}
    c['privado'] = str(output)
    c['psql_comando'] = ['/usr/bin/docker', 'exec', '-i', '-u', 'postgres',
                         args.contenedor, 'psql', '-X', '-U', 'postgres',
                         '-d', 'postgres', '-At', '-v', 'ON_ERROR_STOP=1']
    c['proveedor']['directorio_material'] = str(material)
    c['proveedor']['ruta_configuracion_hmac'] = str(material / 'idempotencia/configuracion.json')
    c['fuentes']['politica_aprobacion'] = str(output / 'aprobaciones/politica.json')
    c['fuentes']['host_admin'] = build['host_admin']
    c['fuentes']['ca_dias'] = args.dias_ca
    c['fuentes']['ttl_personas_segundos'] = args.dias_personas * 86400
    c['fuentes']['ttl_organizacion_segundos'] = args.dias_organizacion * 86400
    c['unidad']['catalogo_entrada_clave'] = args.catalogo_entrada_unidad
    c['unidad']['vigencia_dias'] = args.dias_unidad
    c['bootstrap']['ttl_perfiles_segundos'] = args.dias_perfiles * 86400
    c['kit'] = {'source_snapshot': str(args.kit / 'source'),
                'source_copy': str(output / 'source'),
                'source_manifest': str(args.kit / 'source-manifest.json'),
                'source_manifest_sha256': manifest['fuente_sha256'],
                'modcache': build['modcache'], 'gocache_cache': build['gocache'],
                'go': build['go'], 'openssl': '/usr/bin/openssl',
                'helper_hmac': str(args.kit / 'bin/helper-hmac'),
                'helper_hmac_sha256': manifest['binarios_sha256']['bin/helper-hmac'],
                'cli': {}, 'cli_sha256': {}, 'catalogos': {}, 'catalogo_sha256': {}}
    for key, (binary, catalog) in CATALOGOS.items():
        path = args.kit / 'source/web/static/textos/es' / catalog
        archivo(path)
        c['kit']['cli'][key] = str(args.kit / 'bin' / binary)
        c['kit']['cli_sha256'][key] = manifest['binarios_sha256']['bin/' + binary]
        c['kit']['catalogos'][key] = str(path)
        c['kit']['catalogo_sha256'][key] = digest(path)
    for phase in ('fuentes', 'unidad', 'bootstrap'):
        d = c['dba'][phase]
        d['login'] = 'vec_adm_' + phase + '_' + args.instancia
        d['aprobacion_ref'] = 'aprobacion:' + phase + ':' + args.instancia + ':' + secrets.token_hex(16)
        d['conexion'] = str(output / 'dba' / (phase + '-conexion.json'))
        d['aprobacion_externa'] = str(output / 'aprobaciones' / (phase + '.json'))
        if 'proceso' in d:
            d['proceso'] = 'vec-adm-' + phase + '-' + args.instancia
        password = secrets.token_urlsafe(32)
        dsn = (f'host=127.0.0.1 port={args.puerto_pg} dbname=postgres user={d["login"]} '
               f'password={password} sslmode=verify-full sslrootcert={citar_dsn(args.ca_pg)}')
        escribir(d['conexion'], {'dsn': dsn, 'permitir_socket_desarrollo': False})
        escribir(output / 'dba' / (phase + '-password.sql'),
                f"ALTER ROLE {d['login']} PASSWORD '{password}';\n")
    escribir(output / 'config.json', c)
    escribir(Path(c['fuentes']['politica_aprobacion']),
            {'entorno': 'desarrollo', 'alcance_fuente': 'sintetico_declarado',
             'finalidad': 'arranque_admin_ficticio_copia_autorizada',
             'backup_ref': c['backup']['ref']})
    escribir(output / 'contrato-arranque.json', {'version': 1, 'revision': args.revision,
            'sid_copia': args.sid_copia, 'puerto_admin': args.puerto_admin,
            'conjunto_capacidades': 7, 'kit_manifest_sha256': digest(args.kit / 'manifiesto-kit.json'),
            'kit_contract_sha256': manifest['contrato_sha256'],
            'vigencias_dias': {'organizacion': args.dias_organizacion,
                               'personas': args.dias_personas,
                               'unidad': args.dias_unidad,
                               'perfiles': args.dias_perfiles,
                               'x509': args.dias_ca},
            'config_sha256': digest(output / 'config.json')})
    return contract


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--kit', type=Path, required=True)
    parser.add_argument('--revision', required=True)
    parser.add_argument('--sid-copia', required=True)
    parser.add_argument('--puerto-admin', type=int, required=True)
    parser.add_argument('--instancia', required=True)
    parser.add_argument('--contenedor', required=True)
    parser.add_argument('--puerto-pg', type=int, required=True)
    parser.add_argument('--backup-admin', type=Path, required=True)
    parser.add_argument('--material', type=Path, required=True)
    parser.add_argument('--ca-pg', type=Path, required=True)
    parser.add_argument('--catalogo-entrada-unidad', required=True)
    parser.add_argument('--salida', type=Path, required=True)
    parser.add_argument('--dias-organizacion', type=int, default=90)
    parser.add_argument('--dias-personas', type=int, default=89)
    parser.add_argument('--dias-unidad', type=int, default=89)
    parser.add_argument('--dias-perfiles', type=int, default=88)
    parser.add_argument('--dias-ca', type=int, choices=(397, 400), default=400)
    args = parser.parse_args()
    try:
        preparar(args)
    except (ValueError, OSError, KeyError, TypeError, json.JSONDecodeError) as exc:
        print('Configuración ADMIN detenida: ' + str(exc), file=sys.stderr)
        return 1
    print('Configuración ADMIN privada preparada; aprobaciones externas pendientes')
    return 0


if __name__ == '__main__':
    sys.exit(main())
