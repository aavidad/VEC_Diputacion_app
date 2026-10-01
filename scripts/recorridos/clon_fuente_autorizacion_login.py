"""Pure, preparatory P -> A -> L receipt validation. No files, SQL or runtime.

Pins must come from an independent, reviewed source, never from these receipts.
Successful byte validation is technical evidence only, not operational approval.
The D candidate links L to its durable intention by exact field copies. Its
original intention bytes must be pinned independently before the effect, under
the reviewed source's fsync protocol. A's missing physical database identity
requires external evidence linked to original A/CLONADO bytes. The future
file/live consumer owns acquisition; missing evidence stays pending.
Operational use with the local /dev/shm bind clone is NO-GO until D supplies
a separately reviewed physical variant; volumen remains a hex64 volume ID.
"""
from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime
import hashlib
import json
import re
from uuid import UUID

OPERATIONAL_GATE = None
MAX_BYTES = 131072
LOGIN = 'vec_externo_v3_fuente_autorizacion_desarrollo'
GROUP = 'vec_autorizacion_fuente_externa'
P_LOGIN = 'vec_externo_preflight_v3_desarrollo'
P_GROUP = 'vec_autorizacion_atestada_v3_preflight_externo'
P_HELPER_SHA256 = '3d7c76f73b2fc7b33e59bb1a1e91b92c54b0aaaec4f6cda0c7dfbe48b9f86d45'
A_COMMIT = '9595970a7a591ae2e2e043c1e4b14e6b62e3cd8f'
A_SQL_SHA256 = '1a651f21808bb7c19eae7b0534aaec4ab6d71c3f20a082af67a6aa2d67e4cbdc'
PHYSICAL = ('pgid', 'imagen', 'volumen', 'system_identifier', 'base_oid')
STATES = ('schema_sha256', 'roles_sha256', 'datacl_sha256')
P_BASE_KEYS = frozenset(('paquete_sha256', 'clonado_sha256', 'sql_func_sha256',
    *PHYSICAL, 'sql_sha256', 'post_sql_sha256', 'acl_ro_sha256', 'roles_ro_sha256',
    'script_sha256', 'helper_sha256', 'pre_schema', 'pre_roles', 'pre_datacl',
    'pre_role_inventory_sha256', 'login', 'grupo', 'effect'))
P_REF_KEYS = frozenset(('approval_sha256', 'review1_sha256', 'review2_sha256',
    'trial_receipt_sha256', 'approval_path', 'review1_path', 'review2_path',
    'trial_path', 'trial_journal_path'))
P_INTENT_KEYS = P_BASE_KEYS | P_REF_KEYS | {'contrato', 'pre_inventory'}
P_RECEIPT_KEYS = P_INTENT_KEYS | {'estado', 'intent_sha256', 'commit_confirmed',
    'post_schema', 'post_roles', 'post_datacl', 'post_inventory', 'post_role_inventory_sha256'}
A_KEYS = frozenset(('contrato', 'paquete_sha256', 'pgid', 'imagen', 'volumen',
    'commit', 'sql_sha256', 'pre_schema', 'pre_roles', 'pre_datacl',
    'post_schema', 'post_roles', 'post_datacl'))
P_EFFECT = {'create_role': 'LOGIN INHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD NULL',
    'grant_role': P_GROUP, 'admin': False, 'inherit': True, 'set': False,
    'additional_grants': False}
INTENT_KEYS = frozenset(('kind', 'version', 'aut26_receipt_sha256',
    'ad112_receipt_sha256', *PHYSICAL, 'pre', 'pre_inventory', 'login',
    'grupo', 'operation_id', 'created_at', 'order_sha256', 'approval_sha256',
    'script_sha256'))
RECEIPT_KEYS = INTENT_KEYS | {'post', 'post_inventory', 'commit_confirmed',
                            'committed_at', 'intent_sha256'}
MEMBERSHIP = {'role': GROUP, 'member': LOGIN, 'grantor': 'postgres',
              'admin': False, 'inherit': True, 'set': False}


@dataclass(frozen=True, repr=False)
class Pins:
    """Independent original-byte, state and inventory pins; never CLI input.

nominal_roles maps LOGIN/GROUP to {fingerprint, attributes}. The fingerprint
is the PostgreSQL SQL_INVENTORY digest from an independently verified snapshot;
attributes are its separately observed, password-free projection. This module
does not reconstruct PostgreSQL jsonb hashes from Python JSON serialization.
intent_before_effect_sha256 is captured after the reviewed writer's file and
directory fsync and before SQL, never reconstructed from the final receipt.
a_physical_binding links independently observed A physics to A/CLONADO bytes.
p_intent_before_effect_sha256 pins the original P intention independently;
the caller must supply those bytes as p_intent_raw, never infer its digest.
"""
    receipt_sha256: dict
    physical: dict
    package_sha256: str
    references: dict
    pre: dict
    post: dict
    pre_inventory: dict
    post_inventory: dict
    nominal_roles: dict | None = field(default=None)
    intent_before_effect_sha256: str | None = None
    clonado_sha256: str | None = None
    a_physical_binding: dict | None = None
    p_intent_before_effect_sha256: str | None = None


@dataclass(frozen=True)
class Validation:
    codes: tuple[str, ...]
    required_fields: tuple[str, ...] = ()
    operational_gate: None = None

    @property
    def ok(self) -> bool:
        return not self.codes


def digest(raw: bytes) -> str:
    return hashlib.sha256(raw).hexdigest()


def _hex(value) -> bool:
    return isinstance(value, str) and re.fullmatch('[0-9a-f]{64}', value) is not None


def _same(left, right) -> bool:
    # Python equates True with 1; security-sensitive JSON comparisons must not.
    return json.dumps(left, sort_keys=True, separators=(',', ':'), allow_nan=False) == \
        json.dumps(right, sort_keys=True, separators=(',', ':'), allow_nan=False)


def _pairs(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError()
        result[key] = value
    return result


def _secret(value) -> bool:
    if isinstance(value, dict):
        return any(key.lower() in {'password', 'rolpassword', 'token', 'dsn',
            'secret', 'private_key', 'authorization', 'credential'} or _secret(v)
            for key, v in value.items())
    return isinstance(value, list) and any(_secret(v) for v in value)


def _decode(raw):
    if type(raw) is not bytes or not 0 < len(raw) <= MAX_BYTES:
        raise ValueError()
    def invalid_constant(_):
        raise ValueError()
    value = json.loads(raw.decode('utf-8'), object_pairs_hook=_pairs,
                       parse_constant=invalid_constant)
    if not isinstance(value, dict) or _secret(value):
        raise ValueError()
    return value


def _state(value) -> bool:
    return isinstance(value, dict) and set(value) == set(STATES) and \
        all(_hex(v) for v in value.values())


def _inventory(value) -> bool:
    if not isinstance(value, dict) or set(value) != {'roles', 'memberships', 'settings'}:
        return False
    roles, memberships, settings = (value[k] for k in ('roles', 'memberships', 'settings'))
    if not isinstance(roles, dict) or not roles or len(roles) > 2048 or \
            any(not isinstance(k, str) or not 0 < len(k) <= 63 or not _hex(v)
                for k, v in roles.items()):
        return False
    if not isinstance(memberships, list) or not isinstance(settings, list):
        return False
    if len(memberships) > 8192 or len(settings) > 8192:
        return False
    for m in memberships:
        if not isinstance(m, dict) or set(m) != set(MEMBERSHIP) or \
                any(type(m[k]) is not bool for k in ('admin', 'inherit', 'set')) or \
                any(m[k] not in roles for k in ('role', 'member', 'grantor')):
            return False
    for s in settings:
        if not isinstance(s, dict) or set(s) != {'role_oid', 'database_oid', 'sha256'} or \
                any(type(s[k]) is not int or s[k] < 0 for k in ('role_oid', 'database_oid')) or \
                not _hex(s['sha256']):
            return False
    canonical = [json.dumps(m, sort_keys=True) for m in memberships]
    return len(set(canonical)) == len(canonical)


def _delta(pre, post, login=LOGIN, group=GROUP) -> bool:
    if not _inventory(pre) or not _inventory(post) or login in pre['roles']:
        return False
    if set(post['roles']) != set(pre['roles']) | {login} or \
            any(post['roles'][k] != v for k, v in pre['roles'].items()) or \
            not _same(pre['settings'], post['settings']):
        return False
    a = [json.dumps(m, sort_keys=True) for m in pre['memberships']]
    b = [json.dumps(m, sort_keys=True) for m in post['memberships']]
    member = dict(MEMBERSHIP, role=group, member=login)
    return sorted(b) == sorted(a + [json.dumps(member, sort_keys=True)]) and \
        all(m['member'] != group and m['role'] != login for m in post['memberships'])


def _nominal(value, inventory) -> bool:
    if not isinstance(value, dict) or set(value) != {LOGIN, GROUP}:
        return False
    for name, login in ((LOGIN, True), (GROUP, False)):
        evidence = value[name]
        expected = {'login': login, 'inherit': True, 'super': False, 'createdb': False,
            'createrole': False, 'replication': False, 'bypassrls': False,
            'connlimit': -1, 'validuntil': None, 'config': None, 'password_is_null': True}
        if not isinstance(evidence, dict) or set(evidence) != {'fingerprint', 'attributes'} or \
                evidence['fingerprint'] != inventory['roles'].get(name) or \
                not _same(evidence['attributes'], expected):
            return False
    return True


def _time(value):
    if not isinstance(value, str) or len(value) > 64:
        raise ValueError()
    t = datetime.fromisoformat(value)
    if t.utcoffset() is None:
        raise ValueError()
    return t


def validate_chain(p_raw: bytes, a_raw: bytes, intent_raw: bytes, l_raw: bytes,
                   *, pins: Pins, clonado_raw: bytes | None = None,
                   p_intent_raw: bytes | None = None) -> Validation:
    """Validate original bytes and independently pinned observations.

Returns fixed codes and fixed missing-field identifiers only. It never returns
input content, private paths, hashes, principal names or provider exceptions.
"""
    errors, missing = [], []
    def check(ok, code):
        if not ok and code not in errors:
            errors.append(code)
    def need(ok, name, code):
        if not ok:
            missing.append(name)
        check(ok, code)
    try:
        records = tuple(_decode(raw) for raw in (p_raw, a_raw, intent_raw, l_raw))
        p, a, j, l = records
        if not isinstance(pins, Pins):
            return Validation(('external_pins_required',))
        need(clonado_raw is not None, 'CLONADO.original_bytes', 'clonado_original_bytes_missing')
        if clonado_raw is not None and (type(clonado_raw) is not bytes or
                                       not 0 < len(clonado_raw) <= 4096):
            return Validation(('clonado_bytes_invalid',))
        for name, raw in zip(('p', 'a', 'intent', 'l'), (p_raw, a_raw, intent_raw, l_raw)):
            expected = pins.receipt_sha256.get(name)
            check(_hex(expected) and digest(raw) == expected, 'original_bytes_pin_' + name)
        check(p.get('contrato') == 'h6_p_login_clon_v1' and p.get('estado') == 'R0' and
              p.get('commit_confirmed') is True and set(p) == P_RECEIPT_KEYS and
              all(_hex(v) and v != '0' * 64 for k, v in p.items() if k.endswith('sha256')) and
              p.get('login') == P_LOGIN and p.get('grupo') == P_GROUP and
              _same(p.get('effect'), P_EFFECT) and p.get('helper_sha256') == P_HELPER_SHA256 and
              all(isinstance(p.get(k), str) and p[k].startswith('/') and len(p[k]) <= 4096
                  for k in P_REF_KEYS if k.endswith('_path')), 'p_receipt_contract')
        need(p_intent_raw is not None and _hex(pins.p_intent_before_effect_sha256),
             'external.p_intent_original_bytes_and_prior_pin', 'p_intent_evidence_missing')
        if p_intent_raw is not None:
            pj = _decode(p_intent_raw)
            check(pj.get('contrato') == 'h6_p_login_intento_v1' and set(pj) == P_INTENT_KEYS,
                  'p_intent_contract')
            check(digest(p_intent_raw) == p.get('intent_sha256') == pins.p_intent_before_effect_sha256,
                  'p_intent_original_bytes_pin')
            check(all(_same(p.get(k), v) for k, v in pj.items() if k != 'contrato'),
                  'p_intent_receipt_divergence')
        check(a.get('contrato') == 'h6_ext_aut26_clon_v1' and set(a) == A_KEYS and
              a.get('commit') == A_COMMIT and a.get('sql_sha256') == A_SQL_SHA256,
              'a_receipt_contract')
        check(j.get('kind') == 'h6_login_fuente_autorizacion_intent' and
              type(j.get('version')) is int and j['version'] == 1 and set(j) == INTENT_KEYS,
              'l_intent_contract')
        check(l.get('kind') == 'h6_login_fuente_autorizacion_receipt' and
              type(l.get('version')) is int and l['version'] == 1 and
              set(l) - {'intent_sha256'} == RECEIPT_KEYS - {'intent_sha256'}, 'l_receipt_contract')
        check(l.get('commit_confirmed') is True, 'l_commit_unconfirmed')
        if 'intent_sha256' in l:
            check(l['intent_sha256'] == digest(intent_raw), 'l_intent_bytes_mismatch')
        need(_hex(pins.intent_before_effect_sha256), 'external.intent_before_effect_sha256',
             'l_intent_durability_missing')
        check(pins.intent_before_effect_sha256 == digest(intent_raw), 'l_intent_durability_mismatch')
        for record in (j, l):
            check(record.get('ad112_receipt_sha256') == digest(p_raw) and
                  record.get('aut26_receipt_sha256') == digest(a_raw), 'p_a_original_bytes_link')
        check(all(_same(l.get(k), v) for k, v in j.items() if k != 'kind'),
              'l_intent_receipt_divergence')
        check(_hex(pins.package_sha256) and all(x.get('paquete_sha256') == pins.package_sha256
              for x in (p, a)), 'package_pin')
        check(set(pins.references) == {'order_sha256', 'approval_sha256', 'script_sha256'} and
              all(_hex(v) and j.get(k) == v and l.get(k) == v
                  for k, v in pins.references.items()), 'external_reference_pins')
        check(set(pins.physical) == set(PHYSICAL) and
              all(_hex(pins.physical[k]) for k in ('pgid', 'imagen', 'volumen')) and
              all(isinstance(pins.physical[k], str) and
                  re.fullmatch('[1-9][0-9]{0,19}', pins.physical[k])
                  for k in ('system_identifier', 'base_oid')), 'physical_pins_invalid')
        # A v1 carries only PGID/image/volume. Its database identity must be
        # observed independently and bound to the exact A and CLONADO bytes.
        a_physical = dict(a)
        need(pins.a_physical_binding is not None, 'external.a_physical_binding',
             'a_physical_evidence_missing')
        if clonado_raw is not None:
            clon_sha = digest(clonado_raw)
            lines = clonado_raw.decode('utf-8').splitlines()
            check(_hex(pins.clonado_sha256) and clon_sha == pins.clonado_sha256 and
                  p.get('clonado_sha256') == clon_sha, 'clonado_original_bytes_pin')
            check(len(lines) == 5 and lines[:2] == [pins.package_sha256, pins.physical['pgid']]
                  and all(_hex(v) for v in lines[2:]), 'clonado_contract')
        if pins.a_physical_binding is not None:
            binding = pins.a_physical_binding
            check(set(binding) == {'a_receipt_sha256', 'clonado_sha256', 'physical'} and
                  binding['a_receipt_sha256'] == digest(a_raw) and
                  binding['clonado_sha256'] == pins.clonado_sha256 and
                  _same(binding['physical'], pins.physical), 'a_physical_binding_mismatch')
            for k in ('system_identifier', 'base_oid'):
                a_physical[k] = binding['physical'].get(k)
        for name, record in zip(('P', 'A', 'J', 'L'), (p, a_physical, j, l)):
            for k in PHYSICAL:
                need(k in record, name + '.' + k, 'physical_identity_missing')
                check(record.get(k) == pins.physical.get(k), 'physical_identity_mismatch')
        p_post = {k: p.get('post_' + k.removesuffix('_sha256')) for k in STATES}
        a_pre = {k: a.get('pre_' + k.removesuffix('_sha256')) for k in STATES}
        a_post = {k: a.get('post_' + k.removesuffix('_sha256')) for k in STATES}
        check(all(_state(s) for s in (p_post, a_pre, a_post, j.get('pre'),
                  l.get('pre'), l.get('post'), pins.pre, pins.post)), 'state_shape')
        check(p.get('pre_schema') == p.get('post_schema') and
              p.get('pre_datacl') == p.get('post_datacl') and
              _hex(p.get('pre_roles')) and p['pre_roles'] != p.get('post_roles'), 'p_delta_not_roles_only')
        check(_delta(p.get('pre_inventory'), p.get('post_inventory'), P_LOGIN, P_GROUP),
              'p_inventory_delta')
        for name in ('pre', 'post'):
            inventory = p.get(name + '_inventory')
            check(_inventory(inventory) and digest(json.dumps(inventory, sort_keys=True,
                  separators=(',', ':')).encode()) == p.get(name + '_role_inventory_sha256'),
                  'p_inventory_hash')
        check(_same(p.get('post_inventory'), pins.pre_inventory), 'p_l_inventory_link')
        check(_same(p_post, a_pre), 'p_a_preimage_mismatch')
        check(a_pre['roles_sha256'] == a_post['roles_sha256'] and
              a_pre['datacl_sha256'] == a_post['datacl_sha256'], 'a_roles_datacl_changed')
        check(a_pre['schema_sha256'] != a_post['schema_sha256'], 'a_schema_unchanged')
        check(_same(a_post, j.get('pre')) and _same(a_post, pins.pre) and
              _same(j.get('pre'), l.get('pre')), 'a_l_preimage_mismatch')
        check(_same(l.get('post'), pins.post), 'l_postimage_pin')
        check(pins.pre['schema_sha256'] == pins.post['schema_sha256'] and
              pins.pre['datacl_sha256'] == pins.post['datacl_sha256'] and
              pins.pre['roles_sha256'] != pins.post['roles_sha256'], 'l_delta_not_roles_only')
        check(_delta(j.get('pre_inventory'), l.get('post_inventory')), 'l_inventory_delta')
        check(_same(j.get('pre_inventory'), pins.pre_inventory) and
              _same(l.get('post_inventory'), pins.post_inventory) and
              _same(l.get('pre_inventory'), pins.pre_inventory), 'external_inventory_pins')
        need(pins.nominal_roles is not None, 'external.nominal_roles', 'nominal_evidence_missing')
        check(_nominal(pins.nominal_roles, pins.post_inventory), 'nominal_attributes_mismatch')
        check(j.get('login') == LOGIN and j.get('grupo') == GROUP and
              l.get('login') == LOGIN and l.get('grupo') == GROUP, 'nominal_names_mismatch')
        check(str(UUID(j['operation_id'])) == j['operation_id'], 'operation_id_invalid')
        check(_time(l['committed_at']) >= _time(j['created_at']), 'operation_time_invalid')
    except (ValueError, TypeError, KeyError, AttributeError, RecursionError, OverflowError):
        check(False, 'receipt_or_pins_invalid')
    return Validation(tuple(errors), tuple(missing))
