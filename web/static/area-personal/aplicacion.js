import { escaparAtributo, escaparHTML, listaDatos } from "./vistas/comunes.js";
import { MOTIVOS_PAUSA_DISPONIBILIDAD } from "./contrato.js";
import { textosErrorCargaAreaPersonal, traducir } from "./i18n.js";
import { montarVistaOportunidades } from "../comun/oportunidades/vista.js?v=20260924-f2-b15-area-v1";
import {
  renderizarConvocatorias, renderizarDetalleConvocatoria, renderizarInicio,
} from "./vistas/inicio-convocatorias.js";
import {
  renderizarAutobaremacion, renderizarMeritos, renderizarPerfil,
} from "./vistas/perfil-meritos-solicitud.js";
import {
  renderizarAlegaciones, renderizarLlamamientos, renderizarSeguimiento, renderizarSubsanaciones,
} from "./vistas/seguimiento-tramites.js";
import { renderizarAyuda, renderizarCertificados, renderizarMensajes } from "./vistas/comunicaciones-ayuda.js";
import { crearControladorContactoPropio, montarContactoPropio } from "./contacto-propio.js";
import { enviarPortalMiBolsa } from "./mi-bolsa-portal.js";
import { declaracionFinalConfirmada, localizarSolicitudEdicion } from "./flujo-solicitud.js";
import { montarSolicitudConvocatoria } from "./solicitud-convocatoria.js?v=20260926-convoca-f1-v1";
import { montarMisSolicitudes } from "./mis-solicitudes.js?v=20260926-convoca-f1-v1";
import { instalarCopiaReferencias } from "./justificante-copiable.js?v=20260926-convoca-f1-v1";

const MOTIVO_PAUSA_PREDETERMINADO = MOTIVOS_PAUSA_DISPONIBILIDAD[0];
const VISTAS_SOLICITUD = new Set(["solicitud", "mis-solicitudes"]);

const RUTAS = Object.freeze({
  inicio: ["areaPersonal.rutas.inicio", renderizarInicio],
  convocatorias: ["areaPersonal.rutas.convocatorias", renderizarConvocatorias],
  oportunidades: ["areaPersonal.rutas.oportunidades", () => '<div id="oportunidades-montaje"></div>'],
  convocatoria: ["areaPersonal.rutas.convocatoria", renderizarDetalleConvocatoria],
  perfil: ["areaPersonal.rutas.perfil", renderizarPerfil],
  meritos: ["areaPersonal.rutas.meritos", renderizarMeritos],
  solicitud: ["areaPersonal.rutas.solicitud", () => '<div id="solicitud-montaje"></div>'],
  "mis-solicitudes": ["areaPersonal.rutas.misSolicitudes", () => '<div id="mis-solicitudes-montaje"></div>'],
  autobaremacion: ["areaPersonal.rutas.autobaremacion", renderizarAutobaremacion],
  seguimiento: ["areaPersonal.rutas.seguimiento", renderizarSeguimiento],
  llamamientos: ["areaPersonal.rutas.llamamientos", renderizarLlamamientos],
  subsanaciones: ["areaPersonal.rutas.subsanaciones", renderizarSubsanaciones],
  alegaciones: ["areaPersonal.rutas.alegaciones", renderizarAlegaciones],
  mensajes: ["areaPersonal.rutas.mensajes", renderizarMensajes],
  certificados: ["areaPersonal.rutas.certificados", renderizarCertificados],
  ayuda: ["areaPersonal.rutas.ayuda", renderizarAyuda],
});

const TITULOS_OPERACION = Object.freeze({
  actualizar_contacto: "Actualizar datos de contacto",
  incorporar_merito: "Incorporar mérito y evidencia",
  guardar_borrador: "Guardar borrador",
  calcular_autobaremo: "Calcular autobaremación",
  iniciar_pago: "Realizar pago o acreditar exención",
  firmar_solicitud: "Firmar solicitud",
  registrar_solicitud: "Registrar solicitud",
  cambiar_disponibilidad: "Cambiar disponibilidad",
  responder_llamamiento: "Responder llamamiento",
  presentar_subsanacion: "Presentar subsanación",
  presentar_alegacion: "Presentar alegación",
  marcar_mensaje: "Marcar mensaje como leído",
  actualizar_notificaciones: "Actualizar preferencias",
  solicitar_certificado: "Solicitar certificado",
  solicitar_descarga: "Preparar descarga",
});

export function conservarResultadoContactoPropio(estado, { reciboRef, version, correo }) {
  estado.contactoPropio = { ...estado.contactoPropio, version };
  estado.contactoPropioRecibo = { reciboRef, version };
  estado.datos = structuredClone(estado.datos);
  estado.datos.perfil.correo = correo;
}
export function excluirCorreoDeActualizacionContacto(payload) {
  const { correo: _correo, ...sinCorreo } = payload;
  return sinCorreo;
}

const porId = (id) => document.getElementById(id);
export function esOrigenSinteticoODesarrollo(meta = {}) {
  if (meta === null || typeof meta !== "object") return false;
  return [meta.origen, meta.entorno].filter((declaracion) => typeof declaracion === "string")
    .some((declaracion) => /\bsint(?:e|é)tic(?:o|a|os|as)?\b|\bdesarrollo\b/iu.test(declaracion));
}

// Únicos parámetros que el área personal genera en sus propias URL; cualquier
// otro (incluido el antiguo `?presentacion=`) detiene el arranque.
const PARAMETROS_URL_ADMITIDOS = Object.freeze(new Set(["vista", "id"]));

export function exigirParametrosConocidos(parametros) {
  for (const nombre of parametros.keys()) {
    if (!PARAMETROS_URL_ADMITIDOS.has(nombre)) throw new TypeError("La dirección contiene parámetros no admitidos en el área personal.");
  }
}

export function exigirDatosOperativos(datos) {
  if (datos?.meta?.presentacion !== false || esOrigenSinteticoODesarrollo(datos.meta)) {
    throw new TypeError("La fuente del área personal no está configurada para esta consulta.");
  }
  return datos;
}

function rutaDesdeURL() {
  const parametros = new URLSearchParams(window.location.search);
  const vista = parametros.get("vista") || "llamamientos";
  return RUTAS[vista] ? vista : "llamamientos";
}

function formularioAObjeto(formulario) {
  const resultado = {};
  for (const [nombre, valor] of new FormData(formulario).entries()) {
    let normalizado = valor;
    if (valor instanceof File) {
      normalizado = valor.name ? { nombre: valor.name.slice(0, 180), tipo: valor.type.slice(0, 100), tamano: valor.size } : null;
    }
    if (Object.hasOwn(resultado, nombre)) {
      resultado[nombre] = Array.isArray(resultado[nombre]) ? [...resultado[nombre], normalizado] : [resultado[nombre], normalizado];
    } else {
      resultado[nombre] = normalizado;
    }
  }
  return resultado;
}

function crearURL(estado, vista, opciones = {}) {
  const url = new URL(window.location.pathname, window.location.origin);
  url.searchParams.set("vista", vista);
  if (opciones.id) url.searchParams.set("id", opciones.id);
  return `${url.pathname}${url.search}`;
}

function actualizarEnlacesNavegacion(estado) {
  const inicioInstitucional = porId("enlace-inicio-institucional");
  if (inicioInstitucional) {
    inicioInstitucional.dataset.ruta = "inicio";
    inicioInstitucional.setAttribute("href", crearURL(estado, "inicio"));
    inicioInstitucional.setAttribute("aria-label", "Ir al inicio del área personal");
  }
  document.querySelectorAll("a[data-ruta]").forEach((enlace) => {
    if (enlace === inicioInstitucional) return;
    enlace.setAttribute("href", crearURL(estado, enlace.dataset.ruta || "inicio", {
      id: enlace.dataset.id || "",
    }));
  });
}

function aplicarCapacidadesVisibles(estado) {
  let secuencia = 0;
  document.querySelectorAll("[data-operacion]").forEach((control) => {
    const operacion = control.dataset.operacion || "";
    if (estado.datos.capacidades[operacion] === true) return;
    secuencia += 1;
    const idAyuda = `capacidad-bloqueada-${secuencia}`;
    const botones = control.matches("button")
      ? [control]
      : [...control.querySelectorAll('button[type="submit"], input[type="submit"]')];
    botones.forEach((boton) => {
      boton.disabled = true;
      boton.setAttribute("aria-disabled", "true");
      boton.setAttribute("aria-describedby", idAyuda);
      boton.title = "Acción no habilitada por el servicio autorizado";
    });
    const ayuda = document.createElement("small");
    ayuda.id = idAyuda;
    ayuda.className = "nota aviso ayuda-capacidad";
    ayuda.textContent = `${TITULOS_OPERACION[operacion] || "Esta acción"} no está habilitada para la identidad y el expediente actuales.`;
    control.insertAdjacentElement("afterend", ayuda);
  });
}

function anunciar(mensaje) {
  const region = porId("anuncios");
  if (!region) return;
  region.textContent = "";
  requestAnimationFrame(() => { region.textContent = mensaje; });
}

function notificar(mensaje) {
  const contenedor = porId("notificaciones");
  if (!contenedor) return;
  const aviso = document.createElement("div");
  aviso.className = "notificacion";
  aviso.textContent = mensaje;
  contenedor.append(aviso);
  setTimeout(() => aviso.remove(), 4_500);
}

export function renderizarErrorCargaAreaPersonal(error) {
  const textos = textosErrorCargaAreaPersonal(error); const boton = textos.reintentar ? `<button type="button" class="boton-primario" data-accion="reintentar">${escaparHTML(textos.reintentar)}</button>` : "";
  return `<section class="estado-error" role="alert"><h2>${escaparHTML(textos.titulo)}</h2><p>${escaparHTML(textos.detalle)}</p><p>${escaparHTML(textos.garantia)}</p>${boton}</section>`;
}
function mostrarError(estado, error) {
  porId("estado-carga").hidden = true; porId("titulo-vista").textContent = traducir("areaPersonal.estado.error.titulo"); porId("espacio-trabajo").innerHTML = renderizarErrorCargaAreaPersonal(error);
  estado.error = error;
}

// Mi bolsa no recibe el nombre: Bolsa solo lo conserva cifrado en la importación.
// Sin nombre no se muestra ninguno, ni un rótulo que lo sustituya.
export function datosMinimosMiBolsa(consulta) {
  return Object.freeze({
    meta: { presentacion: false, origen: "GET /api/vec/bolsa/mi-bolsa", generado_en: consulta.consultada_en },
    sesion: { nombre_visible: "", iniciales: "—", metodo: traducir("areaPersonal.miBolsa.identidad.metodoNoFacilitado"), persona_ref: null },
    resumen: { acciones_pendientes: 0, convocatorias_abiertas: 0, solicitudes_activas: 0, mensajes_no_leidos: 0, puntuacion_provisional: 0 },
    perfil: { referencia: null, nombre_visible: "", identificador_visible: traducir("areaPersonal.miBolsa.identidad.valorNoFacilitado"), correo: traducir("areaPersonal.miBolsa.identidad.valorNoFacilitado"), telefono: traducir("areaPersonal.miBolsa.identidad.valorNoFacilitado"), domicilio: traducir("areaPersonal.miBolsa.identidad.valorNoFacilitado"), estado_verificacion: traducir("areaPersonal.miBolsa.identidad.valorNoFacilitado") },
    plazos: [], convocatorias: [], meritos: [], solicitudes: [], baremo: [], llamamientos: [], subsanaciones: [], alegaciones: [], mensajes: [], certificados: [], documentos: [], actividad: [], ayuda: [], contratos: [],
    disponibilidad: { disponible: false, estado: "No disponible" }, capacidades: {},
  });
}

function datosDeRespuesta(respuesta) {
  return respuesta?.datos || (respuesta?.consulta ? datosMinimosMiBolsa(respuesta.consulta) : respuesta);
}

function actualizarShell(estado) {
  const { datos, vista } = estado;
  const titulo = traducir(RUTAS[vista][0]);
  document.title = `${titulo} · Mi área personal`;
  porId("titulo-vista").textContent = titulo;
  porId("migas-pan").textContent = vista === "inicio" ? "Mi área personal" : `Mi área personal → ${titulo}`;
  porId("avatar-sesion").textContent = datos.sesion.iniciales;
  porId("nombre-sesion").textContent = datos.sesion.nombre_visible;
  porId("perfil-sesion").textContent = datos.sesion.metodo;
  document.querySelectorAll("[data-ruta]").forEach((enlace) => {
    const activa = enlace.dataset.ruta === vista || (vista === "convocatoria" && enlace.dataset.ruta === "convocatorias");
    if (activa) enlace.setAttribute("aria-current", "page"); else enlace.removeAttribute("aria-current");
  });
  const pendientesLlamamiento = datos.llamamientos.filter((item) => item.estado === "Pendiente de respuesta").length;
  const contadorLlamamientos = porId("contador-llamamientos");
  contadorLlamamientos.textContent = pendientesLlamamiento;
  contadorLlamamientos.hidden = pendientesLlamamiento === 0;
  const contadorMensajes = porId("contador-mensajes");
  contadorMensajes.textContent = datos.resumen.mensajes_no_leidos;
  contadorMensajes.hidden = datos.resumen.mensajes_no_leidos === 0;
}

function renderizar(estado, { enfocar = false, confirmacionContacto = null } = {}) {
  if (!estado.datos) return;
  estado.desmontarOportunidades?.();
  estado.desmontarOportunidades = null;
  estado.desmontarSolicitudes?.();
  estado.desmontarSolicitudes = null;
  estado.destruirContactoPropio?.();
  estado.destruirContactoPropio = null;
  estado.controladorContactoPropio = null;
  if (estado.vista !== "perfil") Object.assign(estado, { contactoPropio: null, contactoPropioRecibo: null });
  estado.vista = RUTAS[estado.vista] ? estado.vista : "inicio";
  actualizarShell(estado);
  porId("estado-carga").hidden = true;
  porId("espacio-trabajo").innerHTML = RUTAS[estado.vista][1](estado.datos, estado);
  if (estado.vista === "oportunidades") {
    // La bandeja actual no aporta una evaluación B15 autorizada: no derivarla de convocatorias.
    const vista = montarVistaOportunidades({ raiz: porId("oportunidades-montaje"), anunciar });
    estado.desmontarOportunidades = vista.desmontar;
  }
  montarVistaSolicitudes(estado);
  actualizarEnlacesNavegacion(estado);
  aplicarCapacidadesVisibles(estado);
  if (estado.vista === "perfil") {
    estado.controladorContactoPropio ??= crearControladorContactoPropio({
      autorizacionServidor: estado.contactoPropio,
      fetchImpl: estado.fetchImpl,
      presentacion: false,
      alConfirmar: (resultado) => {
        conservarResultadoContactoPropio(estado, resultado);
        if (estado.vista === "perfil") renderizar(estado, { confirmacionContacto: resultado });
      },
      alDenegar: () => {
        Object.assign(estado, { contactoPropio: null, contactoPropioRecibo: null });
      },
    });
    const montajeContacto = montarContactoPropio({
      contenedor: porId("contacto-propio"),
      correo: "",
      autorizacionServidor: estado.contactoPropio,
      fetchImpl: estado.fetchImpl,
      presentacion: false,
      reciboAnterior: estado.contactoPropioRecibo, confirmacionReciente: confirmacionContacto,
      enfocarConfirmacion: Boolean(confirmacionContacto),
      controlador: estado.controladorContactoPropio,
    });
    estado.destruirContactoPropio = montajeContacto?.destruir ?? null;
  }

  if (enfocar) {
    porId("contenido-principal").focus({ preventScroll: true });
    window.scrollTo({ top: 0, behavior: "instant" });
  }
}

// Solicitud de participación y «Mis solicitudes» (Selección): vistas con su
// propio adaptador, independientes de la consulta de Mi bolsa.
function montarVistaSolicitudes(estado) {
  const abrirAyuda = (titulo, texto) => mostrarDetalle(titulo, `<p>${escaparHTML(texto)}</p>`);
  const cliente = estado.clienteSolicitudes;
  if (estado.vista === "solicitud") {
    estado.desmontarSolicitudes = montarSolicitudConvocatoria({ raiz: porId("solicitud-montaje"), cliente,
      convocatoriaRef: estado.convocatoriaSolicitud, anunciar, abrirAyuda }).desmontar;
  } else if (estado.vista === "mis-solicitudes") {
    estado.desmontarSolicitudes = montarMisSolicitudes({ raiz: porId("mis-solicitudes-montaje"), cliente, abrirAyuda }).desmontar;
  }
}

function navegar(estado, vista, opciones = {}) {
  if (!RUTAS[vista]) vista = "inicio";
  estado.vista = vista;
  if (vista === "solicitud") estado.convocatoriaSolicitud = opciones.id || "";
  if (vista === "convocatoria") estado.convocatoriaSeleccionada = opciones.id || estado.convocatoriaSeleccionada;
  if (vista === "seguimiento" && opciones.id) estado.expedienteSeleccionado = opciones.id;
  window.history.pushState({ vista }, "", crearURL(estado, vista, opciones));
  cerrarMenu();
  renderizar(estado, { enfocar: true });
}

function cerrarMenu({ restaurarFoco = false } = {}) {
  document.body.dataset.menuAbierto = "false";
  const boton = document.querySelector('[data-accion="alternar-menu"]');
  boton?.setAttribute("aria-expanded", "false");
  const velo = document.querySelector(".velo-menu");
  if (velo) velo.hidden = true;
  if (restaurarFoco) boton?.focus({ preventScroll: true });
}

function alternarMenu() {
  const abierto = document.body.dataset.menuAbierto !== "true";
  if (!abierto) {
    cerrarMenu({ restaurarFoco: true });
    return;
  }
  document.body.dataset.menuAbierto = String(abierto);
  document.querySelector('[data-accion="alternar-menu"]')?.setAttribute("aria-expanded", String(abierto));
  const velo = document.querySelector(".velo-menu");
  if (velo) velo.hidden = !abierto;
  document.querySelector(".ap-navegacion a[href]")?.focus({ preventScroll: true });
}

function mantenerFocoEnMenu(evento) {
  if (evento.key !== "Tab" || document.body.dataset.menuAbierto !== "true") return;
  const controles = [...document.querySelectorAll('#navegacion-lateral a[href], #navegacion-lateral button:not([disabled]), #navegacion-lateral [tabindex]:not([tabindex="-1"])')];
  if (controles.length === 0) return;
  const primero = controles[0];
  const ultimo = controles.at(-1);
  if (evento.shiftKey && (document.activeElement === primero || !document.getElementById("navegacion-lateral")?.contains(document.activeElement))) {
    evento.preventDefault();
    ultimo.focus();
  } else if (!evento.shiftKey && document.activeElement === ultimo) {
    evento.preventDefault();
    primero.focus();
  }
}

function mostrarDetalle(titulo, contenido) {
  porId("titulo-detalle").textContent = titulo;
  porId("contenido-detalle").innerHTML = contenido;
  porId("dialogo-detalle").showModal();
}

function verSesion(estado) {
  const sesion = estado.datos.sesion;
  const prefijoTraduccion = "areaPersonal.sesion.";
  const campos = [...(sesion.nombre_visible ? [[traducir(`${prefijoTraduccion}persona`), escaparHTML(sesion.nombre_visible)]] : []),
    ...(sesion.persona_ref ? [[traducir(`${prefijoTraduccion}referencia`), escaparHTML(sesion.persona_ref)]] : []),
    [traducir(`${prefijoTraduccion}metodo`), escaparHTML(sesion.metodo)],
    [traducir(`${prefijoTraduccion}origen`), escaparHTML(estado.datos.meta.origen)]];
  mostrarDetalle(traducir(`${prefijoTraduccion}titulo`), `${listaDatos(campos)}<p class="nota">${escaparHTML(traducir(`${prefijoTraduccion}autoridadServidor`))}</p>`);
}

function verDocumento(estado, id) {
  const documento = estado.datos.documentos.find((item) => item.id === id);
  if (!documento) {
    notificar("El documento solicitado no está disponible en el ámbito actual.");
    return;
  }
  mostrarDetalle(documento.nombre, listaDatos([["Referencia", escaparHTML(documento.id)], ["Tipo", escaparHTML(documento.tipo)], ["Fecha", escaparHTML(documento.fecha)], ["Estado", escaparHTML(documento.estado)], ["Huella", escaparHTML(documento.huella || "Pendiente del servicio documental")]]));
}

function prepararOperacion(estado, operacion, {
  id = "", descripcion = "", payload = {}, alCompletar = null,
} = {}) {
  if (!TITULOS_OPERACION[operacion]) {
    notificar("La acción no está reconocida por esta superficie.");
    return;
  }
  if (estado.datos.capacidades[operacion] !== true) {
    notificar(`${TITULOS_OPERACION[operacion]} no está habilitada para la identidad y el expediente actuales.`);
    anunciar("Operación no disponible.");
    return;
  }
  estado.operacionPendiente = { operacion, payload: { ...payload, id }, descripcion, alCompletar };
  porId("titulo-confirmacion").textContent = TITULOS_OPERACION[operacion];
  porId("contenido-confirmacion").innerHTML = `<p>${escaparHTML(descripcion || TITULOS_OPERACION[operacion])}</p><dl class="dato-lista"><dt>Acción</dt><dd>${escaparHTML(operacion)}</dd><dt>Objeto</dt><dd>${escaparHTML(id || "Expediente personal")}</dd><dt>Resultado esperado</dt><dd>Confirmación emitida por el servicio autorizado</dd></dl><p class="nota aviso">El servidor volverá a comprobar identidad, permiso, estado e idempotencia.</p>`;
  const confirmar = porId("formulario-confirmacion").querySelector('[value="confirmar"]');
  confirmar.textContent = "Confirmar operación";
  porId("dialogo-confirmacion").showModal();
}

async function ejecutarPendiente(estado) {
  const pendiente = estado.operacionPendiente;
  estado.operacionPendiente = null;
  if (!pendiente) return;
  try {
    const resultado = await estado.cliente.ejecutar({
      accion: pendiente.operacion,
      payload: pendiente.payload,
      confirmacion: true,
      capacidad: estado.datos.capacidades[pendiente.operacion] === true,
    });
    if (resultado?.recibo?.presentacion !== false) {
      throw new TypeError("El servicio no devolvió un recibo válido para el área personal.");
    }
    const datosActualizados = resultado.datos?.meta
      ? exigirDatosOperativos(resultado.datos)
      : exigirDatosOperativos(datosDeRespuesta(await estado.cliente.cargar()));
    estado.datos = datosActualizados;
    estado.ultimoRecibo = resultado.recibo;
    if (pendiente.alCompletar?.seleccionarBorrador === true) {
      const solicitud = localizarSolicitudEdicion(estado.datos, {
        solicitudId: pendiente.payload.id,
        convocatoriaId: pendiente.payload.convocatoria_id,
      });
      if (!solicitud) throw new Error("El servicio guardó el borrador, pero no devolvió su referencia autorizada.");
      estado.solicitudEdicionId = solicitud.id;
    }
    if (Number.isInteger(pendiente.alCompletar?.pasoSolicitud)) {
      estado.pasoSolicitud = pendiente.alCompletar.pasoSolicitud;
    }
    estado.errorPasoSolicitud = "";
    renderizar(estado);
    mostrarRecibo(estado, resultado.recibo);
  } catch (error) {
    notificar(error instanceof Error ? error.message : "No se pudo completar la operación.");
    anunciar("Operación no completada.");
  }
}

function mostrarRecibo(estado, recibo) {
  if (recibo?.presentacion !== false) throw new TypeError("El recibo no pertenece al área personal habilitada.");
  porId("contenido-recibo").innerHTML = `<div><p><strong>${escaparHTML(recibo.resultado)}</strong></p>${listaDatos([["Referencia", escaparHTML(recibo.referencia)], ["Acción", escaparHTML(recibo.accion)], ["Objetivo", escaparHTML(recibo.objetivo)], ["Fecha UTC", escaparHTML(recibo.fecha)], ["Actor", escaparHTML(recibo.actor)]])}<p>${escaparHTML(recibo.advertencia)}</p></div>`;
  const botonDescarga = document.querySelector('[data-accion="descargar-recibo"]');
  if (botonDescarga) {
    botonDescarga.textContent = recibo.accion === "solicitar_certificado"
      ? "Descargar certificado PDF" : "Descargar recibo PDF";
  }
  porId("dialogo-recibo").showModal();
  anunciar(`Operación completada. Recibo ${recibo.referencia}.`);
}

export function crearDescriptorPDFRecibo({ recibo, certificados = [], origen }) {
  if (!recibo || typeof recibo !== "object" || Array.isArray(recibo)) {
    throw new TypeError("El recibo documental no es válido.");
  }
  if (recibo.presentacion !== false) throw new TypeError("El recibo de presentación no está admitido en esta superficie.");
  const esCertificado = recibo.accion === "solicitar_certificado";
  const certificado = esCertificado
    ? certificados.find((item) => item.id === recibo.objetivo)
    : null;
  if (esCertificado && !certificado) {
    throw new TypeError("El certificado solicitado no pertenece al expediente visible.");
  }
  const urlVerificacion = new URL("/verificar/", origen);
  urlVerificacion.searchParams.set("ref", recibo.referencia);
  const prefijoArchivo = esCertificado ? "certificado" : "recibo";
  return Object.freeze({
    referencia: recibo.referencia,
    tipo_documento: esCertificado ? "CERTIFICADO" : "RECIBO DE ACTUACIÓN",
    titulo: certificado?.tipo || "Recibo de actuación del Portal de Recursos Humanos",
    subtitulo: "Diputación de Granada · Área de Recursos Humanos y Régimen Interior",
    marca: "Documento emitido por el sistema de gestión de Recursos Humanos",
    filas: Object.freeze([
      Object.freeze({ etiqueta: "Actuación", valor: recibo.accion }),
      Object.freeze({ etiqueta: "Resultado", valor: recibo.resultado }),
      Object.freeze({ etiqueta: "Referencia del trámite", valor: recibo.objetivo }),
      Object.freeze({ etiqueta: "Fecha de emisión (UTC)", valor: recibo.fecha }),
      Object.freeze({ etiqueta: "Identidad actuante", valor: recibo.actor }),
    ]),
    comprobacion: Object.freeze({ qr_contenido: urlVerificacion.href }),
    nombre_archivo: `${prefijoArchivo}-${recibo.referencia.toLowerCase()}.pdf`,
    texto_certificacion: certificado
      ? "La emisión del certificado corresponde al servicio documental autorizado y requiere su firma o sello verificable."
      : "Se deja constancia de la actuación indicada, su resultado y referencia de comprobación. El servicio autorizado debe emitir y custodiar el recibo.",
  });
}

async function descargarRecibo(estado) {
  if (!estado.ultimoRecibo) {
    notificar("No existe un recibo disponible para descargar.");
    return;
  }
  if (typeof estado.descargarReciboPDF !== "function") {
    notificar("El generador documental autorizado no está disponible.");
    anunciar("No se ha generado ningún documento.");
    return;
  }
  const recibo = estado.ultimoRecibo;
  try {
    await estado.descargarReciboPDF(crearDescriptorPDFRecibo({
      recibo,
      certificados: estado.datos.certificados,
      origen: window.location.origin,
    }));
    notificar(recibo.accion === "solicitar_certificado"
      ? "Certificado PDF preparado para descarga."
      : "Recibo PDF preparado para descarga.");
  } catch {
    notificar(recibo.accion === "solicitar_certificado"
      ? "No se pudo generar el certificado PDF."
      : "No se pudo generar el recibo PDF.");
    anunciar("No se ha descargado ningún documento.");
  }
}

function leerPantalla(estado) {
  notificar("La lectura de expedientes privados requiere un conector y una política aprobados. Consulte la guía textual de Ayuda.");
  anunciar("Lectura por voz no iniciada. Se abre la guía textual de Ayuda.");
  if (estado.vista !== "ayuda") navegar(estado, "ayuda");
}

function alternarPreferencia(accion) {
  const atributo = accion === "alternar-texto" ? "textoGrande" : "contraste";
  const destino = accion === "alternar-texto" ? document.documentElement : document.body;
  const activo = destino.dataset[atributo] !== "true";
  destino.dataset[atributo] = String(activo);
  document.querySelectorAll(`[data-accion="${accion}"]`).forEach((control) => control.setAttribute("aria-pressed", String(activo)));
  anunciar(activo ? "Preferencia visual activada." : "Preferencia visual desactivada.");
}

function atenderAccion(estado, boton) {
  const accion = boton.dataset.accion;
  if (accion === "alternar-menu") return alternarMenu();
  if (accion === "cerrar-menu") return cerrarMenu({ restaurarFoco: true });
  if (accion === "alternar-texto" || accion === "alternar-contraste") return alternarPreferencia(accion);
  if (accion === "leer-pantalla") return leerPantalla(estado);
  if (accion === "ver-sesion") return verSesion(estado);
  if (accion === "descargar-recibo") return void descargarRecibo(estado);
  if (accion === "reintentar") return cargar(estado);
  if (accion === "pagina-participaciones") {
    estado.paginaParticipaciones = Math.max(1, Number(boton.dataset.pagina || 1));
    return renderizar(estado, { enfocar: true });
  }
  if (accion === "abrir-convocatoria") return navegar(estado, "convocatoria", { id: boton.dataset.id });
  if (accion === "volver-convocatorias") return navegar(estado, "convocatorias");
  if (accion === "iniciar-solicitud") return navegar(estado, "solicitud", { id: boton.dataset.id || "" });
  if (accion === "abrir-expediente") { estado.expedienteSeleccionado = boton.dataset.id; return navegar(estado, "seguimiento", { id: boton.dataset.id }); }
  if (accion === "abrir-documento") return verDocumento(estado, boton.dataset.id);
  if (accion === "enfocar-nuevo-merito") {
    document.querySelector(".panel-nuevo-merito")?.scrollIntoView({ behavior: "smooth", block: "start" });
    return document.querySelector("#merito-tipo")?.focus();
  }
  if (accion === "preparar-operacion") {
    let id = boton.dataset.id || "";
    const payload = {};
    if (boton.dataset.operacion === "responder_llamamiento") {
      const [llamamiento, respuesta = "aceptar"] = id.split("|");
      id = llamamiento;
      payload.respuesta = respuesta;
    }
    if (boton.dataset.operacion === "cambiar_disponibilidad") {
      payload.disponible = id === "true";
      if (!payload.disponible) {
        payload.motivo_clave = boton.dataset.motivoClave || "pausa_voluntaria";
        if (boton.dataset.hasta) payload.hasta = boton.dataset.hasta;
        if (boton.dataset.motivoTexto) payload.motivo_texto = boton.dataset.motivoTexto;
      }
    }
    if (boton.dataset.operacion === "calcular_autobaremo") {
      const borrador = localizarSolicitudEdicion(estado.datos, {
        solicitudId: estado.solicitudEdicionId,
        convocatoriaId: id,
      });
      const seleccionados = estado.progresoSolicitud?.convocatoria_id === id
        && estado.progresoSolicitud.meritos_ids?.length
        ? estado.progresoSolicitud.meritos_ids
        : borrador?.meritos_ids?.length ? borrador.meritos_ids : estado.datos.meritos.map((item) => item.id);
      payload.convocatoria_id = id;
      payload.meritos_ids = [...seleccionados];
    }
    if (["iniciar_pago", "firmar_solicitud"].includes(boton.dataset.operacion) && !id) {
      notificar("Guarde primero el borrador y espere a que el servicio devuelva su referencia.");
      return;
    }
    return prepararOperacion(estado, boton.dataset.operacion, { id, descripcion: boton.dataset.descripcion, payload });
  }
}

function conectarEventos(estado) {
  document.addEventListener("click", (evento) => {
    const enlace = evento.target.closest("[data-ruta]");
    if (enlace) {
      evento.preventDefault();
      navegar(estado, enlace.dataset.ruta, { id: enlace.dataset.id || "" });
      return;
    }
    const boton = evento.target.closest("[data-accion]");
    if (boton) atenderAccion(estado, boton);
  });
  document.addEventListener("submit", (evento) => {
    const formulario = evento.target;
    if (!(formulario instanceof HTMLFormElement)) return;
    if (formulario.method === "dialog") return;
    evento.preventDefault();
    if (formulario.dataset.portalMiBolsa) {
      enviarPortalMiBolsa(formulario, { fetchImpl: estado.fetchImpl, alRegistrar: () => cargar(estado) });
      return;
    }
    if (formulario.id === "busqueda-global") {
      estado.filtros.termino = formularioAObjeto(formulario).consulta || "";
      navegar(estado, "convocatorias");
      return;
    }
    if (formulario.dataset.accion === "filtrar-convocatorias") {
      estado.filtros = formularioAObjeto(formulario);
      renderizar(estado);
      return;
    }
    if (formulario.dataset.accion === "buscar-ayuda") {
      estado.consultaAyuda = formularioAObjeto(formulario).consulta || "";
      renderizar(estado);
      return;
    }
    if (formulario.dataset.operacion) {
      if (estado.datos.capacidades[formulario.dataset.operacion] !== true) {
        notificar("La acción no está habilitada para la identidad y el expediente actuales.");
        anunciar("Operación no disponible.");
        return;
      }
      const payloadInicial = formularioAObjeto(formulario);
      const payload = formulario.dataset.operacion === "actualizar_contacto"
        ? excluirCorreoDeActualizacionContacto(payloadInicial) : payloadInicial;
      if (formulario.dataset.operacion === "cambiar_disponibilidad") {
        const disponible = payload.disponible === "true" || payload.disponible === true;
        if (disponible) {
          payload.disponible = true;
          delete payload.motivo_clave;
          delete payload.motivo_texto;
          delete payload.hasta;
        } else {
          payload.disponible = false;
          payload.motivo_clave ||= MOTIVO_PAUSA_PREDETERMINADO;
          if (payload.hasta && typeof payload.hasta === "string" && payload.hasta.trim()) {
            payload.hasta = payload.hasta.trim();
          } else {
            delete payload.hasta;
          }
          if (payload.motivo_texto && typeof payload.motivo_texto === "string" && payload.motivo_texto.trim()) {
            payload.motivo_texto = payload.motivo_texto.trim();
          } else {
            delete payload.motivo_texto;
          }
        }
        delete payload.confirmacion;
      }
      if (formulario.dataset.operacion === "registrar_solicitud") {
        if (!declaracionFinalConfirmada(payload.declaracion_final)) {
          estado.errorPasoSolicitud = "Debe confirmar la declaración final antes de registrar la solicitud.";
          anunciar("No se puede registrar sin la declaración final.");
          renderizar(estado);
          return;
        }
        payload.declaracion_final = true;
      }
      prepararOperacion(estado, formulario.dataset.operacion, {
        id: formulario.dataset.id || "",
        descripcion: TITULOS_OPERACION[formulario.dataset.operacion],
        payload,
      });
    }
  });
  porId("dialogo-confirmacion").addEventListener("close", () => {
    if (porId("dialogo-confirmacion").returnValue === "confirmar") ejecutarPendiente(estado);
    else estado.operacionPendiente = null;
  });
  window.addEventListener("popstate", () => {
    cerrarMenu();
    estado.vista = rutaDesdeURL();
    estado.convocatoriaSeleccionada = new URLSearchParams(window.location.search).get("id") || estado.convocatoriaSeleccionada;
    if (estado.vista === "solicitud") estado.convocatoriaSolicitud = new URLSearchParams(window.location.search).get("id") || "";
    renderizar(estado, { enfocar: true });
  });
  window.addEventListener("keydown", (evento) => {
    mantenerFocoEnMenu(evento);
    if (evento.key !== "Escape") return;
    window.speechSynthesis?.cancel?.();
    if (document.body.dataset.menuAbierto === "true") {
      evento.preventDefault();
      cerrarMenu({ restaurarFoco: true });
    }
  });
}

async function cargar(estado) {
  const reintento = document.activeElement?.dataset.accion === "reintentar"; porId("estado-carga").hidden = false;
  porId("estado-carga").className = "estado-carga";
  porId("estado-carga").innerHTML = '<span aria-hidden="true"></span>Cargando información autorizada…';
  porId("espacio-trabajo").replaceChildren();
  try {
    const respuesta = await estado.cliente.cargar();
    const datos = datosDeRespuesta(respuesta);
    estado.datos = exigirDatosOperativos(datos);
    estado.participaciones = respuesta?.consulta?.participaciones || [];
    estado.camposMiBolsa = respuesta?.consulta?.campos_visibles || null;
    estado.portalMiBolsa = respuesta?.consulta?.portal || null;
    estado.accionesPortal = respuesta?.consulta?.acciones_portal || null;
    estado.ofertasMiBolsa = respuesta?.consulta?.ofertas || null;
    estado.contactosMiBolsa = respuesta?.consulta?.contactos || null;
    estado.fuenteBolsa = respuesta?.fuente || "real";
    estado.causaBolsa = respuesta?.causa || "";
    estado.error = null;
    renderizar(estado);
  } catch (error) {
    // Quien aún no está en ninguna bolsa puede solicitar: las vistas de
    // solicitud no dependen de Mi bolsa y siguen disponibles sin sus datos.
    if (VISTAS_SOLICITUD.has(estado.vista) && error?.codigo !== "autenticacion_requerida") {
      estado.datos = datosMinimosMiBolsa({ consultada_en: "" });
      estado.error = null;
      renderizar(estado);
      return;
    }
    mostrarError(estado, error); if (reintento) porId("espacio-trabajo").querySelector('[data-accion="reintentar"]')?.focus();
  }
}

export async function iniciarAreaPersonal({ cliente, clienteSolicitudes = null, descargarReciboPDF = null, fetchImpl = globalThis.fetch } = {}) {
  if (!cliente || typeof cliente.cargar !== "function" || typeof cliente.ejecutar !== "function") {
    throw new TypeError("El cliente inyectado no respeta el contrato del área personal.");
  }
  const parametros = new URLSearchParams(window.location.search);
  exigirParametrosConocidos(parametros);
  const estado = {
    cliente,
    descargarReciboPDF,
    datos: null,
    vista: rutaDesdeURL(),
    filtros: { termino: "", estado: "Todas", categoria: "Todas" },
    consultaAyuda: "",
    pasoSolicitud: 1,
    convocatoriaSeleccionada: parametros.get("id") || "",
    convocatoriaSolicitud: rutaDesdeURL() === "solicitud" ? parametros.get("id") || "" : "",
    clienteSolicitudes,
    desmontarSolicitudes: null,
    expedienteSeleccionado: parametros.get("id") || "",
    solicitudEdicionId: "",
    errorPasoSolicitud: "",
    operacionPendiente: null,
    ultimoRecibo: null,
    contactoPropio: null,
    contactoPropioRecibo: null,
    controladorContactoPropio: null,
    destruirContactoPropio: null,
    desmontarOportunidades: null,
    fetchImpl,
    participaciones: [],
    paginaParticipaciones: 1,
    fuenteBolsa: "real",
    causaBolsa: "",
  };
  conectarEventos(estado);
  instalarCopiaReferencias(document);
  await cargar(estado);
  return estado;
}
