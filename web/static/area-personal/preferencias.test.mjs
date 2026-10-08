import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { test } from "node:test";
import { cargarPreferenciasIniciales, crearClientePreferencias, ErrorPreferencias, RUTA_MIS_PREFERENCIAS } from "./cliente-http.js";
import { alternarVisualSesion, crearOperacionPreferencias, renderizarPreferencias, sincronizarAtajosVisuales } from "./preferencias.js";
import { idiomaAreaPersonal, iniciarI18nAreaPersonal } from "./i18n.js";
import { renderizarLlamamientos } from "./vistas/seguimiento-tramites.js";
import { peticionesEnSerie } from "../comun/imagen-propia.js";
import { catalogoPlano, lectorCatalogos } from "./textos-prueba.test-helper.mjs";

const valores = Object.freeze({ idioma: "en", tamano_texto: "grande", alto_contraste: true,
  tema: "oscuro", inicio: "bolsas", filas: 50, aviso_correo_tareas: true, aviso_correo_plazos: false });

test("el área personal usa su ruta exacta exterior, separada de RRHH", () => {
  assert.equal(RUTA_MIS_PREFERENCIAS, "/api/vec/usuarios/area-personal/mis-preferencias");
  assert.notEqual(RUTA_MIS_PREFERENCIAS, "/api/vec/usuarios/mis-preferencias");
});

test("el arranque continúa con error de servicio si preferencias no responde", async () => {
  let peticionAbortada = false;
  const cliente = crearClientePreferencias({ fetchImpl: async (_ruta, opciones) => {
    opciones.signal.addEventListener("abort", () => { peticionAbortada = true; });
    return new Promise(() => {});
  } });
  await assert.rejects(cargarPreferenciasIniciales(cliente, { tiempoMaximoMs: 10 }),
    (error) => error instanceof ErrorPreferencias && error.codigo === "servicio");
  assert.equal(peticionAbortada, true);
});
const catalogo = Object.freeze({ version_ref: "usuarios-preferencias-v1",
  idiomas: ["navegador", "es", "en"].map((codigo) => ({ codigo, nombre_key: codigo })),
  tamanos_texto: ["normal", "grande", "muy_grande"].map((codigo) => ({ codigo, nombre_key: codigo })),
  temas: ["sistema", "claro", "oscuro"].map((codigo) => ({ codigo, nombre_key: codigo })),
  inicios: ["cuadro", "peticiones", "bolsas"].map((codigo) => ({ codigo, nombre_key: codigo })),
  filas: [20, 50, 100], predeterminados: valores });
const estado = Object.freeze({ persona_ref: "persona:propia", version: 0,
  catalogo_version_ref: catalogo.version_ref, valores });
const json = (data, status = 200) => ({ status, headers: { get: (nombre) => nombre === "Content-Type" ? "application/json" : null },
  text: async () => JSON.stringify({ data }) });

test("la misma cola serializa preferencias e imagen y conserva 401, 403 y 503", async () => {
  let activas = 0; let maximo = 0;
  const rutas = [];
  const fetchUsuarios = peticionesEnSerie(async (ruta) => {
    rutas.push(ruta);
    activas += 1;
    maximo = Math.max(maximo, activas);
    await new Promise((resolve) => setTimeout(resolve, 10));
    activas -= 1;
    return new Response(JSON.stringify({ data: { catalogo, estado } }),
      { status: 200, headers: { "Content-Type": "application/json" } });
  });
  const cliente = crearClientePreferencias({ fetchImpl: fetchUsuarios });
  const [lectura, imagen] = await Promise.all([
    cliente.cargar(), fetchUsuarios("/api/vec/usuarios/area-personal/mi-imagen", { method: "GET" }),
  ]);
  assert.deepEqual(lectura.estado, estado);
  assert.equal(imagen.status, 200);
  assert.equal(maximo, 1);
  assert.deepEqual(rutas, [RUTA_MIS_PREFERENCIAS, "/api/vec/usuarios/area-personal/mi-imagen"]);
  for (const [estadoHTTP, codigo] of [[401, "autenticacion"], [403, "denegado"], [503, "servicio"]]) {
    let consultas = 0;
    const denegado = crearClientePreferencias({ fetchImpl: peticionesEnSerie(async () => {
      consultas += 1;
      return new Response("", { status: estadoHTTP });
    }) });
    await assert.rejects(cargarPreferenciasIniciales(denegado),
      (error) => error instanceof ErrorPreferencias && error.codigo === codigo);
    assert.equal(consultas, 1);
  }
});

test("GET y PUT usan el contrato único, usan credenciales de mismo origen y conservan recibo real", async () => {
  assert.equal(RUTA_MIS_PREFERENCIAS, "/api/vec/usuarios/area-personal/mis-preferencias");
  const codigoCliente = await readFile(new URL("./cliente-http.js", import.meta.url), "utf8");
  assert.doesNotMatch(codigoCliente, /["']\/api\/vec\/usuarios\/mis-preferencias["']/u);
  const llamadas = [];
  const recibo = { ...estado, version: 1, recibo_ref: "recibo:propio", fecha_utc: "2026-09-29T00:00:00Z", replay: false };
  const cliente = crearClientePreferencias({ fetchImpl: async (ruta, opciones) => {
    llamadas.push({ ruta, opciones });
    return opciones.method === "GET" ? json({ catalogo, estado }) : json(recibo, 201);
  } });
  assert.deepEqual(await cliente.cargar(), { catalogo, estado });
  const operacion = crearOperacionPreferencias({ catalogo, estado }, valores, { randomUUID: () => "clave-1" });
  assert.deepEqual(await cliente.guardar(operacion), recibo);
  assert.equal(llamadas.length, 2);
  assert.ok(llamadas.every(({ ruta, opciones }) => ruta === RUTA_MIS_PREFERENCIAS && opciones.credentials === "same-origin"));
  assert.deepEqual(JSON.parse(llamadas[1].opciones.body), operacion);
  assert.equal(llamadas[1].opciones.headers["X-Idempotency-Key"], undefined);
});

test("catálogo v2 carga estado v1 y guarda los seis temas con la versión vigente", async () => {
  const nuevos = ["diputacion_granada", "arena", "salvia", "lavanda", "azul_sereno", "noche_suave"];
  const catalogoV2 = { ...catalogo, version_ref: "usuarios-preferencias-v2",
    temas: [...catalogo.temas, ...nuevos.map((codigo) => ({ codigo, nombre_key: `ui.usuarios.preferencias.tema.${codigo}` }))] };
  const solicitudes = [];
  const cliente = crearClientePreferencias({ fetchImpl: async (_ruta, opciones) => {
    solicitudes.push(opciones);
    if (opciones.method === "GET") return json({ catalogo: catalogoV2, estado });
    const peticion = JSON.parse(opciones.body);
    return json({ ...estado, version: 1, catalogo_version_ref: "usuarios-preferencias-v2",
      valores: peticion.valores, recibo_ref: "recibo:tema", fecha_utc: "2026-09-30T00:00:00Z", replay: false }, 201);
  } });
  const leido = await cliente.cargar();
  assert.equal(leido.estado.catalogo_version_ref, "usuarios-preferencias-v1");
  assert.equal(leido.catalogo.version_ref, "usuarios-preferencias-v2");
  for (const tema of nuevos) {
    const valoresNuevos = { ...valores, tema };
    const operacion = crearOperacionPreferencias(leido, valoresNuevos, { randomUUID: () => "clave-v2" });
    const recibo = await cliente.guardar(operacion);
    assert.equal(recibo.valores.tema, tema);
    assert.equal(JSON.parse(solicitudes.at(-1).body).catalogo_version_ref, "usuarios-preferencias-v2");
    await assert.rejects(cliente.guardar({ ...operacion, catalogo_version_ref: "usuarios-preferencias-v1" }),
      (error) => error.codigo === "validacion");
  }
  const inverso = crearClientePreferencias({ fetchImpl: async () => json({ catalogo,
    estado: { ...estado, catalogo_version_ref: "usuarios-preferencias-v2" } }) });
  await assert.rejects(inverso.cargar(), (error) => error.codigo === "respuesta");
  const falsoHistorico = crearClientePreferencias({ fetchImpl: async () => json({ catalogo: catalogoV2,
    estado: { ...estado, valores: { ...valores, tema: "salvia" } } }) });
  await assert.rejects(falsoHistorico.cargar(), (error) => error.codigo === "respuesta");
});

test("rechaza cuerpo directo, respuesta sin recibo y errores HTTP sin confirmar guardado", async () => {
  await assert.rejects(crearClientePreferencias({ fetchImpl: async () => ({ status: 200,
    headers: { get: () => "application/json" }, text: async () => JSON.stringify({ catalogo, estado }) }) }).cargar(),
  (error) => error instanceof ErrorPreferencias && error.codigo === "respuesta");
  const cliente = crearClientePreferencias({ fetchImpl: async () => json({ ...estado, version: 1 }, 201) });
  await assert.rejects(cliente.guardar(crearOperacionPreferencias({ catalogo, estado }, valores,
    { randomUUID: () => "clave-2" })), (error) => error.codigo === "respuesta");
  for (const [status, codigo] of [[401, "autenticacion"], [403, "denegado"], [409, "conflicto"], [422, "validacion"], [503, "servicio"]]) {
    const denegado = crearClientePreferencias({ fetchImpl: async () => ({ status }) });
    await assert.rejects(denegado.cargar(), (error) => error.codigo === codigo && error.estado === status);
  }
});

test("el replay 200 conserva el recibo y exige el indicador de repetición", async () => {
  const operacion = crearOperacionPreferencias({ catalogo, estado }, valores, { randomUUID: () => "clave-replay" });
  const recibo = { ...estado, version: 1, recibo_ref: "recibo:original",
    fecha_utc: "2026-09-29T00:00:00Z", replay: true };
  const cliente = crearClientePreferencias({ fetchImpl: async () => json(recibo, 200) });
  assert.equal((await cliente.guardar(operacion)).recibo_ref, "recibo:original");
  const inconsistente = crearClientePreferencias({ fetchImpl: async () => json({ ...recibo, replay: false }, 200) });
  await assert.rejects(inconsistente.guardar(operacion), (error) => error.codigo === "respuesta");
});

test("la vista distingue error de lectura y confirmación con recibo; URL prevalece sobre servidor", async () => {
  await iniciarI18nAreaPersonal({ querySelectorAll: () => [] }, { leer: lectorCatalogos(), ubicacion: { href: "https://vec.example/area-personal/?lang=es" } });
  assert.match(renderizarPreferencias({ error: { codigo: "servicio" } }), /No se pudieron consultar sus preferencias/);
  assert.doesNotMatch(renderizarPreferencias({ error: { codigo: "servicio" } }), /<form/u);
  const html = renderizarPreferencias({ catalogo, estado, recibo: { recibo_ref: "recibo:propio" } });
  assert.match(html, /preferencias-exito[^>]*>Preferencias guardadas\./u);
  assert.doesNotMatch(html, /recibo:propio/u);
  assert.match(html, /name="filas"/u);
  assert.equal(idiomaAreaPersonal(["es-ES"], { href: "https://vec.example/area-personal/?lang=es" }, "en"), "es");
  assert.equal(idiomaAreaPersonal(["es-ES"], { href: "https://vec.example/area-personal/" }, "en"), "en");
});

test("la vista ofrece únicamente los nuevos temas incluidos en el catálogo v2", async () => {
  await iniciarI18nAreaPersonal({ querySelectorAll: () => [] }, { leer: lectorCatalogos(), ubicacion: { href: "https://vec.example/area-personal/?lang=es" } });
  const nuevos = ["diputacion_granada", "arena", "salvia", "lavanda", "azul_sereno", "noche_suave"];
  const catalogoV2 = { ...catalogo, version_ref: "usuarios-preferencias-v2",
    temas: [...catalogo.temas, ...nuevos.map((codigo) => ({ codigo, nombre_key: `ui.usuarios.preferencias.tema.${codigo}` }))] };
  const html = renderizarPreferencias({ catalogo: catalogoV2, estado: { ...estado, valores: { ...valores, tema: "salvia" } } });
  for (const codigo of nuevos) assert.match(html, new RegExp(`value="${codigo}"`, "u"));
  assert.match(html, /value="salvia" selected>Salvia/u);
  const reducido = renderizarPreferencias({ catalogo: { ...catalogoV2, temas: catalogoV2.temas.slice(0, -1) }, estado });
  assert.doesNotMatch(reducido, /value="noche_suave"/u);
});

test("la lista local respeta 20, 50 o 100 filas sin cambiar el transporte remoto", async () => {
  const fuente = await readFile(new URL("./vistas/seguimiento-tramites.js", import.meta.url), "utf8");
  assert.match(fuente, /\[20, 50, 100\]\.includes\(estado\.filasPreferidas\)/u);
  const programa = await readFile(new URL("./aplicacion.js", import.meta.url), "utf8");
  assert.match(programa, /estado\.paginaParticipaciones = 1;/u);
  assert.equal(typeof renderizarLlamamientos, "function");
});

test("los atajos de cabecera cambian texto y contraste solo en la sesión y reflejan aria-pressed", () => {
  const controles = { "alternar-texto": [], "alternar-contraste": [] };
  for (const accion of Object.keys(controles)) controles[accion].push({ atributos: {}, setAttribute(n, v) { this.atributos[n] = v; } });
  const documento = { querySelectorAll: (selector) => controles[selector.match(/"(.+)"/u)[1]] ?? [] };
  const aplicadas = [];
  let servidor = { tema: "claro", alto_contraste: false, tamano_texto: "normal" };
  const controlador = { leerEstado: () => ({ preferencias_servidor: servidor }),
    aplicarPreferenciasServidor(p) { aplicadas.push(p); servidor = p; } };
  assert.equal(alternarVisualSesion(controlador, "alternar-texto", documento), true);
  assert.deepEqual(servidor, { tema: "claro", alto_contraste: false, tamano_texto: "grande" });
  assert.equal(controles["alternar-texto"][0].atributos["aria-pressed"], "true");
  assert.equal(alternarVisualSesion(controlador, "alternar-contraste", documento), true);
  assert.equal(servidor.tamano_texto, "grande");
  assert.equal(controles["alternar-contraste"][0].atributos["aria-pressed"], "true");
  assert.equal(alternarVisualSesion(controlador, "alternar-texto", documento), false);
  assert.equal(aplicadas.length, 3);
  sincronizarAtajosVisuales({ tamano_texto: "muy_grande", alto_contraste: false }, documento);
  assert.equal(controles["alternar-texto"][0].atributos["aria-pressed"], "true");
  assert.equal(controles["alternar-contraste"][0].atributos["aria-pressed"], "false");
});
