import { obtenerAtlasSinteticoRRHH, TEXTO_DATOS_FICTICIOS_RRHH } from "../../datos-sinteticos-rrhh.js";
import { crearTraductorCronos, MENSAJES_CRONOS_ES } from "./i18n.js";
import { MENSAJES_CRONOS_PERMISOS_ES } from "./i18n-permisos.js";

function crearTraductorRecorridos(mensajes) {
  const general = crearTraductorCronos({ ...MENSAJES_CRONOS_ES, ...mensajes });
  return (clave) => Object.hasOwn(MENSAJES_CRONOS_PERMISOS_ES, clave)
    ? String(mensajes?.[clave] ?? MENSAJES_CRONOS_PERMISOS_ES[clave])
    : general(clave);
}

function escaparHTML(valor) { return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;"); }
function tabla(t, caption, columnas, filas, seleccionable = false) {
  return `<div class="cronos-tabla-contenedor"><table class="cronos-tabla cronos-recorrido-tabla"><caption>${t(caption)}</caption><thead><tr>${columnas.map((c) => `<th scope="col">${t(c)}</th>`).join("")}</tr></thead><tbody>${filas.map((fila, n) => `<tr${seleccionable ? ` data-cronos-fila="${n}" tabindex="0" aria-selected="false"` : ""}>${fila.map((v, i) => `<${i ? "td" : 'th scope="row"'}>${escaparHTML(v)}</${i ? "td" : "th"}>`).join("")}</tr>`).join("")}</tbody></table></div>`;
}
function panel(t, titulo, contenido, clase = "") { return `<article class="cronos-recorrido-panel ${clase}"><h4>${t(titulo)}</h4>${contenido}</article>`; }
function accion(t, clave) { return `<button type="button" class="boton-primario" disabled aria-disabled="true" title="${t("presentacion_accion_pendiente")}">${t(clave)}</button>`; }
function filtros(t, grupo, claves) { return `<nav class="cronos-recorrido-filtros" aria-label="${t("presentacion_filtros")}">${claves.map((clave, i) => `<button type="button" data-cronos-control="${grupo}" aria-pressed="${i === 0}">${t(clave)}</button>`).join("")}</nav>`; }
function claseEstado(estado) {
  const valor = String(estado).toLocaleLowerCase("es");
  if (valor.includes("concedido") || valor.includes("resuelto")) return "exito";
  if (valor.includes("ajuste") || valor.includes("revisión")) return "peligro";
  return "aviso";
}
function chipEstado(estado) { return `<span class="cronos-estado cronos-estado-${claseEstado(estado)}">${escaparHTML(estado)}</span>`; }
function bandeja(t, { id, caption, columnas, filas, detalle }) {
  const cuerpo = filas.map((fila, n) => {
    const detalleID = `${id}-detalle-${n}`;
    const celdas = fila.map((valor, indice) => {
      if (indice === 0) return `<th scope="row"><button type="button" class="cronos-referencia" data-cronos-detalle="${detalleID}" aria-expanded="false" aria-controls="${detalleID}">${escaparHTML(valor)}</button></th>`;
      if (indice === fila.length - 1) return `<td>${chipEstado(valor)}</td>`;
      return `<td>${escaparHTML(valor)}</td>`;
    }).join("");
    return `<tr data-cronos-fila="${n}" aria-selected="false">${celdas}</tr><tr id="${detalleID}" class="cronos-fila-detalle" hidden><td colspan="${columnas.length}">${detalle(fila, n)}</td></tr>`;
  }).join("");
  return `<div class="cronos-tabla-contenedor"><table class="cronos-tabla cronos-recorrido-tabla"><caption>${t(caption)}</caption><thead><tr>${columnas.map((c) => `<th scope="col">${t(c)}</th>`).join("")}</tr></thead><tbody>${cuerpo}</tbody></table></div>`;
}
const ATLAS = obtenerAtlasSinteticoRRHH();
const PERSONA = Object.freeze({ nombre: ATLAS.persona_principal.nombre_visible, unidad: ATLAS.unidad.nombre_visible });
const MOVIMIENTOS = Object.freeze([["18/09/2026", "08:01", "Entrada", "Sede Provincial"], ["18/09/2026", "14:12", "Pausa", "Sede Provincial"], ["18/09/2026", "14:43", "Fin de pausa", "Sede Provincial"], ["18/09/2026", "16:06", "Salida", "Sede Provincial"]]);
const SOLICITUDES = Object.freeze([["Vacaciones", "05–09 oct.", "Pendiente de responsable"], ["Asuntos propios", "21 sep.", "Concedido"], ["Corrección de marcaje", "16 sep.", "En revisión"]]);
const EQUIPO = Object.freeze([["María del Carmen Ruiz Moreno", "Vacaciones · 3 días", "Pendiente"], ["Antonio López Fernández", "Corrección de salida", "Pendiente"], ["José Manuel García Torres", "Asuntos propios · 1 día", "Pendiente"]]);
const INCIDENCIAS = Object.freeze([["Antonio López Fernández", "Salida no registrada · 16 sep.", "En revisión"], ["Lucía Martín Paredes", "Solapamiento de ausencia", "Requiere ajuste"], ["Miguel Ángel Ortega Gil", "Calendario sin asignar", "Pendiente"]]);

function formularioPermisos(t) {
  return `<div class="cronos-alta" data-cronos-alta-panel hidden>
    <ol class="cronos-permisos-pasos" aria-label="${t("permisos_pasos")}">
      <li data-cronos-paso-indicador="1" data-paso="1" data-estado="actual" aria-current="step">${t("permisos_paso_1")}</li>
      <li data-cronos-paso-indicador="2" data-paso="2">${t("permisos_paso_2")}</li>
      <li data-cronos-paso-indicador="3" data-paso="3">${t("permisos_paso_3")}</li>
    </ol>
    <div class="cronos-permisos-rejilla"><form class="cronos-permisos-formulario" data-cronos-permisos-formulario novalidate>
      <section class="cronos-permisos-paso" data-cronos-paso="1" aria-labelledby="cronos-permisos-paso-1-titulo">
        <h5 id="cronos-permisos-paso-1-titulo">${t("permisos_paso_1")}</h5>
        <details class="cronos-permisos-ayuda"><summary>${t("permisos_ayuda")}</summary><p>${t("permisos_ayuda_contenido")}</p></details>
        <label>${t("presentacion_form_tipo")}<select name="tipo"><option value="vacaciones">${t("presentacion_form_vacaciones")}</option><option value="correccion">${t("presentacion_form_correccion")}</option></select></label>
        <p class="cronos-permisos-nota">${t("permisos_catalogo_provisional")}</p>
        <label>${t("permisos_desde")}<input type="date" name="desde" required aria-describedby="cronos-permisos-error"></label>
        <label>${t("permisos_hasta")}<input type="date" name="hasta" required aria-describedby="cronos-permisos-error"></label>
        <p class="cronos-permisos-nota">${t("permisos_periodo_ayuda")}</p>
        <p class="cronos-permisos-error" id="cronos-permisos-error" data-cronos-permisos-error role="alert" hidden>${t("permisos_error_fechas")}</p>
        <div class="cronos-permisos-acciones"><button type="button" class="boton-primario" data-cronos-paso-siguiente>${t("permisos_siguiente")}</button></div>
      </section>
      <section class="cronos-permisos-paso" data-cronos-paso="2" aria-labelledby="cronos-permisos-paso-2-titulo" hidden>
        <h5 id="cronos-permisos-paso-2-titulo">${t("permisos_paso_2")}</h5>
        <label>${t("presentacion_form_observacion")}<textarea name="observacion" maxlength="500" rows="3"></textarea></label>
        <p class="cronos-permisos-nota">${t("permisos_observacion_ayuda")}</p>
        <label>${t("permisos_justificante")}<input type="text" name="documento_ref" maxlength="120"></label>
        <p class="cronos-permisos-nota">${t("permisos_justificante_ayuda")}</p>
        <div class="cronos-permisos-acciones"><button type="button" class="boton-secundario" data-cronos-paso-anterior>${t("permisos_anterior")}</button><button type="button" class="boton-primario" data-cronos-paso-siguiente>${t("permisos_siguiente")}</button></div>
      </section>
      <section class="cronos-permisos-paso" data-cronos-paso="3" aria-labelledby="cronos-permisos-paso-3-titulo" hidden>
        <h5 id="cronos-permisos-paso-3-titulo">${t("permisos_paso_3")}</h5>
        <p class="cronos-permisos-nota">${t("permisos_revision")}</p>
        <p class="cronos-permisos-nota">${t("permisos_sin_calculo")}</p>
        <div class="cronos-permisos-acciones"><button type="button" class="boton-secundario" data-cronos-paso-anterior>${t("permisos_anterior")}</button><button type="button" class="boton-primario" disabled aria-disabled="true" title="${t("permisos_sin_registro")}">${t("permisos_registrar")}</button></div>
      </section>
    </form><aside class="cronos-permisos-resumen" aria-labelledby="cronos-permisos-resumen-titulo"><h5 id="cronos-permisos-resumen-titulo">${t("permisos_resumen")}</h5>
      <dl><div><dt>${t("permisos_resumen_tipo")}</dt><dd data-cronos-resumen="tipo">${t("presentacion_form_vacaciones")}</dd></div><div><dt>${t("permisos_resumen_periodo")}</dt><dd data-cronos-resumen="periodo">${t("permisos_resumen_pendiente")}</dd></div><div><dt>${t("permisos_resumen_saldo")}</dt><dd data-cronos-resumen="saldo">${t("permisos_resumen_saldo_vacaciones")}</dd></div></dl>
      <p class="cronos-permisos-limite">${t("permisos_sin_registro")}</p>
    </aside></div>
  </div>`;
}

/** Recorrido visual para revisión RRHH: datos sintéticos, sin efectos ni acceso a red. */
export function renderizarRecorridosCronos({ mensajes = MENSAJES_CRONOS_ES } = {}) {
  const traducir = crearTraductorRecorridos(mensajes); const t = (clave) => escaparHTML(traducir(clave));
  const formulario = formularioPermisos(t);
  const solicitudes = bandeja(t, {
    id: "cronos-solicitudes", caption: "presentacion_tabla_solicitudes",
    columnas: ["presentacion_concepto", "presentacion_periodo", "presentacion_col_estado"], filas: SOLICITUDES,
    detalle: (fila) => `<dl class="cronos-presentacion-ficha cronos-presentacion-ficha-inline"><div><dt>${t("presentacion_concepto")}</dt><dd>${escaparHTML(fila[0])}</dd></div><div><dt>${t("presentacion_periodo")}</dt><dd>${escaparHTML(fila[1])}</dd></div><div><dt>${t("presentacion_col_estado")}</dt><dd>${chipEstado(fila[2])}</dd></div></dl>`,
  });
  const persona = `<section class="cronos-recorrido-etapa" id="cronos-persona" aria-labelledby="cronos-persona-titulo"><header><p class="sobrelinea">${t("recorridos_persona")}</p><h3 id="cronos-persona-titulo">${t("presentacion_persona_titulo")}</h3><p>${t("presentacion_persona_descripcion")}</p></header><div class="cronos-recorrido-rejilla">
    ${panel(t, "presentacion_solicitudes", `<div class="cronos-cabecera-bandeja"><button type="button" class="boton-primario" data-cronos-alta aria-expanded="false">${t("presentacion_solicitar")}</button></div>${formulario}${solicitudes}`, "cronos-recorrido-panel-ancho cronos-panel-principal")}
    ${panel(t, "presentacion_jornada", `<div class="cronos-presentacion-datos"><strong>07:30</strong><span>${t("presentacion_jornada_teorica")}</span><strong>07:21</strong><span>${t("presentacion_trabajado")}</span><strong class="cronos-saldo-aviso">−00:09</strong><span>${t("presentacion_saldo")}</span></div>${filtros(t, "persona-periodo", ["presentacion_hoy", "presentacion_semana", "presentacion_mes"])}`)}
    ${panel(t, "presentacion_movimientos", tabla(t, "presentacion_tabla_movimientos", ["presentacion_fecha", "presentacion_hora", "presentacion_tipo", "presentacion_origen"], MOVIMIENTOS))}
    ${panel(t, "presentacion_calendario", `<div class="cronos-mini-calendario" aria-label="${t("presentacion_calendario")}"><span>${t("presentacion_lun")}</span><span>${t("presentacion_mar")}</span><span>${t("presentacion_mie")}</span><span>${t("presentacion_jue")}</span><span>${t("presentacion_vie")}</span><b>15</b><b>16</b><b>17</b><b class="cronos-dia-activo">18</b><b>19</b></div><p class="cronos-recorrido-nota">${t("presentacion_calendario_nota")}</p>`)}
    ${panel(t, "presentacion_permisos", tabla(t, "presentacion_tabla_permisos", ["presentacion_concepto", "presentacion_disponible", "presentacion_solicitado", "presentacion_disfrutado"], [["Vacaciones", "14 días", "5 días", "3 días"], ["Asuntos propios", "4 días", "1 día", "1 día"], ["Bolsa horaria", "18:00 h", "02:00 h", "03:00 h"]]))}
    ${panel(t, "presentacion_mensajes", `<ul class="cronos-lista-mensajes"><li><strong>${t("presentacion_mensaje_1")}</strong><span>${t("presentacion_mensaje_1_detalle")}</span></li><li><strong>${t("presentacion_mensaje_2")}</strong><span>${t("presentacion_mensaje_2_detalle")}</span></li></ul>${accion(t, "presentacion_marcar_leido")}`)}
  </div></section>`;
  const responsable = `<section class="cronos-recorrido-etapa" id="cronos-responsable" aria-labelledby="cronos-responsable-titulo" hidden><header><p class="sobrelinea">${t("recorridos_responsable")}</p><h3 id="cronos-responsable-titulo">${t("presentacion_responsable_titulo")}</h3><p>${t("presentacion_responsable_descripcion")}</p></header><div class="cronos-recorrido-rejilla">
    ${panel(t, "presentacion_bandeja", `${filtros(t, "responsable-vista", ["presentacion_pendientes", "presentacion_equipo", "presentacion_historial"])}${bandeja(t, { id: "cronos-equipo", caption: "presentacion_tabla_equipo", columnas: ["presentacion_persona_nombre", "presentacion_tramite", "presentacion_col_estado"], filas: EQUIPO, detalle: (fila) => `<dl class="cronos-presentacion-ficha cronos-presentacion-ficha-inline"><div><dt>${t("presentacion_solicitante")}</dt><dd>${escaparHTML(fila[0])}</dd></div><div><dt>${t("presentacion_unidad")}</dt><dd>${PERSONA.unidad}</dd></div><div><dt>${t("presentacion_detalle_solicitud")}</dt><dd>${escaparHTML(fila[1])}</dd></div></dl><div class="cronos-acciones">${accion(t, "presentacion_aprobar")}${accion(t, "presentacion_devolver")}</div>` })}`, "cronos-recorrido-panel-ancho cronos-panel-principal")}
  </div></section>`;
  const rrhh = `<section class="cronos-recorrido-etapa" id="cronos-rrhh" aria-labelledby="cronos-rrhh-titulo" hidden><header><p class="sobrelinea">${t("recorridos_rrhh")}</p><h3 id="cronos-rrhh-titulo">${t("presentacion_rrhh_titulo")}</h3><p>${t("presentacion_rrhh_descripcion")}</p></header><div class="cronos-recorrido-rejilla">
    ${panel(t, "presentacion_incidencias", `${filtros(t, "rrhh-vista", ["presentacion_incidencias_abiertas", "presentacion_correcciones", "presentacion_calendarios"])}${bandeja(t, { id: "cronos-incidencias", caption: "presentacion_tabla_incidencias", columnas: ["presentacion_persona_nombre", "presentacion_incidencia", "presentacion_col_estado"], filas: INCIDENCIAS, detalle: (fila) => `<dl class="cronos-presentacion-ficha cronos-presentacion-ficha-inline"><div><dt>${t("presentacion_persona_nombre")}</dt><dd>${escaparHTML(fila[0])}</dd></div><div><dt>${t("presentacion_incidencia")}</dt><dd>${escaparHTML(fila[1])}</dd></div><div><dt>${t("presentacion_col_estado")}</dt><dd>${chipEstado(fila[2])}</dd></div></dl>` })}`, "cronos-recorrido-panel-ancho cronos-panel-principal")}
    ${panel(t, "presentacion_calendarios_rrhh", `<dl class="cronos-presentacion-ficha"><div><dt>${t("presentacion_calendario_vigente")}</dt><dd>${t("presentacion_calendario_valor")}</dd></div><div><dt>${t("presentacion_horario")}</dt><dd>${t("presentacion_horario_valor")}</dd></div><div><dt>${t("presentacion_notificaciones")}</dt><dd>${t("presentacion_notificaciones_valor")}</dd></div></dl><div class="cronos-acciones">${accion(t, "presentacion_guardar_calendario")}${accion(t, "presentacion_notificar")}</div>`)}
  </div></section>`;
  return `<section class="cronos-area cronos-recorridos" data-estado-entrega="visual_pendiente_backend" aria-labelledby="cronos-recorridos-titulo"><header class="cronos-encabezado"><div><p class="sobrelinea">${t("recorridos_sobrelinea")}</p><h2 id="cronos-recorridos-titulo">${t("presentacion_titulo")}</h2></div><span class="cronos-recorrido-pendiente">${t("presentacion_estado")}</span></header><nav class="cronos-recorrido-etapas" role="tablist" aria-label="${t("recorridos_etapas")}"><button type="button" role="tab" data-cronos-rol="cronos-persona" aria-controls="cronos-persona" aria-selected="true"><strong>${t("recorridos_persona")}</strong><span>${t("presentacion_etapa_persona")}</span></button><button type="button" role="tab" data-cronos-rol="cronos-responsable" aria-controls="cronos-responsable" aria-selected="false"><strong>${t("recorridos_responsable")}</strong><span>${t("presentacion_etapa_responsable")}</span></button><button type="button" role="tab" data-cronos-rol="cronos-rrhh" aria-controls="cronos-rrhh" aria-selected="false"><strong>${t("recorridos_rrhh")}</strong><span>${t("presentacion_etapa_rrhh")}</span></button></nav>${persona}${responsable}${rrhh}<p class="cronos-recorrido-privacidad">${escaparHTML(TEXTO_DATOS_FICTICIOS_RRHH)} · ${t("presentacion_datos_sinteticos")}</p></section>`;
}

export function montarVistaRecorridosCronos({ raiz, anunciar = () => {}, registrarDesmontar, mensajes = MENSAJES_CRONOS_ES } = {}) {
  if (!raiz?.append || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) throw new TypeError("vista de recorridos de Cronos no disponible");
  const documento = raiz.ownerDocument; if (!documento?.createElement) throw new TypeError("documento de Cronos no disponible");
  const contenedor = documento.createElement("section"); contenedor.dataset.cronosRecorridos = ""; contenedor.innerHTML = renderizarRecorridosCronos({ mensajes }); raiz.append(contenedor);
  const traducir = crearTraductorRecorridos(mensajes);
  let pasoActual = 1;
  const formularioPermiso = () => contenedor.querySelector?.("[data-cronos-permisos-formulario]");
  const actualizarResumen = () => {
    const campos = formularioPermiso()?.elements;
    if (!campos) return;
    const tipo = campos.namedItem("tipo")?.value;
    const desde = campos.namedItem("desde")?.value;
    const hasta = campos.namedItem("hasta")?.value;
    const valores = {
      tipo: traducir(tipo === "correccion" ? "presentacion_form_correccion" : "presentacion_form_vacaciones"),
      periodo: desde && hasta ? `${desde.split("-").reverse().join("/")} – ${hasta.split("-").reverse().join("/")}` : traducir("permisos_resumen_pendiente"),
      saldo: traducir(tipo === "correccion" ? "permisos_resumen_saldo_correccion" : "permisos_resumen_saldo_vacaciones"),
    };
    for (const [clave, valor] of Object.entries(valores)) {
      const nodo = contenedor.querySelector?.(`[data-cronos-resumen="${clave}"]`);
      if (nodo) nodo.textContent = valor;
    }
  };
  const mostrarPaso = (numero) => {
    pasoActual = numero;
    contenedor.querySelectorAll?.("[data-cronos-paso]").forEach((nodo) => { nodo.hidden = Number(nodo.dataset.cronosPaso) !== numero; });
    contenedor.querySelectorAll?.("[data-cronos-paso-indicador]").forEach((nodo) => {
      const orden = Number(nodo.dataset.cronosPasoIndicador);
      nodo.setAttribute("data-estado", orden < numero ? "hecho" : orden === numero ? "actual" : "pendiente");
      if (orden === numero) nodo.setAttribute("aria-current", "step"); else nodo.removeAttribute("aria-current");
    });
    contenedor.querySelector?.(`[data-cronos-paso="${numero}"] summary, [data-cronos-paso="${numero}"] input, [data-cronos-paso="${numero}"] select, [data-cronos-paso="${numero}"] button`)?.focus?.();
  };
  const periodoValido = () => {
    const campos = formularioPermiso()?.elements;
    if (!campos) return false;
    const desde = campos.namedItem("desde")?.value;
    const hasta = campos.namedItem("hasta")?.value;
    const valido = Boolean(desde && hasta && desde <= hasta && campos.namedItem("desde")?.validity?.valid !== false && campos.namedItem("hasta")?.validity?.valid !== false);
    const error = contenedor.querySelector?.("[data-cronos-permisos-error]");
    if (error) error.hidden = valido;
    if (!valido) { anunciar(traducir("permisos_error_fechas")); campos.namedItem("desde")?.focus?.(); }
    return valido;
  };
  const cambiar = (evento) => {
    const siguiente = evento.target?.closest?.("[data-cronos-paso-siguiente]");
    if (siguiente) { if (pasoActual !== 1 || periodoValido()) mostrarPaso(Math.min(3, pasoActual + 1)); return; }
    const anterior = evento.target?.closest?.("[data-cronos-paso-anterior]");
    if (anterior) { mostrarPaso(Math.max(1, pasoActual - 1)); return; }
    const control = evento.target?.closest?.("[data-cronos-control]");
    if (control) contenedor.querySelectorAll?.(`[data-cronos-control="${control.dataset.cronosControl}"]`).forEach((e) => e.setAttribute("aria-pressed", String(e === control)));
    const detalle = evento.target?.closest?.("[data-cronos-detalle]");
    if (detalle) {
      const destino = contenedor.querySelector?.(`#${detalle.dataset.cronosDetalle}`);
      const abrir = detalle.getAttribute?.("aria-expanded") !== "true";
      contenedor.querySelectorAll?.("[data-cronos-detalle]").forEach((e) => e.setAttribute("aria-expanded", "false"));
      contenedor.querySelectorAll?.(".cronos-fila-detalle").forEach((e) => { e.hidden = true; });
      detalle.setAttribute("aria-expanded", String(abrir));
      if (destino) destino.hidden = !abrir;
    }
    const alta = evento.target?.closest?.("[data-cronos-alta]");
    if (alta) {
      const panelAlta = contenedor.querySelector?.("[data-cronos-alta-panel]");
      const abrir = alta.getAttribute?.("aria-expanded") !== "true";
      contenedor.querySelectorAll?.("[data-cronos-alta]").forEach((e) => e.setAttribute("aria-expanded", String(abrir)));
      if (panelAlta) panelAlta.hidden = !abrir;
      if (abrir) mostrarPaso(1);
    }
    const rol = evento.target?.closest?.("[data-cronos-rol]");
    if (rol) {
      contenedor.querySelectorAll?.("[data-cronos-rol]").forEach((e) => {
        e.setAttribute("aria-selected", String(e === rol));
      });
      contenedor.querySelectorAll?.(".cronos-recorrido-etapa").forEach((e) => { e.hidden = e.id !== rol.dataset.cronosRol; });
      contenedor.querySelector?.(`#${rol.dataset.cronosRol}`)?.scrollIntoView?.({ block: "start", behavior: "smooth" });
    }
  };
  const cambiarCampo = (evento) => { if (evento.target?.closest?.("[data-cronos-permisos-formulario]")) actualizarResumen(); };
  const impedirEnvio = (evento) => { if (evento.target?.matches?.("[data-cronos-permisos-formulario]")) evento.preventDefault(); };
  contenedor.addEventListener?.("click", cambiar);
  contenedor.addEventListener?.("input", cambiarCampo);
  contenedor.addEventListener?.("change", cambiarCampo);
  contenedor.addEventListener?.("submit", impedirEnvio);
  let activa = true;
  const desmontar = () => { if (!activa) return; activa = false; contenedor.removeEventListener?.("click", cambiar); contenedor.removeEventListener?.("input", cambiarCampo); contenedor.removeEventListener?.("change", cambiarCampo); contenedor.removeEventListener?.("submit", impedirEnvio); contenedor.remove?.(); };
  registrarDesmontar?.(desmontar); return Object.freeze({ desmontar });
}
