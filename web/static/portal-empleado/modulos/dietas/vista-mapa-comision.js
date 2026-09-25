/**
 * Mapa de la ruta de una comisión en edición.
 *
 * La vista solo acepta códigos del catálogo y un cálculo ya autorizado por el
 * puerto de rutas. No conserva coordenadas, no crea geometrías y no reemplaza
 * la comprobación del alta del borrador.
 */
import {
  ATRIBUCION_OSM_INTERNA,
  ESQUEMA_SOLICITUD_RUTA_DIETAS,
  PLANTILLA_TESELAS_OSM_INTERNA,
  validarCalculoRutaDietas,
  validarSolicitudRutaDietas,
} from "./contrato.js";
import { MENSAJES_DIETAS_ES, crearTraductorDietas } from "./i18n.js?v=20260925-d6p2-v1";

const MAXIMO_LOCALIDADES = 12;

function nodo(documento, etiqueta, texto = "") {
  const resultado = documento.createElement(etiqueta);
  if (texto) resultado.textContent = texto;
  return resultado;
}

function sigueMontada(raiz, contenedor) {
  return raiz.querySelector?.("[data-dietas-mapa-comision]") === contenedor;
}

function retirar(raiz, contenedor) {
  if (!sigueMontada(raiz, contenedor)) return;
  if (typeof contenedor.remove === "function") contenedor.remove();
  else raiz.removeChild?.(contenedor);
}

function codigosRuta(entrada, permitirVacio = false) {
  if (!Array.isArray(entrada) || entrada.length > MAXIMO_LOCALIDADES
    || (!permitirVacio && entrada.length < 2)) {
    throw new TypeError("localidades de la ruta de Dietas no válidas");
  }
  return entrada.map((codigo) => {
    if (typeof codigo !== "string" || codigo !== codigo.trim() || !codigo) {
      throw new TypeError("localidad de la ruta de Dietas no válida");
    }
    return codigo;
  });
}

function mismaRuta(izquierda, derecha) {
  return izquierda.length === derecha.length
    && izquierda.every((codigo, indice) => codigo === derecha[indice]);
}

function descriptorMapa(alternativa) {
  return Object.freeze({
    vista_ref: "comision-ruta-calculada",
    proveedor: "openstreetmap",
    despliegue: "red_interna",
    plantilla_teselas: PLANTILLA_TESELAS_OSM_INTERNA,
    atribucion: ATRIBUCION_OSM_INTERNA,
    demostracion: false,
    geometria: alternativa.geometria,
  });
}

function crearMapa(documento, t) {
  const figura = nodo(documento, "figure");
  figura.className = "dietas-mapa";
  figura.dataset.dietasMapaComisionCanvas = "";
  const lienzo = nodo(documento, "div");
  lienzo.className = "dietas-mapa-canvas";
  lienzo.dataset.dietasMapaCanvas = "";
  lienzo.setAttribute("role", "region");
  lienzo.setAttribute("aria-label", t("mapa_region_accesible"));
  const estado = nodo(documento, "p", t("mapa_cargando_interno"));
  estado.dataset.dietasMapaEstado = "";
  estado.setAttribute("role", "status");
  estado.setAttribute("aria-live", "polite");
  const atribucion = nodo(documento, "small", ATRIBUCION_OSM_INTERNA);
  atribucion.dataset.dietasMapaAtribucion = "";
  atribucion.hidden = true;
  figura.append(nodo(documento, "figcaption", t("mapa_titulo")), lienzo, estado, atribucion);
  return figura;
}

/**
 * Monta el resultado cartográfico de la comisión.
 *
 * `obtenerCalculoParaGuardar(codigos)` es la guardia que debe usar la vista
 * del formulario antes de enviar el borrador: solo devuelve un cálculo del
 * mismo itinerario, con geometría OSRM interna validada.
 */
export async function montarVistaMapaComisionDietas({
  raiz,
  calculador,
  visorRuta,
  codigos = [],
  anunciar = () => {},
  registrarDesmontar,
  mensajes = MENSAJES_DIETAS_ES,
} = {}) {
  if (!raiz?.append || !raiz?.querySelector || !raiz.ownerDocument
    || typeof calculador?.obtenerCatalogo !== "function" || typeof calculador?.calcular !== "function"
    || typeof visorRuta?.montar !== "function" || typeof anunciar !== "function"
    || typeof registrarDesmontar !== "undefined" && typeof registrarDesmontar !== "function") {
    throw new TypeError("vista de mapa de comisión de Dietas no disponible");
  }
  const documento = raiz.ownerDocument;
  const t = crearTraductorDietas(mensajes);
  const contenedor = nodo(documento, "section");
  contenedor.className = "panel dietas-mapa-comision";
  contenedor.dataset.dietasMapaComision = "";
  raiz.append(contenedor);

  let activa = true;
  let catalogo = null;
  let ruta = codigosRuta(codigos, true);
  let calculo = null;
  let controlador = new AbortController();
  let visorActivo = null;
  let calculando = false;
  let operacion = 0;
  let mensaje = "ruta_pendiente_calculo";
  let tono = "informacion";
  const activaAhora = () => activa && sigueMontada(raiz, contenedor);
  const pararMapa = () => { visorActivo?.desmontar?.(); visorActivo = null; };
  const cancelar = () => {
    controlador?.abort();
    controlador = null;
    calculando = false;
    operacion += 1;
  };
  const desmontar = () => {
    if (!activa) return;
    activa = false;
    cancelar();
    pararMapa();
    retirar(raiz, contenedor);
  };
  registrarDesmontar?.(desmontar);

  function pintar() {
    if (!activaAhora()) return;
    pararMapa();
    contenedor.replaceChildren();
    const cabecera = nodo(documento, "div");
    cabecera.className = "cabecera-panel";
    cabecera.append(nodo(documento, "h3", t("ruta_del_dia")));
    const ayuda = nodo(documento, "button", "?");
    ayuda.type = "button";
    ayuda.className = "boton-terciario";
    ayuda.disabled = true;
    ayuda.setAttribute("aria-label", t("recorridos_abrir_ayuda"));
    cabecera.append(ayuda);
    const estado = nodo(documento, "p", t(mensaje));
    estado.dataset.dietasMapaComisionEstado = tono;
    estado.setAttribute("role", tono === "error" ? "alert" : "status");
    estado.setAttribute("aria-live", "polite");
    contenedor.append(cabecera, estado);
    if (!calculo) return;
    const alternativa = calculo.alternativas.find((item) => item.recomendada) || calculo.alternativas[0];
    const mapa = crearMapa(documento, t);
    contenedor.append(mapa);
    if (activaAhora()) visorActivo = visorRuta.montar({ raiz: mapa, descriptor: descriptorMapa(alternativa) });
  }

  async function cargarCatalogo() {
    const miOperacion = ++operacion;
    const signal = controlador.signal;
    try {
      catalogo = await calculador.obtenerCatalogo({ signal });
      if (!activaAhora() || signal.aborted || miOperacion !== operacion) return false;
      mensaje = "ruta_pendiente_calculo";
      tono = "informacion";
      pintar();
      return true;
    } catch {
      if (!activaAhora() || signal.aborted || miOperacion !== operacion) return false;
      mensaje = "ruta_error_servicio";
      tono = "error";
      anunciar(t(mensaje), tono);
      pintar();
      return false;
    } finally {
      if (miOperacion === operacion) controlador = null;
    }
  }

  function establecerCodigos(siguientes) {
    const normalizados = codigosRuta(siguientes, true);
    if (mismaRuta(ruta, normalizados)) return false;
    // El catálogo no depende de las localidades seleccionadas. Dejar que su
    // GET termine evita que un cambio temprano de formulario convierta la
    // vista en un estado permanentemente no disponible; el POST sí se aborta.
    if (calculando) cancelar();
    ruta = normalizados;
    calculo = null;
    mensaje = "ruta_pendiente_calculo";
    tono = "informacion";
    pintar();
    return true;
  }

  async function calcular() {
    if (!catalogo) throw new Error(t("ruta_puerto_no_disponible"));
    cancelar();
    controlador = new AbortController();
    calculando = true;
    const signal = controlador.signal;
    const miOperacion = ++operacion;
    const solicitud = validarSolicitudRutaDietas({
      esquema: ESQUEMA_SOLICITUD_RUTA_DIETAS,
      paradas: ruta,
      alternativas: 3,
    }, catalogo);
    mensaje = "mapa_cargando_interno";
    tono = "informacion";
    pintar();
    try {
      const resultado = await calculador.calcular(solicitud, { signal });
      if (!activaAhora() || signal.aborted || miOperacion !== operacion) return null;
      calculo = validarCalculoRutaDietas(resultado, solicitud);
      if (calculo.demostracion || calculo.motor !== "osrm_interno") {
        throw new Error("el cálculo de la comisión no procede del OSRM interno");
      }
      mensaje = "ruta_calculo_completado";
      tono = "exito";
      anunciar(t(mensaje), tono);
      pintar();
      return calculo;
    } catch (error) {
      if (!activaAhora() || signal.aborted || miOperacion !== operacion) return null;
      calculo = null;
      mensaje = "ruta_error_servicio";
      tono = "error";
      anunciar(t(mensaje), tono);
      pintar();
      throw error;
    } finally {
      if (miOperacion === operacion) {
        controlador = null;
        calculando = false;
      }
    }
  }

  function obtenerCalculoParaGuardar(codigosEsperados = ruta) {
    const esperados = codigosRuta(codigosEsperados);
    if (!calculo || !mismaRuta(ruta, esperados)) {
      throw new Error(t("ruta_calcular_antes_guardar"));
    }
    const solicitud = validarSolicitudRutaDietas({
      esquema: ESQUEMA_SOLICITUD_RUTA_DIETAS,
      paradas: esperados,
      alternativas: 3,
    }, catalogo);
    const validado = validarCalculoRutaDietas(calculo, solicitud);
    if (validado.demostracion || validado.motor !== "osrm_interno") {
      throw new Error(t("ruta_calcular_antes_guardar"));
    }
    return validado;
  }

  pintar();
  await cargarCatalogo();
  return Object.freeze({ establecerCodigos, calcular, obtenerCalculoParaGuardar, desmontar });
}
