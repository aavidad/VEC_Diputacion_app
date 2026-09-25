/**
 * Acciones del paso pendiente del circuito de firma: «Firmar» con AutoFirma
 * y «Devolver» con motivo. La firma se hace en el equipo de la persona; el
 * servidor la verifica y la registra. Una firma registrada es de prueba y no
 * tiene eficacia administrativa hasta el portafirmas corporativo.
 */

import { escaparHTML, solicitudInformeDefinitivoDesdeEstado } from "./componentes-expedientes.js";
import { crearClienteHTTPBorradorRRHH, PERFILES_BORRADOR_RRHH } from "./cliente-http-informe-definitivo.js";
import { crearClienteAutoFirma } from "./firma-autofirma.js";
import { crearClienteFirmaDocumento } from "./firma-documento-cliente.js";

const CLAVE_ERROR = Object.freeze({
  verificacion_no_disponible: "circuito_firma_error_verificacion",
  firma_no_verificada: "circuito_firma_error_no_verificada",
  autofirma_no_disponible: "circuito_firma_error_autofirma",
  plazo_agotado: "circuito_firma_error_autofirma",
  firma_cancelada: "circuito_firma_error_cancelada",
  firma_fallida: "circuito_firma_error_fallida",
  paso_no_pendiente: "circuito_firma_error_cambiado",
  conflicto: "circuito_firma_error_cambiado",
  cadena_rota: "circuito_firma_error_cadena",
  acceso_denegado: "circuito_firma_error_denegado",
});

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
      }))),
    }));
  }
  return Object.freeze({ ...circuito, documentos: Object.freeze(documentos), registro: Object.freeze({ verificacion: estado.verificacion_disponible }) });
}

/** Controles del paso pendiente; solo si el registro está compuesto. */
export function renderizarAccionesPaso(circuito, documento, paso, t) {
  if (!circuito.registro || documento.paso_pendiente !== paso.orden || !Object.hasOwn(PERFILES_BORRADOR_RRHH, documento.documento)) return "";
  const id = `ct-firma-motivo-${documento.documento}-${paso.orden}`;
  const datos = `data-ct-firma-documento="${escaparHTML(documento.documento)}" data-ct-firma-orden="${paso.orden}"`;
  return `<div class="ct-circuito-acciones" ${datos}>
    <button type="button" class="boton-primario" data-ct-firma-accion="firmar" ${datos}>${escaparHTML(t("circuito_firma_firmar"))}</button>
    <button type="button" class="boton-secundario" data-ct-firma-accion="devolver" ${datos} aria-expanded="false" aria-controls="${id}">${escaparHTML(t("circuito_firma_devolver"))}</button>
    <div class="ct-circuito-devolucion" id="${id}" hidden>
      <label for="${id}-texto">${escaparHTML(t("circuito_firma_motivo_etiqueta"))}</label>
      <textarea id="${id}-texto" rows="2" maxlength="500" data-ct-firma-motivo></textarea>
      <div class="ct-circuito-devolucion-botones">
        <button type="button" class="boton-secundario" data-ct-firma-accion="confirmar-devolucion" ${datos}>${escaparHTML(t("circuito_firma_confirmar_devolucion"))}</button>
        <button type="button" class="boton-terciario" data-ct-firma-accion="cancelar-devolucion" ${datos}>${escaparHTML(t("circuito_firma_cancelar"))}</button>
      </div>
    </div>
    <p class="ct-circuito-resultado" role="status" aria-live="polite" data-ct-firma-resultado></p>
  </div>`;
}

function claveIdempotencia(aleatorio) {
  return `firma-${[...aleatorio(16)].map((b) => b.toString(16).padStart(2, "0")).join("")}`;
}

/**
 * Coordina las acciones. `alCambiar` vuelve a consultar el estado y repinta
 * el bloque tras registrar. Las dependencias se pueden sustituir en pruebas.
 */
export function crearAccionesFirma({
  obtenerEstado, t, alCambiar,
  clienteFirma = crearClienteFirmaDocumento(),
  clienteBorrador = crearClienteHTTPBorradorRRHH(),
  autofirma = crearClienteAutoFirma(),
  aleatorio = (n) => globalThis.crypto.getRandomValues(new Uint8Array(n)),
} = {}) {
  let ocupado = false;

  function mostrar(contenedor, texto) {
    const salida = contenedor?.querySelector?.("[data-ct-firma-resultado]");
    if (salida) salida.textContent = texto;
  }

  function textoError(error) {
    const clave = CLAVE_ERROR[error?.codigo] ?? "circuito_firma_error_generico";
    return t(clave, { motivo: error?.motivo || error?.codigo || "" });
  }

  async function ejecutar(contenedor, documento, orden, accion) {
    const solicitud = solicitudInformeDefinitivoDesdeEstado(obtenerEstado());
    if (!solicitud) { mostrar(contenedor, t("circuito_firma_error_cambiado")); return; }
    const base = { expedienteRef: solicitud.expediente_ref, version: solicitud.version_observada, documento, pasoOrden: orden, clave: claveIdempotencia(aleatorio) };
    let recibo;
    if (accion === "devolver") {
      const motivo = contenedor.querySelector?.("[data-ct-firma-motivo]")?.value?.trim() ?? "";
      if (motivo.length < 3 || motivo.length > 500) { mostrar(contenedor, t("circuito_firma_motivo_invalido")); return; }
      mostrar(contenedor, t("circuito_firma_registrando"));
      recibo = await clienteFirma.registrar({ ...base, resultado: "devuelto", motivoDevolucion: motivo });
      mostrar(contenedor, t("circuito_firma_devuelta", { recibo: recibo.recibo_ref }));
    } else {
      mostrar(contenedor, t("circuito_firma_preparando"));
      const blob = await clienteBorrador.descargarBorrador(solicitud, { tipo: documento, formato: "pdf" });
      const original = new Uint8Array(await blob.arrayBuffer());
      mostrar(contenedor, t("circuito_firma_abriendo_autofirma"));
      const firmado = await autofirma.firmarPDF(original);
      mostrar(contenedor, t("circuito_firma_registrando"));
      recibo = await clienteFirma.registrar({ ...base, resultado: "firmado", original, firmado });
      mostrar(contenedor, t("circuito_firma_registrada", { recibo: recibo.recibo_ref }));
    }
    await alCambiar?.(t(accion === "devolver" ? "circuito_firma_devuelta" : "circuito_firma_registrada", { recibo: recibo.recibo_ref }));
  }

  /** Atiende un clic delegado en el bloque; ignora lo que no es suyo. */
  async function manejarClic(evento) {
    const boton = evento?.target?.closest?.("[data-ct-firma-accion]");
    if (!boton) return;
    const contenedor = boton.closest("[data-ct-firma-documento].ct-circuito-acciones");
    const documento = boton.dataset.ctFirmaDocumento;
    const orden = Number(boton.dataset.ctFirmaOrden);
    const accion = boton.dataset.ctFirmaAccion;
    const panel = contenedor?.querySelector?.(".ct-circuito-devolucion");
    const conmutador = contenedor?.querySelector?.('[data-ct-firma-accion="devolver"]');
    if (accion === "devolver" || accion === "cancelar-devolucion") {
      const abrir = accion === "devolver";
      if (panel) panel.hidden = !abrir;
      conmutador?.setAttribute("aria-expanded", String(abrir));
      if (abrir) panel?.querySelector?.("textarea")?.focus?.();
      else conmutador?.focus?.();
      return;
    }
    if (ocupado || !contenedor || !Number.isSafeInteger(orden)) return;
    ocupado = true;
    const botones = [...(contenedor.querySelectorAll?.("button") ?? [])];
    botones.forEach((b) => { b.disabled = true; });
    try {
      await ejecutar(contenedor, documento, orden, accion === "confirmar-devolucion" ? "devolver" : "firmar");
    } catch (error) {
      mostrar(contenedor, textoError(error));
    } finally {
      botones.forEach((b) => { b.disabled = false; });
      ocupado = false;
    }
  }

  return Object.freeze({ manejarClic });
}
