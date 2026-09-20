import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { MENSAJES_DOCUMENTOS_ES, crearTraductorDocumentos } from "./i18n.js";
import { obtenerDatosDocumentosPresentacion } from "./datos-presentacion.js";

const directorio = new URL("./", import.meta.url);
const vista = await readFile(new URL("vista.js", directorio), "utf8");
const css = await readFile(new URL("documentos.css", directorio), "utf8");

assert.match(vista, /export function montarVistaDocumentos/);
assert.match(vista, /obtenerDatosDocumentosPresentacion/);
assert.match(vista, /crearTraductorDocumentos/);
assert.match(vista, /renderizarEstadoEntrega/);
assert.equal(crearTraductorDocumentos()("titulo"), "Biblioteca documental y circuito de firma");
assert.match(MENSAJES_DOCUMENTOS_ES.aclaracion_firma, /FNMT/);
for (const clave of ["generar_version", "subir_original", "firmar", "verificar", "enviar", "descargar", "paso_5", "entrega_auditoria"]) assert.equal(typeof MENSAJES_DOCUMENTOS_ES[clave], "string");
assert.equal(obtenerDatosDocumentosPresentacion().atlas.persona_principal.nombre_visible, "Antonio López Fernández");
assert.equal(obtenerDatosDocumentosPresentacion().documentos.length, 4);
const clavesUsadas = [...vista.matchAll(/t\("([a-z0-9_]+)"/g)].map((coincidencia) => coincidencia[1]);
assert.ok(clavesUsadas.length > 30);
for (const clave of clavesUsadas) assert.equal(typeof MENSAJES_DOCUMENTOS_ES[clave], "string", `falta la clave ${clave}`);
assert.doesNotMatch(`${vista}\n${css}`, /fetch\(|XMLHttpRequest|document\.cookie|localStorage|sessionStorage|indexedDB/i);
assert.match(css, /@media\(max-width:850px\)/);
console.log("documentos presentation tests: ok");
