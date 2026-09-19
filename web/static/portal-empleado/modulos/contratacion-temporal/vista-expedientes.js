/** Vista y enlace DOM de la superficie de expedientes de contratación temporal. */

import { validarReciboAlta } from "./contrato.js";
import { montarFormularioCobertura } from "./formulario-cobertura.js";
import { montarFormularioResolucionFormalizacion } from "./formulario-resolucion-formalizacion.js";
import { montarFormularioAnotacionAdministrativa } from "./formulario-anotacion-administrativa.js";
import { montarFormularioCierreAdministrativo } from "./formulario-cierre-administrativo.js";
import { montarFormularioLlamamiento } from "./formulario-llamamiento.js";
import { montarVistaEstadisticas } from "./vista-estadisticas.js";
import { crearTraductorExpedientesContratacion } from "./i18n-expedientes.js";
import { crearTraductorContratacionTemporal } from "./i18n.js";
import { cerrarFase, mostrarFase } from "./fases-expediente.js";
import { prepararComposicionAnalisis } from "./vista-expedientes-analisis.js";
import {
  contextoLlamamientoDesdeEstado,
  renderizarModuloContratacionTemporal,
} from "./vista-expedientes-render.js";
import { montarModuloFiscalizacionContratacionTemporal } from "./vista-expedientes-fiscalizacion.js";
import { crearGestorDescargaBorradorRRHH } from "./vista-expedientes-borrador.js";
import { crearGestorIncorporacion } from "./vista-expedientes-incorporacion.js";
import { crearGestorTramitacion } from "./vista-expedientes-tramitacion.js";

export { renderizarModuloContratacionTemporal } from "./vista-expedientes-render.js";
export { montarModuloFiscalizacionContratacionTemporal } from "./vista-expedientes-fiscalizacion.js";

// Exportaciones auxiliares conservadas para compatibilidad con tests e importadores
export {
  montarFormularioCobertura,
  montarFormularioResolucionFormalizacion,
  montarFormularioAnotacionAdministrativa,
  montarFormularioCierreAdministrativo,
};

// Contenedores gestionados por el módulo: data-ct-exp-cobertura

export function crearEjecutorAltaConRefresco(
  ejecutor,
  presentador,
  alConfirmar = () => {},
) {
  if (typeof ejecutor !== "function" || typeof presentador?.cargar !== "function"
    || typeof alConfirmar !== "function") {
    throw new TypeError("dependencias del refresco de alta no válidas");
  }
  return async (comando, opciones) => {
    const recibo = validarReciboAlta(await ejecutor(comando, opciones));
    try {
      alConfirmar(recibo);
    } catch {
      // El recibo confirmado prevalece si no puede montarse el siguiente paso.
    }
    return recibo;
  };
}

function extraerDatosTarea(formulario) {
  if (!formulario) return {};
  const datos = new FormData(formulario);
  const resultado = {};
  for (const [campo, valor] of datos.entries()) {
    if (!Object.hasOwn(resultado, campo)) resultado[campo] = String(valor);
  }
  return resultado;
}

function enfocar(raiz, selector) {
  const elemento = raiz.querySelector(selector);
  elemento?.focus?.();
  elemento?.scrollIntoView?.({ block: "nearest", inline: "nearest" });
}

export async function montarModuloContratacionTemporal({
  raiz,
  presentador,
  alta = null,
  analisis = null,
  fiscalizacion = null,
  subsanacion = null,
  continuidad = null,
  llamamiento = null,
  clienteBorradorRRHH,
  entornoDescarga = globalThis,
  mensajes = {},
  anunciar = () => {},
  confirmarOperacion = () => false,
  locale = "es-ES",
  zonaHoraria = "Europe/Madrid",
} = {}) {
  if (!raiz || typeof raiz.addEventListener !== "function"
    || typeof raiz.querySelector !== "function"
    || typeof presentador?.obtenerEstado !== "function"
    || typeof presentador?.cargar !== "function"
    || typeof anunciar !== "function" || typeof confirmarOperacion !== "function") {
    throw new TypeError("dependencias del módulo de contratación temporal no válidas");
  }
  const traducirExpedientes = crearTraductorExpedientesContratacion(mensajes);

  const altaDisponible = alta !== null && typeof alta === "object"
    && typeof alta.ejecutor === "function" && alta.catalogos !== undefined;
  const composicionAnalisis = prepararComposicionAnalisis(analisis);
  const rectificacionDisponible = composicionAnalisis !== null
    && composicionAnalisis.rectificacion !== null
    && Array.isArray(composicionAnalisis.catalogos.motivos_rectificacion)
    && composicionAnalisis.catalogos.motivos_rectificacion.length > 0
    && typeof composicionAnalisis.rectificacion.metodoCliente === "function";
  const coberturaDisponible = composicionAnalisis !== null
    && typeof composicionAnalisis.cliente.proponerCobertura === "function"
    && typeof composicionAnalisis.cliente.decidirCobertura === "function"
    && typeof composicionAnalisis.cliente.consultarResultadoCobertura === "function";
  const asignacionDisponible = coberturaDisponible
    && typeof composicionAnalisis.cliente.asignarUnidad === "function";
  const informeJuridicoDisponible = asignacionDisponible
    && typeof composicionAnalisis.cliente.prepararInformeJuridico === "function"
    && typeof composicionAnalisis.cliente.consultarDetalleRRHH === "function";
  const clienteFiscalizacion = fiscalizacion !== null && typeof fiscalizacion === "object"
    && typeof fiscalizacion.cliente?.registrarResultadoFiscalizacion === "function"
    ? fiscalizacion.cliente : null;
  const fiscalizacionDisponible = clienteFiscalizacion !== null;
  const clienteSubsanacion = subsanacion !== null && typeof subsanacion === "object"
    && subsanacion.disponible === true
    && typeof subsanacion.cliente?.registrarSubsanacionReparos === "function"
    ? subsanacion.cliente : null;
  const subsanacionDisponible = clienteSubsanacion !== null;
  const clienteLlamamiento = continuidad?.cliente ?? llamamiento?.cliente ?? clienteFiscalizacion
    ?? composicionAnalisis?.cliente;
  const llamamientoDisponible = typeof clienteLlamamiento?.seleccionarLlamamiento === "function"
    && typeof clienteLlamamiento?.registrarComunicacionLlamamiento === "function";
  const resolucionFormalizacionDisponible = typeof clienteLlamamiento?.registrarResolucionFormalizacion === "function"
    && typeof clienteLlamamiento?.prepararResolucionFormalizacion === "function";
  const incorporacionEjercicioDisponible = typeof clienteLlamamiento?.prepararIncorporacionEjercicio === "function"
    && typeof clienteLlamamiento?.confirmarIncorporacionEjercicio === "function";

  let montada = true;
  let desmontarLlamamiento = null;
  let reciboPropuestaConfirmado = null;
  let desmontarEstadisticas = null;

  const esMontada = () => montada;

  const gestorBorrador = crearGestorDescargaBorradorRRHH({
    raiz,
    presentador,
    clienteBorradorRRHH,
    entornoDescarga,
    mensajes,
    anunciar,
    esMontada,
  });

  const gestorIncorporacion = crearGestorIncorporacion({
    raiz,
    presentador,
    clienteLlamamiento,
    resolucionFormalizacionDisponible,
    incorporacionEjercicioDisponible,
    confirmarOperacion,
    mensajes,
    locale,
    zonaHoraria,
    anunciar,
    repintar: (foco) => repintar(foco),
    entornoDescarga,
    esMontada,
  });

  const gestorTramitacion = crearGestorTramitacion({
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
    alFiscalizacionConfirmadaLlamamiento: (recibo) => {
      montarLlamamiento({
        expediente_ref: recibo.expediente_ref,
        version_esperada: recibo.version_resultante,
      });
    },
    confirmarOperacion,
    mensajes,
    locale,
    zonaHoraria,
    anunciar,
    repintar: (foco) => repintar(foco),
    esMontada,
  });

  async function refrescarDetalleTrasPropuesta(recibo, solicitud) {
    if (!montada || recibo?.propuesta_ref === undefined
      || reciboPropuestaConfirmado?.recibo !== recibo
      || solicitud?.expediente_ref !== reciboPropuestaConfirmado.solicitud.expediente_ref) return false;
    const expedienteRef = solicitud.expediente_ref;
    const panel = reciboPropuestaConfirmado.panel;
    if (panel === null || raiz.querySelector("[data-ct-exp-llamamiento]") !== panel) return false;
    const sigueSeleccionado = () => montada && panel !== null
      && raiz.querySelector("[data-ct-exp-llamamiento]") === panel;
    // cargar descarta la selección antigua al avanzar la versión. La identidad
    // del panel conserva la intención y cambia si el usuario navega.
    const avisarPendiente = () => {
      if (sigueSeleccionado()) anunciar(
        crearTraductorContratacionTemporal(mensajes)("llamamiento_propuesta_actualizacion_pendiente"), "aviso",
      );
    };
    try {
      await presentador.cargar();
      if (!sigueSeleccionado()) return false;
      const resumen = presentador.obtenerEstado().cuadro?.expedientes?.find(
        ({ expediente_ref: referencia }) => referencia === expedienteRef,
      );
      if (!resumen || resumen.version < recibo.version_resultante) { avisarPendiente(); return false; }
      await presentador.seleccionarExpediente(expedienteRef, "expediente");
      if (!sigueSeleccionado()) return false;
      const actualizado = presentador.obtenerEstado().expediente;
      if (actualizado?.expediente_ref !== expedienteRef
        || actualizado.version < recibo.version_resultante) { avisarPendiente(); return false; }
      repintar("#ct-exp-tarea-titulo");
      return true;
    } catch {
      avisarPendiente();
      return false;
    }
  }

  function montarLlamamiento(contexto = null) {
    if (!montada || !llamamientoDisponible) return;
    if (contexto === null) {
      desmontarLlamamiento?.();
      desmontarLlamamiento = null;
      return;
    }
    if (desmontarLlamamiento !== null) {
      desmontarLlamamiento.actualizarContexto(contexto);
      return;
    }
    const contenedor = raiz.querySelector("[data-ct-exp-llamamiento]");
    if (!contenedor) return;
    desmontarLlamamiento = montarFormularioLlamamiento({
      raiz: contenedor, cliente: clienteLlamamiento, contexto,
      confirmarOperacion, mensajes, locale, zonaHoraria, anunciar,
      alPropuestaConfirmada: (recibo, solicitud) => {
        reciboPropuestaConfirmado = Object.freeze({ recibo, solicitud, panel: contenedor });
      },
      alActualizarPropuesta: refrescarDetalleTrasPropuesta,
    });
  }

  function retirarEstadisticas() {
    desmontarEstadisticas?.();
    desmontarEstadisticas = null;
  }

  function montarEstadisticasSiProcede() {
    const estado = presentador.obtenerEstado();
    if (!montada || estado.vista !== "estadisticas") return;
    const contenedor = raiz.querySelector("[data-ct-exp-estadisticas]");
    if (!contenedor) return;
    retirarEstadisticas();
    const inst = montarVistaEstadisticas({ raiz: contenedor, anunciar });
    desmontarEstadisticas = inst.desmontar;
  }

  function repintar(selectorFoco = "") {
    if (!montada || gestorTramitacion.analisisEstableActivo()) return;
    gestorBorrador.cancelarDescargaInforme();
    retirarEstadisticas();
    desmontarLlamamiento?.();
    desmontarLlamamiento = null;
    gestorIncorporacion.retirar();
    gestorTramitacion.retirarComponentes();
    const estado = presentador.obtenerEstado();
    raiz.innerHTML = renderizarModuloContratacionTemporal(estado, {
      mensajes,
      locale,
      zonaHoraria,
      altaDisponible,
      analisisDisponible: composicionAnalisis !== null,
      coberturaDisponible,
      asignacionDisponible,
      informeJuridicoDisponible,
      fiscalizacionDisponible,
      subsanacionDisponible,
      reciboSubsanacionConfirmado: gestorTramitacion.obtenerReciboSubsanacionConfirmado(),
      reciboFiscalizacionConfirmado: gestorTramitacion.obtenerReciboFiscalizacionConfirmado(),
      reciboAsignacionConfirmado: gestorTramitacion.obtenerReciboAsignacionConfirmado(),
      llamamientoDisponible,
      resolucionFormalizacionDisponible,
      incorporacionEjercicioDisponible,
    });
    gestorTramitacion.montarAltaSiProcede();
    montarEstadisticasSiProcede();
    montarLlamamiento(contextoLlamamientoDesdeEstado(estado));
    gestorIncorporacion.montarResolucionFormalizacion().then(gestorIncorporacion.montarIncorporacionEjercicio);
    if (gestorTramitacion.montarAnalisisSiProcede() === false) {
      gestorTramitacion.retirarComponentes();
      raiz.innerHTML = renderizarModuloContratacionTemporal(estado, {
        mensajes,
        locale,
        zonaHoraria,
        altaDisponible,
        analisisDisponible: false,
        coberturaDisponible: false,
        asignacionDisponible: false,
        informeJuridicoDisponible: false,
        fiscalizacionDisponible: false,
      });
    } else {
      gestorTramitacion.montarCoberturaDesdeEstado();
      gestorTramitacion.montarAsignacionDesdeEstado();
      gestorTramitacion.montarInformeDesdeExpedienteActual();
      gestorTramitacion.montarFiscalizacionDesdeExpedienteActual();
      gestorTramitacion.montarSubsanacionDesdeExpedienteActual();
    }
    if (selectorFoco) enfocar(raiz, selectorFoco);
    if (estado.mensaje_clave) {
      anunciar(
        crearTraductorExpedientesContratacion(mensajes)(estado.mensaje_clave),
        estado.tipo_mensaje,
      );
    }
  }

  async function cambiarVista(vista) {
    if (gestorTramitacion.impedirCambioPorAnalisis()) return;
    const estado = presentador.obtenerEstado();
    if (estado.ocupado) {
      anunciar(crearTraductorExpedientesContratacion(mensajes)("estado_registrando_actuacion"), "aviso");
      return;
    }
    if (["expediente", "documentos", "auditoria"].includes(vista)) {
      const referencia = estado.expediente?.expediente_ref
        ?? estado.cuadro?.expedientes[0]?.expediente_ref;
      if (!referencia) return;
      try {
        const tarea = presentador.seleccionarExpediente(referencia, vista);
        repintar("[data-ct-exp-mensaje]");
        await tarea;
        repintar(vista === "expediente" ? "#ct-exp-tarea-titulo" : ".ct-exp-subcabecera h3");
      } catch {
        anunciar(
          crearTraductorExpedientesContratacion(mensajes)("estado_error_expediente"),
          "error",
        );
        repintar("[data-ct-exp-mensaje]");
      }
      return;
    }
    presentador.cambiarVista(vista);
    repintar(vista === "cuadro" ? "[data-ct-exp-filtros]" : (vista === "alta" ? "#ct-alta-titulo" : (vista === "estadisticas" ? '[data-ct-form="filtros-estadisticas"]' : ".ct-exp-contenido")));
  }

  async function manejarClick(evento) {
    const verFase = evento.target?.closest?.("[data-ct-exp-fase-ver]");
    if (verFase && raiz.contains(verFase)) {
      evento.preventDefault();
      mostrarFase(verFase, traducirExpedientes);
      return;
    }
    const cerrarFaseControl = evento.target?.closest?.("[data-ct-exp-fase-cerrar]");
    if (cerrarFaseControl && raiz.contains(cerrarFaseControl)) {
      evento.preventDefault();
      cerrarFase(cerrarFaseControl);
      return;
    }
    const controlVista = evento.target?.closest?.("[data-ct-exp-vista]");
    if (controlVista && raiz.contains(controlVista)) {
      evento.preventDefault();
      await cambiarVista(controlVista.dataset.ctExpVista);
      return;
    }
    const abrir = evento.target?.closest?.("[data-ct-exp-abrir]");
    if (abrir && raiz.contains(abrir)) {
      evento.preventDefault();
      if (gestorTramitacion.impedirCambioPorAnalisis()) return;
      try {
        const tarea = presentador.seleccionarExpediente(abrir.dataset.ctExpAbrir);
        repintar("[data-ct-exp-mensaje]");
        await tarea;
        repintar("#ct-exp-tarea-titulo");
      } catch {
        anunciar(
          crearTraductorExpedientesContratacion(mensajes)("estado_error_expediente"),
          "error",
        );
        repintar("[data-ct-exp-mensaje]");
      }
      return;
    }
    const pagina = evento.target?.closest?.("[data-ct-exp-pagina]");
    if (pagina && raiz.contains(pagina)) {
      evento.preventDefault();
      if (pagina.disabled || gestorTramitacion.impedirCambioPorAnalisis()) return;
      const promesa = presentador.navegarPagina(pagina.dataset.ctExpPagina);
      repintar("[data-ct-exp-mensaje]");
      await promesa;
      repintar("[data-ct-exp-filtros]");
      return;
    }
    const tareaControl = evento.target?.closest?.("[data-ct-exp-tarea]");
    if (tareaControl && raiz.contains(tareaControl)) {
      evento.preventDefault();
      if (gestorTramitacion.impedirCambioPorAnalisis() || presentador.obtenerEstado().ocupado) return;
      presentador.seleccionarTarea(tareaControl.dataset.ctExpTarea);
      repintar("#ct-exp-tarea-titulo");
      return;
    }
    const efecto = evento.target?.closest?.("[data-ct-exp-efecto]");
    if (efecto && raiz.contains(efecto)) {
      evento.preventDefault();
      if (gestorTramitacion.impedirCambioPorAnalisis()) return;
      const confirmacion = efecto.dataset.ctExpConfirmacion;
      const formulario = efecto.closest("[data-ct-exp-tarea-form]");
      if (typeof formulario?.checkValidity === "function" && !formulario.checkValidity()) {
        formulario.reportValidity?.();
        enfocar(raiz, "[data-ct-exp-tarea-form] :invalid");
        return;
      }
      if (confirmacion && confirmarOperacion({
        titulo: efecto.textContent.trim(),
        advertencia: confirmacion,
        referencia: presentador.obtenerEstado().expediente_ref,
      }) !== true) return;
      const promesa = presentador.ejecutarActuacion({
        accionRef: efecto.dataset.ctExpEfecto,
        datos: extraerDatosTarea(formulario),
      });
      repintar("[data-ct-exp-mensaje]");
      await promesa;
      repintar(presentador.obtenerEstado().recibo ? "[data-ct-exp-recibo]" : "[data-ct-exp-mensaje]");
      return;
    }
    const accion = evento.target?.closest?.("[data-ct-exp-accion]");
    if (!accion || !raiz.contains(accion)) return;
    evento.preventDefault();
    if (gestorTramitacion.impedirCambioPorAnalisis()) return;
    if (presentador.obtenerEstado().ocupado
      && accion.dataset.ctExpAccion !== "cancelar") return;
    if (accion.dataset.ctExpAccion === "volver-cuadro-actualizado") {
      presentador.cambiarVista("cuadro");
      const promesa = presentador.cargar();
      repintar("[data-ct-exp-mensaje]");
      await promesa;
      repintar(["[data-ct-exp-filtros]", ".ct-exp-estado-global"]);
    } else if (accion.dataset.ctExpAccion === "reintentar-resolucion") {
      await gestorIncorporacion.montarResolucionFormalizacion();
    } else if (accion.dataset.ctExpAccion === "reintentar-incorporacion") {
      await gestorIncorporacion.montarIncorporacionEjercicio();
    } else if (accion.dataset.ctExpAccion === "cancelar-descarga") {
      if (gestorBorrador.cancelarDescargaInforme()) gestorBorrador.informarDescarga("descarga_cancelada", "informacion");
    } else if (["descargar-informe-definitivo", "descargar-resolucion", "descargar-diligencia", "descargar-toma-posesion", "descargar-notificacion", "descargar-comunicacion-centro", "descargar-docx-informe-definitivo", "descargar-docx-resolucion", "descargar-docx-diligencia", "descargar-docx-toma-posesion", "descargar-docx-notificacion", "descargar-docx-comunicacion-centro", "reintentar-descarga-informe-definitivo", "reintentar-descarga-resolucion", "reintentar-descarga-diligencia", "reintentar-descarga-toma-posesion", "reintentar-descarga-notificacion", "reintentar-descarga-comunicacion-centro"].includes(accion.dataset.ctExpAccion)) {
      await gestorBorrador.descargarBorrador(accion);
    } else if (accion.dataset.ctExpAccion === "limpiar-filtros") {
      const promesa = presentador.cargar({ texto: "", estado: "", fase: "" });
      repintar("[data-ct-exp-mensaje]");
      await promesa;
      repintar("[data-ct-exp-filtros]");
    } else if (accion.dataset.ctExpAccion === "reintentar") {
      const promesa = presentador.cargar();
      repintar("[data-ct-exp-mensaje]");
      await promesa;
      repintar(".ct-exp-contenido");
    } else if (accion.dataset.ctExpAccion === "cancelar") {
      presentador.cancelar();
      repintar("[data-ct-exp-mensaje]");
    }
  }

  async function manejarEnvio(evento) {
    const formulario = evento.target?.closest?.("[data-ct-exp-filtros]");
    if (!formulario || !raiz.contains(formulario)) return;
    evento.preventDefault();
    if (gestorTramitacion.impedirCambioPorAnalisis() || presentador.obtenerEstado().ocupado) return;
    const datos = new FormData(formulario);
    try {
      const promesa = presentador.cargar({
        texto: String(datos.get("texto") ?? "").trim(),
        estado: String(datos.get("estado") ?? ""),
        fase: String(datos.get("fase") ?? ""),
      });
      repintar("[data-ct-exp-mensaje]");
      await promesa;
      repintar("[data-ct-exp-filtros]");
    } catch {
      const mensaje = crearTraductorExpedientesContratacion(mensajes)("estado_error_filtros");
      const destino = raiz.querySelector("[data-ct-exp-mensaje]");
      destino?.setAttribute?.("role", "alert");
      destino?.setAttribute?.("aria-live", "polite");
      if (destino) destino.textContent = mensaje;
      anunciar(mensaje, "error");
    }
  }

  raiz.addEventListener("click", manejarClick);
  raiz.addEventListener("submit", manejarEnvio);
  repintar();
  if (presentador.obtenerEstado().carga === "inicial") {
    const promesa = presentador.cargar();
    repintar("[data-ct-exp-mensaje]");
    await promesa;
    repintar(".ct-exp-contenido");
  }

  return Object.freeze({
    desmontar() {
      if (!montada) return;
      montada = false;
      gestorBorrador.cancelarDescargaInforme();
      retirarEstadisticas();
      desmontarLlamamiento?.();
      desmontarLlamamiento = null;
      gestorIncorporacion.retirar();
      gestorTramitacion.retirarComponentes();
      raiz.removeEventListener("click", manejarClick);
      raiz.removeEventListener("submit", manejarEnvio);
      presentador.desmontar?.();
    },
  });
}
