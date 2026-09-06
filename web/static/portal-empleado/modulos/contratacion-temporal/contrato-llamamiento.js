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
export const CAMPOS_SIGUIENTE = Object.freeze([
  "clave_idempotencia", "organizacion_ref", "expediente_ref", "resolucion_ref", "intencion_ref",
]);
export const CAMPOS_RECIBO_SIGUIENTE = Object.freeze([
  "esquema", "organizacion_ref", "expediente_ref", "resolucion_ref", "intencion_ref",
  "llamamiento_anterior_ref", "llamamiento_ref", "version_llamamiento", "recibo_bolsa_ref",
  "recibo_ref", "auditoria_ref", "confirmada_en", "estado_intencion", "estado_local",
]);
export const PUBLICACIONES_FORMALIZACION = Object.freeze({
  tipo_formalizacion: "tipo:ct:propuesta-desarrollo:20260906",
  plantilla: "plantilla:ct:propuesta-desarrollo:20260906",
  politica_firma: "politica:ct:firma-pendiente:20260906",
  plan_firma: "plan:ct:firma-pendiente:20260906",
});
export const CAMPOS_PROPUESTA = Object.freeze([
  "clave_idempotencia", "expediente_ref", "llamamiento_ref", "resolucion_llamamiento_aceptada_ref",
  "recibo_resolucion_aceptada_ref", "version_esperada", "tipo_formalizacion", "plantilla",
  "anexos", "politica_firma", "plan_firma",
]);
export const CAMPOS_RECIBO_PROPUESTA = Object.freeze([
  "esquema", "estado_local", "propuesta_ref", "recibo_local_ref", "version_resultante", "confirmada_en",
]);

export async function snapshotsFormalizacionDesarrollo(entrada, criptografia) {
  const asset = registro(entrada, ["esquema", "publicaciones"]);
  exigir(asset.esquema === "vec.ct.propuesta.publicaciones-desarrollo.v1" && criptografia?.subtle?.digest);
  const publicaciones = registro(asset.publicaciones, Object.keys(PUBLICACIONES_FORMALIZACION));
  const snapshots = {};
  for (const [campo, referencia] of Object.entries(PUBLICACIONES_FORMALIZACION)) {
    const p = registro(publicaciones[campo], ["referencia", "version", "contenido"]);
    exigir(p.referencia === referencia && p.version === 1 && typeof p.contenido === "string"
      && p.contenido.trim().length > 0 && !/[\u0000\uD800-\uDFFF]/u.test(p.contenido));
    const bytes = new TextEncoder().encode(p.contenido);
    exigir(bytes.length <= 4096);
    const digest = new Uint8Array(await criptografia.subtle.digest("SHA-256", bytes));
    exigir(digest.length === 32);
    const huella_sha256 = Array.from(digest, (b) => b.toString(16).padStart(2, "0")).join("");
    exigir(huella_sha256 !== "0".repeat(64));
    snapshots[campo] = Object.freeze({ referencia, version: 1, huella_sha256 });
  }
  return Object.freeze(snapshots);
}
export function validarSolicitudPropuestaFormalizacion(entrada) {
  const valor = solicitud(entrada, CAMPOS_PROPUESTA);
  exigir(valor.version_esperada === 6 && Array.isArray(valor.anexos) && valor.anexos.length === 0);
  const snapshots = {};
  for (const [campo, referencia] of Object.entries(PUBLICACIONES_FORMALIZACION)) {
    const p = registro(valor[campo], ["referencia", "version", "huella_sha256"]);
    exigir(p.referencia === referencia && p.version === 1 && typeof p.huella_sha256 === "string"
      && /^[a-f0-9]{64}$/u.test(p.huella_sha256) && p.huella_sha256 !== "0".repeat(64));
    snapshots[campo] = p;
  }
  // Este recorrido mínimo no adjunta documentos ni crea una orden de firma.
  return registro({ ...valor, ...snapshots, anexos: Object.freeze([]) }, CAMPOS_PROPUESTA);
}
export function validarReciboPropuestaFormalizacion(entrada, solicitudEntrada, aceptadaEn) {
  const esperada = validarSolicitudPropuestaFormalizacion(solicitudEntrada);
  const valor = registro(entrada, CAMPOS_RECIBO_PROPUESTA);
  exigir(valor.esquema === "vec.contratacion-temporal.propuesta-formalizacion-local.v1"
    && ["confirmado", "replay_confirmado"].includes(valor.estado_local)
    && valor.version_resultante === esperada.version_esperada + 1
    && referenciaLlamamientoValida(valor.propuesta_ref) && referenciaLlamamientoValida(valor.recibo_local_ref));
  const confirmada = instanteRespuesta(valor.confirmada_en);
  if (aceptadaEn !== undefined) exigir(confirmada >= instanteRespuesta(aceptadaEn));
  return valor;
}

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
    && (version === null || (entero(valor[version]) && valor[version] < Number.MAX_SAFE_INTEGER))
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
export function validarSolicitudContinuacionLlamamiento(entrada) {
  return solicitud(entrada, CAMPOS_SIGUIENTE, null);
}
export function validarReciboContinuacionLlamamiento(entrada, solicitudEntrada, llamamientoAnteriorRef) {
  const esperada = validarSolicitudContinuacionLlamamiento(solicitudEntrada);
  const valor = registro(entrada, CAMPOS_RECIBO_SIGUIENTE);
  exigir(valor.esquema === "vec.contratacion-temporal.continuacion-llamamiento.v1"
    && CAMPOS_SIGUIENTE.filter((campo) => campo !== "clave_idempotencia").every(
      (campo) => valor[campo] === esperada[campo])
    && CAMPOS_RECIBO_SIGUIENTE.filter((campo) => campo.endsWith("_ref")).every(
      (campo) => referenciaLlamamientoValida(valor[campo]))
    && valor.version_llamamiento === 1 && valor.estado_intencion === "despachada"
    && ["confirmado", "replay_confirmado"].includes(valor.estado_local)
    && valor.llamamiento_ref !== valor.llamamiento_anterior_ref);
  // El formulario conserva este antecedente; no se añade al material HTTP de cinco campos.
  if (llamamientoAnteriorRef !== undefined) exigir(referenciaLlamamientoValida(llamamientoAnteriorRef)
    && valor.llamamiento_anterior_ref === llamamientoAnteriorRef);
  instanteRespuesta(valor.confirmada_en);
  return valor;
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
