import assert from "node:assert/strict";
import test from "node:test";

import { iniciarPeticionCentro, iniciarPeticionesCentroRRHH, pedir } from "./peticiones-centro.js";

const catalogos = {
  esquema: "vec.contratacion_temporal.catalogos_alta.v1",
  centros: [{ referencia: "cen_sintetico_001", etiqueta: "Centro sintético", contactos: [{ referencia: "con_sintetico_001", etiqueta: "Contacto sintético" }] }],
  categorias: [{ referencia: "cat_sintetica_001", etiqueta: "Categoría sintética", grupos_subgrupos: [{ clave: "C2", etiqueta: "C2" }] }],
  motivos: [{ clave: "sustitucion", etiqueta: "Sustitución" }], documentos: [],
};
const contexto = { actor: { referencia: "actor:sintetico:001", nombre: "Actor anterior sintético", cargo: "Cargo anterior", centro: "Centro anterior", puede_presentar: true, puede_ratificar: false }, catalogos };
const peticion = { referencia: "peticion:centro:sintetica", version: 2, estado: "ratificada", configuracion: { solicitante: { actor_ref: "actor:sintetico:001", puesto_ref: "puesto:sintetico:001" } }, solicitud: { centro_ref: "cen_sintetico_001", categoria_ref: "cat_sintetica_001", motivo_clave: "sustitucion", detalle: "Dato personal previo sintético", periodo: {} } };
const entrega = { peticion, estado_entrega: "preparada", recibo_alta: { recibo_ref: "recibo:sintetico:anterior", expediente_ref: "expediente:sintetico:anterior", confirmada_en: "2026-09-06T08:00:00Z" } };

function raizFalsa() {
  const manejadores = new Map();
  return {
    innerHTML: "",
    manejadores,
    setAttribute() {},
    querySelector() { return { checked: true }; },
    querySelectorAll() { return []; },
    addEventListener(tipo, manejador) { manejadores.set(tipo, manejador); },
    async pulsar(dataset) {
      await manejadores.get("click")({ target: { dataset, closest: () => ({ dataset }) }, preventDefault() {} });
    },
  };
}

function verificarSinDatos(raiz) {
  assert.doesNotMatch(raiz.innerHTML, /Actor anterior sintético|Cargo anterior|Centro anterior|Dato personal previo sintético|peticion:centro:sintetica|recibo:sintetico:anterior|expediente:sintetico:anterior/);
  assert.doesNotMatch(raiz.innerHTML, /data-accion="(?:nueva|abrir-alta-rrhh|confirmar-alta-rrhh|confirmar-presentar|confirmar-ratificar|reintentar(?:-alta-rrhh)?)"/);
}

for (const status of [401, 403]) {
  test(`centro descarta contexto nuevo si la bandeja responde ${status}`, async () => {
    const raiz = raizFalsa();
    const llamadas = [];
    const cliente = async (ruta, opciones) => {
      llamadas.push([ruta, opciones?.method || "GET"]);
      if (ruta.endsWith("/contexto")) return contexto;
      throw { status };
    };
    await iniciarPeticionCentro({ raiz, cliente });
    assert.equal(llamadas.length, 2);
    assert.match(raiz.innerHTML, /Acceso denegado/);
    verificarSinDatos(raiz);
    await raiz.pulsar({ accion: "nueva" });
    assert.equal(llamadas.length, 2);
  });

  test(`centro retira contexto, petición y acciones al denegar recarga con ${status}`, async () => {
    const raiz = raizFalsa();
    const llamadas = [];
    let denegar = false;
    const cliente = async (ruta, opciones) => {
      llamadas.push([ruta, opciones?.method || "GET"]);
      if (denegar) throw { status };
      return ruta.endsWith("/contexto") ? contexto : { peticiones: [peticion] };
    };
    const vista = await iniciarPeticionCentro({ raiz, cliente });
    assert.match(raiz.innerHTML, /Actor anterior sintético/);
    await raiz.pulsar({ seleccionar: peticion.referencia });
    assert.match(raiz.innerHTML, /Dato personal previo sintético/);
    denegar = true;
    await vista.recargar();
    assert.match(raiz.innerHTML, /Acceso denegado/);
    verificarSinDatos(raiz);
    const anteriores = llamadas.length;
    await raiz.pulsar({ accion: "nueva" });
    await raiz.pulsar({ accion: "confirmar-ratificar" });
    await raiz.pulsar({ accion: "recargar" });
    assert.equal(llamadas.length, anteriores);
  });

  test(`RRHH retira lista, selección y recibo al denegar recarga con ${status}`, async () => {
    const raiz = raizFalsa();
    const llamadas = [];
    let denegar = false;
    const cliente = async (ruta, opciones) => {
      llamadas.push([ruta, opciones?.method || "GET"]);
      if (denegar) throw { status };
      return { limite: 50, peticiones: [entrega] };
    };
    const vista = await iniciarPeticionesCentroRRHH({ raiz, cliente });
    assert.match(raiz.innerHTML, /Dato personal previo sintético/);
    assert.match(raiz.innerHTML, /recibo:sintetico:anterior/);
    denegar = true;
    await vista.recargar();
    assert.match(raiz.innerHTML, /Acceso denegado/);
    verificarSinDatos(raiz);
    const anteriores = llamadas.length;
    await raiz.pulsar({ accion: "abrir-alta-rrhh" });
    await raiz.pulsar({ accion: "confirmar-alta-rrhh" });
    await raiz.pulsar({ accion: "recargar-rrhh" });
    assert.equal(llamadas.length, anteriores);
  });
}

for (const modo of ["centro", "rrhh"]) {
  test(`${modo} distingue fallo temporal y permite solo repetir la lectura`, async () => {
    const raiz = raizFalsa();
    let fallo = false;
    const llamadas = [];
    const cliente = async (ruta, opciones) => {
      llamadas.push([ruta, opciones?.method || "GET"]);
      if (fallo) throw { status: 503 };
      return modo === "centro" ? (ruta.endsWith("/contexto") ? contexto : { peticiones: [peticion] }) : { limite: 50, peticiones: [entrega] };
    };
    const vista = modo === "centro" ? await iniciarPeticionCentro({ raiz, cliente }) : await iniciarPeticionesCentroRRHH({ raiz, cliente });
    if (modo === "centro") await raiz.pulsar({ seleccionar: peticion.referencia });
    fallo = true;
    await vista.recargar();
    assert.match(raiz.innerHTML, /fallo temporal/);
    assert.doesNotMatch(raiz.innerHTML, /Acceso denegado/);
    assert.match(raiz.innerHTML, /Reintentar consulta/);
    verificarSinDatos(raiz);
    const anteriores = llamadas.length;
    await raiz.pulsar({ accion: modo === "centro" ? "nueva" : "abrir-alta-rrhh" });
    assert.equal(llamadas.length, anteriores);
    fallo = false;
    await raiz.pulsar({ accion: modo === "centro" ? "recargar" : "recargar-rrhh" });
    if (modo === "centro") await raiz.pulsar({ seleccionar: peticion.referencia });
    assert.match(raiz.innerHTML, /Dato personal previo sintético/);
    assert.ok(llamadas.length > anteriores);
  });
}

test("una denegación HTTP con cuerpo no JSON conserva su estado", async () => {
  const anterior = globalThis.fetch;
  globalThis.fetch = async () => new Response("<html>denegado</html>", { status: 403 });
  try { await assert.rejects(pedir("/api/sintetica"), (error) => error.status === 403 && !error.indeterminado); }
  finally { globalThis.fetch = anterior; }
});

test("RRHH informa del alta confirmada si la lectura posterior queda denegada, sin revelar el recibo", async () => {
  const raiz = raizFalsa();
  const llamadas = [];
  const cliente = async (ruta, opciones) => {
    llamadas.push([ruta, opciones?.method || "GET"]);
    if (opciones?.method === "POST") return { ...entrega, estado_entrega: "confirmada" };
    if (llamadas.length > 1) throw { status: 403 };
    return { limite: 50, peticiones: [entrega] };
  };
  await iniciarPeticionesCentroRRHH({ raiz, cliente });
  await raiz.pulsar({ accion: "abrir-alta-rrhh" });
  await raiz.pulsar({ accion: "confirmar-alta-rrhh" });
  assert.deepEqual(llamadas.map(([, method]) => method), ["GET", "POST", "GET"]);
  assert.match(raiz.innerHTML, /La operación se registró/);
  assert.match(raiz.innerHTML, /Acceso denegado/);
  verificarSinDatos(raiz);
});
