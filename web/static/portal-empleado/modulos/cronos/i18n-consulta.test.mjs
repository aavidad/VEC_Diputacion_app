import "./test-preparar-textos.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { cargarTextos } from "../../../comun/textos.js";
import { crearTraductorConsultaCronos } from "./i18n-consulta.js?v=20261001-cronos-grafo-bandeja-v5";

test("los catálogos de consulta mantienen claves y variables completas en ambos idiomas", async () => {
  const raiz = new URL("../../../textos/", import.meta.url);
  const indice = JSON.parse(await readFile(new URL("idiomas.json", raiz), "utf8"));
  const textos = await Promise.all(indice.idiomas.map(({ codigo }) => cargarTextos("cronos-consulta", { idioma: codigo, avisar: (aviso) => assert.fail(aviso) })));
  for (const texto of textos) {
    const catalogo = texto.seccion("consulta"); const traducir = crearTraductorConsultaCronos(catalogo);
    assert.deepEqual(Object.keys(catalogo), Object.keys(textos[0].seccion("consulta")));
    const recuento = traducir("recuento", { desde: 1, hasta: 31, total: 65, pagina: 1, paginas: 3 });
    assert.match(recuento, /65/); assert.doesNotMatch(recuento, /\{[a-z]+\}/u);
    assert.match(traducir("periodo_completo", { desde: "A", hasta: "B" }), /A.*B/u);
    assert.throws(() => traducir("inexistente"), TypeError);
  }
  assert.throws(() => crearTraductorConsultaCronos({}), TypeError);
});
