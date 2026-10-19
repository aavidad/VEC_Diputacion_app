import { crearClienteCategorias } from "./cliente.js?v=VEC_WEB_VERSION";
import { TEXTOS_CATEGORIAS, t } from "./i18n.js?v=VEC_WEB_VERSION";
import { montarVistaCategorias } from "./vista.js?v=VEC_WEB_VERSION";
import { IDIOMA_POR_DEFECTO } from "../../comun/idioma.js";

/** Monta la consulta de categorías dentro de la raíz del portal. */
export function montarCategoriasRPT({ raiz }) {
  const vista = montarVistaCategorias({
    raiz,
    puerto: crearClienteCategorias(),
    t,
    idiomaUI: TEXTOS_CATEGORIAS.idioma,
    idiomaDatos: IDIOMA_POR_DEFECTO,
    localizacion: TEXTOS_CATEGORIAS.localizacion,
  });
  void vista.cargar();
  return vista;
}
