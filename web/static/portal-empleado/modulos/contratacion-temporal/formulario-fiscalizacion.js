/** Formulario real para registrar el resultado de fiscalización. */

import {
  validarReciboResultadoFiscalizacion,
  validarSolicitudResultadoFiscalizacion,
} from "./contrato-fiscalizacion.js";
import { crearTraductorContratacionTemporal } from "./i18n.js";

const CAMPOS_CONFIGURACION = new Set([
  "raiz", "cliente", "contexto", "generarClaveIdempotencia",
  "confirmarOperacion", "mensajes", "locale", "zonaHoraria", "anunciar", "alConfirmar",
]);
const PATRON_REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const RESULTADOS = Object.freeze([
  "favorable", "favorable_con_observaciones", "desfavorable",
]);

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
  const campos = ["expediente_ref", "version_esperada", "fase_clave", "informe_ref"];
  if (contexto === null || typeof contexto !== "object" || Array.isArray(contexto)
    || Object.getPrototypeOf(contexto) !== Object.prototype
    || Object.getOwnPropertySymbols(contexto).length !== 0
    || Object.keys(contexto).length !== campos.length
    || campos.some((campo) => !Object.hasOwn(contexto, campo))
    || !PATRON_REFERENCIA.test(contexto.expediente_ref)
    || !Number.isSafeInteger(contexto.version_esperada)
    || contexto.version_esperada < 1 || contexto.version_esperada >= Number.MAX_SAFE_INTEGER
    || contexto.fase_clave !== "informe_juridico"
    || (contexto.informe_ref !== "" && !PATRON_REFERENCIA.test(contexto.informe_ref))) {
    throw new TypeError("contexto de fiscalización no válido");
  }
  return Object.freeze(Object.fromEntries(
    campos.map((campo) => [campo, contexto[campo]]),
  ));
}

function resultadoIndeterminado(error) {
  try { return error?.resultadoIndeterminado === true; } catch { return true; }
}

function validarReciboLigado(respuesta, contexto, solicitud) {
  const recibo = validarReciboResultadoFiscalizacion(JSON.stringify(respuesta));
  if (recibo.expediente_ref !== contexto.expediente_ref
    || recibo.version_resultante !== contexto.version_esperada + 1
    || recibo.resultado !== solicitud.resultado) {
    throw new TypeError("recibo de fiscalización no ligado");
  }
  return recibo;
}

function etiquetaResultado(resultado, t) {
  return t(`fiscalizacion_resultado_${resultado}`);
}

function renderizarContexto(contexto, t) {
  const informe = contexto.informe_ref === ""
    ? t("fiscalizacion_informe_registrado", { version: contexto.version_esperada })
    : contexto.informe_ref;
  return `<dl class="ct-resumen" data-ct-fiscalizacion-contexto>
    <div><dt>${escaparHTML(t("fiscalizacion_contexto_expediente"))}</dt><dd><code>${
  escaparHTML(contexto.expediente_ref)}</code></dd></div>
    <div><dt>${escaparHTML(t("fiscalizacion_contexto_version"))}</dt><dd>${
  contexto.version_esperada}</dd></div>
    <div><dt>${escaparHTML(t("fiscalizacion_contexto_fase"))}</dt><dd>${
  escaparHTML(t("fiscalizacion_fase_informe_juridico"))}</dd></div>
    <div><dt>${escaparHTML(t("fiscalizacion_contexto_informe"))}</dt><dd><code>${
  escaparHTML(informe)}</code></dd></div>
  </dl>`;
}

function renderizarFormulario(estado, t) {
  if (estado.recibo) return "";
  if (estado.indeterminado) {
    return `<section class="ct-alcance" data-ct-fiscalizacion-indeterminada role="status"
      aria-live="assertive" aria-atomic="true" tabindex="-1">
      <h3>${escaparHTML(t("fiscalizacion_indeterminada_titulo"))}</h3>
      <p>${escaparHTML(t("fiscalizacion_indeterminada_descripcion"))}</p>
      <div class="ct-acciones"><button class="boton-secundario" type="button"
        data-ct-fiscalizacion-accion="recuperar"${estado.ocupado ? " disabled" : ""}>${
  escaparHTML(t("fiscalizacion_recuperar"))}</button></div>
    </section>`;
  }
  const necesitaObservaciones = estado.resultado !== ""
    && estado.resultado !== "favorable";
  const prohibeObservaciones = estado.resultado === "favorable";
  return `<form data-ct-fiscalizacion-form novalidate>
    ${estado.validacionError ? `<p data-ct-fiscalizacion-error role="alert" tabindex="-1">${
  escaparHTML(t("fiscalizacion_estado_validacion"))}</p>` : ""}
    <fieldset><legend>${escaparHTML(t("fiscalizacion_resultado_leyenda"))}</legend>
      <p id="ct-fiscalizacion-resultado-ayuda">${escaparHTML(t("fiscalizacion_resultado_ayuda"))}</p>
      ${RESULTADOS.map((resultado) => `<label><input name="resultado" type="radio"
        value="${resultado}" required aria-describedby="ct-fiscalizacion-resultado-ayuda"${
  estado.resultado === resultado ? " checked" : ""}> ${
  escaparHTML(etiquetaResultado(resultado, t))}</label>`).join("")}
    </fieldset>
    <div class="ct-campo"><label for="ct-fiscalizacion-observaciones">${
  escaparHTML(t("fiscalizacion_observaciones"))}</label>
      <textarea id="ct-fiscalizacion-observaciones" name="observaciones" rows="5"
        maxlength="2000" aria-describedby="ct-fiscalizacion-observaciones-ayuda"${
  necesitaObservaciones ? " required aria-required=\"true\"" : ""}${
  prohibeObservaciones ? " disabled" : ""}${
  estado.validacionError ? " aria-invalid=\"true\"" : ""}>${
  escaparHTML(estado.observaciones)}</textarea>
      <p id="ct-fiscalizacion-observaciones-ayuda">${
  escaparHTML(t("fiscalizacion_observaciones_ayuda"))}</p></div>
    <div class="ct-acciones"><button class="boton-primario" type="submit"${
  estado.ocupado ? " disabled" : ""}>${escaparHTML(t("fiscalizacion_confirmar"))}</button></div>
  </form>`;
}

function renderizarRecibo(recibo, t, formateador) {
  const retorno = Object.hasOwn(recibo, "unidad_retorno_ref")
    ? `<div><dt>${escaparHTML(t("fiscalizacion_recibo_unidad_retorno"))}</dt><dd><code>${
      escaparHTML(recibo.unidad_retorno_ref)}</code></dd></div>
      <div><dt>${escaparHTML(t("fiscalizacion_recibo_responsable_retorno"))}</dt><dd><code>${
      escaparHTML(recibo.responsable_retorno_ref)}</code></dd></div>` : "";
  return `<section class="ct-recibo" data-ct-fiscalizacion-recibo role="status"
    aria-live="polite" aria-atomic="true" tabindex="-1"
    aria-labelledby="ct-fiscalizacion-recibo-titulo">
    <p class="sobrelinea">${escaparHTML(t("fiscalizacion_recibo_sobrelinea"))}</p>
    <h3 id="ct-fiscalizacion-recibo-titulo">${escaparHTML(t("fiscalizacion_recibo_titulo"))}</h3>
    <p>${escaparHTML(t("fiscalizacion_recibo_descripcion"))}</p>
    <dl>
      <div><dt>${escaparHTML(t("fiscalizacion_recibo_expediente"))}</dt><dd><code>${
  escaparHTML(recibo.expediente_ref)}</code></dd></div>
      <div><dt>${escaparHTML(t("fiscalizacion_recibo_resultado"))}</dt><dd>${
  escaparHTML(etiquetaResultado(recibo.resultado, t))}</dd></div>
      <div><dt>${escaparHTML(t("fiscalizacion_recibo_fase"))}</dt><dd>${
  escaparHTML(recibo.fase_resultante.replaceAll("_", " "))}</dd></div>
      <div><dt>${escaparHTML(t("fiscalizacion_recibo_estado"))}</dt><dd>${
  escaparHTML(recibo.estado_resultante.replaceAll("_", " "))}</dd></div>
      <div><dt>${escaparHTML(t("fiscalizacion_recibo_version"))}</dt><dd>${
  recibo.version_resultante}</dd></div>
      <div><dt>${escaparHTML(t("fiscalizacion_recibo_referencia"))}</dt><dd><code>${
  escaparHTML(recibo.recibo_ref)}</code></dd></div>
      <div><dt>${escaparHTML(t("fiscalizacion_recibo_auditoria"))}</dt><dd><code>${
  escaparHTML(recibo.auditoria_ref)}</code></dd></div>
      <div><dt>${escaparHTML(t("fiscalizacion_recibo_evento"))}</dt><dd><code>${
  escaparHTML(recibo.evento_ref)}</code></dd></div>
      <div><dt>${escaparHTML(t("fiscalizacion_recibo_actor"))}</dt><dd><code>${
  escaparHTML(recibo.actor_ref)}</code></dd></div>
      ${retorno}
      <div><dt>${escaparHTML(t("fiscalizacion_recibo_fecha"))}</dt><dd>${
  escaparHTML(formateador.format(new Date(recibo.registrada_en)))}</dd></div>
    </dl>
  </section>`;
}

function leerFormulario(formulario) {
  const control = (nombre) => formulario?.elements?.namedItem?.(nombre);
  return {
    resultado: String(control("resultado")?.value ?? ""),
    observaciones: String(control("observaciones")?.value ?? "").normalize("NFC").trim(),
  };
}

export function montarFormularioFiscalizacion(configuracion = {}) {
  if (!configuracionCerrada(configuracion)) {
    throw new TypeError("configuración del formulario de fiscalización no válida");
  }
  let {
    raiz, cliente, contexto: contextoEntrada,
    generarClaveIdempotencia = () => globalThis.crypto?.randomUUID?.(),
    confirmarOperacion = () => false, mensajes = {}, locale = "es-ES",
    zonaHoraria = "Europe/Madrid", anunciar = () => {}, alConfirmar = () => {},
  } = configuracion;
  configuracion = null;
  if (!raiz || typeof raiz.addEventListener !== "function"
    || typeof raiz.removeEventListener !== "function"
    || typeof raiz.querySelector !== "function" || typeof raiz.contains !== "function"
    || typeof raiz.replaceChildren !== "function"
    || typeof cliente?.registrarResultadoFiscalizacion !== "function"
    || typeof generarClaveIdempotencia !== "function"
    || typeof confirmarOperacion !== "function" || typeof anunciar !== "function"
    || typeof alConfirmar !== "function"
    || typeof AbortController !== "function") {
    throw new TypeError("dependencias del formulario de fiscalización no válidas");
  }
  const contexto = normalizarContexto(contextoEntrada);
  let raizActual = raiz;
  let clienteActual = cliente;
  let generarClaveActual = generarClaveIdempotencia;
  let confirmarActual = confirmarOperacion;
  let anunciarActual = anunciar;
  let alConfirmarActual = alConfirmar;
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
    resultado: "", observaciones: "", validacionError: false,
    mensaje_clave: "fiscalizacion_estado_lista", tipo_mensaje: "informacion",
  };
  raiz = null;
  cliente = null;
  contextoEntrada = null;
  generarClaveIdempotencia = null;
  confirmarOperacion = null;
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
    raizActual.innerHTML = `<section class="ct-alta" data-ct-fiscalizacion
      aria-labelledby="ct-fiscalizacion-titulo">
      <header class="ct-cabecera"><div>
        <p class="sobrelinea">${escaparHTML(t("fiscalizacion_sobrelinea"))}</p>
        <h2 id="ct-fiscalizacion-titulo">${escaparHTML(t("fiscalizacion_titulo"))}</h2>
        <p>${escaparHTML(t("fiscalizacion_descripcion"))}</p>
      </div><aside class="ct-alcance" aria-label="${
  escaparHTML(t("fiscalizacion_alcance_etiqueta"))}">${
  escaparHTML(t("fiscalizacion_alcance"))}</aside></header>
      ${renderizarContexto(contexto, t)}
      <div class="ct-estado ct-estado-${escaparHTML(estado.tipo_mensaje)}"
        data-ct-fiscalizacion-estado role="${estado.tipo_mensaje === "error" ? "alert" : "status"}"
        aria-live="polite" aria-atomic="true" tabindex="-1"><strong>${
  escaparHTML(t(estado.mensaje_clave))}</strong></div>
      ${renderizarFormulario(estado, t)}
      ${estado.recibo ? renderizarRecibo(estado.recibo, t, formateador) : ""}
    </section>`;
    if (selectorFoco) enfocar(selectorFoco);
    try { anunciarActual(t(estado.mensaje_clave), estado.tipo_mensaje); } catch {
      // La región viva conserva el estado aunque falle un anunciador auxiliar.
    }
  }

  function prepararSolicitud() {
    if (solicitudActual !== null) return solicitudActual;
    solicitudActual = validarSolicitudResultadoFiscalizacion(JSON.stringify({
      expediente_ref: contexto.expediente_ref,
      version_esperada: contexto.version_esperada,
      clave_idempotencia: generarClaveActual(),
      resultado: estado.resultado,
      observaciones: estado.observaciones,
    }));
    return solicitudActual;
  }

  function enviar(recurriendo = false) {
    if (!montado || vuelo !== null || estado.ocupado || estado.recibo) {
      return vuelo ?? Promise.resolve(null);
    }
    let solicitud;
    try { solicitud = prepararSolicitud(); } catch {
      estado = {
        ...estado, validacionError: true,
        mensaje_clave: "fiscalizacion_estado_validacion", tipo_mensaje: "error",
      };
      repintar("[data-ct-fiscalizacion-error]");
      return Promise.resolve(null);
    }
    controlador = new AbortController();
    estado = {
      ...estado, ocupado: true, indeterminado: recurriendo, validacionError: false,
      mensaje_clave: recurriendo
        ? "fiscalizacion_estado_recuperando" : "fiscalizacion_estado_enviando",
      tipo_mensaje: "informacion",
    };
    repintar("[data-ct-fiscalizacion-estado]");
    const tarea = (async () => {
      try {
        const recibo = validarReciboLigado(
          await clienteActual.registrarResultadoFiscalizacion(
            solicitud,
            Object.freeze({ signal: controlador.signal }),
          ),
          contexto,
          solicitud,
        );
        if (!montado) return null;
        estado = {
          ...estado, ocupado: false, indeterminado: false, recibo,
          mensaje_clave: "fiscalizacion_estado_confirmada", tipo_mensaje: "exito",
        };
        solicitudActual = null;
        try { alConfirmarActual(recibo); } catch {
          // El recibo confirmado prevalece aunque no se pueda montar el siguiente paso.
        }
        return recibo;
      } catch (error) {
        if (!montado) return null;
        if (resultadoIndeterminado(error)) {
          estado = {
            ...estado, ocupado: false, indeterminado: true,
            mensaje_clave: "fiscalizacion_estado_indeterminado", tipo_mensaje: "aviso",
          };
        } else {
          estado = {
            ...estado, ocupado: false, indeterminado: false,
            mensaje_clave: "fiscalizacion_estado_rechazada", tipo_mensaje: "error",
          };
        }
        return null;
      } finally {
        controlador = null;
        vuelo = null;
        if (montado) repintar(estado.recibo
          ? "[data-ct-fiscalizacion-recibo]"
          : estado.indeterminado ? "[data-ct-fiscalizacion-indeterminada]"
            : "[data-ct-fiscalizacion-estado]");
      }
    })();
    vuelo = tarea;
    return tarea;
  }

  function alEnviar(evento) {
    const formulario = evento.target?.closest?.("[data-ct-fiscalizacion-form]");
    if (!formulario || !raizActual.contains(formulario)) return undefined;
    evento.preventDefault();
    const borrador = leerFormulario(formulario);
    estado = { ...estado, ...borrador, validacionError: false };
    solicitudActual = null;
    try { prepararSolicitud(); } catch {
      estado = {
        ...estado, validacionError: true,
        mensaje_clave: "fiscalizacion_estado_validacion", tipo_mensaje: "error",
      };
      repintar("[data-ct-fiscalizacion-error]");
      return Promise.resolve(null);
    }
    if (typeof formulario.checkValidity === "function" && !formulario.checkValidity()) {
      formulario.reportValidity?.();
      enfocar("[data-ct-fiscalizacion-form] :invalid");
      return Promise.resolve(null);
    }
    if (vuelo !== null || estado.ocupado || estado.recibo || estado.indeterminado) {
      return vuelo ?? Promise.resolve(null);
    }
    let confirmada = false;
    try {
      confirmada = confirmarActual({
        titulo: t("fiscalizacion_confirmar"),
        advertencia: t(estado.resultado === "desfavorable"
          ? "fiscalizacion_confirmacion_desfavorable"
          : "fiscalizacion_confirmacion_continuar"),
        referencia: contexto.expediente_ref,
      }) === true;
    } catch { confirmada = false; }
    return confirmada ? enviar(false) : Promise.resolve(null);
  }

  function alPulsar(evento) {
    const control = evento.target?.closest?.("[data-ct-fiscalizacion-accion]");
    if (!control || !raizActual.contains(control)
      || control.dataset.ctFiscalizacionAccion !== "recuperar") return undefined;
    evento.preventDefault();
    return solicitudActual === null ? Promise.resolve(null) : enviar(true);
  }

  function alCambiar(evento) {
    const control = evento.target;
    if (!control || !raizActual.contains(control)
      || !["resultado", "observaciones"].includes(control.name)
      || estado.indeterminado || estado.recibo) return;
    if (control.name === "resultado" && RESULTADOS.includes(control.value)) {
      const esFavorable = control.value === "favorable";
      estado = {
        ...estado,
        resultado: control.value,
        observaciones: esFavorable ? "" : estado.observaciones,
        validacionError: false,
      };
      const observaciones = raizActual.querySelector("[name=\"observaciones\"]");
      if (observaciones) {
        observaciones.required = !esFavorable;
        observaciones.disabled = esFavorable;
        if (esFavorable) observaciones.value = "";
        if (observaciones.required) observaciones.setAttribute?.("aria-required", "true");
        else observaciones.removeAttribute?.("aria-required");
      }
    } else if (control.name === "observaciones") {
      estado = { ...estado, observaciones: String(control.value), validacionError: false };
    }
    solicitudActual = null;
  }

  raizActual.addEventListener("submit", alEnviar);
  raizActual.addEventListener("click", alPulsar);
  raizActual.addEventListener("change", alCambiar);
  raizActual.addEventListener("input", alCambiar);
  repintar();

  return function desmontarFormularioFiscalizacion() {
    if (!montado) return;
    montado = false;
    controlador?.abort();
    raizActual.removeEventListener("submit", alEnviar);
    raizActual.removeEventListener("click", alPulsar);
    raizActual.removeEventListener("change", alCambiar);
    raizActual.removeEventListener("input", alCambiar);
    raizActual.replaceChildren();
    raizActual = null;
    clienteActual = null;
    generarClaveActual = null;
    confirmarActual = null;
    anunciarActual = null;
    alConfirmarActual = null;
    t = null;
    formateador = null;
    solicitudActual = null;
    estado = null;
  };
}
