#!/usr/bin/env node
// Vuelca diccionarios de textos embebidos en JavaScript a un catálogo de datos
// web/static/textos/<idioma>/<modulo>.json, una sección por diccionario, con
// el mismo orden de claves. Sirve para migrar un módulo sin copiar a mano.
//
// Uso:
//   node scripts/extraer_textos.mjs <modulo> <idioma> <seccion>=<ruta.js>#<CONSTANTE> [...]
// Ejemplo:
//   node scripts/extraer_textos.mjs dietas es \
//     general=web/static/portal-empleado/modulos/dietas/i18n.js#MENSAJES_DIETAS_ES
//
// Si el catálogo ya existe, se conservan sus otras secciones y se sustituyen
// las indicadas. Los valores deben ser cadenas u objetos de plurales.
import { mkdir, readFile, writeFile } from "node:fs/promises";
import { pathToFileURL } from "node:url";
import path from "node:path";

const [modulo, idioma, ...pares] = process.argv.slice(2);
if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/u.test(modulo ?? "") || !/^[a-z]{2,3}(?:-[a-z0-9]{2,8})*$/u.test(idioma ?? "") || pares.length === 0) {
  console.error("Uso: node scripts/extraer_textos.mjs <modulo> <idioma> <seccion>=<ruta.js>#<CONSTANTE> [...]");
  process.exit(2);
}

const destino = path.resolve("web/static/textos", idioma, `${modulo}.json`);
let catalogo = {};
try { catalogo = JSON.parse(await readFile(destino, "utf8")); } catch { /* catálogo nuevo */ }

for (const par of pares) {
  const coincidencia = /^([A-Za-z0-9_]+)=(.+\.m?js)#([A-Za-z_$][\w$]*)$/u.exec(par);
  if (!coincidencia) throw new Error(`argumento no válido: ${par}`);
  const [, seccion, ruta, constante] = coincidencia;
  const exportado = (await import(pathToFileURL(path.resolve(ruta)).href))[constante];
  if (!exportado || typeof exportado !== "object") throw new Error(`${ruta} no exporta ${constante}`);
  for (const [clave, valor] of Object.entries(exportado)) {
    const plural = valor && typeof valor === "object" && typeof valor.other === "string";
    if (typeof valor !== "string" && !plural) throw new Error(`${constante}.${clave} no es texto`);
  }
  catalogo[seccion] = { ...exportado };
  console.log(`${seccion}: ${Object.keys(exportado).length} textos de ${constante}`);
}

await mkdir(path.dirname(destino), { recursive: true });
await writeFile(destino, `${JSON.stringify(catalogo, null, 2)}\n`);
console.log(`escrito ${path.relative(process.cwd(), destino)}`);
