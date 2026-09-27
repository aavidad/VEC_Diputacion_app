import test from "node:test";
import assert from "node:assert/strict";
import { crearClientePlantillasRRHH, ErrorPlantillasRRHH, RUTA_RRHH_PLANTILLAS,
  RUTA_RRHH_PLANTILLAS_ENTRADAS, RUTA_RRHH_PLANTILLAS_PUBLICAR } from "./rrhh-plantillas-cliente.js";
import { montarRRHHPlantillas, prepararEntradaPlantilla } from "./rrhh-plantillas-vista.js";
import { MENSAJES_RRHH_PLANTILLAS_ES } from "./rrhh-plantillas-i18n.js";

const instante = "2026-09-28T09:00:00Z";
const catalogo = Object.freeze({
  id: "vec.contratacion_temporal.plantillas_documentos", fuente_ref: "fuente:rrhh:ejemplo",
  version: 3, revision: 2, estado: "borrador", huella_sha256: "a".repeat(64),
  entradas: [{ clave: "modelo_nuevo", etiqueta: "Modelo nuevo", descripcion: "Modelo de prueba", orden: 11,
    vigente_desde: instante, atributos: { titulo: "Título original", "parrafo.01": "Primer párrafo", "parrafo.02": "Segundo párrafo", origen: "desarrollo" } }],
});
const respuesta = (valor, estado = 200) => new Response(JSON.stringify(valor), {
  status: estado, headers: { "Content-Type": "application/json; charset=utf-8" },
});

test("cliente consulta sin credenciales persistidas y conserva versión publicada/borrador", async () => {
  let peticion;
  const cliente = crearClientePlantillasRRHH({ fetchImpl: async (ruta, opciones) => {
    peticion = { ruta, opciones };
    return respuesta({ borrador: catalogo, publicado: null, puede_editar: true, puede_publicar: false });
  } });
  const datos = await cliente.consultar();
  assert.equal(datos.borrador.version, 3);
  assert.equal(datos.puede_editar, true);
  assert.equal(datos.puede_publicar, false);
  assert.equal(peticion.ruta, RUTA_RRHH_PLANTILLAS);
  assert.equal(peticion.opciones.method, "GET");
  assert.equal(peticion.opciones.credentials, "same-origin");
  assert.equal(peticion.opciones.cache, "no-store");
  assert.equal(peticion.opciones.body, undefined);
  const sinPermisos = crearClientePlantillasRRHH({ fetchImpl: async () => respuesta({ borrador: catalogo, publicado: null }) });
  await assert.rejects(sinPermisos.consultar(), (error) => error instanceof ErrorPlantillasRRHH && error.codigo === "respuesta_incompatible");
});

test("cliente solo confirma un cambio con catálogo, versión y recibo válidos", async () => {
  let peticion;
  const cliente = crearClientePlantillasRRHH({ fetchImpl: async (ruta, opciones) => {
    peticion = { ruta, opciones };
    return respuesta({ catalogo: { ...catalogo, revision: 3 }, recibo: {
      recibo_ref: "recibo:plantillas:uno", registrado_en: instante, estado_replay: "registrado",
    } }, 201);
  } });
  const solicitud = { clave_idempotencia: "8cf3f53e-b1c0-4bad-9bc1-08dde1b32789",
    version_esperada: 3, revision_esperada: 2, motivo: "Ajuste RRHH", fuente_ref: "fuente:rrhh:ejemplo",
    entrada: catalogo.entradas[0] };
  const guardado = await cliente.guardar(solicitud);
  assert.equal(guardado.catalogo.revision, 3);
  assert.equal(guardado.recibo.recibo_ref, "recibo:plantillas:uno");
  assert.equal(peticion.ruta, RUTA_RRHH_PLANTILLAS_ENTRADAS);
  assert.deepEqual(JSON.parse(peticion.opciones.body), solicitud);
  assert.equal(peticion.opciones.headers.get("Content-Type"), "application/json");
  const incompatible = crearClientePlantillasRRHH({ fetchImpl: async () => respuesta({
    catalogo: { ...catalogo, revision: 3 }, recibo: { recibo_ref: "", registrado_en: instante, estado_replay: "registrado" },
  }, 201) });
  await assert.rejects(incompatible.guardar(solicitud), (error) => error instanceof ErrorPlantillasRRHH && error.resultadoIndeterminado);
});

test("cliente distingue conflicto previo y resultado indeterminado", async () => {
  const solicitud = { clave_idempotencia: "8cf3f53e-b1c0-4bad-9bc1-08dde1b32789",
    version_esperada: 3, revision_esperada: 2, motivo: "Ajuste RRHH", fuente_ref: "fuente:rrhh:ejemplo",
    entrada: catalogo.entradas[0] };
  const conflicto = crearClientePlantillasRRHH({ fetchImpl: async () => respuesta({ codigo: "version_en_conflicto" }, 409) });
  await assert.rejects(conflicto.guardar(solicitud), (error) => error.estado === 409 && !error.resultadoIndeterminado);
  const sinRespuesta = crearClientePlantillasRRHH({ fetchImpl: async () => { throw new TypeError("red caída"); } });
  await assert.rejects(sinRespuesta.guardar(solicitud), (error) => error.resultadoIndeterminado);
});

test("publicación exige aprobación y solo confirma catálogo publicado con recibo", async () => {
  let peticion;
  const cliente = crearClientePlantillasRRHH({ fetchImpl: async (ruta, opciones) => {
    peticion = { ruta, opciones };
    return respuesta({ catalogo: { ...catalogo, estado: "publicado" }, recibo: {
      recibo_ref: "recibo:plantillas:publicacion", registrado_en: instante, estado_replay: "registrado",
    } }, 201);
  } });
  const solicitud = { clave_idempotencia: "8cf3f53e-b1c0-4bad-9bc1-08dde1b32789",
    version_esperada: 3, revision_esperada: 2, motivo: "Modelo aprobado", aprobacion_ref: "aprobacion:rrhh:uno" };
  assert.equal((await cliente.publicar(solicitud)).catalogo.estado, "publicado");
  assert.equal(peticion.ruta, RUTA_RRHH_PLANTILLAS_PUBLICAR);
  assert.deepEqual(JSON.parse(peticion.opciones.body), solicitud);
  await assert.rejects(cliente.publicar({ ...solicitud, aprobacion_ref: "" }), TypeError);
});

test("formulario conserva metadatos y el instante exacto al editar", () => {
  const datos = new FormData();
  for (const [clave, valor] of Object.entries({ etiqueta: "Modelo revisado", descripcion: "Modelo de prueba", titulo: "Título nuevo",
    orden: "12", vigente_desde: "2026-09-28", fuente_ref: "fuente:rrhh:ejemplo", modalidades: "*", firmantes: "",
    requiere_accion: "", motivo: "Actualización del texto" })) datos.set(clave, valor);
  datos.append("parrafo", "Nuevo párrafo");
  const guardado = prepararEntradaPlantilla(datos, catalogo.entradas[0]);
  assert.equal(guardado.entrada.vigente_desde, instante);
  assert.deepEqual(guardado.entrada.atributos, {
    titulo: "Título nuevo", origen: "desarrollo", modalidades: "*", "parrafo.01": "Nuevo párrafo",
  });
  assert.equal(guardado.fuente_ref, "fuente:rrhh:ejemplo");
  assert.equal(guardado.entrada.clave, "modelo_nuevo");
  datos.set("clave", "etiquetas");
  assert.throws(() => prepararEntradaPlantilla(datos), /clave_invalida/u);
});

test("los textos visibles de la vista están en el catálogo i18n", () => {
  for (const clave of ["plantillas_rrhh_titulo", "plantillas_rrhh_ayuda", "plantillas_rrhh_recibo",
    "plantillas_rrhh_indeterminado", "plantillas_rrhh_fuente"]) {
    assert.equal(typeof MENSAJES_RRHH_PLANTILLAS_ES[clave], "string");
  }
});

test("vista separa borrador y publicación, escapa datos y oculta ayuda hasta pulsar ?", async () => {
  const eventos = new Map();
  const raiz = {
    innerHTML: "",
    addEventListener(nombre, gestor) { eventos.set(nombre, gestor); },
    removeEventListener(nombre) { eventos.delete(nombre); },
    contains() { return true; },
    replaceChildren() { this.innerHTML = ""; },
  };
  const cliente = {
    async consultar() { return { borrador: { ...catalogo, entradas: [{ ...catalogo.entradas[0], etiqueta: "<modelo>" }] },
      publicado: null, puede_editar: true, puede_publicar: true }; },
    async guardar() { throw new Error("no se llama"); },
    async publicar() { throw new Error("no se llama"); },
  };
  const vista = montarRRHHPlantillas({ raiz, cliente });
  await new Promise((resolver) => setImmediate(resolver));
  assert.match(raiz.innerHTML, /<form data-plantillas-publicar/u);
  assert.match(raiz.innerHTML, /&lt;modelo&gt;/u);
  assert.doesNotMatch(raiz.innerHTML, /<modelo>/u);
  assert.match(raiz.innerHTML, /id="rrhh-plantillas-ayuda"[^>]*hidden/u);
  assert.match(raiz.innerHTML, /aria-controls="rrhh-plantillas-ayuda">\?<\/button>/u);
  eventos.get("click")({ target: { closest() { return { dataset: { plantillasAccion: "ayuda" } }; } } });
  assert.doesNotMatch(raiz.innerHTML, /id="rrhh-plantillas-ayuda"[^>]*hidden/u);
  vista.desmontar();
  assert.equal(raiz.innerHTML, "");
});

test("sin concesiones positivas la consulta es solo lectura y no ofrece acciones", async () => {
  const raiz = { innerHTML: "", addEventListener() {}, removeEventListener() {}, replaceChildren() {} };
  const cliente = { async consultar() { return { borrador: catalogo, publicado: null, puede_editar: false, puede_publicar: false }; },
    async guardar() { throw new Error("no se llama"); }, async publicar() { throw new Error("no se llama"); } };
  montarRRHHPlantillas({ raiz, cliente });
  await new Promise((resolver) => setImmediate(resolver));
  assert.match(raiz.innerHTML, /La configuración consultada no admite edición/u);
  assert.doesNotMatch(raiz.innerHTML, /data-plantillas-accion="nueva"|data-plantillas-accion="editar"|data-plantillas-publicar/u);
});
