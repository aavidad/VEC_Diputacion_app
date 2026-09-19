"use strict";

// El runtime público vigente devuelve categorías ya resueltas dentro de cada
// convocatoria. Este contrato las comprueba sin reinterpretarlas contra un
// catálogo local o histórico que el esquema v1 no entrega.
((global) => {
  const PATRON_CLAVE = /^[A-Za-z0-9][A-Za-z0-9._-]{0,79}$/;
  const PATRON_IDENTIFICADOR = /^[a-z0-9][a-z0-9-]{2,79}$/;
  const PATRON_HUELLA = /^[a-f0-9]{64}$/;

  function esObjeto(valor) {
    return valor !== null && typeof valor === "object" && !Array.isArray(valor);
  }

  function fechaValida(valor) {
    return typeof valor === "string" && valor.length > 0 && !Number.isNaN(new Date(valor).getTime());
  }

  function validarValorCatalogo(valor) {
    if (!esObjeto(valor) || typeof valor.clave !== "string" || !PATRON_CLAVE.test(valor.clave) ||
        !Number.isSafeInteger(valor.version) || valor.version < 1 ||
        typeof valor.etiqueta !== "string" || valor.etiqueta.trim() === "" ||
        typeof valor.semantica !== "string" || valor.semantica.trim() === "") {
      throw new Error("valor de catálogo público inválido");
    }
    return valor;
  }

  function validarPlazo(plazo) {
    if (!esObjeto(plazo) || typeof plazo.titulo !== "string" || plazo.titulo.trim() === "" ||
        !fechaValida(plazo.abre_en) || !fechaValida(plazo.cierra_en) ||
        typeof plazo.etiqueta_situacion !== "string" || plazo.etiqueta_situacion.trim() === "" ||
        typeof plazo.semantica_situacion !== "string" || plazo.semantica_situacion.trim() === "") {
      throw new Error("plazo público inválido");
    }
    validarValorCatalogo(plazo.tipo);
    return plazo;
  }

  function validarConvocatoria(convocatoria) {
    if (!esObjeto(convocatoria) || typeof convocatoria.identificador_publico !== "string" ||
        !PATRON_IDENTIFICADOR.test(convocatoria.identificador_publico) ||
        typeof convocatoria.version !== "string" || convocatoria.version.trim() === "" ||
        typeof convocatoria.huella_sha256 !== "string" || !PATRON_HUELLA.test(convocatoria.huella_sha256) ||
        typeof convocatoria.titulo !== "string" || convocatoria.titulo.trim() === "" ||
        typeof convocatoria.resumen !== "string" || !fechaValida(convocatoria.publicada_en) ||
        !Array.isArray(convocatoria.categorias) ||
        !Number.isSafeInteger(convocatoria.numero_requisitos) || convocatoria.numero_requisitos < 0 ||
        !Number.isSafeInteger(convocatoria.numero_documentos) || convocatoria.numero_documentos < 0 ||
        !Number.isSafeInteger(convocatoria.numero_ayudas) || convocatoria.numero_ayudas < 0) {
      throw new Error("convocatoria pública inválida");
    }
    validarValorCatalogo(convocatoria.tipo);
    validarValorCatalogo(convocatoria.estado);
    convocatoria.categorias.forEach(validarValorCatalogo);
    if (convocatoria.plazo_destacado !== undefined && convocatoria.plazo_destacado !== null) {
      validarPlazo(convocatoria.plazo_destacado);
    }
    return convocatoria;
  }

  function validarFuente(fuente) {
    if (!esObjeto(fuente) || typeof fuente.revision !== "string" || fuente.revision.trim() === "" ||
        !fechaValida(fuente.actualizada_en) || typeof fuente.demostracion !== "boolean") {
      throw new Error("fuente pública inválida");
    }
    return fuente;
  }

  function validarPaginacion(paginacion) {
    if (!esObjeto(paginacion) || !Number.isSafeInteger(paginacion.pagina) || paginacion.pagina < 1 ||
        !Number.isSafeInteger(paginacion.tamano) || paginacion.tamano < 1 || paginacion.tamano > 24 ||
        !Number.isSafeInteger(paginacion.total) || paginacion.total < 0 ||
        !Number.isSafeInteger(paginacion.paginas) || paginacion.paginas < 0) {
      throw new Error("paginación pública inválida");
    }
    return paginacion;
  }

  function validarListado(datos) {
    if (!esObjeto(datos) || datos.esquema !== "vec.bolsa.publico.convocatorias.v1" ||
        !esObjeto(datos.facetas) || !Array.isArray(datos.facetas.tipos) ||
        !Array.isArray(datos.facetas.categorias) || !Array.isArray(datos.facetas.estados) ||
        !Array.isArray(datos.convocatorias)) {
      throw new Error("esquema público v1 inesperado");
    }
    validarFuente(datos.fuente);
    validarPaginacion(datos.paginacion);
    datos.facetas.tipos.forEach(validarValorCatalogo);
    datos.facetas.categorias.forEach(validarValorCatalogo);
    datos.facetas.estados.forEach(validarValorCatalogo);
    const categoriasPorConvocatoria = new Map();
    datos.convocatorias.forEach((convocatoria) => {
      validarConvocatoria(convocatoria);
      if (categoriasPorConvocatoria.has(convocatoria.identificador_publico)) {
        throw new Error("convocatoria pública ambigua");
      }
      categoriasPorConvocatoria.set(convocatoria.identificador_publico, convocatoria.categorias);
    });
    return categoriasPorConvocatoria;
  }

  function validarDetalle(datos) {
    if (!esObjeto(datos) || datos.esquema !== "vec.bolsa.publico.convocatoria.v1" ||
        !Array.isArray(datos.plazos) || !Array.isArray(datos.requisitos) ||
        !Array.isArray(datos.documentos) || !Array.isArray(datos.ayuda)) {
      throw new Error("esquema de detalle v1 inesperado");
    }
    validarFuente(datos.fuente);
    const convocatoria = validarConvocatoria(datos.convocatoria);
    datos.plazos.forEach(validarPlazo);
    return convocatoria.categorias;
  }

  global.VECBolsaContratoV1 = Object.freeze({
    validarValorCatalogo,
    validarConvocatoria,
    validarListado,
    validarDetalle,
  });
})(globalThis);
