import assert from "node:assert/strict";
import test from "node:test";
import { crearClientePublicacionBases, erroresCuerpoPublicacion, RUTA_PUBLICACIONES_OFICIALES,
  sondearPublicacionBases, validarListaPublicaciones } from "./rrhh-publicacion-bases-api.js";
import { cuerpoDeFormulario, hoyMadrid, montarPublicacionBasesRRHH } from "./rrhh-publicacion-bases-vista.js";
import { cargarTextos } from "../../../comun/textos.js";
import { aplicarDisponibilidadMenuBolsa, categoriaDeVistaBolsa, vistaBolsaOfrecida } from "../../portal-menu-bolsa.js";

const H = "a".repeat(64);
const REF = `cv1_${"b".repeat(64)}_v2`;
const PUB = `publicacion_bases:${"c".repeat(64)}`;
const pendiente = Object.freeze({ convocatoria_ref: REF, secuencia: 2, identificador_publico: "BOLSA-2026-07",
  titulo: "Bolsa de auxiliares administrativos", huella_version_sha256: H,
  plazo_inscripcion_abre_en: "2026-11-02T08:00:00Z", publicacion: null });
const publicacion = Object.freeze({ publicacion_ref: PUB, medio: "bop_granada", boletin_numero: 187,
  anuncio_ref: "2026/4521", fecha_publicacion: "2026-10-15", direccion: "https://bop.example.invalid/4521",
  registrada_en: "2026-10-16T09:30:00Z" });
const sobreLista = (items = [pendiente], completa = true) => ({ data: { items, completa } });
const json = (status, cuerpo) => new Response(JSON.stringify(cuerpo), { status, headers: { "content-type": "application/json" } });
const camposValidos = { medio: "bop_granada", boletin_numero: "187", anuncio_ref: "2026/4521",
  fecha_publicacion: "2026-10-15", direccion: "" };
const resguardo = (cuerpo, extra = {}) => ({ data: { publicacion_ref: PUB, recibo_ref: "recibo:publicacion:1",
  evento_ref: "evento:1", auditoria_ref: "auditoria:1", convocatoria_ref: cuerpo.convocatoria_ref, secuencia: 2,
  medio: cuerpo.medio, boletin_numero: cuerpo.boletin_numero, anuncio_ref: cuerpo.anuncio_ref,
  fecha_publicacion: cuerpo.fecha_publicacion, direccion: cuerpo.direccion, sustituye_ref: cuerpo.sustituye_ref,
  registrada_en: "2026-10-20T10:00:00Z", reutilizada: false, ...extra } });

test("GET sin parámetros, mismo origen y sin credenciales añadidas por JS", async () => {
  const llamadas = [];
  const cliente = crearClientePublicacionBases({ fetchImpl: async (ruta, opciones) => {
    llamadas.push([ruta, opciones]); return json(200, sobreLista());
  } });
  const lista = await cliente.listar();
  assert.equal(lista.items[0].convocatoria_ref, REF);
  assert.equal(llamadas[0][0], RUTA_PUBLICACIONES_OFICIALES);
  assert.doesNotMatch(llamadas[0][0], /\?/u);
  assert.equal(llamadas[0][1].method, "GET");
  assert.equal(llamadas[0][1].credentials, "same-origin");
  assert.equal(llamadas[0][1].redirect, "error");
  assert.deepEqual(Object.keys(llamadas[0][1].headers), ["Accept"]);
});

test("falla cerrado ante una lista incompatible", () => {
  assert.throws(() => validarListaPublicaciones(sobreLista([{ ...pendiente, convocatoria_ref: "cv1_x_v1" }])));
  assert.throws(() => validarListaPublicaciones(sobreLista([{ ...pendiente, huella_version_sha256: "A".repeat(64) }])));
  assert.throws(() => validarListaPublicaciones(sobreLista([{ ...pendiente, publicacion: { ...publicacion, medio: "diario" } }])));
  assert.throws(() => validarListaPublicaciones({ data: { items: [] } }));
  assert.equal(validarListaPublicaciones(sobreLista([{ ...pendiente, publicacion }])).items[0].publicacion.boletin_numero, 187);
});

test("POST con cuerpo exacto, Idempotency-Key y sin quién ni motivo", async () => {
  const llamadas = [];
  const cliente = crearClientePublicacionBases({ fetchImpl: async (ruta, opciones) => {
    llamadas.push([ruta, opciones]); return json(201, resguardo(JSON.parse(opciones.body)));
  } });
  const cuerpo = cuerpoDeFormulario(pendiente, camposValidos);
  const recibido = await cliente.declarar(cuerpo, { claveIdempotencia: "publicacion-bases:1234" });
  assert.equal(recibido.recibo_ref, "recibo:publicacion:1");
  const [ruta, opciones] = llamadas[0];
  assert.equal(ruta, RUTA_PUBLICACIONES_OFICIALES);
  assert.equal(opciones.method, "POST");
  assert.equal(opciones.headers["Idempotency-Key"], "publicacion-bases:1234");
  assert.equal(opciones.headers["Content-Type"], "application/json");
  assert.equal(opciones.headers.Accept, "application/json");
  assert.deepEqual(Object.keys(JSON.parse(opciones.body)).sort(), ["anuncio_ref", "boletin_numero", "convocatoria_ref",
    "direccion", "fecha_publicacion", "huella_version_sha256", "medio", "sustituye_ref"]);
  assert.deepEqual(JSON.parse(opciones.body), { convocatoria_ref: REF, huella_version_sha256: H, medio: "bop_granada",
    boletin_numero: 187, anuncio_ref: "2026/4521", fecha_publicacion: "2026-10-15", direccion: null, sustituye_ref: null });
  await assert.rejects(cliente.declarar(cuerpo, { claveIdempotencia: "corta" }));
});

test("el tablón no lleva número de boletín y las reglas de campo se comprueban antes de enviar", () => {
  const tablon = cuerpoDeFormulario(pendiente, { ...camposValidos, medio: "tablon_edictos_electronico" });
  assert.equal(tablon.boletin_numero, null);
  assert.deepEqual(erroresCuerpoPublicacion(tablon, "2026-10-20"), []);
  const mal = cuerpoDeFormulario(pendiente, { medio: "", boletin_numero: "100000", anuncio_ref: "con espacio",
    fecha_publicacion: "2026-10-21", direccion: "http://inseguro.example" });
  assert.deepEqual(erroresCuerpoPublicacion(mal, "2026-10-20"),
    ["medio", "boletin_numero", "anuncio_ref", "fecha_publicacion", "direccion"]);
  assert.deepEqual(erroresCuerpoPublicacion(cuerpoDeFormulario(pendiente, { ...camposValidos, boletin_numero: "0" }), "2026-10-20"), ["boletin_numero"]);
  assert.deepEqual(erroresCuerpoPublicacion(cuerpoDeFormulario(pendiente, { ...camposValidos, fecha_publicacion: "2026-02-30" }), "2026-10-20"), ["fecha_publicacion"]);
  // Medianoche en Madrid: el 20 a las 23:30 UTC ya es día 21 en Madrid.
  assert.equal(hoyMadrid(Date.parse("2026-10-20T23:30:00Z")), "2026-10-21");
});

test("sondeo del menú: 200 y 503 muestran la entrada; 403 y 404 la ocultan", async () => {
  const con = (status, codigo) => crearClientePublicacionBases({ fetchImpl: async () => status === 200
    ? json(200, sobreLista()) : json(status, { error: { codigo } }) });
  assert.equal(await sondearPublicacionBases(con(200)), "disponible");
  assert.equal(await sondearPublicacionBases(con(503, "publicacion_oficial_no_habilitada")), "no_habilitada");
  assert.equal(await sondearPublicacionBases(con(403, "acceso_denegado")), "oculta");
  assert.equal(await sondearPublicacionBases(con(404, "")), "oculta");
  assert.equal(categoriaDeVistaBolsa("publicacion-bases"), "bolsas-candidatos");
  assert.equal(vistaBolsaOfrecida("publicacion-bases", { publicacionBases: true }), true);
  assert.equal(vistaBolsaOfrecida("publicacion-bases", { publicacionBases: false }), false);
  const control = { hidden: false, getAttribute: () => "publicacion-bases" };
  aplicarDisponibilidadMenuBolsa({ querySelectorAll: (selector) => selector === ".submenu-bolsa [data-vista]" ? [control] : [] }, {});
  assert.equal(control.hidden, true);
});

const es = await cargarTextos("bolsa-publicacion-bases", { idioma: "es", porDefecto: "es" });
const en = await cargarTextos("bolsa-publicacion-bases", { idioma: "en", porDefecto: "es" });

test("catálogos es/en completos y con las mismas claves", () => {
  const claves = (c) => Object.keys(c.seccion("publicacion")).sort();
  assert.deepEqual(claves(en), claves(es));
  assert.equal(en.faltantes.length, 0);
});

function crearRaiz() {
  const eventos = new Map();
  const enfocados = [];
  const raiz = { innerHTML: "", addEventListener: (tipo, f) => eventos.set(tipo, f),
    removeEventListener: (tipo) => eventos.delete(tipo), replaceChildren() { this.innerHTML = ""; },
    querySelector: (selector) => ({ focus: () => enfocados.push(selector) }) };
  const pulsar = (atributo, valor = "") => eventos.get("click")({ target: { closest: (s) => s === `[${atributo}]`
    ? { getAttribute: () => valor } : null } });
  const escribir = (name, value, type = "input") => eventos.get(type)({ type, target: { name, value } });
  const enviar = () => eventos.get("submit")({ preventDefault() {}, target: { matches: (s) => s === "[data-pb-formulario]" } });
  return { raiz, eventos, enfocados, pulsar, escribir, enviar };
}
const esperar = () => new Promise((resolver) => setTimeout(resolver, 0));

test("vista: lista, formulario con etiquetas, errores por campo y resguardo tras éxito", async () => {
  const { raiz, eventos, enfocados, pulsar, escribir, enviar } = crearRaiz();
  let listado = sobreLista();
  const envios = [];
  const anuncios = [];
  const cliente = { listar: async () => validarListaPublicaciones(listado),
    declarar: async (cuerpo, { claveIdempotencia }) => {
      envios.push({ cuerpo, claveIdempotencia });
      listado = sobreLista([{ ...pendiente, publicacion: { ...publicacion, direccion: null } }]);
      return resguardo(cuerpo).data;
    } };
  const vista = await montarPublicacionBasesRRHH({ raiz, cliente, cargarCatalogo: async () => es,
    anunciar: (m) => anuncios.push(m), ahora: () => Date.parse("2026-10-20T10:00:00Z"), nuevaClave: () => "publicacion-bases:clave-1" });
  await esperar();
  assert.match(raiz.innerHTML, /Bolsa de auxiliares administrativos/u);
  assert.match(raiz.innerHTML, /Sin registrar/u);
  // WCAG 2.5.3: el nombre accesible empieza por el texto visible del botón.
  assert.match(raiz.innerHTML, /aria-label="Registrar publicación: Bolsa de auxiliares administrativos">Registrar publicación<\/button>/u);
  assert.match(raiz.innerHTML, /class="tabla-datos tabla-apilable"/u);
  assert.match(raiz.innerHTML, /<td class="envuelve" data-etiqueta="Publicación">/u);
  assert.match(raiz.innerHTML, /id="pb-ayuda"[^>]*hidden/u, "ayuda solo tras «?»");
  pulsar("data-pb-abrir", REF);
  assert.match(raiz.innerHTML, /<label for="pb-medio">Dónde se publicó/u);
  assert.match(raiz.innerHTML, /<label for="pb-anuncio_ref">Referencia del anuncio/u);
  assert.match(raiz.innerHTML, /<label for="pb-direccion">Enlace al anuncio <span[^>]*>\(opcional\)/u);
  assert.match(raiz.innerHTML, /type="date"[^>]*max="2026-10-20"/u);
  assert.ok(enfocados.includes('[id="pb-form-titulo"]'));
  enviar();
  assert.match(raiz.innerHTML, /id="pb-resumen"/u);
  assert.match(raiz.innerHTML, /id="pb-medio"[^>]*aria-invalid="true" aria-describedby="pb-error-medio"/u);
  assert.match(raiz.innerHTML, /id="pb-error-medio">Elija dónde se publicaron las bases/u);
  assert.equal(envios.length, 0);
  escribir("medio", "tablon_edictos_electronico", "change");
  assert.match(raiz.innerHTML, /id="pb-boletin_numero"[^>]*disabled/u, "el tablón no pide boletín");
  escribir("medio", "bop_granada", "change");
  for (const [nombre, valor] of Object.entries(camposValidos)) if (nombre !== "medio") escribir(nombre, valor);
  enviar();
  await esperar(); await esperar();
  assert.equal(envios.length, 1);
  assert.equal(envios[0].claveIdempotencia, "publicacion-bases:clave-1");
  assert.equal(envios[0].cuerpo.boletin_numero, 187);
  assert.match(raiz.innerHTML, /Publicación registrada/u);
  assert.match(raiz.innerHTML, /Se ha registrado la publicación de las bases de Bolsa de auxiliares administrativos\./u);
  assert.match(raiz.innerHTML, /figura ahora como Registrada/u, "el resguardo dice qué pasa ahora");
  // La referencia técnica solo aparece dentro del detalle plegado.
  assert.match(raiz.innerHTML, /<details class="detalle-tecnico-plegado"><summary>Ver detalle técnico<\/summary>[\s\S]*recibo:publicacion:1[\s\S]*<\/details>/u);
  assert.equal(raiz.innerHTML.replace(/<details[\s\S]*<\/details>/u, "").includes("recibo:publicacion:1"), false);
  assert.match(raiz.innerHTML, /estado-chip exito">Registrada/u, "la fila se refresca");
  assert.doesNotMatch(raiz.innerHTML, /data-pb-formulario/u);
  assert.match(anuncios.at(-1), /Publicación registrada/u);
  vista.desmontar();
  assert.equal(eventos.size, 0);
});

test("corregir envía sustituye_ref; resultado no confirmado reutiliza la misma clave", async () => {
  const { raiz, pulsar, enviar, escribir } = crearRaiz();
  const envios = [];
  let fallar = true;
  let claves = 0;
  const cliente = { listar: async () => validarListaPublicaciones(sobreLista([{ ...pendiente, publicacion }])),
    declarar: async (cuerpo, { claveIdempotencia }) => {
      envios.push({ cuerpo, claveIdempotencia });
      if (fallar) { fallar = false; throw Object.assign(new Error("x"), { estado: 503, codigo: "resultado_no_confirmado" }); }
      return resguardo(cuerpo, { reutilizada: true }).data;
    } };
  await montarPublicacionBasesRRHH({ raiz, cliente, cargarCatalogo: async () => es,
    ahora: () => Date.parse("2026-10-20T10:00:00Z"), nuevaClave: () => `publicacion-bases:clave-${++claves}` });
  await esperar();
  assert.match(raiz.innerHTML, /Corregir/u);
  assert.match(raiz.innerHTML, /Boletín Oficial de la Provincia de Granada, número 187/u);
  pulsar("data-pb-abrir", REF);
  assert.match(raiz.innerHTML, /Corregir la publicación de las bases/u);
  assert.match(raiz.innerHTML, /value="2026\/4521"/u, "los datos actuales se precargan");
  escribir("anuncio_ref", "2026/4522");
  enviar(); await esperar();
  assert.match(raiz.innerHTML, /No se ha podido confirmar si la publicación quedó registrada/u);
  enviar(); await esperar(); await esperar();
  assert.equal(envios.length, 2);
  assert.equal(envios[0].cuerpo.sustituye_ref, PUB);
  assert.equal(envios[1].claveIdempotencia, envios[0].claveIdempotencia, "mismo envío, misma clave");
  assert.match(raiz.innerHTML, /ya constaba con los mismos datos/u);
});

test("409 de versión recarga la lista; 503 sin habilitar no ofrece formulario; 403 avisa y oculta", async () => {
  const { raiz, pulsar, enviar, escribir } = crearRaiz();
  let lecturas = 0;
  const cliente = { listar: async () => { lecturas++; return validarListaPublicaciones(sobreLista()); },
    declarar: async () => { throw Object.assign(new Error("x"), { estado: 409, codigo: "version_no_vigente" }); } };
  await montarPublicacionBasesRRHH({ raiz, cliente, cargarCatalogo: async () => es,
    ahora: () => Date.parse("2026-10-20T10:00:00Z"), nuevaClave: () => "publicacion-bases:clave-9" });
  await esperar();
  pulsar("data-pb-abrir", REF);
  for (const [nombre, valor] of Object.entries(camposValidos)) escribir(nombre, valor);
  enviar(); await esperar(); await esperar();
  assert.equal(lecturas, 2);
  assert.match(raiz.innerHTML, /han cambiado mientras rellenaba/u);
  assert.doesNotMatch(raiz.innerHTML, /data-pb-formulario/u);

  const otra = crearRaiz();
  const disponibilidades = [];
  await montarPublicacionBasesRRHH({ raiz: otra.raiz, cargarCatalogo: async () => es, alDisponibilidad: (d) => disponibilidades.push(d),
    cliente: { listar: async () => { throw Object.assign(new Error("x"), { estado: 503, codigo: "publicacion_oficial_no_habilitada" }); }, declarar: async () => {} } });
  await esperar();
  assert.match(otra.raiz.innerHTML, /Esta pantalla no está disponible/u);
  assert.doesNotMatch(otra.raiz.innerHTML, /data-pb-formulario|data-pb-abrir/u);
  assert.deepEqual(disponibilidades, ["no_habilitada"]);

  const tercera = crearRaiz();
  const avisos = [];
  await montarPublicacionBasesRRHH({ raiz: tercera.raiz, cargarCatalogo: async () => en, alDisponibilidad: (d) => avisos.push(d),
    cliente: { listar: async () => { throw Object.assign(new Error("x"), { estado: 403, codigo: "acceso_denegado" }); }, declarar: async () => {} } });
  await esperar();
  assert.match(tercera.raiz.innerHTML, /role="alert">Your current access/u);
  assert.match(tercera.raiz.innerHTML, /data-vista="portal">Back to Home/u, "sin permiso hay salida");
  assert.doesNotMatch(tercera.raiz.innerHTML, /data-pb-ayuda/u, "sin permiso no se ofrece la ayuda de la tarea");
  assert.deepEqual(avisos, ["oculta"]);
});

test("cancelar con datos escritos pide confirmación; sin cambios cierra directamente", async () => {
  const { raiz, pulsar, escribir } = crearRaiz();
  const preguntas = [];
  let respuesta = false;
  await montarPublicacionBasesRRHH({ raiz, cargarCatalogo: async () => es, confirmar: (m) => { preguntas.push(m); return respuesta; },
    cliente: { listar: async () => validarListaPublicaciones(sobreLista()), declarar: async () => {} } });
  await esperar();
  pulsar("data-pb-abrir", REF);
  pulsar("data-pb-cancelar");
  assert.deepEqual(preguntas, []);
  assert.doesNotMatch(raiz.innerHTML, /data-pb-formulario/u);
  pulsar("data-pb-abrir", REF);
  escribir("anuncio_ref", "2026/1");
  pulsar("data-pb-cancelar");
  assert.match(preguntas[0], /Hay datos sin guardar/u);
  assert.match(raiz.innerHTML, /data-pb-formulario/u, "si no confirma, el formulario sigue abierto");
  respuesta = true;
  pulsar("data-pb-cancelar");
  assert.doesNotMatch(raiz.innerHTML, /data-pb-formulario/u);
});
