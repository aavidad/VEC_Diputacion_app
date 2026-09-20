import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { crearClienteMiBolsa, ErrorMiBolsa, renderizarMiBolsa, validarMiBolsa } from "./mi-bolsa.js";

const participacion = Object.freeze({ bolsa_ref: "bolsa:auxiliares-administrativos:2026", categoria_ref: "categoria:auxiliar-administrativo", version_bolsa: 2, orden: 3, total_participaciones: 42, estado_bolsa: "Vigente", vigente_desde: "2026-09-01", vigente_hasta: null });
const datos = () => ({ esquema: "vec.bolsa.mi-bolsa.v1", consultada_en: "2026-09-20T10:00:00Z", participaciones: [participacion] });
function respuestaJSON(data, status = 200) { const texto = JSON.stringify(data); return { status, headers: { get: (nombre) => ({ "Content-Type": "application/json; charset=utf-8", "Content-Length": String(new TextEncoder().encode(texto).byteLength) })[nombre] ?? null }, text: async () => texto }; }

test("mi bolsa usa GET sin cuerpo, query ni credenciales", async () => {
  let peticion;
  const cliente = crearClienteMiBolsa({ fetchImpl: async (ruta, opciones) => { peticion = { ruta, opciones }; return respuestaJSON({ data: datos() }); } });
  assert.equal((await cliente.cargar()).participaciones[0].orden, 3);
  assert.equal(peticion.ruta, "/api/vec/bolsa/mi-bolsa");
  assert.equal(peticion.opciones.method, "GET");
  assert.equal(peticion.opciones.body, undefined);
  assert.equal(peticion.opciones.credentials, "omit");
  assert.equal(peticion.opciones.headers.Authorization, undefined);
});

test("mi bolsa valida el contrato y solo representa orden histórico y vigencia", () => {
  assert.equal(validarMiBolsa(datos()).esquema, "vec.bolsa.mi-bolsa.v1");
  assert.throws(() => validarMiBolsa({ ...datos(), esquema: "otro" }), /esquema/u);
  assert.throws(() => validarMiBolsa({ ...datos(), participaciones: [{ ...participacion, bolsa_ref: undefined }] }), /bolsa_ref/u);
  assert.throws(() => validarMiBolsa({ ...datos(), participaciones: [{ ...participacion, total_participaciones: 2 }] }), /total_participaciones/u);
  const html = renderizarMiBolsa(validarMiBolsa(datos()));
  assert.match(html, /Orden en la constitución/u);
  assert.match(html, /no indica la posición actual/u);
  assert.doesNotMatch(html, /Puntuación|Último llamamiento|Situación actual|Pausar|Reactivar/u);
});

test("mi bolsa acepta precisión PostgreSQL en UTC y fecha final omitida", () => {
  const entrada = datos();
  entrada.consultada_en = "2026-09-20T10:00:00.123456Z";
  entrada.participaciones = [{ ...participacion }];
  delete entrada.participaciones[0].vigente_hasta;
  assert.equal(validarMiBolsa(entrada).consultada_en, entrada.consultada_en);
});

for (const [estado, codigo] of [[401, "autenticacion_requerida"], [403, "acceso_denegado"], [503, "servicio_no_disponible"]]) {
  test(`mi bolsa informa HTTP ${estado} sin datos aparentes`, async () => {
    const cliente = crearClienteMiBolsa({ fetchImpl: async () => ({ status: estado, headers: { get: () => null }, text: async () => "" }) });
    await assert.rejects(() => cliente.cargar(), (error) => error instanceof ErrorMiBolsa && error.codigo === codigo);
  });
}

test("mi bolsa está enlazada y enumerada en la superficie productiva", async () => {
  const [html, manifiesto] = await Promise.all([
    readFile(new URL("./index.html", import.meta.url), "utf8"),
    readFile(new URL("../../produccion.manifest", import.meta.url), "utf8"),
  ]);
  assert.match(html, /data-ruta="mi-bolsa"/u);
  assert.match(manifiesto, /^static\/area-personal\/mi-bolsa\.js$/mu);
  assert.match(manifiesto, /^static\/area-personal\/i18n\.js$/mu);
  assert.match(manifiesto, /^static\/area-personal\/locales\/es\.json$/mu);
});
