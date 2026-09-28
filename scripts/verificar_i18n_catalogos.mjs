import { readFile, readdir } from "node:fs/promises";
import { resolve, relative, join } from "node:path";
import { pathToFileURL } from "node:url";

const raiz = resolve(import.meta.dirname, "../web/static");
const marcador = /\{[a-z_]+\}/giu;
const errores = [];

async function* ficheros(dir) {
  for (const entrada of await readdir(dir, { withFileTypes: true })) {
    if (entrada.name === "vendor") continue;
    const ruta = join(dir, entrada.name);
    if (entrada.isDirectory()) yield* ficheros(ruta);
    else if (entrada.name.endsWith(".js")) yield ruta;
  }
}

for await (const ruta of ficheros(raiz)) {
  const fuente = await readFile(ruta, "utf8");
  const nombres = [...fuente.matchAll(/export const ([A-Z][A-Z_0-9]*_ES)\s*=/gu)].map((coincidencia) => coincidencia[1]);
  if (!nombres.length) continue;
  let modulo;
  try { modulo = await import(pathToFileURL(ruta).href); }
  catch (error) { errores.push(`${relative(raiz, ruta)}: no se puede cargar catálogo: ${error.message}`); continue; }
  for (const nombre of nombres) {
    const par = nombre.slice(0, -3) + "_EN";
    const es = modulo[nombre];
    const en = modulo[par];
    if (!es || !en || typeof es !== "object" || typeof en !== "object") {
      errores.push(`${relative(raiz, ruta)}: falta catálogo ${par}`);
      continue;
    }
    const clavesES = Object.keys(es).sort();
    const clavesEN = Object.keys(en).sort();
    if (JSON.stringify(clavesES) !== JSON.stringify(clavesEN)) {
      errores.push(`${relative(raiz, ruta)}: claves distintas en ${nombre}/${par}`);
      continue;
    }
    for (const clave of clavesES) {
      const original = es[clave], traducido = en[clave];
      if (typeof original !== "string" || typeof traducido !== "string" || !traducido.trim()) {
        errores.push(`${relative(raiz, ruta)}: traducción vacía o inválida en ${par}.${clave}`);
      } else if (JSON.stringify([...original.matchAll(marcador)].map((m) => m[0]).sort()) !== JSON.stringify([...traducido.matchAll(marcador)].map((m) => m[0]).sort())) {
        errores.push(`${relative(raiz, ruta)}: marcadores distintos en ${par}.${clave}`);
      }
    }
  }
}

if (errores.length) {
  for (const error of errores.slice(0, 100)) process.stderr.write(`${error}\n`);
  process.stderr.write(`i18n JS: ${errores.length} incumplimientos\n`);
  process.exitCode = 1;
} else {
  process.stdout.write("i18n JS: catálogos ES/EN simétricos\n");
}
