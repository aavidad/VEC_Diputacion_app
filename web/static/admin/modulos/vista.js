const PATRON_ID = /^vec\.module\.[a-z0-9]+(?:[._-][a-z0-9]+)*$/u;
const PATRON_CLAVE = /^[a-z][a-z0-9_.-]{0,255}$/u;
let montaje = 0;

function incompatible() {
  return Object.assign(new TypeError("respuesta_incompatible"), { codigo: "respuesta_incompatible" });
}

/** Proyección cerrada: una respuesta que anuncie escritura no abre este consumidor. */
export function validarConsulta(datos) {
  if (!datos || !Number.isSafeInteger(datos.version) || datos.version < 1
    || typeof datos.huella_sha256 !== "string" || !/^[a-f0-9]{64}$/u.test(datos.huella_sha256)
    || datos.escritura_disponible !== false || !Array.isArray(datos.modulos) || datos.modulos.length > 128) {
    throw incompatible();
  }
  const ids = new Set();
  const modulos = datos.modulos.map((m) => {
    if (!m || typeof m.modulo_id !== "string" || m.modulo_id.length > 128
      || !PATRON_ID.test(m.modulo_id) || typeof m.nombre_key !== "string" || !PATRON_CLAVE.test(m.nombre_key)
      || typeof m.habilitado !== "boolean" || typeof m.gobernado !== "boolean" || ids.has(m.modulo_id)) {
      throw incompatible();
    }
    if (m.modulo_id === "vec.module.administracion" && m.gobernado) throw incompatible();
    ids.add(m.modulo_id);
    return Object.freeze({ modulo_id: m.modulo_id, nombre_key: m.nombre_key,
      habilitado: m.habilitado, gobernado: m.gobernado });
  });
  return Object.freeze({ version: datos.version, huella_sha256: datos.huella_sha256,
    escritura_disponible: false, modulos: Object.freeze(modulos) });
}

/** Datos de revisión: no son una autorización, un recibo ni un efecto persistido. */
export function prepararSolicitud(modelo, moduloID, habilitado, motivo) {
  const fuente = validarConsulta(modelo);
  const modulo = fuente.modulos.find((m) => m.modulo_id === moduloID);
  if (!modulo?.gobernado || moduloID === "vec.module.administracion"
    || typeof habilitado !== "boolean" || habilitado === modulo.habilitado
    || typeof motivo !== "string" || !motivo.trim() || motivo.trim().length > 500) {
    throw Object.assign(new TypeError("preparacion_invalida"), { codigo: "preparacion_invalida" });
  }
  return Object.freeze({ modulo_id: modulo.modulo_id, habilitado,
    version_esperada: fuente.version, huella_esperada: fuente.huella_sha256, motivo: motivo.trim() });
}

function esPreparacion(respuesta, propuesta) {
  if (!respuesta || respuesta.publicado !== false
    || respuesta.version_siguiente !== propuesta.version_esperada + 1
    || typeof respuesta.cambio !== "object" || respuesta.cambio === null) return false;
  const claves = Object.keys(propuesta);
  return Object.keys(respuesta.cambio).length === claves.length
    && claves.every((clave) => Object.hasOwn(respuesta.cambio, clave) && respuesta.cambio[clave] === propuesta[clave]);
}

function escapar(valor) {
  return String(valor).replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}

/** Puertos de consulta y preparación inyectados. No contiene cliente de escritura. */
export function montarModulos(root, { t, consultar, preparar } = {}) {
  if (!root || typeof t !== "function") throw new TypeError("montaje_invalido");
  const prefijo = `admin-modulos-${++montaje}`;
  const id = (clave) => `${prefijo}-${clave}`;
  const tx = (clave) => escapar(t(clave));
  let vivo = true;
  let secuencia = 0;
  let controlador = null;
  let modelo = null;
  let propuesta = null;
  let preparando = false;
  const listeners = [];
  root.innerHTML = `
    <div class="rejilla-principal">
      <section class="panel" aria-labelledby="${id("titulo")}">
        <div class="cabecera-panel"><div><h2 id="${id("titulo")}">${tx("consulta.titulo")}</h2>
          <p>${tx("consulta.descripcion")}</p></div><span class="estado-chip">${tx("general.solo_preparacion")}</span></div>
        <div class="cuerpo-panel"><p class="nota-pendiente" id="${id("limite")}">${tx("general.limite")}</p>
          <p id="${id("estado")}" role="status" aria-live="polite"></p>
          <button type="button" class="boton-secundario" id="${id("recargar")}">${tx("consulta.recargar")}</button></div>
        <div class="tabla-contenedor" id="${id("tabla")}" tabindex="0" aria-label="${tx("consulta.tabla")}" hidden>
          <table class="tabla-datos"><caption>${tx("consulta.tabla")}</caption><thead><tr>
            <th scope="col">${tx("consulta.modulo")}</th><th scope="col">${tx("consulta.estado")}</th>
            <th scope="col">${tx("consulta.gobierno")}</th></tr></thead><tbody id="${id("filas")}"></tbody></table></div>
      </section>
      <section class="panel" id="${id("editor")}" aria-labelledby="${id("editor-titulo")}" hidden>
        <div class="cabecera-panel"><h2 id="${id("editor-titulo")}">${tx("preparacion.titulo")}</h2></div>
        <div class="cuerpo-panel"><form id="${id("formulario")}" class="formulario-llamamiento" novalidate>
          <label class="campo campo-ancho" for="${id("modulo")}"><span>${tx("preparacion.modulo")}</span>
            <select id="${id("modulo")}" required aria-describedby="${id("error-modulo")}"></select>
            <span id="${id("error-modulo")}" hidden></span></label>
          <label class="campo campo-ancho" for="${id("habilitado")}"><span>${tx("preparacion.estado")}</span>
            <select id="${id("habilitado")}" required aria-describedby="${id("error-estado")}">
              <option value="">${tx("preparacion.elegir_estado")}</option>
              <option value="true">${tx("estados.habilitado")}</option><option value="false">${tx("estados.deshabilitado")}</option></select>
            <span id="${id("error-estado")}" hidden></span></label>
          <label class="campo campo-ancho" for="${id("motivo")}"><span>${tx("preparacion.motivo")}</span>
            <textarea id="${id("motivo")}" maxlength="500" required aria-describedby="${id("pista-motivo")} ${id("error-motivo")}"></textarea>
            <span id="${id("pista-motivo")}" class="texto-secundario">${tx("preparacion.pista_motivo")}</span>
            <span id="${id("error-motivo")}" hidden></span></label>
          <div id="${id("errores")}" class="resumen-errores campo-ancho" role="alert" tabindex="-1" hidden></div>
          <div class="acciones-paso campo-ancho"><button class="boton-primario" type="submit" id="${id("revisar")}">${tx("preparacion.revisar")}</button></div>
        </form></div>
      </section>
      <section class="panel" id="${id("revision")}" aria-labelledby="${id("revision-titulo")}" tabindex="-1" hidden>
        <div class="cabecera-panel"><h2 id="${id("revision-titulo")}">${tx("revision.titulo")}</h2></div>
        <div class="cuerpo-panel"><dl class="resumen-expediente" id="${id("resumen")}"></dl>
          <p>${tx("revision.limite")}</p><p id="${id("resultado")}" role="status" aria-live="polite"></p>
          <div class="acciones-paso"><button type="button" class="boton-secundario" id="${id("corregir")}">${tx("revision.corregir")}</button>
            <button type="button" class="boton-primario" id="${id("preparar")}" aria-describedby="${id("dependencia")}">${tx("revision.preparar")}</button></div>
          <p class="texto-secundario" id="${id("dependencia")}"></p></div>
      </section>
    </div>`;
  const el = (clave) => root.querySelector(`#${id(clave)}`);
  function escuchar(clave, evento, funcion) {
    const nodo = el(clave);
    nodo.addEventListener(evento, funcion);
    listeners.push(() => nodo.removeEventListener(evento, funcion));
  }
  function nombre(m) {
    try {
      const traducido = t(m.nombre_key);
      if (typeof traducido === "string" && traducido.trim() && traducido !== m.nombre_key) return traducido;
    } catch { /* Un nombre sin traducción nunca muestra la clave del catálogo. */ }
    return t("consulta.nombre_pendiente");
  }
  function borrarRevision() {
    propuesta = null;
    el("revision").hidden = true;
    el("editor").hidden = !modelo?.modulos.some((m) => m.gobernado);
    el("resultado").textContent = "";
  }
  function borrarErrores() {
    el("errores").hidden = true;
    for (const [campo, error] of [["modulo", "error-modulo"], ["habilitado", "error-estado"], ["motivo", "error-motivo"]]) {
      el(campo).removeAttribute("aria-invalid");
      el(error).hidden = true;
    }
  }
  function erroresFormulario() {
    const modulo = modelo?.modulos.find((m) => m.modulo_id === el("modulo").value);
    const errores = [];
    if (!modulo?.gobernado) errores.push(["modulo", "error-modulo", "errores.modulo"]);
    if (!["true", "false"].includes(el("habilitado").value)) errores.push(["habilitado", "error-estado", "errores.estado"]);
    else if (modulo && modulo.habilitado === (el("habilitado").value === "true")) errores.push(["habilitado", "error-estado", "errores.sin_cambio"]);
    if (!el("motivo").value.trim() || el("motivo").value.trim().length > 500) errores.push(["motivo", "error-motivo", "errores.motivo"]);
    return errores;
  }
  function validarFormulario(enviar = false) {
    borrarErrores();
    const errores = erroresFormulario();
    for (const [campo, error, clave] of errores) {
      el(campo).setAttribute("aria-invalid", "true");
      el(error).hidden = false;
      el(error).textContent = t(clave);
    }
    if (enviar && errores.length) {
      el("errores").innerHTML = `<h3>${tx("errores.titulo")}</h3><ul>${errores.map(([campo, , clave]) =>
        `<li><a href="#${id(campo)}">${tx(clave)}</a></li>`).join("")}</ul>`;
      el("errores").hidden = false;
      el("errores").focus();
    }
    return errores.length === 0;
  }
  function filaResumen(clave, valor) {
    return `<div class="fila-resumen"><dt>${tx(clave)}</dt><dd>${escapar(valor)}</dd></div>`;
  }
  escuchar("formulario", "submit", (evento) => {
    evento.preventDefault();
    if (!vivo || preparando || !modelo || !validarFormulario(true)) return;
    propuesta = prepararSolicitud(modelo, el("modulo").value, el("habilitado").value === "true", el("motivo").value);
    const modulo = modelo.modulos.find((m) => m.modulo_id === propuesta.modulo_id);
    el("resumen").innerHTML = filaResumen("revision.modulo", nombre(modulo))
      + filaResumen("revision.actual", t(modulo.habilitado ? "estados.habilitado" : "estados.deshabilitado"))
      + filaResumen("revision.propuesto", t(propuesta.habilitado ? "estados.habilitado" : "estados.deshabilitado"))
      + filaResumen("revision.motivo", propuesta.motivo);
    el("resultado").textContent = "";
    el("preparar").disabled = typeof preparar !== "function";
    el("dependencia").textContent = t(typeof preparar === "function" ? "revision.dependencia" : "revision.sin_preparador");
    el("editor").hidden = true;
    el("revision").hidden = false;
    el("revision").focus();
  });
  for (const campo of ["modulo", "habilitado", "motivo"]) {
    escuchar(campo, "input", borrarRevision);
    escuchar(campo, "change", borrarRevision);
    escuchar(campo, "blur", () => { if (modelo && !preparando) validarFormulario(); });
  }
  escuchar("corregir", "click", () => { borrarRevision(); el("modulo").focus(); });
  escuchar("preparar", "click", async () => {
    if (!vivo || preparando || !propuesta || typeof preparar !== "function") return;
    preparando = true;
    const actual = secuencia;
    const copia = propuesta;
    el("preparar").disabled = true;
    el("corregir").disabled = true;
    el("recargar").disabled = true;
    for (const campo of ["modulo", "habilitado", "motivo", "revisar"]) el(campo).disabled = true;
    el("resultado").textContent = t("revision.preparando");
    try {
      const respuesta = await preparar(copia, { signal: controlador.signal });
      if (!vivo || actual !== secuencia) return;
      if (!esPreparacion(respuesta, copia)) throw incompatible();
      el("resultado").textContent = t("revision.preparada");
      propuesta = null;
    } catch (error) {
      if (!vivo || actual !== secuencia) return;
      el("resultado").textContent = t(error?.codigo === "conflicto_version" ? "errores.conflicto" : "errores.preparacion");
      if (error?.codigo === "conflicto_version") propuesta = null;
    } finally {
      if (vivo && actual === secuencia) {
        preparando = false;
        el("preparar").disabled = propuesta === null;
        el("corregir").disabled = false;
        el("recargar").disabled = false;
        for (const campo of ["modulo", "habilitado", "motivo", "revisar"]) el(campo).disabled = false;
      }
    }
  });
  async function cargar() {
    if (!vivo || preparando) return;
    controlador?.abort();
    controlador = new AbortController();
    const actual = ++secuencia;
    modelo = null;
    borrarRevision();
    borrarErrores();
    el("editor").hidden = true;
    el("tabla").hidden = true;
    el("filas").replaceChildren();
    el("recargar").disabled = true;
    if (typeof consultar !== "function") {
      el("estado").textContent = t("consulta.pendiente");
      return;
    }
    el("estado").textContent = t("consulta.cargando");
    try {
      const respuesta = validarConsulta(await consultar({ signal: controlador.signal }));
      if (!vivo || actual !== secuencia) return;
      // Valida las traducciones antes de ofrecer cualquier selección.
      respuesta.modulos.forEach(nombre);
      modelo = respuesta;
      el("filas").innerHTML = modelo.modulos.map((m) => `<tr><th scope="row">${escapar(nombre(m))}</th>
        <td><span class="estado-chip ${m.habilitado ? "exito" : "neutro"}">${tx(m.habilitado ? "estados.habilitado" : "estados.deshabilitado")}</span></td>
        <td>${tx(m.gobernado ? "consulta.gobernado" : "consulta.no_gobernado")}</td></tr>`).join("");
      el("tabla").hidden = modelo.modulos.length === 0;
      el("modulo").innerHTML = `<option value="">${tx("preparacion.elegir_modulo")}</option>`
        + modelo.modulos.filter((m) => m.gobernado).map((m) => `<option value="${escapar(m.modulo_id)}">${escapar(nombre(m))}</option>`).join("");
      el("habilitado").value = "";
      el("motivo").value = "";
      el("editor").hidden = !modelo.modulos.some((m) => m.gobernado);
      el("estado").textContent = t(!modelo.modulos.length ? "consulta.vacia"
        : el("editor").hidden ? "consulta.sin_gobernados" : "consulta.disponible");
    } catch (error) {
      if (!vivo || actual !== secuencia) return;
      modelo = null;
      el("estado").textContent = t(error?.estado === 401 || error?.estado === 403 ? "consulta.denegada"
        : error?.codigo === "respuesta_incompatible" ? "consulta.incompatible" : "consulta.error");
    } finally {
      if (vivo && actual === secuencia) el("recargar").disabled = false;
    }
  }
  escuchar("recargar", "click", () => { void cargar(); });
  const listo = cargar();
  return Object.freeze({ listo, cargar, desmontar() {
    if (!vivo) return;
    vivo = false;
    ++secuencia;
    controlador?.abort();
    listeners.forEach((quitar) => quitar());
    root.replaceChildren();
  } });
}
