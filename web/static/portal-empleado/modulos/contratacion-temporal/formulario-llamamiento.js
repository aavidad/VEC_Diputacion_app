/** Una intención visible por acción; no se guarda nada en el navegador. */
import { crearTraductorContratacionTemporal } from "./i18n.js";
import { renderizarLlamamiento } from "./renderizado-llamamiento.js";
import { esValidacionRespuestaPendiente, cargarPublicacionesFormalizacionDesarrollo } from "./cliente-http-llamamiento.js";
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
} from "./contrato-llamamiento.js";

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
const OPERACIONES = Object.freeze({
  seleccion: {
    campos: CAMPOS_SELECCION, validar: validarSolicitudSeleccionLlamamiento,
    recibo: validarReciboSeleccionLlamamiento, metodo: "seleccionarLlamamiento",
  },
  comunicacion: OPERACION_COMUNICACION,
  comunicacion_siguiente: { ...OPERACION_COMUNICACION, campos: CAMPOS_COMUNICACION_SIGUIENTE },
  respuesta: OPERACION_RESPUESTA,
  respuesta_siguiente: OPERACION_RESPUESTA,
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
} = {}) {
  if (!raiz || typeof raiz.addEventListener !== "function"
    || typeof raiz.removeEventListener !== "function" || typeof raiz.querySelector !== "function"
    || typeof raiz.contains !== "function" || typeof raiz.replaceChildren !== "function"
    || Object.entries(OPERACIONES).some(([operacion, { metodo }]) => operacion !== "propuesta" && typeof cliente?.[metodo] !== "function")
    || typeof confirmarOperacion !== "function" || typeof generarClaveIdempotencia !== "function"
    || typeof anunciar !== "function") {
    throw new TypeError("dependencias del formulario de llamamiento no válidas");
  }
  const t = crearTraductorContratacionTemporal(mensajes);
  const fecha = new Intl.DateTimeFormat(locale, {
    dateStyle: "medium", timeStyle: "medium", timeZone: zonaHoraria,
  });
  let montado = true;
  const estado = { seleccion: nuevoPaso(), comunicacion: nuevoPaso(), respuesta: nuevoPaso(),
    comunicacion_siguiente: { ...nuevoPaso(), claveConservada: false },
    respuesta_siguiente: { ...nuevoPaso(), claveConservada: false },
    propuesta: { ...nuevoPaso(), aceptacion: null, disponible: false, claveConservada: false, mensaje: "llamamiento_propuesta_no_disponible" },
    siguiente: { ...nuevoPaso(), mensaje: "llamamiento_siguiente_pendiente", claveConservada: false },
    resolucion: nuevoPasoResolucion(), resolucion_siguiente: nuevoPasoResolucion(),
    enlazado: false, comunicacionAbierta: false };

  function puedeDeclarar(operacion) {
    const recibo = estado[operacion === "respuesta_siguiente" ? "comunicacion_siguiente" : "comunicacion"].recibo;
    return recibo?.version_resultante === 2 && (operacion !== "respuesta_siguiente"
      || ["registrada_localmente", "replay_registrada_localmente"].includes(recibo.estado_local));
  }
  function puedeResolver(operacion) {
    return RESPUESTAS_RESOLUCION.includes(
      estado[operacion === "resolucion_siguiente" ? "respuesta_siguiente" : "respuesta"].recibo?.respuesta,
    );
  }
  function repintar(operacion = "") {
    if (!montado) return;
    raiz.innerHTML = renderizarLlamamiento(estado, t, fecha);
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
  function guardarBorradores() {
    for (const [operacion, contrato] of Object.entries(OPERACIONES)) {
      const paso = estado[operacion];
      const formulario = raiz.querySelector(`[data-ct-llamamiento-form="${operacion}"]`);
      if (!formulario?.elements || paso.solicitud !== null) continue;
      for (const campo of contrato.campos) {
        // Los antecedentes de comunicación proceden del recibo, no de los controles.
        const revision = esResolucion(operacion) && CAMPOS_REVISION_RESOLUCION.includes(campo);
        if ((esResolucion(operacion) || ["comunicacion", "comunicacion_siguiente", "siguiente", "propuesta"].includes(operacion)) && campo !== "clave_idempotencia" && !revision) continue;
        if (paso.claveConservada && campo === "clave_idempotencia") continue;
        if (esRespuesta(operacion) && !CAMPOS_RESPUESTA_EDITABLES.includes(campo)) continue;
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
    repintar();
    return true;
  }
  function puedeContinuar() {
    return estado.resolucion.recibo?.respuesta === "renuncia"
      && estado.resolucion.recibo.intencion_siguiente?.estado_local === "pendiente";
  }
  function puedeProponer() {
    return estado.propuesta.aceptacion?.respuesta === "aceptacion"
      && estado.seleccion.solicitud?.version_esperada === 6;
  }
  async function prepararPropuesta(aceptacion) {
    const paso = estado.propuesta;
    if (paso.aceptacion || paso.solicitud !== null || paso.recibo || paso.calculando) return;
    const s = aceptacion.solicitud, r = aceptacion.recibo;
    // Recibo ya validado de la resolución que confirmó la aceptación; no se
    // sustituye al cargar publicaciones, enviar o recuperar esta propuesta.
    paso.aceptacion = r;
    paso.valores = { expediente_ref: s.expediente_ref, llamamiento_ref: s.llamamiento_ref,
      resolucion_llamamiento_aceptada_ref: r.resolucion_ref, recibo_resolucion_aceptada_ref: r.recibo_local_ref,
      version_esperada: estado.seleccion.solicitud.version_esperada, anexos: Object.freeze([]) };
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
    if (paso.ocupado || paso.calculando || paso.recibo || paso.bloqueado) return;
    if (operacion === "comunicacion" && estado.seleccion.recibo === null) return;
    if (operacion === "comunicacion_siguiente" && estado.siguiente.recibo === null) return;
    if (esRespuesta(operacion) && !puedeDeclarar(operacion)) return;
    if (esResolucion(operacion) && !puedeResolver(operacion)) return;
    if (operacion === "siguiente" && !puedeContinuar()) return;
    if (operacion === "propuesta" && (!puedeProponer() || !paso.disponible)) return;
    guardarBorradores();
    const contrato = OPERACIONES[operacion];
    const recuperandoRespuesta = (esResolucion(operacion) || ["respuesta", "respuesta_siguiente", "siguiente", "comunicacion_siguiente", "propuesta"].includes(operacion)) && paso.solicitud !== null;
    let solicitud;
    try {
      solicitud = paso.solicitud ?? contrato.validar(Object.fromEntries(
        contrato.campos.map((campo) => [
          campo, campo === "version_esperada" || campo === "version_comunicacion_esperada"
            ? Number(paso.valores[campo])
            : campo === "recibida_en" && !paso.valores[campo]?.endsWith("Z")
              ? `${paso.valores[campo]}${paso.valores[campo]?.length === 16 ? ":00" : ""}Z`
              : paso.valores[campo],
        ]),
      ));
      if (["comunicacion_siguiente", "respuesta_siguiente", "resolucion_siguiente", "propuesta"].includes(operacion) && Object.keys(OPERACIONES).some(
        (anterior) => anterior !== operacion
          && estado[anterior].solicitud?.clave_idempotencia === solicitud.clave_idempotencia,
      )) throw new TypeError("la operación necesita su propia clave");
      if (["resolucion", "siguiente"].includes(operacion) && [estado.seleccion, estado.comunicacion,
        estado.respuesta, ...(operacion !== "resolucion" ? [estado.resolucion] : [])]
        .some((anterior) => anterior.solicitud?.clave_idempotencia === solicitud.clave_idempotencia)) {
        throw new TypeError("la operación necesita su propia clave");
      }
    } catch {
      paso.mensaje = operacion === "respuesta_siguiente" ? "llamamiento_respuesta_siguiente_validacion"
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
            recibida: solicitud.recibida_en,
          } : {}),
        }),
        referencia: solicitud.expediente_ref,
        datos: Object.freeze({ ...solicitud }),
      }) === true;
    } catch { /* La confirmación es obligatoria. */ }
    if (!confirmado || !montado) return;
    paso.solicitud = solicitud;
    if (["siguiente", "comunicacion_siguiente", "respuesta_siguiente", "resolucion_siguiente", "propuesta"].includes(operacion)) paso.claveConservada = true;
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
      const recibo = contrato.recibo(respuesta, solicitud, operacion === "siguiente"
        ? estado.resolucion.solicitud.llamamiento_ref : operacion === "propuesta" ? estado.propuesta.aceptacion.resuelta_en : undefined);
      if (!montado) return;
      guardarBorradores();
      paso.recibo = recibo;
      paso.mensaje = operacion === "respuesta_siguiente" ? "llamamiento_respuesta_siguiente_recibo"
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
      if (operacion === "resolucion" && puedeContinuar()) {
        estado.siguiente.valores = { ...estado.siguiente.valores,
          organizacion_ref: solicitud.organizacion_ref, expediente_ref: solicitud.expediente_ref,
          resolucion_ref: recibo.resolucion_ref, intencion_ref: recibo.intencion_siguiente.referencia };
      }
      if (esResolucion(operacion) && recibo.respuesta === "aceptacion"
        && estado.seleccion.solicitud?.version_esperada === 6) await prepararPropuesta(paso);
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
      const conflicto = ["conflicto_no_reintentable", "clave_idempotencia_reutilizada",
        "version_en_conflicto", "seleccion_no_disponible", "resolucion_no_aceptada"].includes(error?.codigo);
      const validacionPendiente = esResolucion(operacion) && esValidacionRespuestaPendiente(error)
        && error.resultadoIndeterminado === false && !recuperandoRespuesta && !respuestaRecibida;
      // Denegar un replay no demuestra ausencia de efecto del intento original.
      const rechazo = !validacionPendiente && !recuperandoRespuesta && !respuestaRecibida && error?.resultadoIndeterminado === false
        && error?.envelopeValido === true;
      paso.bloqueado = conflicto;
      paso.mensaje = validacionPendiente ? "llamamiento_validacion_respuesta_pendiente"
        : conflicto ? "llamamiento_conflicto"
        : rechazo ? "llamamiento_rechazada" : "llamamiento_error";
      paso.tono = conflicto || rechazo ? "error" : "aviso";
      // Solo un rechazo conocido de este primer intento permite corregir las
      // casillas. Un intento anterior ambiguo nunca se libera por un replay.
      if (validacionPendiente) {
        paso.solicitud = null;
        paso.claveConservada = true;
      }
      if (rechazo && !conflicto) paso.solicitud = null;
    } finally {
      paso.ocupado = false;
      paso.controlador = null;
      repintar(operacion);
    }
  }
  function alPulsar(evento) {
    const control = evento.target?.closest?.("[data-ct-llamamiento-clave]");
    if (!control || !raiz.contains(control)) return;
    evento.preventDefault();
    const operacion = control.dataset.ctLlamamientoClave;
    if (!Object.hasOwn(OPERACIONES, operacion)) return;
    if (operacion === "comunicacion_siguiente" && estado.siguiente.recibo === null) return;
    if (esRespuesta(operacion) && !puedeDeclarar(operacion)) return;
    if (esResolucion(operacion) && !puedeResolver(operacion)) return;
    if (operacion === "siguiente" && !puedeContinuar()) return;
    if (operacion === "propuesta" && (!puedeProponer() || !estado.propuesta.disponible)) return;
    const paso = estado[operacion];
    if (paso.solicitud !== null || paso.claveConservada || paso.ocupado || paso.calculando) return;
    guardarBorradores();
    try { paso.valores.clave_idempotencia = generarClaveIdempotencia() ?? ""; } catch {
      paso.mensaje = "llamamiento_validacion";
      paso.tono = "error";
    }
    repintar();
    raiz.querySelector(`#ct-llamamiento-${operacion}-clave_idempotencia`)?.focus?.();
  }

  raiz.addEventListener("submit", alEnviar);
  raiz.addEventListener("click", alPulsar);
  raiz.addEventListener("change", alCambiarArchivo);
  if (!actualizarContexto(contexto)) repintar();
  const desmontar = () => {
    if (!montado) return;
    montado = false;
    for (const operacion of Object.keys(OPERACIONES)) {
      estado[operacion].lecturaCorreo += 1;
      estado[operacion].controlador?.abort();
    }
    raiz.removeEventListener("submit", alEnviar);
    raiz.removeEventListener("click", alPulsar);
    raiz.removeEventListener("change", alCambiarArchivo);
    raiz.replaceChildren();
  };
  desmontar.actualizarContexto = actualizarContexto;
  return desmontar;
}
