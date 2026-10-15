import { parValido, validarPublicacionTransiciones } from "./reglas-situacion-api.js?v=20261015-s6-reglas-situacion-v1";

const esc = (valor) => String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
  .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
const ORDEN = ["disponible", "no_disponible", "trabajando", "pendiente_incorporacion", "renuncia", "excluido", "disponible_desde", "en_revision"];
const origen = (par) => par.split(">")[0];
const destino = (par) => par.split(">")[1];
const ordenarPares = (a, b) => ORDEN.indexOf(origen(a)) - ORDEN.indexOf(origen(b))
  || ORDEN.indexOf(destino(a)) - ORDEN.indexOf(destino(b));

export function montarReglasSituacionBolsa({ raiz, cliente, textos, anunciar = () => {}, alDenegacion = () => {} } = {}) {
  if (!raiz?.addEventListener || !raiz?.removeEventListener || !raiz?.replaceChildren
    || typeof cliente?.consultar !== "function" || typeof cliente?.publicar !== "function"
    || typeof textos?.traducir !== "function") throw new TypeError("pantalla no disponible");
  const t = (clave, variables) => esc(textos.traducir(`reglas.${clave}`, variables));
  const situacion = (clave) => t(`situacion_${clave}`);
  let politica = null, seleccion = new Set(), elegida = "", motivo = "", fase = "cargando";
  let ayuda = false, error = "", recibo = null, pendiente = null, controlador = null, secuencia = 0, montada = true;
  const permitidas = () => politica?.limites.transiciones ?? [];
  const obligatorias = () => politica?.transiciones_obligatorias ?? [];
  const cambios = () => politica ? [...new Set([...permitidas(), ...politica.transiciones_guardadas])]
    .filter((par) => seleccion.has(par) !== politica.transiciones_guardadas.includes(par)) : [];
  const habilitada = () => politica?.puede_publicar && !pendiente && fase === "edicion"
    && (cambios().length > 0 || !politica.catalogo_activado) && motivo.trim().length > 0;
  function estadoError() {
    return error ? `<p role="alert">${t(error)}</p>` : "";
  }
  function pintar() {
    if (!montada) return;
    const cabecera = `<header class="cabecera-panel"><div><h3>${t("titulo")}</h3><p>${t("alcance")}</p>
      ${politica ? `<p>${t("version", { version: textos.numero(politica.version) })} · ${t(politica.catalogo_activado ? "limites_publicados" : "limites_pendientes")}</p>` : ""}</div>
      <button type="button" class="boton-secundario" data-reglas-accion="ayuda" aria-label="${t("ayuda_boton")}" aria-expanded="${ayuda}" aria-controls="reglas-situacion-ayuda">?</button></header>
      <div id="reglas-situacion-ayuda" class="cuerpo-panel" ${ayuda ? "" : "hidden"}><p>${t("ayuda")}</p></div>`;
    if (!politica) {
      raiz.innerHTML = `<section class="panel" data-reglas-situacion>${cabecera}<div class="cuerpo-panel" ${fase === "cargando" ? 'role="status" aria-busy="true"' : 'role="alert"'}>
        <p>${t(fase === "cargando" ? "cargando" : fase === "denegado" ? "denegado" : "carga_error")}</p>
        ${fase === "error" ? `<button type="button" class="boton-secundario" data-reglas-accion="recargar">${t("recargar")}</button>` : ""}</div></section>`;
      return;
    }
    const visibles = permitidas().filter((par) => origen(par) === elegida);
    const opciones = ORDEN.filter((clave) => permitidas().some((par) => origen(par) === clave))
      .map((clave) => `<option value="${clave}" ${elegida === clave ? "selected" : ""}>${situacion(clave)}</option>`).join("");
    const casillas = visibles.map((par) => {
      const fija = obligatorias().includes(par);
      return `<label class="campo opcion-casilla"><input type="checkbox" data-reglas-par="${esc(par)}" ${seleccion.has(par) ? "checked" : ""}
        ${fija || !politica.puede_publicar || fase !== "edicion" ? "disabled" : ""}><span>${situacion(destino(par))}${fija ? ` · ${t("obligatoria")}` : ""}
        ${politica.transiciones_guardadas.includes(par) && !politica.transiciones_efectivas.includes(par) ? ` · ${t("no_vigente")}` : ""}</span></label>`;
    }).join("");
    const diferencias = cambios().map((par) => `<li>${situacion(origen(par))} → ${situacion(destino(par))}: ${t(seleccion.has(par) ? "activar"
      : !permitidas().includes(par) ? "retirar_por_limite" : "restringir")}</li>`).join("");
    let cuerpo = "";
    if (recibo) cuerpo = `<div class="cuerpo-panel" role="status"><h4>${t("publicada")}</h4><p>${t(recibo.reutilizada ? "ya_publicada" : "confirmada")}</p>
      <p>${t("version_publicada", { version: textos.numero(recibo.version) })}</p>
      <details><summary>${t("justificante")}</summary><p>${t("recibo")}: <code>${esc(recibo.recibo_ref)}</code></p></details>
      <button type="button" class="boton-secundario" data-reglas-accion="continuar">${t("continuar")}</button></div>`;
    else if (fase === "incierto") cuerpo = `<div class="cuerpo-panel" role="alert"><p>${t("incierto")}</p>${estadoError()}
      <button type="button" class="boton-primario" data-reglas-accion="repetir">${t("repetir")}</button></div>`;
    else if (fase === "revision" || fase === "enviando") cuerpo = `<div class="cuerpo-panel"><h4>${t("revision")}</h4>
      ${diferencias ? `<ul>${diferencias}</ul>` : `<p>${t("sin_diferencias_publicacion")}</p>`}<p>${t("motivo")}: ${esc(motivo)}</p>${estadoError()}
      <div class="acciones-vista"><button type="button" class="boton-secundario" data-reglas-accion="corregir" ${fase === "enviando" ? "disabled" : ""}>${t("corregir")}</button>
      <button type="button" class="boton-primario" data-reglas-accion="confirmar" ${fase === "enviando" ? "disabled" : ""}>${t(fase === "enviando" ? "enviando" : "confirmar")}</button></div></div>`;
    else cuerpo = `<div class="cuerpo-panel cuerpo-formulario-compacto"><label class="campo"><span>${t("origen")}</span><select data-reglas-origen>${opciones}</select></label>
      <fieldset class="lista-opciones-compacta"><legend>${t("destinos")}</legend>${casillas || `<p>${t("sin_destinos")}</p>`}</fieldset>
      <label class="campo"><span>${t("motivo")}</span><textarea data-reglas-motivo maxlength="1000" rows="2" ${politica.puede_publicar ? "" : "disabled"}>${esc(motivo)}</textarea></label>
      ${estadoError()}${["conflicto", "invalida", "denegado"].includes(error) ? `<button type="button" class="boton-secundario" data-reglas-accion="recargar">${t("recargar")}</button>` : ""}
      ${politica.puede_publicar ? `<div class="barra-acciones-formulario"><button type="button" class="boton-primario" data-reglas-accion="revisar" ${habilitada() && !error ? "" : "disabled"}>${t("revisar")}</button></div>` : `<p role="status">${t("solo_lectura")}</p>`}</div>`;
    raiz.innerHTML = `<section class="panel" data-reglas-situacion>${cabecera}${cuerpo}</section>`;
  }
  function aplicar(p) {
    politica = p;
    seleccion = new Set([...p.transiciones_guardadas.filter((par) => p.limites.transiciones.includes(par)), ...p.transiciones_obligatorias]);
    elegida = ORDEN.find((clave) => permitidas().some((par) => origen(par) === clave)) ?? "";
    motivo = ""; error = ""; recibo = null; pendiente = null; fase = "edicion"; pintar();
  }
  async function cargar() {
    controlador?.abort();
    const turno = ++secuencia; controlador = new AbortController(); const signal = controlador.signal;
    politica = null; fase = "cargando"; error = ""; pintar();
    try {
      const p = await cliente.consultar({ signal });
      if (!montada || signal.aborted || turno !== secuencia) return;
      aplicar(p);
    } catch (e) {
      if (!montada || signal.aborted || turno !== secuencia) return;
      fase = [401, 403].includes(e?.estado) ? "denegado" : "error"; pintar();
      if (fase === "denegado") alDenegacion();
    }
  }
  async function publicar() {
    if (!pendiente || fase !== "revision" && fase !== "incierto") return;
    const recuperacionIncierta = fase === "incierto";
    const turno = ++secuencia; controlador = new AbortController(); const signal = controlador.signal;
    fase = "enviando"; error = ""; pintar();
    try {
      const p = await cliente.publicar(pendiente, { signal });
      if (!montada || signal.aborted || turno !== secuencia) return;
      if (!p.recibo_ref || !p.transiciones_efectivas.every(parValido)) throw new TypeError("recibo incompatible");
      recibo = p; politica = p; pendiente = null; fase = "recibo"; pintar();
      anunciar(textos.traducir("reglas.publicada"));
    } catch (e) {
      if (!montada || signal.aborted || turno !== secuencia) return;
      if (recuperacionIncierta) { fase = "incierto"; error = [401, 403].includes(e?.estado) ? "denegado" : ""; }
      else if (e?.estado === 409 || e?.estado === 422) { pendiente = null; fase = "edicion"; error = e.estado === 409 ? "conflicto" : "invalida"; }
      else if ([401, 403].includes(e?.estado)) { pendiente = null; fase = "denegado"; error = "denegado"; alDenegacion(); }
      else { fase = "incierto"; }
      pintar();
    }
  }
  const pulsar = (evento) => {
    const accion = evento.target?.closest?.("[data-reglas-accion]")?.dataset?.reglasAccion;
    if (!accion) return;
    if (accion === "ayuda") { ayuda = !ayuda; pintar(); raiz.querySelector?.('[data-reglas-accion="ayuda"]')?.focus?.({ preventScroll: true }); }
    else if (accion === "recargar" || accion === "continuar") void cargar();
    else if (accion === "corregir") { pendiente = null; fase = "edicion"; pintar(); }
    else if (accion === "revisar" && habilitada()) {
      pendiente = validarPublicacionTransiciones({ version_esperada: politica.version,
        transiciones: permitidas().filter((par) => seleccion.has(par)).sort(ordenarPares),
        motivo: motivo.trim(), clave_idempotencia: crypto.randomUUID() });
      fase = "revision"; pintar();
    } else if (accion === "confirmar" || accion === "repetir") void publicar();
  };
  const cambiar = (evento) => {
    if (fase !== "edicion" || !politica) return;
    if (evento.target?.matches?.("[data-reglas-origen]")) {
      elegida = evento.target.value; pintar(); raiz.querySelector?.("[data-reglas-origen]")?.focus?.({ preventScroll: true }); return;
    }
    const par = evento.target?.dataset?.reglasPar;
    if (!par || !permitidas().includes(par) || obligatorias().includes(par) || !politica.puede_publicar) return;
    if (evento.target.checked) seleccion.add(par); else seleccion.delete(par);
    pintar();
    [...raiz.querySelectorAll?.("[data-reglas-par]") ?? []].find((control) => control.dataset.reglasPar === par)?.focus?.({ preventScroll: true });
  };
  const escribir = (evento) => {
    if (fase !== "edicion" || !evento.target?.matches?.("[data-reglas-motivo]")) return;
    motivo = evento.target.value;
    const boton = raiz.querySelector?.('[data-reglas-accion="revisar"]');
    if (boton) boton.disabled = !habilitada() || Boolean(error);
  };
  raiz.addEventListener("click", pulsar); raiz.addEventListener("change", cambiar); raiz.addEventListener("input", escribir);
  void cargar();
  return Object.freeze({ desmontar() { montada = false; ++secuencia; controlador?.abort();
    raiz.removeEventListener("click", pulsar); raiz.removeEventListener("change", cambiar); raiz.removeEventListener("input", escribir); raiz.replaceChildren(); } });
}
