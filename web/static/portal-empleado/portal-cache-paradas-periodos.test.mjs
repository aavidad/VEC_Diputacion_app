import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const raiz = new URL("./", import.meta.url);
const version = "20260924-web-paradas-periodos-v1";
const versionEntradaAyuda = "20260924-rescate-web-v2";
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
    ["portal-modulos-coordinador.js", `./${dietas}vista-itinerario.js`, "20260924-osm-base-v3", 2],
    ["portal-modulos-coordinador.js", `./${dietas}mapa-ruta.js`, "20260924-osm-base-v3", 2],
    ["portal-modulos-coordinador.js", `./${dietas}vista-recorridos.js`, "20260924-osm-base-v3", 2],
    ["portal-modulos-coordinador.js", `./${cronos}vista.js`, "20260924-cronos-integrado-v1", 1],
    ["portal-modulos-coordinador.js", `./${cronos}vista-recorridos.js`, "20260924-cronos-integrado-v1", 2],
    ["portal-modulos-coordinador.js", `./${cronos}i18n.js`, "20260924-cronos-integrado-v1", 1],
    [`${dietas}i18n.js`, "./i18n-borradores.js", "20260924-osm-base-v3", 1],
    [`${dietas}vista-borradores-propios.js`, "./i18n.js", "20260924-osm-base-v3", 1],
    [`${dietas}vista-borradores-propios.js`, "./i18n-borradores.js", "20260924-osm-base-v3", 1],
    [`${dietas}i18n-d1.js`, "./i18n.js", "20260924-osm-base-v3", 1],
    [`${dietas}i18n-d4.js`, "./i18n.js", "20260924-osm-base-v3", 1],
    [`${dietas}vista-acceso-papeles.js`, "./i18n-d1.js", "20260924-osm-base-v3", 1],
    [`${dietas}vista-itinerario.js`, "./i18n.js", "20260924-osm-base-v3", 1],
    [`${dietas}vista-itinerario.js`, "./i18n-d4.js", "20260924-osm-base-v3", 1],
    [`${dietas}vista-itinerario.js`, "./mapa-ruta.js", "20260924-osm-base-v3", 1],
    [`${dietas}mapa-ruta.js`, "./i18n.js", "20260924-osm-base-v3", 1],
    [`${dietas}vista-recorridos.js`, "./i18n.js", "20260924-osm-base-v3", 1],
    [`${dietas}vista-recorridos.js`, "./vista-borradores-propios.js", "20260924-osm-base-v3", 1],
    [`${dietas}vista-recorridos.js`, "./vista-acceso-papeles.js", "20260924-osm-base-v3", 1],
    [`${dietas}vista-recorridos.js`, "./mapa-ruta.js", "20260924-osm-base-v3", 1],
    [`${cronos}vista.js`, "./i18n.js", "20260924-cronos-integrado-v1", 1],
    [`${cronos}vista.js`, "./vista-calendario.js", "20260924-cronos-integrado-v1", 1],
    [`${cronos}i18n-c4.js`, "./i18n.js", "20260924-cronos-integrado-v1", 1],
    [`${cronos}vista-calendario.js`, "./i18n-c4.js", "20260924-cronos-integrado-v1", 1],
    [`${cronos}vista-correcciones.js`, "./i18n.js", "20260924-cronos-integrado-v1", 1],
    [`${cronos}vista-notificaciones.js`, "./i18n.js", "20260924-cronos-integrado-v1", 1],
    [`${cronos}vista-recorridos.js`, "./i18n.js", "20260924-cronos-integrado-v1", 1],
    [`${cronos}vista-recorridos.js`, "./vista-correcciones.js", "20260924-cronos-integrado-v1", 1],
    [`${cronos}vista-recorridos.js`, "./vista-notificaciones.js", "20260924-cronos-integrado-v1", 1],
  ];
  const cache = new Map();
  const pedidos = new Set();
  for (const [padre, recurso, previa, cantidad] of aristas) {
    const base = new URL(padre, raiz);
    const vieja = new URL(`${recurso}?v=${previa}`, base).href;
    const versionEsperada = padre === "index.html" && recurso === "/portal-empleado/portal.js"
      ? versionEntradaAyuda : version;
    const nueva = new URL(`${recurso}?v=${versionEsperada}`, base).href;
    cache.set(vieja, "bytes PR25");
    const codigo = await readFile(base, "utf8");
    assert.deepEqual(rutas(codigo, recurso), Array(cantidad).fill(versionEsperada), `${padre} → ${recurso}`);
    assert.ok(!cache.has(nueva), `${recurso}: evita la caché previa`);
    pedidos.add(nueva);
  }
  assert.equal(pedidos.size, cache.size, "cada recurso modificado tiene URL renovada");
  for (const url of pedidos) assert.ok(!cache.has(url), `${url}: no pide bytes de PR25`);
});
