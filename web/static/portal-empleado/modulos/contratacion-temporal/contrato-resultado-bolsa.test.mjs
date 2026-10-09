import assert from "node:assert/strict";
import test from "node:test";
import { validarResultadoBolsaCT } from "./contrato-resultado-bolsa.js";

test("una renuncia de situación no se convierte en respuesta del llamamiento", () => {
  const llamamiento = `llamamiento:${"a".repeat(64)}`;
  const resultado = validarResultadoBolsaCT({
    vinculos: [{ bolsa_ref: "bolsa:prueba", llamamiento_ref: llamamiento,
      recibo_emision_ref: `recibo:${llamamiento}`, recibo_vinculo_ref: "recibo:ct:bolsa:prueba",
      vinculado_en: "2026-10-09T11:00:00Z", emitido_en: "2026-10-09T10:00:00Z",
      participaciones: [{ participacion_ref: "participacion:prueba", respuesta: null, modo: null,
        recibo_respuesta_ref: null, respondida_en: null, justificante_ref: null,
        contacto_resultado: "rechaza", recibo_contacto_ref: "recibo:contacto:prueba",
        contacto_en: "2026-10-09T10:30:00Z", situacion_actual: "renuncia",
        recibo_situacion_ref: "recibo:situacion:prueba", situacion_desde: "2026-10-09T10:45:00Z" }] }],
    emisiones_vinculables: [], siguiente_cursor: null,
  });
  assert.equal(resultado.vinculos[0].participaciones[0].respuesta, null);
  assert.equal(resultado.vinculos[0].participaciones[0].situacion_actual, "renuncia");
  assert.throws(() => validarResultadoBolsaCT({ ...resultado, vinculos: [{ ...resultado.vinculos[0],
    recibo_emision_ref: "recibo:ajeno" }] }));
});
