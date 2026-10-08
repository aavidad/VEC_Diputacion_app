import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { moduloDeVistaPortal } from "./portal-modulos-coordinador.js?v=20261008-alta-rpt-circular-v5";
import { traducirPortal } from "./portal-i18n.js?v=20261001-ct-a-i18n-v1";

// Recorrido en cidonia del 06/10/2026 con tecnico_rrhh: Personal · Registro,
// Cronos, Mis trámites y Dietas se quedaban para siempre en «Comprobando
// acceso…» bajo el título «Gestión de Bolsas no disponible». Esas vistas no
// son de Bolsa: dicen ya «no disponible» con su propio título.
const fuente = await readFile(new URL("portal.js", import.meta.url), "utf8");

function extraer(nombre) {
  const inicio = fuente.indexOf(`function ${nombre}(`);
  assert.ok(inicio >= 0, nombre);
  const fin = fuente.indexOf("\n}\n", inicio);
  return fuente.slice(inicio, fin + 2);
}

const escaparHTML = (v) => String(v).replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;");
const encabezadoVista = (_s, titulo) => `<header><h2>${escaparHTML(titulo)}</h2></header>`;
const renderizarFuenteNoDisponible = () => "FUENTE_BOLSA";
const crear = new Function("escaparHTML", "encabezadoVista", "traducirPortal", "moduloDeVistaPortal", "renderizarFuenteNoDisponible",
  `${extraer("renderizarVistaNoDisponible")}\n${extraer("renderizarNoDisponibleDeVista")}\nreturn renderizarNoDisponibleDeVista;`);
const renderizar = crear(escaparHTML, encabezadoVista, traducirPortal, moduloDeVistaPortal, renderizarFuenteNoDisponible);

test("las vistas de otros módulos dicen «no disponible» con su título, sin quedarse comprobando ni hablar de Bolsa", () => {
  for (const vista of ["personal-registro", "personal", "cronos", "cronos-permisos", "mis-tramites", "dietas"]) {
    const html = renderizar(vista, "Registro de Personal");
    assert.match(html, /<h2>Registro de Personal<\/h2>/u, vista);
    assert.match(html, /Esta pantalla no está disponible/u, vista);
    assert.match(html, /avise a Informática/u, vista);
    assert.match(html, /data-vista="portal"/u, vista);
    assert.doesNotMatch(html, /Comprobando|Bolsas|FUENTE_BOLSA/u, vista);
  }
});

test("las vistas de Bolsa siguen con su propia pantalla de fuente no disponible", () => {
  assert.equal(renderizar("resumen", "Cuadro de mando"), "FUENTE_BOLSA");
});

test("renderizar usa la elección por módulo en los dos casos de vista no disponible", () => {
  const cuerpo = extraer("renderizar");
  assert.equal(cuerpo.match(/renderizarNoDisponibleDeVista\(estado\.vista, titulo\)/gu)?.length, 2);
  assert.doesNotMatch(cuerpo, /renderizarFuenteNoDisponible\(\)/u);
});
