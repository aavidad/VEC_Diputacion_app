import { crearTraductorAuditoria } from "./i18n.js";

function escapar(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}

/** Presentación cerrada hasta disponer de una consulta autorizada de Auditoría. */
export function renderizarVistaAuditoria({ estado = "no_configurado", ayudaAbierta = false } = {}) {
  const t = crearTraductorAuditoria();
  const denegado = estado === "denegado";
  const tituloEstado = denegado ? t("estado_denegado") : t("estado_no_configurado");
  const explicacionEstado = denegado ? t("denegado_descripcion") : t("no_configurado_descripcion");
  const tono = denegado ? "peligro" : "aviso";

  return `<section class="modulo-auditoria" data-auditoria-vista>
    <header class="cabecera-vista auditoria-cabecera">
      <div><p class="sobrelinea">${escapar(t("sobrelinea"))}</p><h2>${escapar(t("titulo"))}</h2><p>${escapar(t("descripcion"))}</p></div>
      <button type="button" class="boton-secundario auditoria-ayuda-boton" data-auditoria-ayuda aria-controls="auditoria-ayuda" aria-expanded="${ayudaAbierta}" aria-label="${escapar(t("ayuda_aria"))}">?</button>
    </header>
    <section class="panel auditoria-estado" role="status" aria-labelledby="auditoria-estado-titulo">
      <div class="cabecera-panel"><div><p class="sobrelinea">${escapar(t("estado_operativo"))}</p><h3 id="auditoria-estado-titulo">${escapar(tituloEstado)}</h3></div><span class="estado-chip ${tono}">${escapar(tituloEstado)}</span></div>
      <div class="cuerpo-panel"><p>${escapar(explicacionEstado)}</p></div>
    </section>
    <section id="auditoria-ayuda" class="panel auditoria-ayuda" ${ayudaAbierta ? "" : "hidden"} aria-labelledby="auditoria-ayuda-titulo">
      <div class="cabecera-panel"><h3 id="auditoria-ayuda-titulo">${escapar(t("ayuda_titulo"))}</h3></div>
      <div class="cuerpo-panel"><p>${escapar(t("ayuda_alcance"))}</p><p>${escapar(t("ayuda_lectura"))}</p><p>${escapar(t("ayuda_segregacion"))}</p></div>
    </section>
    <div class="auditoria-espacio">
      <section class="panel auditoria-consulta" aria-labelledby="auditoria-consulta-titulo">
        <div class="cabecera-panel"><div><p class="sobrelinea">${escapar(t("consulta_sobrelinea"))}</p><h3 id="auditoria-consulta-titulo">${escapar(t("consulta_titulo"))}</h3><p>${escapar(t("consulta_subtitulo"))}</p></div></div>
        <div class="cuerpo-panel">
          <ol class="auditoria-condiciones" aria-label="${escapar(t("condiciones_aria"))}">
            <li><span aria-hidden="true">1</span><strong>${escapar(t("condicion_competencia"))}</strong><small>${escapar(t("condicion_competencia_nota"))}</small></li>
            <li><span aria-hidden="true">2</span><strong>${escapar(t("condicion_ambito"))}</strong><small>${escapar(t("condicion_ambito_nota"))}</small></li>
            <li><span aria-hidden="true">3</span><strong>${escapar(t("condicion_lectura"))}</strong><small>${escapar(t("condicion_lectura_nota"))}</small></li>
          </ol>
          <form class="auditoria-filtros" aria-label="${escapar(t("filtros_aria"))}">
            <label>${escapar(t("recurso_exacto"))}<input name="subject_ref" type="text" autocomplete="off" disabled aria-disabled="true" placeholder="${escapar(t("recurso_placeholder"))}"></label>
            <label>${escapar(t("finalidad"))}<input name="finalidad" type="text" autocomplete="off" disabled aria-disabled="true" placeholder="${escapar(t("finalidad_placeholder"))}"></label>
            <label>${escapar(t("periodo"))}<input name="periodo" type="text" autocomplete="off" disabled aria-disabled="true" placeholder="${escapar(t("periodo_placeholder"))}"></label>
            <button type="button" class="boton-primario" disabled aria-disabled="true" title="${escapar(t("consulta_motivo"))}">${escapar(t("consultar"))}</button>
          </form>
          <p class="auditoria-aviso">${escapar(t("consulta_motivo"))}</p>
        </div>
      </section>
      <aside class="panel auditoria-limites" aria-labelledby="auditoria-limites-titulo">
        <div class="cabecera-panel"><div><p class="sobrelinea">${escapar(t("limites_sobrelinea"))}</p><h3 id="auditoria-limites-titulo">${escapar(t("limites_titulo"))}</h3></div></div>
        <div class="cuerpo-panel"><ul><li>${escapar(t("limite_campos"))}</li><li>${escapar(t("limite_separacion"))}</li><li>${escapar(t("limite_exportacion"))}</li></ul>
          <button type="button" class="boton-secundario" disabled aria-disabled="true" title="${escapar(t("exportar_motivo"))}">${escapar(t("exportar"))}</button>
        </div>
      </aside>
    </div>
    <section class="panel auditoria-resultados" aria-labelledby="auditoria-resultados-titulo">
      <div class="cabecera-panel"><div><p class="sobrelinea">${escapar(t("resultados_sobrelinea"))}</p><h3 id="auditoria-resultados-titulo">${escapar(t("resultados_titulo"))}</h3></div><span class="estado-chip ${tono}">${escapar(tituloEstado)}</span></div>
      <div class="cuerpo-panel auditoria-vacio"><span aria-hidden="true" class="auditoria-vacio-icono">∅</span><p><strong>${escapar(denegado ? t("sin_resultados_denegado") : t("sin_resultados"))}</strong></p><p>${escapar(t("sin_resultados_nota"))}</p></div>
    </section>
  </section>`;
}

export function montarVistaAuditoria({ raiz, anunciar = () => {}, registrarDesmontar } = {}) {
  if (!raiz?.replaceChildren || typeof anunciar !== "function" ||
    (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) {
    throw new TypeError("vista de Auditoría no disponible");
  }
  const t = crearTraductorAuditoria();
  let activa = true;
  let ayudaAbierta = false;
  const pintar = () => { if (activa) raiz.innerHTML = renderizarVistaAuditoria({ ayudaAbierta }); };
  const pulsar = (evento) => {
    if (!evento.target.closest?.("[data-auditoria-ayuda]")) return;
    ayudaAbierta = !ayudaAbierta;
    pintar();
    raiz.querySelector("[data-auditoria-ayuda]")?.focus();
    anunciar(t(ayudaAbierta ? "ayuda_abierta" : "ayuda_cerrada"), "info");
  };
  raiz.addEventListener("click", pulsar);
  pintar();
  const desmontar = () => {
    if (!activa) return;
    activa = false;
    raiz.removeEventListener("click", pulsar);
    raiz.replaceChildren();
  };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar });
}
