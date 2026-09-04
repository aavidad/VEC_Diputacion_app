/** Contrato cerrado del informe jurídico de desarrollo. */

const MAXIMO_BYTES_JSON = 288 * 1024;
const MAXIMO_BYTES_CONTENIDO = 256 * 1024;
const PATRON_REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const PATRON_UUID_V4 =
  /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/u;
const PATRON_INSTANTE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$/u;
const PATRON_HUELLA = /^[a-f0-9]{64}$/u;
const UUID_V4_NULO = "00000000-0000-4000-8000-000000000000";
const CAMPOS_SOLICITUD = Object.freeze([
  "expediente_ref", "version_esperada", "clave_idempotencia",
]);
const CAMPOS_RECIBO = Object.freeze([
  "esquema", "operacion", "expediente_ref", "version_resultante",
  "informe_ref", "documento_ref", "version_documento", "formato", "nombre",
  "huella_documento_sha256", "recibo_ref", "auditoria_ref", "evento_ref",
  "contenido_desarrollo", "confirmada_en",
]);

function falloContrato() {
  throw new TypeError("contrato JSON de informe jurídico no válido");
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

function analizarRegistro(texto, campos, ordenCanonico = false) {
  if (typeof texto !== "string" || bytesUTF8(texto) > MAXIMO_BYTES_JSON) falloContrato();
  let datos;
  try { datos = JSON.parse(texto); } catch { falloContrato(); }
  if (datos === null || typeof datos !== "object" || Array.isArray(datos)
    || Object.getPrototypeOf(datos) !== Object.prototype
    || Object.getOwnPropertySymbols(datos).length !== 0
    || Object.keys(datos).length !== campos.length
    || !campos.every((campo) => Object.hasOwn(datos, campo))
    || !Object.values(Object.getOwnPropertyDescriptors(datos)).every(
      (descriptor) => Object.hasOwn(descriptor, "value") && descriptor.enumerable,
    )) falloContrato();
  if (ordenCanonico) {
    const canonico = Object.fromEntries(campos.map((campo) => [campo, datos[campo]]));
    if (JSON.stringify(canonico) !== texto) falloContrato();
  }
  return datos;
}

function referencia(valor) {
  return typeof valor === "string" && PATRON_REFERENCIA.test(valor);
}

function entero(valor, minimo = 1) {
  return Number.isSafeInteger(valor) && valor >= minimo;
}

function textoSeguro(valor, maximo, permiteSaltos = false) {
  if (typeof valor !== "string" || valor.length === 0 || valor.normalize("NFC") !== valor
    || bytesUTF8(valor, maximo) > maximo) return false;
  const controles = permiteSaltos
    ? /[\u0000-\u0008\u000b\u000c\u000e-\u001f\u007f-\u009f]/u
    : /[\u0000-\u001f\u007f-\u009f]/u;
  return !controles.test(valor);
}

function instante(valor) {
  return typeof valor === "string" && PATRON_INSTANTE.test(valor)
    && Number.isFinite(Date.parse(valor));
}

export function validarSolicitudInformeJuridico(texto) {
  const datos = analizarRegistro(texto, CAMPOS_SOLICITUD, true);
  if (!referencia(datos.expediente_ref) || !entero(datos.version_esperada)
    || datos.version_esperada >= Number.MAX_SAFE_INTEGER
    || typeof datos.clave_idempotencia !== "string"
    || !PATRON_UUID_V4.test(datos.clave_idempotencia)
    || datos.clave_idempotencia === UUID_V4_NULO) falloContrato();
  return Object.freeze({
    expediente_ref: datos.expediente_ref,
    version_esperada: datos.version_esperada,
    clave_idempotencia: datos.clave_idempotencia,
  });
}

export function validarReciboInformeJuridico(texto) {
  const datos = analizarRegistro(texto, CAMPOS_RECIBO);
  if (datos.esquema !== "vec.contratacion-temporal.recibo-informe-juridico.v1"
    || datos.operacion !== "preparar" || !referencia(datos.expediente_ref)
    || !entero(datos.version_resultante, 2) || !referencia(datos.informe_ref)
    || !referencia(datos.documento_ref) || !entero(datos.version_documento)
    || datos.formato !== "text/plain; charset=utf-8"
    || !textoSeguro(datos.nombre, 180) || !PATRON_HUELLA.test(datos.huella_documento_sha256)
    || !textoSeguro(datos.contenido_desarrollo, MAXIMO_BYTES_CONTENIDO, true)
    || !datos.contenido_desarrollo.includes("DOCUMENTO DE DESARROLLO")
    || !datos.contenido_desarrollo.includes("SIN FIRMA NI VALIDEZ JURIDICA")
    || !referencia(datos.recibo_ref) || !referencia(datos.auditoria_ref)
    || !referencia(datos.evento_ref) || !instante(datos.confirmada_en)) falloContrato();
  return Object.freeze(Object.fromEntries(
    CAMPOS_RECIBO.map((campo) => [campo, datos[campo]]),
  ));
}
