import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { crearControladorPreferenciasBolsas, cargarPreferenciasBolsas, renderizarPreferenciasBolsas, validarPreferenciasBolsas } from "./mi-bolsa-preferencias-bolsas.js";

const consulta = (extra = {}) => ({ version: 0, orden_guardado: [], bolsas_actuales: ["bolsa:1", "bolsa:2"], bolsas_nuevas: ["bolsa:1", "bolsa:2"], bolsas_retiradas: [], revision_requerida: true, recibo_ref: null, registrado_en: null, ...extra });
const respuesta = (valor, status = 200) => ({ status, text: async () => JSON.stringify(valor) });
const tick = () => new Promise((resolver) => setImmediate(resolver));
function contenedor() {
  const eventos = new Map();
  const vista = { innerHTML: "", eventos, enfocado: "", addEventListener(tipo, fn) { eventos.set(tipo, fn); }, removeEventListener(tipo) { eventos.delete(tipo); },
    querySelector(selector) { return { disabled: false, focus() { vista.enfocado = selector; } }; },
    pulsar(accion, indice) { eventos.get("click")({ target: { closest: () => ({ dataset: { preferenciasBolsas: accion, indice: String(indice) } }) } }); },
    cambiar(checked) { eventos.get("change")({ target: { dataset: { preferenciasBolsas: "confirmar" }, checked } }); } };
  return vista;
}

test("GET exige proyección cerrada y no acepta referencias ajenas al conjunto", async () => {
  const solicitudes = [];
  const datos = await cargarPreferenciasBolsas({ fetchImpl: async (ruta, opciones) => { solicitudes.push({ ruta, opciones }); return respuesta(consulta()); } });
  assert.equal(datos.version, 0);
  assert.equal(solicitudes[0].ruta, "/api/vec/bolsa/mi-bolsa/preferencias");
  assert.equal(solicitudes[0].opciones.method, "GET");
  assert.equal(solicitudes[0].opciones.credentials, "same-origin");
  assert.equal(solicitudes[0].opciones.cache, "no-store");
  assert.throws(() => validarPreferenciasBolsas(consulta({ candidato_ref: "persona:ajena" })), /respuesta/u);
  assert.throws(() => validarPreferenciasBolsas(consulta({ bolsas_actuales: ["bolsa:1", "bolsa:1"] })), /respuesta/u);
  assert.throws(() => validarPreferenciasBolsas(consulta({ bolsas_nuevas: ["bolsa:3"] })), /respuesta/u);
  await assert.rejects(cargarPreferenciasBolsas({ fetchImpl: async () => ({ status: 403 }) }), (error) => error.codigo === "denegado");
});

test("muestra todas las bolsas actuales y exige confirmar altas o bajas", () => {
  const html = renderizarPreferenciasBolsas(consulta(), { estado: "correcto", nombres: new Map([["bolsa:1", "Técnico <A>"], ["bolsa:2", "Administración"]]) });
  assert.match(html, /Técnico &lt;A&gt;/u);
  assert.match(html, /Administración/u);
  assert.doesNotMatch(html, /bolsa:1/u);
  assert.match(html, /Nueva participación/u);
  assert.doesNotMatch(html, /Hay 2 bolsas/u);
  assert.match(html, /data-preferencias-bolsas="guardar" disabled/u);
  assert.match(renderizarPreferenciasBolsas(consulta(), { estado: "correcto", confirmado: true }), /data-preferencias-bolsas="guardar"/u);
  const retiradas = renderizarPreferenciasBolsas(consulta({ bolsas_nuevas: [], bolsas_retiradas: ["bolsa:3", "bolsa:4"] }), { estado: "correcto", nombres: new Map([["bolsa:1", "Gestión"], ["bolsa:2", "Apoyo"]]) });
  assert.match(retiradas, /El orden anterior incluye bolsas/u);
  assert.doesNotMatch(retiradas, /Bolsa sin nombre disponible/u);
});

test("al desmontar y remontar durante GET no aborta ni duplica la petición y conserva el borrador", async () => {
  let completar, signal; let peticiones = 0;
  const controlador = crearControladorPreferenciasBolsas({ fetchImpl: (_ruta, opciones) => { peticiones++; signal = opciones.signal; return new Promise((resolver) => { completar = resolver; }); } });
  const primero = contenedor(), segundo = contenedor();
  controlador.montar({ contenedor: primero });
  controlador.desmontar();
  assert.equal(signal.aborted, false);
  controlador.montar({ contenedor: segundo });
  assert.equal(peticiones, 1);
  completar(respuesta(consulta({ bolsas_nuevas: [], revision_requerida: false, orden_guardado: ["bolsa:1", "bolsa:2"] })));
  await tick();
  segundo.pulsar("bajar", 0);
  assert.match(segundo.innerHTML, /data-indice="0"/u);
  assert.equal(segundo.enfocado, '[data-preferencias-bolsas="bajar"][data-indice="1"]');
  const tercero = contenedor(); controlador.montar({ contenedor: tercero });
  assert.equal(peticiones, 1);
  assert.match(tercero.innerHTML, /Guardar orden/u);
  assert.equal(primero.eventos.size, 0); controlador.destruir();
});

test("un resultado incierto reintenta cuerpo y clave idénticos sin dos PUT simultáneos", async () => {
  const cuerpos = []; let segundoResolver;
  const vista = contenedor();
  const controlador = crearControladorPreferenciasBolsas({ fetchImpl: async (_ruta, opciones) => {
    if (opciones.method === "GET") return respuesta(consulta());
    cuerpos.push(JSON.parse(opciones.body));
    if (cuerpos.length === 1) throw new TypeError("red");
    return new Promise((resolver) => { segundoResolver = resolver; });
  } });
  controlador.montar({ contenedor: vista }); await tick();
  vista.cambiar(true); vista.pulsar("guardar"); await tick();
  assert.match(vista.innerHTML, /No se pudo confirmar/u);
  vista.pulsar("reintentar-guardado"); vista.pulsar("reintentar-guardado");
  assert.equal(cuerpos.length, 2);
  assert.deepEqual(cuerpos[0], cuerpos[1]);
  assert.deepEqual(Object.keys(cuerpos[0]).sort(), ["clave_idempotencia", "orden", "version_esperada"]);
  segundoResolver(respuesta({ recibo_ref: "recibo:1", version: 1, orden: ["bolsa:1", "bolsa:2"], registrado_en: "2026-10-12T10:00:00.000000Z", reutilizado: true }));
  await tick(); assert.match(vista.innerHTML, /Orden guardado/u); assert.doesNotMatch(vista.innerHTML, /recibo:1/u); controlador.destruir();
});

test("403 y 409 no presentan éxito; salir durante GET no pinta otra vista", async () => {
  for (const status of [403, 409]) {
    const vista = contenedor();
    const controlador = crearControladorPreferenciasBolsas({ fetchImpl: async (_ruta, opciones) => opciones.method === "GET" ? respuesta(consulta()) : ({ status }) });
    controlador.montar({ contenedor: vista }); await tick(); vista.cambiar(true); vista.pulsar("guardar"); await tick();
    assert.doesNotMatch(vista.innerHTML, /Orden guardado/u);
    assert.match(vista.innerHTML, status === 403 ? /No tiene permiso/u : /Sus bolsas han cambiado/u);
    if (status === 403) assert.doesNotMatch(vista.innerHTML, /data-preferencias-bolsas="recargar"/u);
    controlador.destruir();
  }
  let completar, signal;
  const vista = contenedor();
  const controlador = crearControladorPreferenciasBolsas({ fetchImpl: (_ruta, opciones) => {
    signal = opciones.signal; return new Promise((resolver) => { completar = resolver; });
  } });
  controlador.montar({ contenedor: vista }); controlador.destruir();
  assert.equal(signal.aborted, true); vista.innerHTML = "Otra vista";
  completar(respuesta(consulta())); await tick();
  assert.equal(vista.innerHTML, "Otra vista");
});

test("catálogos ES/EN incluyen todos los mensajes del componente", async () => {
  for (const idioma of ["es", "en"]) {
    const catalogo = JSON.parse(await readFile(new URL(`../textos/${idioma}/area-personal.json`, import.meta.url), "utf8"));
    const mensajes = catalogo.miBolsa.preferenciasBolsas;
    for (const clave of ["titulo", "cargando", "guardando", "vacio", "bolsaSinNombre", "confirmarNuevoOrden", "guardar", "denegado", "conflicto", "incierto", "reintentarGuardado"]) assert.equal(typeof mensajes[clave], "string");
  }
});
