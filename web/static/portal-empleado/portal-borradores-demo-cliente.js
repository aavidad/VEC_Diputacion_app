/** Cliente en memoria con el mismo contrato que el adaptador HTTP de borradores. */
import { ESQUEMAS_BORRADORES, derivarETagBorrador } from "./portal-borradores-contrato.js";

const HUELLA_A = "a".repeat(64);
const HUELLA_B = "b".repeat(64);

function copia(valor) { return structuredClone(valor); }

function limites() {
  return {
    maximo_categorias: 1024, maximo_plazos: 64, maximo_requisitos: 256,
    maximo_documentos: 256, maximo_ayudas: 128, maximo_titulo: 180,
    maximo_resumen: 500, maximo_descripcion: 12_000, maximo_titulo_plazo: 180,
    maximo_descripcion_plazo: 1_000, maximo_titulo_requisito: 180,
    maximo_descripcion_requisito: 3_000, maximo_pregunta_ayuda: 300,
    maximo_respuesta_ayuda: 5_000,
  };
}

function contenidoInicial() {
  return {
    tipo: "demo_bolsa_temporal",
    categorias: ["demo_auxiliar_administrativo"],
    titulo: "Convocatoria DEMO de Bolsa temporal",
    resumen: "Borrador sintético para recorrer la edición definitiva.",
    descripcion: "Contenido sin validez administrativa y sin datos personales.",
    plazos: [{
      referencia: "DEMO-PLAZO-001", tipo: "presentacion_solicitudes",
      titulo: "Presentación de solicitudes", descripcion: "Periodo sintético.",
      abre_en: "2026-08-01T08:00:00Z", cierra_en: "2026-08-20T21:59:00Z",
    }],
    requisitos: [{
      referencia: "DEMO-REQ-001", orden: 1, titulo: "Titulación requerida",
      descripcion: "Requisito sintético definido por las bases.", obligatorio: true,
    }],
    ayuda: [{
      referencia: "DEMO-AYU-001", categoria: "solicitud", orden: 1,
      pregunta: "¿Cómo se presenta?", respuesta: "Siga el recorrido de presentación.",
    }],
  };
}

export function crearClienteBorradoresPresentacion({ reloj = () => new Date() } = {}) {
  let revision = 3;
  let secuencia = 0;
  let contenido = contenidoInicial();

  function referenciaEstado() {
    return { referencia: "DEMO-BORRADOR-001", revision, huella_estado_sha256: HUELLA_A };
  }

  function etag() { return derivarETagBorrador(referenciaEstado()); }

  function opciones() {
    return {
      esquema: ESQUEMAS_BORRADORES.opciones,
      categorias: [{ categoria_ref: "DEMO-CAT-001", version: 1, huella_sha256: HUELLA_A, clave: "demo_auxiliar_administrativo", etiqueta: "Auxiliar Administrativo" }],
      tipos: [{ tipo_ref: "DEMO-TIPO-001", version: 1, huella_sha256: HUELLA_B, clave: "demo_bolsa_temporal", etiqueta: "Bolsa temporal" }],
      plantillas: [{ plantilla_ref: "DEMO-PLT-001", version: 4, huella_sha256: HUELLA_A, nombre: "Plantilla DEMO general", descripcion: "Plantilla sintética de convocatoria." }],
      motivos: [{ motivo_ref: "DEMO-MOT-001", version: 2, huella_sha256: HUELLA_B, etiqueta: "Edición durante presentación", descripcion: "Actuación sin efectos reales." }],
      limites: limites(), capacidades: { consultar: true, crear: true },
    };
  }

  function fila() {
    return {
      referencia_estado: referenciaEstado(), etag: etag(), codigo_version_publica: "demo_v1",
      identificador_publico: "DEMO-BOLSA-001", titulo: contenido.titulo, tipo: contenido.tipo,
      categorias: [...contenido.categorias], expediente_ref: "DEMO-EXP-001",
      creada_en: "2026-07-18T08:00:00Z", actualizada_en: reloj().toISOString(),
      numero_plazos: contenido.plazos.length, numero_requisitos: contenido.requisitos.length,
      numero_documentos: 0, numero_ayudas: contenido.ayuda.length,
      capacidades: { consultar: true, actualizar: true },
    };
  }

  function lista() {
    return { esquema: ESQUEMAS_BORRADORES.lista, selector: { limite: 40 }, paginacion: { limite: 40, total: 1 }, capacidades: { consultar: true, crear: true }, elementos: [fila()] };
  }

  function detalle() {
    const ref = (referencia) => ({ referencia, version: 1, huella_sha256: HUELLA_A });
    return {
      esquema: ESQUEMAS_BORRADORES.detalle, referencia_estado: referenciaEstado(), etag: etag(),
      codigo_version_publica: "demo_v1", identificador_publico: "DEMO-BOLSA-001",
      ambito_lectura: { organizacion_ref: "DEMO-ORG-001", unidad_gestion_ref: "DEMO-UNI-001" },
      expediente_ref: "DEMO-EXP-001", contenido_editable: copia(contenido),
      configuracion_lectura: {
        catalogos: ref("DEMO-CFG-CATALOGOS"), calendario: ref("DEMO-CFG-CALENDARIO"),
        reglas_baremacion: ref("DEMO-CFG-BAREMO"), flujo_proceso: ref("DEMO-CFG-PROCESO"),
        flujo_solicitud: ref("DEMO-CFG-SOLICITUD"), plantilla: ref("DEMO-PLT-001"), documentos: [],
      },
      capacidades: { consultar: true, actualizar: true },
    };
  }

  function guardar(accion, solicitud) {
    contenido = copia(solicitud.contenido_editable);
    revision += 1;
    secuencia += 1;
    return {
      esquema: ESQUEMAS_BORRADORES.guardado,
      transaccion_ref: `DEMO-TRA-BOR-${String(secuencia).padStart(6, "0")}`,
      accion, referencia_estado: referenciaEstado(), etag: etag(),
      auditoria_ref: `DEMO-AUD-BOR-${String(secuencia).padStart(6, "0")}`,
      evento_outbox_ref: `DEMO-REC-BOR-${String(secuencia).padStart(6, "0")}`,
      confirmada_en: reloj().toISOString(),
    };
  }

  return Object.freeze({
    obtenerOpciones: async () => copia(opciones()),
    listar: async () => copia(lista()),
    obtenerDetalle: async () => copia(detalle()),
    crear: async (solicitud) => copia(guardar("crear", solicitud)),
    actualizar: async (_referencia, solicitud) => copia(guardar("actualizar", solicitud)),
  });
}
