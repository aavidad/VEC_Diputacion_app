import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { exigirVersiones, posterior } from "./versiones-cache.test-helper.mjs";

const raiz = new URL("./", import.meta.url);
// Entrada, coordinador y vistas Dietas cambiaron después de estas versiones
// publicadas; i18n-borradores no, y conserva la suya exacta.
const versionEntradaNueva = posterior("20260924-rescate-web-v4");
const versionCoordinadorNuevo = posterior("20260924-web-paradas-periodos-v1");
const versionVistaNueva = posterior("20260924-web-paradas-periodos-v1");
const versionI18nBorradores = "20260924-web-paradas-periodos-v1";
const versionAnteriorEntrada = "20260924-web-c-ayuda-v5";
const versionAnteriorRecorridos = "20260924-dietas-ayuda-sin-guia-v1";
const versionAnteriorItinerario = "20260924-dietas-ayuda-icono-v1";

function versiones(codigo, recurso) {
  const escapado = recurso.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  return [...codigo.matchAll(new RegExp(`${escapado}\\?v=([^"']+)`, "gu"))].map((coincidencia) => coincidencia[1]);
}

test("la consulta Dietas atraviesa caché caliente desde HTML hasta ambos cargadores", async () => {
  const [html, portal, coordinador, vista, itinerario, borradores, i18nBorradores] = await Promise.all([
    readFile(new URL("index.html", raiz), "utf8"),
    readFile(new URL("portal.js", raiz), "utf8"),
    readFile(new URL("portal-modulos-coordinador.js", raiz), "utf8"),
    readFile(new URL("modulos/dietas/vista-recorridos.js", raiz), "utf8"),
    readFile(new URL("modulos/dietas/vista-itinerario.js", raiz), "utf8"),
    readFile(new URL("modulos/dietas/vista-borradores-propios.js", raiz), "utf8"),
    readFile(new URL("modulos/dietas/i18n-borradores.js", raiz), "utf8"),
  ]);
  const aristas = [
    [html, "/portal-empleado/portal.js", versionEntradaNueva, 1, `/portal-empleado/portal.js?v=${versionAnteriorEntrada}`],
    [portal, "./portal-modulos-coordinador.js", versionCoordinadorNuevo, 1, `./portal-modulos-coordinador.js?v=${versionAnteriorEntrada}`],
    [coordinador, "./modulos/dietas/vista-recorridos.js", versionVistaNueva, 2,
      `./modulos/dietas/vista-recorridos.js?v=${versionAnteriorRecorridos}`],
    [coordinador, "./modulos/dietas/vista-itinerario.js", versionVistaNueva, 2,
      `./modulos/dietas/vista-itinerario.js?v=${versionAnteriorItinerario}`],
    [vista, "./vista-borradores-propios.js", versionVistaNueva, 1,
      `./vista-borradores-propios.js?v=${versionAnteriorRecorridos}`],
    [borradores, "./i18n-borradores.js", versionI18nBorradores, 1,
      `./i18n-borradores.js?v=${versionAnteriorRecorridos}`],
  ];
  const vigentes = new Map();
  for (const [padre, recurso, esperado, cantidad, urlAnterior] of aristas) {
    vigentes.set(recurso, exigirVersiones(padre, recurso, esperado, cantidad));
    assert.ok(!padre.includes(urlAnterior), `${recurso}: no se solicita el objeto antiguo`);
  }

  const cache = new Map([
    [`/portal-empleado/portal.js?v=${versionAnteriorEntrada}`, "respuesta antigua"],
    [`/portal-empleado/portal-modulos-coordinador.js?v=${versionAnteriorEntrada}`, "respuesta antigua"],
    [`/portal-empleado/modulos/dietas/vista-recorridos.js?v=${versionAnteriorRecorridos}`, "respuesta antigua"],
    [`/portal-empleado/modulos/dietas/vista-itinerario.js?v=${versionAnteriorItinerario}`, "respuesta antigua"],
    [`/portal-empleado/modulos/dietas/vista-borradores-propios.js?v=${versionAnteriorRecorridos}`, "respuesta antigua"],
    ["/portal-empleado/modulos/dietas/i18n-borradores.js?v=20260924-dietas-d1d2d4", "respuesta antigua"],
    [`/portal-empleado/modulos/dietas/i18n-borradores.js?v=${versionAnteriorRecorridos}`, "respuesta antigua"],
    [`/portal-empleado/modulos/dietas/dietas.css?v=${versionAnteriorItinerario}`, "respuesta vigente"],
  ]);
  const actuales = new Map([
    [`/portal-empleado/portal.js?v=${vigentes.get("/portal-empleado/portal.js")}`, portal],
    [`/portal-empleado/portal-modulos-coordinador.js?v=${vigentes.get("./portal-modulos-coordinador.js")}`, coordinador],
    [`/portal-empleado/modulos/dietas/vista-recorridos.js?v=${vigentes.get("./modulos/dietas/vista-recorridos.js")}`, vista],
    [`/portal-empleado/modulos/dietas/vista-itinerario.js?v=${vigentes.get("./modulos/dietas/vista-itinerario.js")}`, itinerario],
    [`/portal-empleado/modulos/dietas/vista-borradores-propios.js?v=${vigentes.get("./vista-borradores-propios.js")}`, borradores],
    [`/portal-empleado/modulos/dietas/i18n-borradores.js?v=${vigentes.get("./i18n-borradores.js")}`, i18nBorradores],
  ]);
  for (const [url, codigo] of actuales) {
    assert.ok(!cache.has(url), `${url}: caché antigua no intercepta la carga`);
    cache.set(url, codigo);
    assert.equal(cache.get(url), codigo);
  }
  assert.doesNotMatch(coordinador, /modulos\/dietas\/vista-recorridos\.js\?v=20260924-f2-shell-v1/u);
  // dietas.css cambió después de la versión del icono: la entrada vigente
  // de la caché ya no la sirve y el HTML pide una URL nueva.
  const versionDietasCSS = exigirVersiones(html, "/portal-empleado/modulos/dietas/dietas.css", posterior(versionAnteriorItinerario));
  assert.ok(!cache.has(`/portal-empleado/modulos/dietas/dietas.css?v=${versionDietasCSS}`));
  assert.ok(!borradores.includes("./i18n-borradores.js?v=20260924-dietas-d1d2d4"));
  assert.doesNotMatch(vista, /vista-borradores-propios\.js\?v=20260924-web-paradas-periodos-v1/u);
  assert.doesNotMatch(vista, /vista-borradores-propios\.js\?v=20260924-dietas-ayuda-sin-guia-v1/u);
  assert.doesNotMatch(vista, /vista-borradores-propios\.js\?v=20260924-dietas-recuperacion-v3/u);
  assert.doesNotMatch(vista, /vista-borradores-propios\.js\?v=20260924-f2-consulta-v1/u);
  exigirVersiones(html, "/portal-empleado/portal.css", posterior("20260924-f2-salto-movil-v3"));
  exigirVersiones(html, "/comun/tema-vec.css", posterior("20260924-f2-tema-base-v2"));
});
