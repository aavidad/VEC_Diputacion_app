// Petición RRHH p.4: en la ficha de la participación, cada cambio registrado
// muestra el valor anterior y el nuevo. Situación y fecha de disponibilidad
// llegan con su valor; los datos de contacto solo con la versión cifrada que
// cambió (nunca el correo o el teléfono en claro).

import { actorTraducido } from "./portal-justificante.js";
import { traducirReferencia } from "./portal-referencias-i18n.js?v=20261007-pantallas-textos-final-v1";
import { LOCALIZACION_PORTAL, ZONA_HORARIA_PORTAL, traducirPortal } from "./portal-i18n.js?v=20261007-pantallas-textos-final-v1";

const CAMPOS = Object.freeze({
  situacion: "campo_situacion",
  fecha_disponible: "campo_fecha_disponible",
  datos_contacto: "campo_datos_contacto",
  correo: "campo_correo",
  telefono_1: "campo_telefono_1",
  telefono_2: "campo_telefono_2",
});

const SITUACIONES = Object.freeze(["disponible", "no_disponible", "trabajando", "pendiente_incorporacion", "renuncia", "excluido", "disponible_desde", "en_revision"]);
const VERSION = /^version:([1-9][0-9]{0,18})$/;
const FECHA = /^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z$/;

const CLAVES_COMUNES = Object.freeze({
  col_fecha: "txt_fecha",
  campo_situacion: "txt_situacion",
  campo_fecha_disponible: "txt_disponible_desde",
  campo_correo: "txt_correo",
  situacion_disponible: "txt_disponible",
  situacion_no_disponible: "txt_no_disponible",
  situacion_trabajando: "txt_trabajando",
  situacion_pendiente_incorporacion: "txt_pendiente_de_incorporacion",
  situacion_renuncia: "txt_renuncia",
  situacion_excluido: "txt_excluido",
  situacion_en_revision: "txt_b8_en_revision",
  mostrando: "txt_mostrando_desde_hasta_total",
  anterior: "txt_anterior",
  siguiente: "txt_siguiente",
});

export function textoTraza(clave, valores = {}) {
  return traducirPortal(CLAVES_COMUNES[clave] ?? `traza_${clave}`, valores);
}

function valorValido(campo, valor) {
  if (valor === null) return true;
  if (typeof valor !== "string") return false;
  if (campo === "situacion") return SITUACIONES.includes(valor);
  if (campo === "fecha_disponible") return FECHA.test(valor);
  return VERSION.test(valor);
}

// validarCambiosTraza acepta la ausencia de la clave (servidor sin la
// migración) y rechaza cualquier valor que no tenga la forma minimizada.
export function validarCambiosTraza(cambios) {
  if (cambios === undefined) return [];
  if (!Array.isArray(cambios)) return null;
  const valido = cambios.every((c) => c && typeof c === "object" && Object.hasOwn(CAMPOS, c.campo)
    && typeof c.instante === "string" && !Number.isNaN(Date.parse(c.instante))
    && typeof c.recibo_ref === "string" && typeof c.actor === "string"
    && valorValido(c.campo, c.valor_anterior) && valorValido(c.campo, c.valor_nuevo)
    && !(c.valor_anterior === null && c.valor_nuevo === null));
  return valido ? cambios : null;
}

const formatoFecha = new Intl.DateTimeFormat(LOCALIZACION_PORTAL, { dateStyle: "short", timeStyle: "short", timeZone: ZONA_HORARIA_PORTAL });

function fecha(valor) {
  const instante = new Date(valor);
  return Number.isNaN(instante.getTime()) ? String(valor) : formatoFecha.format(instante);
}

function valorVisible(campo, valor) {
  if (valor === null) return textoTraza("sin_valor");
  if (campo === "situacion") return textoTraza(`situacion_${valor}`);
  if (campo === "fecha_disponible") return fecha(valor);
  return textoTraza("version", { n: VERSION.exec(valor)?.[1] ?? valor });
}

export function renderizarTrazaValores({ cambios = [], pagina = 0, escaparHTML, porPagina = 6 }) {
  const titulo = `<h4>${escaparHTML(textoTraza("titulo"))}</h4>`;
  if (!cambios.length) return `${titulo}<p class="vacio-controlado" role="status">${escaparHTML(textoTraza("vacio"))}</p>`;
  const total = cambios.length;
  const paginas = Math.max(1, Math.ceil(total / porPagina));
  const actual = Math.min(Math.max(0, Number(pagina) || 0), paginas - 1);
  const visibles = cambios.slice(actual * porPagina, (actual + 1) * porPagina);
  const filas = visibles.map((c) => `<tr><td><time datetime="${escaparHTML(c.instante)}">${escaparHTML(fecha(c.instante))}</time></td><th scope="row">${escaparHTML(textoTraza(CAMPOS[c.campo]))}</th><td>${escaparHTML(valorVisible(c.campo, c.valor_anterior))}</td><td>${escaparHTML(valorVisible(c.campo, c.valor_nuevo))}</td><td>${actorTraducido(c.actor, escaparHTML, traducirReferencia)}</td></tr>`).join("");
  const cabecera = ["col_fecha", "col_campo", "col_anterior", "col_nuevo", "col_actor"].map((clave) => `<th scope="col">${escaparHTML(textoTraza(clave))}</th>`).join("");
  const desde = actual * porPagina + 1;
  const hasta = Math.min((actual + 1) * porPagina, total);
  const resumen = escaparHTML(textoTraza("mostrando", { desde, hasta, total }));
  const navegacion = paginas > 1
    ? `<nav class="paginacion-bolsa" aria-label="${escaparHTML(textoTraza("paginacion"))}"><span>${resumen}</span><button type="button" class="boton-secundario" data-b8-accion="pagina-traza" data-pagina="${actual - 1}" ${actual === 0 ? "disabled" : ""}>${escaparHTML(textoTraza("anterior"))}</button><button type="button" class="boton-secundario" data-b8-accion="pagina-traza" data-pagina="${actual + 1}" ${actual + 1 >= paginas ? "disabled" : ""}>${escaparHTML(textoTraza("siguiente"))}</button></nav>`
    : `<p>${resumen}</p>`;
  return `${titulo}<div class="tabla-contenedor" tabindex="0" role="region" aria-label="${escaparHTML(textoTraza("leyenda"))}"><table class="tabla-datos"><caption>${escaparHTML(textoTraza("leyenda"))}</caption><thead><tr>${cabecera}</tr></thead><tbody>${filas}</tbody></table></div>${navegacion}`;
}
