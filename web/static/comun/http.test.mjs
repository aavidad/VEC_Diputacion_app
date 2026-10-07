import test from "node:test";
import assert from "node:assert/strict";
import { consultarJSON, ErrorConsultaJSON } from "./http.js";

function respuesta(datos, estado = 200, cabeceras = {}) {
  return new Response(JSON.stringify(datos), { status: estado, headers: { "Content-Type": "application/json", ...cabeceras } });
}

test("solo admite rutas internas canónicas y fija la frontera de fetch", async () => {
  let opciones;
  const dato = await consultarJSON("/api/vec/session", { fetchImpl: async (_ruta, o) => { opciones = o; return respuesta({ ok: true }); } });
  assert.deepEqual(dato, { ok: true });
  assert.equal(opciones.mode, "same-origin");
  assert.equal(opciones.credentials, "same-origin");
  assert.equal(opciones.cache, "no-store");
  assert.equal(opciones.redirect, "error");
  assert.equal(opciones.referrerPolicy, "no-referrer");
  for (const ruta of ["https://evil.test/api/vec/session", "//evil.test/api/vec/session", "/api/vec/../externo", "/api/vec/%2e%2e/externo", "/api/vec/a#b"]) {
    assert.throws(() => consultarJSON(ruta), TypeError);
  }
});

test("reintenta GET 502, 503 y red; nunca reintenta POST", async () => {
  let intentos = 0;
  const dato = await consultarJSON("/api/vec/session", { fetchImpl: async () => {
    intentos++;
    if (intentos === 1) throw new Error("desconectado");
    if (intentos === 2) return respuesta({}, 503);
    return respuesta({ listo: true });
  } });
  assert.deepEqual(dato, { listo: true });
  assert.equal(intentos, 3);
  intentos = 0;
  assert.deepEqual(await consultarJSON("/api/vec/session", { reintentos: 1, fetchImpl: async () => {
    intentos++;
    return intentos === 1 ? respuesta({}, 502) : respuesta({ recuperado: true });
  } }), { recuperado: true });
  assert.equal(intentos, 2);
  intentos = 0;
  await assert.rejects(consultarJSON("/api/vec/operacion", { metodo: "POST", cuerpo: { a: 1 }, fetchImpl: async () => {
    intentos++;
    return respuesta({}, 503);
  } }), (fallo) => fallo instanceof ErrorConsultaJSON && fallo.codigo === "no_disponible" && fallo.estado === 503);
  assert.equal(intentos, 1);
});

test("si todos los lectores cancelan se aborta GET compartido y se libera la clave", async () => {
  let llamadas = 0;
  let señalInterna;
  const fetchImpl = (_ruta, opciones) => { llamadas++; señalInterna = opciones.signal; return new Promise(() => {}); };
  const uno = new AbortController();
  const dos = new AbortController();
  const primera = consultarJSON("/api/vec/session", { fetchImpl, signal: uno.signal });
  const segunda = consultarJSON("/api/vec/session", { fetchImpl, signal: dos.signal });
  await Promise.resolve();
  uno.abort();
  assert.equal(señalInterna.aborted, false);
  dos.abort();
  await Promise.all([assert.rejects(primera), assert.rejects(segunda)]);
  assert.equal(señalInterna.aborted, true);
  const tres = new AbortController();
  const tercera = consultarJSON("/api/vec/session", { fetchImpl, signal: tres.signal });
  await Promise.resolve();
  assert.equal(llamadas, 2);
  tres.abort();
  await assert.rejects(tercera);
});

test("deduplica GET y la cancelación de un lector conserva el otro", async () => {
  let intentos = 0;
  let entregar;
  const fetchImpl = () => { intentos++; return new Promise((resolver) => { entregar = resolver; }); };
  const primero = new AbortController();
  const segundo = new AbortController();
  const uno = consultarJSON("/api/vec/session", { fetchImpl, signal: primero.signal });
  const dos = consultarJSON("/api/vec/session", { fetchImpl, signal: segundo.signal });
  await Promise.resolve();
  assert.equal(intentos, 1);
  primero.abort();
  await assert.rejects(uno, (fallo) => fallo.codigo === "cancelado");
  entregar(respuesta({ sesion: 1 }));
  assert.deepEqual(await dos, { sesion: 1 });
  assert.equal(intentos, 1);
});

test("dos GET idénticos reciben la misma respuesta y uno posterior consulta otra vez", async () => {
  let llamadas = 0;
  let entregar;
  const fetchImpl = () => { llamadas++; return new Promise((resolve) => { entregar = resolve; }); };
  const uno = consultarJSON("/api/vec/session", { fetchImpl });
  const dos = consultarJSON("/api/vec/session", { fetchImpl });
  assert.strictEqual(uno, dos);
  await Promise.resolve();
  entregar(respuesta({ mismo: true }));
  assert.strictEqual(await uno, await dos);
  assert.equal(llamadas, 1);
  const tres = consultarJSON("/api/vec/session", { fetchImpl });
  await Promise.resolve();
  entregar(respuesta({ siguiente: true }));
  assert.deepEqual(await tres, { siguiente: true });
  assert.equal(llamadas, 2);
});

test("rechaza exceso de bytes, JSON inválido, redirección y estados tipados", async () => {
  await assert.rejects(consultarJSON("/api/vec/session", { limiteBytes: 8, fetchImpl: async () => respuesta({ dato: "largo" }) }),
    (fallo) => fallo.codigo === "respuesta_no_valida");
  await assert.rejects(consultarJSON("/api/vec/session", { fetchImpl: async () => new Response("{", { headers: { "Content-Type": "application/json" } }) }),
    (fallo) => fallo.codigo === "respuesta_no_valida");
  await assert.rejects(consultarJSON("/api/vec/session", { fetchImpl: async () => ({ ok: true, status: 200, redirected: true }) }),
    (fallo) => fallo.codigo === "no_disponible");
  for (const [estado, codigo] of [[401, "sin_permiso"], [403, "sin_permiso"], [404, "no_encontrado"], [409, "conflicto"]]) {
    await assert.rejects(consultarJSON("/api/vec/session", { fetchImpl: async () => respuesta({}, estado) }),
      (fallo) => fallo.codigo === codigo && fallo.estado === estado);
  }
});

test("plazo máximo cancela incluso si fetch ignora AbortSignal", async () => {
  await assert.rejects(consultarJSON("/api/vec/session", { plazoMs: 5, reintentos: 0, fetchImpl: () => new Promise(() => {}) }),
    (fallo) => fallo.codigo === "red");
});
