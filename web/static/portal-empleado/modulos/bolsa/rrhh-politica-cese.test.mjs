import assert from "node:assert/strict";
import test from "node:test";
import { crearClientePoliticaCeseRRHH, RUTA_POLITICA_CESE, validarPoliticaCeseRRHH } from "./rrhh-politica-cese-api.js";
import { montarVistaPoliticaCeseRRHH, renderizarVistaPoliticaCeseRRHH } from "./rrhh-politica-cese-vista.js?v=20261007-pantallas-textos-final-v1";
import { prepararTextosPortal } from "../../portal-i18n.js?v=20261007-pantallas-textos-final-v1";
await prepararTextosPortal("bolsa");

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
  assert.deepEqual(Object.keys(llamadas[0][1].headers), ["Accept"]);
  assert.equal(llamadas[0][1].headers.Authorization, undefined);
  assert.equal(llamadas[0][1].headers.Cookie, undefined);
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

test("respeta los límites B45 y representa cero meses como disponibilidad inmediata", () => {
  const mapa100 = Object.fromEntries(Array.from({ length: 100 }, (_, i) => [`modalidad_${i}`, "general"]));
  const amplia = { ...politica, catalogo_ref: "r".repeat(512), mapeo: mapa100,
    meses_general: 0, meses_acumulacion: 120 };
  const aceptada = validarPoliticaCeseRRHH(sobre(amplia));
  assert.equal(aceptada.meses_general, 0);
  assert.equal(Object.keys(aceptada.mapeo).length, 100);
  assert.match(renderizarVistaPoliticaCeseRRHH({ politica: aceptada }), /Disponibilidad inmediata/u);
  assert.match(renderizarVistaPoliticaCeseRRHH({ politica: aceptada }), /120 meses naturales/u);
  assert.match(renderizarVistaPoliticaCeseRRHH({ politica: { ...aceptada, meses_acumulacion: 1 } }), /1 mes natural/u);
  assert.throws(() => validarPoliticaCeseRRHH(sobre({ ...amplia, meses_general: 121 })));
  assert.throws(() => validarPoliticaCeseRRHH(sobre({ ...amplia, meses_acumulacion: -1 })));
  assert.throws(() => validarPoliticaCeseRRHH(sobre({ ...amplia, catalogo_ref: "r".repeat(513) })));
  assert.throws(() => validarPoliticaCeseRRHH(sobre({ ...amplia,
    mapeo: { ...mapa100, otra_modalidad: "general" } })));
  assert.equal(validarPoliticaCeseRRHH(sobre({ ...politica,
    catalogo_ref: "é".repeat(256), mapeo: { [`${"a".repeat(80)}|${"b".repeat(80)}`]: "general" } })).meses_general, 5);
  const mapaGrande = Object.fromEntries(Array.from({ length: 100 }, (_, i) =>
    [`m${String(i).padStart(3, "0")}${"a".repeat(76)}|${"b".repeat(80)}`, "acumulacion_tareas"]));
  assert.throws(() => validarPoliticaCeseRRHH(sobre({ ...politica, mapeo: mapaGrande })));
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

// Recorrido en cidonia del 06/10/2026: Reglas y versiones daba 404 en
// /api/vec/bolsa/politica-cese y devolvía a Inicio. Sin la API se dice en llano.
test("sin API de política de cese (404) la vista dice «no disponible», sin reintento; un 403 sigue denegado", async () => {
  const raiz = { innerHTML: "", addEventListener() {}, removeEventListener() {}, replaceChildren() { this.innerHTML = ""; } };
  const anuncios = [];
  const vista = montarVistaPoliticaCeseRRHH({ raiz, politica: null, noDisponible: true, anunciar: (m) => anuncios.push(m),
    cliente: { consultar: async () => { throw Object.assign(new Error("x"), { estado: 404 }); } } });
  assert.match(raiz.innerHTML, /Esta pantalla no está disponible/u);
  assert.match(raiz.innerHTML, /avise a Informática/u);
  assert.match(raiz.innerHTML, /aria-labelledby="politica-cese-no-disponible"/u);
  assert.doesNotMatch(raiz.innerHTML, /data-politica-cese-actualizar|role="alert"/u);
  assert.match(raiz.innerHTML, /data-vista="portal"/u);
  vista.desmontar();

  const html = renderizarVistaPoliticaCeseRRHH({ politica: null, estado: "no_disponible" });
  assert.doesNotMatch(html, /data-politica-cese-ayuda/u, "sin ayuda en pantalla ni botones");
  assert.match(renderizarVistaPoliticaCeseRRHH({ politica: null, estado: "denegada" }), /role="alert"/u);
  assert.deepEqual(anuncios, []);
});
