import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { MENSAJES_NOMINAS_ES, crearTraductorNominas } from "./i18n.js";

assert.equal(crearTraductorNominas()("titulo"), "Nóminas y retribuciones");
assert.match(MENSAJES_NOMINAS_ES.explicacion_no_configurado, /No se muestran importes, pagos ni recibos de ejemplo/);
assert.match(MENSAJES_NOMINAS_ES.explicacion_vacio, /no acredita un importe cero ni un pago/);
assert.match(MENSAJES_NOMINAS_ES.certificados_pendientes, /No se generan certificados/);

const vista = await readFile(new URL("./vista.js", import.meta.url), "utf8");
assert.doesNotMatch(vista, /datos-presentacion|datos-sinteticos|localStorage|sessionStorage|document\.cookie|fetch\(/);
assert.match(vista, /fuente\.consultar\(\{ signal: controlador\.signal \}\)/);
assert.match(vista, /descarga\.disabled = !recibo\.descargable \|\| typeof fuente\?\.descargar !== "function"/);

const css = await readFile(new URL("./nominas.css", import.meta.url), "utf8");
assert.match(css, /\.nominas-principal \{ display: grid; grid-template-columns:/);
assert.match(css, /\.nominas-tabla \{ max-width: 100%; max-height: min\(45vh, 420px\); min-width: 0; overflow: auto;/);
assert.match(css, /@media \(max-width: 900px\)/);
assert.match(css, /@media \(max-width: 620px\)/);
console.log("nominas presentation tests: ok");
