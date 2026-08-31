/** Contrato neutral y cerrado para asignar y reasignar expedientes. */

const MAXIMO_ENTERO_SEGURO = Number.MAX_SAFE_INTEGER;
const PATRON_REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const PATRON_CLAVE = /^[a-z][a-z0-9._-]{1,79}$/u;
const PATRON_UUID_V4 =
  /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/u;
const UUID_V4_NULO = "00000000-0000-4000-8000-000000000000";
const ESQUEMA_RECIBO =
  "vec.contratacion-temporal.recibo-asignacion.v1";

const CAMPOS_ASIGNACION = Object.freeze([
  "expediente_ref",
  "version_esperada",
  "clave_idempotencia",
  "unidad_ref",
  "responsable_ref",
]);
const CAMPOS_REASIGNACION = Object.freeze([
  ...CAMPOS_ASIGNACION,
  "motivo_reasignacion_clave",
  "observaciones",
]);
const CAMPOS_RECIBO = Object.freeze([
  "esquema",
  "operacion",
  "expediente_ref",
  "version_resultante",
  "recibo_ref",
  "confirmada_en",
]);

function falloContrato(nombre) {
  throw new TypeError(`${nombre} no respeta el contrato cerrado`);
}

function extraerRegistro(valor, campos, nombre) {
  if (valor === null || typeof valor !== "object" || Array.isArray(valor)) {
    return falloContrato(nombre);
  }
  let prototipo;
  let claves;
  let descriptores;
  try {
    prototipo = Object.getPrototypeOf(valor);
    claves = Reflect.ownKeys(valor);
    descriptores = Object.getOwnPropertyDescriptors(valor);
  } catch {
    return falloContrato(nombre);
  }
  if (prototipo !== Object.prototype || claves.length !== campos.length
    || claves.some((clave) => typeof clave !== "string" || !campos.includes(clave))) {
    return falloContrato(nombre);
  }

  const datos = {};
  for (const campo of campos) {
    const descriptor = descriptores[campo];
    if (descriptor === undefined
      || !Object.hasOwn(descriptor, "value")
      || descriptor.enumerable !== true) {
      return falloContrato(nombre);
    }
    datos[campo] = descriptor.value;
  }
  return datos;
}

function referenciaValida(valor) {
  return typeof valor === "string" && PATRON_REFERENCIA.test(valor);
}

function versionEsperadaValida(valor) {
  return Number.isSafeInteger(valor)
    && valor >= 1
    && valor < MAXIMO_ENTERO_SEGURO;
}

function claveIdempotenciaValida(valor) {
  return typeof valor === "string"
    && PATRON_UUID_V4.test(valor)
    && valor !== UUID_V4_NULO;
}

function observacionesValidas(valor) {
  if (typeof valor !== "string" || valor === "" || valor !== valor.trim()
    || valor.normalize("NFC") !== valor) {
    return false;
  }
  let longitud = 0;
  for (const caracter of valor) {
    longitud += 1;
    if (longitud > 1000) return false;
    const codigo = caracter.codePointAt(0);
    if ((codigo < 32 || (codigo >= 127 && codigo <= 159))
      && caracter !== "\n" && caracter !== "\t"
      || (codigo >= 0xD800 && codigo <= 0xDFFF)) {
      return false;
    }
  }
  return true;
}

function validarSolicitud(solicitud, reasignacion) {
  const nombre = reasignacion ? "solicitud de reasignación" : "solicitud de asignación";
  const datos = extraerRegistro(
    solicitud,
    reasignacion ? CAMPOS_REASIGNACION : CAMPOS_ASIGNACION,
    nombre,
  );
  if (!referenciaValida(datos.expediente_ref)
    || !versionEsperadaValida(datos.version_esperada)
    || !claveIdempotenciaValida(datos.clave_idempotencia)
    || !referenciaValida(datos.unidad_ref)
    || !referenciaValida(datos.responsable_ref)
    || reasignacion && (
      typeof datos.motivo_reasignacion_clave !== "string"
      || !PATRON_CLAVE.test(datos.motivo_reasignacion_clave)
      || !observacionesValidas(datos.observaciones)
    )) {
    throw new TypeError(`${nombre} no válida`);
  }
  return Object.freeze({ ...datos });
}

function diasDelMes(anio, mes) {
  if (mes === 2) {
    return anio % 4 === 0 && (anio % 100 !== 0 || anio % 400 === 0)
      ? 29
      : 28;
  }
  return [4, 6, 9, 11].includes(mes) ? 30 : 31;
}

function instanteUTCValido(valor) {
  if (typeof valor !== "string") return false;
  const partes = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.(\d{1,6}))?Z$/u
    .exec(valor);
  if (partes === null) return false;
  const [, anioTexto, mesTexto, diaTexto, horaTexto, minutoTexto,
    segundoTexto, fraccion] = partes;
  const anio = Number(anioTexto);
  const mes = Number(mesTexto);
  const dia = Number(diaTexto);
  return anio >= 1 && mes >= 1 && mes <= 12
    && dia >= 1 && dia <= diasDelMes(anio, mes)
    && Number(horaTexto) <= 23
    && Number(minutoTexto) <= 59
    && Number(segundoTexto) <= 59
    && (fraccion === undefined || !fraccion.endsWith("0"));
}

export function validarSolicitudAsignacion(solicitud) {
  return validarSolicitud(solicitud, false);
}

export function validarSolicitudReasignacion(solicitud) {
  return validarSolicitud(solicitud, true);
}

export function validarReciboAsignacion(recibo) {
  const datos = extraerRegistro(recibo, CAMPOS_RECIBO, "recibo de asignación");
  if (datos.esquema !== ESQUEMA_RECIBO
    || datos.operacion !== "asignar" && datos.operacion !== "reasignar"
    || !referenciaValida(datos.expediente_ref)
    || !Number.isSafeInteger(datos.version_resultante)
    || datos.version_resultante < 2
    || datos.version_resultante > MAXIMO_ENTERO_SEGURO
    || !referenciaValida(datos.recibo_ref)
    || !instanteUTCValido(datos.confirmada_en)) {
    throw new TypeError("recibo de asignación no válido");
  }
  return Object.freeze({ ...datos });
}
