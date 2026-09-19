/** Catálogo común de las superficies públicas de Bolsa. Sin estado persistente. */
(function registrarI18nPublico(raiz) {
  const mensajes = Object.freeze({
    demostracion_aviso: "Datos públicos reales de referencia; plazos y actuaciones rotulados DEMO son sintéticos y carecen de validez administrativa.",
    volver_presentacion: "Volver al selector de recorridos de la presentación",
    inicio_bolsa: "Inicio de Bolsa y procesos selectivos",
    cargando_convocatorias: "Cargando convocatorias…",
    consulta_no_disponible: "La consulta no está disponible.",
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
  });
  function t(clave, variables = {}) {
    const plantilla = mensajes[clave];
    if (typeof plantilla !== "string") return clave;
    return plantilla.replace(/\{([a-z_]+)\}/g, (_coincidencia, nombre) => String(variables[nombre] ?? ""));
  }
  raiz.VECBolsaI18n = Object.freeze({ t, mensajes });
}(globalThis));
