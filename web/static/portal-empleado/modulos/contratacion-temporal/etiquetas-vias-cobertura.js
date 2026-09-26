/**
 * Etiquetas de las vías de cobertura publicadas en el catálogo de reglas
 * (entradas «c17.via_cobertura.<clave>», duda 7). Una vía nueva del catálogo
 * se nombra con su etiqueta; sin catálogo se usan los textos de siempre.
 */
import { crearCliente } from "../../reglas/reglas.js?v=20260926-integracion-bolsa-ct-v1";

export const PREFIJO_VIA_COBERTURA = "c17.via_cobertura.";
const CATALOGO_REGLAS_CT = "vec.contratacion_temporal.reglas";

/** Mapa clave de vía → etiqueta a partir de la respuesta de reglas vigentes. */
export function etiquetasViasDesdeReglas(datos) {
  const mapa = new Map();
  for (const catalogo of Array.isArray(datos?.catalogos) ? datos.catalogos : []) {
    if (catalogo?.catalogo_id !== CATALOGO_REGLAS_CT || catalogo.estado !== "disponible" || !Array.isArray(catalogo.reglas)) continue;
    for (const regla of catalogo.reglas) {
      if (typeof regla?.clave !== "string" || !regla.clave.startsWith(PREFIJO_VIA_COBERTURA)) continue;
      const via = regla.clave.slice(PREFIJO_VIA_COBERTURA.length);
      if (via !== "" && typeof regla.etiqueta === "string" && regla.etiqueta.trim() !== "") mapa.set(via, regla.etiqueta.trim());
    }
  }
  return mapa;
}

let pendiente = null;

/**
 * Consulta una vez las reglas vigentes y devuelve el mapa de etiquetas. Un
 * fallo no bloquea el formulario: devuelve un mapa vacío y deja reintentar.
 */
export function cargarEtiquetasViasCobertura(cliente = crearCliente()) {
  if (pendiente === null) {
    pendiente = (async () => {
      try {
        return etiquetasViasDesdeReglas(await cliente.reglas());
      } catch {
        pendiente = null;
        return new Map();
      }
    })();
  }
  return pendiente;
}
