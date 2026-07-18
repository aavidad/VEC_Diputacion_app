import { validarDatosAreaPersonal, validarRecibo } from "./contrato.js";
import { calcularAutobaremo } from "./calculo-autobaremo.js";

/**
 * Adaptador efímero y exclusivo de presentación.
 *
 * No usa red, cookies ni almacenamiento del navegador. Todo el estado vive en
 * la instancia y desaparece al recargar. La aplicación productiva no importa
 * este módulo: `arranque.js` solo lo carga con `?presentacion=rrhh`.
 */
const BASE_PRESENTACION = {
  meta: {
    esquema: "vec.bolsa.area-personal.v1",
    presentacion: true,
    origen: "Adaptador efímero con referencias públicas reales",
    generado_en: "2026-07-18T09:00:00Z",
  },
  sesion: {
    persona_ref: "DEMO-PER-0001",
    nombre_visible: "Persona Aspirante de Demostración",
    iniciales: "PD",
    metodo: "Certificado electrónico simulado",
  },
  resumen: {
    acciones_pendientes: 3,
    convocatorias_abiertas: 0,
    solicitudes_activas: 2,
    mensajes_no_leidos: 2,
    puntuacion_provisional: 14.75,
  },
  capacidades: {
    actualizar_contacto: true,
    incorporar_merito: true,
    guardar_borrador: true,
    calcular_autobaremo: true,
    iniciar_pago: true,
    firmar_solicitud: true,
    registrar_solicitud: true,
    cambiar_disponibilidad: true,
    responder_llamamiento: true,
    presentar_subsanacion: true,
    presentar_alegacion: true,
    marcar_mensaje: true,
    actualizar_notificaciones: true,
    solicitar_certificado: true,
    solicitar_descarga: true,
  },
  perfil: {
    referencia: "DEMO-PERFIL-0001",
    nombre_visible: "Persona Aspirante de Demostración",
    identificador_visible: "ID-DEMO-SIN-VALIDEZ",
    correo: "aspirante@vec-demo.test",
    telefono: "000 000 000",
    domicilio: "Domicilio sintético sin localización real",
    estado_verificacion: "Identidad simulada · Contacto pendiente de revisión",
    provincia: "Granada (dato sintético)",
    idioma: "Castellano",
    canales: ["Correo de demostración", "Aviso interno"],
  },
  preferencias_notificacion: {
    correo: true,
    telegram: false,
    interno: true,
    convocatorias: true,
    plazos: true,
    llamamientos: true,
    noticias: false,
  },
  plazos: [
    { id: "DEMO-PLAZO-001", dia: "21", mes: "JUL", titulo: "Completar solicitud simulada", detalle: "Bolsa de empleo de Operario · plazo exclusivamente DEMO", estado: "Simulación sin efectos", ruta: "solicitud" },
    { id: "DEMO-PLAZO-002", dia: "23", mes: "JUL", titulo: "Responder subsanación", detalle: "Expediente DEMO-SOL-0027 · 14:00", estado: "Acción requerida", ruta: "subsanaciones" },
    { id: "DEMO-PLAZO-003", dia: "29", mes: "JUL", titulo: "Fin de alegaciones", detalle: "Listado provisional · 23:59", estado: "Próximamente", ruta: "alegaciones" },
  ],
  convocatorias: [
    {
      id: "DEMO-CONV-001", referencia: "BOP-GRA-2026-043004", titulo: "Bolsa de empleo de Operario de la Diputación de Granada", categoria: "Operario/a",
      estado: "Histórico cerrado", plazo: "Sin plazo abierto en el portal · publicación 05/03/2026", descripcion: "Bases públicas para elaborar una bolsa de empleo de Operario de la Escala de Administración Especial. Expediente 2026/BTN_01/000062.",
      plazas: "Bolsa de empleo temporal", tasa: "No incorporada como dato estructurado", presentacion_hasta: "Proceso histórico · sin plazo abierto",
      publicada_en: "05/03/2026", cve_bop: "BOP-GRA-2026-043004", recorrido_demo: true,
      requisitos: ["Nacionalidad en alguno de los supuestos admitidos por las bases", "Tener cumplidos dieciséis años y no superar la edad máxima de jubilación forzosa", "Capacidad funcional para las tareas", "No haber sido separado ni hallarse inhabilitado", "Sin titulación académica específica para esta agrupación profesional", "Ser personal ajeno a la Función Pública de la Diputación, según las bases"],
      documentos: [
        { titulo: "Bases adaptadas de la bolsa de Operario (PDF)", formato: "PDF", url: "/bolsa/documentos/bases-operario-demo.pdf", aviso: "Reproducción accesible DEMO basada en BOP-GRA-2026-043004; sin validez administrativa" },
        { titulo: "Bases adaptadas de la bolsa de Operario (HTML accesible)", formato: "HTML", url: "/bolsa/documentos/bases-operario-demo.html", aviso: "Versión navegable DEMO con fuentes oficiales identificadas y sin datos personales" },
      ],
    },
    {
      id: "DEMO-CONV-002", referencia: "BOP-GRA-2025-125002", titulo: "Ingreso en la categoría de Auxiliar de Servicios Generales", categoria: "Auxiliar de Servicios Generales",
      estado: "Histórico cerrado", plazo: "Sin plazo abierto en el portal · publicación 04/07/2025", descripcion: "Proceso selectivo público de la Diputación de Granada para el ingreso en la categoría de Auxiliar de Servicios Generales de la Escala de Administración Especial.",
      plazas: "Proceso selectivo · véanse las bases", tasa: "No incorporada como dato estructurado", presentacion_hasta: "Proceso histórico · sin plazo abierto",
      publicada_en: "04/07/2025", cve_bop: "BOP-GRA-2025-125002", recorrido_demo: false,
      requisitos: ["Nacionalidad en alguno de los supuestos admitidos por las bases", "Tener cumplidos dieciséis años y no superar la edad máxima de jubilación forzosa", "Capacidad funcional para las plazas", "No haber sido separado ni hallarse inhabilitado", "Título de Graduado en Educación Secundaria Obligatoria o equivalente indicado por las bases"],
      documentos: [
        { titulo: "Bases adaptadas de Auxiliar de Servicios Generales (PDF)", formato: "PDF", url: "/bolsa/documentos/bases-auxiliar-demo.pdf", aviso: "Reproducción accesible DEMO basada en BOP-GRA-2025-125002; sin validez administrativa" },
        { titulo: "Bases adaptadas de Auxiliar de Servicios Generales (HTML accesible)", formato: "HTML", url: "/bolsa/documentos/bases-auxiliar-demo.html", aviso: "Versión navegable DEMO con fuentes oficiales identificadas y sin datos personales" },
      ],
    },
    {
      id: "DEMO-CONV-003", referencia: "BOP-GRA-2024-244002", titulo: "Ingreso en la Subescala de Gestión de Administración General", categoria: "Técnico de Gestión",
      estado: "Histórico cerrado", plazo: "Sin plazo abierto en el portal · publicación 19/12/2024", descripcion: "Bases del proceso selectivo de la Diputación de Granada para el ingreso en la Subescala de Gestión de Administración General, con turnos libre y de promoción interna según la publicación.",
      plazas: "Proceso selectivo · véanse las bases", tasa: "No incorporada como dato estructurado", presentacion_hasta: "Proceso histórico · sin plazo abierto",
      publicada_en: "19/12/2024", cve_bop: "BOP-GRA-2024-244002", recorrido_demo: false,
      requisitos: ["Nacionalidad en alguno de los supuestos admitidos por las bases", "Tener cumplidos dieciséis años y no superar la edad máxima de jubilación forzosa", "Capacidad funcional para las plazas", "No haber sido separado ni hallarse inhabilitado", "Titulación y, para promoción interna, pertenencia y antigüedad exigidas para cada categoría"],
      documentos: [
        { titulo: "Bases adaptadas de Gestión de Administración General (PDF)", formato: "PDF", url: "/bolsa/documentos/bases-gestion-demo.pdf", aviso: "Reproducción accesible DEMO basada en BOP-GRA-2024-244002; sin validez administrativa" },
        { titulo: "Bases adaptadas de Gestión de Administración General (HTML accesible)", formato: "HTML", url: "/bolsa/documentos/bases-gestion-demo.html", aviso: "Versión navegable DEMO con fuentes oficiales identificadas y sin datos personales" },
      ],
    },
  ],
  meritos: [
    { id: "DEMO-MER-001", tipo: "Titulación", titulo: "Titulación de demostración de nivel 4", detalle: "Rama administrativa · emitida 2021 (dato sintético)", estado: "Validado", documento_ref: "DEMO-DOC-001", puntos_estimados: 2.0 },
    { id: "DEMO-MER-002", tipo: "Experiencia", titulo: "Experiencia en administración pública simulada", detalle: "18 meses · jornada completa · origen de oficio simulado", estado: "Pendiente de contraste", documento_ref: "DEMO-DOC-002", puntos_estimados: 5.4 },
    { id: "DEMO-MER-003", tipo: "Formación", titulo: "Curso sintético de procedimiento administrativo", detalle: "60 horas · entidad de formación de demostración", estado: "Aportado", documento_ref: "DEMO-DOC-003", puntos_estimados: 0.6 },
    { id: "DEMO-MER-004", tipo: "Experiencia", titulo: "Experiencia externa simulada", detalle: "12 meses · jornada 50 % · equivalencia calculada: 6 meses", estado: "Requiere subsanación", documento_ref: "DEMO-DOC-004", puntos_estimados: 1.8 },
  ],
  documentos: [
    { id: "DEMO-DOC-001", nombre: "titulo-demostracion.pdf", tipo: "Titulación", fecha: "10/07/2026", estado: "Firma simulada verificada", huella: "DEMO-SHA256-001" },
    { id: "DEMO-DOC-002", nombre: "servicios-previos-demostracion.pdf", tipo: "Experiencia", fecha: "10/07/2026", estado: "Obtenido de oficio (simulado)", huella: "DEMO-SHA256-002" },
    { id: "DEMO-DOC-003", nombre: "curso-demostracion.pdf", tipo: "Formación", fecha: "11/07/2026", estado: "Pendiente de validación", huella: "DEMO-SHA256-003" },
    { id: "DEMO-DOC-004", nombre: "contrato-demostracion.pdf", tipo: "Experiencia", fecha: "12/07/2026", estado: "Subsanación requerida", huella: "DEMO-SHA256-004" },
  ],
  solicitudes: [
    { id: "DEMO-SOL-0027", convocatoria_id: "DEMO-CONV-001", referencia: "DEMO-REG-2026-0027", titulo: "Bolsa de empleo de Operario de la Diputación de Granada", estado: "Tramitación íntegramente sintética", actualizado: "17/07/2026 12:10 · DEMO", posicion: "18 de 146 (dato sintético)", puntuacion: 14.75, siguiente: "Responder subsanación DEMO antes del 23/07/2026", pago: "Tasa simulada abonada", firma: "Firma simulada válida" },
    { id: "DEMO-SOL-0018", convocatoria_id: "DEMO-CONV-003", referencia: "DEMO-REG-2026-0018", titulo: "Ingreso en la Subescala de Gestión de Administración General", estado: "Listado provisional sintético", actualizado: "16/07/2026 09:20 · DEMO", posicion: "7 de 82 (dato sintético)", puntuacion: 11.2, siguiente: "Puede presentar una alegación DEMO hasta el 29/07/2026", pago: "Exención simulada", firma: "Firma simulada válida" },
  ],
  baremo: [
    { id: "DEMO-BAR-001", merito_id: "DEMO-MER-002", nombre: "Experiencia en la Diputación", detalle: "18 meses × 0,30 puntos · jornada completa", estado: "De oficio · pendiente de revisión", puntos: 5.4, maximo: 8 },
    { id: "DEMO-BAR-002", merito_id: "DEMO-MER-004", nombre: "Experiencia en otras administraciones", detalle: "12 meses × 50 % × 0,30 puntos", estado: "Autobaremado", puntos: 1.8, maximo: 4 },
    { id: "DEMO-BAR-003", merito_id: "DEMO-MER-001", nombre: "Titulaciones adicionales", detalle: "Una titulación de la misma rama", estado: "Validado", puntos: 2, maximo: 3 },
    { id: "DEMO-BAR-004", merito_id: "DEMO-MER-003", nombre: "Formación relacionada", detalle: "60 horas computables × 0,01 puntos", estado: "Pendiente de validación", puntos: 0.6, maximo: 2 },
    { id: "DEMO-BAR-005", nombre: "Ejercicio superado", detalle: "Resultado sintético importado del proceso", estado: "De oficio", puntos: 4.95, maximo: 5, de_oficio: true },
  ],
  disponibilidad: {
    disponible: true,
    estado: "Disponible para llamamientos",
    desde: "01/07/2026",
    bolsas: ["Bolsa de empleo de Operario de la Diputación de Granada (adscripción DEMO)", "Ingreso en la Subescala de Gestión de Administración General (adscripción DEMO)"],
    proxima_revision: "31/12/2026",
  },
  llamamientos: [
    { id: "DEMO-LLA-0045", bolsa: "Bolsa de empleo de Operario de la Diputación de Granada · escenario DEMO", puesto: "Destino y puesto sintéticos", plazo: "Responder antes del 19/07/2026 14:00 · plazo DEMO", estado: "Pendiente de respuesta", jornada: "Completa (dato sintético)", duracion: "3 meses (dato sintético)", posicion: "Primera persona elegible de demostración" },
    { id: "DEMO-LLA-0031", bolsa: "Ingreso en la Subescala de Gestión de Administración General · escenario DEMO", puesto: "Destino y puesto sintéticos", plazo: "Respondido el 02/07/2026 · DEMO", estado: "Aceptado · comprobación pendiente", jornada: "Parcial 50 % (dato sintético)", duracion: "1 mes (dato sintético)", posicion: "Respuesta registrada en demostración" },
  ],
  subsanaciones: [
    { id: "DEMO-SUB-0008", solicitud_ref: "DEMO-SOL-0027", motivo: "Acreditar la jornada de la experiencia externa", plazo: "23/07/2026 14:00", estado: "Pendiente", documento_solicitado: "Certificado con porcentaje de jornada y periodos exactos" },
  ],
  alegaciones: [
    { id: "DEMO-ALE-0003", solicitud_ref: "DEMO-SOL-0018", asunto: "Revisión de formación no computada", estado: "Borrador", fecha: "Creada 17/07/2026" },
  ],
  mensajes: [
    { id: "DEMO-MSG-001", asunto: "Subsanación disponible", resumen: "Revise el requerimiento de su solicitud DEMO-SOL-0027.", fecha: "17/07/2026 12:10", estado: "No leído", tipo: "Acción requerida", ruta: "subsanaciones" },
    { id: "DEMO-MSG-002", asunto: "Nuevo llamamiento", resumen: "Dispone de un llamamiento pendiente de respuesta.", fecha: "17/07/2026 10:30", estado: "No leído", tipo: "Plazo activo", ruta: "llamamientos" },
    { id: "DEMO-MSG-003", asunto: "Listado provisional publicado", resumen: "Puede consultar su posición y puntuación desglosada.", fecha: "16/07/2026 09:20", estado: "Leído", tipo: "Información", ruta: "seguimiento" },
  ],
  certificados: [
    { id: "DEMO-CER-001", tipo: "Certificado de inscripción en bolsa", descripcion: "Acredita la situación mostrada en una bolsa concreta.", estado: "Disponible bajo solicitud", formatos: "PDF, ODT o JSON" },
    { id: "DEMO-CER-002", tipo: "Justificante de registro", descripcion: "Copia de la solicitud y del asiento de presentación.", estado: "Disponible", formatos: "PDF o JSON" },
    { id: "DEMO-CER-003", tipo: "Informe de méritos aportados", descripcion: "Relación de méritos y documentos del expediente personal.", estado: "Disponible bajo solicitud", formatos: "PDF, CSV, ODT o JSON" },
  ],
  actividad: [
    { id: "DEMO-ACT-001", titulo: "Requerimiento de subsanación", detalle: "Se abrió un plazo para completar una experiencia.", fecha: "17/07/2026 12:10", actor: "Servicio de Selección (demostración)", recibo: "DEMO-EVT-001" },
    { id: "DEMO-ACT-002", titulo: "Puntuación provisional calculada", detalle: "Resultado sujeto a revisión técnica y alegaciones.", fecha: "16/07/2026 09:20", actor: "Motor de baremación (demostración)", recibo: "DEMO-EVT-002" },
    { id: "DEMO-ACT-003", titulo: "Solicitud registrada", detalle: "Firma, tasa y asiento simulados.", fecha: "12/07/2026 18:42", actor: "Persona Aspirante de Demostración", recibo: "DEMO-REG-2026-0027" },
  ],
  ayuda: [
    { pregunta: "¿Cómo me inscribo en una convocatoria?", respuesta: "Abra Convocatorias, revise bases y requisitos, y seleccione Iniciar solicitud. El recorrido separa borrador, pago, firma y registro." },
    { pregunta: "¿Puedo reutilizar mis méritos?", respuesta: "Sí. El inventario personal permite reutilizar documentación, pero cada convocatoria aplica sus propias bases y puede exigir acreditación adicional." },
    { pregunta: "¿La autobaremación es definitiva?", respuesta: "No. Es una estimación trazable aplicada a las bases de la convocatoria. El personal técnico revisa cada mérito y puede aceptar o rechazar su cómputo con constancia." },
    { pregunta: "¿Cómo respondo a un llamamiento?", respuesta: "Abra Disponibilidad y llamamientos, revise puesto, jornada, duración y plazo, y confirme su respuesta. La respuesta administrativa real requerirá el servicio conectado." },
    { pregunta: "¿Qué ocurre en esta demostración?", respuesta: "Nada sale del navegador ni afecta a expedientes. Cada acción genera un recibo inequívoco DEMO y el estado se pierde al recargar." },
  ],
};

const ACCIONES = new Set([
  "actualizar_contacto", "incorporar_merito", "guardar_borrador", "calcular_autobaremo",
  "iniciar_pago", "firmar_solicitud", "registrar_solicitud", "cambiar_disponibilidad",
  "responder_llamamiento", "presentar_subsanacion", "presentar_alegacion", "marcar_mensaje",
  "actualizar_notificaciones", "solicitar_certificado", "solicitar_descarga",
]);

function referenciaDemo(valor, alternativa) {
  const texto = typeof valor === "string" && /^DEMO-[A-Z0-9._/-]+$/i.test(valor) ? valor : alternativa;
  return texto.slice(0, 100);
}

function marcado(valor) {
  return valor === true || valor === "true" || valor === "on";
}

function referencias(valor) {
  return [...new Set((Array.isArray(valor) ? valor : valor ? [valor] : [])
    .filter((item) => typeof item === "string" && /^DEMO-[A-Z0-9._/-]+$/i.test(item))
    .map((item) => item.slice(0, 100)))];
}

export function crearAdaptadorPresentacion() {
  let estado = structuredClone(BASE_PRESENTACION);
  let secuencia = 0;

  async function cargar() {
    estado.meta.generado_en = new Date().toISOString();
    estado.resumen.mensajes_no_leidos = estado.mensajes.filter((item) => item.estado === "No leído").length;
    estado.resumen.acciones_pendientes = estado.subsanaciones.filter((item) => item.estado === "Pendiente").length
      + estado.llamamientos.filter((item) => item.estado === "Pendiente de respuesta").length
      + estado.mensajes.filter((item) => item.estado === "No leído").length;
    return validarDatosAreaPersonal(estado, { presentacionEsperada: true });
  }

  function crearRecibo(accion, payload) {
    secuencia += 1;
    const objetivo = referenciaDemo(
      payload.id || payload.convocatoria_id || estado.sesion.persona_ref,
      estado.sesion.persona_ref,
    );
    return validarRecibo({
      esquema: "vec.bolsa.area-personal.recibo-demo.v1",
      presentacion: true,
      referencia: `DEMO-REC-${String(secuencia).padStart(4, "0")}`,
      accion,
      objetivo,
      resultado: "Simulación completada sin efectos administrativos",
      actor: "Persona Aspirante de Demostración",
      fecha: new Date().toISOString(),
      advertencia: "RECIBO DEMO · No acredita presentación, firma, pago, registro ni comunicación real.",
    }, { presentacionEsperada: true });
  }

  function exigirBorrador(payload) {
    const id = referenciaDemo(payload.id, "");
    if (!id) throw new Error("La operación exige la referencia exacta del borrador seleccionado.");
    const solicitud = estado.solicitudes.find((item) => item.id === id);
    if (!solicitud) throw new Error("El borrador seleccionado no existe en el ámbito de la presentación.");
    if (!/borrador/i.test(solicitud.estado)) throw new Error("La solicitud seleccionada ya no admite esta operación de borrador.");
    return solicitud;
  }

  function exigirContenidoCompleto(payload) {
    const meritosRecibidos = Array.isArray(payload.meritos_ids)
      ? payload.meritos_ids
      : payload.meritos_ids ? [payload.meritos_ids] : [];
    if (meritosRecibidos.some((id) => typeof id !== "string" || !/^DEMO-[A-Z0-9._/-]+$/i.test(id))) {
      throw new Error("La solicitud contiene una referencia de mérito no válida.");
    }
    const meritos = referencias(payload.meritos_ids);
    if (!marcado(payload.requisitos_confirmados)) throw new Error("Falta la declaración de requisitos y lectura de bases.");
    if (!marcado(payload.datos_confirmados)) throw new Error("Falta la confirmación de datos personales y de contacto.");
    if (meritos.length === 0) throw new Error("La solicitud debe asociar al menos un mérito.");
    if (!marcado(payload.autobaremo_revisado)) throw new Error("Falta revisar la autobaremación antes de guardar el borrador.");
    if (meritos.some((id) => !estado.meritos.some((item) => item.id === id))) {
      throw new Error("La solicitud contiene un mérito ajeno al inventario autorizado.");
    }
    return meritos;
  }

  function pagoConfirmado(solicitud) {
    return !/pendiente/i.test(solicitud.pago) && /confirmad|abonad|exent/i.test(solicitud.pago);
  }

  function firmaConfirmada(solicitud) {
    return !/pendiente/i.test(solicitud.firma) && /confirmad|firmad|válid/i.test(solicitud.firma);
  }

  function exigirElemento(coleccion, id, etiqueta, estadosPermitidos = null) {
    const referencia = referenciaDemo(id, "");
    if (!referencia) throw new Error(`La operación exige una referencia válida de ${etiqueta}.`);
    const elemento = coleccion.find((item) => item.id === referencia);
    if (!elemento) throw new Error(`El ${etiqueta} indicado no existe en el ámbito de la presentación.`);
    if (estadosPermitidos && !estadosPermitidos.includes(elemento.estado)) {
      throw new Error(`El ${etiqueta} indicado ya no admite esta operación.`);
    }
    return elemento;
  }

  function aplicar(accion, payload) {
    if (accion === "actualizar_contacto") {
      const correo = String(payload.correo || estado.perfil.correo).trim().slice(0, 160);
      if (!/^[^\s@]+@[^\s@]+\.test$/iu.test(correo)) {
        throw new Error("En presentación, el correo debe ser sintético y usar un dominio reservado .test.");
      }
      estado.perfil.correo = correo;
      estado.perfil.telefono = String(payload.telefono || estado.perfil.telefono).slice(0, 40);
      estado.perfil.domicilio = String(payload.domicilio || estado.perfil.domicilio).slice(0, 300);
    } else if (accion === "incorporar_merito") {
      secuencia += 1;
      const documentoRef = `DEMO-DOC-NUEVO-${secuencia}`;
      estado.meritos.push({
        id: `DEMO-MER-NUEVO-${secuencia}`, tipo: String(payload.tipo || "Otro").slice(0, 80),
        titulo: String(payload.titulo || "Mérito incorporado en demostración").slice(0, 300),
        detalle: "Documento sintético pendiente de validación", estado: "Aportado en demostración",
        documento_ref: documentoRef, puntos_estimados: 0,
      });
      estado.documentos.push({
        id: documentoRef,
        nombre: `evidencia-merito-${String(secuencia).padStart(4, "0")}-demo.pdf`,
        tipo: "Evidencia de mérito",
        fecha: "Ahora · se perderá al recargar",
        estado: payload.documento ? "Nombre original no conservado · contenido no leído" : "Sin fichero aportado",
        huella: `DEMO-SIN-HUELLA-${String(secuencia).padStart(4, "0")}`,
      });
    } else if (accion === "guardar_borrador") {
      const convocatoriaId = referenciaDemo(payload.convocatoria_id, "");
      const convocatoria = estado.convocatorias.find((item) => item.id === convocatoriaId && item.recorrido_demo === true);
      if (!convocatoria) throw new Error("La convocatoria seleccionada no admite el recorrido simulado de solicitud.");
      const meritos = exigirContenidoCompleto(payload);
      let existente;
      if (payload.id) {
        const solicitudId = referenciaDemo(payload.id, "");
        if (!solicitudId) throw new Error("La referencia de borrador no es válida.");
        existente = estado.solicitudes.find((item) => item.id === solicitudId);
        if (!existente) throw new Error("El borrador seleccionado no existe en el ámbito de la presentación.");
      } else {
        existente = estado.solicitudes.find((item) => item.convocatoria_id === convocatoriaId && /borrador/i.test(item.estado));
      }
      if (existente && (!/borrador/i.test(existente.estado) || existente.convocatoria_id !== convocatoriaId)) {
        throw new Error("La referencia indicada no corresponde al borrador de esta convocatoria.");
      }
      if (!existente) {
        const numero = estado.solicitudes.filter((item) => /borrador/i.test(item.estado)).length + 1;
        existente = {
          id: `DEMO-SOL-BORRADOR-${String(numero).padStart(4, "0")}`,
          convocatoria_id: convocatoriaId,
          referencia: `DEMO-BORRADOR-${String(numero).padStart(4, "0")}`,
          titulo: convocatoria.titulo,
          estado: "Borrador efímero",
          actualizado: "Ahora · se perderá al recargar",
          posicion: "No aplica",
          puntuacion: 0,
          siguiente: "Confirmar pago o exención, firmar y registrar",
          pago: "Pendiente",
          firma: "Pendiente",
        };
        estado.solicitudes.push(existente);
      }
      existente.requisitos_confirmados = true;
      existente.datos_confirmados = true;
      existente.meritos_ids = meritos;
      existente.autobaremo_revisado = true;
      existente.puntuacion = calcularAutobaremo(estado, meritos).total;
      existente.firma = "Pendiente";
      existente.actualizado = "Ahora · se perderá al recargar";
    } else if (accion === "calcular_autobaremo") {
      const convocatoria = exigirElemento(estado.convocatorias, payload.convocatoria_id || payload.id, "convocatoria");
      const meritos = referencias(payload.meritos_ids);
      if (meritos.length === 0 || meritos.some((id) => !estado.meritos.some((item) => item.id === id))) {
        throw new Error("Seleccione méritos válidos del inventario antes de recalcular la autobaremación.");
      }
      estado.resultado_autobaremo = {
        convocatoria_id: convocatoria.id,
        meritos_ids: meritos,
        puntos: calcularAutobaremo(estado, meritos).total,
        calculado_en: new Date().toISOString(),
      };
    } else if (accion === "iniciar_pago") {
      const solicitud = exigirBorrador(payload);
      exigirContenidoCompleto(solicitud);
      solicitud.pago = "Pago o exención DEMO confirmado · sin cargo real";
    } else if (accion === "firmar_solicitud") {
      const solicitud = exigirBorrador(payload);
      exigirContenidoCompleto(solicitud);
      if (!pagoConfirmado(solicitud)) throw new Error("Debe confirmar el pago o la exención antes de firmar.");
      solicitud.firma = "Firma DEMO confirmada · sin firma electrónica real";
    } else if (accion === "registrar_solicitud") {
      const solicitud = exigirBorrador(payload);
      exigirContenidoCompleto(solicitud);
      if (!pagoConfirmado(solicitud)) throw new Error("No se puede registrar sin pago o exención confirmados.");
      if (!firmaConfirmada(solicitud)) throw new Error("No se puede registrar sin firma confirmada.");
      if (!marcado(payload.declaracion_final)) throw new Error("Debe confirmar la declaración final antes del registro.");
      solicitud.estado = "Registro DEMO completado · sin asiento real";
      solicitud.referencia = `DEMO-REG-NUEVO-${solicitud.id.slice(-4)}`;
      solicitud.siguiente = "Consultar el recibo DEMO sin validez administrativa";
    } else if (accion === "cambiar_disponibilidad") {
      estado.disponibilidad.disponible = payload.disponible === true;
      estado.disponibilidad.estado = payload.disponible === true ? "Disponible para llamamientos" : "No disponible (demostración)";
    } else if (accion === "responder_llamamiento") {
      const item = exigirElemento(estado.llamamientos, payload.id, "llamamiento", ["Pendiente de respuesta"]);
      if (!new Set(["aceptar", "rechazar"]).has(payload.respuesta)) throw new Error("La respuesta al llamamiento no es válida.");
      item.estado = payload.respuesta === "aceptar" ? "Aceptado en demostración" : "Rechazado en demostración";
    } else if (accion === "presentar_subsanacion") {
      const item = exigirElemento(estado.subsanaciones, payload.id, "requerimiento de subsanación", ["Pendiente"]);
      item.estado = "Presentada en demostración";
    } else if (accion === "presentar_alegacion") {
      const item = exigirElemento(estado.alegaciones, payload.id, "borrador de alegación", ["Borrador"]);
      item.estado = "Presentada en demostración";
    } else if (accion === "marcar_mensaje") {
      const item = exigirElemento(estado.mensajes, payload.id, "mensaje", ["No leído"]);
      item.estado = "Leído";
    } else if (accion === "actualizar_notificaciones") {
      const nombres = ["correo", "telegram", "interno", "convocatorias", "plazos", "llamamientos", "noticias"];
      nombres.forEach((nombre) => { estado.preferencias_notificacion[nombre] = marcado(payload[nombre]); });
    } else if (accion === "solicitar_certificado") {
      const item = exigirElemento(estado.certificados, payload.id, "certificado");
      item.estado = `Generado en demostración · ${String(payload.formato || "PDF").toUpperCase()}`;
    } else if (accion === "solicitar_descarga") {
      const recursos = [...estado.documentos, ...estado.solicitudes, ...estado.convocatorias];
      exigirElemento(recursos, payload.id, "recurso descargable");
    }
  }

  async function ejecutar({ accion, payload = {}, confirmacion = false, capacidad = false } = {}) {
    if (!ACCIONES.has(accion)) throw new Error("La acción no pertenece al adaptador de presentación.");
    if (capacidad !== true || estado.capacidades[accion] !== true) throw new Error("La capacidad no está concedida en la presentación.");
    if (confirmacion !== true) throw new Error("La simulación requiere confirmación explícita.");
    const entrada = payload && typeof payload === "object" ? payload : {};
    const estadoAnterior = structuredClone(estado);
    const secuenciaAnterior = secuencia;
    try {
      aplicar(accion, entrada);
      const datos = await cargar();
      return Object.freeze({ recibo: crearRecibo(accion, entrada), datos });
    } catch (error) {
      estado = estadoAnterior;
      secuencia = secuenciaAnterior;
      throw error;
    }
  }

  return Object.freeze({ modo: "presentacion", cargar, ejecutar });
}
