import assert from "node:assert/strict";
import test from "node:test";
import { createHash } from "node:crypto";
import { readFile } from "node:fs/promises";

import { crearCalculadorRutasDietasHTTP } from "./calculador-rutas-http.js";
import {
  calcularHuellaFichero, catalogoOtrosGastosValido, crearLineaOtroGasto, describirOtroGasto, leerOtrosGastos,
} from "./formulario-otros-gastos.js";
import { crearTraductorOtrosGastosDietas, MENSAJES_OTROS_GASTOS_ES, rotuloTipoOtroGasto } from "./i18n-otros-gastos.js";
import { crearTraductorDietas } from "./i18n.js";

const CATALOGO = Object.freeze({ version: "provisional:otros-gastos:20260925", rotulo: "PROVISIONAL · pendiente de confirmación por RRHH",
  tipos: [{ codigo: "tren", clase: "otro_medio" }, { codigo: "taxi", clase: "otro_medio" }, { codigo: "peaje", clase: "otro_gasto" }] });
const t = crearTraductorOtrosGastosDietas(crearTraductorDietas());

function respuestaCatalogo(extra = {}) {
  const punto = (code, name, lat, lon) => ({ code, name, kind: "municipio", municipality_code: code, municipality_name: name,
    lat, lon, source: "Catalogo provincial gobernado", state: "Vigente" });
  return new Response(JSON.stringify({
    province_route_points: [punto("18087", "Granada", 37.1773, -3.5986), punto("18003", "Albolote", 37.2306, -3.6554)],
    province_route_matrix: { matrix_version: "osm-granada-2026-07-19", route_points_loaded: 2, import_required_before_liquidation: true },
    ...extra,
  }), { status: 200, headers: { "Content-Type": "application/json" } });
}

// DOM mínimo: etiquetas, hijos, propiedades y búsqueda por etiqueta o data-*.
class Nodo {
  constructor(documento, etiqueta) { Object.assign(this, { ownerDocument: documento, tagName: etiqueta, children: [], dataset: {}, textContent: "", value: "" }); }
  append(...nodos) { this.children.push(...nodos); nodos.forEach((nodo) => { nodo.parent = this; }); }
  replaceChildren(...nodos) { this.children = []; this.append(...nodos); }
  matches(selector) { return selector.startsWith("[data-") ? Object.hasOwn(this.dataset, selector.slice(6, -1).replace(/-([a-z])/gu, (_m, l) => l.toUpperCase())) : this.tagName === selector; }
  querySelectorAll(selector) { const salida = []; const visitar = (n) => { if (n.matches(selector)) salida.push(n); n.children.forEach(visitar); }; visitar(this); return salida; }
  querySelector(selector) { return this.querySelectorAll(selector)[0] || null; }
}
const documento = { createElement: (etiqueta) => new Nodo(documento, etiqueta) };

test("el calculador sirve los tipos versionados de otros gastos del mismo catálogo y falla cerrado si son incoherentes", async () => {
  let llamadas = 0;
  const calculador = crearCalculadorRutasDietasHTTP({ fetchImpl: async () => { llamadas += 1; return respuestaCatalogo({ otros_gastos: CATALOGO }); } });
  const catalogo = await calculador.obtenerCatalogoOtrosGastos();
  assert.equal(catalogo.version, CATALOGO.version);
  assert.deepEqual(catalogo.tipos.map((tipo) => tipo.codigo), ["tren", "taxi", "peaje"]);
  assert.ok(Object.isFrozen(catalogo.tipos[0]));
  await calculador.obtenerCatalogo();
  assert.equal(llamadas, 2);
  const anterior = crearCalculadorRutasDietasHTTP({ fetchImpl: async () => respuestaCatalogo() });
  assert.equal(await anterior.obtenerCatalogoOtrosGastos(), null);
  for (const otros_gastos of [{ ...CATALOGO, version: "v1" }, { ...CATALOGO, tipos: [{ codigo: "taxi", clase: "otro" }] },
    { ...CATALOGO, tipos: [{ codigo: "taxi", clase: "otro_medio" }, { codigo: "taxi", clase: "otro_medio" }] },
    { ...CATALOGO, tipos: [{ codigo: "taxi", clase: "otro_medio", importe_maximo: 1 }] }, { ...CATALOGO, tipos: [] }]) {
    const incoherente = crearCalculadorRutasDietasHTTP({ fetchImpl: async () => respuestaCatalogo({ otros_gastos }) });
    await assert.rejects(incoherente.obtenerCatalogoOtrosGastos(), /otros gastos/u);
  }
});

test("la línea D5 se lee con tipo, fecha dentro de la comisión, importe y justificante", () => {
  const form = documento.createElement("form");
  const catalogo = catalogoOtrosGastosValido(CATALOGO);
  const fila = crearLineaOtroGasto(documento, { traducir: t, catalogo, fechaInicio: "2026-09-23", fechaFin: "2026-09-24",
    valor: { tipo_gasto: "tren", fecha: "2026-09-23", concepto: "Tren a Motril", importe: "12,30",
      justificante_ref: "billete:0923", justificante_sha256: "C".repeat(64) } });
  form.append(fila);
  const fecha = fila.querySelectorAll("input").find((entrada) => entrada.name === "fecha");
  assert.equal(fecha.min, "2026-09-23");
  assert.equal(fecha.max, "2026-09-24");
  assert.ok(fila.querySelector("[data-dietas-otro-fichero]"));
  assert.deepEqual(leerOtrosGastos(form, catalogo, { fechaInicio: "2026-09-23", fechaFin: "2026-09-24" }), [{
    tipo: "otro_medio", tipo_gasto: "tren", catalogo_version: CATALOGO.version, fecha: "2026-09-23", concepto: "Tren a Motril",
    importe_centimos: 1230, justificante_ref: "billete:0923", justificante_sha256: "c".repeat(64) }]);
  const campo = (nombre) => [...fila.querySelectorAll("input"), ...fila.querySelectorAll("select")].find((entrada) => entrada.name === nombre);
  for (const [nombre, valor] of [["tipo_gasto", ""], ["fecha", "2026-09-25"], ["fecha", "2026-02-30"], ["concepto", "ab"],
    ["importe", "0"], ["importe", "1.234"], ["justificante_ref", "billete 0923"], ["justificante_sha256", "c".repeat(63)]]) {
    const previo = campo(nombre).value;
    campo(nombre).value = valor;
    assert.throws(() => leerOtrosGastos(form, catalogo, { fechaInicio: "2026-09-23", fechaFin: "2026-09-24" }), TypeError, `${nombre}=${valor}`);
    campo(nombre).value = previo;
  }
  assert.throws(() => leerOtrosGastos(form, null, { fechaInicio: "2026-09-23", fechaFin: "2026-09-24" }), TypeError);
  assert.deepEqual(leerOtrosGastos(documento.createElement("form"), null, { fechaInicio: "2026-09-23", fechaFin: "2026-09-24" }), []);
});

test("la huella SHA-256 se calcula en el navegador sin enviar el fichero y acota su tamaño", async () => {
  const contenido = "billete sintético";
  const esperado = createHash("sha256").update(contenido).digest("hex");
  assert.equal(await calcularHuellaFichero(new Blob([contenido])), esperado);
  await assert.rejects(calcularHuellaFichero(new Blob([])), TypeError);
  await assert.rejects(calcularHuellaFichero({ size: 26 * 1024 * 1024, arrayBuffer: async () => new ArrayBuffer(0) }), TypeError);
  await assert.rejects(calcularHuellaFichero(undefined), TypeError);
});

test("las líneas se describen con rótulos de negocio y conservan las anteriores a D5", () => {
  const fecha = (valor) => `día ${valor}`;
  assert.equal(describirOtroGasto({ tipo: "otro_medio", tipo_gasto: "taxi", fecha: "2026-09-23", concepto: "Estación" }, t, fecha),
    "Taxi · día 2026-09-23 · Estación");
  assert.equal(describirOtroGasto({ tipo: "otro_gasto", concepto: "Peaje anterior" }, t, fecha), "Peaje anterior");
  assert.equal(rotuloTipoOtroGasto(t, "restaurante"), "Otro tipo");
  assert.equal(rotuloTipoOtroGasto(t, "__proto__"), "Otro tipo");
});

test("todo tipo del catálogo publicado tiene rótulo y la extensión se monta en vista y bandeja", async () => {
  const go = await readFile(new URL("../../../../../internal/modules/dietas/domain/otros_gastos.go", import.meta.url), "utf8");
  const codigos = [...go.matchAll(/\{"([a-z_]+)", ClaseOtro(?:Medio|Gasto)\}/gu)].map(([, codigo]) => codigo);
  assert.ok(codigos.length >= 2);
  for (const codigo of codigos) assert.ok(Object.hasOwn(MENSAJES_OTROS_GASTOS_ES, `otros_gastos_tipo_${codigo}`), codigo);
  const [vista, bandeja] = await Promise.all(["./vista-borradores-propios.js", "./vista-bandeja-circuito.js"]
    .map((ruta) => readFile(new URL(ruta, import.meta.url), "utf8")));
  for (const fuente of [vista, bandeja]) {
    assert.match(fuente, /i18n-otros-gastos\.js\?v=/u);
    assert.match(fuente, /formulario-otros-gastos\.js\?v=/u);
  }
  for (const texto of Object.values(MENSAJES_OTROS_GASTOS_ES))
    assert.doesNotMatch(texto, /\bD5\b|provisional:|otro_medio|otro_gasto|catalogo_version/u);
});
