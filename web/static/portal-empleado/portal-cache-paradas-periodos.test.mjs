import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { exigirVersiones, posterior } from "./versiones-cache.test-helper.mjs";

const raiz = new URL("./", import.meta.url);
const version = "20260924-web-paradas-periodos-v1";
const versionEntradaAyuda = "20260924-rescate-web-v4";
const dietas = "modulos/dietas/";
const cronos = "modulos/cronos/";

function rutas(codigo, recurso) {
  const escapado = recurso.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  return [...codigo.matchAll(new RegExp(`${escapado}\\?v=([^"']+)`, "gu"))].map((m) => m[1]);
}

test("la caché de PR25 solicita de nuevo las paradas, los periodos y sus hojas", async () => {
  const aristas = [
    ["index.html", "/portal-empleado/portal.js", "20260924-osm-base-v3", 1],
    ["index.html", `/portal-empleado/${dietas}borradores-propios.css`, "20260924-f2-shell-v1", 1],
    ["index.html", `/portal-empleado/${cronos}cronos.css`, "20260924-f2-shell-v1", 1],
    ["portal.js", "./portal-modulos-coordinador.js", "20260924-osm-base-v3", 1],
    ["portal-modulos-coordinador.js", `./${dietas}mapa-ruta.js`, "20260924-osm-base-v3", 1],
    ["portal-modulos-coordinador.js", `./${dietas}vista-recorridos.js`, "20260924-osm-base-v3", 1],
    ["portal-modulos-coordinador.js", `./${cronos}vista-saldo-conectado.js`, "20260925-tanda-v1", 1],
    ["portal-modulos-coordinador.js", `./${cronos}i18n.js`, "20260924-cronos-integrado-v1", 1],
    [`${dietas}i18n.js`, "./i18n-borradores.js", "20260924-osm-base-v3", 1],
    [`${dietas}vista-borradores-propios.js`, "./i18n.js", "20260924-osm-base-v3", 1],
    [`${dietas}vista-borradores-propios.js`, "./i18n-borradores.js", "20260924-osm-base-v3", 1],
    [`${dietas}mapa-ruta.js`, "./i18n.js", "20260924-osm-base-v3", 1],
    [`${dietas}vista-recorridos.js`, "./i18n.js", "20260924-osm-base-v3", 1],
    [`${dietas}vista-recorridos.js`, "./vista-borradores-propios.js", "20260924-osm-base-v3", 1],
    [`${cronos}vista.js`, "./i18n.js", "20260924-cronos-integrado-v1", 1],
    [`${cronos}vista.js`, "./vista-calendario.js", "20260924-cronos-integrado-v1", 1],
    [`${cronos}i18n-c4.js`, "./i18n.js", "20260924-cronos-integrado-v1", 1],
    [`${cronos}vista-calendario.js`, "./i18n-c4.js", "20260924-cronos-integrado-v1", 1],
    [`${cronos}vista-recorridos.js`, "./i18n.js", "20260924-cronos-integrado-v1", 1],
    [`${cronos}vista-recorridos.js`, "./vista-correcciones.js", "20260924-cronos-integrado-v1", 1],
    [`${cronos}vista-recorridos.js`, "./vista-notificaciones.js", "20260924-cronos-integrado-v1", 1],
  ];
  const cache = new Map();
  const pedidos = new Set();
  for (const [padre, recurso, previa, cantidad] of aristas) {
    const base = new URL(padre, raiz);
    const vieja = new URL(`${recurso}?v=${previa}`, base).href;
    // Todo se renovó después de PR25: cada importador pide una URL posterior,
    // única. Dietas interno ya no carga itinerario, D1 ni D4 de presentación.
    const versionEsperada = posterior(padre === "index.html" && recurso === "/portal-empleado/portal.js" ? versionEntradaAyuda : version);
    cache.set(vieja, "bytes PR25");
    const codigo = await readFile(base, "utf8");
    const vigente = exigirVersiones(codigo, recurso, versionEsperada, cantidad);
    const nueva = new URL(`${recurso}?v=${vigente}`, base).href;
    assert.ok(!cache.has(nueva), `${recurso}: evita la caché previa`);
    pedidos.add(nueva);
  }
  assert.equal(pedidos.size, cache.size, "cada recurso modificado tiene URL renovada");
  for (const url of pedidos) assert.ok(!cache.has(url), `${url}: no pide bytes de PR25`);
});
