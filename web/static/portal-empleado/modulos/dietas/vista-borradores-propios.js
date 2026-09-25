import { crearTraductorDietas, MENSAJES_DIETAS_ES } from "./i18n.js?v=20260925-tanda2-v1";
import { crearTraductorBorradoresDietas } from "./i18n-borradores.js?v=20260925-tanda-v1";
import { montarVistaMapaComisionDietas } from "./vista-mapa-comision.js?v=20260925-tanda2-v1";
import { montarVistaRectificacionDietas } from "./vista-rectificacion-dietas.js?v=20260925-tanda-v1";

const MAXIMO_LOCALIDADES = 12;

function nodo(documento, etiqueta, texto = "") {
  const resultado = documento.createElement(etiqueta);
  if (texto !== "") resultado.textContent = texto;
  return resultado;
}
function montada(raiz, contenedor) {
  return (
    raiz.querySelector?.("[data-dietas-borradores-propios]") === contenedor
  );
}
function retirar(raiz, contenedor) {
  if (!montada(raiz, contenedor)) return;
  if (typeof contenedor.remove === "function") contenedor.remove();
  else raiz.removeChild?.(contenedor);
}
function claveContenido(solicitud) {
  return JSON.stringify([
    solicitud.fecha_inicio,
    solicitud.fecha_fin,
    solicitud.hora_inicio,
    solicitud.hora_fin,
    solicitud.motivo,
    solicitud.codigos_ruta,
    solicitud.relacion_ref,
  ]);
}
// Estados que Dietas y su circuito devuelven, con su rótulo y su tono.
const ESTADOS_COMISION = Object.freeze({
  borrador: ["comision_estado_borrador", "info"],
  eliminado: ["comision_estado_eliminado", "aviso"],
  enviado_pendiente_revision: ["comision_estado_enviado", "exito"],
  devuelta: ["circuito_estado_devuelta", "aviso"],
  pendiente_autorizacion: ["circuito_estado_pendiente_autorizacion", "exito"],
  pendiente_liquidacion: ["circuito_estado_pendiente_liquidacion", "exito"],
  pendiente_fiscalizacion: ["circuito_estado_pendiente_fiscalizacion", "exito"],
  fiscalizada: ["circuito_estado_fiscalizada", "exito"],
});
const MOTIVOS_RELACIONES = new Set(["empleado_no_disponible", "empleado_ambiguo"]);
function euros(centimos) { return new Intl.NumberFormat("es-ES", { style: "currency", currency: "EUR" }).format(centimos / 100); }
function rutaLegible(codigos, traducir, nombres = new Map()) {
  return codigos.length
    ? codigos.map((codigo) => nombres.get(codigo) || codigo).join(" → ")
    : traducir("borradores_propios_ruta_no_declarada");
}
function referenciasRelacionAutorizadas(valores) {
  if (!Array.isArray(valores))
    throw new TypeError("relaciones autorizadas de Dietas no válidas");
  const referencias = valores.map((entrada) => {
    const valor = typeof entrada === "string" ? entrada : entrada?.relacion_ref;
    if (
      typeof valor !== "string" ||
      !/^rel_[A-Za-z0-9_-]{22,128}$/u.test(valor)
    )
      throw new TypeError("relaciones autorizadas de Dietas no válidas");
    const unidad = typeof entrada === "string" ? undefined : entrada?.unidad_ref;
    if (unidad !== undefined && (typeof unidad !== "string" || unidad.length < 1 || unidad.length > 256 ||
        unidad.trim() !== unidad || /[\x00-\x1f\x7f]/u.test(unidad)))
      throw new TypeError("unidad autorizada de Dietas no válida");
    return Object.freeze({ relacion_ref: valor, unidad_ref: unidad });
  });
  if (new Set(referencias.map((entrada) => entrada.relacion_ref)).size !== referencias.length)
    throw new TypeError("relaciones autorizadas de Dietas no válidas");
  return Object.freeze(referencias);
}
function fechaLegible(valor, conHora = false) {
  const fecha = new Date(conHora ? valor : `${valor}T00:00:00Z`);
  return Number.isFinite(fecha.getTime())
    ? new Intl.DateTimeFormat(
        "es-ES",
        conHora
          ? {
              dateStyle: "medium",
              timeStyle: "medium",
              timeZone: "Europe/Madrid",
            }
          : { dateStyle: "medium", timeZone: "UTC" },
      ).format(fecha)
    : "—";
}
function errorClave(error, accion = "consulta") {
  if (error?.codigo === "autenticacion_requerida")
    return "borradores_propios_autenticacion_requerida";
  if (error?.codigo === "acceso_denegado")
    return accion === "crear"
      ? "borradores_propios_creacion_denegada"
      : "borradores_propios_consulta_denegada";
  if (
    [
      "relacion_ambigua",
      "relacion_no_disponible",
      "relacion_no_valida",
    ].includes(error?.codigo)
  )
    return "borradores_propios_error_relacion";
  if (error?.codigo === "conflicto_idempotencia")
    return "borradores_propios_error_conflicto";
  return error?.resultadoIndeterminado
    ? "borradores_propios_error_incierto"
    : "borradores_propios_error";
}

/** Monta la única superficie de creación y consulta de borradores propios de Dietas. */
export function montarVistaBorradoresPropios(
  contenedor,
  {
    cliente,
    clienteAsignacion,
    clienteRectificacion,
    calculadorRuta,
    visorRuta,
    catalogoProyectado = [],
    traducir = crearTraductorDietas(MENSAJES_DIETAS_ES),
    anunciar = () => {},
    confirmarOperacion = (texto) => globalThis.confirm?.(texto) === true,
    generarClaveIdempotencia = () => globalThis.crypto?.randomUUID?.(),
    registrarDesmontar,
    formularioInicialmenteVisible = true,
    estadoRelaciones = "disponible",
    // Código cerrado con que Personal denegó las relaciones (sin empleado
    // canónico o con varios); solo cambia el mensaje, no reabre acciones.
    motivoRelaciones,
    fechaReferenciaPersonal,
    // Sólo la composición que haya consultado una fuente autorizada puede
    // aportar estas referencias opacas. Esta vista no deduce ni fabrica una.
    relacionesAutorizadas = [],
  } = {},
) {
  if (
    !contenedor?.append ||
    !contenedor?.querySelector ||
    !contenedor.ownerDocument ||
    (cliente !== undefined &&
      (typeof cliente?.crear !== "function" ||
        typeof cliente?.listar !== "function" ||
        typeof cliente?.obtener !== "function")) ||
    (clienteAsignacion !== undefined && typeof clienteAsignacion?.obtener !== "function") ||
    (clienteRectificacion !== undefined &&
      (typeof clienteRectificacion?.consultar !== "function" || typeof clienteRectificacion?.solicitar !== "function")) ||
    !Array.isArray(catalogoProyectado) || catalogoProyectado.length > 500 ||
    catalogoProyectado.some((punto) => typeof punto?.codigo !== "string" ||
      !/^[A-Za-z0-9][A-Za-z0-9._:-]{1,63}$/u.test(punto.codigo) || typeof punto?.nombre !== "string" || !punto.nombre) ||
    ((calculadorRuta !== undefined || visorRuta !== undefined) &&
      (typeof calculadorRuta?.obtenerCatalogo !== "function" || typeof calculadorRuta?.calcular !== "function" ||
       typeof visorRuta?.montar !== "function")) ||
    typeof traducir !== "function" ||
    typeof anunciar !== "function" ||
    typeof confirmarOperacion !== "function" ||
    typeof generarClaveIdempotencia !== "function" ||
    typeof formularioInicialmenteVisible !== "boolean" ||
    !["disponible", "no_disponible"].includes(estadoRelaciones) ||
    (motivoRelaciones !== undefined && (estadoRelaciones !== "no_disponible" ||
      !MOTIVOS_RELACIONES.has(motivoRelaciones))) ||
    (fechaReferenciaPersonal !== undefined && !/^\d{4}-\d{2}-\d{2}$/u.test(fechaReferenciaPersonal)) ||
    (registrarDesmontar !== undefined &&
      typeof registrarDesmontar !== "function")
  ) {
    throw new TypeError("vista de borradores propios de Dietas no disponible");
  }
  const documento = contenedor.ownerDocument;
  const tBorradores = crearTraductorBorradoresDietas(traducir);
  const enfocar = (elemento) => elemento?.focus?.(
    (documento.defaultView?.innerWidth ?? 1440) >= 1024
      ? { preventScroll: true }
      : undefined,
  );
  const focoSigueEnConsulta = (origen) => {
    const actual = documento.activeElement;
    return !actual || actual === origen || actual === documento.body;
  };
  const entradasRelacion = referenciasRelacionAutorizadas(relacionesAutorizadas);
  const relaciones = Object.freeze(entradasRelacion.map((entrada) => entrada.relacion_ref));
  const unidadesAutorizadas = new Map(entradasRelacion.map((entrada) => [entrada.relacion_ref, entrada.unidad_ref]));
  let puntosRuta = calculadorRuta ? [] : catalogoProyectado;
  let nombresRuta = new Map(puntosRuta.map((punto) => [punto.codigo, punto.nombre]));
  const raiz = nodo(documento, "section");
  raiz.className = "modulo-dietas dietas-borradores-propios";
  raiz.dataset.dietasBorradoresPropios = "";
  contenedor.append(raiz);
  const conectada = cliente !== undefined;
  let activa = true;
  let controlador = null;
  let formularioPersistente = null;
  let avisoPersistente = null;
  let listaPersistente = null;
  let fichaPersistente = null;
  // La clave de toda intención enviada pertenece a su contenido exacto. Otra
  // comisión puede prepararse sin perder la recuperación de la primera.
  const operaciones = new Map();
  let ultimoAlta = null;
  let formularioVisible = formularioInicialmenteVisible;
  let nivelDetalle = "medio";
  let edicion = null;
  let intentoEdicion = null;
  const intentosAccion = new Map();
  let asignacion = null;
  let asignacionFecha = null;
  let asignacionMensaje = "comision_asignacion_pendiente";
  let controladorAsignacion = null;
  let vistaRectificacion = null;
  let rectificacionClave = null;
  let rectificacionContenedor = null;
  let reciboRectificacionConfirmado = null;
  let mapaVista = null;
  let montajeMapa = null;
  let controladorCatalogo = null;
  let calculoCabeceraFirma = null;
  const rutasCalculadas = new Map();
  const metadatosOtros = new WeakMap();
  const asignacionVerificada = () => asignacion?.verificada === true &&
    asignacion?.relacion_ref === estado.detalle?.comision.relacion_ref &&
    asignacionFecha === fechaReferenciaPersonal;
  let cursores = [undefined];
  let indicePagina = 0;
  let siguienteCursor = undefined;
  let relacionSeleccionada =
    relaciones.length === 1 ? relaciones[0] : undefined;
  let estado = {
    carga: conectada,
    items: [],
    mensaje: conectada
      ? "borradores_propios_cargando"
      : "borradores_propios_pendiente_conexion",
    tono: "informacion",
    detalle: null,
    detalleOrigen: null,
    errorLista: false,
    errorListaClave: null,
  };
  const activaAhora = () => activa && montada(contenedor, raiz);
  const desmontar = () => {
    if (!activa) return;
    activa = false;
    controlador?.abort();
    controladorAsignacion?.abort();
    vistaRectificacion?.desmontar();
    mapaVista?.desmontar();
    controladorCatalogo?.abort();
    operaciones.clear();
    intentosAccion.clear();
    ultimoAlta = null;
    estado = { ...estado, items: [], detalle: null, detalleOrigen: null };
    raiz.removeEventListener("submit", enviar);
    raiz.removeEventListener("click", clic);
    raiz.removeEventListener("change", cambiarFormulario);
    raiz.removeEventListener("input", invalidarPreparacion);
    retirar(contenedor, raiz);
  };
  registrarDesmontar?.(desmontar);

  function mensaje(clave, tono = "informacion") {
    estado = { ...estado, mensaje: clave, tono };
    try {
      anunciar(tBorradores(clave), tono);
    } catch {}
  }
  async function consultarAsignacion(item) {
    controladorAsignacion?.abort();
    controladorAsignacion = null;
    asignacion = null;
    asignacionFecha = null;
    const unidad = unidadesAutorizadas.get(item?.comision?.relacion_ref);
    if (!item || !unidad || !clienteAsignacion || !fechaReferenciaPersonal || estadoRelaciones !== "disponible") {
      asignacionMensaje = "comision_asignacion_pendiente";
      if (activaAhora()) pintar();
      return;
    }
    asignacionMensaje = "borradores_propios_cargando";
    controladorAsignacion = new AbortController();
    const signal = controladorAsignacion.signal;
    if (activaAhora()) pintar();
    try {
      const resultado = await clienteAsignacion.obtener(item.comision.relacion_ref, unidad,
        fechaReferenciaPersonal, { signal });
      if (!activaAhora() || signal.aborted || estado.detalle?.comision.referencia !== item.comision.referencia) return;
      asignacion = resultado;
      asignacionFecha = fechaReferenciaPersonal;
      asignacionMensaje = "comision_asignacion_verificada";
    } catch (error) {
      if (!activaAhora() || signal.aborted) return;
      asignacionMensaje = error?.codigo === "no_encontrada" ? "comision_asignacion_sin_datos" : "comision_asignacion_error";
    } finally {
      if (controladorAsignacion?.signal === signal) controladorAsignacion = null;
      if (activaAhora()) pintar();
    }
  }
  function purgarLecturasDenegadas() {
    // Un 401/403 invalida toda la proyección obtenida por GET, incluida la
    // página que permitió abrir el detalle. Sólo sobrevive el último recibo
    // confirmado; las claves anteriores siguen ligadas a su contenido.
    for (const [contenido, operacion] of operaciones) {
      if (operacion.item && contenido !== ultimoAlta?.contenido)
        operaciones.set(contenido, { clave: operacion.clave, confirmada: true });
    }
    cursores = [undefined];
    indicePagina = 0;
    siguienteCursor = undefined;
    asignacion = null;
    controladorAsignacion?.abort();
    edicion = null;
    intentoEdicion = null;
    formularioVisible = false;
    calculoCabeceraFirma = null;
    rutasCalculadas.clear();
    estado = {
      ...estado,
      items: [],
      detalle: ultimoAlta?.item ?? null,
      detalleOrigen: ultimoAlta ? "post" : null,
    };
  }
  function formulario() {
    const form = nodo(documento, "form");
    form.dataset.dietasBorradorForm = "";
    form.className = "panel dietas-comision-formulario";
    const campo = (nombre, etiqueta, tipo = "text", obligatorio = true) => {
      const label = nodo(documento, "label", traducir(etiqueta));
      const input = nodo(documento, "input");
      input.name = nombre;
      input.type = tipo;
      input.required = obligatorio;
      label.append(input);
      return label;
    };
    const motivo = campo("motivo", "borradores_propios_motivo");
    const etiquetaNivel = nodo(documento, "label", traducir("recorridos_nivel_detalle"));
    const nivel = nodo(documento, "select");
    nivel.name = "nivel_detalle";
    [["bajo", "recorridos_nivel_bajo"], ["medio", "recorridos_nivel_medio"], ["alto", "recorridos_nivel_alto"]].forEach(([valor, clave]) => {
      const opcion = nodo(documento, "option", traducir(clave)); opcion.value = valor; nivel.append(opcion);
    });
    nivel.value = nivelDetalle;
    etiquetaNivel.append(nivel);
    const ayudaNivel = nodo(documento, "details");
    ayudaNivel.className = "dietas-borradores-ayuda";
    const resumenNivel = nodo(documento, "summary", "?");
    resumenNivel.setAttribute("aria-label", traducir("recorridos_abrir_ayuda"));
    ayudaNivel.append(resumenNivel, nodo(documento, "p", tBorradores("comision_nivel_ayuda")));
    const aceptacion = nodo(documento, "section");
    aceptacion.className = "dietas-comision-aceptacion";
    aceptacion.dataset.dietasAceptacion = "";
    aceptacion.hidden = true;
    const cabeceraAceptacion = nodo(documento, "div");
    cabeceraAceptacion.className = "dietas-comision-bloque-cabecera";
    cabeceraAceptacion.append(nodo(documento, "h4", tBorradores("comision_aceptacion_titulo")));
    const ayudaAceptacion = nodo(documento, "details");
    ayudaAceptacion.className = "dietas-borradores-ayuda";
    const resumenAceptacion = nodo(documento, "summary", "?");
    resumenAceptacion.setAttribute("aria-label", traducir("recorridos_abrir_ayuda"));
    ayudaAceptacion.append(resumenAceptacion, nodo(documento, "p", tBorradores("comision_aceptacion_ayuda")));
    cabeceraAceptacion.append(ayudaAceptacion);
    const grupoAceptacion = nodo(documento, "p");
    grupoAceptacion.dataset.dietasAceptacionGrupo = "";
    const etiquetaModo = nodo(documento, "label", tBorradores("comision_aceptacion_titulo"));
    const modo = nodo(documento, "select");
    modo.name = "modo_tramos";
    [["todos", "comision_aceptacion_todos"], ["uno", "comision_aceptacion_uno"]].forEach(([valor, clave]) => {
      const opcion = nodo(documento, "option", tBorradores(clave)); opcion.value = valor; modo.append(opcion);
    });
    etiquetaModo.append(modo);
    const etiquetaIndice = nodo(documento, "label", tBorradores("comision_aceptacion_indice"));
    const indice = nodo(documento, "select");
    indice.name = "tramo_indice";
    etiquetaIndice.append(indice);
    aceptacion.append(cabeceraAceptacion, grupoAceptacion, etiquetaModo, etiquetaIndice);
    const puntoRuta = (nombre, clave) => {
      const etiqueta = nodo(documento, "label", traducir(clave));
      const selector = nodo(documento, "select"); selector.name = nombre; selector.required = true;
      const inicial = nodo(documento, "option", traducir("borradores_propios_elegir_localidad")); inicial.value = ""; selector.append(inicial);
      puntosRuta.forEach((punto) => { const opcion=nodo(documento,"option",punto.nombre); opcion.value=punto.codigo; selector.append(opcion); });
      etiqueta.append(selector); return etiqueta;
    };
    const paradas = nodo(documento, "section");
    paradas.className = "dietas-borradores-paradas";
    paradas.dataset.dietasParadas = "";
    const tituloParadas = nodo(documento, "h4", tBorradores("borradores_propios_paradas_titulo"));
    const listaParadas = nodo(documento, "ol");
    listaParadas.dataset.dietasParadasLista = "";
    const anadirParada = nodo(documento, "button", tBorradores("borradores_propios_parada_anadir"));
    anadirParada.type = "button";
    anadirParada.className = "boton-secundario";
    anadirParada.dataset.dietasParadaAnadir = "";
    anadirParada.disabled = !conectada;
    paradas.append(tituloParadas, listaParadas, anadirParada);
    const calcularRuta = nodo(documento, "button", tBorradores("comision_calcular_ruta"));
    calcularRuta.type = "button";
    calcularRuta.className = "boton-secundario";
    calcularRuta.dataset.dietasCalcularRuta = "";
    calcularRuta.disabled = !calculadorRuta;
    const mapaRaiz = nodo(documento, "div");
    mapaRaiz.className = "dietas-comision-mapa-raiz";
    mapaRaiz.dataset.dietasComisionMapaRaiz = "";
    const etiquetaVehiculo = nodo(documento, "label", tBorradores("comision_vehiculo_propio"));
    const vehiculo = nodo(documento, "select");
    vehiculo.name = "vehiculo_propio";
    vehiculo.dataset.dietasVehiculoPropio = "";
    [["", "comision_vehiculo_elegir"], ["si", "comision_vehiculo_si"], ["no", "comision_vehiculo_no"]].forEach(([valor, clave]) => {
      const opcion = nodo(documento, "option", tBorradores(clave)); opcion.value = valor; vehiculo.append(opcion);
    });
    vehiculo.disabled = true;
    vehiculo.title = tBorradores("comision_otros_primero_guardar");
    etiquetaVehiculo.append(vehiculo);
    const rutasVehiculo = nodo(documento, "section");
    rutasVehiculo.className = "dietas-comision-rutas-vehiculo";
    rutasVehiculo.dataset.dietasRutasVehiculo = "";
    rutasVehiculo.append(nodo(documento, "h5", tBorradores("comision_rutas_titulo")));
    const listaRutas = nodo(documento, "div");
    listaRutas.dataset.dietasRutasLista = "";
    const anadirRuta = nodo(documento, "button", tBorradores("comision_ruta_anadir"));
    anadirRuta.type = "button"; anadirRuta.className = "boton-secundario";
    anadirRuta.dataset.dietasRutaAnadir = ""; anadirRuta.disabled = true;
    rutasVehiculo.append(listaRutas, anadirRuta);
    const boton = nodo(
      documento,
      "button",
      traducir("borradores_propios_guardar"),
    );
    boton.type = "submit";
    boton.className = "boton-primario";
    boton.dataset.dietasBorradorGuardar = "";
    boton.disabled = controlador !== null;
    const revisar = nodo(documento, "button", tBorradores("borradores_propios_revisar"));
    revisar.type = "button";
    revisar.className = "boton-secundario";
    revisar.dataset.dietasBorradorRevisar = "";
    const selectorRelacion = (() => {
      if (!relaciones.length) return null;
      const etiqueta = nodo(
        documento,
        "label",
        traducir("borradores_propios_referencia"),
      );
      const selector = nodo(documento, relaciones.length === 1 ? "input" : "select");
      selector.name = "relacion_ref";
      if (relaciones.length === 1) {
        selector.type = "hidden";
        selector.value = relaciones[0];
      } else {
        selector.required = true;
        const inicial = nodo(documento, "option", "—");
        inicial.value = "";
        selector.append(inicial);
        relaciones.forEach((referencia) => {
          const opcion = nodo(documento, "option", referencia);
          opcion.value = referencia;
          selector.append(opcion);
        });
        selector.value = relacionSeleccionada || "";
      }
      etiqueta.append(selector);
      return etiqueta;
    })();
    const cabecera = nodo(documento, "div");
    cabecera.className = "cabecera-panel";
    cabecera.append(nodo(documento,"h3",traducir("borradores_propios_nuevo")));
    const campos = nodo(documento, "div");
    campos.className = "cuerpo-panel dietas-borradores-campos";
    const tituloBloque = (clave, ayudaClave) => {
      const cabeceraBloque = nodo(documento, "div");
      cabeceraBloque.className = "dietas-comision-bloque-cabecera";
      cabeceraBloque.append(nodo(documento, "h4", tBorradores(clave)));
      const detalles = nodo(documento, "details");
      detalles.className = "dietas-borradores-ayuda";
      const abrir = nodo(documento, "summary", "?");
      abrir.setAttribute("aria-label", traducir("recorridos_abrir_ayuda"));
      detalles.append(abrir, nodo(documento, "p", tBorradores(ayudaClave)));
      cabeceraBloque.append(detalles);
      return cabeceraBloque;
    };
    const ayuda = nodo(documento, "details");
    ayuda.className = "dietas-borradores-ayuda";
    const abrirAyuda = nodo(documento, "summary", "?");
    abrirAyuda.setAttribute("aria-label", traducir("recorridos_abrir_ayuda"));
    ayuda.append(abrirAyuda, nodo(documento,"p",traducir("borradores_propios_ruta_ayuda")));
    ayuda.append(nodo(documento, "p", tBorradores("borradores_propios_paradas_ayuda")));
    ayuda.append(nodo(documento, "p", tBorradores("borradores_propios_preparacion_ayuda")));
    ayuda.append(nodo(documento, "p", tBorradores("borradores_propios_ya_registrado_ayuda")));
    const pais = nodo(documento, "label", tBorradores("borradores_propios_pais"));
    const paisValor = nodo(documento, "select");
    paisValor.name = "pais";
    paisValor.dataset.dietasPais = "";
    [["ES", "borradores_propios_pais_espana"], ["OTRO", "comision_pais_otro"]].forEach(([valor, clave]) => {
      const opcion = nodo(documento, "option", tBorradores(clave)); opcion.value = valor; paisValor.append(opcion);
    });
    paisValor.value = "ES";
    pais.append(paisValor);
    const nombrePais = campo("pais_otro", "comision_pais_nombre", "text", false);
    nombrePais.dataset.dietasPaisOtro = "";
    nombrePais.hidden = true;
    const estadoPais = nodo(documento, "p", tBorradores("comision_pais_sin_calculo"));
    estadoPais.dataset.dietasPaisSinCalculo = "";
    estadoPais.hidden = true;
    estadoPais.setAttribute("role", "status");
    campos.append(
      ayuda,
      tituloBloque("comision_bloque_dietas", "comision_bloque_dietas_ayuda"),
      campo("fecha_inicio", "borradores_propios_fecha_inicio", "date"),
      campo("hora_inicio", "borradores_propios_hora_inicio", "time"),
      campo("fecha_fin", "borradores_propios_fecha_fin", "date"),
      campo("hora_fin", "borradores_propios_hora_fin", "time"),
      motivo,
      etiquetaNivel,
      ayudaNivel,
      aceptacion,
      tituloBloque("comision_bloque_kilometraje", "comision_bloque_kilometraje_ayuda"),
      puntoRuta("origen_codigo","borradores_propios_origen"),
      paradas,
      puntoRuta("destino_codigo","borradores_propios_destino"),
      calcularRuta,
      mapaRaiz,
      etiquetaVehiculo,
      rutasVehiculo,
      pais,
      nombrePais,
      estadoPais,
      ...(selectorRelacion ? [selectorRelacion] : []),
    );
    const otros = nodo(documento, "section");
    otros.className = "dietas-comision-otros";
    otros.dataset.dietasOtros = "";
    otros.append(tituloBloque("comision_bloque_otros", "comision_bloque_otros_ayuda"));
    const listaOtros = nodo(documento, "div");
    listaOtros.dataset.dietasOtrosLista = "";
    const anadirOtro = nodo(documento, "button", tBorradores("comision_otros_anadir"));
    anadirOtro.type = "button";
    anadirOtro.className = "boton-secundario";
    anadirOtro.dataset.dietasOtroAnadir = "";
    anadirOtro.disabled = true;
    const indicacionOtros = nodo(documento, "p", tBorradores("comision_otros_primero_guardar"));
    indicacionOtros.dataset.dietasOtrosIndicacion = "";
    otros.append(listaOtros, indicacionOtros, anadirOtro);
    campos.append(otros);
    const acciones = nodo(documento, "div");
    acciones.className = "dietas-borradores-acciones";
    acciones.append(revisar, boton);
    const preparacion = nodo(documento, "section");
    preparacion.dataset.dietasBorradorPreparacion = "";
    preparacion.className = "dietas-comision-calculo";
    preparacion.hidden = true;
    preparacion.setAttribute("tabindex", "-1");
    form.append(cabecera, campos, preparacion, acciones);
    if (!conectada) {
      [...form.querySelectorAll("input"), ...form.querySelectorAll("select"), revisar, boton].forEach((control) => {
        control.disabled = true;
      });
    }
    return form;
  }
  async function cargarCatalogoRuta() {
    if (!calculadorRuta) return;
    controladorCatalogo?.abort();
    controladorCatalogo = new AbortController();
    const signal = controladorCatalogo.signal;
    try {
      const catalogo = await calculadorRuta.obtenerCatalogo({ signal });
      if (!activaAhora() || signal.aborted || !Array.isArray(catalogo?.puntos)) return;
      puntosRuta = catalogo.puntos;
      nombresRuta = new Map(puntosRuta.map((punto) => [punto.codigo, punto.nombre]));
      const controles = formularioPersistente.querySelectorAll("select").filter((selector) =>
        ["origen_codigo", "destino_codigo", "parada_codigo"].includes(selector.name));
      controles.forEach((selector) => {
        const anterior = selector.value;
        const inicial = nodo(documento, "option", traducir("borradores_propios_elegir_localidad"));
        inicial.value = "";
        selector.replaceChildren(inicial, ...puntosRuta.map((punto) => {
          const opcion = nodo(documento, "option", punto.nombre);
          opcion.value = punto.codigo;
          return opcion;
        }));
        selector.value = nombresRuta.has(anterior) ? anterior : "";
      });
      pintar();
    } catch {
      if (!activaAhora() || signal.aborted) return;
      puntosRuta = [];
      nombresRuta = new Map();
      mensaje("ruta_error_servicio", "error");
      pintar();
    } finally {
      if (controladorCatalogo?.signal === signal) controladorCatalogo = null;
    }
  }
  function valoresParadas(form) {
    return Array.from(form.querySelectorAll("select"))
      .filter((selector) => selector.name === "parada_codigo")
      .map((selector) => selector.value || "");
  }
  function pintarParadas(form, valores, focoIndice = -1) {
    const lista = form.querySelector("[data-dietas-paradas-lista]");
    if (!lista) return;
    lista.replaceChildren(...valores.map((valor, indice) => {
      const fila = nodo(documento, "li");
      const etiqueta = nodo(documento, "label", tBorradores("borradores_propios_parada_numero", { numero: indice + 1 }));
      const selector = nodo(documento, "select");
      selector.name = "parada_codigo";
      selector.required = true;
      const inicial = nodo(documento, "option", traducir("borradores_propios_elegir_localidad"));
      inicial.value = "";
      selector.append(inicial);
      puntosRuta.forEach((punto) => {
        const opcion = nodo(documento, "option", punto.nombre);
        opcion.value = punto.codigo;
        selector.append(opcion);
      });
      selector.value = valor;
      etiqueta.append(selector);
      const acciones = nodo(documento, "div");
      acciones.className = "dietas-borradores-parada-acciones";
      [
        ["dietasParadaSubir", "borradores_propios_parada_subir", indice === 0],
        ["dietasParadaBajar", "borradores_propios_parada_bajar", indice === valores.length - 1],
        ["dietasParadaQuitar", "borradores_propios_parada_quitar", false],
      ].forEach(([atributo, clave, bloqueado]) => {
        const boton = nodo(documento, "button", tBorradores(clave));
        boton.type = "button";
        boton.className = "boton-secundario";
        boton.dataset[atributo] = String(indice);
        boton.disabled = bloqueado || !conectada || controlador !== null;
        boton.setAttribute("aria-label", `${tBorradores(clave)}: ${tBorradores("borradores_propios_parada_numero", { numero: indice + 1 })}`);
        acciones.append(boton);
      });
      fila.append(etiqueta, acciones);
      return fila;
    }));
    const anadir = form.querySelector("[data-dietas-parada-anadir]");
    if (anadir) anadir.disabled = !conectada || controlador !== null || valores.length >= MAXIMO_LOCALIDADES - 2;
    mapaVista?.establecerCodigos([]);
    calculoCabeceraFirma = null;
    if (focoIndice >= 0) enfocar(lista.querySelectorAll("select")[focoIndice] || anadir);
  }
  function nuevaLineaOtro(valor = {}) {
    const fila = nodo(documento, "div");
    fila.className = "dietas-comision-otro-linea";
    fila.dataset.dietasOtroLinea = "";
    const campo = (nombre, clave, tipo = "text") => {
      const etiqueta = nodo(documento, "label", tBorradores(clave));
      const entrada = nodo(documento, "input");
      entrada.name = nombre;
      entrada.type = tipo;
      entrada.required = true;
      entrada.value = valor[nombre] ?? "";
      etiqueta.append(entrada);
      return etiqueta;
    };
    const etiquetaTipo = nodo(documento, "label", tBorradores("comision_otros_tipo"));
    const tipo = nodo(documento, "select");
    tipo.name = "tipo";
    [["otro_medio", "comision_otros_medio"], ["otro_gasto", "comision_otros_gasto"]].forEach(([codigo, clave]) => {
      const opcion = nodo(documento, "option", tBorradores(clave));
      opcion.value = codigo;
      tipo.append(opcion);
    });
    tipo.value = valor.tipo || "otro_gasto";
    etiquetaTipo.append(tipo);
    const importe = campo("importe", "comision_otros_importe");
    importe.querySelector("input").inputMode = "decimal";
    const quitar = nodo(documento, "button", tBorradores("comision_otros_quitar"));
    quitar.type = "button";
    quitar.className = "boton-secundario";
    quitar.dataset.dietasOtroQuitar = "";
    fila.append(etiquetaTipo, campo("concepto", "comision_otros_concepto"), importe, quitar);
    if (valor.justificante_ref && valor.justificante_sha256)
      metadatosOtros.set(fila, { justificante_ref: valor.justificante_ref, justificante_sha256: valor.justificante_sha256 });
    return fila;
  }
  function selectorLocalidad(nombre, clave, valor = "", variables = {}) {
    const etiqueta = nodo(documento, "label", tBorradores(clave, variables));
    const selector = nodo(documento, "select");
    selector.name = nombre;
    selector.required = true;
    const inicial = nodo(documento, "option", traducir("borradores_propios_elegir_localidad"));
    inicial.value = ""; selector.append(inicial);
    puntosRuta.forEach((punto) => { const opcion = nodo(documento, "option", punto.nombre); opcion.value = punto.codigo; selector.append(opcion); });
    selector.value = valor;
    etiqueta.append(selector);
    return etiqueta;
  }
  function nuevaRutaVehiculo(valor = {}) {
    const fila = nodo(documento, "section");
    fila.className = "dietas-comision-ruta-vehiculo";
    fila.dataset.dietasRutaLinea = "";
    const cabecera = nodo(documento, "div");
    cabecera.className = "cabecera-panel";
    cabecera.append(nodo(documento, "h6", tBorradores("comision_ruta_numero", { numero: 1 })));
    const quitar = nodo(documento, "button", tBorradores("comision_ruta_quitar"));
    quitar.type = "button"; quitar.className = "boton-secundario"; quitar.dataset.dietasRutaQuitar = "";
    cabecera.append(quitar);
    const cuerpo = nodo(documento, "div");
    cuerpo.className = "cuerpo-panel dietas-comision-ruta-campos";
    const codigos = valor.codigos_ruta || [];
    const paradas = nodo(documento, "div");
    paradas.dataset.dietasRutaParadas = "";
    const listaParadas = nodo(documento, "div");
    listaParadas.dataset.dietasRutaParadasLista = "";
    const anadirParada = nodo(documento, "button", tBorradores("borradores_propios_parada_anadir"));
    anadirParada.type = "button"; anadirParada.className = "boton-secundario";
    anadirParada.dataset.dietasRutaParadaAnadir = "";
    paradas.append(listaParadas, anadirParada);
    const ajuste = nodo(documento, "label", tBorradores("comision_ajuste_km"));
    const valorAjuste = nodo(documento, "input");
    valorAjuste.name = "ajuste_kilometros"; valorAjuste.type = "text"; valorAjuste.inputMode = "decimal";
    valorAjuste.value = valor.ajuste_kilometros || "0.0000"; valorAjuste.required = true;
    ajuste.append(valorAjuste);
    const motivo = nodo(documento, "label", tBorradores("comision_motivo_ajuste"));
    const motivoEntrada = nodo(documento, "input");
    motivoEntrada.name = "motivo_ajuste"; motivoEntrada.type = "text";
    motivoEntrada.value = valor.motivo_ajuste || "";
    motivo.append(motivoEntrada);
    const calcular = nodo(documento, "button", tBorradores("comision_ruta_calcular"));
    calcular.type = "button"; calcular.className = "boton-secundario";
    calcular.dataset.dietasRutaCalcular = "";
    const estado = nodo(documento, "p", tBorradores("comision_ruta_pendiente"));
    estado.dataset.dietasRutaEstado = "";
    estado.setAttribute("role", "status");
    cuerpo.append(selectorLocalidad("ruta_origen_codigo", "borradores_propios_origen", codigos[0] || ""),
      paradas, selectorLocalidad("ruta_destino_codigo", "borradores_propios_destino", codigos.at(-1) || ""),
      ajuste, motivo, calcular, estado);
    fila.append(cabecera, cuerpo);
    pintarParadasRuta(fila, codigos.slice(1, -1));
    return fila;
  }
  function pintarParadasRuta(fila, valores) {
    const lista = fila.querySelector("[data-dietas-ruta-paradas-lista]");
    lista.replaceChildren(...valores.map((codigo, indice) => {
      const grupo = nodo(documento, "div");
      grupo.className = "dietas-comision-ruta-parada";
      grupo.append(selectorLocalidad("ruta_parada_codigo", "borradores_propios_parada_numero", codigo, { numero: indice + 1 }));
      const acciones = nodo(documento, "div");
      [["dietasRutaParadaSubir", "borradores_propios_parada_subir", indice === 0],
        ["dietasRutaParadaBajar", "borradores_propios_parada_bajar", indice === valores.length - 1],
        ["dietasRutaParadaQuitar", "borradores_propios_parada_quitar", false]].forEach(([atributo, clave, bloqueado]) => {
        const boton = nodo(documento, "button", tBorradores(clave));
        boton.type = "button"; boton.className = "boton-secundario";
        boton.dataset[atributo] = String(indice); boton.disabled = bloqueado;
        acciones.append(boton);
      });
      grupo.append(acciones);
      return grupo;
    }));
    const anadir = fila.querySelector("[data-dietas-ruta-parada-anadir]");
    if (anadir) anadir.disabled = valores.length >= 10;
  }
  function codigosRutaVehiculo(fila) {
    const valor = (nombre) => fila.querySelectorAll("select").find((selector) => selector.name === nombre)?.value || "";
    return [valor("ruta_origen_codigo"),
      ...fila.querySelectorAll("select").filter((selector) => selector.name === "ruta_parada_codigo").map((selector) => selector.value || ""),
      valor("ruta_destino_codigo")];
  }
  function rutasDesdeFormulario(form) {
    const vehiculo = form.querySelector("[data-dietas-vehiculo-propio]")?.value;
    if (vehiculo === "no") return { vehiculo_propio: false, rutas: [] };
    if (vehiculo !== "si") throw new TypeError("vehículo propio sin confirmar");
    const filas = form.querySelectorAll("[data-dietas-ruta-linea]");
    if (filas.length < 1 || filas.length > 8) throw new TypeError("rutas no válidas");
    const rutas = filas.map((fila) => {
      const codigos = codigosRutaVehiculo(fila);
      if (!rutaValida(codigos) || !rutasCalculadas.has(JSON.stringify(codigos))) throw new TypeError("ruta no calculada");
      const ajuste = fila.querySelectorAll("input").find((entrada) => entrada.name === "ajuste_kilometros")?.value || "";
      const motivo = String(fila.querySelectorAll("input").find((entrada) => entrada.name === "motivo_ajuste")?.value || "").trim();
      if (!/^-?(?:0|[1-9]\d{0,3})\.\d{4}$/u.test(ajuste) || Math.abs(Number(ajuste)) > 1000 ||
          ajuste === "-0.0000" ||
          (Number(ajuste) === 0 ? motivo !== "" : motivo.length < 3 || motivo.length > 500))
        throw new TypeError("ajuste no válido");
      return { codigos_ruta: codigos, ajuste_kilometros: ajuste, motivo_ajuste: motivo };
    });
    return { vehiculo_propio: true, rutas };
  }
  function pintarAceptacion(item) {
    const seccion = formularioPersistente.querySelector("[data-dietas-aceptacion]");
    if (!seccion || !asignacionVerificada() || !item?.comision.calculo) return;
    const grupo = asignacion.grupo_dieta;
    const tramos = item.comision.calculo.opciones_dieta[grupo - 1]?.calculo.tramos || [];
    const rotuloGrupo = seccion.querySelector("[data-dietas-aceptacion-grupo]");
    rotuloGrupo.textContent = `${traducir("borradores_propios_grupo")} ${grupo} · ${item.comision.calculo.rotulo}`;
    const modo = seccion.querySelectorAll("select").find((selector) => selector.name === "modo_tramos");
    const indice = seccion.querySelectorAll("select").find((selector) => selector.name === "tramo_indice");
    indice.replaceChildren(...tramos.map((tramo, posicion) => {
      const opcion = nodo(documento, "option", `${fechaLegible(tramo.fecha)} · ${tramo.tipo === "manutencion" ?
        traducir("borradores_propios_manutencion") : traducir("borradores_propios_alojamiento_tope")} · ${euros(tramo.importe_centimos)}`);
      opcion.value = String(posicion);
      return opcion;
    }));
    const aceptados = item.comision.documento?.tramos_aceptados;
    modo.value = tramos.length > 1 && aceptados?.length === 1 ? "uno" : "todos";
    modo.disabled = tramos.length < 2;
    indice.value = String(aceptados?.length === 1 ? aceptados[0] : 0);
    const etiquetaIndice = indice.closest("label");
    if (etiquetaIndice) etiquetaIndice.hidden = modo.value !== "uno" || tramos.length === 0;
    seccion.hidden = false;
    if (tramos.length === 0) rotuloGrupo.textContent += ` · ${tBorradores("comision_aceptacion_sin_tramos")}`;
  }
  function tramosAceptadosDesdeFormulario(form) {
    if (!edicion || !asignacionVerificada()) throw new TypeError("grupo de Dietas no acreditado");
    const tramos = edicion.comision.calculo?.opciones_dieta?.[asignacion.grupo_dieta - 1]?.calculo?.tramos;
    if (!Array.isArray(tramos)) throw new TypeError("tramos de Dietas no disponibles");
    if (tramos.length === 0) return [];
    const modo = form.querySelectorAll("select").find((selector) => selector.name === "modo_tramos")?.value;
    if (modo === "todos") return tramos.map((_tramo, indice) => indice);
    const elegido = Number(form.querySelectorAll("select").find((selector) => selector.name === "tramo_indice")?.value);
    if (modo !== "uno" || !Number.isSafeInteger(elegido) || elegido < 0 || elegido >= tramos.length)
      throw new TypeError("tramo de Dietas no seleccionado");
    return [elegido];
  }
  function renumerarRutas(form) {
    form.querySelectorAll("[data-dietas-ruta-linea]").forEach((fila, indice) => {
      const titulo = fila.querySelector("h6");
      if (titulo) titulo.textContent = tBorradores("comision_ruta_numero", { numero: indice + 1 });
    });
  }
  function otrosDesdeFormulario(form) {
    const filas = form.querySelectorAll("[data-dietas-otro-linea]");
    return Array.from(filas).map((fila) => {
      const valor = (nombre) => String(fila.querySelectorAll("input").find((entrada) => entrada.name === nombre)?.value || "").trim();
      const eurosTexto = valor("importe").replace(",", ".");
      if (!/^(?:0|[1-9]\d{0,5})(?:\.\d{1,2})?$/u.test(eurosTexto)) throw new TypeError("importe no válido");
      const [enteros, decimales = ""] = eurosTexto.split(".");
      const custodia = metadatosOtros.get(fila);
      return {
        tipo: fila.querySelector("select")?.value,
        concepto: valor("concepto"),
        importe_centimos: Number(enteros) * 100 + Number(decimales.padEnd(2, "0")),
        justificante_ref: custodia?.justificante_ref || "",
        justificante_sha256: custodia?.justificante_sha256 || "",
      };
    });
  }
  function codigosRuta(form, datos) {
    return [String(datos.get("origen_codigo") || ""), ...valoresParadas(form), String(datos.get("destino_codigo") || "")];
  }
  function rutaValida(codigos) {
    return codigos.length >= 2 && codigos.length <= MAXIMO_LOCALIDADES &&
      codigos.every((codigo) => nombresRuta.has(codigo)) &&
      new Set(codigos).size === codigos.length;
  }
  function recibo(item) {
    const seccion = nodo(documento, "section");
    seccion.className = "dietas-recibo";
    seccion.dataset.dietasBorradorRecibo = "";
    seccion.setAttribute("role", "status");
    seccion.setAttribute("aria-live", "polite");
    seccion.append(
      nodo(documento, "strong", traducir("borradores_propios_recibo_titulo")),
    );
    const datos = nodo(documento, "dl");
    [
      ["borradores_propios_recibo_referencia", item.recibo.referencia],
      ["borradores_propios_recibo_version", String(item.recibo.version)],
      [
        "borradores_propios_recibo_fecha",
        fechaLegible(item.recibo.registrado_en, true),
      ],
      ["borradores_propios_recibo_instante_exacto", item.recibo.registrado_en],
    ].forEach(([etiqueta, valor]) => {
      const fila = nodo(documento, "div");
      fila.append(
        nodo(documento, "dt", traducir(etiqueta)),
        nodo(documento, "dd", valor),
      );
      datos.append(fila);
    });
    seccion.append(datos);
    if (item.recibo.repeticion)
      seccion.append(
        nodo(documento, "p", traducir("borradores_propios_repeticion")),
      );
    return seccion;
  }
  function listado() {
    const seccion = nodo(documento, "section");
    seccion.className = "panel dietas-listado";
    seccion.dataset.dietasBorradoresListado = "";
    seccion.setAttribute("tabindex", "-1");
    const cabecera = nodo(documento, "div");
    cabecera.className = "cabecera-panel";
    const consultar = nodo(documento, "button", tBorradores("borradores_propios_consultar_registrados"));
    consultar.type = "button";
    consultar.className = "boton-secundario";
    consultar.dataset.dietasBorradorRecargar = "";
    consultar.dataset.dietasBorradorConsultarRegistrados = "";
    consultar.disabled = !conectada || controlador !== null || (relaciones.length > 1 && !relacionSeleccionada);
    if (!conectada) consultar.title = traducir("borradores_propios_pendiente_conexion");
    else if (relaciones.length > 1 && !relacionSeleccionada) consultar.title = traducir("borradores_propios_error_relacion");
    const ayuda = nodo(documento, "details");
    ayuda.className = "dietas-borradores-ayuda";
    ayuda.dataset.dietasBorradoresAyuda = "";
    ayuda.open = false;
    const resumenAyuda = nodo(documento, "summary", "?");
    resumenAyuda.setAttribute("aria-label", traducir("recorridos_abrir_ayuda"));
    ayuda.append(resumenAyuda, nodo(documento, "p", tBorradores("borradores_propios_consulta_ayuda")));
    // El «?» comparte la cabecera con la acción, como en el resto de paneles del portal.
    const accionesCabecera = nodo(documento, "div");
    accionesCabecera.className = "dietas-recorridos-cabecera-acciones";
    accionesCabecera.append(ayuda, consultar);
    cabecera.append(nodo(documento, "h3", traducir("borradores_propios_listado")), accionesCabecera);
    seccion.append(cabecera);
    const cuerpo = nodo(documento, "div");
    cuerpo.className = "cuerpo-panel";
    seccion.append(cuerpo);
    if (!conectada) {
      cuerpo.append(
        nodo(documento, "p", traducir("borradores_propios_pendiente_conexion")),
      );
      return seccion;
    }
    if (estado.carga) {
      const carga = nodo(documento, "p", traducir("borradores_propios_cargando"));
      carga.className = "dietas-borradores-indicacion";
      cuerpo.append(carga);
      return seccion;
    }
    if (estado.errorLista) {
      // El aviso del panel ya muestra este mismo error; no se repite dentro de la lista.
      if ((estado.errorListaClave || estado.mensaje) === estado.mensaje) return seccion;
      const fallo = nodo(documento, "p", tBorradores(estado.errorListaClave || estado.mensaje));
      fallo.className = "dietas-borradores-indicacion dietas-borradores-indicacion-error";
      cuerpo.append(fallo);
      return seccion;
    }
    if (!estado.items.length) {
      const vacio = nodo(documento, "p", traducir("borradores_propios_vacio"));
      vacio.dataset.dietasBorradoresVacio = "";
      vacio.className = "dietas-borradores-indicacion";
      cuerpo.append(vacio);
      return seccion;
    }
    const ul = nodo(documento, "ul");
    ul.className = "dietas-borradores-lista";
    estado.items.forEach((item) => {
      const li = nodo(documento, "li");
      li.className = "dietas-borradores-fila";
      li.dataset.seleccionado = String(estado.detalle?.comision.referencia === item.comision.referencia);
      const boton = nodo(
        documento,
        "button",
        `${item.comision.numero_documento || tBorradores("comision_documento_fecha", { fecha: fechaLegible(item.comision.fecha_inicio) })} · ${item.comision.motivo}`,
      );
      boton.type = "button";
      boton.className = "enlace-tabla";
      boton.dataset.dietasBorradorDetalle = item.comision.referencia;
      boton.dataset.dietasBorradorRelacion = item.comision.relacion_ref;
      boton.setAttribute(
        "aria-label",
        `${traducir("borradores_propios_seleccionar")}: ${item.comision.motivo}, ${fechaLegible(item.comision.fecha_inicio)}`,
      );
      if (estado.detalle?.comision.referencia === item.comision.referencia)
        boton.setAttribute("aria-current", "true");
      li.append(
        boton,
        ...(item.comision.fecha_apertura ? [nodo(documento, "small", `${tBorradores("comision_fecha_apertura")}: ${fechaLegible(item.comision.fecha_apertura, true)}`)] : []),
        (() => {
          const [clave, tono] = ESTADOS_COMISION[item.comision.estado] || ["comision_estado_borrador", "info"];
          const chip = nodo(documento, "span", tBorradores(clave));
          chip.className = `estado-chip ${tono}`;
          chip.dataset.dietasEstadoComision = item.comision.estado;
          return chip;
        })(),
      );
      ul.append(li);
    });
    cuerpo.append(ul);
    const paginacion = nodo(documento, "nav");
    paginacion.className = "dietas-borradores-paginacion";
    paginacion.setAttribute("aria-label", traducir("borradores_propios_listado"));
    const anterior = nodo(documento, "button", traducir("borradores_propios_pagina_anterior"));
    anterior.type = "button";
    anterior.dataset.dietasBorradorPagina = "anterior";
    anterior.disabled = indicePagina === 0;
    const siguiente = nodo(documento, "button", traducir("borradores_propios_pagina_siguiente"));
    siguiente.type = "button";
    siguiente.dataset.dietasBorradorPagina = "siguiente";
    siguiente.disabled = !siguienteCursor;
    paginacion.append(anterior, nodo(documento, "span", traducir("borradores_propios_mostrando", { inicio: indicePagina * 6 + 1, fin: indicePagina * 6 + estado.items.length })), siguiente);
    cuerpo.append(paginacion);
    return seccion;
  }
  function detalle() {
    const seccion = nodo(documento, "section");
    seccion.className = "panel dietas-detalle";
    seccion.dataset.dietasBorradorFicha = "";
    seccion.setAttribute("tabindex", "-1");
    seccion.setAttribute("aria-label", traducir("borradores_propios_detalle"));
    const cabecera = nodo(documento, "div");
    cabecera.className = "cabecera-panel";
    cabecera.append(nodo(documento, "h3", traducir("borradores_propios_detalle")));
    seccion.append(cabecera);
    const cuerpo = nodo(documento, "div");
    cuerpo.className = "cuerpo-panel";
    seccion.append(cuerpo);
    const item = estado.detalle;
    if (!item) {
      cuerpo.append(
        nodo(documento, "p", tBorradores("borradores_propios_estado_sin_seleccion")),
      );
      return seccion;
    }
    const datos = nodo(documento, "dl");
    [
      ["comision_numero_documento", item.comision.numero_documento ||
        tBorradores("comision_documento_fecha", { fecha: fechaLegible(item.comision.fecha_inicio) })],
      ...(item.comision.fecha_apertura ? [["comision_fecha_apertura", fechaLegible(item.comision.fecha_apertura, true)]] : []),
      ["borradores_propios_referencia", item.comision.referencia],
      [
        "borradores_propios_fecha_inicio",
        fechaLegible(item.comision.fecha_inicio),
      ],
      ["borradores_propios_fecha_fin", fechaLegible(item.comision.fecha_fin)],
      ["borradores_propios_motivo", item.comision.motivo],
      ["borradores_propios_ruta", rutaLegible(item.comision.codigos_ruta, tBorradores, nombresRuta)],
      ["borradores_propios_pais", tBorradores("comision_pais_es_registrado")],
    ].forEach(([etiqueta, valor]) => {
      const fila = nodo(documento, "div");
      fila.append(
        nodo(documento, "dt", traducir(etiqueta)),
        nodo(documento, "dd", valor),
      );
      datos.append(fila);
    });
    cuerpo.append(datos);
    const panelAsignacion = nodo(documento, "section");
    panelAsignacion.className = "dietas-comision-asignacion";
    const cabeceraAsignacion = nodo(documento, "div");
    cabeceraAsignacion.className = "cabecera-panel";
    cabeceraAsignacion.append(nodo(documento, "h4", tBorradores("comision_asignacion_titulo")));
    panelAsignacion.append(cabeceraAsignacion);
    const cuerpoAsignacion = nodo(documento, "div");
    cuerpoAsignacion.className = "cuerpo-panel";
    if (asignacionVerificada()) {
      const aviso = nodo(documento, "p", tBorradores("comision_asignacion_verificada"));
      aviso.className = "estado-chip exito";
      cuerpoAsignacion.append(aviso);
      const metadatos = nodo(documento, "dl");
      [["comision_asignacion_centro", asignacion.centro_ref],
        ["comision_asignacion_unidad", asignacion.unidad_ref],
        ["comision_asignacion_administrativo", asignacion.administrativo_persona_ref],
        ["comision_asignacion_responsable", asignacion.responsable_persona_ref]].forEach(([clave, referencia]) => {
        const fila = nodo(documento, "div");
        fila.append(nodo(documento, "dt", tBorradores(clave)),
          nodo(documento, "dd", tBorradores("comision_asignacion_nombre_no_disponible")),
          nodo(documento, "small", referencia));
        metadatos.append(fila);
      });
      const grupo = nodo(documento, "div");
      grupo.append(nodo(documento, "dt", tBorradores("comision_asignacion_grupo")),
        nodo(documento, "dd", String(asignacion.grupo_dieta)));
      metadatos.append(grupo);
      cuerpoAsignacion.append(metadatos);
    } else cuerpoAsignacion.append(nodo(documento, "p", tBorradores(asignacionMensaje)));
    if (clienteRectificacion && !asignacionVerificada()) {
      const corregir = nodo(documento, "button", tBorradores("comision_asignacion_corregir"));
      corregir.type = "button";
      corregir.className = "boton-secundario";
      corregir.disabled = true;
      corregir.title = tBorradores("comision_asignacion_corregir_pendiente");
      cuerpoAsignacion.append(corregir);
    }
    if (reciboRectificacionConfirmado?.relacion_ref === item.comision.relacion_ref) {
      const reciboSolicitud = nodo(documento, "p", tBorradores("comision_rectificacion_recibo", {
        recibo: reciboRectificacionConfirmado.recibo_ref,
      }));
      reciboSolicitud.className = "estado-chip exito";
      reciboSolicitud.setAttribute("role", "status");
      cuerpoAsignacion.append(reciboSolicitud);
    }
    panelAsignacion.append(cuerpoAsignacion);
    cuerpo.append(panelAsignacion);
    const acciones = nodo(documento, "div");
    acciones.className = "dietas-borradores-acciones";
    const esBorrador = item.comision.estado === "borrador";
    [["comision_editar", "dietasBorradorEditar", "editar"],
      ["comision_eliminar", "dietasBorradorEliminar", "eliminar"],
      ["comision_enviar", "dietasBorradorEnviar", "enviar"]].forEach(([clave, atributo, metodo]) => {
      const boton = nodo(documento, "button", tBorradores(clave));
      boton.type = "button";
      boton.className = metodo === "enviar" ? "boton-primario" : "boton-secundario";
      boton.dataset[atributo] = item.comision.referencia;
      boton.disabled = !esBorrador || estadoRelaciones === "no_disponible" || typeof cliente?.[metodo] !== "function" || controlador !== null ||
        (metodo !== "eliminar" && !asignacionVerificada()) ||
        (metodo === "editar" && !item.comision.calculo) ||
        (metodo === "enviar" && !item.comision.documento);
      if (boton.disabled) boton.title = tBorradores(metodo === "enviar" && !item.comision.documento
        ? "comision_envio_sin_documento" : "comision_accion_no_disponible");
      acciones.append(boton);
    });
    cuerpo.append(acciones);
    if (item.comision.calculo) cuerpo.append(resumenCalculo(item.comision.calculo, item.comision.documento));
    cuerpo.append(recibo(item));
    return seccion;
  }
  function resumenCalculo(calculo, documentoComision) {
    const resumen=nodo(documento,"section"); resumen.className="dietas-comision-calculo";
    const tituloCalculo = nodo(documento,"h4",traducir("borradores_propios_calculo"));
    if (documentoComision) {
      // La advertencia de total no definitivo es ayuda: se abre desde «?».
      const cabeceraCalculo = nodo(documento, "div"); cabeceraCalculo.className = "dietas-comision-bloque-cabecera";
      const ayudaTotal = nodo(documento, "details"); ayudaTotal.className = "dietas-borradores-ayuda";
      const abrirAyudaTotal = nodo(documento, "summary", "?");
      abrirAyudaTotal.setAttribute("aria-label", traducir("recorridos_abrir_ayuda"));
      ayudaTotal.append(abrirAyudaTotal, nodo(documento, "p", tBorradores("comision_total_no_definitivo")));
      cabeceraCalculo.append(tituloCalculo, ayudaTotal);
      resumen.append(cabeceraCalculo);
    } else resumen.append(tituloCalculo);
    const cifras=nodo(documento,"div"); cifras.className="dietas-comision-cifras";
    [[traducir("borradores_propios_km"),`${Number(calculo.kilometros).toLocaleString("es-ES",{maximumFractionDigits:1})} km`],
      [traducir("borradores_propios_importe_km"),euros(calculo.importe_kilometraje_centimos)]].forEach(([titulo,valor])=>{
        const tarjeta=nodo(documento,"article"); tarjeta.className="tarjeta-kpi";
        tarjeta.append(nodo(documento,"small",titulo),nodo(documento,"strong",valor)); cifras.append(tarjeta);
      });
    resumen.append(cifras,nodo(documento,"p",nivelDetalle === "alto"
      ? `${calculo.rotulo} · ${calculo.version_tarifa} · ${calculo.version_grafo}` : calculo.rotulo));
    const ruta=nodo(documento,"ol"); ruta.className="dietas-comision-tramos-ruta";
    if (calculo.rutas?.length) {
      calculo.rutas.forEach((rutaCalculada, indice) => ruta.append(nodo(documento, "li",
        `${tBorradores("comision_ruta_numero", { numero: indice + 1 })}: ${rutaLegible(rutaCalculada.codigos_ruta, tBorradores, nombresRuta)} · ${tBorradores("comision_km_base")} ${rutaCalculada.kilometros_base} km · ${tBorradores("comision_km_ajuste")} ${rutaCalculada.ajuste_kilometros} km · ${tBorradores("comision_km_final")} ${rutaCalculada.kilometros_finales} km · ${euros(rutaCalculada.importe_centimos)}`)));
    } else calculo.tramos_ruta.forEach((tramo)=>ruta.append(nodo(documento,"li",`${nombresRuta.get(tramo.origen_codigo)||tramo.origen_codigo} → ${nombresRuta.get(tramo.destino_codigo)||tramo.destino_codigo} · ${Number(tramo.kilometros).toLocaleString("es-ES",{maximumFractionDigits:1})} km`)));
    if (nivelDetalle !== "bajo") resumen.append(nodo(documento,"h5",traducir("borradores_propios_tramos_ruta")),ruta);
    if (!asignacionVerificada() && !documentoComision) {
      const aviso=nodo(documento,"p",traducir("borradores_propios_grupo_pendiente")); aviso.className="estado-chip aviso"; resumen.append(aviso);
    }
    const opcionesVisibles = documentoComision ? [] : asignacionVerificada()
      ? calculo.opciones_dieta.filter((opcion) => opcion.grupo === asignacion.grupo_dieta)
      : calculo.opciones_dieta;
    if (nivelDetalle !== "bajo") opcionesVisibles.forEach((opcion)=>{
      const bloque=nodo(documento,"section"); bloque.className="dietas-comision-grupo";
      bloque.append(nodo(documento,"h5",`${traducir("borradores_propios_grupo")} ${opcion.grupo} · ${euros(opcion.calculo.total_maximo_orientativo_centimos)}`));
      const lista=nodo(documento,"ul");
      opcion.calculo.tramos.forEach((tramo)=>lista.append(nodo(documento,"li",`${fechaLegible(tramo.fecha)} · ${tramo.tipo==="manutencion"?traducir("borradores_propios_manutencion"):traducir("borradores_propios_alojamiento_tope")} ${tramo.porcentaje}% · ${euros(tramo.importe_centimos)}`)));
      if (nivelDetalle === "alto") bloque.append(lista);
      resumen.append(bloque);
    });
    if (documentoComision) {
      const dietaCentimos = documentoComision.manutencion_centimos + documentoComision.alojamiento_tope_centimos;
      const categorias = nodo(documento, "div"); categorias.className = "dietas-comision-cifras";
      [["comision_total_dietas", dietaCentimos], ["comision_total_km", documentoComision.kilometraje_centimos],
        ["comision_total_otros", documentoComision.otros_centimos],
        ["comision_total_provisional", documentoComision.total_orientativo_centimos]].forEach(([clave, importe]) => {
        const tarjeta = nodo(documento, "article"); tarjeta.className = "tarjeta-kpi";
        tarjeta.append(nodo(documento, "small", tBorradores(clave)), nodo(documento, "strong", euros(importe)));
        categorias.append(tarjeta);
      });
      resumen.append(categorias);
      const grupoDocumento = nodo(documento, "p", `${tBorradores("comision_grupo_documento")}: ${documentoComision.grupo_dieta}`);
      grupoDocumento.className = "estado-chip info";
      resumen.append(grupoDocumento);
      if (nivelDetalle !== "bajo") {
        const aceptados = nodo(documento, "section"); aceptados.className = "dietas-comision-grupo";
        aceptados.append(nodo(documento, "h5", tBorradores("comision_tramos_aceptados")));
        if (nivelDetalle === "alto") {
          const lista = nodo(documento, "ul");
          documentoComision.lineas.filter((linea) => linea.tipo === "dieta").forEach((linea) =>
            lista.append(nodo(documento, "li", `${fechaLegible(linea.fecha)} · ${linea.concepto === "manutencion" ?
              traducir("borradores_propios_manutencion") : traducir("borradores_propios_alojamiento_tope")} · ${euros(linea.importe_centimos)}`)));
          aceptados.append(lista);
        }
        resumen.append(aceptados);
      }
      const otros = documentoComision.lineas.filter((linea) => linea.tipo === "otro_medio" || linea.tipo === "otro_gasto");
      const bloqueOtros = nodo(documento, "section");
      bloqueOtros.className = "dietas-comision-grupo";
      bloqueOtros.append(nodo(documento, "h5", tBorradores("comision_bloque_otros")));
      const listaOtros = nodo(documento, "ul");
      otros.forEach((linea) => listaOtros.append(nodo(documento, "li", `${linea.concepto} · ${euros(linea.importe_centimos)}`)));
      bloqueOtros.append(listaOtros);
      resumen.append(bloqueOtros);
    } else if (!asignacionVerificada()) resumen.append(nodo(documento, "p", tBorradores("comision_total_sin_grupo")));
    return resumen;
  }
  function pintar() {
    if (!activaAhora()) return;
    if (!formularioPersistente) {
      formularioPersistente = formulario();
      avisoPersistente = nodo(documento, "p");
      avisoPersistente.dataset.dietasBorradoresEstado = "";
      avisoPersistente.setAttribute("aria-live", "polite");
      avisoPersistente.setAttribute("tabindex", "-1");
      const espacio = nodo(documento, "div");
      espacio.className = "dietas-borradores-espacio";
      const principal = nodo(documento, "div");
      principal.className = "dietas-borradores-principal";
      listaPersistente = nodo(documento, "div");
      principal.append(listaPersistente, formularioPersistente);
      fichaPersistente = nodo(documento, "div");
      fichaPersistente.className = "dietas-borradores-lateral";
      rectificacionContenedor = nodo(documento, "div");
      rectificacionContenedor.className = "dietas-borradores-rectificacion";
      espacio.append(principal, fichaPersistente);
      raiz.append(nodo(documento, "h2", formularioInicialmenteVisible
        ? traducir("borradores_propios_titulo")
        : tBorradores("borradores_propios_titulo_registrados")), avisoPersistente, espacio);
      if (calculadorRuta && visorRuta) {
        const mapaRaiz = formularioPersistente.querySelector("[data-dietas-comision-mapa-raiz]");
        montajeMapa = montarVistaMapaComisionDietas({ raiz: mapaRaiz, calculador: calculadorRuta,
          visorRuta, anunciar: (texto, tono) => anunciar(texto, tono) }).then((vista) => {
          if (!activaAhora()) { vista.desmontar(); return null; }
          mapaVista = vista;
          return vista;
        }, () => null);
        void cargarCatalogoRuta();
      }
    }
    raiz.dataset.formularioVisible = String(formularioVisible);
    formularioPersistente.hidden = !formularioVisible;
    const extranjero = formularioPersistente.querySelector("[data-dietas-pais]")?.value === "OTRO";
    const paisOtro = formularioPersistente.querySelector("[data-dietas-pais-otro]");
    const estadoPais = formularioPersistente.querySelector("[data-dietas-pais-sin-calculo]");
    if (paisOtro) paisOtro.hidden = !extranjero;
    if (estadoPais) estadoPais.hidden = !extranjero;
    formularioPersistente.querySelectorAll("select").filter((selector) =>
      ["origen_codigo", "destino_codigo", "parada_codigo"].includes(selector.name)).forEach((selector) => { selector.required = !extranjero; });
    const botonGuardar = formularioPersistente.querySelector("[data-dietas-borrador-guardar]");
    if (botonGuardar) botonGuardar.textContent = tBorradores(edicion ? "comision_editar" : "borradores_propios_guardar");
    const controlesBloqueados = !conectada || controlador !== null || estadoRelaciones === "no_disponible" || (relaciones.length > 1 && !relacionSeleccionada);
    for (const selector of ["[data-dietas-borrador-guardar]", "[data-dietas-borrador-revisar]", "[data-dietas-calcular-ruta]"]) {
      const boton = formularioPersistente.querySelector(selector);
      if (boton) boton.disabled = controlesBloqueados || extranjero || (selector === "[data-dietas-calcular-ruta]" && !calculadorRuta);
    }
    if (formularioPersistente) {
      const vehiculo = formularioPersistente.querySelector("[data-dietas-vehiculo-propio]");
      if (vehiculo) vehiculo.disabled = controlesBloqueados || !edicion || typeof cliente?.editar !== "function";
      const rutasVehiculo = formularioPersistente.querySelector("[data-dietas-rutas-vehiculo]");
      if (rutasVehiculo) rutasVehiculo.hidden = vehiculo?.value !== "si";
      const anadirRuta = formularioPersistente.querySelector("[data-dietas-ruta-anadir]");
      if (anadirRuta) anadirRuta.disabled = controlesBloqueados || !edicion || vehiculo?.value !== "si" || formularioPersistente.querySelectorAll("[data-dietas-ruta-linea]").length >= 8;
      const seccionAceptacion = formularioPersistente.querySelector("[data-dietas-aceptacion]");
      if (seccionAceptacion) seccionAceptacion.hidden = !(edicion && asignacionVerificada());
      const modoAceptacion = formularioPersistente.querySelectorAll("select").find((selector) => selector.name === "modo_tramos");
      const indiceAceptacion = formularioPersistente.querySelectorAll("select").find((selector) => selector.name === "tramo_indice");
      const etiquetaIndice = indiceAceptacion?.closest("label");
      if (etiquetaIndice) etiquetaIndice.hidden = modoAceptacion?.value !== "uno";
      const anadirOtro = formularioPersistente.querySelector("[data-dietas-otro-anadir]");
      if (anadirOtro) anadirOtro.disabled = controlesBloqueados || !edicion || typeof cliente?.editar !== "function" || formularioPersistente.querySelectorAll("[data-dietas-otro-linea]").length >= 32;
      const indicacionOtros = formularioPersistente.querySelector("[data-dietas-otros-indicacion]");
      if (indicacionOtros) indicacionOtros.hidden = Boolean(edicion);
      const anadir = formularioPersistente.querySelector("[data-dietas-parada-anadir]");
      if (anadir) anadir.disabled = controlesBloqueados || valoresParadas(formularioPersistente).length >= MAXIMO_LOCALIDADES - 2;
      const totalParadas = valoresParadas(formularioPersistente).length;
      Array.from(formularioPersistente.querySelectorAll("button")).filter((boton) =>
        boton.dataset.dietasParadaSubir !== undefined || boton.dataset.dietasParadaBajar !== undefined || boton.dataset.dietasParadaQuitar !== undefined,
      ).forEach((boton) => {
        const subir = boton.dataset.dietasParadaSubir !== undefined;
        const bajar = boton.dataset.dietasParadaBajar !== undefined;
        const indice = Number(subir ? boton.dataset.dietasParadaSubir : bajar ? boton.dataset.dietasParadaBajar : boton.dataset.dietasParadaQuitar);
        boton.disabled = controlesBloqueados || (subir && indice === 0) || (bajar && indice === totalParadas - 1);
      });
    }
    avisoPersistente.textContent = tBorradores(estadoRelaciones === "no_disponible"
      ? (motivoRelaciones ? `comision_${motivoRelaciones}` : "comision_relaciones_no_disponibles") : estado.mensaje);
    avisoPersistente.dataset.tono = estado.tono;
    avisoPersistente.className = `estado-chip ${estado.tono === "error" ? "peligro" : estado.tono === "exito" ? "exito" : estado.tono === "aviso" ? "aviso" : "info"}`;
    avisoPersistente.setAttribute("role", estado.tono === "error" ? "alert" : "status");
    listaPersistente.replaceChildren(listado());
    fichaPersistente.replaceChildren(detalle(), rectificacionContenedor);
    const claveRectificacion = asignacionVerificada() && clienteRectificacion
      ? `${asignacion.asignacion_ref}:${asignacion.version}:${asignacion.fecha_referencia}` : null;
    if (claveRectificacion !== rectificacionClave) {
      vistaRectificacion?.desmontar();
      vistaRectificacion = null;
      rectificacionClave = claveRectificacion;
      rectificacionContenedor.replaceChildren();
      if (claveRectificacion) {
        try {
          vistaRectificacion = montarVistaRectificacionDietas(rectificacionContenedor, {
            cliente: clienteRectificacion, asignacion,
            alConfirmar: (resultado) => {
              reciboRectificacionConfirmado = { ...resultado, relacion_ref: asignacion.relacion_ref };
              if (estado.detalle) void consultarAsignacion(estado.detalle);
            },
          });
        } catch { rectificacionClave = null; }
      }
    }
  }
  async function cargar(conservarMensaje = false, consultaExplicita = false) {
    if (!conectada) {
      pintar();
      return;
    }
    if (relaciones.length > 1 && !relacionSeleccionada) {
      estado = { ...estado, carga: false, items: [], detalle: null, errorLista: true, errorListaClave: "borradores_propios_error_relacion" };
      mensaje("borradores_propios_error_relacion", "aviso");
      pintar();
      return;
    }
    controlador?.abort();
    controlador = new AbortController();
    estado = { ...estado, carga: true, errorLista: false, errorListaClave: null };
    pintar();
    const signal = controlador.signal;
    try {
      const pagina = await cliente.listar(
        {
          limit: 6,
          ...(cursores[indicePagina] ? { cursor: cursores[indicePagina] } : {}),
          ...(relacionSeleccionada
            ? { relacion_ref: relacionSeleccionada }
            : {}),
        },
        { signal },
      );
      if (!activaAhora() || signal.aborted) return;
      estado = {
        ...estado,
        carga: false,
        items: pagina.items,
        errorLista: false,
        errorListaClave: null,
        mensaje: conservarMensaje ? estado.mensaje : consultaExplicita
          ? (pagina.items.length ? "borradores_propios_consulta_registrados" : "borradores_propios_consulta_registrados_vacia")
          : "borradores_propios_listado",
        tono: conservarMensaje ? estado.tono : "informacion",
      };
      siguienteCursor = pagina.siguiente_cursor;
      if (consultaExplicita) {
        try { anunciar(tBorradores(estado.mensaje), "informacion"); } catch {}
      }
    } catch (error) {
      if (!activaAhora() || signal.aborted) return;
      const claveError = conservarMensaje
        ? (error?.codigo === "acceso_denegado" ? "borradores_propios_creado_listado_denegado" : "borradores_propios_creado_listado_no_actualizado")
        : errorClave(error);
      const denegada = error?.codigo === "autenticacion_requerida" || error?.codigo === "acceso_denegado";
      if (denegada) purgarLecturasDenegadas();
      estado = {
        ...estado,
        carga: false,
        errorLista: true,
        errorListaClave: claveError,
        items: [],
      };
      mensaje(claveError, conservarMensaje ? "aviso" : "error");
    } finally {
      if (controlador?.signal === signal) controlador = null;
      if (activaAhora()) pintar();
    }
  }
  async function enviar(evento) {
    const form = evento.target?.closest?.("[data-dietas-borrador-form]");
    if (!form || !activaAhora()) return;
    evento.preventDefault();
    if (!conectada || controlador || estadoRelaciones === "no_disponible" || (relaciones.length > 1 && !relacionSeleccionada)) return;
    if (form.querySelector("[data-dietas-pais]")?.value !== "ES") {
      mensaje("comision_pais_sin_calculo", "aviso"); pintar(); return;
    }
    if (!form.checkValidity?.()) {
      form.reportValidity?.();
      return;
    }
    const datos = new FormData(form);
    const base = {
      fecha_inicio: datos.get("fecha_inicio"),
      fecha_fin: datos.get("fecha_fin"),
      motivo: String(datos.get("motivo") || "").trim(),
      ...(datos.get("relacion_ref")
        ? { relacion_ref: String(datos.get("relacion_ref")) }
        : {}),
      hora_inicio: String(datos.get("hora_inicio")||""),
      hora_fin: String(datos.get("hora_fin")||""),
      codigos_ruta: codigosRuta(form, datos),
    };
    if (base.relacion_ref && !relaciones.includes(base.relacion_ref)) {
      mensaje("borradores_propios_error_relacion", "aviso");
      pintar();
      return;
    }
    if (base.fecha_fin < base.fecha_inicio ||
        (base.fecha_fin === base.fecha_inicio && base.hora_fin <= base.hora_inicio)) {
      mensaje("borradores_propios_fechas_invalidas", "aviso");
      pintar();
      return;
    }
    if (!rutaValida(base.codigos_ruta)) { mensaje("borradores_propios_paradas_distintas","aviso"); pintar(); return; }
    if (calculadorRuta && calculoCabeceraFirma !== JSON.stringify(base.codigos_ruta)) {
      mensaje("ruta_calcular_antes_guardar", "aviso"); pintar(); return;
    }
    if (edicion) {
      let otros;
      try { otros = otrosDesdeFormulario(form); }
      catch { mensaje("comision_otros_error", "aviso"); pintar(); return; }
      if (otros.some((linea) => linea.importe_centimos < 1 || linea.concepto.length < 3)) {
        mensaje("comision_otros_error", "aviso"); pintar(); return;
      }
      let transporte;
      try { transporte = rutasDesdeFormulario(form); }
      catch { mensaje("comision_ajuste_error", "aviso"); pintar(); return; }
      let tramosAceptados;
      try { tramosAceptados = tramosAceptadosDesdeFormulario(form); }
      catch { mensaje("comision_aceptacion_error", "aviso"); pintar(); return; }
      const contenido = JSON.stringify([edicion.comision.referencia, edicion.recibo.version, base, otros, transporte, tramosAceptados]);
      const clave = intentoEdicion?.contenido === contenido ? intentoEdicion.clave : generarClaveIdempotencia();
      if (!clave) { mensaje("borradores_propios_error", "error"); pintar(); return; }
      intentoEdicion = { contenido, clave };
      controlador = new AbortController();
      const signal = controlador.signal;
      mensaje("comision_actividad"); pintar();
      try {
        const item = await cliente.editar(edicion.comision.referencia, {
          ...base, relacion_ref: edicion.comision.relacion_ref,
          otros, ...transporte, tramos_aceptados: tramosAceptados,
          version_tarifa_aceptada: edicion.comision.calculo.version_tarifa,
          version_esperada: edicion.recibo.version, clave_idempotencia: clave,
        }, { signal });
        if (!activaAhora() || signal.aborted) return;
        edicion = null;
        intentoEdicion = null;
        estado = { ...estado, detalle: item, detalleOrigen: "put" };
        void consultarAsignacion(item);
        formularioVisible = false;
        mensaje("comision_edicion_guardada", "exito");
        cursores = [undefined]; indicePagina = 0; siguienteCursor = undefined;
        await cargar(true);
      } catch (error) {
        if (!activaAhora() || signal.aborted) return;
        if (error?.codigo === "conflicto_version") intentoEdicion = null;
        if (["autenticacion_requerida", "acceso_denegado"].includes(error?.codigo)) purgarLecturasDenegadas();
        mensaje(error?.codigo === "conflicto_version" ? "comision_conflicto_version" : errorClave(error), "error");
      } finally {
        if (controlador?.signal === signal) controlador = null;
        if (activaAhora()) { pintar(); if (!edicion) enfocarRecibo(); }
      }
      return;
    }
    const contenido = claveContenido(base);
    const operacion = operaciones.get(contenido);
    if (operacion?.item) {
      ultimoAlta = { contenido, item: operacion.item };
      estado = { ...estado, detalle: operacion.item, detalleOrigen: "post" };
      mensaje("borradores_propios_ya_registrado", "exito");
      pintar();
      enfocarRecibo();
      return;
    }
    const clave = operacion?.clave || generarClaveIdempotencia();
    if (typeof clave !== "string" || !clave) {
      mensaje("borradores_propios_error", "error");
      pintar();
      return;
    }
    operaciones.set(contenido, {
      clave,
      incierta: operacion?.incierta === true,
      confirmada: operacion?.confirmada === true,
    });
    const solicitud = { clave_idempotencia: clave, ...base };
    controlador = new AbortController();
    const signal = controlador.signal;
    let altaConfirmada = false;
    mensaje("borradores_propios_enviando");
    pintar();
    try {
      const item = await cliente.crear(solicitud, { signal });
      if (!activaAhora() || signal.aborted) return;
      operaciones.set(contenido, { clave, item, confirmada: true });
      ultimoAlta = { contenido, item };
      altaConfirmada = true;
      const resumenLocal = formularioPersistente?.querySelector?.("[data-dietas-borrador-preparacion]");
      if (resumenLocal) resumenLocal.hidden = true;
      estado = {
        ...estado,
        detalle: item,
        detalleOrigen: "post",
        errorLista: false,
      };
      void consultarAsignacion(item);
      mensaje(
        item.recibo.repeticion
          ? "borradores_propios_repeticion"
          : "borradores_propios_creado",
        "exito",
      );
      cursores = [undefined];
      indicePagina = 0;
      siguienteCursor = undefined;
      await cargar(true);
    } catch (error) {
      if (!activaAhora() || signal.aborted) return;
      // Un 403 posterior no aclara si un intento previo de esta intención
      // quedó registrado. Conservar su clave hasta confirmar o desmontar.
      if (error?.resultadoIndeterminado)
        operaciones.set(contenido, { clave, incierta: true, confirmada: operacion?.confirmada === true });
      else if (!operacion?.incierta && !operacion?.confirmada) operaciones.delete(contenido);
      if (["autenticacion_requerida", "acceso_denegado"].includes(error?.codigo)) {
        purgarLecturasDenegadas();
        estado = { ...estado, errorLista: true, errorListaClave: errorClave(error) };
      }
      mensaje(errorClave(error, "crear"), "error");
    } finally {
      if (controlador?.signal === signal) controlador = null;
      if (activaAhora()) {
        pintar();
        if (altaConfirmada) enfocarRecibo();
      }
    }
  }
  function enfocarRecibo() {
    const reciboActual = fichaPersistente?.querySelector?.("[data-dietas-borrador-recibo]");
    if (!reciboActual) return;
    reciboActual.setAttribute("tabindex", "-1");
    enfocar(reciboActual);
  }
  function invalidarPreparacion(evento) {
    if (!evento.target?.closest?.("[data-dietas-borrador-form]")) return;
    const resumen = formularioPersistente?.querySelector?.("[data-dietas-borrador-preparacion]");
    if (resumen) resumen.hidden = true;
    if (["origen_codigo", "destino_codigo", "parada_codigo"].includes(evento.target?.name)) {
      const form = formularioPersistente;
      const datos = new FormData(form);
      const codigos = codigosRuta(form, datos);
      mapaVista?.establecerCodigos(rutaValida(codigos) ? codigos : []);
      calculoCabeceraFirma = null;
    }
    if (["ruta_origen_codigo", "ruta_destino_codigo", "ruta_parada_codigo"].includes(evento.target?.name)) {
      const fila = evento.target.closest("[data-dietas-ruta-linea]");
      const estadoRuta = fila?.querySelector("[data-dietas-ruta-estado]");
      if (estadoRuta) estadoRuta.textContent = tBorradores("comision_ruta_pendiente");
    }
  }
  function abrirEdicion() {
    const item = estado.detalle;
    if (!item || item.comision.estado !== "borrador" || estadoRelaciones === "no_disponible" ||
        !asignacionVerificada() || !item.comision.calculo || typeof cliente?.editar !== "function") return;
    edicion = item;
    intentoEdicion = null;
    formularioVisible = true;
    const form = formularioPersistente;
    const valores = {
      fecha_inicio: item.comision.fecha_inicio,
      fecha_fin: item.comision.fecha_fin,
      hora_inicio: item.comision.calculo?.hora_inicio || "",
      hora_fin: item.comision.calculo?.hora_fin || "",
      motivo: item.comision.motivo,
      origen_codigo: item.comision.codigos_ruta[0] || "",
      destino_codigo: item.comision.codigos_ruta.at(-1) || "",
      relacion_ref: item.comision.relacion_ref,
    };
    form.querySelectorAll("input").concat(form.querySelectorAll("select")).forEach((control) => {
      if (Object.hasOwn(valores, control.name)) control.value = valores[control.name];
    });
    pintarParadas(form, item.comision.codigos_ruta.slice(1, -1));
    const listaOtros = form.querySelector("[data-dietas-otros-lista]");
    const lineas = item.comision.documento?.lineas?.filter((linea) => ["otro_medio", "otro_gasto"].includes(linea.tipo)) || [];
    listaOtros?.replaceChildren(...lineas.map((linea) => nuevaLineaOtro({
      ...linea, importe: (linea.importe_centimos / 100).toFixed(2).replace(".", ","),
    })));
    const vehiculo = form.querySelector("[data-dietas-vehiculo-propio]");
    if (vehiculo) vehiculo.value = item.comision.vehiculo_propio === true ? "si" :
      item.comision.vehiculo_propio === false ? "no" : "";
    const listaRutas = form.querySelector("[data-dietas-rutas-lista]");
    listaRutas?.replaceChildren(...(item.comision.rutas || []).map(nuevaRutaVehiculo));
    rutasCalculadas.clear();
    pintarAceptacion(item);
    pintar();
    enfocar(form.querySelector("input"));
  }
  async function ejecutarAccionComision(accion) {
    const item = estado.detalle;
    if (!item || item.comision.estado !== "borrador" || estadoRelaciones === "no_disponible" || typeof cliente?.[accion] !== "function" ||
        (accion === "enviar" && (!asignacionVerificada() || !item.comision.documento)) || controlador) return;
    const confirmada = await confirmarOperacion(tBorradores(accion === "enviar" ? "comision_confirmar_enviar" : "comision_confirmar_eliminar"));
    if (!confirmada || !activaAhora()) return;
    const intento = `${accion}:${item.comision.referencia}:${item.recibo.version}`;
    const clave = intentosAccion.get(intento) || generarClaveIdempotencia();
    if (!clave) { mensaje("borradores_propios_error", "error"); pintar(); return; }
    intentosAccion.set(intento, clave);
    controlador = new AbortController();
    const signal = controlador.signal;
    mensaje("comision_actividad"); pintar();
    try {
      const resultado = await cliente[accion](item.comision.referencia, {
        clave_idempotencia: clave,
        version_esperada: item.recibo.version,
        relacion_ref: item.comision.relacion_ref,
      }, { signal });
      if (!activaAhora() || signal.aborted) return;
      intentosAccion.delete(intento);
      estado = { ...estado, detalle: resultado, detalleOrigen: "post" };
      void consultarAsignacion(resultado);
      mensaje(accion === "enviar" ? "comision_envio_confirmado" : "comision_eliminacion_confirmada", "exito");
      cursores = [undefined]; indicePagina = 0; siguienteCursor = undefined;
      await cargar(true);
    } catch (error) {
      if (!activaAhora() || signal.aborted) return;
      if (!error?.resultadoIndeterminado) intentosAccion.delete(intento);
      if (["autenticacion_requerida", "acceso_denegado"].includes(error?.codigo)) purgarLecturasDenegadas();
      mensaje(error?.codigo === "conflicto_version" ? "comision_conflicto_version" : errorClave(error), "error");
    } finally {
      if (controlador?.signal === signal) controlador = null;
      if (activaAhora()) pintar();
    }
  }
  async function clic(evento) {
    const anadirRuta = evento.target?.closest?.("[data-dietas-ruta-anadir]");
    if (anadirRuta && edicion && !controlador && !anadirRuta.disabled) {
      const lista = formularioPersistente.querySelector("[data-dietas-rutas-lista]");
      if (lista.querySelectorAll("[data-dietas-ruta-linea]").length < 8) {
        const fila = nuevaRutaVehiculo(); lista.append(fila); renumerarRutas(formularioPersistente);
        enfocar(fila.querySelector("select")); pintar();
      }
      return;
    }
    const quitarRuta = evento.target?.closest?.("[data-dietas-ruta-quitar]");
    if (quitarRuta && edicion && !controlador) {
      quitarRuta.closest("[data-dietas-ruta-linea]")?.remove();
      renumerarRutas(formularioPersistente); pintar(); return;
    }
    const accionParadaRuta = ["dietasRutaParadaAnadir", "dietasRutaParadaSubir", "dietasRutaParadaBajar", "dietasRutaParadaQuitar"]
      .map((atributo) => [atributo, evento.target?.closest?.(`[data-${atributo.replace(/[A-Z]/gu, (letra) => `-${letra.toLowerCase()}`)}]`)])
      .find(([, boton]) => boton);
    if (accionParadaRuta && edicion && !controlador && !accionParadaRuta[1].disabled) {
      const fila = accionParadaRuta[1].closest("[data-dietas-ruta-linea]");
      const valores = fila.querySelectorAll("select").filter((selector) => selector.name === "ruta_parada_codigo").map((selector) => selector.value || "");
      const [accion, boton] = accionParadaRuta;
      const indice = Number(boton.dataset[accion]);
      if (accion === "dietasRutaParadaAnadir" && valores.length < 10) valores.push("");
      else if (accion === "dietasRutaParadaSubir" && indice > 0 && indice < valores.length) [valores[indice - 1], valores[indice]] = [valores[indice], valores[indice - 1]];
      else if (accion === "dietasRutaParadaBajar" && indice >= 0 && indice < valores.length - 1) [valores[indice], valores[indice + 1]] = [valores[indice + 1], valores[indice]];
      else if (accion === "dietasRutaParadaQuitar" && indice >= 0 && indice < valores.length) valores.splice(indice, 1);
      else return;
      pintarParadasRuta(fila, valores);
      fila.querySelector("[data-dietas-ruta-estado]").textContent = tBorradores("comision_ruta_pendiente");
      return;
    }
    const calcularRutaVehiculo = evento.target?.closest?.("[data-dietas-ruta-calcular]");
    if (calcularRutaVehiculo && edicion && calculadorRuta && !controlador) {
      const fila = calcularRutaVehiculo.closest("[data-dietas-ruta-linea]");
      const codigos = codigosRutaVehiculo(fila);
      if (!rutaValida(codigos)) { mensaje("borradores_propios_paradas_distintas", "aviso"); pintar(); return; }
      try {
        const mapa = mapaVista || await montajeMapa;
        if (!mapa) throw new Error("mapa no disponible");
        mapa.establecerCodigos(codigos);
        const calculo = await mapa.calcular();
        if (calculo) {
          rutasCalculadas.set(JSON.stringify(codigos), calculo);
          fila.querySelector("[data-dietas-ruta-estado]").textContent = tBorradores("comision_ruta_calculada");
        }
      } catch { mensaje("ruta_error_servicio", "error"); pintar(); }
      return;
    }
    const calcular = evento.target?.closest?.("[data-dietas-calcular-ruta]");
    if (calcular && !calcular.disabled && calculadorRuta && !controlador && activaAhora()) {
      const form = calcular.closest("[data-dietas-borrador-form]");
      const codigos = codigosRuta(form, new FormData(form));
      if (!rutaValida(codigos)) { mensaje("borradores_propios_paradas_distintas", "aviso"); pintar(); return; }
      try {
        const mapa = mapaVista || await montajeMapa;
        if (!mapa) throw new Error("mapa no disponible");
        mapa.establecerCodigos(codigos);
        const calculo = await mapa.calcular();
        if (calculo) calculoCabeceraFirma = JSON.stringify(codigos);
      } catch { mensaje("ruta_error_servicio", "error"); pintar(); }
      return;
    }
    const anadirOtro = evento.target?.closest?.("[data-dietas-otro-anadir]");
    if (anadirOtro && edicion && !controlador && !anadirOtro.disabled) {
      const lista = formularioPersistente.querySelector("[data-dietas-otros-lista]");
      if (lista.querySelectorAll("[data-dietas-otro-linea]").length < 32) {
        const fila = nuevaLineaOtro(); lista.append(fila); enfocar(fila.querySelector("input")); pintar();
      }
      return;
    }
    const quitarOtro = evento.target?.closest?.("[data-dietas-otro-quitar]");
    if (quitarOtro && edicion && !controlador) { quitarOtro.closest("[data-dietas-otro-linea]")?.remove(); pintar(); return; }
    if (evento.target?.closest?.("[data-dietas-borrador-editar]") && !controlador) { abrirEdicion(); return; }
    if (evento.target?.closest?.("[data-dietas-borrador-eliminar]")) { await ejecutarAccionComision("eliminar"); return; }
    if (evento.target?.closest?.("[data-dietas-borrador-enviar]")) { await ejecutarAccionComision("enviar"); return; }
    const accionParada = ["dietasParadaAnadir", "dietasParadaSubir", "dietasParadaBajar", "dietasParadaQuitar"]
      .map((atributo) => [atributo, evento.target?.closest?.(`[data-${atributo.replace(/[A-Z]/gu, (letra) => `-${letra.toLowerCase()}`)}]`)])
      .find(([, boton]) => boton);
    if (accionParada && activaAhora() && conectada && !controlador && !accionParada[1].disabled) {
      const form = accionParada[1].closest("[data-dietas-borrador-form]");
      const valores = valoresParadas(form);
      const [accion, boton] = accionParada;
      const indice = Number(boton.dataset[accion]);
      let focoIndice = -1;
      if (accion === "dietasParadaAnadir" && valores.length < MAXIMO_LOCALIDADES - 2) {
        valores.push(""); focoIndice = valores.length - 1;
      } else if (accion === "dietasParadaSubir" && indice > 0 && indice < valores.length) {
        [valores[indice - 1], valores[indice]] = [valores[indice], valores[indice - 1]]; focoIndice = indice - 1;
      } else if (accion === "dietasParadaBajar" && indice >= 0 && indice < valores.length - 1) {
        [valores[indice], valores[indice + 1]] = [valores[indice + 1], valores[indice]]; focoIndice = indice + 1;
      } else if (accion === "dietasParadaQuitar" && indice >= 0 && indice < valores.length) {
        valores.splice(indice, 1); focoIndice = valores.length ? Math.min(indice, valores.length - 1) : 0;
      } else return;
      pintarParadas(form, valores, focoIndice);
      invalidarPreparacion({ target: form });
      return;
    }
    const revisar = evento.target?.closest?.("[data-dietas-borrador-revisar]");
    if (revisar && !revisar.disabled && activaAhora() && !controlador && conectada) {
      const form = revisar.closest("[data-dietas-borrador-form]");
      if (!form?.checkValidity?.()) { form?.reportValidity?.(); return; }
      const datos = new FormData(form);
      const inicio = String(datos.get("fecha_inicio") || "");
      const fin = String(datos.get("fecha_fin") || "");
      const horaInicio = String(datos.get("hora_inicio") || "");
      const horaFin = String(datos.get("hora_fin") || "");
      const codigos = codigosRuta(form, datos);
      if (fin < inicio || (fin === inicio && horaFin <= horaInicio)) {
        mensaje("borradores_propios_fechas_invalidas", "aviso"); pintar(); enfocar(avisoPersistente); return;
      }
      if (!rutaValida(codigos)) {
        mensaje("borradores_propios_paradas_distintas", "aviso"); pintar(); enfocar(avisoPersistente); return;
      }
      const resumen = form.querySelector("[data-dietas-borrador-preparacion]");
      resumen.replaceChildren(
        nodo(documento, "h4", tBorradores("borradores_propios_preparacion_titulo")),
        nodo(documento, "p", `${inicio} ${horaInicio} → ${fin} ${horaFin}`),
        nodo(documento, "p", String(datos.get("motivo") || "").trim()),
        nodo(documento, "p", rutaLegible(codigos, tBorradores, nombresRuta)),
        nodo(documento, "p", `${tBorradores("borradores_propios_pais")}: ${tBorradores("borradores_propios_pais_espana")}`),
      );
      resumen.hidden = false;
      enfocar(resumen);
      return;
    }
    const consultar = evento.target?.closest?.("[data-dietas-borrador-consultar-registrados]");
    if (consultar && activaAhora() && !controlador && !consultar.disabled) {
      cursores = [undefined];
      indicePagina = 0;
      siguienteCursor = undefined;
      mensaje("borradores_propios_consultando_registrados");
      await cargar(false, true);
      if (activaAhora() && focoSigueEnConsulta(consultar))
        enfocar(listaPersistente?.querySelector?.("[data-dietas-borrador-consultar-registrados]"));
      return;
    }
    const pagina = evento.target?.closest?.("[data-dietas-borrador-pagina]");
    if (pagina && activaAhora() && !controlador) {
      if (pagina.dataset.dietasBorradorPagina === "siguiente" && siguienteCursor) {
        cursores = [...cursores.slice(0, indicePagina + 1), siguienteCursor];
        indicePagina += 1;
        await cargar();
        if (activaAhora() && focoSigueEnConsulta(pagina)) enfocar(listaPersistente?.querySelector?.("[data-dietas-borradores-listado]"));
      } else if (pagina.dataset.dietasBorradorPagina === "anterior" && indicePagina > 0) {
        indicePagina -= 1;
        await cargar();
        if (activaAhora() && focoSigueEnConsulta(pagina)) enfocar(listaPersistente?.querySelector?.("[data-dietas-borradores-listado]"));
      }
      return;
    }
    const boton = evento.target?.closest?.("[data-dietas-borrador-detalle]");
    if (!boton || controlador || !activaAhora()) return;
    const referencia = boton.dataset.dietasBorradorDetalle;
    controlador = new AbortController();
    const signal = controlador.signal;
    let detalleActualizado = false;
    mensaje("borradores_propios_cargando");
    pintar();
    try {
      const item = await cliente.obtener(referencia, {
        signal,
        relacion_ref: boton.dataset.dietasBorradorRelacion || undefined,
      });
      if (!activaAhora() || signal.aborted) return;
      edicion = null;
      intentoEdicion = null;
      formularioVisible = false;
      calculoCabeceraFirma = null;
      rutasCalculadas.clear();
      estado = { ...estado, detalle: item, detalleOrigen: "get" };
      void consultarAsignacion(item);
      detalleActualizado = true;
      mensaje("borradores_propios_detalle");
    } catch (error) {
      if (!activaAhora() || signal.aborted) return;
      const denegada = ["autenticacion_requerida", "acceso_denegado"].includes(error?.codigo);
      if (denegada) {
        purgarLecturasDenegadas();
        estado = { ...estado, errorLista: true, errorListaClave: errorClave(error) };
      }
      mensaje(denegada ? errorClave(error) : estado.detalle
        ? (error?.codigo === "acceso_denegado" ? "borradores_propios_detalle_denegado" : "borradores_propios_detalle_no_actualizado")
        : errorClave(error), "error");
    } finally {
      if (controlador?.signal === signal) controlador = null;
      if (activaAhora()) {
        pintar();
        if (detalleActualizado) enfocar(fichaPersistente?.querySelector?.("[data-dietas-borrador-ficha]"));
        else if (!signal.aborted) enfocar(avisoPersistente);
      }
    }
  }
  function cambiarRelacion(evento) {
    const selector = evento.target?.closest?.('[name="relacion_ref"]');
    if (!selector || relaciones.length < 2 || !activaAhora()) return;
    if (controlador) {
      selector.value = relacionSeleccionada || "";
      return;
    }
    relacionSeleccionada = relaciones.includes(selector.value)
      ? selector.value
      : undefined;
    cursores = [undefined];
    indicePagina = 0;
    siguienteCursor = undefined;
    edicion = null;
    intentoEdicion = null;
    formularioVisible = false;
    asignacion = null;
    controladorAsignacion?.abort();
    estado = { ...estado, items: [], detalle: null, detalleOrigen: null };
    cargar();
  }
  function cambiarFormulario(evento) {
    invalidarPreparacion(evento);
    cambiarRelacion(evento);
    if (evento.target?.name === "vehiculo_propio" || evento.target?.name === "pais") {
      if (evento.target?.name === "pais" && evento.target.value !== "ES") {
        calculoCabeceraFirma = null;
        mapaVista?.establecerCodigos([]);
      }
      pintar();
    }
    if (evento.target?.name === "nivel_detalle" && ["bajo", "medio", "alto"].includes(evento.target.value)) {
      nivelDetalle = evento.target.value;
      pintar();
    }
  }
  raiz.addEventListener("submit", enviar);
  raiz.addEventListener("click", clic);
  raiz.addEventListener("change", cambiarFormulario);
  raiz.addEventListener("input", invalidarPreparacion);
  pintar();
  cargar();
  function abrirFormulario() {
    if (!activaAhora()) return false;
    formularioVisible = true;
    pintar();
    enfocar(formularioPersistente.querySelector("input"));
    return true;
  }
  function cerrarFormulario() {
    if (!activaAhora()) return false;
    formularioVisible = false;
    pintar();
    return true;
  }
  return Object.freeze({ desmontar, recargar: cargar, abrirFormulario, cerrarFormulario });
}
