const REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const CLAVE = /^[a-z][a-z0-9._-]{1,79}$/u;
const INSTANTE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.(\d{1,6}))?Z$/u;
const CAMPOS_SOLICITUD = Object.freeze([
  "expediente_ref", "solicitud_personal_ref", "version_actual_expediente_observada",
  "motivo_clave", "documentos_refs", "confirma_revision_personal", "confirma_ejercicio_sintetico",
]);
const CAMPOS_RECIBO = Object.freeze([
  "esquema", "expediente_ref", "solicitud_personal_ref", "relacion_ref", "recibo_ref",
  "actuacion_ref", "registrada_en", "periodo_incorporacion", "version_solicitud_personal",
  "version_actual_expediente", "seguimiento_ref", "version_seguimiento_anterior",
  "version_seguimiento_resultante", "auditoria_ref", "outbox_ref", "ejercicio_sintetico",
  "firma_oficial", "eficacia_administrativa",
]);

function fallo() { throw new TypeError("contrato de incorporación de ejercicio no válido"); }
function registro(valor, campos) {
  if (!valor || typeof valor !== "object" || Array.isArray(valor)
    || Object.getPrototypeOf(valor) !== Object.prototype || Object.getOwnPropertySymbols(valor).length
    || Object.keys(valor).length !== campos.length || !campos.every((campo) => Object.hasOwn(valor, campo))) fallo();
  return Object.fromEntries(campos.map((campo) => [campo, valor[campo]]));
}
function referencia(valor) { return typeof valor === "string" && REFERENCIA.test(valor); }
function clave(valor) { return typeof valor === "string" && CLAVE.test(valor); }
function version(valor, admiteCero = false) { return Number.isSafeInteger(valor) && valor >= (admiteCero ? 0 : 1); }
function lista(valor, validar) {
  if (!Array.isArray(valor) || valor.length > 32 || new Set(valor).size !== valor.length
    || !Array.from(valor).every(validar)) fallo();
  return Object.freeze([...valor]);
}
// Se conserva el texto recibido. La comparación rellena microsegundos, sin
// redondearlos a los milisegundos de Date ni cambiar la fecha del recibo.
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

export function validarSolicitudIncorporacionEjercicio(entrada) {
  const valor = registro(entrada, CAMPOS_SOLICITUD);
  if (!referencia(valor.expediente_ref) || !referencia(valor.solicitud_personal_ref)
    || !version(valor.version_actual_expediente_observada) || !clave(valor.motivo_clave)
    || valor.confirma_revision_personal !== true || valor.confirma_ejercicio_sintetico !== true) fallo();
  valor.documentos_refs = lista(valor.documentos_refs, referencia);
  return Object.freeze(valor);
}

function reciboPara(entrada, expedienteRef, versionActual) {
  const valor = registro(entrada, CAMPOS_RECIBO);
  if (valor.esquema !== "vec.contratacion-temporal.incorporacion-ejercicio.recibo.v2"
    || valor.expediente_ref !== expedienteRef || !referencia(expedienteRef)
    || !["solicitud_personal_ref", "relacion_ref", "recibo_ref", "actuacion_ref", "seguimiento_ref",
      "auditoria_ref", "outbox_ref"].every((campo) => referencia(valor[campo]))
    || !version(valor.version_solicitud_personal) || !version(valor.version_actual_expediente)
    || !version(versionActual) || valor.version_actual_expediente > versionActual
    || !version(valor.version_seguimiento_anterior, true) || !version(valor.version_seguimiento_resultante)
    || valor.version_seguimiento_resultante !== valor.version_seguimiento_anterior + 1
    || valor.ejercicio_sintetico !== true || valor.firma_oficial !== false || valor.eficacia_administrativa !== false) fallo();
  instanteComparable(valor.registrada_en);
  valor.periodo_incorporacion = periodo(valor.periodo_incorporacion);
  return Object.freeze(valor);
}

export function validarReciboIncorporacionEjercicio(entrada, solicitud) {
  const esperada = validarSolicitudIncorporacionEjercicio(solicitud);
  const valor = reciboPara(entrada, esperada.expediente_ref, esperada.version_actual_expediente_observada);
  if (valor.solicitud_personal_ref !== esperada.solicitud_personal_ref) fallo();
  return valor;
}

export function validarPreparacionIncorporacionEjercicio(entrada, expedienteRef) {
  const valor = registro(entrada, ["esquema", "expediente_ref", "version_actual_expediente", "preparacion", "recibo"]);
  if (valor.esquema !== "vec.contratacion-temporal.incorporacion-ejercicio.preparacion.v2"
    || valor.expediente_ref !== expedienteRef || !referencia(expedienteRef)
    || !version(valor.version_actual_expediente) || (valor.preparacion === null) === (valor.recibo === null)) fallo();
  if (valor.recibo !== null) {
    valor.recibo = reciboPara(valor.recibo, expedienteRef, valor.version_actual_expediente);
  } else {
    const p = registro(valor.preparacion, ["solicitud_personal_ref", "version_solicitud_personal",
      "version_seguimiento_esperada", "periodo_incorporacion", "motivos", "documentos_refs", "disponible"]);
    if (!referencia(p.solicitud_personal_ref) || !version(p.version_solicitud_personal)
      || !version(p.version_seguimiento_esperada, true) || typeof p.disponible !== "boolean") fallo();
    p.periodo_incorporacion = periodo(p.periodo_incorporacion);
    p.motivos = lista(p.motivos, clave);
    p.documentos_refs = lista(p.documentos_refs, referencia);
    if (p.disponible && p.motivos.length === 0) fallo();
    valor.preparacion = Object.freeze(p);
  }
  return Object.freeze(valor);
}
