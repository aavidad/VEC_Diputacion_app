import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { crearClienteBorradoresPresentacion } from "./portal-borradores-demo-cliente.js";
import {
  enfocarYMostrarResultado,
  restaurarFocoTrasReintentoBorradores,
} from "./portal-eventos.js";

test("el recibo dinámico conserva el foco y queda visible respetando movimiento reducido", () => {
  const llamadas = [];
  const elemento = {
    focus(opciones) { llamadas.push(["foco", opciones]); },
    scrollIntoView(opciones) { llamadas.push(["desplazamiento", opciones]); },
  };

  assert.equal(enfocarYMostrarResultado(elemento, { movimientoReducido: false }), true);
  assert.deepEqual(llamadas, [
    ["foco", { preventScroll: true }],
    ["desplazamiento", { behavior: "smooth", block: "center", inline: "nearest" }],
  ]);

  llamadas.length = 0;
  assert.equal(enfocarYMostrarResultado(elemento, { movimientoReducido: true }), true);
  assert.equal(llamadas[1][1].behavior, "auto");
  assert.equal(enfocarYMostrarResultado(null), false);
});

test("el reintento devuelve el foco al control renovado o a la tarjeta de Bolsa", () => {
  const focos = [];
  const control = { focus: (opciones) => focos.push(["control", opciones]) };
  const tarjeta = {
    focus: (opciones) => focos.push(["tarjeta", opciones]),
    querySelector: () => control,
  };
  assert.equal(restaurarFocoTrasReintentoBorradores({ querySelector: () => tarjeta }), true);
  assert.deepEqual(focos, [["control", { preventScroll: true }]]);

  tarjeta.querySelector = () => null;
  assert.equal(restaurarFocoTrasReintentoBorradores({ querySelector: () => tarjeta }), true);
  assert.deepEqual(focos.at(-1), ["tarjeta", { preventScroll: true }]);
  assert.equal(restaurarFocoTrasReintentoBorradores({ querySelector: () => null }), false);
});

test("las huellas de presentación son SHA-256 sintéticas creíbles y diferenciadas", async () => {
  const cliente = crearClienteBorradoresPresentacion({
    reloj: () => new Date("2026-07-20T12:00:00Z"),
  });
  const opciones = await cliente.obtenerOpciones();
  const detalle = await cliente.obtenerDetalle();
  const huellas = [
    opciones.categorias[0].huella_sha256,
    opciones.tipos[0].huella_sha256,
    detalle.referencia_estado.huella_estado_sha256,
  ];

  for (const huella of huellas) {
    assert.match(huella, /^[0-9a-f]{64}$/u);
    assert.ok(new Set(huella).size >= 10, "la huella no debe parecer un relleno monótono");
  }
  assert.notEqual(huellas[0], huellas[1]);
});

test("la marca institucional permanece visible al desplazarse el menú lateral", async () => {
  const estilos = await readFile(new URL("./portal.css", import.meta.url), "utf8");
  assert.match(estilos, /\.marca-portal\s*\{[\s\S]*?position:\s*sticky;[\s\S]*?top:\s*0;/u);
  assert.match(estilos, /\.marca-portal\s*\{[\s\S]*?flex:\s*0 0 auto;/u);
});
