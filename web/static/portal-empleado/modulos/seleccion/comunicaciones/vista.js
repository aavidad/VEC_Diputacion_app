import { crearTraductorSeleccionComunicaciones } from "./i18n.js";

const ESTADOS_FUENTE = new Set(["cargando", "disponible", "vacio", "no_configurado", "denegado", "error"]);
const HITOS = ["preparacion", "revision", "transporte", "entrega", "lectura"];
const ESTADOS_HITO = new Set(["pendiente", "preparada", "revisada", "observada", "solicitado", "aceptado", "fallido", "entregado", "leido", "sin_constancia"]);
const CANALES = new Set(["correo", "sms"]);
const e = (valor) => String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
const texto = (valor) => typeof valor === "string" ? valor.slice(0, 240) : "";
const claveEstado = (hito) => ESTADOS_HITO.has(hito?.estado) ? hito.estado : "sin_constancia";
const tono = (clave) => ["revisada", "entregado", "leido"].includes(clave) ? "exito" : clave === "fallido" || clave === "observada" ? "peligro" : ["pendiente", "sin_constancia"].includes(clave) ? "aviso" : "info";
function fechaVisible(valor, t) {
  if (typeof valor !== "string" || !/^\d{4}-\d\d-\d\dT/.test(valor)) return t("dato_no_disponible");
  const fecha = new Date(valor);
  return Number.isNaN(fecha.valueOf()) ? t("dato_no_disponible") : new Intl.DateTimeFormat("es-ES", { dateStyle: "short", timeStyle: "short", timeZone: "Europe/Madrid" }).format(fecha);
}
function pastilla(hito, t) {
  const clave = claveEstado(hito);
  return `<span class="s6-estado s6-estado--${tono(clave)}"><span aria-hidden="true" class="s6-punto"></span>${e(t(`estado_${clave}`))}</span>`;
}
function paso(clave, hito, t) {
  const estado = claveEstado(hito);
  const referencia = texto(hito?.referencia);
  return `<li class="s6-paso s6-paso--${tono(estado)}"><div class="s6-paso-cabecera"><strong>${e(t(clave))}</strong>${pastilla(hito, t)}</div><dl><div><dt>${e(t("fecha"))}</dt><dd>${e(fechaVisible(hito?.fecha, t))}</dd></div><div><dt>${e(t("referencia_evidencia"))}</dt><dd>${e(referencia || t("sin_constancia"))}</dd></div></dl></li>`;
}
function preparacion(t, canalConfigurado) {
  const motivo = canalConfigurado ? t("enviar_motivo_conector") : t("enviar_motivo_canal");
  return `<section class="panel s6-panel" aria-labelledby="s6-preparacion"><div class="cabecera-panel"><div><h3 id="s6-preparacion">${e(t("preparacion_titulo"))}</h3><p>${e(t("preparacion_subtitulo"))}</p></div></div><div class="cuerpo-panel s6-preparacion-cuerpo"><dl class="s6-datos"><div><dt>${e(t("canal"))}</dt><dd>${e(t(canalConfigurado ? "canal_configurado" : "canal_pendiente"))}</dd></div><div><dt>${e(t("audiencia"))}</dt><dd>${e(t("audiencia_pendiente"))}</dd></div><div><dt>${e(t("plantilla"))}</dt><dd>${e(t("plantilla_pendiente"))}</dd></div></dl><div class="s6-acciones"><button type="button" disabled aria-disabled="true" title="${e(t("revisar_motivo"))}">${e(t("revisar"))}</button><button type="button" class="s6-boton-principal" disabled aria-disabled="true" aria-describedby="s6-enviar-motivo">${e(t("enviar"))}</button><p id="s6-enviar-motivo">${e(motivo)}</p></div></div></section>`;
}
function lista(comunicaciones, filtro, canal, t) {
  const busqueda = texto(filtro).trim().toLocaleLowerCase("es");
  const visibles = comunicaciones.filter((item) => (canal === "todas" || item.canal === canal) && (!busqueda || `${texto(item.asunto)} ${texto(item.referencia)}`.toLocaleLowerCase("es").includes(busqueda)));
  return `<section class="panel s6-panel" aria-labelledby="s6-bandeja"><div class="cabecera-panel"><div><h3 id="s6-bandeja">${e(t("bandeja_titulo"))}</h3><p>${e(t("bandeja_subtitulo"))}</p></div></div><div class="cuerpo-panel"><form class="s6-filtros" data-s6-filtros><label>${e(t("buscar"))}<input name="filtro" value="${e(filtro)}" maxlength="120" autocomplete="off"></label><label>${e(t("canal"))}<select name="canal"><option value="todas">${e(t("todas"))}</option>${["correo", "sms"].map((clave) => `<option value="${clave}"${canal === clave ? " selected" : ""}>${e(t(clave))}</option>`).join("")}</select></label><button type="submit">${e(t("filtrar"))}</button></form><div class="s6-tabla-wrap" role="region" tabindex="0" aria-label="${e(t("tabla_region"))}"><table class="s6-tabla"><caption>${e(t("tabla_caption"))}</caption><thead><tr>${["asunto", "canal", "estado_preparacion", "estado_revision", "ver"].map((clave) => `<th scope="col">${e(t(clave))}</th>`).join("")}</tr></thead><tbody>${visibles.length ? visibles.map((item) => `<tr><th scope="row"><span class="s6-asunto">${e(texto(item.asunto) || t("dato_no_disponible"))}</span><small>${e(texto(item.referencia) || t("sin_constancia"))}</small></th><td>${e(CANALES.has(item.canal) ? t(item.canal) : t("dato_no_disponible"))}</td><td>${pastilla(item.preparacion, t)}</td><td>${pastilla(item.revision, t)}</td><td><button type="button" class="s6-ver" data-s6-seleccionar="${e(texto(item.referencia))}">${e(t("ver"))}</button></td></tr>`).join("") : `<tr><td colspan="5" class="s6-sin-resultados">${e(t("sin_resultados"))}</td></tr>`}</tbody></table></div></div></section>`;
}
function detalle(item, t) {
  return `<section class="panel s6-panel s6-detalle" aria-labelledby="s6-detalle"><div class="cabecera-panel"><div><h3 id="s6-detalle" tabindex="-1">${e(t("detalle_titulo"))}</h3><p>${e(item ? texto(item.asunto) : t("sin_seleccion"))}</p></div><details class="s6-ayuda"><summary aria-label="${e(t("ayuda_etiqueta"))}" title="${e(t("ayuda_etiqueta"))}">?</summary><p>${e(t("detalle_ayuda"))}</p></details></div><div class="cuerpo-panel">${item ? `<dl class="s6-resumen"><div><dt>${e(t("referencia"))}</dt><dd>${e(texto(item.referencia) || t("sin_constancia"))}</dd></div><div><dt>${e(t("destinatarios_resumen"))}</dt><dd>${e(texto(item.destinatariosResumen) || t("sin_constancia"))}</dd></div></dl><ol class="s6-pasos">${HITOS.map((clave) => paso(clave, item[clave], t)).join("")}</ol><p class="s6-limite">${e(t("limite"))}</p>` : `<p class="s6-vacio">${e(t("sin_seleccion"))}</p>`}</div></section>`;
}

/** Proyección S6 sin transporte propio. `modelo` debe venir de un consumidor autorizado. */
export function renderizarVistaSeleccionComunicaciones(modelo = {}, vista = {}) {
  modelo = modelo && typeof modelo === "object" ? modelo : {};
  const t = crearTraductorSeleccionComunicaciones();
  const estado = ESTADOS_FUENTE.has(modelo.estadoFuente) ? modelo.estadoFuente : "no_configurado";
  const comunicaciones = estado === "disponible" && Array.isArray(modelo.comunicaciones) ? modelo.comunicaciones.filter((item) => item && typeof item === "object").slice(0, 100) : [];
  const filtro = texto(vista.filtro);
  const canal = CANALES.has(vista.canal) ? vista.canal : "todas";
  const seleccionado = comunicaciones.find((item) => item.referencia === vista.seleccion) || comunicaciones[0];
  const estadoReal = estado === "disponible" && !comunicaciones.length ? "vacio" : estado;
  return `<section class="s6-comunicaciones" data-s6-comunicaciones><header class="s6-cabecera"><div><p class="sobrelinea">${e(t("sobrelinea"))}</p><h2>${e(t("titulo"))}</h2><p>${e(t("descripcion"))}</p></div><details class="s6-ayuda"><summary aria-label="${e(t("ayuda_etiqueta"))}" title="${e(t("ayuda_etiqueta"))}">?</summary><p>${e(t("ayuda"))}</p></details></header><p class="s6-aviso" role="status" data-estado-fuente="${estadoReal}">${e(t(`estado_${estadoReal}`))}${estadoReal === "no_configurado" ? ` ${e(t("fuente_pendiente"))}` : ""}</p>${preparacion(t, modelo.canalCorporativoConfigurado === true)}<div class="s6-trabajo">${estadoReal === "disponible" ? lista(comunicaciones, filtro, canal, t) : `<section class="panel s6-panel" aria-labelledby="s6-bandeja"><div class="cabecera-panel"><h3 id="s6-bandeja">${e(t("bandeja_titulo"))}</h3></div><div class="cuerpo-panel"><p class="s6-vacio">${e(t(`estado_${estadoReal}`))}</p></div></section>`}${detalle(estadoReal === "disponible" ? seleccionado : null, t)}</div></section>`;
}

/** Firma de montaje para el coordinador: sin red ni persistencia; admite actualización de proyección. */
export function montarVistaSeleccionComunicaciones({ raiz, modelo = {}, registrarDesmontar } = {}) {
  if (!raiz?.replaceChildren || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) throw new TypeError("vista S6 no disponible");
  let activa = true;
  let proyeccion = modelo;
  let vista = { filtro: "", canal: "todas", seleccion: "" };
  const pintar = () => { if (activa) raiz.innerHTML = renderizarVistaSeleccionComunicaciones(proyeccion, vista); };
  const alEnviar = (evento) => {
    const formulario = evento.target?.closest?.("[data-s6-filtros]");
    if (!formulario) return;
    evento.preventDefault();
    vista = { ...vista, filtro: formulario.elements.filtro.value, canal: formulario.elements.canal.value };
    pintar();
  };
  const alClick = (evento) => {
    const boton = evento.target?.closest?.("[data-s6-seleccionar]");
    if (!boton) return;
    vista = { ...vista, seleccion: boton.dataset.s6Seleccionar };
    pintar();
    raiz.querySelector("#s6-detalle")?.focus();
  };
  raiz.addEventListener("submit", alEnviar);
  raiz.addEventListener("click", alClick);
  pintar();
  const desmontar = () => { if (!activa) return; activa = false; raiz.removeEventListener("submit", alEnviar); raiz.removeEventListener("click", alClick); raiz.replaceChildren(); };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ actualizarModelo: (siguiente) => { if (activa) { proyeccion = siguiente || {}; pintar(); } }, desmontar });
}
