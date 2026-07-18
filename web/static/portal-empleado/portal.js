import { crearControladorPortal } from "./portal-eventos.js?v=20260717-portal-rrhh";
import { crearPresentadorPanelInterno } from "./portal-panel-interno.js?v=20260717-panel-interno-v1";
import {
  extraerDatosEnvelopeCanonico,
  validarPanelBolsa,
} from "./portal-contrato.js?v=20260717-panel-interno-v1";
import { crearClientePropuestasLlamamiento } from "./portal-llamamientos-api.js?v=20260718-llamamientos-v1";
import { resolverSolicitudPropuestaLlamamiento } from "./portal-llamamientos-flujo.js?v=20260718-llamamientos-v1";
import {
  renderizarConfirmacionCompacta, renderizarDetalleLlamamientoBloqueado, renderizarPasosLlamamiento,
} from "./portal-llamamientos-vista.js?v=20260718-llamamientos-v1";
import { AYUDA_PORTAL_BOLSA } from "./ayuda-contenido.js?v=20260717-ayuda";
import { PROVEEDOR_BEARER_BORRADORES, crearSuperficieBorradoresPortal } from "./portal-borradores-ui.js?v=20260718-borradores-v1";
import { crearUtilidadesVista } from "./portal-vistas-utilidades.js?v=20260718-demo-total-v1";
import { crearVistasConvocatorias } from "./portal-vistas-convocatorias.js?v=20260718-demo-total-v1";
import { crearVistasBaremacion } from "./portal-vistas-baremacion.js?v=20260718-demo-total-v1";
import { crearVistasOperaciones } from "./portal-vistas-operaciones.js?v=20260718-demo-total-v1";
import { crearVistasGobierno } from "./portal-vistas-gobierno.js?v=20260718-demo-total-v1";

/**
 * SUPERFICIE DEFINITIVA DEL PORTAL RRHH.
 *
 * La ruta normal obtiene datos exclusivamente de la API interna protegida. El
 * juego sintético está aislado en `datos-presentacion.js` y solo se importa si
 * la URL declara `?presentacion=rrhh`. Ninguna mutación de negocio se ejecuta
 * en el navegador salvo el guardado durable de borradores, aislado tras su
 * cliente autenticado, CAS e idempotencia. Mapa de sustitución y límites:
 * docs/portal_vec/entregable_rrhh_bolsa_2026-07-17.md.
 */
const API_PANEL_BOLSA = "/api/vec/bolsa/panel";
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
let superficieBorradoresPresentacion = null;
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
});

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
  solicitandoPropuesta: false,
  errorPropuesta: "",
};

const porId = (id) => document.getElementById(id);

function cerrarMenuMovil() {
  delete document.body.dataset.menuAbierto;
  porId("boton-menu")?.setAttribute("aria-expanded", "false");
  if (porId("velo-menu")) porId("velo-menu").hidden = true;
}

function escaparHTML(valor) {
  return String(valor ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

function renderizarContenidoAyuda() {
  const ayuda = AYUDA_PORTAL_BOLSA;
  return `<section class="ayuda-contextual"><p>${escaparHTML(ayuda.introduccion)}</p><h3>Pasos</h3><ol class="lista-ayuda">${ayuda.pasos.map((paso) => `<li>${escaparHTML(paso)}</li>`).join("")}</ol><section class="ayuda-audio" aria-labelledby="titulo-audio-ayuda"><h3 id="titulo-audio-ayuda">Escuchar esta guía</h3><audio controls preload="metadata" aria-describedby="transcripcion-ayuda"><source src="${escaparHTML(ayuda.audio.src)}" type="${escaparHTML(ayuda.audio.tipo)}">Su navegador no puede reproducir este audio.</audio></section><section class="faq-ayuda"><h3>Preguntas frecuentes</h3>${ayuda.preguntas.map((item) => `<details><summary>${escaparHTML(item.pregunta)}</summary><p>${escaparHTML(item.respuesta)}</p></details>`).join("")}</section><details id="transcripcion-ayuda" class="transcripcion-ayuda"><summary>Transcripción del audio</summary><p>${escaparHTML(ayuda.transcripcion)}</p></details></section>`;
}

function numero(valor, decimales = 0) {
  return new Intl.NumberFormat("es-ES", {
    minimumFractionDigits: decimales,
    maximumFractionDigits: decimales,
  }).format(Number(valor || 0));
}

function porcentajeSeguro(valor) {
  const numeroValor = Number(valor);
  if (!Number.isFinite(numeroValor)) return 0;
  return Math.max(0, Math.min(100, Math.round(numeroValor * 10) / 10));
}

function opcionesSelect(valores, seleccionada = "") {
  if (!Array.isArray(valores)) return "";
  return valores.map((valor) => `<option ${String(valor) === String(seleccionada) ? "selected" : ""}>${escaparHTML(valor)}</option>`).join("");
}

function modoPresentacionSolicitado() {
  return new URLSearchParams(window.location.search).get("presentacion") === "rrhh";
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

function ejecutarOperacionPresentacion(operacion, objetivo, motivo) {
  if (!estado.modoPresentacion || adaptadorPresentacion === null) {
    throw new Error("el adaptador de presentación no está activo");
  }
  const recibo = adaptadorPresentacion.ejecutar({ operacion, objetivo, motivo });
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
    if (avisos) {
      avisos.textContent = numero(DATOS_PANEL.indicadores.avisos_pendientes);
      avisos.setAttribute("aria-label", `${numero(DATOS_PANEL.indicadores.avisos_pendientes)} avisos pendientes`);
    }
    return;
  }
  if (estado.fuenteLista && presentadorPanelInterno.actualizarContextoSesion({ avatar, nombre, perfil, avisos })) {
    return;
  }
  avatar.textContent = "—";
  nombre.textContent = "Sesión no resuelta";
  perfil.textContent = estado.errorFuente || "La API interna debe identificar al usuario";
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
      const [adaptador, moduloEfectos, moduloBorradores] = await Promise.all([
        import("./datos-presentacion.js?v=20260718-demo-total-v1"),
        import("./portal-presentacion-adaptador.js?v=20260718-demo-total-v1"),
        import("./portal-borradores-demo-cliente.js?v=20260718-demo-total-v1"),
      ]);
      const datosIniciales = validarPanelBolsa(adaptador.obtenerDatosPresentacion(), true);
      adaptadorPresentacion = moduloEfectos.crearAdaptadorPresentacion({ datosIniciales });
      superficieBorradoresPresentacion = crearSuperficieBorradoresPortal({
        escaparHTML,
        anunciar,
        alCambiar: () => { if (estado.vista === "elaboracion") renderizar(); },
        resolverProveedorBearer: () => null,
        confirmar: (mensaje) => window.confirm(mensaje),
        crearClienteImpl: () => moduloBorradores.crearClienteBorradoresPresentacion(),
      });
      DATOS_PANEL = adaptadorPresentacion.obtenerDatos();
      obtenerPropuestaPresentacion = adaptador.obtenerPropuestaPresentacion;
      aviso.hidden = false;
      estado.fuenteLista = true;
      estado.necesidadSeleccionada = DATOS_PANEL.necesidades_llamamiento[0]?.id || "";
      estado.elaboracionSeleccionada = DATOS_PANEL.elaboraciones[0]?.id || "";
    } catch {
      aviso.hidden = true;
      estado.errorFuente = "No se pudo cargar el adaptador aislado de presentación.";
      estado.fuenteLista = false;
    }
    actualizarSesionVisible();
    return;
  }

  adaptadorPresentacion = null;
  superficieBorradoresPresentacion = null;
  aviso.hidden = true;
  try {
    const respuesta = await fetch(API_PANEL_BOLSA, {
      method: "GET",
      credentials: "omit",
      headers: { Accept: "application/json" },
    });
    if (!respuesta.ok) {
      if (respuesta.status === 401) throw new Error("Se requiere una sesión interna autenticada.");
      if (respuesta.status === 403) throw new Error("La sesión no dispone de ámbito para gestionar Bolsas.");
      if (respuesta.status === 404 || respuesta.status === 501) throw new Error("La API interna del panel de Bolsa aún no está compuesta.");
      throw new Error(`No se pudo cargar el panel (HTTP ${respuesta.status}).`);
    }
    const envelope = await respuesta.json();
    DATOS_PANEL = validarPanelBolsa(extraerDatosEnvelopeCanonico(envelope), false);
    estado.fuenteLista = true;
    actualizarSesionVisible();
  } catch (error) {
    estado.errorFuente = error instanceof Error ? error.message : "No se pudo cargar la fuente interna.";
    estado.fuenteLista = false;
    actualizarSesionVisible();
  }
}

function necesidadLlamamientoSeleccionada() {
  return DATOS_PANEL.necesidades_llamamiento.find((item) => item.id === estado.necesidadSeleccionada)
    || DATOS_PANEL.necesidades_llamamiento[0];
}

function bolsaDeNecesidad(necesidad) {
  return DATOS_PANEL.bolsas.find((item) => item.id === necesidad?.bolsa_id);
}

function puedeSolicitarPropuesta() {
  return DATOS_PANEL.capacidades.solicitar_propuesta_llamamiento === true;
}

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
  if (!valor || valor === "portal") return "portal";
  const segmentos = valor.split("/").filter(Boolean);
  const candidata = segmentos.at(-1);
  return Object.hasOwn(TITULOS, candidata) ? candidata : "portal";
}

function rutaDeVista(vista) {
  return vista === "portal" ? "#portal" : `#bolsa/${vista}`;
}

function anunciar(mensaje) {
  const region = porId("anuncios");
  if (!region) return;
  region.textContent = "";
  window.setTimeout(() => { region.textContent = mensaje; }, 20);
}

function navegar(vista, opciones = {}) {
  if (!Object.hasOwn(TITULOS, vista)) return;
  const hash = rutaDeVista(vista);
  if (window.location.hash !== hash) history.pushState(null, "", hash);
  estado.vista = vista;
  renderizar();
  cerrarMenuMovil();
  if (opciones.enfocar !== false) porId("contenido-principal")?.focus({ preventScroll: true });
  anunciar(`Vista ${TITULOS[vista][1]} abierta`);
}

function renderizar() {
  const contenedor = porId("espacio-trabajo");
  if (!contenedor) return;
  const [migas, titulo] = TITULOS[estado.vista] || TITULOS.portal;
  porId("migas-pan").textContent = migas;
  porId("titulo-vista").textContent = titulo;
  porId("navegacion-bolsa").hidden = estado.vista === "portal";

  document.querySelectorAll("[data-vista]").forEach((boton) => {
    const actual = boton.dataset.vista === estado.vista ||
      (boton.classList.contains("modulo-habilitado") && estado.vista !== "portal");
    if (actual) boton.setAttribute("aria-current", "page");
    else boton.removeAttribute("aria-current");
  });

  if (estado.vista === "elaboracion" && estado.modoPresentacion && !estado.fuenteLista) {
    contenedor.innerHTML = renderizarFuenteNoDisponible();
    return;
  }

  if (estado.vista === "elaboracion") {
    const superficie = estado.modoPresentacion ? superficieBorradoresPresentacion : superficieBorradores;
    if (superficie === null) {
      contenedor.innerHTML = renderizarFuenteNoDisponible();
      return;
    }
    contenedor.innerHTML = superficie.renderizar();
    void superficie.activar();
    return;
  }

  if (estado.vista !== "portal" && !estado.fuenteLista) {
    contenedor.innerHTML = renderizarFuenteNoDisponible();
    return;
  }

  if (estado.vista === "resumen" && presentadorPanelInterno.esActivo()) {
    contenedor.innerHTML = presentadorPanelInterno.renderizarVista(estado.vista);
    return;
  }

  const datosVista = { ...DATOS_VACIOS, ...DATOS_PANEL };

  const renderizadores = {
    portal: renderizarPortal,
    resumen: renderizarResumen,
    convocatorias: () => vistasConvocatorias.renderizarConvocatorias(datosVista, estado),
    solicitudes: () => vistasConvocatorias.renderizarSolicitudes(datosVista),
    meritos: () => vistasBaremacion.renderizarMeritos(datosVista),
    baremacion: () => vistasBaremacion.renderizarBaremacion(datosVista),
    alegaciones: () => vistasBaremacion.renderizarAlegaciones(datosVista),
    importacion: () => vistasOperaciones.renderizarImportacion(datosVista),
    llamamientos: () => vistasOperaciones.renderizarLlamamientos(datosVista),
    contratos: () => vistasOperaciones.renderizarContratos(datosVista),
    reglas: () => vistasBaremacion.renderizarBaremacion(datosVista),
    consulta: renderizarConsulta,
    estadisticas: () => vistasGobierno.renderizarEstadisticas(datosVista),
    documentos: () => vistasOperaciones.renderizarDocumentos(datosVista),
    comunicaciones: () => vistasOperaciones.renderizarComunicaciones(datosVista),
    auditoria: () => vistasGobierno.renderizarAuditoria(datosVista),
    configuracion: () => vistasGobierno.renderizarConfiguracion(datosVista),
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
          <button type="button" class="boton-secundario" data-vista="portal">Volver al portal</button>
          ${cargando ? "" : '<button type="button" class="boton-primario" data-accion="recargar-fuente">Reintentar</button>'}
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
});
const vistasConvocatorias = crearVistasConvocatorias(utilidadesVista);
const vistasBaremacion = crearVistasBaremacion(utilidadesVista);
const vistasOperaciones = crearVistasOperaciones(utilidadesVista);
const vistasGobierno = crearVistasGobierno(utilidadesVista);

function renderizarPortal() {
  const modulos = [
    { clave: "bolsa", sigla: "BOL", titulo: "Bolsas de trabajo", texto: "Elaboración, integrantes, llamamientos, contratos, documentos y trazabilidad.", habilitado: true },
    { clave: "personal", sigla: "PER", titulo: "Personal", texto: "Datos personales, puesto, situación administrativa y servicios prestados." },
    { clave: "nominas", sigla: "NOM", titulo: "Nóminas", texto: "Recibos, certificados fiscales e incidencias retributivas." },
    { clave: "cronos", sigla: "CRO", titulo: "Cronos", texto: "Fichajes, permisos, vacaciones, turnos y calendario laboral." },
    { clave: "dietas", sigla: "DIE", titulo: "Dietas", texto: "Comisiones de servicio, kilometraje, gastos y liquidaciones." },
    { clave: "solicitudes", sigla: "SOL", titulo: "Solicitudes y certificados", texto: "Ventanilla única, aportación documental y seguimiento." },
    { clave: "meritos", sigla: "RUM", titulo: "Méritos y formación", texto: "Títulos, cursos, experiencia y evidencias reutilizables." },
    { clave: "comunicaciones", sigla: "AVI", titulo: "Comunicaciones", texto: "Avisos personales, notificaciones y preferencias de canal." },
  ];
  return `
    ${encabezadoVista("Acceso unificado", "Portal del Empleado", "Los módulos comparten identidad, datos y documentos. En esta primera fase únicamente está habilitada la Gestión de Bolsas.")}
    <section class="nota-seguridad" aria-label="Separación de acceso">
      Este portal representa el acceso interno de RRHH. La zona externa de aspirantes tendrá otra sesión, permisos y proyección de datos; nunca mostrará expedientes de terceras personas.
    </section>
    <div class="rejilla-modulos" aria-label="Módulos del Portal del Empleado">
      ${modulos.map((modulo) => `
        <article class="tarjeta-modulo ${modulo.habilitado ? "tarjeta-modulo-habilitada" : "tarjeta-modulo-bloqueada"}">
          <span class="icono-modulo" aria-hidden="true">${escaparHTML(modulo.sigla)}</span>
          <h3>${escaparHTML(modulo.titulo)}</h3>
          <p>${escaparHTML(modulo.texto)}</p>
          <div class="pie-tarjeta">
            <span class="${modulo.habilitado ? "estado-disponible" : "estado-proximamente"}">${modulo.habilitado ? "Habilitado en fase inicial" : "No habilitado"}</span>
            ${modulo.habilitado ? '<button type="button" class="boton-primario" data-vista="resumen">Entrar</button>' : '<button type="button" class="boton-secundario" disabled>No disponible</button>'}
          </div>
        </article>`).join("")}
    </div>`;
}

function tarjetaKPI(sigla, valor, etiqueta, destino) {
  return `
    <article class="tarjeta-kpi">
      <span class="icono-kpi" aria-hidden="true">${escaparHTML(sigla)}</span>
      <div>
        <strong class="valor-kpi">${escaparHTML(valor)}</strong>
        <span class="etiqueta-kpi">${escaparHTML(etiqueta)}</span>
        <button type="button" class="enlace-kpi" data-vista="${escaparHTML(destino)}">Ver detalle →</button>
      </div>
    </article>`;
}

function renderizarResumen() {
  const indicadores = DATOS_PANEL.indicadores;
  const distribucion = DATOS_PANEL.distribucion_global;
  const totalDistribucion = Object.values(distribucion).reduce((total, valor) => total + (Number(valor) || 0), 0);
  const porcentajes = [distribucion.disponibles, distribucion.en_llamamiento, distribucion.contratados]
    .map((valor) => totalDistribucion > 0 ? porcentajeSeguro((Number(valor) || 0) * 100 / totalDistribucion) : 0);
  const contratosMes = Array.isArray(DATOS_PANEL.series.contratos_mes) ? DATOS_PANEL.series.contratos_mes : [];
  const bolsasTabla = DATOS_PANEL.bolsas.slice(0, 5).map((bolsa) => `
    <tr>
      <td><button type="button" class="enlace-tabla" data-accion="ver-bolsa" data-id="${escaparHTML(bolsa.id)}">${escaparHTML(bolsa.nombre)}</button></td>
      <td>${escaparHTML(bolsa.categoria)}</td>
      <td>${numero(bolsa.integrantes)}</td>
      <td>${numero(bolsa.llamamiento)}</td>
      <td><span class="barra-cobertura" role="meter" aria-label="Cobertura ${porcentajeSeguro(bolsa.cobertura)}%" aria-valuenow="${porcentajeSeguro(bolsa.cobertura)}" aria-valuemin="0" aria-valuemax="100"><span data-ancho="${porcentajeSeguro(bolsa.cobertura)}"></span></span>${porcentajeSeguro(bolsa.cobertura)}%</td>
      <td><button type="button" class="boton-terciario" data-accion="ver-bolsa" data-id="${escaparHTML(bolsa.id)}">Abrir</button></td>
    </tr>`).join("");
  return `
    ${encabezadoVista("Gestión interna de Bolsas", "Cuadro de mando", "Situación operativa, próximos llamamientos, cobertura y actividad trazada.", '<button type="button" class="boton-secundario" data-accion="imprimir">Imprimir resumen</button><button type="button" class="boton-primario" data-accion="nuevo-llamamiento">Nuevo llamamiento</button>')}
    <div class="rejilla-kpi" aria-label="Indicadores de Bolsa">
      ${tarjetaKPI("BOL", numero(indicadores.bolsas_activas), "Bolsas activas", "elaboracion")}
      ${tarjetaKPI("DIS", numero(indicadores.candidatos_disponibles), "Candidatos disponibles", "elaboracion")}
      ${tarjetaKPI("LLA", numero(indicadores.llamamientos_pendientes), "Llamamientos pendientes", "llamamientos")}
      ${tarjetaKPI("CON", numero(indicadores.contratos_activos), "Contratos activos", "contratos")}
      ${tarjetaKPI("COB", `${numero(indicadores.cobertura_media, 1)} %`, "Cobertura media", "estadisticas")}
    </div>
    <div class="rejilla-cuadro-mando">
      <div class="columna-cuadro">
        <section class="panel">
          <div class="cabecera-panel"><h3>Bolsas destacadas</h3><button type="button" class="boton-terciario" data-vista="elaboracion">Ver todas</button></div>
          <div class="tabla-contenedor">
            <table class="tabla-datos">
              <caption>Bolsas destacadas con disponibilidad y cobertura</caption>
              <thead><tr><th scope="col">Bolsa</th><th scope="col">Categoría</th><th scope="col">Integrantes</th><th scope="col">En llamamiento</th><th scope="col">Cobertura</th><th scope="col">Acción</th></tr></thead>
              <tbody>${bolsasTabla}</tbody>
            </table>
          </div>
        </section>
        <section class="panel">
          <div class="cabecera-panel"><h3>Indicadores clave</h3><button type="button" class="boton-terciario" data-vista="estadisticas">Abrir informe</button></div>
          <div class="graficos-resumen">
            <article class="grafico-mini"><h4>Candidatos por estado</h4><div class="grafico-anillo" data-anillo-a="${porcentajes[0]}" data-anillo-b="${porcentajes[1]}" data-anillo-c="${porcentajes[2]}" role="img" aria-label="${numero(distribucion.disponibles)} disponibles, ${numero(distribucion.en_llamamiento)} en llamamiento, ${numero(distribucion.contratados)} contratados y ${numero(distribucion.no_disponibles)} no disponibles"></div><p class="texto-grafico">${numero(distribucion.disponibles)} disponibles · total ${numero(totalDistribucion)}</p></article>
            <article class="grafico-mini"><h4>Cobertura por bolsa</h4><div class="barras-mini" role="img" aria-label="Cobertura por bolsa">${DATOS_PANEL.bolsas.slice(0, 5).map((bolsa) => `<span data-altura="${porcentajeSeguro(bolsa.cobertura)}"></span>`).join("")}</div><p class="texto-grafico">Media de las bolsas destacadas</p></article>
            <article class="grafico-mini"><h4>Contratos por mes</h4><div class="linea-mini" role="img" aria-label="Evolución mensual de contratos">${contratosMes.map((valor) => `<span data-altura="${porcentajeSeguro(valor)}"></span>`).join("")}</div><p class="texto-grafico">${escaparHTML(DATOS_PANEL.series.periodo_contratos)}</p></article>
          </div>
        </section>
      </div>
      <div class="columna-cuadro">
        <section class="panel">
          <div class="cabecera-panel"><h3>Llamamientos próximos (7 días)</h3><button type="button" class="boton-terciario" data-vista="llamamientos">Ver todos</button></div>
          <ol class="lista-proximos">
            ${DATOS_PANEL.proximos.map((item) => `<li class="elemento-proximo"><span class="fecha-proximo"><span><strong>${escaparHTML(item.dia)}</strong>${escaparHTML(item.mes)}</span></span><span class="detalle-lista"><strong>${escaparHTML(item.bolsa)}</strong><span>Llamamiento n.º ${escaparHTML(item.numero)} · ${escaparHTML(item.fecha)}</span></span><span class="insignia">${escaparHTML(item.estado)}</span></li>`).join("")}
          </ol>
        </section>
        <section class="panel">
          <div class="cabecera-panel"><h3>Actividad reciente</h3><button type="button" class="boton-terciario" data-vista="auditoria">Ver trazabilidad</button></div>
          <ol class="lista-actividad">
            ${DATOS_PANEL.actividad.map((item) => `<li class="elemento-actividad"><span class="marca-actividad" aria-hidden="true"></span><span class="detalle-lista"><strong>${escaparHTML(item.accion)}: ${escaparHTML(item.objeto)}</strong><span>Por ${escaparHTML(item.actor)} · ${escaparHTML(item.fecha)}</span></span></li>`).join("")}
          </ol>
        </section>
      </div>
    </div>
    <section class="panel">
      <div class="cabecera-panel"><h3>Accesos rápidos</h3><span class="estado-chip info">${escaparHTML(etiquetaFuentePanel())}</span></div>
      <div class="accesos-rapidos">
        <button type="button" class="acceso-rapido" data-accion="nuevo-llamamiento">Nuevo llamamiento</button>
        <button type="button" class="acceso-rapido" data-vista="elaboracion">Elaborar una bolsa</button>
        <button type="button" class="acceso-rapido" data-vista="contratos">Registrar contrato</button>
        <button type="button" class="acceso-rapido" data-vista="contratos">Registrar cese</button>
        <button type="button" class="acceso-rapido" data-vista="documentos">Generar documento</button>
        <button type="button" class="acceso-rapido" data-vista="comunicaciones">Preparar comunicación</button>
      </div>
    </section>`;
}

function claseEstado(estadoTexto) {
  const texto = String(estadoTexto).toLowerCase();
  if (/publicad|activa|disponible|firmad|válid|complet/.test(texto)) return "exito";
  if (/borrador|pendiente|revisión|preparad/.test(texto)) return "";
  if (/error|revocad|excluid|no disponible/.test(texto)) return "peligro";
  if (/configur|recib|enviad/.test(texto)) return "info";
  return "neutro";
}

function renderizarReglas() {
  const filas = DATOS_PANEL.reglas.map((item) => `<tr><td>${escaparHTML(item.nombre)}</td><td>${escaparHTML(item.ambito)}</td><td>${escaparHTML(item.version)}</td><td>${escaparHTML(item.vigencia)}</td><td><span class="estado-chip ${claseEstado(item.estado)}">${escaparHTML(item.estado)}</span></td><td><button class="boton-terciario" data-accion="detalle-regla">Abrir</button></td></tr>`).join("");
  return `
    ${encabezadoVista("Configuración sin recompilar", "Motor de reglas configurable", "Versionado de criterios derivados de las bases y del Reglamento, con pruebas antes de publicar.", '<button type="button" class="boton-primario" data-accion="nueva-regla">Nueva versión de reglas</button>')}
    <section class="nota-seguridad">Una regla publicada nunca se modifica: se crea otra versión con vigencia y motivo. Los cálculos conservan la versión exacta que los produjo.</section>
    <div class="rejilla-dos-columnas">
      <section class="panel"><div class="cabecera-panel"><h3>Conjuntos de reglas</h3><span class="estado-chip info">Fuente del panel</span></div><div class="tabla-contenedor"><table class="tabla-datos"><caption>Versiones de reglas de llamamiento</caption><thead><tr><th scope="col">Regla</th><th scope="col">Ámbito</th><th scope="col">Versión</th><th scope="col">Vigencia</th><th scope="col">Estado</th><th scope="col">Acción</th></tr></thead><tbody>${filas || '<tr><td colspan="6" class="vacio-controlado">No hay reglas accesibles.</td></tr>'}</tbody></table></div></section>
      <aside class="resumen-lateral"><section class="panel"><div class="cabecera-panel"><h3>Campos gobernados</h3></div><ul class="lista-comprobacion"><li>Fuente jurídica y artículo</li><li>Ámbito y categoría</li><li>Versión, vigencia y sustitución</li><li>Condiciones, excepciones y desempates</li><li>Pruebas con casos límite</li><li>Firmas y publicación</li></ul></section><section class="nota-pendiente">No se copiarán coeficientes del ejemplo a producción: RRHH debe configurarlos desde las bases aprobadas.</section></aside>
    </div>`;
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

function resolverProveedorBearerBorradores() {
  const proveedor = globalThis[PROVEEDOR_BEARER_BORRADORES];
  return proveedor === undefined ? null : proveedor;
}

function superficieBorradoresActiva() {
  return estado.modoPresentacion ? superficieBorradoresPresentacion : superficieBorradores;
}

const superficieBorradores = crearSuperficieBorradoresPortal({
  escaparHTML,
  anunciar,
  alCambiar: () => { if (estado.vista === "elaboracion") renderizar(); },
  resolverProveedorBearer: resolverProveedorBearerBorradores,
  confirmar: (mensaje) => window.confirm(mensaje),
});

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
  claseEstado,
  encabezadoVista,
  escaparHTML,
  numero,
  obtenerDatosPanel: () => DATOS_PANEL,
  tituloVista: (vista) => TITULOS[vista]?.[1] || "Sección de Bolsa",
});

const controlador = crearControladorPortal({
  anunciar, cargarFuenteDatos, confirmarOperacionPresentacion: (mensaje) => window.confirm(mensaje),
  cerrarMenuMovil, describirOperacionPresentacion, escaparHTML, estado,
  etiquetaFuentePanel, ejecutarOperacionPresentacion, navegar, notaOperacionNoCompuesta, numero,
  obtenerDatosPanel: () => DATOS_PANEL, porcentajeSeguro, porId, renderizar,
  renderizarContenidoAyuda, solicitarPropuestaLlamamiento, vistaDesdeHash,
});

async function inicializar() {
  estado.modoPresentacion = modoPresentacionSolicitado();
  controlador.restaurarPreferencias();
  estado.vista = vistaDesdeHash();
  renderizar();
  controlador.instalar();
  instalarEventosBorradores();
  await cargarFuenteDatos();
  renderizar();
}

document.addEventListener("DOMContentLoaded", inicializar, { once: true });
