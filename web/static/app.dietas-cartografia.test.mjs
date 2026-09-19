import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const app = await readFile(new URL("./app.js", import.meta.url), "utf8");

test("Dietas separa el mediador de cartografía de presentación del endpoint interno", () => {
  assert.match(app, /const PRESENTATION_CARTOGRAPHY_ROUTE_API = "\/api\/presentacion\/cartografia\/rutas"/u);
  assert.match(app, /new URLSearchParams\(window\.location\.search\)\.get\("presentacion"\) === "rrhh"/u);
  assert.match(app, /presentation \? PRESENTATION_CARTOGRAPHY_ROUTE_API : DIETAS_ROAD_ROUTE_API/u);
  assert.match(app, /headers: presentation \? \{ "Content-Type": "application\/json" \} : staffHeaders\(\)/u);
  assert.match(app, /Cartografía interna no disponible en esta presentación/u);
  assert.match(app, /presentación no autoritativa y no liquidable/u);
});
