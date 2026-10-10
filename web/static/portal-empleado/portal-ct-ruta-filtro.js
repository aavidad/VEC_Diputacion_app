/** URL del cuadro CT V1: filtros que el servidor aplica al conjunto completo. */
const PARAMETROS = Object.freeze({ texto: "ct_texto", estado_clave: "ct_estado", fase_clave: "ct_fase" });
const ESTADOS = new Set(["", "incidencia", "espera_externa"]);
const FASES = new Set(["", "solicitud", "analisis", "preparacion", "fiscalizacion",
  "llamamiento", "nombramiento", "incorporacion", "cierre"]);
const TEXTO = /^[0-9A-Za-zÁÉÍÓÚÜÑáéíóúüñ/._ -]{0,80}$/u;

export const FILTRO_INCIDENCIA_CT = Object.freeze({ texto: "", estado_clave: "incidencia", fase_clave: "" });
export const FILTRO_CT_NO_SOPORTADO = Object.freeze({ codigo: "filtro_servidor_no_disponible" });

export function filtroServidorCTValido(filtro) {
  return filtro !== null && typeof filtro === "object" && !Array.isArray(filtro)
    && Object.keys(filtro).every((clave) => Object.hasOwn(PARAMETROS, clave))
    && typeof filtro.texto === "string" && filtro.texto === filtro.texto.trim() && TEXTO.test(filtro.texto)
    && ESTADOS.has(filtro.estado_clave) && FASES.has(filtro.fase_clave);
}

export function leerFiltroCTDeRuta(busqueda) {
  const parametros = new URLSearchParams(busqueda);
  const clavesCT = [...parametros.keys()].filter((clave) => clave.startsWith("ct_"));
  if (clavesCT.length === 0) return null;
  if (clavesCT.some((clave) => !Object.values(PARAMETROS).includes(clave))
    || Object.values(PARAMETROS).some((clave) => parametros.getAll(clave).length > 1))
    return FILTRO_CT_NO_SOPORTADO;
  const filtro = Object.freeze({ texto: parametros.get(PARAMETROS.texto) ?? "",
    estado_clave: parametros.get(PARAMETROS.estado_clave) ?? "",
    fase_clave: parametros.get(PARAMETROS.fase_clave) ?? "" });
  return filtroServidorCTValido(filtro) ? filtro : FILTRO_CT_NO_SOPORTADO;
}

export function limpiarFiltroCTDeBusqueda(busqueda) {
  const parametros = new URLSearchParams(busqueda);
  for (const clave of [...parametros.keys()]) if (clave.startsWith("ct_")) parametros.delete(clave);
  const resultado = parametros.toString();
  return resultado ? `?${resultado}` : "";
}

export function rutaPortalConFiltroCT(ubicacion, hash, filtro = null, { conservarLlamamiento = false } = {}) {
  if (!ubicacion || typeof ubicacion.pathname !== "string" || !ubicacion.pathname.startsWith("/")
    || ubicacion.pathname.startsWith("//") || /[\\?#]/u.test(ubicacion.pathname)
    || typeof ubicacion.search !== "string" || typeof hash !== "string" || !/^#[a-z][a-z0-9/-]*$/u.test(hash)
    || (filtro !== null && !filtroServidorCTValido(filtro))) throw new TypeError("ruta CT no válida");
  const parametros = new URLSearchParams(limpiarFiltroCTDeBusqueda(ubicacion.search));
  if (hash !== "#contratacion-temporal") {
    parametros.delete("expediente");
    parametros.delete("expediente_version");
  }
  if (hash !== "#bolsa/bolsa-candidatos" && !(hash === "#bolsa/llamamientos" && conservarLlamamiento)) {
    if (parametros.has("bolsa_ref")) for (const clave of ["bolsa_ref", "estado", "cursor"]) parametros.delete(clave);
    for (const clave of ["origen_expediente", "origen_referencia", "origen_centro", "origen_inicio"]) parametros.delete(clave);
  }
  if (filtro) for (const [campo, parametro] of Object.entries(PARAMETROS)) {
    if (filtro[campo]) parametros.set(parametro, filtro[campo]);
  }
  const busqueda = parametros.toString();
  return `${ubicacion.pathname}${busqueda ? `?${busqueda}` : ""}${hash}`;
}
