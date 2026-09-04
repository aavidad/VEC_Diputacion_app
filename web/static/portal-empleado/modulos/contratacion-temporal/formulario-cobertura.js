import {
  validarPropuestaCobertura,
  validarReciboCobertura,
  validarResultadoConsultaCobertura,
  validarSolicitudConsultaResultadoCobertura,
  validarSolicitudDecisionCobertura,
  validarSolicitudPropuestaCobertura,
} from "./contrato-cobertura.js";
import { crearTraductorContratacionTemporal } from "./i18n.js";

const CAMPOS_CONFIGURACION = new Set([
  "raiz", "cliente", "contexto", "generarClaveIdempotencia",
  "confirmarOperacion", "alConfirmar", "mensajes", "locale", "zonaHoraria", "anunciar",
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
    throw new TypeError("contexto de cobertura no válido");
  }
  return Object.freeze({
    expediente_ref: contexto.expediente_ref,
    version_esperada: contexto.version_esperada,
  });
}

function resultadoIndeterminado(error) {
  try {
    return error?.resultadoIndeterminado === true;
  } catch {
    return true;
  }
}

function renderizarRecibo(recibo, contexto, t, formateador) {
  return `<section class="ct-recibo" data-ct-cobertura-recibo role="status"
    aria-live="polite" aria-atomic="true" tabindex="-1"
    aria-labelledby="ct-cobertura-recibo-titulo">
    <p class="sobrelinea">${escaparHTML(t("cobertura_recibo_sobrelinea"))}</p>
    <h3 id="ct-cobertura-recibo-titulo">${escaparHTML(t("cobertura_recibo_titulo"))}</h3>
    <p>${escaparHTML(t("cobertura_recibo_descripcion"))}</p>
    <dl>
      <div><dt>${escaparHTML(t("cobertura_recibo_expediente"))}</dt><dd><code>${escaparHTML(contexto.expediente_ref)}</code></dd></div>
      <div><dt>${escaparHTML(t("cobertura_recibo_version"))}</dt><dd>${recibo.version_resultante}</dd></div>
      <div><dt>${escaparHTML(t("cobertura_recibo_referencia"))}</dt><dd><code>${escaparHTML(recibo.recibo_ref)}</code></dd></div>
      <div><dt>${escaparHTML(t("cobertura_recibo_decision"))}</dt><dd><code>${escaparHTML(recibo.decision_cobertura_ref)}</code></dd></div>
      <div><dt>${escaparHTML(t("cobertura_recibo_fecha"))}</dt><dd>${escaparHTML(formateador.format(new Date(recibo.confirmada_en)))}</dd></div>
    </dl>
  </section>`;
}

function renderizarPropuesta(propuesta, estado, t) {
  const evaluacion = propuesta.evaluaciones.find(
    ({ via_clave: via }) => via === propuesta.via_recomendada,
  );
  const etiquetaVia = propuesta.via_recomendada === "bolsa_vigente"
    ? t("cobertura_via_bolsa_vigente")
    : propuesta.via_recomendada;
  return `<section class="ct-bloque" data-ct-cobertura-propuesta
    aria-labelledby="ct-cobertura-propuesta-titulo">
    <h3 id="ct-cobertura-propuesta-titulo">${escaparHTML(t("cobertura_propuesta_titulo"))}</h3>
    <dl class="ct-resumen">
      <div><dt>${escaparHTML(t("cobertura_via_recomendada"))}</dt>
      <dd><strong data-ct-cobertura-via="${escaparHTML(propuesta.via_recomendada)}">${
  escaparHTML(etiquetaVia)}</strong></dd></div>
      <div><dt>${escaparHTML(t("cobertura_evaluacion"))}</dt><dd>${escaparHTML(
  t(`cobertura_evaluacion_${evaluacion?.estado ?? propuesta.estado}`),
)}</dd></div>
    </dl>
    <form data-ct-cobertura-form>
      <p>${escaparHTML(t("cobertura_confirmacion_ayuda"))}</p>
      <div class="ct-acciones"><button class="boton-primario" type="submit"${
  estado.ocupado ? " disabled" : ""}>${escaparHTML(t("cobertura_confirmar"))}</button></div>
    </form>
  </section>`;
}

function renderizarContenido(estado, contexto, t, formateador) {
  if (estado.recibo) return renderizarRecibo(estado.recibo, contexto, t, formateador);
  if (estado.indeterminado) {
    return `<section class="ct-alcance" data-ct-cobertura-indeterminado role="status"
      aria-live="assertive" aria-atomic="true" tabindex="-1">
      <h3>${escaparHTML(t("cobertura_indeterminado_titulo"))}</h3>
      <p>${escaparHTML(t("cobertura_indeterminado_descripcion"))}</p>
      <div class="ct-acciones"><button class="boton-secundario" type="button"
        data-ct-cobertura-accion="consultar-resultado"${estado.ocupado ? " disabled" : ""}>${
  escaparHTML(t("cobertura_consultar_resultado"))}</button></div>
    </section>`;
  }
  if (estado.propuesta?.estado === "viable") {
    return renderizarPropuesta(estado.propuesta, estado, t);
  }
  if (estado.propuesta) {
    return `<section class="ct-alcance" role="status">
      <h3>${escaparHTML(t("cobertura_sin_via_titulo"))}</h3>
      <p>${escaparHTML(t("cobertura_sin_via_descripcion"))}</p>
    </section>`;
  }
  return `<section class="ct-alcance" role="status">
    <p>${escaparHTML(t(estado.error ? "cobertura_error_descripcion" : "cobertura_cargando_descripcion"))}</p>
  </section>`;
}

export function montarFormularioCobertura(configuracion = {}) {
  if (!configuracionCerrada(configuracion)) {
    throw new TypeError("configuración del formulario de cobertura no válida");
  }
  let {
    raiz,
    cliente,
    contexto: contextoEntrada,
    generarClaveIdempotencia = () => globalThis.crypto?.randomUUID?.(),
    confirmarOperacion = () => false,
    alConfirmar = () => true,
    mensajes = {},
    locale = "es-ES",
    zonaHoraria = "Europe/Madrid",
    anunciar = () => {},
  } = configuracion;
  configuracion = null;
  if (!raiz || typeof raiz.addEventListener !== "function"
    || typeof raiz.removeEventListener !== "function"
    || typeof raiz.querySelector !== "function" || typeof raiz.contains !== "function"
    || typeof raiz.replaceChildren !== "function"
    || typeof cliente?.proponerCobertura !== "function"
    || typeof cliente?.decidirCobertura !== "function"
    || typeof cliente?.consultarResultadoCobertura !== "function"
    || typeof generarClaveIdempotencia !== "function"
    || typeof confirmarOperacion !== "function" || typeof alConfirmar !== "function"
    || typeof anunciar !== "function"
    || typeof AbortController !== "function") {
    throw new TypeError("dependencias del formulario de cobertura no válidas");
  }
  let contexto = normalizarContexto(contextoEntrada);
  let t = crearTraductorContratacionTemporal(mensajes);
  let formateador = new Intl.DateTimeFormat(locale, {
    dateStyle: "long", timeStyle: "medium", timeZone: zonaHoraria,
  });
  let raizActual = raiz;
  let clienteActual = cliente;
  let generarClaveActual = generarClaveIdempotencia;
  let confirmarActual = confirmarOperacion;
  let alConfirmarActual = alConfirmar;
  let anunciarActual = anunciar;
  let controlador = null;
  let vuelo = null;
  let claveIntento = "";
  let montado = true;
  let estado = {
    propuesta: null,
    recibo: null,
    ocupado: false,
    indeterminado: false,
    error: false,
    mensaje_clave: "cobertura_estado_cargando",
    tipo_mensaje: "informacion",
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

  function enlazarSiguientePaso(recibo) {
    try {
      if (alConfirmarActual(recibo) === true) return;
    } catch {
      // El aviso visible conserva el recibo confirmado sin filtrar el error.
    }
    estado = {
      ...estado,
      mensaje_clave: "cobertura_estado_asignacion_no_disponible",
      tipo_mensaje: "aviso",
    };
  }

  function repintar(selectorFoco = "") {
    if (!montado) return;
    raizActual.innerHTML = `<section class="ct-alta" data-ct-cobertura
      aria-labelledby="ct-cobertura-titulo">
      <header class="ct-cabecera"><div>
        <p class="sobrelinea">${escaparHTML(t("cobertura_sobrelinea"))}</p>
        <h2 id="ct-cobertura-titulo">${escaparHTML(t("cobertura_titulo"))}</h2>
        <p>${escaparHTML(t("cobertura_descripcion"))}</p>
      </div><aside class="ct-alcance" aria-label="${escaparHTML(t("cobertura_alcance_etiqueta"))}">${
  escaparHTML(t("cobertura_alcance"))}</aside></header>
      <div class="ct-estado ct-estado-${escaparHTML(estado.tipo_mensaje)}"
        data-ct-cobertura-estado role="status" aria-live="polite"
        aria-atomic="true" tabindex="-1"><strong>${escaparHTML(t(estado.mensaje_clave))}</strong></div>
      ${renderizarContenido(estado, contexto, t, formateador)}
    </section>`;
    if (selectorFoco) enfocar(selectorFoco);
    try { anunciarActual(t(estado.mensaje_clave), estado.tipo_mensaje); } catch {
      // La región viva sigue siendo la fuente visible y accesible del estado.
    }
  }

  function fijarError(clave = "cobertura_estado_error") {
    estado = {
      ...estado,
      ocupado: false,
      error: true,
      mensaje_clave: clave,
      tipo_mensaje: "error",
    };
  }

  function cargarPropuesta() {
    if (vuelo !== null || !montado) return vuelo ?? Promise.resolve(null);
    controlador = new AbortController();
    const solicitud = validarSolicitudPropuestaCobertura(contexto);
    const tarea = (async () => {
      try {
        const propuesta = validarPropuestaCobertura(await clienteActual.proponerCobertura(
          solicitud,
          Object.freeze({ signal: controlador.signal }),
        ));
        if (!montado) return null;
        estado = {
          ...estado,
          propuesta,
          error: false,
          mensaje_clave: propuesta.estado === "viable"
            ? "cobertura_estado_lista" : "cobertura_estado_sin_via",
          tipo_mensaje: propuesta.estado === "viable" ? "exito" : "aviso",
        };
        return propuesta;
      } catch {
        if (montado) fijarError("cobertura_estado_error_propuesta");
        return null;
      } finally {
        controlador = null;
        vuelo = null;
        if (montado) repintar("[data-ct-cobertura-estado]");
      }
    })();
    vuelo = tarea;
    return tarea;
  }

  function confirmarDecision() {
    if (!montado || vuelo !== null || estado.ocupado || estado.indeterminado
      || estado.recibo || estado.propuesta?.estado !== "viable") {
      return vuelo ?? Promise.resolve(null);
    }
    let confirmada = false;
    try {
      confirmada = confirmarActual({
        titulo: t("cobertura_confirmar"),
        advertencia: t("cobertura_confirmacion_advertencia"),
        referencia: contexto.expediente_ref,
      }) === true;
    } catch {
      confirmada = false;
    }
    if (!confirmada) return Promise.resolve(null);
    if (claveIntento === "") claveIntento = generarClaveActual();
    let solicitud;
    try {
      solicitud = validarSolicitudDecisionCobertura({
        expediente_ref: contexto.expediente_ref,
        version_esperada: contexto.version_esperada,
        clave_idempotencia: claveIntento,
        identidad_semantica: estado.propuesta.identidad_semantica,
        via_elegida: estado.propuesta.via_recomendada,
        motivo_clave: "",
      });
    } catch {
      fijarError();
      repintar("[data-ct-cobertura-estado]");
      return Promise.resolve(null);
    }
    controlador = new AbortController();
    estado = {
      ...estado,
      ocupado: true,
      error: false,
      mensaje_clave: "cobertura_estado_enviando",
      tipo_mensaje: "informacion",
    };
    repintar("[data-ct-cobertura-estado]");
    const tarea = (async () => {
      try {
        const recibo = validarReciboCobertura(await clienteActual.decidirCobertura(
          solicitud,
          Object.freeze({ signal: controlador.signal }),
        ));
        if (recibo.estado !== "aplicada"
          || recibo.version_resultante !== contexto.version_esperada + 1) {
          throw new TypeError("recibo de cobertura no ligado");
        }
        if (!montado) return null;
        estado = {
          ...estado,
          recibo,
          ocupado: false,
          indeterminado: false,
          mensaje_clave: "cobertura_estado_confirmada",
          tipo_mensaje: "exito",
        };
        claveIntento = "";
        enlazarSiguientePaso(recibo);
        return recibo;
      } catch (error) {
        if (!montado) return null;
        if (resultadoIndeterminado(error)) {
          estado = {
            ...estado,
            ocupado: false,
            indeterminado: true,
            mensaje_clave: "cobertura_estado_indeterminado",
            tipo_mensaje: "aviso",
          };
        } else {
          fijarError("cobertura_estado_rechazada");
        }
        return null;
      } finally {
        controlador = null;
        vuelo = null;
        if (montado) repintar(estado.recibo
          ? "[data-ct-cobertura-recibo]" : "[data-ct-cobertura-estado]");
      }
    })();
    vuelo = tarea;
    return tarea;
  }

  function consultarResultado() {
    if (!montado || vuelo !== null || !estado.indeterminado || claveIntento === "") {
      return vuelo ?? Promise.resolve(null);
    }
    const solicitud = validarSolicitudConsultaResultadoCobertura({
      expediente_ref: contexto.expediente_ref,
      clave_idempotencia: claveIntento,
    });
    controlador = new AbortController();
    estado = {
      ...estado,
      ocupado: true,
      mensaje_clave: "cobertura_estado_consultando",
      tipo_mensaje: "informacion",
    };
    repintar("[data-ct-cobertura-estado]");
    const tarea = (async () => {
      try {
        const resultado = validarResultadoConsultaCobertura(
          await clienteActual.consultarResultadoCobertura(
            solicitud,
            Object.freeze({ signal: controlador.signal }),
          ),
        );
        if (!montado) return null;
        if (resultado.estado === "confirmado") {
          const recibo = resultado.recibo;
          if (recibo.estado !== "aplicada"
            || recibo.version_resultante !== contexto.version_esperada + 1) {
            throw new TypeError("resultado de cobertura no ligado");
          }
          estado = {
            ...estado,
            recibo,
            ocupado: false,
            indeterminado: false,
            mensaje_clave: "cobertura_estado_confirmada",
            tipo_mensaje: "exito",
          };
          claveIntento = "";
          enlazarSiguientePaso(recibo);
          return recibo;
        }
        estado = {
          ...estado,
          ocupado: false,
          mensaje_clave: "cobertura_estado_no_observable",
          tipo_mensaje: "aviso",
        };
        return null;
      } catch {
        if (montado) {
          estado = {
            ...estado,
            ocupado: false,
            mensaje_clave: "cobertura_estado_consulta_error",
            tipo_mensaje: "aviso",
          };
        }
        return null;
      } finally {
        controlador = null;
        vuelo = null;
        if (montado) repintar(estado.recibo
          ? "[data-ct-cobertura-recibo]" : "[data-ct-cobertura-indeterminado]");
      }
    })();
    vuelo = tarea;
    return tarea;
  }

  function alEnviar(evento) {
    const formulario = evento.target?.closest?.("[data-ct-cobertura-form]");
    if (!formulario || !raizActual.contains(formulario)) return undefined;
    evento.preventDefault();
    return confirmarDecision();
  }

  function alPulsar(evento) {
    const control = evento.target?.closest?.("[data-ct-cobertura-accion]");
    if (!control || !raizActual.contains(control)
      || control.dataset.ctCoberturaAccion !== "consultar-resultado") return undefined;
    evento.preventDefault();
    return consultarResultado();
  }

  raizActual.addEventListener("submit", alEnviar);
  raizActual.addEventListener("click", alPulsar);
  repintar();
  void cargarPropuesta();

  return function desmontarFormularioCobertura() {
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
    t = null;
    formateador = null;
    raizActual = null;
  };
}
