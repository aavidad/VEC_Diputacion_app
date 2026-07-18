import { escaparAtributo, escaparHTML, listaDatos } from "./vistas/comunes.js";
import {
  renderizarConvocatorias, renderizarDetalleConvocatoria, renderizarInicio,
} from "./vistas/inicio-convocatorias.js";
import {
  renderizarAutobaremacion, renderizarMeritos, renderizarPerfil, renderizarSolicitud,
} from "./vistas/perfil-meritos-solicitud.js";
import {
  renderizarAlegaciones, renderizarLlamamientos, renderizarSeguimiento, renderizarSubsanaciones,
} from "./vistas/seguimiento-tramites.js";
import { renderizarAyuda, renderizarCertificados, renderizarMensajes } from "./vistas/comunicaciones-ayuda.js";
import {
  aplicarPasoSolicitud, crearPayloadBorrador, crearProgresoSolicitud,
  declaracionFinalConfirmada, localizarSolicitudEdicion,
} from "./flujo-solicitud.js";

const RUTAS = Object.freeze({
  inicio: ["Inicio y plazos", renderizarInicio],
  convocatorias: ["Convocatorias", renderizarConvocatorias],
  convocatoria: ["Detalle de convocatoria", renderizarDetalleConvocatoria],
  perfil: ["Perfil y contacto", renderizarPerfil],
  meritos: ["Méritos y documentos", renderizarMeritos],
  solicitud: ["Nueva solicitud", renderizarSolicitud],
  autobaremacion: ["Autobaremación", renderizarAutobaremacion],
  seguimiento: ["Mis expedientes", renderizarSeguimiento],
  llamamientos: ["Disponibilidad y llamamientos", renderizarLlamamientos],
  subsanaciones: ["Subsanaciones", renderizarSubsanaciones],
  alegaciones: ["Alegaciones", renderizarAlegaciones],
  mensajes: ["Mensajes y noticias", renderizarMensajes],
  certificados: ["Certificados y descargas", renderizarCertificados],
  ayuda: ["Ayuda y accesibilidad", renderizarAyuda],
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

const porId = (id) => document.getElementById(id);

function rutaDesdeURL() {
  const parametros = new URLSearchParams(window.location.search);
  const vista = parametros.get("vista") || "inicio";
  return RUTAS[vista] ? vista : "inicio";
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
  if (estado.presentacionSolicitada) url.searchParams.set("presentacion", "rrhh");
  url.searchParams.set("vista", vista);
  if (opciones.id) url.searchParams.set("id", opciones.id);
  return `${url.pathname}${url.search}`;
}

function actualizarEnlacesNavegacion(estado) {
  document.querySelectorAll("a[data-ruta]").forEach((enlace) => {
    if (enlace.id === "enlace-inicio-institucional" && estado.presentacionSolicitada) {
      enlace.setAttribute("href", "/presentacion/");
      enlace.setAttribute("aria-label", "Volver al selector de recorridos de la presentación");
      return;
    }
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

function mostrarError(estado, error) {
  porId("estado-carga").hidden = true;
  porId("espacio-trabajo").innerHTML = `<section class="estado-error" role="alert"><h2>No se pudo cargar el área personal</h2><p>${escaparHTML(error instanceof Error ? error.message : "Servicio no disponible.")}</p><p>No se muestran datos aparentes y no se ha realizado ninguna operación.</p><button type="button" class="boton-primario" data-accion="reintentar">Reintentar conexión segura</button></section>`;
  estado.error = error;
}

function actualizarShell(estado) {
  const { datos, vista } = estado;
  const titulo = RUTAS[vista][0];
  document.title = `${titulo} · Mi área personal`;
  porId("titulo-vista").textContent = titulo;
  porId("migas-pan").textContent = vista === "inicio" ? "Mi área personal" : `Mi área personal → ${titulo}`;
  porId("aviso-presentacion").hidden = datos.meta.presentacion !== true;
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

function renderizar(estado, { enfocar = false } = {}) {
  if (!estado.datos) return;
  estado.vista = RUTAS[estado.vista] ? estado.vista : "inicio";
  actualizarShell(estado);
  porId("estado-carga").hidden = true;
  porId("espacio-trabajo").innerHTML = RUTAS[estado.vista][1](estado.datos, estado);
  actualizarEnlacesNavegacion(estado);
  aplicarCapacidadesVisibles(estado);
  if (enfocar) {
    porId("contenido-principal").focus({ preventScroll: true });
    window.scrollTo({ top: 0, behavior: "instant" });
  }
}

function navegar(estado, vista, opciones = {}) {
  if (!RUTAS[vista]) vista = "inicio";
  estado.vista = vista;
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
  mostrarDetalle("Identidad y contexto de sesión", `${listaDatos([["Persona", escaparHTML(sesion.nombre_visible)], ["Referencia", escaparHTML(sesion.persona_ref)], ["Método", escaparHTML(sesion.metodo)], ["Origen", escaparHTML(estado.datos.meta.origen)]])}<p class="nota ${estado.datos.meta.presentacion ? "demo" : ""}">${estado.datos.meta.presentacion ? "Identidad sintética. No existe autenticación, certificado ni persona real." : "La autoridad efectiva se comprueba en el servidor para cada operación."}</p>`);
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
  porId("contenido-confirmacion").innerHTML = `<p>${escaparHTML(descripcion || TITULOS_OPERACION[operacion])}</p><dl class="dato-lista"><dt>Acción</dt><dd>${escaparHTML(operacion)}</dd><dt>Objeto</dt><dd>${escaparHTML(id || "Expediente personal")}</dd><dt>Resultado esperado</dt><dd>${estado.datos.meta.presentacion ? "Recibo DEMO sin efectos administrativos" : "Confirmación emitida por el servicio autorizado"}</dd></dl><p class="nota ${estado.datos.meta.presentacion ? "demo" : "aviso"}">${estado.datos.meta.presentacion ? "Esta confirmación solo modifica el estado efímero de la demostración." : "El servidor volverá a comprobar identidad, permiso, estado e idempotencia."}</p>`;
  const confirmar = porId("formulario-confirmacion").querySelector('[value="confirmar"]');
  confirmar.textContent = estado.datos.meta.presentacion ? "Confirmar demostración" : "Confirmar operación";
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
    estado.ultimoRecibo = resultado.recibo;
    if (resultado.datos?.meta) estado.datos = resultado.datos;
    else estado.datos = await estado.cliente.cargar();
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
  porId("contenido-recibo").innerHTML = `<div class="${recibo.presentacion ? "recibo-demo" : ""}"><p><strong>${escaparHTML(recibo.resultado)}</strong></p>${listaDatos([["Referencia", escaparHTML(recibo.referencia)], ["Acción", escaparHTML(recibo.accion)], ["Objetivo", escaparHTML(recibo.objetivo)], ["Fecha UTC", escaparHTML(recibo.fecha)], ["Actor", escaparHTML(recibo.actor)]])}<p>${escaparHTML(recibo.advertencia)}</p></div>`;
  porId("dialogo-recibo").showModal();
  anunciar(`Operación completada. Recibo ${recibo.referencia}.`);
}

function descargarRecibo(estado) {
  if (!estado.ultimoRecibo) {
    notificar("No existe un recibo disponible para descargar.");
    return;
  }
  const contenido = JSON.stringify({ data: estado.ultimoRecibo }, null, 2);
  const url = URL.createObjectURL(new Blob([contenido], { type: "application/json;charset=utf-8" }));
  const enlace = document.createElement("a");
  enlace.href = url;
  enlace.download = `${estado.ultimoRecibo.referencia}.json`;
  enlace.click();
  setTimeout(() => URL.revokeObjectURL(url), 0);
  notificar("Recibo preparado para descarga.");
}

function remitirALecturaLocal(estado, mensaje) {
  notificar(mensaje);
  anunciar("Lectura por voz no iniciada. Se abre la ayuda con audio local y transcripción.");
  if (estado.vista !== "ayuda") navegar(estado, "ayuda");
}

function leerPantalla(estado) {
  if (estado.datos?.meta?.presentacion !== true) {
    remitirALecturaLocal(estado, "La lectura de expedientes privados requiere un conector y una política aprobados. Use el audio local y la transcripción de Ayuda.");
    return;
  }
  const sintetizador = window.speechSynthesis;
  if (!sintetizador || typeof window.SpeechSynthesisUtterance !== "function") {
    remitirALecturaLocal(estado, "La síntesis local no está disponible. Use el audio local y la transcripción de Ayuda.");
    return;
  }
  const voz = sintetizador.getVoices?.().find((candidata) => candidata?.localService === true
    && /^es(?:-|$)/i.test(String(candidata.lang || "")));
  if (!voz) {
    remitirALecturaLocal(estado, "No hay una voz local en español. No se ha enviado texto a ningún servicio; use el audio local y la transcripción de Ayuda.");
    return;
  }
  sintetizador.cancel();
  const texto = porId("contenido-principal")?.innerText?.trim().slice(0, 12_000) || "No hay contenido para leer.";
  const locucion = new SpeechSynthesisUtterance(texto);
  locucion.lang = "es-ES";
  locucion.voice = voz;
  locucion.rate = 1;
  sintetizador.speak(locucion);
  notificar("Lectura por voz iniciada. Pulse Esc o cambie de página para detenerla.");
}

function alternarPreferencia(accion, boton) {
  const atributo = accion === "alternar-texto" ? "textoGrande" : "contraste";
  const destino = accion === "alternar-texto" ? document.documentElement : document.body;
  const activo = destino.dataset[atributo] !== "true";
  destino.dataset[atributo] = String(activo);
  document.querySelectorAll(`[data-accion="${accion}"]`).forEach((control) => control.setAttribute("aria-pressed", String(activo)));
  boton?.blur();
  anunciar(activo ? "Preferencia visual activada." : "Preferencia visual desactivada.");
}

function atenderAccion(estado, boton) {
  const accion = boton.dataset.accion;
  if (accion === "alternar-menu") return alternarMenu();
  if (accion === "cerrar-menu") return cerrarMenu({ restaurarFoco: true });
  if (accion === "alternar-texto" || accion === "alternar-contraste") return alternarPreferencia(accion, boton);
  if (accion === "leer-pantalla") return leerPantalla(estado);
  if (accion === "ver-sesion") return verSesion(estado);
  if (accion === "descargar-recibo") return descargarRecibo(estado);
  if (accion === "reintentar") return cargar(estado);
  if (accion === "abrir-convocatoria") return navegar(estado, "convocatoria", { id: boton.dataset.id });
  if (accion === "volver-convocatorias") return navegar(estado, "convocatorias");
  if (accion === "iniciar-solicitud") {
    estado.convocatoriaSolicitud = boton.dataset.id;
    estado.progresoSolicitud = crearProgresoSolicitud(boton.dataset.id);
    estado.solicitudEdicionId = "";
    estado.errorPasoSolicitud = "";
    estado.pasoSolicitud = 1;
    return navegar(estado, "solicitud");
  }
  if (accion === "paso-anterior") { estado.pasoSolicitud = Math.max(1, estado.pasoSolicitud - 1); return renderizar(estado); }
  if (accion === "seleccionar-convocatoria") {
    if (estado.convocatoriaSolicitud !== boton.value) {
      estado.convocatoriaSolicitud = boton.value;
      estado.progresoSolicitud = crearProgresoSolicitud(boton.value);
      estado.solicitudEdicionId = "";
    }
    return;
  }
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
    if (boton.dataset.operacion === "cambiar_disponibilidad") payload.disponible = id === "true";
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
      navegar(estado, enlace.dataset.ruta);
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
    if (formulario.id === "formulario-solicitud-paso") {
      try {
        const paso = Number(formulario.dataset.paso);
        const progreso = aplicarPasoSolicitud(estado.progresoSolicitud, paso, formularioAObjeto(formulario));
        estado.progresoSolicitud = progreso;
        estado.convocatoriaSolicitud = progreso.convocatoria_id;
        estado.errorPasoSolicitud = "";
        if (paso < 4) {
          estado.pasoSolicitud = paso + 1;
          renderizar(estado, { enfocar: true });
          return;
        }
        const solicitud = localizarSolicitudEdicion(estado.datos, {
          solicitudId: estado.solicitudEdicionId,
          convocatoriaId: progreso.convocatoria_id,
        });
        const payload = crearPayloadBorrador(progreso, solicitud?.id || "");
        prepararOperacion(estado, "guardar_borrador", {
          id: solicitud?.id || "",
          descripcion: estado.datos.meta.presentacion
            ? "Guardar el borrador completo en memoria antes de pago, firma y registro"
            : "Guardar el borrador completo antes de pago, firma y registro",
          payload,
          alCompletar: { seleccionarBorrador: true, pasoSolicitud: 5 },
        });
      } catch (error) {
        estado.errorPasoSolicitud = error instanceof Error ? error.message : "El paso no supera las validaciones.";
        anunciar("No se puede continuar. Revise los campos indicados.");
        renderizar(estado);
      }
      return;
    }
    if (formulario.dataset.operacion) {
      if (estado.datos.capacidades[formulario.dataset.operacion] !== true) {
        notificar("La acción no está habilitada para la identidad y el expediente actuales.");
        anunciar("Operación no disponible.");
        return;
      }
      const payload = formularioAObjeto(formulario);
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
  porId("estado-carga").hidden = false;
  porId("estado-carga").className = "estado-carga";
  porId("estado-carga").innerHTML = '<span aria-hidden="true"></span>Cargando información autorizada…';
  porId("espacio-trabajo").replaceChildren();
  try {
    const datos = await estado.cliente.cargar();
    if (datos.meta.presentacion !== estado.presentacionSolicitada) throw new Error("El origen recibido no coincide con el modo solicitado.");
    estado.datos = datos;
    if (!estado.convocatoriaSolicitud) {
      estado.convocatoriaSolicitud = datos.convocatorias.find((item) => item.estado === "Plazo abierto")?.id || "";
      estado.progresoSolicitud = crearProgresoSolicitud(estado.convocatoriaSolicitud);
    }
    estado.error = null;
    renderizar(estado);
  } catch (error) {
    mostrarError(estado, error);
  }
}

export async function iniciarAreaPersonal({ cliente, presentacionSolicitada = false } = {}) {
  if (!cliente || typeof cliente.cargar !== "function" || typeof cliente.ejecutar !== "function") {
    throw new TypeError("El cliente inyectado no respeta el contrato del área personal.");
  }
  const parametros = new URLSearchParams(window.location.search);
  const estado = {
    cliente,
    presentacionSolicitada,
    datos: null,
    vista: rutaDesdeURL(),
    filtros: { termino: "", estado: "Todas", categoria: "Todas" },
    consultaAyuda: "",
    pasoSolicitud: 1,
    convocatoriaSeleccionada: parametros.get("id") || "",
    convocatoriaSolicitud: "",
    expedienteSeleccionado: parametros.get("id") || "",
    progresoSolicitud: crearProgresoSolicitud(),
    solicitudEdicionId: "",
    errorPasoSolicitud: "",
    operacionPendiente: null,
    ultimoRecibo: null,
  };
  if (presentacionSolicitada) porId("aviso-presentacion").hidden = false;
  conectarEventos(estado);
  await cargar(estado);
  return estado;
}
