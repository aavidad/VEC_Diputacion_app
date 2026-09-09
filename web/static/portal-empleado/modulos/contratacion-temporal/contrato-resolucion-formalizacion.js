const REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/u;
const SHA256 = /^[a-f0-9]{64}$/u;
const FECHA = /^\d{4}-\d{2}-\d{2}$/u;
export const CAMPOS_SOLICITUD_RESOLUCION_FORMALIZACION = Object.freeze([
  "expediente_ref", "version_esperada", "propuesta_ref", "clave_idempotencia",
  "numero_resolucion", "fecha_resolucion", "motivo", "confirma_revision_propuesta", "confirma_ejercicio_manual",
]);
export const CAMPOS_RECIBO_RESOLUCION_FORMALIZACION = Object.freeze([
  "esquema", "estado", "expediente_ref", "version_resultante", "propuesta_ref", "resolucion_formalizacion_ref",
  "documento_resolucion_ref", "documento_resolucion_version", "documento_resolucion_sha256", "actuacion_ref",
  "auditoria_ref", "outbox_ref", "recibo_ref", "registrada_en", "tipo_validacion", "firma_oficial", "eficacia_administrativa",
]);
export const CAMPOS_PREPARACION_RESOLUCION_FORMALIZACION = Object.freeze([
  "esquema", "expediente_ref", "propuesta_ref", "version_esperada", "version_actual", "recibo",
]);
function fallo() { throw new TypeError("contrato de resolución de formalización no válido"); }
function registro(valor, campos) {
  if (!valor || typeof valor !== "object" || Array.isArray(valor) || Object.getPrototypeOf(valor) !== Object.prototype
    || Object.getOwnPropertySymbols(valor).length || Object.keys(valor).length !== campos.length
    || !campos.every((campo) => Object.hasOwn(valor, campo))) fallo();
  return Object.freeze(Object.fromEntries(campos.map((campo) => [campo, valor[campo]])));
}
function texto(valor, maximo) { return typeof valor === "string" && valor.normalize("NFC") === valor && valor.trim() === valor && valor.length > 0 && valor.length <= maximo && !/[\u0000-\u001f\u007f-\u009f]/u.test(valor); }
function fecha(val) { return typeof val === "string" && FECHA.test(val) && Number.isFinite(Date.parse(`${val}T00:00:00Z`)); }
function instante(val) { return typeof val === "string" && /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,6})?Z$/u.test(val) && Number.isFinite(Date.parse(val)); }
export function validarSolicitudResolucionFormalizacion(entrada) {
  const valor = registro(entrada, CAMPOS_SOLICITUD_RESOLUCION_FORMALIZACION);
  if (!REFERENCIA.test(valor.expediente_ref) || !REFERENCIA.test(valor.propuesta_ref)
    || !Number.isSafeInteger(valor.version_esperada) || valor.version_esperada !== 7
    || !UUID.test(valor.clave_idempotencia) || valor.clave_idempotencia === "00000000-0000-4000-8000-000000000000"
    || !texto(valor.numero_resolucion, 80) || !fecha(valor.fecha_resolucion) || !texto(valor.motivo, 2000)
    || valor.confirma_revision_propuesta !== true || valor.confirma_ejercicio_manual !== true) fallo();
  return valor;
}
export function validarReciboResolucionFormalizacion(entrada, solicitud) {
  const esperada = validarSolicitudResolucionFormalizacion(solicitud);
  return validarReciboResolucionFormalizacionPara(entrada, esperada.expediente_ref, esperada.propuesta_ref);
}
function validarReciboResolucionFormalizacionPara(entrada, expedienteRef, propuestaRef) {
  const valor = registro(entrada, CAMPOS_RECIBO_RESOLUCION_FORMALIZACION);
  if (valor.esquema !== "vec.contratacion-temporal.resolucion-formalizacion.v1"
    || !["registrada", "replay_registrada"].includes(valor.estado)
    || valor.expediente_ref !== expedienteRef || valor.propuesta_ref !== propuestaRef
    || valor.version_resultante !== 8 || valor.tipo_validacion !== "manual_de_ejercicio"
    || valor.firma_oficial !== false || valor.eficacia_administrativa !== false
    || !["resolucion_formalizacion_ref", "documento_resolucion_ref", "actuacion_ref", "auditoria_ref", "outbox_ref", "recibo_ref"].every((campo) => REFERENCIA.test(valor[campo]))
    || !Number.isSafeInteger(valor.documento_resolucion_version) || valor.documento_resolucion_version < 1
    || !SHA256.test(valor.documento_resolucion_sha256) || valor.documento_resolucion_sha256 === "0".repeat(64) || !instante(valor.registrada_en)) fallo();
  return valor;
}
export function validarPreparacionResolucionFormalizacion(entrada, expedienteRef) {
  const valor = registro(entrada, CAMPOS_PREPARACION_RESOLUCION_FORMALIZACION);
  if (valor.esquema !== "vec.contratacion-temporal.resolucion-formalizacion.preparacion.v1"
    || valor.expediente_ref !== expedienteRef || !REFERENCIA.test(valor.propuesta_ref)
    || valor.version_esperada !== 7 || ![7, 8].includes(valor.version_actual)
    || (valor.recibo !== null && validarReciboResolucionFormalizacionPara(valor.recibo, valor.expediente_ref, valor.propuesta_ref) === undefined)
    || (valor.version_actual === 7 && valor.recibo !== null) || (valor.version_actual === 8 && valor.recibo === null)) fallo();
  return valor;
}
