import test from 'node:test';
import assert from 'node:assert/strict';
import { createCorreoClient, redactOutgoing } from './configuracion-correo.js';

test('cliente GET usa contrato y PUT omite secreto vacío', async () => {
  const calls = [];
  const fetchImpl = async (url, options) => { calls.push({ url, options }); return new Response(JSON.stringify({ configuration: {}, secretConfigured: false }), { status: 200, headers: { 'Content-Type': 'application/json' } }); };
  const client = createCorreoClient({ fetchImpl });
  await client.get(); await client.put({ host: 'smtp.test', secret: '' });
  assert.equal(calls[0].url, '/api/vec/contratacion-temporal/configuracion-correo');
  assert.equal(calls[0].options.method, 'GET');
  assert.deepEqual(JSON.parse(calls[1].options.body), { host: 'smtp.test' });
});

test('secreto sólo entra cuando se introduce y se redactan claves desconocidas', () => {
  assert.deepEqual(redactOutgoing({ host: 'x', secret: 'token', password: 'no' }), { host: 'x', secret: 'token' });
  assert.deepEqual(redactOutgoing({ secret: '' }), {});
});
