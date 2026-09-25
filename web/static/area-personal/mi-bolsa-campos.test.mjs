import assert from "node:assert/strict";
import test from "node:test";
import { validarRespuestaMiBolsa } from "./contrato.js";
import { CAMPOS_MI_BOLSA, validarCamposMiBolsa } from "./mi-bolsa-campos.js";
import { renderizarLlamamientos } from "./vistas/seguimiento-tramites.js";

const participacionCompleta = () => ({
  bolsa: "bolsa:prueba:01", categoria: "Auxiliar", version: 3, orden_inicial: 12, total_instantanea: 87,
  estado_bolsa: "vigente", vigente_desde: "2026-09-01T00:00:00.000Z", vigente_hasta: null,
  situacion_actual: { estado: "disponible_desde", desde: "2026-09-20T10:00:00.000Z", hasta: null, fecha_disponible: "2026-10-01T10:00:00.000Z" },
  ultimo_llamamiento: { emitido_en: "2026-09-21T10:00:00.000Z", canal: "correo", resultado: "enviado" },
});
const respuesta = (campos, participacion) => ({ data: {
  esquema: "vec.bolsa.mi-bolsa.v1", consultada_en: "2026-09-23T10:00:00.000Z", campos_visibles: campos, participaciones: [participacion],
} });

test("sin lista de campos se muestran todos, como antes del catálogo", () => {
  assert.deepEqual(validarCamposMiBolsa(undefined), [...CAMPOS_MI_BOLSA]);
  const datos = validarRespuestaMiBolsa(respuesta(undefined, participacionCompleta()));
  assert.deepEqual(datos.campos_visibles, [...CAMPOS_MI_BOLSA]);
  const html = renderizarLlamamientos({ posicion: null }, { participaciones: datos.participaciones, camposMiBolsa: datos.campos_visibles });
  assert.match(html, /12 de 87/u);
  assert.match(html, /Contratos/u);
  assert.match(html, /Último resultado de correo/u);
});

test("la web rechaza listas rotas y datos que el catálogo oculta", () => {
  for (const rota of [["posicion"], ["bolsa", "bolsa"], ["bolsa", "puntuacion"], "bolsa"]) {
    assert.throws(() => validarCamposMiBolsa(rota), /datos visibles/u);
  }
  assert.throws(() => validarRespuestaMiBolsa(respuesta(["bolsa", "posicion", "estado", "fecha_disponible", "contratos"], participacionCompleta())), /último llamamiento/u);
  assert.throws(() => validarRespuestaMiBolsa(respuesta(["bolsa"], participacionCompleta())), /oculta por el catálogo/u);
});

test("con el catálogo reducido solo se pintan los datos visibles", () => {
  const p = participacionCompleta();
  delete p.orden_inicial; delete p.total_instantanea; delete p.estado_bolsa;
  p.situacion_actual = { fecha_disponible: "2026-10-01T10:00:00.000Z" };
  p.ultimo_llamamiento = null;
  const datos = validarRespuestaMiBolsa(respuesta(["bolsa", "fecha_disponible"], p));
  const html = renderizarLlamamientos({ posicion: null }, { participaciones: datos.participaciones, camposMiBolsa: datos.campos_visibles });
  assert.match(html, /Auxiliar/u);
  assert.match(html, /Fecha de disponibilidad indicada/u);
  assert.doesNotMatch(html, /12 de 87|Estado de la bolsa|Último resultado de correo|>Contratos</u);
});
