import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { exigirMismaVersion, versionDe } from "./versiones-cache.test-helper.mjs";

const raiz = new URL("./", import.meta.url);
const leer = (ruta) => readFile(new URL(ruta, raiz), "utf8");

test("la entrada CT comparte URLs con HTML y caché pública", async () => {
  const [html, cache, portal, coordinador, categorias, montaje, cliente] = await Promise.all([
    "index.html", "cache-publica-v1.json", "portal.js", "portal-modulos-coordinador.js",
    "categorias-rpt/index.html", "categorias-rpt/montaje.js", "categorias-rpt/cliente.js",
  ].map(leer));
  exigirMismaVersion([html, cache], "/portal-empleado/portal.js");
  exigirMismaVersion([html, portal], "portal-modulos-coordinador.js");
  exigirMismaVersion([coordinador, cliente], "cliente-http.js");
  versionDe(categorias, "./arranque.js");
  versionDe(montaje, "./cliente.js");
});

test("la ficha CT y el circuito de firma usan una URL por módulo compartido", async () => {
  const [coordinador, expediente, tramitacion, llamamiento, circuito, formalizacion] = await Promise.all([
    "portal-modulos-coordinador.js", "modulos/contratacion-temporal/vista-expedientes.js",
    "modulos/contratacion-temporal/vista-expedientes-tramitacion.js",
    "modulos/contratacion-temporal/formulario-llamamiento-pruebas.js",
    "modulos/contratacion-temporal/circuito-firma.js",
    "modulos/contratacion-temporal/cliente-http-documentacion-formalizacion.js",
  ].map(leer));
  exigirMismaVersion([coordinador, llamamiento], "vista-expedientes.js");
  exigirMismaVersion([expediente, tramitacion], "vista-expedientes-render.js");
  exigirMismaVersion([circuito, formalizacion], "../documentos/cliente-http.js");
  versionDe(coordinador, "./modulos/contratacion-temporal/vista-expedientes.js");
  versionDe(expediente, "./circuito-firma.js");
  versionDe(circuito, "./circuito-firma-acciones.js");
});

test("Alta CT conserva módulos y catálogos en ambos manifiestos", async () => {
  const [cliente, alta, transporte, vista, contrato, i18n, interno, produccion] = await Promise.all([
    "modulos/contratacion-temporal/cliente-http.js",
    "modulos/contratacion-temporal/cliente-http-alta.js",
    "modulos/contratacion-temporal/cliente-http-transporte.js",
    "modulos/contratacion-temporal/vista.js",
    "modulos/contratacion-temporal/contrato.js",
    "modulos/contratacion-temporal/i18n.js", "../../interno.manifest", "../../produccion.manifest",
  ].map(leer));
  versionDe(cliente, "./cliente-http-alta.js");
  versionDe(cliente, "./cliente-http-transporte.js");
  exigirMismaVersion([alta, vista], "contrato.js");
  assert.match(alta, /obtenerCatalogosNecesidadesAlta/u);
  assert.match(transporte, /campoAlta/u);
  assert.match(i18n, /export async function cargarMensajesNecesidadesAlta/u);
  assert.match(contrato, /export const ESQUEMA_ALTA_NECESIDAD/u);
  for (const manifiesto of [interno, produccion]) {
    for (const recurso of ["firma-externa-cliente.js", "cliente-http-descarga-firma-v2.js"]) {
      assert.match(manifiesto, new RegExp(`^static/portal-empleado/modulos/contratacion-temporal/${recurso.replaceAll(".", "\\.")}$`, "mu"));
    }
    for (const idioma of ["es", "en"]) {
      const ruta = `static/textos/${idioma}/contratacion-temporal-necesidades-alta.json`;
      assert.equal(manifiesto.split(ruta).length - 1, 1, ruta);
    }
  }
});
