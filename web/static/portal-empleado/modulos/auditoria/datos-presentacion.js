import { obtenerAtlasSinteticoRRHH, TEXTO_DATOS_FICTICIOS_RRHH } from "../../datos-sinteticos-rrhh.js";

const atlas = obtenerAtlasSinteticoRRHH();

/** Muestra minimizada para revisar el diseño; no es un registro de auditoría. */
export const EVENTOS_AUDITORIA_PRESENTACION = Object.freeze([
  Object.freeze({ id: "AUD-2026-0918-014", instante: "18/09/2026 · 11:42", modulo: "Contratación", vista: "cambios", operacion: "Preparación de informe", resultado: "Correcto", actor: "E. M. R. · RRHH", recurso: "Expediente CT-2026-0184", decision: "Borrador preparado", recibo: "rec_sint_CT_9F3A", huella: "a8c1…7d4e", correlacion: "cor_sint_5K8P" }),
  Object.freeze({ id: "AUD-2026-0918-011", instante: "18/09/2026 · 10:18", modulo: "Dietas", vista: "documentos", operacion: "Consulta de justificante", resultado: "Denegado", actor: "A. L. F. · persona", recurso: "Comisión DIE-2026-0047", decision: "Ámbito pendiente de validar", recibo: "Sin recibo", huella: "No disponible", correlacion: "cor_sint_2N6R" }),
  Object.freeze({ id: "AUD-2026-0917-032", instante: "17/09/2026 · 16:04", modulo: "Cronos", vista: "accesos", operacion: "Acceso a incidencia", resultado: "Correcto", actor: "M. C. R. · responsable", recurso: "Incidencia CRO-2026-0121", decision: "Consulta visual permitida", recibo: "rec_sint_CRO_4W2Q", huella: "b2d9…1a0c", correlacion: "cor_sint_7H3M" }),
  Object.freeze({ id: "AUD-2026-0917-009", instante: "17/09/2026 · 09:26", modulo: "Personal", vista: "autorizacion", operacion: "Comprobación de concesión", resultado: "Pendiente", actor: "E. M. R. · RRHH", recurso: `Unidad ${atlas.unidad.nombre_visible}`, decision: "Permiso y finalidad sin conexión", recibo: "No generado", huella: "No disponible", correlacion: "cor_sint_8T1V" }),
  Object.freeze({ id: "AUD-2026-0916-026", instante: "16/09/2026 · 14:53", modulo: "Conectores", vista: "conectores", operacion: "Preparación de envío", resultado: "Pendiente", actor: "Sistema · enmascarado", recurso: "Correo corporativo", decision: "Conector no configurado", recibo: "No generado", huella: "No disponible", correlacion: "cor_sint_1C9L" }),
  Object.freeze({ id: "AUD-2026-0916-004", instante: "16/09/2026 · 08:31", modulo: "Bolsa", vista: "cambios", operacion: "Lectura de posición", resultado: "Correcto", actor: "E. M. R. · RRHH", recurso: "Bolsa BOL-2026-0008", decision: "Proyección sintética consultada", recibo: "rec_sint_BOL_6J5E", huella: "e3f0…9b6a", correlacion: "cor_sint_4D7S" }),
]);

export const ESTADO_AUDITORIA_PRESENTACION = Object.freeze({
  estado: "visual_pendiente_backend",
  resumen: "La pantalla ordena una muestra sintética para revisar el recorrido. No consulta el almacén de auditoría ni acredita accesos, decisiones o evidencias reales.",
  pendientes: Object.freeze([
    "Almacén durable, segregado y de solo adición",
    "Permisos, ámbitos, finalidad y separación de funciones",
    "Conservación, bloqueo y política de archivo aplicable",
    "Evidencias verificables con correlación y huellas completas",
    "Exportación firmada y controlada",
  ]),
  fuente: Object.freeze({ etiqueta: TEXTO_DATOS_FICTICIOS_RRHH }),
  conexion: "Sin API, almacén durable ni conector activos en esta pantalla",
});

export const KPIS_AUDITORIA_PRESENTACION = Object.freeze([
  Object.freeze({ etiqueta: "Eventos visibles", valor: "6", nota: "muestra sintética" }),
  Object.freeze({ etiqueta: "Accesos correctos", valor: "2", nota: "no acreditados" }),
  Object.freeze({ etiqueta: "Decisiones pendientes", valor: "2", nota: "sin efecto" }),
  Object.freeze({ etiqueta: "Evidencias verificables", valor: "0", nota: "conexión pendiente" }),
]);
