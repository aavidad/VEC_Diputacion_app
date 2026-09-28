/** Etiquetas exclusivas de la muestra ficticia; nunca se infieren nombres reales de referencias. */
const PERSONAS_EJEMPLO = Object.freeze({ per_1: "Carmen Molina" });
const EXPEDIENTES_EJEMPLO = Object.freeze({ exp_1: "EXP-2026-001" });

export function presentarRegistroAuditoria(registro, t, ejemplo = false) {
  const actor = ejemplo && Object.hasOwn(PERSONAS_EJEMPLO, registro.actor_ref)
    ? `${PERSONAS_EJEMPLO[registro.actor_ref]} (${t("dato_ficticio")})` : t("persona_no_disponible");
  const expediente = ejemplo && Object.hasOwn(EXPEDIENTES_EJEMPLO, registro.expediente_ref)
    ? `${EXPEDIENTES_EJEMPLO[registro.expediente_ref]} (${t("dato_ficticio")})` : t("numero_no_disponible");
  const acciones = Object.freeze({ "relacion.actualizada": "accion_relacion_actualizada",
    "bolsa.participacion.cambiar": "accion_participacion_cambiada" });
  const resultados = Object.freeze({ confirmado: "resultado_confirmado", denegado: "resultado_denegado", ok: "resultado_confirmado" });
  const accion = Object.hasOwn(acciones, registro.accion) ? acciones[registro.accion] : null;
  const resultado = Object.hasOwn(resultados, registro.resultado) ? resultados[registro.resultado] : null;
  return Object.freeze({ actor, expediente, accion: t(accion || "accion_otra"), resultado: t(resultado || "resultado_otro") });
}

export function presentarExpedienteAuditoria(ref, t, ejemplo = false) {
  return ejemplo && Object.hasOwn(EXPEDIENTES_EJEMPLO, ref)
    ? `${EXPEDIENTES_EJEMPLO[ref]} (${t("dato_ficticio")})` : t("numero_no_disponible");
}
