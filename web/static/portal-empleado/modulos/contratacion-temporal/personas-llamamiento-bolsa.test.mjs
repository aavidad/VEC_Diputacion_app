import test from "node:test";
import assert from "node:assert/strict";
import { consultarPersonasLlamamientos, textoPersonasLlamamiento, validarPaginaPersonasBolsa } from "./personas-llamamiento-bolsa.js";
import { renderizarResultadoBolsa } from "./resultado-bolsa.js";
import { crearTraductorExpedientesContratacion } from "./i18n-expedientes.js";

const t = crearTraductorExpedientesContratacion();
const BOLSA = "bolsa:administrativo:prueba";
const LLAMAMIENTO_A = `llamamiento:${"a".repeat(64)}`;
const LLAMAMIENTO_B = `llamamiento:${"b".repeat(64)}`;

function pagina(candidatos, contactos, hayMas = false, bolsa = BOLSA) {
  return { data: { bolsa: { bolsa_ref: bolsa }, candidatos, contactos, hay_mas: hayMas, cursor_siguiente: null } };
}
const persona = (n, nombre) => ({ participacion_ref: `participacion:${n}`, nombre_visible: nombre, orden: n });
const contacto = (n, llamamiento) => ({ participacion_ref: `participacion:${n}`, llamamiento_ref: llamamiento });
// La Bolsa devuelve todos los contactos de la bolsa, pero solo a las personas del llamamiento pedido.
const CONTACTOS = [contacto(5, LLAMAMIENTO_A), contacto(1, LLAMAMIENTO_B), contacto(2, LLAMAMIENTO_B),
  { participacion_ref: "participacion:1", llamamiento_ref: null }];

function fetchFalso(porLlamamiento) {
  const urls = [];
  const fetchImpl = async (url) => {
    urls.push(url);
    const llamamiento = new URL(url, "https://vec.invalid").searchParams.get("llamamiento");
    const respuesta = porLlamamiento[llamamiento];
    return typeof respuesta === "number" ? { ok: false, status: respuesta } : { ok: true, json: async () => respuesta };
  };
  return { urls, fetchImpl };
}

test("cada llamamiento muestra a la persona que la Bolsa registró en él, no a otra", async () => {
  const { fetchImpl, urls } = fetchFalso({
    [LLAMAMIENTO_A]: pagina([persona(5, "Antonio Reyes Álvarez")], CONTACTOS),
    [LLAMAMIENTO_B]: pagina([persona(1, "Isabel Ramírez Vargas"), persona(2, "David Jiménez Navarro")], CONTACTOS) });
  const mapa = await consultarPersonasLlamamientos([
    { bolsa_ref: BOLSA, llamamiento_ref: LLAMAMIENTO_A }, { bolsa_ref: BOLSA, llamamiento_ref: LLAMAMIENTO_B },
    { bolsa_ref: BOLSA, llamamiento_ref: LLAMAMIENTO_A }], { fetchImpl });
  assert.deepEqual(mapa.get(LLAMAMIENTO_A), ["Antonio Reyes Álvarez"]);
  assert.deepEqual(mapa.get(LLAMAMIENTO_B), ["Isabel Ramírez Vargas", "David Jiménez Navarro"]);
  // Una lectura filtrada por llamamiento, no la bolsa entera.
  assert.deepEqual(urls.sort(), [LLAMAMIENTO_A, LLAMAMIENTO_B].map((l) =>
    `/api/vec/bolsa/bolsas/bolsa:administrativo:prueba/candidatos?limite=100&llamamiento=${encodeURIComponent(l)}`).sort());
});

test("no pone el nombre de quien no consta contactado en ese llamamiento", async () => {
  // Isabel no tiene contacto en el llamamiento A: la respuesta no es coherente.
  assert.throws(() => validarPaginaPersonasBolsa(pagina([persona(1, "Isabel Ramírez Vargas")], CONTACTOS), BOLSA, LLAMAMIENTO_A));
  assert.throws(() => validarPaginaPersonasBolsa(pagina([], CONTACTOS, false, "bolsa:otra"), BOLSA, LLAMAMIENTO_A));
  assert.throws(() => validarPaginaPersonasBolsa(pagina([], CONTACTOS, true), BOLSA, LLAMAMIENTO_A));
});

test("un llamamiento que falla se queda sin nombre y no impide los demás", async () => {
  const { fetchImpl } = fetchFalso({ [LLAMAMIENTO_A]: 403,
    [LLAMAMIENTO_B]: pagina([persona(1, "Isabel Ramírez Vargas")], CONTACTOS) });
  const mapa = await consultarPersonasLlamamientos([
    { bolsa_ref: BOLSA, llamamiento_ref: LLAMAMIENTO_A }, { bolsa_ref: BOLSA, llamamiento_ref: LLAMAMIENTO_B }], { fetchImpl });
  assert.equal(mapa.has(LLAMAMIENTO_A), false);
  assert.deepEqual(mapa.get(LLAMAMIENTO_B), ["Isabel Ramírez Vargas"]);
  const control = new AbortController(); control.abort();
  await assert.rejects(consultarPersonasLlamamientos([{ bolsa_ref: BOLSA, llamamiento_ref: LLAMAMIENTO_B }],
    { fetchImpl, signal: control.signal }));
});

test("el selector de vincular dice a quién se llamó y cuándo", () => {
  const expediente = { expediente_ref: "expediente:prueba", version: 6, resultado_bolsa: { vinculos: [],
    emisiones_vinculables: [
      { bolsa_ref: BOLSA, llamamiento_ref: LLAMAMIENTO_A, recibo_emision_ref: "recibo:llamamiento:a", referencia_visible: "2026/93001", emitido_en: "2026-10-20T01:00:00Z" },
      { bolsa_ref: BOLSA, llamamiento_ref: LLAMAMIENTO_B, recibo_emision_ref: "recibo:llamamiento:b", referencia_visible: "2026/93001", emitido_en: "2026-10-20T01:08:00Z" }],
    siguiente_cursor: null, total_vinculos: 0, personas_solicitadas: 1, aceptaciones_firmes: 0 } };
  const personas = new Map([[LLAMAMIENTO_A, ["Antonio Reyes Álvarez"]], [LLAMAMIENTO_B, ["Isabel Ramírez Vargas", "David Jiménez Navarro"]]]);
  const html = renderizarResultadoBolsa(expediente, t, "es-ES", "Europe/Madrid", { personas });
  assert.match(html, /<option value="0">Antonio Reyes Álvarez · 20 oct 2026, 3:00 · 2026\/93001<\/option>/u);
  assert.match(html, /<option value="1">Isabel Ramírez Vargas y 1 más · 20 oct 2026, 3:08 · 2026\/93001<\/option>/u);
  // Sin la lectura de Bolsa queda como antes: referencia y fecha.
  assert.match(renderizarResultadoBolsa(expediente, t, "es-ES", "Europe/Madrid"), /<option value="0">2026\/93001 · 20 oct 2026, 3:00<\/option>/u);
  assert.equal(textoPersonasLlamamiento([], t), "");
});
