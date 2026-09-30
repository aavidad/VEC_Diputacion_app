import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { crearClienteAdministracion, ErrorAdministracionPerfiles, nuevaOperacionRef } from "./cliente.js";
import { cargarTextosAdministracion, estadoError, validarCapacidades, validarConfirmacion } from "./administracion-perfiles.js";

const textosES = JSON.parse(await readFile(new URL("../textos/es/administracion-perfiles.json", import.meta.url), "utf8")).general;
const textosEN = JSON.parse(await readFile(new URL("../textos/en/administracion-perfiles.json", import.meta.url), "utf8")).general;

test("catálogos de la pantalla completos y sin claves vacías", () => {
  assert.deepEqual(Object.keys(textosES).sort(), Object.keys(textosEN).sort());
  for (const [clave, valor] of Object.entries(textosES)) {
    assert.ok(valor.trim(), clave);
    assert.ok(textosEN[clave].trim(), clave);
  }
  const html = new URL("./index.html", import.meta.url);
  return readFile(html, "utf8").then((fuente) => {
    for (const [, clave] of fuente.matchAll(/data-t(?:-aria)?="([a-z_]+)"/gu)) assert.ok(textosES[clave], clave);
    assert.doesNotMatch(fuente, /localStorage|sessionStorage|document\.cookie|cdn\./iu);
  });
});

test("si falla el catálogo principal carga otro idioma declarado y cambia lang", async () => {
  const ingles = JSON.parse(await readFile(new URL("../textos/en/administracion-perfiles.json", import.meta.url), "utf8"));
  const textos = await cargarTextosAdministracion({
    cargar: async () => { throw new Error("catálogo principal ausente"); },
    idiomas: [{ codigo: "en", localizacion: "en-GB" }],
    leer: async () => ingles,
  });
  assert.equal(textos.idioma, "en");
  assert.equal(textos.traducir("general.titulo"), ingles.general.titulo);
});

test("capacidad desconocida o sin consulta cierra las acciones", () => {
  assert.equal(validarCapacidades({ version: "v2", acciones: ["consultar"] }), null);
  assert.equal(validarCapacidades({ version: "v1", acciones: ["consultar", "superusuario"] }), null);
  assert.equal(validarCapacidades({ version: "v1", acciones: [] }).has("consultar"), false);
  assert.equal(validarCapacidades({ version: "v1", acciones: ["consultar", "proponer"] }).has("proponer"), true);
});

test("solo una respuesta ligada a la operación confirma el acto", () => {
  const operacion = "acto_admin:" + "a".repeat(32);
  const propuesta = "propuesta_admin:" + "c".repeat(32);
  const cierre = "cierre_admin:" + "d".repeat(32);
  assert.equal(validarConfirmacion("ordinario", { recibo: { recibo_ref: "recibo_admin:" + "b".repeat(32), operacion_ref: operacion } }, operacion), true);
  assert.equal(validarConfirmacion("ordinario", { recibo: { recibo_ref: "recibo_admin:" + "b".repeat(32), operacion_ref: "otro" } }, operacion), false);
  assert.equal(validarConfirmacion("propuesta", { propuesta: { propuesta_ref: propuesta, huella_sha256: "a".repeat(64), caduca_en: "2026-10-01T10:00:00Z" } }, propuesta), true);
  assert.equal(validarConfirmacion("propuesta", { propuesta: { propuesta_ref: "otro", huella_sha256: "a".repeat(64) } }, operacion), false);
  assert.equal(validarConfirmacion("cierre", { cierre: { operacion_ref: cierre, propuesta_ref: propuesta, decision: "aprobada", huella_cierre_sha256: "a".repeat(64), confirmado_en: "2026-09-30T10:00:00Z", recibo: { recibo_ref: "recibo_admin:" + "b".repeat(32) } } }, cierre, propuesta, "aprobada"), true);
  assert.equal(validarConfirmacion("cierre", { cierre: { operacion_ref: cierre, propuesta_ref: propuesta, decision: "aprobada", huella_cierre_sha256: "a".repeat(64), confirmado_en: "2026-09-30T10:00:00Z" } }, cierre, propuesta, "aprobada"), false);
  assert.equal(validarConfirmacion("cierre", { cierre: { operacion_ref: cierre, propuesta_ref: propuesta, decision: "rechazada", huella_cierre_sha256: "a".repeat(64), confirmado_en: "2026-09-30T10:00:00Z" } }, cierre, propuesta, "aprobada"), false);
  assert.equal(validarConfirmacion("cierre", { cierre: { operacion_ref: operacion, propuesta_ref: "otra", huella_cierre_sha256: "a", confirmado_en: "hoy" } }, operacion, "propuesta_admin:" + "b".repeat(32)), false);
});

test("cliente usa el mismo origen y envía solo motivo y preimagen de la ficha", async () => {
  const recibidas = [];
  const fetchImpl = async (url, opciones) => {
    recibidas.push({ url, opciones });
    return new Response(JSON.stringify({ recibo: { recibo_ref: "recibo_admin:" + "b".repeat(32), operacion_ref: "acto_admin:" + "a".repeat(32) } }),
      { status: 200, headers: { "Content-Type": "application/json" } });
  };
  const cliente = crearClienteAdministracion({ fetchImpl, origen: "https://admin.example.test" });
  const cuerpo = { operacion_ref: "acto_admin:" + "a".repeat(32), motivo: { entrada_clave: "motivo_1" }, objetivo: { persona_ref: "per_" + "c".repeat(22) } };
  await cliente.aplicar(cuerpo);
  assert.equal(recibidas.length, 1);
  assert.equal(recibidas[0].url, "https://admin.example.test/api/admin/perfiles/v1/actos-ordinarios");
  assert.equal(recibidas[0].opciones.credentials, "same-origin");
  assert.equal(recibidas[0].opciones.redirect, "error");
  assert.deepEqual(JSON.parse(recibidas[0].opciones.body), cuerpo);
  assert.equal(recibidas[0].opciones.headers.Authorization, undefined);
});

test("errores 403, 409 y 503 conservan mensajes distintos y no parecen éxito", async () => {
  for (const [estado, codigo, clave] of [[403, "acceso_denegado", "error_denegado"], [409, "conflicto_estado", "error_conflicto"], [503, "servicio_no_disponible", "error_servicio"]]) {
    const cliente = crearClienteAdministracion({ origen: "https://admin.example.test", fetchImpl: async () =>
      new Response(JSON.stringify({ error: { codigo, clave_i18n: `api.admin.perfiles.error.${codigo}` } }), { status: estado, headers: { "Content-Type": "application/json" } }) });
    await assert.rejects(cliente.capacidades(), (error) => error instanceof ErrorAdministracionPerfiles && error.estado === estado && estadoError(error) === clave);
  }
});

test("la clave de operación usa entropía local y formato del dominio", () => {
  const ref = nuevaOperacionRef("propuesta_admin:", { getRandomValues: (bytes) => { bytes.fill(0xab); return bytes; } });
  assert.equal(ref, "propuesta_admin:" + "ab".repeat(16));
  assert.throws(() => nuevaOperacionRef("perfil_admin:"));
});
