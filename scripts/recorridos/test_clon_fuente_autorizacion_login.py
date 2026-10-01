"""Memory-only synthetic P/A/L fixtures; no file/runtime/approval consumer."""
from dataclasses import replace
import copy
import io
import json
import unittest
from contextlib import redirect_stdout, redirect_stderr

import clon_fuente_autorizacion_login as consumer


def raw(value):
    return (json.dumps(value, sort_keys=True, separators=(',', ':')) + '\n').encode()


def attributes(login):
    return {'login': login, 'inherit': True, 'super': False, 'createdb': False,
            'createrole': False, 'replication': False, 'bypassrls': False,
            'connlimit': -1, 'validuntil': None, 'config': None, 'password_is_null': True}


class ChainTests(unittest.TestCase):
    def setUp(self):
        h = lambda n: format(n, '064x')
        self.physical = dict(pgid=h(1), imagen=h(2), volumen=h(3),
                             system_identifier='123456', base_oid='5')
        self.package = h(4)
        self.clonado = ('\n'.join([self.package, self.physical['pgid'], h(5), h(6), h(7)]) + '\n').encode()
        self.pre = dict(schema_sha256=h(8), roles_sha256=h(9), datacl_sha256=h(10))
        self.post = dict(self.pre, roles_sha256=h(11))
        self.pre_inventory = {'roles': {'postgres': h(12), consumer.GROUP: h(13)},
                              'memberships': [], 'settings': []}
        self.post_inventory = copy.deepcopy(self.pre_inventory)
        self.post_inventory['roles'][consumer.LOGIN] = h(14)
        self.post_inventory['memberships'].append(dict(consumer.MEMBERSHIP))
        self.p = dict(contrato='h6_p_login_clon_v1', estado='R0',
            paquete_sha256=self.package, commit_confirmed=True, intent_sha256=h(15),
            clonado_sha256=consumer.digest(self.clonado), **self.physical,
            post_schema=h(16), post_roles=self.pre['roles_sha256'],
            post_datacl=self.pre['datacl_sha256'])
        self.a = dict(contrato='h6_ext_aut26_clon_v1', paquete_sha256=self.package,
            **{k: self.physical[k] for k in ('pgid', 'imagen', 'volumen')},
            pre_schema=self.p['post_schema'], pre_roles=self.p['post_roles'],
            pre_datacl=self.p['post_datacl'], post_schema=self.pre['schema_sha256'],
            post_roles=self.pre['roles_sha256'], post_datacl=self.pre['datacl_sha256'])
        self.references = dict(order_sha256=h(17), approval_sha256=h(18), script_sha256=h(19))
        self.j = dict(kind='h6_login_fuente_autorizacion_intent', version=1,
            **self.physical, **self.references,
            ad112_receipt_sha256=consumer.digest(raw(self.p)),
            aut26_receipt_sha256=consumer.digest(raw(self.a)), pre=copy.deepcopy(self.pre),
            pre_inventory=copy.deepcopy(self.pre_inventory), login=consumer.LOGIN,
            grupo=consumer.GROUP, operation_id='00000000-0000-4000-8000-000000000001',
            created_at='2026-10-01T01:00:00+00:00')
        self.l = dict(copy.deepcopy(self.j), kind='h6_login_fuente_autorizacion_receipt',
            post=copy.deepcopy(self.post), post_inventory=copy.deepcopy(self.post_inventory),
            commit_confirmed=True, committed_at='2026-10-01T01:00:01+00:00')
        nominal = {name: {'fingerprint': self.post_inventory['roles'][name],
                         'attributes': attributes(name == consumer.LOGIN)}
                   for name in (consumer.LOGIN, consumer.GROUP)}
        self.pins = consumer.Pins(self.hashes(), dict(self.physical), self.package,
            dict(self.references), copy.deepcopy(self.pre), copy.deepcopy(self.post),
            copy.deepcopy(self.pre_inventory), copy.deepcopy(self.post_inventory), nominal,
            consumer.digest(raw(self.j)), consumer.digest(self.clonado),
            dict(a_receipt_sha256=consumer.digest(raw(self.a)),
                 clonado_sha256=consumer.digest(self.clonado), physical=dict(self.physical)))

    def hashes(self):
        return {n: consumer.digest(raw(d)) for n, d in
                zip(('p', 'a', 'intent', 'l'), (self.p, self.a, self.j, self.l))}

    def validate(self, **kwargs):
        return consumer.validate_chain(*(raw(d) for d in (self.p, self.a, self.j, self.l)),
            pins=kwargs.get('pins', self.pins), clonado_raw=kwargs.get('clonado_raw', self.clonado))

    def repin_bytes(self):
        # Simulate an attacker with autoconsistent files. Independent state,
        # physical and inventory pins remain outside that rewritten bundle.
        self.pins = replace(self.pins, receipt_sha256=self.hashes())

    def reject(self, code, **kwargs):
        result = self.validate(**kwargs)
        self.assertFalse(result.ok, result)
        self.assertIn(code, result.codes)
        self.assertIsNone(result.operational_gate)
        return result

    def test_synthetic_candidate_bytes_with_independent_evidence(self):
        self.assertTrue(self.validate().ok, self.validate())
        self.assertIsNone(consumer.OPERATIONAL_GATE)
        self.assertNotIn('intent_sha256', self.l)
        self.assertNotIn('system_identifier', self.a)

    def test_source_only_is_pending_without_external_evidence(self):
        result = self.reject('l_intent_durability_missing', pins=replace(self.pins,
            intent_before_effect_sha256=None, nominal_roles=None, a_physical_binding=None))
        self.assertIn('external.intent_before_effect_sha256', result.required_fields)
        self.assertIn('external.nominal_roles', result.required_fields)
        self.assertIn('external.a_physical_binding', result.required_fields)

    def test_original_whitespace_is_not_canonicalized(self):
        result = consumer.validate_chain(raw(self.p) + b' ', raw(self.a), raw(self.j), raw(self.l),
                                         pins=self.pins, clonado_raw=self.clonado)
        self.assertIn('original_bytes_pin_p', result.codes)
        self.assertIn('p_a_original_bytes_link', result.codes)

    def test_mixed_p_receipt(self):
        self.p['intent_sha256'] = 'f' * 64
        self.repin_bytes()
        self.reject('p_a_original_bytes_link')

    def test_mixed_a_receipt(self):
        self.a['post_schema'] = 'f' * 64
        self.repin_bytes()
        self.reject('p_a_original_bytes_link')
        self.reject('a_l_preimage_mismatch')

    def test_mixed_intent_same_post(self):
        self.j['operation_id'] = '00000000-0000-4000-8000-000000000002'
        self.repin_bytes()
        self.reject('l_intent_receipt_divergence')
        self.reject('l_intent_durability_mismatch')

    def test_self_consistent_forged_role_hashes(self):
        self.l['post']['roles_sha256'] = 'f' * 64
        self.repin_bytes()
        self.reject('l_postimage_pin')

    def test_extra_role_rejected_even_with_rewritten_inventory_pins(self):
        self.l['post_inventory']['roles']['other'] = 'f' * 64
        self.pins = replace(self.pins, post_inventory=copy.deepcopy(self.l['post_inventory']))
        self.repin_bytes()
        self.reject('l_inventory_delta')

    def test_existing_role_changed(self):
        self.l['post_inventory']['roles']['postgres'] = 'f' * 64
        self.repin_bytes()
        self.reject('l_inventory_delta')

    def test_hash_shaped_login_attributes_do_not_suffice(self):
        self.l['post_inventory']['roles'][consumer.LOGIN] = 'f' * 64
        self.repin_bytes()
        self.reject('external_inventory_pins')

    def test_membership_flags_grantor_and_delegation(self):
        original = copy.deepcopy(self.l)
        for key, value in (('admin', True), ('inherit', False), ('set', True),
                           ('grantor', consumer.GROUP), ('role', 'postgres'), ('admin', 0)):
            with self.subTest(key=key, value=value):
                self.l = copy.deepcopy(original)
                self.l['post_inventory']['memberships'][0][key] = value
                self.repin_bytes()
                self.reject('l_inventory_delta')

    def test_duplicate_membership(self):
        self.l['post_inventory']['memberships'].append(dict(consumer.MEMBERSHIP))
        self.repin_bytes()
        self.reject('l_inventory_delta')

    def test_settings_added(self):
        self.l['post_inventory']['settings'].append({'role_oid': 55, 'database_oid': 0,
                                                     'sha256': 'f' * 64})
        self.repin_bytes()
        self.reject('l_inventory_delta')

    def test_a_roles_and_acl_invariants(self):
        for key in ('post_roles', 'post_datacl'):
            with self.subTest(key=key):
                self.a[key] = 'f' * 64
                self.repin_bytes()
                self.reject('a_roles_datacl_changed')

    def test_p_post_does_not_link_a_pre(self):
        self.a['pre_schema'] = 'f' * 64
        self.repin_bytes()
        self.reject('p_a_preimage_mismatch')

    def test_l_schema_or_acl_changes(self):
        for key in ('schema_sha256', 'datacl_sha256'):
            with self.subTest(key=key):
                self.l['post'][key] = 'f' * 64
                self.pins = replace(self.pins, post=copy.deepcopy(self.l['post']))
                self.repin_bytes()
                self.reject('l_delta_not_roles_only')

    def test_pending_and_replay_not_adopted(self):
        for key, value in (('commit_confirmed', False), ('pending', True),
                           ('replay_confirmed', True), ('commit_confirmed', 1)):
            with self.subTest(key=key):
                self.l[key] = value
                self.repin_bytes()
                self.assertFalse(self.validate().ok)

    def test_physical_identity_for_every_record(self):
        for record in (self.p, self.a, self.j, self.l):
            for key in consumer.PHYSICAL:
                with self.subTest(record=record['kind'] if 'kind' in record else record['contrato'], key=key):
                    old = record.get(key)
                    record[key] = '9999' if key in ('system_identifier', 'base_oid') else 'f' * 64
                    self.repin_bytes()
                    self.reject('physical_identity_mismatch')
                    if old is None:
                        del record[key]
                    else:
                        record[key] = old

    def test_foreign_or_mutated_clonado(self):
        self.reject('clonado_original_bytes_pin', clonado_raw=self.clonado + b' ')
        binding = dict(self.pins.a_physical_binding, a_receipt_sha256='f' * 64)
        self.reject('a_physical_binding_mismatch', pins=replace(self.pins, a_physical_binding=binding))

    def test_nominal_flags_cannot_be_inferred_from_role_hash(self):
        for flag in ('login', 'super', 'inherit', 'bypassrls', 'password_is_null'):
            with self.subTest(flag=flag):
                evidence = copy.deepcopy(self.pins.nominal_roles)
                evidence[consumer.LOGIN]['attributes'][flag] = not evidence[consumer.LOGIN]['attributes'][flag]
                self.reject('nominal_attributes_mismatch', pins=replace(self.pins, nominal_roles=evidence))

    def test_reference_pin_not_self_asserted(self):
        self.j['approval_sha256'] = self.l['approval_sha256'] = 'f' * 64
        self.repin_bytes()
        self.reject('external_reference_pins')

    def test_version_bool_uuid_timezone_and_time_order(self):
        original = copy.deepcopy(self.j)
        for key, value in (('version', True), ('operation_id', 'a' * 36),
                           ('created_at', '2026-10-01T01:00:00'),
                           ('created_at', '2026-10-02T01:00:00+00:00')):
            with self.subTest(key=key):
                self.j = dict(original, **{key: value})
                self.repin_bytes()
                self.assertFalse(self.validate().ok)

    def test_duplicate_keys_deep_json_oversize_and_no_output(self):
        for bad in (b'{"contrato":"x","contrato":"y"}', b'{' + b' ' * consumer.MAX_BYTES + b'}',
                    b'NaN', b'[]', b'\xff', b'{"password":"synthetic-private-secret"}',
                    b'{"x":' + b'[' * 1200 + b'0' + b']' * 1200 + b'}'):
            with self.subTest(length=len(bad)):
                out, err = io.StringIO(), io.StringIO()
                with redirect_stdout(out), redirect_stderr(err):
                    result = consumer.validate_chain(bad, raw(self.a), raw(self.j), raw(self.l), pins=self.pins)
                self.assertFalse(result.ok)
                self.assertEqual(out.getvalue() + err.getvalue(), '')
                self.assertNotIn('synthetic-private-secret', repr(result))


if __name__ == '__main__':
    unittest.main()
