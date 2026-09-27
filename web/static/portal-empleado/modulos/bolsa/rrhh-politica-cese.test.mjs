import assert from "node:assert/strict";
import test from "node:test";
import { crearClientePoliticaCeseRRHH, RUTA_POLITICA_CESE, validarPoliticaCeseRRHH } from "./rrhh-politica-cese-api.js";
import { montarVistaPoliticaCeseRRHH, renderizarVistaPoliticaCeseRRHH } from "./rrhh-politica-cese-vista.js";

const politica = Object.freeze({
  version: 2, catalogo_ref: "politica:bolsa:cese:2", catalogo_sha256: "a".repeat(64),
  mapeo: { "interinidad": "general", "interinidad|acumulacion_tareas": "acumulacion_tareas", relevo: "general" },
  meses_general: 5, meses_acumulacion: 9,
  computo: "fecha_cese_meses_calendario_ajuste_fin_mes", estado: "ejemplo_sintetico",
  publicada_en: "2026-09-28T10:00:00Z",
});
const sobre = (valor = politica) => ({ data: { esquema: "vec.bolsa.rrhh.politica_cese.v1", politica_cese: valor } });

test("consulta la política B45 por GET sin referencias ni autoridad en URL", async () => {
  const llamadas = [];
  const cliente = crearClientePoliticaCeseRRHH({ fetchImpl: async (ruta, opciones) => {
    llamadas.push([ruta, opciones]);
    return new Response(JSON.stringify(sobre()), { status: 200, headers: { "content-type": "application/json" } });
  } });
  const leida = await cliente.consultar();
  assert.equal(leida.version, 2);
  assert.equal(leida.meses_acumulacion, 9);
  assert.equal(llamadas[0][0], RUTA_POLITICA_CESE);
  assert.equal(llamadas[0][1].credentials, "same-origin");
  assert.equal(llamadas[0][1].cache, "no-store");
  assert.equal(llamadas[0][1].method, "GET");
  assert.equal(llamadas[0][1].body, undefined);
});

test("falla cerrado ante mapeo o régimen no reconocidos y denegación V3", async () => {
  assert.throws(() => validarPoliticaCeseRRHH(sobre({ ...politica, mapeo: { interinidad: "otro" } })));
  assert.throws(() => validarPoliticaCeseRRHH(sobre({ ...politica, estado: "aprobada" })));
  const cliente = crearClientePoliticaCeseRRHH({ fetchImpl: async () => new Response("", { status: 403 }) });
  await assert.rejects(cliente.consultar(), (error) => error.estado === 403);
});

test("la vista muestra versión, meses y mapeo del servidor; ayuda oculta tras ?", () => {
  const html = renderizarVistaPoliticaCeseRRHH({ politica });
  assert.match(html, /Versión 2/u);
  assert.match(html, /5 meses naturales/u);
  assert.match(html, /9 meses naturales/u);
  assert.match(html, /interinidad\|acumulacion_tareas/u);
  assert.match(html, /Ejemplo sintético · sin aprobación/u);
  assert.match(html, /id="politica-cese-ayuda"[^>]*hidden/u);
  const eventos = new Map();
  const raiz = { innerHTML: "", addEventListener: (tipo, fn) => eventos.set(tipo, fn),
    removeEventListener: (tipo) => eventos.delete(tipo), replaceChildren() { this.innerHTML = ""; } };
  const vista = montarVistaPoliticaCeseRRHH({ raiz, politica, cliente: { consultar: async () => politica } });
  eventos.get("click")({ target: { closest: (selector) => selector === "[data-politica-cese-ayuda]" ? {} : null } });
  assert.doesNotMatch(raiz.innerHTML, /id="politica-cese-ayuda"[^>]*hidden/u);
  vista.desmontar();
  assert.equal(eventos.size, 0);
});
