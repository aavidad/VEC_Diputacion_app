import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { MENSAJES_AUDITORIA_ES, crearTraductorAuditoria } from "./i18n.js";

test("el catálogo de Auditoría cubre los estados cerrados, el alcance y la ayuda", async () => {
  const fuente = await readFile(new URL("./vista.js", import.meta.url), "utf8");
  const claves = [...fuente.matchAll(/\bt\("([a-z_]+)"\)/gu)].map((match) => match[1]);
  for (const clave of claves) assert.equal(typeof MENSAJES_AUDITORIA_ES[clave], "string", clave);
  for (const clave of ["estado_denegado", "estado_no_configurado", "ayuda_abierta", "ayuda_cerrada"])
    assert.equal(typeof MENSAJES_AUDITORIA_ES[clave], "string", clave);
  const t = crearTraductorAuditoria();
  assert.throws(() => t("clave_inexistente"), /no definido/u);
});
