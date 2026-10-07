import { validarPermisosPropiosCronos } from "../cronos/cliente-solicitudes-http.js";
import { sinBordes } from "../dietas/texto-dietas.js?v=20260925-d5d6-v1";

const ESTADOS_DIETAS = new Set(["borrador", "eliminado", "enviado_pendiente_revision", "pendiente_autorizacion", "pendiente_liquidacion", "pendiente_fiscalizacion", "fiscalizada", "devuelta"]);
const ETAPAS_DEVOLUCION = new Set(["revision", "autorizacion", "liquidacion", "fiscalizacion"]);
const CODIGOS_LECTURA = new Set(["acceso_denegado", "autenticacion_requerida", "sin_empleado", "relacion_ambigua", "relacion_no_disponible", "no_disponible", "servicio_no_disponible", "red_no_disponible", "plazo_agotado", "operacion_abortada", "respuesta_incompatible", "respuesta_excesiva", "peticion_invalida", "fuente_no_configurada"]);
const codificador = new TextEncoder();

function fallo(codigo) {
  const error = new Error();
  error.name = "ErrorFuenteTramitesPropios";
  error.codigo = codigo;
  return Object.freeze(error);
}
function registro(valor) {
  return valor !== null && typeof valor === "object" && !Array.isArray(valor)
    && (Object.getPrototypeOf(valor) === Object.prototype || Object.getPrototypeOf(valor) === null);
}
function entrada(valor, claves) {
  if (!registro(valor) || Object.keys(valor).some((clave) => !claves.includes(clave))) throw fallo("peticion_invalida");
}
function comprobarSignal(signal) {
  if (signal !== undefined && (!signal || typeof signal.aborted !== "boolean"
    || typeof signal.addEventListener !== "function" || typeof signal.removeEventListener !== "function")) throw fallo("peticion_invalida");
  if (signal?.aborted) throw fallo("operacion_abortada");
}
function fecha(valor) {
  if (typeof valor !== "string" || !/^\d{4}-\d{2}-\d{2}$/u.test(valor)) return false;
  const [anio, mes, dia] = valor.split("-").map(Number);
  const civil = new Date(Date.UTC(anio, mes - 1, dia));
  return civil.getUTCFullYear() === anio && civil.getUTCMonth() === mes - 1 && civil.getUTCDate() === dia;
}
function instante(valor) {
  return typeof valor === "string" && /^\d{4}-\d{2}-\d{2}T(?:[01]\d|2[0-3]):[0-5]\d:[0-5]\d(?:\.\d{1,9})?(?:Z|[+-]\d{2}:\d{2})$/u.test(valor)
    && fecha(valor.slice(0, 10)) && Number.isFinite(Date.parse(valor));
}
function referencia(valor, prefijo) { return typeof valor === "string" && new RegExp(`^${prefijo}[A-Za-z0-9_-]{22,128}$`, "u").test(valor); }
function version(valor) { return Number.isSafeInteger(valor) && valor >= 1; }
function cursorValido(valor) {
  return typeof valor === "string" && valor.length > 0 && codificador.encode(valor).byteLength <= 400 && !/[\x00-\x1F\x7F]/u.test(valor);
}
function incompatible() { throw fallo("respuesta_incompatible"); }

function proyectarCronos(valor, anio) {
  validarPermisosPropiosCronos(valor, anio);
  const nombres = new Map(valor.permisos.map((permiso) => [permiso.permiso_ref, permiso.nombre]));
  const solicitudes = valor.solicitudes.map((solicitud) => {
    if (!instante(solicitud.solicitada_en)) incompatible();
    const nombre = nombres.get(solicitud.permiso_ref);
    if (nombre !== undefined && /[\x00-\x1F\x7F]/u.test(nombre)) incompatible();
    return Object.freeze({
      solicitud_ref: solicitud.solicitud_ref, estado: solicitud.estado, version: solicitud.version,
      solicitada_en: solicitud.solicitada_en, desde: solicitud.desde, hasta: solicitud.hasta,
      pendiente_justificar: solicitud.pendiente_justificar,
      ...(nombre === undefined ? {} : { nombre }),
      ...(solicitud.circuito === undefined ? {} : { circuito: solicitud.circuito }),
      ...(solicitud.pendiente_asignacion === undefined ? {} : { pendiente_asignacion: solicitud.pendiente_asignacion }),
    });
  });
  return Object.freeze({ anio: valor.anio, solicitudes: Object.freeze(solicitudes) });
}

// Hecho devuelto por Dietas: su instante no fija un plazo de subsanación.
function proyectarDevolucion(comision) {
  const d = comision.devolucion;
  if (!registro(d) || Object.keys(d).length !== 4 || !["etapa", "motivo", "version", "devuelta_en"].every((clave) => Object.hasOwn(d, clave))
    || !ETAPAS_DEVOLUCION.has(d.etapa) || !["devuelta", "borrador"].includes(comision.estado)
    || typeof d.motivo !== "string" || d.motivo.length < 3 || codificador.encode(d.motivo).byteLength > 600
    || /[\x00-\x1F\x7F]/u.test(d.motivo) || !sinBordes(d.motivo)
    || !version(d.version) || d.version < 3 || !version(comision.version) || d.version > comision.version
    || typeof d.devuelta_en !== "string" || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$/u.test(d.devuelta_en) || !instante(d.devuelta_en)) incompatible();
  return Object.freeze({ etapa: d.etapa, motivo: d.motivo, version: d.version, devuelta_en: d.devuelta_en });
}

function proyectarItemDietas(item) {
  if (!registro(item) || !registro(item.comision)) incompatible();
  const comision = item.comision;
  if (!referencia(comision.referencia, "dco_") || (comision.version !== undefined && !version(comision.version)) || !ESTADOS_DIETAS.has(comision.estado)
    || !fecha(comision.fecha_inicio) || !fecha(comision.fecha_fin) || comision.fecha_inicio > comision.fecha_fin
    || (comision.numero_documento !== undefined && (typeof comision.numero_documento !== "string" || !/^VEC-D-\d{4}-\d{6,18}$/u.test(comision.numero_documento)))
    || (comision.fecha_apertura !== undefined && !instante(comision.fecha_apertura))) incompatible();
  const devolucion = comision.devolucion === undefined ? undefined : proyectarDevolucion(comision);
  const minima = Object.freeze({
    referencia: comision.referencia, ...(comision.version === undefined ? {} : { version: comision.version }), estado: comision.estado,
    fecha_inicio: comision.fecha_inicio, fecha_fin: comision.fecha_fin,
    ...(comision.numero_documento === undefined ? {} : { numero_documento: comision.numero_documento }),
    ...(comision.fecha_apertura === undefined ? {} : { fecha_apertura: comision.fecha_apertura }),
    ...(devolucion === undefined ? {} : { devolucion }),
  });
  let recibo;
  if (item.recibo !== undefined) {
    const real = item.recibo;
    if (!registro(real) || !referencia(real.referencia, "rcd_") || !version(real.version)
      || !instante(real.registrado_en) || typeof real.repeticion !== "boolean") incompatible();
    recibo = Object.freeze({ referencia: real.referencia, version: real.version, registrado_en: real.registrado_en, repeticion: real.repeticion });
  }
  return Object.freeze({ comision: minima, ...(recibo === undefined ? {} : { recibo }) });
}
function proyectarDietas(valor) {
  if (!registro(valor) || !Array.isArray(valor.items) || valor.items.length > 20
    || (valor.siguiente_cursor !== undefined && !cursorValido(valor.siguiente_cursor))) incompatible();
  return Object.freeze({ items: Object.freeze(Array.from(valor.items, proyectarItemDietas)),
    ...(valor.siguiente_cursor === undefined ? {} : { siguiente_cursor: valor.siguiente_cursor }) });
}

/** Lecturas separadas por origen; la autorización pertenece a cada cliente. */
export function crearFuenteTramitesPropios({ consultarPermisos, listarComisiones } = {}) {
  for (const consultar of [consultarPermisos, listarComisiones]) {
    if (consultar !== undefined && typeof consultar !== "function") throw fallo("fuente_no_configurada");
  }
  async function consultar(funcion, consulta, signal, proyectar) {
    comprobarSignal(signal);
    if (!funcion) throw fallo("fuente_no_configurada");
    try {
      const valor = await funcion(Object.freeze(consulta), Object.freeze({ signal }));
      comprobarSignal(signal);
      const resultado = proyectar(valor);
      comprobarSignal(signal);
      return resultado;
    } catch (error) {
      if (signal?.aborted) throw fallo("operacion_abortada");
      throw fallo(CODIGOS_LECTURA.has(error?.codigo) ? error.codigo : "servicio_no_disponible");
    }
  }
  return Object.freeze({
    disponibles: Object.freeze({ cronos: typeof consultarPermisos === "function", dietas: typeof listarComisiones === "function" }),
    async consultarCronos(opciones = {}) {
      entrada(opciones, ["anio", "signal"]);
      const { anio, signal } = opciones;
      if (!Number.isSafeInteger(anio) || anio < 2000 || anio > 2100) throw fallo("peticion_invalida");
      return consultar(consultarPermisos, { anio }, signal, (valor) => proyectarCronos(valor, anio));
    },
    async consultarDietas(opciones = {}) {
      entrada(opciones, ["cursor", "signal"]);
      const { cursor, signal } = opciones;
      if (cursor !== undefined && !cursorValido(cursor)) throw fallo("peticion_invalida");
      return consultar(listarComisiones, { limit: 20, ...(cursor === undefined ? {} : { cursor }) }, signal, proyectarDietas);
    },
  });
}
