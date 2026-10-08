/** Lectura del catálogo efectivo B45 para RRHH; ninguna regla se calcula en el navegador. */
import { traducirPortal, LOCALIZACION_PORTAL, ZONA_HORARIA_PORTAL } from "../../portal-i18n.js?v=20261007-pantallas-textos-final-v1";

const esc = (valor) => String(valor ?? "").replaceAll("&", "&amp;")
  .replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
const t = (clave, variables) => esc(traducirPortal(`politica_cese_${clave}`, variables));
const plazoVisible = (meses) => meses === 0 ? t("inmediata") : t(meses === 1 ? "mes" : "meses", { meses });

function modalidad(clave) {
  const simple = clave.split(".").at(-1).split("|")[0];
  const etiqueta = simple === "interinidad" ? "interinidad" : simple === "relevo" ? "relevo" : "otro";
  return t(etiqueta);
}

export function renderizarVistaPoliticaCeseRRHH({ politica, estado = "lista", ayudaAbierta = false } = {}) {
  const cargando = estado === "cargando";
  const cabecera = `<header class="cabecera-panel"><div><h3>${t("titulo")}</h3><p>${t("subtitulo")}</p></div>
    <button type="button" class="boton-secundario politica-cese-ayuda-boton" data-politica-cese-ayuda aria-label="${t("ayuda_boton")}" aria-expanded="${ayudaAbierta}" aria-controls="politica-cese-ayuda">?</button></header>
    <p id="politica-cese-ayuda" class="politica-cese-ayuda" ${ayudaAbierta ? "" : "hidden"}>${t("ayuda")}</p>`;
  if (estado === "no_disponible") {
    return `<section class="panel politica-cese-rrhh" data-politica-cese aria-labelledby="politica-cese-no-disponible"><div class="cuerpo-panel vacio-controlado">
      <h3 id="politica-cese-no-disponible">${t("no_disponible_titulo")}</h3><p>${t("no_disponible_texto")}</p>
      <div class="acciones-vista"><button type="button" class="boton-secundario" data-vista="portal">${esc(traducirPortal("accion_volver_portal"))}</button></div>
    </div></section>`;
  }
  if (!politica || estado !== "lista") {
    const clave = cargando ? "cargando" : estado === "denegada" ? "denegada" : "error";
    return `<section class="panel politica-cese-rrhh" data-politica-cese>${cabecera}<div class="cuerpo-panel" role="${cargando ? "status" : "alert"}">
      <p>${t(clave)}</p>${cargando || estado === "denegada" ? "" : `<button type="button" class="boton-secundario" data-politica-cese-actualizar>${t("actualizar")}</button>`}
    </div></section>`;
  }
  const fecha = new Intl.DateTimeFormat(LOCALIZACION_PORTAL, { dateStyle: "medium", timeStyle: "short", timeZone: ZONA_HORARIA_PORTAL })
    .format(new Date(politica.publicada_en));
  const filas = Object.entries(politica.mapeo).sort(([a], [b]) => a.localeCompare(b, LOCALIZACION_PORTAL))
    .map(([clave, clase]) => `<tr><th scope="row"><strong>${modalidad(clave)}</strong><small>${esc(clave)}</small></th>
      <td>${t(clase === "acumulacion_tareas" ? "acumulacion" : "general")}</td></tr>`).join("");
  return `<section class="politica-cese-rrhh" data-politica-cese>${cabecera}
    <div class="politica-cese-resumen">
      <article class="panel"><div class="cabecera-panel"><h3>${t("general")}</h3></div><div class="cuerpo-panel"><strong>${plazoVisible(politica.meses_general)}</strong></div></article>
      <article class="panel"><div class="cabecera-panel"><h3>${t("acumulacion")}</h3></div><div class="cuerpo-panel"><strong>${plazoVisible(politica.meses_acumulacion)}</strong></div></article>
    </div>
    <section class="panel"><div class="cabecera-panel"><div><h3>${t("mapeo")}</h3><p>${t("version", { version: politica.version })}</p></div>
      <span class="estado-chip aviso">${t("ejemplo")}</span></div>
      <div class="cuerpo-panel"><div class="politica-cese-tabla" role="region" tabindex="0" aria-label="${t("mapeo")}">
      <table class="tabla-datos"><thead><tr><th scope="col">${t("modalidad")}</th><th scope="col">${t("clase")}</th></tr></thead><tbody>${filas}</tbody></table></div>
      <p class="dato-secundario">${t("publicada")}: <time datetime="${esc(politica.publicada_en)}">${esc(fecha)}</time></p>
      <details class="politica-cese-fuente"><summary>${t("fuente")}</summary><code>${esc(politica.catalogo_ref)}</code><code>${esc(politica.catalogo_sha256)}</code></details>
      <button type="button" class="boton-secundario" data-politica-cese-actualizar>${t("actualizar")}</button></div></section>
  </section>`;
}

export function montarVistaPoliticaCeseRRHH({ raiz, politica, noDisponible = false, cliente, anunciar = () => {}, alDenegacion = () => {} } = {}) {
  if (!raiz?.addEventListener || !raiz?.removeEventListener || !raiz?.replaceChildren
    || typeof cliente?.consultar !== "function" || typeof anunciar !== "function" || typeof alDenegacion !== "function") {
    throw new TypeError("vista de política de cese no disponible");
  }
  let actual = politica;
  let estado = politica ? "lista" : noDisponible ? "no_disponible" : "error";
  let ayudaAbierta = false;
  let controlador = null;
  let montada = true;
  let secuencia = 0;
  const pintar = () => { if (montada) raiz.innerHTML = renderizarVistaPoliticaCeseRRHH({ politica: actual, estado, ayudaAbierta }); };
  async function actualizar() {
    controlador?.abort();
    const carga = ++secuencia;
    controlador = new AbortController();
    const signal = controlador.signal;
    estado = "cargando"; pintar();
    try {
      const recibida = await cliente.consultar({ signal });
      if (!montada || signal.aborted || carga !== secuencia) return;
      actual = recibida; estado = "lista"; pintar();
    } catch (error) {
      if (!montada || signal.aborted || carga !== secuencia) return;
      actual = null; estado = error?.estado === 401 || error?.estado === 403 ? "denegada" : error?.estado === 404 ? "no_disponible" : "error";
      if (estado === "denegada") alDenegacion();
      if (estado === "no_disponible") { pintar(); anunciar(`${traducirPortal("politica_cese_no_disponible_titulo")} ${traducirPortal("politica_cese_no_disponible_texto")}`); return; }
      pintar(); anunciar(traducirPortal(`politica_cese_${estado === "denegada" ? "denegada" : "error"}`), "error");
    }
  }
  const pulsar = (evento) => {
    if (evento.target?.closest?.("[data-politica-cese-ayuda]")) {
      ayudaAbierta = !ayudaAbierta; pintar(); raiz.querySelector?.("[data-politica-cese-ayuda]")?.focus?.({ preventScroll: true });
    } else if (evento.target?.closest?.("[data-politica-cese-actualizar]")) void actualizar();
  };
  raiz.addEventListener("click", pulsar); pintar();
  return Object.freeze({ desmontar() { montada = false; ++secuencia; controlador?.abort();
    raiz.removeEventListener("click", pulsar); raiz.replaceChildren(); } });
}
