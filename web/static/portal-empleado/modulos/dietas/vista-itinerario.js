/**
 * Herramienta aislada de itinerarios para la presentación de Dietas.
 *
 * Solo consume el catálogo y el puerto de rutas inyectado. No conoce gastos,
 * borradores, aprobaciones, liquidaciones, recibos ni pagos.
 */
import { crearTraductorDietas, MENSAJES_DIETAS_ES } from "./i18n.js";
import { crearPresentadorRutasDietas } from "./presentador-rutas.js";

function elemento(documento, etiqueta, texto = "") {
  const nodo = documento.createElement(etiqueta);
  if (texto) nodo.textContent = texto;
  return nodo;
}

function sigueMontada(raiz, contenedor) {
  return raiz.querySelector?.("[data-dietas-itinerario]") === contenedor;
}

function retirar(raiz, contenedor) {
  if (!sigueMontada(raiz, contenedor)) return;
  if (typeof contenedor.remove === "function") contenedor.remove();
  else raiz.removeChild?.(contenedor);
}

function crearMapa(documento, modelo, traducir) {
  if (!modelo.mapa_ruta) return null;
  const figura = elemento(documento, "figure");
  figura.className = "dietas-mapa";
  figura.dataset.dietasMapaRef = modelo.mapa_ruta.vista_ref;
  const lienzo = elemento(documento, "div");
  lienzo.className = "dietas-mapa-canvas";
  lienzo.dataset.dietasMapaCanvas = "";
  lienzo.setAttribute("role", "region");
  lienzo.setAttribute("aria-label", traducir("mapa_region_accesible"));
  const estado = elemento(documento, "p", traducir("mapa_cargando_interno"));
  estado.dataset.dietasMapaEstado = "";
  estado.setAttribute("role", "status");
  estado.setAttribute("aria-live", "polite");
  const atribucion = elemento(documento, "small", modelo.mapa_ruta.atribucion);
  atribucion.dataset.dietasMapaAtribucion = "";
  atribucion.hidden = true;
  figura.append(elemento(documento, "figcaption", traducir("mapa_titulo")), lienzo, estado, atribucion);
  return figura;
}

/** Monta una consulta no liquidable de rutas internas de Dietas. */
export async function montarVistaItinerarioDietas({
  raiz,
  calculador,
  visorRuta,
  anunciar = () => {},
  mensajes = MENSAJES_DIETAS_ES,
  registrarDesmontar,
} = {}) {
  if (!raiz?.append || !calculador?.obtenerCatalogo || !calculador?.calcular
    || !visorRuta?.montar || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) {
    throw new TypeError("vista de itinerario no disponible");
  }
  const documento = raiz.ownerDocument;
  if (!documento?.createElement) throw new TypeError("documento de itinerario no disponible");

  const traducir = crearTraductorDietas(mensajes);
  const contenedor = elemento(documento, "div");
  contenedor.className = "modulo-dietas";
  contenedor.dataset.dietasItinerario = "";
  raiz.append(contenedor);

  let activa = true;
  let controlador = new AbortController();
  let visorActivo = null;
  let presentador = null;
  let errorVisible = "";
  const activaAhora = () => activa && sigueMontada(raiz, contenedor);
  const pararMapa = () => { visorActivo?.desmontar?.(); visorActivo = null; };
  const desmontar = () => {
    if (!activa) return;
    activa = false;
    controlador?.abort();
    pararMapa();
    contenedor.removeEventListener("change", cambio);
    contenedor.removeEventListener("click", clic);
    retirar(raiz, contenedor);
  };
  // Se entrega antes de esperar el catálogo: el coordinador puede cancelar una
  // navegación sustituida aunque el puerto todavía no haya respondido.
  registrarDesmontar?.(desmontar);

  try {
    const catalogo = await calculador.obtenerCatalogo({ signal: controlador.signal });
    if (activaAhora()) {
      presentador = crearPresentadorRutasDietas({
        catalogo,
        permisos: { consultarRutas: true, gestionarRutas: true },
        demostracion: true,
      });
    }
  } catch {
    if (activaAhora()) {
      errorVisible = traducir("ruta_error_servicio");
      anunciar(errorVisible, "error");
    }
  } finally {
    controlador = null;
  }

  function pintar() {
    if (!activaAhora()) return;
    pararMapa();
    contenedor.replaceChildren();
    const panel = elemento(documento, "section");
    panel.className = "panel dietas-itinerario";
    panel.append(
      elemento(documento, "h2", traducir("ruta_del_dia")),
      elemento(documento, "p", traducir("ruta_no_liquidable")),
      elemento(documento, "p", traducir("ruta_ayuda")),
    );
    if (!presentador) {
      const alerta = elemento(documento, "p", errorVisible || traducir("ruta_error_servicio"));
      alerta.dataset.itinerarioError = "";
      alerta.setAttribute("role", "alert");
      panel.append(alerta);
      contenedor.append(panel);
      return;
    }
    const modelo = presentador.obtenerModelo();
    const resumenCatalogo = elemento(documento, "p", traducir("ruta_catalogo_resumen", {
      total: modelo.catalogo.puntos.length, version: modelo.catalogo.version,
    }));
    resumenCatalogo.dataset.itinerarioCatalogo = "";
    panel.append(resumenCatalogo);
    const paradas = elemento(documento, "div");
    paradas.className = "dietas-ruta-paradas";
    modelo.paradas.forEach((codigo, indice) => {
      const etiquetaTexto = indice === 0 ? traducir("ruta_salida")
        : indice === modelo.paradas.length - 1 ? traducir("ruta_destino_final")
          : traducir("ruta_parada_intermedia", { numero: indice });
      const etiqueta = elemento(documento, "label", etiquetaTexto);
      const selector = elemento(documento, "select");
      selector.dataset.itinerarioParada = String(indice);
      selector.setAttribute("aria-label", etiquetaTexto);
      modelo.catalogo.puntos.forEach((punto) => {
        const opcion = elemento(documento, "option", punto.nombre);
        opcion.value = punto.codigo;
        opcion.selected = punto.codigo === codigo;
        selector.append(opcion);
      });
      etiqueta.append(selector);
      paradas.append(etiqueta);
    });
    const calcular = elemento(documento, "button", traducir("ruta_calcular_osrm"));
    calcular.type = "button";
    calcular.dataset.itinerarioCalcular = "";
    calcular.disabled = controlador !== null;
    panel.append(paradas, calcular);
    if (errorVisible) {
      const alerta = elemento(documento, "p", errorVisible);
      alerta.dataset.itinerarioError = "";
      alerta.setAttribute("role", "alert");
      panel.append(alerta);
    }
    if (modelo.calculado) {
      const resultado = elemento(documento, "section");
      resultado.className = "dietas-ruta-resultado";
      resultado.append(
        elemento(documento, "h3", traducir("ruta_resultado")),
        elemento(documento, "p", traducir("ruta_motor_version", {
          motor: modelo.motor, version: modelo.version_grafo,
        })),
        elemento(documento, "p", `${traducir("ruta_km_base")}: ${new Intl.NumberFormat("es-ES", {
          style: "unit", unit: "kilometer", unitDisplay: "short", maximumFractionDigits: 1,
        }).format(modelo.kilometros_base)}`),
        elemento(documento, "p", `${traducir("ruta_duracion")}: ${new Intl.NumberFormat("es-ES", {
          style: "unit", unit: "minute", unitDisplay: "short",
        }).format(modelo.duracion_minutos)}`),
      );
      const mapa = crearMapa(documento, modelo, traducir);
      if (mapa) resultado.append(mapa);
      panel.append(resultado);
      contenedor.append(panel);
      if (mapa && activaAhora()) visorActivo = visorRuta.montar({ raiz: mapa, descriptor: modelo.mapa_ruta });
      return;
    }
    contenedor.append(panel);
  }

  function aplicar(operacion) {
    try {
      errorVisible = "";
      operacion();
    } catch {
      errorVisible = traducir("ruta_error_operacion");
      anunciar(errorVisible, "error");
    }
    pintar();
  }
  function cambio(evento) {
    const selector = evento.target.closest?.("[data-itinerario-parada]");
    if (!selector || !presentador) return;
    aplicar(() => presentador.establecerParada(Number(selector.dataset.itinerarioParada), selector.value));
  }
  async function clic(evento) {
    if (!evento.target.closest?.("[data-itinerario-calcular]") || !presentador || controlador) return;
    controlador = new AbortController();
    pintar();
    try {
      const solicitud = presentador.prepararSolicitudCalculo();
      const calculo = await calculador.calcular(solicitud, { signal: controlador.signal });
      if (activaAhora() && !controlador.signal.aborted) {
        presentador.registrarCalculo(calculo);
        errorVisible = "";
        anunciar(traducir("ruta_calculo_completado"));
      }
    } catch {
      if (activaAhora() && !controlador.signal.aborted) {
        errorVisible = traducir("ruta_error_servicio");
        anunciar(errorVisible, "error");
      }
    } finally {
      controlador = null;
      pintar();
    }
  }
  contenedor.addEventListener("change", cambio);
  contenedor.addEventListener("click", clic);
  pintar();
  return Object.freeze({ desmontar });
}
