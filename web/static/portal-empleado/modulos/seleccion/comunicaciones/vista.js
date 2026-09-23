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
function comunicacionesDisponibles(modelo) {
  return modelo?.estadoFuente === "disponible" && Array.isArray(modelo.comunicaciones)
    ? modelo.comunicaciones.filter((item) => item && typeof item === "object").slice(0, 100)
    : [];
}
function filtrar(comunicaciones, filtro, canal) {
  const busqueda = texto(filtro).trim().toLocaleLowerCase("es");
  return comunicaciones.filter((item) => (canal === "todas" || item.canal === canal) && (!busqueda || `${texto(item.asunto)} ${texto(item.referencia)}`.toLocaleLowerCase("es").includes(busqueda)));
}
function conteoVisibles(numero, t) {
  return t(numero === 1 ? "visible_uno" : "visibles", { numero: new Intl.NumberFormat("es-ES").format(numero) });
}
function lista(visibles, filtro, canal, seleccionado, t) {
  const filtros = `<form class="s6-filtros" data-s6-filtros><label>${e(t("buscar"))}<input name="filtro" value="${e(filtro)}" maxlength="120" autocomplete="off"></label><label>${e(t("canal"))}<select name="canal"><option value="todas">${e(t("todas"))}</option>${["correo", "sms"].map((clave) => `<option value="${clave}"${canal === clave ? " selected" : ""}>${e(t(clave))}</option>`).join("")}</select></label><button type="submit">${e(t("filtrar"))}</button></form>`;
  const filas = visibles.map((item) => {
    const elegida = item === seleccionado;
    const accion = texto(item.referencia) ? `<button type="button" class="s6-ver" data-s6-seleccionar="${e(texto(item.referencia))}"${elegida ? ' aria-current="true"' : ""}>${e(t("ver"))}</button>` : "";
    const hitos = ["transporte", "entrega", "lectura"].map((clave) => `<div><dt>${e(t(clave))}</dt><dd>${pastilla(item[clave], t)}</dd></div>`).join("");
    return `<tr data-seleccionada="${elegida}"><th scope="row"><span class="s6-asunto">${e(texto(item.asunto) || t("dato_no_disponible"))}</span><small>${e(texto(item.referencia) || t("sin_constancia"))}</small>${accion}</th><td>${e(CANALES.has(item.canal) ? t(item.canal) : t("dato_no_disponible"))}</td><td><dl class="s6-fila-hitos">${hitos}</dl></td></tr>`;
  }).join("");
  const tabla = `<div class="s6-tabla-wrap" role="region" tabindex="0" aria-label="${e(t("tabla_region"))}" aria-describedby="s6-tabla-ayuda"><table class="s6-tabla"><caption>${e(t("tabla_caption"))}</caption><thead><tr>${["asunto", "canal", "hitos"].map((clave) => `<th scope="col">${e(t(clave))}</th>`).join("")}</tr></thead><tbody>${filas || `<tr><td colspan="3" class="s6-sin-resultados">${e(t("sin_resultados"))}</td></tr>`}</tbody></table></div><p id="s6-tabla-ayuda" class="s6-deslizar">${e(t("tabla_deslizar"))}</p>`;
  return `<section class="panel s6-panel" aria-labelledby="s6-bandeja"><div class="cabecera-panel"><div><h3 id="s6-bandeja">${e(t("bandeja_titulo"))}</h3><p>${e(t("bandeja_subtitulo"))}</p></div><span class="s6-contador" aria-live="polite">${e(conteoVisibles(visibles.length, t))}</span></div><div class="cuerpo-panel">${filtros}${tabla}</div></section>`;
}
function detalle(item, t) {
  return `<section class="panel s6-panel s6-detalle" aria-labelledby="s6-detalle"><div class="cabecera-panel"><div><h3 id="s6-detalle" tabindex="-1">${e(t("detalle_titulo"))}</h3><p>${e(item ? texto(item.asunto) : t("sin_seleccion"))}</p></div><details class="s6-ayuda"><summary aria-label="${e(t("ayuda_etiqueta"))}" title="${e(t("ayuda_etiqueta"))}">?</summary><p>${e(t("detalle_ayuda"))}</p></details></div><div class="cuerpo-panel">${item ? `<dl class="s6-resumen"><div><dt>${e(t("referencia"))}</dt><dd>${e(texto(item.referencia) || t("sin_constancia"))}</dd></div><div><dt>${e(t("destinatarios_resumen"))}</dt><dd>${e(texto(item.destinatariosResumen) || t("sin_constancia"))}</dd></div></dl><ol class="s6-pasos">${HITOS.map((clave) => paso(clave, item[clave], t)).join("")}</ol><p class="s6-limite">${e(t("limite"))}</p>` : `<p class="s6-vacio">${e(t("sin_seleccion"))}</p>`}</div></section>`;
}

/** Proyección S6 sin transporte propio. `modelo` debe venir de un consumidor autorizado. */
export function renderizarVistaSeleccionComunicaciones(modelo = {}, vista = {}) {
  modelo = modelo && typeof modelo === "object" ? modelo : {};
  vista = vista && typeof vista === "object" ? vista : {};
  const t = crearTraductorSeleccionComunicaciones();
  const estado = ESTADOS_FUENTE.has(modelo.estadoFuente) ? modelo.estadoFuente : "no_configurado";
  const comunicaciones = comunicacionesDisponibles(modelo);
  const estadoReal = estado === "disponible" && !comunicaciones.length ? "vacio" : estado;
  const filtro = texto(vista.filtro);
  const canal = CANALES.has(vista.canal) ? vista.canal : "todas";
  const visibles = filtrar(comunicaciones, filtro, canal);
  const seleccionado = visibles.find((item) => texto(item.referencia) && texto(item.referencia) === vista.seleccion) || visibles[0];
  const canalConfigurado = ["disponible", "vacio"].includes(estadoReal) && modelo.canalCorporativoConfigurado === true;
  const bandeja = estadoReal === "disponible" ? lista(visibles, filtro, canal, seleccionado, t) : `<section class="panel s6-panel" aria-labelledby="s6-bandeja"><div class="cabecera-panel"><h3 id="s6-bandeja">${e(t("bandeja_titulo"))}</h3></div><div class="cuerpo-panel"><p class="s6-vacio">${e(t(`estado_${estadoReal}`))}</p></div></section>`;
  return `<section class="s6-comunicaciones" data-s6-comunicaciones><header class="s6-cabecera"><div><p class="sobrelinea">${e(t("sobrelinea"))}</p><h2>${e(t("titulo"))}</h2><p>${e(t("descripcion"))}</p></div><details class="s6-ayuda"><summary aria-label="${e(t("ayuda_etiqueta"))}" title="${e(t("ayuda_etiqueta"))}">?</summary><p>${e(t("ayuda"))}</p></details></header><p class="s6-aviso" role="status" data-estado-fuente="${estadoReal}">${e(t(`estado_${estadoReal}`))}${estadoReal === "no_configurado" ? ` ${e(t("fuente_pendiente"))}` : ""}</p>${preparacion(t, canalConfigurado)}<div class="s6-trabajo">${bandeja}${detalle(estadoReal === "disponible" ? seleccionado : null, t)}</div></section>`;
}

/** Firma F1: sin red ni persistencia; acepta una nueva proyección y libera eventos al desmontar. */
export function montarVistaSeleccionComunicaciones({ raiz, modelo = {}, anunciar = () => {}, registrarDesmontar } = {}) {
  if (!raiz?.replaceChildren || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) throw new TypeError("vista S6 no disponible");
  let activa = true;
  let proyeccion = modelo;
  let vista = { filtro: "", canal: "todas", seleccion: "" };
  const t = crearTraductorSeleccionComunicaciones();
  const pintar = () => { if (activa) raiz.innerHTML = renderizarVistaSeleccionComunicaciones(proyeccion, vista); };
  const alEnviar = (evento) => {
    const formulario = evento.target?.closest?.("[data-s6-filtros]");
    if (!formulario) return;
    evento.preventDefault();
    vista = { ...vista, filtro: formulario.elements.filtro.value, canal: formulario.elements.canal.value };
    pintar();
    raiz.querySelector(".s6-tabla-wrap")?.focus();
    const visibles = filtrar(comunicacionesDisponibles(proyeccion), vista.filtro, vista.canal);
    anunciar(conteoVisibles(visibles.length, t), "informacion");
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
