import { crearClienteCategorias } from "./cliente.js?v=20261010-ct-b2-catalogo-registro-cohorte-v11";
import { TEXTOS_CATEGORIAS, t } from "./i18n.js?v=20261001-rpt-categorias-v1";
import { montarVistaCategorias } from "./vista.js?v=20261008-hz8-idioma-v1";
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
