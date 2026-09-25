/** Gestor de descarga y estado de borradores e informes definitivos de RRHH. */

import {
  crearClienteHTTPBorradorRRHH, PERFILES_BORRADOR_RRHH, tipoBorradorDeAccion,
} from "./cliente-http-informe-definitivo.js";
import { solicitudInformeDefinitivoDesdeEstado } from "./componentes-expedientes.js";
import { crearTraductorExpedientesContratacion } from "./i18n-expedientes.js";

export function crearGestorDescargaBorradorRRHH({
  raiz,
  presentador,
  clienteBorradorRRHH = crearClienteHTTPBorradorRRHH(),
  entornoDescarga = globalThis,
  mensajes = {},
  anunciar = () => {},
  esMontada = () => true,
} = {}) {
  let descargaInforme = null;
  let controlesDescargaInforme = null;
  let accionDescargaInforme = null;
  let formatoDescargaInforme = null;
  let urlInforme = null;
  let revocacionInforme = null;

  function liberarURLInforme() {
    clearTimeout(revocacionInforme);
    revocacionInforme = null;
    if (urlInforme !== null) entornoDescarga.URL?.revokeObjectURL?.(urlInforme);
    urlInforme = null;
  }

  function restaurarControlesDescarga({ botones, cancelaciones, reintentos }) {
    botones.forEach((control) => { control.disabled = false; });
    cancelaciones.forEach((control) => { control.disabled = true; });
    reintentos.forEach((control) => { control.disabled = control.hidden !== false; });
  }

  function cancelarDescargaInforme() {
    const activa = descargaInforme !== null;
    const controles = controlesDescargaInforme;
    const accion = accionDescargaInforme;
    const formato = formatoDescargaInforme;
    descargaInforme?.abort();
    descargaInforme = null;
    controlesDescargaInforme = null;
    accionDescargaInforme = null;
    formatoDescargaInforme = null;
    liberarURLInforme();
    if (activa && accion) actualizarResultadoDescarga(accion.replace("descargar-", ""), "descarga_cancelada", true, formato);
    if (activa && controles && esMontada()) restaurarControlesDescarga(controles);
    return activa;
  }

  function informarDescarga(clave, tipo) {
    const texto = crearTraductorExpedientesContratacion(mensajes)(clave);
    const mensaje = raiz.querySelector("[data-ct-exp-mensaje]");
    if (mensaje) {
      mensaje.textContent = texto;
      mensaje.setAttribute("role", tipo === "error" ? "alert" : "status");
    }
    anunciar(texto, tipo);
  }

  function actualizarResultadoDescarga(accion, clave, reintentar = false, formato = null) {
    const t = crearTraductorExpedientesContratacion(mensajes);
    const resultado = raiz.querySelector?.(`[data-ct-exp-resultado-descarga="${accion}"]`);
    if (resultado) resultado.textContent = t(clave);
    const boton = raiz.querySelector?.(`[data-ct-exp-accion="reintentar-descarga-${accion}"]`);
    if (boton) {
      boton.disabled = !reintentar;
      boton.hidden = !reintentar;
      if (reintentar && ["pdf", "docx"].includes(formato)) boton.dataset.ctExpFormato = formato;
      else delete boton.dataset.ctExpFormato;
    }
  }

  async function descargarBorrador(boton) {
    const solicitud = solicitudInformeDefinitivoDesdeEstado(presentador.obtenerEstado());
    if (!esMontada() || descargaInforme || !solicitud) return;
    const esReintento = boton.dataset.ctExpAccion.startsWith("reintentar-descarga-");
    const accionSolicitada = boton.dataset.ctExpAccion.replace("reintentar-descarga-", "descargar-");
    const accionDocumento = accionSolicitada.replace("descargar-docx-", "descargar-");
    const formato = esReintento && ["pdf", "docx"].includes(boton.dataset.ctExpFormato)
      ? boton.dataset.ctExpFormato : accionSolicitada.startsWith("descargar-docx-") ? "docx" : "pdf";
    const tipo = tipoBorradorDeAccion(accionDocumento) ?? "informe_definitivo";
    const botones = typeof raiz.querySelectorAll === "function" ? [...raiz.querySelectorAll(
      Object.keys(PERFILES_BORRADOR_RRHH).map((clave) => clave.replaceAll("_", "-"))
        .flatMap((accion) => [`[data-ct-exp-accion="descargar-${accion}"]`, `[data-ct-exp-accion="descargar-docx-${accion}"]`]).join(", "),
    )] : [boton];
    const cancelaciones = typeof raiz.querySelectorAll === "function" ? [...raiz.querySelectorAll(
      '[data-ct-exp-accion="cancelar-descarga"]',
    )].filter((control) => control.dataset?.ctExpAccion === "cancelar-descarga") : [];
    const reintentos = typeof raiz.querySelectorAll === "function" ? [...raiz.querySelectorAll(
      '[data-ct-exp-accion^="reintentar-descarga-"]',
    )] : [];
    const controlador = new AbortController();
    descargaInforme = controlador;
    controlesDescargaInforme = { botones, cancelaciones, reintentos };
    accionDescargaInforme = accionDocumento;
    formatoDescargaInforme = formato;
    botones.forEach((control) => { control.disabled = true; });
    cancelaciones.forEach((control) => { control.disabled = false; });
    reintentos.forEach((control) => { control.disabled = true; });
    actualizarResultadoDescarga(accionDocumento.replace("descargar-", ""), "informe_definitivo_descargando");
    informarDescarga("informe_definitivo_descargando", "informacion");
    try {
      const { document: documento, URL: urls } = entornoDescarga;
      if (!documento?.body || typeof urls?.createObjectURL !== "function"
        || typeof urls?.revokeObjectURL !== "function") throw new TypeError();
      const blob = await clienteBorradorRRHH.descargarBorrador(solicitud, {
        tipo, formato, signal: controlador.signal,
      });
      if (!esMontada() || descargaInforme !== controlador || controlador.signal.aborted
        || JSON.stringify(solicitudInformeDefinitivoDesdeEstado(presentador.obtenerEstado()))
          !== JSON.stringify(solicitud)) return;
      liberarURLInforme();
      urlInforme = urls.createObjectURL(blob);
      const enlace = documento.createElement("a");
      try {
        enlace.href = urlInforme;
        enlace.download = formato === "pdf" ? PERFILES_BORRADOR_RRHH[tipo].nombre
          : PERFILES_BORRADOR_RRHH[tipo].nombre.replace(/\.pdf$/u, ".docx");
        enlace.hidden = true;
        documento.body.append(enlace);
        enlace.click();
      } finally {
        enlace.remove();
        revocacionInforme = setTimeout(liberarURLInforme, 0);
      }
      informarDescarga("informe_definitivo_listo", "informacion");
      actualizarResultadoDescarga(accionDocumento.replace("descargar-", ""), "informe_definitivo_listo");
    } catch (error) {
      if (!esMontada() || descargaInforme !== controlador || controlador.signal.aborted) return;
      const clave = error?.envelopeValido === true && error.codigo === "documento_no_disponible"
        ? "informe_definitivo_no_disponible"
        : error?.envelopeValido === true && ["acceso_denegado", "autenticacion_requerida"].includes(error.codigo)
          ? "informe_definitivo_denegado" : "informe_definitivo_error";
      informarDescarga(clave, "error");
      actualizarResultadoDescarga(accionDocumento.replace("descargar-", ""), clave, true, formato);
    } finally {
      if (descargaInforme === controlador) {
        descargaInforme = null;
        controlesDescargaInforme = null;
        accionDescargaInforme = null;
        formatoDescargaInforme = null;
        restaurarControlesDescarga({ botones, cancelaciones, reintentos });
      }
    }
  }

  return Object.freeze({
    esAccionDescarga: (accion) => tipoBorradorDeAccion(accion) !== null,
    descargarBorrador,
    cancelarDescargaInforme,
    liberarURLInforme,
    informarDescarga,
  });
}
