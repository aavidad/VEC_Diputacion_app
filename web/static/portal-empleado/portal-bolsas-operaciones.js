import { causasBaja, consultarReglasSituacion, hoyCivil, instalarPropuestaReposicion, motivoConCausa, renderizarCausasBaja } from "./portal-bolsas-reglas-situacion.js?v=20260930-portales-i18n-integracion-v1";
import { traducirReglasSituacion } from "./portal-bolsas-reglas-situacion-i18n.js?v=20260930-portales-i18n-integracion-v1";
import { cargarContratosFicha, manejarClickContratos } from "./portal-bolsas-contratos.js?v=20261007-pantallas-textos-final-v1";
import { cargarReincorporacionesTitularFicha, manejarClickReincorporacionesTitular } from "./portal-bolsas-reincorporaciones.js?v=20261007-pantallas-textos-final-v1";
import { renderizarTrazaValores, validarCambiosTraza } from "./portal-bolsas-traza-valores.js?v=20261007-pantallas-textos-final-v1";
import { LOCALIZACION_PORTAL, textoPortal, traducirPortal, ZONA_HORARIA_PORTAL } from "./portal-i18n.js?v=20261007-pantallas-textos-final-v1";
import { actorTraducido, justificanteTraducido } from "./portal-justificante.js";
import { traducirReferencia } from "./portal-referencias-i18n.js?v=20261007-pantallas-textos-final-v1";
import { ayudaHuellaArchivo, instalarHuellaArchivo, renderizarCampoHuellaArchivo, traducirHuellaArchivo } from "./portal-huella-archivo.js";

const BASE = "/api/vec/bolsa/bolsas";
const TIPOS_JUSTIFICANTE = Object.freeze(["solicitud_candidato", "informe_medico", "resolucion", "correo", "acta_bolsa", "otro"]);
const HEX_SHA256 = /^[a-f0-9]{64}$/;
const INSTANTE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?(?:Z|[+-]\d{2}:\d{2})$/;
function hoyCausa() {
  const partes = new Intl.DateTimeFormat(LOCALIZACION_PORTAL, { timeZone: ZONA_HORARIA_PORTAL, year: "numeric", month: "2-digit", day: "2-digit" }).formatToParts(new Date());
  const valores = Object.fromEntries(partes.map(({ type, value }) => [type, value]));
  return `${valores.year}-${valores.month}-${valores.day}`;
}
function fechaCausaValida(valor) {
  return typeof valor === "string" && /^\d{4}-\d{2}-\d{2}$/.test(valor)
    && Number.isFinite(Date.parse(valor)) && new Date(valor).toISOString().slice(0, 10) === valor && valor <= hoyCausa();
}
function instanteValido(valor) {
  return typeof valor === "string" && INSTANTE.test(valor) && Number.isFinite(Date.parse(valor));
}
// Espejo de patronDocumentoIdentidadEnReferencia y patronEtiquetaDocumentoIdentidad
// del dominio: evita enviar referencias que la API rechaza con HTTP 400.
const DOCUMENTO_IDENTIDAD = /((?:[0-9][._:/#-]?){8}|[XYZ][._:/#-]?(?:[0-9][._:/#-]?){7})[A-Z]/i;
const ETIQUETA_DOCUMENTO_IDENTIDAD = /(^|[._:/#-])(dni|nie|nif|pasaporte|passport)([._:/#-]|$)/i;
// Espejo de ReferenciaPropiaSistema del dominio de Bolsa: las referencias que
// emite el sistema (espacio de nombres alfabético y huella SHA-256 en
// hexadecimal) no pueden llevar un documento escrito por una persona, pero sus
// cifras casan por azar con los patrones de DNI o teléfono.
const REFERENCIA_PROPIA_SISTEMA = /^[a-z_]+(?::[a-z_]+)*:[0-9a-f]{64}$/;
const MENSAJE_REFERENCIA_IDENTIDAD = traducirPortal("txt_la_referencia_no_puede_contener_un_dni_o_nie_use");

export function referenciaContieneDocumentoIdentidad(referencia) {
  return typeof referencia === "string" && ((!REFERENCIA_PROPIA_SISTEMA.test(referencia) && DOCUMENTO_IDENTIDAD.test(referencia)) || ETIQUETA_DOCUMENTO_IDENTIDAD.test(referencia));
}

function segmento(valor) {
  return encodeURIComponent(String(valor ?? "").trim()).replace(/%3A/gi, ":");
}

export function rutaOperacionesSituacion(bolsa, participacion) {
  return `${BASE}/${segmento(bolsa)}/candidatos/${segmento(participacion)}/operaciones`;
}

function respuestaInvalida(mensaje) {
  return { ok: false, status: 0, codigo: "respuesta_invalida", mensaje };
}

export async function consultarOperacionesSituacion(bolsa, participacion, { fetchImpl = fetch, signal } = {}) {
  try {
    const respuesta = await fetchImpl(rutaOperacionesSituacion(bolsa, participacion), {
      method: "GET", credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", signal, headers: { Accept: "application/json" },
    });
    if (!respuesta.ok) {
      const cuerpoError = await respuesta.json().catch(() => ({}));
      if (respuesta.status === 403) return { ...errorHttp(403, cuerpoError?.error?.codigo),
        mensaje: traducirPortal("txt_acceso_denegado") };
      return errorHttp(respuesta.status, cuerpoError?.error?.codigo);
    }
    const cuerpo = await respuesta.json();
    if (cuerpo?.data?.esquema !== "vec.bolsa.rrhh.operaciones_situacion.v1" || !Array.isArray(cuerpo.data.items)) {
      return respuestaInvalida(traducirPortal("txt_la_respuesta_del_historial_de_operaciones_no_res"));
    }
    const items = cuerpo.data.items;
    const valido = items.every((item) => item && typeof item === "object"
      && ["desde", "operacion", "situacion", "motivo", "actor", "validador", "validada_en"].every((campo) => typeof item[campo] === "string")
      && item.justificante && TIPOS_JUSTIFICANTE.includes(item.justificante.tipo)
      && typeof item.justificante.referencia === "string" && HEX_SHA256.test(item.justificante.sha256));
    const cambios = validarCambiosTraza(cuerpo.data.cambios);
    const vigente = cuerpo.data.situacion_vigente;
    const vigenteValida = vigente === undefined || (vigente && typeof vigente.situacion === "string" && instanteValido(vigente.desde));
    return valido && cambios && vigenteValida ? { ok: true, datos: items, cambios, situacionVigente: vigente ?? null } : respuestaInvalida(traducirPortal("txt_un_registro_del_historial_no_respeta_su_contrato"));
  } catch (error) {
    return { ok: false, status: 0, codigo: "error_red", mensaje: traducirPortal("txt_no_se_pudo_comunicar_con_el_historial_de_operaci") };
  }
}

export async function consultarSolicitudesDocumentalesRRHH(bolsa, participacion, { fetchImpl = fetch, signal } = {}) {
  const query = new URLSearchParams({ bolsa_ref: bolsa, participacion_ref: participacion });
  try {
    const respuesta = await fetchImpl(`/api/vec/bolsa/solicitudes-documentales/pendientes?${query}`, {
      method: "GET", credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", signal,
      headers: { Accept: "application/json" },
    });
    if (!respuesta.ok) {
      const clave = ({ 401: "txt_se_requiere_una_sesion_interna_autenticada",
        403: "txt_acceso_denegado", 404: "txt_operacion_no_disponible_todavia" })[respuesta.status]
        || "txt_b8_solicitudes_documentales_error";
      return { ok: false, status: respuesta.status, mensaje: traducirPortal(clave) };
    }
    const cuerpo = await respuesta.json();
    const items = cuerpo?.data?.items;
    const valido = cuerpo?.data?.esquema === "vec.bolsa.rrhh.solicitudes_documentales.v1" && Array.isArray(items)
      && items.every((item) => /^solicitud-documental:[a-f0-9]{64}$/.test(item?.solicitud_ref || "")
        && Number.isSafeInteger(item.version) && item.version > 0 && HEX_SHA256.test(item.contenido_sha256)
        && typeof item.documento_ref === "string" && item.documento_ref.length > 0 && item.documento_ref.length <= 256
        && !referenciaContieneDocumentoIdentidad(item.documento_ref) && HEX_SHA256.test(item.documento_sha256)
        && (item.fecha_fin_causa === null || (/^\d{4}-\d{2}-\d{2}$/.test(item.fecha_fin_causa) && Number.isFinite(Date.parse(item.fecha_fin_causa))))
        && item.estado === "pendiente_rrhh" && typeof item.recibo_ref === "string" && item.recibo_ref.length > 0
        && instanteValido(item.registrada_en));
    return valido ? { ok: true, datos: items } : respuestaInvalida(traducirPortal("txt_b8_solicitudes_documentales_error"));
  } catch {
    return { ok: false, status: 0, mensaje: traducirPortal("txt_b8_solicitudes_documentales_error") };
  }
}

function errorHttp(status, codigoServidor = "") {
  const errores = {
    400: ["solicitud_invalida", traducirPortal("txt_la_solicitud_no_es_valida_revise_los_campos_del")],
    401: ["no_autenticado", traducirPortal("txt_se_requiere_una_sesion_interna_autenticada")],
    403: ["acceso_denegado", traducirPortal("txt_la_sesion_no_dispone_de_permiso_para_registrar_e")],
    404: ["recurso_no_encontrado", traducirPortal("txt_operacion_no_disponible_todavia")],
    409: codigoServidor === "clave_reutilizada"
      ? ["clave_reutilizada", traducirPortal("txt_la_clave_de_idempotencia_ya_se_uso_con_otros_dat")]
      : ["transicion_no_valida", traducirPortal("txt_la_operacion_no_puede_aplicarse_a_la_situacion_v")],
    503: ["servicio_no_disponible", traducirPortal("txt_el_servicio_no_esta_disponible_ahora_puede_reint")],
  };
  const [codigo, mensaje] = errores[status] || ["error_servidor", traducirPortal("txt_no_se_pudo_completar_la_operacion_http", { estado: status })];
  return { ok: false, status, codigo, mensaje };
}

export async function registrarOperacionSituacion(bolsa, participacion, comando, clave, { fetchImpl = fetch } = {}) {
  const vinculada = Boolean(comando?.solicitud_ref || comando?.solicitud_version_esperada || comando?.solicitud_contenido_sha256);
  if (referenciaContieneDocumentoIdentidad(comando?.justificante?.referencia)) {
    return { ok: false, status: 400, codigo: "referencia_identidad", mensaje: MENSAJE_REFERENCIA_IDENTIDAD };
  }
  if (!bolsa || !participacion || !comando || !clave
    || !["revisar", "regularizar", "excluir"].includes(comando.operacion)
    || (["revisar", "regularizar"].includes(comando.operacion) && !instanteValido(comando.situacion_esperada_desde))
    || (comando.operacion === "excluir" && comando.situacion_esperada_desde !== undefined && !instanteValido(comando.situacion_esperada_desde))
    || (comando.operacion === "regularizar" && !fechaCausaValida(comando.causa_finalizada_en))
    || (vinculada && (comando.operacion !== "regularizar" || comando.justificante?.tipo !== "solicitud_candidato"
      || !/^solicitud-documental:[a-f0-9]{64}$/.test(comando.solicitud_ref || "")
      || !Number.isSafeInteger(comando.solicitud_version_esperada) || comando.solicitud_version_esperada < 1
      || !HEX_SHA256.test(comando.solicitud_contenido_sha256 || "")))
    || typeof comando.motivo !== "string" || !comando.motivo.trim()
    || typeof comando.validador !== "string" || !comando.validador.trim()
    || !TIPOS_JUSTIFICANTE.includes(comando.justificante?.tipo)
    || typeof comando.justificante?.referencia !== "string" || !comando.justificante.referencia.trim()
    || !HEX_SHA256.test(comando.justificante?.sha256 || "")) {
    return { ok: false, ...errorHttp(400) };
  }
  try {
    const respuesta = await fetchImpl(rutaOperacionesSituacion(bolsa, participacion), {
      method: "POST", credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error",
      headers: { Accept: "application/json", "Content-Type": "application/json", "Idempotency-Key": clave },
      body: JSON.stringify(comando),
    });
    const cuerpo = await respuesta.json().catch(() => ({}));
    if ((respuesta.status === 200 || respuesta.status === 201) && respuesta.ok
      && typeof cuerpo?.data?.recibo_ref === "string" && cuerpo.data.recibo_ref.trim()
      && cuerpo.data.situacion === DESTINO_OPERACION[comando.operacion]
      && instanteValido(cuerpo.data.desde) && typeof cuerpo.data.reutilizada === "boolean"
      && (!comando.solicitud_ref || (typeof cuerpo.data.recibo_resolucion_ref === "string" && cuerpo.data.recibo_resolucion_ref.trim()
        && instanteValido(cuerpo.data.resuelta_en)))) {
      return { ok: true, datos: cuerpo.data };
    }
    if (respuesta.ok) return respuestaInvalida(traducirPortal("txt_la_respuesta_de_la_operacion_no_contiene_un_reci"));
    return errorHttp(respuesta.status, cuerpo?.error?.codigo);
  } catch (error) {
    return { ok: false, status: 0, codigo: "error_red", mensaje: traducirPortal("txt_no_se_pudo_comunicar_con_el_servicio_puede_reint") };
  }
}

function html(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}

// Las etiquetas históricas permanecen para leer operaciones ya registradas.
const OPERACIONES = Object.freeze({ pausar: traducirPortal("txt_pausar"), reactivar: traducirPortal("txt_reactivar"), revisar: traducirPortal("txt_b8_revisar"), regularizar: traducirPortal("txt_b8_regularizar"), excluir: traducirPortal("txt_excluir") });
const TIPOS_ETIQUETA = Object.freeze({ solicitud_candidato: traducirPortal("txt_solicitud_del_candidato"), informe_medico: traducirPortal("txt_informe_medico"), resolucion: traducirPortal("txt_resolucion"), correo: traducirPortal("txt_correo"), acta_bolsa: traducirPortal("txt_acta_de_bolsa"), otro: traducirPortal("txt_otro") });

// Situación que produce cada operación (espejo de destinoOperacionSituacion).
const DESTINO_OPERACION = Object.freeze({ revisar: "en_revision", regularizar: "disponible", excluir: "excluido" });

/**
 * RRHH18: ninguna suspensión ni reactivación genérica. Revisión y
 * regularización requieren una transición del catálogo vigente. La guarda
 * de sanción viva y la validación del acto de RRHH corresponden al servidor.
 */
export function operacionesDisponibles(estadoClave, transiciones) {
  const base = estadoClave === "renuncia" ? ["revisar", "regularizar", "excluir"]
    : estadoClave === "no_disponible" ? ["revisar", "excluir"]
    : estadoClave === "en_revision" ? ["regularizar", "excluir"]
      : estadoClave === "excluido" ? ["regularizar"] : ["excluir"];
  const destinos = transiciones?.[estadoClave];
  if (!Array.isArray(destinos)) return estadoClave === "excluido" ? [] : ["excluir"];
  return base.filter((operacion) => destinos.includes(DESTINO_OPERACION[operacion])
    && (operacion === "excluir" || Array.isArray(transiciones.en_revision)));
}

function operacionesAdmitidasCandidato(candidato, estado) {
  if (estado.carga !== "listo" || estado.noDisponible) return [];
  return operacionesDisponibles(candidato.estado_clave, estado.transiciones)
    .filter((operacion) => (operacion === "excluir" && candidato.estado_clave !== "en_revision") || instanteValido(candidato.estado_desde))
    .filter((operacion) => {
      if (operacion !== "revisar" || candidato.estado_clave !== "no_disponible") return true;
      const antecedente = estado.items?.find((item) => item.desde === candidato.estado_desde
        && item.operacion === "pausar" && item.situacion === "no_disponible" && typeof item.recibo_ref === "string");
      return Boolean(antecedente && estado.cambios?.some((cambio) => cambio.campo === "situacion"
        && cambio.recibo_ref === antecedente.recibo_ref && cambio.valor_anterior === "renuncia"
        && cambio.valor_nuevo === "no_disponible"));
    });
}

function etiquetaOperacion(operacion, candidato) {
  if (operacion === "regularizar" && candidato.estado_clave === "excluido") return traducirPortal("txt_b8_reincorporar");
  return OPERACIONES[operacion];
}

function justificanteVisible(item) {
  if (item.justificante.tipo === "solicitud_candidato" && /^solicitud-documental:[a-f0-9]{64}$/.test(item.justificante.referencia)) {
    return traducirPortal("txt_b8_solicitud_documental");
  }
  return `${TIPOS_ETIQUETA[item.justificante.tipo] || item.justificante.tipo} · ${item.justificante.referencia}`;
}

export function renderizarOperacionesSituacion({ candidato, estado = {}, escaparHTML = html }) {
  const actual = estado.carga || "cargando";
  const disponibles = operacionesAdmitidasCandidato(candidato, estado);
  const solicitudes = estado.solicitudesDocumentales || [];
  const acciones = disponibles.filter((operacion) => operacion !== "regularizar" || (!solicitudes.length && !estado.solicitudesError))
    .map((operacion) => `<button type="button" class="boton-secundario" data-b8-accion="seleccionar" data-operacion="${operacion}">${escaparHTML(etiquetaOperacion(operacion, candidato))}</button>`).join("");
  const solicitudesReintentables = ![401, 403, 404].includes(estado.solicitudesStatus);
  const solicitudesVista = actual === "listo" && !(estado.paso > 0) ? `${estado.solicitudesCargando
    ? `<p role="status" aria-busy="true">${textoPortal("txt_comprobando_acceso")}</p>`
    : estado.solicitudesError ? `<p class="mensaje-error" role="alert">${escaparHTML(estado.solicitudesError)}</p>${solicitudesReintentables
      ? `<button type="button" class="boton-secundario" data-b8-accion="reintentar-solicitudes">${textoPortal("txt_reintentar_historial")}</button>` : ""}` : ""}${solicitudes.map((solicitud) => `<div class="panel-separado"><p>${textoPortal("txt_b8_solicitud_pendiente", { fecha: instanteLegible(solicitud.registrada_en) })}</p><button type="button" class="boton-secundario" data-b8-accion="seleccionar-solicitud" data-solicitud-ref="${escaparHTML(solicitud.solicitud_ref)}" ${disponibles.includes("regularizar") && (solicitud.fecha_fin_causa === null || fechaCausaValida(solicitud.fecha_fin_causa)) ? "" : "disabled"}>${textoPortal("txt_b8_validar_solicitud")}</button></div>`).join("")}` : "";
  const botones = estado.paso > 0 ? `<button type="button" class="boton-secundario" data-b8-accion="cancelar" ${estado.enviando ? "disabled" : ""}>${textoPortal("txt_cancelar")}</button>`
    : estado.noDisponible ? "" : acciones;
  let contenido = "";
  if (actual === "cargando") contenido = '<p class="vacio-controlado" role="status" aria-busy="true">' + textoPortal("txt_cargando_historial_de_operaciones") + '</p>';
  else if (actual === "error") contenido = `<p class="mensaje-error" role="alert">${escaparHTML(estado.error || traducirPortal("txt_no_se_pudo_consultar_el_historial"))}</p>${[401, 403, 404].includes(estado.errorStatus)
    ? "" : `<button type="button" class="boton-secundario" data-b8-accion="reintentar">${textoPortal("txt_reintentar_historial")}</button>`}`;
  else if (!estado.items?.length) contenido = '<p class="vacio-controlado" role="status">' + textoPortal("txt_no_hay_operaciones_registradas") + '</p>';
  else {
    const total = estado.items.length;
    const paginas = Math.max(1, Math.ceil(total / 6));
    const pagina = Math.min(Math.max(0, Number(estado.paginaHistorial) || 0), paginas - 1);
    const visibles = estado.items.slice(pagina * 6, (pagina + 1) * 6);
    contenido = `<div class="tabla-contenedor" tabindex="0" role="region" aria-label="${textoPortal("txt_historial_de_operaciones")}"><table class="tabla-datos"><caption>${textoPortal("txt_historial_de_operaciones")}</caption><thead><tr><th>${textoPortal("txt_desde")}</th><th>${textoPortal("txt_operacion")}</th><th>${textoPortal("txt_situacion")}</th><th>${textoPortal("txt_motivo")}</th><th>${textoPortal("txt_justificante")}</th><th>${textoPortal("txt_actor_validador")}</th></tr></thead><tbody>${visibles.map((item) => `<tr><td>${escaparHTML(instanteLegible(item.desde))}</td><td>${escaparHTML(OPERACIONES[item.operacion] || item.operacion)}</td><td>${escaparHTML(situacionLegible(item.situacion))}</td><td>${escaparHTML(item.motivo)}</td><td>${escaparHTML(justificanteVisible(item))}</td><td>${personaLegible(item.actor, escaparHTML)} / ${personaLegible(item.validador, escaparHTML)}<br>${escaparHTML(instanteLegible(item.validada_en))}</td></tr>`).join("")}</tbody></table></div>${paginas > 1 ? `<nav class="paginacion-bolsa" aria-label="${textoPortal("txt_paginacion_del_historial_de_operaciones")}"><span>${textoPortal("txt_mostrando_desde_hasta_total", { desde: pagina * 6 + 1, hasta: Math.min((pagina + 1) * 6, total), total })}</span><button type="button" class="boton-secundario" data-b8-accion="pagina" data-pagina="${pagina - 1}" ${pagina === 0 ? "disabled" : ""}>${textoPortal("txt_anterior")}</button><button type="button" class="boton-secundario" data-b8-accion="pagina" data-pagina="${pagina + 1}" ${pagina + 1 >= paginas ? "disabled" : ""}>${textoPortal("txt_siguiente")}</button></nav>` : `<p>${textoPortal("txt_mostrando_desde_hasta_total", { desde: 1, hasta: total, total })}</p>`}`;
  }
  const flujo = estado.paso > 0 && disponibles.includes(estado.operacion) ? renderizarPaso(estado, escaparHTML, candidato) : "";
  const pendiente = actual === "listo" && ["renuncia", "en_revision", "excluido"].includes(candidato.estado_clave)
    && !disponibles.some((operacion) => ["revisar", "regularizar"].includes(operacion))
    ? `<p role="status">${textoPortal("txt_b8_regularizacion_no_disponible")}</p>` : "";
  return `<section class="panel panel-separado" data-b8-raiz="true"><div class="cabecera-panel"><div><h4>${textoPortal("txt_b8_gestion_estado")}</h4></div><details><summary aria-label="${escaparHTML(traducirHuellaArchivo("ayuda_aria"))}">?</summary><p>${escaparHTML(ayudaHuellaArchivo())}</p><p>${textoPortal("txt_b8_ayuda_llamamiento_directo")}</p></details></div><div class="cuerpo-panel"><div class="acciones-vista">${botones}</div>${solicitudesVista}${pendiente}${estado.recibo ? `<p class="mensaje-exito" role="status">${textoPortal("txt_operacion_registrada")} ${justificanteTraducido(estado.recibo, escaparHTML, (clave) => traducirPortal(`panel_${clave}`))}${estado.reutilizada ? traducirPortal("txt_respuesta_recuperada") : ""}</p>` : ""}${estado.reciboResolucion ? `<p class="mensaje-exito" role="status">${textoPortal("txt_b8_resolucion_solicitud")} ${justificanteTraducido(estado.reciboResolucion, escaparHTML, (clave) => traducirPortal(`panel_${clave}`))}</p>` : ""}${estado.errorOperacion ? `<p class="mensaje-error" role="alert">${escaparHTML(estado.errorOperacion)}</p>` : ""}${flujo}<h4>${textoPortal("txt_historial_de_operaciones")}</h4>${contenido}${actual === "listo" ? renderizarTrazaValores({ cambios: estado.cambios || [], pagina: estado.paginaTraza, escaparHTML }) : ""}</div></section>`;
}

// El historial llega con instantes ISO, claves de situación y referencias de
// identidad: se presentan como fecha local, situación traducida y papel.
function instanteLegible(valor) {
  const fecha = new Date(valor);
  if (typeof valor !== "string" || !/^\d{4}-\d{2}-\d{2}T/u.test(valor) || !Number.isFinite(fecha.getTime())) return String(valor ?? "");
  return new Intl.DateTimeFormat(LOCALIZACION_PORTAL, { dateStyle: "short", timeStyle: "short", timeZone: ZONA_HORARIA_PORTAL }).format(fecha);
}

function situacionLegible(clave) {
  if (clave === "en_revision") return traducirPortal("txt_b8_en_revision");
  const texto = String(clave ?? "").replaceAll("_", " ").trim();
  return texto ? texto.charAt(0).toLocaleUpperCase(LOCALIZACION_PORTAL) + texto.slice(1) : "—";
}

function personaLegible(valor, escaparHTML) {
  return actorTraducido(valor, escaparHTML, traducirReferencia);
}

function renderizarPaso(estado, escaparHTML, candidato) {
  const datos = estado.formulario || {};
  const operacion = estado.operacion;
  const etiqueta = etiquetaOperacion(operacion, candidato);
  const validarDisponibilidad = operacion === "regularizar";
  const exclusiones = operacion === "excluir" ? `<label><input type="checkbox" name="confirma_validador_distinto" required ${datos.confirma_validador_distinto ? "checked" : ""}> ${textoPortal("txt_confirmo_que_el_validador_es_otra_persona")}</label>` : "";
  const etapa = estado.paso;
  const causas = operacion === "excluir" ? estado.causasBaja || [] : [];
  const revision = etapa === 3 ? `<fieldset class="grupo-campo"><legend>${textoPortal("txt_revision")}</legend><dl class="resumen-expediente">
    <div class="fila-resumen"><dt>${textoPortal("txt_operacion_seleccionada")}</dt><dd>${escaparHTML(etiqueta || "")}</dd></div>
    <div class="fila-resumen"><dt>${textoPortal("txt_motivo")}</dt><dd>${escaparHTML(datos.motivo || "")}</dd></div>
    <div class="fila-resumen"><dt>${textoPortal("txt_tipo_de_justificante")}</dt><dd>${escaparHTML(TIPOS_ETIQUETA[datos.tipo] || "")}</dd></div>
    <div class="fila-resumen"><dt>${textoPortal("txt_referencia_del_documento_en_su_custodia")}</dt><dd>${escaparHTML(datos.referencia || "")}</dd></div>
    ${validarDisponibilidad ? `<div class="fila-resumen"><dt>${textoPortal("txt_b8_fecha_fin_causa")}</dt><dd>${escaparHTML(fechaCausaValida(datos.causa_finalizada_en) ? new Intl.DateTimeFormat(LOCALIZACION_PORTAL, { dateStyle: "short", timeZone: "UTC" }).format(new Date(datos.causa_finalizada_en)) : "")}</dd></div>` : ""}
  </dl></fieldset>` : "";
  const finCausa = validarDisponibilidad ? `<label>${textoPortal("txt_b8_fecha_fin_causa")} <input type="date" name="causa_finalizada_en" required max="${hoyCausa()}" value="${escaparHTML(datos.causa_finalizada_en || "")}"></label><p>${textoPortal("txt_b8_documento_fin_causa")}</p>` : "";
  const campos = etapa === 1 && causas.length
    ? renderizarCausasBaja({ causas, seleccion: datos.causa, detalle: datos.detalle, escaparHTML })
    : etapa === 1
    ? `<label>${textoPortal("txt_motivo")} <textarea name="motivo" required minlength="2" maxlength="1000">${escaparHTML(datos.motivo || "")}</textarea></label>`
    : etapa === 2
      ? `${finCausa}<label>${textoPortal("txt_tipo_de_justificante")} <select name="tipo" required><option value="">${textoPortal("txt_seleccione_un_tipo")}</option>${TIPOS_JUSTIFICANTE.map((tipo) => `<option value="${tipo}" ${datos.tipo === tipo ? "selected" : ""}>${TIPOS_ETIQUETA[tipo]}</option>`).join("")}</select></label><label>${textoPortal("txt_referencia_del_documento_en_su_custodia")} <input name="referencia" required minlength="2" maxlength="240" value="${escaparHTML(datos.referencia || "")}"></label>${renderizarCampoHuellaArchivo({ id: "b8-justificante-archivo", nombre: "sha256", huella: datos.sha256, escapar: escaparHTML })}`
        : `<label>${textoPortal(["revisar", "regularizar"].includes(operacion) ? "txt_b8_validador_rrhh" : "txt_persona_validadora")} <input name="validador" required minlength="2" maxlength="200" value="${escaparHTML(datos.validador || "")}"></label>${validarDisponibilidad ? `<label><input type="checkbox" name="confirma_fin_causa" required ${datos.confirma_fin_causa ? "checked" : ""}> ${textoPortal("txt_b8_confirma_fin_causa")}</label>` : ""}${exclusiones}`;
  return `<form data-b8-form="operacion" data-b8-paso="${etapa}"><p><strong>${textoPortal("txt_operacion_seleccionada")}</strong> ${escaparHTML(etiqueta)}</p><h5>${textoPortal("txt_paso_de_tres", { etapa, nombre: traducirPortal(etapa === 1 ? "txt_motivo" : etapa === 2 ? "txt_justificante" : "txt_validacion") })}</h5>${revision}${campos}<p class="mensaje-error" role="alert">${escaparHTML(estado.errorFormulario || "")}</p><div class="acciones-vista"><button type="button" class="boton-secundario" data-b8-accion="anterior" ${etapa === 1 || estado.enviando ? "disabled" : ""}>${textoPortal("txt_anterior")}</button><button type="submit" class="boton-primario" ${estado.enviando ? "disabled" : ""}>${estado.enviando ? traducirPortal("txt_registrando") : etapa < 3 ? traducirPortal("txt_continuar") : validarDisponibilidad ? traducirPortal("txt_b8_confirmar_disponibilidad") : traducirPortal("txt_confirmar_operacion_nombre", { operacion: etiqueta })}</button></div></form>`;
}

export function crearControladorOperacionesSituacion({ estado, renderizar, recargar, consultarReglas = consultarReglasSituacion, consultarDocumentales = consultarSolicitudesDocumentalesRRHH }) {
  async function cargar(modalFicha, { incluirSecciones = true } = {}) {
    // B13: el histórico de contratos se carga junto a la ficha, en paralelo.
    if (incluirSecciones) {
      void cargarContratosFicha(modalFicha, { estado, renderizar, renderizarAlIniciar: false });
      void cargarReincorporacionesTitularFicha(modalFicha, { estado, renderizar, renderizarAlIniciar: false });
    }
    const controlador = new AbortController();
    modalFicha.controladorOperaciones?.abort();
    modalFicha.controladorOperaciones = controlador;
    modalFicha.operacionesB8 = { ...modalFicha.operacionesB8, carga: "cargando", items: [], cambios: [] };
    renderizar();
    // Sin catálogo o situación vigente del servidor, las acciones nuevas
    // quedan cerradas; nunca se deduce el CAS del último registro histórico.
    const finRelacion = modalFicha.candidato.estado_clave === "trabajando" ? hoyCivil() : "";
    const solicitudesPrevias = modalFicha.operacionesB8;
    const consultaDocumental = !incluirSecciones && [401, 403, 404].includes(solicitudesPrevias?.solicitudesStatus)
      ? Promise.resolve({ ok: false, status: solicitudesPrevias.solicitudesStatus,
        mensaje: solicitudesPrevias.solicitudesError })
      : consultarDocumentales(estado.bolsaSeleccionada, modalFicha.candidato.participacion_ref, { signal: controlador.signal });
    const [res, reglas, solicitudes] = await Promise.all([
      consultarOperacionesSituacion(estado.bolsaSeleccionada, modalFicha.candidato.participacion_ref, { signal: controlador.signal }),
      consultarReglas({ finRelacion, signal: controlador.signal }),
      consultaDocumental,
    ]);
    if (controlador.signal.aborted || estado.modalFicha !== modalFicha) return;
    modalFicha.reglasSituacion = reglas.ok ? reglas.datos : null;
    if (res.ok) modalFicha.candidato = { ...modalFicha.candidato,
      estado_clave: res.situacionVigente?.situacion ?? modalFicha.candidato.estado_clave,
      estado_desde: res.situacionVigente?.desde };
    const causas = causasBaja(modalFicha.reglasSituacion);
    const transiciones = modalFicha.reglasSituacion?.transiciones ?? null;
    modalFicha.operacionesB8 = res.ok
      ? { ...modalFicha.operacionesB8, carga: "listo", noDisponible: !res.situacionVigente, items: res.datos, cambios: res.cambios, causasBaja: causas, transiciones,
        solicitudesDocumentales: solicitudes.ok ? solicitudes.datos : [], solicitudesError: solicitudes.ok ? "" : solicitudes.mensaje,
        solicitudesStatus: solicitudes.ok ? 200 : solicitudes.status, solicitudesCargando: false }
      : { ...modalFicha.operacionesB8, carga: "error", noDisponible: true, error: res.mensaje,
        errorStatus: res.status, items: [], cambios: [], causasBaja: causas, transiciones,
        solicitudesDocumentales: [], solicitudesError: solicitudes.ok ? "" : solicitudes.mensaje,
        solicitudesStatus: solicitudes.ok ? 200 : solicitudes.status, solicitudesCargando: false };
    renderizar();
  }

  async function reintentarSolicitudes(modal) {
    const flujo = modal.operacionesB8;
    const controlador = modal.controladorOperaciones;
    if (!flujo || flujo.solicitudesCargando || !controlador || controlador.signal.aborted
      || [401, 403, 404].includes(flujo.solicitudesStatus)) return;
    flujo.solicitudesCargando = true;
    renderizar();
    let respuesta;
    try {
      respuesta = await consultarDocumentales(estado.bolsaSeleccionada, modal.candidato.participacion_ref,
        { signal: controlador.signal });
    } catch {
      respuesta = { ok: false, status: 0, mensaje: traducirPortal("txt_b8_solicitudes_documentales_error") };
    }
    if (controlador.signal.aborted || estado.modalFicha !== modal || modal.operacionesB8 !== flujo) return;
    flujo.solicitudesCargando = false;
    flujo.solicitudesDocumentales = respuesta.ok ? respuesta.datos : [];
    flujo.solicitudesError = respuesta.ok ? "" : respuesta.mensaje;
    flujo.solicitudesStatus = respuesta.ok ? 200 : respuesta.status;
    renderizar();
  }

  function manejarClick(evento) {
    const control = evento.target?.closest?.("[data-b8-accion]");
    if (!control || !estado.modalFicha) return false;
    evento.preventDefault();
    const modal = estado.modalFicha;
    const flujo = modal.operacionesB8 || (modal.operacionesB8 = { carga: "listo", items: [] });
    if (flujo.enviando || (flujo.noDisponible && control.dataset.b8Accion !== "reintentar")) return true;
    if (control.dataset.b8Accion === "seleccionar") {
      if (!operacionAdmitida(modal, flujo, control.dataset.operacion)) {
        flujo.errorOperacion = traducirPortal("txt_b8_regularizacion_no_disponible"); renderizar(); return true;
      }
      flujo.operacion = control.dataset.operacion;
      flujo.paso = 1;
      flujo.formulario = {};
      flujo.errorFormulario = "";
      flujo.errorOperacion = "";
      delete flujo.solicitudSeleccionada;
    } else if (control.dataset.b8Accion === "seleccionar-solicitud") {
      const solicitud = flujo.solicitudesDocumentales?.find((item) => item.solicitud_ref === control.dataset.solicitudRef);
      if (!solicitud || (solicitud.fecha_fin_causa !== null && !fechaCausaValida(solicitud.fecha_fin_causa)) || !operacionAdmitida(modal, flujo, "regularizar")) {
        flujo.errorOperacion = traducirPortal("txt_b8_regularizacion_no_disponible"); renderizar(); return true;
      }
      flujo.operacion = "regularizar";
      flujo.solicitudSeleccionada = solicitud;
      flujo.paso = 1;
      flujo.formulario = { tipo: "solicitud_candidato", referencia: solicitud.documento_ref,
        sha256: solicitud.documento_sha256, causa_finalizada_en: solicitud.fecha_fin_causa || "" };
      flujo.errorFormulario = "";
      flujo.errorOperacion = "";
    } else if (control.dataset.b8Accion === "cancelar") {
      delete flujo.operacion; delete flujo.paso; delete flujo.formulario; delete flujo.clave; delete flujo.huella; delete flujo.solicitudSeleccionada;
    } else if (control.dataset.b8Accion === "anterior") {
      const formulario = control.closest?.('[data-b8-form="operacion"]');
      if (formulario) {
        const datos = new FormData(formulario);
        if (flujo.paso === 3) flujo.formulario = { ...flujo.formulario, validador: String(datos.get("validador") || ""), confirma_validador_distinto: datos.has("confirma_validador_distinto"), confirma_fin_causa: datos.has("confirma_fin_causa") };
        else if (flujo.paso === 2) flujo.formulario = { ...flujo.formulario, tipo: String(datos.get("tipo") || ""), referencia: String(datos.get("referencia") || ""), sha256: String(datos.get("sha256") || ""), causa_finalizada_en: String(datos.get("causa_finalizada_en") || "") };
        else {
          const causa = datos.get("causa");
          const motivo = String(datos.get("motivo") || "");
          flujo.formulario = causa === null
            ? { ...flujo.formulario, motivo }
            : { ...flujo.formulario, causa: String(causa), detalle: motivo, motivo: motivoConCausa(flujo.causasBaja || [], String(causa), motivo) };
        }
      }
      flujo.paso = Math.max(1, Number(flujo.paso || 1) - 1);
    } else if (control.dataset.b8Accion === "reintentar") {
      if ([401, 403, 404].includes(flujo.errorStatus)) return true;
      void cargar(modal, { incluirSecciones: false });
      return true;
    } else if (control.dataset.b8Accion === "reintentar-solicitudes") {
      void reintentarSolicitudes(modal);
      return true;
    } else if (control.dataset.b8Accion === "pagina-traza") {
      flujo.paginaTraza = Math.max(0, Number(control.dataset.pagina) || 0);
    } else if (control.dataset.b8Accion === "pagina") {
      flujo.paginaHistorial = Math.max(0, Number(control.dataset.pagina) || 0);
    }
    renderizar();
    return true;
  }

  async function enviar(modal, flujo) {
    const comando = { operacion: flujo.operacion, motivo: flujo.formulario.motivo, validador: flujo.formulario.validador,
      justificante: { tipo: flujo.formulario.tipo, referencia: flujo.formulario.referencia, sha256: flujo.formulario.sha256 } };
    if (["revisar", "regularizar"].includes(flujo.operacion) || (flujo.operacion === "excluir" && modal.candidato.estado_clave === "en_revision")) comando.situacion_esperada_desde = modal.candidato.estado_desde;
    if (flujo.operacion === "regularizar") comando.causa_finalizada_en = flujo.formulario.causa_finalizada_en;
    if (flujo.solicitudSeleccionada) {
      comando.solicitud_ref = flujo.solicitudSeleccionada.solicitud_ref;
      comando.solicitud_version_esperada = flujo.solicitudSeleccionada.version;
      comando.solicitud_contenido_sha256 = flujo.solicitudSeleccionada.contenido_sha256;
    }
    const huella = JSON.stringify(comando);
    if (flujo.huella !== huella) {
      flujo.huella = huella;
      flujo.clave = globalThis.crypto?.randomUUID?.() || `b8-${Date.now()}-${Math.random().toString(16).slice(2)}`;
    }
    flujo.enviando = true; flujo.errorOperacion = ""; renderizar();
    const respuesta = await registrarOperacionSituacion(estado.bolsaSeleccionada, modal.candidato.participacion_ref, comando, flujo.clave);
    if (estado.modalFicha !== modal) return;
    flujo.enviando = false;
    if (!respuesta.ok) {
      flujo.errorOperacion = respuesta.mensaje;
      if (respuesta.status === 409) {
        delete flujo.operacion; delete flujo.paso; delete flujo.formulario; delete flujo.clave; delete flujo.huella; delete flujo.solicitudSeleccionada;
        await recargar(modal.candidato.participacion_ref);
        if (estado.modalFicha === modal) void cargar(modal);
      }
      if (estado.modalFicha === modal) renderizar();
      return;
    }
    flujo.recibo = respuesta.datos.recibo_ref;
    flujo.reciboResolucion = respuesta.datos.recibo_resolucion_ref || "";
    flujo.reutilizada = respuesta.datos.reutilizada;
    modal.candidato = { ...modal.candidato, estado_clave: respuesta.datos.situacion, estado_desde: respuesta.datos.desde };
    delete flujo.operacion; delete flujo.paso; delete flujo.formulario; delete flujo.clave; delete flujo.huella; delete flujo.solicitudSeleccionada;
    await recargar(modal.candidato.participacion_ref);
    if (estado.modalFicha !== modal) return;
    void cargar(modal);
  }

  function manejarSubmit(evento) {
    const formulario = evento.target?.closest?.('[data-b8-form="operacion"]');
    if (!formulario || !estado.modalFicha) return false;
    evento.preventDefault();
    const modal = estado.modalFicha;
    const flujo = modal.operacionesB8;
    const datos = new FormData(formulario);
    if (flujo.enviando) return true;
    if (!operacionAdmitida(modal, flujo, flujo.operacion)) {
      flujo.errorFormulario = traducirPortal("txt_b8_regularizacion_no_disponible"); renderizar(); return true;
    }
    if (Number(flujo.paso) < 3) {
      const causaElegida = flujo.paso === 1 ? datos.get("causa") : null;
      if (causaElegida !== null && causaElegida !== undefined) {
        const causa = String(causaElegida);
        const detalle = String(datos.get("motivo") || "").trim();
        const motivo = motivoConCausa(flujo.causasBaja || [], causa, detalle);
        flujo.formulario = { ...flujo.formulario, causa, detalle, motivo };
        if (!motivo) { flujo.errorFormulario = traducirReglasSituacion("causa_incompleta"); renderizar(); return true; }
      } else if (flujo.paso === 1) flujo.formulario = { ...flujo.formulario, motivo: String(datos.get("motivo") || "").trim() };
      else flujo.formulario = { ...flujo.formulario, tipo: String(datos.get("tipo") || ""), referencia: String(datos.get("referencia") || "").trim(), sha256: String(datos.get("sha256") || "").trim().toLowerCase(), causa_finalizada_en: String(datos.get("causa_finalizada_en") || "") };
      flujo.errorFormulario = flujo.paso === 2 && referenciaContieneDocumentoIdentidad(flujo.formulario.referencia)
        ? MENSAJE_REFERENCIA_IDENTIDAD : "";
      if (flujo.paso === 2 && flujo.operacion === "regularizar" && !fechaCausaValida(flujo.formulario.causa_finalizada_en)) flujo.errorFormulario = traducirPortal("txt_b8_fecha_fin_causa_invalida");
      if (flujo.paso === 2 && flujo.solicitudSeleccionada &&
          (flujo.formulario.tipo !== "solicitud_candidato" || flujo.formulario.referencia !== flujo.solicitudSeleccionada.documento_ref ||
            flujo.formulario.sha256 !== flujo.solicitudSeleccionada.documento_sha256 ||
            (flujo.solicitudSeleccionada.fecha_fin_causa !== null && flujo.formulario.causa_finalizada_en !== flujo.solicitudSeleccionada.fecha_fin_causa))) {
        flujo.errorFormulario = traducirPortal("txt_b8_documento_solicitud_distinto");
      }
      if (!flujo.errorFormulario) flujo.paso += 1;
      renderizar(); return true;
    }
    flujo.formulario = { ...flujo.formulario, validador: String(datos.get("validador") || "").trim(), confirma_validador_distinto: datos.has("confirma_validador_distinto"), confirma_fin_causa: datos.has("confirma_fin_causa") };
    if ((flujo.operacion === "regularizar" && !flujo.formulario.confirma_fin_causa)
      || (flujo.operacion === "excluir" && !flujo.formulario.confirma_validador_distinto)) {
      flujo.errorFormulario = traducirPortal(flujo.operacion === "regularizar" ? "txt_b8_validacion_fin_causa_pendiente" : "txt_confirmo_que_el_validador_es_otra_persona"); renderizar(); return true;
    }
    void enviar(modal, flujo);
    return true;
  }

  function instalar(documento = globalThis.document) {
    instalarHuellaArchivo(documento);
    documento.addEventListener("click", (evento) => {
      if (manejarClickReincorporacionesTitular(evento, { estado, renderizar })) return;
      if (!manejarClickContratos(evento, { estado, renderizar })) manejarClick(evento);
    });
    documento.addEventListener("submit", (evento) => { manejarSubmit(evento); });
    instalarPropuestaReposicion(documento, () => estado.modalFicha);
  }

  return Object.freeze({ cargar, instalar, manejarClick, manejarSubmit });
}

function operacionAdmitida(modal, flujo, operacion) {
  return !flujo.noDisponible && operacionesAdmitidasCandidato(modal.candidato, flujo).includes(operacion)
    && ((operacion === "excluir" && modal.candidato.estado_clave !== "en_revision") || instanteValido(modal.candidato.estado_desde));
}
