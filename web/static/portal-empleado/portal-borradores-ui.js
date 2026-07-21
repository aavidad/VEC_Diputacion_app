import {
  ErrorAPIBorradores,
  crearClienteBorradores,
  generarClaveIdempotencia,
} from "./portal-borradores-api.js";
import { crearControlAccesoBorradores } from "./portal-borradores-acceso.js?v=20260721-acceso-real-v2";
import { ESQUEMAS_BORRADORES } from "./portal-borradores-contrato.js";
import {
  crearEstadoBorradores,
  limpiarEstadoBorradoresRevocado,
} from "./portal-borradores-estado.js?v=20260721-acceso-real-v2";
import { crearCoordinadorOperacionesBorradores } from "./portal-borradores-operaciones.js?v=20260721-acceso-real-v2";
import { crearRenderizadorBorradores } from "./portal-borradores-vista.js";
import { traducirPortal } from "./portal-i18n.js?v=20260721-acceso-real-v2";

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
  traducir = traducirPortal,
} = {}) {
  if ([escaparHTML, anunciar, alCambiar, resolverProveedorBearer, confirmar,
    crearClienteImpl, generarClaveImpl, traducir]
    .some((dependencia) => typeof dependencia !== "function")) {
    throw new TypeError("dependencias de la superficie de borradores no válidas");
  }

  const estado = crearEstadoBorradores();

  let cliente = null;
  let controlAcceso = null;
  const operaciones = crearCoordinadorOperacionesBorradores();

  function notificar() {
    alCambiar();
  }

  function resolverProveedorBearerValido() {
    const proveedor = resolverProveedorBearer();
    if (proveedor !== null && proveedor !== undefined && typeof proveedor !== "function") {
      throw new ErrorAPIBorradores(
        "El proveedor opcional de credencial para borradores no es válido.",
        0,
        undefined,
        { codigo: "proveedor_bearer_no_valido" },
      );
    }
    return proveedor ?? null;
  }

  function clienteAutorizado() {
    if (cliente === null) {
      const proveedor = resolverProveedorBearerValido();
      cliente = crearClienteImpl({
        obtenerBearer: proveedor === null ? null : (signal) => {
          const proveedorActual = resolverProveedorBearerValido();
          if (proveedorActual === null) {
            throw new ErrorAPIBorradores(
              "El proveedor opcional de credencial para borradores dejó de estar disponible.",
              0,
              undefined,
              { codigo: "proveedor_bearer_no_disponible" },
            );
          }
          return proveedorActual(signal);
        },
      });
    }
    return cliente;
  }

  function limpiarEstadoSensible() {
    operaciones.invalidar();
    cliente = null;
    limpiarEstadoBorradoresRevocado(estado, errorSeguro(
      controlAcceso?.obtenerError(), traducir("error_sesion_borradores_denegada"),
    ));
  }

  function alCambiarAcceso(acceso) {
    if (acceso.estado === "denegado") {
      limpiarEstadoSensible();
      anunciar(traducir("anuncio_acceso_borradores_denegado"));
    }
    notificar();
  }

  controlAcceso = crearControlAccesoBorradores({
    consultarOpciones: (opciones) => clienteAutorizado().obtenerOpciones(opciones),
    alCambiar: alCambiarAcceso,
    traducir,
  });

  function invalidarAccesoSiDenegado(error) {
    return controlAcceso.invalidar(error);
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
    const operacion = operaciones.iniciar("detalle");
    estado.faseEditor = "cargando";
    estado.errorEditor = null;
    estado.recibo = null;
    notificar();
    try {
      const detalle = await clienteAutorizado().obtenerDetalle(
        referencia,
        estado.opciones.limites,
        { signal: operacion.signal },
      );
      if (!operacion.vigente()) return false;
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
      if (!operacion.vigente()) return false;
      if (invalidarAccesoSiDenegado(error)) return false;
      estado.faseEditor = "error";
      estado.errorEditor = errorSeguro(error, "No se pudo cargar el borrador seleccionado.");
      notificar();
      anunciar("No se pudo cargar el borrador seleccionado");
      return false;
    } finally {
      operacion.finalizar();
    }
  }

  async function cargarLista({ conservarEditor = true, mantenerVisible = false } = {}) {
    const operacion = operaciones.iniciar("carga");
    if (!mantenerVisible) estado.faseLista = FASE_CARGANDO;
    estado.errorLista = null;
    notificar();
    try {
      const lista = await clienteAutorizado().listar({
        ...opcionesLista(), signal: operacion.signal,
      });
      if (!operacion.vigente()) return false;
      estado.lista = lista;
      estado.faseLista = FASE_LISTA;
      notificar();
      if (!conservarEditor && lista.elementos[0]) {
        await cargarDetalle(lista.elementos[0].referencia_estado.referencia, { descartarSucio: true });
      }
      return true;
    } catch (error) {
      if (!operacion.vigente()) return false;
      if (invalidarAccesoSiDenegado(error)) return false;
      estado.errorLista = errorSeguro(error, "El servicio de borradores no está disponible.");
      estado.faseLista = mantenerVisible && estado.lista ? FASE_LISTA : FASE_ERROR;
      notificar();
      return false;
    } finally {
      operacion.finalizar();
    }
  }

  async function cargarSuperficie({ forzarAcceso = false } = {}) {
    const operacion = operaciones.iniciar("carga");
    estado.faseLista = FASE_CARGANDO;
    estado.errorLista = null;
    notificar();
    try {
      const disponible = await controlAcceso.comprobar({ forzar: forzarAcceso });
      if (!operacion.vigente()) return false;
      if (!disponible) {
        const acceso = controlAcceso.obtenerAcceso();
        estado.opciones = null;
        estado.faseLista = FASE_ERROR;
        estado.errorLista = errorSeguro(
          controlAcceso.obtenerError(),
          acceso.estado === "denegado"
            ? traducir("error_sesion_borradores_denegada")
            : traducir("error_servicio_borradores"),
        );
        notificar();
        anunciar(acceso.estado === "denegado"
          ? traducir("anuncio_acceso_borradores_denegado")
          : traducir("anuncio_servicio_borradores_error"));
        return false;
      }
      estado.opciones = controlAcceso.obtenerOpciones();
      const lista = await clienteAutorizado().listar({ limite: 40, signal: operacion.signal });
      if (!operacion.vigente()) return false;
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
      if (!operacion.vigente()) return false;
      if (invalidarAccesoSiDenegado(error)) return false;
      estado.faseLista = FASE_ERROR;
      estado.errorLista = errorSeguro(error, "El servicio de borradores no está disponible.");
      notificar();
      anunciar(traducir("anuncio_servicio_borradores_error"));
      return false;
    } finally {
      operacion.finalizar();
    }
  }

  function activar() {
    if (estado.faseLista === FASE_INICIAL || estado.faseLista === FASE_ERROR) return cargarSuperficie();
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
    operaciones.cancelar("carga");
    const operacion = operaciones.iniciar("postguardado");
    estado.referenciaSeleccionada = recibo.referencia_estado.referencia;
    estado.detalle = estado.detalle ? {
      ...estado.detalle,
      referencia_estado: recibo.referencia_estado,
      etag: recibo.etag,
      contenido_editable: copiar(estado.editor.contenido_editable),
    } : null;
    try {
      try {
        const lista = await clienteAutorizado().listar({
          ...opcionesLista(), signal: operacion.signal,
        });
        if (!operacion.vigente()) return false;
        estado.lista = lista;
        estado.faseLista = FASE_LISTA;
        estado.errorLista = null;
      } catch (error) {
        if (!operacion.vigente()) return false;
        if (invalidarAccesoSiDenegado(error)) return false;
        estado.errorLista = errorSeguro(error, traducir("error_servicio_borradores"));
      }
      try {
        const detalle = await clienteAutorizado().obtenerDetalle(
          recibo.referencia_estado.referencia, estado.opciones.limites,
          { signal: operacion.signal },
        );
        if (!operacion.vigente()) return false;
        estado.detalle = detalle;
        estado.editor = editorDesdeDetalle(detalle);
        estado.modoEditor = "actualizar";
        estado.faseEditor = "listo";
      } catch (error) {
        if (!operacion.vigente()) return false;
        if (invalidarAccesoSiDenegado(error)) return false;
        estado.errorEditor = {
          mensaje: "El guardado está confirmado, pero el detalle actualizado no se pudo recargar.",
          codigo: "detalle_posterior_no_disponible", correlacion: null, estadoHTTP: 0,
          tipoConflicto: null, conservarCambiosLocales: false,
        };
        estado.modoEditor = teniaDetalle ? "actualizar" : "ninguno";
        estado.faseEditor = teniaDetalle ? "listo" : "confirmado";
        if (!teniaDetalle) {
          estado.detalle = null;
          estado.editor = null;
        }
      }
      if (!operacion.vigente()) return false;
      notificar();
      return true;
    } finally {
      operacion.finalizar();
    }
  }

  async function guardar() {
    if (!estado.editor || estado.guardando || !estado.opciones || !puedeEditar()) return false;
    const operacion = operaciones.iniciar("guardado");
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
          claveIdempotencia: estado.claveIdempotencia, signal: operacion.signal,
        })
        : await api.actualizar(
          estado.referenciaSeleccionada,
          solicitud,
          estado.opciones.limites,
          {
            etag: estado.detalle.etag,
            claveIdempotencia: estado.claveIdempotencia,
            signal: operacion.signal,
          },
        );
      if (!operacion.vigente()) return false;
      estado.recibo = recibo;
      estado.sucio = false;
      estado.claveIdempotencia = "";
      estado.conflictoRemoto = null;
      estado.confirmarReaplicacion = false;
      anunciar(recibo.accion === "crear" ? "Borrador creado y acreditado" : "Borrador actualizado y acreditado");
      return await refrescarTrasGuardado(recibo) && operacion.vigente();
    } catch (error) {
      if (!operacion.vigente()) return false;
      if (invalidarAccesoSiDenegado(error)) return false;
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
      const vigente = operacion.vigente();
      operacion.finalizar();
      if (vigente) {
        estado.guardando = false;
        notificar();
      }
    }
  }

  async function cargarEstadoVigente() {
    if (!estado.referenciaSeleccionada || estado.modoEditor !== "actualizar") return false;
    const operacion = operaciones.iniciar("cas");
    const conflictoAnterior = estado.errorEditor;
    estado.faseEditor = "comparando";
    notificar();
    try {
      const remoto = await clienteAutorizado().obtenerDetalle(
        estado.referenciaSeleccionada,
        estado.opciones.limites,
        { signal: operacion.signal },
      );
      if (!operacion.vigente()) return false;
      estado.conflictoRemoto = remoto;
      estado.faseEditor = "listo";
      estado.confirmarReaplicacion = false;
      notificar();
      anunciar("Estado vigente cargado sin sustituir los cambios locales");
      return true;
    } catch (error) {
      if (!operacion.vigente()) return false;
      if (invalidarAccesoSiDenegado(error)) return false;
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
    } finally {
      operacion.finalizar();
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
    return cargarLista({ conservarEditor: true });
  }

  async function paginaSiguiente() {
    const cursor = estado.lista?.paginacion?.siguiente_cursor;
    if (!cursor) return false;
    estado.cursores = estado.cursores.slice(0, estado.pagina + 1);
    estado.cursores.push(cursor);
    estado.pagina += 1;
    return cargarLista({ conservarEditor: true });
  }

  async function paginaAnterior() {
    if (estado.pagina === 0) return false;
    estado.pagina -= 1;
    return cargarLista({ conservarEditor: true });
  }

  async function manejarAccion({ accion, id = "", coleccion = "", indice = "" } = {}) {
    if (estado.guardando) return false;
    switch (accion) {
      case "borradores-recargar":
        return cargarSuperficie({ forzarAcceso: true });
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
    comprobarDisponibilidad: controlAcceso.comprobar,
    guardar,
    manejarAccion,
    obtenerAcceso: controlAcceso.obtenerAcceso,
    renderizar,
  });
}
