/**
 * Contenido complementario de las pantallas facilitadas por RRHH.
 *
 * Es fixture sintético, no lógica de negocio. Mantenerlo separado permite
 * retirarlo al sustituir el adaptador de presentación sin tocar la vista.
 */

function campo(clave, etiqueta, valor, tono = "neutro") {
  return {
    clave,
    etiqueta,
    valor,
    tono,
    control: "solo_lectura",
    obligatorio: false,
    opciones: [],
  };
}

function columna(clave, etiqueta) {
  return { clave, etiqueta };
}

function fila(filaRef, celdas) {
  return { fila_ref: filaRef, celdas };
}

function panel(panelRef, tipo, titulo, descripcion, {
  campos = [],
  columnas = [],
  filas = [],
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

const PANELES_POR_TAREA = Object.freeze({
  "tarea-asignacion": [
    panel(
      "panel-bandeja-unidad",
      "tabla",
      "Bandeja de la unidad",
      "Vista Mis tareas · Pendientes · En espera · Finalizadas; asignación basada en unidad y competencia.",
      {
        columnas: [
          columna("expediente", "Expediente"),
          columna("categoria", "Categoría"),
          columna("entrada", "Entrada"),
          columna("plazo", "Plazo"),
          columna("estado", "Estado"),
        ],
        filas: [
          fila("fila-bandeja-001", ["2026/CT-05487", "Trabajador/a social", "10/07/2026", "2 días", "Asignado"]),
          fila("fila-bandeja-002", ["2026/CT-05484", "Operario/a de servicios", "07/07/2026", "3 días", "En llamamiento"]),
          fila("fila-bandeja-003", ["2026/CT-05485", "Educador/a social", "08/07/2026", "Vencido", "Con incidencia"]),
        ],
      },
    ),
  ],
  "tarea-informe-juridico": [
    panel(
      "panel-fuentes-informe",
      "comprobaciones",
      "Petición, RC y observaciones",
      "Fuentes incorporadas a la versión del informe; cada referencia se resolverá por su puerto.",
      {
        campos: [
          campo("peticion_ref", "Petición del centro", "doc-demo-peticion · incorporada", "exito"),
          campo("rc_ref", "Retención de crédito", "doc-demo-rc · validada", "exito"),
          campo("observaciones_informe", "Observaciones", "Sin advertencias pendientes"),
        ],
      },
    ),
    panel(
      "panel-editor-informe",
      "documentos",
      "Borrador y edición gobernada",
      "La edición real usará plantilla versionada; el navegador no firma ni guarda documentos.",
      {
        campos: [
          campo("borrador", "Borrador", "Informe jurídico DEMO · versión 3"),
          campo("editor", "Editor responsable", "Unidad DEMO de Gestión Social"),
          campo("estado_borrador", "Estado", "Preparado para vista previa", "exito"),
          campo("siguiente_informe", "Siguiente actuación", "Generar PDF accesible y remitir a firma"),
        ],
      },
    ),
  ],
  "tarea-envio-intervencion": [
    panel(
      "panel-vista-previa-firma",
      "documentos",
      "Vista previa y circuito de firma",
      "Resumen legible antes de firmar; la firma se delega en el conector institucional.",
      {
        campos: [
          campo("paginas", "Páginas", "7"),
          campo("formato_salida", "Formato", "PDF accesible"),
          campo("huella_documento", "Huella de la versión", "sha256-demo-informe-juridico-v3"),
          campo("firma_jefatura", "Firma de jefatura", "Pendiente de circuito institucional", "aviso"),
          campo("recibo_envio", "Referencia de preparación", "rec-demo-preparacion-intervencion-001"),
        ],
      },
    ),
  ],
  "tarea-fiscalizacion": [
    panel(
      "panel-modalidad-fiscalizacion",
      "comprobaciones",
      "Modalidad y remisión",
      "La modalidad, la competencia y los adjuntos quedan ligados al traslado.",
      {
        campos: [
          campo("modalidad_fiscalizacion", "Modalidad", "Fiscalización previa"),
          campo("origen_fiscalizacion", "Unidad remitente", "Servicio de Personal"),
          campo("adjuntos_fiscalizacion", "Adjuntos", "Índice de 4 documentos"),
          campo("accion_resultado", "Acción tras resultado", "Continuar o devolver para subsanación"),
        ],
      },
    ),
  ],
  "tarea-subsanacion": [
    panel(
      "panel-detalle-subsanacion",
      "tabla",
      "Observaciones, correcciones y evidencias",
      "Un reparo abriría una iteración nueva sin sustituir la fiscalización original.",
      {
        columnas: [
          columna("observacion", "Observación"),
          columna("accion", "Corrección prevista"),
          columna("responsable", "Responsable"),
          columna("evidencia", "Adjunto o evidencia"),
          columna("estado", "Estado"),
        ],
        filas: [
          fila("fila-subsanacion-001", ["Sin reparos en este expediente", "No procede", "Unidad gestora", "Informe favorable", "Cerrado"]),
          fila("fila-subsanacion-002", ["Ejemplo de circuito", "Aportar nueva RC", "Centro solicitante", "Pendiente de documento", "No iniciado"]),
        ],
      },
    ),
  ],
  "tarea-iniciar-llamamiento": [
    panel(
      "panel-historial-llamamientos",
      "tabla",
      "Historial de llamamientos",
      "Intentos anteriores y resultado, conservados en orden cronológico.",
      {
        columnas: [
          columna("intento", "Intento"),
          columna("candidatura", "Candidatura"),
          columna("canal", "Canal"),
          columna("fecha", "Fecha"),
          columna("resultado", "Resultado"),
        ],
        filas: [
          fila("fila-llamada-001", ["1", "CAND-DEMO-000", "Resultado anterior", "12/07/2026 09:10", "Renuncia declarada sintética"]),
          fila("fila-llamada-002", ["2", "CAND-DEMO-001", "Aviso local", "12/07/2026 10:20", "Preparado; sin entrega acreditada"]),
        ],
      },
    ),
  ],
  "tarea-seleccion-candidato": [
    panel(
      "panel-candidatura-seleccionada",
      "comprobaciones",
      "Candidatura seleccionada",
      "La selección concreta explica orden, elegibilidad y regla aplicada.",
      {
        campos: [
          campo("candidatura_elegida", "Referencia", "CAND-DEMO-001"),
          campo("posicion_elegida", "Posición en bolsa", "1"),
          campo("elegibilidad", "Elegibilidad", "Disponible y sin exclusiones activas", "exito"),
          campo("regla_aplicada", "Regla aplicada", "Puntuación y disponibilidad · versión DEMO"),
          campo("decision_seleccion", "Decisión", "dec-demo-tarea-010"),
        ],
      },
    ),
  ],
  "tarea-resultado-llamamiento": [
    panel(
      "panel-historial-candidatura",
      "tabla",
      "Resumen e historial de la candidatura",
      "Vista minimizada para tramitar el resultado sin exponer datos personales innecesarios.",
      {
        columnas: [
          columna("momento", "Momento"),
          columna("actuacion", "Actuación"),
          columna("resultado", "Resultado"),
          columna("evidencia", "Evidencia"),
        ],
        filas: [
          fila("fila-candidatura-hist-001", ["12/07/2026 10:20", "Aviso local", "Preparado; sin entrega acreditada", "rec-demo-aviso-local-001"]),
          fila("fila-candidatura-hist-002", ["13/07/2026 09:00", "Respuesta declarada", "Aceptación declarada sintética", "rec-demo-respuesta-declarada-001"]),
          fila("fila-candidatura-hist-003", ["13/07/2026 10:15", "Resolución manual de respuesta", "Registrada sintéticamente; sin firma ni eficacia", "rec-demo-resolucion-respuesta-001"]),
        ],
      },
    ),
  ],
  "tarea-traslado-intervencion": [
    panel(
      "panel-tarjeta-candidatura",
      "datos",
      "Tarjeta minimizada de candidatura",
      "Solo datos imprescindibles para la propuesta y el acta.",
      {
        campos: [
          campo("referencia_candidatura", "Referencia", "CAND-DEMO-001"),
          campo("categoria_candidatura", "Categoría", "Trabajador/a social"),
          campo("orden_candidatura", "Orden acreditado", "1"),
          campo("resultado_candidatura", "Resultado del llamamiento", "Aceptación declarada sintética; sin acreditar plazo", "aviso"),
          campo("acta_candidatura", "Justificante de respuesta", "rec-demo-respuesta-declarada-001"),
        ],
      },
    ),
  ],
  "tarea-informe-definitivo": [
    panel(
      "panel-resumen-candidatura-informe",
      "comprobaciones",
      "Candidatura, observaciones e historial",
      "Comprobaciones previas a la generación de la propuesta definitiva.",
      {
        campos: [
          campo("candidatura_informe", "Candidatura", "CAND-DEMO-001"),
          campo("requisitos_informe", "Requisitos", "Comprobados", "exito"),
          campo("observaciones_informe_final", "Observaciones", "Sin incidencias abiertas"),
          campo("historial_informe", "Historial", "2 llamamientos · 1 respuesta declarada sintética"),
        ],
      },
    ),
  ],
  "tarea-formalizacion": [
    panel(
      "panel-subpasos-formalizacion",
      "tabla",
      "Subpasos de formalización",
      "Cada pieza preparatoria conserva su referencia sintética; ninguna acredita firma ni eficacia.",
      {
        columnas: [
          columna("paso", "Paso"),
          columna("pieza", "Pieza"),
          columna("responsable", "Responsable"),
          columna("estado", "Estado"),
        ],
        filas: [
          fila("fila-subpaso-001", ["1", "Informe definitivo", "Jefatura de Servicio", "Borrador preparatorio"]),
          fila("fila-subpaso-002", ["2", "Resolución de nombramiento", "Órgano competente", "Borrador preparatorio"]),
          fila("fila-subpaso-003", ["3", "Diligencia", "Servicio de Personal", "Borrador preparatorio"]),
          fila("fila-subpaso-004", ["4", "Toma de posesión", "Centro de trabajo", "Borrador preparatorio"]),
          fila("fila-subpaso-005", ["5", "Notificación a la persona interesada", "Servicio de Personal", "Borrador preparatorio"]),
          fila("fila-subpaso-006", ["6", "Comunicación al centro", "Servicio de Personal", "Borrador preparatorio"]),
        ],
      },
    ),
    panel(
      "panel-vista-previa-resolucion",
      "documentos",
      "Vista previa de resolución de nombramiento",
      "Borrador preparatorio de resolución de nombramiento; los documentos reales se obtendrán por el puerto documental.",
      {
        campos: [
          campo("resolucion_version", "Versión", "1"),
          campo("resolucion_paginas", "Páginas", "3"),
          campo("resolucion_firmas", "Firmas pendientes", "Órgano competente y fe pública", "aviso"),
          campo("resolucion_estado", "Estado", "Borrador preparatorio; pendiente de portafirmas", "aviso"),
        ],
      },
    ),
  ],
  "tarea-incorporacion": [
    panel(
      "panel-proyeccion-incorporacion",
      "comprobaciones",
      "Preparación sintética de incorporación",
      "Datos minimizados para preparar la revisión; no confirman nombramiento, incorporación ni relación jurídica.",
      {
        campos: [
          campo("persona_ref_incorporacion", "Referencia de persona", "PER-DEMO-REFERENCIA-001"),
          campo("puesto_incorporacion", "Puesto o destino", "Centro DEMO de Servicios Sociales"),
          campo("relacion_incorporacion", "Relación", "Pendiente de confirmación por Personal", "aviso"),
          campo("jornada_incorporacion", "Jornada", "Pendiente de confirmación", "aviso"),
          campo("validacion_incorporacion", "Estado", "Preparación sintética; sin incorporación confirmada", "aviso"),
        ],
      },
    ),
  ],
  "tarea-ginpix": [
    panel(
      "panel-resumen-final-ginpix",
      "datos",
      "Preparación de ficha manual para GINPIX",
      "Control sintético previo a la ficha manual; no genera ni transmite datos a GINPIX.",
      {
        campos: [
          campo("expediente_ginpix", "Expediente", "2026/CT-05487"),
          campo("documentos_ginpix", "Documentos preparatorios", "Pendientes de confirmar; sin firma", "aviso"),
          campo("firmas_ginpix", "Firmas requeridas", "Pendientes en formalización", "aviso"),
          campo("validaciones_ginpix", "Ficha manual", "Pendiente de preparación y revisión", "aviso"),
        ],
      },
    ),
    panel(
      "panel-historial-ginpix",
      "tabla",
      "Historial GINPIX",
      "Preparaciones de ficha manual; no hay generación ni transmisión a GINPIX.",
      {
        columnas: [
          columna("version", "Versión"),
          columna("fecha", "Fecha"),
          columna("operacion", "Operación"),
          columna("resultado", "Resultado"),
          columna("recibo", "Recibo"),
        ],
        filas: [
          fila("fila-ginpix-hist-001", ["—", "Pendiente", "Ficha manual", "No preparada", "—"]),
          fila("fila-ginpix-hist-002", ["—", "Pendiente", "Transmisión", "No iniciada", "—"]),
        ],
      },
    ),
  ],
  "tarea-seguimiento": [
    panel(
      "panel-historial-seguimiento",
      "tabla",
      "Seguimiento y cierre administrativo",
      "Las actuaciones solo se iniciarán tras una incorporación confirmada; esta presentación no acredita cese ni cierre.",
      {
        columnas: [
          columna("fecha", "Fecha"),
          columna("situacion", "Situación"),
          columna("actuacion", "Actuación"),
          columna("documento", "Documento"),
          columna("estado", "Estado"),
        ],
        filas: [
          fila("fila-seguimiento-001", ["15/08/2026", "Incorporación", "Pendiente de confirmación por Personal", "Toma de posesión", "No iniciada"]),
          fila("fila-seguimiento-002", ["—", "Cierre administrativo", "Pendiente de incorporación y seguimiento", "Anotación sintética", "No iniciado"]),
        ],
      },
    ),
  ],
});

export function enriquecerTareasPresentacion(tareas) {
  return tareas.map((tarea) => ({
    ...tarea,
    paneles: [
      ...tarea.paneles,
      ...(PANELES_POR_TAREA[tarea.tarea_ref] ?? []),
    ],
  }));
}
