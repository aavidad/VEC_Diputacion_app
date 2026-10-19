import assert from "node:assert/strict";
import test from "node:test";
import { renderizarSiguientePasoFicha } from "./vista-expedientes-ficha.js";
import { crearTraductorExpedientesContratacion, cargarMensajesExpedientesContratacionEnIdioma } from "./i18n-expedientes.js";

const expediente = {
  expediente_ref: "expediente:ct:orientacion", version: 5, tareas: [],
  fases: [{ etiqueta: "Gestión de bolsa", estado_clave: "en_curso" }],
  historial: [{ accion_clave: "contratacion_temporal.informe_juridico.generar", version_expediente: 5 }],
};
const estado = {
  expediente_ref: expediente.expediente_ref, carga: "listo",
  cuadro: { expedientes: [{ expediente_ref: expediente.expediente_ref,
    version: 5, fase_clave: "informe_juridico", estado_clave: "en_curso" }] },
};

test("el informe registrado orienta a revisar sus firmas e identifica quién sigue, sin afirmar que ya están", () => {
  const antes = JSON.stringify({ expediente, estado });
  const html = renderizarSiguientePasoFicha(expediente, estado, crearTraductorExpedientesContratacion());
  assert.match(html, /Siguiente paso: revisar las firmas del informe/u);
  assert.match(html, /Con ambas firmas verificadas, Intervención puede registrar la fiscalización/u);
  assert.match(html, /Técnica de Recursos Humanos y Jefatura; después, Intervención/u);
  assert.doesNotMatch(html, /Siguiente paso: Gestión de bolsa|data-ct-exp-accion="ir-tramite"/u);
  assert.equal(JSON.stringify({ expediente, estado }), antes);
});

test("un informe antiguo o una consulta pendiente no anuncia el siguiente paso como confirmado", () => {
  for (const actual of [
    { ...estado, ocupado: true }, { ...estado, actualizacion_pendiente: true },
    { ...estado, resultado_indeterminado: true },
    { ...estado, cuadro: { expedientes: [{ ...estado.cuadro.expedientes[0], version: 4 }] } },
  ]) assert.doesNotMatch(renderizarSiguientePasoFicha(expediente, actual,
    crearTraductorExpedientesContratacion()), /Con ambas firmas verificadas/u);
  assert.doesNotMatch(renderizarSiguientePasoFicha({ ...expediente, historial: [] }, estado,
    crearTraductorExpedientesContratacion()), /Con ambas firmas verificadas/u);
});

test("la orientación del informe se muestra solo en el idioma activo", async () => {
  const t = crearTraductorExpedientesContratacion(await cargarMensajesExpedientesContratacionEnIdioma("en"));
  const html = renderizarSiguientePasoFicha(expediente, estado, t);
  assert.match(html, /Next step: review the report signatures/u);
  assert.match(html, /Human Resources officer and Head of Service; then Financial Control/u);
  assert.doesNotMatch(html, /Siguiente paso|Técnica|Intervención/u);
});
