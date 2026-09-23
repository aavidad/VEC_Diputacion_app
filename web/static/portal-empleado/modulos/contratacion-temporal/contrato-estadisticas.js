/**
 * Contrato cerrado para las estadísticas de contratación temporal (C18 / G16).
 *
 * Esquema: vec.ct.estadisticas.v1
 * Reglas de integridad:
 * - Sin datos personales.
 * - Vocabulario cerrado de periodos: anual, mensual, semanal.
 * - Series ordenadas cronológicamente con enteros no negativos.
 * - Totales agregados coherentes.
 */

export const ESQUEMA_ESTADISTICAS = "vec.ct.estadisticas.v1";

export const PERIODOS_ESTADISTICAS = Object.freeze(["anual", "mensual", "semanal"]);

const CAMPOS_SERIE = Object.freeze([
  "inicio",
  "altas",
  "llamamientos",
  "formalizaciones",
  "cierres",
  "incidencias",
]);

const CAMPOS_TOTALES = Object.freeze([
  "altas",
  "llamamientos",
  "formalizaciones",
  "cierres",
  "incidencias",
]);

const CAMPOS_ESTADISTICAS = Object.freeze([
  "esquema",
  "periodo",
  "desde",
  "hasta",
  "series",
  "totales",
]);

const PATRON_FECHA = /^\d{4}-\d{2}-\d{2}$/;
const PATRON_DNI = /\b\d{8}[A-Za-z]\b/;
const PATRON_EMAIL = /[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}/;
const PATRON_TELEFONO = /(?:\+?34[\s.-]?)?[6789]\d{2}[\s.-]?\d{3}[\s.-]?\d{3}\b/;

function esObjeto(valor) {
  return valor !== null && typeof valor === "object" && !Array.isArray(valor);
}

function contieneDatosPersonales(cadena) {
  if (typeof cadena !== "string") return false;
  return PATRON_DNI.test(cadena)
    || PATRON_EMAIL.test(cadena)
    || PATRON_TELEFONO.test(cadena);
}

function exigirCamposExactos(objeto, campos, nombre) {
  if (!esObjeto(objeto)) throw new TypeError(`${nombre} no es un objeto válido`);
  const claves = Object.keys(objeto);
  if (claves.length !== campos.length || campos.some((campo) => !Object.hasOwn(objeto, campo))) {
    throw new TypeError(`${nombre} no respeta el contrato cerrado`);
  }
}

function exigirEnteroNoNegativo(valor, nombre, maximo = 10_000_000) {
  if (!Number.isSafeInteger(valor) || valor < 0 || valor > maximo) {
    throw new TypeError(`${nombre} debe ser un entero no negativo válido`);
  }
  return valor;
}

function exigirFecha(valor, nombre) {
  if (typeof valor !== "string" || !PATRON_FECHA.test(valor)) {
    throw new TypeError(`${nombre} debe tener formato AAAA-MM-DD`);
  }
  const fecha = new Date(`${valor}T00:00:00Z`);
  if (Number.isNaN(fecha.getTime()) || fecha.toISOString().slice(0, 10) !== valor) {
    throw new TypeError(`${nombre} no es una fecha válida`);
  }
  if (contieneDatosPersonales(valor)) {
    throw new TypeError(`${nombre} contiene datos personales no permitidos`);
  }
  return valor;
}

function inicioPeriodo(fecha, periodo) {
  if (periodo === "anual") return `${fecha.slice(0, 4)}-01-01`;
  if (periodo === "mensual") return `${fecha.slice(0, 7)}-01`;
  const lunes = new Date(`${fecha}T00:00:00Z`);
  lunes.setUTCDate(lunes.getUTCDate() - (lunes.getUTCDay() + 6) % 7);
  return lunes.toISOString().slice(0, 10);
}

export function validarSerieEstadisticas(serie, indice = 0) {
  exigirCamposExactos(serie, CAMPOS_SERIE, `serie[${indice}]`);
  const inicio = exigirFecha(serie.inicio, `serie[${indice}].inicio`);
  const altas = exigirEnteroNoNegativo(serie.altas, `serie[${indice}].altas`);
  const llamamientos = exigirEnteroNoNegativo(serie.llamamientos, `serie[${indice}].llamamientos`);
  const formalizaciones = exigirEnteroNoNegativo(serie.formalizaciones, `serie[${indice}].formalizaciones`);
  const cierres = exigirEnteroNoNegativo(serie.cierres, `serie[${indice}].cierres`);
  const incidencias = exigirEnteroNoNegativo(serie.incidencias, `serie[${indice}].incidencias`);

  return Object.freeze({
    inicio,
    altas,
    llamamientos,
    formalizaciones,
    cierres,
    incidencias,
  });
}

export function validarTotalesEstadisticas(totales) {
  exigirCamposExactos(totales, CAMPOS_TOTALES, "totales");
  return Object.freeze({
    altas: exigirEnteroNoNegativo(totales.altas, "totales.altas"),
    llamamientos: exigirEnteroNoNegativo(totales.llamamientos, "totales.llamamientos"),
    formalizaciones: exigirEnteroNoNegativo(totales.formalizaciones, "totales.formalizaciones"),
    cierres: exigirEnteroNoNegativo(totales.cierres, "totales.cierres"),
    incidencias: exigirEnteroNoNegativo(totales.incidencias, "totales.incidencias"),
  });
}

export function validarRespuestaEstadisticas(envelope) {
  if (!esObjeto(envelope)) throw new TypeError("el envelope de respuesta no es un objeto válido");
  if (!Object.hasOwn(envelope, "data")) throw new TypeError("la respuesta debe contener el campo 'data'");

  const datos = envelope.data;
  if (!esObjeto(datos)) throw new TypeError("data de estadísticas no es un objeto válido");

  const camposObligatorios = ["esquema", "periodo", "desde", "hasta", "series", "totales"];
  const camposPermitidos = new Set([...camposObligatorios, "zona_horaria", "corte_global"]);

  for (const c of camposObligatorios) {
    if (!Object.hasOwn(datos, c)) throw new TypeError(`campo obligatorio ausente en estadísticas: ${c}`);
  }
  for (const k of Object.keys(datos)) {
    if (!camposPermitidos.has(k)) throw new TypeError(`campo no permitido en estadísticas: ${k}`);
  }

  if (datos.esquema !== ESQUEMA_ESTADISTICAS) {
    throw new TypeError(`esquema no compatible: ${datos.esquema}`);
  }
  if (!PERIODOS_ESTADISTICAS.includes(datos.periodo)) {
    throw new TypeError(`periodo no reconocido: ${datos.periodo}`);
  }
  const periodo = datos.periodo;
  const desde = exigirFecha(datos.desde, "desde");
  const hasta = exigirFecha(datos.hasta, "hasta");

  if (desde > hasta) {
    throw new TypeError("el rango temporal no es válido: 'desde' es posterior a 'hasta'");
  }

  if (datos.zona_horaria !== undefined) {
    if (typeof datos.zona_horaria !== "string" || datos.zona_horaria.trim() === "" || contieneDatosPersonales(datos.zona_horaria)) {
      throw new TypeError("zona_horaria no es válida");
    }
  }

  if (datos.corte_global !== undefined) {
    exigirEnteroNoNegativo(datos.corte_global, "corte_global");
  }

  if (!Array.isArray(datos.series)) {
    throw new TypeError("el campo series debe ser un array");
  }
  const series = Object.freeze(datos.series.map(validarSerieEstadisticas));
  const primerInicio = inicioPeriodo(desde, periodo);
  const ultimoInicio = inicioPeriodo(hasta, periodo);
  for (let indice = 0; indice < series.length; indice++) {
    const inicio = series[indice].inicio;
    if (inicio < primerInicio || inicio > ultimoInicio) {
      throw new TypeError(`serie[${indice}].inicio está fuera del rango consultado`);
    }
    if (indice > 0 && inicio <= series[indice - 1].inicio) {
      throw new TypeError(`serie[${indice}].inicio no sigue un orden cronológico estricto`);
    }
  }
  const totales = validarTotalesEstadisticas(datos.totales);

  const suma = series.reduce((acc, s) => ({
    altas: acc.altas + s.altas,
    llamamientos: acc.llamamientos + s.llamamientos,
    formalizaciones: acc.formalizaciones + s.formalizaciones,
    cierres: acc.cierres + s.cierres,
    incidencias: acc.incidencias + s.incidencias,
  }), { altas: 0, llamamientos: 0, formalizaciones: 0, cierres: 0, incidencias: 0 });

  if (CAMPOS_TOTALES.some((k) => suma[k] !== totales[k])) {
    throw new TypeError("los totales acumulados no coinciden con la suma de las series");
  }

  const resultado = {
    esquema: datos.esquema,
    periodo,
    desde,
    hasta,
    series,
    totales,
  };
  if (datos.zona_horaria !== undefined) resultado.zona_horaria = datos.zona_horaria;
  if (datos.corte_global !== undefined) resultado.corte_global = datos.corte_global;

  return Object.freeze(resultado);
}

export function generarCSVEstadisticas(estadisticas) {
  if (!estadisticas || !Array.isArray(estadisticas.series) || !estadisticas.totales) {
    throw new TypeError("estadísticas no válidas para generar CSV");
  }

  const cabeceras = ["Periodo inicio", "Altas", "Llamamientos", "Formalizaciones", "Cierres", "Incidencias"];
  const filas = [cabeceras.join(";")];

  for (const s of estadisticas.series) {
    filas.push([
      s.inicio,
      s.altas,
      s.llamamientos,
      s.formalizaciones,
      s.cierres,
      s.incidencias,
    ].join(";"));
  }

  const t = estadisticas.totales;
  filas.push([
    "TOTAL",
    t.altas,
    t.llamamientos,
    t.formalizaciones,
    t.cierres,
    t.incidencias,
  ].join(";"));

  return filas.join("\r\n");
}
