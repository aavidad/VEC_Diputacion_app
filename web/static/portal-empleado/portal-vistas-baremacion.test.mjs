import assert from "node:assert/strict";
import test from "node:test";
import { crearVistasBaremacion } from "./portal-vistas-baremacion.js";
import { crearTraductorBaremacion, traducirBaremacion } from "./portal-i18n-baremacion.js";

const vistas = crearVistasBaremacion({
  escaparHTML: (valor) => String(valor),
  numero: (valor) => String(valor),
  chip: (valor) => `<span>${valor}</span>`,
  tabla: ({ filas }) => `<table>${filas.join("")}</table>`,
  kpi: (_sigla, valor) => `<span>${valor}</span>`,
  encabezadoVista: (_seccion, titulo, descripcion, accion = "") => `<h1>${titulo}</h1><p>${descripcion}</p>${accion}`,
  avisoPresentacion: (texto = "") => `<aside>${texto}</aside>`,
  botonOperacion: () => "<button>operación</button>",
  campo: (_etiqueta, control) => control,
  fuentePresentacion: () => "",
});

const datos = {
  meritos_revision: [{ id: "DEMO-MER-1", persona_ref: "DEMO-PER-1", tipo: "Formación", evidencia: "DEMO-DOC-1", declarado: "Dato", puntos: "1", estado: "Pendiente" }],
  criterios_baremo: [{ bloque: "Formación", version: "v-demo" }],
  ranking: [{ posicion: 1, persona_ref: "DEMO-PER-1", experiencia: "0", formacion: "1", otros: "0", total: "1", desempate: "No aplicado", estado: "Provisional" }],
  alegaciones: [{ id: "DEMO-ALE-1", persona_ref: "DEMO-PER-1", objeto: "Dato", registrada: "—", plazo: "—", evidencia: "DEMO-DOC-1", estado: "Pendiente" }],
};

test("las vistas de baremación cierran carga, error y denegación sin exponer datos", () => {
  for (const [vista, renderizar] of [
    ["meritos", vistas.renderizarMeritos],
    ["baremacion", vistas.renderizarBaremacion],
    ["alegaciones", vistas.renderizarAlegaciones],
  ]) {
    for (const [carga, texto] of [
      ["cargando", "Comprobando el acceso y cargando datos"],
      ["error", "No se pudo consultar esta información"],
      ["denegado", "La sesión no dispone de acceso"],
    ]) {
      const html = renderizar(datos, { vistasBaremacion: { [vista]: { carga, error: "Fallo controlado" } } });
      assert.match(html, new RegExp(texto));
      assert.doesNotMatch(html, /DEMO-PER-1/);
    }
  }
});

test("la presentación no habilita efectos y comunica que el ranking es sintético", () => {
  const html = vistas.renderizarBaremacion(datos);
  assert.match(html, /sintéticos/i);
  assert.match(html, /disabled aria-disabled="true"/);
  assert.doesNotMatch(html, /data-accion=/);
  assert.match(vistas.renderizarMeritos({}), /Sin registros/);
});

test("la motivación y las decisiones permanecen bloqueadas incluso en la demostración", () => {
  const presentacion = vistas.renderizarMeritos({ ...datos, demostracion: true });
  assert.match(presentacion, /<fieldset disabled aria-disabled="true">/);
  assert.match(presentacion, /La revisión con fuente, autorización y firma no está conectada/);
  for (const comando of ["aceptar-merito", "rechazar-merito", "revocar-merito", "rehabilitar-merito"]) {
    assert.match(presentacion, new RegExp(`data-comando="${comando}"[^>]+disabled aria-disabled="true"`));
  }
  assert.match(presentacion, /Sin evaluación de acceso estructurada/);
  assert.match(presentacion, /Declarado/);
  assert.match(presentacion, /Puntuación DEMO: 1/);

  const real = vistas.renderizarMeritos(datos);
  assert.match(real, /<fieldset disabled aria-disabled="true">/);
});

test("el estado declarado no se transforma en acreditación ni en requisito cumplido", () => {
  const html = vistas.renderizarMeritos({
    meritos_revision: [{ id: "MER-1", tipo: "Titulación", declarado: "Título declarado",
      evidencia: "DOC-1", estado: "Aceptado", puntos: "7" }],
  });
  assert.match(html, /Título declarado/);
  assert.match(html, /Sin fuente verificable/);
  assert.match(html, /Sin evaluación de acceso estructurada/);
  assert.match(html, /Puntuación DEMO: 7/);
  assert.doesNotMatch(html, /Cumple requisito|Acreditado según fuente/);
});

test("una valoración administrativa exige solicitud, convocatoria y versión; no hereda puntos DEMO", () => {
  const item = { id: "MER-2", tipo: "Formación", declarado: "Curso", estado_dato: "acreditado",
    fuente: "Fuente autorizada", evidencia: "DOC-2", vigencia: "2026",
    evaluacion_acceso: { resultado: "pendiente", motivo: "Texto libre" },
    puntos: "99", valoracion: { puntos: "2,5", solicitud_ref: "SOL-1", convocatoria_ref: "CON-1", version_reglas: "v2", fuente_calculo: "CAL-1" } };
  const html = vistas.renderizarMeritos({ meritos_revision: [item] },
    { vistasBaremacion: { meritos: { carga: "disponible", origen: "api_autorizada" } } });
  assert.match(html, /Acreditado según fuente/);
  assert.match(html, /Sin evaluación de acceso estructurada/);
  assert.match(html, /2,5/);
  assert.match(html, /CON-1 · v2/);
  assert.doesNotMatch(html, /99|Cumple requisito/);
});

test("una revisión solo muestra decisión cuando incluye motivo, fuente y versión", () => {
  const item = { id: "MER-4", tipo: "Experiencia", declarado: "Un periodo",
    revision: { estado: "Aceptado" } };
  const estado = { vistasBaremacion: { meritos: { carga: "disponible", origen: "api_autorizada" } } };
  assert.match(vistas.renderizarMeritos({ meritos_revision: [item] }, estado), /Revisión administrativa no conectada/);
  item.revision = { estado: "Aceptado", motivo: "Periodo comprobado", fuente: "Registro de personal", version_criterio: "v3" };
  const html = vistas.renderizarMeritos({ meritos_revision: [item] }, estado);
  assert.match(html, /Periodo comprobado · Registro de personal · v3/);
});

test("no configurado, vacío y denegado no filtran datos, y la ayuda funciona con teclado nativo", () => {
  const noConfigurado = vistas.renderizarBaremacion(datos, {
    vistasBaremacion: { baremacion: { carga: "no_configurado" } },
  });
  assert.match(noConfigurado, /Fuente no configurada/);
  assert.doesNotMatch(noConfigurado, /DEMO-PER-1/);
  assert.match(vistas.renderizarAlegaciones({}), /Sin registros/);
  const html = vistas.renderizarBaremacion(datos);
  assert.match(html, /<details class="baremacion-ayuda"><summary aria-label="Ayuda:/);
  assert.match(html, /data-comando="publicar-lista-provisional"[^>]+disabled/);
});

test("acceso requiere motivo, procedencia y bases; una previsión exige hito", () => {
  const item = { id: "MER-3", tipo: "Titulación", declarado: "Título en curso",
    evaluacion_acceso: { resultado: "cumplimiento_previsto", convocatoria_ref: "CON-1",
      requisito_ref: "REQ-1", fuente: "Bases", version_bases: "v4", motivo: "Previsto" } };
  const pendiente = vistas.renderizarMeritos({ meritos_revision: [item] });
  assert.match(pendiente, /Sin evaluación de acceso estructurada/);
  item.evaluacion_acceso.hito = "Prueba";
  item.evaluacion_acceso.tipo_proceso = "bolsa";
  item.evaluacion_acceso.permite_inscripcion_pendiente = true;
  assert.match(vistas.renderizarMeritos({ meritos_revision: [item] }), /Sin evaluación de acceso estructurada/);
  item.evaluacion_acceso.tipo_proceso = "ope";
  const previsto = vistas.renderizarMeritos({ meritos_revision: [item] });
  assert.match(previsto, /Cumplimiento previsto; sujeto a las bases y al hito/);
  assert.match(previsto, /Previsto · Bases · v4/);
  assert.doesNotMatch(previsto, /Cumple requisito/);
});

test("catálogo S3–S7 traduce variables y rechaza claves o traducciones ausentes", () => {
  assert.equal(traducirBaremacion("meritos_encontrados", { numero: "3" }), "3 méritos encontrados.");
  assert.throws(() => traducirBaremacion("desconocida"), /desconocida/);
  assert.throws(() => crearTraductorBaremacion({}), /incompleto/);
});
