export const RUTA_BORRADORES_LLAMAMIENTO = "/api/vec/bolsa/llamamientos/borradores";

const MAXIMO_RESPUESTA_BYTES = 16 * 1024;
const PATRON_REFERENCIA = /^borrador-llamamiento:alta:[0-9a-f]{64}$/;
const PATRON_RECIBO = /^recibo:[a-p]{64}$/;
const PATRON_CLAVE = /^[A-Za-z0-9][A-Za-z0-9:._/-]{7,191}$/;
const PATRON_INSTANTE = /^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d{1,6})?Z$/;

export class ErrorAPIBorradorLlamamiento extends Error {
  constructor(mensaje, estado = 0, codigo = "error_cliente") {
    super(mensaje);
    this.name = "ErrorAPIBorradorLlamamiento";
    this.estado = estado;
    this.codigo = codigo;
  }
}

export const MENSAJES_BORRADOR_LLAMAMIENTO = Object.freeze({
  401: "Se requiere autenticación interna para preparar el borrador.",
  403: "La sesión no dispone de autorización para preparar este borrador.",
  404: "El borrador no existe o no es visible en el ámbito actual.",
  409: "La clave de reintento ya corresponde a otro contenido. Revise el resumen antes de volver a intentarlo.",
  413: "El resumen o la respuesta supera el límite admitido.",
  500: "El servicio de borradores de llamamiento no está disponible temporalmente.",
});

function mensajeEstado(estado) {
  return MENSAJES_BORRADOR_LLAMAMIENTO[estado]
    || (estado >= 500 ? MENSAJES_BORRADOR_LLAMAMIENTO[500] : `La solicitud fue rechazada (HTTP ${estado}).`);
}

function validarResumen(resumen) {
  const longitud = typeof resumen === "string" ? new TextEncoder().encode(resumen).byteLength : 0;
  if (typeof resumen !== "string" || resumen !== resumen.trim() || longitud < 3 || longitud > 2000
    || /[\0\u2028\u2029]/u.test(resumen)) {
    throw new ErrorAPIBorradorLlamamiento("El resumen debe ocupar entre 3 y 2.000 bytes UTF-8, sin espacios iniciales ni finales.");
  }
  return resumen;
}

function validarClave(clave) {
  if (typeof clave !== "string" || !PATRON_CLAVE.test(clave) || clave !== clave.trim()) {
    throw new ErrorAPIBorradorLlamamiento("La clave de reintento no es válida.");
  }
  return clave;
}

function validarRecibo(envelope) {
  const data = envelope?.data;
  const campos = ["borrador_ref", "estado", "version", "resumen", "recibo_ref", "registrado_en", "reintento_idempotente"];
  if (!data || typeof data !== "object" || Array.isArray(data) || Object.keys(data).length !== campos.length
    || campos.some((campo) => !Object.hasOwn(data, campo)) || !PATRON_REFERENCIA.test(data.borrador_ref)
    || data.estado !== "borrador_interno" || data.version !== "1" || !PATRON_RECIBO.test(data.recibo_ref)
    || !PATRON_INSTANTE.test(data.registrado_en) || typeof data.reintento_idempotente !== "boolean") {
    throw new ErrorAPIBorradorLlamamiento("La respuesta de borrador no respeta el contrato esperado.");
  }
  validarResumen(data.resumen);
  return Object.freeze({ ...data });
}

async function leerJSON(respuesta) {
  const tipo = respuesta?.headers?.get?.("Content-Type") || "";
  if (!/^application\/json(?:;\s*charset=utf-8)?$/i.test(tipo)) {
    throw new ErrorAPIBorradorLlamamiento("La respuesta no es JSON canónico.");
  }
  const longitud = respuesta?.headers?.get?.("Content-Length");
  if (typeof longitud !== "string" || !/^(?:0|[1-9][0-9]*)$/.test(longitud) || Number(longitud) > MAXIMO_RESPUESTA_BYTES) {
    throw new ErrorAPIBorradorLlamamiento("La respuesta supera el límite admitido.", 413);
  }
  const texto = await respuesta.text();
  if (new TextEncoder().encode(texto).byteLength !== Number(longitud)) {
    throw new ErrorAPIBorradorLlamamiento("La respuesta no coincide con su longitud declarada.");
  }
  try { return JSON.parse(texto); } catch { throw new ErrorAPIBorradorLlamamiento("La respuesta no contiene JSON válido."); }
}

function claveAleatoria(criptografia = globalThis.crypto) {
  if (!criptografia?.getRandomValues) throw new ErrorAPIBorradorLlamamiento("No hay generador criptográfico disponible.");
  const bytes = new Uint8Array(24);
  criptografia.getRandomValues(bytes);
  return `blam-${Array.from(bytes, (byte) => byte.toString(16).padStart(2, "0")).join("")}`;
}

export function crearClienteBorradorLlamamiento({ fetchImpl = globalThis.fetch, criptografia = globalThis.crypto } = {}) {
  if (typeof fetchImpl !== "function") throw new TypeError("fetch no está disponible");
  async function ejecutar(ruta, opciones, signal) {
    let respuesta;
    try {
      respuesta = await fetchImpl(ruta, { ...opciones, signal, credentials: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer" });
      const envelope = await leerJSON(respuesta);
      if (!respuesta.ok) {
        const codigo = typeof envelope?.error?.codigo === "string" ? envelope.error.codigo : "error_http";
        throw new ErrorAPIBorradorLlamamiento(mensajeEstado(respuesta.status), respuesta.status, codigo);
      }
      return validarRecibo(envelope);
    } catch (error) {
      if (error?.name === "AbortError") throw error;
      throw error instanceof ErrorAPIBorradorLlamamiento ? error : new ErrorAPIBorradorLlamamiento("No se pudo completar la comunicación con el servicio.");
    }
  }
  return Object.freeze({
    crear: ({ resumen, claveIdempotencia = claveAleatoria(criptografia), signal } = {}) => ejecutar(RUTA_BORRADORES_LLAMAMIENTO, {
      method: "POST", headers: { Accept: "application/json", "Content-Type": "application/json", "Idempotency-Key": validarClave(claveIdempotencia) },
      body: JSON.stringify({ resumen: validarResumen(resumen) }),
    }, signal),
    consultar: ({ referencia, signal } = {}) => {
      if (typeof referencia !== "string" || !PATRON_REFERENCIA.test(referencia)) throw new ErrorAPIBorradorLlamamiento("La referencia de borrador no es válida.");
      return ejecutar(`${RUTA_BORRADORES_LLAMAMIENTO}/${referencia}`, { method: "GET", headers: { Accept: "application/json" } }, signal);
    },
    generarClave: () => claveAleatoria(criptografia),
  });
}
