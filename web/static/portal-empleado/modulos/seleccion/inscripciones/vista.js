import { crearTraductorInscripciones } from "./i18n.js?v=20260924-f2-web2";

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
  return `<section class="panel inscripciones-panel"><div class="cabecera-panel"><div><h3 tabindex="-1" data-inscripciones-titulo>${escapar(titulo)}</h3>${subtitulo ? `<p>${escapar(subtitulo)}</p>` : ""}</div>${extra}</div><div class="cuerpo-panel">${cuerpo}</div></section>`;
}

function pasos(claves, activo, t) {
  return `<ol class="inscripciones-pasos" aria-label="${escapar(t("preparar"))}">${claves.map((clave, indice) => `<li class="${indice + 1 === activo ? "actual" : indice + 1 < activo ? "hecho" : ""}" ${indice + 1 === activo ? 'aria-current="step"' : ""}><span>${indice + 1}</span>${escapar(t(clave))}</li>`).join("")}</ol>`;
}

function errorValidacion(mensaje) {
  return mensaje ? `<p class="inscripciones-error" role="alert" tabindex="-1" data-inscripciones-error>${escapar(mensaje)}</p>` : "";
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

function contenidoSubsanaciones(datos, ui, t) {
  const items = lista(datos.subsanaciones);
  const seleccion = Number.isInteger(ui.subsanacionIndice) ? items[ui.subsanacionIndice] : null;
  if (!seleccion) {
    return panel(t("subsanaciones"), items.length ? `<div class="inscripciones-lista">${items.map((item, indice) => `<article class="inscripciones-ficha"><div class="inscripciones-ficha-cabecera"><h4>${escapar(texto(item?.titulo, t("subsanaciones")))}</h4><span class="inscripciones-estado inscripciones-estado--pendiente">${escapar(texto(item?.estado, t("pendiente")))}</span></div><dl class="inscripciones-metadatos"><div><dt>${escapar(t("motivo"))}</dt><dd>${escapar(texto(item?.motivo, "—"))}</dd></div><div><dt>${escapar(t("fecha_limite"))}</dt><dd>${escapar(texto(item?.fecha_limite, t("sin_fecha_limite")))}</dd></div></dl><button type="button" data-inscripciones-seleccionar-subsanacion="${indice}">${escapar(t("seleccionar_subsanacion"))}</button><p class="inscripciones-nota">${escapar(t("no_enviado"))}</p></article>`).join("")}</div>` : `<p class="inscripciones-vacio">${escapar(t("sin_subsanaciones"))}</p>`);
  }
  const paso = ui.pasoSubsanacion === 3 ? 3 : ui.pasoSubsanacion === 2 ? 2 : 1;
  const resumen = `<dl class="inscripciones-metadatos"><div><dt>${escapar(t("convocatorias"))}</dt><dd>${escapar(texto(seleccion.titulo, t("subsanaciones")))}</dd></div><div><dt>${escapar(t("fecha_limite"))}</dt><dd>${escapar(texto(seleccion.fecha_limite, t("sin_fecha_limite")))}</dd></div><div><dt>${escapar(t("motivo"))}</dt><dd>${escapar(texto(seleccion.motivo, "—"))}</dd></div></dl>`;
  let cuerpo = pasos(["paso_lectura", "paso_respuesta", "paso_revisar"], paso, t) + resumen;
  if (paso === 1) cuerpo += `<label class="inscripciones-confirmacion"><input type="checkbox" data-inscripciones-confirmar-requerimiento ${ui.requerimientoConfirmado ? "checked" : ""}><span>${escapar(t("confirmar_requerimiento"))}</span></label><div class="inscripciones-acciones"><button type="button" data-inscripciones-subsanacion-siguiente ${ui.requerimientoConfirmado ? "" : 'disabled aria-disabled="true"'}>${escapar(t("paso_respuesta"))}</button></div>`;
  if (paso === 2) cuerpo += `<div class="inscripciones-campo"><label for="inscripciones-respuesta">${escapar(t("respuesta"))}</label><textarea id="inscripciones-respuesta" data-inscripciones-respuesta rows="4" maxlength="1000" aria-describedby="inscripciones-respuesta-ayuda" ${ui.errorSubsanacion ? 'aria-invalid="true"' : ""}>${escapar(ui.respuesta ?? "")}</textarea><small id="inscripciones-respuesta-ayuda">${escapar(t("respuesta_ayuda"))}</small></div>${errorValidacion(ui.errorSubsanacion ? t("respuesta_error") : "")}<p class="inscripciones-nota">${escapar(t("respuesta_sin_documentos"))}</p><div class="inscripciones-acciones"><button type="button" data-inscripciones-subsanacion-siguiente>${escapar(t("siguiente"))}</button><button type="button" class="inscripciones-boton-secundario" data-inscripciones-subsanacion-atras>${escapar(t("volver"))}</button></div>`;
  if (paso === 3) cuerpo += `<section class="inscripciones-borrador"><h4>${escapar(t("borrador_subsanacion"))}</h4><p class="inscripciones-respuesta-revision">${escapar(ui.respuesta)}</p><p class="inscripciones-nota">${escapar(t("respuesta_preparada"))}</p><button type="button" disabled aria-disabled="true" title="${escapar(t("aportar_pendiente"))}">${escapar(t("aportar"))}</button><p class="inscripciones-nota">${escapar(t("aportar_pendiente"))}</p></section><div class="inscripciones-acciones"><button type="button" class="inscripciones-boton-secundario" data-inscripciones-subsanacion-atras>${escapar(t("volver"))}</button></div>`;
  cuerpo += `<div class="inscripciones-acciones"><button type="button" class="inscripciones-boton-secundario" data-inscripciones-subsanacion-reiniciar>${escapar(t("reiniciar"))}</button></div>`;
  return panel(t("subsanaciones"), cuerpo, t("no_enviado"));
}

function contenidoPreparar(datos, ui, t) {
  const item = datos.convocatoriaSeleccionada;
  const persona = datos.persona;
  if (!item || !persona) return panel(t("preparar"), `<p class="inscripciones-vacio">${escapar(item ? t("sin_persona") : t("detalle_pendiente"))}</p>`, t("sin_efectos"));
  const clase = persona.condicion === "empleada" ? "empleada" : persona.condicion === "externa" ? "externa" : "persona";
  const paso = ui.confirmada ? ui.preparado ? 3 : 2 : 1;
  let cuerpo = pasos(["paso_confirmar", "paso_completar", "paso_revisar"], paso, t);
  cuerpo += `<div class="inscripciones-preparar-resumen"><div><span>${escapar(t("convocatorias"))}</span><strong>${escapar(texto(item.titulo, t("convocatorias")))}</strong></div><div><span>${escapar(t("persona"))}</span><strong>${escapar(t(clase))}</strong></div></div>`;
  if (paso === 1) cuerpo += `<label class="inscripciones-confirmacion"><input type="checkbox" data-inscripciones-confirmar ${ui.confirmacionMarcada ? "checked" : ""}><span>${escapar(t("confirmar"))}</span></label><p class="inscripciones-nota">${escapar(t("confirmar_ayuda"))}</p><button type="button" data-inscripciones-preparar ${ui.confirmacionMarcada ? "" : 'disabled aria-disabled="true"'}>${escapar(t("preparar_borrador"))}</button>`;
  if (paso === 2) cuerpo += `<div class="inscripciones-formulario"><div class="inscripciones-campo"><label for="inscripciones-nombre">${escapar(t("nombre"))}</label><input id="inscripciones-nombre" value="${escapar(texto(persona.nombre_visible, t("sin_persona")))}" readonly></div><div class="inscripciones-campo"><label for="inscripciones-contacto">${escapar(t("campo_contacto"))}</label><input id="inscripciones-contacto" data-inscripciones-contacto value="${escapar(ui.contacto ?? persona.contacto_visible ?? "")}" maxlength="120" aria-describedby="inscripciones-contacto-ayuda" ${ui.errorContacto ? 'aria-invalid="true"' : ""}><small id="inscripciones-contacto-ayuda">${escapar(t("campo_contacto_ayuda"))}</small></div><div class="inscripciones-campo inscripciones-campo-ancho"><label for="inscripciones-observaciones">${escapar(t("campo_observaciones"))}</label><textarea id="inscripciones-observaciones" data-inscripciones-observaciones maxlength="500" rows="2" aria-describedby="inscripciones-observaciones-ayuda">${escapar(ui.observaciones ?? "")}</textarea><small id="inscripciones-observaciones-ayuda">${escapar(t("campo_observaciones_ayuda"))}</small></div></div>${errorValidacion(ui.errorContacto ? t("campo_contacto_error") : "")}<div class="inscripciones-acciones"><button type="button" data-inscripciones-solicitud-siguiente>${escapar(t("siguiente"))}</button><button type="button" class="inscripciones-boton-secundario" data-inscripciones-solicitud-atras>${escapar(t("volver"))}</button></div>`;
  if (paso === 3) cuerpo += `<section class="inscripciones-borrador" aria-label="${escapar(t("borrador"))}"><h4>${escapar(t("borrador"))}</h4><dl class="inscripciones-metadatos"><div><dt>${escapar(t("nombre"))}</dt><dd>${escapar(texto(persona.nombre_visible, t("sin_persona")))}</dd></div><div><dt>${escapar(t("contacto"))}</dt><dd>${escapar(ui.contacto)}</dd></div>${ui.observaciones?.trim() ? `<div><dt>${escapar(t("campo_observaciones"))}</dt><dd>${escapar(ui.observaciones)}</dd></div>` : ""}</dl><p class="inscripciones-nota">${escapar(t("aviso_borrador"))}</p><button type="button" disabled aria-disabled="true" title="${escapar(t("registrar_pendiente"))}">${escapar(t("registrar"))}</button></section><div class="inscripciones-acciones"><button type="button" class="inscripciones-boton-secundario" data-inscripciones-solicitud-atras>${escapar(t("volver"))}</button></div>`;
  return panel(t("preparar"), cuerpo, t("sin_efectos"));
}

export function renderizarVistaInscripciones(datos = {}, ui = {}, opciones = {}) {
  const t = crearTraductorInscripciones(opciones.catalogo);
  const estado = ESTADOS.has(datos?.estado) ? datos.estado : "no_configurado";
  const pestana = PESTANAS.includes(ui.pestana) ? ui.pestana : "convocatorias";
  const aviso = { cargando: t("cargando"), vacio: t("vacio"), no_configurado: t("sin_conexion"), denegado: t("denegado"), error: t("error") }[estado];
  const contenido = estado !== "disponible" ? panel(t("consulta"), `<p role="status">${escapar(aviso)}</p>`) : {
    convocatorias: () => contenidoConvocatorias(datos, t, opciones.puedeAbrir === true),
    mis_inscripciones: () => contenidoMisInscripciones(datos, t),
    subsanaciones: () => contenidoSubsanaciones(datos, ui, t),
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
  const estadoInicial = () => ({ pestana: "convocatorias", confirmacionMarcada: false, confirmada: false, preparado: false, contacto: "", observaciones: "", subsanacionIndice: null, pasoSubsanacion: 1, requerimientoConfirmado: false, respuesta: "" });
  let ui = estadoInicial();
  const pintar = (foco) => {
    if (!activa) return;
    contenedor.innerHTML = renderizarVistaInscripciones(modelo, ui, { puedeAbrir: !!abrirDetallePublico });
    if (foco) contenedor.querySelector?.(foco)?.focus?.();
  };
  const alClick = (evento) => {
    const tab = evento.target?.closest?.("[data-inscripciones-tab]");
    if (tab) { ui = { ...ui, pestana: tab.dataset.inscripcionesTab }; pintar(`[data-inscripciones-tab="${ui.pestana}"]`); return; }
    const detalle = evento.target?.closest?.("[data-inscripciones-detalle]");
    if (detalle && abrirDetallePublico && modelo.estado === "disponible") {
      const item = lista(modelo.convocatorias)[Number(detalle.dataset.inscripcionesDetalle)];
      const id = texto(item?.identificador_publico, "");
      if (id) abrirDetallePublico(id);
      return;
    }
    const preparar = evento.target?.closest?.("[data-inscripciones-preparar]");
    if (preparar && !preparar.disabled && ui.confirmacionMarcada && modelo.estado === "disponible" && modelo.convocatoriaSeleccionada && modelo.persona) {
      ui = { ...ui, confirmada: true, preparado: false, contacto: texto(modelo.persona.contacto_visible, ""), observaciones: "", errorContacto: false };
      pintar("[data-inscripciones-titulo]");
      return;
    }
    if (evento.target?.closest?.("[data-inscripciones-solicitud-siguiente]") && ui.confirmada && !ui.preparado) {
      const contacto = contenedor.querySelector?.("[data-inscripciones-contacto]")?.value ?? ui.contacto;
      const observaciones = contenedor.querySelector?.("[data-inscripciones-observaciones]")?.value ?? ui.observaciones;
      const valido = typeof contacto === "string" && contacto.trim().length >= 1 && contacto.trim().length <= 120 && typeof observaciones === "string" && observaciones.length <= 500;
      ui = { ...ui, contacto: String(contacto ?? "").trim(), observaciones: String(observaciones ?? "").trim(), errorContacto: !valido, preparado: valido };
      pintar(valido ? "[data-inscripciones-titulo]" : "[data-inscripciones-error]");
      if (valido) anunciar(crearTraductorInscripciones()("aviso_preparado"), "informacion");
      return;
    }
    if (evento.target?.closest?.("[data-inscripciones-solicitud-atras]") && ui.confirmada) {
      ui = ui.preparado ? { ...ui, preparado: false } : { ...ui, confirmada: false, confirmacionMarcada: false, preparado: false, contacto: "", observaciones: "" };
      pintar("[data-inscripciones-titulo]");
      return;
    }
    const seleccionar = evento.target?.closest?.("[data-inscripciones-seleccionar-subsanacion]");
    if (seleccionar && modelo.estado === "disponible") {
      const indice = Number(seleccionar.dataset.inscripcionesSeleccionarSubsanacion);
      if (Number.isSafeInteger(indice) && indice >= 0 && indice < lista(modelo.subsanaciones).length) {
        ui = { ...ui, subsanacionIndice: indice, pasoSubsanacion: 1, requerimientoConfirmado: false, respuesta: "", errorSubsanacion: false };
        pintar("[data-inscripciones-titulo]");
      }
      return;
    }
    if (evento.target?.closest?.("[data-inscripciones-subsanacion-siguiente]") && ui.subsanacionIndice !== null) {
      if (ui.pasoSubsanacion === 1 && ui.requerimientoConfirmado) {
        ui = { ...ui, pasoSubsanacion: 2 };
        pintar("[data-inscripciones-titulo]");
      } else if (ui.pasoSubsanacion === 2) {
        const respuesta = contenedor.querySelector?.("[data-inscripciones-respuesta]")?.value ?? ui.respuesta;
        const valido = typeof respuesta === "string" && respuesta.trim().length >= 10 && respuesta.trim().length <= 1000;
        ui = { ...ui, respuesta: String(respuesta ?? "").trim(), errorSubsanacion: !valido, pasoSubsanacion: valido ? 3 : 2 };
        pintar(valido ? "[data-inscripciones-titulo]" : "[data-inscripciones-error]");
        if (valido) anunciar(crearTraductorInscripciones()("respuesta_preparada"), "informacion");
      }
      return;
    }
    if (evento.target?.closest?.("[data-inscripciones-subsanacion-atras]") && ui.subsanacionIndice !== null) {
      ui = { ...ui, pasoSubsanacion: ui.pasoSubsanacion === 3 ? 2 : 1, errorSubsanacion: false };
      pintar("[data-inscripciones-titulo]");
      return;
    }
    if (evento.target?.closest?.("[data-inscripciones-subsanacion-reiniciar]")) {
      ui = { ...ui, subsanacionIndice: null, pasoSubsanacion: 1, requerimientoConfirmado: false, respuesta: "", errorSubsanacion: false };
      pintar("[data-inscripciones-titulo]");
    }
  };
  const alCambio = (evento) => {
    const confirmar = evento.target?.closest?.("[data-inscripciones-confirmar]");
    if (confirmar) { ui = { ...ui, confirmacionMarcada: confirmar.checked }; pintar("[data-inscripciones-confirmar]"); return; }
    const requerimiento = evento.target?.closest?.("[data-inscripciones-confirmar-requerimiento]");
    if (requerimiento) { ui = { ...ui, requerimientoConfirmado: requerimiento.checked }; pintar("[data-inscripciones-confirmar-requerimiento]"); }
  };
  const alInput = (evento) => {
    if (evento.target?.closest?.("[data-inscripciones-contacto]")) ui = { ...ui, contacto: evento.target.value, errorContacto: false };
    else if (evento.target?.closest?.("[data-inscripciones-observaciones]")) ui = { ...ui, observaciones: evento.target.value };
    else if (evento.target?.closest?.("[data-inscripciones-respuesta]")) ui = { ...ui, respuesta: evento.target.value, errorSubsanacion: false };
  };
  const desmontar = () => { if (!activa) return; activa = false; contenedor.removeEventListener("click", alClick); contenedor.removeEventListener("change", alCambio); contenedor.removeEventListener("input", alInput); contenedor.remove(); };
  const actualizar = (nuevosDatos = {}) => { if (!activa) return; modelo = nuevosDatos; ui = estadoInicial(); pintar("[data-inscripciones-titulo]"); };
  contenedor.addEventListener("click", alClick);
  contenedor.addEventListener("change", alCambio);
  contenedor.addEventListener("input", alInput);
  pintar();
  registrarDesmontar?.(desmontar);
  return Object.freeze({ actualizar, desmontar });
}
