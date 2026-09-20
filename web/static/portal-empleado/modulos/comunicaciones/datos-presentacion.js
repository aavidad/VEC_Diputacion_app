import { obtenerAtlasSinteticoRRHH, TEXTO_DATOS_FICTICIOS_RRHH } from "../../datos-sinteticos-rrhh.js";

/** Datos sintéticos y minimizados para la revisión visual de RRHH. */
const atlas = obtenerAtlasSinteticoRRHH();
export const ATLAS_COMUNICACIONES = Object.freeze({
  fuente: TEXTO_DATOS_FICTICIOS_RRHH,
  persona: atlas.persona_principal.nombre_visible,
  avisos: Object.freeze([
    Object.freeze({ id: "COM-2026-0184", tipo: "Aviso personal", asunto: "Actualización de documentación de Dietas", canal: "Buzón interno", contacto: "a•••••.l••••@d••••.es", fecha: "18/09/2026 · 10:32", estado: "Pendiente de lectura", plantilla: "Recordatorio de justificantes v2", expediente: "DIE-2026-0047", historico: ["Generada desde plantilla v2", "Pendiente de conector de buzón interno", "Sin acuse ni entrega acreditados"] }),
    Object.freeze({ id: "COM-2026-0179", tipo: "Comunicación administrativa", asunto: "Borrador de resolución disponible para revisión", canal: "Correo corporativo", contacto: "a•••••.l••••@d••••.es", fecha: "17/09/2026 · 14:05", estado: "Preparada · no enviada", plantilla: "Resolución de contratación temporal v4", expediente: "2026/CT-000184", historico: ["Contenido preparado", "Correo corporativo pendiente de conectar", "No existe evidencia de envío o entrega"] }),
    Object.freeze({ id: "COM-2026-0168", tipo: "Aviso personal", asunto: "Solicitud de ausencia pendiente de validación", canal: "SMS", contacto: "••• •• 42 18", fecha: "16/09/2026 · 08:45", estado: "Canal pendiente", plantilla: "Aviso a responsable de unidad v1", expediente: "CRO-2026-0121", historico: ["Preferencia de SMS declarada en datos sintéticos", "Conector SMS pendiente", "No se ha despachado ningún mensaje"] }),
  ]),
  plantillas: Object.freeze([
    Object.freeze({ nombre: "Resolución de contratación temporal v4", uso: "Contratación temporal", version: "v4", estado: "Visible · revisión jurídica pendiente" }),
    Object.freeze({ nombre: "Recordatorio de justificantes v2", uso: "Dietas", version: "v2", estado: "Visible · pendiente de canal" }),
    Object.freeze({ nombre: "Aviso a responsable de unidad v1", uso: "Cronos", version: "v1", estado: "Visible · pendiente de auditoría" }),
  ]),
  preferencias: Object.freeze([
    Object.freeze({ canal: "Buzón interno", finalidad: "Avisos personales", estado: "Preferencia sintética visible", consentimiento: "Pendiente de fuente autorizada" }),
    Object.freeze({ canal: "Correo corporativo", finalidad: "Comunicaciones administrativas", estado: "No configurado", consentimiento: "Pendiente de alta propia VEC" }),
    Object.freeze({ canal: "SMS", finalidad: "Avisos urgentes", estado: "No configurado", consentimiento: "Pendiente de base y preferencia" }),
  ]),
});

export const ESTADO_COMUNICACIONES = Object.freeze({
  estado: "visual_pendiente_backend",
  resumen: "La bandeja, las preferencias y el historial son una superficie visual de presentación; no consulta ni modifica comunicaciones reales.",
  pendientes: Object.freeze([
    "Caso de uso autorizado con permisos, ámbito y finalidad",
    "Conectores de correo corporativo y SMS con reintentos e idempotencia",
    "Registro de consentimiento y preferencias desde la fuente autorizada",
    "Documentos, auditoría, recibos y evidencia de despacho o entrega",
  ]),
  fuente: Object.freeze({ etiqueta: TEXTO_DATOS_FICTICIOS_RRHH }),
  conexion: "Sin API ni conectores activos en esta pantalla",
});
