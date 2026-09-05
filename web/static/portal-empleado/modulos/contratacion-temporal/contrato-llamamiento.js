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
// El orden forma parte de la representación JSON canónica del POST.
export const CAMPOS_RESPUESTA_RECIBIDA = Object.freeze([
  "clave_idempotencia", "organizacion_ref", "expediente_ref", "llamamiento_ref",
  "comunicacion_ref", "version_comunicacion_esperada", "respuesta", "correo_ref",
  "correo_sha256", "recibida_en",
]);
export const CAMPOS_RESPUESTA_EDITABLES = Object.freeze([
  "clave_idempotencia", "respuesta", "correo_ref", "recibida_en",
]);
// Configuración fija de este ejercicio de desarrollo; no concede autoridad.
export const CRITERIO_VALIDACION_RESOLUCION_DESARROLLO = "politica:ct:revision-manual-sintetica:20260906";
export const RESPUESTAS_RESOLUCION = Object.freeze(["aceptacion", "renuncia"]);
export const CAMPOS_REVISION_RESOLUCION = Object.freeze([
  "revision_respuesta_rrhh", "revision_plazo_rrhh",
]);
export const CAMPOS_RESOLUCION = Object.freeze([
  "clave_idempotencia", "organizacion_ref", "expediente_ref", "llamamiento_ref",
  "comunicacion_ref", "version_esperada", "respuesta", "prueba_respuesta_ref",
  ...CAMPOS_REVISION_RESOLUCION, "criterio_validacion_ref",
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
function solicitud(entrada, campos, version = "version_esperada") {
  const valor = registro(entrada, campos);
  exigir(typeof valor.clave_idempotencia === "string" && UUID.test(valor.clave_idempotencia)
    && valor.clave_idempotencia !== "00000000-0000-4000-8000-000000000000"
    && entero(valor[version]) && valor[version] < Number.MAX_SAFE_INTEGER
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
// Conserva los seis decimales para comparar instantes sin perder microsegundos
// con Date (milisegundos). Go puede omitir ceros finales en el eco del recibo.
function instanteRespuesta(valor) {
  exigir(instante(valor) && /^[0-9]{4}-.+T\d{2}:\d{2}:\d{2}(?:\.\d{1,6})?Z$/u.test(valor)
    && !valor.startsWith("0000-"));
  return `${valor.slice(0, 19)}.${(valor.match(/\.(\d+)Z$/u)?.[1] ?? "").padEnd(6, "0")}Z`;
}
export function validarSolicitudRespuestaRecibida(entrada) {
  const valor = solicitud(entrada, CAMPOS_RESPUESTA_RECIBIDA, "version_comunicacion_esperada");
  exigir(valor.version_comunicacion_esperada === 2
    && RESPUESTAS_RESOLUCION.includes(valor.respuesta)
    && typeof valor.correo_sha256 === "string" && /^[0-9a-f]{64}$/u.test(valor.correo_sha256)
    && valor.correo_sha256 !== "0".repeat(64));
  instanteRespuesta(valor.recibida_en);
  return valor;
}
export function validarReciboRespuestaRecibida(entrada, solicitudEntrada) {
  const valor = registro(entrada, [...CAMPOS_RESPUESTA_RECIBIDA,
    "esquema", "justificante_ref", "recibo_ref", "auditoria_ref", "registrada_en", "estado"]);
  const esperada = validarSolicitudRespuestaRecibida(solicitudEntrada);
  exigir(valor.esquema === "vec.contratacion-temporal.respuesta-recibida-llamamiento.v1"
    && ["registrada_por_rrhh", "replay_registrada_por_rrhh"].includes(valor.estado)
    && ["justificante_ref", "recibo_ref", "auditoria_ref"].every(
      (campo) => referenciaLlamamientoValida(valor[campo]),
    ) && CAMPOS_RESPUESTA_RECIBIDA.every((campo) => campo === "recibida_en"
      ? instanteRespuesta(valor[campo]) === instanteRespuesta(esperada[campo])
      : valor[campo] === esperada[campo]));
  exigir(instanteRespuesta(valor.registrada_en) >= instanteRespuesta(valor.recibida_en));
  return valor;
}
// La respuesta procede del justificante; este no concede plazo ni
// autorización: ambos se comprueban en el servidor antes de producir un recibo.
export function validarSolicitudResolucionLlamamiento(entrada) {
  const valor = solicitud(entrada, CAMPOS_RESOLUCION);
  exigir(valor.version_esperada === 2 && RESPUESTAS_RESOLUCION.includes(valor.respuesta)
    && CAMPOS_REVISION_RESOLUCION.every((campo) => valor[campo] === true)
    && valor.criterio_validacion_ref === CRITERIO_VALIDACION_RESOLUCION_DESARROLLO);
  return valor;
}
export function validarReciboResolucionLlamamiento(entrada, solicitudEntrada) {
  const esperada = validarSolicitudResolucionLlamamiento(solicitudEntrada);
  const renuncia = esperada.respuesta === "renuncia";
  // Intención obligatoria en renuncia y prohibida, incluso null, en aceptación.
  const valor = registro(entrada, [
    "esquema", "respuesta", "estado_plazo", "estado_local", "resolucion_ref",
    "recibo_local_ref", "auditoria_ref", "version_resultante", "resuelta_en",
    ...(renuncia ? ["intencion_siguiente"] : []),
  ]);
  exigir(valor.esquema === "vec.contratacion-temporal.resolucion-comunicacion-llamamiento.v1"
    && valor.respuesta === esperada.respuesta && valor.estado_plazo === "vigente"
    && ["confirmado", "replay_confirmado"].includes(valor.estado_local)
    && entero(valor.version_resultante) && valor.version_resultante === esperada.version_esperada + 1
    && ["resolucion_ref", "recibo_local_ref", "auditoria_ref"].every(
      (campo) => referenciaLlamamientoValida(valor[campo]),
    ));
  const resuelta = instanteRespuesta(valor.resuelta_en);
  if (!renuncia) return valor;
  const intencion = registro(valor.intencion_siguiente, ["referencia", "estado_local", "actualizada_en"]);
  exigir(referenciaLlamamientoValida(intencion.referencia) && intencion.estado_local === "pendiente"
    && instanteRespuesta(intencion.actualizada_en) >= resuelta);
  return Object.freeze({ ...valor, intencion_siguiente: intencion });
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
