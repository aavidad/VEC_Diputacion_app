import { crearTraductorDietas } from "./i18n.js";

/** Textos del corte D4; las claves anteriores siguen en el catálogo común. */
export const MENSAJES_DIETAS_D4_ES = Object.freeze({
  d4_vehiculo_pendiente: "Vehículo propio pendiente de confirmar",
  d4_vehiculo_ayuda: "Esta consulta de ruta no registra el medio de transporte. El kilometraje de vehículo propio requiere una comisión y un contrato autorizado.",
  d4_alternativa_ayuda: "Puede previsualizar otra alternativa OSRM. La elección y su motivo solo cambian esta vista; no se guardan ni generan importe.",
  d4_previsualizar: "Previsualizar alternativa",
  d4_motivo_alternativa_ayuda: "Escriba entre 8 y 500 caracteres para previsualizar una ruta distinta de la recomendada.",
  d4_motivo_alternativa_error: "Indique un motivo de entre 8 y 500 caracteres para previsualizar esta alternativa.",
  d4_ajustes_pendientes: "Los ajustes manuales requieren un contrato autorizado que conserve el motivo. Aquí no modifican kilómetros ni importes.",
  d4_ajuste_no_disponible: "Pendiente de contrato",
  d4_km_sin_ajuste: "Sin ajuste",
  d4_sin_importe: "Sin importe por kilómetro aprobado o liquidación en esta consulta.",
});

export function crearTraductorDietasD4(mensajes) {
  const traducirComun = crearTraductorDietas(mensajes);
  return (clave, variables = {}) => {
    if (Object.hasOwn(MENSAJES_DIETAS_D4_ES, clave)) {
      const plantilla = mensajes?.[clave] ?? MENSAJES_DIETAS_D4_ES[clave];
      if (typeof plantilla !== "string" || !plantilla) throw new Error("catálogo i18n de Dietas D4 incompleto");
      return plantilla.replace(/\{([a-z_]+)\}/g, (_coincidencia, variable) => String(variables[variable] ?? ""));
    }
    return traducirComun(clave, variables);
  };
}
