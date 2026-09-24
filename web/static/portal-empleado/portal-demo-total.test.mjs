import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const base = new URL("./", import.meta.url);
const leer = (nombre) => readFile(new URL(nombre, base), "utf8");

test("el portal productivo no activa ni empaqueta el recorrido de presentación", async () => {
  const [portal, html, interno, produccion] = await Promise.all([
    leer("portal.js"), leer("index.html"), leer("../../interno.manifest"), leer("../../produccion.manifest"),
  ]);
  assert.match(portal, /modoPresentacion: false/u);
  assert.doesNotMatch(portal, /getAll\("presentacion"\)|getAll\("perfil"\)|\/presentacion\//u);
  assert.doesNotMatch(portal, /import\("[^"]*(?:datos-presentacion|portal-presentacion-adaptador|portal-borradores-demo-cliente|portal-resumen-presentacion|selector-perfiles)/u);
  assert.doesNotMatch(html, /aviso-presentacion|portal-llamamientos\.css|portal-baremacion\.css|portal-convocatorias\.css/u);
  for (const manifiesto of [interno, produccion]) {
    assert.doesNotMatch(manifiesto, /static\/portal-empleado\/(?:datos-presentacion|portal-presentacion-adaptador|portal-borradores-demo-cliente|portal-resumen-presentacion|portal-llamamientos-vista|portal-llamamientos\.css)/u);
  }
});
