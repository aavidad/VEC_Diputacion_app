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
          campo("firma_jefatura", "Firma de jefatura", "Completada", "exito"),
          campo("recibo_envio", "Recibo de envío", "rec-demo-envio-intervencion-001"),
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
          fila("fila-llamada-001", ["1", "CAND-DEMO-000", "Correo y teléfono", "12/07/2026 09:10", "Renuncia acreditada"]),
          fila("fila-llamada-002", ["2", "CAND-DEMO-001", "Correo y teléfono", "12/07/2026 10:20", "Aceptación"]),
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
          fila("fila-candidatura-hist-001", ["12/07/2026 10:20", "Emisión de llamamiento", "Entregado", "rec-demo-llamamiento-001"]),
          fila("fila-candidatura-hist-002", ["13/07/2026 09:00", "Respuesta", "Acepta", "Acta DEMO"]),
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
          campo("resultado_candidatura", "Resultado del llamamiento", "Aceptación en plazo", "exito"),
          campo("acta_candidatura", "Acta", "doc-demo-acta-llamamiento"),
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
          campo("historial_informe", "Historial", "2 llamamientos · 1 aceptación"),
        ],
      },
    ),
  ],
  "tarea-formalizacion": [
    panel(
      "panel-subpasos-formalizacion",
      "tabla",
      "Subpasos de formalización",
      "Cada documento avanza de forma independiente y conserva su recibo.",
      {
        columnas: [
          columna("paso", "Paso"),
          columna("pieza", "Pieza"),
          columna("responsable", "Responsable"),
          columna("estado", "Estado"),
        ],
        filas: [
          fila("fila-subpaso-001", ["1", "Informe definitivo", "Jefatura de Servicio", "Firmado"]),
          fila("fila-subpaso-002", ["2", "Resolución", "Órgano competente", "Pendiente de firma"]),
          fila("fila-subpaso-003", ["3", "Notificación", "Servicio de Personal", "Preparada"]),
          fila("fila-subpaso-004", ["4", "Toma de posesión", "Centro de trabajo", "Pendiente"]),
        ],
      },
    ),
    panel(
      "panel-vista-previa-resolucion",
      "documentos",
      "Vista previa de resolución",
      "Resumen previo a firma; los documentos reales se obtendrán por el puerto documental.",
      {
        campos: [
          campo("resolucion_version", "Versión", "1"),
          campo("resolucion_paginas", "Páginas", "3"),
          campo("resolucion_firmas", "Firmas pendientes", "Órgano competente y fe pública", "aviso"),
          campo("resolucion_estado", "Estado", "Preparada para portafirmas", "aviso"),
        ],
      },
    ),
  ],
  "tarea-incorporacion": [
    panel(
      "panel-proyeccion-incorporacion",
      "comprobaciones",
      "Proyección autorizada para incorporación",
      "Datos minimizados para la unidad y el centro; la identidad completa permanece en el sistema autoritativo.",
      {
        campos: [
          campo("persona_ref_incorporacion", "Referencia de persona", "PER-DEMO-AUTORIZADA-001"),
          campo("puesto_incorporacion", "Puesto o destino", "Centro DEMO de Servicios Sociales"),
          campo("relacion_incorporacion", "Relación", "Nombramiento interino"),
          campo("jornada_incorporacion", "Jornada", "Completa"),
          campo("validacion_incorporacion", "Validación previa", "Correcta", "exito"),
        ],
      },
    ),
  ],
  "tarea-ginpix": [
    panel(
      "panel-resumen-final-ginpix",
      "datos",
      "Resumen final antes de GINPIX",
      "Control de completitud previo a generar o transmitir la proyección.",
      {
        campos: [
          campo("expediente_ginpix", "Expediente", "2026/CT-05487"),
          campo("documentos_ginpix", "Documentos preceptivos", "6 de 6", "exito"),
          campo("firmas_ginpix", "Firmas requeridas", "Pendientes en formalización", "aviso"),
          campo("validaciones_ginpix", "Validaciones del modelo", "5 de 5", "exito"),
        ],
      },
    ),
    panel(
      "panel-historial-ginpix",
      "tabla",
      "Historial GINPIX",
      "Generaciones y transmisiones, cada una con su referencia y resultado.",
      {
        columnas: [
          columna("version", "Versión"),
          columna("fecha", "Fecha"),
          columna("operacion", "Operación"),
          columna("resultado", "Resultado"),
          columna("recibo", "Recibo"),
        ],
        filas: [
          fila("fila-ginpix-hist-001", ["1", "13/07/2026 11:10", "Validación de proyección", "Correcta", "rec-demo-ginpix-validacion-001"]),
          fila("fila-ginpix-hist-002", ["—", "Pendiente", "Transmisión", "No realizada", "—"]),
        ],
      },
    ),
  ],
  "tarea-seguimiento": [
    panel(
      "panel-historial-seguimiento",
      "tabla",
      "Histórico de relación, prórroga y cese",
      "Las actuaciones se añaden sin alterar las anteriores ni cerrar tareas pendientes.",
      {
        columnas: [
          columna("fecha", "Fecha"),
          columna("situacion", "Situación"),
          columna("actuacion", "Actuación"),
          columna("documento", "Documento"),
          columna("estado", "Estado"),
        ],
        filas: [
          fila("fila-seguimiento-001", ["15/08/2026", "Incorporación prevista", "Pendiente de confirmar", "Toma de posesión", "Pendiente"]),
          fila("fila-seguimiento-002", ["14/04/2027", "Fin previsto", "Cese programable", "Resolución de cese", "No iniciado"]),
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
