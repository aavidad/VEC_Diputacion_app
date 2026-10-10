import { crearControladorPortal } from "./portal-eventos.js?v=20261008-ct-inicio-v1";
import { FILTRO_INCIDENCIA_CT, filtroServidorCTValido, leerFiltroCTDeRuta,
  limpiarFiltroCTDeBusqueda, rutaPortalConFiltroCT } from "./portal-ct-ruta-filtro.js?v=20261009-ct-bolsa-cohorte-v9";
import { leerFichaCTDeRuta, rutaConFichaCT } from "./portal-ct-ruta-ficha.js?v=20261008-r-fichas-idioma-nav-v1";
import { extraerDatosEnvelopeCanonico } from "./portal-contrato.js?v=20260925-sin-demo2-v1";
import { crearClientePropuestasLlamamiento } from "./portal-llamamientos-api.js?v=20261007-pantallas-textos-final-v1";
import { resolverSolicitudPropuestaLlamamiento } from "./portal-llamamientos-flujo.js?v=20261007-pantallas-textos-final-v1";
import { crearSuperficieBorradoresPortal } from "./portal-borradores-ui.js?v=20261008-borradores-error-legible-v1";
import { crearUtilidadesVista } from "./portal-vistas-utilidades.js?v=20261007-pantallas-textos-final-v1";
import { CODIGO_CARGA_SUSTITUIDA, crearCoordinadorModulosPortal, moduloDeVistaPortal, rutaDeVistaPortal, vistaConEntradaPortal, VISTA_CATEGORIAS_RPT, VISTA_DOCUMENTOS_EXPEDIENTE, VISTA_PLANTILLAS_RRHH, VISTAS_MODULOS_PERSONALES, VISTAS_AUTOSERVICIO_EMPLEADO } from "./portal-modulos-coordinador.js?v=20261009-ct-bolsa-cohorte-v9";

import { consultarSesionPortal, presentarSesionPortal } from "./portal-catalogo-modulos.js?v=20261007-pantallas-textos-final-v1";
import { crearTraductorPersonal, MENSAJES_PERSONAL } from "./modulos/personal/i18n.js?v=20261008-alta-rpt-circular-v4";
import { accesoBolsaEfectivo, aplicarDisponibilidadMenuBolsa, instalarMenuBolsa, resumenAccesosModulos, sincronizarMenuBolsa, vistaBolsaNavegable, vistaBolsaPendienteNoCompuesta, VISTA_CANDIDATOS_BOLSA, VISTAS_INTERNAS_BOLSA } from "./portal-menu-bolsa.js?v=20261009-ct-bolsa-cohorte-v9";
import { instalarSelectorLlamamientos, renderizarPantallaLlamamientos } from "./portal-llamamientos-selector.js?v=20261009-ct-bolsa-cohorte-v9";
import { LOCALIZACION_PORTAL, textoPortal, traducirPortal, prepararTextosPortal, textosGrupoPortalPreparados } from "./portal-i18n.js?v=20261007-pantallas-textos-final-v1";

import { instalarCopiaJustificantes } from "./portal-justificante.js";
import { aplicarIdiomaDocumento, aplicarTextosPortal, instalarSelectorIdiomaPortal, instalarValidacionI18n } from "./portal-idioma.js?v=20261007-pantallas-textos-final-v1";
import { consultarAvisosBolsa, manejarAccionAvisos } from "./portal-bolsas-avisos.js?v=20261007-pantallas-textos-final-v1";

import { crearFuenteAuditoriaHTTP } from "./modulos/auditoria/cliente-http.js?v=20261007-auditoria-disponibilidad-v1";
import { crearClientePoliticaCeseRRHH } from "./modulos/bolsa/rrhh-politica-cese-api.js?v=20260928-rrhh-politica-cese-v1";
import { montarVistaPoliticaCeseRRHH } from "./modulos/bolsa/rrhh-politica-cese-vista.js?v=20261007-pantallas-textos-final-v1";
import { crearIntegracionPreferenciasPortal } from "./portal-preferencias-integracion.js?v=20261010-http-codigo-v1";
let tamanoPaginaMarco = 6; const tablasPaginadas = new WeakMap(); export function calcularPaginaMarco(total, paginaSolicitada, tamano = tamanoPaginaMarco) { const cantidad = Number.isSafeInteger(total) && total > 0 ? total : 0; const medida = Number.isSafeInteger(tamano) && tamano > 0 ? tamano : tamanoPaginaMarco; const paginas = Math.max(1, Math.ceil(cantidad / medida)); const pagina = Math.min(Math.max(Number.isSafeInteger(paginaSolicitada) ? paginaSolicitada : 1, 1), paginas); const inicio = cantidad === 0 ? 0 : ((pagina - 1) * medida) + 1; const fin = Math.min(pagina * medida, cantidad); return Object.freeze({ total: cantidad, tamano: medida, paginas, pagina, inicio, fin }); } function navegadorRemotoDeTabla(contenedor) { const padre = contenedor.parentElement; return padre?.querySelector(":scope > .ct-exp-paginacion, :scope > .paginacion-bolsa, :scope > nav[aria-label*='aginación'], :scope > nav[aria-label*='aginacion']") || null; } function botonesPaginaMarco(calculo) {
  const paginas = [1, calculo.pagina - 1, calculo.pagina, calculo.pagina + 1, calculo.paginas]
    .filter((pagina) => pagina >= 1 && pagina <= calculo.paginas)
    .filter((pagina, indice, lista) => lista.indexOf(pagina) === indice)
    .sort((a, b) => a - b);
  return paginas.map((pagina, indice) => {
    const puntos = indice > 0 && pagina - paginas[indice - 1] > 1
      ? '<span class="paginacion-marco__puntos" aria-hidden="true">…</span>' : '';
    return `${puntos}<button type="button" data-paginacion-marco-pagina="${pagina}" ${pagina === calculo.pagina ? 'aria-current="page"' : ''} aria-label="${textoPortal("txt_pagina_n", { pagina })}">${pagina}</button>`;
  }).join('');
}
// Filas de detalle que acompañan a la fila anterior: no cuentan como registros
// de la tabla y se muestran u ocultan con ella (resumen de expediente CT, detalle
// RPT público y ficha de participación abierta en Bolsa).
function esFilaDetalleMarco(fila) {
  return fila.hasAttribute('data-personal-rpt-publica-detalle-fila') || fila.hasAttribute('data-ct-exp-resumen-fila')
    || fila.classList.contains('fila-ficha-participacion');
}
function filasPaginablesMarco(tabla) {
  return Array.from(tabla.tBodies?.[0]?.rows || []).filter((fila) => !esFilaDetalleMarco(fila));
}
// Página inicial: la de la fila cuya ficha de participación está abierta, para
// que la ficha no quede oculta en otra página al volver a pintar la tabla.
function paginaInicialMarco(filas, tamano) {
  const indice = filas.findIndex((fila) => fila.nextElementSibling?.classList.contains('fila-ficha-participacion'));
  return indice < 0 ? 1 : Math.floor(indice / tamano) + 1;
}
function pintarPaginacionMarco(tabla, paginaSolicitada = 1) {
  const estado = tablasPaginadas.get(tabla);
  if (!estado || !tabla.isConnected) return;
  const filas = filasPaginablesMarco(tabla);
  const calculo = calcularPaginaMarco(filas.length, paginaSolicitada, estado.tamano);
  filas.forEach((fila, indice) => {
    fila.hidden = indice < calculo.inicio - 1 || indice >= calculo.fin;
    const detalle = fila.nextElementSibling;
    if (detalle?.hasAttribute('data-ct-exp-resumen-fila')) {
      detalle.hidden = fila.hidden || fila.querySelector('[data-ct-exp-resumen]')?.getAttribute('aria-expanded') !== 'true';
    } else if (detalle?.classList.contains('fila-ficha-participacion')) {
      detalle.hidden = fila.hidden;
    }
  });
  estado.pagina = calculo.pagina;
  estado.navegacion.innerHTML = `<span aria-live="polite">${traducirPortal('paginacion_marco_recuento', calculo)}</span><span class="paginacion-marco__paginas"><button type="button" data-paginacion-marco-accion="primera" ${calculo.pagina === 1 ? 'disabled' : ''}>${traducirPortal('paginacion_marco_primera')}</button><button type="button" data-paginacion-marco-accion="anterior" ${calculo.pagina === 1 ? 'disabled' : ''}>${traducirPortal('paginacion_marco_anterior')}</button>${botonesPaginaMarco(calculo)}<button type="button" data-paginacion-marco-accion="siguiente" ${calculo.pagina === calculo.paginas ? 'disabled' : ''}>${traducirPortal('paginacion_marco_siguiente')}</button></span>`;
} function prepararTablaPaginable(contenedor) { const tabla = contenedor.querySelector(":scope > table"); if (!tabla || tablasPaginadas.has(tabla)) return; const remoto = navegadorRemotoDeTabla(contenedor); if (remoto) { remoto.classList.add("paginacion-marco", "paginacion-marco--remota"); contenedor.parentElement?.classList.add("marco-tabla-paginado"); return; } const filas = filasPaginablesMarco(tabla); if (filas.length <= tamanoPaginaMarco) return; const navegacion = document.createElement("nav"); navegacion.className = "paginacion-marco"; navegacion.setAttribute("aria-label", traducirPortal("paginacion_marco_etiqueta")); contenedor.insertAdjacentElement("afterend", navegacion); contenedor.parentElement?.classList.add("marco-tabla-paginado"); tablasPaginadas.set(tabla, { navegacion, pagina: 1, tamano: tamanoPaginaMarco }); pintarPaginacionMarco(tabla, paginaInicialMarco(filas, tamanoPaginaMarco)); } function actualizarPaginacionesMarco() { document.querySelectorAll("#espacio-trabajo .tabla-contenedor").forEach(prepararTablaPaginable); } function aplicarFilasMarco(filas) { if (![20, 50, 100].includes(filas) || filas === tamanoPaginaMarco) return; tamanoPaginaMarco = filas; document.querySelectorAll("#espacio-trabajo .tabla-contenedor > table").forEach((tabla) => { const estadoTabla = tablasPaginadas.get(tabla); if (!estadoTabla) return; estadoTabla.tamano = filas; pintarPaginacionMarco(tabla, 1); }); actualizarPaginacionesMarco(); } function instalarPaginacionMarco() { const espacio = porId("espacio-trabajo"); if (!espacio) return; espacio.addEventListener("click", (evento) => { const control = evento.target.closest("[data-paginacion-marco-accion], [data-paginacion-marco-pagina]"); if (!control || control.disabled) return; const navegacion = control.closest(".paginacion-marco"); const tabla = navegacion?.previousElementSibling?.querySelector(":scope > table"); const estado = tabla && tablasPaginadas.get(tabla); if (!estado) return; const pagina = control.dataset.paginacionMarcoPagina ? Number(control.dataset.paginacionMarcoPagina) : control.dataset.paginacionMarcoAccion === "primera" ? 1 : estado.pagina + (control.dataset.paginacionMarcoAccion === "siguiente" ? 1 : -1); pintarPaginacionMarco(tabla, pagina); tabla.querySelector("tbody tr:not([hidden])")?.querySelector("button, a, [tabindex]")?.focus?.({ preventScroll: true }); }); new MutationObserver(actualizarPaginacionesMarco).observe(espacio, { childList: true, subtree: true }); actualizarPaginacionesMarco(); }
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
const recursosVistas = new Map();
const cargasRecursosVistas = new Map();
const erroresRecursosVistas = new Set();
const grupoRecursosVista = (vista) => vista === "portal" ? "inicio"
  : vista === "mis-tramites" ? "accesos"
    : vista === "contratos" && !vistaBolsaPendienteNoCompuesta(vista) ? "operaciones"
    : vista === VISTA_DOCUMENTOS_EXPEDIENTE ? "documentos"
  : vista === "auditoria" ? "auditoria" : vista === "llamamientos" ? "ofertas" : null;
function cargarRecursosVista(grupo) {
  if (cargasRecursosVistas.has(grupo)) return;
  const carga = grupo === "inicio"
    ? Promise.all([
      import("./portal-inicio.js?v=20261010-seguimiento-siguiente-v1"),
      import("./portal-accesos-empleado.js?v=20261001-g364-reconciliar-v2"),
    ]).then(([inicio, accesos]) => ({ ...inicio, accesos }))
    : grupo === "accesos"
      ? import("./portal-accesos-empleado.js?v=20261001-g364-reconciliar-v2")
      : grupo === "operaciones"
        ? import("./portal-vistas-operaciones.js?v=20260930-portales-i18n-integracion-v1")
      : grupo === "documentos"
    ? import("./modulos/documentos/i18n.js?v=20260928-ppt-v2")
    : grupo === "auditoria"
      ? import("./modulos/auditoria/vista.js?v=20261007-pantallas-textos-final-v1")
      : Promise.all([
        import("./portal-bolsas-ofertas.js?v=20261010-seguimiento-siguiente-v1"),
        import("./modulos/bolsa/rrhh-plazos-ui.js?v=20261007-pantallas-textos-final-v1"),
      ]).then(([ofertas, plazos]) => ({ ...ofertas, ...plazos }));
  cargasRecursosVistas.set(grupo, carga);
  carga.then((recursos) => {
    if (cargasRecursosVistas.get(grupo) !== carga) return;
    if (grupo === "inicio") {
      renderizarPortal = crearRenderizadorInicio(recursos.crearVistaInicioPortal);
      recursosVistas.set("accesos", recursos.accesos);
    }
    recursosVistas.set(grupo, recursos);
    if (grupoRecursosVista(estado.vista) === grupo && vistaPermitida(estado.vista)) renderizar();
  }).catch(() => {
    if (cargasRecursosVistas.get(grupo) !== carga) return;
    erroresRecursosVistas.add(grupo);
    if (grupoRecursosVista(estado.vista) === grupo) renderizar();
  }).finally(() => {
    if (cargasRecursosVistas.get(grupo) === carga) cargasRecursosVistas.delete(grupo);
  });
}
const TITULOS = Object.freeze({
  portal: [traducirPortal("menu_inicio"), traducirPortal("menu_inicio")],
  get "mis-tramites"() {
    const traducir = recursosVistas.get("accesos")?.traducirAccesosEmpleado;
    return [traducirPortal("txt_portal_del_empleado"), traducir ? traducir("mis_tramites") : traducirPortal("txt_accesos_directos")];
  },
  "ofertas-sae": [traducirPortal("ofertas_sae_miga"), traducirPortal("ofertas_sae_titulo")],
  resumen: [traducirPortal("txt_portal_del_empleado_bolsas_de_trabajo"), traducirPortal("txt_bolsas_de_trabajo")],
  elaboracion: [traducirPortal("txt_portal_del_empleado_bolsas_de_trabajo"), traducirPortal("txt_borradores_de_convocatorias")],
  convocatorias: [traducirPortal("txt_portal_del_empleado_bolsas_de_trabajo"), traducirPortal("txt_convocatorias_bases_y_calendario")],
  solicitudes: [traducirPortal("txt_portal_del_empleado_bolsas_de_trabajo"), traducirPortal("txt_solicitudes_y_admision")],
  meritos: [traducirPortal("txt_portal_del_empleado_bolsas_de_trabajo"), traducirPortal("txt_revision_de_meritos")],
  baremacion: [traducirPortal("txt_portal_del_empleado_bolsas_de_trabajo"), traducirPortal("txt_baremacion_y_ranking")],
  alegaciones: [traducirPortal("txt_portal_del_empleado_bolsas_de_trabajo"), traducirPortal("txt_alegaciones")],
  importacion: [traducirPortal("txt_portal_del_empleado_bolsas_de_trabajo"), traducirPortal("txt_importacion_convoca")],
  llamamientos: [traducirPortal("txt_portal_del_empleado_bolsas_de_trabajo_llamamient"), traducirPortal("txt_nuevo_llamamiento")],
  contratos: [traducirPortal("txt_portal_del_empleado_bolsas_de_trabajo"), traducirPortal("txt_contratos_ceses_y_reincorporaciones")],
  reglas: [traducirPortal("txt_portal_del_empleado_bolsas_de_trabajo"), traducirPortal("txt_reglas_y_versiones")],
  consulta: [traducirPortal("txt_portal_del_empleado_bolsas_de_trabajo"), traducirPortal("txt_consulta_segura_para_candidatos")],
  estadisticas: [traducirPortal("txt_portal_del_empleado_bolsas_de_trabajo"), traducirPortal("txt_estadisticas")],
  documentos: [traducirPortal("txt_portal_del_empleado_bolsas_de_trabajo"), traducirPortal("txt_documentos_y_firma")],
  comunicaciones: [traducirPortal("txt_portal_del_empleado_bolsas_de_trabajo"), traducirPortal("txt_correo_y_mensajeria")],
  auditoria: [traducirPortal("txt_portal_del_empleado_bolsas_de_trabajo"), traducirPortal("txt_auditoria_y_trazabilidad")],
  configuracion: [traducirPortal("txt_portal_del_empleado_bolsas_de_trabajo"), traducirPortal("txt_configuracion_y_roles")],
  cronos: [traducirPortal("cronos_miga"), traducirPortal("cronos_jornada_titulo")],
  "cronos-permisos": [traducirPortal("cronos_permisos_miga"), traducirPortal("cronos_permisos_titulo")],
  dietas: [traducirPortal("txt_portal_del_empleado_dietas"), traducirPortal("txt_dietas_y_comisiones_de_servicio")],
  personal: [traducirPortal("txt_portal_del_empleado_personal"), traducirPortal("txt_personal_consulta_informativa")],
  get "personal-registro"() {
    if (!MENSAJES_PERSONAL) return [traducirPortal("txt_portal_del_empleado_personal"), traducirPortal("txt_personal_consulta_informativa")];
    const traducir = crearTraductorPersonal();
    return [traducir("registro_b2_miga"), traducir("registro_b2_titulo")];
  },
  get [VISTA_DOCUMENTOS_EXPEDIENTE]() {
    const traducir = recursosVistas.get("documentos")?.crearTraductorDocumentos();
    return traducir ? [traducir("miga"), traducir("titulo")]
      : [traducirPortal("txt_documentos_y_firma"), traducirPortal("txt_documentos_y_firma")];
  },
  "bolsa-candidatos": [traducirPortal("txt_portal_del_empleado_bolsas_de_trabajo"), traducirPortal("txt_candidatos_de_la_bolsa")],
  "contratacion-temporal": [
    traducirPortal("contratacion_temporal_miga"),
    traducirPortal("contratacion_temporal_titulo"),
  ],
  [VISTA_PLANTILLAS_RRHH]: [traducirPortal("plantillas_rrhh_miga"), traducirPortal("plantillas_rrhh_titulo")],
  [VISTA_CATEGORIAS_RPT]: [traducirPortal("contratacion_temporal_miga"), traducirPortal("inicio_rrhh_categorias_rpt")],
  get "mis-preferencias"() {
    return textosGrupoPortalPreparados("preferencias")
      ? [traducirPortal("preferencias_miga"), traducirPortal("preferencias_titulo")]
      : [traducirPortal("txt_portal_del_empleado"), traducirPortal("preferencias_titulo_base")];
  },
});
const VISTAS_BOLSA_SIN_LECTURA = new Set([
  "contratos",
  "auditoria",
  "reglas",
]);
const requiereLecturaBolsas = (vista) => moduloDeVistaPortal(vista) === "bolsa"
  && !VISTAS_BOLSA_SIN_LECTURA.has(vista);
const estado = {
  vista: "portal",
  fuenteLista: false,
  errorFuente: "",
  plantillasAutorizadas: false,
  pasoLlamamiento: 1,
  llamamientoDesdeMenu: false,
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
  auditoriaReferencia: "",
  politicaCese: null,
  politicaCeseComprobada: false,
  inscripcionesDisponibles: null, // null sin comprobar; false si el servidor no publica la bandeja
  politicaCeseAusente: false, // 404: la instalación no compone la política de cese
  modalResultado: null,
};
let estadoResumenInicio = "sin_cargar";
let promesaResumenInicio = null;
function prepararResumenInicioVisible() {
  if (estado.vista !== "portal" || !esPerfilRRHH()) return Promise.resolve(null);
  if (typeof coordinadorModulos.prepararResumenInicio !== "function") {
    estadoResumenInicio = "error";
    renderizarConservandoFoco();
    return Promise.resolve(null);
  }
  if (promesaResumenInicio) return promesaResumenInicio;
  estadoResumenInicio = "cargando";
  promesaResumenInicio = coordinadorModulos.prepararResumenInicio().then((resultado) => {
    estadoResumenInicio = "listo";
    if (estado.vista === "portal") renderizarConservandoFoco();
    return resultado;
  }).catch(() => {
    estadoResumenInicio = "error";
    if (estado.vista === "portal") renderizarConservandoFoco();
    return null;
  }).finally(() => { promesaResumenInicio = null; });
  return promesaResumenInicio;
}
const cargasTextosVistas = new Map();
const erroresTextosVistas = new Set();
const gruposTextosMontables = new Set();
const cargasEstilosVistas = new Map();
const estilosVistasListos = new Set();
const erroresEstilosVistas = new Set();
function grupoEstilosDeVista(vista) {
  const modulo = moduloDeVistaPortal(vista);
  return ["cronos", "dietas", "personal"].includes(modulo) ? modulo : null;
}
function cargarEstilosVista(grupo) {
  if (cargasEstilosVistas.has(grupo) || estilosVistasListos.has(grupo) || erroresEstilosVistas.has(grupo)) return;
  const plantilla = document.querySelector(`template[data-estilos-vista="${grupo}"]`);
  if (!plantilla) { erroresEstilosVistas.add(grupo); return; }
  const enlaces = [...plantilla.content.querySelectorAll('link[rel="stylesheet"]')]
    .map((origen) => origen.cloneNode(true));
  const carga = Promise.all(enlaces.map((enlace) => new Promise((resolver, rechazar) => {
    enlace.addEventListener("load", resolver, { once: true });
    enlace.addEventListener("error", rechazar, { once: true });
    plantilla.before(enlace);
  })));
  cargasEstilosVistas.set(grupo, { carga, enlaces });
  carga.then(() => {
    estilosVistasListos.add(grupo);
    if (grupoEstilosDeVista(estado.vista) === grupo && vistaPermitida(estado.vista)) renderizar();
  }).catch(() => {
    enlaces.forEach((enlace) => enlace.remove());
    erroresEstilosVistas.add(grupo);
    if (grupoEstilosDeVista(estado.vista) === grupo) renderizar();
  }).finally(() => { cargasEstilosVistas.delete(grupo); });
}
function grupoTextosDeVista(vista) {
  if (vista === "mis-preferencias") return "preferencias";
  return moduloDeVistaPortal(vista) === "bolsa" ? "bolsa" : null;
}
function textosVistaPreparados(grupo) {
  return textosGrupoPortalPreparados(grupo)
    && (grupo !== "preferencias" || gruposTextosMontables.has(grupo));
}
async function prepararTextosBolsa() {
  const preparado = await prepararTextosPortal("bolsa");
  const contratos = await import("./portal-bolsas-contratos.js?v=20261007-pantallas-textos-final-v1");
  await contratos.prepararMensajesContratos(preparado.idioma);
  return preparado;
}
function vistaVigenteParaTextos(grupo, idioma) {
  return grupoTextosDeVista(estado.vista) === grupo
    && vistaPermitida(estado.vista)
    && (idioma === undefined || LOCALIZACION_PORTAL.split("-")[0] === idioma);
}
function esperarTextosDeVista(grupo) {
  if (cargasTextosVistas.has(grupo) || erroresTextosVistas.has(grupo)) return;
  const carga = grupo === "preferencias"
    ? integracionPreferencias.prepararTextosPreferencias()
    : grupo === "bolsa" ? prepararTextosBolsa() : prepararTextosPortal(grupo);
  cargasTextosVistas.set(grupo, carga);
  carga.then((preparado) => {
    if (cargasTextosVistas.get(grupo) !== carga) return;
    gruposTextosMontables.add(grupo);
    if (vistaVigenteParaTextos(grupo, preparado?.idioma)) renderizar();
  }).catch(() => {
    if (cargasTextosVistas.get(grupo) !== carga) return;
    erroresTextosVistas.add(grupo);
    if (vistaVigenteParaTextos(grupo)) renderizar();
  }).finally(() => {
    if (cargasTextosVistas.get(grupo) === carga) cargasTextosVistas.delete(grupo);
  });
}
function instalarReintentoTextosVista() {
  const espacio = porId("espacio-trabajo");
  espacio?.addEventListener("click", (evento) => {
    if (!evento.target?.closest?.("[data-bolsa-ruta-reintentar]") || estado.vista !== "bolsa-candidatos") return;
    void controladorBolsas?.cargarBolsas();
  });
  espacio?.addEventListener("click", (evento) => {
    if (!evento.target?.closest?.("[data-bolsa-inicio-reintentar]")) return;
    if (errorBolsaBase) window.location.reload();
    else pedirCuadroBolsas();
  });
  espacio?.addEventListener("click", (evento) => {
    if (evento.target?.closest?.("[data-bolsa-base-reintentar]")) window.location.reload();
  });
  espacio?.addEventListener("click", (evento) => {
    if (!evento.target?.closest?.("[data-recursos-vista-reintentar]")) return;
    // El navegador puede conservar un import() rechazado para esa URL durante
    // toda la página. Recargar conserva la ruta y vuelve a pedir el recurso.
    window.location.reload();
  });
  espacio?.addEventListener("click", (evento) => {
    const boton = evento.target?.closest?.("[data-estilos-vista-reintentar]");
    if (!boton || !espacio.contains(boton)) return;
    erroresEstilosVistas.delete(boton.dataset.estilosVistaReintentar);
    renderizar();
  });
  espacio?.addEventListener("click", (evento) => {
    const boton = evento.target?.closest?.("[data-textos-vista-reintentar]");
    if (!boton || !espacio.contains(boton)) return;
    erroresTextosVistas.delete(boton.dataset.textosVistaReintentar);
    renderizar();
  });
  espacio?.addEventListener("click", (evento) => {
    const boton = evento.target?.closest?.("[data-resumen-inicio-reintentar]");
    if (!boton || !espacio.contains(boton) || estado.vista !== "portal") return;
    estadoResumenInicio = "cargando";
    renderizar();
    void prepararResumenInicioVisible();
  });
}
let vistaAuditoriaBolsa = null;
let vistaPoliticaCese = null;
let vistaInscripcionesBolsa = null;
let controladorInscripcionesBolsa = null;
function desmontarInscripcionesBolsa() {
  controladorInscripcionesBolsa?.abort();
  controladorInscripcionesBolsa = null;
  vistaInscripcionesBolsa?.desmontar();
  vistaInscripcionesBolsa = null;
}
const clientePoliticaCese = crearClientePoliticaCeseRRHH();
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
  return new Intl.NumberFormat(LOCALIZACION_PORTAL, {
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
// La ficha de CT consulta Bolsa solo tras acreditar su detalle. Este contexto
// no alimenta el cuadro, los indicadores ni el controlador general de Bolsa.
let contextoBolsaCT = null;
async function prepararBolsaFichaCT({ expedienteRef, signal }) {
  if (typeof expedienteRef !== "string" || !expedienteRef || signal?.aborted) return;
  contextoBolsaCT = { expedienteRef, carga: "cargando", datos: null };
  if (estado.datosBolsas?.carga === "listo" && Array.isArray(estado.datosBolsas.datos?.bolsas)) {
    contextoBolsaCT = { expedienteRef, carga: "listo", datos: estado.datosBolsas.datos };
    return;
  }
  try {
    const { consultarBolsas } = await import("./portal-bolsas-api.js?v=20261010-seguimiento-siguiente-v1");
    if (signal?.aborted) return;
    const resultado = await consultarBolsas({ signal });
    if (signal?.aborted || contextoBolsaCT?.expedienteRef !== expedienteRef) return;
    contextoBolsaCT = resultado.ok && Array.isArray(resultado.datos?.bolsas)
      ? { expedienteRef, carga: "listo", datos: resultado.datos }
      : { expedienteRef, carga: resultado.status === 401 || resultado.status === 403 ? "denegado" : "error", datos: null };
  } catch {
    if (!signal?.aborted && contextoBolsaCT?.expedienteRef === expedienteRef)
      contextoBolsaCT = { expedienteRef, carga: "error", datos: null };
  }
}
const coordinadorModulos = crearCoordinadorModulosPortal({ escaparHTML, anunciar,
  consultarSesion: () => obtenerSesion(),
  montajeBolsa: Object.freeze({
    disponible: (vista) => VISTAS_INTERNAS_BOLSA.includes(vista) && !vista.startsWith("seleccion-"),
    prepararFicha: prepararBolsaFichaCT,
    limpiarFicha: () => { contextoBolsaCT = null; },
    fallarFicha: (expedienteRef) => {
      if (contextoBolsaCT?.expedienteRef === expedienteRef)
        contextoBolsaCT = { expedienteRef, carga: "error", datos: null };
    },
    // Otros módulos enlazan con una bolsa solo si el perfil la ve en el cuadro.
    // Sin referencia, «categoriaRef» busca la bolsa vigente de esa categoría de la RPT.
    resolverBolsa: (bolsaRef, { categoriaRef = "" } = {}) => {
      if (estado.vista === "contratacion-temporal" && ["error", "denegado"].includes(contextoBolsaCT?.carga))
        return Object.freeze({ estado: contextoBolsaCT.carga });
      if (!vistaPermitida(VISTA_CANDIDATOS_BOLSA)) return null;
      const bolsas = estado.datosBolsas?.carga === "listo" ? estado.datosBolsas.datos?.bolsas
        : estado.vista === "contratacion-temporal" && contextoBolsaCT?.carga === "listo"
          ? contextoBolsaCT.datos?.bolsas : null;
      if (!Array.isArray(bolsas)) return null;
      const categoria = typeof categoriaRef === "string" ? categoriaRef.replace(/^categoria:rpt:/u, "") : "";
      const bolsa = bolsaRef ? bolsas.find((item) => item?.bolsa_ref === bolsaRef)
        : categoria ? bolsas.find((item) => item?.vigente_hasta === null && item?.categoria_clave === categoria) : null;
      if (!bolsa && !bolsaRef && categoria) return Object.freeze({ estado: "sin_bolsa" });
      return bolsa && typeof bolsa.categoria === "string"
        ? Object.freeze({ categoria: bolsa.categoria, ...(bolsaRef ? {} : { bolsa_ref: bolsa.bolsa_ref }) }) : null;
    },
    montar: ({ vista, raiz, opciones }) => {
      let desmontarRegistrado = null;
      const registrarDesmontar = (limpiar) => { desmontarRegistrado = limpiar; };
      montarVistaBolsa(vista, raiz, opciones);
      return Object.freeze({ desmontar: () => { if (vista === "elaboracion") superficieBorradoresActiva()?.desmontar();
        controladorBolsas?.cancelarPeticiones(); cancelarAvisosBolsa(); if (vista === "llamamientos") { superficieOfertasBolsa?.desmontar(); superficieRRHHPlazos?.desmontar(); }
        if (vista === "auditoria") { vistaAuditoriaBolsa?.desmontar(); vistaAuditoriaBolsa = null; }
        if (vista === "reglas") { vistaPoliticaCese?.desmontar(); vistaPoliticaCese = null; }
        if (vista === "solicitudes") desmontarInscripcionesBolsa(); } });
    },
  }),
  confirmarOperacion: (descriptor) => window.confirm(traducirPortal("txt_confirmar_operacion", { titulo: descriptor.titulo, advertencia: descriptor.advertencia, referencia: descriptor.referencia })) });
let renderizarPortal = null;
function crearRenderizadorInicio(crearVistaInicioPortal) { return crearVistaInicioPortal({
  encabezadoVista,
  escaparHTML,
  numero,
  obtenerCatalogo: coordinadorModulos.obtenerCatalogo,
  resolverAcceso: resolverAccesoPerfil,
  esPerfilRRHH,
  obtenerCuadroInicio: () => coordinadorModulos.obtenerCuadroInicio?.() || null,
  obtenerBolsasInicio: () => estado.datosBolsas,
  obtenerAccesosEmpleado: coordinadorModulos.obtenerAccesosEmpleado,
  locale: LOCALIZACION_PORTAL,
  catalogoFallido: () => estado.errorFuente !== "",
  inicioPendiente: () => coordinadorModulos.inicioPendiente?.() === true,
}); }
function renderizarAyudaPreparada(contexto = null, ayudaContenido, ayudanteTramites) {
  const { AYUDA_PORTAL_RRHH, detectarContextoContratacionTemporal,
    obtenerAyudaContratacionTemporal, renderizarAyudaContratacionTemporal,
    TRAMITES_AYUDANTE_PORTAL } = ayudaContenido;
  const { crearAyudanteTramites } = ayudanteTramites;
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
    const ayuda = AYUDA_PORTAL_RRHH;
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
function renderizarContenidoAyuda(contexto = null) {
  const vistaAlAbrir = estado.vista;
  const cargando = () => `<section role="status" aria-busy="true" class="panel"><div class="cuerpo-panel">${textoPortal("estado_modulo_comprobando")}</div></section>`;
  return {
    titulo: traducirPortal("txt_ayuda_pantalla"),
    contenido: cargando(),
    instalar: (dialogo) => {
      const { contenedor } = dialogo;
      let activo = true;
      let turno = 0;
      let limpiarAyuda = null;
      async function cargar() {
        const actual = ++turno;
        limpiarAyuda?.(); limpiarAyuda = null;
        contenedor.innerHTML = cargando();
        try {
          await prepararTextosPortal("ayuda");
          const [ayudaContenido, ayudanteTramites] = await Promise.all([
            import("./ayuda-contenido.js?v=20261008-alta-etiquetas-ayuda-v2"),
            import("./ayudante-tramites.js?v=20261008-alta-etiquetas-ayuda-v2"),
          ]);
          if (!activo || actual !== turno || estado.vista !== vistaAlAbrir) return;
          const ayuda = renderizarAyudaPreparada(contexto, ayudaContenido, ayudanteTramites);
          porId("titulo-dialogo").textContent = ayuda.titulo || traducirPortal("txt_ayuda_pantalla");
          contenedor.innerHTML = ayuda.contenido;
          limpiarAyuda = ayuda.instalar?.(dialogo) || null;
          document.querySelectorAll('[data-accion="ayuda"]').forEach((control) => {
            control.setAttribute("aria-label", traducirPortal("ayuda_abrir_contextual", { contexto: tituloDeVista(estado.vista)[1] }));
          });
        } catch {
          if (!activo || actual !== turno || estado.vista !== vistaAlAbrir) return;
          contenedor.innerHTML = `<section class="panel" role="alert"><div class="cuerpo-panel"><p>${textoPortal("estado_modulo_no_disponible_titulo")}</p><button type="button" class="boton-secundario" data-ayuda-reintentar>${textoPortal("accion_reintentar")}</button></div></section>`;
          contenedor.querySelector("[data-ayuda-reintentar]")?.focus({ preventScroll: true });
        }
      }
      const reintentar = (evento) => {
        if (evento.target?.closest?.("[data-ayuda-reintentar]")) void cargar();
      };
      contenedor.addEventListener("click", reintentar);
      void cargar();
      return () => { activo = false; turno += 1; limpiarAyuda?.(); contenedor.removeEventListener("click", reintentar); };
    },
  };
}
function porcentajeSeguro(valor) {
  const numeroValor = Number(valor);
  if (!Number.isFinite(numeroValor)) return 0;
  return Math.max(0, Math.min(100, Math.round(numeroValor * 10) / 10));
}
// Mientras se lee el cuadro, Bolsa se ofrece ya si el catálogo la autoriza
// (abrirla muestra el cuadro «cargando»), y tras un fallo transitorio también
// (dentro se ve el error con «Reintentar»); si la API la deniega, deja de
// ofrecerse. Solo es presentación: el servidor autoriza cada consulta.
function disponibilidadBolsa() {
  const acceso = accesoBolsaEfectivo(superficieBorradores.obtenerAcceso(), estado.datosBolsas);
  if (estado.datosBolsas === null && acceso?.disponible !== true
    && coordinadorModulos.obtenerCatalogo().some((modulo) => modulo.clave === "bolsa")) {
    return { disponible: true, vista: "resumen", estado: "cargando", etiqueta: traducirPortal("txt_cuadro_de_bolsas") };
  }
  if (acceso?.disponible !== true && ["cargando", "error"].includes(estado.datosBolsas?.carga) && estado.vista !== "elaboracion") {
    return { disponible: true, vista: "resumen", estado: "disponible", etiqueta: traducirPortal("txt_cuadro_de_bolsas") };
  }
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
    bolsasConsultables: estado.datosBolsas?.carga === "listo",
    borradores: borradores?.disponible === true ? true
      : (borradores?.estado === "denegado" || borradores?.estado === "error" ? false : null),
    contratacionTemporal: coordinadorModulos.vistaDisponible("contratacion-temporal"),
    auditoriaReferencia: estado.auditoriaReferencia !== "",
    politicaCese: estado.politicaCese !== null,
    inscripciones: estado.inscripcionesDisponibles === true,
  };
}
function vistaPermitida(vista) {
  // Una ruta conocida sin consumidor sólo muestra su estado; no abre ninguna operación.
  if (vistaBolsaPendienteNoCompuesta(vista)) return true;
  if (vista === "mis-preferencias") return true;
  if (vista === VISTA_PLANTILLAS_RRHH) return estado.plantillasAutorizadas === true;
  if (vista.startsWith("seleccion-")) return false;
  // Con Bolsa en el catálogo, Reglas se abre para decir «no disponible» (404).
  if (vista === "reglas" && estado.politicaCeseAusente) return coordinadorModulos.obtenerCatalogo().some((m) => m.clave === "bolsa");
  if (moduloDeVistaPortal(vista) === "bolsa") return vistaBolsaNavegable(vista, capacidadesBolsa());
  if (VISTAS_MODULOS_PERSONALES.has(vista) || VISTAS_AUTOSERVICIO_EMPLEADO.has(vista)) {
    return !estado.fuenteLista || coordinadorModulos.vistaDisponible(vista)
      || (VISTAS_AUTOSERVICIO_EMPLEADO.has(vista) && coordinadorModulos.vistaPendiente(vista));
  }
  return true;
}

function etiquetaFuentePanel() {
  return presentadorPanelInterno?.etiquetaFuente() || traducirPortal("txt_api_interna_autorizada");
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
  // Sin iniciales se conserva el icono de persona de la cabecera.
  if (datos.iniciales && datos.iniciales !== "—") integracionPreferencias.fijarIniciales(datos.iniciales);
  sesion.querySelector("strong").textContent = datos.nombre;
  const perfil = sesion.querySelector("small");
  perfil.textContent = datos.perfil;
  perfil.hidden = datos.perfil === "";
  sesion.querySelector("[data-sesion-texto]").hidden = false;
  const boton = porId("boton-identidad");
  boton.disabled = false;
  boton.setAttribute("aria-label", datos.perfil ? `${datos.nombre}, ${datos.perfil}` : datos.nombre);
}
// Repinta en cuanto llega el catálogo y cada vez que termina un módulo: Inicio
// muestra las tarjetas con «Comprobando» y cada una se actualiza sola. Una vista
// de módulo pedida por el enlace se monta en cuanto está disponible, una sola
// vez; si estaba «Comprobando» y su módulo termina sin estarlo, se repinta.
let inicioComprobando = false;
// La API de borradores de convocatorias NO se sondea al cargar: un servidor que
// no la sirve respondería 404 en cada carga. Elaboración solo se ofrece en el
// menú cuando consta disponible; al abrirla por su enlace se comprueba entonces.
// Ciclo de carga (secuenciaFuente) en que se pidió el cuadro de bolsas por
// última vez. Una lectura pedida en el ciclo actual ya revalida el acceso.
let cicloLecturaBolsas = 0;
let generacionCuadroBolsas = 0;
function pedirCuadroBolsas() {
  const pedido = ++generacionCuadroBolsas;
  cicloLecturaBolsas = secuenciaFuente;
  estado.datosBolsas = { carga: "cargando", datos: null, error: "" };
  void prepararBolsaBase().then(() => {
    if (pedido === generacionCuadroBolsas && vistaNecesitaBolsa()
      && cicloLecturaBolsas === secuenciaFuente) void controladorBolsas.cargarBolsas();
  }).catch(() => {
    if (pedido !== generacionCuadroBolsas || !vistaNecesitaBolsa()) return;
    estado.datosBolsas = { carga: "error", datos: null, error: traducirPortal("txt_no_se_pudieron_cargar_las_bolsas_de_trabajo") };
    renderizar();
  });
}
function alCambiarModulos(clave) {
  // La lectura anterior no habilita a seguir mostrando Bolsa tras un cambio
  // del catálogo o de identidad: la API debe revalidar el acceso actual. Si
  // la lectura de este mismo ciclo sigue en curso o ya terminó, no se repite:
  // relanzarla cancelaba la petición y obligaba al servidor a empezar de cero.
  if (clave === "catalogo" && vistaNecesitaBolsa()
    && !(cicloLecturaBolsas === secuenciaFuente && ["cargando", "listo"].includes(estado.datosBolsas?.carga))) {
    pedirCuadroBolsas();
  }
  if (clave === "contratacion_temporal" && coordinadorModulos.vistaDisponible("contratacion-temporal")
    && (destinoPlantillasInicial || plantillasConfirmadas)) {
    void comprobarAccesoPlantillas();
  }
  // Una vista de un módulo sin entrada (URL directa) arranca su carga diferida
  // en cuanto el catálogo la autoriza.
  if (clave === "catalogo") coordinadorModulos.prepararVista(estado.vista);
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
  const pestañaSeleccionada = contenedor?.querySelector(".portal-rrhh-tab-radio:checked")?.id;
  const pestañaEnfocada = activo?.matches?.(".portal-rrhh-tab-radio") ? activo.id : "";
  renderizar();
  const radios = [...(contenedor?.querySelectorAll(".portal-rrhh-tab-radio") ?? [])];
  const seleccionada = radios.find((radio) => radio.id === pestañaSeleccionada);
  if (seleccionada) seleccionada.checked = true;
  const focoRadio = radios.find((radio) => radio.id === pestañaEnfocada);
  if (focoRadio) { focoRadio.focus({ preventScroll: true }); return; }
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
  generacionCuadroBolsas += 1;
  controladorBolsas?.cancelarPeticiones();
  estado.datosBolsas = null;
  contextoBolsaCT = null;
  cicloLecturaBolsas = 0;
  estado.errorFuente = "";
  consultaAccesoPlantillas?.abort();
  consultaAccesoPlantillas = null;
  estado.plantillasAutorizadas = false;
  consultaAccesoPoliticaCese?.abort();
  consultaAccesoPoliticaCese = null;
  estado.politicaCese = null;
  estado.politicaCeseComprobada = false;
  estado.politicaCeseAusente = false;
  if (destinoPoliticaCeseInicial) void comprobarAccesoPoliticaCese();
  // Una vista de Bolsa ya montada no depende del catálogo de módulos: se
  // conserva (el coordinador tampoco la retira) y no se vuelve a montar.
  if (moduloDeVistaPortal(estado.vistaMontada || "") !== "bolsa") estado.vistaMontada = "";
  estado.vistaCerrada = "";
  if (estado.vista === "portal") estadoResumenInicio = "cargando";
  // La disponibilidad de Bolsa en Inicio y en el menú la decide la API real del
  // cuadro de bolsas: se consulta en paralelo con el catálogo, sin esperarla ni
  // bloquear los demás módulos. Una lectura ya en curso no se repite.
  if (vistaNecesitaBolsa() && estado.datosBolsas?.carga !== "cargando"
    && (requiereLecturaBolsas(estado.vista) || estado.datosBolsas?.carga !== "listo")) pedirCuadroBolsas();
  await coordinadorModulos.cargarInterno({ alCambiar: alCambiarModulos }).catch((error) => {
    // Una carga sustituida por otra más reciente no es un fallo del catálogo.
    if (error?.codigo === CODIGO_CARGA_SUSTITUIDA || intento !== secuenciaFuente) return;
    estado.errorFuente = traducirPortal("error_catalogo_modulos");
  });
  if (intento !== secuenciaFuente) return;
  if (estado.vista === "portal" && esPerfilRRHH()) await prepararResumenInicioVisible();
  if (intento !== secuenciaFuente) return;
  actualizarNavegacionModulos();
  renderizarTrasCarga();
  if (politicaCeseConfirmada && coordinadorModulos.obtenerCatalogo().some((modulo) => modulo.clave === "bolsa")) {
    void comprobarAccesoPoliticaCese();
  }
}

// Plantillas de documentos (CT) y política de cese (Bolsa) son capacidades
// opcionales: la composición de la principal puede no montarlas y entonces su
// API responde 404. Igual que los borradores de convocatorias, NO se sondean al
// cargar el portal: solo cuando la persona abre su vista (enlace directo o
// navegación) o cuando ya constaron disponibles en esta sesión y hay que
// revalidarlas tras recargar el catálogo o la identidad.
let consultaAccesoPlantillas = null;
let destinoPlantillasInicial = false;
let plantillasConfirmadas = false;
let consultaAccesoPoliticaCese = null;
let destinoPoliticaCeseInicial = false;
let politicaCeseConfirmada = false;

async function comprobarAccesoPoliticaCese() {
  if (estado.politicaCeseComprobada || consultaAccesoPoliticaCese) return;
  const controlador = new AbortController();
  consultaAccesoPoliticaCese = controlador;
  try {
    const politica = await clientePoliticaCese.consultar({ signal: controlador.signal });
    if (consultaAccesoPoliticaCese !== controlador || controlador.signal.aborted) return;
    estado.politicaCese = politica;
    estado.politicaCeseComprobada = true;
    politicaCeseConfirmada = true;
    if (destinoPoliticaCeseInicial && estado.vista === "portal") {
      destinoPoliticaCeseInicial = false;
      navegar("reglas");
    }
  } catch (error) {
    if (consultaAccesoPoliticaCese !== controlador || controlador.signal.aborted) return;
    estado.politicaCese = null;
    estado.politicaCeseComprobada = true;
    estado.politicaCeseAusente = error?.estado === 404;
    if (destinoPoliticaCeseInicial) {
      destinoPoliticaCeseInicial = false;
      // Sin la API (404) la vista lo dice; un 403 sigue en Inicio, sin revelarla.
      if (estado.politicaCeseAusente && estado.vista === "portal") navegar("reglas");
      else history.replaceState(null, "", rutaDeVista("portal"));
    }
  } finally {
    if (consultaAccesoPoliticaCese === controlador) {
      consultaAccesoPoliticaCese = null;
      actualizarNavegacionModulos();
    }
  }
}
// Abrir una vista opcional aún sin comprobar lanza su sondeo y espera en Inicio;
// al confirmarse la disponibilidad se navega a ella y, si no, se queda en Inicio.
function sondearCapacidadAlAbrir(vista) {
  const plantillas = vista === VISTA_PLANTILLAS_RRHH && estado.plantillasAutorizadas !== true
    && consultaAccesoPlantillas === null && coordinadorModulos.vistaDisponible("contratacion-temporal");
  const politica = vista === "reglas" && !estado.politicaCeseComprobada;
  if (!plantillas && !politica) return false;
  navegar("portal");
  if (plantillas) { destinoPlantillasInicial = true; void comprobarAccesoPlantillas(); }
  else { destinoPoliticaCeseInicial = true; void comprobarAccesoPoliticaCese(); }
  return true;
}
async function comprobarAccesoPlantillas() {
  consultaAccesoPlantillas?.abort();
  const controlador = new AbortController();
  consultaAccesoPlantillas = controlador;
  try {
    const { crearClientePlantillasRRHH } = await import("./modulos/contratacion-temporal/rrhh-plantillas-cliente.js?v=20261008-alta-rpt-circular-v6");
    if (consultaAccesoPlantillas !== controlador || controlador.signal.aborted) return;
    await crearClientePlantillasRRHH().consultar({ signal: controlador.signal });
    if (consultaAccesoPlantillas !== controlador || controlador.signal.aborted) return;
    estado.plantillasAutorizadas = true;
    plantillasConfirmadas = true;
    if (destinoPlantillasInicial && estado.vista === "portal") {
      destinoPlantillasInicial = false;
      navegar(VISTA_PLANTILLAS_RRHH);
    }
  } catch {
    if (consultaAccesoPlantillas !== controlador || controlador.signal.aborted) return;
    estado.plantillasAutorizadas = false;
    if (destinoPlantillasInicial) {
      destinoPlantillasInicial = false;
      history.replaceState(null, "", rutaDeVista("portal"));
    }
  } finally {
    if (consultaAccesoPlantillas === controlador) {
      consultaAccesoPlantillas = null;
      actualizarNavegacionModulos();
    }
  }
}

function necesidadLlamamientoSeleccionada() {
  return DATOS_PANEL.necesidades_llamamiento.find((item) => item.id === estado.necesidadSeleccionada)
    || DATOS_PANEL.necesidades_llamamiento[0];
}

function puedeSolicitarPropuesta() { return DATOS_PANEL.capacidades.solicitar_propuesta_llamamiento === true; }

async function solicitarPropuestaLlamamiento() {
  const necesidad = necesidadLlamamientoSeleccionada();
  if (!necesidad) return { ok: false, mensaje: traducirPortal("txt_seleccione_una_necesidad_de_cobertura") };
  if (estado.solicitandoPropuesta) return { ok: false, mensaje: traducirPortal("txt_la_solicitud_ya_esta_en_curso") };

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
    estado.errorPropuesta = error instanceof Error ? error.message : traducirPortal("txt_no_se_pudo_obtener_la_propuesta");
    return { ok: false, mensaje: estado.errorPropuesta };
  } finally {
    estado.solicitandoPropuesta = false;
    renderizar();
  }
}

// Enlace desde otra página del portal (una petición del centro ya entregada):
// «?expediente=<referencia>#contratacion-temporal» abre ese expediente. La
// referencia solo navega; el servidor decide si el perfil puede consultarlo.
function opcionesDesdeEnlace(vista) {
  const filtroServidorRuta = vista === "contratacion-temporal" ? leerFiltroCTDeRuta(window.location.search) : null;
  const fichaRuta = vista === "contratacion-temporal" ? leerFichaCTDeRuta(window.location.search) : null;
  const opcionesCT = vista === "contratacion-temporal"
    ? { alCambiarFiltroLista: alCambiarFiltroListaCT, alCambiarFicha: alCambiarFichaCT,
      ...(fichaRuta ? { expedienteRef: fichaRuta.expedienteRef,
        ...(fichaRuta.version !== null ? { expedienteVersion: fichaRuta.version } : {}) } : {}),
      ...(filtroServidorRuta ? { filtroServidorRuta } : {}) } : null;
  return opcionesCT;
}

function vistaDesdeHash() {
  const valor = window.location.hash.replace(/^#\/?/, "").trim();
  if (!valor || valor === "portal") {
    const ruta = rutaPortalConFiltroCT(window.location, "#portal");
    if (`${window.location.pathname}${window.location.search}${window.location.hash}` !== ruta)
      history.replaceState(null, "", ruta);
    return "portal";
  } const candidata = valor.split("/").filter(Boolean).at(-1);
  if (Object.hasOwn(TITULOS, candidata)) return candidata;
  history.replaceState(null, "", "#portal");
  return "portal";
}

function rutaDeVista(vista) { return vista === "mis-preferencias" ? "#mis-preferencias" : rutaDeVistaPortal(vista); }
function tituloDeVista(vista) { return TITULOS[vista] || TITULOS.portal; }
function moduloActivoDeVista(vista) { return moduloDeVistaPortal(vista); }
function actualizarNavegacionModulos() {
  const contenedor = porId("navegacion-modulos-dinamica");
  if (!contenedor) return;
  const moduloActivo = moduloActivoDeVista(estado.vista);
  const disponibilidad = disponibilidadBolsa();
  contenedor.innerHTML = coordinadorModulos.renderizarNavegacion(disponibilidad, moduloActivo, vistaPermitida);
  const enlacePlantillas = porId("enlace-plantillas-rrhh");
  if (enlacePlantillas) enlacePlantillas.hidden = estado.plantillasAutorizadas !== true;
  aplicarDisponibilidadMenuBolsa(porId("navegacion-bolsa"), capacidadesBolsa())
    .forEach((indicador, indice) => { indicador.textContent = String(indice + 1); });
  const fase = porId("texto-estado-modulos-portal");
  if (fase) {
    // El recuento corresponde a las entradas de módulo que muestra el menú.
    const accesos = [...contenedor.querySelectorAll("[data-modulo-portal]")].map((boton) => ({
      disponible: boton.classList.contains("modulo-habilitado"),
      estado: boton.getAttribute("aria-busy") === "true" ? "cargando" : "",
    }));
    const accesosPropios = coordinadorModulos.obtenerAccesosEmpleado();
    const crearResumen = recursosVistas.get("accesos")?.crearTraductorResumenAccesosEmpleado;
    const bolsaPendiente = disponibilidadBolsa()?.estado === "cargando";
    fase.textContent = estado.errorFuente || (crearResumen
      ? resumenAccesosModulos(accesos, bolsaPendiente, crearResumen({ accesos: accesosPropios, traducir: traducirPortal }))
      : Object.keys(accesosPropios).length > 0 && !accesos.some((acceso) => acceso.disponible === true)
        ? traducirPortal("txt_accesos_directos") : resumenAccesosModulos(accesos, bolsaPendiente));
  }
}

function anunciar(mensaje) {
  const region = porId("anuncios");
  if (!region) return;
  region.textContent = "";
  window.setTimeout(() => { region.textContent = mensaje; }, 20);
}
function alCambiarFiltroListaCT(filtroAplicado) {
  if (estado.vista !== "contratacion-temporal" || !filtroServidorCTValido(filtroAplicado)) return;
  const rutaFiltrada = rutaPortalConFiltroCT(window.location, rutaDeVista("contratacion-temporal"), filtroAplicado);
  const ruta = rutaConFichaCT(new URL(rutaFiltrada, window.location.href));
  if (`${window.location.pathname}${window.location.search}${window.location.hash}` !== ruta)
    history.replaceState(null, "", ruta);
}
function alCambiarFichaCT(ficha) {
  if (estado.vista !== "contratacion-temporal") return;
  const ruta = rutaConFichaCT(window.location, ficha);
  if (`${window.location.pathname}${window.location.search}${window.location.hash}` !== ruta)
    history.pushState(null, "", ruta);
}
function navegar(vista, opciones = {}) {
  if (!Object.hasOwn(TITULOS, vista)) return;
  if (vista !== VISTA_PLANTILLAS_RRHH) destinoPlantillasInicial = false;
  if (vista !== "reglas") destinoPoliticaCeseInicial = false;
  if (sondearCapacidadAlAbrir(vista)) return;
  if (!vistaPermitida(vista) && vistaBolsaPendienteNoCompuesta(vista)) {
    const hash = rutaDeVista(vista);
    const ruta = rutaPortalConFiltroCT(window.location, hash);
    if (`${window.location.pathname}${window.location.search}${window.location.hash}` !== ruta)
      history.pushState(null, "", ruta);
    estado.vista = vista;
    renderizar();
    cerrarMenuMovil();
    if (opciones.enfocar !== false) porId("contenido-principal")?.focus({ preventScroll: true });
    return;
  }
  if (!vistaPermitida(vista)) {
    const vistaSegura = "portal";
    const hashSeguro = rutaDeVista(vistaSegura);
    const rutaSegura = rutaPortalConFiltroCT(window.location, hashSeguro);
    if (`${window.location.pathname}${window.location.search}${window.location.hash}` !== rutaSegura)
      history.replaceState(null, "", rutaSegura);
    estado.vista = vistaSegura;
    renderizar();
    cerrarMenuMovil();
    if (opciones.enfocar !== false) porId("contenido-principal")?.focus({ preventScroll: true });
    anunciar(traducirPortal("txt_la_vista_solicitada_no_esta_autorizada_para_el_p"));
    return;
  }
  if (vista === "llamamientos" && estado.vista !== vista) {
    estado.llamamientoDesdeMenu = false;
    estado.llamamientoDesdeCT = false;
    estado.origenLlamamientoB7 = null;
  }
  const hash = rutaDeVista(vista);
  if (vistaNecesitaBolsa(estado.vista) && !vistaNecesitaBolsa(vista)) {
    generacionCuadroBolsas += 1;
    controladorBolsas?.cancelarPeticiones();
    estado.datosBolsas = null;
    cicloLecturaBolsas = 0;
  }
  if (estado.vista === "contratacion-temporal" && vista !== "contratacion-temporal") contextoBolsaCT = null;
  if (vista !== "auditoria") estado.auditoriaReferencia = "";
  if (vista !== "bolsa-candidatos") rutaCandidatosAplicada = null;
  const filtroServidorRuta = vista === "contratacion-temporal"
    ? (opciones.desdeRuta === true ? leerFiltroCTDeRuta(window.location.search)
      : opciones.filtroServidorRuta ?? (opciones.filtroLista?.mostrar === "incidencia"
        && !opciones.filtroLista?.fase ? FILTRO_INCIDENCIA_CT : null))
    : null;
  if (vista === "resumen" && rutasBolsa) {
    const ruta = rutasBolsa.rutaResumenBolsasCompartible(limpiarFiltroCTDeBusqueda(window.location.search));
    if (`${window.location.search}${window.location.hash}` !== ruta) history.pushState(null, "", ruta);
  } else if (opciones.desdeRuta !== true) {
    const ruta = rutaPortalConFiltroCT(window.location, hash,
      vista === "contratacion-temporal" && filtroServidorCTValido(filtroServidorRuta) ? filtroServidorRuta : null);
    if (`${window.location.pathname}${window.location.search}${window.location.hash}` !== ruta)
      history.pushState(null, "", ruta);
  } else if (vista !== "contratacion-temporal") {
    const ruta = rutaPortalConFiltroCT(window.location, hash, null,
      { conservarLlamamiento: vista === "llamamientos" && opciones.desdeRuta === true });
    if (`${window.location.pathname}${window.location.search}${window.location.hash}` !== ruta)
      history.replaceState(null, "", ruta);
  }
  estado.vista = vista;
  if (vista === "llamamientos" && estado.datosBolsas?.carga === "listo") selectorLlamamientos?.consumirRuta();
  const fichaRuta = vista === "contratacion-temporal" && opciones.desdeRuta === true
    ? leerFichaCTDeRuta(window.location.search) : null;
  estado.opcionesVista = vista === "contratacion-temporal"
    ? { ...opciones, ...(fichaRuta ? { expedienteRef: fichaRuta.expedienteRef,
      ...(fichaRuta.version !== null ? { expedienteVersion: fichaRuta.version } : {}) } : {}),
      filtroServidorRuta, alCambiarFiltroLista: alCambiarFiltroListaCT, alCambiarFicha: alCambiarFichaCT } : opciones;
  if (vista === "portal" && esPerfilRRHH() && estadoResumenInicio !== "listo") {
    estadoResumenInicio = "cargando";
    void prepararResumenInicioVisible();
  }
  renderizar();
  if (vista === "mis-preferencias" && superficiePreferencias.leerCarga() === "sin_cargar") void superficiePreferencias.cargar();
  if (vistaNecesitaBolsa(vista) && estado.datosBolsas === null) pedirCuadroBolsas();
  cerrarMenuMovil();
  if (opciones.enfocar !== false) porId("contenido-principal")?.focus({ preventScroll: true });
  anunciar(traducirPortal("txt_vista_abierta", { vista: tituloDeVista(vista)[1] }));
}
function montarVistaBolsa(vista, contenedor, opciones = {}, { activar = true } = {}) {
  if (vista === "solicitudes") {
    if (controladorInscripcionesBolsa && contenedor.querySelector("[data-inscripciones-montaje]")) return;
    desmontarInscripcionesBolsa();
    const controlador = new AbortController();
    controladorInscripcionesBolsa = controlador;
    contenedor.innerHTML = `<div data-inscripciones-montaje><section class="panel" role="status" aria-busy="true"><div class="cuerpo-panel"><p>${textoPortal("estado_modulo_comprobando")}</p></div></section></div>`;
    const raiz = contenedor.querySelector("[data-inscripciones-montaje]");
    import("./modulos/bolsa/inscripcion-rrhh-vista.js?v=20261009-inscripciones-rrhh-v1").then(async ({ montarInscripcionesRRHH }) => {
      if (controlador.signal.aborted || estado.vista !== vista) return;
      const montaje = await montarInscripcionesRRHH({ raiz, signal: controlador.signal,
        alDenegacion: () => { estado.solicitudes = []; },
        alDisponibilidad: (disponible) => {
          if (estado.inscripcionesDisponibles === disponible) return;
          estado.inscripcionesDisponibles = disponible; actualizarNavegacionModulos();
        } });
      if (controlador.signal.aborted || estado.vista !== vista) { montaje?.desmontar(); return; }
      vistaInscripcionesBolsa = montaje;
    }).catch((error) => {
      if (controlador.signal.aborted || estado.vista !== vista) return;
      console.error("inscripciones_vista_carga", { tipo: error?.name });
      raiz.innerHTML = `<section class="panel" role="alert"><div class="cuerpo-panel"><p>${textoPortal("estado_modulo_no_disponible_titulo")}</p><button type="button" class="boton-secundario" data-inscripciones-montaje-reintentar>${textoPortal("accion_reintentar")}</button></div></section>`;
      raiz.querySelector("[data-inscripciones-montaje-reintentar]")?.addEventListener("click", () => {
        desmontarInscripcionesBolsa();
        montarVistaBolsa(vista, contenedor);
      }, { once: true });
    });
    return;
  }
  if (vistaBolsaPendienteNoCompuesta(vista)) {
    contenedor.innerHTML = renderizarFuenteNoDisponible();
    return;
  }
  if (vista === "bolsa-candidatos" && estado.datosBolsas?.carga !== "listo") {
    const carga = estado.datosBolsas?.carga;
    const error = carga === "error" || carga === "denegado";
    contenedor.innerHTML = `<section class="panel" ${error ? 'role="alert"' : 'role="status" aria-busy="true"'}><div class="cuerpo-panel"><p>${textoPortal(carga === "denegado" ? "txt_la_sesion_actual_no_dispone_de_permisos_suficien" : error ? "txt_no_se_pudieron_cargar_las_bolsas_de_trabajo" : "estado_modulo_comprobando")}</p>${carga === "error" ? `<button type="button" class="boton-secundario" data-bolsa-ruta-reintentar>${textoPortal("accion_reintentar")}</button>` : ""}</div></section>`;
    return;
  }
  if (vista === "reglas") {
    if (vistaPoliticaCese && contenedor.querySelector("[data-politica-cese]")) return;
    vistaPoliticaCese?.desmontar();
    vistaPoliticaCese = montarVistaPoliticaCeseRRHH({ raiz: contenedor,
      politica: estado.politicaCese, noDisponible: estado.politicaCeseAusente, cliente: clientePoliticaCese, anunciar,
      alDenegacion: () => { estado.politicaCese = null; actualizarNavegacionModulos(); } });
    return;
  }
  if (vista === "auditoria") {
    if (vistaAuditoriaBolsa && contenedor.querySelector("[data-auditoria-vista]")) return;
    vistaAuditoriaBolsa?.desmontar();
    vistaAuditoriaBolsa = null;
    if (!estado.auditoriaReferencia) {
      contenedor.innerHTML = `<section class="panel"><div class="cuerpo-panel vacio-controlado" role="status">${textoPortal("auditoria_sin_ficha")}</div></section>`;
      return;
    }
    vistaAuditoriaBolsa = recursosVistas.get("auditoria").montarVistaAuditoria({ raiz: contenedor,
      fuente: crearFuenteAuditoriaHTTP(), expedienteRef: estado.auditoriaReferencia,
      fuenteContexto: "bolsa", anunciar });
    return;
  }
  if (vista === "contratos") {
    contenedor.innerHTML = recursosVistas.get("operaciones").crearVistasOperaciones(utilidadesVista)
      .renderizarContratos({ contratos_fuente: { estado: "no_configurado" } });
    return;
  }
  if (vista === "llamamientos") {
    contenedor.innerHTML = renderizarPantallaLlamamientos({ estado, encabezadoVista, escaparHTML, presentador: presentadorPanelInterno });
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
  if (vistaBolsas && estado.datosBolsas === null) {
    // Sin lectura del cuadro (p. ej. al recargar con F5 en #bolsa/resumen): se
    // pide ahora y cargarBolsas pinta la vista «cargando» y después el cuadro.
    pedirCuadroBolsas();
    return;
  }
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
    contenedor.innerHTML = renderizarFuenteNoDisponible(); return;
  }
  if (vistaBolsas && presentadorPanelInterno.esActivo()) {
    contenedor.innerHTML = presentadorPanelInterno.renderizarVista(vista); return;
  }
  contenedor.innerHTML = renderizarFuenteNoDisponible();
}

function aplicarRutaCandidatosBolsa() {
  if (estado.vista !== "bolsa-candidatos" || estado.datosBolsas?.carga !== "listo"
    || !rutasBolsa || !controladorBolsas) return false;
  const bolsas = estado.datosBolsas.datos?.bolsas;
  let filtroGlobal;
  try { filtroGlobal = rutasBolsa.leerGlobalBolsaCompartible(window.location.search); }
  catch { filtroGlobal = null; }
  if (filtroGlobal) {
    const ruta = `${window.location.pathname}${window.location.search}${window.location.hash}`;
    if (rutaCandidatosAplicada?.ruta === ruta && rutaCandidatosAplicada?.datos === estado.datosBolsas.datos) return true;
    rutaCandidatosAplicada = { ruta, datos: estado.datosBolsas.datos };
    const pagina = estado.datosCandidatos;
    if (pagina?.global && pagina.carga === "listo" && pagina.filtro === filtroGlobal.filtro
      && pagina.corte === filtroGlobal.corte && pagina.bolsa === filtroGlobal.bolsa) return true;
    estado.filtrosBolsa = {};
    void controladorBolsas.cargarGlobalBolsa(filtroGlobal.filtro, filtroGlobal);
    return true;
  }
  let filtro;
  try { filtro = rutasBolsa.leerCandidatosBolsaCompartible(window.location.search, bolsas); }
  catch { filtro = null; }
  if (!filtro) {
    rutaCandidatosAplicada = null;
    history.replaceState(null, "", rutasBolsa.rutaResumenBolsasCompartible(window.location.search));
    navegar("resumen", { enfocar: false });
    anunciar(traducirPortal("txt_la_vista_solicitada_no_esta_autorizada_para_el_p"));
    return true;
  }
  const ruta = `${window.location.pathname}${window.location.search}${window.location.hash}`;
  if (rutaCandidatosAplicada?.ruta === ruta && rutaCandidatosAplicada?.datos === estado.datosBolsas.datos) return true;
  rutaCandidatosAplicada = { ruta, datos: estado.datosBolsas.datos };
  const mostrarHistorico = estado.bolsaSeleccionada === filtro.bolsaRef
    && estado.filtrosBolsa?.pestana === "historico";
  estado.bolsaSeleccionada = filtro.bolsaRef;
  estado.filtrosBolsa = { estado: filtro.estado, texto: "",
    ...(mostrarHistorico ? { pestana: "historico" } : {}),
    ...(filtro.seguimiento ? { seguimiento: { llamamiento_ref: filtro.seguimiento, bolsa_ref: filtro.bolsaRef } } : {}) };
  // El origen de una petición de personal vive fuera de los filtros: el asistente lo copia al iniciarse.
  estado.origenLlamamientoB7 = filtro.origen ? Object.freeze({ ...filtro.origen, bolsa_ref: filtro.bolsaRef }) : null;
  estado.datosCandidatos = null;
  const carga = controladorBolsas.cargarCandidatosBolsa(filtro.bolsaRef, { enfocarDestino: true });
  // En el seguimiento el origen espera a «Llamar al siguiente»; no abre el asistente.
  if (filtro.origen && !filtro.seguimiento) {
    void carga.then(() => {
      if (estado.vista !== "bolsa-candidatos" || estado.bolsaSeleccionada !== filtro.bolsaRef
        || estado.datosCandidatos?.carga !== "listo" || estado.filtrosBolsa?.nuevo_llamamiento) return;
      // La acción existente prepara plazo, correo y confirmación. No se emite nada aquí.
      porId("espacio-trabajo")?.querySelector('[data-bolsa-accion="iniciar-b7"]')?.click();
      porId("espacio-trabajo")?.querySelector('[aria-current="step"]')?.focus?.({ preventScroll: true });
    });
  }
  return true;
}

function actualizarVistaBolsa({ activar = false } = {}) { actualizarNavegacionModulos();
  if (estado.vista === "bolsa-candidatos" && estado.datosBolsas?.carga === "listo") aplicarRutaCandidatosBolsa();
  if (estado.vista === "llamamientos" && estado.datosBolsas?.carga === "listo") selectorLlamamientos?.consumirRuta();
  const contenedor = porId("espacio-trabajo");
  if (contenedor && VISTAS_INTERNAS_BOLSA.includes(estado.vista)) {
    if (estado.vista.startsWith("seleccion-")) { renderizar(); return; }
    montarVistaBolsa(estado.vista, contenedor, {}, { activar });
    return;
  }
  if (estado.vista === "portal") renderizarConservandoFoco();
  else renderizar();
}
function renderizar() {
  const contenedor = porId("espacio-trabajo");
  if (!contenedor) return;
  if (!vistaPermitida(estado.vista)) {
    estado.vista = "portal";
    history.replaceState(null, "", rutaDeVista("portal"));
  }
  const grupoEstilos = grupoEstilosDeVista(estado.vista);
  const estilosVistaNecesarios = grupoEstilos && coordinadorModulos.vistaDisponible(estado.vista);
  if (estilosVistaNecesarios) cargarEstilosVista(grupoEstilos);
  const necesitaBolsaBase = estado.vista !== "solicitudes" && moduloDeVistaPortal(estado.vista) === "bolsa"
    && !vistaBolsaPendienteNoCompuesta(estado.vista);
  if (necesitaBolsaBase && !controladorBolsas && !promesaBolsaBase && !errorBolsaBase) {
    void prepararBolsaBase().catch(() => {});
  }
  if (estado.vista === "portal" && !recursosVistas.has("inicio")
    && !cargasRecursosVistas.has("inicio") && !erroresRecursosVistas.has("inicio")) cargarRecursosVista("inicio");
  if (estado.vista === "portal" && esPerfilRRHH()
    && ["cargando", "error"].includes(estadoResumenInicio)) {
    const fallo = estadoResumenInicio === "error";
    contenedor.innerHTML = `<section class="panel" ${fallo ? 'role="alert"' : 'role="status" aria-busy="true"'}><div class="cuerpo-panel"><p>${textoPortal(fallo ? "estado_modulo_no_disponible_titulo" : "estado_modulo_comprobando")}</p>${fallo ? `<button type="button" class="boton-secundario" data-resumen-inicio-reintentar>${textoPortal("accion_reintentar")}</button>` : ""}</div></section>`;
    if (fallo) requestAnimationFrame(() => {
      if (estado.vista === "portal" && estadoResumenInicio === "error") {
        contenedor.querySelector("[data-resumen-inicio-reintentar]")?.focus({ preventScroll: true });
      }
    });
    return;
  }
  const grupoTextos = grupoTextosDeVista(estado.vista);
  if (grupoTextos && !textosVistaPreparados(grupoTextos)) {
    const vistaPendiente = estado.vista;
    const [migas, titulo] = tituloDeVista(vistaPendiente);
    porId("migas-pan").textContent = migas;
    porId("titulo-vista").textContent = titulo;
    const error = erroresTextosVistas.has(grupoTextos);
    contenedor.innerHTML = `<section class="panel" ${error ? 'role="alert"' : 'role="status" aria-busy="true"'}><div class="cuerpo-panel"><p>${textoPortal(error ? (grupoTextos === "preferencias" ? "preferencias_textos_error" : "estado_modulo_no_disponible_titulo") : "estado_modulo_comprobando")}</p>${error ? `<button type="button" class="boton-secundario" data-textos-vista-reintentar="${grupoTextos}">${textoPortal("accion_reintentar")}</button>` : ""}</div></section>`;
    if (error) {
      requestAnimationFrame(() => {
        if (estado.vista === vistaPendiente && erroresTextosVistas.has(grupoTextos)) {
          contenedor.querySelector("[data-textos-vista-reintentar]")?.focus({ preventScroll: true });
        }
      });
    } else esperarTextosDeVista(grupoTextos);
    return;
  }
  if (estilosVistaNecesarios && !estilosVistasListos.has(grupoEstilos)) {
    const error = erroresEstilosVistas.has(grupoEstilos);
    const vistaPendiente = estado.vista;
    const [migas, titulo] = tituloDeVista(estado.vista);
    porId("migas-pan").textContent = migas;
    porId("titulo-vista").textContent = titulo;
    contenedor.innerHTML = `<section class="panel" ${error ? 'role="alert"' : 'role="status" aria-busy="true"'}><div class="cuerpo-panel"><p>${textoPortal(error ? "descripcion_superficie_no_montada" : "estado_modulo_comprobando")}</p>${error ? `<button type="button" class="boton-secundario" data-estilos-vista-reintentar="${grupoEstilos}">${textoPortal("accion_reintentar")}</button>` : ""}</div></section>`;
    if (error) requestAnimationFrame(() => {
      if (estado.vista === vistaPendiente && erroresEstilosVistas.has(grupoEstilos)) {
        contenedor.querySelector("[data-estilos-vista-reintentar]")?.focus({ preventScroll: true });
      }
    });
    return;
  }
  if (necesitaBolsaBase && !controladorBolsas) {
    const error = errorBolsaBase;
    const [migas, titulo] = tituloDeVista(estado.vista);
    porId("migas-pan").textContent = migas;
    porId("titulo-vista").textContent = titulo;
    contenedor.innerHTML = `<section class="panel" ${error ? 'role="alert"' : 'role="status" aria-busy="true"'}><div class="cuerpo-panel"><p>${textoPortal(error ? "descripcion_superficie_no_montada" : "estado_modulo_comprobando")}</p>${error ? `<button type="button" class="boton-secundario" data-bolsa-base-reintentar>${textoPortal("accion_reintentar")}</button>` : ""}</div></section>`;
    return;
  }
  const grupoRecursos = grupoRecursosVista(estado.vista);
  const recursosVistaNecesarios = grupoRecursos
    && (!["documentos", "accesos"].includes(grupoRecursos) || coordinadorModulos.vistaDisponible(estado.vista));
  if (recursosVistaNecesarios && !recursosVistas.has(grupoRecursos)) {
    const error = erroresRecursosVistas.has(grupoRecursos);
    const vistaPendiente = estado.vista;
    const [migas, titulo] = tituloDeVista(vistaPendiente);
    porId("migas-pan").textContent = migas;
    porId("titulo-vista").textContent = titulo;
    contenedor.innerHTML = `<section class="panel" ${error ? 'role="alert"' : 'role="status" aria-busy="true"'}><div class="cuerpo-panel"><p>${textoPortal(error ? "descripcion_superficie_no_montada" : "estado_modulo_comprobando")}</p>${error ? `<button type="button" class="boton-secundario" data-recursos-vista-reintentar>${textoPortal("accion_reintentar")}</button>` : ""}</div></section>`;
    if (error) requestAnimationFrame(() => {
      if (estado.vista === vistaPendiente && erroresRecursosVistas.has(grupoRecursos)) {
        contenedor.querySelector("[data-recursos-vista-reintentar]")?.focus({ preventScroll: true });
      }
    });
    if (!error) cargarRecursosVista(grupoRecursos);
    return;
  }
  if (grupoRecursos === "ofertas" && !superficieOfertasBolsa) {
    const recursos = recursosVistas.get("ofertas");
    superficieOfertasBolsa = recursos.crearSuperficieOfertasBolsa({ anunciar,
      alCambiar: () => { if (estado.vista === "llamamientos") actualizarVistaBolsa(); } });
    superficieOfertasBolsa.instalar(document);
    superficieRRHHPlazos = recursos.crearSuperficieRRHHPlazos({ anunciar,
      alCambiar: () => { if (estado.vista === "llamamientos") actualizarVistaBolsa(); } });
  }
  const [migas, titulo] = tituloDeVista(estado.vista);
  const moduloActivo = moduloActivoDeVista(estado.vista);
  porId("migas-pan").textContent = migas;
  porId("titulo-vista").textContent = titulo;
  const etiquetaAyuda = textosGrupoPortalPreparados("ayuda")
    ? traducirPortal("ayuda_abrir_contextual", { contexto: titulo })
    : traducirPortal("txt_ayuda_pantalla");
  document.querySelectorAll('[data-accion="ayuda"]').forEach((control) => {
    control.setAttribute("aria-label", etiquetaAyuda);
  });
  const ayudaLateral = document.querySelector('.enlace-lateral[data-accion="ayuda"]');
  if (ayudaLateral) ayudaLateral.hidden = estado.vista === "portal" && esPerfilRRHH();
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
  if (estado.vista === "mis-preferencias") {
    coordinadorModulos.retirarVistaMontada();
    estado.vistaMontada = "";
    contenedor.innerHTML = superficiePreferencias.renderizar();
    return;
  }
  coordinadorModulos.prepararVista(estado.vista);
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
        : renderizarNoDisponibleDeVista(estado.vista, titulo);
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
      )}<section class="panel"><div class="cuerpo-panel vacio-controlado"><p>${escaparHTML(error instanceof Error ? error.message : traducirPortal("txt_error_de_composicion"))}</p></div></section>`;
    });
    return;
  }

  coordinadorModulos.retirarVistaMontada();
  estado.vistaMontada = "";
  if (estado.vista !== "portal" && estado.vista !== "ofertas-sae" && !estado.fuenteLista) {
    contenedor.innerHTML = renderizarNoDisponibleDeVista(estado.vista, titulo);
    return;
  }

  const renderizadores = {
    portal: renderizarPortal,
    "ofertas-sae": renderizarOfertasSAE,
  };
  contenedor.innerHTML = (renderizadores[estado.vista] || renderizarPortal)();
  contenedor.querySelectorAll('[data-accion="ayuda"]').forEach((control) => {
    control.setAttribute("aria-label", etiquetaAyuda);
  });
  aplicarBarrasDinamicas(contenedor);
}

// Ofertas al SAE: entrada fija del menú de RRHH. Aún no hay datos que mostrar:
// la pantalla lo dice en llano, sin cifras ni recuentos inventados.
function renderizarOfertasSAE() {
  return `${encabezadoVista("", traducirPortal("ofertas_sae_titulo"), "")}
    <section class="panel" aria-labelledby="ofertas-sae-estado">
      <div class="cuerpo-panel vacio-controlado" role="status">
        <h3 id="ofertas-sae-estado">${escaparHTML(traducirPortal("ofertas_sae_estado_titulo"))}</h3>
        <p>${escaparHTML(traducirPortal("ofertas_sae_estado_texto"))}</p>
        <div class="acciones-vista">
          <button type="button" class="boton-secundario" data-vista="portal">${escaparHTML(traducirPortal("ofertas_sae_volver"))}</button>
        </div>
      </div>
    </section>`;
}

// Vista pedida por el enlace cuyo módulo aún carga: ni error ni «no disponible».
function renderizarVistaComprobando(titulo) {
  return `${encabezadoVista("", titulo, "")}
    <section class="panel"><div class="cuerpo-panel vacio-controlado" role="status">
      <p><strong>${escaparHTML(traducirPortal("estado_modulo_comprobando"))}</strong></p>
    </div></section>`;
}

// Vista de otro módulo (Personal, Cronos, Trámites, Dietas…) no ofrecida: «no disponible» con su título, no la de Bolsa.
function renderizarVistaNoDisponible(titulo) {
  return `${encabezadoVista("", titulo, "")}<section class="panel" aria-labelledby="vista-no-disponible-titulo"><div class="cuerpo-panel vacio-controlado">
    <h3 id="vista-no-disponible-titulo">${escaparHTML(traducirPortal("vista_no_disponible_titulo"))}</h3><p>${escaparHTML(traducirPortal("vista_no_disponible_texto"))}</p>
    <div class="acciones-vista"><button type="button" class="boton-secundario" data-vista="portal">${escaparHTML(traducirPortal("accion_volver_portal"))}</button></div></div></section>`;
}
function renderizarNoDisponibleDeVista(vista, titulo) {
  return moduloDeVistaPortal(vista) === "bolsa" ? renderizarFuenteNoDisponible() : renderizarVistaNoDisponible(titulo);
}
function renderizarFuenteNoDisponible() {
  const cargando = estado.errorFuente === "";
  const detalle = cargando
    ? traducirPortal("txt_se_esta_comprobando_la_sesion_y_el_ambito_de_acc")
    : estado.errorFuente;
  return `
    ${encabezadoVista(traducirPortal("txt_acceso_interno_cerrado"), traducirPortal("txt_gestion_de_bolsas_no_disponible"), detalle)}
    <section class="panel">
      <div class="cuerpo-panel vacio-controlado">
        <p><strong>${cargando ? traducirPortal("txt_comprobando_acceso") : traducirPortal("txt_no_se_han_cargado_datos_de_bolsa")}</strong></p>
        <p>${escaparHTML(detalle)}</p>
        <div class="acciones-vista">
          <button type="button" class="boton-secundario" data-vista="portal">${escaparHTML(traducirPortal("accion_volver_portal"))}</button>
          ${cargando ? "" : '<button type="button" class="boton-primario" data-accion="recargar-fuente">' + textoPortal("txt_reintentar") + '</button>'}
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
// Ofertas publicadas de la bolsa elegida (art. 8.1): módulo propio con sus eventos.
let superficieOfertasBolsa = null;
let superficieRRHHPlazos = null;

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
      indicador.textContent = traducirPortal("txt_cambios_locales_sin_guardar");
      indicador.className = "estado-chip";
    }
    const guardar = formulario?.querySelector("[data-borrador-guardar]");
    if (guardar?.dataset.capacidad === "true") guardar.disabled = false;
  });
  document.addEventListener("submit", (evento) => {
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

let presentadorPanelInterno = null;
let controladorBolsas = null;
let rutasBolsa = null;
let selectorLlamamientos = null;
let promesaRutasBolsa = null;
function prepararRutasBolsa() {
  if (rutasBolsa) return Promise.resolve(rutasBolsa);
  promesaRutasBolsa ??= import("./portal-bolsas-ruta-filtros.js?v=20261010-seguimiento-siguiente-v1")
    .then((rutas) => { rutasBolsa = rutas; return rutas; })
    .catch((error) => { promesaRutasBolsa = null; throw error; });
  return promesaRutasBolsa;
}
let rutaCandidatosAplicada = null;
let promesaBolsaBase = null;
let errorBolsaBase = false;
function vistaNecesitaBolsa(vista = estado.vista) {
  return vista === "portal" || (vista !== "solicitudes" && moduloDeVistaPortal(vista) === "bolsa"
    && !vistaBolsaPendienteNoCompuesta(vista));
}
function prepararBolsaBase() {
  if (controladorBolsas && presentadorPanelInterno) return Promise.resolve();
  if (promesaBolsaBase) return promesaBolsaBase;
  promesaBolsaBase = Promise.all([
    import("./portal-panel-interno.js?v=20261010-seguimiento-siguiente-v1"),
    import("./portal-bolsas-api.js?v=20261010-seguimiento-siguiente-v1"),
    import("./portal-bolsas-ruta-filtros.js?v=20261010-seguimiento-siguiente-v1"),
  ]).then(([panel, bolsas, rutas]) => {
    if (!vistaNecesitaBolsa()) {
      promesaBolsaBase = null;
      return;
    }
    const presentador = panel.crearPresentadorPanelInterno({
  claseEstado, encabezadoVista, escaparHTML, numero,
  obtenerDatosPanel: () => DATOS_PANEL,
  tituloVista: (vista) => TITULOS[vista]?.[1] || traducirPortal("txt_seccion_de_bolsa"),
  obtenerDatosBolsas: () => estado.datosBolsas,
  obtenerDatosCandidatosBolsa: () => estado.datosCandidatos,
  obtenerDatosEstadisticas: () => estado.datosEstadisticas,
  obtenerDatosAvisos: () => estado.datosAvisos,
  obtenerEstadoCandidatos: () => estado.filtrosBolsa,
  obtenerModalContactos: () => estado.modalContactos,
  obtenerModalFicha: () => estado.modalFicha,
  obtenerModalResultado: () => estado.modalResultado,
    });

    const controlador = bolsas.crearControladorBolsas({
      estado, renderizar: actualizarVistaBolsa, navegar, obtenerFuenteLectura: () => null,
    });
    controlador.instalar();
    selectorLlamamientos = instalarSelectorLlamamientos({ documento: document, estado, controladorBolsas: controlador,
      actualizarVistaBolsa, porId, rutasBolsa: rutas, obtenerUbicacion: () => window.location,
      limpiarRuta: () => {
        if (window.location.hash !== "#bolsa/llamamientos" || !new URLSearchParams(window.location.search).has("bolsa_ref")) return;
        history.replaceState(null, "", rutaPortalConFiltroCT(window.location, "#bolsa/llamamientos"));
      } });
    presentadorPanelInterno = presentador;
    controladorBolsas = controlador;
    rutasBolsa = rutas;
    errorBolsaBase = false;
    if (moduloDeVistaPortal(estado.vista) === "bolsa") renderizar();
  }).catch((error) => {
    promesaBolsaBase = null;
    errorBolsaBase = true;
    if (vistaNecesitaBolsa()) renderizar();
    throw error;
  });
  return promesaBolsaBase;
}

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

function instalarEnlacesBolsa() {
  document.addEventListener("click", async (evento) => {
    const control = evento.target?.closest?.('[data-accion="ver-bolsa"][data-bolsa-ref]');
    if (!control || !porId("espacio-trabajo")?.contains(control)) return;
    const esEnlace = control.tagName?.toLowerCase() === "a" && control.hasAttribute("href");
    evento.stopImmediatePropagation();
    if (esEnlace && (evento.button > 0 || evento.ctrlKey || evento.metaKey || evento.shiftKey || evento.altKey)) return;
    evento.preventDefault();
    if (estado.filtrosBolsa?.nuevo_llamamiento?.enviando) return;
    const vistaAlClic = estado.vista;
    const cuadroBolsaAlClic = estado.datosBolsas;
    const esLlamamientoCT = estado.vista === "contratacion-temporal"
      && Boolean(control.closest?.("[data-ct-bolsa-ficha]"));
    const esHistoricoCT = estado.vista === "contratacion-temporal" && !esLlamamientoCT
      && control.dataset?.pestana === "historico" && control.classList?.contains("enlace-tabla")
      && Boolean(control.closest?.(".ct-expedientes"));
    const esFichaCT = esLlamamientoCT || esHistoricoCT;
    const fichaCT = esFichaCT ? contextoBolsaCT : null;
    const bolsas = esFichaCT
      ? fichaCT?.carga === "listo" ? fichaCT.datos?.bolsas : null
      : estado.datosBolsas?.carga === "listo" ? estado.datosBolsas.datos?.bolsas : null;
    let destino; let filtro;
    try {
      const datos = control.dataset;
      if (!Array.isArray(bolsas) || (esFichaCT && (!fichaCT?.expedienteRef
        || !bolsas.some((bolsa) => bolsa?.bolsa_ref === datos.bolsaRef)
        || esLlamamientoCT && datos.origenExpediente !== fichaCT.expedienteRef
        || esHistoricoCT && (datos.origenExpediente || datos.estado || datos.seguimiento)))) throw new TypeError("Bolsa pendiente");
      let rutas;
      try { rutas = await prepararRutasBolsa(); }
      catch {
        anunciar(traducirPortal("txt_no_se_pudieron_cargar_las_bolsas_de_trabajo"));
        return;
      }
      // La ficha puede cambiar mientras se carga el helper de rutas. Su bolsa
      // y su origen solo valen para la selección que atendió este clic.
      if (!porId("espacio-trabajo")?.contains(control) || estado.vista !== vistaAlClic
        || (!esFichaCT && estado.datosBolsas !== cuadroBolsaAlClic)
        || esFichaCT && (estado.vista !== "contratacion-temporal" || contextoBolsaCT !== fichaCT)) return;
      const origen = datos.origenExpediente ? { expediente_ref: datos.origenExpediente, referencia: datos.origenReferencia,
        centro: datos.origenCentro, fecha_inicio: datos.origenInicio } : null;
      const href = esEnlace ? control.getAttribute("href") : esLlamamientoCT
        ? rutas.rutaLlamamientoBolsaCompartible(window.location.search, datos.bolsaRef, origen)
        : rutas.rutaCandidatosBolsaCompartible(window.location.search, datos.bolsaRef,
          datos.estado ?? "", { seguimiento: datos.seguimiento ?? "", origen });
      destino = new URL(href, window.location.href);
      if (destino.origin !== window.location.origin || destino.pathname !== window.location.pathname
        || destino.hash !== (esLlamamientoCT ? "#bolsa/llamamientos" : "#bolsa/bolsa-candidatos")) throw new TypeError("destino ajeno");
      filtro = esLlamamientoCT ? rutas.leerLlamamientoBolsaCompartible(destino.search, bolsas)
        : rutas.leerCandidatosBolsaCompartible(destino.search, bolsas);
      if (!filtro || filtro.bolsaRef !== datos.bolsaRef || (filtro.estado ?? "") !== (datos.estado ?? "")
        || (filtro.seguimiento ?? "") !== (datos.seguimiento ?? "") || Boolean(filtro.origen) !== Boolean(origen)) throw new TypeError("filtro ajeno");
      if (esLlamamientoCT && (filtro.seguimiento || !filtro.origen
        || filtro.origen.expediente_ref !== fichaCT.expedienteRef
        || filtro.origen.referencia !== origen.referencia
        || filtro.origen.centro !== origen.centro
        || filtro.origen.fecha_inicio !== origen.fecha_inicio)) throw new TypeError("origen ajeno");
      if (esHistoricoCT && (filtro.origen || filtro.seguimiento)) throw new TypeError("histórico ajeno");
    } catch {
      anunciar(traducirPortal("txt_la_vista_solicitada_no_esta_autorizada_para_el_p"));
      return;
    }
    if (esLlamamientoCT) {
      estado.bolsaSeleccionada = "";
      estado.filtrosBolsa = { estado: "", texto: "" };
      estado.datosCandidatos = null;
      estado.origenLlamamientoB7 = null;
      rutaCandidatosAplicada = null;
      const ruta = `${destino.pathname}${destino.search}${destino.hash}`;
      if (`${window.location.pathname}${window.location.search}${window.location.hash}` !== ruta)
        history.pushState(null, "", ruta);
      navegar("llamamientos", { enfocar: false, desdeRuta: true });
      return;
    }
    if (filtro.seguimiento) controladorBolsas?.olvidarEmisionConfirmada?.();
    estado.bolsaSeleccionada = filtro.bolsaRef;
    estado.filtrosBolsa = { estado: filtro.estado, texto: "",
      ...(control.dataset.pestana === "candidatos" ? { pestana: "candidatos" }
        : esHistoricoCT ? { pestana: "historico" } : {}) };
    estado.datosCandidatos = null;
    estado.origenLlamamientoB7 = null;
    rutaCandidatosAplicada = null;
    const ruta = `${destino.pathname}${destino.search}${destino.hash}`;
    if (`${window.location.pathname}${window.location.search}${window.location.hash}` !== ruta)
      history.pushState(null, "", ruta);
    navegar("bolsa-candidatos", { enfocar: false });
    aplicarRutaCandidatosBolsa();
  }, true);
  window.addEventListener("popstate", () => {
    const vista = vistaDesdeHash();
    if (vista !== estado.vista || vista === "contratacion-temporal")
      navegar(vista, { enfocar: false, desdeRuta: true });
    if (vista !== "bolsa-candidatos") return;
    rutaCandidatosAplicada = null;
    aplicarRutaCandidatosBolsa();
  });
}

function instalarEventosAuditoriaBolsa() {
  document.addEventListener("click", (evento) => {
    const boton = evento.target?.closest?.("[data-bolsa-auditoria]");
    if (!boton || !porId("espacio-trabajo")?.contains(boton)) return;
    const referencia = boton.dataset.bolsaAuditoria;
    if (estado.modalFicha?.abierto !== true
      || estado.modalFicha.candidato?.participacion_ref !== referencia) return;
    evento.preventDefault();
    estado.auditoriaReferencia = referencia;
    navegar("auditoria");
  });
}

const integracionPreferencias = crearIntegracionPreferenciasPortal({ documento: document, ventana: window, porId, estado, renderizar,
  aplicarFilas: aplicarFilasMarco, navegar, vistaBolsaDisponible: () => vistaPermitida("resumen") && resolverAccesoPerfil("bolsa").disponible === true,
  altaCTDisponible: () => coordinadorModulos.altaCTDisponible(), anunciar, traducir: traducirPortal });
const superficiePreferencias = integracionPreferencias.superficie;
const controlador = crearControladorPortal({
  anunciar, cargarFuenteDatos,
  cerrarMenuMovil, comprobarDisponibilidadBorradores: superficieBorradores.comprobarDisponibilidad,
  escaparHTML, estado,
  etiquetaFuentePanel, navegar, notaOperacionNoCompuesta, numero,
  obtenerDatosPanel: () => DATOS_PANEL, porcentajeSeguro, porId, renderizar,
  renderizarContenidoAyuda, solicitarPropuestaLlamamiento, traducir: traducirPortal, vistaDesdeHash,
  alternarPreferenciaVisual: integracionPreferencias.alternarVisualVolatil,
});

async function inicializar() {
  aplicarTextosPortal(document);
  aplicarIdiomaDocumento(document);
  instalarSelectorIdiomaPortal(document);
  instalarValidacionI18n(document);
  configurarInicioInstitucional();
  controlador.restaurarPreferencias();
  const aperturaSinDestino = !window.location.hash || window.location.hash === "#portal";
  const vistaInicial = vistaDesdeHash();
  destinoPlantillasInicial = vistaInicial === VISTA_PLANTILLAS_RRHH;
  destinoPoliticaCeseInicial = vistaInicial === "reglas";
  estado.vista = destinoPlantillasInicial || destinoPoliticaCeseInicial ? "portal" : vistaInicial;
  estado.opcionesVista = opcionesDesdeEnlace(estado.vista);
  renderizar();
  controlador.instalar();
  integracionPreferencias.instalarMenu();
  instalarEventosAvisosBolsa();
  instalarEnlacesBolsa();
  instalarEventosAuditoriaBolsa();
  instalarMenuBolsa(porId("navegacion-bolsa"));
  instalarCopiaJustificantes(document);
  instalarEventosBorradores(); instalarPaginacionMarco(); instalarReintentoTextosVista();
  superficiePreferencias.instalar(porId("espacio-trabajo"));
  void actualizarSesionVisible();
  const cargaPreferencias = superficiePreferencias.cargar();
  if (integracionPreferencias.prepararInicio()) {
    await cargaPreferencias;
    if (integracionPreferencias.cambioIdiomaPendiente()) return;
  }
  // cargarFuenteDatos pinta el resultado: Inicio conservando el foco y una
  // vista de módulo solo si no se montó ya durante la carga.
  await cargarFuenteDatos();
  await cargaPreferencias;
  if (aperturaSinDestino) integracionPreferencias.aplicarInicio();
}

// El índice de idiomas se carga con await de nivel superior en comun/idioma.js,
// así que este módulo puede evaluarse después de DOMContentLoaded.
if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", inicializar, { once: true });
} else {
  inicializar();
}
