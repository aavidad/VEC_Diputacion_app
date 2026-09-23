/** Textos de la vista de contratos y reincorporación de Bolsa. */
export const MENSAJES_CONTRATOS_ES = Object.freeze({
  sobrelinea: "Bolsa · continuidad del expediente",
  titulo: "Contratos, ceses y reincorporaciones",
  descripcion: "Presentación de relaciones y disponibilidad con fuente autorizada. Esta vista no acredita por sí sola relación jurídica, cese ni incorporación.",
  estado_cargando: "Cargando",
  estado_disponible: "Disponible",
  estado_vacio: "Sin registros",
  estado_no_configurado: "No configurado",
  estado_denegado: "Acceso denegado",
  estado_error: "Error de consulta",
  detalle_cargando: "Se está consultando la fuente autorizada. No hay filas disponibles todavía.",
  detalle_disponible: "Se muestran únicamente las relaciones recibidas de la fuente autorizada.",
  detalle_vacio: "La fuente respondió correctamente y no devolvió relaciones.",
  detalle_no_configurado: "Esta vista no tiene una fuente de contratos configurada. No se muestran registros ni se habilitan actuaciones.",
  detalle_denegado: "El perfil actual no tiene permiso para consultar estas relaciones.",
  detalle_error: "No se pudo consultar la fuente de relaciones. Reintente desde el acceso autorizado.",
  circuito_titulo: "Circuito entre módulos",
  circuito_subtitulo: "Cada autoridad debe confirmar su propio hecho antes de avanzar.",
  paso_bolsa: "Propuesta y aceptación · Bolsa",
  paso_bolsa_descripcion: "Una propuesta de llamamiento no acredita aceptación.",
  paso_formalizacion: "Formalización y firma · Contratación",
  paso_formalizacion_descripcion: "El borrador y la autenticación no son firma.",
  paso_personal: "Relación e incorporación · Personal",
  paso_personal_descripcion: "Solo Personal confirma la relación y la incorporación.",
  paso_ginpix: "Entrega a GINPIX · Contratación",
  paso_ginpix_descripcion: "La ficha o descarga no acredita entrega al sistema.",
  paso_reincorporacion: "Cese y disponibilidad · Personal y Bolsa",
  paso_reincorporacion_descripcion: "El cese acreditado precede a la política de Bolsa.",
  registros_titulo: "Relaciones consultadas",
  registros_subtitulo: "Solo se presentan filas aportadas por la fuente autorizada.",
  tabla_titulo: "Contratos, ceses y reincorporaciones",
  columna_expediente: "Expediente",
  columna_acto: "Acto",
  columna_bolsa: "Bolsa",
  columna_fechas: "Inicio / fin",
  columna_estado: "Estado",
  vacio: "No hay relaciones disponibles para mostrar.",
  acciones_titulo: "Actuaciones pendientes de conexión",
  acciones_subtitulo: "Se habilitarán solo con fuente autorizada, operación durable y recibo recuperable.",
  accion_contrato: "Registrar contrato",
  motivo_contrato: "Falta confirmación de Personal, autorización y recibo durable.",
  accion_cese: "Registrar cese",
  motivo_cese: "Falta un cese acreditado por Personal y su recibo.",
  accion_reincorporar: "Reincorporar en Bolsa",
  motivo_reincorporar: "Faltan el cese acreditado y la política versionada de Bolsa.",
  ayuda: "? Ayuda sobre el circuito",
  ayuda_contenido: "Contratación coordina referencias; Bolsa conserva propuestas, aceptaciones, orden y disponibilidad. Personal conserva relaciones e incorporaciones. La firma y la entrega a GINPIX requieren sus propios recibos. Tras un cese, Bolsa aplica la política vigente a la persona, sin inferir disponibilidad desde esta pantalla.",
});

export function crearTraductorContratos(catalogo = MENSAJES_CONTRATOS_ES) {
  const claves = Object.keys(MENSAJES_CONTRATOS_ES);
  if (!catalogo || typeof catalogo !== "object" || claves.some((clave) => typeof catalogo[clave] !== "string" || !catalogo[clave])) {
    throw new TypeError("catálogo i18n de contratos incompleto");
  }
  return (clave) => {
    if (!claves.includes(clave)) throw new TypeError(`clave i18n de contratos desconocida: ${clave}`);
    return catalogo[clave];
  };
}

export const traducirContratos = crearTraductorContratos();
