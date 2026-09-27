import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { MENSAJES_AUDITORIA_ES, crearTraductorAuditoria } from "./i18n.js";

test("las etiquetas visibles proceden del catálogo de Auditoría", async () => {
  const fuente = await readFile(new URL("./vista.js", import.meta.url), "utf8");
  for (const [, clave] of fuente.matchAll(/\bt\("([a-z_]+)"\)/gu)) {
    assert.equal(typeof MENSAJES_AUDITORIA_ES[clave], "string", clave);
  }
  for (const estado of ["no_configurado", "esperando", "cargando", "disponible", "vacio", "denegado", "error", "invalido"]) {
    assert.equal(typeof MENSAJES_AUDITORIA_ES[`estado_${estado}`], "string");
  }
  assert.equal(crearTraductorAuditoria()("pagina", { numero: 3 }), "Página 3");
  assert.throws(() => crearTraductorAuditoria()("desconocida"), /no definido/u);
});
