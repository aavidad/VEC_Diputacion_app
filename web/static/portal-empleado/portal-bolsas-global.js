import { cargarTextos } from "../comun/textos.js";
import { LOCALIZACION_PORTAL, ZONA_HORARIA_PORTAL, traducirBolsaInterna, traducirPortal } from "./portal-i18n.js?v=20261007-pantallas-textos-final-v1";
import { rutaCandidatosBolsaCompartible } from "./portal-bolsas-ruta-filtros.js?v=20261010-bolsa-respuesta-portal-v1";

let mensajes;
export async function prepararTextosGlobalBolsa() {
  mensajes = (await cargarTextos("portal-bolsa")).seccion("global");
}
function t(clave, variables = {}) {
  return mensajes[clave].replace(/\{([a-z_]+)\}/gu, (_, campo) => String(variables[campo] ?? ""));
}
function fecha(valor) {
  const instante = new Date(valor);
  return valor && Number.isFinite(instante.getTime())
    ? new Intl.DateTimeFormat(LOCALIZACION_PORTAL, { dateStyle: "short", timeStyle: "short", timeZone: ZONA_HORARIA_PORTAL }).format(instante)
    : t("sin_fecha");
}
function numero(valor) { return new Intl.NumberFormat(LOCALIZACION_PORTAL).format(valor); }

export function renderizarGlobalBolsa(estado, { encabezadoVista, escaparHTML }) {
  if (!mensajes && estado?.carga === "cargando") return `${encabezadoVista("", traducirPortal("txt_bolsas_de_trabajo"), "", "")}<section class="panel" role="status" aria-busy="true"><div class="cuerpo-panel">${escaparHTML(traducirPortal("txt_cargando_lista_de_candidatos"))}</div></section>`;
  if (!mensajes) return `${encabezadoVista("", traducirPortal("txt_bolsas_de_trabajo"), "", "")}<section class="panel" role="alert"><div class="cuerpo-panel"><p>${escaparHTML(traducirPortal("txt_no_se_pudo_cargar_la_relacion_de_aspirantes"))}</p><button type="button" class="boton-secundario" data-bolsa-accion="reintentar-global">${escaparHTML(traducirPortal("txt_reintentar"))}</button><button type="button" class="boton-secundario" data-vista="resumen">${escaparHTML(traducirPortal("txt_volver_al_cuadro"))}</button></div></section>`;
  const filtro = estado?.filtro || "todos";
  const titulo = t(filtro);
  const cabecera = encabezadoVista("", `${t("titulo")} · ${titulo}`, "",
    `<button type="button" class="boton-secundario" data-vista="resumen">${t("volver")}</button>`);
  if (!estado || estado.carga === "cargando") return `${cabecera}<section class="panel" role="status" aria-busy="true"><div class="cuerpo-panel">${t("cargando")}</div></section>`;
  if (estado.carga === "caducado") return `${cabecera}<section class="panel" role="status"><div class="cuerpo-panel"><p>${t("caducado")}</p><button type="button" class="boton-secundario" data-bolsa-accion="actualizar-global">${t("actualizar")}</button></div></section>`;
  if (estado.carga !== "listo") return `${cabecera}<section class="panel" role="alert"><div class="cuerpo-panel"><p>${estado.carga === "denegado" ? t("denegado") : t("error")}</p>${estado.carga === "error" ? `<button type="button" class="boton-secundario" data-bolsa-accion="reintentar-global">${t("reintentar")}</button>` : ""}</div></section>`;
  const datos = estado.datos;
  const esLlamamientos = filtro === "llamamientos";
  const enlaceBolsa = (item) => `<a class="enlace-tabla" href="${escaparHTML(rutaCandidatosBolsaCompartible(globalThis.location?.search ?? "", item.bolsa_ref))}" data-accion="ver-bolsa" data-bolsa-ref="${escaparHTML(item.bolsa_ref)}">${t("ver_bolsa")}</a>`;
  const filas = datos.items.map((item, indice) => esLlamamientos
    ? `<tr><th scope="row">${escaparHTML(item.categoria || t("sin_categoria"))}</th><td>${numero(datos.desde + indice)}</td><td><time datetime="${escaparHTML(item.emitido_en)}">${escaparHTML(fecha(item.emitido_en))}</time></td><td>${enlaceBolsa(item)}</td></tr>`
    : `<tr><th scope="row">${escaparHTML(item.categoria || t("sin_categoria"))}</th><td>${Number.isSafeInteger(item.orden_acta) ? numero(item.orden_acta) : t("sin_posicion")}</td><td>${escaparHTML(item.estado_clave === "en_revision" ? traducirPortal("txt_b8_en_revision") : traducirBolsaInterna(`bolsa_estado_${item.estado_clave}`))}</td><td><time datetime="${escaparHTML(item.estado_desde)}">${escaparHTML(fecha(item.estado_desde))}</time></td><td>${enlaceBolsa(item)}</td></tr>`).join("");
  const columnas = esLlamamientos ? ["categoria", "llamamiento", "fecha", "bolsa"] : ["categoria", "posicion", "situacion", "fecha", "bolsa"];
  return `${cabecera}<section class="panel"><div class="cabecera-panel"><h3>${titulo}</h3><span role="status">${t("mostrando", { desde: numero(datos.desde), hasta: numero(datos.hasta), total: numero(datos.total) })}</span></div><div class="tabla-contenedor" tabindex="0" role="region" aria-label="${titulo}"><table class="tabla-datos"><thead><tr>${columnas.map((clave) => `<th scope="col">${t(clave)}</th>`).join("")}</tr></thead><tbody>${filas || `<tr><td colspan="${columnas.length}" class="vacio-controlado">${t("vacio")}</td></tr>`}</tbody></table></div>${datos.desde > 1 || datos.hay_mas ? `<div class="cuerpo-panel">${datos.desde > 1 ? `<button type="button" class="boton-secundario" data-bolsa-accion="pagina-global" data-cursor="${Math.max(0, datos.desde - 51)}">${t("anterior")}</button>` : ""}${datos.hay_mas ? `<button type="button" class="boton-secundario" data-bolsa-accion="pagina-global" data-cursor="${escaparHTML(datos.cursor_siguiente)}">${t("siguiente")}</button>` : ""}</div>` : ""}</section>`;
}
