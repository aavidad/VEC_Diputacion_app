/** Idioma de la interfaz: navegación, preferencia explícita del servidor, navegador. */
export function seleccionarIdioma(parametro = "", preferencias = [], idiomaPreferido = "navegador") {
  const solicitado = String(parametro ?? "").toLowerCase();
  if (solicitado === "es" || solicitado === "en") return solicitado;
  if (idiomaPreferido === "es" || idiomaPreferido === "en") return idiomaPreferido;
  for (const preferencia of preferencias ?? []) {
    const base = String(preferencia ?? "").toLowerCase().split("-", 1)[0];
    if (base === "es" || base === "en") return base;
  }
  return "es";
}

function parametroActual(ubicacion) {
  try { return new URL(ubicacion.href).searchParams.get("lang"); }
  catch { return ""; }
}

/** Resuelve cada navegación con los datos recibidos, sin guardar el resultado. */
export function resolverIdiomaNavegacion({
  ubicacion = globalThis.location,
  idiomaPreferido = "navegador",
  navegador = globalThis.navigator,
} = {}) {
  return seleccionarIdioma(parametroActual(ubicacion), navegador?.languages ?? (navegador?.language ? [navegador.language] : []), idiomaPreferido);
}

const ubicacionActual = globalThis.location;
export const IDIOMA_ACTUAL = resolverIdiomaNavegacion({ ubicacion: ubicacionActual });
export const LOCALIZACION_ACTUAL = IDIOMA_ACTUAL === "en" ? "en-GB" : "es-ES";

/** Conserva ruta, parámetros ajenos y ancla al cambiar el idioma. */
export function cambiarIdioma(idioma, ubicacion = globalThis.location) {
  if (idioma !== "es" && idioma !== "en") return false;
  if (!ubicacion?.href || typeof ubicacion.assign !== "function") return false;
  const destino = new URL(ubicacion.href);
  destino.searchParams.set("lang", idioma);
  ubicacion.assign(destino.href);
  return true;
}

/** Conecta un selector nativo al idioma de la interfaz. */
export function montarSelectorIdioma(selector, ubicacion = globalThis.location, idiomaPreferido = "navegador") {
  if (!selector?.addEventListener) return false;
  selector.value = resolverIdiomaNavegacion({ ubicacion, idiomaPreferido });
  selector.addEventListener("change", () => { cambiarIdioma(selector.value, ubicacion); });
  return true;
}
