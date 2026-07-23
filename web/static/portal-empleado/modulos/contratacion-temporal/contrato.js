/** Contrato neutral y cerrado del alta de contratación temporal. */

export const CAPACIDAD_CREAR_SOLICITUD = "contratacion_temporal.solicitud.crear";

export const LIMITES_ALTA_CONTRATACION = Object.freeze({
  texto: 4000,
  adjuntos: 64,
  referencia: 160,
  etiquetaCatalogo: 200,
  opcionesCatalogo: 1000,
  opcionesCatalogoTotales: 5000,
});

const ESQUEMA_CATALOGOS = "vec.contratacion_temporal.catalogos_alta.v1";
const PATRON_REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/;
const PATRON_CLAVE_CATALOGO = /^[a-z][a-z0-9._-]{1,79}$/;
const PATRON_GRUPO = /^[A-Z][A-Z0-9/+.-]{0,19}$/;
const PATRON_NUMERO = /^[0-9]{4}\/[A-Za-z0-9._-]{1,40}$/;
const PATRON_IDEMPOTENCIA =
  /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;
const CLAVE_IDEMPOTENCIA_NULA = "00000000-0000-4000-8000-000000000000";

const CAMPOS_BORRADOR = Object.freeze([
  "centro_ref",
  "contacto_ref",
  "categoria_ref",
  "grupo_subgrupo",
  "motivo_clave",
  "detalle",
  "inicio",
  "fin",
  "rc_existe",
  "rc_numero",
  "rc_fecha",
  "rc_importe",
  "rc_documento_ref",
  "documentos_adjuntos",
  "observaciones",
]);

const CAMPOS_SOLICITUD = Object.freeze([
  "centro_ref",
  "contacto_ref",
  "categoria_ref",
  "grupo_subgrupo",
  "motivo_clave",
  "detalle",
  "periodo",
  "rc",
  "documentos_adjuntos",
  "observaciones",
]);

export class ErrorValidacionAlta extends Error {
  constructor(errores) {
    super("La solicitud no respeta el contrato de alta");
    this.name = "ErrorValidacionAlta";
    this.errores = congelar({ ...errores });
  }
}

function esRegistro(valor) {
  return valor !== null && typeof valor === "object" && !Array.isArray(valor);
}

function congelar(valor) {
  if (Array.isArray(valor)) {
    valor.forEach(congelar);
    return Object.freeze(valor);
  }
  if (esRegistro(valor)) {
    Object.values(valor).forEach(congelar);
    return Object.freeze(valor);
  }
  return valor;
}

function clonar(valor) {
  if (Array.isArray(valor)) return valor.map(clonar);
  if (esRegistro(valor)) {
    return Object.fromEntries(Object.entries(valor).map(([clave, dato]) => [clave, clonar(dato)]));
  }
  return valor;
}

export function clonarYCongelarAlta(valor) {
  return congelar(clonar(valor));
}

function tieneCamposExactos(registro, campos) {
  if (!esRegistro(registro)) return false;
  const claves = Object.keys(registro);
  return claves.length === campos.length
    && claves.every((clave) => campos.includes(clave))
    && campos.every((campo) => Object.hasOwn(registro, campo));
}

function exigirCamposExactos(registro, campos, nombre) {
  if (!tieneCamposExactos(registro, campos)) {
    throw new TypeError(`${nombre} no respeta el contrato cerrado`);
  }
}

function longitudUnicode(texto) {
  return [...texto].length;
}

function textoValido(valor, maximo, permiteVacio) {
  if (typeof valor !== "string" || valor !== valor.trim()
    || valor.normalize("NFC") !== valor || longitudUnicode(valor) > maximo
    || (!permiteVacio && valor === "")) {
    return false;
  }
  for (const caracter of valor) {
    const codigo = caracter.codePointAt(0);
    if ((codigo < 32 || (codigo >= 127 && codigo <= 159))
      && caracter !== "\n" && caracter !== "\t"
      || (codigo >= 0xD800 && codigo <= 0xDFFF)) {
      return false;
    }
  }
  return true;
}

function etiquetaValida(valor) {
  return textoValido(valor, LIMITES_ALTA_CONTRATACION.etiquetaCatalogo, false);
}

function fechaCivilValida(valor) {
  if (typeof valor !== "string" || !/^\d{4}-\d{2}-\d{2}$/.test(valor)) return false;
  const fecha = new Date(`${valor}T00:00:00Z`);
  return Number.isFinite(fecha.valueOf()) && fecha.toISOString().slice(0, 10) === valor;
}

function instanteCivilUTC(fecha) {
  return `${fecha}T00:00:00Z`;
}

function instanteUTCValido(valor) {
  if (typeof valor !== "string") {
    return false;
  }
  const partes =
    /^(\d{4}-\d{2}-\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.\d{1,6})?Z$/.exec(valor);
  if (!partes) return false;
  const [, fecha, hora, minuto, segundo] = partes;
  return fechaCivilValida(fecha)
    && Number(hora) <= 23
    && Number(minuto) <= 59
    && Number(segundo) <= 59;
}

function referenciaValida(valor) {
  return typeof valor === "string" && PATRON_REFERENCIA.test(valor);
}

function agregarError(errores, campo, codigo) {
  if (!Object.hasOwn(errores, campo)) errores[campo] = codigo;
}

function validarOpcionReferencia(opcion, nombre) {
  exigirCamposExactos(opcion, ["referencia", "etiqueta"], nombre);
  if (!referenciaValida(opcion.referencia) || !etiquetaValida(opcion.etiqueta)) {
    throw new TypeError(`${nombre} no válida`);
  }
  return { referencia: opcion.referencia, etiqueta: opcion.etiqueta };
}

function validarOpcionClave(opcion, nombre, patron = PATRON_CLAVE_CATALOGO) {
  exigirCamposExactos(opcion, ["clave", "etiqueta"], nombre);
  if (typeof opcion.clave !== "string" || !patron.test(opcion.clave)
    || !etiquetaValida(opcion.etiqueta)) {
    throw new TypeError(`${nombre} no válida`);
  }
  return { clave: opcion.clave, etiqueta: opcion.etiqueta };
}

function exigirUnicos(opciones, campo, nombre) {
  const valores = opciones.map((opcion) => opcion[campo]);
  if (new Set(valores).size !== valores.length) throw new TypeError(`${nombre} contiene duplicados`);
}

function validarLista(lista, nombre, validarOpcion) {
  if (!Array.isArray(lista) || lista.length > LIMITES_ALTA_CONTRATACION.opcionesCatalogo) {
    throw new TypeError(`${nombre} no válido`);
  }
  return lista.map((opcion, indice) => validarOpcion(opcion, `${nombre}[${indice}]`));
}

function validarCentro(centro, indice) {
  exigirCamposExactos(centro, ["referencia", "etiqueta", "contactos"], `centros[${indice}]`);
  if (!referenciaValida(centro.referencia) || !etiquetaValida(centro.etiqueta)) {
    throw new TypeError(`centros[${indice}] no válido`);
  }
  const contactos = validarLista(
    centro.contactos,
    `centros[${indice}].contactos`,
    validarOpcionReferencia,
  );
  exigirUnicos(contactos, "referencia", `centros[${indice}].contactos`);
  return { referencia: centro.referencia, etiqueta: centro.etiqueta, contactos };
}

function validarCategoria(categoria, indice) {
  exigirCamposExactos(
    categoria,
    ["referencia", "etiqueta", "grupos_subgrupos"],
    `categorias[${indice}]`,
  );
  if (!referenciaValida(categoria.referencia) || !etiquetaValida(categoria.etiqueta)) {
    throw new TypeError(`categorias[${indice}] no válida`);
  }
  const grupos = validarLista(
    categoria.grupos_subgrupos,
    `categorias[${indice}].grupos_subgrupos`,
    (opcion, nombre) => validarOpcionClave(opcion, nombre, PATRON_GRUPO),
  );
  exigirUnicos(grupos, "clave", `categorias[${indice}].grupos_subgrupos`);
  return {
    referencia: categoria.referencia,
    etiqueta: categoria.etiqueta,
    grupos_subgrupos: grupos,
  };
}

export function validarCatalogosAlta(catalogos) {
  exigirCamposExactos(
    catalogos,
    ["esquema", "centros", "categorias", "motivos", "documentos"],
    "catálogos de alta",
  );
  if (catalogos.esquema !== ESQUEMA_CATALOGOS) {
    throw new TypeError("esquema de catálogos de alta no compatible");
  }
  const centros = validarLista(catalogos.centros, "centros", validarCentro);
  const categorias = validarLista(catalogos.categorias, "categorias", validarCategoria);
  const motivos = validarLista(catalogos.motivos, "motivos", validarOpcionClave);
  const documentos = validarLista(
    catalogos.documentos,
    "documentos",
    validarOpcionReferencia,
  );
  exigirUnicos(centros, "referencia", "centros");
  exigirUnicos(categorias, "referencia", "categorías");
  exigirUnicos(motivos, "clave", "motivos");
  exigirUnicos(documentos, "referencia", "documentos");
  const total = centros.length + categorias.length + motivos.length + documentos.length
    + centros.reduce((suma, centro) => suma + centro.contactos.length, 0)
    + categorias.reduce((suma, categoria) => suma + categoria.grupos_subgrupos.length, 0);
  if (total > LIMITES_ALTA_CONTRATACION.opcionesCatalogoTotales) {
    throw new TypeError("los catálogos superan el límite técnico de opciones");
  }
  return clonarYCongelarAlta({
    esquema: ESQUEMA_CATALOGOS,
    centros,
    categorias,
    motivos,
    documentos,
  });
}

export function crearBorradorAlta() {
  return clonarYCongelarAlta({
    centro_ref: "",
    contacto_ref: "",
    categoria_ref: "",
    grupo_subgrupo: "",
    motivo_clave: "",
    detalle: "",
    inicio: "",
    fin: "",
    rc_existe: false,
    rc_numero: "",
    rc_fecha: "",
    rc_importe: "",
    rc_documento_ref: "",
    documentos_adjuntos: [],
    observaciones: "",
  });
}

function catalogosOperables(catalogos) {
  return catalogos.centros.some((centro) => centro.contactos.length > 0)
    && catalogos.categorias.some((categoria) => categoria.grupos_subgrupos.length > 0)
    && catalogos.motivos.length > 0;
}

export function catalogosAltaOperables(catalogos) {
  return catalogosOperables(validarCatalogosAlta(catalogos));
}

function centimosDesdeEntrada(valor) {
  if (typeof valor !== "string" || !/^(?:0|[1-9]\d{0,13})(?:[.,]\d{2})?$/.test(valor)) {
    return null;
  }
  const [entera, decimal = "00"] = valor.replace(",", ".").split(".");
  const centimos = Number(entera) * 100 + Number(decimal);
  return Number.isSafeInteger(centimos) ? centimos : null;
}

function referenciasAdjuntasValidas(valor) {
  return Array.isArray(valor)
    && valor.length <= LIMITES_ALTA_CONTRATACION.adjuntos
    && valor.every(referenciaValida)
    && new Set(valor).size === valor.length;
}

export function validarBorradorAlta(borrador, catalogosSinValidar) {
  const catalogos = validarCatalogosAlta(catalogosSinValidar);
  const errores = {};
  if (!tieneCamposExactos(borrador, CAMPOS_BORRADOR)) {
    return congelar({ valido: false, errores: { general: "contrato_cerrado" } });
  }

  const centro = catalogos.centros.find((opcion) => opcion.referencia === borrador.centro_ref);
  if (!centro) agregarError(errores, "centro_ref", "opcion_catalogo");
  if (!centro?.contactos.some((opcion) => opcion.referencia === borrador.contacto_ref)) {
    agregarError(errores, "contacto_ref", "opcion_catalogo");
  }
  const categoria = catalogos.categorias.find(
    (opcion) => opcion.referencia === borrador.categoria_ref,
  );
  if (!categoria) agregarError(errores, "categoria_ref", "opcion_catalogo");
  if (!categoria?.grupos_subgrupos.some((opcion) => opcion.clave === borrador.grupo_subgrupo)) {
    agregarError(errores, "grupo_subgrupo", "opcion_catalogo");
  }
  if (!catalogos.motivos.some((opcion) => opcion.clave === borrador.motivo_clave)) {
    agregarError(errores, "motivo_clave", "opcion_catalogo");
  }
  if (!textoValido(borrador.detalle, LIMITES_ALTA_CONTRATACION.texto, false)) {
    agregarError(errores, "detalle", "texto_obligatorio");
  }
  if (!textoValido(borrador.observaciones, LIMITES_ALTA_CONTRATACION.texto, true)) {
    agregarError(errores, "observaciones", "texto_opcional");
  }
  if (!fechaCivilValida(borrador.inicio)) agregarError(errores, "inicio", "fecha");
  if (!fechaCivilValida(borrador.fin)) agregarError(errores, "fin", "fecha");
  if (fechaCivilValida(borrador.inicio) && fechaCivilValida(borrador.fin)
    && borrador.fin < borrador.inicio) {
    agregarError(errores, "fin", "periodo");
  }
  if (typeof borrador.rc_existe !== "boolean") {
    agregarError(errores, "rc_existe", "booleano");
  } else if (borrador.rc_existe) {
    if (!referenciaValida(borrador.rc_numero)) agregarError(errores, "rc_numero", "referencia");
    if (!fechaCivilValida(borrador.rc_fecha)) agregarError(errores, "rc_fecha", "fecha");
    const centimos = centimosDesdeEntrada(borrador.rc_importe);
    if (centimos === null || centimos <= 0) agregarError(errores, "rc_importe", "importe");
    if (!catalogos.documentos.some(
      (opcion) => opcion.referencia === borrador.rc_documento_ref,
    )) {
      agregarError(errores, "rc_documento_ref", "opcion_catalogo");
    }
  } else if ([
    borrador.rc_numero,
    borrador.rc_fecha,
    borrador.rc_importe,
    borrador.rc_documento_ref,
  ].some((valor) => valor !== "")) {
    agregarError(errores, "rc_existe", "rc_residual");
  }
  if (!referenciasAdjuntasValidas(borrador.documentos_adjuntos)
    || borrador.documentos_adjuntos.some(
      (referencia) => !catalogos.documentos.some(
        (opcion) => opcion.referencia === referencia,
      ),
    )) {
    agregarError(errores, "documentos_adjuntos", "adjuntos");
  }

  return congelar({ valido: Object.keys(errores).length === 0, errores: { ...errores } });
}

function validarRC(rc) {
  if (!esRegistro(rc) || typeof rc.existe !== "boolean") return false;
  if (!rc.existe) return tieneCamposExactos(rc, ["existe"]);
  return tieneCamposExactos(rc, ["existe", "numero", "fecha", "importe", "documento_ref"])
    && referenciaValida(rc.numero)
    && instanteCivilValido(rc.fecha)
    && tieneCamposExactos(rc.importe, ["centimos", "moneda"])
    && Number.isSafeInteger(rc.importe.centimos)
    && rc.importe.centimos > 0
    && rc.importe.moneda === "EUR"
    && referenciaValida(rc.documento_ref);
}

function instanteCivilValido(valor) {
  return typeof valor === "string"
    && /^\d{4}-\d{2}-\d{2}T00:00:00Z$/.test(valor)
    && fechaCivilValida(valor.slice(0, 10));
}

export function validarComandoAlta(comando) {
  exigirCamposExactos(comando, ["clave_idempotencia", "solicitud"], "comando de alta");
  if (typeof comando.clave_idempotencia !== "string"
    || !PATRON_IDEMPOTENCIA.test(comando.clave_idempotencia)
    || comando.clave_idempotencia === CLAVE_IDEMPOTENCIA_NULA) {
    throw new TypeError("clave de operación no válida");
  }
  const solicitud = comando.solicitud;
  exigirCamposExactos(solicitud, CAMPOS_SOLICITUD, "solicitud de centro");
  exigirCamposExactos(solicitud.periodo, ["inicio", "fin"], "periodo");
  if (!referenciaValida(solicitud.centro_ref)
    || !referenciaValida(solicitud.contacto_ref)
    || !referenciaValida(solicitud.categoria_ref)
    || typeof solicitud.grupo_subgrupo !== "string"
    || !PATRON_GRUPO.test(solicitud.grupo_subgrupo)
    || typeof solicitud.motivo_clave !== "string"
    || !PATRON_CLAVE_CATALOGO.test(solicitud.motivo_clave)
    || !textoValido(solicitud.detalle, LIMITES_ALTA_CONTRATACION.texto, false)
    || !textoValido(solicitud.observaciones, LIMITES_ALTA_CONTRATACION.texto, true)
    || !instanteCivilValido(solicitud.periodo.inicio)
    || !instanteCivilValido(solicitud.periodo.fin)
    || solicitud.periodo.fin < solicitud.periodo.inicio
    || !validarRC(solicitud.rc)
    || !referenciasAdjuntasValidas(solicitud.documentos_adjuntos)) {
    throw new TypeError("solicitud de centro no válida");
  }
  return clonarYCongelarAlta(comando);
}

export function crearComandoAlta(borrador, catalogos, claveIdempotencia) {
  const validacion = validarBorradorAlta(borrador, catalogos);
  if (!validacion.valido) throw new ErrorValidacionAlta(validacion.errores);
  const rc = borrador.rc_existe
    ? {
      existe: true,
      numero: borrador.rc_numero,
      fecha: instanteCivilUTC(borrador.rc_fecha),
      importe: { centimos: centimosDesdeEntrada(borrador.rc_importe), moneda: "EUR" },
      documento_ref: borrador.rc_documento_ref,
    }
    : { existe: false };
  return validarComandoAlta({
    clave_idempotencia: claveIdempotencia,
    solicitud: {
      centro_ref: borrador.centro_ref,
      contacto_ref: borrador.contacto_ref,
      categoria_ref: borrador.categoria_ref,
      grupo_subgrupo: borrador.grupo_subgrupo,
      motivo_clave: borrador.motivo_clave,
      detalle: borrador.detalle,
      periodo: {
        inicio: instanteCivilUTC(borrador.inicio),
        fin: instanteCivilUTC(borrador.fin),
      },
      rc,
      documentos_adjuntos: [...borrador.documentos_adjuntos],
      observaciones: borrador.observaciones,
    },
  });
}

export function validarReciboAlta(recibo) {
  exigirCamposExactos(
    recibo,
    ["expediente_ref", "numero_visible", "version", "recibo_ref", "confirmada_en"],
    "recibo público de alta",
  );
  if (!referenciaValida(recibo.expediente_ref)
    || typeof recibo.numero_visible !== "string"
    || !PATRON_NUMERO.test(recibo.numero_visible)
    || !Number.isSafeInteger(recibo.version)
    || recibo.version < 1
    || !referenciaValida(recibo.recibo_ref)
    || !instanteUTCValido(recibo.confirmada_en)) {
    throw new TypeError("recibo público de alta no válido");
  }
  return clonarYCongelarAlta(recibo);
}
