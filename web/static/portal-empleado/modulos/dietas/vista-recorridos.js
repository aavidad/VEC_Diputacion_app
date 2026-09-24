import { crearTraductorDietas, MENSAJES_DIETAS_ES } from "./i18n.js";
import { montarVistaBorradoresPropios } from "./vista-borradores-propios.js";
import { montarVistaBandejaCircuitoDietas } from "./vista-bandeja-circuito.js";
import { montarVistaRectificacionAdminDietas } from "./vista-rectificacion-admin.js";

const ETAPAS = Object.freeze([
  ["solicitante", "recorridos_solicitante"],
  ["revision", "circuito_etapa_revision"],
  ["autorizacion", "circuito_etapa_autorizacion"],
  ["liquidacion", "circuito_etapa_liquidacion"],
  ["fiscalizacion", "circuito_etapa_fiscalizacion"],
  ["rectificacion_admin", "ra_titulo"],
]);
const nodo = (documento, etiqueta, texto = "") => {
  const resultado = documento.createElement(etiqueta);
  if (texto) resultado.textContent = texto;
  return resultado;
};
const montada = (contenedor, raiz) => contenedor.querySelector?.("[data-dietas-recorridos]") === raiz;

/** Reúne el recorrido propio y las bandejas autorizadas del circuito de Dietas. */
export function montarVistaRecorridosDietas(contenedor, {
  clienteBorradores,
  clienteAsignacion,
  clienteRectificacion,
  clienteCircuito,
  clienteRectificacionAdmin,
  clienteCatalogoCompetente,
  calculadorRuta,
  visorRuta,
  relacionesAutorizadas = [],
  fechaReferenciaPersonal,
  estadoRelaciones = "disponible",
  traducir = crearTraductorDietas(MENSAJES_DIETAS_ES),
  anunciar = () => {},
  registrarDesmontar,
} = {}) {
  if (!contenedor?.append || !contenedor?.querySelector || !contenedor.ownerDocument ||
      typeof traducir !== "function" || typeof anunciar !== "function" ||
      !Array.isArray(relacionesAutorizadas) ||
      (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function"))
    throw new TypeError("recorrido de Dietas no disponible");
  const documento = contenedor.ownerDocument;
  const raiz = nodo(documento, "section");
  raiz.className = "modulo-dietas dietas-recorridos";
  raiz.dataset.dietasRecorridos = "";
  contenedor.replaceChildren(raiz);
  let activa = true;
  let etapa = "solicitante";
  let formularioAbierto = false;
  let vistaCircuito = null;
  const cabecera = nodo(documento, "header");
  cabecera.className = "dietas-recorridos-cabecera panel";
  cabecera.append(nodo(documento, "h2", traducir("titulo")));
  const pasos = nodo(documento, "nav");
  pasos.className = "dietas-recorridos-pasos";
  pasos.setAttribute("aria-label", traducir("recorridos_roles"));
  ETAPAS.forEach(([codigo, clave], indice) => {
    const boton = nodo(documento, "button", `${indice + 1}. ${traducir(clave)}`);
    boton.type = "button";
    boton.dataset.dietasCambiarEtapa = codigo;
    pasos.append(boton);
  });
  const cuerpo = nodo(documento, "div");
  cuerpo.className = "dietas-recorridos-cuerpo";
  const panelPropio = nodo(documento, "section");
  panelPropio.className = "dietas-recorridos-principal";
  panelPropio.dataset.dietasPanelEtapa = "solicitante";
  const cabeceraPropia = nodo(documento, "div");
  cabeceraPropia.className = "dietas-recorridos-cabecera panel";
  const tituloPropio = nodo(documento, "h3", traducir("revision_mis_comisiones"));
  const abrir = nodo(documento, "button", traducir("nueva_comision"));
  abrir.type = "button";
  abrir.className = "boton-primario";
  abrir.dataset.dietasAbrirNuevaComision = "";
  abrir.setAttribute("aria-expanded", "false");
  abrir.disabled = !clienteBorradores || estadoRelaciones === "no_disponible";
  if (abrir.disabled) abrir.title = traducir(estadoRelaciones === "no_disponible" ?
    "comision_relaciones_no_disponibles" : "borradores_propios_pendiente_conexion");
  cabeceraPropia.append(tituloPropio, abrir);
  const areaBorradores = nodo(documento, "div");
  areaBorradores.dataset.dietasAreaBorradores = "";
  panelPropio.append(cabeceraPropia, areaBorradores);
  const panelCircuito = nodo(documento, "section");
  panelCircuito.className = "dietas-recorridos-principal";
  panelCircuito.dataset.dietasPanelEtapa = "circuito";
  panelCircuito.hidden = true;
  const areaCircuito = nodo(documento, "div");
  areaCircuito.dataset.dietasAreaCircuito = "";
  panelCircuito.append(areaCircuito);
  cuerpo.append(panelPropio, panelCircuito);
  raiz.append(cabecera, pasos, cuerpo);

  const vistaBorradores = montarVistaBorradoresPropios(areaBorradores, {
    cliente: clienteBorradores, clienteAsignacion, clienteRectificacion, calculadorRuta, visorRuta,
    relacionesAutorizadas, fechaReferenciaPersonal, estadoRelaciones,
    traducir, anunciar, formularioInicialmenteVisible: false,
  });
  const formulario = areaBorradores.querySelector?.("[data-dietas-borrador-form]");
  if (formulario) formulario.id = "dietas-recorridos-formulario";
  abrir.setAttribute("aria-controls", "dietas-recorridos-formulario");

  function pintar() {
    if (!activa || !montada(contenedor, raiz)) return;
    panelPropio.hidden = etapa !== "solicitante";
    panelCircuito.hidden = etapa === "solicitante";
    pasos.querySelectorAll("button").forEach((boton) => {
      if (boton.dataset.dietasCambiarEtapa === etapa) boton.setAttribute("aria-current", "step");
      else boton.removeAttribute?.("aria-current");
    });
  }
  function seleccionarEtapa(siguiente) {
    if (etapa === siguiente) return;
    etapa = siguiente;
    vistaCircuito?.desmontar();
    vistaCircuito = null;
    areaCircuito.replaceChildren();
    if (siguiente === "rectificacion_admin" && clienteRectificacionAdmin) {
      vistaCircuito = montarVistaRectificacionAdminDietas(areaCircuito, {
        cliente: clienteRectificacionAdmin, clienteCatalogoCompetente, traducir, anunciar,
      });
    } else if (siguiente !== "solicitante" && siguiente !== "rectificacion_admin" && clienteCircuito) {
      vistaCircuito = montarVistaBandejaCircuitoDietas(areaCircuito, {
        cliente: clienteCircuito, traducir, anunciar, etapaInicial: siguiente,
      });
    } else if (siguiente !== "solicitante") {
      const aviso = nodo(documento, "p", traducir("recorridos_pendiente_conexion"));
      aviso.setAttribute("role", "status");
      areaCircuito.append(aviso);
    }
    pintar();
  }
  function clic(evento) {
    const botonNueva = evento.target?.closest?.("[data-dietas-abrir-nueva-comision]");
    if (botonNueva && activa && !botonNueva.disabled) {
      formularioAbierto = !formularioAbierto;
      if (formularioAbierto) vistaBorradores.abrirFormulario();
      else vistaBorradores.cerrarFormulario();
      botonNueva.setAttribute("aria-expanded", String(formularioAbierto));
      return;
    }
    const botonEtapa = evento.target?.closest?.("[data-dietas-cambiar-etapa]");
    if (!botonEtapa || !activa || !ETAPAS.some(([codigo]) => codigo === botonEtapa.dataset.dietasCambiarEtapa)) return;
    seleccionarEtapa(botonEtapa.dataset.dietasCambiarEtapa);
    botonEtapa.focus?.();
  }
  function desmontar() {
    if (!activa) return;
    activa = false;
    raiz.removeEventListener("click", clic);
    vistaCircuito?.desmontar();
    vistaBorradores.desmontar();
    if (montada(contenedor, raiz)) raiz.remove?.();
  }
  raiz.addEventListener("click", clic);
  registrarDesmontar?.(desmontar);
  pintar();
  return Object.freeze({ desmontar });
}
