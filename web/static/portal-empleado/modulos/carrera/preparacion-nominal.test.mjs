import assert from 'node:assert/strict';
import test from 'node:test';
import { readFile } from 'node:fs/promises';
import { crearTextos } from '../../../comun/textos.js';
import { montarPreparacionNominal } from './preparacion-nominal.js';

const es = JSON.parse(await readFile(new URL('../../../textos/es/carrera-preparacion-nominal.json', import.meta.url), 'utf8'));
const en = JSON.parse(await readFile(new URL('../../../textos/en/carrera-preparacion-nominal.json', import.meta.url), 'utf8'));
const textos = crearTextos({ modulo: 'carrera-preparacion-nominal', idioma: 'es', localizacion: 'es-ES', respaldo: es });

// Doble DOM exclusivo de pruebas: ejercita el montaje y sus listeners sin ruta ni red.
class Elemento {
  constructor(etiqueta, documento) {
    this.tagName = etiqueta.toUpperCase(); this.ownerDocument = documento;
    this.children = []; this.attributes = new Map(); this.listeners = new Map(); this.dataset = {};
    this.parentNode = null; this._texto = '';
  }
  set textContent(valor) { this.replaceChildren(); this._texto = String(valor); }
  get textContent() { return this._texto + this.children.map(n => n.textContent).join(''); }
  append(...nodos) { nodos.forEach(n => { n.remove(); n.parentNode = this; this.children.push(n); }); }
  replaceChildren(...nodos) { this.children.forEach(n => { n.parentNode = null; }); this.children = []; this._texto = ''; this.append(...nodos); }
  remove() { if (this.parentNode) { const padre = this.parentNode; padre.children = padre.children.filter(n => n !== this); this.parentNode = null; } }
  setAttribute(clave, valor) { this.attributes.set(clave, String(valor)); }
  getAttribute(clave) { return this.attributes.get(clave); }
  addEventListener(clave, listener) { this.listeners.set(clave, listener); }
  removeEventListener(clave, listener) { if (this.listeners.get(clave) === listener) this.listeners.delete(clave); }
  click() { this.listeners.get('click')?.(); }
}
function raizDOM() { const documento = { createElement: etiqueta => new Elemento(etiqueta, documento) }; return documento.createElement('div'); }
function todos(raiz, etiqueta) { return raiz.children.flatMap(n => [...(n.tagName === etiqueta.toUpperCase() ? [n] : []), ...todos(n, etiqueta)]); }
function pendiente() { let resolve, reject; const promise = new Promise((r, e) => { resolve = r; reject = e; }); return { promise, resolve, reject }; }
const procedencia = () => ({ ActoRef: 'acto:1', FuenteRef: 'personal:1', FuenteVersion: 'fuente:v2', Certeza: 'acreditado' });
function respuesta() {
  return { estado: 'pendiente', pendientes: ['grupo', 'grado', 'politica'], antecedentes: {
    EmpleadoRef: 'empleado:prueba', OrganismoRef: 'organismo:prueba', Version: 3,
    Corte: { vigente_en: '2026-10-02', conocido_en: '2026-10-02T10:00:00.000000Z' }, Cobertura: 'parcial',
    Relaciones: [{ RelacionRef: 'relacion:prueba', Version: 2, Periodo: { Desde: '2020-01-01', Hasta: '' }, Estado: 'vigente', RegimenRef: 'regimen:fuente', RegimenVersion: 4, Procedencia: procedencia() }],
    Servicios: [{ ServicioRef: 'servicio:prueba', RelacionRef: 'relacion:prueba', Version: 2, Periodo: { Desde: '2021-01-01', Hasta: '2022-01-01' }, Estado: 'reconocido', ClaseRef: 'clase:fuente', ClaseVersion: 1, Procedencia: procedencia() }],
    Puestos: [{ PuestoRef: 'puesto:prueba', PuestoVersion: 'rpt:2', RelacionRef: 'relacion:prueba', Periodo: { Desde: '2021-01-01', Hasta: '' }, Nivel: 22, Procedencia: procedencia() }], Situaciones: null,
    Evidencia: { recibo_ref: 'recibo:original', decision_ref: 'decision:original', efecto_ref: 'efecto:original', consumo_huella_sha256: 'a'.repeat(64), auditoria_ref: 'auditoria:original', consultada_en: '2026-10-02T10:00:00.000000Z' },
  } };
}

test('sin consulta inyectada muestra no disponible sin datos ni opción de lectura', async () => {
  const raiz = raizDOM(); const vista = montarPreparacionNominal({ raiz, textos }); await vista.listo;
  assert.equal(raiz.children[0].dataset.estado, 'no_disponible');
  assert.equal(todos(raiz, 'button')[0].hidden, true);
  assert.equal(todos(raiz, 'table').length, 0);
  assert.equal(raiz.children[0].getAttribute('aria-busy'), 'false');
});

test('lectura válida conserva recibo y pendientes con el traductor común', async () => {
  const dto = respuesta(); const original = structuredClone(dto); const raiz = raizDOM();
  const vista = montarPreparacionNominal({ raiz, textos, consultar: async () => dto }); await vista.listo;
  assert.equal(raiz.children[0].dataset.estado, 'disponible');
  assert.deepEqual(dto, original);
  for (const clave of dto.pendientes) assert.ok(raiz.textContent.includes(textos.traducir(`pendientes.${clave}`)));
  assert.ok(raiz.textContent.indexOf(textos.traducir('pendientes_titulo')) < raiz.textContent.indexOf(textos.traducir('tablas.relaciones')));
  const tecnico = todos(raiz, 'details').at(-1);
  assert.ok(tecnico.textContent.includes(dto.antecedentes.Evidencia.recibo_ref));
  assert.ok(tecnico.textContent.includes(textos.traducir('cobertura.parcial')));
  assert.equal(tecnico.open, undefined);
  assert.equal(todos(raiz, 'table').length, 3);
  assert.ok(raiz.textContent.includes('fin excluido'));
  assert.ok(raiz.textContent.includes('Nivel del puesto'));
  assert.equal(todos(raiz, 'button').length, 1);
});

test('denegación delegada y caída eliminan antecedentes previos sin filtrar el error', async () => {
  for (const fallo of [{ estado: 'denegado', message: 'dato:secreto' }, new Error('dato:secreto')]) {
    const raiz = raizDOM(); let contador = 0;
    const vista = montarPreparacionNominal({ raiz, textos, consultar: async () => { if (++contador === 1) return respuesta(); throw fallo; } });
    await vista.listo; await vista.reintentar();
    assert.equal(raiz.children[0].dataset.estado, fallo.estado === 'denegado' ? 'denegado' : 'no_disponible');
    assert.doesNotMatch(raiz.textContent, /dato:secreto|empleado:prueba|recibo:original/);
    assert.equal(todos(raiz, 'table').length, 0);
  }
});

test('payload de denegación o estado inesperado nunca muestra sus datos', async () => {
  for (const estado of ['denegado', 'no_disponible', 'aprobado']) {
    const raiz = raizDOM(); const dto = respuesta(); dto.estado = estado;
    const vista = montarPreparacionNominal({ raiz, textos, consultar: async () => dto }); await vista.listo;
    assert.equal(raiz.children[0].dataset.estado, estado === 'denegado' ? 'denegado' : 'no_disponible');
    assert.doesNotMatch(raiz.textContent, /empleado:prueba|recibo:original/);
  }
});

test('reintento cancela la carga anterior e ignora su respuesta tardía', async () => {
  const primera = pendiente(); const segunda = pendiente(); const señales = []; let llamada = 0; const raiz = raizDOM();
  const vista = montarPreparacionNominal({ raiz, textos, consultar: ({ signal }) => { señales.push(signal); return (++llamada === 1 ? primera : segunda).promise; } });
  assert.equal(raiz.children[0].dataset.estado, 'carga'); assert.equal(raiz.children[0].getAttribute('aria-busy'), 'true');
  const nueva = vista.reintentar(); assert.equal(señales[0].aborted, true);
  const reciente = respuesta(); reciente.antecedentes.Evidencia.recibo_ref = 'recibo:reciente'; segunda.resolve(reciente); await nueva;
  primera.resolve(respuesta()); await vista.listo;
  assert.ok(raiz.textContent.includes('recibo:reciente')); assert.doesNotMatch(raiz.textContent, /recibo:original/);
  assert.equal(señales[1].aborted, false);
});

test('desmontaje cancela, retira listeners y no repinta ni borra otra vista', async () => {
  const carga = pendiente(); let signal; let limpieza; const raiz = raizDOM();
  const vista = montarPreparacionNominal({ raiz, textos, registrarDesmontar: fn => { limpieza = fn; }, consultar: opciones => { signal = opciones.signal; return carga.promise; } });
  const boton = todos(raiz, 'button')[0]; limpieza(); limpieza();
  assert.equal(signal.aborted, true); assert.equal(boton.listeners.size, 0); assert.equal(raiz.children.length, 0);
  const otra = raiz.ownerDocument.createElement('p'); otra.textContent = 'otra vista'; raiz.append(otra);
  carga.resolve(respuesta()); await vista.listo; await vista.reintentar(); vista.desmontar();
  assert.equal(raiz.textContent, 'otra vista');
});

test('limpieza tardía conserva una vista ya montada por la raíz', async () => {
  const raiz = raizDOM(); const vista = montarPreparacionNominal({ raiz, textos, consultar: async () => respuesta() }); await vista.listo;
  const otra = raiz.ownerDocument.createElement('p'); otra.textContent = 'vista nueva'; raiz.replaceChildren(otra); vista.desmontar();
  assert.equal(raiz.textContent, 'vista nueva');
});

test('texto de fuente se inserta como texto, sin nodos ejecutables ni ficha duplicada', async () => {
  const dto = respuesta(); dto.antecedentes.Relaciones[0].RegimenRef = '<img src=x onerror="alert(1)">';
  dto.antecedentes.Persona = { Name: 'Nombre ajeno', Contacto: 'contacto ajeno' };
  dto.antecedentes.Situaciones = [{ Estado: 'situacion sensible' }]; const raiz = raizDOM();
  const vista = montarPreparacionNominal({ raiz, textos, consultar: async () => dto }); await vista.listo;
  assert.ok(raiz.textContent.includes(dto.antecedentes.Relaciones[0].RegimenRef));
  assert.equal(todos(raiz, 'img').length, 0); assert.equal(todos(raiz, 'script').length, 0);
  assert.doesNotMatch(raiz.textContent, /Nombre ajeno|contacto ajeno|situacion sensible/);
});

test('vacío, nivel ausente y certeza pendiente conservan el alcance de la fuente', async () => {
  const dto = respuesta(); dto.antecedentes.Relaciones = []; dto.antecedentes.Servicios = [];
  dto.antecedentes.Puestos[0].Nivel = null; dto.antecedentes.Puestos[0].Procedencia = { ActoRef: '', FuenteRef: '', FuenteVersion: '', Certeza: 'pendiente' };
  const raiz = raizDOM(); const vista = montarPreparacionNominal({ raiz, textos, consultar: async () => dto }); await vista.listo;
  assert.equal(raiz.children[0].dataset.estado, 'disponible'); assert.equal(todos(raiz, 'table').length, 1);
  assert.ok(raiz.textContent.includes(textos.traducir('sin_dato'))); assert.doesNotMatch(raiz.textContent, /Acreditado|22/);
});

test('versiones de catálogo cero conservan la preparación y su certeza pendiente', async () => {
  const dto = respuesta(); const relacion = dto.antecedentes.Relaciones[0]; const servicio = dto.antecedentes.Servicios[0];
  relacion.RegimenVersion = 0; relacion.Procedencia.Certeza = 'pendiente';
  servicio.ClaseVersion = 0; servicio.Procedencia.Certeza = 'pendiente';
  const original = structuredClone(dto); const raiz = raizDOM();
  const vista = montarPreparacionNominal({ raiz, textos, consultar: async () => dto }); await vista.listo;
  assert.equal(raiz.children[0].dataset.estado, 'disponible'); assert.deepEqual(dto, original);
  for (const detalle of todos(raiz, 'details').slice(0, 2)) {
    assert.ok(detalle.textContent.includes(textos.traducir('certeza.pendiente')));
    const datos = todos(detalle, 'dl')[0].children;
    assert.ok(datos.some((dato, i) => dato.tagName === 'DD' && dato.textContent === '0'
      && [textos.traducir('datos.version_regimen'), textos.traducir('datos.version_clase')].includes(datos[i - 1]?.textContent)));
  }
});

test('estado conocido de relación se traduce y el desconocido conserva su literal en procedencia', async () => {
  for (const codigo of ['vigente', 'finalizada', 'suspendida', 'codigo_nuevo_fuente']) {
    const dto = respuesta(); dto.antecedentes.Relaciones[0].Estado = codigo; const raiz = raizDOM();
    const vista = montarPreparacionNominal({ raiz, textos, consultar: async () => dto }); await vista.listo;
    const celdaEstado = todos(todos(raiz, 'table')[0], 'td')[1];
    assert.ok(celdaEstado.textContent.includes(textos.traducir(codigo === 'codigo_nuevo_fuente' ? 'estado_sin_traduccion' : `estados_relaciones.${codigo}`)));
    assert.ok(todos(raiz, 'details')[0].textContent.includes(codigo));
  }
});

test('consulta vacía parcial o no acreditada muestra cobertura sin afirmar inexistencia de registros', async () => {
  for (const cobertura of ['parcial', 'no_acreditada']) {
    const dto = respuesta(); dto.antecedentes.Cobertura = cobertura;
    dto.antecedentes.Relaciones = []; dto.antecedentes.Servicios = []; dto.antecedentes.Puestos = [];
    const raiz = raizDOM(); const vista = montarPreparacionNominal({ raiz, textos, consultar: async () => dto }); await vista.listo;
    const mensajes = todos(raiz, 'p');
    assert.ok(mensajes.some(n => n.textContent === textos.traducir('cobertura_resumen', { cobertura: textos.traducir(`cobertura.${cobertura}`) })));
    assert.equal(mensajes.filter(n => n.textContent === textos.traducir('sin_antecedentes')).length, 3);
    assert.doesNotMatch(raiz.textContent, /No constan antecedentes|No existen/);
  }
});

test('servicio reconocido y nivel informado muestran junto al valor la certeza pendiente o no acreditada', async () => {
  for (const certeza of ['pendiente', 'no_acreditado']) {
    const dto = respuesta(); dto.antecedentes.Servicios[0].Procedencia.Certeza = certeza;
    dto.antecedentes.Puestos[0].Procedencia.Certeza = certeza; const raiz = raizDOM();
    const vista = montarPreparacionNominal({ raiz, textos, consultar: async () => dto }); await vista.listo;
    const tablas = todos(raiz, 'table'); const servicio = todos(tablas[1], 'td')[1]; const nivel = todos(tablas[2], 'td')[1];
    assert.ok(servicio.textContent.includes(textos.traducir('estados_servicios.reconocido')));
    assert.ok(nivel.textContent.includes(textos.numero(22)));
    for (const celda of [servicio, nivel]) {
      assert.ok(todos(celda, 'p').some(n => n.textContent === textos.traducir(`advertencia_certeza.${certeza}`)));
      assert.equal(todos(celda, 'details').length, 0);
    }
  }
});

test('formas incompletas, fechas inválidas y cardinalidad excesiva no presentan resultado', async () => {
  const variantes = [
    dto => { dto.antecedentes.Corte = { VigenteEn: '2026-10-02', ConocidoEn: '2026-10-02T10:00:00Z' }; },
    dto => { dto.antecedentes.Corte.vigente_en = '2026-02-30'; },
    dto => { dto.antecedentes.Evidencia.recibo_ref = ''; },
    dto => { dto.antecedentes.Servicios[0].Periodo.Hasta = '2020-12-31'; },
    dto => { dto.antecedentes.Servicios[0].Periodo.Hasta = dto.antecedentes.Servicios[0].Periodo.Desde; },
    dto => { dto.antecedentes.Relaciones[0].Version = 0; },
    dto => { dto.pendientes = ['grupo']; },
    dto => { dto.antecedentes.Servicios = Array.from({ length: 129 }, () => dto.antecedentes.Servicios[0]); },
  ];
  for (const cambiar of variantes) {
    const dto = respuesta(); cambiar(dto); const raiz = raizDOM();
    const vista = montarPreparacionNominal({ raiz, textos, consultar: async () => dto }); await vista.listo;
    assert.equal(raiz.children[0].dataset.estado, 'no_disponible'); assert.equal(todos(raiz, 'table').length, 0);
  }
});

test('contrato de montaje incompleto usa un código técnico', () => {
  assert.throws(() => montarPreparacionNominal({ raiz: raizDOM() }), { name: 'TypeError', message: 'carrera.preparacion.contrato_incompleto' });
});

test('catálogos completos, fechas localizadas, estados anunciados y tablas enfocables', async () => {
  const english = crearTextos({ modulo: 'carrera-preparacion-nominal', idioma: 'en', localizacion: 'en-GB', respaldo: es, propio: en });
  assert.deepEqual(english.faltantes, []);
  for (const idioma of [textos, english]) {
    const raiz = raizDOM(); const vista = montarPreparacionNominal({ raiz, textos: idioma, consultar: async () => respuesta() }); await vista.listo;
    assert.equal(raiz.children[0].getAttribute('lang'), idioma.idioma);
    assert.equal(todos(raiz, 'p')[0].getAttribute('role'), 'status');
    const marcos = todos(raiz, 'div').filter(n => n.getAttribute('role') === 'region');
    assert.equal(marcos.length, 3); assert.ok(marcos.every(n => n.tabIndex === 0 && n.getAttribute('aria-label')));
    assert.ok(todos(raiz, 'th').every(n => n.getAttribute('scope') === 'col'));
    assert.ok(raiz.textContent.includes(idioma.fecha('2026-10-02T12:00:00Z', { dateStyle: 'medium', timeZone: 'UTC' })));
  }
});

test('botón reconsulta la misma fuente inyectada sin transporte, fixture ni almacenamiento', async () => {
  const raiz = raizDOM(); const siguiente = pendiente(); let contador = 0;
  const vista = montarPreparacionNominal({ raiz, textos, consultar: () => ++contador === 1 ? Promise.resolve(respuesta()) : siguiente.promise }); await vista.listo;
  todos(raiz, 'button')[0].click(); assert.equal(contador, 2); assert.equal(raiz.children[0].dataset.estado, 'carga');
  siguiente.resolve({ estado: 'no_disponible' }); await siguiente.promise; await Promise.resolve();
  assert.equal(raiz.children[0].dataset.estado, 'no_disponible');
  const fuente = await readFile(new URL('./preparacion-nominal.js', import.meta.url), 'utf8');
  assert.doesNotMatch(fuente, /\b(?:fetch|localStorage|sessionStorage|indexedDB|document\.cookie|globalThis)\b|escenario\.json|crearClienteCarrera/);
  assert.doesNotMatch(fuente, /innerHTML|insertAdjacentHTML/);
});
