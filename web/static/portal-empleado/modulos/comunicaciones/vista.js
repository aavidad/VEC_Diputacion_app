import { renderizarEstadoEntrega } from "../../estado-entrega.js";
import { crearTraductorComunicaciones } from "./i18n.js";

const PESTANAS = Object.freeze([
  ["bandeja", "pestana_bandeja"],
  ["preferencias", "pestana_preferencias"],
  ["plantillas", "pestana_plantillas"],
  ["administrativas", "pestana_administrativas"],
]);
const ETAPAS = Object.freeze(["transporte", "entrega", "lectura"]);

function e(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}
function botonBloqueado(texto, motivo) {
  return `<button type="button" class="boton-secundario" disabled aria-disabled="true" title="${e(motivo)}">${e(texto)}</button>`;
}
function estadoSinFuente(t) {
  return renderizarEstadoEntrega({
    estado: "visual_pendiente_backend",
    resumen: t("sin_fuente_resumen"),
    pendientes: [t("pendiente_fuente"), t("pendiente_canal"), t("pendiente_recibo")],
    conexion: t("conexion_pendiente"),
  });
}
function bandaEstado(t) {
  return `<details class="panel comunicaciones-estado"><summary>${e(t("estado_resumen"))} <span aria-hidden="true">?</span></summary>${estadoSinFuente(t)}</details>`;
}
function bandeja(t) {
  return `<section class="panel comunicaciones-bandeja" aria-labelledby="comunicaciones-bandeja-titulo">
    <div class="cabecera-panel"><div><p class="sobrelinea">${e(t("bandeja_sobrelinea"))}</p><h3 id="comunicaciones-bandeja-titulo">${e(t("bandeja_titulo"))}</h3></div><span class="estado-chip aviso">${e(t("visibles", { numero: 0 }))}</span></div>
    <div class="tabla-contenedor" tabindex="0" role="region" aria-label="${e(t("region_bandeja"))}"><table class="tabla-datos comunicaciones-tabla"><caption>${e(t("caption_bandeja"))}</caption><thead><tr>${["asunto", "canal_previsto", ...ETAPAS.map((etapa) => `evidencia_${etapa}`)].map((clave) => `<th scope="col">${e(t(clave))}</th>`).join("")}</tr></thead><tbody><tr><td colspan="5" class="comunicaciones-vacio">${e(t("bandeja_vacia"))}</td></tr></tbody></table></div>
  </section>`;
}
function evidencias(t) {
  return `<section class="panel comunicaciones-detalle" aria-labelledby="comunicaciones-detalle-titulo"><div class="cabecera-panel"><div><p class="sobrelinea">${e(t("detalle_sobrelinea"))}</p><h3 id="comunicaciones-detalle-titulo">${e(t("trazabilidad_titulo"))}</h3></div><span class="estado-chip aviso">${e(t("no_configurado"))}</span></div>
    <div class="cuerpo-panel"><p class="comunicaciones-explicacion">${e(t("trazabilidad_ayuda"))}</p><dl class="comunicaciones-evidencias">${ETAPAS.map((etapa) => `<div><dt>${e(t(`evidencia_${etapa}`))}</dt><dd><span class="estado-chip aviso">${e(t("sin_fuente"))}</span></dd></div>`).join("")}</dl>
    <p class="comunicaciones-limite">${e(t("limite_detalle"))}</p><div class="acciones-vista">${botonBloqueado(t("enviar"), t("enviar_motivo"))}</div><p class="comunicaciones-motivo">${e(t("enviar_motivo"))}</p></div></section>`;
}
function seccionVacia(t, nombre, titulo, accion, motivo, ayuda) {
  return `<section class="panel" aria-labelledby="comunicaciones-${nombre}-titulo"><div class="cabecera-panel"><div><p class="sobrelinea">${e(t("no_configurado"))}</p><h3 id="comunicaciones-${nombre}-titulo">${e(t(titulo))}</h3></div>${botonBloqueado(t(accion), t(motivo))}</div><div class="cuerpo-panel"><p class="comunicaciones-vacio">${e(t(ayuda))}</p></div></section>`;
}
function administrativas(t) {
  return `<section class="panel comunicaciones-segregadas" aria-labelledby="comunicaciones-administrativas-titulo"><div class="cabecera-panel"><div><p class="sobrelinea">${e(t("no_configurado"))}</p><h3 id="comunicaciones-administrativas-titulo">${e(t("administrativas_titulo"))}</h3></div>${botonBloqueado(t("crear_campana"), t("crear_campana_motivo"))}</div><div class="cuerpo-panel"><p>${e(t("administrativas_ayuda"))}</p><div class="acciones-vista">${botonBloqueado(t("enviar"), t("enviar_motivo"))}</div><p class="comunicaciones-motivo">${e(t("enviar_motivo"))}</p></div></section>`;
}

/** Presentación sin fuente conectada: no renderiza datos de muestra ni ejecuta efectos. */
export function renderizarVistaComunicaciones({ pestana = "bandeja" } = {}) {
  const t = crearTraductorComunicaciones();
  if (!PESTANAS.some(([clave]) => clave === pestana)) pestana = "bandeja";
  const contenido = pestana === "preferencias"
    ? seccionVacia(t, "preferencias", "preferencias_titulo", "cambiar_preferencias", "cambiar_preferencias_motivo", "preferencias_ayuda")
    : pestana === "plantillas"
      ? seccionVacia(t, "plantillas", "plantillas_titulo", "redactar", "redactar_motivo", "plantillas_ayuda")
      : pestana === "administrativas" ? administrativas(t)
        : `<div class="comunicaciones-espacio">${bandeja(t)}${evidencias(t)}</div>`;
  return `<section class="modulo-comunicaciones" data-comunicaciones-vista><header class="cabecera-vista"><p class="sobrelinea">${e(t("cabecera_sobrelinea"))}</p><h2>${e(t("titulo"))}</h2><p>${e(t("descripcion"))}</p></header>${bandaEstado(t)}<nav class="comunicaciones-pestanas" aria-label="${e(t("navegacion"))}" role="tablist">${PESTANAS.map(([clave, etiqueta]) => `<button type="button" id="comunicaciones-tab-${clave}" role="tab" aria-controls="comunicaciones-panel" tabindex="${clave === pestana ? "0" : "-1"}" data-comunicaciones-pestana="${clave}" aria-selected="${clave === pestana}">${e(t(etiqueta))}</button>`).join("")}</nav><div id="comunicaciones-panel" role="tabpanel" aria-labelledby="comunicaciones-tab-${pestana}" tabindex="0">${contenido}</div></section>`;
}

export function montarVistaComunicaciones({ raiz, anunciar = () => {}, registrarDesmontar } = {}) {
  if (!raiz?.replaceChildren || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) throw new TypeError("vista de Comunicaciones no disponible");
  let activa = true;
  let pestana = "bandeja";
  const pintar = () => { if (activa) raiz.innerHTML = renderizarVistaComunicaciones({ pestana }); };
  const t = crearTraductorComunicaciones();
  const cambiarPestana = (clave) => {
    if (!PESTANAS.some(([nombre]) => nombre === clave)) return;
    pestana = clave;
    pintar();
    raiz.querySelector(`[data-comunicaciones-pestana="${clave}"]`)?.focus();
    anunciar(t("seccion_seleccionada", { seccion: t(`pestana_${clave}`) }), "info");
  };
  const pulsar = (evento) => {
    const tab = evento.target.closest?.("[data-comunicaciones-pestana]");
    if (tab) cambiarPestana(tab.dataset.comunicacionesPestana);
  };
  const teclado = (evento) => {
    const tab = evento.target.closest?.("[data-comunicaciones-pestana]");
    if (!tab) return;
    const indice = PESTANAS.findIndex(([clave]) => clave === tab.dataset.comunicacionesPestana);
    if (indice < 0) return;
    const siguiente = evento.key === "ArrowRight" ? (indice + 1) % PESTANAS.length
      : evento.key === "ArrowLeft" ? (indice + PESTANAS.length - 1) % PESTANAS.length
        : evento.key === "Home" ? 0 : evento.key === "End" ? PESTANAS.length - 1 : -1;
    if (siguiente < 0) return;
    evento.preventDefault();
    cambiarPestana(PESTANAS[siguiente][0]);
  };
  raiz.addEventListener("click", pulsar);
  raiz.addEventListener("keydown", teclado);
  pintar();
  const desmontar = () => {
    if (!activa) return;
    activa = false;
    raiz.removeEventListener("click", pulsar);
    raiz.removeEventListener("keydown", teclado);
    raiz.replaceChildren();
  };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar });
}
