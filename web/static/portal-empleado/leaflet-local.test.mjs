import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFile } from "node:fs/promises";
import test from "node:test";

const directorioPortal = new URL("./", import.meta.url);
const directorioLeaflet = new URL("vendor/leaflet-1.9.4/", directorioPortal);
const rutaWebLeaflet = "/portal-empleado/vendor/leaflet-1.9.4/";

const huellasAprobadas = Object.freeze({
  "leaflet.css": "a7837102824184820dfa198d1ebcd109ff6d0ff9a2672a074b9a1b4d147d04c6",
  "leaflet.js": "85d455b4522415f6badc42a0e7d17c919d100347d6b8958bd0dc738fdecd6d50",
  "images/layers.png": "1dbbe9d028e292f36fcba8f8b3a28d5e8932754fc2215b9ac69e4cdecf5107c6",
  "images/layers-2x.png": "066daca850d8ffbef007af00b06eac0015728dee279c51f3cb6c716df7c42edf",
  "images/marker-icon.png": "574c3a5cca85f4114085b6841596d62f00d7c892c7b03f28cbfa301deb1dc437",
  "images/marker-icon-2x.png": "00179c4c1ee830d3a108412ae0d294f55776cfeb085c60129a39aa6fc4ae2528",
  "images/marker-shadow.png": "264f5c640339f042dd729062cfc04c17f8ea0f29882b538e3848ed8f10edb4da",
  LICENSE: "53e8dc25862014e4324741ca18fbe3611e11d42ef69f59f86ea8c5389647d4cb",
});

function sha256(contenido) {
  return createHash("sha256").update(contenido).digest("hex");
}

test("los activos locales coinciden con la distribución oficial aprobada", async () => {
  for (const [ruta, huella] of Object.entries(huellasAprobadas)) {
    const contenido = await readFile(new URL(ruta, directorioLeaflet));
    assert.equal(sha256(contenido), huella, `huella inesperada o activo ausente: ${ruta}`);
  }

  const sumas = await readFile(new URL("SHA256SUMS", directorioLeaflet), "utf8");
  for (const [ruta, huella] of Object.entries(huellasAprobadas)) {
    assert.match(sumas, new RegExp(`^${huella}  ${ruta.replaceAll(".", "\\.")}$`, "m"));
  }
});

test("el portal carga Leaflet local antes de evaluar sus módulos", async () => {
  const html = await readFile(new URL("index.html", directorioPortal), "utf8");
  const cargaCSS = `href="${rutaWebLeaflet}leaflet.css?v=1.9.4"`;
  const cargaJS = `src="${rutaWebLeaflet}leaflet.js?v=1.9.4"`;
  const posicionLeaflet = html.indexOf(cargaJS);
  const posicionPrimerModulo = html.indexOf('<script type="module"');

  assert.ok(html.includes(cargaCSS), "falta la hoja de estilos local de Leaflet");
  assert.ok(posicionLeaflet >= 0, "falta el JavaScript local de Leaflet");
  assert.ok(posicionPrimerModulo > posicionLeaflet, "Leaflet debe cargarse antes del primer módulo");
  assert.match(html, /leaflet\.css\?v=1\.9\.4" integrity="sha384-sHL9NAb7lN7rfvG5lfHpm643Xkcjzp4jFvuavGOndn6pjVqS6ny56CAt3nsEVT4H"/);
  assert.match(html, /leaflet\.js\?v=1\.9\.4" integrity="sha384-tM1WyRnQZLXgeuR\/dXHqqQcD5jN9\+bgKmZW55OJtM9Bv7zCG4VFwmgdJudAtzIni"/);

  const etiquetaLeaflet = html.match(/<script[^>]+leaflet\.js\?v=1\.9\.4[^>]*><\/script>/i)?.[0] ?? "";
  assert.notEqual(etiquetaLeaflet, "", "la carga de Leaflet debe ser una etiqueta script cerrada");
  assert.doesNotMatch(etiquetaLeaflet, /\b(?:async|defer|type)\s*=/i, "la carga clásica debe terminar antes de evaluar módulos");
  assert.doesNotMatch(html, /<(?:script|link)\b[^>]*(?:https?:)?\/\//i, "el portal no puede cargar scripts ni estilos desde CDN");
});

test("la distribución no incorpora referencias de ejecución a una CDN", async () => {
  const [javascript, estilos] = await Promise.all([
    readFile(new URL("leaflet.js", directorioLeaflet), "utf8"),
    readFile(new URL("leaflet.css", directorioLeaflet), "utf8"),
  ]);
  const activos = `${javascript}\n${estilos}`;

  assert.match(javascript, /t\.version="1\.9\.4"/);
  assert.match(javascript, /window\.L=t/);
  assert.doesNotMatch(activos, /(?:unpkg|cdnjs|jsdelivr|leafletjs-cdn\.s3)\b/i);
  assert.doesNotMatch(activos, /(?:fetch|XMLHttpRequest|WebSocket)\s*\([^)]*["']https?:\/\//i);

  for (const imagen of [
    "layers.png",
    "layers-2x.png",
    "marker-icon.png",
  ]) {
    assert.match(estilos, new RegExp(`url\\(images/${imagen.replaceAll(".", "\\.")}\\)`));
  }
  assert.match(javascript, /marker-icon-2x\.png/);
  assert.match(javascript, /marker-shadow\.png/);
});
