/**
 * Montaje del panel de cancelación en el detalle del expediente. Vive aparte
 * de la vista para no crecerla: se monta tras los paneles de tramitación y,
 * tras cancelar, vuelve a cargar el expediente conservando el justificante.
 */

import { contextoCancelacionDesdeEstado, montarPanelCancelacion } from "./cancelacion-expediente.js?v=20260926-cancelacion-v1";

export function montarCancelacionSiProcede({ raiz, cliente, mensajes, locale, anunciar, confirmarOperacion, recargar } = {}) {
  const disponible = typeof cliente?.consultarCancelacion === "function" && typeof cliente?.cancelarExpediente === "function";
  let desmontar = null;
  let avisoPendiente = null;

  function retirar() {
    desmontar?.();
    desmontar = null;
  }

  function montar(estado) {
    const contexto = disponible ? contextoCancelacionDesdeEstado(estado) : null;
    const zona = raiz?.querySelector?.(".ct-exp-contenido");
    if (!contexto || !zona) return;
    const contenedor = raiz.ownerDocument.createElement("div");
    contenedor.setAttribute("data-ct-exp-cancelacion", "");
    zona.append(contenedor);
    const avisoInicial = avisoPendiente?.expediente_ref === contexto.expediente_ref ? avisoPendiente.aviso : null;
    avisoPendiente = null;
    try {
      desmontar = montarPanelCancelacion({
        contenedor, cliente, contexto, mensajes, locale, anunciar, confirmarOperacion, avisoInicial,
        alConfirmar: async (_recibo, aviso) => {
          avisoPendiente = aviso ? { expediente_ref: contexto.expediente_ref, aviso } : null;
          try {
            await recargar?.(contexto.expediente_ref);
          } catch { /* el justificante ya se mostró; el detalle se actualiza al volver */ }
        },
      });
    } catch {
      contenedor.remove();
    }
  }

  return Object.freeze({ montar, retirar });
}
