import { crearTraductorDietas, MENSAJES_DIETAS_ES } from "./i18n.js";
import { crearVisorRutaDietas } from "./mapa-ruta.js";
import { crearPresentadorRutasDietas } from "./presentador-rutas.js";

function elemento(documento, etiqueta, texto = "") {
  const nodo = documento.createElement(etiqueta);
  if (texto) nodo.textContent = texto;
  return nodo;
}

function numero(valor, decimales = 0) {
  return new Intl.NumberFormat("es-ES", {
    minimumFractionDigits: decimales,
    maximumFractionDigits: decimales,
  }).format(Number(valor || 0));
}

function sigueMontada(raiz, contenedor) {
  return raiz.querySelector?.("[data-dietas-itinerario]") === contenedor;
}

function retirarContenedor(raiz, contenedor) {
  if (!sigueMontada(raiz, contenedor)) return;
  if (typeof contenedor.remove === "function") {
    contenedor.remove();
  } else if (typeof raiz.removeChild === "function") {
    raiz.removeChild(contenedor);
  }
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

/**
 * Consulta y presenta itinerarios internos. La vista no crea gastos ni efectos
 * económicos, y solo elimina el contenedor del que es propietaria.
 */
export async function montarVistaItinerarioDietas({
  raiz,
  calculador,
  anunciar = () => {},
  mensajes = MENSAJES_DIETAS_ES,
  visorRuta,
  entorno = globalThis,
} = {}) {
  if (!raiz?.append || !calculador?.obtenerCatalogo || !calculador?.calcular) {
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
  let abortador = new AbortController();
  let visorActivo = null;
  let errorVisible = "";

  const pararMapa = () => {
    visorActivo?.desmontar?.();
    visorActivo = null;
  };
  const sigueActiva = () => activa && sigueMontada(raiz, contenedor);
  const desmontar = () => {
    if (!activa) return;
    activa = false;
    abortador?.abort();
    pararMapa();
    contenedor.removeEventListener("change", cambio);
    contenedor.removeEventListener("click", clic);
    retirarContenedor(raiz, contenedor);
  };

  let presentador;
  try {
    const catalogo = await calculador.obtenerCatalogo({ signal: abortador.signal });
    if (!sigueActiva()) return Object.freeze({ desmontar });
    presentador = crearPresentadorRutasDietas({
      catalogo,
      permisos: { consultarRutas: true, gestionarRutas: true },
      demostracion: false,
    });
  } catch (_error) {
    if (!sigueActiva()) return Object.freeze({ desmontar });
    errorVisible = traducir("ruta_error_servicio");
    anunciar(errorVisible, "error");
  } finally {
    abortador = null;
  }

  const visor = visorRuta || crearVisorRutaDietas({ entorno, permitirTeselas: true, mensajes });

  function pintar() {
    if (!sigueActiva()) return;
    pararMapa();
    contenedor.replaceChildren();
    const panel = elemento(documento, "section");
    panel.className = "panel dietas-itinerario";
    panel.setAttribute("aria-busy", String(Boolean(abortador)));
    panel.append(elemento(documento, "h2", traducir("ruta_itinerario_ida_vuelta")));
    const aviso = elemento(documento, "p", traducir("ruta_no_liquidable"));
    aviso.setAttribute("role", "status");
    panel.append(aviso);

    if (!presentador) {
      const alerta = elemento(documento, "p", errorVisible || traducir("ruta_error_servicio"));
      alerta.dataset.itinerarioError = "";
      alerta.setAttribute("role", "alert");
      panel.append(alerta);
      contenedor.append(panel);
      return;
    }

    const modelo = presentador.obtenerModelo();
    const paradas = elemento(documento, "div");
    paradas.className = "dietas-ruta-paradas";
    modelo.paradas.forEach((codigo, indice) => {
      const grupo = elemento(documento, "div");
      grupo.className = "dietas-ruta-parada";
      const nombre = indice === 0 ? traducir("ruta_salida")
        : indice === modelo.paradas.length - 1 ? traducir("ruta_destino_final")
          : traducir("ruta_parada_intermedia", { numero: indice });
      const etiqueta = elemento(documento, "label", nombre);
      const selector = elemento(documento, "select");
      selector.dataset.itinerarioParada = String(indice);
      selector.setAttribute("aria-label", nombre);
      if (codigo === "") {
        const vacia = elemento(documento, "option", traducir("ruta_seleccionar_localidad"));
        vacia.value = "";
        vacia.selected = true;
        selector.append(vacia);
      }
      modelo.catalogo.puntos.forEach((punto) => {
        const opcion = elemento(documento, "option", punto.nombre);
        opcion.value = punto.codigo;
        opcion.selected = punto.codigo === codigo;
        selector.append(opcion);
      });
      etiqueta.append(selector);
      grupo.append(etiqueta);
      if (indice > 0 && indice < modelo.paradas.length - 1) {
        const quitar = elemento(documento, "button", traducir("ruta_quitar_parada"));
        quitar.type = "button";
        quitar.dataset.itinerarioQuitar = String(indice);
        grupo.append(quitar);
      }
      paradas.append(grupo);
    });
    panel.append(paradas);

    const acciones = elemento(documento, "div");
    const anadir = elemento(documento, "button", traducir("ruta_anadir_parada"));
    anadir.type = "button";
    anadir.dataset.itinerarioAnadir = "";
    const calcular = elemento(documento, "button", traducir("ruta_calcular_osrm"));
    calcular.type = "button";
    calcular.dataset.itinerarioCalcular = "";
    calcular.disabled = Boolean(abortador);
    acciones.append(anadir, calcular);
    panel.append(acciones, elemento(documento, "p", traducir("ruta_ayuda")));

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
        elemento(documento, "p", traducir("ruta_motor_version", { motor: modelo.motor, version: modelo.version_grafo })),
        elemento(documento, "p", `${traducir("ruta_km_total")}: ${numero(modelo.kilometros_total, 1)} km · ${traducir("ruta_duracion")}: ${numero(modelo.duracion_minutos)} min`),
      );
      const alternativas = elemento(documento, "div");
      alternativas.setAttribute("aria-label", traducir("ruta_alternativas"));
      modelo.alternativas.forEach((alternativa) => {
        const articulo = elemento(documento, "article");
        articulo.className = "dietas-ruta-alternativa";
        articulo.append(
          elemento(documento, "strong", alternativa.etiqueta),
          elemento(documento, "span", `${numero(alternativa.kilometros, 1)} km · ${numero(alternativa.duracion_minutos)} min`),
        );
        if (!alternativa.recomendada) {
          const motivo = elemento(documento, "input");
          motivo.dataset.itinerarioMotivoAlternativa = alternativa.referencia;
          motivo.value = alternativa.seleccionada ? modelo.motivo_alternativa : "";
          motivo.maxLength = 500;
          motivo.setAttribute("aria-label", traducir("ruta_motivo_alternativa"));
          articulo.append(motivo);
        }
        const usar = elemento(documento, "button", alternativa.recomendada ? traducir("ruta_usar_recomendada") : traducir("ruta_usar_alternativa"));
        usar.type = "button";
        usar.dataset.itinerarioAlternativa = alternativa.referencia;
        usar.setAttribute("aria-pressed", String(alternativa.seleccionada));
        articulo.append(usar);
        alternativas.append(articulo);
      });
      resultado.append(alternativas);

      const tabla = elemento(documento, "table");
      const cuerpo = elemento(documento, "tbody");
      tabla.append(elemento(documento, "caption", traducir("ruta_caption_tramos")));
      modelo.tramos.forEach((tramo) => {
        const fila = elemento(documento, "tr");
        fila.dataset.itinerarioTramo = String(tramo.indice);
        fila.append(
          elemento(documento, "th", `${tramo.origen_nombre} → ${tramo.destino_nombre}`),
          elemento(documento, "td", `${numero(tramo.kilometros, 1)} km · ${numero(tramo.duracion_minutos)} min`),
        );
        const kilometros = elemento(documento, "input");
        kilometros.type = "number";
        kilometros.min = "0";
        kilometros.max = "1000";
        kilometros.step = "0.1";
        kilometros.value = String(tramo.ajuste_kilometros);
        kilometros.dataset.itinerarioAjusteKm = String(tramo.indice);
        kilometros.setAttribute("aria-label", traducir("ruta_ajuste_km"));
        const motivo = elemento(documento, "input");
        motivo.value = tramo.motivo_ajuste;
        motivo.maxLength = 500;
        motivo.dataset.itinerarioMotivoAjuste = String(tramo.indice);
        motivo.setAttribute("aria-label", traducir("ruta_motivo_ajuste"));
        const aplicar = elemento(documento, "button", traducir("ruta_aplicar_ajuste"));
        aplicar.type = "button";
        aplicar.dataset.itinerarioAplicarAjuste = String(tramo.indice);
        for (const control of [kilometros, motivo, aplicar]) {
          const celda = elemento(documento, "td");
          celda.append(control);
          fila.append(celda);
        }
        cuerpo.append(fila);
      });
      tabla.append(cuerpo);
      resultado.append(tabla);
      const mapa = crearMapa(documento, modelo, traducir);
      if (mapa) resultado.append(mapa);
      panel.append(resultado);
      contenedor.append(panel);
      if (mapa && sigueActiva()) visorActivo = visor.montar({ raiz: mapa, descriptor: modelo.mapa_ruta });
      return;
    }
    contenedor.append(panel);
  }

  function valor(selector) {
    return contenedor.querySelector(selector)?.value || "";
  }

  function aplicarOperacion(operacion, mensaje) {
    try {
      errorVisible = "";
      operacion();
      pintar();
      anunciar(mensaje);
    } catch (_error) {
      errorVisible = traducir("ruta_error_operacion");
      pintar();
      anunciar(errorVisible, "error");
    }
  }

  function cambio(evento) {
    const selector = evento.target.closest?.("[data-itinerario-parada]");
    if (!selector) return;
    aplicarOperacion(
      () => presentador.establecerParada(Number(selector.dataset.itinerarioParada), selector.value),
      traducir("ruta_pendiente_calculo"),
    );
  }

  async function clic(evento) {
    const objetivo = evento.target;
    const anadir = objetivo.closest?.("[data-itinerario-anadir]");
    const quitar = objetivo.closest?.("[data-itinerario-quitar]");
    const alternativa = objetivo.closest?.("[data-itinerario-alternativa]");
    const ajuste = objetivo.closest?.("[data-itinerario-aplicar-ajuste]");
    if (anadir) return aplicarOperacion(() => presentador.agregarParada(), traducir("ruta_parada_anadida"));
    if (quitar) return aplicarOperacion(() => presentador.eliminarParada(Number(quitar.dataset.itinerarioQuitar)), traducir("ruta_parada_eliminada"));
    if (alternativa) {
      return aplicarOperacion(
        () => presentador.seleccionarAlternativa(alternativa.dataset.itinerarioAlternativa, valor(`[data-itinerario-motivo-alternativa="${alternativa.dataset.itinerarioAlternativa}"]`)),
        traducir("ruta_alternativa_aplicada"),
      );
    }
    if (ajuste) {
      const indice = ajuste.dataset.itinerarioAplicarAjuste;
      return aplicarOperacion(
        () => presentador.ajustarTramo(Number(indice), valor(`[data-itinerario-ajuste-km="${indice}"]`), valor(`[data-itinerario-motivo-ajuste="${indice}"]`)),
        traducir("ruta_ajuste_aplicado"),
      );
    }
    if (!objetivo.closest?.("[data-itinerario-calcular]") || !presentador) return;
    abortador?.abort();
    abortador = new AbortController();
    pintar();
    try {
      presentador.registrarCalculo(await calculador.calcular(
        presentador.prepararSolicitudCalculo(), { signal: abortador.signal },
      ));
      errorVisible = "";
      anunciar(traducir("ruta_calculo_completado"));
    } catch (_error) {
      if (!abortador.signal.aborted) {
        errorVisible = traducir("ruta_error_servicio");
        anunciar(errorVisible, "error");
      }
    } finally {
      abortador = null;
      pintar();
    }
  }

  contenedor.addEventListener("change", cambio);
  contenedor.addEventListener("click", clic);
  pintar();
  return Object.freeze({ desmontar });
}
