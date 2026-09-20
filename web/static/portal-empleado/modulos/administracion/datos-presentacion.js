import { obtenerAtlasSinteticoRRHH, TEXTO_DATOS_FICTICIOS_RRHH } from "../../datos-sinteticos-rrhh.js";

const atlas = obtenerAtlasSinteticoRRHH();

export const PESTANAS_ADMINISTRACION = Object.freeze(["resumen", "roles", "catalogos", "calendarios", "reglas", "conectores", "modulos", "privacidad", "ia"]);

export const DATOS_ADMINISTRACION_PRESENTACION = Object.freeze({
  aviso: TEXTO_DATOS_FICTICIOS_RRHH,
  responsable: atlas.tecnica_rrhh.nombre_visible,
  unidad: atlas.unidad.nombre_visible,
  roles: Object.freeze([
    ["Gestión de RRHH", "Expedientes, propuestas y revisión", "Ámbito: Personas"],
    ["Jefatura de unidad", "Solicitud y seguimiento de su unidad", `Ámbito: ${atlas.unidad.nombre_visible}`],
    ["Administración de seguridad", "Roles, retención y conectores", "Acceso privilegiado · pendiente de autorización"],
  ]),
  catalogos: Object.freeze([
    ["Puestos RPT", "Importación de referencia", "Pendiente de fuente RPT autorizada"],
    ["Categorías y especialidades", "Versión de catálogo", "Borrador de presentación"],
    ["Centros y jefaturas", "Estructura organizativa", "Pendiente de sincronización corporativa"],
    ["Modalidades y causas", "Contratación temporal", "Gobernado · pendiente de publicación"],
  ]),
  calendarios: Object.freeze([
    ["Calendario laboral provincial 2026", "Ámbito provincial", "Borrador visual"],
    ["Cierres de gestión RRHH", "Recursos Humanos", "Pendiente de calendario corporativo"],
    ["Jornada de referencia", atlas.unidad.nombre_visible, "Pendiente de Cronos"],
  ]),
  reglas: Object.freeze([
    ["Conservación de expedientes", "Versión propuesta 1", "Revisión jurídica y doble control pendientes"],
    ["Priorización de bolsa", "Regla declarativa", "Pendiente de aprobación y motor versionado"],
    ["Avisos de plazos", "Sin plazo operativo", "No configurado hasta decisión de RRHH"],
  ]),
  conectores: Object.freeze([
    ["Directorio corporativo", "Sin conexión", "Identidad y asignaciones"],
    ["Correo corporativo", "Sin conexión", "Despacho y evidencia"],
    ["RPT / estructura", "Sin conexión", "Puestos, centros y jefaturas"],
    ["GINPIX", "Sin conexión", "Consulta o incorporación"],
  ]),
  modulos: Object.freeze([
    ["Contratación temporal", "Presentación disponible", "Conexión progresiva con backend Go"],
    ["Bolsa", "Presentación disponible", "Reglas y llamamientos pendientes de circuito completo"],
    ["Cronos", "Presentación disponible", "Proveedor y autorización pendientes"],
    ["Dietas", "Presentación disponible", "Efectos y justificación pendientes"],
    ["Personal", "Presentación disponible", "Relación y fuentes autorizadas pendientes"],
  ]),
  privacidad: Object.freeze([
    ["Minimización", "Datos ficticios en esta presentación", "Sin datos personales ni secretos"],
    ["Auditoría", "Diseño pendiente de conexión", "Accesos y cambios requieren registro segregado"],
    ["Conservación", "Sin política activa", "Revisión jurídica, configuración versionada y doble control"],
    ["Revocación", "Sin operación disponible", "Autorización vigente y efecto auditable pendientes"],
  ]),
});

export function crearEstadoAdministracion(t) {
  if (typeof t !== "function") throw new TypeError("traductor de Administración no disponible");
  return Object.freeze({
  estado: "visual_pendiente_backend",
  resumen: t("estado_resumen"),
  pendientes: Object.freeze([
    t("estado_pendiente_1"), t("estado_pendiente_2"), t("estado_pendiente_3"), t("estado_pendiente_4"),
  ]),
  fuente: Object.freeze({ etiqueta: TEXTO_DATOS_FICTICIOS_RRHH }),
  conexion: t("estado_conexion"),
  });
}
