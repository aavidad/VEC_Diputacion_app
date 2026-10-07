import assert from "node:assert/strict";
import test from "node:test";
import { diaConsulta, filtrarPeticiones, filtroListaValido, filtrosCuadroRRHHV2,
  leerFiltroListaV2DesdeURL, escribirFiltroListaV2EnURL, resumirPeticiones } from "./recuentos-peticiones.js?v=20261001-ct-a-i18n-v1";

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

test("filtro v2 usa referencias, estados y solo fases administrativas acreditadas en el resumen", () => {
  const filtro = { texto: "auxiliar", centro: "centro:001", categoria: "categoria:auxiliar",
    fase: "solicitud", mostrar: "en_tramite" };
  const seleccion = filtrosCuadroRRHHV2(filtro,
    { solicitud_registrada: 2, solicitud: 1, analisis: 3, desconocida: 1 });
  assert.deepEqual(seleccion.filtros, {
    texto: "auxiliar", centro_ref: "centro:001", categoria_ref: "categoria:auxiliar",
    estados_clave: ["pendiente", "en_curso", "espera_externa", "incidencia"],
    fases_clave: ["solicitud", "solicitud_registrada"],
  });
  assert.equal(seleccion.sinCoincidenciasDeFase, false);
  assert.deepEqual(filtrosCuadroRRHHV2(filtro, {}, ["solicitud_registrada", "solicitud"])
    .filtros.fases_clave, ["solicitud", "solicitud_registrada"]);
  assert.throws(() => filtrosCuadroRRHHV2(filtro, {}, ["analisis"]));
  assert.equal(filtrosCuadroRRHHV2({ fase: "seguimiento" }, { solicitud: 1 }).sinCoincidenciasDeFase, true);
  assert.deepEqual(filtrosCuadroRRHHV2({ mostrar: "terminadas" }).filtros.estados_clave,
    ["completado", "cancelado"]);
  assert.deepEqual(filtrosCuadroRRHHV2({ mostrar: "espera" }).filtros.estados_clave, ["espera_externa"]);
  for (const mostrar of ["vencidos", "vence_hoy", "sin_plazo", "atencion", "vencen_semana"]) {
    assert.throws(() => filtrosCuadroRRHHV2({ mostrar }),
      (error) => error.codigo === "filtro_servidor_no_disponible");
  }
  for (const mostrar of ["terminadas", "todas"]) {
    assert.throws(() => filtrosCuadroRRHHV2({ mostrar, fase: "solicitud" }),
      (error) => error.codigo === "filtro_servidor_no_disponible");
  }
});

test("URL CT v2 conserva otros parámetros, canoniza filtros y rechaza duplicados", () => {
  const entrada = new URLSearchParams("lang=en&expediente=expediente%3Act%3A001&ct_centro=centro%3A001&ct_mostrar=incidencia");
  const leido = leerFiltroListaV2DesdeURL(entrada);
  assert.deepEqual(leido, { filtro: { texto: "", fase: "", centro: "centro:001", categoria: "", mostrar: "incidencia" },
    fasesClave: [] });
  const salida = escribirFiltroListaV2EnURL(entrada, { ...leido.filtro, texto: "auxiliar", fase: "solicitud" },
    ["solicitud_registrada", "solicitud"]);
  assert.equal(salida.get("lang"), "en");
  assert.equal(salida.get("expediente"), "expediente:ct:001");
  assert.deepEqual(leerFiltroListaV2DesdeURL(salida), { filtro: { ...leido.filtro, texto: "auxiliar", fase: "solicitud" },
    fasesClave: ["solicitud", "solicitud_registrada"] });
  assert.equal(escribirFiltroListaV2EnURL(salida, filtroListaValido({})).has("ct_mostrar"), false);
  for (const parametros of ["ct_texto=a&ct_texto=b", "ct_mostrar=vencido", "ct_desconocido=x",
    "ct_centro=%3Cscript%3E", "ct_fase=solicitud", "ct_fases=solicitud",
    "ct_fase=solicitud&ct_fases=analisis", "ct_fase=solicitud&ct_fases=solicitud&ct_fases=solicitud"]) {
    assert.throws(() => leerFiltroListaV2DesdeURL(new URLSearchParams(parametros)));
  }
});
