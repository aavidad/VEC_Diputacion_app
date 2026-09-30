/**
 * Textos del área personal.
 *
 * Viven en `textos/<idioma>/area-personal.json` y, para «Mis preferencias», en
 * la sección `areaPersonal` de `textos/<idioma>/preferencias.json`; se leen con
 * el lector común (`comun/textos.js`). Las pantallas usan claves planas
 * `areaPersonal.<sección>.<clave>`: este módulo aplana el catálogo anidado. En
 * una sección, la clave `_` es el mensaje de la propia sección cuando esta
 * también tiene mensajes hijos (`…explicacion` y `…explicacion.disponible`).
 *
 * El idioma sigue a la URL (`?lang=`), después a la preferencia guardada de la
 * persona y por último al navegador (`comun/idioma.js`). Mientras no se cargue
 * el catálogo elegido se usa el del idioma por defecto.
 */
import { IDIOMA_POR_DEFECTO, localizacionDe, resolverIdiomaNavegacion } from "../comun/idioma.js";
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

async function cargarCatalogo(idioma, opciones = {}) {
  const [propios, preferencias, avisos] = await Promise.all(["area-personal", "preferencias", "avisos-externos"]
    .map((modulo) => cargarTextos(modulo, { ...opciones, idioma })));
  return Object.freeze({
    idioma: propios.idioma,
    entradas: Object.freeze({ ...aplanar(preferencias.seccion(PREFIJO), PREFIJO), ...aplanar(propios.mensajes, PREFIJO),
      ...aplanar(avisos.seccion(PREFIJO), PREFIJO), }),
  });
}

const RESPALDO = await cargarCatalogo(IDIOMA_POR_DEFECTO);
let activo = RESPALDO;

function interpolar(plantilla, variables) {
  return plantilla.replace(PATRON_VARIABLE, (_coincidencia, nombre) => String(variables?.[nombre] ?? ""));
}

/** Idioma de los textos mostrados. */
export function idiomaActivoAreaPersonal() {
  return activo.idioma;
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
  const mensaje = activo.entradas[clave] ?? RESPALDO.entradas[clave];
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
} = {}) {
  const idioma = idiomaAreaPersonal(preferidos, ubicacion, idiomaPreferido);
  try {
    activo = await cargarCatalogo(idioma, leer ? { leer, avisar: () => {} } : {});
  } catch {
    activo = RESPALDO;
  }
  aplicarCatalogoAreaPersonal(documento, activo.entradas);
  if (documento?.documentElement) documento.documentElement.lang = activo.idioma;
  return activo.idioma;
}
