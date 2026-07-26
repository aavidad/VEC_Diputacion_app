/** Contrato neutral y cerrado de propuesta, decisión y rectificación. */

const MAXIMO_ENTERO_SEGURO = Number.MAX_SAFE_INTEGER;
const MAXIMA_PRIORIDAD = 65_535;
const MAXIMAS_VIAS = 64;
const MAXIMAS_CLAVES_POR_VIA = 32;
const PATRON_REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const PATRON_CLAVE = /^[a-z][a-z0-9._-]{1,79}$/u;
const PATRON_UUID_V4 =
  /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/u;
const UUID_V4_NULO = "00000000-0000-4000-8000-000000000000";
const PATRON_HUELLA = /^[0-9a-f]{64}$/u;
const PREFIJO_IDENTIDAD =
  "propuesta-cobertura-semantica:sha256:";
const CANON_IDENTIDAD = Object.freeze({
  dominio:
    "vec.dipgra.contratacion-temporal.propuesta-decision-cobertura-semantica",
  version_esquema: 1,
  algoritmo: "sha-256",
});
const ESTADOS_PROPUESTA = new Set([
  "viable",
  "incompleta",
  "conflictiva",
  "sin_via",
]);
const ESTADOS_EVALUACION = new Set([
  "viable",
  "incompleta",
  "conflictiva",
  "no_viable",
]);
const ESQUEMA_RESULTADO_CONSULTA_COBERTURA =
  "vec.contratacion-temporal.resultado-consulta-cobertura.v1";

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

function esListaPlana(valor, maximo) {
  if (!Array.isArray(valor) || Object.getPrototypeOf(valor) !== Array.prototype
    || valor.length > maximo || Object.getOwnPropertySymbols(valor).length !== 0
    || Reflect.ownKeys(valor).length !== valor.length + 1) {
    return false;
  }
  for (let indice = 0; indice < valor.length; indice += 1) {
    const descriptor = Object.getOwnPropertyDescriptor(valor, String(indice));
    if (!descriptor || !Object.hasOwn(descriptor, "value")
      || descriptor.enumerable !== true) return false;
  }
  return true;
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

function clonar(valor) {
  if (Array.isArray(valor)) return valor.map(clonar);
  if (esRegistro(valor)) {
    return Object.fromEntries(
      Object.entries(valor).map(([clave, dato]) => [clave, clonar(dato)]),
    );
  }
  return valor;
}

function congelar(valor) {
  if (Array.isArray(valor)) valor.forEach(congelar);
  else if (esRegistro(valor)) Object.values(valor).forEach(congelar);
  return Object.freeze(valor);
}

function clonarYCongelar(valor) {
  return congelar(clonar(valor));
}

function referenciaValida(valor) {
  return typeof valor === "string" && PATRON_REFERENCIA.test(valor);
}

function claveValida(valor, permiteVacia = false) {
  return permiteVacia && valor === ""
    || typeof valor === "string" && PATRON_CLAVE.test(valor);
}

function versionValida(valor, admiteMaximo = false) {
  return Number.isSafeInteger(valor)
    && valor >= 1
    && (admiteMaximo
      ? valor <= MAXIMO_ENTERO_SEGURO
      : valor < MAXIMO_ENTERO_SEGURO);
}

function huellaValida(valor) {
  return typeof valor === "string"
    && PATRON_HUELLA.test(valor)
    && !/^0{64}$/u.test(valor);
}

function instanteUTCValido(valor) {
  if (typeof valor !== "string") {
    return false;
  }
  const partes =
    /^(\d{4}-\d{2}-\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.\d{1,6})?Z$/u
      .exec(valor);
  if (!partes) return false;
  const [, fecha, hora, minuto, segundo] = partes;
  if (fecha.startsWith("0000-")) return false;
  const instante = new Date(valor);
  return Number.isFinite(instante.valueOf())
    && instante.toISOString().slice(0, 10) === fecha
    && Number(hora) <= 23
    && Number(minuto) <= 59
    && Number(segundo) <= 59;
}

function validarIdentidadSemantica(identidad) {
  exigirCamposExactos(
    identidad,
    ["referencia", "huella_sha256", "canon"],
    "identidad semántica",
  );
  exigirCamposExactos(
    identidad.canon,
    ["dominio", "version_esquema", "algoritmo"],
    "canon de identidad semántica",
  );
  if (!huellaValida(identidad.huella_sha256)
    || identidad.referencia
      !== `${PREFIJO_IDENTIDAD}${identidad.huella_sha256}`
    || identidad.canon.dominio !== CANON_IDENTIDAD.dominio
    || identidad.canon.version_esquema !== CANON_IDENTIDAD.version_esquema
    || identidad.canon.algoritmo !== CANON_IDENTIDAD.algoritmo) {
    throw new TypeError("identidad semántica no válida");
  }
  return clonar(identidad);
}

function validarListaClaves(lista, nombre) {
  if (!esListaPlana(lista, MAXIMAS_CLAVES_POR_VIA)
    || !lista.every((clave) => claveValida(clave))
    || new Set(lista).size !== lista.length) {
    throw new TypeError(`${nombre} no válida`);
  }
  return [...lista];
}

function validarEvaluacion(evaluacion, indice) {
  const nombre = `evaluaciones[${indice}]`;
  exigirCamposExactos(evaluacion, [
    "via_clave",
    "prioridad",
    "estado",
    "resultados_omitidos",
    "ausencias_bloqueantes",
    "ausencias_admitidas",
    "no_habilitantes",
    "conflictos",
  ], nombre);
  if (!claveValida(evaluacion.via_clave)
    || !Number.isSafeInteger(evaluacion.prioridad)
    || evaluacion.prioridad < 1
    || evaluacion.prioridad > MAXIMA_PRIORIDAD
    || !ESTADOS_EVALUACION.has(evaluacion.estado)) {
    throw new TypeError(`${nombre} no válida`);
  }
  const salida = {
    via_clave: evaluacion.via_clave,
    prioridad: evaluacion.prioridad,
    estado: evaluacion.estado,
    resultados_omitidos: validarListaClaves(
      evaluacion.resultados_omitidos,
      `${nombre}.resultados_omitidos`,
    ),
    ausencias_bloqueantes: validarListaClaves(
      evaluacion.ausencias_bloqueantes,
      `${nombre}.ausencias_bloqueantes`,
    ),
    ausencias_admitidas: validarListaClaves(
      evaluacion.ausencias_admitidas,
      `${nombre}.ausencias_admitidas`,
    ),
    no_habilitantes: validarListaClaves(
      evaluacion.no_habilitantes,
      `${nombre}.no_habilitantes`,
    ),
    conflictos: validarListaClaves(
      evaluacion.conflictos,
      `${nombre}.conflictos`,
    ),
  };
  const claves = [
    ...salida.resultados_omitidos,
    ...salida.ausencias_bloqueantes,
    ...salida.ausencias_admitidas,
    ...salida.no_habilitantes,
    ...salida.conflictos,
  ];
  if (claves.length > MAXIMAS_CLAVES_POR_VIA
    || new Set(claves).size !== claves.length) {
    throw new TypeError(`${nombre} contiene claves repetidas o excesivas`);
  }
  return salida;
}

export function validarSolicitudPropuestaCobertura(solicitud) {
  exigirCamposExactos(
    solicitud,
    ["expediente_ref", "version_esperada"],
    "solicitud de propuesta de cobertura",
  );
  if (!referenciaValida(solicitud.expediente_ref)
    || !versionValida(solicitud.version_esperada)) {
    throw new TypeError("solicitud de propuesta de cobertura no válida");
  }
  return clonarYCongelar(solicitud);
}

export function validarSolicitudConsultaResultadoCobertura(solicitud) {
  exigirCamposExactos(
    solicitud,
    ["expediente_ref", "clave_idempotencia"],
    "consulta de resultado de cobertura",
  );
  if (!referenciaValida(solicitud.expediente_ref)
    || typeof solicitud.clave_idempotencia !== "string"
    || !PATRON_UUID_V4.test(solicitud.clave_idempotencia)
    || solicitud.clave_idempotencia === UUID_V4_NULO) {
    throw new TypeError("consulta de resultado de cobertura no válida");
  }
  return clonarYCongelar(solicitud);
}

function validarDecisionComun(decision, rectificacion) {
  const campos = [
    "expediente_ref",
    "version_esperada",
    "clave_idempotencia",
    "identidad_semantica",
    "via_elegida",
    "motivo_clave",
  ];
  if (rectificacion) {
    campos.push("predecesora_ref", "predecesora_huella");
  }
  exigirCamposExactos(
    decision,
    campos,
    rectificacion ? "rectificación de cobertura" : "decisión de cobertura",
  );
  const motivoValido = rectificacion
    ? claveValida(decision.motivo_clave)
    : claveValida(decision.motivo_clave, true);
  if (!referenciaValida(decision.expediente_ref)
    || !versionValida(decision.version_esperada)
    || typeof decision.clave_idempotencia !== "string"
    || !PATRON_UUID_V4.test(decision.clave_idempotencia)
    || decision.clave_idempotencia === UUID_V4_NULO
    || !claveValida(decision.via_elegida)
    || !motivoValido
    || rectificacion && (
      !referenciaValida(decision.predecesora_ref)
      || !huellaValida(decision.predecesora_huella)
    )) {
    throw new TypeError(
      rectificacion
        ? "rectificación de cobertura no válida"
        : "decisión de cobertura no válida",
    );
  }
  const salida = clonar(decision);
  salida.identidad_semantica = validarIdentidadSemantica(
    decision.identidad_semantica,
  );
  return clonarYCongelar(salida);
}

export function validarSolicitudDecisionCobertura(decision) {
  return validarDecisionComun(decision, false);
}

export function validarSolicitudRectificacionCobertura(rectificacion) {
  return validarDecisionComun(rectificacion, true);
}

export function validarPropuestaCobertura(propuesta) {
  exigirCamposExactos(propuesta, [
    "esquema",
    "estado",
    "via_recomendada",
    "evaluaciones",
    "identidad_semantica",
  ], "propuesta de cobertura");
  if (propuesta.esquema
      !== "vec.contratacion-temporal.propuesta-cobertura.v1"
    || !ESTADOS_PROPUESTA.has(propuesta.estado)
    || !esListaPlana(propuesta.evaluaciones, MAXIMAS_VIAS)
    || propuesta.evaluaciones.length === 0
  ) {
    throw new TypeError("propuesta de cobertura no válida");
  }
  const evaluaciones = propuesta.evaluaciones.map(validarEvaluacion);
  const vias = evaluaciones.map(({ via_clave: via }) => via);
  const prioridades = evaluaciones.map(({ prioridad }) => prioridad);
  if (new Set(vias).size !== vias.length
    || new Set(prioridades).size !== prioridades.length
    || (propuesta.estado === "viable")
      !== claveValida(propuesta.via_recomendada)
    || propuesta.estado === "viable"
      && !evaluaciones.some(
        ({ via_clave: via, estado }) =>
          via === propuesta.via_recomendada && estado === "viable",
      )) {
    throw new TypeError("propuesta de cobertura no válida");
  }
  return clonarYCongelar({
    esquema: propuesta.esquema,
    estado: propuesta.estado,
    via_recomendada: propuesta.via_recomendada,
    evaluaciones,
    identidad_semantica: validarIdentidadSemantica(
      propuesta.identidad_semantica,
    ),
  });
}

export function validarReciboCobertura(recibo) {
  if (!esRegistro(recibo)) {
    throw new TypeError("recibo de cobertura no válido");
  }
  const aplicado = recibo.estado === "aplicada";
  exigirCamposExactos(
    recibo,
    aplicado
      ? [
        "esquema",
        "recibo_ref",
        "estado",
        "decision_cobertura_ref",
        "version_resultante",
        "confirmada_en",
      ]
      : ["esquema", "recibo_ref", "estado", "confirmada_en"],
    "recibo de cobertura",
  );
  if (recibo.esquema !== "vec.contratacion-temporal.recibo-cobertura.v1"
    || !referenciaValida(recibo.recibo_ref)
    || !instanteUTCValido(recibo.confirmada_en)
    || aplicado && (
      !referenciaValida(recibo.decision_cobertura_ref)
      || !versionValida(recibo.version_resultante, true)
    )
    || !aplicado && recibo.estado !== "denegada") {
    throw new TypeError("recibo de cobertura no válido");
  }
  return clonarYCongelar(recibo);
}

export function validarResultadoConsultaCobertura(resultado) {
  if (!esRegistro(resultado)) {
    throw new TypeError("resultado de consulta de cobertura no válido");
  }
  const confirmado = resultado.estado === "confirmado";
  exigirCamposExactos(
    resultado,
    confirmado
      ? ["esquema", "estado", "recibo"]
      : ["esquema", "estado"],
    "resultado de consulta de cobertura",
  );
  if (resultado.esquema !== ESQUEMA_RESULTADO_CONSULTA_COBERTURA
    || !confirmado && resultado.estado !== "no_observable") {
    throw new TypeError("resultado de consulta de cobertura no válido");
  }
  return confirmado
    ? clonarYCongelar({
      esquema: resultado.esquema,
      estado: resultado.estado,
      recibo: validarReciboCobertura(resultado.recibo),
    })
    : clonarYCongelar({
      esquema: resultado.esquema,
      estado: resultado.estado,
    });
}

export function liberaBloqueoResultadoCobertura(resultado) {
  return validarResultadoConsultaCobertura(resultado).estado === "confirmado";
}

export const LIMITES_COBERTURA_CONTRATACION = Object.freeze({
  maximasVias: MAXIMAS_VIAS,
  maximasClavesPorVia: MAXIMAS_CLAVES_POR_VIA,
});
