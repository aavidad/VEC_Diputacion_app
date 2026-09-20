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

/**
 * Catálogo local del ayudante. Describe pasos de orientación, no reglas ni
 * operaciones administrativas: cada límite indica de forma expresa qué
 * conector o validación sigue pendiente.
 */
export const TRAMITES_AYUDANTE_PORTAL = Object.freeze([
  Object.freeze({
    id: "dietas-crear-borrador", titulo: "Crear un borrador de dieta", modulo: "Dietas", vista: "dietas", selector: "[data-dietas-borradores-propios]",
    resumen: "Prepare una comisión de servicio y sus gastos sin enviarla todavía.",
    pasos: Object.freeze([
      Object.freeze({ selector: "[data-dietas-borradores-propios]", bloqueado: true, titulo: "Abrir Dietas", instruccion: "Entre en Dietas y localice sus borradores propios.", objetivo: "Trabajar sobre una comisión de servicio propia.", preparacion: "Tener el motivo y las fechas de la comisión.", resultado: "La zona de borradores queda visible.", actor: "Persona solicitante.", limite: "La creación durable requiere el servicio de Dietas y autorización positiva." }),
      Object.freeze({ selector: "[data-dietas-recorridos]", bloqueado: true, titulo: "Completar gastos y recorrido", instruccion: "Añada los datos del desplazamiento, gastos y, si procede, la ruta orientativa.", objetivo: "Reunir la información que deberá revisar la jefatura.", preparacion: "Fechas, horas, localidades y justificantes que correspondan.", resultado: "El formulario muestra los campos pendientes de registro.", actor: "Persona solicitante.", limite: "Los importes y justificantes no se guardan ni liquidan mientras el conector siga pendiente." }),
      Object.freeze({ selector: "[data-dietas-borradores-propios]", bloqueado: true, titulo: "Revisar antes de enviar", instruccion: "Compruebe el resumen y espere a que esté disponible el envío para validación.", objetivo: "Evitar enviar una comisión incompleta.", preparacion: "Todos los datos requeridos y los justificantes aplicables.", resultado: "Podrá identificar lo pendiente antes de un futuro envío.", actor: "Persona solicitante.", limite: "Enviar, validar y liquidar son efectos separados aún no conectados." }),
    ]),
  }),
  Object.freeze({
    id: "dietas-consultar-borrador", titulo: "Consultar un borrador de dieta", modulo: "Dietas", vista: "dietas", selector: "[data-dietas-borradores-propios]",
    resumen: "Revise el estado y la información de una comisión sin cambiarla.",
    pasos: Object.freeze([
      Object.freeze({ selector: "[data-dietas-borradores-propios]", bloqueado: true, titulo: "Abrir Dietas", instruccion: "Acceda a Dietas y sitúese en el área de borradores propios.", objetivo: "Consultar únicamente comisiones del ámbito autorizado.", preparacion: "Disponer de acceso concedido al módulo.", resultado: "Se muestra la bandeja cuando exista una fuente conectada.", actor: "Persona solicitante.", limite: "La consulta real depende de la lectura autorizada de Dietas." }),
      Object.freeze({ selector: "[data-dietas-borradores-propios]", bloqueado: true, titulo: "Localizar el borrador", instruccion: "Use la referencia y el estado cuando estén disponibles.", objetivo: "Distinguir borradores, revisiones y liquidaciones sin inferir su estado.", preparacion: "La referencia de la comisión si ya la tiene.", resultado: "Podrá abrir el detalle disponible.", actor: "Persona solicitante.", limite: "No se muestran datos ni estados inventados si la fuente no responde." }),
    ]),
  }),
  Object.freeze({
    id: "dietas-ruta", titulo: "Preparar una ruta orientativa", modulo: "Dietas", vista: "dietas", selector: "[data-dietas-area-itinerario]",
    resumen: "Calcule una orientación del trayecto sin convertirla en liquidación.",
    pasos: Object.freeze([
      Object.freeze({ selector: "[data-dietas-area-itinerario]", bloqueado: false, titulo: "Abrir el itinerario", instruccion: "Desde Dietas, desplácese al apartado de recorrido.", objetivo: "Preparar los puntos del desplazamiento.", preparacion: "Origen, destino y paradas necesarias.", resultado: "La herramienta de ruta queda disponible si el proveedor responde.", actor: "Persona solicitante.", limite: "Una ruta orientativa no acredita kilometraje ni autoriza pago." }),
      Object.freeze({ selector: "[data-dietas-area-itinerario]", bloqueado: false, titulo: "Revisar el resultado", instruccion: "Compruebe los tramos y anote cualquier ajuste que deba justificarse.", objetivo: "Identificar la información que deberá verificarse en el expediente.", preparacion: "Motivo documentado de cualquier ajuste.", resultado: "Se muestra una estimación técnica separada del trámite.", actor: "Persona solicitante y, después, gestión.", limite: "La validación de kilómetros y la liquidación dependen de reglas y revisión posteriores." }),
    ]),
  }),
  Object.freeze({
    id: "cronos-corregir-marcaje", titulo: "Solicitar corrección de marcaje", modulo: "Cronos", vista: "cronos", selector: "#cronos-persona",
    resumen: "Prepare una petición de corrección para que la revise quien corresponda.",
    pasos: Object.freeze([
      Object.freeze({ selector: "#cronos-persona", bloqueado: true, titulo: "Abrir Cronos", instruccion: "Entre en Cronos y vaya al bloque de persona.", objetivo: "Consultar el recorrido de jornada y fichajes.", preparacion: "Fecha y descripción del incidente.", resultado: "El formulario de corrección queda localizado.", actor: "Persona solicitante.", limite: "El registro de marcaje todavía depende del servicio Cronos." }),
      Object.freeze({ selector: "#cronos-correccion-ayuda", bloqueado: true, titulo: "Describir la incidencia", instruccion: "Indique la fecha y el detalle cuando el formulario esté conectado.", objetivo: "Dar a la jefatura la información mínima para revisar.", preparacion: "Un motivo claro y verificable.", resultado: "La solicitud podrá quedar preparada para validación.", actor: "Persona solicitante.", limite: "No se crea ninguna corrección ni recibo desde esta guía." }),
      Object.freeze({ selector: "#cronos-responsable", bloqueado: true, titulo: "Esperar la revisión", instruccion: "Consulte la bandeja y el resultado cuando el servicio publique el estado.", objetivo: "Distinguir solicitud, revisión y resolución.", preparacion: "La solicitud registrada por el canal autorizado.", resultado: "El estado podrá reflejarse sin alterar el fichaje original.", actor: "Responsable y RRHH según competencia.", limite: "La aprobación no se infiere por pantalla ni por el acceso al menú." }),
    ]),
  }),
  Object.freeze({
    id: "cronos-solicitar-permiso", titulo: "Solicitar un permiso", modulo: "Cronos", vista: "cronos", selector: "#cronos-persona",
    resumen: "Recorra la solicitud y su revisión sin asumir concesión.",
    pasos: Object.freeze([
      Object.freeze({ selector: "#cronos-persona", bloqueado: true, titulo: "Consultar permisos", instruccion: "Abra Cronos en el bloque de persona y revise el apartado de permisos.", objetivo: "Conocer qué información se requerirá para solicitarlo.", preparacion: "Tipo, fechas y documentación exigible.", resultado: "La estructura de solicitud queda visible.", actor: "Persona solicitante.", limite: "Los saldos y reglas aplicables requieren fuente Cronos autorizada." }),
      Object.freeze({ selector: "#cronos-persona", bloqueado: true, titulo: "Presentar la solicitud", instruccion: "Complete los campos solo cuando el servicio habilite el formulario.", objetivo: "Registrar una petición trazable y revisable.", preparacion: "Datos completos y documentación permitida.", resultado: "La petición podrá entrar en la bandeja responsable.", actor: "Persona solicitante.", limite: "No se concede un permiso ni se adjuntan documentos desde la pantalla pendiente." }),
      Object.freeze({ selector: "#cronos-responsable", bloqueado: true, titulo: "Seguir la decisión", instruccion: "Consulte el resultado de responsable o RRHH cuando esté publicado.", objetivo: "Conocer el estado sin confundir solicitud con concesión.", preparacion: "Una petición registrada.", resultado: "Se podrá consultar el estado y su explicación.", actor: "Responsable o RRHH.", limite: "La decisión exige competencia positiva y auditoría durable." }),
    ]),
  }),
  Object.freeze({
    id: "cronos-consultar-saldo", titulo: "Consultar saldo y movimientos", modulo: "Cronos", vista: "cronos", selector: "#cronos-persona",
    resumen: "Vea dónde se presentarán saldo, movimientos y ausencias del periodo.",
    pasos: Object.freeze([
      Object.freeze({ selector: "#cronos-persona", bloqueado: true, titulo: "Abrir el resumen", instruccion: "Entre en Cronos y seleccione el periodo de saldo o movimientos.", objetivo: "Consultar una vista personal autorizada.", preparacion: "Elegir el periodo que se quiere revisar.", resultado: "Se muestra el selector de periodo.", actor: "Persona solicitante.", limite: "No se muestran cifras sintéticas cuando aún falta la consulta real." }),
      Object.freeze({ selector: "#cronos-persona", bloqueado: true, titulo: "Comprobar detalle", instruccion: "Revise movimientos, ausencias y calendario cuando la fuente esté disponible.", objetivo: "Separar el saldo de los eventos que lo explican.", preparacion: "Acceso concedido al ámbito personal.", resultado: "La información podrá consultarse sin modificarla.", actor: "Persona solicitante.", limite: "No se puede corregir un marcaje desde una consulta de saldo." }),
    ]),
  }),
  Object.freeze({
    id: "personal-consultar-ficha", titulo: "Consultar mi ficha personal", modulo: "Personal", vista: "personal", selector: "[data-personal-ficha-integral]",
    resumen: "Acceda a la ficha sin inventar relaciones, datos económicos ni documentos.",
    pasos: Object.freeze([
      Object.freeze({ selector: '[data-personal-ficha-tab="ficha"]', activar: '[data-personal-ficha-tab="ficha"]', bloqueado: true, titulo: "Abrir Personal", instruccion: "Entre en Personal y manténgase en la pestaña Ficha.", objetivo: "Consultar la información personal que su ámbito permita.", preparacion: "Acceso interno con la concesión correspondiente.", resultado: "La ficha indica claramente si su fuente está pendiente.", actor: "Persona solicitante.", limite: "Una ficha vacía no confirma ni niega una relación de servicio." }),
      Object.freeze({ selector: "[data-personal-ficha-integral]", bloqueado: true, titulo: "Consultar sin modificar", instruccion: "Revise los apartados disponibles y use Dietas o Cronos para sus datos propios.", objetivo: "Distinguir la ficha agregada de los datos propietarios de cada módulo.", preparacion: "Ninguna acción adicional.", resultado: "Podrá navegar a los módulos relacionados.", actor: "Persona solicitante.", limite: "La edición y descarga requieren contratos, permisos y recibos específicos." }),
    ]),
  }),
  Object.freeze({
    id: "personal-relaciones", titulo: "Consultar relaciones y servicios", modulo: "Personal", vista: "personal", selector: '[data-personal-ficha-tab="relaciones"]', activar: '[data-personal-ficha-tab="relaciones"]',
    resumen: "Localice relaciones y servicios reconocidos cuando las fuentes estén conectadas.",
    pasos: Object.freeze([
      Object.freeze({ selector: '[data-personal-ficha-tab="relaciones"]', activar: '[data-personal-ficha-tab="relaciones"]', bloqueado: true, titulo: "Abrir Relaciones", instruccion: "Entre en Personal y seleccione la pestaña Relaciones.", objetivo: "Consultar relaciones de servicio separadas de la identidad de acceso.", preparacion: "Acceso autorizado a los campos aplicables.", resultado: "Se abre la tabla de relaciones o su estado pendiente.", actor: "Persona solicitante o RRHH según ámbito.", limite: "No se deduce una relación de servicio a partir de una cuenta o certificado." }),
      Object.freeze({ selector: '[data-personal-ficha-tab="servicios"]', activar: '[data-personal-ficha-tab="servicios"]', bloqueado: true, titulo: "Consultar Servicios", instruccion: "Cambie a la pestaña Servicios para consultar los reconocimientos disponibles.", objetivo: "Diferenciar servicios reconocidos de jornada, cursos o puntuación.", preparacion: "Fuente de Personal disponible.", resultado: "Se muestra la estructura de consulta correspondiente.", actor: "Persona solicitante o RRHH según ámbito.", limite: "Los datos se mantienen en Personal; la guía no crea ni valida antigüedad." }),
    ]),
  }),
  Object.freeze({
    id: "personal-catalogos", titulo: "Consultar catálogos profesionales", modulo: "Personal", vista: "personal", selector: '[data-personal-ficha-tab="catalogos"]', activar: '[data-personal-ficha-tab="catalogos"]',
    resumen: "Abra los catálogos públicos integrados desde Personal.",
    pasos: Object.freeze([
      Object.freeze({ selector: '[data-personal-ficha-tab="catalogos"]', activar: '[data-personal-ficha-tab="catalogos"]', bloqueado: false, titulo: "Abrir Catálogos", instruccion: "Entre en Personal y seleccione la pestaña Catálogos.", objetivo: "Consultar las referencias disponibles sin mezclar datos personales.", preparacion: "Acceso permitido a la consulta de catálogos.", resultado: "Se cargan los catálogos conectados al abrir la pestaña.", actor: "Persona solicitante, RRHH o jefatura según el catálogo.", limite: "Los catálogos no conceden por sí mismos permisos ni modifican puestos." }),
      Object.freeze({ selector: "[data-personal-ficha-integral]", bloqueado: false, titulo: "Usar la referencia", instruccion: "Revise la información y vuelva al trámite que requiera esa referencia.", objetivo: "Preparar un trámite sin copiar ni alterar datos gobernados.", preparacion: "Identificar el catálogo que se necesita.", resultado: "La consulta queda separada de los efectos administrativos.", actor: "Quien tramite dentro de su competencia.", limite: "Las decisiones posteriores requieren su propio módulo y autorización." }),
    ]),
  }),
]);

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
