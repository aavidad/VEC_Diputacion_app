#!/usr/bin/env python3
"""Prepara solo un JSON privado; la CLI autentica después plan y acuse."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import stat
from preparar_runtime import validar_copia, validar_entrada, validar_gobierno

AUDIENCIAS = frozenset((
    'vec_autorizacion.versionar_rol_bolsa_externa.propuesta.v1',
    'vec_autorizacion.versionar_rol_bolsa_externa.cierre.v1',
))


def privado(path, directorio=False):
    path = Path(path)
    st = path.lstat()
    if (not path.is_absolute() or path.is_symlink() or path.resolve(strict=True) != path
            or st.st_uid != os.getuid() or st.st_mode & 0o077
            or (not stat.S_ISDIR(st.st_mode) if directorio else not stat.S_ISREG(st.st_mode))
            or (not directorio and st.st_nlink != 1)):
        raise ValueError('ruta privada insegura')
    return path


def documento(path):
    value = json.loads(privado(path).read_text())
    if not isinstance(value, dict):
        raise ValueError('documento privado inválido')
    return value


def motivos_validos(value):
    if not isinstance(value, dict) or set(value) != AUDIENCIAS:
        return False
    for ref in value.values():
        if (not isinstance(ref, dict) or set(ref) != {'catalogo_id', 'catalogo_version',
                'catalogo_huella_sha256', 'entrada_clave'}
                or not isinstance(ref['catalogo_id'], str) or not ref['catalogo_id']
                or type(ref['catalogo_version']) is not int or ref['catalogo_version'] < 1
                or not isinstance(ref['catalogo_huella_sha256'], str)
                or not re.fullmatch(r'[0-9a-f]{64}', ref['catalogo_huella_sha256'])
                or not isinstance(ref['entrada_clave'], str) or not ref['entrada_clave']):
            return False
    return True


def usuario_dsn(value):
    if not isinstance(value, str):
        raise ValueError('DSN inválido')
    found = re.findall(r'(?:^| )user=([a-z][a-z0-9_]{2,62})(?: |$)', value)
    if len(found) != 1:
        raise ValueError('usuario de pool ambiguo')
    return found[0]


def crear(path, data):
    raw = (json.dumps(data, ensure_ascii=False, sort_keys=True,
                      separators=(',', ':')) + '\n').encode()
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, 'wb') as out:
        out.write(raw)
        out.flush()
        os.fsync(out.fileno())
    return hashlib.sha256(raw).hexdigest()


def preparar(run, dia, motivos_path):
    run = privado(run, True)
    privado(run / 'vec-admin', True)
    privado(run / 'vec-admin/pools', True)
    if not re.fullmatch(r'[0-9]{8}[a-z0-9]{1,12}', dia):
        raise ValueError('día de gobierno inválido')
    entrada = documento(run / 'entrada-vec-admin.json')
    if entrada.get('dia_gobierno') != dia:
        raise ValueError('día de gobierno ajeno a la entrada')
    kit = privado(Path(entrada['kit_dir']), True)
    contrato = documento(kit / 'contrato-copia-local-admin-v1.json')
    validar_copia(run, kit, entrada.get('sid_copia'))
    validar_entrada(entrada, run, entrada['sid_copia'])
    validar_gobierno(run, dia, aplicado=True)
    if (contrato.get('version') != 2 or contrato.get('conjunto_capacidades') != 7
            or entrada.get('sid_copia') != contrato.get('sid_copia')
            or entrada.get('runner') != str(run)
            or not isinstance(entrada.get('pg_port'), int)):
        raise ValueError('entrada de overlay ajena a la copia')
    original = documento(run / ('config-gobierno-' + dia + '.json'))
    # Este filtro es solo de forma. La CLI Go autentica acuse, plan y material
    # antes de escribir un overlay; este preparador no afirma autoridad.
    acuse = documento(run / 'gobierno' / dia / 'acuse-aplicar.json')
    motivos = documento(motivos_path)
    if (original.get('conjunto_capacidades') != 7
            or original.get('horas_validez_claves') != contrato['vigencia_dias'] * 24
            or original.get('salida') != str(run / 'gobierno' / dia)
            or acuse.get('estado') != 'permitido'
            or not isinstance(acuse.get('recibo'), dict)
            or not acuse['recibo'].get('auditoria_ref')
            or not motivos_validos(motivos)):
        raise ValueError('gobierno, acuse o motivos externos inválidos')
    names = documento(run / 'vec-admin/logins.json')
    roles = ('version_bolsa_externa_ejecutor', 'catalogo_acciones_lector_externa')
    paths = (run / 'vec-admin/pools/version-bolsa-externa-ejecutor.json',
             run / 'vec-admin/pools/catalogo-acciones-lector-externa.json')
    users = []
    for key, path in zip(roles, paths):
        p = documento(path)
        if set(p) != {'dsn'}:
            raise ValueError('pool externo inválido')
        dsn = p['dsn']
        expected = (f"host={entrada['pg_host']} port={entrada['pg_port']} dbname={entrada['pg_base']} "
                    f"user={names.get(key, '')} ")
        if (not isinstance(dsn, str) or not dsn.startswith(expected)
                or ' sslmode=verify-full sslrootcert=' + str(entrada.get('pg_ca', '')) not in dsn):
            raise ValueError('pool externo de otra copia')
        users.append(usuario_dsn(dsn))
    if users[0] == users[1]:
        raise ValueError('LOGIN externos no segregados')
    target = run / ('config-gobierno-externo-' + dia + '.json')
    if target.exists() or target.is_symlink():
        raise ValueError('configuración externa ya preparada')
    nuevo = dict(original)
    if 'version_bolsa_externa' in nuevo:
        raise ValueError('overlay externo duplicado')
    nuevo['version_bolsa_externa'] = {
        'pool_gobierno': str(paths[0]), 'pool_catalogo': str(paths[1]), 'motivos': motivos}
    return target, crear(target, nuevo)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--run', required=True, type=Path)
    parser.add_argument('--dia', required=True)
    parser.add_argument('--motivos-publicados', required=True, type=Path)
    a = parser.parse_args()
    try:
        target, sha = preparar(a.run, a.dia, a.motivos_publicados)
    except (ValueError, OSError, KeyError, json.JSONDecodeError) as error:
        parser.error(str(error))
    print('JSON externo preparado; SHA-256=' + sha)
    print('Revisar y usar como -config de la fase version-bolsa-externa: ' + str(target))


if __name__ == '__main__':
    main()
