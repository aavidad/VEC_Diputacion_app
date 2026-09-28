import assert from "node:assert/strict";
import test from "node:test";
import { montarVistaAuditoria, renderizarVistaAuditoria, validarRespuestaAuditoria } from "./vista.js";

const registro = {
  id: "aud_1", modulo_id: "personal", accion: "relacion.actualizada", actor_ref: "per_1",
  ocurrido_en: "2026-09-28T08:00:00Z", resultado: "confirmado", expediente_ref: "exp_1",
  recibo_ref: "rec_1", antes_sha256: "a".repeat(64), despues_sha256: "b".repeat(64),
  motivo: "rectificacion", fuente: "Personal", datos_disponibles: true,
  antes: { unidad: "A" }, despues: { unidad: "B" },
};
const opciones = { finalidad_ref: "fin_1", motivo_ref: "catalogo:v1:revision", permiso_requerido: "auditoria.consultar", es_ejemplo: true, fuentes: ["ct", "bolsa"] };
const esperar = () => new Promise((resolver) => setImmediate(resolver));
function raizFalsa() {
  const eventos = {};
  return { eventos, raiz: {
    innerHTML: "", replaceChildren() { this.innerHTML = ""; }, querySelector() { return null; },
    addEventListener(k, fn) { eventos[k] = fn; }, removeEventListener(k) { delete eventos[k]; },
  } };
}

test("sin expediente de navegación la consulta permanece cerrada y no ofrece vínculo ni actor libres", () => {
  const html = renderizarVistaAuditoria();
  assert.match(html, /Abra Auditoría desde un expediente o participación autorizados/u);
  assert.match(html, /id="auditoria-ayuda"[^>]+hidden/u);
  assert.match(html, /name="desde"[^>]+disabled/u);
  assert.doesNotMatch(html, /name="expediente_ref"|name="actor_ref"|<table/u);
});

test("solo muestra expediente exacto escapado y aviso de configuración de ejemplo", () => {
  const html = renderizarVistaAuditoria({ estado: "esperando", habilitada: true,
    expedienteRef: 'exp_1"><script>', fuenteContexto: "ct", ejemplo: true });
  assert.match(html, /Número no disponible/u);
  assert.match(html, /Muestra ficticia/u);
  assert.doesNotMatch(html, /exp_1&quot;&gt;&lt;script&gt;/u);
  assert.doesNotMatch(html, /<script>|name="expediente_ref"|name="actor_ref"/u);
  const bolsa = renderizarVistaAuditoria({ estado: "esperando", habilitada: true,
    expedienteRef: "participacion_1", fuenteContexto: "bolsa" });
  assert.match(bolsa, /<strong>Participación:<\/strong> Número no disponible/u);
  assert.doesNotMatch(bolsa, /name="fuente"/u);
});

test("la muestra enseña nombres y números ficticios; las referencias y huellas quedan plegadas", () => {
  const respuesta = validarRespuestaAuditoria({ registros: [{ ...registro, antes: { unidad: "<script>" } }], siguiente_cursor: "" });
  const html = renderizarVistaAuditoria({ estado: "disponible", habilitada: true, ejemplo: true, registros: respuesta.registros });
  assert.match(html, /Carmen Molina \(ficticio\).*Actualizó la relación de servicio.*EXP-2026-001 \(ficticio\)/su);
  assert.match(html, /<details><summary>Ver detalle técnico<\/summary>[\s\S]*per_1[\s\S]*relacion\.actualizada[\s\S]*Huella SHA-256/u);
  assert.match(html, /datetime="2026-09-28T08:00:00Z"/u);
  assert.match(html, /&lt;script&gt;/u);
  assert.doesNotMatch(html, /<script>/u);
  const real = renderizarVistaAuditoria({ estado: "disponible", habilitada: true, registros: respuesta.registros });
  assert.match(real, /Nombre no disponible.*Número no disponible/su);
  assert.doesNotMatch(real.split("<details>")[0], /per_1|exp_1|relacion\.actualizada/u);
  assert.throws(() => validarRespuestaAuditoria({ registros: [{ ...registro, despues: { secreto: {} } }], siguiente_cursor: "" }), /proyección/u);
});

test("paginación se mantiene y denegación no presenta datos previos", () => {
  const pagina = renderizarVistaAuditoria({ estado: "vacio", habilitada: true, pagina: 2, puedeAnterior: true, siguienteCursor: "cursor_3" });
  assert.match(pagina, /Página 2|data-auditoria-siguiente/u);
  const denegado = renderizarVistaAuditoria({ estado: "denegado", habilitada: true, registros: [registro] });
  assert.match(denegado, /Acceso denegado/u);
  assert.doesNotMatch(denegado, /per_1|<table/u);
});

test("GET de opciones antecede al POST, que usa expediente de navegación y refs recibidas", async () => {
  const { eventos, raiz } = raizFalsa();
  const peticiones = [];
  const fuente = {
    obtenerOpciones: async () => opciones,
    consultar: async (entrada) => { peticiones.push(entrada); return { registros: [registro], siguiente_cursor: "" }; },
  };
  const original = globalThis.FormData;
  globalThis.FormData = class { get(k) { return { desde: "2026-09-28T08:00", hasta: "2026-09-28T09:00" }[k]; } };
  try {
    const vista = montarVistaAuditoria({ raiz, fuente, expedienteRef: "exp_1", fuenteContexto: "ct" });
    assert.match(raiz.innerHTML, /Comprobando opciones/u);
    await esperar();
    assert.match(raiz.innerHTML, /Muestra ficticia/u);
    assert.equal(peticiones.length, 1, "consulta automática al abrir");
    assert.equal(Date.parse(peticiones[0].hasta) - Date.parse(peticiones[0].desde), 30 * 86400000);
    assert.match(raiz.innerHTML, /Carmen Molina \(ficticio\)/u);
    eventos.submit({ target: { matches: () => true }, preventDefault() {} });
    await esperar();
    assert.equal(peticiones.length, 2);
    const peticion = peticiones[1];
    assert.equal(peticion.expediente_ref, "exp_1");
    assert.equal(peticion.fuente, "ct");
    assert.equal(peticion.actor_ref, "");
    assert.equal(peticion.finalidad_ref, opciones.finalidad_ref);
    assert.equal(peticion.motivo_ref, opciones.motivo_ref);
    assert.match(raiz.innerHTML, /Ver detalle técnico/u);
    vista.desmontar();
  } finally { globalThis.FormData = original; }
});

test("paginación y reintento conservan el intervalo ligado al cursor aunque avance el reloj", async () => {
  const { eventos, raiz } = raizFalsa();
  const peticiones = [];
  let fallar = false;
  const fuente = { obtenerOpciones: async () => opciones,
    consultar: async (entrada) => {
      peticiones.push(entrada);
      if (fallar) { fallar = false; throw Object.assign(Error("temporal"), { codigo: "consulta_fallida" }); }
      return { registros: [registro], siguiente_cursor: entrada.cursor ? "" : "cursor_2" };
    } };
  const reloj = Date.now;
  try {
    Date.now = () => Date.parse("2026-09-28T12:00:00Z");
    const vista = montarVistaAuditoria({ raiz, fuente, expedienteRef: "exp_1", fuenteContexto: "ct" });
    await esperar();
    Date.now = () => Date.parse("2026-09-28T12:00:30Z");
    eventos.click({ target: { closest: (selector) => selector === "[data-auditoria-siguiente]" ? {} : null } });
    await esperar();
    assert.equal(peticiones[1].cursor, "cursor_2");
    assert.deepEqual([peticiones[1].desde, peticiones[1].hasta], [peticiones[0].desde, peticiones[0].hasta]);
    fallar = true;
    eventos.click({ target: { closest: (selector) => selector === "[data-auditoria-anterior]" ? {} : null } });
    await esperar();
    assert.match(raiz.innerHTML, /No se pudo completar/u);
    Date.now = () => Date.parse("2026-09-28T12:01:00Z");
    eventos.click({ target: { closest: (selector) => selector === "[data-auditoria-reintentar]" ? {} : null } });
    await esperar();
    assert.deepEqual([peticiones.at(-1).desde, peticiones.at(-1).hasta], [peticiones[0].desde, peticiones[0].hasta]);
    vista.desmontar();
  } finally { Date.now = reloj; }
});

test("sin expediente no pide opciones, y 403 en opciones no expone datos", async () => {
  const { raiz } = raizFalsa();
  let llamadas = 0;
  const fuente = { obtenerOpciones: async () => { ++llamadas; throw Object.assign(Error("secreto"), { codigo: "denegado" }); },
    consultar: async () => { throw Error("no debe llamarse"); } };
  const cerrada = montarVistaAuditoria({ raiz, fuente });
  assert.equal(llamadas, 0);
  cerrada.desmontar();
  const denegada = montarVistaAuditoria({ raiz, fuente, expedienteRef: "exp_1", fuenteContexto: "ct" });
  await esperar();
  assert.equal(llamadas, 1);
  assert.match(raiz.innerHTML, /Acceso denegado/u);
  assert.doesNotMatch(raiz.innerHTML, /secreto|<table/u);
  denegada.desmontar();
});

test("sin fuente de navegación o sin fuente ofrecida por GET no consulta", async () => {
  const { raiz } = raizFalsa();
  let get = 0, post = 0;
  const fuente = { obtenerOpciones: async () => { ++get; return { ...opciones, fuentes: ["ct"] }; },
    consultar: async () => { ++post; throw Error("no debe llamarse"); } };
  const sinFuente = montarVistaAuditoria({ raiz, fuente, expedienteRef: "participacion_1" });
  assert.equal(get, 0);
  sinFuente.desmontar();
  const noOfrecida = montarVistaAuditoria({ raiz, fuente, expedienteRef: "participacion_1", fuenteContexto: "bolsa" });
  await esperar();
  assert.equal(get, 1);
  assert.equal(post, 0);
  assert.match(raiz.innerHTML, /Acceso denegado/u);
  assert.doesNotMatch(raiz.innerHTML, /name="fuente"/u);
  noOfrecida.desmontar();
});

test("filtros opcionales usan hora de Madrid y rechazan intervalos mayores de 31 días", async () => {
  const { eventos, raiz } = raizFalsa();
  const peticiones = [];
  const fuente = { obtenerOpciones: async () => opciones,
    consultar: async (entrada) => { peticiones.push(entrada); return { registros: [], siguiente_cursor: "" }; } };
  const original = globalThis.FormData;
  let valores = { desde: "2026-09-28T10:00", hasta: "2026-09-29T10:00" };
  globalThis.FormData = class { get(k) { return valores[k]; } };
  try {
    const vista = montarVistaAuditoria({ raiz, fuente, expedienteRef: "exp_1", fuenteContexto: "ct" });
    await esperar();
    eventos.submit({ target: { matches: () => true }, preventDefault() {} });
    await esperar();
    assert.equal(peticiones.at(-1).desde, "2026-09-28T08:00:00.000Z");
    valores = { desde: "2026-08-01T10:00", hasta: "2026-09-28T10:00" };
    eventos.submit({ target: { matches: () => true }, preventDefault() {} });
    await esperar();
    assert.match(raiz.innerHTML, /Revise las fechas/u);
    assert.equal(peticiones.length, 2);
    vista.desmontar();
  } finally { globalThis.FormData = original; }
});

test("Madrid acepta horas válidas a ambos lados del horario de verano y rechaza el hueco", async () => {
  const { eventos, raiz } = raizFalsa();
  const peticiones = [];
  const fuente = { obtenerOpciones: async () => opciones,
    consultar: async (entrada) => { peticiones.push(entrada); return { registros: [], siguiente_cursor: "" }; } };
  const original = globalThis.FormData;
  let valores = { desde: "2026-03-29T01:30", hasta: "2026-03-29T04:30" };
  globalThis.FormData = class { get(k) { return valores[k]; } };
  try {
    const vista = montarVistaAuditoria({ raiz, fuente, expedienteRef: "exp_1", fuenteContexto: "ct" });
    await esperar();
    for (const [desde, hasta, utc] of [
      ["2026-03-29T01:30", "2026-03-29T04:30", "2026-03-29T00:30:00.000Z"],
      ["2026-10-25T01:30", "2026-10-25T04:30", "2026-10-24T23:30:00.000Z"],
      ["2026-10-25T02:30", "2026-10-25T04:30", "2026-10-25T00:30:00.000Z"],
    ]) {
      valores = { desde, hasta };
      eventos.submit({ target: { matches: () => true }, preventDefault() {} });
      await esperar();
      assert.equal(peticiones.at(-1).desde, utc);
    }
    const cantidad = peticiones.length;
    valores = { desde: "2026-03-29T02:30", hasta: "2026-03-29T04:30" };
    eventos.submit({ target: { matches: () => true }, preventDefault() {} });
    await esperar();
    assert.equal(peticiones.length, cantidad);
    assert.match(raiz.innerHTML, /Revise las fechas/u);
    vista.desmontar();
  } finally { globalThis.FormData = original; }
});

test("cambiar fecha cancela respuesta tardía y limpia el resultado", async () => {
  const { eventos, raiz } = raizFalsa();
  let resolver, signal;
  let focoRestaurado = false;
  raiz.querySelector = () => ({ focus() { focoRestaurado = true; } });
  const fuente = { obtenerOpciones: async () => opciones,
    consultar: (_entrada, o) => { signal = o.signal; return new Promise((r) => { resolver = r; }); } };
  const original = globalThis.FormData;
  globalThis.FormData = class { get(k) { return { desde: "2026-09-28T08:00", hasta: "2026-09-28T09:00" }[k]; } };
  try {
    const vista = montarVistaAuditoria({ raiz, fuente, expedienteRef: "exp_1", fuenteContexto: "ct" });
    await esperar();
    eventos.submit({ target: { matches: () => true }, preventDefault() {} });
    eventos.input({ target: { matches: () => true, name: "desde", value: "2026-09-28T08:30", selectionStart: null, selectionEnd: null } });
    assert.equal(signal.aborted, true);
    assert.match(raiz.innerHTML, /role="status" aria-live="polite">Pulse Consultar para aplicar los filtros/u);
    assert.equal(focoRestaurado, true);
    resolver({ registros: [registro], siguiente_cursor: "" });
    await esperar();
    assert.doesNotMatch(raiz.innerHTML, /per_1/u);
    vista.desmontar();
  } finally { globalThis.FormData = original; }
});
