import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteRemotoCronosHTTP, ErrorClienteRemotoCronos,
  RUTA_DISPONIBILIDAD_REMOTA, RUTA_MARCAJE_REMOTO, RUTA_RECUPERACION_REMOTA } from "./cliente-remoto-http.js";

const referenciaOperacion = "123e4567-e89b-42d3-a456-426614174000";
const permitido = { autorizado: true, continuidad_confirmada: true,
  movimientos_permitidos: ["entrada", "salida", "inicio_pausa", "fin_pausa"],
  periodo: { desde: "2026-09-24T00:00:00Z", hasta: "2026-10-01T00:00:00Z" }, motivo: "autorizado" };
const recibo = { recibo: { referencia: "recibo:cronos:1", instante_utc: "2026-09-24T08:12:34.123456Z",
  marcaje_original_ref: "marcaje:1", replay: false } };
const json = (valor, estado = 200) => new Response(JSON.stringify(valor), { status: estado,
  headers: { "Content-Type": "application/json; charset=utf-8" } });

test("consulta la disponibilidad exacta y no envía identidad ni ubicación", async () => {
  const llamadas = [];
  const cliente = crearClienteRemotoCronosHTTP({ fetchImpl: async (ruta, opciones) => {
    llamadas.push([ruta, opciones]); return json(permitido);
  } });
  assert.deepEqual(await cliente.disponibilidad(), permitido);
  assert.equal(llamadas[0][0], RUTA_DISPONIBILIDAD_REMOTA);
  assert.equal(llamadas[0][1].method, "GET");
  assert.equal(llamadas[0][1].body, undefined);
  assert.equal(llamadas[0][1].cache, "no-store");
  assert.equal(llamadas[0][1].redirect, "error");
  assert.equal(llamadas[0][1].referrerPolicy, "no-referrer");
  assert.equal(llamadas[0][1].credentials, "omit");
  assert.deepEqual(Object.keys(llamadas[0][1].headers), ["Accept"]);
  assert.equal(llamadas[0][1].headers.Cookie, undefined);
});

test("deniega disponibilidad incoherente y reconoce motivo controlado", async () => {
  const denegada = { autorizado: false, continuidad_confirmada: false, movimientos_permitidos: [], motivo: "teletrabajo_no_autorizado" };
  const cliente = crearClienteRemotoCronosHTTP({ fetchImpl: async () => json(denegada) });
  assert.deepEqual(await cliente.disponibilidad(), denegada);
  const mal = crearClienteRemotoCronosHTTP({ fetchImpl: async () => json({ ...denegada, motivo: "<script>" }) });
  await assert.rejects(() => mal.disponibilidad(), ErrorClienteRemotoCronos);
  const intervaloVacio = crearClienteRemotoCronosHTTP({ fetchImpl: async () => json({ autorizado: true,
    continuidad_confirmada: true, movimientos_permitidos: ["entrada"],
    periodo: { desde: "2026-10-01T00:00:00Z", hasta: "2026-10-01T00:00:00Z" }, motivo: "autorizado" }) });
  await assert.rejects(() => intervaloVacio.disponibilidad(), ErrorClienteRemotoCronos);
  const sinContinuidad = crearClienteRemotoCronosHTTP({ fetchImpl: async () => json({ autorizado: true,
    movimientos_permitidos: ["entrada"], periodo: permitido.periodo, motivo: "autorizado" }) });
  await assert.rejects(() => sinContinuidad.disponibilidad(), (error) => error.codigo === "continuidad_no_confirmada");
  const pendiente = { autorizado: true, continuidad_confirmada: false,
    movimientos_permitidos: [], periodo: permitido.periodo, motivo: "continuidad_no_confirmada" };
  assert.deepEqual(await crearClienteRemotoCronosHTTP({ fetchImpl: async () => json(pendiente) }).disponibilidad(), pendiente);
  const sinLista = crearClienteRemotoCronosHTTP({ fetchImpl: async () => json({ ...permitido, movimientos_permitidos: undefined }) });
  await assert.rejects(() => sinLista.disponibilidad(), (error) => error.codigo === "secuencia_no_permitida");
  const listaVacia = { ...permitido, movimientos_permitidos: [], motivo: "secuencia_no_permitida" };
  assert.deepEqual(await crearClienteRemotoCronosHTTP({ fetchImpl: async () => json(listaVacia) }).disponibilidad(), listaVacia);
  const duplicada = crearClienteRemotoCronosHTTP({ fetchImpl: async () => json({ ...permitido, movimientos_permitidos: ["entrada", "entrada"] }) });
  await assert.rejects(() => duplicada.disponibilidad(), ErrorClienteRemotoCronos);
});

test("registra con dos únicos campos y acepta sólo el recibo del servidor", async () => {
  let llamada;
  const cliente = crearClienteRemotoCronosHTTP({ fetchImpl: async (ruta, opciones) => {
    llamada = [ruta, opciones]; return json(recibo);
  } });
  assert.deepEqual(await cliente.registrar({ movimiento: "entrada", clave_operacion: referenciaOperacion }), recibo.recibo);
  assert.equal(llamada[0], RUTA_MARCAJE_REMOTO);
  assert.deepEqual(JSON.parse(llamada[1].body), { movimiento: "entrada", clave_operacion: referenciaOperacion });
  assert.equal(llamada[1].credentials, "omit");
  assert.equal(llamada[1].headers.Cookie, undefined);
  await assert.rejects(() => cliente.registrar({ movimiento: "entrada", clave_operacion: referenciaOperacion, latitud: 1 }), TypeError);
  const falso = crearClienteRemotoCronosHTTP({ fetchImpl: async () => json({ recibo: { ...recibo.recibo, replay: "false" } }) });
  await assert.rejects(() => falso.registrar({ movimiento: "entrada", clave_operacion: referenciaOperacion }), (error) => error.incierto);
});

test("fallo de red tras POST queda incierto para reintentar con la misma clave", async () => {
  const cliente = crearClienteRemotoCronosHTTP({ fetchImpl: async () => { throw new Error("red"); } });
  await assert.rejects(() => cliente.registrar({ movimiento: "salida", clave_operacion: referenciaOperacion }),
    (error) => error instanceof ErrorClienteRemotoCronos && error.codigo === "red_no_disponible" && error.incierto);
});

test("401, 403 central, 403 nominal y 503 conservan causas distintas", async () => {
  for (const [estado, codigoServidor, codigoCliente, incierto] of [
    [401, "autenticacion_requerida", "autenticacion_requerida", false],
    [403, "acceso_denegado", "acceso_denegado", false],
    [403, "teletrabajo_no_autorizado", "teletrabajo_no_autorizado", false],
    [503, "no_disponible", "servicio_no_disponible", true],
    [503, "continuidad_no_confirmada", "continuidad_no_confirmada", false],
    [409, "secuencia_no_permitida", "secuencia_no_permitida", false],
  ]) {
    const cliente = crearClienteRemotoCronosHTTP({ fetchImpl: async () => json({ error: codigoServidor }, estado) });
    await assert.rejects(() => cliente.registrar({ movimiento: "entrada", clave_operacion: referenciaOperacion }),
      (error) => error.codigo === codigoCliente && error.estado === estado && error.incierto === incierto);
  }
  const falso403 = crearClienteRemotoCronosHTTP({ fetchImpl: async () => json({ error: "texto_ajeno" }, 403) });
  await assert.rejects(() => falso403.disponibilidad(), (error) => error.codigo === "acceso_denegado");
});

test("recuperación nominal usa GET sin cuerpo, cookies ni clave en URL", async () => {
  let llamada;
  const cliente = crearClienteRemotoCronosHTTP({ fetchImpl: async (ruta, opciones) => {
    llamada = [ruta, opciones]; return json({ recibo: { ...recibo.recibo, replay: true } });
  } });
  assert.deepEqual(await cliente.recuperar({ movimiento: "entrada", clave_operacion: referenciaOperacion }), { ...recibo.recibo, replay: true });
  assert.equal(llamada[0], RUTA_RECUPERACION_REMOTA);
  assert.equal(llamada[1].method, "GET");
  assert.equal(llamada[1].body, undefined);
  assert.equal(llamada[1].credentials, "omit");
  assert.deepEqual(llamada[1].headers, { Accept: "application/json",
    "X-Cronos-Clave-Operacion": referenciaOperacion, "X-Cronos-Movimiento": "entrada" });
});

test("solo 404 nominal fuerte acredita ausencia; 404 ajeno y 503 no habilitan reenvío", async () => {
  for (const [estado, codigo, esperado] of [
    [404, "ausencia_confirmada", "ausencia_confirmada"],
    [404, "no_disponible", "respuesta_rechazada"],
    [503, "no_disponible", "servicio_no_disponible"],
  ]) {
    const cliente = crearClienteRemotoCronosHTTP({ fetchImpl: async () => json({ error: codigo }, estado) });
    await assert.rejects(() => cliente.recuperar({ movimiento: "entrada", clave_operacion: referenciaOperacion }),
      (error) => error.codigo === esperado && error.estado === estado);
  }
});
