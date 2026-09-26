// Petición RRHH p.4: en la ficha de la participación, cada cambio registrado
// muestra el valor anterior y el nuevo. Situación y fecha de disponibilidad
// llegan con su valor; los datos de contacto solo con la versión cifrada que
// cambió (nunca el correo o el teléfono en claro).

import { actorTraducido } from "./portal-justificante.js";
import { traducirReferencia } from "./portal-referencias-i18n.js";
import { LOCALIZACION_PORTAL, ZONA_HORARIA_PORTAL } from "./portal-i18n.js?v=20260926-i18n-v1";

const CAMPOS = Object.freeze({
  situacion: "campo_situacion",
  fecha_disponible: "campo_fecha_disponible",
  datos_contacto: "campo_datos_contacto",
  correo: "campo_correo",
  telefono_1: "campo_telefono_1",
  telefono_2: "campo_telefono_2",
});

const SITUACIONES = Object.freeze(["disponible", "no_disponible", "trabajando", "pendiente_incorporacion", "renuncia", "excluido", "disponible_desde"]);
const VERSION = /^version:([1-9][0-9]{0,18})$/;
const FECHA = /^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z$/;

const TEXTOS = Object.freeze({
  titulo: "Cambios registrados",
  leyenda: "Cambios registrados con valor anterior y nuevo",
  vacio: "No hay cambios registrados.",
  col_fecha: "Fecha",
  col_campo: "Campo",
  col_anterior: "Valor anterior",
  col_nuevo: "Valor nuevo",
  col_actor: "Registrado por",
  sin_valor: "—",
  version: "Versión {n}",
  campo_situacion: "Situación",
  campo_fecha_disponible: "Disponible desde",
  campo_datos_contacto: "Datos de contacto",
  campo_correo: "Correo",
  campo_telefono_1: "Teléfono 1",
  campo_telefono_2: "Teléfono 2",
  situacion_disponible: "Disponible",
  situacion_no_disponible: "No disponible",
  situacion_trabajando: "Trabajando",
  situacion_pendiente_incorporacion: "Pendiente de incorporación",
  situacion_renuncia: "Renuncia",
  situacion_excluido: "Excluido",
  situacion_disponible_desde: "Disponible desde una fecha",
  paginacion: "Paginación de cambios registrados",
  mostrando: "Mostrando {desde} a {hasta} de {total}",
  anterior: "Anterior",
  siguiente: "Siguiente",
});

export function textoTraza(clave, valores = {}) {
  return String(TEXTOS[clave] ?? clave).replace(/\{(\w+)\}/g, (_, nombre) => String(valores[nombre] ?? ""));
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
  return `${titulo}<div class="tabla-contenedor"><table class="tabla-datos"><caption>${escaparHTML(textoTraza("leyenda"))}</caption><thead><tr>${cabecera}</tr></thead><tbody>${filas}</tbody></table></div>${navegacion}`;
}
