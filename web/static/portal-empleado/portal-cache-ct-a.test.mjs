import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { exigirVersiones, posterior } from "./versiones-cache.test-helper.mjs";

const versionEntradaAnterior = "20261002-r1-post401-v4";
const versionCoordinador = "20261005-b-contacto-v3";
const raiz = new URL("./", import.meta.url);

test("la extracción CT renueva cada padre hasta la entrada del portal", async () => {
  const [html, portal, expediente] = await Promise.all([
    readFile(new URL("index.html", raiz), "utf8"),
    readFile(new URL("portal.js", raiz), "utf8"),
    readFile(new URL("modulos/contratacion-temporal/vista-expedientes.js", raiz), "utf8"),
  ]);
  const versionEntrada = exigirVersiones(html, "/portal-empleado/portal.js", posterior(versionEntradaAnterior));
  exigirVersiones(html, "/portal-empleado/portal-modulos-coordinador.js", versionCoordinador);
  exigirVersiones(portal, "./portal-modulos-coordinador.js", versionCoordinador);
  exigirVersiones(expediente, "./seguimiento-cese.js", "20261001-ct-a-i18n-v1");
});

test("la caché anterior carga la vista y el rail nuevos sin duplicar el circuito de firma", async () => {
  const [html, portal, coordinador, vista, circuito, interno, produccion] = await Promise.all([
    readFile(new URL("index.html", raiz), "utf8"),
    readFile(new URL("portal.js", raiz), "utf8"),
    readFile(new URL("portal-modulos-coordinador.js", raiz), "utf8"),
    readFile(new URL("modulos/contratacion-temporal/vista-expedientes.js", raiz), "utf8"),
    readFile(new URL("modulos/contratacion-temporal/circuito-firma.js", raiz), "utf8"),
    readFile(new URL("../../interno.manifest", raiz), "utf8"),
    readFile(new URL("../../produccion.manifest", raiz), "utf8"),
  ]);
  const nueva = versionCoordinador;
  const firma = "20261003-ct-firma-v2-v1";
  const versiones = [
    exigirVersiones(html, "/portal-empleado/portal.js", nueva),
    exigirVersiones(html, "/portal-empleado/portal-modulos-coordinador.js", nueva),
    exigirVersiones(portal, "./portal-modulos-coordinador.js", nueva),
  ];
  assert.equal(new Set(versiones).size, 1);
  exigirVersiones(coordinador, "./modulos/contratacion-temporal/vista-expedientes.js", firma);
  // La consulta de circuito RRHH no cambió: conserva su URL anterior.
  exigirVersiones(vista, "./vista-circuito-rrhh.js", "20261002-ct-r5-grafo-v2");
  exigirVersiones(vista, "./circuito-firma.js", firma);
  exigirVersiones(circuito, "./circuito-firma-acciones.js", firma);
  for (const manifiesto of [interno, produccion]) {
    assert.match(manifiesto, /^static\/portal-empleado\/modulos\/contratacion-temporal\/firma-externa-cliente\.js$/mu);
  }
});
