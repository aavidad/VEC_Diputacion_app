import assert from "node:assert/strict";
import test from "node:test";
import { montarConsumidorConsultaMerito, obtenerContextoConsultaMerito } from "./consulta-montaje.js";
import { resultadoPrueba, respuestaPrueba, pendiente, raizPrueba } from "./consulta-prueba.test-helper.mjs";

function contexto(datos = { hecho_ref: "hecho:propio-a", sintetico: true }) { return new Response(JSON.stringify(datos), { headers: { "Content-Type": "application/json" } }); }
function documentoPrueba() {
  const espacio = raizPrueba(); const documento = espacio.ownerDocument;
  const nodos = new Map(["consulta-espacio", "consulta-aviso-sintetico", "consulta-anuncio", "consulta-idiomas"].map((id) => [id, id === "consulta-espacio" ? espacio : documento.createElement("div")]));
  documento.getElementById = (id) => nodos.get(id); documento.querySelectorAll = () => []; documento.querySelector = () => null; documento.documentElement = {};
  const oyentes = new Map();
  const ventana = { addEventListener: (nombre, cb) => oyentes.set(nombre, cb), removeEventListener: (nombre, cb) => { if (oyentes.get(nombre) === cb) oyentes.delete(nombre); } };
  return { documento, ventana, espacio, nodos, oyentes };
}

test("contexto GET fijo, cerrado y mínimo, sin cookies ni campos de autoridad", async () => {
  let solicitud;
  const datos = await obtenerContextoConsultaMerito({ signal: new AbortController().signal, fetchImpl: async (ruta, opciones) => { solicitud = { ruta, opciones }; return contexto(); } });
  assert.deepEqual(datos, { hecho_ref: "hecho:propio-a", sintetico: true }); assert.ok(Object.isFrozen(datos));
  assert.equal(solicitud.ruta, "/api/meritos/consulta-contexto");
  for (const [clave, valor] of Object.entries({ method: "GET", credentials: "omit", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer" })) assert.equal(solicitud.opciones[clave], valor);
  assert.equal(Object.hasOwn(solicitud.opciones, "body"), false);
  for (const extra of [{ actor_ref: "actor:1" }, { persona_ref: "persona:1" }, { perfil: "administrador" }, { entorno: "sintetico_interno" }]) {
    await assert.rejects(obtenerContextoConsultaMerito({ signal: new AbortController().signal, fetchImpl: async () => contexto({ ...datos, ...extra }) }));
  }
  for (const invalido of [{ hecho_ref: "hecho:propio-a" }, { ...datos, sintetico: false }, { ...datos, hecho_ref: "https://ajena.invalid" }]) await assert.rejects(obtenerContextoConsultaMerito({ signal: new AbortController().signal, fetchImpl: async () => contexto(invalido) }));
});

test("el consumidor obtiene el contexto y consulta por HTTP real con la referencia recibida", async () => {
  const d = documentoPrueba(); const solicitudes = [];
  const consumidor = montarConsumidorConsultaMerito({ documento: d.documento, ventana: d.ventana, fetchImpl: async (ruta, opciones) => { solicitudes.push({ ruta, opciones }); return ruta.endsWith("consulta-contexto") ? contexto() : respuestaPrueba(resultadoPrueba()); } });
  await consumidor.preparada;
  assert.equal(solicitudes.length, 2); assert.equal(solicitudes[0].opciones.method, "GET"); assert.equal(solicitudes[1].opciones.method, "POST");
  assert.deepEqual(JSON.parse(solicitudes[1].opciones.body), { hecho_ref: "hecho:propio-a" });
  assert.equal(d.nodos.get("consulta-aviso-sintetico").hidden, false);
  assert.match(d.espacio.children[0].innerHTML, /Curso de gestión/u);
  assert.equal(d.nodos.get("consulta-anuncio").textContent, "Detalle del mérito actualizado");
  assert.equal(d.nodos.get("consulta-idiomas").children.length, 2);
  assert.equal(d.documento.documentElement.lang, "es");
  consumidor.desmontar(); assert.equal(d.oyentes.size, 0); assert.equal(d.nodos.get("consulta-aviso-sintetico").hidden, true);
});

test("un contexto fallido o ampliado no realiza POST ni carga ejemplos", async () => {
  for (const respuesta of [contexto({ hecho_ref: "hecho:propio-a", sintetico: true, persona_ref: "persona:ajena" }), new Response("privado", { status: 403 })]) {
    const d = documentoPrueba(); let llamadas = 0;
    const consumidor = montarConsumidorConsultaMerito({ documento: d.documento, ventana: d.ventana, fetchImpl: async () => { llamadas++; return respuesta; } });
    await consumidor.preparada; assert.equal(llamadas, 1); assert.equal(d.nodos.get("consulta-aviso-sintetico").hidden, true);
    assert.doesNotMatch(d.espacio.innerHTML, /Curso de gestión|persona:ajena|privado/u);
    consumidor.desmontar();
  }
});

test("pagehide cancela el contexto pendiente y descarta la respuesta sin consultar el hecho", async () => {
  const d = documentoPrueba(); const remoto = pendiente(); let signal; let llamadas = 0;
  const consumidor = montarConsumidorConsultaMerito({ documento: d.documento, ventana: d.ventana, fetchImpl: (_ruta, opciones) => { signal = opciones.signal; llamadas++; return remoto.promesa; } });
  d.oyentes.get("pagehide")(); assert.equal(signal.aborted, true);
  remoto.resolver(contexto()); await consumidor.preparada;
  assert.equal(llamadas, 1); assert.equal(d.espacio.innerHTML, ""); assert.equal(d.nodos.get("consulta-aviso-sintetico").hidden, true);
});
