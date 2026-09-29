/**
 * Escenario efímero de presentación para Personal.
 *
 * No consulta ni conserva identidad, relación jurídica, nóminas o pagos reales.
 * Este fixture solo permite comprobar la jerarquía visual mientras no exista un
 * adaptador autorizado de Personal. No calcula antigüedad, trienios ni importes.
 */
import { cargarTextos } from "../../../comun/textos.js";

const REFERENCIA_TITULAR_DEMO = "DEMO-PERSONAL-TITULAR-ACTIVO";

const TEXTOS_DEMO = (await cargarTextos("personal")).seccion("demo");

const DATOS_DEMO = Object.freeze({
  esquema: "vec.personal.panel.v1",
  origen: Object.freeze({
    demostracion: true,
    efimero: true,
    adaptador: "presentacion_volatil",
    efectos_reales: false,
    aviso: TEXTOS_DEMO.origen_aviso,
  }),
  titular_ref: REFERENCIA_TITULAR_DEMO,
  actualizado_en: "2026-09-20T09:00:00Z",
  relacion_actual: Object.freeze({
    referencia: "DEMO-REL-2026-01",
    situacion: TEXTOS_DEMO.relacion_situacion,
    puesto: TEXTOS_DEMO.relacion_puesto,
    unidad: TEXTOS_DEMO.relacion_unidad,
    grupo: TEXTOS_DEMO.relacion_grupo,
    observacion: TEXTOS_DEMO.relacion_observacion,
  }),
  servicios: Object.freeze([
    Object.freeze({
      referencia: "DEMO-SERV-2024-01",
      desde: "2024-01-15",
      hasta: "2024-12-31",
      descripcion: TEXTOS_DEMO.servicio_2024_descripcion,
      computable: false,
      observacion: TEXTOS_DEMO.servicio_2024_observacion,
    }),
    Object.freeze({
      referencia: "DEMO-SERV-2025-01",
      desde: "2025-01-01",
      hasta: "2025-12-31",
      descripcion: TEXTOS_DEMO.servicio_2025_descripcion,
      computable: false,
      observacion: TEXTOS_DEMO.servicio_2025_observacion,
    }),
  ]),
  formacion: Object.freeze([
    Object.freeze({
      referencia: "DEMO-FOR-2025-04",
      actividad: TEXTOS_DEMO.formacion_curso_actividad,
      periodo: TEXTOS_DEMO.formacion_curso_periodo,
      estado: TEXTOS_DEMO.formacion_curso_estado,
      acreditacion: TEXTOS_DEMO.formacion_curso_acreditacion,
    }),
    Object.freeze({
      referencia: "DEMO-FOR-2024-11",
      actividad: TEXTOS_DEMO.formacion_taller_actividad,
      periodo: TEXTOS_DEMO.formacion_taller_periodo,
      estado: TEXTOS_DEMO.formacion_taller_estado,
      acreditacion: TEXTOS_DEMO.formacion_taller_acreditacion,
    }),
  ]),
  nominas: Object.freeze([
    Object.freeze({
      referencia: "DEMO-NOM-2026-08",
      periodo: TEXTOS_DEMO.nomina_agosto_periodo,
      estado: TEXTOS_DEMO.nomina_agosto_estado,
      importes_disponibles: false,
      recibo_disponible: false,
      observacion: TEXTOS_DEMO.nomina_agosto_observacion,
    }),
    Object.freeze({
      referencia: "DEMO-NOM-2026-07",
      periodo: TEXTOS_DEMO.nomina_julio_periodo,
      estado: TEXTOS_DEMO.nomina_julio_estado,
      importes_disponibles: false,
      recibo_disponible: false,
      observacion: TEXTOS_DEMO.nomina_julio_observacion,
    }),
  ]),
  dietas_cobradas: Object.freeze([
    Object.freeze({
      referencia: "DEMO-DIETA-2026-02",
      periodo: TEXTOS_DEMO.dieta_periodo,
      estado: TEXTOS_DEMO.dieta_estado,
      importe_disponible: false,
      pago_acreditado: false,
      observacion: TEXTOS_DEMO.dieta_observacion,
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
