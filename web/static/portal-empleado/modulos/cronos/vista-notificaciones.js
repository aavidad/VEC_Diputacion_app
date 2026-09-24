import { MENSAJES_CRONOS_ES } from "./i18n.js";
import { MENSAJES_CRONOS_C9_ES } from "./i18n-c9.js";

const MAXIMO_TEXTO = 512;

function escaparHTML(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}

function traductor(mensajes) {
  const catalogo = { ...MENSAJES_CRONOS_ES, ...MENSAJES_CRONOS_C9_ES, ...mensajes };
  for (const clave of Object.keys(MENSAJES_CRONOS_C9_ES)) {
    if (typeof catalogo[clave] !== "string" || !catalogo[clave]) throw new Error(`texto i18n ausente: ${clave}`);
  }
  return (clave, variables = {}) => catalogo[clave].replace(/\{([a-z_]+)\}/gu,
    (_coincidencia, nombre) => String(variables[nombre] ?? ""));
}

function fechaCivilValida(fecha) {
  if (!/^\d{4}-\d{2}-\d{2}$/u.test(fecha)) return false;
  const [ano, mes, dia] = fecha.split("-").map(Number);
  const fechaUTC = new Date(Date.UTC(ano, mes - 1, dia));
  return fechaUTC.getUTCFullYear() === ano && fechaUTC.getUTCMonth() === mes - 1 && fechaUTC.getUTCDate() === dia;
}

/** Solo marca la estructura del borrador; no representa un envío ni un mensaje recibido. */
export function renderizarVistaNotificacionesCronos({ mensajes } = {}) {
  const traducir = traductor(mensajes);
  const t = (clave, variables) => escaparHTML(traducir(clave, variables));
  return `<section class="cronos-notificaciones" aria-labelledby="cronos-notificaciones-titulo" data-estado-entrega="sin_enviar">
    <article class="panel cronos-notificaciones-panel">
      <header class="cabecera-panel"><div><h3 id="cronos-notificaciones-titulo">${t("notificaciones_titulo")}</h3><p>${t("notificaciones_descripcion")}</p></div><span class="estado-chip neutro">${t("notificaciones_estado")}</span></header>
      <div class="cuerpo-panel"><p class="cronos-notificaciones-aviso" role="status">${t("notificaciones_estado_detalle")}</p>
        <form data-cronos-notificacion-form novalidate><div class="cronos-notificaciones-campos">
          <label>${t("notificaciones_tipo")}<select disabled aria-disabled="true" title="${t("notificaciones_tipo_pendiente")}"><option>${t("notificaciones_tipo_pendiente")}</option></select></label>
          <label>${t("notificaciones_fecha")}<input type="date" name="fecha" required></label>
          <label class="cronos-notificaciones-campo-ancho">${t("notificaciones_texto")}<textarea name="texto" rows="5" maxlength="${MAXIMO_TEXTO}" aria-describedby="cronos-notificaciones-texto-ayuda cronos-notificaciones-contador" required></textarea></label>
        </div><p id="cronos-notificaciones-texto-ayuda" class="cronos-notificaciones-nota">${t("notificaciones_texto_ayuda")}</p>
        <output id="cronos-notificaciones-contador" class="cronos-notificaciones-contador" aria-live="polite" data-cronos-notificacion-contador>${t("notificaciones_contador", { actual: 0, maximo: MAXIMO_TEXTO })}</output>
        <label class="cronos-notificaciones-adjunto">${t("notificaciones_adjunto")}<input type="file" disabled aria-disabled="true" title="${t("notificaciones_adjunto_pendiente")}"></label>
        <p class="cronos-notificaciones-nota">${t("notificaciones_adjunto_pendiente")}</p>
        <div class="cronos-notificaciones-acciones"><button type="button" class="boton-secundario" data-cronos-notificacion-revisar disabled>${t("notificaciones_revisar")}</button>
          <button type="button" class="boton-primario" disabled aria-disabled="true" title="${t("notificaciones_envio_pendiente")}">${t("notificaciones_enviar")}</button></div>
        <p class="cronos-notificaciones-nota">${t("notificaciones_envio_pendiente")}</p>
        <section class="cronos-notificaciones-revision" data-cronos-notificacion-revision aria-labelledby="cronos-notificaciones-revision-titulo" hidden>
          <h4 id="cronos-notificaciones-revision-titulo" tabindex="-1">${t("notificaciones_revision_titulo")}</h4>
          <dl><div><dt>${t("notificaciones_revision_fecha")}</dt><dd data-cronos-notificacion-fecha></dd></div><div><dt>${t("notificaciones_revision_texto")}</dt><dd data-cronos-notificacion-texto></dd></div></dl>
          <p class="cronos-notificaciones-limite">${t("notificaciones_revision_estado")}</p>
        </section></form></div>
    </article>
    <article class="panel cronos-notificaciones-panel"><header class="cabecera-panel"><div><h3>${t("notificaciones_mensajes_titulo")}</h3></div></header>
      <div class="cuerpo-panel"><p role="status">${t("notificaciones_mensajes_pendiente")}</p>
        <button type="button" class="boton-secundario" disabled aria-disabled="true" title="${t("notificaciones_archivo_pendiente")}">${t("notificaciones_archivar")}</button>
        <p class="cronos-notificaciones-nota">${t("notificaciones_archivo_pendiente")}</p></div></article>
  </section>`;
}

/** El texto vive únicamente en los controles del DOM y se elimina al desmontar. */
export function montarVistaNotificacionesCronos({ raiz, registrarDesmontar, anunciar = () => {}, mensajes, locale = "es-ES" } = {}) {
  if (!raiz?.append || !raiz.ownerDocument?.createElement || typeof anunciar !== "function"
    || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) {
    throw new TypeError("vista de notificaciones de Cronos no disponible");
  }
  const traducir = traductor(mensajes);
  const contenedor = raiz.ownerDocument.createElement("section");
  contenedor.innerHTML = renderizarVistaNotificacionesCronos({ mensajes });
  raiz.append(contenedor);
  const fecha = contenedor.querySelector('[name="fecha"]');
  const texto = contenedor.querySelector('[name="texto"]');
  const contador = contenedor.querySelector("[data-cronos-notificacion-contador]");
  const revisar = contenedor.querySelector("[data-cronos-notificacion-revisar]");
  const revision = contenedor.querySelector("[data-cronos-notificacion-revision]");
  const fechaRevisada = contenedor.querySelector("[data-cronos-notificacion-fecha]");
  const textoRevisado = contenedor.querySelector("[data-cronos-notificacion-texto]");
  const tituloRevision = contenedor.querySelector("#cronos-notificaciones-revision-titulo");
  const actualizar = () => {
    contador.textContent = traducir("notificaciones_contador", { actual: texto.value.length, maximo: MAXIMO_TEXTO });
    revisar.disabled = !fechaCivilValida(fecha.value) || !texto.value.trim() || texto.value.length > MAXIMO_TEXTO;
    revision.hidden = true;
    fechaRevisada.textContent = "";
    textoRevisado.textContent = "";
  };
  const mostrarRevision = () => {
    if (revisar.disabled) return;
    const fechaUTC = new Date(`${fecha.value}T12:00:00Z`);
    fechaRevisada.textContent = new Intl.DateTimeFormat(locale, { day: "numeric", month: "long", year: "numeric", timeZone: "UTC" }).format(fechaUTC);
    textoRevisado.textContent = texto.value.trim();
    revision.hidden = false;
    tituloRevision.focus?.();
    anunciar(traducir("notificaciones_revision_estado"));
  };
  const impedirEnvio = (evento) => evento.preventDefault();
  fecha.addEventListener("input", actualizar);
  texto.addEventListener("input", actualizar);
  revisar.addEventListener("click", mostrarRevision);
  contenedor.addEventListener("submit", impedirEnvio);
  let activa = true;
  const desmontar = () => {
    if (!activa) return;
    activa = false;
    fecha.removeEventListener("input", actualizar);
    texto.removeEventListener("input", actualizar);
    revisar.removeEventListener("click", mostrarRevision);
    contenedor.removeEventListener("submit", impedirEnvio);
    fecha.value = "";
    texto.value = "";
    fechaRevisada.textContent = "";
    textoRevisado.textContent = "";
    contenedor.remove();
  };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar });
}
