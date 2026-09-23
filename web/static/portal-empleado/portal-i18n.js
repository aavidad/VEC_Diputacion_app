/** Catálogo común de los estados del shell y del acceso a Borradores. */
export const MENSAJES_PORTAL_ES = Object.freeze({
  contexto_portal_titulo: "Portal interno",
  contexto_portal_descripcion: "Identidad personal no mostrada",
  contexto_portal_accesible: "Portal interno. Identidad personal no mostrada",
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
  titulo_error_catalogo_modulos: "No se pudieron comprobar los módulos",
  error_catalogo_modulos: "El catálogo interno de módulos no está disponible. Reintente la comprobación.",
  personal_catalogo_profesional: "Catálogo profesional de Personal",
  descripcion_superficie_no_montada: "No se pudo montar la superficie solicitada.",
  contratacion_temporal_encabezado: "Contratación temporal",
  contratacion_temporal_miga: "Portal del Empleado → Contratación temporal",
  contratacion_temporal_titulo: "Gestión de expedientes de contratación temporal",
  contratacion_temporal_descripcion_no_disponible:
    "La composición real de Contratación temporal todavía no está disponible en este portal.",
  contratacion_temporal_aviso_no_disponible: "Esta vista no monta el módulo ni habilita sus operaciones.",
  accion_volver_portal: "Volver al portal",
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
  boton_perfil_sin_permiso: "El perfil de presentación no permite esta operación",
  boton_capacidad_no_conectada: "Capacidad de servidor no conectada",
  tabla_sin_registros: "No hay registros para los filtros aplicados.",
  tabla_region_operativa: "Tabla operativa: {titulo}",
  fuente_datos_sinteticos: "Datos sintéticos · Memoria volátil",
  fuente_real_no_conectada: "Fuente real no conectada",
  alcance_presentacion: "Alcance de la presentación",
  modo_presentacion: "Modo presentación.",
  aviso_presentacion: "Este recorrido simula la actuación sin efectos administrativos ni comunicaciones externas.",
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
  contacto_registrado: "Contacto registrado. Recibo {recibo}.",
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
export function formatearNumeroPortal(valor, opciones = {}) {
  const numero = Number(valor);
  return Number.isFinite(numero) ? new Intl.NumberFormat("es-ES", opciones).format(numero) : String(valor ?? "");
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
  return new Intl.DateTimeFormat("es-ES", hora ? { dateStyle: "short", timeStyle: "short" } : { dateStyle: "short" }).format(fecha);
}
