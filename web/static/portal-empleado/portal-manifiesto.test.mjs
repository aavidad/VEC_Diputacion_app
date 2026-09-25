import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

// Regresión del 23/09/2026: los módulos de avisos (D3-AV) y de operaciones (B8)
// se importaban desde el portal pero no figuraban en los manifiestos, y un
// paquete construido desde ellos habría salido sin esas pantallas. Todo import
// estático de un fichero empaquetado debe estar empaquetado también. Los
// import() dinámicos quedan fuera: alguno se excluye del paquete a propósito.
const raizWeb = new URL("../../", import.meta.url);

async function entradas(nombre) {
  const texto = await readFile(new URL(nombre, raizWeb), "utf8");
  return new Set(texto.split(/\r?\n/).map((linea) => linea.trim()).filter((linea) => linea && !linea.startsWith("#")));
}

function importsEstaticos(codigo) {
  const rutas = [];
  for (const [, ruta] of codigo.matchAll(/^\s*import\s[^;]*?\sfrom\s+["']([^"']+)["']/gmu)) rutas.push(ruta);
  for (const [, ruta] of codigo.matchAll(/^\s*import\s+["']([^"']+)["']/gmu)) rutas.push(ruta);
  return rutas.filter((ruta) => ruta.startsWith("./") || ruta.startsWith("../"));
}

for (const manifiesto of ["produccion.manifest", "interno.manifest"]) {
  test(`${manifiesto}: cada import estático de un módulo empaquetado también está empaquetado`, async () => {
    const empaquetados = await entradas(manifiesto);
    const faltan = [];
    for (const entrada of empaquetados) {
      if (!entrada.startsWith("static/portal-empleado/") || !/\.m?js$/u.test(entrada) || /\.test\.m?js$/u.test(entrada)) continue;
      const codigo = await readFile(new URL(entrada, raizWeb), "utf8");
      const carpeta = entrada.slice(0, entrada.lastIndexOf("/") + 1);
      for (const ruta of importsEstaticos(codigo)) {
        const destino = new URL(ruta.split("?")[0], new URL(carpeta, "file:///")).pathname.slice(1);
        // Excepción conocida: los datos de ejemplo de los módulos congelados (R5) no
        // se empaquetan por diseño, aunque sus vistas los importen. Es deuda de esos
        // módulos, anotada aparte; no se amplía a ningún otro fichero.
        if (/\/modulos\/[a-z-]+\/datos-presentacion\.js$/u.test(destino)) continue;
        if (!empaquetados.has(destino)) faltan.push(`${entrada} → ${destino}`);
      }
    }
    assert.deepEqual(faltan, [], `módulos importados sin empaquetar en ${manifiesto}`);
  });
}

// Recorrido en Chrome del 25/09/2026 (fallo 1): el coordinador importaba con
// import() `modulos/personal/cliente-http-ficha-propia.js`, que no figuraba en
// produccion.manifest: el paquete desplegado respondía 404 en cada Inicio. Todo
// import() del coordinador (y sus imports estáticos) debe ir en el paquete.
for (const manifiesto of ["produccion.manifest", "interno.manifest"]) {
  test(`${manifiesto}: cada import() del coordinador de módulos está empaquetado con sus dependencias`, async () => {
    const empaquetados = await entradas(manifiesto);
    const entrada = "static/portal-empleado/portal-modulos-coordinador.js";
    assert.ok(empaquetados.has(entrada));
    const codigo = await readFile(new URL(entrada, raizWeb), "utf8");
    const pendientes = [...codigo.matchAll(/import\(\s*["']([^"']+)["']\s*\)/gu)]
      .map(([, ruta]) => new URL(ruta.split("?")[0], new URL("static/portal-empleado/", "file:///")).pathname.slice(1));
    assert.ok(pendientes.length > 10, "el coordinador carga sus módulos con import()");
    const faltan = [];
    const vistos = new Set();
    while (pendientes.length > 0) {
      const destino = pendientes.pop();
      if (vistos.has(destino)) continue;
      vistos.add(destino);
      if (/\/modulos\/[a-z-]+\/datos-presentacion\.js$/u.test(destino)) continue;
      if (!empaquetados.has(destino)) { faltan.push(destino); continue; }
      const fuente = await readFile(new URL(destino, raizWeb), "utf8");
      const carpeta = destino.slice(0, destino.lastIndexOf("/") + 1);
      for (const ruta of importsEstaticos(fuente)) {
        pendientes.push(new URL(ruta.split("?")[0], new URL(carpeta, "file:///")).pathname.slice(1));
      }
    }
    assert.deepEqual(faltan, [], `módulos cargados con import() sin empaquetar en ${manifiesto}`);
  });
}
