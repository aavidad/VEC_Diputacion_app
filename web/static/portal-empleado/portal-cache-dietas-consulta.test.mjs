import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const raiz = new URL("./", import.meta.url);
const versionEntradaNueva = "20260924-web-c-ayuda-v7";
const versionVistaNueva = "20260924-dietas-preparacion-sin-guia-v3";
const versionAnteriorEntrada = "20260924-web-c-ayuda-v6";
const versionAnteriorRecorridos = "20260924-dietas-ayuda-sin-guia-v2";
const versionCSS = "20260924-dietas-ayuda-icono-v1";

function versiones(codigo, recurso) {
  const escapado = recurso.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  return [...codigo.matchAll(new RegExp(`${escapado}\\?v=([^"']+)`, "gu"))].map((coincidencia) => coincidencia[1]);
}

test("la consulta Dietas atraviesa caché caliente desde HTML hasta ambos cargadores", async () => {
  const [html, portal, coordinador, vista, borradores] = await Promise.all([
    readFile(new URL("index.html", raiz), "utf8"),
    readFile(new URL("portal.js", raiz), "utf8"),
    readFile(new URL("portal-modulos-coordinador.js", raiz), "utf8"),
    readFile(new URL("modulos/dietas/vista-recorridos.js", raiz), "utf8"),
    readFile(new URL("modulos/dietas/vista-borradores-propios.js", raiz), "utf8"),
  ]);
  const aristas = [
    [html, "/portal-empleado/portal.js", [versionEntradaNueva], `/portal-empleado/portal.js?v=${versionAnteriorEntrada}`],
    [portal, "./portal-modulos-coordinador.js", [versionEntradaNueva], `./portal-modulos-coordinador.js?v=${versionAnteriorEntrada}`],
    [coordinador, "./modulos/dietas/vista-recorridos.js", [versionVistaNueva, versionVistaNueva],
      `./modulos/dietas/vista-recorridos.js?v=${versionAnteriorRecorridos}`],
    [vista, "./vista-borradores-propios.js", [versionVistaNueva],
      `./vista-borradores-propios.js?v=${versionAnteriorRecorridos}`],
  ];
  for (const [padre, recurso, esperado, urlAnterior] of aristas) {
    assert.deepEqual(versiones(padre, recurso), esperado, recurso);
    assert.ok(!padre.includes(urlAnterior), `${recurso}: no se solicita el objeto antiguo`);
  }

  const cache = new Map([
    [`/portal-empleado/portal.js?v=${versionAnteriorEntrada}`, "respuesta antigua"],
    [`/portal-empleado/portal-modulos-coordinador.js?v=${versionAnteriorEntrada}`, "respuesta antigua"],
    [`/portal-empleado/modulos/dietas/vista-recorridos.js?v=${versionAnteriorRecorridos}`, "respuesta antigua"],
    [`/portal-empleado/modulos/dietas/vista-itinerario.js?v=${versionAnteriorRecorridos}`, "respuesta vigente"],
    [`/portal-empleado/modulos/dietas/vista-borradores-propios.js?v=${versionAnteriorRecorridos}`, "respuesta antigua"],
    ["/portal-empleado/modulos/dietas/i18n-borradores.js?v=20260924-dietas-d1d2d4", "respuesta antigua"],
    [`/portal-empleado/modulos/dietas/i18n-borradores.js?v=${versionAnteriorRecorridos}`, "respuesta vigente"],
    [`/portal-empleado/modulos/dietas/dietas.css?v=${versionCSS}`, "respuesta vigente"],
  ]);
  const actuales = new Map([
    [`/portal-empleado/portal.js?v=${versionEntradaNueva}`, portal],
    [`/portal-empleado/portal-modulos-coordinador.js?v=${versionEntradaNueva}`, coordinador],
    [`/portal-empleado/modulos/dietas/vista-recorridos.js?v=${versionVistaNueva}`, vista],
    [`/portal-empleado/modulos/dietas/vista-borradores-propios.js?v=${versionVistaNueva}`, borradores],
  ]);
  for (const [url, codigo] of actuales) {
    assert.ok(!cache.has(url), `${url}: caché antigua no intercepta la carga`);
    cache.set(url, codigo);
    assert.equal(cache.get(url), codigo);
  }
  assert.doesNotMatch(coordinador, /modulos\/dietas\/vista-recorridos\.js\?v=20260924-f2-shell-v1/u);
  assert.deepEqual(versiones(coordinador, "./modulos/dietas/vista-itinerario.js"),
    [versionAnteriorRecorridos, versionAnteriorRecorridos]);
  assert.ok(cache.has(`/portal-empleado/modulos/dietas/vista-itinerario.js?v=${versionAnteriorRecorridos}`));
  assert.deepEqual(versiones(borradores, "./i18n-borradores.js"), [versionAnteriorRecorridos]);
  assert.ok(cache.has(`/portal-empleado/modulos/dietas/i18n-borradores.js?v=${versionAnteriorRecorridos}`));
  assert.deepEqual(versiones(html, "/portal-empleado/modulos/dietas/dietas.css"), [versionCSS]);
  assert.ok(cache.has(`/portal-empleado/modulos/dietas/dietas.css?v=${versionCSS}`));
  assert.ok(!borradores.includes("./i18n-borradores.js?v=20260924-dietas-d1d2d4"));
  assert.match(vista, /vista-borradores-propios\.js\?v=20260924-dietas-preparacion-sin-guia-v3/u);
  assert.doesNotMatch(vista, /vista-borradores-propios\.js\?v=20260924-dietas-ayuda-sin-guia-v2/u);
  assert.doesNotMatch(vista, /vista-borradores-propios\.js\?v=20260924-dietas-ayuda-sin-guia-v1/u);
  assert.doesNotMatch(vista, /vista-borradores-propios\.js\?v=20260924-dietas-recuperacion-v3/u);
  assert.doesNotMatch(vista, /vista-borradores-propios\.js\?v=20260924-f2-consulta-v1/u);
  assert.match(html, /portal\.css\?v=20260924-f2-salto-movil-v3/u);
  assert.match(html, /tema-vec\.css\?v=20260924-f2-tema-base-v2/u);
});
