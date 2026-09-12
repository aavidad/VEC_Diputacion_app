/** Vista y enlace DOM de la superficie de expedientes de contratación temporal. */

import {
  crearPresentadorAltaContratacionTemporal,
} from "./presentador.js";
import { validarReciboAlta } from "./contrato.js";
import { validarReciboAnalisis } from "./contrato-analisis.js";
import { montarFormularioAnalisisRRHH } from "./formulario-analisis.js";
import { montarFormularioCobertura } from "./formulario-cobertura.js";
import { montarFormularioAsignacion } from "./formulario-asignacion.js";
import { montarFormularioInformeJuridico } from "./formulario-informe-juridico.js";
import { montarFormularioFiscalizacion } from "./formulario-fiscalizacion.js";
import { montarFormularioSubsanacionReparos } from "./formulario-subsanacion-reparos.js";
import { montarFormularioLlamamiento } from "./formulario-llamamiento.js";
import { montarFormularioResolucionFormalizacion } from "./formulario-resolucion-formalizacion.js";
import { montarFormularioIncorporacionEjercicio } from "./formulario-incorporacion-ejercicio.js";
import { montarFichaGINPIX } from "./ficha-ginpix.js";
import { montarSeguimientoIncorporacion } from "./seguimiento-incorporacion.js";
import { montarFormularioAnotacionAdministrativa } from "./formulario-anotacion-administrativa.js";
import { montarFormularioCierreAdministrativo } from "./formulario-cierre-administrativo.js";
import { crearClienteHTTPBorradorRRHH, PERFILES_BORRADOR_RRHH } from "./cliente-http-informe-definitivo.js";
import { montarAltaContratacionTemporal } from "./vista.js";
import {
  escaparHTML,
  renderizarAuditoria,
  renderizarCuadro,
  renderizarDocumentos,
  renderizarEstadoCarga,
  renderizarExpediente,
  solicitudInformeDefinitivoDesdeEstado,
} from "./componentes-expedientes.js";
import { crearTraductorExpedientesContratacion } from "./i18n-expedientes.js";
import { crearTraductorContratacionTemporal } from "./i18n.js";

const CAMPOS_COMPOSICION_ANALISIS = Object.freeze([
  "cliente", "catalogos", "contexto", "analisisInicial", "rectificacion",
]);
const CAMPOS_COMPOSICION_ANALISIS_OBLIGATORIOS = Object.freeze([
  "cliente", "catalogos", "contexto", "analisisInicial",
]);
const CAMPOS_CONTEXTO_ANALISIS = Object.freeze(["operacion", "artefacto_ref"]);
const CAMPOS_RECTIFICACION_ANALISIS = Object.freeze([
  "operacion", "artefacto_ref", "analisisInicial",
]);
const PATRON_REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;

function descriptoresCerrados(entrada, campos, nombre, obligatorios = campos) {
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
  if (obligatorios.some((campo) => !Object.hasOwn(descriptores, campo))) return null;
  return descriptores;
}

function prepararComposicionAnalisis(entrada) {
  if (entrada === null || entrada === undefined) return null;
  const descriptores = descriptoresCerrados(
    entrada,
    CAMPOS_COMPOSICION_ANALISIS,
    "composición del análisis",
    CAMPOS_COMPOSICION_ANALISIS_OBLIGATORIOS,
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
  let rectificacion = null;
  if (Object.hasOwn(descriptores, "rectificacion")) {
    const entradaRectificacion = descriptores.rectificacion.value;
    const descriptoresRectificacion = descriptoresCerrados(
      entradaRectificacion,
      CAMPOS_RECTIFICACION_ANALISIS,
      "composición de rectificación del análisis",
    );
    if (descriptoresRectificacion === null
      || descriptoresRectificacion.operacion.value !== "rectificar"
      || descriptoresRectificacion.artefacto_ref.value !== artefactoRef
      || descriptoresRectificacion.analisisInicial.value !== null) return null;
    let metodoRectificacion;
    try { metodoRectificacion = cliente?.rectificarAnalisis; } catch { return null; }
    if (typeof metodoRectificacion !== "function") return null;
    rectificacion = Object.freeze({
      operacion: "rectificar",
      artefacto_ref: artefactoRef,
      analisisInicial: null,
      metodoCliente: metodoRectificacion,
    });
  }
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
    rectificacion,
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

function crearClienteAnalisisCercado(
  composicion,
  contexto,
  cambiarEtapa,
  alConfirmar,
  alErrorConfirmado = () => {},
) {
  const nombreMetodo = contexto.operacion === "rectificar"
    ? "rectificarAnalisis" : "registrarAnalisis";
  const metodoCliente = contexto.operacion === composicion.contexto.operacion
    ? composicion.metodoCliente
    : composicion.rectificacion?.metodoCliente;
  if (typeof metodoCliente !== "function") {
    throw new TypeError("método de análisis no disponible");
  }
  const invocar = function invocarAnalisis(solicitud, opciones) {
    const vuelo = Object.freeze({});
    cambiarEtapa("transmitiendo", vuelo);
    let resultado;
    try {
      resultado = Reflect.apply(
        metodoCliente,
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
      try { alConfirmar(recibo); } catch {
        // El recibo de Análisis prevalece si el siguiente paso no puede montarse.
        alErrorConfirmado(recibo);
      }
      return recibo;
    }, (error) => {
      const clasificado = clasificarErrorAnalisis(error, opciones?.signal);
      cambiarEtapa(clasificado.etapa, vuelo);
      throw clasificado.error;
    });
  };
  return Object.freeze({ [nombreMetodo]: invocar });
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

function renderizarAlta(
  t,
  disponible,
  analisisDisponible,
  coberturaDisponible,
  asignacionDisponible,
  informeJuridicoDisponible,
  fiscalizacionDisponible,
) {
  return `<header class="ct-exp-subcabecera">
    <h3>${escaparHTML(t("nueva_peticion_titulo"))}</h3>
    <p>${escaparHTML(t("nueva_peticion_descripcion"))}</p>
  </header>
  ${disponible
    ? `<div data-ct-exp-alta></div>
      ${analisisDisponible ? '<div data-ct-exp-analisis></div>' : ""}
      ${coberturaDisponible ? '<div data-ct-exp-cobertura></div>' : ""}
      ${asignacionDisponible ? '<div data-ct-exp-asignacion></div>' : ""}
      ${informeJuridicoDisponible ? '<div data-ct-exp-informe-juridico></div>' : ""}
      ${fiscalizacionDisponible ? '<div data-ct-exp-fiscalizacion></div>' : ""}`
    : `<section class="ct-exp-estado-global ct-tono-peligro" role="alert">
      <h3>${escaparHTML(t("denegado_titulo"))}</h3>
      <p>${escaparHTML(t("estado_denegado"))}</p>
    </section>`}`;
}

function contextoLlamamientoDesdeEstado(estado) {
  const expediente = estado?.expediente;
  if (estado?.vista !== "expediente" || estado.carga !== "listo"
    || estado.ocupado || estado.actualizacion_pendiente || estado.resultado_indeterminado
    || expediente?.demostracion !== false || estado.cuadro?.demostracion !== false
    || estado.expediente_ref !== expediente.expediente_ref
    || !Array.isArray(estado.cuadro.expedientes)) return null;
  const resumen = estado.cuadro.expedientes.find(({ expediente_ref: referencia }) => (
    referencia === expediente.expediente_ref
  ));
  if (resumen?.fase_clave !== "fiscalizacion"
    || resumen.version !== expediente.version) return null;
  // Es contexto del formulario; el servidor decide vigencia y permisos al enviar.
  return Object.freeze({
    expediente_ref: expediente.expediente_ref,
    version_esperada: expediente.version,
  });
}

function contextoFiscalizacionDesdeEstado(estado) {
  if (estado?.vista !== "expediente" || estado.expediente === null
    || estado.cuadro === null || !Array.isArray(estado.cuadro.expedientes)) return null;
  const resumen = estado.cuadro.expedientes.find(({ expediente_ref: referencia }) => (
    referencia === estado.expediente.expediente_ref
  ));
  if (resumen?.version !== estado.expediente.version) return null;
  const esInformeInicial = resumen.fase_clave === "informe_juridico";
  const ultimoHito = estado.expediente.historial?.at?.(-1);
  const esSubsanacionAutorizada = resumen.fase_clave === "subsanacion_unidad"
    && resumen.estado_clave === "incidencia"
    && ultimoHito?.accion_clave === "contratacion_temporal.subsanacion_reparos.registrar"
    && ultimoHito.version_expediente === estado.expediente.version;
  if (!esInformeInicial && !esSubsanacionAutorizada) return null;
  const informe = estado.expediente.cabecera?.find(
    ({ clave }) => clave === "informe_ref",
  )?.valor;
  return Object.freeze({
    expediente_ref: estado.expediente.expediente_ref,
    version_esperada: estado.expediente.version,
    fase_clave: esSubsanacionAutorizada ? "subsanacion_unidad" : resumen.fase_clave,
    informe_ref: !esSubsanacionAutorizada
      && typeof informe === "string" && PATRON_REFERENCIA.test(informe)
      ? informe : "",
  });
}

// La fase y el reparo proyectado solo acotan el contenedor. La disponibilidad
// efectiva llega como dependencia de composición y el servidor la revalida al
// registrar; la vista nunca la deduce de este estado.
function contextoSubsanacionDesdeEstado(estado) {
  if (estado?.vista !== "expediente" || estado.carga !== "listo"
    || estado.expediente?.demostracion !== false || estado.cuadro?.demostracion !== false
    || !Array.isArray(estado.cuadro?.expedientes)) return null;
  const resumen = estado.cuadro.expedientes.find(({ expediente_ref: referencia }) => (
    referencia === estado.expediente.expediente_ref
  ));
  if (resumen?.fase_clave !== "subsanacion_unidad" || resumen.estado_clave !== "incidencia"
    || resumen.version !== estado.expediente.version) return null;
  return Object.freeze({ expediente_ref: estado.expediente.expediente_ref,
    version_esperada: estado.expediente.version });
}

function contextoInformeJuridicoDesdeEstado(estado) {
  if (estado?.vista !== "expediente" || estado.expediente === null
    || estado.cuadro === null || !Array.isArray(estado.cuadro.expedientes)) return null;
  const resumen = estado.cuadro.expedientes.find(({ expediente_ref: referencia }) => (
    referencia === estado.expediente.expediente_ref
  ));
  if (resumen?.fase_clave !== "asignacion_unidad"
    || resumen.version !== estado.expediente.version) return null;
  return Object.freeze({
    expediente_ref: estado.expediente.expediente_ref,
    version_esperada: estado.expediente.version,
  });
}

function contextoAsignacionDesdeEstado(estado) {
  if (estado?.vista !== "expediente" || estado.carga !== "listo" || estado.expediente == null
    || estado.cuadro == null || !Array.isArray(estado.cuadro.expedientes)) return null;
  const resumen = estado.cuadro.expedientes.find(({ expediente_ref: referencia }) => (
    referencia === estado.expediente.expediente_ref
  ));
  if (resumen?.fase_clave !== "asignacion_unidad" || resumen.estado_clave !== "en_curso"
    || resumen.version !== estado.expediente.version) return null;
  return Object.freeze({
    expediente_ref: estado.expediente.expediente_ref,
    version_esperada: estado.expediente.version,
  });
}

function contextoCoberturaDesdeEstado(estado) {
  if (estado?.vista !== "expediente" || estado.carga !== "listo" || estado.expediente == null
    || estado.cuadro?.demostracion !== false || estado.expediente.demostracion !== false
    || !Array.isArray(estado.cuadro.expedientes) || !Array.isArray(estado.expediente.cabecera)) return null;
  const resumen = estado.cuadro.expedientes.find(({ expediente_ref: referencia }) => (
    referencia === estado.expediente.expediente_ref
  ));
  const analisisConfirmado = estado.expediente.cabecera.some(({ clave, valor }) => (
    clave === "resultado_rc" && typeof valor === "string" && valor !== ""
  ));
  const coberturaOAsignacionExistente = estado.expediente.cabecera.some(({ clave }) => (
    clave === "via_cobertura" || clave === "decision_gobernada" || clave === "unidad"
  ));
  if (resumen?.fase_clave !== "solicitud" || resumen.estado_clave !== "en_curso"
    || resumen.version !== estado.expediente.version || resumen.version < 2
    || !analisisConfirmado || coberturaOAsignacionExistente) return null;
  return Object.freeze({
    expediente_ref: estado.expediente.expediente_ref,
    version_esperada: estado.expediente.version,
  });
}

function contextoRectificacionAnalisisDesdeEstado(estado) {
  if (estado?.vista !== "expediente" || estado.carga !== "listo" || estado.expediente == null
    || estado.cuadro?.demostracion !== false || !Array.isArray(estado.cuadro.expedientes)
    || estado.expediente.demostracion !== false || estado.expediente.version < 2
    || !Array.isArray(estado.expediente.cabecera)) return null;
  const resumen = estado.cuadro.expedientes.find(({ expediente_ref: referencia }) => (
    referencia === estado.expediente.expediente_ref
  ));
  if (resumen?.estado_clave !== "en_curso" || resumen.version !== estado.expediente.version) {
    return null;
  }
  const claves = new Set(estado.expediente.cabecera.map(({ clave }) => clave));
  if (!claves.has("resultado_rc") || claves.has("via_cobertura")
    || claves.has("decision_gobernada") || claves.has("unidad")) return null;
  return Object.freeze({
    operacion: "rectificar",
    expediente_ref: estado.expediente.expediente_ref,
    version_esperada: estado.expediente.version,
  });
}

export function renderizarModuloContratacionTemporal(estado, {
  mensajes = {},
  locale = "es-ES",
  zonaHoraria = "Europe/Madrid",
  altaDisponible = false,
  analisisDisponible = false,
  coberturaDisponible = false,
  asignacionDisponible = false,
  informeJuridicoDisponible = false,
  fiscalizacionDisponible = false,
  subsanacionDisponible = false,
  llamamientoDisponible = false,
  resolucionFormalizacionDisponible = false,
  incorporacionEjercicioDisponible = false,
} = {}) {
  const t = crearTraductorExpedientesContratacion(mensajes);
  let contenido;
  const errorPaginadoRecuperable = estado.vista === "cuadro" && estado.carga === "error"
    && estado.cuadro?.paginacion && estado.paginacion_requiere_reinicio === true;
  if (["cargando", "error", "denegado"].includes(estado.carga)
    && !errorPaginadoRecuperable
    && estado.vista !== "alta") {
    contenido = renderizarEstadoCarga(estado, t);
  } else if (estado.vista === "alta") {
    contenido = renderizarAlta(
      t,
      altaDisponible,
      analisisDisponible,
      coberturaDisponible,
      asignacionDisponible,
      informeJuridicoDisponible,
      fiscalizacionDisponible,
    );
  } else if (estado.vista === "expediente") {
    const detalle = renderizarExpediente(
      estado,
      t,
      locale,
      zonaHoraria,
      analisisDisponible,
    );
    const contextoInforme = informeJuridicoDisponible
      ? contextoInformeJuridicoDesdeEstado(estado)
      : null;
    const contextoAsignacion = asignacionDisponible
      ? contextoAsignacionDesdeEstado(estado)
      : null;
    const contextoCobertura = coberturaDisponible
      ? contextoCoberturaDesdeEstado(estado)
      : null;
    const contextoRectificacion = analisisDisponible
      ? contextoRectificacionAnalisisDesdeEstado(estado)
      : null;
    const contextoFiscalizacion = fiscalizacionDisponible
      ? contextoFiscalizacionDesdeEstado(estado)
      : null;
    const contextoSubsanacion = subsanacionDisponible
      ? contextoSubsanacionDesdeEstado(estado)
      : null;
    contenido = `${detalle}${contextoRectificacion
      ? '<div data-ct-exp-rectificacion></div>'
      : ""}${contextoCobertura
      ? '<div data-ct-exp-cobertura></div>'
      : ""}${contextoAsignacion
      ? '<div data-ct-exp-asignacion></div>'
      : ""}${contextoInforme
      ? '<div data-ct-exp-informe-juridico></div>'
      : ""}${fiscalizacionDisponible && (contextoInforme || contextoFiscalizacion)
      ? '<div data-ct-exp-fiscalizacion></div>'
      : ""}${contextoSubsanacion ? '<div data-ct-exp-subsanacion></div>' : ""}${resolucionFormalizacionDisponible ? '<div data-ct-exp-resolucion-formalizacion></div>' : ""}
      ${incorporacionEjercicioDisponible ? '<div data-ct-exp-incorporacion-ejercicio></div>' : ""}`;
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
    <div class="ct-exp-contenido">${contenido}${llamamientoDisponible
      && ((estado.vista === "alta" && altaDisponible)
        || (estado.vista === "expediente" && estado.carga === "listo"
          && estado.expediente !== null))
      ? '<div data-ct-exp-llamamiento></div>' : ""}</div>
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
  fiscalizacion = null,
  subsanacion = null,
  llamamiento = null,
  clienteBorradorRRHH = crearClienteHTTPBorradorRRHH(),
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
  const clienteLlamamiento = llamamiento?.cliente ?? clienteFiscalizacion
    ?? composicionAnalisis?.cliente;
  const llamamientoDisponible = typeof clienteLlamamiento?.seleccionarLlamamiento === "function"
    && typeof clienteLlamamiento?.registrarComunicacionLlamamiento === "function";
  const resolucionFormalizacionDisponible = typeof clienteLlamamiento?.registrarResolucionFormalizacion === "function"
    && typeof clienteLlamamiento?.prepararResolucionFormalizacion === "function";
  const incorporacionEjercicioDisponible = typeof clienteLlamamiento?.prepararIncorporacionEjercicio === "function"
    && typeof clienteLlamamiento?.confirmarIncorporacionEjercicio === "function";
  const tExpedientes = crearTraductorExpedientesContratacion(mensajes);
  let desmontarIncorporacionEjercicio = null;
  let consultaIncorporacionEjercicio = null;
  let desmontarLlamamiento = null;
  let desmontarResolucionFormalizacion = null;
  let consultaResolucionFormalizacion = null;
  let montada = true;
  let desmontarAlta = null;
  let desmontarAnalisis = null;
  let desmontarCobertura = null;
  let desmontarAsignacion = null;
  let desmontarInformeJuridico = null;
  let desmontarFiscalizacion = null;
  let desmontarSubsanacion = null;
  let reciboSubsanacionConfirmado = null;
  let sesionAnalisis = null;
  let descargaInforme = null;
  let urlInforme = null;
  let revocacionInforme = null;
  const desmontarEstadosMontaje = new Set();
  const estadosMontajePorContenedor = new Map();

  function mostrarErrorMontaje(contenedor, etapa, reintentar) {
    if (!montada || !contenedor || typeof reintentar !== "function") return;
    const existentes = estadosMontajePorContenedor.get(contenedor);
    if (existentes?.has(etapa)) return;
    const documento = contenedor.ownerDocument ?? raiz.ownerDocument;
    if (!documento?.createElement || typeof contenedor.append !== "function") {
      anunciar(tExpedientes("montaje_siguiente_pendiente"), "error");
      return;
    }
    const bloque = documento.createElement("section");
    const boton = documento.createElement("button");
    const limpiar = () => {
      boton.removeEventListener?.("click", manejarReintento);
      bloque.remove?.();
      desmontarEstadosMontaje.delete(limpiar);
      const restantes = estadosMontajePorContenedor.get(contenedor);
      restantes?.delete(etapa);
      if (restantes?.size === 0) estadosMontajePorContenedor.delete(contenedor);
    };
    const manejarReintento = () => {
      if (!montada) return;
      limpiar();
      try {
        if (!reintentar()) mostrarErrorMontaje(contenedor, etapa, reintentar);
      } catch {
        mostrarErrorMontaje(contenedor, etapa, reintentar);
      }
    };
    bloque.setAttribute?.("class", "ct-estado ct-estado-aviso");
    bloque.setAttribute?.("role", "alert");
    bloque.setAttribute?.("aria-live", "assertive");
    bloque.setAttribute?.("data-ct-exp-montaje-pendiente", etapa);
    const titulo = documento.createElement("p");
    titulo.textContent = tExpedientes("montaje_siguiente_pendiente");
    const detalle = documento.createElement("p");
    detalle.textContent = tExpedientes("montaje_siguiente_pendiente_detalle");
    boton.type = "button";
    boton.className = "boton-secundario";
    boton.textContent = tExpedientes("montaje_siguiente_reintentar");
    boton.setAttribute?.("data-ct-exp-reintentar-montaje", etapa);
    boton.addEventListener?.("click", manejarReintento);
    bloque.append(titulo, detalle, boton);
    contenedor.append(bloque);
    desmontarEstadosMontaje.add(limpiar);
    const porEtapa = estadosMontajePorContenedor.get(contenedor) ?? new Map();
    porEtapa.set(etapa, limpiar);
    estadosMontajePorContenedor.set(contenedor, porEtapa);
    anunciar(titulo.textContent, "error");
  }

  function limpiarEstadosMontaje() {
    for (const desmontar of [...desmontarEstadosMontaje]) desmontar();
  }

  function liberarURLInforme() {
    clearTimeout(revocacionInforme);
    revocacionInforme = null;
    if (urlInforme !== null) entornoDescarga.URL.revokeObjectURL(urlInforme);
    urlInforme = null;
  }

  function cancelarDescargaInforme() {
    descargaInforme?.abort();
    descargaInforme = null;
    liberarURLInforme();
  }

  function informarDescarga(clave, tipo) {
    const texto = crearTraductorExpedientesContratacion(mensajes)(clave);
    const mensaje = raiz.querySelector("[data-ct-exp-mensaje]");
    if (mensaje) {
      mensaje.textContent = texto;
      mensaje.setAttribute("role", tipo === "error" ? "alert" : "status");
    }
    anunciar(texto, tipo);
  }

  async function descargarBorrador(boton) {
    const solicitud = solicitudInformeDefinitivoDesdeEstado(presentador.obtenerEstado());
    if (!montada || descargaInforme || !solicitud) return;
    const accionDocumento = boton.dataset.ctExpAccion.replace("descargar-docx-", "descargar-");
    const formato = boton.dataset.ctExpAccion.startsWith("descargar-docx-") ? "docx" : "pdf";
    const tipo = accionDocumento === "descargar-resolucion" ? "resolucion"
      : accionDocumento === "descargar-diligencia" ? "diligencia"
        : accionDocumento === "descargar-toma-posesion" ? "toma_posesion"
          : accionDocumento === "descargar-notificacion" ? "notificacion"
            : accionDocumento === "descargar-comunicacion-centro" ? "comunicacion_centro" : "informe_definitivo";
    const botones = typeof raiz.querySelectorAll === "function" ? [...raiz.querySelectorAll(
      '[data-ct-exp-accion="descargar-informe-definitivo"], [data-ct-exp-accion="descargar-resolucion"], [data-ct-exp-accion="descargar-diligencia"], [data-ct-exp-accion="descargar-toma-posesion"], [data-ct-exp-accion="descargar-notificacion"], [data-ct-exp-accion="descargar-comunicacion-centro"], [data-ct-exp-accion="descargar-docx-informe-definitivo"], [data-ct-exp-accion="descargar-docx-resolucion"], [data-ct-exp-accion="descargar-docx-diligencia"], [data-ct-exp-accion="descargar-docx-toma-posesion"], [data-ct-exp-accion="descargar-docx-notificacion"], [data-ct-exp-accion="descargar-docx-comunicacion-centro"]',
    )] : [boton];
    const controlador = new AbortController();
    descargaInforme = controlador;
    botones.forEach((control) => { control.disabled = true; });
    informarDescarga("informe_definitivo_descargando", "informacion");
    try {
      const { document: documento, URL: urls } = entornoDescarga;
      if (!documento?.body || typeof urls?.createObjectURL !== "function"
        || typeof urls?.revokeObjectURL !== "function") throw new TypeError();
      const blob = await clienteBorradorRRHH.descargarBorrador(solicitud, {
      tipo, formato, signal: controlador.signal,
      });
      if (!montada || descargaInforme !== controlador || controlador.signal.aborted
        || JSON.stringify(solicitudInformeDefinitivoDesdeEstado(presentador.obtenerEstado()))
          !== JSON.stringify(solicitud)) return;
      liberarURLInforme();
      urlInforme = urls.createObjectURL(blob);
      const enlace = documento.createElement("a");
      try {
        enlace.href = urlInforme;
        enlace.download = formato === "pdf" ? PERFILES_BORRADOR_RRHH[tipo].nombre
          : PERFILES_BORRADOR_RRHH[tipo].nombre.replace(/\.pdf$/u, ".docx");
        enlace.hidden = true;
        documento.body.append(enlace);
        enlace.click();
      } finally {
        enlace.remove();
        revocacionInforme = setTimeout(liberarURLInforme, 0);
      }
      informarDescarga("informe_definitivo_listo", "informacion");
    } catch (error) {
      if (!montada || descargaInforme !== controlador || controlador.signal.aborted) return;
      const clave = error?.envelopeValido === true && error.codigo === "documento_no_disponible"
        ? "informe_definitivo_no_disponible"
        : error?.envelopeValido === true && ["acceso_denegado", "autenticacion_requerida"].includes(error.codigo)
          ? "informe_definitivo_denegado" : "informe_definitivo_error";
      informarDescarga(clave, "error");
    } finally {
      if (descargaInforme === controlador) descargaInforme = null;
      botones.forEach((control) => { control.disabled = false; });
    }
  }

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

  function retirarCobertura() {
    if (typeof desmontarCobertura === "function") desmontarCobertura();
    desmontarCobertura = null;
  }

  function retirarAsignacion() {
    consultaResolucionFormalizacion?.abort();
    consultaResolucionFormalizacion = null;
    desmontarLlamamiento?.();
    desmontarLlamamiento = null;
    desmontarResolucionFormalizacion?.();
    desmontarResolucionFormalizacion = null;
    if (typeof desmontarFiscalizacion === "function") desmontarFiscalizacion();
    desmontarFiscalizacion = null;
    desmontarSubsanacion?.();
    desmontarSubsanacion = null;
    if (typeof desmontarInformeJuridico === "function") desmontarInformeJuridico();
    desmontarInformeJuridico = null;
    if (typeof desmontarAsignacion === "function") desmontarAsignacion();
    desmontarAsignacion = null;
  }

  function montarFiscalizacion(contexto) {
    if (!montada || !fiscalizacionDisponible || desmontarFiscalizacion !== null) {
      return desmontarFiscalizacion !== null;
    }
    const contenedor = raiz.querySelector("[data-ct-exp-fiscalizacion]");
    if (!contenedor) return false;
    try {
      desmontarFiscalizacion = montarFormularioFiscalizacion({
        raiz: contenedor,
        cliente: clienteFiscalizacion,
        contexto,
        confirmarOperacion,
        mensajes,
        locale,
        zonaHoraria,
        anunciar,
        alConfirmar: (recibo) => {
          if (recibo.resultado !== "desfavorable" && recibo.version_resultante >= 6) {
            montarLlamamiento({
              expediente_ref: recibo.expediente_ref,
              version_esperada: recibo.version_resultante,
            });
          }
        },
      });
      return true;
    } catch {
      desmontarFiscalizacion = null;
      return false;
    }
  }

  function montarFiscalizacionDesdeInforme(recibo) {
    return montarFiscalizacion(Object.freeze({
      expediente_ref: recibo.expediente_ref,
      version_esperada: recibo.version_resultante,
      fase_clave: "informe_juridico",
      informe_ref: recibo.informe_ref,
    }));
  }

  function montarSubsanacionDesdeExpedienteActual() {
    if (!montada || !subsanacionDisponible || desmontarSubsanacion !== null) return;
    const contexto = contextoSubsanacionDesdeEstado(presentador.obtenerEstado());
    const contenedor = raiz.querySelector("[data-ct-exp-subsanacion]");
    if (!contexto || !contenedor) return;
    const reciboConfirmado = reciboSubsanacionConfirmado?.recibo?.expediente_ref === contexto.expediente_ref
      && [reciboSubsanacionConfirmado.contexto.version_esperada, reciboSubsanacionConfirmado.recibo.version_resultante].includes(contexto.version_esperada)
      ? reciboSubsanacionConfirmado : null;
    try {
      desmontarSubsanacion = montarFormularioSubsanacionReparos({
        raiz: contenedor, cliente: clienteSubsanacion, contexto,
        traducir: crearTraductorContratacionTemporal(mensajes), confirmarOperacion,
        anunciar, reciboConfirmado,
        alConfirmar: refrescarDetalleTrasSubsanacion,
      });
    } catch {
      desmontarSubsanacion = null;
      const t = crearTraductorContratacionTemporal(mensajes);
      contenedor.innerHTML = `<p role="alert">${escaparHTML(t("subsanacion_montaje_error"))}</p>`;
    }
  }

  async function refrescarDetalleTrasSubsanacion(recibo, contextoOriginal) {
    if (!montada || recibo?.expediente_ref !== contextoOriginal?.expediente_ref
      || recibo.version_resultante <= contextoOriginal.version_esperada) return;
    reciboSubsanacionConfirmado = Object.freeze({ recibo, contexto: contextoOriginal });
    const seleccionado = presentador.obtenerEstado();
    if (seleccionado.vista !== "expediente"
      || seleccionado.expediente?.expediente_ref !== contextoOriginal.expediente_ref) return;
    const panel = raiz.querySelector("[data-ct-exp-subsanacion]");
    // cargar descarta la selección antigua cuando cambia la versión; el panel
    // identifica la navegación del usuario mientras se recupera la nueva.
    const sigueSeleccionado = () => montada && panel !== null
      && raiz.querySelector("[data-ct-exp-subsanacion]") === panel;
    const avisarPendiente = () => {
      if (sigueSeleccionado()) anunciar(
        crearTraductorContratacionTemporal(mensajes)("subsanacion_actualizacion_pendiente"), "aviso",
      );
    };
    try {
      await presentador.cargar();
      if (!sigueSeleccionado()) return;
      const resumen = presentador.obtenerEstado().cuadro?.expedientes?.find(
        ({ expediente_ref: referencia }) => referencia === contextoOriginal.expediente_ref,
      );
      if (!resumen || resumen.version < recibo.version_resultante) { avisarPendiente(); return; }
      await presentador.seleccionarExpediente(contextoOriginal.expediente_ref, "expediente");
      if (!sigueSeleccionado()) return;
      const actualizado = presentador.obtenerEstado().expediente;
      if (actualizado?.expediente_ref !== contextoOriginal.expediente_ref
        || actualizado.version < recibo.version_resultante) { avisarPendiente(); return; }
      repintar("[data-ct-subsanacion-recibo]");
    } catch {
      avisarPendiente();
    }
  }

  function montarLlamamiento(contexto = null) {
    if (!montada || !llamamientoDisponible) return;
    // Nueva petición conserva el acceso manual publicado para recuperar con
    // las mismas referencias tras reinicio, sin repetir alta ni fiscalización.
    // La bandeja y los detalles siguen necesitando contexto válido.
    const recuperacionManual = altaDisponible
      && presentador.obtenerEstado().vista === "alta";
    if (contexto === null && !recuperacionManual) return;
    if (desmontarLlamamiento !== null) {
      if (contexto !== null) desmontarLlamamiento.actualizarContexto(contexto);
      return;
    }
    const contenedor = raiz.querySelector("[data-ct-exp-llamamiento]");
    if (!contenedor) return;
    desmontarLlamamiento = montarFormularioLlamamiento({
      raiz: contenedor, cliente: clienteLlamamiento, contexto,
      confirmarOperacion, mensajes, locale, zonaHoraria, anunciar,
    });
  }

  async function montarResolucionFormalizacion() {
    const estado = presentador.obtenerEstado();
    if (!montada || !resolucionFormalizacionDisponible || desmontarResolucionFormalizacion || consultaResolucionFormalizacion
      || estado.carga !== "listo" || estado.expediente?.demostracion !== false
      || estado?.vista !== "expediente" || ![7, 8].includes(estado?.expediente?.version)) return;
    const contenedor = raiz.querySelector("[data-ct-exp-resolucion-formalizacion]");
    if (!contenedor) return;
    const expedienteRef = estado.expediente.expediente_ref;
    const version = estado.expediente.version;
    const controlador = new AbortController();
    consultaResolucionFormalizacion = controlador;
    const vigente = () => {
      const actual = presentador.obtenerEstado();
      return montada && consultaResolucionFormalizacion === controlador
        && !controlador.signal.aborted && raiz.contains?.(contenedor)
        && raiz.querySelector("[data-ct-exp-resolucion-formalizacion]") === contenedor
        && actual?.vista === "expediente" && actual.carga === "listo"
        && actual.expediente?.expediente_ref === expedienteRef
        && actual.expediente?.version === version;
    };
    const t = crearTraductorExpedientesContratacion(mensajes);
    contenedor.innerHTML = `<p class="ct-ayuda" role="status">${escaparHTML(t("resolucion_preparacion_cargando"))}</p>`;
    try {
      const preparacion = await clienteLlamamiento.prepararResolucionFormalizacion(
        expedienteRef, { signal: controlador.signal },
      );
      if (!vigente()) return;
      const promocionAutorizada = version === 7 && preparacion.version_actual === 8
        && preparacion.recibo !== null;
      if (preparacion.expediente_ref !== expedienteRef
        || (preparacion.version_actual !== version && !promocionAutorizada)) throw new TypeError("preparación no ligada");
      desmontarResolucionFormalizacion = montarFormularioResolucionFormalizacion({
        raiz: contenedor, cliente: clienteLlamamiento, confirmarOperacion, mensajes, locale, zonaHoraria,
        preparacion,
      });
    } catch (error) {
      if (!vigente()) return;
      const denegada = error?.envelopeValido === true && [401, 403].includes(error.estado);
      contenedor.innerHTML = `<p class="ct-estado ct-estado-aviso" role="status">${escaparHTML(t(denegada
        ? "resolucion_preparacion_denegada" : "resolucion_preparacion_no_disponible"))}</p>
        ${denegada ? "" : `<button class="boton-secundario" type="button" data-ct-exp-accion="reintentar-resolucion">${escaparHTML(t("resolucion_preparacion_reintentar"))}</button>`}`;
      desmontarResolucionFormalizacion = null;
    } finally {
      if (consultaResolucionFormalizacion === controlador) consultaResolucionFormalizacion = null;
    }
  }

  async function montarIncorporacionEjercicio() {
    const estado = presentador.obtenerEstado();
    if (!montada || !incorporacionEjercicioDisponible || desmontarIncorporacionEjercicio || consultaIncorporacionEjercicio
      || estado.carga !== "listo" || estado.expediente?.demostracion !== false
      || estado.vista !== "expediente" || !Number.isSafeInteger(estado.expediente?.version)
      || estado.expediente.version < 8) return;
    const contenedor = raiz.querySelector("[data-ct-exp-incorporacion-ejercicio]");
    if (!contenedor) return;
    const expedienteRef = estado.expediente.expediente_ref;
    const version = estado.expediente.version;
    const controlador = new AbortController();
    consultaIncorporacionEjercicio = controlador;
    const vigente = () => {
      const actual = presentador.obtenerEstado();
      return montada && consultaIncorporacionEjercicio === controlador && !controlador.signal.aborted
        && raiz.contains?.(contenedor) && raiz.querySelector("[data-ct-exp-incorporacion-ejercicio]") === contenedor
        && actual?.vista === "expediente" && actual.carga === "listo"
        && actual.expediente?.expediente_ref === expedienteRef && actual.expediente?.version === version;
    };
    const t = crearTraductorExpedientesContratacion(mensajes);
    const tCT = crearTraductorContratacionTemporal(mensajes);
    contenedor.innerHTML = `<p class="ct-ayuda" role="status">${escaparHTML(t("incorporacion_preparacion_cargando"))}</p>`;
    try {
      const preparacion = await clienteLlamamiento.prepararIncorporacionEjercicio(expedienteRef, { signal: controlador.signal });
      if (!vigente()) return;
      if (preparacion.expediente_ref !== expedienteRef) {
        throw new TypeError("preparación de incorporación no ligada al detalle actual");
      }
      if (preparacion.version_actual_expediente !== version) {
        if (preparacion.recibo !== null && preparacion.recibo.expediente_ref === expedienteRef
          && preparacion.version_actual_expediente > version) {
          // El recibo acredita una versión posterior; no se presenta el detalle
          // v8 como vigente ni se monta una acción sobre esa proyección.
          await presentador.cargar();
          if (!montada || controlador.signal.aborted) return;
          const resumen = presentador.obtenerEstado().cuadro?.expedientes?.find(
            ({ expediente_ref: referencia }) => referencia === expedienteRef,
          );
          if (!resumen || resumen.version < preparacion.version_actual_expediente) {
            contenedor.innerHTML = `<p class="ct-estado ct-estado-aviso" role="status">El detalle mostrado (v${version}) está obsoleto. No se habilita ninguna acción hasta que se recupere la versión ${preparacion.version_actual_expediente}.</p>`;
            return;
          }
          await presentador.seleccionarExpediente(expedienteRef, "expediente");
          if (!montada || controlador.signal.aborted) return;
          const actualizado = presentador.obtenerEstado().expediente;
          if (actualizado?.version < preparacion.version_actual_expediente) {
            contenedor.innerHTML = `<p class="ct-estado ct-estado-aviso" role="status">El detalle mostrado (v${version}) está obsoleto. No se habilita ninguna acción hasta que se recupere la versión ${preparacion.version_actual_expediente}.</p>`;
            return;
          }
          repintar("[data-ct-exp-mensaje]");
          return;
        }
        throw new TypeError("preparación de incorporación no ligada al detalle actual");
      }
      desmontarIncorporacionEjercicio = montarFormularioIncorporacionEjercicio({
        raiz: contenedor, cliente: clienteLlamamiento, preparacion,
        confirmarOperacion, mensajes, locale, zonaHoraria, anunciar,
      });
      // Sólo una incorporación recuperada y validada habilita la ficha. La
      // descarga vuelve a consultar al servidor; el recibo del DOM no autoriza.
      if (preparacion.recibo !== null && typeof clienteLlamamiento.descargarFichaGINPIX === "function") {
        const documento = contenedor.ownerDocument ?? entornoDescarga.document;
        const bloque = documento.createElement("div");
        contenedor.append(bloque);
        const desmontarFormulario = desmontarIncorporacionEjercicio;
        let urlFicha = null;
        const liberarFicha = () => {
          if (urlFicha !== null) entornoDescarga.URL.revokeObjectURL(urlFicha);
          urlFicha = null;
        };
        const resumenFicha = presentador.obtenerEstado().cuadro?.expedientes?.find(
          (fila) => fila.expediente_ref === expedienteRef && fila.version === version,
        );
        const desmontarFicha = montarFichaGINPIX({
          raiz: bloque, cliente: clienteLlamamiento, recibo: preparacion.recibo, mensajes,
          locale,
          resumen: resumenFicha ? { centro: resumenFicha.centro, categoria: resumenFicha.categoria } : undefined,
          descargarArchivo: (archivo, nombre) => {
            if (!montada || !raiz.contains(contenedor)) return;
            const BlobImpl = entornoDescarga.Blob ?? globalThis.Blob;
            const enlace = documento.createElement("a");
            liberarFicha();
            urlFicha = entornoDescarga.URL.createObjectURL(new BlobImpl([archivo.contenido], { type: "application/json" }));
            try {
              enlace.href = urlFicha; enlace.download = nombre; enlace.hidden = true;
              documento.body.append(enlace); enlace.click();
            } finally {
              enlace.remove(); setTimeout(liberarFicha, 0);
            }
          },
        });
        desmontarIncorporacionEjercicio = () => {
          desmontarFicha(); liberarFicha(); desmontarFormulario();
        };
      }
      if (preparacion.recibo !== null && preparacion.recibo.expediente_ref === expedienteRef
        && typeof clienteLlamamiento.anotacionAdministrativa?.registrar === "function"
        && typeof clienteLlamamiento.anotacionAdministrativa?.recuperar === "function") {
        const documento = contenedor.ownerDocument ?? entornoDescarga.document;
        const bloque = documento.createElement("div");
        const bloqueCierre = documento.createElement("div");
        contenedor.append(bloque);
        contenedor.append(bloqueCierre);
        let panelActivo = true;
        let versionObsoleta = null;
        let desmontarCierre = null;
        let lecturaCierreEnCurso = false;
        const panelSigueMontado = () => panelActivo && montada && !controlador.signal.aborted
          && raiz.contains?.(contenedor);
        const vigenteMontaje = () => {
          const actual = presentador.obtenerEstado();
          return versionObsoleta === null && panelSigueMontado()
            && actual?.vista === "expediente" && actual.carga === "listo"
            && actual.expediente?.expediente_ref === expedienteRef
            && actual.expediente?.version === version;
        };
        const mostrarCierreNoDisponible = (mensaje, recuperable = false) => {
          if (!panelSigueMontado()) return;
          desmontarCierre?.();
          desmontarCierre = null;
          bloqueCierre.innerHTML = `<section class="ct-alta" data-ct-cierre-administrativo>
            <h3>${escaparHTML(tCT("cierre_titulo"))}</h3>
            <p class="ct-ayuda" role="status">${escaparHTML(mensaje)}</p>
            ${recuperable ? `<button type="button" class="boton-secundario" data-ct-exp-reintentar-cierre>${escaparHTML(tCT("cierre_reintentar_lectura"))}</button>` : ""}
          </section>`;
          if (recuperable) {
            const boton = bloqueCierre.querySelector?.("[data-ct-exp-reintentar-cierre]");
            boton?.addEventListener?.("click", () => refrescarCierre());
          }
        };
        async function refrescarCierre(trasConfirmacion = false) {
          if (!vigenteMontaje() || typeof clienteLlamamiento.consultarPreparacionCierreSinCese !== "function"
            || typeof clienteLlamamiento.cerrar !== "function" || lecturaCierreEnCurso) return;
          lecturaCierreEnCurso = true;
          try {
            const cierre = await clienteLlamamiento.consultarPreparacionCierreSinCese({
              expediente_ref: expedienteRef,
              seguimiento_ref: preparacion.recibo.seguimiento_ref,
            }, { signal: controlador.signal });
            if (!vigenteMontaje()) return;
            if (trasConfirmacion) return { estadoActual: cierre.estado_actual };
            desmontarCierre?.();
            desmontarCierre = null;
            bloqueCierre.replaceChildren();
            desmontarCierre = montarFormularioCierreAdministrativo({
              raiz: bloqueCierre,
              cliente: clienteLlamamiento,
              preparacion: cierre.preparacion,
              estadoActual: cierre.estado_actual,
              contextoRecuperacion: {
                expediente_ref: expedienteRef,
                seguimiento_ref: preparacion.recibo.seguimiento_ref,
              },
              confirmarOperacion,
              alConfirmar: () => refrescarCierre(true),
              t: tCT,
            });
            return true;
          } catch {
            if (vigenteMontaje() && !trasConfirmacion) {
              mostrarCierreNoDisponible(
                tCT("cierre_lectura_error"),
                true,
              );
            }
            return false;
          } finally {
            lecturaCierreEnCurso = false;
          }
        }
        async function refrescarDetalleTrasAnotacion(reciboAnotacion) {
          if (!panelSigueMontado() || reciboAnotacion.expediente_ref !== expedienteRef
            || reciboAnotacion.version_resultante <= version) return;
          versionObsoleta = reciboAnotacion.version_resultante;
          mostrarCierreNoDisponible(
            tCT("cierre_detalle_obsoleto", {
              version: reciboAnotacion.version_resultante,
              version_anterior: version,
            }),
          );
          try {
            await presentador.cargar();
            if (!panelSigueMontado()) return;
            const cuadro = presentador.obtenerEstado().cuadro;
            const resumen = cuadro?.expedientes?.find(({ expediente_ref: referencia }) => referencia === expedienteRef);
            if (!resumen || resumen.version < reciboAnotacion.version_resultante) return;
            await presentador.seleccionarExpediente(expedienteRef, "expediente");
            if (!panelSigueMontado()) return;
            const actualizado = presentador.obtenerEstado().expediente;
            if (actualizado?.expediente_ref !== expedienteRef
              || actualizado.version < reciboAnotacion.version_resultante) return;
            repintar("[data-ct-exp-mensaje]");
            const mensaje = raiz.querySelector("[data-ct-exp-mensaje]");
            const confirmacion = tCT("anotacion_detalle_actualizado", {
              recibo: reciboAnotacion.recibo_ref,
              version: actualizado.version,
            });
            if (mensaje) {
              mensaje.textContent = confirmacion;
              mensaje.setAttribute("role", "status");
            }
            anunciar(confirmacion, "exito");
          } catch {
            // El aviso de obsolescencia conserva el recibo; una lectura posterior puede recuperarlo.
          }
        }
        const desmontarAnotacion = montarFormularioAnotacionAdministrativa({
          raiz: bloque,
          cliente: clienteLlamamiento.anotacionAdministrativa,
          expediente: { expediente_ref: expedienteRef, version },
          confirmarOperacion,
          locale,
          zonaHoraria,
          alConfirmar: refrescarDetalleTrasAnotacion,
          t: tCT,
        });
        const desmontarAntesAnotacion = desmontarIncorporacionEjercicio;
        desmontarIncorporacionEjercicio = () => {
          panelActivo = false;
          desmontarCierre?.();
          desmontarAnotacion();
          bloqueCierre.replaceChildren();
          desmontarAntesAnotacion();
        };
        await refrescarCierre();
        if (!vigente()) return;
      }
      if (preparacion.recibo !== null && typeof clienteLlamamiento.seguimientoIncorporacion?.consultar === "function") {
        const documento = contenedor.ownerDocument ?? entornoDescarga.document;
        const bloque = documento.createElement("div");
        contenedor.append(bloque);
        const desmontarAnterior = desmontarIncorporacionEjercicio;
        const desmontarSeguimiento = montarSeguimientoIncorporacion({
          raiz: bloque, cliente: clienteLlamamiento.seguimientoIncorporacion,
          recibo: preparacion.recibo, mensajes,
        });
        desmontarIncorporacionEjercicio = () => {
          desmontarSeguimiento(); desmontarAnterior();
        };
      }
    } catch (error) {
      if (!vigente()) return;
      const denegada = error?.envelopeValido === true && [401, 403].includes(error.estado);
      contenedor.innerHTML = `<p class="ct-estado ct-estado-aviso" role="status">${escaparHTML(t(denegada
        ? "incorporacion_preparacion_denegada" : "incorporacion_preparacion_no_disponible"))}</p>
        ${denegada ? "" : `<button class="boton-secundario" type="button" data-ct-exp-accion="reintentar-incorporacion">${escaparHTML(t("incorporacion_preparacion_reintentar"))}</button>`}`;
    } finally {
      if (consultaIncorporacionEjercicio === controlador) consultaIncorporacionEjercicio = null;
    }
  }

  function montarInformeDesdeAsignacion(recibo) {
    if (!montada || !informeJuridicoDisponible || desmontarInformeJuridico !== null) {
      return desmontarInformeJuridico !== null;
    }
    const contenedor = raiz.querySelector("[data-ct-exp-informe-juridico]");
    if (!contenedor) return false;
    try {
      desmontarInformeJuridico = montarFormularioInformeJuridico({
        raiz: contenedor, cliente: composicionAnalisis.cliente,
        contexto: Object.freeze({
          expediente_ref: recibo.expediente_ref,
          version_esperada: recibo.version_resultante,
        }),
        confirmarOperacion, mensajes, locale, zonaHoraria, anunciar,
        alConfirmar: montarFiscalizacionDesdeInforme,
      });
      return true;
    } catch {
      desmontarInformeJuridico = null;
      mostrarErrorMontaje(
        raiz.querySelector("[data-ct-exp-asignacion]"),
        "informe-juridico",
        () => montarInformeDesdeAsignacion(recibo),
      );
      return false;
    }
  }

  function montarFiscalizacionDesdeExpedienteActual() {
    const contexto = contextoFiscalizacionDesdeEstado(presentador.obtenerEstado());
    if (contexto === null) return null;
    return montarFiscalizacion(contexto);
  }

  function montarInformeDesdeExpedienteActual() {
    const contextoInforme = contextoInformeJuridicoDesdeEstado(
      presentador.obtenerEstado(),
    );
    if (contextoInforme === null) return null;
    return montarInformeDesdeAsignacion({
      expediente_ref: contextoInforme.expediente_ref,
      version_resultante: contextoInforme.version_esperada,
    });
  }

  function retirarComponentes() {
    limpiarEstadosMontaje();
    consultaIncorporacionEjercicio?.abort();
    consultaIncorporacionEjercicio = null;
    desmontarIncorporacionEjercicio?.();
    desmontarIncorporacionEjercicio = null;
    retirarAsignacion();
    retirarCobertura();
    retirarAlta();
    retirarAnalisis();
  }

  function montarAsignacionDesdeCobertura(expedienteRef, recibo) {
    if (!montada || !asignacionDisponible) return false;
    if (desmontarAsignacion !== null) return true;
    const contenedor = raiz.querySelector("[data-ct-exp-asignacion]");
    if (!contenedor) return false;
    try {
      desmontarAsignacion = montarFormularioAsignacion({
        raiz: contenedor,
        cliente: composicionAnalisis.cliente,
        contexto: Object.freeze({
          expediente_ref: expedienteRef,
          version_esperada: recibo.version_resultante,
        }),
        confirmarOperacion,
        mensajes,
        locale,
        zonaHoraria,
        anunciar,
        alConfirmar: montarInformeDesdeAsignacion,
      });
      return true;
    } catch {
      desmontarAsignacion = null;
      mostrarErrorMontaje(
        raiz.querySelector("[data-ct-exp-cobertura]"),
        "asignacion",
        () => montarAsignacionDesdeCobertura(expedienteRef, recibo),
      );
      return false;
    }
  }

  function montarAsignacionDesdeEstado() {
    const contexto = contextoAsignacionDesdeEstado(presentador.obtenerEstado());
    if (contexto === null) return false;
    return montarAsignacionDesdeCobertura(contexto.expediente_ref, {
      version_resultante: contexto.version_esperada,
    });
  }

  function montarCoberturaDesdeEstado() {
    const contexto = contextoCoberturaDesdeEstado(presentador.obtenerEstado());
    if (contexto === null) return false;
    return montarCoberturaDesdeAnalisis({
      expediente_ref: contexto.expediente_ref,
      version_resultante: contexto.version_esperada,
    });
  }

  function montarCoberturaDesdeAnalisis(recibo) {
    if (!montada || !coberturaDisponible || desmontarCobertura !== null) return null;
    const contenedor = raiz.querySelector("[data-ct-exp-cobertura]");
    if (!contenedor) return null;
    try {
      desmontarCobertura = montarFormularioCobertura({
        raiz: contenedor,
        cliente: composicionAnalisis.cliente,
        contexto: Object.freeze({
          expediente_ref: recibo.expediente_ref,
          version_esperada: recibo.version_resultante,
        }),
        confirmarOperacion,
        mensajes,
        locale,
        zonaHoraria,
        anunciar,
        alConfirmar: (reciboCobertura) => montarAsignacionDesdeCobertura(
          recibo.expediente_ref,
          reciboCobertura,
        ),
      });
      return true;
    } catch {
      desmontarCobertura = null;
      mostrarErrorMontaje(
        raiz.querySelector("[data-ct-exp-analisis]"),
        "cobertura",
        () => montarCoberturaDesdeAnalisis(recibo),
      );
      return false;
    }
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
      montarCoberturaDesdeAnalisis,
      (recibo) => mostrarErrorMontaje(
        contenedor,
        "cobertura",
        () => montarCoberturaDesdeAnalisis(recibo),
      ),
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
      mostrarErrorMontaje(
        contenedor,
        "analisis",
        () => montarAnalisisEnContenedor(contenedor, contexto, analisisInicial),
      );
      return null;
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
    if (!montada || estado.vista !== "expediente" || composicionAnalisis === null
      || estado.carga !== "listo" || estado.expediente === null) return null;
    const rectificacion = contextoRectificacionAnalisisDesdeEstado(estado);
    if (rectificacion !== null) {
      const contenedor = raiz.querySelector("[data-ct-exp-rectificacion]");
      if (!contenedor) return null;
      if (!rectificacionDisponible) {
        const t = crearTraductorContratacionTemporal(mensajes);
        contenedor.innerHTML = `<section class="ct-alcance" role="status" aria-live="polite">
          <h3>${escaparHTML(t("analisis_rectificacion_configuracion_pendiente_titulo"))}</h3>
          <p>${escaparHTML(t("analisis_rectificacion_configuracion_pendiente_descripcion"))}</p>
        </section>`;
        return true;
      }
      return montarAnalisisEnContenedor(
        contenedor,
        Object.freeze({
          ...rectificacion,
          artefacto_ref: composicionAnalisis.rectificacion.artefacto_ref,
        }),
        null,
      );
    }
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
    cancelarDescargaInforme();
    retirarComponentes();
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
      llamamientoDisponible,
      resolucionFormalizacionDisponible,
      incorporacionEjercicioDisponible,
    });
    montarAltaSiProcede();
    montarLlamamiento(contextoLlamamientoDesdeEstado(estado));
    // Las dos lecturas comparten la identidad nominal; se encadenan.
    montarResolucionFormalizacion().then(montarIncorporacionEjercicio);
    if (montarAnalisisSiProcede() === false) {
      retirarComponentes();
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
      montarCoberturaDesdeEstado();
      montarAsignacionDesdeEstado();
      montarInformeDesdeExpedienteActual();
      montarFiscalizacionDesdeExpedienteActual();
      montarSubsanacionDesdeExpedienteActual();
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
    const pagina = evento.target?.closest?.("[data-ct-exp-pagina]");
    if (pagina && raiz.contains(pagina)) {
      evento.preventDefault();
      if (pagina.disabled || impedirCambioPorAnalisis()) return;
      const promesa = presentador.navegarPagina(pagina.dataset.ctExpPagina);
      repintar("[data-ct-exp-mensaje]");
      await promesa;
      repintar("[data-ct-exp-filtros]");
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
    if (accion.dataset.ctExpAccion === "volver-cuadro-actualizado") {
      presentador.cambiarVista("cuadro");
      const promesa = presentador.cargar();
      repintar("[data-ct-exp-mensaje]");
      await promesa;
      repintar(["[data-ct-exp-filtros]", ".ct-exp-estado-global"]);
    } else if (accion.dataset.ctExpAccion === "reintentar-resolucion") {
      await montarResolucionFormalizacion();
    } else if (accion.dataset.ctExpAccion === "reintentar-incorporacion") {
      await montarIncorporacionEjercicio();
    } else if (["descargar-informe-definitivo", "descargar-resolucion", "descargar-diligencia", "descargar-toma-posesion", "descargar-notificacion", "descargar-comunicacion-centro", "descargar-docx-informe-definitivo", "descargar-docx-resolucion", "descargar-docx-diligencia", "descargar-docx-toma-posesion", "descargar-docx-notificacion", "descargar-docx-comunicacion-centro"].includes(accion.dataset.ctExpAccion)) {
      await descargarBorrador(accion);
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
    if (impedirCambioPorAnalisis() || presentador.obtenerEstado().ocupado) return;
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
      cancelarDescargaInforme();
      retirarComponentes();
      raiz.removeEventListener("click", manejarClick);
      raiz.removeEventListener("submit", manejarEnvio);
      presentador.desmontar?.();
    },
  });
}

export function montarModuloFiscalizacionContratacionTemporal({
  raiz,
  cliente,
  mensajes = {},
  anunciar = () => {},
  confirmarOperacion = () => false,
  locale = "es-ES",
  zonaHoraria = "Europe/Madrid",
} = {}) {
  if (!raiz || typeof raiz.addEventListener !== "function"
    || typeof raiz.querySelector !== "function"
    || typeof cliente?.registrarResultadoFiscalizacion !== "function"
    || typeof anunciar !== "function" || typeof confirmarOperacion !== "function") {
    throw new TypeError("dependencias de fiscalización no válidas");
  }
  let montado = true;
  let desmontarFormulario = null;
  let desmontarLlamamiento = null;
  raiz.innerHTML = `<section class="ct-expedientes" data-modulo="contratacion-temporal"
    aria-labelledby="ct-fiscalizacion-acceso-titulo">
    <header class="ct-exp-cabecera">
      <p class="sobrelinea">Intervención</p>
      <h2 id="ct-fiscalizacion-acceso-titulo">Fiscalización de contratación temporal</h2>
      <p>Abra un expediente remitido por Recursos Humanos para registrar su resultado.</p>
    </header>
    <form class="ct-exp-filtros" data-ct-fiscalizacion-acceso>
      <div class="ct-campo">
        <label for="ct-fiscalizacion-expediente">Referencia del expediente</label>
        <input id="ct-fiscalizacion-expediente" name="expediente_ref"
          type="text" maxlength="160" autocomplete="off" required>
      </div>
      <div class="ct-campo">
        <label for="ct-fiscalizacion-version">Versión remitida</label>
        <input id="ct-fiscalizacion-version" name="version_esperada"
          type="number" min="1" step="1" required>
      </div>
      <div class="ct-acciones">
        <button class="boton-primario" type="submit">Abrir fiscalización</button>
      </div>
    </form>
  </section>`;

  function manejarEnvio(evento) {
    const formulario = evento.target?.closest?.("[data-ct-fiscalizacion-acceso]");
    if (!formulario || !raiz.contains(formulario) || !montado) return;
    evento.preventDefault();
    if (typeof formulario.checkValidity === "function" && !formulario.checkValidity()) {
      formulario.reportValidity?.();
      return;
    }
    const referencia = String(
      formulario.elements?.namedItem?.("expediente_ref")?.value ?? "",
    ).trim();
    const version = Number(
      formulario.elements?.namedItem?.("version_esperada")?.value ?? 0,
    );
    if (!PATRON_REFERENCIA.test(referencia) || !Number.isSafeInteger(version) || version < 1) {
      formulario.elements?.namedItem?.("expediente_ref")?.setCustomValidity?.(
        "Indique la referencia íntegra del expediente remitido.",
      );
      formulario.reportValidity?.();
      return;
    }
    raiz.innerHTML = `<section class="ct-expedientes" data-modulo="contratacion-temporal">
      <div data-ct-exp-fiscalizacion></div>
      <div data-ct-exp-llamamiento></div>
    </section>`;
    const contenedor = raiz.querySelector("[data-ct-exp-fiscalizacion]");
    desmontarFormulario = montarFormularioFiscalizacion({
      raiz: contenedor,
      cliente,
      contexto: Object.freeze({
        expediente_ref: referencia,
        version_esperada: version,
        fase_clave: "informe_juridico",
        informe_ref: "",
      }),
      confirmarOperacion,
      mensajes,
      locale,
      zonaHoraria,
      anunciar,
      alConfirmar: (recibo) => {
        if (recibo.resultado === "desfavorable" || recibo.version_resultante < 6
          || desmontarLlamamiento !== null
          || typeof cliente.seleccionarLlamamiento !== "function"
          || typeof cliente.registrarComunicacionLlamamiento !== "function") return;
        desmontarLlamamiento = montarFormularioLlamamiento({
          raiz: raiz.querySelector("[data-ct-exp-llamamiento]"), cliente,
          contexto: { expediente_ref: recibo.expediente_ref,
            version_esperada: recibo.version_resultante },
          confirmarOperacion, mensajes, locale, zonaHoraria, anunciar,
        });
      },
    });
  }

  raiz.addEventListener("submit", manejarEnvio);
  return Object.freeze({
    desmontar() {
      if (!montado) return;
      montado = false;
      if (typeof desmontarFormulario === "function") desmontarFormulario();
      desmontarLlamamiento?.();
      raiz.removeEventListener("submit", manejarEnvio);
    },
  });
}
