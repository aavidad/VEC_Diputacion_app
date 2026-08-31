/** Contrato neutral para texto JSON canónico de asignación. */

const {
  create: CREAR_REGISTRO,
  freeze: CONGELAR,
  getPrototypeOf: OBTENER_PROTOTIPO,
  hasOwn: TIENE_PROPIA,
} = Object;
const PROTOTIPO_OBJETO = Object.prototype;
const { isArray: ES_ARRAY } = Array;
const {
  isSafeInteger: ES_ENTERO_SEGURO,
  MAX_SAFE_INTEGER: MAXIMO_ENTERO_SEGURO,
} = Number;
const { parse: ANALIZAR_JSON, stringify: SERIALIZAR_JSON } = JSON;
const { apply: APLICAR } = Reflect;
const EJECUTAR_EXPRESION = RegExp.prototype.exec;
const PROBAR_EXPRESION = RegExp.prototype.test;
const CODIGO_UNIDAD = String.prototype.charCodeAt;
const NORMALIZAR = String.prototype.normalize;
const RECORTAR = String.prototype.trim;
const ERROR_TIPO = TypeError;

const MAXIMO_BYTES_JSON = 8192;
const PATRON_REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const PATRON_CLAVE = /^[a-z][a-z0-9._-]{1,79}$/u;
const PATRON_UUID_V4 =
  /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/u;
const PATRON_INSTANTE_UTC =
  /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.(\d{1,6}))?Z$/u;
const UUID_V4_NULO = "00000000-0000-4000-8000-000000000000";
const ESQUEMA_RECIBO =
  "vec.contratacion-temporal.recibo-asignacion.v1";

const CAMPOS_ASIGNACION = CONGELAR([
  "expediente_ref",
  "version_esperada",
  "clave_idempotencia",
  "unidad_ref",
  "responsable_ref",
]);
const CAMPOS_REASIGNACION = CONGELAR([
  "expediente_ref",
  "version_esperada",
  "clave_idempotencia",
  "unidad_ref",
  "responsable_ref",
  "motivo_reasignacion_clave",
  "observaciones",
]);
const CAMPOS_RECIBO = CONGELAR([
  "esquema",
  "operacion",
  "expediente_ref",
  "version_resultante",
  "recibo_ref",
  "confirmada_en",
]);

function falloContrato() {
  throw new ERROR_TIPO("contrato JSON no válido");
}

function excedeLimiteUTF8(texto) {
  let bytes = 0;
  for (let indice = 0; indice < texto.length; indice += 1) {
    const unidad = APLICAR(CODIGO_UNIDAD, texto, [indice]);
    if (unidad <= 0x7F) {
      bytes += 1;
    } else if (unidad <= 0x7FF) {
      bytes += 2;
    } else if (unidad >= 0xD800 && unidad <= 0xDBFF
      && indice + 1 < texto.length) {
      const siguiente = APLICAR(CODIGO_UNIDAD, texto, [indice + 1]);
      if (siguiente >= 0xDC00 && siguiente <= 0xDFFF) {
        bytes += 4;
        indice += 1;
      } else {
        bytes += 3;
      }
    } else {
      bytes += 3;
    }
    if (bytes > MAXIMO_BYTES_JSON) return true;
  }
  return false;
}

function esPrimitivoJSON(valor) {
  const tipo = typeof valor;
  return valor === null
    || tipo === "string"
    || tipo === "number"
    || tipo === "boolean";
}

function analizarRegistroCanonico(texto, campos) {
  if (typeof texto !== "string" || excedeLimiteUTF8(texto)) {
    return falloContrato();
  }

  let raiz;
  try {
    raiz = ANALIZAR_JSON(texto);
  } catch {
    return falloContrato();
  }
  if (raiz === null
    || typeof raiz !== "object"
    || ES_ARRAY(raiz)
    || OBTENER_PROTOTIPO(raiz) !== PROTOTIPO_OBJETO) {
    return falloContrato();
  }

  const registro = CREAR_REGISTRO(null);
  for (let indice = 0; indice < campos.length; indice += 1) {
    const campo = campos[indice];
    if (!TIENE_PROPIA(raiz, campo)) return falloContrato();
    const valor = raiz[campo];
    if (!esPrimitivoJSON(valor)) return falloContrato();
    registro[campo] = valor;
  }
  if (SERIALIZAR_JSON(registro) !== texto) return falloContrato();
  return registro;
}

function coincide(patron, valor) {
  return APLICAR(PROBAR_EXPRESION, patron, [valor]);
}

function referenciaValida(valor) {
  return typeof valor === "string" && coincide(PATRON_REFERENCIA, valor);
}

function versionEsperadaValida(valor) {
  return ES_ENTERO_SEGURO(valor)
    && valor >= 1
    && valor < MAXIMO_ENTERO_SEGURO;
}

function claveIdempotenciaValida(valor) {
  return typeof valor === "string"
    && coincide(PATRON_UUID_V4, valor)
    && valor !== UUID_V4_NULO;
}

function observacionesValidas(valor) {
  if (typeof valor !== "string"
    || valor === ""
    || APLICAR(RECORTAR, valor, []) !== valor
    || APLICAR(NORMALIZAR, valor, ["NFC"]) !== valor) {
    return false;
  }

  let puntos = 0;
  for (let indice = 0; indice < valor.length; indice += 1) {
    const unidad = APLICAR(CODIGO_UNIDAD, valor, [indice]);
    if (unidad >= 0xD800 && unidad <= 0xDBFF) {
      if (indice + 1 >= valor.length) return false;
      const siguiente = APLICAR(CODIGO_UNIDAD, valor, [indice + 1]);
      if (siguiente < 0xDC00 || siguiente > 0xDFFF) return false;
      indice += 1;
    } else if (unidad >= 0xDC00 && unidad <= 0xDFFF) {
      return false;
    } else if ((unidad < 32 && unidad !== 9 && unidad !== 10)
      || (unidad >= 127 && unidad <= 159)) {
      return false;
    }
    puntos += 1;
    if (puntos > 1000) return false;
  }
  return true;
}

function validarDatosSolicitud(datos, reasignacion) {
  if (!referenciaValida(datos.expediente_ref)
    || !versionEsperadaValida(datos.version_esperada)
    || !claveIdempotenciaValida(datos.clave_idempotencia)
    || !referenciaValida(datos.unidad_ref)
    || !referenciaValida(datos.responsable_ref)
    || (reasignacion && (
      typeof datos.motivo_reasignacion_clave !== "string"
      || !coincide(PATRON_CLAVE, datos.motivo_reasignacion_clave)
      || !observacionesValidas(datos.observaciones)
    ))) {
    return falloContrato();
  }
}

function diasDelMes(anio, mes) {
  if (mes === 2) {
    return anio % 4 === 0 && (anio % 100 !== 0 || anio % 400 === 0)
      ? 29
      : 28;
  }
  return mes === 4 || mes === 6 || mes === 9 || mes === 11 ? 30 : 31;
}

function instanteUTCValido(valor) {
  if (typeof valor !== "string") return false;
  const partes = APLICAR(EJECUTAR_EXPRESION, PATRON_INSTANTE_UTC, [valor]);
  if (partes === null) return false;
  const anio = +partes[1];
  const mes = +partes[2];
  const dia = +partes[3];
  const fraccion = partes[7];
  return anio >= 1 && mes >= 1 && mes <= 12
    && dia >= 1 && dia <= diasDelMes(anio, mes)
    && +partes[4] <= 23
    && +partes[5] <= 59
    && +partes[6] <= 59
    && (fraccion === undefined || fraccion[fraccion.length - 1] !== "0");
}

export function validarSolicitudAsignacion(texto) {
  const datos = analizarRegistroCanonico(texto, CAMPOS_ASIGNACION);
  validarDatosSolicitud(datos, false);
  return CONGELAR({
    expediente_ref: datos.expediente_ref,
    version_esperada: datos.version_esperada,
    clave_idempotencia: datos.clave_idempotencia,
    unidad_ref: datos.unidad_ref,
    responsable_ref: datos.responsable_ref,
  });
}

export function validarSolicitudReasignacion(texto) {
  const datos = analizarRegistroCanonico(texto, CAMPOS_REASIGNACION);
  validarDatosSolicitud(datos, true);
  return CONGELAR({
    expediente_ref: datos.expediente_ref,
    version_esperada: datos.version_esperada,
    clave_idempotencia: datos.clave_idempotencia,
    unidad_ref: datos.unidad_ref,
    responsable_ref: datos.responsable_ref,
    motivo_reasignacion_clave: datos.motivo_reasignacion_clave,
    observaciones: datos.observaciones,
  });
}

export function validarReciboAsignacion(texto) {
  const datos = analizarRegistroCanonico(texto, CAMPOS_RECIBO);
  if (datos.esquema !== ESQUEMA_RECIBO
    || (datos.operacion !== "asignar" && datos.operacion !== "reasignar")
    || !referenciaValida(datos.expediente_ref)
    || !ES_ENTERO_SEGURO(datos.version_resultante)
    || datos.version_resultante < 2
    || datos.version_resultante > MAXIMO_ENTERO_SEGURO
    || !referenciaValida(datos.recibo_ref)
    || !instanteUTCValido(datos.confirmada_en)) {
    return falloContrato();
  }
  return CONGELAR({
    esquema: datos.esquema,
    operacion: datos.operacion,
    expediente_ref: datos.expediente_ref,
    version_resultante: datos.version_resultante,
    recibo_ref: datos.recibo_ref,
    confirmada_en: datos.confirmada_en,
  });
}
