import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
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

test("la marca institucional permanece visible al desplazarse el menú lateral", async () => {
  const estilos = await readFile(new URL("./portal.css", import.meta.url), "utf8");
  assert.match(estilos, /\.marca-portal\s*\{[\s\S]*?position:\s*sticky;[\s\S]*?top:\s*0;/u);
  assert.match(estilos, /\.marca-portal\s*\{[\s\S]*?flex:\s*0 0 auto;/u);
});
