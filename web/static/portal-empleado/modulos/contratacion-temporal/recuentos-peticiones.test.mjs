import assert from "node:assert/strict";
import test from "node:test";
import { diaConsulta, filtrarPeticiones, filtroListaValido, resumirPeticiones } from "./recuentos-peticiones.js?v=20261001-ct-a-i18n-v1";

const e = (numero, fase_clave, estado_clave, extra = {}) => ({
  expediente_ref: `expediente:ct:${numero}`, numero_visible: `2026/CT-${numero}`, centro: "Servicio de Deportes",
  categoria: "Auxiliar", fase_clave, estado_clave, ...extra,
});
const lista = [
  e("0001", "solicitud", "en_curso", { plazo_estado: "en_plazo", plazo_ultimo_dia: "2026-10-01" }),
  e("0002", "subsanacion_unidad", "incidencia"),
  e("0003", "analisis", "pendiente", { plazo_estado: "vencido", plazo_ultimo_dia: "2026-09-25", centro: "Bienestar Social" }),
  e("0004", "fiscalizacion", "espera", { plazo_estado: "vence_hoy", plazo_ultimo_dia: "2026-09-29" }),
  e("0005", "seguimiento", "completado"),
];

test("portada y lista cuentan igual: en trámite, atención, semana y fases", () => {
  const resumen = resumirPeticiones({ expedientes: lista, generadoEn: "2026-09-29T07:00:00Z" });
  assert.equal(resumen.enTramite, 4);
  assert.equal(resumen.vencidos, 1);
  assert.deepEqual(resumen.atencion.map((x) => x.numero_visible), ["2026/CT-0003", "2026/CT-0004", "2026/CT-0002"]);
  assert.equal(resumen.vencenSemana, 2);
  assert.equal(resumen.porFase.fiscalizacion, 2);
  assert.equal(resumen.porFase.seguimiento, 0);
  assert.equal(resumirPeticiones({ expedientes: lista }).vencenSemana, null, "sin fecha de consulta no se inventa");
});

test("el día de la lectura es el de Madrid al cruzar la medianoche local", () => {
  assert.equal(diaConsulta("2026-09-29T22:30:00Z"), "2026-09-30");
  assert.equal(diaConsulta("sin fecha"), "");
});

test("los filtros de la lista se combinan, ignoran tildes y ordenan por plazo", () => {
  const f = (entrada) => filtrarPeticiones(lista, filtroListaValido(entrada), "2026-09-29T07:00:00Z").map((x) => x.numero_visible);
  assert.deepEqual(f({}), ["2026/CT-0003", "2026/CT-0004", "2026/CT-0001", "2026/CT-0002"]);
  assert.deepEqual(f({ fase: "fiscalizacion" }), ["2026/CT-0004", "2026/CT-0002"]);
  assert.deepEqual(f({ texto: "bienestar" }), ["2026/CT-0003"]);
  assert.deepEqual(f({ texto: "DEPORTES", mostrar: "espera" }), ["2026/CT-0004"]);
  assert.deepEqual(f({ mostrar: "terminadas" }), ["2026/CT-0005"]);
  assert.deepEqual(f({ mostrar: "vencen_semana" }), ["2026/CT-0004", "2026/CT-0001"]);
  assert.deepEqual(f({ mostrar: "vencidos" }), ["2026/CT-0003"]);
  assert.equal(f({ mostrar: "vencidos" }).length, resumirPeticiones({ expedientes: lista }).vencidos);
  assert.deepEqual(filtroListaValido({ fase: "inventada", mostrar: "x", ajeno: "y" }),
    { texto: "", fase: "", centro: "", categoria: "", mostrar: "en_tramite" });
});
