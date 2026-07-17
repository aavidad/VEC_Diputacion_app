import { crearControladorPortal } from "./portal-eventos.js?v=20260717-portal-rrhh";

/**
 * SUPERFICIE DEFINITIVA DEL PORTAL RRHH.
 *
 * La ruta normal obtiene datos exclusivamente de la API interna protegida. El
 * juego sintético está aislado en `datos-presentacion.js` y solo se importa si
 * la URL declara `?presentacion=rrhh`. Ninguna mutación de negocio se ejecuta
 * en el navegador. Mapa de sustitución y límites:
 * docs/portal_vec/entregable_rrhh_bolsa_2026-07-17.md.
 */
const API_PANEL_BOLSA = "/api/vec/bolsa/panel";
const DATOS_VACIOS = Object.freeze({
  esquema: "vec.bolsa.panel.v1",
  demostracion: false,
  sesion: null,
  indicadores: {},
  estados_candidatos: {},
  distribucion_global: {},
  series: {},
  avisos: [],
  configuracion_llamamiento: {},
  catalogos_llamamiento: {},
  bolsas: [],
  candidatos: [],
  elaboraciones: [],
  proximos: [],
  actividad: [],
  contratos: [],
  reglas: [],
  documentos: [],
  canales: [],
  auditoria: {},
});

let DATOS_PANEL = DATOS_VACIOS;

const TITULOS = Object.freeze({
  portal: ["Portal del Empleado", "Portal del Empleado"],
  resumen: ["Portal del Empleado → Bolsas de trabajo", "Cuadro de mando"],
  elaboracion: ["Portal del Empleado → Bolsas de trabajo", "Elaboración y gestión de bolsas"],
  llamamientos: ["Portal del Empleado → Bolsas de trabajo → Llamamientos", "Nuevo llamamiento"],
  contratos: ["Portal del Empleado → Bolsas de trabajo", "Contratos, ceses y reincorporaciones"],
  reglas: ["Portal del Empleado → Bolsas de trabajo", "Motor de reglas configurable"],
  consulta: ["Portal del Empleado → Bolsas de trabajo", "Consulta segura para candidatos"],
  estadisticas: ["Portal del Empleado → Bolsas de trabajo", "Estadísticas y explotación de datos"],
  documentos: ["Portal del Empleado → Bolsas de trabajo", "Generación y firma de documentos"],
  comunicaciones: ["Portal del Empleado → Bolsas de trabajo", "Correo y mensajería"],
  auditoria: ["Portal del Empleado → Bolsas de trabajo", "Auditoría y trazabilidad"],
});

const estado = {
  vista: "portal",
  fuenteLista: false,
  modoPresentacion: false,
  errorFuente: "",
  pasoLlamamiento: 1,
  bolsaSeleccionada: "",
  elaboracionSeleccionada: "",
  candidatosSeleccionados: new Set(),
  filtroEstado: "Disponible",
  busquedaCandidato: "",
  puntosDesde: "",
  puntosHasta: "",
  respetarPrelacion: true,
};

const porId = (id) => document.getElementById(id);

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
  return estado.modoPresentacion ? "Datos sintéticos aislados" : "API interna autorizada";
}

function notaOperacionNoCompuesta() {
  return estado.modoPresentacion
    ? "Recorrido de presentación: no escribe en el servidor ni genera un acto administrativo."
    : "Esta operación permanece deshabilitada hasta que su comando de servidor esté compuesto y autorizado.";
}

function validarPanelBolsa(datos, admiteDemostracion) {
  if (!datos || typeof datos !== "object" || Array.isArray(datos)) {
    throw new Error("respuesta del panel no válida");
  }
  const esquemaEsperado = admiteDemostracion ? "vec.bolsa.panel.presentacion.v1" : "vec.bolsa.panel.v1";
  if (datos.esquema !== esquemaEsperado) {
    throw new Error("versión de contrato del panel no compatible");
  }
  const listas = ["bolsas", "candidatos", "elaboraciones", "proximos", "actividad", "contratos", "reglas", "documentos", "canales", "avisos"];
  if (listas.some((clave) => !Array.isArray(datos[clave]))) {
    throw new Error("respuesta del panel incompleta");
  }
  if (datos.demostracion === true && !admiteDemostracion) {
    throw new Error("la API interna no puede responder con datos de demostración");
  }
  return {
    esquema: String(datos.esquema || ""),
    demostracion: datos.demostracion === true,
    sesion: datos.sesion && typeof datos.sesion === "object" ? { ...datos.sesion } : null,
    indicadores: datos.indicadores && typeof datos.indicadores === "object" ? { ...datos.indicadores } : {},
    estados_candidatos: datos.estados_candidatos && typeof datos.estados_candidatos === "object" ? { ...datos.estados_candidatos } : {},
    distribucion_global: datos.distribucion_global && typeof datos.distribucion_global === "object" ? { ...datos.distribucion_global } : {},
    series: datos.series && typeof datos.series === "object" ? { ...datos.series } : {},
    avisos: [...datos.avisos],
    configuracion_llamamiento: datos.configuracion_llamamiento && typeof datos.configuracion_llamamiento === "object" ? { ...datos.configuracion_llamamiento } : {},
    catalogos_llamamiento: datos.catalogos_llamamiento && typeof datos.catalogos_llamamiento === "object" ? { ...datos.catalogos_llamamiento } : {},
    bolsas: [...datos.bolsas],
    candidatos: [...datos.candidatos],
    elaboraciones: [...datos.elaboraciones],
    proximos: [...datos.proximos],
    actividad: [...datos.actividad],
    contratos: [...datos.contratos],
    reglas: [...datos.reglas],
    documentos: [...datos.documentos],
    canales: [...datos.canales],
    auditoria: datos.auditoria && typeof datos.auditoria === "object" ? { ...datos.auditoria } : {},
  };
}

function actualizarSesionVisible() {
  const sesion = porId("sesion-visible");
  if (!sesion) return;
  const datos = DATOS_PANEL.sesion;
  const avatar = sesion.querySelector(".avatar");
  const nombre = sesion.querySelector("strong");
  const perfil = sesion.querySelector("small");
  const avisos = document.querySelector(".boton-avisos span");
  if (estado.fuenteLista && datos) {
    avatar.textContent = String(datos.iniciales || "RR").slice(0, 3);
    nombre.textContent = String(datos.nombre || "Sesión interna");
    perfil.textContent = String(datos.perfil || "Perfil autorizado");
    if (avisos) {
      avisos.textContent = numero(DATOS_PANEL.indicadores.avisos_pendientes);
      avisos.setAttribute("aria-label", `${numero(DATOS_PANEL.indicadores.avisos_pendientes)} avisos pendientes`);
    }
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
      const adaptador = await import("./datos-presentacion.js?v=20260717-portal-rrhh");
      DATOS_PANEL = validarPanelBolsa(adaptador.obtenerDatosPresentacion(), true);
      aviso.hidden = false;
      estado.fuenteLista = true;
      estado.bolsaSeleccionada = DATOS_PANEL.bolsas[0]?.id || "";
      estado.elaboracionSeleccionada = DATOS_PANEL.elaboraciones[0]?.id || "";
      estado.candidatosSeleccionados = new Set(DATOS_PANEL.candidatos.filter((item) => item.seleccionado === true).map((item) => item.id));
    } catch {
      aviso.hidden = true;
      estado.errorFuente = "No se pudo cargar el adaptador aislado de presentación.";
      estado.fuenteLista = false;
    }
    actualizarSesionVisible();
    return;
  }

  aviso.hidden = true;
  try {
    const respuesta = await fetch(API_PANEL_BOLSA, {
      method: "GET",
      credentials: "same-origin",
      headers: { Accept: "application/json" },
    });
    if (!respuesta.ok) {
      if (respuesta.status === 401) throw new Error("Se requiere una sesión interna autenticada.");
      if (respuesta.status === 403) throw new Error("La sesión no dispone de ámbito para gestionar Bolsas.");
      if (respuesta.status === 404 || respuesta.status === 501) throw new Error("La API interna del panel de Bolsa aún no está compuesta.");
      throw new Error(`No se pudo cargar el panel (HTTP ${respuesta.status}).`);
    }
    DATOS_PANEL = validarPanelBolsa(await respuesta.json(), false);
    estado.fuenteLista = true;
    estado.bolsaSeleccionada = DATOS_PANEL.bolsas[0]?.id || "";
    estado.elaboracionSeleccionada = DATOS_PANEL.elaboraciones[0]?.id || "";
    actualizarSesionVisible();
  } catch (error) {
    estado.errorFuente = error instanceof Error ? error.message : "No se pudo cargar la fuente interna.";
    estado.fuenteLista = false;
    actualizarSesionVisible();
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

  if (estado.vista !== "portal" && !estado.fuenteLista) {
    contenedor.innerHTML = renderizarFuenteNoDisponible();
    return;
  }

  const renderizadores = {
    portal: renderizarPortal,
    resumen: renderizarResumen,
    elaboracion: renderizarElaboracion,
    llamamientos: renderizarLlamamiento,
    contratos: renderizarContratos,
    reglas: renderizarReglas,
    consulta: renderizarConsulta,
    estadisticas: renderizarEstadisticas,
    documentos: renderizarDocumentos,
    comunicaciones: renderizarComunicaciones,
    auditoria: renderizarAuditoria,
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

function renderizarElaboracion() {
  const seleccion = DATOS_PANEL.elaboraciones.find((item) => item.id === estado.elaboracionSeleccionada) || DATOS_PANEL.elaboraciones[0];
  if (!seleccion) {
    return `${encabezadoVista("Bases, reglas y publicación", "Elaboración de bolsas", "No existen expedientes accesibles en el ámbito de esta sesión.")}<section class="panel"><div class="vacio-controlado">No hay expedientes de elaboración.</div></section>`;
  }
  const contarEstado = (patron) => DATOS_PANEL.elaboraciones.filter((item) => patron.test(String(item.estado))).length;
  return `
    ${encabezadoVista("Bases, reglas y publicación", "Elaboración de bolsas", "Creación gobernada desde las bases: criterios versionados, calendario, documentación, firmas y publicación trazable.", '<button type="button" class="boton-secundario" data-accion="comparar-versiones">Comparar versiones</button><button type="button" class="boton-primario" data-accion="nueva-bolsa">Nueva bolsa</button>')}
    <section class="nota-pendiente">${escaparHTML(notaOperacionNoCompuesta())} El modelo de convocatorias y revisión firmada ya existe en el núcleo; el guardado administrativo se habilitará únicamente mediante su caso de uso y repositorio.</section>
    <div class="rejilla-kpi">
      ${tarjetaKPI("BOR", numero(contarEstado(/Borrador/i)), "Borradores", "elaboracion")}
      ${tarjetaKPI("REV", numero(contarEstado(/revisión|validación/i)), "Pendiente de validación", "elaboracion")}
      ${tarjetaKPI("PUB", numero(contarEstado(/Publicada/i)), "Publicadas en la proyección", "elaboracion")}
      ${tarjetaKPI("PLA", numero(DATOS_PANEL.indicadores.plazos_proximos), "Plazo próximo", "elaboracion")}
      ${tarjetaKPI("FIR", numero(DATOS_PANEL.indicadores.firmas_pendientes), "Firmas pendientes", "documentos")}
    </div>
    <div class="rejilla-elaboracion">
      <section class="panel">
        <div class="cabecera-panel"><div><h3>Expedientes de elaboración</h3><p>Ámbito de Selección externa</p></div><span class="estado-chip info">${numero(DATOS_PANEL.elaboraciones.length)} expedientes</span></div>
        <div class="barra-filtros">
          <label class="campo"><span>Buscar bolsa o expediente</span><input type="search" value="" placeholder="Nombre, categoría o referencia"></label>
          <label class="campo"><span>Fase</span><select><option>Todas</option><option>Borrador</option><option>Validación</option><option>Publicación</option></select></label>
          <label class="campo"><span>Responsable</span><select><option>Todos</option><option>Selección externa</option><option>Jefatura RRHH</option></select></label>
          <button type="button" class="boton-secundario" data-accion="aplicar-filtros">Aplicar</button>
        </div>
        <div class="tabla-contenedor">
          <table class="tabla-datos">
            <caption>Expedientes de elaboración de bolsas</caption>
            <thead><tr><th scope="col">Bolsa / expediente</th><th scope="col">Fase</th><th scope="col">Reglas</th><th scope="col">Plazo</th><th scope="col">Estado</th><th scope="col">Acción</th></tr></thead>
            <tbody>
              ${DATOS_PANEL.elaboraciones.map((item) => `<tr aria-selected="${item.id === seleccion.id}"><td><strong>${escaparHTML(item.nombre)}</strong><br><small>${escaparHTML(item.expediente)}</small></td><td>${escaparHTML(item.fase)}</td><td>${escaparHTML(item.reglas)}</td><td>${escaparHTML(item.plazo)}</td><td><span class="estado-chip ${claseEstado(item.estado)}">${escaparHTML(item.estado)}</span></td><td><button type="button" class="boton-terciario" data-accion="seleccionar-elaboracion" data-id="${escaparHTML(item.id)}">Abrir</button></td></tr>`).join("")}
            </tbody>
          </table>
        </div>
      </section>
      <aside class="resumen-lateral" aria-label="Detalle del expediente seleccionado">
        <section class="panel">
          <div class="cabecera-panel"><div><h3>${escaparHTML(seleccion.nombre)}</h3><p>${escaparHTML(seleccion.expediente)}</p></div><span class="estado-chip ${claseEstado(seleccion.estado)}">${escaparHTML(seleccion.estado)}</span></div>
          <div class="cuerpo-panel">
            <dl class="resumen-expediente">
              <div class="fila-resumen"><dt>Unidad responsable</dt><dd>${escaparHTML(seleccion.responsable)}</dd></div>
              <div class="fila-resumen"><dt>Versión de bases</dt><dd>${escaparHTML(seleccion.version_bases)}</dd></div>
              <div class="fila-resumen"><dt>Baremo</dt><dd>${escaparHTML(seleccion.reglas)}</dd></div>
              <div class="fila-resumen"><dt>Calendario</dt><dd>${escaparHTML(seleccion.calendario)}</dd></div>
              <div class="fila-resumen"><dt>Firmantes</dt><dd>${escaparHTML(seleccion.firmantes)}</dd></div>
            </dl>
            <button type="button" class="boton-primario" data-accion="configurar-bases">Configurar bases y baremo</button>
          </div>
        </section>
        <section class="panel">
          <div class="cabecera-panel"><h3>Comprobaciones para publicar</h3></div>
          <ul class="lista-comprobacion">
            <li>Identificación, categoría y requisitos</li>
            <li>Bases y documentos versionados</li>
            <li>Baremo reproducible y topes validados</li>
            <li>Calendario de solicitud y alegaciones</li>
            <li class="pendiente">Informe jurídico pendiente</li>
            <li class="pendiente">Circuito de firmas pendiente</li>
          </ul>
        </section>
      </aside>
    </div>`;
}

function pasosLlamamiento() {
  const nombres = ["Seleccionar bolsa", "Seleccionar candidatos", "Configurar llamamiento", "Revisar y preparar"];
  return `<nav class="pasos" aria-label="Pasos del nuevo llamamiento">${nombres.map((nombre, indice) => {
    const paso = indice + 1;
    const clase = paso < estado.pasoLlamamiento ? "completado" : "";
    const actual = paso === estado.pasoLlamamiento ? ' aria-current="step"' : "";
    return `<button type="button" class="paso ${clase}" data-accion="ir-paso" data-paso="${paso}"${actual}><span class="paso-numero">${paso < estado.pasoLlamamiento ? "✓" : paso}</span><span>${escaparHTML(nombre)}</span></button>`;
  }).join("")}</nav>`;
}

function candidatosFiltrados() {
  const consulta = estado.busquedaCandidato.trim().toLocaleLowerCase("es");
  const minimo = estado.puntosDesde === "" ? Number.NEGATIVE_INFINITY : Number(estado.puntosDesde);
  const maximo = estado.puntosHasta === "" ? Number.POSITIVE_INFINITY : Number(estado.puntosHasta);
  return DATOS_PANEL.candidatos.filter((item) => {
    const coincideEstado = estado.filtroEstado === "Todos" || item.estado === estado.filtroEstado;
    const coincideTexto = !consulta || `${item.nombre} ${item.dni}`.toLocaleLowerCase("es").includes(consulta);
    return coincideEstado && coincideTexto && item.puntos >= minimo && item.puntos <= maximo;
  });
}

function resumenBolsaSeleccionada() {
  return DATOS_PANEL.bolsas.find((item) => item.id === estado.bolsaSeleccionada) || DATOS_PANEL.bolsas[0];
}

function renderizarLlamamiento() {
  const bolsa = resumenBolsaSeleccionada();
  if (!bolsa) {
    return `${encabezadoVista("Llamamientos según bases y Reglamento", "Nuevo llamamiento", "No existe ninguna bolsa accesible desde la que iniciar el llamamiento.")}<section class="panel"><div class="vacio-controlado">No hay bolsas disponibles.</div></section>`;
  }
  const candidatos = candidatosFiltrados();
  return `
    ${encabezadoVista("Llamamientos según bases y Reglamento", estado.pasoLlamamiento === 1 ? "Nuevo llamamiento" : `Nuevo llamamiento · Paso ${estado.pasoLlamamiento}`, "Asistente con selección por orden de prelación, filtros explicables, comunicaciones y revisión previa.", '<button type="button" class="boton-secundario" data-vista="resumen">Volver al cuadro de mando</button>')}
    <section class="nota-pendiente">El motor de propuesta y elegibilidad ya está probado en el núcleo. Esta interfaz todavía no persiste el llamamiento ni envía avisos: faltan el adaptador PostgreSQL, la API interna y el conector fehaciente.</section>
    ${pasosLlamamiento()}
    ${estado.pasoLlamamiento === 1 ? renderizarPasoBolsa(bolsa, candidatos) : ""}
    ${estado.pasoLlamamiento === 2 ? renderizarPasoCandidatos(bolsa) : ""}
    ${estado.pasoLlamamiento === 3 ? renderizarPasoConfiguracion(bolsa) : ""}
    ${estado.pasoLlamamiento === 4 ? renderizarPasoRevision(bolsa) : ""}`;
}

function renderizarPasoBolsa(bolsa, candidatos) {
  const estados = DATOS_PANEL.estados_candidatos;
  const categorias = ["Todas las categorías", ...new Set(DATOS_PANEL.bolsas.map((item) => item.categoria).filter(Boolean))];
  const estadosFiltro = ["Todos", ...new Set(DATOS_PANEL.candidatos.map((item) => item.estado).filter(Boolean))];
  const ambitosGeograficos = DATOS_PANEL.catalogos_llamamiento.ambitos_geograficos || [];
  return `
    <div class="distribucion-llamamiento">
      <div class="columna-cuadro">
        <section class="panel">
          <div class="cabecera-panel"><div><h3>1. Seleccionar bolsa</h3><p>Seleccione la bolsa desde la que se propondrá el llamamiento.</p></div></div>
          <div class="barra-filtros">
            <label class="campo"><span>Buscar bolsa</span><input type="search" placeholder="Nombre o categoría"></label>
            <label class="campo"><span>Categoría</span><select>${opcionesSelect(categorias, "Todas las categorías")}</select></label>
            <span></span><button type="button" class="boton-secundario" data-accion="aplicar-filtros">Buscar</button>
          </div>
          <div class="tabla-contenedor">
            <table class="tabla-datos">
              <caption>Bolsas disponibles para el llamamiento</caption>
              <thead><tr><th scope="col">Bolsa</th><th scope="col">Categoría</th><th scope="col">Creación</th><th scope="col">Integrantes</th><th scope="col">Disponibles</th><th scope="col">Acción</th></tr></thead>
              <tbody>${DATOS_PANEL.bolsas.map((item) => `<tr aria-selected="${item.id === bolsa.id}"><td><strong>${escaparHTML(item.nombre)}</strong></td><td>${escaparHTML(item.categoria)}</td><td>${escaparHTML(item.creada)}</td><td>${numero(item.integrantes)}</td><td><span class="estado-chip exito">${numero(item.disponibles)}</span></td><td><button type="button" class="boton-secundario" data-accion="seleccionar-bolsa" data-id="${escaparHTML(item.id)}">${item.id === bolsa.id ? "Seleccionada" : "Seleccionar"}</button></td></tr>`).join("")}</tbody>
            </table>
          </div>
        </section>
        <section class="panel">
          <div class="cabecera-panel"><div><h3>Bolsa seleccionada: ${escaparHTML(bolsa.nombre)}</h3><p>Los identificadores personales están enmascarados.</p></div><button type="button" class="boton-secundario" data-accion="exportar">Exportar listado</button></div>
          <div class="tarjetas-estado">
            <article class="tarjeta-estado"><span>Disponibles</span><strong>${numero(estados.disponibles)}</strong></article>
            <article class="tarjeta-estado"><span>Ocupados</span><strong>${numero(estados.ocupados)}</strong></article>
            <article class="tarjeta-estado"><span>No disponibles</span><strong>${numero(estados.no_disponibles)}</strong></article>
            <article class="tarjeta-estado"><span>Excluidos</span><strong>${numero(estados.excluidos)}</strong></article>
            <article class="tarjeta-estado"><span>Renuncia pendiente</span><strong>${numero(estados.renuncia_pendiente)}</strong></article>
            <article class="tarjeta-estado"><span>En revisión</span><strong>${numero(estados.en_revision)}</strong></article>
          </div>
          <form class="selector-candidatos" id="filtros-candidatos">
            <label>Estado <select name="estado">${opcionesSelect(estadosFiltro, estado.filtroEstado)}</select></label>
            <label>Buscar <input class="control-formulario" name="busqueda" type="search" value="${escaparHTML(estado.busquedaCandidato)}" placeholder="Nombre o DNI parcial"></label>
            <label>Puntos desde <input class="control-formulario" name="desde" type="number" step="0.001" value="${escaparHTML(estado.puntosDesde)}"></label>
            <label>Puntos hasta <input class="control-formulario" name="hasta" type="number" step="0.001" value="${escaparHTML(estado.puntosHasta)}"></label>
            <button type="submit" class="boton-secundario">Aplicar filtros</button>
          </form>
          <div class="tabla-contenedor">
            <table class="tabla-datos">
              <caption>Candidatos de la bolsa seleccionada</caption>
              <thead><tr><th scope="col"><span class="solo-lectura">Seleccionar</span></th><th scope="col">N.º orden</th><th scope="col">DNI</th><th scope="col">Apellidos y nombre</th><th scope="col">Estado</th><th scope="col">Fecha estado</th><th scope="col">Motivo</th><th scope="col">Puntos</th><th scope="col">Acción</th></tr></thead>
              <tbody>${candidatos.length ? candidatos.map((item) => `<tr><td><input type="checkbox" data-candidato="${escaparHTML(item.id)}" aria-label="Seleccionar a ${escaparHTML(item.nombre)}" ${estado.candidatosSeleccionados.has(item.id) ? "checked" : ""}></td><td>${numero(item.orden)}</td><td>${escaparHTML(item.dni)}</td><td>${escaparHTML(item.nombre)}</td><td><span class="estado-chip ${claseEstado(item.estado)}">${escaparHTML(item.estado)}</span></td><td>${escaparHTML(item.fecha)}</td><td>${escaparHTML(item.motivo)}</td><td>${numero(item.puntos, 3)}</td><td><button type="button" class="boton-terciario" data-accion="ver-candidato" data-id="${escaparHTML(item.id)}">Ver</button></td></tr>`).join("") : '<tr><td colspan="9" class="vacio-controlado">Ningún candidato coincide con los filtros.</td></tr>'}</tbody>
            </table>
          </div>
        </section>
      </div>
      <aside class="resumen-lateral" aria-label="Configuración del llamamiento">
        <section class="panel">
          <div class="cabecera-panel"><h3>Resumen del llamamiento</h3></div>
          <div class="cuerpo-panel"><dl class="resumen-expediente"><div class="fila-resumen"><dt>Bolsa</dt><dd>${escaparHTML(bolsa.nombre)}</dd></div><div class="fila-resumen"><dt>Disponibles</dt><dd>${numero(bolsa.disponibles)} integrantes</dd></div><div class="fila-resumen"><dt>Seleccionados</dt><dd>${numero(estado.candidatosSeleccionados.size)}</dd></div><div class="fila-resumen"><dt>Regla aplicable</dt><dd>${escaparHTML(bolsa.regla)}</dd></div></dl></div>
        </section>
        <section class="panel">
          <div class="cabecera-panel"><h3>2. Filtros de selección</h3></div>
          <div class="cuerpo-panel formulario-llamamiento">
            <label class="campo campo-ancho"><span>Estados a incluir</span><select><option>Disponible</option><option>Disponible y renuncia pendiente</option></select></label>
            <label class="campo campo-ancho"><span>Motivo u observación</span><select><option>Todos los motivos</option><option>Sin incidencias</option></select></label>
            <label class="campo"><span>Puntuación mínima</span><input type="number" placeholder="Desde"></label>
            <label class="campo"><span>Puntuación máxima</span><input type="number" placeholder="Hasta"></label>
            <label class="campo campo-ancho"><span>Disponibilidad geográfica</span><select>${opcionesSelect(ambitosGeograficos, ambitosGeograficos[0])}</select></label>
            <label class="campo campo-ancho"><span><input type="checkbox" id="respetar-prelacion" ${estado.respetarPrelacion ? "checked" : ""}> Respetar orden de prelación</span></label>
          </div>
        </section>
        <section class="panel">
          <div class="cabecera-panel"><h3>3. Acción</h3></div>
          <div class="cuerpo-panel">
            <dl class="resumen-expediente"><div class="fila-resumen"><dt>Cumplen filtros</dt><dd>${candidatos.length}</dd></div><div class="fila-resumen"><dt>Selección manual</dt><dd>${estado.candidatosSeleccionados.size}</dd></div></dl>
            <button type="button" class="boton-primario" data-accion="siguiente-paso">Siguiente: seleccionar candidatos →</button>
          </div>
        </section>
      </aside>
    </div>`;
}

function candidatosSeleccionados() {
  return DATOS_PANEL.candidatos.filter((item) => estado.candidatosSeleccionados.has(item.id));
}

function renderizarPasoCandidatos(bolsa) {
  const seleccionados = candidatosSeleccionados();
  const canales = Array.isArray(DATOS_PANEL.configuracion_llamamiento.canales) ? DATOS_PANEL.configuracion_llamamiento.canales : [];
  return `
    <div class="rejilla-dos-columnas">
      <section class="panel">
          <div class="cabecera-panel"><div><h3>2. Confirmar candidatos</h3><p>${escaparHTML(bolsa.nombre)} · orden de prelación protegido</p></div><span class="estado-chip info">${numero(seleccionados.length)} seleccionados</span></div>
        <div class="tabla-contenedor"><table class="tabla-datos"><caption>Candidatos seleccionados para el llamamiento</caption><thead><tr><th scope="col">Orden</th><th scope="col">Identificador</th><th scope="col">Nombre</th><th scope="col">Puntos</th><th scope="col">Canales preparados</th><th scope="col">Acción</th></tr></thead><tbody>${seleccionados.map((item) => `<tr><td>${numero(item.orden)}</td><td>${escaparHTML(item.dni)}</td><td>${escaparHTML(item.nombre)}</td><td>${numero(item.puntos, 3)}</td><td>${canales.map((canal) => `<span class="estado-chip exito">${escaparHTML(canal)}</span>`).join(" ")}</td><td><button type="button" class="boton-terciario" data-accion="quitar-candidato" data-id="${escaparHTML(item.id)}">Retirar</button></td></tr>`).join("") || '<tr><td colspan="6" class="vacio-controlado">No hay candidatos seleccionados. Vuelva al paso anterior.</td></tr>'}</tbody></table></div>
      </section>
      <aside class="resumen-lateral">
        <section class="panel"><div class="cabecera-panel"><h3>Garantías aplicadas</h3></div><ul class="lista-comprobacion"><li>Orden y puntuación ligados a versión</li><li>Identidad enmascarada en la vista</li><li>Elegibilidad reevaluable antes del envío</li><li>Motivo de exclusión obligatorio</li><li class="pendiente">Recepción fehaciente aún no conectada</li></ul></section>
        <section class="panel"><div class="cuerpo-panel"><div class="acciones-paso"><button type="button" class="boton-secundario" data-accion="anterior-paso">← Volver</button><button type="button" class="boton-primario" data-accion="siguiente-paso" ${seleccionados.length ? "" : "disabled"}>Configurar llamamiento →</button></div></div></section>
      </aside>
    </div>`;
}

function renderizarPasoConfiguracion(bolsa) {
  const configuracion = DATOS_PANEL.configuracion_llamamiento;
  const catalogos = DATOS_PANEL.catalogos_llamamiento;
  const canales = Array.isArray(configuracion.canales) ? configuracion.canales.join(" + ") : "";
  return `
    <div class="rejilla-dos-columnas">
      <section class="panel">
        <div class="cabecera-panel"><div><h3>3. Configurar llamamiento</h3><p>${escaparHTML(bolsa.nombre)} · ${numero(estado.candidatosSeleccionados.size)} candidatos</p></div></div>
        <form class="cuerpo-panel formulario-llamamiento" id="configuracion-llamamiento">
          <label class="campo"><span>Fecha y hora de apertura</span><input name="apertura" type="datetime-local" value="${escaparHTML(configuracion.apertura)}"></label>
          <label class="campo"><span>Plazo para responder</span><select name="plazo">${opcionesSelect(catalogos.plazos_respuesta, configuracion.plazo_respuesta)}</select></label>
          <label class="campo"><span>Tipo de cobertura</span><select name="tipo">${opcionesSelect(catalogos.tipos_cobertura, configuracion.tipo_cobertura)}</select></label>
          <label class="campo"><span>Centro o destino</span><input name="destino" value="${escaparHTML(configuracion.destino)}"></label>
          <label class="campo"><span>Jornada</span><select name="jornada">${opcionesSelect(catalogos.jornadas, configuracion.jornada)}</select></label>
          <label class="campo"><span>Duración prevista</span><input name="duracion" value="${escaparHTML(configuracion.duracion)}"></label>
          <label class="campo campo-ancho"><span>Canales</span><input name="canales" value="${escaparHTML(canales)}" readonly></label>
          <label class="campo campo-ancho"><span>Observaciones visibles para la persona llamada</span><textarea name="observaciones">${escaparHTML(configuracion.observaciones)}</textarea></label>
        </form>
        <div class="cuerpo-panel acciones-paso"><button type="button" class="boton-secundario" data-accion="anterior-paso">← Volver</button><button type="button" class="boton-primario" data-accion="siguiente-paso">Revisar llamamiento →</button></div>
      </section>
      <aside class="resumen-lateral"><section class="panel"><div class="cabecera-panel"><h3>Política de contacto</h3></div><div class="cuerpo-panel"><dl class="resumen-expediente"><div class="fila-resumen"><dt>Orden</dt><dd>Prelación estricta</dd></div><div class="fila-resumen"><dt>Intentos</dt><dd>Configurables por las bases</dd></div><div class="fila-resumen"><dt>Acuse</dt><dd>Obligatorio para efecto administrativo</dd></div><div class="fila-resumen"><dt>Silencio</dt><dd>No inferido: depende de la regla publicada</dd></div></dl></div></section><section class="nota-seguridad">Los canales serán conectores intercambiables. El núcleo solo aceptará el resultado cuando exista un recibo verificable del proveedor configurado.</section></aside>
    </div>`;
}

function renderizarPasoRevision(bolsa) {
  const seleccionados = candidatosSeleccionados();
  const configuracion = DATOS_PANEL.configuracion_llamamiento;
  const canales = Array.isArray(configuracion.canales) ? configuracion.canales.join(", ") : "";
  return `
    <div class="rejilla-dos-columnas">
      <section class="panel">
        <div class="cabecera-panel"><div><h3>4. Revisar y preparar</h3><p>La preparación no envía mensajes ni modifica la bolsa.</p></div><span class="estado-chip">Pendiente de validación</span></div>
        <div class="cuerpo-panel">
          <dl class="resumen-expediente"><div class="fila-resumen"><dt>Bolsa</dt><dd>${escaparHTML(bolsa.nombre)}</dd></div><div class="fila-resumen"><dt>Candidatos</dt><dd>${numero(seleccionados.length)} seleccionados según orden visible</dd></div><div class="fila-resumen"><dt>Apertura</dt><dd>${escaparHTML(configuracion.apertura_visible)}</dd></div><div class="fila-resumen"><dt>Respuesta</dt><dd>${escaparHTML(configuracion.plazo_respuesta)} desde recepción fehaciente</dd></div><div class="fila-resumen"><dt>Canales</dt><dd>${escaparHTML(canales)}</dd></div><div class="fila-resumen"><dt>Datos de destino</dt><dd>${escaparHTML(configuracion.destino)} · ${escaparHTML(configuracion.jornada)} · ${escaparHTML(configuracion.duracion)}</dd></div><div class="fila-resumen"><dt>Evidencia prevista</dt><dd>Versión de regla, orden, candidatos, plantilla, recibos y actor</dd></div></dl>
          <div class="nota-pendiente">En producción el botón deberá revalidar elegibilidad y autorización en la misma operación, registrar el llamamiento y entregar una orden a conectores; no puede confiar en el estado del navegador.</div>
          <div class="acciones-paso"><button type="button" class="boton-secundario" data-accion="anterior-paso">← Volver</button><button type="button" class="boton-primario" data-accion="validar-recorrido">Validar recorrido</button></div>
        </div>
      </section>
      <aside class="resumen-lateral"><section class="panel"><div class="cabecera-panel"><h3>Control previo</h3></div><ul class="lista-comprobacion"><li>Bolsa y versión identificadas</li><li>Orden de prelación conservado</li><li>Selección enmascarada</li><li>Condiciones y plazo visibles</li><li class="pendiente">Persistencia PostgreSQL pendiente</li><li class="pendiente">Conector de notificación pendiente</li></ul></section></aside>
    </div>`;
}

function renderizarContratos() {
  const filas = DATOS_PANEL.contratos.map((item) => [item.expediente, item.bolsa, item.acto, item.inicio, item.fin, item.estado]);
  return vistaTablaGenerica("Relaciones y disponibilidad", "Contratos, ceses y reincorporaciones", "Cada movimiento conserva causa, fechas, efecto sobre disponibilidad y evidencia de aprobación.", ["Expediente", "Bolsa", "Acto", "Inicio", "Fin", "Estado"], filas, "El dominio de llamamientos contempla estados y elegibilidad; esta bandeja necesita todavía su repositorio y API internos.");
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

function renderizarEstadisticas() {
  const indicadores = DATOS_PANEL.indicadores;
  const llamamientosMes = Array.isArray(DATOS_PANEL.series.llamamientos_mes) ? DATOS_PANEL.series.llamamientos_mes : [];
  const distribucion = DATOS_PANEL.distribucion_global;
  const total = Object.values(distribucion).reduce((suma, valor) => suma + (Number(valor) || 0), 0);
  const segmentos = [distribucion.disponibles, distribucion.en_llamamiento, distribucion.contratados].map((valor) => total > 0 ? porcentajeSeguro((Number(valor) || 0) * 100 / total) : 0);
  return `
    ${encabezadoVista("Información agregada", "Estadísticas y explotación de datos", "Indicadores operativos anonimizados; cualquier exportación detallada exigirá permiso y finalidad.", '<button type="button" class="boton-secundario" data-accion="exportar">Preparar informe</button>')}
    <div class="rejilla-kpi">${tarjetaKPI("COB", `${numero(indicadores.cobertura_media, 1)} %`, "Cobertura media", "estadisticas")}${tarjetaKPI("TME", indicadores.tiempo_medio_cobertura, "Tiempo medio de cobertura", "estadisticas")}${tarjetaKPI("REN", `${numero(indicadores.renuncias_porcentaje, 1)} %`, "Renuncias justificadas", "estadisticas")}${tarjetaKPI("RES", `${numero(indicadores.respuesta_mediana_horas)} h`, "Respuesta mediana", "estadisticas")}${tarjetaKPI("VIG", numero(indicadores.bolsas_activas), "Bolsas activas", "elaboracion")}</div>
    <section class="panel"><div class="cabecera-panel"><h3>Evolución y cobertura</h3><span class="estado-chip info">Fuente del panel</span></div><div class="graficos-resumen"><article class="grafico-mini"><h4>Cobertura por categoría</h4><div class="barras-mini">${DATOS_PANEL.bolsas.slice(0, 5).map((bolsa) => `<span data-altura="${porcentajeSeguro(bolsa.cobertura)}"></span>`).join("")}</div><p class="texto-grafico">Bolsas visibles para el ámbito</p></article><article class="grafico-mini"><h4>Llamamientos por mes</h4><div class="linea-mini">${llamamientosMes.map((valor) => `<span data-altura="${porcentajeSeguro(valor)}"></span>`).join("")}</div><p class="texto-grafico">Serie aportada por la fuente</p></article><article class="grafico-mini"><h4>Resultado</h4><div class="grafico-anillo" data-anillo-a="${segmentos[0]}" data-anillo-b="${segmentos[1]}" data-anillo-c="${segmentos[2]}"></div><p class="texto-grafico">${numero(total)} personas en la distribución</p></article></div></section>`;
}

function renderizarDocumentos() {
  const filas = DATOS_PANEL.documentos.map((item) => [item.referencia, item.plantilla, item.formatos, item.version, item.estado]);
  return vistaTablaGenerica("Plantillas y formatos", "Generación y firma de documentos", "Salida gobernada por formato, plantilla, firmantes, CSV de cotejo, sello de tiempo y custodia.", ["Referencia", "Plantilla", "Formatos", "Versión", "Estado"], filas, "Los renderizadores PDF y DOCX y los contratos multiformato existen; la firma y publicación oficial de Bolsa todavía no están compuestas extremo a extremo.");
}

function renderizarComunicaciones() {
  const filas = DATOS_PANEL.canales.map((item) => [item.canal, item.uso, item.integracion, item.estado]);
  return vistaTablaGenerica("Conectores intercambiables", "Correo y mensajería", "Preparación, envío, recepción y acuse separados; ningún canal concede por sí solo efecto administrativo.", ["Canal", "Uso", "Integración", "Estado"], filas, "El envío y la lectura fallan de forma cerrada hasta disponer de autorización, persistencia y recibo verificable del conector.");
}

function renderizarAuditoria() {
  return `
    ${encabezadoVista("Hechos reconstruibles", "Auditoría y trazabilidad", "Actor, regla, versión, datos mínimos, decisión, recibos y resultado de cada actuación.", '<button type="button" class="boton-secundario" data-accion="exportar">Preparar paquete de auditoría</button>')}
    <section class="nota-seguridad">La vista de auditoría no modifica negocio. Los datos personales se minimizan y los permisos se evalúan por expediente y finalidad.</section>
    <div class="rejilla-dos-columnas">
      <section class="panel"><div class="cabecera-panel"><h3>Línea temporal del expediente ${escaparHTML(DATOS_PANEL.auditoria.expediente)}</h3><span class="estado-chip info">Fuente del panel</span></div><ol class="lista-trazabilidad">${DATOS_PANEL.actividad.map((item, indice) => `<li class="elemento-trazabilidad"><span class="detalle-lista"><strong>${indice + 1}. ${escaparHTML(item.accion)}</strong><span>${escaparHTML(item.objeto)} · ${escaparHTML(item.actor)} · ${escaparHTML(item.fecha)}</span></span></li>`).join("")}<li class="elemento-trazabilidad"><span class="detalle-lista"><strong>${numero(DATOS_PANEL.actividad.length + 1)}. Revisión de la propuesta</strong><span>Pendiente · no se ha enviado ninguna comunicación</span></span></li></ol></section>
      <aside class="resumen-lateral"><section class="panel"><div class="cabecera-panel"><h3>Evidencias previstas</h3></div><ul class="lista-comprobacion"><li>Identidad y rol autenticados</li><li>Autorización para el expediente</li><li>Versión de bolsa y reglas</li><li>Selección y orden aplicados</li><li>Plantillas y firmas</li><li class="pendiente">Recibos de entrega por conectar</li></ul></section></aside>
    </div>`;
}

function vistaTablaGenerica(sobrelinea, titulo, descripcion, cabeceras, filas, aviso) {
  return `
    ${encabezadoVista(sobrelinea, titulo, descripcion, '<button type="button" class="boton-secundario" data-accion="exportar">Preparar exportación</button>')}
    <section class="nota-pendiente">${escaparHTML(aviso)}</section>
    <section class="panel">
      <div class="cabecera-panel"><h3>${escaparHTML(titulo)}</h3><span class="estado-chip info">${escaparHTML(etiquetaFuentePanel())}</span></div>
      <div class="tabla-contenedor"><table class="tabla-datos"><caption>${escaparHTML(titulo)}</caption><thead><tr>${cabeceras.map((cabecera) => `<th scope="col">${escaparHTML(cabecera)}</th>`).join("")}</tr></thead><tbody>${filas.map((fila) => `<tr>${fila.map((celda, indice) => `<td>${indice === fila.length - 1 ? `<span class="estado-chip ${claseEstado(celda)}">${escaparHTML(celda)}</span>` : escaparHTML(celda)}</td>`).join("")}</tr>`).join("")}</tbody></table></div>
    </section>`;
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

const controlador = crearControladorPortal({
  anunciar, candidatosFiltrados, cargarFuenteDatos, escaparHTML, estado,
  etiquetaFuentePanel, navegar, notaOperacionNoCompuesta, numero,
  obtenerDatosPanel: () => DATOS_PANEL, porcentajeSeguro, porId, renderizar,
  resumenBolsaSeleccionada, vistaDesdeHash,
});

async function inicializar() {
  controlador.restaurarPreferencias();
  estado.vista = vistaDesdeHash();
  renderizar();
  controlador.instalar();
  await cargarFuenteDatos();
  renderizar();
}

document.addEventListener("DOMContentLoaded", inicializar, { once: true });
