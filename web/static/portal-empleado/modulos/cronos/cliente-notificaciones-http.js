import { ErrorClienteSaldoCronos, leerJSONAcotado } from "./cliente-saldo-http.js";

/**
 * Cliente de las notificaciones de la persona a RRHH y de la bandeja de RRHH.
 * El servidor deriva la persona, su empleado y su competencia de la sesión
 * mTLS y del circuito publicado; el cliente sólo envía tipo, fecha, texto, la
 * referencia y la huella del documento (que no se sube), la notificación que
 * se atiende y la clave de su operación.
 */
export const RUTAS_NOTIFICACIONES_CRONOS = Object.freeze({
  propio: "/api/interna/cronos/notificaciones/propio",
  envios: "/api/interna/cronos/notificaciones/envios",
  bandeja: "/api/interna/cronos/notificaciones/bandeja",
  atenciones: "/api/interna/cronos/notificaciones/atenciones",
});

export const MAXIMO_TEXTO_NOTIFICACION_CRONOS = 512;
/** Tamaño máximo del documento cuya huella se calcula en el navegador. */
export const MAXIMO_DOCUMENTO_NOTIFICACION_CRONOS = 10 * 1024 * 1024;

export class ErrorClienteNotificacionesCronos extends Error {
  constructor(codigo, estado = 0) {
    super(`cliente de notificaciones Cronos: ${codigo}`);
    this.name = "ErrorClienteNotificacionesCronos";
    this.codigo = codigo;
    this.estado = estado;
  }
}

function fallo(codigo, estado = 0) { return new ErrorClienteNotificacionesCronos(codigo, estado); }
function objeto(v) { return v !== null && typeof v === "object" && !Array.isArray(v); }
function campos(v, requeridos, opcionales = []) {
  return objeto(v) && requeridos.every((c) => Object.hasOwn(v, c)) && Object.keys(v).every((c) => requeridos.includes(c) || opcionales.includes(c));
}
function fecha(v) {
  if (typeof v !== "string" || !/^\d{4}-\d{2}-\d{2}$/u.test(v)) return false;
  const [a, m, d] = v.split("-").map(Number); const f = new Date(Date.UTC(a, m - 1, d));
  return a >= 2000 && a <= 2100 && f.getUTCFullYear() === a && f.getUTCMonth() === m - 1 && f.getUTCDate() === d;
}
function texto(v, max = 160) { return typeof v === "string" && v.length > 0 && [...v].length <= max; }
function instante(v) { return typeof v === "string" && /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?(?:Z|[+-]\d{2}:\d{2})$/u.test(v) && Number.isFinite(Date.parse(v)); }
function clave(v) { return typeof v === "string" && /^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$/u.test(v); }
function incompatible() { return fallo("respuesta_incompatible", 200); }
const refNotificacion = (v) => typeof v === "string" && /^notificacion:cronos:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/u.test(v);
const refAtencion = (v) => typeof v === "string" && /^notificacion:cronos:atencion:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/u.test(v);
const refTipo = (v) => typeof v === "string" && /^notificacion:cronos:tipo:[a-z0-9-]{1,64}$/u.test(v);
const refVersionTipo = (v) => typeof v === "string" && /^notificacion:cronos:tipo:[a-z0-9-]{1,64}:[A-Za-z0-9_.-]{1,64}$/u.test(v);
const refRecibo = (v) => typeof v === "string" && /^recibo:cronos:[0-9a-f-]{36}$/u.test(v);

/** Texto admitido: 1–512 caracteres, no en blanco, sin controles salvo salto de línea y tabulador. */
export function textoNotificacionValido(valor) {
  return typeof valor === "string" && valor.replace(/[ \n\t]/gu, "") !== "" && [...valor].length <= MAXIMO_TEXTO_NOTIFICACION_CRONOS
    && !/[\p{Cc}]/u.test(valor.replace(/[\n\t]/gu, ""));
}

/** Documento por referencia de custodia y huella SHA-256; ambos o ninguno. */
export function adjuntoNotificacionValido(referencia, huella) {
  if (!referencia && !huella) return true;
  return typeof referencia === "string" && /^[A-Za-z][A-Za-z0-9:_-]{2,127}$/u.test(referencia)
    && typeof huella === "string" && /^[0-9a-f]{64}$/u.test(huella) && huella !== "0".repeat(64);
}

/**
 * Huella SHA-256 de un documento local. El contenido no sale del navegador ni
 * se conserva: se lee, se resume y se borra.
 */
export async function calcularHuellaDocumentoCronos(archivo, cripto = globalThis.crypto) {
  if (!archivo || typeof archivo.arrayBuffer !== "function" || !Number.isSafeInteger(archivo.size) || archivo.size < 1
    || archivo.size > MAXIMO_DOCUMENTO_NOTIFICACION_CRONOS || typeof cripto?.subtle?.digest !== "function") throw new TypeError("documento no válido");
  const bytes = new Uint8Array(await archivo.arrayBuffer());
  try {
    if (bytes.byteLength !== archivo.size) throw new TypeError("documento no válido");
    const resumen = new Uint8Array(await cripto.subtle.digest("SHA-256", bytes));
    const huella = Array.from(resumen, (b) => b.toString(16).padStart(2, "0")).join("");
    if (resumen.length !== 32 || huella === "0".repeat(64)) throw new TypeError("documento no válido");
    return huella;
  } finally { bytes.fill(0); }
}

function adjuntoRespuesta(n) {
  return (n.adjunto_ref === undefined) === (n.adjunto_sha256 === undefined) && adjuntoNotificacionValido(n.adjunto_ref ?? "", n.adjunto_sha256 ?? "");
}

export function validarNotificacionesPropiasCronos(v) {
  if (!campos(v, ["tipos", "notificaciones"]) || !Array.isArray(v.tipos) || !Array.isArray(v.notificaciones) || v.tipos.length > 100 || v.notificaciones.length > 500) throw incompatible();
  for (const t of v.tipos) {
    if (!campos(t, ["tipo_version_ref", "tipo_ref", "nombre"]) || !refVersionTipo(t.tipo_version_ref) || !refTipo(t.tipo_ref) || !texto(t.nombre, 120)) throw incompatible();
  }
  for (const n of v.notificaciones) {
    if (!campos(n, ["notificacion_ref", "tipo_ref", "tipo_nombre", "fecha_referida", "texto", "registrada_en", "estado"], ["adjunto_ref", "adjunto_sha256", "atendida_en"])
      || !refNotificacion(n.notificacion_ref) || !refTipo(n.tipo_ref) || !texto(n.tipo_nombre, 120) || !fecha(n.fecha_referida) || !textoNotificacionValido(n.texto)
      || !adjuntoRespuesta(n) || !instante(n.registrada_en) || !["registrada", "atendida"].includes(n.estado)
      || (n.estado === "atendida") !== (n.atendida_en !== undefined) || (n.atendida_en !== undefined && !instante(n.atendida_en))) throw incompatible();
  }
  return v;
}

export function validarBandejaNotificacionesCronos(v) {
  if (!campos(v, ["notificaciones"]) || !Array.isArray(v.notificaciones) || v.notificaciones.length > 500) throw incompatible();
  for (const n of v.notificaciones) {
    if (!campos(n, ["notificacion_ref", "empleado_ref", "empleado_etiqueta", "tipo_ref", "tipo_nombre", "fecha_referida", "texto", "registrada_en", "atendida"],
      ["adjunto_ref", "adjunto_sha256", "atendida_en"])
      || !refNotificacion(n.notificacion_ref) || typeof n.empleado_ref !== "string" || !/^emp_[A-Za-z0-9_-]{22,128}$/u.test(n.empleado_ref)
      || typeof n.empleado_etiqueta !== "string" || [...n.empleado_etiqueta].length > 120 || !refTipo(n.tipo_ref) || !texto(n.tipo_nombre, 120)
      || !fecha(n.fecha_referida) || !textoNotificacionValido(n.texto) || !adjuntoRespuesta(n) || !instante(n.registrada_en)
      || typeof n.atendida !== "boolean" || n.atendida !== (n.atendida_en !== undefined) || (n.atendida_en !== undefined && !instante(n.atendida_en))) throw incompatible();
  }
  return v;
}

export function validarEntradaNotificacionCronos(e) {
  if (!campos(e, ["clave_operacion", "tipo_version_ref", "fecha_referida", "texto"], ["adjunto_ref", "adjunto_sha256"]) || !clave(e.clave_operacion)
    || !refVersionTipo(e.tipo_version_ref) || !fecha(e.fecha_referida) || !textoNotificacionValido(e.texto)
    || !adjuntoNotificacionValido(e.adjunto_ref ?? "", e.adjunto_sha256 ?? "")) throw new TypeError("notificación no válida");
  return Object.freeze({ clave_operacion: e.clave_operacion, tipo_version_ref: e.tipo_version_ref, fecha_referida: e.fecha_referida, texto: e.texto,
    ...(e.adjunto_ref ? { adjunto_ref: e.adjunto_ref, adjunto_sha256: e.adjunto_sha256 } : {}) });
}

function validarReciboNotificacion(v) {
  const r = v?.recibo;
  if (!campos(v, ["recibo"]) || !campos(r, ["notificacion_ref", "recibo_ref", "instante_utc", "replay"]) || !refNotificacion(r.notificacion_ref)
    || !refRecibo(r.recibo_ref) || !instante(r.instante_utc) || typeof r.replay !== "boolean") throw incompatible();
  return r;
}

function validarReciboAtencion(v, entrada) {
  const r = v?.recibo;
  if (!campos(v, ["recibo"]) || !campos(r, ["atencion_ref", "notificacion_ref", "recibo_ref", "instante_utc", "replay"]) || !refAtencion(r.atencion_ref)
    || r.notificacion_ref !== entrada.notificacion_ref || !refRecibo(r.recibo_ref) || !instante(r.instante_utc) || typeof r.replay !== "boolean") throw incompatible();
  return r;
}

const CODIGOS_403 = new Set(["sin_empleado", "no_competente"]);
const CODIGOS_409 = new Set(["estado_cambiado", "tipo_no_vigente", "bandeja_demasiado_grande"]);

export function crearClienteNotificacionesCronosHTTP({ fetchImpl = globalThis.fetch, plazoMs = 10_000 } = {}) {
  if (typeof fetchImpl !== "function" || !Number.isSafeInteger(plazoMs) || plazoMs < 1 || plazoMs > 30_000) throw new TypeError("cliente de notificaciones Cronos no disponible");

  async function llamar(url, { metodo = "GET", cuerpo, signal, aceptados = [200] }) {
    if (signal !== undefined && (!signal || typeof signal.aborted !== "boolean" || typeof signal.addEventListener !== "function")) throw new TypeError("signal no válida");
    if (signal?.aborted) throw fallo("operacion_abortada");
    const controlador = new AbortController(); let agotado = false;
    const abortar = () => controlador.abort(); signal?.addEventListener("abort", abortar, { once: true });
    const temporizador = setTimeout(() => { agotado = true; controlador.abort(); }, plazoMs);
    const cortado = () => fallo(signal?.aborted ? "operacion_abortada" : "plazo_agotado");
    try {
      let respuesta;
      try {
        // Mismo origen: el navegador presenta el certificado mTLS de la sesión;
        // sin cookies propias, sin redirecciones y sin referer.
        respuesta = await fetchImpl(url, {
          method: metodo, headers: cuerpo ? { Accept: "application/json", "Content-Type": "application/json" } : { Accept: "application/json" },
          ...(cuerpo ? { body: JSON.stringify(cuerpo) } : {}), credentials: "same-origin", mode: "same-origin",
          cache: "no-store", redirect: "error", referrerPolicy: "no-referrer", signal: controlador.signal,
        });
      } catch { throw fallo(signal?.aborted ? "operacion_abortada" : agotado ? "plazo_agotado" : "red_no_disponible"); }
      if (controlador.signal.aborted) throw cortado();
      if (!respuesta || respuesta.redirected || !Number.isInteger(respuesta.status)) throw fallo("respuesta_incompatible");
      const json = /^application\/json(?:;\s*charset=utf-8)?$/iu.test(respuesta.headers?.get?.("content-type") || "");
      const leer = async () => {
        try { return await leerJSONAcotado(respuesta); }
        catch (error) {
          if (controlador.signal.aborted) throw cortado();
          throw error instanceof ErrorClienteSaldoCronos ? fallo(error.codigo, error.estado) : fallo("respuesta_incompatible", respuesta.status);
        }
      };
      if (!aceptados.includes(respuesta.status)) {
        let codigoServidor = "";
        if (json) { try { codigoServidor = (await leer())?.error ?? ""; } catch { codigoServidor = ""; } }
        if (respuesta.status === 401) throw fallo("autenticacion_requerida", 401);
        if (respuesta.status === 403) throw fallo(CODIGOS_403.has(codigoServidor) ? codigoServidor : "acceso_denegado", 403);
        if (respuesta.status === 400) throw fallo("peticion_invalida", 400);
        if (respuesta.status === 409) throw fallo(CODIGOS_409.has(codigoServidor) ? codigoServidor : "conflicto", 409);
        throw fallo("servicio_no_disponible", respuesta.status);
      }
      if (!json) throw fallo("tipo_respuesta_no_valido", respuesta.status);
      const valor = await leer();
      if (controlador.signal.aborted) throw cortado();
      return valor;
    } finally { clearTimeout(temporizador); signal?.removeEventListener("abort", abortar); }
  }

  return Object.freeze({
    async consultarPropias({ signal } = {}) {
      return validarNotificacionesPropiasCronos(await llamar(RUTAS_NOTIFICACIONES_CRONOS.propio, { signal }));
    },
    async enviar(entrada, { signal } = {}) {
      const cuerpo = validarEntradaNotificacionCronos(entrada);
      return validarReciboNotificacion(await llamar(RUTAS_NOTIFICACIONES_CRONOS.envios, { metodo: "POST", cuerpo, signal, aceptados: [200, 201] }));
    },
    async consultarBandeja({ signal } = {}) {
      return validarBandejaNotificacionesCronos(await llamar(RUTAS_NOTIFICACIONES_CRONOS.bandeja, { signal }));
    },
    async atender(entrada, { signal } = {}) {
      if (!campos(entrada, ["clave_operacion", "notificacion_ref"]) || !clave(entrada.clave_operacion) || !refNotificacion(entrada.notificacion_ref)) throw new TypeError("atención no válida");
      const cuerpo = Object.freeze({ clave_operacion: entrada.clave_operacion, notificacion_ref: entrada.notificacion_ref });
      return validarReciboAtencion(await llamar(RUTAS_NOTIFICACIONES_CRONOS.atenciones, { metodo: "POST", cuerpo, signal, aceptados: [200, 201] }), cuerpo);
    },
  });
}
