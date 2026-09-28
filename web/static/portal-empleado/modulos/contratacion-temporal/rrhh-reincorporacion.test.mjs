import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteReincorporacionRRHHHTTP, RUTA_CAPACIDAD_REINCORPORACION_RRHH, RUTA_REINCORPORACION_RRHH } from "./rrhh-reincorporacion-cliente.js";
import { validarReciboReincorporacionRRHH, validarSolicitudReincorporacionRRHH } from "./rrhh-reincorporacion-contrato.js";
import { montarFormularioReincorporacionRRHH } from "./rrhh-reincorporacion-formulario.js";
import { MENSAJES_REINCORPORACION_RRHH_ES, crearTraductorReincorporacionRRHH } from "./rrhh-reincorporacion-i18n.js";

const expediente = { expediente_ref: "expediente:reincorporacion:1", version_esperada: 7 };
const clave = "11111111-1111-4111-8111-111111111111";
const solicitud = {
  ...expediente,
  relacion_ref: "relacion:sustituto:1",
  fecha_efectiva: "2026-09-28",
  documento_ref: "documento:reincorporacion:1",
  documento_sha256: "a".repeat(64),
  clave_idempotencia: clave,
};
const recibo = {
  expediente_ref: solicitud.expediente_ref,
  relacion_ref: solicitud.relacion_ref,
  fecha_efectiva: solicitud.fecha_efectiva,
  cese_evento_ref: "evento:cese:1",
  cese_recibo_ref: "recibo:cese:1",
  recibo_ref: "recibo:reincorporacion:1",
  evento_ref: "evento:reincorporacion:1",
  registrada_en: "2026-09-28T11:30:00Z",
  estado_bolsa: "pendiente_confirmacion",
};

function raizFalsa() {
  const eventos = new Map();
  const raiz = {
    innerHTML: "",
    addEventListener: (nombre, funcion) => eventos.set(nombre, funcion),
    removeEventListener: (nombre) => eventos.delete(nombre),
    replaceChildren() { this.innerHTML = ""; },
    enviar(datos = solicitud) {
      const elementos = Object.fromEntries(["relacion_ref", "fecha_efectiva", "documento_ref", "documento_sha256"]
        .map((nombre) => [nombre, { value: datos[nombre] }]));
      return eventos.get("submit")({ preventDefault() {}, target: { dataset: { ctRrhhReincorporacionForm: "" }, elements: { namedItem: (nombre) => elementos[nombre] } } });
    },
    ayuda() { eventos.get("click")({ target: { closest: () => ({}) } }); },
  };
  return raiz;
}

function montar(cliente, opciones = {}) {
  const raiz = raizFalsa();
  const desmontar = montarFormularioReincorporacionRRHH({ raiz, cliente, expediente,
    puedeRegistrar: true, confirmarOperacion: () => true, generarClaveIdempotencia: () => clave, ...opciones });
  return { raiz, desmontar };
}

test("contrato enlaza expediente, relación, fecha y cese al recibo sin aceptar disponibilidad de Bolsa", () => {
  assert.deepEqual(validarSolicitudReincorporacionRRHH(solicitud), solicitud);
  assert.equal(validarReciboReincorporacionRRHH(recibo, solicitud).cese_recibo_ref, recibo.cese_recibo_ref);
  assert.throws(() => validarReciboReincorporacionRRHH({ ...recibo, relacion_ref: "relacion:ajena:1" }, solicitud));
  assert.throws(() => validarReciboReincorporacionRRHH({ ...recibo, estado_bolsa: "disponible" }, solicitud));
  assert.throws(() => validarSolicitudReincorporacionRRHH({ ...solicitud, fecha_efectiva: "2026-02-30" }));
});

test("cliente usa POST con versión e idempotencia y admite 201 o replay 200", async () => {
  const llamadas = [];
  const cliente = crearClienteReincorporacionRRHHHTTP({
    ejecutar: async (configuracion) => { llamadas.push(configuracion); return configuracion.validarRespuesta(recibo); },
    validarOpciones: (opciones = {}) => opciones,
  });
  assert.equal((await cliente.registrarReincorporacion(solicitud)).recibo_ref, recibo.recibo_ref);
  assert.equal(llamadas[0].ruta, RUTA_REINCORPORACION_RRHH);
  assert.deepEqual(llamadas[0].entrada, solicitud);
  assert.deepEqual(llamadas[0].estadoEsperado, [200, 201]);
  assert.equal(llamadas[0].efecto, true);
});

test("preflight sin efecto exige contexto exacto y solo habilita con decisión V3 positiva", async () => {
  const llamadas = [];
  const cliente = crearClienteReincorporacionRRHHHTTP({
    ejecutar: async (configuracion) => {
      llamadas.push(configuracion);
      return configuracion.validarRespuesta({ esquema: "vec.contratacion-temporal.capacidad-reincorporacion-titular.v1",
        puede_registrar_reincorporacion_titular: llamadas.length === 2 });
    },
    validarOpciones: (opciones = {}) => opciones,
  });
  assert.equal(await cliente.consultarCapacidadReincorporacion(expediente), false);
  assert.equal(await cliente.consultarCapacidadReincorporacion(expediente), true);
  assert.equal(llamadas[0].metodo, "POST");
  assert.equal(llamadas[0].efecto, false);
  assert.equal(llamadas[0].ruta, RUTA_CAPACIDAD_REINCORPORACION_RRHH);
  assert.deepEqual(llamadas[0].entrada, expediente);
  assert.throws(() => cliente.consultarCapacidadReincorporacion({ ...expediente, version_esperada: 0 }));
  await assert.rejects(async () => cliente.consultarCapacidadReincorporacion({ ...expediente, expediente_ref: "ajena?x" }));
});

test("sin concesión positiva no aparece el formulario ni sale petición", () => {
  const raiz = raizFalsa();
  const desmontar = montarFormularioReincorporacionRRHH({ raiz, expediente,
    cliente: { registrarReincorporacion() { assert.fail("POST inesperado"); } } });
  assert.doesNotMatch(raiz.innerHTML, /data-ct-rrhh-reincorporacion-form/u);
  assert.match(raiz.innerHTML, /No tiene permiso/u);
  desmontar();
});

test("el formulario oculta ayuda hasta pulsar ?, escapa i18n y muestra recibo con Bolsa pendiente", async () => {
  const mensajes = { ...MENSAJES_REINCORPORACION_RRHH_ES, rrhh_reincorporacion_titulo: "Título <seguro>" };
  const x = montar({ async registrarReincorporacion() { return recibo; } }, {
    t: crearTraductorReincorporacionRRHH(mensajes),
    datosIniciales: { relacion_ref: "relacion:<titular>" },
  });
  assert.match(x.raiz.innerHTML, /Título &lt;seguro&gt;/u);
  assert.match(x.raiz.innerHTML, /relacion:&lt;titular&gt;/u);
  assert.match(x.raiz.innerHTML, /id="ct-reincorp-ayuda"[^>]*hidden/u);
  x.raiz.ayuda();
  assert.doesNotMatch(x.raiz.innerHTML, /id="ct-reincorp-ayuda"[^>]*hidden/u);
  await x.raiz.enviar();
  assert.match(x.raiz.innerHTML, /recibo:reincorporacion:1/u);
  assert.match(x.raiz.innerHTML, /Pendiente de confirmación por Bolsa/u);
  assert.doesNotMatch(x.raiz.innerHTML, /disponible/u);
  x.desmontar();
});

test("resultado incierto conserva y reenvía exactamente la operación original", async () => {
  const llamadas = [];
  const x = montar({ async registrarReincorporacion(entrada) {
    llamadas.push(entrada);
    if (llamadas.length === 1) throw new Error("red");
    return recibo;
  } });
  await x.raiz.enviar();
  assert.match(x.raiz.innerHTML, /Clave de la operación original/u);
  await x.raiz.enviar({ ...solicitud, relacion_ref: "relacion:distinta:1" });
  assert.deepEqual(llamadas, [solicitud, solicitud]);
  assert.match(x.raiz.innerHTML, /recibo:reincorporacion:1/u);
  x.desmontar();
});

test("409 bloquea el registro hasta actualizar expediente; 422 permite corregir datos", async () => {
  let conflictos = 0;
  const x = montar({ async registrarReincorporacion() {
    conflictos++;
    throw Object.assign(new Error("conflicto"), { estado: 409, resultadoIndeterminado: false, envelopeValido: true });
  } });
  await x.raiz.enviar();
  await x.raiz.enviar();
  assert.equal(conflictos, 1);
  assert.match(x.raiz.innerHTML, /Actualice el detalle/u);
  x.desmontar();

  const recibidas = [];
  const y = montar({ async registrarReincorporacion(entrada) {
    recibidas.push(entrada);
    if (recibidas.length === 1) throw Object.assign(new Error("contenido"), { estado: 422, resultadoIndeterminado: false, envelopeValido: true });
    return { ...recibo, relacion_ref: entrada.relacion_ref };
  } });
  await y.raiz.enviar();
  await y.raiz.enviar({ ...solicitud, relacion_ref: "relacion:titular:2" });
  assert.equal(recibidas[1].relacion_ref, "relacion:titular:2");
  assert.match(y.raiz.innerHTML, /recibo:reincorporacion:1/u);
  y.desmontar();
});

test("403 retira campos y desmontar aborta una respuesta tardía", async () => {
  const denegaciones = [];
  const x = montar({ async registrarReincorporacion() { throw Object.assign(new Error("denegado"), { estado: 403, resultadoIndeterminado: false }); } },
    { alDenegacion: () => denegaciones.push(true) });
  await x.raiz.enviar();
  assert.equal(denegaciones.length, 1);
  assert.doesNotMatch(x.raiz.innerHTML, /relacion:titular:1|documento:reincorporacion:1|data-ct-rrhh-reincorporacion-form/u);
  x.desmontar();

  let resolver;
  let signal;
  const pendiente = montar({ registrarReincorporacion(_entrada, opciones) {
    signal = opciones.signal;
    return new Promise((resuelve) => { resolver = resuelve; });
  } });
  const vuelo = pendiente.raiz.enviar();
  pendiente.desmontar();
  assert.equal(signal.aborted, true);
  resolver(recibo);
  await vuelo;
  assert.equal(pendiente.raiz.innerHTML, "");
});
