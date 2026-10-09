import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { exigirVersiones, posterior } from "./versiones-cache.test-helper.mjs";

const versionEntradaAnterior = "20261002-r1-post401-v4";
const versionCoordinador = "20261009-centro-campos-cohorte-v3";
const versionCircuito = "20261009-centro-campos-cohorte-v3";
const versionVista = "20261009-centro-campos-cohorte-v3";
const versionVistaPortal = "20261009-centro-campos-cohorte-v3";
const versionAdaptador = "20261009-centro-campos-cohorte-v3";
const versionRender = "20261008-r-fichas-idioma-nav-v1";
const versionContratacion = "20261007-pantallas-textos-final-v1";

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
  exigirVersiones(expediente, "./seguimiento-cese.js", "20261007-pantallas-textos-final-v1");
});

test("Alta CT alcanza render y tramitación con una sola versión del formulario", async () => {
  const [vista, tramitacion, llamamiento] = await Promise.all([
    "modulos/contratacion-temporal/vista-expedientes.js",
    "modulos/contratacion-temporal/vista-expedientes-tramitacion.js",
    "modulos/contratacion-temporal/formulario-llamamiento-pruebas.js",
  ].map((ruta) => readFile(new URL(ruta, raiz), "utf8")));
  exigirVersiones(vista, "./vista-expedientes-render.js", versionRender, 2);
  exigirVersiones(vista, "./vista-expedientes-tramitacion.js", versionVista);
  exigirVersiones(tramitacion, "./vista-expedientes-render.js", versionRender);
  exigirVersiones(llamamiento, "./vista-expedientes.js", versionVistaPortal);
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
  const firma = versionCircuito;

  // portal.js puede renovarse después por otros cambios; basta que sea posterior.
  exigirVersiones(html, "/portal-empleado/portal.js", posterior(versionEntradaAnterior));
  const versiones = [
    exigirVersiones(html, "/portal-empleado/portal-modulos-coordinador.js", nueva),
    exigirVersiones(portal, "./portal-modulos-coordinador.js", nueva),
  ];
  assert.equal(new Set(versiones).size, 1);
  exigirVersiones(coordinador, "./modulos/contratacion-temporal/vista-expedientes.js", versionVistaPortal);
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
    "categorias-rpt/index.html", "categorias-rpt/montaje.js", "categorias-rpt/cliente.js",
  ];
  const [html, coordinador, formalizacion, firma, categorias, montajeCategorias, clienteCategorias] =
    await Promise.all(rutas.map((ruta) => readFile(new URL(ruta, raiz), "utf8")));
  const version = "20261007-pantallas-textos-final-v1";
  const cohorteCT = "20261009-centro-campos-cohorte-v3";
  exigirVersiones(html, "/portal-empleado/modulos/contratacion-temporal/expedientes.css", version);
  exigirVersiones(coordinador, "./modulos/contratacion-temporal/cliente-http.js", cohorteCT);
  exigirVersiones(coordinador, "./modulos/contratacion-temporal/adaptador-http-expedientes.js", versionAdaptador);
  for (const hoja of ["./modulos/documentos/vista.js", "./modulos/documentos/cliente-http.js"])
    exigirVersiones(coordinador, hoja, version);
  exigirVersiones(coordinador, "./modulos/contratacion-temporal/vista-expedientes.js", versionVistaPortal);
  exigirVersiones(formalizacion, "../documentos/cliente-http.js", version);
  exigirVersiones(firma, "../documentos/cliente-http.js", version);
  exigirVersiones(categorias, "./arranque.js", "20261008-hz8-idioma-v1");
  exigirVersiones(montajeCategorias, "./cliente.js", cohorteCT);
  exigirVersiones(clienteCategorias, "../modulos/contratacion-temporal/cliente-http.js", cohorteCT);
});

test("Alta por circular renueva su cadena y no reutiliza módulos sin las exportaciones nuevas", async () => {
  const nombres = ["index.html", "portal.js", "portal-modulos-coordinador.js",
    "modulos/contratacion-temporal/cliente-http.js",
    "modulos/contratacion-temporal/cliente-http-alta.js",
    "modulos/contratacion-temporal/cliente-http-transporte.js",
    "modulos/contratacion-temporal/vista-expedientes.js",
    "modulos/contratacion-temporal/vista-expedientes-tramitacion.js",
    "modulos/contratacion-temporal/vista.js",
    "modulos/contratacion-temporal/presentador.js",
    "modulos/contratacion-temporal/alta-renderer-puro.js",
    "modulos/contratacion-temporal/i18n.js",
    "modulos/contratacion-temporal/contrato.js",
    "../../interno.manifest", "../../produccion.manifest", "cache-publica-v1.json"];
  const [html, portal, coordinador, cliente, altaHTTP, transporte, expedientes, tramitacion,
    vista, presentadorAlta, rendererAlta, i18n, contrato, interno, produccion, cache] = await Promise.all(
    nombres.map((nombre) => readFile(new URL(nombre, raiz), "utf8")));
  const cohorte = "20261009-centro-campos-cohorte-v3";
  const cohorteNavegacion = "20261009-centro-campos-cohorte-v3";
  const cohorteClienteHTTP = cohorte;
  const cohorteRPT = "20261008-alta-rpt-circular-v6";
  const cohorteCapacidad = cohorte;
  const cohorteIdioma = "20261009-centro-campos-cohorte-v3";
  const cohorteEntrada = "20261009-centro-campos-cohorte-v3";
  const cohorteFicha = cohorteNavegacion;
  const antiguas = new Map([
    ["/portal-empleado/portal.js", "20261008-bolsa-inicio-v2"],
    ["/portal-empleado/portal-modulos-coordinador.js", "20261008-ct-inicio-v1"],
    ["./modulos/contratacion-temporal/cliente-http.js", "20261007-pantallas-textos-final-v1"],
    ["./cliente-http-alta.js", "20261008-ct-necesidades-v2"],
    ["./i18n.js", "20261007-pantallas-textos-final-v1"],
  ]);
  const aristas = [
    [html, "/portal-empleado/portal.js", cohorteEntrada],
    [html, "/portal-empleado/portal-modulos-coordinador.js", cohorteIdioma],
    [portal, "./portal-modulos-coordinador.js", cohorteIdioma],
    [coordinador, "./modulos/contratacion-temporal/cliente-http.js", cohorteClienteHTTP],
    [cliente, "./cliente-http-alta.js", cohorte],
    [cliente, "./cliente-http-transporte.js", "20261008-alta-corte-v1"],
    [coordinador, "./modulos/contratacion-temporal/vista-expedientes.js", versionVistaPortal],
    [expedientes, "./vista-expedientes-tramitacion.js", cohorteFicha],
    [tramitacion, "./vista.js", cohorteCapacidad],
    [tramitacion, "./presentador.js", cohorte],
    [vista, "./i18n.js", cohorteRPT],
    [vista, "./contrato.js", cohorte],
    [vista, "./alta-renderer-puro.js", cohorte],
    [presentadorAlta, "./contrato.js", cohorte],
    [rendererAlta, "./contrato.js", cohorte, 2],
  ];
  for (const [fuente, ruta, version, cantidad = 1] of aristas) {
    exigirVersiones(fuente, ruta, version, cantidad);
    if (antiguas.has(ruta)) assert.ok(!fuente.includes(`${ruta}?v=${antiguas.get(ruta)}`), ruta);
  }
  exigirVersiones(cache, "/portal-empleado/portal.js", cohorteEntrada);
  assert.match(altaHTTP, /obtenerCatalogosNecesidadesAlta/u);
  assert.match(transporte, /campoAlta/u);
  assert.match(i18n, /export async function cargarMensajesNecesidadesAlta/u);
  assert.match(contrato, /export const ESQUEMA_ALTA_NECESIDAD/u);
  for (const contenido of [interno, produccion]) {
    for (const idioma of ["es", "en"]) {
      const ruta = `static/textos/${idioma}/contratacion-temporal-necesidades-alta.json`;
      assert.equal(contenido.split(ruta).length - 1, 1, ruta);
    }
  }
});

test("la ficha compartible y su aviso usan una única cohorte empaquetada", async () => {
  const [html, portal, coordinador, interno, produccion, cache] = await Promise.all([
    "index.html", "portal.js", "portal-modulos-coordinador.js",
    "../../interno.manifest", "../../produccion.manifest", "cache-publica-v1.json",
  ].map((ruta) => readFile(new URL(ruta, raiz), "utf8")));
  const cohorte = "20261008-r-fichas-idioma-nav-v1";
  const entrada = "20261009-centro-campos-cohorte-v3";
  exigirVersiones(html, "/portal-empleado/portal-ct-ruta-ficha.js", cohorte);
  exigirVersiones(portal, "./portal-ct-ruta-ficha.js", cohorte);
  exigirVersiones(coordinador, "./portal-ct-ruta-ficha.js", cohorte);
  exigirVersiones(html, "/portal-empleado/portal.js", entrada);
  exigirVersiones(cache, "/portal-empleado/portal.js", entrada);
  for (const manifiesto of [interno, produccion]) {
    assert.equal(manifiesto.split("static/portal-empleado/portal-ct-ruta-ficha.js").length - 1, 1);
  }
});
