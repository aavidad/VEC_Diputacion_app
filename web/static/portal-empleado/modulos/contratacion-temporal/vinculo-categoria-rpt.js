/**
 * Paso «Vincular categoría de la relación de puestos» de la ficha CT.
 *
 * La categoría la fija el análisis confirmado de RRHH (CT154 exige que el
 * vínculo coincida con ella); la pantalla la busca entre las publicadas para
 * tomar versión y huella de su publicación, muestra lo que se va a registrar
 * junto al botón y, sólo tras un recibo válido, la muestra como vinculada.
 */
import { validarEntradaVinculoRPT } from "./contrato-vinculo-categoria-rpt.js";
import { escaparHTML as e } from "./componentes-expedientes.js?v=20261010-ct-vinculo-rpt-cohorte-v10";
import { cargarTextos } from "../../../comun/textos.js";

export const cargarTextosVinculoCategoriaRPT = (opciones) => cargarTextos("contratacion-temporal-vinculo-categoria-rpt", opciones);

const MAXIMO_PAGINAS = 20;

export function montarVinculoCategoriaRPT({ raiz, cliente, expedienteRef, textos,
  generarClave = () => globalThis.crypto.randomUUID(), esVigente = () => true, alRegistrar = () => {}, alDenegar = null } = {}) {
  if (!raiz?.addEventListener || !raiz?.removeEventListener || !raiz?.replaceChildren
    || !["consultarVinculoRPT", "listarCategoriasRPT", "registrarVinculoRPT"].every((k) => typeof cliente?.[k] === "function")
    || typeof textos?.traducir !== "function" || typeof textos?.fecha !== "function" || typeof expedienteRef !== "string") {
    throw new TypeError("montaje_vinculo_categoria_rpt_invalido");
  }
  const t = (k, vars) => textos.traducir(k, vars);
  let activo = true, controlador = null, lectura = null, categorias = null, ayudaAbierta = false;
  let mensaje = "cargando", tono = "informacion", entrada = null, incierto = false, recibo = null, caducada = false;
  const vigente = () => activo && esVigente() && raiz.isConnected !== false;
  const etiqueta = (c) => c.atributos[`etiqueta_${textos.idioma}`] || c.etiqueta;
  const fila = (clave, valor) => `<div><dt>${e(t(clave))}</dt><dd>${e(valor)}</dd></div>`;
  const delAnalisis = () => categorias?.find((c) => c.categoria_id === lectura?.analisis.categoria_ref) ?? null;
  const vinculoVigente = () => Boolean(lectura?.vinculo && lectura.vinculo.categoria_id === lectura.analisis.categoria_ref);

  function listaPublicadas() {
    if (!categorias?.length) return "";
    return `<details class="ct-vinculo-publicadas"><summary>${e(textos.plural("ver_publicadas", categorias.length))}</summary>
      <ul aria-label="${e(t("publicadas"))}">${categorias.map((c) => `<li>${e(c.categoria_id === lectura.analisis.categoria_ref
        ? t("publicada_del_analisis", { categoria: etiqueta(c) }) : etiqueta(c))}</li>`).join("")}</ul></details>`;
  }
  function tecnico(v) {
    return `<details><summary>${e(t("detalle_tecnico"))}</summary><dl class="ct-resumen">
      ${fila("justificante", v.recibo_ref)}${fila("revision_vinculo", String(v.revision))}
      ${fila("fuente", v.fuente_ref)}${fila("aprobacion", v.aprobacion_ref)}</dl></details>`;
  }
  const boton = (accion, clave, primario = false) => `<button type="button" class="${primario ? "boton-primario" : "boton-secundario"}" data-vinculo-accion="${accion}"${controlador ? " disabled" : ""}>${e(t(clave))}</button>`;
  // Devuelve el contenido del paso y sus acciones: una sola barra por bloque.
  function cuerpo() {
    const actualizar = boton("consultar", incierto ? "comprobar" : "actualizar");
    if (!lectura || categorias === null) return ["", controlador || mensaje === "cargando" ? [] : [actualizar]];
    const c = delAnalisis();
    if (recibo || vinculoVigente()) {
      const v = recibo?.vinculo ?? lectura.vinculo;
      const vc = categorias.find((x) => x.categoria_id === v.categoria_id);
      return [`<section class="ct-recibo"><h4>${e(t("vinculada"))}</h4><dl class="ct-resumen">
        ${fila("categoria", vc ? etiqueta(vc) : v.categoria_id)}${fila("catalogo", t("version_catalogo", { version: v.catalogo_version }))}
        ${recibo ? fila("fecha_registro", textos.fecha(recibo.registrado_en, { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" })) : ""}</dl>
        <p>${e(t("vinculada_texto"))}</p>${tecnico(v)}</section>`, []];
    }
    if (!c) return [listaPublicadas(), [actualizar]];
    const datos = `<dl class="ct-resumen">${fila("categoria", etiqueta(c))}${c.atributos.grupos ? fila("grupos_rotulo", c.atributos.grupos) : ""}
      ${fila("catalogo", t("version_catalogo", { version: c.catalogo_version }))}</dl>`;
    return [`<section class="ct-revision" data-vinculo-categoria><h4 tabindex="-1" data-vinculo-categoria-titulo>${e(t("categoria_analisis"))}</h4>${datos}
      <p>${e(t("origen"))}</p>${listaPublicadas()}${incierto || caducada ? "" : `<p>${e(t("aviso_vincular"))}</p>`}</section>`,
      incierto || caducada ? [actualizar] : [actualizar, boton("vincular", "vincular", true)]];
  }
  function estadoVisible() {
    if (mensaje) return [mensaje, tono];
    if (!lectura || categorias === null) return ["", "informacion"];
    if (recibo || vinculoVigente()) return ["", "exito"];
    if (!categorias.length) return ["sin_publicadas", "aviso"];
    if (!delAnalisis()) return ["no_publicada", "aviso"];
    if (lectura.vinculo) {
      const anterior = categorias.find((x) => x.categoria_id === lectura.vinculo.categoria_id);
      return [["cambio_analisis", { anterior: anterior ? etiqueta(anterior) : lectura.vinculo.categoria_id }], "aviso"];
    }
    return ["", "informacion"];
  }
  function pintar(foco = null) {
    if (!vigente()) return;
    const hecho = Boolean(recibo || vinculoVigente());
    const etapa = hecho ? 1 : 0;
    const [m, tonoActual] = estadoVisible();
    const textoMensaje = Array.isArray(m) ? t(m[0], m[1]) : m ? t(m) : "";
    const [contenido, acciones] = cuerpo();
    raiz.innerHTML = `<section class="panel" data-vinculo-rpt aria-busy="${Boolean(controlador)}">
      <header class="cabecera-panel"><div><h3 tabindex="-1" data-vinculo-titulo>${e(t("titulo"))}</h3></div>
      <button type="button" class="boton-secundario" data-vinculo-accion="ayuda" aria-expanded="${ayudaAbierta}" aria-controls="ct-vinculo-rpt-ayuda" aria-label="${e(t("ayuda_boton"))}">?</button></header>
      <div class="cuerpo-panel"><p id="ct-vinculo-rpt-ayuda" class="ct-ayuda"${ayudaAbierta ? "" : " hidden"}>${e(t("ayuda"))}</p><ol class="ct-pasos" aria-label="${e(t("pasos"))}">${["comprobar", "vinculada"].map((k, i) => {
        const completado = i < etapa || hecho;
        return `<li${etapa === i && !hecho ? ' aria-current="step"' : ""}${completado ? ' data-completado="true"' : ""}><span aria-hidden="true">${completado ? "✓" : i + 1}</span>${e(t(`paso_${k}`))}${completado ? ` ${e(t("paso_hecho"))}` : ""}</li>`;
      }).join("")}</ol>
      <p class="ct-estado ct-estado-${tonoActual}" role="status" aria-live="polite" tabindex="-1" data-vinculo-mensaje${textoMensaje ? "" : " hidden"}>${e(textoMensaje)}</p>
      ${contenido}${acciones.length ? `<div class="ct-acciones">${acciones.join("")}</div>` : ""}</div></section>`;
    if (foco) raiz.querySelector?.(foco)?.focus?.();
  }
  function fallo(error, alRegistrarFallo = false) {
    const codigo = error?.envelopeValido ? error.codigo : "";
    if (error?.envelopeValido && [401, 403].includes(error.estado)) {
      lectura = null; categorias = null; entrada = null; incierto = false;
      mensaje = "denegada"; tono = "aviso"; return;
    }
    if (alRegistrarFallo && !(error?.envelopeValido && [400, 409, 422].includes(error.estado))) {
      incierto = true; mensaje = "incierto"; tono = "aviso"; return;
    }
    tono = "aviso";
    mensaje = codigo === "preparacion_pendiente" ? "pendiente" : error?.estado === 409 ? "conflicto"
      : [400, 422].includes(error?.estado) ? "rechazada" : "no_disponible";
    // Tras un rechazo del registro los datos leídos ya no sirven: sólo Actualizar.
    if (alRegistrarFallo) { entrada = null; caducada = true; }
  }
  async function leerCategorias(signal) {
    const todas = [];
    let cursor = "";
    for (let i = 0; i < MAXIMO_PAGINAS; i++) {
      const p = await cliente.listarCategoriasRPT(cursor, { signal });
      todas.push(...p.categorias);
      if (!p.hay_mas) return todas;
      cursor = p.siguiente_cursor;
    }
    throw new Error("categorias_rpt_excesivas");
  }
  async function consultar() {
    if (!vigente() || controlador) return;
    const actual = new AbortController(); controlador = actual; mensaje = "cargando"; tono = "informacion"; pintar();
    try {
      const l = await cliente.consultarVinculoRPT(expedienteRef, { signal: actual.signal });
      if (!vigente() || actual.signal.aborted) return;
      const lista = await leerCategorias(actual.signal);
      if (!vigente() || actual.signal.aborted) return;
      lectura = l; categorias = lista; mensaje = ""; caducada = false;
      if (incierto && entrada && lectura.vinculo?.categoria_id === entrada.categoria_id
        && lectura.vinculo.revision === entrada.revision_esperada + 1) {
        incierto = false; entrada = null;
        try { await alRegistrar(lectura.vinculo); } catch { /* El vínculo leído se conserva. */ }
      } else if (incierto) {
        // No quedó registrado: se puede repetir la misma intención y clave.
        incierto = false;
      }
    } catch (error) {
      if (vigente() && !actual.signal.aborted) fallo(error);
    } finally {
      if (controlador === actual) {
        controlador = null;
        // Sin permiso para el vínculo, quien monta puede retirar el bloque.
        if (mensaje === "denegada" && typeof alDenegar === "function") { alDenegar(); return; }
        pintar(mensaje ? "[data-vinculo-mensaje]" : null);
      }
    }
  }
  // Un solo acto: la intención se arma con lo que hay en pantalla y se envía.
  // Si el resultado queda incierto, «Comprobar» reutiliza la misma clave.
  async function vincular() {
    const c = delAnalisis();
    if (!vigente() || controlador || !c || vinculoVigente() || incierto || caducada) return;
    const a = lectura.analisis, previo = lectura.vinculo;
    const reintento = entrada && entrada.categoria_id === c.categoria_id && entrada.catalogo_huella_sha256 === c.catalogo_huella_sha256
      && entrada.version_expediente_esperada === a.version_expediente && entrada.analisis_huella_sha256 === a.analisis_huella_sha256
      && entrada.revision_esperada === (previo?.revision ?? 0);
    if (!reintento) try {
      entrada = validarEntradaVinculoRPT({ expediente_ref: expedienteRef, version_expediente_esperada: a.version_expediente,
        analisis_version: a.analisis_version, analisis_recibo_ref: a.analisis_recibo_ref, analisis_huella_sha256: a.analisis_huella_sha256,
        categoria_ref: a.categoria_ref, catalogo_version: c.catalogo_version, catalogo_huella_sha256: c.catalogo_huella_sha256,
        categoria_id: c.categoria_id, fuente_ref: c.fuente_ref, motivo_ref: a.analisis_recibo_ref, aprobacion_ref: c.aprobacion_ref,
        revision_esperada: previo?.revision ?? 0, anterior_recibo_ref: previo?.recibo_ref ?? "", clave_idempotencia: generarClave() });
    } catch { entrada = null; caducada = true; mensaje = "rechazada"; tono = "aviso"; pintar("[data-vinculo-mensaje]"); return; }
    const actual = new AbortController(); controlador = actual; mensaje = "registrando"; tono = "informacion"; pintar("[data-vinculo-mensaje]");
    try {
      const r = await cliente.registrarVinculoRPT(entrada, { signal: actual.signal });
      if (!vigente() || actual.signal.aborted) return;
      recibo = r; entrada = null; mensaje = "";
      try { await alRegistrar(r.vinculo); } catch { /* El recibo validado se conserva. */ }
    } catch (error) {
      if (vigente() && !actual.signal.aborted) fallo(error, true);
    } finally {
      if (controlador === actual) { controlador = null; pintar("[data-vinculo-mensaje]"); }
    }
  }
  function click(evento) {
    const accion = evento.target?.closest?.("[data-vinculo-accion]")?.getAttribute?.("data-vinculo-accion");
    if (accion === "consultar") return consultar();
    if (accion === "vincular") return vincular();
    if (accion === "ayuda") {
      ayudaAbierta = !ayudaAbierta;
      const boton = raiz.querySelector?.('[data-vinculo-accion="ayuda"]'), ayuda = raiz.querySelector?.("#ct-vinculo-rpt-ayuda");
      if (boton && ayuda) { boton.setAttribute("aria-expanded", String(ayudaAbierta)); ayuda.hidden = !ayudaAbierta; } else pintar();
    }
  }
  raiz.addEventListener("click", click);
  pintar(); void consultar();
  return () => {
    activo = false; controlador?.abort(); controlador = null;
    raiz.removeEventListener("click", click); raiz.replaceChildren();
    lectura = null; categorias = null; entrada = null; recibo = null;
  };
}
