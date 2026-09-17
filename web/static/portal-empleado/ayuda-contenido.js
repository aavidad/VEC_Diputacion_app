/** Contenido de ayuda sustituible por catálogo o conector, sin lógica de negocio. */
export const AYUDA_PORTAL_BOLSA = Object.freeze({
  esquema: "vec.portal.ayuda.v1",
  titulo: "Ayuda para gestionar un llamamiento",
  introduccion: "Recorrido breve por el flujo seguro basado en una necesidad de cobertura.",
  pasos: Object.freeze([
    "Elija una necesidad autorizada y revise puesto, destino, jornada y fecha límite.",
    "Solicite al servidor la propuesta; el navegador no puede escoger personas.",
    "Revise las evaluaciones minimizadas y las versiones de bolsa y reglas.",
    "Configure plazo y canales, y compruebe el resumen antes de confirmar.",
  ]),
  preguntas: Object.freeze([
    Object.freeze({
      pregunta: "¿Por qué no aparecen nombres ni documentos?",
      respuesta: "La interfaz aplica minimización. El servidor conserva bajo autorización la relación entre la propuesta y las personas evaluadas.",
    }),
    Object.freeze({
      pregunta: "¿La presentación realiza un llamamiento?",
      respuesta: "No. La presentación muestra un recorrido sintético y no guarda, firma, comunica ni modifica expedientes.",
    }),
    Object.freeze({
      pregunta: "¿Por qué puede aparecer una acción deshabilitada?",
      respuesta: "La sesión no ha recibido una capacidad positiva o el comando de servidor todavía no está conectado.",
    }),
  ]),
  audio: Object.freeze({
    src: "/portal-empleado/assets/ayuda-llamamiento-bolsa.mp3",
    tipo: "audio/mpeg",
  }),
  transcripcion: "Guía breve para gestionar un llamamiento de bolsa. Primero, elija una necesidad de cobertura autorizada y revise el puesto, destino, jornada y plazo. Segundo, solicite la propuesta. El servidor aplica la prelación y las reglas; el navegador no permite elegir personas. Tercero, revise las evaluaciones minimizadas y las versiones de bolsa y reglas, sin datos de identidad ni contacto. Cuarto, configure el plazo y los canales. Antes de confirmar, compruebe la necesidad, la propuesta y los recibos previstos. En la presentación no se guarda ni se envía nada. Si una acción aparece deshabilitada, su sesión no tiene la capacidad necesaria o el servicio aún no está conectado.",
});

/** Ayuda contextual del módulo de contratación temporal para Recursos Humanos. */
export const AYUDA_CONTRATACION_TEMPORAL = Object.freeze({
  vistas: Object.freeze({
    cuadro: Object.freeze({
      titulo: "Ayuda — Cuadro de mando de contratación temporal",
      frases: Object.freeze([
        "El cuadro de mando centraliza y organiza las solicitudes de contratación temporal del personal de la Diputación.",
        "Permite consultar el estado de cada expediente, aplicar filtros por texto, estado o fase, y acceder a cada caso para continuar su tramitación.",
        "Los criterios de priorización de las solicitudes y los plazos de respuesta están pendientes de definir por RRHH.",
      ]),
    }),
    alta: Object.freeze({
      titulo: "Ayuda — Nueva petición de personal",
      frases: Object.freeze([
        "Este formulario permite registrar una nueva solicitud de contratación temporal para cubrir una necesidad de personal.",
        "Debe indicar el centro solicitante, la categoría profesional requerida, la jornada de trabajo prevista y la justificación del puesto.",
        "Los requisitos de la memoria justificativa y los plazos para presentar peticiones están pendientes de definir por RRHH.",
      ]),
    }),
    expediente: Object.freeze({
      titulo: "Ayuda — Detalle del expediente",
      frases: Object.freeze([
        "Esta pantalla recoge la información completa del expediente de contratación temporal y su avance administrativo.",
        "Permite examinar los antecedentes de la solicitud, consultar las fases del procedimiento y realizar las tareas disponibles.",
        "Los plazos generales de tramitación y los requisitos particulares de cada etapa están pendientes de definir por RRHH.",
      ]),
    }),
    documentos: Object.freeze({
      titulo: "Ayuda — Documentos del expediente",
      frases: Object.freeze([
        "Esta pantalla reúne los borradores y documentos administrativos generados durante la tramitación del expediente.",
        "Permite descargar los documentos en formatos estándar para su revisión antes de la firma.",
        "La integración con el portafirmas oficial y los modelos definitivos de documentos están pendientes de definir por RRHH.",
      ]),
    }),
    auditoria: Object.freeze({
      titulo: "Ayuda — Auditoría del expediente",
      frases: Object.freeze([
        "Esta pantalla muestra la trazabilidad y el registro de todas las actuaciones realizadas sobre el expediente.",
        "Permite verificar las acciones completadas, las fechas de cada hito y los cambios de fase registrados.",
        "Los periodos de conservación de los registros de auditoría están pendientes de definir por RRHH.",
      ]),
    }),
  }),
  fases: Object.freeze({
    solicitud: Object.freeze({
      paso: 1,
      titulo: "Ayuda — Paso 1: Solicitud",
      frases: Object.freeze([
        "Esta pantalla muestra la solicitud inicial de contratación registrada por el centro proponente.",
        "El personal técnico debe revisar los datos del puesto, la categoría solicitada y la retención de crédito antes de iniciar la tramitación.",
        "Los plazos máximos para tramitar esta fase inicial y la documentación complementaria exigida están pendientes de definir por RRHH.",
      ]),
    }),
    analisis_rrhh: Object.freeze({
      paso: 2,
      titulo: "Ayuda — Paso 2: Análisis RRHH",
      frases: Object.freeze([
        "En esta pantalla se realiza el análisis técnico y jurídico de la necesidad de personal temporal.",
        "Recursos Humanos revisa la modalidad de contratación adecuada, comprueba la dotación presupuestaria y registra sus observaciones.",
        "Los criterios específicos de validación del gasto y los plazos de emisión del análisis están pendientes de definir por RRHH.",
      ]),
    }),
    gestion_bolsa: Object.freeze({
      paso: 3,
      titulo: "Ayuda — Paso 3: Gestión de bolsa",
      frases: Object.freeze([
        "En este paso se comprueba la cobertura del puesto a través de la bolsa de empleo y se asigna la unidad responsable.",
        "El sistema verifica la vigencia de la bolsa y la existencia de personas candidatas disponibles para la categoría requerida.",
        "El orden de prelación definitivo, los criterios de desempate y la gestión de bolsas supletorias están pendientes de definir por RRHH.",
      ]),
    }),
    fiscalizacion: Object.freeze({
      paso: 4,
      titulo: "Ayuda — Paso 4: Fiscalización",
      frases: Object.freeze([
        "Esta fase corresponde al control previo de legalidad y fiscalización del expediente por parte de Intervención.",
        "Se prepara el informe jurídico preceptivo y, en caso de advertirse reparos u observaciones, se procede a su subsanación antes de continuar.",
        "Los plazos para subsanar reparos y el circuito de firma electrónica del informe están pendientes de definir por RRHH.",
      ]),
    }),
    obtencion_candidato: Object.freeze({
      paso: 5,
      titulo: "Ayuda — Paso 5: Obtención del candidato",
      frases: Object.freeze([
        "En esta etapa se selecciona la persona candidata de la bolsa según el orden establecido y se tramita el llamamiento.",
        "Quien gestiona el expediente debe emitir la comunicación correspondiente y registrar la acreditación de la notificación remitida.",
        "El plazo de contestación para aceptar el puesto y las causas justificadas de renuncia están pendientes de definir por RRHH.",
      ]),
    }),
    nombramiento: Object.freeze({
      paso: 6,
      titulo: "Ayuda — Paso 6: Nombramiento",
      frases: Object.freeze([
        "Esta pantalla formaliza la resolución administrativa de nombramiento tras la aceptación de la persona candidata.",
        "Se preparan los borradores de resolución y las notificaciones oficiales previas a la incorporación efectiva al puesto.",
        "El circuito de validación en portafirmas y el modelo definitivo de nombramiento están pendientes de definir por RRHH.",
      ]),
    }),
    incorporacion: Object.freeze({
      paso: 7,
      titulo: "Ayuda — Paso 7: Incorporación",
      frases: Object.freeze([
        "En esta fase se registra la toma de posesión y la incorporación de la persona seleccionada en su centro de trabajo.",
        "Se verifica la documentación acreditativa y se gestiona el alta de personal en los sistemas correspondientes.",
        "La fecha límite para tomar posesión y el protocolo de acogida en el centro de destino están pendientes de definir por RRHH.",
      ]),
    }),
    seguimiento: Object.freeze({
      paso: 8,
      titulo: "Ayuda — Paso 8: Seguimiento",
      frases: Object.freeze([
        "Esta pantalla permite supervisar la evolución del contrato temporal y tramitar las incidencias que puedan presentarse durante el desempeño.",
        "Se gestionan las posibles prórrogas del contrato, cambios en la jornada laboral o el cese administrativo al vencer la causa.",
        "Las causas tasadas de prórroga y los criterios de evaluación del periodo de prueba están pendientes de definir por RRHH.",
      ]),
    }),
  }),
});

const EQUIVALENCIAS_FASES_AYUDA = Object.freeze({
  solicitud: "solicitud",
  solicitud_registrada: "solicitud",
  analisis: "analisis_rrhh",
  analisis_rrhh: "analisis_rrhh",
  gestion_bolsa: "gestion_bolsa",
  asignacion: "gestion_bolsa",
  asignacion_unidad: "gestion_bolsa",
  fiscalizacion: "fiscalizacion",
  informe_juridico: "fiscalizacion",
  subsanacion_unidad: "fiscalizacion",
  obtencion_candidato: "obtencion_candidato",
  llamamiento: "obtencion_candidato",
  nombramiento: "nombramiento",
  incorporacion: "incorporacion",
  seguimiento: "seguimiento",
});

function normalizarClaveFase(fase) {
  if (typeof fase !== "string") return null;
  const limpia = fase.trim().toLowerCase()
    .normalize("NFD").replace(/[\u0300-\u036f]/g, "")
    .replaceAll(" ", "_")
    .replaceAll("-", "_");
  if (limpia in EQUIVALENCIAS_FASES_AYUDA) return EQUIVALENCIAS_FASES_AYUDA[limpia];
  if (limpia.includes("solicitud")) return "solicitud";
  if (limpia.includes("analisis")) return "analisis_rrhh";
  if (limpia.includes("bolsa") || limpia.includes("asignacion")) return "gestion_bolsa";
  if (limpia.includes("fiscaliz") || limpia.includes("subsanac") || limpia.includes("juridic")) return "fiscalizacion";
  if (limpia.includes("candidat") || limpia.includes("llamamiento")) return "obtencion_candidato";
  if (limpia.includes("nombramiento") || limpia.includes("formaliz")) return "nombramiento";
  if (limpia.includes("incorporac")) return "incorporacion";
  if (limpia.includes("seguimiento")) return "seguimiento";
  return null;
}

export function detectarContextoContratacionTemporal(doc = (typeof document !== "undefined" ? document : null)) {
  if (!doc) return { vista: "cuadro", fase: null };
  const botonVista = doc.querySelector?.(".ct-exp-navegacion button[data-ct-exp-vista][aria-current='page']");
  let vista = botonVista?.getAttribute?.("data-ct-exp-vista") || null;
  if (!vista) {
    if (doc.querySelector?.("form[data-ct-alta-formulario]")) vista = "alta";
    else if (doc.querySelector?.(".ct-exp-cabecera-expediente")) vista = "expediente";
    else vista = "cuadro";
  }
  let fase = null;
  if (vista === "expediente") {
    const paso = doc.querySelector?.(".ct-exp-progreso li[aria-current='step']")
      || doc.querySelector?.(".ct-exp-progreso li.en_curso")
      || doc.querySelector?.(".ct-exp-progreso li.incidencia");
    fase = paso?.getAttribute?.("data-ct-fase")
      || paso?.querySelector?.("span:not(.ct-exp-numero-fase)")?.textContent?.trim()
      || null;
    if (!fase) {
      const dts = doc.querySelectorAll?.(".ct-exp-cabecera-expediente dt") || [];
      for (const dt of dts) {
        if (dt.textContent?.trim().toLowerCase().includes("fase actual")) {
          fase = dt.nextElementSibling?.textContent?.trim() || null;
          break;
        }
      }
    }
  }
  return { vista, fase };
}

export function obtenerAyudaContratacionTemporal(vista = "cuadro", fase = null) {
  const vistaLimpia = typeof vista === "string" ? vista.trim().toLowerCase() : "cuadro";
  if (vistaLimpia === "expediente") {
    const claveFase = normalizarClaveFase(fase);
    if (claveFase && AYUDA_CONTRATACION_TEMPORAL.fases[claveFase]) {
      return AYUDA_CONTRATACION_TEMPORAL.fases[claveFase];
    }
    return AYUDA_CONTRATACION_TEMPORAL.vistas.expediente;
  }
  if (vistaLimpia in AYUDA_CONTRATACION_TEMPORAL.vistas) {
    return AYUDA_CONTRATACION_TEMPORAL.vistas[vistaLimpia];
  }
  if (vistaLimpia === "nueva_peticion" || vistaLimpia === "peticion") {
    return AYUDA_CONTRATACION_TEMPORAL.vistas.alta;
  }
  return AYUDA_CONTRATACION_TEMPORAL.vistas.cuadro;
}

export function renderizarAyudaContratacionTemporal(ayuda, escapar = (s) => s) {
  if (!ayuda || !Array.isArray(ayuda.frases)) return "";
  const contenido = `<section class="ayuda-contextual ayuda-contratacion-temporal">
  <div class="ayuda-descripcion">
${ayuda.frases.map((frase) => `    <p>${escapar(frase)}</p>`).join("\n")}
  </div>
</section>`;
  return {
    titulo: ayuda.titulo,
    contenido,
    toString() { return this.contenido; },
  };
}
