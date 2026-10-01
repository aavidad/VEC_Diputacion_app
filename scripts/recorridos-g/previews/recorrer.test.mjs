import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { comprobarFuente, crearServidor, ficheroPermitido } from './servidor.mjs';
import { ejecutar } from './recorrer.mjs';

const casos = JSON.parse(fs.readFileSync(new URL('casos.json', import.meta.url), 'utf8'));

test('lista positiva: solo estáticos públicos y tres JSON exactos', () => {
  assert.equal(ficheroPermitido(casos.paginas.informes), casos.paginas.informes.slice(1));
  assert.equal(ficheroPermitido(casos.datos[0]), casos.datos[0].slice(1));
  for (const ruta of ['/api/vec/dietas', '/.git/config', '/data/demo/dietas/otro.json',
    '/web/static/../../AGENTS.md', '/web/static/.env', '/web/static/app.js?x=1', '/web/static/cert.pem']) {
    assert.equal(ficheroPermitido(ruta), null, ruta);
  }
});

test('una preview ausente no da plan verde ni abre Chrome', async () => {
  const fuente = fs.mkdtempSync(path.join(os.tmpdir(), 'vec-preview-ausente-'));
  try {
    assert.equal(comprobarFuente(fuente, casos).length, 5);
    assert.equal(await ejecutar(['--source', fuente, '--modo', 'plan']), 2);
  } finally { fs.rmSync(fuente, { recursive: true, force: true }); }
});

test('servidor deniega API, escritura, datos extra y enlaces', async () => {
  const fuente = fs.mkdtempSync(path.join(os.tmpdir(), 'vec-preview-servidor-'));
  const pagina = path.join(fuente, casos.paginas.informes.slice(1));
  fs.mkdirSync(path.dirname(pagina), { recursive: true });
  fs.writeFileSync(pagina, '<!doctype html><title>Fixture</title>');
  const enlace = path.join(fuente, 'web/static/enlace.html');
  fs.symlinkSync(pagina, enlace);
  const servidor = crearServidor(fuente);
  try {
    const origen = await servidor.escuchar();
    assert.equal((await fetch(`${origen}${casos.paginas.informes}?lang=es`)).status, 200);
    assert.equal((await fetch(`${origen}/api/vec/dietas`)).status, 403);
    assert.equal((await fetch(`${origen}${casos.paginas.informes}`, { method: 'POST' })).status, 403);
    assert.equal((await fetch(`${origen}/data/demo/dietas/otro.json`)).status, 403);
    assert.equal((await fetch(`${origen}/web/static/enlace.html`)).status, 404);
    assert.equal(servidor.contadores().peticiones, 1);
  } finally { if (servidor.servidor.listening) await servidor.cerrar(); fs.rmSync(fuente, { recursive: true, force: true }); }
});

test('preflight no sigue un directorio de datos enlazado', () => {
  const fuente = fs.mkdtempSync(path.join(os.tmpdir(), 'vec-preview-enlace-'));
  const otro = fs.mkdtempSync(path.join(os.tmpdir(), 'vec-preview-otro-'));
  try {
    fs.symlinkSync(otro, path.join(fuente, 'data'));
    assert.ok(comprobarFuente(fuente, casos).includes(casos.datos[0]));
  } finally {
    fs.rmSync(fuente, { recursive: true, force: true });
    fs.rmSync(otro, { recursive: true, force: true });
  }
});
