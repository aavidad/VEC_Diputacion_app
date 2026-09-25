import assert from "node:assert/strict";
import test from "node:test";
import { calcularHuellaPublicacionCatalogoB2, crearClienteCatalogosRegistroB2, ErrorCatalogosRegistroB2 } from "./registro-b2-catalogos-cliente.js";
import { cargarOpcionesPublicadasCatalogoB2, montarCatalogosRegistroB2 } from "./registro-b2-catalogos.js";

const completar = () => new Promise((resolver) => setImmediate(resolver));
const huella = "b08fcb659efbcff4bf474e6d1d1051ca7e110007562dce9253bcd2fb69d47e62";
const evidencia = { decision_ref: "dec_sintetica", auditoria_ref: "aud_sintetica", consumo_huella_sha256: "a".repeat(64), efecto_ref: "organismo:dipgra:regimen", consultada_en: "2026-09-25T10:00:00Z" };
function nodoFalso() {
  class Nodo {
    constructor(documento, etiqueta) { this.ownerDocument = documento; this.tagName = etiqueta; this.children = []; this.dataset = {}; this.listeners = new Map(); this.attrs = new Map(); this.parent = null; this.textContent = ""; this.value = ""; }
    append(...hijos) { for (const h of hijos) { if (h.parent) h.parent.children = h.parent.children.filter((n) => n !== h); h.parent = this; this.children.push(h); } }
    replaceChildren(...hijos) { this.children.forEach((h) => { h.parent = null; }); this.children = []; this.append(...hijos); }
    remove() { if (this.parent) { this.parent.children = this.parent.children.filter((n) => n !== this); this.parent = null; } }
    setAttribute(k, v) { this.attrs.set(k, String(v)); }
    addEventListener(k, f) { this.listeners.set(k, f); }
  }
  const d = { createElement(etiqueta) { return new Nodo(d, etiqueta); } }; return new Nodo(d, "root");
}
function nodos(n) { return [n, ...n.children.flatMap(nodos)]; }
function buscar(n, pred) { return nodos(n).find(pred); }
function texto(n) { return nodos(n).map((x) => x.textContent).join(" "); }

test("la huella de publicación coincide con el contrato canónico Go, con UTF-8 y último campo", async () => {
  const calculada = await calcularHuellaPublicacionCatalogoB2({ organismoRef: "organismo:dipgra", tipo: "regimen", ref: "regimen:uno", version: 1, revision: 1, denominacion: "Régimen sintético", vigenteDesde: "2026-09-20" });
  assert.equal(calculada, huella);
  assert.notEqual(await calcularHuellaPublicacionCatalogoB2({ organismoRef: "organismo:dipgra", tipo: "regimen", ref: "regimen:uno", version: 1, revision: 1, denominacion: "Otro régimen", vigenteDesde: "2026-09-20" }), huella);
});

test("GET vacío entrega organismo de servidor y el cliente omite cookies", async () => {
  const llamadas = [];
  const cliente = crearClienteCatalogosRegistroB2({ fetchImpl: async (url, opciones) => { llamadas.push([url, opciones]); return new Response(JSON.stringify({ data: { organismo_ref: "organismo:dipgra", entradas: [], cursor_siguiente: null, evidencia } }), { status: 200, headers: { "content-type": "application/json; charset=utf-8" } }); } });
  const pagina = await cliente.listar({ tipo: "regimen", estado: "publicada", limite: 50 });
  assert.equal(pagina.organismoRef, "organismo:dipgra"); assert.deepEqual(pagina.entradas, []);
  assert.match(llamadas[0][0], /^\/api\/vec\/personal\/catalogos-registro-empleado\?tipo=regimen&limite=50&estado=publicada$/u);
  assert.equal(llamadas[0][1].credentials, "omit");
  const denegado = crearClienteCatalogosRegistroB2({ fetchImpl: async () => new Response(JSON.stringify({ error: { codigo: "acceso_denegado" } }), { status: 403, headers: { "content-type": "application/json; charset=utf-8" } }) });
  await assert.rejects(denegado.listar({ tipo: "regimen" }), (error) => error instanceof ErrorCatalogosRegistroB2 && error.estado === 403);
  const sinOrganismo = crearClienteCatalogosRegistroB2({ fetchImpl: async () => new Response(JSON.stringify({ data: { organismo_ref: "", entradas: [], cursor_siguiente: null, evidencia } }), { status: 200, headers: { "content-type": "application/json; charset=utf-8" } }) });
  await assert.rejects(sinOrganismo.listar({ tipo: "regimen" }), (error) => error instanceof ErrorCatalogosRegistroB2 && error.codigo === "respuesta_incompatible");
});

test("POST publica sin organismo en el cuerpo y conserva recibo original en replay", async () => {
  const idempotencia = "123e4567-e89b-42d3-a456-426614174000"; const llamadas = [];
  const cuerpo = { operacion: "publicar", tipo: "regimen", ref: "regimen:uno", version: 1, revision: 1, denominacion: "Régimen sintético", huella_sha256: huella, vigente_desde: "2026-09-20", vigente_hasta: "" };
  const entrada = { organismo_ref: "organismo:dipgra", ...cuerpo, estado: "publicada" };
  const recibo = { decision_ref: "dec_original", auditoria_ref: "aud_original", consumo_huella_sha256: "a".repeat(64), registrado_en: "2026-09-25T10:00:00Z" };
  const acceso_actual = { decision_ref: "dec_replay", auditoria_ref: "aud_replay", consumo_huella_sha256: "b".repeat(64), registrado_en: "2026-09-25T10:01:00Z", estado_replay: "replay" };
  const cliente = crearClienteCatalogosRegistroB2({ fetchImpl: async (url, opciones) => { llamadas.push([url, opciones]); return new Response(JSON.stringify({ data: { entrada, recibo, acceso_actual } }), { status: 200, headers: { "content-type": "application/json; charset=utf-8" } }); } });
  const resultado = await cliente.cambiar(cuerpo, { claveIdempotencia: idempotencia });
  assert.deepEqual(resultado.recibo, recibo); assert.equal(resultado.accesoActual.estado_replay, "replay");
  assert.equal(llamadas[0][0], "/api/vec/personal/catalogos-registro-empleado");
  assert.equal(llamadas[0][1].headers["Idempotency-Key"], idempotencia);
  assert.equal(llamadas[0][1].credentials, "omit");
  assert.equal(Object.hasOwn(JSON.parse(llamadas[0][1].body), "organismo_ref"), false);
  assert.equal(Object.hasOwn(JSON.parse(llamadas[0][1].body), "acto_ref"), false);
  await assert.rejects(cliente.cambiar({ ...cuerpo, organismo_ref: "organismo:otro" }, { claveIdempotencia: idempotencia }), TypeError);
  await assert.rejects(cliente.cambiar({ ...cuerpo, acto_ref: "acto:libre" }, { claveIdempotencia: idempotencia }), TypeError);
});

test("las opciones para actuaciones proceden de las cuatro consultas publicadas", async () => {
  const consultas = []; const cliente = { async listar({ tipo, estado }) { consultas.push([tipo, estado]); return { organismoRef: "organismo:dipgra", entradas: [{ organismo_ref: "organismo:dipgra", tipo, ref: `${tipo}:uno`, version: 1, denominacion: `Nombre ${tipo}`, estado: "publicada" }], cursorSiguiente: null }; } };
  const opciones = await cargarOpcionesPublicadasCatalogoB2(cliente);
  assert.deepEqual(consultas.map((c) => c[0]), ["regimen", "modalidad", "situacion", "clase_servicio"]);
  assert.ok(consultas.every((c) => c[1] === "publicada"));
  assert.deepEqual(opciones.regimenes[0], { ref: "regimen:uno", version: 1, denominacion: "Nombre regimen", estado: "publicada" });
});

test("el organismo autorizado de GET habilita la publicación interna", async () => {
  const raiz = nodoFalso(); const cliente = { async listar() { return { organismoRef: "organismo:dipgra", entradas: [], cursorSiguiente: null, evidencia }; }, async cambiar() { throw Error("no debe enviarse"); } };
  montarCatalogosRegistroB2({ raiz, cliente }); await completar();
  assert.match(texto(raiz), /No hay entradas para este filtro/);
  const publicar = buscar(raiz, (n) => n.textContent === "Publicar entrada");
  assert.equal(publicar.disabled, false);
});

test("el panel consulta lista vacía, calcula la huella y revisa antes del POST", async () => {
  const raiz = nodoFalso(); const envios = [];
  const cliente = {
    async listar() { return { organismoRef: "organismo:dipgra", entradas: [], cursorSiguiente: null, evidencia }; },
    async cambiar(cuerpo, { claveIdempotencia }) { envios.push({ cuerpo, claveIdempotencia }); if (envios.length === 1) throw new ErrorCatalogosRegistroB2("resultado_incierto"); return { entrada: { ...cuerpo, organismo_ref: "organismo:dipgra", estado: "publicada" }, recibo: { registrado_en: "2026-09-25T10:00:00Z" }, accesoActual: { estado_replay: "replay" } }; },
  };
  montarCatalogosRegistroB2({ raiz, cliente }); await completar();
  assert.match(texto(raiz), /No hay entradas para este filtro/);
  buscar(raiz, (n) => n.textContent === "Publicar entrada").listeners.get("click")();
  const campos = Object.fromEntries(nodos(raiz).filter((n) => n.dataset.registroB2CatalogoCampo).map((n) => [n.dataset.registroB2CatalogoCampo, n]));
  for (const [clave, valor] of Object.entries({ ref: "regimen:uno", version: "1", denominacion: "Régimen sintético", vigente_desde: "2026-09-20" })) campos[clave].value = valor;
  buscar(raiz, (n) => n.tagName === "form").listeners.get("submit")({ preventDefault() {} }); await new Promise((resolver) => setTimeout(resolver, 30));
  assert.match(texto(raiz), /Confirmar cambio/); assert.equal(envios.length, 0);
  buscar(raiz, (n) => n.textContent === "Confirmar cambio").listeners.get("click")(); await completar();
  assert.match(texto(raiz), /No se pudo confirmar el cambio/);
  buscar(raiz, (n) => n.textContent === "Reintentar exactamente").listeners.get("click")(); await completar();
  assert.deepEqual(envios[0], envios[1]);
  assert.equal(envios[0].cuerpo.huella_sha256, huella);
  assert.equal(envios[0].cuerpo.organismo_ref, undefined);
  assert.match(envios[0].claveIdempotencia, /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/u);
});

test("retirar conserva la huella publicada y añade una revisión sin recalcularla", async () => {
  const raiz = nodoFalso(); let enviado;
  const entrada = { organismo_ref: "organismo:dipgra", tipo: "regimen", ref: "regimen:uno", version: 1, revision: 1, denominacion: "Régimen sintético", huella_sha256: huella, vigente_desde: "2026-09-20", vigente_hasta: "", estado: "publicada" };
  const cliente = { async listar() { return { organismoRef: "organismo:dipgra", entradas: [entrada], cursorSiguiente: null, evidencia }; }, async cambiar(cuerpo) { enviado = cuerpo; return { entrada: { ...entrada, revision: 2, estado: "retirada" }, recibo: { registrado_en: "2026-09-25T10:00:00Z" }, accesoActual: { estado_replay: "registrado" } }; } };
  montarCatalogosRegistroB2({ raiz, cliente }); await completar();
  buscar(raiz, (n) => n.textContent === "Retirar").listeners.get("click")();
  buscar(raiz, (n) => n.tagName === "form").listeners.get("submit")({ preventDefault() {} }); await completar();
  buscar(raiz, (n) => n.textContent === "Confirmar cambio").listeners.get("click")(); await completar();
  assert.equal(enviado.operacion, "retirar"); assert.equal(enviado.revision, 2);
  assert.equal(enviado.huella_sha256, huella); assert.equal(Object.hasOwn(enviado, "acto_ref"), false);
});
