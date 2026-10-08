import assert from "node:assert/strict";
import test from "node:test";

import { iniciarPeticionCentro, iniciarPeticionesCentroRRHH, pedir } from "./peticiones-centro.js?v=20261008-alta-rpt-circular-v6";

const catalogos = {
  esquema: "vec.contratacion_temporal.catalogos_alta.v1",
  numero_expediente_moad: { referencia: "catalogo:numero:moad", version: 1, patron: "^[0-9]{4}/[1-9][0-9]{0,9}$", ejemplo: "2026/12345" },
  centros: [{ referencia: "cen_sintetico_001", etiqueta: "Centro sintético", contactos: [{ referencia: "con_sintetico_001", etiqueta: "Contacto sintético" }] }],
  categorias: [{ referencia: "cat_sintetica_001", etiqueta: "Categoría sintética", grupos_subgrupos: [{ clave: "C2", etiqueta: "C2" }] }],
  motivos: [{ clave: "sustitucion", etiqueta: "Sustitución" }], documentos: [],
};
const contexto = { actor: { referencia: "actor:sintetico:001", nombre: "Actor anterior sintético", cargo: "Cargo anterior", centro: "Centro anterior", puede_presentar: true, puede_ratificar: false }, catalogos };
const peticion = { referencia: "peticion:centro:sintetica", version: 2, estado: "ratificada", configuracion: { solicitante: { actor_ref: "actor:sintetico:001", puesto_ref: "puesto:sintetico:001" } }, solicitud: { centro_ref: "cen_sintetico_001", categoria_ref: "cat_sintetica_001", motivo_clave: "sustitucion", detalle: "Dato personal previo sintético", periodo: {} } };
const entrega = { peticion, estado_entrega: "preparada", recibo_alta: { numero_visible: "2026/12345", recibo_ref: "recibo:sintetico:anterior", expediente_ref: "expediente:sintetico:anterior", confirmada_en: "2026-09-06T08:00:00Z" } };

function raizFalsa() {
  const manejadores = new Map();
  return {
    innerHTML: "",
    isConnected: true,
    manejadores,
    setAttribute() {},
    insertAdjacentHTML(_posicion, contenido) { this.innerHTML = contenido + this.innerHTML; },
    querySelector(selector) { return { checked: true, value: selector === "[name=numero_expediente_moad]" ? "2026/12345" : "Motivo privado sintético" }; },
    querySelectorAll() { return []; },
    addEventListener(tipo, manejador) { manejadores.set(tipo, manejador); },
    async pulsar(dataset) {
      await manejadores.get("click")({ target: { dataset, closest: () => ({ dataset }) }, preventDefault() {} });
    },
  };
}

const catalogosConCausa = { ...catalogos,
  motivos: [{ ...catalogos.motivos[0], causa_fin: "reincorporacion_titular" }] };

test("Nueva petición no revive tras cambiar actor, denegar acceso o cerrar mientras carga el análisis", async () => {
  for (const desenlace of ["otro_actor", 401, 403, "cerrada"]) {
    const raiz = raizFalsa();
    let actor = { ...contexto, catalogos: catalogosConCausa };
    let estadoHTTP = 200;
    let resolverAnalisis;
    let preparaciones = 0, posts = 0;
    const vista = await iniciarPeticionCentro({ raiz,
      prepararAnalisis: () => { preparaciones++; return new Promise((resolver) => { resolverAnalisis = resolver; }); },
      cliente: async (ruta, opciones) => {
        if (opciones?.method === "POST") { posts++; throw new Error("sin efecto autorizado"); }
        if (estadoHTTP !== 200) throw { status: estadoHTTP };
        return ruta.endsWith("/contexto") ? actor : { peticiones: [] };
      },
    });
    const pendiente = raiz.pulsar({ accion: "nueva" });
    assert.equal(preparaciones, 1);
    assert.match(raiz.innerHTML, /Preparando el formulario/u);
    await raiz.pulsar({ accion: "nueva" });
    assert.equal(preparaciones, 1, "la pantalla ocupada no inicia otra preparación");
    if (desenlace === "otro_actor") {
      actor = { ...actor, actor: { ...actor.actor, referencia: "actor:otro", puede_presentar: false, puede_ratificar: true } };
      await vista.recargar();
    } else if (desenlace === "cerrada") raiz.isConnected = false;
    else { estadoHTTP = desenlace; await vista.recargar(); }
    resolverAnalisis({});
    await pendiente;
    assert.doesNotMatch(raiz.innerHTML, /data-ct-form/u);
    assert.equal(posts, 0);
    if ([401, 403].includes(desenlace)) assert.match(raiz.innerHTML, /Acceso denegado/u);
  }
});

test("fallo del catálogo de análisis conserva la bandeja y ofrece reintento sin denegar", async () => {
  const raiz = raizFalsa();
  let posts = 0;
  await iniciarPeticionCentro({ raiz,
    prepararAnalisis: async () => { throw new Error("catálogo temporalmente caído"); },
    cliente: async (ruta, opciones) => {
      if (opciones?.method === "POST") posts++;
      return ruta.endsWith("/contexto") ? { ...contexto, catalogos: catalogosConCausa } : { peticiones: [] };
    },
  });
  await raiz.pulsar({ accion: "nueva" });
  assert.match(raiz.innerHTML, /No se pudieron cargar las causas necesarias/u);
  assert.match(raiz.innerHTML, /data-accion="nueva"/u);
  assert.doesNotMatch(raiz.innerHTML, /Acceso denegado|data-ct-form/u);
  assert.equal(posts, 0);
});

function verificarSinDatos(raiz) {
  assert.doesNotMatch(raiz.innerHTML, /Actor anterior sintético|Cargo anterior|Centro anterior|Dato personal previo sintético|peticion:centro:sintetica|recibo:sintetico:anterior|expediente:sintetico:anterior/);
  assert.doesNotMatch(raiz.innerHTML, /data-accion="(?:nueva|abrir-alta-rrhh|recuperar-alta-anterior|confirmar-alta-rrhh|confirmar-presentar|confirmar-ratificar|reintentar(?:-alta-rrhh)?)"/);
}

test("RRHH puede corregir el número MOAD rechazado por el formato del catálogo", async () => {
  const raiz = raizFalsa();
  let numero = "2026/OTRO";
  raiz.querySelector = (selector) => ({ checked: true, value: selector === "[name=numero_expediente_moad]" ? numero : "" });
  const comandos = [];
  const cliente = async (ruta, opciones) => {
    if (ruta.endsWith("/catalogos-alta")) return catalogos;
    if (opciones?.method !== "POST") return { limite: 50, peticiones: [entrega] };
    comandos.push(opciones.cuerpo);
    if (comandos.length === 1) throw { status: 422 };
    return { peticion, estado_entrega: "confirmada", recibo_alta: { ...entrega.recibo_alta, numero_visible: numero } };
  };
  await iniciarPeticionesCentroRRHH({ raiz, cliente });
  await raiz.pulsar({ accion: "abrir-alta-rrhh" });
  await raiz.pulsar({ accion: "confirmar-alta-rrhh" });
  assert.match(raiz.innerHTML, /Revise el número de MOAD/);
  assert.match(raiz.innerHTML, /value="2026\/OTRO"/);
  assert.match(raiz.innerHTML, /data-accion="confirmar-alta-rrhh"/);
  assert.doesNotMatch(raiz.innerHTML, /data-accion="reintentar-alta-rrhh"/);
  numero = "2026/12345";
  await raiz.pulsar({ accion: "confirmar-alta-rrhh" });
  assert.deepEqual(comandos.map((c) => c.numero_expediente_moad), ["2026/OTRO", "2026/12345"]);
});

test("RRHH recupera una alta anterior preparada sin atribuirle número MOAD", async () => {
  const raiz = raizFalsa();
  const anterior = { ...entrega, recibo_alta: { ...entrega.recibo_alta, numero_visible: "2026/CT-0001" } };
  const comandos = [];
  const cliente = async (_ruta, opciones) => {
    if (opciones?.method !== "POST") return { limite: 50, peticiones: [anterior] };
    comandos.push(opciones.cuerpo);
    return { peticion, estado_entrega: "confirmada", recibo_alta: anterior.recibo_alta };
  };
  await iniciarPeticionesCentroRRHH({ raiz, cliente });
  assert.match(raiz.innerHTML, /data-accion="recuperar-alta-anterior"/);
  await raiz.pulsar({ accion: "recuperar-alta-anterior" });
  assert.equal(comandos.length, 1);
  assert.equal(Object.hasOwn(comandos[0], "numero_expediente_moad"), false);
  assert.match(raiz.innerHTML, /2026\/CT-0001/);
  assert.doesNotMatch(raiz.innerHTML, /Número de expediente MOAD/);
});

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
    if (ruta.endsWith("/catalogos-alta")) return catalogos;
    if (llamadas.length > 2) throw { status: 403 };
    return { limite: 50, peticiones: [entrega] };
  };
  await iniciarPeticionesCentroRRHH({ raiz, cliente });
  await raiz.pulsar({ accion: "abrir-alta-rrhh" });
  await raiz.pulsar({ accion: "confirmar-alta-rrhh" });
  assert.deepEqual(llamadas.map(([, method]) => method), ["GET", "GET", "POST", "GET"]);
  assert.match(raiz.innerHTML, /La operación se registró/);
  assert.match(raiz.innerHTML, /Acceso denegado/);
  verificarSinDatos(raiz);
});

test("centro descarta el comando incierto tras POST 503 y reintento 403", async () => {
  const raiz = raizFalsa();
  const ratificable = { ...peticion, version: 1, estado: "pendiente_ratificacion" };
  const ratificador = { ...contexto, actor: { ...contexto.actor, puede_presentar: false, puede_ratificar: true } };
  const llamadas = [];
  let post = 0;
  let antesDeSalir;
  const anterior = globalThis.addEventListener;
  globalThis.addEventListener = (tipo, manejador) => { if (tipo === "beforeunload") antesDeSalir = manejador; };
  const cliente = async (ruta, opciones) => {
    llamadas.push([ruta, opciones?.method || "GET"]);
    if (opciones?.method === "POST") throw { status: ++post === 1 ? 503 : 403 };
    return ruta.endsWith("/contexto") ? ratificador : { peticiones: [ratificable] };
  };
  try {
    const vista = await iniciarPeticionCentro({ raiz, cliente });
    await raiz.pulsar({ seleccionar: ratificable.referencia });
    await raiz.pulsar({ accion: "abrir-ratificacion" });
    await raiz.pulsar({ accion: "confirmar-ratificar" });
    assert.match(raiz.innerHTML, /Reintentar la misma operación/);
    let intercepciones = 0;
    antesDeSalir({ preventDefault() { intercepciones += 1; }, returnValue: undefined });
    assert.equal(intercepciones, 1);
    await raiz.pulsar({ accion: "reintentar" });
    verificarSinDatos(raiz);
    assert.match(raiz.innerHTML, /resultado sigue sin confirmarse/);
    assert.doesNotMatch(raiz.innerHTML, /Motivo privado sintético/);
    antesDeSalir({ preventDefault() { intercepciones += 1; }, returnValue: undefined });
    assert.equal(intercepciones, 1);
    const previas = llamadas.length;
    await vista.recargar();
    assert.equal(llamadas.length, previas + 2);
    assert.match(raiz.innerHTML, /Resultado pendiente de comprobación/);
    verificarSinDatos(raiz);
    await raiz.pulsar({ accion: "confirmar-ratificar" });
    await raiz.pulsar({ accion: "reintentar" });
    assert.equal(post, 2);
  } finally { globalThis.addEventListener = anterior; }
});

test("RRHH descarta el comando incierto tras POST 503 y reintento 401", async () => {
  const raiz = raizFalsa();
  const llamadas = [];
  let post = 0;
  const cliente = async (ruta, opciones) => {
    llamadas.push([ruta, opciones?.method || "GET"]);
    if (opciones?.method === "POST") throw { status: ++post === 1 ? 503 : 401 };
    return ruta.endsWith("/catalogos-alta") ? catalogos : { limite: 50, peticiones: [entrega] };
  };
  const vista = await iniciarPeticionesCentroRRHH({ raiz, cliente });
  await raiz.pulsar({ accion: "abrir-alta-rrhh" });
  await raiz.pulsar({ accion: "confirmar-alta-rrhh" });
  assert.match(raiz.innerHTML, /Reintentar la misma operación/);
  await raiz.pulsar({ accion: "reintentar-alta-rrhh" });
  verificarSinDatos(raiz);
  assert.match(raiz.innerHTML, /resultado sigue sin confirmarse/);
  const previas = llamadas.length;
  await vista.recargar();
  assert.equal(llamadas.length, previas + 1);
  assert.match(raiz.innerHTML, /Resultado pendiente de comprobación/);
  verificarSinDatos(raiz);
  await raiz.pulsar({ accion: "confirmar-alta-rrhh" });
  await raiz.pulsar({ accion: "reintentar-alta-rrhh" });
  assert.equal(post, 2);
});

test("la recuperación y la denegación notifican el contexto vigente a las secciones", async () => {
  const cambios = [];
  let fallo = 503;
  const capacidades = { incorporaciones: true, cancelaciones: true };
  const vista = await iniciarPeticionCentro({ raiz: raizFalsa(), alCambiarContexto: (c) => cambios.push(c),
    cliente: async (ruta) => {
      if (fallo) throw { status: fallo };
      return ruta.endsWith("/contexto") ? { ...contexto, capacidades } : { peticiones: [] };
    },
  });
  assert.equal(cambios.at(-1), undefined, "un fallo no publica capacidades anteriores");
  fallo = 0;
  await vista.recargar();
  assert.equal(cambios.at(-1), capacidades);
  fallo = 403;
  await vista.recargar();
  assert.equal(cambios.at(-1), undefined, "denegar retira ambas secciones");
});
