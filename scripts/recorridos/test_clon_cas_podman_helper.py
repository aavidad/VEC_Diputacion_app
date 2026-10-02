"""Podman CAS preparation tests use fake drivers; no live SQL or runtime."""
import copy
import io
import json
import os
from pathlib import Path
import sys
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch

import clon_cas_podman_helper as h
import clon_cas_podman_contract as c
import clon_preimagenes_nominales as p
from test_clon_cas_podman_contract import request, sql_observation
from test_clon_preimagenes_nominales import ALIAS, TIME1, TIME2, auth, motives


class FakeCursor:
    def __init__(self, connection):
        self.c = connection
        self.rows = []
        self.closed = False

    def close(self):
        self.closed = True
        if self.c.runtime.fault == 'cursor_close':
            raise RuntimeError('provider fixture secret')

    def execute(self, sql, parameters=()):
        r = self.c.runtime
        r.calls.append((self.c.channel, sql, parameters))
        if sql == h.SESSION_SQL:
            self.c.session_reads += 1
            role = h.ROLES[self.c.channel]
            user = self.c.user if role == 'none' else role
            self.rows = [(self.c.user, user, role, 'on', 'read committed', self.c.pid,
                          r.request['pgid']['database_name'], r.request['pgid']['database_oid'],
                          r.request['pgid']['system_identifier'], '127.0.0.1', '127.0.0.1', 5432,
                          r.fault != 'unsafe_role', r.fault != 'tls', 'TLSv1.2' if r.fault == 'tls12' else 'TLSv1.3', True)]
            if r.fault == 'session_drift' and self.c.session_reads > 1:
                row = list(self.rows[0]); row[5] += 10; self.rows = [tuple(row)]
        elif sql == h.TABLE_SQL:
            self.rows = [(True, True, r.fault != 'rls', r.fault != 'permission', True, r.fault != 'policy')]
        elif sql == h.FUNCTION_SQL:
            self.rows = [(True, True, r.fault != 'permission', True)]
        elif sql == h.AUTH_TABLE_SQL:
            self.rows = [(r.fault == 'auth_direct', False)]
        elif sql == h.ACL_SQL or sql == h.RLS_SQL:
            key = 'acl' if sql == h.ACL_SQL else 'rls'
            value = copy.deepcopy(r.metadata[key])
            if r.fault == key + '_drift':
                value['extra'] = True
            self.rows = [(value,)]
        elif sql in (p.ACCOUNT_SQL, p.ALIAS_SQL, p.CONTEXT_SQL):
            self.rows = []
        elif sql == p.AUTH_SQL:
            if len(r.connections) != 3 or any(x.closed for x in r.connections):
                raise AssertionError('sessions are not simultaneous')
            r.auth_reads += 1
            value = auth(TIME1 if r.auth_reads == 1 else TIME2, present=True)
            value['principal_id'] = r.request['targets']['candidato']['persona_ref']
            value['perfil_activo_ref'] = r.request['targets']['candidato']['perfil_ref']
            if r.fault == 'preimage_drift' and r.auth_reads > 1:
                value['control_rol']['huella_sha256'] = 'b' * 64
            self.rows = [(value,)]
        elif sql == p.MOTIVES_SQL:
            self.rows = [(motives(TIME1 if r.auth_reads == 1 else TIME2),)]
        elif sql == h.ROLLBACK_SQL:
            self.rows = [('on', 'none')]
            if r.fault == 'rollback_role':
                self.rows = [('on', h.ROLES['identidad'])]
        elif sql == 'ROLLBACK':
            self.c.info.transaction_status = 2 if r.fault == 'rollback' else 0
            self.rows = []
        elif sql == 'BEGIN ISOLATION LEVEL READ COMMITTED READ ONLY':
            self.c.info.transaction_status = 2
            self.rows = []
        else:
            self.rows = []

    def fetchmany(self, maximum):
        return self.rows[:maximum]


class FakeConnection:
    def __init__(self, runtime, channel, user, pid):
        self.runtime, self.channel, self.user, self.pid = runtime, channel, user, pid
        self.info = SimpleNamespace(transaction_status=0)
        self.session_reads = self.rollbacks = 0
        self.closed = False
        self.cursors = []

    def cursor(self):
        cursor = FakeCursor(self)
        self.cursors.append(cursor)
        if self.runtime.fault == 'cursor' and self.channel == 'contexto':
            raise RuntimeError('provider fixture secret')
        return cursor

    def rollback(self):
        self.rollbacks += 1
        self.info.transaction_status = 2 if self.runtime.fault == 'cleanup' else 0

    def close(self):
        self.closed = self.runtime.fault != 'close'


class FakeDriver:
    def __init__(self, req, metadata):
        self.request, self.metadata = req, metadata
        self.calls, self.connections, self.connect_kwargs = [], [], []
        self.auth_reads = 0
        self.fault = None
        self.pq = SimpleNamespace(TransactionStatus=SimpleNamespace(IDLE=0, INTRANS=2))

    def connect(self, **kwargs):
        channel = h.CHANNELS[len(self.connections)]
        if self.fault == 'connect' and channel == 'contexto':
            raise RuntimeError('provider fixture secret')
        pid = 100 if self.fault == 'duplicate_pid' else 100 + len(self.connections)
        connection = FakeConnection(self, channel, kwargs['user'], pid)
        self.connections.append(connection)
        self.connect_kwargs.append(kwargs)
        return connection


class HelperTests(unittest.TestCase):
    def setUp(self):
        self.req = request()
        self.metadata = {'aliases': {t['cuenta_ref']: {**ALIAS, 'cuenta_ref': t['cuenta_ref'],
                                    'cuenta_id_hmac': '0123456789abcdef' * 4,
                                    'sujeto_id_hmac': 'fedcba9876543210' * 4}
                                    for t in self.req['targets'].values()},
                         'acl': {'roles': [], 'memberships': [], 'tables': [], 'functions': []},
                         'rls': {'tables': []}}
        self.driver = FakeDriver(self.req, self.metadata)
        self.observed_state = copy.deepcopy(self.req['receipt_bindings']['post'])
        self.observer_fault = None
        self.sql_observations = []

    def observe(self, stage, req, initial_sessions):
        self.assertEqual(len(self.driver.connections), 3)
        self.assertTrue(all(not x.closed and not x.rollbacks and
                            x.info.transaction_status == 2 for x in self.driver.connections))
        self.assertFalse(any(sql == 'ROLLBACK' for _, sql, _ in self.driver.calls))
        self.assertEqual(self.driver.auth_reads, 0 if stage == 'before' else 2)
        self.assertTrue(all(x.session_reads == (1 if stage == 'before' else 2)
                            for x in self.driver.connections))
        self.assertEqual(sum(sql == h.ACL_SQL for _, sql, _ in self.driver.calls),
                         1 if stage == 'before' else 2)
        self.assertEqual(sum(sql == h.RLS_SQL for _, sql, _ in self.driver.calls),
                         1 if stage == 'before' else 2)
        repetitions = 1 if stage == 'before' else 2
        self.assertEqual(sum(sql == h.TABLE_SQL for _, sql, _ in self.driver.calls), 4 * repetitions)
        self.assertEqual(sum(sql == h.FUNCTION_SQL for _, sql, _ in self.driver.calls), 3 * repetitions)
        self.assertEqual(sum(sql == h.AUTH_TABLE_SQL for _, sql, _ in self.driver.calls), 10 * repetitions)
        self.driver.calls.append((None, 'OBSERVER:' + stage, ()))
        if self.observer_fault == (stage, 'error'):
            raise RuntimeError('provider fixture secret')
        if self.observer_fault == (stage, 'timeout'):
            raise TimeoutError('provider fixture secret')
        if self.observer_fault == (stage, 'secret_refusal'):
            raise h.Refused('provider fixture secret')
        observation = sql_observation(req, initial_sessions, stage, state=self.observed_state)
        if self.observer_fault and self.observer_fault[0] == stage:
            field = self.observer_fault[1]
            if field in c.STATE_FIELDS or field in (
                    'nonce', 'request_sha256', 'sessions_sha256', 'pg_container_id', 'normalizer_sha256'):
                observation[field] = 'b' * 64
            elif field == 'stage':
                observation[field] = 'after' if stage == 'before' else 'before'
            elif field == 'system_identifier':
                observation[field] = '2'
            elif field == 'database_oid':
                observation[field] += 1
            elif field == 'pg_image_id':
                observation[field] = 'sha256:' + 'b' * 64
            elif field == 'normalizer_bundle':
                observation[field]['h6_comun.sh'] = 'b' * 64
            elif field == 'postgres_process':
                observation[field]['starttick'] += 1
            elif field == 'network_namespace':
                observation[field]['ino'] += 1
            elif field == 'mount_inventory_sha256':
                observation[field] = 'b' * 64
            elif field == 'missing':
                del observation['sessions_sha256']
            elif field == 'extra':
                observation['private'] = 'provider fixture secret'
            elif field == 'closed':
                self.driver.connections[0].closed = True
            elif field == 'idle':
                self.driver.connections[0].info.transaction_status = 0
        self.sql_observations.append(copy.deepcopy(observation))
        return observation

    def collect(self):
        return h._fixture_core(self.req, self.driver, lambda _: copy.deepcopy(self.metadata), self.observe)

    def test_gate_refuses_even_approved_before_any_io(self):
        for req in (self.req, {**self.req, 'approved': True}, {'approved': True}):
            with patch.object(h, 'read_inputs') as read, patch.object(h, '_fixture_core') as core, \
                    patch.object(h.os, 'open') as opened:
                with self.assertRaisesRegex(h.Refused, 'autoridad_ejecucion_pendiente'):
                    h.run_request(req)
                read.assert_not_called(); core.assert_not_called(); opened.assert_not_called()
        with patch.object(h, 'EXECUTION_AUTHORITY', object()):
            with self.assertRaisesRegex(h.Refused, 'autoridad_ejecucion_pendiente'):
                h.run_request(self.req)
        self.assertEqual(self.driver.connections, [])

    def test_bound_three_sessions_tls_two_readings_minimal_preimages_and_cleanup(self):
        result = self.collect()
        c.bind_result(self.req, result)
        self.assertEqual(result['observations']['candidato'], {'lectura_1': TIME1, 'lectura_2': TIME2})
        self.assertEqual(result['readings_sha256'][0], result['readings_sha256'][1])
        self.assertEqual(set(result['preimages']['usuarios']), p.CONTEXT_FIELDS)
        self.assertEqual(set(result['preimages']['candidato']), p.CAS_FIELDS)
        self.assertIs(result['cas_applied'], False)
        self.assertEqual(result['sql_postimage_observations'],
                         dict(zip(('before', 'after'), self.sql_observations)))
        self.assertEqual(result['postimage'], self.observed_state)
        self.assertIsNot(result['postimage'], self.req['receipt_bindings']['post'])
        self.assertTrue(all(x.closed and x.rollbacks == 1 and all(y.closed for y in x.cursors)
                            for x in self.driver.connections))
        for options in self.driver.connect_kwargs:
            self.assertEqual(options['sslmode'], 'verify-full')
            self.assertEqual(options['hostaddr'], '127.0.0.1')
            self.assertEqual(options['dbname'], self.req['pgid']['database_name'])
            self.assertNotIn('password', options)
            self.assertIn('idle_in_transaction_session_timeout=10000', options['options'])
        aut = [(sql, params) for channel, sql, params in self.driver.calls if channel == 'autorizacion']
        self.assertEqual(sum(sql == p.AUTH_SQL for sql, _ in aut), 2)
        self.assertEqual(sum(sql == p.MOTIVES_SQL for sql, _ in aut), 2)
        self.assertFalse(any(sql.startswith('SET LOCAL ROLE') for sql, _ in aut))
        self.assertTrue(all(params[2:] == (self.req['targets']['candidato']['persona_ref'],
                          self.req['targets']['candidato']['perfil_ref']) for sql, params in aut if sql == p.AUTH_SQL))
        serialized = json.dumps(result)
        for private in ('cuenta_id_hmac', 'sujeto_id_hmac', '0123456789abcdef' * 4, 'fedcba9876543210' * 4):
            self.assertNotIn(private, serialized)

    def test_role_rls_acl_tls_drift_and_failures_close_every_open_connection(self):
        for fault in ('unsafe_role', 'rls', 'permission', 'policy', 'auth_direct', 'tls', 'tls12', 'acl_drift', 'rls_drift',
                      'session_drift', 'duplicate_pid', 'preimage_drift', 'rollback', 'rollback_role',
                      'connect', 'cursor', 'cursor_close', 'cleanup', 'close'):
            with self.subTest(fault=fault):
                self.driver = FakeDriver(self.req, self.metadata)
                self.driver.fault = fault
                with self.assertRaises(h.Refused) as error:
                    self.collect()
                self.assertNotIn('provider fixture secret', str(error.exception))
                self.assertTrue(all(x.rollbacks == 1 for x in self.driver.connections))
                if fault != 'close':
                    self.assertTrue(all(x.closed for x in self.driver.connections))

    def test_private_inputs_change_between_observations_refused(self):
        reads = 0
        def read(_):
            nonlocal reads
            reads += 1
            value = copy.deepcopy(self.metadata)
            if reads > 1:
                value['rls']['changed'] = True
            return value
        with self.assertRaisesRegex(h.Refused, 'cas_private_input_changed'):
            h._fixture_core(self.req, self.driver, read, self.observe)
        self.assertTrue(all(x.closed and x.rollbacks == 1 for x in self.driver.connections))

    def test_external_full_state_detects_body_acl_and_settings_drift(self):
        # The nominal catalog and preimages stay identical; only H6 sees the
        # full function body, column/default ACL and role/database settings.
        for field in c.STATE_FIELDS:
            with self.subTest(field=field):
                self.driver = FakeDriver(self.req, self.metadata)
                self.observer_fault = ('after', field)
                with self.assertRaises(h.Refused):
                    self.collect()
                self.assertTrue(all(x.closed and x.rollbacks == 1 for x in self.driver.connections))
                self.assertEqual(self.driver.auth_reads, 2)
                self.assertFalse(any(sql == 'ROLLBACK' for _, sql, _ in self.driver.calls))

    def test_sql_observer_barrier_binding_errors_and_timeouts_cleanup(self):
        fields = ('error', 'timeout', 'secret_refusal', 'stage', 'nonce', 'request_sha256', 'sessions_sha256',
                  'pg_container_id', 'pg_image_id', 'system_identifier', 'database_oid',
                  'normalizer_sha256', 'normalizer_bundle', 'postgres_process', 'network_namespace',
                  'mount_inventory_sha256', 'missing', 'extra', 'closed', 'idle')
        for stage in ('before', 'after'):
            for field in fields:
                with self.subTest(stage=stage, field=field):
                    self.driver = FakeDriver(self.req, self.metadata)
                    self.observer_fault = (stage, field)
                    with self.assertRaises(h.Refused) as error:
                        self.collect()
                    self.assertNotIn('provider fixture secret', str(error.exception))
                    self.assertTrue(all(x.closed and x.rollbacks == 1 for x in self.driver.connections))
                    self.assertFalse(any(sql == 'ROLLBACK' for _, sql, _ in self.driver.calls))

    def test_observer_cannot_rewrite_request_or_session_anchors(self):
        def observe(stage, req, initial_sessions):
            observation = self.observe(stage, req, initial_sessions)
            req['receipt_bindings']['post']['schema_sha256'] = 'b' * 64
            initial_sessions['identidad']['backend_pid'] += 1
            return observation
        original = copy.deepcopy(self.req)
        result = h._fixture_core(self.req, self.driver, lambda _: copy.deepcopy(self.metadata), observe)
        self.assertEqual(self.req, original)
        self.assertEqual(result['postimage'], self.observed_state)
        self.assertEqual(result['sessions']['identidad']['before']['backend_pid'], 100)

    def test_pinned_private_inputs_hba_and_tls_bytes(self):
        targets = self.req['targets']
        accounts = {}
        for population, target in targets.items():
            account = accounts.setdefault(target['cuenta_ref'], {'cuenta_ref': target['cuenta_ref'],
                'persona_ref': target['persona_ref'], 'poblaciones': []})
            account['poblaciones'].append(population)
        source = {'version': 1, 'datos_sinteticos': True, 'cuentas': list(accounts.values()),
                  'candidatoBolsa': targets['candidato']}
        inputs = {name: c.canonical({'version': 1}) for name in h.INPUT_FILES.values()}
        inputs.update({'source.json': c.canonical(source),
                       'alias.json': c.canonical({'version': 1, 'cuentas': list(self.metadata['aliases'].values())}),
                       'sql-journal.json': c.canonical({'phase': 'awaiting_ad132', 'installed': [None] * 62}),
                       'pg_hba.conf': b'fixture exact HBA bytes\n',
                       'acl.json': c.canonical(self.metadata['acl']), 'rls.json': c.canonical(self.metadata['rls'])})
        self.req['pins'].update({pin: c.digest(inputs[name]) for pin, name in h.INPUT_FILES.items()})
        self.req['tls_hashes'] = {name: c.digest(b'fixture tls') for name in self.req['tls_hashes']}
        def read(path, maximum=1 << 20):
            return b'fixture tls' if path.parent == Path('/tls') else inputs[path.name]
        with patch.object(h, 'file_bytes', side_effect=read):
            self.assertEqual(h.read_inputs(self.req), self.metadata)
            inputs['pg_hba.conf'] += b'changed'
            with self.assertRaisesRegex(h.Refused, 'cas_private_input_changed'):
                h.read_inputs(self.req)

    def test_file_reader_rejects_links_size_and_changed_identity(self):
        with tempfile.TemporaryDirectory() as root:
            path = Path(root) / 'fixture.json'; path.write_bytes(b'{}')
            self.assertEqual(h.file_bytes(path), b'{}')
            linked = Path(root) / 'link.json'; linked.symlink_to(path)
            with self.assertRaises(OSError): h.file_bytes(linked)
            with self.assertRaises(h.Refused): h.file_bytes(path, 1)
            os.link(path, Path(root) / 'hard.json')
            with self.assertRaises(h.Refused): h.file_bytes(path)

    def test_cli_refusal_never_returns_payload_or_provider_error(self):
        stdin = SimpleNamespace(buffer=io.BytesIO(c.canonical({**self.req, 'approved': True})))
        stdout = SimpleNamespace(buffer=io.BytesIO()); stderr = io.StringIO()
        with patch.object(sys, 'stdin', stdin), patch.object(sys, 'stdout', stdout), patch.object(sys, 'stderr', stderr):
            self.assertEqual(h.main(), 1)
        self.assertEqual(stdout.buffer.getvalue(), b'')
        self.assertEqual(stderr.getvalue(), 'cas_helper_refused\n')


if __name__ == '__main__':
    unittest.main()
