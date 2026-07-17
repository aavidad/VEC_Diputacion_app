/**
 * ADAPTADOR EXCLUSIVO DE PRESENTACIÓN RRHH.
 *
 * No es persistencia, no es una API y no puede habilitar actos administrativos.
 * La aplicación normal no importa este fichero: solo se carga dinámicamente con
 * `?presentacion=rrhh`. Véase el mapa de sustitución en
 * `docs/portal_vec/entregable_rrhh_bolsa_2026-07-17.md`.
 */
const DATOS = {
  esquema: "vec.bolsa.panel.presentacion.v1",
  demostracion: true,
  sesion: { iniciales: "MP", nombre: "María Pérez", perfil: "Perfil de presentación RRHH" },
  indicadores: {
    avisos_pendientes: 3,
    bolsas_activas: 6,
    candidatos_disponibles: 67,
    candidatos_total: 264,
    llamamientos_pendientes: 3,
    contratos_activos: 24,
    cobertura_media: 94,
    tiempo_medio_cobertura: "2,4 días",
    renuncias_porcentaje: 7.2,
    respuesta_mediana_horas: 18,
    firmas_pendientes: 2,
    plazos_proximos: 1,
  },
  distribucion_global: {
    disponibles: 145,
    en_llamamiento: 48,
    contratados: 29,
    no_disponibles: 42,
  },
  series: {
    contratos_mes: [20, 38, 32, 58, 52, 68, 63],
    llamamientos_mes: [28, 40, 38, 64, 53, 72, 66],
    periodo_contratos: "Enero a julio de 2026",
  },
  avisos: [
    { texto: "Informe jurídico pendiente en BOL-2026-014." },
    { texto: "Tres llamamientos previstos en siete días." },
    { texto: "Dos circuitos de firma por configurar." },
  ],
  configuracion_llamamiento: {
    regla: "Reglamento y bases · versión de presentación",
    apertura: "2026-07-20T09:00",
    apertura_visible: "20/07/2026 09:00",
    plazo_respuesta: "24 horas",
    tipo_cobertura: "Sustitución temporal",
    destino: "Servicios Centrales",
    jornada: "Completa",
    duracion: "3 meses",
    canales: ["Correo", "Telegram", "Aviso interno"],
    observaciones: "Presentación demostrativa. Las condiciones definitivas procederán del expediente y de las bases aplicables.",
  },
  catalogos_llamamiento: {
    plazos_respuesta: ["24 horas", "48 horas", "72 horas"],
    tipos_cobertura: ["Sustitución temporal", "Vacante", "Acumulación de tareas", "Programa temporal"],
    jornadas: ["Completa", "Parcial 50 %", "Parcial 33 %", "Otra fracción definida"],
    ambitos_geograficos: ["Todas", "Área metropolitana", "Provincia"],
  },
  bolsas: [
    { id: "auxiliar-administrativo", nombre: "Auxiliar Administrativo", categoria: "Administración", creada: "15/01/2026", integrantes: 156, disponibles: 12, llamamiento: 4, cobertura: 96, estado: "Activa", regla: "Reglamento y bases · versión de presentación" },
    { id: "administrativo", nombre: "Administrativo", categoria: "Administración", creada: "10/02/2026", integrantes: 98, disponibles: 8, llamamiento: 2, cobertura: 94, estado: "Activa", regla: "Bases · versión de presentación" },
    { id: "trabajador-social", nombre: "Trabajador Social", categoria: "Servicios Sociales", creada: "20/03/2026", integrantes: 48, disponibles: 3, llamamiento: 3, cobertura: 92, estado: "Activa", regla: "Bases · versión de presentación" },
    { id: "auxiliar-enfermeria", nombre: "Auxiliar de Enfermería", categoria: "Sanidad", creada: "05/04/2026", integrantes: 312, disponibles: 25, llamamiento: 7, cobertura: 98, estado: "Activa", regla: "Bases · versión de presentación" },
    { id: "operario-servicios", nombre: "Operario Servicios Múltiples", categoria: "Servicios Generales", creada: "12/01/2026", integrantes: 278, disponibles: 15, llamamiento: 1, cobertura: 94, estado: "Activa", regla: "Bases · versión de presentación" },
    { id: "educador-social", nombre: "Educador Social", categoria: "Servicios Sociales", creada: "18/02/2026", integrantes: 66, disponibles: 4, llamamiento: 0, cobertura: 90, estado: "Activa", regla: "Bases · versión de presentación" },
  ],
  capacidades: { solicitar_propuesta_llamamiento: false, confirmar_llamamiento: false },
  necesidades_llamamiento: [
    { id: "NEC-2026-0045", referencia: "Necesidad nº 45", puesto: "Auxiliar Administrativo", bolsa_id: "auxiliar-administrativo", bolsa: "Auxiliar Administrativo", destino: "Servicios Centrales", jornada: "Completa", duracion: "3 meses", cobertura: "Sustitución temporal", fecha_limite: "20/07/2026 09:00", regla: "Reglamento y bases · versión de presentación", estado: "Pendiente de propuesta" },
    { id: "NEC-2026-0038", referencia: "Necesidad nº 38", puesto: "Operario Servicios Múltiples", bolsa_id: "operario-servicios", bolsa: "Operario Servicios Múltiples", destino: "Parque móvil provincial", jornada: "Completa", duracion: "6 meses", cobertura: "Vacante", fecha_limite: "21/07/2026 10:00", regla: "Bases · versión de presentación", estado: "Pendiente de propuesta" },
    { id: "NEC-2026-0012", referencia: "Necesidad nº 12", puesto: "Trabajador Social", bolsa_id: "trabajador-social", bolsa: "Trabajador Social", destino: "Centro comarcal Guadix", jornada: "Parcial 50 %", duracion: "3 meses", cobertura: "Programa temporal", fecha_limite: "22/07/2026 09:30", regla: "Bases · versión de presentación", estado: "Pendiente de propuesta" },
  ],
  elaboraciones: [
    { id: "BOL-2026-014", nombre: "Auxiliar Administrativo", expediente: "2026/PES-014", fase: "Validación jurídica", reglas: "v3 · 8 criterios", plazo: "22/07/2026", responsable: "Servicio de Selección", estado: "En revisión", version_bases: "v3 · Huella SHA-256 registrada", calendario: "Solicitud 01/08–20/08/2026", firmantes: "Jefatura RRHH · Secretaría · Diputada delegada" },
    { id: "BOL-2026-021", nombre: "Trabajador Social", expediente: "2026/PES-021", fase: "Configuración de baremo", reglas: "Borrador · 6 criterios", plazo: "29/07/2026", responsable: "Selección externa", estado: "Borrador", version_bases: "v1 · Borrador", calendario: "Pendiente de aprobación", firmantes: "Pendientes de configurar" },
    { id: "BOL-2026-009", nombre: "Operario Servicios Múltiples", expediente: "2026/PES-009", fase: "Publicación", reglas: "v2 · 5 criterios", plazo: "Publicado 11/07/2026", responsable: "Jefatura RRHH", estado: "Publicada", version_bases: "v2 · Huella SHA-256 registrada", calendario: "Plazo cerrado 10/07/2026", firmantes: "Circuito completado" },
  ],
  proximos: [
    { dia: "20", mes: "JUL", bolsa: "Auxiliar Administrativo", numero: "45", fecha: "20/07/2026 09:00", estado: "Pendiente" },
    { dia: "21", mes: "JUL", bolsa: "Operario Servicios Múltiples", numero: "38", fecha: "21/07/2026 10:00", estado: "Pendiente" },
    { dia: "22", mes: "JUL", bolsa: "Trabajador Social", numero: "12", fecha: "22/07/2026 09:30", estado: "Pendiente" },
  ],
  actividad: [
    { accion: "Llamamiento preparado", objeto: "Auxiliar Administrativo (n.º 44)", actor: "María Pérez", fecha: "17/07/2026 10:15" },
    { accion: "Contrato registrado", objeto: "Trabajador Social", actor: "Juan López", fecha: "17/07/2026 09:40" },
    { accion: "Cese registrado", objeto: "Operario Servicios Múltiples", actor: "María Pérez", fecha: "17/07/2026 09:10" },
    { accion: "Reincorporación revisada", objeto: "Auxiliar de Enfermería", actor: "Ana Ruiz", fecha: "16/07/2026 16:30" },
  ],
  contratos: [
    { expediente: "CON-2026-184", bolsa: "Trabajador Social", acto: "Nombramiento interino", inicio: "01/07/2026", fin: "31/12/2026", estado: "Activo" },
    { expediente: "CON-2026-171", bolsa: "Auxiliar Administrativo", acto: "Sustitución", inicio: "14/06/2026", fin: "Pendiente de fin", estado: "Activo" },
    { expediente: "CES-2026-089", bolsa: "Operario Servicios Múltiples", acto: "Cese por reincorporación", inicio: "02/03/2026", fin: "15/07/2026", estado: "Cese registrado" },
    { expediente: "REI-2026-032", bolsa: "Auxiliar de Enfermería", acto: "Reincorporación a bolsa", inicio: "16/07/2026", fin: "—", estado: "En revisión" },
  ],
  reglas: [
    { nombre: "Orden de llamamiento", ambito: "Todas las bolsas", version: "v4", vigencia: "Desde 01/07/2026", estado: "Publicada" },
    { nombre: "Renuncias justificadas", ambito: "Reglamento provincial", version: "v2", vigencia: "Desde 15/06/2026", estado: "Publicada" },
    { nombre: "Intentos y plazos", ambito: "Auxiliar Administrativo", version: "v3", vigencia: "Propuesta 20/07/2026", estado: "En validación" },
  ],
  documentos: [
    { referencia: "DOC-PL-001", plantilla: "Plantilla de bases", formatos: "DOCX, ODT, PDF", version: "v4", estado: "Publicada" },
    { referencia: "DOC-LL-003", plantilla: "Resolución de llamamiento", formatos: "PDF firmado", version: "v2", estado: "En revisión" },
    { referencia: "DOC-LI-005", plantilla: "Listado de integrantes", formatos: "PDF, CSV, JSON", version: "v5", estado: "Publicada" },
    { referencia: "DOC-CO-002", plantilla: "Comunicación individual", formatos: "HTML, TXT, PDF", version: "v3", estado: "Publicada" },
  ],
  canales: [
    { canal: "Correo electrónico", uso: "Aviso y contenido", integracion: "Configuración pendiente", estado: "No conectado" },
    { canal: "Telegram", uso: "Aviso personal con consentimiento", integracion: "Conector previsto", estado: "No conectado" },
    { canal: "Aviso interno", uso: "Bandeja del aspirante", integracion: "Lectura parcial en presentación", estado: "Parcial" },
    { canal: "Notificación fehaciente", uso: "Efectos administrativos", integracion: "Proveedor por seleccionar", estado: "Bloqueante" },
  ],
  auditoria: { expediente: "LLA-2026-0045" },
};

export function obtenerDatosPresentacion() {
  return structuredClone(DATOS);
}

const EVALUACIONES_PRESENTACION = Object.freeze({
  "NEC-2026-0045": [
    { secuencia: 1, resultado: "Elegible", puntuacion: 92.45, regla: "R1 · Orden de bolsa vigente", fundamento: "Primer puesto disponible del orden constituido" },
    { secuencia: 2, resultado: "Elegible", puntuacion: 88.3, regla: "R1 · Orden de bolsa vigente", fundamento: "Disponible sin causa de exclusión" },
    { secuencia: 3, resultado: "Elegible", puntuacion: 85.12, regla: "R1 · Orden de bolsa vigente", fundamento: "Disponible sin causa de exclusión" },
    { secuencia: 4, resultado: "No disponible", puntuacion: 84.61, regla: "R4 · Causas de indisponibilidad", fundamento: "Contrato temporal vigente comunicado" },
    { secuencia: 5, resultado: "Elegible", puntuacion: 82.9, regla: "R1 · Orden de bolsa vigente", fundamento: "Disponible sin causa de exclusión" },
  ],
  "NEC-2026-0038": [
    { secuencia: 1, resultado: "Elegible", puntuacion: 90.1, regla: "R1 · Orden de bolsa vigente", fundamento: "Primer puesto disponible del orden constituido" },
    { secuencia: 2, resultado: "Excluida por regla", puntuacion: 87.4, regla: "R6 · Penalización por renuncia", fundamento: "Renuncia no justificada dentro del plazo reglamentario" },
    { secuencia: 3, resultado: "Elegible", puntuacion: 83.75, regla: "R1 · Orden de bolsa vigente", fundamento: "Disponible sin causa de exclusión" },
  ],
  "NEC-2026-0012": [
    { secuencia: 1, resultado: "Elegible", puntuacion: 89.6, regla: "R1 · Orden de bolsa vigente", fundamento: "Primer puesto disponible del orden constituido" },
    { secuencia: 2, resultado: "Elegible", puntuacion: 86.05, regla: "R1 · Orden de bolsa vigente", fundamento: "Disponible sin causa de exclusión" },
  ],
});

export function obtenerPropuestaPresentacion(necesidadId) {
  const evaluaciones = EVALUACIONES_PRESENTACION[necesidadId] || EVALUACIONES_PRESENTACION["NEC-2026-0045"];
  return {
    esquema: "vec.bolsa.propuesta-llamamiento.presentacion.v1",
    demostracion: true,
    id: `PRO-${String(necesidadId || "NEC-2026-0045").slice(-4)}`,
    necesidad_id: String(necesidadId || "NEC-2026-0045"),
    estado: "Propuesta sintética aislada",
    version_bolsa: "v3 · huella registrada",
    version_regla: "v3 · 8 criterios",
    fecha_corte: "17/07/2026 08:00",
    personas_incluidas: evaluaciones.filter((item) => item.resultado === "Elegible").length,
    evaluaciones: evaluaciones.map((item) => ({ ...item })),
  };
}
