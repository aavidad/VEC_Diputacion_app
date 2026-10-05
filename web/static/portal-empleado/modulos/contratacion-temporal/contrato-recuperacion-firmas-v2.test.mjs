import test from "node:test";
import assert from "node:assert/strict";
import { createHash, webcrypto } from "node:crypto";
import { readFileSync } from "node:fs";
import {
  validarSolicitudRecuperacionFirmasV2, validarRespuestaRecuperacionFirmasV2,
} from "./contrato-recuperacion-firmas-v2.js";
import {
  crearClienteRecuperacionFirmasV2, RUTA_RECUPERACION_FIRMAS_V2,
} from "./cliente-http-recuperacion-firmas-v2.js";
import { crearTraductorRecuperacionFirmasV2 } from "./i18n-recuperacion-firmas-v2.js";

const h = (x) => x.repeat(64);
const solicitud = Object.freeze(JSON.parse(readFileSync(
  new URL("./testdata/recuperacion-firmas-v2-selector-go.json", import.meta.url), "utf8",
)));

function respuesta() {
  return JSON.parse(readFileSync(
    new URL("./testdata/recuperacion-firmas-v2-go.json", import.meta.url), "utf8",
  )).data;
}

function respuestaDosFirmas() {
  return JSON.parse(readFileSync(
    new URL("./testdata/recuperacion-firmas-v2-dos-go.json", import.meta.url), "utf8",
  )).data;
}

test("consulta histórica conserva huellas y descarta el canon bruto", async () => {
  const r = await validarRespuestaRecuperacionFirmasV2(respuesta(), solicitud, webcrypto);
  assert.equal(r.firmas.length, 1);
  assert.equal(r.firmas[0].material_root_sha256, respuesta().recuperaciones[0].material_root_sha256);
  assert.equal(r.firmas[0].canon_nominal_sha256,
    createHash("sha256").update(respuesta().recuperaciones[0].canon_nominal).digest("hex"));
  assert.equal(Object.hasOwn(r.firmas[0], "canon_nominal"), false);
  assert.equal(JSON.stringify(r).includes("certificado_der_sha256"), false);
});

test("acepta la proyección JSON emitida por el handler Go de #590", async () => {
  const contenido = readFileSync(new URL("./testdata/recuperacion-firmas-v2-go.json", import.meta.url), "utf8");
  const envoltorio = JSON.parse(contenido);
  assert.deepEqual(Object.keys(envoltorio), ["data"]);
  const resultado = await validarRespuestaRecuperacionFirmasV2(envoltorio.data, solicitud, webcrypto);
  assert.equal(resultado.firmas[0].recibo_ref, envoltorio.data.firmas[0].recibo_ref);
  assert.equal(Object.hasOwn(resultado.firmas[0], "canon_nominal"), false);
});

test("liga las dos firmas sintéticas con canon generado por el modelo común Go", async () => {
  const r = await validarRespuestaRecuperacionFirmasV2(respuestaDosFirmas(), solicitud, webcrypto);
  assert.equal(r.firmas.length, 2);
  assert.equal(r.firmas[1].recibo_ref, "recibo:segundo");
  assert.equal(Object.hasOwn(r.firmas[1], "canon_nominal"), false);
  for (const alterar of [
    (x) => { x.firmas = x.firmas.slice(1); x.recuperaciones = x.recuperaciones.slice(1); },
    (x) => { x.firmas[1].revision_pdf.firma_anterior_ref = "firma:ajena"; },
    (x) => { x.firmas[1].revision_pdf.recibo_anterior_ref = "recibo:ajeno"; },
    (x) => { x.firmas[1].secuencia = 3; },
    (x) => { x.firmas[1].revision_pdf.entrada_longitud--; },
    (x) => { x.firmas[1].revision_pdf.entrada_documento.huella_sha256 = h("0"); },
  ]) {
    const cruzada = respuestaDosFirmas();
    alterar(cruzada);
    await assert.rejects(validarRespuestaRecuperacionFirmasV2(cruzada, solicitud, webcrypto), TypeError);
  }
});

test("rechaza cruces, JSON inesperado y canon alterado", async () => {
  const mutaciones = [
    (x) => { x.recuperaciones[0].canon_nominal_sha256 = h("0"); },
    (x) => { x.recuperaciones[0].material_root_sha256 = "raiz:inventada"; },
    (x) => { x.recuperaciones[0].firma_ref = "firma:ajena"; },
    (x) => { x.recuperaciones[0].canon_nominal_ref = "evidencia:ajena"; },
    (x) => { x.recuperaciones[0].canon_nominal = null; },
    (x) => { x.ajeno = true; },
    (x) => { x.firmas[0].revision_pdf = null; },
    (x) => { x.campos_no_disponibles = ["canon_nominal"]; },
    (x) => { x.firmas[0].documento_custodiado.expediente_ref = "ref:" + h("0"); },
    (x) => { x.firmas[0].revision_pdf.byte_range = [0, 0, 0, 0]; },
    (x) => { x.firmas[0].revision_pdf.revision_sha256 = h("0"); },
    (x) => { x.firmas[0].revision_pdf.entrada_documento.huella_sha256 = h("0"); },
  ];
  for (const alterar of mutaciones) {
    const x = respuesta();
    alterar(x);
    await assert.rejects(validarRespuestaRecuperacionFirmasV2(x, solicitud, webcrypto), TypeError);
  }
  const ajeno = JSON.parse(readFileSync(new URL(
    "./testdata/recuperacion-firmas-v2-canon-ajeno-go.json", import.meta.url), "utf8"));
  const cruzado = respuesta();
  cruzado.recuperaciones[0].canon_nominal = ajeno.canon_nominal;
  cruzado.recuperaciones[0].canon_nominal_sha256 = ajeno.canon_nominal_sha256;
  await assert.rejects(validarRespuestaRecuperacionFirmasV2(cruzado, solicitud, webcrypto), TypeError);
  assert.throws(() => validarSolicitudRecuperacionFirmasV2({ ...solicitud, actor_ref: "per_cliente" }), TypeError);
});

test("cliente usa ruta real, una petición y descarta respuesta tardía cancelada", async () => {
  let llamada;
  const cliente = crearClienteRecuperacionFirmasV2({
    ejecutar: async (opciones) => { llamada = opciones; return respuesta(); },
    validarOpciones: (opciones) => opciones ?? {},
    cryptoImpl: webcrypto,
  });
  const r = await cliente.recuperar(solicitud);
  assert.equal(llamada.ruta, RUTA_RECUPERACION_FIRMAS_V2);
  assert.equal(llamada.metodo, "POST");
  assert.equal(llamada.efecto, false);
  assert.equal(Object.keys(llamada.entrada).length, 7);
  assert.equal(r.firmas[0].recibo_ref, respuesta().firmas[0].recibo_ref);

  const controlador = new AbortController();
  const tardio = crearClienteRecuperacionFirmasV2({
    ejecutar: async () => {
      controlador.abort();
      return respuesta();
    },
    validarOpciones: (opciones) => opciones ?? {},
    cryptoImpl: webcrypto,
  });
  await assert.rejects(tardio.recuperar(solicitud, { signal: controlador.signal }),
    (error) => error.name === "AbortError");
});

test("los dos catálogos JSON tienen el mismo contrato y no hay idioma en el traductor", () => {
  const catalogos = ["es", "en"].map((idioma) => JSON.parse(readFileSync(
    new URL("../../../textos/" + idioma + "/contratacion-temporal-recuperacion-firmas-v2.json",
      import.meta.url), "utf8",
  )).general);
  assert.deepEqual(Object.keys(catalogos[0]).sort(), Object.keys(catalogos[1]).sort());
  for (const catalogo of catalogos) {
    const t = crearTraductorRecuperacionFirmasV2(catalogo);
    assert.ok(t("titulo"));
    assert.ok(t("firma", { numero: 2 }).includes("2"));
  }
  assert.throws(() => crearTraductorRecuperacionFirmasV2({}), TypeError);
});
