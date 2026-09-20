/** Montaje de vistas del empleado, sin inferir permisos ni componer escrituras. */
export function componerCronosVisible(recursos, contextoActor, entorno) {
  if (typeof recursos.recorridos?.montarVistaRecorridosCronos !== "function") return undefined;
  return Object.freeze({ montar: recursos.recorridos.montarVistaRecorridosCronos });
}

export function componerDietasVisible(recursos, contextoActor, capacidades, entorno) {
  const calculador = recursos.calculador.crearCalculadorRutasDietasPresentacionOSRM({
    contextoActor, capacidades,
    fetchImpl: typeof entorno.fetch === "function" ? entorno.fetch.bind(entorno) : undefined,
  });
  const visorRuta = recursos.mapa.crearVisorRutaDietas({ entorno, permitirTeselas: true });
  return Object.freeze({
    calculador, visorRuta,
    montar: recursos.recorridos?.montarVistaRecorridosDietas
      ? ({ raiz, anunciar, registrarDesmontar }) => recursos.recorridos.montarVistaRecorridosDietas(raiz, {
        anunciar, registrarDesmontar,
        montarItinerario: (hueco) => recursos.vista.montarVistaItinerarioDietas({
          raiz: hueco, calculador, visorRuta, anunciar,
        }),
      })
      : recursos.vista.montarVistaItinerarioDietas,
  });
}

/**
 * El portal interno puede ofrecer los borradores propios sin fabricar un
 * ContextoActor en el navegador. El cálculo de ruta HTTP conserva su contrato
 * más estricto: se añadirá sólo desde un proveedor explícito de identidad y
 * capacidades, nunca desde el catálogo o una entrada de menú.
 */
export function componerDietasInternas(recursos, entorno) {
  if (!recursos?.contrato || typeof recursos.contrato !== "object"
    || typeof recursos?.clienteBorradores?.crearClienteBorradoresDietasHTTP !== "function"
    || typeof recursos?.recorridos?.montarVistaRecorridosDietas !== "function"
    || typeof recursos?.vista?.montarVistaItinerarioPendienteDietas !== "function") return undefined;
  const fetchImpl = typeof entorno?.fetch === "function" ? entorno.fetch.bind(entorno) : undefined;
  const clienteBorradores = recursos.clienteBorradores.crearClienteBorradoresDietasHTTP({ fetchImpl });
  // Solo la raíz de identidad puede inyectar este par ya autorizado. El
  // navegador, el catálogo y el menú no construyen ContextoActor, capacidad ni
  // cliente HTTP para rutas. En su ausencia se conserva el área visible pero
  // no hay cálculo ni geometría.
  const itinerarioAutorizado = entorno?.dietasItinerarioAutorizado;
  const puedeCalcular = itinerarioAutorizado
    && typeof itinerarioAutorizado === "object"
    && typeof itinerarioAutorizado.calculador?.obtenerCatalogo === "function"
    && typeof itinerarioAutorizado.calculador?.calcular === "function"
    && typeof itinerarioAutorizado.visorRuta?.montar === "function";
  return Object.freeze({
    clienteBorradores,
    montar: ({ raiz, anunciar, registrarDesmontar }) => recursos.recorridos.montarVistaRecorridosDietas(raiz, {
      clienteBorradores,
      anunciar,
      registrarDesmontar,
      // El área cartográfica sí queda visible, pero no recibe catálogo,
      // identidad ni calculador. Solo una composición autorizada puede
      // sustituir este estado cerrado por el visor y la ruta reales.
      montarItinerario: (hueco) => (puedeCalcular
        ? recursos.vista.montarVistaItinerarioDietas({
          raiz: hueco,
          calculador: itinerarioAutorizado.calculador,
          visorRuta: itinerarioAutorizado.visorRuta,
          anunciar,
        })
        : recursos.vista.montarVistaItinerarioPendienteDietas({ raiz: hueco })),
    }),
  });
}

export function componerPersonalVisible(recursos, entorno) {
  const catalogos = [
    [recursos.clienteCategorias?.crearClienteHTTPCategoriasPersonal, recursos.vistaCategorias?.montarModuloPersonal],
    [recursos.clienteRPT?.crearClienteHTTPRPTPublica, recursos.vistaRPT?.montarModuloRPTPublica],
    [recursos.clienteEstructura?.crearClienteHTTPEstructuraOrganizativaPublica, recursos.vistaEstructura?.montarModuloEstructuraOrganizativaPublica],
  ];
  if (catalogos.some(([cliente, vista]) => typeof cliente !== "function" || typeof vista !== "function")) return undefined;
  const montarCatalogos = async ({ raiz, anunciar, registrarDesmontar }) => {
    const fetchImpl = typeof entorno.fetch === "function" ? entorno.fetch.bind(entorno) : undefined;
    const limpiezas = new Set();
    const retiradas = new Set();
    let activa = true;
    const retirar = (limpiar) => {
      if (retiradas.has(limpiar)) return;
      retiradas.add(limpiar);
      limpiar();
    };
    const desmontar = () => {
      if (!activa) return;
      activa = false;
      [...limpiezas].reverse().forEach(retirar);
      limpiezas.clear();
    };
    const registrar = (limpiar) => {
      if (typeof limpiar !== "function") throw new TypeError("limpieza de Personal no válida");
      if (!activa) { retirar(limpiar); return; }
      limpiezas.add(limpiar);
    };
    registrarDesmontar?.(desmontar);
    try {
      await Promise.all(catalogos.map(async ([crearCliente, montar]) => {
        const vista = await montar({ raiz, anunciar, registrarDesmontar: registrar, cliente: crearCliente({ fetchImpl }) });
        // Los consumidores actuales registran antes de esperar. El Set evita
        // duplicar esa misma limpieza al resolver; admite también nuevos consumidores.
        registrar(vista.desmontar);
      }));
    } catch (causa) {
      desmontar();
      throw causa;
    }
    return Object.freeze({ desmontar });
  };
  return Object.freeze({
    montar: recursos.ficha?.montarVistaFichaIntegralPersonal
      ? ({ raiz, anunciar, registrarDesmontar }) => recursos.ficha.montarVistaFichaIntegralPersonal({
        raiz, anunciar, registrarDesmontar, montarCatalogos,
        navegarModulo: (modulo) => {
          if (["dietas", "cronos"].includes(modulo) && entorno.location) entorno.location.hash = `#${modulo}`;
        },
      })
      : montarCatalogos,
  });
}
