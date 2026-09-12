import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteAnotacionAdministrativaHTTP, RUTA_ANOTACION_ADMINISTRATIVA, RUTA_RECUPERACION_ANOTACION_ADMINISTRATIVA } from "./cliente-http-anotacion-administrativa.js";
import { montarFormularioAnotacionAdministrativa } from "./formulario-anotacion-administrativa.js";

const expediente = { expediente_ref: "expediente:anotacion:1", version: 7 };
const clave = "11111111-1111-4111-8111-111111111111";
const solicitud = { expediente_ref: expediente.expediente_ref, version_esperada: 7, clave_idempotencia: clave, observaciones: "Se deja constancia sintética." };
const respuesta = { operacion: "registrar_anotacion_administrativa", organizacion_ref: "organizacion:1", expediente_ref: expediente.expediente_ref, version_anterior: 7, version_resultante: 8, fase_resultante: "nombramiento", estado_resultante: "en_curso", seguimiento_original: { seguimiento_ref: "seguimiento:original:1", version_seguimiento: 1, huella_raiz_seguimiento_sha256: "a".repeat(64) }, recibo_ref: "recibo:anotacion:1", auditoria_ref: "auditoria:1", evento_ref: "evento:1", actor_ref: "actor:1", registrada_en: "2026-09-10T12:00:00Z" };
function raizFormulario() { const eventos = new Map(); return { innerHTML: "", addEventListener: (k, f) => eventos.set(k, f), removeEventListener: (k) => eventos.delete(k), replaceChildren() { this.innerHTML = ""; }, enviar(observaciones = solicitud.observaciones) { return eventos.get("submit")?.({ preventDefault() {}, target: { elements: { observaciones: { value: observaciones } } } }); } }; }
function montar(cliente) { const raiz = raizFormulario(); const desmontar = montarFormularioAnotacionAdministrativa({ raiz, cliente, expediente, confirmarOperacion: () => true, generarClaveIdempotencia: () => clave }); return { raiz, desmontar }; }

test("POST estricto contiene sólo la intención y acepta exclusivamente recibo validado", async () => {
  const llamadas = []; const ejecutar = async (configuracion) => { llamadas.push(configuracion); return respuesta; };
  const cliente = crearClienteAnotacionAdministrativaHTTP({ ejecutar, validarOpciones: ({ signal } = {}) => ({ signal }) });
  const x = montar(cliente); await x.raiz.enviar();
  assert.deepEqual(llamadas[0].entrada, solicitud); assert.equal(llamadas[0].ruta, RUTA_ANOTACION_ADMINISTRATIVA); assert.equal(llamadas[0].estadoEsperado, 201);
  assert.match(x.raiz.innerHTML, /recibo:anotacion:1/u); assert.match(x.raiz.innerHTML, /conserva la fase y el estado/u); x.desmontar();
});
test("resultado incierto recupera por la misma clave sin otro POST", async () => {
  const llamadas = []; const x = montar({ async registrar(s) { llamadas.push(["POST", s]); throw new Error("red"); }, async recuperar(s) { llamadas.push(["GET", s]); return respuesta; } });
  await x.raiz.enviar(); assert.deepEqual(llamadas, [["POST", solicitud], ["GET", { expediente_ref: expediente.expediente_ref, clave_idempotencia: clave }]]); assert.match(x.raiz.innerHTML, /recibo:anotacion:1/u); x.desmontar();
});
test("409 queda explícito y no realiza recuperación ni repetición automática", async () => {
  const llamadas = []; const x = montar({ async registrar() { llamadas.push("POST"); throw Object.assign(new Error(), { estado: 409 }); }, recuperar() { llamadas.push("GET"); } });
  await x.raiz.enviar(); await x.raiz.enviar(); assert.deepEqual(llamadas, ["POST"]); assert.match(x.raiz.innerHTML, /conflicto con el estado actual/u); x.desmontar();
});
test("GET codifica expediente y clave; desmontaje aborta respuesta tardía", async () => {
  const llamadas = []; const cliente = crearClienteAnotacionAdministrativaHTTP({ ejecutar: async (c) => { llamadas.push(c); return respuesta; }, validarOpciones: ({ signal } = {}) => ({ signal }) });
  await cliente.recuperar({ expediente_ref: expediente.expediente_ref, clave_idempotencia: clave }, { version_observada: 9 }); assert.equal(llamadas[0].ruta, `${RUTA_RECUPERACION_ANOTACION_ADMINISTRATIVA}?expediente_ref=expediente%3Aanotacion%3A1&clave_idempotencia=${clave}`);
  let resolver, signal; const x = montar({ registrar(_s, opciones) { signal = opciones.signal; return new Promise((r) => { resolver = r; }); }, recuperar() { assert.fail("GET inesperado"); } }); const vuelo = x.raiz.enviar(); await new Promise((r) => setImmediate(r)); x.desmontar(); assert.equal(signal.aborted, true); resolver(respuesta); await vuelo; assert.equal(x.raiz.innerHTML, "");
});
test("alConfirmar recibe sólo recibo validado tras POST, recuperación y tolera su error", async () => {
  const vistos = []; const raiz = raizFormulario(); montarFormularioAnotacionAdministrativa({ raiz, cliente: { async registrar() { return respuesta; }, recuperar() { assert.fail(); } }, expediente, confirmarOperacion: () => true, generarClaveIdempotencia: () => clave, alConfirmar: (r) => vistos.push(r.recibo_ref) }); await raiz.enviar(); assert.deepEqual(vistos, ["recibo:anotacion:1"]);
  const r2 = raizFormulario(); let llamadas = 0; montarFormularioAnotacionAdministrativa({ raiz: r2, cliente: { async registrar() { throw new Error(); }, async recuperar() { return respuesta; } }, expediente, confirmarOperacion: () => true, generarClaveIdempotencia: () => clave, alConfirmar() { llamadas++; throw new Error("externo"); } }); await r2.enviar(); assert.equal(llamadas, 1); assert.match(r2.innerHTML, /recibo:anotacion:1/);
});
