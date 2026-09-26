/**
 * Contratos cerrados para las lecturas autorizadas de Bolsas de trabajo (B12 y B5).
 *
 * Esquemas:
 * - vec.bolsa.rrhh.bolsas.v1
 * - vec.bolsa.rrhh.candidatos.v1
 *
 * Reglas de privacidad e integridad:
 * - Los documentos viajan siempre enmascarados (***1234**).
 * - Se rechaza cualquier documento sin enmascarar (DNI/NIE), correo electrónico o teléfono.
 * - Vocabulario cerrado de SituacionParticipacionBolsa:
 *   disponible, no_disponible, trabajando, pendiente_incorporacion, renuncia,
 *   excluido y disponible_desde.
 * - Contratos estrictos y cerrados: cualquier propiedad no declarada invalida la respuesta.
 */

import { validarMarcasCandidato } from "./portal-bolsas-marcas.js?v=20260926-huecos-rrhh-v1";

export const ESQUEMA_BOLSAS = "vec.bolsa.rrhh.bolsas.v1";
export const ESQUEMA_CANDIDATOS = "vec.bolsa.rrhh.candidatos.v1";
export const ESQUEMA_CONTACTOS = "vec.bolsa.rrhh.contactos.v1";
export const ESQUEMA_ESTADISTICAS = "vec.bolsa.rrhh.estadisticas.v1";
export const ESQUEMA_ACCION_BOLSA = "vec.bolsa.rrhh.accion.v1";

export const SITUACIONES_PARTICIPACION_BOLSA = Object.freeze([
  "disponible",
  "no_disponible",
  "trabajando",
  "pendiente_incorporacion",
  "renuncia",
  "excluido",
  "disponible_desde",
]);

export const CANALES_LLAMAMIENTO = Object.freeze([
  "correo",
  "telefono",
  "sede",
]);
export const CANALES_CONTACTO = Object.freeze(["telefono", "correo", "sms", "presencial", "otro"]);
export const RESULTADOS_CONTACTO = Object.freeze(["contactado", "no_contesta", "buzon", "acepta", "rechaza", "aplazado", "otro", "enviado", "no_enviado"]);

export const RESULTADOS_LLAMAMIENTO_BOLSA = Object.freeze([
  "aceptado",
  "renuncia",
  "sin_respuesta",
  "pendiente",
]);

export const RESULTADOS_REGISTRO_LLAMAMIENTO = Object.freeze([
  "aceptado",
  "renuncia",
  "sin_respuesta",
]);

const CAMPOS_BOLSA = Object.freeze([
  "bolsa_ref",
  "categoria_clave",
  "categoria",
  "tipo_lista",
  "vigente_desde",
  "vigente_hasta",
  "total",
  "por_estado",
  "llamamientos_en_curso",
  "politica_orden",
]);

const CAMPOS_POLITICA_ORDEN = Object.freeze([
  "politica_ref", "version", "criterio", "tipo_lista", "reposicion",
  "provisional", "rotulo", "actor", "vigente_desde",
]);

const CAMPOS_CANDIDATO = Object.freeze([
  "participacion_ref",
  "orden",
  "orden_acta",
  "razon_orden",
  "nombre_visible",
  "documento_enmascarado",
  "estado_clave",
  "estado_desde",
  "disponible_desde",
  "ultimo_llamamiento",
  "contactos_total",
]);

const CAMPOS_ULTIMO_LLAMAMIENTO = Object.freeze([
  "llamamiento_ref",
  "comunicado_en",
  "canal",
  "resultado",
]);

const CAMPOS_CONTACTO = Object.freeze([
  "contacto_ref",
  "participacion_ref",
  "llamamiento_ref",
  "canal",
  "instante",
  "actor_ref",
  "resultado",
  "anotacion",
]);

const PATRON_ENMASCARADO = /^\*{3}\d{4}\*{2}$/;
const PATRON_DNI = /\b\d{8}[A-Za-z]\b/;
const PATRON_NIE = /\b[XYZxyz]\d{7}[A-Za-z]\b/;
// Espejo de ReferenciaPropiaSistema del dominio de Bolsa: las referencias que
// emite el sistema (espacio de nombres alfabético y huella SHA-256 en
// hexadecimal) no pueden llevar un documento escrito por una persona, pero sus
// cifras casan por azar con los patrones de DNI o teléfono.
const REFERENCIA_PROPIA_SISTEMA = /^[a-z_]+(?::[a-z_]+)*:[0-9a-f]{64}$/;
const PATRON_EMAIL = /[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}/;
const PATRON_TELEFONO = /(?:\+?34[\s.-]?)?[6789]\d{2}[\s.-]?\d{3}[\s.-]?\d{3}\b/;

function esObjeto(valor) {
  return valor !== null && typeof valor === "object" && !Array.isArray(valor);
}

function exigirCamposExactos(objeto, campos, nombre) {
  if (!esObjeto(objeto)) throw new Error(`${nombre} no es un objeto válido`);
  const recibidos = Object.keys(objeto);
  if (recibidos.length !== campos.length || campos.some((campo) => !Object.hasOwn(objeto, campo))) {
    throw new Error(`${nombre} no respeta el contrato cerrado`);
  }
}

function contieneDatosPersonalesSensibles(cadena) {
  if (typeof cadena !== "string" || REFERENCIA_PROPIA_SISTEMA.test(cadena)) return false;
  return PATRON_DNI.test(cadena)
    || PATRON_NIE.test(cadena)
    || PATRON_EMAIL.test(cadena)
    || PATRON_TELEFONO.test(cadena);
}

function exigirCadenaSegura(valor, nombre, { maximo = 512, admiteVacia = false } = {}) {
  if (typeof valor !== "string") throw new Error(`${nombre} debe ser una cadena`);
  if (!admiteVacia && valor.trim() === "") throw new Error(`${nombre} no puede estar vacía`);
  if (valor.length > maximo) throw new Error(`${nombre} supera el límite máximo de caracteres`);
  if (contieneDatosPersonalesSensibles(valor)) {
    throw new Error(`${nombre} contiene datos personales no permitidos`);
  }
  return valor;
}

function exigirEnteroNoNegativo(valor, nombre, maximo = 1_000_000) {
  if (!Number.isSafeInteger(valor) || valor < 0 || valor > maximo) {
    throw new Error(`${nombre} debe ser un entero no negativo válido`);
  }
  return valor;
}

function exigirInstanteUTC(valor, nombre) {
  if (typeof valor !== "string" || valor.trim() === "") throw new Error(`${nombre} debe ser un instante válido`);
  const parseado = Date.parse(valor);
  if (!Number.isFinite(parseado)) throw new Error(`${nombre} no es una fecha válida`);
  if (contieneDatosPersonalesSensibles(valor)) throw new Error(`${nombre} contiene datos personales no permitidos`);
  return valor;
}

function exigirFechaOInstante(valor, nombre) {
  if (typeof valor !== "string" || valor.trim() === "") throw new Error(`${nombre} no es una fecha válida`);
  const esFecha = /^\d{4}-\d{2}-\d{2}$/.test(valor);
  const esInstante = Number.isFinite(Date.parse(valor));
  if (!esFecha && !esInstante) throw new Error(`${nombre} no tiene formato de fecha o instante válido`);
  return valor;
}

export function validarDocumentoEnmascarado(valor, nombre = "documento_enmascarado") {
  if (typeof valor !== "string" || !PATRON_ENMASCARADO.test(valor)) {
    throw new Error(`${nombre} debe estar enmascarado con formato ***1234**`);
  }
  return valor;
}

export function extraerDatosEnvelopeCanonico(envelope) {
  if (!esObjeto(envelope) || !Object.hasOwn(envelope, "data") || !esObjeto(envelope.data)) {
    throw new Error("la API debe responder con el envelope canónico {data:{...}}");
  }
  if (Object.hasOwn(envelope, "esquema") || Object.hasOwn(envelope, "bolsas") || Object.hasOwn(envelope, "candidatos")) {
    throw new Error("se ha rechazado una respuesta raw en la raíz");
  }
  return envelope.data;
}

export function validarBolsa(bolsa) {
  exigirCamposExactos(bolsa, CAMPOS_BOLSA, "bolsa");

  const bolsaRef = exigirCadenaSegura(bolsa.bolsa_ref, "bolsa_ref");
  const categoriaClave = exigirCadenaSegura(bolsa.categoria_clave, "categoria_clave");
  const categoria = exigirCadenaSegura(bolsa.categoria, "categoria");
  const tipoLista = exigirCadenaSegura(bolsa.tipo_lista, "tipo_lista");
  const vigenteDesde = exigirFechaOInstante(bolsa.vigente_desde, "vigente_desde");
  const vigenteHasta = bolsa.vigente_hasta === null ? null : exigirFechaOInstante(bolsa.vigente_hasta, "vigente_hasta");
  const total = exigirEnteroNoNegativo(bolsa.total, "total");
	const llamamientosEnCurso = exigirEnteroNoNegativo(bolsa.llamamientos_en_curso, "llamamientos_en_curso");
	if (bolsa.politica_orden === null) throw new Error("politica_orden es obligatoria");
	exigirCamposExactos(bolsa.politica_orden, CAMPOS_POLITICA_ORDEN, "politica_orden");
	const politicaOrden = Object.freeze({
	  politica_ref: exigirCadenaSegura(bolsa.politica_orden.politica_ref, "politica_ref"),
	  version: exigirEnteroNoNegativo(bolsa.politica_orden.version, "version"),
	  criterio: bolsa.politica_orden.criterio,
	  tipo_lista: bolsa.politica_orden.tipo_lista,
	  reposicion: bolsa.politica_orden.reposicion,
	  provisional: bolsa.politica_orden.provisional,
	  rotulo: exigirCadenaSegura(bolsa.politica_orden.rotulo, "rotulo"),
	  actor: exigirCadenaSegura(bolsa.politica_orden.actor, "actor"),
	  vigente_desde: exigirFechaOInstante(bolsa.politica_orden.vigente_desde, "politica vigente_desde"),
	});
	if (politicaOrden.version < 1 || politicaOrden.criterio !== "puntuacion_desc_acta" || !["cerrada","rotatoria"].includes(politicaOrden.tipo_lista) || !["misma_posicion","fin_lista","no_disponible_hasta_fecha"].includes(politicaOrden.reposicion) || typeof politicaOrden.provisional !== "boolean") throw new Error("politica_orden no es valida");
	if (politicaOrden.provisional && politicaOrden.rotulo !== "Provisional, pendiente de RRHH (dudas 13–14)") throw new Error("politica provisional sin rotulo obligatorio");

  exigirCamposExactos(bolsa.por_estado, SITUACIONES_PARTICIPACION_BOLSA, "por_estado");
  const porEstado = {};
  for (const estado of SITUACIONES_PARTICIPACION_BOLSA) {
    porEstado[estado] = exigirEnteroNoNegativo(bolsa.por_estado[estado], `por_estado.${estado}`);
  }

  return Object.freeze({
    bolsa_ref: bolsaRef,
    categoria_clave: categoriaClave,
    categoria,
    tipo_lista: tipoLista,
    vigente_desde: vigenteDesde,
    vigente_hasta: vigenteHasta,
    total,
    por_estado: Object.freeze(porEstado),
	llamamientos_en_curso: llamamientosEnCurso,
	politica_orden: politicaOrden,
  });
}

export function validarRespuestaBolsas(envelope) {
  const datos = extraerDatosEnvelopeCanonico(envelope);
  exigirCamposExactos(datos, ["esquema", "generado_en", "bolsas"], "respuesta de bolsas");

  if (datos.esquema !== ESQUEMA_BOLSAS) {
    throw new Error(`esquema no compatible: ${datos.esquema}`);
  }
  const generadoEn = exigirInstanteUTC(datos.generado_en, "generado_en");
  if (!Array.isArray(datos.bolsas)) {
    throw new Error("el campo bolsas debe ser un array");
  }

  const bolsas = Object.freeze(datos.bolsas.map(validarBolsa));
  return Object.freeze({
    esquema: datos.esquema,
    generado_en: generadoEn,
    bolsas,
  });
}

export function validarCandidato(candidato) {
  // «marcas» solo viaja cuando Bolsa 000041 está compuesta; entonces es cerrado.
  const conMarcas = esObjeto(candidato) && Object.hasOwn(candidato, "marcas");
  exigirCamposExactos(candidato, conMarcas ? [...CAMPOS_CANDIDATO, "marcas"] : CAMPOS_CANDIDATO, "candidato");

  const participacionRef = exigirCadenaSegura(candidato.participacion_ref, "participacion_ref");
  if (candidato.orden !== null && (!Number.isSafeInteger(candidato.orden) || candidato.orden < 1)) throw new Error("orden de candidato debe ser nulo o entero positivo");
  if (!Number.isSafeInteger(candidato.orden_acta) || candidato.orden_acta < 1) throw new Error("orden_acta debe ser entero positivo");
  if (!["orden_acta","reposicion_tras_contrato","pausa","trabajando","sin_turno","sancion_al_final","adelanta_por_sancion"].includes(candidato.razon_orden)) throw new Error("razon_orden no reconocida");
  const nombreVisible = exigirCadenaSegura(candidato.nombre_visible, "nombre_visible");
  const documentoEnmascarado = validarDocumentoEnmascarado(candidato.documento_enmascarado);

  if (!SITUACIONES_PARTICIPACION_BOLSA.includes(candidato.estado_clave)) {
    throw new Error(`estado_clave no reconocido en el catálogo: ${candidato.estado_clave}`);
  }
  const estadoClave = candidato.estado_clave;
  const estadoDesde = exigirFechaOInstante(candidato.estado_desde, "estado_desde");
  const disponibleDesde = candidato.disponible_desde === null
    ? null
    : exigirFechaOInstante(candidato.disponible_desde, "disponible_desde");

  let ultimoLlamamiento = null;
  if (candidato.ultimo_llamamiento !== null) {
    exigirCamposExactos(candidato.ultimo_llamamiento, CAMPOS_ULTIMO_LLAMAMIENTO, "ultimo_llamamiento");
    ultimoLlamamiento = Object.freeze({
      llamamiento_ref: exigirCadenaSegura(candidato.ultimo_llamamiento.llamamiento_ref, "llamamiento_ref"),
      comunicado_en: exigirInstanteUTC(candidato.ultimo_llamamiento.comunicado_en, "comunicado_en"),
      canal: exigirCadenaSegura(candidato.ultimo_llamamiento.canal, "canal"),
      resultado: exigirCadenaSegura(candidato.ultimo_llamamiento.resultado, "resultado"),
    });
  }

  return Object.freeze({
    participacion_ref: participacionRef,
    orden: candidato.orden,
	orden_acta: candidato.orden_acta,
	razon_orden: candidato.razon_orden,
    nombre_visible: nombreVisible,
    documento_enmascarado: documentoEnmascarado,
    estado_clave: estadoClave,
    estado_desde: estadoDesde,
    disponible_desde: disponibleDesde,
    ultimo_llamamiento: ultimoLlamamiento,
    contactos_total: exigirEnteroNoNegativo(candidato.contactos_total, "contactos_total"),
    ...(conMarcas ? { marcas: validarMarcasCandidato(candidato.marcas) } : {}),
  });
}

export function validarRespuestaCandidatosBolsa(envelope) {
  const datos = extraerDatosEnvelopeCanonico(envelope);
  exigirCamposExactos(datos, ["esquema", "generado_en", "bolsa", "candidatos", "contactos", "hay_mas", "cursor_siguiente"], "respuesta de candidatos");

  if (datos.esquema !== ESQUEMA_CANDIDATOS) {
    throw new Error(`esquema no compatible: ${datos.esquema}`);
  }
  const generadoEn = exigirInstanteUTC(datos.generado_en, "generado_en");
  const bolsa = validarBolsa(datos.bolsa);

  if (!Array.isArray(datos.candidatos)) {
    throw new Error("el campo candidatos debe ser un array");
  }
  const candidatos = Object.freeze(datos.candidatos.map(validarCandidato));
  if (!Array.isArray(datos.contactos)) throw new Error("el campo contactos debe ser un array");
  const contactos = Object.freeze(datos.contactos.map(validarContacto));

  if (typeof datos.hay_mas !== "boolean") {
    throw new Error("hay_mas debe ser un booleano");
  }

  let cursorSiguiente = null;
  if (datos.hay_mas) {
    if (typeof datos.cursor_siguiente !== "string" || datos.cursor_siguiente.trim() === "") {
      throw new Error("cursor_siguiente debe ser una cadena no vacía cuando hay_mas es true");
    }
    cursorSiguiente = datos.cursor_siguiente.trim();
  } else if (datos.cursor_siguiente !== null) {
    throw new Error("cursor_siguiente debe ser null cuando hay_mas es false");
  }

  return Object.freeze({
    esquema: datos.esquema,
    generado_en: generadoEn,
    bolsa,
    candidatos,
    contactos,
    hay_mas: datos.hay_mas,
    cursor_siguiente: cursorSiguiente,
  });
}

export function validarContacto(contacto) {
  exigirCamposExactos(contacto, CAMPOS_CONTACTO, "contacto");

  const contactoRef = exigirCadenaSegura(contacto.contacto_ref, "contacto_ref");
  if (!CANALES_CONTACTO.includes(contacto.canal)) {
    throw new Error(`canal de contacto no reconocido: ${contacto.canal}`);
  }
  const canal = contacto.canal;
  const instante = exigirInstanteUTC(contacto.instante, "instante");
  if (!RESULTADOS_CONTACTO.includes(contacto.resultado)) {
    throw new Error(`resultado de contacto no reconocido: ${contacto.resultado}`);
  }
  const resultado = contacto.resultado;
  const anotacion = exigirCadenaSegura(contacto.anotacion, "anotacion", { maximo: 1000 });

  return Object.freeze({
    contacto_ref: contactoRef,
    participacion_ref: exigirCadenaSegura(contacto.participacion_ref, "participacion_ref"),
    llamamiento_ref: contacto.llamamiento_ref === null ? null : exigirCadenaSegura(contacto.llamamiento_ref, "llamamiento_ref"),
    canal,
    instante,
    actor_ref: exigirCadenaSegura(contacto.actor_ref, "actor_ref"),
    resultado,
    anotacion,
  });
}

export function validarRespuestaContactos(envelope) {
  const datos = extraerDatosEnvelopeCanonico(envelope);
  if (datos.esquema && datos.esquema !== ESQUEMA_CONTACTOS) {
    throw new Error(`esquema no compatible: ${datos.esquema}`);
  }
  if (!Array.isArray(datos.contactos)) {
    throw new Error("el campo contactos debe ser un array");
  }
  const contactos = Object.freeze(datos.contactos.map(validarContacto));
  const generadoEn = datos.generado_en ? exigirInstanteUTC(datos.generado_en, "generado_en") : null;
  const participacionRef = datos.participacion_ref ? exigirCadenaSegura(datos.participacion_ref, "participacion_ref") : null;

  return Object.freeze({
    esquema: datos.esquema || ESQUEMA_CONTACTOS,
    generado_en: generadoEn,
    participacion_ref: participacionRef,
    contactos,
  });
}

function validarMapaConteos(mapa, nombre, claves = null) {
  if (!esObjeto(mapa)) throw new Error(`${nombre} debe ser un objeto`);
  const recibidas = Object.keys(mapa);
  if (claves && (recibidas.length !== claves.length || claves.some((clave) => !Object.hasOwn(mapa, clave)))) throw new Error(`${nombre} no respeta el contrato cerrado`);
  const salida = {};
  for (const clave of recibidas) {
    if (!clave.trim() || contieneDatosPersonalesSensibles(clave)) throw new Error(`${nombre} contiene una clave no válida`);
    salida[clave] = exigirEnteroNoNegativo(mapa[clave], `${nombre}.${clave}`);
  }
  return Object.freeze(salida);
}

export function validarRespuestaEstadisticas(envelope) {
  const datos = extraerDatosEnvelopeCanonico(envelope);
  exigirCamposExactos(datos, ["esquema", "generado_en", "bolsas", "personas", "llamamientos", "por_bolsa"], "respuesta de estadísticas");
  if (datos.esquema !== ESQUEMA_ESTADISTICAS) throw new Error(`esquema no compatible: ${datos.esquema}`);
  exigirCamposExactos(datos.bolsas, ["total", "vigentes", "sustituidas"], "estadísticas de bolsas");
  exigirCamposExactos(datos.personas, ["total", "por_estado"], "estadísticas de personas");
  exigirCamposExactos(datos.llamamientos, ["total", "por_canal", "por_resultado"], "estadísticas de llamamientos");
  if (!Array.isArray(datos.por_bolsa)) throw new Error("por_bolsa debe ser un array");
  const por_bolsa = Object.freeze(datos.por_bolsa.map((bolsa) => {
    exigirCamposExactos(bolsa, ["bolsa_ref", "categoria", "tipo_lista", "vigente", "total", "por_estado"], "estadística por bolsa");
    if (typeof bolsa.vigente !== "boolean") throw new Error("vigente debe ser booleano");
    return Object.freeze({ bolsa_ref: exigirCadenaSegura(bolsa.bolsa_ref, "bolsa_ref"), categoria: exigirCadenaSegura(bolsa.categoria, "categoria"), tipo_lista: exigirCadenaSegura(bolsa.tipo_lista, "tipo_lista"), vigente: bolsa.vigente, total: exigirEnteroNoNegativo(bolsa.total, "total"), por_estado: validarMapaConteos(bolsa.por_estado, "por_estado", SITUACIONES_PARTICIPACION_BOLSA) });
  }));
  return Object.freeze({
    esquema: datos.esquema, generado_en: exigirInstanteUTC(datos.generado_en, "generado_en"),
    bolsas: Object.freeze({ total: exigirEnteroNoNegativo(datos.bolsas.total, "bolsas.total"), vigentes: exigirEnteroNoNegativo(datos.bolsas.vigentes, "bolsas.vigentes"), sustituidas: exigirEnteroNoNegativo(datos.bolsas.sustituidas, "bolsas.sustituidas") }),
    personas: Object.freeze({ total: exigirEnteroNoNegativo(datos.personas.total, "personas.total"), por_estado: validarMapaConteos(datos.personas.por_estado, "personas.por_estado", SITUACIONES_PARTICIPACION_BOLSA) }),
    llamamientos: Object.freeze({ total: exigirEnteroNoNegativo(datos.llamamientos.total, "llamamientos.total"), por_canal: validarMapaConteos(datos.llamamientos.por_canal, "llamamientos.por_canal"), por_resultado: validarMapaConteos(datos.llamamientos.por_resultado, "llamamientos.por_resultado") }),
    por_bolsa,
  });
}

export function validarPayloadCrearLlamamiento(payload) {
  if (!esObjeto(payload)) throw new Error("el payload de crear llamamiento debe ser un objeto");

  if (!CANALES_LLAMAMIENTO.includes(payload.canal)) {
    throw new Error(`canal de llamamiento no válido: ${payload.canal}`);
  }
  const canal = payload.canal;
  const comunicadoEn = exigirInstanteUTC(payload.comunicado_en, "comunicado_en");
  const plazoRespuestaHasta = exigirInstanteUTC(payload.plazo_respuesta_hasta, "plazo_respuesta_hasta");

  let anotacion = "";
  if (payload.anotacion !== null && payload.anotacion !== undefined) {
    anotacion = exigirCadenaSegura(payload.anotacion, "anotacion", { maximo: 1024, admiteVacia: true });
  }

  return Object.freeze({
    canal,
    comunicado_en: comunicadoEn,
    plazo_respuesta_hasta: plazoRespuestaHasta,
    anotacion,
  });
}

export function validarPayloadResultadoLlamamiento(payload) {
  if (!esObjeto(payload)) throw new Error("el payload de resultado de llamamiento debe ser un objeto");

  if (!RESULTADOS_REGISTRO_LLAMAMIENTO.includes(payload.resultado_clave)) {
    throw new Error(`resultado_clave no válido: ${payload.resultado_clave}`);
  }
  const resultadoClave = payload.resultado_clave;

  let anotacion = "";
  if (payload.anotacion !== null && payload.anotacion !== undefined) {
    anotacion = exigirCadenaSegura(payload.anotacion, "anotacion", { maximo: 1024, admiteVacia: true });
  }

  return Object.freeze({
    resultado_clave: resultadoClave,
    anotacion,
  });
}

export function construirEnvelopeAccionBolsa(accion, payload, { confirmacion = true } = {}) {
  if (typeof accion !== "string" || accion.trim() === "") {
    throw new Error("accion debe ser una cadena no vacía");
  }
  if (confirmacion !== true) {
    throw new Error("la acción requiere confirmación explícita (confirmacion: true)");
  }
  return Object.freeze({
    esquema: ESQUEMA_ACCION_BOLSA,
    accion: accion.trim(),
    confirmacion: true,
    payload: Object.freeze({ ...payload }),
  });
}
