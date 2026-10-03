/**
 * Vista del paso de firma. CT118 conserva su historia de prueba, pero esta
 * vista no ejecuta acciones hasta que el servidor acredite el preflight R5.
 */

import { escaparHTML } from "./componentes-expedientes.js?v=20261002-ct-fin-modalidad-v1";
import { validarPreflightFirma } from "./preflight-firma-api.js";
import { huellaPDFFirmado } from "./firma-vec-api.js";
import { PERFILES_BORRADOR_RRHH } from "./cliente-http-informe-definitivo.js?v=20261002-ct-fin-moad-v1";

/** Une el circuito del catálogo con el estado real registrado. */
export function fusionarEstadoFirmas(circuito, estado) {
  if (!circuito || !estado || estado.huella_sha256 !== circuito.huella_sha256) return null;
  const documentos = [];
  for (const documento of circuito.documentos) {
    const real = estado.documentos.find((d) => d.documento === documento.documento);
    if (!real || real.pasos.length !== documento.pasos.length) return null;
    documentos.push(Object.freeze({
      ...documento,
      paso_pendiente: real.paso_pendiente,
      pasos: Object.freeze(documento.pasos.map((paso, i) => Object.freeze({
        ...paso, estado: real.pasos[i].estado, motivo_devolucion: real.pasos[i].motivo_devolucion ?? "",
        registrada_en: real.pasos[i].registrada_en ?? "",
        documento_custodiado: real.pasos[i].documento_custodiado ?? null,
      }))),
    }));
  }
  return Object.freeze({ ...circuito, documentos: Object.freeze(documentos),
    registro: Object.freeze({ verificacion: estado.verificacion_disponible }) });
}

/** Solo la composición explícita ofrece la consulta nominal; no concede firma. */
export function renderizarAccionesPaso(circuito, documento, paso, t) {
  if (!circuito.registro || circuito.acciones === false || documento.paso_pendiente !== paso.orden
    || !Object.hasOwn(PERFILES_BORRADOR_RRHH, documento.documento)) return "";
  const avisoId = `ct-firma-resultado-${documento.documento}-${paso.orden}`;
  const compuesto = circuito.preflight_compuesto === true;
  return `<div class="ct-circuito-acciones" data-ct-firma-controles data-ct-firma-documento="${escaparHTML(documento.documento)}" data-ct-firma-paso="${paso.orden}">
    ${compuesto ? `<button type="button" class="boton-primario" data-ct-firma-accion="comprobar" aria-describedby="${avisoId}">${escaparHTML(t("circuito_firma_comprobar"))}</button><button type="button" class="boton-secundario" data-ct-firma-accion="cancelar">${escaparHTML(t("circuito_firma_cancelar"))}</button>`
    : `<button type="button" class="boton-primario" disabled aria-describedby="${avisoId}">${escaparHTML(t("circuito_firma_firmar"))}</button>
    <button type="button" class="boton-secundario" disabled aria-describedby="${avisoId}">${escaparHTML(t("circuito_firma_devolver"))}</button>`}
    <p id="${avisoId}" class="ct-circuito-resultado" role="status" aria-live="polite" tabindex="-1" data-ct-firma-resultado>${escaparHTML(t(compuesto ? "circuito_firma_original_pendiente" : "circuito_firma_vec_bloqueada"))}</p>
  </div>`;
}

function pdfValido(bytes) {
  return bytes instanceof Uint8Array && bytes.length >= 10 && bytes.length <= 1024 * 1024
    && new TextDecoder().decode(bytes.subarray(0, 5)) === "%PDF-"
    && new TextDecoder().decode(bytes.subarray(-1024)).includes("%%EOF");
}
const REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;

function claveError(error) {
  if (["acceso_denegado", "denegado", "recurso_no_encontrado"].includes(error?.codigo)) return "circuito_firma_error_denegado";
  if (["conflicto", "paso_no_pendiente"].includes(error?.codigo)) return "circuito_firma_error_cambiado";
  if (error?.codigo === "cadena_rota") return "circuito_firma_error_cadena";
  if (error?.codigo === "firma_cancelada") return "circuito_firma_error_cancelada";
  if (error?.codigo === "autofirma_no_disponible") return "circuito_firma_error_autofirma";
  if (error?.codigo === "firma_fallida") return "circuito_firma_error_fallida";
  if (error?.codigo === "firma_no_verificada") return "circuito_firma_error_no_verificada";
  if (error?.codigo === "verificacion_no_disponible") return "circuito_firma_error_verificacion";
  return "circuito_firma_error_generico";
}

/** Puertos obligatorios de composición; ningún valor predeterminado acredita éxito. */
export function crearAccionesFirma({
  obtenerEstado, obtenerCircuito, t, clientePreflight, obtenerVinculoOriginal, obtenerOriginal,
  autofirma, registrarVec, registrarExterna, alCambiar, entornoDescarga = globalThis,
  crearClave = () => globalThis.crypto.randomUUID(),
} = {}) {
  let sesion = null;
  let retirada = false;
  let pendiente = false;
  let recuperacion = null;

  function contexto(bloque) {
    const estado = obtenerEstado?.(); const circuito = obtenerCircuito?.();
    const documento = bloque?.dataset?.ctFirmaDocumento;
    const pasoOrden = Number(bloque?.dataset?.ctFirmaPaso);
    const exp = estado?.expediente;
    const real = circuito?.documentos?.find((d) => d.documento === documento);
    if (retirada || estado?.vista !== "expediente" || estado?.carga !== "listo" || exp?.demostracion !== false
      || !REFERENCIA.test(exp?.expediente_ref ?? "") || !Number.isSafeInteger(exp.version) || exp.version < 1
      || !circuito?.registro || circuito.acciones === false || circuito.preflight_compuesto !== true
      || !real || real.paso_pendiente !== pasoOrden || pasoOrden < 1
      || !Object.hasOwn(PERFILES_BORRADOR_RRHH, documento)) return null;
    return { expedienteRef: exp.expediente_ref, version: exp.version, documento, pasoOrden,
      catalogoRef: circuito.catalogo_ref, catalogoHuella: circuito.huella_sha256 };
  }
  function igual(a, b) {
    return a && b && ["expedienteRef", "version", "documento", "pasoOrden", "catalogoRef", "catalogoHuella"].every((c) => a[c] === b[c]);
  }
  function vigente(s) { return sesion === s && !s.controlador.signal.aborted && igual(s.contexto, contexto(s.bloque)); }
  function decir(s, clave, variables = {}, foco = false) {
    if (!vigente(s)) return;
    const aviso = s.bloque.querySelector?.("[data-ct-firma-resultado]");
    if (aviso) { aviso.textContent = t(clave, variables); if (foco) aviso.focus?.({ preventScroll: true }); }
  }
  function ocupada(s, valor) {
    if (!vigente(s)) return;
    s.bloque.setAttribute?.("aria-busy", String(valor));
    for (const control of s.bloque.querySelectorAll?.("button,input") ?? []) {
      if (control.dataset?.ctFirmaAccion === "cancelar") {
        control.disabled = Boolean(s.recibo) || Boolean(valor && s.registroIntentado); continue;
      }
      control.disabled = valor || Boolean(s.recibo);
    }
  }
  function cancelar() {
    const anterior = sesion;
    if (anterior) {
      const incierta = anterior.registroIntentado && !anterior.recibo;
      if (incierta) recuperacion = { contexto: anterior.contexto, solicitud: anterior.solicitud, reintento: anterior.reintento };
      if (!anterior.recibo) decir(anterior, incierta ? "circuito_firma_recuperacion" : "circuito_firma_cancelada", {}, true);
      ocupada(anterior, false);
      const cancelacion = anterior.bloque.querySelector?.('[data-ct-firma-accion="cancelar"]');
      cancelacion?.remove?.();
      if (vigente(anterior) && !anterior.recibo) {
        anterior.bloque.innerHTML = `<button type="button" class="boton-primario" data-ct-firma-accion="comprobar">${escaparHTML(t("circuito_firma_comprobar"))}</button><p class="ct-circuito-resultado" role="status" aria-live="polite" tabindex="-1" data-ct-firma-resultado>${escaparHTML(t(incierta ? "circuito_firma_recuperacion" : "circuito_firma_cancelada"))}</p>`;
        anterior.bloque.querySelector?.('[data-ct-firma-accion="comprobar"]')?.focus?.({ preventScroll: true });
      }
      anterior.controlador.abort();
    }
    sesion = null; pendiente = false;
  }
  function dibujar(s) {
    const p = s.preflight; const id = `ct-firma-${s.contexto.documento}-${s.contexto.pasoOrden}`;
    const vec = p.vias_disponibles.includes("certificado_vec") && typeof autofirma?.firmarPDF === "function" && typeof registrarVec === "function";
    const externa = p.vias_disponibles.includes("portafirmas_registro_rrhh") && typeof registrarExterna === "function";
    s.bloque.innerHTML = `<p id="${id}-resultado" class="ct-circuito-resultado" role="status" aria-live="polite" tabindex="-1" data-ct-firma-resultado>${escaparHTML(t(vec || externa ? "circuito_firma_opciones" : "circuito_firma_sin_vias"))}</p>
      ${s.contexto.pasoOrden > 1 ? `<p>${escaparHTML(t("circuito_firma_siguiente_original"))}</p>` : ""}
      ${vec ? `<button type="button" class="boton-primario" data-ct-firma-accion="certificado_vec" aria-describedby="${id}-resultado">${escaparHTML(t("circuito_firma_autofirma_prueba"))}</button>` : ""}
      ${externa ? `<button type="button" class="boton-secundario" data-ct-firma-accion="descargar" aria-describedby="${id}-resultado">${escaparHTML(t("circuito_firma_descargar_externo"))}</button>
        <div class="ct-circuito-devolucion"><p>${escaparHTML(t("circuito_firma_datos_obligatorios"))}</p><div class="ct-exp-campo"><label for="${id}-pdf">${escaparHTML(t("circuito_firma_pdf_externo"))}</label><input type="file" id="${id}-pdf" accept="application/pdf,.pdf" data-ct-firma-pdf required></div>
        <div class="ct-exp-campo"><label for="${id}-ref">${escaparHTML(t("circuito_firma_referencia_externa"))}</label><input type="text" id="${id}-ref" maxlength="256" data-ct-firma-referencia required></div>
        <div class="ct-exp-campo"><label for="${id}-fecha">${escaparHTML(t("circuito_firma_fecha_externa"))}</label><input type="datetime-local" id="${id}-fecha" step="1" data-ct-firma-fecha required></div>
        <button type="button" class="boton-secundario" data-ct-firma-accion="portafirmas_registro_rrhh" aria-describedby="${id}-resultado">${escaparHTML(t("circuito_firma_enviar"))}</button></div>` : ""}
      <button type="button" class="boton-secundario" data-ct-firma-accion="cancelar">${escaparHTML(t("circuito_firma_cancelar"))}</button>`;
    s.bloque.querySelector?.("button")?.focus?.({ preventScroll: true });
  }
  async function comprobar(bloque, actual) {
    cancelar();
    if (typeof clientePreflight?.consultar !== "function" || typeof obtenerVinculoOriginal !== "function" || typeof obtenerOriginal !== "function") return;
    const s = { bloque, contexto: actual, controlador: new AbortController(), preflight: null };
    sesion = s; pendiente = true; ocupada(s, true); decir(s, "circuito_firma_comprobando");
    try {
      const vinculo = await obtenerVinculoOriginal(actual, { signal: s.controlador.signal });
      if (!vigente(s)) return;
      const solicitud = { ...actual, originalRef: vinculo?.originalRef, originalVersion: vinculo?.originalVersion,
        revisionEntradaRef: vinculo?.revisionEntradaRef, revisionEntradaVersion: vinculo?.revisionEntradaVersion,
        revisionEntradaHuella: vinculo?.revisionEntradaHuella };
      const respuesta = await clientePreflight.consultar(solicitud, { signal: s.controlador.signal });
      if (!vigente(s)) return;
      const p = validarPreflightFirma(respuesta, solicitud);
      if (!p || p.catalogo_ref !== actual.catalogoRef || p.catalogo_huella !== actual.catalogoHuella
        || (p.paso_pendiente !== 0 && p.paso_pendiente !== actual.pasoOrden)) throw { codigo: "conflicto" };
      s.solicitud = { ...solicitud, revisionEntradaRef: p.entrada_documento_ref,
        revisionEntradaVersion: p.entrada_documento_version, revisionEntradaHuella: p.entrada_documento_sha256 };
      s.vinculo = vinculo; s.preflight = p; dibujar(s);
      if (igual(recuperacion?.contexto, actual) && recuperacion.solicitud.originalRef === solicitud.originalRef
        && recuperacion.solicitud.originalVersion === solicitud.originalVersion) s.reintento = recuperacion.reintento;
    } catch (error) { decir(s, claveError(error), {}, true); }
    finally { if (sesion === s) { ocupada(s, false); pendiente = false; } }
  }
  async function bytesOriginal(s) {
    const original = await obtenerOriginal(s.solicitud, { signal: s.controlador.signal });
    if (!vigente(s)) return null;
    if (!pdfValido(original)) throw { codigo: "cadena_rota" };
    if (await huellaPDFFirmado(original) !== s.solicitud.revisionEntradaHuella) throw { codigo: "cadena_rota" };
    if (!vigente(s)) return null;
    return original;
  }
  async function descargar(s) {
    const bytes = await bytesOriginal(s);
    if (!bytes || !vigente(s)) return;
    const pagina = entornoDescarga.document;
    if (!pagina?.body || typeof entornoDescarga.URL?.createObjectURL !== "function" || typeof entornoDescarga.Blob !== "function") throw { codigo: "servicio_no_disponible" };
    const url = entornoDescarga.URL.createObjectURL(new entornoDescarga.Blob([bytes], { type: "application/pdf" }));
    const enlace = pagina.createElement("a");
    try { enlace.href = url; enlace.download = `${s.contexto.documento}.pdf`; enlace.hidden = true; pagina.body.append(enlace); enlace.click(); }
    finally { enlace.remove(); globalThis.setTimeout(() => entornoDescarga.URL.revokeObjectURL(url), 0); }
    decir(s, "circuito_firma_original_descargado", {}, true);
  }
  function validarRecibo(recibo, s) {
    return recibo?.firma_eficaz === false && REFERENCIA.test(recibo.recibo_ref ?? "")
      && recibo.expediente_ref === s.contexto.expedienteRef && recibo.documento === s.contexto.documento
      && recibo.paso_orden === s.contexto.pasoOrden
      && (recibo.version_expediente ?? recibo.expediente_version) === s.contexto.version
      && (recibo.firma_verificada === true || recibo.verificacion_tecnica?.estado === "valida");
  }
  async function registrar(s, via) {
    const registro = via === "certificado_vec" ? registrarVec : registrarExterna;
    if (typeof registro !== "function") return;
    if (via === "certificado_vec" && !s.reintento) {
      const original = await bytesOriginal(s); if (!original) return;
      decir(s, "circuito_firma_abriendo_autofirma");
      const firmado = await autofirma.firmarPDF(original, { signal: s.controlador.signal });
      if (!vigente(s)) return;
      if (!pdfValido(firmado)) throw { codigo: "firma_no_verificada" };
      s.reintento = { via, datos: { ...s.solicitud, original, firmado, clave: crearClave() } };
    }
    if (via === "portafirmas_registro_rrhh") {
      const fichero = s.bloque.querySelector?.("[data-ct-firma-pdf]")?.files?.[0];
      const referenciaPortafirmas = s.bloque.querySelector?.("[data-ct-firma-referencia]")?.value?.trim();
      const fecha = s.bloque.querySelector?.("[data-ct-firma-fecha]")?.value;
      if (!fichero || fichero.size > 1024 * 1024 || fichero.size < 10 || !referenciaPortafirmas || referenciaPortafirmas.length > 256
        || !fecha || Number.isNaN(new Date(fecha).getTime())) { decir(s, "circuito_firma_campos_externos", {}, true); return; }
      if (!s.reintento || s.reintento.fichero !== fichero || s.reintento.referencia !== referenciaPortafirmas || s.reintento.fecha !== fecha) {
        const firmado = new Uint8Array(await fichero.arrayBuffer());
        if (!vigente(s)) return;
        if (!pdfValido(firmado)) { decir(s, "circuito_firma_campos_externos", {}, true); return; }
        const anterior = s.reintento;
        const mismo = anterior?.via === via && anterior.referencia === referenciaPortafirmas && anterior.fecha === fecha
          && anterior.datos.firmado.length === firmado.length && firmado.every((b, i) => b === anterior.datos.firmado[i]);
        s.reintento = { via, fichero, referencia: referenciaPortafirmas, fecha,
          datos: mismo ? anterior.datos : { ...s.solicitud, firmado, referenciaPortafirmas, fechaPortafirmas: new Date(fecha).toISOString().replace(/\.000Z$/u, "Z"), clave: crearClave() } };
      }
    }
    if (!vigente(s) || s.reintento?.via !== via) return;
    decir(s, "circuito_firma_registrando"); s.registroIntentado = true;
    const cancelacion = s.bloque.querySelector?.('[data-ct-firma-accion="cancelar"]');
    if (cancelacion) cancelacion.disabled = true;
    const recibo = await registro(s.reintento.datos, { signal: s.controlador.signal });
    if (!vigente(s)) return;
    if (!validarRecibo(recibo, s)) throw { codigo: "resultado_no_confiable" };
    s.recibo = recibo;
    recuperacion = null;
    const aviso = t("circuito_firma_registro_confirmado", { recibo: recibo.recibo_ref });
    decir(s, "circuito_firma_registro_confirmado", { recibo: recibo.recibo_ref }, true);
    s.preflight = null; s.reintento = null;
    // El recibo ya está confirmado. Un fallo al refrescar no lo convierte en fallo de registro.
    try { await alCambiar?.(aviso, recibo); } catch { /* se conserva la confirmación y se cierran acciones */ }
  }
  async function manejarClic(evento) {
    const boton = evento?.target?.closest?.("[data-ct-firma-accion]");
    if (!boton || boton.disabled || boton.getAttribute?.("aria-disabled") === "true") return;
    const accion = boton.dataset?.ctFirmaAccion;
    if (accion === "cancelar") { if (!sesion?.registroIntentado || !pendiente) cancelar(); return; }
    const bloque = boton.closest?.("[data-ct-firma-controles]");
    const actual = contexto(bloque);
    if (!actual) return;
    if (accion === "comprobar") {
      if (pendiente && sesion?.bloque === bloque) return;
      await comprobar(bloque, actual); return;
    }
    if (pendiente) return;
    const s = sesion;
    if (!s || !vigente(s) || !s.preflight || s.bloque !== bloque) return;
    const via = accion === "descargar" ? "portafirmas_registro_rrhh" : accion;
    if (!s.preflight.vias_disponibles.includes(via)) return;
    pendiente = true; ocupada(s, true);
    try { if (accion === "descargar") await descargar(s); else await registrar(s, via); }
    catch (error) { decir(s, s.registroIntentado && ["servicio_no_disponible", "resultado_no_confiable"].includes(error?.codigo)
      ? "circuito_firma_recuperacion" : claveError(error), {}, true); }
    finally { if (sesion === s) { ocupada(s, false); pendiente = false; } }
  }
  return Object.freeze({ manejarClic, cancelar, retirar() { cancelar(); retirada = true; },
    actualizar() { if (sesion && !igual(sesion.contexto, contexto(sesion.bloque))) cancelar(); } });
}
