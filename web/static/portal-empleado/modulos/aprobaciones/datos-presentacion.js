import { obtenerAtlasSinteticoRRHH } from "../../datos-sinteticos-rrhh.js";

const BASE = Object.freeze({
  estado: "visual_pendiente_backend",
  circuitos: Object.freeze([
    Object.freeze({ id: "todos", etiqueta: "Todos los circuitos" }),
    Object.freeze({ id: "contratacion", etiqueta: "Contratación temporal" }),
    Object.freeze({ id: "dietas", etiqueta: "Dietas y desplazamientos" }),
    Object.freeze({ id: "cronos", etiqueta: "Cronos y permisos" }),
    Object.freeze({ id: "bolsas", etiqueta: "Bolsas y selección" }),
  ]),
  pendientes: Object.freeze([
    Object.freeze({ id: "APR-2026-0148", circuito: "contratacion", tipo: "Informe de necesidad", asunto: "Cobertura temporal · Técnico/a de Gestión", solicitante: "Elena Martín Rojas", unidad: "Servicio de Administración de Personal", prioridad: "Alta", estado: "Pendiente de revisión", recibido: "20/09/2026 · 09:10", firmantes: "1 de 3 revisiones", importe: "Sin importe asociado", documentos: ["Informe de necesidad", "Memoria justificativa", "Propuesta de cobertura"], revisiones: [["Preparación", "Elena Martín Rojas", "Registrada visualmente", "20/09/2026 · 09:10"], ["Revisión de RRHH", "María del Carmen Ruiz Soto", "Pendiente", "Sin conexión"], ["Conformidad", "Antonio López Fernández", "Pendiente", "Sin conexión"]], autoridad: [["Revisión de RRHH", "Ámbito y competencia pendientes de validar", "Pendiente de conexión"], ["Conformidad de unidad", "Matriz de autoridad no conectada", "No evaluada"]], recibo: "REC-APR-SINT-0148" }),
    Object.freeze({ id: "APR-2026-0143", circuito: "dietas", tipo: "Liquidación de comisión", asunto: "Comisión de servicio · Guadix", solicitante: "Antonio López Fernández", unidad: "Servicio de Infraestructuras Provinciales", prioridad: "Normal", estado: "Pendiente de revisión", recibido: "19/09/2026 · 16:25", firmantes: "0 de 2 revisiones", importe: "184,60 € mostrados", documentos: ["Liquidación", "Justificantes declarados", "Itinerario"], revisiones: [["Preparación", "Antonio López Fernández", "Registrada visualmente", "19/09/2026 · 16:25"], ["Aprobación de jefatura", "Javier Moreno Gil", "Pendiente", "Sin conexión"]], autoridad: [["Aprobación de jefatura", "Relación de jefatura pendiente de fuente", "No evaluada"], ["Validación económica", "Regla y umbral pendientes", "No evaluada"]], recibo: "REC-APR-SINT-0143" }),
    Object.freeze({ id: "APR-2026-0139", circuito: "cronos", tipo: "Solicitud de permiso", asunto: "Permiso por asunto particular · 1 día", solicitante: "Lucía Serrano Vega", unidad: "Área de Obras y Servicios", prioridad: "Normal", estado: "Pendiente de revisión", recibido: "19/09/2026 · 11:40", firmantes: "0 de 1 revisión", importe: "1 día solicitado", documentos: ["Solicitud de permiso", "Calendario de ausencia"], revisiones: [["Solicitud", "Lucía Serrano Vega", "Registrada visualmente", "19/09/2026 · 11:40"], ["Aprobación responsable", "Javier Moreno Gil", "Pendiente", "Sin conexión"]], autoridad: [["Aprobación responsable", "Calendario y ámbito pendientes de conexión", "No evaluada"]], recibo: "REC-APR-SINT-0139" }),
    Object.freeze({ id: "APR-2026-0131", circuito: "bolsas", tipo: "Propuesta de llamamiento", asunto: "Bolsa de Técnico/a de Administración General", solicitante: "Elena Martín Rojas", unidad: "Servicio de Administración de Personal", prioridad: "Alta", estado: "Pendiente de revisión", recibido: "18/09/2026 · 12:05", firmantes: "1 de 2 revisiones", importe: "Sin importe asociado", documentos: ["Propuesta de llamamiento", "Orden de bolsa", "Informe de disponibilidad"], revisiones: [["Preparación", "Elena Martín Rojas", "Registrada visualmente", "18/09/2026 · 12:05"], ["Revisión de selección", "María del Carmen Ruiz Soto", "Pendiente", "Sin conexión"]], autoridad: [["Revisión de selección", "Competencia y orden de bolsa pendientes", "No evaluada"], ["Firma de resolución", "Portafirmas no integrado", "No disponible"]], recibo: "REC-APR-SINT-0131" }),
  ]),
  suplencias: Object.freeze([
    Object.freeze({ tipo: "Suplencia mostrada", titular: "Javier Moreno Gil", cobertura: "María del Carmen Ruiz Soto", alcance: "Revisión visual de Dietas", vigencia: "Pendiente de fuente y validación" }),
    Object.freeze({ tipo: "Delegación mostrada", titular: "María del Carmen Ruiz Soto", cobertura: "Elena Martín Rojas", alcance: "Preparación de expedientes", vigencia: "No produce efectos ni firma" }),
  ]),
});

/** Datos locales de presentación; no son una fuente de expedientes, autoridad ni firma. */
export function obtenerDatosAprobacionesPresentacion() {
  const atlas = obtenerAtlasSinteticoRRHH();
  return Object.freeze({ ...BASE, aviso: atlas.aviso_visible, responsable: atlas.responsable.nombre_visible, tecnica: atlas.tecnica_rrhh.nombre_visible });
}
