import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { MENSAJES_DOCUMENTOS_ES, crearTraductorDocumentos } from "./i18n.js";
import { validarArchivoDescarga, validarRespuestaDocumentos } from "./vista.js";

const directorio = new URL("./", import.meta.url);
const [vista, css] = await Promise.all([
  readFile(new URL("vista.js", directorio), "utf8"),
  readFile(new URL("documentos.css", directorio), "utf8"),
]);

function entrada(cambios = {}) {
  return {
    ref: "doc-1", titulo: "Documento de prueba", tipo: "Informe", version: 2,
    estado_firma: "firmado", firma: { validada: true, referencia: "firma-1" },
    custodia: { confirmada: true, recibo_ref: "custodia-1" },
    antivirus: "limpio", descargable: true,
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
  assert.ok(vista.includes("fuente.descargar(ref, { signal })"));
  assert.match(vista, /validarArchivoDescarga\(await fuente\.descargar/u);
  assert.match(vista, /actual !== secuencia \|\| signal\?\.aborted \|\| seleccionado\?\.ref !== ref/u);
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
