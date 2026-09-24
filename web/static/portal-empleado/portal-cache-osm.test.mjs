import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { exigirVersiones, posterior } from "./versiones-cache.test-helper.mjs";

// i18n-revision.js e i18n-borradores.js no han cambiado y conservan su versión
// exacta; el resto de la cadena se renovó después y solo exige una URL
// posterior a la publicada, única en cada importador.
const raiz = new URL("./", import.meta.url);
const version = "20260924-web-paradas-periodos-v1";
const versionAyuda = "20260924-rescate-web-v4";
const versionOSM = "20260924-osm-base-v3";
const dietas = "modulos/dietas/";

test("OSM renueva cada consumidor immutable desde la entrada C hasta mapa e idiomas", async () => {
  const aristas = [
    ["index.html", "/portal-empleado/portal.js", "20260924-web-c-v3", 1],
    ["portal.js", "./portal-modulos-coordinador.js", "20260924-web-c-v3", 1],
    ["portal.js", "./ayuda-contenido.js", "20260917-ayuda-contratacion", 1],
    ["portal.js", "./ayudante-tramites.js", "20260920-ayudante-tramites-v1", 1],
    ["ayudante-tramites.js", "./ayuda-contenido.js", null, 1],
    [`${dietas}i18n.js`, "./i18n-revision.js", "20260924-f2-web2", 1],
    [`${dietas}vista-recorridos.js`, "./i18n-revision.js", null, 1],
    ["portal-modulos-coordinador.js", `./${dietas}mapa-ruta.js`, "20260923-dietas-r1", 2],
    ["portal-modulos-coordinador.js", `./${dietas}vista-itinerario.js`, "20260924-dietas-d1d2d4", 2],
    ["portal-modulos-coordinador.js", `./${dietas}vista-recorridos.js`, "20260924-dietas-recuperacion-v3", 2],
    [`${dietas}vista-recorridos.js`, "./vista-borradores-propios.js", "20260924-dietas-recuperacion-v3", 1],
    [`${dietas}vista-recorridos.js`, "./vista-acceso-papeles.js", "20260924-dietas-d1d2d4", 1],
    [`${dietas}vista-recorridos.js`, "./mapa-ruta.js", null, 1],
    [`${dietas}vista-itinerario.js`, "./mapa-ruta.js", null, 1],
    [`${dietas}vista-itinerario.js`, "./i18n-d4.js", "20260924-dietas-d1d2d4", 1],
    [`${dietas}vista-acceso-papeles.js`, "./i18n-d1.js", "20260924-dietas-d1d2d4", 1],
    [`${dietas}i18n.js`, "./i18n-borradores.js", "20260924-dietas-d1d2d4", 1],
    [`${dietas}vista-borradores-propios.js`, "./i18n-borradores.js", "20260924-dietas-d1d2d4", 1],
    ...["mapa-ruta.js", "vista-recorridos.js", "vista-itinerario.js", "vista-borradores-propios.js", "i18n-d1.js", "i18n-d4.js"]
      .map((padre) => [`${dietas}${padre}`, "./i18n.js", "20260924-dietas-d1d2d4", 1]),
  ];
  for (const [padre, recurso, anterior, cantidad] of aristas) {
    const codigo = await readFile(new URL(padre, raiz), "utf8");
    const literal = recurso.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    const urls = [...codigo.matchAll(new RegExp(`${literal}(?:\\?v=[^"']+)?(?=["'])`, "gu"))].map(([url]) => url);
    const esperada = recurso === "./i18n-revision.js" ? versionOSM : recurso === "./i18n-borradores.js" ? version
      : posterior(padre === "index.html" || ["./ayuda-contenido.js", "./ayudante-tramites.js"].includes(recurso) ? versionAyuda : version);
    assert.equal(urls.length, cantidad, `${padre} → ${recurso}`);
    assert.ok(urls.every((url) => url.includes("?v=")), `${padre} → ${recurso}: URL versionada`);
    exigirVersiones(codigo, recurso, esperada, cantidad);
    const vieja = new URL(`${recurso}${anterior ? `?v=${anterior}` : ""}`, new URL(padre, raiz)).href;
    for (const url of urls) {
      assert.notEqual(new URL(url, new URL(padre, raiz)).href, vieja, "una respuesta anterior en caché no sirve el nuevo consumidor");
    }
  }
});

test("OSM incorpora el estilo de ayudas F1 y renueva el catálogo de estado vacío", async () => {
  const html = await readFile(new URL("index.html", raiz), "utf8");
  const borradores = await readFile(new URL(`${dietas}vista-borradores-propios.js`, raiz), "utf8");
  exigirVersiones(html, "/portal-empleado/modulos/dietas/dietas.css", posterior("20260924-dietas-ayuda-icono-v1"));
  exigirVersiones(html, "/portal-empleado/portal.css", posterior("20260924-f2-salto-movil-v3"));
  assert.match(borradores, /i18n-borradores\.js\?v=20260924-web-paradas-periodos-v1/u);
});

test("el mapa combinado descarga consumidores nuevos también después del corrector de ayudas F1", async () => {
  for (const [padre, recurso, previa, cantidad] of [
    ["index.html", "/portal-empleado/portal.js", "20260924-web-c-ayuda-v4", 1],
    ["portal.js", "./portal-modulos-coordinador.js", "20260924-web-c-ayuda-v4", 1],
    ["portal-modulos-coordinador.js", `./${dietas}vista-itinerario.js`, "20260924-dietas-ayuda-icono-v1", 2],
    ["portal-modulos-coordinador.js", `./${dietas}vista-recorridos.js`, "20260924-dietas-ayuda-icono-v1", 2],
  ]) {
    const codigo = await readFile(new URL(padre, raiz), "utf8");
    exigirVersiones(codigo, recurso, posterior(padre === "index.html" ? versionAyuda : version), cantidad);
    assert.ok(!codigo.includes(`${recurso}?v=${previa}`), `${padre}: no usa bytes de F1 anteriores al mapa`);
  }
});

test("la cadena final no pide módulos previos a la corrección Leaflet ni al estado vacío", async () => {
  const padres = ["index.html", "portal.js", "portal-modulos-coordinador.js", "ayudante-tramites.js",
    ...["i18n.js", "i18n-d1.js", "i18n-d4.js", "mapa-ruta.js", "vista-acceso-papeles.js",
      "vista-borradores-propios.js", "vista-itinerario.js", "vista-recorridos.js"].map((nombre) => dietas + nombre)];
  for (const padre of padres) {
    const codigo = await readFile(new URL(padre, raiz), "utf8");
    assert.doesNotMatch(codigo, /\.js\?v=20260924-(?:osm-base-v[12]|web-c-ayuda-v[4567]|dietas-ayuda-sin-guia-v[12]|dietas-preparacion-sin-guia-v3)["']/u, padre);
  }
  const html = await readFile(new URL("index.html", raiz), "utf8");
  exigirVersiones(html, "/portal-empleado/modulos/dietas/dietas.css", posterior("20260924-dietas-ayuda-icono-v1"));
});
