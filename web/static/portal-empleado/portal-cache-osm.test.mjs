import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const raiz = new URL("./", import.meta.url);
const version = "20260924-osm-base-v1";
const dietas = "modulos/dietas/";

test("OSM renueva cada consumidor immutable desde la entrada C hasta mapa e idiomas", async () => {
  const aristas = [
    ["index.html", "/portal-empleado/portal.js", "20260924-web-c-v3", 1],
    ["portal.js", "./portal-modulos-coordinador.js", "20260924-web-c-v3", 1],
    ["portal-modulos-coordinador.js", `./${dietas}mapa-ruta.js`, "20260923-dietas-r1", 2],
    ["portal-modulos-coordinador.js", `./${dietas}vista-itinerario.js`, "20260924-dietas-d1d2d4", 2],
    ["portal-modulos-coordinador.js", `./${dietas}vista-recorridos.js`, "20260924-dietas-recuperacion-v3", 2],
    [`${dietas}vista-recorridos.js`, "./vista-borradores-propios.js", "20260924-dietas-recuperacion-v3", 1],
    [`${dietas}vista-recorridos.js`, "./vista-acceso-papeles.js", "20260924-dietas-d1d2d4", 1],
    [`${dietas}vista-recorridos.js`, "./mapa-ruta.js", null, 1],
    [`${dietas}vista-itinerario.js`, "./mapa-ruta.js", null, 1],
    [`${dietas}vista-itinerario.js`, "./i18n-d4.js", "20260924-dietas-d1d2d4", 1],
    [`${dietas}vista-acceso-papeles.js`, "./i18n-d1.js", "20260924-dietas-d1d2d4", 1],
    ...["mapa-ruta.js", "vista-recorridos.js", "vista-itinerario.js", "vista-borradores-propios.js", "i18n-d1.js", "i18n-d4.js"]
      .map((padre) => [`${dietas}${padre}`, "./i18n.js", "20260924-dietas-d1d2d4", 1]),
  ];
  for (const [padre, recurso, anterior, cantidad] of aristas) {
    const codigo = await readFile(new URL(padre, raiz), "utf8");
    const literal = recurso.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    const urls = [...codigo.matchAll(new RegExp(`${literal}(?:\\?v=[^"']+)?(?=["'])`, "gu"))].map(([url]) => url);
    assert.deepEqual(urls, Array(cantidad).fill(`${recurso}?v=${version}`), `${padre} → ${recurso}`);
    const vieja = new URL(`${recurso}${anterior ? `?v=${anterior}` : ""}`, new URL(padre, raiz)).href;
    for (const url of urls) {
      assert.notEqual(new URL(url, new URL(padre, raiz)).href, vieja, "una respuesta anterior en caché no sirve el nuevo consumidor");
    }
  }
});

test("OSM incorpora el estilo de ayudas F1 y conserva el catálogo de borradores", async () => {
  const html = await readFile(new URL("index.html", raiz), "utf8");
  const borradores = await readFile(new URL(`${dietas}vista-borradores-propios.js`, raiz), "utf8");
  assert.match(html, /dietas\/dietas\.css\?v=20260924-dietas-ayuda-icono-v1/u);
  assert.match(html, /portal\.css\?v=20260924-f2-salto-movil-v3/u);
  assert.match(borradores, /i18n-borradores\.js\?v=20260924-dietas-d1d2d4/u);
});

test("el mapa combinado descarga consumidores nuevos también después del corrector de ayudas F1", async () => {
  for (const [padre, recurso, previa, cantidad] of [
    ["index.html", "/portal-empleado/portal.js", "20260924-web-c-ayuda-v4", 1],
    ["portal.js", "./portal-modulos-coordinador.js", "20260924-web-c-ayuda-v4", 1],
    ["portal-modulos-coordinador.js", `./${dietas}vista-itinerario.js`, "20260924-dietas-ayuda-icono-v1", 2],
    ["portal-modulos-coordinador.js", `./${dietas}vista-recorridos.js`, "20260924-dietas-ayuda-icono-v1", 2],
  ]) {
    const codigo = await readFile(new URL(padre, raiz), "utf8");
    assert.equal(codigo.split(`${recurso}?v=${version}`).length - 1, cantidad, recurso);
    assert.ok(!codigo.includes(`${recurso}?v=${previa}`), `${padre}: no usa bytes de F1 anteriores al mapa`);
  }
});
