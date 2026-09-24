import assert from "node:assert/strict";
import test from "node:test";
import { montarRegistroB2 } from "./registro-b2.js";
import { crearClienteRegistroB2, ErrorRegistroB2 } from "./registro-b2-cliente.js";
import { montarActosRegistroB2, accionesRegistroB2Disponibles } from "./registro-b2-actos.js";

function raizFalsa() {
  class Nodo {
    constructor(documento, etiqueta) { this.ownerDocument = documento; this.tagName = etiqueta; this.children = []; this.dataset = {}; this.listeners = new Map(); this.attrs = new Map(); this.parent = null; this.textContent = ""; this.value = ""; }
    append(...hijos) { for (const hijo of hijos) { if (hijo.parent) hijo.parent.children = hijo.parent.children.filter((n) => n !== hijo); hijo.parent = this; this.children.push(hijo); } }
    replaceChildren(...hijos) { for (const n of this.children) n.parent = null; this.children = []; this.append(...hijos); }
    remove() { if (this.parent) { this.parent.children = this.parent.children.filter((n) => n !== this); this.parent = null; } }
    setAttribute(clave, valor) { this.attrs.set(clave, String(valor)); }
    addEventListener(tipo, fn) { this.listeners.set(tipo, fn); }
    focus() { this.enfocado = true; }
  }
  const d = { createElement(etiqueta) { return new Nodo(d, etiqueta); } };
  return new Nodo(d, "root");
}
function nodos(n) { return [n, ...n.children.flatMap(nodos)]; }
function buscar(n, predicado) { return nodos(n).find(predicado); }
function texto(n) { return nodos(n).map((item) => item.textContent).join(" "); }
const completar = () => new Promise((resolve) => setImmediate(resolve));
const corte = { vigente_en: "2026-09-25", conocido_en: "2026-09-25T10:00:00Z" };
const traza = { desde: "2024-01-01", registrada_en: "2024-01-01T10:00:00Z", version: 1, acto_ref: "acto-uno", fuente_ref: "fuente-uno", fuente_version: "1" };
function ficha(relaciones = []) { return { ficha: { empleado_ref: "emp_aaaaaaaaaaaaaaaaaaaaaa", persona_ref: "per_bbbbbbbbbbbbbbbbbbbbbb", corte, version: 1, eficacia_administrativa: false, firma_oficial: false, relaciones, ocupaciones: [{ ocupacion_ref: "ocupacion-uno", relacion_ref: "relacion-uno", plaza_ref: "plaza-uno", puesto_ref: "puesto-uno", unidad_ref: "unidad-uno", traza }], situaciones: [], servicios: [] }, evidencia: { recibo_ref: "recibo-uno" } }; }
const relacion = (ref) => ({ relacion_ref: ref, unidad_ref: "unidad-uno", estado: "vigente", traza });
const opcion = (ref, denominacion, otros = {}) => ({ ref, denominacion, ...otros });
const catalogos = {
  organismos: [opcion("org_sintetico", "Organismo sintético")], unidades: [opcion("unidad_sintetica", "Unidad sintética")],
  regimenes: [opcion("regimen_sintetico", "Régimen sintético", { version: 1, estado: "publicada" })], modalidades: [opcion("modalidad_sintetica", "Modalidad sintética", { version: 2, estado: "publicada" })],
  plazas: [opcion("plaza_sintetica", "Plaza sintética", { version_ref: "version_plaza_sintetica" })], puestos: [],
  situaciones: [], clasesServicio: [], actos: [opcion("acto_sintetico", "Acto sintético")],
  fuentes: [opcion("fuente_sintetica", "Fuente sintética", { version: 1, huella_sha256: "a".repeat(64) })],
};

test("la ficha exige empleado seleccionado y no consulta identidad implícita", async () => {
  const raiz = raizFalsa(); let llamadas = 0;
  const montaje = montarRegistroB2({ raiz, cliente: { consultarFicha: () => { llamadas++; return ficha(); }, listarVacantes: () => { throw Error("no esperado"); } }, reloj: () => new Date("2026-09-25T10:00:00Z") });
  await completar();
  assert.equal(llamadas, 0);
  assert.match(texto(raiz), /Seleccione un empleado/);
  assert.equal(buscar(raiz, (n) => n.tagName === "summary").textContent, "?");
  montaje.cambiarEmpleado("emp_aaaaaaaaaaaaaaaaaaaaaa"); await completar();
  assert.equal(llamadas, 1);
  assert.match(texto(raiz), /Relaciones de servicio/);
  montaje.desmontar();
  assert.equal(raiz.children.length, 0);
});

test("varias relaciones requieren elección antes de mostrar ocupaciones", async () => {
  const raiz = raizFalsa();
  montarRegistroB2({ raiz, empleadoRef: "emp_aaaaaaaaaaaaaaaaaaaaaa", cliente: { consultarFicha: () => ficha([relacion("relacion-uno"), relacion("relacion-dos")]), listarVacantes: () => { throw Error("no esperado"); } }, reloj: () => new Date("2026-09-25T10:00:00Z") });
  await completar();
  assert.match(texto(raiz), /Seleccione la relación de servicio/);
  assert.doesNotMatch(texto(raiz), /plaza-uno/);
  const selector = buscar(raiz, (n) => n.dataset.registroB2Relacion !== undefined);
  selector.value = "relacion-uno"; selector.listeners.get("change")();
  assert.match(texto(raiz), /plaza-uno/);
});

test("fallo 503 no finge vacantes vacías y ofrece reintento", async () => {
  const raiz = raizFalsa(); let llamadas = 0;
  montarRegistroB2({ raiz, cliente: { consultarFicha: () => ficha(), listarVacantes: () => { llamadas++; throw new ErrorRegistroB2("estado_no_valido", 503); } }, reloj: () => new Date("2026-09-25T10:00:00Z") });
  buscar(raiz, (n) => n.dataset.registroB2Tab === "vacantes").listeners.get("click")(); await completar();
  assert.equal(llamadas, 1);
  assert.match(texto(raiz), /No se pudo completar la consulta/);
  assert.doesNotMatch(texto(raiz), /No hay registros para esta consulta/);
  buscar(raiz, (n) => n.textContent === "Reintentar consulta").listeners.get("click")(); await completar();
  assert.equal(llamadas, 2);
});

test("solo el código 503 de cobertura acreditada se presenta como indeterminada", async () => {
  const raiz = raizFalsa();
  montarRegistroB2({ raiz, cliente: { consultarFicha: () => ficha(), listarVacantes: () => { throw new ErrorRegistroB2("cobertura_no_acreditada", 503); } }, reloj: () => new Date("2026-09-25T10:00:00Z") });
  buscar(raiz, (n) => n.dataset.registroB2Tab === "vacantes").listeners.get("click")(); await completar();
  assert.match(texto(raiz), /No se puede determinar la ocupación/);
  assert.doesNotMatch(texto(raiz), /No hay registros para esta consulta/);
});

test("cambiar empleado cancela la lectura anterior y descarta su respuesta", async () => {
  const raiz = raizFalsa(); let resolver; const señales = [];
  const montaje = montarRegistroB2({ raiz, empleadoRef: "emp_aaaaaaaaaaaaaaaaaaaaaa", cliente: { consultarFicha: ({ signal }) => { señales.push(signal); return señales.length === 1 ? new Promise((resolve) => { resolver = resolve; }) : ficha(); }, listarVacantes: () => { throw Error("no esperado"); } }, reloj: () => new Date("2026-09-25T10:00:00Z") });
  montaje.cambiarEmpleado("emp_cccccccccccccccccccccc");
  resolver(ficha([relacion("relacion-antigua")])); await completar();
  assert.equal(señales[0].aborted, true);
  assert.doesNotMatch(texto(raiz), /relacion-antigua/);
});

test("el cliente usa GET exacto, corte bitemporal y rechaza acceso denegado", async () => {
  const rutas = [];
  const cliente = crearClienteRegistroB2({ fetchImpl: async (url, opciones) => {
    rutas.push([url, opciones]);
    return new Response(JSON.stringify({ data: ficha().evidencia ? ficha() : {} }), { status: 200, headers: { "content-type": "application/json; charset=utf-8" } });
  } });
  await cliente.consultarFicha({ empleadoRef: "emp_aaaaaaaaaaaaaaaaaaaaaa", vigenteEn: "2026-09-25", conocidoEn: "2026-09-25T10:00:00.000000Z" });
  assert.match(rutas[0][0], /^\/api\/vec\/personal\/empleados\/emp_aaaaaaaaaaaaaaaaaaaaaa\?/u);
  assert.match(rutas[0][0], /vigente_en=2026-09-25/u);
  assert.match(rutas[0][0], /conocido_en=2026-09-25T10%3A00%3A00\.000000Z/u);
  assert.equal(rutas[0][1].credentials, "omit");
  const denegado = crearClienteRegistroB2({ fetchImpl: async () => new Response(null, { status: 403 }) });
  await assert.rejects(denegado.consultarFicha({ empleadoRef: "emp_aaaaaaaaaaaaaaaaaaaaaa", vigenteEn: "2026-09-25", conocidoEn: "2026-09-25T10:00:00.000000Z" }), (error) => error instanceof ErrorRegistroB2 && error.estado === 403);
  const cobertura = crearClienteRegistroB2({ fetchImpl: async () => new Response(JSON.stringify({ error: { codigo: "cobertura_no_acreditada" } }), { status: 503, headers: { "content-type": "application/json; charset=utf-8" } }) });
  await assert.rejects(cobertura.listarVacantes({ vigenteEn: "2026-09-25", conocidoEn: "2026-09-25T10:00:00.000000Z" }), (error) => error instanceof ErrorRegistroB2 && error.codigo === "cobertura_no_acreditada");
});

test("la pestaña de actuaciones solo aparece con objetivo y catálogos autorizados", () => {
  const raiz = raizFalsa(); const cliente = { consultarFicha: () => ficha(), listarVacantes: () => ({}), registrarAlta() {}, registrarHecho() {} };
  montarRegistroB2({ raiz, cliente, catalogos });
  assert.equal(buscar(raiz, (n) => n.dataset.registroB2Tab === "actos"), undefined);
  assert.equal(accionesRegistroB2Disponibles({ cliente, catalogos, personaRef: "per_aaaaaaaaaaaaaaaaaaaaaa" }), true);
  const sinVersion = { ...catalogos, regimenes: [opcion("regimen_sintetico", "Régimen sintético", { estado: "publicada" })] };
  const retirada = { ...catalogos, modalidades: [opcion("modalidad_sintetica", "Modalidad retirada", { version: 2, estado: "retirada" })] };
  assert.equal(accionesRegistroB2Disponibles({ cliente, catalogos: sinVersion, personaRef: "per_aaaaaaaaaaaaaaaaaaaaaa" }), false);
  assert.equal(accionesRegistroB2Disponibles({ cliente, catalogos: retirada, personaRef: "per_aaaaaaaaaaaaaaaaaaaaaa" }), false);
});

test("el alta revisada conserva cuerpo y clave en reintento incierto y muestra recibo confirmado", async () => {
  const raiz = raizFalsa(); const envios = []; let intento = 0;
  const cliente = { registrarAlta(cuerpo, opciones) { envios.push({ cuerpo, clave: opciones.claveIdempotencia }); intento++; if (intento === 1) throw new ErrorRegistroB2("resultado_incierto"); return { recibo: { recibo_ref: "recibo_sintetico", registrado_en: "2026-09-25T10:00:00Z" }, accesoActual: { estado_replay: "replay" } }; }, registrarHecho() {} };
  montarActosRegistroB2({ raiz, cliente, catalogos, personaRef: "per_aaaaaaaaaaaaaaaaaaaaaa" });
  const campos = Object.fromEntries(nodos(raiz).filter((n) => n.dataset.registroB2Campo).map((n) => [n.dataset.registroB2Campo, n]));
  Object.assign(campos.organismo, { value: "org_sintetico" }); Object.assign(campos.unidad, { value: "unidad_sintetica" });
  Object.assign(campos.regimen, { value: "regimen_sintetico" }); Object.assign(campos.modalidad, { value: "modalidad_sintetica" });
  Object.assign(campos.vigente_desde, { value: "2026-09-25" }); Object.assign(campos.acto, { value: "acto_sintetico" }); Object.assign(campos.fuente, { value: "fuente_sintetica" });
  buscar(raiz, (n) => n.tagName === "form").listeners.get("submit")({ preventDefault() {} });
  assert.match(texto(raiz), /Confirmar registro/);
  assert.doesNotMatch(texto(raiz), /recibo_sintetico/);
  buscar(raiz, (n) => n.textContent === "Confirmar registro").listeners.get("click")(); await completar();
  assert.match(texto(raiz), /No se pudo confirmar el resultado/);
  buscar(raiz, (n) => n.textContent === "Reintentar exactamente").listeners.get("click")(); await completar();
  assert.deepEqual(envios[0], envios[1]);
  assert.match(texto(raiz), /Registro ya conservado/);
  assert.match(texto(raiz), /recibo_sintetico/);
  assert.equal(Object.hasOwn(envios[0].cuerpo, "empleado_ref"), false);
});

test("POST de alta envía solo campos gobernados, sin cookies, y exige recibo no eficaz", async () => {
  const peticiones = []; const idempotencia = "123e4567-e89b-42d3-a456-426614174000";
  const cuerpo = { persona_ref: "per_aaaaaaaaaaaaaaaaaaaaaa", organismo_ref: "org_sintetico", unidad_ref: "unidad_sintetica", regimen: { ref: "regimen_sintetico", version: 1 }, modalidad: { ref: "modalidad_sintetica", version: 2 }, vigente_desde: "2026-09-25", acto_ref: "acto_sintetico", fuente_ref: "fuente_sintetica", fuente_version: 1, fuente_huella_sha256: "a".repeat(64) };
  const recibo = { recibo_ref: `perrec_${"a".repeat(32)}`, empleado_ref: "emp_aaaaaaaaaaaaaaaaaaaaaa", relacion_ref: "rel_bbbbbbbbbbbbbbbbbbbbbb", proyeccion_ref: "pep_cccccccccccccccccccccc", tipo: "alta", version: 1, registrado_en: "2026-09-25T10:00:00Z", decision_ref: "decision_sintetica", efecto_ref: "efecto_sintetico", consumo_huella_sha256: "a".repeat(64), auditoria_ref: "auditoria_sintetica", eficacia_administrativa: false, firma_oficial: false };
  const acceso_actual = { decision_ref: "decision_actual", efecto_ref: "efecto_actual", consumo_huella_sha256: "b".repeat(64), auditoria_ref: "auditoria_actual", consultada_en: "2026-09-25T10:01:00Z", estado_replay: "registrado" };
  const cliente = crearClienteRegistroB2({ fetchImpl: async (url, opciones) => { peticiones.push([url, opciones]); return new Response(JSON.stringify({ data: { recibo, acceso_actual } }), { status: 201, headers: { "content-type": "application/json; charset=utf-8" } }); } });
  assert.equal((await cliente.registrarAlta(cuerpo, { claveIdempotencia: idempotencia })).recibo.recibo_ref, recibo.recibo_ref);
  assert.equal(peticiones[0][0], "/api/vec/personal/empleados");
  assert.equal(peticiones[0][1].credentials, "omit");
  assert.equal(peticiones[0][1].headers["Idempotency-Key"], idempotencia);
  assert.deepEqual(JSON.parse(peticiones[0][1].body), cuerpo);
  await assert.rejects(cliente.registrarAlta({ ...cuerpo, regimen: undefined, regimen_ref: "regimen_sintetico" }, { claveIdempotencia: idempotencia }), TypeError);
  const repetido = crearClienteRegistroB2({ fetchImpl: async () => new Response(JSON.stringify({ data: { recibo, acceso_actual: { ...acceso_actual, estado_replay: "replay", decision_ref: "decision_relectura" } } }), { status: 200, headers: { "content-type": "application/json; charset=utf-8" } }) });
  const replay = await repetido.registrarAlta(cuerpo, { claveIdempotencia: idempotencia });
  assert.deepEqual(replay.recibo, recibo);
  assert.equal(replay.accesoActual.decision_ref, "decision_relectura");
  const sinEficacia = crearClienteRegistroB2({ fetchImpl: async () => new Response(JSON.stringify({ data: { recibo: { ...recibo, eficacia_administrativa: true }, acceso_actual } }), { status: 201, headers: { "content-type": "application/json; charset=utf-8" } }) });
  await assert.rejects(sinEficacia.registrarAlta(cuerpo, { claveIdempotencia: idempotencia }), (error) => error instanceof ErrorRegistroB2 && error.codigo === "resultado_incierto");
});

test("la ocupación exige relación explícita y versión de la plaza autorizada", async () => {
  const raiz = raizFalsa(); let enviado;
  const cliente = { registrarAlta() { throw Error("no esperado"); }, registrarHecho(cuerpo) { enviado = cuerpo; return { recibo: { recibo_ref: "recibo_sintetico", registrado_en: "2026-09-25T10:00:00Z" }, accesoActual: { estado_replay: "registrado" } }; } };
  montarActosRegistroB2({ raiz, cliente, catalogos, empleadoRef: "emp_aaaaaaaaaaaaaaaaaaaaaa", ficha: ficha([relacion("rel_aaaaaaaaaaaaaaaaaaaaaa")]).ficha });
  const tipo = buscar(raiz, (n) => n.dataset.registroB2Campo === "tipo"); tipo.value = "ocupacion"; tipo.listeners.get("change")();
  const campos = Object.fromEntries(nodos(raiz).filter((n) => n.dataset.registroB2Campo).map((n) => [n.dataset.registroB2Campo, n]));
  Object.assign(campos.relacion, { value: "rel_aaaaaaaaaaaaaaaaaaaaaa" }); Object.assign(campos.unidad, { value: "unidad_sintetica" });
  Object.assign(campos.modalidad, { value: "modalidad_sintetica" }); Object.assign(campos.plaza, { value: "plaza_sintetica" });
  Object.assign(campos.clase, { value: "titular" }); Object.assign(campos.vigente_desde, { value: "2026-09-25" });
  Object.assign(campos.acto, { value: "acto_sintetico" }); Object.assign(campos.fuente, { value: "fuente_sintetica" });
  buscar(raiz, (n) => n.tagName === "form").listeners.get("submit")({ preventDefault() {} });
  buscar(raiz, (n) => n.textContent === "Confirmar registro").listeners.get("click")(); await completar();
  assert.equal(enviado.relacion_ref, "rel_aaaaaaaaaaaaaaaaaaaaaa");
  assert.equal(enviado.relacion_version_esperada, 1);
  assert.equal(enviado.revision_esperada, 1);
  assert.equal(enviado.version_plaza_ref, "version_plaza_sintetica");
  assert.deepEqual(enviado.modalidad, { ref: "modalidad_sintetica", version: 2 });
  assert.equal(enviado.clase_ocupacion, "titular");
  assert.equal(Object.hasOwn(enviado, "ocupacion_ref"), false);
});

test("la revisión de relación conserva su referencia y exige versión explícita", async () => {
  const raiz = raizFalsa(); let enviado;
  const cliente = { registrarAlta() {}, registrarHecho(cuerpo) { enviado = cuerpo; return { recibo: { recibo_ref: "recibo_sintetico", registrado_en: "2026-09-25T10:00:00Z" }, accesoActual: { estado_replay: "registrado" } }; } };
  montarActosRegistroB2({ raiz, cliente, catalogos, empleadoRef: "emp_aaaaaaaaaaaaaaaaaaaaaa", ficha: ficha([relacion("rel_aaaaaaaaaaaaaaaaaaaaaa")]).ficha });
  const tipo = buscar(raiz, (n) => n.dataset.registroB2Campo === "tipo"); tipo.value = "revision_relacion"; tipo.listeners.get("change")();
  const campos = Object.fromEntries(nodos(raiz).filter((n) => n.dataset.registroB2Campo).map((n) => [n.dataset.registroB2Campo, n]));
  for (const [clave, valor] of Object.entries({ relacion: "rel_aaaaaaaaaaaaaaaaaaaaaa", unidad: "unidad_sintetica", regimen: "regimen_sintetico", modalidad: "modalidad_sintetica", estado: "suspendida", vigente_desde: "2026-09-25", acto: "acto_sintetico", fuente: "fuente_sintetica" })) campos[clave].value = valor;
  buscar(raiz, (n) => n.tagName === "form").listeners.get("submit")({ preventDefault() {} });
  buscar(raiz, (n) => n.textContent === "Confirmar registro").listeners.get("click")(); await completar();
  assert.equal(enviado.tipo, "relacion");
  assert.equal(enviado.relacion_ref, "rel_aaaaaaaaaaaaaaaaaaaaaa");
  assert.equal(enviado.relacion_version_esperada, 1);
  assert.equal(enviado.revision_esperada, 2);
  assert.deepEqual(enviado.regimen, { ref: "regimen_sintetico", version: 1 });
});

test("el conflicto de catálogo o revisión retira el formulario pendiente", async () => {
  const raiz = raizFalsa();
  const cliente = { registrarAlta() { throw new ErrorRegistroB2("acto_rechazado", 409); }, registrarHecho() {} };
  montarActosRegistroB2({ raiz, cliente, catalogos, personaRef: "per_aaaaaaaaaaaaaaaaaaaaaa" });
  const campos = Object.fromEntries(nodos(raiz).filter((n) => n.dataset.registroB2Campo).map((n) => [n.dataset.registroB2Campo, n]));
  for (const [clave, valor] of Object.entries({ organismo: "org_sintetico", unidad: "unidad_sintetica", regimen: "regimen_sintetico", modalidad: "modalidad_sintetica", vigente_desde: "2026-09-25", acto: "acto_sintetico", fuente: "fuente_sintetica" })) campos[clave].value = valor;
  buscar(raiz, (n) => n.tagName === "form").listeners.get("submit")({ preventDefault() {} });
  buscar(raiz, (n) => n.textContent === "Confirmar registro").listeners.get("click")(); await completar();
  assert.match(texto(raiz), /Los datos seleccionados han cambiado/);
  assert.equal(buscar(raiz, (n) => n.tagName === "form"), undefined);
  assert.equal(buscar(raiz, (n) => n.textContent === "Confirmar registro"), undefined);
});
