import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { obtenerDatosAprobacionesPresentacion } from "./datos-presentacion.js";
import { MENSAJES_APROBACIONES_ES, crearTraductorAprobaciones } from "./i18n.js";

const datos = obtenerDatosAprobacionesPresentacion();
assert.equal(datos.estado, "visual_pendiente_backend");
assert.equal(datos.aviso, "Datos ficticios de presentación");
assert.equal(datos.pendientes.length, 4);
assert.ok(datos.pendientes.every((item) => item.documentos.length > 0 && item.revisiones.length > 0 && item.autoridad.length > 0));

const vista = await readFile(new URL("./vista.js", import.meta.url), "utf8");
const css = await readFile(new URL("./aprobaciones.css", import.meta.url), "utf8");
const t = crearTraductorAprobaciones();
assert.equal(t("referencia_ejemplo", { referencia: "REC-APR-SINT-0148" }), "Referencia ficticia: REC-APR-SINT-0148. No existe recibo ni auditoría durable.");
assert.match(t("separacion_funciones"), /aprobar no firma/u);
assert.match(t("motivo_aprobar"), /autorización/u);
for (const estado of ["cargando", "disponible", "vacio", "no_configurado", "denegado", "error"]) assert.ok(t(`estado_${estado}`));
for (const estado of ["cargando", "vacio", "denegado", "error"]) assert.ok(t(`mensaje_${estado}`));
assert.throws(() => t("clave_inexistente"), /no definida/u);
const clavesUsadas = [...vista.matchAll(/t\("([a-z_]+)"/gu)].map((resultado) => resultado[1]);
assert.ok(clavesUsadas.length > 35);
assert.ok(clavesUsadas.every((clave) => Object.hasOwn(MENSAJES_APROBACIONES_ES, clave)));

assert.match(vista, /export function montarVistaAprobaciones/u);
assert.match(vista, /registrarDesmontar\?\.\(desmontar\)/u);
assert.match(vista, /renderizarEstadoEntrega/u);
assert.match(vista, /estadoVista = "no_configurado"/u);
assert.match(vista, /selector\.disabled = true; campo\.disabled = true; aplicar\.disabled = true/u);
assert.match(vista, /disabled = true/u);
assert.match(vista, /bloque.append\(b, crear\(d, "small", motivo\)\)/u);
assert.match(vista, /seleccion = encontrados\[0\] \?\? null/u);
assert.match(vista, /if \(!seleccion\)/u);
assert.doesNotMatch(vista, /fetch\(|localStorage|sessionStorage|indexedDB|document\.cookie/u);
assert.match(css, /\.aprobaciones-principal\s*\{[^}]*grid-template-columns:\s*minmax\(0,\s*2fr\) minmax\(270px,\s*\.7fr\)/u);
assert.match(css, /\.aprobaciones-tabla\s*\{[^}]*overflow-x:\s*auto/u);
assert.match(css, /@media \(max-width: 520px\)/u);
assert.doesNotMatch(css, /#[0-9a-f]{3,8}\b/iu);
console.log("aprobaciones presentation tests: ok");
