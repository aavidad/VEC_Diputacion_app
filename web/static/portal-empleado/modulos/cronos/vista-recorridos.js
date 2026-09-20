import { crearTraductorCronos, MENSAJES_CRONOS_ES } from "./i18n.js";

function escaparHTML(valor) {
  return String(valor)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function sigueMontada(raiz, contenedor) {
  return raiz.querySelector?.("[data-cronos-recorridos]") === contenedor;
}

function retirar(raiz, contenedor) {
  if (!sigueMontada(raiz, contenedor)) return;
  if (typeof contenedor.remove === "function") contenedor.remove();
  else raiz.removeChild?.(contenedor);
}

function panel(titulo, ayuda, contenido) {
  return `<article class="cronos-recorrido-panel"><h4>${titulo}</h4><p>${ayuda}</p>${contenido}</article>`;
}

function vacio(t) {
  return `<p class="cronos-recorrido-vacio" role="status">${t("recorridos_sin_datos")}</p>`;
}

function tablaVacia(t, captionClave, columnas) {
  return `<div class="cronos-tabla-contenedor"><table class="cronos-tabla cronos-recorrido-tabla"><caption>${t(captionClave)}</caption><thead><tr>${columnas.map((clave) => `<th scope="col">${t(clave)}</th>`).join("")}</tr></thead><tbody><tr><td colspan="${columnas.length}" class="cronos-vacio">${t("recorridos_sin_registros")}</td></tr></tbody></table></div>`;
}

function controles(t, etiqueta, grupo, opciones) {
  return `<nav class="cronos-recorrido-filtros" aria-label="${t(etiqueta)}">${opciones.map((clave, indice) => `<button type="button" data-cronos-control="${grupo}" aria-pressed="${indice === 0}">${t(clave)}</button>`).join("")}</nav><p class="cronos-recorrido-vacio" role="status">${t("recorridos_no_consultado")}</p>`;
}

function boton(t, clave, ayudaClave) {
  return `<button type="submit" class="boton-primario" disabled aria-disabled="true" aria-describedby="${ayudaClave}">${t(clave)}</button>`;
}

function formulario(t, { ayudaClave, accionClave, campos }) {
  return `<form class="cronos-recorrido-formulario" aria-describedby="${ayudaClave}" novalidate>${campos}<p class="cronos-recorrido-aviso" id="${ayudaClave}">${t("recorridos_sin_servicio")}</p>${boton(t, accionClave, ayudaClave)}</form>`;
}

/**
 * Estructura visible, sin proveedor ni efectos, para conectar Cronos cuando
 * existan fuente WCRONOS, relación Persona→Empleado y autorización real.
 */
export function renderizarRecorridosCronos({ mensajes = MENSAJES_CRONOS_ES } = {}) {
  const traducir = crearTraductorCronos(mensajes);
  const t = (clave) => escaparHTML(traducir(clave));
  const campoFecha = `<label><span>${t("recorridos_form_fecha")}</span><input type="date" disabled aria-disabled="true"></label>`;
  const campoDetalle = `<label><span>${t("recorridos_form_detalle")}</span><textarea rows="2" disabled aria-disabled="true"></textarea></label>`;
  const campoMotivo = `<label><span>${t("recorridos_form_motivo")}</span><textarea rows="2" disabled aria-disabled="true"></textarea></label>`;
  const campoCalendario = `<label><span>${t("recorridos_form_calendario")}</span><select disabled aria-disabled="true"><option>${t("recorridos_form_opcion")}</option></select></label>`;
  const campoNotificacion = `<label><span>${t("recorridos_form_tipo_notificacion")}</span><select disabled aria-disabled="true"><option>${t("recorridos_form_opcion")}</option></select></label><label><span>${t("recorridos_form_adjunto")}</span><input type="file" disabled aria-disabled="true"></label>`;
  return `<section class="cronos-area cronos-recorridos" aria-labelledby="cronos-recorridos-titulo">
    <header class="cronos-encabezado"><div><p class="sobrelinea">${t("recorridos_sobrelinea")}</p><h2 id="cronos-recorridos-titulo">${t("recorridos_titulo")}</h2><p>${t("recorridos_descripcion")}</p></div><span class="cronos-recorrido-pendiente" role="status">${t("recorridos_pendiente")}</span></header>
    <aside class="cronos-recorrido-conexion" aria-label="${t("recorridos_pendiente")}"><strong>${t("recorridos_pendiente")}</strong><p>${t("recorridos_pendiente_detalle")}</p></aside>
    <nav class="cronos-recorrido-etapas" aria-label="${t("recorridos_etapas")}">
      <a href="#cronos-persona"><strong>1. ${t("recorridos_persona")}</strong><span>${t("recorridos_persona_ayuda")}</span></a>
      <a href="#cronos-responsable"><strong>2. ${t("recorridos_responsable")}</strong><span>${t("recorridos_responsable_ayuda")}</span></a>
      <a href="#cronos-rrhh"><strong>3. ${t("recorridos_rrhh")}</strong><span>${t("recorridos_rrhh_ayuda")}</span></a>
    </nav>
    <section class="cronos-recorrido-etapa" id="cronos-persona" aria-labelledby="cronos-persona-titulo"><header><p class="sobrelinea">${t("recorridos_persona")}</p><h3 id="cronos-persona-titulo">${t("recorridos_persona_titulo")}</h3></header><div class="cronos-recorrido-rejilla">
      ${panel(t("recorridos_saldo_hoy"), t("recorridos_jornada_ayuda"), controles(t, "recorridos_saldo_periodo", "saldo-periodo", ["recorridos_saldo_hoy", "recorridos_saldo_semana", "recorridos_saldo_mes", "recorridos_saldo_ano", "recorridos_saldo_seleccion", "recorridos_resumen", "recorridos_detalle"]))}
      ${panel(t("recorridos_movimientos"), t("recorridos_olvidos"), `${controles(t, "recorridos_movimientos", "movimientos-periodo", ["recorridos_movimientos_hoy", "recorridos_movimientos_semana", "recorridos_movimientos_mes", "recorridos_movimientos_ano", "recorridos_movimientos_seleccion", "recorridos_olvidos_marcaje", "recorridos_absentismos", "recorridos_calendario_anual"])}${tablaVacia(t, "recorridos_tabla_movimientos", ["recorridos_cab_periodo", "recorridos_cab_estado"])}`)}
      ${panel(t("recorridos_permisos_ano"), t("recorridos_solicitudes_ayuda"), `${controles(t, "recorridos_permisos_ano", "permisos-vista", ["recorridos_permisos_resumen", "recorridos_permisos_todos", "recorridos_permisos_justificar", "recorridos_permisos_conceder"])}${tablaVacia(t, "recorridos_tabla_permisos", ["recorridos_cab_permiso", "recorridos_cab_solicitar", "recorridos_cab_maximo", "recorridos_cab_minimo", "recorridos_cab_solicitado", "recorridos_cab_restante"])}${tablaVacia(t, "recorridos_tabla_concedidos", ["recorridos_cab_permiso", "recorridos_cab_inicio", "recorridos_cab_fin", "recorridos_cab_dias_horas", "recorridos_cab_ano"])}`)}
      ${panel(t("recorridos_correccion"), t("recorridos_correccion_ayuda"), formulario(t, { ayudaClave: "cronos-correccion-ayuda", accionClave: "recorridos_accion_solicitar", campos: `${campoFecha}${campoDetalle}` }))}
      ${panel(t("recorridos_notificaciones"), t("recorridos_historial_ayuda"), `${controles(t, "recorridos_notificaciones", "notificaciones-vista", ["recorridos_notificaciones_enviar", "recorridos_notificaciones_seleccion", "recorridos_notificaciones_pendientes"])}${formulario(t, { ayudaClave: "cronos-notificacion-ayuda", accionClave: "recorridos_accion_notificar", campos: `${campoNotificacion}${campoFecha}${campoDetalle}` })}`)}
      ${panel(t("recorridos_mensajes"), t("recorridos_mensajes_ayuda"), `${controles(t, "recorridos_mensajes", "mensajes-vista", ["recorridos_mensajes_pendientes", "recorridos_mensajes_consulta", "recorridos_mensajes_archivar"])}${tablaVacia(t, "recorridos_tabla_mensajes", ["recorridos_cab_periodo", "recorridos_cab_estado"])}`)}
    </div></section>
    <section class="cronos-recorrido-etapa" id="cronos-responsable" aria-labelledby="cronos-responsable-titulo"><header><p class="sobrelinea">${t("recorridos_responsable")}</p><h3 id="cronos-responsable-titulo">${t("recorridos_responsable_titulo")}</h3></header><div class="cronos-recorrido-rejilla">
      ${panel(t("recorridos_bandeja"), `${t("recorridos_bandeja_ayuda")} ${t("recorridos_permisos_pendientes")}.`, vacio(t))}
      ${panel(t("recorridos_evaluacion"), t("recorridos_evaluacion_ayuda"), formulario(t, { ayudaClave: "cronos-evaluacion-ayuda", accionClave: "recorridos_accion_evaluar", campos: campoMotivo }))}
      ${panel(t("recorridos_devolucion"), t("recorridos_devolucion_ayuda"), formulario(t, { ayudaClave: "cronos-devolucion-ayuda", accionClave: "recorridos_accion_devolver", campos: campoDetalle }))}
    </div></section>
    <section class="cronos-recorrido-etapa" id="cronos-rrhh" aria-labelledby="cronos-rrhh-titulo"><header><p class="sobrelinea">${t("recorridos_rrhh")}</p><h3 id="cronos-rrhh-titulo">${t("recorridos_rrhh_titulo")}</h3></header><div class="cronos-recorrido-rejilla">
      ${panel(t("recorridos_revision"), t("recorridos_revision_ayuda"), vacio(t))}
      ${panel(t("recorridos_resolucion"), t("recorridos_resolucion_ayuda"), formulario(t, { ayudaClave: "cronos-resolucion-ayuda", accionClave: "recorridos_accion_resolver", campos: campoMotivo }))}
      ${panel(t("recorridos_calendarios"), t("recorridos_calendarios_ayuda"), formulario(t, { ayudaClave: "cronos-calendario-ayuda", accionClave: "recorridos_accion_guardar", campos: campoCalendario }))}
      ${panel(t("recorridos_correcciones_rrhh"), t("recorridos_correcciones_rrhh_ayuda"), vacio(t))}
    </div></section>
    <p class="cronos-recorrido-privacidad">${t("recorridos_nota_privacidad")}</p>
  </section>`;
}

/** Montaje cancelable para el coordinador del Portal, sin peticiones ni estado persistente. */
export function montarVistaRecorridosCronos({ raiz, anunciar = () => {}, registrarDesmontar, mensajes = MENSAJES_CRONOS_ES } = {}) {
  if (!raiz?.append || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) {
    throw new TypeError("vista de recorridos de Cronos no disponible");
  }
  const documento = raiz.ownerDocument;
  if (!documento?.createElement) throw new TypeError("documento de Cronos no disponible");
  const contenedor = documento.createElement("section");
  contenedor.dataset.cronosRecorridos = "";
  contenedor.innerHTML = renderizarRecorridosCronos({ mensajes });
  raiz.append(contenedor);
  let activa = true;
  const cambiarControl = (evento) => {
    const boton = evento.target?.closest?.("[data-cronos-control]");
    if (!boton || !activa) return;
    const grupo = boton.dataset.cronosControl;
    contenedor.querySelectorAll?.(`[data-cronos-control="${grupo}"]`).forEach((elemento) => {
      elemento.setAttribute("aria-pressed", String(elemento === boton));
    });
  };
  contenedor.addEventListener?.("click", cambiarControl);
  const desmontar = () => {
    if (!activa) return;
    activa = false;
    contenedor.removeEventListener?.("click", cambiarControl);
    retirar(raiz, contenedor);
  };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar });
}
