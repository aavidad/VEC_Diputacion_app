/** Ruta compartible del único recuento de Inicio que el cuadro V1 filtra completo. */
export const PARAMETRO_FILTRO_CT = "ct_mostrar";
export const FILTRO_INCIDENCIA_CT = Object.freeze({ mostrar: "incidencia" });
const FILTRO_NO_SOPORTADO = Object.freeze({ mostrar: "__filtro_no_soportado__" });

export function esFiltroIncidenciaCT(filtro) {
  return filtro !== null && typeof filtro === "object" && !Array.isArray(filtro)
    && filtro.mostrar === "incidencia"
    && Object.entries(filtro).every(([clave, valor]) => clave === "mostrar"
      || ["texto", "fase", "centro", "categoria"].includes(clave) && valor === "");
}

export function leerFiltroCTDeRuta(busqueda) {
  const parametros = new URLSearchParams(busqueda);
  if (!parametros.has(PARAMETRO_FILTRO_CT)) return null;
  const valores = parametros.getAll(PARAMETRO_FILTRO_CT);
  return valores.length === 1 && valores[0] === "incidencia"
    ? FILTRO_INCIDENCIA_CT : FILTRO_NO_SOPORTADO;
}

export function limpiarFiltroCTDeBusqueda(busqueda) {
  const parametros = new URLSearchParams(busqueda);
  parametros.delete(PARAMETRO_FILTRO_CT);
  const resultado = parametros.toString();
  return resultado ? `?${resultado}` : "";
}

export function rutaPortalConFiltroCT(ubicacion, hash, filtro = null) {
  if (!ubicacion || typeof ubicacion.pathname !== "string" || !ubicacion.pathname.startsWith("/")
    || ubicacion.pathname.startsWith("//") || /[\\?#]/u.test(ubicacion.pathname)
    || typeof ubicacion.search !== "string" || typeof hash !== "string" || !/^#[a-z][a-z0-9/-]*$/u.test(hash)
    || (filtro !== null && filtro !== "incidencia")) throw new TypeError("ruta CT no válida");
  const parametros = new URLSearchParams(limpiarFiltroCTDeBusqueda(ubicacion.search));
  if (filtro === "incidencia") parametros.set(PARAMETRO_FILTRO_CT, "incidencia");
  const busqueda = parametros.toString();
  return `${ubicacion.pathname}${busqueda ? `?${busqueda}` : ""}${hash}`;
}
