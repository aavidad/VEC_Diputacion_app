import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

test("el código productivo no contiene cookies, storage, demo ni reintentos", async () => {
  const fuente = await readFile(
    new URL("./cliente-http.js", import.meta.url),
    "utf8",
  );
  assert.doesNotMatch(
    fuente,
    /document\.cookie|localStorage|sessionStorage|indexedDB/u,
  );
  assert.doesNotMatch(
    fuente,
    /adaptador-presentacion|datos-presentacion|setTimeout|Retry-After/u,
  );
  assert.match(fuente, /credentials:\s*"omit"/u);
  assert.match(fuente, /mode:\s*"same-origin"/u);
});
