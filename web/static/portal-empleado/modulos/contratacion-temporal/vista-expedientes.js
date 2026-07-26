/** Vista y enlace DOM de la superficie de expedientes de contratación temporal. */

import { crearPresentadorAltaContratacionTemporal } from "./presentador.js";
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

function renderizarAlta(t, disponible) {
  return `<header class="ct-exp-subcabecera">
    <h3>${escaparHTML(t("nueva_peticion_titulo"))}</h3>
    <p>${escaparHTML(t("nueva_peticion_descripcion"))}</p>
  </header>
  ${disponible
    ? '<div data-ct-exp-alta></div>'
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
} = {}) {
  const t = crearTraductorExpedientesContratacion(mensajes);
  let contenido;
  if (["cargando", "error", "denegado"].includes(estado.carga)
    && estado.vista !== "alta") {
    contenido = renderizarEstadoCarga(estado, t);
  } else if (estado.vista === "alta") {
    contenido = renderizarAlta(t, altaDisponible);
  } else if (estado.vista === "expediente") {
    contenido = renderizarExpediente(estado, t, locale, zonaHoraria);
  } else if (estado.vista === "documentos") {
    contenido = renderizarDocumentos(estado, t);
  } else if (estado.vista === "auditoria") {
    contenido = renderizarAuditoria(estado, t);
  } else {
    contenido = renderizarCuadro(estado, t);
  }
  return `<section class="ct-expedientes" data-modulo="contratacion-temporal"
    aria-labelledby="ct-exp-titulo">
    ${renderizarCabeceraModulo(estado, t).replace("<h2>", '<h2 id="ct-exp-titulo">')}
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
  let montada = true;
  let desmontarAlta = null;

  function retirarAlta() {
    if (typeof desmontarAlta === "function") desmontarAlta();
    desmontarAlta = null;
  }

  function montarAltaSiProcede() {
    const estado = presentador.obtenerEstado();
    if (!montada || estado.vista !== "alta" || !altaDisponible) return;
    const contenedor = raiz.querySelector("[data-ct-exp-alta]");
    if (!contenedor) return;
    const presentadorAlta = crearPresentadorAltaContratacionTemporal({
      catalogos: alta.catalogos,
      capacidad: alta.capacidad,
      ejecutor: alta.ejecutor,
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

  function repintar(selectorFoco = "") {
    if (!montada) return;
    retirarAlta();
    const estado = presentador.obtenerEstado();
    raiz.innerHTML = renderizarModuloContratacionTemporal(estado, {
      mensajes,
      locale,
      zonaHoraria,
      altaDisponible,
    });
    montarAltaSiProcede();
    if (selectorFoco) enfocar(raiz, selectorFoco);
    anunciar(
      crearTraductorExpedientesContratacion(mensajes)(estado.mensaje_clave),
      estado.tipo_mensaje,
    );
  }

  async function cambiarVista(vista) {
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
      if (presentador.obtenerEstado().ocupado) return;
      presentador.seleccionarTarea(tareaControl.dataset.ctExpTarea);
      repintar("#ct-exp-tarea-titulo");
      return;
    }
    const efecto = evento.target?.closest?.("[data-ct-exp-efecto]");
    if (efecto && raiz.contains(efecto)) {
      evento.preventDefault();
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
    if (presentador.obtenerEstado().ocupado) return;
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
      montada = false;
      retirarAlta();
      raiz.removeEventListener("click", manejarClick);
      raiz.removeEventListener("submit", manejarEnvio);
      presentador.desmontar?.();
    },
  });
}
