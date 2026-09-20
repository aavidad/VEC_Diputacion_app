import { traducir } from "./i18n.js";
import { escaparHTML, listaDatos, panel } from "./vistas/comunes.js";

const RUTA_MI_BOLSA = "/api/vec/bolsa/mi-bolsa";
const ESQUEMA_MI_BOLSA = "vec.bolsa.mi-bolsa.v1";
const MAXIMO_JSON_BYTES = 128 * 1024;
const MAXIMO_PARTICIPACIONES = 200;

export class ErrorMiBolsa extends Error {
  constructor(codigo, mensaje, causa) {
    super(mensaje, causa ? { cause: causa } : undefined);
    this.name = "ErrorMiBolsa";
    this.codigo = codigo;
  }
}

function texto(clave, variables) { return traducir(`areaPersonal.miBolsa.${clave}`, variables); }

function exigirObjeto(valor, nombre) {
  if (!valor || typeof valor !== "object" || Array.isArray(valor)) throw new TypeError(texto("errorObjeto", { nombre }));
  return valor;
}

function exigirCadena(valor, nombre, maximo = 300) {
  if (typeof valor !== "string" || valor.trim() === "" || valor.length > maximo) throw new TypeError(texto("errorCadena", { nombre, maximo }));
  return valor;
}

function exigirEntero(valor, nombre, minimo) {
  if (!Number.isSafeInteger(valor) || valor < minimo) throw new TypeError(texto("errorEntero", { nombre, minimo }));
  return valor;
}

function exigirFecha(valor, nombre, { nulo = false } = {}) {
  if (nulo && valor === null) return null;
  const fecha = exigirCadena(valor, nombre, 40);
  const partes = /^(\d{4})-(\d{2})-(\d{2})(?:T(\d{2}):(\d{2}):(\d{2})(?:\.(\d{1,6}))?Z)?$/u.exec(fecha);
  if (!partes) throw new TypeError(texto("errorFecha", { nombre }));
  const [, anoTexto, mesTexto, diaTexto, horaTexto = "00", minutoTexto = "00", segundoTexto = "00", fraccionTexto = ""] = partes;
  const milisegundoTexto = fraccionTexto.padEnd(3, "0").slice(0, 3);
  const [ano, mes, dia, hora, minuto, segundo, milisegundo] = [anoTexto, mesTexto, diaTexto, horaTexto, minutoTexto, segundoTexto, milisegundoTexto].map(Number);
  const calendario = new Date(Date.UTC(ano, mes - 1, dia, hora, minuto, segundo, milisegundo));
  if (calendario.getUTCFullYear() !== ano || calendario.getUTCMonth() !== mes - 1 || calendario.getUTCDate() !== dia || calendario.getUTCHours() !== hora || calendario.getUTCMinutes() !== minuto || calendario.getUTCSeconds() !== segundo || calendario.getUTCMilliseconds() !== milisegundo) throw new TypeError(texto("errorFecha", { nombre }));
  return fecha;
}

function congelarProfundo(valor) {
  if (valor && typeof valor === "object" && !Object.isFrozen(valor)) {
    Object.values(valor).forEach(congelarProfundo);
    Object.freeze(valor);
  }
  return valor;
}

export function validarMiBolsa(entrada) {
  const datos = structuredClone(exigirObjeto(entrada, "datos"));
  if (exigirCadena(datos.esquema, "esquema", 80) !== ESQUEMA_MI_BOLSA) throw new TypeError(texto("errorEsquema"));
  exigirFecha(datos.consultada_en, "consultada_en");
  if (!Array.isArray(datos.participaciones) || datos.participaciones.length > MAXIMO_PARTICIPACIONES) throw new TypeError(texto("errorParticipaciones", { maximo: MAXIMO_PARTICIPACIONES }));
  datos.participaciones.forEach((item, indice) => {
    const prefijo = `participaciones[${indice}]`;
    exigirObjeto(item, prefijo);
    exigirCadena(item.bolsa_ref, `${prefijo}.bolsa_ref`);
    exigirCadena(item.categoria_ref, `${prefijo}.categoria_ref`);
    exigirEntero(item.version_bolsa, `${prefijo}.version_bolsa`, 1);
    exigirEntero(item.orden, `${prefijo}.orden`, 1);
    exigirEntero(item.total_participaciones, `${prefijo}.total_participaciones`, 1);
    if (item.total_participaciones < item.orden) throw new TypeError(texto("errorTotal", { indice }));
    exigirCadena(item.estado_bolsa, `${prefijo}.estado_bolsa`);
    exigirFecha(item.vigente_desde, `${prefijo}.vigente_desde`);
    exigirFecha(item.vigente_hasta ?? null, `${prefijo}.vigente_hasta`, { nulo: true });
  });
  return congelarProfundo(datos);
}

async function leerRespuesta(respuesta) {
  const estado = respuesta?.status;
  if (estado === 401) throw new ErrorMiBolsa("autenticacion_requerida", texto("autenticacionRequerida"));
  if (estado === 403) throw new ErrorMiBolsa("acceso_denegado", texto("accesoDenegado"));
  if (estado === 503) throw new ErrorMiBolsa("servicio_no_disponible", texto("servicioNoDisponible"));
  if (estado !== 200) throw new ErrorMiBolsa("respuesta_http", texto("respuestaHTTP", { estado: estado || "sin respuesta" }));
  const tipo = respuesta.headers?.get?.("Content-Type") || "";
  if (!/^application\/json(?:\s*;\s*charset=utf-8)?$/iu.test(tipo)) throw new ErrorMiBolsa("tipo_respuesta", texto("tipoRespuesta"));
  const declarada = respuesta.headers?.get?.("Content-Length");
  if (declarada && (!/^(?:0|[1-9][0-9]*)$/u.test(declarada) || Number(declarada) > MAXIMO_JSON_BYTES)) throw new ErrorMiBolsa("respuesta_excesiva", texto("respuestaExcesiva"));
  const cuerpo = await respuesta.text();
  if (new TextEncoder().encode(cuerpo).byteLength > MAXIMO_JSON_BYTES) throw new ErrorMiBolsa("respuesta_excesiva", texto("respuestaExcesiva"));
  let envelope;
  try { envelope = JSON.parse(cuerpo); } catch (causa) { throw new ErrorMiBolsa("json_invalido", texto("jsonInvalido"), causa); }
  if (!envelope || typeof envelope !== "object" || Array.isArray(envelope) || !envelope.data || typeof envelope.data !== "object" || Array.isArray(envelope.data)) throw new ErrorMiBolsa("respuesta_incompatible", texto("respuestaIncompatible"));
  return validarMiBolsa(envelope.data);
}

export function crearClienteMiBolsa({ fetchImpl = globalThis.fetch } = {}) {
  if (typeof fetchImpl !== "function") return Object.freeze({ cargar: async () => { throw new ErrorMiBolsa("transporte_no_disponible", texto("transporteNoDisponible")); } });
  return Object.freeze({
    async cargar({ signal } = {}) {
      let respuesta;
      try {
        respuesta = await fetchImpl(RUTA_MI_BOLSA, { method: "GET", headers: { Accept: "application/json" }, credentials: "omit", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer", signal });
      } catch (causa) {
        if (causa?.name === "AbortError") throw causa;
        throw new ErrorMiBolsa("servicio_no_disponible", texto("servicioNoDisponible"), causa);
      }
      return leerRespuesta(respuesta);
    },
  });
}

function formatoFecha(valor) { return new Intl.DateTimeFormat("es-ES", { dateStyle: "long", timeZone: "UTC" }).format(new Date(valor)); }

function prepararShellMiBolsa(documento, ventana) {
  documento.querySelectorAll(".ap-navegacion > *").forEach((elemento) => { elemento.hidden = true; });
  documento.querySelectorAll("[data-mi-bolsa-real]").forEach((elemento) => { elemento.hidden = false; });
  documento.querySelector('[data-ruta="mi-bolsa"]')?.setAttribute("aria-current", "page");
  documento.querySelector(".busqueda-global")?.setAttribute("hidden", "");
  documento.querySelector(".acciones-cabecera")?.setAttribute("hidden", "");
  const lateral = documento.getElementById("navegacion-lateral");
  const botonMenu = documento.querySelector('[data-accion="alternar-menu"]');
  const velo = documento.querySelector('[data-accion="cerrar-menu"]');
  const cerrar = () => {
    documento.body.dataset.menuAbierto = "false";
    botonMenu?.setAttribute("aria-expanded", "false");
    if (velo) velo.hidden = true;
  };
  botonMenu?.addEventListener("click", () => {
    const abierto = documento.body.dataset.menuAbierto !== "true";
    documento.body.dataset.menuAbierto = String(abierto);
    botonMenu.setAttribute("aria-expanded", String(abierto));
    if (velo) velo.hidden = !abierto;
    if (abierto) lateral?.querySelector('[data-ruta="mi-bolsa"]')?.focus({ preventScroll: true });
  });
  velo?.addEventListener("click", cerrar);
  if (new URLSearchParams(ventana.location.search).get("vista") !== "mi-bolsa") {
    const url = new URL(ventana.location.href);
    url.search = "";
    url.searchParams.set("vista", "mi-bolsa");
    ventana.history.replaceState({ vista: "mi-bolsa" }, "", `${url.pathname}${url.search}`);
  }
}

export function renderizarMiBolsa(datos) {
  if (!datos.participaciones.length) return panel(texto("titulo"), texto("consultaDisponible"), `<p>${texto("vacio")}</p>`);
  const tarjetas = datos.participaciones.map((item) => panel(texto("bolsaReferencia", { referencia: item.bolsa_ref }), texto("categoriaReferencia", { referencia: item.categoria_ref }), `${listaDatos([
    [texto("ordenConstitucion"), `${escaparHTML(String(item.orden))} ${texto("de")} ${escaparHTML(String(item.total_participaciones))}`],
    [texto("estadoBolsa"), escaparHTML(item.estado_bolsa)],
    [texto("vigenteDesde"), escaparHTML(formatoFecha(item.vigente_desde))],
    [texto("vigenteHasta"), item.vigente_hasta ? escaparHTML(formatoFecha(item.vigente_hasta)) : texto("sinFechaFin")],
  ])}<p class="nota">${escaparHTML(texto("ordenHistorico", { version: item.version_bolsa }))}</p>`, { estado: item.estado_bolsa })).join("");
  return `<header class="encabezado-vista"><div><h2>${texto("titulo")}</h2><p>${texto("descripcion")}</p></div></header>${tarjetas}`;
}

export async function iniciarMiBolsa({ cliente = crearClienteMiBolsa(), documento = document, ventana = window } = {}) {
  const carga = documento.getElementById("estado-carga");
  const espacio = documento.getElementById("espacio-trabajo");
  const titulo = documento.getElementById("titulo-vista");
  const migas = documento.getElementById("migas-pan");
  if (!carga || !espacio || !titulo || !migas) throw new Error(texto("montajeIncompleto"));
  prepararShellMiBolsa(documento, ventana);
  titulo.textContent = texto("titulo");
  migas.textContent = texto("migas");
  const controlador = new AbortController();
  const cargar = async () => {
    carga.hidden = false;
    carga.className = "estado-carga";
    carga.textContent = texto("carga");
    espacio.replaceChildren();
    try {
      const datos = await cliente.cargar({ signal: controlador.signal });
      carga.hidden = true;
      espacio.innerHTML = renderizarMiBolsa(datos);
    } catch (error) {
      if (error?.name === "AbortError") return;
      carga.hidden = true;
      const mensaje = error instanceof Error ? error.message : texto("errorConsulta");
      espacio.innerHTML = `<section class="estado-error" role="alert"><h2>${texto("errorConsulta")}</h2><p>${escaparHTML(mensaje)}</p><button type="button" class="boton-primario" data-reintentar-mi-bolsa>${texto("reintentar")}</button></section>`;
      espacio.querySelector("[data-reintentar-mi-bolsa]")?.addEventListener("click", cargar, { once: true });
    }
  };
  ventana.addEventListener("pagehide", () => controlador.abort(), { once: true });
  await cargar();
}

export const CONTRATO_MI_BOLSA = Object.freeze({ ruta: RUTA_MI_BOLSA, esquema: ESQUEMA_MI_BOLSA });
