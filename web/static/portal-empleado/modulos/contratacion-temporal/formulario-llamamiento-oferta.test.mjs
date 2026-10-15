import assert from "node:assert/strict";
import test from "node:test";
import { crearLlamamientoClienteHTTP, RUTAS_LLAMAMIENTO } from "./cliente-http-llamamiento.js";
import { ESQUEMA_RECIBO_SELECCION_OFERTA, validarReciboSeleccion, validarReciboSeleccionOferta } from "./contrato-llamamiento.js";
import { EXPEDIENTE, seleccion, raizPrueba, montar, recibo as reciboSingular } from "./formulario-llamamiento-pruebas.js?v=20261008-alta-rpt-circular-v6";

const HUELLA = "a".repeat(64);
const reciboOferta = Object.freeze({
  esquema: ESQUEMA_RECIBO_SELECCION_OFERTA, estado: "confirmado",
  organizacion_ref: "organizacion:sintetica:001", expediente_ref: EXPEDIENTE,
  bolsa_ref: "bolsa:sintetica:001", oferta_ref: `oferta:${HUELLA}`,
  recibo_oferta_ref: `recibo:oferta:${HUELLA}`, recibo_vinculo_ref: "recibo:vinculo:001",
  numero_plazas: 2, publicada_en: "2026-10-15T10:00:00Z", vence_antes_de: "2026-10-17T10:00:00Z",
});

test("el mismo POST acepta recibo de oferta vinculada y rechaza un recibo ajeno", async () => {
  const solicitud = { ...seleccion(), version_esperada: 6 };
  const comandos = [];
  const cliente = crearLlamamientoClienteHTTP({ validarOpciones: () => ({ signal: undefined }),
    ejecutar: async (comando) => { comandos.push(comando); return comando.validarRespuesta(reciboOferta); } });
  assert.deepEqual(await cliente.seleccionarLlamamiento(solicitud), reciboOferta);
  assert.equal(comandos[0].ruta, RUTAS_LLAMAMIENTO.seleccionLlamamiento);
  assert.deepEqual(Object.keys(comandos[0].entrada), ["expediente_ref", "version_esperada", "clave_idempotencia"]);
  assert.equal(validarReciboSeleccion(reciboSingular).esquema, "vec.contratacion-temporal.recibo-seleccion-llamamiento.v1");
  assert.throws(() => validarReciboSeleccionOferta({ ...reciboOferta, expediente_ref: "expediente:otro:001" }, solicitud));
  assert.throws(() => validarReciboSeleccionOferta({ ...reciboOferta, numero_plazas: 1 }, solicitud));
  assert.throws(() => validarReciboSeleccionOferta({ ...reciboOferta, aviso_enviado: true }, solicitud));
});

test("la oferta real muestra plazas, plazo y justificantes sin abrir comunicación singular", async () => {
  const raiz = raizPrueba();
  const solicitudes = [];
  const cerrar = montar(raiz, { seleccionarLlamamiento: async (solicitud) => {
    solicitudes.push(solicitud); return reciboOferta;
  } });
  assert.match(raiz.innerHTML, /Continuar con Bolsa/);
  await raiz.enviar("seleccion", seleccion());
  assert.equal(solicitudes.length, 1);
  assert.deepEqual(Object.keys(solicitudes[0]), ["expediente_ref", "version_esperada", "clave_idempotencia"]);
  assert.match(raiz.innerHTML, /Oferta publicada y vinculada al expediente/);
  assert.match(raiz.innerHTML, /Plazas publicadas<\/dt><dd>2/);
  assert.match(raiz.innerHTML, /recibo:oferta:/);
  assert.match(raiz.innerHTML, /recibo:vinculo:001/);
  assert.match(raiz.innerHTML, /ofertas_bolsa_ref=bolsa%3Asintetica%3A001#bolsa\/llamamientos/);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-llamamiento-comunicacion|data-ct-llamamiento-form="comunicacion"/);
  assert.match(raiz.innerHTML, /no acredita el envío de un correo/);
  cerrar();
});

test("tras fallo incierto recupera la oferta con la misma petición y el singular sigue abriendo comunicación", async () => {
  const raiz = raizPrueba();
  const solicitudes = [];
  const cerrar = montar(raiz, { seleccionarLlamamiento: async (solicitud) => {
    solicitudes.push(solicitud);
    if (solicitudes.length === 1) throw new Error("red interrumpida");
    return reciboOferta;
  } });
  await raiz.enviar("seleccion", seleccion());
  assert.match(raiz.innerHTML, /Volver a intentar/);
  await raiz.enviar("seleccion", { ...seleccion(), expediente_ref: "expediente:otro:001" });
  assert.deepEqual(solicitudes[0], solicitudes[1]);
  assert.match(raiz.innerHTML, /Oferta publicada y vinculada/);
  cerrar();

  const raizSingular = raizPrueba();
  const cerrarSingular = montar(raizSingular, { seleccionarLlamamiento: async () => reciboSingular });
  await raizSingular.enviar("seleccion", seleccion());
  assert.match(raizSingular.innerHTML, /data-ct-llamamiento-comunicacion open/);
  cerrarSingular();
});

test("503, 403 y 403 tras un intento incierto conservan cuerpo y UUID hasta recibo válido", async () => {
  for (const reciboFinal of [reciboOferta, reciboSingular]) {
    const raiz = raizPrueba();
    const solicitudes = [];
    const cerrar = montar(raiz, { seleccionarLlamamiento: async (solicitud) => {
      solicitudes.push(solicitud);
      if (solicitudes.length === 1) throw new Error("503 sin respuesta concluyente");
      if (solicitudes.length < 4) throw Object.assign(new Error("denegado"), {
        estado: 403, codigo: "acceso_denegado", envelopeValido: true, resultadoIndeterminado: false,
      });
      return reciboFinal;
    } });
    for (let intento = 0; intento < 4; intento++) {
      await raiz.enviar("seleccion", { ...seleccion(), expediente_ref: intento ? "expediente:otro:001" : EXPEDIENTE });
      if (intento < 3) assert.match(raiz.innerHTML, /Volver a intentar/u);
    }
    assert.equal(solicitudes.length, 4);
    solicitudes.forEach((solicitud) => assert.deepEqual(solicitud, solicitudes[0]));
    assert.equal(solicitudes[0].expediente_ref, EXPEDIENTE);
    assert.match(raiz.innerHTML, reciboFinal === reciboOferta ? /Oferta publicada y vinculada/ : /data-ct-llamamiento-comunicacion open/u);
    cerrar();
  }
});

test("401, 422 y 409 tras un intento incierto no liberan la selección original", async () => {
  const raiz = raizPrueba();
  const solicitudes = [];
  const rechazos = [[401, "autenticacion_requerida"], [422, "contenido_no_valido"], [409, "conflicto_no_reintentable"]];
  const cerrar = montar(raiz, { seleccionarLlamamiento: async (solicitud) => {
    solicitudes.push(solicitud);
    if (solicitudes.length === 1) throw new Error("red interrumpida");
    if (solicitudes.length <= 4) {
      const [estado, codigo] = rechazos[solicitudes.length - 2];
      throw Object.assign(new Error(codigo), { estado, codigo, envelopeValido: true, resultadoIndeterminado: false });
    }
    return reciboOferta;
  } });
  for (let intento = 0; intento < 5; intento++) {
    await raiz.enviar("seleccion", { ...seleccion(), expediente_ref: intento ? "expediente:otro:001" : EXPEDIENTE });
    if (intento < 4) assert.match(raiz.innerHTML, /Volver a intentar/u);
  }
  assert.equal(solicitudes.length, 5);
  solicitudes.forEach((solicitud) => assert.deepEqual(solicitud, solicitudes[0]));
  assert.match(raiz.innerHTML, /Oferta publicada y vinculada/);
  cerrar();
});
