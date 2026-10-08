import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { hoyCivilCronos } from "./fecha-civil.js";

test("la fecha civil conserva el día de Madrid en ambos cambios de hora", () => {
  assert.equal(hoyCivilCronos("Europe/Madrid", new Date("2026-03-28T23:30:00Z")), "2026-03-29");
  assert.equal(hoyCivilCronos("Europe/Madrid", new Date("2026-03-29T01:30:00Z")), "2026-03-29");
  assert.equal(hoyCivilCronos("Europe/Madrid", new Date("2026-10-25T00:30:00Z")), "2026-10-25");
  assert.equal(hoyCivilCronos("Europe/Madrid", new Date("2026-10-25T01:30:00Z")), "2026-10-25");
});

test("abrir Permisos no evalúa el calendario ni el catálogo de incidencias", async () => {
  const pendientes = [new URL("./vista-permisos-propios.js", import.meta.url)];
  const visitados = new Set();
  while (pendientes.length) {
    const url = pendientes.pop();
    if (visitados.has(url.pathname)) continue;
    visitados.add(url.pathname);
    const fuente = await readFile(url, "utf8");
    for (const match of fuente.matchAll(/\b(?:import|export)\s+(?:[\s\S]*?\s+from\s+)?["'](\.\/[^"']+)["']/gu)) {
      pendientes.push(new URL(match[1], url));
    }
  }
  assert.equal(visitados.has(new URL("./vista-movimientos-propios.js", import.meta.url).pathname), false);
  assert.equal(visitados.has(new URL("./i18n-incidencias.js", import.meta.url).pathname), false);
});
