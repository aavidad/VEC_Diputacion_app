"""Helper simulado en memoria; cero SQL, Docker, red o servicios reales."""
import copy
import io
import json
import sys
import os
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch
import clon_cas_helper as h
import clon_preimagenes_nominales as p
from test_clon_preimagenes_nominales import TARGET, ALIAS, SHA, TIME1, TIME2, auth, motives


def request():
    users = {'identidad': 'vec_fixture_identidad', 'contexto': 'vec_fixture_contexto', 'autorizacion': h.AUTH_USER}
    target_u = {**TARGET, 'provision_ref': 'pue_' + 'a' * 32}
    return {'version': 1, 'kind': 'cas_helper_request_v1', 'nonce': SHA,
      'server_name': 'pg.fixture', 'pgid': {'pg_container_id': SHA, 'pg_image': 'postgres:18.4',
        'pg_image_id': 'sha256:' + SHA, 'pg_volume': '/dev/shm/vec-recorridos-fixture',
        'system_identifier': '1234', 'database_name': 'postgres', 'database_oid': 1},
      'pins': {k: SHA for k in h.INPUT_FILES}, 'users': users,
      'tls_hashes': {k: SHA for k in {'ca.pem'} | {c + ext for c in h.CHANNELS for ext in ('.key', '.crt')}},
      'targets': {'candidato': TARGET, 'usuarios': target_u}}


class FakeCursor:
    def __init__(self, connection):
        self.c = connection
        self.rows = []
    def __enter__(self): return self
    def __exit__(self, *args): return False
    def execute(self, sql, parameters=()):
        self.c.runtime.calls.append((self.c.channel, sql, parameters))
        runtime = self.c.runtime
        if sql == h.SESSION_SQL:
            role = h.ROLES[self.c.channel]
            self.rows = [(self.c.user, self.c.user if role == 'none' else role, role, 'on', 'read committed',
                          self.c.pid, 'postgres', 1, '1234', '127.0.0.1', '127.0.0.1', 5432, True,
                          runtime.tls, 'TLSv1.3', True)]
            self.c.session_reads += 1
            if runtime.pid_drift and self.c.session_reads > 1:
                row = list(self.rows[0]); row[5] += 10; self.rows = [tuple(row)]
        elif sql == h.TABLE_SQL:
            self.rows = [(True, True, runtime.rls, runtime.permission, True, runtime.policy)]
        elif sql == h.FUNCTION_SQL:
            self.rows = [(True, True, runtime.permission, True)]
        elif sql == h.AUTH_TABLE_SQL:
            self.rows = [(False, False)]
        elif sql == p.ACCOUNT_SQL or sql == p.ALIAS_SQL or sql == p.CONTEXT_SQL:
            self.rows = []
        elif sql == p.AUTH_SQL:
            runtime.auth_reads += 1
            self.rows = [(auth(TIME1 if runtime.auth_reads == 1 else TIME2, runtime.present),)]
            if runtime.drift and runtime.auth_reads == 2:
                self.rows[0][0]['control_rol']['huella_sha256'] = 'b' * 64
        elif sql == p.MOTIVES_SQL:
            self.rows = [(motives(TIME1 if runtime.auth_reads == 1 else TIME2),)]
        elif sql == h.ROLLBACK_SQL:
            self.rows = [('on', 'none')]
        elif sql == 'ROLLBACK':
            self.c.info.transaction_status = 0 if runtime.rollback else 2
        else:
            self.rows = []
    def fetchmany(self, size): return self.rows[:size]


class FakeConnection:
    def __init__(self, runtime, channel, user, pid):
        self.runtime, self.channel, self.user, self.pid = runtime, channel, user, pid
        self.info = SimpleNamespace(transaction_status=2)
        self.session_reads = self.rollbacks = 0
        self.closed = False
    def __enter__(self): return self
    def __exit__(self, *args): self.closed = True; return False
    def cursor(self): return FakeCursor(self)
    def rollback(self): self.rollbacks += 1


class FakePsycopg:
    def __init__(self):
        self.calls, self.connections, self.connect_kwargs = [], [], []
        self.permission = self.rls = self.tls = self.rollback = self.policy = True
        self.pid_drift = self.drift = self.present = False
        self.auth_reads = 0
        self.pq = SimpleNamespace(TransactionStatus=SimpleNamespace(IDLE=0))
    def connect(self, **kwargs):
        channel = h.CHANNELS[len(self.connections)]
        connection = FakeConnection(self, channel, kwargs['user'], 100 + len(self.connections))
        self.connections.append(connection); self.connect_kwargs.append(kwargs)
        return connection


class HelperTests(unittest.TestCase):
    def setUp(self):
        self.r = request()
        self.runtime = FakePsycopg()
    def run_fixture(self):
        with patch.object(h, 'require_authority'), patch.object(h, 'read_inputs', return_value={TARGET['cuenta_ref']: ALIAS}), \
             patch.dict(sys.modules, {'psycopg': self.runtime}):
            return h.run_request(self.r)
    def test_production_gate_precedes_reads_and_connect(self):
        with patch.object(h, 'read_inputs') as read, patch.dict(sys.modules, {'psycopg': self.runtime}):
            with self.assertRaisesRegex(h.Refused, 'autoridad_ejecucion_pendiente'):
                h.run_request(self.r)
            read.assert_not_called(); self.assertEqual(self.runtime.connections, [])
    def test_three_sessions_same_helper_and_both_observation_times(self):
        result = self.run_fixture()
        h.validate_result(self.r, result)
        self.assertIs(result['cas_sessions_bound'], True)
        self.assertIs(result['cas_applied'], False)
        self.assertEqual(result['observations']['candidato'], {'lectura_1': TIME1, 'lectura_2': TIME2})
        self.assertEqual(result['readings_sha256'][0], result['readings_sha256'][1])
        self.assertTrue(all(c.closed and c.rollbacks == 1 for c in self.runtime.connections))
        for kwargs in self.runtime.connect_kwargs:
            self.assertEqual(kwargs['sslmode'], 'verify-full')
            self.assertEqual(kwargs['hostaddr'], '127.0.0.1')
            self.assertNotIn('password', kwargs)
        aut_calls = [(sql, args) for channel, sql, args in self.runtime.calls if channel == 'autorizacion']
        self.assertEqual(sum(sql == p.AUTH_SQL for sql, _ in aut_calls), 2)
        self.assertEqual(sum(sql == p.MOTIVES_SQL for sql, _ in aut_calls), 2)
        self.assertTrue(all(args[2] == TARGET['persona_ref'] for sql, args in aut_calls if sql == p.AUTH_SQL))
        self.assertFalse(any(sql.startswith('SET LOCAL ROLE') for sql, _ in aut_calls))
        self.assertEqual(set(result['preimages']['usuarios']), p.CONTEXT_FIELDS)
        output = json.dumps(result)
        for secret in ('cuenta_id_hmac', 'sujeto_id_hmac', ALIAS['cuenta_id_hmac'], ALIAS['sujeto_id_hmac'], TARGET['persona_ref']):
            self.assertNotIn(secret, output)
    def test_permission_rls_tls_drift_and_rollback_refusals(self):
        for field, setting in (('permission', False), ('rls', False), ('policy', False), ('tls', False),
                               ('pid_drift', True), ('drift', True), ('rollback', False)):
            with self.subTest(field=field):
                self.runtime = FakePsycopg()
                setattr(self.runtime, field, setting)
                self.runtime.present = field == 'drift'
                with self.assertRaises(h.Refused):
                    self.run_fixture()
                self.assertTrue(all(c.closed and c.rollbacks == 1 for c in self.runtime.connections))
    def test_request_closed_and_fixed_auth_login(self):
        for field, setting in (('callback', lambda: None), ('sql', 'SELECT 1'), ('connections', {}), ('version', True)):
            self.r = request(); self.r[field] = setting
            with self.assertRaises(h.Refused): h.validate_request(self.r)
        self.r = request(); self.r['users']['autorizacion'] = 'vec_wrong_login'
        with self.assertRaises(h.Refused): h.validate_request(self.r)
    def test_result_closed_and_semantic_pairs(self):
        result = self.run_fixture()
        for change in ('hmac', 'sessions', 'zero', 'pid', 'role', 'times', 'hash'):
            modified = copy.deepcopy(result)
            if change == 'hmac': modified['hmac'] = SHA
            elif change == 'sessions': modified['cas_sessions_bound'] = False
            elif change == 'zero': modified['preimages']['usuarios']['huella_contexto'] = SHA
            elif change == 'pid': modified['sessions']['contexto'] = copy.deepcopy(modified['sessions']['identidad'])
            elif change == 'role': modified['sessions']['autorizacion']['after']['role'] = 'vec_autorizacion_propietario'
            elif change == 'times': modified['observations']['candidato']['lectura_2'] = '2026-09-01T00:00:00Z'
            elif change == 'hash': modified['readings_sha256'][1] = 'b' * 64
            with self.subTest(change=change), self.assertRaises(h.Refused): h.validate_result(self.r, modified)
    def test_private_pins_source_alias_journal_and_tls(self):
        source = {'version': 1, 'datos_sinteticos': True, 'cuentas': [{
            'cuenta_ref': TARGET['cuenta_ref'], 'persona_ref': TARGET['persona_ref'],
            'poblaciones': ['candidato', 'usuarios']}], 'candidatoBolsa': TARGET}
        inputs = {name: p.canonical({'version': 1}) for name in h.INPUT_FILES.values()}
        inputs.update({'source.json': p.canonical(source),
                       'alias.json': p.canonical({'version': 1, 'cuentas': [ALIAS]}),
                       'sql-journal.json': p.canonical({'phase': 'awaiting_ad132', 'installed': [None] * 62})})
        def read(path, maximum=1 << 20):
            return b'fixture tls' if path.parent == Path('/tls') else inputs[path.name]
        self.r['pins'] = {pin: p.digest(inputs[name]) for pin, name in h.INPUT_FILES.items()}
        self.r['tls_hashes'] = {name: p.digest(b'fixture tls') for name in self.r['tls_hashes']}
        with patch.object(h, 'file_bytes', side_effect=read):
            self.assertEqual(h.read_inputs(self.r), {TARGET['cuenta_ref']: ALIAS})
            inputs['source.json'] += b' '
            with self.assertRaisesRegex(h.Refused, 'cas_private_input_changed'):
                h.read_inputs(self.r)
            inputs['source.json'] = p.canonical(source)
            bad_alias = {**ALIAS, 'sujeto_id_hmac': ALIAS['cuenta_id_hmac']}
            inputs['alias.json'] = p.canonical({'version': 1, 'cuentas': [bad_alias]})
            self.r['pins']['alias_sha256'] = p.digest(inputs['alias.json'])
            with self.assertRaisesRegex(h.Refused, 'cas_alias_invalid'):
                h.read_inputs(self.r)
    def test_file_reader_refuses_symlink_hardlink_and_oversize(self):
        with tempfile.TemporaryDirectory() as root:
            path = Path(root) / 'fixture.json'
            path.write_bytes(b'{}')
            self.assertEqual(h.file_bytes(path), b'{}')
            linked = Path(root) / 'symlink.json'
            linked.symlink_to(path)
            with self.assertRaises(OSError): h.file_bytes(linked)
            with self.assertRaises(h.Refused): h.file_bytes(path, 1)
            hard = Path(root) / 'hardlink.json'
            os.link(path, hard)
            with self.assertRaises(h.Refused): h.file_bytes(path)
    def test_json_duplicate_and_nonfinite(self):
        for raw in (b'{"version":1,"version":1}', b'{"value":NaN}'):
            with self.assertRaises(h.Refused): h.decode(raw)
    def test_stdout_refusal_is_neutral(self):
        stdin = SimpleNamespace(buffer=io.BytesIO(p.canonical(self.r)))
        stdout = SimpleNamespace(buffer=io.BytesIO())
        stderr = io.StringIO()
        with patch.object(sys, 'stdin', stdin), patch.object(sys, 'stdout', stdout), patch.object(sys, 'stderr', stderr):
            self.assertEqual(h.main(), 1)
        self.assertEqual(stdout.buffer.getvalue(), b'')
        self.assertEqual(stderr.getvalue(), 'cas_helper_refused\n')


if __name__ == '__main__':
    unittest.main()
