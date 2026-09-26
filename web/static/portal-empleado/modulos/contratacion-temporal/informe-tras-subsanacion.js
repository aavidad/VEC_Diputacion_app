/** Informe jurídico nuevo tras subsanar un reparo (duda 5 de RRHH): botón en la
 * fase de subsanación, emisión con la misma API del informe y refresco del
 * expediente para que aparezca la nueva fiscalización. Que el informe nuevo
 * proceda lo decide el catálogo en el servidor; aquí solo se ofrece cuando la
 * subsanación está registrada y el informe nuevo todavía no. */

import { validarSolicitudInformeJuridico } from "./contrato-informe-juridico.js";
import { escaparHTML } from "./componentes-expedientes.js";
import { crearTraductorContratacionTemporal } from "./i18n.js";
import { MENSAJES_INFORME_TRAS_SUBSANACION_ES } from "./i18n-informe-tras-subsanacion.js?v=20260926-reparos-informe-v1";
import { justificanteTraducido } from "../../portal-justificante.js";

const ACCION_SUBSANACION = "contratacion_temporal.subsanacion_reparos.registrar";
const ACCION_INFORME = "contratacion_temporal.informe_juridico.generar";
const PATRON_REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;

function ultimoHitoVigente(estado) {
  const hito = estado?.expediente?.historial?.at?.(-1);
  return hito && hito.version_expediente === estado.expediente.version
    && hito.secuencia === estado.expediente.version ? hito : null;
}

/** El último hito es el informe nuevo emitido en la subsanación. */
export function informeNuevoEmitidoEnSubsanacion(estado) {
  const hito = ultimoHitoVigente(estado);
  return hito?.accion_clave === ACCION_INFORME && hito.fase_destino === "subsanacion_unidad";
}

/** Contexto del botón: subsanación en incidencia con la subsanación como último hito. */
export function contextoInformeTrasSubsanacionDesdeEstado(estado) {
  if (estado?.vista !== "expediente" || estado.carga !== "listo"
    || estado.expediente?.demostracion !== false || estado.cuadro?.demostracion !== false
    || !Array.isArray(estado.cuadro?.expedientes)) return null;
  const resumen = estado.cuadro.expedientes.find(
    ({ expediente_ref: referencia }) => referencia === estado.expediente.expediente_ref,
  );
  if (resumen?.fase_clave !== "subsanacion_unidad" || resumen.estado_clave !== "incidencia"
    || resumen.version !== estado.expediente.version
    || ultimoHitoVigente(estado)?.accion_clave !== ACCION_SUBSANACION
    || !PATRON_REFERENCIA.test(estado.expediente.expediente_ref)) return null;
  return Object.freeze({
    expediente_ref: estado.expediente.expediente_ref,
    version_esperada: estado.expediente.version,
  });
}

function indeterminado(error) {
  try { return error?.resultadoIndeterminado !== false; } catch { return true; }
}

export function montarFormularioInformeTrasSubsanacion({
  raiz, cliente, contexto, confirmarOperacion = () => false, mensajes = {},
  locale = "es-ES", zonaHoraria = "Europe/Madrid", anunciar = () => {}, alConfirmar = () => {},
  generarClaveIdempotencia = () => globalThis.crypto?.randomUUID?.(),
} = {}) {
  if (!raiz || typeof raiz.addEventListener !== "function" || typeof raiz.querySelector !== "function"
    || typeof cliente?.prepararInformeJuridico !== "function" || typeof confirmarOperacion !== "function"
    || typeof anunciar !== "function" || typeof alConfirmar !== "function"
    || typeof generarClaveIdempotencia !== "function" || typeof AbortController !== "function"
    || !PATRON_REFERENCIA.test(contexto?.expediente_ref ?? "")
    || !Number.isSafeInteger(contexto?.version_esperada) || contexto.version_esperada < 1) {
    throw new TypeError("dependencias del informe tras subsanación no válidas");
  }
  const t = crearTraductorContratacionTemporal({ ...MENSAJES_INFORME_TRAS_SUBSANACION_ES, ...mensajes });
  const formateador = new Intl.DateTimeFormat(locale, { dateStyle: "long", timeStyle: "medium", timeZone: zonaHoraria });
  let montado = true;
  let solicitud = null;
  let controlador = null;
  let estado = { ocupado: false, indeterminado: false, recibo: null, clave: "informe_nuevo_estado_listo", tipo: "informacion" };

  function pintar(foco = "") {
    if (!montado) return;
    const acciones = estado.recibo ? "" : `<div class="ct-acciones"><button class="${
      estado.indeterminado ? "boton-secundario" : "boton-primario"}" type="button" data-ct-informe-nuevo-accion="${
      estado.indeterminado ? "recuperar" : "emitir"}"${estado.ocupado ? " disabled" : ""}>${
      escaparHTML(t(estado.indeterminado ? "informe_nuevo_recuperar" : "informe_nuevo_emitir"))}</button></div>`;
    const recibo = estado.recibo ? `<section class="ct-recibo" data-ct-informe-nuevo-recibo tabindex="-1">
      <h3>${escaparHTML(t("informe_nuevo_recibo_titulo"))}</h3>
      <dl><div><dt>${escaparHTML(t("informe_nuevo_recibo_version"))}</dt><dd>${estado.recibo.version_resultante}</dd></div>
      <div><dt>${escaparHTML(t("informe_nuevo_recibo_referencia"))}</dt><dd>${justificanteTraducido(estado.recibo.recibo_ref, escaparHTML, t)}</dd></div>
      <div><dt>${escaparHTML(t("informe_nuevo_recibo_fecha"))}</dt><dd><time datetime="${escaparHTML(estado.recibo.confirmada_en)}">${
      escaparHTML(formateador.format(new Date(estado.recibo.confirmada_en)))}</time></dd></div></dl>
    </section>` : "";
    raiz.innerHTML = `<section class="ct-bloque" data-ct-informe-nuevo aria-labelledby="ct-informe-nuevo-titulo">
      <h3 id="ct-informe-nuevo-titulo">${escaparHTML(t("informe_nuevo_titulo"))}</h3>
      <div class="ct-estado ct-estado-${escaparHTML(estado.tipo)}" data-ct-informe-nuevo-estado role="status"
        aria-live="polite" aria-atomic="true" tabindex="-1"><strong>${escaparHTML(t(estado.clave))}</strong></div>
      ${acciones}${recibo}
    </section>`;
    if (foco) raiz.querySelector(foco)?.focus?.();
    try { anunciar(t(estado.clave), estado.tipo); } catch { /* la región viva conserva el estado */ }
  }

  async function emitir(recuperando) {
    if (!montado || estado.ocupado || estado.recibo) return null;
    try {
      solicitud ??= validarSolicitudInformeJuridico(JSON.stringify({
        expediente_ref: contexto.expediente_ref,
        version_esperada: contexto.version_esperada,
        clave_idempotencia: generarClaveIdempotencia(),
      }));
    } catch {
      estado = { ...estado, clave: "informe_nuevo_estado_rechazado", tipo: "error" };
      pintar("[data-ct-informe-nuevo-estado]");
      return null;
    }
    controlador = new AbortController();
    estado = { ...estado, ocupado: true, clave: "informe_nuevo_estado_enviando", tipo: "informacion", indeterminado: recuperando };
    pintar("[data-ct-informe-nuevo-estado]");
    try {
      const recibo = await cliente.prepararInformeJuridico(solicitud, Object.freeze({ signal: controlador.signal }));
      if (!montado) return null;
      estado = { ocupado: false, indeterminado: false, recibo, clave: "informe_nuevo_estado_confirmado", tipo: "exito" };
      solicitud = null;
      pintar("[data-ct-informe-nuevo-recibo]");
      try { alConfirmar(recibo); } catch { /* el recibo prevalece aunque falle el refresco */ }
      return recibo;
    } catch (error) {
      if (!montado) return null;
      const noPrevisto = error?.codigo === "informe_nuevo_no_previsto";
      const incierto = !noPrevisto && indeterminado(error);
      if (!incierto) solicitud = null;
      estado = {
        ocupado: false, indeterminado: incierto, recibo: null,
        clave: noPrevisto ? "informe_nuevo_estado_no_previsto"
          : incierto ? "informe_nuevo_estado_indeterminado" : "informe_nuevo_estado_rechazado",
        tipo: noPrevisto ? "informacion" : incierto ? "aviso" : "error",
      };
      pintar("[data-ct-informe-nuevo-estado]");
      return null;
    } finally {
      controlador = null;
    }
  }

  function alPulsar(evento) {
    const control = evento.target?.closest?.("[data-ct-informe-nuevo-accion]");
    if (!control || !raiz.contains(control)) return undefined;
    evento.preventDefault();
    if (control.dataset.ctInformeNuevoAccion === "recuperar") return solicitud ? emitir(true) : Promise.resolve(null);
    let confirmada = false;
    try {
      confirmada = confirmarOperacion({
        titulo: t("informe_nuevo_emitir"),
        advertencia: t("informe_nuevo_confirmacion_advertencia"),
        referencia: contexto.expediente_ref,
      }) === true;
    } catch { confirmada = false; }
    return confirmada ? emitir(false) : Promise.resolve(null);
  }

  raiz.addEventListener("click", alPulsar);
  pintar();
  return function desmontarFormularioInformeTrasSubsanacion() {
    if (!montado) return;
    montado = false;
    controlador?.abort();
    raiz.removeEventListener("click", alPulsar);
    raiz.replaceChildren();
    solicitud = null;
  };
}

/** Monta el botón en la subsanación y, emitido el informe, vuelve a leer el
 * expediente para que la vista muestre la nueva fiscalización. */
export function crearGestorInformeTrasSubsanacion({
  raiz, presentador, cliente, disponible = false, confirmarOperacion = () => false,
  mensajes = {}, locale = "es-ES", zonaHoraria = "Europe/Madrid", anunciar = () => {},
  repintar = () => {}, esMontada = () => true,
} = {}) {
  let desmontar = null;
  const t = crearTraductorContratacionTemporal({ ...MENSAJES_INFORME_TRAS_SUBSANACION_ES, ...mensajes });

  async function refrescar(recibo, panel) {
    const sigue = () => esMontada() && raiz.querySelector("[data-ct-exp-informe-nuevo]") === panel;
    try {
      await presentador.cargar();
      if (!sigue()) return;
      await presentador.seleccionarExpediente(recibo.expediente_ref, "expediente");
      if (!sigue()) return;
      if (presentador.obtenerEstado().expediente?.version !== recibo.version_resultante) {
        anunciar(t("informe_nuevo_estado_actualizacion_pendiente"), "aviso");
        return;
      }
      repintar("[data-ct-exp-fiscalizacion]");
    } catch {
      if (sigue()) anunciar(t("informe_nuevo_estado_actualizacion_pendiente"), "aviso");
    }
  }

  return Object.freeze({
    montarSiProcede() {
      if (!disponible || desmontar !== null || !esMontada()) return false;
      const contexto = contextoInformeTrasSubsanacionDesdeEstado(presentador.obtenerEstado());
      const panel = raiz.querySelector("[data-ct-exp-informe-nuevo]");
      if (!contexto || !panel) return false;
      try {
        desmontar = montarFormularioInformeTrasSubsanacion({
          raiz: panel, cliente, contexto, confirmarOperacion, mensajes, locale, zonaHoraria, anunciar,
          alConfirmar: (recibo) => { void refrescar(recibo, panel); },
        });
        return true;
      } catch {
        desmontar = null;
        return false;
      }
    },
    retirar() {
      desmontar?.();
      desmontar = null;
    },
  });
}

/** Aviso del detalle cuando el informe nuevo ya está emitido y falta fiscalizar. */
export function renderizarAvisoInformeNuevoEmitido(mensajes = {}) {
  const t = crearTraductorContratacionTemporal({ ...MENSAJES_INFORME_TRAS_SUBSANACION_ES, ...mensajes });
  return `<p class="ct-exp-mensaje ct-tono-informacion" role="status">${escaparHTML(t("informe_nuevo_pendiente_fiscalizacion"))}</p>`;
}
