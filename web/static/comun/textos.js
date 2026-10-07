/**
 * Lector común de textos de la interfaz.
 *
 * Todos los textos visibles viven en catálogos de DATOS por idioma:
 * `textos/<idioma>/<modulo>.json`. El código no contiene diccionarios ni nombra
 * idiomas: el idioma actual y el de respaldo salen de `textos/idiomas.json` a
 * través de `idioma.js`.
 *
 * Formato de un catálogo: un objeto de secciones (objetos) y mensajes. Un
 * mensaje es una cadena con variables `{nombre}` o, para plurales, un objeto
 * con categorías de `Intl.PluralRules` (`zero`, `one`, `two`, `few`, `many`,
 * `other`), en el que `other` es obligatoria:
 *
 *   { "general": { "saludo": "Hola, {nombre}",
 *                  "dias": { "one": "{n} día", "other": "{n} días" } } }
 *
 * Si al catálogo del idioma actual le falta una clave, se usa la del idioma por
 * defecto y se registra en `faltantes` (y, en pruebas Node, se avisa).
 */
import { IDIOMA_ACTUAL, IDIOMA_POR_DEFECTO, leerRecursoJSON, localizacionDe } from "./idioma.js";

export const URL_RAIZ_TEXTOS = new URL("../textos/", import.meta.url);

const PATRON_MODULO = /^[a-z0-9]+(?:-[a-z0-9]+)*$/u;
const PATRON_IDIOMA = /^[a-z]{2,3}(?:-[a-z0-9]{2,8})*$/u;
const CATEGORIAS_PLURAL = new Set(["zero", "one", "two", "few", "many", "other"]);
const PATRON_VARIABLE = /\{([A-Za-z_][A-Za-z0-9_]*)\}/gu;
const EN_PRUEBAS = new URL(import.meta.url).protocol === "file:";

function esObjeto(valor) {
  return valor !== null && typeof valor === "object" && !Array.isArray(valor);
}

/** Un objeto cuyas claves son categorías plurales, con `other`, y valores cadena. */
export function esMensajePlural(valor) {
  if (!esObjeto(valor) || typeof valor.other !== "string") return false;
  return Object.entries(valor).every(([clave, texto]) => CATEGORIAS_PLURAL.has(clave) && typeof texto === "string");
}

/** URL del catálogo de un módulo en un idioma; rechaza nombres que no sean simples. */
export function urlCatalogo(idioma, modulo, raiz = URL_RAIZ_TEXTOS) {
  if (!PATRON_IDIOMA.test(String(idioma)) || !PATRON_MODULO.test(String(modulo))) {
    throw new TypeError("idioma o módulo de textos no válido");
  }
  return new URL(`${idioma}/${modulo}.json`, raiz);
}

/**
 * Combina el catálogo del idioma actual sobre el de respaldo. Solo cuentan las
 * claves del respaldo (el idioma por defecto define la forma del catálogo); lo
 * que falte o no tenga la misma forma se toma del respaldo y se anota.
 */
function combinar(respaldo, propio, ruta, faltantes) {
  const resultado = {};
  for (const [clave, base] of Object.entries(respaldo)) {
    const camino = ruta ? `${ruta}.${clave}` : clave;
    const valor = esObjeto(propio) ? propio[clave] : undefined;
    if (typeof base === "string" || esMensajePlural(base)) {
      const valido = typeof valor === "string" ? valor !== "" : esMensajePlural(valor);
      if (!valido) faltantes.push(camino);
      resultado[clave] = Object.freeze(valido ? valor : base);
    } else if (esObjeto(base)) {
      resultado[clave] = combinar(base, valor, camino, faltantes);
    }
  }
  return Object.freeze(resultado);
}

function interpolar(plantilla, variables) {
  return plantilla.replace(PATRON_VARIABLE, (_coincidencia, nombre) => String(variables?.[nombre] ?? ""));
}

/** Construye el objeto de textos a partir de catálogos ya leídos. */
export function crearTextos({ modulo, idioma, localizacion, respaldo, propio = null, avisar } = {}) {
  if (!esObjeto(respaldo)) throw new TypeError(`catálogo de textos de ${modulo} no válido`);
  const faltantes = [];
  const mensajes = combinar(respaldo, propio ?? respaldo, "", faltantes);
  if (faltantes.length > 0 && typeof avisar === "function") {
    avisar(`textos de ${modulo} (${idioma}): ${faltantes.length} claves sin traducir: ${faltantes.join(", ")}`);
  }
  const reglasPlural = new Intl.PluralRules(localizacion);

  function buscar(ruta) {
    let actual = mensajes;
    for (const parte of String(ruta).split(".")) {
      if (!esObjeto(actual) || !Object.hasOwn(actual, parte)) throw new Error(`clave de textos desconocida: ${modulo}.${ruta}`);
      actual = actual[parte];
    }
    return actual;
  }

  const numero = (valor, opciones) => new Intl.NumberFormat(localizacion, opciones).format(valor);
  const fecha = (valor, opciones = { dateStyle: "medium" }) =>
    new Intl.DateTimeFormat(localizacion, opciones).format(valor instanceof Date ? valor : new Date(valor));

  /** Mensaje plural según `Intl.PluralRules`; `{cuenta}` se presenta localizada. */
  function plural(ruta, cuenta, variables = {}) {
    const valor = buscar(ruta);
    if (!esMensajePlural(valor)) throw new Error(`clave de textos no es plural: ${modulo}.${ruta}`);
    const cantidad = Number(cuenta);
    const plantilla = valor[reglasPlural.select(cantidad)] ?? valor.other;
    return interpolar(plantilla, { ...variables, cuenta: numero(cantidad) });
  }

  /** Mensaje `seccion.clave` con sus variables. Un plural usa `variables.cuenta`. */
  function traducir(ruta, variables = {}) {
    const valor = buscar(ruta);
    if (typeof valor === "string") return interpolar(valor, variables);
    if (esMensajePlural(valor)) return plural(ruta, variables.cuenta, variables);
    throw new Error(`clave de textos no es un mensaje: ${modulo}.${ruta}`);
  }

  /** Sección de mensajes (objeto congelado). */
  function seccion(nombre) {
    const valor = buscar(nombre);
    if (!esObjeto(valor) || esMensajePlural(valor)) throw new Error(`sección de textos desconocida: ${modulo}.${nombre}`);
    return valor;
  }

  return Object.freeze({
    modulo, idioma, localizacion, mensajes, faltantes: Object.freeze(faltantes),
    seccion, traducir, plural, numero, fecha,
  });
}

function avisoEnPruebas(mensaje) {
  if (EN_PRUEBAS) globalThis.console?.warn?.(mensaje);
}

async function leerYCrear(modulo, idioma, porDefecto, leer, raiz, avisar) {
  const lecturaRespaldo = leer(urlCatalogo(porDefecto, modulo, raiz));
  const lecturaPropia = idioma === porDefecto ? Promise.resolve(null)
    : leer(urlCatalogo(idioma, modulo, raiz)).catch(() => undefined);
  const [respaldo, propio] = await Promise.all([lecturaRespaldo, lecturaPropia]);
  const efectivo = propio === undefined ? porDefecto : idioma;
  if (propio === undefined) avisar?.(`textos de ${modulo} (${idioma}): catálogo no disponible; se usa ${porDefecto}`);
  return crearTextos({
    modulo, idioma: efectivo, localizacion: localizacionDe(efectivo), respaldo,
    propio: propio ?? null, avisar,
  });
}

/** Cargas con el lector predeterminado: cada catálogo se pide una sola vez por página. */
const CARGAS = new Map();
/**
 * Lecturas de cada fichero con el lector predeterminado. El catálogo del idioma
 * por defecto es el respaldo de todos los demás: sin esto, pedir los textos de
 * un módulo en dos idiomas lo descargaba dos veces. Una lectura fallida se
 * olvida para que se pueda reintentar.
 */
const LECTURAS = new Map();
export function leerCatalogoUnaVez(url) {
  const clave = url.href;
  if (!LECTURAS.has(clave)) {
    const lectura = leerRecursoJSON(url);
    LECTURAS.set(clave, lectura);
    lectura.catch(() => LECTURAS.delete(clave));
  }
  return LECTURAS.get(clave);
}

/**
 * Carga los textos de `modulo` en el idioma de la interfaz, con respaldo en el
 * idioma por defecto. Si falla el catálogo del idioma actual se usa el de
 * respaldo completo; si falla el de respaldo, la carga falla (y no se memoriza).
 */
export function cargarTextos(modulo, {
  idioma = IDIOMA_ACTUAL, porDefecto = IDIOMA_POR_DEFECTO, leer, raiz = URL_RAIZ_TEXTOS, avisar,
} = {}) {
  if (leer !== undefined || avisar !== undefined) {
    return leerYCrear(modulo, idioma, porDefecto, leer ?? leerRecursoJSON, raiz, avisar ?? avisoEnPruebas);
  }
  const clave = `${raiz.href}|${porDefecto}|${idioma}|${modulo}`;
  if (!CARGAS.has(clave)) {
    const carga = leerYCrear(modulo, idioma, porDefecto, leerCatalogoUnaVez, raiz, avisoEnPruebas);
    CARGAS.set(clave, carga);
    carga.catch(() => CARGAS.delete(clave));
  }
  return CARGAS.get(clave);
}
