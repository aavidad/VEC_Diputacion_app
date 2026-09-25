import { textoRef } from "./texto-dietas.js?v=20260925-d5d6-v1";
const RUTA_ASIGNACION = "/api/vec/personal/asignaciones-dietas";
const MAXIMO_RESPUESTA = 64 * 1024;
const MOTIVOS_EMPLEADO = new Set(["empleado_no_disponible", "empleado_ambiguo"]);
const referencia = (valor, prefijo) => typeof valor === "string" &&
  new RegExp(`^${prefijo}[A-Za-z0-9_-]{22,128}$`, "u").test(valor);
const fecha = (valor) => {
  if (typeof valor !== "string" || !/^\d{4}-\d{2}-\d{2}$/u.test(valor)) return false;
  const [ano, mes, dia] = valor.split("-").map(Number);
  const fechaReal = new Date(Date.UTC(ano, mes - 1, dia));
  return fechaReal.getUTCFullYear() === ano && fechaReal.getUTCMonth() === mes - 1 && fechaReal.getUTCDate() === dia;
};
function validarAsignacion(cuerpo, relacion, unidad, fechaReferencia) {
  const asignacion = cuerpo?.asignacion;
  if (!asignacion || typeof asignacion !== "object" || Array.isArray(asignacion) ||
      !referencia(asignacion.asignacion_ref, "ads_") || asignacion.relacion_ref !== relacion ||
      asignacion.unidad_ref !== unidad || !textoRef(asignacion.centro_ref, 160) ||
      !referencia(asignacion.persona_ref, "per_") ||
      !referencia(asignacion.administrativo_persona_ref, "per_") ||
      !referencia(asignacion.responsable_persona_ref, "per_") ||
      new Set([asignacion.persona_ref, asignacion.administrativo_persona_ref, asignacion.responsable_persona_ref]).size !== 3 ||
      ![1, 2, 3].includes(asignacion.grupo_dieta) ||
      !Number.isSafeInteger(asignacion.version) || asignacion.version < 1 ||
      !fecha(asignacion.vigente_desde) || asignacion.vigente_desde > fechaReferencia ||
      !/^rad_[0-9a-f]{32}$/u.test(cuerpo.recibo_ref) || cuerpo.estado_local !== "consultada")
    throw new TypeError("asignación de Dietas incompatible");
  return Object.freeze({
    verificada: true,
    asignacion_ref: asignacion.asignacion_ref,
    relacion_ref: relacion,
    unidad_ref: unidad,
    fecha_referencia: fechaReferencia,
    centro_ref: asignacion.centro_ref,
    administrativo_persona_ref: asignacion.administrativo_persona_ref,
    responsable_persona_ref: asignacion.responsable_persona_ref,
    grupo_dieta: asignacion.grupo_dieta,
    version: asignacion.version,
    recibo_ref: cuerpo.recibo_ref,
  });
}
async function leerJSON(respuesta, signal) {
  const longitud = respuesta.headers?.get?.("content-length");
  if (longitud !== null && longitud !== undefined && (!/^\d+$/u.test(longitud) || Number(longitud) > MAXIMO_RESPUESTA))
    throw new TypeError("respuesta de asignación excesiva");
  if (!/^application\/json(?:;\s*charset=utf-8)?$/iu.test(respuesta.headers?.get?.("content-type") || "") ||
      !respuesta.body?.getReader) throw new TypeError("respuesta de asignación incompatible");
  const lector = respuesta.body.getReader();
  const partes = []; let longitudReal = 0;
  try {
    while (true) {
      if (signal?.aborted) throw new Error("operación cancelada");
      const tramo = await lector.read();
      if (tramo.done) break;
      if (!(tramo.value instanceof Uint8Array) || tramo.value.length === 0 || (longitudReal += tramo.value.length) > MAXIMO_RESPUESTA) throw new TypeError("respuesta de asignación incompatible");
      partes.push(tramo.value);
    }
    const total = partes.reduce((suma, parte) => suma + parte.length, 0);
    if (total > MAXIMO_RESPUESTA || (longitud !== null && Number(longitud) !== total)) throw new TypeError("respuesta de asignación incompatible");
    const bytes = new Uint8Array(total); let desplazamiento = 0;
    partes.forEach((parte) => { bytes.set(parte, desplazamiento); desplazamiento += parte.length; });
    return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes));
  } finally { try { await lector.cancel(); } catch {} }
}

/** Consulta la asignación D7 por relación y unidad acreditadas por Personal. */
export function crearClienteAsignacionDietasHTTP({ fetchImpl = globalThis.fetch } = {}) {
  if (typeof fetchImpl !== "function") throw new TypeError("cliente de asignación no disponible");
  async function obtenerJSON(ruta, signal) {
    const respuesta = await fetchImpl(ruta, { method: "GET", headers: { Accept: "application/json" },
      credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer", signal });
    if (signal?.aborted) throw new Error("operación cancelada");
    if (!respuesta || respuesta.redirected) throw new Error("respuesta de Personal incompatible");
    if (respuesta.status !== 200 || respuesta.ok !== true) {
      const error = new Error("consulta de Personal no disponible");
      error.codigo = respuesta.status === 401 ? "autenticacion_requerida" :
        respuesta.status === 403 ? "acceso_denegado" : respuesta.status === 404 ? "no_encontrada" : "no_disponible";
      // Un 403 puede traer el motivo cerrado de la falta de empleado canónico.
      if (respuesta.status === 403) {
        try {
          const motivo = (await leerJSON(respuesta, signal))?.codigo;
          if (MOTIVOS_EMPLEADO.has(motivo)) error.codigo = motivo;
        } catch { /* Sin cuerpo legible se conserva la denegación genérica. */ }
      }
      throw error;
    }
    return leerJSON(respuesta, signal);
  }
  return Object.freeze({
    async obtenerRelaciones({ signal } = {}) {
      const cuerpo = await obtenerJSON("/api/vec/personal/relaciones-dietas", signal);
      if (!Array.isArray(cuerpo?.relaciones_autorizadas) || cuerpo.relaciones_autorizadas.length > 64 ||
          !fecha(cuerpo.fecha_referencia)) throw new TypeError("relaciones de Dietas incompatibles");
      const relaciones = cuerpo.relaciones_autorizadas.map((entrada) => {
        if (!referencia(entrada?.relacion_ref, "rel_") || !textoRef(entrada?.unidad_ref, 256) ||
            !Number.isSafeInteger(entrada?.version) || entrada.version < 1)
          throw new TypeError("relación de Dietas incompatible");
        return Object.freeze({ relacion_ref: entrada.relacion_ref, unidad_ref: entrada.unidad_ref, version: entrada.version });
      });
      if (new Set(relaciones.map((entrada) => entrada.relacion_ref)).size !== relaciones.length)
        throw new TypeError("relaciones de Dietas duplicadas");
      return Object.freeze({ relaciones_autorizadas: Object.freeze(relaciones), fecha_referencia: cuerpo.fecha_referencia });
    },
    async obtener(relacion_ref, unidad_ref, fecha_referencia, { signal } = {}) {
      if (!referencia(relacion_ref, "rel_") || !textoRef(unidad_ref, 256) || !fecha(fecha_referencia) ||
          (signal !== undefined && (!signal || typeof signal.aborted !== "boolean")))
        throw new TypeError("consulta de asignación no válida");
      const parametros = new URLSearchParams({ fecha_referencia, unidad_ref });
      const ruta = `${RUTA_ASIGNACION}/${encodeURIComponent(relacion_ref)}?${parametros}`;
      const cuerpo = await obtenerJSON(ruta, signal);
      return validarAsignacion(cuerpo, relacion_ref, unidad_ref, fecha_referencia);
    },
  });
}
