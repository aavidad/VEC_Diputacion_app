import assert from "node:assert/strict";
import test from "node:test";

import {
  CAPACIDAD_CONSULTAR_FICHAJES,
  CAPACIDAD_CONSULTAR_HORARIO,
  validarDatosCronos,
} from "./contrato.js";
import { renderizarJornadaCronos, validarSeleccionPeriodoCronos } from "./vista.js";
import { MENSAJES_CRONOS_ES } from "./i18n.js";
import { ESQUEMA_CONTEXTO_ACTOR_FRONTEND, validarYCongelarContextoActor } from "../../identidad/contexto-actor.js";

function contexto() {
  return validarYCongelarContextoActor({
    esquema: ESQUEMA_CONTEXTO_ACTOR_FRONTEND,
    revision: 1,
    demostracion: false,
    persona_ref: "per_persona_sintetica_000001",
    cuenta_ref: "cta_cuenta_sintetica_000001",
    perfil_ref: "prf_perfil_sintetico_000001",
    actor: { actor_ref: "act_actor_sintetico_000001", nombre_visible: "Persona sintética", iniciales: "PS" },
    rol: { clave: "personal_interno", etiqueta: "Personal interno" },
    ambito: {
      clase: "personal_interno",
      organizacion_ref: "org_organizacion_sintetica_000001",
      unidad_ref: "uni_unidad_sintetica_000001",
      modulos: ["cronos"],
    },
    autenticacion: { sesion_ref: "ses_sesion_sintetica_000001", metodo: "kerberos_ad", garantia: "alto" },
    resuelto_en: "2026-09-24T09:00:00.000Z",
  });
}

function datos(actor) {
  return {
    esquema: "vec.cronos.area-personal.v1",
    actor_ref: actor.actor.actor_ref,
    periodo: "Septiembre de 2026",
    actualizado_en: "2026-09-24T09:00:00.000Z",
    perfil_jornada: { nombre: "Jornada ordinaria", jornada_diaria: "07:30", ventana_entrada: "07:00–09:00", tramo_obligatorio: "09:00–14:00" },
    resumen: { teoricas_hoy: "07:30", trabajadas_hoy: "02:00", saldo_hoy: "-05:30", saldo_periodo: "+01:00" },
    fichajes: [{ id: "fic_marcaje_sintetico_000001", actor_ref: actor.actor.actor_ref,
      instante: "2026-09-24T07:00:00.000Z", tipo_clave: "entrada", canal: "Terminal",
      modalidad: "Presencial", estado_clave: "registrado", recibo_ref: "rec_marcaje_sintetico_000001" }],
    saldos: [], solicitudes: [], historial: [],
  };
}

test("Jornada sin fuente muestra estados honestos y no ofrece fichaje genérico", () => {
  for (const estado of ["cargando", "vacio", "no_configurado", "denegado", "error"]) {
    const html = renderizarJornadaCronos({ estado });
    assert.match(html, new RegExp(`data-estado="${estado}"`));
    assert.doesNotMatch(html, /tabla-cronos-jornada|02:00/);
    assert.doesNotMatch(html, /data-cronos-fichar|jornada-acciones/);
  }
  assert.match(renderizarJornadaCronos({ estado: "disponible" }), /data-estado="no_configurado"/);
  assert.throws(() => renderizarJornadaCronos({ estado: "inventado" }), /estado de jornada/);
});

test("la consulta propia exige actor y capacidad explícita", () => {
  const actor = contexto();
  const proyeccion = datos(actor);
  const html = renderizarJornadaCronos({ estado: "disponible", contextoActor: actor,
    capacidades: [CAPACIDAD_CONSULTAR_FICHAJES, CAPACIDAD_CONSULTAR_HORARIO], datos: proyeccion });
  assert.match(html, /data-estado="disponible"/);
  assert.match(html, /tabla-cronos-jornada/);
  assert.match(html, /Terminal/);
  assert.match(html, /aria-label="(Abrir ayuda sobre [^"]+)" title="\1"><span aria-hidden="true">\?<\/span><\/button>/);
  assert.doesNotMatch(html, /cronos-boton-ayuda-texto/);
  for (const clave of ["jornada_descripcion", "jornada_periodo_descripcion",
    "jornada_movimientos_detalle", "horario_descripcion", "jornada_calendario_fuente"]) {
    assert.equal(html.includes(MENSAJES_CRONOS_ES[clave]), false, clave);
  }
  assert.match(renderizarJornadaCronos({ estado: "disponible", contextoActor: actor,
    capacidades: [], datos: proyeccion }), /data-estado="denegado"/);
  assert.throws(() => validarDatosCronos({ ...proyeccion, actor_ref: "act_otro_actor_sintetico_000001" }, actor), /no pertenecen/);
  assert.throws(() => validarDatosCronos({ ...proyeccion, fichajes: [
    { ...proyeccion.fichajes[0], actor_ref: "act_otro_actor_sintetico_000001" },
  ] }, actor), /registros ajenos/);
  assert.throws(() => validarDatosCronos({ ...proyeccion, saldos: [{ id: "sal_otro_000001", estado_clave: "estado_desconocido" }] }, actor), /contrato cerrado/);
});

test("la selección civil rechaza fechas imposibles y no finge un saldo", () => {
  assert.throws(() => validarSeleccionPeriodoCronos({ tipo: "dia", desde: "2026-02-30" }), /fecha/);
  assert.throws(() => validarSeleccionPeriodoCronos({ tipo: "periodo", desde: "2026-09-25", hasta: "2026-09-24" }), /rango/);
  const seleccion = validarSeleccionPeriodoCronos({ tipo: "semana", desde: "2026-09-24" });
  const actor = contexto();
  const html = renderizarJornadaCronos({ estado: "disponible", contextoActor: actor,
    capacidades: [CAPACIDAD_CONSULTAR_HORARIO], datos: datos(actor), seleccion });
  assert.match(html, /Semana del 21\/09\/2026 al 27\/09\/2026/);
  assert.match(html, /Falta una proyección autorizada para el periodo seleccionado/);
  assert.doesNotMatch(html, /tabla-cronos-jornada/);
});
