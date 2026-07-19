/** Catálogo sintético del shell, excluido de producción por su nombre. */
import { validarCatalogoModulosPresentacion } from "./portal-catalogo-modulos.js";

const CATALOGO = [
  { clave: "bolsa", sigla: "BOL", titulo: "Bolsas de trabajo", texto: "Elaboración, integrantes, llamamientos, contratos, documentos y trazabilidad." },
  { clave: "personal", sigla: "PER", titulo: "Personal", texto: "Datos personales, puesto, situación administrativa y servicios prestados." },
  { clave: "nominas", sigla: "NOM", titulo: "Nóminas", texto: "Recibos, certificados fiscales e incidencias retributivas." },
  { clave: "cronos", sigla: "CRO", titulo: "Cronos", texto: "Fichajes, permisos, vacaciones, turnos y calendario laboral." },
  { clave: "dietas", sigla: "DIE", titulo: "Dietas", texto: "Comisiones de servicio, kilometraje, gastos y liquidaciones." },
  { clave: "solicitudes", sigla: "SOL", titulo: "Solicitudes y certificados", texto: "Ventanilla única, aportación documental y seguimiento." },
  { clave: "meritos", sigla: "RUM", titulo: "Méritos y formación", texto: "Títulos, cursos, experiencia y evidencias reutilizables." },
  { clave: "comunicaciones", sigla: "AVI", titulo: "Comunicaciones", texto: "Avisos personales, notificaciones y preferencias de canal." },
  { clave: "documentos", sigla: "DOC", titulo: "Documentos y firma", texto: "Repositorio documental, generación, firma, verificación y descarga autorizada." },
  { clave: "aprobaciones", sigla: "APR", titulo: "Aprobaciones y portafirmas", texto: "Circuitos de revisión, visto bueno, firma múltiple y seguimiento." },
  { clave: "auditoria", sigla: "AUD", titulo: "Auditoría", texto: "Registro de accesos, operaciones, decisiones, evidencias y exportaciones." },
  { clave: "administracion", sigla: "ADM", titulo: "Administración y configuración", texto: "Roles, permisos, catálogos, calendarios, reglas y conectores." },
];

export function obtenerCatalogoModulosPresentacion() {
  return validarCatalogoModulosPresentacion(structuredClone(CATALOGO));
}
