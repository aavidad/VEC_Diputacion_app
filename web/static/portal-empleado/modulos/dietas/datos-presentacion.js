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

// Estas muestras no representan expedientes reales ni un circuito de aprobación.
// La persistencia que se describe aquí vive sólo en la pestaña actual.
const COMISIONES_BASE = Object.freeze([
  {
    referencia: "DEMO-DIE-BORR-001",
    fecha: "2026-07-20",
    motivo: "Ejemplo de comisión para revisar la pantalla",
    ruta: ["Granada", "Motril", "Granada"],
    kilometros: 140.8,
    kilometraje_euros: 36.61,
    manutencion_euros: 18.5,
    alojamiento_euros: 0,
    otros_gastos_euros: 4.25,
    total_euros: 59.36,
    estado: "borrador",
    etapa_actual: 0,
    justificantes: 0,
    siguiente_actuacion: "completar_enviar_validacion",
    historial: [{ estado: "borrador", instante: "2026-07-20T07:10:00Z", actor_ref: "@usuario_actual", recibo: "DEMO-REC-DIE-BORR-001" }],
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
      advertencia: "Datos sintéticos de presentación; no liquidable, no enviado y sin tarifa oficial.",
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
