import { crearTraductorInscripciones } from "./i18n.js";

const PESTANAS = Object.freeze(["convocatorias", "mis_inscripciones", "subsanaciones", "preparar"]);
const ESTADOS = new Set(["cargando", "disponible", "vacio", "no_configurado", "denegado", "error"]);
const REQUISITOS = new Set(["cumple", "no_cumple", "pendiente"]);
const escapar = (valor) => String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
const lista = (valor) => Array.isArray(valor) ? valor : [];
const texto = (valor, defecto) => typeof valor === "string" && valor.trim() ? valor.trim() : defecto;

function ayuda(t) {
  return `<details class="inscripciones-ayuda"><summary aria-label="${escapar(t("ayuda_etiqueta"))}" title="${escapar(t("ayuda_etiqueta"))}">?</summary><p>${escapar(t("ayuda"))}</p></details>`;
}

function panel(titulo, cuerpo, subtitulo = "", extra = "") {
  return `<section class="panel inscripciones-panel"><div class="cabecera-panel"><div><h3>${escapar(titulo)}</h3>${subtitulo ? `<p>${escapar(subtitulo)}</p>` : ""}</div>${extra}</div><div class="cuerpo-panel">${cuerpo}</div></section>`;
}

function estadoRequisito(requisito, t) {
  const estado = REQUISITOS.has(requisito?.estado) ? requisito.estado : "pendiente";
  return `<li class="inscripciones-requisito inscripciones-requisito--${estado}"><div><strong>${escapar(texto(requisito?.descripcion, t("sin_evaluacion")))}</strong>${requisito?.motivo ? `<p>${escapar(requisito.motivo)}</p>` : ""}${requisito?.procedencia ? `<small>${escapar(t("procedencia"))}: ${escapar(requisito.procedencia)}</small>` : ""}</div><span class="inscripciones-estado inscripciones-estado--${estado}">${escapar(t(estado))}</span></li>`;
}

function tarjetaConvocatoria(item, indice, t, puedeAbrir) {
  const requisitos = lista(item?.requisitos);
  const documentos = lista(item?.documentos);
  const id = texto(item?.identificador_publico, "");
  return `<article class="inscripciones-ficha"><div class="inscripciones-ficha-cabecera"><div><h4>${escapar(texto(item?.titulo, t("convocatorias")))}</h4><p>${escapar(t("categoria"))}: ${escapar(texto(item?.categoria, "—"))}</p></div><span class="inscripciones-estado">${escapar(texto(item?.estado, t("consulta")))}</span></div><dl class="inscripciones-metadatos"><div><dt>${escapar(t("plazo"))}</dt><dd>${escapar(texto(item?.plazo, t("sin_plazo")))}</dd></div><div><dt>${escapar(t("bases"))}</dt><dd>${escapar(texto(item?.bases, t("sin_bases")))}</dd></div></dl><details><summary>${escapar(t("requisitos"))} (${requisitos.length})</summary>${requisitos.length ? `<ul class="inscripciones-requisitos">${requisitos.map((r) => estadoRequisito(r, t)).join("")}</ul>` : `<p>${escapar(t("sin_evaluacion"))}</p>`}<p class="inscripciones-nota">${escapar(t("requisito_pendiente"))}</p></details><details><summary>${escapar(t("documentos"))} (${documentos.length})</summary>${documentos.length ? `<ul>${documentos.map((d) => `<li>${escapar(d)}</li>`).join("")}</ul>` : `<p>${escapar(t("sin_documentos"))}</p>`}</details><div class="inscripciones-acciones"><button type="button" class="inscripciones-boton-secundario" data-inscripciones-detalle="${indice}" ${!puedeAbrir || !id ? `disabled aria-disabled="true" title="${escapar(t("detalle_pendiente"))}"` : ""}>${escapar(t("detalle_publico"))}</button>${!puedeAbrir || !id ? `<small>${escapar(t("detalle_pendiente"))}</small>` : ""}</div></article>`;
}

function contenidoConvocatorias(datos, t, puedeAbrir) {
  const items = lista(datos.convocatorias);
  return panel(t("convocatorias"), items.length ? `<div class="inscripciones-lista">${items.map((item, i) => tarjetaConvocatoria(item, i, t, puedeAbrir)).join("")}</div>` : `<p class="inscripciones-vacio">${escapar(t("sin_convocatorias"))}</p>`);
}

function contenidoMisInscripciones(datos, t) {
  const items = lista(datos.inscripciones);
  return panel(t("mis_inscripciones"), items.length ? `<div class="inscripciones-tabla-wrap" role="region" tabindex="0" aria-label="${escapar(t("mis_inscripciones"))}"><table><thead><tr><th scope="col">${escapar(t("convocatorias"))}</th><th scope="col">${escapar(t("referencia"))}</th><th scope="col">${escapar(t("estado"))}</th></tr></thead><tbody>${items.map((item) => `<tr><th scope="row">${escapar(texto(item?.titulo, t("convocatorias")))}</th><td>${escapar(texto(item?.referencia, t("sin_referencia")))}</td><td><span class="inscripciones-estado">${escapar(texto(item?.estado, t("consulta")))}</span></td></tr>`).join("")}</tbody></table></div>` : `<p class="inscripciones-vacio">${escapar(t("sin_inscripciones"))}</p>`);
}

function contenidoSubsanaciones(datos, t) {
  const items = lista(datos.subsanaciones);
  return panel(t("subsanaciones"), items.length ? `<div class="inscripciones-lista">${items.map((item) => `<article class="inscripciones-ficha"><div class="inscripciones-ficha-cabecera"><h4>${escapar(texto(item?.titulo, t("subsanaciones")))}</h4><span class="inscripciones-estado inscripciones-estado--pendiente">${escapar(texto(item?.estado, t("pendiente")))}</span></div><dl class="inscripciones-metadatos"><div><dt>${escapar(t("motivo"))}</dt><dd>${escapar(texto(item?.motivo, "—"))}</dd></div><div><dt>${escapar(t("fecha_limite"))}</dt><dd>${escapar(texto(item?.fecha_limite, t("sin_fecha_limite")))}</dd></div></dl><button type="button" disabled aria-disabled="true" title="${escapar(t("aportar_pendiente"))}">${escapar(t("aportar"))}</button><p class="inscripciones-nota">${escapar(t("aportar_pendiente"))}</p></article>`).join("")}</div>` : `<p class="inscripciones-vacio">${escapar(t("sin_subsanaciones"))}</p>`);
}

function contenidoPreparar(datos, ui, t) {
  const item = datos.convocatoriaSeleccionada;
  const persona = datos.persona;
  if (!item || !persona) return panel(t("preparar"), `<p class="inscripciones-vacio">${escapar(item ? t("sin_persona") : t("detalle_pendiente"))}</p>`, t("sin_efectos"));
  const clase = persona.condicion === "empleada" ? "empleada" : persona.condicion === "externa" ? "externa" : "persona";
  const detalles = `<div class="inscripciones-preparar-resumen"><div><span>${escapar(t("convocatorias"))}</span><strong>${escapar(texto(item.titulo, t("convocatorias")))}</strong></div><div><span>${escapar(t("persona"))}</span><strong>${escapar(t(clase))}</strong></div></div><label class="inscripciones-confirmacion"><input type="checkbox" data-inscripciones-confirmar ${ui.confirmada ? "checked" : ""}><span>${escapar(t("confirmar"))}</span></label><p class="inscripciones-nota">${escapar(t("confirmar_ayuda"))}</p><button type="button" data-inscripciones-preparar ${ui.confirmada ? "" : "disabled aria-disabled=\"true\""}>${escapar(t("preparar_borrador"))}</button>`;
  const borrador = ui.confirmada && ui.preparado ? `<section class="inscripciones-borrador" aria-label="${escapar(t("borrador"))}"><h4>${escapar(t("borrador"))}</h4><dl class="inscripciones-metadatos"><div><dt>${escapar(t("nombre"))}</dt><dd>${escapar(texto(persona.nombre_visible, t("sin_persona")))}</dd></div><div><dt>${escapar(t("contacto"))}</dt><dd>${escapar(texto(persona.contacto_visible, t("sin_contacto")))}</dd></div></dl><p class="inscripciones-nota">${escapar(t("aviso_borrador"))}</p><button type="button" disabled aria-disabled="true" title="${escapar(t("registrar_pendiente"))}">${escapar(t("registrar"))}</button></section>` : "";
  return panel(t("preparar"), detalles + borrador, t("sin_efectos"));
}

export function renderizarVistaInscripciones(datos = {}, ui = {}, opciones = {}) {
  const t = crearTraductorInscripciones(opciones.catalogo);
  const estado = ESTADOS.has(datos?.estado) ? datos.estado : "no_configurado";
  const pestana = PESTANAS.includes(ui.pestana) ? ui.pestana : "convocatorias";
  const aviso = { cargando: t("cargando"), vacio: t("vacio"), no_configurado: t("sin_conexion"), denegado: t("denegado"), error: t("error") }[estado];
  const contenido = estado !== "disponible" ? panel(t("consulta"), `<p role="status">${escapar(aviso)}</p>`) : {
    convocatorias: () => contenidoConvocatorias(datos, t, opciones.puedeAbrir === true),
    mis_inscripciones: () => contenidoMisInscripciones(datos, t),
    subsanaciones: () => contenidoSubsanaciones(datos, t),
    preparar: () => contenidoPreparar(datos, ui, t),
  }[pestana]();
  return `<section class="inscripciones-modulo" data-inscripciones-modulo data-estado-entrega="visual_pendiente_backend"><header class="inscripciones-cabecera"><div><h2>${escapar(t("titulo"))}</h2><p>${escapar(t("descripcion"))}</p></div>${ayuda(t)}</header><p class="inscripciones-limite" role="status">${escapar(t("sin_efectos"))}. ${escapar(t("registrar_pendiente"))} ${escapar(t("fuente"))}: ${escapar(texto(datos?.fuente, t("fuente_pendiente")))}</p><nav class="inscripciones-pestanas" aria-label="${escapar(t("titulo"))}">${PESTANAS.map((clave) => `<button type="button" data-inscripciones-tab="${clave}" aria-current="${clave === pestana ? "page" : "false"}">${escapar(t(clave))}</button>`).join("")}</nav><div class="inscripciones-contenido">${contenido}</div></section>`;
}

/** Vista local sin transporte. El integrador inyecta datos consultados y apertura del detalle público. */
export function montarVistaInscripciones({ raiz, datos = {}, anunciar = () => {}, registrarDesmontar, abrirDetallePublico } = {}) {
  if (!raiz?.append || !raiz?.ownerDocument?.createElement || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function") || (abrirDetallePublico !== undefined && typeof abrirDetallePublico !== "function")) throw new TypeError("vista de inscripciones no disponible");
  const contenedor = raiz.ownerDocument.createElement("div");
  raiz.append(contenedor);
  let activa = true;
  let modelo = datos;
  let ui = { pestana: "convocatorias", confirmada: false, preparado: false };
  const pintar = () => { if (activa) contenedor.innerHTML = renderizarVistaInscripciones(modelo, ui, { puedeAbrir: !!abrirDetallePublico }); };
  const alClick = (evento) => {
    const tab = evento.target?.closest?.("[data-inscripciones-tab]");
    if (tab) { ui = { ...ui, pestana: tab.dataset.inscripcionesTab }; pintar(); return; }
    const detalle = evento.target?.closest?.("[data-inscripciones-detalle]");
    if (detalle && abrirDetallePublico && modelo.estado === "disponible") {
      const item = lista(modelo.convocatorias)[Number(detalle.dataset.inscripcionesDetalle)];
      const id = texto(item?.identificador_publico, "");
      if (id) abrirDetallePublico(id);
      return;
    }
    const preparar = evento.target?.closest?.("[data-inscripciones-preparar]");
    if (preparar && !preparar.disabled && ui.confirmada && modelo.estado === "disponible" && modelo.convocatoriaSeleccionada && modelo.persona) {
      ui = { ...ui, preparado: true };
      pintar();
      anunciar(crearTraductorInscripciones()("aviso_preparado"), "informacion");
    }
  };
  const alCambio = (evento) => {
    const confirmar = evento.target?.closest?.("[data-inscripciones-confirmar]");
    if (confirmar) { ui = { ...ui, confirmada: confirmar.checked, preparado: false }; pintar(); }
  };
  const desmontar = () => { if (!activa) return; activa = false; contenedor.removeEventListener("click", alClick); contenedor.removeEventListener("change", alCambio); contenedor.remove(); };
  const actualizar = (nuevosDatos = {}) => { if (!activa) return; modelo = nuevosDatos; ui = { pestana: "convocatorias", confirmada: false, preparado: false }; pintar(); };
  contenedor.addEventListener("click", alClick);
  contenedor.addEventListener("change", alCambio);
  pintar();
  registrarDesmontar?.(desmontar);
  return Object.freeze({ actualizar, desmontar });
}
