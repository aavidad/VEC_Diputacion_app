/** Cliente HTTP del cese, el cierre y la modificación tras el nombramiento. */

export const RUTA_CESES_NOMBRAMIENTO = "/api/vec/contratacion-temporal/ceses";
export const RUTA_CIERRES_EXPEDIENTE = "/api/vec/contratacion-temporal/cierres-expediente";
export const RUTA_MODIFICACIONES_NOMBRAMIENTO = "/api/vec/contratacion-temporal/modificaciones-nombramiento";
export const RUTA_SEGUIMIENTO_CESE = "/api/vec/contratacion-temporal/seguimiento-cese";
export const RUTAS_SEGUIMIENTO_CESE = Object.freeze([
  RUTA_CESES_NOMBRAMIENTO, RUTA_CIERRES_EXPEDIENTE, RUTA_MODIFICACIONES_NOMBRAMIENTO, RUTA_SEGUIMIENTO_CESE,
]);
// Rechazos de negocio que devuelve el servidor con 409; ninguno escribe nada.
export const CONFLICTOS_SEGUIMIENTO_CESE = Object.freeze([
  "version_en_conflicto", "clave_reutilizada", "sin_incorporacion", "fecha_anterior_incorporacion",
  "cese_existente", "sin_cese", "cierre_existente", "sin_cambios", "credito_insuficiente",
]);

const MAXIMO = 16 * 1024;
const REF = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const CLAVE = /^[a-z][a-z0-9._-]{1,79}$/u;
const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/u;
const FECHA = /^\d{4}-\d{2}-\d{2}$/u;
const HUELLA = /^[0-9a-f]{64}$/u;
const GINPIX = /^[A-Za-z0-9][A-Za-z0-9._/-]{0,63}$/u;
const INSTANTE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$/u;

function registro(valor, campos, opcionales = []) {
  if (!valor || typeof valor !== "object" || Array.isArray(valor) || Object.getPrototypeOf(valor) !== Object.prototype
    || !campos.every((c) => Object.hasOwn(valor, c)) || Object.keys(valor).some((c) => !campos.includes(c) && !opcionales.includes(c))) {
    throw new TypeError("contrato de cese, cierre o modificación no válido");
  }
  return valor;
}

export function fechaCivilValida(valor) {
  if (typeof valor !== "string" || !FECHA.test(valor)) return false;
  const fecha = new Date(`${valor}T00:00:00Z`);
  return Number.isFinite(fecha.getTime()) && fecha.toISOString().slice(0, 10) === valor;
}

function textoValido(valor, vacio = true) {
  return typeof valor === "string" && valor === valor.trim() && valor.length <= 2000 && (vacio || valor.length > 0)
    && !/[\u0000-\u0008\u000b-\u001f\u007f]/u.test(valor);
}

function comun(s) {
  if (!REF.test(s.expediente_ref) || !Number.isSafeInteger(s.version_esperada) || s.version_esperada < 1
    || !UUID.test(s.clave_idempotencia)) throw new TypeError("solicitud de seguimiento no válida");
}

export function validarSolicitudCese(valor) {
  const campos = ["expediente_ref", "version_esperada", "clave_idempotencia", "causa_clave", "fecha_efecto", "justificante_ref", "justificante_sha256", "observaciones"];
  const s = registro(valor, campos);
  comun(s);
  if (!CLAVE.test(s.causa_clave) || !fechaCivilValida(s.fecha_efecto) || !REF.test(s.justificante_ref)
    || !HUELLA.test(s.justificante_sha256) || !textoValido(s.observaciones)) throw new TypeError("solicitud de cese no válida");
  return Object.freeze(Object.fromEntries(campos.map((c) => [c, s[c]])));
}

export function validarSolicitudCierre(valor) {
  const campos = ["expediente_ref", "version_esperada", "clave_idempotencia", "ginpix_numero", "ginpix_confirmada_en", "observaciones"];
  const s = registro(valor, campos);
  comun(s);
  const conGINPIX = s.ginpix_numero !== "";
  if ((conGINPIX && !GINPIX.test(s.ginpix_numero)) || conGINPIX !== (s.ginpix_confirmada_en !== "")
    || (conGINPIX && !fechaCivilValida(s.ginpix_confirmada_en)) || !textoValido(s.observaciones)) throw new TypeError("solicitud de cierre no válida");
  return Object.freeze(Object.fromEntries(campos.map((c) => [c, s[c]])));
}

export function validarSolicitudModificacion(valor) {
  const campos = ["expediente_ref", "version_esperada", "clave_idempotencia", "motivo_clave", "periodo_inicio", "periodo_fin", "porcentaje_jornada", "observaciones"];
  const s = registro(valor, campos);
  comun(s);
  if (!CLAVE.test(s.motivo_clave) || !fechaCivilValida(s.periodo_inicio) || !fechaCivilValida(s.periodo_fin)
    || s.periodo_fin < s.periodo_inicio || !Number.isSafeInteger(s.porcentaje_jornada) || s.porcentaje_jornada < 1
    || s.porcentaje_jornada > 10000 || !textoValido(s.observaciones, false)) throw new TypeError("solicitud de modificación no válida");
  return Object.freeze(Object.fromEntries(campos.map((c) => [c, s[c]])));
}

export function validarReciboSeguimiento(valor, solicitud, operacion) {
  const r = registro(valor, ["esquema", "operacion", "expediente_ref", "version_anterior", "version_resultante", "fase_resultante",
    "estado_resultante", "recibo_ref", "auditoria_ref", "evento_ref", "registrada_en"], ["causa_clave", "fecha_efecto", "cese_recibo_ref", "coste_centimos"]);
  if (r.esquema !== "vec.contratacion-temporal.recibo-seguimiento.v1" || r.operacion !== operacion
    || r.expediente_ref !== solicitud.expediente_ref || r.version_anterior !== solicitud.version_esperada
    || r.version_resultante !== solicitud.version_esperada + 1 || !CLAVE.test(r.fase_resultante) || !CLAVE.test(r.estado_resultante)
    || ![r.recibo_ref, r.auditoria_ref, r.evento_ref].every((ref) => typeof ref === "string" && REF.test(ref))
    || typeof r.registrada_en !== "string" || !INSTANTE.test(r.registrada_en) || !Number.isFinite(Date.parse(r.registrada_en))
    || (r.coste_centimos !== undefined && (!Number.isSafeInteger(r.coste_centimos) || r.coste_centimos < 1))) throw new TypeError("recibo de seguimiento no válido");
  return Object.freeze({ ...r });
}

function opcionValida(o, campos) {
  return registro(o, campos) && campos.every((c) => typeof o[c] === "string") && CLAVE.test(o.clave) && o.etiqueta.length > 0;
}

export function validarConsultaSeguimientoCese(valor, expedienteRef) {
  const d = registro(valor, ["esquema", "opciones", "estado"]);
  const o = registro(d.opciones, ["causas_cese", "condiciones_cierre", "fase_retorno_modificacion", "motivos_modificacion"]);
  const e = registro(d.estado, ["expediente_ref", "incorporacion", "cese", "cierre"]);
  if (d.esquema !== "vec.contratacion-temporal.seguimiento-cese.v1" || e.expediente_ref !== expedienteRef
    || !Array.isArray(o.causas_cese) || o.causas_cese.length > 64
    || !o.causas_cese.every((c) => opcionValida(c, ["clave", "etiqueta", "clave_i18n", "justificante_tipo"]) && CLAVE.test(c.justificante_tipo))
    || !Array.isArray(o.motivos_modificacion) || o.motivos_modificacion.length > 64
    || !o.motivos_modificacion.every((m) => opcionValida(m, ["clave", "etiqueta", "clave_i18n"]))
    || !Array.isArray(o.condiciones_cierre) || !o.condiciones_cierre.includes("cese_registrado")
    || !o.condiciones_cierre.every((c) => ["cese_registrado", "ginpix_confirmado"].includes(c))
    || !CLAVE.test(o.fase_retorno_modificacion)) throw new TypeError("consulta de seguimiento no válida");
  if (e.incorporacion !== null && !fechaCivilValida(registro(e.incorporacion, ["inicio"]).inicio)) throw new TypeError("incorporación no válida");
  if (e.cese !== null) {
    const c = registro(e.cese, ["causa_clave", "fecha_efecto", "justificante_tipo", "justificante_ref", "justificante_sha256", "observaciones", "recibo_ref", "registrada_en"]);
    if (!CLAVE.test(c.causa_clave) || !fechaCivilValida(c.fecha_efecto) || !REF.test(c.recibo_ref)) throw new TypeError("cese no válido");
  }
  if (e.cierre !== null) {
    const c = registro(e.cierre, ["condiciones", "ginpix_numero", "ginpix_confirmada_en", "observaciones", "recibo_ref", "registrada_en"]);
    if (!Array.isArray(c.condiciones) || !REF.test(c.recibo_ref) || e.cese === null) throw new TypeError("cierre no válido");
  }
  return Object.freeze({ opciones: o, estado: e });
}

export function crearClienteSeguimientoCeseHTTP({ ejecutar, validarOpciones, serializarAcotado } = {}) {
  if (typeof ejecutar !== "function" || typeof validarOpciones !== "function" || typeof serializarAcotado !== "function") {
    throw new TypeError("dependencias HTTP de cese, cierre y modificación no disponibles");
  }
  const rechazo = (error) => error?.envelopeValido === true && (["400:peticion_no_permitida", "403:acceso_denegado", "422:contenido_no_valido"]
    .includes(`${error.estado}:${error.codigo}`) || (error.estado === 409 && CONFLICTOS_SEGUIMIENTO_CESE.includes(error.codigo)));
  function efecto(ruta, validar, operacion) {
    return (solicitud, opciones) => {
      const entrada = validar(JSON.parse(serializarAcotado(solicitud, MAXIMO)));
      const { signal } = validarOpciones(opciones);
      return ejecutar({ ruta, entrada, signal, estadoEsperado: 201, maximoSolicitud: MAXIMO, maximoRespuesta: MAXIMO, efecto: true,
        validarRespuesta: (respuesta) => validarReciboSeguimiento(respuesta, entrada, operacion), rechazoDeterminado: rechazo });
    };
  }
  return Object.freeze({
    consultarSeguimientoCese(expedienteRef, opciones) {
      if (typeof expedienteRef !== "string" || !REF.test(expedienteRef)) throw new TypeError("expediente no válido");
      const { signal } = validarOpciones(opciones);
      return ejecutar({ ruta: RUTA_SEGUIMIENTO_CESE, entrada: { expediente_ref: expedienteRef }, signal, estadoEsperado: 200,
        maximoSolicitud: 1024, maximoRespuesta: 64 * 1024, efecto: false,
        validarRespuesta: (respuesta) => validarConsultaSeguimientoCese(respuesta, expedienteRef) });
    },
    registrarCese: efecto(RUTA_CESES_NOMBRAMIENTO, validarSolicitudCese, "registrar_cese"),
    cerrarExpediente: efecto(RUTA_CIERRES_EXPEDIENTE, validarSolicitudCierre, "cerrar_expediente"),
    modificarTrasNombramiento: efecto(RUTA_MODIFICACIONES_NOMBRAMIENTO, validarSolicitudModificacion, "modificar_tras_nombramiento"),
  });
}
