/**
 * Contratos cerrados del corte web de propuestas de llamamiento.
 *
 * La confirmación real es deliberadamente compacta: acredita qué versiones
 * evaluó el servidor, pero nunca transporta personas, evaluaciones ni detalle
 * reutilizable. El contrato de presentación vive separado y siempre conserva
 * su marca inequívoca de demostración.
 */

const ESQUEMA_CONFIRMACION = "vec.bolsa.propuesta-llamamiento.confirmacion.v1";
const ESQUEMA_PRESENTACION = "vec.bolsa.propuesta-llamamiento.presentacion.v1";
const MAXIMO_UINT64 = 18_446_744_073_709_551_615n;
const MAXIMO_PARTICIPACIONES = 250_000n;

const CAMPOS_CONFIRMACION = Object.freeze([
  "esquema", "propuesta_ref", "huella_propuesta_sha256", "bolsa", "necesidad",
  "instantanea", "politica", "instante_referencia", "instantanea_generada_en",
  "total_participaciones_instantanea", "total_evaluaciones", "orden_seleccionado",
  "generada_en",
]);
const CAMPOS_VERSION = Object.freeze(["referencia", "version", "huella_sha256"]);
const CAMPOS_PRESENTACION = Object.freeze([
  "esquema", "demostracion", "id", "necesidad_id", "estado", "version_bolsa",
  "version_regla", "fecha_corte", "personas_incluidas", "evaluaciones",
]);
const CAMPOS_EVALUACION_PRESENTACION = Object.freeze(["orden", "resultado", "motivos"]);
const CAMPOS_MOTIVO_PRESENTACION = Object.freeze(["regla", "fundamento"]);

function esObjeto(valor) {
  return valor !== null && typeof valor === "object" && !Array.isArray(valor);
}

function exigirCamposExactos(objeto, campos, nombre) {
  if (!esObjeto(objeto) || Object.keys(objeto).length !== campos.length
    || campos.some((campo) => !Object.hasOwn(objeto, campo))) {
    throw new Error(`${nombre} no respeta el contrato cerrado`);
  }
}

function exigirCadena(valor, nombre, maximo = 512) {
  if (typeof valor !== "string" || valor === "" || new TextEncoder().encode(valor).length > maximo
    || valor !== valor.trim() || valor.normalize("NFC") !== valor) {
    throw new Error(`${nombre} no válido`);
  }
  return valor;
}

export function validarReferenciaOpacaLlamamiento(valor, nombre = "referencia") {
  const referencia = exigirCadena(valor, nombre);
  const contieneControlOBidi = /[\u0000-\u001f\u007f-\u009f\u061c\u200e\u200f\u2028-\u202e\u2066-\u2069\ufffd]/u;
  const pareceDocumento = /(?:[0-9][._:/#-]?){8}[a-z]|[xyz][._:/#-]?(?:[0-9][._:/#-]?){7}[a-z]/iu;
  const etiquetaPersonal = /(?:^|[._:/#-])(?:dni|nie|nif|pasaporte|passport)(?:[._:/#-]|$)/iu;
  if (referencia.includes("*") || contieneControlOBidi.test(referencia)
    || pareceDocumento.test(referencia) || etiquetaPersonal.test(referencia)) {
    throw new Error(`${nombre} no válida`);
  }
  return referencia;
}

function exigirReferenciaConfirmacion(valor, nombre) {
  const referencia = validarReferenciaOpacaLlamamiento(valor, nombre);
  if (!/^[A-Za-z0-9][A-Za-z0-9._:/#-]*$/.test(referencia)) throw new Error(`${nombre} no válida`);
  return referencia;
}

function exigirHuella(valor, nombre) {
  if (typeof valor !== "string" || !/^[a-f0-9]{64}$/.test(valor)) {
    throw new Error(`${nombre} no válida`);
  }
  return valor;
}

function exigirDecimal(valor, nombre, { admiteCero = false, maximo = MAXIMO_UINT64 } = {}) {
  if (typeof valor !== "string" || !/^(?:0|[1-9][0-9]*)$/.test(valor)) {
    throw new Error(`${nombre} no válido`);
  }
  const numero = BigInt(valor);
  if ((!admiteCero && numero === 0n) || numero > maximo) throw new Error(`${nombre} no válido`);
  return valor;
}

function exigirInstanteUTC(valor, nombre) {
  const cadena = exigirCadena(valor, nombre, 32);
  const partes = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.(\d{1,6}))?Z$/.exec(cadena);
  if (!partes || partes[1] === "0000" || partes[7]?.endsWith("0")) throw new Error(`${nombre} no válido`);
  const [, anio, mes, dia, hora, minuto, segundo] = partes;
  const fecha = new Date(Date.parse(cadena));
  if (!Number.isFinite(fecha.getTime()) || fecha.getUTCFullYear() !== Number(anio)
    || fecha.getUTCMonth() + 1 !== Number(mes) || fecha.getUTCDate() !== Number(dia)
    || fecha.getUTCHours() !== Number(hora) || fecha.getUTCMinutes() !== Number(minuto)
    || fecha.getUTCSeconds() !== Number(segundo)) {
    throw new Error(`${nombre} no válido`);
  }
  return cadena;
}

function claveOrdenInstanteUTC(valor) {
  const partes = /^(.*?)(?:\.(\d{1,6}))?Z$/.exec(valor);
  return `${partes[1]}.${String(partes[2] || "").padEnd(6, "0")}`;
}

function validarVersionHuella(valor, nombre) {
  exigirCamposExactos(valor, CAMPOS_VERSION, nombre);
  return {
    referencia: exigirReferenciaConfirmacion(valor.referencia, `referencia de ${nombre}`),
    version: exigirDecimal(valor.version, `versión de ${nombre}`),
    huella_sha256: exigirHuella(valor.huella_sha256, `huella de ${nombre}`),
  };
}

function exigirETagConfirmacion(valor, huella) {
  const esperado = `"vec-propuesta-llamamiento-v1.sha256-${huella}"`;
  if (valor !== esperado) throw new Error("ETag de confirmación no válido");
  return valor;
}

export function extraerDatosEnvelopeLlamamiento(envelope) {
  exigirCamposExactos(envelope, ["data"], "envelope de confirmación");
  if (!esObjeto(envelope.data)) throw new Error("datos de confirmación no válidos");
  return envelope.data;
}

export function validarConfirmacionPropuestaLlamamiento(datos, etag) {
  exigirCamposExactos(datos, CAMPOS_CONFIRMACION, "confirmación de propuesta");
  if (datos.esquema !== ESQUEMA_CONFIRMACION) {
    throw new Error("versión de confirmación de propuesta no compatible");
  }

  const huellaPropuesta = exigirHuella(datos.huella_propuesta_sha256, "huella de propuesta");
  const bolsa = validarVersionHuella(datos.bolsa, "bolsa");
  const necesidad = validarVersionHuella(datos.necesidad, "necesidad");
  const instantanea = validarVersionHuella(datos.instantanea, "instantánea");
  const politica = validarVersionHuella(datos.politica, "política");
  const instanteReferencia = exigirInstanteUTC(datos.instante_referencia, "instante de referencia");
  const instantaneaGeneradaEn = exigirInstanteUTC(datos.instantanea_generada_en, "generación de instantánea");
  const generadaEn = exigirInstanteUTC(datos.generada_en, "generación de propuesta");
  const totalParticipaciones = exigirDecimal(
    datos.total_participaciones_instantanea,
    "total de participaciones",
    { maximo: MAXIMO_PARTICIPACIONES },
  );
  const totalEvaluaciones = exigirDecimal(
    datos.total_evaluaciones,
    "total de evaluaciones",
    { maximo: MAXIMO_PARTICIPACIONES },
  );
  const ordenSeleccionado = exigirDecimal(
    datos.orden_seleccionado,
    "orden seleccionado",
    { maximo: MAXIMO_PARTICIPACIONES },
  );

  if (BigInt(totalEvaluaciones) > BigInt(totalParticipaciones)
    || totalEvaluaciones !== ordenSeleccionado
    || claveOrdenInstanteUTC(instantaneaGeneradaEn) < claveOrdenInstanteUTC(instanteReferencia)
    || claveOrdenInstanteUTC(generadaEn) < claveOrdenInstanteUTC(instantaneaGeneradaEn)) {
    throw new Error("confirmación de propuesta incoherente");
  }
  exigirETagConfirmacion(etag, huellaPropuesta);

  return {
    esquema: datos.esquema,
    propuesta_ref: exigirReferenciaConfirmacion(datos.propuesta_ref, "referencia de propuesta"),
    huella_propuesta_sha256: huellaPropuesta,
    bolsa,
    necesidad,
    instantanea,
    politica,
    instante_referencia: instanteReferencia,
    instantanea_generada_en: instantaneaGeneradaEn,
    total_participaciones_instantanea: totalParticipaciones,
    total_evaluaciones: totalEvaluaciones,
    orden_seleccionado: ordenSeleccionado,
    generada_en: generadaEn,
  };
}

export function validarPropuestaLlamamientoPresentacion(datos) {
  exigirCamposExactos(datos, CAMPOS_PRESENTACION, "propuesta de presentación");
  if (datos.esquema !== ESQUEMA_PRESENTACION || datos.demostracion !== true
    || datos.estado !== "demostracion" || !Array.isArray(datos.evaluaciones)
    || datos.evaluaciones.length === 0 || datos.evaluaciones.length > 100) {
    throw new Error("contrato de propuesta de presentación no compatible");
  }

  const evaluaciones = datos.evaluaciones.map((evaluacion, indice) => {
    exigirCamposExactos(evaluacion, CAMPOS_EVALUACION_PRESENTACION, "evaluación de presentación");
    const orden = exigirDecimal(evaluacion.orden, "orden de presentación", { maximo: 100n });
    if (orden !== String(indice + 1)
      || (evaluacion.resultado !== "elegible" && evaluacion.resultado !== "no_elegible")
      || !Array.isArray(evaluacion.motivos) || evaluacion.motivos.length === 0
      || evaluacion.motivos.length > 10) {
      throw new Error("evaluación de presentación incoherente");
    }
    return {
      orden,
      resultado: evaluacion.resultado,
      motivos: evaluacion.motivos.map((motivo) => {
        exigirCamposExactos(motivo, CAMPOS_MOTIVO_PRESENTACION, "motivo de presentación");
        return {
          regla: exigirCadena(motivo.regla, "regla de presentación", 160),
          fundamento: exigirCadena(motivo.fundamento, "fundamento de presentación", 500),
        };
      }),
    };
  });
  const personasIncluidas = exigirDecimal(
    datos.personas_incluidas,
    "personas incluidas en presentación",
    { admiteCero: true, maximo: 100n },
  );
  if (BigInt(personasIncluidas) !== BigInt(evaluaciones.filter((item) => item.resultado === "elegible").length)) {
    throw new Error("total de presentación incoherente");
  }

  return {
    esquema: datos.esquema,
    demostracion: true,
    id: exigirCadena(datos.id, "id de presentación", 100),
    necesidad_id: exigirCadena(datos.necesidad_id, "necesidad de presentación", 100),
    estado: "demostracion",
    version_bolsa: exigirCadena(datos.version_bolsa, "versión de bolsa de presentación", 160),
    version_regla: exigirCadena(datos.version_regla, "versión de regla de presentación", 160),
    fecha_corte: exigirInstanteUTC(datos.fecha_corte, "fecha de corte de presentación"),
    personas_incluidas: personasIncluidas,
    evaluaciones,
  };
}
