#!/usr/bin/env python3
"""Congela una revisión Git y prepara el kit ADMIN de una copia local."""
import argparse
import ast
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import stat
import subprocess
import sys
import tarfile

TEXTOS = {
    'arranque.py': 'arranque.py',
    'configuracion.py': 'configuracion.py',
    'guardar_preimagen.py': 'guardar_preimagen.py',
    'preparar_fixture.py': 'preparar_fixture.py',
    'preparar_gobierno.py': 'preparar_gobierno.py',
    'preparar_runtime.py': 'preparar_runtime.py',
    'preparar_vec_admin.py': 'preparar_vec_admin.py',
    'preparar_overlay_externo.py': 'preparar_overlay_externo.py',
    'datos/plantilla_arranque.json': 'plantilla-config.json',
    'datos/plantilla_runtime.json': 'plantilla-runtime.json',
    'datos/fixture_hz37.sql.in': 'fixture-admin-hz37.sql.in',
    'datos/derivar_raiz_test.go.in': 'derivar_raiz_test.go',
}
BINARIOS = ('h-root-seed.test', 'vec-gobierno-usuarios-admin', 'vec-admin',
            'vec-preparar-fuentes-admin', 'vec-aplicar-fuentes-admin',
            'vec-preparar-unidad-admin', 'vec-preparar-admin-bootstrap',
            'vec-aplicar-admin-bootstrap', 'helper-hmac')
CONTRATO_BIN = ('h-root-seed.test', 'vec-gobierno-usuarios-admin', 'vec-admin')
PAQUETES = {'cmd', 'internal', 'web', 'config', 'locales', 'pruebas'}


def exigir(ok, motivo):
    if not ok:
        raise ValueError(motivo)


def seguro(path, directorio=False):
    path = Path(path)
    exigir(path.is_absolute() and not path.is_symlink() and path.resolve(strict=True) == path,
           'ruta ausente, relativa o enlazada')
    st = path.lstat()
    exigir(st.st_uid in (os.getuid(), 0) and not st.st_mode & 0o002,
           'propietario o permisos inseguros')
    exigir(stat.S_ISDIR(st.st_mode) if directorio else
           stat.S_ISREG(st.st_mode) and st.st_nlink == 1, 'tipo de archivo inseguro')
    return path


def digest(path):
    with Path(path).open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()


def guardar(path, data):
    raw = data if isinstance(data, bytes) else (json.dumps(data, ensure_ascii=False,
           sort_keys=True, separators=(',', ':')) + '\n').encode()
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, 'wb') as stream:
        stream.write(raw)
        stream.flush()
        os.fsync(stream.fileno())


def ejecutar(cmd, **kwargs):
    result = subprocess.run(cmd, stdin=subprocess.DEVNULL, capture_output=True,
                            timeout=600, check=False, **kwargs)
    exigir(result.returncode == 0, 'Git o compilación local fallida')
    return result.stdout


def estado_git(repo):
    head = ejecutar(['git', '-C', str(repo), 'rev-parse', '--verify', 'HEAD']).decode().strip()
    limpio = ejecutar(['git', '-C', str(repo), 'status', '--porcelain=v1',
                       '--untracked-files=all'])
    exigir(re.fullmatch(r'[0-9a-f]{40}', head) and not limpio,
           'la candidata debe tener HEAD limpio y estable')
    return head


def extraer_head(repo, revision, source):
    """Extrae exclusivamente archivos regulares de los paquetes admitidos."""
    process = subprocess.Popen(['git', '-C', str(repo), 'archive', '--format=tar', revision],
                               stdin=subprocess.DEVNULL, stdout=subprocess.PIPE,
                               stderr=subprocess.PIPE)
    hashes = {}
    total = 0
    try:
        with tarfile.open(fileobj=process.stdout, mode='r|') as archive:
            for member in archive:
                rel = Path(member.name)
                exigir(not rel.is_absolute() and rel.parts and
                       all(part not in ('', '.', '..') for part in rel.parts),
                       'ruta del archivo Git insegura')
                if member.isdir():
                    continue
                exigir(member.isfile(), 'entrada Git enlazada o especial')
                name = rel.as_posix()
                if rel.parts[0] not in PAQUETES and name not in ('go.mod', 'go.sum'):
                    continue
                exigir(member.isfile() and name not in hashes and
                       0 <= member.size <= 50_000_000, 'fuente Git enlazada o insegura')
                total += member.size
                exigir(total <= 1_000_000_000, 'archivo Git demasiado grande')
                target = source / rel
                target.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
                entry = archive.extractfile(member)
                exigir(entry is not None, 'fuente Git ilegible')
                fd = os.open(target, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
                with entry, os.fdopen(fd, 'wb') as out:
                    shutil.copyfileobj(entry, out)
                exigir(target.stat().st_size == member.size, 'fuente Git incompleta')
                hashes[name] = digest(target)
        exigir(process.wait(timeout=30) == 0 and len(hashes) > 1000 and
               'go.mod' in hashes and 'go.sum' in hashes, 'archivo Git incompleto')
        return hashes
    finally:
        if process.poll() is None:
            process.kill()
            process.wait()
        process.stdout.close()
        process.stderr.close()


def leer_textos_git(repo, revision):
    result = {}
    for relative, output_name in TEXTOS.items():
        path = f'deploy/principal/admin_copia/{relative}'
        entry = ejecutar(['git', '-C', str(repo), 'ls-tree', revision, '--', path]).decode()
        exigir(entry.startswith(('100644 blob ', '100755 blob ')) and
               entry.endswith('\t' + path + '\n'), 'texto Git enlazado o ausente')
        size = ejecutar(['git', '-C', str(repo), 'cat-file', '-s', f'{revision}:{path}'])
        exigir(int(size) <= 2_000_000, 'texto Git demasiado grande')
        result[output_name] = ejecutar(['git', '-C', str(repo), 'show', f'{revision}:{path}'])
    return result


def helper_go(source):
    tree = ast.parse(source)
    for node in tree.body:
        if isinstance(node, ast.Assign) and any(isinstance(target, ast.Name) and
            target.id == 'HMAC_GO' for target in node.targets):
            return ast.literal_eval(node.value).replace(
                '// Copiar a cmd/vec-fixture-originales-k/main.go en el checkout exacto antes de compilar.\n', '')
    raise ValueError('helper HMAC ausente en arranque.py')


def crear_kit(args):
    exigir(re.fullmatch(r'[0-9a-f]{40}', args.revision), 'revisión explícita inválida')
    exigir(re.fullmatch(r'[0-9]{15,20}', args.sid_copia) and int(args.sid_copia) > 0,
           'SID de copia inválido')
    exigir(re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9._:-]{7,127}', args.encargo_ref),
           'referencia de encargo inválida')
    exigir(1024 <= args.puerto_admin <= 65535 and
           re.fullmatch(r'[a-z0-9_]{3,80}', args.catalogo_entrada_unidad),
           'puerto o catálogo inválidos')
    exigir(re.fullmatch(r'[a-z0-9.-]{3,120}', args.host_admin) and
           '..' not in args.host_admin, 'host ADMIN inválido')
    repo = seguro(args.repo, True)
    go, mod, cache = seguro(args.go), seguro(args.modcache, True), seguro(args.gocache, True)
    exigir(cache == Path('/dev/shm/vec/go-build') and mod == Path.home() / 'go/pkg/mod',
           'cachés Go incompatibles con H10')
    exigir(ejecutar(['git', '-C', str(repo), 'rev-parse', '--show-toplevel']).decode().strip()
           == str(repo), 'repo debe ser raíz Git')
    exigir(estado_git(repo) == args.revision, 'revisión Git distinta de HEAD')
    output = args.salida
    exigir(output.is_absolute() and not output.exists() and not output.is_symlink() and
           not output.is_relative_to(repo) and output.parent.is_dir() and
           not output.parent.is_symlink() and output.parent.resolve() == output.parent and
           output.parent.stat().st_mode & 0o077 == 0,
           'salida debe ser carpeta absoluta, nueva y privada')
    texts = leer_textos_git(repo, args.revision)
    output.mkdir(mode=0o700)
    for name, raw in texts.items():
        guardar(output / name, raw)
    source = output / 'source'
    source.mkdir(mode=0o700)
    hashes = extraer_head(repo, args.revision, source)
    exigir(estado_git(repo) == args.revision, 'HEAD cambió durante la extracción')
    for relative, data in (
        ('cmd/vec-fixture-originales-k/main.go', helper_go(texts['arranque.py']).encode()),
        ('internal/app/bootstrap/derivar_raiz_kit_test.go', texts['derivar_raiz_test.go']),
    ):
        target = source / relative
        target.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
        guardar(target, data)
        hashes[relative] = digest(target)
    guardar(output / 'source-manifest.json', {'version': 1, 'files': hashes})
    guardar(output / 'build-config.json', {'version': 1, 'go': str(go),
        'modcache': str(mod), 'gocache': str(cache), 'repo_revision': args.revision,
        'sid_copia': args.sid_copia, 'host_admin': args.host_admin,
        'host_cabecera': f'{args.host_admin}:{args.puerto_admin}'})
    manifest = {'version': 1, 'revision': args.revision, 'sid_copia': args.sid_copia,
        'puerto_admin': args.puerto_admin, 'conjunto_capacidades': 7,
        'fuente_sha256': digest(output / 'source-manifest.json'),
        'textos_sha256': {name: digest(output / name) for name in texts}}
    if args.sin_compilar:
        manifest['estado'] = 'incompleto_sin_binarios_ni_actos'
    else:
        bin_dir = output / 'bin'
        bin_dir.mkdir(mode=0o700)
        build_tmp = output / 'buildtmp'
        build_tmp.mkdir(mode=0o700)
        env = {'PATH': '/usr/local/go/bin:/usr/bin:/bin', 'HOME': str(Path.home()),
               'TMPDIR': str(build_tmp), 'GOTOOLCHAIN': 'local', 'GOPROXY': 'off',
               'GOSUMDB': 'off', 'GOENV': 'off', 'CGO_ENABLED': '0',
               'GOMAXPROCS': '2', 'GOMEMLIMIT': '512MiB',
               'GOMODCACHE': str(mod), 'GOCACHE': str(cache)}
        try:
            for name in BINARIOS:
                target = bin_dir / name
                command = ([str(go), 'test', '-c', '-p', '2', '-o', str(target), './internal/app/bootstrap']
                    if name == 'h-root-seed.test' else
                    [str(go), 'build', '-p', '2', '-o', str(target),
                     './cmd/' + ('vec-fixture-originales-k' if name == 'helper-hmac' else name)])
                ejecutar(command, cwd=source, env=env)
                target.chmod(0o700)
        finally:
            shutil.rmtree(build_tmp)
        contract = {'version': 2, 'vigencia_dias': 89, 'certificados_dias': 400,
            'conjunto_capacidades': 7, 'sid_copia': args.sid_copia,
            'revision_candidata': args.revision, 'gocache': str(cache),
            'catalogo_entrada_unidad': args.catalogo_entrada_unidad,
            'puerto_admin': args.puerto_admin, 'encargo_ref': args.encargo_ref,
            'scripts_sha256': {name: digest(output / name) for name in
                ('arranque.py', 'guardar_preimagen.py', 'preparar_fixture.py',
                 'preparar_gobierno.py', 'preparar_runtime.py',
                 'preparar_vec_admin.py', 'preparar_overlay_externo.py')},
            'bin_sha256': {'bin/' + name: digest(bin_dir / name) for name in CONTRATO_BIN}}
        guardar(output / 'contrato-copia-local-admin-v1.json', contract)
        manifest['estado'] = 'compilado_sin_actos'
        manifest['binarios_sha256'] = {'bin/' + name: digest(bin_dir / name) for name in BINARIOS}
        manifest['contrato_sha256'] = digest(output / 'contrato-copia-local-admin-v1.json')
    exigir(estado_git(repo) == args.revision, 'HEAD cambió durante la preparación')
    guardar(output / 'manifiesto-kit.json', manifest)
    return manifest


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--repo', type=Path, required=True)
    parser.add_argument('--revision', required=True, help='SHA-1 completo de HEAD')
    parser.add_argument('--sid-copia', required=True, help='system_identifier cotejado por el conductor')
    parser.add_argument('--salida', type=Path, required=True)
    parser.add_argument('--go', type=Path, required=True)
    parser.add_argument('--modcache', type=Path, required=True)
    parser.add_argument('--gocache', type=Path, required=True)
    parser.add_argument('--encargo-ref', required=True)
    parser.add_argument('--puerto-admin', type=int, required=True)
    parser.add_argument('--catalogo-entrada-unidad', required=True)
    parser.add_argument('--host-admin', default='admin.localhost')
    parser.add_argument('--sin-compilar', action='store_true')
    args = parser.parse_args()
    try:
        manifest = crear_kit(args)
    except (ValueError, OSError, subprocess.TimeoutExpired, tarfile.TarError) as exc:
        print('Preparación de kit detenida: ' + str(exc), file=sys.stderr)
        return 1
    print('Kit ADMIN preparado: ' + manifest['estado'] + '; no se ejecutaron SQL ni actos')
    return 0


if __name__ == '__main__':
    sys.exit(main())
