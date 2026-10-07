import { traducirPortal } from "../../portal-i18n.js?v=20261001-ct-a-i18n-v1";
import { crearClientePoliticaOfertas, validarPoliticaEditable, cargarEjemploPlazas, cargarConfirmacionAdjudicacion, plazasCompletas, leerReglasVigentes,
  LLAMADAS_PLAZAS, TRAS_RENUNCIA_PLAZAS, MAXIMO_HORAS_RESPUESTA } from "./rrhh-plazos-api.js?v=20261007-p5-solicitudes-reglas-v1";

// Los textos de plazos y plazas viven en textos/<idioma>/bolsa-ofertas.json.
const { cargarTextos } = await import("../../../comun/textos.js");
const CATALOGO_TEXTOS = await cargarTextos("bolsa-ofertas");
const TEXTOS_PLAZAS = CATALOGO_TEXTOS.seccion("politica_plazas");
const TEXTOS_PLAZO = CATALOGO_TEXTOS.seccion("politica_plazo");
const PLAZAS_VACIAS = Object.freeze({ llamada: "", respuesta_horas: null, tras_renuncia: "" });
const CLAVE_PLAZO_CATALOGO = "b10.plazo_publicacion";

const EJEMPLO_VACIO = Object.freeze({
  plazo: { unidad: "", cantidad: null, computo: "", municipio_sede: "", inicio: "" },
  adjudicacion: { criterio: "orden_vigente", elegibilidad: "disposicion_en_plazo" },
  no_cubierta: { accion: "llamamiento_directo", condicion: "sin_disposiciones_elegibles" },
});

function escapar(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}

function copia(valor) { return structuredClone(valor); }

function claveNueva() {
  return globalThis.crypto?.randomUUID?.() ?? `politica-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

/** Propuesta del catálogo de Bolsa; sin regla no se propone una cifra. */
export async function cargarPlazoCatalogo({ cliente } = {}) {
  try {
    const datos = await leerReglasVigentes({ cliente });
    const reglas = datos.catalogos.filter((catalogo) => catalogo.modulo === "bolsa" && catalogo.estado === "disponible")
      .flatMap((catalogo) => catalogo.reglas).filter((regla) => regla.clave === CLAVE_PLAZO_CATALOGO);
    if (reglas.length !== 1) return null;
    const regla = reglas[0];
    const horas = regla.unidad === "horas_naturales";
    const dias = regla.unidad === "dias_habiles" || regla.unidad === "dias_naturales";
    if (regla.origen !== "reglamento" || (!horas && !dias)
      || regla.computo !== (horas ? "continuo_utc" : "administrativo")
      || regla.inicio !== "notificacion" || !Number.isSafeInteger(regla.cantidad)
      || regla.cantidad < 1 || regla.cantidad > (horas ? 720 : 30)) return null;
    return { unidad: regla.unidad, cantidad: regla.cantidad, computo: regla.computo, municipio_sede: "", inicio: regla.inicio };
  } catch {
    return null;
  }
}

/** La capacidad se consulta aparte; el POST de publicación vuelve a autorizar. */
export function crearSuperficieRRHHPlazos({
  cliente = crearClientePoliticaOfertas(), traducir = traducirPortal,
  alCambiar = () => {}, anunciar = () => {},
  generarClave = claveNueva, cargarEjemplo = cargarEjemploPlazas, cargarPlazo = cargarPlazoCatalogo,
  cargarConfirmacion = cargarConfirmacionAdjudicacion,
  textosPlazas = TEXTOS_PLAZAS, textosPlazo = TEXTOS_PLAZO,
} = {}) {
  const estado = { bolsaRef: "", carga: "inactiva", vigente: null, borrador: { ...copia(EJEMPLO_VACIO), plazas: copia(PLAZAS_VACIAS) },
    error: "", mensaje: "", guardando: false, conflicto: false, clave: "", cargaId: 0,
    puedePublicar: false, capacidadCargando: false, capacidadError: false, ayudaAbierta: false,
    ejemploPlazas: null, plazasTocadas: false, plazoCatalogo: null, plazoTocado: false, confirmacionCatalogo: null };
  let ejemploPedido = false;
  let plazoPedido = false;
  let confirmacionPedida = null;
  let lectura = null;
  let capacidad = null;
  let escritura = null;
  let documentoInstalado = null;
  const repintar = () => alCambiar();
  const politicaLegada = () => estado.vigente?.configurada === true && estado.vigente.politica?.plazo?.inicio === undefined;
  const puedeEditar = () => estado.carga === "lista" && estado.puedePublicar === true;
  const t = (clave, variables) => escapar(traducir(`rrhh_plazos_${clave}`, variables));
  const tp = (clave) => escapar(textosPlazas[clave] ?? clave);

  function pedirConfirmacionCatalogo() {
    confirmacionPedida ??= Promise.resolve().then(cargarConfirmacion).then((confirmacion) => {
      estado.confirmacionCatalogo = confirmacion === "aceptacion_previa" ? confirmacion : null;
      repintar();
    }).catch(() => { estado.confirmacionCatalogo = null; });
    return confirmacionPedida;
  }

  // Un borrador sin apartado de plazas lo recibe vacío o con la propuesta del
  // paquete de ejemplo, si existe y la persona aún no lo ha tocado.
  function conPlazas(borrador) {
    if (!plazasCompletas(borrador.plazas)) borrador.plazas = copia(estado.ejemploPlazas ?? PLAZAS_VACIAS);
    return borrador;
  }

  function conPlazo(borrador) {
    if (borrador.plazo.cantidad === null && estado.plazoCatalogo) borrador.plazo = copia(estado.plazoCatalogo);
    else if (borrador.plazo.inicio === undefined && estado.plazoCatalogo) borrador.plazo.inicio = estado.plazoCatalogo.inicio;
    return borrador;
  }

  async function pedirPlazoCatalogo() {
    if (plazoPedido) return;
    plazoPedido = true;
    let ejemplo;
    try { ejemplo = await cargarPlazo(); }
    catch { plazoPedido = false; return; }
    if (!ejemplo) { plazoPedido = false; return; }
    estado.plazoCatalogo = ejemplo;
    if (estado.bolsaRef && !estado.vigente?.configurada && !estado.plazoTocado
      && estado.borrador.plazo.cantidad === null) {
      estado.borrador.plazo = copia(ejemplo);
      repintar();
    } else if (estado.bolsaRef && politicaLegada() && estado.borrador.plazo.inicio === undefined) {
      estado.borrador.plazo.inicio = ejemplo.inicio;
      repintar();
    }
  }

  async function pedirEjemploPlazas() {
    if (ejemploPedido) return;
    ejemploPedido = true;
    const ejemplo = await cargarEjemplo();
    if (!ejemplo) return;
    estado.ejemploPlazas = ejemplo;
    if (!estado.plazasTocadas && !plazasCompletas(estado.borrador.plazas)) { estado.borrador.plazas = copia(ejemplo); repintar(); }
  }

  async function cargarCapacidad(bolsa, id) {
    capacidad?.abort();
    capacidad = new AbortController();
    const controlador = capacidad;
    estado.puedePublicar = false;
    estado.capacidadCargando = true;
    estado.capacidadError = false;
    repintar();
    try {
      const resultado = await cliente.consultarCapacidad(bolsa, { signal: controlador.signal });
      if (controlador.signal.aborted || id !== estado.cargaId || bolsa !== estado.bolsaRef) return;
      estado.puedePublicar = resultado.ok === true && resultado.puede_publicar === true;
      estado.capacidadError = resultado.ok !== true && resultado.status !== 403;
    } catch (error) {
      if (controlador.signal.aborted || id !== estado.cargaId || bolsa !== estado.bolsaRef || error?.name === "AbortError") return;
      estado.puedePublicar = false;
      estado.capacidadError = true;
    }
    estado.capacidadCargando = false;
    repintar();
  }

  async function cargar({ conservarBorrador = false } = {}) {
    if (!estado.bolsaRef) return;
    void pedirConfirmacionCatalogo();
    if (!estado.plazoCatalogo) void pedirPlazoCatalogo();
    lectura?.abort();
    capacidad?.abort();
    lectura = new AbortController();
    const controlador = lectura;
    const bolsa = estado.bolsaRef;
    const id = ++estado.cargaId;
    estado.carga = "cargando"; estado.error = ""; estado.puedePublicar = false;
    estado.capacidadCargando = false; estado.capacidadError = false; repintar();
    try {
      const resultado = await cliente.consultar(bolsa, { signal: controlador.signal });
      if (controlador.signal.aborted || id !== estado.cargaId || bolsa !== estado.bolsaRef) return;
      if (!resultado.ok) {
        estado.vigente = null;
        estado.carga = resultado.status === 403 ? "denegado" : "error";
        estado.error = traducir(`rrhh_plazos_${resultado.status === 403 ? "denegado" : "error_carga"}`);
      } else {
        estado.vigente = resultado.politica;
        if (!conservarBorrador) {
          estado.borrador = conPlazas(conPlazo(copia(resultado.politica.configurada ? resultado.politica.politica : EJEMPLO_VACIO)));
          estado.plazasTocadas = false;
          estado.plazoTocado = false;
          estado.clave = "";
        }
        estado.carga = "lista"; estado.conflicto = false;
        void cargarCapacidad(bolsa, id);
      }
    } catch (error) {
      if (controlador.signal.aborted || id !== estado.cargaId || bolsa !== estado.bolsaRef || error?.name === "AbortError") return;
      estado.carga = "error"; estado.error = traducir("rrhh_plazos_error_carga");
    }
    repintar();
  }

  async function guardar() {
    if (!puedeEditar() || estado.guardando || estado.conflicto) return;
    const bolsa = estado.bolsaRef;
    const id = estado.cargaId;
    await pedirConfirmacionCatalogo();
    if (!puedeEditar() || estado.guardando || estado.conflicto || bolsa !== estado.bolsaRef || id !== estado.cargaId) return;
    if (estado.confirmacionCatalogo) estado.borrador.adjudicacion.confirmacion = estado.confirmacionCatalogo;
    let politica;
    try { politica = validarPoliticaEditable(estado.borrador); }
    catch { estado.error = traducir("rrhh_plazos_validacion"); repintar(); return; }
    const version = estado.vigente.version;
    const clave = estado.clave || generarClave();
    estado.clave = clave;
    estado.guardando = true; estado.error = ""; estado.mensaje = ""; repintar();
    escritura = new AbortController();
    const controlador = escritura;
    try {
      const resultado = await cliente.publicar({ bolsa_ref: bolsa, version_esperada: version,
        clave_idempotencia: clave, politica }, { signal: controlador.signal });
      if (controlador.signal.aborted || id !== estado.cargaId || bolsa !== estado.bolsaRef) return;
      if (resultado.ok) {
        estado.vigente = resultado.politica;
        estado.puedePublicar = false;
        estado.borrador = conPlazas(copia(resultado.politica.politica));
        estado.clave = "";
        estado.mensaje = traducir("rrhh_plazos_guardada", { version: resultado.politica.version,
          recibo: resultado.politica.recibo_ref });
        anunciar(estado.mensaje);
        void cargarCapacidad(bolsa, id);
      } else if (resultado.status === 409) {
        estado.conflicto = true;
        estado.clave = "";
        estado.error = traducir("rrhh_plazos_conflicto");
      } else {
        if (resultado.status === 403) {
          estado.puedePublicar = false;
          estado.capacidadCargando = false;
          estado.capacidadError = false;
        }
        estado.error = traducir(`rrhh_plazos_${resultado.status === 403 ? "sin_permiso" : resultado.status === 422 ? "validacion" : "error_guardado"}`);
      }
    } catch (error) {
      if (controlador.signal.aborted || id !== estado.cargaId || bolsa !== estado.bolsaRef || error?.name === "AbortError") return;
      estado.error = traducir("rrhh_plazos_error_guardado");
    } finally {
      if (id === estado.cargaId) { estado.guardando = false; repintar(); }
    }
  }

  function dato(etiqueta, valor) {
    return `<div class="campo"><span>${t(etiqueta)}</span><p class="rrhh-plazos__valor">${valor}</p></div>`;
  }

  function renderizar() {
    const configurada = estado.vigente?.configurada === true;
    const v = estado.borrador;
    const inicio = v.plazo.inicio === "notificacion" ? t("inicio_notificacion")
      : configurada ? t("inicio_no_registrado") : t("desconocido");
    const deshabilitado = !puedeEditar() || estado.guardando || estado.conflicto;
    const guardarDeshabilitado = deshabilitado || v.plazo.inicio !== "notificacion";
    const disabled = deshabilitado ? " disabled" : "";
    const cabecera = `<div class="cabecera-panel"><div><h3>${t("titulo")}</h3><p>${t("subtitulo")}</p></div><div class="rrhh-plazos__cabecera-estado"><span class="estado-chip advertencia">${t("ejemplo")}</span>${configurada ? `<span class="estado-chip info">${t("version", { version: estado.vigente.version })}</span>` : ""}<button type="button" class="rrhh-plazos__ayuda" data-rrhh-plazos-accion="ayuda" aria-label="${t("ayuda")}" aria-expanded="${estado.ayudaAbierta}" aria-controls="rrhh-plazos-ayuda">?</button></div></div>
      <p id="rrhh-plazos-ayuda" class="rrhh-plazos__ayuda-texto" ${estado.ayudaAbierta ? "" : "hidden"}>${escapar(textosPlazo.ayuda)}</p>`;
    if (estado.carga === "inactiva" || estado.carga === "cargando") {
      return `<section class="panel rrhh-plazos" aria-busy="true">${cabecera}<div class="cuerpo-panel" role="status">${t("cargando")}</div></section>`;
    }
    if (estado.carga !== "lista") {
      return `<section class="panel rrhh-plazos">${cabecera}<div class="cuerpo-panel"><p class="rrhh-plazos__error" role="alert">${escapar(estado.error)}</p>${estado.carga === "error" ? `<button type="button" class="boton-secundario" data-rrhh-plazos-accion="recargar">${t("reintentar")}</button>` : ""}</div></section>`;
    }
    const horas = v.plazo.unidad === "horas_naturales";
    const calendario = !v.plazo.unidad ? t("desconocido") : horas ? t("calendario_continuo_utc") : /^[0-9]{5}$/u.test(v.plazo.municipio_sede)
      ? t("calendario_municipio", { municipio: v.plazo.municipio_sede }) : t("calendario_pendiente");
    const plazo = `<section class="panel"><div class="cabecera-panel"><h3>${t("plazo_titulo")}</h3></div><div class="cuerpo-panel rrhh-plazos__campos"><label class="campo"><span>${t("cantidad")}</span><input name="cantidad" type="number" min="1" max="${horas ? 720 : 30}" step="1" required value="${escapar(v.plazo.cantidad)}"${disabled}></label><label class="campo"><span>${t("unidad")}</span><select name="unidad" required${disabled}><option value=""${v.plazo.unidad ? "" : " selected"}>${tp("elegir")}</option><option value="horas_naturales"${horas ? " selected" : ""}>${t("horas_naturales")}</option><option value="dias_habiles"${v.plazo.unidad === "dias_habiles" ? " selected" : ""}>${t("dias_habiles")}</option><option value="dias_naturales"${v.plazo.unidad === "dias_naturales" ? " selected" : ""}>${t("dias_naturales")}</option></select></label><label class="campo"><span>${t("municipio_sede")}</span><input name="municipio_sede" inputmode="numeric" pattern="[0-9]{5}" minlength="5" maxlength="5" required value="${escapar(v.plazo.municipio_sede)}"${disabled}></label>${dato("inicio", inicio)}${dato("computo", t(v.plazo.computo || "desconocido"))}${dato("calendario_fuente", calendario)}</div></section>`;
    const aceptacionPrevia = (estado.confirmacionCatalogo ?? v.adjudicacion.confirmacion) === "aceptacion_previa";
    const orden = `<section class="panel"><div class="cabecera-panel"><h3>${t("orden_titulo")}</h3></div><div class="cuerpo-panel rrhh-plazos__campos">${dato("criterio", t("orden_vigente"))}${dato("elegibilidad", t("disposicion_en_plazo"))}<p class="campo--ancho dato-secundario">${aceptacionPrevia ? tp("aceptacion_previa") : t("confirmacion")}</p></div></section>`;
    const noCubierta = `<section class="panel"><div class="cabecera-panel"><h3>${t("no_cubierta_titulo")}</h3></div><div class="cuerpo-panel rrhh-plazos__campos">${dato("condicion", t("sin_elegibles"))}${dato("accion", t("llamamiento_directo"))}</div></section>`;
    const p = v.plazas ?? PLAZAS_VACIAS;
    const opciones = (valores, actual, prefijo) => `<option value=""${actual ? "" : " selected"}>${tp("elegir")}</option>` +
      valores.map((valor) => `<option value="${valor}"${actual === valor ? " selected" : ""}>${tp(`${prefijo}_${valor}`)}</option>`).join("");
    const plazas = `<section class="panel"><div class="cabecera-panel"><h3>${tp("titulo")}</h3></div><div class="cuerpo-panel rrhh-plazos__campos">` +
      `<label class="campo campo--ancho"><span>${tp("llamada")}</span><select name="plazas_llamada" required${disabled}>${opciones(LLAMADAS_PLAZAS, p.llamada, "llamada")}</select></label>` +
      (aceptacionPrevia && plazasCompletas(p) ? "" : `<label class="campo"><span>${tp("respuesta_horas")}</span><input name="plazas_respuesta_horas" type="number" min="1" max="${MAXIMO_HORAS_RESPUESTA}" step="1" required value="${escapar(p.respuesta_horas ?? "")}"${disabled}></label>` +
      `<label class="campo campo--ancho"><span>${tp("tras_renuncia")}</span><select name="plazas_tras_renuncia" required${disabled}>${opciones(TRAS_RENUNCIA_PLAZAS, p.tras_renuncia, "tras_renuncia")}</select></label>`) + `</div></section>`;
    const aviso = `${!configurada ? `<p role="status">${t("vacio")}</p>` : ""}${politicaLegada() ? `<p role="status">${t("version_anterior")}</p>` : ""}${estado.error ? `<p class="rrhh-plazos__error" role="alert">${escapar(estado.error)}</p>` : ""}${estado.mensaje ? `<p class="rrhh-plazos__resultado" role="status">${escapar(estado.mensaje)}</p>` : ""}`;
    const boton = puedeEditar() ? `<div class="rrhh-plazos__acciones">${v.plazo.inicio !== "notificacion" ? `<p class="dato-secundario" role="status">${t("inicio_pendiente")}</p>` : ""}${estado.conflicto ? `<button type="button" class="boton-secundario" data-rrhh-plazos-accion="revisar">${t("revisar")}</button>` : ""}<button type="submit" class="boton-primario"${guardarDeshabilitado ? " disabled" : ""}>${t(estado.guardando ? "guardando" : estado.clave ? "reintentar_guardado" : "guardar")}</button></div>`
      : estado.capacidadCargando ? `<p class="dato-secundario" role="status">${t("comprobando_edicion")}</p>`
        : estado.capacidadError ? `<div class="rrhh-plazos__acciones"><p class="rrhh-plazos__error" role="alert">${t("error_capacidad")}</p><button type="button" class="boton-secundario" data-rrhh-plazos-accion="capacidad">${t("reintentar")}</button></div>`
          : `<p class="dato-secundario">${t("sin_edicion")}</p>`;
    return `<section class="rrhh-plazos" aria-label="${t("titulo")}"><div class="panel">${cabecera}${aviso ? `<div class="cuerpo-panel">${aviso}</div>` : ""}</div><form data-rrhh-plazos-form="politica" novalidate><div class="rrhh-plazos__rejilla">${plazo}${orden}${noCubierta}${plazas}</div>${boton}</form></section>`;
  }

  function manejarCambio(evento) {
    const control = evento.target;
    if (!control?.closest?.('[data-rrhh-plazos-form="politica"]') || !puedeEditar()) return false;
    const campoPlazas = { plazas_llamada: "llamada", plazas_respuesta_horas: "respuesta_horas", plazas_tras_renuncia: "tras_renuncia" }[control.name];
    if (campoPlazas) {
      const valor = campoPlazas === "respuesta_horas" ? (control.value === "" ? null : Number(control.value)) : control.value;
      estado.borrador.plazas = { ...(estado.borrador.plazas ?? PLAZAS_VACIAS), [campoPlazas]: valor };
      estado.plazasTocadas = true; estado.clave = ""; estado.mensaje = ""; estado.error = "";
      return true;
    }
    const campo = { cantidad: "cantidad", unidad: "unidad", municipio_sede: "municipio_sede" }[control.name];
    if (!campo) return false;
    const valor = campo === "cantidad" ? Number(control.value) : campo === "municipio_sede" ? control.value.trim() : control.value;
    if (estado.borrador.plazo[campo] === valor) return true;
    if (campo === "unidad") {
      const anterior = estado.borrador.plazo.unidad;
      estado.borrador.plazo.unidad = valor;
      const horas = valor === "horas_naturales";
      estado.borrador.plazo.computo = horas ? "continuo_utc" : "administrativo";
      if (anterior !== valor) {
        const referencia = estado.vigente?.configurada ? estado.vigente.politica.plazo : estado.plazoCatalogo;
        estado.borrador.plazo.cantidad = referencia?.unidad === valor ? referencia.cantidad : null;
      }
      repintar();
    } else estado.borrador.plazo[campo] = valor;
    estado.plazoTocado = true;
    estado.clave = ""; estado.mensaje = ""; estado.error = "";
    return true;
  }

  function manejarSubmit(evento) {
    const formulario = evento.target?.closest?.('[data-rrhh-plazos-form="politica"]');
    if (!formulario) return false;
    evento.preventDefault();
    if (!formulario.reportValidity?.()) { estado.error = traducir("rrhh_plazos_validacion"); repintar(); return true; }
    void guardar(); return true;
  }

  function manejarClick(evento) {
    const boton = evento.target?.closest?.("[data-rrhh-plazos-accion]");
    if (!boton || boton.disabled) return false;
    if (boton.dataset.rrhhPlazosAccion === "ayuda") {
      estado.ayudaAbierta = !estado.ayudaAbierta;
      repintar();
      documentoInstalado?.querySelector?.('[data-rrhh-plazos-accion="ayuda"]')?.focus?.({ preventScroll: true });
    }
    else if (boton.dataset.rrhhPlazosAccion === "recargar") void cargar();
    else if (boton.dataset.rrhhPlazosAccion === "revisar") void cargar({ conservarBorrador: true });
    else if (boton.dataset.rrhhPlazosAccion === "capacidad") void cargarCapacidad(estado.bolsaRef, estado.cargaId);
    else return false;
    return true;
  }

  return Object.freeze({
    activar(bolsaRef) {
      if (!bolsaRef || bolsaRef === estado.bolsaRef) return;
      lectura?.abort(); capacidad?.abort(); escritura?.abort(); estado.cargaId++;
      Object.assign(estado, { bolsaRef, carga: "inactiva", vigente: null, borrador: conPlazas(conPlazo(copia(EJEMPLO_VACIO))),
        error: "", mensaje: "", guardando: false, conflicto: false, clave: "", puedePublicar: false,
        capacidadCargando: false, capacidadError: false,
        ayudaAbierta: false, plazasTocadas: false, plazoTocado: false });
      void pedirEjemploPlazas();
      void pedirPlazoCatalogo();
      void cargar();
    },
    desmontar() {
      lectura?.abort(); capacidad?.abort(); escritura?.abort(); estado.cargaId++;
      estado.bolsaRef = ""; estado.carga = "inactiva"; estado.puedePublicar = false;
      estado.capacidadCargando = false; estado.capacidadError = false;
      documentoInstalado?.removeEventListener("input", manejarCambio);
      documentoInstalado?.removeEventListener("change", manejarCambio);
      documentoInstalado?.removeEventListener("submit", manejarSubmit);
      documentoInstalado?.removeEventListener("click", manejarClick);
      documentoInstalado = null;
    },
    renderizar, manejarCambio, manejarSubmit, manejarClick,
    instalar(documento) {
      if (documentoInstalado === documento) return;
      if (documentoInstalado) throw new TypeError("superficie RRHH de plazos ya instalada");
      documentoInstalado = documento;
      documento.addEventListener("input", manejarCambio);
      documento.addEventListener("change", manejarCambio);
      documento.addEventListener("submit", manejarSubmit);
      documento.addEventListener("click", manejarClick);
    },
    estado: () => copia(estado),
  });
}
