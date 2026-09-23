import test from "node:test";
import assert from "node:assert/strict";
import { montarVistaEstadisticas } from "./vista-estadisticas.js";

function diferida() {
  let resolver;
  let rechazar;
  const promesa = new Promise((resolve, reject) => {
    resolver = resolve;
    rechazar = reject;
  });
  return { promesa, resolver, rechazar };
}

function raizFalsa() {
  let html = "";
  let escrituras = 0;
  const eventos = new Map();
  return {
    eventos,
    get innerHTML() { return html; },
    set innerHTML(valor) { html = valor; escrituras++; },
    get escrituras() { return escrituras; },
    addEventListener(tipo, manejador) { eventos.set(tipo, manejador); },
    removeEventListener(tipo, manejador) {
      if (eventos.get(tipo) === manejador) eventos.delete(tipo);
    },
  };
}

function respuesta(inicio) {
  const conteos = { altas: 1, llamamientos: 0, formalizaciones: 0, cierres: 0, incidencias: 0 };
  return {
    ok: true,
    datos: { periodo: "mensual", series: [{ inicio, ...conteos }], totales: conteos },
  };
}

test("dos consultas inversas: solo la última actualiza estado, pantalla y anuncio", async () => {
  const a = diferida();
  const b = diferida();
  const raiz = raizFalsa();
  const anuncios = [];
  let llamadas = 0;
  const vista = montarVistaEstadisticas({
    raiz,
    cliente: () => [a, b][llamadas++].promesa,
    anunciar: (mensaje) => anuncios.push(mensaje),
  });

  const segundaCarga = vista.cargar();
  b.resolver(respuesta("2026-02-01"));
  await segundaCarga;
  assert.equal(vista.obtenerEstado().carga, "listo");
  assert.match(raiz.innerHTML, /2026-02-01/);
  const htmlFinal = raiz.innerHTML;

  a.resolver(respuesta("2026-01-01"));
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(raiz.innerHTML, htmlFinal);
  assert.equal(vista.obtenerEstado().datos.series[0].inicio, "2026-02-01");
  assert.deepEqual(anuncios, ["Estadísticas actualizadas"]);
  vista.desmontar();
});

test("desmontar invalida la respuesta pendiente y nuevas cargas", async () => {
  const pendiente = diferida();
  const raiz = raizFalsa();
  const anuncios = [];
  let llamadas = 0;
  const vista = montarVistaEstadisticas({
    raiz,
    cliente: () => { llamadas++; return pendiente.promesa; },
    anunciar: (mensaje) => anuncios.push(mensaje),
  });
  vista.desmontar();
  const escriturasAlDesmontar = raiz.escrituras;
  pendiente.resolver(respuesta("2026-03-01"));
  await new Promise((resolve) => setImmediate(resolve));
  await vista.cargar();

  assert.equal(raiz.innerHTML, "");
  assert.equal(raiz.escrituras, escriturasAlDesmontar);
  assert.equal(llamadas, 1);
  assert.deepEqual(anuncios, []);
  assert.equal(raiz.eventos.size, 0);
});

test("rechazo del cliente muestra error controlado y permite reintentar", async () => {
  const raiz = raizFalsa();
  let llamadas = 0;
  const vista = montarVistaEstadisticas({
    raiz,
    cliente: async () => {
      llamadas++;
      if (llamadas === 1) throw new Error("detalle interno");
      return respuesta("2026-04-01");
    },
  });
  await new Promise((resolve) => setImmediate(resolve));
  assert.equal(vista.obtenerEstado().carga, "error");
  assert.match(raiz.innerHTML, /No se pudieron obtener las series estadísticas/);
  assert.doesNotMatch(raiz.innerHTML, /detalle interno/);
  assert.match(raiz.innerHTML, /data-ct-accion="reintentar-estadisticas"/);

  await vista.cargar();
  assert.equal(vista.obtenerEstado().carga, "listo");
  assert.match(raiz.innerHTML, /2026-04-01/);
  vista.desmontar();
});
