/**
 * Adaptador volátil y sustituible del modo presentación.
 *
 * No realiza tráfico de red, no usa cookies ni almacenamiento del navegador y
 * no comparte estado con los clientes reales. Al recargar la página desaparece
 * toda actuación. Las operaciones permitidas están enumeradas y fallan cerradas.
 */

const ACTOR_PRESENTACION = "DEMO-PERFIL-ADMIN-FUNCIONAL-BOLSA-01";

const OPERACIONES = Object.freeze({
  "crear-convocatoria": { efecto: "Crear un borrador sintético de convocatoria", coleccion: "elaboraciones", estado: "Borrador" },
  "guardar-bases": { efecto: "Guardar una versión sintética de bases, calendario y baremo", coleccion: "elaboraciones", estado: "Bases configuradas" },
  "enviar-firma-convocatoria": { efecto: "Enviar la convocatoria sintética al circuito de firma", coleccion: "elaboraciones", estado: "Pendiente de firma" },
  "publicar-convocatoria": { efecto: "Publicar la convocatoria únicamente dentro de la demostración", coleccion: "elaboraciones", estado: "Publicada" },
  "admitir-solicitud": { efecto: "Admitir provisionalmente la solicitud sintética", coleccion: "solicitudes", estado: "Admitida provisional" },
  "excluir-solicitud": { efecto: "Excluir provisionalmente la solicitud sintética", coleccion: "solicitudes", estado: "Excluida provisional" },
  "registrar-subsanacion": { efecto: "Registrar una subsanación sintética", coleccion: "solicitudes", estado: "Subsanada" },
  "aceptar-merito": { efecto: "Aceptar el mérito sintético en revisión técnica", coleccion: "meritos_revision", estado: "Aceptado" },
  "rechazar-merito": { efecto: "Rechazar motivadamente el mérito sintético", coleccion: "meritos_revision", estado: "Rechazado" },
  "revocar-merito": { efecto: "Revocar una aceptación previa y conservar la rectificación", coleccion: "meritos_revision", estado: "Aceptación revocada" },
  "rehabilitar-merito": { efecto: "Rehabilitar un mérito rechazado tras nueva revisión", coleccion: "meritos_revision", estado: "Aceptado tras revisión" },
  "calcular-baremo": { efecto: "Calcular el ranking sintético con la versión de reglas visible", coleccion: "ranking", estado: "Calculado" },
  "publicar-lista-provisional": { efecto: "Publicar una lista provisional solo en memoria", coleccion: "ranking", estado: "Lista provisional" },
  "resolver-alegacion": { efecto: "Resolver motivadamente la alegación sintética", coleccion: "alegaciones", estado: "Estimada" },
  "desestimar-alegacion": { efecto: "Desestimar motivadamente la alegación sintética", coleccion: "alegaciones", estado: "Desestimada" },
  "validar-importacion": { efecto: "Validar el lote Convoca sintético sin cargar archivos reales", coleccion: "importaciones", estado: "Validada" },
  "descartar-importacion": { efecto: "Descartar el lote Convoca sintético", coleccion: "importaciones", estado: "Descartada" },
  "emitir-llamamiento": { efecto: "Preparar un llamamiento sin contactar a ninguna persona", coleccion: "llamamientos_demo", estado: "Preparado" },
  "registrar-respuesta": { efecto: "Registrar una respuesta sintética al llamamiento", coleccion: "llamamientos_demo", estado: "Respuesta aceptada" },
  "registrar-contrato": { efecto: "Registrar una relación temporal sintética", coleccion: "contratos", estado: "Activo" },
  "registrar-cese": { efecto: "Registrar un cese sintético y recalcular disponibilidad", coleccion: "contratos", estado: "Cese registrado" },
  "reincorporar-bolsa": { efecto: "Reincorporar el expediente sintético a la bolsa", coleccion: "contratos", estado: "Reincorporado" },
  "generar-documento": { efecto: "Generar una previsualización documental sintética", coleccion: "documentos", estado: "Generado" },
  "enviar-firma-documento": { efecto: "Enviar el documento sintético a un circuito de firma simulado", coleccion: "documentos", estado: "Pendiente de firma" },
  "firmar-documento": { efecto: "Registrar una firma exclusivamente demostrativa", coleccion: "documentos", estado: "Firmado en demostración" },
  "preparar-comunicacion": { efecto: "Preparar una comunicación sintética sin destinatario real", coleccion: "comunicaciones_demo", estado: "Preparada" },
  "enviar-comunicacion": { efecto: "Simular un envío sin invocar conectores externos", coleccion: "comunicaciones_demo", estado: "Envío simulado" },
  "exportar-informe": { efecto: "Preparar un recibo de exportación sin crear ni descargar un fichero", coleccion: null, estado: "Preparada" },
  "crear-rol": { efecto: "Crear una propuesta sintética de rol con mínimo privilegio", coleccion: "roles_demo", estado: "Borrador" },
  "revisar-permisos": { efecto: "Simular la revisión periódica de permisos", coleccion: "roles_demo", estado: "Revisado" },
  "guardar-configuracion": { efecto: "Guardar una configuración sintética versionada", coleccion: "configuraciones_demo", estado: "Borrador guardado" },
});

function copia(valor) {
  return structuredClone(valor);
}

function localizar(coleccion, objetivo) {
  if (!Array.isArray(coleccion)) return null;
  return coleccion.find((item) => [item.id, item.referencia, item.expediente, item.codigo]
    .some((valor) => String(valor || "") === objetivo)) || null;
}

export function crearAdaptadorPresentacion({ datosIniciales, reloj = () => new Date() } = {}) {
  if (!datosIniciales || datosIniciales.demostracion !== true || typeof reloj !== "function") {
    throw new TypeError("configuración del adaptador de presentación no válida");
  }
  const datos = copia(datosIniciales);
  let secuencia = 0;

  function describir(operacion, objetivo) {
    const definicion = OPERACIONES[operacion];
    if (!definicion) return null;
    return Object.freeze({
      actor: ACTOR_PRESENTACION,
      efecto: definicion.efecto,
      objetivo: String(objetivo || "DEMO-SIN-OBJETIVO"),
    });
  }

  function ejecutar({ operacion, objetivo, motivo = "Actuación solicitada durante la presentación" } = {}) {
    const definicion = OPERACIONES[operacion];
    if (!definicion) throw new Error("operación de presentación no permitida");
    const objetivoSeguro = String(objetivo || "DEMO-SIN-OBJETIVO");
    const instante = reloj();
    if (!(instante instanceof Date) || !Number.isFinite(instante.getTime())) {
      throw new Error("reloj de presentación no válido");
    }
    secuencia += 1;
    const recibo = Object.freeze({
      referencia: `DEMO-REC-${String(secuencia).padStart(6, "0")}`,
      actor: ACTOR_PRESENTACION,
      instante: instante.toISOString(),
      operacion,
      objetivo: objetivoSeguro,
      resultado: definicion.estado,
      motivo: String(motivo || "Motivo no indicado"),
      efectos_reales: false,
    });
    if (definicion.coleccion) {
      const registro = localizar(datos[definicion.coleccion], objetivoSeguro);
      if (registro) {
        registro.estado = definicion.estado;
        registro.ultimo_recibo = recibo.referencia;
      }
    }
    datos.actividad.unshift({
      accion: definicion.efecto,
      objeto: objetivoSeguro,
      actor: ACTOR_PRESENTACION,
      fecha: instante.toLocaleString("es-ES", { timeZone: "Europe/Madrid" }),
      recibo: recibo.referencia,
    });
    datos.auditoria_eventos.unshift(recibo);
    return recibo;
  }

  return Object.freeze({
    actor: ACTOR_PRESENTACION,
    describir,
    ejecutar,
    obtenerDatos: () => copia(datos),
  });
}

export const OPERACIONES_PRESENTACION = Object.freeze(Object.keys(OPERACIONES));
