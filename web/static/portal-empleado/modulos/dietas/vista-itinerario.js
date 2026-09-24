/**
 * Herramienta aislada de itinerarios para la presentación de Dietas.
 *
 * Solo consume el catálogo y el puerto de rutas inyectado. No conoce gastos,
 * borradores, aprobaciones, liquidaciones, recibos ni pagos.
 */
import { MENSAJES_DIETAS_ES } from "./i18n.js?v=20260924-dietas-d1d2d4";
import { crearTraductorDietasD4 } from "./i18n-d4.js?v=20260924-dietas-d1d2d4";
import { crearPresentadorRutasDietas } from "./presentador-rutas.js";
import { ESTILOS_TRAMO_RUTA_DIETAS } from "./mapa-ruta.js";

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
  const leyenda = elemento(documento, "ol");
  leyenda.className = "dietas-mapa-leyenda";
  leyenda.dataset.dietasMapaLeyenda = "";
  modelo.tramos.forEach((tramo, indice) => {
    const estilo = ESTILOS_TRAMO_RUTA_DIETAS[indice % ESTILOS_TRAMO_RUTA_DIETAS.length];
    const item = elemento(documento, "li", `${tramo.origen_nombre} → ${tramo.destino_nombre} · ${formatoKilometros(tramo.kilometros)} · ${estilo.patron}`);
    item.dataset.dietasMapaTramo = String(indice);
    item.setAttribute("style", `--dietas-tramo-color:${estilo.color}`);
    leyenda.append(item);
  });
  figura.append(elemento(documento, "figcaption", traducir("mapa_titulo")), lienzo, leyenda, estado, atribucion);
  return figura;
}

/**
 * Mantiene visible la zona cartográfica en el portal interno antes de que la
 * composición entregue identidad, capacidad y catálogo gobernados. No recibe
 * descriptor ni crea una geometría: es deliberadamente un estado cerrado.
 */
function crearMapaPendiente(documento, traducir) {
  const figura = elemento(documento, "figure");
  figura.className = "dietas-mapa dietas-mapa-pendiente";
  figura.dataset.dietasMapaPendiente = "";
  const lienzo = elemento(documento, "div");
  lienzo.className = "dietas-mapa-canvas";
  lienzo.dataset.dietasMapaCanvas = "";
  lienzo.dataset.modoMapa = "pendiente_calculo_autorizado";
  lienzo.setAttribute("role", "region");
  lienzo.setAttribute("aria-label", traducir("mapa_region_accesible"));
  const centro = elemento(documento, "div");
  centro.className = "dietas-mapa-centro-inicial";
  centro.dataset.dietasMapaCentro = "granada";
  centro.append(
    elemento(documento, "strong", traducir("mapa_centro_granada")),
    elemento(documento, "small", traducir("mapa_centro_inicial")),
  );
  const estado = elemento(documento, "p", traducir("mapa_pendiente_estado"));
  estado.className = "dietas-mapa-espera";
  estado.dataset.dietasMapaEstado = "";
  estado.setAttribute("role", "status");
  estado.setAttribute("aria-live", "polite");
  lienzo.append(centro, estado);
  figura.append(
    elemento(documento, "figcaption", traducir("mapa_pendiente_titulo")),
    lienzo,
  );
  return figura;
}

function crearCabeceraItinerario(documento, traducir) {
  const cabecera = elemento(documento, "div");
  cabecera.className = "cabecera-panel";
  const titulo = elemento(documento, "h2", traducir("ruta_del_dia"));
  const ayuda = elemento(documento, "button", "?");
  ayuda.type = "button";
  ayuda.className = "boton-terciario";
  ayuda.setAttribute("aria-label", `${traducir("recorridos_abrir_ayuda")} · ${traducir("ruta_del_dia")}`);
  // El shell ya abre el ayudante de trámites y contiene los pasos de Dietas.
  ayuda.dataset.accion = "ayuda";
  cabecera.append(titulo, ayuda);
  return cabecera;
}

function crearSelectorParada(documento, modelo, codigo, indice, traducir) {
  const grupo = elemento(documento, "div");
  grupo.className = "dietas-itinerario-parada";
  const esSalida = indice === 0;
  const esDestino = indice === modelo.paradas.length - 1;
  const etiquetaTexto = esSalida ? traducir("ruta_salida")
    : esDestino ? traducir("ruta_destino_final", { numero: indice })
      : traducir("ruta_etapa", { numero: indice });
  const etiqueta = elemento(documento, "label", etiquetaTexto);
  const selector = elemento(documento, "select");
  selector.dataset.itinerarioParada = String(indice);
  selector.setAttribute("aria-label", etiquetaTexto);
  const centroAsociado = esSalida ? modelo.centro_salida_asociado : null;
  if (esSalida && !codigo) {
    const opcionVacia = elemento(documento, "option", centroAsociado?.etiqueta
      ? `${centroAsociado.etiqueta} · seleccione una localidad`
      : traducir("ruta_seleccionar_localidad"));
    opcionVacia.value = "";
    opcionVacia.selected = true;
    selector.append(opcionVacia);
  }
  modelo.catalogo.puntos.forEach((punto) => {
    const esCentroAsociado = esSalida && centroAsociado?.codigo === punto.codigo;
    const opcion = elemento(documento, "option", esCentroAsociado
      ? `${centroAsociado.etiqueta} (centro asociado)` : punto.nombre);
    opcion.value = punto.codigo;
    opcion.selected = punto.codigo === codigo;
    selector.append(opcion);
  });
  etiqueta.append(selector);
  grupo.append(etiqueta);
  if (!esSalida && !esDestino) {
    const acciones = elemento(documento, "div");
    acciones.className = "acciones-vista";
    const boton = (clave, atributo, valor, bloqueado = false) => {
      const control = elemento(documento, "button", traducir(clave));
      control.type = "button";
      control.className = "boton-terciario";
      control.dataset[atributo] = valor;
      control.disabled = bloqueado;
      return control;
    };
    const subir = boton("ruta_subir_parada", "itinerarioMoverParada", String(indice), indice === 1);
    subir.dataset.itinerarioDireccion = "arriba";
    const bajar = boton("ruta_bajar_parada", "itinerarioMoverParada", String(indice), indice === modelo.paradas.length - 2);
    bajar.dataset.itinerarioDireccion = "abajo";
    acciones.append(subir, bajar, boton("ruta_quitar_parada", "itinerarioQuitarParada", String(indice)));
    grupo.append(acciones);
  }
  return grupo;
}

const CLAVES_ETIQUETA_ALTERNATIVA = Object.freeze({
  ruta_alternativa_osrm_1: "ruta_etiqueta_osrm_1",
  ruta_alternativa_osrm_2: "ruta_etiqueta_osrm_2",
  ruta_alternativa_osrm_3: "ruta_etiqueta_osrm_3",
});

function textoAlternativa(alternativa, traducir) {
  return traducir(CLAVES_ETIQUETA_ALTERNATIVA[alternativa.etiqueta] || "ruta_no_liquidable");
}

function formatoKilometros(valor) {
  return new Intl.NumberFormat("es-ES", {
    style: "unit", unit: "kilometer", unitDisplay: "short", maximumFractionDigits: 1,
  }).format(valor);
}

function formatoMinutos(valor) {
  return new Intl.NumberFormat("es-ES", { style: "unit", unit: "minute", unitDisplay: "short" }).format(valor);
}

function crearResumenAlternativas(documento, modelo, traducir, borradorAlternativa) {
  const seccion = elemento(documento, "section");
  seccion.className = "dietas-itinerario-alternativas";
  seccion.setAttribute("aria-labelledby", "dietas-itinerario-alternativas-titulo");
  const titulo = elemento(documento, "h4", traducir("ruta_alternativas_recibidas"));
  titulo.id = "dietas-itinerario-alternativas-titulo";
  const lista = elemento(documento, "ul");
  lista.className = "dietas-itinerario-alternativas-lista";
  modelo.alternativas.forEach((alternativa) => {
    const fila = elemento(documento, "li");
    fila.dataset.itinerarioAlternativa = alternativa.referencia;
    if (alternativa.seleccionada) fila.dataset.itinerarioSeleccionada = "";
    const nombre = elemento(documento, "strong", textoAlternativa(alternativa, traducir));
    const referencia = elemento(documento, "span", `${traducir("ruta_referencia")}: ${alternativa.referencia}`);
    referencia.className = "dietas-itinerario-alternativa-referencia";
    const estado = elemento(documento, "span", alternativa.seleccionada
      ? traducir("ruta_seleccionada") : traducir("ruta_no_seleccionada"));
    estado.className = "estado-chip";
    if (alternativa.recomendada) {
      const recomendada = elemento(documento, "span", traducir("ruta_recomendada"));
      recomendada.className = "estado-chip info";
      fila.append(nombre, referencia, recomendada, estado);
    } else fila.append(nombre, referencia, estado);
    fila.append(elemento(documento, "span", formatoKilometros(alternativa.kilometros)),
      elemento(documento, "span", formatoMinutos(alternativa.duracion_minutos)));
    lista.append(fila);
  });
  const ayuda = elemento(documento, "p", traducir("d4_alternativa_ayuda"));
  ayuda.className = "dietas-itinerario-aviso";
  const controles = elemento(documento, "div");
  controles.className = "dietas-ruta-paradas";
  const etiqueta = elemento(documento, "label", traducir("ruta_alternativas"));
  const selector = elemento(documento, "select");
  selector.dataset.itinerarioElegirAlternativa = "";
  modelo.alternativas.forEach((alternativa) => {
    const opcion = elemento(documento, "option",
      `${textoAlternativa(alternativa, traducir)} · ${formatoKilometros(alternativa.kilometros)} · ${formatoMinutos(alternativa.duracion_minutos)}`);
    opcion.value = alternativa.referencia;
    opcion.selected = alternativa.referencia === (borradorAlternativa?.referencia || modelo.alternativa_ref);
    selector.append(opcion);
  });
  etiqueta.append(selector);
  const motivoEtiqueta = elemento(documento, "label", traducir("ruta_motivo_alternativa"));
  const motivo = elemento(documento, "textarea");
  motivo.dataset.itinerarioMotivoAlternativa = "";
  motivo.maxLength = 500;
  motivo.rows = 2;
  motivo.value = borradorAlternativa?.motivo ?? modelo.motivo_alternativa;
  motivo.setAttribute("aria-describedby", "dietas-itinerario-motivo-ayuda");
  motivoEtiqueta.append(motivo);
  const motivoAyuda = elemento(documento, "small", traducir("d4_motivo_alternativa_ayuda"));
  motivoAyuda.id = "dietas-itinerario-motivo-ayuda";
  const boton = elemento(documento, "button", traducir("d4_previsualizar"));
  boton.type = "button";
  boton.className = "boton-secundario";
  boton.dataset.itinerarioPrevisualizar = "";
  controles.append(etiqueta, motivoEtiqueta, motivoAyuda, boton);
  seccion.append(titulo, ayuda, lista, controles);
  return seccion;
}

function crearTablaTramos(documento, modelo, traducir) {
  const seccion = elemento(documento, "section");
  seccion.className = "dietas-itinerario-tramos";
  seccion.setAttribute("aria-labelledby", "dietas-itinerario-tramos-titulo");
  const titulo = elemento(documento, "h4", traducir("ruta_tramos_seleccionados"));
  titulo.id = "dietas-itinerario-tramos-titulo";
  const desplazamiento = elemento(documento, "div");
  desplazamiento.className = "tabla-contenedor dietas-itinerario-tramos-desplazamiento";
  desplazamiento.setAttribute("tabindex", "0");
  desplazamiento.setAttribute("role", "region");
  desplazamiento.setAttribute("aria-label", traducir("ruta_tramos_seleccionados"));
  const tabla = elemento(documento, "table");
  tabla.className = "tabla-datos dietas-itinerario-tabla-tramos";
  const caption = elemento(documento, "caption", traducir("ruta_tramos_seleccionados"));
  const cabecera = elemento(documento, "thead");
  const filaCabecera = elemento(documento, "tr");
  [["ruta_origen_destino", ""], ["ruta_distancia", "numero"], ["ruta_tiempo", "numero"],
    ["ruta_ajuste_km", "numero"], ["ruta_motivo_ajuste", ""]].forEach(([clave, clase]) => {
    const celda = elemento(documento, "th", traducir(clave));
    celda.setAttribute("scope", "col");
    if (clase) celda.className = clase;
    filaCabecera.append(celda);
  });
  cabecera.append(filaCabecera);
  const cuerpo = elemento(documento, "tbody");
  modelo.tramos.forEach((tramo) => {
    const fila = elemento(documento, "tr");
    fila.dataset.itinerarioTramo = String(tramo.indice);
    const recorrido = elemento(documento, "th", `${tramo.origen_nombre} → ${tramo.destino_nombre}`);
    recorrido.setAttribute("scope", "row");
    const kilometros = elemento(documento, "td", formatoKilometros(tramo.kilometros));
    kilometros.className = "numero";
    const minutos = elemento(documento, "td", formatoMinutos(tramo.duracion_minutos));
    minutos.className = "numero";
    const ajuste = elemento(documento, "td", traducir("d4_km_sin_ajuste"));
    const motivo = elemento(documento, "td", traducir("d4_ajuste_no_disponible"));
    fila.append(recorrido, kilometros, minutos, ajuste, motivo);
    cuerpo.append(fila);
  });
  tabla.append(caption, cabecera, cuerpo);
  desplazamiento.append(tabla);
  seccion.append(titulo, desplazamiento, elemento(documento, "p", traducir("d4_ajustes_pendientes")));
  return seccion;
}

/** Monta una consulta no liquidable de rutas internas de Dietas. */
export async function montarVistaItinerarioDietas({
  raiz,
  calculador,
  visorRuta,
  anunciar = () => {},
  mensajes = MENSAJES_DIETAS_ES,
  registrarDesmontar,
  centroSalidaAsociado,
} = {}) {
  if (!raiz?.append || !calculador?.obtenerCatalogo || !calculador?.calcular
    || !visorRuta?.montar || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) {
    throw new TypeError("vista de itinerario no disponible");
  }
  const documento = raiz.ownerDocument;
  if (!documento?.createElement) throw new TypeError("documento de itinerario no disponible");

  const traducir = crearTraductorDietasD4(mensajes);
  const contenedor = elemento(documento, "div");
  contenedor.className = "modulo-dietas";
  contenedor.dataset.dietasItinerario = "";
  raiz.append(contenedor);

  let activa = true;
  let controlador = new AbortController();
  let visorActivo = null;
  let presentador = null;
  let errorVisible = "";
  let borradorAlternativa = null;
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
        centroSalidaAsociado,
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
    panel.append(crearCabeceraItinerario(documento, traducir));
    if (!presentador) {
      const alerta = elemento(documento, "p", errorVisible || traducir("ruta_error_servicio"));
      alerta.dataset.itinerarioError = "";
      alerta.setAttribute("role", "alert");
      panel.append(alerta);
      contenedor.append(panel);
      return;
    }
    const modelo = presentador.obtenerModelo();
    const vehiculo = elemento(documento, "div");
    vehiculo.className = "dietas-vehiculo-propio";
    vehiculo.dataset.itinerarioVehiculo = "";
    vehiculo.append(
      elemento(documento, "strong", traducir("d4_vehiculo_pendiente")),
      elemento(documento, "p", traducir("d4_vehiculo_ayuda")),
    );
    const paradas = elemento(documento, "div");
    paradas.className = "dietas-ruta-paradas";
    modelo.paradas.forEach((codigo, indice) => paradas.append(
      crearSelectorParada(documento, modelo, codigo, indice, traducir),
    ));
    const anadir = elemento(documento, "button", traducir("ruta_anadir_parada"));
    anadir.type = "button";
    anadir.className = "boton-secundario";
    anadir.dataset.itinerarioAnadirParada = "";
    const calcular = elemento(documento, "button", traducir("ruta_calcular_osrm"));
    calcular.type = "button";
    calcular.className = "boton-primario";
    calcular.dataset.itinerarioCalcular = "";
    calcular.disabled = controlador !== null || modelo.paradas.some((parada) => !parada);
    panel.append(vehiculo, paradas, anadir, calcular);
    if (errorVisible) {
      const alerta = elemento(documento, "p", errorVisible);
      alerta.dataset.itinerarioError = "";
      alerta.setAttribute("role", "alert");
      panel.append(alerta);
    }
    if (modelo.calculado) {
      const resultado = elemento(documento, "section");
      resultado.className = "dietas-ruta-resultado";
      const avisoOrientativo = elemento(documento, "p", traducir("ruta_aviso_orientativa"));
      avisoOrientativo.className = "dietas-itinerario-aviso";
      avisoOrientativo.dataset.itinerarioAvisoNoLiquidable = "";
      const avisoEfectos = elemento(documento, "p", traducir("ruta_aviso_sin_efectos"));
      avisoEfectos.className = "dietas-itinerario-aviso";
      avisoEfectos.dataset.itinerarioAvisoSinEfectos = "";
      resultado.append(
        elemento(documento, "h3", traducir("ruta_resultado")),
        avisoOrientativo,
        avisoEfectos,
        elemento(documento, "p", traducir("ruta_motor_version", {
          motor: modelo.motor, version: modelo.version_grafo,
        })),
        elemento(documento, "p", `${traducir("ruta_km_base")}: ${formatoKilometros(modelo.kilometros_base)}`),
        elemento(documento, "p", `${traducir("ruta_duracion")}: ${formatoMinutos(modelo.duracion_minutos)}`),
        elemento(documento, "p", traducir("d4_sin_importe")),
      );
      resultado.append(crearResumenAlternativas(documento, modelo, traducir, borradorAlternativa));
      resultado.append(crearTablaTramos(documento, modelo, traducir));
      const mapa = crearMapa(documento, modelo, traducir);
      if (mapa) resultado.append(mapa);
      panel.append(resultado);
      contenedor.append(panel);
      if (mapa && activaAhora()) visorActivo = visorRuta.montar({ raiz: mapa, descriptor: modelo.mapa_ruta });
      return;
    }
    // La zona cartográfica permanece visible antes del primer cálculo y ante
    // un fallo del motor. Solo muestra el centro nominal de Granada; ninguna
    // carretera o distancia se crea sin una respuesta OSRM válida.
    panel.append(crearMapaPendiente(documento, traducir));
    contenedor.append(panel);
  }

  function aplicar(operacion, claveError = "ruta_error_operacion") {
    let correcta = false;
    try {
      errorVisible = "";
      operacion();
      correcta = true;
    } catch {
      errorVisible = traducir(claveError);
      anunciar(errorVisible, "error");
    }
    pintar();
    return correcta;
  }
  function cambio(evento) {
    const selector = evento.target.closest?.("[data-itinerario-parada]");
    if (!selector || !presentador) return;
    aplicar(() => presentador.establecerParada(Number(selector.dataset.itinerarioParada), selector.value));
  }
  function moverParada(indice, direccion) {
    const modelo = presentador.obtenerModelo();
    const destino = direccion === "arriba" ? indice - 1 : indice + 1;
    if (indice <= 0 || destino <= 0 || indice >= modelo.paradas.length - 1 || destino >= modelo.paradas.length - 1) {
      throw new Error("posición de etapa no válida");
    }
    const actual = modelo.paradas[indice];
    presentador.establecerParada(indice, modelo.paradas[destino]);
    presentador.establecerParada(destino, actual);
  }
  async function clic(evento) {
    const previsualizar = evento.target.closest?.("[data-itinerario-previsualizar]");
    if (previsualizar && presentador && !controlador) {
      const selector = contenedor.querySelector("[data-itinerario-elegir-alternativa]");
      const motivo = contenedor.querySelector("[data-itinerario-motivo-alternativa]");
      borradorAlternativa = { referencia: selector.value, motivo: motivo.value };
      const correcta = aplicar(
        () => presentador.seleccionarAlternativa(selector.value, motivo.value),
        "d4_motivo_alternativa_error",
      );
      if (correcta) borradorAlternativa = null;
      (correcta
        ? contenedor.querySelector("[data-itinerario-elegir-alternativa]")
        : contenedor.querySelector("[data-itinerario-motivo-alternativa]"))?.focus?.();
      return;
    }
    const anadir = evento.target.closest?.("[data-itinerario-anadir-parada]");
    if (anadir && presentador && !controlador) {
      aplicar(() => presentador.agregarParada());
      return;
    }
    const quitar = evento.target.closest?.("[data-itinerario-quitar-parada]");
    if (quitar && presentador && !controlador) {
      aplicar(() => presentador.eliminarParada(Number(quitar.dataset.itinerarioQuitarParada)));
      return;
    }
    const mover = evento.target.closest?.("[data-itinerario-mover-parada]");
    if (mover && presentador && !controlador) {
      aplicar(() => moverParada(Number(mover.dataset.itinerarioMoverParada), mover.dataset.itinerarioDireccion));
      return;
    }
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

/**
 * Placeholder corporativo del mismo visor. Permite orientar el trámite sin
 * fingir que el catálogo, la identidad o la ruta ya han sido autorizados.
 */
export function montarVistaItinerarioPendienteDietas({
  raiz,
  mensajes = MENSAJES_DIETAS_ES,
  registrarDesmontar,
} = {}) {
  if (!raiz?.append || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) {
    throw new TypeError("zona cartográfica de Dietas no disponible");
  }
  const documento = raiz.ownerDocument;
  if (!documento?.createElement) throw new TypeError("documento de itinerario no disponible");
  const traducir = crearTraductorDietasD4(mensajes);
  const contenedor = elemento(documento, "div");
  contenedor.className = "modulo-dietas";
  contenedor.dataset.dietasItinerario = "";
  const panel = elemento(documento, "section");
  panel.className = "panel dietas-itinerario";
  panel.dataset.dietasItinerarioPendiente = "";
  panel.append(
    crearCabeceraItinerario(documento, traducir),
    crearMapaPendiente(documento, traducir),
  );
  contenedor.append(panel);
  raiz.append(contenedor);
  let activa = true;
  const desmontar = () => {
    if (!activa) return;
    activa = false;
    retirar(raiz, contenedor);
  };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar });
}
