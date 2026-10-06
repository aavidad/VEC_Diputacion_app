import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const app = await readFile(new URL("./app.js", import.meta.url), "utf8");

test("Dietas calcula rutas solo con el endpoint interno; el mediador de presentación está retirado", () => {
  assert.match(app, /const payload = await getData\(DIETAS_ROAD_ROUTE_API, \{/u);
  assert.match(app, /headers: staffHeaders\(\),/u);
  assert.doesNotMatch(app, /\/api\/presentacion\/cartografia|PRESENTATION_CARTOGRAPHY_ROUTE_API|isRRHHPresentationMode/u);
  assert.doesNotMatch(app, /get\("presentacion"\)/u);
});
