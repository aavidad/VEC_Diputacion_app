import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { exigirVersiones, posterior } from "./versiones-cache.test-helper.mjs";

const raiz = new URL("./", import.meta.url);
const VERSION_MONTAJE = "20260925-dietas-montaje-v1";
// Última versión publicada en main de la entrada, el shell y la vista.
const PUBLICADA = "20260925-aspecto-v1";

function versiones(codigo, recurso) {
  const escapado = recurso.replace(/[.*+?^${}()|[\]\\]/gu, "\\$&");
  return [...codigo.matchAll(new RegExp(`${escapado}\\?v=([^"']+)`, "gu"))]
    .map((coincidencia) => coincidencia[1]);
}

test("Dietas interno atraviesa una caché caliente con sus clientes reales", async () => {
  const [html, portal, coordinador, recorridos] = await Promise.all([
    readFile(new URL("index.html", raiz), "utf8"),
    readFile(new URL("portal.js", raiz), "utf8"),
    readFile(new URL("portal-modulos-coordinador.js", raiz), "utf8"),
    readFile(new URL("modulos/dietas/vista-recorridos.js", raiz), "utf8"),
  ]);
  const entrada = versiones(html, "/portal-empleado/portal.js");
  const shell = versiones(portal, "./portal-modulos-coordinador.js");
  assert.equal(entrada.length, 1);
  assert.equal(shell.length, 1);
  assert.notEqual(entrada[0], "20260924-rescate-web-v4", "HTML sirve un portal.js nuevo");
  assert.notEqual(shell[0], "20260924-web-paradas-periodos-v1", "portal.js sirve el shell nuevo");
  assert.notEqual(entrada[0], PUBLICADA, "HTML no reutiliza el portal.js publicado");
  assert.notEqual(shell[0], PUBLICADA, "portal.js no reutiliza el shell publicado");
  assert.deepEqual(versiones(coordinador, "./portal-composicion-empleado.js"), [VERSION_MONTAJE]);

  const cargadorInterno = coordinador.split("const CARGADORES_INTERNOS_PREDETERMINADOS =")[1]
    .split("function componerModuloAislado")[0];
  for (const recurso of [
    "vista-recorridos.js", "cliente-borradores-http.js", "cliente-asignacion-http.js",
    "calculador-rutas-http.js", "mapa-ruta.js",
  ]) {
    assert.deepEqual(versiones(cargadorInterno, `./modulos/dietas/${recurso}`), [VERSION_MONTAJE], recurso);
  }
  assert.doesNotMatch(cargadorInterno, /vista-itinerario|adaptador-presentacion|datos-presentacion|calculador-rutas-presentacion/u);
  assert.doesNotMatch(recorridos, /vista-itinerario\.js|montarMapaInicialGranadaDietas/u);

  const cacheAntigua = new Map([
    ["/portal-empleado/portal.js?v=20260924-rescate-web-v4", "portal antiguo"],
    ["/portal-empleado/portal-modulos-coordinador.js?v=20260924-web-paradas-periodos-v1", "shell antiguo"],
    ["/portal-empleado/modulos/dietas/vista-recorridos.js?v=20260924-web-paradas-periodos-v1", "vista antigua"],
    [`/portal-empleado/portal.js?v=${PUBLICADA}`, "portal publicado"],
    [`/portal-empleado/portal-modulos-coordinador.js?v=${PUBLICADA}`, "shell publicado"],
    [`/portal-empleado/modulos/dietas/vista-recorridos.js?v=${PUBLICADA}`, "vista publicada"],
  ]);
  for (const recurso of [
    `/portal-empleado/portal.js?v=${entrada[0]}`,
    `/portal-empleado/portal-modulos-coordinador.js?v=${shell[0]}`,
    `/portal-empleado/modulos/dietas/vista-recorridos.js?v=${VERSION_MONTAJE}`,
  ]) assert.equal(cacheAntigua.has(recurso), false, recurso);
});
