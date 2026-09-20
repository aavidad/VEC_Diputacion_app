/** Datos sintéticos aislados para la demostración visual de Méritos y Formación. */
export const DATOS_MERITOS_PRESENTACION = Object.freeze({
  persona: "Antonio López Fernández",
  puesto: "Técnico de Administración General",
  unidad: "Servicio de Administración de Personal",
  titulos: Object.freeze([
    Object.freeze({ nombre: "Grado en Derecho", entidad: "Universidad de Granada", fecha: "2014", estado: "Acreditado en demostración" }),
    Object.freeze({ nombre: "Máster Universitario en Gestión Pública", entidad: "Universidad de Granada", fecha: "2016", estado: "Pendiente de validar" }),
  ]),
  cursos: Object.freeze([
    Object.freeze({ nombre: "Protección de datos en la Administración Pública", horas: "30 h", entidad: "INAP", fecha: "2025", estado: "Acreditado en demostración" }),
    Object.freeze({ nombre: "Contratación pública: actualización normativa", horas: "20 h", entidad: "Centro de Estudios Municipales", fecha: "2024", estado: "Evidencia pendiente" }),
    Object.freeze({ nombre: "Excel aplicado a la gestión de personal", horas: "25 h", entidad: "Diputación de Granada", fecha: "2023", estado: "Homologación pendiente" }),
  ]),
  servicios: Object.freeze([
    Object.freeze({ puesto: "Técnico de Gestión", centro: "Área de Recursos Humanos", periodo: "2019 — 2026", estado: "Pendiente de contraste" }),
    Object.freeze({ puesto: "Administrativo", centro: "Servicio de Administración de Personal", periodo: "2015 — 2019", estado: "Acreditado en demostración" }),
  ]),
  evidencias: Object.freeze([
    Object.freeze({ nombre: "Título universitario", tipo: "Titulación", procedencia: "Aportación de la persona", estado: "Disponible para validar" }),
    Object.freeze({ nombre: "Certificado de curso", tipo: "Formación", procedencia: "Aportación de la persona", estado: "Sin documento conectado" }),
    Object.freeze({ nombre: "Informe de servicios", tipo: "Experiencia", procedencia: "Personal", estado: "Fuente pendiente" }),
  ]),
  formacion: Object.freeze([
    Object.freeze({ nombre: "Gestión de expedientes electrónicos", modalidad: "En línea", plazas: "15 plazas", fecha: "Del 6 al 24 de octubre", estado: "Inscripción pendiente de conexión" }),
    Object.freeze({ nombre: "Lenguaje claro en comunicaciones administrativas", modalidad: "Presencial", plazas: "20 plazas", fecha: "12 y 13 de noviembre", estado: "Inscripción pendiente de conexión" }),
  ]),
  procesos: Object.freeze([
    Object.freeze({ nombre: "Bolsa de Técnico de Administración General", fase: "Preparación de solicitud", aplicacion: "No evaluada" }),
    Object.freeze({ nombre: "Provisión interna · Administración General", fase: "Consulta de requisitos", aplicacion: "Pendiente de bases y autorización" }),
  ]),
});
