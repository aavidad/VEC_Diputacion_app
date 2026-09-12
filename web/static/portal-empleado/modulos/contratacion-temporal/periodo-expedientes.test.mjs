import assert from "node:assert/strict";
import test from "node:test";

import { crearAdaptadorHTTPExpedientesContratacionTemporal } from "./adaptador-http-expedientes.js";

const referencia = "expediente:ct:periodo-001";
const resumen = Object.freeze({
  expediente_ref: referencia, numero_visible: "2026/CT-0001", version: 7,
  flujo_ref: "flujo:ct:general", flujo_version: 1, flujo_huella_sha256: "a".repeat(64),
  fase_clave: "nombramiento", estado_clave: "en_curso", centro_ref: "centro:001",
  categoria_ref: "categoria:auxiliar", modalidad_clave: "bolsa",
  creado_en: "2026-09-03T08:00:00Z", actualizado_en: "2026-09-03T09:00:00Z",
});

function detalle(periodo_inicio, periodo_fin) {
  return {
    esquema: "vec.contratacion-temporal.detalle-rrhh.v1", resumen,
    solicitud: { grupo_subgrupo: "A2", motivo_clave: "sustitucion", periodo_inicio, periodo_fin },
    hitos: [{ secuencia: 1, version_expediente: 7, accion_clave: "registrar_propuesta_formalizacion",
      realizada_en: "2026-09-03T09:00:00Z", fase_destino: "nombramiento",
      estado_origen: "en_curso", estado_destino: "en_curso" }],
  };
}

async function proyectar({ locale, inicio, fin }) {
  let solicitudDetalle;
  const entrada = detalle(inicio, fin);
  const adaptador = crearAdaptadorHTTPExpedientesContratacionTemporal({
    locale,
    cliente: {
      async consultarCuadroRRHH() {
        return { esquema: "vec.contratacion-temporal.cuadro-rrhh.v1",
          generada_en: "2026-09-03T09:05:00Z", expedientes: [resumen], hay_mas: false };
      },
      async consultarDetalleRRHH(solicitud) {
        solicitudDetalle = structuredClone(solicitud);
        return entrada;
      },
    },
  });
  await adaptador.listar();
  const expediente = await adaptador.obtener(referencia);
  return {
    periodo: expediente.cabecera.find(({ clave }) => clave === "periodo").valor,
    solicitudDetalle, entrada,
  };
}

test("formatea período civil en es/en con UTC sin desplazar el día ni tocar la solicitud", async () => {
  const inicio = "2026-09-04T00:00:00Z", fin = "2026-12-31T00:00:00Z";
  for (const locale of ["es-ES", "en-GB"]) {
    const resultado = await proyectar({ locale, inicio, fin });
    const formato = new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeZone: "UTC" });
    assert.equal(resultado.periodo, `${formato.format(new Date("2026-09-04T00:00:00Z"))} — ${formato.format(new Date("2026-12-31T00:00:00Z"))}`);
    assert.doesNotMatch(resultado.periodo, /T00:00:00Z/u);
    assert.deepEqual(resultado.solicitudDetalle, { expediente_ref: referencia, version_observada: 7 });
    assert.equal(resultado.entrada.solicitud.periodo_inicio, inicio);
    assert.equal(resultado.entrada.solicitud.periodo_fin, fin);
  }
});

test("respeta 29 de febrero y no convierte una fecha civil imposible en fecha aparente", async () => {
  const bisiesto = await proyectar({ locale: "es-ES", inicio: "2028-02-29T00:00:00Z", fin: "2028-02-29T00:00:00Z" });
  assert.match(bisiesto.periodo, /29.*feb.*2028/iu);
  const imposible = "2026-02-29T00:00:00Z";
  const resultado = await proyectar({ locale: "es-ES", inicio: imposible, fin: "2026-03-01T00:00:00Z" });
  const [inicio] = resultado.periodo.split(" — ");
  assert.equal(inicio, imposible);
});

test("rechaza horas, minutos y segundos inválidos sin normalizarlos", async () => {
  for (const invalido of [
    "2026-09-04T99:00:00Z",
    "2026-09-04T24:00:00Z",
    "2026-09-04T12:60:00Z",
    "2026-09-04T12:00:60Z",
  ]) {
    const resultado = await proyectar({ locale: "es-ES", inicio: invalido, fin: "2026-09-05T00:00:00Z" });
    assert.equal(resultado.periodo.split(" — ")[0], invalido);
  }
});

test("usa es-ES por defecto y conserva la opción explícita en-GB", async () => {
  const inicio = "2026-09-04T00:00:00Z";
  const porDefecto = await proyectar({ inicio, fin: inicio });
  const ingles = await proyectar({ locale: "en-GB", inicio, fin: inicio });
  assert.match(porDefecto.periodo, /sept/iu);
  assert.match(ingles.periodo, /Sept/u);
});

test("no desplaza los años 0000 a 0099 al construir la fecha civil", async () => {
  const inicio = "0099-01-02T00:00:00Z";
  const resultado = await proyectar({ locale: "en-GB", inicio, fin: inicio });
  const fecha = new Date(0);
  fecha.setUTCFullYear(99, 0, 2);
  fecha.setUTCHours(0, 0, 0, 0);
  const esperado = new Intl.DateTimeFormat("en-GB", { dateStyle: "medium", timeZone: "UTC" }).format(fecha);
  assert.equal(resultado.periodo, `${esperado} — ${esperado}`);
  assert.doesNotMatch(resultado.periodo, /1900/u);
});
