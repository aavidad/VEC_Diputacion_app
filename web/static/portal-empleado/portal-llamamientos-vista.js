/** Presentación HTML pura del corte de llamamientos, sin red ni estado global. */

function escaparHTML(valor) {
  return String(valor ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

export function renderizarPasosLlamamiento({ modoPresentacion, pasoActual }) {
  const nombres = modoPresentacion
    ? ["Elegir necesidad de demostración", "Revisar demostración", "Configurar demostración", "Comprobar presentación"]
    : ["Elegir necesidad", "Revisar propuesta", "Configurar llamamiento", "Revisar y preparar"];
  const actualSeguro = modoPresentacion ? pasoActual : 1;
  return `<nav class="pasos" aria-label="Pasos del nuevo llamamiento">${nombres.map((nombre, indice) => {
    const paso = indice + 1;
    const clase = paso < actualSeguro ? "completado" : "";
    const actual = paso === actualSeguro ? ' aria-current="step"' : "";
    const bloqueado = !modoPresentacion && paso > 1 ? " disabled" : "";
    return `<button type="button" class="paso ${clase}" data-accion="ir-paso" data-paso="${paso}"${actual}${bloqueado}><span class="paso-numero">${paso < actualSeguro ? "✓" : paso}</span><span>${escaparHTML(nombre)}</span></button>`;
  }).join("")}</nav>`;
}

export function renderizarConfirmacionCompacta(confirmacion) {
  return `<div class="cuerpo-panel"><dl class="resumen-expediente">
    <div class="fila-resumen"><dt>Propuesta</dt><dd><code>${escaparHTML(confirmacion.propuesta_ref)}</code></dd></div>
    <div class="fila-resumen"><dt>Necesidad</dt><dd><code>${escaparHTML(confirmacion.necesidad.referencia)}</code> · v${escaparHTML(confirmacion.necesidad.version)}</dd></div>
    <div class="fila-resumen"><dt>Bolsa</dt><dd><code>${escaparHTML(confirmacion.bolsa.referencia)}</code> · v${escaparHTML(confirmacion.bolsa.version)}</dd></div>
    <div class="fila-resumen"><dt>Instantánea</dt><dd><code>${escaparHTML(confirmacion.instantanea.referencia)}</code> · v${escaparHTML(confirmacion.instantanea.version)}</dd></div>
    <div class="fila-resumen"><dt>Política</dt><dd><code>${escaparHTML(confirmacion.politica.referencia)}</code> · v${escaparHTML(confirmacion.politica.version)}</dd></div>
    <div class="fila-resumen"><dt>Generada</dt><dd>${escaparHTML(confirmacion.generada_en)}</dd></div>
    <div class="fila-resumen"><dt>Evaluaciones</dt><dd>${escaparHTML(confirmacion.total_evaluaciones)}</dd></div>
    <div class="fila-resumen"><dt>Orden seleccionado</dt><dd>${escaparHTML(confirmacion.orden_seleccionado)}</dd></div>
  </dl><div class="nota-pendiente" role="status"><strong>Detalle no disponible</strong><br>La respuesta no contiene evaluaciones ni datos personales. Los pasos de configuración permanecen bloqueados.</div></div>`;
}

export function renderizarDetalleLlamamientoBloqueado() {
  return '<section class="panel"><div class="cabecera-panel"><h3>Detalle no disponible</h3><span class="estado-chip">Bloqueado</span></div><div class="vacio-controlado">La confirmación real no contiene detalle y no habilita la configuración del llamamiento.</div><div class="cuerpo-panel"><button type="button" class="boton-secundario" data-accion="anterior-paso">← Volver</button></div></section>';
}
