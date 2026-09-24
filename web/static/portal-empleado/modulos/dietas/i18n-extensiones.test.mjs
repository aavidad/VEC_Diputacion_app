import assert from "node:assert/strict";
import test from "node:test";
import { crearTraductorDietas, MENSAJES_DIETAS_ES } from "./i18n.js";
import { crearTraductorBorradoresDietas } from "./i18n-borradores.js";
import { crearTraductorRevisionDietas } from "./i18n-revision.js";

test("las claves nuevas pertenecen al catálogo común y respetan el traductor inyectado", () => {
  const claveBorrador = "borradores_propios_titulo_registrados";
  const claveRevision = "revision_mis_comisiones";
  assert.equal(typeof MENSAJES_DIETAS_ES[claveBorrador], "string");
  assert.equal(typeof MENSAJES_DIETAS_ES[claveRevision], "string");
  const traducir = (clave) => ({ [claveBorrador]: "Registered drafts", [claveRevision]: "My commissions" }[clave] ?? clave);
  assert.equal(crearTraductorBorradoresDietas(traducir)(claveBorrador), "Registered drafts");
  assert.equal(crearTraductorRevisionDietas(traducir)(claveRevision), "My commissions");
  assert.equal(crearTraductorDietas()(claveBorrador), MENSAJES_DIETAS_ES[claveBorrador]);
});
