"use strict";

((global) => {
  const PATRON_CLAVE = /^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$/;

  function claveReferencia(referencia) {
    if (!referencia || typeof referencia !== "object" ||
        typeof referencia.clave !== "string" || referencia.clave.length > 80 ||
        !PATRON_CLAVE.test(referencia.clave) ||
        !Number.isSafeInteger(referencia.version) || referencia.version < 1) {
      throw new Error("referencia de categoría inválida");
    }
    return `${referencia.version}:${referencia.clave}`;
  }

  function crearDiccionario(entradas) {
    if (!Array.isArray(entradas) || entradas.length > 1024) {
      throw new Error("diccionario de categorías ausente o excesivo");
    }
    const porReferencia = new Map();
    const claves = new Set();
    entradas.forEach((entrada) => {
      const referencia = claveReferencia(entrada);
      if (typeof entrada.etiqueta !== "string" || entrada.etiqueta.length === 0 ||
          typeof entrada.semantica !== "string" || entrada.semantica.length === 0 ||
          porReferencia.has(referencia) || claves.has(entrada.clave)) {
        throw new Error("diccionario de categorías ambiguo");
      }
      porReferencia.set(referencia, entrada);
      claves.add(entrada.clave);
    });
    return porReferencia;
  }

  function resolverReferencias(referencias, diccionario, exigirExactitud = false) {
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
      return diccionario.get(clave);
    });
    if (exigirExactitud && vistas.size !== diccionario.size) {
      throw new Error("diccionario de detalle no biyectivo");
    }
    return resultado;
  }

  function validarListado(datos) {
    if (!datos || datos.esquema !== "vec.bolsa.publico.convocatorias.v2" ||
        !datos.facetas || !Array.isArray(datos.convocatorias)) {
      throw new Error("esquema público inesperado");
    }
    const diccionario = crearDiccionario(datos.facetas.categorias);
    const categoriasPorConvocatoria = new Map();
    datos.convocatorias.forEach((convocatoria) => {
      if (!convocatoria || typeof convocatoria.identificador_publico !== "string" ||
          categoriasPorConvocatoria.has(convocatoria.identificador_publico)) {
        throw new Error("convocatoria pública ambigua");
      }
      categoriasPorConvocatoria.set(
        convocatoria.identificador_publico,
        resolverReferencias(convocatoria.categorias, diccionario),
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
    return resolverReferencias(datos.convocatoria.categorias, diccionario, true);
  }

  global.VECBolsaContratoV2 = Object.freeze({
    crearDiccionario,
    resolverReferencias,
    validarListado,
    validarDetalle,
  });
})(globalThis);
