import {
  crearBorradorAlta,
  crearComandoPeticionCentro,
  validarBorradorAlta,
  validarCatalogosAlta,
  numeroExpedienteMOADValido,
} from "../modulos/contratacion-temporal/contrato.js?v=20261008-alta-circular-v3";
import { extraerBorrador, formulario as renderizarFormularioPuro,
  revision as renderizarRevisionPura } from "../modulos/contratacion-temporal/alta-renderer-puro.js?v=20261008-alta-capacidad-v3";
import { IDIOMA_POR_DEFECTO } from "../../comun/idioma.js";
import { IDIOMA_EFECTIVO_PETICIONES_CENTRO, LOCALIZACION_PETICIONES_CENTRO, MENSAJES_AYUDA_PETICIONES_CENTRO,
  TEXTOS_LOCALES_PETICIONES_CENTRO, prepararAnalisisPeticionesCentro,
  traducirPeticionesCentro } from "./i18n-peticiones-centro.js?v=20261007-pc-recuperacion-v1";

const RUTAS = Object.freeze({
  contexto: "/api/vec/contratacion-temporal/peticiones-centro/contexto",
  bandeja: "/api/vec/contratacion-temporal/peticiones-centro/bandeja",
  operaciones: "/api/vec/contratacion-temporal/peticiones-centro/operaciones",
  rrhh: "/api/vec/contratacion-temporal/peticiones-centro/rrhh",
  catalogosAlta: "/api/vec/contratacion-temporal/catalogos-alta",
});
const MAX_BODY = 2 * 1024 * 1024;
const TIMEOUT_MS = 15_000;
const traducirCentro = traducirPeticionesCentro;
const TEXTO = TEXTOS_LOCALES_PETICIONES_CENTRO;

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

// Una lectura que choca con otra simultánea en el servidor puede devolver 503
// transitorio: se repite dos veces con una espera breve. Solo lecturas (GET),
// que no tienen efectos; las escrituras nunca se repiten solas.
export async function pedir(ruta, opciones = {}) {
  const esLectura = (opciones.method ?? "GET") === "GET";
  for (let intento = 0; ; intento += 1) {
    try {
      return await pedirUnaVez(ruta, opciones);
    } catch (error) {
      if (!esLectura || error?.status !== 503 || intento >= 2 || opciones.signal?.aborted) throw error;
      await new Promise((resolver) => setTimeout(resolver, 300 * (intento + 1)));
    }
  }
}

async function pedirUnaVez(ruta, { method = "GET", cuerpo, signal } = {}) {
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

const LOCALIZACION = LOCALIZACION_PETICIONES_CENTRO;

function fecha(valor, hora = false) {
  if (!valor || !Number.isFinite(Date.parse(valor))) return "—";
  return new Intl.DateTimeFormat(LOCALIZACION, { dateStyle: "medium", ...(hora ? { timeStyle: "medium" } : {}), timeZone: hora ? "Europe/Madrid" : "UTC" }).format(new Date(valor));
}

const fechaValida = (valor) => Boolean(valor) && Number.isFinite(Date.parse(valor));

// Nombre legible de la petición: número de expediente si ya lo tiene, o la
// fecha en que se creó. Nunca la referencia técnica.
function nombrePeticion(peticion, expediente) {
  if (expediente?.numero_visible) return traducirCentro("pc_expediente_enlace", { numero: expediente.numero_visible });
  return fechaValida(peticion?.creada_en) ? traducirCentro("pc_peticion_del", { fecha: fecha(peticion.creada_en) }) : traducirCentro("pc_peticion_sin_fecha");
}

// Un código («centro-520», «cen_…») no se enseña: se usa la etiqueta del
// catálogo, la unidad de la persona o «Su centro».
const PARECE_CODIGO = /^[a-z]+[-_:][\w:-]*\d[\w:-]*$/u;
function nombreCentro(referencia, contexto) {
  const etiqueta = contexto?.catalogos?.centros?.find((v) => v.referencia === referencia)?.etiqueta;
  if (etiqueta) return etiqueta;
  if (!contexto) return referencia || "—";
  const unidad = contexto.actor?.centro;
  return unidad && !PARECE_CODIGO.test(unidad) ? unidad : traducirCentro("pc_su_centro");
}

function estadoPeticion(estado) {
  return traducirCentro(estado === "ratificada" ? "pc_estado_ratificada" : "pc_estado_pendiente");
}

function periodoLegible(periodo) {
  const inicio = fechaValida(periodo?.inicio); const fin = fechaValida(periodo?.fin);
  if (inicio && fin) return traducirCentro("pc_periodo_desde_hasta", { inicio: fecha(periodo.inicio), fin: fecha(periodo.fin) });
  if (inicio && periodo?.causa_fin) return traducirCentro("pc_periodo_con_causa", {
    inicio: fecha(periodo.inicio), causa: traducirCentro(`causa_fin_${periodo.causa_fin}`),
  });
  if (inicio) return traducirCentro("pc_periodo_desde", { inicio: fecha(periodo.inicio) });
  if (fin) return traducirCentro("pc_periodo_hasta", { fin: fecha(periodo.fin) });
  return traducirCentro("pc_sin_fechas");
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
  const centro = catalogos?.centros?.find((v) => v.referencia === s.centro_ref);
  const etiqueta = (opciones, referencia) => opciones?.find((v) => v.referencia === referencia)?.etiqueta || referencia || "—";
  const etiquetaMotivo = catalogos?.motivos?.find((v) => v.clave === s.motivo_clave)?.etiqueta || "—";
  const rc = s.rc?.existe
    ? `${s.rc.numero} · ${fecha(s.rc.fecha)} · ${new Intl.NumberFormat("es-ES", { style: "currency", currency: "EUR" }).format(s.rc.importe.centimos / 100)} · ${s.rc.documento_ref}`
    : traducirCentro("ct_txt_sin_retencion_de_credito_aportada");
  const filas = [...(contexto ? [] : [[TEXTO.peticionRef, peticion.referencia], [TEXTO.version, peticion.version]]),
    [TEXTO.estado, contexto ? estadoPeticion(peticion.estado) : peticion.estado],
    [traducirCentro("ct_txt_centro"), nombreCentro(s.centro_ref, contexto)], [traducirCentro("ct_txt_contacto"), etiqueta(centro?.contactos, s.contacto_ref)],
    [traducirCentro("ct_txt_categoria"), etiqueta(catalogos?.categorias, s.categoria_ref)], [traducirCentro("ct_txt_grupo_o_subgrupo"), s.grupo_subgrupo], [traducirCentro("ct_txt_motivo"), etiquetaMotivo],
    [traducirCentro("ct_txt_detalle"), s.detalle], [traducirCentro("ct_txt_periodo"), periodoLegible(s.periodo)], [traducirCentro("ct_txt_observaciones"), s.observaciones || "—"],
    [traducirCentro("ct_txt_retencion_de_credito"), rc], [traducirCentro("ct_txt_documentos_aportados"), (s.documentos_adjuntos || []).map((ref) => etiqueta(catalogos?.documentos, ref)).join(" · ") || traducirCentro("ct_txt_ninguno")],
    [TEXTO.solicitanteDatos, solicitante?.puesto_ref === c?.puesto_ref
      ? `${solicitante.nombre} · ${solicitante.cargo}`
      : contexto ? (c?.actor_ref === actor?.referencia ? [actor?.nombre, actor?.cargo].filter(Boolean).join(" · ") || "—" : "—")
        : `${c?.actor_ref || "—"} · ${c?.puesto_ref || traducirCentro("ct_txt_cargo_resuelto_por_identidad")}`],
    [traducirCentro("ct_txt_creada_en"), fecha(peticion.creada_en, true)]];
  if (peticion.estado === "ratificada") {
    const rat = peticion.configuracion?.ratificador;
    const etiquetaRat = contexto?.intervinientes?.[rat?.actor_ref];
    filas.push([traducirCentro("ct_txt_ratificador_y_cargo"), etiquetaRat && rat && etiquetaRat.puesto_ref === rat.puesto_ref ? `${etiquetaRat.nombre} · ${etiquetaRat.cargo}` : contexto ? "—" : `${rat?.actor_ref || "—"} · ${rat?.puesto_ref || "—"}`],
      [traducirCentro("ct_txt_motivo_de_ratificacion"), peticion.motivo_ratificacion], [traducirCentro("ct_txt_ratificada_en"), fecha(peticion.ratificada_en, true)]);
  }
  return camposDetalle(filas);
}

function reciboHTML(recibo) {
  return `<section class="pc-panel pc-recibo" role="status"><h2>${esc(TEXTO.exito)}</h2><dl>
    <dt>${esc(TEXTO.estado)}</dt><dd>${esc(estadoPeticion(recibo.estado))}</dd>
    <dt>${esc(TEXTO.registrado)}</dt><dd>${esc(fecha(recibo.registrado_en, true))}</dd></dl>
    <p>${textoCT("pc_recibo_nota")}</p></section>`;
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
  return ` <a class="pc-enlace-expediente" href="#${esc(expediente.destino)}" data-pc-ir-expediente="${esc(expediente.destino)}" aria-label="${textoCT("pc_expediente_enlace_centro_aria", { peticion: nombrePeticion(peticion), numero: expediente.numero_visible })}">${textoCT("pc_expediente_enlace", { numero: expediente.numero_visible })}</a>`;
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

export function renderizarPeticionesCentroRRHH({ peticiones = [], entrega = null, modo = "bandeja", confirmado = false, recibo = null, mensaje = "", numeroMOAD = "", politicaNumero = null, errorNumero = false } = {}) {
  const peticion = entrega?.peticion;
  const cabecera = `<section class="pc-cabecera"><p class="sobrelinea">${esc(TEXTO.rrhhSobrelinea)}</p><h1>${esc(TEXTO.rrhhTitulo)}</h1><p>${esc(TEXTO.rrhhDescripcion)}</p></section>`;
  const error = mensaje ? `<p class="pc-error" role="alert">${esc(mensaje)}</p>` : "";
  if (["denegado", "sin_verificar", "resultado_incierto"].includes(modo)) return vistaSinDatos(cabecera, modo, mensaje, "recargar-rrhh");
  if (modo === "confirmar") return `${cabecera}${error}<section class="pc-panel pc-detalle"><h2>${esc(TEXTO.rrhhConfirmar)}</h2>${detallePeticion(peticion, null)}<div class="ct-campo"><label for="pc-numero-moad">${textoCT("numero_moad_obligatorio")}</label><input id="pc-numero-moad" name="numero_expediente_moad" type="text" required maxlength="45" autocomplete="off" value="${esc(numeroMOAD)}" ${politicaNumero?.ejemplo ? `placeholder="${textoCT("numero_moad_ejemplo", { ejemplo: politicaNumero.ejemplo })}"` : ""}${errorNumero ? ' aria-invalid="true" aria-describedby="pc-numero-moad-error"' : ""}>${errorNumero ? `<span class="ct-error-campo" id="pc-numero-moad-error">${textoCT("error_numero_moad")}</span>` : ""}</div><p class="pc-aviso">${esc(TEXTO.rrhhAviso)}</p><label class="pc-confirmacion"><input type="checkbox" name="confirmacion-alta-rrhh"${confirmado ? " checked" : ""}> ${esc(TEXTO.rrhhConfirmacion)}</label><div class="pc-acciones"><button type="button" class="boton-secundario" data-accion="cancelar-alta-rrhh">${esc(TEXTO.cancelar)}</button><button type="button" class="boton-primario" data-accion="confirmar-alta-rrhh">${esc(TEXTO.rrhhConfirmar)}</button></div></section>`;
  if (modo === "pendiente") return `${cabecera}<section class="pc-panel pc-pendiente" role="status"><h2>${esc(TEXTO.estadoPendiente)}</h2><p>${esc(TEXTO.rrhhAviso)}</p><div class="pc-acciones"><button type="button" class="boton-primario" data-accion="reintentar-alta-rrhh">${esc(traducirCentro("ct_txt_reintentar_la_misma_operacion"))}</button></div></section>`;
  const detalle = `<aside class="pc-panel pc-detalle"><h2>${esc(TEXTO.detalle)}</h2>${detallePeticion(peticion, null)}${entrega?.recibo_alta && !recibo ? reciboAltaRRHHHTML(entrega.recibo_alta) : ""}${["pendiente", "preparada"].includes(entrega?.estado_entrega) ? `<div class="pc-acciones"><button type="button" class="boton-primario" data-accion="abrir-alta-rrhh">${esc(entrega.estado_entrega === "preparada" ? TEXTO.rrhhCompletar : TEXTO.rrhhConfirmar)}</button>${entrega.estado_entrega === "preparada" ? `<button type="button" class="boton-secundario" data-accion="recuperar-alta-anterior">${textoCT("numero_moad_recuperar_alta_anterior")}</button>` : ""}</div>` : ""}</aside>`;
  return `${cabecera}${error}${recibo ? reciboAltaRRHHHTML(recibo) : ""}<div class="pc-layout"><section class="pc-panel"><h2>${esc(TEXTO.rrhhTitulo)}</h2>${tablaRRHH(peticiones, peticion?.referencia)}<p>${textoCT("ct_txt_ultimas_50_peticiones_visibles_para_recursos_hum")}</p><div class="pc-acciones"><button type="button" class="boton-secundario" data-accion="recargar-rrhh">${esc(TEXTO.recargar)}</button><a class="boton-secundario" href="/portal-empleado/#contratacion-temporal">${esc(TEXTO.volver)}</a></div></section>${detalle}</div>`;
}

export async function registrarAltaRRHH(cliente, comando) {
  try {
    const resultado = await cliente(RUTAS.rrhh, { method: "POST", cuerpo: { peticion_ref: comando.peticion_ref, version_esperada: 2, ...(Object.hasOwn(comando, "numero_expediente_moad") ? { numero_expediente_moad: comando.numero_expediente_moad } : {}) } });
    const recibo = resultado?.recibo_alta;
    if (!resultado?.peticion?.referencia || resultado.peticion.referencia !== comando.peticion_ref
      || resultado.estado_entrega !== "confirmada" || !recibo?.expediente_ref || !recibo.recibo_ref || !recibo.confirmada_en
      || (Object.hasOwn(comando, "numero_expediente_moad") && recibo.numero_visible !== comando.numero_expediente_moad)
      || !Number.isFinite(Date.parse(recibo.confirmada_en))) throw new Error(TEXTO.rrhhError);
    return resultado;
  } catch (error) {
    if ([400, 401, 403, 409, 422].includes(error?.status)) throw error;
    throw Object.assign(new Error(TEXTO.estadoPendiente), { indeterminado: true });
  }
}

function tabla(peticiones, seleccionada, expedientes = new Map(), contexto = null) {
  if (!peticiones.length) return `<p>${textoCT("pc_sin_peticiones")}</p>`;
  return `<div class="pc-tabla-wrap"><table class="pc-tabla"><caption class="solo-lectura">${textoCT("pc_lista_centro")}</caption><thead><tr><th>${textoCT("pc_col_peticion")}</th><th>${textoCT("ct_txt_centro")}</th><th>${textoCT("ct_txt_periodo")}</th><th>${textoCT("ct_txt_estado")}</th><th>${textoCT("ct_txt_accion")}</th></tr></thead><tbody>${peticiones.map((p) => {
    const nombre = nombrePeticion(p);
    return `<tr${p.referencia === seleccionada ? ' aria-selected="true"' : ""}><td>${esc(nombre)}</td><td>${esc(nombreCentro(p.solicitud?.centro_ref, contexto))}</td><td>${esc(periodoLegible(p.solicitud?.periodo))}</td><td><span class="pc-estado pc-estado-${esc(p.estado === "ratificada" ? "ratificada" : "pendiente")}">${esc(estadoPeticion(p.estado))}</span>${enlaceExpedienteCentro(p, expedientes.get(p.referencia))}</td><td><button type="button" data-seleccionar="${esc(p.referencia)}" aria-label="${esc(`${traducirCentro("pc_ver")}: ${nombre}`)}">${textoCT("pc_ver")}</button></td></tr>`;
  }).join("")}</tbody></table></div>`;
}

function formularioHTML(contexto, estado, revision) {
  const contenido = revision ? renderizarRevisionPura(estado, traducirCentro, LOCALIZACION)
    : renderizarFormularioPuro(estado, traducirCentro);
  return `<section class="pc-panel ct-alta"><h2>${esc(TEXTO.solicitante)}</h2><p class="pc-aviso">${esc(TEXTO.confirmarPregunta)}</p>${contenido}<button type="button" class="boton-secundario" data-accion="cancelar-ratificacion">${esc(TEXTO.cancelar)}</button></section>`;
}

export function renderizarPeticionCentro({ contexto, peticiones = [], peticion = null, modo = "bandeja", estado = null, recibo = null, mensaje = "", motivo = "", confirmado = false, expedientes = new Map() } = {}) {
  const actor = contexto?.actor;
  const esSolicitante = actor?.puede_presentar && !actor?.puede_ratificar;
  if (["denegado", "sin_verificar", "resultado_incierto"].includes(modo)) {
    const cabeceraSegura = `<section class="pc-cabecera"><p class="sobrelinea">${esc(TEXTO.sobrelinea)}</p><h1>${esc(TEXTO.titulo)}</h1></section>`;
    return vistaSinDatos(cabeceraSegura, modo, mensaje, "recargar");
  }
  const persona = [actor?.nombre, actor?.cargo, actor?.centro && !PARECE_CODIGO.test(actor.centro) ? actor.centro : ""].filter(Boolean).join(" · ");
  const cabecera = `<section class="pc-cabecera"><p class="sobrelinea">${textoCT("pc_sobrelinea")}</p><h1>${textoCT("pc_titulo")}</h1><p>${textoCT("pc_descripcion")}</p><div class="pc-etiquetas"><span class="pc-etiqueta">${esc(TEXTO.pendienteEntrada)}</span></div>${persona ? `<p>${esc(persona)}</p>` : ""}</section>`;
  const error = mensaje ? `<p class="pc-error" role="alert">${esc(mensaje)}</p>` : "";
  if (modo === "formulario") return `${cabecera}${error}${formularioHTML(contexto, estado, false)}`;
  if (modo === "revision") return `${cabecera}${error}${formularioHTML(contexto, estado, true)}`;
  if (modo === "ratificacion") return `${cabecera}${error}<section class="pc-panel pc-detalle"><h2>${esc(TEXTO.ratificador)}</h2>${detallePeticion(peticion, contexto)}<p class="pc-aviso">${esc(TEXTO.confirmarPregunta)}</p><label for="motivo-ratificacion">${esc(TEXTO.motivo)}</label><input id="motivo-ratificacion" name="motivo_ratificacion" value="${esc(motivo)}" maxlength="1000" required aria-describedby="motivo-ratificacion-ayuda"><small id="motivo-ratificacion-ayuda">${esc(TEXTO.motivoAyuda)}</small><label class="pc-confirmacion"><input type="checkbox" name="confirmacion_ratificacion"${confirmado ? " checked" : ""}> ${esc(TEXTO.confirmacion)}</label><div class="pc-acciones"><button type="button" class="boton-secundario" data-accion="cancelar-ratificacion">${esc(TEXTO.cancelar)}</button><button type="button" class="boton-primario" data-accion="confirmar-ratificar">${esc(TEXTO.confirmarRatificar)}</button></div></section>`;
  if (modo === "pendiente") return `${cabecera}<section class="pc-panel pc-pendiente" role="status"><h2>${esc(TEXTO.estadoPendiente)}</h2><p>${esc(TEXTO.peticionNoEnviada)}</p><div class="pc-acciones"><button type="button" class="boton-primario" data-accion="reintentar">${esc(traducirCentro("ct_txt_reintentar_la_misma_operacion"))}</button></div></section>`;
  // El recuadro de datos solo aparece cuando hay una petición elegida.
  const detalle = peticion ? `<aside class="pc-panel pc-detalle" aria-labelledby="pc-detalle-titulo"><h2 id="pc-detalle-titulo">${textoCT("pc_detalle_titulo")}: ${esc(nombrePeticion(peticion, expedientes.get(peticion.referencia)))}</h2>${detallePeticion(peticion, contexto)}<div class="pc-acciones">${peticion.estado === "pendiente_ratificacion" && peticion.version === 1 && actor?.puede_ratificar ? `<button type="button" class="boton-primario" data-accion="abrir-ratificacion">${esc(TEXTO.ratificador)}</button>` : ""}<button type="button" class="boton-secundario" data-accion="cerrar-detalle">${textoCT("pc_cerrar_detalle")}</button></div></aside>` : "";
  return `${cabecera}${error}${recibo ? reciboHTML(recibo) : ""}<div class="pc-layout"><section class="pc-panel"><h2>${textoCT(esSolicitante ? "pc_lista_centro" : "pc_lista_ratificar")}</h2>${tabla(peticiones, peticion?.referencia, expedientes, contexto)}<p>${textoCT("pc_ultimas")}</p><div class="pc-acciones">${esSolicitante ? `<button type="button" class="boton-primario" data-accion="nueva">${textoCT("pc_nueva")}</button>` : ""}<button type="button" class="boton-secundario" data-accion="recargar">${textoCT("pc_actualizar")}</button><button type="button" class="boton-secundario" data-accion="volver-contratacion">${textoCT("pc_volver")}</button></div></section>${detalle}</div>`;
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
  raiz.setAttribute?.("lang", IDIOMA_EFECTIVO_PETICIONES_CENTRO);
  let peticiones = []; let entrega = null; let modo = "bandeja"; let recibo = null; let mensaje = "";
  let ocupado = false; let operacionPendiente = null; let resultadoIncierto = false; let confirmado = false;
  let numeroMOAD = ""; let politicaNumero = null; let errorNumero = false;
  const retirarDatos = (error) => {
    const confirmada = Boolean(recibo);
    resultadoIncierto = resultadoIncierto || Boolean(operacionPendiente);
    operacionPendiente = null;
    peticiones = []; entrega = null; recibo = null; confirmado = false;
    numeroMOAD = ""; politicaNumero = null; errorNumero = false;
    modo = esDenegacion(error) ? "denegado" : "sin_verificar";
    mensaje = `${modo === "denegado" ? TEXTO.accesoDenegado : TEXTO.lecturaFallida}${confirmada ? ` ${TEXTO.operacionConfirmadaOculta}` : ""}${resultadoIncierto ? ` ${TEXTO.operacionInciertaOculta}` : ""}`;
  };
  const dibujar = () => {
    raiz.innerHTML = renderizarPeticionesCentroRRHH({ peticiones, entrega, modo, confirmado, recibo, mensaje, numeroMOAD, politicaNumero, errorNumero });
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
      if (peticiones.some((item) => item.peticion?.solicitud?.periodo?.causa_fin)) {
        await prepararAnalisisPeticionesCentro();
      }
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
      else if (error.status === 422 && !Object.hasOwn(comando, "numero_expediente_moad")) {
        operacionPendiente = null; modo = "bandeja";
        mensaje = traducirCentro("numero_moad_recuperacion_no_disponible");
      }
      else if (error.status === 422) {
        operacionPendiente = null; modo = "confirmar"; errorNumero = true;
        mensaje = traducirCentro("numero_moad_formato_no_valido", { ejemplo: politicaNumero?.ejemplo ?? "" });
      }
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
    if (control.dataset.accion === "recuperar-alta-anterior" && entrega?.estado_entrega === "preparada") {
      await ejecutar({ peticion_ref: entrega.peticion.referencia, version_esperada: 2 });
      return;
    }
    if (control.dataset.accion === "abrir-alta-rrhh" && ["pendiente", "preparada"].includes(entrega?.estado_entrega)) {
      ocupado = true; mensaje = ""; dibujar();
      try {
        const catalogos = validarCatalogosAlta(await cliente(RUTAS.catalogosAlta));
        if (!catalogos.numero_expediente_moad) throw new Error();
        politicaNumero = catalogos.numero_expediente_moad;
        numeroMOAD = ""; errorNumero = false; modo = "confirmar"; confirmado = false; recibo = null;
      } catch (error) {
        if (esDenegacion(error)) retirarDatos(error);
        else mensaje = traducirCentro("numero_moad_no_disponible");
      } finally { ocupado = false; dibujar(); }
      return;
    }
    if (control.dataset.accion === "cancelar-alta-rrhh") { modo = "bandeja"; dibujar(); return; }
    if (control.dataset.accion === "reintentar-alta-rrhh" && operacionPendiente) { await ejecutar(operacionPendiente); return; }
    if (control.dataset.accion === "confirmar-alta-rrhh" && modo === "confirmar") {
      numeroMOAD = raiz.querySelector("[name=numero_expediente_moad]")?.value ?? "";
      errorNumero = !numeroExpedienteMOADValido(numeroMOAD);
      confirmado = raiz.querySelector("[name=confirmacion-alta-rrhh]")?.checked === true;
      if (errorNumero) { mensaje = traducirCentro("error_numero_moad"); dibujar(); raiz.querySelector("#pc-numero-moad")?.focus?.(); return; }
      if (!confirmado || !entrega?.peticion?.referencia) { mensaje = TEXTO.confirmacion; dibujar(); return; }
      await ejecutar({ peticion_ref: entrega.peticion.referencia, version_esperada: 2, numero_expediente_moad: numeroMOAD });
    }
  });
  raiz.innerHTML = `<p class="pc-cargando">${esc(TEXTO.cargar)}</p>`;
  await cargar();
  return { recargar: cargar };
}

export async function iniciarPeticionCentro({ raiz = document.querySelector("#aplicacion"), cliente = pedir,
  prepararAnalisis = prepararAnalisisPeticionesCentro } = {}) {
  if (new URLSearchParams(globalThis.location?.search || "").get("vista") === "rrhh") {
    return iniciarPeticionesCentroRRHH({ raiz, cliente });
  }
  if (!raiz) throw new TypeError("falta la raíz de la aplicación");
  raiz.setAttribute?.("lang", IDIOMA_EFECTIVO_PETICIONES_CENTRO);
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
  let generacionContexto = 0; let preparandoAnalisis = false;
  const retirarDatos = (error) => {
    generacionContexto += 1; preparandoAnalisis = false;
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
      raiz.insertAdjacentHTML("afterbegin", `<p class="pc-aviso" role="status">${esc(preparandoAnalisis
        ? TEXTO.preparandoFormulario : TEXTO.enviando)}</p>`);
    }
  };
  const cargarBandeja = async () => {
    const bandeja = await cliente(RUTAS.bandeja);
    if (!bandeja || !Array.isArray(bandeja.peticiones) || bandeja.peticiones.length > 50
      || bandeja.peticiones.some((p) => !p?.referencia || !p.solicitud || ![1, 2].includes(p.version)
        || !["pendiente_ratificacion", "ratificada"].includes(p.estado))) throw new Error(TEXTO.error);
    peticiones = bandeja.peticiones;
    if (peticiones.some((item) => item.solicitud?.periodo?.causa_fin)) {
      await prepararAnalisis();
    }
    peticion = peticiones.find((p) => p.referencia === (recibo?.peticion_ref || peticion?.referencia)) || null;
  };
  const cargar = async () => {
    if ((ocupado && !preparandoAnalisis) || operacionPendiente) return;
    generacionContexto += 1; preparandoAnalisis = false;
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
    if (accion === "nueva" && modo === "bandeja" && contexto?.actor.puede_presentar) {
      const contextoInicial = contexto;
      const actorInicial = contextoInicial.actor.referencia;
      const generacionInicial = generacionContexto;
      if (contextoInicial.catalogos.motivos.some((motivo) => motivo.causa_fin)) {
        preparandoAnalisis = true; ocupado = true; mensaje = ""; dibujar();
        try {
          await prepararAnalisis();
        } catch {
          if (generacionInicial === generacionContexto && preparandoAnalisis && raiz.isConnected !== false) {
            mensaje = TEXTO.analisisNoDisponible;
          }
          return;
        } finally {
          if (generacionInicial === generacionContexto && preparandoAnalisis) {
            preparandoAnalisis = false; ocupado = false;
            if (raiz.isConnected !== false) dibujar();
          }
        }
      }
      if (generacionInicial !== generacionContexto || contexto !== contextoInicial
        || contexto?.actor?.referencia !== actorInicial || contexto.actor.puede_presentar !== true
        || modo !== "bandeja" || raiz.isConnected === false) return;
      modo = "formulario"; recibo = null;
      estado = estadoBase(contextoInicial.catalogos, { ...crearBorradorAlta(), centro_ref: contextoInicial.catalogos.centros[0]?.referencia || "" });
      mensaje = ""; dibujar(); return;
    }
    if (accion === "recargar") { await cargar(); return; }
    if (accion === "volver-contratacion") { globalThis.location.href = "/portal-empleado/#contratacion-temporal"; return; }
    if (accion === "editar") { modo = "formulario"; dibujar(); return; }
    if (accion === "abrir-ratificacion" && contexto?.actor.puede_ratificar && peticion?.version === 1) { modo = "ratificacion"; recibo = null; motivo = ""; confirmado = false; dibujar(); return; }
    if (accion === "cancelar-ratificacion") { modo = "bandeja"; dibujar(); return; }
    if (accion === "cerrar-detalle") { peticion = null; dibujar(); return; }
    if (accion === "reintentar" && operacionPendiente) { await ejecutar(operacionPendiente); return; }
    if (accion === "confirmar-presentar" && modo === "revision" && contexto?.actor.puede_presentar) {
      const comando = crearComandoPeticionCentro(estado.borrador, contexto.catalogos, claveUUID());
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
    const borrador = extraerBorrador(event.target, false);
    const validacion = validarBorradorAlta(borrador, contexto.catalogos);
    estado = { ...estadoBase(contexto.catalogos, borrador), errores: validacion.errores, fase: validacion.valido ? "revision" : "edicion" };
    modo = validacion.valido ? "revision" : "formulario"; dibujar();
    raiz.querySelector(validacion.valido ? "#ct-revision-titulo" : "[data-ct-error-general]")?.focus();
  });
  raiz.addEventListener("change", (event) => {
    const campo = event.target.name;
    const formulario = event.target.closest?.("[data-ct-form]");
    if (ocupado || operacionPendiente || resultadoIncierto || !contexto || ["denegado", "sin_verificar"].includes(modo)
      || !formulario || !["centro_ref", "categoria_ref", "rc_existe", "motivo_clave", "fin"].includes(campo)) return;
    const borrador = extraerBorrador(formulario, false);
    if (campo === "centro_ref") borrador.contacto_ref = "";
    if (campo === "categoria_ref") borrador.grupo_subgrupo = "";
    if (campo === "motivo_clave" && contexto.catalogos.motivos.find(
      ({ clave }) => clave === borrador.motivo_clave)?.fecha_fin === "no_aplica") borrador.fin = "";
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
export const MENSAJES_AYUDA_PETICIONES_CENTRO_ES = IDIOMA_EFECTIVO_PETICIONES_CENTRO === IDIOMA_POR_DEFECTO
  ? MENSAJES_AYUDA_PETICIONES_CENTRO : undefined;

/** Traduce una clave de la ayuda; una clave desconocida nunca muestra texto inventado. */
function traducirAyudaPeticionCentro(clave, mensajes = MENSAJES_AYUDA_PETICIONES_CENTRO) {
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
