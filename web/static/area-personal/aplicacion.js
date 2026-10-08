import { escaparHTML, listaDatos } from "./vistas/comunes.js";
import { IDIOMAS_DISPONIBLES } from "../comun/idioma.js";
import { idiomaActivoAreaPersonal, iniciarI18nAreaPersonal, textosErrorCargaAreaPersonal,
  textosPreferenciasAreaPersonal, traducir } from "./i18n.js";
import { alternarVisualSesion, crearOperacionPreferencias, montarAvatarAreaPersonal, montarUsuariosAreaPersonal,
  pintarInicialesSesion, reintentarImagenAreaPersonal, renderizarPreferencias,
  sincronizarAtajosVisuales, valoresDelFormulario } from "./preferencias.js?v=20261008-b4-v5";
import { montarVistaOportunidades } from "../comun/oportunidades/vista.js?v=20260924-f2-b15-area-v1";
import { renderizarInicio } from "./vistas/inicio-convocatorias.js?v=20261008-b4-v5";
import { renderizarPerfil } from "./vistas/perfil-meritos-solicitud.js?v=20261008-b4-v5";
import { renderizarLlamamientos } from "./vistas/seguimiento-tramites.js?v=20261008-b4-v5";
import { renderizarAyuda } from "./vistas/comunicaciones-ayuda.js?v=20261008-b4-v5";
import { crearControladorContactoPropio, montarContactoPropio } from "./contacto-propio.js?v=20261005-b4b-v1";
import { montarFichaAspirante } from "./ficha-aspirante.js?v=20260930-portales-i18n-integracion-v1";
import { enviarPortalMiBolsa } from "./mi-bolsa-portal.js?v=20261002-rrhh17-v1";
import { montarHistorialMiBolsa } from "./mi-bolsa-historial.js";


const RUTAS = Object.freeze({
  inicio: ["areaPersonal.rutas.inicio", renderizarInicio],
  preferencias: ["areaPersonal.preferencias.titulo", (_, estado) => renderizarPreferenciasArea(estado)],
  oportunidades: ["areaPersonal.rutas.oportunidades", () => '<div id="oportunidades-montaje"></div>'],
  perfil: ["areaPersonal.rutas.perfil", renderizarPerfil],
  llamamientos: ["areaPersonal.rutas.llamamientos", renderizarLlamamientos],
  ayuda: ["areaPersonal.rutas.ayuda", renderizarAyuda],
});

// Una vista se abre solo si existe y el catálogo `vistas.json` la activa.
const rutaDisponible = (estado, vista) => Object.hasOwn(RUTAS, vista) && estado.vistasDisponibles.has(vista);
const t = (clave, variables) => traducir(`areaPersonal.app.${clave}`, variables);

function renderizarPreferenciasArea(estado) {
  const incidencia = estado.errorUsuarios || !textosPreferenciasAreaPersonal()
    ? `<section class="estado-error" role="alert"><p>${escaparHTML(traducir("areaPersonal.preferencias.componentesNoDisponibles"))}</p><button type="button" class="boton-secundario" data-accion="reintentar-usuarios">${escaparHTML(traducir("areaPersonal.preferencias.reintentarComponentes"))}</button></section>`
    : "";
  if (!textosPreferenciasAreaPersonal()) return incidencia;
  return renderizarPreferencias(estado.preferencias) + (estado.imagen?.renderizar() ?? "")
    + (estado.correos?.renderizar() ?? "") + incidencia;
}

export function conservarResultadoContactoPropio(estado, { reciboRef, version, correo }) {
  estado.contactoPropio = { ...estado.contactoPropio, version };
  estado.contactoPropioRecibo = { reciboRef, version };
  estado.datos = structuredClone(estado.datos);
  estado.datos.perfil.correo = correo;
}

const porId = (id) => document.getElementById(id);
export function esOrigenSinteticoODesarrollo(meta = {}) {
  if (meta === null || typeof meta !== "object") return false;
  return [meta.origen, meta.entorno].filter((declaracion) => typeof declaracion === "string")
    .some((declaracion) => /\bsint(?:e|é)tic(?:o|a|os|as)?\b|\bdesarrollo\b/iu.test(declaracion));
}

// El identificador se admite solo para redirigir enlaces antiguos; las URL nuevas
// usan vista e idioma. Otros parámetros (como `?presentacion=`) se rechazan.
const PARAMETROS_URL_ADMITIDOS = Object.freeze(new Set(["vista", "id", "lang"]));

export function exigirParametrosConocidos(parametros) {
  for (const nombre of parametros.keys()) {
    if (!PARAMETROS_URL_ADMITIDOS.has(nombre)) throw new TypeError(t("parametrosNoAdmitidos"));
  }
}

export function exigirDatosOperativos(datos) {
  if (datos?.meta?.presentacion !== false || esOrigenSinteticoODesarrollo(datos.meta)) {
    throw new TypeError(t("fuenteNoConfigurada"));
  }
  return datos;
}

function rutaDesdeURL(estado) {
  const parametros = new URLSearchParams(window.location.search);
  const vista = parametros.get("vista") || "llamamientos";
  return rutaDisponible(estado, vista) ? vista : "llamamientos";
}

function crearURL(estado, vista) {
  const url = new URL(window.location.pathname, window.location.origin);
  url.searchParams.set("vista", vista);
  const idiomaURL = new URLSearchParams(window.location.search).get("lang");
  if (IDIOMAS_DISPONIBLES.some(({ codigo }) => codigo === idiomaURL)) url.searchParams.set("lang", idiomaURL);
  return `${url.pathname}${url.search}`;
}

// Menú y enlaces internos a vistas desactivadas no se muestran.
function ocultarRutasNoDisponibles(estado) {
  document.querySelectorAll("[data-ruta]").forEach((enlace) => {
    enlace.hidden = !rutaDisponible(estado, enlace.dataset.ruta || "");
  });
}

function actualizarEnlacesNavegacion(estado) {
  ocultarRutasNoDisponibles(estado);
  const inicioInstitucional = porId("enlace-inicio-institucional");
  if (inicioInstitucional) {
    inicioInstitucional.dataset.ruta = "inicio";
    inicioInstitucional.setAttribute("href", crearURL(estado, "inicio"));
    inicioInstitucional.setAttribute("aria-label", traducir("areaPersonal.html.irInicio"));
  }
  document.querySelectorAll("a[data-ruta]").forEach((enlace) => {
    if (enlace === inicioInstitucional) return;
    enlace.setAttribute("href", crearURL(estado, enlace.dataset.ruta || "inicio"));
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
    meta: { presentacion: false, origen: "GET /api/vec/bolsa/mi-bolsa", generado_en: consulta.consultada_en, busqueda_convocatorias_disponible: false },
    sesion: { nombre_visible: "", iniciales: "—", metodo: traducir("areaPersonal.miBolsa.identidad.metodoNoFacilitado"), persona_ref: null },
    perfil: { referencia: null, nombre_visible: "", identificador_visible: traducir("areaPersonal.miBolsa.identidad.valorNoFacilitado"), correo: traducir("areaPersonal.miBolsa.identidad.valorNoFacilitado"), telefono: traducir("areaPersonal.miBolsa.identidad.valorNoFacilitado"), domicilio: traducir("areaPersonal.miBolsa.identidad.valorNoFacilitado"), estado_verificacion: traducir("areaPersonal.miBolsa.identidad.valorNoFacilitado") },
  });
}

function datosMinimosPreferencias(identidadConfirmada) {
  const base = datosMinimosMiBolsa({ consultada_en: "" });
  return { ...base,
    meta: { presentacion: false, origen: "GET /api/vec/usuarios/area-personal/mis-preferencias", busqueda_convocatorias_disponible: false },
    sesion: { ...base.sesion, metodo: traducir(identidadConfirmada
      ? "areaPersonal.preferencias.identidadServicio" : "areaPersonal.preferencias.identidadNoConfirmada") },
  };
}

function asegurarShellPreferencias(estado) {
  if (estado.vista !== "preferencias" || estado.datos
    || (!estado.preferencias.estado && !estado.preferencias.error)) return false;
  estado.datos = datosMinimosPreferencias(Boolean(estado.preferencias.estado));
  estado.soloPreferencias = true;
  return true;
}

function datosDeRespuesta(respuesta) {
  return respuesta?.datos || (respuesta?.consulta ? datosMinimosMiBolsa(respuesta.consulta) : respuesta);
}

function actualizarShell(estado) {
  const { datos, vista } = estado;
  const titulo = traducir(RUTAS[vista][0]);
  document.title = t("tituloDocumento", { titulo });
  porId("titulo-vista").textContent = titulo;
  porId("migas-pan").textContent = vista === "inicio" ? t("migas") : t("migasVista", { titulo });
  pintarInicialesSesion(estado, porId("avatar-sesion"), datos.sesion.iniciales);
  porId("nombre-sesion").textContent = datos.sesion.nombre_visible;
  porId("perfil-sesion").textContent = datos.sesion.metodo;
  document.querySelectorAll("[data-ruta]").forEach((enlace) => {
    const activa = enlace.dataset.ruta === vista;
    if (activa) enlace.setAttribute("aria-current", "page"); else enlace.removeAttribute("aria-current");
  });
}

function renderizar(estado, { enfocar = false, confirmacionContacto = null } = {}) {
  if (!estado.datos) return;
  estado.destruirHistorialMiBolsa?.();
  estado.destruirHistorialMiBolsa = null;
  estado.desmontarOportunidades?.();
  estado.desmontarOportunidades = null;
  estado.destruirContactoPropio?.();
  estado.destruirFichaAspirante?.(); estado.destruirFichaAspirante = null;
  estado.destruirContactoPropio = null;
  estado.controladorContactoPropio = null;
  if (estado.vista !== "perfil") Object.assign(estado, { contactoPropio: null, contactoPropioRecibo: null });
  estado.vista = rutaDisponible(estado, estado.vista) ? estado.vista : "inicio";
  actualizarShell(estado);
  porId("estado-carga").hidden = true;
  porId("espacio-trabajo").innerHTML = RUTAS[estado.vista][1](estado.datos, estado);
  if (estado.avisoInicio && estado.vista !== "preferencias") {
    const aviso = document.createElement("p");
    aviso.className = "preferencias-estado preferencias-aviso";
    aviso.textContent = traducir("areaPersonal.preferencias.inicioAjeno");
    porId("espacio-trabajo").prepend(aviso);
  }
  if (estado.vista === "llamamientos") {
    estado.destruirHistorialMiBolsa = montarHistorialMiBolsa({
      contenedor: porId("historial-mi-bolsa"), fetchImpl: estado.fetchImpl,
    })?.destruir ?? null;
  }
  if (estado.vista === "oportunidades") {
    // La bandeja actual no aporta una evaluación B15 autorizada: no derivarla de convocatorias.
    const vista = montarVistaOportunidades({ raiz: porId("oportunidades-montaje"), anunciar });
    estado.desmontarOportunidades = vista.desmontar;
  }
  actualizarEnlacesNavegacion(estado);
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
    estado.destruirFichaAspirante = montarFichaAspirante({ contenedor: porId("ficha-aspirante"), fetchImpl: estado.fetchImpl })?.destruir ?? null;
  }

  if (enfocar) {
    porId("contenido-principal").focus({ preventScroll: true });
    window.scrollTo({ top: 0, behavior: "instant" });
  }
}

async function navegar(estado, vista) {
  const secuencia = ++estado.secuenciaNavegacion;
  if (!rutaDisponible(estado, vista)) vista = "inicio";
  if (vista === "preferencias") {
    await iniciarI18nAreaPersonal(document, { ubicacion: window.location,
      idiomaPreferido: estado.preferencias.estado?.valores?.idioma,
      pantalla: "preferencias" });
    if (secuencia !== estado.secuenciaNavegacion) return;
    estado.errorUsuarios = !montarUsuariosAreaPersonal(estado, estado.fetchImpl, porId("espacio-trabajo"));
  }
  estado.vista = vista;
  estado.avisoInicio = false;
  window.history.pushState({ vista }, "", crearURL(estado, vista));
  cerrarMenu();
  cerrarMenuIdentidad();
  if (vista !== "preferencias" && (estado.soloPreferencias || !estado.datos || estado.recargarDatosAlSalirPreferencias)) {
    estado.datos = null;
    estado.soloPreferencias = false;
    estado.recargarDatosAlSalirPreferencias = false;
    void cargar(estado);
    return;
  }
  asegurarShellPreferencias(estado);
  renderizar(estado, { enfocar: true });
}

function cerrarMenuIdentidad({ restaurarFoco = false } = {}) {
  const boton = document.querySelector('[data-accion="ver-sesion"]');
  const menu = porId("menu-identidad");
  if (menu) menu.hidden = true;
  boton?.setAttribute("aria-expanded", "false");
  if (restaurarFoco) boton?.focus({ preventScroll: true });
}

function alternarMenuIdentidad() {
  const menu = porId("menu-identidad");
  const boton = document.querySelector('[data-accion="ver-sesion"]');
  if (!menu || !boton) return;
  menu.hidden = !menu.hidden;
  boton.setAttribute("aria-expanded", String(!menu.hidden));
  if (!menu.hidden) menu.querySelector("button")?.focus({ preventScroll: true });
  else boton.focus({ preventScroll: true });
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
  document.querySelector(".ap-navegacion a[href]:not([hidden])")?.focus({ preventScroll: true });
}

function mantenerFocoEnMenu(evento) {
  if (evento.key !== "Tab" || document.body.dataset.menuAbierto !== "true") return;
  const controles = [...document.querySelectorAll('#navegacion-lateral a[href]:not([hidden]), #navegacion-lateral button:not([disabled]):not([hidden]), #navegacion-lateral [tabindex]:not([tabindex="-1"]):not([hidden])')];
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
    [traducir(`${prefijoTraduccion}metodo`), escaparHTML(sesion.metodo)]];
  mostrarDetalle(traducir(`${prefijoTraduccion}titulo`), listaDatos(campos));
}

function leerPantalla(estado) {
  notificar(t("lectura.aviso"));
  anunciar(t("lectura.anuncio"));
  if (estado.vista !== "ayuda") navegar(estado, "ayuda");
}

async function recargarPreferencias(estado) {
  const preferencias = estado.preferencias;
  preferencias.guardando = true;
  preferencias.error = null;
  renderizar(estado);
  porId("espacio-trabajo")?.querySelector(".preferencias-panel h2")?.focus({ preventScroll: true });
  try {
    const lectura = await estado.clientePreferencias.cargar();
    Object.assign(preferencias, { ...lectura, error: null, recibo: null, pendiente: null, borrador: null });
    preferencias.avisoInicio = inicioAjenoElegido(lectura.estado);
    estado.filasPreferidas = lectura.estado.valores.filas;
    estado.paginaParticipaciones = 1;
    estado.controladorVisual?.aplicarPreferenciasServidor(lectura.estado.valores);
    sincronizarAtajosVisuales(lectura.estado.valores);
    const idiomaAnterior = idiomaActivoAreaPersonal();
    await iniciarI18nAreaPersonal(document, { idiomaPreferido: lectura.estado.valores.idioma });
    if (idiomaActivoAreaPersonal() !== idiomaAnterior) estado.recargarDatosAlSalirPreferencias = true;
    estado.errorUsuarios = !montarUsuariosAreaPersonal(estado, estado.fetchImpl, porId("espacio-trabajo"));
  } catch (error) {
    preferencias.error = error;
    preferencias.catalogo = null;
    preferencias.estado = null;
  } finally {
    preferencias.guardando = false;
    if (estado.vista === "preferencias") {
      renderizar(estado);
      const contenido = porId("espacio-trabajo");
      (contenido?.querySelector(".preferencias-error") ?? contenido?.querySelector(".preferencias-panel h2"))
        ?.focus({ preventScroll: true });
    }
  }
}

async function guardarPreferencias(estado, formulario, { reintento = false } = {}) {
  const preferencias = estado.preferencias;
  if (!preferencias || preferencias.guardando || !preferencias.estado || !preferencias.catalogo) return;
  let operacion;
  try {
    operacion = reintento ? preferencias.pendiente
      : crearOperacionPreferencias(preferencias, valoresDelFormulario(formulario));
    if (!operacion) return;
  } catch (error) {
    preferencias.error = { codigo: "servicio" };
    renderizar(estado);
    porId("espacio-trabajo")?.querySelector("[data-pref-resultado]")?.focus({ preventScroll: true });
    return;
  }
  preferencias.borrador = operacion.valores;
  preferencias.guardando = true;
  preferencias.error = null;
  renderizar(estado);
  porId("espacio-trabajo")?.querySelector(".preferencias-panel h2")?.focus({ preventScroll: true });
  try {
    const resultado = await estado.clientePreferencias.guardar(operacion);
    if (resultado.persona_ref !== preferencias.estado.persona_ref) {
      throw Object.assign(new Error(t("respuestaAjena")), { codigo: "respuesta" });
    }
    preferencias.estado = { persona_ref: resultado.persona_ref, version: resultado.version,
      catalogo_version_ref: resultado.catalogo_version_ref, valores: resultado.valores };
    preferencias.recibo = resultado;
    preferencias.pendiente = null;
    preferencias.borrador = null;
    preferencias.avisoInicio = inicioAjenoElegido(resultado);
    estado.filasPreferidas = resultado.valores.filas;
    estado.paginaParticipaciones = 1;
    estado.controladorVisual?.aplicarPreferenciasServidor(resultado.valores);
    sincronizarAtajosVisuales(resultado.valores);
    const idiomaAnterior = idiomaActivoAreaPersonal();
    await iniciarI18nAreaPersonal(document, { idiomaPreferido: resultado.valores.idioma });
    if (idiomaActivoAreaPersonal() !== idiomaAnterior) estado.recargarDatosAlSalirPreferencias = true;
    estado.errorUsuarios = !montarUsuariosAreaPersonal(estado, estado.fetchImpl, porId("espacio-trabajo"));
    anunciar(traducir("areaPersonal.preferencias.guardado", { recibo: resultado.recibo_ref }));
  } catch (error) {
    preferencias.error = error;
    preferencias.pendiente = ["servicio", "respuesta"].includes(error?.codigo) ? operacion : null;
  } finally {
    preferencias.guardando = false;
    if (estado.vista === "preferencias") {
      renderizar(estado);
      porId("espacio-trabajo")?.querySelector("[data-pref-resultado]")?.focus({ preventScroll: true });
    }
  }
}

function atenderAccion(estado, boton) {
  const accion = boton.dataset.accion;
  if (accion === "alternar-menu") return alternarMenu();
  if (accion === "cerrar-menu") return cerrarMenu({ restaurarFoco: true });
  if (accion === "alternar-texto" || accion === "alternar-contraste") {
    const activo = alternarVisualSesion(estado.controladorVisual, accion);
    return anunciar(traducir(`areaPersonal.preferencias.atajo.${accion === "alternar-texto" ? "texto" : "contraste"}${activo ? "Activo" : "Inactivo"}`));
  }
  if (accion === "abrir-preferencias") return navegar(estado, "preferencias");
  if (accion === "reintentar-imagen") return reintentarImagenAreaPersonal(estado, document);
  if (accion === "leer-pantalla") return leerPantalla(estado);
  if (accion === "ver-sesion") return alternarMenuIdentidad();
  if (accion === "ver-contexto-sesion") { cerrarMenuIdentidad(); return verSesion(estado); }
  if (accion === "recargar-preferencias") return void recargarPreferencias(estado);
  if (accion === "reintentar-preferencias") return void guardarPreferencias(estado, null, { reintento: true });
  if (accion === "ayuda-preferencias") {
    const ayuda = porId("ayuda-preferencias");
    if (ayuda) { ayuda.hidden = !ayuda.hidden; boton.setAttribute("aria-expanded", String(!ayuda.hidden)); }
    return;
  }
  if (accion === "reintentar") return cargar(estado);
  if (accion === "reintentar-usuarios") {
    return iniciarI18nAreaPersonal(document, { ubicacion: window.location,
      idiomaPreferido: estado.preferencias.estado?.valores?.idioma, pantalla: "preferencias" }).then(() => {
      estado.errorUsuarios = !montarUsuariosAreaPersonal(estado, estado.fetchImpl, porId("espacio-trabajo"));
      if (estado.vista === "preferencias") renderizar(estado);
    });
  }
  if (accion === "pagina-participaciones") {
    estado.paginaParticipaciones = Math.max(1, Number(boton.dataset.pagina || 1));
    return renderizar(estado, { enfocar: true });
  }
}

function conectarEventos(estado) {
  document.addEventListener("click", (evento) => {
    if (!evento.target.closest(".identidad-cabecera") && !porId("menu-identidad")?.hidden) cerrarMenuIdentidad();
    const enlace = evento.target.closest("[data-ruta]");
    if (enlace) {
      evento.preventDefault();
      return navegar(estado, enlace.dataset.ruta);
    }
    const boton = evento.target.closest("[data-accion]");
    if (boton) return atenderAccion(estado, boton);
  });
  document.addEventListener("submit", (evento) => {
    const formulario = evento.target;
    if (!(formulario instanceof HTMLFormElement)) return;
    if (formulario.method === "dialog") return;
    evento.preventDefault();
    if (formulario.id === "formulario-preferencias") {
      void guardarPreferencias(estado, formulario);
      return;
    }
    if (formulario.dataset.portalMiBolsa) {
      enviarPortalMiBolsa(formulario, { fetchImpl: estado.fetchImpl, alRegistrar: () => cargar(estado) });
      return;
    }
  });
  window.addEventListener("popstate", async () => {
    const secuencia = ++estado.secuenciaNavegacion;
    cerrarMenu();
    cerrarMenuIdentidad();
    const parametros = new URLSearchParams(window.location.search);
    const inicioAjeno = inicioAjenoElegido(estado.preferencias.estado);
    const vista = parametros.has("vista") ? rutaDesdeURL(estado) : inicioAjeno ? "inicio" : "llamamientos";
    if (vista === "preferencias") {
      await iniciarI18nAreaPersonal(document, { ubicacion: window.location,
        idiomaPreferido: estado.preferencias.estado?.valores?.idioma,
        pantalla: "preferencias" });
      if (secuencia !== estado.secuenciaNavegacion) return;
      estado.errorUsuarios = !montarUsuariosAreaPersonal(estado, estado.fetchImpl, porId("espacio-trabajo"));
    }
    estado.vista = vista;
    estado.avisoInicio = !parametros.has("vista") && inicioAjeno;
    if (estado.vista !== "preferencias" && (estado.soloPreferencias || !estado.datos || estado.recargarDatosAlSalirPreferencias)) {
      estado.datos = null;
      estado.soloPreferencias = false;
      estado.recargarDatosAlSalirPreferencias = false;
      void cargar(estado);
      return;
    }
    asegurarShellPreferencias(estado);
    renderizar(estado, { enfocar: true });
  });
  window.addEventListener("keydown", (evento) => {
    mantenerFocoEnMenu(evento);
    if (evento.key !== "Escape") return;
    if (!porId("menu-identidad")?.hidden) {
      evento.preventDefault();
      cerrarMenuIdentidad({ restaurarFoco: true });
      return;
    }
    window.speechSynthesis?.cancel?.();
    if (document.body.dataset.menuAbierto === "true") {
      evento.preventDefault();
      cerrarMenu({ restaurarFoco: true });
    }
  });
}

const inicioAjenoElegido = (e) => Boolean(e && e.version > 0 && e.valores?.inicio !== "bolsas"); // solo elección guardada (versión > 0)
async function cargar(estado) {
  if (asegurarShellPreferencias(estado)) {
    renderizar(estado);
    return;
  }
  estado.destruirHistorialMiBolsa?.();
  estado.destruirHistorialMiBolsa = null;
  const reintento = document.activeElement?.dataset.accion === "reintentar"; porId("estado-carga").hidden = false;
  porId("estado-carga").className = "estado-carga";
  porId("estado-carga").innerHTML = `<span aria-hidden="true"></span>${escaparHTML(traducir("areaPersonal.html.cargandoInformacion"))}`;
  porId("espacio-trabajo").replaceChildren();
  try {
    const respuesta = await estado.cliente.cargar();
    const datos = datosDeRespuesta(respuesta);
    estado.datos = exigirDatosOperativos(datos);
    estado.soloPreferencias = false;
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
    mostrarError(estado, error); if (reintento) porId("espacio-trabajo").querySelector('[data-accion="reintentar"]')?.focus();
  }
}

export async function iniciarAreaPersonal({ cliente, vistasDisponibles, fetchImpl = globalThis.fetch,
  clientePreferencias = null, preferencias = null, errorPreferencias = null, controladorVisual = null } = {}) {
  if (!cliente || typeof cliente.cargar !== "function" || !(vistasDisponibles instanceof Set)) {
    throw new TypeError(t("clienteNoValido"));
  }
  const parametros = new URLSearchParams(window.location.search);
  exigirParametrosConocidos(parametros);
  const inicioAjeno = inicioAjenoElegido(preferencias?.estado);
  const estado = {
    cliente,
    vistasDisponibles,
    datos: null,
    vista: parametros.has("vista") ? rutaDesdeURL({ vistasDisponibles }) : inicioAjeno ? "inicio" : "llamamientos",
    clientePreferencias,
    preferencias: { catalogo: preferencias?.catalogo || null, estado: preferencias?.estado || null,
      error: errorPreferencias, recibo: null, pendiente: null, borrador: null,
      guardando: false, avisoInicio: inicioAjeno },
    controladorVisual,
    soloPreferencias: false,
    avisoInicio: inicioAjeno && !parametros.has("vista"),
    filasPreferidas: preferencias?.estado.valores.filas || 20,
    contactoPropio: null,
    contactoPropioRecibo: null,
    controladorContactoPropio: null,
    destruirContactoPropio: null,
    destruirHistorialMiBolsa: null,
    desmontarOportunidades: null,
    fetchImpl,
    participaciones: [],
    paginaParticipaciones: 1,
    fuenteBolsa: "real",
    causaBolsa: "",
    errorUsuarios: false,
    recargarDatosAlSalirPreferencias: false,
    secuenciaNavegacion: 0,
  };
  ocultarRutasNoDisponibles(estado);
  // Una dirección antigua (p. ej. ?vista=solicitud) se corrige a la vista que se muestra.
  if (parametros.has("vista") && !rutaDisponible(estado, parametros.get("vista"))) {
    window.history.replaceState({ vista: estado.vista }, "", crearURL(estado, estado.vista));
  }
  montarAvatarAreaPersonal(estado, fetchImpl, document);
  if (estado.vista === "preferencias") {
    estado.errorUsuarios = !montarUsuariosAreaPersonal(estado, fetchImpl, porId("espacio-trabajo"));
  }
  conectarEventos(estado);
  sincronizarAtajosVisuales(preferencias?.estado?.valores);
  await cargar(estado);
  return estado;
}
