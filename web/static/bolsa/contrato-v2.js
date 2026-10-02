"use strict";

((global) => {
  const PATRON_CLAVE = /^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$/;
  const PATRON_REFERENCIA_CATALOGO = /^[a-z][a-z0-9._-]{0,127}$/;
  const PATRON_HUELLA = /^[a-f0-9]{64}$/;
  const PATRON_IDENTIFICADOR = /^[a-z0-9][a-z0-9-]{2,79}$/;
  const PATRON_CLAVE_CATALOGO = /^[A-Za-z0-9][A-Za-z0-9._-]{0,79}$/;
  const PATRON_FECHA_UTC = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$/;
  const MAXIMA_VERSION_CATALOGO = 2147483647;
  // La respuesta no expone siempre el tamaño de página: admite hasta 4.096
  // entradas históricas (límite agregado de snapshots del contrato C3).
  const MAXIMO_DICCIONARIO_CATEGORIAS = 4096;

  function esObjeto(valor) {
    return valor !== null && typeof valor === "object" && !Array.isArray(valor);
  }

  function textoValido(valor, permitirVacio = false) {
    return typeof valor === "string" && (permitirVacio || valor.trim() !== "");
  }

  function fechaValida(valor) {
    return typeof valor === "string" && PATRON_FECHA_UTC.test(valor) &&
      !Number.isNaN(new Date(valor).getTime());
  }

  function enteroNoNegativo(valor) {
    return Number.isSafeInteger(valor) && valor >= 0;
  }

  function validarValorCatalogo(valor) {
    if (!esObjeto(valor) || typeof valor.clave !== "string" || !PATRON_CLAVE_CATALOGO.test(valor.clave) ||
        !Number.isSafeInteger(valor.version) || valor.version < 1 ||
        !textoValido(valor.etiqueta) || !textoValido(valor.semantica) ||
        (valor.descripcion !== undefined && !textoValido(valor.descripcion, true))) {
      throw new Error("valor de catálogo público inválido");
    }
  }

  function validarFuente(fuente) {
    if (!esObjeto(fuente) || !textoValido(fuente.revision) ||
        !fechaValida(fuente.actualizada_en) || typeof fuente.demostracion !== "boolean") {
      throw new Error("fuente pública inválida");
    }
  }

  function validarPaginacion(paginacion) {
    if (!esObjeto(paginacion) || !Number.isSafeInteger(paginacion.pagina) || paginacion.pagina < 1 ||
        !Number.isSafeInteger(paginacion.tamano) || paginacion.tamano < 1 || paginacion.tamano > 24 ||
        !enteroNoNegativo(paginacion.total) || !enteroNoNegativo(paginacion.paginas)) {
      throw new Error("paginación pública inválida");
    }
  }

  function validarPlazo(plazo) {
    if (!esObjeto(plazo) || !textoValido(plazo.titulo) || !fechaValida(plazo.abre_en) ||
        !fechaValida(plazo.cierra_en) || !textoValido(plazo.etiqueta_situacion) ||
        !textoValido(plazo.semantica_situacion) ||
        (plazo.descripcion !== undefined && !textoValido(plazo.descripcion, true))) {
      throw new Error("plazo público inválido");
    }
    validarValorCatalogo(plazo.tipo);
  }

  function validarConvocatoria(convocatoria) {
    if (!esObjeto(convocatoria) || typeof convocatoria.identificador_publico !== "string" ||
        !PATRON_IDENTIFICADOR.test(convocatoria.identificador_publico) ||
        !textoValido(convocatoria.version) || typeof convocatoria.huella_sha256 !== "string" ||
        !PATRON_HUELLA.test(convocatoria.huella_sha256) || !textoValido(convocatoria.titulo) ||
        !textoValido(convocatoria.resumen) || !fechaValida(convocatoria.publicada_en) ||
        !Array.isArray(convocatoria.categorias) ||
        !enteroNoNegativo(convocatoria.numero_requisitos) ||
        !enteroNoNegativo(convocatoria.numero_documentos) || !enteroNoNegativo(convocatoria.numero_ayudas)) {
      throw new Error("convocatoria pública inválida");
    }
    validarValorCatalogo(convocatoria.tipo);
    validarValorCatalogo(convocatoria.estado);
    validarCatalogoCategorias(convocatoria.catalogo_categorias);
    if (convocatoria.plazo_destacado !== undefined && convocatoria.plazo_destacado !== null) {
      validarPlazo(convocatoria.plazo_destacado);
    }
  }

  // Misma frontera que urlDocumentoPublicoValida de Go: ruta local literal,
  // sin escape porcentual que cambie u.Path ni componentes de URL adicionales.
  function urlDocumentoPublicoValida(valor) {
    if (typeof valor !== "string" || !valor.startsWith("/bolsa/documentos/") ||
        valor.trim() !== valor || /[\\?#%\u0000-\u001f\u007f]/u.test(valor) ||
        valor.split("/").some((segmento) => segmento === "." || segmento === "..")) {
      return false;
    }
    try {
      return encodeURIComponent(valor).replace(/%[0-9A-F]{2}/gu, "x").length <= 240;
    } catch {
      return false;
    }
  }

  function validarDocumento(documento) {
    if (!esObjeto(documento) || !textoValido(documento.titulo) ||
        !textoValido(documento.descripcion) || typeof documento.formato !== "string" ||
        !PATRON_CLAVE_CATALOGO.test(documento.formato) ||
        !urlDocumentoPublicoValida(documento.url)) {
      throw new Error("documento público inválido");
    }
    validarValorCatalogo(documento.tipo);
  }

  function validarRequisito(requisito) {
    if (!esObjeto(requisito) || !textoValido(requisito.titulo) ||
        !textoValido(requisito.descripcion) || typeof requisito.obligatorio !== "boolean") {
      throw new Error("requisito público inválido");
    }
  }

  function validarAyuda(ayuda) {
    if (!esObjeto(ayuda) || !textoValido(ayuda.pregunta) || !textoValido(ayuda.respuesta)) {
      throw new Error("ayuda pública inválida");
    }
    validarValorCatalogo(ayuda.categoria);
  }

  function claveReferencia(referencia) {
    if (!referencia || typeof referencia !== "object" ||
        typeof referencia.clave !== "string" || referencia.clave.length > 80 ||
        !PATRON_CLAVE.test(referencia.clave) ||
        !Number.isSafeInteger(referencia.version) || referencia.version < 1 ||
        referencia.version > MAXIMA_VERSION_CATALOGO) {
      throw new Error("referencia de categoría inválida");
    }
    return `${referencia.version}:${referencia.clave}`;
  }

  function validarCatalogoCategorias(catalogo) {
    if (!catalogo || typeof catalogo !== "object" ||
        typeof catalogo.catalogo_id !== "string" || !PATRON_REFERENCIA_CATALOGO.test(catalogo.catalogo_id) ||
        !Number.isSafeInteger(catalogo.version) || catalogo.version < 1 ||
        catalogo.version > MAXIMA_VERSION_CATALOGO ||
        typeof catalogo.huella_sha256 !== "string" || !PATRON_HUELLA.test(catalogo.huella_sha256) ||
        typeof catalogo.huella_proyeccion_sha256 !== "string" || !PATRON_HUELLA.test(catalogo.huella_proyeccion_sha256)) {
      throw new Error("snapshot de categorías inválido");
    }
    return `${catalogo.catalogo_id}:${catalogo.version}:${catalogo.huella_sha256}:${catalogo.huella_proyeccion_sha256}`;
  }

  function claveHuellaGobernada(catalogo) {
    return `${catalogo.catalogo_id}:${catalogo.version}:${catalogo.huella_sha256}`;
  }

  function crearDiccionario(entradas) {
    if (!Array.isArray(entradas) || entradas.length > MAXIMO_DICCIONARIO_CATEGORIAS) {
      throw new Error("diccionario de categorías ausente o excesivo");
    }
    const porReferencia = new Map();
    const proyeccionPorHuellaGobernada = new Map();
    entradas.forEach((entrada) => {
      const referencia = claveReferencia(entrada);
      const snapshot = validarCatalogoCategorias(entrada.catalogo_categorias);
      const huellaGobernada = claveHuellaGobernada(entrada.catalogo_categorias);
      const proyeccionRegistrada = proyeccionPorHuellaGobernada.get(huellaGobernada);
      const clave = `${snapshot}:${referencia}`;
      if (entrada.version !== entrada.catalogo_categorias.version ||
          (proyeccionRegistrada !== undefined && proyeccionRegistrada !== snapshot) ||
          typeof entrada.etiqueta !== "string" || entrada.etiqueta.length === 0 ||
          typeof entrada.semantica !== "string" || entrada.semantica.length === 0 ||
          (entrada.descripcion !== undefined && !textoValido(entrada.descripcion, true)) ||
          porReferencia.has(clave)) {
        throw new Error("diccionario de categorías ambiguo");
      }
      proyeccionPorHuellaGobernada.set(huellaGobernada, snapshot);
      porReferencia.set(clave, { entrada, snapshot });
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
      const clave = `${catalogoEsperado || ""}:${claveReferencia(referencia)}`;
      if (vistas.has(clave) || !diccionario.has(clave)) {
        throw new Error("categoría desconocida, duplicada o incoherente con su snapshot");
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
    if (!esObjeto(datos) || datos.esquema !== "vec.bolsa.publico.convocatorias.v2" ||
        !esObjeto(datos.facetas) || !Array.isArray(datos.facetas.tipos) ||
        !Array.isArray(datos.facetas.categorias) || !Array.isArray(datos.facetas.estados) ||
        !Array.isArray(datos.diccionario_categorias) || !Array.isArray(datos.convocatorias)) {
      throw new Error("esquema público inesperado");
    }
    validarFuente(datos.fuente);
    validarPaginacion(datos.paginacion);
    datos.facetas.tipos.forEach(validarValorCatalogo);
    datos.facetas.estados.forEach(validarValorCatalogo);
    datos.facetas.categorias.forEach((categoria) => {
      validarValorCatalogo(categoria);
      if (!enteroNoNegativo(categoria.numero_resultados)) {
        throw new Error("faceta de categoría inválida");
      }
    });
    const diccionario = crearDiccionario(datos.diccionario_categorias);
    const categoriasPorConvocatoria = new Map();
    datos.convocatorias.forEach((convocatoria) => {
      validarConvocatoria(convocatoria);
      if (categoriasPorConvocatoria.has(convocatoria.identificador_publico)) {
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
    if (!esObjeto(datos) || datos.esquema !== "vec.bolsa.publico.convocatoria.v2" ||
        !textoValido(datos.descripcion) || !Array.isArray(datos.plazos) ||
        !Array.isArray(datos.requisitos) || !Array.isArray(datos.documentos) || !Array.isArray(datos.ayuda)) {
      throw new Error("esquema de detalle inesperado");
    }
    validarFuente(datos.fuente);
    validarConvocatoria(datos.convocatoria);
    datos.plazos.forEach(validarPlazo);
    datos.requisitos.forEach(validarRequisito);
    datos.documentos.forEach(validarDocumento);
    datos.ayuda.forEach(validarAyuda);
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
    urlDocumentoPublicoValida,
    validarListado,
    validarDetalle,
  });
})(globalThis);
