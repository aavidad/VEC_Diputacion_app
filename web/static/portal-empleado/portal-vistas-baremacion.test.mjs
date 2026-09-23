import assert from "node:assert/strict";
import test from "node:test";
import { crearVistasBaremacion } from "./portal-vistas-baremacion.js";
import { crearTraductorBaremacion, traducirBaremacion } from "./portal-i18n-baremacion.js";
import { crearUtilidadesVista } from "./portal-vistas-utilidades.js";

const escaparHTML = (dato) => String(dato ?? "").replace(/[&<>"]/g,
  (caracter) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" })[caracter]);
const vistas = crearVistasBaremacion({
  escaparHTML,
  numero: (valor) => String(valor),
  fecha: (valor) => String(valor),
  chip: (valor) => `<span>${escaparHTML(valor)}</span>`,
  tabla: ({ filas }) => `<table>${filas.flat().join("")}</table>`,
  kpi: (_sigla, valor, etiqueta) => `<span>${escaparHTML(etiqueta)}:${escaparHTML(valor)}</span>`,
  encabezadoVista: (_seccion, titulo, descripcion, accion = "") =>
    `<h1>${escaparHTML(titulo)}</h1><p>${escaparHTML(descripcion)}</p>${accion}`,
  campo: (_etiqueta, control) => control,
});
const disponible = (vista) => ({ vistasBaremacion: { [vista]: { carga: "disponible", origen: "api_autorizada" } } });
const datos = {
  meritos_revision: [{ id: "MER-1", tipo: "Formación", declarado: "Curso", estado_dato: "declarado",
    evidencia: "DOC-1", puntos: "99", estado: "Aceptado" }],
  criterios_baremo: [{ bloque: "Formación", version: "v2" }],
  ranking: [{ posicion: 1, persona_ref: "PER-1", solicitud_ref: "SOL-3",
    convocatoria_ref: "CON-3", version_reglas: "v2", experiencia: "1,0", formacion: "2,0",
    otros: "0", total: "3,0", desempate: "Según bases", estado: "Provisional" }],
  alegaciones: [{ id: "ALE-1", persona_ref: "PER-1", objeto: "Curso", registrada: "Ayer",
    plazo: "Dentro de plazo", evidencia: "DOC-1", estado: "Pendiente" }],
};

test("sin fuente montada las tres vistas ignoran incluso un payload poblado", () => {
  for (const renderizar of [vistas.renderizarMeritos, vistas.renderizarBaremacion,
    vistas.renderizarAlegaciones]) {
    const html = renderizar(datos);
    assert.match(html, /Fuente no configurada/);
    assert.doesNotMatch(html, /MER-1|PER-1|ALE-1|99|3,0/);
    assert.match(html, /<summary aria-label="Ayuda:/);
  }
});

test("carga, error, denegación y origen no autorizado cierran los datos", () => {
  for (const [vista, renderizar] of [
    ["meritos", vistas.renderizarMeritos], ["baremacion", vistas.renderizarBaremacion],
    ["alegaciones", vistas.renderizarAlegaciones],
  ]) {
    for (const [carga, etiqueta] of [
      ["cargando", "Consultando"], ["error", "Servicio no disponible"],
      ["denegado", "Acceso denegado"], ["no_configurado", "Fuente no configurada"],
    ]) {
      const html = renderizar(datos, { vistasBaremacion: { [vista]: { carga } } });
      assert.match(html, new RegExp(etiqueta));
      assert.doesNotMatch(html, /MER-1|PER-1|ALE-1/);
    }
    assert.match(renderizar(datos, { vistasBaremacion: { [vista]: { carga: "disponible" } } }),
      /Fuente no configurada/);
  }
});

test("una consulta disponible y vacía no fabrica filas ni resultados", () => {
  assert.match(vistas.renderizarMeritos({}, disponible("meritos")), /Sin registros/);
  assert.match(vistas.renderizarBaremacion({}, disponible("baremacion")), /Sin registros/);
  assert.match(vistas.renderizarAlegaciones({}, disponible("alegaciones")), /Sin registros/);
});

test("un dato declarado no cumple requisitos ni hereda puntos o decisiones ajenas", () => {
  const html = vistas.renderizarMeritos(datos, disponible("meritos"));
  assert.match(html, /Curso/);
  assert.match(html, /Declarado/);
  assert.match(html, /Sin fuente verificable/);
  assert.match(html, /Sin evaluación de acceso estructurada/);
  assert.match(html, /Sin valoración de esta convocatoria/);
  assert.match(html, /Revisión administrativa no conectada/);
  assert.doesNotMatch(html, /99|Cumple requisito|Acreditado según fuente/);
  for (const comando of ["aceptar-merito", "rechazar-merito", "revocar-merito", "rehabilitar-merito"]) {
    assert.match(html, new RegExp(`data-comando="${comando}"[^>]+disabled aria-disabled="true"`));
  }
});

test("acreditación, acceso, valoración y revisión exigen sus fuentes propias", () => {
  const item = { id: "MER-2", tipo: "Titulación", declarado: "Título", estado_dato: "acreditado",
    fuente: "Registro", evidencia: "DOC-2", vigencia: "2026",
    evaluacion_acceso: { resultado: "cumple", convocatoria_ref: "CON-1", requisito_ref: "REQ-1",
      fuente: "Bases", version_bases: "v4", motivo: "Título exigido" },
    valoracion: { puntos: "2,5", solicitud_ref: "SOL-1", convocatoria_ref: "CON-1",
      version_reglas: "v4", fuente_calculo: "CAL-1" },
    revision: { estado: "Validado", motivo: "Documento comprobado", fuente: "Registro",
      version_criterio: "v4" } };
  const html = vistas.renderizarMeritos({ meritos_revision: [item] }, disponible("meritos"));
  assert.match(html, /Acreditado según fuente/);
  assert.match(html, /Cumple requisito/);
  assert.match(html, /Título exigido · Bases/);
  assert.match(html, /CON-1 · REQ-1 · v4/);
  assert.match(html, /2,5/);
  assert.match(html, /SOL-1 · CON-1 · v4/);
  assert.match(html, /CAL-1/);
  assert.match(html, /Documento comprobado · Registro · v4/);
  item.evaluacion_acceso.fuente = "";
  item.valoracion.fuente_calculo = "";
  item.revision.motivo = "";
  const incompleto = vistas.renderizarMeritos({ meritos_revision: [item] }, disponible("meritos"));
  assert.match(incompleto, /Sin evaluación de acceso estructurada/);
  assert.match(incompleto, /Sin valoración de esta convocatoria/);
  assert.match(incompleto, /Revisión administrativa no conectada/);
});

test("una titulación futura no habilita una bolsa y necesita previsión admitida por OPE", () => {
  const item = { id: "MER-3", tipo: "Titulación", declarado: "En curso",
    evaluacion_acceso: { resultado: "cumplimiento_previsto", convocatoria_ref: "CON-2",
      requisito_ref: "REQ-2", fuente: "Bases", version_bases: "v1", motivo: "Previsto",
      hito: "Prueba", tipo_proceso: "bolsa", permite_inscripcion_pendiente: true } };
  assert.match(vistas.renderizarMeritos({ meritos_revision: [item] }, disponible("meritos")),
    /Sin evaluación de acceso estructurada/);
  item.evaluacion_acceso.tipo_proceso = "ope";
  const ope = vistas.renderizarMeritos({ meritos_revision: [item] }, disponible("meritos"));
  assert.match(ope, /Cumplimiento previsto; sujeto a las bases y al hito/);
  assert.doesNotMatch(ope, /Cumple requisito/);
});

test("el ranking exige contexto de solicitud, bases y fuente, sin referencia fabricada", () => {
  assert.match(vistas.renderizarBaremacion(datos, disponible("baremacion")), /Fuente no configurada/);
  const completo = { ...datos, contexto_baremacion: { convocatoria_ref: "CON-3",
    entrada_ref: "ENT-3", version_reglas: "v2", fuente_calculo: "CAL-3",
    estado_salida: "Provisional" } };
  const html = vistas.renderizarBaremacion(completo, disponible("baremacion"));
  assert.match(html, /ENT-3 · CAL-3/);
  assert.match(html, /CON-3/);
  assert.match(html, /SOL-3/);
  assert.match(html, /3,0/);
  assert.match(html, /data-comando="publicar-lista-provisional"[^>]+disabled/);
  assert.match(html, /data-vista="baremacion" aria-current="page"/);
  completo.ranking[0].version_reglas = "v1";
  assert.match(vistas.renderizarBaremacion(completo, disponible("baremacion")), /Fuente no configurada/);
  completo.ranking[0].version_reglas = "v2";
  completo.contexto_baremacion.estado_salida = "Definitivo";
  assert.match(vistas.renderizarBaremacion(completo, disponible("baremacion")), /Fuente no configurada/);
  Object.assign(completo.contexto_baremacion, {
    lista_ref: "LIST-3", acta_ref: "ACT-3", firma_ref: "FIR-3", publicacion_ref: "PUB-3",
  });
  assert.match(vistas.renderizarBaremacion(completo, disponible("baremacion")), /Definitivo/);
});

test("registro y plazo de alegación se ocultan si faltan referencias acreditantes", () => {
  const html = vistas.renderizarAlegaciones(datos, disponible("alegaciones"));
  assert.match(html, /Sin registro acreditado/);
  assert.match(html, /Sin plazo acreditado/);
  assert.match(html, /Estado sin verificar/);
  assert.match(html, /Sin resolución acreditada/);
  assert.doesNotMatch(html, /Ayer|Dentro de plazo/);
  assert.doesNotMatch(html, /data-comando="resolver-alegacion"/);
});

test("la consulta disponible ofrece navegación y tablas enfocables con utilidades reales", () => {
  const u = crearUtilidadesVista({
    escaparHTML, numero: String, claseEstado: () => "neutro",
    encabezadoVista: (_seccion, titulo, descripcion, accion = "") =>
      `<h2>${escaparHTML(titulo)}</h2><p>${escaparHTML(descripcion)}</p>${accion}`,
    esPresentacion: () => false, operacionPermitida: () => false,
  });
  const reales = crearVistasBaremacion(u);
  const ranking = reales.renderizarBaremacion({
    ...datos, contexto_baremacion: { convocatoria_ref: "CON-3", entrada_ref: "ENT-3",
      version_reglas: "v2", fuente_calculo: "CAL-3", estado_salida: "Provisional" },
  }, disponible("baremacion"));
  assert.match(ranking, /<nav aria-label="Recorrido de baremación">/);
  assert.match(ranking, /data-vista="meritos"/);
  assert.match(ranking, /data-vista="alegaciones"/);
  assert.match(ranking, /tabindex="0" role="region"/);
  assert.match(ranking, /data-tabla-prioritaria="estado"/);
  assert.match(ranking, /data-columna="total"/);
  const alegaciones = reales.renderizarAlegaciones(datos, disponible("alegaciones"));
  assert.match(alegaciones, /data-tabla-prioritaria="estado"/);
});

test("textos localizados y datos escapados", () => {
  assert.equal(traducirBaremacion("meritos_encontrados", { numero: "3" }), "3 méritos encontrados.");
  assert.throws(() => traducirBaremacion("desconocida"), /desconocida/);
  assert.throws(() => crearTraductorBaremacion({}), /incompleto/);
  const html = vistas.renderizarMeritos({ meritos_revision: [{ id: "<script>", tipo: "Formación",
    declarado: "<img>", estado_dato: "declarado" }] }, disponible("meritos"));
  assert.match(html, /&lt;script&gt;/);
  assert.match(html, /&lt;img&gt;/);
  assert.doesNotMatch(html, /<script>|<img>/);
});
