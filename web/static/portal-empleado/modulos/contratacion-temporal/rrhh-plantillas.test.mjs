import test from "node:test";
import assert from "node:assert/strict";
import { crearClientePlantillasRRHH, ErrorPlantillasRRHH, RUTA_RRHH_PLANTILLAS,
  RUTA_RRHH_PLANTILLAS_ENTRADAS, RUTA_RRHH_PLANTILLAS_PUBLICAR } from "./rrhh-plantillas-cliente.js?v=20261008-alta-rpt-circular-v5";
import { montarRRHHPlantillas, prepararEntradaPlantilla } from "./rrhh-plantillas-vista.js?v=20261008-alta-rpt-circular-v5";
import { MENSAJES_RRHH_PLANTILLAS_ES } from "./rrhh-plantillas-i18n.js";
import { crearTraductorContratacionTemporal } from "./i18n.js?v=20261008-alta-rpt-circular-v5";

const instante = "2026-09-28T09:00:00Z";
const reciboEdicion = "recibo:ad6eaa70-bc5f-4a27-90b4-5bf02043d021";
const reciboPublicacion = "recibo:f805f3e8-4d4f-48d3-8b01-318c9fe66ed4";
const catalogo = Object.freeze({
  id: "vec.contratacion_temporal.plantillas_documentos", fuente_ref: "fuente:rrhh:ejemplo",
  version: 3, revision: 2, estado: "borrador", huella_sha256: "a".repeat(64),
  entradas: [{ clave: "modelo_nuevo", etiqueta: "Modelo nuevo", descripcion: "Modelo de prueba", orden: 11,
    vigente_desde: instante, atributos: { titulo: "Título original", "parrafo.01": "Primer párrafo", "parrafo.02": "Segundo párrafo", origen: "desarrollo" } }],
});
const reciboCambio = (solicitud, resultado, operacion, referencia, estado_replay = "registrado") => ({
  recibo_ref: referencia, clave_idempotencia: solicitud.clave_idempotencia, operacion,
  version: resultado.version, revision: resultado.revision,
  catalogo_huella_sha256: resultado.huella_sha256, registrado_en: instante, estado_replay,
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

test("consulta acepta catálogo legal de más de 4 MiB y corta lectura por encima de 17 MB", async () => {
  const atributos = Object.fromEntries(Array.from({ length: 80 }, (_, indice) =>
    [`parrafo.${String(indice).padStart(2, "0")}`, "a".repeat(65_536)]));
  const grande = { ...catalogo, entradas: [{ ...catalogo.entradas[0], atributos }] };
  const cuerpo = JSON.stringify({ borrador: grande, publicado: null, puede_editar: false, puede_publicar: false });
  const bytes = new TextEncoder().encode(cuerpo);
  assert.ok(bytes.byteLength > 4 * 1024 * 1024 && bytes.byteLength < 17_000_000);
  const clienteGrande = crearClientePlantillasRRHH({ fetchImpl: async () => new Response(bytes, {
    status: 200, headers: { "Content-Type": "application/json; charset=utf-8" },
  }) });
  assert.equal((await clienteGrande.consultar()).borrador.entradas[0].atributos["parrafo.79"].length, 65_536);

  const excesiva = new ReadableStream({ start(controlador) {
    controlador.enqueue(new Uint8Array(9_000_000));
    controlador.enqueue(new Uint8Array(8_000_001));
    controlador.close();
  } });
  const clienteExcesivo = crearClientePlantillasRRHH({ fetchImpl: async () => new Response(excesiva, {
    status: 200, headers: { "Content-Type": "application/json; charset=utf-8" },
  }) });
  await assert.rejects(clienteExcesivo.consultar(), (error) =>
    error instanceof ErrorPlantillasRRHH && error.codigo === "respuesta_incompatible");
});

test("señal ya cancelada impide la petición de plantillas", async () => {
  const controlador = new AbortController();
  controlador.abort();
  const cliente = crearClientePlantillasRRHH({ fetchImpl: async () => assert.fail("red inesperada") });
  const t = crearTraductorContratacionTemporal(MENSAJES_RRHH_PLANTILLAS_ES);
  await assert.rejects(cliente.consultar({ signal: controlador.signal }), (error) =>
    error.name === "AbortError" && error.message === t("plantillas_rrhh_peticion_cancelada"));
});

test("abortar durante la lectura streaming no entrega catálogo parcial", async () => {
  const controlador = new AbortController();
  let tramo = 0;
  const cuerpo = new ReadableStream({ pull(lector) {
    if (tramo++ === 0) lector.enqueue(new TextEncoder().encode('{"borrador":'));
    else { controlador.abort(); lector.enqueue(new TextEncoder().encode("{}")); lector.close(); }
  } });
  const cliente = crearClientePlantillasRRHH({ fetchImpl: async () => new Response(cuerpo, {
    status: 200, headers: { "Content-Type": "application/json; charset=utf-8" },
  }) });
  const t = crearTraductorContratacionTemporal(MENSAJES_RRHH_PLANTILLAS_ES);
  await assert.rejects(cliente.consultar({ signal: controlador.signal }), (error) =>
    error.name === "AbortError" && error.message === t("plantillas_rrhh_lectura_cancelada"));
});

test("cliente solo confirma un cambio con catálogo, versión y recibo válidos", async () => {
  const solicitud = { clave_idempotencia: "8cf3f53e-b1c0-4bad-9bc1-08dde1b32789",
    version_esperada: 3, revision_esperada: 2, motivo: "Ajuste RRHH", fuente_ref: "fuente:rrhh:ejemplo",
    entrada: catalogo.entradas[0] };
  const resultado = { ...catalogo, revision: 3 };
  const recibo = reciboCambio(solicitud, resultado, "editar", reciboEdicion);
  let peticion;
  const cliente = crearClientePlantillasRRHH({ fetchImpl: async (ruta, opciones) => {
    peticion = { ruta, opciones };
    return respuesta({ catalogo: resultado, recibo }, 201);
  } });
  const guardado = await cliente.guardar(solicitud);
  assert.equal(guardado.catalogo.revision, 3);
  assert.equal(guardado.recibo.recibo_ref, reciboEdicion);
  assert.equal(peticion.ruta, RUTA_RRHH_PLANTILLAS_ENTRADAS);
  assert.deepEqual(JSON.parse(peticion.opciones.body), solicitud);
  assert.equal(peticion.opciones.headers.get("Content-Type"), "application/json");
  for (const referencia of ["", "recibo:plantillas:uno", [reciboEdicion], reciboEdicion.toUpperCase(),
    `${reciboEdicion}a`, `${reciboEdicion}\n`, reciboEdicion.slice(0, -1),
    "recibo:ad6eaa70-bc5f-4a27-90b4-5bf02043d02g", `recibo:${"-".repeat(36)}`,
    "recibo:ad6eaa70bc5f4a2790b45bf02043d021----",
    "recibo:ad6eaa70-bc5f-5a27-90b4-5bf02043d021",
    "recibo:ad6eaa70-bc5f-4a27-70b4-5bf02043d021"]) {
    const incompatible = crearClientePlantillasRRHH({ fetchImpl: async () => respuesta({
      catalogo: resultado, recibo: { ...recibo, recibo_ref: referencia },
    }, 201) });
    await assert.rejects(incompatible.guardar(solicitud), (error) => error instanceof ErrorPlantillasRRHH
      && error.codigo === "respuesta_incompatible" && error.resultadoIndeterminado, referencia);
  }
  for (const [estado, estadoReplay] of [[201, "registrado"], [200, "replay"]]) {
    for (const catalogoMalo of [undefined, { ...resultado, revision: "3" }]) {
      const catalogoIncompatible = crearClientePlantillasRRHH({ fetchImpl: async () => respuesta({
        catalogo: catalogoMalo, recibo: { ...recibo, estado_replay: estadoReplay },
      }, estado) });
      await assert.rejects(catalogoIncompatible.guardar(solicitud), (error) => error instanceof ErrorPlantillasRRHH
        && error.codigo === "respuesta_incompatible" && error.resultadoIndeterminado);
    }
  }
  const replay = crearClientePlantillasRRHH({ fetchImpl: async () => respuesta({
    catalogo: resultado, recibo: { ...recibo, estado_replay: "replay" },
  }, 200) });
  assert.equal((await replay.guardar(solicitud)).recibo.recibo_ref, reciboEdicion);
  for (const [campo, valor] of [["clave_idempotencia", "11111111-1111-4111-8111-111111111111"],
    ["operacion", "publicar"], ["version", 4], ["revision", 4],
    ["catalogo_huella_sha256", "b".repeat(64)], ["estado_replay", "replay"]]) {
    const ajeno = crearClientePlantillasRRHH({ fetchImpl: async () => respuesta({
      catalogo: resultado, recibo: { ...recibo, [campo]: valor },
    }, 201) });
    await assert.rejects(ajeno.guardar(solicitud), (error) => error instanceof ErrorPlantillasRRHH
      && error.codigo === "respuesta_incompatible" && error.resultadoIndeterminado, campo);
  }
  const estadoAjeno = crearClientePlantillasRRHH({ fetchImpl: async () => respuesta({
    catalogo: resultado, recibo,
  }, 200) });
  await assert.rejects(estadoAjeno.guardar(solicitud), (error) => error instanceof ErrorPlantillasRRHH
    && error.resultadoIndeterminado);
  const catalogoAjeno = crearClientePlantillasRRHH({ fetchImpl: async () => respuesta({
    catalogo: { ...resultado, huella_sha256: "b".repeat(64) }, recibo,
  }, 201) });
  await assert.rejects(catalogoAjeno.guardar(solicitud), (error) => error instanceof ErrorPlantillasRRHH
    && error.resultadoIndeterminado);
  const versionAjena = crearClientePlantillasRRHH({ fetchImpl: async () => respuesta({
    catalogo: { ...resultado, version: 4, revision: 1 }, recibo: { ...recibo, version: 4, revision: 1 },
  }, 201) });
  await assert.rejects(versionAjena.guardar(solicitud), (error) => error instanceof ErrorPlantillasRRHH
    && error.resultadoIndeterminado);
  const solicitudNueva = { ...solicitud, revision_esperada: 0,
    clave_idempotencia: "3dd7b65c-3f2f-4bda-9a7f-f5e6f5248410" };
  const nuevaVersion = { ...resultado, version: 4, revision: 1 };
  const nueva = crearClientePlantillasRRHH({ fetchImpl: async () => respuesta({
    catalogo: nuevaVersion, recibo: reciboCambio(solicitudNueva, nuevaVersion, "editar", reciboEdicion),
  }, 201) });
  assert.equal((await nueva.guardar(solicitudNueva)).catalogo.version, 4);
});

test("cliente distingue conflicto previo y resultado indeterminado", async () => {
  const solicitud = { clave_idempotencia: "8cf3f53e-b1c0-4bad-9bc1-08dde1b32789",
    version_esperada: 3, revision_esperada: 2, motivo: "Ajuste RRHH", fuente_ref: "fuente:rrhh:ejemplo",
    entrada: catalogo.entradas[0] };
  const conflicto = crearClientePlantillasRRHH({ fetchImpl: async () => respuesta({ codigo: "version_en_conflicto" }, 409) });
  await assert.rejects(conflicto.guardar(solicitud), (error) => error.estado === 409 && !error.resultadoIndeterminado);
  const denegado = crearClientePlantillasRRHH({ fetchImpl: async () => respuesta({ codigo: "acceso_denegado" }, 403) });
  await assert.rejects(denegado.guardar(solicitud), (error) => error.estado === 403 && !error.resultadoIndeterminado);
  const sinRespuesta = crearClientePlantillasRRHH({ fetchImpl: async () => { throw new TypeError("red caída"); } });
  await assert.rejects(sinRespuesta.guardar(solicitud), (error) => error.resultadoIndeterminado);
});

test("publicación exige aprobación y solo confirma catálogo publicado con recibo", async () => {
  const solicitud = { clave_idempotencia: "8cf3f53e-b1c0-4bad-9bc1-08dde1b32789",
    version_esperada: 3, revision_esperada: 2, motivo: "Modelo aprobado", aprobacion_ref: "aprobacion:rrhh:uno" };
  const resultado = { ...catalogo, estado: "publicado" };
  const recibo = reciboCambio(solicitud, resultado, "publicar", reciboPublicacion);
  let peticion;
  const cliente = crearClientePlantillasRRHH({ fetchImpl: async (ruta, opciones) => {
    peticion = { ruta, opciones };
    return respuesta({ catalogo: resultado, recibo }, 201);
  } });
  assert.equal((await cliente.publicar(solicitud)).catalogo.estado, "publicado");
  assert.equal(peticion.ruta, RUTA_RRHH_PLANTILLAS_PUBLICAR);
  assert.deepEqual(JSON.parse(peticion.opciones.body), solicitud);
  await assert.rejects(cliente.publicar({ ...solicitud, aprobacion_ref: "" }), TypeError);
  const incompatible = crearClientePlantillasRRHH({ fetchImpl: async () => respuesta({
    catalogo: resultado, recibo: { ...recibo, recibo_ref: "recibo:plantillas:publicacion" },
  }, 201) });
  await assert.rejects(incompatible.publicar(solicitud), (error) => error instanceof ErrorPlantillasRRHH
    && error.codigo === "respuesta_incompatible" && error.resultadoIndeterminado);
  for (const [estado, estadoReplay] of [[201, "registrado"], [200, "replay"]]) {
    for (const catalogoMalo of [undefined, { ...resultado, revision: "2" }]) {
      const catalogoIncompatible = crearClientePlantillasRRHH({ fetchImpl: async () => respuesta({
        catalogo: catalogoMalo, recibo: { ...recibo, estado_replay: estadoReplay },
      }, estado) });
      await assert.rejects(catalogoIncompatible.publicar(solicitud), (error) => error instanceof ErrorPlantillasRRHH
        && error.codigo === "respuesta_incompatible" && error.resultadoIndeterminado);
    }
  }
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
    "plantillas_rrhh_indeterminado", "plantillas_rrhh_fuente", "plantillas_rrhh_recuperar_resultado",
    "plantillas_rrhh_recuperando", "plantillas_rrhh_recuperacion_pendiente",
    "plantillas_rrhh_recuperacion_conflicto", "plantillas_rrhh_recuperacion_denegada"]) {
    assert.equal(typeof MENSAJES_RRHH_PLANTILLAS_ES[clave], "string");
  }
});

test("vista separa borrador y publicación, escapa datos y oculta ayuda hasta pulsar ?", async () => {
  const eventos = new Map();
  let focoAyuda = 0;
  const raiz = {
    innerHTML: "",
    addEventListener(nombre, gestor) { eventos.set(nombre, gestor); },
    removeEventListener(nombre) { eventos.delete(nombre); },
    contains() { return true; },
    querySelector(selector) { return selector === '[data-plantillas-accion="ayuda"]'
      ? { focus() { focoAyuda++; } } : null; },
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
  assert.doesNotMatch(raiz.innerHTML, /Configuración de los borradores de Contratación temporal|Seleccione una fila para revisar o editar su definición/u);
  assert.doesNotMatch(raiz.innerHTML, /vec\.contratacion_temporal\.plantillas_documentos|<small>modelo_nuevo<\/small>/u);
  assert.match(raiz.innerHTML, /Huella SHA-256/u);
  assert.match(raiz.innerHTML, new RegExp("a".repeat(64)));
  eventos.get("click")({ target: { closest() { return { dataset: { plantillasAccion: "editar", clave: "modelo_nuevo" } }; } } });
  assert.match(raiz.innerHTML, /<form data-plantillas-form/u);
  assert.doesNotMatch(raiz.innerHTML, /name="clave"/u);
  eventos.get("click")({ target: { closest() { return { dataset: { plantillasAccion: "ayuda" } }; } } });
  assert.doesNotMatch(raiz.innerHTML, /id="rrhh-plantillas-ayuda"[^>]*hidden/u);
  assert.equal(focoAyuda, 1);
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

test("cancelar y recargar no pierden la clave ni ocultan un registro indeterminado", async () => {
  const eventos = new Map();
  const raiz = { innerHTML: "", contains: () => true,
    addEventListener(nombre, gestor) { eventos.set(nombre, gestor); },
    removeEventListener(nombre) { eventos.delete(nombre); },
    querySelector: () => null, replaceChildren() { this.innerHTML = ""; } };
  let consultas = 0, guardados = 0, clavesGeneradas = 0, rechazar;
  const pendiente = new Promise((_, rechazo) => { rechazar = rechazo; });
  const cliente = { async consultar() { consultas++; return { borrador: catalogo, publicado: null,
    puede_editar: true, puede_publicar: false }; },
  guardar() { guardados++; return pendiente; },
  async publicar() { assert.fail("publicación inesperada"); } };
  const vista = montarRRHHPlantillas({ raiz, cliente,
    generarClave: () => { clavesGeneradas++; return "8cf3f53e-b1c0-4bad-9bc1-08dde1b32789"; } });
  await new Promise((resolver) => setImmediate(resolver));
  const pulsar = (accion, clave) => eventos.get("click")({ target: { closest: () =>
    ({ dataset: { plantillasAccion: accion, clave } }) } });
  pulsar("editar", "modelo_nuevo");
  const formulario = new FormData();
  for (const [clave, valor] of Object.entries({ etiqueta: "Modelo nuevo", descripcion: "Modelo de prueba",
    titulo: "Título original", orden: "11", vigente_desde: "2026-09-28", fuente_ref: "fuente:rrhh:ejemplo",
    modalidades: "*", firmantes: "", requiere_accion: "", motivo: "Ajuste del texto" })) formulario.set(clave, valor);
  formulario.append("parrafo", "Primer párrafo");
  formulario.matches = (selector) => selector === "[data-plantillas-form]" || selector.includes("[data-plantillas-form]");
  formulario.querySelector = () => null;
  formulario.querySelectorAll = () => [];
  formulario.insertAdjacentHTML = () => {};
  const enviar = () => eventos.get("submit")({ target: formulario, preventDefault() {} });
  enviar();
  assert.equal(guardados, 1);
  pulsar("cancelar"); // Incluso un clic programático durante el POST debe ser inocuo.
  rechazar(Object.assign(new Error("respuesta indeterminada"), { resultadoIndeterminado: true }));
  await new Promise((resolver) => setImmediate(resolver));
  assert.match(raiz.innerHTML, /El resultado del cambio es indeterminado/u);
  assert.match(raiz.innerHTML, /data-plantillas-accion="cancelar" disabled/u);
  pulsar("cancelar");
  pulsar("recargar");
  pulsar("nueva");
  enviar();
  assert.match(raiz.innerHTML, /El resultado del cambio es indeterminado/u);
  assert.equal(consultas, 1);
  assert.equal(guardados, 1);
  assert.equal(clavesGeneradas, 1);
  vista.desmontar();
});

function vistaConEventos(cliente) {
  const eventos = new Map();
  let focosRecuperacion = 0;
  let focosRecibo = 0;
  const raiz = { innerHTML: "", contains: () => true,
    addEventListener(nombre, gestor) { eventos.set(nombre, gestor); },
    removeEventListener(nombre) { eventos.delete(nombre); },
    querySelector: (selector) => selector === '[data-plantillas-accion="recuperar"]'
      ? { focus() { focosRecuperacion++; } }
      : selector === ".rrhh-plantillas-recibo" ? { focus() { focosRecibo++; }, remove() {} } : null,
    replaceChildren() { this.innerHTML = ""; } };
  const vista = montarRRHHPlantillas({ raiz, cliente,
    generarClave: () => "8cf3f53e-b1c0-4bad-9bc1-08dde1b32789" });
  return { raiz, vista, focosRecuperacion: () => focosRecuperacion, focosRecibo: () => focosRecibo,
    pulsar: (accion, clave) => eventos.get("click")({ target: { closest: () =>
      ({ dataset: { plantillasAccion: accion, clave } }) } }),
    enviar: (formulario) => eventos.get("submit")({ target: formulario, preventDefault() {} }) };
}

function formularioPrueba(campos, tipo) {
  const formulario = new FormData();
  for (const [clave, valor] of Object.entries(campos)) formulario.set(clave, valor);
  formulario.matches = (selector) => selector.split(/,\s*/u).includes(`[data-plantillas-${tipo}]`);
  formulario.querySelector = () => null;
  formulario.querySelectorAll = () => [];
  formulario.insertAdjacentHTML = () => {};
  return formulario;
}

const siguienteTurno = () => new Promise((resolver) => setImmediate(resolver));

test("recuperación de edición reenvía bytes lógicos idénticos y acepta solo replay vinculado", async () => {
  const cuerpos = [];
  const resultado = { ...catalogo, revision: 3 };
  const cliente = crearClientePlantillasRRHH({ fetchImpl: async (ruta, opciones) => {
    if (ruta === RUTA_RRHH_PLANTILLAS) return respuesta({ borrador: catalogo, publicado: null,
      puede_editar: true, puede_publicar: false });
    cuerpos.push(opciones.body);
    if (cuerpos.length === 1) return respuesta({ codigo: "servicio_no_disponible" }, 503);
    const solicitud = JSON.parse(opciones.body);
    const recibo = reciboCambio(solicitud, resultado, "editar", reciboEdicion, "replay");
    if (cuerpos.length === 2) {
      const clave = solicitud.clave_idempotencia;
      recibo.clave_idempotencia = `${clave.slice(0, -1)}${clave.endsWith("0") ? "1" : "0"}`;
      assert.notEqual(recibo.clave_idempotencia, clave);
    }
    return respuesta({ catalogo: resultado, recibo }, 200);
  } });
  const { raiz, vista, pulsar, enviar, focosRecuperacion, focosRecibo } = vistaConEventos(cliente);
  await siguienteTurno();
  pulsar("editar", "modelo_nuevo");
  const formulario = formularioPrueba({ etiqueta: "Modelo nuevo", descripcion: "Modelo de prueba",
    titulo: "Título original", orden: "11", vigente_desde: "2026-09-28", fuente_ref: "fuente:rrhh:ejemplo",
    modalidades: "*", firmantes: "", requiere_accion: "", motivo: "Ajuste original" }, "form");
  formulario.append("parrafo", "Primer párrafo");
  enviar(formulario);
  await siguienteTurno();
  assert.match(raiz.innerHTML, /data-plantillas-accion="recuperar"/u);
  assert.match(raiz.innerHTML, /El resultado del cambio es indeterminado/u);
  assert.equal(focosRecuperacion(), 1);
  formulario.set("motivo", "Motivo distinto");
  pulsar("recuperar");
  await siguienteTurno();
  assert.equal(cuerpos.length, 2);
  assert.equal(cuerpos[1], cuerpos[0]);
  assert.match(raiz.innerHTML, /data-plantillas-accion="recuperar"/u);
  assert.doesNotMatch(raiz.innerHTML, new RegExp(reciboEdicion));
  assert.equal(focosRecuperacion(), 2);
  pulsar("recuperar");
  await siguienteTurno();
  assert.equal(cuerpos[2], cuerpos[0]);
  assert.match(raiz.innerHTML, new RegExp(reciboEdicion));
  assert.match(raiz.innerHTML, /class="rrhh-plantillas-recibo" role="status" tabindex="-1"/u);
  assert.equal(focosRecibo(), 1);
  assert.doesNotMatch(raiz.innerHTML, /data-plantillas-accion="recuperar"/u);
  vista.desmontar();
});

test("recuperación de publicación conserva solicitud y bloqueo tras 409 hasta replay válido", async () => {
  const cuerpos = [];
  const resultado = { ...catalogo, estado: "publicado" };
  const cliente = crearClientePlantillasRRHH({ fetchImpl: async (ruta, opciones) => {
    if (ruta === RUTA_RRHH_PLANTILLAS) return respuesta({ borrador: catalogo, publicado: null,
      puede_editar: false, puede_publicar: true });
    cuerpos.push(opciones.body);
    if (cuerpos.length === 1) return respuesta({ codigo: "servicio_no_disponible" }, 503);
    if (cuerpos.length === 2) return respuesta({ codigo: "servicio_no_disponible" }, 503);
    if (cuerpos.length === 3) return respuesta({ codigo: "acceso_denegado" }, 403);
    if (cuerpos.length === 4) return respuesta({ codigo: "version_en_conflicto" }, 409);
    const solicitud = JSON.parse(opciones.body);
    return respuesta({ catalogo: resultado,
      recibo: reciboCambio(solicitud, resultado, "publicar", reciboPublicacion, "replay") }, 200);
  } });
  const { raiz, vista, pulsar, enviar } = vistaConEventos(cliente);
  await siguienteTurno();
  const formulario = formularioPrueba({ aprobacion_ref: "aprobacion:rrhh:uno", motivo: "Publicación aprobada" }, "publicar");
  enviar(formulario);
  await siguienteTurno();
  assert.match(raiz.innerHTML, /data-plantillas-accion="recuperar"/u);
  formulario.set("aprobacion_ref", "aprobacion:distinta");
  pulsar("recuperar");
  await siguienteTurno();
  assert.match(raiz.innerHTML, /Sigue sin poder comprobarse el resultado/u);
  assert.match(raiz.innerHTML, /data-plantillas-accion="recuperar"/u);
  pulsar("recuperar");
  await siguienteTurno();
  assert.match(raiz.innerHTML, /No dispone de autorización para comprobar este resultado/u);
  assert.doesNotMatch(raiz.innerHTML, /<form data-plantillas-publicar/u);
  assert.match(raiz.innerHTML, /data-plantillas-accion="recuperar"/u);
  pulsar("recuperar");
  await siguienteTurno();
  assert.match(raiz.innerHTML, /La misma petición está en conflicto/u);
  assert.match(raiz.innerHTML, /data-plantillas-accion="recuperar"/u);
  pulsar("cancelar");
  enviar(formulario);
  assert.equal(cuerpos.length, 4);
  pulsar("recuperar");
  await siguienteTurno();
  assert.deepEqual(cuerpos, [cuerpos[0], cuerpos[0], cuerpos[0], cuerpos[0], cuerpos[0]]);
  assert.match(raiz.innerHTML, new RegExp(reciboPublicacion));
  assert.doesNotMatch(raiz.innerHTML, /data-plantillas-accion="recuperar"/u);
  vista.desmontar();
});
