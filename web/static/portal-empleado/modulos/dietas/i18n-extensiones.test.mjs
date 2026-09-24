import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
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

test("la consulta y su ayuda están traducidas y sus imports no usan la versión anterior", async () => {
  const clave = "borradores_propios_consultar_registrados";
  assert.equal(crearTraductorDietas()(clave), "Consultar borradores registrados");
  assert.match(crearTraductorDietas()("borradores_propios_consulta_registrados"), /La lista no confirma altas anteriores/u);
  assert.match(crearTraductorDietas()("borradores_propios_consulta_ayuda"), /Elija un borrador para ver su recibo/u);
  const [comun, borradores, recorridos] = await Promise.all([
    readFile(new URL("./i18n.js", import.meta.url), "utf8"),
    readFile(new URL("./vista-borradores-propios.js", import.meta.url), "utf8"),
    readFile(new URL("./vista-recorridos.js", import.meta.url), "utf8"),
  ]);
  assert.match(comun, /i18n-borradores\.js\?v=20260924-f2-consulta-v2/u);
  assert.doesNotMatch(comun, /i18n-borradores\.js\?v=20260924-f2-consulta-v1/u);
  assert.match(borradores, /i18n\.js\?v=20260924-f2-consulta-v2/u);
  assert.match(borradores, /i18n-borradores\.js\?v=20260924-f2-consulta-v2/u);
  assert.doesNotMatch(borradores, /(?:i18n|i18n-borradores)\.js\?v=20260924-f2-consulta-v1/u);
  assert.match(recorridos, /vista-borradores-propios\.js\?v=20260924-f2-consulta-v2/u);
  assert.doesNotMatch(recorridos, /vista-borradores-propios\.js\?v=20260924-f2-consulta-v1/u);
});
