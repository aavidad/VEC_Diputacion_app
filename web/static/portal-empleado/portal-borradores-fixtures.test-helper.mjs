import {
  ESQUEMAS_BORRADORES,
  derivarETagBorrador,
} from "./portal-borradores-contrato.js";

const HUELLA_A = "a".repeat(64);
const HUELLA_B = "b".repeat(64);
const CLAVE_IDEMPOTENCIA_A = "A".repeat(43);
const CLAVE_IDEMPOTENCIA_B = "AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE";

function etagEstado(revision, huella = HUELLA_A) {
  return derivarETagBorrador({ revision, huella_estado_sha256: huella });
}

function limites() {
  return {
    maximo_categorias: 1024,
    maximo_plazos: 64,
    maximo_requisitos: 256,
    maximo_documentos: 256,
    maximo_ayudas: 128,
    maximo_titulo: 180,
    maximo_resumen: 500,
    maximo_descripcion: 12_000,
    maximo_titulo_plazo: 180,
    maximo_descripcion_plazo: 1_000,
    maximo_titulo_requisito: 180,
    maximo_descripcion_requisito: 3_000,
    maximo_pregunta_ayuda: 300,
    maximo_respuesta_ayuda: 5_000,
  };
}

function opciones() {
  return {
    esquema: ESQUEMAS_BORRADORES.opciones,
    categorias: [{
      categoria_ref: "catalogo:categorias:2026", version: 7, huella_sha256: HUELLA_A,
      clave: "auxiliar_administrativo", etiqueta: "Auxiliar administrativo",
    }],
    tipos: [{
      tipo_ref: "catalogo:tipos-convocatoria:2026", version: 3, huella_sha256: HUELLA_B,
      clave: "bolsa_temporal", etiqueta: "Bolsa temporal",
    }],
    plantillas: [{
      plantilla_ref: "plantilla:convocatoria:general", version: 4, huella_sha256: HUELLA_A,
      nombre: "Plantilla general", descripcion: "Plantilla gobernada para convocatorias.",
    }],
    motivos: [{
      motivo_ref: "motivo:convocatoria:alta", version: 2, huella_sha256: HUELLA_B,
      etiqueta: "Alta de convocatoria", descripcion: "Inicio autorizado del expediente.",
    }],
    limites: limites(),
    capacidades: { consultar: true, crear: true },
  };
}

function contenido() {
  return {
    tipo: "bolsa_temporal",
    categorias: ["auxiliar_administrativo"],
    titulo: "Convocatoria de bolsa temporal",
    resumen: "Resumen de la convocatoria.",
    descripcion: "Descripción completa de la convocatoria.",
    plazos: [{
      referencia: "plazo:solicitudes:1", tipo: "presentacion_solicitudes",
      titulo: "Presentación de solicitudes", descripcion: "Periodo de presentación.",
      abre_en: "2026-08-01T08:00:00Z", cierra_en: "2026-08-15T12:00:00Z",
    }],
    requisitos: [{
      referencia: "requisito:titulacion:1", orden: 1, titulo: "Titulación",
      descripcion: "Titulación exigida por las bases.", obligatorio: true,
    }],
    ayuda: [{
      referencia: "ayuda:solicitud:1", categoria: "solicitud", orden: 1,
      pregunta: "¿Cómo se presenta?", respuesta: "Siga las instrucciones publicadas.",
    }],
  };
}

function referenciaEstado(revision = 3) {
  return {
    referencia: "convocatoria:externa:2026#1",
    revision,
    huella_estado_sha256: HUELLA_A,
  };
}

function solicitudCrear() {
  return {
    esquema: ESQUEMAS_BORRADORES.crear,
    plantilla_ref: "plantilla:convocatoria:general",
    plantilla_version: 4,
    plantilla_huella_sha256: HUELLA_A,
    codigo_version_publica: "version_inicial",
    identificador_publico: "bolsa-auxiliares-2026",
    expediente_ref: "expediente:seleccion:2026:17",
    contenido_editable: contenido(),
    motivo_ref: "motivo:convocatoria:alta",
    motivo_version: 2,
    motivo_huella_sha256: HUELLA_B,
  };
}

function solicitudActualizar() {
  return {
    esquema: ESQUEMAS_BORRADORES.actualizar,
    contenido_editable: contenido(),
    motivo_ref: "motivo:convocatoria:correccion",
    motivo_version: 3,
    motivo_huella_sha256: HUELLA_B,
  };
}

function fila() {
  return {
    referencia_estado: referenciaEstado(),
    etag: etagEstado(3),
    codigo_version_publica: "version_inicial",
    identificador_publico: "bolsa-auxiliares-2026",
    titulo: "Convocatoria de bolsa temporal",
    tipo: "bolsa_temporal",
    categorias: ["auxiliar_administrativo"],
    expediente_ref: "expediente:seleccion:2026:17",
    creada_en: "2026-07-18T08:00:00Z",
    actualizada_en: "2026-07-18T09:00:00Z",
    numero_plazos: 1,
    numero_requisitos: 1,
    numero_documentos: 2,
    numero_ayudas: 1,
    capacidades: { consultar: true, actualizar: true },
  };
}

function lista() {
  return {
    esquema: ESQUEMAS_BORRADORES.lista,
    selector: { limite: 40 },
    paginacion: { limite: 40, total: 1 },
    capacidades: { consultar: true, crear: true },
    elementos: [fila()],
  };
}

function configuracionLectura() {
  const referencia = (nombre) => ({ referencia: nombre, version: 1, huella_sha256: HUELLA_A });
  return {
    catalogos: referencia("catalogos:convocatoria:2026"),
    calendario: referencia("calendario:granada:2026"),
    reglas_baremacion: referencia("reglas:baremo:auxiliares"),
    flujo_proceso: referencia("flujo:proceso:bolsa"),
    flujo_solicitud: referencia("flujo:solicitud:bolsa"),
    plantilla: referencia("plantilla:convocatoria:general"),
    documentos: [{
      rol: "bases", publicacion_ref: "publicacion:bases:1", documento_ref: "documento:bases:1",
      version_documento: 1, representacion_ref: "representacion:bases:pdf:1",
      huella_contenido_sha256: HUELLA_B, firma_validada_ref: "firma:bases:1",
      recibo_custodia_ref: "custodia:bases:1",
    }],
  };
}

function detalle() {
  return {
    esquema: ESQUEMAS_BORRADORES.detalle,
    referencia_estado: referenciaEstado(),
    etag: etagEstado(3),
    codigo_version_publica: "version_inicial",
    identificador_publico: "bolsa-auxiliares-2026",
    ambito_lectura: { organizacion_ref: "org_0123456789abcdef", unidad_gestion_ref: "uni_0123456789abcdef" },
    expediente_ref: "expediente:seleccion:2026:17",
    contenido_editable: contenido(),
    configuracion_lectura: configuracionLectura(),
    capacidades: { consultar: true, actualizar: true },
  };
}

function recibo(accion = "crear") {
  const revision = accion === "crear" ? 1 : 4;
  return {
    esquema: ESQUEMAS_BORRADORES.guardado,
    transaccion_ref: "transaccion:borrador:2026:17",
    accion,
    referencia_estado: referenciaEstado(revision),
    etag: etagEstado(revision),
    auditoria_ref: "auditoria:borrador:2026:17",
    evento_outbox_ref: "evento:borrador:2026:17",
    confirmada_en: "2026-07-18T10:00:00Z",
  };
}

function respuestaJSON(data, { estado = 200, etag, location } = {}) {
  const headers = { "content-type": "application/json; charset=utf-8" };
  if (etag !== undefined) headers.etag = etag;
  if (location !== undefined) headers.location = location;
  return new Response(JSON.stringify({ data }), { status: estado, headers });
}

function respuestaError(
  estado,
  codigo,
  correlacionRef = "correlacion:borradores:0123456789abcdef",
  detalleExtra = {},
) {
  return new Response(JSON.stringify({
    error: { codigo, correlacion_ref: correlacionRef, ...detalleExtra },
  }), { status: estado, headers: { "content-type": "application/json; charset=utf-8" } });
}

function crearRespuestaControlada({
  estado = 200,
  tipo = "application/json; charset=utf-8",
  fragmentos = [],
  falloLectura = null,
  lecturaPendiente = false,
  falloCancelacionLector = null,
  falloCancelacionCuerpo = null,
  cabeceras = {},
} = {}) {
  let indice = 0;
  const traza = {
    cancelacionesCuerpo: 0,
    cancelacionesLector: 0,
    liberaciones: 0,
    lecturas: 0,
  };
  const lector = {
    async read() {
      traza.lecturas += 1;
      if (lecturaPendiente) return new Promise(() => {});
      if (falloLectura !== null) throw falloLectura;
      if (indice >= fragmentos.length) return { done: true, value: undefined };
      const value = fragmentos[indice];
      indice += 1;
      return { done: false, value };
    },
    async cancel() {
      traza.cancelacionesLector += 1;
      if (falloCancelacionLector !== null) throw falloCancelacionLector;
    },
    releaseLock() { traza.liberaciones += 1; },
  };
  const body = {
    getReader: () => lector,
    async cancel() {
      traza.cancelacionesCuerpo += 1;
      if (falloCancelacionCuerpo !== null) throw falloCancelacionCuerpo;
    },
  };
  const headers = new Headers({ "content-type": tipo, ...cabeceras });
  return { respuesta: { status: estado, headers, body }, traza };
}

export {
  CLAVE_IDEMPOTENCIA_A,
  CLAVE_IDEMPOTENCIA_B,
  HUELLA_A,
  HUELLA_B,
  configuracionLectura,
  contenido,
  crearRespuestaControlada,
  detalle,
  etagEstado,
  fila,
  limites,
  lista,
  opciones,
  recibo,
  referenciaEstado,
  respuestaError,
  respuestaJSON,
  solicitudActualizar,
  solicitudCrear,
};
