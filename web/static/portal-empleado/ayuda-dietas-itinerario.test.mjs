import assert from "node:assert/strict";
import test from "node:test";
import { crearAyudanteTramites } from "./ayudante-tramites.js";

test("las instrucciones de alternativas y motivo D4 se consultan en el ayudante existente", () => {
  let pulsar;
  let navegaciones = 0;
  const contenedor = {
    innerHTML: "",
    querySelector: () => null,
    addEventListener: (_tipo, accion) => { pulsar = accion; },
    removeEventListener: () => { pulsar = null; },
  };
  const ayudante = crearAyudanteTramites();
  assert.doesNotMatch(ayudante.contenido, /Puede previsualizar otra alternativa|Escriba entre 8 y 500/u);
  const desmontar = ayudante.instalar({ contenedor, documento: { querySelector: () => null }, navegar: () => { navegaciones += 1; } });
  const clic = (dataset, atributo = "") => pulsar({ target: { closest: () => ({ dataset, hasAttribute: (nombre) => nombre === atributo }) } });
  clic({ ayudanteTramite: "dietas-ruta" });
  assert.doesNotMatch(contenedor.innerHTML, /Puede previsualizar otra alternativa/u);
  clic({}, "data-ayudante-siguiente");
  assert.match(contenedor.innerHTML, /Puede previsualizar otra alternativa OSRM/u);
  assert.match(contenedor.innerHTML, /Escriba entre 8 y 500 caracteres/u);
  assert.match(contenedor.innerHTML, /no se guardan ni generan importe/u);
  assert.equal(navegaciones, 0, "consultar las instrucciones no navega ni ejecuta el trámite");
  desmontar();
  assert.equal(pulsar, null);
});
