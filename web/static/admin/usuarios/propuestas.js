import { escapar } from "./render.js?v=20261005-admin-lote-pantalla-v1";
import { validarPropuestas, puedeCerrarPropuesta, prepararCierre, validarCierre } from "./propuestas-contratos.js?v=20261004-admin-usuarios-metadata-v1";
/** Segunda persona: la pista de la lectura nunca sustituye el cierre autorizado. */
export function montarPropuestas(host, { textos, contexto, bloquear, denegar, fallarLectura, cripto }) {
  const t = textos.traducir, tx = (k, vars) => escapar(t(k, vars));
  const fecha = (v) => escapar(textos.fecha(v, { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }));
  const dato = (k, v) => `<div class="fila-resumen"><dt>${tx(k)}</dt><dd>${escapar(v)}</dd></div>`;
  const identidadProponente = (p) => t("historia.identidad", { persona: p.proponente_nombre || t("detalle.identidad_pendiente"), perfil: p.proponente_perfil_nombre || t("historia.perfil_pendiente") });
  const detallePropuesta = (p) => `${dato("propuestas.proponente", identidadProponente(p))}
    ${dato("detalle.ambito", p.ambitos?.map((a) => a.nombre).join(t("general.separador")) || t("detalle.ambito_pendiente"))}
    ${dato("detalle.desde", p.vigente_desde && !p.vigente_desde.startsWith("0001-") ? textos.fecha(p.vigente_desde, { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }) : t("detalle.sin_fecha"))}
    ${dato("detalle.hasta", p.vigente_hasta && !p.vigente_hasta.startsWith("0001-") ? textos.fecha(p.vigente_hasta, { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }) : t("detalle.sin_fecha"))}
    ${dato("propuestas.motivo", p.motivo?.etiqueta || t("historia.motivo_pendiente"))}`;
  let control, viva = true, propuestas = [], revision = null, seleccion = null, origenRef = null, enviando = false, incierto = false;
  host.setAttribute("tabindex", "-1");
  const nodo = (k) => host.querySelector(`[data-propuesta="${k}"]`);
  const vigente = (c) => viva && control === c && !c.signal.aborted;
  const nuevo = () => { control?.abort(); control = new AbortController(); return control; };
  function fallo(e) {
    if ([401, 403].includes(e?.estado)) { incierto = false; bloquear(false); denegar(e); return; }
    const n = nodo("resultado") || nodo("estado");
    if (n) { n.textContent = t(e?.estado === 409 ? "errores.conflicto" : e?.codigo === "respuesta_incompatible" ? "errores.incompatible" : "errores.servicio"); n.focus(); }
  }
  const rolNombre = (p) => contexto().roles.find((r) => r.version_ref === p.rol_version_ref)?.etiqueta || t("detalle.nombre_pendiente");
  function permitido(p) {
    const { actor, capacidades, cliente, roles } = contexto(), rol = roles.find((r) => r.version_ref === p.rol_version_ref);
    return rol && rol.clase !== "ordinario" && puedeCerrarPropuesta(p, actor, capacidades, cliente);
  }
  function tabla(restaurar = false) {
    host.innerHTML = `<div class="cabecera-panel"><h3>${tx("propuestas.titulo")}</h3></div><div class="cuerpo-panel"><p data-propuesta="estado" role="status" tabindex="-1"></p>
      ${propuestas.length ? `<div class="tabla-contenedor" tabindex="0" aria-label="${tx("propuestas.titulo")}"><table class="tabla-datos"><caption>${tx("propuestas.titulo")}</caption><thead><tr><th scope="col">${tx("busqueda.nombre")}</th><th scope="col">${tx("busqueda.perfil")}</th><th scope="col">${tx("propuestas.proponente")}</th><th scope="col">${tx("resultado.caduca")}</th><th scope="col">${tx("propuestas.revision")}</th></tr></thead><tbody>
        ${propuestas.map((p, i) => `<tr><th scope="row">${escapar(p.objetivo_nombre)}</th><td>${escapar(rolNombre(p))}</td><td>${escapar(identidadProponente(p))}</td><td>${fecha(p.caduca_en)}</td><td><button type="button" class="boton-secundario" data-propuesta-accion="abrir" data-indice="${i}">${tx("propuestas.revisar")}</button></td></tr>`).join("")}</tbody></table></div>` : `<p>${tx("propuestas.vacia")}</p>`}</div>`;
    if (restaurar) {
      const i = propuestas.findIndex((p) => p.propuesta_ref === origenRef);
      (host.querySelector(`[data-propuesta-accion="abrir"][data-indice="${i}"]`) || host).focus();
    }
  }
  function ficha(p) {
    host.innerHTML = `<div class="cabecera-panel"><h3>${tx("propuestas.revisar")}</h3><button type="button" class="boton-secundario" data-propuesta-accion="volver">${tx("propuestas.volver")}</button></div><div class="cuerpo-panel">
      <dl class="resumen-expediente"><div class="fila-resumen"><dt>${tx("busqueda.nombre")}</dt><dd>${escapar(p.objetivo_nombre)}</dd></div><div class="fila-resumen"><dt>${tx("busqueda.perfil")}</dt><dd>${escapar(rolNombre(p))}</dd></div><div class="fila-resumen"><dt>${tx("detalle.operacion")}</dt><dd>${tx(`operaciones.${p.operacion}`)}</dd></div><div class="fila-resumen"><dt>${tx("propuestas.proponente")}</dt><dd>${escapar(identidadProponente(p))}</dd></div><div class="fila-resumen"><dt>${tx("resultado.caduca")}</dt><dd>${fecha(p.caduca_en)}</dd></div></dl>
      <dl class="resumen-expediente">${detallePropuesta(p)}</dl>
      ${permitido(p) ? `<form data-propuesta="formulario"><label class="campo">${tx("propuestas.decision")}<select data-propuesta="decision" required><option value="aprobar">${tx("propuestas.aprobar")}</option><option value="rechazar">${tx("propuestas.rechazar")}</option></select></label>
        <label class="campo">${tx("detalle.motivo")}<select data-propuesta="motivo" required><option value="">${tx("detalle.elegir_motivo")}</option>${p.motivos_cierre.map((m, i) => `<option value="${i}">${escapar(m.etiqueta)}</option>`).join("")}</select></label>
        <div class="acciones-paso"><button class="boton-primario" type="submit">${tx("detalle.revisar")}</button></div></form>` : `<p>${tx("propuestas.sin_cierre")}</p>`}<p data-propuesta="estado" role="alert" tabindex="-1"></p></div>`;
    host.focus();
  }
  function revisar() {
    const { actor, capacidades, cliente } = contexto();
    revision = prepararCierre(seleccion, actor, capacidades, cliente, nodo("decision").value, Number(nodo("motivo").value), cripto);
    host.innerHTML = `<div class="cabecera-panel"><h3>${tx("propuestas.confirmar_titulo")}</h3></div><div class="cuerpo-panel"><p>${escapar(revision.propuesta.objetivo_nombre)} · ${escapar(rolNombre(revision.propuesta))}</p><dl class="resumen-expediente">${detallePropuesta(revision.propuesta)}</dl><p>${tx(`propuestas.${revision.decision}`)}</p><p>${escapar(revision.motivo)}</p><p>${tx(revision.decision === "aprobar" ? "propuestas.efecto_aprobar" : "propuestas.efecto_rechazar")}</p>
      <p data-propuesta="resultado" role="status" tabindex="-1" aria-live="polite"></p><div class="acciones-paso"><button class="boton-secundario" type="button" data-propuesta="corregir" data-propuesta-accion="corregir">${tx("revision.corregir")}</button><button class="boton-primario" type="button" data-propuesta="confirmar" data-propuesta-accion="confirmar">${tx("propuestas.confirmar")}</button></div></div>`;
    nodo("confirmar").focus();
  }
  async function confirmar() {
    if (!revision || enviando) return;
    const { cliente } = contexto(), copia = revision, c = nuevo();
    enviando = true; incierto = true; bloquear(true); nodo("confirmar").disabled = nodo("corregir").disabled = true;
    nodo("resultado").textContent = t("revision.enviando");
    try {
      const datos = await cliente.cerrarPropuesta(copia.propuesta.propuesta_ref, copia.cuerpo, c.signal);
      if (!vigente(c)) return;
      const r = validarCierre(datos, copia); incierto = false; bloquear(false); revision = null;
      nodo("resultado").innerHTML = `<strong>${tx(r.decision === "aprobada" ? "propuestas.aprobada" : "propuestas.rechazada")}</strong><p>${fecha(r.fecha)}</p><details><summary>${tx("general.tecnico")}</summary><code>${escapar(r.referencia)}</code></details>`;
      nodo("resultado").focus(); nodo("corregir").textContent = t("propuestas.volver");
    } catch (e) {
      if (!vigente(c)) return;
      if ([400, 401, 403, 409, 413, 422].includes(e?.estado)) { incierto = false; bloquear(false); }
      fallo(e);
      if (incierto) nodo("resultado").textContent = t("errores.resultado_pendiente");
      if (e?.estado === 409) revision = null;
    } finally {
      enviando = false;
      if (vigente(c)) { nodo("confirmar").disabled = !revision; nodo("corregir").disabled = incierto; }
    }
  }
  async function cargar(restaurar = false) {
    if (!viva || enviando || incierto) return;
    const { cliente } = contexto();
    revision = null; seleccion = null; propuestas = []; tabla();
    if (typeof cliente.propuestas !== "function") { nodo("estado").textContent = t("propuestas.sin_consulta"); return; }
    const c = nuevo(); nodo("estado").textContent = t("propuestas.cargando");
    try { const datos = await cliente.propuestas(c.signal); if (vigente(c)) { propuestas = validarPropuestas(datos); tabla(restaurar); } }
    catch (e) { if (vigente(c)) fallarLectura(e); }
  }
  function click(e) {
    const n = e.target.closest("[data-propuesta-accion]");
    if (!n || !host.contains(n) || n.disabled || enviando) return;
    const accion = n.dataset.propuestaAccion;
    if (incierto && accion !== "confirmar") return;
    if (accion === "abrir") { seleccion = propuestas[Number(n.dataset.indice)]; if (seleccion) { origenRef = seleccion.propuesta_ref; ficha(seleccion); } }
    else if (accion === "confirmar") void confirmar();
    else if (accion === "volver") { revision = null; seleccion = null; tabla(true); }
    else if (accion === "corregir") { if (!revision) void cargar(true); else { revision = null; ficha(seleccion); } }
  }
  function submit(e) { if (e.target !== nodo("formulario")) return; e.preventDefault(); if (enviando || incierto) return;
    try { if (nodo("motivo").value === "") throw new Error(); revisar(); } catch { nodo("estado").textContent = t("errores.seleccion"); nodo("estado").focus(); } }
  host.addEventListener("click", click); host.addEventListener("submit", submit);
  return { cargar, cancelarLectura() { if (!enviando && !incierto) control?.abort(); },
    vaciar() { control?.abort(); revision = null; seleccion = null; propuestas = []; host.replaceChildren(); },
    desmontar() { viva = false; control?.abort(); host.removeEventListener("click", click); host.removeEventListener("submit", submit); host.replaceChildren(); } };
}
