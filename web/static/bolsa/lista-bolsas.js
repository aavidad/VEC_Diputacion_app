/**
 * Controlador de la interfaz pública de bolsas y listas de aspirantes (B10).
 * Consulta anónima sin autenticación y sin almacenamiento web.
 */

import {
  consultarBolsasPublicas,
  consultarListaBolsaPublica,
} from "./lista-bolsas-api.js";
import { PATRON_DOCUMENTO_ENMASCARADO } from "./contrato-publico-bolsas.js";

const t = globalThis.VECBolsaI18n?.t || ((clave) => clave);

function escaparHTML(valor) {
  return String(valor ?? "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

function formatoFecha(iso) {
  if (!iso) return t("dato_no_disponible");
  try {
    const d = new Date(iso);
    return isNaN(d.getTime()) ? t("dato_no_disponible") : d.toLocaleDateString("es-ES", { dateStyle: "medium" });
  } catch {
    return t("dato_no_disponible");
  }
}

export function crearControladorListaBolsas({
  elementos,
  api = { consultarBolsasPublicas, consultarListaBolsaPublica },
  ventana = typeof window === "undefined" ? null : window,
}) {
  const estado = {
    bolsas: [],
    bolsaSeleccionada: null,
    posiciones: [],
    hayMas: false,
    cursorSiguiente: null,
    filtroDocumento: "",
    bolsaRefSolicitada: "",
    cursorSolicitado: "",
    cargando: false,
    secuenciaConsulta: 0,
  };

  function etiquetaEstado(clave) {
    return t(clave);
  }

  function mensajeError(error, alcance) {
    if (error?.status === 401 || error?.status === 403) return t(`${alcance}_denegado`);
    if (alcance === "error_bolsas" && error?.status === 404) return t("consulta_no_disponible");
    if (alcance === "error_lista" && error?.status === 404) return t("error_lista_no_encontrada");
    return t(alcance);
  }

  function mostrarTabla(cuerpo, visible) {
    const contenedor = cuerpo?.closest?.(".tabla-contenedor-accesible");
    if (contenedor) contenedor.hidden = !visible;
  }

  function habilitarBusqueda(habilitada) {
    const campos = elementos.formularioBusqueda?.querySelector?.("fieldset");
    if (campos) campos.disabled = !habilitada;
  }

  function enfocarDuranteReintento(cargando, reintentoEnfocado) {
    if (!reintentoEnfocado || !cargando?.focus) return;
    cargando.tabIndex = -1;
    cargando.focus();
  }

  function enfocarResultadoReintento(cargando, destino) {
    if (!destino?.focus || cargando?.ownerDocument?.activeElement !== cargando) return;
    if (destino.tabIndex < 0 && !destino.hasAttribute?.("tabindex")) destino.tabIndex = -1;
    destino.focus();
  }

  async function cargarBolsas({ reintentoEnfocado = false } = {}) {
    const secuencia = ++estado.secuenciaConsulta;
    estado.cargando = true;
    elementos.seccionBolsas?.setAttribute?.("aria-busy", "true");
    elementos.seccionBolsas.hidden = false;
    elementos.seccionLista.hidden = true;
    elementos.bolsasCargando.hidden = false;
    elementos.bolsasError.hidden = true;
    elementos.bolsasVacio.hidden = true;
    if (elementos.cuerpoTablaBolsas) elementos.cuerpoTablaBolsas.innerHTML = "";
    mostrarTabla(elementos.cuerpoTablaBolsas, false);
    enfocarDuranteReintento(elementos.bolsasCargando, reintentoEnfocado);

    try {
      const respuesta = await api.consultarBolsasPublicas();
      if (secuencia !== estado.secuenciaConsulta) return;
      estado.bolsas = respuesta.bolsas;
      if (estado.bolsas.length === 0) {
        elementos.bolsasVacio.hidden = false;
        enfocarResultadoReintento(elementos.bolsasCargando, elementos.bolsasVacio?.querySelector?.("h3"));
        elementos.bolsasCargando.hidden = true;
        return;
      }

      renderizarTablaBolsas(estado.bolsas);
      mostrarTabla(elementos.cuerpoTablaBolsas, true);
      enfocarResultadoReintento(elementos.bolsasCargando, elementos.seccionBolsas?.querySelector?.("h2"));
      elementos.bolsasCargando.hidden = true;
    } catch (err) {
      if (secuencia !== estado.secuenciaConsulta) return;
      elementos.bolsasError.hidden = false;
      if (elementos.mensajeErrorBolsas) {
        elementos.mensajeErrorBolsas.textContent = mensajeError(err, "error_bolsas");
      }
      enfocarResultadoReintento(elementos.bolsasCargando, elementos.bolsasError?.querySelector?.("h3"));
      elementos.bolsasCargando.hidden = true;
    } finally {
      if (secuencia === estado.secuenciaConsulta) {
        estado.cargando = false;
        elementos.seccionBolsas?.removeAttribute?.("aria-busy");
      }
    }
  }

  function renderizarTablaBolsas(bolsas) {
    if (!elementos.cuerpoTablaBolsas) return;
    elementos.cuerpoTablaBolsas.innerHTML = bolsas.map((b) => `
      <tr data-bolsa-ref="${escaparHTML(b.bolsa_ref)}">
        <td><strong>${escaparHTML(b.categoria)}</strong></td>
        <td>${escaparHTML((b.grupos || []).join(", "))}</td>
        <td>${escaparHTML(b.tipo_lista)}</td>
        <td><small>${escaparHTML(formatoFecha(b.vigente_desde))}</small></td>
        <td><strong>${Number(b.total || 0).toLocaleString("es-ES")}</strong></td>
        <td>
          <button type="button" class="boton-secundario" data-accion="ver-bolsa" data-bolsa-ref="${escaparHTML(b.bolsa_ref)}">
            ${escaparHTML(t("consultar_lista"))}
          </button>
        </td>
      </tr>
    `).join("");
  }

  async function seleccionarBolsa(bolsaRef, documento = "", cursor = "", { historial = "push", reintentoEnfocado = false } = {}) {
    const secuencia = ++estado.secuenciaConsulta;
    const documentoSeguro = PATRON_DOCUMENTO_ENMASCARADO.test(documento) ? documento : "";
    estado.cargando = true;
    estado.filtroDocumento = documentoSeguro;
    estado.bolsaRefSolicitada = bolsaRef;
    estado.cursorSolicitado = cursor;
    elementos.seccionLista?.setAttribute?.("aria-busy", "true");
    elementos.seccionBolsas.hidden = true;
    elementos.seccionLista.hidden = false;
    elementos.listaError.hidden = true;
    elementos.listaVacio.hidden = true;
    if (elementos.contenedorPaginacion) elementos.contenedorPaginacion.hidden = true;
    if (elementos.botonSiguiente) elementos.botonSiguiente.disabled = true;

    elementos.listaCargando.hidden = false;
    enfocarDuranteReintento(elementos.listaCargando, reintentoEnfocado);
    if (!cursor) {
      estado.bolsaSeleccionada = null;
      estado.posiciones = [];
      if (elementos.inputDocumento) elementos.inputDocumento.value = documentoSeguro;
      if (elementos.errorDocumento) elementos.errorDocumento.hidden = true;
      if (elementos.infoBolsaActiva) elementos.infoBolsaActiva.innerHTML = "";
      if (elementos.cuerpoTablaLista) elementos.cuerpoTablaLista.innerHTML = "";
      mostrarTabla(elementos.cuerpoTablaLista, false);
      habilitarBusqueda(false);
    }

    try {
      const respuesta = await api.consultarListaBolsaPublica({
        bolsa_ref: bolsaRef,
        documento: documentoSeguro,
        cursor,
      });
      if (secuencia !== estado.secuenciaConsulta) return;
      if (respuesta.bolsa?.bolsa_ref !== bolsaRef) throw new Error("La lista pública no corresponde a la bolsa solicitada");

      estado.bolsaSeleccionada = respuesta.bolsa;
      estado.hayMas = respuesta.hay_mas;
      estado.cursorSiguiente = respuesta.cursor_siguiente;

      if (cursor) {
        estado.posiciones = [...estado.posiciones, ...respuesta.posiciones];
      } else {
        estado.posiciones = respuesta.posiciones;
      }

      renderizarDetalleBolsa(estado.bolsaSeleccionada);
      renderizarTablaLista(estado.posiciones);
      mostrarTabla(elementos.cuerpoTablaLista, estado.posiciones.length > 0);
      habilitarBusqueda(true);

      if (estado.posiciones.length === 0) {
        elementos.listaVacio.hidden = false;
      }

      actualizarPaginacion();
      actualizarURL(bolsaRef, documentoSeguro, historial);
      if (reintentoEnfocado) {
        const destino = estado.posiciones.length === 0
          ? elementos.listaVacio?.querySelector?.("h3")
          : cursor
            ? elementos.cuerpoTablaLista?.closest?.(".tabla-contenedor-accesible")
            : elementos.infoBolsaActiva?.querySelector?.("h2");
        enfocarResultadoReintento(elementos.listaCargando, destino ?? elementos.botonVolverBolsas);
      } else if (!cursor && historial === "push") {
        const titulo = elementos.infoBolsaActiva?.querySelector?.("h2");
        titulo?.focus?.();
      }
      elementos.listaCargando.hidden = true;
    } catch (err) {
      if (secuencia !== estado.secuenciaConsulta) return;
      elementos.listaError.hidden = false;
      if (elementos.mensajeErrorLista) {
        elementos.mensajeErrorLista.textContent = mensajeError(err, "error_lista");
      }
      const tituloError = elementos.listaError?.querySelector?.("h3");
      if (reintentoEnfocado) {
        enfocarResultadoReintento(elementos.listaCargando, tituloError ?? elementos.botonVolverBolsas);
      } else if (tituloError && !cursor && historial === "push") {
        tituloError.tabIndex = -1;
        tituloError.focus();
      }
      elementos.listaCargando.hidden = true;
    } finally {
      if (secuencia === estado.secuenciaConsulta) {
        estado.cargando = false;
        elementos.seccionLista?.removeAttribute?.("aria-busy");
        if (elementos.botonSiguiente) elementos.botonSiguiente.disabled = false;
      }
    }
  }

  function renderizarDetalleBolsa(bolsa) {
    if (!elementos.infoBolsaActiva || !bolsa) return;
    elementos.infoBolsaActiva.innerHTML = `
      <div class="tarjeta-bolsa-activa__info">
        <h2 tabindex="-1">${escaparHTML(bolsa.categoria)}</h2>
        <div class="tarjeta-bolsa-activa__meta">
          <span><strong>${escaparHTML(t("grupos"))}:</strong> ${escaparHTML((bolsa.grupos || []).join(", "))}</span>
          <span><strong>${escaparHTML(t("tipo_lista"))}:</strong> ${escaparHTML(bolsa.tipo_lista)}</span>
          <span><strong>${escaparHTML(t("vigente_desde"))}:</strong> ${escaparHTML(formatoFecha(bolsa.vigente_desde))}</span>
          <span><strong>${escaparHTML(t("total_aspirantes"))}:</strong> ${Number(bolsa.total || 0).toLocaleString("es-ES")}</span>
        </div>
      </div>
    `;
  }

  function renderizarTablaLista(posiciones) {
    if (!elementos.cuerpoTablaLista) return;
    if (posiciones.length === 0) {
      elementos.cuerpoTablaLista.innerHTML = "";
      return;
    }

    elementos.cuerpoTablaLista.innerHTML = posiciones.map((p) => `
      <tr data-orden="${escaparHTML(p.orden)}">
        <td><strong>#${Number(p.orden).toLocaleString("es-ES")}</strong></td>
        <td><code>${escaparHTML(p.documento_enmascarado)}</code></td>
        <td>
          <span class="estado-chip-publico estado-chip-publico--${escaparHTML(p.estado_clave)}">
            ${escaparHTML(etiquetaEstado(p.estado_clave))}
          </span>
        </td>
      </tr>
    `).join("");
  }

  function actualizarPaginacion() {
    if (!elementos.contenedorPaginacion) return;
    if (estado.hayMas && estado.cursorSiguiente) {
      elementos.contenedorPaginacion.hidden = false;
      if (elementos.botonSiguiente) {
        elementos.botonSiguiente.dataset.cursor = estado.cursorSiguiente;
      }
    } else {
      elementos.contenedorPaginacion.hidden = true;
    }
  }

  function actualizarURL(bolsaRef, documento, modo = "push") {
    if (!ventana?.history || !ventana?.location) return;
    const actualizarHistoria = modo === "push" ? ventana.history.pushState : ventana.history.replaceState;
    if (typeof actualizarHistoria !== "function") return;
    const url = new URL(ventana.location.href);
    if (bolsaRef) {
      url.searchParams.set("bolsa", bolsaRef);
    } else {
      url.searchParams.delete("bolsa");
    }
    if (documento) {
      url.searchParams.set("documento", documento);
    } else {
      url.searchParams.delete("documento");
    }
    actualizarHistoria.call(ventana.history, {}, "", url.toString());
  }

  function volverABolsas() {
    estado.bolsaSeleccionada = null;
    estado.posiciones = [];
    estado.filtroDocumento = "";
    estado.bolsaRefSolicitada = "";
    estado.cursorSolicitado = "";
    if (elementos.inputDocumento) elementos.inputDocumento.value = "";
    if (elementos.errorDocumento) elementos.errorDocumento.hidden = true;
    ++estado.secuenciaConsulta;
    actualizarURL("", "", "push");
    cargarBolsas();
    const titulo = elementos.seccionBolsas?.querySelector?.("h2");
    if (titulo) {
      titulo.tabIndex = -1;
      titulo.focus();
    }
  }

  function restaurarDesdeURL() {
    if (!ventana?.location) return cargarBolsas();
    const parametros = new URLSearchParams(ventana.location.search);
    const bolsaRef = parametros.get("bolsa");
    const documentoURL = parametros.get("documento") || "";
    const documento = PATRON_DOCUMENTO_ENMASCARADO.test(documentoURL) ? documentoURL : "";
    if (!bolsaRef) {
      if (documentoURL) actualizarURL("", "", "replace");
      return volverAListadoDesdeHistorial();
    }
    if (documentoURL && !documento) actualizarURL(bolsaRef, "", "replace");
    if (elementos.inputDocumento) {
      elementos.inputDocumento.value = documento;
    }
    return seleccionarBolsa(bolsaRef, documento, "", { historial: "replace" });
  }

  function volverAListadoDesdeHistorial() {
    ++estado.secuenciaConsulta;
    estado.bolsaSeleccionada = null;
    estado.posiciones = [];
    estado.filtroDocumento = "";
    estado.bolsaRefSolicitada = "";
    estado.cursorSolicitado = "";
    if (elementos.inputDocumento) elementos.inputDocumento.value = "";
    if (elementos.errorDocumento) elementos.errorDocumento.hidden = true;
    return cargarBolsas();
  }

  function instalar() {
    if (elementos.seccionBolsas) {
      elementos.seccionBolsas.addEventListener("click", (e) => {
        const boton = e.target.closest('[data-accion="ver-bolsa"]');
        if (!boton) return;
        const bolsaRef = boton.dataset.bolsaRef;
        if (bolsaRef) {
          seleccionarBolsa(bolsaRef);
        }
      });
    }

    if (elementos.formularioBusqueda) {
      elementos.formularioBusqueda.addEventListener("submit", (e) => {
        e.preventDefault();
        if (!estado.bolsaSeleccionada) return;
        const val = (elementos.inputDocumento ? elementos.inputDocumento.value : "").trim();
        if (val && !PATRON_DOCUMENTO_ENMASCARADO.test(val)) {
          if (elementos.errorDocumento) {
            elementos.errorDocumento.hidden = false;
            elementos.errorDocumento.textContent = t("documento_formato");
          }
          return;
        }
        if (elementos.errorDocumento) elementos.errorDocumento.hidden = true;
        seleccionarBolsa(estado.bolsaSeleccionada.bolsa_ref, val);
      });
    }

    if (elementos.botonLimpiarBusqueda) {
      elementos.botonLimpiarBusqueda.addEventListener("click", () => {
        if (!estado.bolsaSeleccionada) return;
        if (elementos.inputDocumento) elementos.inputDocumento.value = "";
        if (elementos.errorDocumento) elementos.errorDocumento.hidden = true;
        seleccionarBolsa(estado.bolsaSeleccionada.bolsa_ref, "");
      });
    }

    if (elementos.botonVolverBolsas) {
      elementos.botonVolverBolsas.addEventListener("click", volverABolsas);
    }

    if (elementos.botonReintentarBolsas) {
      elementos.botonReintentarBolsas.addEventListener("click", () => {
        const boton = elementos.botonReintentarBolsas;
        void cargarBolsas({ reintentoEnfocado: boton.ownerDocument?.activeElement === boton });
      });
    }

    if (elementos.botonReintentarLista) {
      elementos.botonReintentarLista.addEventListener("click", () => {
        if (estado.bolsaRefSolicitada) {
          const boton = elementos.botonReintentarLista;
          void seleccionarBolsa(estado.bolsaRefSolicitada, estado.filtroDocumento, estado.cursorSolicitado, {
            historial: "replace",
            reintentoEnfocado: boton.ownerDocument?.activeElement === boton,
          });
        }
      });
    }

    if (elementos.botonSiguiente) {
      elementos.botonSiguiente.addEventListener("click", () => {
        if (estado.bolsaSeleccionada && estado.cursorSiguiente) {
          seleccionarBolsa(
            estado.bolsaSeleccionada.bolsa_ref,
            estado.filtroDocumento,
            estado.cursorSiguiente,
            { historial: "replace" },
          );
        }
      });
    }

    if (ventana?.addEventListener) ventana.addEventListener("popstate", restaurarDesdeURL);
    restaurarDesdeURL();
  }

  return Object.freeze({
    estado,
    cargarBolsas,
    seleccionarBolsa,
    volverABolsas,
    restaurarDesdeURL,
    instalar,
  });
}

function prepararAyudaListas(documento) {
  const cabecera = documento.querySelector?.(".cabecera-seccion-publica");
  const introduccion = cabecera?.querySelector?.("p:not(.eyebrow)");
  const aviso = documento.querySelector?.(".aviso-privacidad-publica");
  if (!cabecera || !introduccion || !aviso) return;
  const ayuda = documento.createElement("details");
  ayuda.className = "ayuda-listas-publica";
  const resumen = documento.createElement("summary");
  resumen.textContent = "?";
  resumen.setAttribute("aria-label", t("ayuda_privacidad_listas"));
  resumen.title = t("ayuda_privacidad_listas");
  ayuda.append(resumen, introduccion, aviso);
  cabecera.after(ayuda);

  const textoFormato = documento.querySelector?.(".campo-documento .ayuda-busqueda-doc:not([id])");
  if (textoFormato) {
    const ayudaFormato = documento.createElement("details");
    ayudaFormato.className = "ayuda-formato-lista";
    const resumenFormato = documento.createElement("summary");
    resumenFormato.textContent = "?";
    resumenFormato.setAttribute("aria-label", t("ayuda_documento_lista"));
    resumenFormato.title = t("ayuda_documento_lista");
    ayudaFormato.append(resumenFormato, textoFormato);
    documento.querySelector(".campo-documento input")?.after(ayudaFormato);
  }
}

// Inicialización automática al cargar el documento
if (typeof document !== "undefined") {
  document.addEventListener("DOMContentLoaded", () => {
    prepararAyudaListas(document);
    const porId = (id) => document.getElementById(id);
    const elementos = {
      seccionBolsas: porId("seccion-bolsas"),
      seccionLista: porId("seccion-lista"),
      bolsasCargando: porId("bolsas-cargando"),
      bolsasError: porId("bolsas-error"),
      bolsasVacio: porId("bolsas-vacio"),
      mensajeErrorBolsas: porId("mensaje-error-bolsas"),
      cuerpoTablaBolsas: porId("cuerpo-tabla-bolsas"),
      botonReintentarBolsas: porId("reintentar-bolsas"),
      infoBolsaActiva: porId("info-bolsa-activa"),
      formularioBusqueda: porId("formulario-busqueda-lista"),
      inputDocumento: porId("filtro-documento"),
      errorDocumento: porId("error-formato-documento"),
      botonLimpiarBusqueda: porId("boton-limpiar-documento"),
      cuerpoTablaLista: porId("cuerpo-tabla-lista"),
      listaCargando: porId("lista-cargando"),
      listaError: porId("lista-error"),
      listaVacio: porId("lista-vacio"),
      mensajeErrorLista: porId("mensaje-error-lista"),
      botonReintentarLista: porId("reintentar-lista"),
      contenedorPaginacion: porId("paginacion-lista"),
      botonSiguiente: porId("boton-pagina-siguiente"),
      botonVolverBolsas: porId("boton-volver-bolsas"),
    };

    // Preferencias de accesibilidad volátiles (sin almacenamiento persistente)
    const alternarTexto = porId("alternar-texto");
    const alternarContraste = porId("alternar-contraste");
    if (alternarTexto) {
      alternarTexto.addEventListener("click", () => {
        const activa = document.body.classList.toggle("texto-grande");
        alternarTexto.setAttribute("aria-pressed", activa ? "true" : "false");
      });
    }
    if (alternarContraste) {
      alternarContraste.addEventListener("click", () => {
        const activa = document.body.classList.toggle("alto-contraste");
        alternarContraste.setAttribute("aria-pressed", activa ? "true" : "false");
      });
    }

    if (elementos.seccionBolsas && elementos.seccionLista) {
      const ctrl = crearControladorListaBolsas({ elementos });
      ctrl.instalar();
    }
  });
}
