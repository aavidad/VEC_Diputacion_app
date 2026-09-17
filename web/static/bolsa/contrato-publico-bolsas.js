/**
 * Contrato estricto para la consulta pública de bolsas y listas de aspirantes (B10).
 * Esquemas: vec.bolsa.publico.bolsas.v1 y vec.bolsa.publico.lista.v1.
 *
 * REGLAS DE PRIVACIDAD:
 * - Sin identificación personal: solo orden, documento enmascarado y situación.
 * - Sin nombres, teléfonos, correos ni datos de contacto.
 * - Solo documentos enmascarados canónicos con formato exacto: ***1234**
 * - Rechazo fail-closed de claves no declaradas o datos ambientales.
 */

const ESQUEMA_BOLSAS_PUBLICAS = "vec.bolsa.publico.bolsas.v1";
const ESQUEMA_LISTA_PUBLICA = "vec.bolsa.publico.lista.v1";

const SITUACIONES_PERMITIDAS = Object.freeze(new Set([
  "disponible",
  "ocupado",
  "no_disponible",
  "excluido",
  "renuncia_pendiente",
]));

const CLAVES_PERMITIDAS_BOLSA = Object.freeze(new Set([
  "bolsa_ref",
  "categoria",
  "grupos",
  "tipo_lista",
  "vigente_desde",
  "vigente_hasta",
  "total",
]));

const CLAVES_PERMITIDAS_POSICION = Object.freeze(new Set([
  "orden",
  "documento_enmascarado",
  "estado_clave",
]));

export const PATRON_DOCUMENTO_ENMASCARADO = /^\*{3}\d{4}\*{2}$/;
export const PATRON_LEAK_DNI = /\b\d{8}[A-Za-z]\b|\b[XYZxyz]\d{7}[A-Za-z]\b/;
export const PATRON_LEAK_CORREO = /[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}/;
export const PATRON_LEAK_TELEFONO = /\b(?:\+34|0034)?[6789]\d{8}\b/;

function comprobarSinFugasTexto(valor, contexto) {
  if (typeof valor !== "string") return;
  if (PATRON_LEAK_DNI.test(valor)) {
    throw new Error(`Fuga de datos detectada en ${contexto}: contiene DNI/NIE sin enmascarar`);
  }
  if (PATRON_LEAK_CORREO.test(valor)) {
    throw new Error(`Fuga de datos detectada en ${contexto}: contiene dirección de correo`);
  }
  if (PATRON_LEAK_TELEFONO.test(valor)) {
    throw new Error(`Fuga de datos detectada en ${contexto}: contiene número de teléfono`);
  }
}

function validarBolsaPublica(bolsa, indice = 0) {
  if (!bolsa || typeof bolsa !== "object" || Array.isArray(bolsa)) {
    throw new Error(`La bolsa en el índice ${indice} no es un objeto válido`);
  }
  for (const clave of Object.keys(bolsa)) {
    if (!CLAVES_PERMITIDAS_BOLSA.has(clave)) {
      throw new Error(`Propiedad no permitida en bolsa [${indice}]: ${clave}`);
    }
  }

  if (typeof bolsa.bolsa_ref !== "string" || bolsa.bolsa_ref.trim() === "") {
    throw new Error(`bolsa_ref inválida en bolsa [${indice}]`);
  }
  if (typeof bolsa.categoria !== "string" || bolsa.categoria.trim() === "") {
    throw new Error(`categoria inválida en bolsa [${indice}]`);
  }
  comprobarSinFugasTexto(bolsa.categoria, `bolsa[${indice}].categoria`);

  if (!Array.isArray(bolsa.grupos)) {
    throw new Error(`grupos inválidos en bolsa [${indice}]`);
  }
  for (const g of bolsa.grupos) {
    if (typeof g !== "string" || g.trim() === "") {
      throw new Error(`grupo no válido en bolsa [${indice}]`);
    }
  }

  if (typeof bolsa.tipo_lista !== "string" || bolsa.tipo_lista.trim() === "") {
    throw new Error(`tipo_lista inválido en bolsa [${indice}]`);
  }

  if (typeof bolsa.vigente_desde !== "string" || isNaN(Date.parse(bolsa.vigente_desde))) {
    throw new Error(`vigente_desde no es una fecha válida en bolsa [${indice}]`);
  }
  if (bolsa.vigente_hasta !== null && (typeof bolsa.vigente_hasta !== "string" || isNaN(Date.parse(bolsa.vigente_hasta)))) {
    throw new Error(`vigente_hasta no es una fecha válida ni null en bolsa [${indice}]`);
  }

  if (typeof bolsa.total !== "number" || !Number.isInteger(bolsa.total) || bolsa.total < 0) {
    throw new Error(`total inválido en bolsa [${indice}]: debe ser un entero no negativo`);
  }

  return Object.freeze({
    bolsa_ref: bolsa.bolsa_ref,
    categoria: bolsa.categoria,
    grupos: Object.freeze([...bolsa.grupos]),
    tipo_lista: bolsa.tipo_lista,
    vigente_desde: bolsa.vigente_desde,
    vigente_hasta: bolsa.vigente_hasta,
    total: bolsa.total,
  });
}

function validarPosicionPublica(posicion, indice = 0) {
  if (!posicion || typeof posicion !== "object" || Array.isArray(posicion)) {
    throw new Error(`La posición en el índice ${indice} no es un objeto válido`);
  }
  for (const clave of Object.keys(posicion)) {
    if (!CLAVES_PERMITIDAS_POSICION.has(clave)) {
      throw new Error(`Propiedad no permitida en posición [${indice}]: ${clave}`);
    }
  }

  if (typeof posicion.orden !== "number" || !Number.isInteger(posicion.orden) || posicion.orden < 1) {
    throw new Error(`orden inválido en posición [${indice}]`);
  }

  if (typeof posicion.documento_enmascarado !== "string" || !PATRON_DOCUMENTO_ENMASCARADO.test(posicion.documento_enmascarado)) {
    throw new Error(`documento_enmascarado inválido en posición [${indice}]: debe cumplir formato ***1234**`);
  }
  comprobarSinFugasTexto(posicion.documento_enmascarado, `posicion[${indice}].documento_enmascarado`);

  if (typeof posicion.estado_clave !== "string" || !SITUACIONES_PERMITIDAS.has(posicion.estado_clave)) {
    throw new Error(`estado_clave inválido en posición [${indice}]: "${posicion.estado_clave}"`);
  }

  return Object.freeze({
    orden: posicion.orden,
    documento_enmascarado: posicion.documento_enmascarado,
    estado_clave: posicion.estado_clave,
  });
}

/**
 * Valida la respuesta del listado general de bolsas públicas.
 * GET /api/publico/bolsa/bolsas
 */
export function validarRespuestaBolsasPublicas(json) {
  if (!json || typeof json !== "object" || !json.data) {
    throw new Error("Respuesta de bolsas públicas no contiene sobre de datos (data)");
  }
  const { data } = json;
  if (data.esquema !== ESQUEMA_BOLSAS_PUBLICAS) {
    throw new Error(`Esquema inválido: esperado ${ESQUEMA_BOLSAS_PUBLICAS}, recibido ${data.esquema}`);
  }
  if (typeof data.generado_en !== "string" || isNaN(Date.parse(data.generado_en))) {
    throw new Error("generado_en no es una fecha válida");
  }
  if (!Array.isArray(data.bolsas)) {
    throw new Error("bolsas debe ser una lista");
  }

  const bolsas = data.bolsas.map((b, i) => validarBolsaPublica(b, i));
  return Object.freeze({
    esquema: data.esquema,
    generado_en: data.generado_en,
    bolsas: Object.freeze(bolsas),
  });
}

/**
 * Valida la respuesta de posiciones de una bolsa pública concreta.
 * GET /api/publico/bolsa/bolsas/{bolsa_ref}/lista
 */
export function validarRespuestaListaPublica(json) {
  if (!json || typeof json !== "object" || !json.data) {
    throw new Error("Respuesta de lista pública no contiene sobre de datos (data)");
  }
  const { data } = json;
  if (data.esquema !== ESQUEMA_LISTA_PUBLICA) {
    throw new Error(`Esquema inválido: esperado ${ESQUEMA_LISTA_PUBLICA}, recibido ${data.esquema}`);
  }
  if (typeof data.generado_en !== "string" || isNaN(Date.parse(data.generado_en))) {
    throw new Error("generado_en no es una fecha válida");
  }

  const bolsa = validarBolsaPublica(data.bolsa, "cabecera");

  if (!Array.isArray(data.posiciones)) {
    throw new Error("posiciones debe ser una lista");
  }

  const posiciones = data.posiciones.map((p, i) => validarPosicionPublica(p, i));

  if (typeof data.hay_mas !== "boolean") {
    throw new Error("hay_mas debe ser un booleano");
  }

  const cursorSiguiente = data.cursor_siguiente !== undefined && data.cursor_siguiente !== null
    ? String(data.cursor_siguiente)
    : null;

  return Object.freeze({
    esquema: data.esquema,
    generado_en: data.generado_en,
    bolsa,
    posiciones: Object.freeze(posiciones),
    hay_mas: data.hay_mas,
    cursor_siguiente: cursorSiguiente,
  });
}

export const VECPublicoBolsasContrato = Object.freeze({
  ESQUEMA_BOLSAS_PUBLICAS,
  ESQUEMA_LISTA_PUBLICA,
  SITUACIONES_PERMITIDAS,
  PATRON_DOCUMENTO_ENMASCARADO,
  validarRespuestaBolsasPublicas,
  validarRespuestaListaPublica,
});

if (typeof globalThis !== "undefined") {
  globalThis.VECPublicoBolsasContrato = VECPublicoBolsasContrato;
}
