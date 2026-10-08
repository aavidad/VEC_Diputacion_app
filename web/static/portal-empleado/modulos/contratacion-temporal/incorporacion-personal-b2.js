import { validarConsultaB2, validarSolicitudPlanB2, validarSolicitudConfirmacionB2, validarReciboB2, fechaCivilB2 } from "./contrato-incorporacion-personal-b2.js";
import { escaparHTML as e } from "./componentes-expedientes.js?v=20261008-r-fichas-idioma-nav-v1";
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
  let activo = true, controlador = null, consulta = null, opcionesGuardadas = null, recibo = null, fase = "datos", mensaje = "cargando";
  let intencion = null, plan = null, incierto = false, denegado = false, errores = {}, valores = {}, modificado = false;
  let intencionEnviada = false, catalogoRevision = null, reintentoBloqueado = false;
  const seleccionesPendientes = new Set();
  const vigente = () => activo && esVigente() && raiz.isConnected !== false;
  const fila = (clave, valor) => `<div><dt>${e(t(clave))}</dt><dd>${e(valor)}</dd></div>`;
  const fecha = (v) => textos.fecha(`${v}T00:00:00Z`, { dateStyle: "medium", timeZone: "UTC" });
  const puedePreparar = () => consulta && consulta.version_expediente_actual === versionEsperada
    && consulta.prerrequisitos.length > 0 && consulta.prerrequisitos.every((p) => p.cumplido)
    && ["vacantes", "regimenes", "modalidades", "clases_ocupacion", "motivos", "documentos"].every((k) => consulta.opciones[k].length > 0);
  const campos = ["vacante", "regimen", "modalidad", "clase_ocupacion", "desde", "hasta", "motivo", "documento"];
  const valorOpcion = (lista, v) => typeof v === "string" && /^(?:0|[1-9]\d*)$/u.test(v) ? lista[Number(v)] : undefined;
  const opcionesSeleccionables = (o) => ({ vacante: o.vacantes, regimen: o.regimenes, modalidad: o.modalidades,
    clase_ocupacion: o.clases_ocupacion, motivo: o.motivos, documento: o.documentos });
  const identidadOpcion = (campo, opcion) => {
    if (campo === "vacante") return [opcion.plaza_ref, opcion.puesto_ref, opcion.version_plantilla_ref, opcion.version_rpt_ref].join("|");
    if (campo === "regimen" || campo === "modalidad") return `${opcion.ref}|${opcion.version}`;
    if (campo === "documento") return `${opcion.documento_ref}|${opcion.documento_sha256}`;
    return campo === "clase_ocupacion" ? opcion.valor : opcion;
  };
  const mismoCatalogoClases = (a, b) => a && b && a.ref === b.ref && a.version === b.version
    && a.huella_sha256 === b.huella_sha256;
  function seleccionCompatible(c, s) {
    if (!c || !s || c.version_expediente_actual !== s.version_expediente
      || !mismoCatalogoClases(catalogoRevision, c.opciones.catalogo_clases_ocupacion)) return false;
    const o = c.opciones;
    return o.vacantes.some((v) => v.puesto_ref === s.puesto_ref && v.plaza_ref === s.plaza_ref
      && v.version_plantilla_ref === s.version_plantilla_ref && v.version_rpt_ref === s.version_rpt_ref)
      && o.regimenes.some((v) => v.ref === s.regimen.ref && v.version === s.regimen.version)
      && o.modalidades.some((v) => v.ref === s.modalidad.ref && v.version === s.modalidad.version)
      && o.clases_ocupacion.some((v) => v.valor === s.clase_ocupacion)
      && o.motivos.includes(s.motivo_clave)
      && o.documentos.some((v) => v.documento_ref === s.documento_ref && v.documento_sha256 === s.documento_sha256)
      && (!o.periodo.fuente_ref || s.desde === o.periodo.desde && s.hasta === o.periodo.hasta);
  }
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
    const vacante = o.vacantes.find((x) => x.puesto_ref === s.puesto_ref && x.plaza_ref === s.plaza_ref
      && x.version_plantilla_ref === s.version_plantilla_ref && x.version_rpt_ref === s.version_rpt_ref);
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
    if (lista.length === 1 && !seleccionExpresa && !seleccionesPendientes.has(campo)) return `<div class="ct-campo"><span>${e(t(clave))}</span><span class="ct-b2-valor-solo-lectura">${e(rotulo(lista[0]))}</span></div>`;
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
        if (fase === "revision" && (intencion || plan) && !reintentoBloqueado) {
          cuerpo += `<section class="ct-revision"><h4>${e(t("paso_revision"))}</h4>${resumen(plan?.intencion ?? intencion)}
            <p>${e(t(plan ? "plan_preparado" : "limite"))}</p><div class="ct-acciones">
            ${!plan && !incierto ? `<button type="button" class="boton-secundario" data-b2-accion="cambiar">${e(t("cambiar"))}</button>` : ""}
            ${incierto ? "" : `<button type="button" class="boton-primario" data-b2-accion="registrar"${controlador || !consulta.prerrequisitos.length || !consulta.prerrequisitos.every((p) => p.cumplido) ? " disabled" : ""}>${e(t(plan ? "continuar" : "registrar"))}</button>`}</div></section>`;
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
        ${incierto && consulta && !reintentoBloqueado ? `<button type="button" class="boton-primario" data-b2-accion="retomar">${e(t("continuar"))}</button>` : ""}</div>` : ""}</div></section>`;
    if (foco) raiz.querySelector?.(foco)?.focus?.();
  }
  function aceptarConsulta(datos) {
    const c = validarConsultaB2(datos, expedienteRef);
    if (c.version_expediente_actual < versionEsperada || (!c.plan && c.version_expediente_actual !== versionEsperada)) throw Object.assign(new Error(), { estado: 409, envelopeValido: true });
    if (intencion && c.plan && JSON.stringify(c.plan.intencion) !== JSON.stringify(intencion)) throw Object.assign(new Error(), { estado: 409, envelopeValido: true });
    const seleccionCambiada = Boolean(intencion && !c.plan && !seleccionCompatible(c, intencion));
    if (seleccionCambiada && !intencionEnviada) { intencion = null; catalogoRevision = null; fase = "datos"; }
    reintentoBloqueado = seleccionCambiada && intencionEnviada;
    const anteriores = opcionesGuardadas;
    const seleccionAnterior = { ...valores };
    consulta = c; opcionesGuardadas = c.opciones; plan = c.plan; recibo = c.recibo;
    if (plan) { intencion = plan.intencion; fase = "revision"; errores = {}; }
    else if (!intencion || !intencionEnviada) {
      const o = c.opciones;
      errores = {};
      valores = Object.fromEntries(Object.entries(opcionesSeleccionables(o)).map(([campo, lista]) => {
        const anterior = anteriores && valorOpcion(opcionesSeleccionables(anteriores)[campo], seleccionAnterior[campo]);
        if (!anterior) {
          if (seleccionesPendientes.has(campo)) errores[campo] = "opcion_ya_no_disponible";
          return [campo, !seleccionesPendientes.has(campo) && campo !== "clase_ocupacion" && lista.length === 1 ? "0" : ""];
        }
        const clave = identidadOpcion(campo, anterior);
        const indice = lista.findIndex((opcion) => identidadOpcion(campo, opcion) === clave);
        const mismoCatalogo = campo !== "clase_ocupacion" || mismoCatalogoClases(anteriores.catalogo_clases_ocupacion,
          o.catalogo_clases_ocupacion);
        if (indice >= 0 && mismoCatalogo) return [campo, String(indice)];
        seleccionesPendientes.add(campo);
        errores[campo] = "opcion_ya_no_disponible";
        return [campo, ""];
      }));
      const periodoFijo = Boolean(o.periodo.fuente_ref && o.periodo.desde);
      for (const campo of ["desde", "hasta"]) {
        const fechaEditada = anteriores && seleccionAnterior[campo] !== anteriores.periodo[campo];
        valores[campo] = !periodoFijo && fechaEditada ? seleccionAnterior[campo] : o.periodo[campo];
      }
    }
    incierto = Boolean(intencionEnviada && intencion && !recibo && !plan); denegado = false;
    mensaje = recibo ? "confirmada" : reintentoBloqueado ? "seleccion_obsoleta"
      : seleccionCambiada ? "conflicto" : incierto ? "registro_pendiente" : plan ? "plan_preparado" : "";
  }
  function fallo(error, efecto, lecturaIndependiente = false) {
    if (error?.envelopeValido && [401, 403].includes(error.estado)) {
      denegado = true; consulta = null; opcionesGuardadas = null; plan = null; recibo = null; intencion = null;
      valores = {}; errores = {}; fase = "datos"; incierto = false; modificado = false;
      intencionEnviada = false; catalogoRevision = null; reintentoBloqueado = false; seleccionesPendientes.clear();
      mensaje = "denegada"; return;
    }
    if (efecto) { incierto = true; mensaje = "registro_pendiente"; return; }
    mensaje = error?.envelopeValido && error.estado === 409 ? "conflicto" : "no_disponible";
    // Una consulta independiente fallida invalida opciones y versión previas.
    // Se conserva la intención y su clave para recuperarla con otro GET.
    if (lecturaIndependiente || !intencion) consulta = null;
  }
  async function consultar() {
    if (!vigente() || controlador) return;
    if (fase === "datos" && consulta && !intencion) {
      const form = raiz.querySelector?.("[data-b2-form]");
      if (form?.elements) {
        const editados = recoger(form);
        if (campos.some((campo) => editados[campo] !== valores[campo])) modificado = true;
        valores = editados;
      }
    }
    const actual = new AbortController(); controlador = actual; mensaje = "cargando"; pintar("[data-b2-mensaje]");
    try { const c = await cliente.consultar(expedienteRef, { signal: actual.signal }); if (vigente() && !actual.signal.aborted) aceptarConsulta(c); }
    catch (error) { if (vigente() && !actual.signal.aborted) fallo(error, false, true); }
    finally { if (controlador === actual) { controlador = null; pintar("[data-b2-mensaje]"); } }
  }
  async function registrar(retomar = false) {
    if (!vigente() || controlador || recibo || denegado || !intencion || reintentoBloqueado
      || !plan && !seleccionCompatible(consulta, intencion) || incierto && !retomar
      || !consulta?.prerrequisitos.length || !consulta.prerrequisitos.every((p) => p.cumplido)) return;
    try { resumen(plan?.intencion ?? intencion); } catch { mensaje = "no_disponible"; pintar("[data-b2-mensaje]"); return; }
    const actual = new AbortController(); controlador = actual; mensaje = plan ? "confirmando" : "preparando"; pintar("[data-b2-mensaje]");
    try {
      if (!plan) {
        intencionEnviada = true;
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
        catch (lectura) { if (vigente() && !actual.signal.aborted) fallo(lectura, false, true); }
        if (!recibo && !denegado) { incierto = true; mensaje = reintentoBloqueado ? "seleccion_obsoleta" : "registro_pendiente"; }
      }
    } finally { if (controlador === actual) { controlador = null; pintar("[data-b2-mensaje]"); } }
  }
  function enviar(evento) {
    if (!evento.target?.matches?.("[data-b2-form]")) return;
    evento.preventDefault();
    if (!vigente() || controlador || !puedePreparar() || fase !== "datos" || incierto) return;
    valores = recoger(evento.target); modificado = true; errores = erroresDe(valores);
    for (const campo of seleccionesPendientes) {
      if (valorOpcion(opcionesSeleccionables(consulta.opciones)[campo], valores[campo])) seleccionesPendientes.delete(campo);
    }
    if (Object.keys(errores).length) { pintar("[data-b2-errores]"); return; }
    try { intencion = prepararIntencion(); catalogoRevision = consulta.opciones.catalogo_clases_ocupacion;
      intencionEnviada = false; fase = "revision"; mensaje = ""; pintar("[data-b2-titulo]"); }
    catch { mensaje = "rechazada"; pintar("[data-b2-mensaje]"); }
  }
  function click(evento) {
    const accion = evento.target?.closest?.("[data-b2-accion]")?.getAttribute?.("data-b2-accion");
    if (accion === "consultar") return consultar();
    if (accion === "registrar") return registrar();
    if (accion === "retomar") return registrar(true);
    if (accion === "cambiar" && !controlador && !plan && !incierto) {
      fase = "datos"; intencion = null; catalogoRevision = null; errores = {}; pintar("[data-b2-titulo]");
    }
  }
  function blur(evento) {
    if (!vigente() || controlador || fase !== "datos" || !consulta || !campos.includes(evento.target?.name)) return;
    const c = evento.target.name;
    valores[c] = evento.target.value; modificado = true; const encontrados = erroresDe(valores);
    if (!encontrados[c]) seleccionesPendientes.delete(c);
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
    consulta = null; opcionesGuardadas = null; intencion = null; plan = null; recibo = null; valores = {};
    seleccionesPendientes.clear();
  };
}
