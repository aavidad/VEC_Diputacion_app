/** Contrato cerrado para registrar el resultado real de fiscalización. */

const MAXIMO_BYTES_JSON = 16 * 1024;
const MAXIMO_BYTES_OBSERVACIONES = 8 * 1024;
const MAXIMO_CARACTERES_OBSERVACIONES = 2000;
const PATRON_REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const PATRON_CLAVE = /^[a-z][a-z0-9._-]{1,79}$/u;
const PATRON_UUID_V4 =
  /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/u;
const PATRON_INSTANTE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$/u;
const UUID_V4_NULO = "00000000-0000-4000-8000-000000000000";
const RESULTADOS = new Set([
  "favorable", "favorable_con_observaciones", "desfavorable",
]);
const CAMPOS_SOLICITUD = Object.freeze([
  "expediente_ref", "version_esperada", "clave_idempotencia", "resultado",
  "observaciones",
]);
const CAMPOS_RECIBO_OBLIGATORIOS = Object.freeze([
  "esquema", "operacion", "expediente_ref", "version_resultante", "resultado",
  "fase_resultante", "estado_resultante", "recibo_ref", "auditoria_ref",
  "evento_ref", "actor_ref", "registrada_en",
]);
const CAMPOS_RECIBO_OPCIONALES = Object.freeze([
  "unidad_retorno_ref", "responsable_retorno_ref",
]);

function falloContrato() {
  throw new TypeError("contrato JSON de fiscalización no válido");
}

function bytesUTF8(texto, limite = MAXIMO_BYTES_JSON) {
  let bytes = 0;
  for (let indice = 0; indice < texto.length; indice += 1) {
    const unidad = texto.charCodeAt(indice);
    if (unidad <= 0x7f) bytes += 1;
    else if (unidad <= 0x7ff) bytes += 2;
    else if (unidad >= 0xd800 && unidad <= 0xdbff
      && indice + 1 < texto.length
      && texto.charCodeAt(indice + 1) >= 0xdc00
      && texto.charCodeAt(indice + 1) <= 0xdfff) {
      bytes += 4;
      indice += 1;
    } else bytes += 3;
    if (bytes > limite) return bytes;
  }
  return bytes;
}

function esRegistroPlano(valor) {
  if (valor === null || typeof valor !== "object" || Array.isArray(valor)) return false;
  try {
    return Object.getPrototypeOf(valor) === Object.prototype
      && Object.getOwnPropertySymbols(valor).length === 0
      && Object.values(Object.getOwnPropertyDescriptors(valor)).every(
        (descriptor) => Object.hasOwn(descriptor, "value") && descriptor.enumerable,
      );
  } catch {
    return false;
  }
}

function analizarRegistro(texto, obligatorios, opcionales = [], ordenCanonico = false) {
  if (typeof texto !== "string" || bytesUTF8(texto) > MAXIMO_BYTES_JSON) falloContrato();
  let datos;
  try { datos = JSON.parse(texto); } catch { falloContrato(); }
  const permitidos = [...obligatorios, ...opcionales];
  const recibidos = esRegistroPlano(datos) ? Object.keys(datos) : [];
  if (!esRegistroPlano(datos)
    || obligatorios.some((campo) => !Object.hasOwn(datos, campo))
    || recibidos.some((campo) => !permitidos.includes(campo))) falloContrato();
  if (ordenCanonico) {
    const canonico = Object.fromEntries(obligatorios.map((campo) => [campo, datos[campo]]));
    if (recibidos.length !== obligatorios.length || JSON.stringify(canonico) !== texto) {
      falloContrato();
    }
  }
  return datos;
}

function referencia(valor) {
  return typeof valor === "string" && PATRON_REFERENCIA.test(valor);
}

function clave(valor) {
  return typeof valor === "string" && PATRON_CLAVE.test(valor);
}

function entero(valor, minimo = 1) {
  return Number.isSafeInteger(valor) && valor >= minimo;
}

function observacionesValidas(valor, obligatorias) {
  if (typeof valor !== "string" || valor !== valor.trim()
    || valor.normalize("NFC") !== valor
    || [...valor].length > MAXIMO_CARACTERES_OBSERVACIONES
    || bytesUTF8(valor, MAXIMO_BYTES_OBSERVACIONES) > MAXIMO_BYTES_OBSERVACIONES
    || /[\u0000-\u0008\u000b\u000c\u000e-\u001f\u007f-\u009f]/u.test(valor)) {
    return false;
  }
  return !obligatorias || valor.length > 0;
}

function instante(valor) {
  return typeof valor === "string" && PATRON_INSTANTE.test(valor)
    && Number.isFinite(Date.parse(valor));
}

export function validarSolicitudResultadoFiscalizacion(texto) {
  const datos = analizarRegistro(texto, CAMPOS_SOLICITUD, [], true);
  const requiereObservaciones = datos.resultado !== "favorable";
  if (!referencia(datos.expediente_ref) || !entero(datos.version_esperada)
    || datos.version_esperada >= Number.MAX_SAFE_INTEGER
    || typeof datos.clave_idempotencia !== "string"
    || !PATRON_UUID_V4.test(datos.clave_idempotencia)
    || datos.clave_idempotencia === UUID_V4_NULO || !RESULTADOS.has(datos.resultado)
    || (datos.resultado === "favorable" && datos.observaciones !== "")
    || !observacionesValidas(datos.observaciones, requiereObservaciones)) falloContrato();
  return Object.freeze(Object.fromEntries(
    CAMPOS_SOLICITUD.map((campo) => [campo, datos[campo]]),
  ));
}

export function validarReciboResultadoFiscalizacion(texto) {
  const datos = analizarRegistro(
    texto,
    CAMPOS_RECIBO_OBLIGATORIOS,
    CAMPOS_RECIBO_OPCIONALES,
  );
  const tieneUnidad = Object.hasOwn(datos, "unidad_retorno_ref");
  const tieneResponsable = Object.hasOwn(datos, "responsable_retorno_ref");
  const transicionValida = datos.resultado === "desfavorable"
    ? datos.fase_resultante === "subsanacion_unidad"
      && datos.estado_resultante === "incidencia" && tieneUnidad && tieneResponsable
    : datos.fase_resultante === "fiscalizacion"
      && datos.estado_resultante === "en_curso" && !tieneUnidad && !tieneResponsable;
  if (datos.esquema !== "vec.contratacion-temporal.recibo-fiscalizacion.v1"
    || datos.operacion !== "registrar_resultado" || !referencia(datos.expediente_ref)
    || !entero(datos.version_resultante, 2) || !RESULTADOS.has(datos.resultado)
    || !clave(datos.fase_resultante) || !clave(datos.estado_resultante)
    || !referencia(datos.recibo_ref) || !referencia(datos.auditoria_ref)
    || !referencia(datos.evento_ref) || !referencia(datos.actor_ref)
    || !instante(datos.registrada_en) || tieneUnidad !== tieneResponsable || !transicionValida
    || (tieneUnidad && (!referencia(datos.unidad_retorno_ref)
      || !referencia(datos.responsable_retorno_ref)))) falloContrato();
  return Object.freeze(Object.fromEntries([
    ...CAMPOS_RECIBO_OBLIGATORIOS,
    ...CAMPOS_RECIBO_OPCIONALES.filter((campo) => Object.hasOwn(datos, campo)),
  ].map((campo) => [campo, datos[campo]])));
}
