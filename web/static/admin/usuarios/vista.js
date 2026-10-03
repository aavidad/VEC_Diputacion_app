import { validarCapacidades, validarRoles, validarUnidades, validarPersonas, validarFicha, seleccionarActos, prepararDecision, puedeConfirmar, validarResultado, incompatible } from "./contratos.js?v=20261003-admin-usuarios-v1";
import { crearRender } from "./render.js?v=20261003-admin-usuarios-v1";
let montaje = 0;
const filtrosVacios = () => ({ busqueda: "", perfil_ref: "", unidad_ref: "", estado: "", cursor: "" });
export function montarUsuarios(root, { textos, cliente = {}, cripto = globalThis.crypto } = {}) {
  if (!root || typeof textos?.traducir !== "function" || typeof textos.fecha !== "function" || typeof textos.numero !== "function") throw new TypeError("montaje_invalido");
  const prefijo = `admin-usuarios-${++montaje}`;
  const { pantalla, filtros, activos, tabla, catalogo, ficha, opciones, revision, resultado, el, t } = crearRender({ root, id: (c) => `${prefijo}-${c}`, textos });
  let vivo = true, enviando = false, bloqueado = false, conflicto = false;
  let roles = [], unidades = [], capacidades = [], personas = [], detalle = null, decision = null, consulta = filtrosVacios(), siguiente = "";
  const peticiones = new Map();
  pantalla();
  const lenguajePrevio = root.getAttribute("lang");
  root.setAttribute("lang", textos.idioma);
  function iniciar(tipo) {
    peticiones.get(tipo)?.abort();
    const controlador = new AbortController(); peticiones.set(tipo, controlador); return controlador;
  }
  const actual = (tipo, control) => vivo && !control.signal.aborted && peticiones.get(tipo) === control;
  function controles(permitidos) {
    for (const campo of ["consulta", "perfil", "vigencia", "buscar-boton"]) el(campo).disabled = !permitidos;
    el("unidad").disabled = !permitidos || unidades.length === 0;
    el("recargar").disabled = enviando;
    el("tab-usuarios").disabled = enviando;
    el("tab-perfiles").disabled = enviando;
  }
  function etapa(nombre) {
    for (const parte of ["listado", "detalle", "revision"]) el(parte).hidden = parte !== nombre;
    root.querySelector(".usuarios-trabajo")?.scrollTo?.({ top: 0 });
    if (nombre !== "listado") el(nombre).focus({ preventScroll: true });
  }
  function limpiarDatos() {
    detalle = null; decision = null; personas = []; roles = []; unidades = []; capacidades = []; siguiente = "";
    for (const parte of ["resultados", "detalle", "revision", "panel-perfiles", "filtros-activos"]) el(parte).replaceChildren();
    filtros([], []); etapa("listado"); controles(false);
  }
  function error(error, destino = el("estado")) {
    const denegado = error?.estado === 401 || error?.estado === 403;
    if (denegado) {
      bloqueado = true;
      for (const controlador of peticiones.values()) controlador.abort();
      limpiarDatos();
      el("estado").textContent = t("errores.denegado");
      return;
    }
    destino.textContent = t(error?.estado === 409 ? "errores.conflicto" : error?.codigo === "filtros_no_disponibles" ? "errores.filtros"
      : error?.codigo === "respuesta_incompatible" ? "errores.incompatible" : "errores.servicio");
  }
  function disponibles(operacion) {
    if (!detalle) return [];
    return detalle.actos_disponibles.flatMap((acto, indice) => {
      try { return seleccionarActos(detalle, roles, capacidades, operacion, [indice]); } catch { return []; }
    });
  }
  function pintarFicha() { ficha(detalle, roles, disponibles("otorgar")); etapa("detalle"); }
  async function buscar(anadir = false) {
    if (!vivo || bloqueado || enviando || !capacidades.includes("consultar") || typeof cliente.buscar !== "function") return;
    const control = iniciar("buscar");
    if (!anadir) { personas = []; siguiente = ""; el("resultados").replaceChildren(); }
    controles(false); el("estado").textContent = t("busqueda.cargando");
    activos(consulta, roles, unidades);
    try {
      const respuesta = await cliente.buscar({ ...consulta, cursor: anadir ? siguiente : "" }, control.signal);
      if (!actual("buscar", control)) return;
      const pagina = validarPersonas(respuesta);
      const nuevas = anadir ? [...personas, ...pagina.personas] : pagina.personas;
      if (nuevas.length > 2000 || new Set(nuevas.map((p) => p.persona_ref)).size !== nuevas.length) throw incompatible();
      personas = nuevas; siguiente = pagina.siguiente_cursor || "";
      tabla({ personas, siguiente_cursor: siguiente }, roles);
      el("estado").textContent = t(personas.length ? "busqueda.lista" : "busqueda.vacia");
    } catch (e) { if (actual("buscar", control)) { personas = []; siguiente = ""; el("resultados").replaceChildren(); error(e); } }
    finally { if (actual("buscar", control)) controles(!bloqueado); }
  }
  async function cargarPersona(ref) {
    if (!vivo || enviando || bloqueado || !capacidades.includes("consultar") || typeof cliente.persona !== "function") return;
    const control = iniciar("persona");
    detalle = null; decision = null; conflicto = false; el("detalle").replaceChildren(); el("revision").replaceChildren();
    el("estado").textContent = t("detalle.cargando"); etapa("detalle");
    try {
      const respuesta = await cliente.persona(ref, control.signal);
      if (!actual("persona", control)) return;
      detalle = validarFicha(respuesta, ref); pintarFicha(); el("estado").textContent = t("detalle.lista");
    } catch (e) { if (actual("persona", control)) { etapa("listado"); error(e); } }
  }
  async function cargar() {
    if (!vivo || enviando) return;
    for (const c of peticiones.values()) c.abort();
    bloqueado = false; conflicto = false; limpiarDatos();
    if (!["capacidades", "roles", "buscar", "persona"].every((m) => typeof cliente[m] === "function")) {
      el("estado").textContent = t("errores.sin_conexion"); return;
    }
    const control = iniciar("inicio"); el("estado").textContent = t("busqueda.cargando");
    try {
      const [cap, cat] = await Promise.all([cliente.capacidades(control.signal), cliente.roles(control.signal)]);
      if (!actual("inicio", control)) return;
      capacidades = [...validarCapacidades(cap).acciones];
      if (!capacidades.includes("consultar")) throw Object.assign(new Error("denegado"), { estado: 403 });
      roles = validarRoles(cat);
      unidades = validarUnidades(cat);
      filtros(roles, unidades);
      for (const [campo, clave] of [["consulta", "busqueda"], ["perfil", "perfil_ref"], ["unidad", "unidad_ref"], ["vigencia", "estado"]]) el(campo).value = consulta[clave];
      catalogo(roles); controles(true); await buscar();
    } catch (e) { if (actual("inicio", control)) { limpiarDatos(); error(e); } }
  }
  function leerFiltros() {
    return { busqueda: el("consulta").value.trim(), perfil_ref: el("perfil").value, unidad_ref: el("unidad").value, estado: el("vigencia").value, cursor: "" };
  }
  function seleccionar() {
    if (!detalle || enviando || bloqueado) return;
    const indices = [...el("opciones").querySelectorAll("[data-seleccion]:checked")].map((n) => Number(n.dataset.seleccion));
    const motivos = Object.fromEntries([...el("opciones").querySelectorAll("[data-motivo]")].filter((n) => n.value !== "").map((n) => [n.dataset.motivo, Number(n.value)]));
    try {
      decision = prepararDecision(detalle, roles, capacidades, el("operacion").value, indices, motivos, cripto);
      revision(decision, detalle, motivos, Boolean(puedeConfirmar(decision, capacidades, cliente))); conflicto = false; etapa("revision");
    } catch { el("error-seleccion").hidden = false; el("error-seleccion").textContent = t("errores.seleccion"); el("error-seleccion").focus(); }
  }
  async function confirmar() {
    const metodo = decision && puedeConfirmar(decision, capacidades, cliente);
    if (!vivo || enviando || bloqueado || conflicto || !metodo) return;
    enviando = true; controles(false); el("confirmar").disabled = true; el("corregir").disabled = true;
    el("resultado").textContent = t("revision.enviando");
    const copia = decision, control = iniciar("confirmar");
    try {
      const respuesta = await cliente[metodo](copia.cuerpo, control.signal);
      if (!actual("confirmar", control) || decision !== copia) return;
      resultado(validarResultado(respuesta, copia)); decision = null;
      el("resultado").focus();
      el("estado").textContent = t(copia.sensible ? "resultado.propuesta" : "resultado.confirmado");
    } catch (e) {
      if (actual("confirmar", control)) { conflicto = e?.estado === 409; error(e, el("resultado")); }
    } finally {
      enviando = false;
      if (actual("confirmar", control)) { controles(!bloqueado); el("confirmar").disabled = !decision || conflicto; el("corregir").disabled = false;
        if (!decision) el("corregir").textContent = t("resultado.ver_ficha"); }
    }
  }
  function pestaña(nombre) {
    if (enviando) return;
    for (const parte of ["usuarios", "perfiles"]) {
      el(`panel-${parte}`).hidden = nombre !== parte;
      el(`tab-${parte}`).setAttribute("aria-selected", String(nombre === parte)); el(`tab-${parte}`).tabIndex = nombre === parte ? 0 : -1;
    }
  }
  function click(evento) {
    const boton = evento.target.closest("[data-accion]"); if (!boton || !root.contains(boton) || boton.disabled) return;
    const accion = boton.dataset.accion;
    if (["usuarios", "perfiles"].includes(accion)) pestaña(accion);
    else if (accion === "recargar") void cargar();
    else if (accion === "persona") void cargarPersona(boton.dataset.ref);
    else if (accion === "mas" && siguiente) void buscar(true);
    else if (accion === "confirmar") void confirmar();
    else if (accion === "corregir") {
      if (!decision || conflicto) { const ref = detalle?.persona_ref; decision = null; if (ref) void cargarPersona(ref); else etapa("listado"); }
      else { decision = null; pintarFicha(); }
    } else if (accion === "volver") { peticiones.get("persona")?.abort(); detalle = null; decision = null; etapa("listado"); }
    else if (accion === "limpiar" || accion === "quitar-filtro") {
      const campos = { busqueda: "consulta", perfil_ref: "perfil", unidad_ref: "unidad", estado: "vigencia" };
      for (const [campo, control] of Object.entries(campos)) if (accion === "limpiar" || campo === boton.dataset.campo) el(control).value = "";
      consulta = leerFiltros(); void buscar();
    }
  }
  function submit(evento) {
    evento.preventDefault();
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
    if (evento.target.getAttribute("role") !== "tab" || !["ArrowLeft", "ArrowRight", "Home", "End"].includes(evento.key) || enviando) return;
    evento.preventDefault(); const nombre = evento.key === "Home" ? "usuarios" : evento.key === "End" ? "perfiles" : evento.target === el("tab-usuarios") ? "perfiles" : "usuarios";
    pestaña(nombre); el(`tab-${nombre}`).focus();
  }
  const listeners = [["click", click], ["submit", submit], ["change", change], ["keydown", teclado]];
  listeners.forEach(([tipo, fn]) => root.addEventListener(tipo, fn));
  const listo = cargar();
  return Object.freeze({ listo, cargar, desmontar() { if (!vivo) return; vivo = false;
    for (const c of peticiones.values()) c.abort(); listeners.forEach(([tipo, fn]) => root.removeEventListener(tipo, fn)); root.replaceChildren();
    if (lenguajePrevio === null) root.removeAttribute("lang"); else root.setAttribute("lang", lenguajePrevio);
  } });
}
