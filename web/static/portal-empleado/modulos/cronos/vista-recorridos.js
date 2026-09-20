import { renderizarEstadoEntrega } from "../../estado-entrega.js";
import { obtenerAtlasSinteticoRRHH, TEXTO_DATOS_FICTICIOS_RRHH } from "../../datos-sinteticos-rrhh.js";
import { crearTraductorCronos, MENSAJES_CRONOS_ES } from "./i18n.js";

function escaparHTML(valor) { return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;"); }
function tabla(t, caption, columnas, filas, seleccionable = false) {
  return `<div class="cronos-tabla-contenedor"><table class="cronos-tabla cronos-recorrido-tabla"><caption>${t(caption)}</caption><thead><tr>${columnas.map((c) => `<th scope="col">${t(c)}</th>`).join("")}</tr></thead><tbody>${filas.map((fila, n) => `<tr${seleccionable ? ` data-cronos-fila="${n}" tabindex="0" aria-selected="${n === 0}"` : ""}>${fila.map((v, i) => `<${i ? "td" : 'th scope="row"'}>${escaparHTML(v)}</${i ? "td" : "th"}>`).join("")}</tr>`).join("")}</tbody></table></div>`;
}
function panel(t, titulo, contenido, clase = "") { return `<article class="cronos-recorrido-panel ${clase}"><h4>${t(titulo)}</h4>${contenido}</article>`; }
function accion(t, clave) { return `<button type="button" class="boton-primario" disabled aria-disabled="true" title="${t("presentacion_accion_pendiente")}">${t(clave)}</button>`; }
function filtros(t, grupo, claves) { return `<nav class="cronos-recorrido-filtros" aria-label="${t("presentacion_filtros")}">${claves.map((clave, i) => `<button type="button" data-cronos-control="${grupo}" aria-pressed="${i === 0}">${t(clave)}</button>`).join("")}</nav>`; }
const ATLAS = obtenerAtlasSinteticoRRHH();
const PERSONA = Object.freeze({ nombre: ATLAS.persona_principal.nombre_visible, unidad: ATLAS.unidad.nombre_visible });
const MOVIMIENTOS = Object.freeze([["18/09/2026", "08:01", "Entrada", "Sede Provincial"], ["18/09/2026", "14:12", "Pausa", "Sede Provincial"], ["18/09/2026", "14:43", "Fin de pausa", "Sede Provincial"], ["18/09/2026", "16:06", "Salida", "Sede Provincial"]]);
const SOLICITUDES = Object.freeze([["Vacaciones", "05–09 oct.", "Pendiente de responsable"], ["Asuntos propios", "21 sep.", "Concedido"], ["Corrección de marcaje", "16 sep.", "En revisión"]]);
const EQUIPO = Object.freeze([["María del Carmen Ruiz Moreno", "Vacaciones · 3 días", "Pendiente"], ["Antonio López Fernández", "Corrección de salida", "Pendiente"], ["José Manuel García Torres", "Asuntos propios · 1 día", "Pendiente"]]);
const INCIDENCIAS = Object.freeze([["Antonio López Fernández", "Salida no registrada · 16 sep.", "En revisión"], ["Lucía Martín Paredes", "Solapamiento de ausencia", "Requiere ajuste"], ["Miguel Ángel Ortega Gil", "Calendario sin asignar", "Pendiente"]]);

/** Recorrido visual para revisión RRHH: datos sintéticos, sin efectos ni acceso a red. */
export function renderizarRecorridosCronos({ mensajes = MENSAJES_CRONOS_ES } = {}) {
  const traducir = crearTraductorCronos(mensajes); const t = (clave) => escaparHTML(traducir(clave));
  const entrega = renderizarEstadoEntrega({ estado: "visual_pendiente_backend", resumen: traducir("presentacion_limite"), fuente: { etiqueta: "WCRONOS" }, conexion: "pendiente", pendientes: [traducir("presentacion_identidad"), traducir("presentacion_autorizacion"), traducir("presentacion_persistencia")] });
  const persona = `<section class="cronos-recorrido-etapa" id="cronos-persona" aria-labelledby="cronos-persona-titulo"><header><p class="sobrelinea">${t("recorridos_persona")}</p><h3 id="cronos-persona-titulo">${t("presentacion_persona_titulo")}</h3><p>${t("presentacion_persona_descripcion")}</p></header><div class="cronos-recorrido-rejilla">
    ${panel(t, "presentacion_jornada", `<div class="cronos-presentacion-datos"><strong>07:30</strong><span>${t("presentacion_jornada_teorica")}</span><strong>07:21</strong><span>${t("presentacion_trabajado")}</span><strong class="cronos-saldo-aviso">−00:09</strong><span>${t("presentacion_saldo")}</span></div>${filtros(t, "persona-periodo", ["presentacion_hoy", "presentacion_semana", "presentacion_mes"])}`)}
    ${panel(t, "presentacion_movimientos", tabla(t, "presentacion_tabla_movimientos", ["presentacion_fecha", "presentacion_hora", "presentacion_tipo", "presentacion_origen"], MOVIMIENTOS))}
    ${panel(t, "presentacion_calendario", `<div class="cronos-mini-calendario" aria-label="${t("presentacion_calendario")}"><span>${t("presentacion_lun")}</span><span>${t("presentacion_mar")}</span><span>${t("presentacion_mie")}</span><span>${t("presentacion_jue")}</span><span>${t("presentacion_vie")}</span><b>15</b><b>16</b><b>17</b><b class="cronos-dia-activo">18</b><b>19</b></div><p class="cronos-recorrido-nota">${t("presentacion_calendario_nota")}</p>`)}
    ${panel(t, "presentacion_permisos", tabla(t, "presentacion_tabla_permisos", ["presentacion_concepto", "presentacion_disponible", "presentacion_solicitado", "presentacion_disfrutado"], [["Vacaciones", "14 días", "5 días", "3 días"], ["Asuntos propios", "4 días", "1 día", "1 día"], ["Bolsa horaria", "18:00 h", "02:00 h", "03:00 h"]]))}
    ${panel(t, "presentacion_solicitudes", `${tabla(t, "presentacion_tabla_solicitudes", ["presentacion_concepto", "presentacion_periodo", "presentacion_col_estado"], SOLICITUDES, true)}<div class="cronos-acciones">${accion(t, "presentacion_solicitar")}${accion(t, "presentacion_correccion")}</div>`)}
    ${panel(t, "presentacion_mensajes", `<ul class="cronos-lista-mensajes"><li><strong>${t("presentacion_mensaje_1")}</strong><span>${t("presentacion_mensaje_1_detalle")}</span></li><li><strong>${t("presentacion_mensaje_2")}</strong><span>${t("presentacion_mensaje_2_detalle")}</span></li></ul>${accion(t, "presentacion_marcar_leido")}`)}
  </div></section>`;
  const responsable = `<section class="cronos-recorrido-etapa" id="cronos-responsable" aria-labelledby="cronos-responsable-titulo"><header><p class="sobrelinea">${t("recorridos_responsable")}</p><h3 id="cronos-responsable-titulo">${t("presentacion_responsable_titulo")}</h3><p>${t("presentacion_responsable_descripcion")}</p></header><div class="cronos-recorrido-rejilla">
    ${panel(t, "presentacion_bandeja", `${filtros(t, "responsable-vista", ["presentacion_pendientes", "presentacion_equipo", "presentacion_historial"])}${tabla(t, "presentacion_tabla_equipo", ["presentacion_persona_nombre", "presentacion_tramite", "presentacion_col_estado"], EQUIPO, true)}`, "cronos-recorrido-panel-ancho")}
    ${panel(t, "presentacion_detalle", `<dl class="cronos-presentacion-ficha"><div><dt>${t("presentacion_solicitante")}</dt><dd>${PERSONA.nombre}</dd></div><div><dt>${t("presentacion_unidad")}</dt><dd>${PERSONA.unidad}</dd></div><div><dt>${t("presentacion_detalle_solicitud")}</dt><dd>${t("presentacion_detalle_texto")}</dd></div></dl><div class="cronos-acciones">${accion(t, "presentacion_aprobar")}${accion(t, "presentacion_devolver")}</div>`)}
  </div></section>`;
  const rrhh = `<section class="cronos-recorrido-etapa" id="cronos-rrhh" aria-labelledby="cronos-rrhh-titulo"><header><p class="sobrelinea">${t("recorridos_rrhh")}</p><h3 id="cronos-rrhh-titulo">${t("presentacion_rrhh_titulo")}</h3><p>${t("presentacion_rrhh_descripcion")}</p></header><div class="cronos-recorrido-rejilla">
    ${panel(t, "presentacion_incidencias", `${filtros(t, "rrhh-vista", ["presentacion_incidencias_abiertas", "presentacion_correcciones", "presentacion_calendarios"])}${tabla(t, "presentacion_tabla_incidencias", ["presentacion_persona_nombre", "presentacion_incidencia", "presentacion_col_estado"], INCIDENCIAS, true)}`, "cronos-recorrido-panel-ancho")}
    ${panel(t, "presentacion_calendarios_rrhh", `<dl class="cronos-presentacion-ficha"><div><dt>${t("presentacion_calendario_vigente")}</dt><dd>${t("presentacion_calendario_valor")}</dd></div><div><dt>${t("presentacion_horario")}</dt><dd>${t("presentacion_horario_valor")}</dd></div><div><dt>${t("presentacion_notificaciones")}</dt><dd>${t("presentacion_notificaciones_valor")}</dd></div></dl><div class="cronos-acciones">${accion(t, "presentacion_guardar_calendario")}${accion(t, "presentacion_notificar")}</div>`)}
  </div></section>`;
  return `<section class="cronos-area cronos-recorridos" data-estado-entrega="visual_pendiente_backend" aria-labelledby="cronos-recorridos-titulo"><header class="cronos-encabezado"><div><p class="sobrelinea">${t("recorridos_sobrelinea")}</p><h2 id="cronos-recorridos-titulo">${t("presentacion_titulo")}</h2><p>${t("presentacion_descripcion")}</p></div><span class="cronos-recorrido-pendiente">${t("presentacion_estado")}</span></header>${entrega}<nav class="cronos-recorrido-etapas" aria-label="${t("recorridos_etapas")}"><button type="button" data-cronos-rol="cronos-persona" aria-controls="cronos-persona" aria-current="step"><strong>1. ${t("recorridos_persona")}</strong><span>${t("presentacion_etapa_persona")}</span></button><button type="button" data-cronos-rol="cronos-responsable" aria-controls="cronos-responsable"><strong>2. ${t("recorridos_responsable")}</strong><span>${t("presentacion_etapa_responsable")}</span></button><button type="button" data-cronos-rol="cronos-rrhh" aria-controls="cronos-rrhh"><strong>3. ${t("recorridos_rrhh")}</strong><span>${t("presentacion_etapa_rrhh")}</span></button></nav>${persona}${responsable}${rrhh}<p class="cronos-recorrido-privacidad">${escaparHTML(TEXTO_DATOS_FICTICIOS_RRHH)} · ${t("presentacion_datos_sinteticos")}</p></section>`;
}

export function montarVistaRecorridosCronos({ raiz, anunciar = () => {}, registrarDesmontar, mensajes = MENSAJES_CRONOS_ES } = {}) {
  if (!raiz?.append || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) throw new TypeError("vista de recorridos de Cronos no disponible");
  const documento = raiz.ownerDocument; if (!documento?.createElement) throw new TypeError("documento de Cronos no disponible");
  const contenedor = documento.createElement("section"); contenedor.dataset.cronosRecorridos = ""; contenedor.innerHTML = renderizarRecorridosCronos({ mensajes }); raiz.append(contenedor);
  const cambiar = (evento) => {
    const control = evento.target?.closest?.("[data-cronos-control]");
    if (control) contenedor.querySelectorAll?.(`[data-cronos-control="${control.dataset.cronosControl}"]`).forEach((e) => e.setAttribute("aria-pressed", String(e === control)));
    const fila = evento.target?.closest?.("[data-cronos-fila]");
    if (fila) fila.parentElement?.querySelectorAll?.("[data-cronos-fila]").forEach((e) => e.setAttribute("aria-selected", String(e === fila)));
    const rol = evento.target?.closest?.("[data-cronos-rol]");
    if (rol) {
      contenedor.querySelectorAll?.("[data-cronos-rol]").forEach((e) => {
        if (e === rol) e.setAttribute("aria-current", "step"); else e.removeAttribute?.("aria-current");
      });
      contenedor.querySelector?.(`#${rol.dataset.cronosRol}`)?.scrollIntoView?.({ block: "start", behavior: "smooth" });
    }
  };
  contenedor.addEventListener?.("click", cambiar); let activa = true;
  const desmontar = () => { if (!activa) return; activa = false; contenedor.removeEventListener?.("click", cambiar); contenedor.remove?.(); };
  registrarDesmontar?.(desmontar); return Object.freeze({ desmontar });
}
