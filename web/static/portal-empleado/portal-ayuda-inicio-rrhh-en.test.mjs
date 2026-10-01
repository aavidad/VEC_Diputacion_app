import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

globalThis.location = { href: "https://vec.example/portal-empleado/?lang=en" };
const { IDIOMA_ACTUAL } = await import("../comun/idioma.js");
const MENSAJES_AYUDA_EN = JSON.parse(await readFile(new URL("../textos/en/portal-ayuda.json", import.meta.url), "utf8")).ayuda;
const { traducirPortal } = await import("./portal-i18n.js");
const { AYUDA_PORTAL_RRHH } = await import("./ayuda-contenido.js");

test("?lang=en presenta toda la ayuda contextual RRHH en inglés con el traductor común", () => {
  assert.equal(IDIOMA_ACTUAL, "en");
  for (const [clave, esperado] of Object.entries(MENSAJES_AYUDA_EN)) {
    if (esperado.includes("{")) continue;
    assert.equal(traducirPortal(clave), esperado, clave);
  }
  assert.equal(traducirPortal("ayuda_abrir_contextual", { contexto: "Human Resources" }), "Open help for Human Resources");
  assert.equal(traducirPortal("ayuda_paso_de", { actual: 2, total: 4 }), "Step 2 of 4");
  assert.equal(AYUDA_PORTAL_RRHH.titulo, "Human Resources home help");
  assert.equal(AYUDA_PORTAL_RRHH.pasos.length, 4);
  assert.equal(AYUDA_PORTAL_RRHH.preguntas.length, 3);
  const visible = [AYUDA_PORTAL_RRHH.titulo, AYUDA_PORTAL_RRHH.introduccion,
    ...AYUDA_PORTAL_RRHH.pasos, ...AYUDA_PORTAL_RRHH.preguntas.flatMap((item) => [item.pregunta, item.respuesta]),
    AYUDA_PORTAL_RRHH.transcripcion].join(" ");
  assert.match(visible, /Cases|cases/u);
  assert.match(visible, /Job pools/u);
  assert.match(visible, /Offers to SAE/u);
  assert.match(visible, /does not record/u);
  assert.doesNotMatch(visible, /Ayuda|expedientes|Bolsas de trabajo|Ofertas al SAE|llamamiento/u);
});
