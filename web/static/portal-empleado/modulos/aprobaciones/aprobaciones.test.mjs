import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { MENSAJES_APROBACIONES_ES, crearTraductorAprobaciones } from "./i18n.js";

const vista = await readFile(new URL("./vista.js", import.meta.url), "utf8");
const css = await readFile(new URL("./aprobaciones.css", import.meta.url), "utf8");
const t = crearTraductorAprobaciones();

assert.match(t("descripcion"), /aprobación.*firma electrónica.*distinta/u);
assert.match(t("mensaje_no_configurado"), /cero registros/u);
assert.match(t("separacion_funciones"), /Quien prepara no queda autorizado/u);
assert.match(t("motivo_aprobar"), /autorización/u);
for (const estado of ["cargando", "disponible", "vacio", "no_configurado", "denegado", "error"]) assert.ok(t(`estado_${estado}`));
for (const estado of ["cargando", "vacio", "no_configurado", "denegado", "error"]) assert.ok(t(`mensaje_${estado}`));
assert.throws(() => t("clave_inexistente"), /no definida/u);
const clavesUsadas = [...vista.matchAll(/t\("([a-z_]+)"/gu)].map((resultado) => resultado[1]);
assert.ok(clavesUsadas.length > 20);
assert.ok(clavesUsadas.every((clave) => Object.hasOwn(MENSAJES_APROBACIONES_ES, clave)));

assert.match(vista, /export function montarVistaAprobaciones/u);
assert.match(vista, /registrarDesmontar\?\.\(desmontar\)/u);
assert.match(vista, /estadoVista = "no_configurado"/u);
assert.match(vista, /const resumen = \[[\s\S]*"0"[\s\S]*"0"[\s\S]*"0"/u);
assert.match(vista, /selector\.disabled = true/u);
assert.match(vista, /campo\.disabled = true/u);
assert.match(vista, /aplicar\.disabled = true/u);
assert.match(vista, /boton\.disabled = true/u);
assert.match(vista, /renderizarEstadoEntrega/u);
assert.doesNotMatch(vista, /datos-presentacion|obtenerDatosAprobacionesPresentacion|REC-APR|datos\.pendientes|datos\.suplencias/u);
assert.doesNotMatch(vista, /fetch\(|localStorage|sessionStorage|indexedDB|document\.cookie/u);
assert.doesNotMatch(vista, /Elena|María|Antonio|Lucía|Javier|APR-2026/u);
assert.doesNotMatch(JSON.stringify(MENSAJES_APROBACIONES_ES), /fictici|ejemplo|presentaci[oó]n/iu);
assert.match(css, /\.aprobaciones-principal\s*\{[^}]*grid-template-columns:\s*minmax\(0,\s*2fr\) minmax\(270px,\s*\.7fr\)/u);
assert.match(css, /@media \(max-width: 520px\)/u);
assert.doesNotMatch(css, /#[0-9a-f]{3,8}\b/iu);
console.log("aprobaciones sin datos sintéticos: ok");
