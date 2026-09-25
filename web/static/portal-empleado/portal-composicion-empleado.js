/** Montaje de vistas del empleado, sin inferir permisos ni componer escrituras. */
export function componerCronosVisible(recursos, contextoActor, entorno) {
  if (typeof recursos.recorridos?.montarVistaRecorridosCronos !== "function") return undefined;
  return Object.freeze({ montar: recursos.recorridos.montarVistaRecorridosCronos });
}

/**
 * Cronos interno de la persona empleada: saldo, fichaje remoto, movimientos
 * del día y calendario con olvidos en «Jornada»; permisos propios aparte.
 * Falta cualquier pieza → undefined (el módulo no se ofrece). Cada vista
 * consulta su API y muestra su propio estado: un 404 de una capacidad
 * desactivada no afecta a las demás.
 */
export function componerCronosInterno(recursos, entorno) {
  const { saldo, remoto, movimientos, movimientosPropios, permisosPropios,
    clienteSaldo, clienteRemoto, clienteSolicitudes, i18n } = recursos ?? {};
  if (typeof saldo?.montarVistaSaldoCronos !== "function"
    || typeof remoto?.montarVistaRemotoCronos !== "function"
    || typeof movimientos?.montarVistaMovimientosCronos !== "function"
    || typeof movimientosPropios?.montarMovimientosPropiosCronos !== "function"
    || typeof permisosPropios?.montarPermisosPropiosCronos !== "function"
    || typeof clienteSaldo?.crearClienteSaldoCronosHTTP !== "function"
    || typeof clienteRemoto?.crearClienteRemotoCronosHTTP !== "function"
    || typeof clienteSolicitudes?.crearClienteSolicitudesCronosHTTP !== "function"
    || typeof i18n?.crearTraductorCronos !== "function") return undefined;
  const transporte = typeof entorno?.fetch === "function" ? { fetchImpl: entorno.fetch.bind(entorno) } : {};
  const cliente = Object.freeze({
    saldo: clienteSaldo.crearClienteSaldoCronosHTTP(transporte),
    remoto: clienteRemoto.crearClienteRemotoCronosHTTP(transporte),
    solicitudes: clienteSolicitudes.crearClienteSolicitudesCronosHTTP(transporte),
  });
  return Object.freeze({
    traducir: i18n.crearTraductorCronos(),
    montar({ raiz, anunciar = () => {}, registrarDesmontar } = {}) {
      const partes = []; const desmontes = [];
      let propios = null;
      const colgar = (nombre, montarParte) => {
        const nodo = raiz.ownerDocument.createElement("div");
        nodo.className = "cronos-personal-parte"; nodo.dataset.cronosParte = nombre;
        raiz.append(nodo); partes.push(nodo);
        try {
          const parte = montarParte(nodo);
          if (typeof parte?.desmontar === "function") desmontes.push(parte.desmontar);
          return parte;
        } catch { nodo.remove?.(); return null; }
      };
      colgar("saldo", (nodo) => saldo.montarVistaSaldoCronos({ raiz: nodo, cliente: cliente.saldo, anunciar }));
      colgar("remoto", (nodo) => remoto.montarVistaRemotoCronos({ raiz: nodo, cliente: cliente.remoto }));
      colgar("movimientos", (nodo) => movimientos.montarVistaMovimientosCronos({ raiz: nodo, cliente: cliente.saldo, anunciar,
        abrirCorreccion: () => propios?.abrirOlvido?.() }));
      propios = colgar("calendario", (nodo) => movimientosPropios.montarMovimientosPropiosCronos({ raiz: nodo, cliente: cliente.solicitudes, anunciar }));
      let activo = true;
      const desmontar = () => {
        if (!activo) return;
        activo = false;
        for (const retirar of desmontes.splice(0).reverse()) { try { retirar(); } catch { /* cada parte se retira sola */ } }
        for (const nodo of partes.splice(0)) nodo.remove?.();
      };
      registrarDesmontar?.(desmontar);
      return Object.freeze({ desmontar });
    },
    montarPermisos({ raiz, anunciar = () => {}, registrarDesmontar } = {}) {
      return permisosPropios.montarPermisosPropiosCronos({ raiz, cliente: cliente.solicitudes, anunciar, registrarDesmontar });
    },
  });
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
