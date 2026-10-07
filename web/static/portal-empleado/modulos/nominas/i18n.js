import { cargarTextos } from "../../../comun/textos.js";
import { IDIOMA_ACTUAL, IDIOMA_POR_DEFECTO } from "../../../comun/idioma.js";

/** Se pide solo el idioma abierto; el respaldo se lee tras dos fallos. */
export async function cargarTextosNominas({ idioma = IDIOMA_ACTUAL, porDefecto = IDIOMA_POR_DEFECTO, leer, avisar } = {}) {
  const cargar = (codigo) => cargarTextos("nominas", { idioma: codigo, porDefecto: codigo, leer, avisar });
  try {
    return await cargar(idioma);
  } catch {
    try {
      return await cargar(idioma);
    } catch (error) {
      if (idioma === porDefecto) throw error;
      return cargar(porDefecto);
    }
  }
}

export function crearTraductorNominas(textos) {
  if (typeof textos?.traducir !== "function" || typeof textos?.fecha !== "function") {
    throw new TypeError("nominas:textos");
  }
  return (clave, valores = {}) => textos.traducir(`general.${clave}`, valores);
}
