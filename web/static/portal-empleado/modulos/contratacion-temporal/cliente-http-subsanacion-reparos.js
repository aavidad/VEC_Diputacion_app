export const RUTA_SUBSANACION_REPAROS = "/api/vec/contratacion-temporal/subsanacion-reparos";
export const ESQUEMA_RECUPERACION_SUBSANACION = "vec.contratacion-temporal.subsanacion-recuperacion.v1";
export const MAXIMO_ARCHIVO_RECUPERACION_SUBSANACION = 16 * 1024;
const CAMPOS = ["expediente_ref", "version_esperada", "clave_idempotencia", "observaciones"];
const REF = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/u;
const INSTANTE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$/u;

function registro(valor, campos) {
  if (!valor || typeof valor !== "object" || Array.isArray(valor)
    || Object.getPrototypeOf(valor) !== Object.prototype || Object.keys(valor).length !== campos.length
    || !campos.every((campo) => Object.hasOwn(valor, campo))) throw new TypeError("contrato de subsanación no válido");
  return valor;
}
export function validarSolicitudSubsanacionReparos(valor) {
  const s = registro(valor, CAMPOS);
  if (!REF.test(s.expediente_ref) || !Number.isSafeInteger(s.version_esperada) || s.version_esperada < 1
    || !UUID.test(s.clave_idempotencia) || s.clave_idempotencia === "00000000-0000-4000-8000-000000000000"
    || typeof s.observaciones !== "string" || s.observaciones !== s.observaciones.trim()
    || s.observaciones.length < 1 || s.observaciones.length > 2000 || /[\u0000-\u001f\u007f]/u.test(s.observaciones)) throw new TypeError("solicitud de subsanación no válida");
  return Object.freeze(Object.fromEntries(CAMPOS.map((campo) => [campo, s[campo]])));
}
export function validarVersionesRecuperacionSubsanacion(contexto, versiones = []) {
  if (!contexto || typeof contexto.expediente_ref !== "string" || !REF.test(contexto.expediente_ref)
    || !Number.isSafeInteger(contexto.version_esperada) || contexto.version_esperada < 1
    || !Array.isArray(versiones) || versiones.length > 512
    || versiones.some((version) => !Number.isSafeInteger(version) || version < 1 || version >= contexto.version_esperada)
    || new Set(versiones).size !== versiones.length) throw new TypeError("versiones de recuperación de subsanación no válidas");
  return Object.freeze([...versiones]);
}
export function validarDatosRecuperacionSubsanacion(valor, contexto, versionesRecuperacionAnteriores = []) {
  const versiones = validarVersionesRecuperacionSubsanacion(contexto, versionesRecuperacionAnteriores);
  const datos = registro(valor, ["esquema", "solicitud"]);
  if (datos.esquema !== ESQUEMA_RECUPERACION_SUBSANACION) throw new TypeError("esquema de recuperación de subsanación no válido");
  const solicitud = validarSolicitudSubsanacionReparos(datos.solicitud);
  if (!contexto || solicitud.expediente_ref !== contexto.expediente_ref
    || (solicitud.version_esperada !== contexto.version_esperada
      && !versiones.includes(solicitud.version_esperada))) throw new TypeError("archivo de recuperación ajeno al expediente");
  return Object.freeze({ esquema: ESQUEMA_RECUPERACION_SUBSANACION, solicitud });
}
export function serializarDatosRecuperacionSubsanacion(solicitud) {
  const datos = { esquema: ESQUEMA_RECUPERACION_SUBSANACION, solicitud: validarSolicitudSubsanacionReparos(solicitud) };
  const texto = JSON.stringify(datos);
  if (new TextEncoder().encode(texto).byteLength > MAXIMO_ARCHIVO_RECUPERACION_SUBSANACION) throw new TypeError("archivo de recuperación demasiado grande");
  return texto;
}
export function validarReciboSubsanacionReparos(valor, solicitud) {
  const r = registro(valor, ["esquema", "operacion", "expediente_ref", "version_resultante", "fase_resultante", "estado_resultante", "recibo_ref", "auditoria_ref", "evento_ref", "actor_ref", "registrada_en"]);
  if (r.esquema !== "vec.contratacion-temporal.recibo-subsanacion-reparos.v1" || r.operacion !== "registrar_subsanacion"
    || r.expediente_ref !== solicitud.expediente_ref || r.version_resultante !== solicitud.version_esperada + 1
    || r.fase_resultante !== "subsanacion_unidad" || r.estado_resultante !== "incidencia"
    || ![r.recibo_ref, r.auditoria_ref, r.evento_ref, r.actor_ref].every((ref) => typeof ref === "string" && REF.test(ref))
    || typeof r.registrada_en !== "string" || !INSTANTE.test(r.registrada_en) || !Number.isFinite(Date.parse(r.registrada_en))) throw new TypeError("recibo de subsanación no válido");
  return Object.freeze({ ...r });
}
export function crearClienteSubsanacionReparosHTTP({ ejecutar, validarOpciones, serializarAcotado } = {}) {
  if (typeof ejecutar !== "function" || typeof validarOpciones !== "function" || typeof serializarAcotado !== "function") throw new TypeError("dependencias HTTP de subsanación no disponibles");
  return Object.freeze({
    registrarSubsanacionReparos(solicitud, opciones) {
      const entrada = validarSolicitudSubsanacionReparos(JSON.parse(serializarAcotado(solicitud, 16 * 1024)));
      const { signal } = validarOpciones(opciones);
      return ejecutar({ ruta: RUTA_SUBSANACION_REPAROS, entrada, signal, estadoEsperado: 201, maximoSolicitud: 16 * 1024, maximoRespuesta: 16 * 1024, efecto: true,
        validarRespuesta: (respuesta) => validarReciboSubsanacionReparos(respuesta, entrada),
        rechazoDeterminado: (error) => error?.envelopeValido === true && ["400:peticion_no_permitida", "403:acceso_denegado", "409:conflicto", "422:contenido_no_valido"].includes(`${error.estado}:${error.codigo}`), });
    },
  });
}
