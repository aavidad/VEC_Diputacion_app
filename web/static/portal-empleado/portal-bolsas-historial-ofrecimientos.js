import { cargarTextos } from "../comun/textos.js";
import { consultarCandidatosBolsa } from "./portal-bolsas-api.js?v=20261009-ct-bolsa-cohorte-v9";
import { LOCALIZACION_PORTAL, ZONA_HORARIA_PORTAL } from "./portal-i18n.js?v=20261007-pantallas-textos-final-v1";

const textos = await cargarTextos("bolsa-historial-ofrecimientos");
export const RUTA_HISTORIAL_OFRECIMIENTOS = "/api/vec/bolsa/ofertas/contactos";
export const ESQUEMA_HISTORIAL_OFRECIMIENTOS = "vec.bolsa.rrhh.contactos-oferta.v1";
const RESULTADOS = Object.freeze(["enviado", "no_enviado", "no_entregado", "entrega_declarada"]);
const SITUACIONES = Object.freeze(["disponible", "no_disponible", "trabajando", "pendiente_incorporacion", "renuncia", "excluido", "disponible_desde", "en_revision"]);
const REFERENCIA = /^[A-Za-z][A-Za-z0-9:_-]{0,255}$/;
const OFERTA = /^oferta:[a-f0-9]{64}$/;
const HUELLA = /^[a-f0-9]{64}$/;
const INSTANTE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$/;
const escapar = (valor) => String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
export const traducirHistorialOfrecimientos = (clave, variables = {}) => textos.traducir(`historial.${clave}`, variables);
const tecnico = (valor) => escapar(valor).replace(/([A-Za-z0-9:_-]{20})/g, "$1<wbr>");
const fecha = (instante) => new Intl.DateTimeFormat(LOCALIZACION_PORTAL, { dateStyle: "medium", timeStyle: "short", timeZone: ZONA_HORARIA_PORTAL }).format(new Date(instante));
const personalEvidente = (valor) => /@|\b\d{8}[A-Za-z]\b|\b[XYZxyz]\d{7}[A-Za-z]\b|\b(?:\+\d{9,15}|[6789]\d{8})\b|[\x00\u2028\u2029]/u.test(valor);
const esInstante = (v) => typeof v === "string" && INSTANTE.test(v) && Number.isFinite(Date.parse(v));
const bytes = (v) => new TextEncoder().encode(v).length;

export function validarRegistroOfrecimiento(c) {
  if (!c || c.canal !== "correo" || !OFERTA.test(c.oferta_ref ?? "") || !RESULTADOS.includes(c.resultado) || !esInstante(c.instante) ||
    typeof c.anotacion !== "string" || !c.anotacion.trim() || c.anotacion !== c.anotacion.trim() || bytes(c.anotacion) > 1000 || personalEvidente(c.anotacion)) return "error_datos";
  if (c.resultado === "entrega_declarada") {
    if (!REFERENCIA.test(c.evidencia_ref ?? "") || personalEvidente(c.evidencia_ref) || !HUELLA.test(c.evidencia_huella_sha256 ?? "")) return "error_evidencia";
  } else if (c.evidencia_ref || c.evidencia_huella_sha256) return "error_evidencia";
  return "";
}

function validarContacto(c, oferta) {
  if (!c || !REFERENCIA.test(c.contacto_ref ?? "") || !REFERENCIA.test(c.participacion_ref ?? "") || c.oferta_ref !== oferta ||
    c.canal !== "correo" || !RESULTADOS.includes(c.resultado) || !esInstante(c.instante) || typeof c.anotacion !== "string" || validarRegistroOfrecimiento(c)) throw new TypeError("contacto_contrato");
  return c;
}

export function validarHistorialOfrecimientos(sobre, bolsa, oferta) {
  const d = sobre?.data;
  if (!d || d.esquema !== ESQUEMA_HISTORIAL_OFRECIMIENTOS || d.bolsa_ref !== bolsa || d.oferta_ref !== oferta || !Array.isArray(d.contactos) || d.contactos.length > 100 ||
    (d.cursor_siguiente !== null && d.cursor_siguiente !== undefined && (typeof d.cursor_siguiente !== "string" || d.cursor_siguiente.length > 256))) throw new TypeError("historial_contrato");
  const referencias = new Set();
  for (const c of d.contactos) { validarContacto(c, oferta); if (referencias.has(c.contacto_ref)) throw new TypeError("historial_duplicado"); referencias.add(c.contacto_ref); }
  return d;
}

export function crearClienteHistorialOfrecimientos({ fetchImpl = globalThis.fetch, consultarPersonas = consultarCandidatosBolsa } = {}) {
  const opciones = (signal) => ({ credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer", signal, headers: { Accept: "application/json" } });
  return Object.freeze({
    async consultar(bolsa, oferta, { cursor = "", signal } = {}) {
      if (!REFERENCIA.test(bolsa ?? "") || !OFERTA.test(oferta ?? "")) return { ok: false, status: 400 };
      const query = new URLSearchParams({ bolsa_ref: bolsa, oferta_ref: oferta, limite: "50" });
      if (cursor) query.set("cursor", cursor);
      const r = await fetchImpl(`${RUTA_HISTORIAL_OFRECIMIENTOS}?${query}`, { ...opciones(signal), method: "GET" });
      if (!r.ok) return { ok: false, status: r.status };
      return { ok: true, datos: validarHistorialOfrecimientos(await r.json(), bolsa, oferta) };
    },
    buscar(bolsa, filtros = {}, { signal } = {}) { return consultarPersonas(bolsa, { ...filtros, limite: 50 }, { fetchImpl, signal }); },
    async registrar(bolsa, participacion, comando, clave, { signal } = {}) {
      if (!REFERENCIA.test(bolsa ?? "") || !REFERENCIA.test(participacion ?? "") || validarRegistroOfrecimiento(comando) || typeof clave !== "string" || !clave || clave.length > 256) return { ok: false, status: 400 };
      const r = await fetchImpl(`/api/vec/bolsa/bolsas/${bolsa}/candidatos/${participacion}/contactos`, {
        ...opciones(signal), method: "POST", headers: { Accept: "application/json", "Content-Type": "application/json", "Idempotency-Key": clave }, body: JSON.stringify(comando),
      });
      if (!r.ok) return { ok: false, status: r.status };
      const d = (await r.json())?.data;
      validarContacto(d, comando.oferta_ref);
      if (d.participacion_ref !== participacion || !REFERENCIA.test(d.recibo_ref ?? "") || typeof d.reutilizado !== "boolean" ||
        Object.entries(comando).some(([k, v]) => k === "instante" ? Date.parse(d[k]) !== Date.parse(v) : (d[k] ?? "") !== v)) throw new TypeError("recibo_contrato");
      return { ok: true, status: r.status, contacto: d };
    },
  });
}

/** datetime-local se interpreta en Madrid, incluso con un navegador en otra zona. */
export function fechaLocalMadrid(instante) {
  const p = Object.fromEntries(new Intl.DateTimeFormat(LOCALIZACION_PORTAL, { timeZone: ZONA_HORARIA_PORTAL, year: "numeric", month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit", hourCycle: "h23" }).formatToParts(new Date(instante)).map(({ type, value }) => [type, value]));
  return `${p.year}-${p.month}-${p.day}T${p.hour}:${p.minute}`;
}
export function instanteDesdeFechaLocal(valor) {
  if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(valor)) return null;
  const naive = Date.parse(`${valor}:00Z`);
  if (!Number.isFinite(naive)) return null;
  const posibles = [1, 2].map((offset) => new Date(naive - offset * 3600000).toISOString()).filter((v) => fechaLocalMadrid(v) === valor);
  return posibles.length === 1 ? posibles[0] : null;
}

/** El montaje conserva los eventos en su autoridad actual; instalar solo da contexto de foco. */
export function crearSuperficieHistorialOfrecimientos({ cliente = crearClienteHistorialOfrecimientos(), alCambiar = () => {}, anunciar = () => {}, traducir = traducirHistorialOfrecimientos,
  generarClave = () => `ofrecimiento-${globalThis.crypto.randomUUID()}`, ahora = () => new Date().toISOString() } = {}) {
  let bolsa = "", oferta = "", generacion = 0, documento = null;
  let lectura = null, busqueda = null, escritura = null, identificacion = null;
  const s = { carga: "inactiva", contactos: [], cursor: "", error: "", personas: [], cursorPersonas: "", cargaPersonas: false, errorPersonas: "", texto: "", situacion: "", seleccionada: null,
    borrador: null, revision: null, pendiente: null, enviando: false, recibo: null, errorRegistro: "", ayuda: false, identificando: "", errorIdentificacion: "" };
  const intentos = new Map(), nombres = new Map(), cursores = new Set(), cursoresPersonas = new Set(), noIdentificadas = new Set();
  const cambiar = () => alCambiar();
  const enfocar = (selector) => documento?.querySelector?.(selector)?.focus?.();
  const errorOperacion = (r, fallback) => traducir(r?.status === 401 ? "error_401" : r?.status === 403 ? "error_403" : r?.status === 409 ? "error_409" : fallback);
  const nombre = (ref) => nombres.get(ref)?.nombre_visible || traducir(noIdentificadas.has(ref) ? "no_identificada" : "nombre_pendiente");

  async function cargar({ mas = false } = {}) {
    if (!bolsa || !oferta || (mas && (!s.cursor || cursores.has(s.cursor)))) return;
    lectura?.abort(); lectura = new AbortController(); const { signal } = lectura; const turno = generacion;
    const cursor = mas ? s.cursor : "";
    s.carga = "cargando"; s.error = ""; cambiar();
    try {
      const r = await cliente.consultar(bolsa, oferta, { cursor, signal });
      if (signal.aborted || turno !== generacion) return;
      if (!r.ok) { s.carga = "error"; s.error = errorOperacion(r, "error_carga"); }
      else {
        if (!mas) { s.contactos = []; cursores.clear(); }
        if (cursor) cursores.add(cursor);
        const existentes = new Set(s.contactos.map((c) => c.contacto_ref));
        if (r.datos.contactos.some((c) => existentes.has(c.contacto_ref)) || (r.datos.cursor_siguiente && cursores.has(r.datos.cursor_siguiente))) throw new TypeError("pagina_inconsistente");
        s.contactos.push(...r.datos.contactos); s.cursor = r.datos.cursor_siguiente || ""; s.carga = "lista";
      }
    } catch { if (signal.aborted || turno !== generacion) return; s.carga = "error"; s.error = traducir("error_carga"); }
    cambiar();
  }

  async function buscar({ texto = s.texto, estado = s.situacion, mas = false } = {}) {
    if (!bolsa || (mas && (!s.cursorPersonas || cursoresPersonas.has(s.cursorPersonas)))) return;
    busqueda?.abort(); busqueda = new AbortController(); const { signal } = busqueda; const turno = generacion;
    const cursor = mas ? s.cursorPersonas : "";
    if (!mas) { s.personas = []; s.cursorPersonas = ""; cursoresPersonas.clear(); }
    s.texto = texto; s.situacion = estado; s.cargaPersonas = true; s.errorPersonas = ""; cambiar();
    try {
      const r = await cliente.buscar(bolsa, { texto, estado, cursor }, { signal });
      if (signal.aborted || turno !== generacion) return;
      if (!r.ok) s.errorPersonas = errorOperacion(r, "error_personas");
      else {
        if (r.datos.bolsa?.bolsa_ref !== bolsa || !Array.isArray(r.datos.candidatos)) throw new TypeError("personas_contrato");
        if (cursor) cursoresPersonas.add(cursor);
        if ((r.datos.hay_mas && !r.datos.cursor_siguiente) || (r.datos.cursor_siguiente && cursoresPersonas.has(r.datos.cursor_siguiente))) throw new TypeError("pagina_repetida");
        const existentes = new Set(s.personas.map((p) => p.participacion_ref));
        for (const p of r.datos.candidatos) {
          if (!REFERENCIA.test(p.participacion_ref ?? "") || typeof p.nombre_visible !== "string" || existentes.has(p.participacion_ref)) throw new TypeError("persona_contrato");
          nombres.set(p.participacion_ref, p); s.personas.push(p); existentes.add(p.participacion_ref);
        }
        s.cursorPersonas = r.datos.hay_mas ? r.datos.cursor_siguiente || "" : "";
      }
    } catch { if (signal.aborted || turno !== generacion) return; s.errorPersonas = traducir("error_personas"); }
    s.cargaPersonas = false; cambiar();
  }

  async function identificar(ref) {
    if (!s.contactos.some((c) => c.participacion_ref === ref) || nombres.has(ref) || s.identificando === ref) return;
    identificacion?.abort(); identificacion = new AbortController(); const { signal } = identificacion; const turno = generacion;
    s.identificando = ref; s.errorIdentificacion = ""; noIdentificadas.delete(ref); cambiar();
    let cursor = ""; const vistos = new Set();
    try {
      for (;;) {
        const r = await cliente.buscar(bolsa, { cursor }, { signal });
        if (signal.aborted || turno !== generacion) return;
        if (!r.ok || r.datos?.bolsa?.bolsa_ref !== bolsa || !Array.isArray(r.datos.candidatos)) throw new TypeError("persona_no_disponible");
        for (const p of r.datos.candidatos) {
          if (!REFERENCIA.test(p.participacion_ref ?? "") || typeof p.nombre_visible !== "string") throw new TypeError("persona_contrato");
          nombres.set(p.participacion_ref, p);
        }
        if (nombres.has(ref) || !r.datos.hay_mas) break;
        const siguiente = r.datos.cursor_siguiente;
        if (!REFERENCIA.test(siguiente ?? "") || vistos.has(siguiente)) throw new TypeError("cursor_repetido");
        vistos.add(siguiente); cursor = siguiente;
      }
      if (!nombres.has(ref)) noIdentificadas.add(ref);
    } catch { if (signal.aborted || turno !== generacion) return; s.errorIdentificacion = traducir("error_nombre"); }
    s.identificando = ""; cambiar();
  }

  function seleccionar(ref) {
    if (s.enviando || s.revision || s.pendiente) return;
    const p = s.personas.find((p) => p.participacion_ref === ref);
    if (!p) return;
    s.seleccionada = p; s.borrador = { resultado: "enviado", instanteLocal: fechaLocalMadrid(ahora()), anotacion: "", evidencia_ref: "", evidencia_huella_sha256: "" }; s.errorRegistro = ""; s.recibo = null; cambiar();
    enfocar('[data-historial-form="registro"] select');
  }

  function preparar(datos) {
    if (!s.seleccionada || s.enviando || s.pendiente) return false;
    s.borrador = { ...datos }; const instante = instanteDesdeFechaLocal(datos.instanteLocal);
    const c = { canal: "correo", resultado: datos.resultado, oferta_ref: oferta, instante, anotacion: String(datos.anotacion ?? "").trim(),
      evidencia_ref: String(datos.evidencia_ref ?? "").trim(), evidencia_huella_sha256: String(datos.evidencia_huella_sha256 ?? "").trim() };
    const error = !instante ? "error_fecha" : validarRegistroOfrecimiento(c);
    if (error) { s.errorRegistro = traducir(error); cambiar(); enfocar('[data-historial-error]'); return false; }
    s.revision = { comando: c, participacion: s.seleccionada.participacion_ref, nombre: s.seleccionada.nombre_visible };
    s.errorRegistro = ""; cambiar(); enfocar('[data-historial-accion="confirmar"]'); return true;
  }

  async function confirmar() {
    if (s.enviando || (!s.revision && !s.pendiente) || !bolsa) return;
    if (!s.pendiente) {
      const firma = JSON.stringify([bolsa, s.revision.participacion, s.revision.comando]);
      const clave = intentos.get(firma) || generarClave(); intentos.set(firma, clave);
      s.pendiente = { ...s.revision, clave };
    }
    const intento = s.pendiente; const turno = generacion;
    escritura?.abort(); escritura = new AbortController(); const { signal } = escritura;
    s.enviando = true; s.errorRegistro = ""; cambiar();
    try {
      const r = await cliente.registrar(bolsa, intento.participacion, intento.comando, intento.clave, { signal });
      if (signal.aborted || turno !== generacion) return;
      if (r.ok) { s.recibo = r.contacto; s.pendiente = null; s.revision = null; s.seleccionada = null; s.borrador = null; anunciar(traducir(r.contacto.reutilizado ? "recuperado" : "registrado")); void cargar(); }
      else s.errorRegistro = errorOperacion(r, "error_registro");
    } catch { if (signal.aborted || turno !== generacion) return; s.errorRegistro = traducir("error_registro"); }
    s.enviando = false; cambiar(); enfocar(s.recibo && !s.pendiente ? '[data-historial-recibo]' : '[data-historial-error]');
  }

  const boton = (accion, clave, disabled = false, extra = "") => `<button type="button" class="boton-secundario" data-historial-accion="${accion}"${disabled ? " disabled" : ""}${extra}>${escapar(traducir(clave))}</button>`;
  function tabla() {
    if (s.carga === "error" && !s.contactos.length) return "";
    if (!s.contactos.length) return `<p>${escapar(traducir(s.carga === "cargando" ? "cargando" : "vacio"))}</p>`;
    return `<div class="tabla-contenedor"><table class="tabla-datos"><caption class="visualmente-oculto">${escapar(traducir("titulo"))}</caption><thead><tr>${["persona", "fecha", "resultado", "entrega"].map((k) => `<th scope="col">${escapar(traducir(k))}</th>`).join("")}</tr></thead><tbody>${s.contactos.map((c) => `<tr><td tabindex="-1" data-historial-participacion="${escapar(c.participacion_ref)}" data-contacto-ref="${escapar(c.contacto_ref)}">${escapar(nombre(c.participacion_ref))}${!nombres.has(c.participacion_ref) && !noIdentificadas.has(c.participacion_ref) ? ` ${boton("identificar", s.identificando === c.participacion_ref ? "buscando_nombre" : "buscar_nombre", false, ` aria-disabled="${s.identificando === c.participacion_ref}" data-participacion="${escapar(c.participacion_ref)}" data-contacto-ref="${escapar(c.contacto_ref)}"`)}` : ""}</td><td>${escapar(fecha(c.instante))}</td><td>${escapar(traducir(c.resultado))}</td><td>${escapar(traducir(c.resultado === "entrega_declarada" ? "entrega_declarada" : "sin_entrega"))}${c.resultado === "entrega_declarada" ? `<details><summary>${escapar(traducir("detalle"))}</summary><dl><dt>${escapar(traducir("evidencia_ref"))}</dt><dd>${tecnico(c.evidencia_ref)}</dd><dt>${escapar(traducir("huella"))}</dt><dd>${tecnico(c.evidencia_huella_sha256)}</dd></dl></details>` : ""}</td></tr>`).join("")}</tbody></table></div>`;
  }

  function personas() {
    const busy = s.cargaPersonas || s.enviando;
    return `<form data-historial-form="buscar" class="formulario-gobernado"><div class="rejilla-formulario"><label class="campo" for="historial-texto"><span>${escapar(traducir("buscar"))}</span><input id="historial-texto" name="texto" type="search" maxlength="100" value="${escapar(s.texto)}"></label><label class="campo" for="historial-estado"><span>${escapar(traducir("estado"))}</span><select id="historial-estado" name="estado"><option value="">${escapar(traducir("todos"))}</option>${SITUACIONES.map((k) => `<option value="${k}"${s.situacion === k ? " selected" : ""}>${escapar(traducir(k))}</option>`).join("")}</select></label></div><button class="boton-secundario" type="submit"${busy ? " disabled" : ""}>${escapar(traducir("buscar_boton"))}</button></form>` +
      (s.errorPersonas ? `<p role="alert" class="mensaje-error">${escapar(s.errorPersonas)}</p>` : "") +
      (s.cargaPersonas ? `<p role="status">${escapar(traducir("cargando"))}</p>` : "") +
      (s.personas.length ? `<ul class="lista-datos">${s.personas.map((p) => `<li><strong>${escapar(p.nombre_visible)}</strong>${Number.isSafeInteger(p.orden) ? ` <span>${escapar(traducir("orden", { orden: new Intl.NumberFormat(LOCALIZACION_PORTAL).format(p.orden) }))}</span>` : ""} ${boton("seleccionar", "seleccionar", busy, ` data-participacion="${escapar(p.participacion_ref)}" aria-label="${escapar(`${traducir("seleccionar")} ${p.nombre_visible}`)}"`)}</li>`).join("")}</ul>` : !s.cargaPersonas && !s.errorPersonas ? `<p>${escapar(traducir("sin_personas"))}</p>` : "") +
      (s.cursorPersonas ? boton("mas-personas", "mas_personas", busy) : "");
  }

  function formulario() {
    if (!s.seleccionada) return personas();
    const b = s.borrador;
    const campo = (k, tipo, valor, extra = "") => `<label class="campo" for="historial-${k}"><span>${escapar(traducir(k === "instanteLocal" ? "fecha" : k === "evidencia_huella_sha256" ? "huella" : k))}</span><input id="historial-${k}" name="${k}" type="${tipo}" value="${escapar(valor)}"${extra}></label>`;
    return `<p>${escapar(traducir("seleccionada", { nombre: s.seleccionada.nombre_visible }))} ${boton("cambiar-persona", "cambiar")}</p><form data-historial-form="registro" class="formulario-gobernado"><fieldset><legend>${escapar(traducir("registro"))}</legend><div class="rejilla-formulario"><label class="campo" for="historial-resultado"><span>${escapar(traducir("resultado"))}</span><select id="historial-resultado" name="resultado">${RESULTADOS.map((k) => `<option value="${k}"${b.resultado === k ? " selected" : ""}>${escapar(traducir(k))}</option>`).join("")}</select></label>${campo("instanteLocal", "datetime-local", b.instanteLocal, " required")}${campo("anotacion", "text", b.anotacion, ' required maxlength="1000" aria-describedby="historial-privacidad"')}${campo("evidencia_ref", "text", b.evidencia_ref, ' maxlength="256" pattern="[A-Za-z][A-Za-z0-9:_-]*" aria-describedby="historial-privacidad"')}${campo("evidencia_huella_sha256", "text", b.evidencia_huella_sha256, ' maxlength="64" pattern="[a-f0-9]{64}"')}</div><p id="historial-privacidad" class="dato-secundario">${escapar(traducir("privacidad"))}</p><div class="acciones-formulario"><button class="boton-primario" type="submit">${escapar(traducir("revisar"))}</button></div></fieldset></form>`;
  }

  function revision() {
    const r = s.pendiente || s.revision, c = r.comando;
    const par = (k, v) => `<div><dt>${escapar(traducir(k))}</dt><dd>${["evidencia_ref", "huella"].includes(k) ? tecnico(v) : escapar(v)}</dd></div>`;
    return `<section class="panel" aria-labelledby="historial-revision"><h4 id="historial-revision">${escapar(traducir("revision"))}</h4><dl class="lista-datos">${par("persona", r.nombre)}${par("fecha", fecha(c.instante))}${par("resultado", traducir(c.resultado))}${par("entrega", traducir(c.resultado === "entrega_declarada" ? "entrega_declarada" : "sin_entrega"))}${par("anotacion", c.anotacion)}${c.evidencia_ref ? par("evidencia_ref", c.evidencia_ref) + par("huella", c.evidencia_huella_sha256) : ""}</dl><div class="acciones-formulario">${boton("confirmar", s.enviando ? "registrando" : s.pendiente ? "reintentar" : "confirmar", s.enviando)}${boton("cancelar", "cancelar", s.enviando)}</div></section>`;
  }

  function renderizar() {
    if (!bolsa || !oferta) return "";
    return `<section class="panel panel-separado" aria-labelledby="historial-titulo"><div class="cabecera-panel"><h4 id="historial-titulo">${escapar(traducir("titulo"))}</h4>${boton("ayuda", "ayuda_boton", false, ' aria-controls="historial-ayuda" aria-expanded="' + s.ayuda + '"')}</div><div class="cuerpo-panel">${s.ayuda ? `<p id="historial-ayuda">${escapar(traducir("ayuda"))}</p>` : ""}${s.recibo ? `<p class="mensaje-exito" role="status" tabindex="-1" data-historial-recibo>${escapar(traducir(s.recibo.reutilizado ? "recuperado" : "registrado"))} </p><details><summary>${escapar(traducir("ver_recibo"))}</summary><p>${tecnico(traducir("recibo", { recibo: s.recibo.recibo_ref }))}</p></details>` : ""}${s.error ? `<p class="mensaje-error" role="alert">${escapar(s.error)} ${boton("recargar", "reintentar_carga")}</p>` : ""}${tabla()}${s.errorIdentificacion ? `<p class="mensaje-error" role="alert">${escapar(s.errorIdentificacion)}</p>` : ""}${s.cursor ? boton("mas-historial", "mas_historial", s.carga === "cargando") : ""}${s.errorRegistro ? `<p class="mensaje-error" role="alert" tabindex="-1" data-historial-error>${escapar(s.errorRegistro)}</p>` : ""}${s.revision || s.pendiente ? revision() : formulario()}</div></section>`;
  }

  function manejarSubmit(e) {
    const f = e.target?.closest?.("[data-historial-form]"); if (!f?.dataset?.historialForm) return false;
    e.preventDefault(); if (s.enviando) return true;
    if (typeof f.reportValidity === "function" && !f.reportValidity()) return true;
    const d = new FormData(f); const leer = (k) => String(d.get(k) ?? "").trim();
    if (f.dataset.historialForm === "buscar") void buscar({ texto: leer("texto"), estado: leer("estado") });
    else if (f.dataset.historialForm === "registro") preparar(Object.fromEntries(["resultado", "instanteLocal", "anotacion", "evidencia_ref", "evidencia_huella_sha256"].map((k) => [k, leer(k)])));
    else return false;
    return true;
  }
  function manejarInput(e) {
    const fBusqueda = e.target?.closest?.('[data-historial-form="buscar"]');
    if (fBusqueda && e.target.name === "texto") { s.texto = String(e.target.value ?? ""); return true; }
    if (fBusqueda && e.target.name === "estado") { s.situacion = String(e.target.value ?? ""); return true; }
    const f = e.target?.closest?.('[data-historial-form="registro"]');
    const k = e.target?.name;
    if (!f || !s.borrador || !["resultado", "instanteLocal", "anotacion", "evidencia_ref", "evidencia_huella_sha256"].includes(k)) return false;
    s.borrador[k] = String(e.target.value ?? "");
    return true;
  }
  function manejarClick(e) {
    const b = e.target?.closest?.("[data-historial-accion]"); if (!b?.dataset?.historialAccion || b.disabled) return false;
    switch (b.dataset.historialAccion) {
      case "recargar": void cargar(); break;
      case "mas-historial": void cargar({ mas: true }); break;
      case "mas-personas": void buscar({ mas: true }); break;
      case "seleccionar": seleccionar(b.dataset.participacion); break;
      case "identificar": void identificar(b.dataset.participacion); break;
      case "confirmar": void confirmar(); break;
      case "cancelar": if (!s.enviando) { s.revision = null; s.pendiente = null; cambiar(); enfocar('[data-historial-form="registro"] select'); } break;
      case "cambiar-persona": if (!s.enviando && !s.pendiente) { s.seleccionada = null; s.borrador = null; cambiar(); enfocar("#historial-texto"); } break;
      case "ayuda": s.ayuda = !s.ayuda; cambiar(); enfocar('[data-historial-accion="ayuda"]'); break;
      default: return false;
    }
    return true;
  }
  function desmontar() {
    generacion++; lectura?.abort(); busqueda?.abort(); escritura?.abort(); identificacion?.abort(); bolsa = ""; oferta = "";
    Object.assign(s, { carga: "inactiva", contactos: [], cursor: "", error: "", personas: [], cursorPersonas: "", cargaPersonas: false, errorPersonas: "", texto: "", situacion: "", seleccionada: null, borrador: null, revision: null, pendiente: null, enviando: false, recibo: null, errorRegistro: "", ayuda: false, identificando: "", errorIdentificacion: "" });
    intentos.clear(); nombres.clear(); cursores.clear(); cursoresPersonas.clear(); noIdentificadas.clear();
  }
  return Object.freeze({ activar(b, o) { if (b === bolsa && o === oferta) return; desmontar(); if (!REFERENCIA.test(b ?? "") || !OFERTA.test(o ?? "")) return; bolsa = b; oferta = o; void cargar(); void buscar(); }, desmontar,
    renderizar, manejarClick, manejarSubmit, manejarInput, instalar(d) { documento = d; }, cargar, buscar, identificar, seleccionar, preparar, confirmar,
    estado: () => structuredClone({ ...s, bolsa, oferta }),
  });
}
