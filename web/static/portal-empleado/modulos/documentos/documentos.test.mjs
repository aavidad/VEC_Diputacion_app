import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { MENSAJES_DOCUMENTOS_ES, crearTraductorDocumentos } from "./i18n.js";
import { obtenerArchivoDescargaAutorizada, validarArchivoDescarga, validarRespuestaDocumentos } from "./vista.js";

const directorio = new URL("./", import.meta.url);
const [vista, css] = await Promise.all([
  readFile(new URL("vista.js", directorio), "utf8"),
  readFile(new URL("documentos.css", directorio), "utf8"),
]);

test("la vista carga el catálogo Documentos con la URL F2 vigente", async () => {
  const ruta = new URL("./i18n.js?v=20260924-f2-web2", directorio);
  assert.ok(vista.includes(`from "./i18n.js${ruta.search}"`));
  assert.ok(!vista.includes('from "./i18n.js"'));
  const catalogo = await import(ruta.href);
  assert.equal(catalogo.crearTraductorDocumentos()("titulo"), "Repositorio documental");
});

function entrada(cambios = {}) {
  return {
    ref: "doc-1", titulo: "Documento de prueba", tipo: "Informe", version: 2,
    estado_firma: "firmado", firma: { validada: true, referencia: "firma-1" },
    custodia: { confirmada: true, recibo_ref: "custodia-1" },
    antivirus: "limpio", descargable: true,
    permiso_descarga: { accion: "documentos.descargar_original", recurso_ref: "doc-1", version: 2, concedido: true },
    ...cambios,
  };
}

function respuesta(documentos) {
  return { estado: "disponible", origen: "Fuente autorizada", documentos };
}

test("normaliza firma, custodia y descarga solo con sus evidencias independientes", () => {
  const [item] = validarRespuestaDocumentos(respuesta([entrada()])).documentos;
  assert.equal(item.firma, "firmado");
  assert.equal(item.firmaRef, "firma-1");
  assert.equal(item.custodia, "custodia-1");
  assert.equal(item.descargable, true);

  const [sinEvidencia] = validarRespuestaDocumentos(respuesta([entrada({
    firma: { validada: false, referencia: "firma-1" },
    custodia: { confirmada: true }, antivirus: "pendiente",
  })])).documentos;
  assert.equal(sinEvidencia.firma, "sin_acreditar");
  assert.equal(sinEvidencia.firmaRef, null);
  assert.equal(sinEvidencia.custodia, null);
  assert.equal(sinEvidencia.descargable, false);
  assert.equal(validarRespuestaDocumentos(respuesta([entrada({ estado_firma: "borrador" })])).documentos[0].firma, "borrador");
});

test("deniega descarga cuando el DTO no aporta permiso positivo para esa referencia", () => {
  const [sinPermiso] = validarRespuestaDocumentos(respuesta([entrada({ permiso_descarga: undefined })])).documentos;
  assert.equal(sinPermiso.descargable, false);
  for (const permiso of [
    { accion: "documentos.descargar_original", recurso_ref: "otro-doc", version: 2, concedido: true },
    { accion: "documentos.descargar_original", recurso_ref: "doc-1", version: 3, concedido: true },
    { accion: "documentos.leer", recurso_ref: "doc-1", version: 2, concedido: true },
    { accion: "documentos.descargar_original", recurso_ref: "doc-1", version: 2, concedido: false },
  ]) {
    assert.equal(validarRespuestaDocumentos(respuesta([entrada({ permiso_descarga: permiso })])).documentos[0].descargable, false);
  }
});

test("la revalidación denegada o cruzada impide pedir los bytes", async () => {
  const [item] = validarRespuestaDocumentos(respuesta([entrada()])).documentos;
  let descargas = 0;
  const fuente = {
    async confirmarPermisoDescarga() { return { accion: "documentos.descargar_original", recurso_ref: "otro-doc", version: 2, concedido: true }; },
    async descargar() { descargas++; return { contenido: new Uint8Array([37, 80, 68, 70]), nombre: "informe.pdf", tipo: "application/pdf" }; },
  };
  await assert.rejects(obtenerArchivoDescargaAutorizada(fuente, item), /permiso de descarga/);
  assert.equal(descargas, 0);
  fuente.confirmarPermisoDescarga = async () => ({ accion: "documentos.descargar_original", recurso_ref: "doc-1", version: 2, concedido: true });
  await assert.rejects(obtenerArchivoDescargaAutorizada(fuente, item, { vigente: () => false }), /permiso de descarga/);
  assert.equal(descargas, 0);
  assert.equal((await obtenerArchivoDescargaAutorizada(fuente, item)).nombre, "informe.pdf");
  assert.equal(descargas, 1);
});

test("denegación, vacío y datos inválidos no producen filas documentales", () => {
  assert.deepEqual(validarRespuestaDocumentos({ estado: "denegado" }).documentos, []);
  assert.equal(validarRespuestaDocumentos({ estado: "vacio", origen: "Fuente autorizada", documentos: [] }).estado, "vacio");
  assert.throws(() => validarRespuestaDocumentos({ estado: "vacio", origen: "Fuente autorizada", documentos: [entrada()] }));
  assert.throws(() => validarRespuestaDocumentos(respuesta([entrada(), entrada()])));
  assert.throws(() => validarRespuestaDocumentos(respuesta([entrada({ version: 0 })])));
  assert.throws(() => validarRespuestaDocumentos(respuesta([entrada({ estado_firma: "publicado" })])));
});

test("la descarga exige bytes del original, nombre seguro y formato admitido", () => {
  const archivo = { contenido: new Uint8Array([37, 80, 68, 70]), nombre: "informe.pdf", tipo: "application/pdf" };
  assert.equal(validarArchivoDescarga(archivo).nombre, "informe.pdf");
  assert.throws(() => validarArchivoDescarga({ ...archivo, contenido: new Uint8Array() }));
  assert.throws(() => validarArchivoDescarga({ ...archivo, nombre: "../informe.pdf" }));
  assert.throws(() => validarArchivoDescarga({ ...archivo, nombre: "informe.png" }));
  assert.throws(() => validarArchivoDescarga({ ...archivo, nombre: "informe\u202epdf.exe" }));
  assert.throws(() => validarArchivoDescarga({ ...archivo, tipo: "text/html" }));
  assert.throws(() => validarArchivoDescarga({ ...archivo, contenido: "https://example.invalid/original" }));
});

test("el montaje admite fuente inyectada y conserva no_configurado sin ella", () => {
  assert.match(vista, /export function montarVistaDocumentos/u);
  assert.match(vista, /fuente\.listar\(\{ signal:/u);
  assert.ok(vista.includes("fuente.confirmarPermisoDescarga(item.ref, { version: item.version, signal })"));
  assert.ok(vista.includes("fuente.descargar(item.ref, { version: item.version, signal })"));
  assert.match(vista, /obtenerArchivoDescargaAutorizada\(fuente, item, \{ signal, vigente \}\)/u);
  assert.match(vista, /URL\.createObjectURL/u);
  assert.match(vista, /let estado = "no_configurado"/u);
  assert.match(vista, /registrarDesmontar\?\.\(desmontar\)/u);
  assert.match(vista, /controlador\?\.abort\(\)/u);
  assert.doesNotMatch(vista, /datos-presentacion|datos-sinteticos|fetch\(|XMLHttpRequest|document\.cookie|localStorage|sessionStorage|indexedDB/iu);
});

test("filtro, ficha, foco y ayuda son navegables y la descarga tiene guarda", () => {
  assert.match(vista, /"search"/u);
  assert.match(vista, /fichaTitulo\.focus\(\)/u);
  assert.match(vista, /"details"/u);
  assert.match(vista, /"summary"/u);
  assert.match(vista, /descargar\.disabled = !seleccionado\?\.descargable/u);
  assert.match(vista, /descargando = true/u);
  assert.match(vista, /"aria-live", "polite"/u);
  assert.match(css, /\.documentos-tabla\s*\{[^}]*overflow:\s*auto/u);
  assert.match(css, /documentos-tabla-ayuda/u);
  assert.match(css, /@media \(max-width: 1150px\)/u);
  assert.match(css, /@media \(max-width: 850px\)/u);
  assert.match(css, /@media \(max-width: 480px\)/u);
  assert.doesNotMatch(css, /#[0-9a-f]{3,8}\b/iu);
});

test("los textos visibles y los seis estados proceden del catálogo", () => {
  const claves = [...vista.matchAll(/t\("([a-z0-9_]+)"/gu)].map((coincidencia) => coincidencia[1]);
  for (const clave of claves) assert.equal(typeof MENSAJES_DOCUMENTOS_ES[clave], "string", `falta ${clave}`);
  for (const estado of ["no_configurado", "cargando", "disponible", "vacio", "denegado", "error"]) {
    assert.equal(typeof MENSAJES_DOCUMENTOS_ES[`estado_${estado}`], "string");
    assert.equal(typeof MENSAJES_DOCUMENTOS_ES[`explicacion_${estado}`], "string");
  }
  for (const firma of ["borrador", "pendiente_firma", "firmado", "sin_acreditar"]) assert.equal(typeof MENSAJES_DOCUMENTOS_ES[`firma_${firma}`], "string");
  assert.match(crearTraductorDocumentos()("aclaracion_firma"), /Autenticarse con certificado/u);
  assert.match(MENSAJES_DOCUMENTOS_ES.tipos_previstos, /sin documentos asociados/u);
});
