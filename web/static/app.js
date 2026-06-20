const VEC_SHELL_API = "/api/vec";
const VEC_WORKSPACE_API = "/api/vec/workspace";
const VEC_SESSION_API = "/api/vec/session";
const CRONOS_LEAVE_API = "/api/vec/cronos/leave-requests";
const PERSONAL_RPT_POSITIONS_API = "/api/vec/personal/rpt/positions";
const PERSONAL_RPT_STATS_API = "/api/vec/personal/rpt/stats";
const PERSONAL_CATEGORIES_API = "/api/vec/personal/categories";
const PERSONAL_CATALOGS_API = "/api/vec/personal/catalogs";
const BOLSA_PORTAL_API = "/api/portal";
const ADMIN_STATUS_API = "/api/admin/status";
const ADMIN_CAPABILITIES_API = "/api/admin/capabilities";
const STAFF_HEADERS = {
  "Content-Type": "application/json",
  "X-Auth-Mechanism": "kerberos_ad",
  "X-Auth-Subject": "staff",
  "X-Auth-Roles": "tecnico_rrhh",
  "X-VEC-Auth-Mechanism": "kerberos_ad",
  "X-VEC-Subject": "staff",
  "X-VEC-Roles": "validator_l1",
};

function staffHeaders() {
  return { ...STAFF_HEADERS };
}

function candidateHeaders(candidateID = state.candidate.id) {
  return {
    "Content-Type": "application/json",
    "X-Auth-Mechanism": "clave",
    "X-Auth-Subject": String(candidateID || state.candidate.id).trim(),
    "X-Auth-Roles": "ciudadano",
    "X-VEC-Auth-Mechanism": "clave",
    "X-VEC-Subject": String(candidateID || state.candidate.id).trim(),
    "X-VEC-Roles": "candidate",
  };
}

const STAFF_JSON_HEADERS = staffHeaders();
const MODULE_ACTION_ENDPOINT = {
  personal: "personal",
  nominas: "nominas",
  cronos: "cronos",
  horarios: "horarios",
  permisos: "permisos",
  dietas: "dietas",
  rutas: "rutas",
  bolsa: "bolsa",
};

const TEXT = {
  module: {
    procedure_dashboard: "Tablero de procedimiento",
    candidate_management: "Gestion de candidaturas",
    dashboard: "Tablero VEC",
    personal: "Personal",
    nominas: "Nominas",
    cronos: "Cronos",
    permisos: "Permisos y vacaciones",
    dietas: "Dietas",
    rutas: "Mapa kilometraje",
    bolsa: "Bolsa",
    convocatorias: "Convocatorias",
    expediente: "Expediente",
    meritos: "Meritos",
    documentos: "Documentos",
    autobaremo: "Autobaremacion",
    alegaciones: "Alegaciones",
    notificaciones: "Notificaciones",
    perfil: "Perfil",
  },
  action: {
    review_procedure: "Revisar procedimiento",
    publish_listing: "Publicar listado",
    review_candidate: "Revisar candidatura",
    export_expediente: "Exportar expediente",
    review_autobaremo: "Revisar autobaremo",
    check_notifications: "Comprobar notificaciones",
    add_merit: "Anadir merito",
  },
};

const MODULES = [
  { id: "dashboard", label: "Tablero VEC", accent: "blue" },
  { id: "personal", label: "Personal", accent: "blue" },
  { id: "nominas", label: "Nominas", accent: "green" },
  { id: "cronos", label: "Cronos", accent: "indigo" },
  { id: "horarios", label: "Horarios y turnos", accent: "blue" },
  { id: "permisos", label: "Permisos y vacaciones", accent: "teal" },
  { id: "dietas", label: "Dietas", accent: "orange" },
  { id: "rutas", label: "Mapa kilometraje", accent: "cyan" },
  { id: "bolsa", label: "Bolsa", accent: "violet" },
  { id: "solicitudes", label: "Solicitudes", accent: "teal" },
  { id: "meritos", label: "Meritos y RUM", accent: "teal" },
  { id: "autobaremo", label: "Autobaremacion", accent: "violet" },
  { id: "documentos", label: "Documentos", accent: "cyan" },
  { id: "alegaciones", label: "Alegaciones", accent: "orange" },
  { id: "notificaciones", label: "Notificaciones", accent: "amber" },
  { id: "listados", label: "Listados", accent: "green" },
  { id: "manifiestos", label: "Manifiestos", accent: "indigo" },
  { id: "aprobaciones", label: "Aprobaciones", accent: "green" },
  { id: "auditoria", label: "Auditoria", accent: "slate" },
  { id: "administracion", label: "Administracion", accent: "slate" },
];

const MODULE_COPY = {
  dashboard: ["Bandeja VEC unificada", "Fichajes, permisos, dietas y expedientes en una cola comun"],
  personal: ["Modulo Personal", "Empleado, puesto, situacion administrativa, antiguedad y certificados"],
  nominas: ["Nominas y retribuciones", "Incidencias retributivas, trienios, reducciones y cierre mensual"],
  cronos: ["Modulo Cronos", "Fichajes, incidencias, asuntos propios y vacaciones"],
  horarios: ["Horarios del personal", "Flexibilidad, turnos fijos, cobertura obligatoria y reducciones 63/64"],
  permisos: ["Permisos y vacaciones", "Saldos, solapes, ausencias y aprobaciones"],
  dietas: ["Modulo Dietas", "Comisiones de servicio, gastos, medias dietas y dietas completas"],
  rutas: ["Mapa provincial de kilometraje", "Rutas por municipio, kilometros estimados y politica aplicable"],
  bolsa: ["Modulo Bolsa", "Seleccion, solicitudes, meritos y listados como modulo VEC"],
  solicitudes: ["Solicitudes Bolsa", "Alta, borrador, presentacion y seguimiento de candidaturas"],
  meritos: ["Meritos y RUM", "Inventario de titulos, cursos, experiencia y evidencias reutilizables"],
  autobaremo: ["Autobaremacion", "Simulacion de puntos, desglose y recibo de calculo"],
  documentos: ["Documentos", "Justificantes, CSV, ENI, firmas y evidencias"],
  alegaciones: ["Alegaciones", "Reclamaciones, subsanaciones y resoluciones de Bolsa"],
  notificaciones: ["Notificaciones", "Avisos legales, plazos y comunicaciones"],
  listados: ["Listados", "Provisional, definitivo, ranking y exportacion"],
  manifiestos: ["Manifiestos", "Contrato de enganche, capacidades y rutas del modulo"],
  aprobaciones: ["Aprobaciones", "Bandeja de responsables para permisos, dietas y expedientes"],
  auditoria: ["Auditoria", "Trazabilidad de cambios, actores y recibos"],
  administracion: ["Administracion", "Configuracion, colas y supervision"],
};

const DEFAULT_SCREEN_FIELDS = ["Referencia", "Estado", "Responsable", "Plazo", "Documento", "Accion"];
const DEFAULT_SCREEN_ACTIONS = ["Abrir", "Exportar"];
const DEFAULT_SCREEN_STATES = ["Pendiente", "En revision", "Validado", "Bloqueado", "Cerrado"];
const DEFAULT_SCREEN_INTEGRATIONS = ["Documentos", "Auditoria"];
const DEFAULT_SCREEN_VALIDATIONS = ["Permiso de acceso", "Datos obligatorios", "Recibo de auditoria"];

const FALLBACK_SCREEN_BLUEPRINTS = MODULES
  .filter((module) => module.id !== "dashboard")
  .map((module) => screen(
    `${module.id}.workspace`,
    module.id,
    `${module.label}: bandeja`,
    MODULE_COPY[module.id]?.[1] || "Pantalla operativa del modulo.",
    DEFAULT_SCREEN_FIELDS,
    DEFAULT_SCREEN_ACTIONS,
    DEFAULT_SCREEN_STATES,
    DEFAULT_SCREEN_INTEGRATIONS,
    DEFAULT_SCREEN_VALIDATIONS,
    "Estado persistido con recibo y auditoria visible.",
  ));

function screen(id, moduleKey, title, description, fields, actions, states, integrations, validations, doneCriteria) {
  return {
    id,
    module_key: moduleKey,
    menu_id: id,
    title,
    description,
    fields: fields.map((field) => ({ key: slugify(field), label: field, type: "text", required: true })),
    actions,
    states,
    integrations,
    validations,
    done_criteria: doneCriteria,
  };
}

function slugify(value) {
  return String(value || "")
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "_")
    .replace(/^_+|_+$/g, "") || "campo";
}

const FLOW_TEXT = {
  dashboardTitle: "Operacion pendiente",
  dashboardAction: "Ir a la siguiente accion",
  convocatoriaAction: "Recargar y publicar listados demo",
  solicitudAction: "Crear/actualizar candidato",
  meritAction: "Guardar merito o titulo",
  baremoAction: "Calcular autobaremacion",
  expedienteAction: "Exportar expediente",
  documentAction: "Registrar documento",
  claimAction: "Registrar alegacion",
  notificationAction: "Crear aviso",
  notificationSyncAction: "Sincronizar avisos",
  notificationSendAction: "Cambiar aviso a enviado",
  notificationReadAction: "Marcar aviso leido",
  listingAction: "Exportar listados",
  documentSyncAction: "Sincronizar documentos",
  claimSyncAction: "Sincronizar alegaciones",
  auditAction: "Consultar auditoria",
  auditExportAction: "Exportar auditoria",
  auditCandidateField: "Candidato",
  auditEmptyHint: "Consulta el endpoint de auditoria o ejecuta un flujo",
  auditReceipt: "Auditoria consultada",
  manifestAction: "Cargar manifiesto Bolsa",
  manifestHealthAction: "Comprobar salud del modulo",
  manifestTitle: "Manifiesto Bolsa",
  manifestModule: "Modulo",
  manifestVersion: "Version",
  manifestPrototypeAPI: "API prototipo",
  manifestHTTPRoutes: "Rutas HTTP",
  manifestCapabilities: "Capacidades",
  manifestLoadedReceipt: "Manifiesto Bolsa cargado",
  manifestHealthReceipt: "Salud modulo Bolsa comprobada",
  adminAction: "Comprobar salud",
  adminStatusAction: "Comprobar adaptadores",
  adminCapabilitiesAction: "Consultar capacidades",
  adminAdaptersTitle: "Adaptadores desde API",
  adminStorageAdapter: "Storage",
  adminAuthAdapter: "Auth",
  adminExternalAdapters: "Externos legales",
  adminRoutesTitle: "Rutas administrativas",
  adminStatusTitle: "Estado operativo API",
  adminNoExternalAdapters: "Sin adaptadores externos configurados",
  receiptTitle: "Recibos y estado",
  noReceipts: "Sin recibos todavia",
  apiReal: "API local",
  localFlow: "Flujo local trazable",
  demoFlow: "API demo",
};

const MODULE_ENDPOINTS = {
  solicitudes: [{ method: "POST", route: "/api/candidates" }],
  meritos: [{ method: "POST", route: "/api/candidates/{id}/merits" }],
  autobaremo: [
    { method: "POST", route: "/api/candidates/{id}/baremo" },
    { method: "GET", route: "/api/candidates/{id}/expediente" },
  ],
  documentos: [
    { method: "GET", route: "/api/candidates/{id}/documents" },
    { method: "POST", route: "/api/candidates/{id}/documents" },
  ],
  alegaciones: [
    { method: "GET", route: "/api/candidates/{id}/claims" },
    { method: "POST", route: "/api/candidates/{id}/claims" },
  ],
  notificaciones: [
    { method: "GET", route: "/api/notifications?candidate_id={id}" },
    { method: "POST", route: "/api/notifications" },
    { method: "POST", route: "/api/notifications/{id}/send" },
    { method: "POST", route: "/api/notifications/{id}/read" },
    { method: "GET", route: "/api/candidates/{id}/notifications" },
    { method: "POST", route: "/api/candidates/{id}/notifications" },
  ],
  auditoria: [
    { method: "GET", route: "/api/audit?candidate_id={id}" },
    { method: "GET", route: "/api/candidates/{id}/audit" },
  ],
  administracion: [
    { method: "GET", route: "/healthz" },
    { method: "GET", route: ADMIN_STATUS_API },
    { method: "GET", route: ADMIN_CAPABILITIES_API },
  ],
  manifiestos: [
    { method: "GET", route: "/api/modules/bolsa" },
    { method: "GET", route: "/api/modules/bolsa/healthz" },
  ],
};

const $ = (selector, root = document) => root.querySelector(selector);
const $$ = (selector, root = document) => Array.from(root.querySelectorAll(selector));
const statusNode = $("#connection-status");
const reloadButton = $("#reload-demo");
let localeCatalog = {};
const state = {
  portal: null,
  workspace: null,
  session: null,
  personalCatalog: null,
  demo: null,
  rows: [],
  localRows: [],
  rowOverrides: {},
  selectedRowID: "",
  activeModule: "dashboard",
  activeScreen: "",
  screenStateFilter: "",
  leaveSubmitting: false,
  rptSubmitting: false,
  categorySubmitting: false,
  activeTab: "resumen",
  search: "",
  filters: { scope: "", state: "", deadline: "", unit: "" },
  actionLog: [],
  auditEntries: [],
  apiRoutes: [],
  moduleManifest: null,
  adminStatus: null,
  adminCapabilities: null,
  candidate: {
    id: "cand-ui-1",
    dni: "12345678A",
    nombre: "Persona Demo",
    email: "persona.demo@example.test",
    call_id: "convocatoria-default",
  },
  merits: [],
  baremo: null,
  expediente: null,
  documents: [
    { id: "doc-001", title: "Titulo formacion", csv: "CSV-GR-2026-8841", state: "Pendiente" },
    { id: "doc-002", title: "Servicios prestados", csv: "CSV-GR-2026-8842", state: "Metadatos presentes" },
  ],
  claims: [
    { id: "aleg-001", subject: "Revision de puntuacion", state: "Abierta", detail: "Pendiente de informe" },
  ],
  notifications: [
    { id: "notif-001", title: "Subsanacion pendiente", state: "Pendiente", deadline: "72 h" },
    { id: "notif-002", title: "Listado provisional publicado", state: "Leida", deadline: "Sin vencimiento critico" },
  ],
  admin: { autosave: true, csvCheck: "simulado", activeProfile: "personal_interno" },
};

function label(key) {
  if (!key) return "-";
  const id = key.split(".").pop();
  return localeCatalog[key] || TEXT.module[id] || TEXT.action[id] || String(key).replaceAll("_", " ");
}

function ui(key, fallback) {
  return localeCatalog[`ui.portal.${key}`] || FLOW_TEXT[key] || fallback || key;
}

function setStatus(message, status = "idle") {
  if (!statusNode) return;
  statusNode.textContent = message;
  statusNode.dataset.state = status;
}

function setText(selector, value) {
  const node = $(selector);
  if (node) node.textContent = value || "-";
}

function formatPoints(value) {
  return new Intl.NumberFormat("es-ES", {
    minimumFractionDigits: 1,
    maximumFractionDigits: 1,
  }).format(Number(value || 0));
}

function formatCount(value) {
  return new Intl.NumberFormat("es-ES").format(Number(value || 0));
}

async function getData(url, options) {
  const response = await fetch(url, options);
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(payload.error || payload.message || `No se pudo cargar ${url}.`);
  return payload.data || payload;
}

async function sendJSON(url, method, body, headers = candidateHeaders()) {
  const options = { method, headers };
  if (body !== undefined && body !== null) options.body = JSON.stringify(body);
  return getData(url, options);
}

async function loadAPIRootRoutes() {
  try {
    const root = await getData("/api", { method: "GET", headers: staffHeaders() });
    state.apiRoutes = Array.isArray(root.routes) ? root.routes : [];
  } catch (error) {
    state.apiRoutes = [];
  }
}

async function loadModuleManifest() {
  state.moduleManifest = await getData("/api/modules/bolsa", { method: "GET", headers: staffHeaders() });
  return state.moduleManifest;
}

async function loadAdminStatus() {
  state.adminStatus = await getData(ADMIN_STATUS_API, { method: "GET", headers: staffHeaders() });
  return state.adminStatus;
}

async function loadAdminCapabilities() {
  state.adminCapabilities = await getData(ADMIN_CAPABILITIES_API, { method: "GET", headers: staffHeaders() });
  return state.adminCapabilities;
}

async function loadLocale() {
  try {
    localeCatalog = await getData("/locales/es.json", { method: "GET" });
  } catch (error) {
    localeCatalog = {};
  }
}

async function loadDemoData() {
  const response = await fetch("/api/demo", { method: "POST", headers: staffHeaders() });
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) return null;
  return payload.data || null;
}

function declaredRoutes() {
  const manifestRoutes = (state.moduleManifest?.http_routes || []).map((route) => route.route);
  const capabilityRoutes = (state.moduleManifest?.capabilities || []).map((capability) => capability.route).filter(Boolean);
  const adminRoutes = (state.adminStatus?.admin_routes || state.adminCapabilities?.http_routes || []).map((route) => route.route);
  return new Set([...state.apiRoutes, ...manifestRoutes, ...capabilityRoutes, ...adminRoutes, "/healthz"]);
}

function isEndpointAvailable(routeTemplate) {
  return declaredRoutes().has(routeTemplate);
}

function moduleTransport(moduleID) {
  if (moduleID === "convocatorias") return { key: "demoFlow", className: "endpoint-demo" };
  if ((MODULE_ENDPOINTS[moduleID] || []).some((endpoint) => isEndpointAvailable(endpoint.route))) {
    return { key: "apiReal", className: "endpoint-real" };
  }
  return { key: "localFlow", className: "endpoint-local" };
}

function endpointPath(routeTemplate, candidateID = state.candidate.id) {
  return routeTemplate.replace("{id}", encodeURIComponent(candidateID));
}

function notificationActionPath(notificationID, action) {
  return `/api/notifications/${encodeURIComponent(notificationID)}/${action}`;
}

function normalizePersonalCatalog(raw) {
  const positions = raw.positions?.positions || raw.positions || {};
  const categories = raw.categories?.categories || raw.categories || {};
  return {
    positions: {
      items: Array.isArray(positions.items) ? positions.items : [],
      total: Number(positions.total || 0),
      limit: Number(positions.limit || 0),
      offset: Number(positions.offset || 0),
    },
    stats: raw.stats?.stats || raw.stats || {},
    categories: {
      items: Array.isArray(categories.items) ? categories.items : [],
      total: Number(categories.total || 0),
      limit: Number(categories.limit || 0),
      offset: Number(categories.offset || 0),
    },
    catalogs: Array.isArray(raw.catalogs?.catalogs) ? raw.catalogs.catalogs : (Array.isArray(raw.catalogs) ? raw.catalogs : []),
  };
}

function normalizePortal(rawPortal, demo) {
  const sections = Array.isArray(rawPortal.sections) ? rawPortal.sections : [];
  const routes = Array.isArray(rawPortal.routes) ? rawPortal.routes : [];
  const summary = Array.isArray(rawPortal.summary) ? rawPortal.summary : [];
  const modules = Array.isArray(rawPortal.modules) && rawPortal.modules.length
    ? rawPortal.modules
    : sections.map((section, index) => ({
        id: section.key || `section_${index + 1}`,
        label_key: section.key,
        description_key: (section.routes || []).join(", "),
        count: (section.routes || []).length,
        alert_count: (section.actions || []).length,
        accent: index === 0 ? "indigo" : "blue",
        actions: section.actions || [],
      }));
  const definitiveItems = demo?.definitivo?.items || [];
  const provisionalItems = demo?.provisional?.items || [];
  return {
    principal: rawPortal.principal || {},
    vecModules: rawPortal.vec_modules || [],
    workspace: rawPortal.workspace || null,
    personalCatalog: rawPortal.personal_catalog || null,
    routes,
    sections,
    modules,
    summary,
    pendingActions: rawPortal.pending_actions || sections.flatMap((section) => section.actions || []),
    autobaremo: rawPortal.autobaremo || null,
    audit: rawPortal.audit || null,
    call: demo?.convocatoria || { id: rawPortal.header?.call_id || "portal-profesional", estado: "operativo" },
    provisional: demo?.provisional || { version: "pendiente", items: provisionalItems },
    definitive: demo?.definitivo || { version: "pendiente", items: definitiveItems },
  };
}

function renderKPIs(view) {
  const kpis = moduleKPIs(view, state.activeModule);
  const labels = $$(".metrics article .metric-label");
  const values = $$(".metrics article strong");
  const notes = $$(".metrics article .metric-note");
  kpis.forEach((kpi, index) => {
    if (labels[index]) labels[index].textContent = kpi.label;
    if (values[index]) values[index].textContent = String(kpi.value);
    if (notes[index]) notes[index].textContent = kpi.note || "-";
  });
}

function moduleKPIs(view, moduleID) {
  const rows = state.rows.length ? state.rows : rowsFromPortal(view);
  const byModule = (id) => rows.filter((row) => row.modules.includes(id));
  const workspaceKPIs = view.workspace?.kpis || [];
  const cronos = view.workspace?.cronos_daily_summary || {};
  const permissions = view.workspace?.cronos_permission_balances || [];
  const routes = view.workspace?.province_routes || [];
  const catalog = getPersonalCatalog(view);
  const catalogStats = catalog.stats || {};
  const findPermission = (name) => permissions.find((item) => String(item.name || "").includes(name)) || {};
  const byScope = (pattern) => rows.filter((row) => pattern.test(row.scope || row.expediente || row.state || ""));
  const defaults = [
    { label: workspaceKPIs[0]?.label || "Personas activas", value: workspaceKPIs[0]?.value || 0, note: workspaceKPIs[0]?.note || "Puestos, situaciones y nomina" },
    { label: workspaceKPIs[1]?.label || "Perfiles horarios", value: workspaceKPIs[1]?.value || 0, note: workspaceKPIs[1]?.note || "Flexibles y turnos fijos" },
    { label: workspaceKPIs[2]?.label || "Dietas pendientes", value: workspaceKPIs[2]?.value || 0, note: workspaceKPIs[2]?.note || "Medias y completas" },
    { label: workspaceKPIs[3]?.label || "Km provincia", value: workspaceKPIs[3]?.value || "-", note: workspaceKPIs[3]?.note || "Rutas estimadas" },
  ];
  const moduleRows = byModule(moduleID);
  switch (moduleID) {
    case "dashboard":
      return defaults;
    case "personal":
      return [
        { label: "Expedientes RRHH", value: byModule("personal").length, note: "Empleado, puesto y situacion" },
        { label: "Puestos RPT", value: catalogStats.positions || byScope(/puesto/i).length, note: "API Personal/RPT" },
        { label: "Categorias", value: catalogStats.categories || 0, note: "Maestro Bolsa/OPES" },
        { label: "Codigos pendientes", value: catalogStats.pending_legend || 0, note: "Leyenda RPT por validar" },
      ];
    case "nominas":
      return [
        { label: "Incidencias nomina", value: moduleRows.length, note: "Cruces con Cronos/Personal" },
        { label: "Cierre", value: "25/06", note: "Periodo ordinario de junio" },
        { label: "Trienios", value: byScope(/Antiguedad|trienios/i).length, note: "Calculo automatico" },
        { label: "Certificados", value: byScope(/CERT-|Certificados/i).length, note: "Servicios prestados" },
      ];
    case "cronos":
      return [
        { label: "Jornada teorica", value: cronos.theoretical || "07:30", note: "Dia seleccionado" },
        { label: "Trabajadas", value: cronos.worked || "00:00", note: "Movimientos del dia" },
        { label: "Teletrabajo", value: cronos.telework || "07:30", note: "Computo informado" },
        { label: "Defecto mes", value: cronos.period_balance || "-04:34", note: "Acumulado periodo" },
      ];
    case "horarios":
      return [
        { label: "Perfiles", value: view.workspace?.schedule_profiles?.length || 0, note: "Por puesto/unidad" },
        { label: "Flexibles", value: (view.workspace?.schedule_profiles || []).filter((item) => item.flexible).length, note: "Con tramo obligatorio" },
        { label: "Sin flexibilidad", value: byScope(/Sin flexibilidad|personas mayores/i).length, note: "Cobertura presencial" },
        { label: "Reducciones 63/64", value: byScope(/Prejubilacion|63|64/i).length, note: "1h/2h menos diaria" },
      ];
    case "permisos":
      return [
        { label: "Asuntos propios", value: findPermission("ASUNTOS").remaining || "-", note: `Max. ${findPermission("ASUNTOS").max || "-"}` },
        { label: "Vacaciones", value: findPermission("VACACIONES").remaining || "-", note: `Max. ${findPermission("VACACIONES").max || "-"}` },
        { label: "Bolsa conciliacion", value: findPermission("CONCILIACION").remaining || "-", note: `Max. ${findPermission("CONCILIACION").max || "-"}` },
        { label: "Solicitudes", value: moduleRows.length, note: "Pendientes y aprobadas" },
      ];
    case "dietas":
      return [
        { label: "Comisiones", value: moduleRows.length, note: "Pendientes en bandeja" },
        { label: "Medias dietas", value: moduleRows.filter((row) => /media/i.test(row.document)).length, note: "Segun horario/ruta" },
        { label: "Dietas completas", value: moduleRows.filter((row) => /completa/i.test(row.document)).length, note: "Jornada completa" },
        { label: "Liquidaciones", value: moduleRows.filter((row) => /Liquidar/i.test(row.action)).length, note: "Listas para cierre" },
      ];
    case "rutas":
      return [
        { label: "Rutas", value: routes.length, note: "Mapa provincial demo" },
        { label: "Km max.", value: routes.reduce((max, route) => Math.max(max, Number(route.km_one_way || 0)), 0).toFixed(1), note: "Un trayecto" },
        { label: "Con media dieta", value: routes.filter((route) => /media/i.test(route.allowance || "")).length, note: "Segun politica" },
        { label: "Tiempo max.", value: `${routes.reduce((max, route) => Math.max(max, Number(route.estimated_minutes || 0)), 0)} min`, note: "Estimacion ruta" },
      ];
    case "bolsa":
      return [
        { label: "Registros Bolsa", value: moduleRows.length, note: "Modulo independiente" },
        { label: "Categorias", value: catalogStats.categories || 0, note: "Desde Personal/RPT" },
        { label: "Subsanaciones", value: moduleRows.filter((row) => /subsan/i.test(row.state)).length, note: "Pendientes" },
        { label: "Baremo categoria", value: (view.workspace?.bolsa_category_rules || []).length, note: "Misma/otra categoria" },
      ];
    default:
      return [
        { label: "Registros", value: moduleRows.length || rows.length, note: MODULE_COPY[moduleID]?.[1] || "Modulo VEC" },
        { label: "Pendientes", value: moduleRows.filter((row) => /pendiente|validar|revisar/i.test(`${row.state} ${row.action}`)).length, note: "Requieren accion" },
        { label: "Documentos", value: moduleRows.reduce((sum, row) => sum + row.documents.length, 0), note: "Evidencias vinculadas" },
        { label: "Auditoria", value: moduleRows.reduce((sum, row) => sum + row.timeline.length, 0), note: "Trazas visibles" },
      ];
  }
}

function moduleCount(view, moduleID) {
  const rows = state.rows.length ? state.rows : rowsFromPortal(view);
  switch (moduleID) {
    case "dashboard":
      return rows.length;
    case "personal":
      return rows.filter((row) => row.modules.includes("personal")).length || getPersonalCatalog(view).stats?.positions || 0;
    case "nominas":
      return rows.filter((row) => row.modules.includes("nominas")).length;
    case "cronos":
      return rows.filter((row) => row.modules.includes("cronos")).length;
    case "horarios":
      return rows.filter((row) => row.modules.includes("horarios")).length || view.workspace?.schedule_profiles?.length || 0;
    case "permisos":
      return rows.filter((row) => row.modules.includes("permisos")).length;
    case "dietas":
      return rows.filter((row) => row.modules.includes("dietas")).length;
    case "rutas":
      return view.workspace?.province_routes?.length || rows.filter((row) => row.modules.includes("rutas")).length;
    case "bolsa":
      return rows.filter((row) => row.modules.includes("bolsa")).length || getPersonalCatalog(view).stats?.categories || 0;
    case "documentos":
      return rows.reduce((sum, row) => sum + row.documents.length, 0);
    case "aprobaciones":
      return rows.filter((row) => row.modules.includes("aprobaciones")).length || 1;
    case "notificaciones":
      return view.pendingActions.length || 1;
    case "listados":
      return (view.provisional.items || []).length + (view.definitive.items || []).length;
    case "auditoria":
      return rows.reduce((sum, row) => sum + row.timeline.length, 0) + state.actionLog.length;
    case "administracion":
      return view.routes.length || 1;
    default:
      return 0;
  }
}

function renderModules(view) {
  const buttons = $$(".module-link");
  buttons.forEach((button, index) => {
    const module = MODULES[index];
    if (!module) return;
    const name = $("span:first-child", button);
    const count = $(".module-count", button);
    name.textContent = module.label;
    count.textContent = formatCount(moduleCount(view, module.id));
    button.dataset.accent = module.accent || "slate";
    button.dataset.module = module.id;
    button.setAttribute("aria-current", state.activeModule === module.id ? "page" : "false");
  });
}

function rowFromItem(item, index, kind) {
  const definitive = kind === "definitive";
  const estado = item.estado || (definitive ? "AdmitidaDefinitiva" : "AdmitidaProvisional");
  const candidate = item.candidate_id || "personal-interno";
  const expediente = item.solicitud_id || `SOL-PORTAL-${String(index + 1).padStart(4, "0")}`;
  return {
    id: `${kind}-${expediente}`,
    expediente,
    candidate,
    state: estado,
    stateFilter: definitive ? "Definitivo publicado" : (index === 0 ? "Pendiente de accion" : "En revision"),
    deadline: definitive ? "Sin accion critica" : (index === 0 ? "Vence en 72 h" : "Seguimiento ordinario"),
    deadlineBucket: definitive ? "Sin vencimiento critico" : (index === 0 ? "Vence en 72 h" : "Sin vencimiento critico"),
    points: item.total_points == null ? "-" : `${formatPoints(item.total_points)} pt`,
    document: definitive ? "Resolucion definitiva" : "Evidencia provisional",
    action: definitive ? "Abrir" : (index === 0 ? "Revisar" : "Validar"),
    scope: "Modulo Bolsa",
    unit: definitive ? "Personal temporal" : "Tribunal baremacion",
    modules: ["dashboard", "bolsa", definitive ? "documentos" : "bolsa"],
    documents: [
      ["Solicitud registrada", `${expediente} - registro asociado al procedimiento`],
      [definitive ? "Resolucion definitiva" : "Listado provisional", `${kind} - version ${item.version || "v1"}`],
      ["Expediente electronico", `${candidate} - CSV/ENI pendiente de verificacion real`],
    ],
    merits: [
      ["Puntuacion servida por API", item.total_points == null ? "Sin puntuacion publicada" : `${formatPoints(item.total_points)} pt`],
      ["Desglose", "Abrir autobaremacion o expediente para ver calculo por backend"],
      ["Reglas", "Catalogo declarado por la API profesional"],
    ],
    alerts: definitive
      ? [["Listado definitivo publicado", "Estado informativo sin accion critica"]]
      : [["Revision provisional pendiente", "Comprobar evidencias antes de publicar definitivo"]],
    timeline: [
      [definitive ? "Definitivo publicado" : "Provisional calculado", `${new Date().toLocaleString("es-ES")} - motor de demo`],
      ["Autobaremo calculado", `${formatPoints(item.total_points)} pt - reglas v1`],
      ["Solicitud importada", `${candidate} - ${expediente}`],
    ],
  };
}

function workspaceRowFromRecord(record, index) {
  const moduleID = String(record.module_id || "");
  const scope = String(record.scope || record.module || "VEC");
  const modules = ["dashboard", "auditoria", "aprobaciones"];
  if (moduleID.includes("personal")) {
    modules.push("personal");
    if (/nomina|retributiva|trienio|reduccion/i.test(`${scope} ${record.title || ""}`)) modules.push("nominas");
    if (/certificado|servicios prestados|antiguedad/i.test(`${scope} ${record.title || ""}`)) modules.push("documentos");
  }
  if (moduleID.includes("cronos")) {
    modules.push("cronos");
    if (/horario|turno|prejubilacion|63|64/i.test(`${scope} ${record.title || ""}`)) modules.push("horarios");
    if (/permiso|vacacion|asuntos/i.test(`${scope} ${record.title || ""}`)) modules.push("permisos");
  }
  if (moduleID.includes("dietas")) {
    modules.push("dietas");
    if (/ruta|km|kilometraje|mapa/i.test(`${scope} ${record.title || ""}`)) modules.push("rutas");
  }
  if (moduleID.includes("bolsa")) {
    modules.push("bolsa", "documentos");
  }
  const stateText = record.state || "Pendiente";
  const deadline = record.deadline || "Sin vencimiento critico";
  const title = record.title || record.id || `VEC-${index + 1}`;
  return {
    id: record.id || `workspace-${index + 1}`,
    expediente: record.id || title,
    candidate: record.employee || record.module || "Personal interno",
    state: stateText,
    stateFilter: /pendiente|solape|excedida|sin flexibilidad/i.test(stateText) ? "Pendiente de accion" : "En revision",
    deadline,
    deadlineBucket: /vencido/i.test(deadline) ? "Plazo vencido" : /72|hoy/i.test(deadline) ? "Vence en 72 h" : "Sin vencimiento critico",
    points: record.metric || "-",
    document: record.document || scope,
    action: record.action || "Abrir",
    scope,
    unit: record.module || "VEC",
    modules,
    documents: [
      [record.module || "Modulo", scope],
      ["Referencia", title],
      ["Soporte", record.document || "Pendiente de justificante"],
    ],
    merits: [
      ["Magnitud", record.metric || "-"],
      ["Perfil", scope],
      ["Politica", policyHint(record)],
    ],
    alerts: [
      [stateText, title],
      ["Control VEC", "Registro integrado con auditoria, permisos y aprobaciones"],
    ],
    timeline: [
      ["Registro cargado", `${record.id || title} - workspace VEC`],
      ["Modulo", `${record.module || "VEC"} - ${scope}`],
    ],
  };
}

function policyHint(record) {
  const text = `${record.scope || ""} ${record.title || ""} ${record.state || ""}`.toLowerCase();
  if (text.includes("personas mayores") || text.includes("sin flexibilidad")) {
    return "Puesto con cobertura presencial; no aplicar flexibilidad sin autorizacion";
  }
  if (text.includes("63")) return "Reduccion diaria configurable: 1 hora menos";
  if (text.includes("64")) return "Reduccion diaria configurable: 2 horas menos";
  if (text.includes("dieta")) return "Validar horario, ruta, kilometros y tipo de dieta";
  return "Validacion pendiente segun perfil del puesto";
}

function rowsFromPortal(view) {
  const workspaceRows = (view.workspace?.operational_records || []).map(workspaceRowFromRecord);
  const listingRows = [
    ...(view.provisional.items || []).map((item, index) => rowFromItem(item, index, "provisional")),
    ...(view.definitive.items || []).map((item, index) => rowFromItem(item, index, "definitive")),
  ];
  const sectionRows = view.sections.map((section, index) => ({
    id: `section-${section.key || index}`,
    expediente: `PORTAL-${String(index + 1).padStart(3, "0")}`,
    candidate: view.principal.subject || "staff",
    state: label(section.key),
    stateFilter: "En revision",
    deadline: `${(section.routes || []).length} rutas API`,
    deadlineBucket: "Sin vencimiento critico",
    points: `${(section.actions || []).length} acciones`,
    document: (section.routes || [BOLSA_PORTAL_API]).join(", "),
    action: label((section.actions || [])[0]) || "Abrir",
    scope: "Convocatorias abiertas",
    unit: index === 0 ? "Personal temporal" : "Registro y documentos",
    modules: ["dashboard", index === 0 ? "bolsa" : "administracion", "auditoria"],
    documents: (section.routes || [BOLSA_PORTAL_API]).map((route) => [route, "Ruta declarada por API profesional"]),
    merits: [["Modulo", label(section.key)], ["Acciones", (section.actions || []).map(label).join(", ") || "Sin acciones"]],
    alerts: (section.actions || []).map((action) => [label(action), "Accion pendiente desde portal"]),
    timeline: [
      ["Modulo cargado", `${label(section.key)} - ${new Date().toLocaleString("es-ES")}`],
      ["Rutas disponibles", (section.routes || [BOLSA_PORTAL_API]).join(", ")],
    ],
  }));
  const administrativeRows = [
    {
      id: "admin-autobaremo",
      expediente: "AUTOBAREMO-RESUMEN",
      candidate: view.principal.subject || "staff",
      state: "Simulacion disponible",
      stateFilter: "En revision",
      deadline: "Recalculo inmediato",
      deadlineBucket: "Sin vencimiento critico",
      points: `${formatPoints(view.autobaremo?.total_points || 0)} pt`,
      document: `Reglas ${view.autobaremo?.rule_set_version || "v1"}`,
      action: "Revisar autobaremo",
      scope: "Expediente candidato",
      unit: "Tribunal baremacion",
      modules: ["dashboard", "bolsa", "documentos"],
      documents: [["Recibo de calculo", "Autobaremo calculado con reglas vigentes"], ["Detalle de reglas", view.autobaremo?.rule_set_version || "v1"]],
      merits: (view.autobaremo?.sections || []).map((section) => [label(section.id || section.section), `${formatPoints(section.applied_points || section.points)} pt`]),
      alerts: (view.autobaremo?.warnings || []).map((warning) => [label(warning.message_key || warning.code), warning.severity || "Aviso"]),
      timeline: [["Autobaremo consultado", `${new Date().toLocaleString("es-ES")} - ${view.principal.subject || "staff"}`]],
    },
    {
      id: "admin-alegaciones",
      expediente: "ALEGA-2026-COLA",
      candidate: "Unidad de alegaciones",
      state: "Alegaciones en revision",
      stateFilter: "Pendiente de accion",
      deadline: "Plazo vencido",
      deadlineBucket: "Plazo vencido",
      points: "5 casos",
      document: "Resoluciones borrador",
      action: "Resolver",
      scope: "Modulo Bolsa",
      unit: "Personal temporal",
      modules: ["dashboard", "bolsa", "aprobaciones"],
      documents: [["Escrito de alegacion", "Pendiente de propuesta"], ["Resolucion", "Borrador preparado"]],
      merits: [["Objeto", "Discrepancia de puntuacion"], ["Estado", "Pendiente de resolver"]],
      alerts: [["Plazo activo", "Priorizar antes de cierre de audiencia"]],
      timeline: [["Alegacion recibida", "Registro electronico demo"], ["Asignada a unidad", "Personal temporal"]],
    },
    {
      id: "admin-documentos",
      expediente: "DOCS-CSV-ENI",
      candidate: "Registro y documentos",
      state: "Validacion documental",
      stateFilter: "En revision",
      deadline: "Seguimiento ordinario",
      deadlineBucket: "Sin vencimiento critico",
      points: "18 items",
      document: "CSV/ENI",
      action: "Validar",
      scope: "Expediente candidato",
      unit: "Registro y documentos",
      modules: ["dashboard", "documentos", "administracion"],
      documents: [["CSV", "Comprobacion simulada"], ["Firma", "Validacion pendiente de servicio externo"], ["ENI", "Metadatos presentes"]],
      merits: [["Documento", "Evidencia reutilizable en RUM"]],
      alerts: [["Servicio externo pendiente", "La UI no afirma validez juridica real"]],
      timeline: [["Documentos listados", `${new Date().toLocaleString("es-ES")} - API demo`]],
    },
    {
      id: "admin-notificaciones",
      expediente: "NOTIF-LEGAL",
      candidate: "Buzon profesional",
      state: "Avisos pendientes",
      stateFilter: "Pendiente de accion",
      deadline: "Vence en 72 h",
      deadlineBucket: "Vence en 72 h",
      points: `${view.pendingActions.length || 1} avisos`,
      document: "Comunicaciones",
      action: "Comprobar notificaciones",
      scope: "Convocatorias abiertas",
      unit: "Personal temporal",
      modules: ["dashboard", "notificaciones"],
      documents: [["Aviso legal", "Pendiente de lectura"], ["Recordatorio", "Plazo de subsanacion"]],
      merits: [["Canal", "Portal empleado"]],
      alerts: view.pendingActions.map((action) => [label(action.label_key || action), "Accion pendiente desde portal"]),
      timeline: [["Avisos sincronizados", `${new Date().toLocaleString("es-ES")} - buzon demo`]],
    },
  ];
  return [...workspaceRows, ...listingRows, ...sectionRows, ...administrativeRows, ...state.localRows].map((row) => {
    const override = state.rowOverrides[row.id];
    if (!override) return row;
    return {
      ...row,
      ...override,
      documents: override.documents || row.documents,
      merits: override.merits || row.merits,
      alerts: override.alerts || row.alerts,
      timeline: override.timeline || row.timeline,
    };
  });
}

function chipClass(index) {
  return ["chip-orange", "chip-indigo", "chip-teal", "chip-blue", "chip-cyan"][index % 5];
}

function renderTable(view) {
  const tbody = $(".queue-panel tbody");
  if (!tbody) return;
  const rows = filteredRows();
  if (!rows.length) {
    const tr = document.createElement("tr");
    tr.innerHTML = `<td colspan="6"><p class="empty-state">Sin expedientes para los filtros activos.</p></td>`;
    tbody.replaceChildren(tr);
    renderDetail(view, null);
    return;
  }
  if (!rows.some((row) => row.id === state.selectedRowID)) {
    state.selectedRowID = rows[0].id;
  }
  tbody.replaceChildren(...rows.map((row, index) => {
    const tr = document.createElement("tr");
    tr.dataset.rowId = row.id;
    tr.tabIndex = 0;
    tr.setAttribute("aria-selected", row.id === state.selectedRowID ? "true" : "false");
    tr.innerHTML = `
      <td><span class="candidate-ref"><strong></strong><span></span></span></td>
      <td><span class="status-chip"></span></td>
      <td></td><td></td><td><span class="status-chip chip-cyan"></span></td>
      <td><button class="table-action" type="button"></button></td>`;
    $("strong", tr).textContent = row.expediente;
    $(".candidate-ref span", tr).textContent = row.candidate;
    const stateChip = $(".status-chip", tr);
    stateChip.classList.add(chipClass(index));
    stateChip.textContent = row.state;
    $$("td", tr)[2].textContent = row.deadline;
    $$("td", tr)[3].textContent = row.points;
    $$(".status-chip", tr)[1].textContent = row.document;
    $("button", tr).textContent = row.action;
    tr.addEventListener("click", () => selectRow(row.id));
    tr.addEventListener("keydown", (event) => {
      if (event.key === "Enter" || event.key === " ") {
        event.preventDefault();
        selectRow(row.id);
      }
    });
    $("button", tr).addEventListener("click", (event) => {
      event.stopPropagation();
      handleRowAction(row);
    });
    return tr;
  }));
  renderDetail(view, rows.find((row) => row.id === state.selectedRowID) || rows[0]);
}

function renderListing(selector, items) {
  const target = $(selector);
  target.replaceChildren();
  if (!items.length) {
    const empty = document.createElement("p");
    empty.className = "empty-state";
    empty.textContent = "Sin solicitudes publicadas desde la API.";
    target.append(empty);
    return;
  }
  target.replaceChildren(...items.map((item, index) => {
    const row = document.createElement("article");
    row.className = "ranking-row";
    row.innerHTML = `<span class="rank"></span><div class="candidate"><strong></strong><span></span></div><strong class="score"></strong>`;
    $(".rank", row).textContent = item.rank || index + 1;
    $(".candidate strong", row).textContent = item.candidate_id || "-";
    $(".candidate span", row).textContent = `${item.solicitud_id || "-"} - ${item.estado || "sin estado"}`;
    $(".score", row).textContent = `${formatPoints(item.total_points)} pt`;
    row.tabIndex = 0;
    row.setAttribute("role", "button");
    row.setAttribute("aria-label", `Abrir ${item.solicitud_id || item.candidate_id || "solicitud"}`);
    row.addEventListener("click", () => selectListingItem(item));
    row.addEventListener("keydown", (event) => {
      if (event.key === "Enter" || event.key === " ") {
        event.preventDefault();
        selectListingItem(item);
      }
    });
    return row;
  }));
}

function renderCronosPanel(view) {
  const target = $("#cronos-panel");
  if (!target) return;
  const summary = view.workspace?.cronos_daily_summary || {};
  const permissions = view.workspace?.cronos_permission_balances || [];
  const sections = view.workspace?.cronos_sections || [];
  const nav = document.createElement("div");
  nav.className = "cronos-nav";
  nav.replaceChildren(...sections.map((section) => {
    const span = document.createElement("span");
    span.textContent = section;
    return span;
  }));

  const summaryBox = document.createElement("div");
  summaryBox.className = "cronos-summary";
  summaryBox.innerHTML = `
    <div><span>Horas teoricas</span><strong></strong></div>
    <div><span>Horas trabajadas</span><strong></strong></div>
    <div><span>Teletrabajo</span><strong></strong></div>
    <div><span>Exceso / defecto mes</span><strong></strong></div>`;
  const values = [
    summary.theoretical || "-",
    summary.worked || "-",
    summary.telework || "-",
    summary.period_balance || summary.daily_balance || "-",
  ];
  $$("strong", summaryBox).forEach((node, index) => {
    node.textContent = values[index];
    if (String(values[index]).startsWith("-")) node.classList.add("negative");
  });

  const table = document.createElement("table");
  table.className = "mini-table";
  table.innerHTML = `
    <thead><tr><th>Permiso</th><th>Solicitar</th><th>Max.</th><th>Solic</th><th>Resta</th></tr></thead>
    <tbody></tbody>`;
  const tbody = $("tbody", table);
  permissions.slice(0, 8).forEach((item) => {
    const tr = document.createElement("tr");
    tr.innerHTML = `<td></td><td></td><td></td><td></td><td></td>`;
    $$("td", tr)[0].textContent = item.name || "-";
    $$("td", tr)[1].textContent = item.request ? "Solicitar" : "-";
    $$("td", tr)[2].textContent = item.max || "-";
    $$("td", tr)[3].textContent = item.requested || "-";
    $$("td", tr)[4].textContent = item.remaining || "-";
    tbody.append(tr);
  });
  target.replaceChildren(nav, summaryBox, table);
}

function renderDietasPanel(view) {
  const target = $("#dietas-panel");
  if (!target) return;
  const routes = view.workspace?.province_routes || [];
  if (!routes.length) {
    target.replaceChildren(Object.assign(document.createElement("p"), {
      className: "empty-state",
      textContent: "Sin rutas de kilometraje cargadas.",
    }));
    return;
  }
  const routeList = document.createElement("div");
  routeList.className = "route-list";
  routeList.replaceChildren(...routes.map((route, index) => {
    const row = document.createElement("article");
    row.className = "route-row";
    row.innerHTML = `
      <div class="route-pin"></div>
      <div><strong></strong><span></span></div>
      <b></b>`;
    $(".route-pin", row).textContent = index + 1;
    $("strong", row).textContent = `${route.from} -> ${route.to}`;
    $("span", row).textContent = `${route.estimated_minutes} min - ${route.allowance}`;
    $("b", row).textContent = `${formatPoints(route.km_one_way)} km`;
    return row;
  }));
  target.replaceChildren(routeList);
}

function screenDefinitions(view) {
  const catalog = Array.isArray(view?.workspace?.screen_catalog) ? view.workspace.screen_catalog : [];
  const source = catalog.length ? catalog : FALLBACK_SCREEN_BLUEPRINTS;
  const byID = new Map();
  source.forEach((item) => {
    const definition = normalizeScreenDefinition(item);
    if (definition?.id) byID.set(definition.id, definition);
  });
  return Array.from(byID.values());
}

function normalizeScreenDefinition(item) {
  if (!item || typeof item !== "object") return null;
  const id = String(item.id || item.menu_id || "").trim();
  if (!id) return null;
  const moduleKey = normalizeScreenModuleKey(item.module_key || item.module || item.module_id, id);
  return {
    ...item,
    id,
    module_key: moduleKey,
    menu_id: item.menu_id || id,
    title: catalogText(item.title) || id,
    description: catalogText(item.description) || MODULE_COPY[moduleKey]?.[1] || "Pantalla operativa del modulo.",
    fields: normalizeScreenFields(item.fields),
    actions: normalizeCatalogList(item.actions, DEFAULT_SCREEN_ACTIONS),
    states: normalizeCatalogList(item.states, DEFAULT_SCREEN_STATES),
    integrations: normalizeCatalogList(item.integrations, DEFAULT_SCREEN_INTEGRATIONS),
    validations: normalizeCatalogList(item.validations, DEFAULT_SCREEN_VALIDATIONS),
    done_criteria: catalogText(item.done_criteria) || "Estado persistido con recibo y auditoria visible.",
  };
}

function normalizeScreenModuleKey(value, id) {
  const raw = String(value || "").toLowerCase().trim();
  const inferred = String(id || "").split(".")[0].toLowerCase();
  const candidate = raw || inferred;
  if (candidate === "admin" || inferred === "admin") return "administracion";
  const module = MODULES.find((item) =>
    candidate === item.id || candidate.endsWith(`.${item.id}`) || candidate.includes(`/${item.id}`),
  );
  return module?.id || inferred;
}

function normalizeScreenFields(fields) {
  const source = Array.isArray(fields) && fields.length ? fields : DEFAULT_SCREEN_FIELDS;
  return source.map((field) => {
    if (!field || typeof field !== "object") {
      const label = catalogText(field) || "Dato";
      return { key: slugify(label), label, type: "text", required: true };
    }
    const label = catalogText(field.label || field.name || field.key) || "Dato";
    return {
      ...field,
      key: catalogText(field.key) || slugify(label),
      label,
      type: catalogText(field.type) || "text",
      required: field.required !== undefined ? Boolean(field.required) : true,
    };
  }).filter((field) => field.key && field.label);
}

function normalizeCatalogList(items, fallback) {
  const source = Array.isArray(items) && items.length ? items : fallback;
  return source.map(catalogText).filter(Boolean);
}

function catalogText(value) {
  if (value && typeof value === "object") {
    return String(value.label || value.title || value.name || value.id || value.key || "").trim();
  }
  return String(value ?? "").trim();
}

function screensForActiveModule(view) {
  if (state.activeModule === "dashboard") return [];
  return screenDefinitions(view).filter((item) => item.module_key === state.activeModule);
}

function activeScreen(view) {
  const screens = screensForActiveModule(view);
  if (!screens.length) return null;
  return screens.find((item) => item.id === state.activeScreen) || screens[0];
}

function ensureActiveScreen(view) {
  const screen = activeScreen(view);
  state.activeScreen = screen?.id || "";
  return screen;
}

function renderModulePortal(view) {
  const target = $("#module-portal");
  if (!target) return;
  if (state.activeModule === "dashboard") {
    target.hidden = true;
    delete target.dataset.mode;
    target.replaceChildren();
    return;
  }
  target.hidden = false;
  target.replaceChildren();
  
  if (state.activeModule === "cronos") {
    target.dataset.mode = "custom-cronos";
    renderCustomCronosApp(target, view);
    return;
  }

  if (state.activeModule === "dietas") {
    target.dataset.mode = "custom-dietas";
    renderCustomDietasApp(target, view);
    return;
  }

  if (state.activeModule === "nominas") {
    target.dataset.mode = "custom-nominas";
    renderCustomNominasApp(target, view);
    return;
  }
  
  const screen = ensureActiveScreen(view);
  const screens = screensForActiveModule(view);
  if (!screen) {
    target.dataset.mode = "generic";
    renderGenericModulePortal(target);
    return;
  }
  target.dataset.mode = "screen";
  renderScreenNavigation(target, screens);
  renderScreenWorkspace(target, screen, view);
}

function renderCustomCronosApp(container, view) {
  if (state.cronosView === undefined) state.cronosView = "calendario";
  if (state.cronosSelectedDay === undefined) state.cronosSelectedDay = 20;
  if (state.cronosSelectedMonth === undefined) state.cronosSelectedMonth = 6;
  if (state.cronosSelectedYear === undefined) state.cronosSelectedYear = 2026;
  if (state.cronosShowTimebar === undefined) state.cronosShowTimebar = false;

  const wrapper = document.createElement("div");
  wrapper.className = "cronos-wrapper";

  const nav = document.createElement("nav");
  nav.className = "cronos-top-nav";
  nav.setAttribute("aria-label", "Menu Cronos");

  const tabsContainer = document.createElement("div");
  tabsContainer.className = "cronos-nav-tabs";

  const btnInicio = document.createElement("button");
  btnInicio.type = "button";
  btnInicio.className = "cronos-tab-btn";
  btnInicio.textContent = "Inicio";
  if (state.cronosView === "calendario") btnInicio.setAttribute("aria-current", "page");
  btnInicio.addEventListener("click", () => {
    state.cronosView = "calendario";
    renderModulePortal(view);
  });
  tabsContainer.append(btnInicio);

  const divMov = document.createElement("div");
  divMov.className = "cronos-dropdown";
  const btnMov = document.createElement("button");
  btnMov.type = "button";
  btnMov.className = "cronos-tab-btn cronos-dropdown-toggle";
  btnMov.textContent = "Movimientos";
  if (state.cronosView.startsWith("movimientos")) btnMov.setAttribute("aria-current", "page");
  
  const divMovContent = document.createElement("div");
  divMovContent.className = "cronos-dropdown-content";

  const optionsMov = [
    ["Movimientos Hoy", "movimientos-hoy"],
    ["Movimientos Semana Actual", "movimientos-semana"],
    ["Movimientos Mes Actual", "movimientos-mes"],
    ["Movimientos Año Actual", "movimientos-anio"],
    ["Selección Movimientos", "seleccion-movimientos"],
    ["Olvidos de Marcaje", "olvidos-marcaje"],
    ["Absentismos", "absentismos"],
    ["Calendario 2026", "calendario"]
  ];
  optionsMov.forEach(([label, value]) => {
    const optBtn = document.createElement("button");
    optBtn.type = "button";
    optBtn.textContent = label;
    optBtn.addEventListener("click", () => {
      state.cronosView = value;
      if (value === "calendario") {
        state.cronosSelectedDay = 20;
      }
      renderModulePortal(view);
    });
    divMovContent.append(optBtn);
  });
  divMov.append(btnMov, divMovContent);
  tabsContainer.append(divMov);

  const divPerm = document.createElement("div");
  divPerm.className = "cronos-dropdown";
  const btnPerm = document.createElement("button");
  btnPerm.type = "button";
  btnPerm.className = "cronos-tab-btn cronos-dropdown-toggle";
  btnPerm.textContent = "Permisos y Licencias";
  if (state.cronosView.startsWith("permisos") || state.cronosView.startsWith("resumen-permisos") || state.cronosView.startsWith("todos-permisos")) {
    btnPerm.setAttribute("aria-current", "page");
  }

  const divPermContent = document.createElement("div");
  divPermContent.className = "cronos-dropdown-content";

  const optionsPerm = [
    ["Resumen de Permisos", "resumen-permisos"],
    ["Listado de Todos los Permisos del Año 2026", "todos-permisos"],
    ["Permisos Pendientes de Justificar", "permisos-pendientes-justificar"],
    ["Ver todos los Permisos Solicitados Pendientes de Conceder", "permisos-pendientes-conceder"]
  ];
  optionsPerm.forEach(([label, value]) => {
    const optBtn = document.createElement("button");
    optBtn.type = "button";
    optBtn.textContent = label;
    optBtn.addEventListener("click", () => {
      state.cronosView = value;
      renderModulePortal(view);
    });
    divPermContent.append(optBtn);
  });
  divPerm.append(btnPerm, divPermContent);
  tabsContainer.append(divPerm);

  const divNotif = document.createElement("div");
  divNotif.className = "cronos-dropdown";
  const btnNotif = document.createElement("button");
  btnNotif.type = "button";
  btnNotif.className = "cronos-tab-btn cronos-dropdown-toggle";
  btnNotif.textContent = "Notificaciones";
  if (state.cronosView.startsWith("notificaciones") || state.cronosView.startsWith("enviar-notificacion") || state.cronosView.startsWith("seleccion-notificaciones")) {
    btnNotif.setAttribute("aria-current", "page");
  }

  const divNotifContent = document.createElement("div");
  divNotifContent.className = "cronos-dropdown-content";

  const optionsNotif = [
    ["Enviar Notificación", "enviar-notificacion"],
    ["Selección de Notificaciones", "seleccion-notificaciones"],
    ["Ver todas las Notificaciones Pendientes", "notificaciones-pendientes"]
  ];
  optionsNotif.forEach(([label, value]) => {
    const optBtn = document.createElement("button");
    optBtn.type = "button";
    optBtn.textContent = label;
    optBtn.addEventListener("click", () => {
      state.cronosView = value;
      renderModulePortal(view);
    });
    divNotifContent.append(optBtn);
  });
  divNotif.append(btnNotif, divNotifContent);
  tabsContainer.append(divNotif);

  const btnConfig = document.createElement("button");
  btnConfig.type = "button";
  btnConfig.className = "cronos-tab-btn";
  btnConfig.textContent = "Configuración";
  if (state.cronosView === "configuracion") btnConfig.setAttribute("aria-current", "page");
  btnConfig.addEventListener("click", () => {
    state.cronosView = "configuracion";
    renderModulePortal(view);
  });
  tabsContainer.append(btnConfig);

  const btnMensajeria = document.createElement("button");
  btnMensajeria.type = "button";
  btnMensajeria.className = "cronos-tab-btn";
  btnMensajeria.textContent = "Mensajería";
  if (state.cronosView === "mensajes") btnMensajeria.setAttribute("aria-current", "page");
  btnMensajeria.addEventListener("click", () => {
    state.cronosView = "mensajes";
    renderModulePortal(view);
  });
  tabsContainer.append(btnMensajeria);

  nav.append(tabsContainer);
  wrapper.append(nav);

  const contentPane = document.createElement("div");
  contentPane.className = "cronos-content-pane";
  wrapper.append(contentPane);

  if (state.cronosView === "calendario") {
    renderCronosCalendarView(contentPane, view);
  } else {
    renderCronosTableView(contentPane, state.cronosView, view);
  }

  container.append(wrapper);
}

function renderCronosCalendarView(pane, view) {
  const container = document.createElement("div");
  container.className = "cronos-calendar-container";

  const table = document.createElement("table");
  table.className = "cronos-calendar-table";

  const headerTr = document.createElement("tr");
  const headerTh = document.createElement("th");
  headerTh.className = "calendar-header";
  headerTh.setAttribute("colspan", "8");

  const prevArrow = document.createElement("span");
  prevArrow.className = "nav-arrow";
  prevArrow.textContent = "<<";
  prevArrow.addEventListener("click", () => {
    state.cronosSelectedMonth--;
    if (state.cronosSelectedMonth < 1) {
      state.cronosSelectedMonth = 12;
      state.cronosSelectedYear--;
    }
    renderModulePortal(view);
  });

  const nextArrow = document.createElement("span");
  nextArrow.className = "nav-arrow";
  nextArrow.textContent = ">>";
  nextArrow.addEventListener("click", () => {
    state.cronosSelectedMonth++;
    if (state.cronosSelectedMonth > 12) {
      state.cronosSelectedMonth = 1;
      state.cronosSelectedYear++;
    }
    renderModulePortal(view);
  });

  const monthNames = [
    "Enero", "Febrero", "Marzo", "Abril", "Mayo", "Junio",
    "Julio", "Agosto", "Septiembre", "Octubre", "Noviembre", "Diciembre"
  ];
  const titleText = document.createTextNode(` ${monthNames[state.cronosSelectedMonth - 1]} ${state.cronosSelectedYear} `);

  headerTh.append(prevArrow, titleText, nextArrow);
  headerTr.append(headerTh);
  table.append(headerTr);

  const dayNamesTr = document.createElement("tr");
  const dayNames = ["Lu", "Ma", "Mi", "Ju", "Vi", "Sa", "Do", "Semana"];
  dayNames.forEach((dn) => {
    const th = document.createElement("th");
    th.className = "day-name";
    th.textContent = dn;
    dayNamesTr.append(th);
  });
  table.append(dayNamesTr);

  const firstDay = new Date(state.cronosSelectedYear, state.cronosSelectedMonth - 1, 1);
  const totalDays = new Date(state.cronosSelectedYear, state.cronosSelectedMonth, 0).getDate();
  
  let startDay = firstDay.getDay();
  if (startDay === 0) startDay = 6;
  else startDay = startDay - 1;

  const getISOWeek = (d) => {
    const date = new Date(d.getTime());
    date.setHours(0, 0, 0, 0);
    date.setDate(date.getDate() + 3 - (date.getDay() + 6) % 7);
    const week1 = new Date(date.getFullYear(), 0, 4);
    return 1 + Math.round(((date.getTime() - week1.getTime()) / 86400000 - 3 + (week1.getDay() + 6) % 7) / 7);
  };

  const holidays = [4, 5, 7, 14, 21, 28];

  let currentDay = 1;
  let finished = false;

  for (let r = 0; r < 6; r++) {
    if (finished) break;
    const tr = document.createElement("tr");
    let hasDaysInWeek = false;

    for (let c = 0; c < 7; c++) {
      const td = document.createElement("td");
      if ((r === 0 && c < startDay) || currentDay > totalDays) {
        td.className = "empty";
        tr.append(td);
      } else {
        hasDaysInWeek = true;
        td.textContent = currentDay;
        const dayVal = currentDay;

        if (dayVal === state.cronosSelectedDay) {
          td.className = "selected";
        }

        const isSunday = (c === 6);
        const isCustomHoliday = state.cronosSelectedMonth === 6 && state.cronosSelectedYear === 2026 && holidays.includes(dayVal);
        if (isSunday || isCustomHoliday) {
          td.classList.add("holiday");
        }

        td.addEventListener("click", () => {
          state.cronosSelectedDay = dayVal;
          renderModulePortal(view);
        });

        tr.append(td);
        currentDay++;
      }
    }

    if (hasDaysInWeek) {
      const tdWeek = document.createElement("td");
      tdWeek.className = "week-number";
      const sampleDateInWeek = new Date(state.cronosSelectedYear, state.cronosSelectedMonth - 1, currentDay - 4 > 1 ? currentDay - 4 : 1);
      tdWeek.textContent = getISOWeek(sampleDateInWeek);
      tr.append(tdWeek);
      table.append(tr);
    } else {
      finished = true;
    }
  }

  container.append(table);

  const selectsDiv = document.createElement("div");
  selectsDiv.className = "cronos-month-year-selects";
  
  selectsDiv.innerHTML = `
    <label>Mes: <select class="month-select"></select></label>
    <label>Año: <select class="year-select"></select></label>
  `;

  const monthSel = $(".month-select", selectsDiv);
  const yearSel = $(".year-select", selectsDiv);

  monthNames.forEach((name, idx) => {
    const opt = document.createElement("option");
    opt.value = idx + 1;
    opt.textContent = name;
    if (idx + 1 === state.cronosSelectedMonth) opt.selected = true;
    monthSel.append(opt);
  });

  for (let y = 2020; y <= 2030; y++) {
    const opt = document.createElement("option");
    opt.value = y;
    opt.textContent = y;
    if (y === state.cronosSelectedYear) opt.selected = true;
    yearSel.append(opt);
  }

  monthSel.addEventListener("change", (e) => {
    state.cronosSelectedMonth = parseInt(e.target.value, 10);
    renderModulePortal(view);
  });

  yearSel.addEventListener("change", (e) => {
    state.cronosSelectedYear = parseInt(e.target.value, 10);
    renderModulePortal(view);
  });

  container.append(selectsDiv);
  pane.append(container);

  const actionBtn = document.createElement("button");
  actionBtn.type = "button";
  actionBtn.className = "cronos-btn-link";
  actionBtn.textContent = `Visualizar Movimientos / Permisos / Bajas ${state.cronosSelectedDay} de ${monthNames[state.cronosSelectedMonth - 1]} de ${state.cronosSelectedYear}`;
  actionBtn.addEventListener("click", () => {
    recordReceipt("Consulta Movimientos", `Visualizando dia ${state.cronosSelectedDay}/${state.cronosSelectedMonth}/${state.cronosSelectedYear}`, "cronos");
    alert(`Visualizando detalles para el día ${state.cronosSelectedDay} de ${monthNames[state.cronosSelectedMonth - 1]} de ${state.cronosSelectedYear}.`);
  });
  pane.append(actionBtn);

  const dayTitle = document.createElement("h3");
  dayTitle.className = "cronos-table-title";
  dayTitle.textContent = `Resumen ${state.cronosSelectedDay} de ${monthNames[state.cronosSelectedMonth - 1]} de ${state.cronosSelectedYear}`;
  pane.append(dayTitle);

  const daySummaryTable = document.createElement("table");
  daySummaryTable.className = "cronos-summary-table";

  const isWeekendOrHoliday = (state.cronosSelectedMonth === 6 && state.cronosSelectedYear === 2026 && (holidays.includes(state.cronosSelectedDay) || state.cronosSelectedDay % 7 === 0 || state.cronosSelectedDay % 7 === 6));
  
  let theoretical = "07:30";
  let worked = "07:30";
  let balance = "00:00";
  let isNegative = false;

  if (isWeekendOrHoliday) {
    theoretical = "00:00";
    worked = "00:00";
    balance = "00:00";
  } else if (state.cronosSelectedDay === 18 && state.cronosSelectedMonth === 6 && state.cronosSelectedYear === 2026) {
    theoretical = "06:00";
    worked = "05:36";
    balance = "-00:24";
    isNegative = true;
  } else {
    if (state.cronosSelectedDay % 3 === 0) {
      theoretical = "07:30";
      worked = "07:42";
      balance = "+00:12";
    } else if (state.cronosSelectedDay % 5 === 0) {
      theoretical = "07:30";
      worked = "07:18";
      balance = "-00:12";
      isNegative = true;
    }
  }

  daySummaryTable.innerHTML = `
    <tr>
      <td>HORAS TEÓRICAS</td>
      <td>${theoretical}</td>
    </tr>
    <tr>
      <td>HORAS TRABAJADAS</td>
      <td>${worked}</td>
    </tr>
    <tr>
      <td>EXCESO / DEFECTO</td>
      <td class="${isNegative ? 'negative' : ''}">${balance}</td>
    </tr>
  `;
  pane.append(daySummaryTable);

  const toggleHorarioBtn = document.createElement("button");
  toggleHorarioBtn.type = "button";
  toggleHorarioBtn.className = "cronos-visualizar-horario-btn";
  toggleHorarioBtn.textContent = state.cronosShowTimebar ? "Ocultar Horario" : "Visualizar Horario";
  toggleHorarioBtn.addEventListener("click", () => {
    state.cronosShowTimebar = !state.cronosShowTimebar;
    renderModulePortal(view);
  });
  pane.append(toggleHorarioBtn);

  if (state.cronosShowTimebar) {
    const timebarContainer = document.createElement("div");
    timebarContainer.className = "cronos-timebar-container";

    const tbTitle = document.createElement("div");
    tbTitle.className = "cronos-timebar-title";
    tbTitle.textContent = "Su horario de este día es:";
    timebarContainer.append(tbTitle);

    const timebar = document.createElement("div");
    timebar.className = "cronos-timebar";

    for (let cellIdx = 0; cellIdx < 48; cellIdx++) {
      const cell = document.createElement("div");
      cell.className = "cronos-timecell";

      if (isWeekendOrHoliday) {
        cell.classList.add("gray");
      } else {
        if (cellIdx < 14) {
          cell.classList.add("gray");
        } else if (cellIdx >= 14 && cellIdx < 19) {
          cell.classList.add("blue");
        } else if (cellIdx >= 19 && cellIdx < 37) {
          cell.classList.add("green");
        } else if (cellIdx >= 37 && cellIdx < 42) {
          cell.classList.add("blue");
        } else {
          cell.classList.add("gray");
        }
      }
      timebar.append(cell);
    }
    timebarContainer.append(timebar);

    const legend = document.createElement("div");
    legend.className = "cronos-legend";
    legend.innerHTML = `
      <div class="cronos-legend-item">
        <span class="cronos-legend-swatch green"></span>
        <span>PRESENCIA OBLIGADA</span>
      </div>
      <div class="cronos-legend-item">
        <span class="cronos-legend-swatch blue"></span>
        <span>HORARIO FLEXIBLE</span>
      </div>
      <div class="cronos-legend-item">
        <span class="cronos-legend-swatch yellow"></span>
        <span>HORARIO CORTESÍA</span>
      </div>
      <div class="cronos-legend-item">
        <span class="cronos-legend-swatch red"></span>
        <span>NO COMPUTABLE</span>
      </div>
    `;
    timebarContainer.append(legend);
    pane.append(timebarContainer);
  }

  const periodTitle = document.createElement("h3");
  periodTitle.className = "cronos-table-title";
  periodTitle.textContent = `Resumen desde el día 1 al ${state.cronosSelectedDay === 1 ? 1 : state.cronosSelectedDay - 1} de ${monthNames[state.cronosSelectedMonth - 1]} de ${state.cronosSelectedYear}`;
  pane.append(periodTitle);

  const detailLink = document.createElement("span");
  detailLink.className = "cronos-summary-table link-detail";
  detailLink.textContent = "Ver Detalle por Fechas";
  detailLink.addEventListener("click", () => {
    state.cronosView = "movimientos-mes";
    renderModulePortal(view);
  });
  pane.append(detailLink);

  const periodSummaryTable = document.createElement("table");
  periodSummaryTable.className = "cronos-summary-table";

  let pTheoretical = "82:00";
  let pWorked = "62:26";
  let pTelework = "15:00";
  let pBalance = "-04:34";

  if (state.cronosSelectedDay === 18 && state.cronosSelectedMonth === 6 && state.cronosSelectedYear === 2026) {
    pTheoretical = "74:30";
    pWorked = "62:26";
    pTelework = "07:30";
    pBalance = "-04:34";
  } else {
    let weekdays = 0;
    for (let d = 1; d < state.cronosSelectedDay; d++) {
      const dayOfWeek = new Date(state.cronosSelectedYear, state.cronosSelectedMonth - 1, d).getDay();
      const isCustomHol = state.cronosSelectedMonth === 6 && state.cronosSelectedYear === 2026 && holidays.includes(d);
      if (dayOfWeek !== 0 && dayOfWeek !== 6 && !isCustomHol) {
        weekdays++;
      }
    }
    const hoursExpected = weekdays * 7.5;
    pTheoretical = `${hoursExpected}:00`;
    pWorked = `${Math.floor(hoursExpected * 0.85)}:14`;
    pTelework = `${Math.floor(hoursExpected * 0.1)}:00`;
    const balanceVal = Math.floor(hoursExpected * 0.85) + Math.floor(hoursExpected * 0.1) - hoursExpected;
    pBalance = balanceVal < 0 ? `-${Math.abs(balanceVal)}:46` : `+${balanceVal}:46`;
  }

  periodSummaryTable.innerHTML = `
    <tr>
      <td>SALDO ANTERIOR</td>
      <td>00:00</td>
    </tr>
    <tr>
      <td>HORAS TEÓRICAS</td>
      <td>${pTheoretical}</td>
    </tr>
    <tr>
      <td>HORAS TRABAJADAS</td>
      <td>${pWorked}</td>
    </tr>
    <tr>
      <td>HORAS TELETRABAJO</td>
      <td>${pTelework}</td>
    </tr>
    <tr>
      <td>EXCESO / DEFECTO</td>
      <td class="negative">${pBalance}</td>
    </tr>
    <tr>
      <td>TOTAL CON SALDO ANTERIOR</td>
      <td class="negative">${pBalance}</td>
    </tr>
  `;
  pane.append(periodSummaryTable);

  const bottomLinks = document.createElement("div");
  bottomLinks.className = "cronos-bottom-links";
  
  const btnAnt = document.createElement("button");
  btnAnt.type = "button";
  btnAnt.textContent = "Anterior";
  btnAnt.addEventListener("click", () => {
    if (state.cronosSelectedDay > 1) {
      state.cronosSelectedDay--;
    } else {
      state.cronosSelectedDay = 30;
    }
    renderModulePortal(view);
  });

  const btnPrincipal = document.createElement("button");
  btnPrincipal.type = "button";
  btnPrincipal.textContent = "Ir a Principal";
  btnPrincipal.addEventListener("click", () => {
    setActiveModule("dashboard");
  });

  bottomLinks.append(btnAnt, btnPrincipal);
  pane.append(bottomLinks);
}

function renderCronosTableView(pane, viewKey, view) {
  const container = document.createElement("div");
  container.className = "cronos-table-view";

  const title = document.createElement("h3");
  container.append(title);

  const table = document.createElement("table");
  container.append(table);

  if (viewKey === "movimientos-hoy" || viewKey === "movimientos-semana" || viewKey === "movimientos-mes" || viewKey === "movimientos-anio" || viewKey === "seleccion-movimientos") {
    title.textContent = `Movimientos del Personal - ${viewKey.replace("-", " ").toUpperCase()}`;
    table.innerHTML = `
      <thead>
        <tr>
          <th>Fecha</th>
          <th>Hora</th>
          <th>Acción</th>
          <th>Canal / Dispositivo</th>
          <th>Estado</th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <td>18/06/2026</td>
          <td>08:00</td>
          <td>Entrada</td>
          <td>Terminal de acceso F3</td>
          <td>Validado</td>
        </tr>
        <tr>
          <td>18/06/2026</td>
          <td>13:36</td>
          <td>Salida</td>
          <td>Terminal de acceso F3</td>
          <td>Validado</td>
        </tr>
        <tr>
          <td>17/06/2026</td>
          <td>07:55</td>
          <td>Entrada</td>
          <td>Fichaje Web</td>
          <td>Validado</td>
        </tr>
        <tr>
          <td>17/06/2026</td>
          <td>15:30</td>
          <td>Salida</td>
          <td>Fichaje Web</td>
          <td>Validado</td>
        </tr>
        <tr>
          <td>16/06/2026</td>
          <td>08:04</td>
          <td>Entrada</td>
          <td>Terminal de acceso F3</td>
          <td>Validado</td>
        </tr>
        <tr>
          <td>16/06/2026</td>
          <td>15:44</td>
          <td>Salida</td>
          <td>Terminal de acceso F3</td>
          <td>Validado</td>
        </tr>
      </tbody>
    `;
  } else if (viewKey === "olvidos-marcaje") {
    title.textContent = "Olvidos de Marcaje Registrados";
    table.innerHTML = `
      <thead>
        <tr>
          <th>Fecha Incidencia</th>
          <th>Tipo</th>
          <th>Acción Propuesta</th>
          <th>Justificación</th>
          <th>Estado</th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <td>19/06/2026</td>
          <td>Entrada sin fichar</td>
          <td>Crear marcaje 08:00</td>
          <td>Asistencia a reunión comarcal en Motril</td>
          <td>Pendiente de validar</td>
        </tr>
      </tbody>
    `;
  } else if (viewKey === "absentismos") {
    title.textContent = "Registro de Absentismos y Bajas";
    table.innerHTML = `
      <thead>
        <tr>
          <th>Tipo Absentismo</th>
          <th>Fecha Inicio</th>
          <th>Fecha Fin</th>
          <th>Justificante</th>
          <th>Estado</th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <td>Baja Médica Común</td>
          <td>02/06/2026</td>
          <td>05/06/2026</td>
          <td>Parte_Medico_142.pdf</td>
          <td>Aprobado y cerrado</td>
        </tr>
      </tbody>
    `;
  } else if (viewKey === "todos-permisos" || viewKey === "resumen-permisos") {
    table.remove();
    title.remove();
    
    container.innerHTML = `
      <h3 style="text-align:center; font-size:1.3rem; font-weight:bold; text-decoration:underline; margin-bottom:12px; font-family:sans-serif;">
        Listado de Todos los Permisos del Año 2026
      </h3>
      
      <div style="display:flex; justify-content:center; align-items:center; gap:20px; margin-bottom:16px; font-size:0.9rem; font-family:sans-serif;">
        <label>
          Año: 
          <select class="permiso-year-select" style="padding:2px 6px;">
            <option value="2026" selected>2026</option>
            <option value="2025">2025</option>
          </select>
        </label>
        <label style="display:flex; align-items:center; gap:6px;">
          <input type="checkbox" class="permiso-filter-only-requested">
          Ver Sólo Permisos Solicitados o Concedidos
        </label>
      </div>

      <table class="cronos-permisos-table" style="width:100%; border-collapse:collapse; font-size:0.82rem; border:1px solid #777; margin-bottom:20px; font-family:sans-serif;">
        <thead>
          <tr style="background:#000; color:#fff;">
            <th style="padding:6px; border:1px solid #777; text-align:center; font-weight:bold;">Permiso</th>
            <th style="padding:6px; border:1px solid #777; text-align:center; font-weight:bold;">Solicitar</th>
            <th style="padding:6px; border:1px solid #777; text-align:center; font-weight:bold;">Máx.</th>
            <th style="padding:6px; border:1px solid #777; text-align:center; font-weight:bold;">Min.</th>
            <th style="padding:6px; border:1px solid #777; text-align:center; font-weight:bold;">Solic</th>
            <th style="padding:6px; border:1px solid #777; text-align:center; font-weight:bold;">Resta</th>
          </tr>
        </thead>
        <tbody class="permisos-table-body">
        </tbody>
        <tfoot>
          <tr style="background:#000; color:#fff; font-weight:bold;">
            <td style="padding:6px; border:1px solid #777; text-align:center;">Permiso</td>
            <td style="padding:6px; border:1px solid #777; text-align:center;">Solicitar</td>
            <td style="padding:6px; border:1px solid #777; text-align:center;">Máx.</td>
            <td style="padding:6px; border:1px solid #777; text-align:center;">Min.</td>
            <td style="padding:6px; border:1px solid #777; text-align:center;">Solic</td>
            <td style="padding:6px; border:1px solid #777; text-align:center;">Resta</td>
          </tr>
        </tfoot>
      </table>
    `;

    const tbody = $(".permisos-table-body", container);
    const filterCheckbox = $(".permiso-filter-only-requested", container);

    const renderRows = (onlyRequested) => {
      tbody.innerHTML = "";
      
      const rowsData = [
        { name: "ASISTENCIA A EXAMENES", action: "(J-A) Solicitar", max: "07:30 (*)", min: "", solic: "", resta: "07:30", requested: false },
        { name: "ASUNTOS PROPIOS", action: "(A) Solicitar", max: "6", min: "", solic: "", resta: "3", requested: true },
        { name: "BOLSA DIAS POR TRIENIOS", action: "", max: "2", min: "", solic: "", resta: "0", requested: false },
        { name: "BOLSA DIAS VACACIONES AÑOS DE SERVICIO", action: "", max: "2", min: "", solic: "", resta: "0", requested: false },
        { name: "BOLSA HORARIA POR CONCILIACION", action: "(A) Solicitar", max: "30:00", min: "", solic: "", resta: "13:08", requested: true },
        { name: "COMPENSACION FESTIVOS", action: "", max: "28", min: "14", solic: "", resta: "28", requested: false },
        { name: "COMPENSACION HORARIA - PERMISO (SALDO HORAS EXTRAS)", action: "(A) Solicitar", max: "A (00:19)", min: "", solic: "", resta: "00:19", requested: true },
        { name: "COMPENSACION SABADOS NUEVO REGLAMENTO", action: "", max: "2", min: "", solic: "", resta: "0", requested: false },
        { name: "COMPENSACION TIEMPO Y OCIO LIBRE", action: "", max: "15", min: "", solic: "", resta: "15", requested: false },
        { name: "CONTRATO DE RELEVO DIAS DE NO TRABAJO", action: "", max: "50", min: "", solic: "", resta: "50", requested: false },
        { name: "CURSO FORMACION AGRUPADA, IAAP - INAP -", action: "(J-A) Solicitar", max: "90", min: "", solic: "", resta: "90", requested: false },
        { name: "CURSO FORMACION AGRUPADA, IAAP-INAP - PER.CENTROS", action: "(J-A) Solicitar", max: "90", min: "", solic: "", resta: "90", requested: false },
        { name: "EMBARAZO-BAJA MATERNAL", action: "", max: "112", min: "42", solic: "", resta: "112", requested: false },
        { name: "ENF. GRAVE/HOSPIT/INTERV. 2º GRADO", action: "(J-A) Solicitar", max: "4 (*)", min: "", solic: "", resta: "4", requested: false },
        { name: "ENF.GRAVE/HOSPIT/INTER. 1º GRADO", action: "(J-A) Solicitar", max: "5 (*)", min: "", solic: "", resta: "5", requested: false },
        { name: "ENFERMEDAD SIN BAJA (PERMISO)", action: "", max: "4", min: "", solic: "", resta: "3", requested: true },
        { name: "ENFERMEDAD SIN BAJA CON DESCUENTO", action: "", max: "365", min: "", solic: "", resta: "365", requested: false },
        { name: "FALLEC FAMIL. 2º GRADO DIST. LOCALIDAD", action: "(J-A) Solicitar", max: "4 (*)", min: "", solic: "", resta: "4", requested: false },
        { name: "FALLECIMIENTO FAMILIAN 1º GRADO", action: "(J-A) Solicitar", max: "3 (*)", min: "", solic: "", resta: "3", requested: false },
        { name: "FALLECIMIENTO FAMILIAR 1º GRADO DIST. LOCALIDAD", action: "(J-A) Solicitar", max: "5 (*)", min: "", solic: "", resta: "5", requested: false },
        { name: "FALLECIMIENTO FAMILIAR 2º GRADO", action: "(J-A) Solicitar", max: "2 (*)", min: "", solic: "", resta: "2", requested: false },
        { name: "FORMACION EXTERNA", action: "(J-A) Solicitar", max: "60:00", min: "", solic: "", resta: "60:00", requested: false },
        { name: "GESTION DE SERVICIO", action: "(A) Solicitar", max: "999:00 (*)", min: "", solic: "", resta: "999:00", requested: false },
        { name: "HORAS DE MEDICO", action: "(J-A) Solicitar", max: "03:00 (*)", min: "", solic: "", resta: "03:00", requested: false },
        { name: "HORAS SINDICALES ( JUNIO ) Mostrar todos los Meses", action: "Solicitar", max: "60:00 (M)", min: "", solic: "", resta: "01:30", requested: true, customName: true },
        { name: "MIERCOLES SEMANA SANTA Y CORPUS", action: "", max: "1", min: "", solic: "", resta: "0", requested: false },
        { name: "NACIMIENTO HIJO/A", action: "", max: "7", min: "", solic: "", resta: "7", requested: false },
        { name: "NUEVA NORMALIDAD TRABAJO NO PRESENCIAL", action: "", max: "31 (*)", min: "", solic: "", resta: "31", requested: false },
        { name: "P. PROGENITOR DIF. MADRE BIOLOGICA (6 SEMANAS)", action: "", max: "42", min: "42", solic: "", resta: "42", requested: false },
        { name: "PERMISO MATERNAL", action: "", max: "28", min: "28", solic: "", resta: "28", requested: false },
        { name: "TRASLADO DE DOMICILIO", action: "", max: "1 (*)", min: "", solic: "", resta: "1", requested: false },
        { name: "VACACIONES", action: "Solicitar", max: "22", min: "", solic: "", resta: "0", requested: true }
      ];

      rowsData.forEach((row, idx) => {
        if (onlyRequested && !row.requested) return;

        const tr = document.createElement("tr");
        tr.style.background = idx % 2 === 0 ? "#e0eafd" : "#ffffff";

        const tdName = document.createElement("td");
        tdName.style.padding = "6px";
        tdName.style.border = "1px solid #777";
        tdName.style.textAlign = "left";
        
        if (row.customName) {
          tdName.innerHTML = `<span style="font-weight: bold; color: #000;">HORAS SINDICALES ( <span style="text-decoration: underline;">JUNIO</span> )</span> &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; <a href="#" class="show-all-months" style="color:var(--focus); text-decoration:underline; font-size: 0.8rem;">Mostrar todos los Meses</a> <span style="font-size:0.9rem;">📅</span>`;
          const showLink = $(".show-all-months", tdName);
          if (showLink) {
            showLink.addEventListener("click", (e) => {
              e.preventDefault();
              alert("Mostrando horas sindicales para todos los meses.");
            });
          }
        } else {
          const a = document.createElement("a");
          a.href = "#";
          a.style.color = "#000";
          a.style.textDecoration = "underline";
          a.style.fontWeight = "bold";
          a.textContent = row.name;
          a.addEventListener("click", (e) => {
            e.preventDefault();
            alert(`Detalles del permiso: ${row.name}`);
          });
          tdName.append(a);
        }
        tr.append(tdName);

        const tdAction = document.createElement("td");
        tdAction.style.padding = "6px";
        tdAction.style.border = "1px solid #777";
        tdAction.style.textAlign = "center";
        
        if (row.action) {
          const prefix = row.action.includes(" ") ? row.action.substring(0, row.action.indexOf(" ") + 1) : "";
          const linkText = row.action.includes(" ") ? row.action.substring(row.action.indexOf(" ") + 1) : row.action;
          
          if (prefix) {
            const span = document.createElement("span");
            span.style.fontWeight = "bold";
            span.style.color = "#000";
            span.textContent = prefix;
            tdAction.append(span);
          }
          
          const a = document.createElement("a");
          a.href = "#";
          a.style.color = "blue";
          a.style.textDecoration = "underline";
          a.style.fontWeight = "bold";
          a.textContent = linkText;
          a.addEventListener("click", (e) => {
            e.preventDefault();
            openCronosRequestModal(row.name, view);
          });
          tdAction.append(a);
        }
        tr.append(tdAction);

        const tdMax = document.createElement("td");
        tdMax.style.padding = "6px";
        tdMax.style.border = "1px solid #777";
        tdMax.style.textAlign = "center";
        tdMax.textContent = row.max;
        tr.append(tdMax);

        const tdMin = document.createElement("td");
        tdMin.style.padding = "6px";
        tdMin.style.border = "1px solid #777";
        tdMin.style.textAlign = "center";
        tdMin.textContent = row.min;
        tr.append(tdMin);

        const tdSolic = document.createElement("td");
        tdSolic.style.padding = "6px";
        tdSolic.style.border = "1px solid #777";
        tdSolic.style.textAlign = "center";
        tdSolic.textContent = row.solic;
        tr.append(tdSolic);

        const tdResta = document.createElement("td");
        tdResta.style.padding = "6px";
        tdResta.style.border = "1px solid #777";
        tdResta.style.textAlign = "center";
        tdResta.style.fontWeight = "bold";
        tdResta.style.color = "blue";
        tdResta.textContent = row.resta;
        tr.append(tdResta);

        tbody.append(tr);
      });
    };

    renderRows(false);

    filterCheckbox.addEventListener("change", (e) => {
      renderRows(e.target.checked);
    });
  } else if (viewKey === "permisos-pendientes-justificar" || viewKey === "permisos-pendientes-conceder") {
    title.textContent = `Listado de Permisos Solicitados - ${viewKey.replace("-", " ").toUpperCase()}`;
    
    let rowsHtml = "";
    if (state.cronosSubmittedRequests && state.cronosSubmittedRequests.length > 0) {
      state.cronosSubmittedRequests.forEach((req) => {
        rowsHtml += `
          <tr>
            <td>${req.fechaSolicitud}</td>
            <td>${req.concepto}</td>
            <td>${req.desde}</td>
            <td>${req.hasta}</td>
            <td>${req.duracion}</td>
            <td style="color:var(--focus); font-weight:bold;">${req.estado}</td>
          </tr>
        `;
      });
    }

    table.innerHTML = `
      <thead>
        <tr>
          <th>Fecha Solicitud</th>
          <th>Concepto</th>
          <th>Desde</th>
          <th>Hasta</th>
          <th>Días/Horas</th>
          <th>Estado</th>
        </tr>
      </thead>
      <tbody>
        ${rowsHtml}
        <tr>
          <td>15/06/2026</td>
          <td>Asuntos Propios</td>
          <td>22/06/2026</td>
          <td>23/06/2026</td>
          <td>2 días</td>
          <td>Pendiente de validar</td>
        </tr>
        <tr>
          <td>10/05/2026</td>
          <td>Médico Especialista</td>
          <td>12/05/2026</td>
          <td>12/05/2026</td>
          <td>3:00 horas</td>
          <td>Aprobado</td>
        </tr>
      </tbody>
    `;
  } else if (viewKey === "enviar-notificacion") {
    title.textContent = "Enviar Nueva Notificación / Solicitud a RRHH";
    container.innerHTML = `
      <h3>Enviar Nueva Notificación / Solicitud a RRHH</h3>
      <form class="workspace-form" style="display:grid; gap:12px; max-width:480px; margin:20px 0;">
        <label style="display:grid; gap:4px; font-weight:bold;">
          Asunto / Motivo:
          <input type="text" placeholder="Ej. Olvido de fichaje salida" style="padding:8px; border:1px solid var(--line); border-radius:4px;">
        </label>
        <label style="display:grid; gap:4px; font-weight:bold;">
          Mensaje explicativo:
          <textarea rows="4" placeholder="Describa brevemente la solicitud..." style="padding:8px; border:1px solid var(--line); border-radius:4px;"></textarea>
        </label>
        <button type="submit" class="primary-action" style="max-width:180px;">Enviar a RRHH</button>
      </form>
    `;
    const form = $("form", container);
    if (form) {
      form.addEventListener("submit", (e) => {
        e.preventDefault();
        recordReceipt("Notificación Enviada", "Solicitud manual a RRHH registrada", "cronos");
        alert("Notificación enviada con éxito.");
        state.cronosView = "calendario";
        renderModulePortal(view);
      });
    }
  } else if (viewKey === "seleccion-notificaciones" || viewKey === "notificaciones-pendientes") {
    title.textContent = "Bandeja de Notificaciones y Solicitudes";
    table.innerHTML = `
      <thead>
        <tr>
          <th>Fecha</th>
          <th>Tipo</th>
          <th>Asunto</th>
          <th>Destinatario</th>
          <th>Estado</th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <td>19/06/2026</td>
          <td>Solicitud</td>
          <td>Justificante de Asuntos Propios 22-23 Junio</td>
          <td>Responsable de Unidad</td>
          <td>Enviada</td>
        </tr>
        <tr>
          <td>15/06/2026</td>
          <td>Aviso</td>
          <td>Aviso de validación de cuadrante mensual</td>
          <td>Personal RRHH</td>
          <td>Recibida</td>
        </tr>
      </tbody>
    `;
  } else if (viewKey === "configuracion") {
    title.textContent = "Configuración de Fichajes y Turnos";
    container.innerHTML = `
      <h3>Configuración de Fichajes y Turnos</h3>
      <div style="display:grid; gap:16px; margin:20px 0;">
        <label style="display:flex; align-items:center; gap:8px; font-weight:bold;">
          <input type="checkbox" checked> Recibir alertas de olvidos de marcaje en email
        </label>
        <label style="display:flex; align-items:center; gap:8px; font-weight:bold;">
          <input type="checkbox" checked> Permitir geolocalización en marcajes web
        </label>
        <label style="display:flex; align-items:center; gap:8px; font-weight:bold;">
          <input type="checkbox"> Modo nocturno en terminales de presencia
        </label>
        <button type="button" class="primary-action" style="max-width:180px;">Guardar cambios</button>
      </div>
    `;
    const btn = $("button", container);
    if (btn) {
      btn.addEventListener("click", () => {
        recordReceipt("Configuración Guardada", "Preferencias de control horario", "cronos");
        alert("Configuración guardada.");
        state.cronosView = "calendario";
        renderModulePortal(view);
      });
    }
  } else if (viewKey === "mensajes") {
    title.textContent = "Bandeja de Mensajería Interna";
    table.innerHTML = `
      <thead>
        <tr>
          <th>De</th>
          <th>Asunto</th>
          <th>Fecha</th>
          <th>Acciones</th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <td>RRHH (Admin)</td>
          <td>Recordatorio: Fichajes del mes de Junio 2026</td>
          <td>Hoy 09:30</td>
          <td><button class="primary-action" style="padding: 2px 8px; font-size:0.75rem;">Leer</button></td>
        </tr>
      </tbody>
    `;
  }

  const backBtn = document.createElement("button");
  backBtn.type = "button";
  backBtn.className = "cronos-btn-link";
  backBtn.textContent = "Volver al Calendario Principal";
  backBtn.addEventListener("click", () => {
    state.cronosView = "calendario";
    renderModulePortal(view);
  });
  container.append(backBtn);

  pane.append(container);
}

function openCronosRequestModal(permissionName, view) {
  const modalOverlay = document.createElement("div");
  modalOverlay.style.position = "fixed";
  modalOverlay.style.top = "0";
  modalOverlay.style.left = "0";
  modalOverlay.style.width = "100%";
  modalOverlay.style.height = "100%";
  modalOverlay.style.backgroundColor = "rgba(0,0,0,0.5)";
  modalOverlay.style.display = "flex";
  modalOverlay.style.justifyContent = "center";
  modalOverlay.style.alignItems = "center";
  modalOverlay.style.zIndex = "2000";

  const modalContainer = document.createElement("div");
  modalContainer.style.background = "#fff";
  modalContainer.style.padding = "24px";
  modalContainer.style.borderRadius = "8px";
  modalContainer.style.width = "100%";
  modalContainer.style.maxWidth = "480px";
  modalContainer.style.boxShadow = "0 4px 20px rgba(0,0,0,0.25)";
  modalContainer.style.display = "flex";
  modalContainer.style.flexDirection = "column";
  modalContainer.style.gap = "16px";
  modalContainer.style.fontFamily = "sans-serif";

  const modalTitle = document.createElement("h3");
  modalTitle.textContent = "Solicitar Permiso o Licencia";
  modalTitle.style.margin = "0";
  modalTitle.style.borderBottom = "2px solid var(--brand)";
  modalTitle.style.paddingBottom = "8px";
  modalTitle.style.fontSize = "1.2rem";
  modalContainer.append(modalTitle);

  const form = document.createElement("form");
  form.style.display = "flex";
  form.style.flexDirection = "column";
  form.style.gap = "12px";

  form.innerHTML = `
    <label style="display:flex; flex-direction:column; gap:4px; font-weight:bold; font-size:0.85rem;">
      Concepto de Permiso:
      <input type="text" value="${permissionName}" disabled style="padding:8px; border:1px solid #ccc; border-radius:4px; background:#eee; font-weight:bold;">
    </label>
    <div style="display:flex; gap:12px;">
      <label style="flex:1; display:flex; flex-direction:column; gap:4px; font-weight:bold; font-size:0.85rem;">
        Desde:
        <input type="date" required class="req-desde" style="padding:8px; border:1px solid #ccc; border-radius:4px;">
      </label>
      <label style="flex:1; display:flex; flex-direction:column; gap:4px; font-weight:bold; font-size:0.85rem;">
        Hasta:
        <input type="date" required class="req-hasta" style="padding:8px; border:1px solid #ccc; border-radius:4px;">
      </label>
    </div>
    <label style="display:flex; flex-direction:column; gap:4px; font-weight:bold; font-size:0.85rem;">
      Observaciones / Justificación:
      <textarea rows="3" placeholder="Comentario adicional para Recursos Humanos..." style="padding:8px; border:1px solid #ccc; border-radius:4px; font-family:sans-serif;"></textarea>
    </label>
    <div style="display:flex; justify-content:flex-end; gap:12px; margin-top:8px;">
      <button type="button" class="cancel-btn" style="padding:8px 16px; border:1px solid #777; border-radius:4px; background:#fff; cursor:pointer; font-weight:bold;">Cancelar</button>
      <button type="submit" class="submit-btn" style="padding:8px 16px; border:none; border-radius:4px; background:var(--brand); color:#fff; cursor:pointer; font-weight:bold;">Enviar Solicitud</button>
    </div>
  `;

  modalContainer.append(form);
  modalOverlay.append(modalContainer);
  document.body.append(modalOverlay);

  const reqDesde = $(".req-desde", form);
  reqDesde.focus();

  $(".cancel-btn", form).addEventListener("click", () => {
    modalOverlay.remove();
  });

  form.addEventListener("submit", (e) => {
    e.preventDefault();
    const desde = $(".req-desde", form).value;
    const hasta = $(".req-hasta", form).value;

    if (!state.cronosSubmittedRequests) {
      state.cronosSubmittedRequests = [];
    }

    const formatDateStr = (dStr) => {
      const parts = dStr.split("-");
      if (parts.length === 3) return `${parts[2]}/${parts[1]}/${parts[0]}`;
      return dStr;
    };

    const d1 = new Date(desde);
    const d2 = new Date(hasta);
    const diffTime = Math.abs(d2 - d1);
    const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24)) + 1;

    state.cronosSubmittedRequests.push({
      fechaSolicitud: new Date().toLocaleDateString("es-ES"),
      concepto: permissionName,
      desde: formatDateStr(desde),
      hasta: formatDateStr(hasta),
      duracion: isNaN(diffDays) ? "1 día" : `${diffDays} días`,
      estado: "Pendiente de validar"
    });

    recordReceipt("Solicitud de Permiso", `${permissionName} desde ${desde} hasta ${hasta}`, "cronos");
    alert("Solicitud de permiso enviada a Recursos Humanos correctamente.");
    modalOverlay.remove();

    state.cronosView = "permisos-pendientes-conceder";
    renderModulePortal(view);
  });
}

function renderCustomDietasApp(container, view) {
  if (state.dietasRole === undefined) {
    renderDietasRoleSelection(container, view);
  } else if (state.dietasRole === "empleado") {
    if (state.dietasScreen === undefined || state.dietasScreen === "menu-empleado") {
      renderDietasEmployeeMenu(container, view);
    } else if (state.dietasScreen === "menu-dietas") {
      renderDietasMainMenu(container, view);
    } else if (state.dietasScreen === "crear-dieta") {
      renderDietasCreateForm(container, view);
    }
  }
}

function renderDietasRoleSelection(container, view) {
  const wrapper = document.createElement("div");
  wrapper.style.fontFamily = "sans-serif";
  wrapper.style.background = "#fff";
  wrapper.style.minHeight = "480px";

  const header = document.createElement("div");
  header.style.display = "flex";
  header.style.flexDirection = "column";
  
  const redBar = document.createElement("div");
  redBar.style.background = "#8a0c10";
  redBar.style.color = "#fff";
  redBar.style.padding = "4px 12px";
  redBar.style.fontSize = "0.9rem";
  redBar.style.fontWeight = "bold";
  redBar.textContent = "DIPUTACIÓN DE GRANADA";
  
  const yellowBar = document.createElement("div");
  yellowBar.style.background = "#fc0";
  yellowBar.style.padding = "8px 12px";
  yellowBar.style.fontSize = "1.2rem";
  yellowBar.style.fontWeight = "italic";
  yellowBar.style.fontStyle = "italic";
  yellowBar.style.color = "#fff";
  yellowBar.style.textShadow = "1px 1px 2px rgba(0,0,0,0.5)";
  yellowBar.textContent = "Portal Interno";

  header.append(redBar, yellowBar);
  wrapper.append(header);

  const body = document.createElement("div");
  body.style.display = "flex";
  body.style.gap = "40px";
  body.style.padding = "24px";

  const sidebar = document.createElement("div");
  sidebar.style.display = "flex";
  sidebar.style.flexDirection = "column";
  sidebar.style.gap = "10px";
  sidebar.style.minWidth = "180px";

  const sidebarTitle = document.createElement("div");
  sidebarTitle.style.fontSize = "0.75rem";
  sidebarTitle.style.fontWeight = "bold";
  sidebarTitle.style.fontStyle = "italic";
  sidebarTitle.style.marginBottom = "8px";
  sidebarTitle.textContent = "ACCESOS IDENTIFICADOS";
  sidebar.append(sidebarTitle);

  const roles = ["RRHH", "INTERVENCIÓN", "ADMINISTRATIVO", "EMPLEADO", "RESPONSABLE CENTRO"];
  roles.forEach((role) => {
    const btn = document.createElement("button");
    btn.type = "button";
    btn.style.background = "#fc0";
    btn.style.border = "1px solid #c90";
    btn.style.padding = "10px 8px";
    btn.style.fontWeight = "bold";
    btn.style.fontSize = "0.8rem";
    btn.style.color = "#000";
    btn.style.cursor = "pointer";
    btn.style.textAlign = "center";
    btn.style.boxShadow = "inset 0 1px 0 rgba(255,255,255,0.4)";
    btn.textContent = role;
    
    btn.addEventListener("click", () => {
      if (role === "EMPLEADO") {
        state.dietasRole = "empleado";
        state.dietasScreen = "menu-empleado";
        renderModulePortal(view);
      } else {
        alert(`Acceso de rol '${role}' simulado. Utilice el rol EMPLEADO para tramitar dietas.`);
      }
    });
    sidebar.append(btn);
  });
  body.append(sidebar);

  const mainContent = document.createElement("div");
  mainContent.style.flex = "1";
  mainContent.style.display = "flex";
  mainContent.style.flexDirection = "column";
  mainContent.style.gap = "20px";

  const items = [
    { title: "Teléfonos de Empleados Alfabético", icon: "📞👥" },
    { title: "Teléfonos de Ayuntamientos", icon: "📞" },
    { title: "Secciones Sindicales", icon: "🟢🟢" },
    { title: "Tablón de Anuncios", icon: "📋" }
  ];

  items.forEach((item) => {
    const div = document.createElement("div");
    div.style.display = "flex";
    div.style.alignItems = "center";
    div.style.gap = "24px";
    div.style.padding = "12px 0";
    div.style.borderBottom = "1px solid #eee";

    const iconSpan = document.createElement("span");
    iconSpan.style.fontSize = "1.8rem";
    iconSpan.textContent = item.icon;

    const link = document.createElement("a");
    link.href = "#";
    link.style.color = "#333";
    link.style.textDecoration = "none";
    link.style.fontSize = "1rem";
    link.textContent = item.title;
    link.addEventListener("click", (e) => {
      e.preventDefault();
      alert(`Accediendo a: ${item.title}`);
    });

    div.append(iconSpan, link);
    mainContent.append(div);
  });

  body.append(mainContent);
  wrapper.append(body);
  container.append(wrapper);
}

function renderDietasEmployeeMenu(container, view) {
  const wrapper = document.createElement("div");
  wrapper.style.fontFamily = "sans-serif";
  wrapper.style.background = "#fff";
  wrapper.style.minHeight = "480px";

  const header = document.createElement("div");
  header.style.display = "flex";
  header.style.flexDirection = "column";
  
  const redBar = document.createElement("div");
  redBar.style.background = "#8a0c10";
  redBar.style.color = "#fff";
  redBar.style.padding = "4px 12px";
  redBar.style.fontSize = "0.9rem";
  redBar.style.fontWeight = "bold";
  redBar.textContent = "DIPUTACIÓN DE GRANADA";
  
  const yellowBar = document.createElement("div");
  yellowBar.style.background = "#fc0";
  yellowBar.style.padding = "8px 12px";
  yellowBar.style.fontSize = "1.2rem";
  yellowBar.style.fontWeight = "italic";
  yellowBar.style.fontStyle = "italic";
  yellowBar.style.color = "#fff";
  yellowBar.textContent = "Portal Interno - Empleado";

  header.append(redBar, yellowBar);
  wrapper.append(header);

  const body = document.createElement("div");
  body.style.display = "flex";
  body.style.gap = "40px";
  body.style.padding = "24px";

  const sidebar = document.createElement("div");
  sidebar.style.display = "flex";
  sidebar.style.flexDirection = "column";
  sidebar.style.gap = "10px";
  sidebar.style.minWidth = "180px";

  const options = [
    { label: "DIETAS Y GASTOS LOCOMOCION", action: "dietas" },
    { label: "FORMACION", action: "formacion" },
    { label: "CAMBIAR CONTRASEÑA", action: "password" },
    { label: "MIS DATOS", action: "datos" }
  ];

  options.forEach((opt) => {
    const btn = document.createElement("button");
    btn.type = "button";
    btn.style.background = "#fc0";
    btn.style.border = "1px solid #c90";
    btn.style.padding = "10px 8px";
    btn.style.fontWeight = "bold";
    btn.style.fontSize = "0.8rem";
    btn.style.color = "#000";
    btn.style.cursor = "pointer";
    btn.style.textAlign = "center";
    btn.textContent = opt.label;
    
    btn.addEventListener("click", () => {
      if (opt.action === "dietas") {
        state.dietasScreen = "menu-dietas";
        renderModulePortal(view);
      } else {
        alert(`Módulo '${opt.label}' en mantenimiento.`);
      }
    });
    sidebar.append(btn);
  });

  const btnSalir = document.createElement("button");
  btnSalir.type = "button";
  btnSalir.style.background = "#999";
  btnSalir.style.border = "1px solid #777";
  btnSalir.style.padding = "10px 8px";
  btnSalir.style.fontWeight = "bold";
  btnSalir.style.fontSize = "0.8rem";
  btnSalir.style.color = "#fff";
  btnSalir.style.cursor = "pointer";
  btnSalir.style.textAlign = "center";
  btnSalir.style.marginTop = "20px";
  btnSalir.textContent = "Volver a Selección de Roles";
  btnSalir.addEventListener("click", () => {
    state.dietasRole = undefined;
    state.dietasScreen = undefined;
    renderModulePortal(view);
  });
  sidebar.append(btnSalir);

  body.append(sidebar);

  const mainContent = document.createElement("div");
  mainContent.style.flex = "1";
  mainContent.style.display = "flex";
  mainContent.style.flexDirection = "column";
  mainContent.style.gap = "20px";

  const items = [
    { title: "Teléfonos del Personal Alfabético", icon: "📞👥" },
    { title: "Teléfonos de Ayuntamientos", icon: "📞" },
    { title: "Ayuda de Dietas y Gastos de Locomoción", icon: "💾" }
  ];

  items.forEach((item) => {
    const div = document.createElement("div");
    div.style.display = "flex";
    div.style.alignItems = "center";
    div.style.gap = "24px";
    div.style.padding = "12px 0";
    div.style.borderBottom = "1px solid #eee";

    const iconSpan = document.createElement("span");
    iconSpan.style.fontSize = "1.8rem";
    iconSpan.textContent = item.icon;

    const link = document.createElement("a");
    link.href = "#";
    link.style.color = "#333";
    link.style.textDecoration = "none";
    link.style.fontSize = "1rem";
    link.textContent = item.title;
    link.addEventListener("click", (e) => {
      e.preventDefault();
      alert(`Accediendo a: ${item.title}`);
    });

    div.append(iconSpan, link);
    mainContent.append(div);
  });

  body.append(mainContent);
  wrapper.append(body);
  container.append(wrapper);
}

function renderDietasMainMenu(container, view) {
  const wrapper = document.createElement("div");
  wrapper.style.fontFamily = "sans-serif";
  wrapper.style.background = "#fff";
  wrapper.style.padding = "20px";
  wrapper.style.minHeight = "480px";

  const headerDiv = document.createElement("div");
  headerDiv.style.display = "flex";
  headerDiv.style.justifyContent = "space-between";
  headerDiv.style.alignItems = "center";
  headerDiv.style.marginBottom = "24px";

  const iconHome = document.createElement("span");
  iconHome.textContent = "🏠";
  iconHome.style.fontSize = "1.5rem";
  iconHome.style.cursor = "pointer";
  iconHome.addEventListener("click", () => {
    state.dietasScreen = "menu-empleado";
    renderModulePortal(view);
  });

  const title = document.createElement("h2");
  title.style.margin = "0";
  title.style.fontSize = "1.3rem";
  title.style.fontWeight = "bold";
  title.textContent = ":: Menú de Dietas y Gastos de Locomoción ::";

  const iconLock = document.createElement("span");
  iconLock.textContent = "🔓";
  iconLock.style.fontSize = "1.5rem";

  headerDiv.append(iconHome, title, iconLock);
  wrapper.append(headerDiv);

  const blockPending = document.createElement("div");
  blockPending.style.border = "1px solid #777";
  blockPending.style.padding = "16px";
  blockPending.style.marginBottom = "20px";
  
  const blockPendingTitle = document.createElement("h3");
  blockPendingTitle.style.margin = "0 0 12px 0";
  blockPendingTitle.style.fontSize = "1rem";
  blockPendingTitle.style.fontStyle = "italic";
  blockPendingTitle.textContent = "Dietas Pendientes de Completar:";
  blockPending.append(blockPendingTitle);

  const pendingList = state.dietasSheets ? state.dietasSheets.filter(s => s.estado === "Borrador") : [];
  if (pendingList.length === 0) {
    const noReg = document.createElement("div");
    noReg.style.textAlign = "center";
    noReg.style.color = "#666";
    noReg.textContent = "No hay registros";
    blockPending.append(noReg);
  } else {
    const table = document.createElement("table");
    table.style.width = "100%";
    table.style.borderCollapse = "collapse";
    table.innerHTML = `
      <thead>
        <tr style="background:#eee;">
          <th style="border:1px solid #ccc; padding:6px; text-align:left;">ID</th>
          <th style="border:1px solid #ccc; padding:6px; text-align:left;">Motivo</th>
          <th style="border:1px solid #ccc; padding:6px; text-align:left;">Fecha</th>
          <th style="border:1px solid #ccc; padding:6px; text-align:right;">Importe</th>
          <th style="border:1px solid #ccc; padding:6px; text-align:center;">Acción</th>
        </tr>
      </thead>
      <tbody>
        ${pendingList.map(s => `
          <tr>
            <td style="border:1px solid #ccc; padding:6px;">${s.id}</td>
            <td style="border:1px solid #ccc; padding:6px;">${s.motivo}</td>
            <td style="border:1px solid #ccc; padding:6px;">${s.fecha}</td>
            <td style="border:1px solid #ccc; padding:6px; text-align:right; font-weight:bold;">${s.importe}</td>
            <td style="border:1px solid #ccc; padding:6px; text-align:center;">
              <button class="completar-btn" data-id="${s.id}" style="padding:2px 8px; background:var(--brand); color:#fff; border:none; border-radius:4px; cursor:pointer;">Completar</button>
            </td>
          </tr>
        `).join("")}
      </tbody>
    `;
    blockPending.append(table);

    const btns = blockPending.querySelectorAll(".completar-btn");
    btns.forEach(btn => {
      btn.addEventListener("click", (e) => {
        const id = e.target.dataset.id;
        alert(`Completando documento ID: ${id}`);
      });
    });
  }
  wrapper.append(blockPending);

  const blockAssignment = document.createElement("div");
  blockAssignment.style.textAlign = "center";
  blockAssignment.style.marginBottom = "20px";

  const assignTitle = document.createElement("h3");
  assignTitle.style.margin = "0 0 8px 0";
  assignTitle.style.fontSize = "0.9rem";
  assignTitle.style.fontWeight = "bold";
  assignTitle.textContent = "USTED ESTÁ ASIGNADO AL SIGUIENTE CENTRO Y UNIDAD";
  blockAssignment.append(assignTitle);

  const assignDetails = document.createElement("div");
  assignDetails.style.background = "#fffdd0";
  assignDetails.style.border = "1px solid #eee685";
  assignDetails.style.padding = "10px";
  assignDetails.style.fontSize = "0.85rem";
  assignDetails.innerHTML = `
    <div><strong>CENTRO:</strong> TRANSFORMACIÓN DIGITAL</div>
    <div style="margin-top:4px;"><strong>UNIDAD o SERVICIO:</strong> NUEVAS TECNOLOGIAS</div>
  `;
  blockAssignment.append(assignDetails);
  wrapper.append(blockAssignment);

  const newDocDiv = document.createElement("div");
  newDocDiv.style.textAlign = "center";
  newDocDiv.style.marginBottom = "24px";

  const newDocLink = document.createElement("button");
  newDocLink.type = "button";
  newDocLink.style.background = "#fff";
  newDocLink.style.border = "2px solid var(--brand)";
  newDocLink.style.color = "var(--brand)";
  newDocLink.style.padding = "8px 20px";
  newDocLink.style.borderRadius = "20px";
  newDocLink.style.fontWeight = "bold";
  newDocLink.style.cursor = "pointer";
  newDocLink.innerHTML = "➕ Nuevo Documento";
  newDocLink.addEventListener("click", () => {
    state.dietasScreen = "crear-dieta";
    renderModulePortal(view);
  });
  newDocDiv.append(newDocLink);
  wrapper.append(newDocDiv);

  const blockSubmitted = document.createElement("div");
  blockSubmitted.style.border = "1px solid #777";
  blockSubmitted.style.padding = "16px";
  blockSubmitted.style.marginBottom = "20px";

  const blockSubmittedTitle = document.createElement("h3");
  blockSubmittedTitle.style.margin = "0 0 12px 0";
  blockSubmittedTitle.style.fontSize = "1rem";
  blockSubmittedTitle.style.fontStyle = "italic";
  blockSubmittedTitle.textContent = "Control de Documentos: ::Pendientes de revisar por el administrativo de su Servicio::";
  blockSubmitted.append(blockSubmittedTitle);

  const submittedList = state.dietasSheets ? state.dietasSheets.filter(s => s.estado !== "Borrador") : [];
  if (submittedList.length === 0) {
    const noReg = document.createElement("div");
    noReg.style.textAlign = "center";
    noReg.style.color = "#666";
    noReg.textContent = "No hay registros";
    blockSubmitted.append(noReg);
  } else {
    const table = document.createElement("table");
    table.style.width = "100%";
    table.style.borderCollapse = "collapse";
    table.innerHTML = `
      <thead>
        <tr style="background:#eee;">
          <th style="border:1px solid #ccc; padding:6px; text-align:left;">ID</th>
          <th style="border:1px solid #ccc; padding:6px; text-align:left;">Motivo</th>
          <th style="border:1px solid #ccc; padding:6px; text-align:left;">Fecha</th>
          <th style="border:1px solid #ccc; padding:6px; text-align:right;">Importe</th>
          <th style="border:1px solid #ccc; padding:6px; text-align:center;">Estado</th>
        </tr>
      </thead>
      <tbody>
        ${submittedList.map(s => `
          <tr>
            <td style="border:1px solid #ccc; padding:6px;">${s.id}</td>
            <td style="border:1px solid #ccc; padding:6px;">${s.motivo}</td>
            <td style="border:1px solid #ccc; padding:6px;">${s.fecha}</td>
            <td style="border:1px solid #ccc; padding:6px; text-align:right; font-weight:bold;">${s.importe}</td>
            <td style="border:1px solid #ccc; padding:6px; text-align:center; color:var(--focus); font-weight:bold;">${s.estado}</td>
          </tr>
        `).join("")}
      </tbody>
    `;
    blockSubmitted.append(table);
  }
  wrapper.append(blockSubmitted);

  const blockSearch = document.createElement("div");
  blockSearch.style.border = "1px solid #777";
  blockSearch.style.padding = "16px";
  blockSearch.style.display = "flex";
  blockSearch.style.justifyContent = "space-between";
  blockSearch.style.alignItems = "center";
  blockSearch.style.fontSize = "0.9rem";

  blockSearch.innerHTML = `
    <div>
      <strong>Ver Desde:</strong>
      <select style="padding:2px 4px;"><option>20</option></select>
      <select style="padding:2px 4px;"><option>Jun</option></select>
      <select style="padding:2px 4px;"><option>2026</option></select>
      <span>📅</span>
    </div>
    <div>
      <strong>Ver Hasta:</strong>
      <select style="padding:2px 4px;"><option>20</option></select>
      <select style="padding:2px 4px;"><option>Jun</option></select>
      <select style="padding:2px 4px;"><option>2026</option></select>
      <span>📅</span>
    </div>
    <div class="buscar-btn-container" style="display:flex; align-items:center; gap:6px; cursor:pointer;">
      <span>🔭</span>
      <span style="font-weight:bold; color:blue; text-decoration:underline;">Buscar</span>
    </div>
  `;
  wrapper.append(blockSearch);

  const exitDiv = document.createElement("div");
  exitDiv.style.textAlign = "center";
  exitDiv.style.marginTop = "24px";
  const btnExit = document.createElement("button");
  btnExit.type = "button";
  btnExit.className = "primary-action";
  btnExit.textContent = "Volver a la Selección de Roles";
  btnExit.addEventListener("click", () => {
    state.dietasRole = undefined;
    state.dietasScreen = undefined;
    renderModulePortal(view);
  });
  exitDiv.append(btnExit);
  wrapper.append(exitDiv);

  container.append(wrapper);
}

function renderDietasCreateForm(container, view) {
  const wrapper = document.createElement("div");
  wrapper.style.fontFamily = "sans-serif";
  wrapper.style.background = "#fff";
  wrapper.style.padding = "20px";
  wrapper.style.minHeight = "480px";

  const headerDiv = document.createElement("div");
  headerDiv.style.display = "flex";
  headerDiv.style.justifyContent = "space-between";
  headerDiv.style.alignItems = "center";
  headerDiv.style.marginBottom = "20px";

  const iconHome = document.createElement("span");
  iconHome.textContent = "🏠";
  iconHome.style.fontSize = "1.5rem";
  iconHome.style.cursor = "pointer";
  iconHome.addEventListener("click", () => {
    state.dietasScreen = "menu-dietas";
    renderModulePortal(view);
  });

  const title = document.createElement("h2");
  title.style.margin = "0";
  title.style.fontSize = "1.3rem";
  title.style.fontWeight = "bold";
  title.textContent = ":: Añadir Dietas y Gastos de Locomoción ::";

  const iconLock = document.createElement("span");
  iconLock.textContent = "🔓";
  iconLock.style.fontSize = "1.5rem";

  headerDiv.append(iconHome, title, iconLock);
  wrapper.append(headerDiv);

  const metaBlock = document.createElement("div");
  metaBlock.style.border = "1px solid #777";
  metaBlock.style.padding = "10px 16px";
  metaBlock.style.marginBottom = "20px";
  metaBlock.style.fontSize = "0.9rem";
  metaBlock.innerHTML = `
    <div><strong>Documento:</strong> 81158 - NUEVAS TECNOLOGIAS</div>
    <div style="margin-top:4px;"><strong>Fecha:</strong> ${new Date().toLocaleDateString("es-ES")} ${new Date().toLocaleTimeString("es-ES")}</div>
  `;
  wrapper.append(metaBlock);

  const totalHeader = document.createElement("div");
  totalHeader.style.display = "flex";
  totalHeader.style.justifyContent = "flex-end";
  totalHeader.style.alignItems = "center";
  totalHeader.style.gap = "12px";
  totalHeader.style.marginBottom = "20px";
  totalHeader.style.fontSize = "1.2rem";
  totalHeader.style.fontWeight = "bold";
  totalHeader.style.color = "blue";
  
  const totalText = document.createElement("span");
  totalText.textContent = "Importe Total Acumulado:";
  const totalVal = document.createElement("span");
  totalVal.className = "total-acumulado-val";
  totalVal.textContent = "0.00 €";
  totalHeader.append(totalText, totalVal);
  wrapper.append(totalHeader);

  const form = document.createElement("form");
  form.style.display = "flex";
  form.style.flexDirection = "column";
  form.style.gap = "20px";

  const sectObligatory = document.createElement("fieldset");
  sectObligatory.style.border = "1px solid #ccc";
  sectObligatory.style.borderRadius = "4px";
  sectObligatory.style.padding = "16px";
  sectObligatory.innerHTML = `
    <legend style="font-weight:bold; padding:0 8px;">Datos Obligatorios:</legend>
    <div style="display:grid; grid-template-columns:1fr 1fr; gap:16px;">
      <label style="display:flex; flex-direction:column; gap:4px; font-weight:bold; font-size:0.85rem;">
        Fecha de Inicio:
        <input type="date" required class="form-fecha-inicio" style="padding:6px; border:1px solid #ccc; border-radius:4px;">
      </label>
      <label style="display:flex; flex-direction:column; gap:4px; font-weight:bold; font-size:0.85rem;">
        Hora Inicio:
        <input type="time" value="08:00" required class="form-hora-inicio" style="padding:6px; border:1px solid #ccc; border-radius:4px;">
      </label>
      <label style="display:flex; flex-direction:column; gap:4px; font-weight:bold; font-size:0.85rem;">
        Fecha de Fin:
        <input type="date" required class="form-fecha-fin" style="padding:6px; border:1px solid #ccc; border-radius:4px;">
      </label>
      <label style="display:flex; flex-direction:column; gap:4px; font-weight:bold; font-size:0.85rem;">
        Hora Fin:
        <input type="time" value="15:00" required class="form-hora-fin" style="padding:6px; border:1px solid #ccc; border-radius:4px;">
      </label>
      <label style="display:flex; flex-direction:column; gap:4px; font-weight:bold; font-size:0.85rem;">
        Motivo:
        <input type="text" placeholder="Ej. Instalación de red en sede comarcal" required class="form-motivo" style="padding:6px; border:1px solid #ccc; border-radius:4px;">
      </label>
      <label style="display:flex; flex-direction:column; gap:4px; font-weight:bold; font-size:0.85rem;">
        Localidades:
        <input type="text" placeholder="Ej. Motril, Salobreña" required class="form-localidades" style="padding:6px; border:1px solid #ccc; border-radius:4px;">
      </label>
    </div>
  `;
  form.append(sectObligatory);

  const sectDietas = document.createElement("fieldset");
  sectDietas.style.border = "1px solid #ccc";
  sectDietas.style.borderRadius = "4px";
  sectDietas.style.padding = "16px";
  sectDietas.innerHTML = `
    <legend style="font-weight:bold; padding:0 8px;">Calculo de Dietas:</legend>
    <div style="display:grid; grid-template-columns:1fr 1fr; gap:16px; align-items:center; margin-bottom:12px;">
      <label style="display:flex; flex-direction:column; gap:4px; font-weight:bold; font-size:0.85rem;">
        País:
        <select class="form-pais" style="padding:6px; border:1px solid #ccc; border-radius:4px;">
          <option value="España">España</option>
          <option value="Francia">Francia</option>
          <option value="Portugal">Portugal</option>
        </select>
      </label>
      <div style="display:flex; flex-direction:column; gap:4px; font-weight:bold; font-size:0.85rem;">
        <span>Nivel de Detalle:</span>
        <div style="display:flex; gap:12px; margin-top:6px;">
          <label style="font-weight:normal; display:flex; align-items:center; gap:4px;">
            <input type="radio" name="dietas-nivel" value="bajo"> Bajo
          </label>
          <label style="font-weight:normal; display:flex; align-items:center; gap:4px;">
            <input type="radio" name="dietas-nivel" value="medio" checked> Medio
          </label>
          <label style="font-weight:normal; display:flex; align-items:center; gap:4px;">
            <input type="radio" name="dietas-nivel" value="alto"> Alto
          </label>
        </div>
      </div>
    </div>
    
    <div style="display:flex; justify-content:flex-start; margin-bottom:16px;">
      <button type="button" class="calcular-dietas-btn" style="padding:6px 12px; background:#fc0; border:1px solid #c90; font-weight:bold; border-radius:4px; cursor:pointer;">Calcular Dietas</button>
    </div>

    <table style="width:100%; border-collapse:collapse; font-size:0.85rem; margin-top:8px;">
      <thead>
        <tr style="background:#eee;">
          <th style="border:1px solid #ccc; padding:6px; text-align:left;">Tipo De Dieta</th>
          <th style="border:1px solid #ccc; padding:6px; text-align:center;">Intervalo Correspondiente</th>
          <th style="border:1px solid #ccc; padding:6px; text-align:right;">Importe</th>
          <th style="border:1px solid #ccc; padding:6px; text-align:center;">Aceptar</th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <td style="border:1px solid #ccc; padding:6px;">MEDIA DIETA</td>
          <td style="border:1px solid #ccc; padding:6px; text-align:center;">Almuerzo</td>
          <td style="border:1px solid #ccc; padding:6px; text-align:right; font-weight:bold;">18.50 €</td>
          <td style="border:1px solid #ccc; padding:6px; text-align:center;">
            <input type="checkbox" class="aceptar-media-dieta">
          </td>
        </tr>
        <tr>
          <td style="border:1px solid #ccc; padding:6px;">DIETA COMPLETA</td>
          <td style="border:1px solid #ccc; padding:6px; text-align:center;">Manutención completa</td>
          <td style="border:1px solid #ccc; padding:6px; text-align:right; font-weight:bold;">37.40 €</td>
          <td style="border:1px solid #ccc; padding:6px; text-align:center;">
            <input type="checkbox" class="aceptar-dieta-completa">
          </td>
        </tr>
      </tbody>
    </table>
  `;
  form.append(sectDietas);

  const sectKm = document.createElement("fieldset");
  sectKm.style.border = "1px solid #ccc";
  sectKm.style.borderRadius = "4px";
  sectKm.style.padding = "16px";
  sectKm.innerHTML = `
    <legend style="font-weight:bold; padding:0 8px;">Calculo de Kilometraje:</legend>
    <div style="display:grid; grid-template-columns:1fr 1fr; gap:16px; margin-bottom:12px;">
      <label style="display:flex; flex-direction:column; gap:4px; font-weight:bold; font-size:0.85rem;">
        Con vehículo propio:
        <select class="form-vehiculo-propio" style="padding:6px; border:1px solid #ccc; border-radius:4px;">
          <option value="No">No</option>
          <option value="Si">Si</option>
        </select>
      </label>
    </div>
    
    <div class="vehiculo-propio-fields" style="display:none; flex-direction:column; gap:12px;">
      <div style="display:grid; grid-template-columns:1fr 1fr; gap:16px;">
        <label style="display:flex; flex-direction:column; gap:4px; font-weight:bold; font-size:0.85rem;">
          Salida (Origen):
          <select class="form-km-salida" style="padding:6px; border:1px solid #ccc; border-radius:4px;">
            <option value="Granada">Granada</option>
            <option value="Motril">Motril</option>
            <option value="Baza">Baza</option>
          </select>
        </label>
        <label style="display:flex; flex-direction:column; gap:4px; font-weight:bold; font-size:0.85rem;">
          Llegada (Destino):
          <select class="form-km-llegada" style="padding:6px; border:1px solid #ccc; border-radius:4px;">
            <option value="Motril">Motril</option>
            <option value="Granada">Granada</option>
            <option value="Almuñécar">Almuñécar</option>
          </select>
        </label>
      </div>

      <div style="display:flex; align-items:center; gap:20px; font-size:0.9rem; font-weight:bold; margin-top:8px;">
        <span>Kilómetros: <span class="km-number-display">70</span> km</span>
        <label style="display:flex; align-items:center; gap:6px;">
          Ajuste (km):
          <input type="number" value="0" class="form-km-ajuste" style="width:60px; padding:4px; border:1px solid #ccc; border-radius:4px;">
        </label>
        <span style="color:blue;">Importe Kilometraje: <span class="km-importe-display">18.20 €</span></span>
      </div>

      <div style="display:flex; justify-content:flex-start; margin-top:8px;">
        <button type="button" class="btn-anadir-ruta" style="padding:6px 12px; background:#fff; border:1px solid #777; border-radius:4px; font-weight:bold; cursor:pointer;">➕ Añadir ruta</button>
      </div>
    </div>
  `;
  form.append(sectKm);

  const sectOtrosMedios = document.createElement("fieldset");
  sectOtrosMedios.style.border = "1px solid #ccc";
  sectOtrosMedios.style.borderRadius = "4px";
  sectOtrosMedios.style.padding = "16px";
  sectOtrosMedios.innerHTML = `
    <legend style="font-weight:bold; padding:0 8px;">Otros Medios:</legend>
    <div style="display:grid; grid-template-columns:1fr 1fr 1fr; gap:16px; align-items:flex-end;">
      <label style="display:flex; flex-direction:column; gap:4px; font-weight:bold; font-size:0.85rem;">
        Otros Medios:
        <select class="form-otros-medios" style="padding:6px; border:1px solid #ccc; border-radius:4px;">
          <option value="">::Elija el medio::</option>
          <option value="Autobus">Autobús / Tren</option>
          <option value="Taxi">Taxi</option>
          <option value="Avion">Avión</option>
        </select>
      </label>
      <label style="display:flex; flex-direction:column; gap:4px; font-weight:bold; font-size:0.85rem;">
        Gastos Justificados:
        <input type="text" placeholder="Ej. Billete de autobús" class="form-gastos-justificados" style="padding:6px; border:1px solid #ccc; border-radius:4px;">
      </label>
      <label style="display:flex; flex-direction:column; gap:4px; font-weight:bold; font-size:0.85rem;">
        Importe (€):
        <input type="number" min="0" value="0" step="0.01" class="form-otros-medios-importe" style="padding:6px; border:1px solid #ccc; border-radius:4px;">
      </label>
    </div>
    
    <div style="display:flex; justify-content:flex-start; margin-top:12px;">
      <button type="button" class="btn-anadir-otro-medio" style="padding:6px 12px; background:#fff; border:1px solid #777; border-radius:4px; font-weight:bold; cursor:pointer;">➕ Añadir otro medio</button>
    </div>
  `;
  form.append(sectOtrosMedios);

  const sectOtrosGastos = document.createElement("fieldset");
  sectOtrosGastos.style.border = "1px solid #ccc";
  sectOtrosGastos.style.borderRadius = "4px";
  sectOtrosGastos.style.padding = "16px";
  sectOtrosGastos.innerHTML = `
    <legend style="font-weight:bold; padding:0 8px;">Otros Gastos:</legend>
    <div style="display:grid; grid-template-columns:2fr 1fr; gap:16px; align-items:flex-end;">
      <label style="display:flex; flex-direction:column; gap:4px; font-weight:bold; font-size:0.85rem;">
        Motivo Gasto:
        <input type="text" placeholder="Ej. Parking zona azul" class="form-otros-gastos-motivo" style="padding:6px; border:1px solid #ccc; border-radius:4px;">
      </label>
      <label style="display:flex; flex-direction:column; gap:4px; font-weight:bold; font-size:0.85rem;">
        Importe Gasto (€):
        <input type="number" min="0" value="0" step="0.01" class="form-otros-gastos-importe" style="padding:6px; border:1px solid #ccc; border-radius:4px;">
      </label>
    </div>

    <div style="display:flex; justify-content:flex-start; margin-top:12px;">
      <button type="button" class="btn-anadir-otro-gasto" style="padding:6px 12px; background:#fff; border:1px solid #777; border-radius:4px; font-weight:bold; cursor:pointer;">➕ Añadir gasto</button>
    </div>
  `;
  form.append(sectOtrosGastos);

  const actionDiv = document.createElement("div");
  actionDiv.style.display = "flex";
  actionDiv.style.justifyContent = "center";
  actionDiv.style.gap = "16px";
  actionDiv.style.marginTop = "24px";

  const btnCancel = document.createElement("button");
  btnCancel.type = "button";
  btnCancel.style.padding = "10px 24px";
  btnCancel.style.border = "1px solid #777";
  btnCancel.style.borderRadius = "4px";
  btnCancel.style.background = "#fff";
  btnCancel.style.fontWeight = "bold";
  btnCancel.style.cursor = "pointer";
  btnCancel.textContent = "Cancelar";
  btnCancel.addEventListener("click", () => {
    state.dietasScreen = "menu-dietas";
    renderModulePortal(view);
  });

  const btnSave = document.createElement("button");
  btnSave.type = "submit";
  btnSave.style.padding = "10px 24px";
  btnSave.style.border = "none";
  btnSave.style.borderRadius = "4px";
  btnSave.style.background = "orange";
  btnSave.style.color = "#000";
  btnSave.style.fontWeight = "bold";
  btnSave.style.cursor = "pointer";
  btnSave.textContent = "Guardar Documento";

  actionDiv.append(btnCancel, btnSave);
  form.append(actionDiv);
  wrapper.append(form);

  container.append(wrapper);

  const selectVehiculo = $(".form-vehiculo-propio", form);
  const vehiculoFields = $(".vehiculo-propio-fields", form);
  selectVehiculo.addEventListener("change", (e) => {
    if (e.target.value === "Si") {
      vehiculoFields.style.display = "flex";
    } else {
      vehiculoFields.style.display = "none";
    }
    recalculateTotal();
  });

  const checkMedia = $(".aceptar-media-dieta", form);
  const checkCompleta = $(".aceptar-dieta-completa", form);
  const inputAjuste = $(".form-km-ajuste", form);
  const inputOtrosMediosImp = $(".form-otros-medios-importe", form);
  const inputOtrosGastosImp = $(".form-otros-gastos-importe", form);

  const recalculateTotal = () => {
    let sum = 0;
    
    if (checkMedia.checked) sum += 18.50;
    if (checkCompleta.checked) sum += 37.40;

    if (selectVehiculo.value === "Si") {
      const baseKm = 70;
      const ajuste = parseFloat(inputAjuste.value) || 0;
      const totalKm = Math.max(0, baseKm + ajuste);
      $(".km-number-display", form).textContent = totalKm;
      
      const kmRate = 0.26;
      const kmCost = totalKm * kmRate;
      $(".km-importe-display", form).textContent = kmCost.toFixed(2) + " €";
      sum += kmCost;
    }

    const otherMed = parseFloat(inputOtrosMediosImp.value) || 0;
    sum += otherMed;

    const otherExp = parseFloat(inputOtrosGastosImp.value) || 0;
    sum += otherExp;

    totalVal.textContent = sum.toFixed(2) + " €";
    return sum;
  };

  [checkMedia, checkCompleta, inputAjuste, inputOtrosMediosImp, inputOtrosGastosImp].forEach(el => {
    el.addEventListener("input", recalculateTotal);
    el.addEventListener("change", recalculateTotal);
  });

  const btnCalcDietas = $(".calcular-dietas-btn", form);
  btnCalcDietas.addEventListener("click", () => {
    checkMedia.checked = true;
    recalculateTotal();
    alert("Cálculo de dietas sugerido: se ha seleccionado 'MEDIA DIETA' de acuerdo al horario.");
  });

  form.addEventListener("submit", (e) => {
    e.preventDefault();
    
    const finalAmount = recalculateTotal();
    const motivo = $(".form-motivo", form).value;

    if (!state.dietasSheets) {
      state.dietasSheets = [];
    }

    state.dietasSheets.push({
      id: (81158 + state.dietasSheets.length).toString(),
      motivo: motivo || "Comisión de servicio",
      fecha: new Date().toLocaleDateString("es-ES"),
      importe: finalAmount.toFixed(2) + " €",
      estado: "Pendiente de revisar"
    });

    recordReceipt("Dietas Registradas", `Documento guardado con importe ${finalAmount.toFixed(2)} €`, "dietas");
    alert("Documento de Dietas y Gastos de Locomoción guardado con éxito.");
    
    state.dietasScreen = "menu-dietas";
    renderModulePortal(view);
  });
}

function renderScreenNavigation(target, screens) {
  const nav = document.createElement("nav");
  nav.className = "screen-nav";
  nav.setAttribute("aria-label", "Pantallas del modulo");
  nav.replaceChildren(...screens.map((screenItem) => {
    const button = document.createElement("button");
    button.type = "button";
    button.textContent = screenItem.title || screenItem.id;
    button.setAttribute("aria-current", screenItem.id === state.activeScreen ? "page" : "false");
    button.addEventListener("click", () => setActiveScreen(screenItem.id));
    return button;
  }));
  target.append(nav);
  const activeButton = $(".screen-nav button[aria-current='page']", target);
  if (activeButton) {
    window.requestAnimationFrame(() => activeButton.scrollIntoView({ block: "nearest", inline: "center" }));
  }
}

function renderScreenWorkspace(target, screen, view) {
  const headers = screenHeaders(screen);
  const rows = screenRows(screen, view);
  const actions = screenActions(screen);

  // Cabecera compacta: titulo, descripcion y una sola accion primaria.
  target.append(screenHead(
    screen.title || MODULE_COPY[state.activeModule]?.[0] || "Pantalla VEC",
    screen.description || MODULE_COPY[state.activeModule]?.[1] || "Pantalla operativa del modulo.",
    actions,
  ));

  // Contadores de estado clicables que filtran la tabla (patron Factorial/Sesame/Concur).
  target.append(screenStateCounters(screen, rows, headers));

  if (isLeaveScreen(screen)) {
    target.append(leaveRequestPanel(screen, view));
  }

  if (isRPTScreen(screen)) {
    target.append(rptPositionPanel(screen));
  }

  if (screen.id === "admin.catalogos") {
    target.append(categoryCatalogPanel());
  }

  // Una sola tabla densa de trabajo con accion por fila.
  target.append(workTable(screen, headers, rows));

  // La ficha de pantalla (datos del estudio) queda plegada, fuera del flujo principal.
  target.append(screenMeta(screen));
}

function screenHead(title, subtitle, actions) {
  const head = document.createElement("div");
  head.className = "screen-head";
  const copy = document.createElement("div");
  copy.className = "screen-title";
  copy.innerHTML = `<h2></h2><p></p>`;
  $("h2", copy).textContent = title;
  $("p", copy).textContent = subtitle;
  const bar = document.createElement("div");
  bar.className = "screen-primary";
  // Solo la accion principal en cabecera; las demas viven por fila.
  actions.slice(0, 2).forEach((action, index) => {
    const button = document.createElement("button");
    button.type = "button";
    button.className = index === 0 ? "primary-action" : "quiet-action";
    button.textContent = action.label;
    button.addEventListener("click", () => handleModulePortalAction(action));
    bar.append(button);
  });
  head.append(copy, bar);
  return head;
}

const STATE_CHIP_TONE = [
  ["borrador|nueva|detectada|simulado|pendiente seleccion", "chip-slate"],
  ["pendiente|solicitad|subsanacion|en revision|en estudio|presentad|recibid|calculando|en plazo", "chip-amber"],
  ["bloque|error|excedida|rechazad|impugnad|caducad|denegad|degradad|vencid|con errores|con alertas|excluida|desestimada|retirad|caido", "chip-red"],
  ["aprobad|vigente|valid|firmad|publicad|cerrad|conciliad|estimada|admitida|saludable|activa|definitivo|pagad", "chip-green"],
];

function stateTone(stateText) {
  const text = String(stateText || "").toLowerCase();
  const match = STATE_CHIP_TONE.find(([pattern]) => new RegExp(pattern).test(text));
  return match ? match[1] : "chip-blue";
}

function screenStateCounters(screen, rows, headers) {
  const wrap = document.createElement("div");
  wrap.className = "screen-counters";
  wrap.setAttribute("role", "group");
  wrap.setAttribute("aria-label", "Filtrar por estado");
  const stateIndex = stateColumnIndex(headers);
  const states = Array.isArray(screen.states) && screen.states.length ? screen.states : [];
  const countFor = (label) =>
    stateIndex < 0 ? 0 : rows.filter((row) => matchesState(String(row[stateIndex] || ""), label)).length;

  const total = makeCounter("Todos", rows.length, state.screenStateFilter === "", "chip-blue", () =>
    setScreenStateFilter(""),
  );
  wrap.append(total);
  states.forEach((label) => {
    const counter = makeCounter(label, countFor(label), state.screenStateFilter === label, stateTone(label), () =>
      setScreenStateFilter(state.screenStateFilter === label ? "" : label),
    );
    wrap.append(counter);
  });
  return wrap;
}

function makeCounter(label, count, pressed, tone, onClick) {
  const button = document.createElement("button");
  button.type = "button";
  button.className = "screen-counter";
  button.dataset.tone = tone;
  button.setAttribute("aria-pressed", pressed ? "true" : "false");
  button.innerHTML = `<span></span><b></b>`;
  $("span", button).textContent = label;
  $("b", button).textContent = formatCount(count);
  button.addEventListener("click", onClick);
  return button;
}

function setScreenStateFilter(value) {
  state.screenStateFilter = value;
  if (state.portal) renderModulePortal(state.portal);
}

function stateStem(value) {
  // Primer token significativo, sin plural ni genero, para casar
  // "Pendientes" con "Pendiente validar RRHH" o "Validado" con "Validada".
  const first = String(value || "")
    .toLowerCase()
    .normalize("NFD")
    .replace(/[̀-ͯ]/g, "")
    .split(/\s+/)[0] || "";
  return first.replace(/(os|as|es|o|a|s)$/u, "");
}

function matchesState(rowState, filter) {
  const b = String(filter || "");
  if (!b.trim()) return true;
  const fa = stateStem(rowState);
  const fb = stateStem(filter);
  if (fa && fb && (fa.startsWith(fb) || fb.startsWith(fa))) return true;
  const a = rowState.toLowerCase();
  return a.includes(b.toLowerCase()) || b.toLowerCase().includes(a);
}

function stateColumnIndex(headers) {
  const idx = headers.findIndex((header) => /estado|situacion|flexib/i.test(header));
  return idx >= 0 ? idx : -1;
}

function workTable(screen, headers, allRows) {
  const stateIndex = stateColumnIndex(headers);
  const rows = state.screenStateFilter && stateIndex >= 0
    ? allRows.filter((row) => matchesState(String(row[stateIndex] || ""), state.screenStateFilter))
    : allRows;

  const wrap = document.createElement("section");
  wrap.className = "work-table";
  const header = document.createElement("div");
  header.className = "panel-header";
  header.innerHTML = `<div><h3></h3><span class="small-text"></span></div><button class="table-action" type="button">Exportar</button>`;
  $("h3", header).textContent = "Registros de trabajo";
  $(".small-text", header).textContent = state.screenStateFilter
    ? `Filtrado: ${state.screenStateFilter} (${rows.length})`
    : `${rows.length} registros - orden por estado y plazo`;
  $(".table-action", header).addEventListener("click", () => exportScreenRows(screen, headers, rows));

  const tableWrap = document.createElement("div");
  tableWrap.className = "table-wrap";
  const table = document.createElement("table");
  const actionLabel = (screen.actions || [])[0] || "Abrir";
  const cols = [...headers, "Accion"];
  table.innerHTML = `<thead><tr></tr></thead><tbody></tbody>`;
  const headRow = $("thead tr", table);
  cols.forEach((col) => {
    const th = document.createElement("th");
    th.scope = "col";
    th.textContent = col;
    headRow.append(th);
  });
  const tbody = $("tbody", table);
  if (!rows.length) {
    const tr = document.createElement("tr");
    const td = document.createElement("td");
    td.colSpan = cols.length;
    td.innerHTML = `<p class="empty-state">Sin registros para este estado. Ajusta el filtro de arriba.</p>`;
    tr.append(td);
    tbody.append(tr);
  }
  rows.forEach((row) => {
    const tr = document.createElement("tr");
    tr.tabIndex = 0;
    headers.forEach((_, index) => {
      const td = document.createElement("td");
      if (index === stateIndex) {
        const chip = document.createElement("span");
        chip.className = `status-chip ${stateTone(row[index])}`;
        chip.textContent = row[index] || "-";
        td.append(chip);
      } else {
        td.textContent = row[index] != null && row[index] !== "" ? row[index] : "-";
      }
      tr.append(td);
    });
    const actionTd = document.createElement("td");
    const button = document.createElement("button");
    button.type = "button";
    button.className = "row-action";
    button.textContent = actionLabel;
    button.addEventListener("click", (event) => {
      event.stopPropagation();
      handleScreenRowAction(screen, row, headers, actionLabel);
    });
    actionTd.append(button);
    tr.append(actionTd);
    tbody.append(tr);
  });
  tableWrap.append(table);
  wrap.append(header, tableWrap);
  return wrap;
}

function isLeaveScreen(screen) {
  return ["permisos.solicitudes", "permisos.vacaciones", "permisos.saldos"].includes(screen.id);
}

function isRPTScreen(screen) {
  return screen.id === "personal.puestos";
}

function rptPositionPanel() {
  const panel = document.createElement("section");
  panel.className = "leave-request-panel";
  const header = document.createElement("div");
  header.className = "panel-header";
  header.innerHTML = `<div><h3></h3><span class="small-text"></span></div>`;
  $("h3", header).textContent = "Actualizar puesto RPT";
  $(".small-text", header).textContent = "Alta o correccion auditada en el maestro Personal/RPT";

  const form = document.createElement("form");
  form.className = "flow-form leave-form";
  form.innerHTML = `
    <label>Codigo RPT
      <input name="code" value="999" required>
    </label>
    <label>Puesto
      <input name="name" value="Puesto RPT de prueba" required>
    </label>
    <label>Tp
      <select name="type">
        <option value="N">N - ordinario</option>
        <option value="S">S - singularizado</option>
        <option value="E">E - pendiente leyenda</option>
      </select>
    </label>
    <label>Ad
      <select name="administration">
        <option value="F">F</option>
        <option value="E">E</option>
        <option value="L">L</option>
      </select>
    </label>
    <label>Fp
      <select name="provision">
        <option value="C">C - concurso</option>
        <option value="L">L - libre designacion</option>
        <option value="I">I - pendiente leyenda</option>
        <option value="2A">2A - pendiente leyenda</option>
      </select>
    </label>
    <label>Grupo
      <input name="group" value="A1" required>
    </label>
    <label>Categoria
      <input name="category_code" value="TS1">
    </label>
    <label>CD
      <input name="destination_level" type="number" min="0" value="24">
    </label>
    <label>Estado
      <select name="state">
        <option value="Vigente">Vigente</option>
        <option value="Importado demo">Importado demo</option>
        <option value="Pendiente leyenda RPT">Pendiente leyenda RPT</option>
      </select>
    </label>
    <button class="primary-action" type="submit">Guardar puesto</button>
    <button class="quiet-action" type="button" data-rpt-delete>Borrar puesto</button>
  `;
  $("[data-rpt-delete]", form).addEventListener("click", () => handleRPTPositionDelete(form));
  form.addEventListener("submit", handleRPTPositionSubmit);
  panel.append(header, form);
  return panel;
}

async function handleRPTPositionSubmit(event) {
  event.preventDefault();
  if (state.rptSubmitting) return;
  const form = event.currentTarget;
  const data = new FormData(form);
  const code = String(data.get("code") || "").trim();
  const payload = {
    name: String(data.get("name") || "").trim(),
    dot: 1,
    type: String(data.get("type") || "").trim(),
    administration: String(data.get("administration") || "").trim(),
    provision: String(data.get("provision") || "").trim(),
    group: String(data.get("group") || "").trim(),
    category_code: String(data.get("category_code") || "").trim(),
    destination_level: Number(data.get("destination_level") || 0),
    state: String(data.get("state") || "").trim(),
    source: "ui_personal_rpt",
  };
  state.rptSubmitting = true;
  setStatus("Guardando puesto RPT", "loading");
  try {
    const result = await getData(`${PERSONAL_RPT_POSITIONS_API}/${encodeURIComponent(code)}`, {
      method: "PUT",
      headers: STAFF_JSON_HEADERS,
      body: JSON.stringify(payload),
    });
    recordReceipt("Puesto RPT guardado", `${result.position?.code || code} - ${result.receipt?.id || "auditoria"}`, "personal");
    await loadPortal();
    setStatus("Puesto RPT actualizado", "ready");
  } catch (error) {
    setStatus(error.message, "error");
  } finally {
    state.rptSubmitting = false;
  }
}

async function handleRPTPositionDelete(form) {
  if (state.rptSubmitting) return;
  const code = String(new FormData(form).get("code") || "").trim();
  if (!code) {
    setStatus("Codigo RPT obligatorio para borrar", "error");
    return;
  }
  state.rptSubmitting = true;
  setStatus("Borrando puesto RPT", "loading");
  try {
    const result = await getData(`${PERSONAL_RPT_POSITIONS_API}/${encodeURIComponent(code)}`, {
      method: "DELETE",
      headers: STAFF_HEADERS,
    });
    recordReceipt("Puesto RPT borrado", `${result.code || code} - ${result.receipt?.id || "auditoria"}`, "personal");
    await loadPortal();
    setStatus("Puesto RPT borrado", "ready");
  } catch (error) {
    setStatus(error.message, "error");
  } finally {
    state.rptSubmitting = false;
  }
}

function categoryCatalogPanel() {
  const panel = document.createElement("section");
  panel.className = "leave-request-panel";
  const header = document.createElement("div");
  header.className = "panel-header";
  header.innerHTML = `<div><h3></h3><span class="small-text"></span></div>`;
  $("h3", header).textContent = "Categoria profesional";
  $(".small-text", header).textContent = "Crear, modificar o borrar categoria comun para Bolsa, RPT y certificados";
  const form = document.createElement("form");
  form.className = "flow-form leave-form";
  form.innerHTML = `
    <label>Slug
      <input name="slug" value="nueva-categoria" required>
    </label>
    <label>Nombre
      <input name="name" value="Nueva categoria profesional" required>
    </label>
    <label>Area
      <select name="area">
        <option value="administracion_general">Administracion general</option>
        <option value="administracion_especial">Administracion especial</option>
      </select>
    </label>
    <label>Fuente
      <input name="source" value="VEC">
    </label>
    <label>Uso
      <input name="usage" value="Convocatorias, RPT y certificados">
    </label>
    <button class="primary-action" type="submit">Guardar categoria</button>
    <button class="quiet-action" type="button" data-category-delete>Borrar categoria</button>
  `;
  form.addEventListener("submit", handleCategorySubmit);
  $("[data-category-delete]", form).addEventListener("click", () => handleCategoryDelete(form));
  panel.append(header, form);
  return panel;
}

async function handleCategorySubmit(event) {
  event.preventDefault();
  if (state.categorySubmitting) return;
  const form = event.currentTarget;
  const data = new FormData(form);
  const slug = String(data.get("slug") || "").trim();
  const payload = {
    name: String(data.get("name") || "").trim(),
    area: String(data.get("area") || "").trim(),
    source: String(data.get("source") || "").trim(),
    usage: String(data.get("usage") || "").trim(),
    module_key: "bolsa",
    state: "Vigente",
  };
  state.categorySubmitting = true;
  setStatus("Guardando categoria", "loading");
  try {
    const result = await getData(`${PERSONAL_CATEGORIES_API}/${encodeURIComponent(slug)}`, {
      method: "PUT",
      headers: STAFF_JSON_HEADERS,
      body: JSON.stringify(payload),
    });
    recordReceipt("Categoria guardada", `${result.category?.slug || slug} - ${result.receipt?.id || "auditoria"}`, "administracion");
    await loadPortal();
    setStatus("Categoria actualizada", "ready");
  } catch (error) {
    setStatus(error.message, "error");
  } finally {
    state.categorySubmitting = false;
  }
}

async function handleCategoryDelete(form) {
  if (state.categorySubmitting) return;
  const slug = String(new FormData(form).get("slug") || "").trim();
  if (!slug) {
    setStatus("Slug obligatorio para borrar categoria", "error");
    return;
  }
  state.categorySubmitting = true;
  setStatus("Borrando categoria", "loading");
  try {
    const result = await getData(`${PERSONAL_CATEGORIES_API}/${encodeURIComponent(slug)}`, {
      method: "DELETE",
      headers: STAFF_HEADERS,
    });
    recordReceipt("Categoria borrada", `${result.slug || slug} - ${result.receipt?.id || "auditoria"}`, "administracion");
    await loadPortal();
    setStatus("Categoria borrada", "ready");
  } catch (error) {
    setStatus(error.message, "error");
  } finally {
    state.categorySubmitting = false;
  }
}

function leaveRequestPanel(screen, view) {
  const policies = (view.workspace?.cronos_leave_policies || []).filter((policy) => policy.request);
  const balanceByID = new Map((view.workspace?.cronos_permission_balances || []).map((item) => [item.id, item]));
  const defaultPolicyID = screen.id === "permisos.vacaciones" ? "vacaciones" : (policies[0]?.id || "");
  const panel = document.createElement("section");
  panel.className = "leave-request-panel";
  const header = document.createElement("div");
  header.className = "panel-header";
  header.innerHTML = `<div><h3></h3><span class="small-text"></span></div>`;
  $("h3", header).textContent = "Solicitar permiso";
  $(".small-text", header).textContent = "Registra asuntos propios, medico, vacaciones y permisos horarios contra saldo";

  const form = document.createElement("form");
  form.className = "flow-form leave-form";
  form.innerHTML = `
    <label>Empleado
      <input name="employee_id" value="EMP-0031" required>
    </label>
    <label>Tipo
      <select name="policy_id" required></select>
    </label>
    <label>Desde
      <input name="from" type="date" value="2026-06-26" required>
    </label>
    <label>Hasta
      <input name="to" type="date" value="2026-06-26" required>
    </label>
    <label>Cantidad
      <input name="amount" type="number" min="1" value="1" required>
      <span class="field-hint">Dias para permisos diarios; minutos para permisos horarios.</span>
    </label>
    <label>Motivo
      <input name="reason" value="${screen.id === "permisos.vacaciones" ? "Vacaciones" : "Solicitud Cronos"}" required>
    </label>
    <label>Justificante
      <input name="document_ref" placeholder="CSV/DOC si procede">
    </label>
    <button class="primary-action" type="submit">Solicitar</button>
  `;
  const select = $("select[name='policy_id']", form);
  policies.forEach((policy) => {
    const balance = balanceByID.get(policy.id);
    const option = document.createElement("option");
    option.value = policy.id;
    option.textContent = `${policy.name} - resta ${balance?.remaining || policy.annual_allowance || "-"}`;
    option.dataset.requiresDocument = policy.requires_document ? "true" : "false";
    if (policy.id === defaultPolicyID) option.selected = true;
    select.append(option);
  });
  const documentInput = $("input[name='document_ref']", form);
  const syncDocumentRequirement = () => {
    const selected = select.selectedOptions[0];
    const required = selected?.dataset.requiresDocument === "true";
    documentInput.required = required;
    documentInput.placeholder = required ? "Obligatorio para este permiso" : "CSV/DOC si procede";
  };
  select.addEventListener("change", syncDocumentRequirement);
  syncDocumentRequirement();
  form.addEventListener("submit", handleLeaveRequestSubmit);
  panel.append(header, form);
  return panel;
}

async function handleLeaveRequestSubmit(event) {
  event.preventDefault();
  if (state.leaveSubmitting) return;
  const form = event.currentTarget;
  const data = new FormData(form);
  const payload = {
    employee_id: String(data.get("employee_id") || "").trim(),
    policy_id: String(data.get("policy_id") || "").trim(),
    from: String(data.get("from") || "").trim(),
    to: String(data.get("to") || "").trim(),
    amount: Number(data.get("amount") || 0),
    reason: String(data.get("reason") || "").trim(),
    document_ref: String(data.get("document_ref") || "").trim(),
  };
  state.leaveSubmitting = true;
  setStatus("Registrando solicitud Cronos", "loading");
  try {
    const result = await getData(CRONOS_LEAVE_API, {
      method: "POST",
      headers: STAFF_JSON_HEADERS,
      body: JSON.stringify(payload),
    });
    recordReceipt("Solicitud permiso", `${payload.employee_id} - ${payload.policy_id} - ${result.receipt?.id || "recibo"}`, "permisos");
    await loadPortal();
    setStatus("Solicitud Cronos registrada", "ready");
  } catch (error) {
    setStatus(error.message, "error");
  } finally {
    state.leaveSubmitting = false;
  }
}

function screenMeta(screen) {
  const details = document.createElement("details");
  details.className = "screen-meta-toggle";
  const summary = document.createElement("summary");
  summary.textContent = "Ficha de la pantalla (datos, validaciones e integraciones)";
  const body = document.createElement("div");
  body.className = "screen-meta-body";
  const cards = [
    ["Datos visibles", formatList(screen.fields?.map((field) => field.label || field.key))],
    ["Estados del flujo", formatList(screen.states)],
    ["Validaciones", formatList(screen.validations)],
    ["Integraciones", formatList(screen.integrations)],
    ["Criterio de terminado", screen.done_criteria || "Estado persistido, recibo y auditoria visibles."],
  ];
  cards.forEach(([title, detail]) => {
    const card = document.createElement("article");
    card.innerHTML = `<strong></strong><span></span>`;
    $("strong", card).textContent = title;
    $("span", card).textContent = detail;
    body.append(card);
  });
  details.append(summary, body);
  return details;
}

function exportScreenRows(screen, headers, rows) {
  const csv = [
    [...headers, "pantalla"].join(";"),
    ...rows.map((row) => [...headers.map((_, i) => row[i] ?? ""), screen.title || screen.id]
      .map((value) => `"${String(value || "").replaceAll('"', '""')}"`).join(";")),
  ].join("\n");
  const blob = new Blob([csv], { type: "text/csv;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = `vec-${(screen.id || "pantalla").replace(/\W+/g, "-")}-${new Date().toISOString().slice(0, 10)}.csv`;
  document.body.append(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
  recordReceipt("Exportacion pantalla", `${rows.length} filas de ${screen.title || screen.id}`, state.activeModule);
  setStatus(`Exportadas ${rows.length} filas`, "ready");
}

function handleScreenRowAction(screen, row, headers, actionLabel) {
  const ref = row[0] || screen.title || screen.id;
  const nextState = nextStateForAction(actionLabel);
  recordReceipt(actionLabel, `${ref} -> ${nextState}`, state.activeModule);
  setStatus(`${actionLabel}: ${ref}`, "ready");
  if (state.portal) renderFlowPanel();
}

function screenActions(screen) {
  const actions = Array.isArray(screen.actions) && screen.actions.length ? screen.actions : ["Registrar accion", "Exportar"];
  return actions.slice(0, 4).map((action) => ({ label: String(action) }));
}

function screenHeaders(screen) {
  if (screen.id === "personal.puestos") {
    return ["Codigo", "Puesto", "Tp", "Ad", "Fp", "Grupo", "Estado"];
  }
  if (screen.id === "cronos.dashboard" || screen.id === "cronos.fichajes") {
    return ["Empleado", "Fecha", "Teoricas", "Trabajadas", "Saldo", "Estado"];
  }
  if (screen.id === "cronos.incidencias") {
    return ["Empleado", "Fecha", "Tipo", "Impacto", "Severidad", "Estado"];
  }
  if (screen.id === "horarios.reducciones") {
    return ["Empleado", "Edad", "Reduccion", "Objetivo", "Trabajado", "Estado"];
  }
  if (screen.id === "permisos.solicitudes") {
    return ["Solicitud", "Empleado", "Tipo", "Desde", "Cantidad", "Estado"];
  }
  if (screen.id === "permisos.vacaciones") {
    return ["Solicitud", "Empleado", "Periodo", "Dias", "Saldo", "Estado"];
  }
  if (screen.id === "permisos.saldos") {
    return ["Permiso", "Solicitable", "Maximo", "Solicitado", "Restante", "Estado"];
  }
  if (screen.id === "bolsa.convocatorias") {
    return ["Categoria", "Area", "Fuente", "Regla baremo", "Uso", "Estado"];
  }
  if (screen.id === "bolsa.meritos" || screen.id === "bolsa.autobaremo") {
    return ["Regla", "Seccion", "Unidad", "Puntos", "Fuente", "Estado"];
  }
  if (screen.id === "admin.usuarios_roles") {
    return ["Rol", "Ambito", "Usuarios", "Modulos", "Permisos clave", "Estado"];
  }
  if (screen.id === "admin.catalogos") {
    return ["Catalogo", "Codigo", "Descripcion", "Fuente", "Modulo", "Estado"];
  }
  const fields = Array.isArray(screen.fields) ? screen.fields : [];
  const headers = fields.slice(0, 6).map((field) => field.label || field.key || "Dato");
  return headers.length ? headers : ["Referencia", "Estado", "Responsable", "Accion"];
}

function getPersonalCatalog(view) {
  return view?.personalCatalog || state.personalCatalog || {};
}

function screenRows(screen, view) {
  if (screen.id === "personal.puestos") {
    const positions = getPersonalCatalog(view).positions?.items || view.workspace?.rpt_position_samples || [];
    return positions.map((item) => [
      item.code,
      item.name,
      item.type || item.tp,
      item.administration || item.ad,
      item.provision || item.fp,
      item.group,
      item.state || item.coverage,
    ]);
  }
  if (screen.id === "cronos.dashboard" || screen.id === "cronos.fichajes") {
    return (view.workspace?.cronos_timecards || []).map((item) => [
      item.employee,
      item.date,
      item.theoretical,
      item.worked,
      item.balance,
      item.state,
    ]);
  }
  if (screen.id === "cronos.incidencias") {
    return (view.workspace?.cronos_timecards || [])
      .filter((item) => Array.isArray(item.incidents) && item.incidents.length)
      .map((item) => {
        const incident = item.incidents[0] || {};
        return [
          item.employee,
          item.date,
          incident.label || item.title,
          item.balance,
          incident.severity || "warning",
          item.state,
        ];
      });
  }
  if (screen.id === "rutas.mapa_provincia" || screen.id === "rutas.kilometraje") {
    return (view.workspace?.province_routes || []).slice(0, 6).map((route) => [
      route.id,
      route.from,
      route.to,
      `${formatPoints(route.km_one_way)} km`,
      `${route.estimated_minutes} min`,
      route.allowance,
    ]);
  }
  if (screen.id === "horarios.reducciones") {
    return (view.workspace?.cronos_reductions || []).map((item) => [
      item.employee,
      item.age,
      item.daily_reduction,
      item.target,
      item.worked,
      item.state,
    ]);
  }
  if (screen.id === "horarios.perfiles") {
    return (view.workspace?.schedule_profiles || []).map((profile) => [
      profile.id,
      profile.name,
      profile.flexible ? "Flexible" : "Sin flexibilidad",
      profile.entry_window || profile.daily_reduction || profile.core_time || "-",
      profile.weekly_hours || "-",
      "Vigente",
    ]);
  }
  if (screen.id === "permisos.solicitudes") {
    return (view.workspace?.cronos_leave_requests || []).map((item) => [
      item.id,
      item.employee,
      item.name || item.policy_id,
      `${item.from} -> ${item.to}`,
      item.amount,
      item.state,
    ]);
  }
  if (screen.id === "permisos.vacaciones") {
    const vacationRequests = (view.workspace?.cronos_leave_requests || []).filter((item) => item.policy_id === "vacaciones");
    if (vacationRequests.length) {
      return vacationRequests.map((item) => [
        item.id,
        item.employee,
        `${item.from} -> ${item.to}`,
        item.amount,
        "Solicitud registrada",
        item.state,
      ]);
    }
    return (view.workspace?.cronos_permission_balances || []).filter((item) => item.id === "vacaciones").map((item) => [
      item.name,
      item.employee,
      "Sin solicitud activa",
      item.requested || "-",
      item.remaining || "-",
      "Saldo vigente",
    ]);
  }
  if (screen.id === "permisos.saldos") {
    return (view.workspace?.cronos_permission_balances || []).slice(0, 8).map((item) => [
      item.name,
      item.request ? "Solicitable" : "No solicitable",
      item.max || "-",
      item.requested || "-",
      item.remaining || "-",
      "Saldo vigente",
    ]);
  }
  if (screen.id === "bolsa.convocatorias") {
    const categories = getPersonalCatalog(view).categories?.items || view.workspace?.professional_categories || [];
    return categories.slice(0, 14).map((item) => [
      item.name,
      item.area,
      item.source,
      "Misma/otra categoria",
      item.usage,
      item.state,
    ]);
  }
  if (screen.id === "bolsa.meritos" || screen.id === "bolsa.autobaremo") {
    return (view.workspace?.bolsa_category_rules || []).map((rule) => [
      rule.label,
      rule.section,
      "meses",
      `${formatPoints(rule.points_per_month)} pt/mes`,
      rule.source,
      "Vigente",
    ]);
  }
  if (screen.id === "admin.usuarios_roles") {
    return (view.workspace?.access_roles || []).map((role) => [
      role.label || role.id,
      role.scope,
      formatCount(role.users_count || 0),
      formatList(role.modules),
      formatList(role.key_permissions),
      role.state || "Activo",
    ]);
  }
  if (screen.id === "admin.catalogos") {
    const catalogEntries = getPersonalCatalog(view).catalogs || view.workspace?.rpt_contract_types || [];
    const categoryEntries = getPersonalCatalog(view).categories?.items || view.workspace?.professional_categories || [];
    const rptRows = catalogEntries.map((item) => [
      item.catalog,
      item.code,
      item.label || item.name,
      item.source,
      item.module_key || "personal",
      item.state || "Vigente",
    ]);
    const categoryRows = categoryEntries.map((item) => [
      item.catalog || "categoria_profesional",
      item.slug,
      item.name,
      item.source,
      item.module_key || "bolsa",
      item.state || "Vigente",
    ]);
    return [...rptRows, ...categoryRows];
  }
  if (screen.id === "nominas.cierre" || screen.id === "nominas.retribuciones") {
    return (view.workspace?.payroll_run?.concepts || []).map((item) => [
      item.code,
      item.label,
      item.records,
      `${formatPoints(item.amount)} EUR`,
      view.workspace?.payroll_run?.period || "-",
      view.workspace?.payroll_run?.state || "-",
    ]);
  }
  if (screen.id === "nominas.incidencias" || screen.id === "nominas.integraciones") {
    return (view.workspace?.payroll_run?.incidents || view.workspace?.payroll_run?.dependencies || []).map((item) => [
      item.id || item.module_key,
      item.employee || item.label,
      item.source_module || item.state,
      item.summary || `${item.records || 0} registros`,
      item.flow_state || item.next_action || "-",
      item.next_action || item.state || "-",
    ]);
  }
  const rows = filteredRows().filter((row) => row.modules.includes(state.activeModule)).slice(0, 8);
  if (!rows.length) return [["Sin registros", screen.title || "-", "Preparado", "Sin bloqueo", "-", "Abrir"]];
  return rows.map((row) => [
    row.expediente,
    row.candidate,
    row.state,
    row.deadline,
    row.points,
    row.action,
  ]);
}

function validationRows(screen) {
  const validations = Array.isArray(screen.validations) && screen.validations.length
    ? screen.validations
    : ["Permiso de acceso", "Datos obligatorios", "Auditoria"];
  return validations.slice(0, 8).map((item) => [String(item), "Control visible", "Resolver antes de cerrar"]);
}

function formatList(items) {
  const values = (items || []).filter(Boolean);
  return values.length ? values.join(", ") : "-";
}

function modulePortalHeader(title, subtitle, actions = []) {
  const header = document.createElement("div");
  header.className = "module-portal-header";
  const copy = document.createElement("div");
  copy.innerHTML = `<p class="eyebrow"></p><h2></h2><span class="small-text"></span>`;
  $(".eyebrow", copy).textContent = "Portal de modulo";
  $("h2", copy).textContent = title;
  $(".small-text", copy).textContent = subtitle;
  const actionBar = document.createElement("div");
  actionBar.className = "module-actions";
  actions.forEach((action, index) => {
    const actionDef = typeof action === "string" ? { label: action } : action;
    const button = document.createElement("button");
    button.type = "button";
    button.className = index === 0 ? "primary-action" : "quiet-action";
    button.textContent = actionDef.label;
    button.addEventListener("click", () => handleModulePortalAction(actionDef));
    actionBar.append(button);
  });
  header.append(copy, actionBar);
  return header;
}

async function handleModulePortalAction(actionDef) {
  const moduleID = state.activeModule;
  const config = MODULE_FLOW_CONFIG[moduleID];
  if (config) {
    const data = Object.fromEntries(config.fields.map((field) => [field.name, field.value || field.options?.[0]?.[0] || ""]));
    if (/certificado/i.test(actionDef.label)) {
      data.scope = "Servicios prestados";
      data.state = "Pendiente firma";
    }
    if (/cerrar/i.test(actionDef.label)) data.state = "Listo para cierre";
    if (/permiso/i.test(actionDef.label)) data.kind = "Asuntos propios";
    if (/ruta|km/i.test(actionDef.label)) data.state = "Ruta calculada";
    await createModuleFlowRecord(config, data, actionDef.label);
    return;
  }
  const row = selectedRow();
  if (row) {
    await handleRowAction({ ...row, action: actionDef.label });
    return;
  }
  recordReceipt(actionDef.label, "Accion de modulo sin registro seleccionado", moduleID);
  renderPortal(state.portal);
}

function renderPersonalPortal(target) {
  target.append(modulePortalHeader(
    "Gestion integral de Personal",
    "Maestro de empleados, puestos, situaciones, antiguedad, servicios prestados y certificados.",
    ["Nuevo expediente", "Generar certificado", "Exportar servicios"],
  ));
  target.append(portalGrid([
    ["Expediente empleado", "Alta, puesto, unidad, regimen, situacion administrativa e historial."],
    ["Antiguedad y trienios", "Periodos reconocidos, trienios activos y fecha de proximo vencimiento."],
    ["Servicios prestados", "Calculo automatico para certificados y consumo por Bolsa/meritos."],
    ["Certificados", "Emision de servicios prestados con referencia, CSV/firma cuando haya adaptador."],
  ]));
  target.append(portalTable("Expedientes de personal", ["Empleado", "Ambito", "Estado", "Accion"], [
    ["EMP-0042", "Puesto y unidad organica", "Pendiente validar RRHH", "Validar puesto"],
    ["EMP-0031", "Servicios prestados", "Disponible para certificado", "Generar certificado"],
    ["EMP-0088", "Antiguedad y trienios", "Calculado", "Recalcular"],
  ]));
}

function renderNominasPortal(target) {
  target.append(modulePortalHeader(
    "Gestion de Nominas",
    "Borrador mensual, conceptos retributivos, trienios, deducciones, incidencias, cierre y certificados.",
    ["Precalcular junio", "Abrir incidencias", "Cerrar periodo"],
  ));
  target.append(portalGrid([
    ["Periodo", "Junio 2026 - cierre ordinario 25/06"],
    ["Conceptos", "Sueldo, trienios, complemento destino/especifico, productividad, atrasos."],
    ["Deducciones", "IRPF, Seguridad Social, anticipos, reintegros y embargos."],
    ["Cruces", "Cronos aporta reducciones/ausencias; Dietas aporta liquidaciones aprobadas."],
  ]));
  target.append(portalTable("Borrador de nomina", ["Concepto", "Tipo", "Base", "Estado"], [
    ["Sueldo grupo A2", "Retribucion basica", "Cotiza / IRPF", "Calculado"],
    ["Trienios A2", "Antiguedad", "Cotiza / IRPF", "Desde Personal"],
    ["Complemento destino 22", "Complementaria", "Cotiza / IRPF", "Calculado"],
    ["Reduccion 64 anos", "Incidencia Cronos", "A revisar", "Pendiente"],
  ]));
}

function renderCronosPortal(target, view) {
  const summary = view.workspace?.cronos_daily_summary || {};
  const profiles = view.workspace?.schedule_profiles || [];
  target.append(modulePortalHeader(
    "Cronos: horarios, fichajes y permisos",
    "Control horario por perfil de puesto, flexibilidad, puestos sin flexibilidad y reducciones 63/64.",
    ["Registrar incidencia", "Visualizar horario", "Solicitar permiso"],
  ));
  target.append(portalGrid([
    ["Dia seleccionado", `Teoricas ${summary.theoretical || "-"} · trabajadas ${summary.worked || "-"} · teletrabajo ${summary.telework || "-"}`],
    ["Saldo periodo", `${summary.period_balance || "-"} desde ${summary.period_from || "-"} hasta ${summary.period_to || "-"}`],
    ["Puestos sin flexibilidad", "Atencion directa y servicios con cobertura presencial obligatoria."],
    ["Prejubilacion", "63 anos: 1h menos diaria · 64 anos: 2h menos diarias."],
  ]));
  target.append(portalTable("Perfiles horarios", ["Perfil", "Flexibilidad", "Tramo / reduccion", "Horas"], profiles.map((profile) => [
    profile.name,
    profile.flexible ? "Flexible" : "Sin flexibilidad",
    profile.entry_window || profile.daily_reduction || profile.core_time || "-",
    profile.weekly_hours || "-",
  ])));
  target.append(portalTable("Permisos y licencias", ["Permiso", "Max.", "Solic.", "Resta"], (view.workspace?.cronos_permission_balances || []).slice(0, 8).map((item) => [
    item.name,
    item.max || "-",
    item.requested || "-",
    item.remaining || "-",
  ])));
}

function renderDietasPortal(target, view) {
  const routes = view.workspace?.province_routes || [];
  target.append(modulePortalHeader(
    "Dietas: comisiones y kilometraje provincial",
    "Gestion de comisiones de servicio, rutas, kilometros, medias dietas, dietas completas y liquidaciones.",
    ["Nueva comision", "Calcular ruta", "Liquidar aprobadas"],
  ));
  target.append(portalGrid([
    ["Rutas provincia", `${routes.length} rutas de referencia cargadas`],
    ["Politica", "Media dieta / dieta completa segun horario, ruta y justificante."],
    ["Validacion", "Kilometros, motivo, parada intermedia, vehiculo y autorizacion."],
    ["Liquidacion", "Expediente aprobado listo para nomina/contabilidad."],
  ]));
  target.append(portalTable("Mapa de kilometraje", ["Origen", "Destino", "Km ida", "Dieta"], routes.map((route) => [
    route.from,
    route.to,
    `${formatPoints(route.km_one_way)} km`,
    route.allowance,
  ])));
  target.append(portalTable("Comisiones pendientes", ["Expediente", "Ruta", "Estado", "Accion"], filteredRows().filter((row) => row.modules.includes("dietas")).map((row) => [
    row.expediente,
    row.scope,
    row.state,
    row.action,
  ])));
}

function renderGenericModulePortal(target) {
  target.append(modulePortalHeader(
    MODULE_COPY[state.activeModule]?.[0] || "Modulo VEC",
    MODULE_COPY[state.activeModule]?.[1] || "Gestion operativa del modulo.",
    ["Registrar accion", "Exportar"],
  ));
}

function portalGrid(items) {
  const grid = document.createElement("div");
  grid.className = "portal-grid";
  grid.replaceChildren(...items.map(([title, detail]) => {
    const item = document.createElement("article");
    item.innerHTML = `<strong></strong><span></span>`;
    $("strong", item).textContent = title;
    $("span", item).textContent = detail;
    return item;
  }));
  return grid;
}

function portalTable(title, headers, rows) {
  const wrap = document.createElement("section");
  wrap.className = "portal-table";
  const heading = document.createElement("h3");
  heading.textContent = title;
  const table = document.createElement("table");
  table.innerHTML = `<thead><tr></tr></thead><tbody></tbody>`;
  const tr = $("thead tr", table);
  headers.forEach((header) => {
    const th = document.createElement("th");
    th.textContent = header;
    tr.append(th);
  });
  const tbody = $("tbody", table);
  (rows.length ? rows : [["Sin datos", "-", "-", "-"]]).forEach((row) => {
    const bodyRow = document.createElement("tr");
    headers.forEach((_, index) => {
      const td = document.createElement("td");
      td.textContent = row[index] || "-";
      bodyRow.append(td);
    });
    tbody.append(bodyRow);
  });
  wrap.append(heading, table);
  return wrap;
}

function renderDetail(view, row) {
  updateTabSelection();
  if (!row) {
    setText("#detail-title", "Sin seleccion");
    renderList(".alert-list", [["Sin resultados", "Ajusta filtros o busqueda para ver expedientes"]]);
    renderList(".doc-list", []);
    renderList(".timeline-list", state.actionLog.map((entry) => [entry.action, entry.detail]));
    return;
  }
  setText("#detail-title", row.expediente);
  const detailStatus = $(".right-column .detail-panel .panel-header .status-chip");
  if (detailStatus) detailStatus.textContent = row.state;
  const values = $$(".detail-grid strong");
  const detailValues = [row.candidate, row.points, row.document || view.routes[0] || BOLSA_PORTAL_API, new Date().toLocaleString("es-ES")];
  detailValues.forEach((value, index) => { if (values[index]) values[index].textContent = value; });
  renderTabContent(row);
  renderList(".doc-list", row.documents);
  renderList(".timeline-list", [...row.timeline, ...state.actionLog.slice(-4).reverse().map((entry) => [entry.action, entry.detail])]);
}

function renderTabContent(row) {
  const tab = state.activeTab;
  if (tab === "meritos") {
    renderList(".alert-list", row.merits.length ? row.merits : [["Sin meritos", "No hay desglose cargado para esta seleccion"]]);
    return;
  }
  if (tab === "docs") {
    renderList(".alert-list", row.documents.length ? row.documents : [["Sin documentos", "No hay evidencias asociadas"]]);
    return;
  }
  if (tab === "auditoria") {
    renderList(".alert-list", row.timeline.length ? row.timeline : [["Sin auditoria", "No hay trazas para esta seleccion"]]);
    return;
  }
  renderList(".alert-list", row.alerts.length ? row.alerts : [["VEC conectado", "GET /api/vec y modulo Bolsa disponibles"]]);
}

function renderList(selector, rows) {
  const target = $(selector);
  if (!target) return;
  if (!rows.length) {
    const li = document.createElement("li");
    li.innerHTML = `<strong>Sin datos</strong><span class="small-text">No hay registros para esta vista.</span>`;
    target.replaceChildren(li);
    return;
  }
  target.replaceChildren(...rows.map(([title, detail]) => {
    const li = document.createElement("li");
    li.innerHTML = `<strong></strong><span class="small-text"></span>`;
    $("strong", li).textContent = title;
    $("span", li).textContent = detail;
    return li;
  }));
}

function ensureFlowPanel() {
  let panel = $("#flow-panel");
  if (panel) return panel;
  const queuePanel = $(".queue-panel");
  panel = document.createElement("section");
  panel.id = "flow-panel";
  panel.className = "flow-panel";
  panel.setAttribute("aria-labelledby", "flow-title");
  panel.innerHTML = `
    <div class="panel-header">
      <div>
        <p class="eyebrow">Portal empleado VEC</p>
        <h2 id="flow-title">Flujo operativo</h2>
        <span id="flow-mode" class="small-text"></span>
      </div>
      <span id="flow-receipt" class="status-chip chip-slate">Sin recibo</span>
    </div>
    <div id="flow-body" class="flow-body"></div>`;
  queuePanel?.before(panel);
  return panel;
}

function flowRow(labelText, control) {
  const labelNode = document.createElement("label");
  labelNode.append(document.createTextNode(labelText));
  labelNode.append(control);
  return labelNode;
}

function inputControl(name, value, type = "text") {
  const input = document.createElement("input");
  input.name = name;
  input.type = type;
  input.value = value ?? "";
  return input;
}

function selectControl(name, options, selected) {
  const select = document.createElement("select");
  select.name = name;
  options.forEach(([value, text]) => {
    const option = document.createElement("option");
    option.value = value;
    option.textContent = text;
    if (value === selected) option.selected = true;
    select.append(option);
  });
  return select;
}

function flowButton(text, action, kind = "quiet-action") {
  const button = document.createElement("button");
  button.type = "button";
  button.className = kind;
  button.textContent = text;
  button.addEventListener("click", () => runFlowAction(action, button));
  return button;
}

async function runFlowAction(action, button) {
  const previousText = button?.textContent || "";
  if (button) {
    button.disabled = true;
    button.textContent = "Procesando...";
  }
  setStatus("Ejecutando flujo", "loading");
  try {
    await action();
  } catch (error) {
    setStatus(error.message || "No se pudo completar el flujo", "error");
  } finally {
    if (button) {
      button.disabled = false;
      button.textContent = previousText;
    }
  }
}

function renderFlowPanel() {
  const panel = ensureFlowPanel();
  const title = $("#flow-title", panel);
  const mode = $("#flow-mode", panel);
  const receipt = $("#flow-receipt", panel);
  const body = $("#flow-body", panel);
  const module = MODULES.find((item) => item.id === state.activeModule) || MODULES[0];
  const screen = state.portal ? activeScreen(state.portal) : null;
  title.textContent = screen ? `${module.label}: ${screen.title}` : `${module.label}: flujo`;
  receipt.textContent = lastReceiptText();
  receipt.className = `status-chip ${state.actionLog.length ? "chip-green" : "chip-slate"}`;
  body.replaceChildren();
  const transport = moduleTransport(state.activeModule);
  mode.textContent = `${ui(transport.key)} con modulo Bolsa registrado en VEC.`;

  const renderer = flowRenderers[state.activeModule] || renderDashboardFlow;
  renderer(body);
}

function lastReceiptText() {
  const last = state.actionLog[state.actionLog.length - 1];
  return last ? `Recibo ${state.actionLog.length}: ${last.action}` : "Sin recibo";
}

const MODULE_FLOW_CONFIG = {
  personal: {
    title: "Expediente de personal",
    module: "personal",
    endpoint: "personal",
    scope: "Expediente de empleado",
    unit: "Personal",
    prefix: "PERSONAL",
    state: "Borrador RRHH",
    action: "Validar puesto",
    document: "Resolucion pendiente",
    metric: "Alta expediente",
    fields: [
      { name: "employee", label: "Empleado", value: "EMP-0099 - Nueva alta" },
      { name: "scope", label: "Ambito", options: [["Puesto y unidad", "Puesto y unidad"], ["Situacion administrativa", "Situacion administrativa"], ["Servicios prestados", "Servicios prestados"]] },
      { name: "state", label: "Estado", options: [["Borrador RRHH", "Borrador RRHH"], ["Pendiente validar RRHH", "Pendiente validar RRHH"], ["Disponible para certificado", "Disponible para certificado"]] },
    ],
  },
  nominas: {
    title: "Ciclo mensual de nomina",
    module: "nominas",
    endpoint: "nominas",
    scope: "Nomina e incidencias",
    unit: "Nominas",
    prefix: "NOMINA",
    state: "Precalculo generado",
    action: "Revisar nomina",
    document: "Borrador junio",
    metric: "Periodo 2026-06",
    fields: [
      { name: "employee", label: "Empleado", value: "EMP-0064 - Ordenanza" },
      { name: "concept", label: "Concepto", options: [["Trienio", "Trienio"], ["Reduccion 64 anos", "Reduccion 64 anos"], ["Atraso", "Atraso"], ["Dieta liquidada", "Dieta liquidada"]] },
      { name: "state", label: "Estado", options: [["Precalculo generado", "Precalculo generado"], ["Cruce Cronos pendiente", "Cruce Cronos pendiente"], ["Listo para cierre", "Listo para cierre"]] },
    ],
  },
  cronos: {
    title: "Fichaje e incidencia",
    module: "cronos",
    endpoint: "cronos",
    scope: "Fichajes e incidencias",
    unit: "Cronos",
    prefix: "CRONOS",
    state: "Incidencia pendiente",
    action: "Justificar",
    document: "Justificante requerido",
    metric: "07:30 teoricas",
    fields: [
      { name: "employee", label: "Empleado", value: "EMP-0042 - Tecnico/a provincia" },
      { name: "kind", label: "Tipo", options: [["Sin salida", "Sin salida"], ["Defecto horario", "Defecto horario"], ["Teletrabajo", "Teletrabajo"], ["Ausencia", "Ausencia"]] },
      { name: "state", label: "Estado", options: [["Incidencia pendiente", "Incidencia pendiente"], ["Justificada", "Justificada"], ["Pendiente aprobacion", "Pendiente aprobacion"]] },
    ],
  },
  horarios: {
    title: "Perfil horario",
    module: "horarios",
    endpoint: "horarios",
    scope: "Horarios del personal",
    unit: "Cronos",
    prefix: "HORARIO",
    state: "Perfil en revision",
    action: "Editar perfil",
    document: "Tramo obligatorio",
    metric: "37h 30m",
    fields: [
      { name: "employee", label: "Unidad/puesto", value: "Unidad Administracion General" },
      { name: "kind", label: "Perfil", options: [["Flexible administrativo", "Flexible administrativo"], ["Atencion personas mayores", "Atencion personas mayores"], ["Prejubilacion 63", "Prejubilacion 63"], ["Prejubilacion 64", "Prejubilacion 64"]] },
      { name: "state", label: "Estado", options: [["Flexible con tramo obligatorio", "Flexible con tramo obligatorio"], ["Sin flexibilidad", "Sin flexibilidad"], ["1 hora menos diaria", "1 hora menos diaria"], ["2 horas menos diarias", "2 horas menos diarias"]] },
    ],
  },
  permisos: {
    title: "Solicitud de permiso",
    module: "permisos",
    endpoint: "permisos",
    scope: "Permisos y vacaciones",
    unit: "Cronos",
    prefix: "PERMISO",
    state: "Pendiente aprobacion",
    action: "Aprobar",
    document: "Saldo disponible",
    metric: "1 dia",
    fields: [
      { name: "employee", label: "Empleado", value: "EMP-0088 - Secretaria Intervencion" },
      { name: "kind", label: "Tipo", options: [["Asuntos propios", "Asuntos propios"], ["Vacaciones", "Vacaciones"], ["Medico", "Medico"], ["Conciliacion", "Conciliacion"]] },
      { name: "state", label: "Estado", options: [["Pendiente aprobacion", "Pendiente aprobacion"], ["Solape detectado", "Solape detectado"], ["Aprobado", "Aprobado"]] },
    ],
  },
  dietas: {
    title: "Comision de servicio",
    module: "dietas",
    endpoint: "dietas",
    scope: "Comision de servicio",
    unit: "Dietas",
    prefix: "DIETAS",
    state: "Ruta pendiente validar",
    action: "Validar km",
    document: "Media dieta",
    metric: "140,8 km",
    fields: [
      { name: "employee", label: "Empleado", value: "EMP-0031 - Area Obras" },
      { name: "kind", label: "Ruta/dieta", options: [["Granada - Motril", "Granada - Motril"], ["Granada - Baza", "Granada - Baza"], ["Granada - Guadix - Baza", "Granada - Guadix - Baza"]] },
      { name: "state", label: "Estado", options: [["Ruta pendiente validar", "Ruta pendiente validar"], ["Justificante completo", "Justificante completo"], ["Politica excedida", "Politica excedida"]] },
    ],
  },
  rutas: {
    title: "Ruta de kilometraje",
    module: "rutas",
    endpoint: "rutas",
    scope: "Mapa provincial",
    unit: "Dietas",
    prefix: "RUTA",
    state: "Ruta calculada",
    action: "Validar km",
    document: "Tabla kilometraje",
    metric: "70,4 km",
    fields: [
      { name: "employee", label: "Empleado/comision", value: "EMP-0019 - Inspeccion" },
      { name: "kind", label: "Destino", options: [["Motril", "Motril"], ["Loja", "Loja"], ["Guadix", "Guadix"], ["Baza", "Baza"], ["Almunecar", "Almunecar"]] },
      { name: "state", label: "Estado", options: [["Ruta calculada", "Ruta calculada"], ["Parada intermedia", "Parada intermedia"], ["Politica excedida", "Politica excedida"]] },
    ],
  },
};

const flowRenderers = {
  dashboard: renderDashboardFlow,
  personal: (body) => renderConfiguredModuleFlow(body, MODULE_FLOW_CONFIG.personal),
  nominas: (body) => renderConfiguredModuleFlow(body, MODULE_FLOW_CONFIG.nominas),
  cronos: (body) => renderConfiguredModuleFlow(body, MODULE_FLOW_CONFIG.cronos),
  horarios: (body) => renderConfiguredModuleFlow(body, MODULE_FLOW_CONFIG.horarios),
  permisos: (body) => renderConfiguredModuleFlow(body, MODULE_FLOW_CONFIG.permisos),
  dietas: (body) => renderConfiguredModuleFlow(body, MODULE_FLOW_CONFIG.dietas),
  rutas: (body) => renderConfiguredModuleFlow(body, MODULE_FLOW_CONFIG.rutas),
  bolsa: renderWorkspaceModuleFlow,
  aprobaciones: renderWorkspaceModuleFlow,
  convocatorias: renderConvocatoriasFlow,
  solicitudes: renderSolicitudesFlow,
  meritos: renderMeritosFlow,
  autobaremo: renderAutobaremoFlow,
  documentos: renderDocumentosFlow,
  alegaciones: renderAlegacionesFlow,
  notificaciones: renderNotificacionesFlow,
  listados: renderListadosFlow,
  manifiestos: renderManifiestosFlow,
  auditoria: renderAuditoriaFlow,
  administracion: renderAdministracionFlow,
};

function appendFlowSummary(body, title, rows) {
  const summary = document.createElement("div");
  summary.className = "flow-summary";
  const heading = document.createElement("h3");
  heading.textContent = title;
  const list = document.createElement("ul");
  rows.forEach(([labelText, value]) => {
    const li = document.createElement("li");
    li.innerHTML = `<span></span><strong></strong>`;
    $("span", li).textContent = labelText;
    $("strong", li).textContent = value;
    list.append(li);
  });
  summary.append(heading, list);
  body.append(summary);
}

function recordReceipt(action, detail, module = state.activeModule) {
  const entry = {
    action,
    detail,
    module,
    at: new Date().toLocaleString("es-ES"),
    actor: module === "solicitudes" || module === "meritos" || module === "autobaremo" ? state.candidate.id : "portal-vec",
  };
  state.actionLog.push(entry);
  setStatus(`${action}: ${detail}`, "ready");
  return entry;
}

function upsertOperationalRow(row) {
  const index = state.localRows.findIndex((item) => item.id === row.id);
  if (index >= 0) {
    state.localRows.splice(index, 1, row);
  } else {
    state.localRows.unshift(row);
  }
  state.selectedRowID = row.id;
}

function fieldControl(field) {
  if (field.options) return selectControl(field.name, field.options, field.value || field.options[0]?.[0]);
  return inputControl(field.name, field.value || "", field.type || "text");
}

function modulesForFlow(config) {
  const modules = ["dashboard", config.module, "aprobaciones", "auditoria"];
  if (config.endpoint === "personal" || config.module === "nominas") modules.push("personal");
  if (config.endpoint === "cronos" || ["horarios", "permisos"].includes(config.module)) modules.push("cronos");
  if (config.endpoint === "dietas" || config.module === "rutas") modules.push("dietas");
  if (["personal", "nominas", "bolsa"].includes(config.module)) modules.push("documentos");
  return [...new Set(modules)];
}

function buildFlowRow(config, data, receipt, actionLabel) {
  const stamp = Date.now().toString().slice(-6);
  const kind = data.kind || data.scope || data.concept || config.scope;
  const stateText = data.state || config.state;
  const title = `${kind} - ${actionLabel || config.action}`;
  return {
    id: `${config.prefix}-${stamp}`,
    expediente: `${config.prefix}-2026-${stamp}`,
    candidate: data.employee || "Personal interno",
    state: stateText,
    stateFilter: /pendiente|excedida|solape|revision/i.test(stateText) ? "Pendiente de accion" : "En revision",
    deadline: /vencid/i.test(stateText) ? "Plazo vencido" : "Seguimiento ordinario",
    deadlineBucket: /vencid/i.test(stateText) ? "Plazo vencido" : "Sin vencimiento critico",
    points: data.metric || config.metric,
    document: data.document || config.document,
    action: config.action,
    scope: data.scope || config.scope,
    unit: config.unit,
    modules: modulesForFlow(config),
    documents: [
      [config.title, title],
      ["Recibo", receipt?.id || "Flujo local pendiente de integracion productiva"],
      ["Soporte", data.document || config.document],
    ],
    merits: [
      ["Dato maestro", data.employee || "Personal interno"],
      ["Politica aplicada", policyHint({ scope: config.scope, title, state: stateText })],
    ],
    alerts: [
      [stateText, "Registro creado desde el flujo del modulo"],
      ["Integracion", receipt?.id ? "Accion registrada en auditoria VEC" : "Pendiente de adaptador productivo"],
    ],
    timeline: [
      [actionLabel || config.action, `${new Date().toLocaleString("es-ES")} - ${receipt?.id || "sin recibo backend"}`],
      ["Modulo", `${config.unit} - ${config.scope}`],
    ],
  };
}

function moduleEndpointFor(moduleID = state.activeModule, row = selectedRow()) {
  if (MODULE_ACTION_ENDPOINT[moduleID]) return MODULE_ACTION_ENDPOINT[moduleID];
  if (row?.modules?.includes("nominas")) return "nominas";
  if (row?.modules?.includes("personal")) return "personal";
  if (row?.modules?.includes("cronos")) return "cronos";
  if (row?.modules?.includes("dietas")) return "dietas";
  if (row?.modules?.includes("bolsa")) return "bolsa";
  return "";
}

async function requestModuleReceipt(moduleKey, actionLabel) {
  if (!moduleKey) return null;
  try {
    const payload = await sendJSON(`${VEC_SHELL_API}/modules/${moduleKey}/action`, "POST", null, staffHeaders());
    return payload.receipt || payload;
  } catch (error) {
    recordReceipt(`${actionLabel} sin endpoint`, error.message || `No existe accion ${moduleKey}`, moduleKey);
    return null;
  }
}

async function createModuleFlowRecord(config, data, actionLabel) {
  const receipt = await requestModuleReceipt(config.endpoint || moduleEndpointFor(config.module), actionLabel || config.action);
  recordReceipt(actionLabel || config.action, `${config.title} - ${receipt?.id || "flujo local"}`, config.module);
  upsertOperationalRow(buildFlowRow(config, data, receipt, actionLabel));
  renderPortal(state.portal);
}

function renderConfiguredModuleFlow(body, config) {
  const intro = document.createElement("p");
  intro.className = "small-text";
  intro.textContent = `Alta rapida: ${config.title.toLowerCase()}. Se registra con recibo auditable.`;
  body.append(intro);
  const form = document.createElement("form");
  form.className = "flow-form";
  config.fields.forEach((field) => {
    form.append(flowRow(field.label, fieldControl(field)));
  });
  form.append(flowButton(`Registrar ${config.title.toLowerCase()}`, async () => {
    await createModuleFlowRecord(config, Object.fromEntries(new FormData(form).entries()), config.action);
  }, "primary-action"));
  body.append(form);
}

function latestReceipt(action) {
  for (let index = state.actionLog.length - 1; index >= 0; index -= 1) {
    const entry = state.actionLog[index];
    if (entry.action === action) return `${entry.at} - ${entry.actor}`;
  }
  return "Pendiente de recibo";
}

function candidateRowFromState(stateText, action) {
  const baremoPoints = state.baremo ? `${formatPoints(state.baremo.total_points)} pt` : "Pendiente";
  const meritRows = state.baremo?.details?.length
    ? state.baremo.details.map((item) => [
        item.merit_id,
        `${label(item.section)}: ${formatPoints(item.applied_points)} pt${item.capped ? " (tope aplicado)" : ""}`,
      ])
    : state.merits.map((merit) => [merit.id, `${label(merit.tipo)} - ${merit.estado}`]);
  return {
    id: `candidate-${state.candidate.id}`,
    expediente: `EXP-${state.candidate.id}`,
    candidate: `${state.candidate.nombre} - ${state.candidate.dni}`,
    state: stateText,
    stateFilter: stateText.includes("calculad") ? "En revision" : "Pendiente de accion",
    deadline: "Seguimiento ordinario",
    deadlineBucket: "Sin vencimiento critico",
    points: baremoPoints,
    document: state.expediente ? "Expediente exportado" : "Expediente en preparacion",
    action,
    scope: "Portal empleado VEC",
    unit: "Modulo Bolsa",
    modules: ["dashboard", "solicitudes", "meritos", "autobaremo", "documentos", "auditoria"],
    documents: [
      ["Solicitud", `${state.candidate.call_id} - ${latestReceipt("Candidato registrado")}`],
      ["Expediente", state.expediente ? `${state.expediente.merits.length} meritos exportados` : "Pendiente de exportar"],
      ["Identidad", `${state.candidate.dni} - autenticacion Cl@ve simulada por adaptador`],
    ],
    merits: meritRows.length ? meritRows : [["Sin meritos", "Registra titulos, cursos o experiencia desde el flujo Meritos"]],
    alerts: [
      ["Estado operativo", "Datos editados en portal; la validez juridica depende del backend y registro"],
      ["Integracion VEC", "Fila reutilizable en expediente, RUM y notificaciones"],
    ],
    timeline: [
      ["Solicitud actualizada", latestReceipt("Candidato registrado")],
      ["Meritos", `${state.merits.length} registrados en sesion`],
      ["Autobaremo", state.baremo ? `${formatPoints(state.baremo.total_points)} pt calculados por API` : "Pendiente"],
    ],
  };
}

function documentRowFromState(doc) {
  return {
    id: `document-${doc.id}`,
    expediente: `DOC-${doc.id}`,
    candidate: state.candidate.id,
    state: doc.state,
    stateFilter: doc.state === "Pendiente" ? "Pendiente de accion" : "En revision",
    deadline: "Validacion documental",
    deadlineBucket: "Sin vencimiento critico",
    points: "Evidencia",
    document: doc.csv,
    action: "Validar",
    scope: "Expediente candidato",
    unit: "Registro y documentos",
    modules: ["dashboard", "documentos", "auditoria"],
    documents: [[doc.title, `${doc.csv} - ${doc.state}`], ["Version", "Registro local trazable pendiente de adaptador documental VEC"]],
    merits: [["Reutilizacion", "Documento disponible para solicitud, merito o alegacion"]],
    alerts: [["Validez", "El portal no declara validez juridica sin servicio documental"]],
    timeline: [["Documento registrado", latestReceipt("Documento registrado")]],
  };
}

function claimRowFromState(claim) {
  return {
    id: `claim-${claim.id}`,
    expediente: claim.id,
    candidate: state.candidate.id,
    state: claim.state,
    stateFilter: claim.state === "Abierta" ? "Pendiente de accion" : "En revision",
    deadline: "Audiencia/subsanacion",
    deadlineBucket: "Sin vencimiento critico",
    points: "Objeto de revision",
    document: claim.subject,
    action: "Resolver",
    scope: "Modulo Bolsa",
    unit: "Personal temporal",
    modules: ["dashboard", "alegaciones", "documentos", "auditoria"],
    documents: [["Escrito", claim.detail], ["Relacion", `Expediente ${state.candidate.id}`]],
    merits: [["Objeto", claim.subject], ["Estado", claim.state]],
    alerts: [["Revision", "Requiere tramitacion administrativa antes de resolver"]],
    timeline: [["Alegacion registrada", latestReceipt("Alegacion registrada")]],
  };
}

function notificationRowFromState(notification) {
  return {
    id: `notification-${notification.id}`,
    expediente: notification.id,
    candidate: state.candidate.id,
    state: notification.state,
    stateFilter: notification.state === "Pendiente" ? "Pendiente de accion" : "En revision",
    deadline: notification.deadline || "Sin vencimiento critico",
    deadlineBucket: notification.deadline?.includes("72") ? "Vence en 72 h" : "Sin vencimiento critico",
    points: "Comunicacion",
    document: notification.title,
    action: "Comprobar notificaciones",
    scope: "Portal empleado VEC",
    unit: "Buzon profesional",
    modules: ["dashboard", "notificaciones", "auditoria"],
    documents: [["Aviso", notification.title], ["Canal", "Buzon VEC"]],
    merits: [["Relacionado", `Expediente ${state.candidate.id}`]],
    alerts: [[notification.state, notification.deadline || "Sin plazo critico declarado"]],
    timeline: [["Notificacion emitida", latestReceipt("Notificacion emitida")]],
  };
}

async function ensureCandidateExists() {
  const result = await sendJSON("/api/candidates", "POST", state.candidate, candidateHeaders(state.candidate.id));
  state.candidate = { ...state.candidate, ...result };
  if (!state.localRows.some((row) => row.id === `candidate-${state.candidate.id}`)) {
    recordReceipt("Candidato registrado", `${state.candidate.id} en ${state.candidate.call_id}`, "solicitudes");
    upsertOperationalRow(candidateRowFromState("Solicitud registrada", "Abrir"));
  }
  return state.candidate;
}

function exportAudit() {
  const header = ["fecha", "actor", "modulo", "accion", "detalle"];
  const csv = [
    header.join(";"),
    ...state.actionLog.map((entry) => [entry.at, entry.actor || "-", entry.module || "-", entry.action, entry.detail]
      .map((value) => `"${String(value || "").replaceAll('"', '""')}"`).join(";")),
  ].join("\n");
  const blob = new Blob([csv], { type: "text/csv;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = `vec-auditoria-${new Date().toISOString().slice(0, 10)}.csv`;
  document.body.append(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
  recordReceipt("Exportacion auditoria", `${state.actionLog.length} recibos exportados`, "auditoria");
  renderPortal(state.portal);
}

function renderDashboardFlow(body) {
  appendFlowSummary(body, ui("dashboardTitle"), [
    ["Siguiente modulo", selectedRow()?.unit || "Cronos / Dietas / Bolsa"],
    ["Recibos", `${state.actionLog.length} acciones trazadas`],
    ["Integracion VEC", `${state.portal?.vecModules?.length || 3} modulos independientes registrados`],
  ]);
  body.append(flowButton(ui("dashboardAction"), () => {
    const row = selectedRow();
    const next = ["personal", "nominas", "cronos", "horarios", "permisos", "dietas", "rutas", "bolsa"].find((module) => row?.modules.includes(module)) || "personal";
    setActiveModule(next);
  }, "primary-action"));
}

function renderWorkspaceModuleFlow(body) {
  const row = selectedRow();
  appendFlowSummary(body, MODULE_COPY[state.activeModule]?.[0] || "Modulo VEC", [
    ["Registro", row?.expediente || "-"],
    ["Estado", row?.state || "-"],
    ["Politica", row ? policyHint({ scope: row.scope, title: row.expediente, state: row.state }) : "-"],
  ]);
  body.append(flowButton(row?.action || "Registrar accion", () => {
    if (row) handleRowAction(row);
  }, "primary-action"));
}

function renderConvocatoriasFlow(body) {
  appendFlowSummary(body, "Convocatoria vigente", [
    ["Convocatoria", state.portal?.call?.id || "-"],
    ["Estado", state.portal?.call?.estado || "-"],
    ["Version provisional", state.portal?.provisional?.version || "-"],
    ["Version definitiva", state.portal?.definitive?.version || "-"],
  ]);
  body.append(flowButton(ui("convocatoriaAction"), async () => {
    state.demo = await loadDemoData();
    if (state.portal) {
      state.portal = normalizePortal({ ...state.portal }, state.demo);
      recordReceipt("Convocatoria/listados recargados", "Publicacion demo refrescada desde API /api/demo", "convocatorias");
      renderPortal(state.portal);
    }
  }, "primary-action"));
}

function renderSolicitudesFlow(body) {
  const form = document.createElement("form");
  form.className = "flow-form";
  form.append(
    flowRow("ID candidato", inputControl("id", state.candidate.id)),
    flowRow("DNI", inputControl("dni", state.candidate.dni)),
    flowRow("Nombre", inputControl("nombre", state.candidate.nombre)),
    flowRow("Email", inputControl("email", state.candidate.email, "email")),
    flowRow("Convocatoria", inputControl("call_id", state.candidate.call_id)),
  );
  const submit = flowButton(ui("solicitudAction"), async () => {
    const data = Object.fromEntries(new FormData(form).entries());
    const result = await sendJSON("/api/candidates", "POST", data, candidateHeaders(data.id));
    state.candidate = { ...state.candidate, ...result };
    recordReceipt("Candidato registrado", `${result.id} en ${result.call_id}`, "solicitudes");
    upsertOperationalRow(candidateRowFromState("Solicitud registrada", "Abrir"));
    renderPortal(state.portal);
  }, "primary-action");
  form.append(submit);
  body.append(form);
}

function renderMeritosFlow(body) {
  const form = document.createElement("form");
  form.className = "flow-form";
  form.append(
    flowRow("Candidato", inputControl("candidate_id", state.candidate.id)),
    flowRow("ID merito", inputControl("id", `merit-${Date.now().toString().slice(-6)}`)),
    flowRow("Tipo", selectControl("tipo", [
      ["formacion_titulo", "Titulo oficial"],
      ["formacion_curso", "Curso formacion"],
      ["experiencia_misma_categoria", "Experiencia misma categoria"],
      ["experiencia_otra_categoria", "Experiencia otra categoria"],
      ["otros", "Otros meritos"],
    ], "formacion_titulo")),
    flowRow("Meses", inputControl("meses", "0", "number")),
    flowRow("Horas", inputControl("horas", "0", "number")),
    flowRow("Puntos fijos", inputControl("puntos_fijos", "4", "number")),
    flowRow("Estado", selectControl("estado", [
      ["Presentado", "Presentado"],
      ["Borrador", "Borrador"],
      ["Validado", "Validado"],
      ["Subsanacion", "Subsanacion"],
      ["Rechazado", "Rechazado"],
    ], "Presentado")),
  );
  form.append(flowButton(ui("meritAction"), async () => {
    await ensureCandidateExists();
    const data = Object.fromEntries(new FormData(form).entries());
    const candidateID = String(data.candidate_id || state.candidate.id).trim();
    const payload = {
      id: data.id,
      tipo: data.tipo,
      estado: data.estado,
      datos: {
        meses: Number(data.meses || 0),
        horas: Number(data.horas || 0),
        puntos_fijos: Number(data.puntos_fijos || 0),
      },
    };
    const merit = await sendJSON(`/api/candidates/${encodeURIComponent(candidateID)}/merits`, "POST", payload, candidateHeaders(candidateID));
    state.merits.push(merit);
    recordReceipt("Merito/titulo guardado", `${merit.id} (${merit.tipo}) para ${candidateID}`, "meritos");
    upsertOperationalRow(candidateRowFromState("Merito presentado", "Revisar autobaremo"));
    renderPortal(state.portal);
  }, "primary-action"));
  appendFlowSummary(body, "Meritos cargados en sesion", state.merits.length
    ? state.merits.map((merit) => [merit.id, `${label(merit.tipo)} - ${merit.estado}`])
    : [["Sin meritos", "Guarda un titulo, curso o experiencia para calcular"]]);
  body.append(form);
}

function renderAutobaremoFlow(body) {
  appendFlowSummary(body, "Autobaremacion actual", [
    ["Candidato", state.candidate.id],
    ["Total", state.baremo ? `${formatPoints(state.baremo.total_points)} pt` : "Pendiente"],
    ["Reglas", state.baremo?.rule_set_version || "v1"],
    ["Detalles", `${state.baremo?.details?.length || 0}`],
  ]);
  const actions = document.createElement("div");
  actions.className = "flow-actions";
  actions.append(
    flowButton(ui("baremoAction"), async () => {
      await ensureCandidateExists();
      state.baremo = await sendJSON(`/api/candidates/${encodeURIComponent(state.candidate.id)}/baremo`, "POST", null, candidateHeaders(state.candidate.id));
      recordReceipt("Autobaremacion calculada", `${formatPoints(state.baremo.total_points)} pt para ${state.candidate.id}`, "autobaremo");
      upsertOperationalRow(candidateRowFromState("Autobaremo calculado", "Exportar expediente"));
      renderPortal(state.portal);
    }, "primary-action"),
    flowButton(ui("expedienteAction"), async () => {
      await ensureCandidateExists();
      state.expediente = await getData(`/api/candidates/${encodeURIComponent(state.candidate.id)}/expediente`, { headers: candidateHeaders(state.candidate.id) });
      recordReceipt("Expediente exportado", `${state.expediente.merits.length} meritos y ${formatPoints(state.expediente.baremo.total_points)} pt`, "autobaremo");
      renderPortal(state.portal);
    }),
  );
  body.append(actions);
  if (state.baremo?.details?.length) {
    appendFlowSummary(body, "Desglose", state.baremo.details.map((item) => [
      item.merit_id,
      `${item.section}: ${formatPoints(item.applied_points)} pt${item.capped ? " (tope aplicado)" : ""}`,
    ]));
  }
}

function solicitudID() {
  return `SOL-${state.candidate.id}`;
}

function upsertByID(items, next) {
  const filtered = items.filter((item) => item.id !== next.id);
  return [next, ...filtered];
}

function renderDocumentosFlow(body) {
  const form = document.createElement("form");
  form.className = "flow-form";
  form.append(
    flowRow("Documento", inputControl("title", "Titulo formacion")),
    flowRow("Recibo local", inputControl("csv", `CSV-GR-2026-${Math.floor(1000 + Math.random() * 8000)}`)),
    flowRow("Estado", selectControl("state", [["Pendiente", "Pendiente"], ["Metadatos presentes", "Metadatos presentes"], ["Recibo pendiente", "Recibo pendiente"]], "Pendiente")),
    flowButton(ui("documentAction"), async () => {
      const data = Object.fromEntries(new FormData(form).entries());
      const doc = isEndpointAvailable("/api/candidates/{id}/documents")
        ? await registerDocument(data)
        : { id: `doc-${Date.now().toString().slice(-6)}`, ...data, source: ui("localFlow") };
      state.documents = upsertByID(state.documents, doc);
      recordReceipt("Evidencia guardada", `${doc.title} - ${doc.csv} (${doc.source})`, "documentos");
      upsertOperationalRow(documentRowFromState(doc));
      renderPortal(state.portal);
    }, "primary-action"),
  );
  appendFlowSummary(body, "Evidencias del portal", state.documents.map((doc) => [doc.title, `${doc.csv} - ${doc.state} - ${doc.source || "local"}`]));
  body.append(form);
  if (isEndpointAvailable("/api/candidates/{id}/documents")) {
    body.append(flowButton(ui("documentSyncAction"), syncDocuments, "quiet-action"));
  }
}

async function registerDocument(data) {
  await ensureCandidateExists();
  const suffix = Date.now().toString().slice(-6);
  const id = `doc-${suffix}`;
  const payload = {
    id,
    solicitud_id: solicitudID(),
    procedure_id: state.candidate.call_id || "convocatoria-default",
    purpose: "Solicitud",
    csv: data.csv,
    digest_sha256: "0".repeat(64),
    storage_object_ref: `mem://${state.candidate.id}/${id}`,
    document_id: id,
    document_type: "evidencia",
    title: data.title,
    format: "application/pdf",
    language: "es",
    signature_ref: `sig-${suffix}`,
  };
  const created = await sendJSON(endpointPath("/api/candidates/{id}/documents"), "POST", payload, candidateHeaders(state.candidate.id));
  return documentFromAPI(created, data.state, data.title);
}

async function syncDocuments() {
  await ensureCandidateExists();
  const documents = await getData(endpointPath("/api/candidates/{id}/documents"), { method: "GET", headers: candidateHeaders(state.candidate.id) });
  state.documents = (documents || []).map((doc) => documentFromAPI(doc));
  recordReceipt("Documentos sincronizados", `${state.documents.length} documentos desde API`, "documentos");
  state.documents.forEach((doc) => upsertOperationalRow(documentRowFromState(doc)));
  renderPortal(state.portal);
}

function documentFromAPI(doc, fallbackState = "Verificacion externa pendiente", fallbackTitle = "Documento") {
  const status = doc.av_status === "CLEAN"
    ? "Metadatos presentes"
    : doc.av_status === "PENDING"
      ? "Verificacion externa pendiente"
      : fallbackState;
  return {
    id: doc.id,
    title: fallbackTitle || doc.purpose || doc.receipt_i18n_key || "Documento",
    csv: doc.csv,
    state: status,
    source: ui("apiReal"),
    audit_sequence: doc.audit_sequence,
  };
}

function renderAlegacionesFlow(body) {
  const form = document.createElement("form");
  form.className = "flow-form";
  form.append(
    flowRow("Asunto", inputControl("subject", "Revision de puntuacion")),
    flowRow("Detalle", inputControl("detail", "Aporta nuevo titulo o corrige baremo")),
    flowRow("Estado", selectControl("state", [["Abierta", "Abierta"], ["En informe", "En informe"], ["Resuelta", "Resuelta"]], "Abierta")),
    flowButton(ui("claimAction"), async () => {
      const data = Object.fromEntries(new FormData(form).entries());
      const claim = isEndpointAvailable("/api/candidates/{id}/claims")
        ? await registerClaim(data)
        : { id: `aleg-${Date.now().toString().slice(-6)}`, ...data, source: ui("localFlow") };
      state.claims = upsertByID(state.claims, claim);
      recordReceipt("Alegacion guardada", `${claim.id} - ${claim.subject} (${claim.source})`, "alegaciones");
      upsertOperationalRow(claimRowFromState(claim));
      renderPortal(state.portal);
    }, "primary-action"),
  );
  appendFlowSummary(body, "Alegaciones", state.claims.map((claim) => [claim.subject, `${claim.state} - ${claim.detail} - ${claim.source || "local"}`]));
  body.append(form);
  if (isEndpointAvailable("/api/candidates/{id}/claims")) {
    body.append(flowButton(ui("claimSyncAction"), syncClaims, "quiet-action"));
  }
}

async function registerClaim(data) {
  await ensureCandidateExists();
  const suffix = Date.now().toString().slice(-6);
  const payload = {
    id: `aleg-${suffix}`,
    solicitud_id: solicitudID(),
    text: `${data.subject}: ${data.detail}`,
    receipt_csv: `CSV-ALEG-${suffix}`,
  };
  const created = await sendJSON(endpointPath("/api/candidates/{id}/claims"), "POST", payload, candidateHeaders(state.candidate.id));
  return claimFromAPI(created, data.subject, data.detail);
}

async function syncClaims() {
  await ensureCandidateExists();
  const claims = await getData(`${endpointPath("/api/candidates/{id}/claims")}?solicitud_id=${encodeURIComponent(solicitudID())}`, {
    method: "GET",
    headers: candidateHeaders(state.candidate.id),
  });
  state.claims = (claims || []).map((claim) => claimFromAPI(claim));
  recordReceipt("Alegaciones sincronizadas", `${state.claims.length} alegaciones desde API`, "alegaciones");
  state.claims.forEach((claim) => upsertOperationalRow(claimRowFromState(claim)));
  renderPortal(state.portal);
}

function claimFromAPI(claim, fallbackSubject = "", fallbackDetail = "") {
  const text = claim.text || "";
  const [subject, ...detailParts] = text.split(": ");
  return {
    id: claim.id,
    subject: fallbackSubject || subject || claim.receipt_i18n_key || "Alegacion",
    detail: fallbackDetail || detailParts.join(": ") || text || claim.receipt_csv || "Sin detalle",
    state: claim.state || "Presentada",
    source: ui("apiReal"),
    receipt_csv: claim.receipt_csv,
    audit_sequence: claim.audit_sequence,
  };
}

function renderNotificacionesFlow(body) {
  const form = document.createElement("form");
  form.className = "flow-form";
  form.append(
    flowRow("Candidato", inputControl("candidate_id", state.candidate.id)),
    flowRow("Titulo", inputControl("title", "Aviso de subsanacion")),
    flowRow("Plazo", inputControl("deadline", "72 h")),
    flowRow("Estado", selectControl("state", [["Pendiente", "Pendiente"], ["Leida", "Leida"], ["Aceptada", "Aceptada"]], "Pendiente")),
    flowButton(ui("notificationAction"), async () => {
      const data = Object.fromEntries(new FormData(form).entries());
      const notification = notificationCreateAvailable()
        ? await registerNotification(data)
        : { id: `notif-${Date.now().toString().slice(-6)}`, ...data, source: ui("localFlow") };
      state.notifications = upsertByID(state.notifications, notification);
      recordReceipt("Aviso guardado", `${notification.title} - ${notification.deadline} (${notification.source})`, "notificaciones");
      upsertOperationalRow(notificationRowFromState(notification));
      renderPortal(state.portal);
    }, "primary-action"),
  );
  appendFlowSummary(body, "Buzon VEC", state.notifications.map((item) => [item.title, `${item.state} - ${item.deadline} - ${item.source || "local"}`]));
  body.append(form);
  appendNotificationActions(body);
  if (notificationListAvailable()) {
    body.append(flowButton(ui("notificationSyncAction"), syncNotifications, "quiet-action"));
  }
}

async function registerNotification(data) {
  const candidateID = String(data.candidate_id || state.candidate.id).trim();
  if (candidateID) state.candidate.id = candidateID;
  const suffix = Date.now().toString().slice(-6);
  const payload = {
    id: `notif-${suffix}`,
    candidate_id: candidateID,
    solicitud_id: solicitudID(),
    type: "aviso",
    subject: data.title,
    body: `Plazo: ${data.deadline}. Estado inicial UI: ${data.state}`,
  };
  const route = isEndpointAvailable("/api/notifications") ? "/api/notifications" : endpointPath("/api/candidates/{id}/notifications", candidateID);
  const created = await sendJSON(route, "POST", payload, staffHeaders());
  return notificationFromAPI(created, data.deadline);
}

async function syncNotifications() {
  const route = isEndpointAvailable("/api/notifications?candidate_id={id}")
    ? `/api/notifications?candidate_id=${encodeURIComponent(state.candidate.id)}`
    : endpointPath("/api/candidates/{id}/notifications");
  const notifications = await getData(route, { method: "GET", headers: staffHeaders() });
  state.notifications = (notifications || []).map((notification) => notificationFromAPI(notification));
  recordReceipt("Avisos sincronizados", `${state.notifications.length} avisos desde API`, "notificaciones");
  state.notifications.forEach((notification) => upsertOperationalRow(notificationRowFromState(notification)));
  renderPortal(state.portal);
}

function notificationCreateAvailable() {
  return isEndpointAvailable("/api/notifications") || isEndpointAvailable("/api/candidates/{id}/notifications");
}

function notificationListAvailable() {
  return isEndpointAvailable("/api/notifications?candidate_id={id}") || isEndpointAvailable("/api/candidates/{id}/notifications");
}

function notificationMutationAvailable(action) {
  return isEndpointAvailable(`/api/notifications/{id}/${action}`);
}

function appendNotificationActions(body) {
  const actions = document.createElement("div");
  actions.className = "notification-actions";
  const apiNotifications = state.notifications.filter((item) => item.source === ui("apiReal"));
  if (!apiNotifications.length) {
    const empty = document.createElement("p");
    empty.className = "small-text";
    empty.textContent = "Cambiar estado requiere un aviso creado por backend.";
    actions.append(empty);
  }
  apiNotifications.forEach((notification) => {
    const row = document.createElement("div");
    row.className = "notification-action-row";
    row.innerHTML = `<span></span><div class="flow-actions"></div>`;
    $("span", row).textContent = `${notification.id} - ${notification.title} - ${notification.state}`;
    const controls = $(".flow-actions", row);
    if (notificationMutationAvailable("send") && notification.state === "Creada") {
      controls.append(flowButton(ui("notificationSendAction"), () => mutateNotification(notification, "send"), "quiet-action"));
    }
    if (notificationMutationAvailable("read") && notification.state === "Enviada") {
      controls.append(flowButton(ui("notificationReadAction"), () => mutateNotification(notification, "read"), "quiet-action"));
    }
    if (!controls.children.length) {
      const status = document.createElement("strong");
      status.className = "small-text";
      status.textContent = "Sin accion backend disponible";
      controls.append(status);
    }
    actions.append(row);
  });
  body.append(actions);
}

async function mutateNotification(notification, action) {
  const payload = {
    csv: `CSV-NOT-${Date.now().toString().slice(-6)}`,
    recipient_id: notification.candidate_id || state.candidate.id,
    channel: "vec",
    issued_at: new Date().toISOString(),
  };
  const updated = await sendJSON(notificationActionPath(notification.id, action), "POST", payload, staffHeaders());
  const normalized = notificationFromAPI(updated, notification.deadline);
  state.notifications = upsertByID(state.notifications, normalized);
  recordReceipt(
    action === "send" ? "Aviso marcado como enviado" : "Aviso marcado como leido",
    `${normalized.id} - ${normalized.state} - secuencia ${normalized.audit_sequence || "pendiente"}`,
    "notificaciones",
  );
  upsertOperationalRow(notificationRowFromState(normalized));
  renderPortal(state.portal);
}

function notificationFromAPI(notification, fallbackDeadline = "Sin vencimiento critico") {
  return {
    id: notification.id,
    candidate_id: notification.candidate_id,
    title: notification.subject || notification.type || "Aviso",
    deadline: fallbackDeadline,
    state: notification.state || "Creada",
    source: ui("apiReal"),
    audit_sequence: notification.audit_sequence,
  };
}

function renderListadosFlow(body) {
  appendFlowSummary(body, "Listados publicables", [
    ["Provisional", `${state.portal?.provisional?.items?.length || 0} solicitudes`],
    ["Definitivo", `${state.portal?.definitive?.items?.length || 0} solicitudes`],
    ["Exportacion", "CSV operativo desde la tabla actual"],
  ]);
  body.append(flowButton(ui("listingAction"), exportRows, "primary-action"));
}

function renderAuditoriaFlow(body) {
  const apiAudit = (state.auditEntries || []).map((entry) => [
    `#${entry.sequence} ${entry.action}`,
    `${entry.actor} - ${new Date(entry.occurred_at).toLocaleString("es-ES")} - ${entry.signature}`,
  ]);
  const localAudit = state.actionLog.map((entry, index) => [`local #${index + 1} ${entry.action}`, `${entry.detail} - modulo ${entry.module}`]);
  appendFlowSummary(body, ui("receiptTitle"), apiAudit.length || localAudit.length
    ? [...apiAudit, ...localAudit]
    : [[ui("noReceipts"), ui("auditEmptyHint")]]);
  const form = document.createElement("form");
  form.className = "flow-form audit-candidate-form";
  form.append(flowRow(ui("auditCandidateField"), inputControl("candidate_id", state.candidate.id)));
  const actions = document.createElement("div");
  actions.className = "flow-actions";
  actions.append(
    flowButton(ui("auditAction"), () => syncAudit(Object.fromEntries(new FormData(form).entries()).candidate_id), "primary-action"),
    flowButton(ui("auditExportAction"), exportAudit, "quiet-action"),
  );
  form.append(actions);
  body.append(form);
}

async function syncAudit(candidateID = state.candidate.id) {
  const normalizedCandidateID = String(candidateID || state.candidate.id).trim();
  const entries = isEndpointAvailable("/api/audit?candidate_id={id}")
    ? await getData(`/api/audit?candidate_id=${encodeURIComponent(normalizedCandidateID)}`, { method: "GET", headers: staffHeaders() })
    : await getData(endpointPath("/api/candidates/{id}/audit"), { method: "GET", headers: staffHeaders() });
  state.auditEntries = entries || [];
  recordReceipt(ui("auditReceipt"), `${state.auditEntries.length} entradas desde API para ${normalizedCandidateID}`, "auditoria");
  renderPortal(state.portal);
}

function renderManifiestosFlow(body) {
  appendFlowSummary(body, ui("manifestTitle"), [
    [ui("manifestModule"), state.moduleManifest?.module_ref || "vec.module.bolsa"],
    [ui("manifestVersion"), state.moduleManifest?.version || "-"],
    [ui("manifestPrototypeAPI"), state.moduleManifest?.prototype_api_prefix || "/api"],
    [ui("manifestHTTPRoutes"), `${(state.moduleManifest?.http_routes || []).length}`],
  ]);
  const actions = document.createElement("div");
  actions.className = "flow-actions";
  actions.append(
    flowButton(ui("manifestAction"), async () => {
      state.moduleManifest = await loadModuleManifest();
      recordReceipt(ui("manifestLoadedReceipt"), `${state.moduleManifest.module_ref} ${state.moduleManifest.version}`, "manifiestos");
      renderPortal(state.portal);
    }, "primary-action"),
    flowButton(ui("manifestHealthAction"), async () => {
      await getData("/api/modules/bolsa/healthz", { method: "GET", headers: staffHeaders() });
      recordReceipt(ui("manifestHealthReceipt"), "GET /api/modules/bolsa/healthz ok", "manifiestos");
      renderPortal(state.portal);
    }),
  );
  body.append(actions);
  if (state.moduleManifest?.capabilities?.length) {
    appendFlowSummary(body, ui("manifestCapabilities"), state.moduleManifest.capabilities.map((capability) => [
      capability.capability_ref,
      `${capability.mode} ${capability.method || ""} ${capability.route || ""}`,
    ]));
  }
}

function adminAdapterRows() {
  const status = state.adminStatus || {};
  const capabilities = state.adminCapabilities || {};
  const integrations = status.legal_integrations || capabilities.legal_integrations || [];
  const externalSummary = integrations.length
    ? integrations.map((item) => `${item.integration_ref}: ${item.status} (${item.mode})`).join("; ")
    : ui("adminNoExternalAdapters");
  return [
    [ui("adminStorageAdapter"), status.persistence_mode || "pendiente API"],
    [ui("adminAuthAdapter"), status.auth_mode || "pendiente API"],
    [ui("adminExternalAdapters"), externalSummary],
  ];
}

function adminRouteRows() {
  const routes = state.adminCapabilities?.http_routes || state.adminStatus?.admin_routes || [];
  return routes.length
    ? routes.map((route) => [route.route, `${route.method} - ${route.mode}`])
    : [[ADMIN_STATUS_API, "GET - pendiente API"], [ADMIN_CAPABILITIES_API, "GET - pendiente API"]];
}

function adminStatusRows() {
  const status = state.adminStatus || {};
  return [
    ["Modulo", status.module_ref || "vec.module.bolsa"],
    ["Runtime", status.runtime_mode || "vec_module_local_productizable"],
    ["Estado", status.status || "pendiente API"],
    ["Demo", status.demo_enabled === true ? "habilitada" : status.demo_enabled === false ? "deshabilitada" : "pendiente API"],
    ["Produccion legal", status.legal_production_ready ? "lista" : "no declarada"],
  ];
}

function renderAdministracionFlow(body) {
  appendFlowSummary(body, ui("adminAdaptersTitle"), adminAdapterRows());
  appendFlowSummary(body, ui("adminStatusTitle"), adminStatusRows());
  appendFlowSummary(body, ui("adminRoutesTitle"), adminRouteRows());
  const actions = document.createElement("div");
  actions.className = "flow-actions";
  actions.append(
    flowButton(ui("adminStatusAction"), async () => {
      await loadAdminStatus();
      recordReceipt("Adaptadores consultados", `${ADMIN_STATUS_API} ok`, "administracion");
      renderPortal(state.portal);
    }, "primary-action"),
    flowButton(ui("adminCapabilitiesAction"), async () => {
      await loadAdminCapabilities();
      recordReceipt("Capacidades administrativas consultadas", `${ADMIN_CAPABILITIES_API} ok`, "administracion");
      renderPortal(state.portal);
    }, "quiet-action"),
    flowButton(ui("adminAction"), async () => {
      await getData("/healthz", { method: "GET" });
      recordReceipt("Salud API comprobada", "GET /healthz ok", "administracion");
      renderPortal(state.portal);
    }, "quiet-action"),
  );
  body.append(actions);
}

function renderFilters() {
  $$(".filter-bar select").forEach((select) => {
    if (!$("option[value='']", select)) {
      const option = document.createElement("option");
      option.value = "";
      option.textContent = `Todos`;
      select.prepend(option);
    }
    select.value = "";
    state.filters[select.name] = "";
    select.addEventListener("change", () => {
      state.filters[select.name] = select.value;
    });
  });
}

function filteredRows() {
  const query = [state.search, ...Object.values(state.filters)].filter(Boolean).join(" ").toLowerCase();
  return state.rows.filter((row) => {
    const moduleMatches = state.activeModule === "dashboard" || row.modules.includes(state.activeModule);
    if (!moduleMatches) return false;
    if (!query) return true;
    const haystack = [
      row.expediente,
      row.candidate,
      row.state,
      row.stateFilter,
      row.deadline,
      row.deadlineBucket,
      row.points,
      row.document,
      row.action,
      row.scope,
      row.unit,
      row.modules.join(" "),
      ...row.documents.flat(),
      ...row.merits.flat(),
      ...row.alerts.flat(),
      ...row.timeline.flat(),
    ].join(" ").toLowerCase();
    return query.split(/\s+/).every((term) => haystack.includes(term));
  });
}

function selectedRow() {
  return state.rows.find((row) => row.id === state.selectedRowID) || state.rows[0] || null;
}

function hashStateFromLocation() {
  const raw = decodeURIComponent(String(window.location.hash || "").replace(/^#/, "")).trim();
  if (!raw) return { moduleID: "", screenID: "" };
  const [moduleID, screenID] = raw.split("/");
  const validModule = MODULES.some((module) => module.id === moduleID) || flowRenderers[moduleID];
  return {
    moduleID: validModule ? moduleID : "",
    screenID: screenID || "",
  };
}

function moduleFromHash() {
  return hashStateFromLocation().moduleID;
}

function updateLocationHash() {
  const hash = state.activeModule === "dashboard"
    ? "#"
    : `#${encodeURIComponent(state.activeModule)}${state.activeScreen ? `/${encodeURIComponent(state.activeScreen)}` : ""}`;
  if (window.location.hash !== hash) history.replaceState(null, "", hash);
}

function selectRow(rowID) {
  state.selectedRowID = rowID;
  if (state.portal) renderTable(state.portal);
}

function selectListingItem(item) {
  const row = state.rows.find((candidate) => candidate.expediente === item.solicitud_id);
  if (row) {
    state.activeModule = "listados";
    updateModuleSelection();
    selectRow(row.id);
    setStatus(`Listado abierto: ${row.expediente}`, "ready");
  }
}

function setActiveModule(moduleID, screenID = "") {
  state.activeModule = moduleID || "dashboard";
  state.activeScreen = screenID || "";
  state.screenStateFilter = "";
  state.search = "";
  const search = $("#global-search");
  if (search) search.value = "";
  updateModuleSelection();
  renderModuleHeader();
  if (state.portal) {
    renderKPIs(state.portal);
    renderModulePortal(state.portal);
    renderFlowPanel();
    renderTable(state.portal);
    renderCronosPanel(state.portal);
    renderDietasPanel(state.portal);
  }
  updateLocationHash();
  setStatus(`Modulo activo: ${MODULES.find((module) => module.id === state.activeModule)?.label || state.activeModule}`, "ready");
}

function setActiveScreen(screenID) {
  state.activeScreen = screenID || "";
  state.screenStateFilter = "";
  if (state.portal) {
    renderModulePortal(state.portal);
    renderFlowPanel();
    renderTable(state.portal);
  }
  updateLocationHash();
  setStatus(`Pantalla activa: ${activeScreen(state.portal)?.title || state.activeScreen || state.activeModule}`, "ready");
}

function updateModuleSelection() {
  $$(".module-link").forEach((button) => {
    button.setAttribute("aria-current", button.dataset.module === state.activeModule ? "page" : "false");
  });
}

function renderModuleHeader() {
  const copy = MODULE_COPY[state.activeModule] || MODULE_COPY.dashboard;
  const eyebrow = $(".summary-panel .eyebrow");
  const title = $("#summary-title");
  const lead = $(".summary-panel .lead");
  if (eyebrow) eyebrow.textContent = copy[0];
  if (title) title.textContent = copy[1];
  if (lead) {
    lead.textContent = state.activeModule === "dashboard"
      ? "Vista interna para controlar horarios, fichajes, permisos, vacaciones, dietas, kilometraje provincial y expedientes desde un unico portal."
      : moduleLeadText(state.activeModule);
  }
}

function moduleLeadText(moduleID) {
  const leads = {
    personal: "Trabaja sobre expedientes de empleado, puesto, situacion, antiguedad, servicios prestados y certificados.",
    nominas: "Controla el periodo de nomina, conceptos, incidencias, deducciones, cruces con Cronos/Dietas y cierre.",
    cronos: "Gestiona fichajes, incidencias, teletrabajo, saldos diarios, aprobaciones y trazabilidad de jornada.",
    horarios: "Define perfiles horarios por puesto/unidad, flexibilidad, coberturas obligatorias y reducciones 63/64.",
    permisos: "Resuelve solicitudes de asuntos propios, vacaciones, compensaciones y saldos con aprobacion responsable.",
    dietas: "Tramita comisiones de servicio con ruta, kilometraje, justificantes, politica de dieta y liquidacion.",
    rutas: "Consulta y valida kilometraje provincial, paradas intermedias, tiempos estimados y dieta sugerida.",
    bolsa: "Gestiona convocatorias, solicitudes, meritos, certificados consumidos desde Personal y listados.",
    documentos: "Revisa justificantes, CSV/ENI, firmas, versiones y evidencias reutilizables por modulo.",
    aprobaciones: "Bandeja transversal para aprobar, rechazar, devolver o escalar registros de cada modulo.",
    auditoria: "Consulta trazas, recibos, actores, tiempos y exportaciones probatorias del shell VEC.",
    administracion: "Supervisa configuracion, salud de API, catalogos, permisos y colas tecnicas.",
  };
  return leads[moduleID] || MODULE_COPY[moduleID]?.[1] || "Gestion operativa del modulo VEC.";
}

function setActiveTab(button) {
  const tab = button.textContent.trim().toLowerCase();
  state.activeTab = tab === "docs" ? "docs" : tab === "auditoria" ? "auditoria" : tab === "meritos" ? "meritos" : "resumen";
  $$(".tabs [role='tab']").forEach((tabButton) => {
    tabButton.setAttribute("aria-selected", tabButton === button ? "true" : "false");
  });
  renderDetail(state.portal, selectedRow());
}

function updateTabSelection() {
  $$(".tabs [role='tab']").forEach((tabButton) => {
    const tab = tabButton.textContent.trim().toLowerCase();
    const id = tab === "docs" ? "docs" : tab === "auditoria" ? "auditoria" : tab === "meritos" ? "meritos" : "resumen";
    tabButton.setAttribute("aria-selected", id === state.activeTab ? "true" : "false");
  });
}

function nextStateForAction(action) {
  if (/aprobar|validar|liquidar|cerrar/i.test(action)) return "Aprobado / listo";
  if (/justificar|revisar|resolver/i.test(action)) return "En revision por responsable";
  if (/emitir|generar/i.test(action)) return "Pendiente firma";
  if (/editar|aplicar/i.test(action)) return "Cambio aplicado pendiente de cierre";
  return "Accion registrada";
}

async function handleRowAction(row) {
  if (!row) return;
  state.selectedRowID = row.id;
  const action = row.action || "Abrir";
  if (action.toLowerCase().includes("exportar")) {
    exportRows();
    return;
  }
  if (action.toLowerCase().includes("notificaciones")) {
    state.activeModule = "notificaciones";
  }
  if (action.toLowerCase().includes("autobaremo")) {
    state.activeModule = "autobaremo";
    state.activeTab = "meritos";
  }
  const moduleKey = moduleEndpointFor(state.activeModule, row);
  const receipt = await requestModuleReceipt(moduleKey, action);
  const nextState = nextStateForAction(action);
  state.rowOverrides[row.id] = {
    state: nextState,
    stateFilter: /pendiente/i.test(nextState) ? "Pendiente de accion" : "En revision",
    deadline: "Accion registrada",
    deadlineBucket: "Sin vencimiento critico",
    action: /aprobado|listo/i.test(nextState) ? "Abrir" : row.action,
    alerts: [
      [nextState, `${action} ejecutada sobre ${row.expediente}`],
      ["Recibo", receipt?.id || "Auditoria local registrada"],
    ],
    timeline: [
      [action, `${new Date().toLocaleString("es-ES")} - ${receipt?.id || "sin recibo backend"}`],
      ...(row.timeline || []),
    ],
  };
  recordReceipt(action, `${row.expediente} - ${receipt?.id || "flujo local"}`, moduleKey || state.activeModule);
  updateModuleSelection();
  renderModuleHeader();
  renderPortal(state.portal);
  setStatus(`${action}: ${row.expediente}`, "ready");
}

function exportRows() {
  const rows = filteredRows();
  const header = ["expediente", "candidato", "estado", "plazo", "baremo", "documento", "accion"];
  const csv = [
    header.join(";"),
    ...rows.map((row) => [row.expediente, row.candidate, row.state, row.deadline, row.points, row.document, row.action]
      .map((value) => `"${String(value || "").replaceAll('"', '""')}"`).join(";")),
  ].join("\n");
  const blob = new Blob([csv], { type: "text/csv;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = `vec-bolsa-expedientes-${new Date().toISOString().slice(0, 10)}.csv`;
  document.body.append(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
  recordReceipt("Exportacion CSV", `${rows.length} filas exportadas`, "listados");
  setStatus(`Exportadas ${rows.length} filas`, "ready");
  if (state.portal) renderFlowPanel();
}

function setupInteractions() {
  $$(".module-link").forEach((button) => {
    button.addEventListener("click", () => setActiveModule(button.dataset.module));
  });
  $(".search-form")?.addEventListener("submit", (event) => {
    event.preventDefault();
    state.search = String($("#global-search")?.value || "").trim();
    if (state.portal) renderTable(state.portal);
    setStatus(state.search ? `Busqueda aplicada: ${state.search}` : "Busqueda limpia", "ready");
  });
  $(".filter-bar")?.addEventListener("submit", (event) => {
    event.preventDefault();
    if (state.portal) renderTable(state.portal);
    setStatus("Filtros aplicados", "ready");
  });
  $(".filter-bar .quiet-action")?.addEventListener("click", () => {
    if (state.portal) renderTable(state.portal);
    setStatus("Filtros aplicados", "ready");
  });
  $(".queue-panel .panel-header .table-action")?.addEventListener("click", exportRows);
  $(".operator-tools .quiet-action")?.addEventListener("click", () => {
    setActiveModule("aprobaciones");
    const row = state.rows.find((candidate) => candidate.modules.includes("aprobaciones"));
    if (row) state.selectedRowID = row.id;
    renderTable(state.portal);
  });
  $$(".tabs [role='tab']").forEach((button) => {
    button.addEventListener("click", () => setActiveTab(button));
  });
}

function renderPortal(view) {
  state.rows = rowsFromPortal(view);
  if (!state.selectedRowID && state.rows.length) {
    state.selectedRowID = state.rows[0].id;
  }
  renderModuleHeader();
  renderKPIs(view);
  renderModules(view);
  renderModulePortal(view);
  renderFlowPanel();
  renderTable(view);
  renderCronosPanel(view);
  renderDietasPanel(view);
}

async function loadPortal() {
  reloadButton.disabled = true;
  setStatus("Cargando shell VEC", "loading");
  try {
    await loadLocale();
    const [
      portal,
      demo,
      vecModules,
      workspace,
      session,
      rptPositions,
      rptStats,
      categories,
      catalogs,
      ,
      manifest,
      adminStatus,
      adminCapabilities,
    ] = await Promise.all([
      getData(BOLSA_PORTAL_API, { method: "GET", headers: staffHeaders() }),
      loadDemoData(),
      getData(`${VEC_SHELL_API}/modules`, { method: "GET", headers: staffHeaders() }),
      getData(VEC_WORKSPACE_API, { method: "GET", headers: staffHeaders() }),
      getData(VEC_SESSION_API, { method: "GET", headers: staffHeaders() }),
      getData(`${PERSONAL_RPT_POSITIONS_API}?limit=500`, { method: "GET", headers: staffHeaders() }),
      getData(PERSONAL_RPT_STATS_API, { method: "GET", headers: staffHeaders() }),
      getData(`${PERSONAL_CATEGORIES_API}?limit=500`, { method: "GET", headers: staffHeaders() }),
      getData(PERSONAL_CATALOGS_API, { method: "GET", headers: staffHeaders() }),
      loadAPIRootRoutes(),
      loadModuleManifest().catch(() => null),
      loadAdminStatus().catch(() => null),
      loadAdminCapabilities().catch(() => null),
    ]);
    if (manifest) state.moduleManifest = manifest;
    if (adminStatus) state.adminStatus = adminStatus;
    if (adminCapabilities) state.adminCapabilities = adminCapabilities;
    const personalCatalog = normalizePersonalCatalog({ positions: rptPositions, stats: rptStats, categories, catalogs });
    state.demo = demo;
    state.workspace = workspace;
    state.session = session.principal || null;
    state.personalCatalog = personalCatalog;
    state.portal = normalizePortal({
      ...portal,
      principal: state.session || portal.principal,
      vec_modules: vecModules.modules || [],
      workspace,
      personal_catalog: personalCatalog,
    }, demo);
    const hashState = hashStateFromLocation();
    state.activeModule = hashState.moduleID || state.activeModule;
    state.activeScreen = hashState.screenID || state.activeScreen;
    renderPortal(state.portal);
    updateLocationHash();
    setStatus("VEC conectado", "ready");
  } catch (error) {
    setStatus(error.message, "error");
  } finally {
    reloadButton.disabled = false;
  }
}

function getPayrollCalculations(month) {
  const isJune = month === "Junio 2026";
  const sueldoBase = 1113.12;
  const trieniosVal = state.nominasTrieniosCount * 49.59;
  const destVal = state.nominasComplementoDestino;
  const specVal = state.nominasComplementoEspecifico;
  
  let prodVal = state.nominasProductividad;
  if (month === "Marzo 2026") {
    prodVal += 100.00;
  }
  prodVal += state.nominasExtraProductividad;

  let dietasVal = 0;
  if (isJune && state.dietasSheets) {
    dietasVal = state.dietasSheets
      .filter(s => s.estado !== "Borrador")
      .reduce((sum, s) => sum + parseFloat(s.importe || 0), 0);
  }

  const devengos = sueldoBase + trieniosVal + destVal + specVal + prodVal + dietasVal;
  
  const irpf = devengos * (state.nominasIrpfPercent / 100);
  const segSocial = devengos * 0.047;
  const deducciones = irpf + segSocial;

  const liquido = devengos - deducciones;

  return {
    sueldoBase,
    trieniosVal,
    destVal,
    specVal,
    prodVal,
    dietasVal,
    devengos,
    irpf,
    segSocial,
    deducciones,
    liquido
  };
}

function renderCustomNominasApp(container, view) {
  if (state.nominasScreen === undefined) state.nominasScreen = "nomina-mes";
  if (state.nominasSelectedMonth === undefined) state.nominasSelectedMonth = "Junio 2026";
  if (state.nominasIrpfPercent === undefined) state.nominasIrpfPercent = 12.5;
  if (state.nominasTrieniosCount === undefined) state.nominasTrieniosCount = 4;
  if (state.nominasComplementoDestino === undefined) state.nominasComplementoDestino = 562.30;
  if (state.nominasComplementoEspecifico === undefined) state.nominasComplementoEspecifico = 680.44;
  if (state.nominasProductividad === undefined) state.nominasProductividad = 120.00;
  if (state.nominasExtraProductividad === undefined) state.nominasExtraProductividad = 0.00;

  const wrapper = document.createElement("div");
  wrapper.style.fontFamily = "sans-serif";
  wrapper.style.background = "#f4f6f8";
  wrapper.style.minHeight = "600px";
  wrapper.style.borderRadius = "8px";
  wrapper.style.overflow = "hidden";
  wrapper.style.boxShadow = "0 4px 15px rgba(0,0,0,0.1)";

  const header = document.createElement("div");
  header.style.display = "flex";
  header.style.flexDirection = "column";

  const redBar = document.createElement("div");
  redBar.style.background = "#1b5e20";
  redBar.style.color = "#fff";
  redBar.style.padding = "6px 16px";
  redBar.style.fontSize = "0.95rem";
  redBar.style.fontWeight = "bold";
  redBar.style.letterSpacing = "1px";
  redBar.textContent = "DIPUTACIÓN DE GRANADA • AREA DE RECURSOS HUMANOS";

  const yellowBar = document.createElement("div");
  yellowBar.style.background = "#4caf50";
  yellowBar.style.padding = "10px 16px";
  yellowBar.style.fontSize = "1.3rem";
  yellowBar.style.fontWeight = "bold";
  yellowBar.style.fontStyle = "italic";
  yellowBar.style.color = "#fff";
  yellowBar.style.textShadow = "1px 1px 2px rgba(0,0,0,0.4)";
  yellowBar.style.display = "flex";
  yellowBar.style.justifyContent = "space-between";
  yellowBar.style.alignItems = "center";
  
  const titleText = document.createElement("span");
  titleText.textContent = "Portal del Empleado - Consulta de Nóminas";
  yellowBar.append(titleText);

  const backButton = document.createElement("button");
  backButton.type = "button";
  backButton.textContent = "Volver al Tablero";
  backButton.style.background = "#fff";
  backButton.style.color = "#1b5e20";
  backButton.style.border = "none";
  backButton.style.padding = "6px 14px";
  backButton.style.borderRadius = "4px";
  backButton.style.cursor = "pointer";
  backButton.style.fontWeight = "bold";
  backButton.style.fontSize = "0.85rem";
  backButton.addEventListener("click", () => {
    state.activeModule = "dashboard";
    renderModulePortal(view);
  });
  yellowBar.append(backButton);

  header.append(redBar, yellowBar);
  wrapper.append(header);

  const mainGrid = document.createElement("div");
  mainGrid.style.display = "grid";
  mainGrid.style.gridTemplateColumns = "250px 1fr";
  mainGrid.style.minHeight = "520px";

  const sidebar = document.createElement("div");
  sidebar.style.background = "#fff";
  sidebar.style.borderRight = "1px solid #ddd";
  sidebar.style.padding = "20px 10px";
  sidebar.style.display = "flex";
  sidebar.style.flexDirection = "column";
  sidebar.style.gap = "8px";

  const menuItems = [
    { id: "nomina-mes", label: "📄 Nómina Mensual" },
    { id: "historico-evolucion", label: "📈 Histórico y Evolución" },
    { id: "certificado-retenciones", label: "🎓 Certificado Retenciones (10T)" }
  ];

  menuItems.forEach(item => {
    const btn = document.createElement("button");
    btn.type = "button";
    btn.style.width = "100%";
    btn.style.padding = "12px 16px";
    btn.style.border = "none";
    btn.style.borderRadius = "6px";
    btn.style.textAlign = "left";
    btn.style.fontSize = "0.95rem";
    btn.style.fontWeight = "bold";
    btn.style.cursor = "pointer";
    btn.style.transition = "all 0.2s ease";
    
    if (state.nominasScreen === item.id) {
      btn.style.background = "#e8f5e9";
      btn.style.color = "#2e7d32";
      btn.style.borderLeft = "4px solid #2e7d32";
    } else {
      btn.style.background = "transparent";
      btn.style.color = "#555";
      btn.style.borderLeft = "4px solid transparent";
    }

    btn.addEventListener("mouseenter", () => {
      if (state.nominasScreen !== item.id) {
        btn.style.background = "#f1f8e9";
      }
    });
    btn.addEventListener("mouseleave", () => {
      if (state.nominasScreen !== item.id) {
        btn.style.background = "transparent";
      }
    });

    btn.addEventListener("click", () => {
      state.nominasScreen = item.id;
      renderCustomNominasApp(container, view);
    });

    btn.textContent = item.label;
    sidebar.append(btn);
  });

  const infoBadge = document.createElement("div");
  infoBadge.style.marginTop = "auto";
  infoBadge.style.padding = "10px";
  infoBadge.style.background = "#f9f9f9";
  infoBadge.style.borderRadius = "6px";
  infoBadge.style.border = "1px solid #eee";
  infoBadge.style.fontSize = "0.8rem";
  infoBadge.style.color = "#777";
  infoBadge.innerHTML = `
    <div style="font-weight:bold; margin-bottom:4px; color:#555;">Empleado Activo</div>
    <div>ALBERTO SÁNCHEZ GÓMEZ</div>
    <div style="margin-top:2px;">A2 - Técnico de Gestión</div>
    <div style="margin-top:2px; font-family:monospace; color:#888;">NIF: 74839201A</div>
  `;
  sidebar.append(infoBadge);

  mainGrid.append(sidebar);

  const content = document.createElement("div");
  content.style.padding = "24px";
  content.style.overflowY = "auto";

  if (state.nominasScreen === "nomina-mes") {
    renderNominasMesScreen(content, view, container);
  } else if (state.nominasScreen === "historico-evolucion") {
    renderNominasHistoricoScreen(content, view);
  } else if (state.nominasScreen === "certificado-retenciones") {
    renderNominasCertificadoScreen(content, view);
  }

  mainGrid.append(content);
  wrapper.append(mainGrid);
  container.replaceChildren(wrapper);
}

function renderNominasMesScreen(target, view, container) {
  const monthSelectorDiv = document.createElement("div");
  monthSelectorDiv.style.marginBottom = "20px";
  monthSelectorDiv.style.display = "flex";
  monthSelectorDiv.style.alignItems = "center";
  monthSelectorDiv.style.gap = "12px";

  const label = document.createElement("label");
  label.style.fontWeight = "bold";
  label.style.color = "#333";
  label.textContent = "Seleccione el mes a consultar:";
  monthSelectorDiv.append(label);

  const select = document.createElement("select");
  select.style.padding = "6px 12px";
  select.style.borderRadius = "4px";
  select.style.border = "1px solid #ccc";
  select.style.fontSize = "0.95rem";
  select.style.cursor = "pointer";

  const months = ["Junio 2026", "Mayo 2026", "Abril 2026", "Marzo 2026", "Febrero 2026", "Enero 2026"];
  months.forEach(m => {
    const opt = document.createElement("option");
    opt.value = m;
    opt.textContent = m;
    if (state.nominasSelectedMonth === m) opt.selected = true;
    select.append(opt);
  });

  select.addEventListener("change", (e) => {
    state.nominasSelectedMonth = e.target.value;
    renderCustomNominasApp(container, view);
  });
  monthSelectorDiv.append(select);
  target.append(monthSelectorDiv);

  const calc = getPayrollCalculations(state.nominasSelectedMonth);

  const kpiRow = document.createElement("div");
  kpiRow.style.display = "grid";
  kpiRow.style.gridTemplateColumns = "repeat(auto-fit, minmax(200px, 1fr))";
  kpiRow.style.gap = "16px";
  kpiRow.style.marginBottom = "24px";

  const kpis = [
    { title: "LÍQUIDO A RECEBIR", val: `${calc.liquido.toFixed(2)} €`, color: "#1b5e20", bg: "#e8f5e9" },
    { title: "TOTAL DEVENGOS (BRUTO)", val: `${calc.devengos.toFixed(2)} €`, color: "#0d47a1", bg: "#e3f2fd" },
    { title: "TOTAL DEDUCCIONES", val: `${calc.deducciones.toFixed(2)} €`, color: "#b71c1c", bg: "#ffebee" }
  ];

  kpis.forEach(k => {
    const card = document.createElement("div");
    card.style.background = k.bg;
    card.style.borderLeft = `6px solid ${k.color}`;
    card.style.padding = "16px";
    card.style.borderRadius = "6px";
    card.style.boxShadow = "0 2px 4px rgba(0,0,0,0.05)";

    const cardTitle = document.createElement("div");
    cardTitle.style.fontSize = "0.75rem";
    cardTitle.style.fontWeight = "bold";
    cardTitle.style.color = "#666";
    cardTitle.style.letterSpacing = "0.5px";
    cardTitle.style.marginBottom = "4px";
    cardTitle.textContent = k.title;

    const cardVal = document.createElement("div");
    cardVal.style.fontSize = "1.6rem";
    cardVal.style.fontWeight = "bold";
    cardVal.style.color = k.color;
    cardVal.textContent = k.val;

    card.append(cardTitle, cardVal);
    kpiRow.append(card);
  });

  target.append(kpiRow);

  const layout = document.createElement("div");
  layout.style.display = "flex";
  layout.style.flexWrap = "wrap";
  layout.style.gap = "24px";

  const payslip = document.createElement("div");
  payslip.style.flex = "2 1 550px";
  payslip.style.background = "#fff";
  payslip.style.border = "1px solid #ddd";
  payslip.style.borderRadius = "8px";
  payslip.style.padding = "30px";
  payslip.style.boxShadow = "0 4px 10px rgba(0,0,0,0.03)";
  payslip.style.position = "relative";

  const printBtn = document.createElement("button");
  printBtn.type = "button";
  printBtn.style.position = "absolute";
  printBtn.style.top = "15px";
  printBtn.style.right = "15px";
  printBtn.style.background = "#f5f5f5";
  printBtn.style.border = "1px solid #ccc";
  printBtn.style.borderRadius = "4px";
  printBtn.style.padding = "6px 12px";
  printBtn.style.fontSize = "0.8rem";
  printBtn.style.cursor = "pointer";
  printBtn.style.display = "flex";
  printBtn.style.alignItems = "center";
  printBtn.style.gap = "6px";
  printBtn.innerHTML = `🖨️ Imprimir`;
  printBtn.addEventListener("click", () => {
    window.print();
  });
  payslip.append(printBtn);

  const payslipHeader = document.createElement("div");
  payslipHeader.style.display = "flex";
  payslipHeader.style.justifyContent = "space-between";
  payslipHeader.style.borderBottom = "2px solid #333";
  payslipHeader.style.paddingBottom = "15px";
  payslipHeader.style.marginBottom = "20px";

  const logoCol = document.createElement("div");
  logoCol.innerHTML = `
    <div style="font-weight:bold; font-size:1.1rem; color:#1b5e20;">DIPUTACIÓN DE GRANADA</div>
    <div style="font-size:0.75rem; color:#666; margin-top:2px;">Organismo Pagador: Área de Personal y Régimen Interior</div>
    <div style="font-size:0.75rem; color:#666; font-family:monospace;">C.I.F.: P1800000J</div>
  `;

  const titleCol = document.createElement("div");
  titleCol.style.textAlign = "right";
  titleCol.innerHTML = `
    <div style="font-weight:bold; font-size:1rem; color:#333;">RECIBO DE SALARIOS</div>
    <div style="font-size:0.85rem; color:#2e7d32; font-weight:bold; margin-top:4px;">MES: ${state.nominasSelectedMonth.toUpperCase()}</div>
  `;
  payslipHeader.append(logoCol, titleCol);
  payslip.append(payslipHeader);

  const detailsGrid = document.createElement("div");
  detailsGrid.style.display = "grid";
  detailsGrid.style.gridTemplateColumns = "repeat(auto-fit, minmax(200px, 1fr))";
  detailsGrid.style.gap = "12px";
  detailsGrid.style.marginBottom = "24px";
  detailsGrid.style.padding = "10px";
  detailsGrid.style.background = "#fcfcfc";
  detailsGrid.style.border = "1px solid #eee";
  detailsGrid.style.borderRadius = "4px";
  detailsGrid.style.fontSize = "0.8rem";

  detailsGrid.innerHTML = `
    <div><strong>Empleado:</strong> ALBERTO SÁNCHEZ GÓMEZ</div>
    <div><strong>NIF:</strong> 74839201A</div>
    <div><strong>C. Servicio:</strong> TRANSFORMACIÓN DIGITAL / NUEVAS TECNOLOGÍAS</div>
    <div><strong>Puesto:</strong> TÉCNICO DE GESTIÓN (A2)</div>
    <div><strong>Nº Trienios:</strong> ${state.nominasTrieniosCount.toString().padStart(2, '0')}</div>
    <div><strong>Ads.Adm.:</strong> FUNCIONARIO DE CARRERA</div>
    <div><strong>IBAN:</strong> ES91 2100 0482 12 0123456789</div>
    <div><strong>Nº Afiliación:</strong> 18/1234567-89</div>
  `;
  payslip.append(detailsGrid);

  const table = document.createElement("table");
  table.style.width = "100%";
  table.style.borderCollapse = "collapse";
  table.style.fontSize = "0.85rem";

  const thStyle = "background:#f1f8e9; color:#2e7d32; padding:10px; border:1px solid #ddd; text-align:left; font-weight:bold;";
  table.innerHTML = `
    <thead>
      <tr>
        <th style="${thStyle}">CÓDIGO - CONCEPTO</th>
        <th style="${thStyle} text-align:right; width:120px;">DEVENGOS (€)</th>
        <th style="${thStyle} text-align:right; width:120px;">DEDUCCIONES (€)</th>
      </tr>
    </thead>
    <tbody>
      <tr>
        <td style="padding:10px; border:1px solid #ddd;">11 - Sueldo Base (Grupo A2)</td>
        <td style="padding:10px; border:1px solid #ddd; text-align:right;">${calc.sueldoBase.toFixed(2)}</td>
        <td style="padding:10px; border:1px solid #ddd; text-align:right; color:#999;">-</td>
      </tr>
      <tr>
        <td style="padding:10px; border:1px solid #ddd;">12 - Trienios acumulados (${state.nominasTrieniosCount})</td>
        <td style="padding:10px; border:1px solid #ddd; text-align:right;">${calc.trieniosVal.toFixed(2)}</td>
        <td style="padding:10px; border:1px solid #ddd; text-align:right; color:#999;">-</td>
      </tr>
      <tr>
        <td style="padding:10px; border:1px solid #ddd;">52 - Complemento de Destino (Nivel 22)</td>
        <td style="padding:10px; border:1px solid #ddd; text-align:right;">${calc.destVal.toFixed(2)}</td>
        <td style="padding:10px; border:1px solid #ddd; text-align:right; color:#999;">-</td>
      </tr>
      ${state.nominasComplementoEspecifico > 0 ? `
      <tr class="spec-row">
        <td style="padding:10px; border:1px solid #ddd;">53 - Complemento Específico</td>
        <td style="padding:10px; border:1px solid #ddd; text-align:right;">${calc.specVal.toFixed(2)}</td>
        <td style="padding:10px; border:1px solid #ddd; text-align:right; color:#999;">-</td>
      </tr>` : ''}
      <tr>
        <td style="padding:10px; border:1px solid #ddd;">55 - Productividad e Incentivos</td>
        <td style="padding:10px; border:1px solid #ddd; text-align:right;">${calc.prodVal.toFixed(2)}</td>
        <td style="padding:10px; border:1px solid #ddd; text-align:right; color:#999;">-</td>
      </tr>
      ${calc.dietasVal > 0 ? `
      <tr style="background:#e8f5e9; font-weight:bold;" class="dietas-row">
        <td style="padding:10px; border:1px solid #ddd; color:#1b5e20;">59 - Dietas y Locomoción (Cruce VEC)</td>
        <td style="padding:10px; border:1px solid #ddd; text-align:right; color:#1b5e20;">${calc.dietasVal.toFixed(2)}</td>
        <td style="padding:10px; border:1px solid #ddd; text-align:right; color:#999;">-</td>
      </tr>` : ''}
      <tr>
        <td style="padding:10px; border:1px solid #ddd;">1 - I.R.P.F. Retenciones Practicadas (${state.nominasIrpfPercent}%)</td>
        <td style="padding:10px; border:1px solid #ddd; text-align:right; color:#999;">-</td>
        <td style="padding:10px; border:1px solid #ddd; text-align:right; color:#c62828;">${calc.irpf.toFixed(2)}</td>
      </tr>
      <tr>
        <td style="padding:10px; border:1px solid #ddd;">3 - Cotización General Seguridad Social (4.7%)</td>
        <td style="padding:10px; border:1px solid #ddd; text-align:right; color:#999;">-</td>
        <td style="padding:10px; border:1px solid #ddd; text-align:right; color:#c62828;">${calc.segSocial.toFixed(2)}</td>
      </tr>
      <tr style="background:#f9f9f9; font-weight:bold;">
        <td style="padding:12px; border:1px solid #ddd; text-align:right;">TOTALES (€):</td>
        <td style="padding:12px; border:1px solid #ddd; text-align:right; color:#0d47a1;">${calc.devengos.toFixed(2)}</td>
        <td style="padding:12px; border:1px solid #ddd; text-align:right; color:#c62828;">${calc.deducciones.toFixed(2)}</td>
      </tr>
    </tbody>
  `;
  payslip.append(table);

  const netoBox = document.createElement("div");
  netoBox.style.marginTop = "20px";
  netoBox.style.background = "#e8f5e9";
  netoBox.style.border = "1px solid #a5d6a7";
  netoBox.style.borderRadius = "6px";
  netoBox.style.padding = "16px";
  netoBox.style.display = "flex";
  netoBox.style.justifyContent = "space-between";
  netoBox.style.alignItems = "center";

  const netoLabel = document.createElement("div");
  netoLabel.innerHTML = `
    <div style="font-weight:bold; color:#1b5e20; font-size:1.05rem;">Líquido a percibir:</div>
    <div style="font-size:0.75rem; color:#555; margin-top:2px;">Fórmula: Total Devengos (1) - Total Deducciones (3)</div>
  `;

  const netoVal = document.createElement("div");
  netoVal.style.fontSize = "1.8rem";
  netoVal.style.fontWeight = "bold";
  netoVal.style.color = "#1b5e20";
  netoVal.textContent = `${calc.liquido.toFixed(2)} €`;

  netoBox.append(netoLabel, netoVal);
  payslip.append(netoBox);

  const payslipFooter = document.createElement("div");
  payslipFooter.style.marginTop = "20px";
  payslipFooter.style.borderTop = "1px solid #eee";
  payslipFooter.style.paddingTop = "15px";
  payslipFooter.style.display = "flex";
  payslipFooter.style.justifyContent = "space-between";
  payslipFooter.style.fontSize = "0.7rem";
  payslipFooter.style.color = "#888";
  payslipFooter.innerHTML = `
    <div>Código Seguro de Verificación (CSV): CSV-9382-AJ84-29E1-401C</div>
    <div style="text-align:right;">Documento firmado electrónicamente por la Diputación Provincial de Granada</div>
  `;
  payslip.append(payslipFooter);

  layout.append(payslip);

  const simCard = document.createElement("div");
  simCard.style.flex = "1 1 280px";
  simCard.style.background = "#fff";
  simCard.style.border = "1px solid #ddd";
  simCard.style.borderRadius = "8px";
  simCard.style.padding = "20px";
  simCard.style.boxShadow = "0 4px 10px rgba(0,0,0,0.03)";
  simCard.style.display = "flex";
  simCard.style.flexDirection = "column";
  simCard.style.gap = "18px";

  const simHeader = document.createElement("h3");
  simHeader.style.margin = "0";
  simHeader.style.fontSize = "1.1rem";
  simHeader.style.color = "#1b5e20";
  simHeader.style.fontWeight = "bold";
  simHeader.style.borderBottom = "1px solid #eee";
  simHeader.style.paddingBottom = "10px";
  simHeader.textContent = "Simulador de Retribuciones ⚙️";
  simCard.append(simHeader);

  const irpfDiv = document.createElement("div");
  irpfDiv.style.display = "flex";
  irpfDiv.style.flexDirection = "column";
  irpfDiv.style.gap = "6px";
  
  const irpfLabel = document.createElement("label");
  irpfLabel.style.fontSize = "0.85rem";
  irpfLabel.style.fontWeight = "bold";
  irpfLabel.style.color = "#555";
  irpfLabel.innerHTML = `Retención IRPF: <span style="color:#2e7d32; font-weight:bold;">${state.nominasIrpfPercent}%</span>`;
  
  const irpfInput = document.createElement("input");
  irpfInput.type = "range";
  irpfInput.min = "10";
  irpfInput.max = "30";
  irpfInput.step = "0.5";
  irpfInput.value = state.nominasIrpfPercent;
  irpfInput.style.cursor = "pointer";
  irpfInput.style.accentColor = "#2e7d32";

  irpfInput.addEventListener("input", (e) => {
    state.nominasIrpfPercent = parseFloat(e.target.value);
    irpfLabel.innerHTML = `Retención IRPF: <span style="color:#2e7d32; font-weight:bold;">${state.nominasIrpfPercent}%</span>`;
    updateLiveCalculations();
  });

  irpfDiv.append(irpfLabel, irpfInput);
  simCard.append(irpfDiv);

  const trieniosDiv = document.createElement("div");
  trieniosDiv.style.display = "flex";
  trieniosDiv.style.flexDirection = "column";
  trieniosDiv.style.gap = "6px";

  const trieniosLabel = document.createElement("label");
  trieniosLabel.style.fontSize = "0.85rem";
  trieniosLabel.style.fontWeight = "bold";
  trieniosLabel.style.color = "#555";
  trieniosLabel.textContent = "Número de Trienios:";

  const trieniosSelect = document.createElement("select");
  trieniosSelect.style.padding = "6px";
  trieniosSelect.style.borderRadius = "4px";
  trieniosSelect.style.border = "1px solid #ccc";
  trieniosSelect.style.fontSize = "0.85rem";
  
  for(let i = 0; i <= 10; i++) {
    const opt = document.createElement("option");
    opt.value = i;
    opt.textContent = `${i} trienios (${(i * 49.59).toFixed(2)} €)`;
    if (state.nominasTrieniosCount === i) opt.selected = true;
    trieniosSelect.append(opt);
  }

  trieniosSelect.addEventListener("change", (e) => {
    state.nominasTrieniosCount = parseInt(e.target.value);
    updateLiveCalculations();
  });

  trieniosDiv.append(trieniosLabel, trieniosSelect);
  simCard.append(trieniosDiv);

  const specDiv = document.createElement("div");
  specDiv.style.display = "flex";
  specDiv.style.alignItems = "center";
  specDiv.style.gap = "8px";

  const specCheckbox = document.createElement("input");
  specCheckbox.type = "checkbox";
  specCheckbox.id = "spec-checkbox";
  specCheckbox.style.cursor = "pointer";
  specCheckbox.style.accentColor = "#2e7d32";
  if (state.nominasComplementoEspecifico > 0) specCheckbox.checked = true;

  const specLabel = document.createElement("label");
  specLabel.htmlFor = "spec-checkbox";
  specLabel.style.fontSize = "0.85rem";
  specLabel.style.fontWeight = "bold";
  specLabel.style.color = "#555";
  specLabel.style.cursor = "pointer";
  specLabel.textContent = "Incluir Compl. Específico (680,44 €)";

  specCheckbox.addEventListener("change", (e) => {
    state.nominasComplementoEspecifico = e.target.checked ? 680.44 : 0.00;
    updateLiveCalculations();
  });

  specDiv.append(specCheckbox, specLabel);
  simCard.append(specDiv);

  const extraProdDiv = document.createElement("div");
  extraProdDiv.style.display = "flex";
  extraProdDiv.style.flexDirection = "column";
  extraProdDiv.style.gap = "6px";

  const extraProdLabel = document.createElement("label");
  extraProdLabel.style.fontSize = "0.85rem";
  extraProdLabel.style.fontWeight = "bold";
  extraProdLabel.style.color = "#555";
  extraProdLabel.textContent = "Productividad Variable Extra (€):";

  const extraProdInput = document.createElement("input");
  extraProdInput.type = "number";
  extraProdInput.min = "0";
  extraProdInput.max = "2000";
  extraProdInput.step = "50";
  extraProdInput.value = state.nominasExtraProductividad;
  extraProdInput.style.padding = "6px";
  extraProdInput.style.borderRadius = "4px";
  extraProdInput.style.border = "1px solid #ccc";
  extraProdInput.style.fontSize = "0.85rem";

  extraProdInput.addEventListener("input", (e) => {
    state.nominasExtraProductividad = parseFloat(e.target.value || 0);
    updateLiveCalculations();
  });

  extraProdDiv.append(extraProdLabel, extraProdInput);
  simCard.append(extraProdDiv);

  const crossModuleInfo = document.createElement("div");
  crossModuleInfo.style.marginTop = "auto";
  crossModuleInfo.style.padding = "12px";
  crossModuleInfo.style.borderRadius = "6px";
  crossModuleInfo.style.fontSize = "0.8rem";
  
  if (state.nominasSelectedMonth === "Junio 2026" && calc.dietasVal > 0) {
    crossModuleInfo.style.background = "#e8f5e9";
    crossModuleInfo.style.border = "1px solid #a5d6a7";
    crossModuleInfo.style.color = "#2e7d32";
    crossModuleInfo.innerHTML = `
      <div style="font-weight:bold; margin-bottom:4px;">🔄 Cruce VEC Activo</div>
      Se han importado automáticamente <strong>${calc.dietasVal.toFixed(2)} €</strong> en dietas aprobadas para este mes.
    `;
  } else {
    crossModuleInfo.style.background = "#fff8e1";
    crossModuleInfo.style.border = "1px solid #ffe082";
    crossModuleInfo.style.color = "#b77f00";
    crossModuleInfo.innerHTML = `
      <div style="font-weight:bold; margin-bottom:4px;">ℹ️ Cruce VEC Dietas</div>
      No se registran comisiones liquidadas para el mes de ${state.nominasSelectedMonth}.
    `;
  }
  simCard.append(crossModuleInfo);

  layout.append(simCard);
  target.append(layout);

  function updateLiveCalculations() {
    const updated = getPayrollCalculations(state.nominasSelectedMonth);
    
    kpiRow.children[0].querySelector("div:last-child").textContent = `${updated.liquido.toFixed(2)} €`;
    kpiRow.children[1].querySelector("div:last-child").textContent = `${updated.devengos.toFixed(2)} €`;
    kpiRow.children[2].querySelector("div:last-child").textContent = `${updated.deducciones.toFixed(2)} €`;

    table.querySelector("tbody").innerHTML = `
      <tr>
        <td style="padding:10px; border:1px solid #ddd;">11 - Sueldo Base (Grupo A2)</td>
        <td style="padding:10px; border:1px solid #ddd; text-align:right;">${updated.sueldoBase.toFixed(2)}</td>
        <td style="padding:10px; border:1px solid #ddd; text-align:right; color:#999;">-</td>
      </tr>
      <tr>
        <td style="padding:10px; border:1px solid #ddd;">12 - Trienios acumulados (${state.nominasTrieniosCount})</td>
        <td style="padding:10px; border:1px solid #ddd; text-align:right;">${updated.trieniosVal.toFixed(2)}</td>
        <td style="padding:10px; border:1px solid #ddd; text-align:right; color:#999;">-</td>
      </tr>
      <tr>
        <td style="padding:10px; border:1px solid #ddd;">52 - Complemento de Destino (Nivel 22)</td>
        <td style="padding:10px; border:1px solid #ddd; text-align:right;">${updated.destVal.toFixed(2)}</td>
        <td style="padding:10px; border:1px solid #ddd; text-align:right; color:#999;">-</td>
      </tr>
      ${state.nominasComplementoEspecifico > 0 ? `
      <tr>
        <td style="padding:10px; border:1px solid #ddd;">53 - Complemento Específico</td>
        <td style="padding:10px; border:1px solid #ddd; text-align:right;">${updated.specVal.toFixed(2)}</td>
        <td style="padding:10px; border:1px solid #ddd; text-align:right; color:#999;">-</td>
      </tr>` : ''}
      <tr>
        <td style="padding:10px; border:1px solid #ddd;">55 - Productividad e Incentivos</td>
        <td style="padding:10px; border:1px solid #ddd; text-align:right;">${updated.prodVal.toFixed(2)}</td>
        <td style="padding:10px; border:1px solid #ddd; text-align:right; color:#999;">-</td>
      </tr>
      ${updated.dietasVal > 0 ? `
      <tr style="background:#e8f5e9; font-weight:bold;">
        <td style="padding:10px; border:1px solid #ddd; color:#1b5e20;">59 - Dietas y Locomoción (Cruce VEC)</td>
        <td style="padding:10px; border:1px solid #ddd; text-align:right; color:#1b5e20;">${updated.dietasVal.toFixed(2)}</td>
        <td style="padding:10px; border:1px solid #ddd; text-align:right; color:#999;">-</td>
      </tr>` : ''}
      <tr>
        <td style="padding:10px; border:1px solid #ddd;">1 - I.R.P.F. Retenciones Practicadas (${state.nominasIrpfPercent}%)</td>
        <td style="padding:10px; border:1px solid #ddd; text-align:right; color:#999;">-</td>
        <td style="padding:10px; border:1px solid #ddd; text-align:right; color:#c62828;">${updated.irpf.toFixed(2)}</td>
      </tr>
      <tr>
        <td style="padding:10px; border:1px solid #ddd;">3 - Cotización General Seguridad Social (4.7%)</td>
        <td style="padding:10px; border:1px solid #ddd; text-align:right; color:#999;">-</td>
        <td style="padding:10px; border:1px solid #ddd; text-align:right; color:#c62828;">${updated.segSocial.toFixed(2)}</td>
      </tr>
      <tr style="background:#f9f9f9; font-weight:bold;">
        <td style="padding:12px; border:1px solid #ddd; text-align:right;">TOTALES (€):</td>
        <td style="padding:12px; border:1px solid #ddd; text-align:right; color:#0d47a1;">${updated.devengos.toFixed(2)}</td>
        <td style="padding:12px; border:1px solid #ddd; text-align:right; color:#c62828;">${updated.deducciones.toFixed(2)}</td>
      </tr>
    `;

    netoVal.textContent = `${updated.liquido.toFixed(2)} €`;
    detailsGrid.querySelector("div:nth-child(5)").innerHTML = `<strong>Nº Trienios:</strong> ${state.nominasTrieniosCount.toString().padStart(2, '0')}`;
  }
}

function renderNominasHistoricoScreen(target, view) {
  const title = document.createElement("h3");
  title.style.margin = "0 0 16px 0";
  title.style.color = "#1b5e20";
  title.style.fontSize = "1.2rem";
  title.style.fontWeight = "bold";
  title.textContent = "Evolución Salarial de los últimos 6 meses";
  target.append(title);

  const chartWrapper = document.createElement("div");
  chartWrapper.style.background = "#fff";
  chartWrapper.style.border = "1px solid #ddd";
  chartWrapper.style.borderRadius = "8px";
  chartWrapper.style.padding = "24px";
  chartWrapper.style.marginBottom = "24px";
  chartWrapper.style.boxShadow = "0 2px 4px rgba(0,0,0,0.02)";

  const chartContainer = document.createElement("div");
  chartContainer.style.display = "flex";
  chartContainer.style.justifyContent = "space-around";
  chartContainer.style.alignItems = "flex-end";
  chartContainer.style.height = "240px";
  chartContainer.style.borderBottom = "2px solid #ccc";
  chartContainer.style.paddingBottom = "10px";
  chartContainer.style.position = "relative";

  const months = ["Enero 2026", "Febrero 2026", "Marzo 2026", "Abril 2026", "Mayo 2026", "Junio 2026"];
  
  months.forEach(m => {
    const calc = getPayrollCalculations(m);
    
    const monthCol = document.createElement("div");
    monthCol.style.display = "flex";
    monthCol.style.flexDirection = "column";
    monthCol.style.alignItems = "center";
    monthCol.style.gap = "8px";
    monthCol.style.width = "80px";

    const barsDiv = document.createElement("div");
    barsDiv.style.display = "flex";
    barsDiv.style.alignItems = "flex-end";
    barsDiv.style.gap = "6px";
    barsDiv.style.height = "180px";

    const grossBarHeight = (calc.devengos / 3500) * 160;
    const grossBar = document.createElement("div");
    grossBar.style.width = "18px";
    grossBar.style.height = `${grossBarHeight}px`;
    grossBar.style.background = "#42a5f5";
    grossBar.style.borderRadius = "3px 3px 0 0";
    grossBar.style.cursor = "pointer";
    grossBar.style.transition = "all 0.3s ease";
    grossBar.title = `Bruto: ${calc.devengos.toFixed(2)} €`;
    
    const netBarHeight = (calc.liquido / 3500) * 160;
    const netBar = document.createElement("div");
    netBar.style.width = "18px";
    netBar.style.height = `${netBarHeight}px`;
    netBar.style.background = "#66bb6a";
    netBar.style.borderRadius = "3px 3px 0 0";
    netBar.style.cursor = "pointer";
    netBar.style.transition = "all 0.3s ease";
    netBar.title = `Neto: ${calc.liquido.toFixed(2)} €`;

    [grossBar, netBar].forEach(bar => {
      bar.addEventListener("mouseenter", () => {
        bar.style.opacity = "0.85";
        bar.style.transform = "scaleY(1.05)";
      });
      bar.addEventListener("mouseleave", () => {
        bar.style.opacity = "1";
        bar.style.transform = "scaleY(1)";
      });
    });

    barsDiv.append(grossBar, netBar);

    const monthLabel = document.createElement("span");
    monthLabel.style.fontSize = "0.75rem";
    monthLabel.style.fontWeight = "bold";
    monthLabel.style.color = "#666";
    monthLabel.textContent = m.split(" ")[0];

    monthCol.append(barsDiv, monthLabel);
    chartContainer.append(monthCol);
  });

  chartWrapper.append(chartContainer);

  const legend = document.createElement("div");
  legend.style.display = "flex";
  legend.style.justifyContent = "center";
  legend.style.gap = "20px";
  legend.style.marginTop = "12px";
  legend.style.fontSize = "0.8rem";
  legend.style.color = "#555";
  legend.innerHTML = `
    <div style="display:flex; align-items:center; gap:6px;">
      <div style="width:12px; height:12px; background:#42a5f5; border-radius:2px;"></div>
      <span>Total Devengos (Bruto)</span>
    </div>
    <div style="display:flex; align-items:center; gap:6px;">
      <div style="width:12px; height:12px; background:#66bb6a; border-radius:2px;"></div>
      <span>Líquido a percibir (Neto)</span>
    </div>
  `;
  chartWrapper.append(legend);
  target.append(chartWrapper);

  const tableTitle = document.createElement("h4");
  tableTitle.style.margin = "0 0 12px 0";
  tableTitle.style.color = "#333";
  tableTitle.textContent = "Histórico de Recibos de Nómina";
  target.append(tableTitle);

  const table = document.createElement("table");
  table.style.width = "100%";
  table.style.borderCollapse = "collapse";
  table.style.fontSize = "0.85rem";
  table.style.background = "#fff";
  table.style.border = "1px solid #ddd";
  table.style.borderRadius = "4px";

  const thStyle = "background:#f9f9f9; padding:10px; border-bottom:2px solid #ddd; text-align:left; font-weight:bold; color:#555;";
  table.innerHTML = `
    <thead>
      <tr>
        <th style="${thStyle}">PERIODO</th>
        <th style="${thStyle}">CATEGORÍA</th>
        <th style="${thStyle} text-align:right;">BRUTO (€)</th>
        <th style="${thStyle} text-align:right;">NETO (€)</th>
        <th style="${thStyle} text-align:center;">ACCIONES</th>
      </tr>
    </thead>
    <tbody>
      ${months.map(m => {
        const calc = getPayrollCalculations(m);
        return `
          <tr style="border-bottom:1px solid #eee;">
            <td style="padding:10px; font-weight:bold;">${m}</td>
            <td style="padding:10px; color:#666;">Técnico de Gestión (A2)</td>
            <td style="padding:10px; text-align:right; font-weight:bold; color:#0d47a1;">${calc.devengos.toFixed(2)} €</td>
            <td style="padding:10px; text-align:right; font-weight:bold; color:#1b5e20;">${calc.liquido.toFixed(2)} €</td>
            <td style="padding:10px; text-align:center;">
              <button type="button" class="btn-visualizar" data-month="${m}" style="background:#2e7d32; color:#fff; border:none; padding:4px 8px; border-radius:4px; cursor:pointer; font-size:0.75rem; font-weight:bold; margin-right:6px;">Ver Recibo</button>
              <button type="button" class="btn-pdf-dummy" style="background:#f5f5f5; color:#333; border:1px solid #ccc; padding:3px 8px; border-radius:4px; cursor:pointer; font-size:0.75rem;">PDF</button>
            </td>
          </tr>
        `;
      }).join("")}
    </tbody>
  `;

  table.querySelectorAll(".btn-visualizar").forEach(btn => {
    btn.addEventListener("click", (e) => {
      state.nominasSelectedMonth = e.target.getAttribute("data-month");
      state.nominasScreen = "nomina-mes";
      renderCustomNominasApp(target.parentElement, view);
    });
  });

  table.querySelectorAll(".btn-pdf-dummy").forEach(btn => {
    btn.addEventListener("click", () => {
      alert("Generando y descargando PDF firmado digitalmente...");
    });
  });

  target.append(table);
}

function renderNominasCertificadoScreen(target, view) {
  const title = document.createElement("h3");
  title.style.margin = "0 0 16px 0";
  title.style.color = "#1b5e20";
  title.style.fontSize = "1.2rem";
  title.style.fontWeight = "bold";
  title.textContent = "Certificado de Retenciones e Ingresos a Cuenta (I.R.P.F.)";
  target.append(title);

  const certCard = document.createElement("div");
  certCard.style.background = "#fff";
  certCard.style.border = "1px solid #ddd";
  certCard.style.borderRadius = "8px";
  certCard.style.padding = "30px";
  certCard.style.boxShadow = "0 4px 10px rgba(0,0,0,0.03)";
  certCard.style.fontSize = "0.85rem";

  certCard.innerHTML = `
    <div style="display:flex; justify-content:space-between; border-bottom:2px solid #000; padding-bottom:15px; margin-bottom:20px;">
      <div>
        <div style="font-weight:bold; font-size:1rem;">DIPUTACIÓN PROVINCIAL DE GRANADA</div>
        <div style="font-size:0.75rem; color:#555;">Área de Recursos Humanos y Régimen Interior</div>
      </div>
      <div style="text-align:right;">
        <div style="font-weight:bold; font-size:1.1rem;">EJERCICIO FISCAL 2025</div>
        <div style="font-size:0.75rem; color:#555; font-weight:bold;">MODELO 10T</div>
      </div>
    </div>

    <div style="margin-bottom:20px;">
      <h4 style="margin:0 0 10px 0; border-bottom:1px solid #eee; padding-bottom:4px; color:#1b5e20;">DATOS DEL PERCEPTOR</h4>
      <div style="display:grid; grid-template-columns:1fr 1fr; gap:10px;">
        <div><strong>Nombre y Apellidos:</strong> ALBERTO SÁNCHEZ GÓMEZ</div>
        <div><strong>N.I.F. Perceptor:</strong> 74839201A</div>
        <div><strong>Puesto de Trabajo:</strong> TÉCNICO DE GESTIÓN (A2)</div>
        <div><strong>Relación Jurídica:</strong> Funcionario de Carrera</div>
      </div>
    </div>

    <div style="margin-bottom:24px;">
      <h4 style="margin:0 0 10px 0; border-bottom:1px solid #eee; padding-bottom:4px; color:#1b5e20;">RENDIMIENTOS DEL TRABAJO</h4>
      <table style="width:100%; border-collapse:collapse; margin-top:8px;">
        <thead>
          <tr style="background:#f9f9f9; font-weight:bold;">
            <th style="padding:8px; border:1px solid #ddd; text-align:left;">CONCEPTO VALORABLE</th>
            <th style="padding:8px; border:1px solid #ddd; text-align:right; width:150px;">IMPORTE ANUAL (€)</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td style="padding:8px; border:1px solid #ddd;">Percepciones íntegras satisfechas (Bruto)</td>
            <td style="padding:8px; border:1px solid #ddd; text-align:right; font-weight:bold;">32.090,64</td>
          </tr>
          <tr>
            <td style="padding:8px; border:1px solid #ddd;">Retenciones practicadas a cuenta del I.R.P.F.</td>
            <td style="padding:8px; border:1px solid #ddd; text-align:right; font-weight:bold; color:#c62828;">4.011,33</td>
          </tr>
          <tr>
            <td style="padding:8px; border:1px solid #ddd;">Gastos deducibles (Cotizaciones Seguridad Social / MUFACE)</td>
            <td style="padding:8px; border:1px solid #ddd; text-align:right; font-weight:bold; color:#0d47a1;">1.508,26</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div style="border-top:1px solid #eee; padding-top:20px; display:flex; justify-content:space-between; align-items:center;">
      <div style="color:#666; font-size:0.75rem;">
        <div>Firmado electrónicamente por el Ilmo. Sr. Diputado Delegado de Recursos Humanos</div>
        <div style="margin-top:2px; font-family:monospace;">Código seguro de verificación: CSV-CERT-10T-2025-9988</div>
      </div>
      <button type="button" id="btn-descargar-firmado" style="background:#1b5e20; color:#fff; border:none; padding:10px 20px; border-radius:6px; cursor:pointer; font-weight:bold; font-size:0.9rem; transition: background 0.2s;">
        💾 Descargar Certificado Firmado
      </button>
    </div>
  `;

  const btnDescargar = certCard.querySelector("#btn-descargar-firmado");
  btnDescargar.addEventListener("click", () => {
    openCertificateSignatureModal();
  });

  target.append(certCard);

  function openCertificateSignatureModal() {
    const modalOverlay = document.createElement("div");
    modalOverlay.style.position = "fixed";
    modalOverlay.style.top = "0";
    modalOverlay.style.left = "0";
    modalOverlay.style.width = "100%";
    modalOverlay.style.height = "100%";
    modalOverlay.style.background = "rgba(0,0,0,0.5)";
    modalOverlay.style.display = "flex";
    modalOverlay.style.justifyContent = "center";
    modalOverlay.style.alignItems = "center";
    modalOverlay.style.zIndex = "9999";

    const modal = document.createElement("div");
    modal.style.background = "#fff";
    modal.style.padding = "30px";
    modal.style.borderRadius = "8px";
    modal.style.maxWidth = "450px";
    modal.style.width = "90%";
    modal.style.boxShadow = "0 5px 25px rgba(0,0,0,0.2)";
    modal.style.textAlign = "center";

    modal.innerHTML = `
      <div style="font-size:3rem; color:#4caf50; margin-bottom:15px;">✔️</div>
      <h3 style="margin:0 0 10px 0; color:#1b5e20; font-weight:bold;">Documento Firmado Digitalmente</h3>
      <p style="font-size:0.9rem; color:#555; line-height:1.4; margin-bottom:20px;">
        El Certificado de Retenciones IRPF del ejercicio fiscal 2025 ha sido generado correctamente con firma electrónica válida y sello de tiempo oficial.
      </p>

      <div style="background:#f9f9f9; border:1px dashed #ccc; padding:12px; border-radius:6px; font-family:monospace; font-size:0.75rem; color:#666; margin-bottom:20px; text-align:left;">
        <div><strong>Emisor:</strong> FNMT-RCM - Diputación de Granada</div>
        <div><strong>Fecha firma:</strong> ${new Date().toLocaleString()}</div>
        <div><strong>CSV:</strong> CSV-CERT-10T-2025-9988-81A2</div>
      </div>

      <div style="display:flex; justify-content:center; margin-bottom:20px;">
        <div style="display:flex; gap:2px; height:40px; background:#000; padding:5px 15px; border-radius:4px; align-items:stretch;">
          <div style="width:2px; background:#fff;"></div>
          <div style="width:4px; background:#fff;"></div>
          <div style="width:1px; background:#fff;"></div>
          <div style="width:3px; background:#fff;"></div>
          <div style="width:1px; background:#fff;"></div>
          <div style="width:4px; background:#fff;"></div>
          <div style="width:2px; background:#fff;"></div>
        </div>
      </div>

      <div style="display:flex; gap:10px; justify-content:center;">
        <button type="button" id="btn-modal-download" style="background:#1b5e20; color:#fff; border:none; padding:8px 16px; border-radius:4px; cursor:pointer; font-weight:bold;">Descargar PDF</button>
        <button type="button" id="btn-modal-close" style="background:#f5f5f5; color:#333; border:1px solid #ccc; padding:8px 16px; border-radius:4px; cursor:pointer;">Cerrar</button>
      </div>
    `;

    modal.querySelector("#btn-modal-close").addEventListener("click", () => {
      modalOverlay.remove();
    });
    modal.querySelector("#btn-modal-download").addEventListener("click", () => {
      alert("Descarga de PDF de certificado iniciada.");
      modalOverlay.remove();
    });

    modalOverlay.append(modal);
    document.body.append(modalOverlay);
  }
}

renderFilters();
setupInteractions();
window.addEventListener("hashchange", () => {
  const hashState = hashStateFromLocation();
  const moduleID = hashState.moduleID || "dashboard";
  if (moduleID !== state.activeModule || hashState.screenID !== state.activeScreen) {
    setActiveModule(moduleID, hashState.screenID);
  }
});
reloadButton.addEventListener("click", loadPortal);
loadPortal();
