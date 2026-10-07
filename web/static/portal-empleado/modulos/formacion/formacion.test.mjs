import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { crearClienteEscenario } from './cliente.js';
import { validarEscenario, filtrarEdiciones, enlaceOficial } from './modelo.js';
import { pintarFormacion } from './vista.js';
import { crearTextos, esCatalogoValido } from '../../../comun/textos.js';

const es = JSON.parse(await readFile(new URL('../../../textos/es/formacion.json', import.meta.url)));
const en = JSON.parse(await readFile(new URL('../../../textos/en/formacion.json', import.meta.url)));
const errorES = JSON.parse(await readFile(new URL('../../../textos/es/formacion-error.json', import.meta.url)));
const errorEN = JSON.parse(await readFile(new URL('../../../textos/en/formacion-error.json', import.meta.url)));

test('el respaldo de idioma y el error total conservan avisos y reintento en ambos catálogos', () => {
  for (const catalogo of [es, en]) {
    for (const clave of ['idioma_respaldo', 'idioma_error', 'reintentar_idioma']) {
      assert.equal(typeof catalogo[clave], 'string');
      assert.ok(catalogo[clave].trim());
    }
  }
  for (const catalogo of [errorES, errorEN]) {
    assert.equal(esCatalogoValido(catalogo), true);
    for (const clave of ['titulo', 'mensaje', 'reintentar']) assert.ok(catalogo[clave].trim());
  }
});
const fixture = () => ({ alcance: 'preparacion_sintetica', version: 1, fuente: { referencia: 'FOR-001', version: 'v1', escenario: 'sintetico' }, plan: { referencia: 'plan:1', titulo_clave: 'formacion.plan.2027', desde: '2027-01-01', hasta: '2027-12-31', presupuesto_centimos: null, plazas: null, configuracion: { version: 'v1', modalidades: ['formacion.modalidad.presencial', 'formacion.modalidad.mixta'], prioridades: ['formacion.prioridad.alta'] } }, ediciones: ['presencial', 'mixta'].map((m, i) => ({ referencia: `edicion:${i}`, accion_referencia: 'accion:1', titulo_clave: 'formacion.accion.expedientes', necesidad_clave: 'formacion.necesidad.expedientes', modalidad_clave: `formacion.modalidad.${m}`, prioridad_clave: 'formacion.prioridad.alta', desde: '', hasta: '', plazas: null, horas: null, presupuesto_centimos: null, pendientes: ['formacion.pendiente.inscripciones'] })), pendientes: ['formacion.pendiente.fuente_maestra'], checklist: [{ clave: 'formacion.checklist.for001', referencia: 'FOR-001', estado: 'pendiente' }] });
function dom() {
  class N {
    constructor(tag) { this.tagName = tag; this.ownerDocument = d; this.nodeType = 1; this.children = []; this.dataset = {}; this.attrs = {}; this.listeners = {}; }
    append(...nodes) { this.children.push(...nodes); }
    replaceChildren(...nodes) { this.children = nodes; }
    setAttribute(k, v) { this.attrs[k] = v; }
    addEventListener(k, f) { this.listeners[k] = f; }
  }
  const d = { createElement: tag => new N(tag) }; return new N('main');
}
const nodes = n => [n, ...n.children.flatMap(nodes)];
test('scenario scope is preparation; missing quantities stay null and filters preserve source', () => {
  const dto = validarEscenario(fixture()); const before = JSON.stringify(dto);
  assert.equal(filtrarEdiciones(dto, 'formacion.modalidad.mixta').length, 1);
  assert.equal(filtrarEdiciones(dto, 'no-existe').length, 0); assert.equal(JSON.stringify(dto), before);
  assert.equal(dto.plan.plazas, null);
  for (const cambio of [{ alcance: 'oficial' }, { checklist: [{ clave: 'formacion.checklist.for001', estado: 'acreditado', referencia: 'FOR-001' }] }, { version: 2 }]) assert.throws(() => validarEscenario({ ...dto, ...cambio }));
  const duplicate = fixture(); duplicate.ediciones.push(duplicate.ediciones[0]); assert.throws(() => validarEscenario(duplicate));
});
test('transport omits credentials and exports the unmodified original bytes', async () => {
  const raw = new TextEncoder().encode(`  ${JSON.stringify(fixture(), null, 2)}\n`); let called;
  const cliente = crearClienteEscenario({ fetchImpl: async (ruta, opciones) => { called = { ruta, opciones }; return { ok: true, arrayBuffer: async () => raw.buffer }; } });
  const resultado = await cliente.cargar(); assert.deepEqual(resultado.bytes, raw);
  assert.equal(called.ruta, './escenario.json?v=20261001-formacion-preparacion-v1');
  for (const [k, v] of Object.entries({ credentials: 'omit', cache: 'no-store', redirect: 'error', referrerPolicy: 'no-referrer' })) assert.equal(called.opciones[k], v);
  await assert.rejects(crearClienteEscenario({ fetchImpl: async () => ({ ok: false }) }).cargar());
});
test('official links reject arbitrary schemes, hosts and credentials', () => {
  const allowed = Object.values(es.fuentes_permitidas); assert.equal(enlaceOficial(allowed[0], allowed), allowed[0]);
  for (const value of ['javascript:alert(1)', 'https://evil.example/', 'https://user:pass@formacion.dipgra.es/', '//formacion.dipgra.es/', 'https://formacion.dipgra.es/otra']) assert.equal(enlaceOficial(value, allowed), null);
});
test('ES/EN render preserves pending states, escaping and blocked registration/certification', () => {
  for (const [idioma, propio, localizacion] of [['es', es, 'es-ES'], ['en', en, 'en-GB']]) {
    const mensajes = structuredClone(propio); mensajes.formacion.accion.expedientes = '<img src=x onerror=alert(1)>';
    const textos = crearTextos({ modulo: 'formacion', idioma, localizacion, respaldo: es, propio: mensajes }); assert.deepEqual(textos.faltantes, []);
    const raiz = dom(); let ref;
    pintarFormacion({ raiz, escenario: fixture(), textos, seleccion: 'edicion:0', acciones: { elegir: r => { ref = r; }, filtrar() {}, exportar() {}, cerrar() {} } });
    assert.equal(nodes(raiz).some(n => n.tagName === 'img'), false);
    assert.equal(nodes(raiz).some(n => n.textContent === '<img src=x onerror=alert(1)>'), true);
    assert.equal(nodes(raiz).some(n => n.textContent === textos.traducir('sin_dato')), true);
    for (const action of ['presentar', 'certificado']) assert.equal(nodes(raiz).find(n => n.dataset.foco === action).disabled, true);
    nodes(raiz).find(n => n.dataset.foco === 'edicion-edicion:0').listeners.click(); assert.equal(ref, 'edicion:0');
  }
});
test('incomplete CLI projection and unknown configured keys remain usable and pending', () => {
  const incompleto = fixture(); incompleto.plan.configuracion = { version: '', modalidades: null, prioridades: [] };
  for (const e of incompleto.ediciones) {
    e.accion_referencia = ''; e.necesidad_clave = ''; e.modalidad_clave = ''; e.prioridad_clave = '';
    e.pendientes = ['formacion.pendiente.accion', 'formacion.pendiente.necesidad', 'formacion.pendiente.modalidad', 'formacion.pendiente.prioridad'];
  }
  const original = JSON.stringify(incompleto); const escenario = validarEscenario(incompleto);
  assert.deepEqual(escenario.plan.configuracion.modalidades, []); assert.equal(JSON.stringify(incompleto), original);
  const desconocido = fixture(); desconocido.plan.titulo_clave = 'formacion.plan.futuro';
  desconocido.plan.configuracion.modalidades.push('formacion.modalidad.otra');
  desconocido.plan.configuracion.prioridades = ['formacion.prioridad.otra'];
  desconocido.ediciones[0].modalidad_clave = 'formacion.modalidad.otra';
  desconocido.ediciones[0].prioridad_clave = 'formacion.prioridad.otra';
  desconocido.ediciones[0].necesidad_clave = 'formacion.necesidad.otra';
  const textos = crearTextos({ modulo: 'formacion', idioma: 'es', localizacion: 'es-ES', respaldo: es });
  for (const dto of [escenario, validarEscenario(desconocido)]) {
    const raiz = dom(); let abierta;
    pintarFormacion({ raiz, escenario: dto, textos, seleccion: dto.ediciones[0].referencia,
      acciones: { filtrar() {}, elegir: ref => { abierta = ref; }, exportar() {}, cerrar() {} } });
    assert.equal(nodes(raiz).some(n => n.textContent === textos.traducir('sin_dato')), true);
    nodes(raiz).find(n => n.dataset.foco === `edicion-${dto.ediciones[0].referencia}`).listeners.click();
    assert.equal(abierta, dto.ediciones[0].referencia);
    assert.equal(nodes(raiz).find(n => n.dataset.foco === 'exportar').disabled, false);
    for (const action of ['presentar', 'certificado']) assert.equal(nodes(raiz).find(n => n.dataset.foco === action).disabled, true);
  }
});
test('64 editions and 1024 pending checks fit the core contract', () => {
  const dto = fixture(); dto.ediciones = Array.from({ length: 64 }, (_, i) => ({ ...dto.ediciones[0], referencia: `edicion:${i}` }));
  dto.checklist = Array.from({ length: 1024 }, () => ({ clave: 'formacion.pendiente.aprobacion_edicion', referencia: 'edicion:0', estado: 'pendiente' }));
  assert.equal(validarEscenario(dto).ediciones.length, 64);
  assert.throws(() => validarEscenario({ ...dto, ediciones: [...dto.ediciones, { ...dto.ediciones[0], referencia: 'edicion:65' }] }));
  assert.throws(() => validarEscenario({ ...dto, checklist: [...dto.checklist, dto.checklist[0]] }));
});
test('actual incomplete and unknown-key CLI outputs render, filter and export unchanged bytes', async () => {
  const textos = crearTextos({ modulo: 'formacion', idioma: 'es', localizacion: 'es-ES', respaldo: es });
  for (const fichero of ['incompleto-output.json', 'desconocido-output.json']) {
    const originales = await readFile(new URL(`./fixtures/${fichero}`, import.meta.url));
    const carga = await crearClienteEscenario({ fetchImpl: async () => ({ ok: true, arrayBuffer: async () => originales.buffer.slice(originales.byteOffset, originales.byteOffset + originales.byteLength) }) }).cargar();
    assert.deepEqual(Buffer.from(carga.bytes), originales);
    const raiz = dom(); const edicion = carga.escenario.ediciones[0]; let filtro;
    pintarFormacion({ raiz, escenario: carga.escenario, textos, seleccion: edicion.referencia,
      acciones: { filtrar: value => { filtro = value; }, elegir() {}, exportar() {}, cerrar() {} } });
    const select = nodes(raiz).find(n => n.dataset.foco === 'modalidad');
    select.value = edicion.modalidad_clave; select.listeners.change(); assert.equal(filtro, edicion.modalidad_clave);
    assert.equal(filtrarEdiciones(carga.escenario, filtro).length, 1);
    assert.equal(nodes(raiz).some(n => n.textContent === textos.traducir('sin_dato')), true);
    assert.equal(nodes(raiz).find(n => n.dataset.foco === 'exportar').disabled, false);
    assert.equal(nodes(raiz).find(n => n.dataset.foco === 'presentar').disabled, true);
  }
});
