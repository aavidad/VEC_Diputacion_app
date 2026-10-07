import { validarCapacidades, validarRoles, validarUnidades, validarPersonas, validarFicha, seleccionarActos, prepararDecision, puedeConfirmar, validarResultado, incompatible } from "./contratos.js?v=20261005-admin-lote-pantalla-v1";
import { crearRender } from "./render.js?v=20261005-admin-lote-pantalla-v1";
import { montarPropuestas } from "./propuestas.js?v=20261005-admin-lote-pantalla-v1";
import { montarCambioPerfiles } from "./cambio-perfiles.js?v=20261005-admin-lote-pantalla-v1";
import { validarPreparacion } from "./cambio-contratos.js?v=20261005-admin-lote-pantalla-v1";
import { etiquetaNombreMetadatos } from "./metadatos.js?v=20261005-admin-lote-pantalla-v1";
let montaje = 0;
const filtrosVacios = () => ({ busqueda: "", perfil_ref: "", unidad_ref: "", estado: "", cursor: "" });
export function montarUsuarios(root, { textos, cliente = {}, cripto = globalThis.crypto } = {}) {
  if (!root || typeof textos?.traducir !== "function" || typeof textos.fecha !== "function" || typeof textos.numero !== "function") throw new TypeError("montaje_invalido");
  const prefijo = `admin-usuarios-${++montaje}`;
  const metadatos = cliente.proyeccion === "metadatos_v1";
  const puedeLeer = () => metadatos || capacidades.includes("consultar");
  const { pantalla, filtros, activos, tabla, catalogo, ficha, opciones, revision, resultado, el, t } = crearRender({ root, id: (c) => `${prefijo}-${c}`, textos, metadatos });
  let vivo = true, enviando = false, bloqueado = false, conflicto = false, incierto = false;
  let roles = [], unidades = [], capacidades = [], personas = [], detalle = null, decision = null, consulta = filtrosVacios(), siguiente = "";
  const peticiones = new Map();
  pantalla();
  let actor = "";
  const pendientes = montarPropuestas(el("panel-propuestas"), { textos, cripto,
    contexto: () => ({ roles, capacidades, actor, cliente }),
    bloquear: (valor) => { if (valor) for (const c of peticiones.values()) c.abort(); incierto = valor; controles(!valor && !bloqueado); }, denegar: error, fallarLectura });
  const lenguajePrevio = root.getAttribute("lang");
  root.setAttribute("lang", textos.idioma);
  function iniciar(tipo) {
    peticiones.get(tipo)?.abort();
    const controlador = new AbortController(); peticiones.set(tipo, controlador); return controlador;
  }
  const actual = (tipo, control) => vivo && !control.signal.aborted && peticiones.get(tipo) === control;
  function controles(permitidos) {
    for (const campo of ["consulta", "perfil", "vigencia", "buscar-boton", "limpiar"]) el(campo).disabled = !permitidos;
    if (metadatos) for (const campo of ["consulta", "perfil"]) el(campo).disabled = true;
    el("unidad").disabled = !permitidos || unidades.length === 0;
    el("recargar").disabled = enviando || incierto;
    el("tab-usuarios").disabled = bloqueado || enviando || incierto;
    el("tab-perfiles").disabled = bloqueado || enviando || incierto;
    el("tab-propuestas").disabled = bloqueado || enviando || incierto;
    if (metadatos) for (const campo of ["tab-perfiles", "tab-propuestas"]) el(campo).disabled = true;
  }
  function etapa(nombre) {
    for (const parte of ["listado", "detalle", "revision"]) el(parte).hidden = parte !== nombre;
    root.querySelector(".usuarios-trabajo")?.scrollTo?.({ top: 0 });
    if (nombre !== "listado") el(nombre).focus({ preventScroll: true });
  }
  function mostrarPanel(nombre) {
    for (const parte of ["usuarios", "perfiles", "propuestas"]) {
      el(`panel-${parte}`).hidden = nombre !== parte;
      el(`tab-${parte}`).setAttribute("aria-selected", String(nombre === parte)); el(`tab-${parte}`).tabIndex = nombre === parte ? 0 : -1;
    }
  }
  function limpiarDatos() {
    cerrarCambio();
    detalle = null; decision = null; personas = []; roles = []; unidades = []; capacidades = []; siguiente = "";
    for (const parte of ["resultados", "detalle", "revision", "panel-perfiles", "filtros-activos"]) el(parte).replaceChildren();
    filtros([], []);
    for (const campo of ["consulta", "perfil", "unidad", "vigencia"]) el(campo).value = "";
    mostrarPanel("usuarios"); etapa("listado"); controles(false);
    actor = ""; pendientes.vaciar();
  }
  function fallarLectura(e) {
    bloqueado = true; consulta = filtrosVacios();
    for (const controlador of peticiones.values()) controlador.abort();
    limpiarDatos();
    el("estado").textContent = t([401, 403].includes(e?.estado) ? "errores.denegado" : "errores.lectura_retirada");
    el("recargar").focus();
  }
  function error(error, destino = el("estado")) {
    const denegado = error?.estado === 401 || error?.estado === 403;
    if (denegado) {
      fallarLectura(error);
      return;
    }
    destino.textContent = t(error?.estado === 409 ? "errores.conflicto" : error?.codigo === "filtros_no_disponibles" ? "errores.filtros"
      : error?.codigo === "respuesta_incompatible" ? "errores.incompatible" : "errores.servicio");
  }
  function disponibles(operacion) {
    if (!detalle || detalle.proyeccion === "metadatos_v1") return [];
    return detalle.actos_disponibles.flatMap((acto, indice) => {
      try { return seleccionarActos(detalle, roles, capacidades, operacion, [indice]); } catch { return []; }
    });
  }
  // El cambio de perfiles sólo se ofrece si el servidor monta el lote.
  const puedeCambiar = () => metadatos && typeof cliente.preparar === "function" && typeof cliente.aplicarLote === "function";
  let cambio = null, preparacionFicha = null, promesaPreparacion = null;
  function cerrarCambio() { cambio?.desmontar(); cambio = null; }
  function abrirCambio(retirarPerfilRef = "") {
    if (!detalle || !puedeCambiar() || bloqueado || enviando) return;
    cerrarCambio();
    const ref = detalle.persona_ref;
    etapa("revision");
    cambio = montarCambioPerfiles(el("revision"), { textos, cliente, cripto, persona: ref, nombre: etiquetaNombreMetadatos(detalle, t),
      unidadRef: detalle.unidad_ref, retirarPerfilRef, preparacionInicial: preparacionFicha || promesaPreparacion || undefined,
      alVolver: () => { cerrarCambio(); void cargarPersona(ref); } });
  }
  async function prepararRetiradas() {
    if (!puedeCambiar() || !detalle || detalle.proyeccion !== "metadatos_v1" || detalle.perfiles.length === 0) return;
    preparacionFicha = null; promesaPreparacion = null;
    const actualDetalle = detalle, control = iniciar("retiradas");
    try {
      promesaPreparacion = cliente.preparar(actualDetalle.persona_ref, actualDetalle.unidad_ref, control.signal);
      const datos = await promesaPreparacion;
      if (!actual("retiradas", control) || detalle !== actualDetalle) return;
      const autorizadas = new Set(validarPreparacion(datos, actualDetalle.persona_ref, actualDetalle.unidad_ref).bajas.map((b) => b.perfil_ref));
      preparacionFicha = datos;
      promesaPreparacion = null;
      for (const sitio of el("detalle").querySelectorAll("[data-retirada-ref]")) {
        if (!autorizadas.has(sitio.dataset.retiradaRef) || cambio) continue;
        const boton = root.ownerDocument.createElement("button");
        boton.type = "button"; boton.className = "boton-secundario"; boton.dataset.accion = "retirar-perfil";
        boton.dataset.ref = sitio.dataset.retiradaRef; boton.textContent = t("lote.retirar_este");
        sitio.replaceChildren(boton);
      }
    } catch (e) {
      if (!actual("retiradas", control) || detalle !== actualDetalle || e?.name === "AbortError") return;
      promesaPreparacion = null;
      if (e?.estado === 401) { fallarLectura(e); return; }
      if (e?.estado === 403 || e?.estado === 404) el("detalle").querySelector('[data-accion="cambiar-perfiles"]')?.remove();
      el("estado").textContent = t(e?.estado === 403 ? "lote.errores.denegado" : e?.estado === 404 ? "lote.errores.no_disponible" : "lote.retiradas_no_disponibles");
    }
  }
  function pintarFicha() { cerrarCambio(); ficha(detalle, roles, disponibles("otorgar"), puedeCambiar()); etapa("detalle"); void prepararRetiradas(); }
  async function buscar(anadir = false) {
    if (!vivo || bloqueado || enviando || !puedeLeer() || typeof cliente.buscar !== "function") return;
    peticiones.get("persona")?.abort();
    peticiones.get("retiradas")?.abort(); preparacionFicha = null; promesaPreparacion = null;
    detalle = null; decision = null; el("detalle").replaceChildren(); el("revision").replaceChildren(); etapa("listado");
    const control = iniciar("buscar");
    if (!anadir) { personas = []; siguiente = ""; el("resultados").replaceChildren(); }
    controles(false); el("estado").textContent = t("busqueda.cargando");
    activos(consulta, roles, unidades);
    try {
      const respuesta = await cliente.buscar({ ...consulta, cursor: anadir ? siguiente : "" }, control.signal);
      if (!actual("buscar", control)) return;
      const pagina = validarPersonas(respuesta);
      if (metadatos && pagina.proyeccion !== "metadatos_v1") throw incompatible();
      const nuevas = anadir ? [...personas, ...pagina.personas] : pagina.personas;
      if (nuevas.length > 2000 || new Set(nuevas.map((p) => p.persona_ref)).size !== nuevas.length) throw incompatible();
      personas = nuevas; siguiente = pagina.siguiente_cursor || "";
      tabla({ personas, siguiente_cursor: siguiente }, roles);
      el("estado").textContent = t(personas.length ? "busqueda.lista" : "busqueda.vacia");
    } catch (e) { if (actual("buscar", control)) fallarLectura(e); }
    finally { if (actual("buscar", control)) controles(!bloqueado); }
  }
  async function cargarPersona(ref) {
    if (!vivo || enviando || bloqueado || !puedeLeer() || typeof cliente.persona !== "function") return;
    cerrarCambio();
    peticiones.get("retiradas")?.abort(); preparacionFicha = null; promesaPreparacion = null;
    const control = iniciar("persona");
    detalle = null; decision = null; conflicto = false; el("detalle").replaceChildren(); el("revision").replaceChildren();
    el("estado").textContent = t("detalle.cargando"); etapa("detalle");
    try {
      const respuesta = await cliente.persona(ref, control.signal);
      if (!actual("persona", control)) return;
      detalle = validarFicha(respuesta, ref);
      if (metadatos && detalle.proyeccion !== "metadatos_v1") throw incompatible();
      pintarFicha(); el("estado").textContent = t("detalle.lista");
    } catch (e) { if (actual("persona", control)) fallarLectura(e); }
  }
  async function cargar() {
    if (!vivo || enviando || incierto) return;
    for (const c of peticiones.values()) c.abort();
    bloqueado = false; conflicto = false; limpiarDatos();
    if (!(metadatos ? ["buscar", "persona"] : ["capacidades", "roles", "buscar", "persona"]).every((m) => typeof cliente[m] === "function")) {
      el("estado").textContent = t("errores.sin_conexion"); return;
    }
    const control = iniciar("inicio"); el("estado").textContent = t("busqueda.cargando");
    try {
      if (metadatos) {
        filtros([], []); el("panel-perfiles").textContent = t("metadatos.catalogo_no_consultado");
        el("consulta").setAttribute("aria-describedby", `${prefijo}-estado`);
        controles(true); await buscar(); return;
      }
      const [cap, cat] = await Promise.all([cliente.capacidades(control.signal), cliente.roles(control.signal)]);
      if (!actual("inicio", control)) return;
      const capacidad = validarCapacidades(cap); capacidades = [...capacidad.acciones]; actor = capacidad.actor_persona_ref;
      if (!capacidades.includes("consultar")) throw Object.assign(new Error("denegado"), { estado: 403 });
      roles = validarRoles(cat);
      unidades = validarUnidades(cat);
      filtros(roles, unidades);
      for (const [campo, clave] of [["consulta", "busqueda"], ["perfil", "perfil_ref"], ["unidad", "unidad_ref"], ["vigencia", "estado"]]) el(campo).value = consulta[clave];
      catalogo(roles); controles(true); await buscar();
    } catch (e) { if (actual("inicio", control)) fallarLectura(e); }
  }
  function leerFiltros() {
    return { busqueda: el("consulta").value.trim(), perfil_ref: el("perfil").value, unidad_ref: el("unidad").value, estado: el("vigencia").value, cursor: "" };
  }
  function seleccionar() {
    if (!detalle || enviando || bloqueado || metadatos) return;
    const indices = [...el("opciones").querySelectorAll("[data-seleccion]:checked")].map((n) => Number(n.dataset.seleccion));
    const motivos = Object.fromEntries([...el("opciones").querySelectorAll("[data-motivo]")].filter((n) => n.value !== "").map((n) => [n.dataset.motivo, Number(n.value)]));
    try {
      decision = Object.freeze({ ...prepararDecision(detalle, roles, capacidades, el("operacion").value, indices, motivos, cripto), actor });
      revision(decision, detalle, motivos, Boolean(puedeConfirmar(decision, capacidades, cliente)), unidades); conflicto = false; etapa("revision");
    } catch { el("error-seleccion").hidden = false; el("error-seleccion").textContent = t("errores.seleccion"); el("error-seleccion").focus(); }
  }
  async function confirmar() {
    const metodo = decision && puedeConfirmar(decision, capacidades, cliente);
    if (!vivo || enviando || bloqueado || conflicto || !metodo) return;
    for (const c of peticiones.values()) c.abort();
    pendientes.cancelarLectura();
    enviando = true; incierto = true; controles(false); el("confirmar").disabled = true; el("corregir").disabled = true;
    el("resultado").textContent = t("revision.enviando");
    const copia = decision, control = iniciar("confirmar");
    try {
      const respuesta = await cliente[metodo](copia.cuerpo, control.signal);
      if (!actual("confirmar", control) || decision !== copia) return;
      resultado(validarResultado(respuesta, copia)); decision = null; incierto = false;
      el("resultado").focus();
      el("estado").textContent = t(copia.sensible ? "resultado.propuesta" : "resultado.confirmado");
    } catch (e) {
      if (actual("confirmar", control)) {
        conflicto = e?.estado === 409;
        if ([400, 401, 403, 409, 413, 422].includes(e?.estado)) incierto = false;
        error(e, el("resultado"));
        if (incierto) el("resultado").textContent = t("errores.resultado_pendiente");
      }
    } finally {
      enviando = false;
      if (actual("confirmar", control)) { controles(!bloqueado && !incierto); el("confirmar").disabled = !decision || conflicto; el("corregir").disabled = incierto;
        if (!decision) el("corregir").textContent = t("resultado.ver_ficha"); }
    }
  }
  function pestaña(nombre) {
    if (bloqueado || enviando || incierto || metadatos && nombre !== "usuarios") return;
    if (nombre !== "propuestas") pendientes.cancelarLectura();
    mostrarPanel(nombre);
    el("estado").textContent = nombre === "usuarios" ? t(detalle ? "detalle.lista" : personas.length ? "busqueda.lista" : "busqueda.vacia") : "";
    if (nombre === "propuestas") void pendientes.cargar();
  }
  function click(evento) {
    const boton = evento.target.closest("[data-accion]"); if (!boton || !root.contains(boton) || boton.disabled) return;
    const accion = boton.dataset.accion;
    if (bloqueado && accion !== "recargar" || enviando || incierto && accion !== "confirmar") return;
    if (accion === "cambiar-perfiles") { abrirCambio(); return; }
    if (accion === "retirar-perfil") { abrirCambio(boton.dataset.ref); return; }
    if (["usuarios", "perfiles", "propuestas"].includes(accion)) pestaña(accion);
    else if (accion === "recargar") void cargar();
    else if (accion === "persona") void cargarPersona(boton.dataset.ref);
    else if (accion === "mas" && siguiente) void buscar(true);
    else if (accion === "confirmar") void confirmar();
    else if (accion === "corregir") {
      if (!decision || conflicto) { const ref = detalle?.persona_ref; decision = null; if (ref) void cargarPersona(ref); else etapa("listado"); }
      else { decision = null; pintarFicha(); }
    } else if (accion === "volver") {
      const ref = detalle?.persona_ref;
      peticiones.get("persona")?.abort(); peticiones.get("retiradas")?.abort(); preparacionFicha = null; promesaPreparacion = null;
      detalle = null; decision = null; etapa("listado");
      [...el("resultados").querySelectorAll('[data-accion="persona"]')].find((n) => n.dataset.ref === ref)?.focus();
    }
    else if (accion === "limpiar" || accion === "quitar-filtro") {
      const campos = { busqueda: "consulta", perfil_ref: "perfil", unidad_ref: "unidad", estado: "vigencia" };
      for (const [campo, control] of Object.entries(campos)) if (accion === "limpiar" || campo === boton.dataset.campo) el(control).value = "";
      consulta = leerFiltros(); void buscar();
    }
  }
  function submit(evento) {
    evento.preventDefault();
    if (enviando || incierto || bloqueado) return;
    if (evento.target === el("buscar")) {
      const nueva = leerFiltros();
      if (nueva.busqueda && nueva.busqueda.length < 2) { el("estado").textContent = t("errores.busqueda"); el("consulta").focus(); return; }
      consulta = nueva; void buscar();
    } else if (evento.target === el("seleccion")) seleccionar();
  }
  function change(evento) { if (evento.target === el("operacion") && detalle) {
    el("opciones").innerHTML = opciones(disponibles(el("operacion").value)); el("revisar").disabled = disponibles(el("operacion").value).length === 0; el("error-seleccion").hidden = true;
  } }
  function teclado(evento) {
    if (metadatos) return;
    if (evento.target.getAttribute("role") !== "tab" || !["ArrowLeft", "ArrowRight", "Home", "End"].includes(evento.key) || bloqueado || enviando || incierto) return;
    const nombres = ["usuarios", "perfiles", "propuestas"], indice = nombres.findIndex((n) => evento.target === el(`tab-${n}`));
    evento.preventDefault(); const nombre = evento.key === "Home" ? nombres[0] : evento.key === "End" ? nombres.at(-1) : nombres[(indice + (evento.key === "ArrowRight" ? 1 : 2)) % nombres.length];
    pestaña(nombre); el(`tab-${nombre}`).focus();
  }
  const listeners = [["click", click], ["submit", submit], ["change", change], ["keydown", teclado]];
  listeners.forEach(([tipo, fn]) => root.addEventListener(tipo, fn));
  const ventana = root.ownerDocument?.defaultView;
  const avisarSalida = (evento) => { if (enviando || incierto || cambio?.incierto()) { evento.preventDefault(); evento.returnValue = ""; } };
  ventana?.addEventListener("beforeunload", avisarSalida);
  const listo = cargar();
  return Object.freeze({ listo, cargar, desmontar() { if (!vivo) return; vivo = false;
    pendientes.desmontar(); cerrarCambio();
    for (const c of peticiones.values()) c.abort(); listeners.forEach(([tipo, fn]) => root.removeEventListener(tipo, fn)); root.replaceChildren();
    ventana?.removeEventListener("beforeunload", avisarSalida);
    if (lenguajePrevio === null) root.removeAttribute("lang"); else root.setAttribute("lang", lenguajePrevio);
  } });
}
