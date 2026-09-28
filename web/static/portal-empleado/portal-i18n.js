import { MENSAJES_AYUDA_PORTAL_ES, MENSAJES_AYUDA_PORTAL_EN } from "./portal-i18n-ayuda.js?v=20260928-idioma-en-v1";
import { MENSAJES_PANEL_INTERNO_ES, MENSAJES_PANEL_INTERNO_EN } from "./portal-panel-interno-i18n.js?v=20260928-idioma-en-v1";
import { MENSAJES_TEXTOS_PORTAL_ES, MENSAJES_TEXTOS_PORTAL_EN } from "./portal-i18n-textos.js?v=20260928-idioma-en-v1";
import { MENSAJES_RRHH_PLAZOS_ES, MENSAJES_RRHH_PLAZOS_EN } from "./modulos/bolsa/rrhh-plazos-i18n.js?v=20260928-idioma-en-v1";
import { MENSAJES_POLITICA_CESE_ES, MENSAJES_POLITICA_CESE_EN } from "./modulos/bolsa/rrhh-politica-cese-i18n.js?v=20260928-idioma-en-v1";
import { MENSAJES_BORRADORES_ES, MENSAJES_BORRADORES_EN } from "./portal-borradores-i18n.js?v=20260928-idioma-en-v1";
import { MENSAJES_MODULOS_PORTAL_ES, MENSAJES_MODULOS_PORTAL_EN } from "./portal-modulos-i18n.js?v=20260928-idioma-en-v1";
import { IDIOMA_ACTUAL, LOCALIZACION_ACTUAL } from "../comun/idioma.js";
export { IDIOMA_ACTUAL as IDIOMA_PORTAL, LOCALIZACION_ACTUAL as LOCALIZACION_PORTAL } from "../comun/idioma.js";

/** Catálogo común de los estados del shell y del acceso a Borradores. */
export const MENSAJES_PORTAL_ES = Object.freeze({
  ...MENSAJES_AYUDA_PORTAL_ES,
  ...MENSAJES_PANEL_INTERNO_ES,
  ...MENSAJES_TEXTOS_PORTAL_ES,
  ...MENSAJES_RRHH_PLAZOS_ES,
  ...MENSAJES_POLITICA_CESE_ES,
  ...MENSAJES_BORRADORES_ES,
  ...MENSAJES_MODULOS_PORTAL_ES,
  plantillas_rrhh_nav: "Plantillas de documentos",
  plantillas_rrhh_miga: "Portal del Empleado → Contratación temporal → Plantillas",
  plantillas_rrhh_titulo: "Plantillas de contratación temporal",
  auditoria_participacion_accion: "Consultar auditoría de esta participación",
  auditoria_sin_ficha: "Abra una ficha de participación para consultar su auditoría.",
  auditoria_expediente_accion: "Consultar auditoría de este expediente",
  auditoria_expediente_panel: "Auditoría del expediente",
  rrhh_no_disponible_hasta: "No disponible hasta",
  bolsa_razon_restriccion_cese: "Fuera de turno por cese hasta {fecha}",
  bolsa_razon_retorno_tras_cese: "Retorno al turno tras cese verificado",
  acceso_borradores_disponible: "Borradores disponibles",
  acceso_borradores_denegado: "Sin permiso para gestionar borradores",
  acceso_borradores_error: "Servicio de borradores no disponible",
  acceso_borradores_cargando: "Comprobando acceso a borradores",
  error_capacidad_consultar_invalida: "La API no devolvió una capacidad de consulta válida.",
  error_capacidad_consultar_denegada: "La sesión no dispone de capacidad para consultar borradores.",
  error_sesion_borradores_denegada: "La sesión no dispone de acceso a los borradores.",
  error_servicio_borradores: "El servicio de borradores no está disponible.",
  anuncio_acceso_borradores_denegado: "Acceso a borradores no concedido",
  anuncio_servicio_borradores_error: "Servicio de borradores no disponible",
  anuncio_acceso_borradores_comprobado: "Acceso a borradores comprobado",
  anuncio_acceso_borradores_no_disponible: "El acceso a borradores continúa sin estar disponible",
  permiso_perfil_denegado: "Sin permiso para este perfil",
  estado_modulo_activo: "Activo",
  estado_modulo_comprobando: "Comprobando",
  estado_modulo_sin_permiso: "Sin permiso",
  estado_modulo_no_disponible: "No disponible",
  estado_modulo_no_habilitado: "No habilitado",
  estado_modulo_disponible_perfil: "Disponible para el perfil activo",
  estado_modulo_no_disponible_titulo: "Módulo no disponible",
  inicio_titulo_neutro: "Inicio",
  inicio_comprobando_accesos: "Comprobando accesos…",
  inicio_modulos_etiqueta: "Módulos del portal",
  inicio_accesos_comprobados: "Accesos comprobados",
  inicio_empleado_sin_modulos: "No hay módulos disponibles para su perfil.",
  resumen_modulos_comprobando: "Comprobando módulos",
  resumen_modulos_ninguno: "Sin módulos disponibles",
  resumen_modulos_uno: "{cantidad} módulo disponible",
  resumen_modulos_varios: "{cantidad} módulos disponibles",
  perfil_sesion_rrhh: "Recursos Humanos",
  perfil_sesion_intervencion: "Intervención",
  perfil_sesion_personal: "Personal",
  perfil_sesion_jefatura: "Jefatura",
  sesion_etiqueta: "Sesión",
  titulo_error_catalogo_modulos: "No se pudieron comprobar los módulos",
  error_catalogo_modulos: "El catálogo interno de módulos no está disponible. Reintente la comprobación.",
  personal_catalogo_profesional: "Catálogo profesional de Personal",
  cronos_miga: "Portal del Empleado → Cronos",
  cronos_jornada_titulo: "Cronos · jornada y fichajes",
  cronos_permisos_miga: "Portal del Empleado → Cronos → Permisos",
  cronos_permisos_titulo: "Cronos · permisos y ausencias",
  contratos_consulta_etiqueta: "Consulta",
  descripcion_superficie_no_montada: "No se pudo montar la superficie solicitada.",
  contratacion_temporal_encabezado: "Contratación temporal",
  contratacion_temporal_miga: "Portal del Empleado → Contratación temporal",
  contratacion_temporal_titulo: "Gestión de expedientes de contratación temporal",
  contratacion_temporal_descripcion_no_disponible:
    "La composición real de Contratación temporal todavía no está disponible en este portal.",
  contratacion_temporal_aviso_no_disponible: "Esta vista no monta el módulo ni habilita sus operaciones.",
  accion_volver_portal: "Volver al portal",
  accion_ir_inicio_portal: "Ir al inicio del Portal del Empleado",
  operacion_no_compuesta: "Esta operación permanece deshabilitada hasta que su comando de servidor esté compuesto y autorizado.",
  accion_entrar: "Entrar",
  accion_reintentar: "Reintentar",
  paginacion_marco_etiqueta: "Paginación de la tabla",
  paginacion_marco_recuento: "Mostrando {inicio}–{fin} de {total}",
  paginacion_marco_primera: "Primera",
  paginacion_marco_anterior: "Anterior",
  paginacion_marco_siguiente: "Siguiente",
  selector_idioma_etiqueta: "Idioma de la interfaz",
  selector_idioma_es: "Español",
  selector_idioma_en: "Inglés",
});

/** English (UK) wording for messages owned by this module. */
const MENSAJES_PORTAL_PROPIOS_EN = Object.freeze({
  plantillas_rrhh_nav: "Document templates",
  plantillas_rrhh_miga: "Employee Portal → Temporary recruitment → Templates",
  plantillas_rrhh_titulo: "Temporary recruitment templates",
  auditoria_participacion_accion: "View audit trail for this application",
  auditoria_sin_ficha: "Open an application record to view its audit trail.",
  auditoria_expediente_accion: "View audit trail for this case",
  auditoria_expediente_panel: "Case audit trail",
  rrhh_no_disponible_hasta: "Unavailable until",
  bolsa_razon_restriccion_cese: "Out of rotation following termination until {fecha}",
  bolsa_razon_retorno_tras_cese: "Returned to rotation after verified termination",
  acceso_borradores_disponible: "Drafts available",
  acceso_borradores_denegado: "No permission to manage drafts",
  acceso_borradores_error: "Draft service unavailable",
  acceso_borradores_cargando: "Checking access to drafts",
  error_capacidad_consultar_invalida: "The API did not return a valid viewing capability.",
  error_capacidad_consultar_denegada: "This session cannot view drafts.",
  error_sesion_borradores_denegada: "This session cannot access drafts.",
  error_servicio_borradores: "The draft service is unavailable.",
  anuncio_acceso_borradores_denegado: "Access to drafts denied",
  anuncio_servicio_borradores_error: "Draft service unavailable",
  anuncio_acceso_borradores_comprobado: "Access to drafts checked",
  anuncio_acceso_borradores_no_disponible: "Access to drafts remains unavailable",
  permiso_perfil_denegado: "No permission for this profile",
  estado_modulo_activo: "Active",
  estado_modulo_comprobando: "Checking",
  estado_modulo_sin_permiso: "No permission",
  estado_modulo_no_disponible: "Unavailable",
  estado_modulo_no_habilitado: "Not enabled",
  estado_modulo_disponible_perfil: "Available for the active profile",
  estado_modulo_no_disponible_titulo: "Module unavailable",
  inicio_titulo_neutro: "Home",
  inicio_comprobando_accesos: "Checking access…",
  inicio_modulos_etiqueta: "Portal modules",
  inicio_accesos_comprobados: "Access checks complete",
  inicio_empleado_sin_modulos: "No modules are available for your profile.",
  resumen_modulos_comprobando: "Checking modules",
  resumen_modulos_ninguno: "No modules available",
  resumen_modulos_uno: "{cantidad} module available",
  resumen_modulos_varios: "{cantidad} modules available",
  perfil_sesion_rrhh: "Human Resources",
  perfil_sesion_intervencion: "Financial Control",
  perfil_sesion_personal: "Personnel",
  perfil_sesion_jefatura: "Line management",
  sesion_etiqueta: "Session",
  titulo_error_catalogo_modulos: "Modules could not be checked",
  error_catalogo_modulos: "The internal module catalogue is unavailable. Please try again.",
  personal_catalogo_profesional: "Personnel professional catalogue",
  cronos_miga: "Employee Portal → Time and attendance",
  cronos_jornada_titulo: "Time and attendance · working hours and clockings",
  cronos_permisos_miga: "Employee Portal → Time and attendance → Leave",
  cronos_permisos_titulo: "Time and attendance · leave and absences",
  contratos_consulta_etiqueta: "View",
  descripcion_superficie_no_montada: "The requested view could not be loaded.",
  contratacion_temporal_encabezado: "Temporary recruitment",
  contratacion_temporal_miga: "Employee Portal → Temporary recruitment",
  contratacion_temporal_titulo: "Temporary recruitment case management",
  contratacion_temporal_descripcion_no_disponible: "The live temporary recruitment module is not yet available in this portal.",
  contratacion_temporal_aviso_no_disponible: "This view does not load the module or enable its operations.",
  accion_volver_portal: "Return to portal",
  accion_ir_inicio_portal: "Go to the Employee Portal home page",
  operacion_no_compuesta: "This operation remains disabled until its server command is connected and authorised.",
  accion_entrar: "Open",
  accion_reintentar: "Try again",
  paginacion_marco_etiqueta: "Table pagination",
  paginacion_marco_recuento: "Showing {inicio}–{fin} of {total}",
  paginacion_marco_primera: "First",
  paginacion_marco_anterior: "Previous",
  paginacion_marco_siguiente: "Next",
  selector_idioma_etiqueta: "Interface language",
  selector_idioma_es: "Spanish",
  selector_idioma_en: "English",
});

export const MENSAJES_PORTAL_EN = Object.freeze({
  ...MENSAJES_AYUDA_PORTAL_EN,
  ...MENSAJES_PANEL_INTERNO_EN,
  ...MENSAJES_TEXTOS_PORTAL_EN,
  ...MENSAJES_RRHH_PLAZOS_EN,
  ...MENSAJES_POLITICA_CESE_EN,
  ...MENSAJES_BORRADORES_EN,
  ...MENSAJES_MODULOS_PORTAL_EN,
  ...MENSAJES_PORTAL_PROPIOS_EN,
});

function validarParidad(castellano, ingles, nombre) {
  const clavesES = Object.keys(castellano).sort();
  const clavesEN = Object.keys(ingles).sort();
  if (clavesES.length !== clavesEN.length || clavesES.some((clave, indice) => clave !== clavesEN[indice])) {
    throw new Error(`catálogo i18n ${nombre} con claves desiguales`);
  }
  for (const clave of clavesES) {
    if (typeof ingles[clave] !== "string" || !ingles[clave].trim()) {
      throw new Error(`catálogo i18n ${nombre} incompleto: ${clave}`);
    }
    const marcadores = (mensaje) => [...mensaje.matchAll(/\{([a-z_]+)\}/g)].map((coincidencia) => coincidencia[1]).sort();
    if (JSON.stringify(marcadores(castellano[clave])) !== JSON.stringify(marcadores(ingles[clave]))) {
      throw new Error(`catálogo i18n ${nombre} con marcadores desiguales: ${clave}`);
    }
  }
}

validarParidad(MENSAJES_PORTAL_ES, MENSAJES_PORTAL_EN, "portal");

const CLAVES = Object.freeze(Object.keys(MENSAJES_PORTAL_ES));

export function crearTraductorPortal(catalogo = MENSAJES_PORTAL_ES) {
  if (!catalogo || typeof catalogo !== "object"
    || CLAVES.some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === "")) {
    throw new Error("catálogo i18n del Portal del Empleado incompleto");
  }
  return (clave, variables = {}) => {
    if (!CLAVES.includes(clave)) throw new Error(`clave i18n del portal desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/g,
      (_coincidencia, variable) => String(variables[variable] ?? ""));
  };
}

const traducirPortalES = crearTraductorPortal(MENSAJES_PORTAL_ES);
const traducirPortalEN = crearTraductorPortal(MENSAJES_PORTAL_EN);
export function traducirPortal(clave, variables = {}) {
  return (IDIOMA_ACTUAL === "en" ? traducirPortalEN : traducirPortalES)(clave, variables);
}

/** Texto del catálogo común ya escapado para insertarlo en una plantilla HTML. */
export function textoPortal(clave, variables = {}) {
  return traducirPortal(clave, variables).replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;").replaceAll("'", "&#039;");
}

/** Textos comunes de las vistas internas de Bolsa. */
export const MENSAJES_BOLSA_INTERNA_ES = Object.freeze({
  b24_recurso_historial: "Historial del recurso ({total})",
  b24_recurso_evento: "{estado} · {fecha} · Anotado por {actor}",
  b24_recurso_documento: "Escrito: {referencia}",
  b7_seleccion_denegada: "Permiso denegado al consultar candidatos. Se han retirado los datos y la selección.",
  b7_consulta_fallida: "No se pudo completar la consulta: {motivo}",
  b7_error_lectura: "error de lectura",
  b7_estados_obligatorios: "Seleccione al menos un estado antes de consultar la bolsa.",
  b7_estados_cambiados: "Los estados cambiaron. Vuelva a seleccionar los candidatos.",
  b7_limite_envio: "El límite por envío es de 100 candidatos.",
  b7_cuerpo_excesivo: "El texto final del correo supera 4000 caracteres. Reduzca el mensaje antes de continuar.",
  b7_seleccion_cambiada: "La selección ha cambiado. Revise el número de candidatos antes de confirmar.",
  b7_emision_rechazada: "El servidor rechazó la emisión (422). Revise la configuración o actualice la selección según la bolsa vigente.",
  boton_perfil_sin_permiso: "El perfil activo no permite esta operación",
  boton_capacidad_no_conectada: "Capacidad de servidor no conectada",
  tabla_sin_registros: "No hay registros para los filtros aplicados.",
  tabla_region_operativa: "Tabla operativa: {titulo}",
  fuente_real_no_conectada: "Fuente real no conectada",
  capacidad_real_no_disponible: "Capacidad real no disponible",
  funcionalidad_no_conectada: "Funcionalidad no conectada.",
  detalle_funcionalidad_no_conectada: "La misma pantalla queda visible, pero sus acciones permanecen deshabilitadas hasta que el servidor conceda capacidad explícita y aporte datos autorizados.",
  numero_convocatorias: "{numero} convocatorias encontradas.",
  numero_solicitudes: "{numero} solicitudes encontradas.",
  numero_meritos: "{numero} méritos encontrados.",
  fecha_sin_valor: "Sin fecha",
  contacto_historico_titulo: "Histórico de contactos y llamamientos",
  contacto_historico_descripcion: "Contactos y llamamientos, más recientes primero",
  contacto_historico_vacio: "Sin contactos ni llamamientos registrados",
  contacto_tipo: "Contacto",
	contacto_contador: "Contactos",
  contacto_llamamiento_tipo: "Llamamiento",
  contacto_registrado: "Contacto registrado.",
  contacto_ultimo_llamamiento: "Último llamamiento",
  contacto_registrar: "Registrar contacto",
  contacto_canal: "Canal",
  contacto_resultado: "Resultado",
  contacto_llamamiento: "Llamamiento",
  contacto_sin_vincular: "Sin vincular",
  contacto_anotacion: "Anotación",
  contacto_telefono: "Teléfono",
  contacto_correo: "Correo",
  contacto_sms: "SMS",
  contacto_presencial: "Presencial",
  contacto_otro: "Otro",
  contacto_contactado: "Contactado",
  contacto_no_contesta: "No contesta",
  contacto_buzon: "Buzón",
  contacto_acepta: "Acepta",
  contacto_rechaza: "Rechaza",
  contacto_aplazado: "Aplazado",
  contacto_enviado: "Enviado",
  contacto_no_enviado: "No enviado",
  contacto_formulario_incompleto: "Complete canal, resultado y anotación.",
  contacto_solicitud_invalida: "Faltan datos obligatorios del contacto.",
  contacto_permiso_denegado: "La sesión no dispone de permiso para registrar contactos.",
  contacto_clave_conflicto: "La clave corresponde a otro contacto.",
  contacto_registro_error: "No se pudo registrar el contacto.",
  contacto_comunicacion_error: "Error de comunicación.",
  bolsa_estado_disponible: "Disponibles",
  bolsa_reposicion_misma_posicion: "Misma posición",
  bolsa_reposicion_fin_lista: "Al final de la lista",
  bolsa_reposicion_no_disponible_hasta_fecha: "No disponible hasta una fecha",
  bolsa_estado_trabajando: "Ocupados / Trabajando",
  bolsa_estado_no_disponible: "No disponibles",
  bolsa_estado_excluido: "Excluidos",
  bolsa_estado_renuncia: "Renuncia",
  bolsa_estado_pendiente_incorporacion: "Pendiente de incorporación",
  bolsa_estado_disponible_desde: "Disponible desde fecha",
  bolsa_sustituida_aviso: "Sustituida por la bolsa vigente de la categoría el {fecha}.",
  bolsa_abrir_vigente: "Abrir bolsa vigente",
});

export const MENSAJES_BOLSA_INTERNA_EN = Object.freeze({
  b24_recurso_historial: "Appeal history ({total})",
  b24_recurso_evento: "{estado} · {fecha} · Recorded by {actor}",
  b24_recurso_documento: "Submission: {referencia}",
  b7_seleccion_denegada: "Permission to view candidates was denied. The data and selection have been cleared.",
  b7_consulta_fallida: "The search could not be completed: {motivo}",
  b7_error_lectura: "read error",
  b7_estados_obligatorios: "Select at least one status before searching the pool.",
  b7_estados_cambiados: "The statuses have changed. Select the candidates again.",
  b7_limite_envio: "Each dispatch is limited to 100 candidates.",
  b7_cuerpo_excesivo: "The final email text exceeds 4,000 characters. Shorten the message before continuing.",
  b7_seleccion_cambiada: "The selection has changed. Check the number of candidates before confirming.",
  b7_emision_rechazada: "The server rejected dispatch (422). Check the settings or update the selection against the current pool.",
  boton_perfil_sin_permiso: "The active profile does not permit this operation",
  boton_capacidad_no_conectada: "Server capability not connected",
  tabla_sin_registros: "No records match the selected filters.",
  tabla_region_operativa: "Operational table: {titulo}",
  fuente_real_no_conectada: "Live data source not connected",
  capacidad_real_no_disponible: "Live capability unavailable",
  funcionalidad_no_conectada: "Functionality not connected.",
  detalle_funcionalidad_no_conectada: "This view remains visible, but its actions are disabled until the server grants an explicit capability and supplies authorised data.",
  numero_convocatorias: "{numero} recruitment rounds found.",
  numero_solicitudes: "{numero} applications found.",
  numero_meritos: "{numero} merits found.",
  fecha_sin_valor: "No date",
  contacto_historico_titulo: "Contact and call-up history",
  contacto_historico_descripcion: "Contacts and call-ups, most recent first",
  contacto_historico_vacio: "No contacts or call-ups recorded",
  contacto_tipo: "Contact",
  contacto_contador: "Contacts",
  contacto_llamamiento_tipo: "Call-up",
  contacto_registrado: "Contact recorded.",
  contacto_ultimo_llamamiento: "Latest call-up",
  contacto_registrar: "Record contact",
  contacto_canal: "Channel",
  contacto_resultado: "Outcome",
  contacto_llamamiento: "Call-up",
  contacto_sin_vincular: "Not linked",
  contacto_anotacion: "Note",
  contacto_telefono: "Telephone",
  contacto_correo: "Email",
  contacto_sms: "SMS",
  contacto_presencial: "In person",
  contacto_otro: "Other",
  contacto_contactado: "Contacted",
  contacto_no_contesta: "No answer",
  contacto_buzon: "Voicemail",
  contacto_acepta: "Accepts",
  contacto_rechaza: "Declines",
  contacto_aplazado: "Deferred",
  contacto_enviado: "Sent",
  contacto_no_enviado: "Not sent",
  contacto_formulario_incompleto: "Complete the channel, outcome and note.",
  contacto_solicitud_invalida: "Required contact details are missing.",
  contacto_permiso_denegado: "This session cannot record contacts.",
  contacto_clave_conflicto: "The key belongs to another contact.",
  contacto_registro_error: "The contact could not be recorded.",
  contacto_comunicacion_error: "Communication error.",
  bolsa_estado_disponible: "Available",
  bolsa_reposicion_misma_posicion: "Same position",
  bolsa_reposicion_fin_lista: "At the end of the list",
  bolsa_reposicion_no_disponible_hasta_fecha: "Unavailable until a specified date",
  bolsa_estado_trabajando: "Occupied / Working",
  bolsa_estado_no_disponible: "Unavailable",
  bolsa_estado_excluido: "Excluded",
  bolsa_estado_renuncia: "Withdrawal",
  bolsa_estado_pendiente_incorporacion: "Awaiting start",
  bolsa_estado_disponible_desde: "Available from date",
  bolsa_sustituida_aviso: "Replaced by the current pool for this category on {fecha}.",
  bolsa_abrir_vigente: "Open current pool",
});

const CLAVES_BOLSA_INTERNA = Object.freeze(Object.keys(MENSAJES_BOLSA_INTERNA_ES));
validarParidad(MENSAJES_BOLSA_INTERNA_ES, MENSAJES_BOLSA_INTERNA_EN, "Bolsa interna");
export function crearTraductorBolsaInterna(catalogo = MENSAJES_BOLSA_INTERNA_ES) {
  if (!catalogo || typeof catalogo !== "object" || CLAVES_BOLSA_INTERNA.some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === "")) {
    throw new Error("catálogo i18n de Bolsa interna incompleto");
  }
  return (clave, variables = {}) => {
    if (!CLAVES_BOLSA_INTERNA.includes(clave)) throw new Error(`clave i18n de Bolsa interna desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/g, (_coincidencia, variable) => String(variables[variable] ?? ""));
  };
}
const traducirBolsaInternaES = crearTraductorBolsaInterna(MENSAJES_BOLSA_INTERNA_ES);
const traducirBolsaInternaEN = crearTraductorBolsaInterna(MENSAJES_BOLSA_INTERNA_EN);
export function traducirBolsaInterna(clave, variables = {}) {
  return (IDIOMA_ACTUAL === "en" ? traducirBolsaInternaEN : traducirBolsaInternaES)(clave, variables);
}
/** Localización y zona horaria del portal: autoridad común para formatear fechas, horas, importes y cifras. */
export const ZONA_HORARIA_PORTAL = "Europe/Madrid";
export function formatearNumeroPortal(valor, opciones = {}) {
  const numero = Number(valor);
  return Number.isFinite(numero) ? new Intl.NumberFormat(LOCALIZACION_ACTUAL, opciones).format(numero) : String(valor ?? "");
}
export function formatearFechaPortal(valor) {
  if (valor === undefined || valor === null || valor === "") return traducirBolsaInterna("fecha_sin_valor");
  const texto = String(valor).trim();
  const local = texto.match(/^(\d{4})-(\d{2})-(\d{2})(?:T(\d{2}):(\d{2}))?$/) || texto.match(/^(\d{2})\/(\d{2})\/(\d{4})(?:\s+(\d{2}):(\d{2}))?$/);
  if (!local) return texto;
  const iso = texto.includes("-");
  const [dia, mes, ano, hora, minuto] = iso ? [local[3], local[2], local[1], local[4], local[5]] : [local[1], local[2], local[3], local[4], local[5]];
  const fecha = new Date(Number(ano), Number(mes) - 1, Number(dia), Number(hora || 0), Number(minuto || 0));
  if (!Number.isFinite(fecha.getTime())) return texto;
  return new Intl.DateTimeFormat(LOCALIZACION_ACTUAL, hora ? { dateStyle: "short", timeStyle: "short" } : { dateStyle: "short" }).format(fecha);
}
