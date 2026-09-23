import { crearTraductorSolicitudes, MENSAJES_SOLICITUDES_ES } from "./i18n.js";

const PESTANAS = Object.freeze(["bandeja", "nueva", "seguimiento", "certificados"]);
const SITUACIONES = new Set(["cargando", "disponible", "vacio", "no_configurado", "denegado", "error"]);
const ESTADOS = Object.freeze(["registrada", "en_revision", "pendiente_subsanacion", "finalizada"]);
const escapar = (valor) => String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
const texto = (valor, t) => valor === undefined || valor === null || String(valor).trim() === "" ? t("sin_dato") : String(valor);
const lista = (valor) => Array.isArray(valor) ? valor : [];
const referencia = (item) => String(item?.referencia ?? item?.id ?? "");
const titulo = (item) => item?.titulo ?? item?.tipo;
const claveEstado = (item) => typeof item?.estado === "string" && ESTADOS.includes(item.estado) ? item.estado : "no_disponible";
const chip = (item, t) => `<span class="solicitudes-chip solicitudes-chip--${claveEstado(item)}">${escapar(t(`estado_${claveEstado(item)}`))}</span>`;
function fecha(valor, t) {
  if (typeof valor !== "string" || !/^\d{4}-\d\d-\d\d(?:T.*)?$/.test(valor)) return t("sin_dato");
  const instante = new Date(valor.length === 10 ? `${valor}T12:00:00Z` : valor);
  return Number.isNaN(instante.getTime()) ? t("sin_dato") : new Intl.DateTimeFormat("es-ES", { day: "2-digit", month: "2-digit", year: "numeric", timeZone: "Europe/Madrid" }).format(instante);
}
const ayuda = (clave, t) => `<details class="solicitudes-ayuda"><summary aria-label="${escapar(t(clave))}" title="${escapar(t(clave))}">?</summary><p>${escapar(t(clave))}</p></details>`;
const botonPendiente = (clave, motivo, t) => `<div class="solicitudes-accion-pendiente"><button type="button" disabled aria-disabled="true" title="${escapar(t(motivo))}">${escapar(t(clave))}</button><small>${escapar(t(motivo))}</small></div>`;

function estadoConsulta(situacion, t) {
  const detalle = situacion === "disponible" ? "ayuda_estado" : `detalle_${situacion}`;
  return `<div class="solicitudes-estado solicitudes-estado--${situacion}" role="status"><strong>${escapar(t(`estado_${situacion}`))}</strong><span>${escapar(t(detalle))}</span></div>`;
}
function resumen(datos, situacion, t) {
  const tramites = lista(datos.tramites);
  const disponible = situacion === "disponible" || situacion === "vacio";
  const valores = [
    ["resumen_total", tramites.length, "●"],
    ["resumen_activos", tramites.filter((item) => ["registrada", "en_revision"].includes(claveEstado(item))).length, "◔"],
    ["resumen_subsanaciones", tramites.filter((item) => claveEstado(item) === "pendiente_subsanacion").length, "!"],
    ["resumen_finalizados", tramites.filter((item) => claveEstado(item) === "finalizada").length, "✓"],
  ];
  return `<section class="solicitudes-resumen rejilla-kpi" aria-label="${escapar(t("resumen_total"))}">${valores.map(([clave, numero, icono]) => `<article class="tarjeta-kpi solicitudes-kpi"><span class="icono-kpi" aria-hidden="true">${icono}</span><span class="solicitudes-kpi-texto"><span>${escapar(t(clave))}</span><strong class="valor-kpi">${disponible ? numero : escapar(t("kpi_sin_fuente"))}</strong><small>${escapar(disponible ? t("kpi_fuente") : t(`estado_${situacion}`))}</small></span></article>`).join("")}</section>`;
}
function bandeja(datos, estado, t) {
  const busqueda = estado.busqueda.trim().toLocaleLowerCase("es");
  const filtrados = lista(datos.tramites).filter((item) => (estado.filtro === "todos" || claveEstado(item) === estado.filtro)
    && `${referencia(item)} ${titulo(item) ?? ""}`.toLocaleLowerCase("es").includes(busqueda));
  const cabeceras = ["referencia", "tramite", "fecha", "unidad", "estado", "accion"];
  return `<section class="solicitudes-panel panel" aria-labelledby="solicitudes-bandeja-titulo"><header class="cabecera-panel solicitudes-panel-cabecera"><div><h3 id="solicitudes-bandeja-titulo">${escapar(t("bandeja_titulo"))}</h3><p>${escapar(t("bandeja_subtitulo"))}</p></div>${ayuda("ayuda_estado", t)}</header><div class="cuerpo-panel solicitudes-panel-cuerpo"><form class="solicitudes-filtros" data-solicitudes-filtros><label>${escapar(t("buscar"))}<input name="busqueda" value="${escapar(estado.busqueda)}" maxlength="80" autocomplete="off"></label><label>${escapar(t("filtrar"))}<select name="estado"><option value="todos"${estado.filtro === "todos" ? " selected" : ""}>${escapar(t("todos"))}</option>${ESTADOS.map((clave) => `<option value="${clave}"${estado.filtro === clave ? " selected" : ""}>${escapar(t(`estado_${clave}`))}</option>`).join("")}</select></label></form><div class="solicitudes-tabla-wrap" role="region" tabindex="0" aria-label="${escapar(t("region_bandeja"))}"><table class="solicitudes-tabla"><caption>${escapar(t("tabla_tramites"))}</caption><thead><tr>${cabeceras.map((clave) => `<th scope="col">${escapar(t(clave))}</th>`).join("")}</tr></thead><tbody>${filtrados.length ? filtrados.map((item) => `<tr class="solicitudes-fila solicitudes-fila--${claveEstado(item)}"><th scope="row"><button type="button" class="solicitudes-enlace" data-solicitudes-detalle="${escapar(referencia(item))}">${escapar(referencia(item))}</button></th><td>${escapar(texto(titulo(item), t))}</td><td>${escapar(fecha(item.fecha, t))}</td><td>${escapar(texto(item.unidad, t))}</td><td>${chip(item, t)}</td><td><button type="button" data-solicitudes-detalle="${escapar(referencia(item))}">${escapar(t("ver"))}</button></td></tr>`).join("") : `<tr><td colspan="6" class="solicitudes-vacio">${escapar(t("sin_resultados"))}</td></tr>`}</tbody></table></div></div></section>`;
}
function nueva(datos, t) {
  const catalogo = lista(datos.catalogo);
  return `<section class="solicitudes-panel panel" aria-labelledby="solicitudes-nueva-titulo"><header class="cabecera-panel solicitudes-panel-cabecera"><div><h3 id="solicitudes-nueva-titulo">${escapar(t("nueva_titulo"))}</h3><p>${escapar(t("nueva_subtitulo"))}</p></div>${ayuda("nueva_ayuda", t)}</header><div class="cuerpo-panel solicitudes-tarjetas">${catalogo.length ? catalogo.map((item) => `<article class="solicitudes-tarjeta"><span class="solicitudes-chip">${escapar(texto(item.categoria, t))}</span><h4>${escapar(texto(titulo(item), t))}</h4><p>${escapar(texto(item.descripcion, t))}</p>${botonPendiente("iniciar", "iniciar_motivo", t)}</article>`).join("") : `<p class="solicitudes-vacio">${escapar(t("nueva_vacio"))}</p>`}</div></section>`;
}
function seguimiento(datos, estado, t) {
  const item = lista(datos.tramites).find((tramite) => referencia(tramite) === estado.seleccionada);
  const historial = lista(item?.historial);
  return `<section class="solicitudes-panel panel solicitudes-detalle" aria-labelledby="solicitudes-seguimiento-titulo"><header class="cabecera-panel solicitudes-panel-cabecera"><div><h3 id="solicitudes-seguimiento-titulo">${escapar(t("seguimiento_titulo"))}</h3><p>${escapar(t("seguimiento_ayuda"))}</p></div>${ayuda("seguimiento_ayuda", t)}</header><div class="cuerpo-panel">${item ? `<div class="solicitudes-ficha"><dl><div><dt>${escapar(t("referencia"))}</dt><dd>${escapar(referencia(item))}</dd></div><div><dt>${escapar(t("tramite"))}</dt><dd>${escapar(texto(titulo(item), t))}</dd></div><div><dt>${escapar(t("fecha"))}</dt><dd>${escapar(fecha(item.fecha, t))}</dd></div><div><dt>${escapar(t("estado"))}</dt><dd>${chip(item, t)}</dd></div><div><dt>${escapar(t("unidad"))}</dt><dd>${escapar(texto(item.unidad, t))}</dd></div><div><dt>${escapar(t("hito_actual"))}</dt><dd>${escapar(texto(item.paso, t))}</dd></div></dl><div class="solicitudes-ficha-descripcion"><h4>${escapar(t("descripcion"))}</h4><p>${escapar(texto(item.descripcion, t))}</p>${item.siguienteAccion ? `<p><strong>${escapar(t("siguiente_accion"))}:</strong> ${escapar(item.siguienteAccion)}</p>` : ""}</div></div><div class="solicitudes-historial"><h4>${escapar(t("historial"))}</h4>${historial.length ? `<ol>${historial.map((hito) => `<li><time>${escapar(fecha(hito.fecha, t))}</time><strong>${escapar(texto(hito.titulo, t))}</strong>${hito.detalle ? `<span>${escapar(hito.detalle)}</span>` : ""}</li>`).join("")}</ol>` : `<p>${escapar(t("historial_vacio"))}</p>`}</div>${botonPendiente("aportar", "aportar_motivo", t)}` : `<p class="solicitudes-vacio">${escapar(t("seguimiento_vacio"))}</p>`}</div></section>`;
}
function certificados(datos, t) {
  const items = lista(datos.certificados);
  const cabeceras = ["tipo", "alcance", "situacion", "accion"];
  return `<section class="solicitudes-panel panel" aria-labelledby="solicitudes-certificados-titulo"><header class="cabecera-panel solicitudes-panel-cabecera"><div><h3 id="solicitudes-certificados-titulo">${escapar(t("certificados_titulo"))}</h3><p>${escapar(t("certificados_subtitulo"))}</p></div>${ayuda("certificados_ayuda", t)}</header><div class="cuerpo-panel">${items.length ? `<div class="solicitudes-tabla-wrap" role="region" tabindex="0" aria-label="${escapar(t("tabla_certificados"))}"><table class="solicitudes-tabla"><caption>${escapar(t("tabla_certificados"))}</caption><thead><tr>${cabeceras.map((clave) => `<th scope="col">${escapar(t(clave))}</th>`).join("")}</tr></thead><tbody>${items.map((item) => `<tr><th scope="row">${escapar(texto(item.tipo, t))}</th><td>${escapar(texto(item.alcance, t))}</td><td>${escapar(texto(item.situacion, t))}</td><td>${botonPendiente("emitir", "emitir_motivo", t)}</td></tr>`).join("")}</tbody></table></div>` : `<p class="solicitudes-vacio">${escapar(t("certificados_vacio"))}</p>`}</div></section>`;
}

/** Solo presentación. Datos es la respuesta ya autorizada de la fuente inyectada. */
export function renderizarSolicitudes(estado = {}, mensajes = MENSAJES_SOLICITUDES_ES) {
  const t = crearTraductorSolicitudes(mensajes);
  const situacion = SITUACIONES.has(estado.situacion) ? estado.situacion : "no_configurado";
  const datos = situacion === "disponible" && estado.datos && typeof estado.datos === "object" ? estado.datos
    : situacion === "vacio" && estado.datos && typeof estado.datos === "object"
      ? { tramites: [], catalogo: lista(estado.datos.catalogo), certificados: lista(estado.datos.certificados) }
      : {};
  const pestana = PESTANAS.includes(estado.pestana) ? estado.pestana : "bandeja";
  const seguro = { busqueda: String(estado.busqueda ?? ""), filtro: ESTADOS.includes(estado.filtro) ? estado.filtro : "todos", seleccionada: situacion === "disponible" ? String(estado.seleccionada ?? "") : "" };
  const contenido = pestana === "bandeja" ? bandeja(datos, seguro, t)
    : pestana === "nueva" ? nueva(datos, t)
      : pestana === "seguimiento" ? seguimiento(datos, seguro, t)
        : certificados(datos, t);
  return `<section class="solicitudes-modulo" data-solicitudes-modulo><header class="solicitudes-cabecera"><p class="solicitudes-sobrelinea">${escapar(t("sobrelinea"))}</p><h2>${escapar(t("titulo"))}</h2><p>${escapar(t("descripcion_cabecera"))}</p></header>${estadoConsulta(situacion, t)}${resumen(datos, situacion, t)}<nav class="solicitudes-pestanas" role="tablist" aria-label="${escapar(t("navegacion"))}">${PESTANAS.map((clave) => `<button type="button" role="tab" data-solicitudes-tab="${clave}" aria-selected="${clave === pestana}" tabindex="${clave === pestana ? "0" : "-1"}" aria-controls="solicitudes-panel-actual">${escapar(t(clave))}</button>`).join("")}</nav><div id="solicitudes-panel-actual" role="tabpanel" tabindex="0" class="solicitudes-contenido">${contenido}</div></section>`;
}

/** Montaje de consulta. fuente.consultar({ signal }) devuelve datos ya autorizados. */
export function montarVistaSolicitudes({ raiz, fuente, anunciar = () => {}, registrarDesmontar, mensajes = MENSAJES_SOLICITUDES_ES } = {}) {
  if (!raiz?.replaceChildren || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function") || (fuente !== undefined && typeof fuente?.consultar !== "function")) throw new TypeError("vista de solicitudes no disponible");
  const t = crearTraductorSolicitudes(mensajes);
  const controlador = new AbortController();
  let activa = true;
  let estado = { pestana: "bandeja", busqueda: "", filtro: "todos", seleccionada: "", situacion: fuente ? "cargando" : "no_configurado", datos: {} };
  const pintar = () => { if (activa) raiz.innerHTML = renderizarSolicitudes(estado, mensajes); };
  const cambiarPestana = (pestana, foco = false) => { if (!PESTANAS.includes(pestana)) return; estado = { ...estado, pestana }; pintar(); if (foco) raiz.querySelector?.(`[data-solicitudes-tab="${pestana}"]`)?.focus?.(); anunciar(t("anuncio_seccion", { seccion: t(pestana) }), "informacion"); };
  const alClick = (evento) => {
    const tab = evento.target?.closest?.("[data-solicitudes-tab]");
    if (tab) { cambiarPestana(tab.dataset.solicitudesTab); return; }
    const detalle = evento.target?.closest?.("[data-solicitudes-detalle]");
    if (detalle && lista(estado.datos.tramites).some((item) => referencia(item) === detalle.dataset.solicitudesDetalle)) {
      estado = { ...estado, seleccionada: detalle.dataset.solicitudesDetalle, pestana: "seguimiento" };
      pintar(); raiz.querySelector?.("#solicitudes-panel-actual")?.focus?.();
      anunciar(t("anuncio_detalle", { referencia: estado.seleccionada }), "informacion");
    }
  };
  const aplicarFiltros = (formulario) => {
    if (!formulario?.elements) return;
    estado = { ...estado, busqueda: formulario.elements.busqueda.value, filtro: formulario.elements.estado.value };
    pintar();
  };
  const alCambio = (evento) => { if (evento.target?.closest?.("[data-solicitudes-filtros]")) aplicarFiltros(evento.target.closest("form")); };
  const alSubmit = (evento) => { if (!evento.target?.matches?.("[data-solicitudes-filtros]")) return; evento.preventDefault(); aplicarFiltros(evento.target); };
  const alTecla = (evento) => {
    const tab = evento.target?.closest?.("[data-solicitudes-tab]");
    if (!tab || !["ArrowLeft", "ArrowRight", "Home", "End"].includes(evento.key)) return;
    evento.preventDefault();
    const indice = PESTANAS.indexOf(tab.dataset.solicitudesTab);
    const siguiente = evento.key === "Home" ? 0 : evento.key === "End" ? PESTANAS.length - 1 : (indice + (evento.key === "ArrowRight" ? 1 : -1) + PESTANAS.length) % PESTANAS.length;
    cambiarPestana(PESTANAS[siguiente], true);
  };
  pintar();
  raiz.addEventListener("click", alClick);
  raiz.addEventListener("change", alCambio);
  raiz.addEventListener("submit", alSubmit);
  raiz.addEventListener("keydown", alTecla);
  if (fuente) Promise.resolve().then(() => fuente.consultar({ signal: controlador.signal })).then((respuesta) => {
    if (!activa) return;
    if (respuesta?.estado === "denegado" || respuesta?.estado === "no_configurado") { estado = { ...estado, situacion: respuesta.estado, datos: {} }; pintar(); return; }
    if (!respuesta || !Array.isArray(respuesta.tramites) || respuesta.tramites.some((item) => !item || typeof item !== "object" || !referencia(item).trim())) throw new TypeError("respuesta de solicitudes inválida");
    const datos = { tramites: respuesta.tramites, catalogo: lista(respuesta.catalogo), certificados: lista(respuesta.certificados) };
    estado = { ...estado, datos, situacion: datos.tramites.length ? "disponible" : "vacio" };
    pintar();
  }).catch(() => { if (!activa || controlador.signal.aborted) return; estado = { ...estado, situacion: "error", datos: {} }; pintar(); });
  const desmontar = () => { if (!activa) return; activa = false; controlador.abort(); raiz.removeEventListener("click", alClick); raiz.removeEventListener("change", alCambio); raiz.removeEventListener("submit", alSubmit); raiz.removeEventListener("keydown", alTecla); raiz.replaceChildren(); };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar });
}
