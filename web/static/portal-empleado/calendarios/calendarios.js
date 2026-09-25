import {
  crearTraductorCalendarios, formatearFechaCivil, formatearInstante, formatearNumero, nombreMes,
} from "./i18n.js?v=20260925-calendarios-v1";
import { icono } from "../../comun/iconos-vec.js?v=20260925-aspecto-v1";
import { instanteDesdeHoraMadrid } from "../hora-madrid.js";

export const API_CENTROS = "/api/vec/calendarios/centros";
export const API_CALENDARIO = "/api/vec/calendarios/centro";
export const API_PLAZO = "/api/vec/calendarios/plazo";
export const LIMITE_RESPUESTA = 1024 * 1024;
export const UNIDADES = Object.freeze(["dias_habiles", "dias_naturales", "meses", "anios"]);

const t = crearTraductorCalendarios();
const FECHA = /^\d{4}-\d{2}-\d{2}$/u;
const REFERENCIA = /^[a-z][a-z0-9:_-]{1,95}$/u;
const AMBITOS = new Set(["nacional", "autonomico", "local", "centro"]);
const EFECTOS = new Set(["festivo", "inhabil_administrativo", "no_laborable"]);

const esc = (v) => String(v ?? "")
  .replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;")
  .replaceAll('"', "&quot;").replaceAll("'", "&#39;");

export class ErrorCalendarios extends Error {
  constructor(codigo, detalle = null) {
    super(codigo);
    this.codigo = codigo;
    this.detalle = detalle;
  }
}

async function leerJSONLimitado(respuesta) {
  const texto = await respuesta.text();
  if (texto.length > LIMITE_RESPUESTA) throw new ErrorCalendarios("error_respuesta");
  try {
    return JSON.parse(texto);
  } catch {
    throw new ErrorCalendarios("error_respuesta");
  }
}

const CODIGOS_API = new Set(["solicitud_invalida", "calendario_no_publicado", "servicio_no_disponible", "autenticacion_requerida", "acceso_denegado"]);

/** Cliente de solo lectura: sin cookies propias ni cabeceras de identidad. */
export function crearCliente(fetchImpl = globalThis.fetch, timeoutMs = 10000) {
  const pedir = async (ruta, parametros) => {
    const controlador = new AbortController();
    const temporizador = setTimeout(() => controlador.abort(), timeoutMs);
    try {
      const consulta = new URLSearchParams(Object.entries(parametros).filter(([, v]) => v !== "" && v !== undefined && v !== null));
      const respuesta = await fetchImpl(`${ruta}?${consulta}`, {
        method: "GET", credentials: "same-origin", redirect: "error", cache: "no-store",
        headers: { Accept: "application/json" }, signal: controlador.signal,
      });
      if (respuesta.ok) return await leerJSONLimitado(respuesta);
      let codigo = respuesta.status === 401 ? "autenticacion_requerida" : respuesta.status === 403 ? "acceso_denegado" : "servicio_no_disponible";
      let detalle = null;
      try {
        const cuerpo = await leerJSONLimitado(respuesta);
        if (CODIGOS_API.has(cuerpo?.error?.codigo)) codigo = cuerpo.error.codigo;
        detalle = cuerpo?.error?.detalle ?? null;
      } catch { /* se conserva el código por estado */ }
      throw new ErrorCalendarios(`error_${codigo}`, detalle);
    } catch (error) {
      if (error instanceof ErrorCalendarios) throw error;
      throw new ErrorCalendarios("error_servicio_no_disponible");
    } finally {
      clearTimeout(temporizador);
    }
  };
  return Object.freeze({
    centros: async (anio, conocidoEn) => validarCentros(await pedir(API_CENTROS, { anio, conocido_en: conocidoEn })),
    calendario: async (centro, anio, conocidoEn) => validarCalendario(await pedir(API_CALENDARIO, { centro, anio, conocido_en: conocidoEn })),
    plazo: async (parametros) => validarPlazo(await pedir(API_PLAZO, parametros)),
  });
}

function exigir(condicion) {
  if (!condicion) throw new ErrorCalendarios("error_respuesta");
}
const texto = (v, max = 400) => typeof v === "string" && v.length > 0 && v.length <= max;

export function validarCentros(payload) {
  const d = payload?.data;
  exigir(d && Number.isInteger(d.anio) && Array.isArray(d.centros) && d.centros.length <= 1000);
  d.centros.forEach((c) => exigir(REFERENCIA.test(c?.centro_ref) && texto(c?.denominacion, 240) && REFERENCIA.test(c?.municipio_ref) && Number.isInteger(c?.numero)));
  return d;
}

function validarVersion(v) {
  exigir(texto(v?.id, 96) && AMBITOS.has(v?.ambito?.tipo) && texto(v?.ambito?.ref, 96) && Number.isInteger(v?.numero)
    && texto(v?.denominacion, 240) && texto(v?.procedencia?.norma) && texto(v?.procedencia?.referencia)
    && typeof v?.procedencia?.sintetica === "boolean" && !Number.isNaN(Date.parse(v?.conocido_desde)));
}

function validarMotivos(motivos) {
  exigir(motivos === null || motivos === undefined || Array.isArray(motivos));
  (motivos ?? []).forEach((m) => exigir(AMBITOS.has(m?.ambito?.tipo) && EFECTOS.has(m?.efecto) && texto(m?.denominacion, 240) && typeof m?.sintetico === "boolean"));
  return motivos ?? [];
}

export function validarCalendario(payload) {
  const d = payload?.data;
  exigir(d && REFERENCIA.test(d.centro_ref) && Number.isInteger(d.anio) && Array.isArray(d.dias) && d.dias.length >= 365 && d.dias.length <= 366
    && Array.isArray(d.versiones) && d.versiones.length >= 1 && d.versiones.length <= 16 && d.resumen && typeof d.resumen === "object");
  d.versiones.forEach(validarVersion);
  d.dias.forEach((dia) => {
    exigir(FECHA.test(dia?.fecha) && Number(dia.fecha.slice(0, 4)) === d.anio);
    for (const k of ["fin_de_semana", "festivo_oficial", "inhabil_administrativo", "laborable"]) exigir(typeof dia[k] === "boolean");
    dia.motivos = validarMotivos(dia.motivos);
  });
  for (const k of ["dias_naturales", "laborables", "habiles_administrativos", "festivos_oficiales", "no_laborables_centro"]) exigir(Number.isInteger(d.resumen[k]));
  return d;
}

export function validarPlazo(payload) {
  const d = payload?.data;
  exigir(d && FECHA.test(d.inicio) && FECHA.test(d.primer_dia) && FECHA.test(d.fin_nominal) && FECHA.test(d.vencimiento)
    && typeof d.prorrogado === "boolean" && !Number.isNaN(Date.parse(d.vence_antes_de)) && Array.isArray(d.versiones_utilizadas));
  exigir(d.dias_excluidos === null || Array.isArray(d.dias_excluidos));
  d.dias_excluidos = (d.dias_excluidos ?? []).map((x) => {
    exigir(FECHA.test(x?.fecha) && typeof x?.fin_de_semana === "boolean");
    return { ...x, motivos: validarMotivos(x.motivos) };
  });
  d.versiones_utilizadas.forEach(validarVersion);
  return d;
}

const PRIORIDAD = ["nacional", "autonomico", "local", "centro"];

/** Tipo visual del día: el motivo de mayor rango, o fin de semana. */
export function tipoDia(dia) {
  for (const ambito of PRIORIDAD) {
    if ((dia.motivos ?? []).some((m) => m.ambito.tipo === ambito)) return ambito;
  }
  return dia.fin_de_semana ? "finde" : "habil";
}

function etiquetaDia(dia) {
  const motivos = (dia.motivos ?? []).map((m) => `${m.denominacion} (${t(`ambito_${m.ambito.tipo}`)})`);
  if (dia.fin_de_semana) motivos.unshift(t("leyendaFinde"));
  motivos.push(t(dia.inhabil_administrativo ? "diaInhabil" : "diaHabil"));
  return t("diaEtiqueta", { fecha: formatearFechaCivil(dia.fecha), motivos: motivos.join(", ") });
}

export function renderizarMes(anio, mes, porFecha) {
  const primero = new Date(Date.UTC(anio, mes - 1, 1));
  const desplazamiento = (primero.getUTCDay() + 6) % 7;
  const total = new Date(Date.UTC(anio, mes, 0)).getUTCDate();
  const cabecera = t("semanaCorta").split(",").map((d) => `<th scope="col" abbr="${esc(d)}">${esc(d)}</th>`).join("");
  const celdas = [];
  for (let i = 0; i < desplazamiento; i += 1) celdas.push('<td class="cal-vacio"></td>');
  for (let d = 1; d <= total; d += 1) {
    const fecha = `${anio}-${String(mes).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
    const dia = porFecha.get(fecha);
    if (!dia) {
      celdas.push(`<td class="cal-dia"><span>${d}</span></td>`);
      continue;
    }
    const etiqueta = etiquetaDia(dia);
    celdas.push(`<td class="cal-dia cal-dia--${tipoDia(dia)}" title="${esc(etiqueta)}"><span aria-hidden="true">${d}</span><span class="cal-accesible">${esc(etiqueta)}</span></td>`);
  }
  while (celdas.length % 7 !== 0) celdas.push('<td class="cal-vacio"></td>');
  const filas = [];
  for (let i = 0; i < celdas.length; i += 7) filas.push(`<tr>${celdas.slice(i, i + 7).join("")}</tr>`);
  const nombre = nombreMes(mes);
  return `<table class="cal-mes"><caption>${esc(nombre.charAt(0).toUpperCase() + nombre.slice(1))}</caption><thead><tr>${cabecera}</tr></thead><tbody>${filas.join("")}</tbody></table>`;
}

export function renderizarCalendario(cal) {
  const porFecha = new Map(cal.dias.map((d) => [d.fecha, d]));
  return Array.from({ length: 12 }, (_, i) => renderizarMes(cal.anio, i + 1, porFecha)).join("");
}

const ICONOS_KPI = Object.freeze({ habiles: "calendario", laborables: "correcto", festivos: "alerta", cierres: "pendiente" });

export function renderizarResumen(cal) {
  const r = cal.resumen;
  const kpi = (clase, clave, valor) => `<div class="cal-kpi cal-kpi--${clase}"><span class="cal-kpi-icono">${icono(ICONOS_KPI[clase])}</span><strong>${esc(formatearNumero(valor))}</strong><span>${esc(t(clave))}</span></div>`;
  return kpi("habiles", "kpiHabiles", r.habiles_administrativos) + kpi("laborables", "kpiLaborables", r.laborables)
    + kpi("festivos", "kpiFestivos", r.festivos_oficiales) + kpi("cierres", "kpiCierres", r.no_laborables_centro);
}

function sinteticaDe(cal, versionID) {
  return cal.versiones.find((v) => v.id === versionID)?.procedencia?.sintetica === true;
}

export function renderizarFestivos(cal) {
  const filas = [];
  for (const dia of cal.dias) {
    for (const m of dia.motivos ?? []) {
      const sintetico = m.sintetico || sinteticaDe(cal, m.version_id);
      filas.push(`<tr class="cal-fila--${esc(m.ambito.tipo)}"><td><span class="cal-marca" aria-hidden="true"></span>${esc(formatearFechaCivil(dia.fecha, "medium"))}</td><td>${esc(m.denominacion)}</td><td>${esc(t(`ambito_${m.ambito.tipo}`))}</td><td><span class="cal-estado cal-estado--${sintetico ? "sintetico" : "oficial"}">${esc(t(sintetico ? "sintetico" : "oficial"))}</span></td></tr>`);
    }
  }
  return filas.join("");
}

export function renderizarVersiones(versiones) {
  return versiones.map((v) => `<tr><td>${esc(v.denominacion)}</td><td class="cal-numero">${esc(formatearNumero(v.numero))}</td><td>${esc(v.procedencia.norma)}<br><small>${esc(v.procedencia.referencia)}</small></td><td>${esc(formatearInstante(v.conocido_desde))}</td></tr>`).join("");
}

export function etiquetaMunicipio(ref) {
  const sintetico = /^municipio:sintetico:([a-z0-9-]+)$/u.exec(ref);
  if (sintetico) return t("municipio_sintetico", { codigo: sintetico[1].toUpperCase() });
  const ine = /^municipio:ine:(\d{5})$/u.exec(ref);
  if (ine) return t("municipio_ine", { codigo: ine[1] });
  return ref;
}

export function renderizarPlazo(p) {
  const lineas = [`<p class="cal-plazo-vence">${esc(t("plazoVence", { fecha: formatearFechaCivil(p.vencimiento) }))}</p>`];
  if (p.prorrogado) lineas.push(`<p>${esc(t("plazoProrrogado", { fecha: formatearFechaCivil(p.fin_nominal) }))}</p>`);
  lineas.push(`<p>${esc(t("plazoInstante", { fecha: formatearFechaCivil(p.vencimiento) }))}</p>`);
  lineas.push(`<p>${esc(t("plazoPrimerDia", { fecha: formatearFechaCivil(p.primer_dia) }))}</p>`);
  const excluidos = p.dias_excluidos.map((x) => {
    const motivos = x.motivos.map((m) => m.denominacion);
    if (x.fin_de_semana) motivos.unshift(t("plazoFinde"));
    return `<li><time datetime="${esc(x.fecha)}">${esc(formatearFechaCivil(x.fecha, "medium"))}</time> · ${esc(motivos.join(", "))}</li>`;
  });
  lineas.push(`<h3>${esc(t("plazoExcluidos"))}</h3>`, excluidos.length ? `<ul class="cal-excluidos">${excluidos.join("")}</ul>` : `<p>${esc(t("plazoSinExcluidos"))}</p>`);
  return lineas.join("");
}

export function mensajeError(error) {
  const clave = error instanceof ErrorCalendarios ? error.codigo : "error_servicio_no_disponible";
  if (clave === "error_calendario_no_publicado") {
    const faltan = (error.detalle?.faltan ?? []).map((a) => (a.tipo === "local" ? etiquetaMunicipio(a.ref) : t(`ambito_${AMBITOS.has(a.tipo) ? a.tipo : "nacional"}`)));
    return t(clave, { anio: error.detalle?.anio ?? "", faltan: faltan.join(", ") });
  }
  try {
    return t(clave);
  } catch {
    return t("error_servicio_no_disponible");
  }
}

/** Convierte un datetime-local del navegador en RFC 3339 UTC; vacío si no hay valor. */
/** «Conocido en» se escribe en hora de Madrid y viaja en UTC; vacío o hora inexistente → sin selector. */
export function instanteDesdeCampo(valor) {
  const ms = instanteDesdeHoraMadrid(valor);
  return ms === null ? "" : new Date(ms).toISOString();
}

function traducirDocumento(doc) {
  doc.title = t("documentTitle");
  doc.querySelectorAll("[data-i18n]").forEach((el) => { el.textContent = t(el.dataset.i18n); });
  doc.querySelectorAll("[data-i18n-label]").forEach((el) => { el.setAttribute("aria-label", t(el.dataset.i18nLabel)); });
}

export function iniciar(doc, cliente) {
  traducirDocumento(doc);
  const $ = (id) => doc.getElementById(id);
  const estado = { calendario: null, centros: [] };
  const anioActual = Number(new Intl.DateTimeFormat("en-CA", { year: "numeric", timeZone: "Europe/Madrid" }).format(new Date()));
  const selAnio = $("cal-anio");
  selAnio.innerHTML = [anioActual - 1, anioActual, anioActual + 1].map((a) => `<option value="${a}"${a === anioActual ? " selected" : ""}>${a}</option>`).join("");
  $("cal-unidad").innerHTML = UNIDADES.map((u) => `<option value="${u}">${esc(t(`unidad_${u}`))}</option>`).join("");
  const avisar = (id, mensaje, error = false) => {
    const el = $(id);
    el.textContent = mensaje;
    el.classList.toggle("cal-estado-error", error);
    el.hidden = mensaje === "";
  };
  const ayuda = $("cal-ayuda");
  $("cal-ayuda-abrir").addEventListener("click", () => ayuda.showModal?.());
  $("cal-ayuda-cerrar").addEventListener("click", () => ayuda.close?.());

  async function cargarCentros() {
    avisar("cal-estado", t("cargandoCentros"));
    $("cal-resultado").hidden = true;
    try {
      const datos = await cliente.centros(selAnio.value, instanteDesdeCampo($("cal-conocido").value));
      estado.centros = datos.centros;
      const previo = $("cal-centro").value;
      $("cal-centro").innerHTML = datos.centros.map((c) => `<option value="${esc(c.centro_ref)}"${c.centro_ref === previo ? " selected" : ""}>${esc(c.denominacion)}</option>`).join("");
      const municipios = [...new Set(datos.centros.map((c) => c.municipio_ref))];
      $("cal-residencia").innerHTML = `<option value="">${esc(t("plazoMismaSede"))}</option>` + municipios.map((m) => `<option value="${esc(m)}">${esc(etiquetaMunicipio(m))}</option>`).join("");
      if (!datos.centros.length) {
        avisar("cal-estado", t("sinCentros"));
        return;
      }
      await cargarCalendario();
    } catch (error) {
      avisar("cal-estado", mensajeError(error), true);
    }
  }

  async function cargarCalendario() {
    avisar("cal-estado", t("cargando"));
    try {
      const cal = await cliente.calendario($("cal-centro").value, selAnio.value, instanteDesdeCampo($("cal-conocido").value));
      estado.calendario = cal;
      $("cal-kpis").innerHTML = renderizarResumen(cal);
      $("cal-titulo-anual").textContent = t("calendarioTitulo", { anio: cal.anio });
      $("cal-meses").innerHTML = renderizarCalendario(cal);
      $("cal-festivos").innerHTML = renderizarFestivos(cal);
      $("cal-versiones").innerHTML = renderizarVersiones(cal.versiones);
      $("cal-notificacion").min = `${cal.anio}-01-01`;
      $("cal-notificacion").max = `${cal.anio}-12-31`;
      if (!$("cal-notificacion").value.startsWith(String(cal.anio))) $("cal-notificacion").value = `${cal.anio}-01-01`;
      $("cal-resultado").hidden = false;
      $("cal-plazo-resultado").innerHTML = "";
      avisar("cal-estado", "");
    } catch (error) {
      $("cal-resultado").hidden = true;
      avisar("cal-estado", mensajeError(error), true);
    }
  }

  $("cal-filtros").addEventListener("submit", (evento) => { evento.preventDefault(); cargarCalendario(); });
  selAnio.addEventListener("change", cargarCentros);
  $("cal-centro").addEventListener("change", cargarCalendario);
  $("cal-conocido").addEventListener("change", cargarCentros);
  $("cal-plazo").addEventListener("submit", async (evento) => {
    evento.preventDefault();
    const cal = estado.calendario;
    if (!cal) return;
    avisar("cal-plazo-estado", "");
    try {
      const resultado = await cliente.plazo({
        inicio: $("cal-notificacion").value, unidad: $("cal-unidad").value, cantidad: $("cal-cantidad").value,
        sede: cal.municipio_ref, residencia: $("cal-residencia").value, conocido_en: instanteDesdeCampo($("cal-conocido").value),
      });
      $("cal-plazo-resultado").innerHTML = renderizarPlazo(resultado);
    } catch (error) {
      $("cal-plazo-resultado").innerHTML = "";
      avisar("cal-plazo-estado", mensajeError(error), true);
    }
  });
  return cargarCentros();
}

if (typeof document !== "undefined" && document.getElementById("calendarios")) {
  iniciar(document, crearCliente());
}
