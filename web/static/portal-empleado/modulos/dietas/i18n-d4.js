import { crearTraductorDietas, MENSAJES_DIETAS_ES } from "./i18n.js?v=20260925-aspecto-v1";

/** Textos del corte D4; las claves anteriores siguen en el catálogo común. */
export const MENSAJES_DIETAS_D4_ES = Object.freeze({ ...Object.fromEntries([
  "d4_vehiculo_pendiente",
  "d4_vehiculo_ayuda",
  "d4_alternativa_ayuda",
  "d4_previsualizar",
  "d4_motivo_alternativa_ayuda",
  "d4_motivo_alternativa_error",
  "d4_ajustes_pendientes",
  "d4_ajuste_no_disponible",
  "d4_km_sin_ajuste",
  "d4_sin_importe",
].map((clave) => [clave, MENSAJES_DIETAS_ES[clave]])),
  d4_alternativa_estado: "Previsualización sin guardar ni generar importe",
  d4_motivo_alternativa_etiqueta: "Motivo obligatorio (8–500 caracteres)",
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
