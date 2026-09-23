import test from "node:test";
import assert from "node:assert/strict";

import {
  validarRespuestaEstadisticas,
  validarSerieEstadisticas,
} from "./contrato-estadisticas.js";

function serie(inicio) {
  return { inicio, altas: 0, llamamientos: 0, formalizaciones: 0, cierres: 0, incidencias: 0 };
}

function respuesta({ periodo = "mensual", desde = "2026-01-15", hasta = "2026-02-20", inicios = ["2026-01-01", "2026-02-01"] } = {}) {
  return {
    data: {
      esquema: "vec.ct.estadisticas.v1",
      periodo,
      desde,
      hasta,
      series: inicios.map(serie),
      totales: { altas: 0, llamamientos: 0, formalizaciones: 0, cierres: 0, incidencias: 0 },
    },
  };
}

test("rechaza días inexistentes y conserva el 29 de febrero bisiesto", () => {
  for (const campo of ["desde", "hasta"]) {
    assert.throws(() => validarRespuestaEstadisticas(respuesta({ [campo]: "2026-02-30" })), /fecha válida/);
  }
  assert.throws(() => validarSerieEstadisticas(serie("2026-02-30")), /fecha válida/);
  assert.throws(() => validarSerieEstadisticas(serie("2026-02-29")), /fecha válida/);
  assert.equal(validarSerieEstadisticas(serie("2024-02-29")).inicio, "2024-02-29");
  assert.equal(
    validarRespuestaEstadisticas(respuesta({ periodo: "semanal", desde: "2024-02-29", hasta: "2024-02-29", inicios: ["2024-02-26"] })).series[0].inicio,
    "2024-02-26",
  );
});

test("rechaza series invertidas y duplicadas antes de llegar a tabla, gráfico o CSV", () => {
  assert.throws(() => validarRespuestaEstadisticas(respuesta({ inicios: ["2026-02-01", "2026-01-01"] })), /orden cronológico estricto/);
  assert.throws(() => validarRespuestaEstadisticas(respuesta({ inicios: ["2026-01-01", "2026-01-01"] })), /orden cronológico estricto/);
});

test("aplica el rango de inicios de periodo del servidor", () => {
  assert.equal(validarRespuestaEstadisticas(respuesta()).series[0].inicio, "2026-01-01");
  assert.throws(() => validarRespuestaEstadisticas(respuesta({ inicios: ["2025-12-01"] })), /fuera del rango/);
  assert.throws(() => validarRespuestaEstadisticas(respuesta({ inicios: ["2026-03-01"] })), /fuera del rango/);
  assert.deepEqual(
    validarRespuestaEstadisticas(respuesta({ periodo: "semanal", desde: "2026-02-04", hasta: "2026-02-10", inicios: ["2026-02-02", "2026-02-09"] })).series.map((punto) => punto.inicio),
    ["2026-02-02", "2026-02-09"],
  );
  assert.equal(
    validarRespuestaEstadisticas(respuesta({ periodo: "anual", desde: "2026-02-01", hasta: "2026-11-30", inicios: ["2026-01-01"] })).series[0].inicio,
    "2026-01-01",
  );
});
