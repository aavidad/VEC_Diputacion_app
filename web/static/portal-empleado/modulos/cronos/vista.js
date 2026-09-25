import {
  CAPACIDAD_CONSULTAR_FICHAJES,
  CAPACIDAD_CONSULTAR_HORARIO,
  exigirContextoActorCronos,
  tieneCapacidadCronos,
  validarCapacidadesCronos,
  validarDatosCronos,
} from "./contrato.js?v=20260925-tanda-v1";
import { crearTraductorCronos, MENSAJES_CRONOS_ES } from "./i18n.js?v=20260925-tanda-v1";
import { montarCalendarioCivilCronos } from "./vista-calendario.js?v=20260925-tanda-v1";

function escaparHTML(valor) {
  return String(valor ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

function instanteVisible(instante, locale, zonaHoraria) {
  const fecha = new Date(instante);
  if (!Number.isFinite(fecha.getTime())) throw new Error("instante de Cronos no válido");
  return {
    fecha: new Intl.DateTimeFormat(locale, {
      timeZone: zonaHoraria, day: "2-digit", month: "2-digit", year: "numeric",
    }).format(fecha),
    hora: new Intl.DateTimeFormat(locale, {
      timeZone: zonaHoraria, hour: "2-digit", minute: "2-digit", timeZoneName: "short",
    }).format(fecha),
    completo: new Intl.DateTimeFormat(locale, {
      timeZone: zonaHoraria, day: "2-digit", month: "2-digit", year: "numeric",
      hour: "2-digit", minute: "2-digit", timeZoneName: "short",
    }).format(fecha),
  };
}

function claseEstado(valor) {
  const estado = String(valor || "");
  if (new Set(["aprobado", "registrado", "revisado", "disfrutado", "calculado"]).has(estado)) return "exito";
  if (estado === "pendiente_responsable") return "aviso";
  return "neutro";
}

function chip(estado, t) {
  return `<span class="cronos-estado cronos-estado-${claseEstado(estado)}">${escaparHTML(t(`estado_${estado}`))}</span>`;
}

function tabla({ id, titulo, cabeceras, filas, vacio }) {
  return `
    <div class="cronos-tabla-contenedor">
      <table class="cronos-tabla" id="${escaparHTML(id)}">
        <caption>${escaparHTML(titulo)}</caption>
        <thead><tr>${cabeceras.map((cabecera) => `<th scope="col">${escaparHTML(cabecera)}</th>`).join("")}</tr></thead>
        <tbody>${filas.length
    ? filas.map((fila) => `<tr>${fila.map((celda, indice) => `<${indice === 0 ? "th scope=\"row\"" : "td"}>${celda}</${indice === 0 ? "th" : "td"}>`).join("")}</tr>`).join("")
    : `<tr><td colspan="${cabeceras.length}" class="cronos-vacio">${escaparHTML(vacio)}</td></tr>`}</tbody>
      </table>
    </div>`;
}

function indicador(etiqueta, valor, nota, tono = "informacion") {
  return `<article class="cronos-indicador cronos-indicador-${tono}">
    <span>${escaparHTML(etiqueta)}</span>
    <strong>${escaparHTML(valor)}</strong>
    <small>${escaparHTML(nota)}</small>
  </article>`;
}

function botonAyuda(t, asunto) {
  const etiqueta = t("abrir_ayuda", { asunto });
  return `<button type="button" class="cronos-boton-ayuda" data-accion="ayuda" aria-label="${escaparHTML(etiqueta)}" title="${escaparHTML(etiqueta)}"><span aria-hidden="true">?</span></button>`;
}

const ESTADOS_JORNADA = new Set(["cargando", "disponible", "vacio", "no_configurado", "denegado", "error"]);
const TIPOS_PERIODO_JORNADA = new Set(["dia", "semana", "mes", "anio", "periodo"]);

function fechaPeriodoValida(valor) {
  return typeof valor === "string" && /^\d{4}-\d{2}-\d{2}$/.test(valor)
    && Number.isFinite(Date.parse(`${valor}T12:00:00Z`))
    && new Date(`${valor}T12:00:00Z`).toISOString().slice(0, 10) === valor;
}

/** Valida solo el intervalo civil elegido; el saldo lo proporciona Cronos. */
export function validarSeleccionPeriodoCronos(seleccion) {
  if (!seleccion || !TIPOS_PERIODO_JORNADA.has(seleccion.tipo)
    || !fechaPeriodoValida(seleccion.desde)) throw new Error("tipo o fecha de periodo no válidos");
  if (seleccion.tipo === "periodo" && (!fechaPeriodoValida(seleccion.hasta) || seleccion.hasta < seleccion.desde)) {
    throw new Error("rango de periodo no válido");
  }
  return Object.freeze({
    tipo: seleccion.tipo,
    desde: seleccion.desde,
    ...(seleccion.tipo === "periodo" ? { hasta: seleccion.hasta } : {}),
  });
}

function etiquetaSeleccionPeriodo(seleccion, t, locale) {
  const fecha = (valor, opciones = { day: "2-digit", month: "2-digit", year: "numeric" }) =>
    new Intl.DateTimeFormat(locale, { timeZone: "UTC", ...opciones }).format(new Date(`${valor}T12:00:00Z`));
  if (seleccion.tipo === "dia") return t("jornada_periodo_dia_visible", { fecha: fecha(seleccion.desde) });
  if (seleccion.tipo === "mes") return t("jornada_periodo_mes_visible", {
    fecha: fecha(seleccion.desde, { month: "long", year: "numeric" }),
  });
  if (seleccion.tipo === "anio") return t("jornada_periodo_anio_visible", { anio: seleccion.desde.slice(0, 4) });
  if (seleccion.tipo === "periodo") return t("jornada_periodo_rango_visible", {
    desde: fecha(seleccion.desde), hasta: fecha(seleccion.hasta),
  });
  const origen = new Date(`${seleccion.desde}T12:00:00Z`);
  const lunes = new Date(origen);
  lunes.setUTCDate(origen.getUTCDate() - (origen.getUTCDay() + 6) % 7);
  const domingo = new Date(lunes);
  domingo.setUTCDate(lunes.getUTCDate() + 6);
  return t("jornada_periodo_semana_visible", {
    desde: fecha(lunes.toISOString().slice(0, 10)), hasta: fecha(domingo.toISOString().slice(0, 10)),
  });
}

function formularioPeriodoJornada(seleccion, t) {
  const tipo = seleccion?.tipo || "dia";
  return `<section class="panel cronos-jornada-panel cronos-jornada-periodo" aria-labelledby="cronos-jornada-periodo-titulo">
    <header class="cabecera-panel"><h4 id="cronos-jornada-periodo-titulo">${escaparHTML(t("jornada_periodo_titulo"))}</h4></header>
    <form class="cuerpo-panel cronos-jornada-periodo-formulario" data-cronos-form-periodo>
      <label>${escaparHTML(t("jornada_periodo_escala"))}<select name="tipo" data-cronos-periodo-tipo>
        ${["dia", "semana", "mes", "anio", "periodo"].map((opcion) => `<option value="${opcion}"${opcion === tipo ? " selected" : ""}>${escaparHTML(t(`jornada_periodo_${opcion}`))}</option>`).join("")}
      </select></label>
      <label>${escaparHTML(t("jornada_periodo_fecha"))}<input type="date" name="desde" required value="${escaparHTML(seleccion?.desde || "")}"></label>
      <label data-cronos-periodo-hasta${tipo === "periodo" ? "" : " hidden"}>${escaparHTML(t("jornada_periodo_hasta"))}<input type="date" name="hasta"${tipo === "periodo" ? " required" : ""} value="${escaparHTML(seleccion?.hasta || "")}"></label>
      <button type="submit" class="boton-secundario">${escaparHTML(t("jornada_periodo_consultar"))}</button>
      <p class="cronos-jornada-periodo-error" data-cronos-periodo-error role="alert" hidden></p>
    </form>
  </section>`;
}

function estadoJornadaConFuente(estado, contextoActor, datos) {
  if (estado !== "disponible") return estado;
  return contextoActor?.demostracion === false && datos && !Object.hasOwn(datos, "demostracion")
    ? "disponible" : "no_configurado";
}

function resolverEstadoJornada(estado, contextoActor, capacidades, datos, seleccion) {
  const conFuente = estadoJornadaConFuente(estado, contextoActor, datos);
  if (conFuente !== "disponible") return { estado: conFuente };
  const contexto = exigirContextoActorCronos(contextoActor);
  const concedidas = validarCapacidadesCronos(capacidades);
  const puedeConsultarFichajes = tieneCapacidadCronos(concedidas, CAPACIDAD_CONSULTAR_FICHAJES);
  const puedeConsultarHorario = tieneCapacidadCronos(concedidas, CAPACIDAD_CONSULTAR_HORARIO);
  if (!puedeConsultarFichajes && !puedeConsultarHorario) return { estado: "denegado" };
  return {
    estado: seleccion ? "no_configurado" : "disponible",
    contexto, puedeConsultarFichajes, puedeConsultarHorario,
  };
}

/**
 * Superficie de Jornada para el montaje del portal. El llamador solo puede
 * indicar "disponible" después de recibir una proyección propia de Cronos.
 * Esta vista no consulta servicios ni habilita un efecto de fichaje.
 */
export function renderizarJornadaCronos({
  estado = "no_configurado", contextoActor, capacidades = [], datos, seleccion,
  mensajes = MENSAJES_CRONOS_ES, locale = "es-ES", zonaHoraria = "Europe/Madrid",
} = {}) {
  if (!ESTADOS_JORNADA.has(estado)) throw new Error("estado de jornada de Cronos no válido");
  if (seleccion) seleccion = validarSeleccionPeriodoCronos(seleccion);
  const resolucion = resolverEstadoJornada(estado, contextoActor, capacidades, datos, seleccion);
  estado = resolucion.estado;
  const t = crearTraductorCronos(mensajes);
  let vista;
  const puedeConsultarFichajes = estado === "disponible" && resolucion.puedeConsultarFichajes;
  const puedeConsultarHorario = estado === "disponible" && resolucion.puedeConsultarHorario;
  if (estado === "disponible") {
    vista = validarDatosCronos(datos, resolucion.contexto);
  }
  const estadoVisible = t(`jornada_estado_${estado}`);
  const descripcionEstado = t(`jornada_descripcion_${estado}`);
  const resumen = vista?.resumen;
  const perfil = vista?.perfil_jornada;
  const fichajes = puedeConsultarFichajes ? vista.fichajes : [];
  const filas = fichajes.slice(0, 12).map((item) => {
    const instante = instanteVisible(item.instante, locale, zonaHoraria);
    return [
      escaparHTML(instante.fecha), escaparHTML(instante.hora),
      escaparHTML(t(`movimiento_${item.tipo_clave}`)), escaparHTML(item.canal),
      chip(item.estado_clave, t),
    ];
  });
  const saldos = estado === "disponible" ? `<div class="cronos-jornada-indicadores rejilla-kpi" aria-label="${escaparHTML(t("resumen_etiqueta"))}">
    ${indicador(t("indicador_jornada"), puedeConsultarHorario ? resumen.teoricas_hoy : "—", puedeConsultarHorario ? vista.periodo : t("sin_permiso"))}
    ${indicador(t("indicador_trabajado"), puedeConsultarFichajes ? resumen.trabajadas_hoy : "—", puedeConsultarFichajes ? t("nota_movimientos") : t("sin_permiso"), puedeConsultarFichajes ? "exito" : "informacion")}
    ${indicador(t("indicador_saldo_dia"), puedeConsultarFichajes ? resumen.saldo_hoy : "—", puedeConsultarFichajes ? t("nota_calculo") : t("sin_permiso"), puedeConsultarFichajes ? "aviso" : "informacion")}
    ${indicador(t("indicador_saldo_periodo"), puedeConsultarFichajes ? resumen.saldo_periodo : "—", puedeConsultarFichajes ? t("nota_acumulado") : t("sin_permiso"))}
  </div>` : "";
  const contenido = seleccion && ["no_configurado", "vacio"].includes(estado) ? `<section class="panel cronos-jornada-panel" aria-labelledby="cronos-jornada-seleccion-titulo"><header class="cabecera-panel"><h4 id="cronos-jornada-seleccion-titulo">${escaparHTML(etiquetaSeleccionPeriodo(seleccion, t, locale))}</h4></header><div class="cuerpo-panel cronos-jornada-seleccion-estado" role="status" data-cronos-periodo-resultado tabindex="-1"><p>${escaparHTML(t("jornada_periodo_sin_proyeccion"))}</p><dl>${["teorica", "trabajado", "permisos", "saldo"].map((clave) => `<div><dt>${escaparHTML(t(`jornada_periodo_desglose_${clave}`))}</dt><dd>${escaparHTML(t("jornada_periodo_dato_pendiente"))}</dd></div>`).join("")}</dl></div></section>` : estado === "disponible" ? `<div class="cronos-jornada-rejilla">
    <section class="panel cronos-jornada-panel" aria-labelledby="cronos-jornada-movimientos">
      <header class="cabecera-panel"><h4 id="cronos-jornada-movimientos">${escaparHTML(t("jornada_movimientos"))}</h4>${botonAyuda(t, t("jornada_movimientos"))}</header>
      ${puedeConsultarFichajes ? tabla({ id: "tabla-cronos-jornada", titulo: t("jornada_tabla"), cabeceras: [t("cab_fecha"), t("cab_hora"), t("cab_movimiento"), t("cab_canal"), t("cab_estado")], filas, vacio: t("fichajes_vacio") }) : `<p class="cronos-acceso-denegado" role="status">${escaparHTML(t("fichajes_denegado"))}</p>`}
    </section>
    <div class="cronos-jornada-lateral">
      <aside class="panel cronos-jornada-panel" aria-labelledby="cronos-jornada-perfil">
        <header class="cabecera-panel"><h4 id="cronos-jornada-perfil">${escaparHTML(t("horario_titulo"))}</h4>${botonAyuda(t, t("horario_titulo"))}</header>
        ${puedeConsultarHorario ? `<dl class="cronos-resumen-datos"><div><dt>${escaparHTML(t("perfil"))}</dt><dd>${escaparHTML(perfil.nombre)}</dd></div><div><dt>${escaparHTML(t("jornada_diaria"))}</dt><dd>${escaparHTML(perfil.jornada_diaria)}</dd></div><div><dt>${escaparHTML(t("ventana_entrada"))}</dt><dd>${escaparHTML(perfil.ventana_entrada)}</dd></div><div><dt>${escaparHTML(t("tramo_obligatorio"))}</dt><dd>${escaparHTML(perfil.tramo_obligatorio)}</dd></div></dl>` : `<p class="cronos-acceso-denegado" role="status">${escaparHTML(t("horario_denegado"))}</p>`}
      </aside>
      <aside class="panel cronos-jornada-panel" aria-labelledby="cronos-jornada-calendario">
        <header class="cabecera-panel"><h4 id="cronos-jornada-calendario">${escaparHTML(t("jornada_calendario_titulo"))}</h4>${botonAyuda(t, t("jornada_calendario_titulo"))}</header>
        <div class="cuerpo-panel cronos-jornada-calendario-pendiente" role="status"><span class="cronos-estado cronos-estado-aviso">${escaparHTML(t("jornada_estado_no_configurado"))}</span><p>${escaparHTML(t("jornada_calendario_pendiente"))}</p></div>
      </aside>
    </div>
  </div>` : `<section class="panel cronos-jornada-panel" aria-labelledby="cronos-jornada-sin-datos"><header class="cabecera-panel"><h4 id="cronos-jornada-sin-datos">${escaparHTML(estadoVisible)}</h4></header><div class="cuerpo-panel" role="status" data-cronos-jornada-resultado tabindex="-1">${escaparHTML(descripcionEstado)}</div></section>`;

  return `<section class="cronos-jornada cronos-area" data-cronos-jornada data-estado="${estado}" aria-labelledby="cronos-jornada-titulo"${estado === "cargando" ? ' aria-busy="true"' : ""}>
    <header class="cronos-jornada-encabezado"><div><p class="sobrelinea">${escaparHTML(t("sobrelinea"))}</p><h3 id="cronos-jornada-titulo">${escaparHTML(t("jornada_titulo"))}</h3></div><span class="cronos-estado cronos-estado-${estado === "disponible" ? "exito" : "aviso"}" role="status">${escaparHTML(estadoVisible)}</span></header>
    ${estado === "disponible" ? `<p class="cronos-jornada-fuente">${escaparHTML(t("jornada_fuente_servicio", { fecha: instanteVisible(vista.actualizado_en, locale, zonaHoraria).completo }))}</p>` : ""}
    ${estado === "denegado" ? "" : formularioPeriodoJornada(seleccion, t)}
    ${saldos}${contenido}
    <div data-cronos-calendario-raiz></div>
  </section>`;
}
/** Montaje sin efectos ni listeners; el coordinador conserva la consulta y el desmontaje. */
export function montarJornadaCronos({ raiz, registrarDesmontar, anunciar = () => {}, ...proyeccion } = {}) {
  if (!raiz?.append || !raiz.ownerDocument?.createElement
    || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")
    || typeof anunciar !== "function") throw new TypeError("raíz de jornada de Cronos no válida");
  const contenedor = raiz.ownerDocument.createElement("div");
  contenedor.dataset.cronosJornadaMontaje = "";
  contenedor.innerHTML = renderizarJornadaCronos(proyeccion);
  raiz.append(contenedor);
  let activo = true;
  let calendario;
  let seleccion;
  const t = crearTraductorCronos(proyeccion.mensajes);
  const cambiarEscala = (evento) => {
    if (!evento.target?.matches?.("[data-cronos-periodo-tipo]")) return;
    const form = evento.target.closest("[data-cronos-form-periodo]");
    const campoHasta = form?.querySelector("[data-cronos-periodo-hasta]");
    if (!campoHasta) return;
    const esRango = evento.target.value === "periodo";
    campoHasta.hidden = !esRango;
    const entradaHasta = campoHasta.querySelector("input");
    if (entradaHasta) entradaHasta.required = esRango;
  };
  const elegirPeriodo = (evento) => {
    if (!evento.target?.matches?.("[data-cronos-form-periodo]")) return;
    evento.preventDefault();
    const form = evento.target;
    const valor = (nombre) => form.elements.namedItem(nombre)?.value || "";
    let nuevaSeleccion;
    try {
      nuevaSeleccion = validarSeleccionPeriodoCronos({ tipo: valor("tipo"), desde: valor("desde"), hasta: valor("hasta") });
    } catch {
      const error = form.querySelector("[data-cronos-periodo-error]");
      if (error) { error.textContent = t("jornada_periodo_error"); error.hidden = false; }
      form.elements.namedItem(valor("tipo") === "periodo" && valor("hasta") < valor("desde") ? "hasta" : "desde")?.focus?.();
      anunciar(t("jornada_periodo_error"));
      return;
    }
    seleccion = nuevaSeleccion;
    actualizar(proyeccion);
    contenedor.querySelector?.("[data-cronos-periodo-resultado], [data-cronos-jornada-resultado]")?.focus?.();
    if (resolverEstadoJornada(
      proyeccion.estado ?? "no_configurado", proyeccion.contextoActor,
      proyeccion.capacidades ?? [], proyeccion.datos, seleccion,
    ).estado !== "denegado") {
      anunciar(etiquetaSeleccionPeriodo(seleccion, t, proyeccion.locale || "es-ES"));
    }
  };
  contenedor.addEventListener?.("change", cambiarEscala);
  contenedor.addEventListener?.("submit", elegirPeriodo);
  const montarCalendario = () => {
    const destino = contenedor.querySelector?.("[data-cronos-calendario-raiz]");
    if (destino) calendario = montarCalendarioCivilCronos({ raiz: destino, anunciar });
  };
  montarCalendario();
  const actualizar = (siguiente) => {
    if (!activo) throw new Error("jornada de Cronos desmontada");
    const focoDentro = contenedor.contains?.(raiz.ownerDocument.activeElement) ?? false;
    proyeccion = siguiente;
    const html = renderizarJornadaCronos({ ...siguiente, seleccion });
    calendario?.desmontar();
    calendario = undefined;
    contenedor.innerHTML = html;
    montarCalendario();
    if (focoDentro) contenedor.querySelector?.("[data-cronos-periodo-resultado], [data-cronos-jornada-resultado]")?.focus?.();
    const estadoVisible = resolverEstadoJornada(
      siguiente?.estado ?? "no_configurado", siguiente?.contextoActor,
      siguiente?.capacidades ?? [], siguiente?.datos, seleccion,
    ).estado;
    anunciar(crearTraductorCronos(siguiente?.mensajes)(`jornada_estado_${estadoVisible}`));
  };
  const desmontar = () => {
    if (!activo) return;
    activo = false;
    calendario?.desmontar();
    contenedor.removeEventListener?.("change", cambiarEscala);
    contenedor.removeEventListener?.("submit", elegirPeriodo);
    contenedor.remove();
  };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ actualizar, desmontar });
}
