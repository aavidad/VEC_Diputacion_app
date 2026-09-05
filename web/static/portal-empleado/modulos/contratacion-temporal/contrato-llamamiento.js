/** Frontera de datos de llamamiento: referencias opacas, nunca datos de persona. */
const REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/u;
const INSTANTE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$/u;
export const CAMPOS_SELECCION = Object.freeze([
  "expediente_ref", "version_esperada", "clave_idempotencia",
]);
export const CAMPOS_COMUNICACION = Object.freeze([
  "clave_idempotencia", "organizacion_ref", "expediente_ref", "llamamiento_ref",
  "version_esperada", "prueba_entrega_ref",
]);

function exigir(condicion) {
  if (!condicion) throw new TypeError("contrato de llamamiento no válido");
}
export function referenciaLlamamientoValida(valor) {
  return typeof valor === "string" && REFERENCIA.test(valor);
}
function instante(valor) {
  return typeof valor === "string" && INSTANTE.test(valor)
    && Number.isFinite(Date.parse(valor))
    && new Date(valor).toISOString().slice(0, 19) === valor.slice(0, 19);
}
function entero(valor) { return Number.isSafeInteger(valor) && valor > 0; }
function registro(valor, campos) {
  exigir(valor !== null && typeof valor === "object" && !Array.isArray(valor)
    && Object.getPrototypeOf(valor) === Object.prototype
    && Object.getOwnPropertySymbols(valor).length === 0);
  const descriptores = Object.getOwnPropertyDescriptors(valor);
  exigir(Object.keys(descriptores).length === campos.length
    && campos.every((campo) => Object.hasOwn(descriptores, campo)
      && Object.hasOwn(descriptores[campo], "value") && descriptores[campo].enumerable));
  return Object.freeze(Object.fromEntries(campos.map((campo) => [campo, valor[campo]])));
}
function solicitud(entrada, campos) {
  const valor = registro(entrada, campos);
  exigir(typeof valor.clave_idempotencia === "string" && UUID.test(valor.clave_idempotencia)
    && valor.clave_idempotencia !== "00000000-0000-4000-8000-000000000000"
    && entero(valor.version_esperada) && valor.version_esperada < Number.MAX_SAFE_INTEGER
    && campos.filter((campo) => campo.endsWith("_ref")).every(
      (campo) => referenciaLlamamientoValida(valor[campo]),
    ));
  return valor;
}
export function validarSolicitudSeleccionLlamamiento(entrada) {
  return solicitud(entrada, CAMPOS_SELECCION);
}
export function validarSolicitudComunicacionLlamamiento(entrada) {
  return solicitud(entrada, CAMPOS_COMUNICACION);
}
export function validarReciboSeleccionLlamamiento(entrada) {
  const valor = registro(entrada, [
    "esquema", "estado", "recibo_ref", "confirmada_en",
    "organizacion_ref", "llamamiento_ref", "version_llamamiento",
  ]);
  exigir(valor.esquema === "vec.contratacion-temporal.recibo-seleccion-llamamiento.v1"
    && valor.estado === "confirmado" && referenciaLlamamientoValida(valor.recibo_ref)
    && referenciaLlamamientoValida(valor.organizacion_ref)
    && referenciaLlamamientoValida(valor.llamamiento_ref)
    && entero(valor.version_llamamiento) && valor.version_llamamiento < Number.MAX_SAFE_INTEGER
    && instante(valor.confirmada_en));
  return valor;
}
export function validarReciboComunicacionLlamamiento(entrada, solicitudEntrada) {
  const descriptorEstado = entrada && Object.getOwnPropertyDescriptor(entrada, "estado_local");
  const local = ["registrada_localmente", "replay_registrada_localmente"]
    .includes(descriptorEstado?.value);
  const valor = registro(entrada, [
    "esquema", "estado_local", "comunicacion_ref", "recibo_ref", "auditoria_ref",
    "version_resultante", ...(local ? ["registrada_en", "intencion_envio_ref"] : ["respuesta_hasta"]),
  ]);
  exigir(valor.esquema === "vec.contratacion-temporal.registro-comunicacion-llamamiento.v1"
    && (local || ["confirmado", "replay_confirmado"].includes(valor.estado_local))
    && ["comunicacion_ref", "recibo_ref", "auditoria_ref"].every(
      (campo) => referenciaLlamamientoValida(valor[campo]),
    ) && entero(valor.version_resultante)
    && (local ? instante(valor.registrada_en) && referenciaLlamamientoValida(valor.intencion_envio_ref)
      : instante(valor.respuesta_hasta))
    && valor.version_resultante === solicitudEntrada.version_esperada + 1);
  return valor;
}
