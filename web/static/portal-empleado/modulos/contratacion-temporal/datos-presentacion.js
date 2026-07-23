/**
 * Juego sintético exclusivo de la presentación RRHH.
 *
 * No contiene personas, expedientes, documentos ni actos reales. Este archivo
 * no se importa desde la composición interna o productiva.
 */

import { CAPACIDADES_CONTRATACION_TEMPORAL as CAP } from "./contrato-expedientes.js";

const VACIAS = Object.freeze([]);

function opcion(clave, etiqueta) {
  return { clave, etiqueta };
}

function campo(clave, etiqueta, valor, {
  tono = "neutro",
  control = "solo_lectura",
  obligatorio = false,
  opciones = VACIAS,
} = {}) {
  return { clave, etiqueta, valor, tono, control, obligatorio, opciones };
}

function columna(clave, etiqueta) {
  return { clave, etiqueta };
}

function fila(filaRef, celdas) {
  return { fila_ref: filaRef, celdas };
}

function panel(panelRef, tipo, titulo, descripcion, {
  campos = VACIAS,
  columnas = VACIAS,
  filas = VACIAS,
} = {}) {
  return {
    panel_ref: panelRef,
    tipo,
    titulo,
    descripcion,
    campos,
    columnas,
    filas,
  };
}

function accion(accionRef, etiqueta, {
  variante = "primaria",
  capacidad,
  confirmacion = "",
  disponible = false,
  motivoNoDisponible = "La tarea ya está completada o todavía no ha alcanzado su fase.",
} = {}) {
  return {
    accion_ref: accionRef,
    etiqueta,
    tipo: "efecto",
    variante,
    capacidad,
    confirmacion,
    destino_tarea_ref: "",
    disponible,
    motivo_no_disponible: disponible ? "" : motivoNoDisponible,
  };
}

function tarea({
  referencia,
  orden,
  fase,
  etiqueta,
  descripcion,
  estadoClave,
  estado,
  responsable,
  unidad = "Unidad DEMO de RRHH",
  entrada,
  salida = "",
  tiempo,
  reciboRef = "",
  decisionRef = "",
  paneles,
  acciones,
}) {
  return {
    tarea_ref: referencia,
    orden,
    fase_ref: fase,
    etiqueta,
    descripcion,
    estado_clave: estadoClave,
    estado,
    unidad,
    responsable,
    entrada,
    salida,
    tiempo,
    recibo_ref: reciboRef || (estadoClave === "completado"
      ? `rec-demo-tarea-${String(orden).padStart(3, "0")}` : ""),
    decision_ref: decisionRef || (estadoClave === "completado"
      ? `dec-demo-tarea-${String(orden).padStart(3, "0")}` : ""),
    paneles,
    acciones,
  };
}

const FASES = [
  { fase_ref: "fase-solicitud", orden: 1, etiqueta: "Solicitud", estado_clave: "completado" },
  { fase_ref: "fase-analisis", orden: 2, etiqueta: "Análisis RRHH", estado_clave: "completado" },
  { fase_ref: "fase-bolsa", orden: 3, etiqueta: "Gestión de bolsa", estado_clave: "completado" },
  { fase_ref: "fase-fiscalizacion", orden: 4, etiqueta: "Fiscalización", estado_clave: "completado" },
  { fase_ref: "fase-candidato", orden: 5, etiqueta: "Obtención de candidato", estado_clave: "completado" },
  { fase_ref: "fase-nombramiento", orden: 6, etiqueta: "Nombramiento", estado_clave: "en_curso" },
  { fase_ref: "fase-incorporacion", orden: 7, etiqueta: "Incorporación", estado_clave: "pendiente" },
  { fase_ref: "fase-seguimiento", orden: 8, etiqueta: "Seguimiento", estado_clave: "pendiente" },
];

const CABECERA = [
  campo("centro", "Servicio solicitante", "Centro DEMO de Servicios Sociales"),
  campo("categoria", "Categoría", "Trabajador/a social"),
  campo("modalidad", "Modalidad", "Sustitución"),
  campo("procedimiento", "Procedimiento", "Bolsa vigente"),
  campo("periodo", "Periodo previsto", "15/08/2026 — 14/04/2027"),
  campo("coste", "Coste estimado", "31.480,25 €"),
  campo("responsable", "Unidad responsable", "Unidad DEMO de Gestión Social"),
  campo("estado", "Estado actual", "Generación documental", { tono: "informacion" }),
];

const TAREAS = [
  tarea({
    referencia: "tarea-solicitud", orden: 1, fase: "fase-solicitud",
    etiqueta: "Solicitud del centro",
    descripcion: "Petición inicial, retención de crédito y documentación complementaria.",
    estadoClave: "completado", estado: "Registrada", responsable: "Centro solicitante",
    entrada: "10/07/2026 09:15", salida: "10/07/2026 09:18", tiempo: "3 min",
    paneles: [
      panel("panel-solicitud-datos", "datos", "Datos de la petición", "Información registrada por el centro.", {
        campos: [
          campo("peticion", "Petición", "Cobertura temporal durante una ausencia prevista."),
          campo("contacto", "Persona de contacto", "Referencia interna DEMO"),
          campo("inicio", "Fecha prevista de inicio", "15/08/2026"),
          campo("fin", "Fecha prevista de fin", "14/04/2027"),
        ],
      }),
      panel("panel-solicitud-rc", "comprobaciones", "RC y documentos", "Evidencias aportadas al expediente.", {
        campos: [
          campo("rc", "Retención de crédito", "Declarada", { tono: "exito" }),
          campo("numero_rc", "Número de RC", "RC-DEMO-2026-0024"),
          campo("importe_rc", "Importe", "32.450,00 €"),
          campo("adjuntos", "Documentación", "Petición del centro y RC incorporadas", { tono: "exito" }),
        ],
      }),
    ],
    acciones: [accion("reenviar_analisis", "Reenviar a análisis RRHH", {
      variante: "secundaria",
      capacidad: CAP.enviarAnalisis,
      confirmacion: "Se registrará un nuevo envío a la bandeja de análisis de RRHH.",
    })],
  }),
  tarea({
    referencia: "tarea-analisis", orden: 2, fase: "fase-analisis",
    etiqueta: "Análisis de RRHH",
    descripcion: "Comprobación de modalidad, categoría, duración y retención de crédito.",
    estadoClave: "completado", estado: "Validado", responsable: "Dirección de RRHH",
    entrada: "10/07/2026 09:45", salida: "10/07/2026 10:00", tiempo: "15 min",
    paneles: [
      panel("panel-analisis-formulario", "formulario", "Comprobación y validación", "Los catálogos se inyectan desde composición.", {
        campos: [
          campo("modalidad", "Modalidad", "sustitucion", {
            control: "seleccion", obligatorio: true,
            opciones: [opcion("sustitucion", "Sustitución"), opcion("vacante", "Vacante"), opcion("programa", "Programa")],
          }),
          campo("motivo", "Motivo", "Sustitución durante una ausencia prevista.", { control: "texto", obligatorio: true }),
          campo("duracion", "Duración prevista", "8 meses", { control: "texto", obligatorio: true }),
          campo("categoria", "Categoría", "Trabajador/a social"),
          campo("grupo", "Grupo o subgrupo", "A2"),
          campo("observaciones", "Observaciones", "RC suficiente para el periodo previsto.", { control: "area" }),
        ],
      }),
      panel("panel-analisis-rc", "comprobaciones", "Retención de crédito", "Resultado de la fuente presupuestaria.", {
        campos: [
          campo("estado_rc", "Comprobación de RC", "Válida", { tono: "exito" }),
          campo("fuente_rc", "Fuente", "Conector presupuestario DEMO"),
          campo("recibo_rc", "Recibo", "rec-demo-rc-000024"),
          campo("coste", "Coste previsto", "31.480,25 €"),
        ],
      }),
    ],
    acciones: [accion("validar_analisis", "Validar y continuar", {
      capacidad: CAP.analizar,
      confirmacion: "Se registrará la validación del análisis y su recibo.",
    })],
  }),
  tarea({
    referencia: "tarea-cobertura", orden: 3, fase: "fase-bolsa",
    etiqueta: "Comprobaciones y vía de cobertura",
    descripcion: "Consulta de Bolsa, SAE y convocatorias mediante puertos independientes.",
    estadoClave: "completado", estado: "Bolsa vigente", responsable: "Responsable de Gestión",
    entrada: "10/07/2026 10:15", salida: "10/07/2026 10:28", tiempo: "13 min",
    paneles: [
      panel("panel-cobertura-comprobaciones", "comprobaciones", "Comprobaciones automáticas", "Cada resultado conserva fuente, fecha y recibo.", {
        campos: [
          campo("bolsa_vigente", "Existe bolsa vigente", "Sí · Bolsa DEMO de Trabajo Social", { tono: "exito" }),
          campo("bolsa_agotada", "Bolsa agotada", "No · 18 candidaturas disponibles", { tono: "exito" }),
          campo("sae", "Requiere oferta al SAE", "No procede", { tono: "exito" }),
          campo("convocatoria", "Requiere nueva convocatoria", "No procede", { tono: "exito" }),
        ],
      }),
      panel("panel-cobertura-propuesta", "formulario", "Procedimiento a seguir", "La decisión positiva siempre requiere motivación.", {
        campos: [
          campo("via", "Vía de cobertura", "bolsa_vigente", {
            control: "radio", obligatorio: true,
            opciones: [opcion("bolsa_vigente", "Bolsa vigente"), opcion("oferta_sae", "Oferta SAE"), opcion("nueva_bolsa", "Nueva convocatoria de bolsa")],
          }),
          campo("motivacion", "Motivación", "Existe bolsa vigente y con candidaturas disponibles.", { control: "area", obligatorio: true }),
        ],
      }),
    ],
    acciones: [accion("decidir_cobertura", "Confirmar vía de cobertura", {
      capacidad: CAP.decidirCobertura,
      confirmacion: "Se registrará la vía elegida, sus fuentes y la motivación.",
    })],
  }),
  tarea({
    referencia: "tarea-asignacion", orden: 4, fase: "fase-bolsa",
    etiqueta: "Asignación a unidad",
    descripcion: "Entrada en la bandeja de la unidad gestora y notificación a su responsable.",
    estadoClave: "completado", estado: "Asignada", responsable: "Responsable de Gestión",
    entrada: "10/07/2026 10:28", salida: "10/07/2026 10:35", tiempo: "7 min",
    paneles: [
      panel("panel-asignacion", "formulario", "Unidad del departamento", "La asignación cambia la bandeja sin alterar la competencia.", {
        campos: [
          campo("unidad", "Unidad que tramitará el expediente", "unidad_gestion_social", {
            control: "seleccion", obligatorio: true,
            opciones: [opcion("unidad_gestion_social", "Unidad DEMO de Gestión Social"), opcion("unidad_seleccion", "Unidad DEMO de Selección"), opcion("unidad_personal", "Unidad DEMO de Personal")],
          }),
          campo("responsable", "Responsable asignado", "Responsable DEMO de Unidad"),
          campo("notificacion", "Notificación interna", "Preparada", { tono: "exito" }),
        ],
      }),
    ],
    acciones: [accion("asignar_unidad", "Asignar y continuar", {
      capacidad: CAP.asignarUnidad,
      confirmacion: "El expediente saldrá de esta bandeja y se notificará a la unidad seleccionada.",
    })],
  }),
  tarea({
    referencia: "tarea-informe-juridico", orden: 5, fase: "fase-bolsa",
    etiqueta: "Informe jurídico",
    descripcion: "Generación documental con datos del expediente y plantilla gobernada.",
    estadoClave: "completado", estado: "Informe generado", responsable: "Unidad DEMO de Gestión Social",
    entrada: "10/07/2026 11:00", salida: "10/07/2026 11:08", tiempo: "8 min",
    paneles: [
      panel("panel-informe-resumen", "datos", "Datos del informe", "Proyección minimizada del expediente.", { campos: CABECERA.slice(0, 6) }),
      panel("panel-informe-documento", "documentos", "Generación de informe jurídico", "Plantilla, versión y evidencia de generación.", {
        campos: [
          campo("plantilla", "Plantilla", "Informe jurídico · versión DEMO 3"),
          campo("documento", "Documento", "Informe jurídico DEMO.pdf", { tono: "exito" }),
          campo("huella", "Huella", "Referencia de verificación DEMO"),
        ],
      }),
    ],
    acciones: [accion("generar_informe_juridico", "Generar informe jurídico", {
      capacidad: CAP.prepararInforme,
      confirmacion: "Se generará una nueva versión del informe mediante el conector documental.",
    })],
  }),
  tarea({
    referencia: "tarea-envio-intervencion", orden: 6, fase: "fase-fiscalizacion",
    etiqueta: "Firma y envío a Intervención",
    descripcion: "Revisión, firma de jefatura y traslado a la bandeja de fiscalización.",
    estadoClave: "completado", estado: "Enviado", responsable: "Jefatura de Servicio",
    entrada: "10/07/2026 11:08", salida: "11/07/2026 12:15", tiempo: "1 d 1 h",
    paneles: [
      panel("panel-envio-firma", "documentos", "Documento a firmar", "La firma y el envío generan recibos distintos.", {
        campos: [
          campo("documento", "Informe jurídico", "Firmado", { tono: "exito" }),
          campo("firmante", "Firmante", "Jefatura de Servicio DEMO"),
          campo("firma", "Estado de firma", "Firma electrónica DEMO completada", { tono: "exito" }),
          campo("envio", "Traslado", "Intervención · 11/07/2026 12:15", { tono: "exito" }),
        ],
      }),
    ],
    acciones: [accion("enviar_intervencion", "Enviar a Intervención", {
      capacidad: CAP.solicitarFiscalizacion,
      confirmacion: "Se enviará el expediente y su índice documental a Intervención.",
    })],
  }),
  tarea({
    referencia: "tarea-fiscalizacion", orden: 7, fase: "fase-fiscalizacion",
    etiqueta: "Fiscalización",
    descripcion: "Resultado de Intervención, observaciones y documentos asociados.",
    estadoClave: "completado", estado: "Favorable", responsable: "Intervención",
    entrada: "11/07/2026 12:15", salida: "12/07/2026 09:30", tiempo: "21 h 15 min",
    paneles: [
      panel("panel-fiscalizacion", "formulario", "Resultado de fiscalización", "La respuesta queda segregada de la unidad proponente.", {
        campos: [
          campo("resultado", "Resultado", "favorable", {
            control: "radio", obligatorio: true,
            opciones: [opcion("favorable", "Favorable"), opcion("observaciones", "Favorable con observaciones"), opcion("reparo", "Desfavorable o reparo")],
          }),
          campo("observaciones", "Observaciones", "Sin observaciones.", { control: "area" }),
          campo("recibo", "Recibo de fiscalización", "rec-demo-int-001125", { tono: "exito" }),
        ],
      }),
    ],
    acciones: [accion("registrar_fiscalizacion", "Registrar resultado", {
      capacidad: CAP.registrarFiscalizacion,
      confirmacion: "Se registrará el resultado y, si procede, se devolverá para subsanación.",
    })],
  }),
  tarea({
    referencia: "tarea-subsanacion", orden: 8, fase: "fase-fiscalizacion",
    etiqueta: "Subsanación de reparos",
    descripcion: "Circuito de retorno a la unidad sin reescribir el histórico.",
    estadoClave: "completado", estado: "No requerida", responsable: "Unidad gestora",
    entrada: "12/07/2026 09:30", salida: "12/07/2026 09:31", tiempo: "1 min",
    paneles: [
      panel("panel-subsanacion", "aviso", "Subsanación", "El expediente de presentación fue fiscalizado favorablemente.", {
        campos: [
          campo("situacion", "Situación", "No se recibió reparo", { tono: "exito" }),
          campo("alternativa", "Si existiera reparo", "Volvería a la unidad con plazo, observaciones y documentos."),
          campo("historia", "Histórico", "La fiscalización original permanecería inmutable."),
        ],
      }),
    ],
    acciones: [accion("abrir_subsanacion", "Registrar subsanación excepcional", {
      variante: "secundaria",
      capacidad: CAP.registrarSubsanacion,
      confirmacion: "La presentación registrará una actuación sintética de subsanación.",
    })],
  }),
  tarea({
    referencia: "tarea-iniciar-llamamiento", orden: 9, fase: "fase-candidato",
    etiqueta: "Inicio del llamamiento",
    descripcion: "Preparación según bolsa, bases y Reglamento de Bolsas aplicables.",
    estadoClave: "completado", estado: "Iniciado", responsable: "Unidad de llamamientos",
    entrada: "12/07/2026 10:00", salida: "12/07/2026 10:05", tiempo: "5 min",
    paneles: [
      panel("panel-llamamiento-config", "datos", "Configuración del llamamiento", "Parámetros gobernados por la bolsa y sus bases.", {
        campos: [
          campo("bolsa", "Bolsa utilizada", "Bolsa DEMO de Trabajo Social"),
          campo("regla", "Norma de aplicación", "Reglamento y bases · versión DEMO"),
          campo("orden", "Orden", "Puntuación y disponibilidad"),
          campo("candidaturas", "Candidaturas disponibles", "18"),
        ],
      }),
    ],
    acciones: [accion("iniciar_llamamiento", "Iniciar llamamiento", {
      capacidad: CAP.prepararLlamamiento,
      confirmacion: "Se solicitará a Bolsa una propuesta de candidaturas según la regla vigente.",
    })],
  }),
  tarea({
    referencia: "tarea-seleccion-candidato", orden: 10, fase: "fase-candidato",
    etiqueta: "Selección de candidatura",
    descripcion: "Orden, disponibilidad, exclusiones y evidencia de la regla aplicada.",
    estadoClave: "completado", estado: "Candidatura seleccionada", responsable: "Unidad de llamamientos",
    entrada: "12/07/2026 10:05", salida: "12/07/2026 10:20", tiempo: "15 min",
    paneles: [
      panel("panel-candidaturas", "tabla", "Candidaturas propuestas", "Identificadores sintéticos minimizados; sin datos personales reales.", {
        columnas: [
          columna("orden", "Orden"), columna("referencia", "Referencia"),
          columna("estado", "Estado"), columna("situacion", "Situación"),
          columna("puntuacion", "Puntuación"),
        ],
        filas: [
          fila("fila-candidata-001", ["1", "CAND-DEMO-001", "Disponible", "Disponible", "92,450"]),
          fila("fila-candidata-002", ["2", "CAND-DEMO-002", "Ocupada", "Contrato hasta 30/08/2026", "88,300"]),
          fila("fila-candidata-003", ["3", "CAND-DEMO-003", "No disponible", "Causa catalogada", "85,120"]),
          fila("fila-candidata-004", ["4", "CAND-DEMO-004", "Excluida", "Pendiente de resolución", "84,610"]),
          fila("fila-candidata-005", ["5", "CAND-DEMO-005", "Renuncia pendiente", "Plazo abierto", "82,900"]),
        ],
      }),
    ],
    acciones: [accion("seleccionar_candidato", "Llamar a la primera candidatura elegible", {
      capacidad: CAP.seleccionarCandidatura,
      confirmacion: "Se preparará la comunicación a la candidatura que corresponda según el orden acreditado.",
    })],
  }),
  tarea({
    referencia: "tarea-resultado-llamamiento", orden: 11, fase: "fase-candidato",
    etiqueta: "Resultado del llamamiento",
    descripcion: "Aceptación, renuncia, falta de respuesta o incumplimiento de requisitos.",
    estadoClave: "completado", estado: "Aceptado", responsable: "Unidad de llamamientos",
    entrada: "12/07/2026 10:20", salida: "13/07/2026 09:00", tiempo: "22 h 40 min",
    paneles: [
      panel("panel-resultado-llamamiento", "formulario", "Resultado", "La respuesta y su evidencia quedan ligadas al llamamiento.", {
        campos: [
          campo("candidatura", "Candidatura", "CAND-DEMO-001"),
          campo("resultado", "Resultado", "acepta", {
            control: "radio", obligatorio: true,
            opciones: [opcion("acepta", "Acepta"), opcion("renuncia", "Renuncia voluntaria"), opcion("no_localizada", "No localizada"), opcion("rechaza", "Rechaza la oferta"), opcion("no_cumple", "No cumple requisitos")],
          }),
          campo("observaciones", "Observaciones", "Aceptación recibida dentro de plazo.", { control: "area" }),
          campo("acta", "Acta", "Acta de llamamiento DEMO.pdf", { tono: "exito" }),
        ],
      }),
    ],
    acciones: [accion("registrar_resultado_llamamiento", "Guardar resultado", {
      capacidad: CAP.registrarResultadoLlamamiento,
      confirmacion: "Se registrará el resultado y continuará el flujo que corresponda.",
    })],
  }),
  tarea({
    referencia: "tarea-traslado-intervencion", orden: 12, fase: "fase-nombramiento",
    etiqueta: "Traslado de candidatura",
    descripcion: "Preparación de documentación económica y de selección para Intervención.",
    estadoClave: "completado", estado: "Trasladado", responsable: "Unidad gestora",
    entrada: "13/07/2026 09:15", salida: "13/07/2026 09:40", tiempo: "25 min",
    paneles: [
      panel("panel-traslado-docs", "tabla", "Documentación a remitir", "Índice previo a formalización.", {
        columnas: [columna("documento", "Documento"), columna("descripcion", "Descripción"), columna("estado", "Estado")],
        filas: [
          fila("fila-doc-001", ["Acta de llamamiento", "Resultado del llamamiento", "Generado"]),
          fila("fila-doc-002", ["Aceptación", "Evidencia de aceptación", "Generado"]),
          fila("fila-doc-003", ["Informe de necesidad", "Justificación de modalidad", "Generado"]),
          fila("fila-doc-004", ["Informe de fiscalización", "Resultado favorable", "Firmado"]),
          fila("fila-doc-005", ["Certificado de bolsa", "Orden y pertenencia", "Generado"]),
          fila("fila-doc-006", ["Resumen económico", "Coste estimado", "Generado"]),
        ],
      }),
    ],
    acciones: [accion("trasladar_candidato", "Enviar a Intervención", {
      capacidad: CAP.prepararFormalizacion,
      confirmacion: "Se remitirá el índice documental para la formalización.",
    })],
  }),
  tarea({
    referencia: "tarea-informe-definitivo", orden: 13, fase: "fase-nombramiento",
    etiqueta: "Informe definitivo",
    descripcion: "Generación de la propuesta definitiva previa al nombramiento o contrato.",
    estadoClave: "completado", estado: "Generado", responsable: "Unidad gestora",
    entrada: "13/07/2026 10:00", salida: "13/07/2026 10:12", tiempo: "12 min",
    paneles: [
      panel("panel-informe-definitivo", "documentos", "Informe definitivo", "Vista previa y formatos producidos por el conector documental.", {
        campos: [
          campo("documento", "Documento", "Informe definitivo DEMO"),
          campo("version", "Versión", "2"),
          campo("formatos", "Formatos", "PDF accesible y ODT"),
          campo("estado", "Estado", "Generado", { tono: "exito" }),
        ],
      }),
    ],
    acciones: [accion("generar_informe_definitivo", "Regenerar informe", {
      capacidad: CAP.prepararFormalizacion,
      confirmacion: "Se creará una nueva versión sin sustituir la anterior.",
    })],
  }),
  tarea({
    referencia: "tarea-formalizacion", orden: 14, fase: "fase-nombramiento",
    etiqueta: "Formalización y firmas",
    descripcion: "Informe, resolución, notificación, toma de posesión y comunicación al centro.",
    estadoClave: "en_curso", estado: "Firmas pendientes", responsable: "Servicio de Personal",
    entrada: "13/07/2026 10:15", tiempo: "En curso",
    paneles: [
      panel("panel-documentos-formalizacion", "tabla", "Documentos para formalización", "Cada pieza conserva plantilla, versión y estado de firma.", {
        columnas: [
          columna("orden", "Orden"), columna("documento", "Documento"),
          columna("estado", "Estado"), columna("firma", "Firma"),
        ],
        filas: [
          fila("fila-form-001", ["1", "Informe definitivo", "Generado", "Firmado"]),
          fila("fila-form-002", ["2", "Resolución de nombramiento", "Generada", "Pendiente de firma"]),
          fila("fila-form-003", ["3", "Notificación a la persona interesada", "Generada", "Pendiente de firma"]),
          fila("fila-form-004", ["4", "Toma de posesión", "Pendiente", "No iniciada"]),
          fila("fila-form-005", ["5", "Comunicación al centro", "Pendiente", "No iniciada"]),
          fila("fila-form-006", ["6", "Ficha de alta GINPIX", "Generada", "No requiere firma"]),
          fila("fila-form-007", ["7", "Índice del expediente", "Pendiente", "Pendiente"]),
        ],
      }),
      panel("panel-firmas", "comprobaciones", "Circuito de firmas", "Los firmantes y el orden proceden de configuración gobernada.", {
        campos: [
          campo("jefatura", "Jefatura de Servicio", "Firmado", { tono: "exito" }),
          campo("organo", "Órgano competente", "Pendiente de firma", { tono: "aviso" }),
          campo("intervencion", "Intervención", "Fiscalización favorable", { tono: "exito" }),
        ],
      }),
    ],
    acciones: [
      accion("generar_documentos_formalizacion", "Generar documentos pendientes", {
        capacidad: CAP.prepararFormalizacion,
        disponible: true,
        confirmacion: "Se crearán únicamente las piezas pendientes con su versión de plantilla.",
      }),
      accion("enviar_firma_formalizacion", "Enviar a firma electrónica", {
        variante: "secundaria",
        capacidad: CAP.firmarFormalizacion,
        disponible: true,
        confirmacion: "Se enviarán las piezas seleccionadas al portafirmas configurado.",
      }),
    ],
  }),
  tarea({
    referencia: "tarea-incorporacion", orden: 15, fase: "fase-incorporacion",
    etiqueta: "Incorporación",
    descripcion: "Fecha prevista, confirmación real y toma de posesión.",
    estadoClave: "pendiente", estado: "Pendiente de formalización", responsable: "Centro de trabajo",
    entrada: "Pendiente", tiempo: "Sin iniciar",
    paneles: [
      panel("panel-incorporacion", "formulario", "Incorporación", "La fecha real se confirma desde Personal.", {
        campos: [
          campo("fecha_prevista", "Fecha prevista de incorporación", "2026-08-15", { control: "fecha", obligatorio: true }),
          campo("incorporado", "Situación", "pendiente", {
            control: "radio", obligatorio: true,
            opciones: [opcion("pendiente", "Pendiente"), opcion("incorporado", "Incorporado"), opcion("incidencia", "Incidencia")],
          }),
          campo("fecha_real", "Fecha real", "", { control: "fecha" }),
          campo("observaciones", "Observaciones", "", { control: "area" }),
        ],
      }),
    ],
    acciones: [accion("confirmar_incorporacion", "Confirmar incorporación", {
      capacidad: CAP.confirmarIncorporacion,
      confirmacion: "Se solicitará a Personal la confirmación de la incorporación.",
    })],
  }),
  tarea({
    referencia: "tarea-ginpix", orden: 16, fase: "fase-incorporacion",
    etiqueta: "Integración con GINPIX",
    descripcion: "Modelo canónico común para API o fichero estructurado.",
    estadoClave: "pendiente", estado: "Preparado", responsable: "Operación GINPIX",
    entrada: "Pendiente", tiempo: "Sin iniciar",
    paneles: [
      panel("panel-ginpix-datos", "comprobaciones", "Datos preparados", "La presentación no transmite datos reales ni llama a GINPIX.", {
        campos: [
          campo("personales", "Datos personales", "Proyección preparada", { tono: "exito" }),
          campo("profesionales", "Datos profesionales", "Proyección preparada", { tono: "exito" }),
          campo("retributivos", "Datos retributivos", "Proyección preparada", { tono: "exito" }),
          campo("nombramiento", "Nombramiento", "Pendiente de firma", { tono: "aviso" }),
          campo("centro", "Centro de trabajo", "Proyección preparada", { tono: "exito" }),
        ],
      }),
      panel("panel-ginpix-exportacion", "tabla", "Vista previa del modelo canónico", "Mismo contenido para adaptadores API y fichero.", {
        columnas: [columna("campo", "Campo"), columna("valor", "Valor DEMO"), columna("estado", "Validación")],
        filas: [
          fila("fila-ginpix-001", ["TIPO_RELACION", "Nombramiento interino", "Válido"]),
          fila("fila-ginpix-002", ["CATEGORIA_REF", "cat-demo-ts-001", "Válido"]),
          fila("fila-ginpix-003", ["GRUPO_SUBGRUPO", "A2", "Válido"]),
          fila("fila-ginpix-004", ["JORNADA", "Completa", "Válido"]),
          fila("fila-ginpix-005", ["FECHA_INICIO", "15/08/2026", "Válido"]),
        ],
      }),
    ],
    acciones: [
      accion("generar_fichero_ginpix", "Generar ficha estructurada", {
        capacidad: CAP.exportarGinpix,
        confirmacion: "Se generará un artefacto DEMO sin transmisión externa.",
      }),
      accion("enviar_ginpix", "Enviar a GINPIX", {
        variante: "secundaria",
        capacidad: CAP.enviarGinpix,
        confirmacion: "En presentación no se contactará con GINPIX; se emitirá solo un recibo sintético.",
      }),
    ],
  }),
  tarea({
    referencia: "tarea-seguimiento", orden: 17, fase: "fase-seguimiento",
    etiqueta: "Seguimiento y cese",
    descripcion: "Prórrogas, incidencias, cese, conservación y cierre administrativo.",
    estadoClave: "pendiente", estado: "Pendiente de incorporación", responsable: "Servicio de Personal",
    entrada: "Pendiente", tiempo: "Sin iniciar",
    paneles: [
      panel("panel-seguimiento", "formulario", "Seguimiento de la relación", "Las causas y transiciones proceden del flujo gobernado.", {
        campos: [
          campo("situacion", "Situación", "pendiente_incorporacion", {
            control: "seleccion", obligatorio: true,
            opciones: [opcion("pendiente_incorporacion", "Pendiente de incorporación"), opcion("activa", "Relación activa"), opcion("prorroga", "Prórroga en tramitación"), opcion("incidencia", "Incidencia"), opcion("cese", "Cese en tramitación")],
          }),
          campo("fecha_efecto", "Fecha de efecto", "", { control: "fecha" }),
          campo("motivo", "Motivo catalogado", "", { control: "texto" }),
          campo("observaciones", "Observaciones", "", { control: "area" }),
        ],
      }),
      panel("panel-cierre", "aviso", "Cierre administrativo", "No puede cerrarse mientras existan tareas o firmas pendientes.", {
        campos: [
          campo("tareas", "Tareas pendientes", "4", { tono: "aviso" }),
          campo("firmas", "Firmas pendientes", "2", { tono: "aviso" }),
          campo("archivo", "Política de conservación", "Pendiente de aprobación formal", { tono: "aviso" }),
        ],
      }),
    ],
    acciones: [
      accion("registrar_seguimiento", "Registrar actuación de seguimiento", {
        capacidad: CAP.registrarSeguimiento,
        confirmacion: "Se añadirá una actuación sin reescribir las anteriores.",
      }),
      accion("cerrar_expediente", "Cerrar expediente", {
        variante: "peligro",
        capacidad: CAP.cerrarExpediente,
        confirmacion: "El cierre será rechazado mientras existan tareas pendientes.",
      }),
    ],
  }),
];

const DOCUMENTOS = [
  ["doc-demo-peticion", "Petición del centro", "PDF", 1, "Incorporado", "Firma de centro DEMO", "10/07/2026", true],
  ["doc-demo-rc", "Retención de crédito", "PDF", 1, "Validado", "Sello DEMO", "10/07/2026", true],
  ["doc-demo-informe-juridico", "Informe jurídico", "PDF/ODT", 2, "Firmado", "Jefatura de Servicio DEMO", "11/07/2026", true],
  ["doc-demo-fiscalizacion", "Informe de fiscalización", "PDF", 1, "Firmado", "Intervención DEMO", "12/07/2026", true],
  ["doc-demo-acta-llamamiento", "Acta de llamamiento", "PDF", 1, "Generado", "Pendiente de índice", "13/07/2026", true],
  ["doc-demo-resolucion", "Resolución de nombramiento", "PDF/ODT", 1, "Pendiente de firma", "Órgano competente DEMO", "13/07/2026", true],
  ["doc-demo-toma-posesion", "Toma de posesión", "PDF", 1, "Pendiente", "Sin firma", "Pendiente", false],
].map(([documento_ref, titulo, tipo, version, estado, firma, fecha, descarga_disponible]) => ({
  documento_ref, titulo, tipo, version, estado, firma, fecha, descarga_disponible,
}));

const ACTUACIONES = [
  ["act-demo-001", "10/07/2026 09:15", "Solicitud", "Registro de solicitud", "Centro DEMO", "Centro solicitante", "Completado", "Petición inicial registrada.", "doc-demo-peticion"],
  ["act-demo-002", "10/07/2026 10:00", "Análisis RRHH", "Validación de expediente", "Dirección DEMO", "RRHH", "Completado", "Modalidad, categoría y RC validadas.", "doc-demo-rc"],
  ["act-demo-003", "10/07/2026 10:28", "Gestión de bolsa", "Decisión de vía", "Responsable DEMO", "RRHH", "Completado", "Cobertura mediante bolsa vigente.", ""],
  ["act-demo-004", "10/07/2026 10:35", "Gestión de bolsa", "Asignación a unidad", "Responsable DEMO", "RRHH", "Completado", "Asignado a unidad gestora.", ""],
  ["act-demo-005", "11/07/2026 12:15", "Fiscalización", "Envío a Intervención", "Jefatura DEMO", "Unidad gestora", "Completado", "Informe firmado y enviado.", "doc-demo-informe-juridico"],
  ["act-demo-006", "12/07/2026 09:30", "Fiscalización", "Fiscalización favorable", "Intervención DEMO", "Intervención", "Completado", "Sin observaciones.", "doc-demo-fiscalizacion"],
  ["act-demo-007", "13/07/2026 09:00", "Obtención de candidato", "Aceptación de llamamiento", "Unidad DEMO", "Llamamientos", "Completado", "Aceptación sintética en plazo.", "doc-demo-acta-llamamiento"],
  ["act-demo-008", "13/07/2026 10:12", "Nombramiento", "Informe definitivo generado", "Unidad DEMO", "Personal", "Completado", "Pendiente de circuito de firmas.", "doc-demo-resolucion"],
].map(([
  actuacion_ref, fecha, fase, accion, actor, unidad, estado, observaciones, documento_ref,
]) => ({
  actuacion_ref, fecha, fase, accion, actor, unidad, estado, observaciones, documento_ref,
}));

const EXPEDIENTE = {
  esquema: "vec.contratacion_temporal.expediente.v1",
  demostracion: true,
  expediente_ref: "exp-demo-contratacion-005487",
  numero_visible: "2026/CT-05487",
  version: 12,
  flujo_ref: "flujo-demo-contratacion-temporal",
  flujo_version: 3,
  flujo_huella: "3a578f8d8ab623fbcdb497e0634938a39c93f6ac81f1c7707a52407d861f5d43",
  cabecera: CABECERA,
  fases: FASES,
  tareas: TAREAS,
};

const INDICE_DOCUMENTAL = {
  esquema: "vec.contratacion_temporal.documentos.v1",
  demostracion: true,
  expediente_ref: EXPEDIENTE.expediente_ref,
  version: EXPEDIENTE.version,
  documentos: DOCUMENTOS,
};

const AUDITORIA = {
  esquema: "vec.contratacion_temporal.auditoria.v1",
  demostracion: true,
  expediente_ref: EXPEDIENTE.expediente_ref,
  version: EXPEDIENTE.version,
  actuaciones: ACTUACIONES,
};

const CUADRO = {
  esquema: "vec.contratacion_temporal.cuadro.v1",
  demostracion: true,
  generado_en: "2026-07-23T09:30:00Z",
  indicadores: [
    { clave: "pendientes", etiqueta: "Pendientes", valor: "18", tono: "neutro" },
    { clave: "tramitacion", etiqueta: "En tramitación", valor: "24", tono: "informacion" },
    { clave: "firmas", etiqueta: "Pendientes de firma", valor: "7", tono: "aviso" },
    { clave: "intervencion", etiqueta: "En Intervención", valor: "5", tono: "aviso" },
    { clave: "finalizados", etiqueta: "Finalizados", valor: "112", tono: "exito" },
  ],
  expedientes: [
    ["exp-demo-contratacion-005487", "2026/CT-05487", "Centro DEMO de Servicios Sociales", "Trabajador/a social", "Sustitución", "en_curso", "En tramitación", "Generación documental", "10/07/2026", "Servicio de Personal", "2 días", 12],
    ["exp-demo-contratacion-005486", "2026/CT-05486", "Secretaría DEMO", "Auxiliar administrativo/a", "Vacante", "espera", "Pendiente externo", "Fiscalización", "09/07/2026", "Intervención", "Hoy", 8],
    ["exp-demo-contratacion-005485", "2026/CT-05485", "Centro DEMO de Servicios Sociales", "Educador/a social", "Programa", "incidencia", "Con incidencia", "Subsanación", "08/07/2026", "Unidad gestora", "Vencido", 9],
    ["exp-demo-contratacion-005484", "2026/CT-05484", "Infraestructuras DEMO", "Operario/a de servicios", "Acumulación", "en_curso", "En tramitación", "Llamamiento", "07/07/2026", "Unidad de llamamientos", "3 días", 11],
    ["exp-demo-contratacion-005483", "2026/CT-05483", "Tesorería DEMO", "Administrativo/a", "Vacante", "completado", "Finalizado", "Seguimiento", "06/07/2026", "Servicio de Personal", "Cerrado", 18],
  ].map(([
    expediente_ref, numero_visible, centro, categoria, modalidad, estado_clave,
    estado, fase_actual, fecha_solicitud, responsable, plazo, version,
  ]) => ({
    expediente_ref, numero_visible, centro, categoria, modalidad, estado_clave,
    estado, fase_actual, fecha_solicitud, responsable, plazo, version,
  })),
};

const CATALOGOS_ALTA = {
  esquema: "vec.contratacion_temporal.catalogos_alta.v1",
  centros: [
    {
      referencia: "centro-demo-servicios-sociales",
      etiqueta: "Centro DEMO de Servicios Sociales",
      contactos: [{ referencia: "contacto-demo-centro-001", etiqueta: "Responsable interno/a DEMO" }],
    },
    {
      referencia: "centro-demo-secretaria",
      etiqueta: "Secretaría DEMO",
      contactos: [{ referencia: "contacto-demo-centro-002", etiqueta: "Responsable interno/a DEMO" }],
    },
  ],
  categorias: [
    {
      referencia: "categoria-demo-trabajo-social",
      etiqueta: "Trabajador/a social",
      grupos_subgrupos: [{ clave: "A2", etiqueta: "A2" }],
    },
    {
      referencia: "categoria-demo-auxiliar-administrativo",
      etiqueta: "Auxiliar administrativo/a",
      grupos_subgrupos: [{ clave: "C2", etiqueta: "C2" }],
    },
  ],
  motivos: [
    { clave: "sustitucion", etiqueta: "Sustitución" },
    { clave: "vacante", etiqueta: "Vacante" },
    { clave: "acumulacion_tareas", etiqueta: "Acumulación de tareas" },
    { clave: "programa", etiqueta: "Programa temporal" },
    { clave: "relevo", etiqueta: "Relevo" },
  ],
  documentos: [
    { referencia: "doc-demo-peticion-centro", etiqueta: "Petición del centro DEMO.pdf" },
    { referencia: "doc-demo-retencion-credito", etiqueta: "Retención de crédito DEMO.pdf" },
  ],
};

export function crearCuadroContratacionTemporalPresentacion() {
  return structuredClone(CUADRO);
}

export function crearExpedienteContratacionTemporalPresentacion() {
  return structuredClone(EXPEDIENTE);
}

export function crearDocumentosContratacionTemporalPresentacion() {
  return structuredClone(INDICE_DOCUMENTAL);
}

export function crearAuditoriaContratacionTemporalPresentacion() {
  return structuredClone(AUDITORIA);
}

export function crearCatalogosAltaContratacionTemporalPresentacion() {
  return structuredClone(CATALOGOS_ALTA);
}
