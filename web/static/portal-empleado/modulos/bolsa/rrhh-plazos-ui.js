import { traducirPortal } from "../../portal-i18n.js";
import { crearClientePoliticaOfertas, validarPoliticaEditable } from "./rrhh-plazos-api.js";

const EJEMPLO_VACIO = Object.freeze({
  plazo: { unidad: "dias_habiles", cantidad: 1, computo: "administrativo", municipio_sede: "" },
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

/** Superficie independiente: la composición RRHH decide dónde montarla y concede puedeEditar. */
export function crearSuperficieRRHHPlazos({
  cliente = crearClientePoliticaOfertas(), traducir = traducirPortal,
  alCambiar = () => {}, anunciar = () => {}, abrirAyuda = () => {},
  puedeEditar = false, generarClave = claveNueva,
} = {}) {
  const estado = { bolsaRef: "", carga: "inactiva", vigente: null, borrador: copia(EJEMPLO_VACIO),
    error: "", mensaje: "", guardando: false, conflicto: false, clave: "", cargaId: 0 };
  let lectura = null;
  let escritura = null;
  let documentoInstalado = null;
  const repintar = () => alCambiar();
  const t = (clave, variables) => escapar(traducir(`rrhh_plazos_${clave}`, variables));

  async function cargar({ conservarBorrador = false } = {}) {
    if (!estado.bolsaRef) return;
    lectura?.abort();
    lectura = new AbortController();
    const controlador = lectura;
    const bolsa = estado.bolsaRef;
    const id = ++estado.cargaId;
    estado.carga = "cargando"; estado.error = ""; repintar();
    try {
      const resultado = await cliente.consultar(bolsa, { signal: controlador.signal });
      if (controlador.signal.aborted || id !== estado.cargaId || bolsa !== estado.bolsaRef) return;
      if (!resultado.ok) {
        estado.carga = resultado.status === 403 ? "denegado" : "error";
        estado.error = traducir(`rrhh_plazos_${resultado.status === 403 ? "denegado" : "error_carga"}`);
      } else {
        estado.vigente = resultado.politica;
        if (!conservarBorrador) {
          estado.borrador = copia(resultado.politica.configurada ? resultado.politica.politica : EJEMPLO_VACIO);
          estado.clave = "";
        }
        estado.carga = "lista"; estado.conflicto = false;
      }
    } catch (error) {
      if (controlador.signal.aborted || id !== estado.cargaId || bolsa !== estado.bolsaRef || error?.name === "AbortError") return;
      estado.carga = "error"; estado.error = traducir("rrhh_plazos_error_carga");
    }
    repintar();
  }

  async function guardar() {
    if (!puedeEditar || estado.guardando || estado.conflicto || estado.carga !== "lista") return;
    let politica;
    try { politica = validarPoliticaEditable(estado.borrador); }
    catch { estado.error = traducir("rrhh_plazos_validacion"); repintar(); return; }
    const bolsa = estado.bolsaRef;
    const id = estado.cargaId;
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
        estado.borrador = copia(resultado.politica.politica);
        estado.clave = "";
        estado.mensaje = traducir("rrhh_plazos_guardada", { version: resultado.politica.version,
          recibo: resultado.politica.recibo_ref });
        anunciar(estado.mensaje);
      } else if (resultado.status === 409) {
        estado.conflicto = true;
        estado.clave = "";
        estado.error = traducir("rrhh_plazos_conflicto");
      } else {
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
    const deshabilitado = !puedeEditar || estado.guardando || estado.conflicto;
    const disabled = deshabilitado ? " disabled" : "";
    const cabecera = `<div class="cabecera-panel"><div><h3>${t("titulo")}</h3><p>${t("subtitulo")}</p></div><div class="rrhh-plazos__cabecera-estado"><span class="estado-chip advertencia">${t("ejemplo")}</span>${configurada ? `<span class="estado-chip info">${t("version", { version: estado.vigente.version })}</span>` : ""}<button type="button" class="rrhh-plazos__ayuda" data-rrhh-plazos-accion="ayuda" aria-label="${t("ayuda")}">?</button></div></div>`;
    if (estado.carga === "inactiva" || estado.carga === "cargando") {
      return `<section class="panel rrhh-plazos" aria-busy="true">${cabecera}<div class="cuerpo-panel" role="status">${t("cargando")}</div></section>`;
    }
    if (estado.carga !== "lista") {
      return `<section class="panel rrhh-plazos">${cabecera}<div class="cuerpo-panel"><p class="rrhh-plazos__error" role="alert">${escapar(estado.error)}</p>${estado.carga === "error" ? `<button type="button" class="boton-secundario" data-rrhh-plazos-accion="recargar">${t("reintentar")}</button>` : ""}</div></section>`;
    }
    const calendario = /^[0-9]{5}$/u.test(v.plazo.municipio_sede)
      ? t("calendario_municipio", { municipio: v.plazo.municipio_sede }) : t("calendario_pendiente");
    const plazo = `<section class="panel"><div class="cabecera-panel"><h3>${t("plazo_titulo")}</h3></div><div class="cuerpo-panel rrhh-plazos__campos"><label class="campo"><span>${t("cantidad")}</span><input name="cantidad" type="number" min="1" max="30" step="1" required value="${escapar(v.plazo.cantidad)}"${disabled}></label><label class="campo"><span>${t("unidad")}</span><select name="unidad"${disabled}><option value="dias_habiles"${v.plazo.unidad === "dias_habiles" ? " selected" : ""}>${t("dias_habiles")}</option><option value="dias_naturales"${v.plazo.unidad === "dias_naturales" ? " selected" : ""}>${t("dias_naturales")}</option></select></label><label class="campo"><span>${t("municipio_sede")}</span><input name="municipio_sede" inputmode="numeric" pattern="[0-9]{5}" minlength="5" maxlength="5" required value="${escapar(v.plazo.municipio_sede)}"${disabled}></label>${dato("computo", t("administrativo"))}${dato("calendario_fuente", calendario)}</div></section>`;
    const orden = `<section class="panel"><div class="cabecera-panel"><h3>${t("orden_titulo")}</h3></div><div class="cuerpo-panel rrhh-plazos__campos">${dato("criterio", t("orden_vigente"))}${dato("elegibilidad", t("disposicion_en_plazo"))}<p class="campo--ancho dato-secundario">${t("confirmacion")}</p></div></section>`;
    const noCubierta = `<section class="panel"><div class="cabecera-panel"><h3>${t("no_cubierta_titulo")}</h3></div><div class="cuerpo-panel rrhh-plazos__campos">${dato("condicion", t("sin_elegibles"))}${dato("accion", t("llamamiento_directo"))}</div></section>`;
    const aviso = `${!configurada ? `<p role="status">${t("vacio")}</p>` : ""}${estado.error ? `<p class="rrhh-plazos__error" role="alert">${escapar(estado.error)}</p>` : ""}${estado.mensaje ? `<p class="rrhh-plazos__resultado" role="status">${escapar(estado.mensaje)}</p>` : ""}`;
    const boton = puedeEditar ? `<div class="rrhh-plazos__acciones">${estado.conflicto ? `<button type="button" class="boton-secundario" data-rrhh-plazos-accion="revisar">${t("revisar")}</button>` : ""}<button type="submit" class="boton-primario"${disabled}>${t(estado.guardando ? "guardando" : estado.clave ? "reintentar_guardado" : "guardar")}</button></div>` : `<p class="dato-secundario">${t("sin_edicion")}</p>`;
    return `<section class="rrhh-plazos" aria-label="${t("titulo")}"><div class="panel">${cabecera}${aviso ? `<div class="cuerpo-panel">${aviso}</div>` : ""}</div><form data-rrhh-plazos-form="politica" novalidate><div class="rrhh-plazos__rejilla">${plazo}${orden}${noCubierta}</div>${boton}</form></section>`;
  }

  function manejarCambio(evento) {
    const control = evento.target;
    if (!control?.closest?.('[data-rrhh-plazos-form="politica"]') || !puedeEditar) return false;
    const campo = { cantidad: "cantidad", unidad: "unidad", municipio_sede: "municipio_sede" }[control.name];
    if (!campo) return false;
    const valor = campo === "cantidad" ? Number(control.value) : campo === "municipio_sede" ? control.value.trim() : control.value;
    if (estado.borrador.plazo[campo] === valor) return true;
    estado.borrador.plazo[campo] = valor;
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
    if (boton.dataset.rrhhPlazosAccion === "ayuda") abrirAyuda();
    else if (boton.dataset.rrhhPlazosAccion === "recargar") void cargar();
    else if (boton.dataset.rrhhPlazosAccion === "revisar") void cargar({ conservarBorrador: true });
    else return false;
    return true;
  }

  return Object.freeze({
    activar(bolsaRef) {
      if (!bolsaRef || bolsaRef === estado.bolsaRef) return;
      lectura?.abort(); escritura?.abort(); estado.cargaId++;
      Object.assign(estado, { bolsaRef, carga: "inactiva", vigente: null, borrador: copia(EJEMPLO_VACIO),
        error: "", mensaje: "", guardando: false, conflicto: false, clave: "" });
      void cargar();
    },
    desmontar() {
      lectura?.abort(); escritura?.abort(); estado.cargaId++; estado.bolsaRef = ""; estado.carga = "inactiva";
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
