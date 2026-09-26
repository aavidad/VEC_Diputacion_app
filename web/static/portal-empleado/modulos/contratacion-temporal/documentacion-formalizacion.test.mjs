import assert from "node:assert/strict";
import { createHash, webcrypto } from "node:crypto";
import { File } from "node:buffer";
import test from "node:test";
import { crearPanelDocumentacionFormalizacion } from "./documentacion-formalizacion.js";
import {
  crearFuenteDocumentacionFormalizacionHTTP, validarDocumentacionFormalizacion, RUTA_DOCUMENTACION_FORMALIZACION,
} from "./cliente-http-documentacion-formalizacion.js";
import { crearTraductorContratacionTemporal } from "./i18n.js";
import { MENSAJES_DOCUMENTACION_FORMALIZACION_ES } from "./i18n-documentacion-formalizacion.js";

const EXPEDIENTE = "expediente:ct:fe4934a1c7a9f9ad91aaccc6026ff7d39a494031d14d8a98dcd0d6a140619ba7";
const AGRUPACION = `ref:${"2".repeat(64)}`;
const ACEPTADA = "2026-09-25T09:05:00.12345Z";
const HUELLA_CATALOGO = "a".repeat(64);
const UUID = "0b0c6f8e-6a51-4a4b-9d51-3a1f2e5c7d90";
const t = crearTraductorContratacionTemporal();

function regla(clave, extra = {}) {
  return { clave, unidad: "dias_habiles", cantidad: 3, computo: "administrativo", origen: "ejemplo", ejemplo: true,
    norma: "Supuesto de trabajo.", duda: "Duda 13.", referencia: `vec.bolsa.reglas:1:${clave}`, huella_catalogo: HUELLA_CATALOGO, ...extra };
}

function documentacion({ estado = "en_curso", documentos } = {}) {
  return {
    esquema: "vec.ct.formalizacion.documentacion.v1", expediente_documental_ref: AGRUPACION, aceptada_en: ACEPTADA, por_modalidad: false,
    documentos: documentos ?? [
      { clave: "documento_identidad", tipo_documental: "contratacion_temporal.formalizacion.documento_identidad.v1", registrable: true },
      { clave: "titulacion", tipo_documental: "contratacion_temporal.formalizacion.titulacion.v1", registrable: true },
      { clave: "otro_documento", tipo_documental: "contratacion_temporal.formalizacion.otro_documento.v1", registrable: false },
    ],
    regla_documentos: { ...regla("b22.documentos_incorporacion"), unidad: "lista", cantidad: undefined, computo: undefined },
    plazo_documentacion: { ultimo_dia: "2026-09-30", vence_antes_de: "2026-09-30T22:00:00Z", prorrogado: false, estado, regla: regla("b21.plazo_documentacion") },
    plazo_incorporacion: { ultimo_dia: "2026-09-26", vence_antes_de: "2026-09-26T22:00:00Z", prorrogado: false, estado: "en_curso",
      regla: regla("b23.plazo_incorporacion", { cantidad: 1, origen: "reglamento", articulo: "art. 11.1", parte_ejemplo: "El mínimo es de ejemplo." }) },
    consultada_en: "2026-09-25T10:00:00Z",
  };
}

function contenedorPrueba() {
  const oyentes = {};
  const enfocados = [];
  return {
    innerHTML: "", enfocados,
    addEventListener(tipo, fn) { (oyentes[tipo] ??= new Set()).add(fn); },
    removeEventListener(tipo, fn) { oyentes[tipo]?.delete(fn); },
    contains: () => true,
    querySelector(selector) { return { focus: () => enfocados.push(selector) }; },
    async emitir(tipo, target) {
      let prevenido = false;
      const evento = { type: tipo, target, preventDefault: () => { prevenido = true; } };
      for (const fn of oyentes[tipo] ?? []) await fn(evento);
      return prevenido;
    },
  };
}

function boton(atributos) {
  const dataset = { ...atributos };
  return { dataset, closest: () => ({ dataset }) };
}

function fuentePrueba({ datos = documentacion(), anotados = new Map(), registrar } = {}) {
  const llamadas = { consultar: [], anotados: 0, registrar: [] };
  return {
    llamadas,
    async consultar(parametros) { llamadas.consultar.push(parametros.aceptadaEn); llamadas.expediente = parametros.expedienteRef; return validarDocumentacionFormalizacion(datos); },
    async anotados(parametros) { llamadas.anotados += 1; llamadas.agrupacion = parametros.expedienteRef; return new Map(anotados); },
    async registrar(parametros) {
      llamadas.registrar.push(parametros);
      if (registrar) return registrar(parametros);
      return { numero_vec: "VEC-2026-41", huella: parametros.huella };
    },
  };
}

async function esperar() {
  for (let i = 0; i < 10; i += 1) await new Promise((resolver) => setImmediate(resolver));
}

async function panelMontado(opciones = {}) {
  const fuente = fuentePrueba(opciones);
  const contenedor = contenedorPrueba();
  const panel = crearPanelDocumentacionFormalizacion({ fuente, t, criptografia: webcrypto, generarClaveIdempotencia: () => UUID });
  panel.pintar(contenedor, { aceptadaEn: ACEPTADA, expedienteRef: EXPEDIENTE });
  await esperar();
  return { fuente, contenedor, panel };
}

test("sin aceptación confirmada el panel no consulta ni pinta nada", async () => {
  const fuente = fuentePrueba();
  const contenedor = contenedorPrueba();
  const panel = crearPanelDocumentacionFormalizacion({ fuente, t });
  panel.pintar(contenedor, { aceptadaEn: "", expedienteRef: EXPEDIENTE });
  panel.pintar(contenedor, { aceptadaEn: ACEPTADA, expedienteRef: "expediente ct con espacios" });
  await esperar();
  assert.equal(contenedor.innerHTML, "");
  assert.equal(fuente.llamadas.consultar.length, 0);
});

test("muestra documentos, plazo desde la aceptación, procedencia y ayuda solo tras «?»", async () => {
  const { fuente, contenedor } = await panelMontado({
    anotados: new Map([["contratacion_temporal.formalizacion.titulacion.v1", { numero_vec: "VEC-2026-7", huella: "b".repeat(64) }]]),
  });
  const html = contenedor.innerHTML;
  assert.deepEqual(fuente.llamadas.consultar, [ACEPTADA]);
  assert.equal(fuente.llamadas.expediente, EXPEDIENTE);
  assert.equal(fuente.llamadas.agrupacion, AGRUPACION);
  assert.match(html, /Documentación para la formalización/u);
  assert.match(html, /<details data-ct-formalizacion-ayuda><summary aria-label="Ayuda sobre la documentación para la formalización"><span aria-hidden="true">\?<\/span><\/summary>/u);
  const fueraDeAyuda = html.replace(/<details data-ct-formalizacion-ayuda>[\s\S]*?<\/details>/u, "");
  assert.doesNotMatch(fueraDeAyuda, /catálogo de reglas vigente/u);
  assert.match(html, /3 días hábiles · hasta el 30 de septiembre de 2026 \(incluido\)/u);
  assert.match(html, /Margen mínimo hasta la incorporación/u);
  // El origen de la regla y su referencia no se rotulan en la pantalla de trabajo.
  assert.doesNotMatch(fueraDeAyuda, /Regla de ejemplo|vec\.bolsa\.reglas:1|SHA-256/u);
  assert.match(html, /<tr data-ct-formalizacion-documento="titulacion">[\s\S]*?Aportado[\s\S]*?Anotado con el n\.º VEC-2026-7/u);
  assert.match(html, /<tr data-ct-formalizacion-documento="documento_identidad">[\s\S]*?Pendiente[\s\S]*?data-ct-formalizacion-anotar="documento_identidad"/u);
  assert.match(html, /<tr data-ct-formalizacion-documento="otro_documento"><th scope="row">Documento «otro_documento»<\/th>[\s\S]*?Sin política de conservación/u);
  assert.doesNotMatch(html, /data-ct-formalizacion-aviso/u);
});

test("avisa del último día y del vencimiento solo con documentos pendientes", async () => {
  let montaje = await panelMontado({ datos: documentacion({ estado: "vencido" }) });
  assert.match(montaje.contenedor.innerHTML, /role="alert"\s+data-ct-formalizacion-aviso="vencido">El plazo venció el 30 de septiembre de 2026 y quedan 3 documento\(s\) pendiente\(s\)/u);
  montaje = await panelMontado({ datos: documentacion({ estado: "ultimo_dia" }) });
  assert.match(montaje.contenedor.innerHTML, /data-ct-formalizacion-aviso="ultimo_dia">Hoy termina el plazo y quedan 3/u);
  const todos = documentacion({ estado: "vencido" }).documentos.map((d) => [d.tipo_documental, { numero_vec: "VEC-2026-1", huella: "c".repeat(64) }]);
  montaje = await panelMontado({ datos: documentacion({ estado: "vencido" }), anotados: new Map(todos) });
  assert.match(montaje.contenedor.innerHTML, /data-ct-formalizacion-aviso="completa"/u);
  assert.doesNotMatch(montaje.contenedor.innerHTML, /data-ct-formalizacion-aviso="vencido"/u);
});

test("anota un documento con referencia y huella calculada localmente, conservando la clave al reintentar", async () => {
  let fallar = true;
  const { fuente, contenedor } = await panelMontado({
    registrar: async (p) => {
      if (fallar) { fallar = false; throw Object.assign(new Error("x"), { codigo: "consulta_fallida" }); }
      return { numero_vec: "VEC-2026-42", huella: p.huella };
    },
  });
  await contenedor.emitir("click", boton({ ctFormalizacionAnotar: "documento_identidad" }));
  assert.match(contenedor.innerHTML, /data-ct-formalizacion-form/u);
  assert.match(contenedor.innerHTML, /<label for="ct-formalizacion-referencia">Referencia del original \*<\/label>/u);
  const formulario = { dataset: { ctFormalizacionForm: "" } };
  assert.equal(await contenedor.emitir("submit", formulario), true);
  assert.match(contenedor.innerHTML, /Indique una referencia/u);
  assert.equal(fuente.llamadas.registrar.length, 0);

  await contenedor.emitir("input", { dataset: { ctFormalizacionReferencia: "" }, value: "REGE-2026-000123" });
  const contenido = Buffer.from("%PDF-1.7 documento sintético");
  await contenedor.emitir("change", { dataset: { ctFormalizacionArchivo: "" }, files: [new File([contenido], "dni.pdf")] });
  await esperar();
  const huella = createHash("sha256").update(contenido).digest("hex");
  assert.match(contenedor.innerHTML, /Documento leído y comprobado en este equipo/u); assert.ok(!contenedor.innerHTML.includes(huella));
  assert.doesNotMatch(contenedor.innerHTML, /dni\.pdf/u);

  await contenedor.emitir("submit", formulario);
  assert.match(contenedor.innerHTML, /No se pudo anotar el documento/u);
  await contenedor.emitir("submit", formulario);
  assert.equal(fuente.llamadas.registrar.length, 2);
  const [primero, segundo] = fuente.llamadas.registrar;
  assert.deepEqual({ ...primero, signal: undefined }, { expedienteRef: AGRUPACION,
    tipo: "contratacion_temporal.formalizacion.documento_identidad.v1", referencia: "REGE-2026-000123",
    huella, claveIdempotencia: `clave:${UUID}`, signal: undefined });
  assert.equal(segundo.claveIdempotencia, primero.claveIdempotencia);
  assert.match(contenedor.innerHTML, /Documento anotado con el número VEC-2026-42/u);
  assert.match(contenedor.innerHTML, /<tr data-ct-formalizacion-documento="documento_identidad">[\s\S]*?Aportado/u);
  assert.doesNotMatch(contenedor.innerHTML, /data-ct-formalizacion-form/u);
});

test("rechaza referencias con rutas o comodines sin llamar al servidor", async () => {
  const { fuente, contenedor } = await panelMontado();
  await contenedor.emitir("click", boton({ ctFormalizacionAnotar: "titulacion" }));
  for (const referencia of ["../x", "RE*G", "ab", "con espacio"]) {
    await contenedor.emitir("input", { dataset: { ctFormalizacionReferencia: "" }, value: referencia });
    await contenedor.emitir("change", { dataset: { ctFormalizacionArchivo: "" }, files: [new File([Buffer.from("x")], "a.pdf")] });
    await esperar();
    await contenedor.emitir("submit", { dataset: { ctFormalizacionForm: "" } });
  }
  assert.equal(fuente.llamadas.registrar.length, 0);
  await contenedor.emitir("click", boton({ ctFormalizacionAnotar: "otro_documento" }));
  assert.match(contenedor.innerHTML, /Anotar documento aportado: Titulación exigida/u);
});

test("distingue catálogo no configurado y permite reintentar", async () => {
  const contenedor = contenedorPrueba();
  let intento = 0;
  const fuente = { ...fuentePrueba(), async consultar() {
    intento += 1;
    if (intento === 1) throw Object.assign(new Error("x"), { codigo: "sin_reglas" });
    return validarDocumentacionFormalizacion(documentacion());
  } };
  const panel = crearPanelDocumentacionFormalizacion({ fuente, t });
  panel.pintar(contenedor, { aceptadaEn: ACEPTADA, expedienteRef: EXPEDIENTE });
  await esperar();
  assert.match(contenedor.innerHTML, /No hay catálogo de reglas configurado/u);
  await contenedor.emitir("click", boton({ ctFormalizacionReintentar: "" }));
  await esperar();
  assert.match(contenedor.innerHTML, /data-ct-formalizacion-documento="titulacion"/u);
  panel.desmontar();
});

test("todas las claves visibles existen en el catálogo i18n común", () => {
  for (const clave of Object.keys(MENSAJES_DOCUMENTACION_FORMALIZACION_ES)) assert.equal(typeof t(clave), "string");
});

function respuestaJSON(estado, cuerpo) {
  return new Response(JSON.stringify(cuerpo), { status: estado, headers: { "Content-Type": "application/json" } });
}

test("el cliente consulta por GET sin cuerpo y traduce los errores cerrados", async () => {
  const pedidas = [];
  let respuesta = respuestaJSON(200, { data: documentacion() });
  const fuente = crearFuenteDocumentacionFormalizacionHTTP({ fetchImpl: async (url, opciones) => {
    pedidas.push({ url, opciones }); return respuesta;
  } });
  const datos = await fuente.consultar({ expedienteRef: EXPEDIENTE, aceptadaEn: ACEPTADA });
  assert.equal(datos.documentos.length, 3);
  assert.equal(datos.expediente_documental_ref, AGRUPACION);
  assert.equal(pedidas[0].url, `${RUTA_DOCUMENTACION_FORMALIZACION}?expediente_ref=${encodeURIComponent(EXPEDIENTE)}&aceptada_en=${encodeURIComponent(ACEPTADA)}`);
  assert.equal(pedidas[0].opciones.method, "GET");
  assert.equal(pedidas[0].opciones.body, undefined);
  assert.equal(pedidas[0].opciones.credentials, "same-origin");
  respuesta = respuestaJSON(503, { error: { codigo: "reglas_no_configuradas" } });
  await assert.rejects(fuente.consultar({ expedienteRef: EXPEDIENTE, aceptadaEn: ACEPTADA }), { codigo: "sin_reglas" });
  respuesta = respuestaJSON(403, { error: { codigo: "acceso_denegado" } });
  await assert.rejects(fuente.consultar({ expedienteRef: EXPEDIENTE, aceptadaEn: ACEPTADA }), { codigo: "denegado" });
  respuesta = respuestaJSON(200, { data: { ...documentacion(), esquema: "otro" } });
  await assert.rejects(fuente.consultar({ expedienteRef: EXPEDIENTE, aceptadaEn: ACEPTADA }), { codigo: "respuesta_invalida" });
  await assert.rejects(fuente.consultar({ expedienteRef: EXPEDIENTE, aceptadaEn: "25/09/2026" }), { codigo: "referencia_invalida" });
});

test("la respuesta rechaza claves repetidas y estados desconocidos", () => {
  const repetida = documentacion();
  repetida.documentos = [repetida.documentos[0], repetida.documentos[0]];
  assert.throws(() => validarDocumentacionFormalizacion(repetida), { codigo: "respuesta_invalida" });
  assert.throws(() => validarDocumentacionFormalizacion(documentacion({ estado: "proximo" })), { codigo: "respuesta_invalida" });
});

test("el cliente lista anotaciones externas por tipo y registra sin módulo ni custodio", async () => {
  const pedidas = [];
  const paginas = [
    { documentos: [{ tipo: "contratacion_temporal.formalizacion.titulacion.v1", custodia: "externa", numero_vec: "VEC-2026-3", huella: "d".repeat(64) },
      { tipo: "contratacion_temporal.borrador.v1", custodia: "vec", numero_vec: "VEC-2026-4", huella: "e".repeat(64) }], siguiente_cursor: "cursor:2" },
    { documentos: [], siguiente_cursor: "" },
  ];
  const fetchImpl = async (url, opciones) => {
    const cuerpo = JSON.parse(opciones.body);
    pedidas.push({ url, cuerpo });
    if (url.endsWith("/externos/registros")) {
      return respuestaJSON(201, { data: { estado: "registrado", documento: { ref: `ref:${"1".repeat(64)}`, numero_vec: "VEC-2026-9",
        tipo: cuerpo.tipo, version: 1, huella: cuerpo.huella_sha256, custodia: "externa", registrado_en: "2026-09-25T10:00:00Z" } } });
    }
    return respuestaJSON(200, { data: { estado: "disponible", ...paginas.shift() } });
  };
  const fuente = crearFuenteDocumentacionFormalizacionHTTP({ fetchImpl });
  const anotados = await fuente.anotados({ expedienteRef: AGRUPACION });
  assert.deepEqual([...anotados.keys()], ["contratacion_temporal.formalizacion.titulacion.v1"]);
  assert.equal(pedidas[1].cuerpo.cursor, "cursor:2");
  const registro = await fuente.registrar({ expedienteRef: AGRUPACION, tipo: "contratacion_temporal.formalizacion.titulacion.v1",
    referencia: "REGE-2026-1", huella: "f".repeat(64), claveIdempotencia: `clave:${UUID}` });
  assert.equal(registro.numero_vec, "VEC-2026-9");
  assert.deepEqual(Object.keys(pedidas[2].cuerpo).sort(), ["clave_idempotencia", "expediente_ref", "huella_sha256", "referencia", "tipo"]);
  await assert.rejects(fuente.registrar({ expedienteRef: AGRUPACION, tipo: "contratacion_temporal.formalizacion.titulacion.v1",
    referencia: "../etc", huella: "f".repeat(64), claveIdempotencia: `clave:${UUID}` }), { codigo: "referencia_invalida" });
});

test("el formulario de llamamiento pinta el panel tras la aceptación con su instante y expediente", async () => {
  const { raizPrueba, abrirResolucion, resolucionConfirmada, revisionManual, CLAVE_RESOLUCION, PUBLICACIONES_PROPUESTA, EXPEDIENTE: EXP } =
    await import("./formulario-llamamiento-pruebas.js");
  const raiz = raizPrueba();
  const fuente = fuentePrueba();
  const cerrar = await abrirResolucion(raiz, { resolverLlamamiento: async () => resolucionConfirmada }, {
    contexto: { expediente_ref: EXP, version_esperada: 6 }, fuenteDocumentacionFormalizacion: fuente,
    fetchPublicaciones: async () => new Response(PUBLICACIONES_PROPUESTA, { status: 200, headers: { "Content-Type": "application/json" } }),
  });
  assert.equal(fuente.llamadas.consultar.length, 0);
  await raiz.enviar("resolucion", { clave_idempotencia: CLAVE_RESOLUCION, ...revisionManual });
  await esperar();
  assert.match(raiz.innerHTML, /<div data-ct-documentacion-formalizacion><\/div>/u);
  assert.deepEqual([...new Set(fuente.llamadas.consultar)], [resolucionConfirmada.resuelta_en]);
  cerrar();
});
