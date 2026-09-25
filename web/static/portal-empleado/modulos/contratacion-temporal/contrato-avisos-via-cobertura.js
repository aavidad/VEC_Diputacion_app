/**
 * Contrato cerrado de las comprobaciones automáticas de la vía de cobertura
 * (fase 3): bolsa agotada provisionalmente, propuesta de oferta al Servicio
 * Andaluz de Empleo y aviso de nueva convocatoria. Son solo propuestas; la
 * vía la decide RRHH en el formulario de cobertura.
 */

export const ESQUEMA_AVISOS_VIA_COBERTURA = "vec.contratacion-temporal.avisos-via-cobertura.v1";
const ESTADOS = new Set(["evaluados", "sin_bolsa", "no_disponible"]);
const CLAVES_AVISO = new Set([
  "bolsa_agotada_provisionalmente",
  "propuesta_oferta_sae",
  "nueva_convocatoria",
]);
const MOTIVOS = new Set(["vigencia_superada", "bolsa_agotada"]);
const ORIGENES = new Set(["reglamento", "ejemplo"]);
const MAXIMOS_AVISOS = 8;
const MAXIMAS_REGLAS = 8;
const MAXIMO_TEXTO = 2000;
const PATRON_FECHA = /^\d{4}-\d{2}-\d{2}$/u;
const PATRON_CLAVE_REGLA = /^[a-z0-9][a-z0-9._-]{1,79}$/u;

const CAMPOS_AVISO = new Set([
  "clave", "disponibles", "integrantes", "umbral", "duracion_maxima_meses",
  "fin_maximo", "fin_previsto", "excede_duracion", "motivos", "constituida_en",
  "vigencia_hasta", "reglas",
]);
const CAMPOS_REGLA = new Set([
  "clave", "etiqueta", "descripcion", "origen", "articulo", "norma",
  "parte_ejemplo", "referencia", "ejemplo",
]);

function fallo() {
  throw new TypeError("avisos de la vía de cobertura no válidos");
}

function registroPlano(valor, permitidos) {
  if (valor === null || typeof valor !== "object" || Array.isArray(valor)
    || Object.getPrototypeOf(valor) !== Object.prototype
    || Object.getOwnPropertySymbols(valor).length !== 0) fallo();
  for (const campo of Object.keys(valor)) {
    if (!permitidos.has(campo)) fallo();
  }
  return valor;
}

function listaPlana(valor, maximo) {
  if (!Array.isArray(valor) || valor.length > maximo
    || Object.getPrototypeOf(valor) !== Array.prototype) fallo();
  return valor;
}

function texto(valor, { opcional = false } = {}) {
  if (valor === undefined && opcional) return "";
  if (typeof valor !== "string" || valor.length > MAXIMO_TEXTO) fallo();
  return valor;
}

function entero(valor) {
  if (valor === undefined) return undefined;
  if (!Number.isSafeInteger(valor) || valor < 0) fallo();
  return valor;
}

function fecha(valor) {
  if (valor === undefined || valor === "") return "";
  if (typeof valor !== "string" || !PATRON_FECHA.test(valor)) fallo();
  return valor;
}

function validarRegla(regla) {
  registroPlano(regla, CAMPOS_REGLA);
  if (typeof regla.clave !== "string" || !PATRON_CLAVE_REGLA.test(regla.clave)
    || !ORIGENES.has(regla.origen) || typeof regla.ejemplo !== "boolean"
    || (regla.origen === "reglamento") !== (typeof regla.articulo === "string" && regla.articulo !== "")) fallo();
  return {
    clave: regla.clave,
    etiqueta: texto(regla.etiqueta),
    descripcion: texto(regla.descripcion),
    origen: regla.origen,
    articulo: texto(regla.articulo, { opcional: true }),
    norma: texto(regla.norma),
    parte_ejemplo: texto(regla.parte_ejemplo, { opcional: true }),
    referencia: texto(regla.referencia),
    ejemplo: regla.ejemplo,
  };
}

function validarAviso(aviso) {
  registroPlano(aviso, CAMPOS_AVISO);
  if (!CLAVES_AVISO.has(aviso.clave)) fallo();
  const salida = {
    clave: aviso.clave,
    reglas: listaPlana(aviso.reglas, MAXIMAS_REGLAS).map(validarRegla),
  };
  if (salida.reglas.length === 0) fallo();
  if (aviso.clave === "bolsa_agotada_provisionalmente") {
    salida.disponibles = entero(aviso.disponibles);
    salida.integrantes = entero(aviso.integrantes);
    salida.umbral = entero(aviso.umbral);
    if ([salida.disponibles, salida.integrantes, salida.umbral].includes(undefined)
      || salida.disponibles > salida.integrantes) fallo();
  } else if (aviso.clave === "propuesta_oferta_sae") {
    salida.duracion_maxima_meses = entero(aviso.duracion_maxima_meses);
    salida.fin_maximo = fecha(aviso.fin_maximo);
    salida.fin_previsto = fecha(aviso.fin_previsto);
    if (!salida.duracion_maxima_meses || !salida.fin_maximo
      || typeof aviso.excede_duracion !== "boolean") fallo();
    salida.excede_duracion = aviso.excede_duracion;
  } else {
    const motivos = listaPlana(aviso.motivos, MOTIVOS.size);
    if (motivos.length === 0 || !motivos.every((motivo) => MOTIVOS.has(motivo))
      || new Set(motivos).size !== motivos.length) fallo();
    salida.motivos = [...motivos];
    salida.constituida_en = fecha(aviso.constituida_en);
    salida.vigencia_hasta = fecha(aviso.vigencia_hasta);
  }
  return salida;
}

function congelar(valor) {
  if (valor && typeof valor === "object") {
    Object.values(valor).forEach(congelar);
    Object.freeze(valor);
  }
  return valor;
}

export function validarAvisosViaCobertura(avisos) {
  registroPlano(avisos, new Set(["esquema", "estado", "evaluada_en", "avisos"]));
  if (avisos.esquema !== ESQUEMA_AVISOS_VIA_COBERTURA || !ESTADOS.has(avisos.estado)
    || typeof avisos.evaluada_en !== "string" || Number.isNaN(Date.parse(avisos.evaluada_en))) fallo();
  const lista = listaPlana(avisos.avisos, MAXIMOS_AVISOS).map(validarAviso);
  if (avisos.estado !== "evaluados" && lista.length !== 0) fallo();
  return congelar({
    esquema: avisos.esquema,
    estado: avisos.estado,
    evaluada_en: avisos.evaluada_en,
    avisos: lista,
  });
}
