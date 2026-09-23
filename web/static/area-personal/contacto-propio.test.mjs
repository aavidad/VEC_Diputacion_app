import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { traducir } from "./i18n.js";
import { textoContactoPropio } from "./i18n-contacto-propio.js";

import { capturarCorreoEnviado, crearControladorContactoPropio, montarContactoPropio, RUTA_CONTACTO_PROPIO, RUTA_RECIBO_CONTACTO_PROPIO } from "./contacto-propio.js";
import { conservarOperacionContactoPropio, conservarResultadoContactoPropio, excluirCorreoDeActualizacionContacto } from "./aplicacion.js";

function respuesta(estado, cuerpo = {}) {
  return { status: estado, json: async () => cuerpo };
}

function contenedorDOM() {
  const crearNodo = () => ({ children: [], handlers: {}, append(...nodos) { this.children.push(...nodos); }, setAttribute() {}, addEventListener(tipo, fn) { this.handlers[tipo] = fn; }, removeEventListener(tipo) { delete this.handlers[tipo]; }, checkValidity() { return true; }, focus() {} });
  const documento = { createElement: crearNodo };
  return { ownerDocument: documento, children: [], replaceChildren(...nodos) { this.children = nodos; } };
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
  assert.equal(peticion[1].credentials, "same-origin");
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
  for (const [clave, valor] of Object.entries({ method: "POST", credentials: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer" })) {
    assert.equal(peticiones[1][1][clave], valor);
  }
  await assert.rejects(() => controlador.guardar("distinto@ejemplo.test"), /mismo correo/iu);
  assert.equal(peticiones.length, 2);
});

test("navegar y volver conserva intención exacta y consulta sólo su versión", async () => {
  const estado = { contactoPropio: { capacidad: true, consultarRecibo: true, version: 7 }, contactoPropioRecibo: null, datos: { perfil: { correo: "anterior@ejemplo.test" } }, contactoPropioOperacion: null };
  const peticiones = [];
  const actualizar = (operacion) => conservarOperacionContactoPropio(estado, operacion);
  const primero = crearControladorContactoPropio({ autorizacionServidor: estado.contactoPropio, alActualizarOperacion: actualizar, fetchImpl: async (ruta, opciones) => { peticiones.push([ruta, opciones]); throw new TypeError("respuesta perdida"); } });
  await assert.rejects(() => primero.guardar("fijado@ejemplo.test"));
  const vuelto = crearControladorContactoPropio({ autorizacionServidor: estado.contactoPropio, operacion: estado.contactoPropioOperacion, alActualizarOperacion: actualizar, fetchImpl: async (ruta, opciones) => { peticiones.push([ruta, opciones]); return respuesta(200, { recibo_ref: "historico", version: 8 }); } });
  assert.deepEqual(vuelto.intencion, { correo: "fijado@ejemplo.test", version_esperada: 7 });
  await assert.rejects(() => vuelto.guardar("otro@ejemplo.test"), /mismo correo/iu);
  assert.deepEqual(await vuelto.consultarRecibo(), { reciboRef: "historico", version: 8 });
  assert.deepEqual(vuelto.intencion, { correo: "fijado@ejemplo.test", version_esperada: 7 });
  assert.deepEqual(JSON.parse(peticiones[1][1].body), { version: 8 });
});

test("resultado tardío persiste en el modelo sin depender de una vista montada", async () => {
  const estado = { contactoPropio: { capacidad: true, version: 7 }, contactoPropioRecibo: null, datos: { perfil: { correo: "anterior@ejemplo.test" } }, contactoPropioOperacion: null };
  let resolver;
  const controlador = crearControladorContactoPropio({ autorizacionServidor: estado.contactoPropio, alActualizarOperacion: (operacion) => conservarOperacionContactoPropio(estado, operacion), fetchImpl: () => new Promise((resolve) => { resolver = resolve; }) });
  const guardado = controlador.guardar("tardio@ejemplo.test");
  assert.deepEqual(estado.contactoPropioOperacion.intencion, { correo: "tardio@ejemplo.test", version_esperada: 7 });
  resolver(respuesta(201, { recibo_ref: "recibo-tardio", version: 8 }));
  await guardado;
  assert.equal(estado.datos.perfil.correo, "tardio@ejemplo.test");
  assert.deepEqual(estado.contactoPropioRecibo, { reciboRef: "recibo-tardio", version: 8 });
});

test("respuesta tardía tras destruir el DOM actualiza modelo sin callback de la vista retirada", async () => {
  let resolver; let callbacksVista = 0; const operaciones = [];
  const controlador = crearControladorContactoPropio({ autorizacionServidor: { capacidad: true, version: 7 }, alActualizarOperacion: (operacion) => operaciones.push(operacion), fetchImpl: () => new Promise((resolve) => { resolver = resolve; }) });
  const contenedor = contenedorDOM();
  const montaje = montarContactoPropio({ contenedor, correo: "retirado@ejemplo.test", controlador, alGuardar: () => { callbacksVista++; } });
  const envio = contenedor.children[0].handlers.submit({ preventDefault() {} });
  montaje.destruir();
  resolver(respuesta(201, { recibo_ref: "recibo-retirado", version: 8 }));
  await envio;
  assert.equal(callbacksVista, 0);
  assert.deepEqual(operaciones.at(-1).resultado, { reciboRef: "recibo-retirado", version: 8, correo: "retirado@ejemplo.test" });
});

test("el montaje vigente recibe el 201 de un envío pendiente tras volver a la pantalla", async () => {
  let resolver; let guardadosRetirados = 0;
  const controlador = crearControladorContactoPropio({
    autorizacionServidor: { capacidad: true, consultarRecibo: true, version: 7 },
    fetchImpl: () => new Promise((resolve) => { resolver = resolve; }),
  });
  const contenedor = contenedorDOM();
  const retirado = montarContactoPropio({ contenedor, correo: "pendiente@ejemplo.test", controlador, alGuardar: () => { guardadosRetirados++; } });
  const envio = contenedor.children[0].handlers.submit({ preventDefault() {} });
  retirado.destruir();
  montarContactoPropio({ contenedor, correo: "anterior@ejemplo.test", controlador });
  const formularioVigente = contenedor.children[0];
  const entradaVigente = formularioVigente.children[0].children[1];
  const consultaVigente = formularioVigente.children[2];
  const estadoVigente = formularioVigente.children[3];
  assert.equal(entradaVigente.value, "pendiente@ejemplo.test");
  assert.equal(entradaVigente.disabled, true);
  assert.equal(consultaVigente.hidden, false);
  resolver(respuesta(201, { recibo_ref: "recibo-vigente", version: 8 }));
  await envio;
  assert.equal(guardadosRetirados, 0);
  assert.match(estadoVigente.textContent, /recibo-vigente/);
  assert.equal(entradaVigente.disabled, false);
});

test("al volver tras una respuesta perdida muestra consulta y conserva la intención tras su 200", async () => {
  const controlador = crearControladorContactoPropio({
    autorizacionServidor: { capacidad: true, consultarRecibo: true, version: 7 },
    fetchImpl: async (ruta) => ruta === RUTA_CONTACTO_PROPIO
      ? Promise.reject(new TypeError("respuesta perdida"))
      : respuesta(200, { recibo_ref: "recibo-historico", version: 8 }),
  });
  await assert.rejects(() => controlador.guardar("fijado@ejemplo.test"));
  const contenedor = contenedorDOM();
  montarContactoPropio({ contenedor, correo: "anterior@ejemplo.test", controlador });
  const formulario = contenedor.children[0];
  const entrada = formulario.children[0].children[1];
  const consultar = formulario.children[2];
  assert.equal(entrada.value, "fijado@ejemplo.test");
  assert.equal(entrada.disabled, true);
  assert.equal(consultar.hidden, false);
  await consultar.handlers.click();
  assert.deepEqual(controlador.intencion, { correo: "fijado@ejemplo.test", version_esperada: 7 });
  assert.match(formulario.children[3].textContent, /recibo-historico/);
});

test("un recibo anterior no tapa el error ni la consulta de un segundo intento incierto", async () => {
  for (const [respuestaConsulta, patronEstado] of [
    [respuesta(200, { recibo_ref: "recibo-consulta-b", version: 9 }), /recibo-consulta-b/],
    [respuesta(404), /no lo confirma ni lo descarta/iu],
  ]) {
    let guardados = 0;
    const controlador = crearControladorContactoPropio({
      autorizacionServidor: { capacidad: true, consultarRecibo: true, version: 7 },
      fetchImpl: async (ruta) => {
        if (ruta === RUTA_CONTACTO_PROPIO) {
          guardados += 1;
          return guardados === 1
            ? respuesta(201, { recibo_ref: "recibo-a", version: 8 })
            : respuesta(503);
        }
        return respuestaConsulta;
      },
    });
    const contenedor = contenedorDOM();
    montarContactoPropio({ contenedor, correo: "a@ejemplo.test", controlador });
    const formulario = contenedor.children[0];
    const entrada = formulario.children[0].children[1];
    const consultar = formulario.children[2];
    const estado = formulario.children[3];
    const historico = formulario.children[4];
    await formulario.handlers.submit({ preventDefault() {} });
    entrada.value = "b@ejemplo.test";
    await formulario.handlers.submit({ preventDefault() {} });
    assert.match(estado.textContent, /No se pudo confirmar/iu);
    assert.doesNotMatch(estado.textContent, /recibo-a/);
    assert.equal(historico.hidden, false);
    assert.match(historico.textContent, /recibo-a/);
    assert.equal(consultar.hidden, false);
    await consultar.handlers.click();
    assert.match(estado.textContent, patronEstado);
    assert.doesNotMatch(estado.textContent, /recibo-a/);
    assert.match(historico.textContent, /recibo-a/);
    assert.deepEqual(controlador.intencion, { correo: "b@ejemplo.test", version_esperada: 8 });
  }
});

test("la actualización de teléfono y domicilio excluye correo del payload", () => {
  assert.deepEqual(excluirCorreoDeActualizacionContacto({ correo: "oculto@ejemplo.test", telefono: "600000000", domicilio: "Calle 1" }), { telefono: "600000000", domicilio: "Calle 1" });
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

test("un replay 200 conserva el recibo original y avanza una sola versión", async () => {
  const recibo = { recibo_ref: "acc_original_replay", version: 1 };
  const peticiones = [];
  const controlador = crearControladorContactoPropio({
    autorizacionServidor: { capacidad: true, version: 0, consultarRecibo: true },
    fetchImpl: async (_ruta, opciones) => { peticiones.push(opciones); return { status: 200, json: async () => recibo }; },
  });
  const resultado = await controlador.guardar("ana@example.test");
  assert.equal(resultado.reciboRef, recibo.recibo_ref);
  assert.equal(resultado.version, 1);
  assert.equal(peticiones.length, 1);
  assert.equal(peticiones[0].credentials, "same-origin");
});


test("contacto usa el catálogo i18n común con el mismo texto visible", async () => {
  const claves = JSON.parse(await readFile(new URL("./locales/es.json", import.meta.url), "utf8"));
  assert.equal(textoContactoPropio("guardar"), traducir("areaPersonal.contacto.guardar"));
  assert.equal(claves["areaPersonal.contacto.guardar"], textoContactoPropio("guardar"));
  assert.equal(textoContactoPropio("correcto", { recibo: "acc_prueba" }), traducir("areaPersonal.contacto.correcto", { recibo: "acc_prueba" }));
});
