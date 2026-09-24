import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { traducir } from "./i18n.js";

test("Perfil no deduce permiso de un método del cliente ni inicia GET Contacto", async () => {
  const aplicacion = await readFile(new URL("./aplicacion.js", import.meta.url), "utf8");
  assert.doesNotMatch(aplicacion, /\.cargarContactoPropio\s*\(/u);
  assert.doesNotMatch(aplicacion, /typeof estado\.cliente\.cargarContactoPropio/u);
  assert.match(aplicacion, /contactoPropio: null,/u);
  assert.doesNotMatch(aplicacion, /contactoPropio = null, fetchImpl/u);
  assert.equal(traducir("areaPersonal.contacto.noConfigurado").startsWith("Contacto propio no configurado"), true);
});
