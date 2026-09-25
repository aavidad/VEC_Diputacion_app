import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { renderizarCuadro, renderizarExpediente } from "./componentes-expedientes.js";
import {
  validarCuadroContratacionTemporal,
  validarExpedienteContratacionTemporal,
} from "./contrato-expedientes.js";
import {
  crearCuadroContratacionTemporalPresentacion,
  crearExpedienteContratacionTemporalPresentacion,
} from "./datos-presentacion.js";
import { renderizarModuloContratacionTemporal } from "./vista-expedientes.js";

function estadoVista(expediente, tareaRef = expediente.tareas[0].tarea_ref) {
  return {
    vista: "expediente",
    carga: "listo",
    cuadro: null,
    expediente,
    documentos: null,
    auditoria: null,
    expediente_ref: expediente.expediente_ref,
    tarea_ref: tareaRef,
    filtros: { texto: "", estado: "", fase: "" },
    ocupado: false,
    actualizacion_pendiente: false,
    recibo: null,
    mensaje_clave: "estado_expediente_listo",
    tipo_mensaje: "informacion",
  };
}

test("las tareas operativas cubren todos los hitos funcionales de RRHH", () => {
  const expediente = validarExpedienteContratacionTemporal(
    crearExpedienteContratacionTemporalPresentacion(),
  );
  const matriz = [
    ["tarea-solicitud", "Datos de la petición"],
    ["tarea-analisis", "Comprobación y validación"],
    ["tarea-cobertura", "Procedimiento a seguir"],
    ["tarea-asignacion", "Bandeja de la unidad"],
    ["tarea-informe-juridico", "Borrador y edición gobernada"],
    ["tarea-envio-intervencion", "Datos pendientes del circuito"],
    ["tarea-fiscalizacion", "Modalidad y remisión"],
    ["tarea-subsanacion", "Observaciones, correcciones y evidencias"],
    ["tarea-iniciar-llamamiento", "Historial de llamamientos"],
    ["tarea-seleccion-candidato", "Candidatura propuesta"],
    ["tarea-resultado-llamamiento", "Resumen e historial de la candidatura"],
    ["tarea-traslado-intervencion", "Preparación para formalización"],
    ["tarea-informe-definitivo", "Candidatura, observaciones e historial"],
    ["tarea-formalizacion", "Circuito Portafirmas P4 pendiente"],
    ["tarea-incorporacion", "Proyección autorizada para incorporación"],
    ["tarea-ginpix", "Historial GINPIX"],
    ["tarea-envio-ginpix", "Envío a GINPIX"],
    ["tarea-seguimiento", "Histórico de relación, prórroga y cese"],
  ];
  assert.equal(expediente.tareas.length, matriz.length);
  for (const [referencia, evidencia] of matriz) {
    const tarea = expediente.tareas.find(({ tarea_ref }) => tarea_ref === referencia);
    assert.ok(tarea, `falta ${referencia}`);
    assert.match(JSON.stringify(tarea), new RegExp(evidencia, "u"), referencia);
  }
  const cuadro = validarCuadroContratacionTemporal(
    crearCuadroContratacionTemporalPresentacion(),
  );
  const html = renderizarModuloContratacionTemporal({
    ...estadoVista(expediente),
    vista: "cuadro",
    cuadro,
    expediente: null,
    expediente_ref: "",
    tarea_ref: "",
  });
  assert.match(html, /Mis tareas prioritarias/);
  assert.match(html, /Distribución por fase/);
  assert.match(html, /Registrar nueva petición/);
});

test("la presentación delimita plazos, llamamientos y preparación previa sin inventar efectos", () => {
  const expediente = crearExpedienteContratacionTemporalPresentacion();
  const porReferencia = (referencia) => expediente.tareas.find(
    ({ tarea_ref }) => tarea_ref === referencia,
  );
  const texto = (referencia) => JSON.stringify(porReferencia(referencia));

  assert.match(texto("tarea-subsanacion"), /plazo por definir/u);
  assert.match(texto("tarea-iniciar-llamamiento"), /Escenario sintético/u);
  assert.match(texto("tarea-iniciar-llamamiento"), /Canal de comunicación/u);
  assert.doesNotMatch(texto("tarea-iniciar-llamamiento"), /Correo y teléfono|Renuncia acreditada/u);
  assert.match(texto("tarea-seleccion-candidato"), /Candidatura propuesta/u);
  assert.match(texto("tarea-seleccion-candidato"), /pendiente de validar/u);
  assert.match(texto("tarea-seleccion-candidato"), /no adjudica ni llama automáticamente/u);
  assert.doesNotMatch(texto("tarea-seleccion-candidato"), /Llamar a la primera candidatura/u);
  assert.match(texto("tarea-resultado-llamamiento"), /Respuesta manual sintética/u);
  assert.match(texto("tarea-resultado-llamamiento"), /plazo no evaluado/u);
  assert.doesNotMatch(texto("tarea-resultado-llamamiento"), /dentro de plazo|Entregado/u);

  const traslado = porReferencia("tarea-traslado-intervencion");
  assert.equal(traslado.etiqueta, "Preparación para formalización");
  assert.match(JSON.stringify(traslado), /no existe envío externo/u);
  assert.doesNotMatch(JSON.stringify(traslado), /Trasladado|Enviar a Intervención/u);
  assert.deepEqual(traslado.acciones, []);

  const cuadro = crearCuadroContratacionTemporalPresentacion();
  assert.ok(cuadro.expedientes.every(({ plazo }) => (
    ["Regla pendiente", "Plazo por definir", "Cerrado"].includes(plazo)
  )));
});

test("la formalización mantiene P4 pendiente sin inventar circuito ni efectos administrativos", () => {
  const expediente = crearExpedienteContratacionTemporalPresentacion();
  const tarea = expediente.tareas.find(({ tarea_ref }) => tarea_ref === "tarea-formalizacion");
  assert.ok(tarea);
  assert.equal(tarea.estado_clave, "en_curso");
  assert.equal(tarea.salida, "");
  assert.equal(tarea.recibo_ref, "");
  assert.equal(tarea.decision_ref, "");
  assert.equal(tarea.acciones.length, 1);
  assert.equal(tarea.acciones[0].accion_ref, "preparar_borrador_firma_demo");
  assert.equal(tarea.acciones[0].disponible, false);
  const texto = JSON.stringify(tarea);
  assert.match(texto, /Pendiente de definición por RRHH/u);
  assert.match(texto, /solo prepara una vista en memoria/i);
  assert.match(texto, /no envía, firma, registra ni genera un recibo administrativo/i);
  assert.doesNotMatch(texto, /Jefatura de Servicio|Órgano competente|fe pública|Enviada al portafirmas|Firmado|orden proceden/i);
});


test("las vistas neutras quedan aisladas de red, cookies, storage y adaptador sintético", async () => {
  const directorio = new URL("./", import.meta.url);
  const [datosFuente, presentadorFuente, vistaFuente, interno, produccion] =
    await Promise.all([
      readFile(new URL("datos-presentacion.js", directorio), "utf8"),
      readFile(new URL("presentador-expedientes.js", directorio), "utf8"),
      readFile(new URL("vista-expedientes.js", directorio), "utf8"),
      readFile(new URL("../../../../interno.manifest", directorio), "utf8"),
      readFile(new URL("../../../../produccion.manifest", directorio), "utf8"),
    ]);
  const candidato = `${datosFuente}\n${presentadorFuente}\n${vistaFuente}`;
  assert.doesNotMatch(
    candidato,
    /\b(?:fetch|XMLHttpRequest|WebSocket|EventSource)\s*\(|document\.cookie|localStorage|sessionStorage|indexedDB/i,
  );
  assert.doesNotMatch(`${interno}\n${produccion}`, /adaptador-presentacion\.js|datos-presentacion\.js/);
  for (const neutro of [
    "contrato-expedientes.js", "presentador-expedientes.js", "vista-expedientes.js",
    "componentes-expedientes.js", "expedientes.css",
  ]) {
    assert.match(interno, new RegExp(neutro.replace(".", "\\.")));
    assert.match(produccion, new RegExp(neutro.replace(".", "\\.")));
  }
});


test("la vista de alta no renderiza subcabecera redundante ni bloque de llamamiento", () => {
  const estado = {
    vista: "alta",
    carga: "listo",
    tipo_mensaje: "informacion",
    mensaje_clave: "estado_inicial",
    cuadro: null,
    expediente: null,
  };
  const html = renderizarModuloContratacionTemporal(estado, {
    altaDisponible: true,
    llamamientoDisponible: true,
  });
  assert.doesNotMatch(html, /ct-exp-subcabecera/u);
  assert.doesNotMatch(html, /Nueva petición de personal/u);
  assert.doesNotMatch(html, /data-ct-exp-llamamiento/u);
  assert.match(html, /data-ct-exp-alta/u);
});
