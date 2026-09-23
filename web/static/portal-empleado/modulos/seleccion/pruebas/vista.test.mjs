import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { montarVistaPruebas, renderizarVistaPruebas } from "./vista.js";
import { crearTraductorPruebas, MENSAJES_PRUEBAS_ES } from "./i18n.js";

test("sin conector muestra dependencia concreta y ninguna calificación o acta ficticia", () => {
  const html = renderizarVistaPruebas();
  assert.match(html, /data-estado="no_configurado"/u);
  assert.match(html, /Falta conectar la convocatoria, sus pruebas, los resultados por aspirante y las actas/u);
  assert.match(html, /No se han aportado resultados ni actas de ejemplo/u);
  assert.equal((html.match(/<button[^>]*disabled/gu) || []).length, 2);
  assert.doesNotMatch(html, /<tbody>/u);
});

test("distingue carga, vacío, denegación y error sin revelar datos de otra respuesta", () => {
  for (const estado of ["cargando", "vacio", "denegado", "error"]) {
    const html = renderizarVistaPruebas({ estado, convocatoria: { nombre: "Reservada" }, resultados: [{ aspirante: "Privado" }] });
    assert.match(html, new RegExp(`data-estado="${estado}"`));
    assert.doesNotMatch(html, /Reservada|Privado/u);
  }
  assert.match(renderizarVistaPruebas({ estado: "disponible", pruebas: [], resultados: [], actas: [] }), /data-estado="vacio"/u);
});

test("muestra solo valores recibidos, localiza fechas y escapa todo dato externo", () => {
  const html = renderizarVistaPruebas({
    estado: "disponible",
    convocatoria: { nombre: "OPE <2026>", referencia: "conv&1", estado: "publicada" },
    pruebas: [{ nombre: "Ejercicio <1>", fecha: "2026-09-24", estado: "programada" }],
    resultados: [{ prueba: "Ejercicio 1", aspirante: "asp&1", resultado: "Apto <script>", estado: "validado" }],
    actas: [{ prueba: "Ejercicio 1", referencia: "acta&1", fecha: "2026-09-24", estado: "en_revision" }],
  });
  assert.match(html, /data-estado="disponible"/u);
  assert.match(html, /OPE &lt;2026&gt;|conv&amp;1/u);
  assert.match(html, /Estado de convocatoria.*Publicada/u);
  assert.match(html, /24 sept 2026/u);
  assert.match(html, /asp&amp;1/u);
  assert.match(html, /Apto &lt;script&gt;/u);
  assert.doesNotMatch(html, /<script>|<button(?![^>]*disabled)/u);
  assert.equal((html.match(/<table /gu) || []).length, 3);
});

test("estados desconocidos no se presentan como validados o publicados", () => {
  const html = renderizarVistaPruebas({ estado: "disponible", pruebas: [{ nombre: "Uno", estado: "firmada" }] });
  assert.match(html, /Estado no informado/u);
  assert.doesNotMatch(html, /Firmada|firmada/u);
});

test("montaje permite actualizaciones y desmontaje idempotente", () => {
  const raiz = { innerHTML: "", replaceChildren() { this.innerHTML = ""; } };
  let registrado;
  const vista = montarVistaPruebas({ raiz, registrarDesmontar: (desmontar) => { registrado = desmontar; } });
  assert.match(raiz.innerHTML, /Sin conexión configurada/u);
  vista.actualizar({ estado: "cargando" });
  assert.match(raiz.innerHTML, /Cargando pruebas y actas/u);
  registrado();
  assert.equal(raiz.innerHTML, "");
  vista.actualizar({ estado: "disponible" });
  assert.equal(raiz.innerHTML, "");
  vista.desmontar();
  assert.throws(() => montarVistaPruebas({ raiz: null }), TypeError);
});

test("catálogo cerrado, ayuda con teclado y CSS de lienzo responsive", async () => {
  const traducir = crearTraductorPruebas();
  assert.equal(traducir("titulo"), "Pruebas y actas");
  assert.throws(() => traducir("desconocida"), /clave i18n/u);
  const [js, css] = await Promise.all([
    readFile(new URL("./vista.js", import.meta.url), "utf8"),
    readFile(new URL("./pruebas.css", import.meta.url), "utf8"),
  ]);
  const claves = [...js.matchAll(/t\("([a-z_]+)"\)/gu)].map((hallazgo) => hallazgo[1]);
  claves.forEach((clave) => assert.ok(Object.hasOwn(MENSAJES_PRUEBAS_ES, clave), `falta ${clave}`));
  assert.match(renderizarVistaPruebas(), /<details class="pruebas-ayuda"><summary aria-label=/u);
  assert.match(css, /@media \(max-width: 760px\)/u);
  assert.match(css, /\.pruebas-tabla \{ max-height: min\(30vh, 250px\); \}/u);
  assert.doesNotMatch(js, /(?:fetch\(|localStorage|sessionStorage|document\.cookie|indexedDB|new XMLHttpRequest)/u);
  assert.doesNotMatch(css, /#[0-9a-f]{3,8}\b/iu);
});
