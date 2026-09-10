import { validarFichaGINPIX, validarReciboV2ParaFichaGINPIX } from "./contrato-ficha-ginpix.js";
export const RUTA_FICHA_GINPIX = "/api/vec/contratacion-temporal/incorporaciones-ejercicio/ficha-ginpix";
export const NOMBRE_FICHA_GINPIX = "ficha-ginpix-ejercicio.json";
export const CLAVES_ERROR_FICHA_GINPIX = Object.freeze(["acceso_denegado", "recibo_no_confirmado", "servicio_no_disponible"]);
export function crearFichaGINPIXClienteHTTP({ descargar, validarOpciones = (opciones) => opciones ?? {} } = {}) {
  if (typeof descargar !== "function" || typeof validarOpciones !== "function") throw new TypeError("cliente de ficha GINPIX no disponible");
  return Object.freeze({ async descargarFichaGINPIX(recibo, opciones) { const r = validarReciboV2ParaFichaGINPIX(recibo); const { signal } = validarOpciones(opciones); const resultado = await descargar({ metodo: "GET", ruta: `${RUTA_FICHA_GINPIX}?expediente_ref=${encodeURIComponent(r.expediente_ref)}`, signal, efecto: false, maximoRespuesta: 512 * 1024, tipoRespuesta: "application/json", disposition: `attachment; filename=${NOMBRE_FICHA_GINPIX}`, errores: CLAVES_ERROR_FICHA_GINPIX }); if (!resultado || typeof resultado !== "object" || resultado.tipo !== "application/json" || resultado.nombre !== NOMBRE_FICHA_GINPIX || !(resultado.json && typeof resultado.json === "object") || !(resultado.contenido instanceof Uint8Array)) throw new TypeError("respuesta de ficha GINPIX no válida"); return Object.freeze({ ...resultado, json: validarFichaGINPIX(resultado.json, r) }); } });
}
