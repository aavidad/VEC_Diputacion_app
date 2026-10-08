import { cargarTextos } from "../../../comun/textos.js";
import { ZONA_HORARIA_PORTAL } from "../../portal-i18n.js?v=20261007-pantallas-textos-final-v1";

const TEXTOS = await cargarTextos("tramites-empleado");
const TAMANO = 20;
const BLOQUES = Object.freeze(["cronos", "dietas"]);
const ESTADOS = Object.freeze({
  cronos: ["solicitado", "pendiente_administracion", "concedido", "denegado", "cancelado"],
  dietas: ["borrador", "eliminado", "enviado_pendiente_revision", "pendiente_autorizacion", "pendiente_liquidacion", "pendiente_fiscalizacion", "fiscalizada", "devuelta"],
});
const escapar = (valor) => String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
const tDe = (textos) => (clave, variables) => textos.traducir(`general.${clave}`, variables);
const textoDato = (valor, t) => typeof valor === "string" && valor.trim() ? valor : t("sin_dato");
const numero = (valor, localizacion) => new Intl.NumberFormat(localizacion).format(valor);

function fechaCivilValida(valor) {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(valor)) return false;
  const fecha = new Date(`${valor}T12:00:00Z`);
  return Number.isFinite(fecha.getTime()) && fecha.toISOString().slice(0, 10) === valor;
}
function fecha(valor, t, localizacion, instante = false) {
  if (typeof valor !== "string" || !fechaCivilValida(valor.slice(0, 10))) return t("sin_dato");
  const civil = valor.length === 10;
  if (!civil && !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$/.test(valor)) return t("sin_dato");
  if (instante && civil) return t("sin_dato");
  const dato = new Date(civil ? `${valor}T12:00:00Z` : valor);
  if (!Number.isFinite(dato.getTime())) return t("sin_dato");
  return new Intl.DateTimeFormat(localizacion, {
    dateStyle: "short", ...(instante ? { timeStyle: "short" } : {}),
    timeZone: civil ? "UTC" : ZONA_HORARIA_PORTAL,
  }).format(dato);
}
const periodo = (desde, hasta, t, loc) => t("periodo", { desde: fecha(desde, t, loc), hasta: fecha(hasta, t, loc) });
const estadoOriginal = (bloque, valor, t) => t(ESTADOS[bloque].includes(valor) ? `${bloque}_estado_${valor}` : "estado_no_disponible");
function etiquetaEstado(bloque, valor, t) {
  const tono = ["concedido", "fiscalizada"].includes(valor) ? "exito"
    : ["denegado", "devuelta"].includes(valor) ? "peligro"
      : ["cancelado", "eliminado", "borrador"].includes(valor) || !ESTADOS[bloque].includes(valor) ? "neutro" : "info";
  return `<span class="estado-chip ${tono}">${escapar(estadoOriginal(bloque, valor, t))}</span>`;
}
function enlace(bloque, t) {
  const destino = bloque === "cronos" ? "cronos-permisos" : "dietas";
  return `<a href="#${destino}" data-vista="${destino}" id="tramites-${bloque}-enlace">${escapar(t(`${bloque}_enlace`))}</a>`;
}
function boton(bloque, accion, etiqueta, impedido = false) {
  return `<button type="button" class="boton-secundario" id="tramites-${bloque}-${accion}" data-tramites-bloque="${bloque}" data-tramites-accion="${accion}" aria-disabled="${impedido}">${escapar(etiqueta)}</button>`;
}
function tabla(bloque, columnas, filas, t) {
  return `<div class="tabla-contenedor" role="region" tabindex="0" aria-label="${escapar(t(`${bloque}_tabla`))}" id="tramites-${bloque}-tabla"><table class="tabla-datos${bloque === "dietas" ? " tabla-datos--prioritaria tabla-apilable" : ""}"><caption>${escapar(t(`${bloque}_tabla`))}</caption><thead><tr>${columnas.map((clave) => `<th scope="col">${escapar(t(clave))}</th>`).join("")}</tr></thead><tbody>${filas}</tbody></table></div>`;
}
function tablaCronos(panel, t, loc) {
  const inicio = panel.pagina * TAMANO;
  const filas = panel.datos.solicitudes.slice(inicio, inicio + TAMANO).map((item) => `<tr>
    <th scope="row">${escapar(textoDato(item.nombre, () => t("cronos_nombre")))}</th>
    <td>${etiquetaEstado("cronos", item.estado, t)}</td>
    <td>${escapar(periodo(item.desde, item.hasta, t, loc))}</td>
    <td>${escapar(fecha(item.solicitada_en, t, loc, true))}</td>
    <td>${escapar(item.pendiente_justificar ? t("pendiente_justificar") : item.pendiente_asignacion ? t("pendiente_asignacion") : t("sin_pendientes"))}</td>
  </tr>`).join("");
  return tabla("cronos", ["solicitud", "estado", "periodo_solicitado", "fecha_solicitud", "pendiente"], filas, t);
}
function recibo(item, t, loc) {
  if (!item.recibo) return escapar(t("sin_dato"));
  return `<details><summary>${escapar(t("ver_justificante"))}</summary><dl><dt>${escapar(t("referencia_operacion"))}</dt><dd>${escapar(textoDato(item.recibo.referencia, t))}</dd><dt>${escapar(t("version_justificante"))}</dt><dd>${escapar(numero(item.recibo.version, loc))}</dd><dt>${escapar(t("fecha_operacion"))}</dt><dd>${escapar(fecha(item.recibo.registrado_en, t, loc, true))}</dd></dl></details>`;
}
function devolucion(item, t, loc) {
  const dato = item.comision.devolucion;
  if (!dato) return escapar(t("sin_pendientes"));
  const campos = [
    ["devolucion_motivo", dato.motivo],
    ["devolucion_etapa", t(`devolucion_etapa_${dato.etapa}`)],
    ["devolucion_version", numero(dato.version, loc)],
    ["devolucion_fecha", fecha(dato.devuelta_en, t, loc, true)],
  ];
  return `<details data-tramites-devolucion><summary>${escapar(t("devolucion_ver"))}</summary>
    <dl class="datos-clave">${campos.map(([clave, valor]) => `<div><dt>${escapar(t(clave))}</dt><dd>${escapar(valor)}</dd></div>`).join("")}</dl>
    <a href="#dietas" data-vista="dietas">${escapar(t("devolucion_abrir"))}</a></details>`;
}
function tablaDietas(panel, t, loc) {
  const filas = panel.datos.items.map((item) => `<tr>
    <th scope="row">${escapar(textoDato(item.comision.numero_documento, () => t("dietas_nombre")))}</th>
    <td data-etiqueta="${escapar(t("estado"))}">${etiquetaEstado("dietas", item.comision.estado, t)}</td>
    <td data-etiqueta="${escapar(t("periodo_comision"))}">${escapar(periodo(item.comision.fecha_inicio, item.comision.fecha_fin, t, loc))}</td>
    <td class="envuelve" data-etiqueta="${escapar(t("pendiente"))}">${devolucion(item, t, loc)}</td>
    <td data-etiqueta="${escapar(t("justificante_operacion"))}">${recibo(item, t, loc)}</td>
  </tr>`).join("");
  return tabla("dietas", ["comision", "estado", "periodo_comision", "pendiente", "justificante_operacion"], filas, t);
}
function paginacion(bloque, panel, t, loc) {
  const esCronos = bloque === "cronos";
  const anterior = esCronos ? panel.pagina > 0 : panel.cursores.length > 0;
  const siguiente = esCronos ? (panel.pagina + 1) * TAMANO < (panel.datos?.solicitudes.length ?? 0) : Boolean(panel.datos?.siguiente_cursor);
  const disponible = ["disponible", "vacio"].includes(panel.situacion);
  const mensaje = esCronos && panel.situacion === "disponible" ? t("cronos_recuento", {
    inicio: numero(panel.pagina * TAMANO + 1, loc),
    fin: numero(Math.min((panel.pagina + 1) * TAMANO, panel.datos.solicitudes.length), loc),
    total: numero(panel.datos.solicitudes.length, loc),
  }) : !esCronos && disponible ? t("dietas_pagina", { pagina: numero(panel.cursores.length + 1, loc) }) : "";
  return `<nav class="acciones-fila paginacion-marco" aria-label="${escapar(t(`${bloque}_paginacion`))}"><span>${escapar(mensaje)}</span>${boton(bloque, "anterior", t("anterior"), !anterior || !disponible)}${boton(bloque, "siguiente", t("siguiente"), !siguiente || !disponible)}</nav>`;
}
function contenidoPanel(bloque, panel, t, loc) {
  const situacion = panel.situacion;
  const sinAutenticacion = panel.codigo === "autenticacion_requerida";
  const cabecera = `<header class="cabecera-panel"><h3 id="tramites-${bloque}-titulo">${escapar(t(`${bloque}_titulo`))}</h3>${boton(bloque, "consultar", t(situacion === "error" ? "reintentar" : "actualizar"), sinAutenticacion || situacion === "cargando" || situacion === "no_configurado")}</header>`;
  const filtro = bloque === "cronos" ? `<form class="acciones-fila" data-tramites-anio novalidate><div class="campo-filtro"><label for="tramites-cronos-anio">${escapar(t("anio"))}</label><input id="tramites-cronos-anio" name="anio" type="number" inputmode="numeric" min="2000" max="2100" step="1" required value="${escapar(panel.anio)}" aria-invalid="${Boolean(panel.errorAnio)}"${sinAutenticacion ? " disabled" : ""}${panel.errorAnio ? ' aria-describedby="tramites-cronos-anio-error"' : ""}></div><button class="boton-secundario" id="tramites-cronos-aplicar" type="submit"${sinAutenticacion ? " disabled" : ""}>${escapar(t("consultar_anio"))}</button></form>${panel.errorAnio ? `<p id="tramites-cronos-anio-error" role="alert">${escapar(t("anio_invalido"))}</p>` : ""}` : "";
  const mensaje = situacion === "cargando" ? t(`${bloque}_cargando`)
    : situacion === "vacio" ? t(`${bloque}_vacio`)
      : situacion === "disponible" ? t(`${bloque}_consulta_lista`)
        : t(sinAutenticacion ? "autenticacion_requerida" : panel.codigo === "relacion_ambigua" ? "relacion_ambigua" : `${bloque}_${situacion}`);
  const listado = situacion === "disponible" ? (bloque === "cronos" ? tablaCronos(panel, t, loc) : tablaDietas(panel, t, loc)) : "";
  const aviso = sinAutenticacion ? "" : `<p id="tramites-${bloque}-estado" role="status" aria-live="polite">${escapar(mensaje)}</p>`;
  return `${cabecera}<div class="cuerpo-panel">${filtro}${aviso}${listado}${paginacion(bloque, panel, t, loc)}<div class="acciones-fila">${enlace(bloque, t)}</div></div>`;
}

/** Los paneles sólo presentan las proyecciones propias de sus autoridades. */
export function renderizarVistaTramitesPropios(estado = {}, { textos = TEXTOS, t = tDe(textos), localizacion = textos.localizacion } = {}) {
  const paneles = BLOQUES.filter((bloque) => estado[bloque]?.visible);
  return `<section class="columna-cuadro" data-tramites-propios><h2>${escapar(t("titulo"))}</h2><p id="tramites-autenticacion-estado" tabindex="-1" hidden></p>${paneles.length ? paneles.map((bloque) => `<section class="panel" data-tramites-panel="${bloque}" aria-labelledby="tramites-${bloque}-titulo" aria-busy="${estado[bloque].situacion === "cargando"}">${contenidoPanel(bloque, estado[bloque], t, localizacion)}</section>`).join("") : `<p role="status">${escapar(t("sin_fuentes"))}</p>`}</section>`;
}

function validarRespuesta(bloque, datos, anio) {
  const objeto = (valor) => valor && typeof valor === "object" && !Array.isArray(valor);
  if (!objeto(datos)) throw new TypeError("respuesta_incompatible");
  if (bloque === "cronos") {
    if (datos.anio !== anio || !Array.isArray(datos.solicitudes) || datos.solicitudes.some((item) => !objeto(item) || typeof item.solicitud_ref !== "string" || !item.solicitud_ref.trim())) throw new TypeError("respuesta_incompatible");
  } else if (!Array.isArray(datos.items) || datos.items.length > TAMANO || datos.items.some((item) => !objeto(item?.comision) || typeof item.comision.referencia !== "string" || !item.comision.referencia.trim() || (item.recibo !== undefined && !objeto(item.recibo))) || (datos.siguiente_cursor !== undefined && (typeof datos.siguiente_cursor !== "string" || !datos.siguiente_cursor || new TextEncoder().encode(datos.siguiente_cursor).length > 400))) throw new TypeError("respuesta_incompatible");
  return datos;
}
function situacionError(codigo) {
  if (["acceso_denegado", "autenticacion_requerida"].includes(codigo)) return "denegado";
  if (codigo === "fuente_no_configurada") return "no_configurado";
  return "error";
}

/** Sin identidad en el cliente: la fuente inyectada resuelve y autoriza la consulta. */
export function montarVistaTramitesPropios({ raiz, fuente, anunciar = () => {}, registrarDesmontar, textos = TEXTOS, t = tDe(textos), localizacion = textos.localizacion, ahora = () => new Date() } = {}) {
  if (!raiz?.replaceChildren || typeof t !== "function" || typeof anunciar !== "function" || typeof ahora !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) throw new TypeError("montaje_incompatible");
  const anio = Number(new Intl.DateTimeFormat(localizacion, { year: "numeric", timeZone: ZONA_HORARIA_PORTAL }).format(ahora()));
  const estado = Object.fromEntries(BLOQUES.map((bloque) => [bloque, {
    visible: fuente?.disponibles?.[bloque] !== false,
    situacion: "cargando", anio, pagina: 0, cursores: [], cursor: undefined,
    datos: undefined, secuencia: 0, controlador: undefined, codigo: undefined,
  }]));
  let activa = true;
  let autenticacionRequerida = false;
  raiz.innerHTML = renderizarVistaTramitesPropios(estado, { textos, t, localizacion });
  function pintar(bloque) {
    if (!activa) return;
    const panel = raiz.querySelector(`[data-tramites-panel="${bloque}"]`);
    if (!panel) return;
    const foco = raiz.ownerDocument?.activeElement;
    const id = panel.contains?.(foco) && /^tramites-(cronos|dietas)-[a-z-]+$/.test(foco?.id) ? foco.id : undefined;
    const entradaAnio = id === "tramites-cronos-anio" ? foco.value : undefined;
    panel.innerHTML = contenidoPanel(bloque, estado[bloque], t, localizacion);
    panel.setAttribute?.("aria-busy", String(estado[bloque].situacion === "cargando"));
    if (id) {
      const reemplazo = panel.querySelector?.(`#${id}`);
      if (entradaAnio !== undefined && reemplazo) reemplazo.value = entradaAnio;
      reemplazo?.focus?.();
    }
  }
  function purgar(panel) {
    panel.secuencia += 1;
    panel.controlador?.abort();
    Object.assign(panel, { controlador: undefined, datos: undefined, cursor: undefined, cursores: [], pagina: 0, errorAnio: false });
  }
  function requerirAutenticacion() {
    autenticacionRequerida = true;
    const focoEnVista = raiz.contains?.(raiz.ownerDocument?.activeElement);
    for (const bloque of BLOQUES) {
      const panel = estado[bloque];
      purgar(panel);
      Object.assign(panel, { situacion: "denegado", codigo: "autenticacion_requerida" });
      pintar(bloque);
    }
    const mensaje = t("autenticacion_requerida");
    const aviso = raiz.querySelector("#tramites-autenticacion-estado");
    if (aviso) {
      aviso.textContent = mensaje;
      aviso.hidden = false;
      const foco = raiz.ownerDocument?.activeElement;
      if (focoEnVista && (!raiz.contains?.(foco) || foco?.disabled || foco?.getAttribute?.("aria-disabled") === "true")) aviso.focus?.();
    }
    anunciar(mensaje, "error");
  }
  async function consultar(bloque, opciones = {}) {
    if (!activa || autenticacionRequerida) return;
    const panel = estado[bloque];
    // La primera página tiene cursor undefined; no sustituirlo por el cursor actual.
    const cursor = Object.hasOwn(opciones, "cursor") ? opciones.cursor : panel.cursor;
    const cursores = opciones.cursores ?? panel.cursores;
    panel.controlador?.abort();
    const controlador = new AbortController();
    const secuencia = ++panel.secuencia;
    Object.assign(panel, { controlador, cursor, cursores, situacion: "cargando", datos: undefined, codigo: undefined });
    pintar(bloque);
    try {
      const metodo = bloque === "cronos" ? fuente?.consultarCronos : fuente?.consultarDietas;
      if (typeof metodo !== "function") throw { codigo: "fuente_no_configurada" };
      const resultado = await metodo.call(fuente, bloque === "cronos" ? { anio: panel.anio, signal: controlador.signal } : { cursor, signal: controlador.signal });
      if (!activa || secuencia !== panel.secuencia || controlador.signal.aborted) return;
      const datos = validarRespuesta(bloque, resultado, panel.anio);
      if (bloque === "cronos") panel.pagina = Math.min(panel.pagina, Math.max(0, Math.ceil(datos.solicitudes.length / TAMANO) - 1));
      Object.assign(panel, { datos, cursor, cursores, situacion: (bloque === "cronos" ? datos.solicitudes : datos.items).length ? "disponible" : "vacio" });
    } catch (error) {
      if (!activa || secuencia !== panel.secuencia || controlador.signal.aborted) return;
      if (error?.codigo === "autenticacion_requerida") { requerirAutenticacion(); return; }
      Object.assign(panel, { situacion: situacionError(error?.codigo), codigo: error?.codigo, datos: undefined });
    }
    pintar(bloque);
  }
  function aplicarAnio(formulario) {
    if (!activa || autenticacionRequerida) return;
    const panel = estado.cronos;
    const valor = String(formulario.elements?.anio?.value ?? "");
    const nuevo = /^\d{4}$/.test(valor) ? Number(valor) : NaN;
    if (!Number.isInteger(nuevo) || nuevo < 2000 || nuevo > 2100) {
      panel.errorAnio = true; pintar("cronos");
      const entrada = raiz.querySelector("#tramites-cronos-anio");
      if (entrada) { entrada.value = valor; entrada.focus?.(); } return;
    }
    if (nuevo === panel.anio && panel.situacion === "cargando") {
      if (panel.errorAnio) { panel.errorAnio = false; pintar("cronos"); }
      return;
    }
    Object.assign(panel, { anio: nuevo, pagina: 0, errorAnio: false });
    void consultar("cronos");
  }
  const alSubmit = (evento) => {
    if (!evento.target?.matches?.("[data-tramites-anio]")) return;
    evento.preventDefault(); aplicarAnio(evento.target);
  };
  const alCambio = (evento) => {
    if (evento.target?.id !== "tramites-cronos-anio") return;
    const formulario = evento.target.closest?.("[data-tramites-anio]");
    if (formulario) aplicarAnio(formulario);
  };
  const alClick = (evento) => {
    if (!activa || autenticacionRequerida) return;
    const control = evento.target?.closest?.("[data-tramites-accion]");
    if (!control || control.getAttribute?.("aria-disabled") === "true") return;
    const { tramitesBloque: bloque, tramitesAccion: accion } = control.dataset;
    if (!BLOQUES.includes(bloque) || !estado[bloque].visible) return;
    const panel = estado[bloque];
    if (accion === "consultar" && !["cargando", "no_configurado"].includes(panel.situacion)) { void consultar(bloque); return; }
    if (!["disponible", "vacio"].includes(panel.situacion)) return;
    if (bloque === "cronos") {
      const pagina = panel.pagina + (accion === "siguiente" ? 1 : accion === "anterior" ? -1 : 0);
      if (pagina < 0 || pagina * TAMANO >= panel.datos.solicitudes.length || pagina === panel.pagina) return;
      panel.pagina = pagina; pintar(bloque);
      anunciar(t("cronos_pagina_anuncio", { pagina: numero(pagina + 1, localizacion) }), "informacion");
    } else if (accion === "siguiente" && panel.datos.siguiente_cursor) {
      void consultar(bloque, { cursor: panel.datos.siguiente_cursor, cursores: [...panel.cursores, panel.cursor] });
    } else if (accion === "anterior" && panel.cursores.length) {
      void consultar(bloque, { cursor: panel.cursores.at(-1), cursores: panel.cursores.slice(0, -1) });
    }
  };
  raiz.addEventListener("click", alClick);
  raiz.addEventListener("submit", alSubmit);
  raiz.addEventListener("change", alCambio);
  const desmontar = () => {
    if (!activa) return;
    activa = false;
    for (const panel of Object.values(estado)) purgar(panel);
    raiz.removeEventListener("click", alClick); raiz.removeEventListener("submit", alSubmit); raiz.removeEventListener("change", alCambio);
    raiz.replaceChildren();
  };
  registrarDesmontar?.(desmontar);
  for (const bloque of BLOQUES) if (activa && estado[bloque].visible) void consultar(bloque);
  return Object.freeze({ desmontar });
}
