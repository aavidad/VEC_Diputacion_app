/**
 * Controlador de la interfaz pública de bolsas y listas de aspirantes (B10).
 * Consulta anónima sin autenticación y sin almacenamiento web.
 */

import {
  consultarBolsasPublicas,
  consultarListaBolsaPublica,
} from "./lista-bolsas-api.js";
import { PATRON_DOCUMENTO_ENMASCARADO } from "./contrato-publico-bolsas.js";

const ETIQUETAS_ESTADO = Object.freeze({
  disponible: "Disponible",
  ocupado: "Ocupado / Nombrado",
  no_disponible: "No disponible (pausa)",
  excluido: "Excluido",
  renuncia_pendiente: "Renuncia en trámite",
});

function escaparHTML(valor) {
  return String(valor ?? "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

function formatoFecha(iso) {
  if (!iso) return "—";
  try {
    const d = new Date(iso);
    return isNaN(d.getTime()) ? String(iso) : d.toLocaleDateString("es-ES", { dateStyle: "medium" });
  } catch {
    return String(iso);
  }
}

export function crearControladorListaBolsas({
  elementos,
  api = { consultarBolsasPublicas, consultarListaBolsaPublica },
}) {
  const estado = {
    bolsas: [],
    bolsaSeleccionada: null,
    posiciones: [],
    hayMas: false,
    cursorSiguiente: null,
    filtroDocumento: "",
    cargando: false,
  };

  function mostrarPanel(panelAMostrar) {
    const paneles = [
      elementos.seccionBolsas,
      elementos.seccionLista,
      elementos.bolsasCargando,
      elementos.bolsasError,
      elementos.bolsasVacio,
      elementos.listaCargando,
      elementos.listaError,
      elementos.listaVacio,
    ];
    for (const p of paneles) {
      if (p) p.hidden = true;
    }
    if (panelAMostrar) panelAMostrar.hidden = false;
  }

  function etiquetaEstado(clave) {
    return ETIQUETAS_ESTADO[clave] || clave;
  }

  async function cargarBolsas() {
    estado.cargando = true;
    elementos.seccionBolsas.hidden = false;
    elementos.seccionLista.hidden = true;
    elementos.bolsasCargando.hidden = false;
    elementos.bolsasError.hidden = true;
    elementos.bolsasVacio.hidden = true;
    if (elementos.cuerpoTablaBolsas) elementos.cuerpoTablaBolsas.innerHTML = "";

    try {
      const respuesta = await api.consultarBolsasPublicas();
      estado.bolsas = respuesta.bolsas;
      elementos.bolsasCargando.hidden = true;

      if (estado.bolsas.length === 0) {
        elementos.bolsasVacio.hidden = false;
        return;
      }

      renderizarTablaBolsas(estado.bolsas);
    } catch (err) {
      elementos.bolsasCargando.hidden = true;
      elementos.bolsasError.hidden = false;
      if (elementos.mensajeErrorBolsas) {
        elementos.mensajeErrorBolsas.textContent = err.message || "Error al consultar bolsas";
      }
    } finally {
      estado.cargando = false;
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
            Consultar lista
          </button>
        </td>
      </tr>
    `).join("");
  }

  async function seleccionarBolsa(bolsaRef, documento = "", cursor = "") {
    estado.cargando = true;
    estado.filtroDocumento = documento;
    elementos.seccionBolsas.hidden = true;
    elementos.seccionLista.hidden = false;
    elementos.listaError.hidden = true;
    elementos.listaVacio.hidden = true;

    if (!cursor) {
      estado.posiciones = [];
      if (elementos.cuerpoTablaLista) elementos.cuerpoTablaLista.innerHTML = "";
      elementos.listaCargando.hidden = false;
    }

    try {
      const respuesta = await api.consultarListaBolsaPublica({
        bolsa_ref: bolsaRef,
        documento,
        cursor,
      });

      estado.bolsaSeleccionada = respuesta.bolsa;
      estado.hayMas = respuesta.hay_mas;
      estado.cursorSiguiente = respuesta.cursor_siguiente;

      if (cursor) {
        estado.posiciones = [...estado.posiciones, ...respuesta.posiciones];
      } else {
        estado.posiciones = respuesta.posiciones;
      }

      elementos.listaCargando.hidden = true;
      renderizarDetalleBolsa(estado.bolsaSeleccionada);
      renderizarTablaLista(estado.posiciones);

      if (estado.posiciones.length === 0) {
        elementos.listaVacio.hidden = false;
      }

      actualizarPaginacion();
      actualizarURL(bolsaRef, documento);
    } catch (err) {
      elementos.listaCargando.hidden = true;
      elementos.listaError.hidden = false;
      if (elementos.mensajeErrorLista) {
        elementos.mensajeErrorLista.textContent = err.message || "Error al consultar la lista";
      }
    } finally {
      estado.cargando = false;
    }
  }

  function renderizarDetalleBolsa(bolsa) {
    if (!elementos.infoBolsaActiva || !bolsa) return;
    elementos.infoBolsaActiva.innerHTML = `
      <div class="tarjeta-bolsa-activa__info">
        <h2>${escaparHTML(bolsa.categoria)}</h2>
        <div class="tarjeta-bolsa-activa__meta">
          <span><strong>Grupos:</strong> ${escaparHTML((bolsa.grupos || []).join(", "))}</span>
          <span><strong>Tipo:</strong> ${escaparHTML(bolsa.tipo_lista)}</span>
          <span><strong>Vigente desde:</strong> ${escaparHTML(formatoFecha(bolsa.vigente_desde))}</span>
          <span><strong>Total aspirantes:</strong> ${Number(bolsa.total || 0).toLocaleString("es-ES")}</span>
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

  function actualizarURL(bolsaRef, documento) {
    if (typeof window === "undefined" || !window.history || !window.history.replaceState) return;
    const url = new URL(window.location.href);
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
    window.history.replaceState({}, "", url.toString());
  }

  function volverABolsas() {
    estado.bolsaSeleccionada = null;
    estado.posiciones = [];
    estado.filtroDocumento = "";
    if (elementos.inputDocumento) elementos.inputDocumento.value = "";
    if (elementos.errorDocumento) elementos.errorDocumento.hidden = true;
    actualizarURL("", "");
    cargarBolsas();
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
            elementos.errorDocumento.textContent = "El documento debe tener formato ***1234** (3 asteriscos, 4 dígitos y 2 asteriscos).";
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
      elementos.botonReintentarBolsas.addEventListener("click", cargarBolsas);
    }

    if (elementos.botonReintentarLista) {
      elementos.botonReintentarLista.addEventListener("click", () => {
        if (estado.bolsaSeleccionada) {
          seleccionarBolsa(estado.bolsaSeleccionada.bolsa_ref, estado.filtroDocumento);
        }
      });
    }

    if (elementos.botonSiguiente) {
      elementos.botonSiguiente.addEventListener("click", () => {
        if (estado.bolsaSeleccionada && estado.cursorSiguiente) {
          seleccionarBolsa(
            estado.bolsaSeleccionada.bolsa_ref,
            estado.filtroDocumento,
            estado.cursorSiguiente
          );
        }
      });
    }

    // Comprobar parámetros iniciales en la URL
    if (typeof window !== "undefined" && window.location) {
      const parametros = new URLSearchParams(window.location.search);
      const bolsaRef = parametros.get("bolsa");
      const documento = parametros.get("documento") || "";
      if (bolsaRef) {
        if (documento && PATRON_DOCUMENTO_ENMASCARADO.test(documento)) {
          if (elementos.inputDocumento) elementos.inputDocumento.value = documento;
        }
        seleccionarBolsa(bolsaRef, documento);
      } else {
        cargarBolsas();
      }
    } else {
      cargarBolsas();
    }
  }

  return Object.freeze({
    estado,
    cargarBolsas,
    seleccionarBolsa,
    volverABolsas,
    instalar,
  });
}

// Inicialización automática al cargar el documento
if (typeof document !== "undefined") {
  document.addEventListener("DOMContentLoaded", () => {
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
