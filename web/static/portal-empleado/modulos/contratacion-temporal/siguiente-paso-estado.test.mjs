import assert from "node:assert/strict";
import test from "node:test";
import { renderizarSiguientePasoFicha } from "./vista-expedientes-ficha.js";
import {
  crearTraductorExpedientesContratacion,
  cargarMensajesExpedientesContratacionEnIdioma,
} from "./i18n-expedientes.js?v=20261001-ct-a-i18n-v1";
const MENSAJES_EXPEDIENTES_CONTRATACION_EN = await cargarMensajesExpedientesContratacionEnIdioma("en");

const expediente = Object.freeze({
  expediente_ref: "expediente:sintetico:001",
  tareas: [{ estado_clave: "en_curso", etiqueta: "Análisis de la petición",
    responsable: "Recursos Humanos", unidad: "Recursos Humanos",
    acciones: [{ tipo: "efecto", disponible: true, etiqueta: "Revisar petición" }] }],
});
const listo = Object.freeze({
  carga: "listo", ocupado: false, actualizacion_pendiente: false, resultado_indeterminado: false,
  cuadro: { expedientes: [{ expediente_ref: expediente.expediente_ref,
    fase_clave: "analisis_rrhh", estado_clave: "en_curso" }] },
});
const boton = (html) => html.match(/<button\b[^>]*data-ct-exp-accion="ir-tramite"[^>]*>/u)?.[0];

for (const [idioma, mensajes] of [["es", {}], ["en", MENSAJES_EXPEDIENTES_CONTRATACION_EN]]) {
  const t = crearTraductorExpedientesContratacion(mensajes);
  for (const [nombre, cambio, mensaje] of [
    ["registrando", { ocupado: true }, "estado_registrando_actuacion"],
    ["actualización pendiente", { actualizacion_pendiente: true }, "estado_actualizacion_pendiente"],
    ["resultado sin confirmar", { resultado_indeterminado: true }, "estado_resultado_indeterminado"],
    ["cargando", { carga: "cargando" }, "estado_cargando_expediente"],
    ["error de consulta", { carga: "error" }, "estado_error_expediente"],
    ["acceso denegado", { carga: "denegado" }, "estado_denegado_expediente"],
    ["sin consulta inicial", { carga: "inicial" }, "estado_actualizacion_pendiente"],
    ["consulta vacía", { carga: "vacio" }, "estado_actualizacion_pendiente"],
  ]) {
    test(`${idioma}: el próximo trámite explica ${nombre} y no ofrece un acceso activo`, () => {
      const estado = { ...listo, ...cambio };
      const anterior = JSON.stringify({ expediente, estado });
      const html = renderizarSiguientePasoFicha(expediente, estado, t);
      assert.match(boton(html), / disabled aria-disabled="true"/u);
      assert.match(boton(html), /aria-describedby="ct-exp-siguiente-paso-estado"/u);
      assert.ok(html.includes(t(mensaje)));
      assert.match(html, /id="ct-exp-siguiente-paso-estado" role="status" aria-live="polite" aria-atomic="true"/u);
      assert.doesNotMatch(html, /<dd>Revisar petición<\/dd>|<form|data-ct-exp-efecto/u);
      assert.equal(JSON.stringify({ expediente, estado }), anterior);
    });
  }

  test(`${idioma}: vuelve a ofrecer la navegación cuando la consulta queda lista`, () => {
    const html = renderizarSiguientePasoFicha(expediente, listo, t);
    assert.ok(boton(html));
    assert.doesNotMatch(boton(html), /disabled|aria-describedby/u);
    assert.match(html, /<dd>Revisar petición<\/dd>/u);
    assert.match(html, /aria-atomic="true"><\/p>/u);
    assert.doesNotMatch(html, /data-ct-exp-efecto|<form/u);
  });

  for (const estadoClave of ["completado", "cancelado", "espera"]) {
    test(`${idioma}: ${estadoClave} conserva su título y no abre otro trámite`, () => {
      const estado = { ...listo, resultado_indeterminado: true,
        cuadro: { expedientes: [{ ...listo.cuadro.expedientes[0], estado_clave: estadoClave }] } };
      const html = renderizarSiguientePasoFicha(expediente, estado, t);
      assert.equal(boton(html), undefined);
      const titulo = t(estadoClave === "espera" ? "ficha_siguiente_paso_espera" : "ficha_siguiente_paso_terminado");
      assert.ok(html.includes(titulo));
      assert.ok(html.includes(t("estado_resultado_indeterminado")));
    });
  }
}

test("la falta de confirmación prevalece sobre el registro y los mensajes se escapan", () => {
  const estado = { ...listo, ocupado: true, actualizacion_pendiente: true, resultado_indeterminado: true };
  const t = crearTraductorExpedientesContratacion({ estado_resultado_indeterminado: "Revise <el recibo> & confirme" });
  const html = renderizarSiguientePasoFicha(expediente, estado, t);
  assert.match(html, /Revise &lt;el recibo&gt; &amp; confirme/u);
  assert.doesNotMatch(html, /<el recibo>|Registrando la actuación/u);
});

test("en fiscalización, a quien no es Intervención se le dice quién actúa y no se le ofrece trámite", async () => {
  const mensajes = { ficha_siguiente_paso_fiscalizacion_intervencion: "Intervención tiene que registrar el resultado de la fiscalización.",
    ficha_siguiente_paso_espera: "Esperando a otra unidad" };
  const t = crearTraductorExpedientesContratacion(mensajes);
  const enFiscalizacion = { ...listo, cuadro: { expedientes: [{ ...listo.cuadro.expedientes[0],
    fase_clave: "fiscalizacion", estado_clave: "espera_externa" }] } };
  const sinTareas = { ...expediente, tareas: [] };
  const ajeno = renderizarSiguientePasoFicha(sinTareas, enFiscalizacion, t, false, { fiscalizacionAjena: true });
  assert.match(ajeno, /Esperando a otra unidad/u);
  assert.match(ajeno, /Intervención tiene que registrar el resultado/u);
  assert.equal(boton(ajeno), undefined);
  const propio = renderizarSiguientePasoFicha(sinTareas, enFiscalizacion, t, false, { fiscalizacionAjena: false });
  assert.doesNotMatch(propio, /Intervención tiene que registrar/u);
});
