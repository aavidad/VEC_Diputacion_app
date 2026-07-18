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
  sesion: {
    actor_ref: "DEMO-PERFIL-ADMIN-FUNCIONAL-BOLSA-01",
    iniciales: "AD",
    nombre: "Administrador DEMO 01",
    perfil: "Administrador funcional de Bolsa · ámbito DEMO completo",
    vistas_permitidas: ["*"],
    operaciones_permitidas: ["*"],
  },
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
    { texto: "Informe jurídico pendiente en DEMO-BOL-014." },
    { texto: "Tres llamamientos previstos en siete días." },
    { texto: "Dos circuitos de firma por configurar." },
  ],
  configuracion_llamamiento: {
    regla: "Reglamento y bases · versión de presentación",
    apertura: "2026-07-20T09:00",
    apertura_visible: "20/07/2026 09:00",
    plazo_respuesta: "24 horas",
    tipo_cobertura: "Sustitución temporal",
    destino: "Centro DEMO 01",
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
    { id: "DEMO-BOL-AUXILIAR-ADMIN", nombre: "Auxiliar Administrativo", categoria: "Administración", creada: "15/01/2026", integrantes: 156, disponibles: 12, llamamiento: 4, cobertura: 96, estado: "Activa", regla: "Reglamento y bases · versión de presentación" },
    { id: "DEMO-BOL-ADMIN", nombre: "Administrativo", categoria: "Administración", creada: "10/02/2026", integrantes: 98, disponibles: 8, llamamiento: 2, cobertura: 94, estado: "Activa", regla: "Bases · versión de presentación" },
    { id: "DEMO-BOL-TRABAJO-SOCIAL", nombre: "Trabajador Social", categoria: "Servicios Sociales", creada: "20/03/2026", integrantes: 48, disponibles: 3, llamamiento: 3, cobertura: 92, estado: "Activa", regla: "Bases · versión de presentación" },
    { id: "DEMO-BOL-AUXILIAR-ENFERMERIA", nombre: "Auxiliar de Enfermería", categoria: "Sanidad", creada: "05/04/2026", integrantes: 312, disponibles: 25, llamamiento: 7, cobertura: 98, estado: "Activa", regla: "Bases · versión de presentación" },
    { id: "DEMO-BOL-OPERARIO-SERVICIOS", nombre: "Operario Servicios Múltiples", categoria: "Servicios Generales", creada: "12/01/2026", integrantes: 278, disponibles: 15, llamamiento: 1, cobertura: 94, estado: "Activa", regla: "Bases · versión de presentación" },
    { id: "DEMO-BOL-EDUCADOR-SOCIAL", nombre: "Educador Social", categoria: "Servicios Sociales", creada: "18/02/2026", integrantes: 66, disponibles: 4, llamamiento: 0, cobertura: 90, estado: "Activa", regla: "Bases · versión de presentación" },
  ],
  capacidades: { solicitar_propuesta_llamamiento: false, confirmar_llamamiento: false },
  necesidades_llamamiento: [
    { id: "DEMO-NEC-0045", referencia: "DEMO-NECESIDAD-0045", puesto: "Auxiliar Administrativo", bolsa_id: "DEMO-BOL-AUXILIAR-ADMIN", bolsa: "Auxiliar Administrativo", destino: "Centro DEMO 01", jornada: "Completa", duracion: "3 meses", cobertura: "Sustitución temporal", fecha_limite: "20/07/2026 09:00", regla: "Reglamento y bases · versión de presentación", estado: "Pendiente de propuesta" },
    { id: "DEMO-NEC-0038", referencia: "DEMO-NECESIDAD-0038", puesto: "Operario Servicios Múltiples", bolsa_id: "DEMO-BOL-OPERARIO-SERVICIOS", bolsa: "Operario Servicios Múltiples", destino: "Centro DEMO 02", jornada: "Completa", duracion: "6 meses", cobertura: "Vacante", fecha_limite: "21/07/2026 10:00", regla: "Bases · versión de presentación", estado: "Pendiente de propuesta" },
    { id: "DEMO-NEC-0012", referencia: "DEMO-NECESIDAD-0012", puesto: "Trabajador Social", bolsa_id: "DEMO-BOL-TRABAJO-SOCIAL", bolsa: "Trabajador Social", destino: "Centro DEMO 03", jornada: "Parcial 50 %", duracion: "3 meses", cobertura: "Programa temporal", fecha_limite: "22/07/2026 09:30", regla: "Bases · versión de presentación", estado: "Pendiente de propuesta" },
  ],
  elaboraciones: [
    { id: "DEMO-BOL-014", nombre: "Auxiliar Administrativo", expediente: "DEMO-EXP-014", fase: "Validación jurídica", reglas: "v3 · 8 criterios", plazo: "22/07/2026", responsable: "Unidad DEMO de Selección", estado: "En revisión", version_bases: "v3 · Huella SHA-256 sintética", calendario: "Solicitud 01/08–20/08/2026", firmantes: "Perfil DEMO Jefatura · Secretaría · Delegación" },
    { id: "DEMO-BOL-021", nombre: "Trabajador Social", expediente: "DEMO-EXP-021", fase: "Configuración de baremo", reglas: "Borrador · 6 criterios", plazo: "29/07/2026", responsable: "Unidad DEMO de Selección", estado: "Borrador", version_bases: "v1 · Borrador", calendario: "Pendiente de aprobación", firmantes: "Pendientes de configurar" },
    { id: "DEMO-BOL-009", nombre: "Operario Servicios Múltiples", expediente: "DEMO-EXP-009", fase: "Publicación", reglas: "v2 · 5 criterios", plazo: "Publicado 11/07/2026", responsable: "Perfil DEMO Jefatura", estado: "Publicada", version_bases: "v2 · Huella SHA-256 sintética", calendario: "Plazo cerrado 10/07/2026", firmantes: "Circuito demostrativo completado" },
  ],
  proximos: [
    { dia: "20", mes: "JUL", bolsa: "Auxiliar Administrativo", numero: "45", fecha: "20/07/2026 09:00", estado: "Pendiente" },
    { dia: "21", mes: "JUL", bolsa: "Operario Servicios Múltiples", numero: "38", fecha: "21/07/2026 10:00", estado: "Pendiente" },
    { dia: "22", mes: "JUL", bolsa: "Trabajador Social", numero: "12", fecha: "22/07/2026 09:30", estado: "Pendiente" },
  ],
  actividad: [
    { accion: "Llamamiento preparado", objeto: "DEMO-NEC-0044", actor: "DEMO-PERFIL-TECNICO-01", fecha: "17/07/2026 10:15", recibo: "DEMO-REC-BASE-004" },
    { accion: "Contrato registrado", objeto: "DEMO-CON-184", actor: "DEMO-PERFIL-TECNICO-02", fecha: "17/07/2026 09:40", recibo: "DEMO-REC-BASE-003" },
    { accion: "Cese registrado", objeto: "DEMO-CES-089", actor: "DEMO-PERFIL-TECNICO-01", fecha: "17/07/2026 09:10", recibo: "DEMO-REC-BASE-002" },
    { accion: "Reincorporación revisada", objeto: "DEMO-REI-032", actor: "DEMO-PERFIL-TECNICO-03", fecha: "16/07/2026 16:30", recibo: "DEMO-REC-BASE-001" },
  ],
  contratos: [
    { expediente: "DEMO-CON-184", bolsa: "Trabajador Social", acto: "Nombramiento interino", inicio: "01/07/2026", fin: "31/12/2026", estado: "Activo" },
    { expediente: "DEMO-CON-171", bolsa: "Auxiliar Administrativo", acto: "Sustitución", inicio: "14/06/2026", fin: "Pendiente de fin", estado: "Activo" },
    { expediente: "DEMO-CES-089", bolsa: "Operario Servicios Múltiples", acto: "Cese por reincorporación", inicio: "02/03/2026", fin: "15/07/2026", estado: "Cese registrado" },
    { expediente: "DEMO-REI-032", bolsa: "Auxiliar de Enfermería", acto: "Reincorporación a bolsa", inicio: "16/07/2026", fin: "—", estado: "En revisión" },
  ],
  reglas: [
    { nombre: "Orden de llamamiento", ambito: "Todas las bolsas", version: "v4", vigencia: "Desde 01/07/2026", estado: "Publicada" },
    { nombre: "Renuncias justificadas", ambito: "Reglamento provincial", version: "v2", vigencia: "Desde 15/06/2026", estado: "Publicada" },
    { nombre: "Intentos y plazos", ambito: "Auxiliar Administrativo", version: "v3", vigencia: "Propuesta 20/07/2026", estado: "En validación" },
  ],
  documentos: [
    { referencia: "DEMO-DOC-001", plantilla: "Plantilla de bases", formatos: "DOCX, ODT, PDF", version: "v4", estado: "Publicada" },
    { referencia: "DEMO-DOC-003", plantilla: "Resolución de llamamiento", formatos: "PDF firmado", version: "v2", estado: "En revisión" },
    { referencia: "DEMO-DOC-005", plantilla: "Listado de integrantes", formatos: "PDF, CSV, JSON", version: "v5", estado: "Publicada" },
    { referencia: "DEMO-DOC-002", plantilla: "Comunicación individual", formatos: "HTML, TXT, PDF", version: "v3", estado: "Publicada" },
  ],
  canales: [
    { canal: "Correo electrónico", uso: "Aviso y contenido", integracion: "Configuración pendiente", estado: "No conectado" },
    { canal: "Telegram", uso: "Aviso personal con consentimiento", integracion: "Conector previsto", estado: "No conectado" },
    { canal: "Aviso interno", uso: "Bandeja del aspirante", integracion: "Lectura parcial en presentación", estado: "Parcial" },
    { canal: "Notificación fehaciente", uso: "Efectos administrativos", integracion: "Proveedor por seleccionar", estado: "Bloqueante" },
  ],
  solicitudes: [
    { id: "DEMO-SOL-001", persona_ref: "DEMO-PER-001", convocatoria: "DEMO-BOL-014", registrada: "12/07/2026 09:14", requisitos: "8/8", subsanacion: "No requerida", estado: "Pendiente de revisión" },
    { id: "DEMO-SOL-002", persona_ref: "DEMO-PER-002", convocatoria: "DEMO-BOL-014", registrada: "12/07/2026 09:31", requisitos: "7/8", subsanacion: "Hasta 23/07/2026", estado: "Pendiente de subsanación" },
    { id: "DEMO-SOL-003", persona_ref: "DEMO-PER-003", convocatoria: "DEMO-BOL-014", registrada: "12/07/2026 10:02", requisitos: "8/8", subsanacion: "Recibida", estado: "Subsanada" },
    { id: "DEMO-SOL-004", persona_ref: "DEMO-PER-004", convocatoria: "DEMO-BOL-021", registrada: "13/07/2026 11:22", requisitos: "6/6", subsanacion: "No requerida", estado: "Admitida provisional" },
  ],
  meritos_revision: [
    { id: "DEMO-MER-001", persona_ref: "DEMO-PER-001", tipo: "Experiencia profesional", evidencia: "DEMO-DOC-MER-001", declarado: "36 meses · jornada completa", puntos: "3,60", estado: "Pendiente" },
    { id: "DEMO-MER-002", persona_ref: "DEMO-PER-002", tipo: "Experiencia profesional", evidencia: "DEMO-DOC-MER-002", declarado: "18 meses · jornada 50 %", puntos: "0,90", estado: "Aceptado" },
    { id: "DEMO-MER-003", persona_ref: "DEMO-PER-003", tipo: "Formación", evidencia: "DEMO-DOC-MER-003", declarado: "Curso 120 horas", puntos: "0,60", estado: "Rechazado" },
    { id: "DEMO-MER-004", persona_ref: "DEMO-PER-004", tipo: "Titulación", evidencia: "DEMO-DOC-MER-004", declarado: "Titulación superior relacionada", puntos: "1,00", estado: "Pendiente" },
  ],
  criterios_baremo: [
    { id: "DEMO-CRI-001", bloque: "Experiencia", criterio: "Mes trabajado en administración convocante", formula: "0,10 puntos × mes × fracción de jornada", maximo: "6,00", version: "v3", estado: "Validado" },
    { id: "DEMO-CRI-002", bloque: "Experiencia", criterio: "Mes trabajado en otra administración", formula: "0,05 puntos × mes × fracción de jornada", maximo: "3,00", version: "v3", estado: "Validado" },
    { id: "DEMO-CRI-003", bloque: "Formación", criterio: "Curso relacionado y acreditado", formula: "0,005 puntos × hora", maximo: "2,00", version: "v3", estado: "Validado" },
    { id: "DEMO-CRI-004", bloque: "Titulación", criterio: "Titulación superior de la misma rama", formula: "1,00 punto", maximo: "1,00", version: "v3", estado: "Validado" },
  ],
  ranking: [
    { id: "DEMO-RAN-001", posicion: 1, persona_ref: "DEMO-PER-004", experiencia: "5,40", formacion: "1,50", otros: "1,00", total: "7,90", desempate: "No aplicado", estado: "Provisional" },
    { id: "DEMO-RAN-002", posicion: 2, persona_ref: "DEMO-PER-001", experiencia: "3,60", formacion: "1,80", otros: "1,00", total: "6,40", desempate: "No aplicado", estado: "Provisional" },
    { id: "DEMO-RAN-003", posicion: 3, persona_ref: "DEMO-PER-002", experiencia: "2,90", formacion: "2,00", otros: "0,50", total: "5,40", desempate: "Fecha de solicitud", estado: "Provisional" },
  ],
  alegaciones: [
    { id: "DEMO-ALE-001", persona_ref: "DEMO-PER-003", objeto: "DEMO-MER-003 · formación", registrada: "16/07/2026 08:42", plazo: "Dentro de plazo", evidencia: "DEMO-DOC-ALE-001", estado: "Pendiente" },
    { id: "DEMO-ALE-002", persona_ref: "DEMO-PER-002", objeto: "DEMO-RAN-003 · desempate", registrada: "16/07/2026 09:05", plazo: "Dentro de plazo", evidencia: "DEMO-DOC-ALE-002", estado: "En estudio" },
  ],
  importaciones: [
    { id: "DEMO-IMP-001", origen: "Convoca · XLS sintético", lote: "Lote DEMO de 12 filas", huella: "SHA-256 DEMO…A19F", validas: 10, incidencias: 2, autoridad: "No autoritativa", estado: "Pendiente de validación" },
    { id: "DEMO-IMP-002", origen: "Convoca · XLS sintético", lote: "Lote DEMO de 8 filas", huella: "SHA-256 DEMO…72C4", validas: 8, incidencias: 0, autoridad: "No autoritativa", estado: "Validada" },
  ],
  llamamientos_demo: [
    { id: "DEMO-LLA-045", necesidad: "DEMO-NEC-0045", bolsa: "Auxiliar Administrativo", orden: "Prelación v3", incluidos: 1, plazo: "20/07/2026 09:00", canal: "Sin envío real", estado: "Pendiente de propuesta" },
    { id: "DEMO-LLA-038", necesidad: "DEMO-NEC-0038", bolsa: "Operario Servicios Múltiples", orden: "Prelación v2", incluidos: 1, plazo: "21/07/2026 10:00", canal: "Sin envío real", estado: "Preparado" },
  ],
  comunicaciones_demo: [
    { id: "DEMO-COM-001", expediente: "DEMO-LLA-045", plantilla: "Aviso de llamamiento v3", canal: "Correo + aviso interno", destinatario: "DEMO-PER-001", acuse: "No generado", estado: "Borrador" },
    { id: "DEMO-COM-002", expediente: "DEMO-ALE-001", plantilla: "Resolución de alegación v2", canal: "Notificación fehaciente", destinatario: "DEMO-PER-003", acuse: "No generado", estado: "Preparada" },
  ],
  auditoria_eventos: [
    { referencia: "DEMO-REC-BASE-004", actor: "DEMO-PERFIL-TECNICO-01", instante: "2026-07-17T08:15:00Z", operacion: "emitir-llamamiento", objetivo: "DEMO-LLA-044", resultado: "Preparado", efectos_reales: false },
    { referencia: "DEMO-REC-BASE-003", actor: "DEMO-PERFIL-TECNICO-02", instante: "2026-07-17T07:40:00Z", operacion: "registrar-contrato", objetivo: "DEMO-CON-184", resultado: "Activo", efectos_reales: false },
  ],
  roles_demo: [
    { id: "DEMO-ROL-ADMIN-BOLSA", nombre: "Administración de Bolsa", ambito: "Convocatoria asignada", permisos: "Configurar, revisar, proponer", segregacion: "No firma publicación", estado: "Activo" },
    { id: "DEMO-ROL-TECNICO", nombre: "Técnico revisor", ambito: "Expedientes asignados", permisos: "Revisar méritos y alegaciones", segregacion: "No publica listas", estado: "Activo" },
    { id: "DEMO-ROL-AUDITOR", nombre: "Auditor de consulta", ambito: "Finalidad autorizada", permisos: "Solo lectura trazada", segregacion: "Sin mutaciones", estado: "Activo" },
  ],
  configuraciones_demo: [
    { id: "DEMO-CFG-CALENDARIO", parametro: "Calendario de aportación documental", valor: "01/08/2026–20/08/2026", version: "v2", estado: "Publicada" },
    { id: "DEMO-CFG-DESEMPATE", parametro: "Orden de desempate", valor: "Bases → mayor experiencia → solicitud", version: "v3", estado: "Publicada" },
    { id: "DEMO-CFG-RETENCION", parametro: "Retención documental", valor: "Pendiente de tabla aprobada", version: "v1", estado: "Bloqueante" },
  ],
  auditoria: { expediente: "DEMO-LLA-0045" },
};

const PERFIL_TECNICO = Object.freeze({
  actor_ref: "DEMO-PERFIL-TECNICO-RRHH-01",
  iniciales: "TR",
  nombre: "Técnico DEMO 01",
  perfil: "Técnico revisor de RRHH · ámbito DEMO restringido",
  vistas_permitidas: ["portal", "resumen", "solicitudes", "meritos", "baremacion", "alegaciones", "documentos", "auditoria"],
  operaciones_permitidas: [
    "admitir-solicitud", "excluir-solicitud", "registrar-subsanacion",
    "aceptar-merito", "rechazar-merito", "revocar-merito", "rehabilitar-merito",
    "calcular-baremo", "resolver-alegacion", "desestimar-alegacion",
    "generar-documento", "exportar-informe",
  ],
});

export function obtenerDatosPresentacion(perfil = "administrador") {
  if (perfil !== "administrador" && perfil !== "tecnico") {
    throw new Error("perfil de presentación no permitido");
  }
  const datos = structuredClone(DATOS);
  if (perfil === "tecnico") datos.sesion = structuredClone(PERFIL_TECNICO);
  return datos;
}

const EVALUACIONES_PRESENTACION = Object.freeze({
  "DEMO-NEC-0045": [
    { orden: "1", resultado: "no_elegible", motivos: [{ regla: "R4 · Indisponibilidad", fundamento: "Contrato temporal sintético vigente" }] },
    { orden: "2", resultado: "no_elegible", motivos: [{ regla: "R6 · Renuncia", fundamento: "Supuesto sintético de renuncia dentro del periodo configurado" }] },
    { orden: "3", resultado: "elegible", motivos: [{ regla: "R1 · Orden vigente", fundamento: "Primera posición sintética sin causa de exclusión" }] },
  ],
  "DEMO-NEC-0038": [
    { orden: "1", resultado: "no_elegible", motivos: [{ regla: "R4 · Indisponibilidad", fundamento: "Situación sintética incompatible con la necesidad" }] },
    { orden: "2", resultado: "elegible", motivos: [{ regla: "R1 · Orden vigente", fundamento: "Primera posición sintética elegible" }] },
  ],
  "DEMO-NEC-0012": [
    { orden: "1", resultado: "elegible", motivos: [{ regla: "R1 · Orden vigente", fundamento: "Primera posición sintética elegible" }] },
  ],
});

export function obtenerPropuestaPresentacion(necesidadId) {
  const evaluaciones = EVALUACIONES_PRESENTACION[necesidadId] || EVALUACIONES_PRESENTACION["DEMO-NEC-0045"];
  return {
    esquema: "vec.bolsa.propuesta-llamamiento.presentacion.v1",
    demostracion: true,
    id: `DEMO-PRO-${String(necesidadId || "DEMO-NEC-0045").slice(-4)}`,
    necesidad_id: String(necesidadId || "DEMO-NEC-0045"),
    estado: "demostracion",
    version_bolsa: "v3 · versión sintética",
    version_regla: "v3 · regla sintética",
    fecha_corte: "2026-07-17T08:00:00Z",
    personas_incluidas: String(evaluaciones.filter((item) => item.resultado === "elegible").length),
    evaluaciones: structuredClone(evaluaciones),
  };
}
