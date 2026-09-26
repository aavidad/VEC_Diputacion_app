/** Cliente HTTP de la cancelación del expediente antes de la fiscalización. */

export const RUTA_CANCELACIONES_EXPEDIENTE = "/api/vec/contratacion-temporal/cancelaciones-expediente";
export const RUTA_CANCELACION_EXPEDIENTE = "/api/vec/contratacion-temporal/cancelacion-expediente";
export const RUTAS_CANCELACION_EXPEDIENTE = Object.freeze([RUTA_CANCELACIONES_EXPEDIENTE, RUTA_CANCELACION_EXPEDIENTE]);
// Rechazos de negocio que devuelve el servidor con 409; ninguno escribe nada.
export const CONFLICTOS_CANCELACION_EXPEDIENTE = Object.freeze([
  "version_en_conflicto", "clave_reutilizada", "fase_no_admitida", "tras_fiscalizacion", "cancelacion_existente",
]);

const MAXIMO = 8 * 1024;
const REF = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const CLAVE = /^[a-z][a-z0-9._-]{1,79}$/u;
const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/u;
const INSTANTE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$/u;

function registro(valor, campos) {
  if (!valor || typeof valor !== "object" || Array.isArray(valor) || Object.getPrototypeOf(valor) !== Object.prototype
    || !campos.every((c) => Object.hasOwn(valor, c)) || Object.keys(valor).some((c) => !campos.includes(c))) {
    throw new TypeError("contrato de cancelación no válido");
  }
  return valor;
}

function textoValido(valor) {
  return typeof valor === "string" && valor === valor.trim() && valor.length <= 2000
    && !/[\u0000-\u0008\u000b-\u001f\u007f]/u.test(valor);
}

const instanteValido = (v) => typeof v === "string" && INSTANTE.test(v) && Number.isFinite(Date.parse(v));

export function validarSolicitudCancelacion(valor) {
  const campos = ["expediente_ref", "version_esperada", "clave_idempotencia", "motivo_clave", "observaciones"];
  const s = registro(valor, campos);
  if (!REF.test(s.expediente_ref) || !Number.isSafeInteger(s.version_esperada) || s.version_esperada < 1
    || !UUID.test(s.clave_idempotencia) || !CLAVE.test(s.motivo_clave) || !textoValido(s.observaciones)) {
    throw new TypeError("solicitud de cancelación no válida");
  }
  return Object.freeze(Object.fromEntries(campos.map((c) => [c, s[c]])));
}

export function validarReciboCancelacion(valor, solicitud) {
  const r = registro(valor, ["esquema", "operacion", "expediente_ref", "version_anterior", "version_resultante", "fase_resultante",
    "estado_resultante", "motivo_clave", "recibo_ref", "auditoria_ref", "evento_ref", "registrada_en"]);
  if (r.esquema !== "vec.contratacion-temporal.recibo-cancelacion.v1" || r.operacion !== "cancelar_expediente"
    || r.expediente_ref !== solicitud.expediente_ref || r.version_anterior !== solicitud.version_esperada
    || r.version_resultante !== solicitud.version_esperada + 1 || !CLAVE.test(r.fase_resultante) || r.estado_resultante !== "cancelado"
    || r.motivo_clave !== solicitud.motivo_clave
    || ![r.recibo_ref, r.auditoria_ref, r.evento_ref].every((ref) => typeof ref === "string" && REF.test(ref))
    || !instanteValido(r.registrada_en)) throw new TypeError("recibo de cancelación no válido");
  return Object.freeze({ ...r });
}

export function validarConsultaCancelacion(valor, expedienteRef) {
  const d = registro(valor, ["esquema", "expediente_ref", "fases_admitidas", "motivos", "cancelacion"]);
  if (d.esquema !== "vec.contratacion-temporal.cancelacion-expediente.v1" || d.expediente_ref !== expedienteRef
    || !Array.isArray(d.fases_admitidas) || d.fases_admitidas.length < 1 || d.fases_admitidas.length > 16
    || !d.fases_admitidas.every((f) => typeof f === "string" && CLAVE.test(f))
    || !Array.isArray(d.motivos) || d.motivos.length < 1 || d.motivos.length > 64
    || !d.motivos.every((m) => registro(m, ["clave", "etiqueta", "clave_i18n"]) && CLAVE.test(m.clave)
      && typeof m.etiqueta === "string" && m.etiqueta.length > 0 && typeof m.clave_i18n === "string")) {
    throw new TypeError("consulta de cancelación no válida");
  }
  if (d.cancelacion !== null) {
    const c = registro(d.cancelacion, ["canal", "motivo_clave", "motivo_etiqueta", "motivo_clave_i18n", "fase_previa", "observaciones", "recibo_ref", "registrada_en"]);
    if (!["rrhh", "centro"].includes(c.canal) || !CLAVE.test(c.motivo_clave) || !CLAVE.test(c.fase_previa)
      || typeof c.motivo_etiqueta !== "string" || c.motivo_etiqueta.length === 0 || typeof c.motivo_clave_i18n !== "string"
      || !textoValido(c.observaciones) || !REF.test(c.recibo_ref) || !instanteValido(c.registrada_en)) {
      throw new TypeError("cancelación registrada no válida");
    }
  }
  return Object.freeze({ fases_admitidas: Object.freeze([...d.fases_admitidas]), motivos: Object.freeze(d.motivos.map((m) => Object.freeze({ ...m }))),
    cancelacion: d.cancelacion === null ? null : Object.freeze({ ...d.cancelacion }) });
}

export function crearClienteCancelacionHTTP({ ejecutar, validarOpciones, serializarAcotado } = {}) {
  if (typeof ejecutar !== "function" || typeof validarOpciones !== "function" || typeof serializarAcotado !== "function") {
    throw new TypeError("dependencias HTTP de la cancelación no disponibles");
  }
  const rechazo = (error) => error?.envelopeValido === true && (["400:peticion_no_permitida", "403:acceso_denegado", "422:contenido_no_valido"]
    .includes(`${error.estado}:${error.codigo}`) || (error.estado === 409 && CONFLICTOS_CANCELACION_EXPEDIENTE.includes(error.codigo)));
  return Object.freeze({
    consultarCancelacion(expedienteRef, opciones) {
      if (typeof expedienteRef !== "string" || !REF.test(expedienteRef)) throw new TypeError("expediente no válido");
      const { signal } = validarOpciones(opciones);
      return ejecutar({ ruta: RUTA_CANCELACION_EXPEDIENTE, entrada: { expediente_ref: expedienteRef }, signal, estadoEsperado: 200,
        maximoSolicitud: 1024, maximoRespuesta: 32 * 1024, efecto: false,
        validarRespuesta: (respuesta) => validarConsultaCancelacion(respuesta, expedienteRef) });
    },
    cancelarExpediente(solicitud, opciones) {
      const entrada = validarSolicitudCancelacion(JSON.parse(serializarAcotado(solicitud, MAXIMO)));
      const { signal } = validarOpciones(opciones);
      return ejecutar({ ruta: RUTA_CANCELACIONES_EXPEDIENTE, entrada, signal, estadoEsperado: 201, maximoSolicitud: MAXIMO, maximoRespuesta: MAXIMO,
        efecto: true, validarRespuesta: (respuesta) => validarReciboCancelacion(respuesta, entrada), rechazoDeterminado: rechazo });
    },
  });
}
