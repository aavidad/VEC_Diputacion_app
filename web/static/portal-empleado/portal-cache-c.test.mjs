import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { PLANTILLA_TESELAS_OSM_INTERNA } from "./modulos/dietas/contrato.js";

const raiz = new URL("./", import.meta.url);
const vistasC = "20260924-web-c-v1";
const recuperacion = "20260924-dietas-recuperacion-v3";
const entrada = "20260924-web-c-ayuda-v5";
const cronos = "20260924-cronos-integrado-v1";
const dietas = "20260924-dietas-ayuda-icono-v1";
const sinGuia = "20260924-dietas-ayuda-sin-guia-v1";
const versiones = (codigo, recurso) => [...codigo.matchAll(new RegExp(`${recurso.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}\\?v=([^"']+)`, "gu"))].map((m) => m[1]);

test("capa C no reutiliza los consumidores previos de B con caché immutable", async () => {
  const html = await readFile(new URL("index.html", raiz), "utf8");
  const portal = await readFile(new URL("portal.js", raiz), "utf8");
  const coordinador = await readFile(new URL("portal-modulos-coordinador.js", raiz), "utf8");
  const aristas = [
    [html, "/portal-empleado/portal.js", "20260924-web-integrada-v2", entrada, 1],
    [portal, "./portal-modulos-coordinador.js", "20260924-web-integrada-v2", entrada, 1],
    [coordinador, "./modulos/cronos/vista.js", "20260924-f2-shell-v1", cronos, 1],
    [coordinador, "./modulos/cronos/vista-recorridos.js", "20260924-cronos-ayuda-v1", cronos, 2],
    [coordinador, "./modulos/cronos/i18n.js", "20260924-f2-web2", cronos, 1],
    [coordinador, "./modulos/dietas/vista-recorridos.js", "20260924-f2-consulta-v2", sinGuia, 2],
    [coordinador, "./modulos/dietas/vista-itinerario.js", "20260923-dietas-r1", dietas, 2],
    [coordinador, "./modulos/personal/vista-ficha-integral.js", "20260924-f2-shell-v1", vistasC, 2],
    [coordinador, "./modulos/personal/vista-rpt-publica.js", "20260920-personal-rpt-publica-v3", vistasC, 1],
    [coordinador, "./modulos/personal/vista-estructura-organizativa-publica.js", "20260924-f2-cache-v3", vistasC, 1],
    [coordinador, "./modulos/nominas/vista.js", "20260924-f2-shell-v1", vistasC, 1],
    [html, "/portal-empleado/modulos/cronos/permisos.css", "20260924-cronos-ayuda-v1", cronos, 1],
    [html, "/portal-empleado/modulos/dietas/dietas.css", "20260924-f2-shell-v1", dietas, 1],
  ];
  const cache = new Map(aristas.map(([, ruta, previa]) => [`${ruta}?v=${previa}`, "bytes anteriores"]));
  for (const [codigo, ruta, previa, nueva, cantidad] of aristas) {
    assert.deepEqual(versiones(codigo, ruta), Array(cantidad).fill(nueva), ruta);
    assert.ok(!codigo.includes(`${ruta}?v=${previa}`), `${ruta}: no conserva URL anterior`);
    assert.ok(!cache.has(`${ruta}?v=${nueva}`), `${ruta}: necesita bytes nuevos`);
  }
});

test("la ayuda de Dietas no reutiliza C v3 ni las vistas y CSS anteriores", async () => {
  const [html, portal, coordinador] = await Promise.all([
    readFile(new URL("index.html", raiz), "utf8"),
    readFile(new URL("portal.js", raiz), "utf8"),
    readFile(new URL("portal-modulos-coordinador.js", raiz), "utf8"),
  ]);
  const aristas = [
    [html, "/portal-empleado/portal.js", "20260924-web-c-v3", entrada, 1],
    [portal, "./portal-modulos-coordinador.js", "20260924-web-c-v3", entrada, 1],
    [coordinador, "./modulos/dietas/vista-itinerario.js", "20260924-dietas-d1d2d4", dietas, 2],
    [coordinador, "./modulos/dietas/vista-recorridos.js", recuperacion, sinGuia, 2],
    [html, "/portal-empleado/modulos/dietas/dietas.css", "20260924-dietas-d1d2d4", dietas, 1],
  ];
  const cache = new Map(aristas.map(([, ruta, previa]) => [`${ruta}?v=${previa}`, "respuesta C antigua"]));
  for (const [codigo, ruta, previa, nueva, cantidad] of aristas) {
    assert.deepEqual(versiones(codigo, ruta), Array(cantidad).fill(nueva), ruta);
    assert.ok(!codigo.includes(`${ruta}?v=${previa}`), `${ruta}: no usa URL C anterior`);
    assert.ok(!cache.has(`${ruta}?v=${nueva}`), `${ruta}: debe descargar bytes nuevos`);
  }
});

test("capa C empaqueta las hojas nuevas y conserva los recursos del mapa OSM", async () => {
  const activosCronos = ["i18n-c4.js", "i18n-c5.js", "i18n-c6.js", "i18n-c9.js", ...["vista-calendario", "vista-correcciones", "vista-catalogo-permisos", "vista-notificaciones"].flatMap((nombre) => [nombre + ".js", nombre + ".css"])];
  const activosDietas = ["i18n-d1.js", "i18n-d4.js", "vista-acceso-papeles.js", "mapa-ruta.js"];
  for (const nombre of ["interno.manifest", "produccion.manifest"]) {
    const contenido = await readFile(new URL(`../../${nombre}`, raiz), "utf8");
    const entradas = contenido.split(/\r?\n/u);
    for (const [modulo, activos] of [["cronos", activosCronos], ["dietas", activosDietas]]) {
      for (const activo of activos) assert.ok(entradas.includes(`static/portal-empleado/modulos/${modulo}/${activo}`), `${nombre}: ${activo}`);
    }
  }
  const html = await readFile(new URL("index.html", raiz), "utf8");
  for (const nombre of ["vista-calendario", "vista-correcciones", "vista-catalogo-permisos", "vista-notificaciones"]) {
    assert.deepEqual(versiones(html, `/portal-empleado/modulos/cronos/${nombre}.css`), [cronos]);
  }
  const mapa = await readFile(new URL("modulos/dietas/mapa-ruta.js", raiz), "utf8");
  assert.match(mapa, /OpenStreetMap/u);
  assert.equal(PLANTILLA_TESELAS_OSM_INTERNA, "/tiles/osm/{z}/{x}/{y}.png");
});

test("el corrector Dietas descarga de nuevo la cadena que tenía C inicial", async () => {
  const [html, portal, coordinador, recorridos] = await Promise.all([
    readFile(new URL("index.html", raiz), "utf8"),
    readFile(new URL("portal.js", raiz), "utf8"),
    readFile(new URL("portal-modulos-coordinador.js", raiz), "utf8"),
    readFile(new URL("modulos/dietas/vista-recorridos.js", raiz), "utf8"),
  ]);
  const aristas = [
    [html, "/portal-empleado/portal.js", "20260924-web-c-v1", entrada, 1],
    [portal, "./portal-modulos-coordinador.js", "20260924-web-c-v1", entrada, 1],
    [coordinador, "./modulos/dietas/vista-recorridos.js", "20260924-dietas-d1d2d4", sinGuia, 2],
    [recorridos, "./vista-borradores-propios.js", "20260924-dietas-d1d2d4", sinGuia, 1],
  ];
  for (const [codigo, ruta, anterior] of [
    [html, "/portal-empleado/portal.js", "20260924-web-c-v2"],
    [portal, "./portal-modulos-coordinador.js", "20260924-web-c-v2"],
    [coordinador, "./modulos/dietas/vista-recorridos.js", "20260924-dietas-recuperacion-v2"],
    [recorridos, "./vista-borradores-propios.js", "20260924-dietas-recuperacion-v2"],
  ]) assert.ok(!codigo.includes(`${ruta}?v=${anterior}`), "no conserva el corrector intermedio");
  const cache = new Map(aristas.map(([, ruta, previa]) => [`${ruta}?v=${previa}`, "versión anterior"]));
  for (const [codigo, ruta, previa, nueva, cantidad] of aristas) {
    assert.deepEqual(versiones(codigo, ruta), Array(cantidad).fill(nueva));
    assert.ok(!codigo.includes(`${ruta}?v=${previa}`));
    assert.ok(!cache.has(`${ruta}?v=${nueva}`));
  }
});
