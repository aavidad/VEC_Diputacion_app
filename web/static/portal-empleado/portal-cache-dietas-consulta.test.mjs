import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const raiz = new URL("./", import.meta.url);
const versionEntradaNueva = "20260924-web-c-v1";
const versionVistaNueva = "20260924-dietas-d1d2d4";
const versionAnteriorEntrada = "20260924-f2-dietas-consulta-v2";
const versionAnteriorDietas = "20260924-f2-dietas-consulta-v2";

function versiones(codigo, recurso) {
  const escapado = recurso.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  return [...codigo.matchAll(new RegExp(`${escapado}\\?v=([^"']+)`, "gu"))].map((coincidencia) => coincidencia[1]);
}

test("la consulta Dietas atraviesa caché caliente desde HTML hasta ambos cargadores", async () => {
  const [html, portal, coordinador, vista] = await Promise.all([
    readFile(new URL("index.html", raiz), "utf8"),
    readFile(new URL("portal.js", raiz), "utf8"),
    readFile(new URL("portal-modulos-coordinador.js", raiz), "utf8"),
    readFile(new URL("modulos/dietas/vista-recorridos.js", raiz), "utf8"),
  ]);
  const aristas = [
    [html, "/portal-empleado/portal.js", [versionEntradaNueva], `/portal-empleado/portal.js?v=${versionAnteriorEntrada}`],
    [portal, "./portal-modulos-coordinador.js", [versionEntradaNueva], `./portal-modulos-coordinador.js?v=${versionAnteriorEntrada}`],
    [coordinador, "./modulos/dietas/vista-recorridos.js", [versionVistaNueva, versionVistaNueva],
      `./modulos/dietas/vista-recorridos.js?v=${versionAnteriorDietas}`],
  ];
  for (const [padre, recurso, esperado, urlAnterior] of aristas) {
    assert.deepEqual(versiones(padre, recurso), esperado, recurso);
    assert.ok(!padre.includes(urlAnterior), `${recurso}: no se solicita el objeto antiguo`);
  }

  const cache = new Map([
    [`/portal-empleado/portal.js?v=${versionAnteriorEntrada}`, "respuesta antigua"],
    [`/portal-empleado/portal-modulos-coordinador.js?v=${versionAnteriorEntrada}`, "respuesta antigua"],
    [`/portal-empleado/modulos/dietas/vista-recorridos.js?v=${versionAnteriorDietas}`, "respuesta antigua"],
  ]);
  const actuales = new Map([
    [`/portal-empleado/portal.js?v=${versionEntradaNueva}`, portal],
    [`/portal-empleado/portal-modulos-coordinador.js?v=${versionEntradaNueva}`, coordinador],
    [`/portal-empleado/modulos/dietas/vista-recorridos.js?v=${versionVistaNueva}`, vista],
  ]);
  for (const [url, codigo] of actuales) {
    assert.ok(!cache.has(url), `${url}: caché antigua no intercepta la carga`);
    cache.set(url, codigo);
    assert.equal(cache.get(url), codigo);
  }
  assert.doesNotMatch(coordinador, /modulos\/dietas\/vista-recorridos\.js\?v=20260924-f2-shell-v1/u);
  assert.match(vista, /vista-borradores-propios\.js\?v=20260924-dietas-d1d2d4/u);
  assert.doesNotMatch(vista, /vista-borradores-propios\.js\?v=20260924-f2-consulta-v1/u);
  assert.match(html, /portal\.css\?v=20260924-f2-salto-movil-v3/u);
  assert.match(html, /tema-vec\.css\?v=20260924-f2-tema-base-v2/u);
});
