import assert from "node:assert/strict";
import test from "node:test";
import { raizPrueba, montar, EXPEDIENTE } from "./formulario-llamamiento-pruebas.js?v=VEC_WEB_VERSION";

const llamamiento = `llamamiento:${"a".repeat(64)}`;
const persona = { participacion_ref: "participacion:portal:001", respuesta: "acepta", modo: "firme",
  recibo_respuesta_ref: "recibo:respuesta:portal:001", respondida_en: "2026-10-20T09:05:00Z",
  justificante_ref: null, contacto_resultado: null, recibo_contacto_ref: null, contacto_en: null,
  situacion_actual: "disponible", recibo_situacion_ref: null, situacion_desde: null,
  origen_ct_confirmado: true, resolucion_portal_disponible: true };
const resultado_bolsa = { vinculos: [{ bolsa_ref: "bolsa:portal:001", llamamiento_ref: llamamiento,
  recibo_emision_ref: `recibo:${llamamiento}`, recibo_vinculo_ref: "recibo:vinculo:portal:001",
  vinculado_en: "2026-10-20T09:00:00Z", emitido_en: "2026-10-20T09:00:00Z",
  participaciones: [persona] }], emisiones_vinculables: [], siguiente_cursor: null,
  total_vinculos: 1, personas_solicitadas: 1, aceptaciones_firmes: 1 };

test("desmontar durante el POST aborta y la respuesta tardía no repinta ni anuncia", async () => {
  const raiz = raizPrueba(), anuncios = [];
  let html = raiz.innerHTML, pintadas = 0, abortada = false, llamadas = 0, rechazar;
  Object.defineProperty(raiz, "innerHTML", { get: () => html,
    set(valor) { html = valor; pintadas += 1; } });
  const pendiente = new Promise((_resolver, rechazo) => { rechazar = rechazo; });
  const cerrar = montar(raiz, { resolverLlamamiento: async (_solicitud, { signal }) => {
    llamadas += 1;
    signal.addEventListener("abort", () => { abortada = true; }, { once: true });
    return pendiente;
  } }, { contexto: { expediente_ref: EXPEDIENTE, version_esperada: 6,
    numero_personas: 1, resultado_bolsa },
  generarClaveIdempotencia: () => "123e4567-e89b-42d3-a456-426614174001",
  anunciar: (...datos) => anuncios.push(datos) });
  const formulario = { dataset: { ctPortalResolucion: "0" },
    elements: { namedItem: () => ({ checked: true }) },
    closest(selector) { return selector === "[data-ct-portal-resolucion]" ? this : null; } };
  const envio = raiz.eventos.get("submit")({ target: formulario, preventDefault() {} });
  assert.equal(llamadas, 1);
  cerrar();
  const pintadasTrasDesmontar = pintadas;
  assert.equal(abortada, true);
  assert.equal(html, "");
  rechazar(new DOMException("cancelada", "AbortError"));
  await envio;
  assert.equal(pintadas, pintadasTrasDesmontar);
  assert.equal(html, "");
  assert.deepEqual(raiz.foco, []);
  assert.deepEqual(anuncios, []);
  assert.equal(raiz.eventos.has("submit"), false);
  assert.equal(llamadas, 1);
});
