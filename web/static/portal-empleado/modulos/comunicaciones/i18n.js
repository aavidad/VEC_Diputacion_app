/** Catálogo cerrado de la vista de Comunicaciones. */
export const MENSAJES_COMUNICACIONES_ES = Object.freeze({
  cabecera_sobrelinea: "Comunicaciones",
  titulo: "Comunicaciones y avisos",
  descripcion: "Bandeja pendiente de una fuente autorizada. Un aviso no acredita envío, entrega, notificación ni lectura.",
  estado_resumen: "No configurado · ver límites y dependencias",
  no_configurado: "No configurado",
  sin_fuente_resumen: "Esta bandeja no consulta comunicaciones ni preferencias porque todavía no tiene una fuente autorizada.",
  pendiente_fuente: "Consulta autorizada de comunicaciones, con ámbito y finalidad.",
  pendiente_canal: "Canal corporativo y destinatario del alta propia de VEC.",
  pendiente_recibo: "Recibos y evidencias diferenciadas de transporte, entrega y lectura.",
  conexion_pendiente: "Sin API ni canal activos en esta vista",
  navegacion: "Secciones de Comunicaciones",
  pestana_bandeja: "Bandeja",
  pestana_preferencias: "Preferencias",
  pestana_plantillas: "Plantillas",
  pestana_administrativas: "Administrativas",
  seccion_seleccionada: "Sección {seccion} seleccionada.",
  bandeja_sobrelinea: "Comunicaciones propias",
  bandeja_titulo: "Bandeja",
  visibles: "{numero} visibles",
  region_bandeja: "Bandeja de comunicaciones",
  caption_bandeja: "Comunicaciones y etapas de evidencia",
  asunto: "Asunto",
  canal_previsto: "Canal previsto",
  evidencia_transporte: "Aceptado por transporte",
  evidencia_entrega: "Entregado al destinatario",
  evidencia_lectura: "Leído por el destinatario",
  bandeja_vacia: "No hay comunicaciones para mostrar: la fuente todavía no está conectada.",
  detalle_sobrelinea: "Seguimiento de evidencias",
  trazabilidad_titulo: "Transporte, entrega y lectura",
  trazabilidad_ayuda: "La aceptación del transporte solo confirmaría la recepción por el canal técnico. La entrega y la lectura requieren evidencias distintas.",
  sin_fuente: "Sin fuente",
  limite_detalle: "No hay despacho, entrega, lectura ni notificación fehaciente acreditados.",
  enviar: "Enviar",
  enviar_motivo: "Envío no disponible: faltan un canal corporativo conectado, el destinatario del alta propia de VEC y un recibo verificable.",
  preferencias_titulo: "Preferencias de canal",
  preferencias_ayuda: "No se consultan ni modifican preferencias hasta conectar su fuente autorizada.",
  cambiar_preferencias: "Cambiar preferencias",
  cambiar_preferencias_motivo: "Disponible al conectar la fuente autorizada de preferencias.",
  plantillas_titulo: "Plantillas",
  plantillas_ayuda: "No hay plantillas disponibles en esta vista sin un catálogo versionado conectado.",
  redactar: "Redactar comunicación",
  redactar_motivo: "Disponible al conectar plantilla, autorización y recibo.",
  administrativas_titulo: "Campañas y comunicaciones administrativas",
  administrativas_ayuda: "La gestión colectiva requiere autorización específica, audiencia y auditoría. Esta vista no selecciona destinatarios ni prepara despachos.",
  crear_campana: "Crear campaña",
  crear_campana_motivo: "Disponible con autorización específica, audiencia y auditoría.",
});

const CLAVES = Object.freeze(Object.keys(MENSAJES_COMUNICACIONES_ES));
export function crearTraductorComunicaciones(catalogo = MENSAJES_COMUNICACIONES_ES) {
  if (!catalogo || typeof catalogo !== "object" || CLAVES.some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === "")) throw new Error("catálogo i18n de Comunicaciones incompleto");
  return (clave, variables = {}) => {
    if (!CLAVES.includes(clave)) throw new Error(`clave i18n de Comunicaciones desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/g, (_coincidencia, variable) => String(variables[variable] ?? ""));
  };
}
