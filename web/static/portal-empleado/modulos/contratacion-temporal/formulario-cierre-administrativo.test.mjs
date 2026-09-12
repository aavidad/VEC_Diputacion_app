import assert from "node:assert/strict";
import test from "node:test";
import { montarFormularioCierreAdministrativo } from "./formulario-cierre-administrativo.js";
import { crearClienteCierreAdministrativoHTTP, RUTA_CIERRE_ADMINISTRATIVO } from "./cliente-http-cierre-administrativo.js";
import { crearTraductorContratacionTemporal } from "./i18n.js";
const p = { expediente_ref: "expediente:1", seguimiento_ref: "seguimiento:1", version_esperada: 3, motivos: ["fin_ejercicio_sintetico"] }, id = "11111111-1111-4111-8111-111111111111", s = { expediente_ref: p.expediente_ref, seguimiento_ref: p.seguimiento_ref, version_esperada: p.version_esperada, clave_idempotencia: id, transicion_clave: "cerrar_administrativamente_sin_cese", motivo_clave: "fin_ejercicio_sintetico" }, r = { recibo_ref: "recibo:1", version_seguimiento: 4 };
function raiz() { const handlers = {}; const emitir = (selector, elements) => handlers.submit({ type: "submit", preventDefault() {}, target: { matches: x => x === selector, elements } }); return { innerHTML: "", addEventListener(tipo, fn) { handlers[tipo] = fn; }, removeEventListener(tipo) { delete handlers[tipo]; }, replaceChildren() { this.innerHTML = ""; }, preparar(motivo = s.motivo_clave) { return emitir("[data-ct-cierre-administrativo-form]", { motivo_clave: { value: motivo } }); }, continuar() { return emitir("[data-ct-cierre-continuar-form]"); } }; }
function monta(c, confirmarOperacion = () => true, t = undefined) { const x = raiz(); return { x, d: montarFormularioCierreAdministrativo({ raiz: x, cliente: c, preparacion: p, confirmarOperacion, generarClaveIdempotencia: () => id, t }) }; }
test("preparar deja seis datos y guardado visibles sin POST", async () => { const a = []; const x = monta({ async cerrar(q) { a.push(q); return r; } }); await x.x.preparar(); assert.deepEqual(a, []); assert.match(x.x.innerHTML, /Guardar datos de recuperación/); assert.match(x.x.innerHTML, new RegExp(id)); assert.match(x.x.innerHTML, /Cerrar administrativamente/); });
test("continuar confirma y POST contiene intención exacta", async () => { let c; const cliente = crearClienteCierreAdministrativoHTTP({ ejecutar: async x => (c = x, r), validarOpciones: o => o ?? {} }), x = monta(cliente); await x.x.preparar(); await x.x.continuar(); assert.equal(c.ruta, RUTA_CIERRE_ADMINISTRATIVO); assert.deepEqual(c.entrada, s); assert.deepEqual(c.estadoEsperado, [200, 201]); assert.match(x.x.innerHTML, /recibo:1/); });
test("cancelar confirmación no hace POST y permite continuar la misma intención", async () => { const a = []; let confirma = false; const x = monta({ async cerrar(q) { a.push(q); return r; } }, () => confirma); await x.x.preparar(); await x.x.continuar(); assert.deepEqual(a, []); assert.match(x.x.innerHTML, /Cerrar administrativamente/); confirma = true; await x.x.continuar(); assert.deepEqual(a, [s]); });
test("incertidumbre conserva datos de recuperación y no reintenta", async () => { const a = []; const x = monta({ async cerrar(q) { a.push(q); throw Object.assign(new Error(), { estado: 503 }); } }); await x.x.preparar(); await x.x.continuar(); assert.deepEqual(a, [s]); assert.match(x.x.innerHTML, /Guardar datos de recuperación/); });
test("401,403,409 son claros y no cambian intención", async () => { for (const estado of [401, 403, 409]) { const a = []; const x = monta({ async cerrar(q) { a.push(q); throw Object.assign(new Error(), { estado }); } }); await x.x.preparar(); await x.x.continuar(); assert.equal(a.length, 1); assert.match(x.x.innerHTML, estado === 401 ? /identificarse/ : estado === 403 ? /permiso/ : /conflicto/); } });
test("desmontar aborta y descarta respuesta tardía", async () => { let resolver, signal; const x = monta({ cerrar(_q, o) { signal = o.signal; return new Promise(z => resolver = z); } }); await x.x.preparar(); const v = x.x.continuar(); await new Promise(z => setImmediate(z)); x.d(); assert.equal(signal.aborted, true); resolver(r); await v; assert.equal(x.x.innerHTML, ""); });
test("preparación ausente sin contexto queda inactiva", () => { const x = raiz(); const d = montarFormularioCierreAdministrativo({ raiz: x, cliente: { cerrar() { assert.fail("POST"); } }, preparacion: null }); assert.match(x.innerHTML, /no está disponible/); d(); assert.equal(x.innerHTML, ""); });

test("las seis etiquetas y el motivo se traducen, se escapan y conservan la clave exacta del POST", async () => {
  const actual = { ...p, motivos: ["cierre_administrativo_ejercicio"] };
  const t = crearTraductorContratacionTemporal({
    cierre_titulo: "Cierre <revisado>", cierre_dato_expediente: "Expediente <seguro>",
    cierre_motivo_cierre_administrativo_ejercicio: "Cierre humano <seguro>",
  });
  const x = raiz(); let enviada = null;
  montarFormularioCierreAdministrativo({ raiz: x, cliente: { async cerrar(solicitud) { enviada = solicitud; return { recibo_ref: "recibo:2", version_seguimiento: 4 }; } }, preparacion: actual, confirmarOperacion: () => true, generarClaveIdempotencia: () => id, t });
  assert.match(x.innerHTML, /Cierre humano &lt;seguro&gt;/u);
  assert.match(x.innerHTML, /value="cierre_administrativo_ejercicio"/u);
  assert.doesNotMatch(x.innerHTML, />cierre_administrativo_ejercicio</u);
  await x.preparar("cierre_administrativo_ejercicio");
  for (const etiqueta of ["Expediente &lt;seguro&gt;", "Seguimiento", "Versión esperada", "Clave de recuperación", "Transición", "Motivo"]) assert.match(x.innerHTML, new RegExp(etiqueta, "u"));
  await x.continuar();
  assert.equal(enviada.motivo_clave, "cierre_administrativo_ejercicio");
  assert.doesNotMatch(x.innerHTML, /<revisado>|<seguro>/u);
});
