import assert from "node:assert/strict";
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

test("un catálogo de Permisos fallido no borra los textos de Jornada y el lector común lo recupera", async () => {
  await prepararTextosCronos({ pantalla: "jornada", lector: {
    cargarTextos: (fuente) => cargarTextos(fuente, { idioma: "es", porDefecto: "es" }),
  } });
  const t = crearTraductorCronos();
  const anterior = t("jornada_titulo");
  await assert.rejects(prepararTextosCronos({ pantalla: "permisos", lector: {
    cargarTextos: (fuente) => fuente === "cronos-permisos" ? Promise.reject(new Error("red"))
      : cargarTextos(fuente, { idioma: "es", porDefecto: "es" }),
  } }), /red/);
  assert.equal(crearTraductorCronos()("jornada_titulo"), anterior);
  const recuperadas = [];
  await prepararTextosCronos({ pantalla: "permisos", reintentar: true, lector: {
    reintentarTextos: (fuente) => { recuperadas.push(fuente); return cargarTextos(fuente, { idioma: "es", porDefecto: "es" }); },
  } });
  assert.equal(recuperadas.length, 4);
  assert.equal(crearTraductorCronos()("jornada_titulo"), anterior);
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

test("el reintento usa la operación real del lector V", {
  skip: typeof lectorReal.reintentarTextos !== "function" && "la base aún no incluye V 9ec23fd1b",
}, async () => {
  const resultado = await prepararTextosCronos({ pantalla: "jornada", reintentar: true });
  assert.equal(resultado.pantalla, "jornada");
  assert.ok(resultado.idioma);
});
