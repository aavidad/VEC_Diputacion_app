import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import * as lectorReal from "../../../comun/textos.js";
import { prepararTextosCronos } from "./preparar-textos.js";
import { crearTraductorCronos } from "./i18n.js?v=20260929-i18n-textos-v1";
import { crearTraductorConsultaPermisosCronos } from "./i18n-permisos-consulta.js?v=20261001-cronos-c7-consulta-v2";
import { renderizarPermisosPropiosCronos } from "./vista-permisos-propios.js";

const { cargarTextos } = lectorReal;

test("Jornada y Permisos cargan solo sus catálogos y conservan el idioma activo", async () => {
  const jornada = [];
  const lectorJornada = { cargarTextos: (fuente) => {
    jornada.push(fuente);
    return cargarTextos(fuente, { idioma: "en", porDefecto: "en" });
  } };
  const preparada = await prepararTextosCronos({ pantalla: "jornada", lector: lectorJornada });
  assert.equal(preparada.idioma, "en");
  assert.deepEqual(jornada, ["cronos", "cronos-consulta", "cronos-fichaje", "cronos-incidencias"]);
  assert.equal(crearTraductorCronos()("titulo"), "Cronos");

  const permisos = [];
  const lectorPermisos = { cargarTextos: (fuente) => {
    permisos.push(fuente);
    return cargarTextos(fuente, { idioma: "en", porDefecto: "en" });
  } };
  await prepararTextosCronos({ pantalla: "permisos", lector: lectorPermisos });
  assert.deepEqual(permisos, ["cronos", "cronos-historial", "cronos-permisos", "cronos-permisos-consulta"]);
  assert.equal(crearTraductorConsultaPermisosCronos()("actualizar"), "Refresh");
  assert.match(renderizarPermisosPropiosCronos({ estado: "cargando", anio: 2026 }), /Leave and permits/);
});

test("un catálogo de Permisos fallido invalida textos capturados y el lector común lo recupera", async () => {
  await prepararTextosCronos({ pantalla: "jornada", lector: {
    cargarTextos: (fuente) => cargarTextos(fuente, { idioma: "es", porDefecto: "es" }),
  } });
  await prepararTextosCronos({ pantalla: "permisos", lector: {
    cargarTextos: (fuente) => cargarTextos(fuente, { idioma: "es", porDefecto: "es" }),
  } });
  const t = crearTraductorCronos();
  const tc = crearTraductorConsultaPermisosCronos();
  const anterior = t("jornada_titulo");
  await assert.rejects(prepararTextosCronos({ pantalla: "permisos", lector: {
    cargarTextos: (fuente) => fuente === "cronos-permisos" ? Promise.reject(new Error("red"))
      : cargarTextos(fuente, { idioma: "es", porDefecto: "es" }),
  } }), /red/);
  assert.throws(() => t("jornada_titulo"), /sustituido/);
  assert.throws(() => tc("actualizar"), /sustituido/);
  assert.throws(() => crearTraductorCronos(), /sin preparar/);
  const recuperadas = [];
  await prepararTextosCronos({ pantalla: "permisos", reintentar: true, lector: {
    reintentarTextos: (fuente) => { recuperadas.push(fuente); return cargarTextos(fuente, { idioma: "es", porDefecto: "es" }); },
  } });
  assert.equal(recuperadas.length, 4);
  assert.equal(crearTraductorCronos()("jornada_titulo"), anterior);
});

test("una recarga de Jornada fallida no deja utilizable su traductor anterior", async () => {
  await prepararTextosCronos({ pantalla: "jornada", lector: {
    cargarTextos: (fuente) => cargarTextos(fuente, { idioma: "en", porDefecto: "en" }),
  } });
  const anterior = crearTraductorCronos();
  assert.equal(anterior("jornada_titulo"), "My working hours");
  await assert.rejects(prepararTextosCronos({ pantalla: "jornada", reintentar: true, lector: {
    reintentarTextos: () => Promise.reject(new Error("catálogo inaccesible")),
  } }), /inaccesible/);
  assert.throws(() => anterior("jornada_titulo"), /sustituido/);
  assert.throws(() => crearTraductorCronos(), /sin preparar/);
});

test("un intento antiguo no publica textos después de un reintento más reciente", async () => {
  const pendiente = new Map();
  const antigua = prepararTextosCronos({ pantalla: "permisos", lector: {
    cargarTextos: (fuente) => new Promise((resolver) => pendiente.set(fuente, resolver)),
  } });
  const nueva = await prepararTextosCronos({ pantalla: "permisos", reintentar: true, lector: {
    reintentarTextos: (fuente) => cargarTextos(fuente, { idioma: "en", porDefecto: "en" }),
  } });
  assert.equal(nueva.idioma, "en");
  for (const [fuente, resolver] of pendiente) resolver(await cargarTextos(fuente, { idioma: "es", porDefecto: "es" }));
  await assert.rejects(antigua, /superada/);
  assert.equal(crearTraductorConsultaPermisosCronos()("actualizar"), "Refresh");
});

test("dos preparaciones normales simultáneas del mismo grupo conservan el resultado", async () => {
  const pendientes = new Map();
  const lenta = prepararTextosCronos({ pantalla: "jornada", lector: {
    cargarTextos: (fuente) => new Promise((resolver) => pendientes.set(fuente, resolver)),
  } });
  const rapida = prepararTextosCronos({ pantalla: "jornada", lector: {
    cargarTextos: (fuente) => cargarTextos(fuente, { idioma: "en", porDefecto: "en" }),
  } });
  assert.equal((await rapida).idioma, "en");
  const t = crearTraductorCronos();
  for (const [fuente, resolver] of pendientes) resolver(await cargarTextos(fuente, { idioma: "en", porDefecto: "en" }));
  assert.equal((await lenta).idioma, "en");
  assert.equal(t("jornada_titulo"), "My working hours");
});

test("el respaldo válido se aplica a todo el grupo sin mezclar idiomas", async () => {
  const lecturas = [];
  const lector = { cargarTextos: async (fuente, opciones) => {
    lecturas.push([fuente, opciones?.idioma ?? "activo"]);
    if (!opciones && fuente === "cronos-historial") {
      const texto = await cargarTextos(fuente, { idioma: "es", porDefecto: "es" });
      return { ...texto, incidenciaCatalogo: { idioma: "en", respaldo: "es" } };
    }
    return cargarTextos(fuente, opciones ?? { idioma: "en", porDefecto: "en" });
  } };
  const resultado = await prepararTextosCronos({ pantalla: "permisos", lector });
  assert.equal(resultado.idioma, "es");
  assert.deepEqual(resultado.incidenciaCatalogo, { idioma: "en", respaldo: "es" });
  assert.equal(resultado.incidenciaIndice, null);
  assert.deepEqual(lecturas.slice(4), [
    ["cronos", "es"], ["cronos-historial", "es"], ["cronos-permisos", "es"], ["cronos-permisos-consulta", "es"],
  ]);
  assert.equal(crearTraductorConsultaPermisosCronos()("actualizar"), "Actualizar");
});

test("Jornada lenta de otro idioma no sustituye Permisos más reciente", async () => {
  const pendientes = new Map();
  const jornadaLenta = prepararTextosCronos({ pantalla: "jornada", lector: {
    cargarTextos: (fuente) => new Promise((resolver) => pendientes.set(fuente, resolver)),
  } });
  const permisos = await prepararTextosCronos({ pantalla: "permisos", lector: {
    cargarTextos: (fuente) => cargarTextos(fuente, { idioma: "en", porDefecto: "en" }),
  } });
  assert.equal(permisos.idioma, "en");
  const t = crearTraductorCronos();
  for (const [fuente, resolver] of pendientes) resolver(await cargarTextos(fuente, { idioma: "es", porDefecto: "es" }));
  await assert.rejects(jornadaLenta, /superada/);
  assert.equal(t("jornada_titulo"), "My working hours");
});

test("el reintento usa la operación real del lector V", {
  skip: typeof lectorReal.reintentarTextos !== "function" && "la base aún no incluye V 9ec23fd1b",
}, async () => {
  const resultado = await prepararTextosCronos({ pantalla: "jornada", reintentar: true });
  assert.equal(resultado.pantalla, "jornada");
  assert.ok(resultado.idioma);
});

test("el respaldo real de V mantiene Permisos disponible en un solo idioma", {
  skip: typeof lectorReal.reintentarTextos !== "function" && "la base aún no incluye V 9ec23fd1b",
}, async () => {
  const leer = async (url) => {
    if (url.pathname.includes("/en/")) throw new Error("idioma activo no disponible");
    return JSON.parse(await readFile(url, "utf8"));
  };
  const lector = { cargarTextos: (fuente, opciones = {}) => lectorReal.cargarTextos(fuente, {
    idioma: opciones.idioma ?? "en", porDefecto: opciones.porDefecto ?? "es", leer,
  }) };
  const resultado = await prepararTextosCronos({ pantalla: "permisos", lector });
  assert.equal(resultado.idioma, "es");
  assert.equal(resultado.incidenciaCatalogo.codigo, "catalogo_no_disponible");
  assert.equal(crearTraductorConsultaPermisosCronos()("actualizar"), "Actualizar");
});

test("el índice fallido con catálogo íntegro permite Jornada y recupera la incidencia", {
  skip: typeof lectorReal.reintentarTextos !== "function" && "la base aún no incluye V 9ec23fd1b",
}, async () => {
  const incidenciaIndice = Object.freeze({ codigo: "indice_no_disponible", idioma: "es" });
  const lector = { cargarTextos: (fuente) => lectorReal.cargarTextos(fuente, {
    idioma: "es", porDefecto: "es", incidenciaIndice,
    leer: async (url) => JSON.parse(await readFile(url, "utf8")),
  }) };
  const provisional = await prepararTextosCronos({ pantalla: "jornada", lector });
  assert.equal(provisional.idioma, "es");
  assert.equal(provisional.incidenciaIndice, incidenciaIndice);
  assert.equal(provisional.incidenciaCatalogo, null);
  assert.equal(crearTraductorCronos()("jornada_titulo"), "Mi jornada");
  const recuperado = await prepararTextosCronos({ pantalla: "jornada", reintentar: true });
  assert.equal(recuperado.incidenciaIndice, null);
  assert.equal(recuperado.incidenciaCatalogo, null);
  assert.equal(crearTraductorCronos()("jornada_titulo"), "Mi jornada");
});

test("una localización inválida sigue bloqueando la publicación", async () => {
  const lector = { cargarTextos: async (fuente) => {
    const textos = await cargarTextos(fuente, { idioma: "es", porDefecto: "es" });
    return fuente === "cronos" ? { ...textos, localizacion: "invalida_!" } : textos;
  } };
  await assert.rejects(prepararTextosCronos({ pantalla: "jornada", lector }), /pendientes de recuperación/);
  assert.throws(() => crearTraductorCronos(), /sin preparar/);
});
