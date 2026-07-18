import {
  ErrorAPIBorradores,
  crearClienteBorradores,
  generarClaveIdempotencia,
} from "./portal-borradores-api.js";
import { ESQUEMAS_BORRADORES } from "./portal-borradores-contrato.js";
import { crearRenderizadorBorradores } from "./portal-borradores-vista.js";

export const PROVEEDOR_BEARER_BORRADORES = "__VEC_PORTAL_EMPLEADO_OBTENER_BEARER_BORRADORES__";

const FASE_INICIAL = "inicial";
const FASE_CARGANDO = "cargando";
const FASE_LISTA = "lista";
const FASE_ERROR = "error";

function copiar(valor) {
  return valor === undefined ? undefined : JSON.parse(JSON.stringify(valor));
}

function errorSeguro(error, mensajeDefecto) {
  if (error instanceof ErrorAPIBorradores) {
    return {
      mensaje: error.message,
      codigo: error.codigo,
      correlacion: error.correlacion,
      estadoHTTP: error.estado,
      tipoConflicto: error.tipoConflicto,
      conservarCambiosLocales: error.conservarCambiosLocales,
    };
  }
  return {
    mensaje: mensajeDefecto,
    codigo: "error_interfaz_borradores",
    correlacion: null,
    estadoHTTP: 0,
    tipoConflicto: null,
    conservarCambiosLocales: false,
  };
}

function editorNuevo(opciones) {
  return {
    plantilla_indice: 0,
    motivo_indice: 0,
    codigo_version_publica: "",
    identificador_publico: "",
    expediente_ref: "",
    contenido_editable: {
      tipo: opciones.tipos[0]?.clave || "",
      categorias: opciones.categorias[0] ? [opciones.categorias[0].clave] : [],
      titulo: "",
      resumen: "",
      descripcion: "",
      plazos: [{
        referencia: "",
        tipo: "",
        titulo: "",
        descripcion: "",
        abre_en: "",
        cierra_en: "",
      }],
      requisitos: [],
      ayuda: [],
    },
  };
}

function editorDesdeDetalle(detalle) {
  return {
    motivo_indice: 0,
    contenido_editable: copiar(detalle.contenido_editable),
  };
}

function asignarRutaEditor(editor, ruta, valor) {
  const partes = String(ruta).split(".");
  if (partes.length < 1 || partes.length > 5
    || partes.some((parte) => !/^(?:[a-z_]+|[0-9]+)$/.test(parte)
      || parte === "__proto__" || parte === "constructor" || parte === "prototype")) {
    throw new TypeError("ruta de editor no válida");
  }
  let actual = editor;
  for (let indice = 0; indice < partes.length - 1; indice += 1) {
    if (actual === null || typeof actual !== "object" || !Object.hasOwn(actual, partes[indice])) {
      throw new TypeError("ruta de editor no disponible");
    }
    actual = actual[partes[indice]];
  }
  const ultimo = partes.at(-1);
  if (actual === null || typeof actual !== "object" || !Object.hasOwn(actual, ultimo)) {
    throw new TypeError("campo de editor no disponible");
  }
  actual[ultimo] = valor;
}

export function crearSuperficieBorradoresPortal({
  escaparHTML,
  anunciar,
  alCambiar,
  resolverProveedorBearer,
  confirmar = (mensaje) => globalThis.confirm(mensaje),
  crearClienteImpl = crearClienteBorradores,
  generarClaveImpl = generarClaveIdempotencia,
} = {}) {
  if ([escaparHTML, anunciar, alCambiar, resolverProveedorBearer, confirmar,
    crearClienteImpl, generarClaveImpl].some((dependencia) => typeof dependencia !== "function")) {
    throw new TypeError("dependencias de la superficie de borradores no válidas");
  }

  const estado = {
    faseLista: FASE_INICIAL,
    opciones: null,
    lista: null,
    errorLista: null,
    filtro: { texto: "", categoria: "" },
    cursores: [undefined],
    pagina: 0,
    modoEditor: "ninguno",
    faseEditor: "vacio",
    referenciaSeleccionada: "",
    detalle: null,
    editor: null,
    sucio: false,
    guardando: false,
    errorEditor: null,
    recibo: null,
    claveIdempotencia: "",
    conflictoRemoto: null,
    confirmarReaplicacion: false,
  };

  let cliente = null;
  let controladorCarga = null;
  let controladorDetalle = null;

  function notificar() {
    alCambiar();
  }

  function clienteAutorizado() {
    const proveedor = resolverProveedorBearer();
    if (typeof proveedor !== "function") {
      throw new ErrorAPIBorradores(
        "El proveedor de credencial para borradores no está configurado.",
        0,
        undefined,
        { codigo: "proveedor_bearer_no_configurado" },
      );
    }
    if (cliente === null) {
      cliente = crearClienteImpl({
        obtenerBearer: (signal) => {
          const proveedorActual = resolverProveedorBearer();
          if (typeof proveedorActual !== "function") {
            throw new ErrorAPIBorradores(
              "El proveedor de credencial para borradores dejó de estar disponible.",
              0,
              undefined,
              { codigo: "proveedor_bearer_no_configurado" },
            );
          }
          return proveedorActual(signal);
        },
      });
    }
    return cliente;
  }

  function opcionesLista() {
    const opciones = { limite: 40 };
    const cursor = estado.cursores[estado.pagina];
    if (cursor !== undefined) opciones.cursor = cursor;
    if (estado.filtro.texto) opciones.texto = estado.filtro.texto;
    if (estado.filtro.categoria) opciones.categoria = estado.filtro.categoria;
    return opciones;
  }

  async function cargarDetalle(referencia, { descartarSucio = false } = {}) {
    if (!estado.opciones || !referencia) return false;
    if (estado.sucio && !descartarSucio
      && !confirmar("Hay cambios locales sin guardar. ¿Desea descartarlos y abrir otro borrador?")) {
      return false;
    }
    controladorDetalle?.abort();
    const controlador = new AbortController();
    controladorDetalle = controlador;
    estado.faseEditor = "cargando";
    estado.errorEditor = null;
    estado.recibo = null;
    notificar();
    try {
      const detalle = await clienteAutorizado().obtenerDetalle(
        referencia,
        estado.opciones.limites,
        { signal: controlador.signal },
      );
      if (controlador !== controladorDetalle) return false;
      estado.modoEditor = "actualizar";
      estado.faseEditor = "listo";
      estado.referenciaSeleccionada = referencia;
      estado.detalle = detalle;
      estado.editor = editorDesdeDetalle(detalle);
      estado.sucio = false;
      estado.claveIdempotencia = "";
      estado.conflictoRemoto = null;
      estado.confirmarReaplicacion = false;
      notificar();
      anunciar("Borrador abierto para edición");
      return true;
    } catch (error) {
      if (controlador.signal.aborted || controlador !== controladorDetalle) return false;
      estado.faseEditor = "error";
      estado.errorEditor = errorSeguro(error, "No se pudo cargar el borrador seleccionado.");
      notificar();
      anunciar("No se pudo cargar el borrador seleccionado");
      return false;
    }
  }

  async function cargarLista({ conservarEditor = true, mantenerVisible = false } = {}) {
    const controlador = controladorCarga;
    if (!mantenerVisible) estado.faseLista = FASE_CARGANDO;
    estado.errorLista = null;
    notificar();
    try {
      const lista = await clienteAutorizado().listar({
        ...opcionesLista(), signal: controlador?.signal,
      });
      if (controlador !== controladorCarga) return false;
      estado.lista = lista;
      estado.faseLista = FASE_LISTA;
      notificar();
      if (!conservarEditor && lista.elementos[0]) {
        await cargarDetalle(lista.elementos[0].referencia_estado.referencia, { descartarSucio: true });
      }
      return true;
    } catch (error) {
      if (controlador?.signal.aborted || controlador !== controladorCarga) return false;
      estado.errorLista = errorSeguro(error, "El servicio de borradores no está disponible.");
      estado.faseLista = mantenerVisible && estado.lista ? FASE_LISTA : FASE_ERROR;
      notificar();
      return false;
    }
  }

  async function cargarSuperficie() {
    controladorCarga?.abort();
    const controlador = new AbortController();
    controladorCarga = controlador;
    estado.faseLista = FASE_CARGANDO;
    estado.errorLista = null;
    notificar();
    try {
      const api = clienteAutorizado();
      const [opciones, lista] = await Promise.all([
        api.obtenerOpciones({ signal: controlador.signal }),
        api.listar({ limite: 40, signal: controlador.signal }),
      ]);
      if (controlador !== controladorCarga) return false;
      estado.opciones = opciones;
      estado.lista = lista;
      estado.faseLista = FASE_LISTA;
      estado.cursores = [undefined];
      estado.pagina = 0;
      notificar();
      if (estado.modoEditor === "ninguno" && lista.elementos[0]) {
        await cargarDetalle(lista.elementos[0].referencia_estado.referencia, { descartarSucio: true });
      }
      return true;
    } catch (error) {
      if (controlador.signal.aborted || controlador !== controladorCarga) return false;
      estado.faseLista = FASE_ERROR;
      estado.errorLista = errorSeguro(error, "El servicio de borradores no está disponible.");
      notificar();
      anunciar("Servicio de borradores no disponible");
      return false;
    }
  }

  function activar() {
    if (estado.faseLista === FASE_INICIAL) return cargarSuperficie();
    return Promise.resolve(true);
  }

  function iniciarNuevo() {
    if (!estado.opciones) return false;
    if (estado.sucio
      && !confirmar("Hay cambios locales sin guardar. ¿Desea descartarlos y crear otro borrador?")) {
      return false;
    }
    if (estado.opciones.capacidades.crear !== true) {
      estado.errorEditor = {
        mensaje: "La sesión no dispone de capacidad para crear borradores.",
        codigo: "capacidad_crear_no_concedida",
        correlacion: null,
        estadoHTTP: 403,
        tipoConflicto: null,
        conservarCambiosLocales: false,
      };
      notificar();
      return false;
    }
    estado.modoEditor = "crear";
    estado.faseEditor = "listo";
    estado.referenciaSeleccionada = "";
    estado.detalle = null;
    estado.editor = editorNuevo(estado.opciones);
    estado.sucio = false;
    estado.guardando = false;
    estado.errorEditor = null;
    estado.recibo = null;
    estado.claveIdempotencia = "";
    estado.conflictoRemoto = null;
    estado.confirmarReaplicacion = false;
    notificar();
    anunciar("Editor de nuevo borrador preparado");
    return true;
  }

  function motivoSeleccionado() {
    return estado.opciones?.motivos[Number(estado.editor?.motivo_indice)] || null;
  }

  function plantillaSeleccionada() {
    return estado.opciones?.plantillas[Number(estado.editor?.plantilla_indice)] || null;
  }

  function puedeEditar() {
    if (estado.modoEditor === "crear") return estado.opciones?.capacidades?.crear === true;
    return estado.modoEditor === "actualizar" && estado.detalle?.capacidades?.actualizar === true;
  }

  function contenidoParaSolicitud() {
    const contenido = copiar(estado.editor.contenido_editable);
    contenido.requisitos = contenido.requisitos.map((item) => ({
      ...item, orden: Number(item.orden), obligatorio: item.obligatorio === true,
    }));
    contenido.ayuda = contenido.ayuda.map((item) => ({ ...item, orden: Number(item.orden) }));
    return contenido;
  }

  function construirSolicitud() {
    const motivo = motivoSeleccionado();
    if (!motivo) throw new ErrorAPIBorradores("Seleccione un motivo gobernado para guardar.");
    const base = {
      contenido_editable: contenidoParaSolicitud(),
      motivo_ref: motivo.motivo_ref,
      motivo_version: motivo.version,
      motivo_huella_sha256: motivo.huella_sha256,
    };
    if (estado.modoEditor === "actualizar") {
      return { esquema: ESQUEMAS_BORRADORES.actualizar, ...base };
    }
    const plantilla = plantillaSeleccionada();
    if (!plantilla) throw new ErrorAPIBorradores("Seleccione una plantilla gobernada para crear.");
    return {
      esquema: ESQUEMAS_BORRADORES.crear,
      plantilla_ref: plantilla.plantilla_ref,
      plantilla_version: plantilla.version,
      plantilla_huella_sha256: plantilla.huella_sha256,
      codigo_version_publica: estado.editor.codigo_version_publica,
      identificador_publico: estado.editor.identificador_publico,
      expediente_ref: estado.editor.expediente_ref,
      ...base,
    };
  }

  async function refrescarTrasGuardado(recibo) {
    const teniaDetalle = Boolean(estado.detalle);
    estado.referenciaSeleccionada = recibo.referencia_estado.referencia;
    estado.detalle = estado.detalle ? {
      ...estado.detalle,
      referencia_estado: recibo.referencia_estado,
      etag: recibo.etag,
      contenido_editable: copiar(estado.editor.contenido_editable),
    } : null;
    controladorCarga?.abort();
    controladorCarga = new AbortController();
    await cargarLista({ conservarEditor: true, mantenerVisible: true });
    try {
      const detalle = await clienteAutorizado().obtenerDetalle(
        recibo.referencia_estado.referencia,
        estado.opciones.limites,
      );
      estado.detalle = detalle;
      estado.editor = editorDesdeDetalle(detalle);
      estado.modoEditor = "actualizar";
      estado.faseEditor = "listo";
    } catch {
      // El recibo acredita el guardado. Una lectura posterior fallida no lo degrada.
      estado.errorEditor = {
        mensaje: "El guardado está confirmado, pero el detalle actualizado no se pudo recargar.",
        codigo: "detalle_posterior_no_disponible",
        correlacion: null,
        estadoHTTP: 0,
        tipoConflicto: null,
        conservarCambiosLocales: false,
      };
      if (teniaDetalle) {
        estado.modoEditor = "actualizar";
        estado.faseEditor = "listo";
      } else {
        estado.modoEditor = "ninguno";
        estado.faseEditor = "confirmado";
        estado.detalle = null;
        estado.editor = null;
      }
    }
    notificar();
  }

  async function guardar() {
    if (!estado.editor || estado.guardando || !estado.opciones || !puedeEditar()) return false;
    estado.guardando = true;
    estado.errorEditor = null;
    estado.recibo = null;
    notificar();
    try {
      const solicitud = construirSolicitud();
      estado.claveIdempotencia ||= generarClaveImpl();
      const api = clienteAutorizado();
      const recibo = estado.modoEditor === "crear"
        ? await api.crear(solicitud, estado.opciones.limites, {
          claveIdempotencia: estado.claveIdempotencia,
        })
        : await api.actualizar(
          estado.referenciaSeleccionada,
          solicitud,
          estado.opciones.limites,
          {
            etag: estado.detalle.etag,
            claveIdempotencia: estado.claveIdempotencia,
          },
        );
      estado.recibo = recibo;
      estado.sucio = false;
      estado.claveIdempotencia = "";
      estado.conflictoRemoto = null;
      estado.confirmarReaplicacion = false;
      anunciar(recibo.accion === "crear" ? "Borrador creado y acreditado" : "Borrador actualizado y acreditado");
      await refrescarTrasGuardado(recibo);
      return true;
    } catch (error) {
      estado.errorEditor = {
        ...errorSeguro(error, "No se pudo guardar el borrador."),
        conservarCambiosLocales: true,
      };
      estado.sucio = true;
      if (estado.errorEditor.tipoConflicto === "idempotencia") estado.claveIdempotencia = "";
      anunciar(estado.errorEditor.tipoConflicto
        ? "Conflicto detectado; se conservan los cambios locales"
        : "No se pudo guardar; se conservan los cambios locales");
      return false;
    } finally {
      estado.guardando = false;
      notificar();
    }
  }

  async function cargarEstadoVigente() {
    if (!estado.referenciaSeleccionada || estado.modoEditor !== "actualizar") return false;
    const conflictoAnterior = estado.errorEditor;
    estado.faseEditor = "comparando";
    notificar();
    try {
      const remoto = await clienteAutorizado().obtenerDetalle(
        estado.referenciaSeleccionada,
        estado.opciones.limites,
      );
      estado.conflictoRemoto = remoto;
      estado.faseEditor = "listo";
      estado.confirmarReaplicacion = false;
      notificar();
      anunciar("Estado vigente cargado sin sustituir los cambios locales");
      return true;
    } catch (error) {
      estado.faseEditor = "listo";
      const fallo = errorSeguro(error, "No se pudo cargar el estado vigente para comparar.");
      estado.errorEditor = {
        ...fallo,
        correlacion: fallo.correlacion || conflictoAnterior?.correlacion || null,
        tipoConflicto: "cas",
        conservarCambiosLocales: true,
      };
      notificar();
      return false;
    }
  }

  async function reaplicarSobreVigente() {
    if (!estado.conflictoRemoto || !estado.confirmarReaplicacion) return false;
    if (estado.conflictoRemoto.capacidades.actualizar !== true) {
      estado.errorEditor = {
        mensaje: "La revisión vigente ya no concede capacidad para actualizar este borrador.",
        codigo: "capacidad_actualizar_no_concedida",
        correlacion: null,
        estadoHTTP: 403,
        tipoConflicto: "cas",
        conservarCambiosLocales: true,
      };
      notificar();
      return false;
    }
    estado.detalle = estado.conflictoRemoto;
    estado.conflictoRemoto = null;
    estado.confirmarReaplicacion = false;
    estado.errorEditor = null;
    estado.claveIdempotencia = "";
    estado.sucio = true;
    return guardar();
  }

  function descartarLocalesPorVigente() {
    if (!estado.conflictoRemoto) return false;
    if (!confirmar("Se descartarán definitivamente los cambios locales. ¿Desea continuar?")) return false;
    estado.detalle = estado.conflictoRemoto;
    estado.editor = editorDesdeDetalle(estado.conflictoRemoto);
    estado.conflictoRemoto = null;
    estado.confirmarReaplicacion = false;
    estado.errorEditor = null;
    estado.claveIdempotencia = "";
    estado.sucio = false;
    notificar();
    anunciar("Se ha cargado la versión vigente del servidor");
    return true;
  }

  function cancelarEdicion() {
    if (estado.sucio && !confirmar("¿Desea descartar los cambios locales sin guardar?")) return false;
    if (estado.modoEditor === "actualizar" && estado.detalle) {
      estado.editor = editorDesdeDetalle(estado.detalle);
      estado.sucio = false;
      estado.errorEditor = null;
      estado.recibo = null;
      estado.claveIdempotencia = "";
      estado.conflictoRemoto = null;
      notificar();
      return true;
    }
    estado.modoEditor = "ninguno";
    estado.faseEditor = "vacio";
    estado.editor = null;
    estado.detalle = null;
    estado.sucio = false;
    estado.errorEditor = null;
    estado.recibo = null;
    estado.claveIdempotencia = "";
    notificar();
    return true;
  }

  function marcarSucia() {
    estado.sucio = true;
    estado.errorEditor = null;
    estado.recibo = null;
    estado.claveIdempotencia = "";
    estado.conflictoRemoto = null;
    estado.confirmarReaplicacion = false;
  }

  function actualizarCampo({ ruta, valor, checked = false, tipo = "text" } = {}) {
    if (!estado.editor || estado.guardando || !puedeEditar()) return false;
    if (ruta === "confirmar_reaplicacion") {
      estado.confirmarReaplicacion = checked === true;
      return true;
    }
    if (ruta === "contenido_editable.categorias") {
      const categorias = new Set(estado.editor.contenido_editable.categorias);
      if (checked) categorias.add(valor);
      else categorias.delete(valor);
      estado.editor.contenido_editable.categorias = [...categorias];
    } else {
      let normalizado = valor;
      if (tipo === "checkbox") normalizado = checked === true;
      else if (tipo === "number") normalizado = valor === "" ? "" : Number(valor);
      asignarRutaEditor(estado.editor, ruta, normalizado);
    }
    marcarSucia();
    return true;
  }

  function agregarElemento(coleccion) {
    if (!estado.editor || !puedeEditar()) return false;
    const contenido = estado.editor.contenido_editable;
    if (coleccion === "plazos") {
      contenido.plazos.push({
        referencia: "", tipo: "", titulo: "", descripcion: "", abre_en: "", cierra_en: "",
      });
    } else if (coleccion === "requisitos") {
      contenido.requisitos.push({
        referencia: "", orden: contenido.requisitos.length + 1,
        titulo: "", descripcion: "", obligatorio: true,
      });
    } else if (coleccion === "ayuda") {
      contenido.ayuda.push({
        referencia: "", categoria: "general", orden: contenido.ayuda.length + 1,
        pregunta: "", respuesta: "",
      });
    } else return false;
    marcarSucia();
    notificar();
    return true;
  }

  function quitarElemento(coleccion, indice) {
    if (!puedeEditar()) return false;
    const lista = estado.editor?.contenido_editable?.[coleccion];
    if (!Array.isArray(lista) || !Number.isInteger(indice) || indice < 0 || indice >= lista.length) {
      return false;
    }
    if (coleccion === "plazos" && lista.length === 1) {
      estado.errorEditor = {
        mensaje: "El borrador debe conservar al menos un plazo.",
        codigo: "plazo_obligatorio",
        correlacion: null,
        estadoHTTP: 0,
        tipoConflicto: null,
        conservarCambiosLocales: true,
      };
      notificar();
      return false;
    }
    lista.splice(indice, 1);
    marcarSucia();
    notificar();
    return true;
  }

  async function aplicarFiltro({ texto = "", categoria = "" } = {}) {
    if (estado.guardando) return false;
    estado.filtro = {
      texto: String(texto).trim().normalize("NFC"),
      categoria: String(categoria),
    };
    estado.cursores = [undefined];
    estado.pagina = 0;
    controladorCarga?.abort();
    controladorCarga = new AbortController();
    return cargarLista({ conservarEditor: true });
  }

  async function paginaSiguiente() {
    const cursor = estado.lista?.paginacion?.siguiente_cursor;
    if (!cursor) return false;
    estado.cursores = estado.cursores.slice(0, estado.pagina + 1);
    estado.cursores.push(cursor);
    estado.pagina += 1;
    controladorCarga?.abort();
    controladorCarga = new AbortController();
    return cargarLista({ conservarEditor: true });
  }

  async function paginaAnterior() {
    if (estado.pagina === 0) return false;
    estado.pagina -= 1;
    controladorCarga?.abort();
    controladorCarga = new AbortController();
    return cargarLista({ conservarEditor: true });
  }

  async function manejarAccion({ accion, id = "", coleccion = "", indice = "" } = {}) {
    if (estado.guardando) return false;
    switch (accion) {
      case "borradores-recargar":
        return cargarSuperficie();
      case "borradores-nuevo":
        return iniciarNuevo();
      case "borradores-abrir":
        return cargarDetalle(id);
      case "borradores-pagina-siguiente":
        return paginaSiguiente();
      case "borradores-pagina-anterior":
        return paginaAnterior();
      case "borradores-agregar":
        return agregarElemento(coleccion);
      case "borradores-quitar":
        return quitarElemento(coleccion, Number(indice));
      case "borradores-cancelar-edicion":
        return cancelarEdicion();
      case "borradores-reintentar-guardado":
        return guardar();
      case "borradores-rotar-idempotencia":
        estado.claveIdempotencia = "";
        return guardar();
      case "borradores-cargar-vigente":
        return cargarEstadoVigente();
      case "borradores-reaplicar-vigente":
        return reaplicarSobreVigente();
      case "borradores-descartar-locales":
        return descartarLocalesPorVigente();
      default:
        return false;
    }
  }

  const { renderizar } = crearRenderizadorBorradores({
    escaparHTML,
    estado,
    motivoSeleccionado,
    plantillaSeleccionada,
  });

  return Object.freeze({
    activar,
    aplicarFiltro,
    actualizarCampo,
    guardar,
    manejarAccion,
    renderizar,
  });
}
