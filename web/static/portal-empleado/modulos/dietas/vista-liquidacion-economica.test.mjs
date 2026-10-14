import assert from 'node:assert/strict';
import test from 'node:test';
import { montarVistaLiquidacionEconomica } from './vista-liquidacion-economica.js';
import { cargarTextos } from '../../../comun/textos.js';

const dato = (atributo) => atributo.slice(5).replace(/-([a-z])/g, (_m, letra) => letra.toUpperCase());
class Nodo {
  constructor(doc, tag = 'div') { this.ownerDocument = doc; this.tagName = tag; this.children = []; this.dataset = {}; this.attrs = {}; this.listeners = {}; this.parent = null; this.textContent = ''; this.value = ''; this.disabled = false; }
  append(...nodos) { this.children.push(...nodos); nodos.forEach((n) => { n.parent = this; }); }
  replaceChildren(...nodos) { this.children = []; this.append(...nodos); }
  remove() { this.parent?.removeChild(this); }
  removeChild(nodo) { this.children = this.children.filter((n) => n !== nodo); nodo.parent = null; }
  addEventListener(tipo, f) { this.listeners[tipo] = f; } removeEventListener(tipo) { delete this.listeners[tipo]; }
  setAttribute(k, v) { this.attrs[k] = String(v); }
  removeAttribute(k) { delete this.attrs[k]; }
  focus() { this.ownerDocument.activeElement = this; }
  matches(sel) { const m = sel.match(/^\[([^=\]]+)(?:="([^"]*)")?\]$/u); if (!m) return this.tagName === sel; const v = this.dataset[dato(m[1])]; return v !== undefined && (m[2] === undefined || v === m[2]); }
  closest(sel) { for (let a = this; a; a = a.parent) if (a.matches(sel)) return a; return null; }
  querySelectorAll(sel) { const out = []; const visitar = (n) => { if (n.matches(sel)) out.push(n); n.children.forEach(visitar); }; visitar(this); return out; }
  querySelector(sel) { return this.querySelectorAll(sel)[0] || null; }
}
const raiz = () => { const doc = { createElement: (tag) => new Nodo(doc, tag) }; return new Nodo(doc, 'root'); };
const ref = 'dco_1234567890123456789012';
const propuesta = { comision_ref: ref, comision_version: 3, catalogo: { referencia: 'cat', version: 'v1', publicacion_ref: 'publicada', fecha_publicacion: '2026-01-01', version_tarifa_ref: 'tarifa-v1', vigente_desde: '2026-01-01', vigente_hasta: '2026-12-31', ambito: 'España', grupo: 2,
  normas: [{ referencia: 'RD 462/2002', articulo: 'Artículo 1', url: 'https://example.invalid/norma' }], reglas: [{ referencia: 'regla', tipo: 'dieta', concepto: 'manutencion' }] },
  lineas: [{ indice: 0, regla_ref: 'regla', original_centimos: 1000, calculado_publicado_centimos: 1000, diferencia_centimos: 0, reconocido_propuesto_centimos: 1000, rechazado_centimos: 0, motivo_codigo: '' }],
  totales: { original_centimos: 1000, calculado_publicado_centimos: 1000, diferencia_centimos: 0, reconocido_propuesto_centimos: 1000, rechazado_centimos: 0 }, decision_pendiente: false };
const recibo = { constancia: { recibo_ref: 'recibo-real', fecha: '2026-10-14T10:00:00Z', version_entrada: 3, version_resultante: 4 }, economia: propuesta };
const tick = async () => { await new Promise((r) => setTimeout(r, 0)); };
const click = async (panel, accion) => panel.listeners.click({ target: panel.querySelector(`[data-dietas-liquidacion-accion="${accion}"]`) });
const texto = (n) => [n.textContent, ...n.children.map(texto)].join(' ');

test('revisa propuesta publicada, obliga motivo de reducción y muestra recibo real pendiente de firma', async () => {
  const contenedor = raiz(); const entradas = []; const textos = await cargarTextos('dietas', { idioma: 'es', porDefecto: 'es' });
  const cliente = { propuesta: async () => propuesta, registrar: async (_ref, entrada) => { entradas.push(entrada); return recibo; }, leer: async () => recibo };
  const vista = montarVistaLiquidacionEconomica(contenedor, { cliente, referencia: ref, version: 3,
    generarClaveIdempotencia: () => 'liquidacion-solicitud-0001', cargarTextosImpl: async () => textos });
  await tick(); const panel = contenedor.querySelector('[data-dietas-liquidacion]');
  assert.match(texto(panel), /publicada/u);
  assert.match(texto(panel), /RD 462\/2002/u);
  const decision = panel.querySelector('[data-dietas-liquidacion-decision]'); decision.value = 'ajustar';
  await panel.listeners.change({ target: decision });
  const campo = panel.querySelector('[data-dietas-liquidacion-indice]'); campo.value = '5.00';
  await click(panel, 'revisar');
  assert.equal(entradas.length, 0);
  assert.match(panel.querySelector('[data-dietas-liquidacion-estado]').textContent, /motivo/u);
  panel.querySelector('[data-dietas-liquidacion-motivo]').value = 'revision_documental';
  await click(panel, 'revisar'); await click(panel, 'confirmar');
  assert.equal(entradas.length, 1);
  assert.deepEqual(entradas[0], { version_esperada: 3, clave_idempotencia: 'liquidacion-solicitud-0001', revisiones: [{ indice: 0, regla_ref: 'regla', reconocido_propuesto_centimos: 500, motivo_codigo: 'revision_documental' }] });
  assert.match(texto(panel), /recibo-real/u); assert.match(texto(panel), /pendiente de firma/u); assert.doesNotMatch(texto(panel), /Pagada/u);
  vista.desmontar();
});

test('resultado incierto repite exactamente la misma solicitud y un doble clic no duplica POST', async () => {
  const contenedor = raiz(); const entradas = []; let resolver; let intento = 0; const textos = await cargarTextos('dietas', { idioma: 'es', porDefecto: 'es' });
  const cliente = { propuesta: async () => propuesta, leer: async () => recibo,
    registrar: async (_ref, entrada) => { entradas.push(entrada); if (!intento++) { const error = new Error(); error.resultadoIndeterminado = true; throw error; }
      return new Promise((resolve) => { resolver = resolve; }); } };
  const vista = montarVistaLiquidacionEconomica(contenedor, { cliente, referencia: ref, version: 3, generarClaveIdempotencia: () => 'liquidacion-solicitud-0001', cargarTextosImpl: async () => textos });
  await tick(); const panel = contenedor.querySelector('[data-dietas-liquidacion]');
  await click(panel, 'revisar'); await click(panel, 'confirmar');
  assert.match(panel.querySelector('[data-dietas-liquidacion-estado]').textContent, /misma solicitud/u);
  const reintento = click(panel, 'reintentar'); await tick();
  await panel.listeners.click({ target: panel.querySelector('[data-dietas-liquidacion-accion="reintentar"]') });
  assert.equal(entradas.length, 2); assert.deepEqual(entradas[0], entradas[1]);
  resolver(recibo); await reintento; vista.desmontar();
});

test('desmontar cancela consulta y descarta respuesta tardía', async () => {
  const contenedor = raiz(); let resolver; let signal;
  const vista = montarVistaLiquidacionEconomica(contenedor, { cliente: { propuesta: (_r, _v, o) => { signal = o.signal; return new Promise((resolve) => { resolver = resolve; }); }, registrar: async () => assert.fail('POST'), leer: async () => assert.fail('GET') }, referencia: ref, version: 3 });
  await tick(); vista.desmontar(); assert.equal(signal.aborted, true); resolver(propuesta); await tick(); assert.equal(contenedor.children.length, 0);
});

test('catálogos cargados mantienen sección e importes localizados', async () => {
  const [es, en] = await Promise.all([cargarTextos('dietas', { idioma: 'es', porDefecto: 'es' }), cargarTextos('dietas', { idioma: 'en', porDefecto: 'es' })]);
  assert.match(es.traducir('liquidacion_economica.pendiente_firma'), /pendiente/u);
  assert.match(en.traducir('liquidacion_economica.pendiente_firma'), /awaiting/u);
});

test('la propuesta cargada no deja una ayuda permanente en ninguno de los idiomas', async () => {
  for (const idioma of ['es', 'en']) {
    const textos = await cargarTextos('dietas', { idioma, porDefecto: 'es' }); const contenedor = raiz();
    const vista = montarVistaLiquidacionEconomica(contenedor, { referencia: ref, version: 3,
      cliente: { propuesta: async () => propuesta, registrar: async () => assert.fail('POST inesperado'), leer: async () => assert.fail('GET inesperado') },
      cargarTextosImpl: async () => textos });
    await tick(); const panel = contenedor.querySelector('[data-dietas-liquidacion]');
    assert.equal(panel.querySelector('[data-dietas-liquidacion-estado]').textContent, '');
    assert.ok(panel.querySelector('[data-dietas-liquidacion-decision]'));
    vista.desmontar();
  }
});

test('el teclado mantiene el foco en cada decisión y pasa al importe al ajustar', async () => {
  const textos = await cargarTextos('dietas', { idioma: 'es', porDefecto: 'es' }); const contenedor = raiz();
  const pendiente = { ...propuesta, decision_pendiente: true,
    lineas: [{ ...propuesta.lineas[0], calculado_publicado_centimos: 1200, diferencia_centimos: 200, reconocido_propuesto_centimos: 1200 }] };
  const vista = montarVistaLiquidacionEconomica(contenedor, { referencia: ref, version: 3,
    cliente: { propuesta: async () => pendiente, registrar: async () => assert.fail('POST inesperado'), leer: async () => assert.fail('GET inesperado') },
    cargarTextosImpl: async () => textos });
  await tick(); const panel = contenedor.querySelector('[data-dietas-liquidacion]');
  for (const opcion of ['aceptar', 'orientativo', 'ajustar']) {
    const anterior = panel.querySelector('[data-dietas-liquidacion-decision]'); anterior.focus(); anterior.value = opcion;
    await panel.listeners.change({ target: anterior });
    const esperado = panel.querySelector(opcion === 'ajustar' ? '[data-dietas-liquidacion-indice]' : '[data-dietas-liquidacion-decision]');
    assert.notEqual(esperado, anterior); assert.equal(contenedor.ownerDocument.activeElement, esperado, opcion);
  }
  vista.desmontar();
});

test('el error se anuncia junto a decisión, importe y motivo y desaparece al corregir', async () => {
  const textos = await cargarTextos('dietas', { idioma: 'es', porDefecto: 'es' }); const contenedor = raiz();
  const pendiente = { ...propuesta, decision_pendiente: true,
    lineas: [{ ...propuesta.lineas[0], calculado_publicado_centimos: 1200, diferencia_centimos: 200, reconocido_propuesto_centimos: 1200 }] };
  const vista = montarVistaLiquidacionEconomica(contenedor, { referencia: ref, version: 3,
    cliente: { propuesta: async () => pendiente, registrar: async () => assert.fail('POST inesperado'), leer: async () => assert.fail('GET inesperado') },
    cargarTextosImpl: async () => textos });
  await tick(); const panel = contenedor.querySelector('[data-dietas-liquidacion]');
  const comprobarError = (control) => {
    const aviso = panel.querySelector('[data-dietas-liquidacion-error-campo]');
    assert.ok(aviso); assert.equal(control.attrs['aria-invalid'], 'true'); assert.equal(control.attrs['aria-describedby'], aviso.id);
    assert.equal(control.closest('td').querySelector('[data-dietas-liquidacion-error-campo]'), aviso);
    assert.equal(contenedor.ownerDocument.activeElement, control);
  };
  await click(panel, 'revisar'); const decision = panel.querySelector('[data-dietas-liquidacion-decision]'); comprobarError(decision);
  decision.value = 'ajustar'; await panel.listeners.change({ target: decision });
  assert.equal(panel.querySelector('[data-dietas-liquidacion-error-campo]'), null);
  const importe = panel.querySelector('[data-dietas-liquidacion-indice]'); importe.value = '13';
  await click(panel, 'revisar'); comprobarError(importe);
  importe.value = '11'; panel.listeners.input({ target: importe });
  assert.equal(importe.attrs['aria-invalid'], undefined); assert.equal(importe.attrs['aria-describedby'], undefined);
  assert.equal(panel.querySelector('[data-dietas-liquidacion-error-campo]'), null);
  await click(panel, 'revisar'); const motivo = panel.querySelector('[data-dietas-liquidacion-motivo]'); comprobarError(motivo);
  motivo.value = 'revision_documental'; await panel.listeners.change({ target: motivo });
  assert.equal(motivo.attrs['aria-invalid'], undefined); assert.equal(motivo.attrs['aria-describedby'], undefined);
  assert.equal(panel.querySelector('[data-dietas-liquidacion-error-campo]'), null);
  vista.desmontar();
});

test('un 409 no atribuye recibo ajeno; reintenta exactamente el mismo POST y su clave', async () => {
  const textos = await cargarTextos('dietas', { idioma: 'es', porDefecto: 'es' }); const contenedor = raiz(); let lecturas = 0; const entradas = [];
  const cliente = { propuesta: async () => propuesta, registrar: async (_ref, entrada) => { entradas.push(entrada); if (entradas.length === 1) { const e = Error(); e.codigo = 'conflicto_version'; throw e; } return recibo; },
    leer: async () => { lecturas++; return recibo; } };
  const vista = montarVistaLiquidacionEconomica(contenedor, { cliente, referencia: ref, version: 3,
    generarClaveIdempotencia: () => 'liquidacion-solicitud-0001', cargarTextosImpl: async () => textos });
  await tick(); const panel = contenedor.querySelector('[data-dietas-liquidacion]');
  await click(panel, 'revisar'); await click(panel, 'confirmar');
  assert.equal(lecturas, 0); assert.doesNotMatch(texto(panel), /recibo-real/u);
  assert.match(panel.querySelector('[data-dietas-liquidacion-estado]').textContent, /misma operación/u);
  await click(panel, 'reintentar');
  assert.equal(lecturas, 0); assert.deepEqual(entradas[1], entradas[0]); assert.match(texto(panel), /recibo-real/u);
  vista.desmontar();
});

test('un 403 en la propuesta no ofrece registrar ni reintentar sin permiso', async () => {
  const textos = await cargarTextos('dietas', { idioma: 'es', porDefecto: 'es' }); const contenedor = raiz();
  const vista = montarVistaLiquidacionEconomica(contenedor, { cliente: { propuesta: async () => { const e = Error(); e.codigo = 'acceso_denegado'; throw e; },
    registrar: async () => assert.fail('POST denegado'), leer: async () => assert.fail('GET denegado') }, referencia: ref, version: 3, cargarTextosImpl: async () => textos });
  await tick(); const panel = contenedor.querySelector('[data-dietas-liquidacion]');
  assert.match(panel.querySelector('[data-dietas-liquidacion-estado]').textContent, /permiso/u);
  assert.equal(panel.querySelector('[data-dietas-liquidacion-accion]'), null); vista.desmontar();
});

test('solo el recibo durable ofrece PDF y Word; descarga la versión registrada sin simular firma', async () => {
  const textos = await cargarTextos('dietas', { idioma: 'es', porDefecto: 'es' }); const contenedor = raiz();
  const nombre = 'VEC-D-2026-000012'; const resultado = { ...recibo, constancia: { ...recibo.constancia, numero_documento: nombre } };
  const llamadas = []; const enlaces = []; const revocadas = [];
  const entornoDescarga = { Blob, URL: { createObjectURL() { return 'blob:sintetico'; }, revokeObjectURL(url) { revocadas.push(url); } },
    document: { body: { append(enlace) { enlaces.push(enlace); } }, createElement() { return { click() { this.pulsado = true; }, remove() { this.retirado = true; } }; } } };
  const vista = montarVistaLiquidacionEconomica(contenedor, { referencia: ref, version: 3,
    cliente: { propuesta: async () => propuesta, registrar: async () => resultado, leer: async () => resultado,
      descargar: async (...args) => { llamadas.push(args); return { nombre: `${nombre}.${args[3]}`, bytes: Uint8Array.from([1, 2, 3]), mime: 'application/pdf' }; } },
    generarClaveIdempotencia: () => 'liquidacion-solicitud-0001', cargarTextosImpl: async () => textos, entornoDescarga });
  await tick(); const panel = contenedor.querySelector('[data-dietas-liquidacion]');
  assert.equal(panel.querySelector('[data-dietas-liquidacion-accion="descargar_pdf"]'), null);
  await click(panel, 'revisar'); await click(panel, 'confirmar');
  assert.ok(panel.querySelector('[data-dietas-liquidacion-accion="descargar_pdf"]'));
  assert.ok(panel.querySelector('[data-dietas-liquidacion-accion="descargar_docx"]'));
  await click(panel, 'descargar_pdf');
  assert.equal(llamadas.length, 1); assert.deepEqual(llamadas[0].slice(0, 5), [ref, 4, nombre, 'pdf', 'es']);
  assert.equal(enlaces[0].download, `${nombre}.pdf`); assert.equal(enlaces[0].pulsado, true); assert.equal(enlaces[0].retirado, true);
  assert.match(texto(panel), /pendiente de firma/u);
  vista.desmontar(); assert.deepEqual(revocadas, ['blob:sintetico']);
});

test('descarga denegada conserva recibo y permite volver sin ofrecer un archivo falso', async () => {
  const textos = await cargarTextos('dietas', { idioma: 'es', porDefecto: 'es' }); const contenedor = raiz();
  const durable = { ...recibo, constancia: { ...recibo.constancia, numero_documento: 'VEC-D-2026-000012' } };
  let intentos = 0;
  const vista = montarVistaLiquidacionEconomica(contenedor, { referencia: ref, version: 3,
    cliente: { propuesta: async () => propuesta, registrar: async () => durable, leer: async () => durable,
      descargar: async () => { intentos++; const e = Error(); e.codigo = 'acceso_denegado'; e.estado = 403; throw e; } },
    generarClaveIdempotencia: () => 'liquidacion-solicitud-0001', cargarTextosImpl: async () => textos });
  await tick(); const panel = contenedor.querySelector('[data-dietas-liquidacion]');
  await click(panel, 'revisar'); await click(panel, 'confirmar'); await click(panel, 'descargar_pdf');
  assert.equal(intentos, 1); assert.match(panel.querySelector('[data-dietas-liquidacion-estado]').textContent, /permiso/u);
  assert.match(texto(panel), /recibo-real/u); assert.ok(panel.querySelector('[data-dietas-liquidacion-accion="descargar_pdf"]'));
  vista.desmontar();
});

test('desmontar aborta una descarga pendiente y descarta bytes tardíos', async () => {
  const textos = await cargarTextos('dietas', { idioma: 'es', porDefecto: 'es' }); const contenedor = raiz(); let resolver; let signal; let enlaces = 0;
  const durable = { ...recibo, constancia: { ...recibo.constancia, numero_documento: 'VEC-D-2026-000012' } };
  const vista = montarVistaLiquidacionEconomica(contenedor, { referencia: ref, version: 3,
    cliente: { propuesta: async () => propuesta, registrar: async () => durable, leer: async () => durable,
      descargar: async (_r, _v, _n, _f, _i, opciones) => { signal = opciones.signal; return new Promise((resolve) => { resolver = resolve; }); } },
    generarClaveIdempotencia: () => 'liquidacion-solicitud-0001', cargarTextosImpl: async () => textos,
    entornoDescarga: { Blob, URL: { createObjectURL() { enlaces++; return 'blob:no'; }, revokeObjectURL() {} },
      document: { body: { append() { enlaces++; } }, createElement() { return { click() { enlaces++; }, remove() {} }; } } } });
  await tick(); const panel = contenedor.querySelector('[data-dietas-liquidacion]');
  await click(panel, 'revisar'); await click(panel, 'confirmar');
  const pendiente = click(panel, 'descargar_pdf'); assert.equal(signal.aborted, false);
  vista.desmontar(); assert.equal(signal.aborted, true);
  resolver({ nombre: 'VEC-D-2026-000012.pdf', bytes: Uint8Array.from([1, 2, 3]), mime: 'application/pdf' });
  await pendiente; assert.equal(enlaces, 0); assert.equal(contenedor.children.length, 0);
});

test('diferencias publicadas pendientes exigen elegir por línea y permiten reconocer más que la orientación', async () => {
  const textos = await cargarTextos('dietas', { idioma: 'es', porDefecto: 'es' }); const contenedor = raiz(); const entradas = [];
  const pendiente = { ...propuesta, decision_pendiente: true,
    lineas: [
      { indice: 0, regla_ref: 'regla', original_centimos: 1000, calculado_publicado_centimos: 1200, diferencia_centimos: 200, reconocido_propuesto_centimos: 1200, rechazado_centimos: 0, motivo_codigo: '' },
      { indice: 1, regla_ref: 'regla', original_centimos: 1200, calculado_publicado_centimos: 1000, diferencia_centimos: -200, reconocido_propuesto_centimos: 1000, rechazado_centimos: 0, motivo_codigo: '' },
    ], totales: { original_centimos: 2200, calculado_publicado_centimos: 2200, diferencia_centimos: 0, reconocido_propuesto_centimos: 2200, rechazado_centimos: 0 } };
  const vista = montarVistaLiquidacionEconomica(contenedor, { referencia: ref, version: 3,
    cliente: { propuesta: async () => pendiente, leer: async () => assert.fail('GET innecesario'),
      registrar: async (_r, entrada) => { entradas.push(entrada); const economia = { ...pendiente, decision_pendiente: false,
        lineas: pendiente.lineas.map((l) => ({ ...l, motivo_codigo: 'diferencia_tarifa_publicada' })) };
        return { ...recibo, economia }; } },
    generarClaveIdempotencia: () => 'liquidacion-solicitud-0001', cargarTextosImpl: async () => textos });
  await tick(); const panel = contenedor.querySelector('[data-dietas-liquidacion]');
  assert.match(texto(panel), /12,00/u); assert.match(texto(panel), /10,00/u); assert.match(texto(panel), /\+2,00/u);
  await click(panel, 'revisar'); assert.equal(entradas.length, 0);
  assert.match(panel.querySelector('[data-dietas-liquidacion-estado]').textContent, /Elija/u);
  for (let i = 0; i < 2; i++) {
    const select = panel.querySelector(`[data-dietas-liquidacion-decision="${i}"]`); select.value = 'aceptar';
    await panel.listeners.change({ target: select });
  }
  await click(panel, 'revisar'); await click(panel, 'confirmar');
  assert.deepEqual(entradas[0].revisiones.map((r) => [r.reconocido_propuesto_centimos, r.motivo_codigo]),
    [[1200, 'diferencia_tarifa_publicada'], [1000, 'diferencia_tarifa_publicada']]);
  assert.match(texto(panel), /recibo-real/u); assert.match(texto(panel), /pendiente de firma/u);
  vista.desmontar();
});

test('puede mantener expresamente la orientación o ajustar entre ella y el cálculo publicado', async () => {
  const textos = await cargarTextos('dietas', { idioma: 'es', porDefecto: 'es' });
  const pendiente = { ...propuesta, decision_pendiente: true,
    lineas: [{ ...propuesta.lineas[0], calculado_publicado_centimos: 1200, diferencia_centimos: 200, reconocido_propuesto_centimos: 1200 }],
    totales: { original_centimos: 1000, calculado_publicado_centimos: 1200, diferencia_centimos: 200, reconocido_propuesto_centimos: 1200, rechazado_centimos: 0 } };
  for (const [opcion, importe, motivo] of [['orientativo', 1000, ''], ['ajustar', 1100, 'revision_documental']]) {
    const contenedor = raiz(); const entradas = [];
    const vista = montarVistaLiquidacionEconomica(contenedor, { referencia: ref, version: 3,
      cliente: { propuesta: async () => pendiente, leer: async () => assert.fail('GET innecesario'),
        registrar: async (_r, entrada) => { entradas.push(entrada); return recibo; } },
      generarClaveIdempotencia: () => 'liquidacion-solicitud-0001', cargarTextosImpl: async () => textos });
    await tick(); const panel = contenedor.querySelector('[data-dietas-liquidacion]');
    const select = panel.querySelector('[data-dietas-liquidacion-decision]'); select.value = opcion;
    await panel.listeners.change({ target: select });
    if (opcion === 'ajustar') {
      assert.equal(panel.querySelector('[data-dietas-liquidacion-indice]').max, '12');
      panel.querySelector('[data-dietas-liquidacion-indice]').value = '11.00';
      await click(panel, 'revisar'); assert.equal(entradas.length, 0);
      panel.querySelector('[data-dietas-liquidacion-motivo]').value = motivo;
    }
    await click(panel, 'revisar'); await click(panel, 'confirmar');
    assert.equal(entradas[0].revisiones[0].reconocido_propuesto_centimos, importe);
    assert.equal(entradas[0].revisiones[0].motivo_codigo, motivo);
    vista.desmontar();
  }
});

test('consulta de instantánea durable recupera 404, 403 y 503 con GET; no prepara ni escribe', async () => {
  const textos = await cargarTextos('dietas', { idioma: 'es', porDefecto: 'es' });
  for (const [codigo, patron] of [['no_encontrada', /No hay una liquidación/u], ['acceso_denegado', /permiso/u], ['no_disponible', /No se pudo consultar/u]]) {
    const contenedor = raiz(); let consultas = 0;
    const vista = montarVistaLiquidacionEconomica(contenedor, { referencia: ref, version: 'actual', modoConsulta: true,
      cliente: { leer: async (_r, version) => { assert.equal(version, 'actual'); consultas++; if (consultas === 1) throw Object.assign(Error(), { codigo });
        return { ...recibo, constancia: { ...recibo.constancia, numero_documento: 'VEC-D-2026-000012' } }; } },
      cargarTextosImpl: async () => textos });
    await tick(); const panel = contenedor.querySelector('[data-dietas-liquidacion]');
    assert.match(panel.querySelector('[data-dietas-liquidacion-estado]').textContent, patron);
    assert.equal(panel.querySelector('[data-dietas-liquidacion-accion="confirmar"]'), null);
    await click(panel, 'recargar');
    assert.equal(consultas, 2); assert.match(texto(panel), /VEC-D-2026-000012/u);
    vista.desmontar();
  }
});
