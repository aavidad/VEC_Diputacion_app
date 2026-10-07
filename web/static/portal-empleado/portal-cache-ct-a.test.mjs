import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { exigirVersiones, posterior } from "./versiones-cache.test-helper.mjs";

const versionEntradaAnterior = "20261002-r1-post401-v4";
const versionCoordinador = "20261007-ct-ficha-b7-v1";
const versionCircuito = "20261007-ct-ficha-final-v1";
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
  exigirVersiones(expediente, "./seguimiento-cese.js", versionCircuito);
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
  const versionVista = versionCircuito;
  const firma = versionVista;
  // portal.js puede renovarse después por otros cambios; basta que sea posterior.
  exigirVersiones(html, "/portal-empleado/portal.js", posterior(versionEntradaAnterior));
  const versiones = [
    exigirVersiones(html, "/portal-empleado/portal-modulos-coordinador.js", nueva),
    exigirVersiones(portal, "./portal-modulos-coordinador.js", nueva),
  ];
  assert.equal(new Set(versiones).size, 1);
  exigirVersiones(coordinador, "./modulos/contratacion-temporal/vista-expedientes.js", versionVista);
  // La consulta de circuito RRHH no cambió: conserva su URL anterior.
  exigirVersiones(vista, "./vista-circuito-rrhh.js", "20261002-ct-r5-grafo-v2");
  exigirVersiones(vista, "./circuito-firma.js", firma);
  exigirVersiones(circuito, "./circuito-firma-acciones.js", firma);
  for (const manifiesto of [interno, produccion]) {
    assert.match(manifiesto, /^static\/portal-empleado\/modulos\/contratacion-temporal\/firma-externa-cliente\.js$/mu);
  }
});

test("la ficha CT y Documentos renuevan las dos entradas sin reutilizar hojas anteriores", async () => {
  const rutas = [
    "index.html", "portal-modulos-coordinador.js",
    "modulos/contratacion-temporal/cliente-http-documentacion-formalizacion.js",
    "modulos/contratacion-temporal/circuito-firma.js",
    "categorias-rpt/index.html", "categorias-rpt/arranque.js", "categorias-rpt/cliente.js",
  ];
  const [html, coordinador, formalizacion, firma, categorias, arranque, clienteCategorias] =
    await Promise.all(rutas.map((ruta) => readFile(new URL(ruta, raiz), "utf8")));
  const version = "20261007-ct-ficha-final-v1";
  exigirVersiones(html, "/portal-empleado/modulos/contratacion-temporal/expedientes.css", version);
  for (const hoja of [
    "./modulos/contratacion-temporal/cliente-http.js",
    "./modulos/contratacion-temporal/adaptador-http-expedientes.js",
    "./modulos/contratacion-temporal/vista-expedientes.js",
    "./modulos/documentos/vista.js", "./modulos/documentos/cliente-http.js",
  ]) exigirVersiones(coordinador, hoja, version);
  exigirVersiones(formalizacion, "../documentos/cliente-http.js", version);
  exigirVersiones(firma, "../documentos/cliente-http.js", version);
  exigirVersiones(categorias, "./arranque.js", version);
  exigirVersiones(arranque, "./cliente.js", version);
  exigirVersiones(clienteCategorias, "../modulos/contratacion-temporal/cliente-http.js", version);
});
