/**
 * Idioma de la interfaz: castellano por defecto; inglés solo si la persona lo
 * elige en el selector (parámetro lang). No se sigue el idioma del navegador.
 */
export function seleccionarIdioma(parametro = "", _preferencias = []) {
  const solicitado = String(parametro ?? "").toLowerCase();
  return solicitado === "en" ? "en" : "es";
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
