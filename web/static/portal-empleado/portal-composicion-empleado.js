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
