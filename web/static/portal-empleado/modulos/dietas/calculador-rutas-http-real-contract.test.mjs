import assert from "node:assert/strict";
import test from "node:test";

import {
  ESQUEMA_SOLICITUD_RUTA_DIETAS,
} from "./contrato.js";
import { crearCalculadorRutasDietasHTTP } from "./calculador-rutas-http.js";

function respuestaJSON(datos) {
  return new Response(JSON.stringify(datos), {
    status: 200, headers: { "Content-Type": "application/json; charset=utf-8" },
  });
}

function punto(code, name, lat, lon) {
  return { code, name, kind: "municipio", municipality_code: code, municipality_name: name,
    lat, lon, source: "Catálogo gobernado", state: "Vigente" };
}

test("usa los contratos reales route-catalog y road-route sin contexto del navegador", async () => {
  const llamadas = [];
  const cliente = crearCalculadorRutasDietasHTTP({
    fetchImpl: async (ruta, opciones) => {
      llamadas.push({ ruta, opciones });
      if (ruta === "/api/vec/dietas/route-catalog") return respuestaJSON({
        province_route_points: [punto("18087", "Granada", 37.1773, -3.5986), punto("18140", "Motril", 36.7447, -3.518)],
        province_route_matrix: { matrix_version: "granada-v1", route_points_loaded: 2, import_required_before_liquidation: true },
      });
      return respuestaJSON({ data: {
        code: "Ok", engine: "osrm_on_premise", route_scope: "Granada provincia + 15 km", data_version: "granada-v1",
        routes: [{ distance: 70_400, duration: 3_300,
          legs: [{ distance: 70_400, duration: 3_300, geometry: { type: "LineString", coordinates: [[-3.5986, 37.1773], [-3.518, 36.7447]] } }],
          geometry: { type: "LineString", coordinates: [[-3.5986, 37.1773], [-3.518, 36.7447]] } }],
      } });
    },
  });
  await cliente.obtenerCatalogo();
  const calculo = await cliente.calcular({
    esquema: ESQUEMA_SOLICITUD_RUTA_DIETAS, paradas: ["18087", "18140"], alternativas: 1,
  });
  assert.deepEqual(llamadas.map(({ ruta }) => ruta), [
    "/api/vec/dietas/route-catalog", "/api/vec/dietas/road-route",
  ]);
  assert.equal(llamadas[0].opciones.method, "GET");
  assert.equal(llamadas[1].opciones.method, "POST");
  assert.deepEqual(JSON.parse(llamadas[1].opciones.body), {
    coordinates: [{ lat: 37.1773, lon: -3.5986 }, { lat: 36.7447, lon: -3.518 }], alternatives: 1,
  });
  assert.equal(calculo.motor, "osrm_interno");
  assert.equal(calculo.demostracion, false);
});
