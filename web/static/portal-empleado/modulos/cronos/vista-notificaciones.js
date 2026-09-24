import { MENSAJES_CRONOS_C9_ES } from "./i18n-c9.js?v=20260925-tanda-v1";

function escaparHTML(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}

/** Sin transporte ni consulta autorizada no se prepara un borrador ni se inventan mensajes. */
export function renderizarVistaNotificacionesCronos({ mensajes = MENSAJES_CRONOS_C9_ES } = {}) {
  const t = (clave) => escaparHTML(mensajes[clave] ?? MENSAJES_CRONOS_C9_ES[clave]);
  return `<section class="cronos-notificaciones" aria-labelledby="cronos-notificaciones-titulo" data-estado-entrega="no_configurado">
    <article class="panel cronos-notificaciones-panel">
      <header class="cabecera-panel"><h3 id="cronos-notificaciones-titulo">${t("notificaciones_titulo")}</h3><span class="estado-chip neutro" role="status">${t("notificaciones_estado")}</span></header>
      <div class="cuerpo-panel"><button type="button" class="boton-primario" disabled aria-disabled="true" aria-describedby="cronos-notificaciones-sin-envio">${t("notificaciones_enviar")}</button>
        <p id="cronos-notificaciones-sin-envio" class="cronos-notificaciones-motivo">${t("notificaciones_envio_pendiente")}</p></div>
    </article>
    <article class="panel cronos-notificaciones-panel"><header class="cabecera-panel"><h3>${t("notificaciones_mensajes_titulo")}</h3></header>
      <div class="cuerpo-panel"><p role="status">${t("notificaciones_mensajes_pendiente")}</p>
        <button type="button" class="boton-secundario" disabled aria-disabled="true" aria-describedby="cronos-notificaciones-sin-archivo">${t("notificaciones_archivar")}</button>
        <p id="cronos-notificaciones-sin-archivo" class="cronos-notificaciones-motivo">${t("notificaciones_archivo_pendiente")}</p></div></article>
  </section>`;
}

export function montarVistaNotificacionesCronos({ raiz, registrarDesmontar, anunciar = () => {}, mensajes } = {}) {
  if (!raiz?.append || !raiz.ownerDocument?.createElement || typeof anunciar !== "function"
    || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) {
    throw new TypeError("vista de notificaciones de Cronos no disponible");
  }
  const contenedor = raiz.ownerDocument.createElement("section");
  contenedor.innerHTML = renderizarVistaNotificacionesCronos({ mensajes });
  raiz.append(contenedor);
  let activa = true;
  const desmontar = () => {
    if (!activa) return;
    activa = false;
    contenedor.innerHTML = "";
    contenedor.remove?.();
  };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar });
}
