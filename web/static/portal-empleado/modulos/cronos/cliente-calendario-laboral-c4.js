import { leerJSONAcotado } from "./cliente-saldo-http.js";

export const RUTA_CALENDARIO_LABORAL_C4 = "/api/interna/cronos/calendario/propio";

export class ErrorCalendarioLaboralC4 extends Error {
  constructor(codigo, estado = 0) {
    super(`calendario laboral: ${codigo}`);
    this.codigo = codigo;
    this.estado = estado;
  }
}

export function fechaCalendarioC4Valida(valor) {
  if (typeof valor !== "string" || !/^\d{4}-\d{2}-\d{2}$/u.test(valor)) return false;
  const [anio, mes, dia] = valor.split("-").map(Number);
  const fecha = new Date(Date.UTC(anio, mes - 1, dia));
  return anio >= 1900 && anio <= 2100 && fecha.getUTCFullYear() === anio
    && fecha.getUTCMonth() === mes - 1 && fecha.getUTCDate() === dia;
}

export function validarCalendarioLaboralC4(valor, fecha) {
  const objeto = (v) => v !== null && typeof v === "object" && !Array.isArray(v);
  if (!objeto(valor) || valor.fecha !== fecha || !["no_determinable", "calendario_publicado_jornada_no_determinable", "programado"].includes(valor.estado)
    || !Array.isArray(valor.versiones_calendario) || valor.versiones_calendario.length > 4
    || !Object.hasOwn(valor, "minutos_programados")
    || (valor.estado === "programado" ? (!Number.isSafeInteger(valor.minutos_programados) || valor.minutos_programados < 0 || valor.minutos_programados > 1440
      || !valor.turno_ref || !valor.regimen_ref || !valor.relacion_ref || !valor.programacion_ref || !valor.politica_version_ref
      || !valor.fuente_programacion_ref || !Number.isSafeInteger(valor.version_programacion) || valor.version_programacion < 1
      || !Number.isSafeInteger(valor.fuente_version_programacion) || valor.fuente_version_programacion < 1
      || !Number.isFinite(Date.parse(valor.inicio_utc)) || !Number.isFinite(Date.parse(valor.fin_utc))
      || Date.parse(valor.fin_utc) <= Date.parse(valor.inicio_utc) || Date.parse(valor.fin_utc) - Date.parse(valor.inicio_utc) > 86_400_000) : valor.minutos_programados !== null)) {
    throw new ErrorCalendarioLaboralC4("respuesta_incompatible", 200);
  }
  if (valor.estado !== "no_determinable") {
    if (![valor.centro_ref, valor.fuente_unidad_ref, valor.fuente_unidad_version, valor.fuente_organizacion_ref]
      .every((v) => typeof v === "string" && v.length > 0 && v.length <= 255)
      || !Number.isSafeInteger(valor.version_organizacion)
      || valor.version_organizacion < 1 || valor.versiones_calendario.length !== 4 || !objeto(valor.dia)
      || typeof valor.dia.laborable !== "boolean" || valor.versiones_calendario.some((v) => !objeto(v)
        || typeof v.denominacion !== "string" || !v.denominacion.trim() || !Number.isSafeInteger(v.numero) || v.numero < 1)) {
      throw new ErrorCalendarioLaboralC4("respuesta_incompatible", 200);
    }
  }
  return valor;
}

export function crearClienteCalendarioLaboralC4({ fetchImpl = globalThis.fetch, plazoMs = 10_000 } = {}) {
  if (typeof fetchImpl !== "function" || !Number.isSafeInteger(plazoMs) || plazoMs < 1 || plazoMs > 30_000) throw new TypeError("cliente de calendario no disponible");
  return Object.freeze({
    async consultar(fecha, { signal } = {}) {
      if (!fechaCalendarioC4Valida(fecha)) throw new TypeError("fecha de calendario no válida");
      if (signal !== undefined && (!signal || typeof signal.aborted !== "boolean" || typeof signal.addEventListener !== "function")) throw new TypeError("signal de calendario no válido");
      if (signal?.aborted) throw new ErrorCalendarioLaboralC4("operacion_abortada");
      const plazo = AbortSignal.timeout(plazoMs);
      const senal = signal ? AbortSignal.any([signal, plazo]) : plazo;
      let respuesta;
      try {
        respuesta = await fetchImpl(`${RUTA_CALENDARIO_LABORAL_C4}?fecha=${encodeURIComponent(fecha)}`, {
          method: "GET", credentials: "omit", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer", signal: senal,
          headers: { Accept: "application/json" },
        });
        if (respuesta.redirected || !Number.isInteger(respuesta.status)) throw new ErrorCalendarioLaboralC4("respuesta_incompatible");
        if (respuesta.status === 401 || respuesta.status === 403) throw new ErrorCalendarioLaboralC4("acceso_denegado", respuesta.status);
        if (respuesta.status !== 200 || respuesta.ok !== true) throw new ErrorCalendarioLaboralC4("no_disponible", respuesta.status);
        if (!/^application\/json(?:;\s*charset=utf-8)?$/iu.test(respuesta.headers?.get?.("content-type") || "")) throw new ErrorCalendarioLaboralC4("respuesta_incompatible",200);
        const cuerpo = await leerJSONAcotado(respuesta);
        if (signal?.aborted || plazo.aborted) throw new ErrorCalendarioLaboralC4(signal?.aborted ? "operacion_abortada" : "plazo_agotado");
        return validarCalendarioLaboralC4(cuerpo, fecha);
      } catch (error) {
        if (signal?.aborted) throw new ErrorCalendarioLaboralC4("operacion_abortada");
        if (error instanceof ErrorCalendarioLaboralC4) throw error;
        throw new ErrorCalendarioLaboralC4("no_disponible", respuesta?.status ?? 0);
      }
    },
  });
}
