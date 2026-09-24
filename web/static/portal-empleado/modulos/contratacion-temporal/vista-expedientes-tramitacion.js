/** Montaje y gestión de estados de las fases de tramitación (alta, análisis, cobertura, asignación, informe, fiscalización y subsanación). */

import { escaparHTML } from "./componentes-expedientes.js";
import { montarFormularioAnalisisRRHH } from "./formulario-analisis.js";
import { montarFormularioAsignacion } from "./formulario-asignacion.js";
import { montarFormularioCobertura } from "./formulario-cobertura.js";
import { montarFormularioFiscalizacion } from "./formulario-fiscalizacion.js";
import { montarFormularioInformeJuridico } from "./formulario-informe-juridico.js";
import { montarFormularioSubsanacionReparos } from "./formulario-subsanacion-reparos.js";
import { validarReciboSubsanacionReparos, validarSolicitudSubsanacionReparos } from "./cliente-http-subsanacion-reparos.js";
import { crearTraductorExpedientesContratacion } from "./i18n-expedientes.js";
import { crearTraductorContratacionTemporal } from "./i18n.js";
import { crearPresentadorAltaContratacionTemporal } from "./presentador.js";
import { crearClienteAnalisisCercado, PATRON_REFERENCIA } from "./vista-expedientes-analisis.js";
import {
  contextoAsignacionDesdeEstado, contextoCoberturaDesdeEstado,
  contextoFiscalizacionDesdeEstado, contextoInformeJuridicoDesdeEstado,
  contextoRectificacionAnalisisDesdeEstado, contextoSubsanacionDesdeEstado,
} from "./vista-expedientes-render.js";
import { montarAltaContratacionTemporal } from "./vista.js";

function enfocarElemento(raiz, selector) {
  const elemento = raiz.querySelector(selector);
  elemento?.focus?.();
  elemento?.scrollIntoView?.({ block: "nearest", inline: "nearest" });
}

export function crearGestorTramitacion({
  raiz,
  presentador,
  alta,
  altaDisponible,
  composicionAnalisis,
  rectificacionDisponible,
  coberturaDisponible,
  asignacionDisponible,
  informeJuridicoDisponible,
  clienteFiscalizacion,
  fiscalizacionDisponible,
  clienteSubsanacion,
  subsanacionDisponible,
  crearEjecutorAltaConRefresco,
  alFiscalizacionConfirmadaLlamamiento = () => {},
  confirmarOperacion = () => false,
  mensajes = {},
  locale = "es-ES",
  zonaHoraria = "Europe/Madrid",
  anunciar = () => {},
  repintar = () => {},
  esMontada = () => true,
} = {}) {
  const tExpedientes = crearTraductorExpedientesContratacion(mensajes);
  let desmontarAlta = null;
  let desmontarAnalisis = null;
  let desmontarCobertura = null;
  let desmontarAsignacion = null;
  let desmontarInformeJuridico = null;
  let desmontarFiscalizacion = null;
  let desmontarSubsanacion = null;
  const intencionesSubsanacion = new Map();
  const recibosSubsanacion = new Map();
  let reciboFiscalizacionConfirmado = null;
  let reciboAsignacionConfirmado = null;
  let sesionAnalisis = null;
  const desmontarEstadosMontaje = new Set();
  const estadosMontajePorContenedor = new Map();

  function invalidarSubsanacionPorDenegacion() {
    intencionesSubsanacion.clear();
    recibosSubsanacion.clear();
  }

  function mostrarErrorMontaje(contenedor, etapa, reintentar) {
    if (!esMontada() || !contenedor || typeof reintentar !== "function") return;
    const existentes = estadosMontajePorContenedor.get(contenedor);
    if (existentes?.has(etapa)) return;
    const documento = contenedor.ownerDocument ?? raiz.ownerDocument;
    if (!documento?.createElement || typeof contenedor.append !== "function") {
      anunciar(tExpedientes("montaje_siguiente_pendiente"), "error");
      return;
    }
    const bloque = documento.createElement("section");
    const boton = documento.createElement("button");
    const limpiar = () => {
      boton.removeEventListener?.("click", manejarReintento);
      bloque.remove?.();
      desmontarEstadosMontaje.delete(limpiar);
      const restantes = estadosMontajePorContenedor.get(contenedor);
      restantes?.delete(etapa);
      if (restantes?.size === 0) estadosMontajePorContenedor.delete(contenedor);
    };
    const manejarReintento = () => {
      if (!esMontada()) return;
      limpiar();
      try {
        if (!reintentar()) mostrarErrorMontaje(contenedor, etapa, reintentar);
      } catch {
        mostrarErrorMontaje(contenedor, etapa, reintentar);
      }
    };
    bloque.setAttribute?.("class", "ct-estado ct-estado-aviso");
    bloque.setAttribute?.("role", "alert");
    bloque.setAttribute?.("aria-live", "assertive");
    bloque.setAttribute?.("data-ct-exp-montaje-pendiente", etapa);
    const titulo = documento.createElement("p");
    titulo.textContent = tExpedientes("montaje_siguiente_pendiente");
    const detalle = documento.createElement("p");
    detalle.textContent = tExpedientes("montaje_siguiente_pendiente_detalle");
    boton.type = "button";
    boton.className = "boton-secundario";
    boton.textContent = tExpedientes("montaje_siguiente_reintentar");
    boton.setAttribute?.("data-ct-exp-reintentar-montaje", etapa);
    boton.addEventListener?.("click", manejarReintento);
    bloque.append(titulo, detalle, boton);
    contenedor.append(bloque);
    desmontarEstadosMontaje.add(limpiar);
    const porEtapa = estadosMontajePorContenedor.get(contenedor) ?? new Map();
    porEtapa.set(etapa, limpiar);
    estadosMontajePorContenedor.set(contenedor, porEtapa);
    anunciar(titulo.textContent, "error");
  }

  function limpiarEstadosMontaje() {
    for (const desmontar of [...desmontarEstadosMontaje]) desmontar();
  }

  function bloquearControlesAnalisis(sesion) {
    if (sesion.controles === null) {
      const controles = typeof raiz.querySelectorAll === "function"
        ? [...raiz.querySelectorAll(
          "[data-ct-exp-vista], [data-ct-exp-abrir], [data-ct-exp-tarea], [data-ct-exp-efecto], [data-ct-exp-accion]",
        )] : [];
      sesion.controles = controles.map((control) => ({
        control,
        deshabilitado: control.disabled === true,
        aria: control.getAttribute?.("aria-disabled") ?? null,
      }));
      for (const registro of sesion.controles) {
        registro.control.disabled = true;
        registro.control.setAttribute?.("aria-disabled", "true");
      }
      sesion.ariaBusy = raiz.getAttribute?.("aria-busy") ?? null;
    }
    raiz.setAttribute?.("aria-busy", "true");
  }

  function restaurarOcupacionAnalisis(sesion) {
    if (!sesion || sesion.controles === null) return;
    if (sesion.ariaBusy === null) raiz.removeAttribute?.("aria-busy");
    else raiz.setAttribute?.("aria-busy", sesion.ariaBusy);
  }

  function restaurarControlesAnalisis(sesion) {
    if (!sesion || sesion.controles === null) return;
    for (const registro of sesion.controles) {
      registro.control.disabled = registro.deshabilitado;
      if (registro.aria === null) registro.control.removeAttribute?.("aria-disabled");
      else registro.control.setAttribute?.("aria-disabled", registro.aria);
    }
    restaurarOcupacionAnalisis(sesion);
    sesion.controles = null;
  }

  function cambiarEtapaAnalisis(sesion, etapa, vuelo) {
    if (!esMontada() || sesionAnalisis !== sesion) return;
    if (etapa === "transmitiendo") {
      sesion.intentoIniciado = true;
      sesion.vuelo = vuelo;
      bloquearControlesAnalisis(sesion);
    } else if (sesion.vuelo !== vuelo) {
      return;
    }
    sesion.etapa = etapa;
    if (etapa !== "transmitiendo") restaurarOcupacionAnalisis(sesion);
  }

  function analisisEstableActivo() {
    return sesionAnalisis?.intentoIniciado === true;
  }

  function anunciarBloqueoAnalisis() {
    const clave = {
      transmitiendo: "estado_registrando_actuacion",
      reintentable: "estado_error_actuacion",
      indeterminado: "estado_resultado_indeterminado",
      confirmado: "estado_confirmada_actualizacion_pendiente",
    }[sesionAnalisis?.etapa] ?? "estado_resultado_indeterminado";
    anunciar(crearTraductorExpedientesContratacion(mensajes)(clave), "aviso");
    enfocarElemento(raiz, "[data-ct-analisis-estado]");
  }

  function impedirCambioPorAnalisis() {
    if (!analisisEstableActivo()) return false;
    anunciarBloqueoAnalisis();
    return true;
  }

  function retirarAlta() {
    if (typeof desmontarAlta === "function") desmontarAlta();
    desmontarAlta = null;
  }

  function retirarAnalisis() {
    if (typeof desmontarAnalisis === "function") desmontarAnalisis();
    desmontarAnalisis = null;
    restaurarControlesAnalisis(sesionAnalisis);
    sesionAnalisis = null;
  }

  function retirarCobertura() {
    if (typeof desmontarCobertura === "function") desmontarCobertura();
    desmontarCobertura = null;
  }

  function retirarAsignacion() {
    if (typeof desmontarFiscalizacion === "function") desmontarFiscalizacion();
    desmontarFiscalizacion = null;
    desmontarSubsanacion?.();
    desmontarSubsanacion = null;
    if (typeof desmontarInformeJuridico === "function") desmontarInformeJuridico();
    desmontarInformeJuridico = null;
    if (typeof desmontarAsignacion === "function") desmontarAsignacion();
    desmontarAsignacion = null;
  }

  function montarFiscalizacion(contexto) {
    if (!esMontada() || !fiscalizacionDisponible || desmontarFiscalizacion !== null) {
      return desmontarFiscalizacion !== null;
    }
    const contenedor = raiz.querySelector("[data-ct-exp-fiscalizacion]");
    if (!contenedor) return false;
    try {
      desmontarFiscalizacion = montarFormularioFiscalizacion({
        raiz: contenedor,
        cliente: clienteFiscalizacion,
        contexto,
        confirmarOperacion,
        mensajes,
        locale,
        zonaHoraria,
        anunciar,
        alConfirmar: (recibo) => {
          try {
            if (recibo.resultado !== "desfavorable" && recibo.version_resultante >= 6) {
              alFiscalizacionConfirmadaLlamamiento(recibo);
            }
          } finally {
            void refrescarDetalleTrasFiscalizacion(recibo);
          }
        },
      });
      return true;
    } catch {
      desmontarFiscalizacion = null;
      return false;
    }
  }

  function montarFiscalizacionDesdeInforme(recibo) {
    return montarFiscalizacion(Object.freeze({
      expediente_ref: recibo.expediente_ref,
      version_esperada: recibo.version_resultante,
      fase_clave: "informe_juridico",
      informe_ref: recibo.informe_ref,
    }));
  }

  function intencionSubsanacionParaEstado(estado, contexto = contextoSubsanacionDesdeEstado(estado)) {
    if (!contexto) return null;
    const intencion = intencionesSubsanacion.get(contexto.expediente_ref);
    if (!intencion) return null;
    const versionOriginal = intencion.solicitud.version_esperada;
    // Una lectura autorizada puede avanzar varias versiones mientras el POST
    // original sigue incierto. Esa lectura no libera su clave ni sus datos.
    return contexto.version_esperada >= versionOriginal ? intencion : null;
  }

  function subsanacionRegistradaEnEstado(estado, contexto = contextoSubsanacionDesdeEstado(estado)) {
    if (!contexto || contexto.version_esperada <= 1) return false;
    const ultimoHito = estado.expediente?.historial?.at(-1);
    return ultimoHito?.accion_clave === "contratacion_temporal.subsanacion_reparos.registrar"
      && ultimoHito.version_expediente === contexto.version_esperada
      && ultimoHito.secuencia === contexto.version_esperada;
  }

  function versionesRecuperacionAnteriores(estado, contexto) {
    const versiones = (estado.expediente?.historial ?? [])
      .filter((hito) => hito.accion_clave === "contratacion_temporal.subsanacion_reparos.registrar"
        && Number.isSafeInteger(hito.version_expediente)
        && hito.version_expediente === hito.secuencia
        && hito.version_expediente > 1
        && hito.version_expediente <= contexto.version_esperada)
      .map((hito) => hito.version_expediente - 1);
    return Object.freeze([...new Set(versiones)]);
  }

  function montarSubsanacionDesdeExpedienteActual() {
    if (!esMontada() || !subsanacionDisponible || desmontarSubsanacion !== null) return;
    const estado = presentador.obtenerEstado();
    const contextoActual = contextoSubsanacionDesdeEstado(estado);
    const contenedor = raiz.querySelector("[data-ct-exp-subsanacion]")
      ?? raiz.querySelector("[data-ct-exp-recuperar-archivo]");
    if (!contextoActual || !contenedor) return;
    const intencionInicial = intencionSubsanacionParaEstado(estado, contextoActual);
    const reciboGuardado = recibosSubsanacion.get(contextoActual.expediente_ref);
    const reciboConfirmado = reciboGuardado
      && [reciboGuardado.contexto.version_esperada, reciboGuardado.recibo.version_resultante].includes(contextoActual.version_esperada)
      ? reciboGuardado : null;
    const soloImportar = !intencionInicial && !reciboConfirmado
      && subsanacionRegistradaEnEstado(estado, contextoActual);
    const contexto = intencionInicial || soloImportar ? Object.freeze({
      expediente_ref: contextoActual.expediente_ref,
      version_esperada: intencionInicial?.solicitud.version_esperada
        ?? contextoActual.version_esperada - 1,
    }) : contextoActual;
    const versionesAnteriores = Object.freeze(versionesRecuperacionAnteriores(estado, contextoActual)
      .filter((version) => version < contexto.version_esperada));
    try {
      desmontarSubsanacion = montarFormularioSubsanacionReparos({
        raiz: contenedor, cliente: clienteSubsanacion, contexto,
        traducir: crearTraductorContratacionTemporal(mensajes), confirmarOperacion,
        anunciar, reciboConfirmado, intencionInicial, soloImportar,
        versionesRecuperacionAnteriores: versionesAnteriores,
        alDenegacion: () => {
          invalidarSubsanacionPorDenegacion();
          // Una denegación de un POST antiguo puede llegar después de navegar.
          // Retirar el formulario activo elimina también su copia local del DTO.
          const panelActual = raiz.querySelector("[data-ct-exp-subsanacion]")
            ?? raiz.querySelector("[data-ct-exp-recuperar-archivo]");
          if (esMontada() && panelActual && panelActual !== contenedor) repintar();
        },
        alCambiarIntencion: (intencion) => {
          if (intencion === null) {
            // El formulario confirma antes de retirar la intención. Un aborto o
            // un error nunca pueden hacer perder la clave de recuperación.
            return !intencionesSubsanacion.has(contexto.expediente_ref)
              && recibosSubsanacion.has(contexto.expediente_ref);
          }
          if (!intencion || typeof intencion.incierta !== "boolean") return false;
          try {
            const solicitud = validarSolicitudSubsanacionReparos(intencion.solicitud);
            if (solicitud.expediente_ref !== contexto.expediente_ref
              || (solicitud.version_esperada !== contexto.version_esperada
                && !(intencion.incierta && versionesAnteriores.includes(solicitud.version_esperada)))) return false;
            const anterior = intencionesSubsanacion.get(contexto.expediente_ref);
            if (anterior && (anterior.solicitud.version_esperada !== solicitud.version_esperada
              || anterior.solicitud.clave_idempotencia !== solicitud.clave_idempotencia
              || anterior.solicitud.observaciones !== solicitud.observaciones
              || (anterior.incierta && !intencion.incierta))) return false;
            intencionesSubsanacion.set(contexto.expediente_ref, Object.freeze({
              solicitud, incierta: intencion.incierta,
            }));
            return true;
          } catch { return false; }
        },
        alConfirmar: refrescarDetalleTrasSubsanacion,
      });
    } catch {
      desmontarSubsanacion = null;
      const t = crearTraductorContratacionTemporal(mensajes);
      contenedor.innerHTML = `<p role="alert">${escaparHTML(t("subsanacion_montaje_error"))}</p>`;
    }
  }

  async function refrescarDetalleTrasSubsanacion(recibo, contextoOriginal) {
    if (!esMontada() || recibo?.expediente_ref !== contextoOriginal?.expediente_ref
      || recibo.version_resultante <= contextoOriginal.version_esperada) return;
    const intencion = intencionesSubsanacion.get(contextoOriginal.expediente_ref);
    if (!intencion || intencion.solicitud.version_esperada !== contextoOriginal.version_esperada) return;
    let validado;
    try { validado = validarReciboSubsanacionReparos(recibo, intencion.solicitud); }
    catch { return; }
    recibosSubsanacion.set(contextoOriginal.expediente_ref, Object.freeze({
      recibo: validado,
      contexto: Object.freeze({ expediente_ref: contextoOriginal.expediente_ref,
        version_esperada: contextoOriginal.version_esperada }),
    }));
    intencionesSubsanacion.delete(contextoOriginal.expediente_ref);
    const seleccionado = presentador.obtenerEstado();
    if (seleccionado.vista !== "expediente"
      || seleccionado.expediente?.expediente_ref !== contextoOriginal.expediente_ref) return;
    const panel = raiz.querySelector("[data-ct-exp-subsanacion]")
      ?? raiz.querySelector("[data-ct-exp-recuperar-archivo]");
    const sigueSeleccionado = () => esMontada() && panel !== null
      && (raiz.querySelector("[data-ct-exp-subsanacion]")
        ?? raiz.querySelector("[data-ct-exp-recuperar-archivo]")) === panel;
    const avisarPendiente = () => {
      if (sigueSeleccionado()) anunciar(
        crearTraductorContratacionTemporal(mensajes)("subsanacion_actualizacion_pendiente"), "aviso",
      );
    };
    try {
      await presentador.cargar();
      if (!sigueSeleccionado()) return;
      const resumen = presentador.obtenerEstado().cuadro?.expedientes?.find(
        ({ expediente_ref: referencia }) => referencia === contextoOriginal.expediente_ref,
      );
      if (!resumen || resumen.version < recibo.version_resultante) { avisarPendiente(); return; }
      await presentador.seleccionarExpediente(contextoOriginal.expediente_ref, "expediente");
      if (!sigueSeleccionado()) return;
      const actualizado = presentador.obtenerEstado().expediente;
      if (actualizado?.expediente_ref !== contextoOriginal.expediente_ref
        || actualizado.version < recibo.version_resultante) { avisarPendiente(); return; }
      // Un reparo posterior no debe borrar el recibo recién recuperado al
      // repintar: la actuación nueva se abrirá al volver al expediente.
      if (actualizado.version > recibo.version_resultante) return;
      repintar("[data-ct-subsanacion-recibo]");
    } catch {
      avisarPendiente();
    }
  }

  async function refrescarDetalleTrasFiscalizacion(recibo) {
    const seleccionado = presentador.obtenerEstado();
    if (!esMontada() || recibo?.expediente_ref !== seleccionado.expediente?.expediente_ref
      || !Number.isSafeInteger(recibo?.version_resultante)
      || recibo.version_resultante <= seleccionado.expediente.version) return false;
    const panel = raiz.querySelector("[data-ct-exp-fiscalizacion]");
    if (panel === null) return false;
    reciboFiscalizacionConfirmado = Object.freeze({ ...recibo });
    const sigueSeleccionado = () => esMontada()
      && raiz.querySelector("[data-ct-exp-fiscalizacion]") === panel;
    const avisarPendiente = () => {
      if (sigueSeleccionado()) anunciar(
        crearTraductorContratacionTemporal(mensajes)("fiscalizacion_actualizacion_pendiente"),
        "aviso",
      );
    };
    try {
      await presentador.cargar();
      if (!sigueSeleccionado()) return false;
      const resumen = presentador.obtenerEstado().cuadro?.expedientes?.find(
        ({ expediente_ref: referencia }) => referencia === recibo.expediente_ref,
      );
      if (!resumen || resumen.version < recibo.version_resultante) {
        avisarPendiente();
        return false;
      }
      await presentador.seleccionarExpediente(recibo.expediente_ref, "expediente");
      if (!sigueSeleccionado()) return false;
      const actualizado = presentador.obtenerEstado().expediente;
      if (actualizado?.expediente_ref !== recibo.expediente_ref
        || actualizado.version !== recibo.version_resultante) {
        avisarPendiente();
        return false;
      }
      repintar("[data-ct-exp-mensaje]");
      return true;
    } catch {
      avisarPendiente();
      return false;
    }
  }

  function montarInformeDesdeAsignacion(recibo) {
    if (!esMontada() || !informeJuridicoDisponible || desmontarInformeJuridico !== null) {
      return desmontarInformeJuridico !== null;
    }
    const contenedor = raiz.querySelector("[data-ct-exp-informe-juridico]");
    if (!contenedor) return false;
    try {
      desmontarInformeJuridico = montarFormularioInformeJuridico({
        raiz: contenedor, cliente: composicionAnalisis.cliente,
        contexto: Object.freeze({
          expediente_ref: recibo.expediente_ref,
          version_esperada: recibo.version_resultante,
        }),
        confirmarOperacion, mensajes, locale, zonaHoraria, anunciar,
        alConfirmar: montarFiscalizacionDesdeInforme,
      });
      return true;
    } catch {
      desmontarInformeJuridico = null;
      mostrarErrorMontaje(
        raiz.querySelector("[data-ct-exp-asignacion]"),
        "informe-juridico",
        () => montarInformeDesdeAsignacion(recibo),
      );
      return false;
    }
  }

  async function refrescarDetalleTrasAsignacion(recibo, solicitud) {
    const seleccionado = presentador.obtenerEstado();
    if (!esMontada() || recibo?.expediente_ref !== seleccionado.expediente?.expediente_ref
      || !Number.isSafeInteger(recibo?.version_resultante)
      || solicitud?.expediente_ref !== recibo.expediente_ref
      || solicitud.version_esperada + 1 !== recibo.version_resultante
      || typeof solicitud.unidad_ref !== "string" || !PATRON_REFERENCIA.test(solicitud.unidad_ref)) return false;
    const panel = raiz.querySelector("[data-ct-exp-asignacion]");
    if (panel === null) return false;
    reciboAsignacionConfirmado = Object.freeze({ ...recibo });
    const sigueSeleccionado = () => esMontada()
      && raiz.querySelector("[data-ct-exp-asignacion]") === panel;
    const avisarPendiente = () => {
      if (sigueSeleccionado()) anunciar(
        crearTraductorExpedientesContratacion(mensajes)("estado_actualizacion_pendiente"), "aviso",
      );
    };
    try {
      await presentador.cargar();
      if (!sigueSeleccionado()) return false;
      const resumen = presentador.obtenerEstado().cuadro?.expedientes?.find(
        ({ expediente_ref: referencia }) => referencia === recibo.expediente_ref,
      );
      if (!resumen || resumen.version < recibo.version_resultante) { avisarPendiente(); return false; }
      await presentador.seleccionarExpediente(recibo.expediente_ref, "expediente");
      if (!sigueSeleccionado()) return false;
      const actualizado = presentador.obtenerEstado().expediente;
      if (actualizado?.expediente_ref !== recibo.expediente_ref
        || actualizado.version !== recibo.version_resultante
        || actualizado.cabecera?.find(({ clave }) => clave === "unidad")?.valor !== solicitud.unidad_ref) { avisarPendiente(); return false; }
      repintar("[data-ct-asignacion-confirmada]");
      return true;
    } catch {
      avisarPendiente();
      return false;
    }
  }

  function montarFiscalizacionDesdeExpedienteActual() {
    const contexto = contextoFiscalizacionDesdeEstado(presentador.obtenerEstado());
    if (contexto === null) return null;
    return montarFiscalizacion(contexto);
  }

  function montarInformeDesdeExpedienteActual() {
    const contextoInforme = contextoInformeJuridicoDesdeEstado(
      presentador.obtenerEstado(),
    );
    if (contextoInforme === null) return null;
    return montarInformeDesdeAsignacion({
      expediente_ref: contextoInforme.expediente_ref,
      version_resultante: contextoInforme.version_esperada,
    });
  }

  function montarAsignacionDesdeCobertura(expedienteRef, recibo) {
    if (!esMontada() || !asignacionDisponible) return false;
    if (desmontarAsignacion !== null) return true;
    const contenedor = raiz.querySelector("[data-ct-exp-asignacion]");
    if (!contenedor) return false;
    try {
      desmontarAsignacion = montarFormularioAsignacion({
        raiz: contenedor,
        cliente: composicionAnalisis.cliente,
        contexto: Object.freeze({
          expediente_ref: expedienteRef,
          version_esperada: recibo.version_resultante,
        }),
        confirmarOperacion,
        mensajes,
        locale,
        zonaHoraria,
        anunciar,
        alConfirmar: refrescarDetalleTrasAsignacion,
      });
      return true;
    } catch {
      desmontarAsignacion = null;
      mostrarErrorMontaje(
        raiz.querySelector("[data-ct-exp-cobertura]"),
        "asignacion",
        () => montarAsignacionDesdeCobertura(expedienteRef, recibo),
      );
      return false;
    }
  }

  function montarAsignacionDesdeEstado() {
    const contexto = contextoAsignacionDesdeEstado(presentador.obtenerEstado());
    if (contexto === null) return false;
    return montarAsignacionDesdeCobertura(contexto.expediente_ref, {
      version_resultante: contexto.version_esperada,
    });
  }

  function montarCoberturaDesdeEstado() {
    const contexto = contextoCoberturaDesdeEstado(presentador.obtenerEstado());
    if (contexto === null) return false;
    return montarCoberturaDesdeAnalisis({
      expediente_ref: contexto.expediente_ref,
      version_resultante: contexto.version_esperada,
    });
  }

  function montarCoberturaDesdeAnalisis(recibo) {
    if (!esMontada() || !coberturaDisponible || desmontarCobertura !== null) return null;
    const contenedor = raiz.querySelector("[data-ct-exp-cobertura]");
    if (!contenedor) return null;
    try {
      desmontarCobertura = montarFormularioCobertura({
        raiz: contenedor,
        cliente: composicionAnalisis.cliente,
        contexto: Object.freeze({
          expediente_ref: recibo.expediente_ref,
          version_esperada: recibo.version_resultante,
        }),
        confirmarOperacion,
        mensajes,
        locale,
        zonaHoraria,
        anunciar,
        alConfirmar: (reciboCobertura) => montarAsignacionDesdeCobertura(
          recibo.expediente_ref,
          reciboCobertura,
        ),
      });
      return true;
    } catch {
      desmontarCobertura = null;
      mostrarErrorMontaje(
        raiz.querySelector("[data-ct-exp-analisis]"),
        "cobertura",
        () => montarCoberturaDesdeAnalisis(recibo),
      );
      return false;
    }
  }

  function montarAltaSiProcede() {
    const estado = presentador.obtenerEstado();
    if (!esMontada() || estado.vista !== "alta") return;
    const contenedor = raiz.querySelector("[data-ct-exp-alta]");
    if (!contenedor) return;
    if (!altaDisponible || !alta?.catalogos || typeof alta?.ejecutor !== "function") {
      contenedor.innerHTML = `<section class="ct-exp-estado-global ct-tono-peligro" role="alert" tabindex="-1"><h3>${escaparHTML(tExpedientes("catalogo_no_disponible_titulo"))}</h3><p>${escaparHTML(tExpedientes("catalogo_no_disponible_detalle"))}</p><div class="ct-exp-acciones-estado"><button type="button" class="boton-secundario" data-ct-exp-accion="reintentar">${escaparHTML(tExpedientes("reintentar"))}</button><button type="button" class="boton-secundario" data-ct-exp-vista="cuadro">${escaparHTML(tExpedientes("volver_cuadro"))}</button></div></section>`;
      return;
    }
    try {
      const presentadorAlta = crearPresentadorAltaContratacionTemporal({
        catalogos: alta.catalogos,
        capacidad: alta.capacidad,
        ejecutor: crearEjecutorAltaConRefresco(
          alta.ejecutor,
          presentador,
          montarAnalisisDesdeAlta,
        ),
        generarClaveIdempotencia: alta.generarClaveIdempotencia,
      });
      desmontarAlta = montarAltaContratacionTemporal({
        raiz: contenedor,
        presentador: presentadorAlta,
        anunciar,
        locale,
        zonaHoraria,
      });
    } catch {
      contenedor.innerHTML = `<section class="ct-exp-estado-global ct-tono-peligro" role="alert" tabindex="-1"><h3>${escaparHTML(tExpedientes("catalogo_no_disponible_titulo"))}</h3><p>${escaparHTML(tExpedientes("catalogo_no_disponible_detalle"))}</p><div class="ct-exp-acciones-estado"><button type="button" class="boton-secundario" data-ct-exp-accion="reintentar">${escaparHTML(tExpedientes("reintentar"))}</button><button type="button" class="boton-secundario" data-ct-exp-vista="cuadro">${escaparHTML(tExpedientes("volver_cuadro"))}</button></div></section>`;
    }
  }

  function montarAnalisisEnContenedor(contenedor, contexto, analisisInicial, datosPrevios = null) {
    if (!contenedor) return null;
    if (typeof desmontarAnalisis === "function") return true;
    const sesion = {
      expedienteRef: contexto.expediente_ref,
      intentoIniciado: false,
      etapa: "preparado",
      vuelo: null,
      controles: null,
      ariaBusy: null,
    };
    const clienteCercado = crearClienteAnalisisCercado(
      composicionAnalisis,
      contexto,
      (etapa, vuelo) => cambiarEtapaAnalisis(sesion, etapa, vuelo),
      montarCoberturaDesdeAnalisis,
      (recibo) => mostrarErrorMontaje(
        contenedor,
        "cobertura",
        () => montarCoberturaDesdeAnalisis(recibo),
      ),
    );
    sesionAnalisis = sesion;
    try {
      desmontarAnalisis = montarFormularioAnalisisRRHH({
        raiz: contenedor,
        cliente: clienteCercado,
        contexto,
        catalogos: composicionAnalisis.catalogos,
        analisisInicial,
        datosPrevios,
        mensajes,
        locale,
        zonaHoraria,
        anunciar,
      });
      return true;
    } catch {
      desmontarAnalisis = null;
      restaurarControlesAnalisis(sesion);
      if (sesionAnalisis === sesion) sesionAnalisis = null;
      mostrarErrorMontaje(
        contenedor,
        "analisis",
        () => montarAnalisisEnContenedor(contenedor, contexto, analisisInicial, datosPrevios),
      );
      return null;
    }
  }

  function montarAnalisisDesdeAlta(recibo) {
    const estado = presentador.obtenerEstado();
    if (!esMontada() || estado.vista !== "alta" || composicionAnalisis === null
      || composicionAnalisis.contexto.operacion !== "registrar") return null;
    const contexto = Object.freeze({
      operacion: "registrar",
      expediente_ref: recibo.expediente_ref,
      version_esperada: recibo.version,
      artefacto_ref: composicionAnalisis.contexto.artefacto_ref,
    });
    return montarAnalisisEnContenedor(
      raiz.querySelector("[data-ct-exp-analisis]"),
      contexto,
      null,
    );
  }

  function montarAnalisisSiProcede() {
    const estado = presentador.obtenerEstado();
    if (!esMontada() || estado.vista !== "expediente" || composicionAnalisis === null
      || estado.carga !== "listo" || estado.expediente === null) return null;
    const rectificacion = contextoRectificacionAnalisisDesdeEstado(estado);
    if (rectificacion !== null) {
      const contenedor = raiz.querySelector("[data-ct-exp-rectificacion]");
      if (!contenedor) return null;
      if (!rectificacionDisponible) {
        const t = crearTraductorContratacionTemporal(mensajes);
        contenedor.innerHTML = `<section class="ct-alcance" role="status" aria-live="polite">
          <h3>${escaparHTML(t("analisis_rectificacion_configuracion_pendiente_titulo"))}</h3>
          <p>${escaparHTML(t("analisis_rectificacion_configuracion_pendiente_descripcion"))}</p>
        </section>`;
        return true;
      }
      return montarAnalisisEnContenedor(
        contenedor,
        Object.freeze({
          ...rectificacion,
          artefacto_ref: composicionAnalisis.rectificacion.artefacto_ref,
        }),
        null,
        estado.expediente.analisis_previo ?? null,
      );
    }
    const contexto = Object.freeze({
      operacion: composicionAnalisis.contexto.operacion,
      expediente_ref: estado.expediente.expediente_ref,
      version_esperada: estado.expediente.version,
      artefacto_ref: composicionAnalisis.contexto.artefacto_ref,
    });
    return montarAnalisisEnContenedor(
      raiz.querySelector("[data-ct-exp-analisis]"),
      contexto,
      composicionAnalisis.analisisInicial,
    );
  }

  return Object.freeze({
    montarAltaSiProcede,
    montarAnalisisSiProcede,
    montarCoberturaDesdeEstado,
    montarAsignacionDesdeEstado,
    montarInformeDesdeExpedienteActual,
    montarFiscalizacionDesdeExpedienteActual,
    montarSubsanacionDesdeExpedienteActual,
    impedirCambioPorAnalisis,
    analisisEstableActivo,
    limpiarEstadosMontaje,
    retirarComponentes() {
      limpiarEstadosMontaje();
      retirarAsignacion();
      retirarCobertura();
      retirarAlta();
      retirarAnalisis();
    },
    obtenerReciboSubsanacionConfirmado: (estado = presentador.obtenerEstado()) => {
      const contexto = contextoSubsanacionDesdeEstado(estado);
      return contexto ? recibosSubsanacion.get(contexto.expediente_ref) ?? null : null;
    },
    tieneIntencionSubsanacionParaEstado: (estado) => intencionSubsanacionParaEstado(estado) !== null,
    invalidarSubsanacionPorDenegacion,
    obtenerReciboFiscalizacionConfirmado: () => reciboFiscalizacionConfirmado,
    obtenerReciboAsignacionConfirmado: () => reciboAsignacionConfirmado,
  });
}
