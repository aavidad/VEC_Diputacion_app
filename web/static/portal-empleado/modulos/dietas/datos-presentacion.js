/**
 * Escenario exclusivamente demostrativo de Dietas.
 *
 * Este adaptador no posee un directorio de personas. La identidad debe llegar
 * desde el nucleo del Portal del Empleado y se conserva como referencia opaca.
 * Los expedientes, importes, fechas y actuaciones son sintéticos; al sustituir
 * este fichero por el adaptador HTTP no cambia la vista.
 */

import {
  ESQUEMA_GEOMETRIA_RUTA_DIETAS,
  ESQUEMA_PANEL_DIETAS,
  exigirContextoActorDietas,
  validarGeometriaRutaDietas,
} from "./contrato.js";

function copiar(valor) {
  return structuredClone(valor);
}

const COORDENADAS_PUBLICAS_APROXIMADAS = Object.freeze({
  granada: Object.freeze([37.177, -3.599]),
  albolote: Object.freeze([37.231, -3.657]),
  motril: Object.freeze([36.744, -3.518]),
  guadix: Object.freeze([37.299, -3.136]),
  loja: Object.freeze([37.168, -4.151]),
  baza: Object.freeze([37.491, -2.773]),
});

function claveLugar(valor) {
  return String(valor ?? "").normalize("NFD").replace(/[\u0300-\u036f]/g, "").toLocaleLowerCase("es").trim();
}

export function crearGeometriaRutaDietasPresentacion(ruta) {
  if (!Array.isArray(ruta) || ruta.length < 2 || ruta.length > 12) throw new Error("ruta DEMO no valida");
  const paradas = ruta.map((etiqueta, indice) => {
    const texto = String(etiqueta ?? "").trim();
    if (!texto || texto.length > 80) throw new Error("parada DEMO no valida");
    const conocida = COORDENADAS_PUBLICAS_APROXIMADAS[claveLugar(texto)];
    const coordenada = conocida || [37.177 + indice * 0.035, -3.599 + (indice % 2 ? 0.08 : -0.05)];
    return { etiqueta: texto, latitud: coordenada[0], longitud: coordenada[1] };
  });
  const trazado = [];
  paradas.forEach((parada, indice) => {
    if (indice === 0) trazado.push([parada.latitud, parada.longitud]);
    const siguiente = paradas[indice + 1];
    if (!siguiente) return;
    const signo = indice % 2 === 0 ? 1 : -1;
    trazado.push([
      (parada.latitud + siguiente.latitud) / 2 + signo * 0.008,
      (parada.longitud + siguiente.longitud) / 2 + signo * 0.011,
    ]);
    trazado.push([siguiente.latitud, siguiente.longitud]);
  });
  return validarGeometriaRutaDietas({
    esquema: ESQUEMA_GEOMETRIA_RUTA_DIETAS,
    origen: "sintetica_demo",
    liquidable: false,
    paradas,
    trazado,
  }, ruta);
}

const COMISIONES_BASE = Object.freeze([
  {
    referencia: "DEMO-DIE-2026-0084",
    fecha: "2026-06-19",
    motivo: "Reunión técnica de coordinación",
    ruta: ["Granada", "Albolote", "Granada"],
    kilometros: 21.6,
    kilometraje_euros: 5.62,
    manutencion_euros: 22.78,
    alojamiento_euros: 0,
    otros_gastos_euros: 0,
    total_euros: 28.4,
    estado: "aprobada",
    etapa_actual: 2,
    justificantes: 1,
    siguiente_actuacion: "remision_rrhh",
    historial: [
      { estado: "borrador", instante: "2026-06-17T07:10:00Z", actor_ref: "@usuario_actual", recibo: "DEMO-REC-DIE-0084-01" },
      { estado: "pendiente_jefatura", instante: "2026-06-17T07:12:00Z", actor_ref: "@usuario_actual", recibo: "DEMO-REC-DIE-0084-02" },
      { estado: "aprobada", instante: "2026-06-18T09:40:00Z", actor_ref: "DEMO-JEFATURA-OBRAS-01", recibo: "DEMO-REC-DIE-0084-03" },
    ],
  },
  {
    referencia: "DEMO-DIE-2026-0091",
    fecha: "2026-06-21",
    motivo: "Visita técnica pendiente de completar",
    ruta: ["Granada", "Motril", "Granada"],
    kilometros: 140.8,
    kilometraje_euros: 36.61,
    manutencion_euros: 0,
    alojamiento_euros: 0,
    otros_gastos_euros: 0,
    total_euros: 36.61,
    estado: "borrador",
    etapa_actual: 0,
    justificantes: 0,
    siguiente_actuacion: "completar_enviar_validacion",
    historial: [
      { estado: "borrador", instante: "2026-06-21T06:00:00Z", actor_ref: "@usuario_actual", recibo: "DEMO-REC-DIE-0091-01" },
    ],
  },
  {
    referencia: "DEMO-DIE-2026-0073",
    fecha: "2026-05-27",
    motivo: "Inspección de obra provincial",
    ruta: ["Granada", "Guadix", "Granada"],
    kilometros: 107.2,
    kilometraje_euros: 27.87,
    manutencion_euros: 34.01,
    alojamiento_euros: 0,
    otros_gastos_euros: 0,
    total_euros: 61.88,
    estado: "pagada",
    etapa_actual: 5,
    justificantes: 2,
    siguiente_actuacion: "expediente_finalizado",
    nomina: "Nómina de junio de 2026",
    referencia_pago: "DEMO-LIQ-2026-0337",
    historial: [
      { estado: "borrador", instante: "2026-05-27T16:30:00Z", actor_ref: "@usuario_actual", recibo: "DEMO-REC-DIE-0073-01" },
      { estado: "pendiente_jefatura", instante: "2026-05-27T16:31:00Z", actor_ref: "@usuario_actual", recibo: "DEMO-REC-DIE-0073-02" },
      { estado: "aprobada", instante: "2026-05-28T08:05:00Z", actor_ref: "DEMO-JEFATURA-OBRAS-01", recibo: "DEMO-REC-DIE-0073-03" },
      { estado: "enviada_rrhh", instante: "2026-05-30T07:00:00Z", actor_ref: "DEMO-RRHH-DIETAS-01", recibo: "DEMO-REC-DIE-0073-04" },
      { estado: "enviada_nomina", instante: "2026-06-05T10:20:00Z", actor_ref: "DEMO-RRHH-DIETAS-01", recibo: "DEMO-REC-DIE-0073-05" },
      { estado: "pagada", instante: "2026-06-28T00:00:00Z", actor_ref: "DEMO-NOMINA-2026-06", recibo: "DEMO-REC-DIE-0073-06" },
    ],
  },
  {
    referencia: "DEMO-DIE-2026-0061",
    fecha: "2026-04-18",
    motivo: "Asistencia a jornada institucional",
    ruta: ["Granada", "Loja", "Granada"],
    kilometros: 112.6,
    kilometraje_euros: 29.28,
    manutencion_euros: 19.26,
    alojamiento_euros: 0,
    otros_gastos_euros: 0,
    total_euros: 48.54,
    estado: "enviada_nomina",
    etapa_actual: 4,
    justificantes: 1,
    siguiente_actuacion: "inclusion_nomina",
    historial: [
      { estado: "borrador", instante: "2026-04-18T15:00:00Z", actor_ref: "@usuario_actual", recibo: "DEMO-REC-DIE-0061-01" },
      { estado: "pendiente_jefatura", instante: "2026-04-18T15:01:00Z", actor_ref: "@usuario_actual", recibo: "DEMO-REC-DIE-0061-02" },
      { estado: "aprobada", instante: "2026-04-21T07:30:00Z", actor_ref: "DEMO-JEFATURA-OBRAS-01", recibo: "DEMO-REC-DIE-0061-03" },
      { estado: "enviada_rrhh", instante: "2026-04-24T06:45:00Z", actor_ref: "DEMO-RRHH-DIETAS-01", recibo: "DEMO-REC-DIE-0061-04" },
      { estado: "enviada_nomina", instante: "2026-05-02T09:00:00Z", actor_ref: "DEMO-RRHH-DIETAS-01", recibo: "DEMO-REC-DIE-0061-05" },
    ],
  },
  {
    referencia: "DEMO-DIE-2026-0048",
    fecha: "2026-03-12",
    motivo: "Supervisión de centro provincial",
    ruta: ["Granada", "Baza", "Granada"],
    kilometros: 216.8,
    kilometraje_euros: 56.37,
    manutencion_euros: 35.99,
    alojamiento_euros: 0,
    otros_gastos_euros: 0,
    total_euros: 92.36,
    estado: "pagada",
    etapa_actual: 5,
    justificantes: 2,
    siguiente_actuacion: "expediente_finalizado",
    nomina: "Nómina de mayo de 2026",
    referencia_pago: "DEMO-LIQ-2026-0298",
    historial: [
      { estado: "borrador", instante: "2026-03-12T17:00:00Z", actor_ref: "@usuario_actual", recibo: "DEMO-REC-DIE-0048-01" },
      { estado: "pendiente_jefatura", instante: "2026-03-12T17:02:00Z", actor_ref: "@usuario_actual", recibo: "DEMO-REC-DIE-0048-02" },
      { estado: "aprobada", instante: "2026-03-13T08:00:00Z", actor_ref: "DEMO-JEFATURA-OBRAS-01", recibo: "DEMO-REC-DIE-0048-03" },
      { estado: "enviada_rrhh", instante: "2026-03-18T07:15:00Z", actor_ref: "DEMO-RRHH-DIETAS-01", recibo: "DEMO-REC-DIE-0048-04" },
      { estado: "enviada_nomina", instante: "2026-04-10T10:00:00Z", actor_ref: "DEMO-RRHH-DIETAS-01", recibo: "DEMO-REC-DIE-0048-05" },
      { estado: "pagada", instante: "2026-05-28T00:00:00Z", actor_ref: "DEMO-NOMINA-2026-05", recibo: "DEMO-REC-DIE-0048-06" },
    ],
  },
]);

export function crearDatosDietasPresentacion(contextoInyectado) {
  const contexto = exigirContextoActorDietas(contextoInyectado);
  if (contexto.demostracion !== true) throw new Error("los datos de presentación exigen un contexto DEMO");
  const actorRef = contexto.actor.actor_ref;
  const comisiones = copiar(COMISIONES_BASE).map((comision) => ({
    ...comision,
    titular_ref: actorRef,
    geometria_ruta: crearGeometriaRutaDietasPresentacion(comision.ruta),
    historial: comision.historial.map((evento) => ({
      ...evento,
      actor_ref: evento.actor_ref === "@usuario_actual" ? actorRef : evento.actor_ref,
    })),
  }));
  return {
    esquema: ESQUEMA_PANEL_DIETAS,
    origen: {
      demostracion: true,
      efectos_reales: false,
      adaptador: "presentacion_volatil",
    },
    politica: {
      tarifa_kilometro_euros: 0.26,
      version: "DEMO-POL-DIETAS-2026-01",
      advertencia: "Tarifas e importes sintéticos para validar la interfaz; no constituyen una liquidación oficial.",
    },
    borrador_inicial: {
      fecha: "2026-07-20",
      motivo: "Visita técnica",
      origen: "Granada",
      destino: "Motril",
      kilometros: 140.8,
      manutencion_euros: 0,
      alojamiento_euros: 0,
      otros_gastos_euros: 0,
    },
    etapas: ["borrador", "jefatura", "aprobada", "rrhh", "nomina", "pagada"],
    comisiones,
    ultimo_recibo: null,
  };
}
