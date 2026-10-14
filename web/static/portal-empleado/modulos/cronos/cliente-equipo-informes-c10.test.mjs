import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteEquipoInformesCronosHTTP, ErrorClienteEquipoInformesCronos } from "./cliente-equipo-informes-c10.js";

const empleado = `emp_${"a".repeat(22)}`;
const consulta = { equipo: "equipo_1", vista: "presencia", desde: "2026-10-01", hasta: "2026-10-02" };
const resultado = () => ({ equipo_ref: "equipo_1", alcance: "jefatura", desde: "2026-10-01", hasta: "2026-10-02", vista: "presencia", version_ref: "version_1",
  filas: [{ empleado_ref: empleado, empleado_etiqueta: "Ana Ruiz", fecha: "2026-10-01", estado: "indeterminado", causa: "sin_marcajes", cobertura: "complete", hechos_registrados: 0 }],
  agregado: { total: 1, indeterminado: 1, registrado: 0, incidencias: 0 }, siguiente_cursor: "" });

test("consulta de equipo valida rol, etiquetas y recuentos de la página", async () => {
  const cliente = crearClienteEquipoInformesCronosHTTP({ fetchImpl: async (url, opciones) => {
    assert.equal(opciones.method, "GET"); assert.match(url, /desde=2026-10-01/u);
    assert.doesNotMatch(url, /alcance=/u);
    return new Response(JSON.stringify(resultado()), { headers: { "Content-Type": "application/json; charset=utf-8" } });
  } });
  assert.equal((await cliente.consultar(consulta)).filas[0].empleado_etiqueta, "Ana Ruiz");
});

test("consulta rechaza un recuento que no abre la misma lista", async () => {
  const mal = resultado(); mal.agregado.indeterminado = 0;
  const cliente = crearClienteEquipoInformesCronosHTTP({ fetchImpl: async () => new Response(JSON.stringify(mal), { headers: { "Content-Type": "application/json" } }) });
  await assert.rejects(cliente.consultar(consulta), (e) => e instanceof ErrorClienteEquipoInformesCronos && e.codigo === "respuesta_incompatible");
});

test("oferta de informes viene de política del servidor y conserva el ámbito", async () => {
  const cliente = crearClienteEquipoInformesCronosHTTP({ idiomaActual: () => "en", fetchImpl: async (url, opciones) => {
    assert.match(url, /\/equipos\/equipo_1\/informes\/oferta$/u);
    assert.equal(opciones.headers["Accept-Language"], "en");
    return new Response(JSON.stringify({ equipo_ref: "equipo_1", alcance: "jefatura", exportaciones: ["saldo.pdf"] }),
      { headers: { "Content-Type": "application/json" } });
  } });
  assert.deepEqual((await cliente.oferta({ equipo: "equipo_1" })).exportaciones, ["saldo.pdf"]);
});

test("exportación POST envía la clave en JSON y conserva los bytes y recibo confirmados", async () => {
  const recibo = `informe:recibo:${"a".repeat(8)}-${"b".repeat(4)}-${"c".repeat(4)}-${"d".repeat(4)}-${"e".repeat(12)}`;
  const original = new TextEncoder().encode("%PDF-1.4\n%%EOF");
  const cliente = crearClienteEquipoInformesCronosHTTP({ idiomaActual: () => "en", fetchImpl: async (url, opciones) => {
    assert.equal(opciones.method, "POST"); assert.doesNotMatch(url, /clave_operacion/u);
    assert.equal(JSON.parse(opciones.body).clave_operacion, "operacion_123");
    assert.equal(opciones.headers["Accept-Language"], "en");
    return new Response(original, { headers: { "Content-Type": "application/pdf", "X-VEC-Recibo-Ref": recibo } });
  } });
  const r = await cliente.exportar({ equipo: "equipo_1", tipo: "saldo", formato: "pdf", desde: "2026-10-01", hasta: "2026-10-02", clave_operacion: "operacion_123" });
  assert.deepEqual(r.bytes, original); assert.equal(r.recibo, recibo);
});
