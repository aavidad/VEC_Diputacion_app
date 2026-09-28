/** Idioma de la interfaz. La URL tiene prioridad sobre la preferencia del navegador. */
export function seleccionarIdioma(parametro = "", preferencias = []) {
  const solicitado = String(parametro ?? "").toLowerCase();
  if (solicitado === "es" || solicitado === "en") return solicitado;
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

const ubicacionActual = globalThis.location;
export const IDIOMA_ACTUAL = seleccionarIdioma(parametroActual(ubicacionActual), globalThis.navigator?.languages ?? []);
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
export function montarSelectorIdioma(selector, ubicacion = globalThis.location) {
  if (!selector?.addEventListener) return false;
  selector.value = seleccionarIdioma(parametroActual(ubicacion), globalThis.navigator?.languages ?? []);
  selector.addEventListener("change", () => { cambiarIdioma(selector.value, ubicacion); });
  return true;
}
