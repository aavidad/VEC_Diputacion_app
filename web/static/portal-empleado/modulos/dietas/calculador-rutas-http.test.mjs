import assert from "node:assert/strict";
import test from "node:test";

import {
  ESQUEMA_CALCULO_RUTA_DIETAS,
  ESQUEMA_CATALOGO_RUTAS_DIETAS,
  ESQUEMA_SOLICITUD_RUTA_DIETAS,
} from "./contrato.js";
import { crearCalculadorRutasDietasHTTP } from "./calculador-rutas-http.js";

function punto(code, name, lat, lon) {
  return {
    code,
    name,
    kind: "municipio",
    municipality_code: code,
    municipality_name: name,
    lat,
    lon,
    source: "Catalogo provincial gobernado",
    state: "Vigente",
  };
}

function catalogoBackend({ requeridoAntesLiquidar = true } = {}) {
  return {
    generated_at: "2026-07-19T10:00:00Z",
    province_route_points: [
      punto("18087", "Granada", 37.1773, -3.5986),
      punto("18003", "Albolote", 37.2306, -3.6554),
      punto("18140", "Motril", 36.7447, -3.518),
    ],
    province_route_matrix: {
      matrix_version: "osm-granada-2026-07-19",
      route_points_loaded: 3,
      import_required_before_liquidation: requeridoAntesLiquidar,
    },
  };
}

function respuestaOSRM() {
  return {
    data: {
    code: "Ok",
    engine: "osrm_on_premise",
    route_scope: "Granada provincia + 15 km",
    data_version: "grafo-osm-granada-2026-07-19T04:00:00Z",
    routes: [
      {
        distance: 83_800,
        duration: 4_380,
        legs: [
          { distance: 13_400, duration: 1_080, geometry: { type: "LineString", coordinates: [[-3.5986, 37.1773], [-3.62, 37.205], [-3.6554, 37.2306]] } },
          { distance: 70_400, duration: 3_300, geometry: { type: "LineString", coordinates: [[-3.6554, 37.2306], [-3.59, 36.98], [-3.518, 36.7447]] } },
        ],
        geometry: {
          type: "LineString",
          coordinates: [
            [-3.5986, 37.1773], [-3.62, 37.205], [-3.6554, 37.2306],
            [-3.59, 36.98], [-3.518, 36.7447],
          ],
        },
      },
      {
        distance: 90_000,
        duration: 4_100,
        legs: [
          { distance: 15_000, duration: 900, geometry: { type: "LineString", coordinates: [[-3.5986, 37.1773], [-3.7, 37.05], [-3.6554, 37.2306]] } },
          { distance: 75_000, duration: 3_200, geometry: { type: "LineString", coordinates: [[-3.6554, 37.2306], [-3.64, 36.91], [-3.518, 36.7447]] } },
        ],
        geometry: {
          type: "LineString",
          coordinates: [
            [-3.5986, 37.1773], [-3.7, 37.05], [-3.6554, 37.2306],
            [-3.64, 36.91], [-3.518, 36.7447],
          ],
        },
      },
    ],
    waypoints: [{ location: [-3.5986, 37.1773] }],
    },
  };
}

function respuestaJSON(datos, estado = 200) {
  return new Response(JSON.stringify(datos), {
    status: estado,
    headers: { "Content-Type": "application/json" },
  });
}

function solicitud(paradas = ["18087", "18003", "18140"], alternativas = 2) {
  return {
    esquema: ESQUEMA_SOLICITUD_RUTA_DIETAS,
    paradas,
    alternativas,
  };
}

test("proyecta catalogo sin coordenadas y mapea OSRM interno al contrato no liquidable", async () => {
  const llamadas = [];
  const fetchImpl = async (ruta, opciones) => {
    llamadas.push({ ruta, opciones });
    if (ruta === "/api/vec/dietas/route-catalog") return respuestaJSON(catalogoBackend());
    if (ruta === "/api/vec/dietas/road-route") return respuestaJSON(respuestaOSRM());
    throw new Error("ruta inesperada");
  };
  const adaptador = crearCalculadorRutasDietasHTTP({ fetchImpl });

  const catalogo = await adaptador.obtenerCatalogo();
  assert.equal(catalogo.esquema, ESQUEMA_CATALOGO_RUTAS_DIETAS);
  assert.equal(catalogo.demostracion, false);
  assert.equal(catalogo.completo, false);
  assert.deepEqual(Object.keys(catalogo.puntos[0]).sort(), [
    "codigo", "municipio_codigo", "municipio_nombre", "nombre", "tipo",
  ]);
  assert.doesNotMatch(JSON.stringify(catalogo), /latitud|longitud|"lat"|"lon"/u);

  const calculo = await adaptador.calcular(solicitud());
  assert.equal(calculo.esquema, ESQUEMA_CALCULO_RUTA_DIETAS);
  assert.equal(calculo.demostracion, false);
  assert.equal(calculo.liquidable, false);
  assert.equal(calculo.motor, "osrm_interno");
  assert.equal(calculo.version_grafo, "grafo-osm-granada-2026-07-19T04:00:00Z");
  assert.match(calculo.referencia, /^RUTA-OSRM-[A-F0-9]{32}$/u);
  assert.equal(calculo.alternativas.length, 2);
  assert.equal(calculo.alternativas[0].recomendada, true);
  assert.equal(calculo.alternativas[1].recomendada, false);
  assert.equal(calculo.alternativas[0].kilometros, 83.8);
  assert.equal(calculo.alternativas[0].tramos[0].origen_nombre, "Granada");
  assert.deepEqual(calculo.alternativas[0].geometria.trazado[0], [37.1773, -3.5986]);
  assert.equal(Object.hasOwn(calculo, "waypoints"), false);

  assert.equal(llamadas.length, 2);
  for (const llamada of llamadas) {
    assert.match(llamada.ruta, /^\/api\/vec\/[a-z/-]+$/u);
    assert.doesNotMatch(llamada.ruta, /^[a-z][a-z0-9+.-]*:/iu);
    assert.equal(llamada.opciones.credentials, "same-origin");
    assert.equal(llamada.opciones.mode, "same-origin");
    assert.equal(llamada.opciones.redirect, "error");
    assert.equal(llamada.opciones.cache, "no-store");
    assert.equal(llamada.opciones.referrer, "");
    assert.equal(llamada.opciones.referrerPolicy, "no-referrer");
    assert.equal(llamada.opciones.signal instanceof AbortSignal, true);
  }
  const cuerpo = JSON.parse(llamadas[1].opciones.body);
  assert.deepEqual(cuerpo, {
    coordinates: [
      { lat: 37.1773, lon: -3.5986 },
      { lat: 37.2306, lon: -3.6554 },
      { lat: 36.7447, lon: -3.518 },
    ],
    alternatives: 2,
  });
  assert.doesNotMatch(llamadas[1].opciones.body, /persona_ref|cuenta_ref|sesion_ref/u);
});

test("impone en todo fetch productivo la política same-origin sin cookies ni referente", async () => {
  const llamadas = [];
  const fetchImpl = async (ruta, opciones) => {
    llamadas.push({ ruta, opciones });
    return respuestaJSON(
      ruta === "/api/vec/dietas/route-catalog" ? catalogoBackend() : respuestaOSRM(),
    );
  };
  const adaptador = crearCalculadorRutasDietasHTTP({ fetchImpl });

  await adaptador.obtenerCatalogo();
  await adaptador.calcular(solicitud());

  assert.equal(llamadas.length, 2);
  llamadas.forEach(({ ruta, opciones }) => {
    assert.match(ruta, /^\/api\/vec\//u);
    assert.equal(opciones.credentials, "same-origin");
    assert.equal(opciones.mode, "same-origin");
    assert.equal(opciones.redirect, "error");
    assert.equal(opciones.cache, "no-store");
    assert.equal(opciones.referrer, "");
    assert.equal(opciones.referrerPolicy, "no-referrer");
    assert.equal(Object.hasOwn(opciones.headers, "Cookie"), false);
    assert.equal(Object.hasOwn(opciones.headers, "cookie"), false);
  });
});

test("solo se compone con cliente HTTP inyectado", () => {
  let llamadas = 0;
  const fetchImpl = async () => {
    llamadas += 1;
    return respuestaJSON(catalogoBackend());
  };
  assert.throws(() => crearCalculadorRutasDietasHTTP(), /conector de identidad sin cookies/u);
  assert.throws(() => crearCalculadorRutasDietasHTTP({ fetchImpl, capacidad: "dietas.ruta.read" }), /opciones/u);
  assert.equal(llamadas, 0);
});

test("falla cerrada si route-catalog no ofrece una proyeccion gobernada coherente", async () => {
  const incoherente = catalogoBackend();
  incoherente.province_route_matrix.route_points_loaded = 2;
  const adaptador = crearCalculadorRutasDietasHTTP({ fetchImpl: async () => respuestaJSON(incoherente) });
  await assert.rejects(adaptador.obtenerCatalogo(), /catalogo y su manifiesto/u);

  const bloqueado = crearCalculadorRutasDietasHTTP({ fetchImpl: async () => respuestaJSON({ error: "superficie no disponible" }, 503) });
  await assert.rejects(bloqueado.obtenerCatalogo(), /HTTP 503/u);
});

test("rechaza cuerpos que no coinciden con Content-Length antes de proyectar datos", async () => {
  const cuerpo = JSON.stringify(catalogoBackend());
  const adaptador = crearCalculadorRutasDietasHTTP({ fetchImpl: async () => new Response(cuerpo, {
      status: 200,
      headers: {
        "Content-Type": "application/json",
        "Content-Length": String(new TextEncoder().encode(cuerpo).byteLength + 1),
      },
    }) });
  await assert.rejects(adaptador.obtenerCatalogo(), /longitud declarada/u);
});

test("acepta JSON UTF-8 canonico con mayusculas y espacios en Content-Type", async () => {
  const adaptador = crearCalculadorRutasDietasHTTP({ fetchImpl: async () => new Response(JSON.stringify(catalogoBackend()), {
      status: 200,
      headers: { "Content-Type": "Application/JSON ; Charset = UTF-8" },
    }) });
  assert.equal((await adaptador.obtenerCatalogo()).puntos.length, 3);
});

test("rechaza solicitudes fuera de 12 paradas y no alcanza el endpoint OSRM", async () => {
  let llamadasRuta = 0;
  const adaptador = crearCalculadorRutasDietasHTTP({
    fetchImpl: async (ruta) => {
      if (ruta === "/api/vec/dietas/route-catalog") return respuestaJSON(catalogoBackend());
      llamadasRuta += 1;
      return respuestaJSON(respuestaOSRM());
    },
  });
  await adaptador.obtenerCatalogo();
  await assert.rejects(
    adaptador.calcular(solicitud(Array.from({ length: 13 }, () => "18087"), 1)),
    /solicitud de calculo de ruta no valida/u,
  );
  await assert.rejects(adaptador.calcular(solicitud(["18087", "18003"], 4)), /alternativas/u);
  assert.equal(llamadasRuta, 0);
});

test("no devuelve resultados parciales si OSRM omite version o tramos", async () => {
  const sinVersion = respuestaOSRM();
  delete sinVersion.data.data_version;
  let respuestaRuta = sinVersion;
  const adaptador = crearCalculadorRutasDietasHTTP({
    fetchImpl: async (ruta) => respuestaJSON(
      ruta === "/api/vec/dietas/route-catalog" ? catalogoBackend() : respuestaRuta,
    ),
  });
  await adaptador.obtenerCatalogo();
  await assert.rejects(adaptador.calcular(solicitud()), /version del grafo/u);

  respuestaRuta = respuestaOSRM();
  delete respuestaRuta.data.routes[0].legs;
  await assert.rejects(adaptador.calcular(solicitud()), /tramos OSRM/u);
});

test("conserva una sola version de catalogo durante un calculo concurrente", async () => {
  let consultasCatalogo = 0;
  let resolverRuta;
  let cuerpoRuta;
  const respuestaRutaPendiente = new Promise((resolve) => { resolverRuta = resolve; });
  const fetchImpl = async (ruta, opciones) => {
    if (ruta === "/api/vec/dietas/road-route") {
      cuerpoRuta = opciones.body;
      return respuestaRutaPendiente;
    }
    consultasCatalogo += 1;
    const datos = catalogoBackend();
    if (consultasCatalogo === 2) {
      datos.province_route_points[0].name = "Granada actualizada";
      datos.province_route_points[0].municipality_name = "Granada actualizada";
      datos.province_route_points[0].lat = 37.18;
    }
    return respuestaJSON(datos);
  };
  const adaptador = crearCalculadorRutasDietasHTTP({ fetchImpl });
  await adaptador.obtenerCatalogo();
  const pendiente = adaptador.calcular(solicitud());
  await adaptador.obtenerCatalogo();
  resolverRuta(respuestaJSON(respuestaOSRM()));
  const calculo = await pendiente;

  assert.equal(calculo.alternativas[0].tramos[0].origen_nombre, "Granada");
  assert.equal(JSON.parse(cuerpoRuta).coordinates[0].lat, 37.1773);
});

test("propaga cancelacion externa y aplica un timeout propio", async () => {
  // El corte no depende de que el cliente inyectado respete AbortSignal.
  const fetchBloqueado = async () => new Promise(() => {});
  const cancelable = crearCalculadorRutasDietasHTTP({ fetchImpl: fetchBloqueado });
  const controlador = new AbortController();
  const pendiente = cancelable.obtenerCatalogo({ signal: controlador.signal });
  controlador.abort();
  await assert.rejects(pendiente, /fue cancelada/u);

  const conTimeout = crearCalculadorRutasDietasHTTP({
    fetchImpl: fetchBloqueado,
    tiempoEsperaMs: 10,
  });
  await assert.rejects(conTimeout.obtenerCatalogo(), /agoto su tiempo/u);
});
