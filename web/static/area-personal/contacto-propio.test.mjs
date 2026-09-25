import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { crearControladorContactoPropio, capturarCorreoEnviado, montarContactoPropio } from "./contacto-propio.js";
import { crearClienteOperacionesContactoPropio, RUTAS_OPERACIONES_CONTACTO } from "./cliente-http.js?v=20260926-portal-candidato-v1";
import { textoContactoPropio } from "./i18n-contacto-propio.js";
import { traducir } from "./i18n.js";
import { exigirRenovado } from "../portal-empleado/versiones-cache.test-helper.mjs";

const REF = `opr_${"a".repeat(22)}`;
const preparado = (version = 7) => ({ operacion_ref: REF, estado: "preparada", version_esperada: version });
const confirmado = (version = 7) => ({ ...preparado(version), estado: "confirmada", version: version + 1, recibo_ref: "recibo-original" });
function respuesta(status, dato) {
  return { status, headers: { get: (clave) => clave === "Content-Type" ? "application/json" : null }, text: async () => JSON.stringify(dato) };
}
function servidor(secuencia) {
  const peticiones = [];
  const fetchImpl = async (ruta, opciones) => {
    peticiones.push({ ruta, opciones, cuerpo: JSON.parse(opciones.body) });
    const siguiente = secuencia.shift();
    if (typeof siguiente === "function") return siguiente(ruta, opciones);
    return respuesta(...siguiente);
  };
  return { peticiones, fetchImpl };
}
function contenedorDOM() {
  const documento = { activeElement: null };
  const crearNodo = () => ({ children: [], handlers: {}, attrs: {}, append(...nodos) { this.children.push(...nodos); },
    replaceChildren(...nodos) { this.children = nodos; }, setAttribute(clave, valor) { this.attrs[clave] = valor; },
    addEventListener(tipo, fn) { this.handlers[tipo] = fn; }, removeEventListener(tipo) { delete this.handlers[tipo]; },
    checkValidity() { return true; }, focus() { documento.activeElement = this; } });
  documento.createElement = crearNodo;
  return { ownerDocument: documento, replaceChildren(...nodos) { this.children = nodos; } };
}
const autorizado = { capacidad: true, version: 7 };

test("Perfil sin capacidad positiva muestra no configurado y no consulta ni habilita Contacto", () => {
  let llamadas = 0;
  const contenedor = contenedorDOM();
  const vista = montarContactoPropio({ contenedor, fetchImpl: async () => { llamadas += 1; } });
  const [formulario] = contenedor.children;
  const [, acciones, estado] = formulario.children;
  assert.match(estado.textContent, /no configurado/iu);
  assert.equal(acciones.children.every((boton) => boton.disabled), true);
  assert.equal(llamadas, 0);
  assert.equal(vista.controlador.autorizado, false);
  vista.destruir();
});

test("Contacto y arranque comparten la URL versionada del cliente HTTP", async () => {
  const [contacto, arranque] = await Promise.all([
    readFile(new URL("./contacto-propio.js", import.meta.url), "utf8"),
    readFile(new URL("./arranque.js", import.meta.url), "utf8"),
  ]);
  exigirRenovado([contacto, arranque], "./cliente-http.js", ["20260924-f2-b11-v1", "20260924-f2-b11-v2"]);
});

test("prepara 201 y confirma 201 solo por acción explícita, conserva recibo original", async () => {
  const s = servidor([[201, preparado()], [201, confirmado()]]);
  const recibos = [];
  const c = crearControladorContactoPropio({ autorizacionServidor: autorizado, fetchImpl: s.fetchImpl, alConfirmar: (r) => recibos.push(r) });
  const op = await c.preparar(" uno@ejemplo.test ");
  assert.equal(op.estado, "preparada");
  assert.equal(s.peticiones.length, 1);
  assert.deepEqual(s.peticiones[0].cuerpo, { correo: "uno@ejemplo.test", version_esperada: 7 });
  assert.equal(s.peticiones[0].ruta, RUTAS_OPERACIONES_CONTACTO.preparar);
  assert.equal(s.peticiones[0].opciones.credentials, "omit");
  await c.confirmar("uno@ejemplo.test");
  assert.deepEqual(s.peticiones[1].cuerpo, { operacion_ref: REF, correo: "uno@ejemplo.test", version_esperada: 7 });
  assert.equal(s.peticiones[1].ruta, RUTAS_OPERACIONES_CONTACTO.confirmar);
  assert.equal(c.seleccion.recibo_ref, "recibo-original");
  assert.deepEqual(recibos, [{ reciboRef: "recibo-original", version: 8, correo: "uno@ejemplo.test" }]);
});

test("replay 200 de preparación y confirmación conserva una sola versión y recibo", async () => {
  const s = servidor([[200, preparado()], [200, confirmado()]]);
  const c = crearControladorContactoPropio({ autorizacionServidor: autorizado, fetchImpl: s.fetchImpl });
  await c.preparar("uno@ejemplo.test");
  await c.confirmar("uno@ejemplo.test");
  assert.equal(c.operaciones.length, 1);
  assert.equal(c.seleccion.version, 8);
  assert.equal(c.seleccion.recibo_ref, "recibo-original");
});

test("preparar replay 200 ya confirmado recupera recibo sin otro POST confirmar", async () => {
  const s = servidor([[200, confirmado()]]);
  const recibos = [];
  const c = crearControladorContactoPropio({ autorizacionServidor: autorizado, fetchImpl: s.fetchImpl,
    alConfirmar: (r) => recibos.push(r) });
  const resultado = await c.preparar("uno@ejemplo.test");
  assert.equal(resultado.estado, "confirmada");
  assert.equal(c.seleccion.recibo_ref, "recibo-original");
  assert.equal(c.operaciones.length, 1);
  assert.deepEqual(recibos, [{ reciboRef: "recibo-original", version: 8, correo: "uno@ejemplo.test" }]);
  assert.deepEqual(s.peticiones.map((p) => p.ruta), [RUTAS_OPERACIONES_CONTACTO.preparar]);
  const invalida = crearClienteOperacionesContactoPropio({ fetchImpl: async () => respuesta(201, confirmado()) });
  await assert.rejects(() => invalida.preparar("uno@ejemplo.test", 7), (e) => e.codigo === "respuesta_incompatible");
});

test("cliente falla cerrado ante DTO con correo, versión incoherente o recibo en preparación", async () => {
  for (const dato of [
    { ...preparado(), correo: "filtrado@ejemplo.test" },
    { ...preparado(), recibo_ref: "indebido" },
    { ...confirmado(), version: 12 },
  ]) {
    const s = servidor([[200, dato]]);
    const cliente = crearClienteOperacionesContactoPropio({ fetchImpl: s.fetchImpl });
    await assert.rejects(() => cliente.detalle(REF), (error) => error.codigo === "respuesta_incompatible");
  }
});

test("recarga consulta índice completo sin storage y exige selección explícita del confirmado", async () => {
  const s = servidor([[200, { operaciones: [confirmado(), { ...preparado(5), operacion_ref: `opr_${"b".repeat(22)}`, estado: "cancelada" }] }], [200, confirmado()]]);
  const c = crearControladorContactoPropio({ autorizacionServidor: autorizado, fetchImpl: s.fetchImpl });
  await c.cargar();
  assert.equal(c.operaciones.length, 2);
  assert.equal(c.seleccion, null);
  assert.deepEqual(s.peticiones[0].cuerpo, { limite: 20 });
  assert.equal(JSON.stringify(c.operaciones).includes("correo"), false);
  await c.seleccionar(REF);
  assert.equal(c.seleccion.recibo_ref, "recibo-original");
  assert.deepEqual(s.peticiones.map((p) => p.ruta), [RUTAS_OPERACIONES_CONTACTO.consultas, RUTAS_OPERACIONES_CONTACTO.detalle]);
});

test("503 de confirmación consulta detalle exacto y no repite POST; recupera 200 con recibo", async () => {
  const s = servidor([[201, preparado()], [503, { codigo: "confirmacion_incierta", operacion_ref: REF }], [200, confirmado()]]);
  const recibos = [];
  const c = crearControladorContactoPropio({ autorizacionServidor: autorizado, fetchImpl: s.fetchImpl, alConfirmar: (r) => recibos.push(r) });
  await c.preparar("uno@ejemplo.test");
  const resultado = await c.confirmar("uno@ejemplo.test");
  assert.equal(resultado.recibo_ref, "recibo-original");
  assert.equal(c.seleccion.recibo_ref, "recibo-original");
  assert.equal(recibos.length, 1);
  assert.deepEqual(s.peticiones.map((p) => p.ruta), [RUTAS_OPERACIONES_CONTACTO.preparar, RUTAS_OPERACIONES_CONTACTO.confirmar, RUTAS_OPERACIONES_CONTACTO.detalle]);
});

test("503 y detalle 404 dejan incertidumbre, sin atribuir recibo vigente ni reenviar", async () => {
  const s = servidor([[201, preparado()], [503, { codigo: "confirmacion_incierta", operacion_ref: REF }], [404, { codigo: "no_encontrada" }]]);
  const c = crearControladorContactoPropio({ autorizacionServidor: autorizado, fetchImpl: s.fetchImpl });
  await c.preparar("uno@ejemplo.test");
  await assert.rejects(() => c.confirmar("uno@ejemplo.test"));
  assert.equal(c.seleccion, null);
  assert.equal(c.autorizado, false);
  assert.equal(s.peticiones.filter((p) => p.ruta === RUTAS_OPERACIONES_CONTACTO.confirmar).length, 1);
});

test("destruir durante confirmación tardía invalida recibo, correo y callback", async () => {
  let resolverConfirmacion;
  const fetchImpl = async (ruta) => {
    if (ruta === RUTAS_OPERACIONES_CONTACTO.consultas) return respuesta(200, { operaciones: [] });
    if (ruta === RUTAS_OPERACIONES_CONTACTO.preparar) return respuesta(201, preparado());
    if (ruta === RUTAS_OPERACIONES_CONTACTO.confirmar) return new Promise((resolve) => { resolverConfirmacion = resolve; });
    throw new Error("ruta inesperada");
  };
  let callbacks = 0;
  const c = crearControladorContactoPropio({ autorizacionServidor: autorizado, fetchImpl, alConfirmar: () => { callbacks++; } });
  await c.cargar();
  await c.preparar("uno@ejemplo.test");
  const montaje = montarContactoPropio({ contenedor: contenedorDOM(), controlador: c });
  const pendiente = c.confirmar("uno@ejemplo.test");
  montaje.destruir();
  resolverConfirmacion(respuesta(201, confirmado()));
  await assert.rejects(pendiente);
  assert.equal(callbacks, 0);
  assert.equal(c.seleccion, null);
  assert.deepEqual(c.operaciones, []);
  assert.equal(c.autorizado, false);
  assert.equal(JSON.stringify(c).includes("uno@ejemplo.test"), false);
});

test("403 posterior a lista y detalle 200 borra datos y bloquea el controlador", async () => {
  const s = servidor([[200, { operaciones: [confirmado()] }], [200, confirmado()], [403, { codigo: "acceso_denegado" }]]);
  let denegaciones = 0;
  const c = crearControladorContactoPropio({ autorizacionServidor: autorizado, fetchImpl: s.fetchImpl, alDenegar: () => { denegaciones++; } });
  await c.cargar();
  await c.seleccionar(REF);
  assert.equal(c.seleccion.recibo_ref, "recibo-original");
  await assert.rejects(() => c.cargar(), (e) => e.codigo === "acceso_denegado");
  assert.equal(c.seleccion, null);
  assert.deepEqual(c.operaciones, []);
  assert.equal(c.autorizado, false);
  assert.equal(denegaciones, 1);
  await assert.rejects(() => c.cargar(), (e) => e.codigo === "acceso_denegado");
  assert.equal(s.peticiones.length, 3);
});

test("401/403/404 vacíos o HTML purgan lista, detalle, recibo y capacidad antes de leer el cuerpo", async () => {
  for (const [status, codigo] of [[401, "autenticacion_requerida"], [403, "acceso_denegado"], [404, "no_encontrada"]]) {
    for (const cuerpo of ["", "<html>denegado</html>"]) {
      let llamadas = 0; let cuerpoLeido = false; let denegaciones = 0;
      const fetchImpl = async (_ruta, opciones) => {
        llamadas += 1;
        if (llamadas === 1) return respuesta(200, { operaciones: [confirmado()] });
        if (llamadas === 2) return respuesta(200, confirmado());
        assert.equal(opciones.credentials, "omit");
        return { status, headers: { get: () => cuerpo ? "text/html" : null }, text: async () => { cuerpoLeido = true; return cuerpo; } };
      };
      const c = crearControladorContactoPropio({ autorizacionServidor: autorizado, fetchImpl, alDenegar: () => { denegaciones += 1; } });
      await c.cargar();
      await c.seleccionar(REF);
      assert.equal(c.seleccion.recibo_ref, "recibo-original");
      await assert.rejects(() => c.cargar(), (error) => error.codigo === codigo && error.estado === status);
      assert.equal(cuerpoLeido, false);
      assert.equal(c.autorizado, false);
      assert.equal(c.seleccion, null);
      assert.deepEqual(c.operaciones, []);
      assert.equal(denegaciones, 1);
      assert.doesNotMatch(JSON.stringify({ operaciones: c.operaciones, seleccion: c.seleccion, aviso: c.aviso }), /recibo-original/u);
      await assert.rejects(() => c.cargar(), (error) => error.codigo === "acceso_denegado");
      assert.equal(llamadas, 3);
    }
  }
});

test("denegación posterior oculta también el recibo previo del montaje", async () => {
  const s = servidor([[200, { operaciones: [confirmado()] }], [200, confirmado()], [403, { codigo: "acceso_denegado" }]]);
  const c = crearControladorContactoPropio({ autorizacionServidor: autorizado, fetchImpl: s.fetchImpl });
  await c.cargar(); await c.seleccionar(REF);
  const contenedor = contenedorDOM();
  const vista = montarContactoPropio({ contenedor, controlador: c, reciboAnterior: { reciboRef: "recibo-vigente", version: 8 } });
  const situacion = contenedor.children[2];
  assert.equal(situacion.hidden, undefined);
  await assert.rejects(() => c.cargar(), (error) => error.codigo === "acceso_denegado");
  assert.equal(situacion.hidden, true);
  assert.equal(situacion.children[1].children[1].children.length, 0);
  assert.match(contenedor.children[0].children[2].textContent, /permiso/iu);
  vista.destruir();
});

test("409 con consulta automática 403 o 404 borra recibo y no mantiene autorización", async () => {
  for (const [estado, codigo] of [[403, "acceso_denegado"], [404, "no_encontrada"]]) {
    const s = servidor([[200, { operaciones: [confirmado()] }], [200, confirmado()],
      [409, { codigo: "operacion_preparada", operacion_ref: REF }], [estado, { codigo }]]);
    let denegaciones = 0;
    const c = crearControladorContactoPropio({ autorizacionServidor: autorizado, fetchImpl: s.fetchImpl, alDenegar: () => { denegaciones++; } });
    await c.cargar();
    await c.seleccionar(REF);
    await assert.rejects(() => c.preparar("otro@ejemplo.test"), (e) => e.codigo === "operacion_preparada");
    assert.equal(c.autorizado, false);
    assert.equal(c.seleccion, null);
    assert.deepEqual(c.operaciones, []);
    assert.equal(denegaciones, 1);
  }
});

test("dos operaciones con igual estado y versión mantienen rótulo y selección distinguibles", async () => {
  const otra = `opr_${"b".repeat(22)}`;
  const a = { ...preparado(), estado: "cancelada" };
  const b = { ...a, operacion_ref: otra };
  const s = servidor([[200, { operaciones: [a, b] }], [200, b]]);
  const c = crearControladorContactoPropio({ autorizacionServidor: autorizado, fetchImpl: s.fetchImpl });
  await c.cargar();
  const contenedor = contenedorDOM();
  const montaje = montarContactoPropio({ contenedor, controlador: c });
  await c.seleccionar(otra);
  const filas = contenedor.children[1].children[1].children;
  assert.notEqual(filas[0].children[1].textContent, filas[1].children[1].textContent);
  assert.equal(filas.filter((fila) => fila.className.includes("seleccionada")).length, 1);
  assert.equal(filas.find((fila) => fila.className.includes("seleccionada")).children[0].attrs["aria-pressed"], "true");
  assert.equal(filas.find((fila) => !fila.className.includes("seleccionada")).children[0].attrs["aria-pressed"], "false");
  assert.match(c.aviso, new RegExp(otra.slice(-8), "u"));
  montaje.destruir();
});

test("remonte durante callback de confirmación no convierte el recibo válido en error", async () => {
  const s = servidor([[201, preparado()], [201, confirmado()]]);
  let c;
  let callbacks = 0;
  c = crearControladorContactoPropio({ autorizacionServidor: autorizado, fetchImpl: s.fetchImpl,
    alConfirmar: () => { callbacks++; c.destruir(); } });
  await c.preparar("uno@ejemplo.test");
  const resultado = await c.confirmar("uno@ejemplo.test");
  assert.equal(resultado.recibo_ref, "recibo-original");
  assert.equal(callbacks, 1);
  assert.equal(c.autorizado, false);
});

test("remonte tras 201 o 503 recuperado enfoca estado concreto con recibo original", async () => {
  for (const incierto of [false, true]) {
    const peticiones = [];
    let consultas = 0;
    let resolverIndiceNuevo;
    const fetchImpl = async (ruta) => {
      peticiones.push(ruta);
      if (ruta === RUTAS_OPERACIONES_CONTACTO.consultas) return consultas++ === 0
        ? respuesta(200, { operaciones: [] })
        : new Promise((resolve) => { resolverIndiceNuevo = resolve; });
      if (ruta === RUTAS_OPERACIONES_CONTACTO.preparar) return respuesta(201, preparado());
      if (ruta === RUTAS_OPERACIONES_CONTACTO.confirmar) return incierto
        ? respuesta(503, { codigo: "confirmacion_incierta", operacion_ref: REF }) : respuesta(201, confirmado());
      if (ruta === RUTAS_OPERACIONES_CONTACTO.detalle) return respuesta(200, confirmado());
      throw new Error("ruta inesperada");
    };
    const contenedor = contenedorDOM();
    let montaje;
    const c = crearControladorContactoPropio({ autorizacionServidor: autorizado, fetchImpl, alConfirmar: (resultado) => {
      montaje.destruir();
      const nuevo = crearControladorContactoPropio({ autorizacionServidor: { capacidad: true, version: resultado.version }, fetchImpl });
      montaje = montarContactoPropio({ contenedor, controlador: nuevo, reciboAnterior: resultado,
        confirmacionReciente: resultado, enfocarConfirmacion: true });
    } });
    await c.cargar();
    await c.preparar("uno@ejemplo.test");
    montaje = montarContactoPropio({ contenedor, controlador: c });
    const resultado = await c.confirmar("uno@ejemplo.test");
    assert.equal(resultado.recibo_ref, "recibo-original");
    const status = contenedor.children[0].children[2];
    assert.equal(contenedor.ownerDocument.activeElement, status);
    assert.match(status.textContent, /recibo-original/u);
    assert.equal(contenedor.children[0].children.length, 3);
    assert.match(contenedor.children[2].className, /contacto-situacion-panel/u);
    resolverIndiceNuevo(respuesta(200, { operaciones: [] }));
    await new Promise((resolve) => setImmediate(resolve));
    assert.doesNotMatch(status.textContent, /recibo-original/u);
    assert.equal(peticiones.filter((ruta) => ruta === RUTAS_OPERACIONES_CONTACTO.confirmar).length, 1);
    montaje.destruir();
  }
});

test("recarga con dos operaciones separa recibo vigente GET del detalle seleccionado", async () => {
  const otra = `opr_${"b".repeat(22)}`;
  const a = { ...confirmado(6), recibo_ref: "recibo-A" };
  const b = { ...confirmado(7), operacion_ref: otra, recibo_ref: "recibo-B" };
  const s = servidor([[200, { operaciones: [b, a] }], [200, a]]);
  const c = crearControladorContactoPropio({ autorizacionServidor: { capacidad: true, version: 8 }, fetchImpl: s.fetchImpl });
  await c.cargar();
  const contenedor = contenedorDOM();
  const montaje = montarContactoPropio({ contenedor, controlador: c, autorizacionServidor: { capacidad: true, version: 8 },
    reciboAnterior: { reciboRef: "recibo-B", version: 8 } });
  const status = contenedor.children[0].children[2];
  assert.doesNotMatch(status.textContent, /recibo-[AB]/u);
  const detalleVigente = contenedor.children[2].children[1].children[1];
  assert.match(detalleVigente.children[1].textContent, /recibo-B/u);
  await c.seleccionar(REF);
  assert.match(status.textContent, /recibo-A/u);
  assert.doesNotMatch(status.textContent, /recibo-B/u);
  montaje.destruir();
});

test("403/404 no muestran datos ajenos; 409 entre pestañas exige consulta", async () => {
  for (const [status, codigo] of [[403, "acceso_denegado"], [404, "no_encontrada"]]) {
    const s = servidor([[status, { codigo }]]);
    const c = crearControladorContactoPropio({ autorizacionServidor: autorizado, fetchImpl: s.fetchImpl });
    await assert.rejects(() => c.seleccionar(REF), (e) => e.estado === status);
    assert.equal(c.seleccion, null);
  }
  const s = servidor([[409, { codigo: "operacion_preparada", operacion_ref: REF }], [200, { operaciones: [preparado()] }], [200, preparado()]]);
  const c = crearControladorContactoPropio({ autorizacionServidor: autorizado, fetchImpl: s.fetchImpl });
  await assert.rejects(() => c.preparar("dos@ejemplo.test"), (e) => e.codigo === "operacion_preparada");
  assert.equal(c.seleccion, null);
  assert.equal(c.operaciones[0].estado, "preparada");
  await c.seleccionar(REF);
  assert.equal(c.seleccion.estado, "preparada");
});

test("cancelación 200 desbloquea intención nueva; respuesta tardía de otra pestaña queda en conflicto", async () => {
  const s = servidor([[201, preparado()], [200, { ...preparado(), estado: "cancelada" }],
    [201, { ...preparado(), operacion_ref: `opr_${"c".repeat(22)}` }]]);
  const c = crearControladorContactoPropio({ autorizacionServidor: autorizado, fetchImpl: s.fetchImpl });
  await c.preparar("uno@ejemplo.test");
  await c.cancelar();
  assert.equal(c.seleccion.estado, "cancelada");
  await c.preparar("dos@ejemplo.test");
  assert.equal(c.seleccion.estado, "preparada");
  const a = servidor([[201, preparado()], [409, { codigo: "conflicto" }]]);
  const pestañaA = crearControladorContactoPropio({ autorizacionServidor: autorizado, fetchImpl: a.fetchImpl });
  await pestañaA.preparar("uno@ejemplo.test");
  await assert.rejects(() => pestañaA.confirmar("uno@ejemplo.test"), (e) => e.codigo === "conflicto");
  assert.equal(pestañaA.seleccion.recibo_ref, undefined);
});

test("entrada capturada e i18n real sin correo en el índice", async () => {
  const entrada = { value: " persona@ejemplo.test " };
  assert.equal(capturarCorreoEnviado(entrada), "persona@ejemplo.test");
  entrada.value = "otra@ejemplo.test";
  assert.equal(textoContactoPropio("historialTitulo"), traducir("areaPersonal.contacto.historialTitulo"));
  const claves = JSON.parse(await readFile(new URL("./locales/es.json", import.meta.url), "utf8"));
  assert.equal(claves["areaPersonal.contacto.historialTitulo"], textoContactoPropio("historialTitulo"));
});
