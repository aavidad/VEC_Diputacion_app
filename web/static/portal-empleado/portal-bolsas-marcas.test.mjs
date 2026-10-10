import test from "node:test";
import assert from "node:assert/strict";

import { renderizarChipsMarcas, renderizarMarcasFicha, seleccionableEnLlamamiento, traducirMarcasBolsa, validarMarcasCandidato } from "./portal-bolsas-marcas.js?v=20261001-ct-a-i18n-v1";
import { validarCandidato } from "./portal-bolsas-contrato.js?v=20261001-ct-a-i18n-v1";
import { consultarSeleccionMasivaBolsa, seleccionarParticipacionesPorEstado } from "./portal-bolsas-api.js?v=20261008-bolsa-global-v2";

const escapar = (valor) => String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;");
const marcasCompletas = { presta_servicios: "excluir", en_revision: "renuncia_pendiente", encadenamiento: { dias_acumulados: 578, umbral_meses: 18, ventana_meses: 24 } };
const candidatoBase = {
  participacion_ref: "participacion:001", orden: 1, orden_acta: 1, razon_orden: "orden_acta", nombre_visible: "Antonio Reyes Álvarez",
  documento_enmascarado: "***1234**", estado_clave: "disponible", estado_desde: "2026-09-20T10:00:00Z", disponible_desde: null,
  ultimo_llamamiento: null, contactos_total: 0,
};

test("valida el bloque cerrado de marcas", () => {
  assert.deepEqual(validarMarcasCandidato({ presta_servicios: null, en_revision: null, encadenamiento: null }), { presta_servicios: null, en_revision: null, encadenamiento: null });
  assert.equal(validarMarcasCandidato(marcasCompletas).encadenamiento.dias_acumulados, 578);
  for (const mala of [
    { ...marcasCompletas, presta_servicios: "bloquear" },
    { ...marcasCompletas, en_revision: "pendiente" },
    { ...marcasCompletas, encadenamiento: { dias_acumulados: 0, umbral_meses: 18, ventana_meses: 24 } },
    { ...marcasCompletas, extra: true },
    { presta_servicios: null, en_revision: null },
  ]) assert.throws(() => validarMarcasCandidato(mala));
});

test("el contrato de candidatos admite marcas solo completas y las conserva", () => {
  assert.equal(validarCandidato(candidatoBase).marcas, undefined);
  const conMarcas = validarCandidato({ ...candidatoBase, marcas: marcasCompletas });
  assert.equal(conMarcas.marcas.en_revision, "renuncia_pendiente");
  assert.throws(() => validarCandidato({ ...candidatoBase, marcas: { presta_servicios: "otro", en_revision: null, encadenamiento: null } }));
});

test("rotula en revisión, servicios y encadenamiento sin referencias internas", () => {
  const html = renderizarChipsMarcas({ marcas: marcasCompletas }, escapar);
  assert.match(html, /En revisión/);
  assert.match(html, /Ya presta servicios: no se puede llamar/);
  assert.match(html, /Encadenamiento/);
  assert.match(html, /578 días con contrato en los últimos 24 meses/);
  assert.equal(renderizarChipsMarcas({}, escapar), "");
  assert.equal(renderizarChipsMarcas({ marcas: { presta_servicios: null, en_revision: null, encadenamiento: null } }, escapar), "");
  const ficha = renderizarMarcasFicha({ marcas: { ...marcasCompletas, en_revision: "baja_propuesta" } }, escapar);
  assert.match(ficha, /Revisión pendiente[\s\S]*Baja propuesta por intentos sin contacto/);
  assert.match(ficha, /Encadenamiento de contratos/);
  assert.doesNotMatch(html + ficha, /b16|b17|000041|participacion:/);
  assert.throws(() => traducirMarcasBolsa("inexistente"));
});

test("una respuesta de Mi bolsa sin reflejar se rotula y deja a la persona sin turno", () => {
  for (const motivo of ["aceptacion_pendiente", "renuncia_pendiente"]) {
    assert.equal(validarMarcasCandidato({ presta_servicios: null, en_revision: motivo, encadenamiento: null }).en_revision, motivo);
  }
  const ficha = renderizarMarcasFicha({ marcas: { presta_servicios: null, en_revision: "aceptacion_pendiente", encadenamiento: null } }, escapar);
  assert.match(ficha, /Revisión pendiente[\s\S]*Ha aceptado en «Mi bolsa»\. Confírmelo con «Cambiar situación»/);
  const chips = renderizarChipsMarcas({ marcas: { presta_servicios: null, en_revision: "renuncia_pendiente", encadenamiento: null } }, escapar);
  assert.match(chips, /En revisión/);
  assert.match(chips, /Ha renunciado en «Mi bolsa»/);
  const fueraDeTurno = { ...candidatoBase, orden: null, razon_orden: "respuesta_portal_pendiente" };
  assert.equal(validarCandidato(fueraDeTurno).razon_orden, "respuesta_portal_pendiente");
  assert.throws(() => validarCandidato({ ...fueraDeTurno, orden: 1 }), /respuesta del portal pendiente con turno/);
  assert.equal(validarCandidato({ ...candidatoBase, orden: null, razon_orden: "cese_pendiente", estado_clave: "no_disponible" }).razon_orden, "cese_pendiente");
});

test("quien ya presta servicios en modo excluir no entra en la selección del llamamiento", async () => {
  assert.equal(seleccionableEnLlamamiento({ marcas: { presta_servicios: "aviso" } }), true);
  assert.equal(seleccionableEnLlamamiento({ marcas: { presta_servicios: "excluir" } }), false);
  assert.equal(seleccionableEnLlamamiento({}), true);
  const candidatos = [
    { ...candidatoBase, marcas: { ...marcasCompletas } },
    { ...candidatoBase, participacion_ref: "participacion:002", orden: 2, marcas: { presta_servicios: "aviso", en_revision: null, encadenamiento: null } },
  ];
  assert.deepEqual(seleccionarParticipacionesPorEstado(candidatos, ["disponible"]), ["participacion:002"]);
  const bolsa = { bolsa_ref: "bolsa:1", total: 2, por_estado: { disponible: 2 }, politica_orden: null };
  const resultado = await consultarSeleccionMasivaBolsa("bolsa:1", ["disponible"], {
    consultar: async () => ({ ok: true, datos: { generado_en: "2026-09-20T10:00:00Z", bolsa, candidatos, hay_mas: false, cursor_siguiente: null } }),
  });
  assert.equal(resultado.ok, true);
  assert.deepEqual(resultado.participaciones, ["participacion:002"]);
  assert.equal(resultado.total, 1);
});
