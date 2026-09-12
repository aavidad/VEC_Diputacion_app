import assert from "node:assert/strict";
import test from "node:test";

import { obtenerDatosPresentacion } from "../../datos-presentacion.js";
import { crearContextoActorPresentacionDesdeSesion } from "../../identidad/presentacion.js";
import { crearAdaptadorContratacionTemporalPresentacion } from "./adaptador-presentacion.js";
import { CAPACIDADES_CONTRATACION_TEMPORAL as CAP } from "./contrato-expedientes.js";

const REFERENCIA = "exp-demo-contratacion-005487";

function crearAdaptadorAdministrativo() {
  return crearAdaptadorContratacionTemporalPresentacion({
    contextoActor: crearContextoActorPresentacionDesdeSesion(
      obtenerDatosPresentacion("administrador").sesion,
    ),
  });
}

function comando(expediente, tarea, accion) {
  return {
    esquema: "vec.contratacion_temporal.actuacion.v1",
    expediente_ref: expediente.expediente_ref,
    version_esperada: expediente.version,
    tarea_ref: tarea.tarea_ref,
    accion_ref: accion.accion_ref,
    datos: {},
  };
}

test("la alternativa manual DEMO abre seguimiento sin transmitir GINPIX", async () => {
  const fuente = crearAdaptadorAdministrativo();
  let expediente = await fuente.obtener(REFERENCIA);
  const tareaEnvioInicial = expediente.tareas.find(({ tarea_ref }) => tarea_ref === "tarea-envio-ginpix");
  const continuidadInicial = tareaEnvioInicial.acciones.find(
    ({ accion_ref }) => accion_ref === "continuar_seguimiento_manual",
  );
  assert.equal(continuidadInicial.capacidad, CAP.registrarSeguimiento);
  assert.equal(continuidadInicial.disponible, false);

  await assert.rejects(
    fuente.ejecutar(comando(expediente, tareaEnvioInicial, continuidadInicial)),
    /no está disponible/u,
  );

  for (const [tareaRef, accionRef] of [
    ["tarea-formalizacion", "generar_documentos_formalizacion"],
    ["tarea-formalizacion", "enviar_firma_formalizacion"],
    ["tarea-incorporacion", "confirmar_incorporacion"],
    ["tarea-ginpix", "generar_fichero_ginpix"],
  ]) {
    const tarea = expediente.tareas.find(({ tarea_ref }) => tarea_ref === tareaRef);
    const accion = tarea.acciones.find(({ accion_ref }) => accion_ref === accionRef);
    await fuente.ejecutar(comando(expediente, tarea, accion));
    expediente = await fuente.obtener(REFERENCIA);
  }

  const tareaEnvio = expediente.tareas.find(({ tarea_ref }) => tarea_ref === "tarea-envio-ginpix");
  const envioExterno = tareaEnvio.acciones.find(({ accion_ref }) => accion_ref === "enviar_ginpix");
  const continuidad = tareaEnvio.acciones.find(
    ({ accion_ref }) => accion_ref === "continuar_seguimiento_manual",
  );
  assert.equal(envioExterno.disponible, false);
  assert.match(envioExterno.motivo_no_disponible, /conector corporativo/u);
  assert.equal(continuidad.disponible, true);

  const recibo = await fuente.ejecutar(comando(expediente, tareaEnvio, continuidad));
  assert.equal(recibo.actuacion, "Continuar por vía manual DEMO");
  assert.match(recibo.estado_resultante, /sin transmisión externa ni efectos reales/u);
  expediente = await fuente.obtener(REFERENCIA);

  const envioTrasContinuidad = expediente.tareas.find(
    ({ tarea_ref }) => tarea_ref === "tarea-envio-ginpix",
  );
  const seguimiento = expediente.tareas.find(({ tarea_ref }) => tarea_ref === "tarea-seguimiento");
  assert.equal(envioTrasContinuidad.estado_clave, "completado");
  assert.equal(
    envioTrasContinuidad.acciones.find(({ accion_ref }) => accion_ref === "enviar_ginpix").disponible,
    false,
  );
  assert.equal(seguimiento.estado_clave, "en_curso");
  assert.equal(
    seguimiento.acciones.find(({ accion_ref }) => accion_ref === "registrar_seguimiento").disponible,
    true,
  );

  const auditoria = await fuente.obtenerAuditoria(REFERENCIA);
  const actuacion = auditoria.actuaciones.at(-1);
  assert.equal(actuacion.accion, "Continuar por vía manual DEMO");
  assert.match(actuacion.observaciones, /sin transmisión externa ni efectos administrativos/u);
});
