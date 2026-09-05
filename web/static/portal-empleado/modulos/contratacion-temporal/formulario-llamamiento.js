/** Una intención visible por acción; no se guarda nada en el navegador. */
import { crearTraductorContratacionTemporal } from "./i18n.js";
import { renderizarLlamamiento } from "./renderizado-llamamiento.js";
import {
  CAMPOS_SELECCION, CAMPOS_COMUNICACION, referenciaLlamamientoValida,
  validarSolicitudSeleccionLlamamiento, validarSolicitudComunicacionLlamamiento,
  validarReciboSeleccionLlamamiento, validarReciboComunicacionLlamamiento,
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
});
function nuevoPaso() {
  return { valores: {}, solicitud: null, recibo: null, ocupado: false, bloqueado: false,
    mensaje: "llamamiento_pendiente", tono: "informacion", controlador: null };
}
export function montarFormularioLlamamiento({
  raiz, cliente, contexto = null, confirmarOperacion = () => false,
  generarClaveIdempotencia = () => globalThis.crypto?.randomUUID?.(),
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
  let montado = true;
  const estado = { seleccion: nuevoPaso(), comunicacion: nuevoPaso(),
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
        if (operacion === "comunicacion" && campo !== "clave_idempotencia") continue;
        paso.valores[campo] = String(formulario.elements.namedItem(campo)?.value ?? "");
      }
    }
    estado.comunicacionAbierta = raiz.querySelector(
      "[data-ct-llamamiento-comunicacion]",
    )?.open === true || estado.comunicacionAbierta;
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
    if (paso.ocupado || paso.recibo || paso.bloqueado) return;
    if (operacion === "comunicacion" && estado.seleccion.recibo === null) return;
    guardarBorradores();
    const contrato = OPERACIONES[operacion];
    let solicitud;
    try {
      solicitud = paso.solicitud ?? contrato.validar(Object.fromEntries(
        contrato.campos.map((campo) => [
          campo, campo === "version_esperada" ? Number(paso.valores[campo]) : paso.valores[campo],
        ]),
      ));
    } catch {
      paso.mensaje = "llamamiento_validacion";
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
    paso.mensaje = "llamamiento_enviando";
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
      paso.mensaje = "llamamiento_recibo";
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
    } catch (error) {
      if (!montado) return;
      guardarBorradores();
      const conflicto = ["conflicto_no_reintentable", "clave_idempotencia_reutilizada",
        "version_en_conflicto", "seleccion_no_disponible"].includes(error?.codigo);
      const rechazo = !respuestaRecibida && error?.resultadoIndeterminado === false
        && error?.envelopeValido === true;
      paso.bloqueado = conflicto;
      paso.mensaje = conflicto ? "llamamiento_conflicto"
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
    const paso = estado[operacion];
    if (paso.solicitud !== null || paso.ocupado) return;
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
  if (!actualizarContexto(contexto)) repintar();
  const desmontar = () => {
    if (!montado) return;
    montado = false;
    estado.seleccion.controlador?.abort();
    estado.comunicacion.controlador?.abort();
    raiz.removeEventListener("submit", alEnviar);
    raiz.removeEventListener("click", alPulsar);
    raiz.replaceChildren();
  };
  desmontar.actualizarContexto = actualizarContexto;
  return desmontar;
}
