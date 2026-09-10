import { validarPreparacionIncorporacionEjercicio } from "./contrato-incorporacion-ejercicio.js";

const REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const CLAVE = /^[a-z][a-z0-9._-]{1,79}$/u;
const INSTANTE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.(\d{1,6}))?Z$/u;
const CAMPOS_SEGUIMIENTO = Object.freeze([
  "esquema", "alcance", "expediente_ref", "version_expediente", "recibo_incorporacion_ref",
  "seguimiento_ref", "version_seguimiento", "estado_clave", "periodo", "registrado_en",
  "actuaciones", "ejercicio_sintetico", "firma_oficial", "eficacia_administrativa",
]);
const CAMPOS_ACTUACION = Object.freeze([
  "actuacion_ref", "transicion_clave", "estado_origen", "estado_destino", "efectivo_en",
  "registrada_en", "documentos",
]);
const CAMPOS_DOCUMENTO = Object.freeze(["tipo_clave", "referencia"]);
const MAXIMO_ENVELOPE_BYTES = 512 * 1024;

function fallo() { throw new TypeError("contrato de seguimiento de incorporación no válido"); }
function registro(valor, campos) {
  if (!valor || typeof valor !== "object" || Array.isArray(valor)
    || Object.getPrototypeOf(valor) !== Object.prototype || Object.getOwnPropertySymbols(valor).length
    || Object.keys(valor).length !== campos.length || !campos.every((campo) => Object.hasOwn(valor, campo))) fallo();
  return Object.fromEntries(campos.map((campo) => [campo, valor[campo]]));
}
function referencia(valor) { return typeof valor === "string" && REFERENCIA.test(valor); }
function clave(valor) { return typeof valor === "string" && CLAVE.test(valor); }
function version(valor) { return Number.isSafeInteger(valor) && valor >= 1; }
function envelope(valor) {
  let texto;
  try { texto = JSON.stringify(valor); } catch { fallo(); }
  if (typeof texto !== "string" || new TextEncoder().encode(texto).byteLength > MAXIMO_ENVELOPE_BYTES) fallo();
}
function instanteComparable(valor) {
  const partes = typeof valor === "string" && INSTANTE.exec(valor);
  if (!partes || !Number.isFinite(Date.parse(valor))
    || new Date(valor).toISOString().slice(0, 19) !== valor.slice(0, 19)) fallo();
  const normalizado = `${valor.slice(0, 19)}.${(partes[1] ?? "").padEnd(6, "0")}Z`;
  if (normalizado.startsWith("0000-") || normalizado === "0001-01-01T00:00:00.000000Z") fallo();
  return normalizado;
}
function periodo(entrada) {
  const valor = registro(entrada, ["desde", "hasta"]);
  if (instanteComparable(valor.hasta) <= instanteComparable(valor.desde)) fallo();
  return Object.freeze(valor);
}
function documento(entrada) {
  const valor = registro(entrada, CAMPOS_DOCUMENTO);
  if (!clave(valor.tipo_clave) || !referencia(valor.referencia)) fallo();
  return Object.freeze(valor);
}
function actuacion(entrada) {
  const valor = registro(entrada, CAMPOS_ACTUACION);
  if (!referencia(valor.actuacion_ref) || !["transicion_clave", "estado_origen", "estado_destino"].every(
    (campo) => clave(valor[campo]))) fallo();
  instanteComparable(valor.efectivo_en);
  instanteComparable(valor.registrada_en);
  if (!Array.isArray(valor.documentos) || valor.documentos.length > 32) fallo();
  valor.documentos = Object.freeze(valor.documentos.map(documento));
  if (new Set(valor.documentos.map(({ referencia: ref }) => ref)).size !== valor.documentos.length) fallo();
  return Object.freeze(valor);
}

// Reutiliza el validador V2 del recibo: este contrato no duplica su esquema.
function reciboV2Ligado(recibo, expedienteRef) {
  return validarPreparacionIncorporacionEjercicio({
    esquema: "vec.contratacion-temporal.incorporacion-ejercicio.preparacion.v2",
    expediente_ref: expedienteRef,
    version_actual_expediente: recibo?.version_actual_expediente,
    preparacion: null,
    recibo,
  }, expedienteRef).recibo;
}

export function validarConsultaSeguimientoIncorporacion(entrada, expedienteRef) {
  const valor = registro(entrada, CAMPOS_SEGUIMIENTO);
  envelope(valor);
  if (valor.esquema !== "vec.contratacion-temporal.seguimiento-incorporacion.v2"
    || valor.alcance !== "original_incorporacion" || valor.expediente_ref !== expedienteRef
    || !referencia(expedienteRef) || !referencia(valor.recibo_incorporacion_ref)
    || !referencia(valor.seguimiento_ref)
    || !version(valor.version_expediente) || !version(valor.version_seguimiento) || !clave(valor.estado_clave)
    || valor.ejercicio_sintetico !== true || valor.firma_oficial !== false
    || valor.eficacia_administrativa !== false) fallo();
  valor.periodo = periodo(valor.periodo);
  instanteComparable(valor.registrado_en);
  if (!Array.isArray(valor.actuaciones) || valor.actuaciones.length === 0 || valor.actuaciones.length > 512) fallo();
  valor.actuaciones = Object.freeze(valor.actuaciones.map(actuacion));
  if (new Set(valor.actuaciones.map(({ actuacion_ref: ref }) => ref)).size !== valor.actuaciones.length) fallo();
  const hito = valor.actuaciones.at(-1);
  if (hito.transicion_clave !== "confirmar_incorporacion" || hito.estado_destino !== valor.estado_clave
    || instanteComparable(hito.efectivo_en) !== instanteComparable(valor.periodo.desde)
    || instanteComparable(hito.registrada_en) !== instanteComparable(valor.registrado_en)) fallo();
  return Object.freeze(valor);
}

export function validarSeguimientoIncorporacion(entrada, reciboEntrada) {
  const recibo = reciboV2Ligado(reciboEntrada, reciboEntrada?.expediente_ref);
  const valor = validarConsultaSeguimientoIncorporacion(entrada, recibo.expediente_ref);
  const hito = valor.actuaciones.at(-1);
  if (valor.recibo_incorporacion_ref !== recibo.recibo_ref || valor.seguimiento_ref !== recibo.seguimiento_ref
    || valor.version_expediente !== recibo.version_actual_expediente
    || valor.version_seguimiento !== recibo.version_seguimiento_resultante
    || valor.periodo.desde !== recibo.periodo_incorporacion.desde || valor.periodo.hasta !== recibo.periodo_incorporacion.hasta
    || instanteComparable(valor.registrado_en) !== instanteComparable(recibo.registrada_en)
    || hito.actuacion_ref !== recibo.actuacion_ref || instanteComparable(hito.registrada_en) !== instanteComparable(recibo.registrada_en)) fallo();
  return valor;
}
