import { IDIOMAS_DISPONIBLES, leerRecursoJSON } from "../comun/idioma.js";
import { cargarTextos, crearTextos, urlCatalogo } from "../comun/textos.js";
import { crearClienteAdministracion, nuevaOperacionRef, ErrorAdministracionPerfiles } from "./cliente.js?v=20261002-admin-perfiles-v3";

const FECHA = { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" };
const CLASE_DOBLE = new Set(["administrador", "intervencion"]);
const MOTIVO_CAMPOS = ["catalogo_id", "catalogo_version", "catalogo_huella_sha256", "entrada_clave"];
const REF_RECIBO = /^recibo_admin:[a-f0-9]{32}$/u;
const REF_PROPUESTA = /^propuesta_admin:[a-f0-9]{32}$/u;
const HUELLA = /^[a-f0-9]{64}$/u;

function elemento(documento, nombre, clase = "", contenido = "") {
  const nodo = documento.createElement(nombre);
  if (clase) nodo.className = clase;
  if (contenido) nodo.textContent = contenido;
  return nodo;
}

function textoValido(valor) { return typeof valor === "string" && valor.trim().length > 0; }
function lista(valor) { return Array.isArray(valor) ? valor : []; }
function motivoValido(valor) { return valor && textoValido(valor.catalogo_id) && Number.isSafeInteger(valor.catalogo_version)
  && valor.catalogo_version > 0 && HUELLA.test(valor.catalogo_huella_sha256)
  && textoValido(valor.entrada_clave) && textoValido(valor.etiqueta); }
function motivoParaPOST(valor) { return Object.fromEntries(MOTIVO_CAMPOS.map((clave) => [clave, valor[clave]])); }
function fechaValida(valor) { const fecha = new Date(valor); return Number.isFinite(fecha.getTime()) ? fecha : null; }

export async function cargarTextosAdministracion({
  cargar = cargarTextos, leer = leerRecursoJSON, idiomas = IDIOMAS_DISPONIBLES,
} = {}) {
  try { return await cargar("administracion-perfiles"); }
  catch {
    for (const idioma of idiomas) {
      try {
        const respaldo = await leer(urlCatalogo(idioma.codigo, "administracion-perfiles"));
        return crearTextos({ modulo: "administracion-perfiles", idioma: idioma.codigo,
          localizacion: idioma.localizacion, respaldo });
      } catch { /* Se prueba el siguiente idioma del catálogo, sin inventar textos. */ }
    }
    throw new Error("catálogos de administración no disponibles");
  }
}

export function estadoError(error) {
  if (!Number.isInteger(error?.estado)) return "error_servicio";
  if (error.estado === 401 || error.estado === 403) return "error_denegado";
  if (error.estado === 409) return "error_conflicto";
  if (error.estado === 400) return "error_solicitud";
  return "error_servicio";
}

export function validarCapacidades(datos) {
  if (datos?.version !== "v1" || !Array.isArray(datos.acciones)) return null;
  const conocidas = ["consultar", "aplicar_ordinario", "proponer", "cerrar_propuesta"];
  if (datos.acciones.some((accion) => !conocidas.includes(accion))) return null;
  return new Set(datos.acciones);
}

export function validarConfirmacion(tipo, datos, operacionRef, propuestaRef = "", decision = "") {
  if (tipo === "ordinario") return REF_RECIBO.test(datos?.recibo?.recibo_ref) && datos.recibo.operacion_ref === operacionRef;
  if (tipo === "propuesta") return REF_PROPUESTA.test(datos?.propuesta?.propuesta_ref) && HUELLA.test(datos.propuesta.huella_sha256)
    && fechaValida(datos.propuesta.caduca_en) !== null && datos.propuesta.propuesta_ref === operacionRef;
  if (tipo !== "cierre") return false;
  return datos?.cierre?.operacion_ref === operacionRef && datos.cierre.propuesta_ref === propuestaRef
    && datos.cierre.decision === decision && HUELLA.test(datos.cierre.huella_cierre_sha256)
    && fechaValida(datos.cierre.confirmado_en) !== null
    && (decision === "aprobada" ? REF_RECIBO.test(datos.cierre.recibo?.recibo_ref)
      : decision === "rechazada" && !datos.cierre.recibo);
}

/** Montaje aislado: solo consume capacidades y preimágenes de la API ADMIN. */
export function montarAdministracionPerfiles({ documento = globalThis.document, cliente, textos } = {}) {
  if (!documento || !cliente || !textos?.traducir) throw new TypeError("montaje no disponible");
  const $ = (id) => documento.getElementById(id);
  const t = (clave, variables) => textos.traducir(`general.${clave}`, variables);
  const elementos = {
    conexion: $("estado-conexion"), mensaje: $("mensaje"), buscar: $("busqueda-form"), consulta: $("busqueda-texto"),
    resultados: $("resultados"), ficha: $("ficha"), pendientes: $("pendientes"), recargar: $("pendientes-recargar"),
    ayuda: $("ayuda-contenido"), ayudaBoton: $("ayuda-boton"), dialogo: $("decision-dialogo"),
    decision: $("decision-form"), tituloDecision: $("decision-titulo"), descripcionDecision: $("decision-descripcion"),
    resumenDecision: $("decision-resumen"), errorDecision: $("decision-error"), motivo: $("decision-motivo"), confirmar: $("decision-confirmar"),
    motivoDescripcion: $("decision-motivo-descripcion"), cancelar: $("decision-cancelar"), acto: $("decision-acto"), actoGrupo: $("decision-acto-grupo"),
  };
  if (Object.values(elementos).some((nodo) => !nodo)) throw new TypeError("pantalla incompleta");
  documento.documentElement.lang = textos.idioma;
  documento.title = t("titulo");
  for (const nodo of documento.querySelectorAll("[data-t]")) nodo.textContent = t(nodo.dataset.t);
  for (const nodo of documento.querySelectorAll("[data-t-aria]")) nodo.setAttribute("aria-label", t(nodo.dataset.tAria));

  let activa = true;
  let capacidades = new Set();
  let actorPersonaRef = "";
  let roles = new Map();
  let personaActual = null;
  let fichaActual = null;
  let resultadoActual = [];
  let decisionPendiente = null;
  let peticionBusqueda = null;
  let peticionFicha = null;
  let peticionPendientes = null;
  let peticionInicio = null;
  let peticionDecision = null;

  function mensaje(clave = "", tipo = "") {
    elementos.mensaje.textContent = clave ? t(clave) : "";
    elementos.mensaje.dataset.tipo = tipo;
  }
  function conexion(clave, clase) {
    elementos.conexion.textContent = t(clave);
    elementos.conexion.className = `admin-chip admin-chip--${clase}`;
  }
  function vacio(contenedor, clave) { contenedor.replaceChildren(elemento(documento, "p", "admin-vacio", t(clave))); }
  function errorDialogo(clave = "") {
    elementos.errorDecision.textContent = clave ? t(clave) : "";
    elementos.errorDecision.hidden = !clave;
    if (clave) elementos.errorDecision.focus();
  }
  function tecnico(referencia) {
    if (!textoValido(referencia)) return null;
    const d = elemento(documento, "details", "admin-tecnico");
    d.append(elemento(documento, "summary", "", t("tecnico_ver")), elemento(documento, "code", "", referencia));
    return d;
  }
  function fecha(valor) {
    const instante = fechaValida(valor);
    return instante ? textos.fecha(instante, FECHA) : "";
  }
  function etiquetaRol(ref) { return roles.get(ref)?.etiqueta || t("perfil_generico"); }
  function etiquetaOperacion(operacion, pasada = false) {
    return operacion === "otorgar" ? t(pasada ? "operacion_otorgar_pasada" : "operacion_otorgar")
      : operacion === "revocar" ? t(pasada ? "operacion_revocar_pasada" : "operacion_revocar") : t("operacion_otro");
  }
  function errorDe(error) {
    const clave = estadoError(error);
    mensaje(clave, "error");
    if (error?.estado === 401 || error?.estado === 403) {
      for (const controlador of [peticionInicio, peticionBusqueda, peticionFicha, peticionPendientes]) controlador?.abort();
      capacidades = new Set();
      actorPersonaRef = "";
      roles = new Map();
      personaActual = null;
      fichaActual = null;
      resultadoActual = [];
      vacio(elementos.resultados, "error_denegado");
      vacio(elementos.ficha, "error_denegado");
      vacio(elementos.pendientes, "error_denegado");
      conexion("conexion_denegada", "error");
      elementos.consulta.disabled = true;
      elementos.buscar.querySelector("button[type=submit]").disabled = true;
    } else if (error?.estado === 503 || !error?.estado) conexion("conexion_error", "error");
    elementos.recargar.disabled = false;
    return clave;
  }
  function accionPermitida(accion) { return capacidades.has(accion); }
  function botonAccion(clave, alPulsar, clase = "admin-boton admin-boton--secundario") {
    const boton = elemento(documento, "button", clase, t(clave));
    boton.type = "button";
    boton.addEventListener("click", alPulsar);
    return boton;
  }

  function pintarResultados(personas, hayMas) {
    resultadoActual = personas;
    if (!personas.length) return vacio(elementos.resultados, "buscar_vacio");
    const ul = elemento(documento, "ul", "admin-lista");
    for (const persona of personas) {
      if (!textoValido(persona?.persona_ref) || !textoValido(persona.nombre)) continue;
      const li = elemento(documento, "li");
      const boton = elemento(documento, "button", "admin-persona");
      boton.type = "button";
      boton.dataset.personaRef = persona.persona_ref;
      boton.setAttribute("aria-current", String(persona.persona_ref === personaActual));
      boton.append(elemento(documento, "strong", "", persona.nombre));
      if (textoValido(persona.unidad_nombre)) boton.append(elemento(documento, "small", "", persona.unidad_nombre));
      boton.addEventListener("click", () => cargarPersona(persona.persona_ref));
      li.append(boton);
      ul.append(li);
    }
    elementos.resultados.replaceChildren(ul);
    if (hayMas) elementos.resultados.append(elemento(documento, "p", "admin-limite", t("buscar_mas")));
  }

  function pintarFicha(datos) {
    fichaActual = datos;
    const raiz = elementos.ficha;
    raiz.replaceChildren();
    const persona = elemento(documento, "div", "admin-ficha-persona");
    persona.append(elemento(documento, "strong", "", datos.nombre));
    if (textoValido(datos.unidad_nombre)) persona.append(elemento(documento, "small", "", datos.unidad_nombre));
    raiz.append(persona, elemento(documento, "h3", "admin-subtitulo", t("perfiles_titulo")));
    const perfiles = lista(datos.perfiles);
    if (!perfiles.length) raiz.append(elemento(documento, "p", "admin-vacio", t("ficha_sin_perfiles")));
    else {
      const ul = elemento(documento, "ul", "admin-lista");
      for (const perfil of perfiles) {
        const li = elemento(documento, "li", "admin-perfil");
        li.dataset.estado = perfil.estado;
        const cabecera = elemento(documento, "div", "admin-perfil-cabecera");
        cabecera.append(elemento(documento, "strong", "", etiquetaRol(perfil.rol_version_ref)));
        const estado = perfil.estado === "activo" ? "perfil_activo" : perfil.estado === "revocado" ? "perfil_revocado" : "perfil_estado_otro";
        cabecera.append(elemento(documento, "span", `admin-chip admin-chip--${perfil.estado === "activo" ? "exito" : "neutro"}`, t(estado)));
        li.append(cabecera);
        if (fecha(perfil.vigente_hasta)) li.append(elemento(documento, "p", "", t("perfil_vigente_hasta", { fecha: fecha(perfil.vigente_hasta) })));
        const detalle = tecnico(perfil.perfil_ref);
        if (detalle) li.append(detalle);
        ul.append(li);
      }
      raiz.append(ul);
    }
    const acciones = lista(datos.actos_disponibles).filter((acto) => {
      const rol = roles.get(acto?.rol_version_ref);
      const permiso = rol?.clase === "ordinario" ? "aplicar_ordinario" : CLASE_DOBLE.has(rol?.clase) ? "proponer" : "";
      return permiso && accionPermitida(permiso) && ["otorgar", "revocar"].includes(acto.operacion)
        && acto.objetivo?.persona_ref === datos.persona_ref && Array.isArray(acto.motivos) && acto.motivos.some(motivoValido)
        && (acto.operacion !== "revocar" || perfiles.some((perfil) => perfil.perfil_ref === acto.objetivo.perfil_ref
          && perfil.rol_version_ref === acto.rol_version_ref && perfil.estado === "activo"));
    });
    if (acciones.length) {
      const grupo = elemento(documento, "div", "admin-acciones");
      for (const acto of acciones) {
        const boton = botonAccion(`operacion_${acto.operacion}`, () => abrirDecision({ tipo: roles.get(acto.rol_version_ref).clase === "ordinario" ? "ordinario" : "propuesta", acto, persona: datos }));
        boton.textContent = t("accion_perfil", { accion: etiquetaOperacion(acto.operacion), perfil: etiquetaRol(acto.rol_version_ref) });
        grupo.append(boton);
      }
      raiz.append(grupo);
    } else raiz.append(elemento(documento, "p", "admin-limite", t(accionPermitida("aplicar_ordinario") || accionPermitida("proponer") ? "accion_no_disponible" : "lectura_sin_cambio")));
    raiz.append(elemento(documento, "h3", "admin-subtitulo", t("historia_titulo")));
    const historia = lista(datos.historia);
    if (!historia.length) raiz.append(elemento(documento, "p", "admin-vacio", t("ficha_sin_historia")));
    else {
      const ul = elemento(documento, "ul", "admin-lista");
      for (const item of historia) {
        const li = elemento(documento, "li", "admin-historia-item", etiquetaOperacion(item.operacion, true));
        if (fecha(item.confirmado_en)) {
          const time = elemento(documento, "time", "", fecha(item.confirmado_en));
          time.dateTime = item.confirmado_en;
          li.append(time);
        }
        const detalle = tecnico(item.acto_ref);
        if (detalle) li.append(detalle);
        ul.append(li);
      }
      raiz.append(ul);
    }
  }

  function pintarPendientes(propuestas) {
    if (!propuestas.length) return vacio(elementos.pendientes, "pendientes_vacio");
    const ul = elemento(documento, "ul", "admin-lista");
    for (const propuesta of propuestas) {
      if (!textoValido(propuesta?.propuesta_ref)) continue;
      const li = elemento(documento, "li", "admin-propuesta");
      li.append(elemento(documento, "strong", "", `${etiquetaOperacion(propuesta.operacion)} · ${etiquetaRol(propuesta.rol_version_ref)}`));
      if (textoValido(propuesta.objetivo_nombre)) li.append(elemento(documento, "p", "", t("pendientes_objetivo", { nombre: propuesta.objetivo_nombre })));
      if (fecha(propuesta.caduca_en)) li.append(elemento(documento, "p", "", t("pendientes_caduca", { fecha: fecha(propuesta.caduca_en) })));
      const detalle = tecnico(propuesta.propuesta_ref);
      if (detalle) li.append(detalle);
      const motivos = lista(propuesta.motivos_cierre).filter(motivoValido);
      if (accionPermitida("cerrar_propuesta") && propuesta.puede_cerrar === true && motivos.length
        && textoValido(propuesta.objetivo_nombre) && textoValido(actorPersonaRef)
        && propuesta.proponente_persona_ref !== actorPersonaRef && propuesta.objetivo_persona_ref !== actorPersonaRef) {
        const acciones = elemento(documento, "div", "admin-acciones");
        acciones.append(
          botonAccion("aprobar", () => abrirDecision({ tipo: "cierre", propuesta, decision: "aprobada" })),
          botonAccion("rechazar", () => abrirDecision({ tipo: "cierre", propuesta, decision: "rechazada" })),
        );
        li.append(acciones);
      } else li.append(elemento(documento, "p", "", t("pendientes_sin_accion")));
      ul.append(li);
    }
    elementos.pendientes.replaceChildren(ul);
  }

  function abrirDecision(seleccion) {
    if (!activa || decisionPendiente) return;
    const rolRef = seleccion.tipo === "cierre" ? seleccion.propuesta.rol_version_ref : seleccion.acto.rol_version_ref;
    const rol = roles.get(rolRef);
    const motivos = lista(seleccion.tipo === "cierre" ? seleccion.propuesta.motivos_cierre : seleccion.acto.motivos).filter(motivoValido);
    if (!rol || !motivos.length || seleccion.tipo === "cierre" && (seleccion.propuesta.puede_cerrar !== true
      || !textoValido(seleccion.propuesta.objetivo_nombre) || seleccion.propuesta.proponente_persona_ref === actorPersonaRef
      || seleccion.propuesta.objetivo_persona_ref === actorPersonaRef)) return;
    const prefijo = seleccion.tipo === "ordinario" ? "acto_admin:" : seleccion.tipo === "propuesta" ? "propuesta_admin:" : "cierre_admin:";
    decisionPendiente = { ...seleccion, motivos, operacionRef: nuevaOperacionRef(prefijo) };
    errorDialogo();
    elementos.tituloDecision.textContent = t("decision_titulo");
    elementos.descripcionDecision.textContent = t(seleccion.tipo === "propuesta" ? "decision_descripcion_doble" : seleccion.tipo === "cierre" ? "decision_descripcion_cierre" : "decision_descripcion");
    const operacion = seleccion.tipo === "cierre" ? seleccion.decision === "aprobada" ? t("aprobar") : t("rechazar") : etiquetaOperacion(seleccion.acto.operacion);
    const persona = seleccion.tipo === "cierre" ? seleccion.propuesta.objetivo_nombre : seleccion.persona.nombre;
    const pares = [["decision_persona", persona], ["decision_perfil", rol.etiqueta || t("perfil_generico")], ["decision_accion", operacion]];
    elementos.resumenDecision.replaceChildren();
    if (seleccion.tipo !== "cierre" && seleccion.persona.unidad_nombre) pares.push(["decision_unidad", seleccion.persona.unidad_nombre]);
    for (const [clave, valor] of pares) elementos.resumenDecision.append(elemento(documento, "dt", "", t(clave)), elemento(documento, "dd", "", valor));
    elementos.motivo.replaceChildren();
    const elegir = elemento(documento, "option", "", t("motivo_elegir"));
    elegir.value = "";
    elementos.motivo.append(elegir);
    motivos.forEach((motivo, indice) => {
      const opcion = elemento(documento, "option", "", motivo.etiqueta);
      opcion.value = String(indice);
      elementos.motivo.append(opcion);
    });
    elementos.motivo.value = "";
    elementos.motivoDescripcion.textContent = "";
    elementos.motivo.disabled = false;
    elementos.acto.value = "";
    elementos.acto.disabled = false;
    elementos.actoGrupo.hidden = seleccion.tipo === "cierre";
    elementos.confirmar.textContent = t(seleccion.tipo === "propuesta" ? "confirmar_propuesta" : seleccion.tipo === "cierre" ? "confirmar_cierre" : "confirmar");
    elementos.confirmar.classList.toggle("admin-boton--peligro", seleccion.tipo === "ordinario" && seleccion.acto.operacion === "revocar" || seleccion.tipo === "cierre" && seleccion.decision === "rechazada");
    elementos.confirmar.disabled = false;
    elementos.dialogo.showModal();
    elementos.motivo.focus();
  }

  function cerrarDialogo() {
    if (peticionDecision) return;
    if (elementos.dialogo.open) elementos.dialogo.close();
    decisionPendiente = null;
    errorDialogo();
  }

  async function registrarDecision(evento) {
    evento.preventDefault();
    if (!decisionPendiente || peticionDecision || !elementos.motivo.value) return;
    const seleccion = decisionPendiente;
    const motivo = seleccion.motivo ?? seleccion.motivos[Number(elementos.motivo.value)];
    if (!motivoValido(motivo)) return;
    let referenciaActo;
    try { referenciaActo = seleccion.referenciaActo ?? referenciaActoParaPOST(elementos.acto.value); }
    catch { errorDialogo("error_acto"); elementos.acto.focus(); return; }
    seleccion.referenciaActo = referenciaActo;
    seleccion.motivo = motivo;
    elementos.motivo.disabled = true;
    elementos.acto.disabled = true;
    const controlador = new AbortController();
    peticionDecision = controlador;
    elementos.confirmar.disabled = true;
    elementos.cancelar.disabled = true;
    mensaje("guardando");
    let resultadoIncierto = false;
    errorDialogo();
    try {
      let resultado;
      if (seleccion.tipo === "cierre") {
        if (!accionPermitida("cerrar_propuesta") || seleccion.propuesta.puede_cerrar !== true) throw new ErrorAdministracionPerfiles(403);
        resultado = await cliente.cerrar(seleccion.propuesta.propuesta_ref, {
          operacion_ref: seleccion.operacionRef, propuesta_huella_sha256: seleccion.propuesta.huella_sha256,
          decision: seleccion.decision, motivo: motivoParaPOST(motivo),
        }, controlador.signal);
      } else {
        const accion = seleccion.tipo === "ordinario" ? "aplicar_ordinario" : "proponer";
        if (!accionPermitida(accion)) throw new ErrorAdministracionPerfiles(403);
        const cuerpo = { operacion_ref: seleccion.operacionRef, operacion: seleccion.acto.operacion,
          rol_version_ref: seleccion.acto.rol_version_ref, objetivo: seleccion.acto.objetivo, motivo: motivoParaPOST(motivo),
          ...(referenciaActo ? { referencia_acto: referenciaActo } : {}) };
        resultado = await (seleccion.tipo === "ordinario" ? cliente.aplicar(cuerpo, controlador.signal) : cliente.proponer(cuerpo, controlador.signal));
      }
      if (!validarConfirmacion(seleccion.tipo, resultado, seleccion.operacionRef, seleccion.propuesta?.propuesta_ref, seleccion.decision)) {
        resultadoIncierto = true;
        mensaje("error_respuesta", "error");
        errorDialogo("error_respuesta");
        return;
      }
      const ref = seleccion.tipo === "ordinario" ? resultado.recibo.recibo_ref : seleccion.tipo === "propuesta" ? resultado.propuesta.propuesta_ref : resultado.cierre.recibo?.recibo_ref || resultado.cierre.operacion_ref;
      const claveRef = seleccion.tipo === "ordinario" || seleccion.tipo === "cierre" && resultado.cierre.recibo ? "recibo_referencia"
        : seleccion.tipo === "propuesta" ? "propuesta_referencia" : "decision_referencia";
      peticionDecision = null;
      cerrarDialogo();
      mensaje(seleccion.tipo === "ordinario" ? "recibo_titulo" : seleccion.tipo === "propuesta" ? "propuesta_confirmada" : "cierre_confirmado", "exito");
      if (ref) elementos.mensaje.append(elemento(documento, "span", "admin-recibo", ` ${t(claveRef, { referencia: ref })}`));
      elementos.mensaje.focus();
      if (personaActual) cargarPersona(personaActual);
      cargarPendientes();
    } catch (error) {
      if (error?.name !== "AbortError") {
        const clave = errorDe(error);
        if ([401, 403, 409].includes(error?.estado)) {
          peticionDecision = null;
          cerrarDialogo();
          elementos.mensaje.focus();
        } else errorDialogo(clave === "error_servicio" ? "error_servicio_dialogo" : clave);
        if (error?.estado === 409) {
          if (personaActual) cargarPersona(personaActual);
          cargarPendientes();
        }
      }
    } finally {
      peticionDecision = null;
      elementos.cancelar.disabled = false;
      if (decisionPendiente) elementos.confirmar.disabled = resultadoIncierto;
    }
  }

  async function cargarPersona(ref) {
    peticionFicha?.abort();
    const controlador = new AbortController();
    peticionFicha = controlador;
    personaActual = ref;
    fichaActual = null;
    vacio(elementos.ficha, "ficha_cargando");
    for (const boton of elementos.resultados.querySelectorAll(".admin-persona")) boton.setAttribute("aria-current", String(boton.dataset.personaRef === ref));
    try {
      const datos = await cliente.persona(ref, controlador.signal);
      if (!activa || controlador.signal.aborted) return;
      if (datos.persona_ref !== ref || !textoValido(datos.nombre) || !Array.isArray(datos.perfiles) || !Array.isArray(datos.historia)) throw new ErrorAdministracionPerfiles();
      pintarFicha(datos);
    } catch (error) { if (error?.name !== "AbortError" && activa) { errorDe(error); vacio(elementos.ficha, estadoError(error)); } }
    finally { if (peticionFicha === controlador) peticionFicha = null; }
  }

  async function buscar(evento) {
    evento?.preventDefault();
    const consulta = elementos.consulta.value.trim();
    if (!accionPermitida("consultar")) return;
    if (consulta.length < 2 || consulta.length > 80) { mensaje("buscar_corto", "error"); elementos.consulta.focus(); return; }
    peticionBusqueda?.abort();
    peticionFicha?.abort();
    personaActual = null;
    fichaActual = null;
    vacio(elementos.ficha, "ficha_inicial");
    const controlador = new AbortController();
    peticionBusqueda = controlador;
    vacio(elementos.resultados, "buscar_cargando");
    try {
      const datos = await cliente.buscar(consulta, controlador.signal);
      if (!activa || controlador.signal.aborted) return;
      if (!Array.isArray(datos.personas)) throw new ErrorAdministracionPerfiles();
      pintarResultados(datos.personas, Boolean(datos.siguiente_cursor));
    } catch (error) { if (error?.name !== "AbortError" && activa) { errorDe(error); vacio(elementos.resultados, estadoError(error)); } }
    finally { if (peticionBusqueda === controlador) peticionBusqueda = null; }
  }

  async function cargarPendientes() {
    if (!accionPermitida("consultar")) return;
    peticionPendientes?.abort();
    const controlador = new AbortController();
    peticionPendientes = controlador;
    vacio(elementos.pendientes, "pendientes_cargando");
    try {
      const datos = await cliente.propuestas(controlador.signal);
      if (!activa || controlador.signal.aborted) return;
      if (!Array.isArray(datos.propuestas)) throw new ErrorAdministracionPerfiles();
      pintarPendientes(datos.propuestas);
    } catch (error) { if (error?.name !== "AbortError" && activa) { errorDe(error); vacio(elementos.pendientes, estadoError(error)); } }
    finally { if (peticionPendientes === controlador) peticionPendientes = null; }
  }

  async function iniciar() {
    conexion("conexion_cargando", "neutro");
    vacio(elementos.resultados, "buscar_inicial");
    vacio(elementos.ficha, "ficha_inicial");
    vacio(elementos.pendientes, "pendientes_inicial");
    elementos.consulta.disabled = true;
    elementos.buscar.querySelector("button[type=submit]").disabled = true;
    elementos.recargar.disabled = true;
    const controlador = new AbortController();
    peticionInicio = controlador;
    try {
      const acceso = await cliente.capacidades(controlador.signal);
      if (!activa || controlador.signal.aborted) return;
      const verificadas = validarCapacidades(acceso);
      if (!verificadas?.has("consultar")) throw new ErrorAdministracionPerfiles(403, "acceso_denegado");
      if (!textoValido(acceso.actor_persona_ref)) throw new ErrorAdministracionPerfiles(403, "acceso_denegado");
      const datosRoles = await cliente.roles(controlador.signal);
      if (!activa || controlador.signal.aborted) return;
      if (!Array.isArray(datosRoles.roles)) throw new ErrorAdministracionPerfiles();
      if (datosRoles.roles.some((rol) => !textoValido(rol.version_ref) || !textoValido(rol.etiqueta)
        || !["ordinario", "administrador", "intervencion"].includes(rol.clase))) throw new ErrorAdministracionPerfiles();
      roles = new Map(datosRoles.roles.map((rol) => [rol.version_ref, rol]));
      if (roles.size !== datosRoles.roles.length) throw new ErrorAdministracionPerfiles();
      capacidades = verificadas;
      actorPersonaRef = acceso.actor_persona_ref;
      conexion(capacidades.size > 1 ? "conexion_lista" : "conexion_lectura", capacidades.size > 1 ? "exito" : "aviso");
      elementos.consulta.disabled = false;
      elementos.buscar.querySelector("button[type=submit]").disabled = false;
      elementos.recargar.disabled = false;
      await cargarPendientes();
    } catch (error) { if (error?.name !== "AbortError" && activa) errorDe(error); }
    finally { if (peticionInicio === controlador) peticionInicio = null; }
  }

  elementos.buscar.addEventListener("submit", buscar);
  elementos.recargar.addEventListener("click", () => { if (accionPermitida("consultar")) cargarPendientes(); else iniciar(); });
  elementos.ayudaBoton.addEventListener("click", () => {
    const abierto = elementos.ayuda.hidden;
    elementos.ayuda.hidden = !abierto;
    elementos.ayudaBoton.setAttribute("aria-expanded", String(abierto));
  });
  elementos.cancelar.addEventListener("click", cerrarDialogo);
  elementos.dialogo.addEventListener("cancel", (evento) => { if (peticionDecision) evento.preventDefault(); else decisionPendiente = null; });
  elementos.motivo.addEventListener("change", () => {
    const motivo = decisionPendiente?.motivos?.[Number(elementos.motivo.value)];
    elementos.motivoDescripcion.textContent = elementos.motivo.value && motivoValido(motivo) ? motivo.etiqueta : "";
  });
  elementos.decision.addEventListener("submit", registrarDecision);
  iniciar();
  return Object.freeze({
    destruir() {
      activa = false;
      for (const controlador of [peticionInicio, peticionBusqueda, peticionFicha, peticionPendientes, peticionDecision]) controlador?.abort();
    },
  });
}

if (typeof document !== "undefined" && document.getElementById("contenido")) {
  cargarTextosAdministracion().then((textos) => {
    const cliente = crearClienteAdministracion();
    const vista = montarAdministracionPerfiles({ cliente, textos });
    globalThis.addEventListener("pagehide", () => vista.destruir(), { once: true });
  }).catch(() => { document.getElementById("estado-conexion").textContent = ""; });
}

// La referencia se conserva con la misma operación al recuperar una respuesta.
export function referenciaActoParaPOST(valor = "") {
 if (typeof valor !== "string") throw new TypeError("referencia inválida");
 const ref = valor.trim();
 if (Array.from(ref).length > 256 || /[\p{Cc}\p{Zl}\p{Zp}]/u.test(ref)) throw new TypeError("referencia inválida");
 return ref;
}
