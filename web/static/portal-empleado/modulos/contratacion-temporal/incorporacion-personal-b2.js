import { validarConsultaB2, validarSolicitudPlanB2, validarSolicitudConfirmacionB2, validarReciboB2, fechaCivilB2 } from "./contrato-incorporacion-personal-b2.js";
import { escaparHTML as e } from "./componentes-expedientes.js";
import { cargarTextos } from "../../../comun/textos.js";

export const cargarTextosIncorporacionPersonalB2 = (opciones) => cargarTextos("contratacion-temporal-incorporacion-personal-b2", opciones);

/** Montaje cancelable: la revisión local precede al plan durable y su confirmación. */
export function montarIncorporacionPersonalB2({ raiz, cliente, expedienteRef, versionEsperada, textos,
  resolverEtiqueta, generarClave = () => globalThis.crypto.randomUUID(), esVigente = () => true,
  alConfirmar = () => {}, entorno = globalThis } = {}) {
  if (!raiz?.addEventListener || !raiz?.removeEventListener || !raiz?.replaceChildren
    || !["consultar", "preparar", "confirmar"].every((k) => typeof cliente?.[k] === "function")
    || typeof textos?.traducir !== "function" || typeof textos?.fecha !== "function"
    || !Number.isSafeInteger(versionEsperada) || versionEsperada < 1) throw new TypeError("montaje_incorporacion_b2_invalido");
  const t = (k, vars) => textos.traducir(k, vars);
  const traducirDato = (clave) => {
    let valor;
    try { valor = t(clave); } catch { valor = resolverEtiqueta?.(clave); }
    if (typeof valor !== "string" || !valor || valor === clave) throw new TypeError("etiqueta_b2_no_disponible");
    return valor;
  };
  let activo = true, controlador = null, consulta = null, recibo = null, fase = "datos", mensaje = "cargando";
  let intencion = null, plan = null, incierto = false, denegado = false, errores = {}, valores = {}, modificado = false;
  const vigente = () => activo && esVigente() && raiz.isConnected !== false;
  const fila = (clave, valor) => `<div><dt>${e(t(clave))}</dt><dd>${e(valor)}</dd></div>`;
  const fecha = (v) => textos.fecha(`${v}T00:00:00Z`, { dateStyle: "medium", timeZone: "UTC" });
  const puedePreparar = () => consulta && consulta.version_expediente_actual === versionEsperada
    && consulta.prerrequisitos.length > 0 && consulta.prerrequisitos.every((p) => p.cumplido)
    && ["vacantes", "regimenes", "modalidades", "clases_ocupacion", "motivos", "documentos"].every((k) => consulta.opciones[k].length > 0);
  const campos = ["vacante", "regimen", "modalidad", "clase_ocupacion", "desde", "hasta", "motivo", "documento"];
  const valorOpcion = (lista, v) => typeof v === "string" && /^(?:0|[1-9]\d*)$/u.test(v) ? lista[Number(v)] : undefined;
  const fechaFuente = () => Boolean(consulta?.opciones.periodo.fuente_ref && consulta.opciones.periodo.desde);
  function erroresDe(v) {
    const o = consulta.opciones, resultado = {};
    for (const [campo, lista] of [["vacante", o.vacantes], ["regimen", o.regimenes], ["modalidad", o.modalidades], ["clase_ocupacion", o.clases_ocupacion], ["motivo", o.motivos], ["documento", o.documentos]]) {
      if (valorOpcion(lista, v[campo]) === undefined) resultado[campo] = "error_campo";
    }
    if (!fechaCivilB2(v.desde)) resultado.desde = "error_campo";
    if (v.hasta !== "" && (!fechaCivilB2(v.hasta) || v.hasta <= v.desde)) resultado.hasta = "error_periodo";
    if (fechaFuente() && (v.desde !== o.periodo.desde || v.hasta !== o.periodo.hasta)) resultado.desde = "error_campo";
    return resultado;
  }
  function recoger(formulario) {
    return Object.fromEntries(campos.map((c) => [c, formulario.elements[c]?.value ?? valores[c] ?? ""]));
  }
  function prepararIntencion() {
    const o = consulta.opciones, vacante = valorOpcion(o.vacantes, valores.vacante);
    const doc = valorOpcion(o.documentos, valores.documento);
    const cat = (campo, lista) => { const x = valorOpcion(lista, valores[campo]); return { ref: x.ref, version: x.version }; };
    return validarSolicitudPlanB2({ expediente_ref: expedienteRef, version_expediente: consulta.version_expediente_actual,
      puesto_ref: vacante.puesto_ref, plaza_ref: vacante.plaza_ref, version_plantilla_ref: vacante.version_plantilla_ref,
      version_rpt_ref: vacante.version_rpt_ref, regimen: cat("regimen", o.regimenes), modalidad: cat("modalidad", o.modalidades), clase_ocupacion: valorOpcion(o.clases_ocupacion, valores.clase_ocupacion).valor,
      desde: valores.desde, hasta: valores.hasta, motivo_clave: valorOpcion(o.motivos, valores.motivo),
      documento_ref: doc.documento_ref, documento_sha256: doc.documento_sha256, clave_idempotencia: generarClave() });
  }
  function resumen(s) {
    const o = consulta.opciones;
    const vacante = o.vacantes.find((x) => x.puesto_ref === s.puesto_ref && x.plaza_ref === s.plaza_ref);
    const regimen = o.regimenes.find((x) => x.ref === s.regimen.ref && x.version === s.regimen.version);
    const modalidad = o.modalidades.find((x) => x.ref === s.modalidad.ref && x.version === s.modalidad.version);
    const clase = o.clases_ocupacion.find((x) => x.valor === s.clase_ocupacion);
    const documento = o.documentos.find((x) => x.documento_ref === s.documento_ref && x.documento_sha256 === s.documento_sha256);
    if (!vacante || !regimen || !modalidad || !clase || !documento) throw new TypeError("seleccion_b2_no_disponible");
    return `<dl class="ct-resumen">${fila("persona", t("persona_aceptada"))}
      ${fila("puesto", `${vacante.puesto_etiqueta} · ${vacante.plaza_etiqueta}`)}
      ${fila("regimen", regimen.denominacion)}${fila("modalidad", modalidad.denominacion)}${fila("clase_ocupacion", traducirDato(clase.texto_clave))}
      ${fila("desde", fecha(s.desde))}${fila("hasta", s.hasta ? fecha(s.hasta) : t("sin_fin"))}
      ${fila("motivo", traducirDato(`motivo.${s.motivo_clave}`))}${fila("documento", traducirDato(documento.etiqueta_clave_i18n))}</dl>`;
  }
  function select(campo, clave, lista, rotulo, seleccionExpresa = false) {
    if (lista.length === 1 && !seleccionExpresa) return `<div class="ct-campo"><span>${e(t(clave))}</span><span class="ct-b2-valor-solo-lectura">${e(rotulo(lista[0]))}</span></div>`;
    return `<label class="ct-campo" for="ct-b2-${campo}"><span>${e(t(clave))}</span>
      <select id="ct-b2-${campo}" name="${campo}" required${errores[campo] ? ' aria-invalid="true"' : ""}
        aria-describedby="ct-b2-error-${campo}"><option value="">${e(t("seleccionar"))}</option>
        ${lista.map((x, i) => `<option value="${i}"${valores[campo] === String(i) ? " selected" : ""}>${e(rotulo(x))}</option>`).join("")}</select>
      <small class="ct-error-campo" id="ct-b2-error-${campo}">${errores[campo] ? e(t(errores[campo])) : ""}</small></label>`;
  }
  function formulario() {
    const o = consulta.opciones;
    const datoFecha = (c) => fechaFuente() ? `<label class="ct-campo"><span>${e(t(c))}</span><input type="text" readonly
      value="${e(valores[c] ? fecha(valores[c]) : t("sin_fin"))}"></label>` : `<label class="ct-campo" for="ct-b2-${c}"><span>${e(t(c))}</span><input type="date" id="ct-b2-${c}"
      name="${c}" value="${e(valores[c])}"${c === "desde" ? " required" : ""}${fechaFuente() ? " readonly" : ""}
      ${errores[c] ? 'aria-invalid="true"' : ""} aria-describedby="ct-b2-error-${c} ct-b2-pista-${c}">
      <small id="ct-b2-pista-${c}">${e(t(c === "hasta" ? "hasta_exclusiva" : "fecha_formato"))}</small>
      <small class="ct-error-campo" id="ct-b2-error-${c}">${errores[c] ? e(t(errores[c])) : ""}</small></label>`;
    return `<form class="ct-formulario" data-b2-form novalidate><fieldset class="ct-bloque"${controlador ? " disabled" : ""}>
      <legend>${e(t("contexto"))}</legend><p>${e(t("persona_aceptada"))}</p>
      <div class="ct-campos">${select("vacante", "puesto", o.vacantes, (x) => `${x.puesto_etiqueta} · ${x.plaza_etiqueta}`)}
        ${select("regimen", "regimen", o.regimenes, (x) => x.denominacion)}
        ${select("modalidad", "modalidad", o.modalidades, (x) => x.denominacion)}
        ${select("clase_ocupacion", "clase_ocupacion", o.clases_ocupacion, (x) => traducirDato(x.texto_clave), true)}${datoFecha("desde")}${datoFecha("hasta")}
        ${select("motivo", "motivo", o.motivos, (x) => traducirDato(`motivo.${x}`))}
        ${select("documento", "documento", o.documentos, (x) => traducirDato(x.etiqueta_clave_i18n))}</div>
      <p class="ct-ayuda">${e(t(fechaFuente() ? "periodo_fuente" : "periodo_pendiente"))}</p></fieldset>
      <div class="ct-acciones"><button class="boton-primario" type="submit"${controlador ? " disabled" : ""}>${e(t("revisar"))}</button></div></form>`;
  }
  function pintar(foco = null) {
    if (!vigente()) return;
    let cuerpo = "";
    try {
      if (recibo) {
        cuerpo = `<section class="ct-recibo" role="status"><h4>${e(t("confirmada"))}</h4><dl class="ct-resumen">
          ${fila("recibo", recibo.recibo_ref)}${fila("fecha_registro", textos.fecha(recibo.registrada_en, { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }))}
          </dl><p>${e(t("siguiente"))}</p><p>${e(t("limite"))}</p><details><summary>${e(t("detalle_tecnico"))}</summary><dl>
          ${["plan", "empleado", "relacion", "ocupacion"].map((k) => fila(k, recibo[`${k}_ref`])).join("")}</dl></details></section>`;
      } else if (!denegado && consulta) {
        const previos = `<h4>${e(t("prerrequisitos"))}</h4><ul>${consulta.prerrequisitos.map((p) => `<li>${e(traducirDato(p.clave_i18n))}: ${e(t(p.cumplido ? "cumplido" : "pendiente"))}</li>`).join("")}</ul>`;
        cuerpo = previos;
        if (fase === "revision" && (intencion || plan)) {
          cuerpo += `<section class="ct-revision"><h4>${e(t("paso_revision"))}</h4>${resumen(plan?.intencion ?? intencion)}
            <p>${e(t(plan ? "plan_preparado" : "limite"))}</p><div class="ct-acciones">
            ${!plan && !incierto ? `<button type="button" class="boton-secundario" data-b2-accion="cambiar">${e(t("cambiar"))}</button>` : ""}
            <button type="button" class="boton-primario" data-b2-accion="registrar"${controlador || incierto || !consulta.prerrequisitos.length || !consulta.prerrequisitos.every((p) => p.cumplido) ? " disabled" : ""}>${e(t(plan ? "continuar" : "registrar"))}</button></div></section>`;
        } else if (puedePreparar() && !incierto) cuerpo += formulario();
        else if (!incierto) {
          cuerpo += `<p>${e(t(!consulta.opciones.vacantes.length ? "sin_puestos"
            : !consulta.opciones.documentos.length ? "sin_documento"
              : !consulta.opciones.regimenes.length || !consulta.opciones.modalidades.length || !consulta.opciones.clases_ocupacion.length ? "sin_catalogos" : "faltan_comprobaciones"))}</p>`;
        }
      }
    } catch { cuerpo = ""; mensaje = "no_disponible"; }
    const etapa = recibo ? 2 : fase === "revision" ? 1 : 0;
    raiz.innerHTML = `<section class="panel" data-b2-incorporacion aria-busy="${Boolean(controlador)}">
      <header class="cabecera-panel"><div><h3 tabindex="-1" data-b2-titulo>${e(t("titulo"))}</h3><p>${e(t("subtitulo"))}</p></div>
      <details><summary aria-label="${e(t("ayuda_boton"))}">?</summary><p>${e(t("ayuda"))}</p></details></header>
      <div class="cuerpo-panel"><ol class="ct-pasos" aria-label="${e(t("pasos"))}">${["datos", "revision", "confirmacion"].map((k, i) =>
        `<li${etapa === i ? ' aria-current="step"' : ""}${i < etapa ? ' data-completado="true"' : ""}><span aria-hidden="true">${i + 1}</span>${e(t(`paso_${k}`))}</li>`).join("")}</ol>
      ${Object.keys(errores).length ? `<section class="ct-resumen-errores" role="alert" tabindex="-1" data-b2-errores><h4>${e(t("errores"))}</h4><ul>
        ${Object.keys(errores).map((c) => `<li><a href="#ct-b2-${c}">${e(t(c === "vacante" ? "puesto" : c))}: ${e(t(errores[c]))}</a></li>`).join("")}</ul></section>` : ""}
      <p class="ct-estado ${recibo ? "ct-estado-exito" : "ct-estado-informacion"}" role="status" aria-live="polite" tabindex="-1" data-b2-mensaje>${e(t(mensaje || (recibo ? "confirmada" : plan ? "plan_preparado" : fase === "revision" ? "revision_lista" : "paso_datos")))}</p>
      ${cuerpo}${!recibo && !controlador ? `<div class="ct-acciones"><button type="button" class="boton-secundario" data-b2-accion="consultar">${e(t(incierto ? "comprobar" : "actualizar"))}</button>
        ${incierto ? `<button type="button" class="boton-primario" data-b2-accion="retomar">${e(t("continuar"))}</button>` : ""}</div>` : ""}</div></section>`;
    if (foco) raiz.querySelector?.(foco)?.focus?.();
  }
  function aceptarConsulta(datos) {
    const c = validarConsultaB2(datos, expedienteRef);
    if (c.version_expediente_actual < versionEsperada || (!c.plan && c.version_expediente_actual !== versionEsperada)) throw Object.assign(new Error(), { estado: 409, envelopeValido: true });
    if (intencion && c.plan && JSON.stringify(c.plan.intencion) !== JSON.stringify(intencion)) throw Object.assign(new Error(), { estado: 409, envelopeValido: true });
    consulta = c; plan = c.plan; recibo = c.recibo;
    if (plan) { intencion = plan.intencion; fase = "revision"; }
    else if (!intencion) {
      const o = c.opciones;
      valores = { ...Object.fromEntries([["vacante", o.vacantes], ["regimen", o.regimenes], ["modalidad", o.modalidades], ["clase_ocupacion", o.clases_ocupacion], ["motivo", o.motivos], ["documento", o.documentos]].map(([k, lista]) => [k, k !== "clase_ocupacion" && lista.length === 1 ? "0" : ""])),
        desde: o.periodo.desde, hasta: o.periodo.hasta };
    }
    incierto = Boolean(intencion && !recibo && !plan); denegado = false;
    mensaje = recibo ? "confirmada" : incierto ? "registro_pendiente" : plan ? "plan_preparado" : "";
  }
  function fallo(error, efecto) {
    if (error?.envelopeValido && [401, 403].includes(error.estado)) {
      denegado = true; consulta = null; plan = null; recibo = null; valores = {};
      incierto = false; mensaje = "denegada"; return;
    }
    if (efecto) { incierto = true; mensaje = "registro_pendiente"; return; }
    mensaje = error?.envelopeValido && error.estado === 409 ? "conflicto" : "no_disponible";
    if (!intencion) consulta = null;
  }
  async function consultar() {
    if (!vigente() || controlador) return;
    const actual = new AbortController(); controlador = actual; mensaje = "cargando"; pintar("[data-b2-mensaje]");
    try { const c = await cliente.consultar(expedienteRef, { signal: actual.signal }); if (vigente() && !actual.signal.aborted) aceptarConsulta(c); }
    catch (error) { if (vigente() && !actual.signal.aborted) fallo(error, false); }
    finally { if (controlador === actual) { controlador = null; pintar("[data-b2-mensaje]"); } }
  }
  async function registrar(retomar = false) {
    if (!vigente() || controlador || recibo || denegado || !intencion || incierto && !retomar
      || !consulta?.prerrequisitos.length || !consulta.prerrequisitos.every((p) => p.cumplido)) return;
    try { resumen(plan?.intencion ?? intencion); } catch { mensaje = "no_disponible"; pintar("[data-b2-mensaje]"); return; }
    const actual = new AbortController(); controlador = actual; mensaje = plan ? "confirmando" : "preparando"; pintar("[data-b2-mensaje]");
    try {
      if (!plan) {
        const c = await cliente.preparar(intencion, { signal: actual.signal });
        if (!vigente() || actual.signal.aborted) return;
        aceptarConsulta(c);
        if (!plan) throw new Error("plan_b2_no_verificado");
      }
      if (!recibo) {
        mensaje = "confirmando"; pintar("[data-b2-mensaje]");
        const solicitud = validarSolicitudConfirmacionB2({ expediente_ref: expedienteRef, plan_ref: plan.plan_ref,
          version_plan: plan.version, clave_idempotencia: plan.intencion.clave_idempotencia });
        const r = await cliente.confirmar(solicitud, { signal: actual.signal });
        if (!vigente() || actual.signal.aborted) return;
        recibo = validarReciboB2(r, solicitud); incierto = false; mensaje = "confirmada";
      }
      try { await alConfirmar(recibo); } catch { /* El recibo validado se conserva si falla el refresco del detalle. */ }
    } catch (error) {
      if (!vigente() || actual.signal.aborted) return;
      fallo(error, true);
      if (!denegado) {
        try { const c = await cliente.consultar(expedienteRef, { signal: actual.signal }); if (vigente() && !actual.signal.aborted) aceptarConsulta(c); }
        catch (lectura) { if (vigente() && !actual.signal.aborted) fallo(lectura, false); }
        if (!recibo && !denegado) { incierto = true; mensaje = "registro_pendiente"; }
      }
    } finally { if (controlador === actual) { controlador = null; pintar("[data-b2-mensaje]"); } }
  }
  function enviar(evento) {
    if (!evento.target?.matches?.("[data-b2-form]")) return;
    evento.preventDefault();
    if (!vigente() || controlador || !puedePreparar() || fase !== "datos" || incierto) return;
    valores = recoger(evento.target); modificado = true; errores = erroresDe(valores);
    if (Object.keys(errores).length) { pintar("[data-b2-errores]"); return; }
    try { intencion = prepararIntencion(); fase = "revision"; mensaje = ""; pintar("[data-b2-titulo]"); }
    catch { mensaje = "rechazada"; pintar("[data-b2-mensaje]"); }
  }
  function click(evento) {
    const accion = evento.target?.closest?.("[data-b2-accion]")?.getAttribute?.("data-b2-accion");
    if (accion === "consultar") return consultar();
    if (accion === "registrar") return registrar();
    if (accion === "retomar") return registrar(true);
    if (accion === "cambiar" && !controlador && !plan && !incierto) { fase = "datos"; intencion = null; errores = {}; pintar("[data-b2-titulo]"); }
  }
  function blur(evento) {
    if (!vigente() || controlador || fase !== "datos" || !consulta || !campos.includes(evento.target?.name)) return;
    const c = evento.target.name;
    valores[c] = evento.target.value; modificado = true; const encontrados = erroresDe(valores);
    if (encontrados[c]) errores[c] = encontrados[c]; else delete errores[c];
    evento.target.setAttribute?.("aria-invalid", Boolean(errores[c]));
    const aviso = raiz.querySelector?.(`#ct-b2-error-${c}`); if (aviso) aviso.textContent = errores[c] ? t(errores[c]) : "";
  }
  function avisarSalida(evento) {
    if (vigente() && !recibo && (incierto || modificado)) { evento.preventDefault(); evento.returnValue = ""; }
  }
  raiz.addEventListener("submit", enviar); raiz.addEventListener("click", click); raiz.addEventListener("focusout", blur);
  entorno.addEventListener?.("beforeunload", avisarSalida);
  pintar(); void consultar();
  return () => {
    activo = false; controlador?.abort(); controlador = null;
    raiz.removeEventListener("submit", enviar); raiz.removeEventListener("click", click); raiz.removeEventListener("focusout", blur);
    entorno.removeEventListener?.("beforeunload", avisarSalida); raiz.replaceChildren();
    consulta = null; intencion = null; plan = null; recibo = null; valores = {};
  };
}
