/** Ronda de firma del informe nuevo tras subsanar un reparo. Sin dependencias
 * para que la ficha y el circuito de firma la compartan sin ciclos. El
 * servidor vuelve a decidir todo; aquí solo se evita ofrecer acciones o
 * presentar como firmado lo que pertenece a la ronda anterior. */

const ACCION_INFORME = "contratacion_temporal.informe_juridico.generar";

/** El último hito es la emisión del informe nuevo en la versión actual y el
 * expediente sigue en subsanación (mismo criterio que el original firmable del
 * servidor: la fase del resumen es el destino de ese último hito). */
export function informeNuevoVigenteEnSubsanacion(estado) {
  const expediente = estado?.expediente;
  const hito = expediente?.historial?.at?.(-1);
  if (!hito || hito.accion_clave !== ACCION_INFORME || hito.version_expediente !== expediente.version
    || hito.secuencia !== expediente.version || !Array.isArray(estado?.cuadro?.expedientes)) return false;
  const resumen = estado.cuadro.expedientes.find(({ expediente_ref: ref }) => ref === expediente.expediente_ref);
  return resumen?.version === expediente.version && resumen.fase_clave === "subsanacion_unidad"
    && resumen.estado_clave === "incidencia";
}

/** Versión desde la que cuentan las firmas del informe: la de su última
 * emisión cuando sustituye a otra anterior; 0 si solo hubo una. */
export function inicioRondaInformeNuevo(estado) {
  const historial = estado?.expediente?.historial;
  if (!Array.isArray(historial)) return 0;
  const emisiones = historial.filter((h) => h?.accion_clave === ACCION_INFORME && Number.isSafeInteger(h.version_expediente));
  return emisiones.length > 1 ? emisiones.at(-1).version_expediente : 0;
}
