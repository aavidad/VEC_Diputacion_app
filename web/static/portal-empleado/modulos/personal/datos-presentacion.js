/**
 * Escenario efímero de presentación para Personal.
 *
 * No consulta ni conserva identidad, relación jurídica, nóminas o pagos reales.
 * Este fixture solo permite comprobar la jerarquía visual mientras no exista un
 * adaptador autorizado de Personal. No calcula antigüedad, trienios ni importes.
 */

const REFERENCIA_TITULAR_DEMO = "DEMO-PERSONAL-TITULAR-ACTIVO";

const DATOS_DEMO = Object.freeze({
  esquema: "vec.personal.panel.v1",
  origen: Object.freeze({
    demostracion: true,
    efimero: true,
    adaptador: "presentacion_volatil",
    efectos_reales: false,
    aviso: "Datos sintéticos de presentación; no acreditan relación, servicio, nómina, dieta ni pago.",
  }),
  titular_ref: REFERENCIA_TITULAR_DEMO,
  actualizado_en: "2026-09-20T09:00:00Z",
  relacion_actual: Object.freeze({
    referencia: "DEMO-REL-2026-01",
    situacion: "Escenario informativo DEMO",
    puesto: "Puesto de ejemplo · DEMO",
    unidad: "Unidad de ejemplo · DEMO",
    grupo: "Grupo de ejemplo · DEMO",
    observacion: "Referencia visual sin eficacia jurídica ni acto de personal.",
  }),
  servicios: Object.freeze([
    Object.freeze({
      referencia: "DEMO-SERV-2024-01",
      desde: "2024-01-15",
      hasta: "2024-12-31",
      descripcion: "Periodo informativo de ejemplo · DEMO",
      computable: false,
      observacion: "No se calcula antigüedad, trienios ni ningún derecho.",
    }),
    Object.freeze({
      referencia: "DEMO-SERV-2025-01",
      desde: "2025-01-01",
      hasta: "2025-12-31",
      descripcion: "Periodo informativo de ejemplo · DEMO",
      computable: false,
      observacion: "No se calcula antigüedad, trienios ni derechos; no sustituye certificación.",
    }),
  ]),
  formacion: Object.freeze([
    Object.freeze({
      referencia: "DEMO-FOR-2025-04",
      actividad: "Curso de ejemplo sobre atención administrativa · DEMO",
      periodo: "Mayo de 2025",
      estado: "Informativo",
      acreditacion: "No disponible en DEMO",
    }),
    Object.freeze({
      referencia: "DEMO-FOR-2024-11",
      actividad: "Taller de ejemplo de herramientas digitales · DEMO",
      periodo: "Noviembre de 2024",
      estado: "Informativo",
      acreditacion: "No disponible en DEMO",
    }),
  ]),
  nominas: Object.freeze([
    Object.freeze({
      referencia: "DEMO-NOM-2026-08",
      periodo: "Agosto de 2026 · DEMO",
      estado: "Referencia de presentación",
      importes_disponibles: false,
      recibo_disponible: false,
      observacion: "No contiene importes, bases, retenciones ni recibo de nómina.",
    }),
    Object.freeze({
      referencia: "DEMO-NOM-2026-07",
      periodo: "Julio de 2026 · DEMO",
      estado: "Referencia de presentación",
      importes_disponibles: false,
      recibo_disponible: false,
      observacion: "No contiene importes, bases, retenciones ni recibo de nómina.",
    }),
  ]),
  dietas_cobradas: Object.freeze([
    Object.freeze({
      referencia: "DEMO-DIETA-2026-02",
      periodo: "Junio de 2026 · DEMO",
      estado: "Referencia sin pago real",
      importe_disponible: false,
      pago_acreditado: false,
      observacion: "No acredita liquidación, inclusión en nómina ni pago.",
    }),
  ]),
});

function copiarDatosDemo() {
  return structuredClone(DATOS_DEMO);
}

/**
 * Devuelve datos sintéticos y efímeros para una demostración local de la vista.
 * No acepta identidad, capacidades ni datos procedentes del navegador.
 */
export function crearDatosPersonalPresentacion() {
  return copiarDatosDemo();
}

export { REFERENCIA_TITULAR_DEMO };
