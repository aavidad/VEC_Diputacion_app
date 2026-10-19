import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteHTTPContratacionTemporal } from "./cliente-http.js";
import { esComprobacionOfrecimientoPendiente } from "./cliente-http-llamamiento.js";
import { raizPrueba, montar, recibo } from "./formulario-llamamiento-pruebas.js";

const solicitud = { expediente_ref: "expediente:ct:sintetico:001", version_esperada: 6,
  clave_idempotencia: "123e4567-e89b-42d3-a456-426614174000" };
const detalle = () => ({ bolsa_ref: "bolsa:sintetica:001", orden_acta: 7,
  version_politica_ofertas: 3, motivo_clave: "pendiente_comprobacion_rrhh" });
const envelope = (comprobacion = detalle()) => ({ error: {
  codigo: "comprobacion_ofrecimiento_pendiente",
  clave_i18n: "api.contratacion_temporal.seleccion_llamamiento.error.comprobacion_ofrecimiento_pendiente",
  correlacion_ref: "corr_0123456789abcdef0123456789abcdef",
}, comprobacion_ofrecimiento: comprobacion });
const respuesta = (datos) => new Response(JSON.stringify(datos), {
  status: 409, headers: { "content-type": "application/json; charset=utf-8" },
});

test("rechazo tipado anterior a reserva conserva posición y regla, sin resultado incierto", async () => {
  const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async () => respuesta(envelope()) });
  await assert.rejects(cliente.seleccionarLlamamiento(solicitud), (error) => {
    assert.equal(esComprobacionOfrecimientoPendiente(error), true);
    assert.equal(error.resultadoIndeterminado, false);
    assert.deepEqual(error.comprobacionOfrecimiento, detalle());
    return true;
  });
});

test("RRHH ve la posición, versión y enlace a la bolsa sin referencias opacas; reintenta con la misma clave", async () => {
  const raiz = raizPrueba(); const claves = [];
  const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async (_ruta, opciones) => {
    claves.push(JSON.parse(opciones.body).clave_idempotencia);
    return respuesta(envelope());
  } });
  const cerrar = montar(raiz, { seleccionarLlamamiento: cliente.seleccionarLlamamiento });
  await raiz.enviar("seleccion", {});
  assert.match(raiz.innerHTML, /La persona en la posición 7 está pendiente de comprobación por RRHH/u);
  assert.match(raiz.innerHTML, /Regla consultada: versión 3/u);
  assert.ok(raiz.innerHTML.includes('href="/portal-empleado/?bolsa_ref=bolsa%3Asintetica%3A001'));
  assert.doesNotMatch(raiz.innerHTML, /resultado es incierto|corr_|bolsa:sintetica:001/u);
  await raiz.enviar("seleccion", {});
  assert.equal(claves.length, 2);
  assert.equal(claves[0], claves[1]);
  cerrar();
});

test("un detalle no fiable no se muestra ni se clasifica como rechazo seguro", async () => {
  const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async () => respuesta(envelope({
    ...detalle(), orden_acta: "<script>alert(1)</script>",
  })) });
  await assert.rejects(cliente.seleccionarLlamamiento(solicitud), (error) => {
    assert.equal(esComprobacionOfrecimientoPendiente(error), false);
    assert.equal(error.codigo, "respuesta_error_no_valida");
    assert.equal(error.resultadoIndeterminado, true);
    assert.equal(error.comprobacionOfrecimiento, null);
    return true;
  });
});

test("un recibo posterior retira el aviso pendiente", async () => {
  const raiz = raizPrueba(); let intentos = 0;
  const cerrar = montar(raiz, { seleccionarLlamamiento: async () => {
    intentos += 1;
    if (intentos === 1) throw Object.assign(new Error("pendiente"), {
      estado: 409, codigo: "comprobacion_ofrecimiento_pendiente", envelopeValido: true,
      resultadoIndeterminado: false, comprobacionOfrecimiento: detalle(),
    });
    return recibo;
  } });
  await raiz.enviar("seleccion", {});
  assert.match(raiz.innerHTML, /La persona en la posición 7/u);
  await raiz.enviar("seleccion", {});
  assert.doesNotMatch(raiz.innerHTML, /La persona en la posición 7/u);
  assert.match(raiz.innerHTML, /Persona llamada/u);
  cerrar();
});

const contexto = () => ({ ambito_ref: "ambito:sintetico:001", bolsa_ref: "bolsa:sintetica:001",
  categoria_ref: "categoria:sintetica:001", causa_ofrecida: "sustitucion",
  expediente_ref: "expediente:ct:sintetico:001", huella_politica_ofertas_sha256: "a".repeat(64),
  modalidad_ofrecida: "interinidad", necesidad_ref: "necesidad:sintetica:001", orden_acta: 7,
  organizacion_ref: "organizacion:sintetica:001", participacion_ref: "participacion:sintetica:007",
  unidad_ref: "unidad:sintetica:001", version_bolsa: 4, version_expediente: 6,
  version_instantanea: 2, version_politica_ofertas: 3 });
const estado = (cambios = {}) => ({ version_esperada: 0, comprobacion_ref: "", recibo_ref: "",
  registrada_en: "0001-01-01T00:00:00Z", version_politica_ofertas: 0, resultado: "", motivo_clave: "",
  vigente: null, ...cambios });
const completo = (cambios = {}) => ({ ...detalle(), contexto: contexto(), estado: estado(), ...cambios });

test("el aviso con contexto y estado del corte se acepta como rechazo seguro", async () => {
  for (const comprobacion of [completo(), completo({ estado: estado({ version_esperada: 2,
    comprobacion_ref: "comprobacion:sintetica:001", recibo_ref: "recibo:sintetico:001",
    registrada_en: "2026-10-19T10:00:00Z", version_politica_ofertas: 3, resultado: "no_cumple",
    motivo_clave: "requisito_no_acreditado", vigente: true }) })]) {
    const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async () => respuesta(envelope(comprobacion)) });
    await assert.rejects(cliente.seleccionarLlamamiento(solicitud), (error) => {
      assert.equal(esComprobacionOfrecimientoPendiente(error), true);
      assert.deepEqual(error.comprobacionOfrecimiento, comprobacion);
      assert.ok(Object.isFrozen(error.comprobacionOfrecimiento.contexto));
      return true;
    });
  }
});

test("un contexto o estado con campos de más o mal formados no se acepta", async () => {
  const malos = [
    completo({ contexto: { ...contexto(), nombre: "Persona" } }),
    completo({ contexto: { ...contexto(), orden_acta: 0 } }),
    completo({ contexto: { ...contexto(), huella_politica_ofertas_sha256: "z" } }),
    completo({ estado: { ...estado(), dni: "00000000T" } }),
    completo({ estado: estado({ version_esperada: 1 }) }),
    completo({ estado: estado({ vigente: "si" }) }),
  ];
  for (const comprobacion of malos) {
    const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async () => respuesta(envelope(comprobacion)) });
    await assert.rejects(cliente.seleccionarLlamamiento(solicitud), (error) => {
      assert.equal(esComprobacionOfrecimientoPendiente(error), false);
      assert.equal(error.codigo, "respuesta_error_no_valida");
      return true;
    });
  }
});
