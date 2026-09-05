/** Una intención visible por acción; no se guarda nada en el navegador. */
import { crearTraductorContratacionTemporal } from "./i18n.js";
import { renderizarLlamamiento } from "./renderizado-llamamiento.js";
import {
  CAMPOS_SELECCION, CAMPOS_COMUNICACION, referenciaLlamamientoValida,
  validarSolicitudSeleccionLlamamiento, validarSolicitudComunicacionLlamamiento,
  validarReciboSeleccionLlamamiento, validarReciboComunicacionLlamamiento,
  CAMPOS_RESPUESTA_RECIBIDA, CAMPOS_RESPUESTA_EDITABLES,
  validarSolicitudRespuestaRecibida, validarReciboRespuestaRecibida,
  CAMPOS_RESOLUCION, validarSolicitudResolucionLlamamiento, validarReciboResolucionLlamamiento,
} from "./contrato-llamamiento.js";

const OPERACIONES = Object.freeze({
  seleccion: {
    campos: CAMPOS_SELECCION, validar: validarSolicitudSeleccionLlamamiento,
    recibo: validarReciboSeleccionLlamamiento, metodo: "seleccionarLlamamiento",
  },
  comunicacion: {
    campos: CAMPOS_COMUNICACION, validar: validarSolicitudComunicacionLlamamiento,
    recibo: validarReciboComunicacionLlamamiento, metodo: "registrarComunicacionLlamamiento",
  },
  respuesta: {
    campos: CAMPOS_RESPUESTA_RECIBIDA, validar: validarSolicitudRespuestaRecibida,
    recibo: validarReciboRespuestaRecibida, metodo: "registrarRespuestaRecibida",
  },
  resolucion: {
    campos: CAMPOS_RESOLUCION, validar: validarSolicitudResolucionLlamamiento,
    recibo: validarReciboResolucionLlamamiento, metodo: "resolverLlamamiento",
  },
});
function nuevoPaso() {
  return { valores: {}, solicitud: null, recibo: null, ocupado: false, bloqueado: false,
    calculando: false,
    mensaje: "llamamiento_pendiente", tono: "informacion", controlador: null };
}
export function montarFormularioLlamamiento({
  raiz, cliente, contexto = null, confirmarOperacion = () => false,
  generarClaveIdempotencia = () => globalThis.crypto?.randomUUID?.(),
  criptografia = globalThis.crypto,
  mensajes = {}, locale = "es-ES", zonaHoraria = "Europe/Madrid", anunciar = () => {},
} = {}) {
  if (!raiz || typeof raiz.addEventListener !== "function"
    || typeof raiz.removeEventListener !== "function" || typeof raiz.querySelector !== "function"
    || typeof raiz.contains !== "function" || typeof raiz.replaceChildren !== "function"
    || Object.values(OPERACIONES).some(({ metodo }) => typeof cliente?.[metodo] !== "function")
    || typeof confirmarOperacion !== "function" || typeof generarClaveIdempotencia !== "function"
    || typeof anunciar !== "function") {
    throw new TypeError("dependencias del formulario de llamamiento no válidas");
  }
  const t = crearTraductorContratacionTemporal(mensajes);
  const fecha = new Intl.DateTimeFormat(locale, {
    dateStyle: "medium", timeStyle: "medium", timeZone: zonaHoraria,
  });
  let montado = true, lecturaCorreo = 0;
  const estado = { seleccion: nuevoPaso(), comunicacion: nuevoPaso(), respuesta: nuevoPaso(),
    resolucion: { ...nuevoPaso(), mensaje: "llamamiento_resolucion_pendiente" },
    enlazado: false, comunicacionAbierta: false };

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
        if (["comunicacion", "resolucion"].includes(operacion) && campo !== "clave_idempotencia") continue;
        if (operacion === "respuesta" && !CAMPOS_RESPUESTA_EDITABLES.includes(campo)) continue;
        paso.valores[campo] = String(formulario.elements.namedItem(campo)?.value ?? "");
      }
    }
    estado.comunicacionAbierta = raiz.querySelector(
      "[data-ct-llamamiento-comunicacion]",
    )?.open === true || estado.comunicacionAbierta;
  }
  async function alCambiarArchivo(evento) {
    const control = evento.target?.closest?.("[data-ct-llamamiento-correo]");
    const paso = estado.respuesta;
    if (!control || !raiz.contains(control) || paso.solicitud || paso.ocupado
      || estado.comunicacion.recibo?.version_resultante !== 2) return;
    guardarBorradores();
    const lectura = ++lecturaCorreo;
    const archivo = control.files?.length === 1 ? control.files[0] : null;
    paso.valores.correo_sha256 = "";
    paso.calculando = true;
    paso.mensaje = "llamamiento_correo_calculando";
    paso.tono = "informacion";
    repintar("respuesta");
    let bytes;
    try {
      // Se limita antes de leer. Solo la huella llega al estado y al POST;
      // el nombre y contenido del correo nunca se proyectan ni se guardan.
      if (!archivo || !/\.eml$/iu.test(archivo.name) || archivo.size < 1
        || archivo.size > 2 * 1024 * 1024 || !criptografia?.subtle?.digest) throw new TypeError();
      bytes = new Uint8Array(await archivo.arrayBuffer());
      if (!montado || lectura !== lecturaCorreo) return;
      if (bytes.byteLength !== archivo.size) throw new TypeError();
      const digest = new Uint8Array(await criptografia.subtle.digest("SHA-256", bytes));
      if (!montado || lectura !== lecturaCorreo) return;
      const huella = Array.from(digest, (byte) => byte.toString(16).padStart(2, "0")).join("");
      if (digest.length !== 32 || huella === "0".repeat(64)) throw new TypeError();
      paso.valores.correo_sha256 = huella;
      paso.mensaje = "llamamiento_correo_calculado";
    } catch {
      if (!montado || lectura !== lecturaCorreo) return;
      paso.mensaje = "llamamiento_correo_error";
      paso.tono = "error";
    } finally {
      bytes?.fill(0);
      if (montado && lectura === lecturaCorreo) {
        paso.calculando = false;
        repintar("respuesta");
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
  async function alEnviar(evento) {
    const formulario = evento.target?.closest?.("[data-ct-llamamiento-form]");
    if (!formulario || !raiz.contains(formulario)) return;
    evento.preventDefault();
    const operacion = formulario.dataset.ctLlamamientoForm;
    if (!Object.hasOwn(OPERACIONES, operacion)) return;
    const paso = estado[operacion];
    if (paso.ocupado || paso.calculando || paso.recibo || paso.bloqueado) return;
    if (operacion === "comunicacion" && estado.seleccion.recibo === null) return;
    if (operacion === "respuesta" && estado.comunicacion.recibo?.version_resultante !== 2) return;
    if (operacion === "resolucion" && estado.respuesta.recibo?.respuesta !== "aceptacion") return;
    guardarBorradores();
    const contrato = OPERACIONES[operacion];
    const recuperandoRespuesta = ["respuesta", "resolucion"].includes(operacion) && paso.solicitud !== null;
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
      if (operacion === "resolucion" && [estado.seleccion, estado.comunicacion, estado.respuesta]
        .some((anterior) => anterior.solicitud?.clave_idempotencia === solicitud.clave_idempotencia)) {
        throw new TypeError("la resolución necesita su propia clave");
      }
    } catch {
      paso.mensaje = operacion === "resolucion" ? "llamamiento_resolucion_validacion"
        : operacion === "respuesta" ? "llamamiento_respuesta_validacion" : "llamamiento_validacion";
      paso.tono = "error";
      repintar(operacion);
      return;
    }
    let confirmado = false;
    try {
      confirmado = confirmarOperacion({
        titulo: t("llamamiento_" + operacion),
        advertencia: t("llamamiento_confirmacion_" + operacion, {
          version: solicitud.version_esperada,
          justificante: solicitud.prueba_respuesta_ref,
          ...(operacion === "respuesta" ? {
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
    paso.valores = { ...solicitud };
    paso.ocupado = true;
    paso.controlador = new AbortController();
    paso.mensaje = operacion === "resolucion" ? "llamamiento_solicitando_resolucion" : "llamamiento_enviando";
    paso.tono = "informacion";
    repintar(operacion);
    let respuestaRecibida = false;
    try {
      const respuesta = await cliente[contrato.metodo](solicitud, {
        signal: paso.controlador.signal,
      });
      respuestaRecibida = true;
      const recibo = contrato.recibo(respuesta, solicitud);
      if (!montado) return;
      guardarBorradores();
      paso.recibo = recibo;
      paso.mensaje = operacion === "resolucion" ? "llamamiento_resolucion_recibo"
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
        estado.respuesta.valores = {
          ...estado.respuesta.valores,
          organizacion_ref: solicitud.organizacion_ref,
          expediente_ref: solicitud.expediente_ref,
          llamamiento_ref: solicitud.llamamiento_ref,
          comunicacion_ref: recibo.comunicacion_ref,
          version_comunicacion_esperada: recibo.version_resultante,
        };
      }
      if (operacion === "respuesta" && recibo.respuesta === "aceptacion") {
        estado.resolucion.valores = {
          ...estado.resolucion.valores,
          organizacion_ref: recibo.organizacion_ref, expediente_ref: recibo.expediente_ref,
          llamamiento_ref: recibo.llamamiento_ref, comunicacion_ref: recibo.comunicacion_ref,
          version_esperada: recibo.version_comunicacion_esperada,
          respuesta: recibo.respuesta, prueba_respuesta_ref: recibo.justificante_ref,
        };
      }
    } catch (error) {
      if (!montado) return;
      guardarBorradores();
      const conflicto = ["conflicto_no_reintentable", "clave_idempotencia_reutilizada",
        "version_en_conflicto", "seleccion_no_disponible"].includes(error?.codigo);
      const validacionPendiente = operacion === "resolucion" && error?.estado === 409
        && error?.codigo === "validacion_respuesta_pendiente" && error?.envelopeValido === true;
      // Denegar un replay no demuestra ausencia de efecto del intento original.
      const rechazo = !validacionPendiente && !recuperandoRespuesta && !respuestaRecibida && error?.resultadoIndeterminado === false
        && error?.envelopeValido === true;
      paso.bloqueado = conflicto;
      paso.mensaje = validacionPendiente ? "llamamiento_validacion_respuesta_pendiente"
        : conflicto ? "llamamiento_conflicto"
        : rechazo ? "llamamiento_rechazada" : "llamamiento_error";
      paso.tono = conflicto || rechazo ? "error" : "aviso";
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
    if (operacion === "resolucion" && estado.respuesta.recibo?.respuesta !== "aceptacion") return;
    const paso = estado[operacion];
    if (paso.solicitud !== null || paso.ocupado || paso.calculando) return;
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
    lecturaCorreo += 1;
    for (const operacion of Object.keys(OPERACIONES)) estado[operacion].controlador?.abort();
    raiz.removeEventListener("submit", alEnviar);
    raiz.removeEventListener("click", alPulsar);
    raiz.removeEventListener("change", alCambiarArchivo);
    raiz.replaceChildren();
  };
  desmontar.actualizarContexto = actualizarContexto;
  return desmontar;
}
