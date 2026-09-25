import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteBorradoresDietasHTTP } from "./cliente-borradores-http.js";

const relacion = "rel_1234567890123456789012";
const referencia = "dco_1234567890123456789012";
const idempotencia = "123e4567-e89b-42d3-a456-426614174000";
const tarifa = "provisional:rd462:20260923";
const tramosGrupo = [{ fecha: "2026-09-24", tipo: "manutencion", importe_centimos: 1500 }];
const opciones = [1, 2, 3].map((grupo) => ({ grupo, calculo: {
  tramos: grupo === 2 ? tramosGrupo : [], total_maximo_orientativo_centimos: grupo === 2 ? 1500 : 0,
} }));
const base = { referencia, version: 2, numero_documento: "VEC-D-2026-000001",
  fecha_apertura: "2026-09-24T10:00:00.000000Z", estado: "borrador", fecha_inicio: "2026-09-24",
  fecha_fin: "2026-09-25", motivo: "Visita de coordinación", codigos_ruta: ["18087", "18003"], relacion_ref: relacion };
const calculoBase = { procedencia: "sin_vehiculo_propio", motor: "no_aplica", version_grafo: "no_aplica",
  version_tarifa: tarifa, rotulo: "PROVISIONAL · pendiente de confirmación por RRHH", hora_inicio: "09:00", hora_fin: "18:00",
  kilometros: "0.0000", eur_por_km: "0.2600", importe_kilometraje_centimos: 0, tramos_ruta: [], opciones_dieta: opciones };
const documento = { vehiculo_propio: false, grupo_dieta: 2, version_tarifa_aceptada: tarifa,
  tramos_aceptados: [0], lineas: [{ tipo: "dieta", grupo: 2, indice_tramo: 0, fecha: "2026-09-24",
    concepto: "manutencion", importe_centimos: 1500 }], manutencion_centimos: 1500,
  alojamiento_tope_centimos: 0, kilometraje_centimos: 0, otros_centimos: 0, total_orientativo_centimos: 1500 };
const item = { comision: { ...base, calculo: calculoBase, vehiculo_propio: false, rutas: [], documento },
  recibo: { referencia: "rcd_1234567890123456789012", version: 2,
    registrado_en: "2026-09-24T10:01:00.000000Z", repeticion: false } };
const solicitud = { clave_idempotencia: idempotencia, version_esperada: 1, relacion_ref: relacion,
  fecha_inicio: base.fecha_inicio, fecha_fin: base.fecha_fin, hora_inicio: "09:00", hora_fin: "18:00",
  motivo: base.motivo, codigos_ruta: base.codigos_ruta, vehiculo_propio: false, rutas: [], otros: [],
  tramos_aceptados: [0], version_tarifa_aceptada: tarifa };
const respuesta = (cuerpo, estado = 201) => new Response(JSON.stringify(cuerpo), {
  status: estado, headers: { "Content-Type": "application/json; charset=utf-8" },
});

test("PUT guarda un solo grupo acreditado y un total orientativo singular con recibo", async () => {
  const llamadas = [];
  const cliente = crearClienteBorradoresDietasHTTP({ fetchImpl: async (ruta, opciones) => {
    llamadas.push({ ruta, opciones }); return respuesta(item);
  } });
  const resultado = await cliente.editar(referencia, solicitud);
  assert.equal(resultado.comision.documento.grupo_dieta, 2);
  assert.equal(resultado.comision.documento.total_orientativo_centimos, 1500);
  assert.equal(llamadas[0].ruta, `/api/vec/dietas/comisiones/${referencia}`);
  assert.equal(llamadas[0].opciones.method, "PUT");
  assert.equal(llamadas[0].opciones.credentials, "same-origin");
  assert.deepEqual(JSON.parse(llamadas[0].opciones.body), solicitud);
  assert.equal(llamadas[0].opciones.headers.Cookie, undefined);
});

test("rechaza documento que atribuye dieta a otro grupo o altera su total", async () => {
  for (const documentoAdulterado of [
    { ...documento, grupo_dieta: 1 },
    { ...documento, total_orientativo_centimos: 3000 },
    { ...documento, lineas: [{ ...documento.lineas[0], indice_tramo: 1 }] },
  ]) {
    const cliente = crearClienteBorradoresDietasHTTP({ fetchImpl: async () => respuesta({ ...item,
      comision: { ...item.comision, documento: documentoAdulterado },
    }) });
    await assert.rejects(() => cliente.editar(referencia, solicitud),
      (error) => error.codigo === "respuesta_incompatible" && error.resultadoIndeterminado);
  }
});

test("PUT conserva ruta propia con ajuste y otro gasto sin inventar custodia", async () => {
  const ruta = { codigos_ruta: ["18087", "18003"], ajuste_kilometros: "-0.5000", motivo_ajuste: "Atajo documentado" };
  const otros = [{ tipo: "otro_gasto", concepto: "Aparcamiento", importe_centimos: 100,
    justificante_ref: "", justificante_sha256: "" }];
  const calculo = { ...calculoBase, procedencia: "osrm_interno", motor: "OSRM", version_grafo: "grafo:uno",
    vehiculo_propio: true, kilometros: "11.5000", importe_kilometraje_centimos: 299,
    rutas: [{ ...ruta, version_grafo: "grafo:uno", tramos_ruta: [{ origen_codigo: "18087", destino_codigo: "18003", kilometros: "12.0000" }],
      kilometros_base: "12.0000", kilometros_finales: "11.5000", importe_centimos: 299 }] };
  const documentoRuta = { ...documento, vehiculo_propio: true, kilometraje_centimos: 299, otros_centimos: 100,
    total_orientativo_centimos: 1899, lineas: [...documento.lineas,
      { tipo: "kilometraje", ruta_indice: 1, kilometros: "11.5000", importe_centimos: 299 },
      { ...otros[0] }] };
  const completo = { ...item, comision: { ...base, vehiculo_propio: true, rutas: [ruta], calculo, documento: documentoRuta } };
  const cliente = crearClienteBorradoresDietasHTTP({ fetchImpl: async () => respuesta(completo) });
  const guardado = await cliente.editar(referencia, { ...solicitud, vehiculo_propio: true, rutas: [ruta], otros });
  assert.equal(guardado.comision.documento.kilometraje_centimos, 299);
  assert.equal(guardado.comision.documento.otros_centimos, 100);
});

test("enviar solo confirma estado de revisión y recibo válido", async () => {
  const enviado = { ...item, comision: { ...item.comision, estado: "enviado_pendiente_revision", version: 3 },
    recibo: { ...item.recibo, version: 3 } };
  const llamadas = [];
  const cliente = crearClienteBorradoresDietasHTTP({ fetchImpl: async (ruta, opciones) => {
    llamadas.push({ ruta, opciones }); return respuesta(enviado);
  } });
  const resultado = await cliente.enviar(referencia, { clave_idempotencia: idempotencia, version_esperada: 2, relacion_ref: relacion });
  assert.equal(resultado.comision.estado, "enviado_pendiente_revision");
  assert.equal(llamadas[0].ruta, `/api/vec/dietas/comisiones/${referencia}/enviar`);
  assert.equal(llamadas[0].opciones.method, "POST");
  const invalido = crearClienteBorradoresDietasHTTP({ fetchImpl: async () => respuesta(item) });
  await assert.rejects(() => invalido.enviar(referencia, { clave_idempotencia: idempotencia, version_esperada: 2, relacion_ref: relacion }),
    (error) => error.codigo === "respuesta_incompatible" && error.resultadoIndeterminado);
});
