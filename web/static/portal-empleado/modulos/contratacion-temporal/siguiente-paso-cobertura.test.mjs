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

test("un cliente de fiscalización ausente no se confunde con un perfil ajeno", () => {
  const enFiscalizacion = { ...estado, cuadro: { ...estado.cuadro, expedientes: [
    { ...resumen, fase_clave: "fiscalizacion" },
  ] } };
  const sinCliente = renderizarModuloContratacionTemporal(enFiscalizacion, {
    fiscalizacionDisponible: false, fiscalizacionAjena: false,
  });
  assert.match(sinCliente, /data-ct-exp-accion="ir-tramite"/u);
  assert.doesNotMatch(sinCliente, /Pendiente de Intervención/u);
  const perfilAjeno = renderizarModuloContratacionTemporal(enFiscalizacion, {
    fiscalizacionDisponible: false, fiscalizacionAjena: true,
  });
  assert.doesNotMatch(perfilAjeno, /data-ct-exp-accion="ir-tramite"/u);
});

test("justo tras registrar el análisis, la cobertura aparece aunque el cuadro siga en la versión anterior", () => {
  const conHito = { ...expediente, historial: [{ ...expediente.historial[0],
    accion_clave: "contratacion_temporal.analisis.registrar", version_expediente: 2 }] };
  const cuadroAnterior = { ...estado.cuadro, expedientes: [{ ...resumen, version: 1 }] };
  const html = renderizarModuloContratacionTemporal({ ...estado, expediente: conHito, cuadro: cuadroAnterior },
    { analisisDisponible: true, coberturaDisponible: true });
  assert.match(html, /data-ct-exp-cobertura/u);
  // Otra actuación posterior al cuadro (no el análisis) no se da por buena.
  const otroHito = { ...conHito, historial: [{ ...conHito.historial[0], accion_clave: "contratacion_temporal.cobertura.decidir" }] };
  assert.doesNotMatch(renderizarModuloContratacionTemporal({ ...estado, expediente: otroHito, cuadro: cuadroAnterior },
    { analisisDisponible: true, coberturaDisponible: true }), /data-ct-exp-cobertura/u);
  // Ni un cuadro dos versiones por detrás.
  assert.doesNotMatch(renderizarModuloContratacionTemporal({ ...estado, expediente: { ...conHito, version: 3,
    historial: [{ ...conHito.historial[0], version_expediente: 3 }] }, cuadro: cuadroAnterior },
  { analisisDisponible: true, coberturaDisponible: true }), /data-ct-exp-cobertura/u);
});

const fiscalizacionVigente = {
  ...estado, expediente: {
    ...expediente, version: 6,
    cabecera: cabecera.map((campo) => campo.clave === "fase"
      ? { ...campo, valor: "Fase 4 de 8: Fiscalización" } : campo),
    fases: base.fases.map((fase, indice) => ({ ...fase,
      estado_clave: indice < 3 ? "completado" : indice === 3 ? "en_curso" : "pendiente" })),
    historial: [{ secuencia: 6, version_expediente: 6,
      accion_clave: "contratacion_temporal.fiscalizacion.registrar",
      fecha: "18/10/2026", accion: "Fiscalización registrada", fase: "Fiscalización", estado: "en trámite" }],
  },
  cuadro: { ...estado.cuadro, expedientes: [{ ...resumen, version: 6,
    fase_clave: "fiscalizacion", estado_clave: "en_curso" }] },
};
const conResultadoFiscalizacion = (resultado, cambios = {}) => ({
  ...fiscalizacionVigente, ...cambios,
  expediente: { ...fiscalizacionVigente.expediente,
    fiscalizacion: { resultado_clave: resultado, registrada_en: "18/10/2026", reparos: [] },
    ...(cambios.expediente ?? {}) },
});
const siguientePaso = (html) => html.match(/<section class="siguiente-paso[^]*?<\/section>/u)?.[0] ?? "";
const verPasoFiscalizacion = (actual, mensajes) => siguientePaso(renderizarModuloContratacionTemporal(actual,
  { fiscalizacionAjena: true, fiscalizacionDisponible: false, mensajes, locale: mensajes ? "en-GB" : "es-ES" }));

test("resultado favorable registrado orienta al llamamiento sin adelantar la fase ni abrir otro trámite", () => {
  for (const resultado of ["favorable", "favorable_con_observaciones"]) {
    const actual = conResultadoFiscalizacion(resultado);
    const antes = JSON.stringify(actual);
    const html = renderizarModuloContratacionTemporal(actual,
      { fiscalizacionAjena: true, fiscalizacionDisponible: false });
    const paso = siguientePaso(html);
    assert.match(paso, /Siguiente paso: llamamiento/u);
    assert.match(paso, /La fiscalización favorable está registrada\. Continúe en el apartado «Llamamiento»/u);
    assert.doesNotMatch(paso, /Intervención tiene que registrar|data-ct-exp-accion="ir-tramite"/u);
    assert.match(html, /Fase 4 de 8: Fiscalización/u);
    assert.match(html, /3 hechas · 1 ahora · 4 faltan/u);
    assert.equal(JSON.stringify(actual), antes);
  }
});

test("un hito, un resultado pendiente o un detalle sin fiscalización no acreditan aprobación", () => {
  for (const actual of [fiscalizacionVigente,
    conResultadoFiscalizacion("pendiente"),
    conResultadoFiscalizacion("favorable", { expediente: { fiscalizacion: undefined } })]) {
    const paso = verPasoFiscalizacion(actual);
    assert.match(paso, /Intervención tiene que registrar el resultado/u);
    assert.doesNotMatch(paso, /fiscalización favorable está registrada|data-ct-exp-accion="ir-tramite"/u);
  }
  const desfavorable = verPasoFiscalizacion(conResultadoFiscalizacion("desfavorable"));
  assert.doesNotMatch(desfavorable, /fiscalización favorable está registrada|Siguiente paso: llamamiento/u);
});

test("un resultado obsoleto o una ficha bloqueada no anuncia el llamamiento como siguiente paso", () => {
  const favorable = conResultadoFiscalizacion("favorable");
  const obsoletos = [
    { ...favorable, cuadro: { ...favorable.cuadro, expedientes: [
      { ...favorable.cuadro.expedientes[0], version: 5 }] } },
    { ...favorable, expediente: { ...favorable.expediente, fiscalizacion: {
      ...favorable.expediente.fiscalizacion, registrada_en: "" } } },
  ];
  for (const actual of [...obsoletos,
    ...[{ carga: "error" }, { ocupado: true }, { actualizacion_pendiente: true },
      { resultado_indeterminado: true }].map((cambio) => ({ ...favorable, ...cambio })),
    { ...favorable, cuadro: { ...favorable.cuadro, expedientes: [
      { ...favorable.cuadro.expedientes[0], estado_clave: "completado" }] } }]) {
    const paso = verPasoFiscalizacion(actual);
    assert.doesNotMatch(paso, /Siguiente paso: llamamiento|fiscalización favorable está registrada/u);
    assert.doesNotMatch(paso, /data-ct-exp-accion="ir-tramite"(?![^>]*disabled)/u);
  }
});

test("la orientación inglesa usa su catálogo y conserva la fase en curso", async () => {
  const mensajes = await cargarMensajesExpedientesContratacionEnIdioma("en");
  const html = renderizarModuloContratacionTemporal(conResultadoFiscalizacion("favorable"),
    { fiscalizacionAjena: true, fiscalizacionDisponible: false, mensajes, locale: "en-GB" });
  assert.match(siguientePaso(html), /Next step: call-up/u);
  assert.match(siguientePaso(html), /The favourable audit result has been recorded/u);
  assert.doesNotMatch(siguientePaso(html), /Siguiente paso|fiscalización favorable/u);
  assert.match(html, /3 done · 1 now · 4 to do/u);
});
