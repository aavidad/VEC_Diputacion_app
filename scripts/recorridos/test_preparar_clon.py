"""Fixtures desechables del orquestador, sin Docker, PostgreSQL ni red."""
from pathlib import Path
import importlib.util
import json
import os
import shutil
import subprocess
import tempfile
import unittest
import uuid


SCRIPT = Path(__file__).with_name('preparar_clon.sh')
REPO = SCRIPT.resolve().parents[2]
spec = importlib.util.spec_from_file_location('sql_orquestador_test', SCRIPT.with_name('clon_sql.py'))
sql = importlib.util.module_from_spec(spec)
spec.loader.exec_module(sql)


class OrquestadorTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.state = self.root / 'state'
        self.state.mkdir(mode=0o700)
        self.calls = self.root / 'docker-called'
        tools = self.root / 'tools'
        tools.mkdir()
        docker = tools / 'docker'
        docker.write_text('#!/bin/sh\nprintf "%s\\n" "$*" >> "$VEC_TEST_CALLS"\nexit 97\n')
        docker.chmod(0o700)
        self.environment = {'PATH': str(tools) + ':/usr/bin:/bin',
            'HOME': str(self.root), 'TMPDIR': str(self.root),
            'PYTHONDONTWRITEBYTECODE': '1', 'GIT_CONFIG_NOSYSTEM': '1',
            'VEC_TEST_CALLS': str(self.calls),
            'VEC_RECORRIDOS_ESTADO': str(self.state),
            'VEC_RECORRIDOS_CONTENEDOR': 'vec-fixture'}

    def run_action(self, action, **environment):
        return subprocess.run(['bash', str(SCRIPT), action],
            env=self.environment | environment, capture_output=True, text=True, timeout=20)

    def write_json(self, name, value):
        p = self.state / name
        p.write_text(json.dumps(value))
        p.chmod(0o600)

    def owner(self, **fields):
        self.write_json('clon.json', {'propietario': 'Codex-M', 'estado': str(self.state),
            'contenedor': 'vec-fixture', 'commit': sql.H6_FIRMA_FINAL_REF,
            'pgdata': '/dev/shm/vec-recorridos-fixture', 'puerto_pg': 55531,
            'puerto_web': 18531} | fields)

    def journal(self, **fields):
        record = {'version': 2, 'run_id': str(uuid.uuid4()), 'pending': None} | fields
        record['journal_sha'] = sql.record_hash(record)
        self.write_json('sql-journal.json', record)

    def assert_no_services(self):
        self.assertFalse(self.calls.exists())
        self.assertFalse((self.state / 'runtime-process.json').exists())
        self.assertFalse((self.state / 'DB_READY.json').exists())

    def test_preparar_y_reiniciar_rechazan_antes_de_servicios_y_estado(self):
        self.state.rmdir()
        for action in ('preparar', 'reiniciar'):
            with self.subTest(action=action):
                p = self.run_action(action)
                self.assertNotEqual(p.returncode, 0)
                self.assertIn('H6-NO-GO composition_incomplete', p.stderr)
                self.assertFalse(self.state.exists())
                self.assert_no_services()

    def test_plan_lee_git_sin_h1_estado_ni_servicios(self):
        self.state.rmdir()
        p = self.run_action('plan', VEC_RECORRIDOS_ARCHIVO='/no-existe')
        self.assertEqual(p.returncode, 0, p.stderr)
        lines = p.stdout.splitlines()
        self.assertEqual(len(lines), 45)
        self.assertIn('000153_perfil_reincorporacion_titular.up.sql', lines[-1])
        self.assertNotIn('000132', p.stdout)
        self.assertFalse(self.state.exists())
        self.assert_no_services()

    def test_bloqueo_composicion_identifica_transporte_material_y_runtime(self):
        p = self.run_action('preparar', VEC_H6_APPROVED='true')
        self.assertNotEqual(p.returncode, 0)
        value = json.loads(p.stdout)
        self.assertEqual(value['sql_count'], 62)
        self.assertTrue(value['ad132_separate'])
        self.assertFalse(value['executable'])
        for dependency in ('fresh_definitive_material_authority_missing',
                           'archive_Docker_controller_and_canonical_pair_not_connected',
                           'PG_namespace_runtime_and_pinned_loopback_relay_not_connected'):
            self.assertIn(dependency, value['blockers'])
        self.assert_no_services()

    def test_estado_sin_journal_es_lectura_y_no_ready(self):
        self.owner()
        before = sorted(p.name for p in self.state.iterdir())
        p = self.run_action('estado')
        self.assertEqual(p.returncode, 0, p.stderr)
        value = json.loads(p.stdout)
        self.assertEqual(value['journal'], 'ausente')
        self.assertFalse(value['ready'])
        self.assertFalse(value['composicion_h6']['executable'])
        self.assertEqual(before, sorted(p.name for p in self.state.iterdir()))
        self.assert_no_services()

    def test_pending_despues_de_crash_se_conserva_y_bloquea_ready(self):
        self.owner()
        self.journal(pending={'position': 1})
        self.write_json('READY.json', {'legado': True})
        before = (self.state / 'sql-journal.json').read_bytes()
        p = self.run_action('estado')
        self.assertEqual(p.returncode, 0, p.stderr)
        value = json.loads(p.stdout)
        self.assertEqual(value['journal'], 'bloqueado')
        self.assertIn('no reaplicar ni publicar READY', value['motivo'])
        self.assertTrue(value['ready_legacy_presente'])
        self.assertFalse(value['ready'])
        p = self.run_action('reiniciar')
        self.assertNotEqual(p.returncode, 0)
        self.assertEqual((self.state / 'sql-journal.json').read_bytes(), before)
        self.assert_no_services()

    def test_diario_v1_y_corrupto_no_se_convierten(self):
        self.owner()
        for record in ({'version': 1, 'installed': []}, {'version': 2, 'journal_sha': 'incorrecto'}):
            with self.subTest(record=record):
                self.write_json('sql-journal.json', record)
                before = (self.state / 'sql-journal.json').read_bytes()
                p = self.run_action('estado')
                self.assertEqual(p.returncode, 0, p.stderr)
                self.assertEqual(json.loads(p.stdout)['journal'], 'bloqueado')
                self.assertEqual((self.state / 'sql-journal.json').read_bytes(), before)
                self.assert_no_services()

    def test_journal_completo_local_no_acredita_ready_sin_kit(self):
        self.owner()
        plan = sql.validate_git_source(sql.H6_FIRMA_FINAL_REF, REPO)
        # Confirmaciones ficticias para probar exclusivamente la lectura local.
        receipts = [dict(position=i, **row, confirmed_at='2026-09-30T00:00:00+00:00',
                    confirmation='commit_returned_and_postcheck')
                    for i, row in enumerate(plan['entries'], 1)]
        self.journal(**{k: 'a'*64 for k in sql.CONTEXT_KEYS}, source_commit=plan['source_ref'],
            approved_sql_ref=plan['approved_sql_ref'], plan_sha=plan['plan_sha'],
            inventory_sha=plan['inventory_sha'], entries=plan['entries'], file_count=45,
            installed=receipts, phase='ad132_confirmed')
        p = self.run_action('estado')
        self.assertEqual(p.returncode, 0, p.stderr)
        value = json.loads(p.stdout)
        self.assertEqual(value['journal'], 'v2')
        self.assertEqual(value['sql_confirmadas'], 45)
        self.assertFalse(value['ready'])
        self.assert_no_services()

    def test_estado_rechaza_permisos_abiertos_enlaces_y_git(self):
        self.owner()
        self.state.chmod(0o755)
        p = self.run_action('estado')
        self.assertNotEqual(p.returncode, 0)
        self.state.chmod(0o700)
        alias = self.root / 'alias'
        alias.symlink_to(self.state, target_is_directory=True)
        p = self.run_action('estado', VEC_RECORRIDOS_ESTADO=str(alias))
        self.assertNotEqual(p.returncode, 0)
        (self.state / '.git').write_text('fixture')
        p = self.run_action('estado')
        self.assertNotEqual(p.returncode, 0)
        self.assert_no_services()

    def test_estado_rechaza_marker_enlazado_y_journal_no_lo_sigue(self):
        self.owner()
        external = self.root / 'foreign'
        external.write_text('privado-sintetico')
        (self.state / 'sql-journal.json').symlink_to(external)
        p = self.run_action('estado')
        self.assertEqual(p.returncode, 0, p.stderr)
        self.assertEqual(json.loads(p.stdout)['journal'], 'bloqueado')
        self.assertNotIn('privado-sintetico', p.stdout + p.stderr)
        (self.state / 'clon.json').unlink()
        (self.state / 'clon.json').symlink_to(external)
        p = self.run_action('estado')
        self.assertNotEqual(p.returncode, 0)
        self.assert_no_services()

    def test_parar_y_retirar_rechazan_propietario_y_volumen_ajenos(self):
        for fields in ({'propietario': 'Otro'}, {'pgdata': '/otra/ruta'},
                       {'estado': str(self.root)}, {'contenedor': 'vec-ajeno'}):
            for action in ('parar', 'retirar'):
                with self.subTest(fields=fields, action=action):
                    self.owner(**fields)
                    p = self.run_action(action)
                    self.assertNotEqual(p.returncode, 0)
                    self.assert_no_services()

    def test_retirar_sin_contenedor_conserva_volumen_prefijado_y_evidencia(self):
        with tempfile.TemporaryDirectory(prefix='vec-recorridos-ajeno-', dir='/dev/shm') as volume:
            self.owner(pgdata=volume)
            payload=Path(volume)/'ajeno'
            payload.write_bytes(b'contenido-ajeno')
            self.write_json('READY.json',{'legado':True})
            self.journal(pending={'position':1})
            before={p.name:p.read_bytes() for p in self.state.iterdir()}
            p=self.run_action('retirar')
            self.assertNotEqual(p.returncode,0)
            self.assertIn('Retirada bloqueada',p.stderr)
            self.assertEqual(self.calls.read_text().splitlines(),['inspect vec-fixture'])
            self.assertEqual(payload.read_bytes(),b'contenido-ajeno')
            for name,data in before.items():
                self.assertEqual((self.state/name).read_bytes(),data)
            self.assertFalse((self.state/'RETIRADO.json').exists())
            self.assertFalse((self.state/'runtime-process.json').exists())

    def test_plan_y_estado_no_crean_bytecode_con_fuente_escribible(self):
        self.owner()
        control=self.root/'control'
        scripts=control/'scripts/recorridos'
        scripts.mkdir(parents=True)
        for name in ('preparar_clon.sh','clon_sql.py','sql_main.txt',
                     'sql_main_h6.txt','sql_main_h6_firma.txt','clon_h6_orquestador.py'):
            shutil.copyfile(SCRIPT.with_name(name),scripts/name)
        git=self.root/'tools/git'
        git.write_text('#!/bin/sh\ncase "$*" in *"rev-parse --show-toplevel") printf "%s\\n" "$VEC_TEST_REPO" ;; *) exec /usr/bin/git "$@" ;; esac\n')
        git.chmod(0o700)
        environment=self.environment | {'VEC_TEST_REPO':str(REPO),
            'PYTHONPYCACHEPREFIX':str(self.root/'bytecode')}
        environment.pop('PYTHONDONTWRITEBYTECODE',None)
        for action in ('plan','estado'):
            with self.subTest(action=action):
                p=subprocess.run(['bash',str(scripts/'preparar_clon.sh'),action],
                    capture_output=True,text=True,env=environment,timeout=20)
                self.assertEqual(p.returncode,0,p.stderr)
                self.assertFalse((self.root/'bytecode').exists())
                self.assertEqual(list(control.rglob('*.pyc')),[])
                self.assertEqual(list(control.rglob('__pycache__')),[])
        self.assert_no_services()

    def test_optimizacion_python_no_desactiva_propiedad(self):
        self.owner(propietario='Otro')
        p=self.run_action('retirar',PYTHONOPTIMIZE='1')
        self.assertNotEqual(p.returncode,0)
        self.assert_no_services()

    def test_limpieza_conserva_diario_material_y_archivos_no_reconocidos(self):
        source = SCRIPT.read_text()
        start = source.index('    python3 -B - "$marcador" <<\'PY\'')
        code = source[start:].split("\n", 1)[1].split('\nPY\n', 1)[0]
        with tempfile.TemporaryDirectory(prefix='vec-recorridos-', dir='/dev/shm') as volume:
            self.owner(pgdata=volume)
            for name in ('sql-journal.json', 'material', 'capturas', 'ajeno', 'READY-historico.json'):
                self.write_json(name, {'conservar': True})
            (self.state / 'fuente').mkdir()
            (self.state / 'fuente' / 'dummy').write_text('fixture')
            (self.state / 'build.log').write_text('fixture')
            # Sustituir sólo el lanzamiento Docker; el volumen de fixture está vacío.
            harness = "import subprocess,sys\nsubprocess.run=lambda *a,**kw: None\nsys.argv=['fixture',sys.argv[1]]\n" + code
            p = subprocess.run(['/usr/bin/python3', '-c', harness, str(self.state / 'clon.json')],
                capture_output=True, text=True, env=self.environment, timeout=5)
            self.assertEqual(p.returncode, 0, p.stderr)
            self.assertFalse(Path(volume).exists())
            self.assertFalse((self.state / 'fuente').exists())
            self.assertFalse((self.state / 'build.log').exists())
            for name in ('sql-journal.json', 'material', 'capturas', 'ajeno', 'READY-historico.json'):
                self.assertTrue((self.state / name).exists())
            self.assert_no_services()

    def test_preparador_no_tiene_ledger_sql_ni_camino_de_instalacion_heredado(self):
        source = SCRIPT.read_text()
        self.assertNotIn('vec_recorridos_clon', source)
        self.assertNotIn('docker exec', source)
        self.assertNotIn('clon_material.py', source)
        self.assertNotIn('--steps', source)
        self.assertNotIn('db.query', source)


if __name__ == '__main__':
    unittest.main()
