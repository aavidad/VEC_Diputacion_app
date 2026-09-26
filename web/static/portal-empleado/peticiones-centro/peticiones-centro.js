import {
  crearBorradorAlta,
  crearComandoAlta,
  validarBorradorAlta,
  validarCatalogosAlta,
} from "../modulos/contratacion-temporal/contrato.js";
import {
  extraerBorradorPeticionCentro,
  renderizarFormularioPeticionCentro,
  renderizarRevisionPeticionCentro,
} from "../modulos/contratacion-temporal/vista.js";
import { MENSAJES_CONTRATACION_TEMPORAL_ES, crearTraductorContratacionTemporal } from "../modulos/contratacion-temporal/i18n.js";
import { aplicarIdiomaDocumento, aplicarTextosPortal, instalarValidacionI18n } from "../portal-idioma.js?v=20260926-pulido-portal-v1";

const RUTAS = Object.freeze({
  contexto: "/api/vec/contratacion-temporal/peticiones-centro/contexto",
  bandeja: "/api/vec/contratacion-temporal/peticiones-centro/bandeja",
  operaciones: "/api/vec/contratacion-temporal/peticiones-centro/operaciones",
  rrhh: "/api/vec/contratacion-temporal/peticiones-centro/rrhh",
});
const MAX_BODY = 2 * 1024 * 1024;
const TIMEOUT_MS = 15_000;
const traducirCentro = crearTraductorContratacionTemporal();
const TEXTO = Object.freeze({
  sobrelinea: "Contratación temporal · circuito previo",
  titulo: "Petición del centro y ratificación",
  descripcion: "Una petición previa reúne la necesidad del centro antes de que RRHH la transfiera al expediente de contratación.",
  pendienteEntrada: "RRHH tramita las peticiones ratificadas desde su bandeja",
  solicitante: "Presentar petición",
  ratificador: "Bandeja de ratificación",
  peticiones: "Peticiones del centro",
  detalle: "Detalle revisable",
  sinPeticiones: "No hay peticiones disponibles para este actor.",
  cargar: "Cargando contexto y peticiones…",
  recargar: "Recargar bandeja",
  seleccionar: "Revisar",
  volver: "Volver a Contratación",
  confirmarPresentar: "Confirmar presentación de esta petición",
  confirmarRatificar: "Confirmar ratificación de esta petición",
  confirmarPregunta: "Revise todos los datos y confirme expresamente para continuar.",
  motivo: "Motivo de la ratificación",
  motivoAyuda: "Explique brevemente la revisión realizada.",
  ratificar: "Ratificar petición",
  cancelar: "Volver a la bandeja",
  estadoPendiente: "Resultado pendiente: conserve esta pantalla y reintente la misma operación.",
  error: "No se pudo completar la operación.",
  conflicto: "La petición cambió. Se ha recargado la bandeja; revise antes de continuar.",
  exito: "Operación registrada",
  peticionRef: "Referencia de petición",
  reciboRef: "Referencia del recibo",
  version: "Versión",
  actor: "Referencia de quien registró",
  registrado: "Registrado en",
  estado: "Estado",
  solicitanteDatos: "Solicitante y cargo",
  transferencia: "El alta se realiza en la bandeja de RRHH. Esta vista no consulta su estado de entrega.",
  peticionNoEnviada: "Este recibo acredita la actuación del centro, no el alta del expediente",
  confirmacion: "Confirmo expresamente esta operación",
  volverEditar: "Volver a editar",
  datosNoDisponibles: "No hay detalle seleccionado.",
  nombre: "Nombre",
  cargo: "Cargo",
  centro: "Centro",
  pendiente: "pendiente de ratificación",
  ratificada: "ratificada",
  enviando: "Registrando la operación. Espere el recibo antes de cerrar.",
  bandejaNoActualizada: "El registro está confirmado. No se pudo actualizar la bandeja; puede recargarla sin volver a registrar.",
  motivoInvalido: "Escriba el motivo sin saltos de línea (máximo 1000 bytes) y marque la confirmación.",
  rrhhSobrelinea: "Contratación temporal · Recursos Humanos",
  rrhhTitulo: "Peticiones de los centros",
  rrhhDescripcion: "Revise los datos ratificados antes de crear el expediente de contratación. Esta acción no modifica la petición original.",
  rrhhPendiente: "Pendiente de preparación",
  rrhhPreparada: "Preparada para crear expediente",
  rrhhConfirmada: "Expediente creado",
  rrhhConfirmar: "Crear expediente en RRHH",
  rrhhCompletar: "Completar registro",
  rrhhConfirmacion: "Confirmo expresamente la creación del expediente en RRHH con estos datos.",
  rrhhAviso: "La confirmación crea un único expediente a partir de la petición ratificada. Revise los datos originales antes de continuar.",
  rrhhRecibo: "Recibo histórico de alta",
  rrhhExpediente: "Referencia del expediente",
  rrhhBandeja: "Abrir bandeja de expedientes",
  rrhhSinPeticiones: "No hay peticiones disponibles para Recursos Humanos.",
  rrhhError: "No se pudo completar el registro en RRHH.",
  accesoDenegado: traducirCentro("pc_acceso_denegado"),
  lecturaFallida: traducirCentro("pc_lectura_fallida"),
  operacionConfirmadaOculta: traducirCentro("pc_operacion_confirmada_oculta"),
  operacionInciertaOculta: traducirCentro("pc_operacion_incierta_oculta"),
  operacionInciertaVerificada: traducirCentro("pc_operacion_incierta_verificada"),
});
const MENSAJES = Object.freeze({
  ...MENSAJES_CONTRATACION_TEMPORAL_ES,
  sobrelinea: TEXTO.sobrelinea,
  titulo: TEXTO.solicitante,
  descripcion: TEXTO.descripcion,
  alcance: TEXTO.pendienteEntrada,
  progreso_etiqueta: "Progreso de la petición",
  progreso_datos: "Datos",
  progreso_revision: "Revisión",
  progreso_recibo: "Registro",
  revision_titulo: "Revise la petición antes de presentarla",
  revision_aviso: "La confirmación registrará una petición previa; no crea un expediente.",
  confirmar: "Confirmar presentación",
  revisar: "Revisar petición",
  estado_disponible: "Petición preparada para revisión",
  resumen_contacto: "Contacto del centro (no quien presenta)",
  contacto_ref: "Contacto del centro",
});

const textoCT = (clave, variables) => esc(traducirCentro(clave, variables));

function esc(value) {
  return String(value ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#039;");
}

function claveUUID() {
  if (globalThis.crypto?.randomUUID) return globalThis.crypto.randomUUID();
  throw new Error("El navegador no permite generar una clave UUIDv4");
}

async function leerCuerpoLimitado(respuesta) {
  if (!respuesta.body?.getReader) {
    const texto = await respuesta.text();
    if (texto.length > MAX_BODY) throw new Error("respuesta demasiado grande");
    return texto;
  }
  const lector = respuesta.body.getReader();
  const partes = [];
  let total = 0;
  try {
    while (true) {
      const parte = await lector.read();
      if (parte.done) break;
      total += parte.value.byteLength;
      if (total > MAX_BODY) throw new Error("respuesta demasiado grande");
      partes.push(parte.value);
    }
  } finally { await lector.cancel().catch(() => {}); }
  const bytes = new Uint8Array(total);
  let offset = 0;
  for (const parte of partes) { bytes.set(parte, offset); offset += parte.byteLength; }
  return new TextDecoder().decode(bytes);
}

export async function pedir(ruta, { method = "GET", cuerpo, signal } = {}) {
  const controlador = new AbortController();
  const temporizador = setTimeout(() => controlador.abort("timeout"), TIMEOUT_MS);
  const abortar = () => controlador.abort(signal?.reason || "cancelado");
  signal?.addEventListener("abort", abortar, { once: true });
  try {
    const respuesta = await fetch(ruta, {
      // Igual que el cliente CT existente: Firefox necesita same-origin para
      // presentar el certificado TLS. El servidor no usa cookies.
      method, credentials: "same-origin", mode: "same-origin", cache: "no-store",
      redirect: "error", referrerPolicy: "no-referrer", signal: controlador.signal,
      headers: cuerpo === undefined ? {} : { "Content-Type": "application/json; charset=utf-8" },
      body: cuerpo === undefined ? undefined : JSON.stringify(cuerpo),
    });
    if (esDenegacion({ status: respuesta.status })) {
      throw Object.assign(new Error(TEXTO.accesoDenegado), { status: respuesta.status });
    }
    const texto = await leerCuerpoLimitado(respuesta);
    let data = null;
    try { data = texto ? JSON.parse(texto) : null; } catch {
      if (!respuesta.ok) throw Object.assign(new Error(TEXTO.error), { status: respuesta.status });
      throw Object.assign(new Error(TEXTO.error), { indeterminado: true });
    }
    if (!respuesta.ok) throw Object.assign(new Error(data?.error?.codigo || TEXTO.error), { status: respuesta.status, payload: data });
    return data?.data;
  } catch (error) {
    if (controlador.signal.aborted || !error?.status) {
      throw Object.assign(new Error(TEXTO.estadoPendiente), { indeterminado: true });
    }
    throw error;
  } finally {
    clearTimeout(temporizador); signal?.removeEventListener("abort", abortar);
  }
}

function estadoBase(catalogos, borrador, extra = {}) {
  return { disponible: true, ocupado: false, fase: "edicion", borrador, catalogos,
    errores: {}, mensaje_clave: "estado_disponible", tipo_mensaje: "informacion", ...extra };
}

// Evento con el que la bandeja de incorporaciones (incorporaciones-centro.js)
// publica sus expedientes; se repite el nombre para no importar ese módulo,
// que se monta solo al cargarse.
export const EVENTO_EXPEDIENTES_CENTRO = "vec:expedientes-centro";

const selectorSeguro = (valor) => String(valor).replace(/["\\]/gu, "\\$&");

function esDenegacion(error) { return [401, 403].includes(error?.status); }

function vistaSinDatos(cabecera, modo, mensaje, accionRecargar) {
  const titulo = traducirCentro(modo === "denegado" ? "pc_titulo_denegado" : modo === "resultado_incierto" ? "pc_titulo_incierto" : "pc_titulo_sin_consulta");
  return `${cabecera}<section class="pc-panel pc-detalle" role="alert"><h2>${esc(titulo)}</h2><p>${esc(mensaje)}</p>${modo === "sin_verificar" ? `<div class="pc-acciones"><button type="button" class="boton-secundario" data-accion="${esc(accionRecargar)}">${esc(traducirCentro("pc_reintentar_consulta"))}</button></div>` : ""}</section>`;
}

function fecha(valor, hora = false) {
  if (!valor || !Number.isFinite(Date.parse(valor))) return "—";
  return new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", ...(hora ? { timeStyle: "medium" } : {}), timeZone: hora ? "Europe/Madrid" : "UTC" }).format(new Date(valor));
}

// Campos anchos: texto libre o valores compuestos que necesitan toda la fila.
const CAMPOS_ANCHOS = new Set([traducirCentro("ct_txt_detalle"), traducirCentro("ct_txt_observaciones"), traducirCentro("ct_txt_motivo_de_ratificacion"), traducirCentro("ct_txt_retencion_de_credito"), traducirCentro("ct_txt_documentos_aportados")]);
// Referencias opacas y códigos: se muestran en monoespaciada y pueden partirse.
const PATRON_REFERENCIA = /^[a-z_]+[:_][A-Za-z0-9:_./-]{12,}$/;

function camposDetalle(filas) {
  return `<dl class="pc-campos">${filas.map(([k, v]) => {
    const valor = String(v ?? "—");
    const clases = ["pc-campo"];
    if (CAMPOS_ANCHOS.has(k) || valor.length > 90) clases.push("pc-campo-ancho");
    const valorClase = PATRON_REFERENCIA.test(valor) ? ' class="pc-referencia"' : "";
    return `<div class="${clases.join(" ")}"><dt>${esc(k)}</dt><dd${valorClase}>${esc(valor)}</dd></div>`;
  }).join("")}</dl>`;
}

function detallePeticion(peticion, contexto) {
  if (!peticion) return `<p>${esc(TEXTO.datosNoDisponibles)}</p>`;
  const s = peticion.solicitud || {};
  const c = peticion.configuracion?.solicitante;
  const actor = contexto?.actor;
  const solicitante = contexto?.intervinientes?.[c?.actor_ref];
  const catalogos = contexto?.catalogos;
  const centro = catalogos?.centros.find((v) => v.referencia === s.centro_ref);
  const etiqueta = (opciones, referencia) => opciones?.find((v) => v.referencia === referencia)?.etiqueta || referencia || "—";
  const rc = s.rc?.existe
    ? `${s.rc.numero} · ${fecha(s.rc.fecha)} · ${new Intl.NumberFormat("es-ES", { style: "currency", currency: "EUR" }).format(s.rc.importe.centimos / 100)} · ${s.rc.documento_ref}`
    : traducirCentro("ct_txt_sin_retencion_de_credito_aportada");
  const filas = [[TEXTO.peticionRef, peticion.referencia], [TEXTO.estado, peticion.estado], [TEXTO.version, peticion.version],
    [traducirCentro("ct_txt_centro"), centro?.etiqueta || s.centro_ref], [traducirCentro("ct_txt_contacto"), etiqueta(centro?.contactos, s.contacto_ref)],
    [traducirCentro("ct_txt_categoria"), etiqueta(catalogos?.categorias, s.categoria_ref)], [traducirCentro("ct_txt_grupo_o_subgrupo"), s.grupo_subgrupo], [traducirCentro("ct_txt_motivo"), s.motivo_clave],
    [traducirCentro("ct_txt_detalle"), s.detalle], [traducirCentro("ct_txt_periodo"), `${fecha(s.periodo?.inicio)} — ${fecha(s.periodo?.fin)}`], [traducirCentro("ct_txt_observaciones"), s.observaciones || "—"],
    [traducirCentro("ct_txt_retencion_de_credito"), rc], [traducirCentro("ct_txt_documentos_aportados"), (s.documentos_adjuntos || []).map((ref) => etiqueta(catalogos?.documentos, ref)).join(" · ") || traducirCentro("ct_txt_ninguno")],
    [TEXTO.solicitanteDatos, solicitante?.puesto_ref === c?.puesto_ref
      ? `${solicitante.nombre} · ${solicitante.cargo}`
      : `${c?.actor_ref || actor?.referencia || "—"} · ${c?.puesto_ref || traducirCentro("ct_txt_cargo_resuelto_por_identidad")}`],
    [traducirCentro("ct_txt_creada_en"), fecha(peticion.creada_en, true)]];
  if (peticion.estado === "ratificada") {
    const rat = peticion.configuracion?.ratificador;
    const etiquetaRat = contexto?.intervinientes?.[rat?.actor_ref];
    filas.push([traducirCentro("ct_txt_ratificador_y_cargo"), etiquetaRat && rat && etiquetaRat.puesto_ref === rat.puesto_ref ? `${etiquetaRat.nombre} · ${etiquetaRat.cargo}` : `${rat?.actor_ref || "—"} · ${rat?.puesto_ref || "—"}`],
      [traducirCentro("ct_txt_motivo_de_ratificacion"), peticion.motivo_ratificacion], [traducirCentro("ct_txt_ratificada_en"), fecha(peticion.ratificada_en, true)]);
  }
  return camposDetalle(filas);
}

function reciboHTML(recibo) {
  return `<section class="pc-panel pc-recibo" role="status"><h2>${esc(TEXTO.exito)}</h2><dl>
    <dt>${esc(TEXTO.reciboRef)}</dt><dd>${esc(recibo.recibo_ref)}</dd><dt>${esc(TEXTO.peticionRef)}</dt><dd>${esc(recibo.peticion_ref)}</dd>
    <dt>${esc(TEXTO.version)}</dt><dd>${esc(recibo.version)}</dd><dt>${esc(TEXTO.estado)}</dt><dd>${esc(recibo.estado)}</dd>
    <dt>${esc(TEXTO.actor)}</dt><dd>${esc(recibo.actor_ref)}</dd><dt>${esc(TEXTO.registrado)}</dt><dd>${esc(fecha(recibo.registrado_en, true))}</dd></dl>
    <p>${esc(TEXTO.peticionNoEnviada)}. ${esc(TEXTO.transferencia)}</p></section>`;
}

export function validarReciboPeticionCentro(recibo, objetivo) {
  if (!recibo || recibo.peticion_ref !== objetivo.peticionRef || recibo.version !== objetivo.version
    || recibo.estado !== objetivo.estado || recibo.actor_ref !== objetivo.actorRef
    || !recibo.recibo_ref || !recibo.registrado_en || !Number.isFinite(Date.parse(recibo.registrado_en))
    || !["registrado", "replay_confirmado"].includes(recibo.estado_local)) throw new Error(TEXTO.error);
  return recibo;
}

function textoEstadoEntrega(estado) {
  return ({ pendiente: TEXTO.rrhhPendiente, preparada: TEXTO.rrhhPreparada, confirmada: TEXTO.rrhhConfirmada })[estado] || "—";
}

// Enlace de una petición entregada a su expediente en Contratación temporal
// (RRHH). La referencia solo navega: el servidor decide si el perfil lo ve.
function urlExpedienteRRHH(expedienteRef) {
  return `/portal-empleado/?expediente=${encodeURIComponent(expedienteRef)}#contratacion-temporal`;
}

function enlaceExpedienteRRHH(peticion, recibo) {
  if (!recibo?.expediente_ref) return "";
  const numero = recibo.numero_visible || recibo.expediente_ref;
  return ` <a class="pc-enlace-expediente" href="${esc(urlExpedienteRRHH(recibo.expediente_ref))}" aria-label="${textoCT("pc_expediente_enlace_rrhh_aria", { peticion: peticion?.referencia, numero })}">${textoCT("pc_expediente_enlace", { numero })}</a>`;
}

// Enlace de una petición del centro a la fila de su expediente en la bandeja
// de incorporaciones de la misma página: el centro no tiene vista de expediente.
function enlaceExpedienteCentro(peticion, expediente) {
  if (!expediente) return "";
  return ` <a class="pc-enlace-expediente" href="#${esc(expediente.destino)}" data-pc-ir-expediente="${esc(expediente.destino)}" aria-label="${textoCT("pc_expediente_enlace_centro_aria", { peticion: peticion.referencia, numero: expediente.numero_visible })}">${textoCT("pc_expediente_enlace", { numero: expediente.numero_visible })}</a>`;
}

function reciboAltaRRHHHTML(recibo) {
  if (!recibo) return "";
  const filas = [[TEXTO.rrhhExpediente, recibo.expediente_ref], [traducirCentro("ct_txt_numero_visible"), recibo.numero_visible], [TEXTO.version, recibo.version],
    [TEXTO.reciboRef, recibo.recibo_ref], [traducirCentro("ct_txt_referencia_de_auditoria"), recibo.auditoria_ref], [traducirCentro("ct_txt_referencia_de_evento"), recibo.evento_ref],
    [traducirCentro("ct_txt_confirmada_en"), fecha(recibo.confirmada_en, true)]];
  return `<section class="pc-panel pc-recibo" role="status"><h2>${esc(TEXTO.rrhhRecibo)}</h2>${camposDetalle(filas.map(([k, v]) => [k, v || "—"]))}${recibo.expediente_ref ? `<p class="pc-acciones"><a class="boton-primario" href="${esc(urlExpedienteRRHH(recibo.expediente_ref))}">${textoCT("pc_abrir_expediente")}</a><a class="boton-secundario" href="/portal-empleado/#contratacion-temporal">${esc(TEXTO.rrhhBandeja)}</a></p>` : ""}</section>`;
}

function tablaRRHH(peticiones, seleccionada) {
  if (!peticiones.length) return `<p>${esc(TEXTO.rrhhSinPeticiones)}</p>`;
  return `<div class="pc-tabla-wrap"><table class="pc-tabla"><caption class="solo-lectura">${esc(TEXTO.rrhhTitulo)}</caption><thead><tr><th>${textoCT("ct_txt_referencia")}</th><th>${textoCT("ct_txt_estado_de_entrega")}</th><th>${textoCT("ct_txt_ratificacion")}</th><th>${textoCT("ct_txt_accion")}</th></tr></thead><tbody>${peticiones.map(({ peticion, estado_entrega: estadoEntrega, recibo_alta: reciboAlta }) => `<tr${peticion?.referencia === seleccionada ? ' aria-selected="true"' : ""}><td>${esc(peticion?.referencia)}</td><td><span class="pc-estado pc-estado-${esc(estadoEntrega)}">${esc(textoEstadoEntrega(estadoEntrega))}</span>${estadoEntrega === "confirmada" ? enlaceExpedienteRRHH(peticion, reciboAlta) : ""}</td><td>${esc(peticion?.ratificada_en ? fecha(peticion.ratificada_en, true) : "—")}</td><td><button type="button" data-seleccionar-rrhh="${esc(peticion?.referencia)}">${esc(TEXTO.seleccionar)}</button></td></tr>`).join("")}</tbody></table></div>`;
}

export function renderizarPeticionesCentroRRHH({ peticiones = [], entrega = null, modo = "bandeja", confirmado = false, recibo = null, mensaje = "" } = {}) {
  const peticion = entrega?.peticion;
  const cabecera = `<section class="pc-cabecera"><p class="sobrelinea">${esc(TEXTO.rrhhSobrelinea)}</p><h1>${esc(TEXTO.rrhhTitulo)}</h1><p>${esc(TEXTO.rrhhDescripcion)}</p></section>`;
  const error = mensaje ? `<p class="pc-error" role="alert">${esc(mensaje)}</p>` : "";
  if (["denegado", "sin_verificar", "resultado_incierto"].includes(modo)) return vistaSinDatos(cabecera, modo, mensaje, "recargar-rrhh");
  if (modo === "confirmar") return `${cabecera}${error}<section class="pc-panel pc-detalle"><h2>${esc(TEXTO.rrhhConfirmar)}</h2>${detallePeticion(peticion, null)}<p class="pc-aviso">${esc(TEXTO.rrhhAviso)}</p><label class="pc-confirmacion"><input type="checkbox" name="confirmacion-alta-rrhh"${confirmado ? " checked" : ""}> ${esc(TEXTO.rrhhConfirmacion)}</label><div class="pc-acciones"><button type="button" class="boton-secundario" data-accion="cancelar-alta-rrhh">${esc(TEXTO.cancelar)}</button><button type="button" class="boton-primario" data-accion="confirmar-alta-rrhh">${esc(TEXTO.rrhhConfirmar)}</button></div></section>`;
  if (modo === "pendiente") return `${cabecera}<section class="pc-panel pc-pendiente" role="status"><h2>${esc(TEXTO.estadoPendiente)}</h2><p>${esc(TEXTO.rrhhAviso)}</p><div class="pc-acciones"><button type="button" class="boton-primario" data-accion="reintentar-alta-rrhh">${esc(traducirCentro("ct_txt_reintentar_la_misma_operacion"))}</button></div></section>`;
  const detalle = `<aside class="pc-panel pc-detalle"><h2>${esc(TEXTO.detalle)}</h2>${detallePeticion(peticion, null)}${entrega?.recibo_alta && !recibo ? reciboAltaRRHHHTML(entrega.recibo_alta) : ""}${["pendiente", "preparada"].includes(entrega?.estado_entrega) ? `<div class="pc-acciones"><button type="button" class="boton-primario" data-accion="abrir-alta-rrhh">${esc(entrega.estado_entrega === "preparada" ? TEXTO.rrhhCompletar : TEXTO.rrhhConfirmar)}</button></div>` : ""}</aside>`;
  return `${cabecera}${error}${recibo ? reciboAltaRRHHHTML(recibo) : ""}<div class="pc-layout"><section class="pc-panel"><h2>${esc(TEXTO.rrhhTitulo)}</h2>${tablaRRHH(peticiones, peticion?.referencia)}<p>${textoCT("ct_txt_ultimas_50_peticiones_visibles_para_recursos_hum")}</p><div class="pc-acciones"><button type="button" class="boton-secundario" data-accion="recargar-rrhh">${esc(TEXTO.recargar)}</button><a class="boton-secundario" href="/portal-empleado/#contratacion-temporal">${esc(TEXTO.volver)}</a></div></section>${detalle}</div>`;
}

export async function registrarAltaRRHH(cliente, comando) {
  try {
    const resultado = await cliente(RUTAS.rrhh, { method: "POST", cuerpo: { peticion_ref: comando.peticion_ref, version_esperada: 2 } });
    const recibo = resultado?.recibo_alta;
    if (!resultado?.peticion?.referencia || resultado.peticion.referencia !== comando.peticion_ref
      || resultado.estado_entrega !== "confirmada" || !recibo?.expediente_ref || !recibo.recibo_ref || !recibo.confirmada_en
      || !Number.isFinite(Date.parse(recibo.confirmada_en))) throw new Error(TEXTO.rrhhError);
    return resultado;
  } catch (error) {
    if ([400, 401, 403, 409].includes(error?.status)) throw error;
    throw Object.assign(new Error(TEXTO.estadoPendiente), { indeterminado: true });
  }
}

function tabla(peticiones, seleccionada, expedientes = new Map()) {
  if (!peticiones.length) return `<p>${esc(TEXTO.sinPeticiones)}</p>`;
  return `<div class="pc-tabla-wrap"><table class="pc-tabla"><caption class="solo-lectura">${esc(TEXTO.peticiones)}</caption><thead><tr><th>${textoCT("ct_txt_referencia")}</th><th>${textoCT("ct_txt_centro")}</th><th>${textoCT("ct_txt_estado")}</th><th>${textoCT("ct_txt_creada")}</th><th>${textoCT("ct_txt_accion")}</th></tr></thead><tbody>${peticiones.map((p) => `<tr${p.referencia === seleccionada ? ' aria-selected="true"' : ""}><td>${esc(p.referencia)}</td><td>${esc(p.solicitud?.centro_ref)}</td><td><span class="pc-estado pc-estado-${esc(p.estado === "ratificada" ? "ratificada" : "pendiente")}">${esc(p.estado === "ratificada" ? TEXTO.ratificada : TEXTO.pendiente)}</span>${enlaceExpedienteCentro(p, expedientes.get(p.referencia))}</td><td>${esc(fecha(p.creada_en))}</td><td><button type="button" data-seleccionar="${esc(p.referencia)}">${esc(TEXTO.seleccionar)}</button></td></tr>`).join("")}</tbody></table></div>`;
}

function formularioHTML(contexto, estado, revision) {
  const contenido = revision ? renderizarRevisionPeticionCentro(estado, { mensajes: MENSAJES }) : renderizarFormularioPeticionCentro(estado, { mensajes: MENSAJES });
  return `<section class="pc-panel ct-alta"><h2>${esc(TEXTO.solicitante)}</h2><p class="pc-aviso">${esc(TEXTO.confirmarPregunta)}</p>${contenido}<button type="button" class="boton-secundario" data-accion="cancelar-ratificacion">${esc(TEXTO.cancelar)}</button></section>`;
}

export function renderizarPeticionCentro({ contexto, peticiones = [], peticion = null, modo = "bandeja", estado = null, recibo = null, mensaje = "", motivo = "", confirmado = false, expedientes = new Map() } = {}) {
  const actor = contexto?.actor;
  const esSolicitante = actor?.puede_presentar && !actor?.puede_ratificar;
  if (["denegado", "sin_verificar", "resultado_incierto"].includes(modo)) {
    const cabeceraSegura = `<section class="pc-cabecera"><p class="sobrelinea">${esc(TEXTO.sobrelinea)}</p><h1>${esc(TEXTO.titulo)}</h1></section>`;
    return vistaSinDatos(cabeceraSegura, modo, mensaje, "recargar");
  }
  const cabecera = `<section class="pc-cabecera"><p class="sobrelinea">${esc(TEXTO.sobrelinea)}</p><h1>${esc(TEXTO.titulo)}</h1><p>${esc(TEXTO.descripcion)}</p><div class="pc-etiquetas"><span class="pc-etiqueta">${esc(TEXTO.pendienteEntrada)}</span></div><p>${esc(actor?.nombre || "—")} · ${esc(actor?.cargo || "—")} · ${esc(actor?.centro || "—")}</p></section>`;
  const error = mensaje ? `<p class="pc-error" role="alert">${esc(mensaje)}</p>` : "";
  if (modo === "formulario") return `${cabecera}${error}${formularioHTML(contexto, estado, false)}`;
  if (modo === "revision") return `${cabecera}${error}${formularioHTML(contexto, estado, true)}`;
  if (modo === "ratificacion") return `${cabecera}${error}<section class="pc-panel pc-detalle"><h2>${esc(TEXTO.ratificador)}</h2>${detallePeticion(peticion, contexto)}<p class="pc-aviso">${esc(TEXTO.confirmarPregunta)}</p><label for="motivo-ratificacion">${esc(TEXTO.motivo)}</label><input id="motivo-ratificacion" name="motivo_ratificacion" value="${esc(motivo)}" maxlength="1000" required aria-describedby="motivo-ratificacion-ayuda"><small id="motivo-ratificacion-ayuda">${esc(TEXTO.motivoAyuda)}</small><label class="pc-confirmacion"><input type="checkbox" name="confirmacion_ratificacion"${confirmado ? " checked" : ""}> ${esc(TEXTO.confirmacion)}</label><div class="pc-acciones"><button type="button" class="boton-secundario" data-accion="cancelar-ratificacion">${esc(TEXTO.cancelar)}</button><button type="button" class="boton-primario" data-accion="confirmar-ratificar">${esc(TEXTO.confirmarRatificar)}</button></div></section>`;
  if (modo === "pendiente") return `${cabecera}<section class="pc-panel pc-pendiente" role="status"><h2>${esc(TEXTO.estadoPendiente)}</h2><p>${esc(TEXTO.peticionNoEnviada)}</p><div class="pc-acciones"><button type="button" class="boton-primario" data-accion="reintentar">${esc(traducirCentro("ct_txt_reintentar_la_misma_operacion"))}</button></div></section>`;
  const detalle = `<aside class="pc-panel pc-detalle"><h2>${esc(TEXTO.detalle)}</h2>${detallePeticion(peticion, contexto)}${peticion?.estado === "pendiente_ratificacion" && peticion.version === 1 && actor?.puede_ratificar ? `<div class="pc-acciones"><button type="button" class="boton-primario" data-accion="abrir-ratificacion">${esc(TEXTO.ratificador)}</button></div>` : ""}</aside>`;
  return `${cabecera}${error}${recibo ? reciboHTML(recibo) : ""}<div class="pc-layout"><section class="pc-panel"><h2>${esc(esSolicitante ? TEXTO.peticiones : TEXTO.ratificador)}</h2>${tabla(peticiones, peticion?.referencia, expedientes)}<p>${textoCT("ct_txt_ultimas_50_peticiones_visibles_para_su_identidad")}</p><div class="pc-acciones">${esSolicitante ? `<button type="button" class="boton-primario" data-accion="nueva">${esc(TEXTO.solicitante)}</button>` : ""}<button type="button" class="boton-secundario" data-accion="recargar">${esc(TEXTO.recargar)}</button><button type="button" class="boton-secundario" data-accion="volver-contratacion">${esc(TEXTO.volver)}</button></div></section>${detalle}</div>`;
}

export async function registrarOperacionPeticionCentro(cliente, comando, actorRef) {
  const objetivo = {
    peticionRef: comando.operacion === "presentar" ? `peticion:centro:${comando.clave_idempotencia}` : comando.peticion_ref,
    version: comando.operacion === "presentar" ? 1 : 2,
    estado: comando.operacion === "presentar" ? "pendiente_ratificacion" : "ratificada", actorRef,
  };
  try {
    const recibo = await cliente(RUTAS.operaciones, { method: "POST", cuerpo: structuredClone(comando) });
    return validarReciboPeticionCentro(recibo, objetivo);
  } catch (error) {
    // Solo un rechazo explícito permite descartar la clave. Transporte, lectura
    // o recibo inválido no prueban que PostgreSQL no haya confirmado.
    if ([400, 401, 403, 409].includes(error?.status)) throw error;
    throw Object.assign(new Error(TEXTO.estadoPendiente), { indeterminado: true });
  }
}

export async function iniciarPeticionesCentroRRHH({ raiz = document.querySelector("#aplicacion"), cliente = pedir } = {}) {
  if (!raiz) throw new TypeError("falta la raíz de la aplicación");
  let peticiones = []; let entrega = null; let modo = "bandeja"; let recibo = null; let mensaje = "";
  let ocupado = false; let operacionPendiente = null; let resultadoIncierto = false; let confirmado = false;
  const retirarDatos = (error) => {
    const confirmada = Boolean(recibo);
    resultadoIncierto = resultadoIncierto || Boolean(operacionPendiente);
    operacionPendiente = null;
    peticiones = []; entrega = null; recibo = null; confirmado = false;
    modo = esDenegacion(error) ? "denegado" : "sin_verificar";
    mensaje = `${modo === "denegado" ? TEXTO.accesoDenegado : TEXTO.lecturaFallida}${confirmada ? ` ${TEXTO.operacionConfirmadaOculta}` : ""}${resultadoIncierto ? ` ${TEXTO.operacionInciertaOculta}` : ""}`;
  };
  const dibujar = () => {
    raiz.innerHTML = renderizarPeticionesCentroRRHH({ peticiones, entrega, modo, confirmado, recibo, mensaje });
    raiz.setAttribute("aria-busy", String(ocupado));
    if (ocupado) raiz.querySelectorAll("button, input").forEach((control) => { control.disabled = true; });
  };
  const cargar = async ({ posterior = false } = {}) => {
    if ((!posterior && ocupado) || operacionPendiente) return false;
    if (!posterior) { ocupado = true; mensaje = ""; }
    try {
      const bandeja = await cliente(RUTAS.rrhh);
      if (!bandeja || bandeja.limite !== 50 || !Array.isArray(bandeja.peticiones) || bandeja.peticiones.length > 50
        || bandeja.peticiones.some((item) => !item?.peticion?.referencia || item.peticion.version !== 2
          || !["pendiente", "preparada", "confirmada"].includes(item.estado_entrega))) throw new Error(TEXTO.rrhhError);
      peticiones = bandeja.peticiones;
      entrega = peticiones.find((item) => item.peticion.referencia === entrega?.peticion?.referencia) || peticiones[0] || null;
      if (resultadoIncierto) {
        peticiones = []; entrega = null;
        modo = "resultado_incierto"; mensaje = TEXTO.operacionInciertaVerificada;
      } else if (modo === "denegado" || modo === "sin_verificar") modo = "bandeja";
      return true;
    } catch (error) { retirarDatos(error); return false; }
    finally { if (!posterior) ocupado = false; dibujar(); }
  };
  const ejecutar = async (comando) => {
    if (ocupado || resultadoIncierto) return;
    ocupado = true; mensaje = ""; dibujar();
    try {
      const resultado = await registrarAltaRRHH(cliente, comando);
      recibo = resultado.recibo_alta; entrega = { peticion: resultado.peticion, estado_entrega: resultado.estado_entrega, recibo_alta: resultado.recibo_alta };
      modo = "bandeja"; operacionPendiente = null;
      // La respuesta confirma el alta; una lectura posterior fallida retira
      // el recibo de la vista e informa de que el registro sigue en el servidor.
      await cargar({ posterior: true });
    } catch (error) {
      if (esDenegacion(error)) retirarDatos(error);
      else if (error.indeterminado) { operacionPendiente = comando; modo = "pendiente"; mensaje = TEXTO.estadoPendiente; }
      else { modo = "bandeja"; mensaje = error.status === 409 ? TEXTO.conflicto : TEXTO.rrhhError; }
    } finally { ocupado = false; dibujar(); }
  };
  raiz.addEventListener("click", async (event) => {
    const control = event.target.closest?.("[data-accion], [data-seleccionar-rrhh]");
    if (!control || ocupado || modo === "denegado" || modo === "resultado_incierto"
      || (modo === "sin_verificar" && control.dataset.accion !== "recargar-rrhh")
      || (operacionPendiente && control.dataset.accion !== "reintentar-alta-rrhh")) return;
    event.preventDefault();
    if (control.dataset.seleccionarRrhh) { entrega = peticiones.find((item) => item.peticion.referencia === control.dataset.seleccionarRrhh) || null; recibo = null; dibujar(); return; }
    if (control.dataset.accion === "recargar-rrhh") { await cargar(); return; }
    if (control.dataset.accion === "abrir-alta-rrhh" && ["pendiente", "preparada"].includes(entrega?.estado_entrega)) { modo = "confirmar"; confirmado = false; recibo = null; dibujar(); return; }
    if (control.dataset.accion === "cancelar-alta-rrhh") { modo = "bandeja"; dibujar(); return; }
    if (control.dataset.accion === "reintentar-alta-rrhh" && operacionPendiente) { await ejecutar(operacionPendiente); return; }
    if (control.dataset.accion === "confirmar-alta-rrhh" && modo === "confirmar") {
      confirmado = raiz.querySelector("[name=confirmacion-alta-rrhh]")?.checked === true;
      if (!confirmado || !entrega?.peticion?.referencia) { mensaje = TEXTO.confirmacion; dibujar(); return; }
      await ejecutar({ peticion_ref: entrega.peticion.referencia, version_esperada: 2 });
    }
  });
  raiz.innerHTML = `<p class="pc-cargando">${esc(TEXTO.cargar)}</p>`;
  await cargar();
  return { recargar: cargar };
}

export async function iniciarPeticionCentro({ raiz = document.querySelector("#aplicacion"), cliente = pedir } = {}) {
  if (new URLSearchParams(globalThis.location?.search || "").get("vista") === "rrhh") {
    return iniciarPeticionesCentroRRHH({ raiz, cliente });
  }
  if (!raiz) throw new TypeError("falta la raíz de la aplicación");
  let contexto; let peticiones = []; let peticion = null; let modo = "bandeja";
  // Expedientes de las peticiones del centro, tal como los publica la bandeja de
  // incorporaciones de esta página (solo si el perfil puede consultarla).
  let expedientes = new Map();
  raiz.ownerDocument?.addEventListener?.(EVENTO_EXPEDIENTES_CENTRO, (evento) => {
    const lista = Array.isArray(evento?.detail) ? evento.detail : [];
    expedientes = new Map(lista.filter((e) => typeof e?.peticion_ref === "string" && typeof e.destino === "string"
      && typeof e.numero_visible === "string").map((e) => [e.peticion_ref, e]));
    if (!ocupado && modo === "bandeja" && contexto) dibujar();
  });
  let estado = null; let recibo = null; let mensaje = "";
  let ocupado = false; let operacionPendiente = null; let resultadoIncierto = false; let motivo = ""; let confirmado = false;
  const retirarDatos = (error) => {
    const confirmada = Boolean(recibo);
    resultadoIncierto = resultadoIncierto || Boolean(operacionPendiente);
    operacionPendiente = null;
    contexto = undefined; peticiones = []; peticion = null; estado = null; recibo = null;
    motivo = ""; confirmado = false;
    modo = esDenegacion(error) ? "denegado" : "sin_verificar";
    mensaje = `${modo === "denegado" ? TEXTO.accesoDenegado : TEXTO.lecturaFallida}${confirmada ? ` ${TEXTO.operacionConfirmadaOculta}` : ""}${resultadoIncierto ? ` ${TEXTO.operacionInciertaOculta}` : ""}`;
  };
  const dibujar = () => {
    const activo = raiz.ownerDocument?.activeElement;
    const enfocado = raiz.contains?.(activo) ? activo.dataset?.seleccionar || activo.dataset?.pcIrExpediente || "" : "";
    raiz.innerHTML = renderizarPeticionCentro({ contexto, peticiones, peticion, modo, estado, recibo, mensaje, motivo, confirmado, expedientes });
    if (enfocado) raiz.querySelector?.(`[data-seleccionar="${selectorSeguro(enfocado)}"], [data-pc-ir-expediente="${selectorSeguro(enfocado)}"]`)?.focus?.();
    raiz.setAttribute("aria-busy", String(ocupado));
    if (ocupado) {
      raiz.querySelectorAll("button, input, select, textarea").forEach((control) => { control.disabled = true; });
      raiz.insertAdjacentHTML("afterbegin", `<p class="pc-aviso" role="status">${esc(TEXTO.enviando)}</p>`);
    }
  };
  const cargarBandeja = async () => {
    const bandeja = await cliente(RUTAS.bandeja);
    if (!bandeja || !Array.isArray(bandeja.peticiones) || bandeja.peticiones.length > 50
      || bandeja.peticiones.some((p) => !p?.referencia || !p.solicitud || ![1, 2].includes(p.version)
        || !["pendiente_ratificacion", "ratificada"].includes(p.estado))) throw new Error(TEXTO.error);
    peticiones = bandeja.peticiones;
    peticion = peticiones.find((p) => p.referencia === (recibo?.peticion_ref || peticion?.referencia)) || null;
  };
  const cargar = async () => {
    if (ocupado || operacionPendiente) return;
    ocupado = true; mensaje = "";
    try {
      const nuevo = await cliente(RUTAS.contexto);
      if (!nuevo?.actor?.referencia || nuevo.actor.puede_presentar === nuevo.actor.puede_ratificar) throw new Error(TEXTO.error);
      validarCatalogosAlta(nuevo.catalogos);
      contexto = nuevo;
      await cargarBandeja();
      if (resultadoIncierto) {
        contexto = undefined; peticiones = []; peticion = null;
        modo = "resultado_incierto"; mensaje = TEXTO.operacionInciertaVerificada;
      } else if (modo === "denegado" || modo === "sin_verificar") modo = "bandeja";
    } catch (error) {
      retirarDatos(error);
    } finally { ocupado = false; dibujar(); }
  };
  const ejecutar = async (comando) => {
    if (ocupado || resultadoIncierto) return;
    const cuerpo = structuredClone(comando);
    ocupado = true; mensaje = ""; dibujar();
    try {
      recibo = await registrarOperacionPeticionCentro(cliente, cuerpo, contexto.actor.referencia);
      operacionPendiente = null; modo = "bandeja";
      // El fallo de una consulta posterior nunca convierte un recibo válido en
      // escritura pendiente ni invita a registrar otra vez.
      try { await cargarBandeja(); } catch (error) { retirarDatos(error); }
    } catch (error) {
      if (esDenegacion(error)) {
        retirarDatos(error);
      } else if (error.indeterminado) {
        operacionPendiente = cuerpo; modo = "pendiente"; mensaje = TEXTO.estadoPendiente;
      } else {
        operacionPendiente = null;
        modo = cuerpo.operacion === "presentar" ? "formulario" : "bandeja";
        mensaje = error.status === 409 ? TEXTO.conflicto : TEXTO.error;
        if (error.status === 409) {
          modo = "bandeja";
          try { await cargarBandeja(); } catch (lecturaError) { retirarDatos(lecturaError); }
        }
      }
    } finally { ocupado = false; dibujar(); }
  };
  raiz.addEventListener("click", async (event) => {
    const irExpediente = event.target.closest?.("[data-pc-ir-expediente]");
    if (irExpediente?.dataset?.pcIrExpediente) {
      const destino = raiz.ownerDocument?.getElementById?.(irExpediente.dataset.pcIrExpediente);
      if (!destino) return;
      event.preventDefault();
      destino.scrollIntoView?.({ block: "center" });
      destino.focus?.({ preventScroll: true });
      return;
    }
    const control = event.target.closest?.("[data-accion], [data-seleccionar], [data-ct-accion], [data-ct-enfocar]");
    if (!control || ocupado || modo === "denegado" || modo === "resultado_incierto"
      || (modo === "sin_verificar" && control.dataset.accion !== "recargar")
      || (operacionPendiente && control.dataset.accion !== "reintentar")) return;
    event.preventDefault();
    if (control.dataset.ctEnfocar) { raiz.querySelector(`#ct-${control.dataset.ctEnfocar}`)?.focus(); return; }
    const accion = control.dataset.accion || ({ volver: "editar", confirmar: "confirmar-presentar" })[control.dataset.ctAccion];
    try {
    if (control.dataset.seleccionar) { peticion = peticiones.find((p) => p.referencia === control.dataset.seleccionar) || null; dibujar(); return; }
    if (accion === "nueva" && contexto?.actor.puede_presentar) {
      modo = "formulario"; recibo = null;
      estado = estadoBase(contexto.catalogos, { ...crearBorradorAlta(), centro_ref: contexto.catalogos.centros[0]?.referencia || "" });
      mensaje = ""; dibujar(); return;
    }
    if (accion === "recargar") { await cargar(); return; }
    if (accion === "volver-contratacion") { globalThis.location.href = "/portal-empleado/#contratacion-temporal"; return; }
    if (accion === "editar") { modo = "formulario"; dibujar(); return; }
    if (accion === "abrir-ratificacion" && contexto?.actor.puede_ratificar && peticion?.version === 1) { modo = "ratificacion"; recibo = null; motivo = ""; confirmado = false; dibujar(); return; }
    if (accion === "cancelar-ratificacion") { modo = "bandeja"; dibujar(); return; }
    if (accion === "reintentar" && operacionPendiente) { await ejecutar(operacionPendiente); return; }
    if (accion === "confirmar-presentar" && modo === "revision" && contexto?.actor.puede_presentar) {
      const comando = crearComandoAlta(estado.borrador, contexto.catalogos, claveUUID());
      await ejecutar({ operacion: "presentar", ...comando }); return;
    }
    if (accion === "confirmar-ratificar" && modo === "ratificacion" && contexto?.actor.puede_ratificar) {
      motivo = raiz.querySelector("[name=motivo_ratificacion]")?.value.trim() || "";
      confirmado = raiz.querySelector("[name=confirmacion_ratificacion]")?.checked === true;
      if (!motivo || !confirmado || !peticion || new TextEncoder().encode(motivo).length > 1000 || /\p{Cc}/u.test(motivo)) { mensaje = TEXTO.motivoInvalido; dibujar(); return; }
      await ejecutar({ operacion: "ratificar", clave_idempotencia: claveUUID(), peticion_ref: peticion.referencia, version_esperada: 1, motivo });
    }
    } catch { mensaje = TEXTO.error; dibujar(); }
  });
  raiz.addEventListener("submit", (event) => {
    if (!event.target.matches("[data-ct-form]")) return;
    event.preventDefault();
    if (ocupado || operacionPendiente || resultadoIncierto || !contexto || ["denegado", "sin_verificar"].includes(modo)) return;
    const borrador = extraerBorradorPeticionCentro(event.target);
    const validacion = validarBorradorAlta(borrador, contexto.catalogos);
    estado = { ...estadoBase(contexto.catalogos, borrador), errores: validacion.errores, fase: validacion.valido ? "revision" : "edicion" };
    modo = validacion.valido ? "revision" : "formulario"; dibujar();
    raiz.querySelector(validacion.valido ? "#ct-revision-titulo" : "[data-ct-error-general]")?.focus();
  });
  raiz.addEventListener("change", (event) => {
    const campo = event.target.name;
    const formulario = event.target.closest?.("[data-ct-form]");
    if (ocupado || operacionPendiente || resultadoIncierto || !contexto || ["denegado", "sin_verificar"].includes(modo)
      || !formulario || !["centro_ref", "categoria_ref", "rc_existe"].includes(campo)) return;
    const borrador = extraerBorradorPeticionCentro(formulario);
    if (campo === "centro_ref") borrador.contacto_ref = "";
    if (campo === "categoria_ref") borrador.grupo_subgrupo = "";
    estado = estadoBase(contexto.catalogos, borrador); dibujar();
    raiz.querySelector(campo === "rc_existe" ? `[name="rc_existe"][value="${borrador.rc_existe ? "si" : "no"}"]` : `#ct-${campo}`)?.focus();
  });
  globalThis.addEventListener?.("beforeunload", (event) => { if (ocupado || operacionPendiente) { event.preventDefault(); event.returnValue = ""; } });
  raiz.innerHTML = `<p class="pc-cargando">${esc(TEXTO.cargar)}</p>`;
  await cargar();
  return { recargar: cargar };
}

/**
 * Textos de la ayuda «?» de la petición del centro.
 *
 * La ayuda deja claro que identificarse con certificado no es firmar: la
 * petición y su ratificación quedan registradas, pero el circuito de firma
 * electrónica sigue pendiente del procedimiento corporativo.
 */
export const MENSAJES_AYUDA_PETICIONES_CENTRO_ES = Object.freeze({
  pc_ayuda_abrir: "Ayuda sobre la petición del centro",
  pc_ayuda_titulo: "Petición del centro: qué hace y qué no hace",
  pc_ayuda_certificado_titulo: "¿Entrar con certificado es firmar?",
  pc_ayuda_certificado: "No. El certificado solo sirve para identificarle al entrar. Presentar o ratificar una petición no firma electrónicamente ningún documento.",
  pc_ayuda_registro_titulo: "¿Qué queda registrado?",
  pc_ayuda_registro: "La petición y su ratificación quedan registradas con su autor, su fecha y un recibo. La firma electrónica todavía no está disponible: se incorporará cuando se establezca el circuito de firma corporativo.",
  pc_ayuda_despues_titulo: "¿Qué pasa después?",
  pc_ayuda_despues: "Cuando la petición está ratificada, llega a la bandeja de Recursos Humanos. RRHH revisa los datos y, si procede, crea con ellos el expediente de contratación. El centro no tiene que volver a enviarla.",
  pc_ayuda_cerrar: "Cerrar",
});

/** Traduce una clave de la ayuda; una clave desconocida nunca muestra texto inventado. */
function traducirAyudaPeticionCentro(clave, mensajes = MENSAJES_AYUDA_PETICIONES_CENTRO_ES) {
  return Object.hasOwn(mensajes, clave) ? mensajes[clave] : "";
}

/**
 * Conecta el botón «?» con su diálogo de ayuda. Los textos llegan del catálogo
 * i18n; el diálogo nativo aporta foco atrapado y cierre con Escape, y al
 * cerrarse el foco vuelve al botón que lo abrió.
 */
export function instalarAyudaPeticionCentro(doc) {
  const boton = doc?.getElementById?.("pc-ayuda-abrir");
  const dialogo = doc?.getElementById?.("pc-ayuda");
  if (!boton || !dialogo) return false;
  for (const elemento of dialogo.querySelectorAll("[data-i18n-ayuda]")) {
    elemento.textContent = traducirAyudaPeticionCentro(elemento.dataset.i18nAyuda);
  }
  const etiqueta = traducirAyudaPeticionCentro("pc_ayuda_abrir");
  boton.setAttribute("aria-label", etiqueta);
  boton.setAttribute("title", etiqueta);
  boton.addEventListener("click", () => {
    if (typeof dialogo.showModal === "function") dialogo.showModal();
    else dialogo.setAttribute("open", "");
    dialogo.querySelector("#pc-ayuda-cerrar")?.focus?.();
  });
  dialogo.querySelector("#pc-ayuda-cerrar")?.addEventListener("click", () => {
    if (typeof dialogo.close === "function") dialogo.close();
    else { dialogo.removeAttribute("open"); boton.focus?.(); }
  });
  dialogo.addEventListener("close", () => boton.focus?.());
  return true;
}

if (typeof document !== "undefined" && document.querySelector("#aplicacion")) {
  aplicarTextosPortal(document);
  aplicarIdiomaDocumento(document);
  instalarValidacionI18n(document);
  instalarAyudaPeticionCentro(document);
  iniciarPeticionCentro();
}
