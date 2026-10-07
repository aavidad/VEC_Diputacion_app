import test from "node:test";
import assert from "node:assert/strict";
import { iniciarRegistroErrores, registrarErrorCliente } from "./registro-errores.js";

test("el registro omite mensaje, pila, URL y datos libres", async () => {
  const previoFetch = globalThis.fetch;
  const previaLocation = globalThis.location;
  const peticiones = [];
  globalThis.location = { pathname: "/portal-empleado/expedientes/persona:123", search: "?dni=12345678Z" };
  globalThis.fetch = async (ruta, opciones) => { peticiones.push({ ruta, opciones }); return { ok: true }; };
  try {
    assert.equal(registrarErrorCliente({ codigo: "CLIENTE_FALLO_NO_CLASIFICADO", pantalla: "portal_empleado", mensaje: "DNI 12345678Z", stack: "secreto" }), true);
    assert.equal(registrarErrorCliente({ codigo: "DNI 12345678Z" }), false);
    assert.equal(peticiones.length, 1);
    const { ruta, opciones } = peticiones[0];
    assert.equal(ruta, "/api/vec/observabilidad/errores-cliente");
    assert.deepEqual(Object.keys(JSON.parse(opciones.body)).sort(), ["codigo", "correlacion", "pantalla"]);
    assert.match(JSON.parse(opciones.body).correlacion, /^[0-9a-f]{32}$/u);
    assert.doesNotMatch(JSON.stringify(peticiones), /12345678Z|secreto|persona:123|dni=/u);
    assert.equal(opciones.credentials, "same-origin");
    assert.equal(opciones.referrerPolicy, "no-referrer");
    assert.equal(opciones.keepalive, true);
  } finally { globalThis.fetch = previoFetch; globalThis.location = previaLocation; }
});

test("window.error y unhandledrejection registran códigos cerrados sin leer eventos", async () => {
  const previoFetch = globalThis.fetch;
  const previaLocation = globalThis.location;
  const eventos = new Map();
  const cuerpos = [];
  globalThis.location = { pathname: "/portal-empleado/", search: "?dato=privado" };
  globalThis.fetch = async (_ruta, opciones) => { cuerpos.push(JSON.parse(opciones.body)); return { ok: true }; };
  try {
    const ventana = { addEventListener: (tipo, callback) => eventos.set(tipo, callback),
      removeEventListener: (tipo) => eventos.delete(tipo) };
    const detenerUno = iniciarRegistroErrores(ventana);
    const detenerDos = iniciarRegistroErrores(ventana);
    assert.equal(eventos.size, 2);
    eventos.get("error")({ message: "dato privado", filename: "/persona/123" });
    eventos.get("unhandledrejection")({ reason: new Error("dato privado") });
    assert.equal(cuerpos.length, 2);
    assert.ok(cuerpos.every((cuerpo) => cuerpo.codigo === "CLIENTE_FALLO_NO_CLASIFICADO" && cuerpo.pantalla === "portal_empleado"));
    detenerUno();
    assert.equal(eventos.size, 2);
    detenerDos();
    assert.equal(eventos.size, 0);
  } finally { globalThis.fetch = previoFetch; globalThis.location = previaLocation; }
});

test("el límite local evita ráfagas y no envía desde otra pantalla", async () => {
  const previoFetch = globalThis.fetch;
  const previaLocation = globalThis.location;
  const previoAhora = Date.now;
  let envios = 0;
  Date.now = () => previoAhora() + 61_000;
  globalThis.location = { pathname: "/portal-empleado/" };
  globalThis.fetch = async () => { envios++; return { ok: true }; };
  try {
    for (let i = 0; i < 5; i++) assert.equal(registrarErrorCliente(), true);
    assert.equal(registrarErrorCliente(), false);
    assert.equal(envios, 5);
    globalThis.location = { pathname: "/area-personal/" };
    assert.equal(registrarErrorCliente(), false);
  } finally { Date.now = previoAhora; globalThis.fetch = previoFetch; globalThis.location = previaLocation; }
});
