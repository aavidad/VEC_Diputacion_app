import assert from "node:assert/strict";
import test from "node:test";
import { filtrarPeticiones, filtroListaValido } from "./recuentos-peticiones.js";
import { renderizarListaPeticiones, renderizarResultadosLista } from "./vista-expedientes-lista.js";
import { crearTraductorExpedientesContratacion, MENSAJES_EXPEDIENTES_CONTRATACION_EN } from "./i18n-expedientes.js";

const lista = Object.freeze([
  Object.freeze({ expediente_ref: "expediente:ct:demo:1", numero_visible: "2026/CT-0001", centro: "centro:demo:1",
    categoria: "Administración", fase_clave: "analisis", fase_actual: "Análisis", estado_clave: "en_curso", estado: "En trámite",
    plazo_estado: "en_plazo", plazo_ultimo_dia: "2026-10-03", plazo: "03/10/2026" }),
  Object.freeze({ expediente_ref: "expediente:ct:demo:2", numero_visible: "2026/CT-aaaaaaaaaaaaaaaa", centro: "centro:demo:2",
    categoria: "Administración", fase_clave: "analisis", fase_actual: "Análisis", estado_clave: "en_curso", estado: "En trámite",
    plazo_estado: "en_plazo", plazo_ultimo_dia: "2026-10-02", plazo: "02/10/2026" }),
]);
const estado = Object.freeze({ cuadro: Object.freeze({ expedientes: lista, generado_en: "2026-10-01T08:00:00Z" }) });
const ayudas = (t) => ({
  numeroVisible: (numero) => numero === lista[1].numero_visible ? t("numero_expediente_sin_asignar") : numero,
  // Una etiqueta autorizada inyectada puede ser compartida por referencias distintas.
  centroVisible: (referencia) => ({ etiqueta: "Área técnica", referencia }),
});
const valores = (a) => (e) => [a.numeroVisible(e.numero_visible), a.centroVisible(e.centro).etiqueta];
const referenciasHTML = (html) => [...html.matchAll(/data-ct-exp-abrir="([^"]+)"/gu)].map((m) => m[1]);

test("la lista encuentra el centro mostrado aunque su etiqueta no aparezca en la referencia", () => {
  const t = crearTraductorExpedientesContratacion();
  const a = ayudas(t);
  const filtro = filtroListaValido({ texto: "AREA TECNICA" });
  const html = renderizarListaPeticiones(estado, t, filtro, a);
  assert.deepEqual(referenciasHTML(html), [lista[1].expediente_ref, lista[0].expediente_ref]);
  assert.ok(html.includes(">Área técnica<small>"), "el resultado utiliza la misma etiqueta que la fila");
  assert.ok(html.includes('value="centro:demo:1"'), "el selector conserva la referencia como valor");
  assert.equal(filtrarPeticiones(lista, filtro, estado.cuadro.generado_en).length, 0, "sin la proyección no existe ese nombre en el dato");
});

test("los números visibles de expedientes sin numerar se pueden buscar en ambos idiomas", () => {
  for (const mensajes of [{}, MENSAJES_EXPEDIENTES_CONTRATACION_EN]) {
    const t = crearTraductorExpedientesContratacion(mensajes);
    const a = ayudas(t);
    const filtro = filtroListaValido({ texto: t("numero_expediente_sin_asignar").toUpperCase() });
    const html = renderizarResultadosLista(estado, t, filtro, a);
    assert.deepEqual(referenciasHTML(html), [lista[1].expediente_ref]);
    assert.ok(html.includes(`>${t("numero_expediente_sin_asignar")}</button>`));
  }
});

test("centro y categoría exactos siguen restringiendo la búsqueda por etiqueta", () => {
  const t = crearTraductorExpedientesContratacion();
  const a = ayudas(t);
  const filtro = filtroListaValido({ texto: "área", centro: lista[0].centro, categoria: lista[0].categoria, fase: "analisis_rrhh" });
  assert.deepEqual(filtrarPeticiones(lista, filtro, estado.cuadro.generado_en, valores(a)), [lista[0]]);
  assert.deepEqual(filtrarPeticiones(lista, { ...filtro, centro: "Área técnica" }, estado.cuadro.generado_en, valores(a)), []);
  assert.deepEqual(filtrarPeticiones(lista, { ...filtro, categoria: "administracion" }, estado.cuadro.generado_en, valores(a)), []);
});

test("el proveedor añade etiquetas sin perder la búsqueda antigua ni cambiar la consulta", () => {
  const t = crearTraductorExpedientesContratacion();
  const a = ayudas(t);
  const antes = structuredClone(lista);
  for (const texto of [lista[0].centro, lista[0].numero_visible, "ADMINISTRACION"]) {
    const filtro = filtroListaValido({ texto });
    assert.deepEqual(filtrarPeticiones(lista, filtro, estado.cuadro.generado_en, valores(a)),
      filtrarPeticiones(lista, filtro, estado.cuadro.generado_en));
  }
  const filtro = filtroListaValido({ texto: "area tecnica" });
  const filtradas = filtrarPeticiones(lista, filtro, estado.cuadro.generado_en, valores(a));
  assert.equal(filtradas[0], lista[1]);
  assert.equal(filtradas[1], lista[0]);
  assert.deepEqual(lista, antes);
  assert.equal(filtro.texto, "area tecnica");
});

test("sin texto no se necesita calcular etiquetas y los filtros de tres argumentos conservan el orden", () => {
  const filtro = filtroListaValido({});
  const sinProveedor = filtrarPeticiones(lista, filtro, estado.cuadro.generado_en);
  assert.deepEqual(sinProveedor, [lista[1], lista[0]]);
  assert.deepEqual(filtrarPeticiones(lista, filtro, estado.cuadro.generado_en, () => {
    assert.fail("sin búsqueda no se solicita una etiqueta");
  }), sinProveedor);
});

test("con más páginas y un filtro puesto, avisa de que la búsqueda solo mira esta página", () => {
  const t = crearTraductorExpedientesContratacion();
  const a = ayudas(t);
  const conMas = { cuadro: { ...estado.cuadro, paginacion: { pagina: 1, cursor_siguiente: "c2" } } };
  const conFiltro = renderizarResultadosLista(conMas, t, filtroListaValido({ texto: "área" }), a);
  assert.match(conFiltro, /La búsqueda solo revisa las peticiones de esta página/u);
  const sinFiltro = renderizarResultadosLista(conMas, t, filtroListaValido({}), a);
  assert.doesNotMatch(sinFiltro, /data-ct-exp-busqueda-parcial/u);
  const unaPagina = renderizarResultadosLista(estado, t, filtroListaValido({ texto: "área" }), a);
  assert.doesNotMatch(unaPagina, /data-ct-exp-busqueda-parcial/u);
});
