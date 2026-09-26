import { MENSAJES_AYUDA_PORTAL_ES } from "./portal-i18n-ayuda.js?v=20260926-huecos-rrhh-v1";
import { MENSAJES_PANEL_INTERNO_ES } from "./portal-panel-interno-i18n.js?v=20260926-integracion-bolsa-ct-v1";

/** Catálogo común de los estados del shell y del acceso a Borradores. */
export const MENSAJES_PORTAL_ES = Object.freeze({
  ...MENSAJES_AYUDA_PORTAL_ES,
  ...MENSAJES_PANEL_INTERNO_ES,
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
});

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

export const traducirPortal = crearTraductorPortal();

/** Textos comunes de las vistas internas de Bolsa. */
export const MENSAJES_BOLSA_INTERNA_ES = Object.freeze({
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

const CLAVES_BOLSA_INTERNA = Object.freeze(Object.keys(MENSAJES_BOLSA_INTERNA_ES));
export function crearTraductorBolsaInterna(catalogo = MENSAJES_BOLSA_INTERNA_ES) {
  if (!catalogo || typeof catalogo !== "object" || CLAVES_BOLSA_INTERNA.some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === "")) {
    throw new Error("catálogo i18n de Bolsa interna incompleto");
  }
  return (clave, variables = {}) => {
    if (!CLAVES_BOLSA_INTERNA.includes(clave)) throw new Error(`clave i18n de Bolsa interna desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/g, (_coincidencia, variable) => String(variables[variable] ?? ""));
  };
}
export const traducirBolsaInterna = crearTraductorBolsaInterna();
/** Localización y zona horaria del portal: autoridad común para formatear fechas, horas, importes y cifras. */
export const LOCALIZACION_PORTAL = "es-ES";
export const ZONA_HORARIA_PORTAL = "Europe/Madrid";
export function formatearNumeroPortal(valor, opciones = {}) {
  const numero = Number(valor);
  return Number.isFinite(numero) ? new Intl.NumberFormat(LOCALIZACION_PORTAL, opciones).format(numero) : String(valor ?? "");
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
  return new Intl.DateTimeFormat(LOCALIZACION_PORTAL, hora ? { dateStyle: "short", timeStyle: "short" } : { dateStyle: "short" }).format(fecha);
}
