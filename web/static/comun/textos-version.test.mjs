import assert from "node:assert/strict";
import test from "node:test";
import { huellaTextos, leerPorRed, URL_INDICE_IDIOMAS } from "./idioma.js";

// La huella de los catálogos no se escribe en el código: el servidor la calcula
// y la anuncia en la cabecera del índice de idiomas. El navegador solo la toma
// de ahí y solo si tiene la forma esperada.
function respuesta(huella) {
  const cabeceras = new Map([["X-VEC-Huella-Textos", huella]]);
  return { ok: true, status: 200, headers: { get: (n) => cabeceras.get(n) ?? null }, text: async () => "{}" };
}

test("la huella de los catálogos llega con el índice de idiomas", async () => {
  // leerPorRed exige el mismo origen que el módulo; en Node es file:, así que se simula esa URL.
  await leerPorRed(new URL(URL_INDICE_IDIOMAS.href), async () => respuesta("0123456789abcdef"));
  assert.equal(huellaTextos(), "0123456789abcdef");
  await leerPorRed(new URL(URL_INDICE_IDIOMAS.href), async () => respuesta("<script>"));
  assert.equal(huellaTextos(), "", "una cabecera inválida no fija huella");
});

test("otro recurso no cambia la huella", async () => {
  await leerPorRed(new URL(URL_INDICE_IDIOMAS.href), async () => respuesta("fedcba9876543210"));
  await leerPorRed(new URL("../textos/es/portal.json", URL_INDICE_IDIOMAS), async () => respuesta("0000000000000000"));
  assert.equal(huellaTextos(), "fedcba9876543210");
});
