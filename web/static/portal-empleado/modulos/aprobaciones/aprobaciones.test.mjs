import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { normalizarConsultaAprobaciones } from "./vista.js";
import { MENSAJES_APROBACIONES_ES, crearTraductorAprobaciones } from "./i18n.js?v=20260924-f2-web2";

const pendiente = {
  referencia: " APR-1 ", modulo: "Dietas", tipo: "Liquidación", solicitante: "Persona de prueba",
  estado: "pendiente", prioridad: "alta", resumen: "Revisión", recibido: "2026-09-24T08:00:00Z",
  evidencias: ["documento-ref-1"],
};
const lectura = normalizarConsultaAprobaciones({ estado: "disponible", pendientes: [pendiente] });
assert.equal(lectura.estado, "disponible");
assert.equal(lectura.pendientes.length, 1);
assert.equal(lectura.pendientes[0].referencia, "APR-1");
assert.equal(lectura.pendientes[0].evidencias[0], "documento-ref-1");
assert.match(lectura.pendientes[0].recibido, /24\/9\/26|24\/09\/2026/u);
assert.equal(normalizarConsultaAprobaciones({ estado: "disponible", pendientes: [] }).estado, "vacio");
assert.equal(normalizarConsultaAprobaciones({ estado: "denegado" }).pendientes.length, 0);
assert.throws(() => normalizarConsultaAprobaciones({ estado: "denegado", pendientes: [pendiente] }), /pendientes/u);
assert.throws(() => normalizarConsultaAprobaciones({ estado: "disponible", pendientes: [pendiente, pendiente] }), /duplicadas/u);
assert.throws(() => normalizarConsultaAprobaciones({ estado: "disponible", pendientes: [{ ...pendiente, prioridad: "urgente" }] }), /estado o prioridad/u);
assert.throws(() => normalizarConsultaAprobaciones({ estado: "disponible", pendientes: [{ ...pendiente, recibido: "ayer" }] }), /recibido/u);
assert.throws(() => normalizarConsultaAprobaciones({ estado: "disponible", pendientes: [{ ...pendiente, plazo: "2026-02-30" }] }), /plazo/u);
assert.throws(() => normalizarConsultaAprobaciones({ estado: "disponible", pendientes: [{ ...pendiente, evidencias: Array(11).fill("x") }] }), /evidencias/u);

const vista = await readFile(new URL("./vista.js", import.meta.url), "utf8");
assert.match(vista, /from "\.\/i18n\.js\?v=20260924-f2-web2"/u);
const css = await readFile(new URL("./aprobaciones.css", import.meta.url), "utf8");
const t = crearTraductorAprobaciones();
assert.match(t("descripcion"), /aprobación.*firma electrónica.*distinta/u);
assert.match(t("mensaje_no_configurado"), /cero registros/u);
assert.match(t("separacion_funciones"), /Quien prepara no queda autorizado/u);
for (const estado of ["cargando", "disponible", "vacio", "no_configurado", "denegado", "error"]) assert.ok(t(`estado_${estado}`));
for (const estado of ["pendiente", "vencida", "bloqueada", "subsanacion", "reasignada"]) assert.ok(t(`estado_item_${estado}`));
assert.throws(() => t("clave_inexistente"), /no definida/u);
const clavesUsadas = [...vista.matchAll(/t\("([a-z_]+)"/gu)].map((resultado) => resultado[1]);
assert.ok(clavesUsadas.length > 20);
assert.ok(clavesUsadas.every((clave) => Object.hasOwn(MENSAJES_APROBACIONES_ES, clave) || ["prioridad_", "estado_item_", "estado_"].includes(clave)));
assert.match(vista, /fuentePendientes\.consultarPendientes\(\{ signal: controlador\.signal \}\)/u);
assert.match(vista, /controlador\?\.abort\(\)/u);
assert.match(vista, /boton\.disabled = true/u);
assert.match(vista, /registrarDesmontar\?\.\(desmontar\)/u);
assert.doesNotMatch(vista, /estadoVista = "no_configurado"|datos-presentacion|fetch\(|localStorage|sessionStorage|indexedDB|document\.cookie/u);
assert.doesNotMatch(JSON.stringify(MENSAJES_APROBACIONES_ES), /fictici|ejemplo/iu);
assert.match(css, /\.aprobaciones-principal\s*\{[^}]*grid-template-columns:\s*minmax\(0,\s*2fr\) minmax\(270px,\s*\.7fr\)/u);
assert.match(css, /@media \(max-width: 520px\)/u);
assert.doesNotMatch(css, /#[0-9a-f]{3,8}\b/iu);
console.log("aprobaciones lectura inyectada: ok");
