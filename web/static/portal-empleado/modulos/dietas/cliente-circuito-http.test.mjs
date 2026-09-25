import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteCircuitoDietasHTTP, ErrorClienteCircuitoDietas } from "./cliente-circuito-http.js";

const sufijo = "1234567890123456789012";
const referencia = `dco_${sufijo}`;
const recibo = { referencia: `rcd_${sufijo}`, version: 4, registrado_en: "2026-09-24T10:00:00.000000Z", repeticion: false };
const item = { referencia, estado: "enviado_pendiente_revision", version: 3, fecha_inicio: "2026-09-20", fecha_fin: "2026-09-21" };
const respuesta = (cuerpo, estado = 200) => new Response(JSON.stringify(cuerpo), { status: estado, headers: { "content-type": "application/json; charset=utf-8" } });

test("consulta la bandeja D8 con parámetros cerrados y transporte same-origin", async () => {
  const llamadas = []; const cliente = crearClienteCircuitoDietasHTTP({ fetchImpl: async (ruta, opciones) => { llamadas.push({ ruta, opciones }); return respuesta({ items: [item], siguiente_cursor: referencia, competencia: "acreditada" }); } });
  const pagina = await cliente.listar({ etapa: "revision", fecha_desde: "2026-09-01", fecha_hasta: "2026-09-30", limit: 20 });
  assert.equal(pagina.items[0].referencia, referencia); assert.equal(pagina.competencia, "acreditada");
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

const documento = {
  referencia, numero_documento: "VEC-D-2026-000012", fecha_apertura: "2026-09-20T08:00:00.000000Z", estado: "pendiente_autorizacion", version: 3,
  fecha_inicio: "2026-09-20", fecha_fin: "2026-09-20", hora_inicio: "08:00", hora_fin: "15:30", motivo: "Reunión técnica", codigos_ruta: [],
  calculo: {}, documento: { lineas: [{ tipo: "otro_gasto", concepto: "Aparcamiento", importe_centimos: 650, justificante_ref: "", justificante_sha256: "" }], total_orientativo_centimos: 650 },
};

test("sin fuente de competencia la bandeja llega vacía y las competencias no acreditan etapas", async () => {
  const rutas = []; const cliente = crearClienteCircuitoDietasHTTP({ fetchImpl: async (ruta) => { rutas.push(ruta); return ruta.endsWith("/competencias") ? respuesta({ fuente: "sin_fuente", etapas: [] }) : respuesta({ items: [], competencia: "sin_fuente" }); } });
  assert.deepEqual(await cliente.competencias(), { fuente: "sin_fuente", etapas: [] });
  assert.equal((await cliente.listar({ etapa: "revision" })).competencia, "sin_fuente");
  assert.equal(rutas[0], "/api/vec/dietas/comisiones/circuito/competencias");
  const incoherente = crearClienteCircuitoDietasHTTP({ fetchImpl: async () => respuesta({ fuente: "sin_fuente", etapas: ["revision"] }) });
  await assert.rejects(() => incoherente.competencias(), (error) => error.codigo === "respuesta_incompatible");
  const conFilas = crearClienteCircuitoDietasHTTP({ fetchImpl: async () => respuesta({ items: [item], competencia: "sin_fuente" }) });
  await assert.rejects(() => conFilas.listar({ etapa: "revision" }), (error) => error.codigo === "respuesta_incompatible");
  const denegado = crearClienteCircuitoDietasHTTP({ fetchImpl: async () => respuesta({ error: "dietas.error.competencia_sin_fuente" }, 403) });
  await assert.rejects(() => denegado.decidir(referencia, { etapa: "revision", decision: "aprobar", motivo: "", clave_idempotencia: "decision-circuito-0001", version_esperada: 3 }), (error) => error.codigo === "competencia_sin_fuente" && !error.resultadoIndeterminado);
});

test("lee el documento de la etapa exacta y rechaza otro estado o campos ajenos", async () => {
  const rutas = []; const cliente = crearClienteCircuitoDietasHTTP({ fetchImpl: async (ruta) => { rutas.push(ruta); return respuesta({ comision: documento }); } });
  const leido = await cliente.documento(referencia, "autorizacion");
  assert.equal(leido.numero_documento, "VEC-D-2026-000012");
  assert.equal(rutas[0], `/api/vec/dietas/comisiones/circuito/${referencia}?etapa=autorizacion`);
  await assert.rejects(() => cliente.documento(referencia, "revision"), (error) => error.codigo === "respuesta_incompatible");
  const ajeno = crearClienteCircuitoDietasHTTP({ fetchImpl: async () => respuesta({ comision: { ...documento, relacion_ref: `rel_${sufijo}` } }) });
  await assert.rejects(() => ajeno.documento(referencia, "autorizacion"), (error) => error.codigo === "respuesta_incompatible");
  await assert.rejects(() => cliente.documento(referencia, "otra"), TypeError);
});
