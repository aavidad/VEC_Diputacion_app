import { crearControladorPortal } from "./portal-eventos.js?v=20260926-portal-rrhh-main-v1";
import { crearPresentadorPanelInterno } from "./portal-panel-interno.js?v=20260925-reglas-vigentes-v1";
import { extraerDatosEnvelopeCanonico } from "./portal-contrato.js?v=20260925-sin-demo2-v1";
import { crearClientePropuestasLlamamiento } from "./portal-llamamientos-api.js?v=20260718-llamamientos-v1";
import { resolverSolicitudPropuestaLlamamiento } from "./portal-llamamientos-flujo.js?v=20260925-sin-demo-v1";
import { AYUDA_PORTAL_BOLSA, detectarContextoContratacionTemporal, obtenerAyudaContratacionTemporal, renderizarAyudaContratacionTemporal, TRAMITES_AYUDANTE_PORTAL } from "./ayuda-contenido.js?v=20260926-portal-rrhh-main-v1";
import { crearAyudanteTramites } from "./ayudante-tramites.js?v=20260926-portal-rrhh-main-v1";
import { crearSuperficieBorradoresPortal } from "./portal-borradores-ui.js?v=20260926-portal-rrhh-main-v1";
import { crearUtilidadesVista } from "./portal-vistas-utilidades.js?v=20260926-portal-rrhh-main-v1";
import { crearVistasOperaciones } from "./portal-vistas-operaciones.js?v=20260924-f2-shell-v1";
import { CLAVES_SIN_ENTRADA_PORTAL, CODIGO_CARGA_SUSTITUIDA, crearCoordinadorModulosPortal, moduloDeVistaPortal, rutaDeVistaPortal, vistaConEntradaPortal, VISTA_DOCUMENTOS_EXPEDIENTE, VISTAS_MODULOS_PERSONALES } from "./portal-modulos-coordinador.js?v=20260926-ct-avisos-via-v1";
import { crearTraductorDocumentos } from "./modulos/documentos/i18n.js?v=20260925-documentos-web-v3";
import { consultarSesionPortal, presentarSesionPortal } from "./portal-catalogo-modulos.js?v=20260926-portal-rrhh-main-v1";
import { crearTraductorPersonal } from "./modulos/personal/i18n.js?v=20260925-personal-e10-v1";
import { crearVistaInicioPortal } from "./portal-inicio.js?v=20260926-portal-rrhh-main-v1";
import { accesoBolsaEfectivo, aplicarDisponibilidadMenuBolsa, instalarMenuBolsa, resumenAccesosModulos, sincronizarMenuBolsa, vistaBolsaNavegable, VISTAS_INTERNAS_BOLSA } from "./portal-menu-bolsa.js?v=20260926-portal-rrhh-main-v1";
import { traducirPortal } from "./portal-i18n.js?v=20260926-portal-rrhh-main-v1";
import { crearControladorBolsas } from "./portal-bolsas-api.js?v=20260926-reglas-ejemplo-v3";
import { crearSuperficieBorradorLlamamiento } from "./portal-borrador-llamamiento-ui.js?v=20260921-bback01-v1";
import { consultarAvisosBolsa, manejarAccionAvisos } from "./portal-bolsas-avisos.js?v=20260925-aspecto-v1";
import { crearSuperficieOfertasBolsa } from "./portal-bolsas-ofertas.js?v=20260925-ofertas-bolsa-v1";
const TAMANO_PAGINA_MARCO = 6; const tablasPaginadas = new WeakMap(); export function calcularPaginaMarco(total, paginaSolicitada, tamano = TAMANO_PAGINA_MARCO) { const cantidad = Number.isSafeInteger(total) && total > 0 ? total : 0; const medida = Number.isSafeInteger(tamano) && tamano > 0 ? tamano : TAMANO_PAGINA_MARCO; const paginas = Math.max(1, Math.ceil(cantidad / medida)); const pagina = Math.min(Math.max(Number.isSafeInteger(paginaSolicitada) ? paginaSolicitada : 1, 1), paginas); const inicio = cantidad === 0 ? 0 : ((pagina - 1) * medida) + 1; const fin = Math.min(pagina * medida, cantidad); return Object.freeze({ total: cantidad, tamano: medida, paginas, pagina, inicio, fin }); } function navegadorRemotoDeTabla(contenedor) { const padre = contenedor.parentElement; return padre?.querySelector(":scope > .ct-exp-paginacion, :scope > .paginacion-bolsa, :scope > nav[aria-label*='aginación'], :scope > nav[aria-label*='aginacion']") || null; } function botonesPaginaMarco(calculo) {
  const paginas = [1, calculo.pagina - 1, calculo.pagina, calculo.pagina + 1, calculo.paginas]
    .filter((pagina) => pagina >= 1 && pagina <= calculo.paginas)
    .filter((pagina, indice, lista) => lista.indexOf(pagina) === indice)
    .sort((a, b) => a - b);
  return paginas.map((pagina, indice) => {
    const puntos = indice > 0 && pagina - paginas[indice - 1] > 1
      ? '<span class="paginacion-marco__puntos" aria-hidden="true">…</span>' : '';
    return `${puntos}<button type="button" data-paginacion-marco-pagina="${pagina}" ${pagina === calculo.pagina ? 'aria-current="page"' : ''} aria-label="Página ${pagina}">${pagina}</button>`;
  }).join('');
}
function pintarPaginacionMarco(tabla, paginaSolicitada = 1) {
  const estado = tablasPaginadas.get(tabla);
  if (!estado || !tabla.isConnected) return;
  const filas = Array.from(tabla.tBodies?.[0]?.rows || [])
    .filter((fila) => !fila.hasAttribute('data-personal-rpt-publica-detalle-fila') && !fila.hasAttribute('data-ct-exp-resumen-fila'));
  const calculo = calcularPaginaMarco(filas.length, paginaSolicitada, estado.tamano);
  filas.forEach((fila, indice) => {
    fila.hidden = indice < calculo.inicio - 1 || indice >= calculo.fin;
    const detalle = fila.nextElementSibling;
    if (detalle?.hasAttribute('data-ct-exp-resumen-fila')) {
      detalle.hidden = fila.hidden || fila.querySelector('[data-ct-exp-resumen]')?.getAttribute('aria-expanded') !== 'true';
    }
  });
  estado.pagina = calculo.pagina;
  estado.navegacion.innerHTML = `<span aria-live="polite">${traducirPortal('paginacion_marco_recuento', calculo)}</span><span class="paginacion-marco__paginas"><button type="button" data-paginacion-marco-accion="primera" ${calculo.pagina === 1 ? 'disabled' : ''}>${traducirPortal('paginacion_marco_primera')}</button><button type="button" data-paginacion-marco-accion="anterior" ${calculo.pagina === 1 ? 'disabled' : ''}>${traducirPortal('paginacion_marco_anterior')}</button>${botonesPaginaMarco(calculo)}<button type="button" data-paginacion-marco-accion="siguiente" ${calculo.pagina === calculo.paginas ? 'disabled' : ''}>${traducirPortal('paginacion_marco_siguiente')}</button></span>`;
} function prepararTablaPaginable(contenedor) { const tabla = contenedor.querySelector(":scope > table"); if (!tabla || tablasPaginadas.has(tabla)) return; const remoto = navegadorRemotoDeTabla(contenedor); if (remoto) { remoto.classList.add("paginacion-marco", "paginacion-marco--remota"); contenedor.parentElement?.classList.add("marco-tabla-paginado"); return; } const filas = Array.from(tabla.tBodies?.[0]?.rows || []).filter((fila) => !fila.hasAttribute('data-ct-exp-resumen-fila')); if (filas.length <= TAMANO_PAGINA_MARCO) return; const navegacion = document.createElement("nav"); navegacion.className = "paginacion-marco"; navegacion.setAttribute("aria-label", traducirPortal("paginacion_marco_etiqueta")); contenedor.insertAdjacentElement("afterend", navegacion); contenedor.parentElement?.classList.add("marco-tabla-paginado"); tablasPaginadas.set(tabla, { navegacion, pagina: 1, tamano: TAMANO_PAGINA_MARCO }); pintarPaginacionMarco(tabla); } function actualizarPaginacionesMarco() { document.querySelectorAll("#espacio-trabajo .tabla-contenedor").forEach(prepararTablaPaginable); } function instalarPaginacionMarco() { const espacio = porId("espacio-trabajo"); if (!espacio) return; espacio.addEventListener("click", (evento) => { const control = evento.target.closest("[data-paginacion-marco-accion], [data-paginacion-marco-pagina]"); if (!control || control.disabled) return; const navegacion = control.closest(".paginacion-marco"); const tabla = navegacion?.previousElementSibling?.querySelector(":scope > table"); const estado = tabla && tablasPaginadas.get(tabla); if (!estado) return; const pagina = control.dataset.paginacionMarcoPagina ? Number(control.dataset.paginacionMarcoPagina) : control.dataset.paginacionMarcoAccion === "primera" ? 1 : estado.pagina + (control.dataset.paginacionMarcoAccion === "siguiente" ? 1 : -1); pintarPaginacionMarco(tabla, pagina); tabla.querySelector("tbody tr:not([hidden])")?.querySelector("button, a, [tabindex]")?.focus?.({ preventScroll: true }); }); new MutationObserver(actualizarPaginacionesMarco).observe(espacio, { childList: true, subtree: true }); actualizarPaginacionesMarco(); }
// El portal usa la API interna; Borradores mantiene su cliente autenticado, CAS e idempotencia.
const DATOS_VACIOS = Object.freeze({
  esquema: "vec.bolsa.panel.no-cargado.v1",
  sesion: null,
  indicadores: {},
  distribucion_global: {},
  series: {},
  avisos: [],
  capacidades: {},
  configuracion_llamamiento: {},
  catalogos_llamamiento: {},
  bolsas: [],
  necesidades_llamamiento: [],
  elaboraciones: [],
  proximos: [],
  actividad: [],
  contratos: [],
  reglas: [],
  documentos: [],
  canales: [],
  solicitudes: [],
  meritos_revision: [],
  criterios_baremo: [],
  ranking: [],
  alegaciones: [],
  importaciones: [],
  auditoria_eventos: [],
  auditoria: {},
});
const DATOS_PANEL = DATOS_VACIOS;
const clientePropuestasLlamamiento = crearClientePropuestasLlamamiento();
const TITULOS = Object.freeze({
  portal: ["Portal del Empleado", "Portal del Empleado"],
  resumen: ["Portal del Empleado → Bolsas de trabajo", "Cuadro de mando"],
  elaboracion: ["Portal del Empleado → Bolsas de trabajo", "Borradores de convocatorias"],
  convocatorias: ["Portal del Empleado → Bolsas de trabajo", "Convocatorias, bases y calendario"],
  solicitudes: ["Portal del Empleado → Bolsas de trabajo", "Solicitudes y admisión"],
  meritos: ["Portal del Empleado → Bolsas de trabajo", "Revisión de méritos"],
  baremacion: ["Portal del Empleado → Bolsas de trabajo", "Baremación y ranking"],
  alegaciones: ["Portal del Empleado → Bolsas de trabajo", "Alegaciones"],
  importacion: ["Portal del Empleado → Bolsas de trabajo", "Importación Convoca"],
  llamamientos: ["Portal del Empleado → Bolsas de trabajo → Llamamientos", "Nuevo llamamiento"],
  contratos: ["Portal del Empleado → Bolsas de trabajo", "Contratos, ceses y reincorporaciones"],
  reglas: ["Portal del Empleado → Bolsas de trabajo", "Reglas y versiones"],
  consulta: ["Portal del Empleado → Bolsas de trabajo", "Consulta segura para candidatos"],
  estadisticas: ["Portal del Empleado → Bolsas de trabajo", "Estadísticas"],
  documentos: ["Portal del Empleado → Bolsas de trabajo", "Documentos y firma"],
  comunicaciones: ["Portal del Empleado → Bolsas de trabajo", "Correo y mensajería"],
  auditoria: ["Portal del Empleado → Bolsas de trabajo", "Auditoría y trazabilidad"],
  configuracion: ["Portal del Empleado → Bolsas de trabajo", "Configuración y roles"],
  cronos: [traducirPortal("cronos_miga"), traducirPortal("cronos_jornada_titulo")],
  "cronos-permisos": [traducirPortal("cronos_permisos_miga"), traducirPortal("cronos_permisos_titulo")],
  dietas: ["Portal del Empleado → Dietas", "Dietas y comisiones de servicio"],
  personal: ["Portal del Empleado → Personal", "Personal · consulta informativa"],
  "personal-registro": [crearTraductorPersonal()("registro_b2_miga"), crearTraductorPersonal()("registro_b2_titulo")],
  [VISTA_DOCUMENTOS_EXPEDIENTE]: [crearTraductorDocumentos()("miga"), crearTraductorDocumentos()("titulo")],
  "bolsa-candidatos": ["Portal del Empleado → Bolsas de trabajo", "Candidatos de la bolsa"],
  "contratacion-temporal": [
    traducirPortal("contratacion_temporal_miga"),
    traducirPortal("contratacion_temporal_titulo"),
  ],
});
const VISTAS_BOLSA_SIN_LECTURA = new Set([
  "contratos",
]);
const requiereLecturaBolsas = (vista) => moduloDeVistaPortal(vista) === "bolsa"
  && !VISTAS_BOLSA_SIN_LECTURA.has(vista);
const estado = {
  vista: "portal",
  fuenteLista: false,
  errorFuente: "",
  pasoLlamamiento: 1,
  necesidadSeleccionada: "",
  elaboracionSeleccionada: "",
  propuestaLlamamiento: null,
  confirmacionPropuestaLlamamiento: null,
  configuracionLlamamiento: null,
  erroresConfiguracionLlamamiento: [],
  reciboLlamamiento: null,
  solicitandoPropuesta: false,
  errorPropuesta: "",
  filtros: {
    convocatorias: Object.freeze({ texto: "", estado: "Todos", unidad: "Todas" }),
    solicitudes: Object.freeze({ referencia: "", convocatoria: "Todas", estado: "Todos" }),
    meritos: Object.freeze({ referencia: "", tipo: "Todos", estado: "Todos" }),
  },
  bolsaSeleccionada: "",
  datosBolsas: null,
  datosCandidatos: null,
  datosEstadisticas: null,
  datosAvisos: null,
  filtrosBolsa: { estado: "", texto: "" },
  modalContactos: null,
  modalFicha: null,
  modalResultado: null,
};
const porId = (id) => document.getElementById(id);
function cerrarMenuMovil({ restaurarFoco = false } = {}) {
  delete document.body.dataset.menuAbierto;
  const boton = porId("boton-menu");
  boton?.setAttribute("aria-expanded", "false");
  if (porId("velo-menu")) porId("velo-menu").hidden = true;
  if (restaurarFoco) boton?.focus({ preventScroll: true });
}
function escaparHTML(valor) {
  return String(valor ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}
function numero(valor, decimales = 0) {
  return new Intl.NumberFormat("es-ES", {
    minimumFractionDigits: decimales,
    maximumFractionDigits: decimales,
  }).format(Number(valor || 0));
}
function esPerfilRRHH() {
  return typeof coordinadorModulos.esPerfilRRHH === "function"
    ? coordinadorModulos.esPerfilRRHH()
    : false;
}
// Sesión del núcleo: una consulta compartida por la cabecera y por la carga de
// módulos. Si falla se vuelve a pedir en la siguiente carga.
let promesaSesion = null;
function obtenerSesion() {
  promesaSesion ??= consultarSesionPortal().catch((error) => { promesaSesion = null; throw error; });
  return promesaSesion;
}
const coordinadorModulos = crearCoordinadorModulosPortal({ escaparHTML, anunciar,
  consultarSesion: () => obtenerSesion(),
  montajeBolsa: Object.freeze({
    disponible: (vista) => VISTAS_INTERNAS_BOLSA.includes(vista) && !vista.startsWith("seleccion-"),
    montar: ({ vista, raiz, opciones }) => {
      let desmontarRegistrado = null;
      const registrarDesmontar = (limpiar) => { desmontarRegistrado = limpiar; };
      montarVistaBolsa(vista, raiz, opciones);
      return Object.freeze({ desmontar: () => { if (vista === "elaboracion") superficieBorradoresActiva()?.desmontar();
        controladorBolsas.cancelarPeticiones(); cancelarAvisosBolsa(); if (vista === "llamamientos") { superficieBorradorLlamamiento.desmontar(); superficieOfertasBolsa.desmontar(); } } });
    },
  }),
  confirmarOperacion: (descriptor) => window.confirm(`${descriptor.titulo}\n\n${descriptor.advertencia}\n\nReferencia: ${descriptor.referencia}`) });
const renderizarPortal = crearVistaInicioPortal({
  encabezadoVista,
  escaparHTML,
  numero,
  obtenerCatalogo: coordinadorModulos.obtenerCatalogo,
  resolverAcceso: resolverAccesoPerfil,
  esPerfilRRHH,
  obtenerMetricasCuadro: () => coordinadorModulos.obtenerMetricasCuadro?.() || null,
  obtenerTramitesInicio: () => coordinadorModulos.obtenerTramitesInicio?.() || null,
  catalogoFallido: () => estado.errorFuente !== "",
  inicioPendiente: () => coordinadorModulos.inicioPendiente?.() === true,
});
function renderizarContenidoAyuda(contexto = null) {
  const enfocarAyuda = ({ contenedor }) => {
    contenedor.querySelector(".ayuda-contextual")?.focus?.({ preventScroll: true });
  };
  if (estado.vista === "contratacion-temporal") {
    const ctx = contexto || (estado.vista === "contratacion-temporal"
      ? detectarContextoContratacionTemporal()
      : { vista: "cuadro", fase: null });
    const ayuda = obtenerAyudaContratacionTemporal(ctx.vista, ctx.fase);
    return { ...renderizarAyudaContratacionTemporal(ayuda, escaparHTML), instalar: enfocarAyuda };
  }
  if (estado.vista === "portal" && esPerfilRRHH()) {
    const ayuda = AYUDA_PORTAL_BOLSA;
    return {
      titulo: ayuda.titulo,
      contenido: `<section class="ayuda-contextual" tabindex="-1"><p>${escaparHTML(ayuda.introduccion)}</p><h3>${escaparHTML(traducirPortal("ayuda_pasos"))}</h3><ol class="lista-ayuda">${ayuda.pasos.map((paso) => `<li>${escaparHTML(paso)}</li>`).join("")}</ol><section class="faq-ayuda"><h3>${escaparHTML(traducirPortal("ayuda_preguntas"))}</h3>${ayuda.preguntas.map((item) => `<details><summary>${escaparHTML(item.pregunta)}</summary><p>${escaparHTML(item.respuesta)}</p></details>`).join("")}</section><details class="transcripcion-ayuda"><summary>${escaparHTML(traducirPortal("ayuda_transcripcion"))}</summary><p>${escaparHTML(ayuda.transcripcion)}</p></details></section>`,
      instalar: enfocarAyuda,
    };
  }
  // El ayudante solo guía por los módulos que el portal ofrece.
  return crearAyudanteTramites({ escapar: escaparHTML,
    tramites: TRAMITES_AYUDANTE_PORTAL.filter((tramite) => vistaConEntradaPortal(tramite.vista)) });
}
function porcentajeSeguro(valor) {
  const numeroValor = Number(valor);
  if (!Number.isFinite(numeroValor)) return 0;
  return Math.max(0, Math.min(100, Math.round(numeroValor * 10) / 10));
}
function disponibilidadBolsa() {
  const acceso = accesoBolsaEfectivo(superficieBorradores.obtenerAcceso(), estado.datosBolsas);
  if (acceso?.estado !== "cargando" || estado.datosBolsas?.carga === "cargando" || estado.vista === "elaboracion") return acceso;
  return { disponible: false, vista: "", estado: estado.datosBolsas?.carga === "denegado" ? "denegado" : estado.datosBolsas?.carga === "error" ? "error" : "no_disponible" };
}
function resolverAccesoPerfil(clave) {
  const disponibilidad = disponibilidadBolsa();
  const acceso = coordinadorModulos.resolverAcceso(clave, disponibilidad);
  if (acceso.disponible !== true || vistaPermitida(acceso.vista)) return acceso;
  return { ...acceso, disponible: false, vista: "", estado: "denegado",
    etiqueta: traducirPortal("permiso_perfil_denegado") };
}
function configurarInicioInstitucional() {
  const enlace = document.getElementById("enlace-inicio-institucional");
  if (!enlace) return;
  enlace.href = "#portal";
  enlace.dataset.vista = "portal";
  enlace.setAttribute("aria-label", traducirPortal("accion_ir_inicio_portal"));
}
// Capacidades reales que deciden qué entradas de Bolsa se ofrecen: el panel
// interno agregado, la API de borradores (null mientras no se ha comprobado)
// y la vista de contratación temporal a la que lleva «Documentos y firma».
function capacidadesBolsa() {
  const borradores = superficieBorradores.obtenerAcceso();
  return {
    panelInterno: estado.fuenteLista === true,
    borradores: borradores?.disponible === true ? true
      : (borradores?.estado === "denegado" || borradores?.estado === "error" ? false : null),
    contratacionTemporal: coordinadorModulos.vistaDisponible("contratacion-temporal"),
  };
}
function vistaPermitida(vista) {
  if (vista.startsWith("seleccion-")) return false;
  if (moduloDeVistaPortal(vista) === "bolsa") return vistaBolsaNavegable(vista, capacidadesBolsa());
  if (VISTAS_MODULOS_PERSONALES.has(vista)) {
    return !estado.fuenteLista || coordinadorModulos.vistaDisponible(vista);
  }
  return true;
}

function etiquetaFuentePanel() {
  return presentadorPanelInterno.etiquetaFuente() || "API interna autorizada";
}

function notaOperacionNoCompuesta() {
  return traducirPortal("operacion_no_compuesta");
}

// La cabecera muestra el nombre y el perfil de la sesión atestada; sin ellos
// queda solo el avatar, sin textos de relleno.
async function actualizarSesionVisible() {
  const sesion = porId("sesion-visible");
  if (!sesion) return;
  let datos;
  try {
    datos = presentarSesionPortal(await obtenerSesion());
  } catch {
    return;
  }
  if (!datos.nombre) return;
  sesion.querySelector(".avatar").textContent = datos.iniciales;
  sesion.querySelector("strong").textContent = datos.nombre;
  const perfil = sesion.querySelector("small");
  perfil.textContent = datos.perfil;
  perfil.hidden = datos.perfil === "";
  sesion.querySelector("[data-sesion-texto]").hidden = false;
  sesion.setAttribute("aria-label", datos.perfil ? `${datos.nombre}, ${datos.perfil}` : datos.nombre);
}
// Repinta en cuanto llega el catálogo y cada vez que termina un módulo: Inicio
// muestra las tarjetas con «Comprobando» y cada una se actualiza sola. Una vista
// de módulo pedida por el enlace se monta en cuanto está disponible, una sola
// vez; si estaba «Comprobando» y su módulo termina sin estarlo, se repinta.
let inicioComprobando = false;
// Elaboración se ofrece en el menú según su API de borradores. Se comprueba en
// cuanto el catálogo confirma Bolsa para esta sesión, en paralelo y sin
// esperarla; al responder, la superficie avisa y el menú se repinta.
function comprobarBorradoresTrasCatalogo() {
  if (!coordinadorModulos.obtenerCatalogo().some((modulo) => modulo.clave === "bolsa")) return;
  if (superficieBorradores.obtenerAcceso()?.estado !== "cargando") return;
  void superficieBorradores.comprobarDisponibilidad().catch(() => {});
}
function alCambiarModulos(clave) {
  if (clave === "catalogo") comprobarBorradoresTrasCatalogo();
  if (estado.vista === "portal") { renderizarConservandoFoco(); anunciarAccesosComprobados(); return; }
  if (coordinadorModulos.vistaGestionada(estado.vista) && estado.vistaMontada !== estado.vista
    && (coordinadorModulos.vistaDisponible(estado.vista)
      || (!coordinadorModulos.vistaPendiente(estado.vista) && estado.vistaCerrada !== estado.vista))) {
    renderizar();
    return;
  }
  actualizarNavegacionModulos();
}
// Las tarjetas no llevan región viva propia: un solo anuncio cuando ningún
// módulo queda «Comprobando», en lugar de uno por tarjeta en cada repintado.
function anunciarAccesosComprobados() {
  const comprobando = coordinadorModulos.inicioPendiente() || coordinadorModulos.obtenerCatalogo()
    .some((modulo) => coordinadorModulos.resolverAcceso(modulo.clave).estado === "cargando");
  if (comprobando) { inicioComprobando = true; return; }
  if (!inicioComprobando) return;
  inicioComprobando = false;
  anunciar(traducirPortal("inicio_accesos_comprobados"));
}
function escaparSelector(valor) {
  if (typeof globalThis.CSS?.escape === "function") return globalThis.CSS.escape(valor);
  return String(valor).replace(/["\\]/g, "\\$&");
}
// Identifica el control enfocado de Inicio por su dato estable (tarjeta,
// métrica, acceso directo o trámite) para recuperarlo tras repintar.
const AMBITOS_FOCO = ".portal-rrhh-accesos, .portal-rrhh-resumen, .portal-rrhh-tramites-seccion";
const ATRIBUTOS_FOCO = ["data-metrica", "data-ct-exp-abrir-inicio", "data-ct-exp-vista", "data-accion", "data-vista"];
function selectorFoco(activo, contenedor) {
  if (!activo || activo === contenedor || !contenedor.contains(activo)) return "";
  const tarjeta = activo.closest("[data-modulo-catalogo]");
  if (tarjeta) return `[data-modulo-catalogo="${escaparSelector(tarjeta.dataset.moduloCatalogo)}"]`;
  const ambito = activo.closest(AMBITOS_FOCO)?.classList?.[0];
  for (const atributo of ATRIBUTOS_FOCO) {
    const valor = activo.getAttribute?.(atributo);
    if (!valor) continue;
    return `${ambito ? `.${escaparSelector(ambito)} ` : ""}${activo.tagName.toLowerCase()}[${atributo}="${escaparSelector(valor)}"]`;
  }
  return "";
}
function renderizarConservandoFoco() {
  const contenedor = porId("espacio-trabajo");
  const activo = document.activeElement;
  const selector = contenedor && activo ? selectorFoco(activo, contenedor) : "";
  renderizar();
  if (!selector) return;
  const destino = contenedor.querySelector(selector);
  const enfocable = destino?.matches?.("[data-modulo-catalogo]")
    ? (destino.querySelector("button:not([disabled])") || destino) : destino;
  // Si el control ya no existe (el Inicio neutro pasa a otro perfil), el foco
  // vuelve al contenido principal en lugar de perderse en el documento.
  (enfocable || porId("contenido-principal"))?.focus?.({ preventScroll: true });
}
// Pinta el resultado de una carga sin volver a montar la vista que ya montó
// durante la carga y conservando el foco en Inicio.
function renderizarTrasCarga() {
  if (estado.vista === "portal") renderizarConservandoFoco();
  else if (estado.vistaMontada !== estado.vista) renderizar();
}
let secuenciaFuente = 0;
async function cargarFuenteDatos() {
  const intento = ++secuenciaFuente;
  estado.errorFuente = "";
  estado.vistaMontada = "";
  estado.vistaCerrada = "";
  // La disponibilidad de Bolsa en Inicio y en el menú la decide la API real del
  // cuadro de bolsas: se consulta en paralelo con el catálogo, sin esperarla ni
  // bloquear los demás módulos. Borradores se comprueba al llegar el catálogo.
  if (requiereLecturaBolsas(estado.vista) || estado.datosBolsas?.carga !== "listo") void controladorBolsas.cargarBolsas();
  await coordinadorModulos.cargarInterno({ alCambiar: alCambiarModulos }).catch((error) => {
    // Una carga sustituida por otra más reciente no es un fallo del catálogo.
    if (error?.codigo === CODIGO_CARGA_SUSTITUIDA || intento !== secuenciaFuente) return;
    estado.errorFuente = traducirPortal("error_catalogo_modulos");
  });
  if (intento !== secuenciaFuente) return;
  actualizarNavegacionModulos();
  renderizarTrasCarga();
}

function necesidadLlamamientoSeleccionada() {
  return DATOS_PANEL.necesidades_llamamiento.find((item) => item.id === estado.necesidadSeleccionada)
    || DATOS_PANEL.necesidades_llamamiento[0];
}

function puedeSolicitarPropuesta() { return DATOS_PANEL.capacidades.solicitar_propuesta_llamamiento === true; }

async function solicitarPropuestaLlamamiento() {
  const necesidad = necesidadLlamamientoSeleccionada();
  if (!necesidad) return { ok: false, mensaje: "Seleccione una necesidad de cobertura." };
  if (estado.solicitandoPropuesta) return { ok: false, mensaje: "La solicitud ya está en curso." };

  estado.solicitandoPropuesta = true;
  estado.errorPropuesta = "";
  renderizar();
  try {
    const resultado = await resolverSolicitudPropuestaLlamamiento({
      necesidadId: necesidad.id, capacidad: puedeSolicitarPropuesta(),
      cliente: clientePropuestasLlamamiento,
    });
    if (!resultado.ok) throw new Error(resultado.mensaje);
    estado.confirmacionPropuestaLlamamiento = resultado.confirmacion;
    return resultado;
  } catch (error) {
    estado.errorPropuesta = error instanceof Error ? error.message : "No se pudo obtener la propuesta.";
    return { ok: false, mensaje: estado.errorPropuesta };
  } finally {
    estado.solicitandoPropuesta = false;
    renderizar();
  }
}

function vistaDesdeHash() {
  const valor = window.location.hash.replace(/^#\/?/, "").trim();
  if (!valor || valor === "portal") {
    if (window.location.hash !== "#portal") history.replaceState(null, "", "#portal");
    return "portal";
  } const candidata = valor.split("/").filter(Boolean).at(-1);
  if (Object.hasOwn(TITULOS, candidata)) return candidata;
  history.replaceState(null, "", "#portal");
  return "portal";
}

function rutaDeVista(vista) { return rutaDeVistaPortal(vista); }
function tituloDeVista(vista) { return TITULOS[vista] || TITULOS.portal; }
function moduloActivoDeVista(vista) { return moduloDeVistaPortal(vista); }
function actualizarNavegacionModulos() {
  const contenedor = porId("navegacion-modulos-dinamica");
  if (!contenedor) return;
  const moduloActivo = moduloActivoDeVista(estado.vista);
  const disponibilidad = disponibilidadBolsa();
  contenedor.innerHTML = coordinadorModulos.renderizarNavegacion(disponibilidad, moduloActivo, vistaPermitida);
  aplicarDisponibilidadMenuBolsa(porId("navegacion-bolsa"), capacidadesBolsa())
    .forEach((indicador, indice) => { indicador.textContent = String(indice + 1); });
  const fase = porId("texto-estado-modulos-portal");
  if (fase) {
    const accesos = ["bolsa", "contratacion_temporal", "cronos", "dietas"]
      .filter((clave) => !CLAVES_SIN_ENTRADA_PORTAL.includes(clave))
      .map((clave) => resolverAccesoPerfil(clave));
    fase.textContent = estado.errorFuente || resumenAccesosModulos(accesos,
      estado.datosBolsas?.carga === "cargando");
  }
}

function anunciar(mensaje) {
  const region = porId("anuncios");
  if (!region) return;
  region.textContent = "";
  window.setTimeout(() => { region.textContent = mensaje; }, 20);
}
function navegar(vista, opciones = {}) {
  if (!Object.hasOwn(TITULOS, vista)) return;
  if (!vistaPermitida(vista)) {
    const vistaSegura = "portal";
    const hashSeguro = rutaDeVista(vistaSegura);
    if (window.location.hash !== hashSeguro) history.replaceState(null, "", hashSeguro);
    estado.vista = vistaSegura;
    renderizar();
    cerrarMenuMovil();
    if (opciones.enfocar !== false) porId("contenido-principal")?.focus({ preventScroll: true });
    anunciar("La vista solicitada no está autorizada para el perfil activo");
    return;
  }
  const hash = rutaDeVista(vista);
  if (window.location.hash !== hash) history.pushState(null, "", hash);
  estado.vista = vista;
  estado.opcionesVista = opciones;
  renderizar();
  if (requiereLecturaBolsas(vista) && estado.datosBolsas === null) void controladorBolsas.cargarBolsas();
  cerrarMenuMovil();
  if (opciones.enfocar !== false) porId("contenido-principal")?.focus({ preventScroll: true });
  anunciar(`Vista ${tituloDeVista(vista)[1]} abierta`);
}
function montarVistaBolsa(vista, contenedor, opciones = {}, { activar = true } = {}) {
  if (vista === "contratos") {
    contenedor.innerHTML = vistasOperaciones.renderizarContratos({ contratos_fuente: { estado: "no_configurado" } });
    return;
  }
  if (vista === "llamamientos" && !estado.bolsaSeleccionada) {
    contenedor.innerHTML = renderizarLlamamientoSinBolsa();
    return;
  }
  if (vista === "elaboracion") {
    const superficie = superficieBorradores;
    if (superficie === null) { contenedor.innerHTML = renderizarFuenteNoDisponible(); return; }
    contenedor.innerHTML = superficie.renderizar();
    if (activar) void superficie.activar({ referencia: opciones.referencia });
    return;
  }
  const vistaBolsas = vista === "resumen" || vista === "bolsa-candidatos";
  if (vista === "estadisticas") {
    contenedor.innerHTML = presentadorPanelInterno.renderizarEstadisticasBolsa();
    if (estado.datosEstadisticas === null) void controladorBolsas.cargarEstadisticas();
    return;
  }
  if (vistaBolsas && !presentadorPanelInterno.esActivo() && estado.datosBolsas) {
    contenedor.innerHTML = presentadorPanelInterno.renderizarSoloBolsas(vista);
    if (vista === "resumen" && estado.datosAvisos === null) void cargarAvisosBolsa();
    return;
  }
  if (!estado.fuenteLista) {
    // Con una bolsa elegida, el llamamiento usa su propia API de borradores:
    // no depende del panel interno agregado ni muestra su aviso.
    if (vista === "llamamientos") { superficieBorradorLlamamiento.activar();
      contenedor.innerHTML = `${encabezadoVista("", tituloDeVista(vista)[1], "")}${superficieBorradorLlamamiento.renderizar()}${superficieOfertasBolsa.renderizar()}`;
      superficieOfertasBolsa.activar(estado.bolsaSeleccionada); return; }
    contenedor.innerHTML = renderizarFuenteNoDisponible(); return;
  }
  if (vistaBolsas && presentadorPanelInterno.esActivo()) {
    contenedor.innerHTML = presentadorPanelInterno.renderizarVista(vista); return;
  }
  contenedor.innerHTML = renderizarFuenteNoDisponible();
}

function renderizarLlamamientoSinBolsa() {
  return `${encabezadoVista("", "Nuevo llamamiento", "")}
    <section class="panel"><div class="cuerpo-panel vacio-controlado" role="status">
      <p><strong>Elija una bolsa para iniciar un llamamiento.</strong></p>
      <div class="acciones-vista"><a class="boton-primario" href="#bolsa/resumen">Ir al cuadro de bolsas</a></div>
    </div></section>`;
}
function actualizarVistaBolsa({ activar = false } = {}) { actualizarNavegacionModulos();
  const contenedor = porId("espacio-trabajo");
  if (contenedor && VISTAS_INTERNAS_BOLSA.includes(estado.vista)) {
    if (estado.vista.startsWith("seleccion-")) { renderizar(); return; }
    montarVistaBolsa(estado.vista, contenedor, {}, { activar });
    return;
  }
  renderizar();
}
function renderizar() {
  const contenedor = porId("espacio-trabajo");
  if (!contenedor) return;
  if (!vistaPermitida(estado.vista)) {
    estado.vista = "portal";
    history.replaceState(null, "", rutaDeVista("portal"));
  }
  const [migas, titulo] = tituloDeVista(estado.vista);
  const moduloActivo = moduloActivoDeVista(estado.vista);
  porId("migas-pan").textContent = migas;
  porId("titulo-vista").textContent = titulo;
  document.querySelector('[data-accion="ayuda"]')?.setAttribute("aria-label",
    traducirPortal("ayuda_abrir_contextual", { contexto: titulo }));
  porId("navegacion-bolsa").hidden = moduloActivo !== "bolsa";
  actualizarNavegacionModulos();

  document.querySelectorAll("[data-vista]").forEach((boton) => {
    const actual = boton.dataset.moduloPortal
      ? boton.dataset.moduloPortal === moduloActivo
      : boton.dataset.vista === estado.vista;
    if (actual) boton.setAttribute("aria-current", "page");
    else boton.removeAttribute("aria-current");
  });
  sincronizarMenuBolsa(porId("navegacion-bolsa"), estado.vista);
  const pendiente = estado.vista === "portal" ? coordinadorModulos.inicioPendiente()
    : coordinadorModulos.vistaPendiente(estado.vista);
  if (pendiente) contenedor.setAttribute("aria-busy", "true");
  else contenedor.removeAttribute("aria-busy");
  if (coordinadorModulos.vistaGestionada(estado.vista)) {
    if (!coordinadorModulos.vistaDisponible(estado.vista)) {
      coordinadorModulos.retirarVistaMontada();
      estado.vistaMontada = "";
      // Ya pintada como no disponible: otro módulo que termine no la repinta.
      estado.vistaCerrada = pendiente ? "" : estado.vista;
      // Su módulo aún carga: «Comprobando», no los textos de «no disponible».
      if (pendiente) contenedor.innerHTML = renderizarVistaComprobando(titulo);
      else contenedor.innerHTML = estado.vista === "contratacion-temporal"
        ? renderizarContratacionTemporalNoDisponible()
        : renderizarFuenteNoDisponible();
      return;
    }
    estado.vistaCerrada = "";
    const opcionesMontaje = estado.opcionesVista || {};
    estado.opcionesVista = null;
    estado.vistaMontada = estado.vista;
    void coordinadorModulos.montarVista(estado.vista, contenedor, opcionesMontaje).catch((error) => {
      contenedor.innerHTML = `${encabezadoVista(
        traducirPortal("estado_modulo_no_disponible_titulo"),
        titulo,
        traducirPortal("descripcion_superficie_no_montada"),
      )}<section class="panel"><div class="cuerpo-panel vacio-controlado"><p>${escaparHTML(error instanceof Error ? error.message : "Error de composición")}</p></div></section>`;
    });
    return;
  }

  coordinadorModulos.retirarVistaMontada();
  estado.vistaMontada = "";
  if (estado.vista !== "portal" && !estado.fuenteLista) {
    contenedor.innerHTML = renderizarFuenteNoDisponible();
    return;
  }

  const renderizadores = {
    portal: renderizarPortal,
  };
  contenedor.innerHTML = (renderizadores[estado.vista] || renderizarPortal)();
  aplicarBarrasDinamicas(contenedor);
}

// Vista pedida por el enlace cuyo módulo aún carga: ni error ni «no disponible».
function renderizarVistaComprobando(titulo) {
  return `${encabezadoVista("", titulo, "")}
    <section class="panel"><div class="cuerpo-panel vacio-controlado" role="status">
      <p><strong>${escaparHTML(traducirPortal("estado_modulo_comprobando"))}</strong></p>
    </div></section>`;
}

function renderizarFuenteNoDisponible() {
  const cargando = estado.errorFuente === "";
  const detalle = cargando
    ? "Se está comprobando la sesión y el ámbito de acceso con la API interna."
    : estado.errorFuente;
  return `
    ${encabezadoVista("Acceso interno cerrado", "Gestión de Bolsas no disponible", detalle)}
    <section class="panel">
      <div class="cuerpo-panel vacio-controlado">
        <p><strong>${cargando ? "Comprobando acceso…" : "No se han cargado datos de Bolsa"}</strong></p>
        <p>${escaparHTML(detalle)}</p>
        <div class="acciones-vista">
          <button type="button" class="boton-secundario" data-vista="portal">${escaparHTML(traducirPortal("accion_volver_portal"))}</button>
          ${cargando ? "" : '<button type="button" class="boton-primario" data-accion="recargar-fuente">Reintentar</button>'}
        </div>
      </div>
    </section>`;
}

function renderizarContratacionTemporalNoDisponible() {
  return `
    ${encabezadoVista(
      traducirPortal("contratacion_temporal_encabezado"),
      traducirPortal("estado_modulo_no_disponible_titulo"),
      traducirPortal("contratacion_temporal_descripcion_no_disponible"),
    )}
    <section class="panel">
      <div class="cuerpo-panel vacio-controlado" role="status">
        <p><strong>${escaparHTML(traducirPortal("estado_modulo_no_disponible_titulo"))}</strong></p>
        <p>${escaparHTML(traducirPortal("contratacion_temporal_aviso_no_disponible"))}</p>
        <div class="acciones-vista">
          <button type="button" class="boton-secundario" data-vista="portal">${escaparHTML(traducirPortal("accion_volver_portal"))}</button>
        </div>
      </div>
    </section>`;
}

// La sobrelínea y la descripción se aceptan por compatibilidad con las vistas que aún las
// pasan, pero no se pintan: el título ya está en la cabecera y la explicación vive tras «?».
function encabezadoVista(sobrelinea, titulo, descripcion, acciones = "") {
  return `
    <header class="encabezado-vista">
      <div>
        <h2>${escaparHTML(titulo)}</h2>
      </div>
      ${acciones ? `<div class="acciones-vista">${acciones}</div>` : ""}
    </header>`;
}

const utilidadesVista = crearUtilidadesVista({
  escaparHTML, numero, claseEstado, encabezadoVista, esPresentacion: () => false,
  operacionPermitida: () => false,
});
const vistasOperaciones = crearVistasOperaciones(utilidadesVista);
function claseEstado(estadoTexto) {
  const texto = String(estadoTexto).toLowerCase();
  if (/publicad|activa|disponible|firmad|válid|complet/.test(texto)) return "exito";
  if (/borrador|pendiente|revisión|preparad/.test(texto)) return "";
  if (/error|revocad|excluid|no disponible/.test(texto)) return "peligro";
  if (/configur|recib|enviad/.test(texto)) return "info";
  return "neutro";
}

function aplicarBarrasDinamicas(contenedor) {
  contenedor.querySelectorAll("[data-ancho]").forEach((elemento) => {
    const valor = Math.max(0, Math.min(100, Number(elemento.dataset.ancho || 0)));
    elemento.style.width = `${valor}%`;
  });
  contenedor.querySelectorAll("[data-altura]").forEach((elemento) => {
    const valor = Math.max(8, Math.min(100, Number(elemento.dataset.altura || 8)));
    elemento.style.height = `${valor}%`;
  });
  contenedor.querySelectorAll("[data-anillo-a]").forEach((elemento) => {
    const a = porcentajeSeguro(elemento.dataset.anilloA);
    const b = porcentajeSeguro(elemento.dataset.anilloB);
    const c = porcentajeSeguro(elemento.dataset.anilloC);
    const limiteB = Math.min(100, a + b);
    const limiteC = Math.min(100, limiteB + c);
    elemento.style.background = `conic-gradient(#2b9ec5 0 ${a}%, #f2bc36 ${a}% ${limiteB}%, #ef8b1f ${limiteB}% ${limiteC}%, #d74646 ${limiteC}% 100%)`;
  });
}

function superficieBorradoresActiva() {
  return superficieBorradores;
}

const superficieBorradores = crearSuperficieBorradoresPortal({
  escaparHTML,
  anunciar,
  alCambiar: () => {
    if (estado.vista === "portal") renderizarConservandoFoco();
    else if (estado.vista === "elaboracion") actualizarVistaBolsa();
    else actualizarNavegacionModulos();
  },
  confirmar: (mensaje) => window.confirm(mensaje),
});
const superficieBorradorLlamamiento = crearSuperficieBorradorLlamamiento({ anunciar,
  alCambiar: () => { if (estado.vista === "llamamientos") actualizarVistaBolsa(); } });
// Ofertas publicadas de la bolsa elegida (art. 8.1): módulo propio con sus eventos.
const superficieOfertasBolsa = crearSuperficieOfertasBolsa({ anunciar,
  alCambiar: () => { if (estado.vista === "llamamientos") actualizarVistaBolsa(); } });
superficieOfertasBolsa.instalar(document);

function instalarEventosBorradores() {
  document.addEventListener("click", (evento) => {
    const boton = evento.target.closest("[data-borrador-accion]");
    if (!boton || boton.disabled) return;
    evento.preventDefault();
    void superficieBorradoresActiva()?.manejarAccion({ accion: boton.dataset.borradorAccion,
      id: boton.dataset.id, coleccion: boton.dataset.coleccion, indice: boton.dataset.indice });
  });
  document.addEventListener("input", (evento) => {
    const control = evento.target.closest("[data-borrador-ruta]");
    if (!control) return;
    const requiereRender = ["confirmar_reaplicacion", "plantilla_indice", "motivo_indice"]
      .includes(control.dataset.borradorRuta);
    const actualizado = superficieBorradoresActiva()?.actualizarCampo({ ruta: control.dataset.borradorRuta,
      valor: control.value, checked: control.checked, tipo: control.type });
    if (!actualizado) return;
    if (requiereRender) { renderizar(); return; }
    const formulario = control.closest('[data-borrador-form="editor"]');
    const indicador = formulario?.closest(".editor-borrador")?.querySelector("[data-estado-editor]");
    if (indicador) {
      indicador.textContent = "Cambios locales sin guardar";
      indicador.className = "estado-chip";
    }
    const guardar = formulario?.querySelector("[data-borrador-guardar]");
    if (guardar?.dataset.capacidad === "true") guardar.disabled = false;
  });
  document.addEventListener("submit", (evento) => {
    const formularioLlamamiento = evento.target.closest("[data-borrador-llamamiento-form]");
    if (formularioLlamamiento) {
      evento.preventDefault();
      const datos = new FormData(formularioLlamamiento);
      void superficieBorradorLlamamiento.manejarFormulario({ accion: formularioLlamamiento.dataset.accion,
        resumen: datos.get("resumen"), referencia: datos.get("referencia") }); return;
    }
    const formulario = evento.target.closest("[data-borrador-form]");
    if (!formulario) return;
    evento.preventDefault();
    if (formulario.dataset.borradorForm === "filtros") {
      const datos = new FormData(formulario);
      void superficieBorradoresActiva()?.aplicarFiltro({ texto: datos.get("texto") || "",
        categoria: datos.get("categoria") || "" });
      return;
    }
    if (typeof formulario.reportValidity === "function" && !formulario.reportValidity()) return;
    void superficieBorradoresActiva()?.guardar();
  });
}

const presentadorPanelInterno = crearPresentadorPanelInterno({
  claseEstado, encabezadoVista, escaparHTML, numero,
  obtenerDatosPanel: () => DATOS_PANEL,
  tituloVista: (vista) => TITULOS[vista]?.[1] || "Sección de Bolsa",
  obtenerDatosBolsas: () => estado.datosBolsas,
  obtenerDatosCandidatosBolsa: () => estado.datosCandidatos,
  obtenerDatosEstadisticas: () => estado.datosEstadisticas,
  obtenerDatosAvisos: () => estado.datosAvisos,
  obtenerEstadoCandidatos: () => estado.filtrosBolsa,
  obtenerModalContactos: () => estado.modalContactos,
  obtenerModalFicha: () => estado.modalFicha,
  obtenerModalResultado: () => estado.modalResultado,
});

const controladorBolsas = crearControladorBolsas({
  estado,
  renderizar: actualizarVistaBolsa,
  navegar,
  obtenerFuenteLectura: () => null,
});

let controladorAvisos = null;
function cancelarAvisosBolsa() {
  controladorAvisos?.abort();
  controladorAvisos = null;
  if (estado.datosAvisos?.carga === "cargando") estado.datosAvisos = null;
}

async function cargarAvisosBolsa({ cursor = "", forzar = false } = {}) {
  if (!forzar && estado.datosAvisos !== null) return;
  cancelarAvisosBolsa();
  const controlador = new AbortController();
  controladorAvisos = controlador;
  estado.datosAvisos = { carga: "cargando", datos: null, cursor, error: "" };
  actualizarVistaBolsa();
  const resultado = await consultarAvisosBolsa({ cursor, signal: controlador.signal });
  if (controlador.signal.aborted || controladorAvisos !== controlador) return;
  controladorAvisos = null;
  estado.datosAvisos = resultado.ok
    ? { carga: "listo", datos: resultado.datos, cursor, error: "" }
    : { carga: "error", datos: null, cursor, error: resultado.mensaje };
  actualizarVistaBolsa();
}

function instalarEventosAvisosBolsa() {
  document.addEventListener("click", (evento) => {
    const consumida = manejarAccionAvisos(evento, {
      abrirFichaB5: (bolsaRef, participacionRef) => {
        controladorBolsas.suspenderLlamamientoB7();
        estado.bolsaSeleccionada = bolsaRef;
        estado.filtrosBolsa = { estado: "", texto: "" };
        navegar("bolsa-candidatos", { enfocar: false });
        void controladorBolsas.cargarCandidatosBolsa(bolsaRef, { enfocarDestino: true })
          .then(() => controladorBolsas.abrirFicha(participacionRef));
      },
      siguiente: () => void cargarAvisosBolsa({
        cursor: estado.datosAvisos?.datos?.paginacion?.cursor_siguiente || "",
        forzar: true,
      }),
      reintentar: () => void cargarAvisosBolsa({
        cursor: estado.datosAvisos?.cursor || "",
        forzar: true,
      }),
    });
    if (consumida) evento.preventDefault();
  });
}

const controlador = crearControladorPortal({
  anunciar, cargarFuenteDatos,
  cerrarMenuMovil, comprobarDisponibilidadBorradores: superficieBorradores.comprobarDisponibilidad,
  escaparHTML, estado,
  etiquetaFuentePanel, navegar, notaOperacionNoCompuesta, numero,
  obtenerDatosPanel: () => DATOS_PANEL, porcentajeSeguro, porId, renderizar,
  renderizarContenidoAyuda, solicitarPropuestaLlamamiento, traducir: traducirPortal, vistaDesdeHash,
});

async function inicializar() {
  document.querySelectorAll("[data-i18n-portal]").forEach((elemento) => {
    elemento.textContent = traducirPortal(elemento.dataset.i18nPortal);
  });
  document.querySelectorAll("[data-i18n-portal-aria-label]").forEach((elemento) => {
    elemento.setAttribute("aria-label", traducirPortal(elemento.dataset.i18nPortalAriaLabel));
  });
  configurarInicioInstitucional();
  controlador.restaurarPreferencias();
  estado.vista = vistaDesdeHash();
  renderizar();
  controlador.instalar();
  controladorBolsas.instalar();
  instalarEventosAvisosBolsa();
  instalarMenuBolsa(porId("navegacion-bolsa"));
  instalarEventosBorradores(); instalarPaginacionMarco();
  void actualizarSesionVisible();
  // cargarFuenteDatos pinta el resultado: Inicio conservando el foco y una
  // vista de módulo solo si no se montó ya durante la carga.
  await cargarFuenteDatos();
}

document.addEventListener("DOMContentLoaded", inicializar, { once: true });
