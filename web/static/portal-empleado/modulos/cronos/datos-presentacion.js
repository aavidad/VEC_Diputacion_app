import { exigirContextoActorCronos, validarDatosCronos } from "./contrato.js";

/**
 * Fixture aislado de presentación. Todas las referencias operativas son
 * sintéticas y se vinculan al actor ya autenticado por el portal; nunca crea
 * una segunda cuenta ni incorpora DNI, contacto o coordenadas.
 */
export function crearDatosCronosPresentacion(contextoActor) {
  const contexto = exigirContextoActorCronos(contextoActor);
  const actorRef = contexto.actor.actor_ref;
  return validarDatosCronos({
    esquema: "vec.cronos.area-personal.v1",
    demostracion: true,
    actor_ref: actorRef,
    periodo: "Julio de 2026 · escenario DEMO",
    actualizado_en: "2026-07-19T09:15:00Z",
    perfil_jornada: {
      referencia: "DEMO-HOR-FLEX-01",
      nombre: "Jornada flexible ordinaria · escenario DEMO",
      jornada_diaria: "07:30",
      jornada_semanal: "37:30",
      ventana_entrada: "07:00–09:00",
      tramo_obligatorio: "09:00–14:00",
      teletrabajo: "Según autorización vigente",
    },
    resumen: {
      teoricas_hoy: "07:30",
      trabajadas_hoy: "07:36",
      saldo_hoy: "+00:06",
      saldo_periodo: "+02:18",
      incidencias_abiertas: 1,
      solicitudes_pendientes: 1,
    },
    fichajes: [
      { id: "DEMO-FIC-20260719-03", actor_ref: actorRef, instante: "2026-07-19T13:08:00Z", tipo_clave: "salida", canal: "Terminal DEMO", modalidad: "Presencial", estado_clave: "registrado", recibo_ref: "DEMO-REC-FIC-1903" },
      { id: "DEMO-FIC-20260719-02", actor_ref: actorRef, instante: "2026-07-19T08:36:00Z", tipo_clave: "fin_pausa", canal: "Terminal DEMO", modalidad: "Presencial", estado_clave: "registrado", recibo_ref: "DEMO-REC-FIC-1902" },
      { id: "DEMO-FIC-20260719-01", actor_ref: actorRef, instante: "2026-07-19T08:16:00Z", tipo_clave: "inicio_pausa", canal: "Terminal DEMO", modalidad: "Presencial", estado_clave: "registrado", recibo_ref: "DEMO-REC-FIC-1901" },
      { id: "DEMO-FIC-20260719-00", actor_ref: actorRef, instante: "2026-07-19T05:12:00Z", tipo_clave: "entrada", canal: "Terminal DEMO", modalidad: "Presencial", estado_clave: "registrado", recibo_ref: "DEMO-REC-FIC-1900" },
      { id: "DEMO-FIC-20260718-01", actor_ref: actorRef, instante: "2026-07-18T12:56:00Z", tipo_clave: "salida", canal: "Web interna DEMO", modalidad: "Teletrabajo", estado_clave: "revisado", recibo_ref: "DEMO-REC-FIC-1801" },
      { id: "DEMO-FIC-20260718-00", actor_ref: actorRef, instante: "2026-07-18T05:20:00Z", tipo_clave: "entrada", canal: "Web interna DEMO", modalidad: "Teletrabajo", estado_clave: "revisado", recibo_ref: "DEMO-REC-FIC-1800" },
    ],
    saldos: [
      { id: "asuntos_propios", nombre: "Asuntos propios", unidad_clave: "dia", concedido: 6, solicitado: 1, aprobado: 1, disfrutado: 1, restante: 3, estado_clave: "saldo_demo" },
      { id: "vacaciones", nombre: "Vacaciones", unidad_clave: "dia", concedido: 22, solicitado: 5, aprobado: 4, disfrutado: 3, restante: 10, estado_clave: "saldo_demo" },
      { id: "bolsa_conciliacion", nombre: "Bolsa horaria por conciliación", unidad_clave: "minuto", concedido: 1800, solicitado: 120, aprobado: 60, disfrutado: 180, restante: 1440, estado_clave: "saldo_demo" },
      { id: "compensacion_horaria", nombre: "Compensación horaria", unidad_clave: "minuto", concedido: 1140, solicitado: 0, aprobado: 120, disfrutado: 240, restante: 780, estado_clave: "saldo_demo" },
    ],
    solicitudes: [
      { id: "DEMO-VAC-2026-0031", actor_ref: actorRef, tipo: "Vacaciones", desde: "2026-08-05", hasta: "2026-08-09", cantidad_valor: 5, unidad_clave: "dia", estado_clave: "pendiente_responsable", recibo_ref: "DEMO-REC-VAC-0031" },
      { id: "DEMO-PER-2026-0018", actor_ref: actorRef, tipo: "Asuntos propios", desde: "2026-06-02", hasta: "2026-06-02", cantidad_valor: 1, unidad_clave: "dia", estado_clave: "aprobado", recibo_ref: "DEMO-REC-PER-0018" },
      { id: "DEMO-CON-2026-0007", actor_ref: actorRef, tipo: "Conciliación", desde: "2026-05-14", hasta: "2026-05-14", cantidad_valor: 120, unidad_clave: "minuto", estado_clave: "disfrutado", recibo_ref: "DEMO-REC-CON-0007" },
    ],
    historial: [
      { id: "DEMO-HIS-006", actor_ref: actorRef, instante: "2026-07-19T13:08:00Z", evento: "Jornada evaluada", detalle: "Saldo diario calculado: +00:06", estado_clave: "calculado", recibo_ref: "DEMO-REC-JOR-0719" },
      { id: "DEMO-HIS-005", actor_ref: actorRef, instante: "2026-07-17T07:42:00Z", evento: "Solicitud registrada", detalle: "Vacaciones DEMO del 05/08 al 09/08", estado_clave: "pendiente_responsable", recibo_ref: "DEMO-REC-VAC-0031" },
      { id: "DEMO-HIS-004", actor_ref: actorRef, instante: "2026-06-03T10:10:00Z", evento: "Permiso resuelto", detalle: "Asuntos propios DEMO", estado_clave: "aprobado", recibo_ref: "DEMO-REC-PER-0018" },
      { id: "DEMO-HIS-003", actor_ref: actorRef, instante: "2026-05-14T12:05:00Z", evento: "Permiso consumido", detalle: "Conciliación DEMO: 2 horas", estado_clave: "disfrutado", recibo_ref: "DEMO-REC-CON-0007" },
    ],
  }, contexto);
}
