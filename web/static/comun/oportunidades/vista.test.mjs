import assert from "node:assert/strict";
import test from "node:test";
import { montarVistaOportunidades, renderizarOportunidades } from "./vista.js";

const oportunidad = (cambios = {}) => ({
  identificador_publico: "bolsa-auxiliar-2026",
  titulo: "Bolsa de Auxiliar",
  categoria: "Administración",
  estado_plazo: "abierto",
  fuente_plazo: "Bases publicadas",
  plazo_etiqueta: "Plazo abierto según las bases",
  estado_solicitud: "no_presentada",
  fuente_evaluacion: "Evaluación autorizada",
  evaluacion_autorizada: true,
  version_bases: "v2",
  requisitos: [{ etiqueta: "Titulación", estructurado: true, temporal: true, hito_cumplimiento: "fin_plazo", estado: "cumple", motivo: "Título acreditado vigente", procedencia: "Expediente de méritos" }],
  ...cambios,
});

test("una evaluación completa muestra el resultado, sus pruebas y el detalle público canónico", () => {
  const html = renderizarOportunidades({ estado: "disponible", oportunidades: [oportunidad()] });
  assert.match(html, /data-estado="cumple"/);
  assert.match(html, /Título acreditado vigente/);
  assert.match(html, /Expediente de méritos/);
  assert.match(html, /Al finalizar el plazo/);
  assert.match(html, /class="oportunidades-requisitos" open/);
  assert.match(html, /href="\/bolsa\/\?convocatoria=bolsa-auxiliar-2026"/);
  assert.match(html, /Iniciar solicitud precompletada<\/button>/);
  assert.match(html, /disabled aria-disabled="true"/);
});

test("texto libre, falta de fuente, versión, motivo o procedencia impiden afirmar cumplimiento", () => {
  for (const cambio of [
    { requisitos: [{ etiqueta: "Requisito textual", estado: "cumple", motivo: "supuesto", procedencia: "méritos" }] },
    { fuente_evaluacion: "" },
    { evaluacion_autorizada: false },
    { version_bases: "" },
    { requisitos: [{ etiqueta: "Titulación", estructurado: true, temporal: true, estado: "cumple", motivo: "sí", procedencia: "méritos" }] },
    { requisitos: [{ etiqueta: "Titulación", estructurado: true, temporal: true, hito_cumplimiento: "fecha_explicita", fecha_hito: "2026-02-30", estado: "cumple", motivo: "sí", procedencia: "méritos" }] },
    { requisitos: [{ etiqueta: "Titulación", estructurado: true, estado: "no_cumple", motivo: "", procedencia: "méritos" }] },
    { requisitos: [{ etiqueta: "Titulación", estructurado: true, estado: "cumple", motivo: "sí", procedencia: "" }] },
  ]) {
    const html = renderizarOportunidades({ estado: "disponible", oportunidades: [oportunidad(cambio)] });
    assert.match(html, /data-estado="pendiente"/);
    assert.doesNotMatch(html, /oportunidades-estado--cumple/);
    assert.doesNotMatch(html, /oportunidades-estado--no_cumple/);
  }
});

test("no deduce plazo, solicitud ni datos personales sin fuente", () => {
  const sinFuente = renderizarOportunidades({ estado: "disponible", oportunidades: [oportunidad({ estado_plazo: "cerrado", fuente_plazo: "", estado_solicitud: "presentada", fuente_solicitud: "", plazo_etiqueta: "ayer" })] });
  assert.match(sinFuente, /Situación del plazo sin fuente verificable/);
  assert.match(sinFuente, /Plazo no disponible/);
  assert.doesNotMatch(sinFuente, /ayer|Ya existe una solicitud|Plazo cerrado/);
  assert.match(renderizarOportunidades({ estado: "disponible", oportunidades: [oportunidad({ estado_plazo: "cerrado" })] }), /Plazo cerrado/);
  assert.match(renderizarOportunidades({ estado: "disponible", oportunidades: [oportunidad({ estado_solicitud: "presentada", fuente_solicitud: "Registro autorizado" })] }), /Ya existe una solicitud/);
});

test("un incumplimiento estructurado y explicado queda visible sin abrir una solicitud", () => {
  const html = renderizarOportunidades({ estado: "disponible", oportunidades: [oportunidad({ requisitos: [{ etiqueta: "Título exigido", estructurado: true, temporal: true, hito_cumplimiento: "incorporacion", estado: "no_cumple", motivo: "No consta el título exigido en el hito fijado por las bases", procedencia: "Evaluación de acceso" }] })] });
  assert.match(html, /data-estado="no_cumple"/);
  assert.match(html, /No consta el título exigido/);
  assert.match(html, /En la incorporación/);
  assert.match(html, /No cumple/);
  assert.match(html, /disabled aria-disabled="true"/);
});

test("la fecha explícita se localiza y el cumplimiento previsto nunca se muestra acreditado", () => {
  const requisito = { etiqueta: "Título previsto", estructurado: true, temporal: true, hito_cumplimiento: "fecha_explicita", fecha_hito: "2027-06-15", cumplimiento_previsto: true, estado: "cumple", motivo: "Previsión declarada", procedencia: "Expediente de méritos" };
  const html = renderizarOportunidades({ estado: "disponible", oportunidades: [oportunidad({ requisitos: [requisito] })] });
  assert.match(html, /15\/06\/2027/);
  assert.match(html, /previsión de cumplimiento no acredita/);
  assert.match(html, /data-estado="pendiente"/);
  assert.doesNotMatch(html, /oportunidades-estado--cumple/);
});

test("escapa campos de proyección e identificador y distingue estados de consulta", () => {
  const html = renderizarOportunidades({ estado: "disponible", oportunidades: [oportunidad({ titulo: '<img src=x onerror="alert(1)">', identificador_publico: 'a&b"<', requisitos: [{ etiqueta: "<script>", estructurado: false, estado: "cumple" }] })] });
  assert.doesNotMatch(html, /<img|<script>/);
  assert.match(html, /convocatoria=a%26b%22%3C/);
  for (const estado of ["cargando", "vacio", "no_configurado", "denegado", "error"]) {
    const salida = renderizarOportunidades({ estado, oportunidades: [oportunidad()] });
    assert.match(salida, new RegExp(`data-estado="${estado}"`));
    assert.doesNotMatch(salida, /Bolsa de Auxiliar/);
  }
});

test("el desmontaje cancela una consulta y evita pintar una respuesta tardía", async () => {
  const eventos = new Map();
  const raiz = { innerHTML: "", addEventListener: (nombre, fn) => eventos.set(nombre, fn), removeEventListener: (nombre) => eventos.delete(nombre), replaceChildren() { this.innerHTML = ""; } };
  let resolver; let signal;
  const carga = ({ signal: senal }) => { signal = senal; return new Promise((resolve) => { resolver = resolve; }); };
  const vista = montarVistaOportunidades({ raiz, cargar: carga });
  assert.match(raiz.innerHTML, /Consultando oportunidades/);
  vista.desmontar();
  resolver({ oportunidades: [oportunidad()] });
  await Promise.resolve();
  assert.equal(signal.aborted, true);
  assert.equal(raiz.innerHTML, "");
  assert.equal(eventos.size, 0);
});
