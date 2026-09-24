import assert from "node:assert/strict";
import { randomUUID } from "node:crypto";
import test from "node:test";
import { montarFormularioSubsanacionReparos } from "./formulario-subsanacion-reparos.js";
import { ESQUEMA_RECUPERACION_SUBSANACION, MAXIMO_ARCHIVO_RECUPERACION_SUBSANACION, serializarDatosRecuperacionSubsanacion, validarDatosRecuperacionSubsanacion } from "./cliente-http-subsanacion-reparos.js";
import { MENSAJES_SUBSANACION_REPAROS_ES as textos } from "./i18n-subsanacion-reparos.js";

const contexto = { expediente_ref: "expediente:subsanacion:archivo", version_esperada: 6 };
const solicitud = { ...contexto, clave_idempotencia: randomUUID(), observaciones: "Justificación corregida." };
const recibo = { esquema: "vec.contratacion-temporal.recibo-subsanacion-reparos.v1", operacion: "registrar_subsanacion", expediente_ref: contexto.expediente_ref, version_resultante: 7, fase_resultante: "subsanacion_unidad", estado_resultante: "incidencia", recibo_ref: "recibo:subsanacion:archivo", auditoria_ref: "auditoria:subsanacion:archivo", evento_ref: "evento:subsanacion:archivo", actor_ref: "actor:subsanacion:archivo", registrada_en: "2026-09-13T08:00:00Z" };

function archivo(texto, size = new TextEncoder().encode(texto).byteLength) {
  return { size, text: async () => texto };
}

function escenario({ respuestas = [], intencionInicial = null, soloImportar = false, guardar = () => true } = {}) {
  const eventos = new Map(), peticiones = [], cambios = [], confirmaciones = [];
  const enlaces = [];
  const raiz = {
    innerHTML: "",
    ownerDocument: {
      body: { append: (enlace) => enlaces.push(enlace) },
      createElement: () => ({ click() { this.pulsado = true; }, remove() {} }),
    },
    addEventListener: (nombre, funcion) => eventos.set(nombre, funcion),
    removeEventListener: (nombre) => eventos.delete(nombre),
    contains: () => true,
    querySelector: () => ({ focus() {} }),
    replaceChildren() { this.innerHTML = ""; },
  };
  const desmontar = montarFormularioSubsanacionReparos({
    raiz, contexto, intencionInicial, soloImportar,
    traducir: (claveTexto) => textos[claveTexto] ?? claveTexto,
    confirmarOperacion: () => true,
    generarClaveIdempotencia: () => solicitud.clave_idempotencia,
    alCambiarIntencion: (intencion) => { cambios.push(intencion); return guardar(intencion); },
    alDenegacion: () => {},
    alConfirmar: (confirmado, contextoOriginal) => confirmaciones.push({ confirmado, contextoOriginal }),
    cliente: { registrarSubsanacionReparos: async (dto) => {
      peticiones.push(dto);
      const respuesta = respuestas.shift();
      if (respuesta instanceof Error) throw respuesta;
      return respuesta;
    } },
  });
  function pulsar(selector) {
    return eventos.get("click")({ target: { closest: (patron) => patron === selector ? {} : null }, preventDefault() {} });
  }
  return {
    raiz, peticiones, cambios, confirmaciones, enlaces, desmontar,
    preparar: () => eventos.get("submit")({ target: { closest: (patron) => patron === "[data-ct-subsanacion-form]" ? { elements: { namedItem: () => ({ value: solicitud.observaciones }) } } : null }, preventDefault() {} }),
    descargar: () => pulsar("[data-ct-subsanacion-guardar]"),
    enviar: () => pulsar("[data-ct-subsanacion-enviar]"),
    recuperar: () => pulsar("[data-ct-subsanacion-recuperar]"),
    importar: (dato) => eventos.get("change")({ target: { closest: (patron) => patron === "[data-ct-subsanacion-archivo]" ? { files: [dato] } : null } }),
  };
}

test("archivo cerrado: solo esquema y DTO original del expediente/version", () => {
  const texto = serializarDatosRecuperacionSubsanacion(solicitud);
  const datos = JSON.parse(texto);
  assert.deepEqual(Object.keys(datos), ["esquema", "solicitud"]);
  assert.deepEqual(datos.solicitud, solicitud);
  assert.deepEqual(validarDatosRecuperacionSubsanacion(datos, contexto).solicitud, solicitud);
  for (const alterado of [
    { ...datos, actor_ref: "actor:ajeno" },
    { ...datos, esquema: "otro.esquema" },
    { ...datos, solicitud: { ...solicitud, expediente_ref: "expediente:ajeno" } },
    { ...datos, solicitud: { ...solicitud, version_esperada: 5 } },
    { ...datos, solicitud: { ...solicitud, recibo_ref: "recibo:falso" } },
  ]) assert.throws(() => validarDatosRecuperacionSubsanacion(alterado, contexto));
});

test("descarga previa, 503, importación posterior y replay del mismo comando sin alta nueva", async (t) => {
  const crearURL = URL.createObjectURL, revocarURL = URL.revokeObjectURL;
  const blobs = [], revocadas = [];
  URL.createObjectURL = (blob) => { blobs.push(blob); return "blob:subsanacion-prueba"; };
  URL.revokeObjectURL = (url) => revocadas.push(url);
  t.after(() => { URL.createObjectURL = crearURL; URL.revokeObjectURL = revocarURL; });
  const fallo = new Error("servicio no disponible"); fallo.estado = 503;
  const primera = escenario({ respuestas: [fallo] });
  primera.preparar();
  assert.equal(primera.peticiones.length, 0);
  assert.deepEqual(primera.cambios.map((intencion) => intencion.incierta), [false]);
  await primera.enviar();
  assert.equal(primera.peticiones.length, 0, "no se envía sin iniciar la descarga");
  primera.descargar();
  assert.equal(primera.enlaces.length, 1);
  assert.equal(primera.enlaces[0].download, "recuperacion-subsanacion-v1.json");
  assert.equal(primera.enlaces[0].pulsado, true);
  const texto = await blobs[0].text();
  assert.deepEqual(JSON.parse(texto), { esquema: ESQUEMA_RECUPERACION_SUBSANACION, solicitud });
  await primera.enviar();
  assert.equal(primera.peticiones.length, 1);
  assert.deepEqual(primera.cambios.map((intencion) => intencion.incierta), [false, true]);
  primera.desmontar();
  assert.equal(revocadas.length, 1);

  const segunda = escenario({ respuestas: [recibo], soloImportar: true });
  assert.doesNotMatch(segunda.raiz.innerHTML, /data-ct-subsanacion-form/u);
  await segunda.importar(archivo(texto));
  assert.equal(segunda.peticiones.length, 0);
  assert.deepEqual(segunda.cambios[0], { solicitud, incierta: true });
  assert.match(segunda.raiz.innerHTML, /data-ct-subsanacion-recuperar/u);
  await segunda.recuperar();
  assert.equal(segunda.peticiones.length, 1);
  assert.deepEqual(segunda.peticiones[0], primera.peticiones[0]);
  assert.deepEqual(segunda.confirmaciones, [{ confirmado: recibo, contextoOriginal: contexto }]);
  assert.equal(segunda.cambios.at(-1), null);
  segunda.desmontar();
});

test("archivo ajeno, extra y tamaños declarados/leídos no sustituyen ni producen POST", async () => {
  const bueno = JSON.parse(serializarDatosRecuperacionSubsanacion(solicitud));
  const x = escenario({ soloImportar: true });
  for (const dato of [
    archivo(JSON.stringify({ ...bueno, otra_cosa: true })),
    archivo(JSON.stringify({ ...bueno, solicitud: { ...solicitud, expediente_ref: "expediente:ajeno" } })),
    archivo(JSON.stringify(bueno), MAXIMO_ARCHIVO_RECUPERACION_SUBSANACION + 1),
    archivo(" ".repeat(MAXIMO_ARCHIVO_RECUPERACION_SUBSANACION + 1), 10),
  ]) {
    await x.importar(dato);
    assert.equal(x.peticiones.length, 0);
    assert.equal(x.cambios.length, 0);
    assert.doesNotMatch(x.raiz.innerHTML, /data-ct-subsanacion-recuperar/u);
  }
  await x.importar(archivo(JSON.stringify(bueno)));
  assert.equal(x.cambios.length, 1);
  await x.importar(archivo(JSON.stringify({ ...bueno, solicitud: { ...solicitud, clave_idempotencia: randomUUID() } })));
  assert.equal(x.cambios.length, 1, "una intención pendiente no puede sustituirse");
  assert.equal(x.peticiones.length, 0);
  x.desmontar();
});

test("rechazo de conservar intención impide primer POST e importación", async () => {
  const x = escenario({ guardar: () => false });
  x.preparar();
  x.descargar();
  await x.enviar();
  assert.equal(x.peticiones.length, 0);
  assert.doesNotMatch(x.raiz.innerHTML, /data-ct-subsanacion-enviar/u);
  await x.importar(archivo(serializarDatosRecuperacionSubsanacion(solicitud)));
  assert.equal(x.peticiones.length, 0);
  assert.doesNotMatch(x.raiz.innerHTML, /data-ct-subsanacion-recuperar/u);
  x.desmontar();
  const y = escenario({ guardar: (intencion) => intencion?.incierta !== true });
  y.preparar(); y.descargar(); await y.enviar();
  assert.equal(y.peticiones.length, 0);
  assert.deepEqual(y.cambios.map((intencion) => intencion.incierta), [false, true]);
  y.desmontar();
});

test("reentrada con intención inicial usa el DTO antiguo sin generar otra clave", async () => {
  const x = escenario({ intencionInicial: { solicitud, incierta: true }, soloImportar: true, respuestas: [recibo] });
  assert.doesNotMatch(x.raiz.innerHTML, /data-ct-subsanacion-form/u);
  await x.recuperar();
  assert.deepEqual(x.peticiones, [solicitud]);
  assert.deepEqual(x.confirmaciones, [{ confirmado: recibo, contextoOriginal: contexto }]);
  x.desmontar();
});
