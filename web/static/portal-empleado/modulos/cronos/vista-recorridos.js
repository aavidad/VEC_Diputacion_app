import { crearTraductorCronos, MENSAJES_CRONOS_ES } from "./i18n.js?v=20260925-aspecto-v1";
import { montarCatalogoPermisosCronos } from "./vista-catalogo-permisos.js?v=20260924-cronos-integrado-v1";
import { montarVistaCorreccionesCronos } from "./vista-correcciones.js?v=20260925-cronos-v1";
import { montarVistaNotificacionesCronos } from "./vista-notificaciones.js?v=20260925-cronos-v1";

function escaparHTML(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}

function crearTraductorRecorridos(mensajes) {
  return crearTraductorCronos({ ...MENSAJES_CRONOS_ES, ...mensajes });
}

function panel(t, titulo, contenido, clase = "") {
  return `<article class="cronos-recorrido-panel ${clase}"><h4>${t(titulo)}</h4>${contenido}</article>`;
}

function vacio(t) {
  return `<p class="cronos-recorrido-vacio" role="status">${t("recorridos_sin_datos")}</p>`;
}

/** Estructura sin datos: las consultas y decisiones requieren servicios y concesiones propios. */
export function renderizarRecorridosCronos({ mensajes = MENSAJES_CRONOS_ES } = {}) {
  const traducir = crearTraductorRecorridos(mensajes);
  const t = (clave) => escaparHTML(traducir(clave));
  const persona = `<section class="cronos-recorrido-etapa" id="cronos-persona" aria-labelledby="cronos-persona-titulo"><header><p class="sobrelinea">${t("recorridos_persona")}</p><h3 id="cronos-persona-titulo">${t("recorridos_persona_titulo")}</h3></header>
    <nav class="cronos-apartados" role="tablist" aria-label="${t("recorridos_persona_titulo")}">
      <button type="button" role="tab" data-cronos-apartado="solicitudes" aria-controls="cronos-apartado-solicitudes" aria-selected="true" tabindex="0">${t("apartado_solicitudes")}</button>
      <button type="button" role="tab" data-cronos-apartado="catalogo" aria-controls="cronos-apartado-catalogo" aria-selected="false" tabindex="-1">${t("apartado_catalogo")}</button>
      <button type="button" role="tab" data-cronos-apartado="correcciones" aria-controls="cronos-apartado-correcciones" aria-selected="false" tabindex="-1">${t("apartado_correcciones")}</button>
      <button type="button" role="tab" data-cronos-apartado="notificaciones" aria-controls="cronos-apartado-notificaciones" aria-selected="false" tabindex="-1">${t("apartado_notificaciones")}</button>
    </nav><div class="cronos-recorrido-rejilla" id="cronos-apartado-solicitudes" role="tabpanel" aria-label="${t("apartado_solicitudes")}">
    ${panel(t, "recorridos_solicitudes", vacio(t), "cronos-recorrido-panel-ancho cronos-panel-principal")}
    ${panel(t, "recorridos_jornada", vacio(t))}
    ${panel(t, "recorridos_movimientos", vacio(t))}
    ${panel(t, "recorridos_permisos_ano", vacio(t))}
    </div><div id="cronos-apartado-catalogo" role="tabpanel" aria-label="${t("apartado_catalogo")}" data-cronos-hoja="catalogo" hidden></div>
    <div id="cronos-apartado-correcciones" role="tabpanel" aria-label="${t("apartado_correcciones")}" data-cronos-hoja="correcciones" hidden></div>
    <div id="cronos-apartado-notificaciones" role="tabpanel" aria-label="${t("apartado_notificaciones")}" data-cronos-hoja="notificaciones" hidden></div>
  </section>`;
  const responsable = `<section class="cronos-recorrido-etapa" id="cronos-responsable" aria-labelledby="cronos-responsable-titulo" hidden><header><p class="sobrelinea">${t("recorridos_responsable")}</p><h3 id="cronos-responsable-titulo">${t("recorridos_responsable_titulo")}</h3></header><div class="cronos-recorrido-rejilla">
    ${panel(t, "recorridos_bandeja", vacio(t), "cronos-recorrido-panel-ancho")}
  </div></section>`;
  const rrhh = `<section class="cronos-recorrido-etapa" id="cronos-rrhh" aria-labelledby="cronos-rrhh-titulo" hidden><header><p class="sobrelinea">${t("recorridos_rrhh")}</p><h3 id="cronos-rrhh-titulo">${t("recorridos_rrhh_titulo")}</h3></header><div class="cronos-recorrido-rejilla">
    ${panel(t, "recorridos_revision", vacio(t), "cronos-recorrido-panel-ancho")}
  </div></section>`;
  return `<section class="cronos-area cronos-recorridos" data-estado-entrega="no_configurado" aria-labelledby="cronos-recorridos-titulo"><header class="cronos-encabezado"><div><p class="sobrelinea">${t("recorridos_sobrelinea")}</p><h2 id="cronos-recorridos-titulo">${t("recorridos_titulo")}</h2></div><span class="cronos-recorrido-pendiente" role="status">${t("recorridos_pendiente")}</span></header><nav class="cronos-recorrido-etapas" role="tablist" aria-label="${t("recorridos_etapas")}"><button type="button" role="tab" data-cronos-rol="cronos-persona" aria-controls="cronos-persona" aria-selected="true" tabindex="0">${t("recorridos_persona")}</button><button type="button" role="tab" data-cronos-rol="cronos-responsable" aria-controls="cronos-responsable" aria-selected="false" tabindex="-1">${t("recorridos_responsable")}</button><button type="button" role="tab" data-cronos-rol="cronos-rrhh" aria-controls="cronos-rrhh" aria-selected="false" tabindex="-1">${t("recorridos_rrhh")}</button></nav>${persona}${responsable}${rrhh}</section>`;
}

export function montarVistaRecorridosCronos({ raiz, anunciar = () => {}, registrarDesmontar, mensajes = MENSAJES_CRONOS_ES } = {}) {
  if (!raiz?.append || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) throw new TypeError("vista de recorridos de Cronos no disponible");
  const documento = raiz.ownerDocument;
  if (!documento?.createElement) throw new TypeError("documento de Cronos no disponible");
  const contenedor = documento.createElement("section");
  contenedor.dataset.cronosRecorridos = "";
  contenedor.innerHTML = renderizarRecorridosCronos({ mensajes });
  raiz.append(contenedor);
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
  contenedor.addEventListener?.("click", cambiar);
  contenedor.addEventListener?.("keydown", teclado);
  let activa = true;
  const desmontar = () => {
    if (!activa) return;
    activa = false;
    limpiarHoja();
    contenedor.removeEventListener?.("click", cambiar);
    contenedor.removeEventListener?.("keydown", teclado);
    contenedor.remove?.();
  };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar });
}
