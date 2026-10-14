import { crearTraductorDietas, MENSAJES_DIETAS } from "./i18n.js?v=20260929-i18n-dietas-v1";
import { montarVistaBorradoresPropios } from "./vista-borradores-propios.js?v=20261014-dietas-tarifa-publicada-web-v6";
import { montarVistaBandejaCircuitoDietas } from "./vista-bandeja-circuito.js?v=20261014-dietas-tarifa-publicada-web-v6";

import { montarVistaRectificacionAdminDietas } from "./vista-rectificacion-admin.js?v=20260929-i18n-dietas-v1";

const ETAPAS_CIRCUITO = Object.freeze(["revision", "autorizacion", "liquidacion", "fiscalizacion"]);
const nodo = (documento, etiqueta, texto = "") => {
  const resultado = documento.createElement(etiqueta);
  if (texto) resultado.textContent = texto;
  return resultado;
};
const montada = (contenedor, raiz) => contenedor.querySelector?.("[data-dietas-recorridos]") === raiz;

/**
 * Reúne el recorrido propio y las bandejas del circuito de Dietas. Las
 * bandejas salen de las competencias que acredita el servidor: una por etapa
 * acreditada más el «Control de documentos». Sin fuente gobernada no se
 * ofrece ninguna pestaña del circuito: la persona ve solo sus documentos,
 * igual que sin cliente de circuito.
 */
export function montarVistaRecorridosDietas(contenedor, {
  clienteBorradores,
  clienteAsignacion,
  clienteRectificacion,
  clienteCircuito,
  clienteLiquidacion,
  clienteRectificacionAdmin,
  clienteCatalogoCompetente,
  calculadorRuta,
  visorRuta,
  relacionesAutorizadas = [],
  fechaReferenciaPersonal,
  estadoRelaciones = "disponible",
  motivoRelaciones,
  traducir = crearTraductorDietas(MENSAJES_DIETAS),
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
  let etapasDisponibles = [["solicitante", "recorridos_solicitante"],
    ...(clienteRectificacionAdmin ? [["rectificacion_admin", "ra_titulo"]] : [])];
  let etapasCircuito = [];
  function pintarPasos() {
    pasos.replaceChildren();
    pasos.hidden = etapasDisponibles.length < 2;
    if (pasos.hidden) return;
    etapasDisponibles.forEach(([codigo, clave], indice) => {
      const boton = nodo(documento, "button", `${indice + 1}. ${traducir(clave)}`);
      boton.type = "button";
      boton.dataset.dietasCambiarEtapa = codigo;
      pasos.append(boton);
    });
  }
  pintarPasos();
  const estadoCompetencias = nodo(documento, "div");
  estadoCompetencias.className = "panel";
  estadoCompetencias.dataset.dietasEstadoCompetencias = "";
  estadoCompetencias.setAttribute("role", "status");
  estadoCompetencias.setAttribute("aria-live", "polite");
  estadoCompetencias.setAttribute("tabindex", "-1");
  estadoCompetencias.hidden = true;
  const mensajeCompetencias = nodo(documento, "p");
  const reintentarCompetencias = nodo(documento, "button", traducir("circuito_competencias_reintentar"));
  reintentarCompetencias.type = "button";
  reintentarCompetencias.className = "boton-secundario";
  reintentarCompetencias.dataset.dietasReintentarCompetencias = "";
  estadoCompetencias.append(mensajeCompetencias, reintentarCompetencias);
  const cuerpo = nodo(documento, "div");
  cuerpo.className = "dietas-recorridos-cuerpo";
  const panelPropio = nodo(documento, "section");
  panelPropio.className = "dietas-recorridos-principal";
  panelPropio.dataset.dietasPanelEtapa = "solicitante";
  const cabeceraPropia = nodo(documento, "div");
  cabeceraPropia.className = "dietas-recorridos-cabecera panel";
  const tituloPropio = nodo(documento, "h3", traducir("revision_mis_comisiones"));
  tituloPropio.setAttribute("tabindex", "-1");
  const abrir = nodo(documento, "button", traducir("nueva_comision"));
  abrir.type = "button";
  abrir.className = "boton-primario";
  abrir.dataset.dietasAbrirNuevaComision = "";
  abrir.setAttribute("aria-expanded", "false");
  abrir.disabled = !clienteBorradores || estadoRelaciones === "no_disponible";
  if (abrir.disabled) abrir.title = traducir(estadoRelaciones !== "no_disponible"
    ? "borradores_propios_pendiente_conexion"
    : motivoRelaciones ? `comision_${motivoRelaciones}` : "comision_relaciones_no_disponibles");
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
  raiz.append(cabecera, pasos, estadoCompetencias, cuerpo);

  const vistaBorradores = montarVistaBorradoresPropios(areaBorradores, {
    cliente: clienteBorradores, clienteAsignacion, clienteRectificacion, clienteLiquidacion, calculadorRuta, visorRuta,
    relacionesAutorizadas, fechaReferenciaPersonal, estadoRelaciones, motivoRelaciones,
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
    if (etapa === siguiente || vistaCircuito?.bloqueaSalida?.()) return;
    etapa = siguiente;
    vistaCircuito?.desmontar();
    vistaCircuito = null;
    areaCircuito.replaceChildren();
    if (siguiente === "rectificacion_admin" && clienteRectificacionAdmin) {
      vistaCircuito = montarVistaRectificacionAdminDietas(areaCircuito, {
        cliente: clienteRectificacionAdmin, clienteCatalogoCompetente, traducir, anunciar,
      });
    } else if (siguiente === "control" && clienteCircuito && etapasCircuito.length) {
      vistaCircuito = montarVistaBandejaCircuitoDietas(areaCircuito, {
        cliente: clienteCircuito, clienteLiquidacion, traducir, anunciar, etapaInicial: etapasCircuito[0], etapas: etapasCircuito, control: true,
        alCambiarBloqueo: (bloqueada) => { pasos.querySelectorAll("button").forEach((boton) => { boton.disabled = bloqueada; }); },
      });
    } else if (ETAPAS_CIRCUITO.includes(siguiente) && clienteCircuito) {
      vistaCircuito = montarVistaBandejaCircuitoDietas(areaCircuito, {
        cliente: clienteCircuito, clienteLiquidacion, traducir, anunciar, etapaInicial: siguiente, etapas: [siguiente],
        alCambiarBloqueo: (bloqueada) => { pasos.querySelectorAll("button").forEach((boton) => { boton.disabled = bloqueada; }); },
      });
    }
    pintar();
  }
  function clic(evento) {
    if (evento.target?.closest?.("[data-dietas-reintentar-competencias]")) {
      if (activa && !reintentarCompetencias.disabled) void cargarCompetencias();
      return;
    }
    const botonNueva = evento.target?.closest?.("[data-dietas-abrir-nueva-comision]");
    if (botonNueva && activa && !botonNueva.disabled) {
      formularioAbierto = !formularioAbierto;
      if (formularioAbierto) vistaBorradores.abrirFormulario();
      else vistaBorradores.cerrarFormulario();
      botonNueva.setAttribute("aria-expanded", String(formularioAbierto));
      return;
    }
    const botonEtapa = evento.target?.closest?.("[data-dietas-cambiar-etapa]");
    if (!botonEtapa || !activa || !etapasDisponibles.some(([codigo]) => codigo === botonEtapa.dataset.dietasCambiarEtapa)) return;
    seleccionarEtapa(botonEtapa.dataset.dietasCambiarEtapa);
    botonEtapa.focus?.();
  }
  function desmontar() {
    if (!activa) return;
    activa = false;
    cancelacionCompetencias.abort();
    raiz.removeEventListener("click", clic);
    vistaCircuito?.desmontar();
    vistaBorradores.desmontar();
    if (montada(contenedor, raiz)) raiz.remove?.();
  }
  // Las pestañas del circuito dependen de lo que acredite la fuente gobernada.
  const cancelacionCompetencias = new AbortController();
  let cargaCompetencias = false;
  let competenciasResueltas = false;
  async function cargarCompetencias() {
    if (!activa || competenciasResueltas || cargaCompetencias || typeof clienteCircuito?.competencias !== "function") return;
    const focoReintento = documento.activeElement === reintentarCompetencias;
    cargaCompetencias = true;
    reintentarCompetencias.disabled = true;
    reintentarCompetencias.hidden = false;
    estadoCompetencias.hidden = false;
    mensajeCompetencias.textContent = traducir("circuito_competencias_cargando");
    return (async () => {
      try {
        const competencias = await clienteCircuito.competencias({ signal: cancelacionCompetencias.signal });
        if (!activa || !montada(contenedor, raiz)) return;
        competenciasResueltas = true;
        estadoCompetencias.hidden = true;
        // Sin fuente o sin etapas acreditadas no hay bandejas ni consulta de listados.
        if (competencias.fuente === "sin_fuente" || competencias.etapas.length === 0) {
          if (focoReintento && documento.activeElement === reintentarCompetencias) tituloPropio.focus?.();
          return;
        }
        etapasCircuito = ETAPAS_CIRCUITO.filter((codigo) => competencias.etapas.includes(codigo));
        const circuito = [...etapasCircuito.map((codigo) => [codigo, `circuito_etapa_${codigo}`]), ["control", "circuito_control"]];
        etapasDisponibles = [etapasDisponibles[0], ...circuito, ...etapasDisponibles.slice(1)];
        pintarPasos();
        pintar();
        if (focoReintento && documento.activeElement === reintentarCompetencias) {
          pasos.querySelector(`[data-dietas-cambiar-etapa="${etapasCircuito[0]}"]`)?.focus?.();
        }
      } catch (error) {
        if (!activa || !montada(contenedor, raiz) || error?.codigo === "operacion_abortada") return;
        const denegada = error?.codigo === "acceso_denegado" || error?.estado === 403;
        const autenticacion = error?.codigo === "autenticacion_requerida" || error?.estado === 401;
        if (error?.codigo === "competencia_sin_fuente") {
          competenciasResueltas = true;
          estadoCompetencias.hidden = true;
          if (focoReintento && documento.activeElement === reintentarCompetencias) tituloPropio.focus?.();
          return;
        }
        mensajeCompetencias.textContent = traducir(denegada ? "circuito_competencias_denegadas"
          : autenticacion ? "circuito_competencias_sesion" : "circuito_competencias_error");
        reintentarCompetencias.hidden = denegada || autenticacion;
        estadoCompetencias.hidden = false;
        if (focoReintento && reintentarCompetencias.hidden && documento.activeElement === reintentarCompetencias) {
          estadoCompetencias.focus?.();
        }
      } finally {
        cargaCompetencias = false;
        reintentarCompetencias.disabled = false;
      }
    })();
  }
  raiz.addEventListener("click", clic);
  registrarDesmontar?.(desmontar);
  pintar();
  cargarCompetencias();
  return Object.freeze({ desmontar });
}
