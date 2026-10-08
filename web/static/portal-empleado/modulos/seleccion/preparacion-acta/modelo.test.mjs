import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { leerArchivo, leerSalida, MAXIMO_BYTES, FIJOS, FIJOS_COTEJO_LOCAL, COTEJO_LOCAL } from './modelo.js';
import { pintarSalida } from './vista.js';
import { cargarTextos } from '../../../../comun/textos.js';

const bytes = new Uint8Array(await readFile(new URL('./testdata/sesion-propuesta.json', import.meta.url)));
const base = () => JSON.parse(new TextDecoder().decode(bytes));
const codificar = dto => new TextEncoder().encode(JSON.stringify(dto));
const leer = dto => leerSalida(codificar(dto));
class NodoPrueba {
  constructor(documento, etiqueta) {
    this.ownerDocument = documento; this.etiqueta = etiqueta; this.hijos = []; this.texto = '';
    this.dataset = {}; this.classList = { add() {} };
  }
  set textContent(valor) { this.texto = valor; this.hijos = []; }
  get textContent() { return this.texto + this.hijos.map(h => h.textContent).join(' '); }
  append(...hijos) { this.hijos.push(...hijos); }
  replaceChildren(...hijos) { this.texto = ''; this.hijos = hijos; }
  setAttribute() {}
  querySelector(etiqueta) { return this.hijos.find(h => h.etiqueta === etiqueta || h.querySelector(etiqueta)); }
}
const documentoPrueba = {
  createElement(etiqueta) { return new NodoPrueba(this, etiqueta); },
  createDocumentFragment() { return new NodoPrueba(this, 'fragmento'); },
};
function conCotejoLocal() {
  const dto = base(); dto.cotejo_local = COTEJO_LOCAL;
  for (const [i, codigo] of ['antecedente_cotejado_local', 'fase_cotejada_local'].entries()) {
    dto.preparacion.pendientes[i].codigo = codigo; dto.mensajes[i].codigo = codigo;
  }
  return dto;
}

test('salidas reales ES/EN conservan propuesta, antecedente no cotejado y fecha ausente', async () => {
  const dto = leerSalida(bytes);
  const en = leerSalida(new Uint8Array(await readFile(new URL('./testdata/sesion-propuesta-en.json', import.meta.url))));
  assert.deepEqual(en.preparacion, dto.preparacion);
  assert.equal(dto.preparacion.estado, 'borrador_propuesto');
  assert.equal(dto.preparacion.material_propuesto.fecha_propuesta, undefined);
  assert.equal(dto.preparacion.pendientes.length, 11);
  const archivo = await leerArchivo({ size: bytes.length, arrayBuffer: async () => bytes.buffer });
  assert.deepEqual(archivo.bytes, bytes);
});
test('acepta sólo las combinaciones completas de aviso local y pendientes', () => {
  const local = conCotejoLocal();
  assert.equal(leer(local).cotejo_local, COTEJO_LOCAL);
  for (const mutar of [
    d => { delete d.cotejo_local; },
    d => { d.cotejo_local = 'verificado_institucionalmente'; },
    d => { d.preparacion.pendientes[0].codigo = 'antecedente_no_cotejado'; d.mensajes[0].codigo = 'antecedente_no_cotejado'; },
    d => { d.preparacion.pendientes[1].codigo = 'pertenencia_no_verificada'; d.mensajes[1].codigo = 'pertenencia_no_verificada'; },
    d => { d.mensajes[0].codigo = 'antecedente_no_cotejado'; },
  ]) { const dto = conCotejoLocal(); mutar(dto); assert.throws(() => leer(dto), /formato/); }
  const legadoConMarca = base(); legadoConMarca.cotejo_local = COTEJO_LOCAL;
  assert.throws(() => leer(legadoConMarca), /formato/);
  const legadoConCodigoNuevo = base(); legadoConCodigoNuevo.preparacion.pendientes[0].codigo = 'antecedente_cotejado_local';
  legadoConCodigoNuevo.mensajes[0].codigo = 'antecedente_cotejado_local';
  assert.throws(() => leer(legadoConCodigoNuevo), /formato/);
});
test('la vista atribuye al archivo el cotejo declarado en ES/EN y conserva el aviso anterior', async () => {
  for (const idioma of ['es', 'en']) {
    const textos = await cargarTextos('selectivos-acta-visor', { idioma });
    const raiz = new NodoPrueba(documentoPrueba, 'raiz');
    pintarSalida({ raiz, dto: leer(conCotejoLocal()), textos });
    assert.ok(raiz.textContent.includes(textos.traducir('antecedente_cotejo_declarado')));
    assert.ok(raiz.textContent.includes(textos.traducir('fase_cotejo_declarado')));
    assert.ok(raiz.textContent.includes(textos.traducir('motivos.antecedente_cotejado_local')));
    assert.ok(!raiz.textContent.includes(textos.traducir('antecedente_pendiente')));
    pintarSalida({ raiz, dto: leer(base()), textos });
    assert.ok(raiz.textContent.includes(textos.traducir('antecedente_pendiente')));
    assert.ok(!raiz.textContent.includes(textos.traducir('antecedente_cotejo_declarado')));
  }
});
test('se rechazan estados oficiales, datos nominales y propiedades fuera del contrato', () => {
  for (const mutar of [
    d => { d.preparacion.estado = 'firmada'; }, d => { d.preparacion.aprobada = true; },
    d => { d.preparacion.material_propuesto.asistentes = []; }, d => { d.preparacion.material_propuesto.votos = []; },
    d => { d.preparacion.material_propuesto.antecedente_tribunal.miembros = []; },
    d => { d.preparacion.material_propuesto.acuerdos_propuestos[0].adoptado = true; },
    d => { d.preparacion.material_propuesto.alcance = 'nominal'; }, d => { d.preparacion.material_propuesto.version_material = 0; },
    d => { d.preparacion.material_propuesto.antecedente_tribunal.huella_aportada_sha256 = 'A'.repeat(64); },
    d => { d.preparacion.material_propuesto.fase_propuesta = '<img>'; },
    d => { d.preparacion.material_propuesto.fecha_propuesta = '2026-02-30T12:00:00Z'; },
  ]) { const dto = base(); mutar(dto); assert.throws(() => leer(dto), /formato/); }
});
test('referencias duplicadas y acuerdos sin punto de agenda son incompatibles', () => {
  for (const mutar of [
    d => { d.preparacion.material_propuesto.orden_dia_propuesto.push(d.preparacion.material_propuesto.orden_dia_propuesto[0]); },
    d => { d.preparacion.material_propuesto.acuerdos_propuestos.push(d.preparacion.material_propuesto.acuerdos_propuestos[0]); },
    d => { d.preparacion.material_propuesto.acuerdos_propuestos[0].punto_ref = 'punto:ajeno'; },
  ]) { const dto = base(); mutar(dto); assert.throws(() => leer(dto), /formato/); }
});
test('los pendientes fijos no se pueden retirar, cambiar o desvincular de sus mensajes', () => {
  for (const mutar of [
    d => { d.preparacion.pendientes.pop(); }, d => { d.preparacion.pendientes[0].codigo = 'circuito_pendiente'; },
    d => { d.preparacion.pendientes[1] = d.preparacion.pendientes[0]; },
    d => { d.mensajes[0].campo = 'fase_propuesta'; }, d => { d.mensajes[0].codigo = 'material_ausente'; },
  ]) { const dto = base(); mutar(dto); assert.throws(() => leer(dto), /formato/); }
});
test('JSON rechaza claves duplicadas, prototipos, profundidad excesiva y bytes UTF-8 inválidos', () => {
  for (const texto of ['{"titulo":1,"titulo":2}', '{"__proto__":{}}', '{"a":{"constructor":1}}',
    '{"a":{"prototype":1}}', '['.repeat(34) + '0' + ']'.repeat(34)]) {
    assert.throws(() => leerSalida(new TextEncoder().encode(texto)), /formato/);
  }
  assert.throws(() => leerSalida(new Uint8Array([255])), /formato/);
});
test('textos propuestos limitados a 4096 bytes, sin controles ni sustitución de propuestas', () => {
  for (const texto of ['á'.repeat(2049), 'texto\ntexto']) {
    const dto = base(); dto.preparacion.material_propuesto.orden_dia_propuesto[0].texto_propuesto = texto;
    assert.throws(() => leer(dto), /formato/);
  }
  const dto = base(); dto.preparacion.material_propuesto.orden_dia_propuesto[0].texto_propuesto = 'á'.repeat(2048);
  assert.equal(leer(dto).preparacion.material_propuesto.orden_dia_propuesto.length, 1);
});
test('JSON con sustituto aislado se rechaza; un par válido y U+FFFD literal se conservan', () => {
  for (const valor of ['\ud800', '\udfff', '\ud83d\ude00', '\ufffd']) {
    const dto = base(); dto.preparacion.material_propuesto.orden_dia_propuesto[0].texto_propuesto = valor;
    if (valor.length === 1 && valor !== '\ufffd') assert.throws(() => leer(dto), /formato/);
    else assert.equal(leer(dto).preparacion.material_propuesto.orden_dia_propuesto[0].texto_propuesto, valor);
  }
});
test('material incompleto sigue pendiente sin inventar fecha, agenda o acuerdos', () => {
  const dto = base(), m = dto.preparacion.material_propuesto;
  m.sesion_ref = ''; m.orden_dia_propuesto = []; m.acuerdos_propuestos = [];
  for (const campo of ['sesion_ref', 'orden_dia_propuesto', 'acuerdos_propuestos']) {
    dto.preparacion.pendientes.push({ campo, codigo: 'material_ausente' }); dto.mensajes.push({ campo, codigo: 'material_ausente', mensaje: '' });
  }
  assert.deepEqual(leer(dto).preparacion.material_propuesto.orden_dia_propuesto, []);
});
test('límites de archivo y listas se comprueban antes de mostrar contenido', async () => {
  for (const size of [0, MAXIMO_BYTES + 1]) await assert.rejects(leerArchivo({ size, arrayBuffer() { throw new Error('no leer'); } }), /tamano/);
  await assert.rejects(leerArchivo({ size: 1, arrayBuffer: async () => new ArrayBuffer(MAXIMO_BYTES + 1) }), /tamano/);
  const dto = base(); dto.preparacion.material_propuesto.orden_dia_propuesto = Array(101).fill(dto.preparacion.material_propuesto.orden_dia_propuesto[0]);
  assert.throws(() => leer(dto), /formato/);
});
test('catálogos reales ES/EN traducen cada pendiente sin claves ausentes', async () => {
  for (const idioma of ['es', 'en']) {
    const t = await cargarTextos('selectivos-acta-visor', { idioma }); assert.deepEqual(t.faltantes, []);
    for (const fijos of [FIJOS, FIJOS_COTEJO_LOCAL]) {
      for (const [campo, codigo] of Object.entries(fijos)) { assert.ok(t.traducir(`campos.${campo}`)); assert.ok(t.traducir(`motivos.${codigo}`)); }
    }
    for (const clave of ['cargado_cotejo_declarado', 'fase_cotejo_declarado', 'antecedente_cotejo_declarado',
      'referencias_limite_cotejo_declarado', 'antecedente_huella_cotejo_declarado']) assert.ok(t.traducir(clave));
    for (const campo of ['sesion_ref', 'fecha_propuesta', 'orden_dia_propuesto', 'acuerdos_propuestos', 'textos_orden_dia', 'textos_acuerdos']) assert.ok(t.traducir(`campos.${campo}`));
  }
});
