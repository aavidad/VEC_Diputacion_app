/**
 * Llamamiento del expediente abierto: una acción visible por paso; no se
 * guarda nada en el navegador. La clave de operación se genera sola al enviar
 * y nunca se muestra: un reintento de la misma operación reutiliza la petición
 * congelada (y su clave) para no duplicar el efecto.
 */
import { crearTraductorContratacionTemporal } from "./i18n.js?v=20261008-alta-rpt-circular-v6";
import { renderizarLlamamiento, reciboAntecedenteSiguiente } from "./renderizado-llamamiento.js?v=20261008-w-fichas-capacidades-v2";
import { mensajeValidacionPortal } from "../../portal-idioma.js?v=20261007-pantallas-textos-final-v1";
import { esValidacionRespuestaPendiente, cargarPublicacionesFormalizacionDesarrollo } from "./cliente-http-llamamiento.js";
import { crearPanelDocumentacionFormalizacion } from "./documentacion-formalizacion.js?v=20261008-w-fichas-capacidades-v2";
import { crearFuenteDocumentacionFormalizacionHTTP } from "./cliente-http-documentacion-formalizacion.js?v=20261007-pantallas-textos-final-v1";
import {
  CAMPOS_SELECCION, CAMPOS_COMUNICACION, referenciaLlamamientoValida,
  CAMPOS_COMUNICACION_SIGUIENTE, TIPO_ANTECEDENTE_CONTINUACION,
  validarSolicitudSeleccionLlamamiento, validarSolicitudComunicacionLlamamiento,
  validarReciboSeleccionLlamamiento, validarReciboComunicacionLlamamiento,
  CAMPOS_RESPUESTA_RECIBIDA, CAMPOS_RESPUESTA_EDITABLES,
  validarSolicitudRespuestaRecibida, validarReciboRespuestaRecibida,
  CAMPOS_RESOLUCION, validarSolicitudResolucionLlamamiento, validarReciboResolucionLlamamiento,
  CAMPOS_REVISION_RESOLUCION, CRITERIO_VALIDACION_RESOLUCION_DESARROLLO,
  RESPUESTAS_RESOLUCION,
  CAMPOS_SIGUIENTE, validarSolicitudContinuacionLlamamiento, validarReciboContinuacionLlamamiento,
  CAMPOS_PROPUESTA, validarSolicitudPropuestaFormalizacion, validarReciboPropuestaFormalizacion,
  CAMPOS_EVENTO_PLAZO, CAMPOS_EVENTO_PLAZO_EDITABLES, validarSolicitudEventoPlazo, validarReciboEventoPlazo,
  RESPUESTA_EXPIRACION,
} from "./contrato-llamamiento.js";
import { lecturaPlazoLlamamiento } from "./renderizado-plazo-llamamiento.js?v=20261008-w-fichas-capacidades-v2";
import { validarConsultaReciboRespuesta, validarReciboRespuestaConsultado } from "./cliente-http-consulta-recibo-respuesta.js";
import { LIMITE_TOTAL_COMUNICACIONES, instanteOrdenComunicacion, validarPaginaComunicacionesExpediente } from "./cliente-http-consulta-comunicaciones-expediente.js";

const OPERACION_COMUNICACION = Object.freeze({
  campos: CAMPOS_COMUNICACION, validar: validarSolicitudComunicacionLlamamiento,
  recibo: validarReciboComunicacionLlamamiento, metodo: "registrarComunicacionLlamamiento",
});
const OPERACION_RESPUESTA = Object.freeze({
  campos: CAMPOS_RESPUESTA_RECIBIDA, validar: validarSolicitudRespuestaRecibida,
  recibo: validarReciboRespuestaRecibida, metodo: "registrarRespuestaRecibida",
});
const OPERACION_RESOLUCION = Object.freeze({
  campos: CAMPOS_RESOLUCION, validar: validarSolicitudResolucionLlamamiento,
  recibo: validarReciboResolucionLlamamiento, metodo: "resolverLlamamiento",
});
// Plazo de respuesta: operaciones opcionales si el cliente no las ofrece.
const OPERACION_EVENTO_PLAZO = Object.freeze({
  campos: CAMPOS_EVENTO_PLAZO, validar: validarSolicitudEventoPlazo,
  recibo: validarReciboEventoPlazo, metodo: "registrarEventoPlazoLlamamiento",
});
const OPERACION_EXPIRACION = Object.freeze({ ...OPERACION_RESOLUCION });
const OPERACIONES = Object.freeze({
  seleccion: {
    campos: CAMPOS_SELECCION, validar: validarSolicitudSeleccionLlamamiento,
    recibo: validarReciboSeleccionLlamamiento, metodo: "seleccionarLlamamiento",
  },
  comunicacion: OPERACION_COMUNICACION,
  comunicacion_siguiente: { ...OPERACION_COMUNICACION, campos: CAMPOS_COMUNICACION_SIGUIENTE },
  respuesta: OPERACION_RESPUESTA,
  respuesta_siguiente: OPERACION_RESPUESTA,
  contacto: OPERACION_EVENTO_PLAZO,
  causa: OPERACION_EVENTO_PLAZO,
  expiracion: OPERACION_EXPIRACION,
  resolucion: OPERACION_RESOLUCION,
  resolucion_siguiente: OPERACION_RESOLUCION,
  siguiente: {
    campos: CAMPOS_SIGUIENTE, validar: validarSolicitudContinuacionLlamamiento,
    recibo: validarReciboContinuacionLlamamiento, metodo: "continuarLlamamiento",
  },
  propuesta: {
    campos: CAMPOS_PROPUESTA, validar: validarSolicitudPropuestaFormalizacion,
    recibo: validarReciboPropuestaFormalizacion, metodo: "prepararPropuestaFormalizacion",
  },
});
function esRespuesta(operacion) { return OPERACIONES[operacion] === OPERACION_RESPUESTA; }
function esResolucion(operacion) { return OPERACIONES[operacion] === OPERACION_RESOLUCION; }
function esEventoPlazo(operacion) { return OPERACIONES[operacion] === OPERACION_EVENTO_PLAZO; }
const OPERACIONES_OPCIONALES = Object.freeze(["propuesta", "contacto", "causa"]);
// El control muestra hora civil de Madrid; el contrato HTTP exige un instante UTC.
// Rechazamos la hora repetida del cambio de otoño en vez de elegir una sin avisar.
export function fechaRespuestaMadridUTC(valor) {
  if (typeof valor !== "string" || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}(?::\d{2})?$/u.test(valor)) throw new TypeError();
  const normalizada = valor.length === 16 ? `${valor}:00` : valor;
  const base = Date.parse(`${normalizada}Z`);
  if (!Number.isFinite(base) || new Date(base).toISOString().slice(0, 19) !== normalizada) throw new TypeError();
  const partes = new Intl.DateTimeFormat("en-GB", { timeZone: "Europe/Madrid", hourCycle: "h23",
    year: "numeric", month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit", second: "2-digit" });
  const coincidencias = [-2, -1, 0, 1, 2].map((horas) => new Date(base + horas * 3600000)).filter((instante) => {
    const p = Object.fromEntries(partes.formatToParts(instante).map(({ type, value }) => [type, value]));
    return `${p.year}-${p.month}-${p.day}T${p.hour}:${p.minute}:${p.second}` === normalizada;
  });
  if (coincidencias.length !== 1) throw new TypeError("hora_madrid_no_univoca");
  return coincidencias[0].toISOString().replace(/\.000Z$/u, "Z");
}
function nuevoPaso() {
  return { valores: {}, solicitud: null, recibo: null, ocupado: false, bloqueado: false,
    calculando: false, lecturaCorreo: 0,
    mensaje: "llamamiento_pendiente", tono: "informacion", controlador: null };
}
function nuevoPasoResolucion() {
  return { ...nuevoPaso(), mensaje: "llamamiento_resolucion_pendiente", claveConservada: false,
    valores: { revision_respuesta_rrhh: false, revision_plazo_rrhh: false,
      criterio_validacion_ref: CRITERIO_VALIDACION_RESOLUCION_DESARROLLO } };
}
export function montarFormularioLlamamiento({
  raiz, cliente, contexto = null, confirmarOperacion = () => false,
  generarClaveIdempotencia = () => globalThis.crypto?.randomUUID?.(),
  criptografia = globalThis.crypto,
  fetchPublicaciones = globalThis.fetch,
  mensajes = {}, locale = "es-ES", zonaHoraria = "Europe/Madrid", anunciar = () => {},
  alPropuestaConfirmada = () => {},
  alActualizarPropuesta = null,
  fuenteDocumentacionFormalizacion = null,
  reloj = () => Date.now(),
} = {}) {
  if (!raiz || typeof raiz.addEventListener !== "function"
    || typeof raiz.removeEventListener !== "function" || typeof raiz.querySelector !== "function"
    || typeof raiz.contains !== "function" || typeof raiz.replaceChildren !== "function"
    || Object.entries(OPERACIONES).some(([operacion, { metodo }]) => !OPERACIONES_OPCIONALES.includes(operacion) && typeof cliente?.[metodo] !== "function")
    || typeof reloj !== "function"
    || typeof confirmarOperacion !== "function" || typeof generarClaveIdempotencia !== "function"
    || typeof anunciar !== "function" || typeof alPropuestaConfirmada !== "function"
    || (alActualizarPropuesta !== null && typeof alActualizarPropuesta !== "function")) {
    throw new TypeError("dependencias del formulario de llamamiento no válidas");
  }
  const t = crearTraductorContratacionTemporal(mensajes);
  const fecha = new Intl.DateTimeFormat(locale, {
    dateStyle: "medium", timeStyle: "medium", timeZone: zonaHoraria,
  });
  let montado = true;
  // La fuente HTTP solo se crea si hay fetch; sin ella el panel no se pinta.
  const fuenteDocumentacion = fuenteDocumentacionFormalizacion
    ?? (typeof globalThis.fetch === "function" ? crearFuenteDocumentacionFormalizacionHTTP() : null);
  const crearDocumentacion = () => fuenteDocumentacion ? crearPanelDocumentacionFormalizacion({
    fuente: fuenteDocumentacion, t, locale, zonaHoraria, criptografia, generarClaveIdempotencia, anunciar,
  }) : null;
  let documentacion = crearDocumentacion();
  const estado = { seleccion: nuevoPaso(), comunicacion: nuevoPaso(), respuesta: nuevoPaso(),
    comunicacion_siguiente: { ...nuevoPaso(), claveConservada: false },
    respuesta_siguiente: { ...nuevoPaso(), claveConservada: false },
    propuesta: { ...nuevoPaso(), aceptacion: null, disponible: false, claveConservada: false,
      actualizando: false, actualizacionPendiente: false, mensaje: "llamamiento_propuesta_no_disponible" },
    siguiente: { ...nuevoPaso(), mensaje: "llamamiento_siguiente_pendiente", claveConservada: false },
    resolucion: nuevoPasoResolucion(), resolucion_siguiente: nuevoPasoResolucion(),
    contacto: { ...nuevoPaso(), mensaje: "llamamiento_contacto_pendiente" },
    causa: { ...nuevoPaso(), mensaje: "llamamiento_causa_pendiente" },
    expiracion: { ...nuevoPaso(), mensaje: "llamamiento_expiracion_pendiente",
      valores: { revision_respuesta_rrhh: false, revision_plazo_rrhh: false } },
    plazoDisponible: typeof cliente.registrarEventoPlazoLlamamiento === "function",
    enlazado: false, comunicacionAbierta: false,
    consultaRespuesta: { estado: "sin_contexto", referencias: null, recibo: null,
      controlador: null, generacion: 0, mensaje: "llamamiento_consulta_sin_contexto", tono: "informacion" },
    comunicaciones: { estado: "sin_contexto", filas: [], seleccionada: null, intentoNoConfirmado: null, controlador: null,
      generacion: 0, mensaje: "llamamiento_comunicaciones_sin_contexto", tono: "informacion" } };
  function ahora() {
    try { const valor = reloj(); return Number.isFinite(valor) ? valor : Date.now(); } catch { return Date.now(); }
  }
  function lecturaPlazo() { return lecturaPlazoLlamamiento(estado, ahora()); }
  // La vista pasa a «vencido» y muestra la propuesta al llegar el vencimiento.
  let temporizadorPlazo = null;
  function programarVencimiento() {
    clearTimeout(temporizadorPlazo);
    const restante = Date.parse(estado.contacto.recibo?.plazo?.respuesta_hasta ?? "") - ahora();
    if (Number.isFinite(restante) && restante > 0 && restante < 2 ** 31 - 1) {
      temporizadorPlazo = setTimeout(repintarVencimiento, restante + 1000);
    }
  }
  // El contacto efectivo abre el plazo; sin él, no hay expiración ni causa.
  function puedeRegistrarEventoPlazo(operacion) {
    if (!estado.plazoDisponible || estado.comunicacion.recibo?.version_resultante !== 2 || estado.resolucion.recibo) return false;
    if (operacion === "contacto") return true;
    return lecturaPlazo()?.exigeCausa === true;
  }
  function puedeConfirmarExpiracion() {
    return lecturaPlazo()?.propuestaExpiracion === true || estado.expiracion.solicitud !== null;
  }

  function puedeDeclarar(operacion) {
    if (estado.comunicaciones.intentoNoConfirmado) return false;
    const comunicado = estado[operacion === "respuesta_siguiente" ? "comunicacion_siguiente" : "comunicacion"];
    const recibo = comunicado.recibo;
    const coincideRecibo = estado.consultaRespuesta.estado === "ausente"
      && estado.consultaRespuesta.referencias?.organizacion_ref === comunicado.solicitud?.organizacion_ref
      && estado.consultaRespuesta.referencias?.expediente_ref === comunicado.solicitud?.expediente_ref
      && estado.consultaRespuesta.referencias?.comunicacion_ref === recibo?.comunicacion_ref
      && recibo?.version_resultante === 2 && (operacion !== "respuesta_siguiente"
        || ["registrada_localmente", "replay_registrada_localmente"].includes(recibo.estado_local));
    if (coincideRecibo) return true;
    if (estado.comunicaciones.estado === "lista") {
      const fila = estado.comunicaciones.seleccionada;
      return fila?.estado_respuesta === "sin_respuesta"
        && estado.consultaRespuesta.estado === "ausente"
        && fila.organizacion_ref === estado.consultaRespuesta.referencias?.organizacion_ref
        && fila.expediente_ref === estado.consultaRespuesta.referencias?.expediente_ref
        && fila.comunicacion_ref === estado.consultaRespuesta.referencias?.comunicacion_ref
        && fila.expediente_ref === estado.seleccion.valores.expediente_ref
        && fila.version === 2
        && operacion === (fila.antecedente_tipo === "continuacion_confirmada" ? "respuesta_siguiente" : "respuesta");
    }
    return false;
  }
  function antecedenteConsultaDisponible() {
    const consulta = estado.consultaRespuesta.referencias;
    return [estado.comunicacion, estado.comunicacion_siguiente].some((paso) =>
      paso.recibo?.comunicacion_ref === consulta?.comunicacion_ref
      && paso.solicitud?.organizacion_ref === consulta?.organizacion_ref
      && paso.solicitud?.expediente_ref === consulta?.expediente_ref)
      || (estado.comunicaciones.estado === "lista"
        && estado.comunicaciones.seleccionada?.comunicacion_ref === consulta?.comunicacion_ref
        && estado.comunicaciones.seleccionada?.organizacion_ref === consulta?.organizacion_ref
        && estado.comunicaciones.seleccionada?.expediente_ref === consulta?.expediente_ref);
  }
  function consultaDesdeRecibo(recibo, solicitud) {
    return { organizacion_ref: solicitud.organizacion_ref, expediente_ref: solicitud.expediente_ref,
      comunicacion_ref: recibo.comunicacion_ref };
  }
  function consultarReciboRespuesta(referencias, forzar = false) {
    if (!montado) return;
    let consulta;
    try { consulta = validarConsultaReciboRespuesta(referencias); } catch { return; }
    const paso = estado.consultaRespuesta;
    const misma = paso.referencias?.organizacion_ref === consulta.organizacion_ref
      && paso.referencias?.expediente_ref === consulta.expediente_ref
      && paso.referencias?.comunicacion_ref === consulta.comunicacion_ref;
    if (!forzar && misma && ["cargando", "ausente", "confirmado"].includes(paso.estado)) return;
    paso.controlador?.abort();
    const generacion = ++paso.generacion;
    const controlador = new AbortController();
    paso.controlador = controlador;
    paso.referencias = consulta;
    paso.recibo = null;
    paso.estado = "cargando";
    paso.mensaje = "llamamiento_consulta_cargando";
    paso.tono = "informacion";
    repintar();
    return Promise.resolve().then(() => {
      if (!montado || generacion !== paso.generacion || controlador.signal.aborted) return null;
      if (typeof cliente.consultarReciboRespuesta !== "function") throw new TypeError("consulta no compuesta");
      return cliente.consultarReciboRespuesta(consulta, { signal: controlador.signal });
    }).then((respuesta) => {
      if (!montado || generacion !== paso.generacion || controlador.signal.aborted) return;
      paso.recibo = validarReciboRespuestaConsultado(respuesta, consulta);
      paso.estado = "confirmado";
      paso.mensaje = "llamamiento_consulta_confirmada";
      paso.tono = "exito";
      repintar("consultaRespuesta");
    }).catch((error) => {
      if (!montado || generacion !== paso.generacion || controlador.signal.aborted) return;
      const conocido = error?.envelopeValido === true;
      paso.estado = conocido && error.estado === 404 && error.codigo === "recurso_no_encontrado"
        ? "ausente" : conocido && [401, 403].includes(error.estado)
          && ["autenticacion_requerida", "acceso_denegado"].includes(error.codigo)
          ? "denegado" : "error";
      paso.mensaje = `llamamiento_consulta_${paso.estado}`;
      paso.tono = paso.estado === "denegado" ? "error" : "aviso";
      repintar("consultaRespuesta");
    }).finally(() => {
      if (generacion === paso.generacion) paso.controlador = null;
    });
  }
  function cargarComunicaciones(expedienteRef, forzar = false) {
    if (!montado || typeof cliente.consultarComunicacionesExpediente !== "function") return;
    const paso = estado.comunicaciones;
    if (!forzar && paso.expedienteRef === expedienteRef && ["cargando", "lista", "vacia"].includes(paso.estado)) return;
    paso.controlador?.abort();
    estado.consultaRespuesta.controlador?.abort();
    estado.consultaRespuesta.generacion += 1;
    estado.consultaRespuesta.estado = "sin_contexto";
    estado.consultaRespuesta.referencias = null;
    estado.consultaRespuesta.recibo = null;
    const generacion = ++paso.generacion;
    const controlador = new AbortController();
    paso.controlador = controlador;
    paso.expedienteRef = expedienteRef;
    paso.filas = [];
    paso.seleccionada = null;
    paso.estado = "cargando";
    paso.mensaje = "llamamiento_comunicaciones_cargando";
    paso.tono = "informacion";
    repintar();
    return (async () => {
      const filas = [], cursores = new Set();
      let cursor, organizacion, anterior;
      for (;;) {
        const consulta = { expediente_ref: expedienteRef, ...(cursor ? { cursor } : {}) };
        const pagina = validarPaginaComunicacionesExpediente(
          await cliente.consultarComunicacionesExpediente(consulta, { signal: controlador.signal }), consulta);
        if (!montado || controlador.signal.aborted || generacion !== paso.generacion) return;
        for (const fila of pagina.comunicaciones) {
          if ((organizacion && fila.organizacion_ref !== organizacion)
            || (anterior && (instanteOrdenComunicacion(fila.registrada_en) < instanteOrdenComunicacion(anterior.registrada_en)
              || (instanteOrdenComunicacion(fila.registrada_en) === instanteOrdenComunicacion(anterior.registrada_en)
                && fila.comunicacion_ref <= anterior.comunicacion_ref)))) {
            throw new TypeError("paginación de comunicaciones incoherente");
          }
          organizacion = fila.organizacion_ref;
          anterior = fila;
          filas.push(fila);
          if (filas.length > LIMITE_TOTAL_COMUNICACIONES) throw new TypeError("demasiadas comunicaciones");
        }
        if (!pagina.siguiente_cursor) break;
        if (filas.length >= LIMITE_TOTAL_COMUNICACIONES || cursores.has(pagina.siguiente_cursor)) {
          throw new TypeError("paginación de comunicaciones incompleta");
        }
        cursores.add(pagina.siguiente_cursor);
        cursor = pagina.siguiente_cursor;
      }
      if (!montado || controlador.signal.aborted || generacion !== paso.generacion) return;
      paso.filas = Object.freeze(filas);
      paso.estado = filas.length ? "lista" : "vacia";
      paso.mensaje = filas.length ? "llamamiento_comunicaciones_lista" : "llamamiento_comunicaciones_vacia";
      paso.tono = "informacion";
      repintar("comunicaciones");
    })().catch((error) => {
      if (!montado || controlador.signal.aborted || generacion !== paso.generacion) return;
      paso.filas = [];
      paso.seleccionada = null;
      paso.estado = error?.envelopeValido === true && [401, 403].includes(error.estado)
        ? "denegado" : "error";
      paso.mensaje = `llamamiento_comunicaciones_${paso.estado}`;
      paso.tono = paso.estado === "denegado" ? "error" : "aviso";
      repintar("comunicaciones");
    }).finally(() => {
      if (generacion === paso.generacion) paso.controlador = null;
    });
  }
  function seleccionarComunicacion(indice) {
    const lista = estado.comunicaciones;
    if (lista.estado !== "lista" || !Number.isSafeInteger(indice) || indice < 0 || indice >= lista.filas.length) return;
    const fila = lista.filas[indice];
    if (fila.expediente_ref !== estado.seleccion.valores.expediente_ref) return;
    if (Object.keys(OPERACIONES).some((operacion) => {
      const paso = estado[operacion];
      return paso.ocupado || paso.calculando || paso.actualizando
        || (paso.solicitud !== null && paso.recibo === null
          && paso.solicitud !== lista.intentoNoConfirmado?.solicitud);
    })) return;
    if (lista.seleccionada?.comunicacion_ref === fila.comunicacion_ref) return;
    const expedienteRef = estado.seleccion.valores.expediente_ref;
    const versionEsperada = estado.seleccion.valores.version_esperada;
    clearTimeout(temporizadorPlazo);
    documentacion?.desmontar();
    documentacion = crearDocumentacion();
    estado.seleccion = { ...nuevoPaso(), valores: { expediente_ref: expedienteRef, version_esperada: versionEsperada } };
    estado.comunicacion = nuevoPaso();
    estado.respuesta = nuevoPaso();
    estado.contacto = { ...nuevoPaso(), mensaje: "llamamiento_contacto_pendiente" };
    estado.causa = { ...nuevoPaso(), mensaje: "llamamiento_causa_pendiente" };
    estado.expiracion = { ...nuevoPaso(), mensaje: "llamamiento_expiracion_pendiente",
      valores: { revision_respuesta_rrhh: false, revision_plazo_rrhh: false } };
    estado.siguiente = { ...nuevoPaso(), mensaje: "llamamiento_siguiente_pendiente", claveConservada: false };
    estado.comunicacion_siguiente = { ...nuevoPaso(), claveConservada: false };
    estado.respuesta_siguiente = { ...nuevoPaso(), claveConservada: false };
    estado.resolucion = nuevoPasoResolucion();
    estado.resolucion_siguiente = nuevoPasoResolucion();
    estado.propuesta = { ...nuevoPaso(), aceptacion: null, disponible: false, claveConservada: false,
      actualizando: false, actualizacionPendiente: false, mensaje: "llamamiento_propuesta_no_disponible" };
    estado.comunicacionAbierta = false;
    lista.seleccionada = fila;
    const operacion = fila.antecedente_tipo === "continuacion_confirmada" ? "respuesta_siguiente" : "respuesta";
    estado[operacion].valores = { ...estado[operacion].valores,
      organizacion_ref: fila.organizacion_ref, expediente_ref: fila.expediente_ref,
      llamamiento_ref: fila.llamamiento_ref, comunicacion_ref: fila.comunicacion_ref,
      version_comunicacion_esperada: fila.version };
    consultarReciboRespuesta({ organizacion_ref: fila.organizacion_ref,
      expediente_ref: fila.expediente_ref,
      comunicacion_ref: fila.comunicacion_ref });
  }
  function puedeResolver(operacion) {
    const lectura = operacion === "resolucion" ? lecturaPlazo() : null;
    // La regla capturada al abrir el plazo decide sobre la respuesta tardía.
    if (lectura?.noAdmitida || (lectura?.exigeCausa && !estado.causa.recibo)) return false;
    return RESPUESTAS_RESOLUCION.includes(
      estado[operacion === "resolucion_siguiente" ? "respuesta_siguiente" : "respuesta"].recibo?.respuesta,
    );
  }
  function repintar(operacion = "") {
    if (!montado) return;
    limpiarValidacion();
    raiz.innerHTML = renderizarLlamamiento(estado, t, fecha, ahora());
    documentacion?.pintar(raiz.querySelector("[data-ct-documentacion-formalizacion]"), {
      aceptadaEn: estado.propuesta.aceptacion?.respuesta === "aceptacion" ? estado.propuesta.aceptacion.resuelta_en : "",
      expedienteRef: estado.propuesta.valores.expediente_ref,
    });
    if (operacion) {
      const paso = estado[operacion];
      const foco = raiz.querySelector(paso.recibo
        ? `[data-ct-llamamiento-recibo="${operacion}"]`
        : `[data-ct-llamamiento-estado="${operacion}"]`);
      foco?.focus?.();
      foco?.scrollIntoView?.({ block: "nearest" });
      try { anunciar(t(paso.mensaje), paso.tono); } catch { /* La región viva permanece. */ }
    }
  }
  function repintarVencimiento() {
    if (!montado) return;
    const documento = raiz.ownerDocument;
    const activo = documento?.activeElement;
    const dentro = activo && raiz.contains(activo);
    const formulario = dentro ? activo.closest?.("[data-ct-llamamiento-form]") : null;
    const operacion = formulario?.dataset.ctLlamamientoForm;
    const indice = formulario ? Array.from(formulario.elements).indexOf(activo) : -1;
    const nombreControl = dentro ? activo.name : "";
    const tipoControl = dentro ? activo.type : "";
    const id = dentro ? activo.id : "";
    const ancla = dentro ? ["recibo", "estado"].map((tipo) => {
      const paso = activo.getAttribute?.(`data-ct-llamamiento-${tipo}`);
      return Object.hasOwn(estado, paso) ? `[data-ct-llamamiento-${tipo}="${paso}"]` : "";
    }).find(Boolean) : "";
    const seleccion = dentro && Number.isInteger(activo.selectionStart)
      ? [activo.selectionStart, activo.selectionEnd, activo.selectionDirection] : null;
    // Solo el repintado pasivo recoge la edición actual; los envíos conservan
    // su propio borrado y su foco de resultado.
    guardarBorradores();
    repintar();
    if (!dentro) return;
    const control = id ? documento.getElementById(id) : ancla ? raiz.querySelector(ancla)
      : Object.hasOwn(OPERACIONES, operacion) && indice >= 0
        ? raiz.querySelector(`[data-ct-llamamiento-form="${operacion}"]`)?.elements[indice] : null;
    if (control && raiz.contains(control) && control.name === nombreControl
      && control.type === tipoControl && !control.matches(":disabled")) {
      control.focus({ preventScroll: true });
      if (documento.activeElement === control) {
        if (seleccion) control.setSelectionRange?.(...seleccion);
        control.scrollIntoView?.({ block: "nearest" });
        return;
      }
    }
    const titulo = raiz.querySelector("#ct-llamamiento-plazo-titulo");
    titulo?.setAttribute("tabindex", "-1");
    titulo?.focus({ preventScroll: true });
    titulo?.scrollIntoView({ block: "nearest" });
  }
  // La validez nativa orienta los campos; el contrato sigue decidiendo la semántica.
  const marcasValidacion = new Map();
  const firmasResumen = new WeakMap();
  function quitarMarca(control) {
    const marca = marcasValidacion.get(control);
    if (!marca) return;
    if (control.getAttribute("aria-invalid") === "true") {
      if (marca.anterior === null) control.removeAttribute("aria-invalid");
      else control.setAttribute("aria-invalid", marca.anterior);
    }
    const descripciones = (control.getAttribute("aria-describedby") ?? "").split(/\s+/u)
      .filter((id) => id && id !== marca.id);
    if (descripciones.length) control.setAttribute("aria-describedby", descripciones.join(" "));
    else control.removeAttribute("aria-describedby");
    marcasValidacion.delete(control);
  }
  function limpiarValidacion() {
    for (const control of marcasValidacion.keys()) quitarMarca(control);
  }
  function mostrarErroresNativos(formulario, editado = null) {
    const controles = Array.from(formulario.elements ?? []);
    const errores = new Map();
    for (const control of controles) {
      const revisar = !editado || control === editado || marcasValidacion.has(control)
        || (control.type === "radio" && control.name === editado.name);
      if (!revisar) continue;
      quitarMarca(control);
      if (control.willValidate !== true || control.validity.valid) continue;
      const id = `ct-llamamiento-${formulario.dataset.ctLlamamientoForm}-${control.name}-error`;
      const mensaje = raiz.ownerDocument?.getElementById(id);
      if (!mensaje || !formulario.contains(mensaje)) continue;
      marcasValidacion.set(control, { id, anterior: control.getAttribute("aria-invalid") });
      control.setAttribute("aria-invalid", "true");
      const descripciones = (control.getAttribute("aria-describedby") ?? "").split(/\s+/u).filter(Boolean);
      control.setAttribute("aria-describedby", [...new Set([...descripciones, id])].join(" "));
      if (!errores.has(id)) errores.set(id, { control, mensaje, texto: mensajeValidacionPortal(control) });
    }
    formulario.querySelectorAll?.("[data-ct-llamamiento-error-campo]").forEach((mensaje) => {
      mensaje.hidden = !errores.has(mensaje.id);
      mensaje.textContent = errores.get(mensaje.id)?.texto ?? "";
    });
    const resumen = formulario.querySelector?.("[data-ct-llamamiento-errores]");
    const lista = resumen?.querySelector("ul");
    if (lista) {
      const enlaces = [...errores.values()].map(({ control, texto }) => {
        const rotulo = control.type === "radio" ? control.closest("fieldset")?.querySelector("legend") : control.labels?.[0];
        return { control, texto: `${rotulo?.textContent.trim() ?? control.name}: ${texto}` };
      });
      const firma = JSON.stringify(enlaces.map(({ control, texto }) => [control.id, texto]));
      // Conservar los enlaces si no cambia el resumen: salir del campo no cancela un clic.
      if (firmasResumen.get(lista) !== firma) {
        lista.replaceChildren();
        for (const { control, texto } of enlaces) {
          const enlace = raiz.ownerDocument.createElement("a");
          enlace.href = `#${control.id}`;
          enlace.dataset.ctLlamamientoErrorEnlace = control.id;
          enlace.textContent = texto;
          const fila = raiz.ownerDocument.createElement("li");
          fila.append(enlace);
          lista.append(fila);
        }
        firmasResumen.set(lista, firma);
      }
      resumen.hidden = errores.size === 0;
    }
    return controles.find((control) => control.willValidate === true && !control.validity.valid);
  }
  function alEditarCampo(evento) {
    const control = evento.target;
    const formulario = control?.closest?.("[data-ct-llamamiento-form]");
    if (!formulario || !raiz.contains(formulario) || control.willValidate !== true
      || estado[formulario.dataset.ctLlamamientoForm]?.solicitud) return;
    // El envío valida antes de actuar; no desplazar su botón entre pulsación y clic.
    if (evento.type === "focusout" && evento.relatedTarget?.closest?.("button, a")) return;
    // Al editar sólo se actualizan errores ya mostrados; al salir se valida el campo.
    if (evento.type === "focusout" || marcasValidacion.has(control)) mostrarErroresNativos(formulario, control);
  }
  function guardarBorradores() {
    for (const [operacion, contrato] of Object.entries(OPERACIONES)) {
      const paso = estado[operacion];
      const formulario = raiz.querySelector(`[data-ct-llamamiento-form="${operacion}"]`);
      if (!formulario?.elements || paso.solicitud !== null) continue;
      for (const campo of contrato.campos) {
        // Los antecedentes de comunicación proceden del recibo, no de los controles.
        const revision = (esResolucion(operacion) || operacion === "expiracion") && CAMPOS_REVISION_RESOLUCION.includes(campo);
        // La clave vive solo en el estado y las referencias del expediente vienen del contexto.
        if (campo === "clave_idempotencia") continue;
        if ((esResolucion(operacion) || ["seleccion", "comunicacion", "comunicacion_siguiente", "siguiente", "propuesta", "expiracion"].includes(operacion)) && !revision) continue;
        if (esRespuesta(operacion) && !CAMPOS_RESPUESTA_EDITABLES.includes(campo)) continue;
        if (esEventoPlazo(operacion) && !CAMPOS_EVENTO_PLAZO_EDITABLES.includes(campo)) continue;
        const control = formulario.elements.namedItem(campo);
        paso.valores[campo] = revision ? control?.checked === true : String(control?.value ?? "");
      }
    }
    estado.comunicacionAbierta = raiz.querySelector(
      "[data-ct-llamamiento-comunicacion]",
    )?.open === true || estado.comunicacionAbierta;
  }
  async function alCambiarArchivo(evento) {
    const control = evento.target?.closest?.("[data-ct-llamamiento-correo]");
    const operacion = control?.closest?.("[data-ct-llamamiento-form]")?.dataset.ctLlamamientoForm;
    if (!control || !raiz.contains(control) || !esRespuesta(operacion) || !puedeDeclarar(operacion)) return;
    const paso = estado[operacion];
    if (paso.solicitud || paso.ocupado || paso.bloqueado) return;
    guardarBorradores();
    const lectura = ++paso.lecturaCorreo;
    const archivo = control.files?.length === 1 ? control.files[0] : null;
    paso.valores.correo_sha256 = "";
    paso.calculando = true;
    paso.mensaje = "llamamiento_correo_calculando";
    paso.tono = "informacion";
    repintar(operacion);
    let bytes;
    try {
      // Se limita antes de leer. Solo la huella llega al estado y al POST;
      // el nombre y contenido del correo nunca se proyectan ni se guardan.
      if (!archivo || !/\.eml$/iu.test(archivo.name) || archivo.size < 1
        || archivo.size > 2 * 1024 * 1024 || !criptografia?.subtle?.digest) throw new TypeError();
      bytes = new Uint8Array(await archivo.arrayBuffer());
      if (!montado || lectura !== paso.lecturaCorreo) return;
      if (bytes.byteLength !== archivo.size) throw new TypeError();
      const digest = new Uint8Array(await criptografia.subtle.digest("SHA-256", bytes));
      if (!montado || lectura !== paso.lecturaCorreo) return;
      const huella = Array.from(digest, (byte) => byte.toString(16).padStart(2, "0")).join("");
      if (digest.length !== 32 || huella === "0".repeat(64)) throw new TypeError();
      paso.valores.correo_sha256 = huella;
      paso.mensaje = "llamamiento_correo_calculado";
    } catch {
      if (!montado || lectura !== paso.lecturaCorreo) return;
      paso.mensaje = "llamamiento_correo_error";
      paso.tono = "error";
    } finally {
      bytes?.fill(0);
      if (montado && lectura === paso.lecturaCorreo) {
        paso.calculando = false;
        repintar(operacion);
      }
    }
  }
  function actualizarContexto(nuevo) {
    if (montado && estado.enlazado
      && estado.seleccion.valores.expediente_ref !== nuevo?.expediente_ref) {
      desmontar();
      return false;
    }
    if (!montado || estado.seleccion.solicitud !== null
      || !referenciaLlamamientoValida(nuevo?.expediente_ref)
      || !Number.isSafeInteger(nuevo?.version_esperada) || nuevo.version_esperada < 1) return false;
    guardarBorradores();
    estado.seleccion.valores.expediente_ref = nuevo.expediente_ref;
    estado.seleccion.valores.version_esperada = nuevo.version_esperada;
    if (estado.comunicacion.solicitud === null) {
      estado.comunicacion.valores.expediente_ref = nuevo.expediente_ref;
    }
    estado.enlazado = true;
    if (typeof cliente.consultarComunicacionesExpediente === "function") cargarComunicaciones(nuevo.expediente_ref);
    else if (nuevo.consulta_respuesta !== undefined) consultarReciboRespuesta(nuevo.consulta_respuesta);
    else repintar();
    return true;
  }
  // Antecedente del siguiente llamamiento: renuncia resuelta o expiración
  // confirmada por RRHH, ambas con la intención de siguiente pendiente.
  function puedeContinuar() {
    return reciboAntecedenteSiguiente(estado)?.intencion_siguiente?.estado_local === "pendiente";
  }
  function versionFiscalizadaPermitePropuesta() {
    const version = estado.seleccion.solicitud?.version_esperada
      ?? estado.seleccion.valores.version_esperada;
    return Number.isSafeInteger(version) && version >= 6 && version < Number.MAX_SAFE_INTEGER;
  }
  function puedeProponer() {
    return estado.propuesta.aceptacion?.respuesta === "aceptacion"
      && versionFiscalizadaPermitePropuesta();
  }
  async function prepararPropuesta(origen) {
    const paso = estado.propuesta;
    if (paso.solicitud !== null || paso.recibo || paso.calculando) return;
    if (!paso.aceptacion) {
      const s = origen?.solicitud, r = origen?.recibo;
      if (!s || !r) return;
      // Recibo ya validado de la resolución que confirmó la aceptación; no se
      // sustituye al cargar publicaciones, enviar o recuperar esta propuesta.
      paso.aceptacion = r;
      paso.valores = { expediente_ref: s.expediente_ref, llamamiento_ref: s.llamamiento_ref,
        resolucion_llamamiento_aceptada_ref: r.resolucion_ref, recibo_resolucion_aceptada_ref: r.recibo_local_ref,
        version_esperada: estado.seleccion.solicitud?.version_esperada
          ?? estado.seleccion.valores.version_esperada, anexos: Object.freeze([]) };
    }
    if (typeof cliente.prepararPropuestaFormalizacion !== "function") return;
    paso.calculando = true;
    paso.mensaje = "llamamiento_propuesta_cargando";
    paso.controlador = new AbortController();
    repintar();
    try {
      const publicaciones = await cargarPublicacionesFormalizacionDesarrollo({
        fetchImpl: fetchPublicaciones, criptografia, signal: paso.controlador.signal,
      });
      if (!montado) return;
      paso.valores = { ...paso.valores, ...publicaciones };
      paso.disponible = true;
      paso.mensaje = "llamamiento_propuesta_pendiente";
    } catch {
      paso.mensaje = "llamamiento_propuesta_no_disponible";
      paso.tono = "aviso";
    } finally {
      paso.calculando = false;
      paso.controlador = null;
      repintar();
    }
  }
  async function alEnviar(evento) {
    const formulario = evento.target?.closest?.("[data-ct-llamamiento-form]");
    if (!formulario || !raiz.contains(formulario)) return;
    evento.preventDefault();
    const operacion = formulario.dataset.ctLlamamientoForm;
    if (!Object.hasOwn(OPERACIONES, operacion)) return;
    const paso = estado[operacion];
    // Sin expediente abierto no hay llamamiento: no se teclean referencias ni versiones.
    if (!estado.enlazado) return;
    if (paso.ocupado || paso.calculando || paso.recibo || paso.bloqueado) return;
    if (["cargando", "error", "denegado"].includes(estado.comunicaciones.estado)) return;
    if (estado.comunicaciones.estado === "lista"
      && (!estado.comunicaciones.seleccionada || !["respuesta", "respuesta_siguiente", "resolucion", "resolucion_siguiente",
        "siguiente", "comunicacion_siguiente", "propuesta"].includes(operacion))) return;
    if (estado.consultaRespuesta.referencias !== null
      && (estado.consultaRespuesta.estado !== "ausente" || !antecedenteConsultaDisponible())) return;
    if (operacion === "comunicacion" && estado.seleccion.recibo === null) return;
    if (operacion === "comunicacion_siguiente" && estado.siguiente.recibo === null) return;
    if (esRespuesta(operacion) && !puedeDeclarar(operacion)) return;
    if (esResolucion(operacion) && !puedeResolver(operacion)) return;
    if (esEventoPlazo(operacion) && !puedeRegistrarEventoPlazo(operacion) && paso.solicitud === null) return;
    if (operacion === "expiracion" && !puedeConfirmarExpiracion()) return;
    if (operacion === "siguiente" && !puedeContinuar()) return;
    if (operacion === "propuesta" && (!puedeProponer() || !paso.disponible)) return;
    guardarBorradores();
    if (paso.solicitud === null) {
      const primero = mostrarErroresNativos(formulario);
      // Consultar validity evita que invalid global deje mensajes propios en otros radios.
      if (primero) {
        primero?.focus?.();
        primero?.scrollIntoView?.({ block: "nearest" });
        return;
      }
    }
    const contrato = OPERACIONES[operacion];
    // Un reintento reenvía la petición congelada; su clave nunca se sustituye.
    const reintento = paso.solicitud !== null;
    // Primera vez: clave nueva e invisible. Si ya hay una (reintento tras un error
    // de validación o una confirmación cancelada), se conserva la misma.
    if (paso.solicitud === null && contrato.campos.includes("clave_idempotencia") && !paso.valores.clave_idempotencia) {
      try {
        const clave = generarClaveIdempotencia(operacion);
        if (typeof clave !== "string" || !clave) throw new TypeError();
        paso.valores.clave_idempotencia = clave;
      } catch {
        paso.mensaje = "llamamiento_preparacion_error";
        paso.tono = "error";
        repintar(operacion);
        return;
      }
    }
    const recuperandoRespuesta = (esResolucion(operacion) || ["respuesta", "respuesta_siguiente", "siguiente", "comunicacion_siguiente", "propuesta", "contacto", "causa", "expiracion"].includes(operacion)) && paso.solicitud !== null;
    let solicitud;
    try {
      solicitud = paso.solicitud ?? contrato.validar(Object.fromEntries(
        contrato.campos.map((campo) => [
          campo, campo === "version_esperada" || campo === "version_comunicacion_esperada"
            ? Number(paso.valores[campo])
            : campo === "recibida_en" && esRespuesta(operacion) && !paso.valores[campo]?.endsWith("Z")
              ? fechaRespuestaMadridUTC(paso.valores[campo])
              : campo === "instante_en" && !paso.valores[campo]?.endsWith("Z")
              ? `${paso.valores[campo]}${paso.valores[campo]?.length === 16 ? ":00" : ""}Z`
              : paso.valores[campo],
        ]),
      ));
      if (solicitud.clave_idempotencia && ["comunicacion_siguiente", "resolucion_siguiente", "propuesta", "contacto", "causa", "expiracion"].includes(operacion) && Object.keys(OPERACIONES).some(
        (anterior) => anterior !== operacion
          && (estado[anterior].solicitud?.clave_idempotencia ?? estado[anterior].recibo?.clave_idempotencia)
            === solicitud.clave_idempotencia,
      )) throw new TypeError("clave_repetida");
      if (["resolucion", "siguiente"].includes(operacion) && [estado.seleccion, estado.comunicacion,
        estado.respuesta, ...(operacion !== "resolucion" ? [estado.resolucion, estado.expiracion] : [])]
        .some((anterior) => (anterior.solicitud?.clave_idempotencia ?? anterior.recibo?.clave_idempotencia)
          === solicitud.clave_idempotencia)) {
        throw new TypeError("clave_repetida");
      }
    } catch (error) {
      // Una clave generada que coincide con la de otra operación se descarta: el
      // siguiente intento genera otra, sin que la persona tenga que hacer nada.
      if (error?.message === "clave_repetida") delete paso.valores.clave_idempotencia;
      paso.mensaje = esRespuesta(operacion) && error?.message === "hora_madrid_no_univoca"
        ? "llamamiento_respuesta_hora_no_univoca"
        : ["contacto", "causa", "expiracion"].includes(operacion) ? `llamamiento_${operacion}_validacion`
        : operacion === "respuesta_siguiente" ? "llamamiento_respuesta_siguiente_validacion"
        : operacion === "comunicacion_siguiente" ? "llamamiento_comunicacion_siguiente_validacion"
        : operacion === "propuesta" ? "llamamiento_propuesta_validacion" : operacion === "siguiente" ? "llamamiento_siguiente_validacion"
        : esResolucion(operacion) ? "llamamiento_resolucion_validacion"
        : operacion === "respuesta" ? "llamamiento_respuesta_validacion" : "llamamiento_validacion";
      paso.tono = "error";
      repintar(operacion);
      return;
    }
    let confirmado = false;
    try {
      confirmado = confirmarOperacion({
        titulo: t("llamamiento_" + operacion),
        advertencia: t("llamamiento_confirmacion_" + (esRespuesta(operacion) ? "respuesta" : esResolucion(operacion) ? "resolucion" : operacion), {
          version: solicitud.version_esperada,
          justificante: solicitud.prueba_respuesta_ref,
          criterio: solicitud.criterio_validacion_ref,
          antecedente: solicitud.prueba_entrega_ref,
          ...(esEventoPlazo(operacion) ? { instante: fecha.format(new Date(solicitud.instante_en)), prueba: solicitud.prueba_ref } : {}),
          ...(operacion === "siguiente" ? {
            resolucion: solicitud.resolucion_ref, intencion: solicitud.intencion_ref,
          } : {}),
          ...(operacion === "propuesta" ? {
            llamamiento: solicitud.llamamiento_ref, resolucion: solicitud.resolucion_llamamiento_aceptada_ref,
          } : {}),
          ...(esResolucion(operacion) ? { respuesta: t("llamamiento_resolucion_" + solicitud.respuesta) } : {}),
          ...(esRespuesta(operacion) ? {
            respuesta: t("llamamiento_respuesta_" + solicitud.respuesta),
            correo: solicitud.correo_ref, huella: solicitud.correo_sha256,
            recibida: fecha.format(new Date(solicitud.recibida_en)),
          } : {}),
        }),
        referencia: solicitud.expediente_ref,
        datos: Object.freeze({ ...solicitud }),
      }) === true;
    } catch { /* La confirmación es obligatoria. */ }
    if (!confirmado || !montado) return;
    paso.solicitud = solicitud;
    if (["siguiente", "comunicacion_siguiente", "respuesta_siguiente", "resolucion_siguiente", "propuesta", "contacto", "causa", "expiracion"].includes(operacion)) paso.claveConservada = true;
    paso.valores = { ...solicitud };
    paso.ocupado = true;
    paso.controlador = new AbortController();
    paso.mensaje = esResolucion(operacion) ? "llamamiento_solicitando_resolucion" : "llamamiento_enviando";
    paso.tono = "informacion";
    repintar(operacion);
    let respuestaRecibida = false;
    try {
      const respuesta = await cliente[contrato.metodo](solicitud, {
        signal: paso.controlador.signal,
      });
      respuestaRecibida = true;
      const antecedente = estado.resolucion.recibo?.respuesta === "renuncia" ? estado.resolucion : estado.expiracion;
      const recibo = contrato.recibo(respuesta, solicitud, operacion === "siguiente"
        ? antecedente.solicitud.llamamiento_ref : operacion === "propuesta" ? estado.propuesta.aceptacion.resuelta_en : undefined);
      if (!montado) return;
      guardarBorradores();
      paso.recibo = recibo;
      if (operacion === "propuesta") {
        paso.actualizacionPendiente = alActualizarPropuesta !== null;
        try { alPropuestaConfirmada(recibo, solicitud); } catch { /* El recibo ya prevalece. */ }
      }
      paso.mensaje = ["contacto", "causa", "expiracion"].includes(operacion) ? `llamamiento_${operacion}_recibo`
        : operacion === "respuesta_siguiente" ? "llamamiento_respuesta_siguiente_recibo"
        : operacion === "comunicacion_siguiente" ? "llamamiento_comunicacion_siguiente_recibo"
        : operacion === "propuesta" ? "llamamiento_propuesta_recibo" : operacion === "siguiente" ? "llamamiento_siguiente_recibo"
        : esResolucion(operacion) ? "llamamiento_" + operacion + "_recibo_" + recibo.respuesta
        : operacion === "respuesta" ? "llamamiento_respuesta_recibo" : "llamamiento_recibo";
      paso.tono = "exito";
      if (operacion === "seleccion") {
        estado.comunicacionAbierta = true;
        if (estado.comunicacion.solicitud === null) {
          estado.comunicacion.valores = {
            ...estado.comunicacion.valores,
            organizacion_ref: recibo.organizacion_ref,
            expediente_ref: solicitud.expediente_ref,
            llamamiento_ref: recibo.llamamiento_ref,
            version_esperada: recibo.version_llamamiento,
            prueba_entrega_ref: recibo.recibo_ref,
          };
        }
      }
      if (operacion === "comunicacion" && recibo.version_resultante === 2) {
        const contexto = {
          organizacion_ref: solicitud.organizacion_ref, expediente_ref: solicitud.expediente_ref,
          llamamiento_ref: solicitud.llamamiento_ref, comunicacion_ref: recibo.comunicacion_ref,
        };
        for (const [evento, tipo] of [["contacto", "contacto_efectivo"], ["causa", "causa_justificada"]]) {
          estado[evento].valores = { ...estado[evento].valores, ...contexto,
            version_comunicacion_esperada: recibo.version_resultante, tipo };
        }
        estado.expiracion.valores = { ...estado.expiracion.valores, ...contexto,
          version_esperada: recibo.version_resultante, respuesta: RESPUESTA_EXPIRACION, prueba_respuesta_ref: "" };
      }
      if (operacion === "contacto") {
        programarVencimiento();
        // Los criterios los fija el servidor con el catálogo vigente al abrir el plazo.
        estado.expiracion.valores = { ...estado.expiracion.valores,
          criterio_validacion_ref: recibo.plazo.criterio_expiracion_ref };
        if (estado.resolucion.solicitud === null) {
          estado.resolucion.valores = { ...estado.resolucion.valores,
            criterio_validacion_ref: recibo.plazo.criterio_respuesta_ref };
        }
      }
      if (["comunicacion", "comunicacion_siguiente"].includes(operacion) && recibo.version_resultante === 2) {
        const destino = estado[operacion === "comunicacion" ? "respuesta" : "respuesta_siguiente"];
        destino.valores = {
          ...destino.valores,
          organizacion_ref: solicitud.organizacion_ref,
          expediente_ref: solicitud.expediente_ref,
          llamamiento_ref: solicitud.llamamiento_ref,
          comunicacion_ref: recibo.comunicacion_ref,
          version_comunicacion_esperada: recibo.version_resultante,
        };
        await consultarReciboRespuesta(consultaDesdeRecibo(recibo, solicitud));
      }
      if (esRespuesta(operacion) && RESPUESTAS_RESOLUCION.includes(recibo.respuesta)) {
        const destino = estado[operacion === "respuesta" ? "resolucion" : "resolucion_siguiente"];
        destino.valores = {
          ...destino.valores,
          organizacion_ref: recibo.organizacion_ref, expediente_ref: recibo.expediente_ref,
          llamamiento_ref: recibo.llamamiento_ref, comunicacion_ref: recibo.comunicacion_ref,
          version_esperada: recibo.version_comunicacion_esperada,
          respuesta: recibo.respuesta, prueba_respuesta_ref: recibo.justificante_ref,
        };
      }
      if (["resolucion", "expiracion"].includes(operacion) && puedeContinuar()) {
        estado.siguiente.valores = { ...estado.siguiente.valores,
          organizacion_ref: solicitud.organizacion_ref, expediente_ref: solicitud.expediente_ref,
          resolucion_ref: recibo.resolucion_ref, intencion_ref: recibo.intencion_siguiente.referencia };
      }
      if (esResolucion(operacion) && recibo.respuesta === "aceptacion"
        && versionFiscalizadaPermitePropuesta()) await prepararPropuesta(paso);
      if (operacion === "siguiente" && estado.comunicacion_siguiente.solicitud === null) {
        estado.comunicacion_siguiente.valores = {
          ...estado.comunicacion_siguiente.valores,
          organizacion_ref: recibo.organizacion_ref, expediente_ref: recibo.expediente_ref,
          llamamiento_ref: recibo.llamamiento_ref, version_esperada: recibo.version_llamamiento,
          prueba_entrega_ref: recibo.recibo_ref, tipo_antecedente: TIPO_ANTECEDENTE_CONTINUACION,
        };
      }
      // El sucesor conserva su propio estado; no rearma recibos ni continuación anteriores.
    } catch (error) {
      if (!montado) return;
      guardarBorradores();
      const conflictoContenido = esRespuesta(operacion) && error?.codigo === "contenido_respuesta_en_conflicto";
      const conflicto = conflictoContenido || ["conflicto_no_reintentable", "clave_idempotencia_reutilizada",
        "version_en_conflicto", "seleccion_no_disponible", "resolucion_no_aceptada", "evento_en_conflicto"].includes(error?.codigo);
      const validacionPendiente = esResolucion(operacion) && esValidacionRespuestaPendiente(error)
        && error.resultadoIndeterminado === false && !recuperandoRespuesta && !respuestaRecibida;
      // Denegar un replay no demuestra ausencia de efecto del intento original.
      const rechazo = !validacionPendiente && !recuperandoRespuesta && !respuestaRecibida && error?.resultadoIndeterminado === false
        && error?.envelopeValido === true;
      paso.bloqueado = conflicto;
      paso.mensaje = conflictoContenido ? "llamamiento_contenido_respuesta_en_conflicto"
        : validacionPendiente ? "llamamiento_validacion_respuesta_pendiente"
        : conflicto ? "llamamiento_conflicto"
        : rechazo ? "llamamiento_rechazada"
        : esRespuesta(operacion) ? "llamamiento_respuesta_resultado_incierto" : "llamamiento_error";
      paso.tono = conflicto || rechazo ? "error" : "aviso";
      // Solo un rechazo conocido de este primer intento permite corregir las
      // casillas. Un intento anterior ambiguo nunca se libera por un replay.
      if (validacionPendiente) {
        paso.solicitud = null;
        paso.claveConservada = true;
      }
      if (rechazo && !conflicto) {
        paso.solicitud = null;
        // El servidor rechazó el primer intento sin efecto: el siguiente, corregido, lleva
        // clave nueva. Tras un intento incierto la clave se conserva, porque el original
        // pudo surtir efecto; y también en las operaciones que la guardan para recuperar.
        if (!reintento && !paso.claveConservada) delete paso.valores.clave_idempotencia;
      }
      if (esRespuesta(operacion) && estado.comunicaciones.estado === "lista"
        && error?.envelopeValido === true && [403, 409].includes(error.estado)) {
        estado.comunicaciones.intentoNoConfirmado = Object.freeze({ operacion, solicitud });
        paso.bloqueado = true;
        estado.comunicaciones.filas = [];
        estado.comunicaciones.seleccionada = null;
        estado.comunicaciones.estado = "error";
        estado.comunicaciones.mensaje = "llamamiento_comunicaciones_revisar_tras_rechazo";
        estado.comunicaciones.tono = "aviso";
      }
    } finally {
      paso.ocupado = false;
      paso.controlador = null;
      repintar(operacion);
    }
  }
  function alPulsar(evento) {
    const enlaceError = evento.target?.closest?.("[data-ct-llamamiento-error-enlace]");
    if (typeof enlaceError?.dataset?.ctLlamamientoErrorEnlace === "string" && raiz.contains(enlaceError)) {
      evento.preventDefault();
      const controlError = raiz.ownerDocument?.getElementById(enlaceError.dataset.ctLlamamientoErrorEnlace);
      if (controlError && raiz.contains(controlError)) {
        controlError.focus();
        controlError.scrollIntoView?.({ block: "nearest" });
      }
      return;
    }
    const reintentarComunicaciones = evento.target?.closest?.("[data-ct-comunicaciones-reintentar]");
    if (reintentarComunicaciones?.dataset?.ctComunicacionesReintentar !== undefined
      && raiz.contains(reintentarComunicaciones)) {
      evento.preventDefault();
      if (estado.comunicaciones.estado === "error") {
        cargarComunicaciones(estado.comunicaciones.expedienteRef, true);
      }
      return;
    }
    const elegirComunicacion = evento.target?.closest?.("[data-ct-comunicacion-indice]");
    if (elegirComunicacion?.dataset?.ctComunicacionIndice !== undefined
      && raiz.contains(elegirComunicacion)) {
      evento.preventDefault();
      seleccionarComunicacion(Number(elegirComunicacion.dataset.ctComunicacionIndice));
      return;
    }
    const reintentarConsulta = evento.target?.closest?.("[data-ct-llamamiento-reintentar-consulta]");
    if (reintentarConsulta?.dataset?.ctLlamamientoReintentarConsulta !== undefined
      && raiz.contains(reintentarConsulta)) {
      evento.preventDefault();
      if (estado.consultaRespuesta.estado === "error") {
        consultarReciboRespuesta(estado.consultaRespuesta.referencias, true);
      }
      return;
    }
    const revisarPropuesta = evento.target?.closest?.("[data-ct-propuesta-formalizacion-siguiente]");
    if (revisarPropuesta?.dataset?.ctPropuestaFormalizacionSiguiente !== undefined
      && raiz.contains(revisarPropuesta)) {
      // El hash pertenece al router del portal; esta acción permanece en el formulario.
      evento.preventDefault();
      const titulo = raiz.querySelector("#ct-llamamiento-propuesta-titulo");
      titulo?.setAttribute?.("tabindex", "-1");
      titulo?.scrollIntoView?.({ block: "center" });
      titulo?.focus?.({ preventScroll: true });
      return;
    }
    const reintentarPublicaciones = evento.target?.closest?.("[data-ct-llamamiento-reintentar-publicaciones]");
    if (reintentarPublicaciones?.dataset?.ctLlamamientoReintentarPublicaciones !== undefined
      && raiz.contains(reintentarPublicaciones)) {
      evento.preventDefault();
      if (!estado.propuesta.aceptacion || estado.propuesta.disponible) return;
      return prepararPropuesta();
    }
    const actualizarPropuesta = evento.target?.closest?.("[data-ct-llamamiento-actualizar-propuesta]");
    if (actualizarPropuesta?.dataset?.ctLlamamientoActualizarPropuesta !== undefined
      && raiz.contains(actualizarPropuesta)) {
      evento.preventDefault();
      const paso = estado.propuesta;
      if (!paso.recibo || !paso.actualizacionPendiente || paso.actualizando) return;
      paso.actualizando = true;
      paso.mensaje = "llamamiento_propuesta_actualizando";
      paso.tono = "informacion";
      repintar("propuesta");
      return Promise.resolve().then(() => alActualizarPropuesta(paso.recibo, paso.solicitud)).then((actualizado) => {
        if (!montado) return;
        paso.actualizando = false;
        if (actualizado === true) return;
        paso.mensaje = "llamamiento_propuesta_actualizacion_pendiente";
        paso.tono = "aviso";
        repintar("propuesta");
      }).catch(() => {
        if (!montado) return;
        paso.actualizando = false;
        paso.mensaje = "llamamiento_propuesta_actualizacion_pendiente";
        paso.tono = "aviso";
        repintar("propuesta");
      });
    }
  }

  raiz.addEventListener("submit", alEnviar);
  raiz.addEventListener("click", alPulsar);
  raiz.addEventListener("change", alCambiarArchivo);
  raiz.addEventListener("input", alEditarCampo);
  raiz.addEventListener("focusout", alEditarCampo);
  if (!actualizarContexto(contexto)) repintar();
  const desmontar = () => {
    if (!montado) return;
    montado = false;
    documentacion?.desmontar();
    clearTimeout(temporizadorPlazo);
    for (const operacion of Object.keys(OPERACIONES)) {
      estado[operacion].lecturaCorreo += 1;
      estado[operacion].controlador?.abort();
    }
    estado.consultaRespuesta.generacion += 1;
    estado.consultaRespuesta.controlador?.abort();
    estado.comunicaciones.generacion += 1;
    estado.comunicaciones.controlador?.abort();
    raiz.removeEventListener("submit", alEnviar);
    raiz.removeEventListener("click", alPulsar);
    raiz.removeEventListener("change", alCambiarArchivo);
    raiz.removeEventListener("input", alEditarCampo);
    raiz.removeEventListener("focusout", alEditarCampo);
    limpiarValidacion();
    raiz.replaceChildren();
  };
  desmontar.actualizarContexto = actualizarContexto;
  desmontar.revisarPlazo = repintarVencimiento;
  return desmontar;
}
