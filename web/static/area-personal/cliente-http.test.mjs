import assert from "node:assert/strict";
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

test("la carga HTTP omite cookies, caché y credenciales inventadas", async () => {
  let peticion;
  const datos = await datosProductivosSintéticos();
  const cliente = crearClienteHTTPAreaPersonal({ fetchImpl: async (ruta, opciones) => {
    peticion = { ruta, opciones };
    return respuestaJSON({ data: datos });
  } });
  const recibido = await cliente.cargar();
  assert.equal(recibido.meta.presentacion, false);
  assert.equal(peticion.ruta, "/api/vec/bolsa/area-personal");
  assert.equal(peticion.opciones.credentials, "omit");
  assert.equal(peticion.opciones.cache, "no-store");
  assert.equal(peticion.opciones.headers.Authorization, undefined);
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

test("el cliente HTTP rechaza una respuesta demo en la ruta productiva", async () => {
  const demo = await crearAdaptadorPresentacion().cargar();
  const cliente = crearClienteHTTPAreaPersonal({ fetchImpl: async () => respuestaJSON({ data: demo }) });
  await assert.rejects(() => cliente.cargar(), /origen no coincide/);
});

test("una acción real exige confirmación, idempotencia y recibo productivo", async () => {
  let peticion;
  const recibo = {
    esquema: "vec.bolsa.area-personal.recibo.v1",
    presentacion: false,
    referencia: "REC-PRUEBA-0001",
    accion: "guardar_borrador",
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
