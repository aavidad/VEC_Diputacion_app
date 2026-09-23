import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { crearAdaptadorPresentacion } from "./adaptador-presentacion.js";
import { crearClienteHTTPAreaPersonal, ErrorClienteAreaPersonal } from "./cliente-http.js";

function respuestaJSON(datos, estado = 200) {
  const texto = JSON.stringify(datos);
  return {
    status: estado,
    headers: { get: (nombre) => ({ "Content-Type": "application/json; charset=utf-8", "Content-Length": String(new TextEncoder().encode(texto).byteLength) })[nombre] ?? null },
    text: async () => texto,
  };
}

async function datosProductivosSintéticos() {
  const datos = structuredClone(await crearAdaptadorPresentacion().cargar());
  datos.meta.presentacion = false;
  datos.meta.origen = "API interna autenticada de prueba";
  datos.sesion.persona_ref = "PER-PRUEBA-0001";
  datos.perfil.referencia = "PERFIL-PRUEBA-0001";
  return datos;
}

test("el cliente real no cae jamás al adaptador demo ante un fallo de red", async () => {
  const cliente = crearClienteHTTPAreaPersonal({ fetchImpl: async () => { throw new Error("sin red"); } });
  await assert.rejects(
    () => cliente.cargar(),
    (error) => error instanceof ErrorClienteAreaPersonal && error.codigo === "servicio_no_disponible",
  );
});

test("mi bolsa se consulta primero sin cookies, query ni cabeceras de identidad", async () => {
  let peticion;
  const cliente = crearClienteHTTPAreaPersonal({ fetchImpl: async (ruta, opciones) => {
    peticion = { ruta, opciones };
    return respuestaJSON({ data: { esquema: "vec.bolsa.mi-bolsa.v1", consultada_en: "2026-09-23T10:00:00.000000Z", participaciones: [{ bolsa: "bolsa:prueba:01", categoria: "Auxiliar", version: 3, orden_inicial: 12, total_instantanea: 87, estado_bolsa: "vigente", vigente_desde: "2026-09-01T00:00:00.000000Z", vigente_hasta: null }] } });
  } });
  const recibido = await cliente.cargar();
  assert.equal(recibido.fuente, "real");
  assert.equal(recibido.consulta.participaciones[0].orden_inicial, 12);
  assert.equal(peticion.ruta, "/api/vec/bolsa/mi-bolsa");
  assert.equal(peticion.opciones.credentials, "omit");
  assert.equal(peticion.opciones.cache, "no-store");
  assert.equal(peticion.opciones.headers.Authorization, undefined);
  assert.equal(peticion.opciones.headers["X-Identity"], undefined);
});

test("el cliente acepta el payload exacto serializado por httppersonal", async () => {
  const archivo = new URL("../../../internal/modules/bolsa/adapters/httppersonal/testdata/mi_bolsa_situacion.json", import.meta.url);
  const payload = JSON.parse(await readFile(archivo, "utf8"));
  const cliente = crearClienteHTTPAreaPersonal({ fetchImpl: async () => respuestaJSON(payload) });
  const recibido = await cliente.cargar();
  assert.equal(recibido.consulta.participaciones[0].situacion_actual.estado, "no_disponible");
  assert.equal(recibido.consulta.consultada_en, "2026-09-20T11:00:00.000000Z");
});

test("solo 401, 404 y 503 permiten volver al panel sintético rotulado", async () => {
  const datos = await datosProductivosSintéticos();
  const rutas = [];
  const cliente = crearClienteHTTPAreaPersonal({ fetchImpl: async (ruta) => {
    rutas.push(ruta);
    return rutas.length === 1 ? respuestaJSON({}, 404) : respuestaJSON({ data: datos });
  } });
  const recibido = await cliente.cargar();
  assert.equal(recibido.fuente, "ejemplo");
  assert.deepEqual(rutas, ["/api/vec/bolsa/mi-bolsa", "/api/vec/bolsa/area-personal"]);
});

test("403 no mezcla datos de ejemplo con una identidad sin acceso", async () => {
  const cliente = crearClienteHTTPAreaPersonal({ fetchImpl: async () => respuestaJSON({}, 403) });
  await assert.rejects(() => cliente.cargar(), (error) => error.codigo === "acceso_denegado");
});

test("el cliente HTTP rechaza capacidad ausente antes de tocar la red", async () => {
  let llamadas = 0;
  const cliente = crearClienteHTTPAreaPersonal({ fetchImpl: async () => { llamadas += 1; } });
  await assert.rejects(
    () => cliente.ejecutar({ accion: "guardar_borrador", confirmacion: true, capacidad: false }),
    (error) => error.codigo === "capacidad_denegada",
  );
  assert.equal(llamadas, 0);
});

test("un fichero no sale por JSON si el puerto documental no está compuesto", async () => {
  let llamadas = 0;
  const cliente = crearClienteHTTPAreaPersonal({ fetchImpl: async () => { llamadas += 1; } });
  await assert.rejects(
    () => cliente.ejecutar({
      accion: "incorporar_merito",
      payload: { documento: { nombre: "evidencia.pdf", tipo: "application/pdf", tamano: 1200 } },
      confirmacion: true,
      capacidad: true,
    }),
    (error) => error.codigo === "carga_documental_no_compuesta",
  );
  assert.equal(llamadas, 0);
});

test("el cliente HTTP rechaza un contrato que no sea mi-bolsa", async () => {
  const demo = await crearAdaptadorPresentacion().cargar();
  const cliente = crearClienteHTTPAreaPersonal({ fetchImpl: async () => respuestaJSON({ data: demo }) });
  await assert.rejects(() => cliente.cargar(), /mi-bolsa\.esquema/u);
});

test("una acción real exige confirmación, idempotencia y recibo productivo", async () => {
  let peticion;
  const recibo = {
    esquema: "vec.bolsa.area-personal.recibo.v1",
    presentacion: false,
    referencia: "REC-PRUEBA-0001",
    accion: "guardar_borrador",
    objetivo: "PERSONA-PRUEBA-0001",
    resultado: "Borrador guardado",
    actor: "PERSONA-PRUEBA-0001",
    fecha: "2026-07-18T09:00:00Z",
    advertencia: "Conserve este recibo para futuras comprobaciones.",
  };
  const cliente = crearClienteHTTPAreaPersonal({ fetchImpl: async (ruta, opciones) => {
    peticion = { ruta, opciones };
    return respuestaJSON({ data: { recibo, resultado: { version: "1" } } }, 201);
  } });
  const resultado = await cliente.ejecutar({
    accion: "guardar_borrador",
    payload: { convocatoria_id: "CONV-PRUEBA-0001" },
    confirmacion: true,
    capacidad: true,
  });
  assert.equal(resultado.recibo.presentacion, false);
  assert.equal(peticion.ruta, "/api/vec/bolsa/mis-solicitudes/borrador");
  assert.equal(peticion.opciones.credentials, "omit");
  assert.match(peticion.opciones.headers["X-Idempotency-Key"], /^WEB-[0-9a-f-]{36}$/u);
  assert.equal(peticion.opciones.headers.Authorization, undefined);
});

test("una respuesta no JSON o excesiva falla cerrada", async () => {
  const clienteTipo = crearClienteHTTPAreaPersonal({ fetchImpl: async () => ({
    status: 200,
    headers: { get: (nombre) => nombre === "Content-Type" ? "text/html" : null },
    text: async () => "<html></html>",
  }) });
  await assert.rejects(() => clienteTipo.cargar(), (error) => error.codigo === "tipo_respuesta");
  const clienteTamano = crearClienteHTTPAreaPersonal({ fetchImpl: async () => ({
    status: 200,
    headers: { get: (nombre) => nombre === "Content-Type" ? "application/json" : "9999999" },
    text: async () => "{}",
  }) });
  await assert.rejects(() => clienteTamano.cargar(), (error) => error.codigo === "respuesta_excesiva");
});
