import { icono } from "../comun/iconos-vec.js?v=20260925-aspecto-v1";
import { referenciaCopiableTraducida } from "./portal-justificante.js";
import { traducirReferencia } from "./portal-referencias-i18n.js";
import { LOCALIZACION_PORTAL, textoPortal, traducirPortal } from "./portal-i18n.js?v=20260926-i18n-v1";

export const RUTA_AVISOS_BOLSA = "/api/vec/bolsa/avisos";
export const ESQUEMA_AVISOS_BOLSA = "vec.bolsa.rrhh.avisos.v1";

const TIPOS = new Set(["salto_orden", "tres_anos", "solicitud_portal", "respuesta_portal", "encadenamiento", "no_incorporacion_revision"]);
// Conteos opcionales: portal del candidato, encadenamiento (Bolsa 000041) y
// no incorporaciones pendientes de revisión (Bolsa 000042).
const TIPOS_PORTAL = ["solicitud_portal", "respuesta_portal", "encadenamiento", "no_incorporacion_revision"];

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
      const mensajes = { 401: traducirPortal("txt_se_requiere_una_sesion_interna_autenticada"), 403: traducirPortal("txt_la_sesion_no_dispone_de_ambito_para_consultar_av"), 404: traducirPortal("txt_el_servicio_de_avisos_no_esta_disponible") };
      return { ok: false, status: respuesta.status, mensaje: mensajes[respuesta.status] || traducirPortal("txt_no_se_pudieron_consultar_los_avisos_http", { estado: respuesta.status }) };
    }
    return { ok: true, datos: validarAvisosBolsa(await respuesta.json()) };
  } catch (error) {
    return { ok: false, status: 0, mensaje: error instanceof Error ? error.message : traducirPortal("txt_error_de_comunicacion_con_los_avisos") };
  }
}

const FORMATO_NUMERO = new Intl.NumberFormat(LOCALIZACION_PORTAL);

function fechaVisible(valor) {
  const fecha = new Date(valor);
  return Number.isNaN(fecha.valueOf()) ? traducirPortal("txt_fecha_no_disponible") : new Intl.DateTimeFormat(LOCALIZACION_PORTAL, { dateStyle: "medium", timeStyle: "short" }).format(fecha);
}

// Por qué Bolsa no ha aplicado la no incorporación que publicó Contratación
// temporal; hasta que RRHH lo resuelva no se abre el siguiente llamamiento.
const REVISION_NO_INCORPORACION = Object.freeze({
  sin_aceptacion: traducirPortal("txt_bolsa_aun_no_tiene_registrada_la_aceptacion_de_e"),
  participacion_no_constituida: traducirPortal("txt_la_persona_no_figura_en_la_bolsa_constituida"),
  incorporacion_registrada: traducirPortal("txt_bolsa_ya_tiene_registrada_la_incorporacion_de_es"),
  consecuencia_no_admitida: traducirPortal("txt_la_consecuencia_indicada_no_esta_en_las_reglas_v"),
  segunda_persona_ausente: traducirPortal("txt_la_resolucion_no_consta_de_una_segunda_persona_d"),
  transicion_no_admitida: traducirPortal("txt_la_situacion_actual_de_la_persona_no_admite_ese"),
});

const SOLICITUDES_PORTAL = Object.freeze({ pausa: traducirPortal("txt_pausa_voluntaria"), reactivacion: traducirPortal("txt_reactivacion") });
const RESPUESTAS_PORTAL = Object.freeze({ acepta: traducirPortal("txt_acepta"), renuncia: traducirPortal("txt_renuncia"), renuncia_justificada: traducirPortal("txt_renuncia_justificada") });

// Las referencias de solicitud y justificante son opacas: se copian, no se leen.
function copiable(referencia, claveAria) {
  return referenciaCopiableTraducida(referencia, texto, traducirReferencia, traducirReferencia(claveAria));
}

function detalleAviso(aviso) {
  if (aviso.tipo === "solicitud_portal") {
    const hasta = aviso.detalle.pausa_hasta ? ` hasta ${texto(fechaVisible(aviso.detalle.pausa_hasta))}` : "";
    return `${texto(SOLICITUDES_PORTAL[aviso.detalle.solicitud] || traducirPortal("txt_solicitud"))}${hasta}. ${texto(traducirReferencia("aviso_solicitud_valida"))} ${copiable(aviso.referencia, "aviso_solicitud_copiar_aria")}`;
  }
  if (aviso.tipo === "respuesta_portal") {
    const causa = aviso.detalle.causa ? ` ${texto(traducirReferencia("aviso_causa", { causa: aviso.detalle.causa }))} ${copiable(aviso.detalle.justificante_ref, "aviso_justificante_copiar_aria")}` : "";
    const modo = aviso.detalle.modo === "propuesta_rrhh" ? traducirPortal("txt_pendiente_de_confirmar_por_rrhh") : traducirPortal("txt_respuesta_firme");
    return `${texto(RESPUESTAS_PORTAL[aviso.detalle.respuesta] || traducirPortal("txt_respuesta"))}. ${modo}${causa}`;
  }
  if (aviso.tipo === "salto_orden") {
    return textoPortal("txt_aviso_salto_orden", { orden: aviso.detalle.orden, primero: aviso.detalle.orden_primero_llamado });
  }
  if (aviso.tipo === "no_incorporacion_revision") {
    const causa = REVISION_NO_INCORPORACION[aviso.detalle.estado] || traducirPortal("txt_pendiente_de_revision");
    const notificada = aviso.detalle.fecha_notificacion ? ` ${textoPortal("txt_aviso_notificada_el", { fecha: new Intl.DateTimeFormat(LOCALIZACION_PORTAL, { dateStyle: "medium", timeZone: "UTC" }).format(new Date(`${aviso.detalle.fecha_notificacion}T00:00:00Z`)) })}` : "";
    return `${texto(causa)}${notificada} ${textoPortal("txt_aviso_siguiente_espera")}`;
  }
  if (aviso.tipo === "encadenamiento") {
    const d = aviso.detalle;
    return textoPortal("txt_aviso_encadenamiento", { dias: FORMATO_NUMERO.format(Number(d.dias_acumulados) || 0), ventana: d.ventana_meses, umbral: d.umbral_meses });
  }
  // Con Bolsa 000041 el plazo sale del catálogo; sin él, tres años.
  const plazo = Number.isSafeInteger(aviso.detalle.plazo_meses) ? traducirPortal("txt_n_meses", { numero: FORMATO_NUMERO.format(aviso.detalle.plazo_meses) }) : traducirPortal("txt_tres_anos");
  return textoPortal("txt_aviso_alcanza", { plazo, fecha: fechaVisible(aviso.detalle.alcanza_tres_anos_en) });
}

function filaAviso(aviso) {
  const titulo = { salto_orden: traducirPortal("txt_posible_salto_de_orden"), tres_anos: traducirPortal("txt_trabajo_continuado"), solicitud_portal: traducirPortal("txt_solicitud_desde_mi_bolsa"), respuesta_portal: traducirPortal("txt_respuesta_desde_mi_bolsa"), encadenamiento: traducirPortal("txt_encadenamiento_de_contratos"), no_incorporacion_revision: traducirPortal("txt_no_incorporacion_pendiente_de_revision") }[aviso.tipo];
  const participacion = aviso.detalle.participacion_ref;
  const enlace = referenciaOpaca(participacion)
    ? `<button type="button" class="boton-enlace" data-accion="abrir-ficha-b5" data-bolsa-ref="${texto(aviso.bolsa)}" data-participacion-ref="${texto(participacion)}">${textoPortal("txt_abrir_ficha")}</button>`
    : "";
  return `<li class="lista-actividad__item" data-tipo-aviso="${texto(aviso.tipo)}"><div><strong>${titulo}</strong><p>${detalleAviso(aviso)}</p><small>${texto(fechaVisible(aviso.fecha))}</small></div>${enlace}</li>`;
}

export function renderizarBloqueAvisos({ estado = "cargando", datos = null, error = "" } = {}) {
  const conteos = datos?.conteos && datos.items.length > 0
    ? `<span class="estado-chip advertencia">${textoPortal("txt_n_saltos_de_orden", { numero: datos.conteos.salto_orden })}</span><span class="estado-chip info">${textoPortal("txt_n_tres_anos", { numero: datos.conteos.tres_anos })}</span>${numeroNatural(datos.conteos.solicitud_portal) ? `<span class="estado-chip advertencia">${textoPortal("txt_n_solicitudes_del_portal", { numero: datos.conteos.solicitud_portal })}</span>` : ""}${numeroNatural(datos.conteos.respuesta_portal) ? `<span class="estado-chip info">${textoPortal("txt_n_respuestas_del_portal", { numero: datos.conteos.respuesta_portal })}</span>` : ""}${numeroNatural(datos.conteos.encadenamiento) ? `<span class="estado-chip advertencia">${textoPortal("txt_n_encadenamientos", { numero: datos.conteos.encadenamiento })}</span>` : ""}${datos.conteos.no_incorporacion_revision > 0 ? `<span class="estado-chip advertencia">${textoPortal("txt_n_no_incorporaciones_por_revisar", { numero: datos.conteos.no_incorporacion_revision })}</span>` : ""}`
    : "";
  // El cómputo legal aún pendiente de RRHH no se rotula en la pantalla de trabajo:
  // la explicación vive en la ayuda («?») y el origen de la regla en «Reglas vigentes».
  const pendiente = "";
  const cabecera = `<div class="cabecera-panel"><h2>${textoPortal("txt_avisos")}</h2>${conteos || pendiente ? `<div class="avisos-bolsa-conteos" aria-label="${textoPortal("txt_avisos_por_tipo")}">${conteos}${pendiente}</div>` : ""}</div>`;
  if (estado === "cargando") return `<section class="panel avisos-bolsa" aria-busy="true" aria-live="polite">${cabecera}<div class="cuerpo-panel avisos-bolsa-vacio" role="status">${textoPortal("txt_cargando_avisos")}</div></section>`;
  if (estado === "error") return `<section class="panel avisos-bolsa" aria-live="assertive">${cabecera}<div class="cuerpo-panel aviso aviso--error"><span>${texto(error || traducirPortal("txt_no_se_pudieron_cargar_los_avisos"))}</span><button type="button" data-accion="reintentar-avisos">${textoPortal("txt_reintentar")}</button></div></section>`;
  if (!datos || datos.items.length === 0) return `<section class="panel avisos-bolsa" aria-live="polite">${cabecera}<div class="cuerpo-panel avisos-bolsa-vacio"><span class="avisos-bolsa-icono" aria-hidden="true">${icono("correcto")}</span><span><strong>${textoPortal("txt_sin_avisos")}</strong> ${textoPortal("txt_no_hay_saltos_de_orden_periodos_de_trabajo_conti")}</span></div></section>`;
  const paginacion = `<footer class="paginacion"><span>${textoPortal("txt_mostrando_desde_hasta_total", { desde: datos.paginacion.desde, hasta: datos.paginacion.hasta, total: datos.paginacion.total })}</span><button type="button" data-accion="siguiente-avisos"${datos.paginacion.cursor_siguiente ? "" : " disabled"}>${textoPortal("txt_siguiente")}</button></footer>`;
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
