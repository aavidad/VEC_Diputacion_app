import { traducirPortal } from "../../portal-i18n.js?v=20260928-rrhh-corte3-i18n-v2";
import { crearClienteCausasParticipacion, validarCausaEditable } from "./rrhh-causas-participacion-api.js";

const escapar = (valor) => String(valor ?? "").replaceAll("&", "&amp;")
  .replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;")
  .replaceAll("'", "&#39;");
const CAMPOS = ["codigo", "version", "etiqueta", "aplica_situacion", "aplica_contacto", "publicable", "activa"];
const borradorVacio = () => ({ codigo: "", version: 1, etiqueta: "", aplica_situacion: false,
  aplica_contacto: false, publicable: false, activa: false });

/** Cada POST vuelve a autorizar. Proponer y publicar exigen identidades distintas en B57. */
export function crearSuperficieRRHHCausasParticipacion({
  cliente = crearClienteCausasParticipacion(), traducir = traducirPortal,
  alCambiar = () => {}, anunciar = () => {},
  confirmarDescarte = (texto) => globalThis.confirm?.(texto) === true,
} = {}) {
  const estado = { carga: "inactiva", modo: "proponer", causas: [], borrador: borradorVacio(),
    propuestaRef: "", propuestaConsultada: null, consultaPropuesta: false, ayudaAbierta: false,
    error: "", errorPropuesta: "", errorPublicacion: "", errorConsulta: "", guardando: "",
    propuestaConfirmada: null, confirmacion: null, pendiente: null, idCarga: 0 };
  let lectura = null;
  let lecturaPropuesta = null;
  let escritura = null;
  let documentoInstalado = null;
  const cambiar = () => alCambiar();
  const mensaje = (clave, variables) => traducir(`rrhh_causas_${clave}`, variables);
  const t = (clave, variables) => escapar(mensaje(clave, variables));
  const enfocar = (selector) => queueMicrotask(() =>
    documentoInstalado?.querySelector?.(selector)?.focus?.({ preventScroll: true }));
  const olvidarDatosProtegidos = () => {
    estado.causas = []; estado.borrador = borradorVacio();
    estado.propuestaRef = ""; estado.propuestaConsultada = null;
    estado.propuestaConfirmada = null; estado.confirmacion = null; estado.pendiente = null;
  };

  async function cargar({ recuperarFoco = false } = {}) {
    lectura?.abort();
    lectura = new AbortController();
    const controlador = lectura;
    const id = ++estado.idCarga;
    estado.carga = "cargando"; estado.error = ""; cambiar();
    if (recuperarFoco) enfocar("[data-rrhh-causas-carga]");
    try {
      const resultado = await cliente.consultar({ signal: controlador.signal });
      if (controlador.signal.aborted || id !== estado.idCarga) return;
      if (!resultado.ok) {
        estado.causas = [];
        estado.carga = resultado.status === 401 || resultado.status === 403 ? "denegado" : "error";
        if (estado.carga === "denegado") olvidarDatosProtegidos();
        estado.error = mensaje(estado.carga === "denegado" ? "denegado"
          : resultado.status === 503 ? "no_disponible" : "error_carga");
      } else { estado.causas = resultado.causas; estado.carga = "lista"; }
    } catch {
      if (controlador.signal.aborted || id !== estado.idCarga) return;
      estado.causas = []; estado.carga = "error"; estado.error = mensaje("error_carga");
    }
    cambiar();
    if (recuperarFoco) enfocar(estado.carga === "lista" ? "[data-rrhh-causas-lista]" : "[data-rrhh-causas-carga-error]");
  }

  async function consultarPropuesta() {
    if (estado.carga !== "lista" || estado.modo !== "publicar" || estado.guardando || estado.pendiente) return;
    lecturaPropuesta?.abort(); lecturaPropuesta = new AbortController();
    const controlador = lecturaPropuesta;
    const ref = estado.propuestaRef.trim();
    const id = estado.idCarga;
    estado.consultaPropuesta = true; estado.errorConsulta = "";
    estado.propuestaConsultada = null; cambiar();
    enfocar('[data-rrhh-causas-accion="consultar-propuesta"]');
    try {
      const resultado = await cliente.consultarPropuesta(ref, { signal: controlador.signal });
      if (controlador.signal.aborted || id !== estado.idCarga || ref !== estado.propuestaRef.trim()) return;
      if (!resultado.ok) {
        estado.errorConsulta = mensaje(resultado.status === 401 || resultado.status === 403
          ? "denegado_propuesta" : resultado.status === 404 ? "propuesta_no_encontrada" : "error_consulta_propuesta");
        if (resultado.status === 401 || resultado.status === 403) olvidarDatosProtegidos();
      } else estado.propuestaConsultada = resultado;
    } catch {
      if (controlador.signal.aborted || id !== estado.idCarga) return;
      estado.errorConsulta = mensaje("error_consulta_propuesta");
    } finally {
      if (id === estado.idCarga) {
        estado.consultaPropuesta = false; cambiar();
        enfocar(estado.propuestaConsultada ? "[data-rrhh-causas-detalle]" : "[data-rrhh-causas-consulta-error]");
      }
    }
  }

  async function enviar(tipo) {
    if (estado.carga !== "lista" || estado.guardando || (estado.pendiente && estado.pendiente.tipo !== tipo)
      || estado.modo !== (tipo === "propuesta" ? "proponer" : "publicar")) return;
    let envio;
    try {
      if (estado.pendiente) envio = estado.pendiente;
      else if (tipo === "propuesta") envio = { tipo, entrada: validarCausaEditable(estado.borrador) };
      else {
        const consultada = estado.propuestaConsultada;
        if (!consultada || consultada.estado !== "pendiente"
          || consultada.propuesta.propuesta_ref !== estado.propuestaRef.trim()) {
          throw new TypeError("propuesta causal no consultada");
        }
        envio = { tipo, propuestaRef: consultada.propuesta.propuesta_ref,
          entrada: validarCausaEditable(consultada.propuesta) };
      }
    } catch {
      estado[tipo === "propuesta" ? "errorPropuesta" : "errorPublicacion"] = mensaje("validacion");
      cambiar(); enfocar(`[data-rrhh-causas-error="${tipo}"]`); return;
    }
    escritura?.abort(); escritura = new AbortController();
    const controlador = escritura;
    const id = estado.idCarga;
    estado.pendiente = envio;
    estado.guardando = tipo;
    estado[tipo === "propuesta" ? "errorPropuesta" : "errorPublicacion"] = "";
    cambiar();
    enfocar("[data-rrhh-causas-guardando]");
    try {
      const resultado = tipo === "propuesta"
        ? await cliente.proponer(envio.entrada, { signal: controlador.signal })
        : await cliente.publicar(envio.propuestaRef, envio.entrada, { signal: controlador.signal });
      if (controlador.signal.aborted || id !== estado.idCarga) return;
      if (!resultado.ok) {
        const clave = resultado.status === 401 || resultado.status === 403 ? "denegado_guardado"
          : resultado.status === 409 ? "conflicto" : resultado.status === 400 ? "validacion"
            : "indeterminado";
        estado[tipo === "propuesta" ? "errorPropuesta" : "errorPublicacion"] = mensaje(clave);
        if (resultado.status === 401 || resultado.status === 403) {
          olvidarDatosProtegidos();
        }
        enfocar(`[data-rrhh-causas-error="${tipo}"]`);
      } else if (tipo === "propuesta") {
        estado.propuestaConfirmada = { ...resultado.propuesta, recibo: resultado.recibo,
          entrada: envio.entrada };
        estado.pendiente = null;
        estado.borrador = borradorVacio();
        anunciar(mensaje("propuesta_confirmada", { version: resultado.propuesta.version }));
        enfocar("[data-rrhh-causas-propuesta]");
      } else {
        estado.confirmacion = { ...resultado.catalogo, recibo: resultado.recibo };
        estado.pendiente = null;
        estado.propuestaRef = ""; estado.propuestaConsultada = null;
        estado.guardando = "";
        anunciar(mensaje("confirmacion", { version: resultado.catalogo.version }));
        void cargar().then(() => {
          if (estado.carga === "error") { estado.error = mensaje("actualizacion_fallida"); cambiar(); }
          enfocar("[data-rrhh-causas-recibo]");
        });
      }
    } catch {
      if (controlador.signal.aborted || id !== estado.idCarga) return;
      estado[tipo === "propuesta" ? "errorPropuesta" : "errorPublicacion"] = mensaje("indeterminado");
      enfocar(`[data-rrhh-causas-error="${tipo}"]`);
    } finally {
      if (id === estado.idCarga) { estado.guardando = ""; cambiar(); }
    }
  }

  function fila(causa, indice) {
    const ambito = causa.aplica_situacion && causa.aplica_contacto ? "ambos"
      : causa.aplica_situacion ? "situacion" : "contacto";
    return `<tr><th scope="row">${escapar(causa.etiqueta)}</th><td>${t(ambito)}</td>
      <td>${t("version", { version: causa.version })}</td><td class="rrhh-causas__accion-fila">
      <button type="button" class="boton-secundario" data-rrhh-causas-versionar="${indice}"
      aria-label="${t("versionar", { etiqueta: causa.etiqueta })}"${estado.guardando || estado.pendiente ? " disabled" : ""}>${t("versionar_corta")}</button></td></tr>`;
  }

  function renderizarPropuesta() {
    const p = estado.propuestaConfirmada;
    if (!p) return "";
    return `<section class="panel rrhh-causas__confirmacion" data-rrhh-causas-propuesta tabindex="-1" role="status">
      <div class="cabecera-panel"><h3>${t("propuesta_confirmada", { version: p.version })}</h3></div>
      <div class="cuerpo-panel"><p>${t("segunda_identidad")}</p><dl>
      <div><dt>${t("propuesta_referencia")}</dt><dd><code>${escapar(p.propuesta_ref)}</code></dd></div>
      <div><dt>${t("recibo")}</dt><dd><code>${escapar(p.recibo)}</code></dd></div>
      <div><dt>${t("huella")}</dt><dd><code>${escapar(p.huella_sha256)}</code></dd></div></dl>
      <p class="dato-secundario">${t("consultar_antes_publicar")}</p></div></section>`;
  }

  function renderizarConfirmacion() {
    const r = estado.confirmacion;
    if (!r) return "";
    return `<section class="panel rrhh-causas__confirmacion" data-rrhh-causas-recibo tabindex="-1" role="status">
      <div class="cabecera-panel"><h3>${t("confirmacion", { version: r.version })}</h3></div>
      <div class="cuerpo-panel"><p>${escapar(r.etiqueta)}</p><dl>
      <div><dt>${t("recibo")}</dt><dd><code>${escapar(r.recibo)}</code></dd></div>
      <div><dt>${t("huella")}</dt><dd><code>${escapar(r.huella_sha256)}</code></dd></div></dl>
      ${!r.publicable || !r.activa ? `<p class="dato-secundario">${t("version_no_publicada")}</p>` : ""}</div></section>`;
  }

  function renderizar() {
    const cabecera = `<header class="cabecera-panel"><div><h3>${t("titulo")}</h3><p>${t("subtitulo")}</p></div>
      <div class="rrhh-causas__navegacion"><button type="button" class="boton-secundario" data-rrhh-causas-accion="modo-proponer"
      aria-pressed="${estado.modo === "proponer"}">${t("vista_proponer")}</button>
      <button type="button" class="boton-secundario" data-rrhh-causas-accion="modo-publicar"
      aria-pressed="${estado.modo === "publicar"}">${t("vista_publicar")}</button>
      <button type="button" class="rrhh-causas__ayuda-boton" data-rrhh-causas-accion="ayuda"
      aria-label="${t("ayuda_boton")}" aria-expanded="${estado.ayudaAbierta}" aria-controls="rrhh-causas-ayuda">?</button></div></header>
      <p id="rrhh-causas-ayuda" class="rrhh-causas__ayuda" ${estado.ayudaAbierta ? "" : "hidden"}>${t("ayuda")}</p>`;
    if (estado.carga === "inactiva" || estado.carga === "cargando") {
      return `<section class="panel rrhh-causas" data-rrhh-causas aria-busy="true">${cabecera}
        ${renderizarPropuesta()}${renderizarConfirmacion()}
        <div class="cuerpo-panel" data-rrhh-causas-carga tabindex="-1" role="status">${t("cargando")}</div></section>`;
    }
    if (estado.carga !== "lista") {
      return `<section class="panel rrhh-causas" data-rrhh-causas>${cabecera}<div class="cuerpo-panel">
        <p class="rrhh-causas__error" data-rrhh-causas-carga-error tabindex="-1" role="alert">${escapar(estado.error)}</p>
        ${estado.carga === "error" ? `<button type="button" class="boton-secundario" data-rrhh-causas-accion="recargar">${t("reintentar")}</button>` : ""}
        ${renderizarPropuesta()}${renderizarConfirmacion()}</div></section>`;
    }
    const b = estado.borrador;
    const bloquearPropuesta = estado.guardando || estado.pendiente ? " disabled" : "";
    const bloquearPublicacion = estado.guardando || estado.pendiente ? " disabled" : "";
    const lista = `<section class="panel" data-rrhh-causas-lista tabindex="-1"><div class="cabecera-panel"><h3>${t("lista")}</h3></div><div class="cuerpo-panel">
      ${estado.causas.length === 0 ? `<p role="status">${t("vacio")}</p>` : `<div class="rrhh-causas__tabla" role="region" tabindex="0" aria-label="${t("lista")}"><table class="tabla-datos"><thead><tr>
        <th scope="col">${t("etiqueta")}</th><th scope="col">${t("ambito")}</th><th scope="col">${t("numero_version")}</th><th scope="col">${t("accion")}</th>
        </tr></thead><tbody>${estado.causas.map(fila).join("")}</tbody></table></div>`}</div></section>`;
    const propuesta = `<section class="panel"><div class="cabecera-panel"><h3>${t("configurar")}</h3>
      <button type="button" class="boton-secundario" data-rrhh-causas-accion="nueva"${bloquearPropuesta}>${t("nueva")}</button></div>
      <div class="cuerpo-panel"><form data-rrhh-causas-form="propuesta" novalidate>
      <div class="rrhh-causas__campos"><label class="campo"><span>${t("codigo")}</span><input name="codigo" maxlength="64" pattern="[a-z][a-z0-9_]{2,63}" required value="${escapar(b.codigo)}"${bloquearPropuesta}></label>
      <label class="campo"><span>${t("numero_version")}</span><input name="version" type="number" min="1" step="1" required value="${escapar(b.version)}"${bloquearPropuesta}></label>
      <label class="campo rrhh-causas__campo-ancho"><span>${t("etiqueta_publica")}</span><input name="etiqueta" maxlength="120" required value="${escapar(b.etiqueta)}"${bloquearPropuesta}></label></div>
      <div class="rrhh-causas__opciones"><label><input name="aplica_situacion" type="checkbox"${b.aplica_situacion ? " checked" : ""}${bloquearPropuesta}>${t("aplica_situacion")}</label>
      <label><input name="aplica_contacto" type="checkbox"${b.aplica_contacto ? " checked" : ""}${bloquearPropuesta}>${t("aplica_contacto")}</label>
      <label><input name="publicable" type="checkbox"${b.publicable ? " checked" : ""}${bloquearPropuesta}>${t("publicable")}</label>
      <label><input name="activa" type="checkbox"${b.activa ? " checked" : ""}${bloquearPropuesta}>${t("activa")}</label></div>
      ${estado.errorPropuesta ? `<p class="rrhh-causas__error" data-rrhh-causas-error="propuesta" tabindex="-1" role="alert">${escapar(estado.errorPropuesta)}</p>` : ""}
      ${estado.guardando === "propuesta" ? `<p data-rrhh-causas-guardando tabindex="-1" role="status">${t("guardando")}</p>` : ""}
      <div class="rrhh-causas__acciones">${estado.pendiente?.tipo === "propuesta" ? `<button type="button" class="boton-secundario" data-rrhh-causas-accion="descartar">${t("descartar_pendiente")}</button>` : ""}
      <button type="submit" class="boton-primario"${estado.guardando ? " disabled" : ""}>${t(estado.guardando === "propuesta" ? "guardando" : estado.pendiente?.tipo === "propuesta" ? "reintentar_mismo" : "proponer")}</button></div>
      </form></div></section>`;
    const consultada = estado.propuestaConsultada;
    const detalle = consultada ? `<div class="rrhh-causas__propuesta-detalle" data-rrhh-causas-detalle tabindex="-1" role="status">
      <p><strong>${escapar(consultada.propuesta.etiqueta)}</strong> · ${t("version", { version: consultada.propuesta.version })}</p>
      <p>${t(consultada.propuesta.aplica_situacion && consultada.propuesta.aplica_contacto ? "ambos"
        : consultada.propuesta.aplica_situacion ? "situacion" : "contacto")}</p>
      <p><span class="estado-chip ${consultada.estado === "pendiente" ? "" : consultada.estado === "publicada" ? "exito" : "neutro"}">${t(consultada.estado === "pendiente" ? "pendiente_revision" : consultada.estado === "publicada" ? "ya_publicada" : "superada")}</span></p>
      <dl><div><dt>${t("huella")}</dt><dd><code>${escapar(consultada.propuesta.huella_sha256)}</code></dd></div>
      <div><dt>${t("recibo")}</dt><dd><code>${escapar(consultada.recibo)}</code></dd></div></dl></div>` : "";
    const publicacion = `<section class="panel rrhh-causas__publicacion"><div class="cabecera-panel"><h3>${t("publicar_titulo")}</h3></div>
      <div class="cuerpo-panel"><p class="dato-secundario">${t("segunda_identidad")}</p>
      <div class="rrhh-causas__consulta"><label class="campo"><span>${t("propuesta_referencia")}</span>
      <input name="propuesta_ref" data-rrhh-causas-ref required value="${escapar(estado.propuestaRef)}"${bloquearPublicacion}></label>
      <button type="button" class="boton-secundario" data-rrhh-causas-accion="consultar-propuesta"${bloquearPublicacion}>${t(estado.consultaPropuesta ? "consultando_propuesta" : "consultar_propuesta")}</button></div>
      ${estado.errorConsulta ? `<p class="rrhh-causas__error" data-rrhh-causas-consulta-error tabindex="-1" role="alert">${escapar(estado.errorConsulta)}</p>` : ""}
      ${detalle}<form data-rrhh-causas-form="publicacion" novalidate>
      ${estado.errorPublicacion ? `<p class="rrhh-causas__error" data-rrhh-causas-error="publicacion" tabindex="-1" role="alert">${escapar(estado.errorPublicacion)}</p>` : ""}
      ${estado.guardando === "publicacion" ? `<p data-rrhh-causas-guardando tabindex="-1" role="status">${t("guardando")}</p>` : ""}
      <div class="rrhh-causas__acciones">${estado.pendiente?.tipo === "publicacion" ? `<button type="button" class="boton-secundario" data-rrhh-causas-accion="descartar">${t("descartar_pendiente")}</button>` : ""}
      <button type="submit" class="boton-primario"${estado.guardando || (!estado.pendiente && consultada?.estado !== "pendiente") ? " disabled" : ""}>${t(estado.guardando === "publicacion" ? "guardando" : estado.pendiente?.tipo === "publicacion" ? "reintentar_mismo" : "publicar")}</button></div></form></div></section>`;
    return `<section class="rrhh-causas" data-rrhh-causas>${cabecera}${renderizarPropuesta()}${renderizarConfirmacion()}
      <div class="rrhh-causas__rejilla">${lista}${estado.modo === "proponer" ? propuesta : publicacion}</div></section>`;
  }

  function manejarCambio(evento) {
    const control = evento.target;
    if (control?.matches?.("[data-rrhh-causas-ref]")) {
      if (estado.guardando || estado.pendiente) return false;
      estado.propuestaRef = control.value; estado.propuestaConsultada = null;
      estado.errorConsulta = ""; estado.errorPublicacion = "";
      const raiz = control.closest?.("[data-rrhh-causas]");
      raiz?.querySelector?.(".rrhh-causas__propuesta-detalle")?.setAttribute?.("hidden", "");
      raiz?.querySelector?.('[data-rrhh-causas-form="publicacion"] button[type="submit"]')?.setAttribute?.("disabled", "");
      return true;
    }
    const formulario = control?.closest?.("[data-rrhh-causas-form]")?.dataset?.rrhhCausasForm;
    if (!formulario || estado.carga !== "lista" || estado.guardando || estado.pendiente) return false;
    if (formulario !== "propuesta" || !CAMPOS.includes(control.name)) return false;
    estado.borrador[control.name] = control.name === "version" ? Number(control.value)
      : control.type === "checkbox" ? control.checked : control.value;
    estado.errorPropuesta = ""; return true;
  }

  function manejarSubmit(evento) {
    const tipo = evento.target?.closest?.("[data-rrhh-causas-form]")?.dataset?.rrhhCausasForm;
    if (tipo !== "propuesta" && tipo !== "publicacion") return false;
    evento.preventDefault();
    if (!estado.pendiente && !evento.target.reportValidity?.()) {
      const nombre = evento.target.querySelector?.(":invalid")?.name;
      estado[tipo === "propuesta" ? "errorPropuesta" : "errorPublicacion"] = mensaje("validacion");
      cambiar();
      enfocar(CAMPOS.includes(nombre) ? `[data-rrhh-causas-form="${tipo}"] [name="${nombre}"]`
        : `[data-rrhh-causas-error="${tipo}"]`);
      return true;
    }
    void enviar(tipo); return true;
  }

  function manejarClick(evento) {
    const boton = evento.target?.closest?.("[data-rrhh-causas-accion],[data-rrhh-causas-versionar]");
    if (!boton || boton.disabled || !boton.closest?.("[data-rrhh-causas]")) return false;
    if (boton.dataset.rrhhCausasAccion === "ayuda") {
      estado.ayudaAbierta = !estado.ayudaAbierta; cambiar();
      enfocar('[data-rrhh-causas-accion="ayuda"]');
    } else if (boton.dataset.rrhhCausasAccion === "modo-proponer" && !estado.pendiente) {
      estado.modo = "proponer"; cambiar(); enfocar('[data-rrhh-causas-accion="modo-proponer"]');
    } else if (boton.dataset.rrhhCausasAccion === "modo-publicar" && !estado.pendiente) {
      estado.modo = "publicar"; cambiar(); enfocar('[data-rrhh-causas-accion="modo-publicar"]');
    } else if (boton.dataset.rrhhCausasAccion === "recargar") void cargar({ recuperarFoco: true });
    else if (boton.dataset.rrhhCausasAccion === "consultar-propuesta") void consultarPropuesta();
    else if (boton.dataset.rrhhCausasAccion === "descartar" && estado.pendiente) {
      if (!confirmarDescarte(mensaje("descartar_aviso"))) return true;
      const tipo = estado.pendiente.tipo;
      estado.pendiente = null;
      estado[tipo === "propuesta" ? "errorPropuesta" : "errorPublicacion"] = mensaje("descartado_sin_confirmar");
      cambiar();
      enfocar(`[data-rrhh-causas-error="${tipo}"]`);
    } else if (boton.dataset.rrhhCausasAccion === "nueva" && !estado.pendiente) {
      estado.borrador = borradorVacio(); estado.errorPropuesta = ""; cambiar();
      enfocar('[data-rrhh-causas-form="propuesta"] [name="codigo"]');
    } else if (boton.dataset.rrhhCausasVersionar !== undefined && !estado.pendiente) {
      const indice = Number(boton.dataset.rrhhCausasVersionar);
      const causa = estado.causas[indice];
      if (!causa || !Number.isSafeInteger(indice) || causa.version >= Number.MAX_SAFE_INTEGER) return false;
      estado.borrador = { codigo: causa.codigo, version: causa.version + 1, etiqueta: causa.etiqueta,
        aplica_situacion: causa.aplica_situacion, aplica_contacto: causa.aplica_contacto,
        publicable: true, activa: true };
      estado.errorPropuesta = ""; estado.modo = "proponer"; cambiar();
      enfocar('[data-rrhh-causas-form="propuesta"] [name="etiqueta"]');
    } else return false;
    return true;
  }

  return Object.freeze({
    activar() { if (estado.carga === "inactiva") void cargar(); },
    cargar, consultarPropuesta, renderizar, manejarCambio, manejarSubmit, manejarClick,
    instalar(documento) {
      if (documentoInstalado === documento) return;
      if (documentoInstalado || !documento?.addEventListener) throw new TypeError("superficie causal ya instalada");
      documentoInstalado = documento;
      for (const [tipo, funcion] of [["input", manejarCambio], ["change", manejarCambio],
        ["submit", manejarSubmit], ["click", manejarClick]]) documento.addEventListener(tipo, funcion);
    },
    desmontar() {
      lectura?.abort(); lecturaPropuesta?.abort(); escritura?.abort(); ++estado.idCarga;
      estado.carga = "inactiva"; olvidarDatosProtegidos();
      estado.error = ""; estado.errorPropuesta = ""; estado.errorPublicacion = "";
      estado.errorConsulta = ""; estado.guardando = ""; estado.consultaPropuesta = false;
      for (const [tipo, funcion] of [["input", manejarCambio], ["change", manejarCambio],
        ["submit", manejarSubmit], ["click", manejarClick]]) documentoInstalado?.removeEventListener(tipo, funcion);
      documentoInstalado = null;
    },
    estado: () => structuredClone(estado),
  });
}
