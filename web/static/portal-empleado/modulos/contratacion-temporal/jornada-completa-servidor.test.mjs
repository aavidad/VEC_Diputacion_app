import assert from "node:assert/strict";
import test from "node:test";

import { crearAdaptadorHTTPExpedientesContratacionTemporal } from "./adaptador-http-expedientes.js";
import { validarConfiguracionAnalisis } from "./contrato-analisis.js";
import { diezmilesimasDesdeHorasMinutos, horasMinutosDesdeDiezmilesimas, montarFormularioAnalisisRRHH } from "./formulario-analisis.js";

// La jornada completa de referencia la sirve el servidor (regla c07 del
// catálogo de reglas); la web no conserva ningún valor propio.

const HUELLA = "a".repeat(64);
const CLAVE = "123e4567-e89b-42d3-a456-426614174000";
const FormDataOriginal = globalThis.FormData;
globalThis.FormData = class {
  constructor(formulario) { this.valores = formulario.valores; }
  get(nombre) { return this.valores[nombre] ?? null; }
};
test.after(() => { globalThis.FormData = FormDataOriginal; });

function catalogos(minutos) {
  return {
    modalidades: ["sustitucion", "vacante", "acumulacion_tareas", "programa", "relevo"]
      .map((clave) => ({ clave, etiqueta: clave })),
    categorias: [{ referencia: "categoria:rrhh:001", etiqueta: "Categoría", grupos_subgrupos: [
      { clave: "C2", etiqueta: "C2" },
    ] }],
    causas: [{ clave: "necesidad_temporal", etiqueta: "Necesidad temporal" }],
    entradas_rc: [{ referencia: "entrada-rc:opaca:001", huella_sha256: HUELLA, etiqueta: "RC" }],
    motivos_rectificacion: [],
    jornada_completa_minutos_semanales: minutos,
  };
}

function configuracion(minutos) {
  const { jornada_completa_minutos_semanales: _omitida, ...resto } = catalogos(minutos);
  return {
    esquema: "vec.contratacion_temporal.configuracion_analisis.v1",
    artefacto_ref: "artefacto:analisis:desarrollo:v1",
    ...resto,
    ...(minutos === undefined ? {} : { jornada_completa_minutos_semanales: minutos }),
  };
}

function montar(minutos) {
  const eventos = new Map();
  const solicitudes = [];
  const raiz = {
    innerHTML: "",
    addEventListener(tipo, manejador) { eventos.set(tipo, manejador); },
    removeEventListener(tipo) { eventos.delete(tipo); },
    contains() { return true; },
    querySelector() { return { focus() {}, scrollIntoView() {} }; },
    replaceChildren() { this.innerHTML = ""; },
  };
  const desmontar = montarFormularioAnalisisRRHH({
    raiz,
    contexto: {
      operacion: "registrar", expediente_ref: "expediente:opaco:001",
      version_esperada: 1, artefacto_ref: "artefacto:opaco:001",
    },
    catalogos: catalogos(minutos),
    generarClaveIdempotencia: () => CLAVE,
    cliente: {
      registrarAnalisis(solicitud) {
        solicitudes.push(solicitud);
        return Promise.resolve({
          esquema: "vec.contratacion-temporal.recibo-analisis-rrhh.v1",
          operacion: "registrar", expediente_ref: "expediente:opaco:001",
          version_resultante: 2, recibo_ref: "recibo:opaco:analisis:001",
          confirmada_en: "2026-09-25T09:30:00Z",
        });
      },
    },
  });
  return {
    raiz, solicitudes, desmontar,
    enviar(jornada_horas, jornada_minutos) {
      const formulario = {
        valores: {
          modalidad_clave: "sustitucion", categoria_ref: "categoria:rrhh:001",
          grupo_subgrupo: "C2", causa_clave: "necesidad_temporal",
          inicio: "2026-10-01", fin: "2026-12-31", jornada_horas, jornada_minutos,
          entrada_rc_referencia: "entrada-rc:opaca:001",
        },
        closest(selector) { return selector === "[data-ct-analisis-form]" ? this : null; },
      };
      return eventos.get("submit")({ target: formulario, preventDefault() {} });
    },
  };
}

test("la configuración del análisis exige la jornada completa servida", () => {
  assert.equal(validarConfiguracionAnalisis(configuracion(2100)).jornada_completa_minutos_semanales, 2100);
  for (const minutos of [undefined, 0, 10_081, 37.5, "2250", null]) {
    assert.throws(() => validarConfiguracionAnalisis(configuracion(minutos)), TypeError, String(minutos));
  }
});

test("las conversiones usan la jornada completa recibida", () => {
  assert.equal(diezmilesimasDesdeHorasMinutos("35", "0", 2100), "10000");
  assert.equal(diezmilesimasDesdeHorasMinutos("35", "01", 2100), "");
  assert.equal(diezmilesimasDesdeHorasMinutos("17", "30", 2100), "5000");
  assert.deepEqual(horasMinutosDesdeDiezmilesimas("5000", 2100), { horas: "17", minutos: "30" });
  assert.throws(() => diezmilesimasDesdeHorasMinutos("7", "0"), TypeError);
  assert.throws(() => horasMinutosDesdeDiezmilesimas("5000"), TypeError);
});

test("el formulario muestra y valida la jornada completa del servidor", async () => {
  const vista = montar(2100);
  assert.match(vista.raiz.innerHTML, /Jornada completa: 35 h 0 min\./u);
  assert.match(vista.raiz.innerHTML, /name="jornada_horas"[^>]*max="35"/u);
  await vista.enviar("35", "30");
  assert.equal(vista.solicitudes.length, 0);
  assert.match(vista.raiz.innerHTML, /entre 1 minuto y 35 h 0 min\./u);
  await vista.enviar("35", "0");
  assert.equal(vista.solicitudes[0].analisis.porcentaje_jornada, 10_000);
  vista.desmontar();
  assert.throws(() => montar(undefined), TypeError);
});

function clienteDetalle() {
  const resumen = {
    expediente_ref: "expediente:ct:001", numero_visible: "2026/CT-0001", version: 2,
    flujo_ref: "flujo:ct:general", flujo_version: 1, flujo_huella_sha256: HUELLA,
    fase_clave: "analisis", estado_clave: "en_curso", centro_ref: "centro:001",
    categoria_ref: "categoria:auxiliar", modalidad_clave: "bolsa", unidad_ref: "unidad:rrhh",
    creado_en: "2026-09-03T08:00:00Z", actualizado_en: "2026-09-03T09:00:00Z",
  };
  return {
    async consultarCuadroRRHH() {
      return { esquema: "vec.contratacion-temporal.cuadro-rrhh.v1", generada_en: "2026-09-03T09:05:00Z", expedientes: [resumen], hay_mas: false };
    },
    async consultarDetalleRRHH() {
      return {
        esquema: "vec.contratacion-temporal.detalle-rrhh.v1", resumen,
        solicitud: { grupo_subgrupo: "C2", motivo_clave: "sustitucion", periodo_inicio: "2026-09-04T00:00:00Z", periodo_fin: "2026-12-31T00:00:00Z" },
        analisis: {
          modalidad_clave: "bolsa", categoria_ref: "categoria:auxiliar", causa_clave: "sustitucion",
          periodo_inicio: "2026-09-04T00:00:00Z", periodo_fin: "2026-12-31T00:00:00Z",
          porcentaje_jornada: 5_000, resultado_rc: "no_requerida",
        },
        hitos: [],
      };
    },
  };
}

test("el detalle convierte la jornada con la referencia servida o solo muestra el porcentaje", async () => {
  let minutos = 2100;
  const adaptador = crearAdaptadorHTTPExpedientesContratacionTemporal({
    cliente: clienteDetalle(), obtenerJornadaCompleta: () => minutos,
  });
  await adaptador.listar({ filtros: { texto: "", estado: "", fase: "" } });
  const jornada = async () => (await adaptador.obtener("expediente:ct:001"))
    .cabecera.find(({ clave }) => clave === "jornada").valor;
  assert.match(await jornada(), /^17 h 30 min de media semanal \(50\s%\)$/u);
  minutos = null;
  assert.match(await jornada(), /^50\s% de la jornada completa$/u);
  assert.throws(() => crearAdaptadorHTTPExpedientesContratacionTemporal({
    cliente: clienteDetalle(), obtenerJornadaCompleta: 2250,
  }), TypeError);
});
