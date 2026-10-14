import { leerJSONAcotado } from "./cliente-saldo-http.js";

export const RUTAS_ORIGINAL_C8 = Object.freeze({
  aportar: "/api/interna/cronos/permisos/justificantes",
  revisar: "/api/interna/cronos/permisos/justificantes/revisiones",
  propio: "/api/interna/cronos/permisos/justificantes/propio",
  detalle: "/api/interna/cronos/permisos/justificantes/detalle",
  bandeja: "/api/interna/cronos/permisos/justificantes/bandeja",
});

export class ErrorClienteOriginalC8 extends Error {
  constructor(codigo, estado = 0) {
    super(`justificación Cronos: ${codigo}`);
    this.codigo = codigo;
    this.estado = estado;
  }
}

const solicitud = (v) => typeof v === "string" && /^permiso:cronos:solicitud:[A-Za-z0-9][A-Za-z0-9_-]{7,127}$/u.test(v);
const referencia = (v) => typeof v === "string" && /^ref:[0-9a-f]{64}$/u.test(v) && v !== `ref:${"0".repeat(64)}`;
const huella = (v) => typeof v === "string" && /^[0-9a-f]{64}$/u.test(v) && v !== "0".repeat(64);
const entero = (v, min = 0) => Number.isSafeInteger(v) && v >= min;
const objeto = (v) => v !== null && typeof v === "object" && !Array.isArray(v);
const clavesExactas = (v, lista) => objeto(v) && Object.keys(v).length === lista.length && lista.every((k) => Object.hasOwn(v, k));
const instante = (v) => typeof v === "string" && /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?(?:Z|[+-]\d{2}:\d{2})$/u.test(v) && Number.isFinite(Date.parse(v));

export function validarAporteOriginalC8(v) {
  if (!clavesExactas(v, ["solicitud_ref", "clave_operacion", "version_esperada", "documento_id", "documento_version"]) ||
    !solicitud(v.solicitud_ref) || !referencia(v.clave_operacion) || !entero(v.version_esperada) ||
    !referencia(v.documento_id) || !entero(v.documento_version, 1)) throw new TypeError("aporte no válido");
  return Object.freeze({ solicitud_ref: v.solicitud_ref, clave_operacion: v.clave_operacion,
    version_esperada: String(v.version_esperada), documento_id: v.documento_id,
    documento_version: String(v.documento_version) });
}

export function validarRevisionOriginalC8(v) {
  if (!clavesExactas(v, ["solicitud_ref", "clave_operacion", "version_esperada", "decision", "motivo_ref"]) ||
    !solicitud(v.solicitud_ref) || !referencia(v.clave_operacion) || !entero(v.version_esperada, 1) ||
    !["aceptada", "rechazada"].includes(v.decision) || typeof v.motivo_ref !== "string" ||
    !/^[-A-Za-z0-9_.:/#]{1,160}$/u.test(v.motivo_ref)) throw new TypeError("revisión no válida");
  return Object.freeze({ solicitud_ref: v.solicitud_ref, clave_operacion: v.clave_operacion,
    version_esperada: String(v.version_esperada), decision: v.decision, motivo_ref: v.motivo_ref });
}

function validarRecibo(v, entrada, estado, codigoHTTP) {
  const r = v?.recibo;
  if (!clavesExactas(v, ["recibo"]) || !clavesExactas(r, ["solicitud_ref", "documento_id", "documento_version",
    "sha256", "estado", "version", "recibo_ref", "instante_utc", "replay"]) ||
    r.solicitud_ref !== entrada.solicitud_ref || !referencia(r.documento_id) ||
    !entero(r.documento_version, 1) || !huella(r.sha256) || r.estado !== estado ||
    r.version !== Number(entrada.version_esperada) + 1 ||
    typeof r.recibo_ref !== "string" || !/^recibo:cronos:[0-9a-f-]{36}$/u.test(r.recibo_ref) ||
    !instante(r.instante_utc) || typeof r.replay !== "boolean" || r.replay !== (codigoHTTP === 200) ||
    (entrada.documento_id && (r.documento_id !== entrada.documento_id || r.documento_version !== Number(entrada.documento_version)))) {
    throw new ErrorClienteOriginalC8("respuesta_incompatible", codigoHTTP);
  }
  return Object.freeze(r);
}

function validarDetalle(v, solicitudRef) {
  if (!clavesExactas(v, ["solicitud_ref", "estado_permiso", "version_permiso", "pendiente_justificar", "historia"]) ||
    v.solicitud_ref !== solicitudRef || v.estado_permiso !== "concedido" || !entero(v.version_permiso, 1) ||
    v.pendiente_justificar !== true || !Array.isArray(v.historia) || v.historia.length > 500) {
    throw new ErrorClienteOriginalC8("respuesta_incompatible", 200);
  }
  for (const [i, h] of v.historia.entries()) {
    if (!clavesExactas(h, ["version", "estado", "motivo_ref", "documento_id", "documento_version",
      "sha256", "recibo_documento_ref", "custodio_id", "recibo_ref", "instante_utc"]) ||
      h.version !== i + 1 || !["pendiente_revision", "aceptada", "rechazada"].includes(h.estado) ||
      typeof h.motivo_ref !== "string" || !referencia(h.documento_id) || !entero(h.documento_version, 1) ||
      !huella(h.sha256) || typeof h.recibo_documento_ref !== "string" || !h.recibo_documento_ref ||
      typeof h.custodio_id !== "string" || !h.custodio_id ||
      typeof h.recibo_ref !== "string" || !/^recibo:cronos:[0-9a-f-]{36}$/u.test(h.recibo_ref) || !instante(h.instante_utc)) {
      throw new ErrorClienteOriginalC8("respuesta_incompatible", 200);
    }
  }
  return v;
}

function validarBandeja(v) {
  if (!clavesExactas(v, ["pendientes"]) || !Array.isArray(v.pendientes) || v.pendientes.length > 500) {
    throw new ErrorClienteOriginalC8("respuesta_incompatible", 200);
  }
  for (const p of v.pendientes) {
    if (!clavesExactas(p, ["solicitud_ref", "empleado_etiqueta", "documento_id", "documento_version",
      "sha256", "estado", "version", "registrada_en"]) || !solicitud(p.solicitud_ref) ||
      typeof p.empleado_etiqueta !== "string" || p.empleado_etiqueta.length > 120 ||
      !referencia(p.documento_id) || !entero(p.documento_version, 1) || !huella(p.sha256) ||
      p.estado !== "pendiente_revision" || !entero(p.version, 1) || !instante(p.registrada_en)) {
      throw new ErrorClienteOriginalC8("respuesta_incompatible", 200);
    }
  }
  return v;
}

export function crearClienteOriginalC8HTTP({ fetchImpl = globalThis.fetch, plazoMs = 10_000 } = {}) {
  if (typeof fetchImpl !== "function" || !entero(plazoMs, 1) || plazoMs > 30_000) throw new TypeError("cliente no disponible");

  async function enviar(ruta, entrada, estado, signal) {
    if (signal !== undefined && (!signal || typeof signal.aborted !== "boolean" || typeof signal.addEventListener !== "function")) throw new TypeError("signal no válida");
    if (signal?.aborted) throw new ErrorClienteOriginalC8("operacion_abortada");
    const controlador = new AbortController(); let agotado = false;
    const abortar = () => controlador.abort(); signal?.addEventListener("abort", abortar, { once: true });
    const timer = setTimeout(() => { agotado = true; controlador.abort(); }, plazoMs);
    try {
      let respuesta;
      try {
        respuesta = await fetchImpl(ruta, { method: "POST", headers: { Accept: "application/json", "Content-Type": "application/json" },
          body: JSON.stringify(entrada), credentials: "same-origin", mode: "same-origin", cache: "no-store",
          redirect: "error", referrerPolicy: "no-referrer", signal: controlador.signal });
      } catch { throw new ErrorClienteOriginalC8(signal?.aborted ? "operacion_abortada" : agotado ? "plazo_agotado" : "red_no_disponible"); }
      if (controlador.signal.aborted || !respuesta || respuesta.redirected || !Number.isInteger(respuesta.status)) {
        throw new ErrorClienteOriginalC8("respuesta_incompatible");
      }
      if (!/^application\/json(?:;\s*charset=utf-8)?$/iu.test(respuesta.headers?.get?.("content-type") || "")) {
        throw new ErrorClienteOriginalC8("respuesta_incompatible", respuesta.status);
      }
      let contenido;
      try { contenido = await leerJSONAcotado(respuesta); }
      catch { throw new ErrorClienteOriginalC8("respuesta_incompatible", respuesta.status); }
      if (respuesta.status === 200 || respuesta.status === 201) return validarRecibo(contenido, entrada, estado, respuesta.status);
      throw new ErrorClienteOriginalC8(({ 400: "peticion_invalida", 401: "autenticacion_requerida", 403: "acceso_denegado",
        409: "estado_cambiado", 503: "servicio_no_disponible" })[respuesta.status] ?? "servicio_no_disponible", respuesta.status);
    } finally { clearTimeout(timer); signal?.removeEventListener("abort", abortar); }
  }

  async function consultar(ruta, signal) {
    if (signal !== undefined && (!signal || typeof signal.aborted !== "boolean" || typeof signal.addEventListener !== "function")) throw new TypeError("signal no válida");
    if (signal?.aborted) throw new ErrorClienteOriginalC8("operacion_abortada");
    const controlador = new AbortController(); let agotado = false;
    const abortar = () => controlador.abort(); signal?.addEventListener("abort", abortar, { once: true });
    const timer = setTimeout(() => { agotado = true; controlador.abort(); }, plazoMs);
    try {
      let respuesta;
      try { respuesta = await fetchImpl(ruta, { method: "GET", headers: { Accept: "application/json" },
        credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error",
        referrerPolicy: "no-referrer", signal: controlador.signal }); }
      catch { throw new ErrorClienteOriginalC8(signal?.aborted ? "operacion_abortada" : agotado ? "plazo_agotado" : "red_no_disponible"); }
      if (controlador.signal.aborted || !respuesta || respuesta.redirected || !Number.isInteger(respuesta.status)) {
        throw new ErrorClienteOriginalC8("respuesta_incompatible");
      }
      if (!/^application\/json(?:;\s*charset=utf-8)?$/iu.test(respuesta.headers?.get?.("content-type") || "")) {
        throw new ErrorClienteOriginalC8("respuesta_incompatible", respuesta.status);
      }
      let contenido;
      try { contenido = await leerJSONAcotado(respuesta); }
      catch { throw new ErrorClienteOriginalC8("respuesta_incompatible", respuesta.status); }
      if (respuesta.status === 200) return contenido;
      const codigo = respuesta.status === 409 && ["bandeja_demasiado_grande", "estado_cambiado"].includes(contenido?.error)
        ? contenido.error : ({ 400: "peticion_invalida", 401: "autenticacion_requerida",
          403: "acceso_denegado", 409: "conflicto", 503: "servicio_no_disponible" })[respuesta.status] ?? "servicio_no_disponible";
      throw new ErrorClienteOriginalC8(codigo, respuesta.status);
    } finally { clearTimeout(timer); signal?.removeEventListener("abort", abortar); }
  }

  return Object.freeze({
    aportar: (v, opciones = {}) => enviar(RUTAS_ORIGINAL_C8.aportar, validarAporteOriginalC8(v), "pendiente_revision", opciones.signal),
    revisar: (v, opciones = {}) => enviar(RUTAS_ORIGINAL_C8.revisar, validarRevisionOriginalC8(v), v.decision, opciones.signal),
    consultarPropia: async (ref, opciones = {}) => {
      if (!solicitud(ref)) throw new TypeError("solicitud no válida");
      return validarDetalle(await consultar(`${RUTAS_ORIGINAL_C8.propio}?solicitud_ref=${encodeURIComponent(ref)}`, opciones.signal), ref);
    },
    consultarDetalleRRHH: async (ref, opciones = {}) => {
      if (!solicitud(ref)) throw new TypeError("solicitud no válida");
      return validarDetalle(await consultar(`${RUTAS_ORIGINAL_C8.detalle}?solicitud_ref=${encodeURIComponent(ref)}`, opciones.signal), ref);
    },
    consultarBandeja: async (opciones = {}) => validarBandeja(await consultar(RUTAS_ORIGINAL_C8.bandeja, opciones.signal)),
  });
}
