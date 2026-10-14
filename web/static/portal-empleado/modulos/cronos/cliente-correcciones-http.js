import { ErrorClienteSaldoCronos, leerJSONAcotado } from "./cliente-saldo-http.js";

export const RUTAS_CORRECCIONES_CRONOS = Object.freeze({
  bandeja: "/api/interna/cronos/correcciones/bandeja",
  actuaciones: "/api/interna/cronos/correcciones/actuaciones",
  historia: "/api/interna/cronos/correcciones/historia",
});
export const PASOS_BANDEJA_CORRECCIONES = Object.freeze(["responsable", "administracion"]);

export class ErrorClienteCorreccionesCronos extends Error {
  constructor(codigo, estado = 0) {
    super(`cliente de correcciones Cronos: ${codigo}`);
    this.name = "ErrorClienteCorreccionesCronos";
    this.codigo = codigo;
    this.estado = estado;
  }
}

const MOVIMIENTOS = new Set(["entrada", "salida", "inicio_pausa", "fin_pausa"]);
const ESTADOS = new Set(["pendiente_responsable", "pendiente_rrhh", "denegada_responsable", "denegada_rrhh", "pendiente_aplicacion", "aplicada"]);
const ESTADO_POR_ACTO = Object.freeze({
  "responsable:favorable": ["pendiente_rrhh", 2],
  "responsable:desfavorable": ["denegada_responsable", 2],
  "rrhh:favorable": ["pendiente_aplicacion", 3],
  "rrhh:desfavorable": ["denegada_rrhh", 3],
  "aplicar:": ["aplicada", 4],
});
const UUID_V4 = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/u;
const REF = /^[A-Za-z0-9:._-]{1,200}$/u;
const fecha = (v) => {
  if (typeof v !== "string" || !/^\d{4}-\d{2}-\d{2}$/u.test(v)) return false;
  const [a, m, d] = v.split("-").map(Number); const f = new Date(Date.UTC(a, m - 1, d));
  return f.getUTCFullYear() === a && f.getUTCMonth() === m - 1 && f.getUTCDate() === d;
};
const hora = (v) => typeof v === "string" && /^(?:[01]\d|2[0-3]):[0-5]\d$/u.test(v);
const instante = (v) => typeof v === "string" && /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?(?:Z|[+-]\d{2}:\d{2})$/u.test(v) && Number.isFinite(Date.parse(v));
const objeto = (v) => v !== null && typeof v === "object" && !Array.isArray(v);
const campos = (v, requeridos, opcionales = []) => objeto(v) && requeridos.every((k) => Object.hasOwn(v, k)) && Object.keys(v).every((k) => requeridos.includes(k) || opcionales.includes(k));
const ref = (v, prefijo) => typeof v === "string" && v.startsWith(prefijo) && REF.test(v);
const entero = (v, min = 0) => Number.isSafeInteger(v) && v >= min;
const fallo = (codigo, estado = 0) => new ErrorClienteCorreccionesCronos(codigo, estado);
const incompatible = () => fallo("respuesta_incompatible", 200);

export function motivoCorreccionValido(motivo) {
  return typeof motivo === "string" && motivo.length > 0 && [...motivo].length <= 500
    && motivo === motivo.trim() && !/\p{Cc}/u.test(motivo);
}

export function validarBandejaCorreccionesCronos(v, paso) {
  if (!PASOS_BANDEJA_CORRECCIONES.includes(paso) || !campos(v, ["paso", "pendientes"]) || v.paso !== paso
    || !Array.isArray(v.pendientes) || v.pendientes.length > 500) throw incompatible();
  for (const s of v.pendientes) {
    if (!campos(s, ["solicitud_ref", "empleado_ref", "empleado_etiqueta", "marcaje_original_ref", "hueco_declarado", "movimiento", "fecha_civil", "hora_pretendida", "motivo_codigo", "estado", "version", "solicitada_en"])
      || !ref(s.solicitud_ref, "correccion:cronos:") || !ref(s.empleado_ref, "emp_")
      || typeof s.empleado_etiqueta !== "string" || [...s.empleado_etiqueta].length > 120
      || (s.marcaje_original_ref !== "" && !REF.test(s.marcaje_original_ref)) || typeof s.hueco_declarado !== "boolean"
      || !MOVIMIENTOS.has(s.movimiento) || !fecha(s.fecha_civil) || !hora(s.hora_pretendida) || s.motivo_codigo !== "olvido_marcaje"
      || !ESTADOS.has(s.estado) || !entero(s.version, 1) || !instante(s.solicitada_en)
      || (paso === "responsable" && (s.estado !== "pendiente_responsable" || s.version !== 1))
      || (paso === "administracion" && ![["pendiente_rrhh", 2], ["pendiente_aplicacion", 3]].some(([e, n]) => s.estado === e && s.version === n))) throw incompatible();
  }
  return v;
}

export function validarEntradaActuacionCorreccion(e) {
  const esperado = ESTADO_POR_ACTO[`${e?.paso}:${e?.resultado}`];
  if (!campos(e, ["solicitud_ref", "clave_operacion", "paso", "resultado", "motivo", "version_esperada"])
    || !ref(e.solicitud_ref, "correccion:cronos:") || !UUID_V4.test(e.clave_operacion)
    || !esperado || e.version_esperada !== String(esperado[1] - 1) || !motivoCorreccionValido(e.motivo)) throw new TypeError("actuación de corrección no válida");
  return Object.freeze({ ...e });
}

export function validarReciboActuacionCorreccion(v, entrada) {
  const r = v?.recibo;
  const [estado, version] = ESTADO_POR_ACTO[`${entrada.paso}:${entrada.resultado}`] ?? [];
  if (!campos(v, ["recibo"]) || !campos(r, ["solicitud_ref", "actuacion_ref", "recibo_ref", "estado", "version", "instante_utc", "replay"])
    || r.solicitud_ref !== entrada.solicitud_ref || !ref(r.actuacion_ref, "correccion:actuacion:") || !ref(r.recibo_ref, "recibo:cronos:")
    || r.estado !== estado || r.version !== version || !instante(r.instante_utc) || typeof r.replay !== "boolean") throw incompatible();
  return r;
}

export function validarHistoriaCorreccionCronos(h, solicitudRef, vista) {
  if (!campos(h, ["solicitud_ref", "empleado_ref", "estado", "version", "fecha_civil", "hora_pretendida", "movimiento", "motivo_codigo", "actuaciones", "asientos"])
    || h.solicitud_ref !== solicitudRef || !ref(h.empleado_ref, "emp_") || !ESTADOS.has(h.estado)
    || !entero(h.version, 1) || !fecha(h.fecha_civil) || !hora(h.hora_pretendida) || !MOVIMIENTOS.has(h.movimiento)
    || h.motivo_codigo !== "olvido_marcaje" || !Array.isArray(h.actuaciones) || h.actuaciones.length !== h.version
    || h.actuaciones.length > 4 || !Array.isArray(h.asientos) || h.asientos.length > 2) throw incompatible();
  for (let i = 0; i < h.actuaciones.length; i++) {
    const a = h.actuaciones[i];
    if (!campos(a, ["paso", "resultado", "motivo", "actor_ref", "perfil_ref", "estado", "version", "recibo_ref", "instante_utc"])
      || !["solicitud", "decision_responsable", "resolucion_rrhh", "aplicacion"].includes(a.paso)
      || !["", "favorable", "desfavorable"].includes(a.resultado) || typeof a.motivo !== "string" || [...a.motivo].length > 500
      || (vista === "propia" && (a.actor_ref !== "" || a.perfil_ref !== ""))
      || (vista === "equipo" && (!ref(a.actor_ref, "per_") || !ref(a.perfil_ref, "prf_")))
      || !ESTADOS.has(a.estado) || a.version !== i + 1 || !ref(a.recibo_ref, "recibo:cronos:") || !instante(a.instante_utc)) throw incompatible();
  }
  if (h.actuaciones.at(-1).estado !== h.estado) throw incompatible();
  for (const a of h.asientos) {
    if (!campos(a, ["tipo", "marcaje_original_ref", "movimiento", "instante_utc", "fuente_version_ref"])
      || !["anulacion", "incorporacion"].includes(a.tipo) || typeof a.marcaje_original_ref !== "string"
      || typeof a.movimiento !== "string" || !(a.movimiento === "" || MOVIMIENTOS.has(a.movimiento))
      || !(a.instante_utc === null || instante(a.instante_utc)) || typeof a.fuente_version_ref !== "string") throw incompatible();
  }
  return h;
}

export function crearClienteCorreccionesCronosHTTP({ fetchImpl = globalThis.fetch, plazoMs = 10_000 } = {}) {
  if (typeof fetchImpl !== "function" || !entero(plazoMs, 1) || plazoMs > 30_000) throw new TypeError("cliente de correcciones no disponible");
  async function llamar(url, { metodo = "GET", cuerpo, signal, aceptados = [200] } = {}) {
    if (signal !== undefined && (!signal || typeof signal.aborted !== "boolean" || typeof signal.addEventListener !== "function")) throw new TypeError("signal no válida");
    if (signal?.aborted) throw fallo("operacion_abortada");
    const controlador = new AbortController(); let agotado = false;
    const abortar = () => controlador.abort(); signal?.addEventListener("abort", abortar, { once: true });
    const temporizador = setTimeout(() => { agotado = true; controlador.abort(); }, plazoMs);
    try {
      let respuesta;
      try {
        respuesta = await fetchImpl(url, { method: metodo, headers: cuerpo ? { Accept: "application/json", "Content-Type": "application/json" } : { Accept: "application/json" },
          ...(cuerpo ? { body: JSON.stringify(cuerpo) } : {}), credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error",
          referrerPolicy: "no-referrer", signal: controlador.signal });
      } catch { throw fallo(signal?.aborted ? "operacion_abortada" : agotado ? "plazo_agotado" : "red_no_disponible"); }
      if (controlador.signal.aborted) throw fallo(signal?.aborted ? "operacion_abortada" : "plazo_agotado");
      if (!respuesta || respuesta.redirected || !Number.isInteger(respuesta.status)) throw incompatible();
      const json = /^application\/json(?:;\s*charset=utf-8)?$/iu.test(respuesta.headers?.get?.("content-type") || "");
      const leer = async () => {
        try { return await leerJSONAcotado(respuesta); }
        catch (error) { throw error instanceof ErrorClienteSaldoCronos ? fallo(error.codigo, error.estado) : incompatible(); }
      };
      if (!aceptados.includes(respuesta.status)) {
        let codigo = ""; if (json) { try { codigo = (await leer())?.error ?? ""; } catch { /* respuesta de error no fiable */ } }
        if (respuesta.status === 401) throw fallo("autenticacion_requerida", 401);
        if (respuesta.status === 403) throw fallo(codigo === "sin_empleado" ? "sin_empleado" : codigo === "no_competente" ? "no_competente" : "acceso_denegado", 403);
        if (respuesta.status === 400) throw fallo("peticion_invalida", 400);
        if (respuesta.status === 409) throw fallo(codigo === "estado_cambiado" ? "estado_cambiado" : "conflicto", 409);
        throw fallo("servicio_no_disponible", respuesta.status);
      }
      if (!json) throw incompatible();
      const valor = await leer();
      if (controlador.signal.aborted) throw fallo(signal?.aborted ? "operacion_abortada" : "plazo_agotado");
      return valor;
    } finally { clearTimeout(temporizador); signal?.removeEventListener("abort", abortar); }
  }
  return Object.freeze({
    async consultarBandeja(paso, { signal } = {}) {
      if (!PASOS_BANDEJA_CORRECCIONES.includes(paso)) throw new TypeError("paso de bandeja no válido");
      const valor = await llamar(`${RUTAS_CORRECCIONES_CRONOS.bandeja}?${new URLSearchParams({ paso })}`, { signal });
      return validarBandejaCorreccionesCronos(valor, paso);
    },
    async consultarHistoria(solicitudRef, { vista = "propia", paso, signal } = {}) {
      if (!ref(solicitudRef, "correccion:cronos:") || !["propia", "equipo"].includes(vista)
        || (vista === "propia" && paso !== undefined) || (vista === "equipo" && !PASOS_BANDEJA_CORRECCIONES.includes(paso))) throw new TypeError("consulta de historia no válida");
      const q = new URLSearchParams({ solicitud_ref: solicitudRef, vista });
      if (vista === "equipo") q.set("paso", paso);
      return validarHistoriaCorreccionCronos(await llamar(`${RUTAS_CORRECCIONES_CRONOS.historia}?${q}`, { signal }), solicitudRef, vista);
    },
    async registrarActuacion(entrada, { signal } = {}) {
      const cuerpo = validarEntradaActuacionCorreccion(entrada);
      return validarReciboActuacionCorreccion(await llamar(RUTAS_CORRECCIONES_CRONOS.actuaciones, { metodo: "POST", cuerpo, signal, aceptados: [200, 201] }), cuerpo);
    },
  });
}
