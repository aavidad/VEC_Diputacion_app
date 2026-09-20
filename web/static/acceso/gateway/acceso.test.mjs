import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { crearClienteGateway, ErrorGateway, validarEstadoSesion } from "./acceso.js";

function respuesta(datos, status = 200) {
  const texto = JSON.stringify(datos);
  return { status, headers: { get: () => "application/json; charset=utf-8" }, text: async () => texto };
}

test("el cliente consume sólo el estado cerrado de sesión y no envía cuerpo ni autorización", async () => {
  const llamadas = [];
  const cliente = crearClienteGateway({ fetchImpl: async (ruta, opciones) => {
    llamadas.push({ ruta, opciones });
    return respuesta({ data: { autenticada: true, expira_en: "2026-09-20T12:00:00Z" } });
  } });
  const estado = await cliente.iniciarCertificado();
  assert.deepEqual(estado, { autenticada: true, expira_en: "2026-09-20T12:00:00Z" });
  assert.equal(llamadas[0].ruta, "/api/acceso/certificado");
  assert.equal(llamadas[0].opciones.method, "POST");
  assert.equal(llamadas[0].opciones.credentials, "same-origin");
  assert.equal(llamadas[0].opciones.cache, "no-store");
  assert.equal(llamadas[0].opciones.body, undefined);
  assert.equal(llamadas[0].opciones.headers.Authorization, undefined);
});

test("el contrato rechaza envolturas, campos y expiraciones ambiguos", () => {
  assert.throws(() => validarEstadoSesion({ autenticada: true, expira_en: "2026-09-20T12:00:00Z" }), ErrorGateway);
  assert.throws(() => validarEstadoSesion({ data: { autenticada: true, expira_en: "2026-09-20T12:00:00Z", cuenta: "no" } }), ErrorGateway);
  assert.throws(() => validarEstadoSesion({ data: { autenticada: false, expira_en: "2026-09-20T12:00:00Z" } }), ErrorGateway);
});

test("la denegación de certificado y el cierre se tratan sin leer identidad", async () => {
  const rechazado = crearClienteGateway({ fetchImpl: async () => ({ status: 403 }) });
  await assert.rejects(() => rechazado.iniciarCertificado(), (error) => error.codigo === "rechazado");
  let opciones;
  const cliente = crearClienteGateway({ fetchImpl: async (_, recibidas) => { opciones = recibidas; return { status: 204 }; } });
  await cliente.cerrarSesion();
  assert.equal(opciones.method, "POST");
  assert.equal(opciones.body, undefined);
});

test("la superficie no contiene campos de credencial ni almacenamiento web", async () => {
  const directorio = new URL("./", import.meta.url);
  const [html, javascript] = await Promise.all([readFile(new URL("index.html", directorio), "utf8"), readFile(new URL("acceso.js", directorio), "utf8")]);
  assert.match(html, /Pendiente de configuración institucional/);
  assert.match(html, /Certificado digital o DNIe/);
  assert.doesNotMatch(html, /<input/iu);
  assert.doesNotMatch(javascript, /localStorage|sessionStorage|document\.cookie|Authorization/iu);
  assert.match(javascript, /AbortController/);
  assert.match(javascript, /cache: "no-store"/);
});
