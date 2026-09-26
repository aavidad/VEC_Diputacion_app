import { LOCALIZACION_PORTAL, ZONA_HORARIA_PORTAL } from "./portal-i18n.js?v=20260926-pulido-portal-v1";
// Ofertas publicadas de una bolsa (Petición RRHH p. 3; Reglamento de bolsas,
// art. 8.1): RRHH publica la oferta, consulta quién manifestó disposición y,
// al vencer el plazo de la regla del catálogo, confirma la propuesta de
// adjudicación que calcula VEC o el paso a llamamiento directo.
export const RUTA_OFERTAS_BOLSA = "/api/vec/bolsa/ofertas";
export const RUTA_RESOLUCIONES_OFERTA = "/api/vec/bolsa/ofertas/resoluciones";
export const ESQUEMA_OFERTAS_BOLSA = "vec.bolsa.rrhh.ofertas.v1";

export const MENSAJES_OFERTAS_BOLSA_ES = Object.freeze({
  titulo: "Ofertas publicadas",
  subtitulo: "Publicación, disposición y adjudicación",
  publicar_titulo: "Nueva oferta",
  categoria: "Categoría",
  centro: "Centro",
  fecha_inicio: "Fecha de inicio",
  fecha_fin: "Fecha de fin (opcional)",
  descripcion: "Descripción",
  publicar: "Publicar oferta",
  publicando: "Publicando…",
  reintentar: "Reintentar la misma publicación",
  publicada: "Oferta publicada. Plazo de disposición hasta {vence}.",
  publicada_antes: "La oferta ya estaba publicada con esta misma solicitud.",
  cargando: "Cargando ofertas…",
  vacio: "Esta bolsa no tiene ofertas publicadas.",
  col_oferta: "Oferta",
  col_fechas: "Fechas",
  col_plazo: "Disposición hasta",
  col_disposiciones: "Disposiciones",
  col_estado: "Estado",
  col_propuesta: "Propuesta de VEC",
  sin_fin: "sin fecha de fin",
  estado_abierta: "Abierta",
  estado_pendiente_resolucion: "Pendiente de resolver",
  estado_adjudicada: "Adjudicada",
  estado_llamamiento_directo: "Llamamiento directo",
  regla_ejemplo: "Regla de ejemplo",
  propuesta_adjudicar: "Adjudicar al orden {orden}",
  propuesta_directo: "Sin disposiciones",
  confirmar_adjudicacion: "Confirmar adjudicación",
  confirmar_directo: "Pasar a llamamiento directo",
  resolviendo: "Confirmando…",
  resuelta_adjudicada: "Adjudicada al orden {orden}",
  resuelta_directo: "Pasa a llamamiento directo",
  resolucion_registrada: "Resolución registrada.",
  en_plazo: "En plazo",
  orden: "Orden {orden}",
  sin_turno: "Sin turno",
  total: "{n} ofertas",
  error_carga: "No se pudieron consultar las ofertas.",
  reintentar_carga: "Reintentar",
  error_403: "La sesión no dispone de ámbito sobre esta bolsa.",
  error_409_clave_divergente: "Esa solicitud ya se usó con otros datos. Revise el formulario.",
  error_409_oferta_ya_resuelta: "Otra persona ya resolvió esta oferta.",
  error_409_plazo_abierto: "El plazo de disposición sigue abierto.",
  error_409_propuesta_cambiada: "El orden ha cambiado. Revise la propuesta actualizada.",
  error_422: "Revise los datos de la oferta.",
  error_503_plazo_no_configurado: "No hay regla de plazo configurada: no se puede publicar.",
  error_generico: "No se pudo completar la operación. Consulte las ofertas antes de repetirla.",
});

export function crearTraductorOfertas(catalogo = MENSAJES_OFERTAS_BOLSA_ES) {
  return (clave, valores = {}) => String(catalogo[clave] ?? clave).replace(/\{(\w+)\}/g, (_, nombre) => String(valores[nombre] ?? ""));
}

function escapar(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}

const ESTADOS = Object.freeze({ abierta: "info", pendiente_resolucion: "advertencia", adjudicada: "exito", llamamiento_directo: "neutro" });
const REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9:._/-]{0,255}$/;
const FECHA_DIA = /^\d{4}-\d{2}-\d{2}$/;

function instante(valor) {
  return typeof valor === "string" && !Number.isNaN(Date.parse(valor));
}

export function validarOferta(oferta) {
  const d = oferta?.datos;
  if (!oferta || !REFERENCIA.test(oferta.oferta_ref ?? "") || !REFERENCIA.test(oferta.bolsa_ref ?? "") || !(oferta.estado in ESTADOS) ||
      !d || typeof d.categoria !== "string" || typeof d.centro !== "string" || !FECHA_DIA.test(d.fecha_inicio ?? "") ||
      (d.fecha_fin !== undefined && !FECHA_DIA.test(d.fecha_fin)) || !instante(oferta.publicada_en) || !instante(oferta.vence_antes_de) ||
      !Array.isArray(oferta.disposiciones) || !Number.isSafeInteger(oferta.disposiciones_total) || typeof oferta.plazo?.ejemplo !== "boolean") {
    throw new TypeError("Contrato de oferta de Bolsa no válido.");
  }
  const p = oferta.propuesta;
  if (p !== null && p !== undefined && !(p.tipo === "llamamiento_directo" || (p.tipo === "adjudicar" && REFERENCIA.test(p.participacion_ref ?? "") && Number.isSafeInteger(p.orden_vigente)))) {
    throw new TypeError("Propuesta de oferta no válida.");
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
    publicar: (bolsaRef, datos, clave, { signal } = {}) => escribir(RUTA_OFERTAS_BOLSA, { bolsa_ref: bolsaRef, datos }, clave, signal),
    resolver: (bolsaRef, ofertaRef, participacionRef, clave, { signal } = {}) =>
      escribir(RUTA_RESOLUCIONES_OFERTA, { bolsa_ref: bolsaRef, oferta_ref: ofertaRef, participacion_ref: participacionRef ?? null }, clave, signal),
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

export function crearSuperficieOfertasBolsa({ cliente = crearClienteOfertas(), alCambiar = () => {}, anunciar = () => {}, traducir = crearTraductorOfertas(), generarClave = generarClavePorDefecto } = {}) {
  const estado = { bolsa: "", carga: "inactiva", ofertas: [], error: "", enviando: false, claveEnCurso: null, borrador: null, mensaje: "", errorOperacion: "", resolviendo: "", clavesResolucion: new Map() };
  let controlador = null;
  const cambiar = () => alCambiar();

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

  async function publicar(datos) {
    if (estado.enviando || !estado.bolsa) return;
    // Reintentar con el mismo contenido reutiliza la clave y obtiene la misma oferta.
    if (!estado.claveEnCurso || JSON.stringify(estado.borrador) !== JSON.stringify(datos)) estado.claveEnCurso = generarClave();
    estado.borrador = datos; estado.enviando = true; estado.mensaje = ""; estado.errorOperacion = ""; cambiar();
    try {
      const resultado = await cliente.publicar(estado.bolsa, datos, estado.claveEnCurso);
      if (resultado.ok) {
        estado.mensaje = resultado.status === 200 ? traducir("publicada_antes") : traducir("publicada", { vence: fechaHora(resultado.oferta.vence_antes_de) });
        estado.claveEnCurso = null; estado.borrador = null; anunciar(estado.mensaje);
        void cargar();
      } else estado.errorOperacion = mensajeError(traducir, resultado);
    } catch { estado.errorOperacion = traducir("error_generico"); }
    estado.enviando = false; cambiar();
  }

  async function resolver(ofertaRef, participacionRef) {
    if (estado.resolviendo || !estado.bolsa) return;
    const clave = estado.clavesResolucion.get(ofertaRef) || generarClave();
    estado.clavesResolucion.set(ofertaRef, clave);
    estado.resolviendo = ofertaRef; estado.mensaje = ""; estado.errorOperacion = ""; cambiar();
    try {
      const resultado = await cliente.resolver(estado.bolsa, ofertaRef, participacionRef, clave);
      if (resultado.ok) { estado.mensaje = traducir("resolucion_registrada"); estado.clavesResolucion.delete(ofertaRef); anunciar(estado.mensaje); void cargar(); }
      else {
        estado.errorOperacion = mensajeError(traducir, resultado);
        // Si la propuesta cambió, la clave usada queda ligada a la anterior.
        if (resultado.status === 409) { estado.clavesResolucion.delete(ofertaRef); void cargar(); }
      }
    } catch { estado.errorOperacion = traducir("error_generico"); }
    estado.resolviendo = ""; cambiar();
  }

  function filaOferta(o) {
    const d = o.datos;
    const fechas = `${escapar(fechaDia(d.fecha_inicio))} – ${d.fecha_fin ? escapar(fechaDia(d.fecha_fin)) : escapar(traducir("sin_fin"))}`;
    let propuesta = `<span class="dato-secundario">${escapar(traducir("en_plazo"))}</span>`;
    if (o.resolucion) {
      propuesta = o.resolucion.tipo === "adjudicada" ? escapar(traducir("resuelta_adjudicada", { orden: o.resolucion.orden_vigente })) : escapar(traducir("resuelta_directo"));
    } else if (o.propuesta) {
      const adjudicar = o.propuesta.tipo === "adjudicar";
      const ocupado = estado.resolviendo === o.oferta_ref;
      propuesta = `<div class="acciones-fila"><span>${escapar(adjudicar ? traducir("propuesta_adjudicar", { orden: o.propuesta.orden_vigente }) : traducir("propuesta_directo"))}</span>` +
        `<button type="button" class="${adjudicar ? "boton-primario" : "boton-secundario"}" data-ofertas-accion="resolver" data-oferta-ref="${escapar(o.oferta_ref)}"` +
        `${adjudicar ? ` data-participacion-ref="${escapar(o.propuesta.participacion_ref)}"` : ""}${estado.resolviendo ? " disabled" : ""}>` +
        `${escapar(ocupado ? traducir("resolviendo") : traducir(adjudicar ? "confirmar_adjudicacion" : "confirmar_directo"))}</button></div>`;
    }
    const disposiciones = o.disposiciones.length === 0 ? "" : `<div class="lista-chips">${o.disposiciones.map((x) =>
      `<span class="estado-chip${Number.isSafeInteger(x.orden_vigente) ? " info" : ""}">${escapar(Number.isSafeInteger(x.orden_vigente) ? traducir("orden", { orden: x.orden_vigente }) : traducir("sin_turno"))}</span>`).join("")}</div>`;
    return `<tr data-oferta-ref="${escapar(o.oferta_ref)}"><td><strong>${escapar(d.categoria)}</strong><br><span class="dato-secundario">${escapar(d.centro)} · ${escapar(d.descripcion)}</span></td>` +
      `<td>${fechas}</td><td>${escapar(fechaHora(o.vence_antes_de))}</td>` +
      `<td class="numero">${escapar(o.disposiciones_total)}${disposiciones}</td>` +
      `<td><span class="estado-chip ${ESTADOS[o.estado]}">${escapar(traducir(`estado_${o.estado}`))}</span></td><td>${propuesta}</td></tr>`;
  }

  function formulario() {
    const b = estado.borrador || {};
    const inactivo = estado.enviando ? " disabled" : "";
    const campo = (id, tipo, etiqueta, valor, extra = "") => `<label class="campo${tipo === "textarea" ? " campo-ancho" : ""}" for="oferta-${id}"><span>${escapar(traducir(etiqueta))}</span>` +
      (tipo === "textarea" ? `<textarea id="oferta-${id}" name="${id}" rows="2" maxlength="2000" required${inactivo}>${escapar(valor ?? "")}</textarea>`
        : `<input id="oferta-${id}" name="${id}" type="${tipo}" value="${escapar(valor ?? "")}"${extra}${inactivo}>`) + "</label>";
    return `<form class="formulario-gobernado" data-ofertas-form="publicar"><fieldset><legend>${escapar(traducir("publicar_titulo"))}</legend><div class="rejilla-formulario">` +
      campo("categoria", "text", "categoria", b.categoria, ' required minlength="2" maxlength="2000"') +
      campo("centro", "text", "centro", b.centro, ' required minlength="2" maxlength="2000"') +
      campo("fecha_inicio", "date", "fecha_inicio", b.fecha_inicio, " required") +
      campo("fecha_fin", "date", "fecha_fin", b.fecha_fin) +
      campo("descripcion", "textarea", "descripcion", b.descripcion) +
      `</div><div class="acciones-formulario"><button type="submit" class="boton-primario"${inactivo}>${escapar(traducir(estado.enviando ? "publicando" : estado.claveEnCurso ? "reintentar" : "publicar"))}</button></div></fieldset></form>`;
  }

  function renderizar() {
    const total = estado.carga === "lista" ? `<span class="estado-chip info">${escapar(traducir("total", { n: estado.ofertas.length }))}</span>` : "";
    let cuerpo;
    if (estado.carga === "cargando" || estado.carga === "inactiva") cuerpo = `<p role="status">${escapar(traducir("cargando"))}</p>`;
    else if (estado.carga === "error") cuerpo = `<p class="mensaje-error" role="alert">${escapar(estado.error)} <button type="button" class="boton-secundario" data-ofertas-accion="recargar">${escapar(traducir("reintentar_carga"))}</button></p>`;
    else if (estado.ofertas.length === 0) cuerpo = `<p>${escapar(traducir("vacio"))}</p>`;
    else cuerpo = `<div class="tabla-contenedor" tabindex="0"><table class="tabla-datos"><thead><tr><th scope="col">${escapar(traducir("col_oferta"))}</th><th scope="col">${escapar(traducir("col_fechas"))}</th><th scope="col">${escapar(traducir("col_plazo"))}</th><th scope="col" class="numero">${escapar(traducir("col_disposiciones"))}</th><th scope="col">${escapar(traducir("col_estado"))}</th><th scope="col">${escapar(traducir("col_propuesta"))}</th></tr></thead><tbody>${estado.ofertas.map(filaOferta).join("")}</tbody></table></div>`;
    const avisos = `${estado.mensaje ? `<p class="mensaje-exito" role="status">${escapar(estado.mensaje)}</p>` : ""}${estado.errorOperacion ? `<p class="mensaje-error" role="alert">${escapar(estado.errorOperacion)}</p>` : ""}`;
    return `<section class="panel panel-separado ofertas-bolsa" aria-labelledby="titulo-ofertas-bolsa"><div class="cabecera-panel"><div><h3 id="titulo-ofertas-bolsa">${escapar(traducir("titulo"))}</h3><p>${escapar(traducir("subtitulo"))}</p></div>${total}</div><div class="cuerpo-panel">${formulario()}${avisos}${cuerpo}</div></section>`;
  }

  function manejarSubmit(evento) {
    const form = evento.target?.closest?.('[data-ofertas-form="publicar"]');
    if (!form) return false;
    evento.preventDefault();
    if (typeof form.reportValidity === "function" && !form.reportValidity()) return true;
    const valores = new FormData(form);
    const datos = { categoria: String(valores.get("categoria") ?? "").trim(), centro: String(valores.get("centro") ?? "").trim(),
      fecha_inicio: String(valores.get("fecha_inicio") ?? ""), descripcion: String(valores.get("descripcion") ?? "").trim() };
    const fin = String(valores.get("fecha_fin") ?? "");
    if (fin) datos.fecha_fin = fin;
    void publicar(datos);
    return true;
  }

  function manejarClick(evento) {
    const control = evento.target?.closest?.("[data-ofertas-accion]");
    if (!control || control.disabled) return false;
    if (control.dataset.ofertasAccion === "recargar") void cargar();
    else if (control.dataset.ofertasAccion === "resolver") void resolver(control.dataset.ofertaRef, control.dataset.participacionRef || null);
    else return false;
    return true;
  }

  return Object.freeze({
    activar(bolsaRef) {
      if (!bolsaRef || bolsaRef === estado.bolsa) return;
      Object.assign(estado, { bolsa: bolsaRef, carga: "inactiva", ofertas: [], mensaje: "", errorOperacion: "", claveEnCurso: null, borrador: null });
      estado.clavesResolucion.clear();
      void cargar();
    },
    desmontar() { controlador?.abort(); controlador = null; estado.bolsa = ""; estado.carga = "inactiva"; },
    renderizar,
    instalar(documento) {
      documento.addEventListener("submit", (evento) => { manejarSubmit(evento); });
      documento.addEventListener("click", (evento) => { manejarClick(evento); });
    },
    manejarSubmit, manejarClick, estado: () => ({ ...estado }),
  });
}
