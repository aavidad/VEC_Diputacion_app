/**
 * Acciones del paso pendiente del circuito de firma: «Firmar» con AutoFirma
 * y «Devolver» con motivo. La firma se hace en el equipo de la persona; el
 * servidor la verifica y la registra. Una firma registrada es de prueba y no
 * tiene eficacia administrativa hasta el portafirmas corporativo.
 */

import { escaparHTML, solicitudInformeDefinitivoDesdeEstado } from "./componentes-expedientes.js?v=20261001-f-reconciliacion-325-v1";
import { crearClienteHTTPBorradorRRHH, PERFILES_BORRADOR_RRHH } from "./cliente-http-informe-definitivo.js";
import { crearClienteAutoFirma } from "./firma-autofirma.js";
import { crearClienteFirmaDocumento } from "./firma-documento-cliente.js?v=20260930-custodia-506-e3-v3";
import { crearClienteFirmaExterna } from "./firma-externa-cliente.js";
import { localizacionDe } from "../../../comun/idioma.js";

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
  autenticacion_requerida: "circuito_firma_error_denegado",
  operacion_abortada: "circuito_firma_error_cambiado",
});

const REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
function referenciaValida(valor) { return typeof valor === "string" && REFERENCIA.test(valor); }

// Este contrato solo se puede completar con una respuesta autorizada del
// servidor. La consulta actual no lo entrega y, por tanto, falla cerrado.
export function contextoFirmaExternaValido(contexto, expedienteRef, version, documento, orden) {
  return contexto?.permitido === true && referenciaValida(contexto.expediente_ref)
    && Number.isSafeInteger(contexto.version_expediente) && contexto.version_expediente > 0
    && contexto.expediente_ref === expedienteRef
    && contexto.version_expediente === version && contexto.documento === documento
    && contexto.paso_orden === orden && referenciaValida(contexto.original_ref)
    && Number.isSafeInteger(contexto.original_version) && contexto.original_version > 0;
}

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
    registro: Object.freeze({ verificacion: estado.verificacion_disponible, externo: null }) });
}

/**
 * Controles del paso pendiente; solo si el registro está compuesto y los
 * borradores del expediente se pueden descargar (`acciones` no es false).
 */
export function renderizarAccionesPaso(circuito, documento, paso, t) {
  if (!circuito.registro || circuito.acciones === false || documento.paso_pendiente !== paso.orden || !Object.hasOwn(PERFILES_BORRADOR_RRHH, documento.documento)) return "";
  const id = `ct-firma-motivo-${documento.documento}-${paso.orden}`;
  const avisoId = `ct-firma-resultado-${documento.documento}-${paso.orden}`;
  const externoId = `ct-firma-externa-${documento.documento}-${paso.orden}`;
  const verificacionApagada = circuito.registro.verificacion === false;
  const externo = contextoFirmaExternaValido(circuito.registro.externo,
    circuito.registro.externo?.expediente_ref, circuito.registro.externo?.version_expediente,
    documento.documento, paso.orden);
  const datos = `data-ct-firma-documento="${escaparHTML(documento.documento)}" data-ct-firma-orden="${paso.orden}"`;
  return `<div class="ct-circuito-acciones" ${datos}>
    <button type="button" class="boton-primario" data-ct-firma-accion="firmar" ${datos}${verificacionApagada ? ` aria-disabled="true" aria-describedby="${avisoId}"` : ""}>${escaparHTML(t("circuito_firma_firmar"))}</button>
    <button type="button" class="boton-secundario" data-ct-firma-accion="abrir-externo" ${datos}${externo ? ` aria-expanded="false" aria-controls="${externoId}"` : ` disabled aria-describedby="${externoId}-motivo"`}>${escaparHTML(t("circuito_firma_externa_abrir"))}</button>
    ${externo ? `<div class="ct-circuito-devolucion ct-circuito-externo" id="${externoId}" hidden>
      <label for="${externoId}-pdf">${escaparHTML(t("circuito_firma_externa_pdf"))}</label>
      <input id="${externoId}-pdf" type="file" accept="application/pdf,.pdf" data-ct-firma-externa-pdf required>
      <label for="${externoId}-ref">${escaparHTML(t("circuito_firma_externa_referencia"))}</label>
      <input id="${externoId}-ref" type="text" maxlength="256" data-ct-firma-externa-ref required>
      <label for="${externoId}-fecha">${escaparHTML(t("circuito_firma_externa_fecha"))}</label>
      <input id="${externoId}-fecha" type="datetime-local" data-ct-firma-externa-fecha required>
      <p>${escaparHTML(t("circuito_firma_externa_advertencia"))}</p>
      <button type="button" class="boton-secundario" data-ct-firma-accion="revisar-externo" ${datos}>${escaparHTML(t("circuito_firma_externa_revisar"))}</button>
      <div data-ct-firma-externa-confirmacion hidden>
        <p data-ct-firma-externa-resumen></p>
        <button type="button" class="boton-primario" data-ct-firma-accion="confirmar-externo" ${datos}>${escaparHTML(t("circuito_firma_externa_confirmar"))}</button>
      </div>
      <button type="button" class="boton-terciario" data-ct-firma-accion="cancelar-externo" ${datos}>${escaparHTML(t("circuito_firma_cancelar"))}</button>
    </div>` : `<p id="${externoId}-motivo" class="ct-circuito-resultado">${escaparHTML(t("circuito_firma_externa_bloqueada"))}</p>`}
    <button type="button" class="boton-secundario" data-ct-firma-accion="devolver" ${datos} aria-expanded="false" aria-controls="${id}">${escaparHTML(t("circuito_firma_devolver"))}</button>
    <div class="ct-circuito-devolucion" id="${id}" hidden>
      <label for="${id}-texto">${escaparHTML(t("circuito_firma_motivo_etiqueta"))}</label>
      <textarea id="${id}-texto" rows="2" maxlength="500" data-ct-firma-motivo></textarea>
      <div class="ct-circuito-devolucion-botones">
        <button type="button" class="boton-secundario" data-ct-firma-accion="confirmar-devolucion" ${datos}>${escaparHTML(t("circuito_firma_confirmar_devolucion"))}</button>
        <button type="button" class="boton-terciario" data-ct-firma-accion="cancelar-devolucion" ${datos}>${escaparHTML(t("circuito_firma_cancelar"))}</button>
      </div>
    </div>
    <p id="${avisoId}" class="ct-circuito-resultado" role="status" aria-live="polite" data-ct-firma-resultado>${verificacionApagada ? escaparHTML(t("circuito_firma_error_verificacion")) : ""}</p>
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
  clienteExterno = crearClienteFirmaExterna(),
  obtenerContextoExterno = () => null,
  clienteBorrador = crearClienteHTTPBorradorRRHH(),
  autofirma = crearClienteAutoFirma(),
  aleatorio = (n) => globalThis.crypto.getRandomValues(new Uint8Array(n)),
} = {}) {
  let ocupado = false;
  let preparada = null;

  function mostrar(contenedor, texto) {
    const salida = contenedor?.querySelector?.("[data-ct-firma-resultado]");
    if (salida) salida.textContent = texto;
  }

  function textoError(error) {
    if (error?.codigo === "servicio_no_disponible" &&
      ["validador_no_disponible", "credencial_rechazada"].includes(error?.motivo)) {
      return t("circuito_firma_error_validador_no_disponible");
    }
    const clave = CLAVE_ERROR[error?.codigo] ?? "circuito_firma_error_generico";
    return t(clave);
  }

  function textoErrorExterna(error) {
    if (error?.codigo === "acceso_denegado" || error?.codigo === "autenticacion_requerida") {
      return t("circuito_firma_externa_error_denegado");
    }
    if (error?.codigo === "contenido_no_valido" || error?.codigo === "peticion_no_valida") {
      return t("circuito_firma_externa_error_datos");
    }
    return textoError(error);
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

  function solicitudExterna(documento, orden) {
    const solicitud = solicitudInformeDefinitivoDesdeEstado(obtenerEstado());
    if (!solicitud) return null;
    const contexto = obtenerContextoExterno(documento, orden);
    if (!contextoFirmaExternaValido(contexto, solicitud.expediente_ref, solicitud.version_observada, documento, orden)) return null;
    return { expedienteRef: solicitud.expediente_ref, version: solicitud.version_observada, documento, pasoOrden: orden,
      originalRef: contexto.original_ref, originalVersion: contexto.original_version };
  }

  async function revisarExterna(contenedor, documento, orden) {
    const base = solicitudExterna(documento, orden);
    if (!base) { mostrar(contenedor, t("circuito_firma_externa_bloqueada")); return; }
    const panel = contenedor.querySelector?.("[data-ct-firma-externa-confirmacion]");
    if (panel) panel.hidden = true;
    const archivo = contenedor.querySelector?.("[data-ct-firma-externa-pdf]")?.files?.[0];
    const referencia = contenedor.querySelector?.("[data-ct-firma-externa-ref]")?.value?.trim() ?? "";
    const fechaLocal = contenedor.querySelector?.("[data-ct-firma-externa-fecha]")?.value ?? "";
    const fecha = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/u.test(fechaLocal) ? new Date(fechaLocal) : null;
    if (!archivo || archivo.size < 10 || archivo.size > (1 << 20) || archivo.type !== "application/pdf"
      || referencia.length < 1 || referencia.length > 256 || /[\u0000-\u001f\u007f]/u.test(referencia)
      || !fecha || Number.isNaN(fecha.getTime())) {
      preparada = null;
      mostrar(contenedor, t("circuito_firma_externa_datos_invalidos")); return;
    }
    const claveAnterior = preparada?.archivo === archivo && preparada.referenciaPortafirmas === referencia
      && preparada.fechaLocal === fechaLocal && Object.keys(base).every((clave) => preparada[clave] === base[clave])
      ? preparada.clave : null;
    preparada = { ...base, firmado: new Uint8Array(await archivo.arrayBuffer()), referenciaPortafirmas: referencia,
      fechaPortafirmas: fecha.toISOString().replace(/\.000Z$/u, "Z"), clave: claveAnterior ?? claveIdempotencia(aleatorio), archivo, fechaLocal };
    const resumen = contenedor.querySelector?.("[data-ct-firma-externa-resumen]");
    const idioma = localizacionDe(globalThis.document?.documentElement?.lang);
    const fechaVista = new Date(`${fechaLocal}:00Z`);
    if (resumen) resumen.textContent = t("circuito_firma_externa_resumen", { archivo: archivo.name, referencia,
      fecha: new Intl.DateTimeFormat(idioma, { dateStyle: "short", timeStyle: "short", timeZone: "UTC" }).format(fechaVista) });
    if (panel) panel.hidden = false;
    mostrar(contenedor, t("circuito_firma_externa_revisada"));
    return panel?.querySelector?.("button");
  }

  async function registrarExterna(contenedor, documento, orden) {
    const base = solicitudExterna(documento, orden);
    const archivoActual = contenedor.querySelector?.("[data-ct-firma-externa-pdf]")?.files?.[0];
    const referenciaActual = contenedor.querySelector?.("[data-ct-firma-externa-ref]")?.value?.trim() ?? "";
    const fechaActual = contenedor.querySelector?.("[data-ct-firma-externa-fecha]")?.value ?? "";
    if (!base || !preparada || Object.keys(base).some((clave) => preparada[clave] !== base[clave])) {
      mostrar(contenedor, t("circuito_firma_error_cambiado")); return;
    }
    if (preparada.archivo !== archivoActual || preparada.referenciaPortafirmas !== referenciaActual
      || preparada.fechaLocal !== fechaActual) {
      mostrar(contenedor, t("circuito_firma_externa_datos_modificados")); return;
    }
    mostrar(contenedor, t("circuito_firma_registrando"));
    const { archivo: _archivo, fechaLocal: _fechaLocal, ...solicitud } = preparada;
    const recibo = await clienteExterno.registrar(solicitud);
    preparada = null;
    const aviso = t(recibo.ya_registrada ? "circuito_firma_externa_repetida" : "circuito_firma_externa_registrada", { recibo: recibo.recibo_ref });
    mostrar(contenedor, aviso);
    await alCambiar?.(aviso);
  }

  /** Atiende un clic delegado en el bloque; ignora lo que no es suyo. */
  async function manejarClic(evento) {
    const boton = evento?.target?.closest?.("[data-ct-firma-accion]");
    if (!boton) return;
    const contenedor = boton.closest("[data-ct-firma-documento].ct-circuito-acciones");
    const documento = boton.dataset.ctFirmaDocumento;
    const orden = Number(boton.dataset.ctFirmaOrden);
    const accion = boton.dataset.ctFirmaAccion;
    const panel = contenedor?.querySelector?.("[id^='ct-firma-motivo-']");
    const conmutador = contenedor?.querySelector?.('[data-ct-firma-accion="devolver"]');
    if (accion === "devolver" || accion === "cancelar-devolucion") {
      const abrir = accion === "devolver";
      if (panel) panel.hidden = !abrir;
      conmutador?.setAttribute("aria-expanded", String(abrir));
      if (abrir) panel?.querySelector?.("textarea")?.focus?.();
      else conmutador?.focus?.();
      return;
    }
    if (accion === "abrir-externo") {
      if (!solicitudExterna(documento, orden) || boton.getAttribute?.("aria-disabled") === "true") {
        mostrar(contenedor, t("circuito_firma_externa_bloqueada")); return;
      }
      const externo = contenedor.querySelector?.("[id^='ct-firma-externa-']");
      if (externo) externo.hidden = false;
      boton.setAttribute?.("aria-expanded", "true");
      externo?.querySelector?.("input")?.focus?.();
      return;
    }
    if (accion === "cancelar-externo") {
      preparada = null;
      const externo = contenedor?.querySelector?.("[id^='ct-firma-externa-']");
      if (externo) externo.hidden = true;
      const confirmacion = contenedor?.querySelector?.("[data-ct-firma-externa-confirmacion]");
      if (confirmacion) confirmacion.hidden = true;
      const abrir = contenedor?.querySelector?.('[data-ct-firma-accion="abrir-externo"]');
      abrir?.setAttribute?.("aria-expanded", "false");
      abrir?.focus?.();
      return;
    }
    if (accion === "firmar" && boton.getAttribute?.("aria-disabled") === "true") {
      mostrar(contenedor, t("circuito_firma_error_verificacion"));
      return;
    }
    if (!["firmar", "confirmar-devolucion", "revisar-externo", "confirmar-externo"].includes(accion)) return;
    if (ocupado || !contenedor || !Number.isSafeInteger(orden)) return;
    ocupado = true;
    const botones = [...(contenedor.querySelectorAll?.("button") ?? [])];
    const deshabilitados = botones.map((b) => b.disabled);
    botones.forEach((b) => { b.disabled = true; });
    let destinoFoco;
    try {
      if (accion === "revisar-externo") destinoFoco = await revisarExterna(contenedor, documento, orden);
      else if (accion === "confirmar-externo") await registrarExterna(contenedor, documento, orden);
      else await ejecutar(contenedor, documento, orden, accion === "confirmar-devolucion" ? "devolver" : "firmar");
    } catch (error) {
      mostrar(contenedor, accion === "revisar-externo" || accion === "confirmar-externo" ? textoErrorExterna(error) : textoError(error));
    } finally {
      botones.forEach((b, i) => { b.disabled = deshabilitados[i]; });
      destinoFoco?.focus?.();
      ocupado = false;
    }
  }

  return Object.freeze({ manejarClic });
}
