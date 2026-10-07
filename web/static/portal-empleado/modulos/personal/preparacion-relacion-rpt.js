import { cargarTextos } from "../../../comun/textos.js";
import { LOCALIZACION_ACTUAL } from "../../../comun/idioma.js";

const textos = await cargarTextos("preparacion-relacion-rpt");
const esquema = "vec.personal.preparacion-relacion-rpt.v1";
const refRelacion = /^rel_[A-Za-z0-9_-]{22,128}$/u;
const referencia = /^[a-z][a-z0-9_:-]{2,159}$/u;
const estados = new Set(["vigente", "suspendida", "finalizada"]);
function fecha(valor) {
  if (typeof valor !== "string" || !/^\d{4}-\d{2}-\d{2}$/u.test(valor)) return false;
  const d = new Date(`${valor}T12:00:00Z`);
  return Number.isFinite(d.getTime()) && d.toISOString().slice(0, 10) === valor;
}
function instante(valor) { return typeof valor === "string" && /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,6})?Z$/u.test(valor) && Number.isFinite(Date.parse(valor)); }
function positivo(valor) { return Number.isSafeInteger(valor) && valor > 0; }
function referenciaValida(valor) { return typeof valor === "string" && referencia.test(valor); }
function validar(modelo) {
  if (!modelo || modelo.esquema !== esquema || modelo.uso !== "preparacion" || modelo.cobertura !== "no_acreditada" || modelo.estado_rpt !== "pendiente_fuente_rpt" ||
      typeof modelo.empleado_ref !== "string" || !/^emp_[A-Za-z0-9_-]{22,128}$/u.test(modelo.empleado_ref) ||
      !positivo(modelo.version_ficha) || !fecha(modelo.corte?.vigente_en) || !instante(modelo.corte?.conocido_en) ||
      !Array.isArray(modelo.relaciones) || modelo.relaciones.length > 200) throw new TypeError("preparacion_rpt_invalida");
  const ids = new Set();
  for (const r of modelo.relaciones) {
    const t = r?.traza;
    if (!r || !refRelacion.test(r.relacion_ref) || ids.has(r.relacion_ref) || !estados.has(r.estado) || typeof r.en_intervalo !== "boolean" ||
        !t || !fecha(t.desde) || (t.hasta && (!fecha(t.hasta) || t.hasta <= t.desde)) || !instante(t.registrada_en) ||
        !positivo(t.version) || !referenciaValida(t.acto_ref) || !referenciaValida(t.fuente_ref) || !positivo(t.fuente_version)) throw new TypeError("preparacion_rpt_invalida");
    ids.add(r.relacion_ref);
  }
}
function nodo(d, tag, texto) { const n = d.createElement(tag); if (texto !== undefined) n.textContent = texto; return n; }

/** Representa exclusivamente el modelo calculado por Personal tras la lectura
 * B2 autorizada. No consulta, acredita ni recalcula relaciones u ocupaciones.
 * El montaje debe retirar el panel al cambiar sujeto, revocar o fallar la lectura.
 */
export function crearPanelRelacionParaRPT({ documento, modelo, relacionRef = "", traducir = textos.traducir, localizacion = LOCALIZACION_ACTUAL }) {
  validar(modelo);
  const t = (clave) => traducir(`general.${clave}`);
  const panel = nodo(documento, "section"); panel.className = "panel";
  const cabecera = nodo(documento, "header"); cabecera.className = "cabecera-panel";
  cabecera.append(nodo(documento, "h3", t("titulo")));
  const cuerpo = nodo(documento, "div"); cuerpo.className = "cuerpo-panel";
  panel.append(cabecera, cuerpo);
  const pendiente = nodo(documento, "p", t("pendiente")); pendiente.className = "personal-registro-b2-estado";
  cuerpo.append(pendiente);
  const relacion = modelo.relaciones.find((r) => r.relacion_ref === relacionRef);
  if (!relacion) {
    cuerpo.append(nodo(documento, "p", t(modelo.relaciones.length === 0 ? "vacio" : relacionRef ? "relacion_no_disponible" : "seleccione")));
    return panel;
  }
  const f = (valor) => new Intl.DateTimeFormat(localizacion, { dateStyle: "medium", timeZone: "Europe/Madrid" }).format(new Date(`${valor}T12:00:00Z`));
  const numero = (valor) => new Intl.NumberFormat(localizacion).format(valor);
  const datos = nodo(documento, "div"); datos.className = "personal-registro-b2-traza";
  const lista = nodo(documento, "dl");
  const dato = (destino, clave, valor) => destino.append(nodo(documento, "dt", t(clave)), nodo(documento, "dd", valor));
  dato(lista, "estado", t(relacion.estado));
  dato(lista, "periodo", `${f(relacion.traza.desde)} – ${relacion.traza.hasta ? f(relacion.traza.hasta) : t("sin_fin")}`);
  dato(lista, "intervalo", t(relacion.en_intervalo ? "si" : "no"));
  dato(lista, "fuente", t("sin_denominacion"));
  dato(lista, "version", numero(relacion.traza.version));
  dato(lista, "conocido", new Intl.DateTimeFormat(localizacion, { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }).format(new Date(modelo.corte.conocido_en)));
  datos.append(lista); cuerpo.append(datos);
  const detalle = nodo(documento, "details"); detalle.className = "personal-registro-b2-traza-tecnica";
  detalle.append(nodo(documento, "summary", t("detalle_tecnico")));
  const referencias = nodo(documento, "dl");
  dato(referencias, "relacion_ref", relacion.relacion_ref);
  dato(referencias, "fuente_ref", relacion.traza.fuente_ref);
  dato(referencias, "fuente_version", numero(relacion.traza.fuente_version));
  dato(referencias, "acto_ref", relacion.traza.acto_ref);
  detalle.append(referencias); datos.append(detalle);
  return panel;
}
