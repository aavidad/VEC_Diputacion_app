import {
  documentoNotificacionCronos, fechaCivilVisibleCronos, instanteVisibleCronos,
} from "./i18n-notificaciones.js";
import { crearTraductorBandejaNotificacionesCronos } from "./i18n-bandeja-notificaciones.js?v=20261001-cronos-c9-recuperacion-v3";
import { ErrorClienteNotificacionesCronos, crearClienteNotificacionesCronosHTTP } from "./cliente-notificaciones-http.js";
import { LOCALIZACION_ACTUAL } from "../../../comun/idioma.js";

const FILTROS = Object.freeze(["pendientes", "atendidas"]);
const ERRORES = new Map([["no_competente", "error_no_competente_notificacion"]]);

function escaparHTML(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}
function claveNueva() { return globalThis.crypto.randomUUID(); }
function persona(n, t) { return n.empleado_etiqueta || t("persona_sin_nombre"); }

function fila(n, filtro, atendiendo, t, locale, zonaHoraria) {
  const documento = documentoNotificacionCronos(n, t);
  const accion = filtro === "pendientes"
    ? `<button type="button" class="boton-secundario" data-cronos-atender="${escaparHTML(n.notificacion_ref)}" aria-label="${escaparHTML(t("atender_notificacion", { persona: persona(n, t) }))}"${atendiendo ? " disabled" : ""}>${escaparHTML(t(atendiendo ? "atendiendo" : "atender"))}</button>`
    : `<span class="cronos-estado" data-estado="atendida">${escaparHTML(t("estado_atendida", { fecha: instanteVisibleCronos(n.atendida_en, locale, zonaHoraria) }))}</span>`;
  return `<tr><th scope="row">${escaparHTML(persona(n, t))}</th><td>${escaparHTML(n.tipo_nombre)}</td><td>${escaparHTML(fechaCivilVisibleCronos(n.fecha_referida, locale))}</td>
    <td class="cronos-notificacion-texto">${escaparHTML(n.texto)}</td><td>${documento}</td><td>${escaparHTML(instanteVisibleCronos(n.registrada_en, locale, zonaHoraria))}</td><td>${accion}</td></tr>`;
}

/** Bandeja de RRHH: notificaciones de las personas de su circuito, pendientes o atendidas. */
export function renderizarBandejaNotificacionesCronos({ estado = "cargando", filtro = "pendientes", datos = null, atendiendo = "", mensaje = "", tonoMensaje = "exito", recibo = null, confirmadas = [],
  mensajes, locale = LOCALIZACION_ACTUAL, zonaHoraria = "Europe/Madrid" } = {}) {
  const t = crearTraductorBandejaNotificacionesCronos(mensajes);
  if (!FILTROS.includes(filtro)) throw new RangeError("filtro de notificaciones no válido");
  const ayuda = t("abrir_ayuda", { asunto: t("bandeja_notificaciones_titulo") });
  const cabecera = `<header class="cronos-encabezado"><div><p class="sobrelinea">${escaparHTML(t("sobrelinea"))}</p><h2 id="cronos-bandeja-notificaciones-titulo">${escaparHTML(t("bandeja_notificaciones_titulo"))}</h2></div>
    <button type="button" class="cronos-boton-ayuda" data-accion="ayuda" aria-label="${escaparHTML(ayuda)}" title="${escaparHTML(ayuda)}"><span aria-hidden="true">?</span></button></header>`;
  const selector = `<div class="cronos-selector-paso" role="group" aria-label="${escaparHTML(t("bandeja_notificaciones_titulo"))}">${FILTROS.map((f) =>
    `<button type="button" class="boton-secundario" data-cronos-filtro-notificaciones="${f}" aria-pressed="${f === filtro}">${escaparHTML(t(`filtro_${f}`))}</button>`).join("")}</div>`;
  const cabeceraPanel = `<div class="cabecera-panel"><h3 id="cronos-bandeja-notificaciones-filtro">${escaparHTML(t(`filtro_${filtro}`))}</h3>${selector}</div>`;
  const tono = tonoMensaje === "error" ? "error" : "exito";
  const detalleRecibo = recibo ? `<details class="cronos-huella"><summary>${escaparHTML(t("detalle_recibo_atencion"))}</summary><dl>
    <dt>${escaparHTML(t("referencia_recibo_atencion"))}</dt><dd class="cronos-huella-valor">${escaparHTML(recibo.recibo_ref)}</dd>
    <dt>${escaparHTML(t("fecha_recibo_atencion"))}</dt><dd>${escaparHTML(instanteVisibleCronos(recibo.instante_utc, locale, zonaHoraria))}</dd></dl></details>` : "";
  const aviso = mensaje || recibo ? `<div class="cuerpo-panel">${mensaje ? `<p class="cronos-solicitud-aviso" data-tono="${tono}" role="${tono === "error" ? "alert" : "status"}">${escaparHTML(mensaje)}</p>` : ""}${detalleRecibo}</div>` : "";
  if (estado !== "listo") {
    const clave = { denegado: "denegado", sin_empleado: "sin_empleado", error: "error_actualizar_bandeja", demasiado_grande: "bandeja_demasiado_grande" }[estado] ?? "cargando";
    const alerta = estado === "error" || estado === "demasiado_grande";
    const reintentar = estado === "error" ? `<button type="button" class="boton-secundario" data-cronos-reintentar-bandeja>${escaparHTML(t("notificaciones_reintentar_consulta"))}</button>` : "";
    return `<section class="cronos-area cronos-bandeja-notificaciones" aria-labelledby="cronos-bandeja-notificaciones-titulo" data-estado="${escaparHTML(estado)}">${cabecera}
      <section class="panel cronos-panel" aria-labelledby="cronos-bandeja-notificaciones-filtro">${cabeceraPanel}${aviso}<div class="cuerpo-panel"><p class="cronos-${estado === "cargando" ? "vacio" : "acceso-denegado"}" role="${alerta ? "alert" : "status"}">${escaparHTML(t(clave))}</p>${reintentar}</div></section></section>`;
  }
  const visibles = datos.notificaciones.filter((n) => n.atendida === (filtro === "atendidas") && (filtro !== "pendientes" || !confirmadas.includes(n.notificacion_ref)));
  const cabeceras = ["col_persona", "col_tipo", "col_fecha", "col_mensaje", "col_documento", "col_enviada", filtro === "pendientes" ? "col_accion" : "col_estado"];
  const cuerpo = visibles.length ? visibles.map((n) => fila(n, filtro, atendiendo === n.notificacion_ref, t, locale, zonaHoraria)).join("")
    : `<tr><td colspan="${cabeceras.length}">${escaparHTML(t(filtro === "pendientes" ? "sin_pendientes" : "sin_atendidas"))}</td></tr>`;
  return `<section class="cronos-area cronos-bandeja-notificaciones" aria-labelledby="cronos-bandeja-notificaciones-titulo" data-estado="listo">${cabecera}
    <section class="panel cronos-panel" aria-labelledby="cronos-bandeja-notificaciones-filtro">${cabeceraPanel}${aviso}
      <div class="cronos-tabla-contenedor"><table class="cronos-tabla"><thead><tr>${cabeceras.map((c) => `<th scope="col">${escaparHTML(t(c))}</th>`).join("")}</tr></thead><tbody>${cuerpo}</tbody></table></div></section></section>`;
}

function estadoError(error) {
  if (error instanceof ErrorClienteNotificacionesCronos) {
    if (error.codigo === "sin_empleado") return "sin_empleado";
    if (["acceso_denegado", "autenticacion_requerida", "no_competente"].includes(error.codigo)) return "denegado";
    // Más de 500: el servidor la rechaza entera en vez de recortarla.
    if (error.codigo === "bandeja_demasiado_grande") return "demasiado_grande";
  }
  return "error";
}

export function montarBandejaNotificacionesCronos({ raiz, cliente = crearClienteNotificacionesCronosHTTP(), mensajes, anunciar = () => {}, registrarDesmontar,
  locale = LOCALIZACION_ACTUAL, zonaHoraria = "Europe/Madrid" } = {}) {
  if (!raiz?.append || !raiz.ownerDocument?.createElement || typeof cliente?.consultarBandeja !== "function" || typeof cliente?.atender !== "function"
    || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) {
    throw new TypeError("montaje de la bandeja de notificaciones de Cronos no disponible");
  }
  const t = crearTraductorBandejaNotificacionesCronos(mensajes);
  const contenedor = raiz.ownerDocument.createElement("section"); contenedor.dataset.cronosBandejaNotificaciones = ""; raiz.append(contenedor);
  let activa = true; let secuencia = 0; let controlador = null; let peticion = null;
  let filtro = "pendientes"; let estado = "cargando"; let datos = null; let mensaje = ""; let tonoMensaje = "exito";
  let recibo = null;
  const confirmadas = new Set();
  // Una atención a la vez; su clave se conserva en el reintento.
  let atencion = null;
  const dibujar = () => {
    if (activa) contenedor.innerHTML = renderizarBandejaNotificacionesCronos({ estado, filtro, datos, atendiendo: atencion?.enCurso ? atencion.notificacionRef : "",
      mensaje, tonoMensaje, recibo, confirmadas: [...confirmadas], mensajes, locale, zonaHoraria });
  };
  const cargar = async ({ recuperarFoco = false } = {}) => {
    if (!activa) return;
    controlador?.abort(); const consulta = new AbortController(); controlador = consulta; const turno = ++secuencia;
    estado = "cargando"; datos = null; dibujar();
    try {
      const r = await cliente.consultarBandeja({ signal: consulta.signal });
      if (!activa || turno !== secuencia || consulta.signal.aborted) return;
      estado = "listo"; datos = r; dibujar();
    } catch (error) {
      if (!activa || turno !== secuencia || consulta.signal.aborted) return;
      estado = estadoError(error); dibujar(); anunciar(t(estado === "demasiado_grande" ? "bandeja_demasiado_grande" : estado === "error" ? "error_actualizar_bandeja" : estado));
    }
    if (recuperarFoco) contenedor.querySelector?.(estado === "error" ? "[data-cronos-reintentar-bandeja]" : `[data-cronos-filtro-notificaciones="${filtro}"]`)?.focus?.();
  };
  const atender = async (notificacionRef) => {
    if (!activa || estado !== "listo" || atencion?.enCurso || confirmadas.has(notificacionRef)
      || !datos?.notificaciones.some((n) => n.notificacion_ref === notificacionRef && !n.atendida)) return;
    if (!atencion || atencion.notificacionRef !== notificacionRef) atencion = { notificacionRef, clave: claveNueva() };
    atencion.enCurso = true; mensaje = ""; dibujar();
    const operacion = atencion; const envio = new AbortController(); peticion = envio;
    try {
      const confirmado = await cliente.atender({ clave_operacion: operacion.clave, notificacion_ref: notificacionRef }, { signal: envio.signal });
      if (!activa || envio.signal.aborted) return;
      recibo = Object.freeze({ ...confirmado }); confirmadas.add(notificacionRef);
      atencion = null; mensaje = t(recibo.replay ? "ya_atendida" : "atendida"); tonoMensaje = "exito"; anunciar(mensaje);
      await cargar();
    } catch (error) {
      if (!activa || envio.signal.aborted) return;
      const codigo = error instanceof ErrorClienteNotificacionesCronos ? error.codigo : "";
      atencion.enCurso = false;
      if (codigo === "estado_cambiado") { confirmadas.add(notificacionRef); atencion = null; mensaje = t("ya_atendida"); tonoMensaje = "exito"; anunciar(mensaje); await cargar(); return; }
      mensaje = t(ERRORES.get(codigo) ?? "error_atender"); tonoMensaje = "error"; anunciar(mensaje); dibujar();
    }
  };
  const alPulsar = (evento) => {
    const botonReintentar = evento.target?.closest?.("[data-cronos-reintentar-bandeja]");
    if (botonReintentar && estado === "error" && !botonReintentar.disabled) { void cargar({ recuperarFoco: true }); return; }
    const botonFiltro = evento.target?.closest?.("[data-cronos-filtro-notificaciones]");
    if (botonFiltro && FILTROS.includes(botonFiltro.dataset.cronosFiltroNotificaciones)) { filtro = botonFiltro.dataset.cronosFiltroNotificaciones; dibujar(); return; }
    const botonAtender = evento.target?.closest?.("[data-cronos-atender]");
    if (botonAtender && !botonAtender.disabled) void atender(botonAtender.dataset.cronosAtender);
  };
  contenedor.addEventListener("click", alPulsar);
  void cargar();
  const desmontar = () => {
    if (!activa) return;
    activa = false; ++secuencia; controlador?.abort(); peticion?.abort();
    contenedor.removeEventListener("click", alPulsar); contenedor.remove?.();
  };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar, recargar: cargar });
}
