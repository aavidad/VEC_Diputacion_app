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
