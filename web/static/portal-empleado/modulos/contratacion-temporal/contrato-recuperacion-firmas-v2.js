export const ESQUEMA_RECUPERACION_FIRMAS_V2 = "vec.contratacion-temporal.recuperacion-firmas-r5.v2";

const REF = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const CLAVE_DOCUMENTO = /^[a-z][a-z0-9_]{1,63}$/u;
const CLAVE_IDEMPOTENCIA = /^[A-Za-z0-9][A-Za-z0-9._-]{15,63}$/u;
const SHA = /^[0-9a-f]{64}$/u;
const REF_CANON = /^evidencia:competencia-firmante-ct:[0-9a-f]{64}$/u;
const FECHA = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$/u;
const CAMPOS_RESPUESTA = ["esquema", "expediente_ref", "version_expediente", "documento",
  "historia_revision", "historia_sha256", "firmas", "recuperacion",
  "campos_no_disponibles", "firma_eficaz", "recuperaciones"];
const CAMPOS_FIRMA = ["firma_ref", "recibo_ref", "registrada_en", "secuencia", "paso_orden",
  "paso_ref", "version_expediente", "via", "resultado", "catalogo_ref", "catalogo_huella",
  "original", "documento_custodiado", "revision_pdf"];
const CAMPOS_REVISION = ["orden_firma", "firma_anterior_ref", "recibo_anterior_ref",
  "entrada_documento", "entrada_longitud", "byte_range", "revision_sha256",
  "contenido_firmado_sha256", "revision_longitud", "evidencia_firmas_sha256"];
const CAMPOS_RECUPERACION = ["firma_ref", "material_root_sha256", "canon_nominal",
  "canon_nominal_sha256", "canon_nominal_ref"];

function registro(x) {
  return x !== null && typeof x === "object" && !Array.isArray(x)
    && Object.getPrototypeOf(x) === Object.prototype;
}

function exacto(x, claves) {
  return registro(x) && Object.keys(x).length === claves.length
    && claves.every((clave) => Object.hasOwn(x, clave));
}

function entero(x, minimo = 0) {
  return Number.isSafeInteger(x) && x >= minimo && x <= 9007199254740991;
}

function coincide(x, patron) {
  return typeof x === "string" && patron.test(x);
}

function fecha(x) {
  return coincide(x, FECHA) && Number.isFinite(Date.parse(x));
}

async function sha256Hex(bytes, cryptoImpl) {
  const suma = await cryptoImpl.subtle.digest("SHA-256", bytes);
  return [...new Uint8Array(suma)].map((n) => n.toString(16).padStart(2, "0")).join("");
}

function documento(x) {
  return exacto(x, ["documento_ref", "version", "huella_sha256"])
    && coincide(x.documento_ref, REF) && entero(x.version, 1) && coincide(x.huella_sha256, SHA);
}

export function validarSolicitudRecuperacionFirmasV2(x) {
  if (!exacto(x, ["expediente_ref", "version_expediente", "documento", "paso_orden",
    "clave_idempotencia", "catalogo_huella", "via"])
    || !coincide(x.expediente_ref, REF) || !entero(x.version_expediente, 1)
    || !coincide(x.documento, CLAVE_DOCUMENTO) || ![1, 2].includes(x.paso_orden)
    || !coincide(x.clave_idempotencia, CLAVE_IDEMPOTENCIA) || !coincide(x.catalogo_huella, SHA)
    || !["certificado_vec", "portafirmas_registro_rrhh"].includes(x.via)) {
    throw new TypeError("solicitud_recuperacion_firmas_invalida");
  }
  return Object.freeze({ ...x });
}

function validarFirma(x, solicitud) {
  if (!exacto(x, CAMPOS_FIRMA) || !coincide(x.firma_ref, REF) || !coincide(x.recibo_ref, REF)
    || !fecha(x.registrada_en) || !entero(x.secuencia, 1) || ![1, 2].includes(x.paso_orden)
    || !coincide(x.paso_ref, REF) || !entero(x.version_expediente, 1)
    || x.version_expediente > solicitud.version_expediente
    || !["", "certificado_vec", "portafirmas_registro_rrhh"].includes(x.via)
    || !["firmado", "devuelto"].includes(x.resultado)
    || !coincide(x.catalogo_ref, REF) || !coincide(x.catalogo_huella, SHA)
    || !(documento(x.original) || x.via === "" && exacto(x.original,
      ["documento_ref", "version", "huella_sha256"])
      && x.original.documento_ref === "" && x.original.version === 0
      && x.original.huella_sha256 === "")) throw new TypeError("firma_historica_invalida");
  if (x.documento_custodiado !== null
    && (!exacto(x.documento_custodiado,
      ["expediente_ref", "documento_ref", "version", "huella_sha256"])
      || !coincide(x.documento_custodiado.expediente_ref, REF)
      || !coincide(x.documento_custodiado.documento_ref, REF)
      || !entero(x.documento_custodiado.version, 1)
      || !coincide(x.documento_custodiado.huella_sha256, SHA))) {
    throw new TypeError("custodia_historica_invalida");
  }
  if (x.revision_pdf !== null) {
    const r = x.revision_pdf;
    if (!exacto(r, CAMPOS_REVISION) || ![1, 2].includes(r.orden_firma)
      || r.orden_firma !== x.paso_orden
      || typeof r.firma_anterior_ref !== "string" || typeof r.recibo_anterior_ref !== "string"
      || r.orden_firma === 1 && (r.firma_anterior_ref !== "" || r.recibo_anterior_ref !== "")
      || r.orden_firma === 2 && (!coincide(r.firma_anterior_ref, REF) || !coincide(r.recibo_anterior_ref, REF))
      || !documento(r.entrada_documento) || !entero(r.entrada_longitud, 1)
      || !Array.isArray(r.byte_range) || r.byte_range.length !== 4
      || !r.byte_range.every((v) => entero(v))
      || !coincide(r.revision_sha256, SHA) || !coincide(r.contenido_firmado_sha256, SHA)
      || !entero(r.revision_longitud, 1) || !coincide(r.evidencia_firmas_sha256, SHA)
      || x.documento_custodiado === null
      || r.entrada_longitud >= r.revision_longitud || r.byte_range[0] !== 0
      || r.byte_range[1] < r.entrada_longitud || r.byte_range[2] <= r.byte_range[1]
      || r.byte_range[2] >= r.revision_longitud
      || r.byte_range[3] !== r.revision_longitud - r.byte_range[2]
      || r.revision_sha256 !== x.documento_custodiado.huella_sha256
      || r.orden_firma === 1 && (
        r.entrada_documento.documento_ref !== x.original.documento_ref
        || r.entrada_documento.version !== x.original.version
        || r.entrada_documento.huella_sha256 !== x.original.huella_sha256
      )) throw new TypeError("revision_historica_invalida");
  }
}

async function validarCanon(rec, firma, solicitud, cryptoImpl) {
  if (!exacto(rec, CAMPOS_RECUPERACION) || !coincide(rec.firma_ref, REF)
    || !coincide(rec.material_root_sha256, SHA) || !coincide(rec.canon_nominal_sha256, SHA)
    || !coincide(rec.canon_nominal_ref, REF_CANON) || typeof rec.canon_nominal !== "string"
    || typeof TextEncoder !== "function" || typeof TextDecoder !== "function"
    || typeof cryptoImpl?.subtle?.digest !== "function") {
    throw new TypeError("canon_historico_invalido");
  }
  const bytes = new TextEncoder().encode(rec.canon_nominal);
  if (bytes.length < 512 || bytes.length > 32768
    || new TextDecoder("utf-8", { fatal: true }).decode(bytes) !== rec.canon_nominal) {
    throw new TypeError("canon_historico_invalido");
  }
  let canon;
  try { canon = JSON.parse(rec.canon_nominal); } catch { throw new TypeError("canon_historico_invalido"); }
  if (!registro(canon) || canon.esquema !== "vec.competencia-firmante.historica.v1") {
    throw new TypeError("canon_historico_invalido");
  }
  if (await sha256Hex(bytes, cryptoImpl) !== rec.canon_nominal_sha256) {
    throw new TypeError("canon_historico_invalido");
  }
  const recurso = canon.recurso;
  const original = recurso?.original;
  const firmado = recurso?.firmado;
  const revision = firma.revision_pdf;
  if (!registro(recurso) || !registro(original) || !registro(firmado)
    || recurso.expediente_ref !== solicitud.expediente_ref
    || recurso.documento_ref !== firma.original.documento_ref
    || original.referencia !== firma.original.documento_ref
    || original.version !== firma.original.version
    || original.huella_sha256 !== firma.original.huella_sha256
    || firmado.referencia !== firma.documento_custodiado.documento_ref
    || firmado.version !== firma.documento_custodiado.version
    || firmado.huella_sha256 !== firma.documento_custodiado.huella_sha256
    || recurso.numero_firmas !== revision.orden_firma
    || canon.paso_ref !== firma.paso_ref || canon.paso_orden !== firma.paso_orden
    || canon.circuito?.referencia !== firma.catalogo_ref
    || canon.circuito?.huella_sha256 !== firma.catalogo_huella
    || canon.fecha_historica !== firma.registrada_en
    || revision.orden_firma === 1 && recurso.entrada_revision !== null
    || revision.orden_firma === 2 && (
      recurso.entrada_revision?.referencia !== revision.entrada_documento.documento_ref
      || recurso.entrada_revision?.version !== revision.entrada_documento.version
      || recurso.entrada_revision?.huella_sha256 !== revision.entrada_documento.huella_sha256
    )) throw new TypeError("canon_historico_cruzado");
  // Documentos usa una referencia opaca derivada; no es la referencia CT.
  const documental = "vec.contratacion_temporal.custodia_firmado.v1\u0000expediente\u0000"
    + recurso.organizacion_ref + "\u0000" + solicitud.expediente_ref + "\u0000";
  const refDocumental = "ref:" + await sha256Hex(new TextEncoder().encode(documental), cryptoImpl);
  if (firma.documento_custodiado.expediente_ref !== refDocumental) {
    throw new TypeError("custodia_historica_cruzada");
  }
}

// La respuesta de 48 campos es una capacidad del servidor. El canal proyecta
// once claves; conserva sólo las referencias y huellas tras verificar el canon.
export async function validarRespuestaRecuperacionFirmasV2(x, solicitud, cryptoImpl = globalThis.crypto) {
  const q = validarSolicitudRecuperacionFirmasV2(solicitud);
  if (!exacto(x, CAMPOS_RESPUESTA) || x.esquema !== ESQUEMA_RECUPERACION_FIRMAS_V2
    || x.expediente_ref !== q.expediente_ref || x.version_expediente !== q.version_expediente
    || x.documento !== q.documento || !entero(x.historia_revision)
    || !coincide(x.historia_sha256, SHA) || x.recuperacion !== "recuperada"
    || x.firma_eficaz !== false || !Array.isArray(x.campos_no_disponibles)
    || x.campos_no_disponibles.length !== 0 || !Array.isArray(x.firmas)
    || !Array.isArray(x.recuperaciones) || x.firmas.length > 128
    || x.recuperaciones.length > 128) throw new TypeError("respuesta_recuperacion_firmas_invalida");
  const firmas = new Map();
  for (const firma of x.firmas) {
    validarFirma(firma, q);
    if (firmas.has(firma.firma_ref)) throw new TypeError("firma_historica_duplicada");
    firmas.set(firma.firma_ref, firma);
  }
  for (const firma of firmas.values()) {
    if (firma.paso_orden !== 2 || firma.revision_pdf === null) continue;
    const revision = firma.revision_pdf;
    const previa = firmas.get(revision.firma_anterior_ref);
    const custodia = previa?.documento_custodiado;
    const revisionPrevia = previa?.revision_pdf;
    if (!previa || previa.paso_orden !== 1 || !custodia || !revisionPrevia
      || revision.recibo_anterior_ref !== previa.recibo_ref
      || firma.secuencia !== previa.secuencia + 1
      || firma.catalogo_ref !== previa.catalogo_ref
      || firma.catalogo_huella !== previa.catalogo_huella
      || firma.original.documento_ref !== previa.original.documento_ref
      || firma.original.version !== previa.original.version
      || firma.original.huella_sha256 !== previa.original.huella_sha256
      || revision.entrada_documento.documento_ref !== custodia.documento_ref
      || revision.entrada_documento.version !== custodia.version
      || revision.entrada_documento.huella_sha256 !== custodia.huella_sha256
      || revision.entrada_longitud !== revisionPrevia.revision_longitud) {
      throw new TypeError("cadena_historica_invalida");
    }
  }
  const recuperadas = new Map();
  for (const rec of x.recuperaciones) {
    const firma = firmas.get(rec?.firma_ref);
    if (!firma || firma.revision_pdf === null || recuperadas.has(rec.firma_ref)) {
      throw new TypeError("recuperacion_historica_cruzada");
    }
    await validarCanon(rec, firma, q, cryptoImpl);
    recuperadas.set(rec.firma_ref, rec);
  }
  if ([...firmas.values()].filter((firma) => firma.revision_pdf !== null).length !== recuperadas.size) {
    throw new TypeError("recuperacion_historica_incompleta");
  }
  return Object.freeze({
    expediente_ref: x.expediente_ref, version_expediente: x.version_expediente,
    documento: x.documento, historia_revision: x.historia_revision,
    historia_sha256: x.historia_sha256,
    firmas: Object.freeze(x.firmas.map((firma) => {
      const rec = recuperadas.get(firma.firma_ref);
      return Object.freeze({
        firma_ref: firma.firma_ref, recibo_ref: firma.recibo_ref,
        registrada_en: firma.registrada_en, resultado: firma.resultado,
        paso_orden: firma.paso_orden, via: firma.via,
        revision_sha256: firma.revision_pdf?.revision_sha256 ?? "",
        material_root_sha256: rec?.material_root_sha256 ?? "",
        canon_nominal_sha256: rec?.canon_nominal_sha256 ?? "",
        canon_nominal_ref: rec?.canon_nominal_ref ?? "",
      });
    })),
  });
}
