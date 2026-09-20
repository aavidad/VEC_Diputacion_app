import { obtenerAtlasSinteticoRRHH, TEXTO_DATOS_FICTICIOS_RRHH } from "../../datos-sinteticos-rrhh.js";

const BASE = Object.freeze({
  documentos: [
    { id: "doc-informe", titulo: "Informe jurídico de necesidad", tipo: "Informe", expediente: "CT-2026-0148", version: "v3", fecha: "18/09/2026", responsable: "Elena Martín Rojas", estado: "Pendiente de firma admitida", estado_clave: "pendiente", huella: "SHA-256 · 94a8…c120", conservacion: "Pendiente de repositorio documental", circuito: "Revisión jurídica → firma" },
    { id: "doc-resolucion", titulo: "Propuesta de resolución", tipo: "Resolución", expediente: "CT-2026-0148", version: "v2", fecha: "17/09/2026", responsable: "María del Carmen Ruiz Soto", estado: "Borrador preparado", estado_clave: "neutro", huella: "SHA-256 · 7d31…9e44", conservacion: "Pendiente de repositorio documental", circuito: "RRHH → Intervención → firma" },
    { id: "doc-rc", titulo: "Documento RC de cobertura", tipo: "Presupuestario", expediente: "CT-2026-0148", version: "v1", fecha: "16/09/2026", responsable: "Antonio López Fernández", estado: "Pendiente de validación", estado_clave: "aviso", huella: "SHA-256 · a829…5b10", conservacion: "Pendiente de validación de origen", circuito: "Validación presupuestaria" },
    { id: "doc-diligencia", titulo: "Diligencia de formalización", tipo: "Diligencia", expediente: "CT-2026-0148", version: "v1", fecha: "15/09/2026", responsable: "Elena Martín Rojas", estado: "Plantilla disponible", estado_clave: "neutro", huella: "Sin huella final", conservacion: "Pendiente de generación y repositorio", circuito: "Preparación → firma" },
  ],
  plantillas: [["Informe jurídico", "Versión gobernada pendiente de conexión"], ["Resolución", "Revisión y firma manual pendientes"], ["Diligencia", "Modelo de formalización pendiente"], ["Comunicación al centro", "Entrega corporativa pendiente"]],
});

export function obtenerDatosDocumentosPresentacion() {
  const atlas = obtenerAtlasSinteticoRRHH();
  return Object.freeze({ ...structuredClone(BASE), atlas, aviso: TEXTO_DATOS_FICTICIOS_RRHH });
}
