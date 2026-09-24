import { crearTraductorDietas, MENSAJES_DIETAS_ES } from "./i18n.js";
import { crearTraductorRevisionDietas } from "./i18n-revision.js";
import { montarVistaBorradoresPropios } from "./vista-borradores-propios.js?v=20260924-f2-consulta-v1";
import { montarMapaInicialGranadaDietas } from "./mapa-ruta.js";

const ETAPAS = Object.freeze([
  ["solicitante", "recorridos_solicitante"],
  ["jefatura", "recorridos_jefatura"],
  ["gestion", "recorridos_gestion"],
]);
const nodo = (documento, etiqueta, texto = "") => {
  const resultado = documento.createElement(etiqueta);
  if (texto) resultado.textContent = texto;
  return resultado;
};
const sigueMontada = (contenedor, raiz) => contenedor.querySelector?.("[data-dietas-recorridos]") === raiz;
function retirar(contenedor, raiz) {
  if (!sigueMontada(contenedor, raiz)) return;
  if (typeof raiz.remove === "function") raiz.remove();
  else contenedor.removeChild?.(raiz);
}
function ayuda(documento, t, clave) {
  const detalles = nodo(documento, "details");
  detalles.className = "dietas-recorridos-ayuda";
  detalles.append(nodo(documento, "summary", t("recorridos_abrir_ayuda")), nodo(documento, "p", t(clave)));
  return detalles;
}
function panelPendiente(documento, t, etapa, titulo, descripcion, acciones) {
  const panel = nodo(documento, "section");
  panel.className = "panel dietas-recorridos-principal dietas-recorridos-pendiente";
  panel.dataset.dietasPanelEtapa = etapa;
  const cabecera = nodo(documento, "div");
  cabecera.className = "cabecera-panel";
  cabecera.append(nodo(documento, "h2", t(titulo)), ayuda(documento, t, "revision_ayuda_circuito"));
  const cuerpo = nodo(documento, "div");
  cuerpo.className = "cuerpo-panel";
  const aviso = nodo(documento, "p", t(descripcion));
  aviso.className = "dietas-recorridos-aviso";
  aviso.setAttribute("role", "status");
  cuerpo.append(aviso);
  const barra = nodo(documento, "div");
  barra.className = "acciones-vista";
  acciones.forEach((clave) => {
    const boton = nodo(documento, "button", t(clave));
    boton.type = "button";
    boton.className = "boton-secundario";
    boton.disabled = true;
    boton.title = t("revision_accion_sin_servicio");
    barra.append(boton);
  });
  cuerpo.append(barra);
  panel.append(cabecera, cuerpo);
  return panel;
}
function panelSolicitante(documento, t, areaBorradores, areaItinerario, puedeCrear) {
  const panel = nodo(documento, "section");
  panel.className = "dietas-recorridos-principal";
  panel.dataset.dietasPanelEtapa = "solicitante";
  const cabecera = nodo(documento, "div");
  cabecera.className = "dietas-recorridos-cabecera panel";
  const titulos = nodo(documento, "div");
  titulos.append(nodo(documento, "h2", t("revision_mis_comisiones")), nodo(documento, "p", t(puedeCrear ? "revision_subtitulo" : "revision_subtitulo_sin_cliente")));
  const acciones = nodo(documento, "div");
  acciones.className = "dietas-recorridos-cabecera-acciones";
  const abrir = nodo(documento, "button", puedeCrear ? t("nueva_comision", { demo: "" }) : t("revision_explorar_itinerario"));
  abrir.type = "button";
  abrir.className = "boton-primario";
  abrir.dataset.dietasAbrirNuevaComision = "";
  abrir.disabled = !puedeCrear && !areaItinerario;
  if (!puedeCrear) abrir.title = areaItinerario ? t("revision_itinerario_sin_registro") : t("borradores_propios_pendiente_conexion");
  abrir.setAttribute("aria-expanded", "false");
  abrir.setAttribute("aria-controls", puedeCrear ? "dietas-recorridos-nueva dietas-recorridos-formulario" : "dietas-recorridos-nueva");
  acciones.append(ayuda(documento, t, puedeCrear ? "revision_ayuda_propia" : "revision_ayuda_sin_cliente"), abrir);
  cabecera.append(titulos, acciones);
  const nueva = nodo(documento, "section");
  nueva.className = "dietas-nueva-comision";
  nueva.id = "dietas-recorridos-nueva";
  nueva.dataset.dietasNuevaComision = "";
  nueva.setAttribute("tabindex", "-1");
  nueva.setAttribute("aria-label", puedeCrear ? t("nueva_comision", { demo: "" }) : t("revision_explorar_itinerario"));
  nueva.hidden = true;
  const cabeceraNueva = nodo(documento, "div");
  cabeceraNueva.className = "cabecera-panel";
  const cerrar = nodo(documento, "button", t(puedeCrear ? "cerrar_nueva_comision" : "revision_cerrar_itinerario"));
  cerrar.type = "button";
  cerrar.className = "boton-secundario";
  cerrar.dataset.dietasCerrarNuevaComision = "";
  cabeceraNueva.append(nodo(documento, "h3", puedeCrear ? t("nueva_comision", { demo: "" }) : t("revision_explorar_itinerario")), cerrar);
  nueva.append(cabeceraNueva);
  if (!puedeCrear) {
    const aviso = nodo(documento, "p", t("revision_itinerario_sin_registro"));
    aviso.className = "dietas-recorridos-aviso dietas-recorridos-aviso-registro";
    aviso.setAttribute("role", "status");
    nueva.append(aviso);
  }
  if (areaItinerario) nueva.append(areaItinerario);
  const revision = nodo(documento, "section");
  revision.className = "dietas-recorridos-revision";
  revision.dataset.dietasRevisionComision = "";
  const cabeceraRevision = nodo(documento, "div");
  cabeceraRevision.className = "dietas-recorridos-revision-cabecera";
  cabeceraRevision.append(nodo(documento, "h3", t("revision_titulo")), nodo(documento, "p", t(puedeCrear ? "revision_instruccion" : "borradores_propios_pendiente_conexion")));
  revision.append(cabeceraRevision, areaBorradores);
  panel.append(cabecera, nueva, revision);
  return panel;
}

/** Muestra comisiones propias con el GET/POST autorizado; las demás etapas quedan cerradas. */
export function montarVistaRecorridosDietas(contenedor, {
  clienteBorradores,
  traducir = crearTraductorDietas(MENSAJES_DIETAS_ES),
  montarItinerario,
  anunciar = () => {},
  registrarDesmontar,
} = {}) {
  if (!contenedor?.append || !contenedor?.querySelector || !contenedor.ownerDocument ||
      typeof traducir !== "function" || typeof anunciar !== "function" ||
      (montarItinerario !== undefined && typeof montarItinerario !== "function") ||
      (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function"))
    throw new TypeError("recorrido de Dietas no disponible");
  const documento = contenedor.ownerDocument;
  const t = crearTraductorRevisionDietas(traducir);
  const raiz = nodo(documento, "section");
  raiz.className = "modulo-dietas dietas-recorridos";
  raiz.dataset.dietasRecorridos = "";
  contenedor.replaceChildren(raiz);
  let activa = true;
  let etapa = "solicitante";
  let vistaBorradores;
  let desmontarItinerario = () => {};
  let desmontarMapaInicial = () => {};
  let itinerarioIniciado = false;
  let generacionItinerario = 0;
  const areaBorradores = nodo(documento, "div");
  areaBorradores.dataset.dietasAreaBorradores = "";
  try {
    vistaBorradores = montarVistaBorradoresPropios(areaBorradores, {
      cliente: clienteBorradores, traducir: t, anunciar, formularioInicialmenteVisible: false,
    });
    const formulario = areaBorradores.querySelector?.("[data-dietas-borrador-form]");
    if (formulario) formulario.id = "dietas-recorridos-formulario";
  } catch {
    const aviso = nodo(documento, "p", t("borradores_propios_pendiente_conexion"));
    aviso.setAttribute("role", "status");
    areaBorradores.append(aviso);
  }
  const areaItinerario = montarItinerario ? nodo(documento, "div") : null;
  if (areaItinerario) areaItinerario.dataset.dietasAreaItinerario = "";
  function detenerItinerario() {
    generacionItinerario += 1;
    itinerarioIniciado = false;
    desmontarMapaInicial();
    desmontarMapaInicial = () => {};
    desmontarItinerario();
    desmontarItinerario = () => {};
    areaItinerario?.replaceChildren?.();
  }
  function iniciarItinerario() {
    if (!areaItinerario || itinerarioIniciado || !activa) return;
    itinerarioIniciado = true;
    const generacion = ++generacionItinerario;
    try {
      Promise.resolve(montarItinerario(areaItinerario)).then(
        (vista) => {
          if (typeof vista?.desmontar !== "function") return;
          if (!activa || generacion !== generacionItinerario) { vista.desmontar(); return; }
          desmontarItinerario = vista.desmontar;
          const mapaPendiente = areaItinerario.querySelector?.("[data-dietas-mapa-pendiente]");
          if (mapaPendiente) desmontarMapaInicial = montarMapaInicialGranadaDietas({ raiz: mapaPendiente, permitirTeselas: true }).desmontar;
        },
        () => {
          if (activa && generacion === generacionItinerario && sigueMontada(contenedor, raiz))
            areaItinerario.append(nodo(documento, "p", t("recorridos_sin_datos")));
        },
      );
    } catch {
      areaItinerario.append(nodo(documento, "p", t("recorridos_sin_datos")));
    }
  }
  const pasos = nodo(documento, "nav");
  pasos.className = "dietas-recorridos-pasos";
  pasos.setAttribute("aria-label", t("recorridos_roles"));
  const botonesEtapa = new Map();
  ETAPAS.forEach(([valor, etiqueta], indice) => {
    const boton = nodo(documento, "button", `${indice + 1}. ${t(etiqueta)}`);
    boton.type = "button";
    boton.dataset.dietasCambiarEtapa = valor;
    botonesEtapa.set(valor, boton);
    pasos.append(boton);
  });
  const paneles = new Map([
    ["solicitante", panelSolicitante(documento, t, areaBorradores, areaItinerario, Boolean(vistaBorradores && clienteBorradores))],
    ["jefatura", panelPendiente(documento, t, "jefatura", "recorridos_jefatura", "revision_jefatura_sin_servicio", ["recorridos_validar", "recorridos_devolver", "recorridos_rechazar"])],
    ["gestion", panelPendiente(documento, t, "gestion", "recorridos_gestion", "revision_gestion_sin_servicio", ["recorridos_revisar_conceptos", "recorridos_liquidar", "recorridos_seguir_pago"])],
  ]);
  const cuerpo = nodo(documento, "div");
  cuerpo.className = "dietas-recorridos-cuerpo";
  cuerpo.append(...paneles.values());
  raiz.append(pasos, cuerpo);
  function pintar() {
    if (!activa || !sigueMontada(contenedor, raiz)) return;
    paneles.forEach((panel, valor) => { panel.hidden = valor !== etapa; });
    botonesEtapa.forEach((boton, valor) => {
      if (valor === etapa) boton.setAttribute("aria-current", "step");
      else boton.removeAttribute?.("aria-current");
    });
    if (etapa === "solicitante" && areaItinerario) documento.defaultView?.dispatchEvent?.(new Event("resize"));
  }
  function clic(evento) {
    const abrir = evento.target?.closest?.("[data-dietas-abrir-nueva-comision]");
    const cerrar = evento.target?.closest?.("[data-dietas-cerrar-nueva-comision]");
    if ((abrir || cerrar) && activa && !abrir?.disabled) {
      const abierta = Boolean(abrir);
      const nueva = raiz.querySelector?.("[data-dietas-nueva-comision]");
      if (nueva) nueva.hidden = !abierta;
      const botonAbrir = raiz.querySelector?.("[data-dietas-abrir-nueva-comision]");
      if (botonAbrir) { botonAbrir.hidden = abierta; botonAbrir.setAttribute("aria-expanded", String(abierta)); }
      if (abierta) {
        iniciarItinerario();
        if (!clienteBorradores || !vistaBorradores?.abrirFormulario?.()) nueva?.focus?.();
      }
      else { vistaBorradores?.cerrarFormulario?.(); detenerItinerario(); botonAbrir?.focus?.(); }
      return;
    }
    const boton = evento.target?.closest?.("[data-dietas-cambiar-etapa]");
    if (!boton || !activa || !paneles.has(boton.dataset.dietasCambiarEtapa)) return;
    etapa = boton.dataset.dietasCambiarEtapa;
    pintar();
    boton.focus?.();
  }
  function desmontar() {
    if (!activa) return;
    activa = false;
    raiz.removeEventListener("click", clic);
    vistaBorradores?.desmontar?.();
    detenerItinerario();
    retirar(contenedor, raiz);
  }
  raiz.addEventListener("click", clic);
  registrarDesmontar?.(desmontar);
  pintar();
  return Object.freeze({ desmontar });
}
