/** Formulario real para preparar y consultar el informe jurídico de desarrollo. */

import {
  validarReciboInformeJuridico,
  validarSolicitudInformeJuridico,
} from "./contrato-informe-juridico.js";
import { crearTraductorContratacionTemporal } from "./i18n.js";

const CAMPOS_CONFIGURACION = new Set([
  "raiz", "cliente", "contexto", "generarClaveIdempotencia",
  "confirmarOperacion", "mensajes", "locale", "zonaHoraria", "anunciar",
]);
const PATRON_REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;

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
    throw new TypeError("contexto de informe jurídico no válido");
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
  const recibo = validarReciboInformeJuridico(JSON.stringify(respuesta));
  if (recibo.expediente_ref !== contexto.expediente_ref
    || recibo.version_resultante !== contexto.version_esperada + 1) {
    throw new TypeError("recibo de informe jurídico no ligado");
  }
  return recibo;
}

function renderizarFormulario(estado, contexto, t) {
  if (estado.recibo) return "";
  if (estado.indeterminado) {
    return `<section class="ct-alcance" data-ct-informe-indeterminado role="status"
      aria-live="assertive" aria-atomic="true" tabindex="-1">
      <h3>${escaparHTML(t("informe_indeterminado_titulo"))}</h3>
      <p>${escaparHTML(t("informe_indeterminado_descripcion"))}</p>
      <div class="ct-acciones"><button class="boton-secundario" type="button"
        data-ct-informe-accion="recuperar"${estado.ocupado ? " disabled" : ""}>${
  escaparHTML(t("informe_recuperar"))}</button></div>
    </section>`;
  }
  return `<form data-ct-informe-form>
    <fieldset><legend>${escaparHTML(t("informe_confirmacion_leyenda"))}</legend>
      <p>${escaparHTML(t("informe_resumen", {
    expediente: contexto.expediente_ref,
    version: contexto.version_esperada,
  }))}</p>
      <div class="ct-campo"><label><input name="confirmacion" type="checkbox" required>
        ${escaparHTML(t("informe_confirmacion"))}</label></div>
    </fieldset>
    <div class="ct-acciones"><button class="boton-primario" type="submit"${
  estado.ocupado ? " disabled" : ""}>${escaparHTML(t("informe_confirmar"))}</button></div>
  </form>`;
}

function renderizarRecibo(recibo, t, formateador) {
  return `<section class="ct-recibo" data-ct-informe-recibo role="status"
    aria-live="polite" aria-atomic="true" tabindex="-1"
    aria-labelledby="ct-informe-recibo-titulo">
    <p class="sobrelinea">${escaparHTML(t("informe_recibo_sobrelinea"))}</p>
    <h3 id="ct-informe-recibo-titulo">${escaparHTML(t("informe_recibo_titulo"))}</h3>
    <p>${escaparHTML(t("informe_recibo_descripcion"))}</p>
    <dl>
      <div><dt>${escaparHTML(t("informe_recibo_expediente"))}</dt><dd><code>${escaparHTML(recibo.expediente_ref)}</code></dd></div>
      <div><dt>${escaparHTML(t("informe_recibo_version"))}</dt><dd>${recibo.version_resultante}</dd></div>
      <div><dt>${escaparHTML(t("informe_recibo_informe"))}</dt><dd><code>${escaparHTML(recibo.informe_ref)}</code></dd></div>
      <div><dt>${escaparHTML(t("informe_recibo_documento"))}</dt><dd><code>${escaparHTML(recibo.documento_ref)}</code></dd></div>
      <div><dt>${escaparHTML(t("informe_recibo_referencia"))}</dt><dd><code>${escaparHTML(recibo.recibo_ref)}</code></dd></div>
      <div><dt>${escaparHTML(t("informe_recibo_auditoria"))}</dt><dd><code>${escaparHTML(recibo.auditoria_ref)}</code></dd></div>
      <div><dt>${escaparHTML(t("informe_recibo_evento"))}</dt><dd><code>${escaparHTML(recibo.evento_ref)}</code></dd></div>
      <div><dt>${escaparHTML(t("informe_recibo_fecha"))}</dt><dd>${escaparHTML(formateador.format(new Date(recibo.confirmada_en)))}</dd></div>
    </dl>
  </section>`;
}

function renderizarDocumento(recibo, t) {
  return `<section class="ct-bloque" data-ct-informe-documento role="note"
    aria-labelledby="ct-informe-documento-titulo">
    <p class="sobrelinea">${escaparHTML(t("informe_documento_rotulo"))}</p>
    <h3 id="ct-informe-documento-titulo">${escaparHTML(recibo.nombre)}</h3>
    <p><strong>${escaparHTML(t("informe_documento_advertencia"))}</strong></p>
    <dl class="ct-resumen">
      <div><dt>${escaparHTML(t("informe_documento_version"))}</dt><dd>${recibo.version_documento}</dd></div>
      <div><dt>${escaparHTML(t("informe_documento_formato"))}</dt><dd>${escaparHTML(recibo.formato)}</dd></div>
      <div><dt>${escaparHTML(t("informe_documento_huella"))}</dt><dd><code>${escaparHTML(recibo.huella_documento_sha256)}</code></dd></div>
    </dl>
    <h4>${escaparHTML(t("informe_documento_contenido"))}</h4>
    <pre tabindex="0" aria-label="${escaparHTML(t("informe_documento_contenido"))}">${
  escaparHTML(recibo.contenido_desarrollo)}</pre>
  </section>`;
}

function renderizarHistorial(estado, t, formateador) {
  if (!estado.recibo) return "";
  let contenido;
  if (estado.historialCargando) {
    contenido = `<p role="status">${escaparHTML(t("informe_historial_cargando"))}</p>`;
  } else if (estado.historialError) {
    contenido = `<p data-ct-informe-historial-error role="alert" tabindex="-1">${
  escaparHTML(t("informe_historial_no_disponible"))}</p>`;
  } else if (estado.historial?.length === 0) {
    contenido = `<p>${escaparHTML(t("informe_historial_vacio"))}</p>`;
  } else {
    contenido = `<div class="tabla-contenedor" tabindex="0">
      <table class="tabla-datos ct-exp-tabla-panel">
        <caption>${escaparHTML(t("informe_historial_tabla"))}</caption>
        <thead><tr>
          <th scope="col">${escaparHTML(t("informe_historial_secuencia"))}</th>
          <th scope="col">${escaparHTML(t("informe_historial_version"))}</th>
          <th scope="col">${escaparHTML(t("informe_historial_accion"))}</th>
          <th scope="col">${escaparHTML(t("informe_historial_fase"))}</th>
          <th scope="col">${escaparHTML(t("informe_historial_estado"))}</th>
          <th scope="col">${escaparHTML(t("informe_historial_fecha"))}</th>
        </tr></thead>
        <tbody>${estado.historial.map((hito) => `<tr>
          <th scope="row">${hito.secuencia}</th><td>${hito.version_expediente}</td>
          <td>${escaparHTML(hito.accion_clave)}</td>
          <td>${escaparHTML(hito.fase_origen ?? "—")} → ${escaparHTML(hito.fase_destino)}</td>
          <td>${escaparHTML(hito.estado_origen)} → ${escaparHTML(hito.estado_destino)}</td>
          <td>${escaparHTML(formateador.format(new Date(hito.realizada_en)))}</td>
        </tr>`).join("")}</tbody>
      </table>
    </div>`;
  }
  return `<section class="ct-bloque" data-ct-informe-historial
    aria-labelledby="ct-informe-historial-titulo">
    <h3 id="ct-informe-historial-titulo">${escaparHTML(t("informe_historial_titulo"))}</h3>
    <p>${escaparHTML(t("informe_historial_descripcion"))}</p>${contenido}
  </section>`;
}

export function montarFormularioInformeJuridico(configuracion = {}) {
  if (!configuracionCerrada(configuracion)) {
    throw new TypeError("configuración del formulario de informe jurídico no válida");
  }
  let {
    raiz, cliente, contexto: contextoEntrada,
    generarClaveIdempotencia = () => globalThis.crypto?.randomUUID?.(),
    confirmarOperacion = () => false, mensajes = {}, locale = "es-ES",
    zonaHoraria = "Europe/Madrid", anunciar = () => {},
  } = configuracion;
  configuracion = null;
  if (!raiz || typeof raiz.addEventListener !== "function"
    || typeof raiz.removeEventListener !== "function"
    || typeof raiz.querySelector !== "function" || typeof raiz.contains !== "function"
    || typeof raiz.replaceChildren !== "function"
    || typeof cliente?.prepararInformeJuridico !== "function"
    || typeof cliente?.consultarDetalleRRHH !== "function"
    || typeof generarClaveIdempotencia !== "function"
    || typeof confirmarOperacion !== "function" || typeof anunciar !== "function"
    || typeof AbortController !== "function") {
    throw new TypeError("dependencias del formulario de informe jurídico no válidas");
  }
  let contexto = normalizarContexto(contextoEntrada);
  let raizActual = raiz;
  let clienteActual = cliente;
  let generarClaveActual = generarClaveIdempotencia;
  let confirmarActual = confirmarOperacion;
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
    historial: null, historialCargando: false, historialError: false,
    mensaje_clave: "informe_estado_listo", tipo_mensaje: "informacion",
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
    raizActual.innerHTML = `<section class="ct-alta" data-ct-informe
      aria-labelledby="ct-informe-titulo">
      <header class="ct-cabecera"><div>
        <p class="sobrelinea">${escaparHTML(t("informe_sobrelinea"))}</p>
        <h2 id="ct-informe-titulo">${escaparHTML(t("informe_titulo"))}</h2>
        <p>${escaparHTML(t("informe_descripcion"))}</p>
      </div><aside class="ct-alcance" aria-label="${escaparHTML(t("informe_alcance_etiqueta"))}">${
  escaparHTML(t("informe_alcance"))}</aside></header>
      <div class="ct-estado ct-estado-${escaparHTML(estado.tipo_mensaje)}"
        data-ct-informe-estado role="status" aria-live="polite"
        aria-atomic="true" tabindex="-1"><strong>${escaparHTML(t(estado.mensaje_clave))}</strong></div>
      ${renderizarFormulario(estado, contexto, t)}
      ${estado.recibo ? renderizarRecibo(estado.recibo, t, formateador) : ""}
      ${estado.recibo ? renderizarDocumento(estado.recibo, t) : ""}
      ${renderizarHistorial(estado, t, formateador)}
    </section>`;
    if (selectorFoco) enfocar(selectorFoco);
    try { anunciarActual(t(estado.mensaje_clave), estado.tipo_mensaje); } catch {
      // La región viva conserva el estado aunque falle un anunciador auxiliar.
    }
  }

  function prepararSolicitud() {
    if (solicitudActual !== null) return solicitudActual;
    solicitudActual = validarSolicitudInformeJuridico(JSON.stringify({
      expediente_ref: contexto.expediente_ref,
      version_esperada: contexto.version_esperada,
      clave_idempotencia: generarClaveActual(),
    }));
    return solicitudActual;
  }

  async function cargarHistorial(recibo, signal) {
    try {
      const detalle = await clienteActual.consultarDetalleRRHH(Object.freeze({
        expediente_ref: recibo.expediente_ref,
        version_observada: recibo.version_resultante,
      }), Object.freeze({ signal }));
      const ultimoHito = detalle.hitos.at(-1);
      if (detalle.resumen.expediente_ref !== recibo.expediente_ref
        || detalle.resumen.version !== recibo.version_resultante
        || ultimoHito?.secuencia !== recibo.version_resultante
        || ultimoHito?.version_expediente !== recibo.version_resultante
        || ultimoHito?.accion_clave
          !== "contratacion_temporal.informe_juridico.generar"
        || ultimoHito?.fase_destino !== "informe_juridico") {
        throw new TypeError("historial no ligado al recibo de informe jurídico");
      }
      if (!montado) return;
      estado = {
        ...estado, historial: Object.freeze([...detalle.hitos]),
        historialCargando: false, historialError: false,
      };
    } catch {
      if (!montado) return;
      estado = {
        ...estado, historial: null, historialCargando: false, historialError: true,
        mensaje_clave: "informe_estado_historial_no_disponible", tipo_mensaje: "aviso",
      };
    }
  }

  function enviar(recurriendo = false) {
    if (!montado || vuelo !== null || estado.ocupado || estado.recibo) {
      return vuelo ?? Promise.resolve(null);
    }
    let solicitud;
    try { solicitud = prepararSolicitud(); } catch {
      estado = { ...estado, mensaje_clave: "informe_estado_error", tipo_mensaje: "error" };
      repintar("[data-ct-informe-estado]");
      return Promise.resolve(null);
    }
    controlador = new AbortController();
    estado = {
      ...estado, ocupado: true, indeterminado: recurriendo,
      mensaje_clave: recurriendo ? "informe_estado_recuperando" : "informe_estado_enviando",
      tipo_mensaje: "informacion",
    };
    repintar("[data-ct-informe-estado]");
    const tarea = (async () => {
      try {
        const recibo = validarReciboLigado(await clienteActual.prepararInformeJuridico(
          solicitud,
          Object.freeze({ signal: controlador.signal }),
        ), contexto);
        if (!montado) return null;
        estado = {
          ...estado, ocupado: false, indeterminado: false, recibo,
          historial: null, historialCargando: true, historialError: false,
          mensaje_clave: "informe_estado_confirmado", tipo_mensaje: "exito",
        };
        solicitudActual = null;
        repintar("[data-ct-informe-recibo]");
        await cargarHistorial(recibo, controlador.signal);
        return recibo;
      } catch (error) {
        if (!montado) return null;
        if (resultadoIndeterminado(error)) {
          estado = {
            ...estado, ocupado: false, indeterminado: true,
            mensaje_clave: "informe_estado_indeterminado", tipo_mensaje: "aviso",
          };
        } else {
          estado = {
            ...estado, ocupado: false, indeterminado: false,
            mensaje_clave: "informe_estado_rechazado", tipo_mensaje: "error",
          };
        }
        return null;
      } finally {
        controlador = null;
        vuelo = null;
        if (montado) repintar(estado.historialError
          ? "[data-ct-informe-historial-error]"
          : estado.recibo ? "[data-ct-informe-historial]"
            : estado.indeterminado ? "[data-ct-informe-indeterminado]"
              : "[data-ct-informe-estado]");
      }
    })();
    vuelo = tarea;
    return tarea;
  }

  function alEnviar(evento) {
    const formulario = evento.target?.closest?.("[data-ct-informe-form]");
    if (!formulario || !raizActual.contains(formulario)) return undefined;
    evento.preventDefault();
    if (typeof formulario.checkValidity === "function" && !formulario.checkValidity()) {
      formulario.reportValidity?.();
      enfocar("[data-ct-informe-form] :invalid");
      return Promise.resolve(null);
    }
    if (vuelo !== null || estado.ocupado || estado.recibo || estado.indeterminado) {
      return vuelo ?? Promise.resolve(null);
    }
    let confirmada = false;
    try {
      confirmada = confirmarActual({
        titulo: t("informe_confirmar"),
        advertencia: t("informe_confirmacion_advertencia"),
        referencia: contexto.expediente_ref,
      }) === true;
    } catch { confirmada = false; }
    return confirmada ? enviar(false) : Promise.resolve(null);
  }

  function alPulsar(evento) {
    const control = evento.target?.closest?.("[data-ct-informe-accion]");
    if (!control || !raizActual.contains(control)
      || control.dataset.ctInformeAccion !== "recuperar") return undefined;
    evento.preventDefault();
    return solicitudActual === null ? Promise.resolve(null) : enviar(true);
  }

  raizActual.addEventListener("submit", alEnviar);
  raizActual.addEventListener("click", alPulsar);
  repintar();

  return function desmontarFormularioInformeJuridico() {
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
    anunciarActual = null;
    controlador = null;
    vuelo = null;
    solicitudActual = null;
    t = null;
    formateador = null;
    raizActual = null;
  };
}
