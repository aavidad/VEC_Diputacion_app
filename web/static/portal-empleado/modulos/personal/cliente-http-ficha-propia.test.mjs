import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { LIMITE_TEXTO_FICHA_PROPIA, RUTA_FICHA_PROPIA, crearFuentesFichaPropia } from "./cliente-http-ficha-propia.js";
import { LIMITE_TEXTO_CAMPO_FICHA } from "./vista-ficha-integral.js";
import { crearTraductorFichaPropia, formatearDiasFichaPropia } from "./i18n-ficha-propia.js";

const FICHA = Object.freeze({
  data: {
    ficha: {
      corte: { vigente_en: "2026-09-25", conocido_en: "2026-09-25T08:59:59.000000Z" },
      relaciones: [
        { inicio: "2026-01-01", fin: "", estado: "vigente", regimen: "Funcionario interino", modalidad: "Vacante", unidad: "Servicio de Personal", puesto: "Técnico/a de gestión", situacion: "Servicio activo" },
        { inicio: "2020-03-01", fin: "2020-12-31", estado: "finalizada", regimen: "Laboral temporal", modalidad: "", unidad: "", puesto: "", situacion: "" },
      ],
      servicios: [{ inicio: "2019-01-01", fin: "2019-12-31", clase: "Servicios previos", dias: 1365, estado: "reconocido" }],
    },
    recibo_ref: "fichapropia:0f0e0d0c-0b0a-4908-8706-050403020100",
    consultada_en: "2026-09-25T09:00:00.000000Z",
  },
});

function respuesta(cuerpo, estado = 200) {
  return new Response(typeof cuerpo === "string" ? cuerpo : JSON.stringify(cuerpo), { status: estado, headers: { "Content-Type": "application/json; charset=utf-8" } });
}

test("una sola consulta same-origin alimenta relaciones y servicios con textos legibles", async () => {
  const llamadas = [];
  const fuentes = await crearFuentesFichaPropia({ fetchImpl: async (ruta, opciones) => { llamadas.push([ruta, opciones]); return respuesta(FICHA); } }).preparar();
  assert.deepEqual(Object.keys(fuentes), ["relaciones", "servicios"]);
  const relaciones = await fuentes.relaciones.consultarPropios({ signal: new AbortController().signal });
  const servicios = await fuentes.servicios.consultarPropios({});
  assert.equal(llamadas.length, 1, "los dos apartados comparten la misma respuesta");
  const [ruta, opciones] = llamadas[0];
  assert.equal(ruta, RUTA_FICHA_PROPIA);
  assert.deepEqual([opciones.method, opciones.credentials, opciones.mode, opciones.cache, opciones.redirect, opciones.referrerPolicy],
    ["GET", "same-origin", "same-origin", "no-store", "error", "no-referrer"]);
  assert.equal(opciones.body, undefined);
  assert.equal(relaciones.estado, "disponible");
  assert.equal(relaciones.fuente, "Registro de Personal");
  assert.equal(relaciones.actualizado_en, "2026-09-25T09:00:00.000000Z");
  assert.deepEqual(relaciones.items[0], { desde: "2026-01-01", hasta: "Actualidad", regimen: "Funcionario interino · Vacante", puesto: "Técnico/a de gestión", unidad: "Servicio de Personal", estado: "Servicio activo" });
  assert.deepEqual(relaciones.items[1], { desde: "2020-03-01", hasta: "2020-12-31", regimen: "Laboral temporal", puesto: "", unidad: "", estado: "Finalizada" });
  assert.deepEqual(servicios.items, [{ desde: "2019-01-01", hasta: "2019-12-31", procedencia: "Servicios previos", reconocimiento: "1365 días", estado: "Reconocido" }]);
  assert.ok(!JSON.stringify([relaciones, servicios]).match(/(?:emp|per|rel|srv)_/u), "sin referencias internas");
});

test("sin fuente servida, sin permiso o con respuesta no válida los apartados no se ofrecen", async () => {
  const casos = [
    () => respuesta({ error: "no_encontrada" }, 404),
    () => respuesta({ error: "sin_empleado" }, 403),
    () => respuesta({ error: "autenticacion_requerida" }, 401),
    () => respuesta({ error: "no_disponible" }, 503),
    () => respuesta({ data: { ...FICHA.data, persona_ref: "per_x" } }),
    () => respuesta({ data: { ...FICHA.data, ficha: { ...FICHA.data.ficha, relaciones: [{ ...FICHA.data.ficha.relaciones[0], relacion_ref: "rel_x" }] } } }),
    () => respuesta({ data: { ...FICHA.data, ficha: { ...FICHA.data.ficha, servicios: [{ ...FICHA.data.ficha.servicios[0], estado: "pendiente" }] } } }),
    () => respuesta("no es JSON"),
    () => new Response(JSON.stringify(FICHA), { status: 200, headers: { "Content-Type": "text/html" } }),
    () => { throw new TypeError("red"); },
  ];
  for (const [indice, caso] of casos.entries()) {
    const fuentes = await crearFuentesFichaPropia({ fetchImpl: async () => caso() }).preparar();
    assert.deepEqual(fuentes, {}, `caso ${indice}`);
  }
});

test("la columna Estado da el estado de la relación si no está vigente; la abierta llega hasta la actualidad", async () => {
  const relaciones = [
    { inicio: "2026-02-01", fin: "", estado: "suspendida", regimen: "Laboral", modalidad: "", unidad: "", puesto: "", situacion: "Servicio activo" },
    { inicio: "2025-01-01", fin: "2025-12-31", estado: "finalizada", regimen: "Laboral", modalidad: "", unidad: "", puesto: "", situacion: "Excedencia voluntaria" },
    { inicio: "2026-03-01", fin: "", estado: "vigente", regimen: "Laboral", modalidad: "", unidad: "", puesto: "", situacion: "" },
  ];
  const sobre = { data: { ...FICHA.data, ficha: { ...FICHA.data.ficha, relaciones } } };
  const fuentes = await crearFuentesFichaPropia({ fetchImpl: async () => respuesta(sobre) }).preparar();
  const { items } = await fuentes.relaciones.consultarPropios({});
  assert.deepEqual(items.map((item) => [item.hasta, item.estado]),
    [["Actualidad", "Suspendida"], ["2025-12-31", "Finalizada"], ["Actualidad", "Vigente"]]);
});

test("límite de texto único: régimen y modalidad largos se recortan y la vista los admite", async () => {
  assert.equal(LIMITE_TEXTO_FICHA_PROPIA, 300);
  assert.equal(LIMITE_TEXTO_CAMPO_FICHA, LIMITE_TEXTO_FICHA_PROPIA, "cliente y vista comparten límite");
  const largo = "R".repeat(300); const emoji = "😀".repeat(150);
  const relaciones = [
    { inicio: "2026-01-01", fin: "", estado: "vigente", regimen: largo, modalidad: largo, unidad: largo, puesto: largo, situacion: largo },
    { inicio: "2026-01-01", fin: "", estado: "vigente", regimen: "A", modalidad: emoji, unidad: "", puesto: "", situacion: "" },
  ];
  const sobre = { data: { ...FICHA.data, ficha: { ...FICHA.data.ficha, relaciones } } };
  const fuentes = await crearFuentesFichaPropia({ fetchImpl: async () => respuesta(sobre) }).preparar();
  const { items } = await fuentes.relaciones.consultarPropios({});
  assert.equal(items[0].regimen.length, 300);
  assert.ok(items[0].regimen.endsWith("…"));
  assert.ok(items.every((item) => Object.values(item).every((valor) => valor.length <= LIMITE_TEXTO_CAMPO_FICHA)));
  assert.doesNotMatch(items[1].regimen, /[\ud800-\udbff](?![\udc00-\udfff])/u, "sin pares sustitutos partidos");
  const excedido = { data: { ...FICHA.data, ficha: { ...FICHA.data.ficha, relaciones: [{ ...relaciones[0], unidad: `${largo}x` }] } } };
  assert.deepEqual(await crearFuentesFichaPropia({ fetchImpl: async () => respuesta(excedido) }).preparar(), {});
});

test("más filas de las que se muestran: los apartados se ofrecen con estado propio, sin volver a consultar", async () => {
  const exceso = [
    () => respuesta({ error: "excede_limite" }, 422),
    () => respuesta({ data: { ...FICHA.data, ficha: { ...FICHA.data.ficha, servicios: Array.from({ length: 201 }, () => FICHA.data.ficha.servicios[0]) } } }),
  ];
  for (const [indice, caso] of exceso.entries()) {
    let llamadas = 0;
    const fuentes = await crearFuentesFichaPropia({ fetchImpl: async () => { llamadas += 1; return caso(); } }).preparar();
    assert.deepEqual(Object.keys(fuentes), ["relaciones", "servicios"], `caso ${indice}`);
    assert.deepEqual(await fuentes.relaciones.consultarPropios({}), { estado: "excede_limite" });
    assert.deepEqual(await fuentes.servicios.consultarPropios({}), { estado: "excede_limite" });
    assert.equal(llamadas, 1, `caso ${indice}`);
  }
  // Un 422 con otro código no es un exceso de filas.
  assert.deepEqual(await crearFuentesFichaPropia({ fetchImpl: async () => respuesta({ error: "otra_cosa" }, 422) }).preparar(), {});
});

test("una ficha sin registros deja los apartados vacíos, no en cero inventado", async () => {
  const vacia = { data: { ...FICHA.data, ficha: { ...FICHA.data.ficha, relaciones: [], servicios: [] } } };
  const fuentes = await crearFuentesFichaPropia({ fetchImpl: async () => respuesta(vacia) }).preparar();
  assert.deepEqual(await fuentes.servicios.consultarPropios({}), { estado: "vacio", fuente: "Registro de Personal", actualizado_en: "2026-09-25T09:00:00.000000Z", items: [] });
});

test("la consulta se cancela con la vista y respeta el plazo", async () => {
  const controlador = new AbortController();
  const fuentes = crearFuentesFichaPropia({ fetchImpl: (_ruta, { signal }) => new Promise((_resolver, rechazar) => signal.addEventListener("abort", () => rechazar(new Error("abortada")))) });
  const pendiente = fuentes.preparar({ signal: controlador.signal });
  controlador.abort();
  await assert.rejects(pendiente, (causa) => causa.codigo === "operacion_abortada");
  const conPlazo = crearFuentesFichaPropia({ plazoMs: 5, fetchImpl: (_ruta, { signal }) => new Promise((_resolver, rechazar) => signal.addEventListener("abort", () => rechazar(new Error("plazo")))) });
  assert.deepEqual(await conPlazo.preparar(), {});
  assert.throws(() => crearFuentesFichaPropia({ fetchImpl: null }), TypeError);
});

test("i18n de la ficha propia: plural de días y estados cerrados", () => {
  const t = crearTraductorFichaPropia();
  assert.equal(formatearDiasFichaPropia(1, t), "1 día");
  assert.equal(formatearDiasFichaPropia(0, t), "0 días");
  assert.equal(formatearDiasFichaPropia(12345, t), "12.345 días");
  assert.throws(() => t("estado_relacion_activa"), /desconocida/u);
  assert.throws(() => formatearDiasFichaPropia(-1, t), TypeError);
});

test("el cliente no usa almacenamiento del navegador ni credenciales entre orígenes", async () => {
  const fuente = await readFile(new URL("cliente-http-ficha-propia.js", import.meta.url), "utf8");
  assert.doesNotMatch(fuente, /localStorage|sessionStorage|indexedDB|document\.cookie|credentials:\s*"include"|Authorization/u);
  assert.match(fuente, /from "\.\/i18n-ficha-propia\.js\?v=[\w.-]+"/u);
});
