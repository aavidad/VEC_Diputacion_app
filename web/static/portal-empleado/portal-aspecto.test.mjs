import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const raiz = new URL("./", import.meta.url);
const leer = (ruta) => readFile(new URL(ruta, raiz), "utf8");
const [tema, componentes, expedientes, vista, pagina] = await Promise.all([
  leer("portal.css"),
  leer("portal-componentes.css"),
  leer("modulos/contratacion-temporal/expedientes.css"),
  leer("modulos/contratacion-temporal/componentes-expedientes.js"),
  leer("portal.js"),
]);

test("el tema común separa lienzo, panel y cabecera también en alto contraste", () => {
  for (const token of ["fondo", "superficie", "superficie-alterna", "cabecera-panel", "borde", "sombra-sm", "radio-lg"]) {
    assert.match(tema, new RegExp(`--portal-${token}:`));
  }
  assert.match(tema, /data-contraste="true"[\s\S]*--portal-superficie-alterna:[\s\S]*--portal-cabecera-panel:/u);
  assert.match(componentes, /\.panel,[\s\S]*border: 1px solid var\(--portal-borde\)[\s\S]*border-radius: var\(--portal-radio-lg\)/u);
  assert.match(componentes, /\.cabecera-panel[\s\S]*background: var\(--portal-cabecera-panel\)/u);
});

test("tablas, KPI y estados consumen colores semánticos comunes", () => {
  assert.match(componentes, /tbody tr:nth-child\(even\)[\s\S]*var\(--portal-superficie-alterna\)/u);
  assert.match(componentes, /tbody tr \{ height: 44px; \}/u);
  assert.match(componentes, /tbody tr:focus-within/u);
  assert.match(componentes, /\.icono-kpi[\s\S]*border-radius:/u);
  assert.match(componentes, /\.estado-chip::before[\s\S]*background: currentColor/u);
  assert.match(expedientes, /\.ct-exp-indicador::before[\s\S]*border-radius: 50%/u);
  assert.doesNotMatch(expedientes, /#[0-9a-fA-F]{3,8}\b/u);
});

test("los módulos visuales revisados consumen tokens y no fijan hexadecimales", async () => {
  const rutas = [
    "modulos/contratacion-temporal/estadisticas.css",
    "modulos/contratacion-temporal/expedientes-incidencia.css",
    "modulos/documentos/documentos.css",
    "modulos/meritos/meritos.css",
    "modulos/solicitudes/solicitudes.css",
    "modulos/aprobaciones/aprobaciones.css",
  ];
  for (const ruta of rutas) {
    const css = await leer(ruta);
    assert.doesNotMatch(css, /#[0-9a-fA-F]{3,8}\b/u, ruta);
    assert.match(css, /var\(--portal-/u, ruta);
  }
});

test("la bandeja conserva el detalle cerrado y explica una sola vez la modalidad ausente", () => {
  assert.match(vista, /ct-exp-nota-tabla/u);
  assert.match(vista, /modalidadAusente \? ` title=/u);
  assert.match(expedientes, /\.ct-exp-numero,[\s\S]*\.ct-exp-fase,[\s\S]*white-space: nowrap/u);
  assert.match(pagina, /!fila\.hasAttribute\('data-ct-exp-resumen-fila'\)/u);
  assert.match(pagina, /detalle\.hidden = fila\.hidden \|\|/u);
});
