import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import {
  CAPACIDAD_CONSULTAR_RUTA,
  CODIGO_ERROR_SERVICIO_RUTAS_DIETAS,
  ErrorServicioRutasDietas,
  ESQUEMA_SOLICITUD_RUTA_DIETAS,
} from "./contrato.js";
import { crearCalculadorRutasDietasPresentacionOSRM } from "./calculador-rutas-presentacion-osrm.js";
import {
  obtenerCatalogoRutasProvincial,
  resolverPuntosRutasProvincial,
} from "./catalogo-rutas-provincial.js";
import {
  ESQUEMA_CONTEXTO_ACTOR_FRONTEND,
  validarYCongelarContextoActor,
} from "../../identidad/contexto-actor.js";

const VERSION_GRAFO = "granada-buffer-osrm-v1-53aba0ad43c4";

function contexto(demostracion = true) {
  return validarYCongelarContextoActor({
    esquema: ESQUEMA_CONTEXTO_ACTOR_FRONTEND,
    revision: 1,
    demostracion,
    persona_ref: demostracion
      ? "per_persona_demostracion_osrm_000001"
      : "per_persona_productiva_osrm_000001",
    cuenta_ref: demostracion
      ? "cta_cuenta_demostracion_osrm_000001"
      : "cta_cuenta_productiva_osrm_000001",
    perfil_ref: demostracion
      ? "prf_perfil_demostracion_osrm_000001"
      : "prf_perfil_productivo_osrm_000001",
    actor: {
      actor_ref: demostracion ? "DEMO-PERFIL-DIETAS-OSRM-01" : "act_actor_productivo_osrm_000001",
      nombre_visible: demostracion ? "Empleado DEMO" : "Empleado autorizado",
      iniciales: "ED",
    },
    rol: { clave: "personal_interno", etiqueta: "Personal interno" },
    ambito: {
      clase: "personal_interno",
      organizacion_ref: "org_diputacion_granada_osrm_000001",
      unidad_ref: "uni_unidad_dietas_osrm_000001",
      modulos: ["dietas"],
    },
    autenticacion: {
      sesion_ref: "ses_sesion_dietas_osrm_000001",
      metodo: demostracion ? "demo" : "kerberos_ad",
      garantia: demostracion ? "bajo" : "alto",
    },
    resuelto_en: "2026-07-19T10:00:00.000Z",
  });
}

function solicitud(paradas = ["18087", "18140"], alternativas = 1) {
  return { esquema: ESQUEMA_SOLICITUD_RUTA_DIETAS, paradas, alternativas };
}

function respuestaOSRM({
  version = VERSION_GRAFO,
  geometria = [[-3.59869101, 37.17428891], [-3.56, 36.95], [-3.52045559, 36.74535308]],
} = {}) {
  return {
    code: "Ok",
    engine: "osrm_on_premise",
    route_scope: "Granada provincia + 15 km",
    graph_version: version,
    routes: [{
      distance: 70_400,
      duration: 3_300,
      legs: [{ distance: 70_400, duration: 3_300 }],
      geometry: { type: "LineString", coordinates: geometria },
    }],
    waypoints: [],
  };
}

function respuestaJSON(datos, opciones = {}) {
  const contenido = JSON.stringify(datos);
  return new Response(contenido, {
    status: opciones.estado ?? 200,
    headers: {
      "Content-Type": opciones.tipo ?? "application/json; charset=UTF-8",
      ...(opciones.longitud === true ? { "Content-Length": String(Buffer.byteLength(contenido)) } : {}),
    },
  });
}

function crearAdaptador(fetchImpl, opciones = {}) {
  return crearCalculadorRutasDietasPresentacionOSRM({
    contextoActor: contexto(),
    capacidades: [CAPACIDAD_CONSULTAR_RUTA],
    fetchImpl,
    ...opciones,
  });
}

async function exigirErrorCerrado(promesa) {
  await assert.rejects(promesa, (error) => {
    assert.ok(error instanceof ErrorServicioRutasDietas);
    assert.equal(error.codigo, CODIGO_ERROR_SERVICIO_RUTAS_DIETAS);
    assert.equal(error.message, "No se pudo calcular la ruta con el servicio interno.");
    return true;
  });
}

test("consulta el mediador POST fijo y proyecta OSRM real como DEMO sin efectos", async () => {
  const llamadas = [];
  const adaptador = crearAdaptador(async (ruta, opciones) => {
    llamadas.push({ ruta, opciones });
    return respuestaJSON(respuestaOSRM(), { longitud: true });
  });

  const catalogo = adaptador.obtenerCatalogo();
  assert.equal(catalogo.demostracion, true);
  assert.equal(catalogo.puntos.length, 175);
  assert.doesNotMatch(JSON.stringify(catalogo), /latitud|longitud|coordinates/iu);

  const calculo = await adaptador.calcular(solicitud());
  assert.equal(calculo.demostracion, true);
  assert.equal(calculo.liquidable, false);
  assert.equal(calculo.motor, "osrm_interno");
  assert.equal(calculo.version_grafo, VERSION_GRAFO);
  assert.match(calculo.referencia, /^OSRM-[A-Z2-7]{52}$/u);
  assert.equal(calculo.alternativas[0].geometria.origen, "osrm_interno");
  assert.deepEqual(calculo.alternativas[0].geometria.trazado[0], [37.17428891, -3.59869101]);
  assert.equal(Object.isFrozen(calculo), true);

  assert.equal(llamadas.length, 1);
  assert.equal(llamadas[0].ruta, "/api/presentacion/cartografia/rutas");
  assert.equal(llamadas[0].opciones.method, "POST");
  assert.equal(llamadas[0].opciones.credentials, "omit");
  assert.equal(llamadas[0].opciones.mode, "same-origin");
  assert.equal(llamadas[0].opciones.redirect, "error");
  assert.equal(llamadas[0].opciones.referrer, "");
  assert.equal(llamadas[0].opciones.referrerPolicy, "no-referrer");
  assert.deepEqual(JSON.parse(llamadas[0].opciones.body), {
    coordinates: [
      { lat: 37.17428891, lon: -3.59869101, name: "Granada" },
      { lat: 36.74535308, lon: -3.52045559, name: "Motril" },
    ],
    alternatives: 1,
  });
});

test("genera una huella SHA-256 estable y conserva como maximo 2000 puntos", async () => {
  const geometria = Array.from({ length: 2_001 }, (_, indice) => [
    -3.59869101 + indice / 100_000,
    37.17428891 - indice / 100_000,
  ]);
  const fetchImpl = async () => respuestaJSON(respuestaOSRM({ geometria }));
  const adaptador = crearAdaptador(fetchImpl);
  const primero = await adaptador.calcular(solicitud());
  const segundo = await adaptador.calcular(solicitud());
  assert.equal(primero.referencia, segundo.referencia);
  assert.equal(primero.alternativas[0].geometria.trazado.length, 2_000);
  assert.deepEqual(primero.alternativas[0].geometria.trazado.at(0), [geometria[0][1], geometria[0][0]]);
  assert.deepEqual(primero.alternativas[0].geometria.trazado.at(-1), [geometria.at(-1)[1], geometria.at(-1)[0]]);
});

test("acepta data_version gobernada y rechaza versiones declaradas contradictorias", async () => {
  const soloDataVersion = respuestaOSRM();
  soloDataVersion.data_version = soloDataVersion.graph_version;
  delete soloDataVersion.graph_version;
  const calculo = await crearAdaptador(async () => respuestaJSON(soloDataVersion))
    .calcular(solicitud());
  assert.equal(calculo.version_grafo, VERSION_GRAFO);

  const ambasIguales = respuestaOSRM();
  ambasIguales.data_version = ambasIguales.graph_version;
  assert.equal((await crearAdaptador(async () => respuestaJSON(ambasIguales))
    .calcular(solicitud())).version_grafo, VERSION_GRAFO);

  const contradictoria = respuestaOSRM();
  contradictoria.data_version = "granada-buffer-osrm-v2-no-gobernada";
  await exigirErrorCerrado(
    crearAdaptador(async () => respuestaJSON(contradictoria)).calcular(solicitud()),
  );

  const segundaVersion = respuestaOSRM({ version: "granada-buffer-osrm-v2-otra-version" });
  const segundo = await crearAdaptador(async () => respuestaJSON(segundaVersion)).calcular(solicitud());
  assert.equal(segundo.version_grafo, "granada-buffer-osrm-v2-otra-version");
  assert.notEqual(segundo.referencia, calculo.referencia);
});

test("falla cerrado ante producto, simulacion implícita u opciones de red inyectadas", () => {
  const basicas = {
    contextoActor: contexto(),
    capacidades: [CAPACIDAD_CONSULTAR_RUTA],
    fetchImpl: async () => respuestaJSON(respuestaOSRM()),
  };
  assert.throws(
    () => crearCalculadorRutasDietasPresentacionOSRM({ ...basicas, contextoActor: contexto(false) }),
    /exige un contexto DEMO/u,
  );
  assert.throws(
    () => crearCalculadorRutasDietasPresentacionOSRM({ ...basicas, capacidades: [] }),
    /falta capacidad/u,
  );
  assert.throws(
    () => crearCalculadorRutasDietasPresentacionOSRM({ ...basicas, fetchImpl: undefined }),
    /falta el cliente HTTP/u,
  );
  assert.throws(
    () => crearCalculadorRutasDietasPresentacionOSRM({ ...basicas, versionGrafo: VERSION_GRAFO }),
    /opciones .* no validas/u,
  );
  assert.throws(
    () => crearCalculadorRutasDietasPresentacionOSRM({ ...basicas, baseURL: "https://malicioso.invalid" }),
    /opciones .* no validas/u,
  );
});

test("rechaza entrada de ruta o SSRF antes de invocar red y nunca cambia el endpoint", async () => {
  let llamadas = 0;
  const adaptador = crearAdaptador(async (ruta) => {
    llamadas += 1;
    assert.equal(ruta, "/api/presentacion/cartografia/rutas");
    return respuestaJSON(respuestaOSRM());
  });
  await exigirErrorCerrado(
    adaptador.calcular(solicitud(["18087", "https://169.254.169.254/latest/meta-data"])),
  );
  await exigirErrorCerrado(
    adaptador.calcular({ ...solicitud(), destino: "//malicioso.invalid" }),
  );
  assert.equal(llamadas, 0);
  await adaptador.calcular(solicitud());
  assert.equal(llamadas, 1);
});

test("valida identidad y version del mediador, code, rutas, legs y GeoJSON", async (t) => {
  const casos = [
    ["code", (dato) => { dato.code = "NoRoute"; }],
    ["motor", (dato) => { dato.engine = "servicio_externo"; }],
    ["ambito", (dato) => { dato.route_scope = ""; }],
    ["version ausente", (dato) => { delete dato.graph_version; }],
    ["rutas", (dato) => { dato.routes = []; }],
    ["legs", (dato) => { dato.routes[0].legs = []; }],
    ["GeoJSON tipo", (dato) => { dato.routes[0].geometry.type = "Point"; }],
    ["GeoJSON coordenada", (dato) => { dato.routes[0].geometry.coordinates[0] = [999, 999]; }],
  ];
  for (const [nombre, mutar] of casos) {
    await t.test(nombre, async () => {
      const dato = respuestaOSRM();
      mutar(dato);
      const adaptador = crearAdaptador(async () => respuestaJSON(dato));
      await exigirErrorCerrado(adaptador.calcular(solicitud()));
    });
  }
});

test("no aplica fallback silencioso y limita tipo, tamano y UTF-8 de la respuesta", async () => {
  await exigirErrorCerrado(
    crearAdaptador(async () => { throw new Error("servicio caido"); }).calcular(solicitud()),
  );
  await exigirErrorCerrado(
    crearAdaptador(async () => respuestaJSON(respuestaOSRM(), { tipo: "text/html" })).calcular(solicitud()),
  );
  const declaracionExcesiva = new Response("{}", {
    status: 200,
    headers: { "Content-Type": "application/json", "Content-Length": String(9 * 1024 * 1024) },
  });
  await exigirErrorCerrado(
    crearAdaptador(async () => declaracionExcesiva).calcular(solicitud()),
  );
  const invalido = new Response(new Uint8Array([0xC3, 0x28]), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
  await exigirErrorCerrado(
    crearAdaptador(async () => invalido).calcular(solicitud()),
  );
});

test("el adaptador OSRM depende del catálogo neutral y no importa el simulador", async () => {
  const [fuenteOSRM, fuenteSimulador] = await Promise.all([
    readFile(new URL("calculador-rutas-presentacion-osrm.js", import.meta.url), "utf8"),
    readFile(new URL("calculador-rutas-presentacion.js", import.meta.url), "utf8"),
  ]);
  assert.match(fuenteOSRM, /catalogo-rutas-provincial\.js/u);
  assert.match(fuenteSimulador, /catalogo-rutas-provincial\.js/u);
  assert.doesNotMatch(fuenteOSRM, /from "\.\/calculador-rutas-presentacion\.js"/u);
  assert.doesNotMatch(fuenteOSRM, /simulacion_osrm_demo|SEMILLAS_TRAMOS/u);

  const catalogo = obtenerCatalogoRutasProvincial();
  assert.equal(catalogo.puntos.length, 175);
  assert.doesNotMatch(JSON.stringify(catalogo), /latitud|longitud/iu);
  const puntos = resolverPuntosRutasProvincial(["18087", "18140"]);
  assert.deepEqual(puntos.map(({ nombre }) => nombre), ["Granada", "Motril"]);
  assert.equal(Object.isFrozen(puntos), true);
  assert.equal(Object.isFrozen(puntos[0]), true);
});
