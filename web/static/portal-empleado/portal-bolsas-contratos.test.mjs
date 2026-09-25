import test from "node:test";
import assert from "node:assert/strict";
import {
  cargarContratosFicha,
  consultarContratosParticipacion,
  ESQUEMA_CONTRATOS,
  manejarClickContratos,
  MENSAJES_CONTRATOS_ES,
  renderizarContratosParticipacion,
  rutaContratosParticipacion,
  traducirContratos,
} from "./portal-bolsas-contratos.js";

const escaparHTML = (v) => String(v ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;");

const item = Object.freeze({
  evento_ref: "evento:ct:contrato-bolsa:" + "a".repeat(64),
  tipo: "incorporacion",
  inicio: "2027-01-04T00:00:00Z",
  fin_previsto: "2027-03-31T00:00:00Z",
  modalidad_clave: "sustitucion",
  categoria_ref: "categoria:desarrollo:c2",
  causa_clave: "necesidad_temporal",
  expediente_ref: "expediente:ct:1",
  ocurrido_en: "2027-01-02T09:00:01Z",
});

function respuesta(status, cuerpo) {
  return { ok: status >= 200 && status < 300, status, json: async () => cuerpo };
}

test("B13 construye la ruta de la ficha con segmentos codificados", () => {
  assert.equal(rutaContratosParticipacion("bolsa:uno", "participacion x"),
    "/api/vec/bolsa/bolsas/bolsa:uno/candidatos/participacion%20x/contratos");
});

test("B13 consulta por GET sin cuerpo, sin cookies ajenas ni caché", async () => {
  let llamada;
  const res = await consultarContratosParticipacion("b", "p", {
    fetchImpl: async (url, opciones) => { llamada = { url, opciones }; return respuesta(200, { data: { esquema: ESQUEMA_CONTRATOS, items: [item] } }); },
  });
  assert.equal(res.ok, true);
  assert.deepEqual(res.datos, [item]);
  assert.equal(llamada.opciones.method, "GET");
  assert.equal(llamada.opciones.credentials, "same-origin");
  assert.equal(llamada.opciones.cache, "no-store");
  assert.equal(llamada.opciones.body, undefined);
});

test("B13 rechaza respuestas que no respetan el contrato", async () => {
  for (const cuerpo of [
    { data: { esquema: "otro", items: [] } },
    { data: { esquema: ESQUEMA_CONTRATOS, items: [{ ...item, inicio: "2027-01-04" }] } },
    { data: { esquema: ESQUEMA_CONTRATOS, items: [{ ...item, tipo: "<script>" }] } },
    { data: { esquema: ESQUEMA_CONTRATOS, items: [{ ...item, causa_clave: "Mayúsculas" }] } },
    { data: { esquema: ESQUEMA_CONTRATOS, items: {} } },
  ]) {
    const res = await consultarContratosParticipacion("b", "p", { fetchImpl: async () => respuesta(200, cuerpo) });
    assert.equal(res.ok, false, JSON.stringify(cuerpo));
    assert.equal(res.mensaje, MENSAJES_CONTRATOS_ES.error_contrato);
  }
});

test("B13 traduce los errores HTTP y de red sin detalles internos", async () => {
  for (const [status, clave] of [[403, "error_403"], [404, "error_404"], [503, "error_503"]]) {
    const res = await consultarContratosParticipacion("b", "p", { fetchImpl: async () => respuesta(status, { error: { codigo: "x" } }) });
    assert.equal(res.mensaje, MENSAJES_CONTRATOS_ES[clave]);
  }
  const red = await consultarContratosParticipacion("b", "p", { fetchImpl: async () => { throw new TypeError("fallo"); } });
  assert.equal(red.mensaje, MENSAJES_CONTRATOS_ES.error_red);
  assert.equal((await consultarContratosParticipacion("b", "p", { fetchImpl: async () => respuesta(500, {}) })).mensaje,
    traducirContratos("error_http", { estado: 500 }));
});

test("B13 el traductor es estricto con las claves", () => {
  assert.throws(() => traducirContratos("no_existe"));
  assert.equal(traducirContratos("mostrando", { desde: 1, hasta: 2, total: 3 }), "Mostrando 1 a 2 de 3");
});

test("B13 renderiza estados, tabla accesible, fechas locales y escapa datos", () => {
  assert.match(renderizarContratosParticipacion({ estado: {}, escaparHTML }), /aria-busy="true"/);
  assert.match(renderizarContratosParticipacion({ estado: { carga: "listo", items: [] }, escaparHTML }), new RegExp(MENSAJES_CONTRATOS_ES.vacio));
  const error = renderizarContratosParticipacion({ estado: { carga: "error", error: "<b>x</b>" }, escaparHTML });
  assert.match(error, /role="alert"/);
  assert.match(error, /&lt;b&gt;x&lt;\/b&gt;/);
  assert.match(error, /data-b13-accion="reintentar"/);
  const html = renderizarContratosParticipacion({ estado: { carga: "listo", items: [{ ...item, categoria_ref: "cat:<x>" }] }, escaparHTML });
  assert.match(html, /<caption>Contratos de la participación<\/caption>/);
  assert.match(html, /<th scope="col">Periodo<\/th>/);
  assert.match(html, /Incorporación/);
  assert.match(html, /04\/01\/2027 – 31\/03\/2027 \(prevista\)/);
  assert.match(html, /Sustitucion/);
  assert.match(html, /Necesidad temporal/);
  assert.match(html, /cat:&lt;x&gt;/);
  assert.doesNotMatch(html, /expediente:ct:1/);
  const sinFin = renderizarContratosParticipacion({ estado: { carga: "listo", items: [{ ...item, fin_previsto: null, modalidad_clave: "" }] }, escaparHTML });
  assert.match(sinFin, /sin fecha de fin/);
  assert.match(sinFin, /<td>—<\/td>/);
});

test("B13 pagina de seis en seis", () => {
  const items = Array.from({ length: 8 }, (_, i) => ({ ...item, evento_ref: item.evento_ref.slice(0, -1) + i }));
  const html = renderizarContratosParticipacion({ estado: { carga: "listo", items, pagina: 1 }, escaparHTML });
  assert.match(html, /Mostrando 7 a 8 de 8/);
  assert.match(html, /aria-label="Paginación del histórico de contratos"/);
  assert.equal((html.match(/<tr><td>/g) || []).length, 2);
});

test("B13 carga solo para la ficha vigente y descarta respuestas tardías", async () => {
  let renders = 0;
  const modal = { candidato: { participacion_ref: "p1" } };
  const estado = { bolsaSeleccionada: "b1", modalFicha: modal };
  let recibido;
  await cargarContratosFicha(modal, { estado, renderizar: () => { renders += 1; }, consultar: async (b, p, { signal }) => { recibido = { b, p, signal }; return { ok: true, datos: [item] }; } });
  assert.deepEqual([recibido.b, recibido.p], ["b1", "p1"]);
  assert.equal(modal.contratosB13.carga, "listo");
  assert.equal(renders, 2);
  const otra = { candidato: { participacion_ref: "p2" } };
  const estadoCambiado = { bolsaSeleccionada: "b1", modalFicha: modal };
  await cargarContratosFicha(otra, { estado: estadoCambiado, renderizar: () => {}, consultar: async () => ({ ok: true, datos: [item] }) });
  assert.equal(otra.contratosB13.carga, "cargando");
});

test("B13 atiende reintento y paginación sin tocar otros clics", async () => {
  let renders = 0;
  const estado = { bolsaSeleccionada: "b", modalFicha: { candidato: { participacion_ref: "p" }, contratosB13: { carga: "listo", items: [], pagina: 0 } } };
  const evento = (accion, pagina) => ({ preventDefault() {}, target: { closest: () => ({ dataset: { b13Accion: accion, pagina } }) } });
  assert.equal(manejarClickContratos(evento("pagina", "1"), { estado, renderizar: () => { renders += 1; } }), true);
  assert.equal(estado.modalFicha.contratosB13.pagina, 1);
  let consultas = 0;
  manejarClickContratos(evento("reintentar"), { estado, renderizar: () => {}, consultar: async () => { consultas += 1; return { ok: true, datos: [] }; } });
  await new Promise((r) => setTimeout(r, 0));
  assert.equal(consultas, 1);
  assert.equal(manejarClickContratos({ target: { closest: () => null } }, { estado, renderizar: () => {} }), false);
});
