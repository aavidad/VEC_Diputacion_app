import { MENSAJES_CRONOS_C6_ES } from "./i18n-c6.js?v=20260925-aspecto-v1";

function escaparHTML(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}

/** El catálogo requiere una fuente versionada; la vista nunca fabrica tipos ni cuantías. */
export function renderizarCatalogoPermisosCronos({ mensajes = MENSAJES_CRONOS_C6_ES } = {}) {
  const t = (clave) => escaparHTML(mensajes[clave] ?? MENSAJES_CRONOS_C6_ES[clave]);
  return `<section class="panel cronos-c6" aria-labelledby="cronos-c6-titulo" data-cronos-c6-estado="no_configurado">
    <header class="cabecera-panel"><h3 id="cronos-c6-titulo">${t("titulo")}</h3><span class="cronos-c6-estado" role="status">${t("estado")}</span></header>
    <div class="cuerpo-panel"><div class="cronos-c6-tabla"><table><thead><tr>
      <th scope="col">${t("tipo")}</th><th scope="col">${t("unidad")}</th><th scope="col">${t("computo")}</th><th scope="col">${t("maximo")}</th><th scope="col">${t("minimo")}</th><th scope="col">${t("autorizador")}</th><th scope="col">${t("justificante")}</th>
    </tr></thead><tbody><tr><td colspan="7" role="status">${t("vacio")}</td></tr></tbody></table></div>
      <button type="button" class="boton-primario" disabled aria-disabled="true" aria-describedby="cronos-c6-sin-solicitud">${t("solicitar")}</button>
      <p id="cronos-c6-sin-solicitud" class="cronos-c6-motivo">${t("sin_solicitud")}</p>
    </div>
  </section>`;
}

export function montarCatalogoPermisosCronos({ raiz, registrarDesmontar, mensajes = MENSAJES_CRONOS_C6_ES } = {}) {
  if (!raiz?.append || !raiz.ownerDocument?.createElement || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) {
    throw new TypeError("contenedor de catálogo de Cronos no disponible");
  }
  const contenedor = raiz.ownerDocument.createElement("div");
  contenedor.innerHTML = renderizarCatalogoPermisosCronos({ mensajes });
  raiz.append(contenedor);
  let activo = true;
  const desmontar = () => {
    if (!activo) return;
    activo = false;
    contenedor.remove?.();
  };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar });
}
