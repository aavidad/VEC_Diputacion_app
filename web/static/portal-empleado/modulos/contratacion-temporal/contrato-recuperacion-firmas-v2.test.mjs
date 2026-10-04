import test from "node:test";
import assert from "node:assert/strict";
import { createHash, webcrypto } from "node:crypto";
import {
  validarSolicitudRecuperacionFirmasV2, validarRespuestaRecuperacionFirmasV2,
} from "./contrato-recuperacion-firmas-v2.js";
import {
  crearClienteRecuperacionFirmasV2, RUTA_RECUPERACION_FIRMAS_V2,
} from "./cliente-http-recuperacion-firmas-v2.js";

const h = (x) => x.repeat(64);
const solicitud = Object.freeze({
  expediente_ref: "expediente:prueba", version_expediente: 7,
  documento: "informe_definitivo", paso_orden: 1,
  clave_idempotencia: "clave-prueba-firma-000001", catalogo_huella: h("a"),
  via: "certificado_vec",
});

function respuesta() {
  const canon = JSON.stringify({
    esquema: "vec.competencia-firmante.historica.v1",
    prueba_sintetica: "a".repeat(510),
  });
  const sha = createHash("sha256").update(canon, "utf8").digest("hex");
  const doc = { documento_ref: "original:prueba", version: 1, huella_sha256: h("b") };
  return {
    esquema: "vec.contratacion-temporal.recuperacion-firmas-r5.v2",
    expediente_ref: solicitud.expediente_ref, version_expediente: 7,
    documento: solicitud.documento, historia_revision: 1, historia_sha256: h("c"),
    recuperacion: "recuperada", campos_no_disponibles: [], firma_eficaz: false,
    firmas: [{
      firma_ref: "firma:prueba", recibo_ref: "recibo:prueba",
      registrada_en: "2026-10-03T10:00:00Z", secuencia: 1,
      paso_orden: 1, paso_ref: "paso:primero", version_expediente: 7,
      via: "certificado_vec", resultado: "firmado", catalogo_ref: "catalogo:prueba",
      catalogo_huella: h("a"), original: doc,
      documento_custodiado: {
        expediente_ref: "expediente-documental:prueba", documento_ref: "custodia:prueba",
        version: 1, huella_sha256: h("d"),
      },
      revision_pdf: {
        orden_firma: 1, firma_anterior_ref: "", recibo_anterior_ref: "",
        entrada_documento: doc, entrada_longitud: 100,
        byte_range: [0, 120, 180, 20], revision_sha256: h("d"),
        contenido_firmado_sha256: h("e"), revision_longitud: 200,
        evidencia_firmas_sha256: h("f"),
      },
    }],
    recuperaciones: [{
      firma_ref: "firma:prueba", material_root_sha256: h("1"),
      canon_nominal: canon, canon_nominal_sha256: sha,
      canon_nominal_ref: "evidencia:competencia-firmante-ct:" + h("2"),
    }],
  };
}

test("consulta histórica conserva huellas y descarta el canon bruto", async () => {
  const r = await validarRespuestaRecuperacionFirmasV2(respuesta(), solicitud, webcrypto);
  assert.equal(r.firmas.length, 1);
  assert.equal(r.firmas[0].material_root_sha256, h("1"));
  assert.equal(r.firmas[0].canon_nominal_sha256,
    createHash("sha256").update(respuesta().recuperaciones[0].canon_nominal).digest("hex"));
  assert.equal(JSON.stringify(r).includes("prueba_sintetica"), false);
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
  ];
  for (const alterar of mutaciones) {
    const x = respuesta();
    alterar(x);
    await assert.rejects(validarRespuestaRecuperacionFirmasV2(x, solicitud, webcrypto), TypeError);
  }
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
  assert.equal(r.firmas[0].recibo_ref, "recibo:prueba");

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
