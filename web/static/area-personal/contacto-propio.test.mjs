import assert from "node:assert/strict";
import test from "node:test";

import { capturarCorreoEnviado, crearControladorContactoPropio, RUTA_CONTACTO_PROPIO, RUTA_RECIBO_CONTACTO_PROPIO } from "./contacto-propio.js";
import { conservarResultadoContactoPropio } from "./aplicacion.js";

function respuesta(estado, cuerpo = {}) {
  return { status: estado, json: async () => cuerpo };
}

test("guarda exclusivamente correo con la versión autorizada", async () => {
  let peticion;
  const controlador = crearControladorContactoPropio({
    autorizacionServidor: { capacidad: true, version: 7 },
    fetchImpl: async (...argumentos) => {
      peticion = argumentos;
      return respuesta(201, { recibo_ref: "recibo-opaco", version: 8 });
    },
  });
  const resultado = await controlador.guardar(" persona@ejemplo.test ");
  assert.equal(peticion[0], RUTA_CONTACTO_PROPIO);
  assert.deepEqual(JSON.parse(peticion[1].body), { correo: "persona@ejemplo.test", version_esperada: 7 });
  assert.equal(peticion[1].credentials, "omit");
  assert.equal(peticion[1].cache, "no-store");
  assert.deepEqual(resultado, { reciboRef: "recibo-opaco", version: 8 });
});

test("avanza la versión desde el recibo para un segundo guardado", async () => {
  const cuerpos = [];
  const controlador = crearControladorContactoPropio({
    autorizacionServidor: { capacidad: true, version: 7 },
    fetchImpl: async (_ruta, opciones) => {
      cuerpos.push(JSON.parse(opciones.body));
      return respuesta(201, { recibo_ref: `recibo-${cuerpos.length}`, version: 7 + cuerpos.length });
    },
  });
  await controlador.guardar("primero@ejemplo.test");
  await controlador.guardar("segundo@ejemplo.test");
  assert.deepEqual(cuerpos.map((cuerpo) => cuerpo.version_esperada), [7, 8]);
});

test("rechaza una versión de respuesta que no sea la sucesora exacta", async () => {
  const controlador = crearControladorContactoPropio({
    autorizacionServidor: { capacidad: true, version: 7 },
    fetchImpl: async () => respuesta(201, { recibo_ref: "recibo-opaco", version: 10 }),
  });
  await assert.rejects(() => controlador.guardar("persona@ejemplo.test"), /No se pudo confirmar/iu);
});

test("conserva recibo y versión al reconstruir la pantalla en memoria", () => {
  const estado = {
    contactoPropio: { capacidad: true, version: 7 },
    contactoPropioRecibo: null,
    datos: { perfil: { correo: "anterior@ejemplo.test" } },
  };
  conservarResultadoContactoPropio(estado, {
    reciboRef: "recibo-opaco", version: 8, correo: "nuevo@ejemplo.test",
  });
  assert.deepEqual(estado.contactoPropio, { capacidad: true, version: 8 });
  assert.deepEqual(estado.contactoPropioRecibo, { reciboRef: "recibo-opaco", version: 8 });
  assert.equal(estado.datos.perfil.correo, "nuevo@ejemplo.test");
});

test("captura el correo antes de que la interfaz pueda cambiar durante el envío", () => {
  const entrada = { value: "correo-a@ejemplo.test" };
  const correoEnviado = capturarCorreoEnviado(entrada);
  entrada.value = "correo-b@ejemplo.test";
  assert.equal(correoEnviado, "correo-a@ejemplo.test");
});

test("falla cerrada sin permiso o versión recibidos del servidor", async () => {
  for (const autorizacionServidor of [null, { capacidad: false, version: 2 }, { capacidad: true }]) {
    const controlador = crearControladorContactoPropio({ autorizacionServidor, fetchImpl: async () => assert.fail("no debe llamar a red") });
    await assert.rejects(() => controlador.guardar("persona@ejemplo.test"), /no ha aportado permiso expreso y versión vigente/iu);
  }
});

test("la presentación no puede emitir el POST aunque reciba permiso y versión", async () => {
  const controlador = crearControladorContactoPropio({
    presentacion: true,
    autorizacionServidor: { capacidad: true, version: 7 },
    fetchImpl: async () => assert.fail("la presentación no puede enviar correo"),
  });
  assert.equal(controlador.autorizado, false);
  await assert.rejects(() => controlador.guardar("persona@ejemplo.test"), /no ha aportado permiso expreso y versión vigente/iu);
});

test("muestra un error acotado si el servicio rechaza el correo", async () => {
  const controlador = crearControladorContactoPropio({
    autorizacionServidor: { capacidad: true, version: 1 },
    fetchImpl: async () => respuesta(403),
  });
  await assert.rejects(() => controlador.guardar("persona@ejemplo.test"), /No dispone de permiso/iu);
});

test("no duplica un envío mientras el primero sigue pendiente", async () => {
  let resolver;
  let llamadas = 0;
  const controlador = crearControladorContactoPropio({
    autorizacionServidor: { capacidad: true, version: 0 },
    fetchImpl: async () => {
      llamadas += 1;
      return new Promise((resolve) => { resolver = resolve; });
    },
  });
  const primero = controlador.guardar("persona@ejemplo.test");
  await assert.rejects(() => controlador.guardar("persona@ejemplo.test"), /Guardando correo/iu);
  assert.equal(llamadas, 1);
  resolver(respuesta(201, { recibo_ref: "recibo-inicial", version: 1 }));
  await primero;
});

test("recupera recibo tras respuesta perdida sin confirmar el correo ni avanzar su versión", async () => {
  const peticiones = [];
  const controlador = crearControladorContactoPropio({
    autorizacionServidor: { capacidad: true, consultarRecibo: true, version: 7 },
    fetchImpl: async (ruta, opciones) => {
      peticiones.push([ruta, opciones]);
      if (ruta === RUTA_CONTACTO_PROPIO) throw new TypeError("respuesta perdida");
      return respuesta(200, { recibo_ref: "recibo-original", version: 8 });
    },
  });
  assert.equal(controlador.puedeConsultarRecibo, false);
  await assert.rejects(() => controlador.guardar("primero@ejemplo.test"));
  assert.equal(controlador.puedeConsultarRecibo, true);
  assert.deepEqual(await controlador.consultarRecibo(), { reciboRef: "recibo-original", version: 8 });
  assert.equal(controlador.recibo, null);
  assert.equal(peticiones[1][0], RUTA_RECIBO_CONTACTO_PROPIO);
  assert.deepEqual(JSON.parse(peticiones[1][1].body), { version: 8 });
  for (const [clave, valor] of Object.entries({ method: "POST", credentials: "omit", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer" })) {
    assert.equal(peticiones[1][1][clave], valor);
  }
  await assert.rejects(() => controlador.guardar("distinto@ejemplo.test"));
  assert.deepEqual(JSON.parse(peticiones[2][1].body), { correo: "distinto@ejemplo.test", version_esperada: 7 });
});

test("consultar exige permiso separado y un intento previo, incluso en presentación", async () => {
  for (const opciones of [
    { autorizacionServidor: { capacidad: true, version: 0 } },
    { autorizacionServidor: { capacidad: true, consultarRecibo: true, version: 0 }, presentacion: true },
    { autorizacionServidor: { capacidad: true, consultarRecibo: true, version: 0 } },
  ]) {
    const controlador = crearControladorContactoPropio({ ...opciones, fetchImpl: async () => assert.fail("sin red") });
    await assert.rejects(() => controlador.consultarRecibo(), /no está disponible/iu);
  }
  let llamadas = 0;
  const controlador = crearControladorContactoPropio({
    autorizacionServidor: { capacidad: true, version: 0 },
    fetchImpl: async () => { llamadas++; throw new Error(); },
  });
  await assert.rejects(() => controlador.guardar("persona@ejemplo.test"));
  await assert.rejects(() => controlador.consultarRecibo(), /no está disponible/iu);
  assert.equal(llamadas, 1);
});

test("una consulta sin recibo o con otra versión conserva el resultado incierto", async () => {
  for (const resultado of [respuesta(404), respuesta(403), respuesta(200, { recibo_ref: "ajeno", version: 9 })]) {
    const controlador = crearControladorContactoPropio({
      autorizacionServidor: { capacidad: true, consultarRecibo: true, version: 7 },
      fetchImpl: async (ruta) => {
        if (ruta === RUTA_CONTACTO_PROPIO) throw new Error();
        return resultado;
      },
    });
    await assert.rejects(() => controlador.guardar("persona@ejemplo.test"));
    await assert.rejects(() => controlador.consultarRecibo(), /no lo confirma ni lo descarta/iu);
    assert.equal(controlador.recibo, null);
    assert.equal(controlador.puedeConsultarRecibo, true);
  }
});

test("una consulta pendiente impide solapar guardado y otra consulta", async () => {
  let resolver;
  let llamadas = 0;
  const controlador = crearControladorContactoPropio({
    autorizacionServidor: { capacidad: true, consultarRecibo: true, version: 7 },
    fetchImpl: async (ruta) => {
      llamadas++;
      if (ruta === RUTA_CONTACTO_PROPIO) throw new Error();
      return new Promise((resolve) => { resolver = resolve; });
    },
  });
  await assert.rejects(() => controlador.guardar("persona@ejemplo.test"));
  const consulta = controlador.consultarRecibo();
  await assert.rejects(() => controlador.guardar("otro@ejemplo.test"));
  await assert.rejects(() => controlador.consultarRecibo());
  assert.equal(llamadas, 2);
  resolver(respuesta(200, { recibo_ref: "original", version: 8 }));
  await consulta;
});
