"use strict";

((global) => {
  const PATRON_CLAVE = /^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$/;
  const PATRON_HUELLA = /^[a-f0-9]{64}$/;
  // La respuesta no expone siempre el tamaño de página: admite hasta 4.096
  // entradas históricas (límite agregado de snapshots del contrato C3).
  const MAXIMO_DICCIONARIO_CATEGORIAS = 4096;

  function claveReferencia(referencia) {
    if (!referencia || typeof referencia !== "object" ||
        typeof referencia.clave !== "string" || referencia.clave.length > 80 ||
        !PATRON_CLAVE.test(referencia.clave) ||
        !Number.isSafeInteger(referencia.version) || referencia.version < 1) {
      throw new Error("referencia de categoría inválida");
    }
    return `${referencia.version}:${referencia.clave}`;
  }

  function validarCatalogoCategorias(catalogo) {
    if (!catalogo || typeof catalogo !== "object" ||
        typeof catalogo.referencia !== "string" || catalogo.referencia.length === 0 ||
        catalogo.referencia.length > 160 ||
        !Number.isSafeInteger(catalogo.version) || catalogo.version < 1 ||
        typeof catalogo.huella_sha256 !== "string" || !PATRON_HUELLA.test(catalogo.huella_sha256)) {
      throw new Error("snapshot de categorías inválido");
    }
    return `${catalogo.referencia}:${catalogo.version}:${catalogo.huella_sha256}`;
  }

  function crearDiccionario(entradas) {
    if (!Array.isArray(entradas) || entradas.length > MAXIMO_DICCIONARIO_CATEGORIAS) {
      throw new Error("diccionario de categorías ausente o excesivo");
    }
    const porReferencia = new Map();
    entradas.forEach((entrada) => {
      const referencia = claveReferencia(entrada);
      const snapshot = validarCatalogoCategorias(entrada.catalogo_categorias);
      if (typeof entrada.etiqueta !== "string" || entrada.etiqueta.length === 0 ||
          typeof entrada.semantica !== "string" || entrada.semantica.length === 0 ||
          porReferencia.has(referencia)) {
        throw new Error("diccionario de categorías ambiguo");
      }
      porReferencia.set(referencia, { entrada, snapshot });
    });
    return porReferencia;
  }

  function resolverReferencias(referencias, diccionario, exigirExactitud = false, catalogoEsperado = null) {
    if (!Array.isArray(referencias) || referencias.length < 1 || referencias.length > 128 ||
        !(diccionario instanceof Map)) {
      throw new Error("categorías de convocatoria inválidas");
    }
    const vistas = new Set();
    const resultado = referencias.map((referencia) => {
      const clave = claveReferencia(referencia);
      if (vistas.has(clave) || !diccionario.has(clave)) {
        throw new Error("categoría desconocida o duplicada");
      }
      vistas.add(clave);
      const registrada = diccionario.get(clave);
      if (catalogoEsperado && registrada.snapshot !== catalogoEsperado) {
        throw new Error("categoría incoherente con su snapshot");
      }
      return registrada.entrada;
    });
    if (exigirExactitud && vistas.size !== diccionario.size) {
      throw new Error("diccionario de detalle no biyectivo");
    }
    return resultado;
  }

  function validarListado(datos) {
    if (!datos || datos.esquema !== "vec.bolsa.publico.convocatorias.v2" ||
        !datos.facetas || !Array.isArray(datos.facetas.categorias) ||
        !Array.isArray(datos.diccionario_categorias) || !Array.isArray(datos.convocatorias)) {
      throw new Error("esquema público inesperado");
    }
    const diccionario = crearDiccionario(datos.diccionario_categorias);
    const categoriasPorConvocatoria = new Map();
    datos.convocatorias.forEach((convocatoria) => {
      if (!convocatoria || typeof convocatoria.identificador_publico !== "string" ||
          categoriasPorConvocatoria.has(convocatoria.identificador_publico)) {
        throw new Error("convocatoria pública ambigua");
      }
      const catalogo = validarCatalogoCategorias(convocatoria.catalogo_categorias);
      categoriasPorConvocatoria.set(
        convocatoria.identificador_publico,
        resolverReferencias(convocatoria.categorias, diccionario, false, catalogo),
      );
    });
    return categoriasPorConvocatoria;
  }

  function validarDetalle(datos) {
    if (!datos || datos.esquema !== "vec.bolsa.publico.convocatoria.v2" || !datos.convocatoria) {
      throw new Error("esquema de detalle inesperado");
    }
    const diccionario = crearDiccionario(datos.diccionario_categorias);
    if (diccionario.size < 1 || diccionario.size > 128) {
      throw new Error("diccionario de detalle fuera de límites");
    }
    return resolverReferencias(
      datos.convocatoria.categorias,
      diccionario,
      true,
      validarCatalogoCategorias(datos.convocatoria.catalogo_categorias),
    );
  }

  global.VECBolsaContratoV2 = Object.freeze({
    crearDiccionario,
    validarCatalogoCategorias,
    resolverReferencias,
    validarListado,
    validarDetalle,
  });
})(globalThis);
