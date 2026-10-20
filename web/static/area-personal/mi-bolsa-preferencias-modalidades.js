// Elección de modalidades de oferta en «Mi bolsa» (F1-14). La persona decide
// de qué modalidades quiere ver ofertas publicadas y recibir sus avisos. No
// cambia su inscripción, su puesto en la bolsa ni los llamamientos directos.
// Contrato del servidor (F1-13, parte 3): GET/PUT con «?tipo=modalidades»,
// versión esperada, clave de idempotencia y recibo. `null` son todas y `[]`
// ninguna. Las opciones salen del catálogo que envía el servidor.
import { localizacionAreaPersonal, traducir } from "./i18n.js";
import { escaparAtributo, escaparHTML, panel } from "./vistas/comunes.js";

export const RUTA_PREFERENCIAS_MODALIDADES = "/api/vec/bolsa/mi-bolsa/preferencias?tipo=modalidades";
const MAXIMO_BYTES = 64 * 1024;
const MAXIMO_OPCIONES = 256;
const CATALOGO_MODALIDADES = "vec.contratacion_temporal.reglas";
const PREFIJO_MODALIDAD = "c12.modalidad.";
const CLAVE = /^[a-z][a-z0-9_]{1,79}$/u;
const INSTANTE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$/u;
const CAMPOS_ESTADO = ["version", "modalidades_guardadas", "modalidades_disponibles", "recibo_ref", "registrado_en"];
const CAMPOS_FUENTE = ["catalogo_id", "catalogo_version", "catalogo_huella_sha256", "entrada_clave"];
const CAMPOS_RECIBO = ["recibo_ref", "version", "modalidades", "registrado_en", "reutilizado"];
const PREFIJO_TEXTOS = "areaPersonal.miBolsa.preferenciasModalidades";

const t = (clave, variables) => traducir(`${PREFIJO_TEXTOS}.${clave}`, variables);
const objeto = (v) => v !== null && typeof v === "object" && !Array.isArray(v);
const fallo = (codigo) => Object.assign(new Error(codigo), { codigo });
const mismasClaves = (v, claves) => objeto(v) && Object.keys(v).sort().join("|") === [...claves].sort().join("|");
const instante = (v) => typeof v === "string" && INSTANTE.test(v) && !Number.isNaN(Date.parse(v));
const listaClaves = (v) => Array.isArray(v) && v.length <= MAXIMO_OPCIONES && v.every((c) => typeof c === "string" && CLAVE.test(c)) && new Set(v).size === v.length;
const iguales = (a, b) => (a === null || b === null ? a === b : a.length === b.length && a.every((c) => b.includes(c)));
const copiar = (v) => (v === null ? null : [...v]);

// La opción solo se muestra si el catálogo de textos la nombra: nunca se
// enseña una clave interna a la persona.
export function nombreModalidad(clave, textos = t) {
  const nombre = textos(`opciones.${clave}`);
  return nombre === `${PREFIJO_TEXTOS}.opciones.${clave}` ? null : nombre;
}

export function validarEstadoPreferenciasModalidades(v, textos = t) {
  if (!mismasClaves(v, CAMPOS_ESTADO) || !Number.isSafeInteger(v.version) || v.version < 0 ||
    !(v.modalidades_guardadas === null || listaClaves(v.modalidades_guardadas)) ||
    !Array.isArray(v.modalidades_disponibles) || !v.modalidades_disponibles.length || v.modalidades_disponibles.length > MAXIMO_OPCIONES ||
    (v.recibo_ref !== null && (typeof v.recibo_ref !== "string" || !v.recibo_ref)) ||
    (v.registrado_en !== null && !instante(v.registrado_en)) ||
    (v.version === 0) !== (v.recibo_ref === null)) throw fallo("respuesta");
  const vistas = new Set();
  for (const disponible of v.modalidades_disponibles) {
    const f = disponible?.fuente;
    if (!mismasClaves(disponible, ["clave", "fuente"]) || typeof disponible.clave !== "string" || !CLAVE.test(disponible.clave) ||
      vistas.has(disponible.clave) || !mismasClaves(f, CAMPOS_FUENTE) || f.catalogo_id !== CATALOGO_MODALIDADES ||
      f.entrada_clave !== PREFIJO_MODALIDAD + disponible.clave || !Number.isSafeInteger(f.catalogo_version) || f.catalogo_version < 1 ||
      typeof f.catalogo_huella_sha256 !== "string" || !/^[a-f0-9]{64}$/u.test(f.catalogo_huella_sha256) ||
      nombreModalidad(disponible.clave, textos) === null) throw fallo("respuesta");
    vistas.add(disponible.clave);
  }
  return structuredClone(v);
}

export function validarReciboPreferenciasModalidades(v, peticion) {
  if (!mismasClaves(v, CAMPOS_RECIBO) || typeof v.recibo_ref !== "string" || !v.recibo_ref ||
    !Number.isSafeInteger(v.version) || v.version !== peticion.version_esperada + 1 ||
    !(v.modalidades === null || listaClaves(v.modalidades)) || !iguales(v.modalidades, peticion.modalidades) ||
    !instante(v.registrado_en) || typeof v.reutilizado !== "boolean") throw fallo("respuesta");
  return structuredClone(v);
}

const CODIGOS = Object.freeze({ 401: "autenticacion", 403: "denegado", 404: "noDisponible", 409: "conflicto", 422: "invalido", 503: "servicio" });

async function solicitar(api, metodo, cuerpo, signal) {
  let respuesta;
  try {
    respuesta = await api(RUTA_PREFERENCIAS_MODALIDADES, {
      method: metodo, credentials: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
      headers: { Accept: "application/json", ...(cuerpo ? { "Content-Type": "application/json" } : {}) },
      ...(cuerpo ? { body: JSON.stringify(cuerpo) } : {}), ...(signal ? { signal } : {}),
    });
  } catch { throw fallo(metodo === "PUT" ? "incierto" : "error"); }
  if (respuesta?.status !== 200 && !(metodo === "PUT" && respuesta?.status === 201)) {
    throw fallo(CODIGOS[respuesta?.status] || (metodo === "PUT" ? "incierto" : "error"));
  }
  const incierto = metodo === "PUT" ? "incierto" : "respuesta";
  let texto;
  try { texto = await respuesta.text(); } catch { throw fallo(incierto); }
  if (new TextEncoder().encode(texto).byteLength > MAXIMO_BYTES) throw fallo(incierto);
  try { return JSON.parse(texto); } catch { throw fallo(incierto); }
}

export async function cargarPreferenciasModalidades({ api = globalThis.fetch, signal, textos = t } = {}) {
  return validarEstadoPreferenciasModalidades(await solicitar(api, "GET", null, signal), textos);
}

export async function guardarPreferenciasModalidades(peticion, { api = globalThis.fetch, signal } = {}) {
  if (!mismasClaves(peticion, ["modalidades", "version_esperada", "clave_idempotencia"]) ||
    !(peticion.modalidades === null || listaClaves(peticion.modalidades)) ||
    !Number.isSafeInteger(peticion.version_esperada) || peticion.version_esperada < 0 ||
    typeof peticion.clave_idempotencia !== "string" || !/^[A-Za-z0-9._:-]{8,128}$/u.test(peticion.clave_idempotencia)) throw fallo("respuesta");
  const respuesta = await solicitar(api, "PUT", peticion, signal);
  try { return validarReciboPreferenciasModalidades(respuesta, peticion); } catch { throw fallo("incierto"); }
}

// modo traduce la elección al control visible: «todas», «algunas» o «ninguna».
export function modoEleccion(eleccion) {
  if (eleccion === null) return "todas";
  return eleccion.length ? "algunas" : "ninguna";
}

function fecha(valor, idioma) {
  return new Intl.DateTimeFormat(idioma(), { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }).format(new Date(valor));
}

const ERRORES = new Set(["autenticacion", "denegado", "conflicto", "invalido", "respuesta", "servicio", "incierto", "error"]);

export function renderizarPreferenciasModalidades(datos = null, {
  estado = "cargando", borrador = null, modo = null, aviso = "", recibo = null, pendiente = false, textos = t, idioma = localizacionAreaPersonal,
} = {}) {
  if (estado === "oculto") return "";
  const tr = (clave, vars) => escaparHTML(textos(clave, vars));
  const titulo = textos("titulo");
  if (!datos) {
    if (estado === "cargando") return panel(titulo, "", `<p role="status">${tr("cargando")}</p>`, { clase: "preferencias-modalidades" });
    const codigo = ERRORES.has(estado) ? estado : "error";
    const boton = codigo === "autenticacion" ? "" : `<div class="fila-acciones"><button type="button" class="boton-secundario" data-modalidades="recargar">${tr("recargar")}</button></div>`;
    return panel(titulo, "", `<div class="nota error" role="alert"><p>${tr(codigo)}</p></div>${boton}`, { clase: "preferencias-modalidades" });
  }
  const actual = modo ?? modoEleccion(borrador);
  const bloqueado = pendiente || estado === "guardando" || estado === "cargando";
  const deshabilitado = bloqueado ? "disabled" : "";
  const radio = (valor) => `<label class="opcion-check"><input type="radio" name="preferencias-modalidades-modo" value="${valor}" data-modalidades-modo ${actual === valor ? "checked" : ""} ${deshabilitado}><span>${tr(`modo.${valor}`)}</span></label>`;
  const casillas = actual === "algunas" ? `<fieldset class="preferencias-modalidades__grupo"><legend>${tr("modalidades")}</legend>${datos.modalidades_disponibles.map(({ clave }) =>
    `<label class="opcion-check"><input type="checkbox" data-modalidades-opcion="${escaparAtributo(clave)}" ${borrador?.includes(clave) ? "checked" : ""} ${deshabilitado}><span>${escaparHTML(nombreModalidad(clave, textos))}</span></label>`).join("")}</fieldset>` : "";
  const eleccion = actual === "todas" ? null : actual === "ninguna" ? [] : borrador ?? [];
  const sinMarcar = actual === "algunas" && !eleccion.length;
  const cambios = !iguales(eleccion, datos.modalidades_guardadas);
  // Aviso por elegir «ninguna»: se lee en cuanto se marca, antes de guardar.
  const avisoNinguna = actual === "ninguna" ? `<p class="nota aviso" id="preferencias-modalidades-ninguna">${tr("avisoNinguna")}</p>` : "";
  const avisoMarcar = sinMarcar ? `<p class="nota aviso" id="preferencias-modalidades-marcar">${tr("marqueUna")}</p>` : "";
  let mensaje = "";
  if (estado === "guardando") mensaje = `<p role="status">${tr("guardando")}</p>`;
  else if (ERRORES.has(estado) || aviso) mensaje = `<p class="nota ${aviso === "conflicto" ? "aviso" : "error"}" role="alert">${tr(aviso || estado)}</p>`;
  const guardado = recibo || (datos.recibo_ref && datos.registrado_en ? { registrado_en: datos.registrado_en } : null);
  const pie = guardado ? `<p class="preferencias-modalidades__recibo"${estado === "guardado" ? ' role="status"' : ""}>${tr(estado === "guardado" ? "guardado" : "ultimoGuardado", { fecha: fecha(guardado.registrado_en, idioma) })}</p>` : "";
  const reintentar = pendiente && (estado === "incierto" || estado === "servicio");
  const describe = [avisoNinguna && "preferencias-modalidades-ninguna", avisoMarcar && "preferencias-modalidades-marcar"].filter(Boolean).join(" ");
  const botonGuardar = `<button type="button" class="boton-primario" data-modalidades="${reintentar ? "reintentar" : "guardar"}" ${!reintentar && (bloqueado || !cambios || sinMarcar) ? "disabled" : ""}${describe ? ` aria-describedby="${describe}"` : ""}>${tr(reintentar ? "reintentarGuardado" : "guardar")}</button>`;
  const recargar = estado === "incierto" || estado === "servicio" || estado === "respuesta" || estado === "error" ? `<button type="button" class="boton-secundario" data-modalidades="recargar">${tr("recargar")}</button>` : "";
  const contenido = `<fieldset class="preferencias-modalidades__grupo"><legend>${tr("pregunta")}</legend>${radio("todas")}${radio("algunas")}${radio("ninguna")}</fieldset>
    ${casillas}${avisoNinguna}${avisoMarcar}<div aria-live="polite">${mensaje}</div>
    <div class="fila-acciones">${botonGuardar}${recargar}</div>${pie}`;
  return panel(titulo, "", contenido, { clase: "preferencias-modalidades" });
}

// crearControladorPreferenciasModalidades monta el panel en `host`. Si la
// consulta está denegada o no existe para esta persona, el panel no se
// muestra: la función no está activada para ella. `alGuardar` recibe la
// elección guardada para refrescar las ofertas visibles.
export function crearControladorPreferenciasModalidades({ host, api = globalThis.fetch, textos = t, idioma = localizacionAreaPersonal, alGuardar = () => {} } = {}) {
  let datos = null, borrador = null, modo = null, recibo = null, estado = "cargando", aviso = "", pendiente = null;
  let lectura = null, escritura = null, secuencia = 0, destruido = false, montado = false;
  const pintar = () => {
    if (destruido || !host) return;
    host.hidden = estado === "oculto";
    host.innerHTML = renderizarPreferenciasModalidades(datos, { estado, borrador, modo, aviso, recibo, pendiente: Boolean(pendiente), textos, idioma });
  };
  async function cargar({ conservarAviso = false } = {}) {
    if (destruido || lectura || escritura) return;
    const controller = new AbortController(); lectura = controller;
    const propia = ++secuencia; estado = "cargando"; if (!conservarAviso) aviso = ""; pintar();
    try {
      const nuevo = await cargarPreferenciasModalidades({ api, signal: controller.signal, textos });
      if (destruido || propia !== secuencia) return;
      datos = nuevo; borrador = copiar(nuevo.modalidades_guardadas); modo = null; recibo = null; estado = "correcto";
      if (pendiente) {
        // Tras un guardado sin confirmar: si ya consta, se da por guardado;
        // si la versión cambió por otra vía, se avisa del conflicto.
        if (nuevo.version > pendiente.version_esperada && iguales(nuevo.modalidades_guardadas, pendiente.modalidades) && nuevo.registrado_en) {
          recibo = { registrado_en: nuevo.registrado_en }; pendiente = null; estado = "guardado"; alGuardar(copiar(nuevo.modalidades_guardadas));
        } else if (nuevo.version !== pendiente.version_esperada) { pendiente = null; aviso = "conflicto"; }
        else estado = "incierto";
      }
      pintar();
    } catch (error) {
      if (destruido || propia !== secuencia || controller.signal.aborted) return;
      // Sin concesión o sin ruta, la función no está activada: no se muestra.
      estado = !datos && (error.codigo === "denegado" || error.codigo === "noDisponible") ? "oculto" : ERRORES.has(error.codigo) ? error.codigo : "error";
      pintar();
    } finally { if (lectura === controller) lectura = null; }
  }
  function eleccionActual() {
    const actual = modo ?? modoEleccion(borrador);
    return actual === "todas" ? null : actual === "ninguna" ? [] : borrador ?? [];
  }
  async function guardar() {
    if (destruido || escritura || lectura || !datos) return;
    if (!pendiente) {
      const eleccion = eleccionActual();
      if (iguales(eleccion, datos.modalidades_guardadas) || (modo ?? modoEleccion(borrador)) === "algunas" && !eleccion.length) return;
      if (eleccion?.some((c) => !datos.modalidades_disponibles.some((o) => o.clave === c))) return;
      if (typeof globalThis.crypto?.randomUUID !== "function") { estado = "error"; pintar(); return; }
      pendiente = { modalidades: copiar(eleccion), version_esperada: datos.version, clave_idempotencia: globalThis.crypto.randomUUID() };
    }
    const peticion = pendiente, controller = new AbortController(); escritura = controller; estado = "guardando"; aviso = ""; pintar();
    try {
      const nuevo = await guardarPreferenciasModalidades(peticion, { api, signal: controller.signal });
      if (destruido || escritura !== controller) return;
      pendiente = null; recibo = { registrado_en: nuevo.registrado_en };
      datos = { ...datos, version: nuevo.version, modalidades_guardadas: copiar(nuevo.modalidades), recibo_ref: nuevo.recibo_ref, registrado_en: nuevo.registrado_en };
      borrador = copiar(nuevo.modalidades); modo = null; estado = "guardado"; pintar();
      host?.querySelector?.('[data-modalidades="guardar"]')?.focus?.();
      alGuardar(copiar(nuevo.modalidades));
    } catch (error) {
      if (destruido || controller.signal.aborted) return;
      if (error.codigo === "conflicto") {
        // Otra ventana guardó antes: se recarga la elección vigente y se avisa.
        pendiente = null; escritura = null; aviso = "conflicto"; void cargar({ conservarAviso: true }); return;
      }
      if (["invalido", "denegado", "autenticacion", "noDisponible"].includes(error.codigo)) pendiente = null;
      estado = ERRORES.has(error.codigo) ? error.codigo : "error";
      pintar();
    } finally { if (escritura === controller) escritura = null; }
  }
  function pulsar(evento) {
    const boton = evento.target?.closest?.("[data-modalidades]");
    if (!boton || destruido || boton.disabled) return;
    const accion = boton.dataset.modalidades;
    if (accion === "recargar") void cargar();
    else if (accion === "guardar" || accion === "reintentar") void guardar();
  }
  function cambiar(evento) {
    if (destruido || pendiente || escritura || lectura || !datos) return;
    const el = evento.target;
    let selector;
    if (el?.hasAttribute?.("data-modalidades-modo") && ["todas", "algunas", "ninguna"].includes(el.value)) {
      modo = el.value;
      if (modo === "algunas" && !borrador?.length) borrador = [];
      selector = `[data-modalidades-modo][value="${el.value}"]`;
    } else if (el?.hasAttribute?.("data-modalidades-opcion") && datos.modalidades_disponibles.some((o) => o.clave === el.dataset.modalidadesOpcion)) {
      const elegidas = new Set(borrador ?? []);
      if (el.checked) elegidas.add(el.dataset.modalidadesOpcion); else elegidas.delete(el.dataset.modalidadesOpcion);
      borrador = datos.modalidades_disponibles.map((o) => o.clave).filter((c) => elegidas.has(c)); modo = "algunas";
      selector = `[data-modalidades-opcion="${el.dataset.modalidadesOpcion}"]`;
    } else return;
    estado = "correcto";
    aviso = ""; pintar();
    host?.querySelector?.(selector)?.focus?.();
  }
  return {
    montar() {
      if (!host || destruido) return null;
      if (!montado) { host.addEventListener("click", pulsar); host.addEventListener("change", cambiar); montado = true; }
      pintar(); if (!datos && !lectura) void cargar(); return this;
    },
    destruir() {
      if (montado && host) { host.removeEventListener("click", pulsar); host.removeEventListener("change", cambiar); }
      destruido = true; montado = false; secuencia++; lectura?.abort(); escritura?.abort(); lectura = null; escritura = null; pendiente = null;
    },
  };
}
