import { crearTraductorCronos, MENSAJES_CRONOS_ES } from "./i18n.js?v=20260924-web-paradas-periodos-v1";
import { MENSAJES_CRONOS_PERMISOS_ES } from "./i18n-permisos.js?v=20260924-f2-web2";
import { montarCatalogoPermisosCronos } from "./vista-catalogo-permisos.js?v=20260924-cronos-integrado-v1";
import { montarVistaCorreccionesCronos } from "./vista-correcciones.js?v=20260924-web-paradas-periodos-v1";
import { montarVistaNotificacionesCronos } from "./vista-notificaciones.js?v=20260924-web-paradas-periodos-v1";

function escaparHTML(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}

function crearTraductorRecorridos(mensajes) {
  const general = crearTraductorCronos({ ...MENSAJES_CRONOS_ES, ...mensajes });
  return (clave) => Object.hasOwn(MENSAJES_CRONOS_PERMISOS_ES, clave)
    ? String(mensajes?.[clave] ?? MENSAJES_CRONOS_PERMISOS_ES[clave])
    : general(clave);
}

function panel(t, titulo, contenido, clase = "") {
  return `<article class="cronos-recorrido-panel ${clase}"><h4>${t(titulo)}</h4>${contenido}</article>`;
}

function vacio(t) {
  return `<p class="cronos-recorrido-vacio" role="status">${t("permisos_no_configurado_detalle")}</p>`;
}

function formularioPermisos(t) {
  return `<div class="cronos-alta" id="cronos-permisos-alta" data-cronos-alta-panel hidden>
    <ol class="cronos-permisos-pasos" aria-label="${t("permisos_pasos")}">
      <li data-cronos-paso-indicador="1" data-paso="1" data-estado="actual" aria-current="step">${t("permisos_paso_1")}</li>
      <li data-cronos-paso-indicador="2" data-paso="2">${t("permisos_paso_2")}</li>
      <li data-cronos-paso-indicador="3" data-paso="3">${t("permisos_paso_3")}</li>
    </ol>
    <div class="cronos-permisos-rejilla"><form class="cronos-permisos-formulario" data-cronos-permisos-formulario>
      <section class="cronos-permisos-paso" data-cronos-paso="1" aria-labelledby="cronos-permisos-paso-1-titulo">
        <h5 id="cronos-permisos-paso-1-titulo" tabindex="-1">${t("permisos_paso_1")}</h5>
        <details class="cronos-permisos-ayuda"><summary aria-label="${t("permisos_ayuda")}">?</summary><p>${t("permisos_ayuda_contenido")}</p></details>
        <label>${t("presentacion_form_tipo")}<select name="tipo" disabled aria-disabled="true" title="${t("permisos_no_configurado")}"><option value="">${t("permisos_no_configurado")}</option></select></label>
        <p class="cronos-permisos-nota">${t("permisos_catalogo_ausente")}</p>
        <label>${t("permisos_desde")}<input type="date" name="desde" disabled aria-disabled="true" title="${t("permisos_no_configurado")}"></label>
        <label>${t("permisos_hasta")}<input type="date" name="hasta" disabled aria-disabled="true" title="${t("permisos_no_configurado")}"></label>
        <div class="cronos-permisos-acciones"><button type="button" class="boton-secundario" data-cronos-paso-siguiente>${t("permisos_siguiente")}</button></div>
      </section>
      <section class="cronos-permisos-paso" data-cronos-paso="2" aria-labelledby="cronos-permisos-paso-2-titulo" hidden>
        <h5 id="cronos-permisos-paso-2-titulo" tabindex="-1">${t("permisos_paso_2")}</h5>
        <label>${t("presentacion_form_observacion")}<textarea name="observacion" maxlength="500" rows="3" disabled aria-disabled="true" title="${t("permisos_no_configurado")}"></textarea></label>
        <p class="cronos-permisos-nota">${t("permisos_observacion_ayuda")}</p>
        <label>${t("permisos_justificante")}<input type="text" name="documento_ref" maxlength="120" disabled aria-disabled="true" title="${t("permisos_no_configurado")}"></label>
        <p class="cronos-permisos-nota">${t("permisos_justificante_ayuda")}</p>
        <div class="cronos-permisos-acciones"><button type="button" class="boton-secundario" data-cronos-paso-anterior>${t("permisos_anterior")}</button><button type="button" class="boton-secundario" data-cronos-paso-siguiente>${t("permisos_siguiente")}</button></div>
      </section>
      <section class="cronos-permisos-paso" data-cronos-paso="3" aria-labelledby="cronos-permisos-paso-3-titulo" hidden>
        <h5 id="cronos-permisos-paso-3-titulo" tabindex="-1">${t("permisos_paso_3")}</h5>
        <p class="cronos-permisos-nota">${t("permisos_revision_no_configurada")}</p>
        <div class="cronos-permisos-acciones"><button type="button" class="boton-secundario" data-cronos-paso-anterior>${t("permisos_anterior")}</button><button type="button" class="boton-primario" disabled aria-disabled="true" title="${t("permisos_sin_registro")}">${t("permisos_registrar")}</button></div>
      </section>
    </form><aside class="cronos-permisos-resumen" aria-labelledby="cronos-permisos-resumen-titulo"><h5 id="cronos-permisos-resumen-titulo">${t("permisos_resumen")}</h5>
      <dl><div><dt>${t("permisos_resumen_tipo")}</dt><dd>${t("permisos_no_configurado")}</dd></div><div><dt>${t("permisos_resumen_periodo")}</dt><dd>${t("permisos_resumen_vacio")}</dd></div><div><dt>${t("permisos_resumen_saldo")}</dt><dd>${t("permisos_resumen_vacio")}</dd></div></dl>
      <p class="cronos-permisos-limite">${t("permisos_sin_registro")}</p>
    </aside></div>
  </div>`;
}

/** Vista sin fuente ni concesión: no proyecta personas, saldos ni un catálogo supuesto. */
export function renderizarRecorridosCronos({ mensajes = MENSAJES_CRONOS_ES } = {}) {
  const traducir = crearTraductorRecorridos(mensajes);
  const t = (clave) => escaparHTML(traducir(clave));
  const persona = `<section class="cronos-recorrido-etapa" id="cronos-persona" aria-labelledby="cronos-persona-titulo"><header><p class="sobrelinea">${t("recorridos_persona")}</p><h3 id="cronos-persona-titulo">${t("presentacion_persona_titulo")}</h3><p>${t("permisos_no_configurado_detalle")}</p></header>
    <nav class="cronos-apartados" role="tablist" aria-label="${t("presentacion_persona_titulo")}">
      <button type="button" role="tab" data-cronos-apartado="solicitudes" aria-controls="cronos-apartado-solicitudes" aria-selected="true" tabindex="0">${t("apartado_solicitudes")}</button>
      <button type="button" role="tab" data-cronos-apartado="catalogo" aria-controls="cronos-apartado-catalogo" aria-selected="false" tabindex="-1">${t("apartado_catalogo")}</button>
      <button type="button" role="tab" data-cronos-apartado="correcciones" aria-controls="cronos-apartado-correcciones" aria-selected="false" tabindex="-1">${t("apartado_correcciones")}</button>
      <button type="button" role="tab" data-cronos-apartado="notificaciones" aria-controls="cronos-apartado-notificaciones" aria-selected="false" tabindex="-1">${t("apartado_notificaciones")}</button>
    </nav><div class="cronos-recorrido-rejilla" id="cronos-apartado-solicitudes" role="tabpanel" aria-label="${t("apartado_solicitudes")}">
    ${panel(t, "presentacion_solicitudes", `<div class="cronos-cabecera-bandeja"><button type="button" class="boton-secundario" data-cronos-alta aria-expanded="false" aria-controls="cronos-permisos-alta">${t("permisos_ver_pasos")}</button></div>${formularioPermisos(t)}${vacio(t)}`, "cronos-recorrido-panel-ancho cronos-panel-principal")}
    ${panel(t, "presentacion_jornada", vacio(t))}
    ${panel(t, "presentacion_movimientos", vacio(t))}
    ${panel(t, "presentacion_permisos", vacio(t))}
    </div><div id="cronos-apartado-catalogo" role="tabpanel" aria-label="${t("apartado_catalogo")}" data-cronos-hoja="catalogo" hidden></div>
    <div id="cronos-apartado-correcciones" role="tabpanel" aria-label="${t("apartado_correcciones")}" data-cronos-hoja="correcciones" hidden></div>
    <div id="cronos-apartado-notificaciones" role="tabpanel" aria-label="${t("apartado_notificaciones")}" data-cronos-hoja="notificaciones" hidden></div>
  </section>`;
  const responsable = `<section class="cronos-recorrido-etapa" id="cronos-responsable" aria-labelledby="cronos-responsable-titulo" hidden><header><p class="sobrelinea">${t("recorridos_responsable")}</p><h3 id="cronos-responsable-titulo">${t("presentacion_responsable_titulo")}</h3><p>${t("permisos_no_configurado_detalle")}</p></header><div class="cronos-recorrido-rejilla">
    ${panel(t, "presentacion_bandeja", vacio(t), "cronos-recorrido-panel-ancho")}
  </div></section>`;
  const rrhh = `<section class="cronos-recorrido-etapa" id="cronos-rrhh" aria-labelledby="cronos-rrhh-titulo" hidden><header><p class="sobrelinea">${t("recorridos_rrhh")}</p><h3 id="cronos-rrhh-titulo">${t("presentacion_rrhh_titulo")}</h3><p>${t("permisos_no_configurado_detalle")}</p></header><div class="cronos-recorrido-rejilla">
    ${panel(t, "presentacion_incidencias", vacio(t), "cronos-recorrido-panel-ancho")}
  </div></section>`;
  return `<section class="cronos-area cronos-recorridos" data-estado-entrega="no_configurado" aria-labelledby="cronos-recorridos-titulo"><header class="cronos-encabezado"><div><p class="sobrelinea">${t("recorridos_sobrelinea")}</p><h2 id="cronos-recorridos-titulo">${t("presentacion_titulo")}</h2></div><span class="cronos-recorrido-pendiente" role="status">${t("permisos_no_configurado")}</span></header><nav class="cronos-recorrido-etapas" role="tablist" aria-label="${t("recorridos_etapas")}"><button type="button" role="tab" data-cronos-rol="cronos-persona" aria-controls="cronos-persona" aria-selected="true" tabindex="0"><strong>${t("recorridos_persona")}</strong><span>${t("presentacion_etapa_persona")}</span></button><button type="button" role="tab" data-cronos-rol="cronos-responsable" aria-controls="cronos-responsable" aria-selected="false" tabindex="-1"><strong>${t("recorridos_responsable")}</strong><span>${t("presentacion_etapa_responsable")}</span></button><button type="button" role="tab" data-cronos-rol="cronos-rrhh" aria-controls="cronos-rrhh" aria-selected="false" tabindex="-1"><strong>${t("recorridos_rrhh")}</strong><span>${t("presentacion_etapa_rrhh")}</span></button></nav>${persona}${responsable}${rrhh}<p class="cronos-recorrido-privacidad">${t("recorridos_nota_privacidad")}</p></section>`;
}

export function montarVistaRecorridosCronos({ raiz, anunciar = () => {}, registrarDesmontar, mensajes = MENSAJES_CRONOS_ES } = {}) {
  if (!raiz?.append || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) throw new TypeError("vista de recorridos de Cronos no disponible");
  const documento = raiz.ownerDocument;
  if (!documento?.createElement) throw new TypeError("documento de Cronos no disponible");
  const contenedor = documento.createElement("section");
  contenedor.dataset.cronosRecorridos = "";
  contenedor.innerHTML = renderizarRecorridosCronos({ mensajes });
  raiz.append(contenedor);
  let pasoActual = 1;
  let apartadoActual = "solicitudes";
  let hoja;
  const limpiarHoja = () => { hoja?.desmontar(); hoja = undefined; };
  const activarApartado = (apartado) => {
    if (!new Set(["solicitudes", "catalogo", "correcciones", "notificaciones"]).has(apartado)) return;
    if (apartado === apartadoActual) return;
    limpiarHoja();
    apartadoActual = apartado;
    contenedor.querySelectorAll?.("[data-cronos-apartado]").forEach((nodo) => {
      const activo = nodo.dataset.cronosApartado === apartado;
      nodo.setAttribute("aria-selected", String(activo));
      nodo.setAttribute("tabindex", activo ? "0" : "-1");
    });
    contenedor.querySelectorAll?.("[id^='cronos-apartado-']").forEach((nodo) => { nodo.hidden = nodo.id !== `cronos-apartado-${apartado}`; });
    if (apartado === "solicitudes") return;
    const destino = contenedor.querySelector?.(`[data-cronos-hoja="${apartado}"]`);
    if (!destino) return;
    if (apartado === "catalogo") hoja = montarCatalogoPermisosCronos({ raiz: destino });
    if (apartado === "correcciones") hoja = montarVistaCorreccionesCronos({ raiz: destino, estado: "no_configurado", mensajes });
    if (apartado === "notificaciones") hoja = montarVistaNotificacionesCronos({ raiz: destino, anunciar, mensajes });
  };
  const mostrarPaso = (numero) => {
    pasoActual = numero;
    contenedor.querySelectorAll?.("[data-cronos-paso]").forEach((nodo) => { nodo.hidden = Number(nodo.dataset.cronosPaso) !== numero; });
    contenedor.querySelectorAll?.("[data-cronos-paso-indicador]").forEach((nodo) => {
      const orden = Number(nodo.dataset.cronosPasoIndicador);
      nodo.setAttribute("data-estado", orden < numero ? "hecho" : orden === numero ? "actual" : "pendiente");
      if (orden === numero) nodo.setAttribute("aria-current", "step"); else nodo.removeAttribute("aria-current");
    });
    contenedor.querySelector?.(`#cronos-permisos-paso-${numero}-titulo`)?.focus?.();
  };
  const activarRol = (rol) => {
    if (rol.dataset.cronosRol !== "cronos-persona") {
      limpiarHoja();
      activarApartado("solicitudes");
    }
    contenedor.querySelectorAll?.("[data-cronos-rol]").forEach((nodo) => {
      const activo = nodo === rol;
      nodo.setAttribute("aria-selected", String(activo));
      nodo.setAttribute("tabindex", activo ? "0" : "-1");
    });
    contenedor.querySelectorAll?.(".cronos-recorrido-etapa").forEach((nodo) => { nodo.hidden = nodo.id !== rol.dataset.cronosRol; });
    contenedor.querySelector?.(`#${rol.dataset.cronosRol}`)?.scrollIntoView?.({ block: "start", behavior: "smooth" });
  };
  const cambiar = (evento) => {
    const apartado = evento.target?.closest?.("[data-cronos-apartado]");
    if (apartado) { activarApartado(apartado.dataset.cronosApartado); return; }
    if (evento.target?.closest?.("[data-cronos-paso-siguiente]")) { mostrarPaso(Math.min(3, pasoActual + 1)); return; }
    if (evento.target?.closest?.("[data-cronos-paso-anterior]")) { mostrarPaso(Math.max(1, pasoActual - 1)); return; }
    const alta = evento.target?.closest?.("[data-cronos-alta]");
    if (alta) {
      const panelAlta = contenedor.querySelector?.("[data-cronos-alta-panel]");
      const abrir = alta.getAttribute?.("aria-expanded") !== "true";
      alta.setAttribute("aria-expanded", String(abrir));
      if (panelAlta) panelAlta.hidden = !abrir;
      if (abrir) mostrarPaso(1);
      return;
    }
    const rol = evento.target?.closest?.("[data-cronos-rol]");
    if (rol) activarRol(rol);
  };
  const teclado = (evento) => {
    const apartado = evento.target?.closest?.("[data-cronos-apartado]");
    if (apartado && ["ArrowLeft", "ArrowRight", "Home", "End"].includes(evento.key)) {
      const apartados = [...(contenedor.querySelectorAll?.("[data-cronos-apartado]") || [])];
      const indice = apartados.indexOf(apartado);
      if (indice < 0 || !apartados.length) return;
      evento.preventDefault();
      const siguiente = evento.key === "Home" ? 0 : evento.key === "End" ? apartados.length - 1
        : (indice + (evento.key === "ArrowRight" ? 1 : -1) + apartados.length) % apartados.length;
      activarApartado(apartados[siguiente].dataset.cronosApartado);
      apartados[siguiente].focus?.();
      return;
    }
    const rol = evento.target?.closest?.("[data-cronos-rol]");
    if (!rol || !["ArrowLeft", "ArrowRight", "ArrowUp", "ArrowDown", "Home", "End"].includes(evento.key)) return;
    const roles = [...(contenedor.querySelectorAll?.("[data-cronos-rol]") || [])];
    const indice = roles.indexOf(rol);
    if (indice < 0 || roles.length === 0) return;
    evento.preventDefault();
    const siguiente = evento.key === "Home" ? 0 : evento.key === "End" ? roles.length - 1
      : (indice + (["ArrowRight", "ArrowDown"].includes(evento.key) ? 1 : -1) + roles.length) % roles.length;
    activarRol(roles[siguiente]);
    roles[siguiente].focus?.();
  };
  const impedirEnvio = (evento) => { if (evento.target?.matches?.("[data-cronos-permisos-formulario]")) evento.preventDefault(); };
  contenedor.addEventListener?.("click", cambiar);
  contenedor.addEventListener?.("keydown", teclado);
  contenedor.addEventListener?.("submit", impedirEnvio);
  let activa = true;
  const desmontar = () => {
    if (!activa) return;
    activa = false;
    limpiarHoja();
    contenedor.removeEventListener?.("click", cambiar);
    contenedor.removeEventListener?.("keydown", teclado);
    contenedor.removeEventListener?.("submit", impedirEnvio);
    contenedor.remove?.();
  };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar });
}
