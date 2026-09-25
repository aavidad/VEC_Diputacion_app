import { crearTraductorPersonal } from "./i18n.js?v=20260925-personal-real-v1";
import { ErrorRegistroB2 } from "./registro-b2-cliente.js?v=20260925-b2-selector-v1";

const EMPLEADO = /^emp_[A-Za-z0-9_-]{22,128}$/u;
const PERSONA = /^per_[A-Za-z0-9_-]{22,128}$/u;
const TIPOS = Object.freeze({ alta: "registro_b2_alta", relacion: "registro_b2_nueva_relacion", revision_relacion: "registro_b2_revisar_relacion", ocupacion: "registro_b2_nueva_ocupacion", situacion: "registro_b2_nueva_situacion", servicio: "registro_b2_nuevo_servicio" });
const OPCIONES = Object.freeze(["organismos", "unidades", "regimenes", "modalidades", "plazas", "puestos", "situaciones", "clasesServicio", "actos", "fuentes"]);
function nodo(d, etiqueta, texto) { const n = d.createElement(etiqueta); if (texto !== undefined) n.textContent = texto; return n; }
function opcionesValidas(catalogos) {
  return catalogos && typeof catalogos === "object" && OPCIONES.every((clave) => Array.isArray(catalogos[clave]) &&
    catalogos[clave].length <= 200 && catalogos[clave].every((v) => v && typeof v.ref === "string" && v.ref.length > 0 && v.ref.length <= 160 && typeof v.denominacion === "string" && v.denominacion.length > 0 && v.denominacion.length <= 240)) &&
    ["organismos", "unidades", "regimenes", "modalidades", "actos", "fuentes"].every((clave) => catalogos[clave].length > 0) &&
    ["regimenes", "modalidades", "situaciones", "clasesServicio"].every((clave) => catalogos[clave].every((v) => Number.isSafeInteger(v.version) && v.version >= 1 && v.estado === "publicada")) &&
    catalogos.fuentes.every((v) => Number.isSafeInteger(v.version) && v.version >= 1 && /^[a-f0-9]{64}$/u.test(v.huella_sha256)) &&
    [...catalogos.plazas, ...catalogos.puestos].every((v) => typeof v.version_ref === "string" && v.version_ref.length > 0);
}
export function accionesRegistroB2Disponibles({ cliente, catalogos, personaRef = "", empleadoRef = "" } = {}) {
  return typeof cliente?.registrarAlta === "function" && typeof cliente?.registrarHecho === "function" && opcionesValidas(catalogos) &&
    (PERSONA.test(personaRef) || EMPLEADO.test(empleadoRef));
}
function estado(d, texto, alerta = false) { const n = nodo(d, "p", texto); n.className = "personal-registro-b2-estado"; n.setAttribute("role", alerta ? "alert" : "status"); return n; }
function formatoInstante(valor) { return new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }).format(new Date(valor)); }
function fechaCivil(valor) { const fecha = new Date(`${valor}T12:00:00Z`); return /^\d{4}-\d{2}-\d{2}$/u.test(valor) && Number.isFinite(fecha.getTime()) && fecha.toISOString().slice(0, 10) === valor; }
function formatoFecha(valor) { return new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", timeZone: "Europe/Madrid" }).format(new Date(`${valor}T12:00:00Z`)); }

/** Formularios RRHH con opciones ya autorizadas por sus fuentes propietarias. */
export function montarActosRegistroB2({ raiz, cliente, catalogos, personaRef = "", empleadoRef = "", ficha = null, anunciar = () => {}, alRegistrar = () => {} } = {}) {
  if (!raiz?.append || !accionesRegistroB2Disponibles({ cliente, catalogos, personaRef, empleadoRef }) ||
      typeof anunciar !== "function" || typeof alRegistrar !== "function") throw new TypeError("actuaciones de Registro de Personal no disponibles");
  const d = raiz.ownerDocument; const t = crearTraductorPersonal(); const contenedor = nodo(d, "section"); contenedor.className = "panel personal-registro-b2-actos"; contenedor.dataset.personalRegistroB2Actos = ""; raiz.append(contenedor);
  let activa = true; let pendiente = null; let fase = "edicion";
  const campos = new Map(); const etiquetas = new Map(); let valoresBorrador = new Map();
  const tipos = [];
  if (PERSONA.test(personaRef)) tipos.push("alta");
  if (EMPLEADO.test(empleadoRef) && Number.isSafeInteger(ficha?.version) && ficha.version > 0) {
    tipos.push("relacion");
    if (Array.isArray(ficha.relaciones) && ficha.relaciones.length) tipos.push("revision_relacion");
    if (Array.isArray(ficha.relaciones) && ficha.relaciones.length && catalogos.plazas.length) tipos.push("ocupacion");
    if (Array.isArray(ficha.relaciones) && ficha.relaciones.length && catalogos.situaciones.length) tipos.push("situacion");
    if (Array.isArray(ficha.relaciones) && ficha.relaciones.length && catalogos.clasesServicio.length) tipos.push("servicio");
  }
  if (!tipos.length) { contenedor.remove?.(); throw new TypeError("actuación sin objetivo acreditado"); }
  let tipo = tipos[0];
  const obtener = (clave) => campos.get(clave)?.value || "";
  const opcion = (grupo, valor) => catalogos[grupo].find((item) => item.ref === valor);
  const versionado = (grupo, valor) => { const elegido = opcion(grupo, valor); return elegido?.estado === "publicada" && Number.isSafeInteger(elegido.version) && elegido.version >= 1 ? { ref: elegido.ref, version: elegido.version } : null; };
  const fila = (clave, etiqueta, control) => { const label = nodo(d, "label", t(etiqueta)); label.append(control); control.dataset.registroB2Campo = clave; campos.set(clave, control); etiquetas.set(clave, t(etiqueta)); return label; };
  const seleccionar = (clave, etiqueta, lista, requerido = true) => {
    const select = nodo(d, "select"); select.required = requerido;
    const vacia = nodo(d, "option", t("registro_b2_elegir")); vacia.value = ""; select.append(vacia);
    for (const item of lista) { const op = nodo(d, "option", item.denominacion); op.value = item.ref; select.append(op); }
    return fila(clave, etiqueta, select);
  };
  const entrada = (clave, etiqueta, tipoEntrada = "date", requerido = true) => {
    const input = nodo(d, "input"); input.type = tipoEntrada; input.required = requerido;
    if (tipoEntrada === "number") { input.min = "0"; input.step = "1"; }
    return fila(clave, etiqueta, input);
  };
  const enumerado = (clave, etiqueta, valores) => seleccionar(clave, etiqueta, valores.map(([ref, llave]) => ({ ref, denominacion: t(llave) })));
  const relacionesActuales = () => { const ultimas = new Map(); for (const r of ficha?.relaciones || []) if (!ultimas.has(r.relacion_ref) || ultimas.get(r.relacion_ref).traza.version < r.traza.version) ultimas.set(r.relacion_ref, r); return [...ultimas.values()]; };
  const relaciones = () => relacionesActuales().map((r) => ({ ref: r.relacion_ref, denominacion: `${r.unidad_denominacion || t("registro_b2_sin_denominacion")} · ${r.traza?.desde ? formatoFecha(r.traza.desde) : t("registro_b2_sin_valor")} – ${r.traza?.hasta ? formatoFecha(r.traza.hasta) : t("registro_b2_actual")}` }));
  const exito = ({ recibo, accesoActual }) => {
    fase = "exito"; pendiente = null; contenedor.replaceChildren();
    const cab = nodo(d, "header"); cab.className = "cabecera-panel"; cab.append(nodo(d, "h3", t(accesoActual.estado_replay === "replay" ? "registro_b2_replay" : "registro_b2_guardado")));
    const cuerpo = nodo(d, "div"); cuerpo.className = "cuerpo-panel";
    cuerpo.append(estado(d, t("registro_b2_registrado_en", { fecha: formatoInstante(recibo.registrado_en) })));
    contenedor.append(cab, cuerpo); alRegistrar(recibo);
  };
  const errorDeActo = (error) => {
    if (!activa) return;
    const errorCliente = error instanceof ErrorRegistroB2 || error?.name === "ErrorRegistroB2";
    const clave = errorCliente && [401, 403].includes(error.estado) ? "registro_b2_acto_denegado" :
      errorCliente && error.estado === 409 ? "registro_b2_acto_conflicto" :
      errorCliente && error.codigo === "resultado_incierto" ? "registro_b2_acto_incierto" : "registro_b2_acto_error";
    if (clave === "registro_b2_acto_incierto") fase = "incierto";
    else if (clave === "registro_b2_acto_conflicto") { fase = "conflicto"; pendiente = null; contenedor.replaceChildren(); }
    else { fase = "edicion"; pendiente = null; }
    pintarMensaje(clave, true);
    if (fase === "incierto") { reintentarNodo = nodo(d, "button", t("registro_b2_reintentar_exacto")); reintentarNodo.type = "button"; reintentarNodo.addEventListener("click", confirmar); contenedor.append(reintentarNodo); }
    anunciar(t(clave), "error");
  };
  let mensajeNodo; let reintentarNodo;
  function pintarMensaje(clave, alerta) { mensajeNodo?.remove?.(); reintentarNodo?.remove?.(); reintentarNodo = undefined; mensajeNodo = estado(d, t(clave), alerta); contenedor.append(mensajeNodo); }
  async function confirmar() {
    if (!activa || !pendiente || (fase !== "revision" && fase !== "incierto")) return;
    fase = "enviando"; pintarMensaje("registro_b2_guardando");
    const exacto = pendiente;
    try {
      const resultado = exacto.tipo === "alta"
        ? await cliente.registrarAlta(exacto.cuerpo, { claveIdempotencia: exacto.clave })
        : await cliente.registrarHecho(exacto.cuerpo, { claveIdempotencia: exacto.clave });
      if (activa && pendiente === exacto) exito(resultado);
    } catch (error) { if (activa && pendiente === exacto) errorDeActo(error); }
  }
  function preparar() {
    const necesarios = [...campos.entries()].filter(([, c]) => c.required).map(([clave]) => clave);
    if (necesarios.some((clave) => !obtener(clave))) { pintarMensaje("registro_b2_formulario_invalido", true); return; }
    if (!fechaCivil(obtener("vigente_desde")) || (obtener("vigente_hasta") && (!fechaCivil(obtener("vigente_hasta")) || obtener("vigente_hasta") <= obtener("vigente_desde"))) ||
        (tipo === "servicio" && (!fechaCivil(obtener("periodo_desde")) || !fechaCivil(obtener("periodo_hasta")) || obtener("periodo_hasta") < obtener("periodo_desde") || !/^\d+$/u.test(obtener("dias_reconocidos"))))) {
      pintarMensaje("registro_b2_formulario_invalido", true); return;
    }
    const fuente = opcion("fuentes", obtener("fuente"));
    if (!fuente || !globalThis.crypto?.randomUUID) { pintarMensaje("registro_b2_formulario_invalido", true); return; }
    const base = { vigente_desde: obtener("vigente_desde"), acto_ref: obtener("acto"), fuente_ref: fuente.ref, fuente_version: fuente.version, fuente_huella_sha256: fuente.huella_sha256 };
    if (obtener("vigente_hasta")) base.vigente_hasta = obtener("vigente_hasta");
    let cuerpo;
    if (tipo === "alta") {
      const regimen = versionado("regimenes", obtener("regimen")); const modalidad = versionado("modalidades", obtener("modalidad"));
      if (!regimen || !modalidad) { pintarMensaje("registro_b2_formulario_invalido", true); return; }
      cuerpo = { persona_ref: personaRef, organismo_ref: obtener("organismo"), unidad_ref: obtener("unidad"), regimen, modalidad, ...base };
    }
    else {
      cuerpo = { tipo: tipo === "revision_relacion" ? "relacion" : tipo, empleado_ref: empleadoRef, revision_esperada: 1, relacion_version_esperada: 0, ...base };
      if (tipo !== "relacion") {
        const relacion = relacionesActuales().find((r) => r.relacion_ref === obtener("relacion"));
        if (!relacion || !Number.isSafeInteger(relacion.traza?.version) || relacion.traza.version < 1) { pintarMensaje("registro_b2_formulario_invalido", true); return; }
        cuerpo.relacion_ref = relacion.relacion_ref; cuerpo.relacion_version_esperada = relacion.traza.version;
        if (tipo === "revision_relacion") cuerpo.revision_esperada = relacion.traza.version + 1;
      }
      if (tipo === "relacion" || tipo === "revision_relacion") {
        const regimen = versionado("regimenes", obtener("regimen")); const modalidad = versionado("modalidades", obtener("modalidad"));
        if (!regimen || !modalidad) { pintarMensaje("registro_b2_formulario_invalido", true); return; }
        Object.assign(cuerpo, { unidad_ref: obtener("unidad"), regimen, modalidad, estado: obtener("estado") });
      }
      if (tipo === "ocupacion") {
        const plaza = opcion("plazas", obtener("plaza")); const puesto = opcion("puestos", obtener("puesto"));
        const modalidad = versionado("modalidades", obtener("modalidad"));
        if (!plaza?.version_ref || !modalidad) { pintarMensaje("registro_b2_formulario_invalido", true); return; }
        Object.assign(cuerpo, { plaza_ref: plaza.ref, unidad_ref: obtener("unidad"), modalidad, clase_ocupacion: obtener("clase"), version_plaza_ref: plaza.version_ref });
        if (puesto) { cuerpo.puesto_ref = puesto.ref; if (puesto.version_ref) cuerpo.version_puesto_ref = puesto.version_ref; }
      }
      if (tipo === "situacion") { const situacion = versionado("situaciones", obtener("situacion")); if (!situacion) { pintarMensaje("registro_b2_formulario_invalido", true); return; } cuerpo.situacion = situacion; }
      if (tipo === "servicio") { const claseServicio = versionado("clasesServicio", obtener("clase_servicio")); if (!claseServicio) { pintarMensaje("registro_b2_formulario_invalido", true); return; } Object.assign(cuerpo, { clase_servicio: claseServicio, estado: obtener("estado"), periodo_desde: obtener("periodo_desde"), periodo_hasta: obtener("periodo_hasta"), dias_reconocidos: Number(obtener("dias_reconocidos")) }); }
    }
    pendiente = Object.freeze({ tipo, cuerpo: Object.freeze(cuerpo), clave: globalThis.crypto.randomUUID() });
    valoresBorrador = new Map([...campos].map(([clave, control]) => [clave, control.value]));
    fase = "revision"; pintarRevision();
  }
  function pintarRevision() {
    contenedor.replaceChildren(); mensajeNodo = undefined;
    const cab = nodo(d, "header"); cab.className = "cabecera-panel"; cab.append(nodo(d, "h3", t(TIPOS[tipo])));
    const cuerpo = nodo(d, "div"); cuerpo.className = "cuerpo-panel personal-registro-b2-revision";
    for (const [clave, control] of campos) {
      if (!control.value) continue;
      const texto = control.tagName === "select" ? [...control.children].find((o) => o.value === control.value)?.textContent || control.value : control.type === "date" ? formatoFecha(control.value) : control.type === "number" ? new Intl.NumberFormat("es-ES").format(Number(control.value)) : control.value;
      const linea = nodo(d, "div"); linea.append(nodo(d, "span", etiquetas.get(clave)), nodo(d, "strong", texto)); cuerpo.append(linea);
    }
    const acciones = nodo(d, "div"); acciones.className = "personal-registro-b2-acciones";
    const volver = nodo(d, "button", t("registro_b2_cancelar")); volver.type = "button"; volver.addEventListener("click", () => { pendiente = null; fase = "edicion"; pintarFormulario(valoresBorrador); });
    const confirmarBoton = nodo(d, "button", t("registro_b2_confirmar")); confirmarBoton.type = "button"; confirmarBoton.addEventListener("click", confirmar);
    acciones.append(volver, confirmarBoton); cuerpo.append(acciones); contenedor.append(cab, cuerpo);
  }
  function pintarFormulario(valores = new Map()) {
    contenedor.replaceChildren(); mensajeNodo = undefined; campos.clear(); etiquetas.clear();
    const cab = nodo(d, "header"); cab.className = "cabecera-panel"; cab.append(nodo(d, "h3", t("registro_b2_actuaciones")));
    const form = nodo(d, "form"); form.className = "cuerpo-panel personal-registro-b2-formulario";
    form.append(seleccionar("tipo", "registro_b2_tipo_actuacion", tipos.map((ref) => ({ ref, denominacion: t(TIPOS[ref]) }))));
    campos.get("tipo").value = tipo;
    campos.get("tipo").addEventListener("change", () => { tipo = obtener("tipo"); pintarFormulario(); });
    if (tipo === "alta") {
      form.append(estado(d, t("registro_b2_persona")));
      form.append(seleccionar("organismo", "registro_b2_organismo", catalogos.organismos));
    } else if (tipo !== "relacion") form.append(seleccionar("relacion", "registro_b2_relacion", relaciones()));
    if (["alta", "relacion", "revision_relacion", "ocupacion"].includes(tipo)) form.append(seleccionar("unidad", "registro_b2_unidad", catalogos.unidades));
    if (["alta", "relacion", "revision_relacion"].includes(tipo)) form.append(seleccionar("regimen", "registro_b2_regimen", catalogos.regimenes));
    if (["alta", "relacion", "revision_relacion", "ocupacion"].includes(tipo)) form.append(seleccionar("modalidad", "registro_b2_modalidad", catalogos.modalidades));
    if (tipo === "relacion" || tipo === "revision_relacion") form.append(enumerado("estado", "registro_b2_estado", [["vigente", "registro_b2_estado_vigente"], ["suspendida", "registro_b2_estado_suspendida"], ["finalizada", "registro_b2_estado_finalizada"]]));
    if (tipo === "ocupacion") { form.append(seleccionar("plaza", "registro_b2_plaza", catalogos.plazas)); form.append(seleccionar("puesto", "registro_b2_puesto", catalogos.puestos, false)); form.append(enumerado("clase", "registro_b2_clase", [["titular", "registro_b2_clase_titular"], ["provisional", "registro_b2_clase_provisional"], ["temporal", "registro_b2_clase_temporal"], ["reserva", "registro_b2_clase_reserva"]])); }
    if (tipo === "situacion") form.append(seleccionar("situacion", "registro_b2_situacion", catalogos.situaciones));
    if (tipo === "servicio") { form.append(seleccionar("clase_servicio", "registro_b2_clase_servicio", catalogos.clasesServicio)); form.append(enumerado("estado", "registro_b2_estado", [["declarado", "registro_b2_estado_declarado"], ["comprobado", "registro_b2_estado_comprobado"], ["reconocido", "registro_b2_estado_reconocido"]])); form.append(entrada("periodo_desde", "registro_b2_periodo_inicio")); form.append(entrada("periodo_hasta", "registro_b2_periodo_fin")); form.append(entrada("dias_reconocidos", "registro_b2_dias_reconocidos", "number")); }
    form.append(entrada("vigente_desde", "registro_b2_fecha_inicio"), entrada("vigente_hasta", "registro_b2_fecha_fin", "date", false));
    form.append(seleccionar("acto", "registro_b2_acto", catalogos.actos), seleccionar("fuente", "registro_b2_fuente_acto", catalogos.fuentes));
    for (const [clave, control] of campos) if (valores.has(clave)) control.value = valores.get(clave);
    const revisar = nodo(d, "button", t("registro_b2_revisar")); revisar.type = "submit"; form.append(revisar);
    form.addEventListener("submit", (evento) => { evento.preventDefault(); preparar(); });
    contenedor.append(cab, form);
  }
  pintarFormulario();
  return Object.freeze({ desmontar() { activa = false; pendiente = null; contenedor.remove?.(); } });
}
