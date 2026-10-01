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
            'VEC_RECORRIDOS_CONTENEDOR': 'vec-fixture',
            'VEC_RECORRIDOS_PUERTO_PG': '55531'}

    def run_action(self, action, *arguments, **environment):
        return subprocess.run(['bash', str(SCRIPT), action, *arguments],
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

    def offline_fixture(self):
        """Sustituir sólo el ejecutable Python de las dos autoridades offline."""
        python = self.root / 'tools/python3'
        python.write_text('''#!/usr/bin/python3
import json, os, sys
from pathlib import Path
args = sys.argv[1:]
if len(args) > 1 and Path(args[1]).name in {
        'clon_material_externo_offline.py', 'clon_alias_export.py'}:
    with open(os.environ['VEC_TEST_OFFLINE_CALLS'], 'a') as log:
        log.write(json.dumps(args) + '\\n')
    if Path(args[1]).name == 'clon_alias_export.py' and '--replay-receipt-sha256' not in args:
        with open(os.environ['VEC_TEST_BINARY_CALLS'], 'a') as log:
            log.write('exportador-simulado\\n')
    # Una verificación lleva siempre el pin externo: simular rechazo si no hay
    # intento previo, sin crear salida ni ejecutar el exportador.
    if '--replay-receipt-sha256' in args and os.environ.get('VEC_TEST_ATTEMPT_ABSENT'):
        sys.exit(2)
    sys.exit(int(os.environ.get('VEC_TEST_OFFLINE_EXIT', '0')))
os.execv('/usr/bin/python3', ['/usr/bin/python3', *args])
''')
        python.chmod(0o700)
        self.offline_calls = self.root / 'offline-called'
        self.binary_calls = self.root / 'binary-called'
        self.environment['VEC_TEST_OFFLINE_CALLS'] = str(self.offline_calls)
        self.environment['VEC_TEST_BINARY_CALLS'] = str(self.binary_calls)
        # Espacios y metacaracteres deben viajar en un argumento literal.
        self.fixture_paths = {name: str(self.root / ('privado ' + name + ' $(touch NO)'))
                              for name in ('fuente', 'acuse', 'directorio', 'binario', 'material', 'salida')}
        self.write_json('READY.json', {'legado': True})
        self.write_json('sql-journal.json', {'conservar': True})
        self.offline_preimage = {p.name: p.read_bytes() for p in self.state.iterdir()}

    def offline_arguments(self, action):
        names = ('fuente', 'acuse', 'directorio') if action == 'preparar-material-externo' else (
            'binario', 'fuente', 'acuse', 'material', 'salida')
        return [part for name in names for part in ('--' + name, self.fixture_paths[name])]

    def assert_offline_preserves_state(self):
        self.assertEqual({p.name: p.read_bytes() for p in self.state.iterdir()}, self.offline_preimage)
        self.assertFalse((self.root / 'NO').exists())
        self.assert_no_services()

    def test_offline_despacha_cli_exacta_y_rutas_literales_sin_servicios(self):
        self.offline_fixture()
        for action, authority in (
                ('preparar-material-externo', 'clon_material_externo_offline.py'),
                ('exportar-alias', 'clon_alias_export.py'),
                ('verificar-alias', 'clon_alias_export.py')):
            with self.subTest(action=action):
                arguments = self.offline_arguments(action)
                if action == 'verificar-alias':
                    arguments += ['--replay-receipt-sha256', 'a' * 64]
                p = self.run_action(action, *arguments)
                self.assertEqual(p.returncode, 0, p.stderr)
                call = json.loads(self.offline_calls.read_text().splitlines()[-1])
                self.assertEqual(call, ['-B', str(SCRIPT.with_name(authority)), *arguments])
                self.assertEqual(p.stdout + p.stderr, '')
                self.assert_offline_preserves_state()

    def test_offline_faltan_rutas_rechaza_antes_del_ejecutable(self):
        self.offline_fixture()
        for action in ('preparar-material-externo', 'exportar-alias', 'verificar-alias'):
            with self.subTest(action=action):
                arguments = self.offline_arguments(action)
                if action == 'verificar-alias':
                    arguments += ['--replay-receipt-sha256', 'a' * 64]
                for index in range(0, len(arguments), 2):
                    p = self.run_action(action, *(arguments[:index] + arguments[index + 2:]))
                    self.assertEqual(p.returncode, 2)
                    self.assertEqual(p.stdout, '')
                    self.assertEqual(p.stderr, 'H6-OFFLINE arguments_invalid\n')
                    self.assertFalse(self.offline_calls.exists())
        self.assert_offline_preserves_state()

    def test_verificar_alias_exige_pin_externo_sin_inferir_del_paquete(self):
        self.offline_fixture()
        output = Path(self.fixture_paths['salida'])
        output.mkdir(mode=0o700)
        receipt = output / 'alias-export.receipt.json'
        receipt.write_text('recibo-fixture-no-observado')
        arguments = self.offline_arguments('verificar-alias')
        for pin in (None, '', 'A' * 64, 'a' * 63, 'a' * 65, 'no-es-SHA'):
            with self.subTest(pin=pin):
                supplied = [] if pin is None else ['--replay-receipt-sha256', pin]
                p = self.run_action('verificar-alias', *arguments, *supplied)
                self.assertEqual(p.returncode, 2)
                self.assertFalse(self.offline_calls.exists())
                self.assertEqual(receipt.read_text(), 'recibo-fixture-no-observado')
        self.assert_offline_preserves_state()

    def test_offline_rechaza_opciones_duplicadas_abreviadas_ajenas_y_rutas_relativas(self):
        self.offline_fixture()
        for action in ('preparar-material-externo', 'exportar-alias', 'verificar-alias'):
            base = self.offline_arguments(action)
            if action == 'verificar-alias':
                base += ['--replay-receipt-sha256', 'a' * 64]
            invalid = [base + ['--fuente', '/otra'], base + ['--fu', '/otra'],
                       base + ['--approved', 'true'], base + ['--steps', '62'],
                       base + ['--fuente'], ['--fuente', 'relativa', *base[2:]]]
            wrong = '--binario' if action == 'preparar-material-externo' else '--directorio'
            invalid += [base + [wrong, '/otra']]
            if action != 'verificar-alias':
                invalid += [base + ['--replay-receipt-sha256', 'a' * 64]]
            for arguments in invalid:
                with self.subTest(action=action, arguments=arguments):
                    p = self.run_action(action, *arguments)
                    self.assertEqual(p.returncode, 2)
                    self.assertEqual(p.stderr, 'H6-OFFLINE arguments_invalid\n')
                    self.assertFalse(self.offline_calls.exists())
        self.assert_offline_preserves_state()

    def test_offline_propaga_rechazo_y_verificacion_no_se_convierte_en_exportacion(self):
        self.offline_fixture()
        for action in ('preparar-material-externo', 'exportar-alias'):
            p = self.run_action(action, *self.offline_arguments(action), VEC_TEST_OFFLINE_EXIT='17')
            self.assertEqual(p.returncode, 17)
        for _ in range(2):
            p = self.run_action('verificar-alias', *self.offline_arguments('verificar-alias'),
                                '--replay-receipt-sha256', 'b' * 64, VEC_TEST_ATTEMPT_ABSENT='1')
            self.assertEqual(p.returncode, 2)
            call = json.loads(self.offline_calls.read_text().splitlines()[-1])
            self.assertEqual(call[-2:], ['--replay-receipt-sha256', 'b' * 64])
        self.assertFalse(Path(self.fixture_paths['salida']).exists())
        self.assert_offline_preserves_state()

    def test_verificar_alias_con_recibo_conservado_no_repite_el_exportador(self):
        self.offline_fixture()
        p = self.run_action('exportar-alias', *self.offline_arguments('exportar-alias'))
        self.assertEqual(p.returncode, 0, p.stderr)
        output = Path(self.fixture_paths['salida'])
        output.mkdir(mode=0o700)
        receipt = output / 'alias-export.receipt.json'
        receipt.write_bytes(b'recibo-primera-ejecucion-simulada')
        before = (receipt.stat().st_ino, receipt.stat().st_mtime_ns, receipt.read_bytes())
        for _ in range(2):
            p = self.run_action('verificar-alias', *self.offline_arguments('verificar-alias'),
                                '--replay-receipt-sha256', 'c' * 64)
            self.assertEqual(p.returncode, 0, p.stderr)
            self.assertEqual((receipt.stat().st_ino, receipt.stat().st_mtime_ns, receipt.read_bytes()), before)
        self.assertEqual(self.binary_calls.read_text().splitlines(), ['exportador-simulado'])
        self.assert_offline_preserves_state()

    def test_preparar_y_reiniciar_rechazan_antes_de_servicios_y_estado(self):
        self.state.rmdir()
        for action in ('preparar', 'reiniciar'):
            with self.subTest(action=action):
                p = self.run_action(action)
                self.assertNotEqual(p.returncode, 0)
                self.assertIn('H6-NO-GO composition_incomplete', p.stderr)
                self.assertFalse(self.state.exists())
                self.assert_no_services()

    def test_plan_sin_entradas_externas_deniega_sin_historia45_estado_o_servicios(self):
        self.state.rmdir()
        p = self.run_action('plan', VEC_RECORRIDOS_ARCHIVO='/no-existe')
        self.assertNotEqual(p.returncode, 0)
        self.assertIn('arguments_invalid_external_inputs_required', p.stderr)
        self.assertIn('approved_package_sha256', p.stderr)
        self.assertIn('approved_lock_sha256', p.stderr)
        self.assertIn('approved_h1_sha256', p.stderr)
        self.assertEqual(p.stdout, '')
        self.assertFalse(self.state.exists())
        self.assert_no_services()

    def test_plan_postmain_es_solo_lectura_y_exige_pines(self):
        self.state.rmdir()
        ayuda = self.run_action('plan-postmain', '--help')
        self.assertEqual(ayuda.returncode, 0, ayuda.stderr)
        self.assertIn('--expected-e3-fixture-sha256', ayuda.stdout)
        sin_pines = self.run_action('plan-postmain')
        self.assertEqual(sin_pines.returncode, 2)
        self.assertIn('--target-commit', sin_pines.stderr)
        self.assertEqual(sin_pines.stdout, '')
        self.assertFalse(self.state.exists())
        self.assert_no_services()

    def test_sql_phase_without_nominal_external_pins_rejects_before_Docker(self):
        before = sorted(self.state.iterdir())
        for action in ('preparar-sql', 'verificar-sql'):
            p = self.run_action(action, VEC_H6_APPROVED='true')
            self.assertNotEqual(p.returncode, 0)
            self.assertIn('external_inputs_required', p.stderr)
            self.assertEqual(p.stdout, '')
        self.assertEqual(before, sorted(self.state.iterdir()))
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

    def test_estado_h6_con_journal_sin_marcador_legacy(self):
        self.journal(phase='awaiting_ad132', file_count=62,
            source_commit='73e56c106d12fdda0bd16d6fe573503c42c5495f',
            approved_sql_ref='73e56c106d12fdda0bd16d6fe573503c42c5495f',
            entries=[{'position': i} for i in range(1, 63)],
            installed=[{'position': i} for i in range(1, 63)])
        before = (self.state / 'sql-journal.json').read_bytes()
        p = self.run_action('estado')
        self.assertEqual(p.returncode, 0, p.stderr)
        value = json.loads(p.stdout)
        self.assertEqual(value['journal'], 'v2_sin_revalidacion_viva')
        self.assertEqual(value['fase'], 'awaiting_ad132')
        self.assertEqual(value['sql_declaradas'], 62)
        self.assertFalse(value['ready'])
        self.assertEqual((self.state / 'sql-journal.json').read_bytes(), before)
        self.assert_no_services()

    def test_estado_sin_marcador_ni_journal_no_infiere_clon(self):
        p = self.run_action('estado')
        self.assertEqual(p.returncode, 2)
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
                self.assertEqual(p.returncode, 1 if action == 'plan' else 0, p.stderr)
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
