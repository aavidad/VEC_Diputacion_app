import test from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { crearClienteDocumentosBolsa, validarCorrespondenciaReciboDocumento } from "./documentos-api.js";
import { cargarTextos } from "../../../comun/textos.js";

const bolsaRef = "bolsa:categoria:administrativo";
const documentoRef = "acta:importacion-convoca:abcdef012345";
const listadoRef = "listado:definitivo:abcdef012345";
const huellaListado = "a".repeat(64);
const instante = "2026-10-13T08:30:00Z";
const reciboRuta = `/api/vec/bolsa/bolsas/${bolsaRef}/documentos/${documentoRef}/recibo`;
const documento = { documento_ref: documentoRef, tipo: "constitucion", titulo_clave: "bolsa_documento_constitucion",
  fecha: instante, acta_ref: documentoRef, version_bolsa: 4, instantanea_ref: "instantanea:abc",
  version_instantanea: 2, listado_definitivo_ref: listadoRef, version_listado: 3,
  huella_listado_sha256: huellaListado, total_participaciones: 17, estado_firma: "sin_firma_acreditada",
  formato: "application/json", accion: { tipo: "descargar_recibo", href: reciboRuta },
  ficha_candidatos_href: `/api/vec/bolsa/bolsas/${bolsaRef}/candidatos` };
const lista = { data: { esquema: "vec.bolsa.rrhh.documentos.v1", bolsa_ref: bolsaRef,
  documentos: [documento], paginacion: { limite: 50, total: 1, cursor_siguiente: null } } };
const recibo = { data: { esquema: "vec.bolsa.rrhh.recibo_constitucion.v1", documento_ref: documentoRef,
  bolsa_ref: bolsaRef, acta_ref: documentoRef, version_bolsa: 4, instantanea_ref: "instantanea:abc",
  version_instantanea: 2, listado_definitivo_ref: listadoRef, version_listado: 3,
  huella_listado_sha256: huellaListado, confirmada_en: instante, total_participaciones: 17,
  estado_firma: "sin_firma_acreditada" } };
const responder = (datos) => new Response(JSON.stringify(datos), { status: 200, headers: { "Content-Type": "application/json; charset=utf-8" } });

test("consulta la bolsa concreta y conserva los bytes exactos del recibo", async () => {
  const llamadas = [];
  const cuerpoExacto = ` {\n  "data": ${JSON.stringify(recibo.data)}\n}\n`;
  const cliente = crearClienteDocumentosBolsa({ fetchImpl: async (url, opciones) => {
    llamadas.push({ url, opciones });
    return url === reciboRuta ? new Response(cuerpoExacto, { headers: { "Content-Type": "application/json; charset=utf-8" } }) : responder(lista);
  } });
  const pagina = await cliente.listar({ bolsaRef, tipo: "constitucion", texto: "acta" });
  assert.equal(pagina.documentos[0].documento_ref, documentoRef);
  assert.match(llamadas[0].url, /tipo=constitucion&texto=acta/u);
  assert.equal(llamadas[0].opciones.credentials, "same-origin");
  assert.equal(llamadas[0].opciones.redirect, "error");
  const archivo = await cliente.recibir({ bolsaRef, documentoRef });
  assert.equal(new TextDecoder().decode(archivo.bytes), cuerpoExacto);
  assert.equal(validarCorrespondenciaReciboDocumento(archivo.datos, pagina.documentos[0]), archivo.datos);
  assert.equal(llamadas[1].url, reciboRuta);
});

test("rechaza metadatos del listado ausentes o recibo de otra versión", async () => {
  const sinHuella = crearClienteDocumentosBolsa({ fetchImpl: async () => responder({ data: {
    ...lista.data, documentos: [{ ...documento, huella_listado_sha256: undefined }],
  } }) });
  await assert.rejects(sinHuella.listar({ bolsaRef }), TypeError);
  const sinVersion = crearClienteDocumentosBolsa({ fetchImpl: async () => responder({ data: {
    ...lista.data, documentos: [{ ...documento, version_listado: undefined }],
  } }) });
  await assert.rejects(sinVersion.listar({ bolsaRef }), TypeError);
  const sinReferencia = crearClienteDocumentosBolsa({ fetchImpl: async () => responder({ data: {
    ...lista.data, documentos: [{ ...documento, listado_definitivo_ref: undefined }],
  } }) });
  await assert.rejects(sinReferencia.listar({ bolsaRef }), TypeError);
  const reciboSinHuella = crearClienteDocumentosBolsa({ fetchImpl: async () => responder({ data: {
    ...recibo.data, huella_listado_sha256: undefined,
  } }) });
  await assert.rejects(reciboSinHuella.recibir({ bolsaRef, documentoRef }), TypeError);
  assert.throws(() => validarCorrespondenciaReciboDocumento({ ...recibo.data, version_listado: 4 }, documento), TypeError);
  assert.throws(() => validarCorrespondenciaReciboDocumento({ ...recibo.data, huella_listado_sha256: "b".repeat(64) }, documento), TypeError);
});

test("falla cerrado cuando el transporte no ofrece lectura por streaming", async () => {
  let arrayBufferLlamado = false;
  const cliente = crearClienteDocumentosBolsa({ fetchImpl: async () => ({
    ok: true, headers: new Headers({ "Content-Type": "application/json" }),
    arrayBuffer: () => { arrayBufferLlamado = true; throw new Error("lectura sin límite"); },
  }) });
  await assert.rejects(cliente.listar({ bolsaRef }), TypeError);
  assert.equal(arrayBufferLlamado, false);
});

test("rechaza un enlace de descarga que no pertenece al documento solicitado", async () => {
  const cliente = crearClienteDocumentosBolsa({ fetchImpl: async () => responder({ data: {
    ...lista.data, documentos: [{ ...documento, accion: { tipo: "descargar_recibo", href: "https://otro.invalid/archivo" } }],
  } }) });
  await assert.rejects(cliente.listar({ bolsaRef }), TypeError);
});

test("catálogos de la vista completos en castellano e inglés", async () => {
  const es = JSON.parse(await readFile(new URL("../../../textos/es/bolsa-documentos.json", import.meta.url), "utf8"));
  const en = JSON.parse(await readFile(new URL("../../../textos/en/bolsa-documentos.json", import.meta.url), "utf8"));
  assert.deepEqual(Object.keys(es.general).sort(), Object.keys(en.general).sort());
  for (const idioma of ["es", "en"]) {
    const textos = await cargarTextos("bolsa-documentos", { idioma });
    assert.deepEqual(textos.faltantes, []);
  }
});
