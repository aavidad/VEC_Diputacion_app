import { icono } from "../comun/iconos-vec.js?v=20260925-aspecto-v1";
import { referenciaCopiableTraducida } from "./portal-justificante.js";
import { traducirReferencia } from "./portal-referencias-i18n.js";

export const RUTA_AVISOS_BOLSA = "/api/vec/bolsa/avisos";
export const ESQUEMA_AVISOS_BOLSA = "vec.bolsa.rrhh.avisos.v1";

const TIPOS = new Set(["salto_orden", "tres_anos", "solicitud_portal", "respuesta_portal", "encadenamiento"]);
// Conteos opcionales: portal del candidato y encadenamiento (Bolsa 000041).
const TIPOS_PORTAL = ["solicitud_portal", "respuesta_portal", "encadenamiento"];

function texto(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}

function referenciaOpaca(valor) {
  return typeof valor === "string" && /^[A-Za-z0-9][A-Za-z0-9:._/-]{2,255}$/.test(valor);
}

function numeroNatural(valor) {
  return Number.isSafeInteger(valor) && valor >= 0;
}

export function validarAvisosBolsa(sobre) {
  const datos = sobre?.data;
  if (!datos || datos.esquema !== ESQUEMA_AVISOS_BOLSA || !Array.isArray(datos.items) ||
      !datos.conteos || !datos.paginacion || typeof datos.provisionalidad !== "string" ||
      !numeroNatural(datos.conteos.salto_orden) || !numeroNatural(datos.conteos.tres_anos) ||
      TIPOS_PORTAL.some((tipo) => datos.conteos[tipo] !== undefined && !numeroNatural(datos.conteos[tipo])) ||
      !numeroNatural(datos.paginacion.desde) || !numeroNatural(datos.paginacion.hasta) || !numeroNatural(datos.paginacion.total)) {
    throw new TypeError("Contrato de avisos de Bolsa no válido.");
  }
  for (const aviso of datos.items) {
    if (!TIPOS.has(aviso?.tipo) || !referenciaOpaca(aviso.bolsa) || !referenciaOpaca(aviso.referencia) ||
        !aviso.detalle || typeof aviso.detalle !== "object" || Number.isNaN(Date.parse(aviso.fecha))) {
      throw new TypeError("Aviso de Bolsa no válido.");
    }
    const participacion = aviso.detalle.participacion_ref;
    if (participacion !== undefined && !referenciaOpaca(participacion)) throw new TypeError("Participación de aviso no válida.");
  }
  return datos;
}

export async function consultarAvisosBolsa({ cursor = "", limite = 6, fetchImpl = fetch, signal } = {}) {
  const parametros = new URLSearchParams({ limite: String(limite) });
  if (cursor) parametros.set("cursor", cursor);
  try {
    const respuesta = await fetchImpl(`${RUTA_AVISOS_BOLSA}?${parametros}`, { method: "GET", credentials: "same-origin", signal, headers: { Accept: "application/json" } });
    if (!respuesta.ok) {
      const mensajes = { 401: "Se requiere una sesión interna autenticada.", 403: "La sesión no dispone de ámbito para consultar avisos.", 404: "El servicio de avisos no está disponible." };
      return { ok: false, status: respuesta.status, mensaje: mensajes[respuesta.status] || `No se pudieron consultar los avisos (HTTP ${respuesta.status}).` };
    }
    return { ok: true, datos: validarAvisosBolsa(await respuesta.json()) };
  } catch (error) {
    return { ok: false, status: 0, mensaje: error instanceof Error ? error.message : "Error de comunicación con los avisos." };
  }
}

const FORMATO_NUMERO = new Intl.NumberFormat("es-ES");

function fechaVisible(valor) {
  const fecha = new Date(valor);
  return Number.isNaN(fecha.valueOf()) ? "Fecha no disponible" : new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", timeStyle: "short" }).format(fecha);
}

const SOLICITUDES_PORTAL = Object.freeze({ pausa: "Pausa voluntaria", reactivacion: "Reactivación" });
const RESPUESTAS_PORTAL = Object.freeze({ acepta: "Acepta", renuncia: "Renuncia", renuncia_justificada: "Renuncia justificada" });

// Las referencias de solicitud y justificante son opacas: se copian, no se leen.
function copiable(referencia, claveAria) {
  return referenciaCopiableTraducida(referencia, texto, traducirReferencia, traducirReferencia(claveAria));
}

function detalleAviso(aviso) {
  if (aviso.tipo === "solicitud_portal") {
    const hasta = aviso.detalle.pausa_hasta ? ` hasta ${texto(fechaVisible(aviso.detalle.pausa_hasta))}` : "";
    return `${texto(SOLICITUDES_PORTAL[aviso.detalle.solicitud] || "Solicitud")}${hasta}. ${texto(traducirReferencia("aviso_solicitud_valida"))} ${copiable(aviso.referencia, "aviso_solicitud_copiar_aria")}`;
  }
  if (aviso.tipo === "respuesta_portal") {
    const causa = aviso.detalle.causa ? ` ${texto(traducirReferencia("aviso_causa", { causa: aviso.detalle.causa }))} ${copiable(aviso.detalle.justificante_ref, "aviso_justificante_copiar_aria")}` : "";
    const modo = aviso.detalle.modo === "propuesta_rrhh" ? "Pendiente de confirmar por RRHH." : "Respuesta firme.";
    return `${texto(RESPUESTAS_PORTAL[aviso.detalle.respuesta] || "Respuesta")}. ${modo}${causa}`;
  }
  if (aviso.tipo === "salto_orden") {
    return `Orden ${texto(aviso.detalle.orden)}; primera persona llamada: ${texto(aviso.detalle.orden_primero_llamado)}.`;
  }
  if (aviso.tipo === "encadenamiento") {
    const d = aviso.detalle;
    return `${texto(FORMATO_NUMERO.format(Number(d.dias_acumulados) || 0))} días con contrato en los últimos ${texto(d.ventana_meses)} meses (umbral: ${texto(d.umbral_meses)} meses).`;
  }
  // Con Bolsa 000041 el plazo sale del catálogo; sin él, tres años.
  const plazo = Number.isSafeInteger(aviso.detalle.plazo_meses) ? `${FORMATO_NUMERO.format(aviso.detalle.plazo_meses)} meses` : "tres años";
  return `Alcanza ${texto(plazo)}: ${texto(fechaVisible(aviso.detalle.alcanza_tres_anos_en))}.`;
}

function filaAviso(aviso) {
  const titulo = { salto_orden: "Posible salto de orden", tres_anos: "Trabajo continuado", solicitud_portal: "Solicitud desde «Mi bolsa»", respuesta_portal: "Respuesta desde «Mi bolsa»", encadenamiento: "Encadenamiento de contratos" }[aviso.tipo];
  const participacion = aviso.detalle.participacion_ref;
  const enlace = referenciaOpaca(participacion)
    ? `<button type="button" class="boton-enlace" data-accion="abrir-ficha-b5" data-bolsa-ref="${texto(aviso.bolsa)}" data-participacion-ref="${texto(participacion)}">Abrir ficha</button>`
    : "";
  return `<li class="lista-actividad__item" data-tipo-aviso="${texto(aviso.tipo)}"><div><strong>${titulo}</strong><p>${detalleAviso(aviso)}</p><small>${texto(fechaVisible(aviso.fecha))}</small></div>${enlace}</li>`;
}

export function renderizarBloqueAvisos({ estado = "cargando", datos = null, error = "" } = {}) {
  const conteos = datos?.conteos && datos.items.length > 0
    ? `<span class="estado-chip advertencia">${datos.conteos.salto_orden} saltos de orden</span><span class="estado-chip info">${datos.conteos.tres_anos} tres años</span>${numeroNatural(datos.conteos.solicitud_portal) ? `<span class="estado-chip advertencia">${datos.conteos.solicitud_portal} solicitudes del portal</span>` : ""}${numeroNatural(datos.conteos.respuesta_portal) ? `<span class="estado-chip info">${datos.conteos.respuesta_portal} respuestas del portal</span>` : ""}${numeroNatural(datos.conteos.encadenamiento) ? `<span class="estado-chip advertencia">${datos.conteos.encadenamiento} encadenamientos</span>` : ""}`
    : "";
  // El cómputo legal aún pendiente de RRHH no se rotula en la pantalla de trabajo:
  // la explicación vive en la ayuda («?») y el origen de la regla en «Reglas vigentes».
  const pendiente = "";
  const cabecera = `<div class="cabecera-panel"><h2>Avisos</h2>${conteos || pendiente ? `<div class="avisos-bolsa-conteos" aria-label="Avisos por tipo">${conteos}${pendiente}</div>` : ""}</div>`;
  if (estado === "cargando") return `<section class="panel avisos-bolsa" aria-busy="true" aria-live="polite">${cabecera}<div class="cuerpo-panel avisos-bolsa-vacio" role="status">Cargando avisos…</div></section>`;
  if (estado === "error") return `<section class="panel avisos-bolsa" aria-live="assertive">${cabecera}<div class="cuerpo-panel aviso aviso--error"><span>${texto(error || "No se pudieron cargar los avisos.")}</span><button type="button" data-accion="reintentar-avisos">Reintentar</button></div></section>`;
  if (!datos || datos.items.length === 0) return `<section class="panel avisos-bolsa" aria-live="polite">${cabecera}<div class="cuerpo-panel avisos-bolsa-vacio"><span class="avisos-bolsa-icono" aria-hidden="true">${icono("correcto")}</span><span><strong>Sin avisos.</strong> No hay saltos de orden, periodos de trabajo continuado, encadenamientos ni solicitudes o respuestas del portal.</span></div></section>`;
  const paginacion = `<footer class="paginacion"><span>Mostrando ${datos.paginacion.desde} a ${datos.paginacion.hasta} de ${datos.paginacion.total}</span><button type="button" data-accion="siguiente-avisos"${datos.paginacion.cursor_siguiente ? "" : " disabled"}>Siguiente</button></footer>`;
  return `<section class="panel avisos-bolsa" aria-live="polite">${cabecera}<div class="tabla-contenedor avisos-bolsa-lista" tabindex="0"><ul class="lista-actividad">${datos.items.map(filaAviso).join("")}</ul></div>${paginacion}</section>`;
}

// El montaje P-WEB-10 puede delegar aquí sin conocer el contrato: la ficha de
// participación sigue siendo la única dueña de la apertura y el foco del detalle inline.
export function manejarAccionAvisos(evento, { abrirFichaB5, siguiente, reintentar } = {}) {
  const control = evento?.target?.closest?.("[data-accion]");
  if (!control) return false;
  const accion = control.dataset.accion;
  if (accion === "abrir-ficha-b5" && typeof abrirFichaB5 === "function") abrirFichaB5(control.dataset.bolsaRef, control.dataset.participacionRef, control);
  else if (accion === "siguiente-avisos" && typeof siguiente === "function" && !control.disabled) siguiente();
  else if (accion === "reintentar-avisos" && typeof reintentar === "function") reintentar();
  else return false;
  return true;
}
