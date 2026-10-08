import assert from "node:assert/strict";
import test from "node:test";
import { crearCuadroContratacionTemporalPresentacion, crearExpedienteContratacionTemporalPresentacion } from "./datos-presentacion.js";
import { renderizarModuloContratacionTemporal } from "./vista-expedientes-render.js";
import { cargarMensajesExpedientesContratacionEnIdioma } from "./i18n-expedientes.js";

const base = crearExpedienteContratacionTemporalPresentacion();
const cabecera = [
  { clave: "centro", valor: "Residencia", etiqueta: "Centro" },
  { clave: "categoria", valor: "Auxiliar administrativo/a", etiqueta: "Categoría" },
  { clave: "fase", valor: "Fase 1 de 8: Solicitud", etiqueta: "Fase" },
  { clave: "estado", valor: "En trámite", etiqueta: "Estado" },
  { clave: "resultado_rc", valor: "Crédito comprobado", etiqueta: "Resultado del crédito" },
  { clave: "motivo", valor: "Corrección datos (ejercicio sintético)", etiqueta: "Motivo" },
];
const expediente = {
  ...base, demostracion: false, version: 2, cabecera,
  fases: base.fases.map((fase, indice) => ({ ...fase,
    estado_clave: indice === 0 ? "en_curso" : indice === 1 ? "completado" : "pendiente" })),
  tareas: [], historial: [{ secuencia: 2, fecha: "04/09/2026", accion: "Análisis de RRHH registrado",
    fase: "Solicitud", estado: "en trámite" }],
};
const cuadroBase = crearCuadroContratacionTemporalPresentacion();
const resumen = { ...cuadroBase.expedientes[0], expediente_ref: expediente.expediente_ref,
  version: 2, fase_clave: "solicitud", estado_clave: "en_curso", plazo_estado: "no_calculado" };
const estado = {
  vista: "expediente", carga: "listo", ocupado: false, actualizacion_pendiente: false,
  resultado_indeterminado: false, expediente_ref: expediente.expediente_ref, tarea_ref: "",
  expediente, cuadro: { ...cuadroBase, demostracion: false, expedientes: [resumen] },
};

test("el análisis registrado guía a la cobertura disponible sin cambiar la fase guardada", () => {
  const antes = JSON.stringify(estado);
  const html = renderizarModuloContratacionTemporal(estado, {
    analisisDisponible: true, coberturaDisponible: true,
  });
  assert.match(html, /Siguiente paso: decidir la vía de cobertura/u);
  assert.match(html, /El análisis está registrado\. Decida cómo cubrir la petición en el apartado de abajo\./u);
  assert.match(html, /data-ct-exp-cobertura/u);
  assert.doesNotMatch(html, /data-ct-exp-accion="ir-tramite"/u,
    "el botón genérico llevaría a rectificar el análisis");
  assert.match(html, /Fase 1 de 8: Solicitud/u);
  assert.match(html, /1 hechas · 1 ahora · 6 faltan/u);
  assert.match(html, /Corrección datos \(ejercicio sintético\)/u,
    "la historia y los valores de origen permanecen legibles");
  assert.equal(JSON.stringify(estado), antes);
});

test("sin capacidad, análisis confirmado o versión vigente no se anuncia cobertura", () => {
  const casos = [
    [estado, { coberturaDisponible: false }],
    [{ ...estado, expediente: { ...expediente, cabecera: cabecera.filter(({ clave }) => clave !== "resultado_rc") } },
      { coberturaDisponible: true }],
    [{ ...estado, cuadro: { ...estado.cuadro, expedientes: [{ ...resumen, version: 1 }] } },
      { coberturaDisponible: true }],
    [{ ...estado, expediente: { ...expediente, cabecera: [...cabecera,
      { clave: "via_cobertura", valor: "Bolsa", etiqueta: "Vía de cobertura" }] } },
      { coberturaDisponible: true }],
  ];
  for (const [actual, opciones] of casos) {
    const html = renderizarModuloContratacionTemporal(actual, opciones);
    assert.doesNotMatch(html, /Siguiente paso: decidir la vía de cobertura/u);
    assert.doesNotMatch(html, /data-ct-exp-cobertura/u);
    assert.match(html, /Siguiente paso: Solicitud/u);
  }
});

test("la guía inglesa usa el catálogo de la ficha", async () => {
  const mensajes = await cargarMensajesExpedientesContratacionEnIdioma("en");
  const html = renderizarModuloContratacionTemporal(estado, {
    mensajes, locale: "en-GB", analisisDisponible: true, coberturaDisponible: true,
  });
  assert.match(html, /Next step: choose the recruitment route/u);
  assert.match(html, /The analysis has been recorded\. Choose how to cover the request in the section below\./u);
  assert.doesNotMatch(html, /Siguiente paso: decidir/u);
});
