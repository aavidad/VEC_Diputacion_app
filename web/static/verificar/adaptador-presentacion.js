/** Adaptador local y no autoritativo, excluido del artefacto productivo. */
export function cotejarDocumentoPresentacion(referencia) {
  if (typeof referencia !== "string" || !/^DEMO-(?:REC|DIE-REC|CRONOS-REC)-[A-Z0-9-]{4,64}$/.test(referencia)) {
    return Object.freeze({ valido: false, mensaje: "La referencia no pertenece al catálogo sintético de la presentación." });
  }
  return Object.freeze({
    valido: true,
    titulo: "Documento de demostración reconocido",
    mensaje: "La referencia corresponde a un recibo generado localmente para revisar el recorrido. No acredita una actuación administrativa real.",
    referencia,
    estado: "DEMO · sin validez administrativa",
    alcance: "Comprobación local, sin consulta a registros, firma o sello reales",
  });
}
