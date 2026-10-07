import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { chromeAccesible, verificarZoom } from './a11y.mjs';
import { idiomas } from './config.mjs';
const t = idiomas.disponibles[idiomas.respaldo].accesibilidad_pruebas;

test(t.zoom, () => {
  const native = { ancho_css: 720, dpr: 2, escala_visual: 1, zoom_css: '1' };
  assert.doesNotThrow(() => verificarZoom(native, 1440, 2));
  assert.doesNotThrow(() => verificarZoom({ ...native, ancho_css: 390, dpr: 1 }, 390, 1));
  for (const m of [{ ...native, ancho_css: 1440 }, { ...native, dpr: 1 }, { ...native, escala_visual: 2 }, { ...native, zoom_css: '2' }, { zoom_css: '1' }]) assert.throws(() => verificarZoom(m, 1440, 2), /^Error: zoom_nativo$/);
});

test(t.perfil, async () => {
  const dir = fs.mkdtempSync(path.join(process.env.VEC_F_TEST_SCRATCH || os.tmpdir(), 'a11y-cleanup-'));
  try {
    for (const fase of ['launch', 'ajustes']) {
      let cerrado = false;
      const chromium = { launchPersistentContext: async perfil => {
        assert.equal(fs.statSync(perfil).mode & 0o777, 0o700);
        if (fase === 'launch') throw new Error('fixture');
        return { route: async () => {}, routeWebSocket: async () => {}, pages: () => [{ goto: async () => { throw new Error('fixture'); } }], close: async () => { cerrado = true; } };
      } };
      await assert.rejects(chromeAccesible(chromium, dir));
      assert.deepEqual(fs.readdirSync(dir), []);
      assert.equal(cerrado, fase === 'ajustes');
    }
  } finally { fs.rmSync(dir, { recursive: true, force: true }); }
});
