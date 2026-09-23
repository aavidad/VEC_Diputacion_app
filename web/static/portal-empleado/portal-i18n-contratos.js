/** Textos de la vista de contratos y reincorporación de Bolsa. */
export const MENSAJES_CONTRATOS_ES = Object.freeze({
  sobrelinea: "Bolsa · continuidad del expediente",
  titulo: "Contratos, ceses y reincorporaciones",
  descripcion: "Lectura de ejemplos sintéticos. Esta vista no acredita relación jurídica, cese ni incorporación.",
  aviso_sintetico: "Los registros son ejemplos sintéticos. Ninguna acción de esta vista crea una relación jurídica ni modifica la disponibilidad de Bolsa.",
  circuito_titulo: "Circuito entre módulos",
  circuito_subtitulo: "Cada autoridad debe confirmar su propio hecho antes de avanzar.",
  sin_conector: "Sin conectores en esta vista",
  sin_fuente: "Sin fuente aquí",
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
  registros_titulo: "Referencias de ejemplo",
  registros_subtitulo: "El estado mostrado pertenece solo a la muestra; no procede de Personal ni de Bolsa.",
  tabla_titulo: "Contratos, ceses y reincorporaciones sintéticos",
  columna_expediente: "Expediente",
  columna_acto: "Acto de ejemplo",
  columna_bolsa: "Bolsa",
  columna_fechas: "Inicio / fin",
  columna_estado: "Estado en muestra",
  vacio: "No hay ejemplos de contratos para mostrar.",
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
