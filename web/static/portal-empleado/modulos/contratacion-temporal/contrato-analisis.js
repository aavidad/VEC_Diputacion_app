/** Contrato neutral y cerrado para registrar y rectificar análisis RRHH. */

const MAXIMO_ENTERO_SEGURO = Number.MAX_SAFE_INTEGER;
const MAXIMO_ANIOS_PERIODO = 100;
const PATRON_REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const PATRON_CLAVE = /^[a-z][a-z0-9._-]{1,79}$/u;
const PATRON_GRUPO = /^[A-Z][A-Z0-9/+.-]{0,19}$/u;
const PATRON_UUID_V4 =
  /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/u;
const UUID_V4_NULO = "00000000-0000-4000-8000-000000000000";
const PATRON_HUELLA = /^[0-9a-f]{64}$/u;
const ESQUEMA_RECIBO =
  "vec.contratacion-temporal.recibo-analisis-rrhh.v1";
const OPERACIONES = new Set(["registrar", "rectificar"]);

function esRegistro(valor) {
  if (valor === null || typeof valor !== "object" || Array.isArray(valor)) {
    return false;
  }
  try {
    if (Object.getPrototypeOf(valor) !== Object.prototype
      || Object.getOwnPropertySymbols(valor).length !== 0) return false;
    return Object.values(Object.getOwnPropertyDescriptors(valor)).every(
      (descriptor) => Object.hasOwn(descriptor, "value")
        && descriptor.enumerable === true,
    );
  } catch {
    return false;
  }
}

function exigirCamposExactos(valor, campos, nombre) {
  if (!esRegistro(valor)) {
    throw new TypeError(`${nombre} no respeta el contrato cerrado`);
  }
  const recibidos = Object.keys(valor);
  if (recibidos.length !== campos.length
    || !recibidos.every((campo) => campos.includes(campo))
    || !campos.every((campo) => Object.hasOwn(valor, campo))) {
    throw new TypeError(`${nombre} no respeta el contrato cerrado`);
  }
}

function referenciaValida(valor) {
  return typeof valor === "string" && PATRON_REFERENCIA.test(valor);
}

function claveValida(valor) {
  return typeof valor === "string" && PATRON_CLAVE.test(valor);
}

function huellaValida(valor) {
  return typeof valor === "string"
    && PATRON_HUELLA.test(valor)
    && !/^0{64}$/u.test(valor);
}

function versionConIncrementoValida(valor) {
  return Number.isSafeInteger(valor)
    && valor >= 1
    && valor < MAXIMO_ENTERO_SEGURO;
}

function diasDelMes(anio, mes) {
  if (mes === 2) {
    return anio % 4 === 0 && (anio % 100 !== 0 || anio % 400 === 0)
      ? 29
      : 28;
  }
  return [4, 6, 9, 11].includes(mes) ? 30 : 31;
}

function descomponerFechaCivil(valor) {
  if (typeof valor !== "string") return null;
  const partes = /^(\d{4})-(\d{2})-(\d{2})T00:00:00Z$/u.exec(valor);
  if (!partes) return null;
  const anio = Number(partes[1]);
  const mes = Number(partes[2]);
  const dia = Number(partes[3]);
  if (anio < 1 || mes < 1 || mes > 12
    || dia < 1 || dia > diasDelMes(anio, mes)) return null;
  return { anio, mes, dia };
}

function ordinalFecha({ anio, mes, dia }) {
  return anio * 10_000 + mes * 100 + dia;
}

function periodoValido(periodo) {
  exigirCamposExactos(periodo, ["inicio", "fin"], "periodo del análisis");
  const inicio = descomponerFechaCivil(periodo.inicio);
  const fin = descomponerFechaCivil(periodo.fin);
  if (inicio === null || fin === null
    || ordinalFecha(fin) < ordinalFecha(inicio)) return false;

  const anioLimite = inicio.anio + MAXIMO_ANIOS_PERIODO;
  if (anioLimite > 9_999) return true;
  let mesLimite = inicio.mes;
  let diaLimite = inicio.dia;
  if (mesLimite === 2 && diaLimite === 29
    && diasDelMes(anioLimite, mesLimite) === 28) {
    mesLimite = 3;
    diaLimite = 1;
  }
  return ordinalFecha(fin) <= ordinalFecha({
    anio: anioLimite,
    mes: mesLimite,
    dia: diaLimite,
  });
}

function instanteUTCValido(valor) {
  if (typeof valor !== "string") return false;
  const partes =
    /^(\d{4}-\d{2}-\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.\d{1,6})?Z$/u
      .exec(valor);
  if (!partes || partes[1].startsWith("0000-")) return false;
  const instante = new Date(valor);
  return Number.isFinite(instante.valueOf())
    && instante.toISOString().slice(0, 10) === partes[1]
    && Number(partes[2]) <= 23
    && Number(partes[3]) <= 59
    && Number(partes[4]) <= 59;
}

function validarAnalisis(analisis) {
  exigirCamposExactos(analisis, [
    "modalidad_clave",
    "categoria_ref",
    "grupo_subgrupo",
    "causa_clave",
    "periodo",
    "porcentaje_jornada",
    "entrada_rc",
  ], "datos funcionales del análisis");
  exigirCamposExactos(
    analisis.entrada_rc,
    ["referencia", "huella_sha256"],
    "entrada RC del análisis",
  );
  if (!claveValida(analisis.modalidad_clave)
    || !referenciaValida(analisis.categoria_ref)
    || typeof analisis.grupo_subgrupo !== "string"
    || !PATRON_GRUPO.test(analisis.grupo_subgrupo)
    || !claveValida(analisis.causa_clave)
    || !periodoValido(analisis.periodo)
    || !Number.isSafeInteger(analisis.porcentaje_jornada)
    || analisis.porcentaje_jornada < 1
    || analisis.porcentaje_jornada > 10_000
    || !referenciaValida(analisis.entrada_rc.referencia)
    || !huellaValida(analisis.entrada_rc.huella_sha256)) {
    throw new TypeError("datos funcionales del análisis no válidos");
  }
  return {
    modalidad_clave: analisis.modalidad_clave,
    categoria_ref: analisis.categoria_ref,
    grupo_subgrupo: analisis.grupo_subgrupo,
    causa_clave: analisis.causa_clave,
    periodo: Object.freeze({
      inicio: analisis.periodo.inicio,
      fin: analisis.periodo.fin,
    }),
    porcentaje_jornada: analisis.porcentaje_jornada,
    entrada_rc: Object.freeze({
      referencia: analisis.entrada_rc.referencia,
      huella_sha256: analisis.entrada_rc.huella_sha256,
    }),
  };
}

function validarSolicitud(solicitud, rectificacion) {
  const nombre = rectificacion
    ? "rectificación del análisis"
    : "registro del análisis";
  const campos = [
    "expediente_ref",
    "version_esperada",
    "clave_idempotencia",
    "artefacto_ref",
    "analisis",
  ];
  if (rectificacion) campos.push("motivo_rectificacion_clave");
  exigirCamposExactos(solicitud, campos, nombre);
  if (!referenciaValida(solicitud.expediente_ref)
    || !versionConIncrementoValida(solicitud.version_esperada)
    || typeof solicitud.clave_idempotencia !== "string"
    || !PATRON_UUID_V4.test(solicitud.clave_idempotencia)
    || solicitud.clave_idempotencia === UUID_V4_NULO
    || !referenciaValida(solicitud.artefacto_ref)
    || rectificacion && !claveValida(
      solicitud.motivo_rectificacion_clave,
    )) {
    throw new TypeError(`${nombre} no válido`);
  }
  const salida = {
    expediente_ref: solicitud.expediente_ref,
    version_esperada: solicitud.version_esperada,
    clave_idempotencia: solicitud.clave_idempotencia,
    artefacto_ref: solicitud.artefacto_ref,
    analisis: Object.freeze(validarAnalisis(solicitud.analisis)),
  };
  if (rectificacion) {
    salida.motivo_rectificacion_clave =
      solicitud.motivo_rectificacion_clave;
  }
  return Object.freeze(salida);
}

export function validarSolicitudRegistroAnalisis(solicitud) {
  return validarSolicitud(solicitud, false);
}

export function validarSolicitudRectificacionAnalisis(solicitud) {
  return validarSolicitud(solicitud, true);
}

export function validarReciboAnalisis(recibo) {
  exigirCamposExactos(recibo, [
    "esquema",
    "operacion",
    "expediente_ref",
    "version_resultante",
    "recibo_ref",
    "confirmada_en",
  ], "recibo del análisis");
  if (recibo.esquema !== ESQUEMA_RECIBO
    || !OPERACIONES.has(recibo.operacion)
    || !referenciaValida(recibo.expediente_ref)
    || !Number.isSafeInteger(recibo.version_resultante)
    || recibo.version_resultante < 2
    || recibo.version_resultante > MAXIMO_ENTERO_SEGURO
    || !referenciaValida(recibo.recibo_ref)
    || !instanteUTCValido(recibo.confirmada_en)) {
    throw new TypeError("recibo del análisis no válido");
  }
  return Object.freeze({
    esquema: recibo.esquema,
    operacion: recibo.operacion,
    expediente_ref: recibo.expediente_ref,
    version_resultante: recibo.version_resultante,
    recibo_ref: recibo.recibo_ref,
    confirmada_en: recibo.confirmada_en,
  });
}
