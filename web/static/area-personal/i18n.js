/**
 * Textos del área personal.
 *
 * Viven en `textos/<idioma>/area-personal.json` y, al abrir «Mis preferencias»,
 * en la sección `areaPersonal` de `textos/<idioma>/preferencias.json`; se leen con
 * el lector común (`comun/textos.js`). Las pantallas usan claves planas
 * `areaPersonal.<sección>.<clave>`: este módulo aplana el catálogo anidado. En
 * una sección, la clave `_` es el mensaje de la propia sección cuando esta
 * también tiene mensajes hijos (`…explicacion` y `…explicacion.disponible`).
 *
 * El idioma sigue a la URL (`?lang=`), después a la preferencia guardada de la
 * persona y por último al navegador (`comun/idioma.js`). Mientras no se cargue
 * el catálogo elegido se reintenta y se usa el del idioma por defecto si falla.
 */
import { IDIOMA_POR_DEFECTO, leerRecursoJSON, localizacionDe, prepararIdiomas, reintentarIdiomas,
  resolverIdiomaNavegacion } from "../comun/idioma.js";
import { cargarTextos, esMensajePlural } from "../comun/textos.js";

const PREFIJO = "areaPersonal";
const PATRON_VARIABLE = /\{([A-Za-z_][A-Za-z0-9_]*)\}/gu;

function aplanar(seccion, prefijo, salida = {}) {
  for (const [clave, valor] of Object.entries(seccion)) {
    const ruta = clave === "_" ? prefijo : `${prefijo}.${clave}`;
    if (typeof valor === "string" || esMensajePlural(valor)) salida[ruta] = valor;
    else aplanar(valor, ruta, salida);
  }
  return salida;
}

async function cargarCatalogo(idioma, { leer, pantalla = "" } = {}) {
  // El transporte común reintenta errores de red/503; el lector inyectado en
  // pruebas o un adaptador también recibe un segundo intento.
  const leerConReintento = leer ? async (url) => {
    try { return await leer(url); }
    catch { return leer(url); }
  } : leerRecursoJSON;
  const opciones = { idioma, porDefecto: idioma, leer: leerConReintento, avisar: () => {} };
  const reutilizarArea = !leer && activo.idioma === idioma && Object.keys(activo.entradas).length > 0;
  const propios = reutilizarArea ? null : await cargarTextos("area-personal", opciones);
  const preferencias = pantalla === "preferencias" ? await cargarTextos("preferencias", opciones) : null;
  return Object.freeze({
    idioma,
    entradas: Object.freeze({ ...(preferencias ? aplanar(preferencias.seccion(PREFIJO), PREFIJO) : {}),
      ...(propios ? aplanar(propios.mensajes, PREFIJO) : activo.entradas) }),
    preferencias,
  });
}

let activo = Object.freeze({ idioma: IDIOMA_POR_DEFECTO, entradas: Object.freeze({}), preferencias: null });
let secuenciaCarga = 0;

function interpolar(plantilla, variables) {
  return plantilla.replace(PATRON_VARIABLE, (_coincidencia, nombre) => String(variables?.[nombre] ?? ""));
}

/** Idioma de los textos mostrados. */
export function idiomaActivoAreaPersonal() {
  return activo.idioma;
}

/** Catálogo ya leído al abrir Preferencias, compartido con Imagen y Correos. */
export function textosPreferenciasAreaPersonal() {
  return activo.preferencias;
}

/** Localización Intl del idioma de los textos mostrados (fechas, cifras). */
export function localizacionAreaPersonal() {
  return localizacionDe(activo.idioma);
}

/**
 * Mensaje `areaPersonal.…` con sus variables `{nombre}`. Un mensaje plural se
 * elige con `variables.cuenta`, que se muestra con la localización activa.
 * Una clave desconocida se devuelve tal cual.
 */
export function traducir(clave, variables = {}) {
  const mensaje = activo.entradas[clave];
  if (typeof mensaje === "string") return interpolar(mensaje, variables);
  if (esMensajePlural(mensaje)) {
    const localizacion = localizacionAreaPersonal();
    const cantidad = Number(variables.cuenta);
    const plantilla = mensaje[new Intl.PluralRules(localizacion).select(cantidad)] ?? mensaje.other;
    return interpolar(plantilla, { ...variables, cuenta: new Intl.NumberFormat(localizacion).format(cantidad) });
  }
  return clave;
}

const CLAVES_ERROR_CARGA = Object.freeze({
  autenticacion_requerida: Object.freeze({ titulo: "autenticacion.titulo", detalle: "autenticacion.detalle" }),
  acceso_denegado: Object.freeze({ titulo: "acceso.titulo", detalle: "acceso.detalle", garantia: "acceso.garantia", reintentar: false }),
  recurso_no_encontrado: Object.freeze({ titulo: "recurso.titulo", detalle: "recurso.detalle" }),
  servicio_no_disponible: Object.freeze({ detalle: "servicio.detalle" }),
});
export function textosErrorCargaAreaPersonal(error) {
  const base = "areaPersonal.estado.error.";
  const claves = Object.hasOwn(CLAVES_ERROR_CARGA, error?.codigo) ? CLAVES_ERROR_CARGA[error.codigo] : {};
  const detalle = error?.codigo === "servicio_no_disponible" && error?.cause ? "red.detalle" : claves.detalle ?? "detalle";
  return Object.freeze({
    titulo: traducir(`${base}${claves.titulo ?? "titulo"}`), detalle: traducir(`${base}${detalle}`),
    garantia: traducir(`${base}${claves.garantia ?? "carga.garantia"}`),
    reintentar: claves.reintentar === false ? "" : traducir(`${base}reintentar`),
  });
}

/** Idioma de la interfaz: URL, preferencia guardada (`idiomaPreferido`) y navegador, por ese orden. */
export function idiomaAreaPersonal(preferidos = globalThis.navigator?.languages ?? [], ubicacion = globalThis.location, idiomaPreferido = "navegador") {
  return resolverIdiomaNavegacion({ ubicacion, idiomaPreferido, navegador: { languages: preferidos } });
}

/**
 * Sustituye el texto propio de un elemento sin tocar sus hijos (iconos o
 * marcas decorativas): el primer nodo de texto no vacío o, si no tiene hijos
 * elemento, todo su contenido.
 */
function ponerTexto(elemento, texto) {
  const nodos = elemento.childNodes ? [...elemento.childNodes] : [];
  if (!nodos.some((nodo) => nodo.nodeType === 1)) {
    elemento.textContent = texto;
    return;
  }
  const propio = nodos.find((nodo) => nodo.nodeType === 3 && nodo.data.trim() !== "");
  if (propio) propio.data = texto;
}

const ATRIBUTOS_TRADUCIBLES = Object.freeze([
  ["data-i18n-aria-label", "aria-label"], ["data-i18n-placeholder", "placeholder"], ["data-i18n-alt", "alt"],
]);

/** Aplica un catálogo plano a los `data-i18n*` del documento. */
export function aplicarCatalogoAreaPersonal(documento, entradas) {
  if (!documento?.querySelectorAll || !entradas || typeof entradas !== "object" || Array.isArray(entradas)) return;
  documento.querySelectorAll("[data-i18n]").forEach((elemento) => {
    const texto = entradas[elemento.getAttribute("data-i18n")];
    if (typeof texto === "string") ponerTexto(elemento, texto);
  });
  for (const [marca, atributo] of ATRIBUTOS_TRADUCIBLES) {
    documento.querySelectorAll(`[${marca}]`).forEach((elemento) => {
      const texto = entradas[elemento.getAttribute(marca)];
      if (typeof texto === "string") elemento.setAttribute(atributo, texto);
    });
  }
}

/**
 * Carga el catálogo del idioma elegido, lo aplica al documento y devuelve el
 * idioma efectivo. Si el catálogo no puede leerse se conserva el del idioma
 * por defecto. `leer` sustituye al lector de ficheros (pruebas).
 */
export async function iniciarI18nAreaPersonal(documento = globalThis.document, {
  preferidos = globalThis.navigator?.languages ?? [], ubicacion = globalThis.location, idiomaPreferido = "navegador", leer,
  pantalla,
} = {}) {
  const secuencia = ++secuenciaCarga;
  try { await prepararIdiomas(); }
  catch { try { await reintentarIdiomas(); } catch { /* Se conserva el último idioma válido. */ } }
  if (secuencia !== secuenciaCarga) return activo.idioma;
  const idioma = idiomaAreaPersonal(preferidos, ubicacion, idiomaPreferido);
  const vista = pantalla ?? (ubicacion?.href ? new URL(ubicacion.href).searchParams.get("vista") : "");
  if (!leer && activo.idioma === idioma && Object.keys(activo.entradas).length > 0
    && (vista !== "preferencias" || Object.hasOwn(activo.entradas, "areaPersonal.preferencias.campo.idioma"))) {
    aplicarCatalogoAreaPersonal(documento, activo.entradas);
    if (documento?.documentElement) documento.documentElement.lang = activo.idioma;
    return activo.idioma;
  }
  let siguiente = null;
  try { siguiente = await cargarCatalogo(idioma, { leer, pantalla: vista }); }
  catch {
    try { siguiente = await cargarCatalogo(IDIOMA_POR_DEFECTO, { leer, pantalla: vista }); }
    catch { /* Se conserva el último catálogo válido y el documento no se desmonta. */ }
  }
  if (secuencia !== secuenciaCarga) return activo.idioma;
  if (siguiente) activo = siguiente;
  aplicarCatalogoAreaPersonal(documento, activo.entradas);
  if (documento?.documentElement) documento.documentElement.lang = activo.idioma;
  return activo.idioma;
}
