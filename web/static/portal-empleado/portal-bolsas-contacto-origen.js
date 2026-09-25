/**
 * Contacto de origen CONVOCA (duda 45). La ficha de RRHH muestra si el
 * contacto vigente es propio o de origen CONVOCA y hasta cuándo vale; el
 * llamamiento por correo muestra los avisos de contactos vencidos sin
 * confirmar. Solo se lee la forma enmascarada: el claro nunca se pide aquí.
 */
import { LOCALIZACION_PORTAL } from "./portal-i18n.js?v=20260926-portal-rrhh-main-v1";
import { MENSAJES_CONTACTO_ORIGEN_ES } from "./portal-i18n-contacto-origen.js?v=20260926-contacto-origen-v1";

const RUTA_BOLSAS = "/api/vec/bolsa/bolsas";
const FECHA = /^\d{4}-\d{2}-\d{2}$/;
const ESTADOS = Object.freeze(["vigente", "vencido"]);

export function traducirContactoOrigen(clave, variables = {}) {
  const plantilla = MENSAJES_CONTACTO_ORIGEN_ES[clave] ?? clave;
  return plantilla.replace(/\{(\w+)\}/g, (_, nombre) => String(variables[nombre] ?? ""));
}

function html(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}

function segmento(valor) {
  return encodeURIComponent(String(valor ?? "").trim()).replace(/%3A/gi, ":");
}

/** Día civil «AAAA-MM-DD» en el formato local, sin desplazarlo de zona. */
export function fechaCivilVisible(dia) {
  if (!FECHA.test(dia || "")) return "";
  const [anio, mes, diaMes] = dia.split("-").map(Number);
  return new Intl.DateTimeFormat(LOCALIZACION_PORTAL, { timeZone: "UTC", day: "2-digit", month: "2-digit", year: "numeric" })
    .format(new Date(Date.UTC(anio, mes - 1, diaMes)));
}

export function origenValido(origen) {
  return origen === null || (typeof origen === "object" && origen.origen === "convoca"
    && ESTADOS.includes(origen.estado) && FECHA.test(origen.ultimo_dia || ""));
}

export async function consultarOrigenContacto(bolsa, participacion, { fetchImpl = fetch, signal } = {}) {
  try {
    const respuesta = await fetchImpl(`${RUTA_BOLSAS}/${segmento(bolsa)}/candidatos/${segmento(participacion)}/datos-contacto`, {
      method: "GET", credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", signal, headers: { Accept: "application/json" },
    });
    if (respuesta.status === 404) return { ok: true, datos: { sin_contacto: true, origen: null } };
    if (respuesta.status === 403) return { ok: false, status: 403, codigo: "acceso_denegado" };
    if (!respuesta.ok) return { ok: false, status: respuesta.status, codigo: "error_servidor" };
    const cuerpo = await respuesta.json();
    const datos = cuerpo?.data;
    const origen = datos?.origen ?? null;
    if (!datos || typeof datos !== "object" || !origenValido(origen)) return { ok: false, status: 0, codigo: "respuesta_invalida" };
    return { ok: true, datos: { sin_contacto: false, origen } };
  } catch (_error) {
    return { ok: false, status: 0, codigo: "error_red" };
  }
}

/** Fila «Contacto» de la ficha de participación. */
export function renderizarOrigenContacto({ estado = {}, escaparHTML = html } = {}) {
  const t = traducirContactoOrigen;
  let valor;
  if (!estado.carga || estado.carga === "cargando") valor = `<span role="status" aria-busy="true">${escaparHTML(t("cargando"))}</span>`;
  else if (estado.carga === "error") valor = `<span role="alert">${escaparHTML(t(estado.status === 403 ? "denegado" : "error"))}</span>`;
  else if (estado.datos?.sin_contacto) valor = escaparHTML(t("sin_contacto"));
  else if (!estado.datos?.origen) valor = `<span class="estado-chip exito">${escaparHTML(t("propio"))}</span>`;
  else {
    const o = estado.datos.origen;
    const fecha = fechaCivilVisible(o.ultimo_dia);
    valor = o.estado === "vencido"
      ? `<span class="estado-chip aviso">${escaparHTML(t("no_confirmado"))}</span><br><small>${escaparHTML(t("vencido_el", { fecha }))}</small>`
      : `<span class="estado-chip info">${escaparHTML(t("origen_convoca"))}</span><br><small>${escaparHTML(t("vigente_hasta", { fecha }))}</small>`;
  }
  return `<div class="fila-resumen" data-contacto-origen="true"><dt>${escaparHTML(t("etiqueta"))}</dt><dd>${valor}</dd></div>`;
}

/** Avisos a RRHH tras emitir: no bloquean, se muestran junto al recibo. */
export function renderizarAvisosContactoEmision({ avisos = [], candidatos = [], escaparHTML = html } = {}) {
  if (!Array.isArray(avisos) || avisos.length === 0) return "";
  const nombres = new Map(candidatos.map((c) => [c.participacion_ref, c.nombre_visible]));
  const t = traducirContactoOrigen;
  const elementos = avisos.map((a) => {
    const nombre = nombres.get(a.participacion_ref) || a.participacion_ref;
    const texto = a.aviso === "contacto_origen_convoca_no_confirmado"
      ? t("aviso_no_confirmado", { nombre, fecha: fechaCivilVisible(a.ultimo_dia) })
      : t("aviso_no_disponible", { nombre });
    return `<li>${escaparHTML(texto)}</li>`;
  }).join("");
  return `<section class="nota-pendiente" role="status" data-b7-avisos-contacto><strong>${escaparHTML(t("avisos_titulo"))}</strong><ul>${elementos}</ul></section>`;
}

/** Carga el estado del contacto al abrir la ficha, como el historial B8. */
export function crearControladorOrigenContacto({ estado, renderizar, consultar = consultarOrigenContacto }) {
  async function cargar(modalFicha) {
    if (!modalFicha?.candidato) return;
    modalFicha.controladorContactoOrigen?.abort?.();
    const controlador = new AbortController();
    modalFicha.controladorContactoOrigen = controlador;
    modalFicha.contactoOrigen = { carga: "cargando" };
    const res = await consultar(estado.bolsaSeleccionada, modalFicha.candidato.participacion_ref, { signal: controlador.signal });
    if (controlador.signal.aborted || estado.modalFicha !== modalFicha) return;
    modalFicha.contactoOrigen = res.ok ? { carga: "listo", datos: res.datos } : { carga: "error", status: res.status };
    renderizar();
  }
  return Object.freeze({ cargar });
}
