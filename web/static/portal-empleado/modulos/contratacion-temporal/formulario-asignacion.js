/** Formulario real y cerrado para asignar un expediente a la unidad de desarrollo. */

import {
  validarReciboAsignacion,
  validarSolicitudAsignacion,
} from "./contrato-asignacion.js";
import { crearTraductorContratacionTemporal } from "./i18n.js";

const CAMPOS_CONFIGURACION = new Set([
  "raiz", "cliente", "contexto", "generarClaveIdempotencia",
  "confirmarOperacion", "alConfirmar", "mensajes", "locale", "zonaHoraria", "anunciar",
]);
const PATRON_REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const UNIDAD_REF = "unidad:desarrollo:rrhh";
const RESPONSABLE_REF = "persona:responsable-sintetica-001";

function escaparHTML(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#039;");
}

function configuracionCerrada(configuracion) {
  if (configuracion === null || typeof configuracion !== "object"
    || Array.isArray(configuracion)
    || Object.getPrototypeOf(configuracion) !== Object.prototype
    || Object.getOwnPropertySymbols(configuracion).length !== 0) return false;
  const descriptores = Object.getOwnPropertyDescriptors(configuracion);
  return Object.keys(descriptores).every((campo) => CAMPOS_CONFIGURACION.has(campo)
    && Object.hasOwn(descriptores[campo], "value")
    && descriptores[campo].enumerable === true);
}

function normalizarContexto(contexto) {
  if (contexto === null || typeof contexto !== "object" || Array.isArray(contexto)
    || Object.getPrototypeOf(contexto) !== Object.prototype
    || Object.getOwnPropertySymbols(contexto).length !== 0
    || Object.keys(contexto).length !== 2
    || !Object.hasOwn(contexto, "expediente_ref")
    || !Object.hasOwn(contexto, "version_esperada")
    || !PATRON_REFERENCIA.test(contexto.expediente_ref)
    || !Number.isSafeInteger(contexto.version_esperada)
    || contexto.version_esperada < 1
    || contexto.version_esperada >= Number.MAX_SAFE_INTEGER) {
    throw new TypeError("contexto de asignación no válido");
  }
  return Object.freeze({
    expediente_ref: contexto.expediente_ref,
    version_esperada: contexto.version_esperada,
  });
}

function resultadoIndeterminado(error) {
  try { return error?.resultadoIndeterminado === true; } catch { return true; }
}

function validarReciboLigado(respuesta, contexto) {
  const recibo = validarReciboAsignacion(JSON.stringify(respuesta));
  if (recibo.operacion !== "asignar"
    || recibo.expediente_ref !== contexto.expediente_ref
    || recibo.version_resultante !== contexto.version_esperada + 1) {
    throw new TypeError("recibo de asignación no ligado");
  }
  return recibo;
}

function renderizarRecibo(recibo, t, formateador) {
  return `<section class="ct-recibo" data-ct-asignacion-recibo role="status"
    aria-live="polite" aria-atomic="true" tabindex="-1"
    aria-labelledby="ct-asignacion-recibo-titulo">
    <p class="sobrelinea">${escaparHTML(t("asignacion_recibo_sobrelinea"))}</p>
    <h3 id="ct-asignacion-recibo-titulo">${escaparHTML(t("asignacion_recibo_titulo"))}</h3>
    <p>${escaparHTML(t("asignacion_recibo_descripcion"))}</p>
    <dl>
      <div><dt>${escaparHTML(t("asignacion_recibo_expediente"))}</dt><dd><code>${escaparHTML(recibo.expediente_ref)}</code></dd></div>
      <div><dt>${escaparHTML(t("asignacion_recibo_version"))}</dt><dd>${recibo.version_resultante}</dd></div>
      <div><dt>${escaparHTML(t("asignacion_recibo_referencia"))}</dt><dd><code>${escaparHTML(recibo.recibo_ref)}</code></dd></div>
      <div><dt>${escaparHTML(t("asignacion_recibo_fecha"))}</dt><dd>${escaparHTML(formateador.format(new Date(recibo.confirmada_en)))}</dd></div>
    </dl>
  </section>`;
}

function renderizarFormulario(estado, contexto, t) {
  if (estado.recibo) return "";
  if (estado.indeterminado) {
    return `<section class="ct-alcance" data-ct-asignacion-indeterminada role="status"
      aria-live="assertive" aria-atomic="true" tabindex="-1">
      <h3>${escaparHTML(t("asignacion_indeterminada_titulo"))}</h3>
      <p>${escaparHTML(t("asignacion_indeterminada_descripcion"))}</p>
      <div class="ct-acciones"><button class="boton-secundario" type="button"
        data-ct-asignacion-accion="recuperar"${estado.ocupado ? " disabled" : ""}>${
  escaparHTML(t("asignacion_recuperar"))}</button></div>
    </section>`;
  }
  return `<form data-ct-asignacion-form>
    <fieldset><legend>${escaparHTML(t("asignacion_destino_leyenda"))}</legend>
      <div class="ct-campo"><span>${escaparHTML(t("asignacion_unidad"))}</span>
        <output><code>${escaparHTML(UNIDAD_REF)}</code></output>
        <small>${escaparHTML(t("asignacion_unidad_ayuda"))}</small></div>
      <div class="ct-campo"><label for="ct-asignacion-responsable">${escaparHTML(t("asignacion_responsable"))}</label>
        <select id="ct-asignacion-responsable" name="responsable_ref" required>
          <option value="${escaparHTML(RESPONSABLE_REF)}">${escaparHTML(RESPONSABLE_REF)}</option>
        </select><small>${escaparHTML(t("asignacion_responsable_ayuda"))}</small></div>
      <div class="ct-campo"><label><input name="confirmacion" type="checkbox" required>
        ${escaparHTML(t("asignacion_confirmacion"))}</label></div>
    </fieldset>
    <p class="ct-alcance">${escaparHTML(t("asignacion_resumen", {
      expediente: contexto.expediente_ref,
      version: contexto.version_esperada,
    }))}</p>
    <div class="ct-acciones"><button class="boton-primario" type="submit"${
  estado.ocupado ? " disabled" : ""}>${escaparHTML(t("asignacion_confirmar"))}</button></div>
  </form>`;
}

export function montarFormularioAsignacion(configuracion = {}) {
  if (!configuracionCerrada(configuracion)) {
    throw new TypeError("configuración del formulario de asignación no válida");
  }
  let {
    raiz, cliente, contexto: contextoEntrada,
    generarClaveIdempotencia = () => globalThis.crypto?.randomUUID?.(),
    confirmarOperacion = () => false, alConfirmar = () => true,
    mensajes = {}, locale = "es-ES",
    zonaHoraria = "Europe/Madrid", anunciar = () => {},
  } = configuracion;
  configuracion = null;
  if (!raiz || typeof raiz.addEventListener !== "function"
    || typeof raiz.removeEventListener !== "function"
    || typeof raiz.querySelector !== "function" || typeof raiz.contains !== "function"
    || typeof raiz.replaceChildren !== "function"
    || typeof cliente?.asignarUnidad !== "function"
    || typeof generarClaveIdempotencia !== "function"
    || typeof confirmarOperacion !== "function" || typeof alConfirmar !== "function"
    || typeof anunciar !== "function"
    || typeof AbortController !== "function") {
    throw new TypeError("dependencias del formulario de asignación no válidas");
  }
  let contexto = normalizarContexto(contextoEntrada);
  let raizActual = raiz;
  let clienteActual = cliente;
  let generarClaveActual = generarClaveIdempotencia;
  let confirmarActual = confirmarOperacion;
  let alConfirmarActual = alConfirmar;
  let anunciarActual = anunciar;
  let t = crearTraductorContratacionTemporal(mensajes);
  let formateador = new Intl.DateTimeFormat(locale, {
    dateStyle: "long", timeStyle: "medium", timeZone: zonaHoraria,
  });
  let controlador = null;
  let vuelo = null;
  let solicitudActual = null;
  let montado = true;
  let estado = {
    ocupado: false, indeterminado: false, recibo: null,
    mensaje_clave: "asignacion_estado_lista", tipo_mensaje: "informacion",
  };
  raiz = null;
  cliente = null;
  contextoEntrada = null;
  generarClaveIdempotencia = null;
  confirmarOperacion = null;
  alConfirmar = null;
  mensajes = null;
  locale = null;
  zonaHoraria = null;
  anunciar = null;

  function enfocar(selector) {
    const elemento = raizActual?.querySelector?.(selector);
    elemento?.focus?.();
    elemento?.scrollIntoView?.({ block: "nearest", inline: "nearest" });
  }

  function repintar(selectorFoco = "") {
    if (!montado) return;
    raizActual.innerHTML = `<section class="ct-alta" data-ct-asignacion
      aria-labelledby="ct-asignacion-titulo">
      <header class="ct-cabecera"><div>
        <p class="sobrelinea">${escaparHTML(t("asignacion_sobrelinea"))}</p>
        <h2 id="ct-asignacion-titulo">${escaparHTML(t("asignacion_titulo"))}</h2>
        <p>${escaparHTML(t("asignacion_descripcion"))}</p>
      </div><aside class="ct-alcance" aria-label="${escaparHTML(t("asignacion_alcance_etiqueta"))}">${
  escaparHTML(t("asignacion_alcance"))}</aside></header>
      <div class="ct-estado ct-estado-${escaparHTML(estado.tipo_mensaje)}"
        data-ct-asignacion-estado role="status" aria-live="polite"
        aria-atomic="true" tabindex="-1"><strong>${escaparHTML(t(estado.mensaje_clave))}</strong></div>
      ${estado.recibo ? renderizarRecibo(estado.recibo, t, formateador) : ""}
      ${renderizarFormulario(estado, contexto, t)}
    </section>`;
    if (selectorFoco) enfocar(selectorFoco);
    try { anunciarActual(t(estado.mensaje_clave), estado.tipo_mensaje); } catch {
      // La región viva conserva el estado aunque falle un anunciador auxiliar.
    }
  }

  function prepararSolicitud() {
    if (solicitudActual !== null) return solicitudActual;
    const clave = generarClaveActual();
    solicitudActual = validarSolicitudAsignacion(JSON.stringify({
      expediente_ref: contexto.expediente_ref,
      version_esperada: contexto.version_esperada,
      clave_idempotencia: clave,
      unidad_ref: UNIDAD_REF,
      responsable_ref: RESPONSABLE_REF,
    }));
    return solicitudActual;
  }

  function enviar(recurriendo = false) {
    if (!montado || vuelo !== null || estado.ocupado || estado.recibo) {
      return vuelo ?? Promise.resolve(null);
    }
    let solicitud;
    try { solicitud = prepararSolicitud(); } catch {
      estado = { ...estado, mensaje_clave: "asignacion_estado_error", tipo_mensaje: "error" };
      repintar("[data-ct-asignacion-estado]");
      return Promise.resolve(null);
    }
    controlador = new AbortController();
    estado = {
      ...estado, ocupado: true, indeterminado: recurriendo,
      mensaje_clave: recurriendo ? "asignacion_estado_recuperando" : "asignacion_estado_enviando",
      tipo_mensaje: "informacion",
    };
    repintar("[data-ct-asignacion-estado]");
    const tarea = (async () => {
      try {
        const recibo = validarReciboLigado(await clienteActual.asignarUnidad(
          solicitud,
          Object.freeze({ signal: controlador.signal }),
        ), contexto);
        if (!montado) return null;
        estado = {
          ...estado, ocupado: false, indeterminado: false, recibo,
          mensaje_clave: "asignacion_estado_confirmada", tipo_mensaje: "exito",
        };
        solicitudActual = null;
        try {
          if (alConfirmarActual(recibo) !== true) {
            estado = {
              ...estado,
              mensaje_clave: "asignacion_estado_informe_no_disponible",
              tipo_mensaje: "aviso",
            };
          }
        } catch {
          estado = {
            ...estado,
            mensaje_clave: "asignacion_estado_informe_no_disponible",
            tipo_mensaje: "aviso",
          };
        }
        return recibo;
      } catch (error) {
        if (!montado) return null;
        if (resultadoIndeterminado(error)) {
          estado = {
            ...estado, ocupado: false, indeterminado: true,
            mensaje_clave: "asignacion_estado_indeterminado", tipo_mensaje: "aviso",
          };
        } else {
          estado = {
            ...estado, ocupado: false, indeterminado: false,
            mensaje_clave: "asignacion_estado_rechazada", tipo_mensaje: "error",
          };
        }
        return null;
      } finally {
        controlador = null;
        vuelo = null;
        if (montado) repintar(estado.recibo
          ? "[data-ct-asignacion-recibo]"
          : estado.indeterminado ? "[data-ct-asignacion-indeterminada]"
            : "[data-ct-asignacion-estado]");
      }
    })();
    vuelo = tarea;
    return tarea;
  }

  function alEnviar(evento) {
    const formulario = evento.target?.closest?.("[data-ct-asignacion-form]");
    if (!formulario || !raizActual.contains(formulario)) return undefined;
    evento.preventDefault();
    if (typeof formulario.checkValidity === "function" && !formulario.checkValidity()) {
      formulario.reportValidity?.();
      enfocar("[data-ct-asignacion-form] :invalid");
      return Promise.resolve(null);
    }
    if (vuelo !== null || estado.ocupado || estado.recibo || estado.indeterminado) {
      return vuelo ?? Promise.resolve(null);
    }
    let confirmada = false;
    try {
      confirmada = confirmarActual({
        titulo: t("asignacion_confirmar"),
        advertencia: t("asignacion_confirmacion_advertencia"),
        referencia: contexto.expediente_ref,
      }) === true;
    } catch { confirmada = false; }
    return confirmada ? enviar(false) : Promise.resolve(null);
  }

  function alPulsar(evento) {
    const control = evento.target?.closest?.("[data-ct-asignacion-accion]");
    if (!control || !raizActual.contains(control)
      || control.dataset.ctAsignacionAccion !== "recuperar") return undefined;
    evento.preventDefault();
    return solicitudActual === null ? Promise.resolve(null) : enviar(true);
  }

  raizActual.addEventListener("submit", alEnviar);
  raizActual.addEventListener("click", alPulsar);
  repintar();

  return function desmontarFormularioAsignacion() {
    if (!montado) return;
    montado = false;
    controlador?.abort();
    raizActual.removeEventListener("submit", alEnviar);
    raizActual.removeEventListener("click", alPulsar);
    raizActual.replaceChildren();
    estado = null;
    contexto = null;
    clienteActual = null;
    generarClaveActual = null;
    confirmarActual = null;
    alConfirmarActual = null;
    anunciarActual = null;
    controlador = null;
    vuelo = null;
    solicitudActual = null;
    t = null;
    formateador = null;
    raizActual = null;
  };
}
