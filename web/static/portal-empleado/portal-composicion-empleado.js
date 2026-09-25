/** Montaje de vistas del empleado, sin inferir permisos ni componer escrituras. */
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
  const traducir = i18n.crearTraductorCronos();
  return Object.freeze({
    traducir,
    montar({ raiz, anunciar = () => {}, registrarDesmontar } = {}) {
      const t = traducir; const documento = raiz.ownerDocument;
      const elemento = (etiqueta, clase, texto) => {
        const nodo = documento.createElement(etiqueta);
        if (clase) nodo.className = clase;
        if (texto !== undefined) nodo.textContent = texto;
        return nodo;
      };
      // Un único encabezado de página; cada parte se monta incrustada, con
      // encabezado de tarjeta y sin sobrelínea propia.
      const cabecera = elemento("header", "cronos-encabezado");
      const titulo = elemento("div");
      titulo.append(elemento("p", "sobrelinea", t("titulo")), elemento("h2", undefined, t("jornada_titulo")));
      cabecera.append(titulo); raiz.append(cabecera);
      const nombres = ["saldo", "remoto", "movimientos", "calendario"];
      const nodos = new Map(nombres.map((nombre) => {
        const nodo = elemento("div", "cronos-personal-parte"); nodo.dataset.cronosParte = nombre;
        raiz.append(nodo); return [nombre, nodo];
      }));
      const desmontes = new Map();
      // Una parte que no se puede montar deja su aviso en su sitio (sin
      // detalles internos) y no arrastra a las demás.
      const colgar = (nombre, montarParte) => {
        const nodo = nodos.get(nombre);
        try {
          const parte = montarParte(nodo);
          if (typeof parte?.desmontar === "function") desmontes.set(nombre, parte.desmontar);
          return parte;
        } catch {
          const aviso = elemento("p", "cronos-acceso-denegado", t("jornada_parte_error"));
          aviso.setAttribute("role", "alert");
          nodo.replaceChildren(aviso); nodo.dataset.cronosParteEstado = "error";
          return null;
        }
      };
      colgar("saldo", (nodo) => saldo.montarVistaSaldoCronos({ raiz: nodo, cliente: cliente.saldo, anunciar, incrustada: true }));
      colgar("remoto", (nodo) => remoto.montarVistaRemotoCronos({ raiz: nodo, cliente: cliente.remoto }));
      // El calendario se monta antes que los movimientos del día: «olvido de
      // marcaje» solo se ofrece si hay un formulario de olvido al que llevar.
      const propios = colgar("calendario", (nodo) => movimientosPropios.montarMovimientosPropiosCronos({
        raiz: nodo, cliente: cliente.solicitudes, anunciar, incrustada: true }));
      const abrirOlvido = typeof propios?.abrirOlvido === "function" ? propios.abrirOlvido : undefined;
      colgar("movimientos", (nodo) => movimientos.montarVistaMovimientosCronos({ raiz: nodo, cliente: cliente.saldo, anunciar,
        incrustada: true, ...(abrirOlvido ? { abrirCorreccion: () => abrirOlvido() } : {}) }));
      let activo = true;
      const desmontar = () => {
        if (!activo) return;
        activo = false;
        for (const nombre of [...nombres].reverse()) {
          try { desmontes.get(nombre)?.(); } catch { /* cada parte se retira sola */ }
        }
        desmontes.clear();
        for (const nodo of [cabecera, ...nodos.values()]) nodo.remove?.();
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
  // El circuito de revisión se autoriza en servidor por acción; sus bandejas
  // dependen de la competencia que acredite la fuente gobernada.
  const clienteCircuito = typeof recursos?.clienteCircuito?.crearClienteCircuitoDietasHTTP === "function"
    ? recursos.clienteCircuito.crearClienteCircuitoDietasHTTP({ fetchImpl }) : undefined;
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
        clienteBorradores, clienteAsignacion, clienteCircuito, calculadorRuta, visorRuta, ...relaciones,
        anunciar, registrarDesmontar,
      });
    },
  });
}

export function componerPersonalVisible(recursos, entorno, {
  catalogosPublicos = true, ocultarSinFuente = false, destinosDisponibles = () => ({}),
} = {}) {
  // `catalogosPublicos` admite todos (true), ninguno (false) o la lista de los
  // que el servidor ha servido de verdad («rpt», «estructura»).
  const publicos = catalogosPublicos === true ? ["rpt", "estructura"]
    : Array.isArray(catalogosPublicos) ? catalogosPublicos : [];
  if (publicos.some((clave) => !["rpt", "estructura"].includes(clave))
    || typeof ocultarSinFuente !== "boolean" || typeof destinosDisponibles !== "function") return undefined;
  const catalogos = [
    [recursos.clienteCategorias?.crearClienteHTTPCategoriasPersonal, recursos.vistaCategorias?.montarModuloPersonal],
    ...(publicos.includes("rpt") ? [[recursos.clienteRPT?.crearClienteHTTPRPTPublica, recursos.vistaRPT?.montarModuloRPTPublica]] : []),
    ...(publicos.includes("estructura") ? [[recursos.clienteEstructura?.crearClienteHTTPEstructuraOrganizativaPublica,
      recursos.vistaEstructura?.montarModuloEstructuraOrganizativaPublica]] : []),
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
  const montarFicha = ({ raiz, anunciar, registrarDesmontar }, fuentes = {}) => recursos.ficha.montarVistaFichaIntegralPersonal({
    raiz, anunciar, registrarDesmontar, montarCatalogos, fuentes, ocultarSinFuente,
    destinosDisponibles: destinosDisponibles(),
    navegarModulo: (modulo) => {
      if (["dietas", "cronos"].includes(modulo) && entorno.location) entorno.location.hash = `#${modulo}`;
    },
  });
  // Ficha propia servida por Personal: una consulta al entrar decide qué
  // apartados tienen fuente para esta persona; sin ella no se ofrecen.
  const crearFuentes = recursos.clienteFichaPropia?.crearFuentesFichaPropia;
  const conFichaPropia = typeof crearFuentes === "function" && typeof entorno.fetch === "function";
  return Object.freeze({
    montar: recursos.ficha?.montarVistaFichaIntegralPersonal
      ? (conFichaPropia
        ? async (entrada) => {
          const fuentes = await crearFuentes({ fetchImpl: entorno.fetch.bind(entorno) }).preparar();
          return montarFicha(entrada, fuentes);
        }
        : (entrada) => montarFicha(entrada))
      : montarCatalogos,
  });
}

/**
 * Registro de Personal para RRHH: lista de empleados del organismo, ficha,
 * vacantes y catálogos. El organismo y el permiso los fija el servidor en cada
 * petición (V3); aquí solo se crean los clientes del mismo origen. Falta
 * cualquier pieza → undefined (la vista no se ofrece).
 */
export function componerRegistroPersonal(recursos, entorno) {
  const { registro, clienteRegistro, clienteCatalogosRegistro, i18n } = recursos ?? {};
  if (typeof registro?.montarRegistroB2 !== "function"
    || typeof clienteRegistro?.crearClienteRegistroB2 !== "function"
    || typeof clienteCatalogosRegistro?.crearClienteCatalogosRegistroB2 !== "function"
    || typeof i18n?.crearTraductorPersonal !== "function"
    || typeof entorno?.fetch !== "function") return undefined;
  const fetchImpl = entorno.fetch.bind(entorno);
  const cliente = clienteRegistro.crearClienteRegistroB2({ fetchImpl });
  const clienteCatalogos = clienteCatalogosRegistro.crearClienteCatalogosRegistroB2({ fetchImpl });
  return Object.freeze({
    traducir: i18n.crearTraductorPersonal(),
    // Consulta mínima autorizada: acredita que esta superficie sirve el
    // registro y que V3 concede su lectura al actor antes de ofrecerlo.
    sondear: ({ signal } = {}) => clienteCatalogos.listar({ tipo: "regimen", limite: 1, signal }),
    montar: ({ raiz, anunciar = () => {}, registrarDesmontar } = {}) => registro.montarRegistroB2({
      raiz, cliente, clienteCatalogos, anunciar, registrarDesmontar,
    }),
  });
}
