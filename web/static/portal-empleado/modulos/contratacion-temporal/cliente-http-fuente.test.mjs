import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

test("el código productivo no contiene cookies, storage, demo ni reintentos", async () => {
  const [cliente, alta] = await Promise.all([
    readFile(new URL("./cliente-http.js", import.meta.url), "utf8"),
    readFile(new URL("./cliente-http-alta.js", import.meta.url), "utf8"),
  ]);
  const fuente = `${cliente}\n${alta}`;
  assert.doesNotMatch(
    fuente,
    /document\.cookie|localStorage|sessionStorage|indexedDB/u,
  );
  assert.doesNotMatch(
    fuente,
    /adaptador-presentacion|datos-presentacion|setTimeout|Retry-After/u,
  );
  assert.match(cliente, /credentials:\s*"same-origin"/u);
  assert.match(cliente, /mode:\s*"same-origin"/u);
  assert.match(alta, /metodo:\s*"POST"/u);
  assert.match(alta, /catalogos-alta/u);
  assert.match(alta, /obtenerCatalogosAlta/u);
  assert.match(alta, /metodo:\s*"GET"/u);
  assert.doesNotMatch(fuente, /desarrolloNoAutoritativo|DEMO/u);
});
