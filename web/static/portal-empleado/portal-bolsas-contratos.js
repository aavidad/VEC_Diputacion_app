import { cargarTextos } from "../comun/textos.js";
import { LOCALIZACION_PORTAL, ZONA_HORARIA_PORTAL } from "./portal-i18n.js?v=20261007-pantallas-textos-final-v1";
/**
 * B13 · Histórico de contratos de la participación (Petición RRHH p. 2).
 * Solo lectura: los contratos proceden de Contratación temporal por evento
 * y Bolsa los guarda en su propio histórico. Textos por clave i18n.
 */
const BASE = "/api/vec/bolsa/bolsas";
export const ESQUEMA_CONTRATOS = "vec.bolsa.rrhh.contratos_participacion.v1";
const INSTANTE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z$/;
const CLAVE = /^[a-z][a-z0-9._-]{1,79}$/;
const POR_PAGINA = 6;

/** Catálogo propio, leído por la autoridad común de idioma y textos. */
export async function cargarMensajesContratos(idioma) {
  return (await cargarTextos("bolsa", { idioma })).seccion("contratos_participacion");
}
export let MENSAJES_CONTRATOS;
let preparacionContratos = null;
/** La ficha de Bolsa prepara su catálogo al abrir, conservando la instancia y el idioma activo. */
export function prepararMensajesContratos(idioma) {
  if (MENSAJES_CONTRATOS) return Promise.resolve(MENSAJES_CONTRATOS);
  if (preparacionContratos) return preparacionContratos;
  preparacionContratos = cargarMensajesContratos(idioma).then((catalogo) => {
    if (!catalogo || typeof catalogo !== "object" || Object.keys(catalogo).length === 0
      || Object.values(catalogo).some((valor) => typeof valor !== "string" || valor.trim() === "")) {
      throw new Error("catálogo de contratos de Bolsa incompleto");
    }
    MENSAJES_CONTRATOS = Object.freeze({ ...catalogo });
    return MENSAJES_CONTRATOS;
  }).catch((error) => { preparacionContratos = null; throw error; });
  return preparacionContratos;
}

/** Traductor estricto: una clave inexistente es un error de programación. */
export function traducirContratos(clave, variables = {}, catalogo = MENSAJES_CONTRATOS) {
  if (!catalogo || typeof catalogo !== "object") throw new Error("textos de contratos pendientes de preparación");
  const plantilla = catalogo[clave];
  if (!Object.hasOwn(catalogo, clave) || typeof plantilla !== "string") throw new Error(`Clave i18n de contratos inexistente: ${clave}`);
  return plantilla.replace(/\{(\w+)\}/g, (_, nombre) => String(variables[nombre] ?? ""));
}

function segmento(valor) {
  return encodeURIComponent(String(valor ?? "").trim()).replace(/%3A/gi, ":");
}

export function rutaContratosParticipacion(bolsa, participacion) {
  return `${BASE}/${segmento(bolsa)}/candidatos/${segmento(participacion)}/contratos`;
}

function instanteOpcional(valor) {
  return valor === null || (typeof valor === "string" && INSTANTE.test(valor));
}

function itemValido(item) {
  return item && typeof item === "object"
    && typeof item.evento_ref === "string" && item.evento_ref !== ""
    && typeof item.tipo === "string" && /^[a-z][a-z0-9_]{1,39}$/.test(item.tipo)
    && instanteOpcional(item.inicio) && instanteOpcional(item.fin_previsto)
    && typeof item.ocurrido_en === "string" && INSTANTE.test(item.ocurrido_en)
    && ["modalidad_clave", "causa_clave"].every((campo) => item[campo] === "" || (typeof item[campo] === "string" && CLAVE.test(item[campo])))
    && typeof item.categoria_ref === "string" && item.categoria_ref.length <= 512;
}

function errorHttp(status, catalogo) {
  const clave = { 403: "error_403", 404: "error_404", 503: "error_503" }[status];
  return { ok: false, status, mensaje: traducirContratos(clave || "error_http", {}, catalogo) };
}

export async function consultarContratosParticipacion(bolsa, participacion, { fetchImpl = fetch, signal, catalogo = MENSAJES_CONTRATOS } = {}) {
  try {
    const respuesta = await fetchImpl(rutaContratosParticipacion(bolsa, participacion), {
      method: "GET", credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer", signal, headers: { Accept: "application/json" },
    });
    if (!respuesta.ok) return errorHttp(respuesta.status, catalogo);
    const cuerpo = await respuesta.json();
    const items = cuerpo?.data?.items;
    if (cuerpo?.data?.esquema !== ESQUEMA_CONTRATOS || !Array.isArray(items) || !items.every(itemValido)) {
      return { ok: false, status: 0, mensaje: traducirContratos("error_contrato", {}, catalogo) };
    }
    return { ok: true, datos: items };
  } catch (error) {
    if (error?.name === "AbortError") return { ok: false, status: 0, abortada: true, mensaje: "" };
    return { ok: false, status: 0, mensaje: traducirContratos("error_red", {}, catalogo) };
  }
}

/**
 * Carga el histórico de la ficha abierta. Solo escribe si la ficha sigue
 * siendo la misma; una ficha nueva cancela la lectura anterior. Quien ya
 * va a pintar la ficha a continuación pasa renderizarAlIniciar=false.
 */
export async function cargarContratosFicha(modalFicha, { estado, renderizar, consultar = consultarContratosParticipacion, renderizarAlIniciar = true }) {
  if (!modalFicha?.candidato?.participacion_ref) return;
  modalFicha.controladorContratos?.abort();
  const controlador = new AbortController();
  modalFicha.controladorContratos = controlador;
  modalFicha.contratosB13 = { carga: "cargando", items: [], pagina: 0 };
  if (renderizarAlIniciar) renderizar();
  const res = await consultar(estado.bolsaSeleccionada, modalFicha.candidato.participacion_ref, { signal: controlador.signal });
  if (controlador.signal.aborted || estado.modalFicha !== modalFicha) return;
  modalFicha.contratosB13 = res.ok
    ? { carga: "listo", items: res.datos, pagina: 0 }
    : { carga: "error", error: res.mensaje, items: [], pagina: 0 };
  renderizar();
}

function fecha(valor, textos) {
  return valor ? textos.formatoFecha.format(new Date(valor)) : textos.t("sin_dato");
}

/** Las claves desconocidas siguen visibles como datos del catálogo de origen. */
function rotuloClave(clave, prefijo, textos) {
  if (!clave) return textos.t("sin_dato");
  const mensaje = `${prefijo}_${clave}`;
  if (Object.hasOwn(textos.catalogo, mensaje)) return textos.t(mensaje);
  const texto = clave.replace(/[._-]+/g, " ");
  return texto.charAt(0).toUpperCase() + texto.slice(1);
}

function periodo(item, textos) {
  if (!item.inicio) return textos.t("sin_dato");
  return item.fin_previsto
    ? textos.t("hasta_previsto", { inicio: fecha(item.inicio, textos), fin: fecha(item.fin_previsto, textos) })
    : textos.t("sin_fin", { inicio: fecha(item.inicio, textos) });
}

// La categoría del contrato llega como referencia opaca de Contratación: se
// nombra con la categoría de la bolsa en la que se hizo el llamamiento.
export function renderizarContratosParticipacion({ estado = {}, escaparHTML, categoria = "", catalogo = MENSAJES_CONTRATOS, localizacion = LOCALIZACION_PORTAL }) {
  const textos = {
    catalogo,
    t: (clave, variables) => traducirContratos(clave, variables, catalogo),
    formatoFecha: new Intl.DateTimeFormat(localizacion, { timeZone: ZONA_HORARIA_PORTAL, day: "2-digit", month: "2-digit", year: "numeric" }),
    formatoNumero: new Intl.NumberFormat(localizacion),
  };
  const t = (clave, variables) => escaparHTML(textos.t(clave, variables));
  const carga = estado.carga || "cargando";
  let contenido;
  if (carga === "cargando") contenido = `<p class="vacio-controlado" role="status" aria-busy="true">${t("cargando")}</p>`;
  else if (carga === "error") contenido = `<p class="mensaje-error" role="alert">${escaparHTML(estado.error || textos.t("error_red"))}</p><button type="button" class="boton-secundario" data-b13-accion="reintentar">${t("reintentar")}</button>`;
  else if (!estado.items?.length) contenido = `<p class="vacio-controlado" role="status">${t("vacio")}</p>`;
  else {
    const total = estado.items.length;
    const paginas = Math.max(1, Math.ceil(total / POR_PAGINA));
    const pagina = Math.min(Math.max(0, Number(estado.pagina) || 0), paginas - 1);
    const visibles = estado.items.slice(pagina * POR_PAGINA, (pagina + 1) * POR_PAGINA);
    const filas = visibles.map((item) => `<tr><td>${escaparHTML(rotuloClave(item.tipo, "tipo", textos))}</td><td>${escaparHTML(periodo(item, textos))}</td><td>${escaparHTML(rotuloClave(item.modalidad_clave, "modalidad", textos))}</td><td>${item.categoria_ref && categoria ? escaparHTML(categoria) : t("sin_dato")}</td><td>${escaparHTML(rotuloClave(item.causa_clave, "causa", textos))}</td><td>${escaparHTML(fecha(item.ocurrido_en, textos))}</td></tr>`).join("");
    const resumen = t("mostrando", {
      desde: textos.formatoNumero.format(pagina * POR_PAGINA + 1),
      hasta: textos.formatoNumero.format(Math.min((pagina + 1) * POR_PAGINA, total)),
      total: textos.formatoNumero.format(total),
    });
    const navegacion = paginas > 1
      ? `<nav class="paginacion-bolsa" aria-label="${t("paginacion")}"><span>${resumen}</span><button type="button" class="boton-secundario" data-b13-accion="pagina" data-pagina="${pagina - 1}" ${pagina === 0 ? "disabled" : ""}>${t("anterior")}</button><button type="button" class="boton-secundario" data-b13-accion="pagina" data-pagina="${pagina + 1}" ${pagina + 1 >= paginas ? "disabled" : ""}>${t("siguiente")}</button></nav>`
      : `<p>${resumen}</p>`;
    contenido = `<div class="tabla-contenedor" tabindex="0" role="region" aria-label="${t("tabla")}"><table class="tabla-datos"><caption>${t("tabla")}</caption><thead><tr><th scope="col">${t("col_tipo")}</th><th scope="col">${t("col_periodo")}</th><th scope="col">${t("col_modalidad")}</th><th scope="col">${t("col_categoria")}</th><th scope="col">${t("col_causa")}</th><th scope="col">${t("col_registrado")}</th></tr></thead><tbody>${filas}</tbody></table></div>${navegacion}`;
  }
  return `<section class="panel panel-separado" data-b13-raiz="true" aria-labelledby="b13-titulo"><div class="cabecera-panel"><div><h4 id="b13-titulo">${t("titulo")}</h4><p>${t("descripcion")}</p></div></div><div class="cuerpo-panel">${contenido}</div></section>`;
}

/** Atiende reintento y paginación de la sección B13 de la ficha abierta. */
export function manejarClickContratos(evento, { estado, renderizar, consultar } = {}) {
  const control = evento.target?.closest?.("[data-b13-accion]");
  if (!control || !estado?.modalFicha) return false;
  evento.preventDefault();
  const modal = estado.modalFicha;
  if (control.dataset.b13Accion === "reintentar") {
    void cargarContratosFicha(modal, { estado, renderizar, ...(consultar ? { consultar } : {}) });
    return true;
  }
  if (control.dataset.b13Accion === "pagina" && modal.contratosB13) {
    modal.contratosB13 = { ...modal.contratosB13, pagina: Math.max(0, Number(control.dataset.pagina) || 0) };
    renderizar();
  }
  return true;
}
