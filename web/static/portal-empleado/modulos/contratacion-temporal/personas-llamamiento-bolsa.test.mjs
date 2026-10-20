import test from "node:test";
import assert from "node:assert/strict";
import { consultarPersonasLlamamientos, textoPersonasLlamamiento, validarPaginaPersonasBolsa } from "./personas-llamamiento-bolsa.js";
import { renderizarResultadoBolsa } from "./resultado-bolsa.js";
import { crearTraductorExpedientesContratacion } from "./i18n-expedientes.js";

const t = crearTraductorExpedientesContratacion();
const BOLSA = "bolsa:administrativo:prueba";
const LLAMAMIENTO_A = `llamamiento:${"a".repeat(64)}`;
const LLAMAMIENTO_B = `llamamiento:${"b".repeat(64)}`;

function pagina(candidatos, contactos, cursor = null) {
  return { data: { bolsa: { bolsa_ref: BOLSA }, candidatos, contactos, hay_mas: cursor !== null, cursor_siguiente: cursor } };
}
const persona = (n, nombre) => ({ participacion_ref: `participacion:${n}`, nombre_visible: nombre, orden: n });
const contacto = (n, llamamiento) => ({ participacion_ref: `participacion:${n}`, llamamiento_ref: llamamiento });

function fetchFalso(paginas) {
  const urls = [];
  const fetchImpl = async (url) => {
    urls.push(url);
    const cursor = new URL(url, "https://vec.invalid").searchParams.get("cursor") ?? "";
    return { ok: true, json: async () => paginas[cursor] };
  };
  return { urls, fetchImpl };
}

test("cada llamamiento muestra a la persona que la Bolsa registró en él, no a otra", async () => {
  // Dos llamamientos del mismo expediente: el primero a la persona del orden 5.
  const { fetchImpl, urls } = fetchFalso({ "": pagina(
    [persona(1, "Isabel Ramírez Vargas"), persona(5, "Antonio Reyes Álvarez")],
    [contacto(5, LLAMAMIENTO_A), contacto(1, LLAMAMIENTO_B), { participacion_ref: "participacion:1", llamamiento_ref: null }]) });
  const mapa = await consultarPersonasLlamamientos([
    { bolsa_ref: BOLSA, llamamiento_ref: LLAMAMIENTO_A }, { bolsa_ref: BOLSA, llamamiento_ref: LLAMAMIENTO_B }], { fetchImpl });
  assert.deepEqual(mapa.get(LLAMAMIENTO_A), ["Antonio Reyes Álvarez"]);
  assert.deepEqual(mapa.get(LLAMAMIENTO_B), ["Isabel Ramírez Vargas"]);
  assert.equal(urls.length, 1);
  assert.match(urls[0], /^\/api\/vec\/bolsa\/bolsas\/bolsa:administrativo:prueba\/candidatos\?limite=100$/u);
});

test("pide más páginas solo mientras falta un nombre y no inventa el que no encuentra", async () => {
  const { fetchImpl, urls } = fetchFalso({
    "": pagina([persona(1, "Isabel Ramírez Vargas")], [contacto(150, LLAMAMIENTO_A), contacto(1, LLAMAMIENTO_B), contacto(999, LLAMAMIENTO_B)], "participacion:1"),
    "participacion:1": pagina([persona(150, "Laura Martínez Díaz")], [], null),
  });
  const mapa = await consultarPersonasLlamamientos([
    { bolsa_ref: BOLSA, llamamiento_ref: LLAMAMIENTO_A }, { bolsa_ref: BOLSA, llamamiento_ref: LLAMAMIENTO_B }], { fetchImpl });
  assert.equal(urls.length, 2);
  assert.deepEqual(mapa.get(LLAMAMIENTO_A), ["Laura Martínez Díaz"]);
  // Falta la persona 999: mejor sin nombre que con uno incompleto.
  assert.equal(mapa.has(LLAMAMIENTO_B), false);
});

test("una respuesta de otra bolsa o sin permiso no aporta nombres", async () => {
  assert.throws(() => validarPaginaPersonasBolsa({ data: { ...pagina([], []).data, bolsa: { bolsa_ref: "bolsa:otra" } } }, BOLSA));
  await assert.rejects(consultarPersonasLlamamientos([{ bolsa_ref: BOLSA, llamamiento_ref: LLAMAMIENTO_A }],
    { fetchImpl: async () => ({ ok: false, status: 403 }) }), (error) => error.estado === 403);
});

test("el selector de vincular dice a quién se llamó y cuándo", () => {
  const expediente = { expediente_ref: "expediente:prueba", version: 6, resultado_bolsa: { vinculos: [],
    emisiones_vinculables: [
      { bolsa_ref: BOLSA, llamamiento_ref: LLAMAMIENTO_A, recibo_emision_ref: "recibo:llamamiento:a", referencia_visible: "2026/93001", emitido_en: "2026-10-20T01:00:00Z" },
      { bolsa_ref: BOLSA, llamamiento_ref: LLAMAMIENTO_B, recibo_emision_ref: "recibo:llamamiento:b", referencia_visible: "2026/93001", emitido_en: "2026-10-20T01:08:00Z" }],
    siguiente_cursor: null, total_vinculos: 0, personas_solicitadas: 1, aceptaciones_firmes: 0 } };
  const personas = new Map([[LLAMAMIENTO_A, ["Antonio Reyes Álvarez"]], [LLAMAMIENTO_B, ["Isabel Ramírez Vargas", "David Jiménez Navarro"]]]);
  const html = renderizarResultadoBolsa(expediente, t, "es-ES", "Europe/Madrid", { personas });
  assert.match(html, /<option value="0">2026\/93001 · Antonio Reyes Álvarez · 20 oct 2026, 3:00<\/option>/u);
  assert.match(html, /<option value="1">2026\/93001 · Isabel Ramírez Vargas y 1 más · 20 oct 2026, 3:08<\/option>/u);
  // Sin la lectura de Bolsa queda como antes: referencia y fecha.
  assert.match(renderizarResultadoBolsa(expediente, t, "es-ES", "Europe/Madrid"), /<option value="0">2026\/93001 · 20 oct 2026, 3:00<\/option>/u);
  assert.equal(textoPersonasLlamamiento([], t), "");
});
