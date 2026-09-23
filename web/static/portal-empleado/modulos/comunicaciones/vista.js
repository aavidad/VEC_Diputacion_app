import { crearTraductorComunicaciones } from "./i18n.js?v=20260924-f2-web2";

const PESTANAS = Object.freeze([
  ["bandeja", "pestana_bandeja"],
  ["preferencias", "pestana_preferencias"],
  ["plantillas", "pestana_plantillas"],
  ["administrativas", "pestana_administrativas"],
]);
const ETAPAS = Object.freeze([
  ["transporte", "aceptado"],
  ["entrega", "entregado"],
  ["lectura", "leido"],
]);
const ESTADOS_VISTA = new Set(["no_configurado", "cargando", "disponible", "vacio", "denegado", "error"]);
const MAX_COMUNICACIONES = 100;

function e(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}
function texto(valor, nombre, maximo = 200) {
  if (typeof valor !== "string" || !valor.trim() || valor.length > maximo) throw new TypeError(`comunicación inválida: ${nombre}`);
  return valor.trim();
}
function normalizarComunicacion(entrada) {
  if (!entrada || typeof entrada !== "object" || Array.isArray(entrada)) throw new TypeError("comunicación inválida");
  const evidencias = entrada.evidencias;
  if (evidencias !== undefined && (!evidencias || typeof evidencias !== "object" || Array.isArray(evidencias))) throw new TypeError("evidencias inválidas");
  const estados = {};
  for (const [etapa, permitido] of ETAPAS) {
    const valor = evidencias?.[etapa] ?? null;
    if (valor !== null && (!valor || typeof valor !== "object" || Array.isArray(valor) || valor.estado !== permitido)) throw new TypeError(`evidencia inválida: ${etapa}`);
    estados[etapa] = valor === null ? null : Object.freeze({ estado: permitido, recibo_ref: texto(valor.recibo_ref, `recibo ${etapa}`, 120) });
  }
  if ((estados.entrega && !estados.transporte) || (estados.lectura && !estados.entrega)) throw new TypeError("evidencias sin antecedente");
  return Object.freeze({
    referencia: texto(entrada.referencia, "referencia", 120),
    asunto: texto(entrada.asunto, "asunto"),
    canal_previsto: texto(entrada.canal_previsto, "canal previsto", 80),
    evidencias: Object.freeze(estados),
  });
}

/** La fuente debe devolver { estado: 'disponible', autorizado: true, comunicaciones: [...] }.
 * Denegado y no_configurado se aceptan sin datos. Nunca se interpreta una lista sola como autorización. */
export function normalizarConsultaComunicaciones(respuesta) {
  if (!respuesta || typeof respuesta !== "object" || Array.isArray(respuesta)) throw new TypeError("consulta de comunicaciones inválida");
  if (respuesta.estado === "denegado" || respuesta.autorizado === false) return Object.freeze({ estado: "denegado", comunicaciones: Object.freeze([]) });
  if (respuesta.estado === "no_configurado") return Object.freeze({ estado: "no_configurado", comunicaciones: Object.freeze([]) });
  if (respuesta.estado !== "disponible" || respuesta.autorizado !== true || !Array.isArray(respuesta.comunicaciones) || respuesta.comunicaciones.length > MAX_COMUNICACIONES) throw new TypeError("consulta de comunicaciones inválida");
  const referencias = new Set();
  const comunicaciones = respuesta.comunicaciones.map((entrada) => {
    const comunicacion = normalizarComunicacion(entrada);
    if (referencias.has(comunicacion.referencia)) throw new TypeError("referencia de comunicación duplicada");
    referencias.add(comunicacion.referencia);
    return comunicacion;
  });
  return Object.freeze({ estado: comunicaciones.length ? "disponible" : "vacio", comunicaciones: Object.freeze(comunicaciones) });
}

function chip(texto, tono = "aviso") { return `<span class="estado-chip ${tono}">${e(texto)}</span>`; }
function botonBloqueado(texto, motivo) {
  return `<button type="button" class="boton-secundario" disabled aria-disabled="true" title="${e(motivo)}">${e(texto)}</button>`;
}
function estadoEtapa(comunicacion, etapa, t) {
  const evidencia = comunicacion?.evidencias?.[etapa];
  return evidencia ? chip(t(`estado_${evidencia.estado}`), etapa === "transporte" ? "info" : "exito") : chip(t("sin_evidencia"));
}
function bandaEstado(estado, t) {
  return `<details class="panel comunicaciones-estado"><summary>${e(t(`estado_${estado}`))} <span aria-hidden="true">?</span></summary><div class="cuerpo-panel"><p>${e(t(`estado_ayuda_${estado}`))}</p><p>${e(t("limite_general"))}</p></div></details>`;
}
function bandeja(comunicaciones, seleccion, estado, filtro, t) {
  const disponibles = estado === "disponible" || estado === "vacio";
  const buscadas = comunicaciones.filter((item) => !filtro || `${item.referencia} ${item.asunto} ${item.canal_previsto}`.toLocaleLowerCase("es").includes(filtro.toLocaleLowerCase("es")));
  const mensajeVacio = disponibles && filtro ? t("sin_coincidencias") : t(`bandeja_${estado}`);
  return `<section class="panel comunicaciones-bandeja" aria-labelledby="comunicaciones-bandeja-titulo"><div class="cabecera-panel"><div><p class="sobrelinea">${e(t("bandeja_sobrelinea"))}</p><h3 id="comunicaciones-bandeja-titulo">${e(t("bandeja_titulo"))}</h3></div>${chip(t(buscadas.length === 1 ? "visible_uno" : "visibles", { numero: buscadas.length }), disponibles ? "info" : "aviso")}</div>
    ${disponibles && comunicaciones.length ? `<form class="comunicaciones-filtros" data-comunicaciones-filtro><label>${e(t("filtrar"))}<input name="filtro" value="${e(filtro)}" maxlength="120" autocomplete="off" placeholder="${e(t("filtro_placeholder"))}"></label><button type="submit" class="boton-secundario">${e(t("aplicar_filtro"))}</button></form>` : ""}
    <div class="tabla-contenedor" tabindex="0" role="region" aria-label="${e(t("region_bandeja"))}"><table class="tabla-datos comunicaciones-tabla"><caption>${e(t("caption_bandeja"))}</caption><thead><tr>${["asunto", "canal_previsto", ...ETAPAS.map(([etapa]) => `evidencia_${etapa}`)].map((clave) => `<th scope="col">${e(t(clave))}</th>`).join("")}</tr></thead><tbody>${buscadas.length ? buscadas.map((item) => `<tr data-seleccionada="${item.referencia === seleccion}"><th scope="row"><button type="button" class="enlace-tabla" data-comunicaciones-seleccionar="${e(item.referencia)}" ${item.referencia === seleccion ? 'aria-current="true"' : ""}>${e(item.asunto)}</button><small>${e(item.referencia)}</small></th><td>${e(item.canal_previsto)}</td>${ETAPAS.map(([etapa]) => `<td>${estadoEtapa(item, etapa, t)}</td>`).join("")}</tr>`).join("") : `<tr><td colspan="5" class="comunicaciones-vacio">${e(mensajeVacio)}</td></tr>`}</tbody></table></div></section>`;
}
function detalle(comunicacion, estado, t) {
  return `<section class="panel comunicaciones-detalle" aria-labelledby="comunicaciones-detalle-titulo"><div class="cabecera-panel"><div><p class="sobrelinea">${e(t("detalle_sobrelinea"))}</p><h3 id="comunicaciones-detalle-titulo" tabindex="-1">${e(comunicacion?.asunto || t("trazabilidad_titulo"))}</h3></div>${chip(t(`estado_${estado}`), estado === "disponible" ? "info" : "aviso")}</div><div class="cuerpo-panel">${comunicacion ? `<p class="comunicaciones-referencia">${e(t("referencia"))}: ${e(comunicacion.referencia)}</p>` : ""}<p class="comunicaciones-explicacion">${e(t("trazabilidad_ayuda"))}</p><dl class="comunicaciones-evidencias">${ETAPAS.map(([etapa]) => `<div><dt>${e(t(`evidencia_${etapa}`))}</dt><dd>${estadoEtapa(comunicacion, etapa, t)}${comunicacion?.evidencias?.[etapa] ? `<small>${e(t("recibo"))}: ${e(comunicacion.evidencias[etapa].recibo_ref)}</small>` : ""}</dd></div>`).join("")}</dl><p class="comunicaciones-limite">${e(t("limite_detalle"))}</p><div class="acciones-vista">${botonBloqueado(t("enviar"), t("enviar_motivo"))}</div><p class="comunicaciones-motivo">${e(t("enviar_motivo"))}</p></div></section>`;
}
function seccionVacia(t, nombre, titulo, accion, motivo, ayuda) {
  return `<section class="panel" aria-labelledby="comunicaciones-${nombre}-titulo"><div class="cabecera-panel"><div><p class="sobrelinea">${e(t("estado_no_configurado"))}</p><h3 id="comunicaciones-${nombre}-titulo">${e(t(titulo))}</h3></div>${botonBloqueado(t(accion), t(motivo))}</div><div class="cuerpo-panel"><p class="comunicaciones-vacio">${e(t(ayuda))}</p></div></section>`;
}
function administrativas(t) {
  return `<section class="panel comunicaciones-segregadas" aria-labelledby="comunicaciones-administrativas-titulo"><div class="cabecera-panel"><div><p class="sobrelinea">${e(t("estado_no_configurado"))}</p><h3 id="comunicaciones-administrativas-titulo">${e(t("administrativas_titulo"))}</h3></div>${botonBloqueado(t("crear_campana"), t("crear_campana_motivo"))}</div><div class="cuerpo-panel"><p>${e(t("administrativas_ayuda"))}</p><div class="acciones-vista">${botonBloqueado(t("enviar"), t("enviar_motivo"))}</div><p class="comunicaciones-motivo">${e(t("enviar_motivo"))}</p></div></section>`;
}

/** Render puro. El montaje solo entrega comunicaciones tras validar una respuesta autorizada. */
export function renderizarVistaComunicaciones({ pestana = "bandeja", estado = "no_configurado", comunicaciones = [], filtro = "", seleccion = "" } = {}) {
  const t = crearTraductorComunicaciones();
  if (!PESTANAS.some(([clave]) => clave === pestana)) pestana = "bandeja";
  if (!ESTADOS_VISTA.has(estado)) estado = "error";
  if (!Array.isArray(comunicaciones)) comunicaciones = [];
  if (estado !== "disponible" && estado !== "vacio") comunicaciones = [];
  const textoFiltro = String(filtro).trim().slice(0, 120);
  const visibles = comunicaciones.filter((item) => !textoFiltro || `${item.referencia} ${item.asunto} ${item.canal_previsto}`.toLocaleLowerCase("es").includes(textoFiltro.toLocaleLowerCase("es")));
  const seleccionada = visibles.find((item) => item.referencia === seleccion) || visibles[0] || null;
  const contenido = pestana === "preferencias"
    ? seccionVacia(t, "preferencias", "preferencias_titulo", "cambiar_preferencias", "cambiar_preferencias_motivo", "preferencias_ayuda")
    : pestana === "plantillas"
      ? seccionVacia(t, "plantillas", "plantillas_titulo", "redactar", "redactar_motivo", "plantillas_ayuda")
      : pestana === "administrativas" ? administrativas(t)
        : `<div class="comunicaciones-espacio">${bandeja(comunicaciones, seleccionada?.referencia, estado, textoFiltro, t)}${detalle(seleccionada, estado, t)}</div>`;
  return `<section class="modulo-comunicaciones" data-comunicaciones-vista><header class="cabecera-vista"><p class="sobrelinea">${e(t("cabecera_sobrelinea"))}</p><h2>${e(t("titulo"))}</h2><p>${e(t("descripcion"))}</p></header>${bandaEstado(estado, t)}<nav class="comunicaciones-pestanas" aria-label="${e(t("navegacion"))}" role="tablist">${PESTANAS.map(([clave, etiqueta]) => `<button type="button" id="comunicaciones-tab-${clave}" role="tab" aria-controls="comunicaciones-panel" tabindex="${clave === pestana ? "0" : "-1"}" data-comunicaciones-pestana="${clave}" aria-selected="${clave === pestana}">${e(t(etiqueta))}</button>`).join("")}</nav><div id="comunicaciones-panel" role="tabpanel" aria-labelledby="comunicaciones-tab-${pestana}" tabindex="0">${contenido}</div></section>`;
}

/** fuente.listarAutorizadas({signal}) es el único puerto de lectura; no hay operación de envío. */
export function montarVistaComunicaciones({ raiz, anunciar = () => {}, registrarDesmontar, fuente } = {}) {
  if (!raiz?.replaceChildren || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function") || (fuente !== undefined && typeof fuente?.listarAutorizadas !== "function")) throw new TypeError("vista de Comunicaciones no disponible");
  let activa = true;
  let controlador;
  let secuencia = 0;
  let vista = { pestana: "bandeja", estado: fuente ? "cargando" : "no_configurado", comunicaciones: [], filtro: "", seleccion: "" };
  const t = crearTraductorComunicaciones();
  const pintar = () => { if (activa) raiz.innerHTML = renderizarVistaComunicaciones(vista); };
  const consultar = async () => {
    if (!fuente || !activa) return;
    controlador?.abort();
    controlador = new AbortController();
    const intento = ++secuencia;
    vista = { ...vista, estado: "cargando", comunicaciones: [], seleccion: "", filtro: "" };
    pintar();
    try {
      const respuesta = await fuente.listarAutorizadas({ signal: controlador.signal });
      if (!activa || controlador.signal.aborted || intento !== secuencia) return;
      const consulta = normalizarConsultaComunicaciones(respuesta);
      vista = { ...vista, ...consulta };
      pintar();
    } catch {
      if (!activa || controlador.signal.aborted || intento !== secuencia) return;
      vista = { ...vista, estado: "error", comunicaciones: [] };
      pintar();
    }
  };
  const cambiarPestana = (clave) => {
    if (!PESTANAS.some(([nombre]) => nombre === clave)) return;
    vista = { ...vista, pestana: clave };
    pintar();
    raiz.querySelector(`[data-comunicaciones-pestana="${clave}"]`)?.focus();
    anunciar(t("seccion_seleccionada", { seccion: t(`pestana_${clave}`) }), "info");
  };
  const pulsar = (evento) => {
    const tab = evento.target.closest?.("[data-comunicaciones-pestana]");
    const seleccion = evento.target.closest?.("[data-comunicaciones-seleccionar]");
    if (tab) cambiarPestana(tab.dataset.comunicacionesPestana);
    if (seleccion) {
      vista = { ...vista, seleccion: seleccion.dataset.comunicacionesSeleccionar };
      pintar();
      raiz.querySelector("#comunicaciones-detalle-titulo")?.focus();
    }
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
  const filtrar = (evento) => {
    const formulario = evento.target.closest?.("[data-comunicaciones-filtro]");
    if (!formulario) return;
    evento.preventDefault();
    vista = { ...vista, filtro: String(new FormData(formulario).get("filtro") || "").trim().slice(0, 120), seleccion: "" };
    pintar();
    raiz.querySelector("[name=filtro]")?.focus();
    anunciar(t("filtro_aplicado"), "info");
  };
  raiz.addEventListener("click", pulsar);
  raiz.addEventListener("keydown", teclado);
  raiz.addEventListener("submit", filtrar);
  pintar();
  void consultar();
  const desmontar = () => {
    if (!activa) return;
    activa = false;
    ++secuencia;
    controlador?.abort();
    raiz.removeEventListener("click", pulsar);
    raiz.removeEventListener("keydown", teclado);
    raiz.removeEventListener("submit", filtrar);
    raiz.replaceChildren();
  };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar, refrescar: consultar });
}
