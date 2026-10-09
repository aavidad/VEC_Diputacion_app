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
 * `crearTextos` puede mezclar catálogos ya leídos y registrar claves ausentes.
 * La carga normal usa sólo el catálogo elegido; si falla, carga el de defecto.
 */
import { ERROR_INDICE_IDIOMAS, IDIOMA_ACTUAL, IDIOMA_POR_DEFECTO, INDICE_IDIOMAS,
  leerRecursoJSON, localizacionDe, prepararIdiomas, reintentarIdiomas } from "./idioma.js";

export const URL_RAIZ_TEXTOS = new URL("../textos/", import.meta.url);
/**
 * Huella de todos los catálogos de `textos/<idioma>/` (la comprueba
 * `textos-version.test.mjs`). Va como `?huella=` en cada lectura para que el
 * navegador los guarde sin volver a pedirlos; cambia con cualquier catálogo.
 */
export const VERSION_TEXTOS = "a3d14c8c789c96c0";

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

/** Valida la forma del catálogo recibido sin cargar otro idioma. */
export function esCatalogoValido(catalogo) {
  let entradas = 0;
  function valido(valor, profundidad) {
    if (typeof valor === "string") return valor.trim().length > 0;
    if (!esObjeto(valor) || profundidad > 12) return false;
    const claves = Object.keys(valor);
    if (claves.length === 0 || claves.length > 1024 || (entradas += claves.length) > 4096
      || claves.some((clave) => !/^[A-Za-z0-9_]+$/u.test(clave)
        || ["__proto__", "constructor", "prototype"].includes(clave))) return false;
    if (claves.every((clave) => CATEGORIAS_PLURAL.has(clave))) {
      if (profundidad === 0) return false;
      return esMensajePlural(valor) && claves.every((clave) => valido(valor[clave], profundidad + 1));
    }
    return claves.every((clave) => valido(valor[clave], profundidad + 1));
  }
  return esObjeto(catalogo) && valido(catalogo, 0);
}

/** URL del catálogo de un módulo en un idioma; rechaza nombres que no sean simples. */
export function urlCatalogo(idioma, modulo, raiz = URL_RAIZ_TEXTOS) {
  if (!PATRON_IDIOMA.test(String(idioma)) || !PATRON_MODULO.test(String(modulo))) {
    throw new TypeError("idioma o módulo de textos no válido");
  }
  return new URL(`${idioma}/${modulo}.json`, raiz);
}

/** URL de lectura: los catálogos propios llevan la huella común. */
function urlLectura(idioma, modulo, raiz) {
  const url = urlCatalogo(idioma, modulo, raiz);
  if (raiz.href === URL_RAIZ_TEXTOS.href) url.searchParams.set("huella", VERSION_TEXTOS);
  return url;
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
export function crearTextos({ modulo, idioma, localizacion, respaldo, propio = null, avisar,
  incidenciaCatalogo = null, incidenciaIndice = null } = {}) {
  if (!esCatalogoValido(respaldo)) throw new TypeError(`catálogo de textos de ${modulo} no válido`);
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
    modulo, idioma, localizacion, mensajes, faltantes: Object.freeze(faltantes), incidenciaCatalogo, incidenciaIndice,
    seccion, traducir, plural, numero, fecha,
  });
}

function avisoEnPruebas(mensaje) {
  if (EN_PRUEBAS) globalThis.console?.warn?.(mensaje);
}

async function leerYCrear(modulo, idioma, porDefecto, leer, raiz, avisar, incidenciaIndice = null) {
  async function leerValido(codigo) {
    const url = urlLectura(codigo, modulo, raiz);
    for (let intento = 0; intento < 2; intento++) {
      const datos = await leer(url);
      if (esCatalogoValido(datos)) return datos;
      if (intento === 0) {
        LECTURAS.delete(url.href);
        await new Promise((resolver) => setTimeout(resolver, 150));
      }
    }
    throw new TypeError(`catálogo de textos de ${modulo} no válido`);
  }
  let catalogo;
  let incidenciaCatalogo = null;
  try { catalogo = await leerValido(idioma); }
  catch (error) {
    if (idioma === porDefecto) throw error;
    incidenciaCatalogo = Object.freeze({ codigo: "catalogo_no_disponible", idioma, respaldo: porDefecto, causa: error });
    avisar?.(`textos de ${modulo} (${idioma}): catálogo no disponible; se usa ${porDefecto}: ${error}`);
    catalogo = await leerValido(porDefecto);
  }
  const efectivo = incidenciaCatalogo ? porDefecto : idioma;
  return crearTextos({
    modulo, idioma: efectivo, localizacion: localizacionDe(efectivo), respaldo: catalogo,
    avisar, incidenciaCatalogo, incidenciaIndice,
  });
}

/** Cargas con el lector predeterminado: cada catálogo se pide una sola vez por página. */
const CARGAS = new Map();
/**
 * Lecturas de cada fichero con el lector predeterminado. Se comparte el
 * resultado entre peticiones de la misma página; una lectura fallida se olvida
 * para que la persona pueda reintentar.
 */
const LECTURAS = new Map();
const INDICES_AVISADOS = new Set();

async function incidenciaAlPrepararIndice(avisar, reintentar = false) {
  try {
    await (reintentar && ERROR_INDICE_IDIOMAS ? reintentarIdiomas() : prepararIdiomas());
    return null;
  }
  catch (causa) {
    if (!INDICES_AVISADOS.has(causa)) {
      INDICES_AVISADOS.add(causa);
      (avisar ?? avisoEnPruebas)(`índice de idiomas no disponible; se usa el idioma del documento: ${causa}`);
    }
    return Object.freeze({ codigo: "indice_no_disponible", idioma: IDIOMA_ACTUAL, causa });
  }
}
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
 * Carga sólo el catálogo elegido. Si falla, usa el idioma por defecto completo.
 * Si también falla el de defecto, propaga el error para que la pantalla muestre
 * su estado de recuperación.
 */
export function cargarTextos(modulo, {
  idioma, porDefecto, leer, raiz = URL_RAIZ_TEXTOS, avisar, incidenciaIndice = null,
} = {}) {
  if (idioma === undefined || porDefecto === undefined) {
    return incidenciaAlPrepararIndice(avisar).then((incidencia) => {
      const elegido = INDICE_IDIOMAS.idiomas.some(({ codigo }) => codigo === idioma) ? idioma : IDIOMA_ACTUAL;
      return cargarTextos(modulo, {
        idioma: elegido, porDefecto: porDefecto ?? IDIOMA_POR_DEFECTO, leer, raiz, avisar,
        incidenciaIndice: incidencia ?? incidenciaIndice,
      });
    });
  }
  if (leer !== undefined || avisar !== undefined) {
    return leerYCrear(modulo, idioma, porDefecto, leer ?? leerRecursoJSON, raiz, avisar ?? avisoEnPruebas, incidenciaIndice);
  }
  const clave = `${raiz.href}|${porDefecto}|${idioma}|${modulo}`;
  if (!CARGAS.has(clave)) {
    const carga = leerYCrear(modulo, idioma, porDefecto, leerCatalogoUnaVez, raiz, avisoEnPruebas, incidenciaIndice);
    CARGAS.set(clave, carga);
    carga.catch(() => CARGAS.delete(clave));
  }
  return CARGAS.get(clave);
}

/** Relee un catálogo tras una incidencia, sin conservar el respaldo anterior. */
export async function reintentarTextos(modulo, { idioma, porDefecto, raiz = URL_RAIZ_TEXTOS } = {}) {
  const incidenciaIndice = await incidenciaAlPrepararIndice(undefined, true);
  const elegido = idioma ?? IDIOMA_ACTUAL;
  const respaldo = porDefecto ?? IDIOMA_POR_DEFECTO;
  CARGAS.delete(`${raiz.href}|${respaldo}|${elegido}|${modulo}`);
  LECTURAS.delete(urlLectura(elegido, modulo, raiz).href);
  LECTURAS.delete(urlLectura(respaldo, modulo, raiz).href);
  return cargarTextos(modulo, { idioma: elegido, porDefecto: respaldo, raiz, incidenciaIndice });
}
