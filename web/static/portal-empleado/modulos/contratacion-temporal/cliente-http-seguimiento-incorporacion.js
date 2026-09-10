import { validarConsultaSeguimientoIncorporacion } from "./contrato-seguimiento-incorporacion.js";

export const RUTA_SEGUIMIENTO_INCORPORACION =
  "/api/vec/contratacion-temporal/incorporaciones-ejercicio/seguimiento";

export function crearClienteSeguimientoIncorporacion({ consultar, validarOpciones } = {}) {
  if (typeof consultar !== "function" || typeof validarOpciones !== "function") {
    throw new TypeError("cliente de seguimiento de incorporación no disponible");
  }
  return Object.freeze({
    async consultar(expediente_ref, opciones) {
      const { signal } = validarOpciones(opciones);
      const data = await consultar(expediente_ref, { signal });
      return validarConsultaSeguimientoIncorporacion(data, expediente_ref);
    },
  });
}
