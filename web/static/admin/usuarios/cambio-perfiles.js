// Cambio de perfiles de una persona: elegir (asignar o retirar), rellenar,
// revisar y confirmar. Cada cambio es un lote de un solo cambio. Los textos
// vienen del catálogo «admin-usuarios» (claves lote.*).
import { escapar } from "./render.js?v=20261005-admin-lote-pantalla-v1";
import { validarPreparacion, fechasAlta, hastaPropuesto, construirLote, validarReciboLote, diaMadrid } from "./cambio-contratos.js?v=20261005-admin-lote-pantalla-v1";

export function montarCambioPerfiles(cont, { textos, cliente, cripto = globalThis.crypto, persona, nombre, unidadRef, ahora = () => Date.now(), alVolver } = {}) {
  if (!cont || typeof textos?.traducir !== "function" || typeof cliente?.preparar !== "function" || typeof cliente?.aplicarLote !== "function"
    || typeof persona !== "string" || typeof unidadRef !== "string" || typeof alVolver !== "function") throw new TypeError("montaje_invalido");
  const t = textos.traducir;
  const tx = (clave, variables) => escapar(t(clave, variables));
  const fecha = (valor, larga = false) => textos.fecha(new Date(valor), larga ? { dateStyle: "long", timeZone: "Europe/Madrid" } : { dateStyle: "medium", timeZone: "Europe/Madrid" });
  let vivo = true, prep = null, cambio = null, cuerpo = null, enviando = false, incierto = false, control = null;
  const id = (c) => `${cont.id || "cambio"}-${c}`;
  const nodo = (c) => cont.querySelector(`#${id(c)}`);
  const paso = (n) => `<p class="cambio-paso">${tx("lote.paso", { paso: textos.numero(n), total: textos.numero(3) })}</p>`;
  const cabecera = (clave, n) => `<div class="cabecera-panel"><div>${n ? paso(n) : ""}<h3 id="${id("titulo")}" tabindex="-1">${tx(clave)}</h3><p>${escapar(nombre)}</p></div></div>`;
  const enfocar = () => nodo("titulo")?.focus?.();
  function pintar(html) { cont.innerHTML = html; enfocar(); }
  function abortar() { control?.abort(); control = null; }

  function mensajeError(e) {
    if (e?.codigo === "respuesta_incompatible") return "lote.errores.servicio";
    switch (e?.estado) {
      case 401: case 403: return "lote.errores.denegado";
      case 404: return "lote.errores.no_disponible";
      case 409: return "lote.errores.conflicto";
      case 400: case 413: case 422: return "lote.errores.invalido";
      default: return "lote.errores.servicio";
    }
  }
  function pintarError(clave, reintentar = true) {
    pintar(`${cabecera("lote.titulo")}<div class="cuerpo-panel"><div class="resumen-errores" role="alert"><p>${tx(clave)}</p></div>
      <div class="acciones-paso"><button type="button" class="boton-secundario" data-cambio="volver">${tx("lote.volver_ficha")}</button>
      ${reintentar ? `<button type="button" class="boton-primario" data-cambio="preparar">${tx("lote.volver_a_preparar")}</button>` : ""}</div></div>`);
  }

  async function preparar() {
    abortar(); prep = null; cambio = null; cuerpo = null;
    pintar(`${cabecera("lote.titulo")}<div class="cuerpo-panel"><p role="status" aria-live="polite">${tx("lote.cargando")}</p></div>`);
    const c = new AbortController(); control = c;
    try {
      const datos = await cliente.preparar(persona, unidadRef, c.signal);
      if (!vivo || control !== c) return;
      prep = validarPreparacion(datos, persona, unidadRef);
      pintarEleccion();
    } catch (e) { if (vivo && control === c && e?.name !== "AbortError") pintarError(mensajeError(e)); }
  }

  function pintarEleccion() {
    const altas = prep.altas.length, bajas = prep.bajas.length;
    pintar(`${cabecera("lote.elegir", 1)}<div class="cuerpo-panel">
      <div class="cambio-vias">
        <button type="button" class="boton-via" data-cambio="via-asignar" ${altas ? "" : `disabled aria-describedby="${id("sin-altas")}"`}><strong>${tx("lote.asignar")}</strong><span>${tx("lote.asignar_resumen", { cuenta: textos.numero(altas) })}</span></button>
        <button type="button" class="boton-via" data-cambio="via-retirar" ${bajas ? "" : `disabled aria-describedby="${id("sin-bajas")}"`}><strong>${tx("lote.retirar")}</strong><span>${tx("lote.retirar_resumen", { cuenta: textos.numero(bajas) })}</span></button>
      </div>
      ${altas ? "" : `<p id="${id("sin-altas")}" class="texto-secundario">${tx("lote.sin_altas")}</p>`}
      ${bajas ? "" : `<p id="${id("sin-bajas")}" class="texto-secundario">${tx("lote.sin_bajas")}</p>`}
      ${prep.truncado ? `<p class="texto-secundario">${tx("lote.truncado")}</p>` : ""}
      <div class="acciones-paso"><button type="button" class="boton-secundario" data-cambio="volver">${tx("lote.cancelar")}</button></div></div>`);
  }

  const motivos = (elegido = "") => `<label class="campo" for="${id("motivo")}"><span>${tx("lote.motivo")} <span class="obligatorio">${tx("lote.obligatorio")}</span></span>
    <select id="${id("motivo")}" required aria-describedby="${id("error-motivo")}"><option value="">${tx("lote.elegir_motivo")}</option>${prep.motivos.map((m) =>
      `<option value="${escapar(m.clave_i18n)}" ${m.clave_i18n === elegido ? "selected" : ""}>${tx(`lote.motivos.${m.clave_i18n}`)}</option>`).join("")}</select>
    <span id="${id("error-motivo")}" class="error-campo" hidden></span></label>
    <label class="campo" for="${id("referencia")}"><span>${tx("lote.referencia")} <span class="opcional">${tx("lote.opcional")}</span></span>
      <input type="text" id="${id("referencia")}" maxlength="256" autocomplete="off" aria-describedby="${id("pista-referencia")}"><span id="${id("pista-referencia")}" class="ayuda-campo">${tx("lote.referencia_pista")}</span></label>`;
  const resumenErrores = `<div id="${id("errores")}" class="resumen-errores" role="alert" tabindex="-1" hidden></div>`;

  function pintarAsignar() {
    const primera = prep.altas[0];
    const hoy = diaMadrid(ahora());
    pintar(`${cabecera("lote.asignar_titulo", 2)}<form id="${id("form")}" class="cuerpo-panel" novalidate>${resumenErrores}
      <label class="campo" for="${id("perfil")}"><span>${tx("lote.perfil")} <span class="obligatorio">${tx("lote.obligatorio")}</span></span>
        <select id="${id("perfil")}" required>${prep.altas.map((a, i) => `<option value="${i}">${escapar(a.nombre || t("lote.perfil_sin_nombre"))}</option>`).join("")}</select></label>
      <fieldset class="grupo-campo"><legend>${tx("lote.empieza")}</legend>
        <label class="usuarios-checkbox"><input type="radio" name="${id("empieza")}" value="ahora" checked> <span>${tx("lote.empieza_ahora")}</span></label>
        <label class="usuarios-checkbox"><input type="radio" name="${id("empieza")}" value="fecha"> <span>${tx("lote.empieza_fecha")}</span></label>
        <label class="campo" for="${id("desde")}" id="${id("bloque-desde")}" hidden><span>${tx("lote.desde")}</span><input type="date" id="${id("desde")}" min="${hoy}" aria-describedby="${id("error-desde")}"><span id="${id("error-desde")}" class="error-campo" hidden></span></label>
      </fieldset>
      <label class="campo" for="${id("hasta")}"><span>${tx("lote.hasta")} <span class="obligatorio">${tx("lote.obligatorio")}</span></span>
        <input type="date" id="${id("hasta")}" required value="${hastaPropuesto(primera, ahora())}" min="${hoy}" max="${diaMadrid(Date.parse(primera.vigente_hasta_maxima))}" aria-describedby="${id("error-hasta")}">
        <span id="${id("error-hasta")}" class="error-campo" hidden></span></label>
      ${motivos()}
      <div class="acciones-paso"><button type="button" class="boton-secundario" data-cambio="eleccion">${tx("lote.atras")}</button><button type="submit" class="boton-primario">${tx("lote.revisar")}</button></div></form>`);
  }

  function pintarRetirar() {
    pintar(`${cabecera("lote.retirar_titulo", 2)}<form id="${id("form")}" class="cuerpo-panel" novalidate>${resumenErrores}
      <fieldset class="grupo-campo"><legend>${tx("lote.perfil_retirar")} <span class="obligatorio">${tx("lote.obligatorio")}</span></legend>
        ${prep.bajas.map((b, i) => `<label class="usuarios-checkbox"><input type="radio" name="${id("baja")}" value="${i}" ${i === 0 ? "checked" : ""}>
          <span>${escapar(b.nombre || t("lote.perfil_sin_nombre"))} · ${tx("lote.vigente", { desde: fecha(b.vigente_desde), hasta: fecha(b.vigente_hasta) })}</span></label>`).join("")}
      </fieldset>
      ${motivos()}
      <div class="acciones-paso"><button type="button" class="boton-secundario" data-cambio="eleccion">${tx("lote.atras")}</button><button type="submit" class="boton-primario">${tx("lote.revisar")}</button></div></form>`);
  }

  function errorCampo(campo, clave) {
    const e = nodo(`error-${campo}`); if (e) { e.hidden = false; e.textContent = t(clave); }
    nodo(campo)?.setAttribute?.("aria-invalid", "true");
    return { campo, clave };
  }
  function mostrarErrores(errores) {
    const caja = nodo("errores");
    caja.innerHTML = `<p>${tx("lote.errores.resumen")}</p><ul>${errores.map((e) => `<li><a href="#${id(e.campo)}">${tx(e.clave)}</a></li>`).join("")}</ul>`;
    caja.hidden = false; caja.focus?.();
  }

  function leerFormulario() {
    for (const c of ["motivo", "hasta", "desde"]) { const e = nodo(`error-${c}`); if (e) { e.hidden = true; e.textContent = ""; } nodo(c)?.removeAttribute?.("aria-invalid"); }
    const errores = [];
    const motivo = nodo("motivo").value;
    if (!motivo) errores.push(errorCampo("motivo", "lote.errores.motivo_requerido"));
    let nuevo;
    if (cambio?.via === "asignar") {
      const alta = prep.altas[Number(nodo("perfil").value)];
      const empieza = cont.querySelector(`input[name="${id("empieza")}"]:checked`)?.value || "ahora";
      const fechas = alta ? fechasAlta(alta, { hasta: nodo("hasta").value, empieza, desde: nodo("desde").value }, ahora()) : { error: "fecha_hasta" };
      if (fechas.error) errores.push(errorCampo(fechas.error === "fecha_desde" ? "desde" : "hasta", `lote.errores.${fechas.error}`));
      nuevo = { via: "asignar", operacion: "otorgar", alta, fechas };
    } else {
      const baja = prep.bajas[Number(cont.querySelector(`input[name="${id("baja")}"]:checked`)?.value ?? -1)];
      if (!baja) errores.push(errorCampo("motivo", "lote.errores.perfil_requerido"));
      nuevo = { via: "retirar", operacion: "revocar", baja };
    }
    if (errores.length) { mostrarErrores(errores); return null; }
    return { ...nuevo, motivo, referencia: nodo("referencia").value };
  }

  function pintarRevision() {
    const filas = cambio.operacion === "otorgar"
      ? [["lote.perfil", cambio.alta.nombre], ["lote.empieza", cambio.fechas.vigente_desde ? fecha(cambio.fechas.vigente_desde, true) : t("lote.empieza_ahora")],
        ["lote.hasta", fecha(cambio.fechas.vigente_hasta, true)]]
      : [["lote.perfil", cambio.baja.nombre], ["lote.efecto", t("lote.efecto_retirar")]];
    filas.push(["lote.motivo", t(`lote.motivos.${cambio.motivo}`)]);
    if (cambio.referencia.trim()) filas.push(["lote.referencia", cambio.referencia.trim()]);
    pintar(`${cabecera(cambio.operacion === "otorgar" ? "lote.revision_asignar" : "lote.revision_retirar", 3)}<div class="cuerpo-panel">
      <dl class="resumen-expediente">${filas.map(([c, v]) => `<div class="fila-resumen"><dt>${tx(c)}</dt><dd>${escapar(v)}</dd></div>`).join("")}</dl>
      <div id="${id("resultado")}" role="status" aria-live="polite" tabindex="-1"></div>
      <div class="acciones-paso"><button type="button" class="boton-secundario" data-cambio="corregir" id="${id("corregir")}">${tx("lote.cambiar_datos")}</button>
      <button type="button" class="boton-primario" data-cambio="confirmar" id="${id("confirmar")}">${tx(cambio.operacion === "otorgar" ? "lote.confirmar_asignar" : "lote.confirmar_retirar")}</button></div></div>`);
  }

  async function confirmar() {
    if (enviando || !cambio || !prep) return;
    try { cuerpo = cuerpo || construirLote(prep, cambio, cambio.motivo, cambio.referencia, cripto); }
    catch (e) { nodo("resultado").textContent = t(e?.codigo === "referencia" ? "lote.errores.referencia" : "lote.errores.motivo_requerido"); return; }
    abortar(); const c = new AbortController(); control = c;
    enviando = true; incierto = true; nodo("confirmar").disabled = true; nodo("corregir").disabled = true;
    nodo("resultado").textContent = t("lote.enviando");
    try {
      const respuesta = await cliente.aplicarLote(cuerpo, c.signal);
      if (!vivo || control !== c) return;
      const recibo = validarReciboLote(respuesta, cuerpo);
      incierto = false; pintarRecibo(recibo);
    } catch (e) {
      if (!vivo || control !== c) return;
      // Sólo una respuesta del servidor dice que no se aplicó; sin ella, no se sabe.
      if (Number.isInteger(e?.estado) && e.estado >= 400 && e.estado < 500) { incierto = false; pintarError(mensajeError(e), e.estado === 409); }
      else pintarError("lote.errores.incierto", false);
    } finally { enviando = false; }
  }

  function pintarRecibo(r) {
    const c = r.cambios[0], otorgar = c.estado_posterior === "activo";
    const filas = [["lote.perfil", otorgar ? cambio.alta.nombre : cambio.baja.nombre], ["lote.fecha_confirmacion", fecha(r.confirmado_en, true)]];
    if (otorgar) { filas.push(["lote.empieza", fecha(c.vigente_desde, true)], ["lote.hasta", fecha(c.vigente_hasta, true)]); }
    pintar(`${cabecera(otorgar ? "lote.asignado" : "lote.retirado")}<div class="cuerpo-panel">
      <p>${tx(otorgar ? "lote.efecto_asignado" : "lote.efecto_retirado")}</p>
      <dl class="resumen-expediente">${filas.map(([k, v]) => `<div class="fila-resumen"><dt>${tx(k)}</dt><dd>${escapar(v)}</dd></div>`).join("")}</dl>
      <details><summary>${tx("general.tecnico")}</summary><code>${escapar(r.recibo_ref)}</code></details>
      <div class="acciones-paso"><button type="button" class="boton-primario" data-cambio="volver">${tx("lote.volver_ficha")}</button></div></div>`);
  }

  function click(evento) {
    const boton = evento.target?.closest?.("[data-cambio]"); if (!boton || boton.disabled || !cont.contains(boton)) return;
    const a = boton.dataset.cambio;
    if (enviando && a !== "volver") return;
    if (a === "volver") { if (incierto && enviando) return; abortar(); alVolver(); }
    else if (a === "preparar") void preparar();
    else if (a === "eleccion") pintarEleccion();
    else if (a === "via-asignar") { cambio = { via: "asignar" }; pintarAsignar(); }
    else if (a === "via-retirar") { cambio = { via: "retirar" }; pintarRetirar(); }
    else if (a === "corregir") { cuerpo = null; if (cambio.via === "asignar") pintarAsignar(); else pintarRetirar(); }
    else if (a === "confirmar") void confirmar();
  }
  function submit(evento) {
    evento.preventDefault?.();
    if (evento.target !== nodo("form") || enviando) return;
    const leido = leerFormulario(); if (!leido) return;
    cambio = leido; cuerpo = null; pintarRevision();
  }
  function change(evento) {
    if (evento.target?.name === id("empieza")) { nodo("bloque-desde").hidden = evento.target.value !== "fecha"; }
    if (evento.target === nodo("perfil")) {
      const alta = prep.altas[Number(nodo("perfil").value)];
      if (alta) { nodo("hasta").value = hastaPropuesto(alta, ahora()); nodo("hasta").max = diaMadrid(Date.parse(alta.vigente_hasta_maxima)); }
    }
  }
  const listeners = [["click", click], ["submit", submit], ["change", change]];
  listeners.forEach(([tipo, fn]) => cont.addEventListener(tipo, fn));
  const listo = preparar();
  return Object.freeze({ listo, incierto: () => incierto, desmontar() {
    if (!vivo) return; vivo = false; abortar(); listeners.forEach(([tipo, fn]) => cont.removeEventListener(tipo, fn)); cont.replaceChildren();
  } });
}
