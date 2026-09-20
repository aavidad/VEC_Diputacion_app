import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { MENSAJES_NOMINAS_ES, crearTraductorNominas } from "./i18n.js";
import { obtenerDatosNominasPresentacion } from "./datos-presentacion.js";

assert.equal(crearTraductorNominas()("titulo"), "Nóminas y retribuciones");
assert.match(MENSAJES_NOMINAS_ES.falta, /fuente de nómina/);
const datos = obtenerDatosNominasPresentacion();
assert.equal(datos.persona, "Antonio López Fernández");
assert.equal(datos.estado, "visual_pendiente_backend");
assert.equal(datos.aviso, "Datos ficticios de presentación");
assert.equal(datos.nominas.length, 3);
assert.ok(datos.nominas.every((n) => n.estado === "Documento de ejemplo"));

const css = await readFile(new URL("./nominas.css", import.meta.url), "utf8");
assert.match(css, /\.nominas-principal > \*, \.nominas-secundarios > \*, \.nominas-detalle, \.nominas-listado \{ min-width: 0; \}/);
assert.match(css, /\.nominas-tabla \{ max-width: 100%; min-width: 0; overflow-x: auto;/);
assert.match(css, /\.nominas-tabla table \{ width: 100%; min-width: max-content;/);
assert.match(css, /@media \(max-width: 800px\) \{[\s\S]*grid-template-columns: minmax\(0, 1fr\)/);
console.log("nominas presentation tests: ok");
