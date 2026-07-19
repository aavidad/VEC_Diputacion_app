import { exigirContextoActorCronos, validarDatosCronos } from "./contrato.js";

function referenciaOpaca(valor, nombre) {
  if (typeof valor !== "string" || !/^[A-Za-z0-9][A-Za-z0-9:._-]{5,127}$/.test(valor)) {
    throw new Error(`${nombre} no válida`);
  }
  return valor;
}

function fila(etiqueta, valor) {
  return Object.freeze({ etiqueta, valor: String(valor ?? "-") });
}

function instanteEuropeMadrid(instante) {
  const fecha = new Date(instante);
  if (!Number.isFinite(fecha.getTime())) throw new Error("instante de recibo no válido");
  return new Intl.DateTimeFormat("es-ES", {
    timeZone: "Europe/Madrid", day: "2-digit", month: "2-digit", year: "numeric",
    hour: "2-digit", minute: "2-digit", timeZoneName: "short",
  }).format(fecha);
}

function fechaCalendarioEuropeMadrid(fechaISO) {
  const [anio, mes, dia] = fechaISO.split("-").map(Number);
  return new Intl.DateTimeFormat("es-ES", {
    timeZone: "Europe/Madrid", day: "2-digit", month: "2-digit", year: "numeric",
  }).format(new Date(Date.UTC(anio, mes - 1, dia, 12)));
}

function cantidadDocumento(valor, unidad) {
  if (unidad === "dia") return `${valor} ${valor === 1 ? "día" : "días"}`;
  return `${String(Math.floor(valor / 60)).padStart(2, "0")}:${String(valor % 60).padStart(2, "0")} h`;
}

function registroDeRecibo(datos, referencia) {
  const movimientos = {
    entrada: "Entrada", salida: "Salida", inicio_pausa: "Inicio de pausa", fin_pausa: "Fin de pausa",
  };
  const estados = {
    registrado: "Registrado", revisado: "Revisado", simulado: "Simulado",
    pendiente_responsable: "Pendiente responsable", aprobado: "Aprobado",
    disfrutado: "Disfrutado", calculado: "Calculado",
    preparado_no_registrado: "Preparado · no registrado", sin_registrar: "Sin registrar",
  };
  const fichaje = datos.fichajes.find((item) => item.recibo_ref === referencia);
  if (fichaje) {
    return {
      clase: "fichaje",
      filas: [
        fila("Actuación", `Fichaje: ${movimientos[fichaje.tipo_clave]}`),
        fila("Instante", instanteEuropeMadrid(fichaje.instante)),
        fila("Modalidad", fichaje.modalidad),
        fila("Canal", fichaje.canal),
        fila("Estado", estados[fichaje.estado_clave]),
      ],
    };
  }
  const solicitud = datos.solicitudes.find((item) => item.recibo_ref === referencia);
  if (solicitud) {
    return {
      clase: "permiso",
      filas: [
        fila("Actuación", `Solicitud: ${solicitud.tipo}`),
        fila("Desde", fechaCalendarioEuropeMadrid(solicitud.desde)),
        fila("Hasta", fechaCalendarioEuropeMadrid(solicitud.hasta)),
        fila("Cantidad", cantidadDocumento(solicitud.cantidad_valor, solicitud.unidad_clave)),
        fila("Estado", estados[solicitud.estado_clave]),
      ],
    };
  }
  const evento = datos.historial.find((item) => item.recibo_ref === referencia);
  if (evento) {
    return {
      clase: "historial",
      filas: [
        fila("Actuación", evento.evento),
        fila("Instante", instanteEuropeMadrid(evento.instante)),
        fila("Detalle", evento.detalle),
        fila("Estado", estados[evento.estado_clave]),
      ],
    };
  }
  throw new Error("recibo de Cronos no encontrado en el ámbito propio");
}

/** Descriptor común consumido por el generador PDF institucional. */
export function prepararDescriptorReciboCronos({
  contextoActor, datos: datosSinValidar, recibo_ref: reciboRef, origenComprobacion = "",
}) {
  const contexto = exigirContextoActorCronos(contextoActor);
  const datos = validarDatosCronos(datosSinValidar, contexto);
  const referencia = referenciaOpaca(reciboRef, "referencia de recibo");
  const registro = registroDeRecibo(datos, referencia);
  const ruta = `/verificar/?ref=${encodeURIComponent(referencia)}${datos.demostracion ? "&presentacion=rrhh" : ""}`;
  let contenidoQR = ruta;
  if (origenComprobacion) {
    try {
      const origen = new URL(origenComprobacion);
      if (!new Set(["http:", "https:"]).has(origen.protocol)) throw new Error("protocolo no permitido");
      contenidoQR = new URL(ruta, origen).href;
    } catch {
      throw new Error("origen de comprobación no válido");
    }
  }
  return Object.freeze({
    esquema: "vec.documentos.recibo.cronos.v1",
    modulo: "cronos",
    formato: "pdf",
    referencia,
    titulo: "Recibo de actuación en Cronos",
    subtitulo: "Portal del Empleado · Diputación de Granada",
    identidad_visual: Object.freeze({
      entidad: "Diputación de Granada",
      logo_src: "/portal-empleado/assets/logo-diputacion-granada.svg",
      texto_alternativo: "Diputación de Granada",
    }),
    demostracion: datos.demostracion,
    marca: datos.demostracion ? "DOCUMENTO DEMO · SIN EFECTOS ADMINISTRATIVOS" : "",
    clase: registro.clase,
    filas: Object.freeze(registro.filas),
    comprobacion: Object.freeze({
      ruta,
      qr_contenido: contenidoQR,
      contiene_datos_personales: false,
      metodo: datos.demostracion ? "consulta_estatica_demo" : "post_servicio_cotejo",
    }),
    nombre_archivo: `recibo-cronos-${referencia.toLowerCase().replace(/[^a-z0-9_-]+/g, "-")}.pdf`,
  });
}

/**
 * Construye la petición al puerto documental común. El presentador no genera,
 * firma, sella ni descarga PDFs por su cuenta.
 */
export function crearSolicitudPDFReciboCronos({ contextoActor, recibo_ref: reciboRef }) {
  const contexto = exigirContextoActorCronos(contextoActor);
  return Object.freeze({
    esquema: "vec.documentos.solicitud-generacion.v1",
    modulo: "cronos",
    actor_ref: contexto.actor.actor_ref,
    recibo_ref: referenciaOpaca(reciboRef, "referencia de recibo"),
    formato: "pdf",
    plantilla: "recibo_cronos_institucional",
    marca_institucional: true,
    verificacion_qr: true,
    incluir_datos_sensibles_en_verificacion: false,
  });
}

export async function solicitarPDFReciboCronos({ puertoDocumental, contextoActor, recibo_ref: reciboRef }) {
  if (typeof puertoDocumental !== "function") throw new Error("puerto documental no conectado");
  const solicitud = crearSolicitudPDFReciboCronos({ contextoActor, recibo_ref: reciboRef });
  const resultado = await puertoDocumental(solicitud);
  if (!resultado || typeof resultado !== "object"
    || resultado.esquema !== "vec.documentos.resultado-generacion.v1"
    || resultado.medio !== "application/pdf"
    || typeof resultado.nombre !== "string" || !resultado.nombre.toLowerCase().endsWith(".pdf")
    || typeof resultado.documento_ref !== "string"
    || typeof resultado.verificacion_ref !== "string") {
    throw new Error("el puerto documental no ha devuelto un PDF verificable");
  }
  return Object.freeze({ ...resultado });
}
