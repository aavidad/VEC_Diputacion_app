import assert from "node:assert/strict";
import test from "node:test";
import { crearLectorConsultaMeritoPropio, RUTA_CONSULTA_MERITO_PROPIO } from "./consulta-cliente-http.js";
import { resultadoPrueba, noEncontradaPrueba, respuestaPrueba, pendiente } from "./consulta-prueba.test-helper.mjs";

test("POST cerrado de lectura al origen propio sin cookies, caché ni identidad enviada", async () => {
  let solicitud;
  const lector = crearLectorConsultaMeritoPropio({ fetchImpl: async (ruta, opciones) => { solicitud = { ruta, opciones }; return respuestaPrueba(); } });
  const datos = await lector.consultar({ hechoRef: "hecho:propio-a" });
  assert.equal(solicitud.ruta, RUTA_CONSULTA_MERITO_PROPIO);
  assert.equal(solicitud.opciones.method, "POST");
  assert.deepEqual(JSON.parse(solicitud.opciones.body), { hecho_ref: "hecho:propio-a" });
  for (const [clave, valor] of Object.entries({ credentials: "omit", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer" })) assert.equal(solicitud.opciones[clave], valor);
  assert.equal(datos.hecho_actual.version, 2);
  assert.ok(Object.isFrozen(datos));
});

test("no admite campos de persona, actor, perfil o rutas desde el cliente", async () => {
  let llamadas = 0;
  const lector = crearLectorConsultaMeritoPropio({ fetchImpl: async () => { llamadas++; return respuestaPrueba(); } });
  for (const campo of ["actorRef", "persona_ref", "perfil", "endpoint", "headers"]) {
    await assert.rejects(lector.consultar({ hechoRef: "hecho:propio-a", [campo]: "ajena" }), (e) => e.codigo === "peticion_invalida");
  }
  await assert.rejects(lector.consultar({ hechoRef: "https://otra.invalid" }));
  assert.throws(() => crearLectorConsultaMeritoPropio({ actorRef: "actor:1" }));
  assert.equal(llamadas, 0);
});

test("una búsqueda sin hecho devuelve el recibo nuevo de versión cero", async () => {
  const lector = crearLectorConsultaMeritoPropio({ fetchImpl: async () => respuestaPrueba(noEncontradaPrueba()) });
  const datos = await lector.consultar({ hechoRef: "hecho:propio-a" });
  assert.equal(datos.codigo, "no_encontrada"); assert.equal(datos.hecho_actual, null); assert.equal(datos.recibo_consulta.version_consultada, 0);
});

test("los errores HTTP cancelan el cuerpo y no propagan datos de otra persona", async () => {
  for (const status of [401, 403, 404, 500, 503]) {
    let cancelado = false; let leido = false;
    const lector = crearLectorConsultaMeritoPropio({ fetchImpl: async () => ({ status, body: { cancel() { cancelado = true; }, getReader() { leido = true; } } }) });
    await assert.rejects(lector.consultar({ hechoRef: "hecho:propio-a" }), (e) => { assert.equal(Object.hasOwn(e, "data"), false); return e.codigo === ([401, 403].includes(status) ? "denegada" : "error"); });
    assert.equal(cancelado, true); assert.equal(leido, false);
  }
});

test("aborta el transporte y rechaza una respuesta tardía incluso si el fetch inyectado ignora el aborto", async () => {
  const remoto = pendiente(); let interna;
  const lector = crearLectorConsultaMeritoPropio({ fetchImpl: (_, opciones) => { interna = opciones.signal; return remoto.promesa; } });
  const controlador = new AbortController(); const carga = lector.consultar({ hechoRef: "hecho:propio-a", signal: controlador.signal });
  controlador.abort(); assert.equal(interna.aborted, true); remoto.resolver(respuestaPrueba());
  await assert.rejects(carga, (e) => e.codigo === "abortada");
});

test("no acepta sobre ampliado, versión cruzada, JSON inválido ni cuerpos por encima del límite", async () => {
  const ajena = resultadoPrueba(); ajena.hecho_actual.persona_ref = "persona:otra";
  const cruzada = resultadoPrueba(); cruzada.recibo_consulta.version_consultada = 1;
  const casos = [respuestaPrueba(ajena), respuestaPrueba(cruzada), respuestaPrueba({ data: resultadoPrueba() }),
    new Response("{", { headers: { "Content-Type": "application/json" } }),
    new Response("x".repeat(65537), { headers: { "Content-Type": "application/json" } }),
    new Response("{}", { headers: { "Content-Type": "text/html" } })];
  for (const respuesta of casos) await assert.rejects(crearLectorConsultaMeritoPropio({ fetchImpl: async () => respuesta }).consultar({ hechoRef: "hecho:propio-a" }));
});
