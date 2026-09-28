const REF = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/iu;
const SHA256 = /^[0-9a-f]{64}$/iu;
const INSTANTE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$/u;
const CAMPOS_SOLICITUD = ["expediente_ref", "relacion_ref", "fecha_efectiva", "documento_ref", "documento_sha256", "version_esperada", "clave_idempotencia"];
const ref = (valor) => typeof valor === "string" && REF.test(valor);

function registro(valor) {
  return valor !== null && typeof valor === "object" && !Array.isArray(valor)
    && Object.getPrototypeOf(valor) === Object.prototype;
}

function fechaCivil(valor) {
  if (typeof valor !== "string" || !/^\d{4}-\d{2}-\d{2}$/u.test(valor)) return false;
  const [anio, mes, dia] = valor.split("-").map(Number);
  const fecha = new Date(0);
  fecha.setUTCFullYear(anio, mes - 1, dia);
  return anio >= 1 && fecha.getUTCFullYear() === anio
    && fecha.getUTCMonth() === mes - 1 && fecha.getUTCDate() === dia;
}

export function validarSolicitudReincorporacionRRHH(valor) {
  if (!registro(valor) || Object.keys(valor).length !== CAMPOS_SOLICITUD.length
    || !CAMPOS_SOLICITUD.every((campo) => Object.hasOwn(valor, campo))
    || !ref(valor.expediente_ref) || !ref(valor.relacion_ref)
    || !fechaCivil(valor.fecha_efectiva) || !ref(valor.documento_ref)
    || typeof valor.documento_sha256 !== "string" || !SHA256.test(valor.documento_sha256)
    || !Number.isSafeInteger(valor.version_esperada) || valor.version_esperada < 1
    || typeof valor.clave_idempotencia !== "string" || !UUID.test(valor.clave_idempotencia)) {
    throw new TypeError("solicitud de reincorporación RRHH no válida");
  }
  return Object.freeze(Object.fromEntries(CAMPOS_SOLICITUD.map((campo) => [campo, valor[campo]])));
}

export function validarReciboReincorporacionRRHH(valor, solicitud) {
  const entrada = validarSolicitudReincorporacionRRHH(solicitud);
  if (!registro(valor) || !ref(valor.recibo_ref) || !ref(valor.evento_ref)
    || typeof valor.registrada_en !== "string" || !INSTANTE.test(valor.registrada_en)
    || !Number.isFinite(Date.parse(valor.registrada_en))
    || valor.expediente_ref !== entrada.expediente_ref
    || valor.relacion_ref !== entrada.relacion_ref
    || valor.fecha_efectiva !== entrada.fecha_efectiva
    || !ref(valor.cese_evento_ref) || !ref(valor.cese_recibo_ref)
    || valor.estado_bolsa !== "pendiente_confirmacion") {
    throw new TypeError("recibo de reincorporación RRHH no verificable");
  }
  return Object.freeze({
    recibo_ref: valor.recibo_ref,
    evento_ref: valor.evento_ref,
    cese_evento_ref: valor.cese_evento_ref,
    cese_recibo_ref: valor.cese_recibo_ref,
    registrada_en: valor.registrada_en,
    expediente_ref: valor.expediente_ref,
    relacion_ref: valor.relacion_ref,
    fecha_efectiva: valor.fecha_efectiva,
    estado_bolsa: valor.estado_bolsa,
  });
}
