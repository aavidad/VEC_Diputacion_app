import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { crearClienteHTTPAreaPersonal, ErrorClienteAreaPersonal } from "./cliente-http.js";

function respuestaJSON(datos, estado = 200) {
  const texto = JSON.stringify(datos);
  return {
    status: estado,
    headers: { get: (nombre) => ({ "Content-Type": "application/json; charset=utf-8", "Content-Length": String(new TextEncoder().encode(texto).byteLength) })[nombre] ?? null },
    text: async () => texto,
  };
}

test("el cliente real no cae jamás al adaptador demo ante un fallo de red", async () => {
  const rutas = [];
  const cliente = crearClienteHTTPAreaPersonal({ fetchImpl: async (ruta) => {
    rutas.push(ruta);
    throw new Error("sin red");
  } });
  await assert.rejects(
    () => cliente.cargar(),
    (error) => error instanceof ErrorClienteAreaPersonal && error.codigo === "servicio_no_disponible",
  );
  assert.deepEqual(rutas, ["/api/vec/bolsa/mi-bolsa"]);
});

test("mi bolsa se consulta primero sin cookies, query ni cabeceras de identidad", async () => {
  const peticiones = [];
  const cliente = crearClienteHTTPAreaPersonal({ fetchImpl: async (ruta, opciones) => {
    peticiones.push({ ruta, opciones });
    return respuestaJSON({ data: { esquema: "vec.bolsa.mi-bolsa.v1", consultada_en: "2026-09-23T10:00:00.000000Z", participaciones: [{ bolsa: "bolsa:prueba:01", categoria: "Auxiliar", version: 3, orden_inicial: 12, total_instantanea: 87, estado_bolsa: "vigente", vigente_desde: "2026-09-01T00:00:00.000000Z", vigente_hasta: null }] } });
  } });
  const recibido = await cliente.cargar();
  assert.equal(recibido.fuente, "real");
  assert.equal(recibido.consulta.participaciones[0].orden_inicial, 12);
  assert.equal(peticiones.length, 1);
  const peticion = peticiones[0];
  assert.equal(peticion.ruta, "/api/vec/bolsa/mi-bolsa");
  assert.equal(peticion.opciones.credentials, "same-origin");
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
  assert.equal(recibido.consulta.participaciones[0].ultimo_llamamiento.resultado, "enviado");
  assert.equal(recibido.consulta.consultada_en, "2026-09-20T11:00:00.000000Z");
});

for (const [estado, codigo] of [
  [401, "autenticacion_requerida"],
  [403, "acceso_denegado"],
  [404, "recurso_no_encontrado"],
  [503, "servicio_no_disponible"],
]) {
  test(`mi bolsa HTTP ${estado} conserva el error y no consulta el panel anterior`, async () => {
    const rutas = [];
    const cliente = crearClienteHTTPAreaPersonal({ fetchImpl: async (ruta) => {
      rutas.push(ruta);
      return respuestaJSON({}, estado);
    } });
    await assert.rejects(
      () => cliente.cargar(),
      (error) => error instanceof ErrorClienteAreaPersonal && error.codigo === codigo,
    );
    assert.deepEqual(rutas, ["/api/vec/bolsa/mi-bolsa"]);
  });
}

test("el cliente solo consulta Mi bolsa y el contacto propio: no ofrece acciones sin servicio", () => {
  const cliente = crearClienteHTTPAreaPersonal({ fetchImpl: async () => respuestaJSON({}) });
  assert.deepEqual(Object.keys(cliente).sort(), ["cargar", "cargarContactoPropio", "modo"]);
});

test("el cliente HTTP rechaza un contrato que no sea mi-bolsa", async () => {
  // Doble mínimo del panel agregado: otro esquema que el cliente no debe aceptar.
  const panel = { meta: { esquema: "vec.bolsa.area-personal.v1", presentacion: false, origen: "prueba", generado_en: "2026-07-18T09:00:00Z" } };
  const cliente = crearClienteHTTPAreaPersonal({ fetchImpl: async () => respuestaJSON({ data: panel }) });
  await assert.rejects(() => cliente.cargar(), /mi-bolsa\.esquema/u);
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

test("recargar contacto consulta versión y recibo originales desde servidor", async () => {
  const peticiones = [];
  const cliente = crearClienteHTTPAreaPersonal({ fetchImpl: async (ruta, opciones) => {
    peticiones.push({ ruta, opciones });
    return peticiones.length === 1
      ? respuestaJSON({ capacidad: true, encontrado: true, version: 3 })
      : respuestaJSON({ recibo_ref: "acc_1234567890123456789012345678901234567890", version: 3 });
  } });
  const recibido = await cliente.cargarContactoPropio();
  assert.equal(recibido.autorizacion.version, 3);
  assert.equal(recibido.recibo.version, 3);
  assert.deepEqual(peticiones.map((p) => [p.ruta, p.opciones.method, p.opciones.credentials]), [
    ["/api/vec/usuarios/contacto-propio", "GET", "same-origin"],
    ["/api/vec/usuarios/contacto-propio/recibo", "POST", "same-origin"],
  ]);
  assert.equal(JSON.parse(peticiones[1].opciones.body).version, 3);
  assert.equal(JSON.stringify(recibido).includes("correo"), false);
});

test("contacto denegado no presenta capacidad ni consulta recibo", async () => {
  let llamadas = 0;
  const cliente = crearClienteHTTPAreaPersonal({ fetchImpl: async () => { llamadas += 1; return respuestaJSON({}, 403); } });
  await assert.rejects(() => cliente.cargarContactoPropio(), (error) => error.codigo === "acceso_denegado");
  assert.equal(llamadas, 1);
});

test("GET contacto sin capacidad positiva no habilita escritura ni consulta recibo", async () => {
  let llamadas = 0;
  const cliente = crearClienteHTTPAreaPersonal({ fetchImpl: async () => {
    llamadas += 1;
    return respuestaJSON({ encontrado: true, version: 3 });
  } });
  assert.equal(await cliente.cargarContactoPropio(), null);
  assert.equal(llamadas, 1);
});

for (const [status, codigo] of [[401, "autenticacion_requerida"], [403, "acceso_denegado"], [404, "recurso_no_encontrado"]]) {
  test(`GET contacto ${status} clasifica denegación antes de leer HTML o cuerpo vacío`, async () => {
    for (const cuerpo of ["<html>denegado</html>", ""]) {
      let leido = false;
      const cliente = crearClienteHTTPAreaPersonal({ fetchImpl: async () => ({
        status, headers: { get: () => cuerpo ? "text/html" : null },
        text: async () => { leido = true; return cuerpo; },
      }) });
      await assert.rejects(() => cliente.cargarContactoPropio(), (error) => error.codigo === codigo);
      assert.equal(leido, false);
    }
  });
}

test("versión sin recibo coincidente no confirma contacto", async () => {
  let llamadas = 0;
  const cliente = crearClienteHTTPAreaPersonal({ fetchImpl: async () => {
    llamadas += 1;
    return llamadas === 1 ? respuestaJSON({ capacidad: true, encontrado: true, version: 2 }) : respuestaJSON({ recibo_ref: "acc_otra", version: 1 });
  } });
  await assert.rejects(() => cliente.cargarContactoPropio(), (error) => error.codigo === "respuesta_incompatible");
});
