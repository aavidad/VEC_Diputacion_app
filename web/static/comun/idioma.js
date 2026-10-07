/**
 * Idioma de la interfaz.
 *
 * Los idiomas disponibles, el idioma por defecto, su nombre propio y su
 * localización Intl son DATOS de `textos/idiomas.json`; este módulo no nombra
 * ningún idioma concreto. Añadir un idioma consiste en añadirlo al índice y
 * copiar y traducir su carpeta `textos/<codigo>/`.
 *
 * Orden de elección: el parámetro `lang` de la URL (elección de la persona en
 * el selector) y, si el índice lo permite (`seguir_navegador`), la preferencia
 * del navegador; en otro caso, el idioma por defecto. No se usa almacenamiento.
 *
 * Es también el único transporte de los JSON estáticos de textos: el índice
 * de idiomas y los catálogos `textos/<idioma>/<modulo>.json` (vía `textos.js`).
 *
 * Son datos públicos de la interfaz, sin datos personales. En la superficie
 * interna el certificado de cliente TLS exige credenciales del mismo origen;
 * no se envían a otro origen, no se sigue ninguna redirección y no se manda
 * `Referer`. El servidor responde estos JSON con `no-store`, de modo que un
 * cambio de traducción no requiere renovar versiones de caché.
 *
 * En Node (pruebas y herramientas) el módulo se carga desde `file:` y lee el
 * fichero del repositorio; el navegador nunca usa esa rama.
 */

const LIMITE_BYTES = 2 * 1024 * 1024;

async function leerFicheroLocal(url) {
  const { readFile } = await import("node:fs/promises");
  return readFile(url, "utf8");
}

export async function leerPorRed(url, fetchImpl, signal) {
  const origen = new URL(import.meta.url);
  if (url.protocol !== origen.protocol || url.host !== origen.host) throw new Error("recurso JSON fuera del propio origen");
  const opciones = {
    method: "GET",
    credentials: "same-origin",
    redirect: "error",
    referrerPolicy: "no-referrer",
    headers: { Accept: "application/json" },
    signal,
  };
  for (let intento = 0; intento < 2; intento++) {
    if (signal?.aborted) throw signal.reason ?? new DOMException("Aborted", "AbortError");
    try {
      const respuesta = await fetchImpl(url.href, opciones);
      if (!respuesta?.ok) {
        const error = new Error(`recurso JSON no disponible (${respuesta?.status ?? "sin respuesta"})`);
        if (intento === 0 && [502, 503].includes(respuesta?.status)) continue;
        throw error;
      }
      const declarado = Number(respuesta.headers?.get?.("Content-Length"));
      if (Number.isFinite(declarado) && declarado > LIMITE_BYTES) throw new Error("recurso JSON demasiado grande");
      return respuesta.text();
    } catch (error) {
      if (signal?.aborted || intento > 0 || /demasiado grande|no disponible/u.test(String(error?.message))) throw error;
    }
  }
}

/** Devuelve el JSON de `url` (absoluta y del mismo origen que este módulo). */
export async function leerRecursoJSON(url, { fetchImpl = globalThis.fetch, signal } = {}) {
  const destino = new URL(url);
  const propio = new URL(import.meta.url);
  if (destino.protocol !== propio.protocol || destino.host !== propio.host) {
    throw new Error("recurso JSON fuera del propio origen");
  }
  if (signal?.aborted) throw signal.reason ?? new DOMException("Aborted", "AbortError");
  const texto = destino.protocol === "file:" ? await leerFicheroLocal(destino) : await leerPorRed(destino, fetchImpl, signal);
  if (texto.length > LIMITE_BYTES) throw new Error("recurso JSON demasiado grande");
  return JSON.parse(texto);
}

export const URL_INDICE_IDIOMAS = new URL("../textos/idiomas.json", import.meta.url);

const PATRON_CODIGO = /^[a-z]{2,3}(?:-[a-z0-9]{2,8})*$/u;
const MAXIMO_IDIOMAS = 64;

function texto(valor, maximo = 64) {
  return typeof valor === "string" && valor.trim() === valor && valor.length > 0 && valor.length <= maximo;
}

function localizacionValida(valor) {
  if (!texto(valor, 35)) return false;
  try { return Intl.getCanonicalLocales(valor).length === 1; }
  catch { return false; }
}

/**
 * Valida el índice de idiomas y lo devuelve congelado como
 * `{ porDefecto, seguirNavegador, idiomas: [{ codigo, nombre, localizacion }] }`.
 */
export function normalizarIndiceIdiomas(datos) {
  if (!datos || typeof datos !== "object" || Array.isArray(datos) || !Array.isArray(datos.idiomas)
    || datos.idiomas.length === 0 || datos.idiomas.length > MAXIMO_IDIOMAS) {
    throw new TypeError("índice de idiomas no válido");
  }
  const vistos = new Set();
  const idiomas = datos.idiomas.map((idioma) => {
    if (!idioma || typeof idioma !== "object" || !PATRON_CODIGO.test(String(idioma.codigo ?? ""))
      || vistos.has(idioma.codigo) || !texto(idioma.nombre) || !localizacionValida(idioma.localizacion)) {
      throw new TypeError("idioma no válido en el índice");
    }
    vistos.add(idioma.codigo);
    return Object.freeze({ codigo: idioma.codigo, nombre: idioma.nombre, localizacion: idioma.localizacion });
  });
  if (!vistos.has(datos.por_defecto)) throw new TypeError("el idioma por defecto no figura en el índice");
  return Object.freeze({
    porDefecto: datos.por_defecto,
    seguirNavegador: datos.seguir_navegador === true,
    idiomas: Object.freeze(idiomas),
  });
}

/**
 * Índice mínimo si el de datos no puede leerse: el idioma declarado por el
 * propio documento (`<html lang>`), que ya viene escrito en ese idioma.
 */
function indiceDelDocumento() {
  const lang = String(globalThis.document?.documentElement?.lang ?? "").toLowerCase();
  const codigo = PATRON_CODIGO.test(lang) ? lang : "und";
  return normalizarIndiceIdiomas({
    por_defecto: codigo, seguir_navegador: false,
    idiomas: [{ codigo, nombre: codigo, localizacion: codigo }],
  });
}

export let INDICE_IDIOMAS = indiceDelDocumento();
export let IDIOMAS_DISPONIBLES = INDICE_IDIOMAS.idiomas;
export let IDIOMA_POR_DEFECTO = INDICE_IDIOMAS.porDefecto;
export let ERROR_INDICE_IDIOMAS = null;
let cargaIndice;

/** La importación nunca espera a la red. Una lectura fallida permite otro intento. */
export function prepararIdiomas({ leer = leerRecursoJSON } = {}) {
  if (leer !== leerRecursoJSON) return leer(URL_INDICE_IDIOMAS).then(normalizarIndiceIdiomas);
  if (!cargaIndice) {
    cargaIndice = leer(URL_INDICE_IDIOMAS).then(normalizarIndiceIdiomas).then((indice) => {
      INDICE_IDIOMAS = indice;
      IDIOMAS_DISPONIBLES = indice.idiomas;
      IDIOMA_POR_DEFECTO = indice.porDefecto;
      ERROR_INDICE_IDIOMAS = null;
      IDIOMA_ACTUAL = resolverIdiomaNavegacion();
      LOCALIZACION_ACTUAL = localizacionDe(IDIOMA_ACTUAL);
      return indice;
    }).catch((error) => {
      ERROR_INDICE_IDIOMAS = error;
      throw error;
    });
  }
  return cargaIndice;
}

/** Nuevo intento explícito tras una incidencia de índice. */
export function reintentarIdiomas() {
  if (!ERROR_INDICE_IDIOMAS) return prepararIdiomas();
  cargaIndice = null;
  return prepararIdiomas();
}

function admitido(codigo, indice) {
  return indice.idiomas.some((idioma) => idioma.codigo === codigo);
}

/** Idioma de la interfaz entre los del índice. La URL prevalece sobre el navegador. */
export function seleccionarIdioma(parametro = "", preferencias = [], indiceOPreferido = INDICE_IDIOMAS) {
  const indice = typeof indiceOPreferido === "string" ? INDICE_IDIOMAS : indiceOPreferido;
  const solicitado = String(parametro ?? "").toLowerCase();
  if (admitido(solicitado, indice)) return solicitado;
  if (typeof indiceOPreferido === "string" && admitido(indiceOPreferido, indice)) return indiceOPreferido;
  if (indice.seguirNavegador) {
    for (const preferencia of preferencias ?? []) {
      const completa = String(preferencia ?? "").toLowerCase();
      if (admitido(completa, indice)) return completa;
      const base = completa.split("-", 1)[0];
      if (admitido(base, indice)) return base;
    }
  }
  return indice.porDefecto;
}

/** Localización Intl del idioma indicado (o la del idioma por defecto). */
export function localizacionDe(codigo, indice = INDICE_IDIOMAS) {
  const encontrado = indice.idiomas.find((idioma) => idioma.codigo === codigo)
    ?? indice.idiomas.find((idioma) => idioma.codigo === indice.porDefecto);
  return encontrado.localizacion;
}

function parametroActual(ubicacion) {
  try { return new URL(ubicacion.href).searchParams.get("lang"); }
  catch { return ""; }
}

/** Resuelve cada navegación sin guardar el resultado. URL, servidor y navegador. */
export function resolverIdiomaNavegacion({
  ubicacion = globalThis.location,
  idiomaPreferido = "navegador",
  navegador = globalThis.navigator,
} = {}) {
  const solicitado = parametroActual(ubicacion);
  if (admitido(solicitado, INDICE_IDIOMAS)) return solicitado;
  if (admitido(idiomaPreferido, INDICE_IDIOMAS)) return idiomaPreferido;
  return seleccionarIdioma("", navegador?.languages ?? (navegador?.language ? [navegador.language] : []));
}

const ubicacionActual = globalThis.location;
export let IDIOMA_ACTUAL = resolverIdiomaNavegacion({ ubicacion: ubicacionActual });
export let LOCALIZACION_ACTUAL = localizacionDe(IDIOMA_ACTUAL);

/** Conserva ruta, parámetros ajenos y ancla al cambiar el idioma. */
export function cambiarIdioma(idioma, ubicacion = globalThis.location, indice = INDICE_IDIOMAS) {
  if (!admitido(idioma, indice)) return false;
  if (!ubicacion?.href || typeof ubicacion.assign !== "function") return false;
  const destino = new URL(ubicacion.href);
  destino.searchParams.set("lang", idioma);
  ubicacion.assign(destino.href);
  return true;
}

/**
 * Rellena un selector nativo con los idiomas del índice (cada uno con su
 * nombre propio y su `lang`) si el documento lo permite.
 */
function rellenarOpciones(selector, indice) {
  const documento = selector.ownerDocument;
  if (typeof documento?.createElement !== "function" || typeof selector.replaceChildren !== "function") return;
  selector.replaceChildren(...indice.idiomas.map((idioma) => {
    const opcion = documento.createElement("option");
    opcion.value = idioma.codigo;
    opcion.lang = idioma.codigo;
    opcion.textContent = idioma.nombre;
    return opcion;
  }));
}

/** Conecta un selector nativo al idioma de la interfaz. */
export function montarSelectorIdioma(selector, ubicacion = globalThis.location, preferidoOIndice = "navegador") {
  if (!selector?.addEventListener) return false;
  const indice = typeof preferidoOIndice === "object" ? preferidoOIndice : INDICE_IDIOMAS;
  rellenarOpciones(selector, indice);
  selector.value = typeof preferidoOIndice === "object"
    ? seleccionarIdioma(parametroActual(ubicacion), globalThis.navigator?.languages ?? [], indice)
    : resolverIdiomaNavegacion({ ubicacion, idiomaPreferido: preferidoOIndice });
  selector.addEventListener("change", () => { cambiarIdioma(selector.value, ubicacion, indice); });
  return true;
}
