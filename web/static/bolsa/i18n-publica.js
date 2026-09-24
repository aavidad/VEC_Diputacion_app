/** Catálogo común de las superficies públicas de Bolsa. Sin estado persistente. */
(function registrarI18nPublico(raiz) {
  const mensajes = Object.freeze({
    cargando_convocatorias: "Cargando convocatorias…",
    consulta_no_disponible: "La consulta no está disponible.",
    fuente_no_configurada: "La fuente pública de Bolsa no está configurada para esta consulta.",
    ficha_no_disponible: "Ficha no disponible",
    seleccione_convocatoria: "Seleccione una convocatoria",
    todos_tipos: "Todos los tipos",
    todas_categorias: "Todas con procesos",
    todos_estados: "Todos los estados",
    todas_areas: "Todas las áreas",
    sin_plazos: "No hay plazos públicos asociados.",
    sin_requisitos: "No hay requisitos públicos asociados.",
    sin_documentos: "No hay documentos públicos asociados.",
    sin_ayuda: "No hay respuestas de ayuda asociadas.",
    consultar_ficha: "Consultar ficha pública",
    error_bolsas: "Error al consultar bolsas",
    error_lista: "Error al consultar la lista",
    documento_formato: "El documento debe tener formato ***1234** (3 asteriscos, 4 dígitos y 2 asteriscos).",
    consultar_lista: "Consultar lista",
    disponible: "Disponible",
    ocupado: "Ocupado / Nombrado",
    no_disponible: "No disponible (pausa)",
    excluido: "Excluido",
    renuncia_pendiente: "Renuncia en trámite",
    dato_no_disponible: "No consta",
    grupos: "Grupos",
    tipo_lista: "Tipo de lista",
    vigente_desde: "Vigente desde",
    total_aspirantes: "Aspirantes",
    error_bolsas_denegado: "La consulta pública de bolsas no está disponible para este acceso.",
    error_lista_denegado: "La consulta pública de esta lista no está disponible para este acceso.",
    error_lista_no_encontrada: "La bolsa solicitada no está disponible para consulta pública.",
    ayuda_privacidad_listas: "Ayuda y privacidad de la consulta",
    ayuda_documento_lista: "Formato de búsqueda",
    area_no_indicada: "Área no indicada",
    plazo_desde: "Desde",
    plazo_hasta: "hasta",
    opcion_con_numero: "{etiqueta} ({total})",
    categoria_sin_procesos: "{categoria} (sin procesos publicados)",
    seleccion_sin_resultados: "Selección sin resultados: {seleccionado}",
    requisito_uno: "{total} requisito",
    requisito_otros: "{total} requisitos",
    documento_uno: "{total} documento",
    documento_otros: "{total} documentos",
    ayuda_uno: "{total} ayuda",
    ayuda_otros: "{total} ayudas",
    convocatoria_encontrada_uno: "{total} convocatoria encontrada",
    convocatoria_encontrada_otros: "{total} convocatorias encontradas",
    proceso_publicado_uno: "{total} proceso publicado",
    proceso_publicado_otros: "{total} procesos publicados",
    plazo_abierto_uno: "{total} plazo abierto",
    plazo_abierto_otros: "{total} plazos abiertos",
    publicada_el: "Publicada el",
    fuente_actualizada: "Fuente {revision} · actualizada {fecha}",
    pagina_de: "Página {pagina} de {paginas}",
    obligatorio: "Obligatorio",
    no_obligatorio: "No obligatorio",
    abrir_documento: "Abrir {formato}: {titulo}",
    bases_publicadas_el: "Bases publicadas el",
    version_huella: "Versión {version} · huella SHA-256 {huella}",
    ver_procesos: "Ver procesos",
    ver_procesos_de: "Ver procesos de {categoria}",
    sin_convocatorias_publicadas: "Sin convocatorias publicadas actualmente",
    categorias_mostradas: "{mostradas} de {total} categorías mostradas",
    catalogo_resumen: "Catálogo {referencia} · versión {version} · {total} categorías · huella {huella}…",
    catalogo_resumen_aria: "Catálogo {referencia}, versión {version}, {total} categorías, huella SHA-256 {huella}",
    huella_sha256: "SHA-256 {huella}",
    cargando_catalogo: "Cargando el catálogo profesional…",
    directorio_no_disponible: "El directorio no está disponible.",
  });
  const formateadorNumero = new Intl.NumberFormat("es-ES");
  function t(clave, variables = {}) {
    const plantilla = mensajes[clave];
    if (typeof plantilla !== "string") return clave;
    return plantilla.replace(/\{([a-z_]+)\}/g, (_coincidencia, nombre) => String(variables[nombre] ?? ""));
  }
  function numero(valor) {
    return formateadorNumero.format(valor);
  }
  function plural(clave, total) {
    return t(`${clave}_${total === 1 ? "uno" : "otros"}`, { total: numero(total) });
  }
  raiz.VECBolsaI18n = Object.freeze({ t, numero, plural, mensajes });
}(globalThis));
