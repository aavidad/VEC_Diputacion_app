/**
 * Cliente HTTP del registro de firmas de prueba de los borradores. Consulta el
 * estado real del circuito de un expediente y registra una firma (ya hecha
 * con AutoFirma) o una devolución. El servidor verifica la firma; este
 * cliente solo valida la forma de las respuestas y falla cerrado.
 */

export const RUTA_FIRMA_DOCUMENTO = "/api/vec/contratacion-temporal/firmas-documento";
export const RUTA_CONSULTA_FIRMA_DOCUMENTO = "/api/vec/contratacion-temporal/firmas-documento/consultas";
const ESQUEMA_ESTADO = "vec.contratacion-temporal.estado-firmas-documento.v1";
const ESQUEMA_RECIBO = "vec.contratacion-temporal.recibo-firma-documento.v1";
const MAXIMO_RESPUESTA = 256 * 1024;
const REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const HUELLA = /^[0-9a-f]{64}$/u;
const ESTADOS = ["pendiente_firma", "en_espera", "firmado", "devuelto"];
const CODIGOS_ERROR = new Set([
  "recurso_no_encontrado", "metodo_no_permitido", "peticion_no_permitida", "contenido_no_valido", "acceso_denegado",
  "verificacion_no_disponible", "firma_no_verificada", "paso_no_pendiente", "cadena_rota", "conflicto",
  "circuito_no_disponible", "resultado_no_confiable", "servicio_no_disponible",
]);

/** Error del registro con código cerrado y, si procede, motivo del validador. */
export class ErrorFirmaDocumento extends Error {
  constructor(codigo, motivo = "") {
    super(codigo);
    this.name = "ErrorFirmaDocumento";
    this.codigo = codigo;
    this.motivo = motivo;
  }
}

function objetoPlano(valor) {
  return valor !== null && typeof valor === "object" && !Array.isArray(valor) && Object.getPrototypeOf(valor) === Object.prototype;
}

function soloClaves(valor, obligatorias, opcionales = []) {
  return objetoPlano(valor) && obligatorias.every((c) => Object.hasOwn(valor, c))
    && Object.keys(valor).every((c) => obligatorias.includes(c) || opcionales.includes(c));
}

function texto(valor, maximo = 512) {
  return typeof valor === "string" && valor.length > 0 && valor.length <= maximo;
}

function validarPaso(paso, indice) {
  if (!soloClaves(paso, ["orden", "cargo", "accion", "devolucion", "estado"], ["motivo_devolucion", "recibo_ref", "registrada_en"])
    || paso.orden !== indice + 1 || !texto(paso.cargo) || !ESTADOS.includes(paso.estado)
    || (paso.motivo_devolucion !== undefined && !texto(paso.motivo_devolucion, 500))
    || (paso.recibo_ref !== undefined && !REFERENCIA.test(paso.recibo_ref))
    || (paso.registrada_en !== undefined && !texto(paso.registrada_en, 64))) return null;
  return Object.freeze({ ...paso });
}

/** Valida el estado real del circuito; cualquier desviación lo invalida. */
export function validarEstadoFirmas(datos) {
  if (!soloClaves(datos, ["esquema", "catalogo_ref", "huella_sha256", "ejemplo", "firma_eficaz", "verificacion_disponible", "documentos"])
    || datos.esquema !== ESQUEMA_ESTADO || datos.firma_eficaz !== false || !HUELLA.test(datos.huella_sha256)
    || typeof datos.ejemplo !== "boolean" || typeof datos.verificacion_disponible !== "boolean"
    || !Array.isArray(datos.documentos) || datos.documentos.length > 32) return null;
  const documentos = [];
  for (const d of datos.documentos) {
    if (!soloClaves(d, ["documento", "etiqueta", "paso_pendiente", "completo", "ultima_secuencia", "pasos"], ["original_esperado_sha256"])
      || !/^[a-z][a-z0-9_]{1,63}$/u.test(d.documento) || !Number.isSafeInteger(d.paso_pendiente) || d.paso_pendiente < 0
      || typeof d.completo !== "boolean" || d.completo !== (d.paso_pendiente === 0) || !Number.isSafeInteger(d.ultima_secuencia)
      || !Array.isArray(d.pasos) || d.pasos.length < 1 || d.pasos.length > 16 || d.paso_pendiente > d.pasos.length
      || (d.original_esperado_sha256 !== undefined && !HUELLA.test(d.original_esperado_sha256))) return null;
    const pasos = d.pasos.map(validarPaso);
    if (pasos.includes(null)) return null;
    documentos.push(Object.freeze({ ...d, pasos: Object.freeze(pasos) }));
  }
  return Object.freeze({ ...datos, documentos: Object.freeze(documentos) });
}

function validarRecibo(datos) {
  const firmado = datos?.resultado === "firmado";
  if (!soloClaves(datos, ["esquema", "firma_ref", "recibo_ref", "expediente_ref", "expediente_version", "documento", "paso_orden",
    "paso_ref", "catalogo_ref", "catalogo_huella_sha256", "secuencia", "resultado", "perfil_ref", "registrada_en", "ya_registrada",
    "firma_verificada", "firma_eficaz", firmado ? "verificacion" : "motivo_devolucion"])
    || datos.esquema !== ESQUEMA_RECIBO || datos.firma_eficaz !== false || datos.firma_verificada !== firmado
    || !REFERENCIA.test(datos.recibo_ref) || !REFERENCIA.test(datos.firma_ref) || !Number.isSafeInteger(datos.secuencia)
    || !["firmado", "devuelto"].includes(datos.resultado) || !texto(datos.registrada_en, 64)) return null;
  if (firmado && (!objetoPlano(datos.verificacion) || datos.verificacion.estado !== "valida"
    || datos.verificacion.motivo !== "verificada" || !HUELLA.test(datos.verificacion.firmado_sha256 ?? ""))) return null;
  return Object.freeze({ ...datos });
}

async function leerJSON(respuesta) {
  const longitud = respuesta.headers.get("Content-Length");
  if (longitud !== null && Number(longitud) > MAXIMO_RESPUESTA) throw new ErrorFirmaDocumento("resultado_no_confiable");
  if (!/^application\/json(?:;\s*charset=utf-8)?$/iu.test(respuesta.headers.get("Content-Type") ?? "")) {
    throw new ErrorFirmaDocumento("resultado_no_confiable");
  }
  const cuerpo = await respuesta.text();
  if (cuerpo.length > MAXIMO_RESPUESTA) throw new ErrorFirmaDocumento("resultado_no_confiable");
  return JSON.parse(cuerpo);
}

function base64(bytes) {
  let binario = "";
  for (let i = 0; i < bytes.length; i += 0x8000) binario += String.fromCharCode(...bytes.subarray(i, i + 0x8000));
  return btoa(binario);
}

export function crearClienteFirmaDocumento({ fetchImpl = globalThis.fetch } = {}) {
  async function pedir(ruta, cuerpo, signal) {
    if (typeof fetchImpl !== "function") throw new ErrorFirmaDocumento("servicio_no_disponible");
    let respuesta;
    try {
      respuesta = await fetchImpl(ruta, {
        method: "POST", headers: { "Content-Type": "application/json", Accept: "application/json" },
        body: JSON.stringify(cuerpo), signal, mode: "same-origin", credentials: "same-origin",
        cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
      });
    } catch {
      throw new ErrorFirmaDocumento(signal?.aborted ? "operacion_abortada" : "servicio_no_disponible");
    }
    if (respuesta.redirected) throw new ErrorFirmaDocumento("resultado_no_confiable");
    let datos;
    try { datos = await leerJSON(respuesta); } catch (error) {
      if (error instanceof ErrorFirmaDocumento) throw error;
      throw new ErrorFirmaDocumento("resultado_no_confiable");
    }
    if (respuesta.status >= 400) {
      const e = datos?.error;
      const codigo = soloClaves(e, ["codigo", "clave_i18n", "correlacion_ref"], ["motivo"]) && CODIGOS_ERROR.has(e.codigo)
        ? e.codigo : "servicio_no_disponible";
      throw new ErrorFirmaDocumento(codigo, typeof e?.motivo === "string" && /^[a-z_]{3,64}$/u.test(e.motivo) ? e.motivo : "");
    }
    if (!soloClaves(datos, ["data"])) throw new ErrorFirmaDocumento("resultado_no_confiable");
    return { estado: respuesta.status, datos: datos.data };
  }

  return Object.freeze({
    /** Estado real del circuito; null si el registro no está compuesto. */
    async consultar(expedienteRef, { signal } = {}) {
      if (!REFERENCIA.test(expedienteRef ?? "")) return null;
      try {
        const { estado, datos } = await pedir(RUTA_CONSULTA_FIRMA_DOCUMENTO, { expediente_ref: expedienteRef }, signal);
        return estado === 200 ? validarEstadoFirmas(datos) : null;
      } catch {
        return null;
      }
    },
    /** Registra una firma verificada o una devolución y devuelve el recibo. */
    async registrar({ expedienteRef, version, documento, pasoOrden, resultado, motivoDevolucion = "", original, firmado, clave }, { signal } = {}) {
      const esFirma = resultado === "firmado";
      if (!REFERENCIA.test(expedienteRef ?? "") || !Number.isSafeInteger(version) || !Number.isSafeInteger(pasoOrden)
        || !["firmado", "devuelto"].includes(resultado) || (esFirma && (!(original instanceof Uint8Array) || !(firmado instanceof Uint8Array)))
        || !/^[A-Za-z0-9][A-Za-z0-9._-]{15,63}$/u.test(clave ?? "")) throw new ErrorFirmaDocumento("contenido_no_valido");
      const { estado, datos } = await pedir(RUTA_FIRMA_DOCUMENTO, {
        expediente_ref: expedienteRef, version_expediente: version, documento, paso_orden: pasoOrden, resultado,
        motivo_devolucion: esFirma ? "" : motivoDevolucion,
        original_base64: esFirma ? base64(original) : "", firmado_base64: esFirma ? base64(firmado) : "", clave_idempotencia: clave,
      }, signal);
      const recibo = (estado === 200 || estado === 201) ? validarRecibo(datos) : null;
      if (!recibo) throw new ErrorFirmaDocumento("resultado_no_confiable");
      return recibo;
    },
  });
}
