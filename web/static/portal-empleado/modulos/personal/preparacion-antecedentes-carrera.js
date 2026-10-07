import { cargarTextos } from "../../../comun/textos.js";
import { LOCALIZACION_ACTUAL } from "../../../comun/idioma.js";

const TEXTOS = await cargarTextos("personal-antecedentes-carrera");
const PENDIENTES = new Set([
  "fuente_institucional", "cobertura_antecedentes", "politica_carrera", "sin_relacion", "grupo_subgrupo", "puesto_nivel_m", "grado_personal_h",
  "regimen_sin_catalogo", "servicios_no_reconocidos", "servicios_fuera_corte", "sin_servicios", "sin_situaciones", "periodos_solapados", "hechos_fuera_corte",
]);
const objeto = (v) => v !== null && typeof v === "object" && !Array.isArray(v);
const referencia = (v) => typeof v === "string" && /^[a-z][a-z0-9_:-]{2,159}$/u.test(v);
const empleado = (v) => typeof v === "string" && /^emp_[A-Za-z0-9_-]{22,128}$/u.test(v);
const relacionRef = (v) => typeof v === "string" && /^rel_[A-Za-z0-9_-]{22,128}$/u.test(v);
const entero = (v) => Number.isSafeInteger(v) && v >= 1;
function fecha(v) {
  if (typeof v !== "string" || !/^\d{4}-\d{2}-\d{2}$/u.test(v)) return false;
  const d = new Date(`${v}T12:00:00Z`);
  return Number.isFinite(d.getTime()) && d.toISOString().slice(0, 10) === v;
}
function instante(v) { return typeof v === "string" && /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,6})?Z$/u.test(v) && Number.isFinite(Date.parse(v)); }
function lista(v, comprobar) { return Array.isArray(v) && v.length <= 200 && v.every(comprobar); }
function pendientes(v) { return lista(v, (p) => PENDIENTES.has(p)) && new Set(v).size === v.length; }
function traza(v) {
  return objeto(v) && fecha(v.desde) && (v.hasta === undefined || v.hasta === "" || fecha(v.hasta)) && instante(v.registrada_en) && entero(v.version) && referencia(v.acto_ref) && referencia(v.fuente_ref) && entero(v.fuente_version);
}
function servicio(v) {
  return objeto(v) && referencia(v.servicio_ref) && ["declarado", "comprobado", "reconocido"].includes(v.estado) && fecha(v.periodo_desde) && fecha(v.periodo_hasta) && Number.isSafeInteger(v.dias_reconocidos) && v.dias_reconocidos >= 0 && typeof v.ultima_revision_conocida === "boolean" && traza(v.traza);
}
function situacion(v) { return objeto(v) && referencia(v.situacion_ref) && referencia(v.codigo_ref) && ["vigente", "finalizada", "rectificada"].includes(v.estado) && traza(v.traza); }
function relacion(v) {
  return objeto(v) && relacionRef(v.relacion_ref) && ["vigente", "suspendida", "finalizada"].includes(v.estado) && referencia(v.regimen_ref) && typeof v.regimen === "string" && v.regimen.length <= 256 && !/[\x00-\x1f\x7f]/u.test(v.regimen) && traza(v.traza) && lista(v.historia, (h) => objeto(h) && referencia(h.regimen_ref) && typeof h.regimen === "string" && h.regimen.length <= 256 && ["vigente", "suspendida", "finalizada"].includes(h.estado) && traza(h.traza)) && pendientes(v.pendientes) && lista(v.servicios, servicio) && lista(v.situaciones, situacion);
}

// Valida la forma del resultado calculado por Personal; no calcula Carrera ni
// convierte la ficha del navegador en una fuente para otros módulos.
export function validarPreparacionAntecedentesCarrera(v) {
  if (!objeto(v) || v.esquema !== "vec.personal.preparacion-antecedentes-carrera.v1" || v.alcance !== "preparacion" || !empleado(v.empleado_ref) || !entero(v.version) || !objeto(v.corte) || !fecha(v.corte.vigente_en) || !instante(v.corte.conocido_en) || !pendientes(v.pendientes) || !lista(v.relaciones, relacion) || v.eficacia_administrativa === true || v.firma_oficial === true) throw new TypeError("personal.antecedentes_carrera.invalidos");
  return v;
}
function nodo(d, tipo, texto) { const n = d.createElement(tipo); if (texto !== undefined) n.textContent = texto; return n; }
function dato(d, dl, etiqueta, valor) { dl.append(nodo(d, "dt", etiqueta), nodo(d, "dd", valor)); }
function fechaVisible(v, locale) { return new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeZone: "UTC" }).format(new Date(`${v}T12:00:00Z`)); }
function instanteVisible(v, locale) { return new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }).format(new Date(v)); }
function listaPendientes(d, claves, t) {
  const ul = nodo(d, "ul");
  for (const clave of claves) ul.append(nodo(d, "li", t(`pendientes.${clave}`)));
  return ul;
}
function origen(d, tr, t, locale) {
  const details = nodo(d, "details"); details.className = "personal-registro-b2-traza";
  details.append(nodo(d, "summary", t("general.origen")));
  const dl = nodo(d, "dl");
  dato(d, dl, t("general.fecha_registro"), instanteVisible(tr.registrada_en, locale));
  dato(d, dl, t("general.version_hecho"), new Intl.NumberFormat(locale).format(tr.version));
  dato(d, dl, t("general.version_fuente"), new Intl.NumberFormat(locale).format(tr.fuente_version));
  dato(d, dl, t("general.acto"), tr.acto_ref);
  dato(d, dl, t("general.fuente"), tr.fuente_ref);
  details.append(dl); return details;
}
function tablaServicios(d, servicios, t, locale) {
  const region = nodo(d, "div"); region.className = "tabla-contenedor"; region.setAttribute("role", "region"); region.setAttribute("tabindex", "0"); region.setAttribute("aria-label", t("general.servicios"));
  const table = nodo(d, "table"); table.className = "tabla-datos"; table.append(nodo(d, "caption", t("general.servicios")));
  const thead = nodo(d, "thead"); const cabecera = nodo(d, "tr");
  for (const clave of ["estado", "periodo", "dias_acto", "origen"]) { const th = nodo(d, "th", t(`general.${clave}`)); th.setAttribute("scope", "col"); cabecera.append(th); }
  thead.append(cabecera); table.append(thead);
  const tbody = nodo(d, "tbody");
  for (const s of servicios) {
    const tr = nodo(d, "tr");
    tr.append(nodo(d, "td", t(s.ultima_revision_conocida ? "general.estado_ultima_revision" : "general.estado_revision_anterior", { estado: t(`estados.${s.estado}`) })), nodo(d, "td", t("general.periodo_fechas", { desde: fechaVisible(s.periodo_desde, locale), hasta: fechaVisible(s.periodo_hasta, locale) })), nodo(d, "td", new Intl.NumberFormat(locale).format(s.dias_reconocidos)));
    const td = nodo(d, "td"); td.append(origen(d, s.traza, t, locale)); tr.append(td); tbody.append(tr);
  }
  table.append(tbody); region.append(table); return region;
}

export function renderizarPreparacionAntecedentesCarrera({ modelo, documento = globalThis.document, t = TEXTOS.traducir, localizacion = LOCALIZACION_ACTUAL } = {}) {
  if (typeof documento?.createElement !== "function" || typeof t !== "function") throw new TypeError("personal.antecedentes_carrera.dependencias");
  const panel = nodo(documento, "section"); panel.className = "panel";
  const cabecera = nodo(documento, "header"); cabecera.className = "cabecera-panel"; cabecera.append(nodo(documento, "h3", t("general.titulo")));
  const cuerpo = nodo(documento, "div"); cuerpo.className = "cuerpo-panel"; panel.append(cabecera, cuerpo);
  try { validarPreparacionAntecedentesCarrera(modelo); } catch {
    const aviso = nodo(documento, "p", t("general.error")); aviso.setAttribute("role", "alert"); cuerpo.append(aviso); return panel;
  }
  const estado = nodo(documento, "p", t("general.preparacion")); estado.setAttribute("role", "status"); cuerpo.append(estado);
  const corte = nodo(documento, "dl");
  dato(documento, corte, t("general.vigente_en"), fechaVisible(modelo.corte.vigente_en, localizacion));
  dato(documento, corte, t("general.conocido_en"), instanteVisible(modelo.corte.conocido_en, localizacion));
  dato(documento, corte, t("general.version_ficha"), new Intl.NumberFormat(localizacion).format(modelo.version));
  cuerpo.append(corte, listaPendientes(documento, modelo.pendientes, t));
  for (const [i, r] of modelo.relaciones.entries()) {
    const detalle = nodo(documento, "details"); detalle.className = "personal-registro-b2-traza";
    const nombre = r.regimen || t("general.sin_denominacion");
    detalle.append(nodo(documento, "summary", t("general.relacion", { numero: new Intl.NumberFormat(localizacion).format(i + 1), regimen: nombre, estado: t(`estados.${r.estado}`) })));
    detalle.append(listaPendientes(documento, r.pendientes, t), origen(documento, r.traza, t, localizacion));
    if (r.servicios.length) detalle.append(tablaServicios(documento, r.servicios, t, localizacion));
    if (r.situaciones.length) {
      const lista = nodo(documento, "ul"); lista.setAttribute("aria-label", t("general.situaciones"));
      for (const s of r.situaciones) { const li = nodo(documento, "li", t(`estados.${s.estado}`)); li.append(origen(documento, s.traza, t, localizacion)); lista.append(li); }
      detalle.append(lista);
    }
    cuerpo.append(detalle);
  }
  return panel;
}
