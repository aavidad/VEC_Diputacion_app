/** Contrato neutral y cerrado del alta de contratación temporal. */
import { validarCatalogoPreparacion } from "./contrato-cobertura.js?v=20261002-ct-fin-modalidad-v1";

export const CAPACIDAD_CREAR_SOLICITUD = "contratacion_temporal.solicitud.crear";

export const LIMITES_ALTA_CONTRATACION = Object.freeze({
  texto: 4000,
  adjuntos: 64,
  referencia: 160,
  numeroExpediente: 45,
  patronNumero: 1024,
  etiquetaCatalogo: 200,
  opcionesCatalogo: 1000,
  opcionesCatalogoTotales: 5000,
  maximoAniosPeriodo: 100,
  maximoCentimosRC: 922_337_203_685_477,
});

const ESQUEMA_CATALOGOS = "vec.contratacion_temporal.catalogos_alta.v1";
export const ESQUEMA_CATALOGOS_NECESIDADES = "vec.contratacion_temporal.catalogos_alta.v2";
export const ESQUEMA_ALTA_NECESIDAD = "vec.ct.alta_necesidad.v1";
const CAMPOS_NECESIDAD = Object.freeze([
  "numero_personas", "puesto_codigo", "plaza_codigo", "titular_ref", "vacancia_fuente_ref",
  "rpt_catalogo_ref", "rpt_catalogo_huella_sha256",
  "organica_codigo", "funcional_codigo", "proyecto_gasto_codigo", "porcentaje_financiacion",
  "justificacion_temporal", "programa_denominacion", "programa_fin", "proyecto_codigo",
  "financiacion_ref", "rc_ref", "intervencion_ref",
]);
const PATRON_CODIGO_NECESIDAD = /^[A-Za-z0-9][A-Za-z0-9._/-]{0,79}$/u;
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
  if (valor === null || typeof valor !== "object" || Array.isArray(valor)) {
    return false;
  }
  try {
    if (Object.getPrototypeOf(valor) !== Object.prototype
      || Object.getOwnPropertySymbols(valor).length !== 0) {
      return false;
    }
    return Object.values(Object.getOwnPropertyDescriptors(valor)).every(
      (descriptor) => Object.hasOwn(descriptor, "value")
        && descriptor.enumerable === true,
    );
  } catch {
    return false;
  }
}

function esListaPlana(valor, maximo) {
  if (!Array.isArray(valor) || Object.getPrototypeOf(valor) !== Array.prototype
    || valor.length > maximo || Object.getOwnPropertySymbols(valor).length !== 0
    || Reflect.ownKeys(valor).length !== valor.length + 1) {
    return false;
  }
  for (let indice = 0; indice < valor.length; indice += 1) {
    const descriptor = Object.getOwnPropertyDescriptor(valor, String(indice));
    if (!descriptor || !Object.hasOwn(descriptor, "value")
      || descriptor.enumerable !== true) return false;
  }
  return true;
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
  if (typeof valor !== "string" || valor.startsWith("0000-")
    || !/^\d{4}-\d{2}-\d{2}$/.test(valor)) return false;
  const fecha = new Date(`${valor}T00:00:00Z`);
  return Number.isFinite(fecha.valueOf()) && fecha.toISOString().slice(0, 10) === valor;
}

function fechaFinDentroDeMeses(inicio, fin, meses) {
  if (!fechaCivilValida(inicio) || !fechaCivilValida(fin)
    || !Number.isSafeInteger(meses) || meses < 1) return false;
  const fecha = new Date(`${inicio}T00:00:00Z`);
  const dia = fecha.getUTCDate();
  fecha.setUTCDate(1);
  fecha.setUTCMonth(fecha.getUTCMonth() + meses);
  const siguienteMes = new Date(fecha);
  siguienteMes.setUTCMonth(siguienteMes.getUTCMonth() + 1);
  const ultimoDia = new Date(siguienteMes);
  ultimoDia.setUTCDate(0);
  const limite = dia > ultimoDia.getUTCDate() ? siguienteMes : fecha;
  if (dia <= ultimoDia.getUTCDate()) limite.setUTCDate(dia);
  return Date.parse(`${fin}T00:00:00Z`) < limite.getTime();
}

// El formulario acepta horas decimales o h:mm, pero el contrato conserva
// minutos enteros. Una fracción inferior a un minuto se rechaza sin redondear.
export function minutosDesdeJornadaVisible(valor) {
  if (typeof valor !== "string" || valor !== valor.trim()) return null;
  const reloj = /^(\d{1,3}):([0-5]\d)$/u.exec(valor);
  const decimal = /^(\d{1,3})(?:[.,](\d{1,3}))?$/u.exec(valor);
  if (!reloj && !decimal) return null;
  let minutos;
  if (reloj) minutos = Number(reloj[1]) * 60 + Number(reloj[2]);
  else {
    const divisor = 10 ** (decimal[2]?.length ?? 0);
    const fraccion = Number(decimal[2] ?? 0) * 60;
    if (fraccion % divisor !== 0) return null;
    minutos = Number(decimal[1]) * 60 + fraccion / divisor;
  }
  return Number.isSafeInteger(minutos) && minutos >= 1 && minutos <= 10080 ? minutos : null;
}

export function jornadaVisibleDesdeMinutos(valor) {
  const minutos = Number(valor);
  if (!/^(?:0|[1-9]\d*)$/u.test(String(valor)) || !Number.isSafeInteger(minutos)
    || minutos < 1 || minutos > 10080) return "";
  return `${Math.floor(minutos / 60)}:${String(minutos % 60).padStart(2, "0")}`;
}

function instanteCivilUTC(fecha) {
  return `${fecha}T00:00:00Z`;
}

function periodoNoSuperaMaximo(inicio, fin) {
  const fechaMaxima = new Date(`${inicio}T00:00:00Z`);
  const anioMaximo = fechaMaxima.getUTCFullYear()
    + LIMITES_ALTA_CONTRATACION.maximoAniosPeriodo;
  fechaMaxima.setUTCFullYear(anioMaximo);
  return new Date(`${fin}T00:00:00Z`).valueOf() <= fechaMaxima.valueOf();
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

function validarMotivoConFin(opcion, nombre) {
  const tieneRegla = esRegistro(opcion) && Object.hasOwn(opcion, "fecha_fin");
  const tieneCausa = esRegistro(opcion) && Object.hasOwn(opcion, "causa_fin");
  const tieneReferencia = esRegistro(opcion) && Object.hasOwn(opcion, "regla_ref");
  const tieneVersion = esRegistro(opcion) && Object.hasOwn(opcion, "catalogo_version");
  const tieneHuella = esRegistro(opcion) && Object.hasOwn(opcion, "catalogo_huella_sha256");
  exigirCamposExactos(opcion, ["clave", "etiqueta", ...(tieneRegla ? ["fecha_fin"] : []),
    ...(tieneCausa ? ["causa_fin"] : []), ...(tieneReferencia ? ["regla_ref"] : []),
    ...(tieneVersion ? ["catalogo_version"] : []),
    ...(tieneHuella ? ["catalogo_huella_sha256"] : [])], nombre);
  if (typeof opcion.clave !== "string" || !PATRON_CLAVE_CATALOGO.test(opcion.clave)
    || !etiquetaValida(opcion.etiqueta)
    || (tieneRegla && !["obligatoria", "opcional", "no_aplica"].includes(opcion.fecha_fin))
    || (tieneCausa && (typeof opcion.causa_fin !== "string"
      || !PATRON_CLAVE_CATALOGO.test(opcion.causa_fin)))
    || (opcion.fecha_fin === "no_aplica" && !tieneCausa)
    || (tieneReferencia !== tieneVersion || tieneVersion !== tieneHuella)
    || (tieneReferencia && (!tieneRegla || !referenciaValida(opcion.regla_ref)
      || !Number.isSafeInteger(opcion.catalogo_version) || opcion.catalogo_version < 1
      || typeof opcion.catalogo_huella_sha256 !== "string"
      || !/^[0-9a-f]{64}$/u.test(opcion.catalogo_huella_sha256)
      || /^0{64}$/u.test(opcion.catalogo_huella_sha256)))) {
    throw new TypeError(`${nombre} no válido`);
  }
  return { clave: opcion.clave, etiqueta: opcion.etiqueta,
    fecha_fin: opcion.fecha_fin ?? "obligatoria",
    ...(tieneCausa ? { causa_fin: opcion.causa_fin } : {}),
    ...(tieneReferencia ? { regla_ref: opcion.regla_ref,
      catalogo_version: opcion.catalogo_version,
      catalogo_huella_sha256: opcion.catalogo_huella_sha256 } : {}) };
}

function exigirUnicos(opciones, campo, nombre) {
  const valores = opciones.map((opcion) => opcion[campo]);
  if (new Set(valores).size !== valores.length) throw new TypeError(`${nombre} contiene duplicados`);
}

function validarLista(lista, nombre, validarOpcion) {
  if (!esListaPlana(
    lista,
    LIMITES_ALTA_CONTRATACION.opcionesCatalogo,
  )) {
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

export function numeroExpedienteMOADValido(valor) {
  // Misma envolvente técnica que el servidor; la política publicada se
  // comprueba allí sin ejecutar su patrón en el navegador.
  return typeof valor === "string" && PATRON_NUMERO.test(valor);
}

export function validarPoliticaNumeroMOAD(politica) {
  exigirCamposExactos(politica, ["referencia", "version", "patron", "ejemplo"], "política de número MOAD");
  if (!referenciaValida(politica.referencia) || !Number.isSafeInteger(politica.version)
    || politica.version < 1 || !textoValido(politica.patron, LIMITES_ALTA_CONTRATACION.patronNumero, false)
    || !numeroExpedienteMOADValido(politica.ejemplo)) {
    throw new TypeError("política de número MOAD no válida");
  }
  return clonarYCongelarAlta(politica);
}

export function validarCatalogosAlta(catalogos) {
  // La relación de documentos y datos por vía de cobertura es opcional: la
  // publica la ruta de catálogos del alta de RRHH, no el contexto del centro.
  const conNumero = esRegistro(catalogos) && Object.hasOwn(catalogos, "numero_expediente_moad");
  const conPreparacion = esRegistro(catalogos) && Object.hasOwn(catalogos, "preparacion_vias");
  const conNecesidades = catalogos?.esquema === ESQUEMA_CATALOGOS_NECESIDADES;
  if (conNecesidades) return validarCatalogosNecesidades(catalogos);
  exigirCamposExactos(
    catalogos,
    ["esquema", "centros", "categorias", "motivos", "documentos",
      ...(conPreparacion ? ["preparacion_vias"] : []),
      ...(conNumero ? ["numero_expediente_moad"] : [])],
    "catálogos de alta",
  );
  if (catalogos.esquema !== ESQUEMA_CATALOGOS) {
    throw new TypeError("esquema de catálogos de alta no compatible");
  }
  const centros = validarLista(catalogos.centros, "centros", validarCentro);
  const categorias = validarLista(catalogos.categorias, "categorias", validarCategoria);
  const motivos = validarLista(catalogos.motivos, "motivos", validarMotivoConFin);
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
    ...(conNumero ? { numero_expediente_moad: validarPoliticaNumeroMOAD(catalogos.numero_expediente_moad) } : {}),
    centros,
    categorias,
    motivos,
    documentos,
    ...(conPreparacion
      ? { preparacion_vias: validarCatalogoPreparacion(catalogos.preparacion_vias) } : {}),
  });
}

function validarCatalogosNecesidades(catalogos) {
  const conNumero = Object.hasOwn(catalogos, "numero_expediente_moad");
  const conPreparacion = Object.hasOwn(catalogos, "preparacion_vias");
  // "motivos" solo aparece en la copia ya validada, nunca en la respuesta v2.
  const conMotivos = Object.hasOwn(catalogos, "motivos");
  exigirCamposExactos(catalogos, ["esquema", "centros", "categorias", "documentos", "necesidades",
    ...(conNumero ? ["numero_expediente_moad"] : []),
    ...(conPreparacion ? ["preparacion_vias"] : []), ...(conMotivos ? ["motivos"] : [])], "catálogos de necesidades");
  const n = catalogos.necesidades;
  exigirCamposExactos(n, ["referencia", "version", "huella_sha256", "es_ejemplo", "fuente_ref",
    "fuente_url", "jornada_referencia_minutos", "jornada_fuente_ref", "causas"], "necesidades");
  if (!referenciaValida(n.referencia) || !Number.isSafeInteger(n.version) || n.version < 1
    || !/^[a-f0-9]{64}$/u.test(n.huella_sha256) || typeof n.es_ejemplo !== "boolean"
    || !referenciaValida(n.fuente_ref) || typeof n.fuente_url !== "string"
    || !/^https:\/\//u.test(n.fuente_url) || !Number.isSafeInteger(n.jornada_referencia_minutos)
    || n.jornada_referencia_minutos < 1 || n.jornada_referencia_minutos > 10080
    || !referenciaValida(n.jornada_fuente_ref) || !esListaPlana(n.causas, 16)
    || n.causas.length !== 4) throw new TypeError("catálogo de necesidades incompatible");
  const causas = n.causas.map((causa) => {
    const conFin = Object.hasOwn(causa, "causa_fin");
    const conVentana = Object.hasOwn(causa, "ventana_meses");
    exigirCamposExactos(causa, ["clave", "etiqueta_clave", "fuente_ref", "fuente_url", "regla_ref",
      "fecha_fin", "maximo_meses", "campos_permitidos", "campos_obligatorios",
      ...(conFin ? ["causa_fin"] : []), ...(conVentana ? ["ventana_meses"] : []),
      ...(Object.hasOwn(causa, "uno_de") ? ["uno_de"] : [])], "causa de necesidad");
    if (!PATRON_CLAVE_CATALOGO.test(causa.clave) || !etiquetaValida(causa.etiqueta_clave)
      || !referenciaValida(causa.fuente_ref) || typeof causa.fuente_url !== "string"
      || !/^https:\/\//u.test(causa.fuente_url) || !referenciaValida(causa.regla_ref)
      || !["obligatoria", "opcional", "no_aplica"].includes(causa.fecha_fin)
      || (conFin && !PATRON_CLAVE_CATALOGO.test(causa.causa_fin))
      || !Number.isSafeInteger(causa.maximo_meses) || causa.maximo_meses < 1
      || (conVentana && (!Number.isSafeInteger(causa.ventana_meses) || causa.ventana_meses < 1))) {
      throw new TypeError("causa de necesidad incompatible");
    }
    const permitidos = causa.campos_permitidos;
    const obligatorios = causa.campos_obligatorios;
    if (!esListaPlana(permitidos, 32) || !esListaPlana(obligatorios, 32)
      || new Set(permitidos).size !== permitidos.length
      || permitidos.some((campo) => !CAMPOS_NECESIDAD.includes(campo))
      || obligatorios.some((campo) => !permitidos.includes(campo))
      || (causa.uno_de && (!esListaPlana(causa.uno_de, 8)
        || causa.uno_de.some((grupo) => !esListaPlana(grupo, 8)
          || grupo.length < 2 || grupo.some((campo) => !permitidos.includes(campo)))))) {
      throw new TypeError("campos de necesidad incompatibles");
    }
    return causa;
  });
  if (new Set(causas.map((causa) => causa.clave)).size !== 4
    || !["vacante", "sustitucion", "acumulacion_tareas", "programa_temporal"]
      .every((clave) => causas.some((causa) => causa.clave === clave))) {
    throw new TypeError("causas de necesidad incompatibles");
  }
  const motivos = causas.map((causa) => ({ clave: causa.clave, etiqueta: causa.etiqueta_clave,
    fecha_fin: causa.fecha_fin, ...(causa.causa_fin ? { causa_fin: causa.causa_fin } : {}) }));
  if (conMotivos && JSON.stringify(catalogos.motivos) !== JSON.stringify(motivos)) {
    throw new TypeError("motivos de necesidad alterados");
  }
  const base = validarCatalogosAlta({ esquema: ESQUEMA_CATALOGOS, centros: catalogos.centros,
    categorias: catalogos.categorias, documentos: catalogos.documentos, motivos,
    ...(conNumero ? { numero_expediente_moad: catalogos.numero_expediente_moad } : {}),
    ...(conPreparacion ? { preparacion_vias: catalogos.preparacion_vias } : {}) });
  return clonarYCongelarAlta({ ...base, esquema: ESQUEMA_CATALOGOS_NECESIDADES,
    necesidades: n });
}

function valorNecesidadValido(campo, valor) {
  if (typeof valor !== "string" || valor === "" || valor !== valor.trim()) return false;
  if (campo === "numero_personas") {
    return /^[1-9][0-9]{0,9}$/u.test(valor) && Number(valor) <= 4_294_967_295;
  }
  if (["justificacion_temporal", "programa_denominacion"].includes(campo)) {
    return textoValido(valor, 4000, false);
  }
  if (["titular_ref", "financiacion_ref", "rc_ref", "intervencion_ref",
    "vacancia_fuente_ref", "rpt_catalogo_ref"].includes(campo)) return referenciaValida(valor);
  if (campo === "rpt_catalogo_huella_sha256") return /^[a-f0-9]{64}$/u.test(valor);
  if (campo === "porcentaje_financiacion") return /^(?:[1-9]|[1-9][0-9]|100)$/u.test(valor);
  if (campo === "programa_fin") return fechaCivilValida(valor);
  return PATRON_CODIGO_NECESIDAD.test(valor);
}

function errorValorNecesidad(campo) {
  if (["numero_personas", "porcentaje_financiacion", "justificacion_temporal"].includes(campo)) return campo;
  if (["organica_codigo", "funcional_codigo", "proyecto_gasto_codigo", "proyecto_codigo"].includes(campo)) {
    return "codigo_necesidad";
  }
  return "campo_necesidad";
}

export function crearBorradorAlta({ conNumeroMOAD = false, conNecesidad = false,
  jornadaReferenciaMinutos = 0 } = {}) {
  return clonarYCongelarAlta({
    ...(conNumeroMOAD ? { numero_expediente_moad: "" } : {}),
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
    ...(conNecesidad ? { jornada_minutos: String(jornadaReferenciaMinutos),
      ...Object.fromEntries(CAMPOS_NECESIDAD.map((campo) => [campo, ""])) } : {}),
  });
}

function catalogosOperables(catalogos) {
  return catalogos.centros.some((centro) => centro.contactos.length > 0)
    && catalogos.categorias.some((categoria) => categoria.grupos_subgrupos.length > 0)
    && catalogos.motivos.length > 0;
}

export function catalogosAltaOperables(catalogos) {
  const validados = validarCatalogosAlta(catalogos);
  return Boolean(validados.numero_expediente_moad) && catalogosOperables(validados);
}

function analizarEntradaMonetaria(valor) {
  if (typeof valor !== "string" || !/^(?:0|[1-9]\d{0,13})(?:[.,]\d{2})?$/.test(valor)) {
    return null;
  }
  const [entera, decimal = "00"] = valor.replace(",", ".").split(".");
  const digitos = `${entera}${decimal}`.replace(/^0+(?=\d)/, "");
  const maximo = String(LIMITES_ALTA_CONTRATACION.maximoCentimosRC);
  const excedeMaximo = digitos.length > maximo.length
    || (digitos.length === maximo.length && digitos > maximo);
  return { centimos: excedeMaximo ? null : Number(digitos), excedeMaximo };
}

function centimosDesdeEntrada(valor) {
  const entrada = analizarEntradaMonetaria(valor);
  return entrada === null || entrada.excedeMaximo ? null : entrada.centimos;
}

function referenciasAdjuntasValidas(valor) {
  return esListaPlana(valor, LIMITES_ALTA_CONTRATACION.adjuntos)
    && valor.every(referenciaValida)
    && new Set(valor).size === valor.length;
}

export function validarBorradorAlta(borrador, catalogosSinValidar) {
  const catalogos = validarCatalogosAlta(catalogosSinValidar);
  const errores = {};
  const conNumero = esRegistro(borrador) && Object.hasOwn(borrador, "numero_expediente_moad");
  const conNecesidad = catalogos.esquema === ESQUEMA_CATALOGOS_NECESIDADES;
  if (!tieneCamposExactos(borrador, [...CAMPOS_BORRADOR, ...(conNumero ? ["numero_expediente_moad"] : []),
    ...(conNecesidad ? ["jornada_minutos", ...CAMPOS_NECESIDAD] : [])])) {
    return congelar({ valido: false, errores: { general: "contrato_cerrado" } });
  }

  if (conNumero && !numeroExpedienteMOADValido(borrador.numero_expediente_moad)) {
    agregarError(errores, "numero_expediente_moad", "numero_moad_formato");
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
  const motivo = catalogos.motivos.find((opcion) => opcion.clave === borrador.motivo_clave);
  if (!motivo) {
    agregarError(errores, "motivo_clave", "opcion_catalogo");
  }
  if (!textoValido(borrador.detalle, LIMITES_ALTA_CONTRATACION.texto, false)) {
    agregarError(errores, "detalle", "texto_obligatorio");
  }
  if (!textoValido(borrador.observaciones, LIMITES_ALTA_CONTRATACION.texto, true)) {
    agregarError(errores, "observaciones", "texto_opcional");
  }
  if (!fechaCivilValida(borrador.inicio)) agregarError(errores, "inicio", "fecha");
  if (motivo?.fecha_fin === "no_aplica" && borrador.fin !== "") agregarError(errores, "fin", "fecha_no_aplica");
  else if (borrador.fin === "" && motivo?.fecha_fin === "obligatoria") agregarError(errores, "fin", "fecha");
  else if (borrador.fin !== "" && !fechaCivilValida(borrador.fin)) agregarError(errores, "fin", "fecha");
  if (borrador.fin === "" && motivo && motivo.fecha_fin !== "obligatoria"
    && !motivo.causa_fin) agregarError(errores, "fin", "causa_fin");
  if (fechaCivilValida(borrador.inicio) && fechaCivilValida(borrador.fin)
    && borrador.fin < borrador.inicio) {
    agregarError(errores, "fin", "periodo");
  } else if (fechaCivilValida(borrador.inicio) && fechaCivilValida(borrador.fin)
    && !periodoNoSuperaMaximo(borrador.inicio, borrador.fin)) {
    agregarError(errores, "fin", "periodo_maximo");
  }
  if (typeof borrador.rc_existe !== "boolean") {
    agregarError(errores, "rc_existe", "booleano");
  } else if (borrador.rc_existe) {
    if (!referenciaValida(borrador.rc_numero)) agregarError(errores, "rc_numero", "referencia");
    if (!fechaCivilValida(borrador.rc_fecha)) agregarError(errores, "rc_fecha", "fecha");
    const importe = analizarEntradaMonetaria(borrador.rc_importe);
    if (importe === null || (!importe.excedeMaximo && importe.centimos <= 0)) {
      agregarError(errores, "rc_importe", "importe");
    } else if (importe.excedeMaximo) {
      agregarError(errores, "rc_importe", "importe_maximo");
    }
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
  if (conNecesidad) {
    if (!/^(?:[1-9][0-9]{0,4})$/u.test(borrador.jornada_minutos)
      || Number(borrador.jornada_minutos) > 10080) agregarError(errores, "jornada_minutos", "jornada");
    const causa = catalogos.necesidades.causas.find((dato) => dato.clave === borrador.motivo_clave);
    if (causa) {
      if (fechaCivilValida(borrador.inicio) && fechaCivilValida(borrador.fin)
        && !fechaFinDentroDeMeses(borrador.inicio, borrador.fin, causa.maximo_meses)) {
        agregarError(errores, "fin", "periodo_maximo_necesidad");
      }
      if (borrador.programa_fin && fechaCivilValida(borrador.programa_fin)
        && ((fechaCivilValida(borrador.inicio) && borrador.programa_fin < borrador.inicio)
          || (fechaCivilValida(borrador.fin) && borrador.programa_fin < borrador.fin))) {
        agregarError(errores, "programa_fin", "campo_necesidad");
      }
      for (const campo of CAMPOS_NECESIDAD) {
        const valor = borrador[campo];
        const codigoError = errorValorNecesidad(campo);
        if (typeof valor !== "string" || (valor && !valorNecesidadValido(campo, valor))) {
          agregarError(errores, campo, codigoError);
        }
        if (valor && !causa.campos_permitidos.includes(campo)) agregarError(errores, campo, "campo_necesidad");
        if (causa.campos_obligatorios.includes(campo) && !valor) {
          agregarError(errores, campo, codigoError === "campo_necesidad" ? "texto_obligatorio" : codigoError);
        }
      }
      for (const grupo of causa.uno_de ?? []) {
        if (grupo.filter((campo) => borrador[campo]).length !== 1) agregarError(errores, grupo[0], "uno_de");
      }
    }
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
    && rc.importe.centimos <= LIMITES_ALTA_CONTRATACION.maximoCentimosRC
    && rc.importe.moneda === "EUR"
    && referenciaValida(rc.documento_ref);
}

function instanteCivilValido(valor) {
  return typeof valor === "string"
    && /^\d{4}-\d{2}-\d{2}T00:00:00Z$/.test(valor)
    && fechaCivilValida(valor.slice(0, 10));
}

function validarComandoPeticionCentro(comando) {
  exigirCamposExactos(comando, ["clave_idempotencia", "solicitud"], "comando de alta");
  if (typeof comando.clave_idempotencia !== "string"
    || !PATRON_IDEMPOTENCIA.test(comando.clave_idempotencia)
    || comando.clave_idempotencia === CLAVE_IDEMPOTENCIA_NULA) {
    throw new TypeError("clave de operación no válida");
  }
  const solicitud = comando.solicitud;
  exigirCamposExactos(solicitud, CAMPOS_SOLICITUD, "solicitud de centro");
  const tieneFin = esRegistro(solicitud.periodo) && Object.hasOwn(solicitud.periodo, "fin");
  const tieneCausa = esRegistro(solicitud.periodo) && Object.hasOwn(solicitud.periodo, "causa_fin");
  exigirCamposExactos(solicitud.periodo, ["inicio", ...(tieneFin ? ["fin"] : []),
    ...(tieneCausa ? ["causa_fin"] : [])], "periodo");
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
    || (tieneFin ? !instanteCivilValido(solicitud.periodo.fin)
      || tieneCausa || solicitud.periodo.fin < solicitud.periodo.inicio
      || !periodoNoSuperaMaximo(
      solicitud.periodo.inicio.slice(0, 10),
      solicitud.periodo.fin.slice(0, 10),
      ) : !tieneCausa || typeof solicitud.periodo.causa_fin !== "string"
        || !PATRON_CLAVE_CATALOGO.test(solicitud.periodo.causa_fin))
    || !validarRC(solicitud.rc)
    || !referenciasAdjuntasValidas(solicitud.documentos_adjuntos)) {
    throw new TypeError("solicitud de centro no válida");
  }
  return clonarYCongelarAlta(comando);
}

export function crearComandoPeticionCentro(borrador, catalogos, claveIdempotencia) {
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
  return validarComandoPeticionCentro({
    clave_idempotencia: claveIdempotencia,
    solicitud: {
      centro_ref: borrador.centro_ref,
      contacto_ref: borrador.contacto_ref,
      categoria_ref: borrador.categoria_ref,
      grupo_subgrupo: borrador.grupo_subgrupo,
      motivo_clave: borrador.motivo_clave,
      detalle: borrador.detalle,
      periodo: { inicio: instanteCivilUTC(borrador.inicio),
        ...(borrador.fin ? { fin: instanteCivilUTC(borrador.fin) }
          : { causa_fin: validarCatalogosAlta(catalogos).motivos.find(
            ({ clave }) => clave === borrador.motivo_clave).causa_fin }) },
      rc,
      documentos_adjuntos: [...borrador.documentos_adjuntos],
      observaciones: borrador.observaciones,
    },
  });
}

export function validarComandoAlta(comando) {
  exigirCamposExactos(comando, ["clave_idempotencia", "numero_expediente_moad", "solicitud"], "comando de alta");
  if (!numeroExpedienteMOADValido(comando.numero_expediente_moad)) {
    throw new TypeError("número de expediente MOAD no válido");
  }
  validarComandoPeticionCentro({ clave_idempotencia: comando.clave_idempotencia, solicitud: comando.solicitud });
  return clonarYCongelarAlta(comando);
}

export function crearComandoAlta(borrador, catalogos, claveIdempotencia) {
  const validacion = validarBorradorAlta(borrador, catalogos);
  if (!validacion.valido) throw new ErrorValidacionAlta(validacion.errores);
  if (!numeroExpedienteMOADValido(borrador.numero_expediente_moad)) {
    throw new ErrorValidacionAlta({ numero_expediente_moad: "numero_moad" });
  }
  const catalogosValidados = validarCatalogosAlta(catalogos);
  if (!catalogosValidados.numero_expediente_moad) {
    throw new ErrorValidacionAlta({ general: "contrato_cerrado" });
  }
  const { numero_expediente_moad, jornada_minutos, ...resto } = borrador;
  const solicitud = { ...resto };
  for (const campo of CAMPOS_NECESIDAD) delete solicitud[campo];
  const catalogoPeticion = catalogosValidados.esquema === ESQUEMA_CATALOGOS_NECESIDADES
    ? { esquema: ESQUEMA_CATALOGOS, centros: catalogosValidados.centros, categorias: catalogosValidados.categorias,
      motivos: catalogosValidados.motivos, documentos: catalogosValidados.documentos,
      numero_expediente_moad: catalogosValidados.numero_expediente_moad,
      ...(catalogosValidados.preparacion_vias ? { preparacion_vias: catalogosValidados.preparacion_vias } : {}) }
    : catalogosValidados;
  const base = crearComandoPeticionCentro(solicitud, catalogoPeticion, claveIdempotencia);
  if (catalogosValidados.esquema === ESQUEMA_CATALOGOS_NECESIDADES) {
    const n = catalogosValidados.necesidades;
    const causa = n.causas.find((dato) => dato.clave === borrador.motivo_clave);
    return validarComandoAltaNecesidad({ ...base, numero_expediente_moad,
      esquema: ESQUEMA_ALTA_NECESIDAD,
      necesidad: { esquema: "vec.ct.necesidad_alta.v1", catalogo_ref: n.referencia,
        catalogo_version: n.version, catalogo_huella_sha256: n.huella_sha256,
        causa_clave: causa.clave, jornada_minutos: Number(jornada_minutos),
        campos: Object.fromEntries(causa.campos_permitidos.filter((campo) => borrador[campo])
          .map((campo) => [campo, borrador[campo]])) } });
  }
  return validarComandoAlta({
    ...base,
    numero_expediente_moad,
  });
}

export function validarComandoAltaNecesidad(comando) {
  exigirCamposExactos(comando, ["clave_idempotencia", "numero_expediente_moad", "solicitud",
    "esquema", "necesidad"], "alta con necesidad");
  if (comando.esquema !== ESQUEMA_ALTA_NECESIDAD) throw new TypeError("esquema de alta incompatible");
  validarComandoAlta({ clave_idempotencia: comando.clave_idempotencia,
    numero_expediente_moad: comando.numero_expediente_moad, solicitud: comando.solicitud });
  const n = comando.necesidad;
  exigirCamposExactos(n, ["esquema", "catalogo_ref", "catalogo_version", "catalogo_huella_sha256",
    "causa_clave", "jornada_minutos", "campos"], "necesidad");
  if (n.esquema !== "vec.ct.necesidad_alta.v1" || !referenciaValida(n.catalogo_ref)
    || !Number.isSafeInteger(n.catalogo_version) || n.catalogo_version < 1
    || !/^[a-f0-9]{64}$/u.test(n.catalogo_huella_sha256)
    || !PATRON_CLAVE_CATALOGO.test(n.causa_clave) || n.causa_clave !== comando.solicitud.motivo_clave
    || !Number.isSafeInteger(n.jornada_minutos) || n.jornada_minutos < 1
    || n.jornada_minutos > 10080 || !esRegistro(n.campos)
    || Object.keys(n.campos).length > 32 || Object.entries(n.campos).some(([campo, valor]) =>
      !CAMPOS_NECESIDAD.includes(campo) || !valorNecesidadValido(campo, valor))) {
    throw new TypeError("necesidad de alta no válida");
  }
  return clonarYCongelarAlta(comando);
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
