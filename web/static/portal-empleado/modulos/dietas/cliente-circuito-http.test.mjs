import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteCircuitoDietasHTTP, ErrorClienteCircuitoDietas } from "./cliente-circuito-http.js";

const sufijo = "1234567890123456789012";
const referencia = `dco_${sufijo}`;
const recibo = { referencia: `rcd_${sufijo}`, version: 4, registrado_en: "2026-09-24T10:00:00.000000Z", repeticion: false };
const item = { referencia, estado: "enviado_pendiente_revision", version: 3, fecha_inicio: "2026-09-20", fecha_fin: "2026-09-21" };
const respuesta = (cuerpo, estado = 200) => new Response(JSON.stringify(cuerpo), { status: estado, headers: { "content-type": "application/json; charset=utf-8" } });

test("consulta la bandeja D8 con parámetros cerrados y transporte same-origin", async () => {
  const llamadas = []; const cliente = crearClienteCircuitoDietasHTTP({ fetchImpl: async (ruta, opciones) => { llamadas.push({ ruta, opciones }); return respuesta({ items: [item], siguiente_cursor: referencia }); } });
  const pagina = await cliente.listar({ etapa: "revision", fecha_desde: "2026-09-01", fecha_hasta: "2026-09-30", limit: 20 });
  assert.equal(pagina.items[0].referencia, referencia);
  assert.equal(llamadas[0].ruta, "/api/vec/dietas/comisiones/circuito?etapa=revision&limit=20&fecha_desde=2026-09-01&fecha_hasta=2026-09-30");
  assert.equal(llamadas[0].opciones.credentials, "same-origin"); assert.equal(llamadas[0].opciones.mode, "same-origin");
  await assert.rejects(() => cliente.listar({ etapa: "revision", unidad_ref: "uni_ajena" }), /consulta/u);
});

test("registra una decisión sin identidad ni unidad libres y conserva el recibo", async () => {
  const llamadas = []; const cliente = crearClienteCircuitoDietasHTTP({ fetchImpl: async (ruta, opciones) => { llamadas.push({ ruta, opciones }); return respuesta({ comision: { referencia, estado: "pendiente_autorizacion", version: 4 }, recibo }, 201); } });
  const entrada = { etapa: "revision", decision: "aprobar", motivo: "", clave_idempotencia: "decision-circuito-0001", version_esperada: 3 };
  const resultado = await cliente.decidir(referencia, entrada);
  assert.equal(resultado.recibo.referencia, recibo.referencia); assert.equal(llamadas[0].ruta, `/api/vec/dietas/comisiones/circuito/${referencia}/decisiones`);
  assert.deepEqual(JSON.parse(llamadas[0].opciones.body), entrada);
  await assert.rejects(() => cliente.decidir(referencia, { ...entrada, actor_ref: "per_ajena" }), /decisión/u);
});

test("exige motivo de devolución y distingue conflicto, denegación e incertidumbre", async () => {
  const cliente = crearClienteCircuitoDietasHTTP({ fetchImpl: async () => respuesta({}, 200) });
  await assert.rejects(() => cliente.decidir(referencia, { etapa: "revision", decision: "devolver", motivo: "no", clave_idempotencia: "decision-circuito-0001", version_esperada: 3 }), /decisión/u);
  for (const [estado, codigo, incierta] of [[403, "acceso_denegado", false], [409, "conflicto_estado", false], [503, "resultado_incierto", true]]) {
    const rechazado = crearClienteCircuitoDietasHTTP({ fetchImpl: async () => respuesta({ error: `dietas.error.${codigo}` }, estado) });
    await assert.rejects(() => rechazado.decidir(referencia, { etapa: "revision", decision: "aprobar", motivo: "", clave_idempotencia: "decision-circuito-0001", version_esperada: 3 }), (error) => error instanceof ErrorClienteCircuitoDietas && error.codigo === codigo && error.resultadoIndeterminado === incierta);
  }
});

test("propaga cancelación y nunca inicia un POST con una señal ya abortada", async () => {
  const controlador = new AbortController(); controlador.abort(); const cliente = crearClienteCircuitoDietasHTTP({ fetchImpl: async () => assert.fail("fetch inesperado") });
  await assert.rejects(() => cliente.listar({ etapa: "revision" }, { signal: controlador.signal }), (error) => error.codigo === "operacion_abortada");
});
