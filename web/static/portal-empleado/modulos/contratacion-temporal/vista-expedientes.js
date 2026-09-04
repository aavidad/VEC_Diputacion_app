/** Vista y enlace DOM de la superficie de expedientes de contratación temporal. */

import {
  crearPresentadorAltaContratacionTemporal,
} from "./presentador.js";
import { validarReciboAlta } from "./contrato.js";
import { validarReciboAnalisis } from "./contrato-analisis.js";
import { montarFormularioAnalisisRRHH } from "./formulario-analisis.js";
import { montarAltaContratacionTemporal } from "./vista.js";
import {
  escaparHTML,
  renderizarAuditoria,
  renderizarCuadro,
  renderizarDocumentos,
  renderizarEstadoCarga,
  renderizarExpediente,
} from "./componentes-expedientes.js";
import { crearTraductorExpedientesContratacion } from "./i18n-expedientes.js";

const CAMPOS_COMPOSICION_ANALISIS = Object.freeze([
  "cliente", "catalogos", "contexto", "analisisInicial",
]);
const CAMPOS_CONTEXTO_ANALISIS = Object.freeze(["operacion", "artefacto_ref"]);
const PATRON_REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;

function descriptoresCerrados(entrada, campos, nombre) {
  if (entrada === null || typeof entrada !== "object" || Array.isArray(entrada)
    || Object.getPrototypeOf(entrada) !== Object.prototype) {
    throw new TypeError(`${nombre} no válida`);
  }
  const descriptores = Object.getOwnPropertyDescriptors(entrada);
  const claves = Object.keys(descriptores);
  if (Object.getOwnPropertySymbols(entrada).length !== 0
    || claves.some((clave) => !campos.includes(clave))
    || claves.some((clave) => !Object.hasOwn(descriptores[clave], "value")
      || descriptores[clave].enumerable !== true)) {
    throw new TypeError(`${nombre} no válida`);
  }
  if (campos.some((campo) => !Object.hasOwn(descriptores, campo))) return null;
  return descriptores;
}

function prepararComposicionAnalisis(entrada) {
  if (entrada === null || entrada === undefined) return null;
  const descriptores = descriptoresCerrados(
    entrada,
    CAMPOS_COMPOSICION_ANALISIS,
    "composición del análisis",
  );
  if (descriptores === null) return null;
  const contextoEntrada = descriptores.contexto.value;
  if (contextoEntrada === null || typeof contextoEntrada !== "object"
    || Array.isArray(contextoEntrada)) return null;
  const contextoDescriptores = descriptoresCerrados(
    contextoEntrada,
    CAMPOS_CONTEXTO_ANALISIS,
    "contexto de composición del análisis",
  );
  if (contextoDescriptores === null) return null;
  const operacion = contextoDescriptores.operacion.value;
  const artefactoRef = contextoDescriptores.artefacto_ref.value;
  const cliente = descriptores.cliente.value;
  const catalogos = descriptores.catalogos.value;
  const metodo = operacion === "rectificar" ? "rectificarAnalisis" : "registrarAnalisis";
  let metodoCliente;
  try { metodoCliente = cliente?.[metodo]; } catch { return null; }
  if (!(["registrar", "rectificar"].includes(operacion))
    || typeof artefactoRef !== "string" || !PATRON_REFERENCIA.test(artefactoRef)
    || cliente === null || typeof cliente !== "object" || typeof metodoCliente !== "function"
    || catalogos === null || typeof catalogos !== "object" || Array.isArray(catalogos)) {
    return null;
  }
  return Object.freeze({
    cliente,
    metodoCliente,
    nombreMetodo: metodo,
    catalogos,
    contexto: Object.freeze({ operacion, artefacto_ref: artefactoRef }),
    analisisInicial: descriptores.analisisInicial.value,
  });
}

function errorIndeterminadoAnalisis() {
  const error = new Error("resultado de análisis indeterminado");
  Object.defineProperties(error, {
    codigo: { value: "resultado_indeterminado", enumerable: true },
    resultadoIndeterminado: { value: true, enumerable: true },
  });
  return error;
}

function clasificarErrorAnalisis(error, signal) {
  let indeterminado;
  let abortado = false;
  try {
    indeterminado = error?.resultadoIndeterminado;
    abortado = signal?.aborted === true;
  } catch {
    return Object.freeze({ error: errorIndeterminadoAnalisis(), etapa: "indeterminado" });
  }
  if (indeterminado === false && !abortado) {
    return Object.freeze({ error, etapa: "reintentable" });
  }
  return Object.freeze({
    error: indeterminado === true && !abortado ? error : errorIndeterminadoAnalisis(),
    etapa: "indeterminado",
  });
}

function crearClienteAnalisisCercado(composicion, contexto, cambiarEtapa) {
  const invocar = function invocarAnalisis(solicitud, opciones) {
    const vuelo = Object.freeze({});
    cambiarEtapa("transmitiendo", vuelo);
    let resultado;
    try {
      resultado = Reflect.apply(
        composicion.metodoCliente,
        composicion.cliente,
        [solicitud, opciones],
      );
    } catch (error) {
      const clasificado = clasificarErrorAnalisis(error, opciones?.signal);
      cambiarEtapa(clasificado.etapa, vuelo);
      throw clasificado.error;
    }
    return Promise.resolve(resultado).then((respuesta) => {
      let recibo;
      try {
        recibo = validarReciboAnalisis(respuesta);
        if (recibo.operacion !== contexto.operacion
          || recibo.expediente_ref !== contexto.expediente_ref
          || recibo.version_resultante !== contexto.version_esperada + 1) {
          throw new TypeError("recibo no ligado");
        }
      } catch {
        cambiarEtapa("indeterminado", vuelo);
        throw errorIndeterminadoAnalisis();
      }
      cambiarEtapa("confirmado", vuelo);
      return recibo;
    }, (error) => {
      const clasificado = clasificarErrorAnalisis(error, opciones?.signal);
      cambiarEtapa(clasificado.etapa, vuelo);
      throw clasificado.error;
    });
  };
  return Object.freeze({ [composicion.nombreMetodo]: invocar });
}

function renderizarNavegacion(estado, t) {
  const opciones = [
    ["cuadro", "nav_cuadro"],
    ["alta", "nav_alta"],
    ["expediente", "nav_expediente"],
    ["documentos", "nav_documentos"],
    ["auditoria", "nav_auditoria"],
  ];
  return `<nav class="ct-exp-navegacion" aria-label="${escaparHTML(t("navegacion"))}">
    ${opciones.map(([vista, clave]) => {
    const requiereExpediente = ["expediente", "documentos", "auditoria"].includes(vista);
    return `<button type="button" data-ct-exp-vista="${vista}"
      ${estado.vista === vista ? 'aria-current="page"' : ""}
      ${requiereExpediente && !estado.expediente && !estado.cuadro?.expedientes.length ? "disabled" : ""}>
      ${escaparHTML(t(clave))}
    </button>`;
  }).join("")}
  </nav>`;
}

function renderizarCabeceraModulo(estado, t) {
  const demostracion = estado.cuadro?.demostracion === true
    || estado.expediente?.demostracion === true;
  return `<header class="ct-exp-cabecera-modulo">
    <div>
      <p class="sobrelinea">${escaparHTML(t("sobrelinea"))}</p>
      <h2>${escaparHTML(t("titulo"))}</h2>
      <p>${escaparHTML(t("descripcion"))}</p>
    </div>
    ${demostracion ? `<p class="ct-exp-aviso-presentacion" role="note">${escaparHTML(t("presentacion"))}</p>` : ""}
  </header>`;
}

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

function renderizarAlta(t, disponible, analisisDisponible) {
  return `<header class="ct-exp-subcabecera">
    <h3>${escaparHTML(t("nueva_peticion_titulo"))}</h3>
    <p>${escaparHTML(t("nueva_peticion_descripcion"))}</p>
  </header>
  ${disponible
    ? `<div data-ct-exp-alta></div>
      ${analisisDisponible ? '<div data-ct-exp-analisis></div>' : ""}`
    : `<section class="ct-exp-estado-global ct-tono-peligro" role="alert">
      <h3>${escaparHTML(t("denegado_titulo"))}</h3>
      <p>${escaparHTML(t("estado_denegado"))}</p>
    </section>`}`;
}

export function renderizarModuloContratacionTemporal(estado, {
  mensajes = {},
  locale = "es-ES",
  zonaHoraria = "Europe/Madrid",
  altaDisponible = false,
  analisisDisponible = false,
} = {}) {
  const t = crearTraductorExpedientesContratacion(mensajes);
  let contenido;
  if (["cargando", "error", "denegado"].includes(estado.carga)
    && estado.vista !== "alta") {
    contenido = renderizarEstadoCarga(estado, t);
  } else if (estado.vista === "alta") {
    contenido = renderizarAlta(t, altaDisponible, analisisDisponible);
  } else if (estado.vista === "expediente") {
    contenido = renderizarExpediente(
      estado,
      t,
      locale,
      zonaHoraria,
      analisisDisponible,
    );
  } else if (estado.vista === "documentos") {
    contenido = renderizarDocumentos(estado, t);
  } else if (estado.vista === "auditoria") {
    contenido = renderizarAuditoria(estado, t);
  } else {
    contenido = renderizarCuadro(estado, t);
  }
  return `<section class="ct-expedientes" data-modulo="contratacion-temporal"
    aria-labelledby="ct-exp-titulo">
    ${renderizarCabeceraModulo(estado, t)
    .replace("<h2>", '<h2 id="ct-exp-titulo">')}
    ${renderizarNavegacion(estado, t)}
    <div class="ct-exp-mensaje ct-tono-${escaparHTML(estado.tipo_mensaje)}"
      data-ct-exp-mensaje role="${estado.tipo_mensaje === "error" ? "alert" : "status"}"
      aria-live="polite">${escaparHTML(t(estado.mensaje_clave))}</div>
    <div class="ct-exp-contenido">${contenido}</div>
  </section>`;
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
  const altaDisponible = alta !== null && typeof alta === "object"
    && typeof alta.ejecutor === "function" && alta.catalogos !== undefined;
  const composicionAnalisis = prepararComposicionAnalisis(analisis);
  let montada = true;
  let desmontarAlta = null;
  let desmontarAnalisis = null;
  let sesionAnalisis = null;

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
    if (!montada || sesionAnalisis !== sesion) return;
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
    enfocar(raiz, "[data-ct-analisis-estado]");
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

  function retirarComponentes() {
    retirarAlta();
    retirarAnalisis();
  }

  function montarAltaSiProcede() {
    const estado = presentador.obtenerEstado();
    if (!montada || estado.vista !== "alta" || !altaDisponible) return;
    const contenedor = raiz.querySelector("[data-ct-exp-alta]");
    if (!contenedor) return;
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
  }

  function montarAnalisisEnContenedor(contenedor, contexto, analisisInicial) {
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
    );
    sesionAnalisis = sesion;
    try {
      desmontarAnalisis = montarFormularioAnalisisRRHH({
        raiz: contenedor,
        cliente: clienteCercado,
        contexto,
        catalogos: composicionAnalisis.catalogos,
        analisisInicial,
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
      return false;
    }
  }

  function montarAnalisisDesdeAlta(recibo) {
    const estado = presentador.obtenerEstado();
    if (!montada || estado.vista !== "alta" || composicionAnalisis === null
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
    if (!montada || estado.vista !== "expediente" || composicionAnalisis === null) return null;
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

  function repintar(selectorFoco = "") {
    if (!montada || analisisEstableActivo()) return;
    retirarComponentes();
    const estado = presentador.obtenerEstado();
    raiz.innerHTML = renderizarModuloContratacionTemporal(estado, {
      mensajes,
      locale,
      zonaHoraria,
      altaDisponible,
      analisisDisponible: composicionAnalisis !== null,
    });
    montarAltaSiProcede();
    if (montarAnalisisSiProcede() === false) {
      retirarComponentes();
      raiz.innerHTML = renderizarModuloContratacionTemporal(estado, {
        mensajes,
        locale,
        zonaHoraria,
        altaDisponible,
        analisisDisponible: false,
      });
    }
    if (selectorFoco) enfocar(raiz, selectorFoco);
    anunciar(
      crearTraductorExpedientesContratacion(mensajes)(estado.mensaje_clave),
      estado.tipo_mensaje,
    );
  }

  async function cambiarVista(vista) {
    if (impedirCambioPorAnalisis()) return;
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
    repintar(vista === "cuadro" ? "[data-ct-exp-filtros]" : ".ct-exp-contenido");
  }

  async function manejarClick(evento) {
    const controlVista = evento.target?.closest?.("[data-ct-exp-vista]");
    if (controlVista && raiz.contains(controlVista)) {
      evento.preventDefault();
      await cambiarVista(controlVista.dataset.ctExpVista);
      return;
    }
    const abrir = evento.target?.closest?.("[data-ct-exp-abrir]");
    if (abrir && raiz.contains(abrir)) {
      evento.preventDefault();
      if (impedirCambioPorAnalisis()) return;
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
    const tareaControl = evento.target?.closest?.("[data-ct-exp-tarea]");
    if (tareaControl && raiz.contains(tareaControl)) {
      evento.preventDefault();
      if (impedirCambioPorAnalisis() || presentador.obtenerEstado().ocupado) return;
      presentador.seleccionarTarea(tareaControl.dataset.ctExpTarea);
      repintar("#ct-exp-tarea-titulo");
      return;
    }
    const efecto = evento.target?.closest?.("[data-ct-exp-efecto]");
    if (efecto && raiz.contains(efecto)) {
      evento.preventDefault();
      if (impedirCambioPorAnalisis()) return;
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
    if (impedirCambioPorAnalisis()) return;
    if (presentador.obtenerEstado().ocupado
      && accion.dataset.ctExpAccion !== "cancelar") return;
    if (accion.dataset.ctExpAccion === "limpiar-filtros") {
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
    if (impedirCambioPorAnalisis() || presentador.obtenerEstado().ocupado) return;
    const datos = new FormData(formulario);
    const promesa = presentador.cargar({
      texto: String(datos.get("texto") ?? "").trim(),
      estado: String(datos.get("estado") ?? ""),
      fase: String(datos.get("fase") ?? ""),
    });
    repintar("[data-ct-exp-mensaje]");
    await promesa;
    repintar("[data-ct-exp-filtros]");
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
      retirarComponentes();
      raiz.removeEventListener("click", manejarClick);
      raiz.removeEventListener("submit", manejarEnvio);
      presentador.desmontar?.();
    },
  });
}
