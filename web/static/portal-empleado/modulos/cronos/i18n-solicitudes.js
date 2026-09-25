/** Textos de movimientos (calendario, ausencias y olvidos) y de permisos propios. */
export const MENSAJES_CRONOS_SOLICITUDES_ES = Object.freeze({
  sobrelinea: "Cronos",
  abrir_ayuda: "Abrir ayuda sobre {asunto}",
  anio_anterior: "Año anterior",
  anio_siguiente: "Año siguiente",
  cargando: "Consultando…",
  denegado: "No tiene permiso para consultar estos datos.",
  sin_empleado: "Su usuario no tiene una relación de empleo vigente en Cronos.",
  error: "No se pudieron consultar los datos. Inténtelo de nuevo más tarde.",
  cancelar: "Cancelar",
  fecha: "Fecha",
  desde: "Desde",
  hasta: "Hasta",
  hora: "Hora",
  hora_inicio: "Hora de inicio",
  hora_fin: "Hora de fin",
  estado: "Estado",
  duracion: "Duración",
  permiso: "Permiso",
  periodo: "Periodo",
  sin_filas: "No hay registros.",
  dias_uno: "{n} día",
  dias_otros: "{n} días",
  horas_minutos: "{h} h {m} min",
  horas: "{h} h",
  minutos: "{m} min",
  por_mes: "{valor} al mes",
  por_solicitud: "{valor} por solicitud",
  sin_limite: "—",

  calendario_titulo: "Calendario y ausencias",
  calendario_anual: "Calendario {anio}",
  calendario_sin_publicar: "El calendario laboral de {anio} no está publicado: no se marcan festivos.",
  calendario_mes: "{mes} de {anio}",
  leyenda: "Leyenda",
  tipo_marcaje: "Con marcajes",
  tipo_ausencia: "Ausencia",
  tipo_festivo: "Festivo",
  tipo_no_laborable: "No laborable",
  tipo_olvido: "Olvido comunicado",
  dia_con: "{fecha}: {tipos}",
  absentismos_titulo: "Ausencias",
  justificante_pendiente: "Pendiente de justificar",
  justificante_ok: "—",
  olvidos_titulo: "Olvidos de marcaje",
  olvido_solicitar: "Comunicar un olvido",
  olvido_formulario: "Comunicar un olvido de marcaje",
  olvido_movimiento: "Marcaje olvidado",
  olvido_enviar: "Enviar solicitud",
  olvido_enviando: "Enviando…",
  olvido_registrado: "Solicitud registrada el {fecha}. Queda pendiente de la jefatura.",
  olvido_ya_registrado: "Esta solicitud ya estaba registrada el {fecha}.",
  olvido_invalido: "Revise la fecha y la hora: no puede ser un día futuro ni de hace más de un año.",
  olvido_conflicto: "Ya existe otra solicitud con esa referencia. Vuelva a abrir el formulario.",
  olvido_error: "No se pudo registrar la solicitud. Puede reintentarla sin duplicarla.",
  movimiento_entrada: "Entrada",
  movimiento_salida: "Salida",
  movimiento_inicio_pausa: "Inicio de pausa",
  movimiento_fin_pausa: "Fin de pausa",
  correccion_pendiente_responsable: "Pendiente de la jefatura",
  correccion_pendiente_rrhh: "Pendiente de RRHH",
  correccion_denegada_responsable: "Denegada por la jefatura",
  correccion_denegada_rrhh: "Denegada por RRHH",
  correccion_pendiente_aplicacion: "Aprobada, pendiente de aplicar",
  correccion_aplicada: "Aplicada",

  permisos_titulo: "Permisos y licencias",
  permisos_anio: "Permisos de {anio}",
  permisos_a_confirmar: "Cuantías pendientes de confirmar por RRHH",
  kpi_pendientes_conceder: "Pendientes de conceder",
  kpi_pendientes_justificar: "Pendientes de justificar",
  kpi_concedidos: "Concedidos",
  col_concede: "Concede",
  col_maximo: "Máx.",
  col_minimo: "Mín.",
  col_solicitado: "Solicitado",
  col_concedido: "Concedido",
  col_resta: "Resta",
  col_accion: "Acción",
  circuito_A: "Administración",
  "circuito_J-A": "Jefatura y administración",
  solicitar: "Solicitar",
  solicitar_permiso: "Solicitar {permiso}",
  solicitar_enviar: "Enviar solicitud",
  solicitar_enviando: "Enviando…",
  solicitud_registrada: "Solicitud registrada: {cantidad}. Queda pendiente de conceder ({circuito}).",
  solicitud_ya_registrada: "Esta solicitud ya estaba registrada: {cantidad}.",
  error_peticion_invalida: "Revise las fechas y, si es por horas, que sea un solo día con la hora de fin posterior a la de inicio.",
  error_conflicto: "Ya existe otra solicitud con esa referencia. Vuelva a abrir el formulario.",
  error_permiso_no_solicitable: "Este permiso no se puede solicitar desde el portal.",
  error_calendario_no_publicado: "No se puede calcular en días laborables: su calendario laboral no está publicado.",
  error_fuera_de_limites: "La solicitud supera el máximo o no alcanza el mínimo de este permiso.",
  error_solapado: "Las fechas coinciden con otro permiso en días ya solicitado o concedido.",
  error_solicitud: "No se pudo registrar la solicitud. Puede reintentarla sin duplicarla.",
  pendientes_conceder_titulo: "Solicitados pendientes de conceder",
  pendientes_justificar_titulo: "Pendientes de justificar",
  concedidos_titulo: "Concedidos",
  estado_pendiente_jefatura: "Pendiente de la jefatura",
  estado_pendiente_administracion: "Pendiente de administración",
  estado_concedido: "Concedido",
  periodo_dias: "{desde} – {hasta}",
  periodo_horas: "{fecha}, {inicio}–{fin}",
});

const CLAVES = Object.freeze(Object.keys(MENSAJES_CRONOS_SOLICITUDES_ES));

/** Traductor estricto: una clave desconocida o un catálogo incompleto fallan. */
export function crearTraductorSolicitudesCronos(mensajes = MENSAJES_CRONOS_SOLICITUDES_ES) {
  const catalogo = { ...MENSAJES_CRONOS_SOLICITUDES_ES, ...mensajes };
  if (CLAVES.some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === "")) throw new Error("catálogo i18n de solicitudes de Cronos incompleto");
  return (clave, variables = {}) => {
    if (!CLAVES.includes(clave)) throw new Error(`clave i18n de Cronos desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/gu, (_c, variable) => String(variables[variable] ?? ""));
  };
}

/** Cantidad de un permiso: días enteros o minutos presentados en horas y minutos. */
export function formatearCantidadCronos(cantidad, unidad, t, locale = "es-ES") {
  if (!Number.isSafeInteger(cantidad) || cantidad < 0) return t("sin_limite");
  const numero = new Intl.NumberFormat(locale);
  if (unidad === "dia") {
    const regla = new Intl.PluralRules(locale).select(cantidad);
    return t(regla === "one" ? "dias_uno" : "dias_otros", { n: numero.format(cantidad) });
  }
  const h = Math.floor(cantidad / 60); const m = cantidad % 60;
  if (h === 0) return t("minutos", { m: numero.format(m) });
  if (m === 0) return t("horas", { h: numero.format(h) });
  return t("horas_minutos", { h: numero.format(h), m: numero.format(m) });
}
