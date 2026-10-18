import assert from "node:assert/strict";
import { readFile, readdir } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

export const MARCADOR_VERSION_WEB = "VEC_WEB_VERSION";

function escapar(texto) {
  return texto.replace(/[.*+?^${}()|[\]\\]/gu, "\\$&");
}

export function versionesDe(fuente, recurso) {
  const patron = new RegExp(`(?<![\\w-])${escapar(recurso)}\\?v=([\\w.-]+)`, "gu");
  return [...fuente.matchAll(patron)].map(([, version]) => version);
}

export function versionDe(fuente, recurso) {
  const versiones = new Set(versionesDe(fuente, recurso));
  assert.equal(versiones.size, 1, `${recurso}: se espera una sola URL versionada`);
  return [...versiones][0];
}

export function exigirMismaVersion(fuentes, recurso) {
  const versiones = fuentes.map((fuente) => versionDe(fuente, recurso));
  assert.equal(new Set(versiones).size, 1, `${recurso}: URL distinta entre importadores`);
  assert.equal(versiones[0], MARCADOR_VERSION_WEB, `${recurso}: versión calculada por el servidor`);
  return versiones[0];
}

const URL_VERSIONADA = /["'`]((?:\.{1,2}\/|\/portal-empleado\/)[^"'`\s?#]+\.(?:js|css))\?v=([\w.-]+)["'`]/gu;

async function archivosDe(directorio) {
  const entradas = await readdir(directorio, { withFileTypes: true });
  const rutas = await Promise.all(entradas.map(async (entrada) => {
    const ruta = path.join(directorio, entrada.name);
    if (entrada.isDirectory()) return archivosDe(ruta);
    return entrada.isFile() && ruta.endsWith(".js") ? [ruta] : [];
  }));
  return rutas.flat();
}

/** Compara las URL literales de un mismo recurso en JS, HTML y caché pública. */
export async function comprobarGrafoVersionado(raizURL) {
  const raiz = fileURLToPath(raizURL);
  const ficheros = [...await archivosDe(raiz), path.join(raiz, "index.html"), path.join(raiz, "cache-publica-v1.json")];
  const destinos = new Map();
  for (const fichero of ficheros) {
    const fuente = await readFile(fichero, "utf8");
    for (const [, recurso, version] of fuente.matchAll(URL_VERSIONADA)) {
      const destino = recurso.startsWith("/portal-empleado/")
        ? recurso.slice("/portal-empleado/".length)
        : path.relative(raiz, path.resolve(path.dirname(fichero), recurso)).split(path.sep).join("/");
      if (destino.startsWith("..")) continue;
      const referencias = destinos.get(destino) ?? [];
      referencias.push({ importador: path.relative(raiz, fichero), version });
      destinos.set(destino, referencias);
    }
  }
  assert.ok(destinos.has("portal.js"), "la entrada figura en HTML y caché");
  assert.ok(destinos.has("portal-i18n.js"), "el catálogo figura en el grafo");
  for (const [destino, referencias] of destinos) {
    const versiones = new Set(referencias.map(({ version }) => version));
    assert.equal(versiones.size, 1, `${destino}: versiones distintas en ${JSON.stringify(referencias)}`);
    assert.equal([...versiones][0], MARCADOR_VERSION_WEB, `${destino}: versión manual en ${JSON.stringify(referencias)}`);
  }
  return destinos;
}
