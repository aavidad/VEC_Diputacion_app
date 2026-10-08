import { crearClienteHTTPContratacionTemporal } from "../modulos/contratacion-temporal/cliente-http.js?v=20261008-alta-rechazo-v2";
import { crearPuertoCategorias } from "./puerto.js?v=20261001-rpt-categorias-v1";

export const PLAZO_CONSULTA_MS = 10000;

/** Proyecta únicamente las opciones del alta; no deduce estado, versión ni uso. */
export function crearClienteCategorias({ clienteAlta = crearClienteHTTPContratacionTemporal(), plazoMs = PLAZO_CONSULTA_MS } = {}) {
  if (typeof clienteAlta?.obtenerCatalogosAlta !== "function"
    || !Number.isSafeInteger(plazoMs) || plazoMs < 1) {
    throw new TypeError("cliente de categorías no disponible");
  }
  return crearPuertoCategorias({
    async listarOpciones({ signal } = {}) {
      if (signal !== undefined && !(signal instanceof AbortSignal)) {
        throw new TypeError("cancelación no válida");
      }
      const controlador = new AbortController();
      const cancelar = () => controlador.abort();
      signal?.addEventListener("abort", cancelar, { once: true });
      const temporizador = setTimeout(cancelar, plazoMs);
      try {
        if (signal?.aborted) controlador.abort();
        const catalogos = await clienteAlta.obtenerCatalogosAlta({ signal: controlador.signal });
        if (controlador.signal.aborted) throw new DOMException("", "AbortError");
        // El cliente de Contratación valida y congela el contrato completo.
        return Object.freeze(catalogos.categorias.map((categoria) => Object.freeze({
          referencia: categoria.referencia,
          etiqueta: categoria.etiqueta,
          grupos_subgrupos: Object.freeze(categoria.grupos_subgrupos.map((grupo) => Object.freeze({
            clave: grupo.clave,
            etiqueta: grupo.etiqueta,
          }))),
        })));
      } finally {
        clearTimeout(temporizador);
        signal?.removeEventListener("abort", cancelar);
      }
    },
  });
}
