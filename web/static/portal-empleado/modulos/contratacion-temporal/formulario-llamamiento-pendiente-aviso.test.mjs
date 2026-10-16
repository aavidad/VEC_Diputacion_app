import assert from "node:assert/strict";
import test from "node:test";
import { CLAVE, EXPEDIENTE, recibo, comunicacionRegistrada, raizPrueba, montar } from "./formulario-llamamiento-pruebas.js";

const contexto = { expediente_ref: EXPEDIENTE, version_esperada: 6 };
const pendiente = (sufijo = "uno", instante = "2026-09-24T10:10:00Z") => ({
  organizacion_ref: recibo.organizacion_ref, expediente_ref: EXPEDIENTE,
  llamamiento_ref: `llamamiento:pendiente:${sufijo}`,
  estado: "seleccion_confirmada_pendiente_aviso", registrada_en: instante, version: 1,
  antecedente_tipo: "seleccion_confirmada", recibo_antecedente_ref: `recibo:pendiente:${sufijo}`,
});
const anterior = (llamamiento_ref = "llamamiento:anterior") => ({
  organizacion_ref: recibo.organizacion_ref, expediente_ref: EXPEDIENTE,
  llamamiento_ref, comunicacion_ref: "comunicacion:anterior", version: 2,
  estado: "registrada_localmente", registrada_en: "2026-09-24T10:11:00Z",
  recibo_comunicacion_ref: "recibo:comunicacion:anterior", antecedente_tipo: "seleccion_confirmada",
  recibo_antecedente_ref: "recibo:seleccion:anterior", estado_respuesta: "sin_respuesta",
});
const esperar = () => new Promise(setImmediate);

test("selección O6 persistida se restaura tras cada carga y solo habilita anotar el aviso", async () => {
  let lecturas = 0, selecciones = 0, avisos = 0;
  const cliente = { consultarComunicacionesExpediente: async () => {
    lecturas += 1; return { expediente_ref: EXPEDIENTE, comunicaciones: [pendiente()] };
  }, seleccionarLlamamiento: async () => { selecciones += 1; },
  registrarComunicacionLlamamiento: async () => { avisos += 1; } };
  for (let recarga = 0; recarga < 2; recarga += 1) {
    const raiz = raizPrueba(); const cerrar = montar(raiz, cliente, { contexto });
    await esperar();
    assert.match(raiz.innerHTML, /Pendiente de aviso/u);
    assert.match(raiz.innerHTML, /Continuar con el aviso/u);
    assert.match(raiz.innerHTML, /data-ct-llamamiento-recibo="seleccion"/u);
    assert.match(raiz.innerHTML, /data-ct-llamamiento-form="comunicacion"/u);
    assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="seleccion"|data-ct-llamamiento-form="respuesta"/u);
    await raiz.enviar("seleccion", {});
    assert.equal(selecciones, 0);
    assert.equal(avisos, 0);
    cerrar();
  }
  assert.equal(lecturas, 2);
});

test("continuar el aviso usa el justificante y la versión reales sin repetir la selección", async () => {
  const raiz = raizPrueba(); const avisos = []; let selecciones = 0;
  const cerrar = montar(raiz, {
    consultarComunicacionesExpediente: async () => ({ expediente_ref: EXPEDIENTE, comunicaciones: [pendiente()] }),
    seleccionarLlamamiento: async () => { selecciones += 1; },
    registrarComunicacionLlamamiento: async (solicitud) => {
      avisos.push(solicitud);
      throw new Error("sin recibo de comunicación");
    },
  }, { contexto });
  await esperar();
  await raiz.enviar("comunicacion", { clave_idempotencia: CLAVE });
  assert.deepEqual(avisos, [{ clave_idempotencia: CLAVE, organizacion_ref: recibo.organizacion_ref,
    expediente_ref: EXPEDIENTE, llamamiento_ref: pendiente().llamamiento_ref, version_esperada: 1,
    prueba_entrega_ref: pendiente().recibo_antecedente_ref }]);
  assert.equal(selecciones, 0);
  cerrar();
});

test("un recibo real de comunicación deja de presentar el aviso como pendiente", async () => {
  const raiz = raizPrueba();
  const cerrar = montar(raiz, {
    consultarComunicacionesExpediente: async () => ({ expediente_ref: EXPEDIENTE, comunicaciones: [pendiente()] }),
    registrarComunicacionLlamamiento: async () => comunicacionRegistrada,
  }, { contexto });
  await esperar();
  await raiz.enviar("comunicacion", { clave_idempotencia: CLAVE });
  assert.match(raiz.innerHTML, /Aviso anotado/u);
  assert.doesNotMatch(raiz.innerHTML, /Pendiente de aviso|Continuar con el aviso/u);
  assert.match(raiz.innerHTML, /data-ct-llamamiento-recibo="comunicacion"/u);
  cerrar();
});

test("pendiente mezclada con comunicación del mismo llamamiento o segunda O6 cierra la vista", async () => {
  for (const filas of [[pendiente(), anterior(pendiente().llamamiento_ref)],
    [pendiente(), pendiente("dos", "2026-09-24T10:12:00Z")]]) {
    const raiz = raizPrueba(); let selecciones = 0;
    const cerrar = montar(raiz, { consultarComunicacionesExpediente: async () => ({
      expediente_ref: EXPEDIENTE, comunicaciones: filas,
    }), seleccionarLlamamiento: async () => { selecciones += 1; } }, { contexto });
    await esperar();
    assert.match(raiz.innerHTML, /No se han podido cargar los llamamientos/u);
    assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form=|data-ct-llamamiento-recibo="seleccion"/u);
    await raiz.enviar("seleccion", {});
    assert.equal(selecciones, 0);
    cerrar();
  }
});

test("la segunda página debe terminar antes de mostrar el recibo o permitir un aviso", async () => {
  const raiz = raizPrueba(); let lecturas = 0, avisos = 0;
  const filas = Array.from({ length: 9 }, (_, i) => ({ ...anterior(), comunicacion_ref: `comunicacion:previa:${i + 1}`,
    llamamiento_ref: `llamamiento:previo:${i + 1}`, registrada_en: `2026-09-24T10:0${i}:00Z` }));
  const cerrar = montar(raiz, { consultarComunicacionesExpediente: async (consulta) => {
    lecturas += 1;
    if (consulta.cursor) throw Object.assign(new Error("sin página"), { estado: 503,
      codigo: "servicio_no_disponible", envelopeValido: true });
    return { expediente_ref: EXPEDIENTE, comunicaciones: [...filas, pendiente()],
      siguiente_cursor: `${pendiente().llamamiento_ref}#${"a".repeat(64)}` };
  }, registrarComunicacionLlamamiento: async () => { avisos += 1; } }, { contexto });
  await esperar();
  assert.equal(lecturas, 2);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-recibo="seleccion"|data-ct-llamamiento-form="comunicacion"/u);
  await raiz.enviar("comunicacion", {});
  assert.equal(avisos, 0);
  cerrar();
});

test("una pendiente en la segunda página aparece solo tras consultar también esa página", async () => {
  const raiz = raizPrueba(); const consultas = [];
  const anteriores = Array.from({ length: 10 }, (_, i) => ({ ...anterior(),
    comunicacion_ref: `comunicacion:previa:${i + 1}`,
    llamamiento_ref: `llamamiento:previo:${i + 1}`,
    registrada_en: `2026-09-24T10:${String(i).padStart(2, "0")}:00Z` }));
  const cierre = montar(raiz, { consultarComunicacionesExpediente: async (consulta) => {
    consultas.push(consulta);
    return consulta.cursor ? { expediente_ref: EXPEDIENTE,
      comunicaciones: [pendiente("segunda", "2026-09-24T10:11:00Z")] }
      : { expediente_ref: EXPEDIENTE, comunicaciones: anteriores,
        siguiente_cursor: `${anteriores.at(-1).comunicacion_ref}#${"c".repeat(64)}` };
  } }, { contexto });
  await esperar();
  assert.equal(consultas.length, 2);
  assert.equal(consultas[1].cursor, `${anteriores.at(-1).comunicacion_ref}#${"c".repeat(64)}`);
  assert.match(raiz.innerHTML, /Llamamiento 11 de 11.*Pendiente de aviso/u);
  assert.match(raiz.innerHTML, /data-ct-llamamiento-form="comunicacion"/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form="seleccion"/u);
  cierre();
});

test("duplicar el llamamiento pendiente en otra página cierra la consulta completa", async () => {
  const raiz = raizPrueba(); let selecciones = 0;
  const anteriores = Array.from({ length: 9 }, (_, i) => ({ ...anterior(),
    comunicacion_ref: `comunicacion:previa:${i + 1}`,
    llamamiento_ref: `llamamiento:previo:${i + 1}`,
    registrada_en: `2026-09-24T10:0${i}:00Z` }));
  const cierre = montar(raiz, { consultarComunicacionesExpediente: async (consulta) => consulta.cursor
    ? { expediente_ref: EXPEDIENTE, comunicaciones: [pendiente("uno", "2026-09-24T10:11:00Z")] }
    : { expediente_ref: EXPEDIENTE, comunicaciones: [...anteriores, pendiente()],
      siguiente_cursor: `${pendiente().llamamiento_ref}#${"d".repeat(64)}` },
  seleccionarLlamamiento: async () => { selecciones += 1; } }, { contexto });
  await esperar();
  assert.match(raiz.innerHTML, /No se han podido cargar los llamamientos/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-form=|data-ct-llamamiento-recibo="seleccion"/u);
  await raiz.enviar("seleccion", {});
  assert.equal(selecciones, 0);
  cierre();
});
