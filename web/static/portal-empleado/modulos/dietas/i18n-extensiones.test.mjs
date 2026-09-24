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
  const claveConsulta = "borradores_propios_consultar_registrados";
  const claveEstado = "borradores_propios_estado_sin_seleccion";
  assert.equal(crearTraductorDietas()(claveConsulta), "Consultar borradores registrados");
  assert.equal(MENSAJES_DIETAS_ES[claveEstado], "Ningún borrador seleccionado.");
  assert.equal(crearTraductorBorradoresDietas((clave) => clave)(claveEstado), "Ningún borrador seleccionado.");
  assert.equal(crearTraductorBorradoresDietas(crearTraductorDietas())(claveEstado), "Ningún borrador seleccionado.");
  assert.match(crearTraductorDietas()("borradores_propios_consulta_registrados"), /La lista no confirma altas anteriores/u);
  assert.match(crearTraductorDietas()("borradores_propios_consulta_ayuda"), /Elija un borrador para ver su recibo/u);
  const [comun, borradores, recorridos] = await Promise.all([
    readFile(new URL("./i18n.js", import.meta.url), "utf8"),
    readFile(new URL("./vista-borradores-propios.js", import.meta.url), "utf8"),
    readFile(new URL("./vista-recorridos.js", import.meta.url), "utf8"),
  ]);
  assert.match(comun, /i18n-borradores\.js\?v=20260924-osm-base-v3/u);
  assert.match(borradores, /i18n\.js\?v=20260924-osm-base-v3/u);
  assert.match(borradores, /i18n-borradores\.js\?v=20260924-osm-base-v3/u);
  assert.match(recorridos, /vista-borradores-propios\.js\?v=20260924-osm-base-v3/u);
  assert.doesNotMatch(comun, /i18n-borradores\.js\?v=20260924-f2-consulta-v1/u);
  assert.doesNotMatch(borradores, /(?:i18n|i18n-borradores)\.js\?v=20260924-f2-consulta-v1/u);
  assert.doesNotMatch(recorridos, /vista-borradores-propios\.js\?v=20260924-f2-consulta-v1|vista-borradores-propios\.js\?v=20260924-dietas-recuperacion-v3/u);
  for (const fuente of [comun, borradores, recorridos])
    assert.doesNotMatch(fuente, /\?v=20260924-dietas-d1d2d4|\?v=20260924-dietas-ayuda-sin-guia-v1|\?v=20260924-osm-base-v1/u);
});

test("D1 y D4 están en el catálogo común, con sus importaciones de versión renovadas", async () => {
  for (const clave of ["d1_titulo", "d1_intervencion", "d4_vehiculo_pendiente", "d4_sin_importe"])
    assert.equal(crearTraductorDietas()(clave), MENSAJES_DIETAS_ES[clave]);
  const [recorridos, papeles, itinerario, d1, d4] = await Promise.all([
    readFile(new URL("./vista-recorridos.js", import.meta.url), "utf8"),
    readFile(new URL("./vista-acceso-papeles.js", import.meta.url), "utf8"),
    readFile(new URL("./vista-itinerario.js", import.meta.url), "utf8"),
    readFile(new URL("./i18n-d1.js", import.meta.url), "utf8"),
    readFile(new URL("./i18n-d4.js", import.meta.url), "utf8"),
  ]);
  assert.match(recorridos, /i18n\.js\?v=20260924-osm-base-v3/u);
  assert.match(recorridos, /vista-acceso-papeles\.js\?v=20260924-osm-base-v3/u);
  assert.match(papeles, /i18n-d1\.js\?v=20260924-osm-base-v3/u);
  assert.match(itinerario, /i18n\.js\?v=20260924-osm-base-v3/u);
  assert.match(itinerario, /i18n-d4\.js\?v=20260924-osm-base-v3/u);
  for (const fuente of [d1, d4])
    assert.match(fuente, /i18n\.js\?v=20260924-osm-base-v3/u);
  for (const fuente of [recorridos, papeles, itinerario, d1, d4])
    assert.doesNotMatch(fuente, /\?v=20260924-dietas-d1d2d4|\?v=20260924-dietas-ayuda-sin-guia-v1|\?v=20260924-osm-base-v1/u);
});
