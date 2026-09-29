import { IDIOMA_ACTUAL, montarSelectorIdioma } from "./comun/idioma.js";

// El texto castellano del HTML es el fallback legible sin JavaScript.
// Cada entrada tiene una clave estable y las dos traducciones completas.
export const MENSAJES_PORTADA = {
  titulo: ["VEC Diputacion Granada", "VEC Granada Provincial Council"],
  institucion: ["Diputación de Granada", "Diputación de Granada"],
  portal_corporativo: ["Portal corporativo", "Corporate portal"],
  recursos_humanos: ["VEC · Recursos Humanos", "VEC · Human Resources"],
  modulos_vec: ["Modulos VEC", "VEC modules"],
  navegacion_principal: ["Navegacion principal", "Main navigation"],
  portal_empleado: ["Portal empleado", "Employee portal"],
  personal: ["Personal", "Personnel"],
  nominas: ["Nominas", "Payroll"],
  cronos: ["Cronos", "Cronos"],
  dietas: ["Dietas", "Expenses"],
  bolsa: ["Bolsa", "Candidate pool"],
  expediente_control: ["Expediente y control", "Cases and oversight"],
  documentos: ["Documentos", "Documents"],
  aprobaciones: ["Aprobaciones", "Approvals"],
  auditoria: ["Auditoria", "Audit"],
  administracion: ["Administracion", "Administration"],
  espacio_administrativo: ["Workspace administrativo", "Administrative workspace"],
  tablero: ["Tablero operativo VEC", "VEC operations dashboard"],
  sesion: ["Sesion personal interno - contexto del modulo activo", "Internal staff session · active module context"],
  buscar_ayuda: ["Buscar expediente, DNI parcial o solicitud", "Search by case, partial identity number or application"],
  buscar: ["Buscar", "Search"],
  preparando: ["Preparando demo", "Preparing demonstration"],
  abrir_aprobaciones: ["Abrir aprobaciones", "Open approvals"],
  aprobaciones_ocho: ["Aprobaciones 8", "8 approvals"],
  bandeja_principal: ["Bandeja principal", "Main work queue"],
  cola_tramitacion: ["Cola VEC de tramitacion", "VEC processing queue"],
  expedientes_modulos: ["Expedientes, modulos y acciones pendientes", "Cases, modules and pending actions"],
  vista_interna: ["Vista interna para controlar horarios, fichajes, permisos, vacaciones, dietas, kilometraje provincial y expedientes desde un unico portal.", "Internal view of schedules, attendance, leave, holidays, expenses, travel distances and cases in one portal."],
  actualizar: ["Actualizar tablero", "Refresh dashboard"],
  resumen_operativo: ["Resumen operativo", "Operations summary"],
  personas_activas: ["Personas activas", "Active people"],
  puestos_situaciones: ["Puestos, situaciones y nomina", "Posts, employment status and payroll"],
  perfiles_horarios: ["Perfiles horarios", "Working patterns"],
  flexibles_turnos: ["Flexibles y turnos fijos", "Flexible and fixed shifts"],
  registros_vec: ["Registros VEC", "VEC records"],
  definitivo: ["Definitivo publicado", "Final version published"],
  dietas_pendientes: ["Dietas pendientes", "Pending expenses"],
  alegaciones_evidencias: ["5 alegaciones - 7 evidencias", "5 representations · 7 pieces of evidence"],
  filtros: ["Filtros de expedientes", "Case filters"],
  ambito: ["Ambito", "Scope"],
  horarios: ["Horarios del personal", "Staff schedules"],
  expediente_empleado: ["Expediente de empleado", "Employee record"],
  nomina_incidencias: ["Nomina e incidencias", "Payroll and issues"],
  antiguedad: ["Antiguedad y trienios", "Length of service and three-year increments"],
  servicios: ["Servicios prestados", "Service history"],
  fichajes: ["Fichajes e incidencias", "Attendance and issues"],
  permisos: ["Permisos y vacaciones", "Leave and holidays"],
  comision: ["Comision de servicio", "Secondment"],
  mapa: ["Mapa provincial", "Provincial map"],
  convocatorias: ["Convocatorias abiertas", "Open calls"],
  expediente_candidato: ["Expediente candidato", "Candidate record"],
  estado: ["Estado", "Status"],
  pendiente_accion: ["Pendiente de accion", "Action pending"],
  en_revision: ["En revision", "Under review"],
  subsanacion: ["Subsanacion requerida", "Correction required"],
  riesgo_plazo: ["Riesgo plazo", "Deadline risk"],
  vence_72: ["Vence en 72 h", "Due in 72 hours"],
  sin_vencimiento: ["Sin vencimiento critico", "No urgent deadline"],
  plazo_vencido: ["Plazo vencido", "Deadline passed"],
  unidad: ["Unidad", "Unit"],
  registro_documentos: ["Registro y documentos", "Registry and documents"],
  aplicar: ["Aplicar", "Apply"],
  cola_expedientes: ["Cola de expedientes VEC", "VEC case queue"],
  orden_riesgo: ["Horarios, permisos, dietas y expedientes - orden por riesgo", "Schedules, leave, expenses and cases · ordered by risk"],
  exportar: ["Exportar", "Export"],
  caption_registros: ["Registros administrativos con modulo, estado, plazo, magnitud y accion siguiente.", "Administrative records with module, status, deadline, amount and next action."],
  expediente: ["Expediente", "Case"],
  plazo: ["Plazo", "Deadline"],
  magnitud: ["Magnitud", "Amount"],
  justificante: ["Justificante", "Receipt"],
  accion: ["Accion", "Action"],
  csv_pendiente: ["CSV pendiente", "Verification code pending"],
  revisar: ["Revisar", "Review"],
  alegacion: ["Alegacion presentada", "Representation submitted"],
  tres_evidencias: ["3 evidencias", "3 pieces of evidence"],
  puntos_61: ["61,4 pt", "61.4 points"],
  puntos_58: ["58,0 pt", "58.0 points"],
  puntos_47: ["47,5 pt", "47.5 points"],
  puntos_44: ["44,2 pt", "44.2 points"],
  resolver: ["Resolver", "Resolve"],
  meritos_revision: ["Meritos en revision", "Merits under review"],
  metadatos: ["Metadatos presentes", "Metadata available"],
  validar: ["Validar", "Validate"],
  admitida: ["Admitida provisional", "Provisionally admitted"],
  sin_accion: ["Sin accion", "No action"],
  archivado: ["Archivado", "Archived"],
  abrir: ["Abrir", "Open"],
  control_cronos_dietas: ["Control Cronos y Dietas", "Cronos and expenses overview"],
  cronos_saldo: ["Cronos: saldo horario y permisos", "Cronos: working time and leave balances"],
  junio: ["Junio 2026", "June 2026"],
  dietas_mapa: ["Dietas: mapa provincial de kilometraje", "Expenses: provincial travel distances"],
  rutas_demo: ["Rutas demo", "Demonstration routes"],
  detalle_seleccionado: ["Detalle del expediente seleccionado", "Selected case details"],
  seleccion_actual: ["Seleccion actual - registro VEC", "Current selection · VEC record"],
  accion_requerida: ["Accion requerida", "Action required"],
  datos_clave: ["Datos clave", "Key details"],
  persona_ref: ["Persona / ref.", "Person / ref."],
  ultimo_cambio: ["Ultimo cambio", "Last change"],
  secciones: ["Secciones del expediente", "Case sections"],
  resumen: ["Resumen", "Summary"],
  meritos: ["Meritos", "Merits"],
  docs: ["Docs", "Documents"],
  alertas: ["Alertas del expediente", "Case alerts"],
  sin_csv: ["Documento sin CSV verificable", "Document without a verifiable code"],
  experiencia_categoria: ["Experiencia misma categoria - requiere subsanacion", "Experience in the same category · correction required"],
  tope_formacion: ["Tope de formacion aplicado", "Training cap applied"],
  puntos_tope: ["30,0 pt computados - 4,5 pt no computan por regla", "30.0 points counted · 4.5 points excluded by rule"],
  evidencias_documentos: ["Evidencias y documentos", "Evidence and documents"],
  cuatro_items: ["4 items", "4 items"],
  csv_sha: ["Pendiente CSV - SHA-256 no confirmado", "Verification code pending · SHA-256 unconfirmed"],
  titulo_formacion: ["Titulo formacion", "Training certificate"],
  metadatos_firma: ["Metadatos presentes - firma externa pendiente", "Metadata available · external signature pending"],
  justificante_registro: ["Justificante registro", "Registry receipt"],
  csv_descargable: ["CSV-GR-2026-8841 - descargable", "CSV-GR-2026-8841 · downloadable"],
  historial: ["Historial y auditoria", "History and audit"],
  trazado: ["Trazado", "Traced"],
  personal_interno: ["19/06/2026 09:42 - personal interno - AUD-4421", "19/06/2026 09:42 · internal staff · AUD-4421"],
  autobaremo: ["Autobaremo recalculado", "Self-assessment recalculated"],
  reglas_v1: ["18/06/2026 16:10 - reglas v1 - AUD-4388", "18/06/2026 16:10 · rules v1 · AUD-4388"],
  solicitud_registrada: ["Solicitud registrada", "Application registered"],
  solicitud_fecha: ["17/06/2026 11:25 - CSV-GR-2026-8841", "17/06/2026 11:25 · CSV-GR-2026-8841"],
  ultima_sincronizacion: ["Ultima sincronizacion local: 19/06/2026 10:15 - adaptador demo en memoria", "Last local synchronisation: 19/06/2026 10:15 · in-memory demonstration adapter"],
  refs_opacas: ["Refs opacas preservadas: worktree-bolsa-diputacion-existing-app - branch-bolsa-profesional-002", "Opaque references retained: worktree-bolsa-diputacion-existing-app · branch-bolsa-profesional-002"],
  idioma: ["Idioma", "Language"],
  castellano: ["Castellano", "Spanish"],
  ingles: ["English", "English"],
};

export const CATALOGO_PORTADA = Object.freeze({
  es: Object.fromEntries(Object.entries(MENSAJES_PORTADA).map(([clave, par]) => [clave, par[0]])),
  en: Object.fromEntries(Object.entries(MENSAJES_PORTADA).map(([clave, par]) => [clave, par[1]])),
});

export function textoPortada(clave, idioma = IDIOMA_ACTUAL) {
  return CATALOGO_PORTADA[idioma]?.[clave] ?? CATALOGO_PORTADA.es[clave] ?? clave;
}

export function aplicarIdiomaPortada(documento = document, idioma = IDIOMA_ACTUAL) {
  documento.documentElement.lang = idioma;
  documento.title = textoPortada("titulo", idioma);
  const raiz = documento.querySelector(".workspace");
  if (!raiz) return;
  for (const elemento of raiz.querySelectorAll("[data-i18n]")) {
    const clave = elemento.dataset.i18n;
    const nodo = [...elemento.childNodes].find((hijo) => hijo.nodeType === 3 && hijo.nodeValue.trim());
    if (nodo) nodo.nodeValue = nodo.nodeValue.replace(/\S[\s\S]*\S|\S/u, textoPortada(clave, idioma));
  }
  for (const elemento of raiz.querySelectorAll("[data-i18n-aria-label], [data-i18n-placeholder], [data-i18n-title], [data-i18n-alt]")) {
    for (const atributo of ["aria-label", "placeholder", "title", "alt"]) {
      const clave = elemento.getAttribute(`data-i18n-${atributo}`);
      if (clave) elemento.setAttribute(atributo, textoPortada(clave, idioma));
    }
  }
  const selector = documento.querySelector("#vec-language");
  montarSelectorIdioma(selector);
  if (selector) selector.setAttribute("aria-label", textoPortada("idioma", idioma));
}

if (typeof document !== "undefined") {
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", () => aplicarIdiomaPortada(), { once: true });
  else aplicarIdiomaPortada();
}
