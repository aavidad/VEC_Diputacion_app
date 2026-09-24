/** Montaje de vistas del empleado, sin inferir permisos ni componer escrituras. */
export function componerCronosVisible(recursos, contextoActor, entorno) {
  if (typeof recursos.recorridos?.montarVistaRecorridosCronos !== "function") return undefined;
  return Object.freeze({ montar: recursos.recorridos.montarVistaRecorridosCronos });
}

/** El portal interno inyecta clientes HTTP; cada operación se autoriza en servidor. */
export function componerDietasInternas(recursos, entorno) {
  if (!recursos?.contrato || typeof recursos.contrato !== "object"
    || typeof recursos?.clienteBorradores?.crearClienteBorradoresDietasHTTP !== "function"
    || typeof recursos?.clienteAsignacion?.crearClienteAsignacionDietasHTTP !== "function"
    || typeof recursos?.calculador?.crearCalculadorRutasDietasHTTP !== "function"
    || typeof recursos?.mapa?.crearVisorRutaDietas !== "function"
    || typeof recursos?.recorridos?.montarVistaRecorridosDietas !== "function"
    || typeof entorno?.fetch !== "function") return undefined;
  const fetchImpl = entorno.fetch.bind(entorno);
  const clienteBorradores = recursos.clienteBorradores.crearClienteBorradoresDietasHTTP({ fetchImpl });
  const clienteAsignacion = recursos.clienteAsignacion.crearClienteAsignacionDietasHTTP({ fetchImpl });
  // Catálogo y ruta por carretera los autoriza el servidor en cada petición;
  // las teselas son las propias del mismo origen, sin proveedor externo.
  const calculadorRuta = recursos.calculador.crearCalculadorRutasDietasHTTP({ fetchImpl });
  const visorRuta = recursos.mapa.crearVisorRutaDietas({ entorno, permitirTeselas: true });
  return Object.freeze({
    clienteBorradores, clienteAsignacion,
    montar: async ({ raiz, anunciar, registrarDesmontar }) => {
      // Personal acredita las relaciones antes de montar. Si el portal
      // abandona la vista mientras llega la respuesta, no se monta nada.
      const cancelacion = new AbortController();
      let vigente = true;
      registrarDesmontar?.(() => { vigente = false; cancelacion.abort(); });
      let relaciones = { relacionesAutorizadas: [], fechaReferenciaPersonal: undefined,
        estadoRelaciones: "no_disponible", motivoRelaciones: undefined };
      try {
        const respuesta = await clienteAsignacion.obtenerRelaciones({ signal: cancelacion.signal });
        relaciones = { relacionesAutorizadas: respuesta.relaciones_autorizadas,
          fechaReferenciaPersonal: respuesta.fecha_referencia, estadoRelaciones: "disponible", motivoRelaciones: undefined };
      } catch (error) {
        if (["empleado_no_disponible", "empleado_ambiguo"].includes(error?.codigo))
          relaciones = { ...relaciones, motivoRelaciones: error.codigo };
      }
      if (!vigente) return Object.freeze({ desmontar() {} });
      return recursos.recorridos.montarVistaRecorridosDietas(raiz, {
        clienteBorradores, clienteAsignacion, calculadorRuta, visorRuta, ...relaciones,
        anunciar, registrarDesmontar,
      });
    },
  });
}

export function componerPersonalVisible(recursos, entorno, { catalogosPublicos = true } = {}) {
  const catalogos = [
    [recursos.clienteCategorias?.crearClienteHTTPCategoriasPersonal, recursos.vistaCategorias?.montarModuloPersonal],
    ...(catalogosPublicos ? [
      [recursos.clienteRPT?.crearClienteHTTPRPTPublica, recursos.vistaRPT?.montarModuloRPTPublica],
      [recursos.clienteEstructura?.crearClienteHTTPEstructuraOrganizativaPublica, recursos.vistaEstructura?.montarModuloEstructuraOrganizativaPublica],
    ] : []),
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
        raiz, anunciar, registrarDesmontar, montarCatalogos, fuentes: {},
        navegarModulo: (modulo) => {
          if (["dietas", "cronos"].includes(modulo) && entorno.location) entorno.location.hash = `#${modulo}`;
        },
      })
      : montarCatalogos,
  });
}
