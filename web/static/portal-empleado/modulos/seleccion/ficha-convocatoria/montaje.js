import { cargarTextosFicha } from "./i18n.js?v=20261001-s1-ficha-v1";
import { crearLectorFichaHTTP } from "./cliente-http.js?v=20261001-s1-ficha-v1";
import { montarFichaConvocatoria } from "./vista.js?v=20261001-s1-ficha-v1";

/** Handle inmediato: el shell puede desmontar mientras se carga el catálogo. */
export function montarFichaConvocatoriaHTTP(contenedor, {
  selector, alVolver, lector = crearLectorFichaHTTP(), idioma, prepararTextos = cargarTextosFicha,
} = {}) {
  let cerrada = false, vista;
  const montada = Promise.resolve().then(() => prepararTextos(idioma === undefined ? {} : { idioma })).then((textos) => {
    if (cerrada) return;
    vista = montarFichaConvocatoria(contenedor, { lector, textos, alVolver });
  });
  const consultar = async (entrada) => { await montada; if (!cerrada) await vista.consultar(entrada); };
  const ready = selector === undefined ? montada : consultar(selector);
  return Object.freeze({
    ready, consultar,
    desmontar() { if (cerrada) return; cerrada = true; vista?.desmontar(); },
  });
}
