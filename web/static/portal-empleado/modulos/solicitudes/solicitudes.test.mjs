import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { DATOS_SOLICITUDES_PRESENTACION } from "./datos-presentacion.js";
import { renderizarSolicitudes } from "./vista.js";

test("la demostración de solicitudes muestra datos sintéticos plausibles y límites de entrega", () => {
  const html = renderizarSolicitudes();
  assert.match(html, /Antonio López Fernández/);
  assert.match(html, /Pendiente de conexión/);
  assert.match(html, /Registro: alta idempotente y recibo/);
  assert.match(html, /Auditoría: trazabilidad de lecturas y efectos/);
  assert.match(html, /Firma: firma electrónica y verificación/);
  assert.match(html, /Sin llamadas de red, cookies ni almacenamiento web/);
  assert.match(html, /data-solicitudes-tab="certificados"/);
  assert.doesNotMatch(html, /localStorage|sessionStorage|document\.cookie|javascript:/i);
});

test("bandeja, filtro y seguimiento son locales; los efectos están deshabilitados", () => {
  const html = renderizarSolicitudes({ filtro: "Pendiente de subsanación", busqueda: "permiso" });
  assert.match(html, /SOL-2026-00167/);
  assert.doesNotMatch(html, /SOL-2026-00184/);
  const certificados = renderizarSolicitudes({ pestana: "certificados" });
  assert.match(certificados, /Emitir \/ descargar[^>]*<\/button>/);
  assert.match(certificados, /disabled aria-disabled="true"/);
  assert.equal(DATOS_SOLICITUDES_PRESENTACION.tramites.length, 4);
});

test("la implementación declara el montaje cancelable y el CSS conserva adaptación móvil", async () => {
  const [vista, css] = await Promise.all([readFile(new URL("vista.js", import.meta.url), "utf8"), readFile(new URL("solicitudes.css", import.meta.url), "utf8")]);
  assert.match(vista, /export function montarVistaSolicitudes/);
  assert.match(vista, /removeEventListener/);
  assert.match(css, /@media \(max-width: 780px\)/);
  assert.match(css, /@media \(forced-colors: active\)/);
});
