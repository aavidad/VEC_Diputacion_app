import { leerJSONAcotado } from "./cliente-saldo-http.js";

export const RUTAS_JUSTIFICANTES_CRONOS = Object.freeze({
  aportar: "/api/interna/cronos/permisos/justificantes",
  revisar: "/api/interna/cronos/permisos/justificantes/revisiones",
});

export class ErrorClienteJustificantesCronos extends Error {
  constructor(codigo, estado = 0) {
    super(`cronos justificantes: ${codigo}`);
    this.codigo = codigo;
    this.estado = estado;
  }
}

const ref = (v) => typeof v === "string" && /^ref:[0-9a-f]{64}$/u.test(v);
const solicitud = (v) => typeof v === "string" && /^permiso:cronos:solicitud:[A-Za-z0-9:._-]{1,128}$/u.test(v);
const entero = (v, minimo = 0) => Number.isSafeInteger(v) && v >= minimo;
const sha = (v) => typeof v === "string" && /^[0-9a-f]{64}$/u.test(v);
const objeto = (v) => v !== null && typeof v === "object" && !Array.isArray(v);

export function validarAporteJustificanteCronos(v) {
  if (!objeto(v) || !solicitud(v.solicitud_ref) || !ref(v.clave_operacion) ||
    !ref(v.documento_id) || !entero(v.documento_version, 1) || !sha(v.sha256) || !entero(v.version_esperada)) {
    throw new TypeError("aporte no válido");
  }
  return Object.freeze({ solicitud_ref: v.solicitud_ref, clave_operacion: v.clave_operacion,
    documento_id: v.documento_id, documento_version: String(v.documento_version),
    sha256: v.sha256, version_esperada: String(v.version_esperada) });
}

export function validarRevisionJustificanteCronos(v) {
  if (!objeto(v) || !solicitud(v.solicitud_ref) || !ref(v.clave_operacion) ||
    !entero(v.version_esperada, 1) || !["aceptada", "rechazada"].includes(v.decision) ||
    typeof v.motivo_ref !== "string" || !/^[a-z][a-z0-9:._-]{2,159}$/u.test(v.motivo_ref)) {
    throw new TypeError("revisión no válida");
  }
  return Object.freeze({ solicitud_ref: v.solicitud_ref, clave_operacion: v.clave_operacion,
    version_esperada: String(v.version_esperada), decision: v.decision, motivo_ref: v.motivo_ref });
}

function validarRecibo(v, entrada, estado) {
  const r = v?.recibo;
  if (!objeto(v) || !objeto(r) || r.solicitud_ref !== entrada.solicitud_ref ||
    (entrada.documento_id && r.documento_id !== entrada.documento_id) || !ref(r.documento_id) ||
    !entero(r.documento_version, 1) || !sha(r.sha256) || r.estado !== estado ||
    r.version !== Number(entrada.version_esperada) + 1 ||
    typeof r.recibo_ref !== "string" || !r.recibo_ref.startsWith("recibo:") ||
    !Number.isFinite(Date.parse(r.instante_utc)) || typeof r.replay !== "boolean") {
    throw new ErrorClienteJustificantesCronos("respuesta_incompatible", 200);
  }
  if (entrada.sha256 && (r.sha256 !== entrada.sha256 || r.documento_version !== Number(entrada.documento_version))) {
    throw new ErrorClienteJustificantesCronos("respuesta_incompatible", 200);
  }
  return Object.freeze(r);
}

export function crearClienteJustificantesCronosHTTP({ fetchImpl = globalThis.fetch, plazoMs = 10_000 } = {}) {
  if (typeof fetchImpl !== "function" || !entero(plazoMs, 1) || plazoMs > 30_000) throw new TypeError("cliente no disponible");

  async function enviar(ruta, entrada, estado, signal) {
    const controlador = new AbortController();
    const abortar = () => controlador.abort();
    signal?.addEventListener("abort", abortar, { once: true });
    const timer = setTimeout(abortar, plazoMs);
    try {
      let respuesta;
      try {
        respuesta = await fetchImpl(ruta, { method: "POST", headers: { Accept: "application/json", "Content-Type": "application/json" },
          body: JSON.stringify(entrada), credentials: "same-origin", mode: "same-origin", cache: "no-store",
          redirect: "error", referrerPolicy: "no-referrer", signal: controlador.signal });
      } catch { throw new ErrorClienteJustificantesCronos(controlador.signal.aborted ? "plazo_agotado" : "red_no_disponible"); }
      if (controlador.signal.aborted || respuesta.redirected || ![200, 201, 202, 400, 401, 403, 409, 503].includes(respuesta.status) ||
        !/^application\/json(?:;\s*charset=utf-8)?$/iu.test(respuesta.headers?.get?.("content-type") || "")) {
        throw new ErrorClienteJustificantesCronos("respuesta_incompatible", respuesta.status);
      }
      let contenido;
      try { contenido = await leerJSONAcotado(respuesta); }
      catch { throw new ErrorClienteJustificantesCronos("respuesta_incompatible", respuesta.status); }
      if (respuesta.status === 202) {
        if (contenido?.enlace_pendiente !== true || Object.hasOwn(contenido, "recibo")) throw new ErrorClienteJustificantesCronos("respuesta_incompatible", 202);
        return Object.freeze({ enlace_pendiente: true });
      }
      if (respuesta.status !== 200 && respuesta.status !== 201) {
        const codigo = ({ 400: "peticion_invalida", 401: "autenticacion_requerida", 403: "acceso_denegado",
          409: "estado_cambiado", 503: "servicio_no_disponible" })[respuesta.status];
        throw new ErrorClienteJustificantesCronos(codigo, respuesta.status);
      }
      const recibo = validarRecibo(contenido, entrada, estado);
      if (recibo.replay !== (respuesta.status === 200)) throw new ErrorClienteJustificantesCronos("respuesta_incompatible", respuesta.status);
      return recibo;
    } finally { clearTimeout(timer); signal?.removeEventListener("abort", abortar); }
  }

  return Object.freeze({
    aportar: (v, signal) => enviar(RUTAS_JUSTIFICANTES_CRONOS.aportar, validarAporteJustificanteCronos(v), "pendiente_revision", signal),
    revisar: (v, signal) => enviar(RUTAS_JUSTIFICANTES_CRONOS.revisar, validarRevisionJustificanteCronos(v), v.decision, signal),
  });
}
