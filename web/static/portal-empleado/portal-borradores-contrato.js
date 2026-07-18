/**
 * Contrato cerrado de la superficie interna de borradores de convocatorias.
 *
 * Este módulo no completa datos ausentes, no adapta contratos anteriores y no
 * acepta campos de gobierno declarados por el navegador. Actor, autorización,
 * ámbito efectivo, estado, firma, custodia y configuración se resuelven o se
 * acreditan exclusivamente en el servidor.
 */

export const ESQUEMAS_BORRADORES = Object.freeze({
  opciones: "vec.bolsa.borradores.opciones.v1",
  lista: "vec.bolsa.borradores.lista.v1",
  detalle: "vec.bolsa.borrador.detalle.v1",
  crear: "vec.bolsa.borrador.crear.v1",
  actualizar: "vec.bolsa.borrador.actualizar.v1",
  guardado: "vec.bolsa.borrador.guardado.v1",
});

const CAMPOS_LIMITES = Object.freeze([
  "maximo_categorias", "maximo_plazos", "maximo_requisitos", "maximo_documentos", "maximo_ayudas",
  "maximo_titulo", "maximo_resumen", "maximo_descripcion",
  "maximo_titulo_plazo", "maximo_descripcion_plazo",
  "maximo_titulo_requisito", "maximo_descripcion_requisito",
  "maximo_pregunta_ayuda", "maximo_respuesta_ayuda",
]);

const LIMITES_ABSOLUTOS_DOMINIO = Object.freeze({
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
});

const CAMPOS_CONTENIDO = Object.freeze([
  "tipo", "categorias", "titulo", "resumen", "descripcion",
  "plazos", "requisitos", "ayuda",
]);

function esObjeto(valor) {
  return valor !== null && typeof valor === "object" && !Array.isArray(valor);
}

function exigirCamposExactos(objeto, campos, nombre, opcionales = []) {
  if (!esObjeto(objeto)) throw new Error(`${nombre} no válido`);
  const permitidos = new Set(campos);
  const opcionalesSet = new Set(opcionales);
  if (Object.keys(objeto).some((campo) => !permitidos.has(campo))
    || campos.some((campo) => !opcionalesSet.has(campo) && !Object.hasOwn(objeto, campo))) {
    throw new Error(`${nombre} no respeta el contrato cerrado`);
  }
}

function exigirCadena(valor, nombre, maximo = 20_000, admiteVacia = false, multilinea = false) {
  if (typeof valor !== "string" || valor !== valor.trim() || valor.normalize("NFC") !== valor
    || (!admiteVacia && valor === "") || [...valor].length > maximo) {
    throw new Error(`${nombre} no válido`);
  }
  for (const caracter of valor) {
    const codigo = caracter.codePointAt(0);
    const controlPermitido = multilinea && (codigo === 9 || codigo === 10);
    if ((!controlPermitido && (codigo < 32 || (codigo >= 127 && codigo <= 159)))
      || (codigo >= 0xD800 && codigo <= 0xDFFF) || /\p{Cf}/u.test(caracter)) {
      throw new Error(`${nombre} no válido`);
    }
  }
  return valor;
}

function exigirClave(valor, nombre) {
  const clave = exigirCadena(valor, nombre, 80);
  if (!/^[a-z0-9][a-z0-9._-]{0,79}$/.test(clave)) throw new Error(`${nombre} no válida`);
  return clave;
}

function exigirReferencia(valor, nombre, maximo = 512) {
  const referencia = exigirCadena(valor, nombre, maximo);
  if (!/^[A-Za-z0-9][A-Za-z0-9._:/#@-]{0,511}$/.test(referencia)
    || referencia.length > maximo) {
    throw new Error(`${nombre} no válida`);
  }
  return referencia;
}

export function descomponerReferenciaEstadoBorrador(referencia) {
  const valor = exigirCadena(referencia, "referencia de estado", 180);
  const coincidencia = /^([A-Za-z0-9][A-Za-z0-9._:-]{0,159})#([1-9][0-9]{0,15})$/.exec(valor);
  if (!coincidencia) throw new Error("referencia de estado no canónica");
  const secuencia = Number(coincidencia[2]);
  if (!Number.isSafeInteger(secuencia)) throw new Error("secuencia de estado no válida");
  return Object.freeze({ id: coincidencia[1], secuencia });
}

function exigirIdentificadorPublico(valor) {
  const identificador = exigirCadena(valor, "código público", 80);
  if (!/^[a-z0-9][a-z0-9-]{2,79}$/.test(identificador)) {
    throw new Error("código público no válido");
  }
  return identificador;
}

function exigirCursor(valor) {
  const cursor = exigirCadena(valor, "cursor", 512);
  if (!/^[A-Za-z0-9][A-Za-z0-9._:@-]{0,511}$/.test(cursor)) {
    throw new Error("cursor no canónico");
  }
  return cursor;
}

function exigirHuella(valor, nombre = "huella") {
  if (typeof valor !== "string" || !/^[a-f0-9]{64}$/.test(valor)) {
    throw new Error(`${nombre} no válida`);
  }
  return valor;
}

function exigirEntero(valor, nombre, minimo = 0, maximo = Number.MAX_SAFE_INTEGER) {
  if (!Number.isSafeInteger(valor) || valor < minimo || valor > maximo) {
    throw new Error(`${nombre} no válido`);
  }
  return valor;
}

function exigirBooleano(valor, nombre) {
  if (typeof valor !== "boolean") throw new Error(`${nombre} no válido`);
  return valor;
}

function exigirInstante(valor, nombre) {
  if (typeof valor !== "string") throw new Error(`${nombre} no válido`);
  const coincidencia = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.(\d{1,6}))?Z$/.exec(valor);
  if (!coincidencia) {
    throw new Error(`${nombre} no válido`);
  }
  const [, anoTexto, mesTexto, diaTexto, horaTexto, minutoTexto, segundoTexto, fraccion = ""] = coincidencia;
  const ano = Number(anoTexto);
  const mes = Number(mesTexto);
  const dia = Number(diaTexto);
  const hora = Number(horaTexto);
  const minuto = Number(minutoTexto);
  const segundo = Number(segundoTexto);
  if (ano < 1 || mes < 1 || mes > 12 || dia < 1 || dia > 31
    || hora > 23 || minuto > 59 || segundo > 59) {
    throw new Error(`${nombre} no válido`);
  }
  const fecha = new Date(0);
  fecha.setUTCHours(0, 0, 0, 0);
  fecha.setUTCFullYear(ano, mes - 1, dia);
  fecha.setUTCHours(hora, minuto, segundo, Number(fraccion.padEnd(3, "0").slice(0, 3)));
  if (fecha.getUTCFullYear() !== ano || fecha.getUTCMonth() !== mes - 1
    || fecha.getUTCDate() !== dia || fecha.getUTCHours() !== hora
    || fecha.getUTCMinutes() !== minuto || fecha.getUTCSeconds() !== segundo) {
    throw new Error(`${nombre} no válido`);
  }
  return valor;
}

function instanteOrdenable(valor) {
  const punto = valor.indexOf(".");
  if (punto === -1) return `${valor.slice(0, -1)}.000000Z`;
  const base = valor.slice(0, punto);
  const fraccion = valor.slice(punto + 1, -1).padEnd(6, "0");
  return `${base}.${fraccion}Z`;
}

function compararInstantes(primero, segundo) {
  const a = instanteOrdenable(primero);
  const b = instanteOrdenable(segundo);
  if (a === b) return 0;
  return a < b ? -1 : 1;
}

/**
 * Representación fuerte y determinista del estado, sin operación criptográfica
 * en el navegador: la huella ya es el SHA-256 acreditado por el servidor.
 */
export function derivarETagBorrador(referenciaEstado) {
  if (!esObjeto(referenciaEstado)) throw new Error("referencia de estado no válida para ETag");
  const revision = exigirEntero(referenciaEstado.revision, "revisión de estado para ETag", 1);
  const huella = exigirHuella(
    referenciaEstado.huella_estado_sha256, "huella de estado para ETag",
  );
  return `"vec-borrador-v1.r${revision}.sha256-${huella}"`;
}

function exigirETag(valor, referenciaEstado = undefined) {
  if (typeof valor !== "string"
    || !/^"vec-borrador-v1\.r[1-9][0-9]{0,15}\.sha256-[a-f0-9]{64}"$/.test(valor)) {
    throw new Error("ETag fuerte no válido");
  }
  const coincidencia = /^"vec-borrador-v1\.r([1-9][0-9]{0,15})\.sha256-([a-f0-9]{64})"$/.exec(valor);
  if (!Number.isSafeInteger(Number(coincidencia[1]))) throw new Error("ETag fuerte no válido");
  if (referenciaEstado !== undefined && valor !== derivarETagBorrador(referenciaEstado)) {
    throw new Error("ETag no corresponde a revisión y huella de estado");
  }
  return valor;
}

export function validarETagBorrador(valor, referenciaEstado = undefined) {
  return exigirETag(valor, referenciaEstado);
}

function exigirLista(valor, nombre, maximo) {
  if (!Array.isArray(valor) || valor.length > maximo) throw new Error(`${nombre} no válida`);
  return valor;
}

function exigirListaUnica(valores, nombre, maximo, validarElemento, claveElemento) {
  const lista = exigirLista(valores, nombre, maximo).map(validarElemento);
  const vistos = new Set();
  for (const elemento of lista) {
    const clave = claveElemento(elemento);
    if (vistos.has(clave)) throw new Error(`${nombre} contiene elementos repetidos`);
    vistos.add(clave);
  }
  return lista;
}

function validarOpcionCatalogo(item, indice, nombre) {
  exigirCamposExactos(item, [
    `${nombre.slice(0, -1)}_ref`, "version", "huella_sha256", "clave", "etiqueta",
  ], `${nombre}[${indice}]`);
  const campoReferencia = `${nombre.slice(0, -1)}_ref`;
  return {
    [campoReferencia]: exigirReferencia(item[campoReferencia], `${nombre}[${indice}].${campoReferencia}`),
    version: exigirEntero(item.version, `${nombre}[${indice}].version`, 1),
    huella_sha256: exigirHuella(item.huella_sha256, `${nombre}[${indice}].huella_sha256`),
    clave: exigirClave(item.clave, `${nombre}[${indice}].clave`),
    etiqueta: exigirCadena(item.etiqueta, `${nombre}[${indice}].etiqueta`, 180),
  };
}

function exigirHuellaUnicaPorIdentidadCatalogo(elementos, nombre, campoReferencia) {
  const huellasPorIdentidad = new Map();
  for (const item of elementos) {
    const identidad = `${item[campoReferencia]}\u0000${item.version}`;
    const huellaAnterior = huellasPorIdentidad.get(identidad);
    if (huellaAnterior !== undefined && huellaAnterior !== item.huella_sha256) {
      throw new Error(`${nombre} contiene huellas contradictorias para catálogo y versión`);
    }
    huellasPorIdentidad.set(identidad, item.huella_sha256);
  }
}

function validarPlantilla(item, indice) {
  exigirCamposExactos(item, [
    "plantilla_ref", "version", "huella_sha256", "nombre", "descripcion",
  ], `plantillas[${indice}]`);
  return {
    plantilla_ref: exigirReferencia(item.plantilla_ref, `plantillas[${indice}].plantilla_ref`),
    version: exigirEntero(item.version, `plantillas[${indice}].version`, 1),
    huella_sha256: exigirHuella(item.huella_sha256, `plantillas[${indice}].huella_sha256`),
    nombre: exigirCadena(item.nombre, `plantillas[${indice}].nombre`, 180),
    descripcion: exigirCadena(item.descripcion, `plantillas[${indice}].descripcion`, 1000, true, true),
  };
}

function validarMotivo(item, indice) {
  exigirCamposExactos(item, [
    "motivo_ref", "version", "huella_sha256", "etiqueta", "descripcion",
  ], `motivos[${indice}]`);
  return {
    motivo_ref: exigirReferencia(item.motivo_ref, `motivos[${indice}].motivo_ref`),
    version: exigirEntero(item.version, `motivos[${indice}].version`, 1),
    huella_sha256: exigirHuella(item.huella_sha256, `motivos[${indice}].huella_sha256`),
    etiqueta: exigirCadena(item.etiqueta, `motivos[${indice}].etiqueta`, 180),
    descripcion: exigirCadena(item.descripcion, `motivos[${indice}].descripcion`, 1000, true, true),
  };
}

function validarLimites(limites) {
  exigirCamposExactos(limites, CAMPOS_LIMITES, "límites");
  const salida = {};
  for (const campo of CAMPOS_LIMITES) {
    salida[campo] = exigirEntero(
      limites[campo], `límites.${campo}`, 1, LIMITES_ABSOLUTOS_DOMINIO[campo],
    );
  }
  return salida;
}

function validarCapacidadesGlobales(capacidades) {
  exigirCamposExactos(capacidades, ["consultar", "crear"], "capacidades globales");
  return {
    consultar: exigirBooleano(capacidades.consultar, "capacidad consultar"),
    crear: exigirBooleano(capacidades.crear, "capacidad crear"),
  };
}

function validarCapacidadesFila(capacidades) {
  exigirCamposExactos(capacidades, ["consultar", "actualizar"], "capacidades del borrador");
  return {
    consultar: exigirBooleano(capacidades.consultar, "capacidad consultar"),
    actualizar: exigirBooleano(capacidades.actualizar, "capacidad actualizar"),
  };
}

export function extraerDatosEnvelopeBorradores(envelope) {
  exigirCamposExactos(envelope, ["data"], "envelope");
  if (!esObjeto(envelope.data)) throw new Error("la API debe responder con el envelope canónico {data:{...}}");
  return envelope.data;
}

export function extraerErrorEnvelopeBorradores(envelope) {
  exigirCamposExactos(envelope, ["error"], "envelope de error");
  exigirCamposExactos(envelope.error, ["codigo", "correlacion_ref"], "detalle de error");
  return {
    codigo: exigirClave(envelope.error.codigo, "código máquina de error"),
    correlacion_ref: exigirReferencia(
      envelope.error.correlacion_ref, "referencia de correlación de error", 180,
    ),
  };
}

export function validarOpcionesBorradores(datos) {
  exigirCamposExactos(datos, [
    "esquema", "categorias", "tipos", "plantillas", "motivos", "limites", "capacidades",
  ], "opciones de borradores");
  if (datos.esquema !== ESQUEMAS_BORRADORES.opciones) throw new Error("esquema de opciones no compatible");
  const categorias = exigirListaUnica(
    datos.categorias, "categorias", 10_000,
    (item, indice) => validarOpcionCatalogo(item, indice, "categorias"),
    (item) => `${item.categoria_ref}:${item.version}:${item.clave}`,
  );
  const tipos = exigirListaUnica(
    datos.tipos, "tipos", 4096,
    (item, indice) => validarOpcionCatalogo(item, indice, "tipos"),
    (item) => `${item.tipo_ref}:${item.version}:${item.clave}`,
  );
  exigirHuellaUnicaPorIdentidadCatalogo(
    categorias, "categorias", "categoria_ref",
  );
  exigirHuellaUnicaPorIdentidadCatalogo(tipos, "tipos", "tipo_ref");
  const plantillas = exigirListaUnica(
    datos.plantillas, "plantillas", 1024, validarPlantilla,
    (item) => `${item.plantilla_ref}:${item.version}`,
  );
  const motivos = exigirListaUnica(
    datos.motivos, "motivos", 1024, validarMotivo,
    (item) => `${item.motivo_ref}:${item.version}`,
  );
  if (new Set(categorias.map((item) => item.clave)).size !== categorias.length) {
    throw new Error("categorias contiene claves visibles ambiguas");
  }
  if (new Set(tipos.map((item) => item.clave)).size !== tipos.length) {
    throw new Error("tipos contiene claves visibles ambiguas");
  }
  return {
    esquema: datos.esquema,
    categorias,
    tipos,
    plantillas,
    motivos,
    limites: validarLimites(datos.limites),
    capacidades: validarCapacidadesGlobales(datos.capacidades),
  };
}

function validarReferenciaEstado(estado) {
  exigirCamposExactos(estado, ["referencia", "revision", "huella_estado_sha256"], "referencia de estado");
  descomponerReferenciaEstadoBorrador(estado.referencia);
  return {
    referencia: exigirReferencia(estado.referencia, "referencia de estado"),
    revision: exigirEntero(estado.revision, "revisión de estado", 1),
    huella_estado_sha256: exigirHuella(estado.huella_estado_sha256, "huella de estado"),
  };
}

function validarPlazo(plazo, indice, limites) {
  exigirCamposExactos(plazo, [
    "referencia", "tipo", "titulo", "descripcion", "abre_en", "cierra_en",
  ], `plazos[${indice}]`);
  const abreEn = exigirInstante(plazo.abre_en, `plazos[${indice}].abre_en`);
  const cierraEn = exigirInstante(plazo.cierra_en, `plazos[${indice}].cierra_en`);
  if (compararInstantes(abreEn, cierraEn) >= 0) throw new Error(`plazos[${indice}] no tiene un intervalo válido`);
  return {
    referencia: exigirReferencia(plazo.referencia, `plazos[${indice}].referencia`, 160),
    tipo: exigirClave(plazo.tipo, `plazos[${indice}].tipo`),
    titulo: exigirCadena(plazo.titulo, `plazos[${indice}].titulo`, limites.maximo_titulo_plazo),
    descripcion: exigirCadena(plazo.descripcion, `plazos[${indice}].descripcion`, limites.maximo_descripcion_plazo, false, true),
    abre_en: abreEn,
    cierra_en: cierraEn,
  };
}

function validarRequisito(requisito, indice, limites) {
  exigirCamposExactos(requisito, [
    "referencia", "orden", "titulo", "descripcion", "obligatorio",
  ], `requisitos[${indice}]`);
  return {
    referencia: exigirReferencia(requisito.referencia, `requisitos[${indice}].referencia`, 160),
    orden: exigirEntero(requisito.orden, `requisitos[${indice}].orden`, 1),
    titulo: exigirCadena(requisito.titulo, `requisitos[${indice}].titulo`, limites.maximo_titulo_requisito),
    descripcion: exigirCadena(requisito.descripcion, `requisitos[${indice}].descripcion`, limites.maximo_descripcion_requisito, false, true),
    obligatorio: exigirBooleano(requisito.obligatorio, `requisitos[${indice}].obligatorio`),
  };
}

function validarAyuda(ayuda, indice, limites) {
  exigirCamposExactos(ayuda, ["referencia", "categoria", "orden", "pregunta", "respuesta"], `ayuda[${indice}]`);
  return {
    referencia: exigirReferencia(ayuda.referencia, `ayuda[${indice}].referencia`, 160),
    categoria: exigirClave(ayuda.categoria, `ayuda[${indice}].categoria`),
    orden: exigirEntero(ayuda.orden, `ayuda[${indice}].orden`, 1),
    pregunta: exigirCadena(ayuda.pregunta, `ayuda[${indice}].pregunta`, limites.maximo_pregunta_ayuda),
    respuesta: exigirCadena(ayuda.respuesta, `ayuda[${indice}].respuesta`, limites.maximo_respuesta_ayuda, false, true),
  };
}

export function validarContenidoEditable(contenido, limites) {
  const limitesValidados = validarLimites(limites);
  exigirCamposExactos(contenido, CAMPOS_CONTENIDO, "contenido editable");
  const categorias = exigirListaUnica(
    contenido.categorias, "categorías del contenido", limitesValidados.maximo_categorias,
    (item, indice) => exigirClave(item, `categorias[${indice}]`),
    (item) => item,
  );
  if (categorias.length === 0) throw new Error("el contenido requiere al menos una categoría");
  const plazos = exigirListaUnica(
    contenido.plazos, "plazos", limitesValidados.maximo_plazos,
    (item, indice) => validarPlazo(item, indice, limitesValidados),
    (item) => item.referencia,
  );
  if (plazos.length === 0) throw new Error("el contenido requiere al menos un plazo");
  const requisitos = exigirListaUnica(
    contenido.requisitos, "requisitos", limitesValidados.maximo_requisitos,
    (item, indice) => validarRequisito(item, indice, limitesValidados),
    (item) => item.referencia,
  );
  if (new Set(requisitos.map((item) => item.orden)).size !== requisitos.length) {
    throw new Error("requisitos contiene órdenes repetidos");
  }
  const ayuda = exigirListaUnica(
    contenido.ayuda, "ayuda", limitesValidados.maximo_ayudas,
    (item, indice) => validarAyuda(item, indice, limitesValidados),
    (item) => item.referencia,
  );
  if (new Set(ayuda.map((item) => item.orden)).size !== ayuda.length) {
    throw new Error("ayuda contiene órdenes repetidos");
  }
  return {
    tipo: exigirClave(contenido.tipo, "tipo de convocatoria"),
    categorias,
    titulo: exigirCadena(contenido.titulo, "título", limitesValidados.maximo_titulo),
    resumen: exigirCadena(contenido.resumen, "resumen", limitesValidados.maximo_resumen),
    descripcion: exigirCadena(contenido.descripcion, "descripción", limitesValidados.maximo_descripcion, false, true),
    plazos,
    requisitos,
    ayuda,
  };
}

export function validarSelectorListaBorradores(selector) {
  exigirCamposExactos(
    selector,
    ["limite", "cursor", "texto", "categoria"],
    "selector de borradores",
    ["cursor", "texto", "categoria"],
  );
  const salida = { limite: exigirEntero(selector.limite, "límite", 1, 50) };
  if (Object.hasOwn(selector, "cursor")) salida.cursor = exigirCursor(selector.cursor);
  if (Object.hasOwn(selector, "texto")) salida.texto = exigirCadena(selector.texto, "texto de búsqueda", 180);
  if (Object.hasOwn(selector, "categoria")) salida.categoria = exigirClave(selector.categoria, "categoría del selector");
  return salida;
}

function validarPaginacion(paginacion) {
  exigirCamposExactos(
    paginacion,
    ["limite", "total", "siguiente_cursor"],
    "paginación",
    ["siguiente_cursor"],
  );
  const salida = {
    limite: exigirEntero(paginacion.limite, "paginación.límite", 1, 50),
    total: exigirEntero(paginacion.total, "paginación.total", 0),
  };
  if (Object.hasOwn(paginacion, "siguiente_cursor")) {
    salida.siguiente_cursor = exigirCursor(paginacion.siguiente_cursor);
  }
  return salida;
}

function validarFilaBorrador(item, indice) {
  exigirCamposExactos(item, [
    "referencia_estado", "etag", "codigo_version_publica", "identificador_publico",
    "titulo", "tipo", "categorias", "expediente_ref", "creada_en", "actualizada_en",
    "numero_plazos", "numero_requisitos", "numero_documentos", "numero_ayudas", "capacidades",
  ], `elementos[${indice}]`);
  const categorias = exigirListaUnica(
    item.categorias, `elementos[${indice}].categorias`, LIMITES_ABSOLUTOS_DOMINIO.maximo_categorias,
    (valor, posicion) => exigirClave(valor, `elementos[${indice}].categorias[${posicion}]`),
    (valor) => valor,
  );
  if (categorias.length === 0) throw new Error(`elementos[${indice}] requiere una categoría`);
  const creadaEn = exigirInstante(item.creada_en, `elementos[${indice}].creada_en`);
  const actualizadaEn = exigirInstante(item.actualizada_en, `elementos[${indice}].actualizada_en`);
  if (compararInstantes(creadaEn, actualizadaEn) > 0) {
    throw new Error(`elementos[${indice}] tiene fechas incoherentes`);
  }
  const referenciaEstado = validarReferenciaEstado(item.referencia_estado);
  return {
    referencia_estado: referenciaEstado,
    etag: exigirETag(item.etag, referenciaEstado),
    codigo_version_publica: exigirClave(item.codigo_version_publica, `elementos[${indice}].codigo_version_publica`),
    identificador_publico: exigirIdentificadorPublico(item.identificador_publico),
    titulo: exigirCadena(item.titulo, `elementos[${indice}].titulo`, LIMITES_ABSOLUTOS_DOMINIO.maximo_titulo),
    tipo: exigirClave(item.tipo, `elementos[${indice}].tipo`),
    categorias,
    expediente_ref: exigirReferencia(item.expediente_ref, `elementos[${indice}].expediente_ref`),
    creada_en: creadaEn,
    actualizada_en: actualizadaEn,
    numero_plazos: exigirEntero(item.numero_plazos, `elementos[${indice}].numero_plazos`, 0, LIMITES_ABSOLUTOS_DOMINIO.maximo_plazos),
    numero_requisitos: exigirEntero(item.numero_requisitos, `elementos[${indice}].numero_requisitos`, 0, LIMITES_ABSOLUTOS_DOMINIO.maximo_requisitos),
    numero_documentos: exigirEntero(item.numero_documentos, `elementos[${indice}].numero_documentos`, 0, LIMITES_ABSOLUTOS_DOMINIO.maximo_documentos),
    numero_ayudas: exigirEntero(item.numero_ayudas, `elementos[${indice}].numero_ayudas`, 0, LIMITES_ABSOLUTOS_DOMINIO.maximo_ayudas),
    capacidades: validarCapacidadesFila(item.capacidades),
  };
}

export function validarListaBorradores(datos) {
  exigirCamposExactos(datos, [
    "esquema", "selector", "paginacion", "capacidades", "elementos",
  ], "lista de borradores");
  if (datos.esquema !== ESQUEMAS_BORRADORES.lista) throw new Error("esquema de lista no compatible");
  const selector = validarSelectorListaBorradores(datos.selector);
  const paginacion = validarPaginacion(datos.paginacion);
  if (selector.limite !== paginacion.limite) throw new Error("selector y paginación no coinciden");
  const elementos = exigirListaUnica(
    datos.elementos, "elementos", 50, validarFilaBorrador,
    (item) => item.referencia_estado.referencia,
  );
  if (elementos.length > paginacion.total || elementos.length > selector.limite) {
    throw new Error("número de elementos incoherente");
  }
  if ((paginacion.total === 0) !== (elementos.length === 0)) {
    throw new Error("paginación y elementos no representan el mismo vacío");
  }
  if (Object.hasOwn(paginacion, "siguiente_cursor")
    && (paginacion.total === 0 || elementos.length === 0
      || paginacion.total <= elementos.length)) {
    throw new Error("paginación con siguiente cursor incoherente");
  }
  return {
    esquema: datos.esquema,
    selector,
    paginacion,
    capacidades: validarCapacidadesGlobales(datos.capacidades),
    elementos,
  };
}

function validarReferenciaConfiguracion(valor, nombre) {
  exigirCamposExactos(valor, ["referencia", "version", "huella_sha256"], nombre);
  return {
    referencia: exigirReferencia(valor.referencia, `${nombre}.referencia`),
    version: exigirEntero(valor.version, `${nombre}.version`, 1),
    huella_sha256: exigirHuella(valor.huella_sha256, `${nombre}.huella_sha256`),
  };
}

function validarDocumentoLectura(documento, indice) {
  exigirCamposExactos(documento, [
    "rol", "publicacion_ref", "documento_ref", "version_documento",
    "representacion_ref", "huella_contenido_sha256", "firma_validada_ref", "recibo_custodia_ref",
  ], `configuracion_lectura.documentos[${indice}]`);
  return {
    rol: exigirClave(documento.rol, `documentos[${indice}].rol`),
    publicacion_ref: exigirReferencia(documento.publicacion_ref, `documentos[${indice}].publicacion_ref`),
    documento_ref: exigirReferencia(documento.documento_ref, `documentos[${indice}].documento_ref`),
    version_documento: exigirEntero(documento.version_documento, `documentos[${indice}].version_documento`, 1),
    representacion_ref: exigirReferencia(documento.representacion_ref, `documentos[${indice}].representacion_ref`),
    huella_contenido_sha256: exigirHuella(documento.huella_contenido_sha256, `documentos[${indice}].huella_contenido_sha256`),
    firma_validada_ref: exigirReferencia(documento.firma_validada_ref, `documentos[${indice}].firma_validada_ref`),
    recibo_custodia_ref: exigirReferencia(documento.recibo_custodia_ref, `documentos[${indice}].recibo_custodia_ref`),
  };
}

function validarConfiguracionLectura(configuracion, limites) {
  exigirCamposExactos(configuracion, [
    "catalogos", "calendario", "reglas_baremacion", "flujo_proceso",
    "flujo_solicitud", "plantilla", "documentos",
  ], "configuración de solo lectura");
  const documentos = exigirListaUnica(
    configuracion.documentos, "configuración.documentos", limites.maximo_documentos, validarDocumentoLectura,
    (item) => item.publicacion_ref,
  );
  if (documentos.length === 0) throw new Error("la configuración requiere documentos gobernados");
  const identidades = [
    ["documento y versión", (item) => `${item.documento_ref}#${item.version_documento}`],
    ["representación", (item) => item.representacion_ref],
    ["firma", (item) => item.firma_validada_ref],
    ["custodia", (item) => item.recibo_custodia_ref],
  ];
  for (const [nombre, obtenerClave] of identidades) {
    if (new Set(documentos.map(obtenerClave)).size !== documentos.length) {
      throw new Error(`configuración.documentos repite ${nombre}`);
    }
  }
  return {
    catalogos: validarReferenciaConfiguracion(configuracion.catalogos, "configuración.catalogos"),
    calendario: validarReferenciaConfiguracion(configuracion.calendario, "configuración.calendario"),
    reglas_baremacion: validarReferenciaConfiguracion(configuracion.reglas_baremacion, "configuración.reglas_baremacion"),
    flujo_proceso: validarReferenciaConfiguracion(configuracion.flujo_proceso, "configuración.flujo_proceso"),
    flujo_solicitud: validarReferenciaConfiguracion(configuracion.flujo_solicitud, "configuración.flujo_solicitud"),
    plantilla: validarReferenciaConfiguracion(configuracion.plantilla, "configuración.plantilla"),
    documentos,
  };
}

function validarAmbitoLectura(ambito) {
  exigirCamposExactos(ambito, ["organizacion_ref", "unidad_gestion_ref"], "ámbito de solo lectura", ["unidad_gestion_ref"]);
  const salida = { organizacion_ref: exigirReferencia(ambito.organizacion_ref, "ámbito.organizacion_ref") };
  if (Object.hasOwn(ambito, "unidad_gestion_ref")) {
    salida.unidad_gestion_ref = exigirReferencia(ambito.unidad_gestion_ref, "ámbito.unidad_gestion_ref");
  }
  return salida;
}

export function validarDetalleBorrador(datos, limites) {
  exigirCamposExactos(datos, [
    "esquema", "referencia_estado", "etag", "codigo_version_publica", "identificador_publico",
    "ambito_lectura", "expediente_ref", "contenido_editable", "configuracion_lectura", "capacidades",
  ], "detalle de borrador");
  if (datos.esquema !== ESQUEMAS_BORRADORES.detalle) throw new Error("esquema de detalle no compatible");
  const limitesValidados = validarLimites(limites);
  const referenciaEstado = validarReferenciaEstado(datos.referencia_estado);
  const etag = exigirETag(datos.etag, referenciaEstado);
  return {
    esquema: datos.esquema,
    referencia_estado: referenciaEstado,
    etag,
    codigo_version_publica: exigirClave(datos.codigo_version_publica, "código de versión pública"),
    identificador_publico: exigirIdentificadorPublico(datos.identificador_publico),
    ambito_lectura: validarAmbitoLectura(datos.ambito_lectura),
    expediente_ref: exigirReferencia(datos.expediente_ref, "referencia de expediente"),
    contenido_editable: validarContenidoEditable(datos.contenido_editable, limitesValidados),
    configuracion_lectura: validarConfiguracionLectura(datos.configuracion_lectura, limitesValidados),
    capacidades: validarCapacidadesFila(datos.capacidades),
  };
}

export function validarSolicitudCrearBorrador(datos, limites) {
  exigirCamposExactos(datos, [
    "esquema", "plantilla_ref", "plantilla_version", "plantilla_huella_sha256",
    "codigo_version_publica", "identificador_publico", "expediente_ref", "contenido_editable",
    "motivo_ref", "motivo_version", "motivo_huella_sha256",
  ], "solicitud de alta de borrador");
  if (datos.esquema !== ESQUEMAS_BORRADORES.crear) throw new Error("esquema de alta no compatible");
  return {
    esquema: datos.esquema,
    plantilla_ref: exigirReferencia(datos.plantilla_ref, "referencia de plantilla"),
    plantilla_version: exigirEntero(datos.plantilla_version, "versión de plantilla", 1),
    plantilla_huella_sha256: exigirHuella(datos.plantilla_huella_sha256, "huella de plantilla"),
    codigo_version_publica: exigirClave(datos.codigo_version_publica, "código de versión pública"),
    identificador_publico: exigirIdentificadorPublico(datos.identificador_publico),
    expediente_ref: exigirReferencia(datos.expediente_ref, "referencia de expediente"),
    contenido_editable: validarContenidoEditable(datos.contenido_editable, limites),
    motivo_ref: exigirReferencia(datos.motivo_ref, "referencia de motivo"),
    motivo_version: exigirEntero(datos.motivo_version, "versión de motivo", 1),
    motivo_huella_sha256: exigirHuella(datos.motivo_huella_sha256, "huella de motivo"),
  };
}

export function validarSolicitudActualizarBorrador(datos, limites) {
  exigirCamposExactos(datos, [
    "esquema", "contenido_editable", "motivo_ref", "motivo_version", "motivo_huella_sha256",
  ], "solicitud de actualización de borrador");
  if (datos.esquema !== ESQUEMAS_BORRADORES.actualizar) throw new Error("esquema de actualización no compatible");
  return {
    esquema: datos.esquema,
    contenido_editable: validarContenidoEditable(datos.contenido_editable, limites),
    motivo_ref: exigirReferencia(datos.motivo_ref, "referencia de motivo"),
    motivo_version: exigirEntero(datos.motivo_version, "versión de motivo", 1),
    motivo_huella_sha256: exigirHuella(datos.motivo_huella_sha256, "huella de motivo"),
  };
}

export function validarReciboGuardadoBorrador(datos) {
  exigirCamposExactos(datos, [
    "esquema", "transaccion_ref", "accion", "referencia_estado", "etag",
    "auditoria_ref", "evento_outbox_ref", "confirmada_en",
  ], "recibo de guardado");
  if (datos.esquema !== ESQUEMAS_BORRADORES.guardado) throw new Error("esquema de recibo no compatible");
  if (datos.accion !== "crear" && datos.accion !== "actualizar") throw new Error("acción del recibo no válida");
  const referenciaEstado = validarReferenciaEstado(datos.referencia_estado);
  const etag = exigirETag(datos.etag, referenciaEstado);
  return {
    esquema: datos.esquema,
    transaccion_ref: exigirReferencia(datos.transaccion_ref, "referencia de transacción"),
    accion: datos.accion,
    referencia_estado: referenciaEstado,
    etag,
    auditoria_ref: exigirReferencia(datos.auditoria_ref, "referencia de auditoría"),
    evento_outbox_ref: exigirReferencia(datos.evento_outbox_ref, "referencia de evento"),
    confirmada_en: exigirInstante(datos.confirmada_en, "fecha de confirmación"),
  };
}
