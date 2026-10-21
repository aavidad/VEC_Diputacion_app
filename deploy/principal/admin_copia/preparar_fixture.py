#!/usr/bin/env python3
"""Vincula la plantilla HZ37 a una copia cotejada por el conductor."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import sys

HZ37_SHA256 = '7ae0613fa174aebc8aa36c079a6c4b268b771df41c9310129ee9dfae9650cc45'


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def preparar(args):
    if not re.fullmatch(r'[0-9]{15,20}', args.sid_copia) or int(args.sid_copia) <= 0:
        raise ValueError('SID de copia inválido')
    if not args.salida.is_absolute() or not args.salida.is_dir() or args.salida.is_symlink() or args.salida.stat().st_mode & 0o077:
        raise ValueError('carpeta privada inválida')
    for path in (args.respaldo_hz37, args.preimagen_admin):
        if not path.is_absolute() or not path.is_file() or path.is_symlink() or path.stat().st_mode & 0o077:
            raise ValueError('respaldo privado ausente')
    if digest(args.respaldo_hz37) != HZ37_SHA256:
        raise ValueError('el respaldo no es HZ37 exacto')
    root = Path(__file__).resolve().parent
    manifest = root / 'manifiesto-kit.json'
    template = root / 'fixture-admin-hz37.sql.in'
    if not manifest.is_file() or manifest.is_symlink() or not template.is_file() or template.is_symlink():
        raise ValueError('kit o plantilla ausentes')
    document = json.loads(manifest.read_text())
    if not isinstance(document, dict) or not isinstance(document.get('textos_sha256'), dict):
        raise ValueError('manifiesto de kit inválido')
    expected = document['textos_sha256'].get(template.name)
    if not isinstance(expected, str) or not re.fullmatch(r'[0-9a-f]{64}', expected) or digest(template) != expected:
        raise ValueError('plantilla del kit divergente')
    source = template.read_text()
    if source.count('@@SYSTEM_IDENTIFIER@@') != 1:
        raise ValueError('plantilla no cotejada')
    sql = args.salida / 'fixture-admin-hz37.sql'
    with sql.open('x') as stream:
        stream.write(source.replace('@@SYSTEM_IDENTIFIER@@', args.sid_copia))
    sql.chmod(0o600)
    evidence = args.salida / 'fixture-admin-hz37-preimagen.json'
    with evidence.open('x') as stream:
        json.dump({'system_identifier': args.sid_copia,
                   'backup_hz37_sha256': HZ37_SHA256,
                   'preimagen_admin_sha256': digest(args.preimagen_admin),
                   'sql_sha256': digest(sql)}, stream, indent=1)
        stream.write('\n')
    evidence.chmod(0o600)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--sid-copia', required=True, help='SID cotejado por el conductor')
    parser.add_argument('--respaldo-hz37', required=True, type=Path)
    parser.add_argument('--preimagen-admin', required=True, type=Path)
    parser.add_argument('--salida', required=True, type=Path)
    args = parser.parse_args()
    try:
        preparar(args)
    except (ValueError, OSError) as exc:
        print('Fixture ADMIN detenido: ' + str(exc), file=sys.stderr)
        return 1
    print('Fixture HZ37 preparado; SQL pendiente de revisión y ejecución')
    return 0


if __name__ == '__main__':
    sys.exit(main())
