/**
 * Adaptador volátil y sustituible del modo presentación.
 *
 * No realiza tráfico de red, no usa cookies ni almacenamiento del navegador y
 * no comparte estado con los clientes reales. Al recargar la página desaparece
 * toda actuación. Las operaciones permitidas están enumeradas y fallan cerradas.
 */

const CAMPOS_BASES = ["denominacion", "categoria", "expediente", "tipo_proceso", "apertura", "cierre",
  "subsanacion_desde", "subsanacion_hasta", "version_bases", "medio_publicacion", "plantilla", "circuito_firma"];
const CAMPOS_DECISION = ["criterio", "motivo_tipificado", "observacion"];
const OPERACIONES = Object.freeze({
  "crear-convocatoria": { efecto: "Crear un borrador sintético de convocatoria", coleccion: "elaboraciones", estado: "Borrador", estrategia: "crear" },
  "guardar-bases": { efecto: "Guardar una versión sintética de bases y calendario", coleccion: "elaboraciones", estado: "Bases configuradas", estrategia: "actualizar", campos: CAMPOS_BASES },
  "guardar-reglas-baremo": { efecto: "Guardar una versión sintética de reglas de baremación", coleccion: "criterios_baremo", estado: "Borrador guardado", estrategia: "actualizar", campos: ["unidad_tiempo", "puntos_unidad", "fraccion_jornada", "tope_bloque", "ambito_experiencia", "redondeo", "desempate_1", "desempate_2", "desempate_3", "ultimo_recurso"] },
  "enviar-firma-convocatoria": { efecto: "Enviar la convocatoria sintética al circuito de firma", coleccion: "elaboraciones", estado: "Pendiente de firma", estrategia: "actualizar" },
  "publicar-convocatoria": { efecto: "Publicar la convocatoria únicamente dentro de la demostración", coleccion: "elaboraciones", estado: "Publicada", estrategia: "actualizar" },
  "admitir-solicitud": { efecto: "Admitir provisionalmente la solicitud sintética", coleccion: "solicitudes", estado: "Admitida provisional", estrategia: "actualizar" },
  "excluir-solicitud": { efecto: "Excluir provisionalmente la solicitud sintética", coleccion: "solicitudes", estado: "Excluida provisional", estrategia: "actualizar" },
  "registrar-subsanacion": { efecto: "Registrar una subsanación sintética", coleccion: "solicitudes", estado: "Subsanada", estrategia: "actualizar" },
  "aceptar-merito": { efecto: "Aceptar el mérito sintético en revisión técnica", coleccion: "meritos_revision", estado: "Aceptado", estrategia: "actualizar", campos: CAMPOS_DECISION },
  "rechazar-merito": { efecto: "Rechazar motivadamente el mérito sintético", coleccion: "meritos_revision", estado: "Rechazado", estrategia: "actualizar", campos: CAMPOS_DECISION },
  "revocar-merito": { efecto: "Revocar una aceptación previa y conservar la rectificación", coleccion: "meritos_revision", estado: "Aceptación revocada", estrategia: "actualizar", campos: CAMPOS_DECISION },
  "rehabilitar-merito": { efecto: "Rehabilitar un mérito rechazado tras nueva revisión", coleccion: "meritos_revision", estado: "Aceptado tras revisión", estrategia: "actualizar", campos: CAMPOS_DECISION },
  "calcular-baremo": { efecto: "Calcular el ranking sintético con la versión de reglas visible", coleccion: "ranking", estado: "Calculado", estrategia: "conjunto" },
  "publicar-lista-provisional": { efecto: "Publicar una lista provisional solo en memoria", coleccion: "ranking", estado: "Lista provisional", estrategia: "conjunto" },
  "resolver-alegacion": { efecto: "Resolver motivadamente la alegación sintética", coleccion: "alegaciones", estado: "Estimada", estrategia: "actualizar" },
  "desestimar-alegacion": { efecto: "Desestimar motivadamente la alegación sintética", coleccion: "alegaciones", estado: "Desestimada", estrategia: "actualizar" },
  "validar-importacion": { efecto: "Validar el lote Convoca sintético sin cargar archivos reales", coleccion: "importaciones", estado: "Validada", estrategia: "crear_o_actualizar" },
  "descartar-importacion": { efecto: "Descartar el lote Convoca sintético", coleccion: "importaciones", estado: "Descartada", estrategia: "actualizar" },
  "emitir-llamamiento": { efecto: "Preparar un llamamiento sin contactar a ninguna persona", coleccion: "llamamientos_demo", estado: "Preparado", estrategia: "crear_o_actualizar", campos: ["bolsa", "destino", "jornada", "duracion", "regla", "plazo_respuesta", "canales", "plantilla"] },
  "registrar-respuesta": { efecto: "Registrar una respuesta sintética al llamamiento", coleccion: "llamamientos_demo", estado: "Respuesta aceptada", estrategia: "actualizar" },
  "registrar-contrato": { efecto: "Registrar una relación temporal sintética", coleccion: "contratos", estado: "Activo", estrategia: "crear_o_actualizar" },
  "registrar-cese": { efecto: "Registrar un cese sintético y recalcular disponibilidad", coleccion: "contratos", estado: "Cese registrado", estrategia: "actualizar" },
  "reincorporar-bolsa": { efecto: "Reincorporar el expediente sintético a la bolsa", coleccion: "contratos", estado: "Reincorporado", estrategia: "actualizar" },
  "generar-documento": { efecto: "Generar una previsualización documental sintética", coleccion: "documentos", estado: "Generado", estrategia: "crear_o_actualizar", campos: ["plantilla", "formato", "circuito", "cotejo"] },
  "enviar-firma-documento": { efecto: "Enviar el documento sintético a un circuito de firma simulado", coleccion: "documentos", estado: "Pendiente de firma", estrategia: "actualizar" },
  "firmar-documento": { efecto: "Registrar una firma exclusivamente demostrativa", coleccion: "documentos", estado: "Firmado en demostración", estrategia: "actualizar" },
  "preparar-comunicacion": { efecto: "Preparar una comunicación sintética sin destinatario real", coleccion: "comunicaciones_demo", estado: "Preparada", estrategia: "crear_o_actualizar", campos: ["expediente", "plantilla", "canales", "plazo", "asunto", "contenido"] },
  "enviar-comunicacion": { efecto: "Simular un envío sin invocar conectores externos", coleccion: "comunicaciones_demo", estado: "Envío simulado", estrategia: "actualizar" },
  "exportar-informe": { efecto: "Preparar un recibo de exportación sin crear ni descargar un fichero", coleccion: null, estado: "Preparada", estrategia: "auditar", campos: ["informe", "ambito", "formato", "finalidad"] },
  "crear-rol": { efecto: "Crear una propuesta sintética de rol con mínimo privilegio", coleccion: "roles_demo", estado: "Borrador", estrategia: "crear", campos: ["nombre", "unidad", "recursos", "acciones", "vigencia_desde", "revision"] },
  "revisar-permisos": { efecto: "Simular la revisión periódica de permisos", coleccion: "roles_demo", estado: "Revisado", estrategia: "actualizar" },
  "guardar-configuracion": { efecto: "Guardar una configuración sintética versionada", coleccion: "configuraciones_demo", estado: "Borrador guardado", estrategia: "crear_o_actualizar" },
});

function copia(valor) {
  return structuredClone(valor);
}

function localizar(coleccion, objetivo) {
  if (!Array.isArray(coleccion)) return null;
  return coleccion.find((item) => [item.id, item.referencia, item.expediente, item.codigo]
    .some((valor) => String(valor || "") === objetivo)) || null;
}

function validarCampos(definicion, campos) {
  if (campos === undefined) return Object.freeze({});
  if (!campos || typeof campos !== "object" || Array.isArray(campos) || Object.getPrototypeOf(campos) !== Object.prototype) {
    throw new Error("campos de operación no válidos");
  }
  const entradas = Object.entries(campos);
  if (entradas.length > 32) throw new Error("demasiados campos en la operación");
  const permitidos = new Set(definicion.campos || []);
  const salida = {};
  for (const [nombre, valor] of entradas) {
    if (!permitidos.has(nombre) || !/^[a-z][a-z0-9_]{0,47}$/.test(nombre)) {
      throw new Error(`campo no permitido: ${nombre}`);
    }
    if (typeof valor !== "string" || valor.length > 2_000 || /[\u0000-\u0008\u000B\u000C\u000E-\u001F]/.test(valor)) {
      throw new Error(`valor no válido para ${nombre}`);
    }
    salida[nombre] = valor.trim();
  }
  return Object.freeze(salida);
}

function crearRegistro(operacion, objetivo, campos, estado, actor) {
  const comunes = { estado, ultima_entrada: copia(campos) };
  switch (operacion) {
    case "crear-convocatoria":
      return { id: objetivo, nombre: "Nueva convocatoria DEMO", expediente: "DEMO-EXP-NUEVO", fase: "Configuración inicial", reglas: "Sin versión", plazo: "Pendiente", responsable: "Unidad DEMO de Selección", version_bases: "Sin versión", calendario: "Pendiente", firmantes: "Pendientes", ...comunes };
    case "validar-importacion":
      return { id: objetivo, origen: "Lectura sintética", lote: "Lote DEMO sin fichero", huella: "DEMO-HUELLA-NUEVA", validas: 0, incidencias: 0, autoridad: actor, ...comunes };
    case "emitir-llamamiento":
      return { id: objetivo, necesidad: "DEMO-NEC-NUEVA", bolsa: campos.bolsa || "Bolsa DEMO", orden: campos.regla || "Regla DEMO", incluidos: 0, plazo: campos.plazo_respuesta || "Pendiente", canal: campos.canales || "Sin canal", ...comunes };
    case "registrar-contrato":
      return { expediente: objetivo, bolsa: "Bolsa DEMO", acto: "Relación temporal DEMO", inicio: "Pendiente", fin: "Pendiente", ...comunes };
    case "generar-documento":
      return { referencia: objetivo, plantilla: campos.plantilla || "Plantilla DEMO", formatos: campos.formato || "Pendiente", version: "v1 DEMO", ...comunes };
    case "preparar-comunicacion":
      return { id: objetivo, expediente: campos.expediente || "DEMO-EXP-SIN-ASIGNAR", plantilla: campos.plantilla || "Plantilla DEMO", canal: campos.canales || "Sin canal", destinatario: "DEMO-PERSONA-DESTINO", acuse: "No emitido", ...comunes };
    case "crear-rol":
      return { id: objetivo, nombre: campos.nombre || "Rol DEMO propuesto", ambito: campos.recursos || "Sin ámbito", permisos: campos.acciones || "Sin permisos", segregacion: "Pendiente de revisión", ...comunes };
    case "guardar-configuracion":
      return { id: objetivo, parametro: "Configuración DEMO", valor: "Pendiente de definir", version: "v1", ...comunes };
    default:
      throw new Error("la operación no admite crear el objetivo");
  }
}

function aplicarCampos(operacion, registro, campos) {
  registro.ultima_entrada = copia(campos);
  if (operacion === "guardar-bases" && Object.keys(campos).length > 0) {
    registro.nombre = campos.denominacion || registro.nombre;
    registro.expediente = campos.expediente || registro.expediente;
    registro.version_bases = campos.version_bases || registro.version_bases;
    registro.calendario = campos.apertura && campos.cierre ? `${campos.apertura} — ${campos.cierre}` : registro.calendario;
    registro.reglas = campos.version_bases ? `${campos.version_bases} · configuración guardada` : registro.reglas;
  }
  if (operacion === "guardar-reglas-baremo" && Object.keys(campos).length > 0) {
    registro.formula = `${campos.puntos_unidad || "0"} puntos por ${campos.unidad_tiempo || "unidad"}; ${campos.fraccion_jornada || "sin regla de jornada"}`;
    registro.maximo = campos.tope_bloque || registro.maximo;
    registro.version = "v4 DEMO";
  }
  if (["aceptar-merito", "rechazar-merito", "revocar-merito", "rehabilitar-merito"].includes(operacion)) {
    registro.criterio_aplicado = campos.criterio || registro.criterio_aplicado;
    registro.motivacion = campos.motivo_tipificado || registro.motivacion;
    registro.observacion = campos.observacion || registro.observacion;
  }
}

function aplicarOperacion(datos, operacion, objetivo, definicion, campos, actor) {
  if (definicion.estrategia === "auditar") return;
  const coleccion = datos[definicion.coleccion];
  if (!Array.isArray(coleccion)) throw new Error("colección de presentación no disponible");
  if (definicion.estrategia === "conjunto") {
    if (coleccion.length === 0) throw new Error("no existe un conjunto sobre el que operar");
    for (const registro of coleccion) {
      registro.estado = definicion.estado;
      aplicarCampos(operacion, registro, campos);
    }
    return;
  }
  let registro = localizar(coleccion, objetivo);
  if (definicion.estrategia === "crear" && registro) throw new Error("el objetivo sintético ya existe");
  if (definicion.estrategia === "actualizar" && !registro) throw new Error("el objetivo sintético no existe");
  if (!registro) {
    registro = crearRegistro(operacion, objetivo, campos, definicion.estado, actor);
    coleccion.unshift(registro);
  } else {
    aplicarCampos(operacion, registro, campos);
    registro.estado = definicion.estado;
  }
}

export function crearAdaptadorPresentacion({ datosIniciales, reloj = () => new Date() } = {}) {
  if (!datosIniciales || datosIniciales.demostracion !== true || typeof reloj !== "function") {
    throw new TypeError("configuración del adaptador de presentación no válida");
  }
  const datos = copia(datosIniciales);
  const actor = String(datos.sesion?.actor_ref || "");
  if (!/^DEMO-PERFIL-[A-Z0-9-]{2,100}$/.test(actor)) {
    throw new TypeError("identidad del actor de presentación no válida");
  }
  const operacionesPermitidas = Array.isArray(datos.sesion?.operaciones_permitidas)
    ? new Set(datos.sesion.operaciones_permitidas) : new Set();
  let secuencia = 0;

  function describir(operacion, objetivo) {
    const definicion = OPERACIONES[operacion];
    if (!definicion) return null;
    return Object.freeze({
      actor,
      efecto: definicion.efecto,
      objetivo: String(objetivo || "DEMO-SIN-OBJETIVO"),
    });
  }

  function ejecutar({ operacion, objetivo, motivo = "Actuación solicitada durante la presentación", campos } = {}) {
    const definicion = OPERACIONES[operacion];
    if (!definicion) throw new Error("operación de presentación no permitida");
    if (!operacionesPermitidas.has("*") && !operacionesPermitidas.has(operacion)) {
      throw new Error("el perfil de presentación no permite la operación");
    }
    const objetivoSeguro = String(objetivo || "DEMO-SIN-OBJETIVO");
    if (!/^DEMO-[A-Z0-9-]{2,120}$/.test(objetivoSeguro)) throw new Error("objetivo de presentación no válido");
    const camposSeguros = validarCampos(definicion, campos);
    const instante = reloj();
    if (!(instante instanceof Date) || !Number.isFinite(instante.getTime())) {
      throw new Error("reloj de presentación no válido");
    }
    aplicarOperacion(datos, operacion, objetivoSeguro, definicion, camposSeguros, actor);
    secuencia += 1;
    const recibo = Object.freeze({
      referencia: `DEMO-REC-${String(secuencia).padStart(6, "0")}`,
      actor,
      instante: instante.toISOString(),
      operacion,
      objetivo: objetivoSeguro,
      resultado: definicion.estado,
      motivo: String(motivo || "Motivo no indicado"),
      campos_aplicados: Object.keys(camposSeguros).length,
      efectos_reales: false,
    });
    if (definicion.coleccion && definicion.estrategia !== "conjunto") {
      const registro = localizar(datos[definicion.coleccion], objetivoSeguro);
      if (registro) registro.ultimo_recibo = recibo.referencia;
    } else if (definicion.estrategia === "conjunto") {
      for (const registro of datos[definicion.coleccion]) registro.ultimo_recibo = recibo.referencia;
    }
    datos.actividad.unshift({
      accion: definicion.efecto,
      objeto: objetivoSeguro,
      actor,
      fecha: instante.toLocaleString("es-ES", { timeZone: "Europe/Madrid" }),
      recibo: recibo.referencia,
    });
    datos.auditoria_eventos.unshift(recibo);
    return recibo;
  }

  return Object.freeze({
    actor,
    describir,
    ejecutar,
    obtenerDatos: () => copia(datos),
  });
}

export const OPERACIONES_PRESENTACION = Object.freeze(Object.keys(OPERACIONES));
