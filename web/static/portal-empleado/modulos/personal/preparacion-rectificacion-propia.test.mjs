import assert from "node:assert/strict";
import test from "node:test";
import { readFileSync } from "node:fs";
import { crearPreparacionRectificacionPropia } from "./preparacion-rectificacion-propia.js";

function datos() {
  return { historia: { corte: { efectos_desde: "2024-01-01", efectos_hasta: "2025-01-01", conocido_en: "2026-10-04T08:00:00.000000Z" }, cobertura: "parcial", revisiones: [{ servicio_ref: "srv_AAAAAAAAAAAAAAAAAAAAAA", relacion_ref: "rel_CCCCCCCCCCCCCCCCCCCCCC", periodo_desde: "2019-01-01", periodo_hasta: "2019-12-31", dias_reconocidos: 365, estado: "reconocido", clase: "Servicios previos", traza: { desde: "2024-01-01", registrada_en: "2024-03-01T09:00:00.000000Z", version: 2, acto_ref: "acto:uno", fuente_ref: "fuente:uno", fuente_version: 1 } }] }, consultada_en: "2026-10-04T08:00:01.000000Z", recibo_ref: "aud_v3_abcdef0123456789abcdef0123456789" };
}
const propuesta = { campo: "dias_reconocidos", propuesta: "366", motivo: "Periodo indicado en el certificado", evidencia: "Certificado de servicios" };

test("rechaza selector ajeno y campos no recibidos; limpiar invalida revisión y acceso al valor", () => {
  const d = datos(); assert.throws(() => crearPreparacionRectificacionPropia(d, { ...d.historia.revisiones[0] }), { codigo: "seleccion_no_valida" });
  const p = crearPreparacionRectificacionPropia(d, d.historia.revisiones[0]);
  assert.throws(() => p.preparar({ ...propuesta, campo: "empleado_ref" }), { codigo: "borrador_invalido" });
  p.preparar(propuesta); assert.equal(p.revisar(datos()).valor_actual, 365); p.limpiar();
  assert.throws(() => p.revisar(datos()), { codigo: "borrador_invalido" }); assert.throws(() => p.valor("clase"));
});

test("reconsulta comprueba revisión completa y fuente, no sólo que siga presente el identificador", () => {
  for (const cambio of [(d) => { d.historia.revisiones[0].traza.version++; }, (d) => { d.historia.revisiones[0].traza.fuente_version++; }, (d) => { d.historia.revisiones[0].traza.acto_ref = "acto:otro"; }, (d) => { d.historia.revisiones = []; }]) {
    const d = datos(), p = crearPreparacionRectificacionPropia(d, d.historia.revisiones[0]); p.preparar(propuesta);
    const nuevo = datos(); cambio(nuevo); assert.throws(() => p.revisar(nuevo), { codigo: "revision_sustituida" });
  }
});

test("admite prosa multilínea sin inventar reglas de rectificabilidad ni límites funcionales", () => {
  const d = datos(), p = crearPreparacionRectificacionPropia(d, d.historia.revisiones[0]);
  const motivo = "Primera línea\nSegunda línea\n" + "A".repeat(1500);
  p.preparar({ ...propuesta, motivo }); assert.equal(p.revisar(datos()).motivo, motivo);
});

test("catálogos completos, recuperación y estado de preparación en ambos idiomas", () => {
  const cargar = (idioma) => JSON.parse(readFileSync(new URL(`../../../textos/${idioma}/personal-preparacion-rectificacion.json`, import.meta.url), "utf8"));
  const claves = (c) => Object.entries(c).flatMap(([k, v]) => typeof v === "string" ? [k] : claves(v).map((x) => `${k}.${x}`)).sort();
  const es = cargar("es"), en = cargar("en"); assert.deepEqual(claves(es), claves(en));
  assert.equal(es.general.estado, "Preparación sin presentar"); assert.match(en.general.sin_presentar, /not been sent or registered/u);
});
