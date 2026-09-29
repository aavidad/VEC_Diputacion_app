import { validarPreparacionCoberturaVigente } from "./contrato-cobertura.js";

export const RUTA_PREPARACION_COBERTURA_VIGENTE =
  "/api/vec/contratacion-temporal/cobertura/preparacion-vigente";
const MAXIMO_RESPUESTA_PREPARACION_BYTES = 256 * 1024;

export function crearPreparacionCoberturaClienteHTTP({ ejecutar, validarOpciones } = {}) {
  if (typeof ejecutar !== "function" || typeof validarOpciones !== "function") {
    throw new TypeError("dependencias HTTP de preparación de cobertura no válidas");
  }
  async function obtenerPreparacionCoberturaVigente(opciones) {
    const { signal } = validarOpciones(opciones);
    return ejecutar({
      metodo: "GET", ruta: RUTA_PREPARACION_COBERTURA_VIGENTE,
      signal, estadoEsperado: 200,
      maximoRespuesta: MAXIMO_RESPUESTA_PREPARACION_BYTES,
      validarRespuesta: validarPreparacionCoberturaVigente,
      efecto: false,
    });
  }
  return Object.freeze({ obtenerPreparacionCoberturaVigente });
}
