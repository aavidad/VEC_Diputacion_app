import { crearControladorPortal } from "./portal-eventos.js?v=20260721-acceso-real-v2";
import { crearPresentadorPanelInterno } from "./portal-panel-interno.js?v=20260717-panel-interno-v1";
import { extraerDatosEnvelopeCanonico, validarPanelBolsa } from "./portal-contrato.js?v=20260717-panel-interno-v1";
import { crearClientePropuestasLlamamiento } from "./portal-llamamientos-api.js?v=20260718-llamamientos-v1";
import { resolverSolicitudPropuestaLlamamiento } from "./portal-llamamientos-flujo.js?v=20260718-llamamientos-v1";
import { crearAsistenteLlamamientos } from "./portal-llamamientos-vista.js?v=20260719-asistente-llamamientos-v2";
import { AYUDA_PORTAL_BOLSA, detectarContextoContratacionTemporal, obtenerAyudaContratacionTemporal, renderizarAyudaContratacionTemporal } from "./ayuda-contenido.js?v=20260917-ayuda-contratacion";
import { crearAyudanteTramites } from "./ayudante-tramites.js?v=20260920-ayudante-tramites-v1";
import { crearSuperficieBorradoresPortal, instalarDeeplinkAvisosBorradores } from "./portal-borradores-ui.js?v=20260921-avisos-r5-v1";
import { crearUtilidadesVista } from "./portal-vistas-utilidades.js?v=20260720-pulido-escritorio-v2";
import { crearVistasConvocatorias } from "./portal-vistas-convocatorias.js?v=20260720-pulido-escritorio-v2";
import { crearVistasBaremacion } from "./portal-vistas-baremacion.js?v=20260720-pulido-escritorio-v2";
import { crearVistaReglas } from "./portal-vistas-reglas.js?v=20260720-pulido-escritorio-v2";
import { crearVistasOperaciones } from "./portal-vistas-operaciones.js?v=20260720-pulido-escritorio-v2";
import { crearVistasGobierno } from "./portal-vistas-gobierno.js?v=20260718-formularios-v2";
import { crearCoordinadorModulosPortal, moduloDeVistaPortal, rutaDeVistaPortal, VISTAS_MODULOS_PERSONALES, VISTAS_PRESENTACION_VISUALES } from "./portal-modulos-coordinador.js?v=20260921-avisos-r5-v1";
import { crearVistaInicioPortal } from "./portal-inicio.js?v=20260721-acceso-real-v2";
import { accesoBolsaEfectivo, instalarMenuBolsa, sincronizarMenuBolsa, VISTAS_INTERNAS_BOLSA } from "./portal-menu-bolsa.js?v=20260919-acceso-bolsa-v1";
import { traducirPortal } from "./portal-i18n.js?v=20260920-personal-catalogo-v1";
import { crearControladorBolsas } from "./portal-bolsas-api.js?v=20260921-montaje-modulos-b1";
import { crearSuperficieBorradorLlamamiento } from "./portal-borrador-llamamiento-ui.js?v=20260921-bback01-v1";
const TAMANO_PAGINA_MARCO = 6; const tablasPaginadas = new WeakMap(); export function calcularPaginaMarco(total, paginaSolicitada, tamano = TAMANO_PAGINA_MARCO) { const cantidad = Number.isSafeInteger(total) && total > 0 ? total : 0; const medida = Number.isSafeInteger(tamano) && tamano > 0 ? tamano : TAMANO_PAGINA_MARCO; const paginas = Math.max(1, Math.ceil(cantidad / medida)); const pagina = Math.min(Math.max(Number.isSafeInteger(paginaSolicitada) ? paginaSolicitada : 1, 1), paginas); const inicio = cantidad === 0 ? 0 : ((pagina - 1) * medida) + 1; const fin = Math.min(pagina * medida, cantidad); return Object.freeze({ total: cantidad, tamano: medida, paginas, pagina, inicio, fin }); } function navegadorRemotoDeTabla(contenedor) { const padre = contenedor.parentElement; return padre?.querySelector(":scope > .ct-exp-paginacion, :scope > .paginacion-bolsa, :scope > nav[aria-label*='aginación'], :scope > nav[aria-label*='aginacion']") || null; } function botonesPaginaMarco(calculo) { return Array.from({ length: calculo.paginas }, (_valor, indice) => { const pagina = indice + 1; return `<button type="button" data-paginacion-marco-pagina="${pagina}" ${pagina === calculo.pagina ? 'aria-current="page"' : ""} aria-label="Página ${pagina}">${pagina}</button>`; }).join(""); } function pintarPaginacionMarco(tabla, paginaSolicitada = 1) { const estado = tablasPaginadas.get(tabla); if (!estado || !tabla.isConnected) return; const filas = Array.from(tabla.tBodies?.[0]?.rows || []).filter((fila) => !fila.hasAttribute("data-personal-rpt-publica-detalle-fila")); const calculo = calcularPaginaMarco(filas.length, paginaSolicitada, estado.tamano); filas.forEach((fila, indice) => { fila.hidden = indice < calculo.inicio - 1 || indice >= calculo.fin; }); estado.pagina = calculo.pagina; estado.navegacion.innerHTML = `<span aria-live="polite">${traducirPortal("paginacion_marco_recuento", calculo)}</span><span class="paginacion-marco__paginas"><button type="button" data-paginacion-marco-accion="anterior" ${calculo.pagina === 1 ? "disabled" : ""}>${traducirPortal("paginacion_marco_anterior")}</button>${botonesPaginaMarco(calculo)}<button type="button" data-paginacion-marco-accion="siguiente" ${calculo.pagina === calculo.paginas ? "disabled" : ""}>${traducirPortal("paginacion_marco_siguiente")}</button></span>`; } function prepararTablaPaginable(contenedor) { const tabla = contenedor.querySelector(":scope > table"); if (!tabla || tablasPaginadas.has(tabla)) return; const remoto = navegadorRemotoDeTabla(contenedor); if (remoto) { remoto.classList.add("paginacion-marco", "paginacion-marco--remota"); contenedor.parentElement?.classList.add("marco-tabla-paginado"); return; } const filas = Array.from(tabla.tBodies?.[0]?.rows || []); if (filas.length <= TAMANO_PAGINA_MARCO) return; const navegacion = document.createElement("nav"); navegacion.className = "paginacion-marco"; navegacion.setAttribute("aria-label", traducirPortal("paginacion_marco_etiqueta")); contenedor.insertAdjacentElement("afterend", navegacion); contenedor.parentElement?.classList.add("marco-tabla-paginado"); tablasPaginadas.set(tabla, { navegacion, pagina: 1, tamano: TAMANO_PAGINA_MARCO }); pintarPaginacionMarco(tabla); } function actualizarPaginacionesMarco() { document.querySelectorAll("#espacio-trabajo .tabla-contenedor").forEach(prepararTablaPaginable); } function instalarPaginacionMarco() { const espacio = porId("espacio-trabajo"); if (!espacio) return; espacio.addEventListener("click", (evento) => { const control = evento.target.closest("[data-paginacion-marco-accion], [data-paginacion-marco-pagina]"); if (!control || control.disabled) return; const navegacion = control.closest(".paginacion-marco"); const tabla = navegacion?.previousElementSibling?.querySelector(":scope > table"); const estado = tabla && tablasPaginadas.get(tabla); if (!estado) return; const pagina = control.dataset.paginacionMarcoPagina ? Number(control.dataset.paginacionMarcoPagina) : estado.pagina + (control.dataset.paginacionMarcoAccion === "siguiente" ? 1 : -1); pintarPaginacionMarco(tabla, pagina); tabla.querySelector("tbody tr:not([hidden])")?.querySelector("button, a, [tabindex]")?.focus?.({ preventScroll: true }); }); new MutationObserver(actualizarPaginacionesMarco).observe(espacio, { childList: true, subtree: true }); actualizarPaginacionesMarco(); }
/**
 * SUPERFICIE DEFINITIVA DEL PORTAL RRHH.
 * La ruta normal obtiene datos exclusivamente de la API interna protegida. El
 * juego sintético está aislado en `datos-presentacion.js` y solo se importa si
 * la URL declara `?presentacion=rrhh`. Ninguna mutación de negocio se ejecuta
 * en el navegador salvo el guardado durable de borradores, aislado tras su
 * cliente autenticado, CAS e idempotencia. Mapa de sustitución y límites:
 * docs/portal_vec/entregable_rrhh_bolsa_2026-07-17.md.
 */
const DATOS_VACIOS = Object.freeze({
  esquema: "vec.bolsa.panel.no-cargado.v1",
  demostracion: false,
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
  llamamientos_demo: [],
  comunicaciones_demo: [],
  auditoria_eventos: [],
  roles_demo: [],
  configuraciones_demo: [],
  auditoria: {},
});
let DATOS_PANEL = DATOS_VACIOS;
let obtenerPropuestaPresentacion = null;
let adaptadorPresentacion = null;
let fuenteLecturaBolsasPresentacion = null;
let superficieBorradoresPresentacion = null;
let renderizarResumenPresentacion = null;
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
  reglas: ["Portal del Empleado → Bolsas de trabajo", "Motor de reglas configurable"],
  consulta: ["Portal del Empleado → Bolsas de trabajo", "Consulta segura para candidatos"],
  estadisticas: ["Portal del Empleado → Bolsas de trabajo", "Estadísticas y explotación de datos"],
  documentos: ["Portal del Empleado → Bolsas de trabajo", "Generación y firma de documentos"],
  comunicaciones: ["Portal del Empleado → Bolsas de trabajo", "Correo y mensajería"],
  auditoria: ["Portal del Empleado → Bolsas de trabajo", "Auditoría y trazabilidad"],
  configuracion: ["Portal del Empleado → Bolsas de trabajo", "Configuración y roles"],
  cronos: ["Portal del Empleado → Cronos", "Cronos · jornada, fichajes y permisos"],
  dietas: ["Portal del Empleado → Dietas", "Dietas y comisiones de servicio"],
  personal: ["Portal del Empleado → Personal", "Personal · consulta informativa"],
  "bolsa-candidatos": ["Portal del Empleado → Bolsas de trabajo", "Candidatos de la bolsa"],
  "contratacion-temporal": [
    traducirPortal("contratacion_temporal_miga"),
    traducirPortal("contratacion_temporal_titulo"),
  ],
});
const TITULOS_PRESENTACION = Object.freeze({ "nominas-empleado": ["Portal del Empleado → Presentación RRHH", "Nóminas y retribuciones"], "solicitudes-empleado": ["Portal del Empleado → Presentación RRHH", "Solicitudes y certificados"], "meritos-empleado": ["Portal del Empleado → Presentación RRHH", "Méritos y formación"], "comunicaciones-empleado": ["Portal del Empleado → Presentación RRHH", "Comunicaciones"], "documentos-empleado": ["Portal del Empleado → Presentación RRHH", "Documentos y firma"], "aprobaciones-empleado": ["Portal del Empleado → Presentación RRHH", "Aprobaciones y portafirmas"], "auditoria-empleado": ["Portal del Empleado → Presentación RRHH", "Auditoría"], "administracion-empleado": ["Portal del Empleado → Presentación RRHH", "Administración y configuración"] });
const estado = {
  vista: "portal",
  fuenteLista: false,
  modoPresentacion: false,
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
  filtrosBolsa: { estado: "", texto: "" },
  modalContactos: null,
  modalFicha: null,
  modalLlamar: null,
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
  if (estado.modoPresentacion) {
    const perfil = perfilPresentacionSolicitado();
    return perfil === "administrador" || perfil === "tecnico";
  }
  return typeof coordinadorModulos.esPerfilRRHH === "function"
    ? coordinadorModulos.esPerfilRRHH()
    : false;
}
const coordinadorModulos = crearCoordinadorModulosPortal({ escaparHTML, anunciar,
  montajeBolsa: Object.freeze({
    disponible: (vista) => VISTAS_INTERNAS_BOLSA.includes(vista),
    montar: ({ vista, raiz, opciones }) => {
      montarVistaBolsa(vista, raiz, opciones);
      return Object.freeze({ desmontar: () => { if (vista === "elaboracion") superficieBorradoresActiva()?.desmontar();
        controladorBolsas.cancelarPeticiones(); if (vista === "llamamientos") superficieBorradorLlamamiento.desmontar(); } });
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
});
function renderizarContenidoAyuda(contexto = null) {
  if (estado.vista === "contratacion-temporal") {
    const ctx = contexto || (estado.vista === "contratacion-temporal"
      ? detectarContextoContratacionTemporal()
      : { vista: "cuadro", fase: null });
    const ayuda = obtenerAyudaContratacionTemporal(ctx.vista, ctx.fase);
    return renderizarAyudaContratacionTemporal(ayuda, escaparHTML);
  }
  if (estado.vista === "portal" && esPerfilRRHH()) {
    const ayuda = AYUDA_PORTAL_BOLSA;
    return `<section class="ayuda-contextual"><p>${escaparHTML(ayuda.introduccion)}</p><h3>Pasos</h3><ol class="lista-ayuda">${ayuda.pasos.map((paso) => `<li>${escaparHTML(paso)}</li>`).join("")}</ol><section class="ayuda-audio" aria-labelledby="titulo-audio-ayuda"><h3 id="titulo-audio-ayuda">Escuchar esta guía</h3><audio controls preload="metadata" aria-describedby="transcripcion-ayuda"><source src="${escaparHTML(ayuda.audio.src)}" type="${escaparHTML(ayuda.audio.tipo)}">Su navegador no puede reproducir este audio.</audio></section><section class="faq-ayuda"><h3>Preguntas frecuentes</h3>${ayuda.preguntas.map((item) => `<details><summary>${escaparHTML(item.pregunta)}</summary><p>${escaparHTML(item.respuesta)}</p></details>`).join("")}</section><details id="transcripcion-ayuda" class="transcripcion-ayuda"><summary>Transcripción del audio</summary><p>${escaparHTML(ayuda.transcripcion)}</p></details></section>`;
  }
  return crearAyudanteTramites({ escapar: escaparHTML });
}
function porcentajeSeguro(valor) {
  const numeroValor = Number(valor);
  if (!Number.isFinite(numeroValor)) return 0;
  return Math.max(0, Math.min(100, Math.round(numeroValor * 10) / 10));
}
function modoPresentacionSolicitado() {
  const valores = new URLSearchParams(window.location.search).getAll("presentacion");
  return valores.length === 1 && valores[0] === "rrhh";
}
function perfilPresentacionSolicitado() {
  const valores = new URLSearchParams(window.location.search).getAll("perfil");
  if (valores.length !== 1 || !["administrador", "tecnico", "funcionario"].includes(valores[0])) return null;
  return valores[0];
}
function resolverAccesoPerfil(clave) {
  const disponibilidad = estado.modoPresentacion ? estado.fuenteLista : accesoBolsaEfectivo(superficieBorradores.obtenerAcceso(), estado.datosBolsas);
  const acceso = coordinadorModulos.resolverAcceso(clave, disponibilidad);
  if (acceso.disponible !== true || vistaPermitida(acceso.vista)) return acceso;
  return { ...acceso, disponible: false, vista: "", estado: "denegado",
    etiqueta: traducirPortal("permiso_perfil_denegado") };
}
function configurarInicioInstitucional() {
  const enlace = document.getElementById("enlace-inicio-institucional");
  if (!enlace) return;
  if (estado.modoPresentacion) {
    enlace.href = "/presentacion/";
    enlace.removeAttribute("data-vista");
    enlace.setAttribute("aria-label", "Volver al selector de recorridos de la presentación");
    return;
  }
  enlace.href = "#portal";
  enlace.dataset.vista = "portal";
  enlace.setAttribute("aria-label", "Ir al inicio del Portal del Empleado");
}
function vistaPermitida(vista) {
  if (estado.modoPresentacion && perfilPresentacionSolicitado() === null) return vista === "portal";
  if (estado.modoPresentacion && VISTAS_PRESENTACION_VISUALES.has(vista)) return !estado.fuenteLista || coordinadorModulos.vistaDisponible(vista);
  if (VISTAS_MODULOS_PERSONALES.has(vista)) {
    return !estado.fuenteLista || coordinadorModulos.vistaDisponible(vista);
  }
  if (!estado.modoPresentacion || !estado.fuenteLista) return true;
  const vistas = DATOS_PANEL.sesion?.vistas_permitidas;
  return Array.isArray(vistas) && (vistas.includes("*") || vistas.includes(vista));
}

function aplicarRestriccionesVistas() {
  if (!estado.modoPresentacion || !estado.fuenteLista) return;
  document.querySelectorAll("[data-vista], [data-requiere-vista]").forEach((control) => {
    const vistaRequerida = control.dataset.vista || control.dataset.requiereVista;
    const permitida = vistaPermitida(vistaRequerida);
    if (permitida) {
      if (control.dataset.restringidoPerfil === "true") {
        control.disabled = false;
        control.removeAttribute("aria-disabled");
        control.removeAttribute("title");
        delete control.dataset.restringidoPerfil;
      }
      return;
    }
    control.disabled = true;
    control.setAttribute("aria-disabled", "true");
    control.setAttribute("title", "No disponible para el perfil activo");
    control.dataset.restringidoPerfil = "true";
  });
}

function etiquetaFuentePanel() {
  if (estado.modoPresentacion) return "Datos sintéticos aislados";
  return presentadorPanelInterno.etiquetaFuente() || "API interna autorizada";
}

function notaOperacionNoCompuesta() {
  return estado.modoPresentacion
    ? "Recorrido de presentación: no escribe en el servidor ni genera un acto administrativo."
    : "Esta operación permanece deshabilitada hasta que su comando de servidor esté compuesto y autorizado.";
}

function describirOperacionPresentacion(operacion, objetivo) {
  if (!estado.modoPresentacion || adaptadorPresentacion === null) return null;
  return adaptadorPresentacion.describir(operacion, objetivo);
}

function operacionPermitida(operacion) {
  if (!estado.modoPresentacion) return false;
  const operaciones = DATOS_PANEL.sesion?.operaciones_permitidas;
  return Array.isArray(operaciones) && (operaciones.includes("*") || operaciones.includes(operacion));
}

function ejecutarOperacionPresentacion(operacion, objetivo, motivo, campos) {
  if (!estado.modoPresentacion || adaptadorPresentacion === null) {
    throw new Error("el adaptador de presentación no está activo");
  }
  if (!operacionPermitida(operacion)) throw new Error("operación no autorizada para el perfil activo");
  const recibo = adaptadorPresentacion.ejecutar({ operacion, objetivo, motivo, campos });
  DATOS_PANEL = adaptadorPresentacion.obtenerDatos();
  return recibo;
}

function actualizarSesionVisible() {
  const sesion = porId("sesion-visible");
  if (!sesion) return;
  const datos = DATOS_PANEL.sesion;
  const avatar = sesion.querySelector(".avatar");
  const nombre = sesion.querySelector("strong");
  const perfil = sesion.querySelector("small");
  const avisos = document.querySelector(".boton-avisos span");
  if (estado.fuenteLista && estado.modoPresentacion && datos) {
    avatar.textContent = String(datos.iniciales || "RR").slice(0, 3);
    nombre.textContent = String(datos.nombre || "Sesión interna");
    perfil.textContent = String(datos.perfil || "Perfil autorizado");
    sesion.dataset.actorRef = String(datos.actor_ref || "");
    sesion.setAttribute("aria-label", `${nombre.textContent}. ${perfil.textContent}`);
    if (avisos) {
      avisos.textContent = numero(DATOS_PANEL.indicadores.avisos_pendientes);
      avisos.setAttribute("aria-label", `${numero(DATOS_PANEL.indicadores.avisos_pendientes)} avisos pendientes`);
    }
    return;
  }
  if (estado.fuenteLista && presentadorPanelInterno.actualizarContextoSesion({ avatar, nombre, perfil, avisos })) {
    return;
  }
  // El fallo del panel de Bolsa no determina la identidad del portal.
  avatar.textContent = "—";
  nombre.textContent = traducirPortal("contexto_portal_titulo");
  perfil.textContent = traducirPortal("contexto_portal_descripcion");
  delete sesion.dataset.actorRef;
  sesion.setAttribute("aria-label", traducirPortal("contexto_portal_accesible"));
  if (avisos) {
    avisos.textContent = "—";
    avisos.setAttribute("aria-label", "Avisos pendientes sin resolver");
  }
}
async function cargarFuenteDatos() {
  estado.modoPresentacion = modoPresentacionSolicitado();
  const aviso = document.querySelector(".aviso-presentacion");
  if (estado.modoPresentacion) {
    try {
      const [adaptador, moduloEfectos, moduloBorradores, moduloSelector, moduloResumen] = await Promise.all([
        import("./datos-presentacion.js?v=20260718-demo-total-v1"),
        import("./portal-presentacion-adaptador.js?v=20260718-demo-total-v1"),
        import("./portal-borradores-demo-cliente.js?v=20260720-pulido-escritorio-v2"),
        import("../presentacion/selector-perfiles.js?v=20260720-selector-perfiles-v1"),
        import("./portal-resumen-presentacion.js?v=20260721-acceso-real-v2"),
      ]);
      const perfil = perfilPresentacionSolicitado();
      if (perfil === null) throw new Error("perfil de presentación no permitido");
      const datosIniciales = validarPanelBolsa(adaptador.obtenerDatosPresentacion(perfil), true);
      const contextoActorBolsa = await coordinadorModulos.cargarPresentacion(datosIniciales.sesion);
      adaptadorPresentacion = moduloEfectos.crearAdaptadorPresentacion({
        datosIniciales,
        contextoActor: contextoActorBolsa,
      });
      fuenteLecturaBolsasPresentacion = moduloEfectos.crearFuenteLecturaBolsasPresentacion({ datosIniciales });
      superficieBorradoresPresentacion = crearSuperficieBorradoresPortal({
        escaparHTML,
        anunciar,
        alCambiar: () => { if (estado.vista === "elaboracion") actualizarVistaBolsa(); },
        confirmar: (mensaje) => window.confirm(mensaje),
        crearClienteImpl: () => moduloBorradores.crearClienteBorradoresPresentacion(),
      });
      renderizarResumenPresentacion = moduloResumen.crearVistaResumenPresentacion({
        escaparHTML, encabezadoVista, etiquetaFuentePanel, numero,
        obtenerDatosPanel: () => DATOS_PANEL, porcentajeSeguro,
      });
      DATOS_PANEL = adaptadorPresentacion.obtenerDatos();
      obtenerPropuestaPresentacion = adaptador.obtenerPropuestaPresentacion;
      aviso.hidden = false;
      estado.fuenteLista = true;
      estado.necesidadSeleccionada = DATOS_PANEL.necesidades_llamamiento[0]?.id || "";
      estado.elaboracionSeleccionada = DATOS_PANEL.elaboraciones[0]?.id || "";
      if (moduloDeVistaPortal(estado.vista) === "bolsa") void controladorBolsas.cargarBolsas();
      moduloSelector.instalarSelectorPerfilesPresentacion({ disparador: porId("sesion-visible"), perfilActivo: perfil });
    } catch {
      aviso.hidden = true;
      estado.errorFuente = "No se pudo cargar el adaptador aislado de presentación.";
      estado.fuenteLista = false;
    }
    actualizarSesionVisible();
    actualizarNavegacionModulos();
    return;
  }

  adaptadorPresentacion = null;
  fuenteLecturaBolsasPresentacion = null;
  superficieBorradoresPresentacion = null;
  renderizarResumenPresentacion = null;
  aviso.hidden = true;
  await coordinadorModulos.cargarInterno().catch(() => null);
  // Borradores comprueba su API al abrir la vista. B12/B5 usa su propia API compuesta.
  if (moduloDeVistaPortal(estado.vista) === "bolsa") void controladorBolsas.cargarBolsas();
  actualizarNavegacionModulos();
}

function necesidadLlamamientoSeleccionada() {
  return DATOS_PANEL.necesidades_llamamiento.find((item) => item.id === estado.necesidadSeleccionada)
    || DATOS_PANEL.necesidades_llamamiento[0];
}

function puedeSolicitarPropuesta() { return DATOS_PANEL.capacidades.solicitar_propuesta_llamamiento === true; }

async function solicitarPropuestaLlamamiento() {
  const necesidad = necesidadLlamamientoSeleccionada();
  if (!necesidad) return { ok: false, mensaje: "Seleccione una necesidad de cobertura." };
  if (estado.modoPresentacion) {
    const resultado = await resolverSolicitudPropuestaLlamamiento({
      modoPresentacion: true, necesidadId: necesidad.id, capacidad: false,
      obtenerPresentacion: obtenerPropuestaPresentacion, cliente: clientePropuestasLlamamiento,
    });
    if (resultado.ok) estado.propuestaLlamamiento = resultado.propuesta;
    return resultado;
  }
  if (estado.solicitandoPropuesta) return { ok: false, mensaje: "La solicitud ya está en curso." };

  estado.solicitandoPropuesta = true;
  estado.errorPropuesta = "";
  renderizar();
  try {
    const resultado = await resolverSolicitudPropuestaLlamamiento({
      modoPresentacion: false, necesidadId: necesidad.id, capacidad: puedeSolicitarPropuesta(),
      obtenerPresentacion: null, cliente: clientePropuestasLlamamiento,
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
  if (estado.modoPresentacion && VISTAS_PRESENTACION_VISUALES.has(candidata)) return candidata;
  history.replaceState(null, "", "#portal");
  return "portal";
}

function rutaDeVista(vista) { return estado.modoPresentacion && VISTAS_PRESENTACION_VISUALES.has(vista) ? `#${vista}` : rutaDeVistaPortal(vista); }
function tituloDeVista(vista) { return estado.modoPresentacion && VISTAS_PRESENTACION_VISUALES.has(vista) ? TITULOS_PRESENTACION[vista] || TITULOS.portal : TITULOS[vista] || TITULOS.portal; }
function moduloActivoDeVista(vista) { return estado.modoPresentacion && VISTAS_PRESENTACION_VISUALES.has(vista) ? vista.replace(/-empleado$/u, "") : moduloDeVistaPortal(vista); }
function actualizarNavegacionModulos() {
  const contenedor = porId("navegacion-modulos-dinamica");
  if (!contenedor) return;
  const moduloActivo = moduloActivoDeVista(estado.vista);
  const disponibilidad = estado.modoPresentacion ? estado.fuenteLista : accesoBolsaEfectivo(superficieBorradores.obtenerAcceso(), estado.datosBolsas);
  contenedor.innerHTML = coordinadorModulos.renderizarNavegacion(disponibilidad, moduloActivo, vistaPermitida);
  const fase = porId("texto-estado-modulos-portal");
  if (fase) {
    if (estado.datosBolsas === null && !estado.modoPresentacion) {
      fase.textContent = "Fase inicial: comprobando módulos";
      return;
    }
    const disponibles = ["bolsa", "contratacion_temporal", "cronos", "dietas"]
      .filter((clave) => resolverAccesoPerfil(clave).disponible).length;
    fase.textContent = disponibles > 0
      ? `${disponibles} módulos habilitados en fase inicial`
      : "Módulos pendientes de sesión autorizada";
  }
}

function anunciar(mensaje) {
  const region = porId("anuncios");
  if (!region) return;
  region.textContent = "";
  window.setTimeout(() => { region.textContent = mensaje; }, 20);
}
function navegar(vista, opciones = {}) {
  if (!Object.hasOwn(TITULOS, vista) && !(estado.modoPresentacion && VISTAS_PRESENTACION_VISUALES.has(vista))) return;
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
  if (moduloDeVistaPortal(vista) === "bolsa" && estado.datosBolsas === null) void controladorBolsas.cargarBolsas();
  cerrarMenuMovil();
  if (opciones.enfocar !== false) porId("contenido-principal")?.focus({ preventScroll: true });
  anunciar(`Vista ${tituloDeVista(vista)[1]} abierta`);
}
function montarVistaBolsa(vista, contenedor, opciones = {}, { activar = true } = {}) {
  if (vista === "llamamientos" && !estado.bolsaSeleccionada) {
    contenedor.innerHTML = renderizarLlamamientoSinBolsa();
    return;
  }
  if (vista === "elaboracion" && estado.modoPresentacion && !estado.fuenteLista) {
    contenedor.innerHTML = renderizarFuenteNoDisponible(); return;
  }
  if (vista === "elaboracion") {
    const superficie = estado.modoPresentacion ? superficieBorradoresPresentacion : superficieBorradores;
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
  if (vistaBolsas && !presentadorPanelInterno.esActivo() && estado.datosBolsas
    && (!estado.modoPresentacion || vista === "bolsa-candidatos")) {
    contenedor.innerHTML = presentadorPanelInterno.renderizarSoloBolsas(vista); return;
  }
  if (!estado.fuenteLista) {
    if (vista === "llamamientos") { superficieBorradorLlamamiento.activar();
      contenedor.innerHTML = `${renderizarFuenteNoDisponible()}${superficieBorradorLlamamiento.renderizar()}`; return; }
    contenedor.innerHTML = renderizarFuenteNoDisponible(); return;
  }
  if (vistaBolsas && presentadorPanelInterno.esActivo()) {
    contenedor.innerHTML = presentadorPanelInterno.renderizarVista(vista); return;
  }
  const datosVista = { ...DATOS_VACIOS, ...DATOS_PANEL };
  const renderizadores = {
    resumen: () => renderizarResumenPresentacion?.() || renderizarFuenteNoDisponible(),
    convocatorias: () => vistasConvocatorias.renderizarConvocatorias(datosVista, estado), solicitudes: () => vistasConvocatorias.renderizarSolicitudes(datosVista, estado),
    meritos: () => vistasBaremacion.renderizarMeritos(datosVista, estado), baremacion: () => vistasBaremacion.renderizarBaremacion(datosVista), alegaciones: () => vistasBaremacion.renderizarAlegaciones(datosVista),
    importacion: () => vistasOperaciones.renderizarImportacion(datosVista),
    llamamientos: () => (superficieBorradorLlamamiento.activar(), `${asistenteLlamamientos.renderizar(datosVista, estado)}${superficieBorradorLlamamiento.renderizar()}`),
    contratos: () => vistasOperaciones.renderizarContratos(datosVista), reglas: () => vistaReglas.renderizarReglas(datosVista), consulta: () => renderizarConsulta(),
    estadisticas: () => vistasGobierno.renderizarEstadisticas(datosVista), documentos: () => vistasOperaciones.renderizarDocumentos(datosVista), comunicaciones: () => vistasOperaciones.renderizarComunicaciones(datosVista),
    auditoria: () => vistasGobierno.renderizarAuditoria(datosVista), configuracion: () => vistasGobierno.renderizarConfiguracion(datosVista),
  };
  contenedor.innerHTML = (renderizadores[vista] || renderizarFuenteNoDisponible)();
  aplicarBarrasDinamicas(contenedor);
}

function renderizarLlamamientoSinBolsa() {
  return `${encabezadoVista(
    "Gestión interna de Bolsas",
    "Nuevo llamamiento",
    "El llamamiento se inicia desde una bolsa concreta para conservar el orden B6 y su ámbito autorizado.",
  )}
    <section class="panel"><div class="cuerpo-panel vacio-controlado" role="status">
      <p><strong>Elija una bolsa para iniciar un llamamiento.</strong></p>
      <p>El orden del reglamento sigue provisional hasta resolver las dudas 13–14. El correo usa el relay de pruebas, no un buzón corporativo.</p>
      <div class="acciones-vista"><a class="boton-primario" href="#bolsa/resumen">Ir al cuadro de bolsas</a></div>
    </div></section>`;
}
function actualizarVistaBolsa({ activar = false } = {}) {
  const contenedor = porId("espacio-trabajo");
  if (contenedor && VISTAS_INTERNAS_BOLSA.includes(estado.vista)) {
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
  queueMicrotask(aplicarRestriccionesVistas);
  const [migas, titulo] = tituloDeVista(estado.vista);
  const moduloActivo = moduloActivoDeVista(estado.vista);
  porId("migas-pan").textContent = migas;
  porId("titulo-vista").textContent = titulo;
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
  if (coordinadorModulos.vistaGestionada(estado.vista)) {
    if (!coordinadorModulos.vistaDisponible(estado.vista)) {
      coordinadorModulos.desmontarVistaActual();
      contenedor.innerHTML = estado.vista === "contratacion-temporal"
        ? renderizarContratacionTemporalNoDisponible()
        : renderizarFuenteNoDisponible();
      return;
    }
    const opcionesMontaje = estado.opcionesVista || {};
    estado.opcionesVista = null;
    void coordinadorModulos.montarVista(estado.vista, contenedor, opcionesMontaje).catch((error) => {
      contenedor.innerHTML = `${encabezadoVista(
        traducirPortal("estado_modulo_no_disponible_titulo"),
        titulo,
        traducirPortal("descripcion_superficie_no_montada"),
      )}<section class="panel"><div class="cuerpo-panel vacio-controlado"><p>${escaparHTML(error instanceof Error ? error.message : "Error de composición")}</p></div></section>`;
    });
    return;
  }

  coordinadorModulos.desmontarVistaActual();
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

function encabezadoVista(sobrelinea, titulo, descripcion, acciones = "") {
  return `
    <header class="encabezado-vista">
      <div>
        <p class="sobrelinea">${escaparHTML(sobrelinea)}</p>
        <h2>${escaparHTML(titulo)}</h2>
        <p>${escaparHTML(descripcion)}</p>
      </div>
      ${acciones ? `<div class="acciones-vista">${acciones}</div>` : ""}
    </header>`;
}

const utilidadesVista = crearUtilidadesVista({
  escaparHTML, numero, claseEstado, encabezadoVista, esPresentacion: () => estado.modoPresentacion,
  operacionPermitida,
});
const vistasConvocatorias = crearVistasConvocatorias(utilidadesVista);
const vistasBaremacion = crearVistasBaremacion(utilidadesVista);
const vistaReglas = crearVistaReglas(utilidadesVista);
const vistasOperaciones = crearVistasOperaciones(utilidadesVista);
const vistasGobierno = crearVistasGobierno(utilidadesVista);
const asistenteLlamamientos = crearAsistenteLlamamientos({ ...utilidadesVista, operacionPermitida });
function claseEstado(estadoTexto) {
  const texto = String(estadoTexto).toLowerCase();
  if (/publicad|activa|disponible|firmad|válid|complet/.test(texto)) return "exito";
  if (/borrador|pendiente|revisión|preparad/.test(texto)) return "";
  if (/error|revocad|excluid|no disponible/.test(texto)) return "peligro";
  if (/configur|recib|enviad/.test(texto)) return "info";
  return "neutro";
}

function renderizarConsulta() {
  return `
    ${encabezadoVista("Zona externa separada", "Portal de consulta para candidatos", "Consulta pública de convocatorias y zona privada del propio aspirante, sin acceso a datos de terceros.", '<a class="boton-primario" href="/bolsa/">Abrir consulta pública</a>')}
    <div class="rejilla-dos-columnas">
      <section class="panel"><div class="cabecera-panel"><h3>Consulta pública disponible</h3><span class="estado-chip exito">API pública separada</span></div><div class="cuerpo-panel"><dl class="resumen-expediente"><div class="fila-resumen"><dt>Convocatorias</dt><dd>Listado y detalle con filtros</dd></div><div class="fila-resumen"><dt>Categorías</dt><dd>Catálogo profesional gobernado</dd></div><div class="fila-resumen"><dt>Contenido</dt><dd>Bases, requisitos, plazos, ayuda y documentos</dd></div><div class="fila-resumen"><dt>Transparencia</dt><dd>Versión y huella de la fuente publicada</dd></div></dl><a class="boton-primario" href="/bolsa/">Ver portal público</a></div></section>
      <aside class="resumen-lateral"><section class="panel"><div class="cabecera-panel"><h3>Frontera de privacidad</h3></div><ul class="lista-comprobacion"><li>Datos públicos sin autenticación</li><li>Expediente propio con identidad fuerte</li><li>Identificadores de listados minimizados</li><li>Gestión RRHH solo en red y sesión internas</li><li class="pendiente">Zona privada E2E pendiente de composición</li></ul></section></aside>
    </div>`;
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
  return estado.modoPresentacion ? superficieBorradoresPresentacion : superficieBorradores;
}

const superficieBorradores = crearSuperficieBorradoresPortal({
  escaparHTML,
  anunciar,
  alCambiar: () => {
    if (estado.vista === "portal") renderizar();
    else if (estado.vista === "elaboracion") actualizarVistaBolsa();
    else actualizarNavegacionModulos();
  },
  confirmar: (mensaje) => window.confirm(mensaje),
});
const superficieBorradorLlamamiento = crearSuperficieBorradorLlamamiento({ anunciar,
  alCambiar: () => { if (estado.vista === "llamamientos") actualizarVistaBolsa(); } });

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
  obtenerEstadoCandidatos: () => estado.filtrosBolsa,
  obtenerModalContactos: () => estado.modalContactos,
  obtenerModalFicha: () => estado.modalFicha,
  obtenerModalLlamar: () => estado.modalLlamar,
  obtenerModalResultado: () => estado.modalResultado,
  esLecturaPresentacion: () => estado.modoPresentacion,
});

const controladorBolsas = crearControladorBolsas({
  estado,
  renderizar: actualizarVistaBolsa,
  navegar,
  obtenerFuenteLectura: () => estado.modoPresentacion ? fuenteLecturaBolsasPresentacion : null,
});

const controlador = crearControladorPortal({
  anunciar, asistenteLlamamientos, cargarFuenteDatos, confirmarOperacionPresentacion: (mensaje) => window.confirm(mensaje),
  cerrarMenuMovil, comprobarDisponibilidadBorradores: superficieBorradores.comprobarDisponibilidad,
  describirOperacionPresentacion, escaparHTML, estado,
  etiquetaFuentePanel, ejecutarOperacionPresentacion, navegar, notaOperacionNoCompuesta, numero,
  obtenerDatosPanel: () => DATOS_PANEL, operacionPermitida, porcentajeSeguro, porId, renderizar,
  renderizarContenidoAyuda, solicitarPropuestaLlamamiento, traducir: traducirPortal, vistaDesdeHash,
});

async function inicializar() {
  estado.modoPresentacion = modoPresentacionSolicitado();
  configurarInicioInstitucional();
  controlador.restaurarPreferencias();
  estado.vista = vistaDesdeHash();
  renderizar();
  controlador.instalar();
  controladorBolsas.instalar();
  instalarMenuBolsa(porId("navegacion-bolsa"));
  instalarDeeplinkAvisosBorradores({ documento: document, escaparHTML, porId,
    obtenerAvisos: () => DATOS_PANEL.avisos,
    disponible: () => estado.modoPresentacion && estado.fuenteLista && vistaPermitida("elaboracion"),
    navegar, anunciar });
  instalarEventosBorradores(); instalarPaginacionMarco();
  await cargarFuenteDatos();
  renderizar();
}

document.addEventListener("DOMContentLoaded", inicializar, { once: true });
