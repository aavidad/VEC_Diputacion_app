const VEC_SHELL_API = "/api/vec";
const VEC_WORKSPACE_API = "/api/vec/workspace";
const VEC_SESSION_API = "/api/vec/session";
const CRONOS_LEAVE_API = "/api/vec/cronos/leave-requests";
const PERSONAL_RPT_POSITIONS_API = "/api/vec/personal/rpt/positions";
const PERSONAL_RPT_STATS_API = "/api/vec/personal/rpt/stats";
const PERSONAL_CATEGORIES_API = "/api/vec/personal/categories";
const PERSONAL_CATALOGS_API = "/api/vec/personal/catalogs";
const DIETAS_ROAD_ROUTE_API = "/api/vec/dietas/road-route";
const DIETAS_SHEETS_STORAGE_KEY = "vec_demo_dietas_sheets";
const BOLSA_PORTAL_API = "/api/portal";
const ADMIN_STATUS_API = "/api/admin/status";
const ADMIN_CAPABILITIES_API = "/api/admin/capabilities";
const STAFF_HEADERS = {
  "Content-Type": "application/json",
  "Authorization": "Bearer empleado-token",
  "X-Auth-Token": "empleado-token",
  "X-Auth-Mechanism": "clave",
  "X-Auth-Subject": "demo.empleado",
  "X-Auth-Display-Name": "Empleado demo",
  "X-Auth-Roles": "personal_interno",
  "X-VEC-Auth-Mechanism": "clave",
  "X-VEC-Subject": "demo.empleado",
  "X-VEC-Roles": "personal_interno",
};

const DEMO_USERS = [
  {
    id: "empleado",
    label: "Empleado",
    displayName: "Empleado demo",
    subject: "demo.empleado",
    roles: ["personal_interno"],
    auth: "clave",
    defaultModule: "personal",
  },
  {
    id: "administrativo",
    label: "Administrativo",
    displayName: "Administrativo unidad",
    subject: "demo.administrativo",
    roles: ["administrativo"],
    auth: "clave",
    defaultModule: "dietas",
  },
  {
    id: "jefe_seccion",
    label: "Jefe seccion",
    displayName: "Jefatura de Seccion",
    subject: "demo.seccion",
    roles: ["jefe_seccion"],
    auth: "clave",
    defaultModule: "aprobaciones",
  },
  {
    id: "jefe_servicio",
    label: "Jefe servicio",
    displayName: "Jefatura de Servicio",
    subject: "demo.servicio",
    roles: ["jefe_servicio"],
    auth: "clave",
    defaultModule: "aprobaciones",
  },
  {
    id: "tecnico_rrhh",
    label: "Tecnico RRHH",
    displayName: "Tecnico RRHH",
    subject: "demo.rrhh.tecnico",
    roles: ["tecnico_rrhh"],
    auth: "dnie",
    defaultModule: "personal",
  },
  {
    id: "administrador",
    label: "Administrador",
    displayName: "Administrador VEC",
    subject: "demo.admin",
    roles: ["administrador"],
    auth: "dnie",
    defaultModule: "dashboard",
  },
];

let activeDemoUserID = "empleado";

function staffHeaders() {
  return { ...STAFF_HEADERS };
}

function activeDemoUser() {
  return DEMO_USERS.find((user) => user.id === activeDemoUserID) || DEMO_USERS[0];
}

function applyDemoUser(userID, options = {}) {
  const user = DEMO_USERS.find((candidate) => candidate.id === userID) || DEMO_USERS[0];
  const previousID = activeDemoUserID;
  activeDemoUserID = user.id;
  const roles = user.roles.join(",");
  STAFF_HEADERS.Authorization = `Bearer ${user.id}-token`;
  STAFF_HEADERS["X-Auth-Token"] = `${user.id}-token`;
  STAFF_HEADERS["X-Auth-Mechanism"] = user.auth;
  STAFF_HEADERS["X-Auth-Subject"] = user.subject;
  STAFF_HEADERS["X-Auth-Display-Name"] = user.displayName;
  STAFF_HEADERS["X-Auth-Roles"] = roles;
  STAFF_HEADERS["X-VEC-Auth-Mechanism"] = user.auth;
  STAFF_HEADERS["X-VEC-Subject"] = user.subject;
  STAFF_HEADERS["X-VEC-Roles"] = roles;
  if (previousID !== user.id) {
    resetRoleScopedViewState();
  }
  updateDemoUserUI();
  if (options.switchModule !== false) {
    state.activeModule = moduleIDForSession(user.defaultModule);
    state.activeScreen = "";
    clearLocationHash();
  } else if (!moduleVisibleForSession(state.activeModule)) {
    state.activeModule = defaultModuleID();
    state.activeScreen = "";
  }
  if (options.reload && state.portal) {
    return loadPortal();
  }
  return Promise.resolve();
}

function updateDemoUserUI() {
  const user = activeDemoUser();
  if (document.body) {
    document.body.dataset.accessProfile = sessionAccessProfile().id;
  }
  const select = $("#demo-user-select");
  if (select && select.value !== user.id) {
    select.value = user.id;
  }
  const label = $("#demo-user-role-label");
  if (label) {
    label.textContent = `${user.displayName} - ${sessionAccessProfile().label}`;
  }
  updateTopbarContext();
}

function sessionContextText() {
  const user = activeDemoUser();
  return `Sesion demo: ${user.displayName} (${user.roles.join(", ")})`;
}

function updateTopbarContext() {
  const copy = MODULE_COPY[state.activeModule] || MODULE_COPY.dashboard;
  const eyebrow = $(".context-block .eyebrow");
  const title = $(".context-block h2");
  const context = $(".context-block .small-text");
  if (eyebrow) eyebrow.textContent = copy[0];
  if (title) title.textContent = copy[1];
  if (context) {
    const lead = state.activeModule === "dashboard"
      ? "Vista interna para controlar horarios, fichajes, permisos, vacaciones, dietas, kilometraje provincial y expedientes desde un unico portal."
      : moduleLeadText(state.activeModule);
    context.textContent = `${sessionContextText()} · ${lead}`;
  }
}

function resetRoleScopedViewState() {
  state.selectedRowID = "";
  state.search = "";
  state.screenStateFilter = "";
  state.filters = { scope: "", state: "", deadline: "", unit: "" };
  const search = $("#global-search");
  if (search) search.value = "";
  $$(".filter-bar select").forEach((select) => { select.value = ""; });
}

function clearLocationHash() {
  if (!window.location.hash) return;
  history.replaceState(null, "", `${window.location.pathname}${window.location.search}`);
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

const MODULE_ACTION_ENDPOINT = {
  personal: "personal",
  nominas: "nominas",
  cronos: "cronos",
  horarios: "horarios",
  permisos: "permisos",
  dietas: "dietas",
  rutas: "rutas",
  bolsa: "bolsa",
  administracion: "administracion",
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

const MODULE_PARENT = {
  horarios: "cronos",
  permisos: "cronos",
  rutas: "dietas",
};

const MODULE_DEFAULT_SCREEN = {
  horarios: "horarios.perfiles",
  permisos: "permisos.solicitudes",
  rutas: "rutas.kilometraje",
};

const ROOT_MENU_GROUPS = [
  ["Portal empleado", ["dashboard", "personal", "nominas", "cronos", "dietas", "bolsa"]],
  ["Expediente y control", ["documentos", "aprobaciones", "auditoria", "administracion"]],
];

const ROLE_ACCESS_PROFILES = [
  {
    id: "administrador",
    label: "Perfil administrador",
    roles: ["administrador", "admin", "vec_admin"],
    modules: ["dashboard", "personal", "nominas", "cronos", "dietas", "bolsa", "documentos", "aprobaciones", "auditoria", "administracion"],
  },
  {
    id: "tecnico_rrhh",
    label: "Perfil RRHH",
    roles: ["tecnico_rrhh", "rrhh", "personal_rrhh"],
    modules: ["personal", "nominas", "cronos", "dietas", "bolsa", "documentos", "aprobaciones", "auditoria", "administracion"],
  },
  {
    id: "jefatura",
    label: "Perfil jefatura",
    roles: ["jefe_seccion", "jefe_servicio", "responsable", "responsable_centro"],
    modules: ["cronos", "dietas", "documentos", "aprobaciones", "auditoria"],
  },
  {
    id: "administrativo",
    label: "Perfil administrativo",
    roles: ["administrativo", "administrativo_unidad"],
    modules: ["personal", "nominas", "cronos", "dietas", "bolsa", "documentos", "aprobaciones"],
  },
  {
    id: "empleado",
    label: "Perfil empleado",
    roles: ["personal_interno", "empleado", "employee"],
    modules: ["personal", "nominas", "cronos", "dietas", "bolsa", "documentos", "notificaciones"],
  },
];

function currentRoleList() {
  return [
    STAFF_HEADERS["X-Auth-Roles"],
    STAFF_HEADERS["X-VEC-Roles"],
  ]
    .join(",")
    .split(/[,\s]+/g)
    .map((role) => role.trim().toLowerCase())
    .filter(Boolean);
}

function isAdminSession() {
  return sessionAccessProfile().id === "administrador";
}

function isEmployeeSelfServiceSession() {
  return sessionAccessProfile().id === "empleado";
}

function sessionAccessProfile() {
  const roles = currentRoleList();
  return ROLE_ACCESS_PROFILES.find((profile) => profile.roles.some((role) => roles.includes(role))) || ROLE_ACCESS_PROFILES[ROLE_ACCESS_PROFILES.length - 1];
}

function sessionModuleIDs() {
  return new Set(sessionAccessProfile().modules);
}

function defaultModuleID() {
  if (isAdminSession()) return "dashboard";
  const preferred = activeDemoUser()?.defaultModule || "dietas";
  if (moduleVisibleForSession(preferred)) return preferred;
  return sessionAccessProfile().modules.find((moduleID) => moduleID !== "dashboard") || "dietas";
}

function moduleVisibleForSession(moduleID) {
  const normalized = MODULE_PARENT[moduleID] || moduleID;
  if (normalized === "dashboard") return isAdminSession();
  return sessionModuleIDs().has(normalized);
}

function rowVisibleForSession(row) {
  if (isAdminSession()) return true;
  const modules = Array.isArray(row?.modules) ? row.modules : [];
  if (isEmployeeSelfServiceSession()) {
    const ownText = [
      row?.candidate,
      row?.candidate_id,
      row?.employee,
      row?.name,
      row?.subject,
      row?.expediente,
      row?.scope,
      row?.document,
      row?.action,
      ...(Array.isArray(row?.documents) ? row.documents : []),
    ].join(" ").toLowerCase();
    return /alberto|sanchez|74839201a|demo\.empleado|personal-interno|csv-9382|csv-cert/.test(ownText);
  }
  if (!modules.length) return true;
  return modules.some((moduleID) => moduleID !== "dashboard" && moduleVisibleForSession(moduleID));
}

function visibleMenuGroups() {
  return ROOT_MENU_GROUPS
    .map(([title, ids]) => [title, ids.filter(moduleVisibleForSession)])
    .filter(([, ids]) => ids.length);
}

function moduleIDForSession(moduleID) {
  const candidate = moduleID || defaultModuleID();
  return moduleVisibleForSession(candidate) ? candidate : defaultModuleID();
}

const MODULE_COPY = {
  dashboard: ["Bandeja VEC unificada", "Fichajes, permisos, dietas y expedientes en una cola comun"],
  personal: ["Modulo Personal", "Empleado, puesto, situacion administrativa, antiguedad y certificados"],
  nominas: ["Nominas y retribuciones", "Incidencias retributivas, trienios, reducciones y cierre mensual"],
  cronos: ["Modulo Cronos", "Fichajes, horarios, turnos, permisos, asuntos propios y vacaciones"],
  horarios: ["Horarios del personal", "Flexibilidad, turnos fijos, cobertura obligatoria y reducciones 63/64"],
  permisos: ["Permisos y vacaciones", "Saldos, solapes, ausencias y aprobaciones"],
  dietas: ["Modulo Dietas", "Comisiones de servicio, kilometraje provincial, gastos, medias dietas y dietas completas"],
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
  activeModule: defaultModuleID(),
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

function formatCurrency(value) {
  return new Intl.NumberFormat("es-ES", {
    style: "currency",
    currency: "EUR",
  }).format(moneyNumber(value));
}

function moneyNumber(value) {
  if (typeof value === "number") return Number.isFinite(value) ? value : 0;
  if (value == null || value === "") return 0;
  const normalized = String(value)
    .replace(/\s/g, "")
    .replace(/[^\d,.-]/g, "")
    .replace(/\.(?=\d{3}(?:\D|$))/g, "")
    .replace(",", ".");
  const parsed = Number(normalized);
  return Number.isFinite(parsed) ? parsed : 0;
}

function formatCount(value) {
  return new Intl.NumberFormat("es-ES").format(Number(value || 0));
}

function todayISODate() {
  return new Date().toISOString().slice(0, 10);
}

function formatDateForDisplay(value) {
  if (!value) return "-";
  const parts = String(value).split("-");
  if (parts.length === 3) return `${parts[2]}/${parts[1]}/${parts[0]}`;
  return String(value);
}

function readStoredArray(key) {
  try {
    const parsed = JSON.parse(window.localStorage?.getItem(key) || "[]");
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

function writeStoredArray(key, value) {
  try {
    window.localStorage?.setItem(key, JSON.stringify(Array.isArray(value) ? value : []));
  } catch {
    // La demo sigue funcionando en memoria si el navegador bloquea localStorage.
  }
}

function defaultEmployeeDietasSheets() {
  return [
    { id: "DIET-2026-0084", fecha: "19/06/2026", travelDate: "2026-06-19", ruta: "Granada - Albolote - Granada", estado: "Aprobada", importe: 28.40, km: 21.6, mileage_amount: 5.62 },
    { id: "DIET-2026-0091", fecha: "21/06/2026", travelDate: "2026-06-21", ruta: "Granada - Motril - Granada", estado: "Borrador", importe: 0, km: 0, mileage_amount: 0 },
    { id: "DIET-2026-0073", fecha: "27/05/2026", travelDate: "2026-05-27", ruta: "Granada - Guadix - Granada", estado: "Liquidada", importe: 61.88, km: 107.2, mileage_amount: 27.87 },
    { id: "DIET-2026-0061", fecha: "18/04/2026", travelDate: "2026-04-18", ruta: "Granada - Loja - Granada", estado: "Aprobada", importe: 48.54, km: 112.6, mileage_amount: 29.28 },
    { id: "DIET-2026-0048", fecha: "12/03/2026", travelDate: "2026-03-12", ruta: "Granada - Baza - Granada", estado: "Liquidada", importe: 92.36, km: 216.8, mileage_amount: 56.37 },
  ];
}

function ensureDietasSheets() {
  if (Array.isArray(state.dietasSheets)) return state.dietasSheets;
  const stored = readStoredArray(DIETAS_SHEETS_STORAGE_KEY);
  state.dietasSheets = stored.length ? stored : defaultEmployeeDietasSheets();
  if (!stored.length) writeStoredArray(DIETAS_SHEETS_STORAGE_KEY, state.dietasSheets);
  return state.dietasSheets;
}

function saveDietasSheets(items = state.dietasSheets) {
  state.dietasSheets = Array.isArray(items) ? items : [];
  writeStoredArray(DIETAS_SHEETS_STORAGE_KEY, state.dietasSheets);
  return state.dietasSheets;
}

function isoDateFromDietasValue(value) {
  const raw = String(value || "").trim();
  if (!raw) return "";
  if (/^\d{4}-\d{2}-\d{2}$/.test(raw)) return raw;
  const displayMatch = raw.match(/^(\d{1,2})\/(\d{1,2})\/(\d{4})$/);
  if (displayMatch) {
    return `${displayMatch[3]}-${displayMatch[2].padStart(2, "0")}-${displayMatch[1].padStart(2, "0")}`;
  }
  const parsed = new Date(raw);
  if (!Number.isNaN(parsed.getTime())) return parsed.toISOString().slice(0, 10);
  return "";
}

function dietasMonthKeyFromDate(value) {
  const iso = isoDateFromDietasValue(value);
  return iso ? iso.slice(0, 7) : todayISODate().slice(0, 7);
}

function dietasMonthKeyForRecord(record) {
  return dietasMonthKeyFromDate(record?.travelDate || record?.dateISO || record?.fecha || record?.date);
}

function dietasMonthLabel(key) {
  const [year, month] = String(key || todayISODate().slice(0, 7)).split("-").map(Number);
  const date = new Date(year || new Date().getFullYear(), (month || 1) - 1, 1);
  const label = date.toLocaleDateString("es-ES", { month: "long", year: "numeric" });
  return label.charAt(0).toUpperCase() + label.slice(1);
}

function aggregateDietasByMonth(records) {
  const map = new Map();
  (records || []).forEach((record) => {
    const key = dietasMonthKeyForRecord(record);
    const current = map.get(key) || { key, count: 0, km: 0, mileage: 0, allowances: 0, total: 0, pending: 0 };
    const total = moneyNumber(record.importe ?? record.amount ?? record.total);
    const mileage = moneyNumber(record.mileage_amount ?? record.mileageAmount);
    current.count += 1;
    current.km += moneyNumber(record.km);
    current.mileage += mileage;
    current.allowances += Math.max(0, total - mileage);
    current.total += total;
    if (/pendiente|borrador|validacion|revisar/i.test(String(record.estado || record.state || ""))) current.pending += 1;
    map.set(key, current);
  });
  return map;
}

function previousDietasMonthKeys(currentKey, records) {
  const keys = new Set();
  const [currentYear, currentMonth] = String(currentKey).split("-").map(Number);
  if (currentYear && currentMonth) {
    for (let month = 1; month < currentMonth; month += 1) {
      keys.add(`${currentYear}-${String(month).padStart(2, "0")}`);
    }
  }
  (records || []).forEach((record) => {
    const key = dietasMonthKeyForRecord(record);
    if (key < currentKey) keys.add(key);
  });
  return Array.from(keys).sort().reverse();
}

function updateEmployeeDietasMonthlyPanel(panel, selectedDate, draftTotals = {}) {
  const target = $("[data-dietas-month-panel]", panel);
  if (!target) return;
  const records = ensureDietasSheets();
  const monthly = aggregateDietasByMonth(records);
  const currentKey = dietasMonthKeyFromDate(selectedDate || todayISODate());
  const base = monthly.get(currentKey) || { key: currentKey, count: 0, km: 0, mileage: 0, allowances: 0, total: 0, pending: 0 };
  const draftTotal = moneyNumber(draftTotals.total);
  const draftKM = moneyNumber(draftTotals.km);
  const currentTotal = base.total + draftTotal;
  const historyRows = previousDietasMonthKeys(currentKey, records).map((key) => monthly.get(key) || {
    key,
    count: 0,
    km: 0,
    mileage: 0,
    allowances: 0,
    total: 0,
    pending: 0,
  });
  target.innerHTML = `
    <div class="employee-dietas-month-total">
      <span>Total del mes</span>
      <strong>${formatCurrency(currentTotal)}</strong>
      <small>${escapeHTML(dietasMonthLabel(currentKey))} · ${formatPoints(base.km + draftKM)} km · ${formatCount(base.count)} expedientes${draftTotal ? " + solicitud actual" : ""}</small>
    </div>
    <div class="employee-dietas-month-breakdown">
      <span><b>Registrado</b>${formatCurrency(base.total)}</span>
      <span><b>Solicitud actual</b>${formatCurrency(draftTotal)}</span>
      <span><b>Pendientes</b>${formatCount(base.pending)}</span>
    </div>
    <div class="employee-dietas-history">
      <div class="employee-dietas-history-head">
        <strong>Historico meses pasados</strong>
        <span>${formatCount(historyRows.length)} meses</span>
      </div>
      <div class="employee-dietas-history-list">
        ${historyRows.map((row) => `
          <article>
            <span>${escapeHTML(dietasMonthLabel(row.key))}</span>
            <b>${formatCurrency(row.total)}</b>
            <small>${formatCount(row.count)} solicitudes · ${formatPoints(row.km)} km</small>
          </article>
        `).join("")}
      </div>
    </div>
  `;
}

async function getData(url, options) {
  const response = await fetch(url, options);
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) {
    if (response.status === 401) {
      throw new Error(`Autenticacion requerida en ${url}. En desarrollo se usa identidad demo de personal interno; recarga la pagina o revisa certificado/DNIe cuando se active el modo real.`);
    }
    throw new Error(payload.error || payload.message || `No se pudo cargar ${url}.`);
  }
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
  const metrics = $(".metrics");
  if (metrics && (!isAdminSession() || state.activeModule !== "dashboard")) {
    metrics.hidden = true;
    return;
  }
  if (metrics) metrics.hidden = false;
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
  const routes = view.workspace?.province_route_pairs || view.workspace?.province_routes || [];
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
        { label: "Solicitudes", value: moduleRows.length, note: "Comisiones y gastos" },
        { label: "Pendientes validar", value: moduleRows.filter((row) => /pendiente|validar|aprobar|revisar/i.test(`${row.state} ${row.action}`)).length, note: "Jefatura/RRHH" },
        { label: "Con justificante", value: moduleRows.filter((row) => /justificante|factura|csv/i.test(`${row.document} ${row.action}`)).length, note: "Alojamiento o gasto" },
        { label: "Liquidaciones", value: moduleRows.filter((row) => /Liquidar|liquidacion/i.test(`${row.action} ${row.document}`)).length, note: "Listas para cierre" },
      ];
    case "rutas":
      return [
        { label: "Tramos usados", value: routes.length, note: "En solicitudes" },
        { label: "Rutas alternativas", value: moduleRows.filter((row) => /ruta|km/i.test(`${row.state} ${row.action}`)).length, note: "Con motivo" },
        { label: "Km a validar", value: moduleRows.filter((row) => /km|kilometraje/i.test(`${row.document} ${row.action}`)).length, note: "Pendientes" },
        { label: "Liquidables", value: moduleRows.filter((row) => /Aprobado|Liquidar/i.test(`${row.state} ${row.action}`)).length, note: "Tras validacion" },
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
  if (isEmployeeSelfServiceSession()) {
    const ownCounts = {
      personal: 3,
      nominas: 3,
      cronos: 3,
      dietas: 2,
      bolsa: 2,
      documentos: 3,
      notificaciones: 1,
    };
    return ownCounts[moduleID] || 0;
  }
  const rows = state.rows.length ? state.rows : rowsFromPortal(view);
  switch (moduleID) {
    case "dashboard":
      return rows.length;
    case "personal":
      return rows.filter((row) => row.modules.includes("personal")).length || getPersonalCatalog(view).stats?.positions || 0;
    case "nominas":
      return rows.filter((row) => row.modules.includes("nominas")).length;
    case "cronos":
      return rows.filter((row) =>
        row.modules.includes("cronos") || row.modules.includes("horarios") || row.modules.includes("permisos"),
      ).length;
    case "horarios":
      return rows.filter((row) => row.modules.includes("horarios")).length || view.workspace?.schedule_profiles?.length || 0;
    case "permisos":
      return rows.filter((row) => row.modules.includes("permisos")).length;
    case "dietas":
      return rows.filter((row) => row.modules.includes("dietas") || row.modules.includes("rutas")).length;
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
  const nav = $(".module-group");
  if (!nav) return;
  const moduleByID = new Map(MODULES.map((module) => [module.id, module]));
  const nodes = [];
  visibleMenuGroups().forEach(([title, ids]) => {
    const label = document.createElement("p");
    label.className = "module-title";
    label.textContent = title;
    nodes.push(label);
    ids.forEach((id) => {
      const module = moduleByID.get(id);
      if (!module) return;
      const button = document.createElement("button");
      button.className = "module-link";
      button.type = "button";
      button.dataset.accent = module.accent || "slate";
      button.dataset.module = module.id;
      button.setAttribute("aria-current", state.activeModule === module.id ? "page" : "false");
      button.innerHTML = `<span></span><span class="module-count"></span>`;
      $("span:first-child", button).textContent = module.label;
      $(".module-count", button).textContent = formatCount(moduleCount(view, module.id));
      button.addEventListener("click", () => setActiveModule(module.id));
      nodes.push(button);
    });
  });
  nav.replaceChildren(...nodes);
}

function updateModuleButton(button, module, view) {
  if (!button || !module) return;
    const name = $("span:first-child", button);
    const count = $(".module-count", button);
    name.textContent = module.label;
    count.textContent = formatCount(moduleCount(view, module.id));
    button.dataset.accent = module.accent || "slate";
    button.dataset.module = module.id;
    button.setAttribute("aria-current", state.activeModule === module.id ? "page" : "false");
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
  const routes = view.workspace?.province_route_pairs || view.workspace?.province_routes || [];
  if (!routes.length) {
    target.replaceChildren(Object.assign(document.createElement("p"), {
      className: "empty-state",
      textContent: "Sin solicitudes de dietas cargadas.",
    }));
    return;
  }
  const routeList = document.createElement("div");
  routeList.className = "route-list";
  const summary = document.createElement("article");
  summary.className = "route-row";
  summary.innerHTML = `
      <div class="route-pin"></div>
      <div><strong></strong><span></span></div>
      <b></b>`;
  $(".route-pin", summary).textContent = "D";
  $("strong", summary).textContent = "Nueva solicitud diaria";
  $("span", summary).textContent = "Dia de viaje, rutas, manutencion, alojamiento y validacion";
  $("b", summary).textContent = "Borrador";
  const routeRows = routes.slice(0, 6).map((route, index) => {
    const row = document.createElement("article");
    row.className = "route-row";
    row.innerHTML = `
      <div class="route-pin"></div>
      <div><strong></strong><span></span></div>
      <b></b>`;
    $(".route-pin", row).textContent = index + 1;
    $("strong", row).textContent = `${route.from} -> ${route.to}`;
    $("span", row).textContent = `${route.duration_minutes ?? route.estimated_minutes} min - ${route.allowance}`;
    $("b", row).textContent = `${formatPoints(route.distance_km ?? route.km_one_way)} km`;
    return row;
  });
  routeList.replaceChildren(summary, ...routeRows);
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
  return Array.from(byID.values()).filter((definition) =>
    moduleVisibleForSession(definition.module_key) && (isAdminSession() || !String(definition.id || "").endsWith(".dashboard")),
  );
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
  if (MODULE_PARENT[candidate]) return MODULE_PARENT[candidate];
  if (MODULE_PARENT[inferred]) return MODULE_PARENT[inferred];
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

const EMPLOYEE_SELF_SERVICE_MODULES = new Set(["personal", "cronos", "horarios", "permisos", "dietas", "rutas", "bolsa", "documentos", "notificaciones"]);

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

  if (state.activeModule === "nominas") {
    target.dataset.mode = "custom-nominas";
    renderCustomNominasApp(target, view);
    return;
  }

  const selfModuleID = MODULE_PARENT[state.activeModule] || state.activeModule;
  if (isEmployeeSelfServiceSession() && EMPLOYEE_SELF_SERVICE_MODULES.has(state.activeModule)) {
    target.dataset.mode = "employee-self-service";
    renderEmployeeSelfServiceModule(target, view, selfModuleID);
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

function renderEmployeeSelfServiceModule(target, view, moduleID) {
  const employee = payrollEmployeeData();
  const summary = view.workspace?.cronos_daily_summary || {};
  const permissions = view.workspace?.cronos_permission_balances || [];
  const findPermission = (name) => permissions.find((item) => String(item.name || "").toUpperCase().includes(name)) || {};
  const ownLeaveRows = [
    ...(state.cronosSubmittedRequests || []).map((item) => [
      item.id || item.ref || "PER-NUEVO",
      item.tipo || item.type || item.concepto || "Permiso",
      `${item.desde || "-"} - ${item.hasta || "-"}`,
      item.estado || item.state || "Registrada",
      "Ver solicitud",
    ]),
    ["PER-2026-0142", "Asuntos propios", "24/06/2026", "Pendiente responsable", "Ver solicitud"],
    ["PER-2026-0108", "Consulta medica", "18/06/2026", "Justificada", "Ver justificante"],
    ["VAC-2026-0031", "Vacaciones", "05/08/2026 - 18/08/2026", "Aprobada", "Ver resolucion"],
  ];
  const ownDietRows = ensureDietasSheets().map((item) => [
    item.id || item.ref || "DIET-PROP",
    item.fecha || item.date || "-",
    item.ruta || item.route || item.motivo || "Comision de servicio",
    item.estado || item.state || "Borrador",
    formatEuroCompact(item.importe || item.amount || item.total || 0),
    /borrador/i.test(item.estado || item.state || "") ? "Completar" : "Ver liquidacion",
  ]);
  if (moduleID === "personal") {
    target.append(modulePortalHeader("Mis datos personales", "Consulta de tus datos propios de personal y expedientes asociados.", []));
    const eligibleOffers = renderEmployeeEligibleBolsaPanel(view, { compact: true });
    if (eligibleOffers) target.append(eligibleOffers);
    target.append(portalGrid([
      ["Identidad", `${employee.name} · NIF ${employee.nif}`],
      ["Puesto", `${employee.position} · Transformacion digital`],
      ["Situacion administrativa", `${employee.relationship} · servicio activo`],
      ["Antiguedad", `${String(employee.trienios).padStart(2, "0")} trienios reconocidos`],
      ["Cuenta bancaria", employee.iban.replace(/\d(?=\d{4})/g, "*")],
      ["Afiliacion", employee.affiliation],
    ]));
    target.append(portalTable("Mis expedientes", ["Expediente", "Objeto", "Estado", "Accion"], [
      ["EMP-PROP-2026", "Ficha personal y puesto RPT", "Verificada", "Consultar"],
      ["CERT-SERV-2026", "Certificado de servicios prestados", "Disponible", "Solicitar certificado"],
      ["ANT-TRI-2026", "Antiguedad y trienios", "Calculado", "Ver detalle"],
    ], { actionColumn: true }));
    return;
  }

  if (moduleID === "cronos") {
    const asuntos = findPermission("ASUNTOS");
    const vacaciones = findPermission("VACACIONES");
    const conciliacion = findPermission("CONCILIACION");
    target.append(modulePortalHeader("Mi Cronos", "Consulta de tus fichajes, saldos, permisos y vacaciones.", []));
    target.append(portalGrid([
      ["Jornada de hoy", `Teoricas ${summary.theoretical || "07:30"} · trabajadas ${summary.worked || "00:00"}`],
      ["Saldo del periodo", `${summary.period_balance || "-04:34"} · desde ${summary.period_from || "01/06/2026"}`],
      ["Asuntos propios", `${asuntos.remaining || "5"} disponibles`],
      ["Vacaciones", `${vacaciones.remaining || "18"} dias disponibles`],
      ["Conciliacion", `${conciliacion.remaining || "12:00"} horas disponibles`],
      ["Mi horario", "Jornada personal asignada"],
    ]));
    target.append(renderEmployeeLeaveRequestPanel(view));
    target.append(portalTable("Mis solicitudes", ["Expediente", "Tipo", "Periodo", "Estado", "Accion"], ownLeaveRows, { actionColumn: true }));
    return;
  }

  if (moduleID === "dietas" || moduleID === "rutas") {
    target.append(modulePortalHeader("Mis dietas y comisiones", "Consulta y tramitacion de tus propias comisiones de servicio.", []));
    target.append(portalGrid([
      ["Borradores", String(ownDietRows.filter((row) => /borrador/i.test(row[3])).length)],
      ["Pendientes", String(ownDietRows.filter((row) => /pendiente|validar/i.test(row[3])).length)],
      ["Aprobadas", String(ownDietRows.filter((row) => /aprob/i.test(row[3])).length)],
      ["Liquidacion", "Solo tus importes y justificantes"],
    ]));
    target.append(renderEmployeeDietasRequestPanel(view));
    target.append(portalTable("Mis comisiones de servicio", ["Expediente", "Fecha", "Ruta / motivo", "Estado", "Importe", "Accion"], ownDietRows, { actionColumn: true }));
    return;
  }

  if (moduleID === "bolsa") {
    renderEmployeeBolsaModule(target, view);
    return;
  }

  if (moduleID === "documentos") {
    target.append(modulePortalHeader("Mis documentos", "Documentos, justificantes, certificados y CSV vinculados a tus expedientes.", []));
    target.append(portalGrid([
      ["Certificados", "2 disponibles"],
      ["Justificantes", "3 aportados"],
      ["Firmas", "CSV verificable en documentos emitidos"],
      ["Pendientes", "0 requerimientos abiertos"],
    ]));
    target.append(portalTable("Mis documentos", ["Documento", "Expediente", "Estado", "CSV", "Accion"], [
      ["Certificado retenciones 10T", "NOM-2025", "Firmado", "CSV-CERT-10T-2025-9988-81A2", "Descargar"],
      ["Recibo salarios junio", "NOM-2026-06", "Firmado", "CSV-9382-AJ84-29E1-401C", "Ver recibo"],
      ["Justificante medico", "PER-2026-0108", "Validado", "CSV-JUS-MED-0108", "Ver justificante"],
    ], { actionColumn: true }));
    return;
  }

  if (moduleID === "notificaciones") {
    target.append(modulePortalHeader("Mis notificaciones", "Avisos y tareas pendientes asociados a tus expedientes.", []));
    target.append(portalGrid([
      ["Pendientes de lectura", "1"],
      ["Firmas pendientes", "0"],
      ["Requerimientos", "0"],
      ["Ultima actualizacion", new Date().toLocaleDateString("es-ES")],
    ]));
    target.append(portalTable("Mis avisos", ["Fecha", "Asunto", "Expediente", "Estado", "Accion"], [
      ["20/06/2026", "Recibo de nomina disponible", "NOM-2026-06", "No leida", "Abrir"],
      ["18/06/2026", "Permiso medico justificado", "PER-2026-0108", "Leida", "Ver detalle"],
      ["15/06/2026", "Certificado de servicios disponible", "CERT-SERV-2026", "Leida", "Ver certificado"],
    ], { actionColumn: true }));
    return;
  }

  target.append(modulePortalHeader("Mis datos VEC", "Consulta de tus datos y expedientes propios.", []));
  target.append(portalTable("Mis expedientes", ["Expediente", "Modulo", "Estado", "Accion"], [
    ["EMP-PROP-2026", "Personal", "Verificado", "Consultar"],
    ["NOM-2026-06", "Nominas", "Recibo publicado", "Descargar"],
  ], { actionColumn: true }));
}

function employeeLeavePolicies(view) {
  const policies = (view.workspace?.cronos_leave_policies || [])
    .filter((policy) => policy.request !== false)
    .map((policy) => ({
      id: policy.id || slugify(policy.name || "permiso"),
      name: policy.name || policy.label || "Permiso",
      remaining: policy.remaining || policy.annual_allowance || "",
      requiresDocument: policy.requires_document === true,
    }));
  if (policies.length) return policies;
  return [
    { id: "asuntos_propios", name: "Asuntos propios", remaining: "5 dias", requiresDocument: false },
    { id: "vacaciones", name: "Vacaciones", remaining: "18 dias", requiresDocument: false },
    { id: "consulta_medica", name: "Consulta medica", remaining: "Justificante si procede", requiresDocument: true },
    { id: "deber_inexcusable", name: "Deber inexcusable", remaining: "Segun justificante", requiresDocument: true },
    { id: "conciliacion", name: "Conciliacion", remaining: "12:00 horas", requiresDocument: false },
    { id: "formacion", name: "Formacion", remaining: "Segun convocatoria", requiresDocument: true },
    { id: "compensacion_horaria", name: "Compensacion horaria", remaining: "Saldo horario", requiresDocument: false },
  ];
}

function nextEmployeeFlowID(prefix, store) {
  const year = new Date().getFullYear();
  const serial = String((Array.isArray(store) ? store.length : 0) + 1001).padStart(4, "0");
  return `${prefix}-${year}-${serial}`;
}

function daysBetweenInclusive(from, to) {
  const parse = (value) => {
    const parts = String(value || "").split("-").map(Number);
    if (parts.length !== 3 || parts.some((part) => !Number.isFinite(part))) return NaN;
    return Date.UTC(parts[0], parts[1] - 1, parts[2]);
  };
  const start = parse(from);
  const end = parse(to || from);
  if (!Number.isFinite(start) || !Number.isFinite(end)) return 1;
  return Math.max(1, Math.round((end - start) / 86400000) + 1);
}

function renderEmployeeLeaveRequestPanel(view) {
  const policies = employeeLeavePolicies(view);
  const panel = document.createElement("section");
  panel.className = "employee-flow-panel employee-leave-panel";
  panel.innerHTML = `
    <div class="employee-flow-head">
      <div>
        <h3>Solicitar permiso, asuntos propios o vacaciones</h3>
        <span>La solicitud queda pendiente de responsable y continua hacia RRHH si procede.</span>
      </div>
    </div>
    <form class="employee-flow-form">
      <label>Tipo de solicitud
        <select name="policy_id" required></select>
      </label>
      <label>Desde
        <input name="from" type="date" value="${todayISODate()}" required>
      </label>
      <label>Hasta
        <input name="to" type="date" value="${todayISODate()}" required>
      </label>
      <label>Cantidad
        <input name="amount" type="number" min="1" step="1" value="1" required>
      </label>
      <label class="employee-field-wide">Motivo
        <input name="reason" value="Solicitud de permiso" required>
      </label>
      <label>Justificante
        <input name="document_ref" placeholder="CSV, factura o referencia si procede">
      </label>
      <div class="employee-flow-note" data-leave-note></div>
      <div class="employee-form-actions">
        <button type="submit" class="primary-action">Enviar solicitud</button>
      </div>
    </form>
  `;

  const form = $("form", panel);
  const select = $("select[name='policy_id']", form);
  policies.forEach((policy) => {
    const option = document.createElement("option");
    option.value = policy.id;
    option.textContent = policy.remaining ? `${policy.name} - saldo ${policy.remaining}` : policy.name;
    option.dataset.requiresDocument = policy.requiresDocument ? "true" : "false";
    select.append(option);
  });

  const amountInput = $("input[name='amount']", form);
  const docInput = $("input[name='document_ref']", form);
  const note = $("[data-leave-note]", form);
  const syncLeaveForm = () => {
    amountInput.value = String(daysBetweenInclusive($("input[name='from']", form).value, $("input[name='to']", form).value));
    const selected = policies.find((policy) => policy.id === select.value) || policies[0];
    const requiresDocument = selected?.requiresDocument || /medic|deber|formacion|justificante/i.test(selected?.name || "");
    docInput.required = requiresDocument;
    docInput.placeholder = requiresDocument ? "Justificante obligatorio" : "CSV, factura o referencia si procede";
    note.textContent = selected?.remaining
      ? `Saldo orientativo: ${selected.remaining}. La validacion definitiva la realiza la cadena de aprobacion.`
      : "La validacion definitiva la realiza la cadena de aprobacion.";
  };
  ["change", "input"].forEach((eventName) => {
    select.addEventListener(eventName, syncLeaveForm);
    $("input[name='from']", form).addEventListener(eventName, syncLeaveForm);
    $("input[name='to']", form).addEventListener(eventName, syncLeaveForm);
  });
  syncLeaveForm();

  form.addEventListener("submit", (event) => {
    event.preventDefault();
    if (!form.reportValidity()) return;
    const data = new FormData(form);
    const selected = policies.find((policy) => policy.id === data.get("policy_id")) || policies[0];
    if (!state.cronosSubmittedRequests) state.cronosSubmittedRequests = [];
    const prefix = selected.id === "vacaciones" ? "VAC" : "PER";
    const id = nextEmployeeFlowID(prefix, state.cronosSubmittedRequests);
    const request = {
      id,
      tipo: selected.name,
      concepto: selected.name,
      desde: formatDateForDisplay(data.get("from")),
      hasta: formatDateForDisplay(data.get("to")),
      duracion: `${data.get("amount")} ${Number(data.get("amount")) === 1 ? "dia" : "dias"}`,
      motivo: String(data.get("reason") || "").trim(),
      document_ref: String(data.get("document_ref") || "").trim(),
      estado: "Pendiente responsable",
      fechaSolicitud: new Date().toLocaleString("es-ES"),
      workflow: ["Solicitante", "Responsable", "RRHH"],
    };
    state.cronosSubmittedRequests.unshift(request);
    recordReceipt("Solicitud Cronos enviada", `${id} - ${request.tipo} - ${request.desde} a ${request.hasta}`, "cronos");
    setStatus(`Solicitud enviada: ${id}`, "ready");
    renderModulePortal(view);
  });

  return panel;
}

function renderEmployeeDietasRequestPanel(view) {
  const stopOptions = routeSelectableStops(view);
  const points = routeSelectableViaPoints(view.workspace?.province_route_points || []);
  const expensePolicy = view.workspace?.expense_policy || {};
  const allowanceTypes = normalizeAllowanceTypes(expensePolicy.allowance_types);
  const rate = Number(expensePolicy?.mileage?.rate_eur_km || 0.26);
  if (!Array.isArray(state.employeeDietasStops) || state.employeeDietasStops.length < 2) {
    state.employeeDietasStops = ["Granada", "Motril", "Granada"];
  }

  const panel = document.createElement("section");
  panel.className = "employee-flow-panel employee-dietas-panel";
  panel.innerHTML = `
    <div class="employee-flow-head">
      <div>
        <h3>Nueva dieta o comision de servicio</h3>
        <span>Registra el dia, la ruta, kilometraje, manutencion, alojamiento y gastos para validacion.</span>
      </div>
    </div>
    <div class="employee-dietas-month-panel" data-dietas-month-panel aria-live="polite"></div>
    <form class="employee-flow-form">
      <label>Dia del viaje
        <input name="travel_date" type="date" value="${todayISODate()}" required>
      </label>
      <label class="employee-field-wide">Motivo del desplazamiento
        <input name="purpose" value="Comision de servicio" required>
      </label>
      <fieldset class="employee-fieldset employee-field-wide">
        <legend>Ruta del dia</legend>
        <div class="employee-route-list"></div>
        <div class="employee-inline-actions">
          <button type="button" class="quiet-action" data-add-dietas-stop>Anadir parada</button>
          <button type="button" class="quiet-action" data-estimate-dietas-route>Calcular ruta</button>
        </div>
        <div class="employee-flow-note" data-route-note>Selecciona municipios del catalogo oficial. Puede ajustar los kilometros si el desplazamiento real incluye recorridos dentro del municipio.</div>
      </fieldset>
      <label class="employee-check">
        <input name="own_car" type="checkbox" checked>
        Vehiculo propio
      </label>
      <label>Km ruta
        <input name="km" type="number" min="0" step="0.1" value="0">
      </label>
      <label>Km compensacion
        <input name="compensation_km" type="number" min="0" step="0.1" value="0">
      </label>
      <label class="employee-field-wide">Motivo compensacion
        <input name="compensation_reason" placeholder="Vuelta dentro del municipio, desvio, corte de carretera...">
      </label>
      <label>Tarifa kilometro
        <input name="rate" type="number" min="0" step="0.01" value="${rate}">
      </label>
      <label>Manutencion
        <select name="allowance_type" required></select>
      </label>
      <label>Importe manutencion
        <input name="allowance_amount" type="number" min="0" step="0.01" value="0">
      </label>
      <label>Alojamiento
        <input name="lodging_amount" type="number" min="0" step="0.01" value="0">
      </label>
      <label class="employee-field-wide">Justificante alojamiento
        <input name="lodging_reference" placeholder="Factura/CSV si hay alojamiento">
      </label>
      <label>Otros gastos
        <input name="other_expenses" type="number" min="0" step="0.01" value="0">
      </label>
      <label class="employee-field-wide">Justificante otros gastos
        <input name="expense_reference" placeholder="Parking, peaje, transporte publico...">
      </label>
      <div class="employee-summary-strip employee-field-wide" aria-live="polite">
        <span>Km liquidables</span><b data-dietas-own-summary="km">0,0 km</b>
        <span>Kilometraje</span><b data-dietas-own-summary="mileage">0,00 €</b>
        <span>Dietas/gastos</span><b data-dietas-own-summary="allowances">0,00 €</b>
        <span>Total</span><b data-dietas-own-summary="total">0,00 €</b>
      </div>
      <div class="employee-form-actions employee-field-wide">
        <button type="submit" class="primary-action">Enviar a validacion</button>
      </div>
    </form>
  `;

  const form = $("form", panel);
  const mapPanel = routeMapPanel();
  mapPanel.classList.add("employee-route-map");
  const routeResult = document.createElement("div");
  routeResult.className = "route-result employee-route-result";
  $(".employee-fieldset", form).append(routeResult, mapPanel);

  const selectAllowance = $("select[name='allowance_type']", form);
  allowanceTypes.forEach((allowance) => {
    const option = document.createElement("option");
    option.value = allowance.id;
    option.textContent = `${allowance.label} (${formatCurrency(allowance.amount)})`;
    option.dataset.amount = String(allowance.amount || 0);
    selectAllowance.append(option);
  });
  const defaultAllowance = allowanceTypes.find((item) => item.id === "no_dieta") || allowanceTypes[0];
  selectAllowance.value = defaultAllowance?.id || "";
  $("input[name='allowance_amount']", form).value = String(defaultAllowance?.amount || 0);

  const readStops = () => $$(".employee-route-stop select", panel)
    .map((select) => select.value.trim())
    .filter(Boolean);
  const syncStopsState = () => {
    state.employeeDietasStops = $$(".employee-route-stop select", panel).map((select) => select.value.trim());
  };
  const stopList = $(".employee-route-list", panel);
  const renderStops = () => {
    stopList.replaceChildren(...state.employeeDietasStops.map((stop, index) => {
      const row = document.createElement("div");
      row.className = "employee-route-stop";
      const label = document.createElement("label");
      const labelText = index === 0 ? "Salida" : (index === state.employeeDietasStops.length - 1 ? "Destino final" : `Parada ${index}`);
      label.innerHTML = `<span></span>`;
      $("span", label).textContent = labelText;
      const select = document.createElement("select");
      select.required = true;
      const placeholder = document.createElement("option");
      placeholder.value = "";
      placeholder.textContent = "Seleccionar municipio";
      select.append(placeholder);
      stopOptions.forEach((point) => {
        const option = document.createElement("option");
        option.value = point.name;
        option.textContent = point.municipality_name && point.municipality_name !== point.name
          ? `${point.name} (${point.municipality_name})`
          : point.name;
        option.dataset.code = point.code || point.ine_code || "";
        select.append(option);
      });
      if (stopOptions.some((point) => point.name === stop)) select.value = stop;
      select.addEventListener("change", () => {
        state.employeeDietasStops[index] = select.value;
      });
      label.append(select);
      const remove = document.createElement("button");
      remove.type = "button";
      remove.className = "row-action is-quiet";
      remove.textContent = "Quitar";
      remove.disabled = state.employeeDietasStops.length <= 2;
      remove.addEventListener("click", () => {
        state.employeeDietasStops.splice(index, 1);
        renderStops();
        syncDietasTotal();
      });
      row.append(label, remove);
      return row;
    }));
  };

  const syncDietasTotal = () => {
    const baseKM = moneyNumber($("input[name='km']", form).value);
    const compensationKM = moneyNumber($("input[name='compensation_km']", form).value);
    const ownCar = $("input[name='own_car']", form).checked;
    const km = ownCar ? baseKM + compensationKM : 0;
    const mileage = km * moneyNumber($("input[name='rate']", form).value);
    const allowances = moneyNumber($("input[name='allowance_amount']", form).value)
      + moneyNumber($("input[name='lodging_amount']", form).value)
      + moneyNumber($("input[name='other_expenses']", form).value);
    const setSummary = (key, value) => {
      const node = $(`[data-dietas-own-summary="${key}"]`, panel);
      if (node) node.textContent = value;
    };
    setSummary("km", `${formatPoints(km)} km`);
    setSummary("mileage", formatCurrency(mileage));
    setSummary("allowances", formatCurrency(allowances));
    setSummary("total", formatCurrency(mileage + allowances));
    updateEmployeeDietasMonthlyPanel(panel, $("input[name='travel_date']", form).value, { km, mileage, allowances, total: mileage + allowances });
    return { km, mileage, allowances, total: mileage + allowances };
  };

  mapPanel._onRoadRouteCalculated = (updatedCalculation) => {
    $("input[name='km']", form).value = Number(updatedCalculation.totalBaseKM || 0).toFixed(1);
    renderItineraryResult(routeResult, updatedCalculation, null, []);
    const note = $("[data-route-note]", panel);
    if (note) {
      note.textContent = `Ruta calculada con OSRM interno: ${formatPoints(updatedCalculation.totalBaseKM)} km base. Puedes sumar compensacion justificada.`;
    }
    syncDietasTotal();
    setupRouteResultLegSelection(mapPanel);
    setStatus("Kilometraje calculado con OSRM interno", "ready");
  };

  const estimateRoute = () => {
    syncStopsState();
    const stops = readStops();
    const note = $("[data-route-note]", panel);
    if (stops.length < 2) {
      note.textContent = "Anade al menos salida y destino para calcular la ruta.";
      setStatus("Ruta incompleta para calcular kilometraje", "error");
      return null;
    }
    const calculation = calculateItinerary(stops, moneyNumber($("input[name='rate']", form).value), view, new Map());
    renderItineraryResult(routeResult, calculation, null, []);
    if (calculation.missing.length || calculation.totalKM <= 0) {
      note.textContent = `No hay matriz completa para: ${calculation.missing.join(", ") || stops.join(" - ")}. Indica los km liquidables manualmente.`;
      setStatus("Ruta pendiente de matriz interna; introduce km manuales", "warning");
      syncDietasTotal();
      renderRouteMap(mapPanel, calculation, view);
      return calculation;
    }
    $("input[name='km']", form).value = calculation.totalBaseKM.toFixed(1);
    note.textContent = `Ruta calculada con matriz interna: ${formatPoints(calculation.totalBaseKM)} km base. Puedes sumar compensacion justificada.`;
    setStatus("Kilometraje calculado con matriz interna", "ready");
    syncDietasTotal();
    renderRouteMap(mapPanel, calculation, view);
    return calculation;
  };

  renderStops();
  $("[data-add-dietas-stop]", panel).addEventListener("click", () => {
    syncStopsState();
    const insertAt = state.employeeDietasStops.length > 1 && state.employeeDietasStops[0] === state.employeeDietasStops[state.employeeDietasStops.length - 1]
      ? state.employeeDietasStops.length - 1
      : state.employeeDietasStops.length;
    state.employeeDietasStops.splice(insertAt, 0, "");
    renderStops();
    setStatus("Anade la nueva parada de la comision", "ready");
  });
  $("[data-estimate-dietas-route]", panel).addEventListener("click", estimateRoute);
  selectAllowance.addEventListener("change", () => {
    const option = selectAllowance.selectedOptions[0];
    $("input[name='allowance_amount']", form).value = option?.dataset.amount || "0";
    syncDietasTotal();
  });
  $$("input", form).forEach((input) => {
    input.addEventListener("input", syncDietasTotal);
    input.addEventListener("change", syncDietasTotal);
  });
  syncDietasTotal();
  window.requestAnimationFrame(() => {
    syncStopsState();
    const initialCalculation = calculateItinerary(
      readStops(),
      moneyNumber($("input[name='rate']", form).value),
      view,
      new Map(),
    );
    if (initialCalculation.totalBaseKM > 0) {
      $("input[name='km']", form).value = initialCalculation.totalBaseKM.toFixed(1);
      syncDietasTotal();
    }
    renderItineraryResult(routeResult, initialCalculation, null, []);
    renderRouteMap(mapPanel, initialCalculation, view);
  });

  form.addEventListener("submit", (event) => {
    event.preventDefault();
    syncStopsState();
    if (!form.reportValidity()) return;
    const stops = readStops();
    if (stops.length < 2) {
      setStatus("Anade al menos salida y destino", "error");
      return;
    }
    const compensationKM = moneyNumber($("input[name='compensation_km']", form).value);
    const compensationReason = String($("input[name='compensation_reason']", form).value || "").trim();
    if (compensationKM > 0 && !compensationReason) {
      setStatus("Indica el motivo de la compensacion de kilometros", "error");
      $("input[name='compensation_reason']", form).focus();
      return;
    }
    const lodgingAmount = moneyNumber($("input[name='lodging_amount']", form).value);
    const lodgingReference = String($("input[name='lodging_reference']", form).value || "").trim();
    if (lodgingAmount > 0 && !lodgingReference) {
      setStatus("Indica justificante para el alojamiento", "error");
      $("input[name='lodging_reference']", form).focus();
      return;
    }
    const totals = syncDietasTotal();
    const sheets = ensureDietasSheets();
    const id = nextEmployeeFlowID("DIET", sheets);
    const allowanceOption = selectAllowance.selectedOptions[0];
    const travelDate = $("input[name='travel_date']", form).value;
    const routeText = stops.join(" - ");
    const record = {
      id,
      fecha: formatDateForDisplay(travelDate),
      travelDate,
      motivo: String($("input[name='purpose']", form).value || "Comision de servicio").trim(),
      ruta: routeText,
      estado: "Pendiente jefe de servicio",
      importe: totals.total,
      amount: totals.total,
      km: totals.km,
      mileage_amount: totals.mileage,
      allowance: allowanceOption?.textContent || "Sin dieta",
      lodging_amount: lodgingAmount,
      other_expenses: moneyNumber($("input[name='other_expenses']", form).value),
      compensation_km: compensationKM,
      compensation_reason: compensationReason,
      workflow: ["Empleado", "Jefe de servicio", "Tecnico RRHH"],
    };
    sheets.unshift(record);
    saveDietasSheets(sheets);
    recordReceipt("Solicitud dieta enviada", `${id} - ${routeText} - ${formatCurrency(totals.total)}`, "dietas");
    setStatus(`Dieta enviada a jefe de servicio: ${id}`, "ready");
    renderModulePortal(view);
  });

  return panel;
}

function canManageBolsaOffers() {
  return currentRoleList().some((role) => ["administrador", "tecnico_rrhh", "rrhh", "personal_rrhh"].includes(role));
}

function textSearchBase(value) {
  return String(value || "")
    .normalize("NFD")
    .replace(/[̀-ͯ]/g, "")
    .toLowerCase();
}

function employeeBolsaProfile() {
  const employee = payrollEmployeeData();
  const text = textSearchBase(`${employee.position} ${employee.relationship} ${employee.service}`);
  const group = (text.match(/\b(a1|a2|c1|c2|ap|e)\b/i) || [])[1]?.toUpperCase() || "";
  return {
    employee,
    group,
    text,
  };
}

function bolsaOfferEligibility(offer) {
  const profile = employeeBolsaProfile();
  const text = textSearchBase(`${offer.title} ${offer.category} ${offer.requirements} ${offer.basesRef}`);
  const requiredGroups = Array.from(new Set((text.match(/\b(a1|a2|c1|c2|ap|e)\b/gi) || []).map((item) => item.toUpperCase())));
  if (requiredGroups.length) {
    const matchesGroup = requiredGroups.includes(profile.group);
    return {
      eligible: matchesGroup,
      reason: matchesGroup
        ? `Grupo ${profile.group} compatible con la convocatoria`
        : `Requiere ${requiredGroups.join("/")} y tu perfil es ${profile.group || "sin grupo detectado"}`,
    };
  }
  const skillPairs = [
    ["tecnico", "tecnico"],
    ["gestion", "gestion"],
    ["informatica", "informatica"],
    ["administracion", "administracion"],
  ];
  const matched = skillPairs.find(([offerToken, employeeToken]) => text.includes(offerToken) && profile.text.includes(employeeToken));
  if (matched) {
    return { eligible: true, reason: `Categoria relacionada con tu puesto: ${profile.employee.position}` };
  }
  return { eligible: false, reason: "Requisitos a revisar en las bases antes de solicitar" };
}

function employeeEligibleBolsaOffers(view, options = {}) {
  const offers = ensureBolsaOffers(view);
  const applications = ensureEmployeeBolsaApplications(view);
  return offers
    .filter((offer) => /abierta|publicada|en plazo/i.test(offer.state || ""))
    .map((offer) => ({
      offer,
      application: activeBolsaApplicationForOffer(applications, offer.id),
      eligibility: bolsaOfferEligibility(offer),
    }))
    .filter((item) => item.eligibility.eligible || item.application)
    .slice(0, options.limit || 3);
}

function renderEmployeeEligibleBolsaPanel(view, options = {}) {
  const items = employeeEligibleBolsaOffers(view, { limit: options.compact ? 2 : 4 });
  if (!items.length) return null;
  const panel = document.createElement("section");
  panel.className = "employee-flow-panel bolsa-eligible-panel";
  panel.innerHTML = `
    <div class="employee-flow-head">
      <div>
        <h3>Ofertas compatibles con tu perfil</h3>
        <span>Acceso directo a solicitudes de empleo publico que puedes tramitar desde VEC.</span>
      </div>
      <button type="button" class="quiet-action" data-open-bolsa-module>Ver todas</button>
    </div>
    <div class="bolsa-recommendation-grid"></div>
  `;
  $("[data-open-bolsa-module]", panel).addEventListener("click", () => setActiveModule("bolsa"));
  const grid = $(".bolsa-recommendation-grid", panel);
  items.forEach(({ offer, application, eligibility }) => {
    const card = document.createElement("article");
    card.className = "bolsa-offer-card";
    const feeText = offer.feeRequired ? `Tasa ${formatCurrency(offer.feeAmount)}` : "Sin tasa";
    const actionLabel = application ? "Continuar solicitud" : "Apuntarme";
    card.innerHTML = `
      <div>
        <strong></strong>
        <span></span>
      </div>
      <dl>
        <div><dt>Plazo</dt><dd></dd></div>
        <div><dt>Tasa</dt><dd></dd></div>
        <div><dt>Estado</dt><dd></dd></div>
      </dl>
      <p></p>
      <div class="row-action-group"></div>
    `;
    $("strong", card).textContent = offer.title;
    $("span", card).textContent = offer.category || "Categoria profesional";
    const dds = $$("dd", card);
    dds[0].textContent = offer.deadline || "-";
    dds[1].textContent = feeText;
    dds[2].textContent = application ? application.state : "Disponible";
    $("p", card).textContent = eligibility.reason;
    const actions = $(".row-action-group", card);
    const main = document.createElement("button");
    main.type = "button";
    main.className = "row-action";
    main.textContent = actionLabel;
    main.addEventListener("click", () => {
      if (application) {
        state.bolsaSelectedOfferID = offer.id;
        setActiveModule("bolsa");
        return;
      }
      handleEmployeeBolsaSignup(offer, view);
    });
    actions.append(main);
    if (application) {
      const withdraw = document.createElement("button");
      withdraw.type = "button";
      withdraw.className = "row-action is-quiet";
      withdraw.textContent = "Desapuntarme";
      withdraw.addEventListener("click", () => handleEmployeeBolsaWithdraw(offer, view));
      actions.append(withdraw);
    }
    grid.append(card);
  });
  return panel;
}

function normalizeBolsaOffer(offer, index = 0) {
  const feeRequired = offer.feeRequired === true || offer.fee_required === true || /^(si|true|1)$/i.test(String(offer.feeRequired || offer.fee_required || ""));
  const feeAmount = moneyNumber(offer.feeAmount ?? offer.fee_amount ?? (feeRequired ? 15.12 : 0));
  return {
    ...offer,
    feeRequired,
    feeAmount: feeRequired ? feeAmount : 0,
    feeCode: offer.feeCode || offer.fee_code || (feeRequired ? `TASA-BOL-${String(index + 1).padStart(2, "0")}` : ""),
    feeLabel: offer.feeLabel || offer.fee_label || (feeRequired ? "Tasa de derechos de examen/participacion" : "Sin tasa"),
    applications: Number(offer.applications || 0),
  };
}

function ensureBolsaOffers(view) {
  if (Array.isArray(state.bolsaOffers) && state.bolsaOffers.length) return state.bolsaOffers;
  const storedOffers = readStoredArray("vec_demo_bolsa_offers");
  if (storedOffers.length) {
    state.bolsaOffers = storedOffers.map(normalizeBolsaOffer);
    return state.bolsaOffers;
  }
  const categories = getPersonalCatalog(view).categories?.items || view.workspace?.professional_categories || [];
  const seeded = categories.slice(0, 4).map((category, index) => ({
    id: `OFE-2026-${String(index + 1).padStart(4, "0")}`,
    title: `Bolsa ${category.name || category.label || "categoria profesional"}`,
    category: category.name || category.label || "Categoria profesional",
    unit: category.area || "Diputacion de Granada",
    deadline: index === 0 ? "30/06/2026" : "15/07/2026",
    requirements: "Solicitud, titulacion requerida y meritos reutilizables del expediente VEC.",
    basesRef: "Bases publicadas en portal interno",
    state: "Abierta",
    applications: index,
    feeRequired: index === 0,
    feeAmount: index === 0 ? 15.12 : 0,
    feeCode: index === 0 ? "TASA-BOL-01" : "",
    feeLabel: index === 0 ? "Tasa de participacion" : "Sin tasa",
  }));
  state.bolsaOffers = (seeded.length ? seeded : [
    {
      id: "OFE-2026-0001",
      title: "Bolsa tecnico de gestion A2",
      category: "Tecnico de gestion A2",
      unit: "Administracion general",
      deadline: "30/06/2026",
      requirements: "Titulacion A2, servicios prestados y meritos documentados.",
      basesRef: "Bases demo VEC",
      state: "Abierta",
      applications: 0,
      feeRequired: true,
      feeAmount: 15.12,
      feeCode: "TASA-BOL-01",
      feeLabel: "Tasa de participacion",
    },
    {
      id: "OFE-2026-0002",
      title: "Bolsa administrativo C1",
      category: "Administracion general C1",
      unit: "Servicios centrales",
      deadline: "15/07/2026",
      requirements: "Titulacion C1, experiencia y formacion baremable.",
      basesRef: "Bases demo VEC",
      state: "Abierta",
      applications: 0,
      feeRequired: false,
      feeAmount: 0,
      feeCode: "",
      feeLabel: "Sin tasa",
    },
  ]).map(normalizeBolsaOffer);
  writeStoredArray("vec_demo_bolsa_offers", state.bolsaOffers);
  return state.bolsaOffers;
}

function normalizeBolsaApplication(application) {
  const feeRequired = application.feeRequired === true || application.fee_required === true || moneyNumber(application.feeAmount || application.fee_amount) > 0;
  return {
    ...application,
    feeRequired,
    feeAmount: moneyNumber(application.feeAmount ?? application.fee_amount ?? 0),
    paymentState: application.paymentState || application.payment_state || (feeRequired ? "Pendiente pago" : "No exige tasa"),
    signatureState: application.signatureState || application.signature_state || "Pendiente firma",
    form: application.form || {},
  };
}

function ensureEmployeeBolsaApplications(view) {
  if (Array.isArray(state.employeeBolsaApplications)) return state.employeeBolsaApplications;
  const storedApplications = readStoredArray("vec_demo_bolsa_employee_applications");
  if (storedApplications.length) {
    state.employeeBolsaApplications = storedApplications.map(normalizeBolsaApplication);
    return state.employeeBolsaApplications;
  }
  const offers = ensureBolsaOffers(view);
  state.employeeBolsaApplications = [
    {
      id: "BOL-2026-0172",
      offerID: offers[0]?.id || "OFE-2026-0001",
      title: offers[0]?.title || "Bolsa tecnico de gestion A2",
      category: offers[0]?.category || "Tecnico de gestion A2",
      state: "Admitida provisional",
      submittedAt: "18/06/2026",
      feeRequired: offers[0]?.feeRequired === true,
      feeAmount: moneyNumber(offers[0]?.feeAmount || 0),
      paymentState: offers[0]?.feeRequired ? "Pagada" : "No exige tasa",
      paymentReceipt: offers[0]?.feeRequired ? "TASA-20260618-0172" : "",
      signatureState: "Firmada",
      signatureCSV: "CSV-FIR-BOL-2026-0172",
      registryNumber: "REG-BOL-2026-0172",
      form: {
        fullName: payrollEmployeeData().name,
        nif: payrollEmployeeData().nif,
        email: state.candidate.email,
        phone: "600000000",
        qualification: "Titulacion aportada en expediente VEC",
        meritsRef: "Servicios prestados y formacion reutilizados",
        consent: true,
      },
    },
    {
      id: "BOL-2025-0094",
      offerID: "historico-2025",
      title: "Bolsa administracion general",
      category: "Administracion general",
      state: "Cerrada",
      submittedAt: "12/11/2025",
      feeRequired: false,
      feeAmount: 0,
      paymentState: "No exige tasa",
      signatureState: "Firmada",
      signatureCSV: "CSV-FIR-BOL-2025-0094",
      form: {},
    },
  ].map(normalizeBolsaApplication);
  writeStoredArray("vec_demo_bolsa_employee_applications", state.employeeBolsaApplications);
  return state.employeeBolsaApplications;
}

function saveBolsaApplications(applications) {
  state.employeeBolsaApplications = applications.map(normalizeBolsaApplication);
  writeStoredArray("vec_demo_bolsa_employee_applications", state.employeeBolsaApplications);
  return state.employeeBolsaApplications;
}

function activeBolsaApplicationForOffer(applications, offerID) {
  return applications.find((item) =>
    item.offerID === offerID && !/desistida|retirada|cerrada|anulada/i.test(String(item.state || "")),
  );
}

function renderEmployeeBolsaModule(target, view) {
  const offers = ensureBolsaOffers(view);
  const applications = ensureEmployeeBolsaApplications(view);
  const openOffers = offers.filter((offer) => /abierta|publicada|en plazo/i.test(offer.state || ""));
  const activeApplications = applications.filter((item) => !/desistida|retirada|cerrada|anulada/i.test(item.state || ""));
  const selectedOffer = offers.find((offer) => offer.id === state.bolsaSelectedOfferID);
  const selectedApplication = selectedOffer ? activeBolsaApplicationForOffer(applications, selectedOffer.id) : null;
  target.append(modulePortalHeader("Mis bolsas y ofertas", "Consulta ofertas abiertas y tramita tus propias solicitudes de Bolsa.", []));
  target.append(portalGrid([
    ["Ofertas abiertas", String(openOffers.length)],
    ["Mis solicitudes", String(applications.length)],
    ["Firmas pendientes", String(activeApplications.filter((item) => !/firmada/i.test(item.signatureState || "")).length)],
    ["Tasas pendientes", String(activeApplications.filter((item) => item.feeRequired && !/pagada/i.test(item.paymentState || "")).length)],
  ]));
  target.append(renderEmployeeBolsaOffersPanel(view, offers, applications));
  if (selectedOffer) {
    target.append(renderBolsaApplicationForm(view, selectedOffer, selectedApplication));
  }
  target.append(portalTable("Mis solicitudes", ["Expediente", "Oferta", "Fecha", "Estado", "Firma", "Tasa", "Accion"], applications.map((item) => [
    item.id,
    item.title,
    item.submittedAt || "-",
    item.state,
    item.signatureState || "Pendiente firma",
    item.paymentState || (item.feeRequired ? "Pendiente pago" : "No exige tasa"),
    bolsaApplicationAction(item),
  ]), { actionColumn: true }));
}

function bolsaApplicationAction(application) {
  const stateText = String(application.state || "");
  if (/cerrada/i.test(stateText)) return "Ver certificado";
  if (/desistida|retirada/i.test(stateText)) return "Ver desistimiento";
  if (/borrador/i.test(stateText)) return "Completar";
  if (application.feeRequired && !/pagada/i.test(application.paymentState || "")) return "Pagar tasa";
  if (!/firmada/i.test(application.signatureState || "")) return "Firmar";
  return "Ver solicitud";
}

function renderEmployeeBolsaOffersPanel(view, offers, applications) {
  const panel = document.createElement("section");
  panel.className = "employee-flow-panel bolsa-offer-panel";
  panel.innerHTML = `
    <div class="employee-flow-head">
      <div>
        <h3>Ofertas disponibles para apuntarme</h3>
        <span>Solo se muestran ofertas abiertas o en plazo. La inscripcion queda registrada como solicitud propia.</span>
      </div>
    </div>
    <div class="table-wrap">
      <table>
        <thead><tr><th>Oferta</th><th>Categoria</th><th>Unidad</th><th>Plazo</th><th>Tasa</th><th>Estado</th><th>Accion</th></tr></thead>
        <tbody></tbody>
      </table>
    </div>
  `;
  const tbody = $("tbody", panel);
  const visibleOffers = offers.filter((offer) => !/cerrada|retirada|finalizada/i.test(offer.state || ""));
  if (!visibleOffers.length) {
    const row = document.createElement("tr");
    const cell = document.createElement("td");
    cell.colSpan = 7;
    cell.textContent = "No hay ofertas abiertas en este momento.";
    row.append(cell);
    tbody.append(row);
  }
  visibleOffers.forEach((offer, index) => {
    const row = document.createElement("tr");
    row.style.background = index % 2 ? "#f8fafc" : "#fff";
    const activeApplication = activeBolsaApplicationForOffer(applications, offer.id);
    const feeText = offer.feeRequired ? `${formatCurrency(offer.feeAmount)} · ${offer.feeCode || "tasa"}` : "No";
    [offer.title, offer.category, offer.unit, offer.deadline, feeText, offer.state].forEach((value, cellIndex) => {
      const cell = document.createElement("td");
      if (cellIndex === 5) {
        const chip = document.createElement("span");
        chip.className = `status-chip ${stateTone(value)}`;
        chip.textContent = value || "-";
        cell.append(chip);
      } else {
        cell.textContent = value || "-";
      }
      row.append(cell);
    });
    const actionCell = document.createElement("td");
    const actions = document.createElement("div");
    actions.className = "row-action-group";
    const mainButton = document.createElement("button");
    mainButton.type = "button";
    mainButton.className = "row-action";
    mainButton.textContent = activeApplication ? "Formulario" : "Apuntarme";
    mainButton.addEventListener("click", () => {
      if (activeApplication) {
        state.bolsaSelectedOfferID = offer.id;
        renderModulePortal(view);
        return;
      }
      handleEmployeeBolsaSignup(offer, view);
    });
    actions.append(mainButton);
    if (activeApplication) {
      const withdrawButton = document.createElement("button");
      withdrawButton.type = "button";
      withdrawButton.className = "row-action is-quiet";
      withdrawButton.textContent = "Desapuntarme";
      withdrawButton.addEventListener("click", () => handleEmployeeBolsaWithdraw(offer, view));
      actions.append(withdrawButton);
    }
    actionCell.append(actions);
    row.append(actionCell);
    tbody.append(row);
  });
  return panel;
}

function handleEmployeeBolsaSignup(offer, view) {
  const applications = ensureEmployeeBolsaApplications(view);
  if (activeBolsaApplicationForOffer(applications, offer.id)) {
    setStatus(`Ya estas inscrito en ${offer.id}`, "ready");
    state.bolsaSelectedOfferID = offer.id;
    renderModulePortal(view);
    return;
  }
  const id = nextEmployeeFlowID("BOL", applications);
  const employee = payrollEmployeeData();
  const application = {
    id,
    offerID: offer.id,
    title: offer.title,
    category: offer.category,
    state: "Borrador",
    createdAt: new Date().toLocaleString("es-ES"),
    feeRequired: offer.feeRequired === true,
    feeAmount: moneyNumber(offer.feeAmount || 0),
    paymentState: offer.feeRequired ? "Pendiente pago" : "No exige tasa",
    signatureState: "Pendiente firma",
    form: {
      fullName: employee.name,
      nif: employee.nif,
      email: state.candidate.email,
      phone: "",
      qualification: "",
      meritsRef: "Servicios prestados y meritos disponibles en VEC",
      consent: false,
    },
    workflow: ["Empleado", "Pago de tasa si procede", "Firma electronica", "Tecnico RRHH", "Listado provisional", "Listado definitivo"],
  };
  applications.unshift(application);
  offer.applications = Number(offer.applications || 0) + 1;
  saveBolsaApplications(applications);
  writeStoredArray("vec_demo_bolsa_offers", ensureBolsaOffers(view));
  state.bolsaSelectedOfferID = offer.id;
  recordReceipt("Borrador Bolsa creado", `${id} - ${offer.title}`, "bolsa");
  setStatus(`Completa formulario, tasa y firma para presentar ${id}`, "ready");
  if (isEmployeeSelfServiceSession()) {
    state.activeModule = "bolsa";
    state.activeScreen = "";
  }
  renderModulePortal(view);
}

function handleEmployeeBolsaWithdraw(offer, view) {
  const applications = ensureEmployeeBolsaApplications(view);
  const application = activeBolsaApplicationForOffer(applications, offer.id);
  if (!application) {
    setStatus(`No hay solicitud activa en ${offer.id}`, "error");
    return;
  }
  if (/admitida definitiva|cerrada/i.test(String(application.state || ""))) {
    setStatus("La solicitud ya esta cerrada; no puede desistirse desde autoservicio.", "error");
    return;
  }
  if (!window.confirm(`Desapuntarte de ${offer.title}? Se registrara un desistimiento.`)) return;
  application.state = "Desistida";
  application.withdrawnAt = new Date().toLocaleString("es-ES");
  application.withdrawalReceipt = `DES-BOL-${String(Date.now()).slice(-8)}`;
  offer.applications = Math.max(0, Number(offer.applications || 0) - 1);
  saveBolsaApplications(applications);
  writeStoredArray("vec_demo_bolsa_offers", ensureBolsaOffers(view));
  if (state.bolsaSelectedOfferID === offer.id) state.bolsaSelectedOfferID = "";
  recordReceipt("Desistimiento Bolsa", `${application.withdrawalReceipt} - ${application.id} - ${offer.title}`, "bolsa");
  setStatus(`Desapuntado de ${offer.id}`, "ready");
  if (isEmployeeSelfServiceSession()) {
    state.activeModule = "bolsa";
    state.activeScreen = "";
  }
  renderModulePortal(view);
}

function ensureBolsaApplicationDraft(offer, view) {
  const applications = ensureEmployeeBolsaApplications(view);
  let application = activeBolsaApplicationForOffer(applications, offer.id);
  if (application) return application;
  handleEmployeeBolsaSignup(offer, view);
  return activeBolsaApplicationForOffer(ensureEmployeeBolsaApplications(view), offer.id);
}

function renderBolsaApplicationForm(view, offer, application) {
  const panel = document.createElement("section");
  panel.className = "employee-flow-panel bolsa-application-panel";
  const employee = payrollEmployeeData();
  const app = application || null;
  const feeRequired = offer.feeRequired === true;
  const feeState = app?.paymentState || (feeRequired ? "Pendiente pago" : "No exige tasa");
  const signatureState = app?.signatureState || "Pendiente firma";
  const formData = app?.form || {};
  panel.innerHTML = `
    <div class="employee-flow-head">
      <div>
        <h3>Formulario de solicitud</h3>
        <span>${offer.title} · ${offer.category || "Categoria profesional"}</span>
      </div>
    </div>
    <form class="employee-flow-form" data-bolsa-application-form>
      <input name="application_id" type="hidden">
      <input name="offer_id" type="hidden">
      <label class="employee-field-wide">Nombre y apellidos
        <input name="full_name" required>
      </label>
      <label>NIF/NIE
        <input name="nif" required>
      </label>
      <label>Email
        <input name="email" type="email" required>
      </label>
      <label>Telefono
        <input name="phone" inputmode="tel" required>
      </label>
      <label class="employee-field-wide">Titulacion alegada
        <input name="qualification" placeholder="Titulacion exigida por las bases" required>
      </label>
      <label class="employee-field-wide">Meritos que se incorporan
        <input name="merits_ref" placeholder="Servicios prestados, formacion, titulacion, expediente VEC">
      </label>
      <label>Justificante titulo
        <input name="qualification_doc" placeholder="CSV/documento">
      </label>
      <label>Servicios prestados
        <input name="services_doc" placeholder="CSV/certificado">
      </label>
      <label class="employee-check employee-field-wide">
        <input name="consent" type="checkbox" required>
        Declaro que los datos son ciertos y solicito la firma electronica de esta solicitud.
      </label>
      <div class="bolsa-flow-strip employee-field-wide">
        <span><b>Estado</b>${app?.state || "Borrador no guardado"}</span>
        <span><b>Firma</b>${signatureState}${app?.signatureCSV ? ` · ${app.signatureCSV}` : ""}</span>
        <span><b>Tasa</b>${feeRequired ? `${formatCurrency(offer.feeAmount)} · ${feeState}` : "No exige tasa"}</span>
      </div>
      <div class="employee-flow-note employee-field-wide">
        La presentacion requiere firma electronica. Si la oferta exige tasa, debe constar pagada antes de registrar la solicitud.
      </div>
      <div class="employee-form-actions employee-field-wide">
        <button type="button" class="quiet-action" data-save-bolsa-draft>Guardar borrador</button>
        <button type="button" class="quiet-action" data-pay-bolsa-fee ${feeRequired ? "" : "disabled"}>Pagar tasa</button>
        <button type="button" class="quiet-action" data-sign-bolsa-application>Solicitar firma electronica</button>
        <button type="button" class="primary-action" data-submit-bolsa-application>Presentar solicitud</button>
      </div>
    </form>
  `;
  const form = $("form", panel);
  setFormValue(form, "application_id", app?.id || "");
  setFormValue(form, "offer_id", offer.id);
  setFormValue(form, "full_name", formData.fullName || employee.name);
  setFormValue(form, "nif", formData.nif || employee.nif);
  setFormValue(form, "email", formData.email || state.candidate.email || "empleado.demo@example.test");
  setFormValue(form, "phone", formData.phone || "");
  setFormValue(form, "qualification", formData.qualification || "");
  setFormValue(form, "merits_ref", formData.meritsRef || "Servicios prestados y meritos disponibles en VEC");
  setFormValue(form, "qualification_doc", formData.qualificationDoc || "");
  setFormValue(form, "services_doc", formData.servicesDoc || "");
  const consent = $("input[name='consent']", form);
  if (consent) consent.checked = formData.consent === true;

  $("[data-save-bolsa-draft]", form).addEventListener("click", () => {
    const saved = saveBolsaApplicationDraft(offer, form, view);
    state.bolsaSelectedOfferID = offer.id;
    recordReceipt("Borrador Bolsa guardado", `${saved.id} - ${offer.title}`, "bolsa");
    setStatus(`Borrador guardado: ${saved.id}`, "ready");
    renderModulePortal(view);
  });
  $("[data-pay-bolsa-fee]", form).addEventListener("click", () => handleBolsaFeePayment(offer, form, view));
  $("[data-sign-bolsa-application]", form).addEventListener("click", () => handleBolsaSignature(offer, form, view));
  $("[data-submit-bolsa-application]", form).addEventListener("click", () => handleBolsaApplicationSubmit(offer, form, view));
  return panel;
}

function readBolsaApplicationForm(form) {
  return {
    fullName: String(form.elements.full_name?.value || "").trim(),
    nif: String(form.elements.nif?.value || "").trim(),
    email: String(form.elements.email?.value || "").trim(),
    phone: String(form.elements.phone?.value || "").trim(),
    qualification: String(form.elements.qualification?.value || "").trim(),
    meritsRef: String(form.elements.merits_ref?.value || "").trim(),
    qualificationDoc: String(form.elements.qualification_doc?.value || "").trim(),
    servicesDoc: String(form.elements.services_doc?.value || "").trim(),
    consent: form.elements.consent?.checked === true,
  };
}

function saveBolsaApplicationDraft(offer, form, view) {
  const applications = ensureEmployeeBolsaApplications(view);
  let application = activeBolsaApplicationForOffer(applications, offer.id);
  if (!application) {
    const employee = payrollEmployeeData();
    application = {
      id: nextEmployeeFlowID("BOL", applications),
      offerID: offer.id,
      title: offer.title,
      category: offer.category,
      state: "Borrador",
      createdAt: new Date().toLocaleString("es-ES"),
      feeRequired: offer.feeRequired === true,
      feeAmount: moneyNumber(offer.feeAmount || 0),
      paymentState: offer.feeRequired ? "Pendiente pago" : "No exige tasa",
      signatureState: "Pendiente firma",
      form: { fullName: employee.name, nif: employee.nif, email: state.candidate.email },
      workflow: ["Empleado", "Pago de tasa si procede", "Firma electronica", "Tecnico RRHH", "Listados"],
    };
    applications.unshift(application);
    offer.applications = Number(offer.applications || 0) + 1;
  }
  application.form = readBolsaApplicationForm(form);
  application.updatedAt = new Date().toLocaleString("es-ES");
  if (!application.state || /pendiente firma|pendiente pago/i.test(application.state)) application.state = "Borrador";
  application.feeRequired = offer.feeRequired === true;
  application.feeAmount = moneyNumber(offer.feeAmount || 0);
  if (!application.feeRequired) application.paymentState = "No exige tasa";
  saveBolsaApplications(applications);
  writeStoredArray("vec_demo_bolsa_offers", ensureBolsaOffers(view));
  return ensureEmployeeBolsaApplications(view).find((item) => item.id === application.id) || application;
}

function handleBolsaFeePayment(offer, form, view) {
  const application = saveBolsaApplicationDraft(offer, form, view);
  if (!offer.feeRequired) {
    setStatus("Esta oferta no exige tasa.", "ready");
    return;
  }
  application.paymentState = "Pagada";
  application.paymentReceipt = `TASA-${new Date().toISOString().slice(0, 10).replaceAll("-", "")}-${String(Date.now()).slice(-5)}`;
  application.paidAt = new Date().toLocaleString("es-ES");
  saveBolsaApplications(ensureEmployeeBolsaApplications(view));
  recordReceipt("Tasa Bolsa pagada", `${application.paymentReceipt} - ${application.id} - ${formatCurrency(offer.feeAmount)}`, "bolsa");
  setStatus(`Tasa pagada: ${application.paymentReceipt}`, "ready");
  renderModulePortal(view);
}

function handleBolsaSignature(offer, form, view) {
  const application = saveBolsaApplicationDraft(offer, form, view);
  const auth = activeDemoUser().auth === "dnie" ? "DNIe/certificado digital" : "certificado electronico/Cl@ve firma";
  application.signatureState = "Firmada";
  application.signatureCSV = `CSV-FIR-${application.id.replace(/\W+/g, "-")}`;
  application.signedAt = new Date().toLocaleString("es-ES");
  application.signatureProvider = auth;
  saveBolsaApplications(ensureEmployeeBolsaApplications(view));
  recordReceipt("Firma electronica Bolsa", `${application.signatureCSV} - ${application.id} - ${auth}`, "bolsa");
  setStatus(`Firma electronica solicitada y registrada: ${application.signatureCSV}`, "ready");
  renderModulePortal(view);
}

function handleBolsaApplicationSubmit(offer, form, view) {
  if (!form.reportValidity()) {
    setStatus("Completa los campos obligatorios antes de presentar.", "error");
    return;
  }
  const application = saveBolsaApplicationDraft(offer, form, view);
  if (offer.feeRequired && !/pagada/i.test(application.paymentState || "")) {
    setStatus("Esta oferta exige tasa: realiza el pago antes de presentar.", "error");
    return;
  }
  if (!/firmada/i.test(application.signatureState || "")) {
    setStatus("Solicita la firma electronica antes de presentar.", "error");
    return;
  }
  application.state = "Presentada";
  application.submittedAt = new Date().toLocaleDateString("es-ES");
  application.registryNumber = application.registryNumber || `REG-BOL-${new Date().getFullYear()}-${String(Date.now()).slice(-6)}`;
  saveBolsaApplications(ensureEmployeeBolsaApplications(view));
  recordReceipt("Solicitud Bolsa presentada", `${application.registryNumber} - ${application.id} - ${offer.title}`, "bolsa");
  setStatus(`Solicitud presentada y firmada: ${application.registryNumber}`, "ready");
  renderModulePortal(view);
}

function bolsaOfferManagementPanel(view) {
  const offers = ensureBolsaOffers(view);
  const categories = getPersonalCatalog(view).categories?.items || view.workspace?.professional_categories || [];
  const panel = document.createElement("section");
  panel.className = "employee-flow-panel bolsa-offer-panel";
  panel.innerHTML = `
    <div class="employee-flow-head">
      <div>
        <h3>Nueva oferta de Bolsa</h3>
        <span>Alta operativa para RRHH. Al publicar, los empleados la ven en su autoservicio y pueden apuntarse.</span>
      </div>
    </div>
    <form class="employee-flow-form" data-bolsa-offer-form>
      <input name="offer_id" type="hidden">
      <label class="employee-field-wide">Titulo de la oferta
        <input name="title" value="Bolsa temporal" required>
      </label>
      <label>Categoria
        <select name="category" required></select>
      </label>
      <label>Unidad / area
        <input name="unit" value="Diputacion de Granada" required>
      </label>
      <label>Plazo fin
        <input name="deadline" type="date" value="2026-07-15" required>
      </label>
      <label>Estado
        <select name="state" required>
          <option>Abierta</option>
          <option>Publicada</option>
          <option>Borrador</option>
          <option>Cerrada</option>
        </select>
      </label>
      <label class="employee-check">
        <input name="fee_required" type="checkbox">
        Exige tasa
      </label>
      <label>Importe tasa
        <input name="fee_amount" type="number" min="0" step="0.01" value="0">
      </label>
      <label>Codigo tasa
        <input name="fee_code" placeholder="TASA-BOL-01">
      </label>
      <label class="employee-field-wide">Requisitos
        <input name="requirements" value="Titulacion requerida, servicios prestados y meritos baremables" required>
      </label>
      <label class="employee-field-wide">Referencia bases
        <input name="bases_ref" value="Bases publicadas en VEC" required>
      </label>
      <div class="employee-form-actions employee-field-wide">
        <button type="submit" class="primary-action">Guardar oferta</button>
      </div>
    </form>
  `;
  const form = $("form", panel);
  const categorySelect = $("select[name='category']", form);
  const categoryValues = categories.length
    ? categories.map((item) => item.name || item.label || item.slug).filter(Boolean)
    : ["Tecnico de gestion A2", "Administrativo C1", "Auxiliar administrativo C2", "Trabajador social A2"];
  categoryValues.slice(0, 40).forEach((category) => {
    const option = document.createElement("option");
    option.value = category;
    option.textContent = category;
    categorySelect.append(option);
  });
  if (categoryValues[0]) {
    categorySelect.value = categoryValues[0];
    $("input[name='title']", form).value = `Bolsa ${categoryValues[0]}`;
  }
  categorySelect.addEventListener("change", () => {
    $("input[name='title']", form).value = `Bolsa ${categorySelect.value}`;
  });
  form.addEventListener("submit", (event) => {
    event.preventDefault();
    if (!form.reportValidity()) return;
    const data = new FormData(form);
    const deadline = formatDateForDisplay(data.get("deadline"));
    const existingID = String(data.get("offer_id") || "").trim();
    const existing = offers.find((item) => item.id === existingID);
    const offer = existing || {
      id: nextEmployeeFlowID("OFE", offers),
      applications: 0,
    };
    Object.assign(offer, {
      title: String(data.get("title") || "").trim(),
      category: String(data.get("category") || "").trim(),
      unit: String(data.get("unit") || "").trim(),
      deadline,
      requirements: String(data.get("requirements") || "").trim(),
      basesRef: String(data.get("bases_ref") || "").trim(),
      state: String(data.get("state") || "Abierta").trim(),
      feeRequired: data.get("fee_required") === "on",
      feeAmount: data.get("fee_required") === "on" ? moneyNumber(data.get("fee_amount")) : 0,
      feeCode: data.get("fee_required") === "on" ? String(data.get("fee_code") || "TASA-BOL").trim() : "",
      feeLabel: data.get("fee_required") === "on" ? "Tasa de participacion" : "Sin tasa",
      publishedAt: offer.publishedAt || new Date().toLocaleString("es-ES"),
      updatedAt: new Date().toLocaleString("es-ES"),
      workflow: ["Tecnico RRHH", "Publicada", "Solicitudes empleado", "Listados"],
    });
    if (!existing) offers.unshift(offer);
    writeStoredArray("vec_demo_bolsa_offers", offers);
    recordReceipt(existing ? "Oferta Bolsa actualizada" : "Oferta Bolsa publicada", `${offer.id} - ${offer.title}`, "bolsa");
    setStatus(`${existing ? "Oferta actualizada" : "Oferta publicada"}: ${offer.id}`, "ready");
    renderModulePortal(view);
  });
  return panel;
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
    setActiveModule(defaultModuleID());
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
        { name: "ENF.GRAVE/HOSPIT/INTERV. 1º GRADO", action: "(J-A) Solicitar", max: "5 (*)", min: "", solic: "", resta: "5", requested: false },
        { name: "ENFERMEDAD SIN BAJA (PERMISO)", action: "", max: "4", min: "", solic: "", resta: "3", requested: true },
        { name: "ENFERMEDAD SIN BAJA CON DESCUENTO", action: "", max: "365", min: "", solic: "", resta: "365", requested: false },
        { name: "FALLEC FAMIL. 2º GRADO DIST. LOCALIDAD", action: "(J-A) Solicitar", max: "4 (*)", min: "", solic: "", resta: "4", requested: false },
        { name: "FALLECIMIENTO FAMILIAR 1º GRADO", action: "(J-A) Solicitar", max: "3 (*)", min: "", solic: "", resta: "3", requested: false },
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
        { name: "VACACIONES", action: "", max: "22", min: "", solic: "", resta: "0", requested: true }
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
    const readButton = $("button", table);
    if (readButton) {
      readButton.addEventListener("click", () => {
        const row = readButton.closest("tr");
        const subject = row?.children?.[1]?.textContent || "Mensaje interno";
        row?.classList.add("is-selected");
        readButton.textContent = "Leido";
        readButton.disabled = true;
        recordReceipt("Mensaje Cronos leido", subject, "cronos");
        setStatus(`Mensaje leido: ${subject}`, "ready");
      });
    }
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

  const sectLineas = document.createElement("section");
  sectLineas.className = "dietas-draft-lines";
  sectLineas.style.border = "1px solid #cbd5e1";
  sectLineas.style.borderRadius = "6px";
  sectLineas.style.padding = "12px";
  sectLineas.style.background = "#f8fafc";
  sectLineas.innerHTML = `
    <div style="display:flex; justify-content:space-between; gap:12px; align-items:center; margin-bottom:8px;">
      <strong>Lineas anadidas a la solicitud</strong>
      <span class="small-text" data-dietas-lines-count>0 lineas</span>
    </div>
    <div class="dietas-lines-empty small-text">Todavia no hay rutas, medios o gastos anadidos.</div>
    <table class="dietas-lines-table" style="width:100%; border-collapse:collapse; display:none;">
      <thead>
        <tr style="background:#e2e8f0;">
          <th style="text-align:left; padding:6px;">Tipo</th>
          <th style="text-align:left; padding:6px;">Detalle</th>
          <th style="text-align:right; padding:6px;">Importe</th>
          <th style="text-align:center; padding:6px;">Accion</th>
        </tr>
      </thead>
      <tbody></tbody>
    </table>
  `;
  form.append(sectLineas);

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
  const draftLines = { routes: [], transports: [], expenses: [] };
  const draftLineRows = () => [
    ...draftLines.routes.map((item, localIndex) => ({ ...item, localIndex, kind: "Ruta" })),
    ...draftLines.transports.map((item, localIndex) => ({ ...item, localIndex, kind: "Medio" })),
    ...draftLines.expenses.map((item, localIndex) => ({ ...item, localIndex, kind: "Gasto" })),
  ];
  const renderDraftLines = () => {
    const rows = draftLineRows();
    const count = $("[data-dietas-lines-count]", form);
    const empty = $(".dietas-lines-empty", form);
    const table = $(".dietas-lines-table", form);
    const tbody = $("tbody", table);
    if (count) count.textContent = `${rows.length} ${rows.length === 1 ? "linea" : "lineas"}`;
    if (empty) empty.style.display = rows.length ? "none" : "block";
    if (table) table.style.display = rows.length ? "table" : "none";
    tbody.replaceChildren(...rows.map((item, index) => {
      const row = document.createElement("tr");
      row.style.background = index % 2 ? "#fff" : "#f1f5f9";
      const removeButton = document.createElement("button");
      removeButton.type = "button";
      removeButton.className = "table-action";
      removeButton.textContent = "Quitar";
      removeButton.style.padding = "3px 8px";
      removeButton.addEventListener("click", () => {
        if (item.kind === "Ruta") draftLines.routes.splice(item.localIndex, 1);
        if (item.kind === "Medio") draftLines.transports.splice(item.localIndex, 1);
        if (item.kind === "Gasto") draftLines.expenses.splice(item.localIndex, 1);
        renderDraftLines();
        recalculateTotal();
      });
      [
        item.kind,
        item.detail,
        formatCurrency(item.amount),
      ].forEach((value, cellIndex) => {
        const td = document.createElement("td");
        td.style.padding = "6px";
        td.style.borderTop = "1px solid #e2e8f0";
        if (cellIndex === 2) td.style.textAlign = "right";
        td.textContent = value;
        row.append(td);
      });
      const actionCell = document.createElement("td");
      actionCell.style.padding = "6px";
      actionCell.style.borderTop = "1px solid #e2e8f0";
      actionCell.style.textAlign = "center";
      actionCell.append(removeButton);
      row.append(actionCell);
      return row;
    }));
  };

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

    draftLineRows().forEach((item) => {
      sum += Number(item.amount || 0);
    });

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

  $(".btn-anadir-ruta", form)?.addEventListener("click", () => {
    if (selectVehiculo.value !== "Si") {
      setStatus("Activa vehiculo propio antes de anadir una ruta.", "error");
      return;
    }
    const from = $(".form-km-salida", form)?.value || "Origen";
    const to = $(".form-km-llegada", form)?.value || "Destino";
    const calculation = calculateItinerary([from, to], 0.26, view, new Map());
    const baseKm = calculation.totalBaseKM > 0 ? calculation.totalBaseKM : 70;
    const ajuste = parseFloat(inputAjuste.value) || 0;
    const totalKm = Math.max(0, baseKm + ajuste);
    const amount = totalKm * 0.26;
    draftLines.routes.push({
      localIndex: draftLines.routes.length,
      detail: `${from} -> ${to} · ${formatPoints(totalKm)} km${ajuste ? ` (${formatPoints(ajuste)} km compensacion)` : ""}`,
      amount,
      km: totalKm,
    });
    renderDraftLines();
    recalculateTotal();
    recordReceipt("Ruta Dietas anadida", `${from} -> ${to} - ${formatPoints(totalKm)} km`, "dietas");
    setStatus("Ruta anadida a la solicitud de dietas", "ready");
  });

  $(".btn-anadir-otro-medio", form)?.addEventListener("click", () => {
    const medium = $(".form-otros-medios", form)?.value || "Otro medio";
    const description = $(".form-gastos-justificados", form)?.value || "Gasto justificado";
    const amount = parseFloat(inputOtrosMediosImp.value) || 0;
    if (amount <= 0) {
      setStatus("Indica el importe del medio de transporte.", "error");
      inputOtrosMediosImp.focus();
      return;
    }
    draftLines.transports.push({
      localIndex: draftLines.transports.length,
      detail: `${medium} · ${description}`,
      amount,
    });
    $(".form-gastos-justificados", form).value = "";
    inputOtrosMediosImp.value = "0";
    renderDraftLines();
    recalculateTotal();
    recordReceipt("Medio de transporte anadido", `${medium} - ${formatCurrency(amount)}`, "dietas");
    setStatus("Medio de transporte anadido a la solicitud", "ready");
  });

  $(".btn-anadir-otro-gasto", form)?.addEventListener("click", () => {
    const reason = $(".form-otros-gastos-motivo", form)?.value || "Gasto justificado";
    const amount = parseFloat(inputOtrosGastosImp.value) || 0;
    if (amount <= 0) {
      setStatus("Indica el importe del gasto.", "error");
      inputOtrosGastosImp.focus();
      return;
    }
    draftLines.expenses.push({
      localIndex: draftLines.expenses.length,
      detail: reason,
      amount,
    });
    $(".form-otros-gastos-motivo", form).value = "";
    inputOtrosGastosImp.value = "0";
    renderDraftLines();
    recalculateTotal();
    recordReceipt("Gasto Dietas anadido", `${reason} - ${formatCurrency(amount)}`, "dietas");
    setStatus("Gasto anadido a la solicitud", "ready");
  });

  form.addEventListener("submit", (e) => {
    e.preventDefault();

    const finalAmount = recalculateTotal();
    const motivo = $(".form-motivo", form).value;

    const sheets = ensureDietasSheets();
    sheets.push({
      id: (81158 + sheets.length).toString(),
      motivo: motivo || "Comisión de servicio",
      fecha: new Date().toLocaleDateString("es-ES"),
      importe: finalAmount.toFixed(2) + " €",
      estado: "Pendiente de revisar",
      lineas: draftLineRows().map((item) => ({
        tipo: item.kind,
        detalle: item.detail,
        importe: item.amount,
        km: item.km || 0,
      })),
    });
    saveDietasSheets(sheets);

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
  const routeMatrixScreen = isRouteMatrixScreen(screen);

  // Cabecera compacta: titulo, descripcion y una sola accion primaria.
  if (!routeMatrixScreen) {
    target.append(screenHead(
      screen.title || MODULE_COPY[state.activeModule]?.[0] || "Pantalla VEC",
      screen.description || MODULE_COPY[state.activeModule]?.[1] || "Pantalla operativa del modulo.",
      actions,
    ));
  }

  // Contadores de estado clicables que filtran la tabla (patron Factorial/Sesame/Concur).
  if (!routeMatrixScreen) {
    target.append(screenStateCounters(screen, rows, headers));
  }

  if (isLeaveScreen(screen)) {
    target.append(leaveRequestPanel(screen, view));
  }

  if (isRPTScreen(screen)) {
    target.append(rptPositionPanel(screen));
  }

  if (screen.id === "admin.catalogos") {
    target.append(categoryCatalogPanel());
  }

  if (screen.id === "bolsa.convocatorias" && canManageBolsaOffers()) {
    target.append(bolsaOfferManagementPanel(view));
  }

  if (routeMatrixScreen) {
    target.append(routeMatrixPanel(screen, view));
    return;
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
  const actionLabel = rowActionLabel(screen);
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
    const rowLabel = row._actionLabel || actionLabel;
    button.textContent = rowLabel;
    button.addEventListener("click", (event) => {
      event.stopPropagation();
      handleScreenRowAction(screen, row, headers, rowLabel);
    });
    actionTd.append(button);
    tr.append(actionTd);
    tbody.append(tr);
  });
  tableWrap.append(table);
  wrap.append(header, tableWrap);
  return wrap;
}

function rowActionLabel(screen) {
  if (screen.id === "personal.puestos") return "Editar";
  if (screen.id === "admin.catalogos") return "Editar";
  if (screen.id === "bolsa.convocatorias") return "Editar oferta";
  return (screen.actions || [])[0] || "Abrir";
}

function isLeaveScreen(screen) {
  return ["permisos.solicitudes", "permisos.vacaciones", "permisos.saldos"].includes(screen.id);
}

function isRPTScreen(screen) {
  return screen.id === "personal.puestos";
}

function isRouteMatrixScreen(screen) {
  return screen.id === "rutas.kilometraje" || screen.id === "rutas.mapa_provincia";
}

function routeMatrixPanel(screen, view) {
  const panel = document.createElement("section");
  panel.className = "leave-request-panel route-matrix-panel";
  const matrix = view.workspace?.province_route_matrix || {};
  const points = view.workspace?.province_route_points || [];
  const expensePolicy = view.workspace?.expense_policy || {};
  const allowanceTypes = normalizeAllowanceTypes(expensePolicy.allowance_types);
  const examples = view.workspace?.province_itinerary_examples || [];
  const example = examples[0] || {};
  const rate = Number(example.mileage_rate_eur_km || expensePolicy?.mileage?.rate_eur_km || 0.26);
  const defaultStops = (example.stops || ["Granada", "Albolote", "Mecina Bombarón", "Motril", "Granada"]).slice();
  let selectedStops = defaultStops.length >= 2 ? defaultStops : ["Granada", "Granada"];
  const legAdjustmentStore = new Map();
  const viaPoints = routeSelectableViaPoints(points);
  let currentCalculation = null;
  const header = document.createElement("div");
  header.className = "panel-header";
  header.innerHTML = `<div><h3></h3><span class="small-text"></span></div>`;
  $("h3", header).textContent = "Solicitud diaria de dietas y kilometraje";
  $(".small-text", header).textContent = "Selecciona el dia, registra las rutas del desplazamiento, anade manutencion/alojamiento y envia a validacion.";

  const claimPanel = dietasDailyClaimPanel(allowanceTypes, expensePolicy);

  const form = document.createElement("form");
  form.className = "flow-form leave-form";
  form.innerHTML = `
    <div class="route-stop-builder">
      <div class="route-stop-toolbar">
        <strong>Rutas del dia</strong>
        <button type="button" class="quiet-action" data-add-stop>Anadir parada</button>
      </div>
      <div class="route-stop-list"></div>
    </div>
    <label>Tarifa kilometraje
      <input name="rate" type="number" min="0" step="0.01" value="${rate}">
    </label>
    <button type="submit" class="primary-action">Calcular itinerario</button>
  `;
  const stopList = $(".route-stop-list", form);
  const renderStops = () => {
    stopList.replaceChildren(...selectedStops.map((stop, index) =>
      routeStopRow(stop, index, selectedStops.length, points, (nextValue) => {
        selectedStops[index] = nextValue;
      }, () => {
        selectedStops.splice(index, 1);
        renderStops();
      }),
    ));
  };
  renderStops();
  $("[data-add-stop]", form).addEventListener("click", () => {
    const insertAt = selectedStops.length > 1 && selectedStops[0] === selectedStops[selectedStops.length - 1]
      ? selectedStops.length - 1
      : selectedStops.length;
    selectedStops.splice(insertAt, 0, "");
    renderStops();
    setStatus("Selecciona la nueva localidad del itinerario", "ready");
  });

  const result = document.createElement("div");
  result.className = "route-result";
  const mapPanel = routeMapPanel();
  const routeWorkspace = document.createElement("div");
  routeWorkspace.className = "route-workspace";
  const routeEditor = document.createElement("div");
  routeEditor.className = "route-editor-column";
  const routeMapColumn = document.createElement("div");
  routeMapColumn.className = "route-map-column";

  const updateClaimSummary = (calculation = currentCalculation) => {
    currentCalculation = calculation || currentCalculation;
    renderDietasClaimSummary(claimPanel, currentCalculation, readDietasDailyClaim(claimPanel, allowanceTypes));
  };

  const handleRouteAdjustment = (leg, nextAdjustment) => {
    const adjustment = normalizeRouteAdjustment(nextAdjustment);
    if (adjustment.compensationKM <= 0 && !adjustment.compensationReason && !adjustment.viaName && !adjustment.viaReason) {
      legAdjustmentStore.delete(leg.adjustmentKey);
    } else {
      legAdjustmentStore.set(leg.adjustmentKey, adjustment);
    }
    const routeChanged = nextAdjustment?.routeChanged === true;
    const updated = renderCurrentItinerary(routeChanged);
    if (mapPanel._leafletMap && !routeChanged) {
      setupRouteResultLegSelection(mapPanel);
    }
    const status = routeItineraryStatus(updated);
    setStatus(status.message, status.tone);
  };

  mapPanel._onRoadRouteCalculated = (updatedCalculation) => {
    currentCalculation = updatedCalculation;
    renderItineraryResult(result, updatedCalculation, handleRouteAdjustment, viaPoints);
    updateClaimSummary(updatedCalculation);
    setupRouteResultLegSelection(mapPanel);
  };

  const renderCurrentItinerary = (refreshMap = false) => {
    const calculation = calculateItinerary(selectedStops, Number($("[name='rate']", form).value || rate), view, legAdjustmentStore);
    currentCalculation = calculation;
    renderItineraryResult(result, calculation, handleRouteAdjustment, viaPoints);
    updateClaimSummary(calculation);
    if (refreshMap) {
      renderRouteMap(mapPanel, calculation, view);
    }
    return calculation;
  };

  form.addEventListener("submit", (event) => {
    event.preventDefault();
    const calculation = renderCurrentItinerary(true);
    recordReceipt(
      "Calculo kilometraje",
      `${calculation.stops.join(" -> ")} - base ${formatPoints(calculation.totalBaseKM)} km - liquidable ${formatPoints(calculation.totalKM)} km`,
      "dietas",
    );
    const status = routeItineraryStatus(calculation);
    setStatus(status.message, status.tone);
  });

  const initial = renderCurrentItinerary(false);
  setupDietasDailyClaimPanel(claimPanel, allowanceTypes, () => updateClaimSummary());
  $("[data-send-dietas-claim]", claimPanel)?.addEventListener("click", () => {
    const values = readDietasDailyClaim(claimPanel, allowanceTypes);
    const validation = validateDietasDailyClaim(currentCalculation, values);
    if (validation.error) {
      setStatus(validation.error, "error");
      return;
    }
    const receipt = `DIE-${values.travelDate.replaceAll("-", "")}-${String(Date.now()).slice(-5)}`;
    renderDietasClaimSubmitted(claimPanel, currentCalculation, values, receipt);
    recordReceipt(
      "Solicitud dieta enviada",
      `${receipt} - ${values.travelDate} - ${formatPoints(currentCalculation.totalKM)} km - ${formatCurrency(dietasClaimTotal(currentCalculation, values))}`,
      "dietas",
    );
    setStatus("Solicitud enviada a validacion de jefe de servicio", "ready");
  });

  routeEditor.append(form, result);
  routeMapColumn.append(mapPanel);
  routeWorkspace.append(routeEditor, routeMapColumn);
  const content = [header, claimPanel, routeWorkspace];
  if (isAdminSession()) {
    content.push(routeMatrixSourceNote(matrix));
  }
  panel.append(...content);
  window.requestAnimationFrame(() => renderRouteMap(mapPanel, initial, view));
  return panel;
}

function normalizeAllowanceTypes(items) {
  const source = Array.isArray(items) && items.length
    ? items
    : [
      { id: "no_dieta", label: "Sin dieta", amount: 0, rule: "Sin manutencion solicitada." },
      { id: "media_dieta", label: "Media dieta", amount: 26.67, rule: "Importe editable pendiente de politica vigente." },
      { id: "dieta_completa", label: "Dieta completa", amount: 53.34, rule: "Importe editable pendiente de politica vigente." },
    ];
  return source.map((item) => ({
    id: String(item.id || item.label || "no_dieta"),
    label: String(item.label || item.id || "Sin dieta"),
    amount: Number(item.amount || 0),
    rule: String(item.rule || "Politica pendiente de validacion."),
    requiresMealProof: item.requires_meal_proof === true,
  }));
}

function dietasDailyClaimPanel(allowanceTypes, policy) {
  const panel = document.createElement("section");
  panel.className = "dietas-claim-panel";
  const defaultAllowance = allowanceTypes.find((item) => item.id === "media_dieta") || allowanceTypes[0];
  panel.innerHTML = `
    <div class="dietas-day-grid">
      <label>Dia del desplazamiento
        <input name="travel_date" type="date" value="${todayISODate()}" required>
      </label>
      <label>Empleado
        <input name="employee" value="Empleado demo" autocomplete="off">
      </label>
      <label>Unidad
        <input name="unit" value="Servicio provincial" autocomplete="off">
      </label>
      <label>Motivo del viaje
        <input name="purpose" value="Comision de servicio" autocomplete="off">
      </label>
    </div>
    <div class="dietas-expense-grid">
      <label>Manutencion
        <select name="allowance_type"></select>
      </label>
      <label>Importe manutencion
        <input name="allowance_amount" type="number" min="0" step="0.01">
      </label>
      <label>Alojamiento
        <input name="lodging_amount" type="number" min="0" step="0.01" value="0">
      </label>
      <label>Justificante alojamiento
        <input name="lodging_reference" placeholder="Factura/CSV si procede" autocomplete="off">
      </label>
    </div>
    <div class="dietas-policy-line">
      <span class="status-chip chip-amber">Politica demo</span>
      <span data-allowance-rule></span>
    </div>
    <div class="dietas-submit-strip">
      <div class="dietas-summary" aria-live="polite">
        <span>Km</span><b data-dietas-summary="km">0,0 km</b>
        <span>Kilometraje</span><b data-dietas-summary="mileage">0,00 €</b>
        <span>Dietas/gastos</span><b data-dietas-summary="allowances">0,00 €</b>
        <span>Total</span><b data-dietas-summary="total">0,00 €</b>
      </div>
      <div class="dietas-approval-chain" aria-label="Cadena de validacion">
        <span class="is-current">Jefe servicio</span>
        <span>Tecnico RRHH</span>
      </div>
      <button type="button" class="primary-action" data-send-dietas-claim>Enviar a validar</button>
    </div>
  `;
  const select = $("[name='allowance_type']", panel);
  allowanceTypes.forEach((item) => {
    const option = document.createElement("option");
    option.value = item.id;
    option.textContent = item.label;
    select.append(option);
  });
  select.value = defaultAllowance?.id || allowanceTypes[0]?.id || "no_dieta";
  $("[name='allowance_amount']", panel).value = String(defaultAllowance?.amount || 0);
  $("[data-allowance-rule]", panel).textContent = defaultAllowance?.rule || policy?.demo_notice || "Importes editables pendientes de validacion.";
  return panel;
}

function setupDietasDailyClaimPanel(panel, allowanceTypes, onChange) {
  const select = $("[name='allowance_type']", panel);
  select?.addEventListener("change", () => {
    const selected = allowanceTypes.find((item) => item.id === select.value) || allowanceTypes[0];
    $("[name='allowance_amount']", panel).value = String(selected?.amount || 0);
    $("[data-allowance-rule]", panel).textContent = selected?.rule || "Politica pendiente de validacion.";
    onChange();
  });
  $$("input", panel).forEach((input) => {
    input.addEventListener("input", onChange);
    input.addEventListener("change", onChange);
  });
}

function readDietasDailyClaim(panel, allowanceTypes) {
  const selectedID = $("[name='allowance_type']", panel)?.value || "no_dieta";
  const allowance = allowanceTypes.find((item) => item.id === selectedID) || allowanceTypes[0] || {};
  return {
    travelDate: $("[name='travel_date']", panel)?.value || "",
    employee: $("[name='employee']", panel)?.value || "",
    unit: $("[name='unit']", panel)?.value || "",
    purpose: $("[name='purpose']", panel)?.value || "",
    allowanceType: selectedID,
    allowanceLabel: allowance.label || selectedID,
    allowanceAmount: Number($("[name='allowance_amount']", panel)?.value || 0),
    lodgingAmount: Number($("[name='lodging_amount']", panel)?.value || 0),
    lodgingReference: $("[name='lodging_reference']", panel)?.value || "",
  };
}

function dietasClaimTotal(calculation, values) {
  return Number(calculation?.mileageAmount || 0) + Number(values?.allowanceAmount || 0) + Number(values?.lodgingAmount || 0);
}

function renderDietasClaimSummary(panel, calculation, values) {
  const mileage = Number(calculation?.mileageAmount || 0);
  const allowances = Number(values?.allowanceAmount || 0) + Number(values?.lodgingAmount || 0);
  const setSummary = (key, text) => {
    const node = $(`[data-dietas-summary="${key}"]`, panel);
    if (node) node.textContent = text;
  };
  setSummary("km", `${formatPoints(calculation?.totalKM || 0)} km`);
  setSummary("mileage", formatCurrency(mileage));
  setSummary("allowances", formatCurrency(allowances));
  setSummary("total", formatCurrency(mileage + allowances));
}

function validateDietasDailyClaim(calculation, values) {
  if (!values.travelDate) return { error: "Selecciona el dia del desplazamiento." };
  if (!calculation || calculation.legs.length === 0 || calculation.stops.length < 2) return { error: "Anade al menos origen y destino." };
  if (calculation.missing?.length) return { error: "Hay tramos pendientes en matriz. Corrige la ruta antes de enviar." };
  if (calculation.compensationIssues?.length) return { error: "Hay km de compensacion sin motivo." };
  if (calculation.routeViaIssues?.length) return { error: "Hay rutas alternativas sin motivo." };
  if (values.lodgingAmount > 0 && !values.lodgingReference.trim()) return { error: "Indica justificante o referencia del alojamiento." };
  return { ok: true };
}

function renderDietasClaimSubmitted(panel, calculation, values, receipt) {
  const chain = $(".dietas-approval-chain", panel);
  if (chain) {
    chain.innerHTML = `
      <span class="is-done">Solicitante</span>
      <span class="is-current">Jefe servicio</span>
      <span>Tecnico RRHH</span>
    `;
  }
  const button = $("[data-send-dietas-claim]", panel);
  if (button) {
    button.textContent = "Enviado";
    button.disabled = true;
  }
  const line = $(".dietas-policy-line", panel);
  if (line) {
    line.innerHTML = `<span class="status-chip chip-green">Enviado</span><span>${receipt} - ${values.travelDate} - pendiente de validacion por jefe de servicio.</span>`;
  }
  renderDietasClaimSummary(panel, calculation, values);
}

function routeStopRow(value, index, total, points, onChange, onRemove) {
  const row = document.createElement("div");
  row.className = "route-stop-row";
  const label = document.createElement("label");
  const labelText = index === 0 ? "Salida" : (index === total - 1 ? "Destino final" : `Parada ${index}`);
  label.innerHTML = `<span></span>`;
  $("span", label).textContent = labelText;
  const select = document.createElement("select");
  select.name = `stop_${index}`;
  select.required = true;
  const placeholder = document.createElement("option");
  placeholder.value = "";
  placeholder.textContent = "Seleccionar localidad";
  select.append(placeholder);
  points.forEach((point) => {
    const option = document.createElement("option");
    option.value = point.name;
    option.textContent = point.municipality_name && point.municipality_name !== point.name
      ? `${point.name} (${point.municipality_name})`
      : point.name;
    select.append(option);
  });
  if (points.some((point) => point.name === value)) {
    select.value = value;
  }
  select.addEventListener("change", () => onChange(select.value));
  label.append(select);
  const remove = document.createElement("button");
  remove.type = "button";
  remove.className = "row-action is-quiet";
  remove.textContent = "Quitar";
  remove.disabled = total <= 2;
  remove.addEventListener("click", onRemove);
  row.append(label, remove);
  return row;
}

function matrixMetric(label, value, tone) {
  const metric = document.createElement("article");
  metric.className = "screen-counter";
  metric.dataset.tone = tone;
  metric.innerHTML = `<span></span><b></b>`;
  $("span", metric).textContent = label;
  $("b", metric).textContent = formatCount(value);
  return metric;
}

function normalizeRouteName(value) {
  return String(value || "")
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLowerCase()
    .replace(/\b(la|el|los|las)\b/g, "")
    .replace(/[^a-z0-9]+/g, " ")
    .trim();
}

function routePairKey(from, to) {
  return `${normalizeRouteName(from)}>${normalizeRouteName(to)}`;
}

function routeLegAdjustmentKey(index, from, to) {
  return `${index}:${routePairKey(from, to)}`;
}

function normalizeRouteAdjustment(value) {
  const compensationKM = Number(value?.compensationKM ?? value?.km ?? 0);
  return {
    compensationKM: Number.isFinite(compensationKM) && compensationKM > 0 ? compensationKM : 0,
    compensationReason: String(value?.compensationReason ?? value?.reason ?? "").trim(),
    viaName: String(value?.viaName ?? "").trim(),
    viaReason: String(value?.viaReason ?? "").trim(),
  };
}

function routeAdjustmentFromStore(store, key) {
  if (store instanceof Map) {
    return normalizeRouteAdjustment(store.get(key));
  }
  return normalizeRouteAdjustment(null);
}

function routePairLookup(view) {
  const pairs = view.workspace?.province_route_pairs || view.workspace?.province_routes || [];
  const map = new Map();
  pairs.forEach((pair) => {
    map.set(routePairKey(pair.from, pair.to), pair);
  });
  return map;
}

function routeSelectableViaPoints(points) {
  return (points || [])
    .filter((point) =>
      point?.name
      && Number.isFinite(Number(point.lat))
      && Number.isFinite(Number(point.lon))
      && !(Number(point.lat) === 0 && Number(point.lon) === 0),
    )
    .slice()
    .sort((a, b) => String(a.name).localeCompare(String(b.name), "es"));
}

function routeSelectableStops(view) {
  const routePoints = view.workspace?.province_route_points || [];
  const localities = view.workspace?.province_localities || [];
  const byName = new Map();
  localities.forEach((item) => {
    if (!item?.name) return;
    byName.set(normalizeRouteName(item.name), {
      code: item.ine_code || item.code || "",
      name: item.name,
      kind: item.kind || "municipio",
      municipality_name: item.name,
      source: item.source || "INE/CNIG",
    });
  });
  routePoints.forEach((point) => {
    if (!point?.name) return;
    byName.set(normalizeRouteName(point.name), {
      ...byName.get(normalizeRouteName(point.name)),
      ...point,
      code: point.code || point.ine_code || "",
      name: point.name,
      municipality_name: point.municipality_name || point.name,
    });
  });
  return Array.from(byName.values())
    .filter((point) => point.name)
    .sort((a, b) => String(a.name).localeCompare(String(b.name), "es"));
}

function calculateItinerary(rawStops, rate, view, adjustmentStore = new Map()) {
  const stops = Array.isArray(rawStops)
    ? rawStops.map((item) => String(item || "").trim()).filter(Boolean)
    : String(rawStops || "")
      .split(/\s*(?:->|;|\n)\s*/g)
      .map((item) => item.trim())
      .filter(Boolean);
  const pairs = routePairLookup(view);
  const legs = [];
  const missing = [];
  let totalBaseKM = 0;
  let totalCompensationKM = 0;
  let totalMinutes = 0;
  for (let index = 0; index < stops.length - 1; index += 1) {
    const from = stops[index];
    const to = stops[index + 1];
    const pair = pairs.get(routePairKey(from, to));
    const adjustmentKey = routeLegAdjustmentKey(index, from, to);
    const adjustment = routeAdjustmentFromStore(adjustmentStore, adjustmentKey);
    if (!pair) {
      missing.push(`${from} -> ${to}`);
      legs.push({
        index,
        from,
        to,
        adjustmentKey,
        viaName: adjustment.viaName,
        viaReason: adjustment.viaReason,
        distanceKM: 0,
        compensationKM: adjustment.compensationKM,
        compensationReason: adjustment.compensationReason,
        liquidableKM: 0,
        minutes: 0,
        state: "Pendiente en matriz",
        color: routeLegColor(index),
      });
      continue;
    }
    const distanceKM = Number(pair.distance_km ?? pair.km_one_way ?? 0);
    const minutes = Number(pair.duration_minutes ?? pair.estimated_minutes ?? 0);
    const liquidableKM = distanceKM + adjustment.compensationKM;
    totalBaseKM += distanceKM;
    totalCompensationKM += adjustment.compensationKM;
    totalMinutes += minutes;
    legs.push({
      index,
      from,
      to,
      adjustmentKey,
      viaName: adjustment.viaName,
      viaReason: adjustment.viaReason,
      distanceKM,
      compensationKM: adjustment.compensationKM,
      compensationReason: adjustment.compensationReason,
      liquidableKM,
      minutes,
      state: pair.state || "Matriz cargada",
      color: routeLegColor(index),
    });
  }
  const totalKM = totalBaseKM + totalCompensationKM;
  return {
    stops,
    legs,
    missing,
    totalBaseKM,
    totalCompensationKM,
    totalKM,
    totalMinutes,
    rate,
    mileageAmount: totalKM * rate,
    compensationIssues: legs.filter((leg) => leg.compensationKM > 0 && !leg.compensationReason),
    routeViaIssues: legs.filter((leg) => leg.viaName && !leg.viaReason),
  };
}

function routeItineraryStatus(calculation) {
  if (calculation.missing.length) {
    return {
      message: "Itinerario con tramos pendientes en matriz",
      tone: "warning",
    };
  }
  if ((calculation.compensationIssues || []).length) {
    return {
      message: "Hay kilometros de compensacion pendientes de motivo",
      tone: "warning",
    };
  }
  if ((calculation.routeViaIssues || []).length) {
    return {
      message: "Hay rutas alternativas pendientes de motivo",
      tone: "warning",
    };
  }
  if (calculation.legs.some((leg) => leg.viaName)) {
    return {
      message: "Itinerario calculado con ruta alternativa justificada",
      tone: "ready",
    };
  }
  if (calculation.totalCompensationKM > 0) {
    return {
      message: "Itinerario calculado con compensacion justificada por tramo",
      tone: "ready",
    };
  }
  return {
    message: "Itinerario calculado con matriz",
    tone: "ready",
  };
}

const ROUTE_LEG_COLORS = ["#1d4f91", "#0f766e", "#b45309", "#7c3aed", "#be123c", "#087990", "#4b5563"];

function routeLegColor(index) {
  return ROUTE_LEG_COLORS[index % ROUTE_LEG_COLORS.length];
}

function routeMapPanel() {
  const panel = document.createElement("section");
  panel.className = "route-map-card";
  panel.innerHTML = `
    <div class="route-map-head">
      <div>
        <h3>Mapa del recorrido</h3>
        <span class="small-text">Vista interna con geometria de carretera OSRM; si el motor propio no responde se muestra un croquis no liquidable.</span>
      </div>
      <div class="route-map-actions"></div>
    </div>
    <div class="route-map-canvas" role="img" aria-label="Mapa del recorrido seleccionado" hidden></div>
    <div class="route-leg-legend" hidden></div>
    <div class="route-alternatives" hidden></div>
    <p class="route-map-fallback" hidden></p>
  `;
  return panel;
}

function openStreetMapDirectionsURL(coords) {
  const valid = (coords || []).filter((coord) =>
    Number.isFinite(Number(coord.lat)) && Number.isFinite(Number(coord.lon)),
  );
  if (valid.length < 2) return "";
  const route = valid.map((coord) => `${Number(coord.lat).toFixed(6)},${Number(coord.lon).toFixed(6)}`).join(";");
  return `https://www.openstreetmap.org/directions?engine=fossgis_osrm_car&route=${encodeURIComponent(route)}`;
}

function addOpenStreetMapButton(actions, coords) {
  const href = openStreetMapDirectionsURL(coords);
  if (!href) return;
  const link = document.createElement("a");
  link.className = "table-action route-osm-link";
  link.href = href;
  link.target = "_blank";
  link.rel = "noopener";
  link.textContent = "Abrir OSM completo";
  actions.append(link);
}

function routePointCoordinateLookup(view) {
  const map = new Map();
  (view.workspace?.province_route_points || []).forEach((point) => {
    const lat = Number(point.lat);
    const lon = Number(point.lon);
    if (!Number.isFinite(lat) || !Number.isFinite(lon) || (lat === 0 && lon === 0)) {
      return;
    }
    map.set(normalizeRouteName(point.name), { lat, lon, name: point.name });
  });
  return map;
}

function routeCoordinatesForStops(stops, view) {
  const lookup = routePointCoordinateLookup(view);
  const coords = [];
  const missing = [];
  stops.forEach((stop) => {
    const coord = lookup.get(normalizeRouteName(stop));
    if (coord) {
      coords.push(coord);
      return;
    }
    missing.push(stop);
  });
  return { coords, missing };
}

function pushRouteCoordinate(coords, coord, label) {
  if (!coord) return false;
  const next = { ...coord, name: label || coord.name };
  const previous = coords[coords.length - 1];
  if (previous && previous.lat === next.lat && previous.lon === next.lon) {
    return false;
  }
  coords.push(next);
  return true;
}

function routeCoordinatesForCalculation(calculation, view) {
  const lookup = routePointCoordinateLookup(view);
  const coords = [];
  const missing = [];
  const segmentMeta = [];
  if (!calculation?.legs?.length) {
    const fallback = routeCoordinatesForStops(calculation?.stops || [], view);
    return { ...fallback, segmentMeta };
  }
  const addPoint = (name, label, legIndex) => {
    const coord = lookup.get(normalizeRouteName(name));
    if (!coord) {
      missing.push(name);
      return false;
    }
    const hadPrevious = coords.length > 0;
    const added = pushRouteCoordinate(coords, coord, label);
    if (added && hadPrevious) {
      segmentMeta.push({ legIndex });
    }
    return added;
  };
  calculation.legs.forEach((leg, index) => {
    if (index === 0) {
      addPoint(leg.from, leg.from, leg.index);
    }
    if (leg.viaName) {
      addPoint(leg.viaName, `Paso ${leg.index + 1}: ${leg.viaName}`, leg.index);
    }
    addPoint(leg.to, leg.to, leg.index);
  });
  return { coords, missing, segmentMeta };
}

function aggregateRoadLegsByIndex(roadLegs) {
  const map = new Map();
  (roadLegs || []).forEach((leg) => {
    const current = map.get(leg.index) || {
      index: leg.index,
      latLngs: [],
      distanceKM: 0,
      durationMin: 0,
      color: leg.color,
    };
    current.distanceKM += Number(leg.distanceKM || 0);
    current.durationMin += Number(leg.durationMin || 0);
    current.latLngs = current.latLngs.concat(leg.latLngs || []);
    map.set(leg.index, current);
  });
  return map;
}

function nearestRoutePointIndex(latLngs, coord, minIndex = 0) {
  const targetLat = Number(coord?.lat);
  const targetLon = Number(coord?.lon);
  if (!Number.isFinite(targetLat) || !Number.isFinite(targetLon) || !latLngs.length) {
    return Math.max(0, Math.min(minIndex, latLngs.length - 1));
  }
  let bestIndex = Math.max(0, Math.min(minIndex, latLngs.length - 1));
  let bestScore = Number.POSITIVE_INFINITY;
  for (let index = bestIndex; index < latLngs.length; index += 1) {
    const [lat, lon] = latLngs[index];
    const score = (Number(lat) - targetLat) ** 2 + (Number(lon) - targetLon) ** 2;
    if (score < bestScore) {
      bestScore = score;
      bestIndex = index;
    }
  }
  return bestIndex;
}

function splitRouteLatLngsByCoordinates(latLngs, coords, segmentMeta = [], sourceLegs = []) {
  if (!latLngs.length || !Array.isArray(coords) || coords.length < 2) return sourceLegs;
  const waypointIndexes = [];
  let minIndex = 0;
  coords.forEach((coord) => {
    const nextIndex = nearestRoutePointIndex(latLngs, coord, minIndex);
    waypointIndexes.push(nextIndex);
    minIndex = nextIndex;
  });
  return coords.slice(0, -1).map((coord, legIndex) => {
    const nextCoord = coords[legIndex + 1];
    const start = Math.min(waypointIndexes[legIndex] ?? 0, waypointIndexes[legIndex + 1] ?? latLngs.length - 1);
    const end = Math.max(waypointIndexes[legIndex] ?? 0, waypointIndexes[legIndex + 1] ?? latLngs.length - 1);
    const fallbackLatLngs = [
      [Number(coord.lat), Number(coord.lon)],
      [Number(nextCoord.lat), Number(nextCoord.lon)],
    ].filter(([lat, lon]) => Number.isFinite(lat) && Number.isFinite(lon));
    const slice = latLngs.slice(start, end + 1);
    const mappedIndex = Number(segmentMeta[legIndex]?.legIndex ?? legIndex);
    const routeLegIndex = Number.isFinite(mappedIndex) ? mappedIndex : legIndex;
    const source = sourceLegs[legIndex] || {};
    return {
      ...source,
      index: routeLegIndex,
      segmentIndex: legIndex,
      latLngs: slice.length >= 2 ? slice : fallbackLatLngs,
      color: source.color || routeLegColor(routeLegIndex),
    };
  });
}

function routeGeometryFromOSRM(payload, segmentMeta = [], coords = []) {
  const routes = (payload?.routes || []).map((route, index) => {
    const coordinates = route?.geometry?.coordinates || [];
    const latLngs = coordinates
      .map((point) => [Number(point[1]), Number(point[0])])
      .filter(([lat, lon]) => Number.isFinite(lat) && Number.isFinite(lon));
    const legs = (route?.legs || []).map((leg, legIndex) => {
      const legSteps = leg.steps || [];
      const legLatLngs = legSteps.flatMap((step) =>
        (step?.geometry?.coordinates || [])
          .map((point) => [Number(point[1]), Number(point[0])])
          .filter(([lat, lon]) => Number.isFinite(lat) && Number.isFinite(lon)),
      );
      const mappedIndex = Number(segmentMeta[legIndex]?.legIndex ?? legIndex);
      const routeLegIndex = Number.isFinite(mappedIndex) ? mappedIndex : legIndex;
      return {
        index: routeLegIndex,
        segmentIndex: legIndex,
        latLngs: legLatLngs.length >= 2 ? legLatLngs : [],
        distanceKM: Number(leg.distance || 0) / 1000,
        durationMin: Math.round(Number(leg.duration || 0) / 60),
        color: routeLegColor(routeLegIndex),
      };
    });
    const drawableLegs = legs.some((leg) => leg.latLngs.length >= 2)
      ? legs.map((leg) => (leg.latLngs.length >= 2 ? leg : splitRouteLatLngsByCoordinates(latLngs, coords, segmentMeta, legs)[leg.segmentIndex] || leg))
      : splitRouteLatLngsByCoordinates(latLngs, coords, segmentMeta, legs);
    return {
      index,
      label: index === 0 ? "Ruta recomendada" : `Alternativa ${index}`,
      latLngs,
      legs: drawableLegs,
      distanceKM: Number(route.distance || 0) / 1000,
      durationMin: Math.round(Number(route.duration || 0) / 60),
    };
  }).filter((route) => route.latLngs.length >= 2);
  if (payload?.code !== "Ok" || !routes.length) {
    throw new Error(payload?.message || payload?.code || "OSRM no ha devuelto una ruta por carretera.");
  }
  return {
    routes,
    dataVersion: payload.data_version || "",
  };
}

function applyRoadRouteToCalculation(calculation, route) {
  if (!calculation || !route?.legs?.length) return calculation;
  const roadByIndex = aggregateRoadLegsByIndex(route.legs);
  let totalBaseKM = 0;
  let totalCompensationKM = 0;
  let totalMinutes = 0;
  calculation.legs.forEach((leg) => {
    const roadLeg = roadByIndex.get(leg.index);
    if (roadLeg && roadLeg.distanceKM > 0) {
      leg.distanceKM = roadLeg.distanceKM;
      leg.minutes = roadLeg.durationMin;
      leg.state = "OSRM interno";
    }
    leg.liquidableKM = Number(leg.distanceKM || 0) + Number(leg.compensationKM || 0);
    totalBaseKM += Number(leg.distanceKM || 0);
    totalCompensationKM += Number(leg.compensationKM || 0);
    totalMinutes += Number(leg.minutes || 0);
  });
  calculation.totalBaseKM = totalBaseKM;
  calculation.totalCompensationKM = totalCompensationKM;
  calculation.totalKM = totalBaseKM + totalCompensationKM;
  calculation.totalMinutes = totalMinutes;
  calculation.mileageAmount = calculation.totalKM * Number(calculation.rate || 0);
  calculation.missing = calculation.legs
    .filter((leg) => Number(leg.distanceKM || 0) <= 0)
    .map((leg) => `${leg.from} -> ${leg.to}`);
  calculation.compensationIssues = calculation.legs.filter((leg) => leg.compensationKM > 0 && !leg.compensationReason);
  calculation.routeViaIssues = calculation.legs.filter((leg) => leg.viaName && !leg.viaReason);
  return calculation;
}

async function fetchRoadRouteGeometry(coords, segmentMeta = []) {
  const payload = await getData(DIETAS_ROAD_ROUTE_API, {
    method: "POST",
    headers: staffHeaders(),
    body: JSON.stringify({
      coordinates: coords.map((coord) => ({
        lat: coord.lat,
        lon: coord.lon,
        name: coord.name,
      })),
      alternatives: 3,
    }),
  });
  return routeGeometryFromOSRM(payload, segmentMeta, coords);
}

function destroyRouteMap(panel) {
  panel._routeLine = null;
  panel._localRouteLines = [];
  panel._activeRouteBounds = null;
  if (panel._leafletMap) {
    panel._leafletMap.remove();
    panel._leafletMap = null;
  }
}

function routeBoundsForLocalMap(coords, routes = []) {
  const points = [];
  (routes[0]?.latLngs || []).forEach(([lat, lon]) => points.push({ lat: Number(lat), lon: Number(lon) }));
  (coords || []).forEach((coord) => points.push({ lat: Number(coord.lat), lon: Number(coord.lon) }));
  const valid = points.filter((point) => Number.isFinite(point.lat) && Number.isFinite(point.lon));
  if (!valid.length) return null;
  let minLat = Math.min(...valid.map((point) => point.lat));
  let maxLat = Math.max(...valid.map((point) => point.lat));
  let minLon = Math.min(...valid.map((point) => point.lon));
  let maxLon = Math.max(...valid.map((point) => point.lon));
  if (minLat === maxLat) {
    minLat -= 0.05;
    maxLat += 0.05;
  }
  if (minLon === maxLon) {
    minLon -= 0.05;
    maxLon += 0.05;
  }
  return { minLat, maxLat, minLon, maxLon };
}

function projectLocalRoutePoint(point, bounds, width, height, padding) {
  const lat = Array.isArray(point) ? Number(point[0]) : Number(point.lat);
  const lon = Array.isArray(point) ? Number(point[1]) : Number(point.lon);
  const x = padding + ((lon - bounds.minLon) / (bounds.maxLon - bounds.minLon)) * (width - padding * 2);
  const y = padding + ((bounds.maxLat - lat) / (bounds.maxLat - bounds.minLat)) * (height - padding * 2);
  return { x, y, lat, lon };
}

function renderLocalRouteMap(panel, canvas, legend, alternatives, fallback, actions, calculation, coords, routes = [], message = "") {
  const svgNS = "http://www.w3.org/2000/svg";
  const width = 920;
  const height = 330;
  const padding = 34;
  const bounds = routeBoundsForLocalMap(coords, routes);
  canvas.hidden = false;
  canvas.classList.add("is-local-route");
  canvas.replaceChildren();
  alternatives.hidden = true;
  alternatives.replaceChildren();
  actions.replaceChildren();
  panel._localRouteLines = [];
  if (!bounds) {
    fallback.hidden = false;
    fallback.textContent = message || "No hay coordenadas suficientes para pintar el croquis local.";
    return;
  }

  const allButton = document.createElement("button");
  allButton.type = "button";
  allButton.className = "table-action route-leg-all is-active";
  allButton.textContent = "Todos";
  allButton.addEventListener("click", () => resetRouteLegHighlight(panel));
  actions.append(allButton);
  addOpenStreetMapButton(actions, coords);

  const svg = document.createElementNS(svgNS, "svg");
  svg.setAttribute("viewBox", `0 0 ${width} ${height}`);
  svg.setAttribute("role", "img");
  svg.setAttribute("aria-label", "Croquis local del recorrido");

  const background = document.createElementNS(svgNS, "rect");
  background.setAttribute("x", "0");
  background.setAttribute("y", "0");
  background.setAttribute("width", String(width));
  background.setAttribute("height", String(height));
  background.setAttribute("fill", "#eef4f7");
  svg.append(background);

  for (let index = 0; index < 8; index += 1) {
    const vertical = document.createElementNS(svgNS, "line");
    vertical.setAttribute("x1", String(padding + ((width - padding * 2) / 7) * index));
    vertical.setAttribute("x2", vertical.getAttribute("x1"));
    vertical.setAttribute("y1", String(padding));
    vertical.setAttribute("y2", String(height - padding));
    vertical.setAttribute("stroke", "#d7e2ea");
    vertical.setAttribute("stroke-width", "1");
    svg.append(vertical);
    const horizontal = document.createElementNS(svgNS, "line");
    horizontal.setAttribute("x1", String(padding));
    horizontal.setAttribute("x2", String(width - padding));
    horizontal.setAttribute("y1", String(padding + ((height - padding * 2) / 7) * index));
    horizontal.setAttribute("y2", horizontal.getAttribute("y1"));
    horizontal.setAttribute("stroke", "#d7e2ea");
    horizontal.setAttribute("stroke-width", "1");
    svg.append(horizontal);
  }

  const label = document.createElementNS(svgNS, "text");
  label.setAttribute("x", String(width - padding));
  label.setAttribute("y", "24");
  label.setAttribute("text-anchor", "end");
  label.setAttribute("fill", "#607080");
  label.setAttribute("font-size", "13");
  label.textContent = routes.length ? "Ruta OSRM interna" : "Croquis local no liquidable";
  svg.append(label);

  const drawPolyline = (latLngs, index, color, opacity = 0.9) => {
    const points = (latLngs || [])
      .map((point) => projectLocalRoutePoint(point, bounds, width, height, padding))
      .filter((point) => Number.isFinite(point.x) && Number.isFinite(point.y));
    if (points.length < 2) return;
    const polyline = document.createElementNS(svgNS, "polyline");
    polyline.setAttribute("points", points.map((point) => `${point.x.toFixed(1)},${point.y.toFixed(1)}`).join(" "));
    polyline.setAttribute("fill", "none");
    polyline.setAttribute("stroke", color);
    polyline.setAttribute("stroke-width", "5");
    polyline.setAttribute("stroke-linecap", "round");
    polyline.setAttribute("stroke-linejoin", "round");
    polyline.setAttribute("opacity", String(opacity));
    polyline.dataset.routeLegIndex = String(index);
    polyline.addEventListener("click", () => highlightRouteLeg(panel, index));
    svg.append(polyline);
    panel._localRouteLines.push({ index, line: polyline, color });
  };

  const roadLegs = routes[0]?.legs || [];
  if (roadLegs.some((leg) => leg.latLngs?.length >= 2)) {
    roadLegs.filter((leg) => leg.latLngs.length >= 2).forEach((leg) => {
      drawPolyline(leg.latLngs, leg.index, leg.color || routeLegColor(leg.index), 0.92);
    });
  } else {
    const coordByName = new Map((coords || []).map((coord) => [normalizeRouteName(coord.name), coord]));
    (calculation?.legs || []).forEach((leg, index) => {
      const legCoords = [
        coordByName.get(normalizeRouteName(leg.from)),
        leg.viaName ? coordByName.get(normalizeRouteName(leg.viaName)) : null,
        coordByName.get(normalizeRouteName(leg.to)),
      ].filter(Boolean);
      drawPolyline(legCoords, index, leg.color || routeLegColor(index), 0.78);
    });
    if (!panel._localRouteLines.length) {
      drawPolyline(coords, 0, "#1d4f91", 0.78);
    }
  }

  (coords || []).forEach((coord, index) => {
    const point = projectLocalRoutePoint(coord, bounds, width, height, padding);
    const marker = document.createElementNS(svgNS, "circle");
    marker.setAttribute("cx", point.x.toFixed(1));
    marker.setAttribute("cy", point.y.toFixed(1));
    marker.setAttribute("r", "7");
    marker.setAttribute("fill", "#fff");
    marker.setAttribute("stroke", "#13202d");
    marker.setAttribute("stroke-width", "2");
    svg.append(marker);
    const number = document.createElementNS(svgNS, "text");
    number.setAttribute("x", point.x.toFixed(1));
    number.setAttribute("y", String(point.y + 4));
    number.setAttribute("text-anchor", "middle");
    number.setAttribute("fill", "#13202d");
    number.setAttribute("font-size", "10");
    number.setAttribute("font-weight", "800");
    number.textContent = String(index + 1);
    svg.append(number);
    const name = document.createElementNS(svgNS, "text");
    name.setAttribute("x", String(Math.min(width - padding, point.x + 11)));
    name.setAttribute("y", String(point.y - 9));
    name.setAttribute("fill", "#13202d");
    name.setAttribute("font-size", "12");
    name.setAttribute("font-weight", "700");
    name.textContent = coord.name || `Punto ${index + 1}`;
    svg.append(name);
  });

  canvas.append(svg);
  fallback.hidden = false;
  fallback.textContent = message || (routes.length
    ? "Ruta calculada con OSRM interno. Se muestra en visor local porque Leaflet no esta disponible."
    : "OSRM interno no disponible. Se muestra croquis local solo para orientacion; no sustituye la ruta por carretera.");
  renderRouteLegLegend(legend, calculation?.legs || [], roadLegs, (legIndex) => highlightRouteLeg(panel, legIndex), () => resetRouteLegHighlight(panel));
  setupRouteResultLegSelection(panel);
}

async function renderRouteMap(panel, calculation, view) {
  const canvas = $(".route-map-canvas", panel);
  const legend = $(".route-leg-legend", panel);
  const alternatives = $(".route-alternatives", panel);
  const fallback = $(".route-map-fallback", panel);
  const actions = $(".route-map-actions", panel);
  const { coords, missing, segmentMeta } = routeCoordinatesForCalculation(calculation, view);
  const requestID = (panel._routeRequestID || 0) + 1;
  panel._routeRequestID = requestID;
  actions.replaceChildren();
  canvas.hidden = true;
  canvas.classList.remove("is-local-route");
  canvas.replaceChildren();
  legend.hidden = true;
  legend.replaceChildren();
  alternatives.hidden = true;
  alternatives.replaceChildren();
  fallback.hidden = true;
  fallback.textContent = "";
  destroyRouteMap(panel);

  if (coords.length < 2) {
    fallback.hidden = false;
    fallback.textContent = "No hay coordenadas suficientes para pintar el recorrido. Importa coordenadas NGMEP/municipales para las paradas o puntos de paso seleccionados.";
    return;
  }

  fallback.hidden = false;
  fallback.textContent = "Calculando ruta por carretera con el OSRM interno...";

  try {
    const roadRoute = await fetchRoadRouteGeometry(coords, segmentMeta);
    if (panel._routeRequestID !== requestID) {
      return;
    }
    applyRoadRouteToCalculation(calculation, roadRoute.routes[0]);
    if (typeof panel._onRoadRouteCalculated === "function") {
      panel._onRoadRouteCalculated(calculation, roadRoute.routes[0], roadRoute);
    }
    if (!window.L) {
      renderLocalRouteMap(
        panel,
        canvas,
        legend,
        alternatives,
        fallback,
        actions,
        calculation,
        coords,
        roadRoute.routes,
        `Ruta por carretera calculada con OSRM interno${roadRoute.dataVersion ? ` - grafo ${roadRoute.dataVersion}` : ""}. Visor local activo porque Leaflet no esta disponible.`,
      );
      return;
    }
    canvas.hidden = false;
    const fullRouteButton = document.createElement("button");
    fullRouteButton.type = "button";
    fullRouteButton.className = "table-action";
    fullRouteButton.textContent = "Todos";
    fullRouteButton.addEventListener("click", () => resetRouteLegHighlight(panel));
    actions.append(fullRouteButton);
    addOpenStreetMapButton(actions, coords);
    panel._leafletMap = window.L.map(canvas, { scrollWheelZoom: false });
    window.L.tileLayer("https://tile.openstreetmap.org/{z}/{x}/{y}.png", {
      maxZoom: 19,
      attribution: "&copy; OpenStreetMap contributors",
    }).addTo(panel._leafletMap);

    coords.forEach((coord, index) => {
      window.L.marker([coord.lat, coord.lon])
        .addTo(panel._leafletMap)
        .bindPopup(`${index + 1}. ${coord.name}`);
    });

    const source = roadRoute.dataVersion ? ` - grafo ${roadRoute.dataVersion}` : "";
    const drawAlternative = (route) => {
      clearRouteLines(panel);
      const drawableLegs = route.legs.filter((leg) => leg.latLngs.length >= 2);
      if (drawableLegs.length) {
        panel._routeLegLines = drawableLegs.map((leg) => {
          const line = window.L.polyline(leg.latLngs, {
            color: leg.color,
            weight: 5,
            opacity: 0.92,
          }).addTo(panel._leafletMap);
          line.on("click", () => highlightRouteLeg(panel, leg.index));
          return { index: leg.index, line, color: leg.color };
        });
      } else {
        panel._routeLine = window.L.polyline(route.latLngs, {
          color: route.index === 0 ? "#1d4f91" : "#0f766e",
          weight: 5,
          opacity: 0.9,
        }).addTo(panel._leafletMap);
      }
      panel._activeRouteBounds = window.L.latLngBounds(route.latLngs);
      panel._leafletMap.fitBounds(panel._activeRouteBounds.pad(0.12));
      fallback.textContent = `${route.label}: ${formatPoints(route.distanceKM)} km - ${formatCount(route.durationMin)} min${source}. ${route.index > 0 ? "Uso de alternativa: exige motivo por corte, obra, seguridad o instruccion del servicio." : "Distancia calculada con OSRM interno y pendiente de validacion/homologacion para liquidacion."}`;
      renderRouteLegLegend(legend, calculation.legs, route.legs, (legIndex) => highlightRouteLeg(panel, legIndex), () => resetRouteLegHighlight(panel));
      setupRouteResultLegSelection(panel);
    };
    renderRouteAlternativeControls(alternatives, roadRoute.routes, drawAlternative);
    drawAlternative(roadRoute.routes[0]);
  } catch (error) {
    if (panel._routeRequestID !== requestID) {
      return;
    }
    destroyRouteMap(panel);
    renderLocalRouteMap(
      panel,
      canvas,
      legend,
      alternatives,
      fallback,
      actions,
      calculation,
      coords,
      [],
      `No se ha podido calcular la ruta por carretera con el OSRM interno: ${error.message}. Se muestra croquis local no liquidable en el mismo apartado de rutas.`,
    );
  }

  if (missing.length) {
    fallback.textContent += ` Paradas o puntos de paso sin coordenadas todavia: ${missing.join(", ")}.`;
  }
}

function clearRouteLines(panel) {
  if (panel._routeLine) {
    panel._routeLine.remove();
    panel._routeLine = null;
  }
  (panel._routeLegLines || []).forEach((item) => item.line.remove());
  panel._routeLegLines = [];
}

function resetRouteLegHighlight(panel) {
  const scope = panel.closest(".route-matrix-panel") || document;
  (panel._routeLegLines || []).forEach((item) => {
    item.line.setStyle({
      color: item.color,
      weight: 5,
      opacity: 0.92,
    });
  });
  (panel._localRouteLines || []).forEach((item) => {
    item.line.setAttribute("stroke", item.color);
    item.line.setAttribute("stroke-width", "5");
    item.line.setAttribute("opacity", "0.9");
  });
  $$("[data-route-leg-index]", scope).forEach((row) => row.classList.remove("is-selected"));
  $$(".route-leg-legend button[data-route-leg-index]", scope).forEach((button) => button.classList.remove("is-active"));
  $$(".route-leg-all", scope).forEach((button) => button.classList.add("is-active"));
  if (panel._leafletMap && panel._activeRouteBounds) {
    panel._leafletMap.fitBounds(panel._activeRouteBounds.pad(0.12));
  }
}

function highlightRouteLeg(panel, legIndex) {
  const numericIndex = Number(legIndex);
  if (!Number.isFinite(numericIndex)) return;
  const scope = panel.closest(".route-matrix-panel") || document;
  (panel._routeLegLines || []).forEach((item) => {
    const active = item.index === numericIndex;
    item.line.setStyle({
      color: item.color,
      weight: active ? 9 : 5,
      opacity: active ? 1 : 0.42,
    });
    if (active) {
      item.line.bringToFront();
      panel._leafletMap.fitBounds(item.line.getBounds().pad(0.25));
    }
  });
  (panel._localRouteLines || []).forEach((item) => {
    const active = item.index === numericIndex;
    item.line.setAttribute("stroke-width", active ? "9" : "5");
    item.line.setAttribute("opacity", active ? "1" : "0.28");
  });
  $$("[data-route-leg-index]", scope).forEach((row) => {
    row.classList.toggle("is-selected", Number(row.dataset.routeLegIndex) === numericIndex);
  });
  $$(".route-leg-legend button[data-route-leg-index]", scope).forEach((button) => {
    button.classList.toggle("is-active", Number(button.dataset.routeLegIndex) === numericIndex);
  });
  $$(".route-leg-all", scope).forEach((button) => button.classList.remove("is-active"));
}

function setupRouteResultLegSelection(panel) {
  const routePanel = panel.closest(".route-matrix-panel");
  if (!routePanel) return;
  $$("[data-route-leg-index]", routePanel).forEach((row) => {
    if (row.dataset.routeSelectionBound === "true") return;
    row.dataset.routeSelectionBound = "true";
    row.addEventListener("click", () => highlightRouteLeg(panel, row.dataset.routeLegIndex));
    row.addEventListener("keydown", (event) => {
      if (event.key === "Enter" || event.key === " ") {
        event.preventDefault();
        highlightRouteLeg(panel, row.dataset.routeLegIndex);
      }
    });
  });
}

function renderRouteLegLegend(target, calculationLegs, roadLegs, onSelect, onReset) {
  target.hidden = false;
  target.replaceChildren();
  const head = document.createElement("div");
  head.className = "route-leg-legend-head";
  const title = document.createElement("strong");
  title.textContent = "Color por tramo";
  const allButton = document.createElement("button");
  allButton.type = "button";
  allButton.className = "route-leg-chip route-leg-all is-active";
  allButton.textContent = "Todos";
  allButton.addEventListener("click", () => onReset());
  head.append(title, allButton);
  target.append(head);
  const roadByIndex = aggregateRoadLegsByIndex(roadLegs);
  calculationLegs.forEach((leg, index) => {
    const roadLeg = roadByIndex.get(index);
    const button = document.createElement("button");
    button.type = "button";
    button.className = "route-leg-chip";
    button.dataset.routeLegIndex = String(index);
    button.innerHTML = `<span class="route-leg-swatch"></span><span></span>`;
    $(".route-leg-swatch", button).style.background = leg.color;
    const viaText = leg.viaName ? ` · por ${leg.viaName}` : "";
    $("span:last-child", button).textContent = `${index + 1}. ${leg.from} -> ${leg.to}${viaText}${roadLeg ? ` · ${formatPoints(roadLeg.distanceKM)} km` : ""}`;
    button.addEventListener("click", () => onSelect(index));
    target.append(button);
  });
}

function renderRouteAlternativeControls(target, routes, onSelect) {
  target.hidden = routes.length <= 1;
  target.replaceChildren();
  if (routes.length <= 1) return;
  const label = document.createElement("strong");
  label.textContent = "Rutas disponibles";
  target.append(label);
  routes.forEach((route, index) => {
    const button = document.createElement("button");
    button.type = "button";
    button.className = `route-alt-button${index === 0 ? " is-active" : ""}`;
    button.textContent = `${route.label} · ${formatPoints(route.distanceKM)} km · ${formatCount(route.durationMin)} min`;
    button.addEventListener("click", () => {
      $$(".route-alt-button", target).forEach((item) => item.classList.remove("is-active"));
      button.classList.add("is-active");
      onSelect(route);
    });
    target.append(button);
  });
}

function renderItineraryResult(target, calculation, onAdjustmentChange, viaPoints = []) {
  const status = calculation.missing.length
    ? `Faltan ${calculation.missing.length} tramos en la matriz`
    : `${formatPoints(calculation.totalBaseKM)} km base + ${formatPoints(calculation.totalCompensationKM)} km compensacion`;
  const table = document.createElement("section");
  table.className = "work-table";
  table.innerHTML = `
    <div class="panel-header">
      <div><h3>Resultado del itinerario</h3><span class="small-text"></span></div>
      <strong></strong>
    </div>
    <div class="table-wrap">
      <table>
        <thead><tr><th>Tramo</th><th>Origen</th><th>Destino</th><th>Ruta</th><th>Km base</th><th>Km comp.</th><th>Km liquid.</th><th>Min.</th><th>Validacion</th></tr></thead>
        <tbody></tbody>
      </table>
    </div>
  `;
  $(".small-text", table).textContent = status;
  $(".panel-header strong", table).textContent = `${formatPoints(calculation.totalKM)} km liquidables - ${formatCurrency(calculation.mileageAmount)}`;
  const tbody = $("tbody", table);
  calculation.legs.forEach((leg, index) => {
    const tr = document.createElement("tr");
    tr.dataset.routeLegIndex = String(index);
    tr.style.setProperty("--route-leg-color", leg.color);
    tr.tabIndex = 0;
    tr.title = "Seleccionar tramo en el mapa";
    tr.innerHTML = `<td><span class="route-leg-ref"><span class="route-leg-swatch"></span><span></span></span></td><td></td><td></td><td></td><td></td><td></td><td></td><td></td><td></td>`;
    const cells = $$("td", tr);
    $(".route-leg-swatch", cells[0]).style.background = leg.color;
    $(".route-leg-ref span:last-child", cells[0]).textContent = String(index + 1);
    cells[1].textContent = leg.from;
    cells[2].textContent = leg.to;
    const routeCell = document.createElement("div");
    routeCell.className = "route-via-cell";
    const routeSelect = document.createElement("select");
    routeSelect.className = "route-via-select";
    routeSelect.setAttribute("aria-label", `Ruta alternativa del tramo ${index + 1}`);
    const defaultOption = document.createElement("option");
    defaultOption.value = "";
    defaultOption.textContent = "Recomendada";
    routeSelect.append(defaultOption);
    const legViaPoints = viaPoints.filter((point) =>
      normalizeRouteName(point.name) !== normalizeRouteName(leg.from)
      && normalizeRouteName(point.name) !== normalizeRouteName(leg.to),
    );
    if (leg.viaName && !legViaPoints.some((point) => point.name === leg.viaName)) {
      legViaPoints.unshift({ name: leg.viaName });
    }
    legViaPoints.forEach((point) => {
      const option = document.createElement("option");
      option.value = point.name;
      option.textContent = `Por ${point.name}`;
      routeSelect.append(option);
    });
    routeSelect.value = leg.viaName || "";
    const routeReasonInput = document.createElement("input");
    routeReasonInput.type = "text";
    routeReasonInput.className = "route-reason-input";
    routeReasonInput.placeholder = "Motivo ruta";
    routeReasonInput.value = leg.viaReason || "";
    routeReasonInput.hidden = !leg.viaName && !leg.viaReason;
    routeReasonInput.setAttribute("aria-label", `Motivo de ruta alternativa del tramo ${index + 1}`);
    routeCell.append(routeSelect, routeReasonInput);
    cells[3].append(routeCell);
    cells[4].textContent = `${formatPoints(leg.distanceKM)} km`;
    const compensationInput = document.createElement("input");
    compensationInput.type = "number";
    compensationInput.min = "0";
    compensationInput.step = "0.1";
    compensationInput.className = "route-comp-input";
    compensationInput.value = leg.compensationKM ? String(leg.compensationKM) : "0";
    compensationInput.setAttribute("aria-label", `Kilometros de compensacion del tramo ${index + 1}`);
    cells[5].append(compensationInput);
    cells[6].textContent = `${formatPoints(leg.liquidableKM)} km`;
    cells[7].textContent = `${formatCount(leg.minutes)}`;
    const stateWrap = document.createElement("div");
    stateWrap.className = "route-state-cell";
    const needsCompReason = leg.compensationKM > 0 && !leg.compensationReason;
    const needsRouteReason = leg.viaName && !leg.viaReason;
    const chip = document.createElement("span");
    chip.className = `status-chip ${needsCompReason || needsRouteReason ? "chip-red" : leg.viaName ? "chip-amber" : leg.distanceKM ? "chip-green" : "chip-amber"}`;
    chip.textContent = needsCompReason && needsRouteReason
      ? "Motivos requeridos"
      : needsRouteReason
        ? "Motivo ruta requerido"
        : needsCompReason
          ? "Motivo km requerido"
          : leg.viaName
            ? "Ruta alternativa"
            : leg.state;
    const compensationReasonInput = document.createElement("input");
    compensationReasonInput.type = "text";
    compensationReasonInput.className = "route-reason-input";
    compensationReasonInput.placeholder = "Motivo km";
    compensationReasonInput.value = leg.compensationReason || "";
    compensationReasonInput.hidden = !(leg.compensationKM > 0 || leg.compensationReason);
    compensationReasonInput.setAttribute("aria-label", `Motivo de compensacion del tramo ${index + 1}`);
    stateWrap.append(chip, compensationReasonInput);
    cells[8].append(stateWrap);
    const emitAdjustment = (routeChanged = false) => {
      onAdjustmentChange?.(leg, {
        compensationKM: Number(compensationInput.value || 0),
        compensationReason: compensationReasonInput.value,
        viaName: routeSelect.value,
        viaReason: routeSelect.value ? routeReasonInput.value : "",
        routeChanged,
      });
    };
    [routeSelect, routeReasonInput, compensationInput, compensationReasonInput].forEach((input) => {
      input.addEventListener("click", (event) => event.stopPropagation());
      input.addEventListener("keydown", (event) => event.stopPropagation());
    });
    routeSelect.addEventListener("change", () => emitAdjustment(true));
    routeReasonInput.addEventListener("change", () => emitAdjustment(false));
    compensationInput.addEventListener("change", () => emitAdjustment(false));
    compensationReasonInput.addEventListener("change", () => emitAdjustment(false));
    tbody.append(tr);
  });
  target.replaceChildren(table);
}

function routeMatrixSourceNote(matrix) {
  const details = document.createElement("details");
  details.className = "screen-meta-toggle";
  details.innerHTML = `
    <summary>Fuente, version y criterio de auditoria de la matriz</summary>
    <div class="screen-meta-body">
      <article><strong>Catalogo</strong><span></span></article>
      <article><strong>Motor</strong><span></span></article>
      <article><strong>Grafo</strong><span></span></article>
      <article><strong>Version</strong><span></span></article>
      <article><strong>Criterio</strong><span></span></article>
    </div>
  `;
  const spans = $$("span", details);
  spans[0].textContent = matrix.locality_source || "INE/NGMEP";
  spans[1].textContent = matrix.routing_engine || "Motor de rutas on-premise";
  spans[2].textContent = matrix.graph_source || "Grafo viario versionado";
  spans[3].textContent = matrix.matrix_version || "Pendiente";
  spans[4].textContent = matrix.distance_policy || "Guardar version y tramos del expediente.";
  return details;
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
  form.dataset.rptPositionForm = "true";
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
      headers: staffHeaders(),
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
  if (!window.confirm(`Borrar el puesto RPT ${code}. Esta accion se auditara y no se puede deshacer desde esta pantalla.`)) {
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
  form.dataset.categoryForm = "true";
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
      headers: staffHeaders(),
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
  if (!window.confirm(`Borrar la categoria ${slug}. Esta accion se auditara y puede afectar a Bolsa, RPT y certificados.`)) {
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
      headers: staffHeaders(),
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
  if (screen.id === "aprobaciones.bandeja") {
    handleApprovalRowAction(row, actionLabel);
    return;
  }
  if (screen.id === "personal.puestos") {
    populateRPTPositionForm(String(row[0] || ""));
    return;
  }
  if (screen.id === "admin.catalogos" && String(row[0] || "") === "categoria_profesional") {
    populateCategoryForm(String(row[1] || ""));
    return;
  }
  if (screen.id === "bolsa.convocatorias") {
    populateBolsaOfferForm(String(row[0] || ""));
    return;
  }
  const ref = row[0] || screen.title || screen.id;
  const nextState = nextStateForAction(actionLabel);
  recordReceipt(actionLabel, `${ref} -> ${nextState}`, state.activeModule);
  setStatus(`${actionLabel}: ${ref}`, "ready");
  if (state.portal) renderFlowPanel();
}

function handleApprovalRowAction(row, actionLabel = "Aprobar") {
  const approvalID = String(row?._approvalID || "");
  const approvalKind = String(row?._approvalKind || "");
  if (!approvalID || approvalKind === "empty") {
    setStatus("No hay expediente de aprobacion seleccionado", "ready");
    return;
  }
  if (row._blockedReason && /^ver$/i.test(actionLabel)) {
    setStatus(row._blockedReason, "warning");
    return;
  }
  if (/^ver$/i.test(actionLabel)) {
    setStatus(`Expediente consultado: ${row[1] || approvalID}`, "ready");
    return;
  }
  const [, rawID] = approvalID.split(":");
  const nextState = row._nextState || nextStateForAction(actionLabel);
  if (approvalKind === "dietas") {
    const record = (state.dietasSheets || []).find((item) => String(item.id) === rawID);
    if (!record) {
      setStatus(`No se encuentra la dieta ${rawID}`, "error");
      return;
    }
    const previous = record.estado || record.state || "Pendiente";
    record.estado = nextState;
    record.state = nextState;
    record.validatedAt = new Date().toLocaleString("es-ES");
    record.validatedBy = activeDemoUser().displayName;
    record.workflow = ["Empleado", "Jefe de servicio", "Tecnico RRHH", "Liquidacion"];
    saveDietasSheets(state.dietasSheets);
    recordReceipt(actionLabel, `${record.id}: ${previous} -> ${nextState}`, "dietas");
    setStatus(`${actionLabel}: ${record.id} queda ${nextState}`, "ready");
    renderModulePortal(state.portal);
    renderFlowPanel();
    return;
  }
  if (approvalKind === "cronos") {
    const record = (state.cronosSubmittedRequests || []).find((item) => String(item.id) === rawID);
    if (!record) {
      setStatus(`No se encuentra la solicitud ${rawID}`, "error");
      return;
    }
    const previous = record.estado || record.state || "Pendiente";
    record.estado = nextState;
    record.state = nextState;
    record.validatedAt = new Date().toLocaleString("es-ES");
    record.validatedBy = activeDemoUser().displayName;
    recordReceipt(actionLabel, `${record.id}: ${previous} -> ${nextState}`, "cronos");
    setStatus(`${actionLabel}: ${record.id} queda ${nextState}`, "ready");
    renderModulePortal(state.portal);
    renderFlowPanel();
    return;
  }
  if (approvalKind === "workspace") {
    const rowID = rawID || approvalID.replace(/^workspace:/, "");
    const previous = row[4] || "Pendiente";
    state.rowOverrides[rowID] = {
      ...(state.rowOverrides[rowID] || {}),
      state: nextState,
      stateFilter: /pendiente/i.test(nextState) ? "Pendiente de accion" : "En revision",
      deadline: "Accion registrada",
      deadlineBucket: "Sin vencimiento critico",
      action: "Abrir expediente",
      alerts: [[nextState, `${actionLabel} ejecutada desde bandeja de aprobaciones`]],
      timeline: [[actionLabel, `${new Date().toLocaleString("es-ES")} - ${previous} -> ${nextState}`]],
    };
    recordReceipt(actionLabel, `${row[1] || rowID}: ${previous} -> ${nextState}`, "aprobaciones");
    renderPortal(state.portal);
    setStatus(`${actionLabel}: ${row[1] || rowID}`, "ready");
  }
}

function populateBolsaOfferForm(title) {
  const form = document.querySelector("[data-bolsa-offer-form]");
  const offer = ensureBolsaOffers(state.portal).find((item) => String(item.title) === title || String(item.id) === title);
  if (!offer || !form) {
    setStatus(`No se ha encontrado la oferta ${title}`, "error");
    return;
  }
  setFormValue(form, "offer_id", offer.id);
  setFormValue(form, "title", offer.title);
  setFormValue(form, "category", offer.category);
  setFormValue(form, "unit", offer.unit);
  const deadlineParts = String(offer.deadline || "").split("/");
  const deadlineISO = deadlineParts.length === 3 ? `${deadlineParts[2]}-${deadlineParts[1]}-${deadlineParts[0]}` : offer.deadline;
  setFormValue(form, "deadline", deadlineISO);
  setFormValue(form, "state", offer.state || "Abierta");
  setFormValue(form, "requirements", offer.requirements);
  setFormValue(form, "bases_ref", offer.basesRef);
  const feeRequired = form.elements?.fee_required;
  if (feeRequired) feeRequired.checked = offer.feeRequired === true;
  setFormValue(form, "fee_amount", offer.feeAmount || 0);
  setFormValue(form, "fee_code", offer.feeCode || "");
  form.scrollIntoView({ block: "nearest", behavior: "smooth" });
  setStatus(`Oferta seleccionada: ${offer.id}`, "ready");
}

function setFormValue(form, name, value) {
  const field = form?.elements?.[name];
  if (!field) return;
  field.value = value == null ? "" : String(value);
}

function populateRPTPositionForm(code) {
  const position = (getPersonalCatalog(state.portal).positions?.items || []).find((item) => String(item.code) === code);
  const form = document.querySelector("[data-rpt-position-form]");
  if (!position || !form) {
    setStatus(`No se ha encontrado el puesto RPT ${code}`, "error");
    return;
  }
  setFormValue(form, "code", position.code);
  setFormValue(form, "name", position.name);
  setFormValue(form, "type", position.type || "N");
  setFormValue(form, "administration", position.administration || "F");
  setFormValue(form, "provision", position.provision || "C");
  setFormValue(form, "group", position.group || "");
  setFormValue(form, "category_code", position.category_code || "");
  setFormValue(form, "destination_level", position.destination_level || "");
  setFormValue(form, "state", position.state || "Vigente");
  form.scrollIntoView({ block: "nearest", behavior: "smooth" });
  setStatus(`Puesto RPT seleccionado: ${position.code}`, "ready");
}

function populateCategoryForm(slug) {
  const category = (getPersonalCatalog(state.portal).categories?.items || []).find((item) => String(item.slug) === slug);
  const form = document.querySelector("[data-category-form]");
  if (!category || !form) {
    setStatus(`No se ha encontrado la categoria ${slug}`, "error");
    return;
  }
  setFormValue(form, "slug", category.slug);
  setFormValue(form, "name", category.name);
  setFormValue(form, "area", category.area || "administracion_general");
  setFormValue(form, "source", category.source || "VEC");
  setFormValue(form, "usage", category.usage || "");
  form.scrollIntoView({ block: "nearest", behavior: "smooth" });
  setStatus(`Categoria seleccionada: ${category.name}`, "ready");
}

function screenActions(screen) {
  const actions = Array.isArray(screen.actions) && screen.actions.length ? screen.actions : ["Registrar accion", "Exportar"];
  return actions.slice(0, 4).map((action) => ({ label: String(action) }));
}

function screenHeaders(screen) {
  if (screen.id === "aprobaciones.bandeja") {
    return ["Modulo", "Solicitud", "Solicitante", "Importe/dias", "Estado", "Responsable"];
  }
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
  if (screen.id === "rutas.kilometraje") {
    return ["Tramo", "Origen", "Destino", "Km matriz", "Minutos", "Estado"];
  }
  if (screen.id === "rutas.mapa_provincia") {
    return ["Codigo", "Localidad", "Tipo", "Municipio", "Fuente", "Estado"];
  }
  if (screen.id === "bolsa.convocatorias") {
    return ["Oferta", "Categoria", "Unidad", "Plazo", "Estado", "Solicitudes"];
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

function currentApprovalRole() {
  const roles = currentRoleList();
  if (isAdminSession()) return "administrador";
  if (roles.some((role) => ["tecnico_rrhh", "rrhh", "personal_rrhh"].includes(role))) return "tecnico_rrhh";
  if (roles.includes("jefe_servicio")) return "jefe_servicio";
  if (roles.includes("jefe_seccion")) return "jefe_seccion";
  return "empleado";
}

function approvalInboxRow(values, meta) {
  const row = values.slice();
  row._approvalID = meta.id;
  row._approvalKind = meta.kind;
  row._actionLabel = meta.actionLabel || "Abrir";
  row._nextState = meta.nextState || "";
  row._blockedReason = meta.blockedReason || "";
  return row;
}

function dietasApprovalAction(record) {
  const role = currentApprovalRole();
  const stateText = String(record.estado || record.state || "");
  if (/pendiente jefe de servicio/i.test(stateText)) {
    if (role === "jefe_servicio" || role === "administrador") {
      return { actionLabel: "Aprobar servicio", nextState: "Pendiente validacion RRHH" };
    }
    return { actionLabel: "Ver", blockedReason: "Pendiente de jefe de servicio" };
  }
  if (/pendiente validacion rrhh|validada servicio/i.test(stateText)) {
    if (role === "tecnico_rrhh" || role === "administrador") {
      return { actionLabel: "Validar RRHH", nextState: "Lista para liquidacion" };
    }
    return { actionLabel: "Ver", blockedReason: "Pendiente de tecnico RRHH" };
  }
  return { actionLabel: "Ver", nextState: stateText };
}

function cronosApprovalAction(record) {
  const role = currentApprovalRole();
  const stateText = String(record.estado || record.state || "");
  if (/pendiente responsable/i.test(stateText)) {
    if (["jefe_servicio", "jefe_seccion", "administrador"].includes(role)) {
      return { actionLabel: "Aprobar", nextState: "Aprobada" };
    }
    return { actionLabel: "Ver", blockedReason: "Pendiente de responsable" };
  }
  if (/pendiente rrhh|validacion rrhh/i.test(stateText)) {
    if (role === "tecnico_rrhh" || role === "administrador") {
      return { actionLabel: "Validar RRHH", nextState: "Cerrada RRHH" };
    }
    return { actionLabel: "Ver", blockedReason: "Pendiente de RRHH" };
  }
  return { actionLabel: "Ver", nextState: stateText };
}

function approvalInboxRows(view) {
  const rows = [];
  ensureDietasSheets().forEach((record) => {
    const action = dietasApprovalAction(record);
    rows.push(approvalInboxRow([
      "Dietas",
      record.id || "DIET-PROP",
      record.employee || "Empleado demo",
      formatEuroCompact(record.importe || record.amount || record.total || 0),
      record.estado || record.state || "Pendiente",
      /rrhh/i.test(record.estado || record.state || "") ? "Tecnico RRHH" : "Jefe de servicio",
    ], {
      id: `dietas:${record.id}`,
      kind: "dietas",
      ...action,
    }));
  });
  (state.cronosSubmittedRequests || []).forEach((record) => {
    const action = cronosApprovalAction(record);
    rows.push(approvalInboxRow([
      "Cronos",
      record.id || "PER-PROP",
      "Empleado demo",
      record.duracion || record.amount || "-",
      record.estado || record.state || "Pendiente",
      /rrhh/i.test(record.estado || record.state || "") ? "Tecnico RRHH" : "Responsable",
    ], {
      id: `cronos:${record.id}`,
      kind: "cronos",
      ...action,
    }));
  });
  filteredRows()
    .filter((row) => row.modules.includes("aprobaciones"))
    .slice(0, 8)
    .forEach((row) => {
      const override = state.rowOverrides[row.id] || {};
      const stateText = override.state || row.state;
      const actionLabel = /aprobad|cerrad|listo|firmad/i.test(stateText) ? "Ver" : "Aprobar";
      rows.push(approvalInboxRow([
        row.unit || row.modules.find((item) => item !== "dashboard" && item !== "aprobaciones") || "VEC",
        row.expediente,
        row.candidate,
        row.points || row.deadline || "-",
        stateText,
        row.scope || row.document || "-",
      ], {
        id: `workspace:${row.id}`,
        kind: "workspace",
        actionLabel,
        nextState: "Aprobado / listo",
      }));
    });
  if (!rows.length) {
    return [approvalInboxRow(["Sin pendientes", "-", "-", "-", "Sin tareas", "-"], { id: "empty", kind: "empty", actionLabel: "Ver" })];
  }
  return rows;
}

function screenRows(screen, view) {
  if (screen.id === "aprobaciones.bandeja") {
    return approvalInboxRows(view);
  }
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
  if (screen.id === "rutas.kilometraje") {
    return (view.workspace?.province_route_pairs || view.workspace?.province_routes || []).map((route) => [
      route.id,
      route.from,
      route.to,
      `${formatPoints(route.distance_km ?? route.km_one_way)} km`,
      `${formatCount(route.duration_minutes ?? route.estimated_minutes)} min`,
      route.state || "Matriz cargada",
    ]);
  }
  if (screen.id === "rutas.mapa_provincia") {
    return (view.workspace?.province_route_points || view.workspace?.province_localities || []).map((point) => [
      point.code || point.ine_code,
      point.name,
      point.kind,
      point.municipality_name || point.name,
      point.source,
      point.state || "Vigente",
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
    return ensureBolsaOffers(view).map((item) => [
      item.title,
      item.category,
      item.unit,
      item.deadline,
      item.state,
      formatCount(item.applications || 0),
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
  if (moduleID === "aprobaciones") {
    const rows = approvalInboxRows(state.portal);
    const target = rows.find((row) =>
      row._actionLabel
      && !/^ver$/i.test(row._actionLabel)
      && !row._blockedReason
      && (String(actionDef.label || "").toLowerCase().includes("aprobar") || String(row._actionLabel).toLowerCase().includes(String(actionDef.label || "").toLowerCase().split(/\s+/)[0] || "")),
    ) || rows.find((row) => row._actionLabel && !/^ver$/i.test(row._actionLabel) && !row._blockedReason);
    if (target) {
      handleApprovalRowAction(target, target._actionLabel || actionDef.label);
      return;
    }
    setStatus("No hay aprobaciones accionables para el rol activo", "ready");
    return;
  }
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

function portalTable(title, headers, rows, options = {}) {
  const wrap = document.createElement("section");
  wrap.className = "portal-table";
  const heading = document.createElement("h3");
  heading.textContent = title;
  const table = document.createElement("table");
  table.innerHTML = `<thead><tr></tr></thead><tbody></tbody>`;
  const tr = $("thead tr", table);
  const actionColumns = new Set(options.actionColumns || []);
  headers.forEach((header) => {
    const th = document.createElement("th");
    th.textContent = header;
    tr.append(th);
  });
  if (options.actionColumn) {
    headers.forEach((header, index) => {
      if (String(header || "").toLowerCase() === "accion") actionColumns.add(index);
    });
  }
  const tbody = $("tbody", table);
  (rows.length ? rows : [["Sin datos", "-", "-", "-"]]).forEach((row) => {
    const bodyRow = document.createElement("tr");
    headers.forEach((_, index) => {
      const td = document.createElement("td");
      const value = row[index] == null || row[index] === "" ? "-" : row[index];
      if (actionColumns.has(index) && value !== "-") {
        const button = document.createElement("button");
        button.type = "button";
        button.className = "table-action portal-row-action";
        button.textContent = value;
        button.addEventListener("click", () => handlePortalRowAction(title, row, value));
        td.append(button);
      } else {
        td.textContent = value;
      }
      bodyRow.append(td);
    });
    tbody.append(bodyRow);
  });
  wrap.append(heading, table);
  return wrap;
}

function portalDocumentCSV(prefix, ref) {
  const cleanPrefix = String(prefix || "DOC").replace(/[^A-Z0-9]/gi, "").toUpperCase().slice(0, 8) || "DOC";
  const cleanRef = slugify(ref || "vec").replace(/-/g, "").toUpperCase().slice(0, 16) || "VEC";
  return `CSV-${cleanPrefix}-${cleanRef}-${String(Date.now()).slice(-6)}`;
}

function buildSignedPortalPDF({ title, subtitle, rows, csv, module, note }) {
  const issuedAt = new Date().toLocaleString("es-ES");
  const verificationURL = documentVerificationURL(csv);
  const shortText = (value, max) => {
    const text = String(value || "");
    return text.length > max ? `${text.slice(0, max - 3)}...` : text;
  };
  const lines = [
    "0.98 0.99 1 rg 36 36 523 770 re f",
    "0.78 0.84 0.88 RG 36 36 523 770 re S",
    "1 1 1 rg 45 736 505 72 re f",
    "0.72 0.80 0.88 RG 45 736 505 72 re S",
    "0.67 0.80 0.29 rg 45 736 505 4 re f",
    "0 G 0 g",
  ];
  drawDiputacionLogoPDF(lines, 58, 782, 0.70);
  lines.push("0.09 0.23 0.31 rg");
  pdfLine(lines, 372, 787, shortText(String(title || "Documento VEC").toUpperCase(), 26), { size: 10.5, bold: true });
  pdfLine(lines, 372, 770, shortText(module || "Portal VEC", 28), { size: 8 });

  lines.push("0.08 0.13 0.20 rg");
  pdfLine(lines, 58, 704, shortText(title || "Documento VEC", 62), { size: 13, bold: true });
  pdfLine(lines, 58, 686, shortText(subtitle || "Documento generado desde VEC", 76), { size: 9 });

  lines.push("0.10 0.29 0.47 rg 58 642 474 24 re f");
  lines.push("1 1 1 rg");
  pdfLine(lines, 66, 650, "DATO", { size: 8.5, bold: true });
  pdfLine(lines, 230, 650, "VALOR", { size: 8.5, bold: true });

  let y = 610;
  (rows || []).slice(0, 12).forEach(([labelText, value], index) => {
    lines.push(`${index % 2 ? "0.94 0.97 0.95" : "0.97 0.99 1"} rg 58 ${y - 8} 474 24 re f`);
    lines.push("0.84 0.88 0.92 RG 58 " + (y - 8) + " 474 24 re S");
    lines.push("0.08 0.13 0.20 rg");
    pdfLine(lines, 66, y, shortText(labelText, 26), { size: 8.6, bold: true });
    pdfLine(lines, 230, y, shortText(value, 52), { size: 8.6 });
    y -= 28;
  });

  lines.push("0.96 0.98 1 rg 58 174 310 70 re f");
  lines.push("0.72 0.80 0.88 RG 58 174 310 70 re S");
  lines.push("0.08 0.13 0.20 rg");
  pdfLine(lines, 66, 221, "Firma y verificacion", { size: 10, bold: true });
  pdfLine(lines, 66, 203, `CSV: ${csv}`, { size: 8.5, bold: true });
  pdfLine(lines, 66, 187, `Fecha de firma demo: ${issuedAt}`, { size: 8.3 });
  pdfLine(lines, 66, 158, shortText(note || "Documento emitido en entorno demo para validar el flujo VEC.", 78), { size: 8 });
  drawQRCodePDF(lines, verificationURL, 480, 54, 1.30);
  pdfLine(lines, 382, 98, "Verificar documento", { size: 8, bold: true });
  pdfLine(lines, 382, 84, shortText(csv, 30), { size: 7 });
  return buildSimplePDF(lines.join("\n"));
}

function downloadSignedPortalPDF({ title, subtitle, rows, csv, ref, module, filename, note }) {
  const docCSV = csv || portalDocumentCSV("VEC", ref || title);
  const blob = buildSignedPortalPDF({ title, subtitle, rows, csv: docCSV, module, note });
  downloadBlob(blob, filename || `vec-${slugify(title || "documento")}-${slugify(ref || docCSV)}.pdf`);
  return docCSV;
}

function portalModuleLabel() {
  return MODULES.find((module) => module.id === state.activeModule)?.label || state.activeModule || "VEC";
}

function downloadEmployeePortalDocument(title, row, action) {
  const ref = String(row?.[1] || row?.[0] || title || "documento");
  const rowText = (row || []).join(" | ");
  if (/retenciones|10t/i.test(rowText)) {
    downloadRetencionesCertificatePDF();
    return;
  }
  if (/recibo|salarios|nomina|nom-2026-06/i.test(rowText)) {
    const month = /2026-06|junio/i.test(rowText) ? "Junio 2026" : "Junio 2026";
    printPayrollPDF(getPayrollCalculations(month), month);
    return;
  }
  const csv = String(row?.[3] || "").startsWith("CSV-") ? row[3] : portalDocumentCSV("DOC", ref);
  downloadSignedPortalPDF({
    title: String(row?.[0] || action || "Documento VEC"),
    subtitle: `${portalModuleLabel()} - ${String(action || "consulta")}`,
    ref,
    csv,
    module: portalModuleLabel(),
    filename: `vec-${slugify(action || "documento")}-${slugify(ref)}.pdf`,
    rows: [
      ["Referencia", ref],
      ["Expediente", row?.[1] || row?.[0] || "-"],
      ["Estado", row?.[2] || "-"],
      ["Accion", action],
      ["Empleado", payrollEmployeeData().name],
      ["Usuario", activeDemoUser().displayName],
    ],
  });
}

function downloadPersonalCertificate(row, action) {
  const employee = payrollEmployeeData();
  const ref = String(row?.[0] || "CERT-SERV-2026");
  const csv = portalDocumentCSV("CERT", ref);
  downloadSignedPortalPDF({
    title: "Certificado de servicios prestados",
    subtitle: "Area de Recursos Humanos y Regimen Interior",
    ref,
    csv,
    module: "Personal",
    filename: `certificado-servicios-${slugify(ref)}.pdf`,
    rows: [
      ["Empleado", employee.name],
      ["NIF", employee.nif],
      ["Puesto", employee.position],
      ["Centro", employee.service],
      ["Relacion juridica", employee.relationship],
      ["Antiguedad", `${String(employee.trienios).padStart(2, "0")} trienios reconocidos`],
      ["Expediente", ref],
      ["Estado", "Emitido para firma demo"],
    ],
    note: "Certificado demo emitido desde VEC para comprobar descarga, firma y CSV.",
  });
  recordReceipt(action, `${ref} - ${csv}`, "personal");
  setStatus(`Certificado generado: ${csv}`, "ready");
}

function handleBolsaPortalApplicationAction(row, action) {
  const applications = ensureEmployeeBolsaApplications(state.portal);
  const application = applications.find((item) => String(item.id) === String(row?.[0]));
  if (!application) {
    setStatus(`No se encuentra la solicitud ${row?.[0] || ""}`, "error");
    return true;
  }
  const offers = ensureBolsaOffers(state.portal);
  const offer = offers.find((item) => item.id === application.offerID) || {
    id: application.offerID,
    title: application.title,
    feeRequired: application.feeRequired,
    feeAmount: application.feeAmount,
  };
  const actionText = String(action || "");
  if (/completar|ver solicitud|formulario/i.test(actionText)) {
    state.bolsaSelectedOfferID = offer.id;
    recordReceipt(actionText, `${application.id} - ${application.title}`, "bolsa");
    renderModulePortal(state.portal);
    $(".bolsa-application-panel")?.scrollIntoView({ block: "nearest", behavior: "smooth" });
    setStatus(`${actionText}: ${application.id}`, "ready");
    return true;
  }
  if (/pagar tasa/i.test(actionText)) {
    if (!application.feeRequired) {
      setStatus("Esta solicitud no exige tasa.", "ready");
      return true;
    }
    application.paymentState = "Pagada";
    application.paymentReceipt = application.paymentReceipt || `TASA-${new Date().toISOString().slice(0, 10).replaceAll("-", "")}-${String(Date.now()).slice(-5)}`;
    application.paidAt = new Date().toLocaleString("es-ES");
    saveBolsaApplications(applications);
    state.bolsaSelectedOfferID = offer.id;
    recordReceipt("Tasa Bolsa pagada", `${application.paymentReceipt} - ${application.id} - ${formatCurrency(application.feeAmount)}`, "bolsa");
    renderModulePortal(state.portal);
    setStatus(`Tasa pagada: ${application.paymentReceipt}`, "ready");
    return true;
  }
  if (/firmar/i.test(actionText)) {
    application.signatureState = "Firmada";
    application.signatureCSV = application.signatureCSV || `CSV-FIR-${application.id.replace(/\W+/g, "-")}`;
    application.signedAt = new Date().toLocaleString("es-ES");
    application.signatureProvider = activeDemoUser().auth === "dnie" ? "DNIe/certificado digital" : "certificado electronico";
    saveBolsaApplications(applications);
    state.bolsaSelectedOfferID = offer.id;
    recordReceipt("Firma electronica Bolsa", `${application.signatureCSV} - ${application.id}`, "bolsa");
    renderModulePortal(state.portal);
    setStatus(`Solicitud firmada: ${application.signatureCSV}`, "ready");
    return true;
  }
  if (/certificado|desistimiento/i.test(actionText)) {
    const csv = application.withdrawalReceipt || application.signatureCSV || application.registryNumber || portalDocumentCSV("BOL", application.id);
    downloadSignedPortalPDF({
      title: /desistimiento/i.test(actionText) ? "Justificante de desistimiento Bolsa" : "Justificante de solicitud Bolsa",
      subtitle: application.title,
      ref: application.id,
      csv,
      module: "Bolsa",
      filename: `bolsa-${slugify(actionText)}-${slugify(application.id)}.pdf`,
      rows: [
        ["Solicitud", application.id],
        ["Oferta", application.title],
        ["Categoria", application.category],
        ["Estado", application.state],
        ["Registro", application.registryNumber || "-"],
        ["Firma", application.signatureCSV || application.signatureState || "-"],
        ["Tasa", application.paymentReceipt || application.paymentState || "-"],
        ["Fecha", application.submittedAt || application.withdrawnAt || application.createdAt || "-"],
      ],
    });
    recordReceipt(actionText, `${application.id} - ${csv}`, "bolsa");
    setStatus(`${actionText}: ${application.id}`, "ready");
    return true;
  }
  return false;
}

function handlePortalRowAction(title, row, action) {
  const ref = String(row?.[0] || title || "Expediente propio");
  const detail = `${title}: ${ref}`;
  if (state.activeModule === "bolsa" && /mis solicitudes/i.test(String(title || ""))) {
    if (handleBolsaPortalApplicationAction(row, action)) return;
  }
  if (state.activeModule === "personal" && /solicitar certificado/i.test(String(action || ""))) {
    downloadPersonalCertificate(row, action);
    return;
  }
  if (/descargar|ver recibo|ver justificante|ver certificado|ver resolucion|ver liquidacion/i.test(String(action || ""))) {
    downloadEmployeePortalDocument(title, row, action);
    recordReceipt(action, detail, state.activeModule || "portal-empleado");
    setStatus(`${action}: ${ref}`, "ready");
    return;
  }
  recordReceipt(action, detail, state.activeModule || "portal-empleado");
  if (/descargar/i.test(String(action))) {
    const content = [
      "VEC Diputacion de Granada",
      "Justificante de autoservicio",
      `Accion: ${action}`,
      `Modulo: ${MODULES.find((module) => module.id === state.activeModule)?.label || state.activeModule}`,
      `Referencia: ${ref}`,
      `Detalle: ${(row || []).join(" | ")}`,
      `Fecha: ${new Date().toLocaleString("es-ES")}`,
      `Usuario: ${activeDemoUser().displayName}`,
    ].join("\n");
    const blob = new Blob([content], { type: "text/plain;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `vec-${slugify(action)}-${slugify(ref)}.txt`;
    document.body.append(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(url);
  }
  setStatus(`${action}: ${ref}`, "ready");
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
    const rawNext = ["personal", "nominas", "cronos", "horarios", "permisos", "dietas", "rutas", "bolsa"].find((module) => row?.modules.includes(module)) || "personal";
    const next = MODULE_PARENT[rawNext] || rawNext;
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
    const moduleMatches = state.activeModule === "dashboard" || rowMatchesActiveModule(row, state.activeModule);
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

function rowMatchesActiveModule(row, moduleID) {
  if (row.modules.includes(moduleID)) return true;
  return Object.entries(MODULE_PARENT).some(([child, parent]) => parent === moduleID && row.modules.includes(child));
}

function selectedRow() {
  return state.rows.find((row) => row.id === state.selectedRowID) || state.rows[0] || null;
}

function hashStateFromLocation() {
  const raw = decodeURIComponent(String(window.location.hash || "").replace(/^#/, "")).trim();
  if (!raw) return { moduleID: "", screenID: "" };
  let [moduleID, screenID] = raw.split("/");
  if (MODULE_PARENT[moduleID]) {
    screenID = screenID || MODULE_DEFAULT_SCREEN[moduleID] || "";
    moduleID = MODULE_PARENT[moduleID];
  }
  const validModule = MODULES.some((module) => module.id === moduleID && !MODULE_PARENT[module.id]) || flowRenderers[moduleID];
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
  if (MODULE_PARENT[moduleID]) {
    screenID = screenID || MODULE_DEFAULT_SCREEN[moduleID] || "";
    moduleID = MODULE_PARENT[moduleID];
  }
  state.activeModule = moduleIDForSession(moduleID);
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
    if (isEmployeeSelfServiceSession()) {
      setOperationalPanelsHidden(true);
      updateLocationHash();
      setStatus(`Modulo activo: ${MODULES.find((module) => module.id === state.activeModule)?.label || state.activeModule}`, "ready");
      return;
    }
    setOperationalPanelsHidden(false);
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
    if (isEmployeeSelfServiceSession()) {
      setOperationalPanelsHidden(true);
      updateLocationHash();
      setStatus(`Pantalla activa: ${activeScreen(state.portal)?.title || state.activeScreen || state.activeModule}`, "ready");
      return;
    }
    setOperationalPanelsHidden(false);
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
  updateTopbarContext();
  const panel = $(".summary-panel");
  const showSummaryPanel = isAdminSession() && state.activeModule === "dashboard";
  if (panel) {
    panel.hidden = !showSummaryPanel;
  }
  if (!showSummaryPanel) {
    return;
  }
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
  if (isEmployeeSelfServiceSession()) {
    const ownLeads = {
      personal: "Consulta tus datos personales, puesto, situacion administrativa, antiguedad y certificados propios.",
      nominas: "Consulta tus recibos, historico salarial y certificados propios.",
      cronos: "Consulta tus fichajes, saldos, permisos, vacaciones e incidencias propias.",
      horarios: "Consulta tu jornada y condiciones horarias personales.",
      permisos: "Solicita y consulta tus permisos, vacaciones y asuntos propios.",
      dietas: "Consulta y tramita tus propias comisiones de servicio, kilometraje y gastos.",
      rutas: "Calcula rutas vinculadas a tus propias solicitudes de dieta.",
      bolsa: "Consulta tus expedientes y solicitudes propias en Bolsa.",
      documentos: "Consulta tus documentos, justificantes, certificados y firmas.",
      notificaciones: "Consulta tus notificaciones y tareas pendientes.",
    };
    return ownLeads[moduleID] || "Consulta tus datos y expedientes propios en VEC.";
  }
  const leads = {
    personal: "Trabaja sobre expedientes de empleado, puesto, situacion, antiguedad, servicios prestados y certificados.",
    nominas: "Controla el periodo de nomina, conceptos, incidencias, deducciones, cruces con Cronos/Dietas y cierre.",
    cronos: "Gestiona fichajes, horarios, turnos, flexibilidad, reducciones 63/64, permisos, vacaciones e incidencias.",
    horarios: "Define perfiles horarios por puesto/unidad, flexibilidad, coberturas obligatorias y reducciones 63/64.",
    permisos: "Resuelve solicitudes de asuntos propios, vacaciones, compensaciones y saldos con aprobacion responsable.",
    dietas: "Tramita comisiones de servicio con ruta, mapa de kilometraje provincial, justificantes, politica de dieta y liquidacion.",
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

function isOpenOnlyAction(action) {
  const text = String(action || "").toLowerCase();
  return /abrir|ver|consultar/.test(text) && !/aprobar|validar|revisar|resolver|liquidar|cerrar|emitir|generar|exportar|rechazar|denegar/.test(text);
}

function openOperationalRow(row, action = "Abrir expediente") {
  state.selectedRowID = row.id;
  state.activeTab = "resumen";
  const previous = state.rowOverrides[row.id] || {};
  state.rowOverrides[row.id] = {
    ...previous,
    timeline: [
      [action, `${new Date().toLocaleString("es-ES")} - expediente abierto por ${activeDemoUser().displayName}`],
      ...(previous.timeline || row.timeline || []),
    ],
    alerts: previous.alerts || row.alerts,
  };
  recordReceipt(action, `${row.expediente} - ${row.scope || row.document || "expediente"}`, moduleEndpointFor(state.activeModule, row) || state.activeModule);
  renderPortal(state.portal);
  const detail = $(".right-column .detail-panel") || $(".right-column");
  if (detail && !detail.hidden) {
    detail.scrollIntoView({ block: "nearest", behavior: "smooth" });
  }
  setStatus(`${action}: ${row.expediente}`, "ready");
}

async function handleRowAction(row) {
  if (!row) return;
  state.selectedRowID = row.id;
  const action = row.action || "Abrir";
  if (action.toLowerCase().includes("exportar")) {
    exportRows();
    return;
  }
  if (isOpenOnlyAction(action)) {
    openOperationalRow(row, action);
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
  renderDemoUserSwitcher();
  renderDocumentVerificationNotice();
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

function renderDocumentVerificationNotice() {
  const params = new URLSearchParams(window.location.search);
  const csv = String(params.get("v") || params.get("validar_csv") || "").trim();
  if (!csv || $("#document-verification-banner")) return;
  const documentInfo = SIGNED_DOCUMENTS[csv];
  const banner = document.createElement("section");
  banner.id = "document-verification-banner";
  banner.style.margin = "12px 20px 0";
  banner.style.padding = "12px 14px";
  banner.style.borderRadius = "8px";
  banner.style.border = documentInfo ? "1px solid #9fd0a6" : "1px solid #f2b8b5";
  banner.style.background = documentInfo ? "#e8f5e9" : "#fff1f0";
  banner.style.color = documentInfo ? "#1b5e20" : "#9b1c1c";
  banner.style.display = "grid";
  banner.style.gridTemplateColumns = "minmax(0, 1fr) auto";
  banner.style.gap = "12px";
  banner.style.alignItems = "center";

  const copy = document.createElement("div");
  const title = document.createElement("strong");
  title.textContent = documentInfo ? "Documento verificado por CSV" : "CSV no reconocido";
  const detail = document.createElement("div");
  detail.style.marginTop = "4px";
  detail.style.fontSize = "0.9rem";
  detail.textContent = documentInfo
    ? `${documentInfo.title} · ${documentInfo.issuer} · Estado: ${documentInfo.state} · CSV: ${csv}`
    : `No existe un documento firmado registrado para el CSV ${csv}.`;
  copy.append(title, detail);

  const close = document.createElement("button");
  close.type = "button";
  close.textContent = "Cerrar";
  close.style.minHeight = "34px";
  close.style.border = "1px solid currentColor";
  close.style.borderRadius = "6px";
  close.style.background = "#fff";
  close.style.color = "inherit";
  close.style.cursor = "pointer";
  close.style.fontWeight = "700";
  close.addEventListener("click", () => banner.remove());

  banner.append(copy, close);
  $(".topbar")?.after(banner);
}

function renderDemoUserSwitcher() {
  const tools = $(".operator-tools");
  if (!tools || $("#demo-user-select")) return;
  const wrap = document.createElement("label");
  wrap.className = "demo-user-switch";
  wrap.innerHTML = `<span id="demo-user-role-label"></span><select id="demo-user-select" aria-label="Cambiar usuario y rol de prueba"></select>`;
  const select = $("select", wrap);
  DEMO_USERS.forEach((user) => {
    const option = document.createElement("option");
    option.value = user.id;
    option.textContent = `${user.label}`;
    select.append(option);
  });
  select.value = activeDemoUserID;
  select.addEventListener("change", async () => {
    const nextUserID = select.value;
    setStatus(`Cambiando a ${DEMO_USERS.find((user) => user.id === nextUserID)?.label || nextUserID}`, "loading");
    await applyDemoUser(nextUserID, { reload: true, switchModule: true });
    setStatus(`Usuario activo: ${activeDemoUser().displayName} - ${sessionAccessProfile().label}`, "ready");
  });
  tools.prepend(wrap);
  updateDemoUserUI();
}

function renderPortal(view) {
  if (document.body) {
    document.body.dataset.accessProfile = sessionAccessProfile().id;
  }
  state.rows = rowsFromPortal(view).filter(rowVisibleForSession);
  if (!state.selectedRowID && state.rows.length) {
    state.selectedRowID = state.rows[0].id;
  }
  renderModuleHeader();
  renderKPIs(view);
  renderModules(view);
  renderModulePortal(view);
  if (isEmployeeSelfServiceSession()) {
    setOperationalPanelsHidden(true);
    return;
  }
  setOperationalPanelsHidden(false);
  renderFlowPanel();
  renderTable(view);
  renderCronosPanel(view);
  renderDietasPanel(view);
}

function setOperationalPanelsHidden(hidden) {
  [
    ".filter-bar",
    ".queue-panel",
    ".workspace-row",
    ".right-column",
    "#flow-panel",
  ].forEach((selector) => {
    $$(selector).forEach((node) => {
      node.hidden = Boolean(hidden);
    });
  });
  if (hidden) {
    $$(".summary-panel, .metrics").forEach((node) => {
      node.hidden = true;
    });
  }
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
      getData(`${PERSONAL_RPT_POSITIONS_API}?limit=2000`, { method: "GET", headers: staffHeaders() }),
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
    state.activeModule = moduleIDForSession(hashState.moduleID || state.activeModule);
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

function payrollEmployeeData() {
  return {
    name: "ALBERTO SÁNCHEZ GÓMEZ",
    nif: "74839201A",
    service: "TRANSFORMACIÓN DIGITAL / NUEVAS TECNOLOGÍAS",
    position: "TÉCNICO DE GESTIÓN (A2)",
    trienios: state.nominasTrieniosCount ?? 4,
    relationship: "FUNCIONARIO DE CARRERA",
    iban: "ES91 2100 0482 12 0123456789",
    affiliation: "18/1234567-89",
  };
}

function payrollConceptRows(calc) {
  const rows = [
    { code: "11", concept: "Sueldo Base (Grupo A2)", devengo: calc.sueldoBase, deduccion: null },
    { code: "12", concept: `Trienios acumulados (${state.nominasTrieniosCount})`, devengo: calc.trieniosVal, deduccion: null },
    { code: "52", concept: "Complemento de Destino (Nivel 22)", devengo: calc.destVal, deduccion: null },
  ];
  if (state.nominasComplementoEspecifico > 0) {
    rows.push({ code: "53", concept: "Complemento Específico", devengo: calc.specVal, deduccion: null });
  }
  rows.push({ code: "55", concept: "Productividad e Incentivos", devengo: calc.prodVal, deduccion: null });
  if (calc.dietasVal > 0) {
    rows.push({ code: "59", concept: "Dietas y Locomoción (Cruce VEC)", devengo: calc.dietasVal, deduccion: null });
  }
  rows.push(
    { code: "1", concept: `I.R.P.F. Retenciones Practicadas (${state.nominasIrpfPercent}%)`, devengo: null, deduccion: calc.irpf },
    { code: "3", concept: "Cotización General Seguridad Social (4.7%)", devengo: null, deduccion: calc.segSocial },
  );
  return rows;
}

function escapeHTML(value) {
  return String(value ?? "").replace(/[&<>"']/g, (char) => ({
    "&": "&amp;",
    "<": "&lt;",
    ">": "&gt;",
    "\"": "&quot;",
    "'": "&#39;",
  }[char]));
}

function formatPayrollMoney(value) {
  return `${new Intl.NumberFormat("es-ES", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(Number(value || 0))} €`;
}

function payrollFilename(month, extension) {
  const slug = String(month || "nomina")
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
  return `nomina-${slug || "periodo"}.${extension}`;
}

function buildPayrollPDF(calc, month) {
  const employee = payrollEmployeeData();
  const rows = payrollConceptRows(calc);
  const payrollCSV = "CSV-9382-AJ84-29E1-401C";
  const verificationURL = documentVerificationURL(payrollCSV);
  const money = (value) => formatPayrollMoney(value).replace(" €", " EUR");
  const shortText = (value, max) => {
    const text = String(value || "");
    return text.length > max ? `${text.slice(0, max - 3)}...` : text;
  };
  const lines = [
    "0.98 0.99 1 rg 36 36 523 770 re f",
    "0.78 0.84 0.88 RG 36 36 523 770 re S",
    "1 1 1 rg 45 736 505 72 re f",
    "0.72 0.80 0.88 RG 45 736 505 72 re S",
    "0.67 0.80 0.29 rg 45 736 505 4 re f",
    "0.92 0.96 0.92 rg 45 604 505 125 re f",
    "0.82 0.91 0.83 rg 45 706 505 23 re f",
    "0.72 0.80 0.88 RG 45 604 505 125 re S",
    "0.96 0.98 1 rg 45 120 505 66 re f",
    "0.72 0.80 0.88 RG 45 120 505 66 re S",
    "0 G 0 g",
  ];
  drawDiputacionLogoPDF(lines, 58, 780, 0.78);
  lines.push("0.09 0.23 0.31 rg");
  pdfLine(lines, 372, 788, "RECIBO DE SALARIOS", { size: 12, bold: true });
  pdfLine(lines, 372, 771, month.toUpperCase(), { size: 8, bold: true });

  lines.push("0.08 0.13 0.20 rg");
  pdfLine(lines, 58, 714, "DATOS DEL PERCEPTOR", { size: 11, bold: true });
  pdfLine(lines, 65, 690, `Empleado: ${employee.name}`, { size: 9 });
  pdfLine(lines, 330, 690, `NIF: ${employee.nif}`, { size: 9 });
  pdfLine(lines, 65, 673, `Centro de servicio: ${shortText(employee.service, 42)}`, { size: 9 });
  pdfLine(lines, 65, 656, `Puesto: ${employee.position}`, { size: 9 });
  pdfLine(lines, 330, 656, `Trienios: ${String(employee.trienios).padStart(2, "0")}`, { size: 9 });
  pdfLine(lines, 65, 639, `Relacion juridica: ${employee.relationship}`, { size: 9 });
  pdfLine(lines, 65, 622, `IBAN: ${employee.iban}`, { size: 9 });
  pdfLine(lines, 330, 622, `Afiliacion: ${employee.affiliation}`, { size: 9 });

  pdfLine(lines, 58, 585, "DEVENGOS Y DEDUCCIONES", { size: 11, bold: true });
  lines.push("0.10 0.29 0.47 rg 45 553 505 24 re f");
  lines.push("1 1 1 rg");
  pdfLine(lines, 58, 561, "CODIGO", { size: 8, bold: true });
  pdfLine(lines, 105, 561, "CONCEPTO", { size: 8, bold: true });
  pdfLine(lines, 390, 561, "DEVENGOS", { size: 8, bold: true });
  pdfLine(lines, 470, 561, "DEDUCCIONES", { size: 8, bold: true });

  let y = 531;
  rows.forEach((row, index) => {
    if (row.deduccion !== null) {
      lines.push("1 0.94 0.94 rg 45 " + (y - 5) + " 505 20 re f");
    } else if (row.code === "59") {
      lines.push("0.90 0.97 0.91 rg 45 " + (y - 5) + " 505 20 re f");
    } else if (index % 2 === 0) {
      lines.push("0.97 0.99 1 rg 45 " + (y - 5) + " 505 20 re f");
    } else {
      lines.push("0.94 0.97 0.95 rg 45 " + (y - 5) + " 505 20 re f");
    }
    lines.push("0.84 0.88 0.92 RG 45 " + (y - 7) + " 505 0 m 550 " + (y - 7) + " l S");
    lines.push("0.08 0.13 0.20 rg");
    pdfLine(lines, 60, y, row.code, { size: 8, bold: true });
    pdfLine(lines, 105, y, shortText(row.concept, 48), { size: 8 });
    pdfLine(lines, 382, y, row.devengo === null ? "-" : money(row.devengo), { size: 8 });
    if (row.deduccion !== null) lines.push("0.66 0.12 0.12 rg");
    pdfLine(lines, 470, y, row.deduccion === null ? "-" : money(row.deduccion), { size: 8 });
    y -= 22;
  });

  lines.push("0.91 0.96 1 rg 58 142 138 32 re f");
  lines.push("0.98 0.93 0.93 rg 225 142 138 32 re f");
  lines.push("0.90 0.97 0.91 rg 392 138 140 38 re f");
  lines.push("0.72 0.80 0.88 RG 58 142 138 32 re S");
  lines.push("0.72 0.80 0.88 RG 225 142 138 32 re S");
  lines.push("0.40 0.66 0.42 RG 392 138 140 38 re S");
  lines.push("0.08 0.13 0.20 rg");
  pdfLine(lines, 66, 163, "Total devengos", { size: 8, bold: true });
  pdfLine(lines, 66, 149, money(calc.devengos), { size: 11, bold: true });
  pdfLine(lines, 233, 163, "Total deducciones", { size: 8, bold: true });
  pdfLine(lines, 233, 149, money(calc.deducciones), { size: 11, bold: true });
  lines.push("0.08 0.34 0.13 rg");
  pdfLine(lines, 401, 162, "Liquido a percibir", { size: 8, bold: true });
  pdfLine(lines, 401, 146, money(calc.liquido), { size: 13, bold: true });

  lines.push("0.08 0.13 0.20 rg");
  pdfLine(lines, 58, 94, `CSV: ${payrollCSV}`, { size: 8 });
  pdfLine(lines, 58, 80, "Documento firmado electronicamente por la Diputacion Provincial de Granada", { size: 8 });
  drawQRCodePDF(lines, verificationURL, 482, 50, 1.35);
  pdfLine(lines, 374, 102, "Verificar documento", { size: 8, bold: true });
  pdfLine(lines, 374, 88, payrollCSV, { size: 7 });
  return buildSimplePDF(lines.join("\n"));
}

function printPayrollPDF(calc, month) {
  downloadBlob(buildPayrollPDF(calc, month), payrollFilename(month, "pdf"));
  recordReceipt("Nomina PDF", `${month} - recibo PDF descargado`, "nominas");
  setStatus(`Nomina de ${month} descargada en PDF`, "ready");
  if (state.portal) renderFlowPanel();
}

function exportPayrollExcel(calc, month) {
  const employee = payrollEmployeeData();
  const rows = payrollConceptRows(calc);
  const moneyCell = (value) => value === null ? "" : Number(value).toFixed(2);
  const excelRows = rows.map((row, index) => {
    const tone = row.deduccion !== null
      ? "deduction-row"
      : row.code === "59"
        ? "allowance-row"
        : index % 2 === 0 ? "pay-row-even" : "pay-row-odd";
    return `
    <tr class="${tone}">
      <td class="center">${escapeHTML(row.code)}</td>
      <td>${escapeHTML(row.concept)}</td>
      <td class="money">${moneyCell(row.devengo)}</td>
      <td class="money deduction">${moneyCell(row.deduccion)}</td>
      <td>${row.deduccion !== null ? "Deduccion" : "Devengo"}</td>
    </tr>
  `;
  }).join("");
  const html = `<!doctype html>
<html lang="es">
<head>
  <meta charset="utf-8">
  <style>
    body { margin: 0; font-family: Arial, sans-serif; color: #13202d; }
    table { border-collapse: collapse; font-size: 11pt; }
    col.code { width: 70px; }
    col.concept { width: 390px; }
    col.amount { width: 130px; }
    col.kind { width: 120px; }
    td, th { border: 1px solid #8ea9c1; padding: 7px 9px; vertical-align: middle; }
    .brand { background: #1b5e20; color: #ffffff; font-size: 16pt; font-weight: 700; }
    .brand-sub { background: #27425f; color: #ffffff; font-weight: 700; }
    .vec-logo { background: #e8f5e9; color: #1b5e20; font-size: 14pt; font-weight: 700; text-align: center; }
    .section { background: #d9ead3; color: #1b5e20; font-weight: 700; text-transform: uppercase; }
    .meta-label { background: #edf2f7; color: #334155; font-weight: 700; }
    .meta-value { background: #f8fafc; }
    .header th { background: #0f476f; color: #ffffff; font-weight: 700; text-align: center; }
    .pay-row-even td { background: #f7fbff; }
    .pay-row-odd td { background: #eef7ef; }
    .allowance-row td { background: #e2f0d9; color: #1b5e20; font-weight: 700; }
    .deduction-row td { background: #fde9e7; }
    .money { text-align: right; mso-number-format: "0.00"; }
    .deduction { color: #a52727; font-weight: 700; }
    .center { text-align: center; }
    .total-label { background: #edf2f7; font-weight: 700; text-align: right; }
    .total-devengo { background: #ddebf7; font-weight: 700; text-align: right; mso-number-format: "0.00"; }
    .total-deduccion { background: #f4cccc; color: #a52727; font-weight: 700; text-align: right; mso-number-format: "0.00"; }
    .total-neto { background: #d9ead3; color: #1b5e20; font-size: 13pt; font-weight: 700; text-align: right; mso-number-format: "0.00"; }
    .note { background: #f8fafc; color: #64748b; font-size: 9pt; }
  </style>
</head>
<body>
  <table>
    <colgroup>
      <col class="code">
      <col class="concept">
      <col class="amount">
      <col class="amount">
      <col class="kind">
    </colgroup>
    <tbody>
      <tr>
        <td class="vec-logo">VEC</td>
        <td colspan="4" class="brand">DIPUTACION PROVINCIAL DE GRANADA</td>
      </tr>
      <tr>
        <td class="brand-sub">Nomina</td>
        <td colspan="4" class="brand-sub">Recibo de salarios - ${escapeHTML(month)}</td>
      </tr>
      <tr><td colspan="5" class="section">Datos del perceptor</td></tr>
      <tr>
        <td class="meta-label">Empleado</td>
        <td colspan="2" class="meta-value">${escapeHTML(employee.name)}</td>
        <td class="meta-label">NIF</td>
        <td class="meta-value">${escapeHTML(employee.nif)}</td>
      </tr>
      <tr>
        <td class="meta-label">Puesto</td>
        <td colspan="2" class="meta-value">${escapeHTML(employee.position)}</td>
        <td class="meta-label">Trienios</td>
        <td class="meta-value center">${String(employee.trienios).padStart(2, "0")}</td>
      </tr>
      <tr>
        <td class="meta-label">Centro</td>
        <td colspan="4" class="meta-value">${escapeHTML(employee.service)}</td>
      </tr>
      <tr><td colspan="5" class="section">Devengos y deducciones</td></tr>
      <tr class="header">
        <th>Codigo</th>
        <th>Concepto</th>
        <th>Devengo</th>
        <th>Deduccion</th>
        <th>Tipo</th>
      </tr>
      ${excelRows}
      <tr>
        <td colspan="2" class="total-label">TOTAL DEVENGOS</td>
        <td class="total-devengo">${calc.devengos.toFixed(2)}</td>
        <td></td>
        <td></td>
      </tr>
      <tr>
        <td colspan="2" class="total-label">TOTAL DEDUCCIONES</td>
        <td></td>
        <td class="total-deduccion">${calc.deducciones.toFixed(2)}</td>
        <td></td>
      </tr>
      <tr>
        <td colspan="2" class="total-label">LIQUIDO A PERCIBIR</td>
        <td colspan="2" class="total-neto">${calc.liquido.toFixed(2)}</td>
        <td></td>
      </tr>
      <tr>
        <td colspan="5" class="note">CSV: CSV-9382-AJ84-29E1-401C · Exportacion VEC para tratamiento de datos de nomina.</td>
      </tr>
    </tbody>
  </table>
</body>
</html>`;
  const blob = new Blob(["\ufeff", html], { type: "application/vnd.ms-excel;charset=utf-8" });
  downloadBlob(blob, payrollFilename(month, "xls"));
  recordReceipt("Exportacion Excel", `${month} - ${rows.length} conceptos`, "nominas");
  setStatus(`Nomina de ${month} exportada a Excel`, "ready");
  if (state.portal) renderFlowPanel();
}

function downloadBlob(blob, filename) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.append(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

function pdfText(value) {
  return String(value ?? "")
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .replace(/[^\x20-\x7E]/g, " ")
    .replace(/[\\()]/g, "\\$&");
}

function pdfLine(parts, x, y, text, options = {}) {
  const size = options.size || 10;
  const font = options.bold ? "F2" : "F1";
  parts.push(`BT /${font} ${size} Tf ${x} ${y} Td (${pdfText(text)}) Tj ET`);
}

function drawDiputacionLogoPDF(parts, x, y, scale = 1) {
  const s = Number(scale || 1);
  const p = (value) => value.toFixed(3);
  const markScale = 0.25 * s;
  const e = x - 106.13 * markScale;
  const f = y + 212.04 * markScale;
  parts.push(`q ${p(markScale)} 0 0 ${p(-markScale)} ${p(e)} ${p(f)} cm`);
  parts.push("0.67 0.80 0.29 rg 264.19 212.04 m 191.23 285.00 l 219.60 313.37 l 235.82 297.16 l 223.80 285.14 l 264.33 244.61 l 288.65 268.93 l 207.59 349.99 l 138.55 280.95 l 149.84 269.66 l 153.32 266.18 157.95 264.26 162.87 264.26 c 167.79 264.26 172.42 266.18 175.90 269.66 c 179.08 272.84 l 195.29 256.63 l 192.11 253.45 l 184.30 245.64 173.91 241.34 162.87 241.34 c 151.83 241.34 141.44 245.64 133.63 253.45 c 106.13 280.95 l 207.60 382.41 l 321.09 268.93 l 264.21 212.05 l h f Q");
  parts.push(`0.09 0.23 0.31 rg`);
  pdfLine(parts, x + 62 * s, y + 11 * s, "Diputacion de Granada", { size: 15 * s, bold: true });
  pdfLine(parts, x + 63 * s, y - 6 * s, "Area de Recursos Humanos y Regimen Interior", { size: 7.5 * s });
  parts.push(`0.67 0.80 0.29 rg ${p(x + 63 * s)} ${p(y - 12 * s)} ${p(156 * s)} ${p(2 * s)} re f`);
}

const SIGNED_DOCUMENTS = {
  "CSV-9382-AJ84-29E1-401C": {
    title: "Recibo de salarios",
    issuer: "Diputacion Provincial de Granada",
    state: "Valido",
  },
  "CSV-CERT-10T-2025-9988-81A2": {
    title: "Certificado de retenciones IRPF 2025",
    issuer: "Diputacion Provincial de Granada",
    state: "Valido demo",
  },
};

function documentVerificationURL(csv) {
  const origin = window.location?.origin || "http://127.0.0.1:18180";
  return `${origin}/?v=${encodeURIComponent(csv)}`;
}

function qrGFMul(x, y) {
  let z = 0;
  for (let i = 7; i >= 0; i--) {
    z = ((z << 1) ^ (((z >>> 7) & 1) * 0x11d)) & 0xff;
    z ^= ((y >>> i) & 1) * x;
  }
  return z;
}

function qrReedSolomonDivisor(degree) {
  const result = Array(degree).fill(0);
  result[degree - 1] = 1;
  let root = 1;
  for (let i = 0; i < degree; i++) {
    for (let j = 0; j < result.length; j++) {
      result[j] = qrGFMul(result[j], root);
      if (j + 1 < result.length) {
        result[j] ^= result[j + 1];
      }
    }
    root = qrGFMul(root, 0x02);
  }
  return result;
}

function qrReedSolomonRemainder(data, divisor) {
  const result = Array(divisor.length).fill(0);
  data.forEach((value) => {
    const factor = value ^ result.shift();
    result.push(0);
    divisor.forEach((coef, index) => {
      result[index] ^= qrGFMul(coef, factor);
    });
  });
  return result;
}

function qrAppendBits(bits, value, length) {
  for (let i = length - 1; i >= 0; i--) {
    bits.push((value >>> i) & 1);
  }
}

function qrFormatBits(mask) {
  const data = (1 << 3) | mask; // ECC L = 01.
  let rem = data;
  for (let i = 0; i < 10; i++) {
    rem = (rem << 1) ^ (((rem >>> 9) & 1) * 0x537);
  }
  return ((data << 10) | (rem & 0x3ff)) ^ 0x5412;
}

function qrMatrix(text) {
  const version = 5;
  const size = version * 4 + 17;
  const dataCodewords = 108;
  const ecCodewords = 26;
  const bytes = Array.from(new TextEncoder().encode(String(text)));
  if (bytes.length > dataCodewords - 2) {
    throw new Error("El texto del QR excede la capacidad configurada");
  }
  const bits = [];
  qrAppendBits(bits, 0x4, 4);
  qrAppendBits(bits, bytes.length, 8);
  bytes.forEach((value) => qrAppendBits(bits, value, 8));
  const capacityBits = dataCodewords * 8;
  qrAppendBits(bits, 0, Math.min(4, capacityBits - bits.length));
  while (bits.length % 8 !== 0) bits.push(0);
  const data = [];
  for (let i = 0; i < bits.length; i += 8) {
    data.push(bits.slice(i, i + 8).reduce((sum, bit) => (sum << 1) | bit, 0));
  }
  for (let pad = 0xec; data.length < dataCodewords; pad ^= 0xfd) {
    data.push(pad);
  }
  const codewords = [...data, ...qrReedSolomonRemainder(data, qrReedSolomonDivisor(ecCodewords))];
  const modules = Array.from({ length: size }, () => Array(size).fill(false));
  const reserved = Array.from({ length: size }, () => Array(size).fill(false));
  const setFunction = (x, y, dark) => {
    if (x < 0 || y < 0 || x >= size || y >= size) return;
    modules[y][x] = Boolean(dark);
    reserved[y][x] = true;
  };
  const drawFinder = (x, y) => {
    for (let dy = -1; dy <= 7; dy++) {
      for (let dx = -1; dx <= 7; dx++) {
        const xx = x + dx;
        const yy = y + dy;
        const inFinder = dx >= 0 && dx <= 6 && dy >= 0 && dy <= 6;
        const dark = inFinder && (dx === 0 || dx === 6 || dy === 0 || dy === 6 || (dx >= 2 && dx <= 4 && dy >= 2 && dy <= 4));
        setFunction(xx, yy, dark);
      }
    }
  };
  const drawFormat = (mask) => {
    const bitsValue = qrFormatBits(mask);
    const bit = (index) => ((bitsValue >>> index) & 1) !== 0;
    for (let i = 0; i <= 5; i++) setFunction(8, i, bit(i));
    setFunction(8, 7, bit(6));
    setFunction(8, 8, bit(7));
    setFunction(7, 8, bit(8));
    for (let i = 9; i < 15; i++) setFunction(14 - i, 8, bit(i));
    for (let i = 0; i < 8; i++) setFunction(size - 1 - i, 8, bit(i));
    for (let i = 8; i < 15; i++) setFunction(8, size - 15 + i, bit(i));
    setFunction(8, size - 8, true);
  };
  drawFinder(0, 0);
  drawFinder(size - 7, 0);
  drawFinder(0, size - 7);
  for (let i = 8; i < size - 8; i++) {
    setFunction(6, i, i % 2 === 0);
    setFunction(i, 6, i % 2 === 0);
  }
  const align = 30;
  for (let dy = -2; dy <= 2; dy++) {
    for (let dx = -2; dx <= 2; dx++) {
      setFunction(align + dx, align + dy, Math.max(Math.abs(dx), Math.abs(dy)) !== 1);
    }
  }
  setFunction(8, version * 4 + 9, true);
  drawFormat(0);
  const dataBits = codewords.flatMap((value) => Array.from({ length: 8 }, (_, index) => (value >>> (7 - index)) & 1));
  let bitIndex = 0;
  let upward = true;
  for (let right = size - 1; right >= 1; right -= 2) {
    if (right === 6) right = 5;
    for (let vertical = 0; vertical < size; vertical++) {
      const y = upward ? size - 1 - vertical : vertical;
      for (let column = 0; column < 2; column++) {
        const x = right - column;
        if (!reserved[y][x]) {
          modules[y][x] = bitIndex < dataBits.length && dataBits[bitIndex] === 1;
          bitIndex++;
        }
      }
    }
    upward = !upward;
  }
  for (let y = 0; y < size; y++) {
    for (let x = 0; x < size; x++) {
      if (!reserved[y][x] && (x + y) % 2 === 0) {
        modules[y][x] = !modules[y][x];
      }
    }
  }
  drawFormat(0);
  return modules;
}

function drawQRCodePDF(parts, text, x, y, moduleSize) {
  const modules = qrMatrix(text);
  const quiet = 4;
  const totalModules = modules.length + quiet * 2;
  const totalSize = totalModules * moduleSize;
  parts.push(`1 1 1 rg ${x} ${y} ${totalSize} ${totalSize} re f`);
  parts.push(`0.72 0.80 0.88 RG ${x} ${y} ${totalSize} ${totalSize} re S`);
  const rects = [];
  modules.forEach((row, rowIndex) => {
    row.forEach((dark, columnIndex) => {
      if (!dark) return;
      const rx = x + (columnIndex + quiet) * moduleSize;
      const ry = y + (modules.length + quiet - 1 - rowIndex) * moduleSize;
      rects.push(`${rx.toFixed(2)} ${ry.toFixed(2)} ${moduleSize.toFixed(2)} ${moduleSize.toFixed(2)} re`);
    });
  });
  if (rects.length) {
    parts.push(`0 0 0 rg ${rects.join(" ")} f`);
  }
}

function buildSimplePDF(content) {
  const objects = [
    "<< /Type /Catalog /Pages 2 0 R >>",
    "<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
    "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 4 0 R /F2 5 0 R >> >> /Contents 6 0 R >>",
    "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
    "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold >>",
    `<< /Length ${content.length} >>\nstream\n${content}\nendstream`,
  ];
  let pdf = "%PDF-1.4\n";
  const offsets = [];
  objects.forEach((object, index) => {
    offsets.push(pdf.length);
    pdf += `${index + 1} 0 obj\n${object}\nendobj\n`;
  });
  const xrefOffset = pdf.length;
  pdf += `xref\n0 ${objects.length + 1}\n0000000000 65535 f \n`;
  pdf += offsets.map((offset) => `${String(offset).padStart(10, "0")} 00000 n \n`).join("");
  pdf += `trailer\n<< /Size ${objects.length + 1} /Root 1 0 R >>\nstartxref\n${xrefOffset}\n%%EOF`;
  return new Blob([pdf], { type: "application/pdf" });
}

function buildRetencionesCertificatePDF() {
  const employee = payrollEmployeeData();
  const issuedAt = new Date().toLocaleString("es-ES");
  const certCSV = "CSV-CERT-10T-2025-9988-81A2";
  const verificationURL = documentVerificationURL(certCSV);
  const money = (value) => formatPayrollMoney(value).replace(" €", " EUR");
  const shortText = (value, max) => {
    const text = String(value || "");
    return text.length > max ? `${text.slice(0, max - 3)}...` : text;
  };
  const annualGross = 32090.64;
  const annualWithholding = 4011.33;
  const deductibleExpenses = 1508.26;
  const taxableBase = annualGross - deductibleExpenses;
  const lines = [
    "0.98 0.99 1 rg 36 36 523 770 re f",
    "0.78 0.84 0.88 RG 36 36 523 770 re S",
    "1 1 1 rg 45 736 505 72 re f",
    "0.72 0.80 0.88 RG 45 736 505 72 re S",
    "0.67 0.80 0.29 rg 45 736 505 4 re f",
    "0.96 0.98 0.96 rg 45 674 505 48 re f",
    "0.78 0.84 0.88 RG 45 674 505 48 re S",
    "0.90 0.96 0.90 rg 45 554 505 108 re f",
    "0.82 0.91 0.83 rg 45 639 505 23 re f",
    "0.78 0.84 0.88 RG 45 554 505 108 re S",
    "0.94 0.97 1 rg 45 388 505 150 re f",
    "0.78 0.84 0.88 RG 45 388 505 150 re S",
    "0.96 0.98 1 rg 45 203 505 98 re f",
    "0.78 0.84 0.88 RG 45 203 505 98 re S",
    "0 G 0 g",
  ];
  drawDiputacionLogoPDF(lines, 58, 782, 0.72);
  lines.push("0.09 0.23 0.31 rg");
  pdfLine(lines, 392, 787, "CERTIFICADO 10T", { size: 11, bold: true });
  pdfLine(lines, 392, 770, "EJERCICIO FISCAL 2025", { size: 8.5, bold: true });
  pdfLine(lines, 392, 754, "IRPF - Retenciones", { size: 8 });

  lines.push("0.08 0.13 0.20 rg");
  pdfLine(lines, 58, 704, "CERTIFICADO DE RETENCIONES E INGRESOS A CUENTA", { size: 13, bold: true });
  pdfLine(lines, 58, 686, "Modelo 10T - documento demo con firma electronica y CSV verificable", { size: 9 });

  pdfLine(lines, 58, 647, "DATOS DEL PERCEPTOR", { size: 11, bold: true });
  pdfLine(lines, 65, 622, `Nombre y apellidos: ${employee.name}`, { size: 8.8 });
  pdfLine(lines, 350, 622, `NIF: ${employee.nif}`, { size: 8.8 });
  pdfLine(lines, 65, 604, `Puesto: ${shortText(employee.position, 34)}`, { size: 8.8 });
  pdfLine(lines, 350, 604, `Relacion: ${shortText(employee.relationship, 20)}`, { size: 8.8 });
  pdfLine(lines, 65, 586, `Centro de servicio: ${shortText(employee.service, 54)}`, { size: 8.8 });
  pdfLine(lines, 350, 586, `Trienios: ${String(employee.trienios).padStart(2, "0")}`, { size: 8.8 });
  pdfLine(lines, 65, 568, `IBAN: ${employee.iban}`, { size: 8.8 });
  pdfLine(lines, 350, 568, `Afiliacion: ${employee.affiliation}`, { size: 8.8 });

  pdfLine(lines, 58, 523, "RENDIMIENTOS DEL TRABAJO", { size: 11, bold: true });
  lines.push("0.10 0.29 0.47 rg 58 486 474 24 re f");
  lines.push("1 1 1 rg");
  pdfLine(lines, 65, 494, "CONCEPTO", { size: 8.5, bold: true });
  pdfLine(lines, 405, 494, "IMPORTE ANUAL", { size: 8.5, bold: true });
  lines.push("0.72 0.78 0.84 RG 392 396 0 114 re S");
  [
    ["Percepciones integras satisfechas", annualGross, "#eef6ff"],
    ["Retenciones practicadas a cuenta del I.R.P.F.", annualWithholding, "#fff1f2"],
    ["Gastos deducibles: Seguridad Social / MUFACE", deductibleExpenses, "#f8fafc"],
  ].forEach((row, index) => {
    const y = 454 - index * 29;
    const fill = index === 0 ? "0.93 0.97 1 rg" : index === 1 ? "1 0.94 0.95 rg" : "0.97 0.99 1 rg";
    lines.push(`${fill} 58 ${y} 474 24 re f`);
    lines.push("0.84 0.88 0.92 RG 58 " + y + " 474 24 re S");
    lines.push("0.08 0.13 0.20 rg");
    pdfLine(lines, 65, y + 8, shortText(row[0], 50), { size: 8.8 });
    if (index === 1) lines.push("0.66 0.12 0.12 rg");
    pdfLine(lines, 433, y + 8, money(row[1]), { size: 8.8, bold: true });
  });

  lines.push("0.91 0.96 1 rg 58 319 140 44 re f");
  lines.push("0.98 0.93 0.93 rg 228 319 140 44 re f");
  lines.push("0.90 0.97 0.91 rg 398 319 134 44 re f");
  lines.push("0.72 0.80 0.88 RG 58 319 140 44 re S");
  lines.push("0.72 0.80 0.88 RG 228 319 140 44 re S");
  lines.push("0.40 0.66 0.42 RG 398 319 134 44 re S");
  lines.push("0.08 0.13 0.20 rg");
  pdfLine(lines, 66, 346, "Base despues de gastos", { size: 7.5, bold: true });
  pdfLine(lines, 66, 329, money(taxableBase), { size: 10, bold: true });
  pdfLine(lines, 236, 346, "Retenciones IRPF", { size: 7.5, bold: true });
  pdfLine(lines, 236, 329, money(annualWithholding), { size: 10, bold: true });
  lines.push("0.08 0.34 0.13 rg");
  pdfLine(lines, 406, 346, "Tipo efectivo", { size: 7.5, bold: true });
  pdfLine(lines, 406, 329, `${((annualWithholding / annualGross) * 100).toFixed(2)} %`, { size: 10, bold: true });

  lines.push("0.08 0.13 0.20 rg");
  pdfLine(lines, 58, 278, "FIRMA Y VERIFICACION", { size: 10.5, bold: true });
  pdfLine(lines, 65, 256, `CSV: ${certCSV}`, { size: 8.8, bold: true });
  pdfLine(lines, 65, 239, `Fecha de firma simulada: ${issuedAt}`, { size: 8.8 });
  pdfLine(lines, 65, 222, shortText("Firmado electronicamente por Diputacion Provincial de Granada (entorno demo)", 72), { size: 8.8 });
  pdfLine(lines, 65, 179, "Este PDF permite probar la descarga del certificado firmado en VEC.", { size: 8.2 });
  pdfLine(lines, 65, 164, "No sustituye al documento emitido por la plataforma oficial de firma.", { size: 8.2, bold: true });

  pdfLine(lines, 58, 92, `Sello de tiempo demo: FNMT-RCM / ${certCSV}`, { size: 8 });
  drawQRCodePDF(lines, verificationURL, 480, 54, 1.30);
  pdfLine(lines, 382, 98, "Verificar documento", { size: 8, bold: true });
  pdfLine(lines, 382, 84, certCSV, { size: 7 });
  return buildSimplePDF(lines.join("\n"));
}

function downloadRetencionesCertificatePDF() {
  downloadBlob(buildRetencionesCertificatePDF(), "certificado-retenciones-2025-firmado-demo.pdf");
  recordReceipt("Certificado retenciones PDF", "10T 2025 - firma demo descargada", "nominas");
  setStatus("Certificado de retenciones descargado", "ready");
  if (state.portal) renderFlowPanel();
}

const NOMINAS_CONTROL_DATA = {
  months: ["Enero", "Febrero", "Marzo", "Abril", "Mayo", "Junio", "Julio", "Agosto", "Septiembre", "Octubre", "Noviembre", "Diciembre"],
  expenses: [2860000, 2895000, 2910000, 2882000, 3126400, 3189500, 2930000, 2942000, 2921000, 2958000, 3015000, 3263000],
  plantillaStatus: [
    { label: "Funcionarios de carrera", count: 1248, percent: 42, color: "#0f766e" },
    { label: "Personal laboral", count: 868, percent: 29, color: "#2563eb" },
    { label: "Interinos / temporales", count: 731, percent: 25, color: "#7c3aed" },
    { label: "Programas y sustituciones", count: 121, percent: 4, color: "#d97706" },
  ],
  areaAbsences: [
    { area: "Centros sociales y residencias", active: 642, baja: 58, percent: 9.0, trend: "+1.8", days: 724, risk: "Alto", reason: "IT comun, turnos sin cubrir y permisos de conciliacion" },
    { area: "Medio ambiente y obras", active: 388, baja: 22, percent: 5.7, trend: "+0.6", days: 286, risk: "Medio", reason: "Accidentes leves, campanas y retenes" },
    { area: "Cultura, deportes y juventud", active: 214, baja: 9, percent: 4.2, trend: "-0.4", days: 91, risk: "Bajo", reason: "Bajas cortas y sustituciones previstas" },
    { area: "Administracion general", active: 702, baja: 26, percent: 3.7, trend: "-0.2", days: 312, risk: "Bajo", reason: "IT comun y asuntos propios" },
    { area: "Transformacion digital", active: 146, baja: 8, percent: 5.5, trend: "+1.1", days: 79, risk: "Medio", reason: "Bajas medicas y guardias TIC" },
    { area: "Carreteras y asistencia a municipios", active: 438, baja: 34, percent: 7.8, trend: "+2.2", days: 421, risk: "Alto", reason: "Cuadrillas, disponibilidad y desplazamientos" },
    { area: "Intervencion y tesoreria", active: 102, baja: 2, percent: 2.0, trend: "-0.7", days: 18, risk: "Bajo", reason: "Cobertura ordinaria" },
  ],
  pendingGestiones: [
    { code: "1323", label: "Alta masiva y cambios de puesto", count: 24579, percent: 54.56, color: "#dc2626" },
    { code: "286", label: "Incidencias de calculo pendientes", count: 3057, percent: 6.79, color: "#0284c7" },
    { code: "287", label: "Variaciones de contrato sin revisar", count: 3056, percent: 6.78, color: "#65a30d" },
    { code: "4100", label: "Avisos de RPT / categoria", count: 2707, percent: 6.01, color: "#475569" },
    { code: "84", label: "Centros con gran volumen", count: 2501, percent: 5.55, color: "#f59e0b" },
    { code: "38066", label: "Expedientes de nomina retenidos", count: 2048, percent: 4.55, color: "#be185d" },
  ],
  reminders: [
    { label: "Cumpleanos", value: 8440 },
    { label: "Inicio bonif. sustitucion", value: 0 },
    { label: "Contratos", value: 1 },
    { label: "Fin IT interinidad", value: 0 },
    { label: "IT >18m", value: 1075 },
    { label: "Fin bonif. sustitucion", value: 0 },
    { label: "Fin mov. geografica", value: 0 },
    { label: "Hijo >12 anos", value: 107 },
    { label: "Fin relevo", value: 0 },
    { label: "Fin embargo", value: 43 },
    { label: "Empleados sin CEO", value: 25942 },
    { label: "Caducidad DNI/NIE", value: 0 },
    { label: "Encadenamiento", value: 269 },
    { label: "IT >12m", value: 928 },
    { label: "IT >15d", value: 0 },
  ],
  categoryControls: [
    { label: "P. extra sin remesar", value: 0 },
    { label: "IRPF anual sin presentar", value: 9 },
    { label: "Contrato TP sin parcialidad", value: 2035 },
    { label: "P. extra sin calculo", value: 249 },
    { label: "Ficheros Siltra pendientes", value: 9 },
    { label: "Afiliaciones", value: 1 },
    { label: "P. mensual sin calculo", value: 74108 },
    { label: "HS mensual sin listar", value: 17 },
    { label: "HS resto sin listar", value: 1 },
    { label: "HS extra sin listar", value: 0 },
    { label: "P. resto sin remesar", value: 0 },
    { label: "IRPF mensual sin presentar", value: 2 },
  ],
  finalizedControls: [
    { label: "Recalculo nominas Seg. Social", value: 0 },
    { label: "Ficheros Siltra sin conciliar", value: 0 },
    { label: "Recalculo nominas IRPF", value: 0 },
    { label: "P. mensual sin remesar", value: 17 },
  ],
  employees: [
    { name: "Ana Martin Ruiz", area: "RRHH", center: "Palacio Provincial", role: "Jefa de Seccion", regime: "Funcionaria carrera", group: "A1", rpt: "RPT-RRHH-0101", plaza: "PLZ-A1-0034", situation: "Servicio activo", seniority: "18 anos", trienios: 6, schedule: "Flexible RRHH", iban: "ES91 **** 6789", hours: 160, salary: 3480.75, status: "Pagada" },
    { name: "Jose Garcia Leon", area: "Centros sociales", center: "Residencia Rodriguez Penalva", role: "Auxiliar enfermeria", regime: "Laboral temporal", group: "C2", rpt: "RPT-TS-1042", plaza: "PLZ-C2-1187", situation: "IT sustituida", seniority: "7 anos", trienios: 2, schedule: "Turnos sin flexibilidad", iban: "ES22 **** 1205", hours: 152, salary: 1960.40, status: "Pendiente IT" },
    { name: "Maria Lopez Castro", area: "Intervencion", center: "Intervencion", role: "Tecnica A1", regime: "Funcionaria carrera", group: "A1", rpt: "RPT-INT-0008", plaza: "PLZ-A1-0040", situation: "Servicio activo", seniority: "22 anos", trienios: 7, schedule: "Flexible general", iban: "ES11 **** 4328", hours: 164, salary: 3895.20, status: "Pagada" },
    { name: "Rafael Ortega Perez", area: "Carreteras", center: "Parque movil", role: "Oficial conductor", regime: "Funcionario carrera", group: "C1", rpt: "RPT-OB-0211", plaza: "PLZ-C1-0244", situation: "Servicio activo", seniority: "31 anos", trienios: 10, schedule: "Reten y disponibilidad", iban: "ES45 **** 7731", hours: 176, salary: 2310.88, status: "Pendiente dieta" },
    { name: "Elena Jimenez Soto", area: "Transformacion digital", center: "Nuevas tecnologias", role: "Tecnica gestion A2", regime: "Funcionaria carrera", group: "A2", rpt: "RPT-2026-A2-042", plaza: "PLZ-2026-1187", situation: "Servicio activo", seniority: "12 anos", trienios: 4, schedule: "Flexible TIC", iban: "ES73 **** 0091", hours: 160, salary: 2874.16, status: "Pagada" },
  ],
  contracts: [
    { ref: "RPT-TS-1042", employee: "Jose Garcia Leon", type: "Interinidad por sustitucion", center: "Residencia Rodriguez Penalva", end: "30/06/2026", state: "Fin IT interinidad" },
    { ref: "RPT-OB-0211", employee: "Rafael Ortega Perez", type: "Funcionario carrera", center: "Parque movil", end: "Sin vencimiento", state: "Vigente" },
    { ref: "RPT-CU-0788", employee: "Laura Medina", type: "Programa temporal", center: "Cultura", end: "15/07/2026", state: "Proximo vencimiento" },
    { ref: "RPT-SO-1401", employee: "Isabel Torres", type: "Laboral fijo", center: "Servicios sociales comunitarios", end: "Sin vencimiento", state: "Reduccion jornada" },
  ],
  organization: [
    { ref: "ORG-LEGAL-01", type: "Organizacion legal", name: "Diputacion Provincial de Granada", scope: "Entidad pagadora", owner: "Secretaria / Intervencion", occupant: "Entidad", provision: "Estructura organica", requirements: "CIF P1800000J", budgetApp: "920/12000", state: "Vigente" },
    { ref: "ORG-INT-SS", type: "Organizacion interna", name: "Centros sociales y residencias", scope: "Area delegada", owner: "RRHH", occupant: "642 empleados", provision: "Plantilla estructural", requirements: "Turnos y sustituciones", budgetApp: "231/13000", state: "Con riesgo de cobertura" },
    { ref: "RPT-2026-A2-042", type: "Puesto RPT", name: "Tecnico de Gestion A2", scope: "Nivel 22 / especifico 680,44", owner: "Personal", occupant: "Elena Jimenez Soto", provision: "Concurso", requirements: "A2, administracion general", budgetApp: "920/12000", state: "Ocupado" },
    { ref: "PLZ-2026-1187", type: "Plaza", name: "Escala Administracion General", scope: "Subgrupo A2", owner: "Plantilla", occupant: "Vinculada a RPT-2026-A2-042", provision: "Dotacion presupuestaria", requirements: "Plaza estructural", budgetApp: "920/12000", state: "Dotada" },
    { ref: "EQ-OBRAS-03", type: "Equipo de trabajo", name: "Cuadrilla carreteras Poniente", scope: "Guardias y retenes", owner: "Servicio de Carreteras", occupant: "18 efectivos / 3 vacantes", provision: "Adscripcion operativa", requirements: "Permiso C y disponibilidad", budgetApp: "153/12101", state: "Cobertura parcial" },
  ],
  budgetApplications: [
    { app: "920/12000", service: "Administracion general", program: "Retribuciones basicas A1/A2", budget: 8420000, consumed: 4218800, forecast: "50,1%", state: "Normal" },
    { app: "231/13000", service: "Centros sociales", program: "Personal laboral residencias", budget: 18350000, consumed: 10340000, forecast: "56,3%", state: "Riesgo por sustituciones" },
    { app: "153/12101", service: "Carreteras", program: "Complementos y guardias", budget: 5210000, consumed: 3189000, forecast: "61,2%", state: "Vigilancia" },
    { app: "920/16000", service: "Seguridad Social", program: "Cuotas sociales", budget: 14200000, consumed: 7296000, forecast: "51,4%", state: "Normal" },
  ],
  payrollDefinitions: [
    { code: "PAGA-MENSUAL", block: "Pagas", name: "Nomina mensual ordinaria", value: "12 ciclos", state: "Abierta junio" },
    { code: "PAGA-EXTRA-JUN", block: "Pagas", name: "Paga extra junio", value: "Devengo 01/12-31/05", state: "Pendiente remesa" },
    { code: "TAB-TRIENIOS", block: "Tablas", name: "Trienios por grupo/subgrupo", value: "A1/A2/C1/C2/AP", state: "Vigente" },
    { code: "VAL-CD-22", block: "Valores", name: "Complemento destino nivel 22", value: "562,30 EUR", state: "Vigente" },
    { code: "CONC-IT", block: "Conceptos", name: "Prestacion IT / descuento", value: "Formula por contingencia", state: "Revisar 3 casos" },
    { code: "CONC-PROD", block: "Conceptos", name: "Productividad y gratificaciones", value: "Variable por expediente", state: "Pendiente aprobacion" },
  ],
  inspectorRules: [
    { check: "Neto negativo", affected: 0, severity: "Bloqueante", state: "Correcto", action: "Sin accion" },
    { check: "Variacion salarial anomala >25%", affected: 17, severity: "Alta", state: "Revisar", action: "Comparar mes anterior" },
    { check: "Trienios no actualizados", affected: 9, severity: "Media", state: "Pendiente", action: "Recalcular antiguedad" },
    { check: "Empleado sin cuenta bancaria", affected: 2, severity: "Bloqueante", state: "Pendiente", action: "Requerir IBAN" },
    { check: "IRPF cero o incoherente", affected: 6, severity: "Alta", state: "Revisar", action: "Regularizar IRPF" },
    { check: "Plaza sin aplicacion presupuestaria", affected: 4, severity: "Bloqueante", state: "Pendiente", action: "Asignar partida" },
    { check: "Incidencias incompatibles", affected: 11, severity: "Alta", state: "Revisar", action: "Cruzar Cronos/Personal" },
    { check: "Nomina calculada sin SLD validado", affected: 1, severity: "Bloqueante", state: "Parar cierre", action: "Validar SLD" },
  ],
  retroactivity: [
    { ref: "RETRO-2026-0041", employee: "Maria Lopez Castro", origin: "Revision salarial", period: "Ene-May 2026", amount: 842.50, state: "Calculada" },
    { ref: "RETRO-2026-0042", employee: "Jose Garcia Leon", origin: "Cambio situacion administrativa", period: "Mar-Jun 2026", amount: -126.30, state: "Revisar" },
    { ref: "RETRO-2026-0043", employee: "Elena Jimenez Soto", origin: "Trienio reconocido", period: "Abr-Jun 2026", amount: 148.77, state: "Lista para pago" },
    { ref: "REV-2026-0007", employee: "Colectivo A2", origin: "Revision valores tabla", period: "2026", amount: 32140.00, state: "Simulacion" },
  ],
  socialSecurity: [
    { channel: "AFI", file: "AFI-ALTAS-202606", records: 42, state: "Pendiente envio", error: "3 movimientos sin CCC" },
    { channel: "CRA", file: "CRA-202606", records: 2968, state: "Preparado", error: "Validacion pendiente" },
    { channel: "SLD", file: "SILTRA-202606", records: 2968, state: "Con diferencias", error: "17 tramos a conciliar" },
    { channel: "Contrat@", file: "CONTRATA-202606", records: 18, state: "Enviado", error: "Sin errores" },
    { channel: "Delt@", file: "DELTA-ACC-202606", records: 2, state: "Borrador", error: "Parte accidente pendiente" },
    { channel: "FDI", file: "FDI-IT-202606", records: 132, state: "Pendiente partes", error: "8 partes sin confirmar" },
  ],
  reports: [
    { name: "Informe de costes por area", scope: "Direccion / Intervencion", output: "Excel + PDF", state: "Disponible" },
    { name: "Certificado de haberes", scope: "Empleado", output: "PDF firmado CSV", state: "Plantilla vigente" },
    { name: "Certificado de servicios prestados", scope: "Bolsa / RRHH", output: "PDF firmado CSV", state: "Cruza Personal" },
    { name: "Modelo 190", scope: "AEAT", output: "Fichero anual", state: "Borrador 2026" },
    { name: "Libro de salarios", scope: "Auditoria", output: "Excel", state: "Pendiente cierre" },
    { name: "Analisis absentismo", scope: "Jefaturas", output: "Dashboard", state: "Actualizado" },
  ],
  loans: [
    { ref: "ANT-2026-0012", employee: "Ana Martin Ruiz", type: "Anticipo reintegrable", amount: 1800, quota: 150, pending: 900, state: "En curso" },
    { ref: "PRE-2026-0005", employee: "Rafael Ortega Perez", type: "Prestamo personal", amount: 3000, quota: 125, pending: 2625, state: "Aprobacion RRHH" },
    { ref: "EMB-2026-0044", employee: "Jose Garcia Leon", type: "Retencion judicial", amount: 1240, quota: 206.67, pending: 620, state: "Aplicado en nomina" },
  ],
  socialFund: [
    { ref: "FS-2026-118", employee: "Laura Medina", aid: "Ayuda estudios hijos", amount: 420, state: "Pendiente justificante", decision: "Requerir documento" },
    { ref: "FS-2026-119", employee: "Isabel Torres", aid: "Tratamiento odontologico", amount: 310, state: "Aceptada", decision: "Incluir en pago" },
    { ref: "FS-2026-120", employee: "Elena Jimenez Soto", aid: "Ayuda discapacidad familiar", amount: 620, state: "Revision social", decision: "Informe pendiente" },
  ],
  employeePortal: [
    { tile: "Mi informacion", detail: "Datos personales, bancarios, puesto RPT y situacion administrativa", state: "Verificado", action: "Abrir expediente" },
    { tile: "Ultimos recibos", detail: "Nomina mensual, atrasos y certificados descargables", state: "3 documentos", action: "Descargar" },
    { tile: "Vacaciones y ausencias", detail: "Solicitud, aprobacion, saldos y calendario Cronos", state: "2 pendientes", action: "Solicitar" },
    { tile: "Quien es quien", detail: "Directorio interno, unidad, extension y responsable", state: "Activo", action: "Buscar" },
    { tile: "Tareas y notificaciones", detail: "Requerimientos de justificante, firma y lectura", state: "4 avisos", action: "Revisar" },
  ],
  payments: [
    { lot: "NOM-2026-06", concept: "Nomina ordinaria junio", amount: 3189500, date: "25/06/2026", state: "Preparada" },
    { lot: "DIET-2026-06", concept: "Dietas aprobadas y locomocion", amount: 42680.55, date: "27/06/2026", state: "Pendiente fiscalizacion" },
    { lot: "ATR-2026-02", concept: "Atrasos y regularizaciones", amount: 18640.90, date: "30/06/2026", state: "En revision" },
  ],
  serviceCenter: [
    { ref: "CASO-2026-3812", subject: "Error en tramo SLD", requester: "Tecnico RRHH", sla: "4 h", state: "En diagnostico", owner: "Soporte nominas" },
    { ref: "CASO-2026-3813", subject: "Alta usuario jefatura", requester: "Servicio de Carreteras", sla: "8 h", state: "Pendiente autorizacion", owner: "Administrador VEC" },
    { ref: "CASO-2026-3814", subject: "Recibo no descargado", requester: "Empleado", sla: "24 h", state: "Resuelto", owner: "Atencion empleado" },
  ],
  peoplenetUsers: [
    { user: "rrhh.nominas", profile: "Tecnico RRHH", scope: "Nomina, RPT, SLD, IRPF", auth: "DNIe/certificado", lastAccess: "20/06/2026 08:12", state: "Activo" },
    { user: "adm.unidad", profile: "Administrativo operativo", scope: "Expedientes, contratos, ausencias", auth: "Clave + MFA", lastAccess: "19/06/2026 14:40", state: "Limitado" },
    { user: "jefatura.servicio", profile: "Validacion jerarquica", scope: "Ausencias, dietas, informes", auth: "Certificado", lastAccess: "18/06/2026 09:03", state: "Activo" },
  ],
};

function payrollFullControlAllowed() {
  const roles = currentRoleList();
  return isAdminSession() || roles.some((role) => ["tecnico_rrhh", "rrhh", "personal_rrhh"].includes(role));
}

function payrollOperationalAllowed() {
  const roles = currentRoleList();
  return payrollFullControlAllowed() || roles.some((role) => ["administrativo", "administrativo_unidad"].includes(role));
}

function payrollControlAllowed() {
  return payrollOperationalAllowed();
}

function defaultNominasScreen() {
  if (payrollFullControlAllowed()) return "calculo-retribuciones";
  if (payrollOperationalAllowed()) return "trabajadores-centros";
  return "portal-peoplenet";
}

function payrollMenuItems() {
  const employeeItems = [
    { id: "portal-peoplenet", label: "Portal empleado Peoplenet" },
    { id: "nomina-mes", label: "Nomina mensual" },
    { id: "historico-evolucion", label: "Historico y evolucion" },
    { id: "certificado-retenciones", label: "Certificado retenciones 10T" },
  ];
  if (!payrollControlAllowed()) return employeeItems;
  const operationalItems = [
    { id: "trabajadores-centros", label: "Expediente empleado publico" },
    { id: "organizacion-rpt", label: "Capitulo I, RPT y plazas" },
    { id: "resumen-control", label: "Resumen y estadisticas" },
    { id: "checklist-avisos", label: "Checklist y avisos" },
    { id: "contratos-vencimientos", label: "Contratos y vencimientos" },
    { id: "incapacidades-ausencias", label: "Incapacidades y ausencias" },
    { id: "estadisticas-bajas", label: "Bajas por areas" },
  ];
  if (!payrollFullControlAllowed()) return [...operationalItems, ...employeeItems];
  return [
    { id: "calculo-retribuciones", label: "Cierre mensual y calculo" },
    { id: "inspector-nomina", label: "Inspector de nomina" },
    { id: "comunicaciones-legales", label: "Cotizacion RED/SLD" },
    ...operationalItems.slice(0, 2),
    { id: "pagas-tablas", label: "Tablas, valores y conceptos" },
    { id: "retroactividad-revision", label: "Retroactividad y revision salarial" },
    { id: "irpf-acumulados", label: "IRPF, cotizacion y acumulados" },
    { id: "pagos-contabilidad", label: "Pagos y remesas" },
    ...operationalItems.slice(2),
    { id: "informes-certificados", label: "Informes, certificados y 190" },
    { id: "prestamos-fondo-social", label: "Prestamos, embargos y fondo social" },
    { id: "centro-servicio-usuarios", label: "Centro de servicio y usuarios" },
    ...employeeItems,
  ];
}

function formatInteger(value) {
  return new Intl.NumberFormat("es-ES", { maximumFractionDigits: 0 }).format(Number(value || 0));
}

function formatEuroCompact(value) {
  return `${new Intl.NumberFormat("es-ES", {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(moneyNumber(value))} €`;
}

const NOMINAS_WORKFLOW_DEFS = [
  {
    match: /Ejecutar proceso nomina|Abrir proceso de nominas|Revisar paso paga/i,
    title: "Flujo de cierre mensual y calculo",
    sensitive: true,
    steps: ["Preparar paga y alcance", "Validar incidencias de Personal, Cronos y Dietas", "Calcular bruto, IRPF y cotizacion", "Generar recibos y remesas", "Cerrar con recibo de auditoria"],
    blockers: ["17 avisos del inspector deben quedar resueltos antes del cierre definitivo."],
  },
  {
    match: /Resolver regla inspector|Ejecutar inspector|Avanzar cierre inspector/i,
    title: "Flujo del inspector de nomina",
    sensitive: true,
    steps: ["Abrir afectados", "Corregir maestro o incidencia origen", "Recalcular empleados afectados", "Liberar regla", "Emitir recibo de validacion"],
    blockers: ["Las reglas bloqueantes impiden generar remesa o recibos definitivos."],
  },
  {
    match: /Enviar comunicacion legal|Generar paquete comunicaciones|Abrir integracion legal/i,
    title: "Flujo RED, SILTRA y comunicaciones legales",
    sensitive: true,
    steps: ["Validar CCC, NAF y tramos", "Generar fichero AFI/CRA/SLD/Contrat@/Delt@/FDI", "Firmar paquete", "Registrar envio", "Conciliar acuse y diferencias"],
    blockers: ["No se permite enviar SLD con tramos sin conciliar."],
  },
  {
    match: /Preparar pago contable/i,
    title: "Flujo de pagos, remesa y contabilidad",
    sensitive: true,
    steps: ["Conciliar liquidos", "Generar remesa SEPA", "Enviar a fiscalizacion", "Contabilizar Capitulo I", "Publicar recibos pagados"],
    blockers: ["La remesa no sale hasta completar fiscalizacion e intervencion."],
  },
  {
    match: /Validar modelo 190|Emitir informe nomina|Generar informe certificado nomina/i,
    title: "Flujo de informes oficiales y modelo 190",
    sensitive: true,
    steps: ["Cruzar acumulados", "Validar AEAT", "Firmar salida", "Registrar CSV", "Publicar certificado o fichero"],
    blockers: ["El Modelo 190 queda en borrador mientras no cierre el ejercicio."],
  },
  {
    match: /Accion prestamos fondo social/i,
    title: "Flujo de prestamos, embargos y fondo social",
    sensitive: true,
    steps: ["Abrir expediente economico", "Validar resolucion o mandamiento", "Calcular cuota en nomina", "Notificar al empleado", "Enviar a pago o retencion"],
    blockers: ["Los embargos requieren prioridad legal y trazabilidad del saldo pendiente."],
  },
  {
    match: /Abrir empleado nomina|Exportar empleados nomina|Abrir contrato|Tramitar incapacidad ausencia|Abrir area bajas|Plan cobertura bajas|Cruzar RPT|Abrir elemento RPT|Exportar presupuesto nomina/i,
    title: "Flujo operativo de personal",
    sensitive: false,
    steps: ["Abrir expediente", "Validar datos origen", "Anotar actuacion", "Derivar si requiere RRHH", "Guardar recibo de auditoria"],
    blockers: ["Las modificaciones con impacto salarial se derivan a tecnico RRHH."],
  },
];

function nominasWorkflowForAction(action) {
  return NOMINAS_WORKFLOW_DEFS.find((item) => item.match.test(action || "")) || {
    title: "Flujo de tramitacion de nominas",
    sensitive: false,
    steps: ["Abrir actuacion", "Revisar datos", "Registrar recibo", "Notificar resultado"],
    blockers: [],
  };
}

function downloadNominasOperationalExport(action, detail) {
  const employee = payrollEmployeeData();
  const calc = getPayrollCalculations("Junio 2026");
  const rows = [
    ["accion", action],
    ["detalle", detail || "Nominas"],
    ["empleado", employee.name],
    ["nif", employee.nif],
    ["puesto", employee.position],
    ["centro", employee.service],
    ["devengos", calc.devengos.toFixed(2)],
    ["deducciones", calc.deducciones.toFixed(2)],
    ["liquido", calc.liquido.toFixed(2)],
    ["fecha", new Date().toLocaleString("es-ES")],
  ];
  const csv = rows.map((row) => row.map((value) => `"${String(value).replaceAll('"', '""')}"`).join(";")).join("\n");
  downloadBlob(new Blob(["\ufeff", csv], { type: "text/csv;charset=utf-8" }), `nominas-${slugify(action)}.csv`);
}

function downloadNominasOperationalReport(action, detail) {
  const calc = getPayrollCalculations("Junio 2026");
  const csv = portalDocumentCSV("NOM", detail || action);
  downloadSignedPortalPDF({
    title: String(action || "Informe de nominas"),
    subtitle: String(detail || "Informe operativo VEC"),
    ref: detail || action,
    csv,
    module: "Nominas",
    filename: `nominas-${slugify(action)}.pdf`,
    rows: [
      ["Periodo", "Junio 2026"],
      ["Detalle", detail || "-"],
      ["Devengos", formatCurrency(calc.devengos)],
      ["Deducciones", formatCurrency(calc.deducciones)],
      ["Liquido", formatCurrency(calc.liquido)],
      ["Actor", activeDemoUser().displayName],
      ["Estado", "Informe emitido en entorno demo"],
    ],
  });
}

function nominasAction(action, detail, rerender = false) {
  const workflow = nominasWorkflowForAction(action);
  if (workflow.sensitive && !payrollFullControlAllowed()) {
    recordReceipt("Accion reservada RRHH", `${action}: ${detail || "sin detalle"}`, "nominas");
    setStatus("Accion reservada a RRHH o administracion del sistema", "error");
    if (state.portal) renderFlowPanel();
    return;
  }
  const receipt = recordReceipt(action, detail || "Actuacion de nominas", "nominas");
  if (/exportar/i.test(action || "")) {
    downloadNominasOperationalExport(action, detail);
  } else if (/crear informe|generar informe|emitir informe/i.test(action || "")) {
    downloadNominasOperationalReport(action, detail);
  }
  state.nominasWorkflow = {
    action,
    detail: detail || "Actuacion de nominas",
    title: workflow.title,
    steps: workflow.steps,
    blockers: workflow.blockers || [],
    activeStep: 0,
    state: workflow.blockers?.length ? "Pendiente validacion" : "En curso",
    receipt: `${receipt.module}-${state.actionLog.length.toString().padStart(5, "0")}`,
    at: receipt.at,
  };
  setStatus(`${action}: flujo abierto`, "ready");
  if (state.portal) renderFlowPanel();
  if (state.portal) renderModulePortal(state.portal);
  else if (rerender && state.portal) renderModulePortal(state.portal);
}

function attachNominasActionButtons(root) {
  $$("[data-nominas-action]", root).forEach((button) => {
    button.addEventListener("click", () => {
      nominasAction(button.dataset.nominasAction, button.dataset.nominasDetail || button.textContent.trim());
    });
  });
  $$("[data-nominas-screen]", root).forEach((button) => {
    button.addEventListener("click", () => {
      state.nominasScreen = button.dataset.nominasScreen;
      if (state.portal) renderModulePortal(state.portal);
    });
  });
  $$("[data-portal-module]", root).forEach((button) => {
    button.addEventListener("click", () => {
      setActiveModule(button.dataset.portalModule);
    });
  });
  $$("[data-nominas-workflow]", root).forEach((button) => {
    button.addEventListener("click", () => {
      if (!state.nominasWorkflow) return;
      const command = button.dataset.nominasWorkflow;
      if (command === "advance") {
        state.nominasWorkflow.activeStep = Math.min(state.nominasWorkflow.activeStep + 1, state.nominasWorkflow.steps.length - 1);
        state.nominasWorkflow.state = state.nominasWorkflow.activeStep >= state.nominasWorkflow.steps.length - 1 ? "Listo para cierre" : "En curso";
        recordReceipt("Avanzar flujo nomina", `${state.nominasWorkflow.title}: ${state.nominasWorkflow.steps[state.nominasWorkflow.activeStep]}`, "nominas");
      } else if (command === "close") {
        recordReceipt("Cerrar flujo nomina", `${state.nominasWorkflow.title}: ${state.nominasWorkflow.detail}`, "nominas");
        state.nominasWorkflow.state = "Cerrado";
      } else if (command === "clear") {
        state.nominasWorkflow = null;
      }
      if (state.portal) renderModulePortal(state.portal);
    });
  });
}

function renderNominasWorkflowPanel() {
  const workflow = state.nominasWorkflow;
  if (!workflow) return "";
  const activeStep = Math.max(0, Math.min(workflow.activeStep || 0, workflow.steps.length - 1));
  const isClosed = workflow.state === "Cerrado";
  return `
    <section style="background:#fff; border:1px solid #bfdbfe; border-left:5px solid #2563eb; border-radius:8px; padding:14px; margin-bottom:16px;">
      <div style="display:flex; justify-content:space-between; gap:14px; align-items:flex-start; margin-bottom:12px;">
        <div>
          <h3 style="margin:0; color:#13202d; font-size:1rem;">${workflow.title}</h3>
          <p style="margin:4px 0 0; color:#607080; font-size:0.84rem;">${workflow.detail} · ${workflow.receipt} · ${workflow.at}</p>
        </div>
        ${renderNominasStatusBadge(workflow.state)}
      </div>
      ${workflow.blockers?.length ? `<div style="background:#fff7ed; border:1px solid #fed7aa; border-radius:7px; padding:9px; color:#9a3412; font-size:0.84rem; margin-bottom:12px;"><strong>Validacion:</strong> ${workflow.blockers.join(" ")}</div>` : ""}
      <ol style="display:grid; grid-template-columns:repeat(auto-fit, minmax(150px, 1fr)); gap:8px; list-style:none; padding:0; margin:0 0 12px;">
        ${workflow.steps.map((step, index) => {
          const done = isClosed || index < activeStep;
          const active = !isClosed && index === activeStep;
          const bg = done ? "#dcfce7" : active ? "#dbeafe" : "#f8fafc";
          const color = done ? "#166534" : active ? "#1d4ed8" : "#64748b";
          return `<li style="background:${bg}; border:1px solid ${active ? "#93c5fd" : "#e2e8f0"}; border-radius:7px; padding:9px; min-height:62px;">
            <strong style="display:block; color:${color}; font-size:0.78rem;">${index + 1}. ${done ? "Completado" : active ? "En curso" : "Pendiente"}</strong>
            <span style="display:block; margin-top:4px; color:#334155; font-size:0.82rem;">${step}</span>
          </li>`;
        }).join("")}
      </ol>
      <div style="display:flex; flex-wrap:wrap; gap:8px; justify-content:flex-end;">
        <button type="button" data-nominas-workflow="advance" ${isClosed ? "disabled" : ""} style="border:1px solid #93c5fd; background:#eff6ff; color:#1d4ed8; border-radius:6px; padding:8px 12px; cursor:pointer; font-weight:700;">Avanzar paso</button>
        <button type="button" data-nominas-workflow="close" ${isClosed ? "disabled" : ""} style="border:1px solid #86efac; background:#f0fdf4; color:#166534; border-radius:6px; padding:8px 12px; cursor:pointer; font-weight:700;">Cerrar flujo</button>
        <button type="button" data-nominas-workflow="clear" style="border:1px solid #cbd5e1; background:#fff; color:#475569; border-radius:6px; padding:8px 12px; cursor:pointer; font-weight:700;">Ocultar</button>
      </div>
    </section>`;
}

function renderPayrollBarChart(months, values) {
  const max = Math.max(...values, 1);
  return months.map((month, index) => {
    const h = Math.max(16, Math.round((values[index] / max) * 150));
    const alt = Math.max(8, Math.round(h * 0.62));
    return `
      <div style="display:flex; flex-direction:column; align-items:center; gap:8px; min-width:44px;">
        <div style="height:160px; display:flex; align-items:flex-end; gap:4px;">
          <span title="Coste ${month}: ${formatEuroCompact(values[index])}" style="width:14px; height:${h}px; background:#7c3aed; border-radius:4px 4px 0 0;"></span>
          <span title="Comparativo ${month}" style="width:14px; height:${alt}px; background:#dbeafe; border-radius:4px 4px 0 0;"></span>
        </div>
        <strong style="font-size:0.74rem; color:#64748b;">${month.slice(0, 3)}</strong>
      </div>`;
  }).join("");
}

function renderNominasPanelGrid(title, items, options = {}) {
  const tone = options.tone || "#0284c7";
  return `
    <section style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:16px;">
      <div style="display:flex; justify-content:space-between; align-items:center; gap:12px; margin-bottom:12px;">
        <h3 style="margin:0; color:#13202d; font-size:1rem;">${title}</h3>
        <button type="button" data-nominas-action="Revisar ${title}" data-nominas-detail="${title}" style="border:1px solid #cbd5e1; background:#fff; border-radius:6px; padding:6px 10px; cursor:pointer;">Abrir</button>
      </div>
      <div style="display:grid; grid-template-columns:repeat(auto-fit, minmax(118px, 1fr)); gap:8px;">
        ${items.map((item) => `
          <button type="button" data-nominas-action="Abrir aviso nominas" data-nominas-detail="${item.label}: ${item.value}" style="background:${tone}; color:#fff; border:none; border-radius:4px; min-height:62px; padding:8px; cursor:pointer; text-align:center;">
            <strong style="display:block; font-size:1.25rem;">${formatInteger(item.value)}</strong>
            <span style="font-size:0.72rem; line-height:1.15; display:block;">${item.label}</span>
          </button>
        `).join("")}
      </div>
    </section>`;
}

function renderNominasStatusBadge(text) {
  const value = text || "-";
  return `<span class="status-chip ${stateTone(value)}" style="white-space:nowrap;">${value}</span>`;
}

function renderNominasMetricCards(cards) {
  return `
    <div style="display:grid; grid-template-columns:repeat(auto-fit, minmax(180px, 1fr)); gap:12px; margin-bottom:16px;">
      ${cards.map(([label, value, note, color = "#2563eb"]) => `
        <article style="background:#fff; border:1px solid #d8e0e8; border-left:5px solid ${color}; border-radius:8px; padding:14px;">
          <span style="font-size:0.74rem; font-weight:700; color:#475569;">${label}</span>
          <strong style="display:block; margin-top:5px; color:${color}; font-size:1.35rem;">${value}</strong>
          <small style="display:block; margin-top:3px; color:#607080;">${note}</small>
        </article>`).join("")}
    </div>`;
}

function renderNominasScreenHeader(title, subtitle, actionLabel, action, detail) {
  return `
    <div style="display:flex; justify-content:space-between; gap:16px; align-items:flex-start; margin-bottom:16px;">
      <div>
        <h2 style="margin:0; color:#13202d;">${title}</h2>
        <p style="margin:4px 0 0; color:#607080; font-size:0.88rem;">${subtitle}</p>
      </div>
      ${actionLabel ? `<button type="button" data-nominas-action="${action}" data-nominas-detail="${detail}" style="background:#1b5e20; color:#fff; border:none; border-radius:6px; padding:10px 14px; font-weight:700; cursor:pointer;">${actionLabel}</button>` : ""}
    </div>`;
}

function renderNominasResumenControlScreen(target) {
  const totalExpense = NOMINAS_CONTROL_DATA.expenses[5];
  const nextExpense = NOMINAS_CONTROL_DATA.expenses[11];
  target.innerHTML = `
    <div style="display:flex; justify-content:space-between; gap:16px; align-items:flex-start; margin-bottom:18px;">
      <div>
        <h2 style="margin:0; color:#13202d;">Resumen operativo de nominas</h2>
        <p style="margin:4px 0 0; color:#607080; font-size:0.88rem;">Costes, plantilla, vencimientos, pagos y alertas relevantes de RRHH.</p>
      </div>
      <button type="button" data-nominas-action="Crear informe nominas" data-nominas-detail="Resumen operativo junio 2026" style="background:#1b5e20; color:#fff; border:none; border-radius:6px; padding:10px 14px; font-weight:700; cursor:pointer;">Crear informe</button>
    </div>
    <div style="display:grid; grid-template-columns:repeat(auto-fit, minmax(190px, 1fr)); gap:14px; margin-bottom:18px;">
      ${[
        ["Saldo presupuestario cap. I", formatEuroCompact(123350000.56), "30% incremento sobre mes anterior", "#ede9fe", "#6d28d9"],
        ["Gasto nomina junio", formatEuroCompact(totalExpense), "Nomina ordinaria y atrasos", "#e0f2fe", "#0369a1"],
        ["Proxima remesa salarial", formatEuroCompact(nextExpense), "2.968 empleados incluidos", "#dcfce7", "#15803d"],
        ["Fecha prevista de pago", "25/06/2026", "En 5 dias naturales", "#fef3c7", "#b45309"],
      ].map(([label, value, note, bg, color]) => `
        <article style="background:${bg}; border-left:5px solid ${color}; border-radius:8px; padding:14px;">
          <span style="font-size:0.72rem; font-weight:700; color:#475569;">${label}</span>
          <strong style="display:block; margin-top:6px; font-size:1.25rem; color:${color};">${value}</strong>
          <small style="display:block; margin-top:4px; color:#607080;">${note}</small>
        </article>`).join("")}
    </div>
    <div style="display:grid; grid-template-columns:minmax(0, 2fr) minmax(280px, 1fr); gap:18px;">
      <section style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:16px;">
        <div style="display:flex; justify-content:space-between; align-items:center; gap:12px; margin-bottom:10px;">
          <h3 style="margin:0; font-size:1rem;">Evolucion costes de empresa</h3>
          <select aria-label="Periodo costes nomina" style="border:1px solid #cbd5e1; border-radius:6px; padding:7px;">
            <option>01 Ene - 31 Dic 2026</option>
            <option>Mes actual</option>
            <option>Trimestre actual</option>
          </select>
        </div>
        <div style="display:flex; align-items:flex-end; gap:10px; overflow-x:auto; border-bottom:1px solid #d8e0e8; padding:10px 4px 2px;">
          ${renderPayrollBarChart(NOMINAS_CONTROL_DATA.months, NOMINAS_CONTROL_DATA.expenses)}
        </div>
      </section>
      <section style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:16px;">
        <h3 style="margin:0 0 12px; font-size:1rem;">Estado de plantilla</h3>
        <div style="display:flex; height:10px; overflow:hidden; border-radius:999px; background:#e2e8f0; margin-bottom:14px;">
          ${NOMINAS_CONTROL_DATA.plantillaStatus.map((item) => `<span style="width:${item.percent}%; background:${item.color};"></span>`).join("")}
        </div>
        <div style="display:grid; gap:9px;">
          ${NOMINAS_CONTROL_DATA.plantillaStatus.map((item) => `
            <button type="button" data-nominas-action="Filtrar plantilla" data-nominas-detail="${item.label}" style="display:flex; justify-content:space-between; align-items:center; gap:10px; background:#fff; border:1px solid #d8e0e8; border-radius:7px; padding:9px; cursor:pointer;">
              <span><i style="display:inline-block; width:8px; height:8px; border-radius:50%; background:${item.color}; margin-right:6px;"></i>${item.label}</span>
              <strong>${formatInteger(item.count)} | ${item.percent}%</strong>
            </button>`).join("")}
        </div>
      </section>
    </div>
    <div style="display:grid; grid-template-columns:repeat(auto-fit, minmax(280px, 1fr)); gap:18px; margin-top:18px;">
      ${renderNominasPanelGrid("Recordatorios finalizacion", NOMINAS_CONTROL_DATA.reminders.slice(0, 8))}
      ${renderNominasPanelGrid("Control de categorias", NOMINAS_CONTROL_DATA.categoryControls.slice(0, 8), { tone: "#0f766e" })}
      ${renderNominasPanelGrid("Procesos finalizados", NOMINAS_CONTROL_DATA.finalizedControls, { tone: "#475569" })}
    </div>
  `;
  attachNominasActionButtons(target);
}

function renderNominasTrabajadoresCentrosScreen(target) {
  target.innerHTML = `
    <h2 style="margin:0 0 14px; color:#13202d;">Expediente empleado publico</h2>
    <div style="display:grid; grid-template-columns:repeat(auto-fit, minmax(180px, 1fr)); gap:12px; margin-bottom:16px;">
      ${[
        ["Trabajadores en alta", 2968, "Incluidos en ciclo junio"],
        ["Trabajadores en baja", 159, "IT, excedencias y suspensiones"],
        ["Dias de ausencia mes", 1931, "Computo consolidado Cronos"],
        ["Centros con alerta", 22, "Necesitan revision de cobertura"],
      ].map(([label, value, note]) => `
        <article style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:14px;">
          <span style="font-weight:700; color:#475569; font-size:0.74rem;">${label}</span>
          <strong style="display:block; color:#1b5e20; font-size:1.5rem; margin-top:5px;">${formatInteger(value)}</strong>
          <small style="color:#607080;">${note}</small>
        </article>`).join("")}
    </div>
    <section style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:16px; overflow-x:auto;">
      <div style="display:flex; justify-content:space-between; align-items:center; gap:12px; margin-bottom:12px;">
        <h3 style="margin:0; font-size:1rem;">Listado operativo por empleado</h3>
        <button type="button" data-nominas-action="Exportar empleados nomina" data-nominas-detail="Plantilla y horas de junio" style="border:1px solid #9fc9a3; background:#fff; color:#1b5e20; border-radius:6px; padding:8px 12px; cursor:pointer; font-weight:700;">Exportar</button>
      </div>
      <table style="width:100%; min-width:1280px; border-collapse:collapse; font-size:0.82rem;">
        <thead><tr style="background:#edf2f7;"><th style="text-align:left; padding:9px;">Empleado</th><th style="text-align:left; padding:9px;">Regimen</th><th style="text-align:left; padding:9px;">Grupo</th><th style="text-align:left; padding:9px;">Puesto / plaza</th><th style="text-align:left; padding:9px;">Centro</th><th style="text-align:left; padding:9px;">Situacion</th><th style="text-align:left; padding:9px;">Antiguedad</th><th style="text-align:right; padding:9px;">Trienios</th><th style="text-align:left; padding:9px;">Jornada</th><th style="text-align:left; padding:9px;">IBAN</th><th style="text-align:right; padding:9px;">Horas</th><th style="text-align:right; padding:9px;">Total</th><th style="text-align:left; padding:9px;">Estado</th><th style="padding:9px;">Accion</th></tr></thead>
        <tbody>
          ${NOMINAS_CONTROL_DATA.employees.map((item, index) => `
            <tr style="background:${index % 2 ? "#f8fafc" : "#fff"};">
              <td style="padding:9px; border-top:1px solid #e2e8f0; font-weight:700;">${item.name}</td>
              <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.regime}</td>
              <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.group}</td>
              <td style="padding:9px; border-top:1px solid #e2e8f0;"><strong>${item.role}</strong><br><small>${item.rpt} · ${item.plaza}</small></td>
              <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.center}</td>
              <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.situation}</td>
              <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.seniority}</td>
              <td style="padding:9px; border-top:1px solid #e2e8f0; text-align:right;">${formatInteger(item.trienios)}</td>
              <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.schedule}</td>
              <td style="padding:9px; border-top:1px solid #e2e8f0; font-family:monospace;">${item.iban}</td>
              <td style="padding:9px; border-top:1px solid #e2e8f0; text-align:right;">${item.hours}</td>
              <td style="padding:9px; border-top:1px solid #e2e8f0; text-align:right; font-weight:700;">${formatEuroCompact(item.salary)}</td>
              <td style="padding:9px; border-top:1px solid #e2e8f0;">${renderNominasStatusBadge(item.status)}</td>
              <td style="padding:9px; border-top:1px solid #e2e8f0; text-align:center;"><button type="button" data-nominas-action="Abrir empleado nomina" data-nominas-detail="${item.name}" style="border:1px solid #cbd5e1; background:#fff; border-radius:5px; padding:5px 8px; cursor:pointer;">Abrir</button></td>
            </tr>`).join("")}
        </tbody>
      </table>
    </section>`;
  attachNominasActionButtons(target);
}

function renderNominasChecklistAvisosScreen(target) {
  target.innerHTML = `
    <h2 style="margin:0 0 14px; color:#13202d;">Checklist y tablero de avisos</h2>
    <div style="display:grid; grid-template-columns:220px minmax(0, 1fr); gap:16px;">
      <aside style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:14px;">
        <strong style="display:block; margin-bottom:10px;">Filtros</strong>
        <label style="display:block; font-size:0.78rem; margin-bottom:10px;">Empresa / entidad<select style="width:100%; margin-top:4px; padding:7px; border:1px solid #cbd5e1; border-radius:5px;"><option>Diputacion de Granada</option></select></label>
        <label style="display:block; font-size:0.78rem; margin-bottom:10px;">Centro<select style="width:100%; margin-top:4px; padding:7px; border:1px solid #cbd5e1; border-radius:5px;"><option>Todos los centros</option><option>Residencias</option><option>Carreteras</option></select></label>
        <label style="display:block; font-size:0.78rem; margin-bottom:10px;">Fecha inicio<input type="date" value="2026-06-01" style="width:100%; margin-top:4px; padding:7px; border:1px solid #cbd5e1; border-radius:5px;"></label>
        <label style="display:block; font-size:0.78rem; margin-bottom:12px;">Fecha fin<input type="date" value="2026-06-30" style="width:100%; margin-top:4px; padding:7px; border:1px solid #cbd5e1; border-radius:5px;"></label>
        <button type="button" data-nominas-action="Aplicar filtros avisos nomina" data-nominas-detail="Junio 2026" style="width:100%; background:#0284c7; color:#fff; border:none; border-radius:6px; padding:9px; cursor:pointer; font-weight:700;">Aplicar filtros</button>
        <button type="button" data-nominas-action="Limpiar filtros avisos nomina" data-nominas-detail="Tablero completo" style="width:100%; margin-top:10px; background:#fff; color:#b91c1c; border:1px solid #fecaca; border-radius:6px; padding:8px; cursor:pointer; font-weight:700;">Eliminar filtros</button>
      </aside>
      <div style="display:grid; gap:16px;">
        <section style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:16px;">
          <h3 style="margin:0 0 12px; font-size:1rem;">Gestiones pendientes</h3>
          ${NOMINAS_CONTROL_DATA.pendingGestiones.map((item) => `
            <button type="button" data-nominas-action="Abrir gestion pendiente" data-nominas-detail="${item.code} - ${item.label}" style="display:grid; grid-template-columns:220px minmax(0, 1fr) 130px; align-items:center; gap:10px; width:100%; border:none; background:#fff; padding:6px 0; cursor:pointer; text-align:left;">
              <span style="color:#475569;">${item.code} - ${item.label}</span>
              <span style="height:10px; background:#e2e8f0; border-radius:999px; overflow:hidden;"><i style="display:block; width:${item.percent}%; height:10px; background:${item.color};"></i></span>
              <strong style="text-align:right;">${formatInteger(item.count)} (${item.percent}%)</strong>
            </button>`).join("")}
        </section>
        <div style="display:grid; grid-template-columns:repeat(auto-fit, minmax(280px, 1fr)); gap:16px;">
          ${renderNominasPanelGrid("Recordatorios finalizacion", NOMINAS_CONTROL_DATA.reminders)}
          ${renderNominasPanelGrid("Control de categorias", NOMINAS_CONTROL_DATA.categoryControls, { tone: "#0f766e" })}
          ${renderNominasPanelGrid("Control de procesos finalizados", NOMINAS_CONTROL_DATA.finalizedControls, { tone: "#475569" })}
        </div>
      </div>
    </div>`;
  attachNominasActionButtons(target);
}

function renderNominasCalculoRetribucionesScreen(target) {
  const concepts = [
    ["Cierre mensual", "Abrir ciclo mensual, recalculo, revision de errores y cierre"],
    ["Calculo", "Precalcular bruto/neto, deducciones, atrasos y regularizaciones"],
    ["Retribuciones", "Sueldo, trienios, complementos, productividad y dietas integradas"],
    ["Ayudas", "Ayudas sociales, anticipos, reintegros y embargos"],
    ["Analisis cotizacion", "Bases de cotizacion, SILTRA y conciliacion Seguridad Social"],
    ["Finiquitos", "Liquidaciones por cese, vacaciones pendientes y pagas extra"],
    ["Actuaciones", "Correcciones con auditoria, firma y recibo de tramitacion"],
  ];
  target.innerHTML = `
    <h2 style="margin:0 0 14px; color:#13202d;">Cierre mensual, calculo y retribuciones</h2>
    <div style="display:grid; grid-template-columns:repeat(auto-fit, minmax(210px, 1fr)); gap:12px; margin-bottom:16px;">
      ${concepts.map(([title, detail]) => `
        <button type="button" data-nominas-action="Abrir proceso de nominas" data-nominas-detail="${title}" style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:13px; cursor:pointer; text-align:left; min-height:92px;">
          <strong style="display:block; color:#1b5e20; margin-bottom:6px;">${title}</strong>
          <span style="color:#607080; font-size:0.8rem;">${detail}</span>
        </button>`).join("")}
    </div>
    <section style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:16px;">
      <h3 style="margin:0 0 12px; font-size:1rem;">Cola de calculo junio 2026</h3>
      <table style="width:100%; border-collapse:collapse; font-size:0.86rem;">
        <thead><tr style="background:#edf2f7;"><th style="text-align:left; padding:9px;">Proceso</th><th style="text-align:left; padding:9px;">Origen</th><th style="text-align:right; padding:9px;">Registros</th><th style="text-align:left; padding:9px;">Estado</th><th style="padding:9px;">Accion</th></tr></thead>
        <tbody>
          ${[
            ["Nomina ordinaria", "Personal + Cronos", 2968, "Calculada con 17 avisos"],
            ["Dietas y locomocion", "Dietas", 184, "Pendiente fiscalizacion"],
            ["Reducciones 63/64", "Cronos + Personal", 36, "Aplicadas a jornada"],
            ["Trienios y antiguedad", "Personal", 91, "Recalculo proximo vencimiento"],
            ["Pagas extra", "Nominas", 2968, "Sin remesar"],
          ].map((row, index) => `
            <tr style="background:${index % 2 ? "#f8fafc" : "#fff"};"><td style="padding:9px; border-top:1px solid #e2e8f0; font-weight:700;">${row[0]}</td><td style="padding:9px; border-top:1px solid #e2e8f0;">${row[1]}</td><td style="padding:9px; border-top:1px solid #e2e8f0; text-align:right;">${formatInteger(row[2])}</td><td style="padding:9px; border-top:1px solid #e2e8f0;">${row[3]}</td><td style="padding:9px; border-top:1px solid #e2e8f0; text-align:center;"><button type="button" data-nominas-action="Ejecutar proceso nomina" data-nominas-detail="${row[0]}" style="border:1px solid #9fc9a3; background:#fff; color:#1b5e20; border-radius:5px; padding:5px 8px; cursor:pointer;">Ejecutar</button></td></tr>`).join("")}
        </tbody>
      </table>
    </section>`;
  attachNominasActionButtons(target);
}

function renderNominasContratosVencimientosScreen(target) {
  target.innerHTML = `
    <h2 style="margin:0 0 14px; color:#13202d;">Contratos, vencimientos y periodos previsibles</h2>
    <div style="display:grid; grid-template-columns:repeat(auto-fit, minmax(170px, 1fr)); gap:12px; margin-bottom:16px;">
      ${[
        ["Contratos en vigor", 2968],
        ["Proximos vencimientos", 18],
        ["Periodos previsibles", 42],
        ["Encadenamientos", 269],
        ["Fin bonificacion", 0],
      ].map(([label, value]) => `<article style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:14px;"><span style="font-size:0.74rem; font-weight:700; color:#475569;">${label}</span><strong style="display:block; margin-top:5px; color:#1b5e20; font-size:1.4rem;">${formatInteger(value)}</strong></article>`).join("")}
    </div>
    <section style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:16px;">
      <table style="width:100%; border-collapse:collapse; font-size:0.86rem;">
        <thead><tr style="background:#edf2f7;"><th style="text-align:left; padding:9px;">RPT / contrato</th><th style="text-align:left; padding:9px;">Empleado</th><th style="text-align:left; padding:9px;">Tipo</th><th style="text-align:left; padding:9px;">Centro</th><th style="text-align:left; padding:9px;">Vencimiento</th><th style="text-align:left; padding:9px;">Estado</th><th style="padding:9px;">Accion</th></tr></thead>
        <tbody>
          ${NOMINAS_CONTROL_DATA.contracts.map((item, index) => `
            <tr style="background:${index % 2 ? "#f8fafc" : "#fff"};"><td style="padding:9px; border-top:1px solid #e2e8f0; font-weight:700;">${item.ref}</td><td style="padding:9px; border-top:1px solid #e2e8f0;">${item.employee}</td><td style="padding:9px; border-top:1px solid #e2e8f0;">${item.type}</td><td style="padding:9px; border-top:1px solid #e2e8f0;">${item.center}</td><td style="padding:9px; border-top:1px solid #e2e8f0;">${item.end}</td><td style="padding:9px; border-top:1px solid #e2e8f0;">${item.state}</td><td style="padding:9px; border-top:1px solid #e2e8f0; text-align:center;"><button type="button" data-nominas-action="Abrir contrato" data-nominas-detail="${item.ref}" style="border:1px solid #cbd5e1; background:#fff; border-radius:5px; padding:5px 8px; cursor:pointer;">Abrir</button></td></tr>`).join("")}
        </tbody>
      </table>
    </section>`;
  attachNominasActionButtons(target);
}

function renderNominasIncapacidadesAusenciasScreen(target) {
  const cases = [
    ["IT-2026-1042", "Centros sociales y residencias", "IT comun", "02/06/2026", "Abierta", "Sustitucion requerida"],
    ["AUS-2026-771", "Carreteras y asistencia a municipios", "Ausencia justificada", "10/06/2026", "Validada", "Enviar a nomina"],
    ["IT-2026-1130", "Transformacion digital", "Baja medica", "12/06/2026", "Pendiente parte", "Requerir documento"],
    ["PER-2026-558", "Administracion general", "Permiso medico", "18/06/2026", "Cerrada", "Sin accion"],
  ];
  target.innerHTML = `
    <h2 style="margin:0 0 14px; color:#13202d;">Incapacidades, ausencias y partes de trabajo</h2>
    <div style="display:grid; grid-template-columns:repeat(auto-fit, minmax(180px, 1fr)); gap:12px; margin-bottom:16px;">
      ${[
        ["IT abiertas", 132],
        ["IT >12 meses", 928],
        ["IT >18 meses", 1075],
        ["Ausencias sin justificar", 27],
        ["Partes trabajo pendientes", 43],
      ].map(([label, value]) => `<button type="button" data-nominas-action="Filtrar incapacidad ausencia" data-nominas-detail="${label}" style="background:#fff; border:1px solid #d8e0e8; border-left:5px solid #b45309; border-radius:8px; padding:14px; text-align:left; cursor:pointer;"><span style="font-size:0.74rem; font-weight:700; color:#475569;">${label}</span><strong style="display:block; margin-top:5px; color:#b45309; font-size:1.4rem;">${formatInteger(value)}</strong></button>`).join("")}
    </div>
    <section style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:16px;">
      <table style="width:100%; border-collapse:collapse; font-size:0.86rem;">
        <thead><tr style="background:#edf2f7;"><th style="text-align:left; padding:9px;">Expediente</th><th style="text-align:left; padding:9px;">Area</th><th style="text-align:left; padding:9px;">Tipo</th><th style="text-align:left; padding:9px;">Inicio</th><th style="text-align:left; padding:9px;">Estado</th><th style="text-align:left; padding:9px;">Siguiente accion</th><th style="padding:9px;">Accion</th></tr></thead>
        <tbody>
          ${cases.map((row, index) => `<tr style="background:${index % 2 ? "#f8fafc" : "#fff"};">${row.map((cell) => `<td style="padding:9px; border-top:1px solid #e2e8f0;">${cell}</td>`).join("")}<td style="padding:9px; border-top:1px solid #e2e8f0; text-align:center;"><button type="button" data-nominas-action="Tramitar incapacidad ausencia" data-nominas-detail="${row[0]}" style="border:1px solid #cbd5e1; background:#fff; border-radius:5px; padding:5px 8px; cursor:pointer;">Tramitar</button></td></tr>`).join("")}
        </tbody>
      </table>
    </section>`;
  attachNominasActionButtons(target);
}

function renderNominasEstadisticasBajasScreen(target) {
  const maxPercent = Math.max(...NOMINAS_CONTROL_DATA.areaAbsences.map((item) => item.percent), 1);
  const average = NOMINAS_CONTROL_DATA.areaAbsences.reduce((sum, item) => sum + item.percent, 0) / NOMINAS_CONTROL_DATA.areaAbsences.length;
  target.innerHTML = `
    <div style="display:flex; justify-content:space-between; gap:16px; align-items:flex-start; margin-bottom:16px;">
      <div>
        <h2 style="margin:0; color:#13202d;">Estadisticas de bajas por areas</h2>
        <p style="margin:4px 0 0; color:#607080; font-size:0.88rem;">Porcentaje de personas en baja, dias acumulados y riesgo de cobertura de servicios.</p>
      </div>
      <button type="button" data-nominas-action="Generar informe bajas por areas" data-nominas-detail="Junio 2026" style="background:#1b5e20; color:#fff; border:none; border-radius:6px; padding:10px 14px; font-weight:700; cursor:pointer;">Generar informe</button>
    </div>
    <div style="display:grid; grid-template-columns:repeat(auto-fit, minmax(190px, 1fr)); gap:12px; margin-bottom:16px;">
      ${[
        ["Media provincial", `${average.toFixed(1)}%`, "Promedio ponderado demo"],
        ["Areas en riesgo alto", NOMINAS_CONTROL_DATA.areaAbsences.filter((item) => item.risk === "Alto").length, "Requieren plan de cobertura"],
        ["Dias de baja acumulados", NOMINAS_CONTROL_DATA.areaAbsences.reduce((sum, item) => sum + item.days, 0), "Junio 2026"],
        ["Personas actualmente en baja", NOMINAS_CONTROL_DATA.areaAbsences.reduce((sum, item) => sum + item.baja, 0), "Todas las areas"],
      ].map(([label, value, note]) => `<article style="background:#fff; border:1px solid #d8e0e8; border-left:5px solid #b45309; border-radius:8px; padding:14px;"><span style="font-size:0.74rem; font-weight:700; color:#475569;">${label}</span><strong style="display:block; margin-top:5px; color:#b45309; font-size:1.4rem;">${typeof value === "number" ? formatInteger(value) : value}</strong><small style="color:#607080;">${note}</small></article>`).join("")}
    </div>
    <section style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:16px; margin-bottom:16px;">
      <h3 style="margin:0 0 12px; font-size:1rem;">Comparativa visual de bajas</h3>
      <div style="display:grid; gap:10px;">
        ${NOMINAS_CONTROL_DATA.areaAbsences.map((item) => {
          const width = Math.max(8, Math.round((item.percent / maxPercent) * 100));
          const color = item.risk === "Alto" ? "#dc2626" : item.risk === "Medio" ? "#d97706" : "#15803d";
          return `
            <button type="button" data-nominas-action="Abrir area bajas" data-nominas-detail="${item.area}" style="display:grid; grid-template-columns:230px minmax(0, 1fr) 110px 95px; align-items:center; gap:10px; width:100%; background:#fff; border:none; padding:6px; cursor:pointer; text-align:left;">
              <strong style="font-size:0.84rem;">${item.area}</strong>
              <span style="height:18px; background:#e2e8f0; border-radius:999px; overflow:hidden;"><i style="display:block; width:${width}%; height:18px; background:${color};"></i></span>
              <span style="text-align:right; font-weight:700; color:${color};">${item.percent.toFixed(1)}%</span>
              <span style="font-size:0.75rem; color:${color};">${item.risk}</span>
            </button>`;
        }).join("")}
      </div>
    </section>
    <section style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:16px;">
      <table style="width:100%; border-collapse:collapse; font-size:0.85rem;">
        <thead><tr style="background:#edf2f7;"><th style="text-align:left; padding:9px;">Area / servicio</th><th style="text-align:right; padding:9px;">Activos</th><th style="text-align:right; padding:9px;">En baja</th><th style="text-align:right; padding:9px;">%</th><th style="text-align:right; padding:9px;">Dias</th><th style="text-align:left; padding:9px;">Tendencia</th><th style="text-align:left; padding:9px;">Motivo dominante</th><th style="padding:9px;">Accion</th></tr></thead>
        <tbody>
          ${NOMINAS_CONTROL_DATA.areaAbsences.map((item, index) => `
            <tr style="background:${index % 2 ? "#f8fafc" : "#fff"};">
              <td style="padding:9px; border-top:1px solid #e2e8f0; font-weight:700;">${item.area}</td>
              <td style="padding:9px; border-top:1px solid #e2e8f0; text-align:right;">${formatInteger(item.active)}</td>
              <td style="padding:9px; border-top:1px solid #e2e8f0; text-align:right;">${formatInteger(item.baja)}</td>
              <td style="padding:9px; border-top:1px solid #e2e8f0; text-align:right; font-weight:700;">${item.percent.toFixed(1)}%</td>
              <td style="padding:9px; border-top:1px solid #e2e8f0; text-align:right;">${formatInteger(item.days)}</td>
              <td style="padding:9px; border-top:1px solid #e2e8f0; color:${item.trend.startsWith("+") ? "#dc2626" : "#15803d"};">${item.trend} pp</td>
              <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.reason}</td>
              <td style="padding:9px; border-top:1px solid #e2e8f0; text-align:center;"><button type="button" data-nominas-action="Plan cobertura bajas" data-nominas-detail="${item.area}" style="border:1px solid #cbd5e1; background:#fff; border-radius:5px; padding:5px 8px; cursor:pointer;">Plan</button></td>
            </tr>`).join("")}
        </tbody>
      </table>
    </section>`;
  attachNominasActionButtons(target);
}

function renderNominasIRPFAcumuladosScreen(target) {
  target.innerHTML = `
    <h2 style="margin:0 0 14px; color:#13202d;">IRPF, cotizacion y acumulados</h2>
    <div style="display:grid; grid-template-columns:repeat(auto-fit, minmax(210px, 1fr)); gap:12px; margin-bottom:16px;">
      ${[
        ["IRPF mensual presentado", "98,7%", "2 avisos pendientes"],
        ["IRPF anual sin presentar", "9", "Control de categorias"],
        ["SILTRA pendiente", "9", "Ficheros por conciliar"],
        ["Acumulados revisados", "2.876", "Ejercicio 2026"],
      ].map(([label, value, note]) => `<article style="background:#fff; border:1px solid #d8e0e8; border-left:5px solid #2563eb; border-radius:8px; padding:14px;"><span style="font-size:0.74rem; font-weight:700; color:#475569;">${label}</span><strong style="display:block; margin-top:5px; color:#2563eb; font-size:1.4rem;">${value}</strong><small style="color:#607080;">${note}</small></article>`).join("")}
    </div>
    <section style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:16px;">
      <table style="width:100%; border-collapse:collapse; font-size:0.86rem;">
        <thead><tr style="background:#edf2f7;"><th style="text-align:left; padding:9px;">Control</th><th style="text-align:left; padding:9px;">Periodo</th><th style="text-align:right; padding:9px;">Importe / base</th><th style="text-align:left; padding:9px;">Estado</th><th style="padding:9px;">Accion</th></tr></thead>
        <tbody>
          ${[
            ["Modelo 111 IRPF", "Junio 2026", "411.320,40 EUR", "Borrador generado"],
            ["Modelo 190 acumulado", "2026", "2.876 perceptores", "Validando NIF"],
            ["Bases Seguridad Social", "Junio 2026", "2.944.820,10 EUR", "SILTRA pendiente"],
            ["Analisis cotizacion", "Junio 2026", "17 diferencias", "Revisar"],
          ].map((row, index) => `<tr style="background:${index % 2 ? "#f8fafc" : "#fff"};">${row.map((cell, cellIndex) => `<td style="padding:9px; border-top:1px solid #e2e8f0; ${cellIndex === 2 ? "text-align:right; font-weight:700;" : ""}">${cell}</td>`).join("")}<td style="padding:9px; border-top:1px solid #e2e8f0; text-align:center;"><button type="button" data-nominas-action="Abrir control IRPF cotizacion" data-nominas-detail="${row[0]}" style="border:1px solid #cbd5e1; background:#fff; border-radius:5px; padding:5px 8px; cursor:pointer;">Abrir</button></td></tr>`).join("")}
        </tbody>
      </table>
    </section>`;
  attachNominasActionButtons(target);
}

function renderNominasPagosContabilidadScreen(target) {
  target.innerHTML = `
    <h2 style="margin:0 0 14px; color:#13202d;">Pagos, remesas y contabilidad</h2>
    <div style="display:grid; grid-template-columns:repeat(auto-fit, minmax(220px, 1fr)); gap:12px; margin-bottom:16px;">
      <article style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:14px;"><span style="font-size:0.74rem; font-weight:700; color:#475569;">Cuenta de pago</span><strong style="display:block; margin-top:5px; color:#1b5e20; font-size:1.1rem;">Caja Rural Granada</strong><small style="color:#15803d;">Conectada</small></article>
      <article style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:14px;"><span style="font-size:0.74rem; font-weight:700; color:#475569;">Remesas SEPA pendientes</span><strong style="display:block; margin-top:5px; color:#b45309; font-size:1.4rem;">3</strong><small style="color:#607080;">Nomina, dietas y atrasos</small></article>
      <article style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:14px;"><span style="font-size:0.74rem; font-weight:700; color:#475569;">Fiscalizacion</span><strong style="display:block; margin-top:5px; color:#2563eb; font-size:1.4rem;">1 lote</strong><small style="color:#607080;">Dietas junio</small></article>
    </div>
    <section style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:16px;">
      <table style="width:100%; border-collapse:collapse; font-size:0.86rem;">
        <thead><tr style="background:#edf2f7;"><th style="text-align:left; padding:9px;">Lote</th><th style="text-align:left; padding:9px;">Concepto</th><th style="text-align:right; padding:9px;">Importe</th><th style="text-align:left; padding:9px;">Fecha</th><th style="text-align:left; padding:9px;">Estado SEPA / contable</th><th style="padding:9px;">Accion</th></tr></thead>
        <tbody>
          ${NOMINAS_CONTROL_DATA.payments.map((item, index) => `
            <tr style="background:${index % 2 ? "#f8fafc" : "#fff"};"><td style="padding:9px; border-top:1px solid #e2e8f0; font-weight:700;">${item.lot}</td><td style="padding:9px; border-top:1px solid #e2e8f0;">${item.concept}</td><td style="padding:9px; border-top:1px solid #e2e8f0; text-align:right; font-weight:700;">${formatEuroCompact(item.amount)}</td><td style="padding:9px; border-top:1px solid #e2e8f0;">${item.date}</td><td style="padding:9px; border-top:1px solid #e2e8f0;">${item.state}</td><td style="padding:9px; border-top:1px solid #e2e8f0; text-align:center;"><button type="button" data-nominas-action="Preparar pago contable" data-nominas-detail="${item.lot}" style="border:1px solid #9fc9a3; background:#fff; color:#1b5e20; border-radius:5px; padding:5px 8px; cursor:pointer;">Preparar</button></td></tr>`).join("")}
        </tbody>
      </table>
    </section>`;
  attachNominasActionButtons(target);
}

function renderNominasOrganizacionRPTScreen(target) {
  const totalBudget = NOMINAS_CONTROL_DATA.budgetApplications.reduce((sum, item) => sum + item.budget, 0);
  const consumedBudget = NOMINAS_CONTROL_DATA.budgetApplications.reduce((sum, item) => sum + item.consumed, 0);
  const budgetRisk = NOMINAS_CONTROL_DATA.budgetApplications.filter((item) => item.state !== "Normal").length;
  target.innerHTML = `
    ${renderNominasScreenHeader("Organizacion, RPT y presupuestos", "Capitulo I, estructura legal e interna, puestos RPT, plazas, equipos de trabajo y dotacion presupuestaria.", "Validar RPT", "Validar organizacion RPT", "RPT/plazas con presupuesto junio 2026")}
    ${renderNominasMetricCards([
      ["Elementos organizativos", formatInteger(NOMINAS_CONTROL_DATA.organization.length), "Legal, interna, RPT, plazas y equipos", "#0f766e"],
      ["Aplicaciones presupuestarias", formatInteger(NOMINAS_CONTROL_DATA.budgetApplications.length), "Partidas activas de Capitulo I", "#2563eb"],
      ["Credito consumido", formatEuroCompact(consumedBudget), `${((consumedBudget / totalBudget) * 100).toFixed(1)}% sobre credito`, "#7c3aed"],
      ["Partidas en vigilancia", formatInteger(budgetRisk), "Sustituciones, guardias o complementos", "#b45309"],
    ])}
    <div style="display:grid; grid-template-columns:240px minmax(0, 1fr); gap:16px; margin-bottom:16px;">
      <aside style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:14px;">
        <strong style="display:block; margin-bottom:10px;">Ambito</strong>
        <label style="display:block; font-size:0.78rem; margin-bottom:10px;">Entidad<select style="width:100%; margin-top:4px; padding:7px; border:1px solid #cbd5e1; border-radius:5px;"><option>Diputacion Provincial</option><option>Organismos dependientes</option></select></label>
        <label style="display:block; font-size:0.78rem; margin-bottom:10px;">Tipo<select style="width:100%; margin-top:4px; padding:7px; border:1px solid #cbd5e1; border-radius:5px;"><option>Todos</option><option>Puesto RPT</option><option>Plaza</option><option>Equipo de trabajo</option></select></label>
        <label style="display:block; font-size:0.78rem; margin-bottom:12px;">Estado<select style="width:100%; margin-top:4px; padding:7px; border:1px solid #cbd5e1; border-radius:5px;"><option>Todos</option><option>Dotada</option><option>Ocupado</option><option>Cobertura parcial</option></select></label>
        <button type="button" data-nominas-action="Cruzar RPT con presupuesto" data-nominas-detail="Capitulo I junio 2026" style="width:100%; background:#2563eb; color:#fff; border:none; border-radius:6px; padding:9px; cursor:pointer; font-weight:700;">Cruzar presupuesto</button>
      </aside>
      <section style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:16px; overflow-x:auto;">
        <h3 style="margin:0 0 12px; font-size:1rem;">Organizacion publica y RPT</h3>
        <table style="width:100%; border-collapse:collapse; font-size:0.82rem; min-width:1120px;">
          <thead><tr style="background:#edf2f7;"><th style="text-align:left; padding:9px;">Referencia</th><th style="text-align:left; padding:9px;">Tipo</th><th style="text-align:left; padding:9px;">Nombre</th><th style="text-align:left; padding:9px;">Alcance</th><th style="text-align:left; padding:9px;">Ocupante / dotacion</th><th style="text-align:left; padding:9px;">Provision</th><th style="text-align:left; padding:9px;">Requisitos</th><th style="text-align:left; padding:9px;">Aplicacion</th><th style="text-align:left; padding:9px;">Estado</th><th style="padding:9px;">Accion</th></tr></thead>
          <tbody>
            ${NOMINAS_CONTROL_DATA.organization.map((item, index) => `
              <tr style="background:${index % 2 ? "#f8fafc" : "#fff"};">
                <td style="padding:9px; border-top:1px solid #e2e8f0; font-weight:700;">${item.ref}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.type}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.name}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.scope}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.occupant}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.provision}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.requirements}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0; font-family:monospace;">${item.budgetApp}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${renderNominasStatusBadge(item.state)}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0; text-align:center;"><button type="button" data-nominas-action="Abrir elemento RPT" data-nominas-detail="${item.ref}" style="border:1px solid #cbd5e1; background:#fff; border-radius:5px; padding:5px 8px; cursor:pointer;">Abrir</button></td>
              </tr>`).join("")}
          </tbody>
        </table>
      </section>
    </div>
    <section style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:16px; overflow-x:auto;">
      <div style="display:flex; justify-content:space-between; align-items:center; gap:12px; margin-bottom:12px;">
        <h3 style="margin:0; font-size:1rem;">Aplicaciones presupuestarias y prevision</h3>
        <button type="button" data-nominas-action="Exportar presupuesto nomina" data-nominas-detail="Aplicaciones Capitulo I" style="border:1px solid #9fc9a3; background:#fff; color:#1b5e20; border-radius:6px; padding:8px 12px; cursor:pointer; font-weight:700;">Exportar</button>
      </div>
      <table style="width:100%; border-collapse:collapse; font-size:0.85rem; min-width:820px;">
        <thead><tr style="background:#edf2f7;"><th style="text-align:left; padding:9px;">Aplicacion</th><th style="text-align:left; padding:9px;">Servicio</th><th style="text-align:left; padding:9px;">Programa</th><th style="text-align:right; padding:9px;">Credito</th><th style="text-align:right; padding:9px;">Consumido</th><th style="text-align:left; padding:9px;">Prevision</th><th style="text-align:left; padding:9px;">Estado</th></tr></thead>
        <tbody>
          ${NOMINAS_CONTROL_DATA.budgetApplications.map((item, index) => `
            <tr style="background:${index % 2 ? "#f8fafc" : "#fff"};">
              <td style="padding:9px; border-top:1px solid #e2e8f0; font-weight:700;">${item.app}</td>
              <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.service}</td>
              <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.program}</td>
              <td style="padding:9px; border-top:1px solid #e2e8f0; text-align:right;">${formatEuroCompact(item.budget)}</td>
              <td style="padding:9px; border-top:1px solid #e2e8f0; text-align:right; font-weight:700;">${formatEuroCompact(item.consumed)}</td>
              <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.forecast}</td>
              <td style="padding:9px; border-top:1px solid #e2e8f0;">${renderNominasStatusBadge(item.state)}</td>
            </tr>`).join("")}
        </tbody>
      </table>
    </section>`;
  attachNominasActionButtons(target);
}

function renderNominasPagasTablasScreen(target) {
  const byBlock = NOMINAS_CONTROL_DATA.payrollDefinitions.reduce((acc, item) => {
    acc[item.block] = (acc[item.block] || 0) + 1;
    return acc;
  }, {});
  target.innerHTML = `
    ${renderNominasScreenHeader("Pagas, tablas, valores y conceptos", "Configuracion funcional que alimenta el calculo: pagas, tablas de valores, conceptos, unidades, precios e importes.", "Versionar tablas", "Versionar tablas nomina", "Tablas y conceptos 2026")}
    ${renderNominasMetricCards([
      ["Pagas definidas", formatInteger(byBlock.Pagas || 0), "Mensual ordinaria y extra", "#0f766e"],
      ["Tablas y valores", formatInteger((byBlock.Tablas || 0) + (byBlock.Valores || 0)), "Trienios, CD, IRPF y bases", "#2563eb"],
      ["Conceptos activos", formatInteger(byBlock.Conceptos || 0), "Totales, unidades e importes", "#7c3aed"],
      ["Avisos de configuracion", "3", "Pendientes antes del cierre", "#b45309"],
    ])}
    <div style="display:grid; grid-template-columns:minmax(0, 1.45fr) minmax(260px, 0.55fr); gap:16px;">
      <section style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:16px; overflow-x:auto;">
        <div style="display:flex; justify-content:space-between; align-items:center; gap:12px; margin-bottom:12px;">
          <h3 style="margin:0; font-size:1rem;">Catalogo de definicion de nomina</h3>
          <select aria-label="Filtrar bloque de definicion" style="border:1px solid #cbd5e1; border-radius:6px; padding:7px;"><option>Todos los bloques</option><option>Pagas</option><option>Tablas</option><option>Valores</option><option>Conceptos</option></select>
        </div>
        <table style="width:100%; border-collapse:collapse; font-size:0.86rem; min-width:720px;">
          <thead><tr style="background:#edf2f7;"><th style="text-align:left; padding:9px;">Codigo</th><th style="text-align:left; padding:9px;">Bloque</th><th style="text-align:left; padding:9px;">Nombre</th><th style="text-align:left; padding:9px;">Valor / formula</th><th style="text-align:left; padding:9px;">Estado</th><th style="padding:9px;">Accion</th></tr></thead>
          <tbody>
            ${NOMINAS_CONTROL_DATA.payrollDefinitions.map((item, index) => `
              <tr style="background:${index % 2 ? "#f8fafc" : "#fff"};">
                <td style="padding:9px; border-top:1px solid #e2e8f0; font-weight:700;">${item.code}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.block}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.name}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.value}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${renderNominasStatusBadge(item.state)}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0; text-align:center;"><button type="button" data-nominas-action="Abrir definicion nomina" data-nominas-detail="${item.code}" style="border:1px solid #cbd5e1; background:#fff; border-radius:5px; padding:5px 8px; cursor:pointer;">Abrir</button></td>
              </tr>`).join("")}
          </tbody>
        </table>
      </section>
      <aside style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:16px;">
        <h3 style="margin:0 0 12px; font-size:1rem;">Flujo de paga Peoplenet</h3>
        ${["Preparar paga y filtros", "Cargar incidencias y absentismos", "Validar tablas maestras", "Calcular bruto, IRPF y cotizacion", "Generar recibos, pago y contabilidad"].map((item, index) => `
          <button type="button" data-nominas-action="Revisar paso paga" data-nominas-detail="${item}" style="width:100%; display:flex; gap:9px; align-items:center; text-align:left; background:#fff; border:1px solid #e2e8f0; border-radius:7px; padding:9px; margin-bottom:8px; cursor:pointer;">
            <strong style="display:inline-grid; place-items:center; width:24px; height:24px; border-radius:50%; background:#e0f2fe; color:#0369a1;">${index + 1}</strong>
            <span>${item}</span>
          </button>`).join("")}
      </aside>
    </div>`;
  attachNominasActionButtons(target);
}

function renderNominasInspectorScreen(target) {
  const rules = NOMINAS_CONTROL_DATA.inspectorRules;
  const blockers = rules.filter((item) => item.severity === "Bloqueante");
  const affected = rules.reduce((sum, item) => sum + item.affected, 0);
  target.innerHTML = `
    ${renderNominasScreenHeader("Inspector de nomina", "Controles automaticos previos al cierre: netos negativos, variaciones, trienios, IRPF, IBAN, RPT y SLD.", "Ejecutar inspector", "Ejecutar inspector nomina", "Junio 2026 con ${formatInteger(rules.length)} reglas")}
    ${renderNominasMetricCards([
      ["Reglas ejecutadas", formatInteger(rules.length), "Catalogo Peoplenet AAPP", "#2563eb"],
      ["Incidencias afectadas", formatInteger(affected), "Personas o tramos a revisar", "#b45309"],
      ["Bloqueantes", formatInteger(blockers.length), "Impiden cierre automatico", "#dc2626"],
      ["Correctas", formatInteger(rules.filter((item) => item.state === "Correcto").length), "Sin accion pendiente", "#15803d"],
    ])}
    <div style="display:grid; grid-template-columns:minmax(0, 1fr) 280px; gap:16px;">
      <section style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:16px; overflow-x:auto;">
        <table style="width:100%; border-collapse:collapse; font-size:0.86rem; min-width:780px;">
          <thead><tr style="background:#edf2f7;"><th style="text-align:left; padding:9px;">Control</th><th style="text-align:right; padding:9px;">Afectados</th><th style="text-align:left; padding:9px;">Severidad</th><th style="text-align:left; padding:9px;">Estado</th><th style="text-align:left; padding:9px;">Siguiente accion</th><th style="padding:9px;">Accion</th></tr></thead>
          <tbody>
            ${rules.map((item, index) => `
              <tr style="background:${index % 2 ? "#f8fafc" : "#fff"};">
                <td style="padding:9px; border-top:1px solid #e2e8f0; font-weight:700;">${item.check}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0; text-align:right;">${formatInteger(item.affected)}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${renderNominasStatusBadge(item.severity)}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${renderNominasStatusBadge(item.state)}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.action}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0; text-align:center;"><button type="button" data-nominas-action="Resolver regla inspector" data-nominas-detail="${item.check}" style="border:1px solid #cbd5e1; background:#fff; border-radius:5px; padding:5px 8px; cursor:pointer;">Resolver</button></td>
              </tr>`).join("")}
          </tbody>
        </table>
      </section>
      <aside style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:16px;">
        <h3 style="margin:0 0 12px; font-size:1rem;">Orden de cierre</h3>
        ${["Corregir bloqueantes", "Recalcular afectados", "Conciliar SLD", "Emitir recibo de validacion", "Cerrar paga mensual"].map((item) => `
          <button type="button" data-nominas-action="Avanzar cierre inspector" data-nominas-detail="${item}" style="width:100%; border:1px solid #e2e8f0; background:#fff; border-radius:7px; padding:9px; margin-bottom:8px; cursor:pointer; text-align:left;">${item}</button>`).join("")}
      </aside>
    </div>`;
  attachNominasActionButtons(target);
}

function renderNominasRetroactividadScreen(target) {
  const total = NOMINAS_CONTROL_DATA.retroactivity.reduce((sum, item) => sum + item.amount, 0);
  const reviewCount = NOMINAS_CONTROL_DATA.retroactivity.filter((item) => /revisar|simulacion/i.test(item.state)).length;
  target.innerHTML = `
    ${renderNominasScreenHeader("Retroactividad y revision salarial", "Atrasos manuales o automaticos por revision salarial, trienios, cambios de situacion y tablas de valores.", "Simular revision", "Simular revision salarial", "Escenario 2026")}
    ${renderNominasMetricCards([
      ["Expedientes retroactivos", formatInteger(NOMINAS_CONTROL_DATA.retroactivity.length), "Abiertos en junio", "#2563eb"],
      ["Importe neto previsto", formatEuroCompact(total), "Suma de atrasos y descuentos", total < 0 ? "#dc2626" : "#15803d"],
      ["En revision", formatInteger(reviewCount), "Requieren validacion RRHH", "#b45309"],
      ["Listos para pago", formatInteger(NOMINAS_CONTROL_DATA.retroactivity.filter((item) => /lista|calculada/i.test(item.state)).length), "Pasan a nomina", "#0f766e"],
    ])}
    <section style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:16px; overflow-x:auto; margin-bottom:16px;">
      <table style="width:100%; border-collapse:collapse; font-size:0.86rem; min-width:780px;">
        <thead><tr style="background:#edf2f7;"><th style="text-align:left; padding:9px;">Expediente</th><th style="text-align:left; padding:9px;">Empleado/colectivo</th><th style="text-align:left; padding:9px;">Origen</th><th style="text-align:left; padding:9px;">Periodo</th><th style="text-align:right; padding:9px;">Importe</th><th style="text-align:left; padding:9px;">Estado</th><th style="padding:9px;">Accion</th></tr></thead>
        <tbody>
          ${NOMINAS_CONTROL_DATA.retroactivity.map((item, index) => {
            const amount = `${item.amount < 0 ? "-" : ""}${formatEuroCompact(Math.abs(item.amount))}`;
            return `
              <tr style="background:${index % 2 ? "#f8fafc" : "#fff"};">
                <td style="padding:9px; border-top:1px solid #e2e8f0; font-weight:700;">${item.ref}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.employee}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.origin}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.period}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0; text-align:right; font-weight:700; color:${item.amount < 0 ? "#dc2626" : "#15803d"};">${amount}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${renderNominasStatusBadge(item.state)}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0; text-align:center;"><button type="button" data-nominas-action="Validar retroactividad" data-nominas-detail="${item.ref}" style="border:1px solid #cbd5e1; background:#fff; border-radius:5px; padding:5px 8px; cursor:pointer;">Validar</button></td>
              </tr>`;
          }).join("")}
        </tbody>
      </table>
    </section>
    <section style="display:grid; grid-template-columns:repeat(auto-fit, minmax(210px, 1fr)); gap:12px;">
      ${["Alta de expediente", "Calculo automatico", "Comparativa mes anterior", "Revision juridica", "Incluir en paga"].map((item, index) => `
        <button type="button" data-nominas-action="Abrir paso retroactividad" data-nominas-detail="${item}" style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:13px; cursor:pointer; text-align:left; min-height:82px;">
          <strong style="display:block; color:#7c3aed;">${index + 1}. ${item}</strong>
          <span style="display:block; margin-top:6px; color:#607080; font-size:0.8rem;">Trazabilidad de revision salarial con recibo de calculo.</span>
        </button>`).join("")}
    </section>`;
  attachNominasActionButtons(target);
}

function renderNominasComunicacionesLegalesScreen(target) {
  const records = NOMINAS_CONTROL_DATA.socialSecurity.reduce((sum, item) => sum + item.records, 0);
  const pending = NOMINAS_CONTROL_DATA.socialSecurity.filter((item) => !/sin errores/i.test(item.error) || /pendiente|diferencias|borrador/i.test(item.state)).length;
  target.innerHTML = `
    ${renderNominasScreenHeader("Comunicaciones AFI, cotizacion RED/SLD y legales", "Preparacion y seguimiento de AFI, CRA, SLD/Siltra, Contrat@, Delt@ y FDI.", "Generar paquete", "Generar paquete comunicaciones", "AFI CRA SLD Contrat@ Delt@ FDI junio 2026")}
    ${renderNominasMetricCards([
      ["Canales activos", formatInteger(NOMINAS_CONTROL_DATA.socialSecurity.length), "AFI, CRA, SLD, Contrat@, Delt@, FDI", "#2563eb"],
      ["Registros incluidos", formatInteger(records), "Movimientos y tramos", "#0f766e"],
      ["Con aviso", formatInteger(pending), "Pendientes, diferencias o borradores", "#b45309"],
      ["Ciclo SLD", "17 tramos", "Pendientes de conciliacion Siltra", "#dc2626"],
    ])}
    <div style="display:grid; grid-template-columns:minmax(0, 1fr) 280px; gap:16px;">
      <section style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:16px; overflow-x:auto;">
        <table style="width:100%; border-collapse:collapse; font-size:0.86rem; min-width:760px;">
          <thead><tr style="background:#edf2f7;"><th style="text-align:left; padding:9px;">Canal</th><th style="text-align:left; padding:9px;">Fichero</th><th style="text-align:right; padding:9px;">Registros</th><th style="text-align:left; padding:9px;">Estado</th><th style="text-align:left; padding:9px;">Validacion</th><th style="padding:9px;">Accion</th></tr></thead>
          <tbody>
            ${NOMINAS_CONTROL_DATA.socialSecurity.map((item, index) => `
              <tr style="background:${index % 2 ? "#f8fafc" : "#fff"};">
                <td style="padding:9px; border-top:1px solid #e2e8f0; font-weight:700;">${item.channel}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.file}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0; text-align:right;">${formatInteger(item.records)}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${renderNominasStatusBadge(item.state)}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.error}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0; text-align:center;"><button type="button" data-nominas-action="Enviar comunicacion legal" data-nominas-detail="${item.channel} ${item.file}" style="border:1px solid #cbd5e1; background:#fff; border-radius:5px; padding:5px 8px; cursor:pointer;">Enviar</button></td>
              </tr>`).join("")}
          </tbody>
        </table>
      </section>
      <aside style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:16px;">
        <h3 style="margin:0 0 12px; font-size:1rem;">Mapa de integraciones</h3>
        ${[
          ["AFI", "Afiliacion: altas, bajas y variaciones"],
          ["CRA", "Conceptos retributivos abonados"],
          ["SLD", "Liquidacion directa y conciliacion Siltra"],
          ["Contrat@", "Comunicacion de contratos al SEPE"],
          ["Delt@", "Accidentes de trabajo"],
          ["FDI", "Partes de incapacidad temporal"],
        ].map(([label, text]) => `
          <button type="button" data-nominas-action="Abrir integracion legal" data-nominas-detail="${label}" style="width:100%; border:1px solid #e2e8f0; background:#fff; border-radius:7px; padding:9px; margin-bottom:8px; cursor:pointer; text-align:left;">
            <strong style="color:#2563eb;">${label}</strong><span style="display:block; color:#607080; font-size:0.78rem; margin-top:3px;">${text}</span>
          </button>`).join("")}
      </aside>
    </div>`;
  attachNominasActionButtons(target);
}

function renderNominasInformesCertificadosScreen(target) {
  const model190 = NOMINAS_CONTROL_DATA.reports.find((item) => item.name === "Modelo 190");
  target.innerHTML = `
    ${renderNominasScreenHeader("Informes, certificados y modelo 190", "Analisis de costes, certificados firmados, informes oficiales y ficheros tributarios anuales.", "Generar informe", "Generar informe certificado nomina", "Paquete informes junio 2026")}
    ${renderNominasMetricCards([
      ["Informes disponibles", formatInteger(NOMINAS_CONTROL_DATA.reports.length), "Costes, retribuciones y auditoria", "#2563eb"],
      ["Certificados", formatInteger(NOMINAS_CONTROL_DATA.reports.filter((item) => /certificado/i.test(item.name)).length), "PDF firmado con CSV", "#0f766e"],
      ["Modelo 190", model190 ? model190.state : "Pendiente", "Fichero anual AEAT", "#7c3aed"],
      ["Salidas", "Excel + PDF", "Exportacion y firma", "#475569"],
    ])}
    <div style="display:grid; grid-template-columns:minmax(0, 1fr) 280px; gap:16px;">
      <section style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:16px; overflow-x:auto;">
        <table style="width:100%; border-collapse:collapse; font-size:0.86rem; min-width:760px;">
          <thead><tr style="background:#edf2f7;"><th style="text-align:left; padding:9px;">Informe / certificado</th><th style="text-align:left; padding:9px;">Ambito</th><th style="text-align:left; padding:9px;">Salida</th><th style="text-align:left; padding:9px;">Estado</th><th style="padding:9px;">Accion</th></tr></thead>
          <tbody>
            ${NOMINAS_CONTROL_DATA.reports.map((item, index) => `
              <tr style="background:${index % 2 ? "#f8fafc" : "#fff"};">
                <td style="padding:9px; border-top:1px solid #e2e8f0; font-weight:700;">${item.name}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.scope}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.output}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${renderNominasStatusBadge(item.state)}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0; text-align:center;"><button type="button" data-nominas-action="Emitir informe nomina" data-nominas-detail="${item.name}" style="border:1px solid #cbd5e1; background:#fff; border-radius:5px; padding:5px 8px; cursor:pointer;">Emitir</button></td>
              </tr>`).join("")}
          </tbody>
        </table>
      </section>
      <aside style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:16px;">
        <h3 style="margin:0 0 12px; font-size:1rem;">Modelo 190</h3>
        <div style="display:grid; gap:8px; margin-bottom:12px;">
          ${[
            ["Perceptores", "2.876"],
            ["NIF pendientes", "9"],
            ["Claves validadas", "98,7%"],
            ["Estado", model190 ? model190.state : "Sin generar"],
          ].map(([label, value]) => `<div style="display:flex; justify-content:space-between; gap:10px; border-bottom:1px solid #e2e8f0; padding-bottom:7px;"><span style="color:#607080;">${label}</span><strong>${value}</strong></div>`).join("")}
        </div>
        <button type="button" data-nominas-action="Validar modelo 190" data-nominas-detail="Ejercicio 2026" style="width:100%; background:#7c3aed; color:#fff; border:none; border-radius:6px; padding:9px; cursor:pointer; font-weight:700;">Validar 190</button>
        <button type="button" data-nominas-screen="certificado-retenciones" style="width:100%; margin-top:8px; background:#fff; color:#1b5e20; border:1px solid #9fc9a3; border-radius:6px; padding:9px; cursor:pointer; font-weight:700;">Ver certificado 10T</button>
      </aside>
    </div>`;
  attachNominasActionButtons(target);
}

function renderNominasPrestamosFondoSocialScreen(target) {
  const judicial = NOMINAS_CONTROL_DATA.loans.filter((item) => /judicial|embargo|retencion/i.test(item.type));
  const pendingFund = NOMINAS_CONTROL_DATA.socialFund.filter((item) => /pendiente|revision/i.test(item.state));
  target.innerHTML = `
    ${renderNominasScreenHeader("Prestamos, retenciones judiciales y fondo social", "Anticipos, prestamos, embargos, cuotas, ayudas sociales, solicitud, aceptacion/rechazo y pago.", "Liquidar ayudas", "Liquidar fondo social y prestamos", "Junio 2026")}
    ${renderNominasMetricCards([
      ["Prestamos y anticipos", formatInteger(NOMINAS_CONTROL_DATA.loans.length - judicial.length), "Cuotas activas en nomina", "#2563eb"],
      ["Retenciones judiciales", formatInteger(judicial.length), "Embargos aplicados", "#dc2626"],
      ["Fondo social", formatInteger(NOMINAS_CONTROL_DATA.socialFund.length), "Solicitudes abiertas", "#0f766e"],
      ["Ayudas pendientes", formatInteger(pendingFund.length), "Justificante o informe social", "#b45309"],
    ])}
    <div style="display:grid; grid-template-columns:repeat(auto-fit, minmax(360px, 1fr)); gap:16px;">
      <section style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:16px; overflow-x:auto;">
        <h3 style="margin:0 0 12px; font-size:1rem;">Prestamos y retenciones</h3>
        <table style="width:100%; border-collapse:collapse; font-size:0.85rem; min-width:640px;">
          <thead><tr style="background:#edf2f7;"><th style="text-align:left; padding:9px;">Ref.</th><th style="text-align:left; padding:9px;">Empleado</th><th style="text-align:left; padding:9px;">Tipo</th><th style="text-align:right; padding:9px;">Cuota</th><th style="text-align:right; padding:9px;">Pendiente</th><th style="text-align:left; padding:9px;">Estado</th></tr></thead>
          <tbody>
            ${NOMINAS_CONTROL_DATA.loans.map((item, index) => `
              <tr style="background:${index % 2 ? "#f8fafc" : "#fff"};">
                <td style="padding:9px; border-top:1px solid #e2e8f0; font-weight:700;">${item.ref}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.employee}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.type}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0; text-align:right;">${formatEuroCompact(item.quota)}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0; text-align:right; font-weight:700;">${formatEuroCompact(item.pending)}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${renderNominasStatusBadge(item.state)}</td>
              </tr>`).join("")}
          </tbody>
        </table>
      </section>
      <section style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:16px; overflow-x:auto;">
        <h3 style="margin:0 0 12px; font-size:1rem;">Fondo social</h3>
        <table style="width:100%; border-collapse:collapse; font-size:0.85rem; min-width:640px;">
          <thead><tr style="background:#edf2f7;"><th style="text-align:left; padding:9px;">Ref.</th><th style="text-align:left; padding:9px;">Empleado</th><th style="text-align:left; padding:9px;">Ayuda</th><th style="text-align:right; padding:9px;">Importe</th><th style="text-align:left; padding:9px;">Estado</th><th style="text-align:left; padding:9px;">Decision</th></tr></thead>
          <tbody>
            ${NOMINAS_CONTROL_DATA.socialFund.map((item, index) => `
              <tr style="background:${index % 2 ? "#f8fafc" : "#fff"};">
                <td style="padding:9px; border-top:1px solid #e2e8f0; font-weight:700;">${item.ref}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.employee}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.aid}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0; text-align:right; font-weight:700;">${formatEuroCompact(item.amount)}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${renderNominasStatusBadge(item.state)}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.decision}</td>
              </tr>`).join("")}
          </tbody>
        </table>
      </section>
    </div>
    <div style="display:flex; flex-wrap:wrap; gap:10px; margin-top:14px;">
      ${["Generar cuotas", "Aplicar embargo", "Aceptar ayuda", "Requerir justificante", "Enviar a pago"].map((item) => `<button type="button" data-nominas-action="Accion prestamos fondo social" data-nominas-detail="${item}" style="border:1px solid #cbd5e1; background:#fff; border-radius:6px; padding:8px 12px; cursor:pointer; font-weight:700;">${item}</button>`).join("")}
    </div>`;
  attachNominasActionButtons(target);
}

function renderNominasCentroServicioUsuariosScreen(target) {
  const openCases = NOMINAS_CONTROL_DATA.serviceCenter.filter((item) => !/resuelto/i.test(item.state)).length;
  target.innerHTML = `
    ${renderNominasScreenHeader("Centro de servicio y usuarios PeopleNet", "Soporte funcional, administracion de usuarios, perfiles de acceso, SLA y mantenimiento de la plataforma de nominas.", "Abrir caso soporte", "Abrir caso centro servicio Peoplenet", "Incidencia funcional nominas")}
    ${renderNominasMetricCards([
      ["Casos abiertos", formatInteger(openCases), "Soporte funcional y tecnico", "#b45309"],
      ["Usuarios activos", formatInteger(NOMINAS_CONTROL_DATA.peoplenetUsers.length), "Perfiles internos configurados", "#2563eb"],
      ["Autenticacion", "DNIe/Cert + MFA", "Control de identidad fuerte", "#0f766e"],
      ["SLA critico", "4 h", "Nomina, SLD y pagos", "#dc2626"],
    ])}
    <div style="display:grid; grid-template-columns:minmax(0, 1fr) minmax(320px, 0.75fr); gap:16px;">
      <section style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:16px; overflow-x:auto;">
        <h3 style="margin:0 0 12px; font-size:1rem;">Centro de servicio</h3>
        <table style="width:100%; border-collapse:collapse; font-size:0.85rem; min-width:760px;">
          <thead><tr style="background:#edf2f7;"><th style="text-align:left; padding:9px;">Caso</th><th style="text-align:left; padding:9px;">Asunto</th><th style="text-align:left; padding:9px;">Solicitante</th><th style="text-align:left; padding:9px;">SLA</th><th style="text-align:left; padding:9px;">Responsable</th><th style="text-align:left; padding:9px;">Estado</th><th style="padding:9px;">Accion</th></tr></thead>
          <tbody>
            ${NOMINAS_CONTROL_DATA.serviceCenter.map((item, index) => `
              <tr style="background:${index % 2 ? "#f8fafc" : "#fff"};">
                <td style="padding:9px; border-top:1px solid #e2e8f0; font-weight:700;">${item.ref}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.subject}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.requester}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.sla}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.owner}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${renderNominasStatusBadge(item.state)}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0; text-align:center;"><button type="button" data-nominas-action="Gestionar caso centro servicio" data-nominas-detail="${item.ref}" style="border:1px solid #cbd5e1; background:#fff; border-radius:5px; padding:5px 8px; cursor:pointer;">Gestionar</button></td>
              </tr>`).join("")}
          </tbody>
        </table>
      </section>
      <section style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:16px; overflow-x:auto;">
        <h3 style="margin:0 0 12px; font-size:1rem;">Usuarios, roles y alcance</h3>
        <table style="width:100%; border-collapse:collapse; font-size:0.85rem; min-width:640px;">
          <thead><tr style="background:#edf2f7;"><th style="text-align:left; padding:9px;">Usuario</th><th style="text-align:left; padding:9px;">Perfil</th><th style="text-align:left; padding:9px;">Alcance</th><th style="text-align:left; padding:9px;">Autenticacion</th><th style="text-align:left; padding:9px;">Estado</th></tr></thead>
          <tbody>
            ${NOMINAS_CONTROL_DATA.peoplenetUsers.map((item, index) => `
              <tr style="background:${index % 2 ? "#f8fafc" : "#fff"};">
                <td style="padding:9px; border-top:1px solid #e2e8f0; font-family:monospace;">${item.user}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0; font-weight:700;">${item.profile}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.scope}</td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${item.auth}<br><small>Ultimo acceso: ${item.lastAccess}</small></td>
                <td style="padding:9px; border-top:1px solid #e2e8f0;">${renderNominasStatusBadge(item.state)}</td>
              </tr>`).join("")}
          </tbody>
        </table>
      </section>
    </div>
    <section style="display:grid; grid-template-columns:repeat(auto-fit, minmax(190px, 1fr)); gap:12px; margin-top:16px;">
      ${["Crear usuario", "Asignar perfil", "Bloquear acceso", "Revisar auditoria", "Planificar mantenimiento"].map((item) => `
        <button type="button" data-nominas-action="Administrar usuarios Peoplenet" data-nominas-detail="${item}" style="background:#fff; border:1px solid #d8e0e8; border-left:5px solid #2563eb; border-radius:8px; padding:13px; cursor:pointer; text-align:left; font-weight:700; color:#1d4ed8;">${item}</button>
      `).join("")}
    </section>`;
  attachNominasActionButtons(target);
}

function renderNominasPortalPeoplenetScreen(target) {
  const portalTargets = {
    "Mi informacion": { attr: 'data-portal-module="personal"', tone: "#2563eb" },
    "Ultimos recibos": { attr: 'data-nominas-screen="nomina-mes"', tone: "#15803d" },
    "Vacaciones y ausencias": { attr: 'data-portal-module="cronos"', tone: "#0f766e" },
    "Tareas y notificaciones": { attr: 'data-portal-module="notificaciones"', tone: "#b45309" },
  };
  const ownPortalItems = NOMINAS_CONTROL_DATA.employeePortal.filter((item) => item.tile !== "Quien es quien");
  target.innerHTML = `
    ${renderNominasScreenHeader("Portal empleado Peoplenet", "Vista propia del empleado: informacion personal, ultimos recibos, ausencias, tareas y notificaciones propias.", "Actualizar portal", "Actualizar portal empleado Peoplenet", "Autoservicio empleado junio 2026")}
    ${renderNominasMetricCards([
      ["Recibos disponibles", "3", "Nomina, atrasos y certificado", "#2563eb"],
      ["Solicitudes abiertas", "2", "Vacaciones y justificante", "#b45309"],
      ["Datos verificados", "100%", "Personal, bancario y RPT", "#15803d"],
      ["Notificaciones", "4", "Lectura o firma pendiente", "#7c3aed"],
    ])}
    <section style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:16px; margin-bottom:16px;">
      <div style="display:grid; grid-template-columns:repeat(auto-fit, minmax(220px, 1fr)); gap:12px;">
        ${ownPortalItems.map((item) => {
          const targetInfo = portalTargets[item.tile] || { attr: `data-nominas-action="Abrir portal empleado" data-nominas-detail="${item.tile}"`, tone: "#2563eb" };
          return `
          <button type="button" ${targetInfo.attr} style="background:#fff; border:1px solid #d8e0e8; border-left:5px solid ${targetInfo.tone}; border-radius:8px; padding:13px; cursor:pointer; text-align:left; min-height:112px;">
            <strong style="display:block; color:#13202d; margin-bottom:6px;">${item.tile}</strong>
            <span style="display:block; color:#607080; font-size:0.82rem; line-height:1.35;">${item.detail}</span>
            <span style="display:flex; justify-content:space-between; align-items:center; gap:8px; margin-top:10px;">
              ${renderNominasStatusBadge(item.state)}
              <small style="font-weight:700; color:${targetInfo.tone};">${item.action}</small>
            </span>
          </button>`;
        }).join("")}
      </div>
    </section>
    <div style="display:grid; grid-template-columns:minmax(0, 1fr) minmax(280px, 0.55fr); gap:16px;">
      <section style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:16px;">
        <h3 style="margin:0 0 12px; font-size:1rem;">Accesos propios</h3>
        <div style="display:grid; grid-template-columns:repeat(auto-fit, minmax(180px, 1fr)); gap:10px;">
          <button type="button" data-nominas-screen="nomina-mes" style="border:1px solid #9fc9a3; background:#f0fdf4; color:#1b5e20; border-radius:7px; padding:12px; cursor:pointer; font-weight:700; text-align:left;">Ver nomina mensual<span style="display:block; color:#607080; font-size:0.78rem; font-weight:400; margin-top:4px;">Junio 2026 y meses anteriores</span></button>
          <button type="button" data-nominas-screen="historico-evolucion" style="border:1px solid #bfdbfe; background:#eff6ff; color:#1d4ed8; border-radius:7px; padding:12px; cursor:pointer; font-weight:700; text-align:left;">Historico salarial<span style="display:block; color:#607080; font-size:0.78rem; font-weight:400; margin-top:4px;">Evolucion de conceptos</span></button>
          <button type="button" data-nominas-screen="certificado-retenciones" style="border:1px solid #ddd6fe; background:#f5f3ff; color:#6d28d9; border-radius:7px; padding:12px; cursor:pointer; font-weight:700; text-align:left;">Certificado 10T<span style="display:block; color:#607080; font-size:0.78rem; font-weight:400; margin-top:4px;">PDF firmado con CSV demo</span></button>
        </div>
      </section>
      <aside style="background:#fff; border:1px solid #d8e0e8; border-radius:8px; padding:16px;">
        <h3 style="margin:0 0 12px; font-size:1rem;">Mi informacion</h3>
        ${[
          ["Empleado", "ALBERTO SANCHEZ GOMEZ"],
          ["Puesto RPT", "A2 - Tecnico de Gestion"],
          ["Situacion", "Servicio activo"],
          ["Trienios", "4 reconocidos"],
          ["IRPF", `${state.nominasIrpfPercent || 12.5}%`],
        ].map(([label, value]) => `<div style="display:flex; justify-content:space-between; gap:10px; border-bottom:1px solid #e2e8f0; padding:7px 0;"><span style="color:#607080;">${label}</span><strong style="text-align:right;">${value}</strong></div>`).join("")}
      </aside>
    </div>`;
  attachNominasActionButtons(target);
}

function renderCustomNominasApp(container, view) {
  const visibleNominasItems = payrollMenuItems();
  const hasPayrollControl = payrollControlAllowed();
  const hasPayrollFullControl = payrollFullControlAllowed();
  if (state.nominasScreen === undefined || !visibleNominasItems.some((item) => item.id === state.nominasScreen)) {
    state.nominasScreen = defaultNominasScreen();
  }
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
  titleText.textContent = hasPayrollFullControl
    ? "Control de Nominas Peoplenet AAPP"
    : hasPayrollControl
      ? "Gestion operativa de Personal y Nominas"
      : "Portal del Empleado - Consulta de Nominas";
  yellowBar.append(titleText);

  const backButton = document.createElement("button");
  backButton.type = "button";
  backButton.textContent = "Volver";
  backButton.style.background = "#fff";
  backButton.style.color = "#1b5e20";
  backButton.style.border = "none";
  backButton.style.padding = "6px 14px";
  backButton.style.borderRadius = "4px";
  backButton.style.cursor = "pointer";
  backButton.style.fontWeight = "bold";
  backButton.style.fontSize = "0.85rem";
  backButton.addEventListener("click", () => {
    setActiveModule(defaultModuleID());
  });
  yellowBar.append(backButton);

  header.append(redBar, yellowBar);
  wrapper.append(header);

  const mainGrid = document.createElement("div");
  mainGrid.style.display = "grid";
  mainGrid.style.gridTemplateColumns = "280px 1fr";
  mainGrid.style.minHeight = "520px";

  const sidebar = document.createElement("div");
  sidebar.style.background = "#fff";
  sidebar.style.borderRight = "1px solid #ddd";
  sidebar.style.padding = "20px 10px";
  sidebar.style.display = "flex";
  sidebar.style.flexDirection = "column";
  sidebar.style.gap = "8px";
  sidebar.style.maxHeight = "calc(100vh - 160px)";
  sidebar.style.overflowY = "auto";

  visibleNominasItems.forEach(item => {
    const btn = document.createElement("button");
    btn.type = "button";
    btn.style.width = "100%";
    btn.style.padding = "10px 12px";
    btn.style.border = "none";
    btn.style.borderRadius = "6px";
    btn.style.textAlign = "left";
    btn.style.fontSize = "0.88rem";
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
  infoBadge.innerHTML = hasPayrollFullControl
    ? `
      <div style="font-weight:bold; margin-bottom:4px; color:#555;">Perfil RRHH / admin</div>
      <div>Control completo Peoplenet AAPP</div>
      <div style="margin-top:2px;">Nomina, RPT, SLD y Capitulo I</div>
      <div style="margin-top:2px; font-family:monospace; color:#888;">Junio 2026</div>
    `
    : hasPayrollControl
      ? `
      <div style="font-weight:bold; margin-bottom:4px; color:#555;">Perfil administrativo</div>
      <div>Gestion operativa limitada</div>
      <div style="margin-top:2px;">Expedientes, RPT, contratos y ausencias</div>
      <div style="margin-top:2px; font-family:monospace; color:#888;">Sin SLD, pagos ni 190</div>
    `
    : `
      <div style="font-weight:bold; margin-bottom:4px; color:#555;">Empleado activo</div>
      <div>ALBERTO SANCHEZ GOMEZ</div>
      <div style="margin-top:2px;">A2 - Tecnico de Gestion</div>
      <div style="margin-top:2px; font-family:monospace; color:#888;">NIF: 74839201A</div>
    `;
  sidebar.append(infoBadge);

  mainGrid.append(sidebar);

  const content = document.createElement("div");
  content.style.padding = "24px";
  content.style.overflowY = "auto";

  if (state.nominasScreen === "resumen-control") {
    renderNominasResumenControlScreen(content, view);
  } else if (state.nominasScreen === "organizacion-rpt") {
    renderNominasOrganizacionRPTScreen(content, view);
  } else if (state.nominasScreen === "trabajadores-centros") {
    renderNominasTrabajadoresCentrosScreen(content, view);
  } else if (state.nominasScreen === "checklist-avisos") {
    renderNominasChecklistAvisosScreen(content, view);
  } else if (state.nominasScreen === "pagas-tablas") {
    renderNominasPagasTablasScreen(content, view);
  } else if (state.nominasScreen === "calculo-retribuciones") {
    renderNominasCalculoRetribucionesScreen(content, view);
  } else if (state.nominasScreen === "inspector-nomina") {
    renderNominasInspectorScreen(content, view);
  } else if (state.nominasScreen === "retroactividad-revision") {
    renderNominasRetroactividadScreen(content, view);
  } else if (state.nominasScreen === "contratos-vencimientos") {
    renderNominasContratosVencimientosScreen(content, view);
  } else if (state.nominasScreen === "incapacidades-ausencias") {
    renderNominasIncapacidadesAusenciasScreen(content, view);
  } else if (state.nominasScreen === "estadisticas-bajas") {
    renderNominasEstadisticasBajasScreen(content, view);
  } else if (state.nominasScreen === "comunicaciones-legales") {
    renderNominasComunicacionesLegalesScreen(content, view);
  } else if (state.nominasScreen === "irpf-acumulados") {
    renderNominasIRPFAcumuladosScreen(content, view);
  } else if (state.nominasScreen === "informes-certificados") {
    renderNominasInformesCertificadosScreen(content, view);
  } else if (state.nominasScreen === "prestamos-fondo-social") {
    renderNominasPrestamosFondoSocialScreen(content, view);
  } else if (state.nominasScreen === "centro-servicio-usuarios") {
    renderNominasCentroServicioUsuariosScreen(content, view);
  } else if (state.nominasScreen === "pagos-contabilidad") {
    renderNominasPagosContabilidadScreen(content, view);
  } else if (state.nominasScreen === "portal-peoplenet") {
    renderNominasPortalPeoplenetScreen(content, view);
  } else if (state.nominasScreen === "nomina-mes") {
    renderNominasMesScreen(content, view, container);
  } else if (state.nominasScreen === "historico-evolucion") {
    renderNominasHistoricoScreen(content, view, container);
  } else if (state.nominasScreen === "certificado-retenciones") {
    renderNominasCertificadoScreen(content, view);
  } else {
    renderNominasPortalPeoplenetScreen(content, view);
  }
  const workflowPanel = renderNominasWorkflowPanel();
  if (workflowPanel) {
    const holder = document.createElement("div");
    holder.innerHTML = workflowPanel.trim();
    const panel = holder.firstElementChild;
    if (panel) {
      content.prepend(panel);
      attachNominasActionButtons(panel);
    }
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

  const actionsBar = document.createElement("div");
  actionsBar.style.display = "flex";
  actionsBar.style.flexWrap = "wrap";
  actionsBar.style.justifyContent = "flex-end";
  actionsBar.style.gap = "10px";
  actionsBar.style.margin = "-8px 0 18px";

  const printPdfBtn = document.createElement("button");
  printPdfBtn.type = "button";
  printPdfBtn.textContent = "Imprimir PDF";
  printPdfBtn.style.background = "#1b5e20";
  printPdfBtn.style.color = "#fff";
  printPdfBtn.style.border = "1px solid #174d1d";
  printPdfBtn.style.borderRadius = "6px";
  printPdfBtn.style.padding = "9px 16px";
  printPdfBtn.style.fontWeight = "bold";
  printPdfBtn.style.cursor = "pointer";
  printPdfBtn.addEventListener("click", () => {
    printPayrollPDF(getPayrollCalculations(state.nominasSelectedMonth), state.nominasSelectedMonth);
  });

  const exportExcelBtn = document.createElement("button");
  exportExcelBtn.type = "button";
  exportExcelBtn.textContent = "Exportar Excel";
  exportExcelBtn.style.background = "#fff";
  exportExcelBtn.style.color = "#1b5e20";
  exportExcelBtn.style.border = "1px solid #9fc9a3";
  exportExcelBtn.style.borderRadius = "6px";
  exportExcelBtn.style.padding = "9px 16px";
  exportExcelBtn.style.fontWeight = "bold";
  exportExcelBtn.style.cursor = "pointer";
  exportExcelBtn.addEventListener("click", () => {
    exportPayrollExcel(getPayrollCalculations(state.nominasSelectedMonth), state.nominasSelectedMonth);
  });

  actionsBar.append(printPdfBtn, exportExcelBtn);
  target.append(actionsBar);

  const kpiRow = document.createElement("div");
  kpiRow.style.display = "grid";
  kpiRow.style.gridTemplateColumns = "repeat(auto-fit, minmax(200px, 1fr))";
  kpiRow.style.gap = "16px";
  kpiRow.style.marginBottom = "24px";

  const kpis = [
    { title: "LÍQUIDO A RECIBIR", val: `${calc.liquido.toFixed(2)} €`, color: "#1b5e20", bg: "#e8f5e9" },
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

  if (!payrollFullControlAllowed()) {
    simHeader.textContent = "Recibo firmado en modo consulta";
    const readonlyItems = [
      ["Estado del recibo", "Firmado y publicado"],
      ["CSV", "CSV-9382-AJ84-29E1-401C"],
      ["Origen", "Nomina cerrada por RRHH"],
      ["Cruce dietas", calc.dietasVal > 0 ? `${calc.dietasVal.toFixed(2)} € incluidos` : "Sin dietas liquidadas"],
      ["Permisos", "Solo consulta, descarga PDF y exportacion"],
    ];
    readonlyItems.forEach(([itemLabel, itemValue]) => {
      const row = document.createElement("div");
      row.style.display = "flex";
      row.style.justifyContent = "space-between";
      row.style.gap = "10px";
      row.style.borderBottom = "1px solid #e2e8f0";
      row.style.padding = "8px 0";
      row.style.fontSize = "0.84rem";
      row.innerHTML = `<span style="color:#607080;">${itemLabel}</span><strong style="text-align:right; color:#13202d;">${itemValue}</strong>`;
      simCard.append(row);
    });
    const notice = document.createElement("div");
    notice.style.marginTop = "auto";
    notice.style.background = "#eff6ff";
    notice.style.border = "1px solid #bfdbfe";
    notice.style.borderRadius = "7px";
    notice.style.padding = "11px";
    notice.style.color = "#1d4ed8";
    notice.style.fontSize = "0.82rem";
    notice.textContent = "Las modificaciones de IRPF, trienios, complementos o productividad se tramitan por expediente y validacion de RRHH.";
    simCard.append(notice);
    layout.append(simCard);
    target.append(layout);
    return;
  }

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

function renderNominasHistoricoScreen(target, view, container) {
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
              <button type="button" class="btn-pdf-dummy" data-month="${m}" style="background:#f5f5f5; color:#333; border:1px solid #ccc; padding:3px 8px; border-radius:4px; cursor:pointer; font-size:0.75rem;">PDF</button>
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
      renderCustomNominasApp(container, view);
    });
  });

  table.querySelectorAll(".btn-pdf-dummy").forEach(btn => {
    btn.addEventListener("click", (event) => {
      const month = event.currentTarget.getAttribute("data-month") || state.nominasSelectedMonth;
      printPayrollPDF(getPayrollCalculations(month), month);
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
        Descargar certificado firmado
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
      <div style="font-size:2.4rem; color:#4caf50; margin-bottom:15px; font-weight:bold;">OK</div>
      <h3 style="margin:0 0 10px 0; color:#1b5e20; font-weight:bold;">Documento firmado para pruebas</h3>
      <p style="font-size:0.9rem; color:#555; line-height:1.4; margin-bottom:20px;">
        El certificado de retenciones IRPF del ejercicio fiscal 2025 se generara con firma electronica simulada y sello de tiempo demo.
      </p>

      <div style="background:#f9f9f9; border:1px dashed #ccc; padding:12px; border-radius:6px; font-family:monospace; font-size:0.75rem; color:#666; margin-bottom:20px; text-align:left;">
        <div><strong>Emisor:</strong> FNMT-RCM - Diputación de Granada (demo)</div>
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
      downloadRetencionesCertificatePDF();
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
  const moduleID = moduleIDForSession(hashState.moduleID);
  if (moduleID !== state.activeModule || hashState.screenID !== state.activeScreen) {
    setActiveModule(moduleID, hashState.screenID);
  }
});
reloadButton.addEventListener("click", loadPortal);
loadPortal();
