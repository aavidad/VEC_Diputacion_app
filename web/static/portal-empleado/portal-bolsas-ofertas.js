import { LOCALIZACION_PORTAL, ZONA_HORARIA_PORTAL } from "./portal-i18n.js?v=20261007-pantallas-textos-final-v1";
import { crearSuperficieHistorialOfrecimientos, traducirHistorialOfrecimientos } from "./portal-bolsas-historial-ofrecimientos.js?v=20261010-ct-ficha-cohorte-v1";
import { instanteDesdeHoraMadrid } from "./hora-madrid.js";
import { referenciaContieneDocumentoIdentidad } from "./portal-bolsas-operaciones.js?v=20261008-r-traza-idioma-v1";
// Ofertas publicadas de una bolsa (Petición RRHH 3.06 y 3.07; Reglamento de
// bolsas, art. 8.1): RRHH publica la oferta con su número de plazas; al vencer
// el plazo para ofrecerse, VEC propone plaza a plaza a la siguiente persona
// por orden y RRHH confirma cada acto (adjudicar, respuesta, renuncia, falta
// de respuesta o llamamiento directo). Los textos viven en
// textos/<idioma>/bolsa-ofertas.json.
const { cargarTextos } = await import("../comun/textos.js");

const TEXTOS = await cargarTextos("bolsa-ofertas");
export const MENSAJES_OFERTAS_BOLSA = TEXTOS.seccion("ofertas");

export const RUTA_OFERTAS_BOLSA = "/api/vec/bolsa/ofertas";
export const RUTA_RESOLUCIONES_OFERTA = "/api/vec/bolsa/ofertas/resoluciones";
export const ESQUEMA_OFERTAS_BOLSA = "vec.bolsa.rrhh.ofertas.v1";
export const MAXIMO_PLAZAS_OFERTA = 100;

/** Traductor de la sección de ofertas; admite plurales con `cuenta`. */
export function crearTraductorOfertas(catalogo = MENSAJES_OFERTAS_BOLSA, localizacion = TEXTOS.localizacion) {
  const reglas = new Intl.PluralRules(localizacion);
  const numero = new Intl.NumberFormat(localizacion);
  return (clave, valores = {}) => {
    let plantilla = catalogo[clave];
    if (plantilla && typeof plantilla === "object") plantilla = plantilla[reglas.select(Number(valores.cuenta))] ?? plantilla.other;
    if (typeof plantilla !== "string") return clave;
    const variables = Object.hasOwn(valores, "cuenta") ? { ...valores, cuenta: numero.format(Number(valores.cuenta)) } : valores;
    return plantilla.replace(/\{(\w+)\}/g, (_, nombre) => String(variables[nombre] ?? ""));
  };
}

function escapar(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}

const ESTADOS = Object.freeze({ abierta: "info", pendiente_resolucion: "advertencia", en_curso: "advertencia", adjudicada: "exito", llamamiento_directo: "neutro", cerrada: "neutro" });
const ESTADOS_PLAZA = Object.freeze({ vacante: "advertencia", pendiente_respuesta: "info", cubierta: "exito", llamamiento_directo: "neutro" });
const ACTOS = new Set(["adjudicada", "aceptada", "renuncia", "sin_respuesta", "llamamiento_directo"]);
const REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9:._/-]{0,255}$/;
const REFERENCIA_CORREO = /^[A-Za-z0-9][A-Za-z0-9:_.-]{0,255}$/;
const HUELLA_CORREO = /^[a-f0-9]{64}$/;
const FECHA_NOTIFICACION = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,6})?Z$/;
const FUENTE_CORREO = "correo_externo_declarado_rrhh";
const FECHA_DIA = /^\d{4}-\d{2}-\d{2}$/;

const instante = (valor) => typeof valor === "string" && !Number.isNaN(Date.parse(valor));
const enteroOpcional = (valor) => valor === null || valor === undefined || Number.isSafeInteger(valor);
const referenciaOpcional = (valor) => valor === null || valor === undefined || REFERENCIA.test(valor);

function notificacionValida(notificacion) {
  return notificacion && FECHA_NOTIFICACION.test(notificacion.notificada_en ?? "") && instante(notificacion.notificada_en) &&
    REFERENCIA_CORREO.test(notificacion.referencia_correo ?? "") && !referenciaContieneDocumentoIdentidad(notificacion.referencia_correo) &&
    HUELLA_CORREO.test(notificacion.huella_correo_sha256 ?? "") &&
    notificacion.fuente === FUENTE_CORREO;
}

function fechaNotificacionLocal(valor) {
  const partes = /^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2})(?::([0-5]\d))?$/.exec(String(valor ?? ""));
  if (!partes) return "";
  const minutoMadrid = instanteDesdeHoraMadrid(partes[1]);
  if (minutoMadrid === null) return "";
  const fecha = new Date(minutoMadrid + Number(partes[2] ?? 0) * 1000);
  return fecha.toISOString().replace(/\.(\d{3})Z$/, (_total, milisegundos) => `.${milisegundos}000Z`);
}

function propuestaValida(p) {
  return p === null || p === undefined || p.tipo === "llamamiento_directo" ||
    (p.tipo === "adjudicar" && REFERENCIA.test(p.participacion_ref ?? "") && Number.isSafeInteger(p.orden_vigente));
}

function plazaValida(plaza, indice) {
  return plaza && plaza.numero_de_plaza === indice + 1 && plaza.estado in ESTADOS_PLAZA &&
    Number.isSafeInteger(plaza.secuencia) && plaza.secuencia >= 0 && referenciaOpcional(plaza.participacion_ref) &&
    enteroOpcional(plaza.orden_vigente) && (plaza.responder_antes_de === null || plaza.responder_antes_de === undefined || instante(plaza.responder_antes_de)) &&
    typeof plaza.puede_sin_respuesta === "boolean" && propuestaValida(plaza.propuesta) && Array.isArray(plaza.historial) &&
    plaza.historial.every((a) => a && ACTOS.has(a.tipo) && Number.isSafeInteger(a.secuencia) && enteroOpcional(a.orden_vigente));
}

export function validarOferta(oferta) {
  const d = oferta?.datos;
  if (!oferta || !REFERENCIA.test(oferta.oferta_ref ?? "") || !REFERENCIA.test(oferta.bolsa_ref ?? "") || !(oferta.estado in ESTADOS) ||
      !d || typeof d.categoria !== "string" || typeof d.centro !== "string" || !FECHA_DIA.test(d.fecha_inicio ?? "") ||
      (d.fecha_fin !== undefined && !FECHA_DIA.test(d.fecha_fin)) || !instante(oferta.publicada_en) || !instante(oferta.vence_antes_de) ||
      !Array.isArray(oferta.disposiciones) || !Number.isSafeInteger(oferta.disposiciones_total) || typeof oferta.plazo?.ejemplo !== "boolean" ||
      (oferta.confirmacion_adjudicacion !== undefined && oferta.confirmacion_adjudicacion !== null && oferta.confirmacion_adjudicacion !== "aceptacion_previa") ||
      (oferta.plazo.notificacion != null && !notificacionValida(oferta.plazo.notificacion)) ||
      !Number.isSafeInteger(oferta.numero_plazas) || oferta.numero_plazas < 1 || oferta.numero_plazas > MAXIMO_PLAZAS_OFERTA ||
      !Array.isArray(oferta.plazas) || oferta.plazas.length !== oferta.numero_plazas || !oferta.plazas.every(plazaValida) ||
      (oferta.politica_plazas !== null && oferta.politica_plazas !== undefined && typeof oferta.politica_plazas !== "object")) {
    throw new TypeError("Contrato de oferta de Bolsa no válido.");
  }
  return oferta;
}

export function validarOfertasBolsa(sobre) {
  const datos = sobre?.data;
  if (!datos || datos.esquema !== ESQUEMA_OFERTAS_BOLSA || !Array.isArray(datos.ofertas)) throw new TypeError("Contrato de ofertas de Bolsa no válido.");
  datos.ofertas.forEach(validarOferta);
  return datos;
}

async function codigoError(respuesta) {
  try { return (await respuesta.json())?.error?.codigo || ""; } catch { return ""; }
}

/** Cliente HTTP mínimo: nunca envía identidad; la sesión la acredita el servidor. */
export function crearClienteOfertas({ fetchImpl = fetch } = {}) {
  async function escribir(ruta, cuerpo, clave, signal) {
    const respuesta = await fetchImpl(ruta, { method: "POST", credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer", signal, body: JSON.stringify(cuerpo),
      headers: { Accept: "application/json", "Content-Type": "application/json", "Idempotency-Key": clave } });
    if (!respuesta.ok) return { ok: false, status: respuesta.status, codigo: await codigoError(respuesta) };
    return { ok: true, status: respuesta.status, oferta: validarOferta((await respuesta.json())?.data) };
  }
  return Object.freeze({
    async consultar(bolsaRef, { signal } = {}) {
      const respuesta = await fetchImpl(`${RUTA_OFERTAS_BOLSA}?${new URLSearchParams({ bolsa_ref: bolsaRef, limite: "50" })}`,
        { method: "GET", credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer", signal, headers: { Accept: "application/json" } });
      if (!respuesta.ok) return { ok: false, status: respuesta.status, codigo: await codigoError(respuesta) };
      return { ok: true, datos: validarOfertasBolsa(await respuesta.json()) };
    },
    publicar: (bolsaRef, datos, numeroPlazas, notificacion, clave, { signal } = {}) =>
      !notificacionValida(notificacion)
        ? Promise.resolve({ ok: false, status: 400, codigo: "solicitud_invalida" })
        : escribir(RUTA_OFERTAS_BOLSA, { bolsa_ref: bolsaRef, datos, numero_plazas: numeroPlazas, notificacion }, clave, signal),
    /** Acto sobre una plaza: tipo, persona (null en llamamiento directo) y la secuencia que vio RRHH. */
    registrarActo: (bolsaRef, acto, clave, { signal } = {}) =>
      escribir(RUTA_RESOLUCIONES_OFERTA, { bolsa_ref: bolsaRef, oferta_ref: acto.oferta_ref, numero_de_plaza: acto.numero_de_plaza,
        tipo: acto.tipo, secuencia_esperada: acto.secuencia_esperada, participacion_ref: acto.participacion_ref ?? null }, clave, signal),
  });
}

function generarClavePorDefecto() {
  return `oferta-${globalThis.crypto.randomUUID()}`;
}

export function mensajeError(traducir, resultado) {
  const especifica = `error_${resultado.status}_${resultado.codigo}`;
  if (traducir(especifica) !== especifica) return traducir(especifica);
  if (resultado.status === 403 || resultado.status === 422) return traducir(`error_${resultado.status}`);
  return traducir("error_generico");
}

function fechaHora(valor) {
  return new Intl.DateTimeFormat(LOCALIZACION_PORTAL, { dateStyle: "medium", timeStyle: "short", timeZone: ZONA_HORARIA_PORTAL }).format(new Date(valor));
}

function fechaDia(valor) {
  const [a, m, d] = valor.split("-").map(Number);
  return new Intl.DateTimeFormat(LOCALIZACION_PORTAL, { dateStyle: "medium", timeZone: "UTC" }).format(new Date(Date.UTC(a, m - 1, d)));
}

/** Acciones que admite una plaza en el instante de la consulta. */
export function accionesPlaza(plaza) {
  if (plaza.estado === "vacante" && plaza.propuesta?.tipo === "adjudicar") {
    return [{ tipo: "adjudicada", participacion: plaza.propuesta.participacion_ref, orden: plaza.propuesta.orden_vigente, principal: true }];
  }
  if (plaza.estado === "vacante" && plaza.propuesta?.tipo === "llamamiento_directo") return [{ tipo: "llamamiento_directo", participacion: null, principal: true }];
  if (plaza.estado === "pendiente_respuesta") {
    const comun = { participacion: plaza.participacion_ref, orden: plaza.orden_vigente };
    return [{ tipo: "aceptada", ...comun, principal: true }, { tipo: "renuncia", ...comun },
      { tipo: "sin_respuesta", ...comun, deshabilitada: !plaza.puede_sin_respuesta }];
  }
  return [];
}

const listaFormato = new Intl.ListFormat(LOCALIZACION_PORTAL, { type: "conjunction" });

export function crearSuperficieOfertasBolsa({ cliente = crearClienteOfertas(), clienteHistorial, alCambiar = () => {}, anunciar = () => {}, traducir = crearTraductorOfertas(), generarClave = generarClavePorDefecto } = {}) {
  const estado = { bolsa: "", carga: "inactiva", ofertas: [], error: "", enviando: false, claveEnCurso: null, borrador: null, mensaje: "", errorOperacion: "",
    registrando: false, confirmacion: null, clavesActo: new Map(), historialOferta: "" };
  let controlador = null;
  let controladorPublicacion = null;
  let documentoInstalado = null;
  const cambiar = () => {
    const activo = documentoInstalado?.activeElement;
    const enOfertas = activo?.closest?.(".ofertas-bolsa");
    const id = enOfertas ? activo.id : "";
    const accionOfertas = enOfertas ? activo.dataset?.ofertasAccion : "";
    const accionHistorial = enOfertas ? activo.dataset?.historialAccion : "";
    const ofertaRef = enOfertas ? activo.dataset?.ofertaRef : "";
    const participacion = enOfertas ? activo.dataset?.participacion : "";
    const contactoRef = enOfertas ? activo.dataset?.contactoRef : "";
    const seleccion = typeof activo?.selectionStart === "number" ? [activo.selectionStart, activo.selectionEnd] : null;
    alCambiar();
    if (!enOfertas) return;
    const nuevo = (id && documentoInstalado?.getElementById?.(id)) ||
      [...(documentoInstalado?.querySelectorAll?.("[data-ofertas-accion], [data-historial-accion]") || [])].find((e) =>
        (accionOfertas && e.dataset.ofertasAccion === accionOfertas && e.dataset.ofertaRef === ofertaRef) ||
        (accionHistorial && e.dataset.historialAccion === accionHistorial && e.dataset.participacion === participacion && e.dataset.contactoRef === contactoRef)) ||
      (accionHistorial === "identificar" && [...(documentoInstalado?.querySelectorAll?.("[data-historial-participacion]") || [])].find((e) => e.dataset.historialParticipacion === participacion && e.dataset.contactoRef === contactoRef));
    nuevo?.focus?.({ preventScroll: true });
    if (seleccion && nuevo?.setSelectionRange) nuevo.setSelectionRange(...seleccion);
  };
  const historial = crearSuperficieHistorialOfrecimientos({ cliente: clienteHistorial, alCambiar: cambiar, anunciar });
  const enfocar = (selector) => { documentoInstalado?.querySelector?.(selector)?.focus?.(); };

  async function cargar() {
    controlador?.abort(); controlador = new AbortController(); const { signal } = controlador;
    // Desde activar() no se repinta aquí: la vista se está montando.
    const repintar = estado.carga !== "inactiva";
    estado.carga = "cargando"; estado.error = ""; if (repintar) cambiar();
    try {
      const resultado = await cliente.consultar(estado.bolsa, { signal });
      if (signal.aborted) return;
      if (resultado.ok) { estado.ofertas = resultado.datos.ofertas; estado.carga = "lista"; }
      else { estado.carga = "error"; estado.error = resultado.status === 403 ? traducir("error_403") : traducir("error_carga"); }
    } catch (error) {
      if (signal.aborted || error?.name === "AbortError") return;
      estado.carga = "error"; estado.error = traducir("error_carga");
    }
    cambiar();
  }

  async function publicar(datos, numeroPlazas, notificacion, fechaLocal) {
    if (estado.enviando || !estado.bolsa) return;
    // Reintentar con el mismo contenido reutiliza la clave y obtiene la misma oferta.
    const borrador = { datos, numeroPlazas, notificacion, fechaLocal };
    if (!estado.claveEnCurso || JSON.stringify(estado.borrador) !== JSON.stringify(borrador)) estado.claveEnCurso = generarClave();
    estado.borrador = borrador; estado.enviando = true; estado.mensaje = ""; estado.errorOperacion = ""; cambiar();
    controladorPublicacion = new AbortController();
    const { signal } = controladorPublicacion;
    try {
      const resultado = await cliente.publicar(estado.bolsa, datos, numeroPlazas, notificacion, estado.claveEnCurso, { signal });
      if (signal.aborted) return;
      if (resultado.ok) {
        estado.mensaje = resultado.status === 200 ? traducir("publicada_antes") : traducir("publicada", { vence: fechaHora(resultado.oferta.vence_antes_de) });
        estado.claveEnCurso = null; estado.borrador = null; anunciar(estado.mensaje);
        void cargar();
      } else estado.errorOperacion = mensajeError(traducir, resultado);
    } catch { if (signal.aborted) return; estado.errorOperacion = traducir("error_generico"); }
    controladorPublicacion = null;
    estado.enviando = false; cambiar();
  }

  function pedirConfirmacion(acto) {
    if (estado.registrando) return;
    estado.confirmacion = acto; estado.mensaje = ""; estado.errorOperacion = ""; cambiar();
    enfocar('[data-ofertas-accion="confirmar"]');
  }

  function cancelarConfirmacion() {
    const acto = estado.confirmacion;
    estado.confirmacion = null; cambiar();
    if (acto) enfocar(`[data-ofertas-plaza="${acto.ofertaRef}|${acto.plaza}"] [data-ofertas-accion="preparar"][data-tipo="${acto.tipo}"]`);
  }

  async function confirmar() {
    const acto = estado.confirmacion;
    if (!acto || estado.registrando || !estado.bolsa) return;
    // Repetir el mismo acto sobre la misma versión de la plaza reutiliza la clave.
    const firma = `${acto.ofertaRef}|${acto.plaza}|${acto.tipo}|${acto.participacion ?? ""}|${acto.secuencia}`;
    const clave = estado.clavesActo.get(firma) || generarClave();
    estado.clavesActo.set(firma, clave);
    estado.registrando = true; estado.mensaje = ""; estado.errorOperacion = ""; cambiar();
    try {
      const resultado = await cliente.registrarActo(estado.bolsa, { oferta_ref: acto.ofertaRef, numero_de_plaza: acto.plaza, tipo: acto.tipo,
        participacion_ref: acto.participacion, secuencia_esperada: acto.secuencia }, clave);
      if (resultado.ok) {
        estado.clavesActo.delete(firma); estado.confirmacion = null;
        const telematica = resultado.oferta.confirmacion_adjudicacion === "aceptacion_previa" && acto.tipo === "adjudicada";
        estado.mensaje = traducir(telematica ? "hecho_adjudicada_telematica" : `hecho_${acto.tipo}`, { plaza: acto.plaza, orden: acto.orden ?? "" });
        anunciar(estado.mensaje); void cargar();
        enfocar('[data-ofertas-aviso="exito"]');
      } else {
        estado.errorOperacion = mensajeError(traducir, resultado);
        // Con la plaza cambiada, la clave queda ligada al acto anterior.
        if (resultado.status === 409) { estado.clavesActo.delete(firma); estado.confirmacion = null; void cargar(); }
      }
    } catch { estado.errorOperacion = traducir("error_generico"); }
    estado.registrando = false; cambiar();
  }

  function detallePlaza(o, plaza) {
    const partes = [];
    if (Number.isSafeInteger(plaza.orden_vigente) && (plaza.estado === "pendiente_respuesta" || plaza.estado === "cubierta")) partes.push(traducir("orden", { orden: plaza.orden_vigente }));
    if (plaza.estado === "pendiente_respuesta" && plaza.responder_antes_de) partes.push(traducir("responder_antes", { fecha: fechaHora(plaza.responder_antes_de) }));
    if (plaza.estado === "pendiente_respuesta" && !plaza.puede_sin_respuesta) partes.push(traducir("sin_respuesta_cuando"));
    if (plaza.estado === "llamamiento_directo") partes.push(traducir("directo_siguiente"));
    if (plaza.estado === "vacante" && plaza.propuesta?.tipo === "adjudicar") partes.push(traducir("siguiente", { orden: plaza.propuesta.orden_vigente }));
    if (plaza.estado === "vacante" && plaza.propuesta?.tipo === "llamamiento_directo") {
      const ultimo = plaza.historial.at(-1)?.tipo;
      const porPolitica = (ultimo === "renuncia" || ultimo === "sin_respuesta") && o.politica_plazas?.tras_renuncia === "llamamiento_directo";
      partes.push(traducir(porPolitica ? "directo_por_politica" : "sin_personas"));
    }
    if (plaza.estado === "vacante" && !plaza.propuesta && o.estado !== "abierta") partes.push(traducir("espera_anterior"));
    const antes = plaza.historial.filter((a) => a.tipo === "renuncia" || a.tipo === "sin_respuesta")
      .map((a) => traducir(`antes_${a.tipo}`, { orden: a.orden_vigente }));
    if (antes.length > 0) partes.push(traducir("antes", { lista: antes.join(" · ") }));
    return partes;
  }

  function botonesPlaza(o, plaza) {
    const confirmando = estado.confirmacion;
    if (confirmando && confirmando.ofertaRef === o.oferta_ref && confirmando.plaza === plaza.numero_de_plaza) {
      const telematica = o.confirmacion_adjudicacion === "aceptacion_previa" && confirmando.tipo === "adjudicada";
      const pregunta = traducir(telematica ? "confirmar_adjudicada_telematica" : `confirmar_${confirmando.tipo}`, { plaza: plaza.numero_de_plaza, orden: confirmando.orden ?? "" });
      const ocupado = estado.registrando ? " disabled" : "";
      return `<div class="acciones-fila" role="group" aria-label="${escapar(pregunta)}"><span>${escapar(pregunta)}</span>` +
        `<button type="button" class="boton-primario" data-ofertas-accion="confirmar"${ocupado}>${escapar(traducir(estado.registrando ? "registrando" : "confirmar"))}</button>` +
        `<button type="button" class="boton-secundario" data-ofertas-accion="cancelar"${ocupado}>${escapar(traducir("cancelar"))}</button></div>`;
    }
    const acciones = accionesPlaza(plaza);
    if (acciones.length === 0) return "";
    const bloqueado = estado.registrando || estado.confirmacion !== null;
    const idPlazo = `plazo-${escapar(o.oferta_ref)}-${plaza.numero_de_plaza}`;
    return `<div class="acciones-fila">${acciones.map((a) => {
      const deshabilitado = bloqueado || a.deshabilitada;
      const describe = a.deshabilitada ? ` aria-describedby="${idPlazo}"` : "";
      return `<button type="button" class="${a.principal ? "boton-primario" : "boton-secundario"}" data-ofertas-accion="preparar" data-oferta-ref="${escapar(o.oferta_ref)}"` +
        ` data-plaza="${plaza.numero_de_plaza}" data-tipo="${a.tipo}" data-secuencia="${plaza.secuencia}"` +
        `${a.participacion ? ` data-participacion-ref="${escapar(a.participacion)}"` : ""}${Number.isSafeInteger(a.orden) ? ` data-orden="${a.orden}"` : ""}${describe}${deshabilitado ? " disabled" : ""}>` +
        `${escapar(traducir(`accion_${a.tipo}`, { orden: a.orden ?? "" }))}</button>`;
    }).join("")}</div>`;
  }

  function listaPlazas(o) {
    const elementos = o.plazas.map((plaza) => {
      const detalle = detallePlaza(o, plaza);
      const idPlazo = `plazo-${escapar(o.oferta_ref)}-${plaza.numero_de_plaza}`;
      return `<li class="plaza-oferta" data-ofertas-plaza="${escapar(`${o.oferta_ref}|${plaza.numero_de_plaza}`)}">` +
        `<div class="plaza-oferta__cabecera"><strong>${escapar(traducir("plaza", { numero: plaza.numero_de_plaza }))}</strong>` +
        `<span class="estado-chip ${ESTADOS_PLAZA[plaza.estado]}">${escapar(traducir(`plaza_${plaza.estado}`))}</span></div>` +
        (detalle.length ? `<p class="dato-secundario" id="${idPlazo}">${detalle.map(escapar).join(" · ")}</p>` : "") +
        botonesPlaza(o, plaza) + "</li>";
    }).join("");
    return `<ol class="plazas-oferta" aria-label="${escapar(traducir("plazas_lista", { categoria: o.datos.categoria }))}">${elementos}</ol>`;
  }

  function fichaOferta(o) {
    const d = o.datos;
    const fechas = `${fechaDia(d.fecha_inicio)} – ${d.fecha_fin ? fechaDia(d.fecha_fin) : traducir("sin_fin")}`;
    const cubiertas = o.plazas.filter((p) => p.estado === "cubierta").length;
    const plazas = o.estado === "abierta" ? traducir("plazas_total", { cuenta: o.numero_plazas }) : traducir("plazas_cubiertas", { cuenta: o.numero_plazas, cubiertas });
    const ordenes = o.disposiciones.filter((x) => Number.isSafeInteger(x.orden_vigente)).map((x) => x.orden_vigente);
    const sinTurno = o.disposiciones.length - ordenes.length;
    const detalleOfrecidas = [ordenes.length ? traducir("ordenes", { lista: listaFormato.format(ordenes.map(String)) }) : "",
      sinTurno ? traducir("sin_turno_cuenta", { cuenta: sinTurno }) : ""].filter(Boolean).join(" · ");
    const dato = (etiqueta, valor, detalle = "") => `<div><dt>${escapar(traducir(etiqueta))}</dt><dd>${escapar(valor)}${detalle ? `<span class="dato-secundario">${escapar(detalle)}</span>` : ""}</dd></div>`;
    const idTitulo = `oferta-${escapar(o.oferta_ref.slice(-12))}`;
    const notificacion = o.plazo.notificacion;
    const correo = notificacion ? `<p class="dato-secundario">${escapar(traducir("correo_declarado", { fecha: fechaHora(notificacion.notificada_en) }))}</p>` +
      `<details><summary>${escapar(traducir("correo_evidencia"))}</summary><dl class="oferta-ficha__datos">${dato("correo_referencia", notificacion.referencia_correo)}` +
      `${dato("correo_huella", notificacion.huella_correo_sha256)}</dl></details>` : "";
    return `<li class="oferta-ficha" data-oferta-ref="${escapar(o.oferta_ref)}" aria-labelledby="${idTitulo}">` +
      `<div class="oferta-ficha__cabecera"><div><h4 id="${idTitulo}">${escapar(d.categoria)}</h4><p class="dato-secundario">${escapar(d.centro)} · ${escapar(d.descripcion)}</p></div>` +
      `<span class="estado-chip ${ESTADOS[o.estado]}">${escapar(traducir(`estado_${o.estado}`))}</span></div>` +
      `<dl class="oferta-ficha__datos">${dato("col_fechas", fechas)}${dato("col_plazo", fechaHora(o.vence_antes_de))}` +
      `${dato("col_disposiciones", String(o.disposiciones_total), detalleOfrecidas)}${dato("col_plazas", plazas)}</dl>${correo}` +
      // Las plazas se muestran cuando ya hay algo que decidir o ver: tras el plazo.
      (o.estado === "abierta" ? "" : listaPlazas(o)) +
      `<div class="acciones-fila"><button type="button" class="boton-secundario" data-ofertas-accion="historial" data-oferta-ref="${escapar(o.oferta_ref)}" aria-expanded="${estado.historialOferta === o.oferta_ref}">${escapar(traducirHistorialOfrecimientos("titulo"))}</button></div>` +
      (estado.historialOferta === o.oferta_ref ? historial.renderizar() : "") + "</li>";
  }

  function formulario() {
    const b = estado.borrador?.datos || {};
    const n = estado.borrador?.notificacion || {};
    const plazas = estado.borrador?.numeroPlazas ?? 1;
    const inactivo = estado.enviando ? " disabled" : "";
    const campo = (id, tipo, etiqueta, valor, extra = "") => `<label class="campo${tipo === "textarea" ? " campo-ancho" : ""}" for="oferta-${id}"><span>${escapar(traducir(etiqueta))}</span>` +
      (tipo === "textarea" ? `<textarea id="oferta-${id}" name="${id}" rows="2" maxlength="2000" required${inactivo}>${escapar(valor ?? "")}</textarea>`
        : `<input id="oferta-${id}" name="${id}" type="${tipo}" value="${escapar(valor ?? "")}"${extra}${inactivo}>`) + "</label>";
    return `<form class="formulario-gobernado" data-ofertas-form="publicar"><fieldset><legend>${escapar(traducir("publicar_titulo"))}</legend><div class="rejilla-formulario">` +
      campo("categoria", "text", "categoria", b.categoria, ' required minlength="2" maxlength="2000"') +
      campo("centro", "text", "centro", b.centro, ' required minlength="2" maxlength="2000"') +
      campo("fecha_inicio", "date", "fecha_inicio", b.fecha_inicio, " required") +
      campo("fecha_fin", "date", "fecha_fin", b.fecha_fin) +
      campo("numero_plazas", "number", "numero_plazas", plazas, ` required min="1" max="${MAXIMO_PLAZAS_OFERTA}" step="1" inputmode="numeric"`) +
      campo("descripcion", "textarea", "descripcion", b.descripcion) +
      `</div></fieldset><fieldset><legend>${escapar(traducir("correo_titulo"))}</legend><p class="dato-secundario">${escapar(traducir("correo_limite"))}</p><div class="rejilla-formulario">` +
      campo("notificada_en", "datetime-local", "correo_fecha", estado.borrador?.fechaLocal, ' required step="1"') +
      campo("referencia_correo", "text", "correo_referencia_obligatoria", n.referencia_correo, ' required maxlength="256" pattern="[A-Za-z0-9][A-Za-z0-9:_.-]{0,255}"') +
      campo("huella_correo_sha256", "text", "correo_huella_obligatoria", n.huella_correo_sha256, ' required minlength="64" maxlength="64" pattern="[a-f0-9]{64}" autocomplete="off" spellcheck="false"') +
      `</div><div class="acciones-formulario"><button type="submit" class="boton-primario"${inactivo}>${escapar(traducir(estado.enviando ? "publicando" : estado.claveEnCurso ? "reintentar" : "publicar"))}</button></div></fieldset></form>`;
  }

  function renderizar() {
    const total = estado.carga === "lista" ? `<span class="estado-chip info">${escapar(traducir("total", { cuenta: estado.ofertas.length }))}</span>` : "";
    let cuerpo;
    if (estado.carga === "cargando" || estado.carga === "inactiva") cuerpo = `<p role="status">${escapar(traducir("cargando"))}</p>`;
    else if (estado.carga === "error") cuerpo = `<p class="mensaje-error" role="alert">${escapar(estado.error)} <button type="button" class="boton-secundario" data-ofertas-accion="recargar">${escapar(traducir("reintentar_carga"))}</button></p>`;
    else if (estado.ofertas.length === 0) cuerpo = `<p>${escapar(traducir("vacio"))}</p>`;
    else cuerpo = `<ul class="ofertas-lista">${estado.ofertas.map(fichaOferta).join("")}</ul>`;
    const avisos = `<div aria-live="polite">${estado.mensaje ? `<p class="mensaje-exito ofertas-bolsa__aviso" data-ofertas-aviso="exito" tabindex="-1" role="status">${escapar(estado.mensaje)}</p>` : ""}</div>` +
      `${estado.errorOperacion ? `<p class="mensaje-error" role="alert">${escapar(estado.errorOperacion)}</p>` : ""}`;
    return `<section class="panel panel-separado ofertas-bolsa" aria-labelledby="titulo-ofertas-bolsa"><div class="cabecera-panel"><h3 id="titulo-ofertas-bolsa">${escapar(traducir("titulo"))}</h3>${total}</div><div class="cuerpo-panel">${avisos}${cuerpo}${formulario()}</div></section>`;
  }

  function manejarSubmit(evento) {
    if (historial.manejarSubmit(evento)) return true;
    const form = evento.target?.closest?.('[data-ofertas-form="publicar"]');
    if (!form) return false;
    evento.preventDefault();
    if (typeof form.reportValidity === "function" && !form.reportValidity()) return true;
    const valores = new FormData(form);
    const datos = { categoria: String(valores.get("categoria") ?? "").trim(), centro: String(valores.get("centro") ?? "").trim(),
      fecha_inicio: String(valores.get("fecha_inicio") ?? ""), descripcion: String(valores.get("descripcion") ?? "").trim() };
    const fin = String(valores.get("fecha_fin") ?? "");
    if (fin) datos.fecha_fin = fin;
    const plazas = Number(valores.get("numero_plazas") ?? 1);
    if (!Number.isSafeInteger(plazas) || plazas < 1 || plazas > MAXIMO_PLAZAS_OFERTA) { estado.errorOperacion = traducir("error_422"); cambiar(); return true; }
    const fechaLocal = String(valores.get("notificada_en") ?? "");
    const notificacion = { notificada_en: fechaNotificacionLocal(fechaLocal),
      referencia_correo: String(valores.get("referencia_correo") ?? "").trim(),
      huella_correo_sha256: String(valores.get("huella_correo_sha256") ?? "").trim(), fuente: FUENTE_CORREO };
    if (referenciaContieneDocumentoIdentidad(notificacion.referencia_correo)) {
      estado.errorOperacion = traducir("error_correo_referencia_personal"); cambiar(); return true;
    }
    if (!notificacionValida(notificacion)) { estado.errorOperacion = traducir("error_correo"); cambiar(); return true; }
    void publicar(datos, plazas, notificacion, fechaLocal);
    return true;
  }

  function manejarClick(evento) {
    if (historial.manejarClick(evento)) return true;
    const control = evento.target?.closest?.("[data-ofertas-accion]");
    if (!control || control.disabled) return false;
    const accion = control.dataset.ofertasAccion;
    if (accion === "recargar") void cargar();
    else if (accion === "historial") {
      const ofertaRef = control.dataset.ofertaRef;
      if (estado.historialOferta === ofertaRef) {
        historial.desmontar(); estado.historialOferta = "";
      } else if (estado.ofertas.some((o) => o.oferta_ref === ofertaRef)) {
        estado.historialOferta = ofertaRef;
        historial.activar(estado.bolsa, ofertaRef);
      }
      cambiar();
    }
    else if (accion === "preparar") {
      const orden = control.dataset.orden === undefined ? null : Number(control.dataset.orden);
      pedirConfirmacion({ ofertaRef: control.dataset.ofertaRef, plaza: Number(control.dataset.plaza), tipo: control.dataset.tipo,
        participacion: control.dataset.participacionRef || null, orden, secuencia: Number(control.dataset.secuencia) });
    } else if (accion === "confirmar") void confirmar();
    else if (accion === "cancelar") cancelarConfirmacion();
    else return false;
    return true;
  }

  return Object.freeze({
    activar(bolsaRef) {
      if (!bolsaRef || bolsaRef === estado.bolsa) return;
      controladorPublicacion?.abort(); controladorPublicacion = null;
      historial.desmontar();
      Object.assign(estado, { bolsa: bolsaRef, carga: "inactiva", ofertas: [], mensaje: "", errorOperacion: "", enviando: false, claveEnCurso: null, borrador: null, confirmacion: null, historialOferta: "" });
      estado.clavesActo.clear();
      void cargar();
    },
    desmontar() { controlador?.abort(); controlador = null; controladorPublicacion?.abort(); controladorPublicacion = null; historial.desmontar();
      estado.bolsa = ""; estado.carga = "inactiva"; estado.enviando = false; estado.confirmacion = null; estado.historialOferta = ""; },
    renderizar,
    instalar(documento) {
      documentoInstalado = documento;
      historial.instalar(documento);
      documento.addEventListener("input", (evento) => { historial.manejarInput(evento); });
      documento.addEventListener("change", (evento) => { historial.manejarInput(evento); });
      documento.addEventListener("submit", (evento) => { manejarSubmit(evento); });
      documento.addEventListener("click", (evento) => { manejarClick(evento); });
    },
    manejarSubmit, manejarClick, estado: () => ({ ...estado }),
  });
}
