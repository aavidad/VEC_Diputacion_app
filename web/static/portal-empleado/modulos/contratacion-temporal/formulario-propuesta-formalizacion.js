/** Orientación de la pantalla RRHH 13 hacia la única acción real CT65. */
import { escaparHTML } from "./componentes-expedientes.js";

// La pantalla original pide generación automática posterior a la selección.
// Esta pieza no remite a destinatarios, no selecciona documentos ni afirma
// firma o entrega: presenta el antecedente aceptado y deja la confirmación al
// formulario CT65 ya existente.
export function renderizarResumenPropuestaFormalizacion(aceptacion, t) {
  if (!aceptacion || aceptacion.respuesta !== "aceptacion") return "";
  return `<section class="ct-exp-mensaje ct-tono-informacion"
    data-ct-propuesta-formalizacion-resumen aria-labelledby="ct-propuesta-formalizacion-titulo">
    <p class="sobrelinea">${escaparHTML(t("llamamiento_propuesta_pantalla_sobrelinea"))}</p>
    <h3 id="ct-propuesta-formalizacion-titulo">${escaparHTML(t("llamamiento_propuesta_pantalla_titulo"))}</h3>
    <p>${escaparHTML(t("llamamiento_propuesta_pantalla_ayuda"))}</p>
    <dl class="ct-resumen"><div><dt>${escaparHTML(t("llamamiento_resolucion_llamamiento_aceptada_ref"))}</dt>
      <dd><code>${escaparHTML(aceptacion.resolucion_ref)}</code></dd></div>
      <div><dt>${escaparHTML(t("llamamiento_recibo_resolucion_aceptada_ref"))}</dt>
      <dd><code>${escaparHTML(aceptacion.recibo_local_ref)}</code></dd></div></dl>
    <p>${escaparHTML(t("llamamiento_propuesta_pantalla_limite"))}</p>
  </section>`;
}
