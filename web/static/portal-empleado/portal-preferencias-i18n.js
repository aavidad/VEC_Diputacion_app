import { IDIOMAS_DISPONIBLES, IDIOMA_POR_DEFECTO } from "../comun/idioma.js";
import { cargarTextos } from "../comun/textos.js";

// El portal-i18n legado aún consume dos exportaciones sincrónicas.
// Ambas proceden del mismo catálogo de datos que usan los módulos nuevos.
const idiomaAlternativo = IDIOMAS_DISPONIBLES.find(({ codigo }) => codigo !== IDIOMA_POR_DEFECTO)?.codigo
  ?? IDIOMA_POR_DEFECTO;
const [textosBase, textosAlternativos] = await Promise.all([
  cargarTextos("preferencias", { idioma: IDIOMA_POR_DEFECTO }),
  cargarTextos("preferencias", { idioma: idiomaAlternativo }),
]);

function aplanarMensajes(seccion, prefijo = "", salida = {}) {
  for (const [clave, valor] of Object.entries(seccion)) {
    const ruta = prefijo ? `${prefijo}.${clave}` : clave;
    if (typeof valor === "string") salida[ruta] = valor;
    else aplanarMensajes(valor, ruta, salida);
  }
  return salida;
}

export const PREFERENCIAS_ES = Object.freeze(aplanarMensajes(textosBase.seccion("portal")));
export const PREFERENCIAS_EN = Object.freeze(aplanarMensajes(textosAlternativos.seccion("portal")));
