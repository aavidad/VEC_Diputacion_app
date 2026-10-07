import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { crearTextos } from '../../../comun/textos.js';
import { INDICE_IDIOMAS, localizacionDe } from '../../../comun/idioma.js';
import { crearClienteSeleccion } from './cliente.js';
import { aMicropuntos, validarConfiguracion, validarEjemplos, validarResultado, validarNotasPropuestas, validarNotasEjemplo } from './configuracion.js';
import { crearEstadoEnsayo } from './estado.js';
import { montarSeleccion } from './montaje.js';

const configuracion = () => ({ version: 1, modalidad: 'concurso_oposicion', turno_acceso: 'libre', destino: 'plaza', plazas: 1,
  fases: [{ referencia: 'prueba_1', tipo: 'prueba', minimo_micropuntos: 5000000, maximo_micropuntos: 10000000, peso: 60 },
    { referencia: 'meritos', tipo: 'meritos', minimo_micropuntos: 0, maximo_micropuntos: 4000000, peso: 40 }], desempates: ['prueba_1'] });
const ejemplo = () => ({ referencia: 'ejemplo_combinado', titulo_clave: 'seleccion.ejemplo.concurso_oposicion', convocatoria_ref: 'con_ejemplo', bases_version: 1, configuracion: configuracion() });
const resultado = (datos = ejemplo()) => ({ version: datos.configuracion.version, convocatoria_ref: datos.convocatoria_ref,
  bases_version: datos.bases_version, modalidad: datos.configuracion.modalidad, turno_acceso: datos.configuracion.turno_acceso,
  destino: datos.configuracion.destino, plazas: datos.configuracion.plazas, alcance: 'ensayo_sintetico', estado: 'provisional', causas: [],
  solicitudes: [{ referencia: 'sol_1', nombre: '<img src=x onerror=alert(1)>', acceso: 'cumple', acceso_detalle: [{ referencia: 'titulacion_acceso', estado: 'cumple' }], estado: 'apta', propuesta: 'propuesta_provisional', total_micropuntos: 5000000, orden: 1, causas: [],
    fases: datos.configuracion.fases.map(f => ({ ...f, puntos_micropuntos: 2000000, estado: 'superada', origen: f.tipo === 'meritos' ? 'motor_bolsa' : 'prueba_embebida', reglas: f.tipo === 'meritos' ? [{ referencia: 'curso', puntos_micropuntos: 1000000 }] : [] })) }] });

const catalogos = await Promise.all(INDICE_IDIOMAS.idiomas.map(async ({ codigo }) => ({ codigo,
  fuente: await readFile(new URL(`../../../textos/${codigo}/seleccion.json`, import.meta.url), 'utf8') })));
const respaldo = JSON.parse(catalogos.find(c => c.codigo === INDICE_IDIOMAS.por_defecto)?.fuente ?? catalogos[0].fuente);
const traductores = catalogos.map(c => crearTextos({ modulo: 'seleccion', idioma: c.codigo, localizacion: localizacionDe(c.codigo), respaldo, propio: JSON.parse(c.fuente) }));
function hojas(d, prefijo = '') { return Object.entries(d).flatMap(([k, v]) => typeof v === 'string' ? [`${prefijo}${k}`] : hojas(v, `${prefijo}${k}.`)).sort(); }

test('catálogos de todos los idiomas conservan claves, variables y traducciones completas', () => {
  for (const [i, textos] of traductores.entries()) {
    assert.deepEqual(textos.faltantes, []);
    const propio = JSON.parse(catalogos[i].fuente);
    assert.deepEqual(hojas(propio), hojas(respaldo));
    for (const clave of hojas(respaldo)) {
      const ruta = clave.split('.');
      const base = ruta.reduce((v, k) => v[k], respaldo), traduccion = ruta.reduce((v, k) => v[k], propio);
      assert.deepEqual([...traduccion.matchAll(/\{(\w+)\}/gu)].map(m => m[1]).sort(), [...base.matchAll(/\{(\w+)\}/gu)].map(m => m[1]).sort());
      assert.ok(textos.traducir(clave));
    }
    // Detecta las colisiones originales etiqueta/objeto antes de JSON.parse.
    const clavesRaiz = [...catalogos[i].fuente.matchAll(/^  "([^"]+)":/gmu)].map(m => m[1]);
    assert.equal(new Set(clavesRaiz).size, clavesRaiz.length);
  }
});

test('las reglas aceptan decimales exactos y rechazan mínimos, pesos y plazas inválidos', () => {
  assert.equal(aMicropuntos('5,000001'), 5000001);
  for (const dato of ['-1', '1e2', '0.1234567', '', '1,2,3']) assert.ok(Number.isNaN(aMicropuntos(dato)));
  const c = configuracion(); assert.deepEqual(validarConfiguracion(c), []);
  c.fases[0].minimo_micropuntos = 10000001; c.fases[1].peso = 41; c.plazas = 0;
  assert.deepEqual(validarConfiguracion(c).map(e => e.clave), ['validacion.plazas', 'validacion.minimo', 'validacion.suma_pesos']);
  assert.throws(() => validarEjemplos({ ejemplos: [ejemplo(), ejemplo()] }));
});

test('una respuesta de otras bases, reglas o alcance nunca se muestra como resultado', () => {
  const e = ejemplo(); assert.equal(validarResultado(resultado(e), e).alcance, 'ensayo_sintetico');
  for (const cambio of [{ alcance: 'oficial' }, { bases_version: 2 }, { plazas: 3 }, { convocatoria_ref: 'otra' }]) {
    assert.throws(() => validarResultado({ ...resultado(e), ...cambio }, e));
  }
  const distinto = resultado(e); distinto.solicitudes[0].fases[0].peso = 99;
  assert.throws(() => validarResultado(distinto, e));
  const empate = resultado(e); empate.solicitudes[0].estado = 'empate_pendiente';
  assert.throws(() => validarResultado(empate, e));
});

test('el cliente usa rutas cerradas, sin credenciales y solo envía referencia de ejemplo y reglas', async () => {
  const llamadas = []; const cliente = crearClienteSeleccion({ fetchImpl: async (ruta, opciones) => { llamadas.push({ ruta, opciones }); return new Response('{}', { headers: { 'Content-Type': 'application/json' } }); } });
  await cliente.listar(); await cliente.simular({ ejemplo_ref: 'ejemplo', configuracion: configuracion(), actor: 'no_enviar' });
  assert.deepEqual(llamadas.map(c => c.ruta), ['/api/seleccion/v1/ensayos', '/api/seleccion/v1/simulaciones']);
  assert.equal(llamadas[0].opciones.method, 'GET'); assert.equal(llamadas[1].opciones.method, 'POST');
  assert.deepEqual(Object.keys(JSON.parse(llamadas[1].opciones.body)).sort(), ['configuracion', 'ejemplo_ref']);
  for (const { opciones } of llamadas) { assert.equal(opciones.credentials, 'omit'); assert.equal(opciones.redirect, 'error'); assert.equal(opciones.cache, 'no-store'); }
});

test('el cliente clasifica denegación/validación/error y limita el cuerpo recibido', async () => {
  for (const [status, codigo] of [[403, 'denegado'], [422, 'validacion'], [503, 'error']]) {
    const c = crearClienteSeleccion({ fetchImpl: async () => new Response('{}', { status }) });
    await assert.rejects(c.listar(), e => e.codigo === codigo);
  }
  const c = crearClienteSeleccion({ fetchImpl: async () => new Response('x'.repeat(1024 * 1024 + 1)) });
  await assert.rejects(c.listar());
});

test('editar reglas o desmontar aborta la petición y descarta respuestas tardías', async () => {
  let resolver, signal; const cambios = [];
  const estado = crearEstadoEnsayo({ cliente: { simular: (_, opciones) => { signal = opciones.signal; return new Promise(r => { resolver = r; }); } }, publicar: v => cambios.push(v), validarResultado });
  const pendiente = estado.simular(ejemplo()); estado.invalidar();
  assert.ok(signal.aborted); resolver(resultado()); await pendiente;
  assert.equal(cambios.at(-1).estado, 'inicial'); assert.equal(cambios.at(-1).resultado, null);
  const segundo = estado.simular(ejemplo()); estado.cerrar(); resolver(resultado()); await segundo;
  assert.ok(signal.aborted); assert.ok(!cambios.some(c => c.estado === 'resultado'));
});

// DOM mínimo para verificar controles y foco con el traductor real.
class Elemento {
  constructor(d, tag) { this.ownerDocument = d; this.tagName = tag.toUpperCase(); this.children = []; this.attrs = {}; this.listeners = {}; this._text = ''; this.scrollTop = 0; this.scrollLeft = 0; this.clientHeight = tag === 'div' ? 60 : 480; this.clientTop = 1; this.scrollHeight = tag === 'section' ? 1100 : this.clientHeight; }
  set textContent(v) { this._text = String(v); this.children = []; }
  get textContent() { return this._text + this.children.map(c => c.textContent).join(' '); }
  append(...items) { items.forEach(e => { e.parentNode = this; }); this.children.push(...items); }
  replaceChildren(...items) { this.children = []; this.append(...items); this._text = ''; }
  setAttribute(k, v) { this.attrs[k] = String(v); }
  getAttribute(k) { return this.attrs[k]; }
  addEventListener(k, f) { this.listeners[k] = f; }
  closest(selector) { return selector === '.panel' ? (this.className?.split(' ').includes('panel') ? this : this.parentNode?.closest(selector)) : null; }
  getBoundingClientRect() {
    if (this.className === 'panel') return { top: 400, bottom: 880 };
    const top = 1100 + (this.ownerDocument.desplazamientoControl ?? 0) - (this.closest('.panel')?.scrollTop ?? 0);
    return { top, bottom: top + 24 };
  }
  focus() { this.ownerDocument.activeElement = this; }
  setSelectionRange(a, b) { this.selectionStart = a; this.selectionEnd = b; }
  get firstChild() { return this.children[0]; }
  async emitir(evento) { await this.listeners[evento]?.({ preventDefault() {} }); }
}
function dom() {
  const d = { createElement: t => new Elemento(d, t), getElementById: id => elementos(d.raiz).find(e => e.id === id), activeElement: null };
  d.raiz = d.createElement('div'); return d;
}
function elementos(e) { return [e, ...e.children.flatMap(elementos)]; }
const terminarCarga = () => new Promise(r => setImmediate(r));

test('montaje traducido: etiquetas, errores enlazados, resultado escapado y detalle del baremador', async () => {
  for (const textos of traductores) {
    const d = dom(); const llamadas = [];
    const montaje = montarSeleccion({ raiz: d.raiz, textos, cliente: { listar: async () => ({ ejemplos: [ejemplo()] }), simular: async datos => { llamadas.push(datos); return resultado(datos); } } });
    await terminarCarga();
    const controles = elementos(d.raiz).filter(e => ['INPUT', 'SELECT'].includes(e.tagName));
    for (const c of controles) assert.ok(elementos(d.raiz).some(e => e.tagName === 'LABEL' && (e.htmlFor === c.id || e.children.includes(c))));
    const plazas = d.getElementById('seleccion-plazas'); plazas.value = '0'; plazas.focus(); await plazas.emitir('input');
    assert.equal(d.getElementById('seleccion-plazas').getAttribute('aria-invalid'), 'true');
    await elementos(d.raiz).find(e => e.tagName === 'FORM').emitir('submit');
    assert.equal(d.activeElement.id, 'seleccion-errores'); assert.equal(llamadas.length, 0);
    const corregido = d.getElementById('seleccion-plazas'); corregido.value = '1'; await corregido.emitir('input');
    await elementos(d.raiz).find(e => e.tagName === 'FORM').emitir('submit');
    assert.equal(montaje.obtenerEstado().situacion, 'resultado');
    assert.equal(llamadas[0].ejemplo_ref, ejemplo().referencia);
    assert.ok(d.raiz.textContent.includes(textos.traducir('origen.motor_bolsa')));
    assert.ok(d.raiz.textContent.includes('<img src=x onerror=alert(1)>'));
    assert.ok(!elementos(d.raiz).some(e => e.tagName === 'IMG'));
    assert.ok(elementos(d.raiz).filter(e => e.tagName === 'TH').every(e => ['row', 'col'].includes(e.scope)));
    assert.ok(elementos(d.raiz).find(e => e.tagName === 'SUMMARY').getAttribute('aria-label'));
    const peso = d.getElementById('seleccion-peso-0'); peso.value = '59'; await peso.emitir('input');
    assert.equal(montaje.obtenerEstado().resultado, null);
    montaje.desmontar(); assert.equal(d.raiz.children.length, 0);
  }
});

test('carga vacía, denegación, error recuperable y cálculo pendiente conservan estados claros', async () => {
  const textos = traductores[0];
  for (const [respuesta, esperado] of [[{ ejemplos: [] }, 'vacio'], [Object.assign(new Error(), { codigo: 'denegado' }), 'denegado'], [new Error(), 'error']]) {
    const d = dom(); const m = montarSeleccion({ raiz: d.raiz, textos, cliente: { listar: async () => { if (respuesta instanceof Error) throw respuesta; return respuesta; }, simular: async () => {} } });
    await terminarCarga(); assert.equal(m.obtenerEstado().situacion, esperado);
    if (esperado === 'error') assert.ok(elementos(d.raiz).some(e => e.tagName === 'BUTTON' && e.textContent === textos.traducir('reintentar')));
    m.desmontar();
  }
  let resolver; const d = dom();
  const m = montarSeleccion({ raiz: d.raiz, textos, cliente: { listar: async () => ({ ejemplos: [ejemplo()] }), simular: datos => new Promise(r => { resolver = () => r(resultado(datos)); }) } });
  await terminarCarga(); const pendiente = elementos(d.raiz).find(e => e.tagName === 'FORM').emitir('submit');
  assert.ok(elementos(d.raiz).find(e => e.type === 'submit').disabled);
  assert.ok(elementos(d.raiz).some(e => e.getAttribute('aria-busy') === 'true'));
  resolver(); await pendiente; m.desmontar();
});


test('editar una fase inferior conserva scroll de ambos paneles, foco visible y cursor', async () => {
  const d = dom();
  const montaje = montarSeleccion({ raiz: d.raiz, textos: traductores[0], cliente: { listar: async () => ({ ejemplos: [ejemplo()] }), simular: async datos => resultado(datos) } });
  await terminarCarga();
  const reglas = d.getElementById('seleccion-panel-configuracion'); reglas.scrollTop = 350; reglas.scrollLeft = 12;
  const resultados = d.getElementById('seleccion-panel-resultado'); resultados.scrollTop = 120;
  const casilla = d.getElementById('seleccion-desempate-1'); casilla.focus(); casilla.checked = true; await casilla.emitir('change');
  assert.equal(d.activeElement.id, 'seleccion-desempate-1');
  assert.equal(d.getElementById('seleccion-panel-configuracion').scrollTop, 350);
  assert.equal(d.getElementById('seleccion-panel-configuracion').scrollLeft, 12);
  assert.equal(d.getElementById('seleccion-panel-resultado').scrollTop, 120);
  const minimo = d.getElementById('seleccion-minimo-1'); minimo.focus(); minimo.value = '0,5'; minimo.setSelectionRange(3, 3);
  // El contenido insertado desplaza el campo: se ajusta solo su marco.
  d.desplazamientoControl = 300; await minimo.emitir('input');
  assert.equal(d.activeElement.id, 'seleccion-minimo-1');
  assert.equal(d.activeElement.selectionStart, 3); assert.equal(d.activeElement.selectionEnd, 3);
  const marco = d.getElementById('seleccion-panel-configuracion'); const posicion = d.activeElement.getBoundingClientRect();
  assert.ok(marco.scrollTop > 350); assert.ok(posicion.top >= 465 && posicion.bottom <= 877);
  assert.equal(d.getElementById('seleccion-panel-resultado').scrollTop, 120);
  assert.equal(montaje.obtenerEstado().configuracion.fases[1].minimo_micropuntos, 500000);
  montaje.desmontar();
});

function ejemploConNotas() {
  const e = ejemplo();
  e.notas_prueba = [{ solicitud_ref: 'solicitud_ana', nombre: 'Ana Molina Ruiz', fase_ref: e.configuracion.fases[0].referencia, puntos_micropuntos: 7000000 }];
  return e;
}

test('notas del ejemplo: vacío pendiente, cero válido, exactitud y referencias cerradas', () => {
  const e = ejemploConNotas(); const n = e.notas_prueba[0];
  assert.equal(validarNotasEjemplo(e).length, 1);
  for (const puntos_micropuntos of [null, 0, 5000001, 10000000]) {
    assert.deepEqual(validarNotasPropuestas(e, e.configuracion, [{ ...n, puntos_micropuntos }]), []);
  }
  for (const puntos_micropuntos of [-1, 10000001, NaN, 0.5, undefined]) {
    assert.equal(validarNotasPropuestas(e, e.configuracion, [{ ...n, puntos_micropuntos }])[0].clave, 'validacion.nota');
  }
  for (const cambio of [{ solicitud_ref: 'otra' }, { fase_ref: e.configuracion.fases[1].referencia }]) {
    assert.equal(validarNotasPropuestas(e, e.configuracion, [{ ...n, ...cambio }]).length, 1);
  }
  assert.equal(validarNotasPropuestas(e, e.configuracion, [n, n]).length, 1);
  assert.throws(() => validarNotasEjemplo({ ...e, notas_prueba: [n, n] }));
});

test('el transporte envía sólo referencias de notas y valor, nunca nombres o hechos', async () => {
  let enviado;
  const c = crearClienteSeleccion({ fetchImpl: async (_, opciones) => { enviado = JSON.parse(opciones.body); return new Response('{}'); } });
  await c.simular({ ...ejemploConNotas(), ejemplo_ref: 'ensayo', actor: 'no_enviar' });
  assert.deepEqual(Object.keys(enviado).sort(), ['configuracion', 'ejemplo_ref', 'notas_prueba']);
  assert.deepEqual(Object.keys(enviado.notas_prueba[0]).sort(), ['fase_ref', 'puntos_micropuntos', 'solicitud_ref']);
  assert.equal(enviado.notas_prueba[0].puntos_micropuntos, 7000000);
});

test('editar notas invalida resultado, conserva texto inválido/foco y restablece el ejemplo', async () => {
  for (const textos of traductores) {
    const e = ejemploConNotas(); const d = dom(); const llamadas = [];
    const m = montarSeleccion({ raiz: d.raiz, textos, cliente: { listar: async () => ({ ejemplos: [e] }), simular: async datos => { llamadas.push(datos); return resultado(datos); } } });
    await terminarCarga();
    const nota = d.getElementById('seleccion-nota-0');
    assert.equal(nota.value, '7'); nota.focus(); nota.value = '5,000001'; nota.setSelectionRange(8, 8); await nota.emitir('input');
    assert.equal(d.activeElement.id, nota.id); assert.equal(d.activeElement.selectionStart, 8);
    await elementos(d.raiz).find(el => el.tagName === 'FORM').emitir('submit');
    assert.equal(llamadas[0].notas_prueba[0].puntos_micropuntos, 5000001);
    const invalida = d.getElementById(nota.id); invalida.value = '11'; await invalida.emitir('input');
    assert.equal(m.obtenerEstado().resultado, null);
    assert.equal(d.getElementById(nota.id).value, '11'); assert.equal(d.getElementById(nota.id).getAttribute('aria-invalid'), 'true');
    await elementos(d.raiz).find(el => el.tagName === 'FORM').emitir('submit');
    assert.equal(llamadas.length, 1); assert.equal(d.activeElement.id, 'seleccion-errores');
    const vacia = d.getElementById(nota.id); vacia.value = ''; await vacia.emitir('input');
    await elementos(d.raiz).find(el => el.tagName === 'FORM').emitir('submit');
    assert.equal(llamadas[1].notas_prueba[0].puntos_micropuntos, null);
    await elementos(d.raiz).find(el => el.tagName === 'BUTTON' && el.textContent === textos.traducir('restablecer')).emitir('click');
    assert.equal(d.getElementById(nota.id).value, '7');
    await elementos(d.raiz).find(el => el.tagName === 'FORM').emitir('submit');
    assert.deepEqual(llamadas[2].notas_prueba, []); m.desmontar();
  }
});


test('editar una nota en móvil conserva el scroll y el foco debajo de la cabecera', async () => {
  const d = dom(); d.scrollingElement = { scrollTop: 900, scrollLeft: 0 }; d.defaultView = { innerHeight: 800 };
  d.querySelector = () => ({ getBoundingClientRect: () => ({ bottom: 105 }) });
  const e = ejemploConNotas();
  const m = montarSeleccion({ raiz: d.raiz, textos: traductores[0], cliente: { listar: async () => ({ ejemplos: [e] }), simular: async datos => resultado(datos) } });
  await terminarCarga();
  const rect = () => ({ top: 1350 - d.scrollingElement.scrollTop, bottom: 1390 - d.scrollingElement.scrollTop });
  const crear = d.createElement;
  d.createElement = tag => { const el = crear(tag); if (tag === 'section') el.scrollHeight = el.clientHeight; if (tag === 'input') el.getBoundingClientRect = rect; return el; };
  const campo = d.getElementById('seleccion-nota-0'); campo.focus(); campo.value = '6,1';
  await campo.emitir('input');
  assert.equal(d.activeElement.id, campo.id); assert.equal(d.scrollingElement.scrollTop, 900);
  d.scrollingElement.scrollTop = 1300; const arriba = d.getElementById(campo.id); arriba.value = '6,2';
  await arriba.emitir('input');
  assert.equal(d.scrollingElement.scrollTop, 1241); assert.ok(rect().top >= 109 && rect().bottom <= 796);
  m.desmontar();
});
