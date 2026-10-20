"""Pruebas focales del empaquetado ADMIN sin Go, SQL ni red."""
import argparse
import hashlib
import io
import json
import os
from pathlib import Path
import stat
import subprocess
import tarfile
import tempfile
import unittest
from unittest import mock
from types import SimpleNamespace

from admin_copia import arranque, guardar_preimagen, kit, configuracion


class FakeProcess:
    def __init__(self, data):
        self.stdout = io.BytesIO(data)
        self.stderr = io.BytesIO()

    def wait(self, timeout=None):
        return 0

    def poll(self):
        return 0


class KitTest(unittest.TestCase):
    def setUp(self):
        self.scratch = tempfile.TemporaryDirectory(dir=os.environ['VEC_TEST_SCRATCH'])
        self.addCleanup(self.scratch.cleanup)
        self.base = Path(self.scratch.name)
        os.chmod(self.base, 0o700)

    def test_snapshot_git_regulares_y_enlace_rechazado(self):
        def archive(symlink=False):
            out = io.BytesIO()
            with tarfile.open(fileobj=out, mode='w') as tar:
                for name, data in [('go.mod', b'module example\n'), ('go.sum', b'')] + [
                    (f'config/f{i:04}.json', b'{}') for i in range(1001)]:
                    info = tarfile.TarInfo(name)
                    info.size = len(data)
                    tar.addfile(info, io.BytesIO(data))
                if symlink:
                    info = tarfile.TarInfo('config/fuera')
                    info.type = tarfile.SYMTYPE
                    info.linkname = '/etc/passwd'
                    tar.addfile(info)
            return out.getvalue()

        source = self.base / 'source'
        source.mkdir(mode=0o700)
        with mock.patch.object(kit.subprocess, 'Popen', return_value=FakeProcess(archive())):
            hashes = kit.extraer_head(self.base, 'a' * 40, source)
        self.assertEqual(len(hashes), 1003)
        self.assertEqual(hashes['go.mod'], hashlib.sha256(b'module example\n').hexdigest())
        self.assertFalse((source / 'config/fuera').exists())
        other = self.base / 'other'
        other.mkdir(mode=0o700)
        with mock.patch.object(kit.subprocess, 'Popen', return_value=FakeProcess(archive(True))):
            with self.assertRaisesRegex(ValueError, 'enlazada'):
                kit.extraer_head(self.base, 'a' * 40, other)

    def test_revision_inestable_y_empaquetado_sin_actos(self):
        repo = self.base / 'repo'
        repo.mkdir(mode=0o700)
        output = self.base / 'kit'
        revision = 'a' * 40
        texts = {name: b'x' for name in kit.TEXTOS.values()}
        texts['arranque.py'] = (Path(__file__).parent / 'admin_copia/arranque.py').read_bytes()
        args = argparse.Namespace(repo=repo, revision=revision, sid_copia='1234567890123456789',
            salida=output, go=Path('/usr/local/go/bin/go'),
            modcache=Path.home() / 'go/pkg/mod', gocache=Path('/dev/shm/vec/go-build'),
            encargo_ref='encargo:test', puerto_admin=18443,
            catalogo_entrada_unidad='rrhh_centro', host_admin='admin.localhost',
            sin_compilar=True)

        def extract(_repo, _revision, source):
            path = source / 'go.mod'
            kit.guardar(path, b'module example\n')
            return {'go.mod': kit.digest(path)}

        patches = (mock.patch.object(kit, 'seguro', side_effect=lambda path, directorio=False: path),
                   mock.patch.object(kit, 'ejecutar', return_value=str(repo).encode()),
                   mock.patch.object(kit, 'leer_textos_git', return_value=texts),
                   mock.patch.object(kit, 'extraer_head', side_effect=extract))
        with patches[0], patches[1], patches[2], patches[3], mock.patch.object(
                kit, 'estado_git', side_effect=[revision, revision, 'b' * 40]):
            with self.assertRaisesRegex(ValueError, 'HEAD cambió'):
                kit.crear_kit(args)
        self.assertFalse((output / 'manifiesto-kit.json').exists())
        output2 = self.base / 'kit2'
        args.salida = output2
        with patches[0], patches[1], patches[2], patches[3], mock.patch.object(
                kit, 'estado_git', return_value=revision):
            manifest = kit.crear_kit(args)
        self.assertEqual(manifest['estado'], 'incompleto_sin_binarios_ni_actos')
        self.assertFalse((output2 / 'bin').exists())
        self.assertFalse((output2 / 'contrato-copia-local-admin-v1.json').exists())
        self.assertFalse((output2 / 'aprobaciones').exists())
        self.assertEqual(json.loads((output2 / 'build-config.json').read_text())['sid_copia'], args.sid_copia)

    def test_manifiesto_alterado_impide_configuracion(self):
        root = self.base / 'kit'
        root.mkdir(mode=0o700)
        kit.guardar(root / 'source-manifest.json', b'{}')
        kit.guardar(root / 'build-config.json', {'repo_revision': 'a' * 40,
                                                'sid_copia': '1234567890123456789'})
        kit.guardar(root / 'contrato-copia-local-admin-v1.json',
                   {'revision_candidata': 'a' * 40, 'sid_copia': '1234567890123456789',
                    'puerto_admin': 18443, 'catalogo_entrada_unidad': 'rrhh_centro',
                    'conjunto_capacidades': 7})
        contract_hash = kit.digest(root / 'contrato-copia-local-admin-v1.json')
        manifest = {'estado': 'compilado_sin_actos', 'revision': 'a' * 40,
                    'sid_copia': '1234567890123456789', 'puerto_admin': 18443,
                    'conjunto_capacidades': 7, 'contrato_sha256': contract_hash,
                    'fuente_sha256': hashlib.sha256(b'original').hexdigest()}
        kit.guardar(root / 'manifiesto-kit.json', manifest)
        with self.assertRaisesRegex(ValueError, 'manifiesto de fuentes alterado'):
            configuracion.validar_kit(root, 'a' * 40, '1234567890123456789',
                                       18443, 'rrhh_centro')

    def test_snapshot_incluye_fuentes_runtime_cotejadas(self):
        root = Path(__file__).parent / 'admin_copia'
        revision = 'a' * 40
        def fake_git(command, **_kwargs):
            operation = command[3]
            path = command[-1].split(':', 1)[-1]
            if operation == 'ls-tree':
                path = command[-1]
            source = Path(__file__).parents[2] / path
            self.assertTrue(source.is_file(), path)
            raw = source.read_bytes()
            if operation == 'ls-tree':
                return f'100644 blob {"0" * 40}\t{path}\n'.encode()
            if operation == 'cat-file':
                return str(len(raw)).encode()
            self.assertEqual(operation, 'show')
            return raw

        with mock.patch.object(kit, 'ejecutar', side_effect=fake_git):
            texts = kit.leer_textos_git(self.base, revision)
        self.assertEqual(len(texts), 14)
        self.assertEqual(len(set(texts)), 14)
        runtime = ('preparar_gobierno.py', 'preparar_lote_plan.py',
                   'preparar_relevo_logins.py', 'preparar_runtime.py',
                   'preparar_vec_admin.py', 'preparar_overlay_externo.py')
        for name in runtime:
            self.assertEqual(texts[name], (root / name).read_bytes())
        self.assertEqual(texts['plantilla-runtime.json'],
                         (root / 'datos/plantilla_runtime.json').read_bytes())

    def test_configuracion_h10_con_kit_cotejado_sin_aprobaciones(self):
        root = self.base / 'kit'
        root.mkdir(mode=0o700)
        source = root / 'source'
        source.mkdir(mode=0o700)
        (source / 'config').mkdir(mode=0o700)
        hashes = {}
        for i in range(1001):
            rel = f'config/f{i:04}.json'
            kit.guardar(source / rel, b'{}')
            hashes[rel] = kit.digest(source / rel)
        catalog_dir = source / 'web/static/textos/es'
        catalog_dir.mkdir(parents=True, mode=0o700)
        for _name, catalog in configuracion.CATALOGOS.values():
            rel = 'web/static/textos/es/' + catalog
            kit.guardar(source / rel, b'{}')
            hashes[rel] = kit.digest(source / rel)
        kit.guardar(root / 'source-manifest.json', {'version': 1, 'files': hashes})
        kit.guardar(root / 'build-config.json', {'repo_revision': 'a' * 40,
            'sid_copia': '1234567890123456789', 'host_admin': 'admin.localhost',
            'go': '/usr/local/go/bin/go', 'modcache': '/scratch/go/pkg/mod',
            'gocache': '/dev/shm/vec/go-build'})
        originals = Path(__file__).parent / 'admin_copia'
        for local, packed in kit.TEXTOS.items():
            kit.guardar(root / packed, (originals / local).read_bytes())
        (root / 'bin').mkdir(mode=0o700)
        bin_hashes = {}
        for name in kit.BINARIOS:
            rel = 'bin/' + name
            kit.guardar(root / rel, b'binario-ficticio')
            bin_hashes[rel] = kit.digest(root / rel)
        contract = {'revision_candidata': 'a' * 40, 'sid_copia': '1234567890123456789',
            'puerto_admin': 18443, 'catalogo_entrada_unidad': 'rrhh_centro',
            'conjunto_capacidades': 7,
            'scripts_sha256': {name: kit.digest(root / name) for name in
                ('arranque.py', 'guardar_preimagen.py', 'preparar_fixture.py',
                 'preparar_gobierno.py', 'preparar_lote_plan.py',
                 'preparar_relevo_logins.py', 'preparar_runtime.py',
                 'preparar_vec_admin.py', 'preparar_overlay_externo.py')},
            'bin_sha256': {name: bin_hashes[name] for name in
                ('bin/h-root-seed.test', 'bin/vec-gobierno-usuarios-admin', 'bin/vec-admin')}}
        kit.guardar(root / 'contrato-copia-local-admin-v1.json', contract)
        kit.guardar(root / 'manifiesto-kit.json', {'estado': 'compilado_sin_actos',
            'revision': 'a' * 40, 'sid_copia': '1234567890123456789',
            'puerto_admin': 18443, 'conjunto_capacidades': 7,
            'contrato_sha256': kit.digest(root / 'contrato-copia-local-admin-v1.json'),
            'fuente_sha256': kit.digest(root / 'source-manifest.json'),
            'textos_sha256': {name: kit.digest(root / name) for name in kit.TEXTOS.values()},
            'binarios_sha256': bin_hashes})
        backup, ca = self.base / 'backup.dump', self.base / 'ca.pem'
        kit.guardar(backup, b'preimagen-ficticia')
        kit.guardar(ca, b'ca-ficticia')
        material = self.base / 'material'
        material.mkdir(mode=0o700)
        (material / 'idempotencia').mkdir(mode=0o700)
        kit.guardar(material / 'idempotencia/configuracion.json', b'{}')
        output = self.base / 'arranque'
        args = argparse.Namespace(kit=root, revision='a' * 40,
            sid_copia='1234567890123456789', puerto_admin=18443, instancia='ensayo1',
            contenedor='contenedor-copia', puerto_pg=55441, backup_admin=backup,
            material=material, ca_pg=ca, catalogo_entrada_unidad='rrhh_centro',
            salida=output, dias_organizacion=90, dias_personas=89,
            dias_unidad=89, dias_perfiles=88, dias_ca=400)
        configuracion.preparar(args)
        self.assertEqual(len(contract['scripts_sha256']), 9)
        loaded = arranque.cargar_config(output / 'config.json')
        self.assertEqual(loaded['fuentes']['ttl_organizacion_segundos'], 90 * 86400)
        self.assertEqual(loaded['bootstrap']['ttl_perfiles_segundos'], 88 * 86400)
        self.assertEqual(loaded['fuentes']['ca_dias'], 400)
        self.assertFalse((output / 'aprobaciones/fuentes.json').exists())
        self.assertEqual(json.loads((output / 'contrato-arranque.json').read_text())['sid_copia'],
                         args.sid_copia)


class PreimagenTest(unittest.TestCase):
    SID = '1234567890123456789'

    def setUp(self):
        self.scratch = tempfile.TemporaryDirectory(dir=os.environ['VEC_TEST_SCRATCH'])
        self.addCleanup(self.scratch.cleanup)
        self.output = Path(self.scratch.name) / 'preimagen.dump'
        os.chmod(self.output.parent, 0o700)

    def invoke(self, fake_run):
        argv = ['guardar_preimagen.py', '--contenedor', 'contenedor-copia',
                '--sid-copia', self.SID, '--salida', str(self.output)]
        with mock.patch('sys.argv', argv), mock.patch.object(
                guardar_preimagen.subprocess, 'run', side_effect=fake_run), mock.patch.dict(
                os.environ, {'DOCKER_HOST': 'tcp://remote.invalid:2375',
                             'DOCKER_CONTEXT': 'remoto'}):
            guardar_preimagen.main()

    def check_environment(self, kwargs):
        self.assertEqual(kwargs['env']['DOCKER_HOST'], 'unix:///var/run/docker.sock')
        self.assertNotIn('DOCKER_CONTEXT', kwargs['env'])
        self.assertEqual(kwargs['env']['PATH'], '/usr/bin:/bin')
        self.assertEqual(kwargs['env']['LANG'], 'C')

    def test_volcado_nace_privado_con_socket_local(self):
        calls = []

        def fake_run(command, **kwargs):
            self.check_environment(kwargs)
            calls.append((command, kwargs))
            if 'psql' in command:
                return SimpleNamespace(stdout=self.SID + '\n')
            self.assertEqual(stat.S_IMODE(os.fstat(kwargs['stdout'].fileno()).st_mode), 0o600)
            kwargs['stdout'].write(b'x' * 120)
            return SimpleNamespace(returncode=0)

        previous = os.umask(0o022)
        try:
            self.invoke(fake_run)
        finally:
            os.umask(previous)
        self.assertEqual([call[1]['timeout'] for call in calls], [30, 120])
        self.assertEqual(stat.S_IMODE(self.output.stat().st_mode), 0o600)

    def test_timeout_borra_volcado_parcial(self):
        def fake_run(command, **kwargs):
            self.check_environment(kwargs)
            if 'psql' in command:
                return SimpleNamespace(stdout=self.SID + '\n')
            self.assertEqual(kwargs['timeout'], 120)
            self.assertEqual(stat.S_IMODE(os.fstat(kwargs['stdout'].fileno()).st_mode), 0o600)
            kwargs['stdout'].write(b'volcado incompleto')
            raise subprocess.TimeoutExpired(command, 120)

        with self.assertRaises(subprocess.TimeoutExpired):
            self.invoke(fake_run)
        self.assertFalse(self.output.exists())


if __name__ == '__main__':
    unittest.main()
