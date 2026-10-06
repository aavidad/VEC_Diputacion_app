/**
 * Catálogo del Portal del Empleado y portada de RRHH.
 *
 * Esta vista no conoce adaptadores ni datos de negocio. La composición decide
 * qué módulos están disponibles para el ContextoActor activo. Solo se ofrecen
 * los disponibles y los que aún se comprueban: un módulo sin acceso para este
 * perfil, o sin servicio, no aparece en lugar de mostrar una tarjeta vacía.
 *
 * La portada de RRHH es un cuadro de mandos: primero los expedientes que
 * piden atención (plazo de fase vencido o que vence hoy, o incidencia), después
 * los indicadores y el reparto por fase. Cuenta con los mismos criterios que la
 * lista (recuentos-peticiones.js) y no deduce responsables ni tareas.
 */
import { finVigenciaBolsaPortal, traducirPortal } from "./portal-i18n.js?v=20261001-ct-a-i18n-v1";
import { faseRRHH, FASES_RRHH } from "./modulos/contratacion-temporal/i18n-fases-rrhh.js?v=20261001-ct-a-i18n-v1";
import { resumirPeticiones } from "./modulos/contratacion-temporal/recuentos-peticiones.js?v=20261001-f-reconciliacion-325-v1";
import { icono } from "../comun/iconos-vec.js?v=20260925-aspecto-v1";
import { IDIOMA_ACTUAL } from "../comun/idioma.js";
import { renderizarAccesosEmpleado } from "./portal-accesos-empleado.js?v=20261001-g364-reconciliar-v2";

const DESTINO_LISTA = 'data-vista="contratacion-temporal" data-ct-exp-vista="cuadro"';

function faseConOrden(claveOrigen, traducir) {
  const fase = faseRRHH(claveOrigen);
  return fase ? traducir("tramite_fase_de_nombre", {
    orden: fase.orden, total: fase.total, fase: traducir(`tramite_fase_${fase.clave}`),
  }) : "";
}

function diaYMes(dia, locale) {
  const fecha = new Date(`${dia}T00:00:00Z`);
  if (!Number.isFinite(fecha.getTime())) return null;
  const partes = (opciones) => new Intl.DateTimeFormat(locale, { ...opciones, timeZone: "UTC" }).format(fecha);
  return { dia: partes({ day: "2-digit" }), mes: partes({ month: "short" }).replace(".", "") };
}

// Un expediente que pide atención: fecha del plazo, número, categoría, fase y
// motivo (vencido, vence hoy o incidencia), con un único botón para abrirlo.
function renderizarPendiente(expediente, escaparHTML, traducir, locale) {
  const t = (clave, variables) => escaparHTML(traducir(clave, variables));
  const fecha = diaYMes(expediente.plazo_ultimo_dia ?? "", locale);
  const tono = expediente.plazo_estado === "vencido" ? "vencido"
    : (expediente.plazo_estado === "vence_hoy" ? "hoy" : (fecha ? "" : "sin-fecha"));
  const motivos = [
    faseConOrden(expediente.fase_clave, traducir),
    expediente.plazo_estado === "vencido" ? traducir("inicio_rrhh_plazo_vencido", { fecha: expediente.plazo ?? "" }) : "",
    expediente.plazo_estado === "vence_hoy" ? traducir("inicio_rrhh_plazo_hoy") : "",
    expediente.estado_clave === "incidencia" ? traducir("tramite_estado_incidencia") : "",
  ].filter(Boolean);
  return `<li>
    <span class="fecha-tarea${tono ? ` ${tono}` : ""}" aria-hidden="true">${fecha
    ? `<strong>${escaparHTML(fecha.dia)}</strong>${escaparHTML(fecha.mes)}` : "—"}</span>
    <div>
      <h3>${escaparHTML(expediente.numero_visible ?? "")} · ${escaparHTML(expediente.categoria ?? "—")}</h3>
      <p>${escaparHTML(expediente.centro ?? "—")} · ${escaparHTML(motivos.join(" · "))}</p>
    </div>
    <button type="button" class="boton-secundario" data-vista="contratacion-temporal" data-ct-exp-abrir-inicio="${escaparHTML(expediente.expediente_ref ?? "")}"
      aria-label="${t("inicio_rrhh_abrir_expediente_aria", { numero: expediente.numero_visible ?? "" })}">${t("inicio_rrhh_abrir_expediente")}</button>
  </li>`;
}

// Expedientes idénticos a la vista (misma categoría, centro, fase, estado y
// plazo) se agrupan en una sola fila con contador; los números siguen visibles.
function claveGrupoPendiente(e) {
  return JSON.stringify([e.categoria ?? "", e.centro ?? "", e.fase_clave ?? "", e.estado_clave ?? "",
    e.plazo_estado ?? "", e.plazo_ultimo_dia ?? "", e.plazo ?? ""]);
}

export function agruparPendientes(atencion) {
  const grupos = new Map();
  for (const expediente of atencion) {
    const clave = claveGrupoPendiente(expediente);
    if (!grupos.has(clave)) grupos.set(clave, []);
    grupos.get(clave).push(expediente);
  }
  return [...grupos.values()];
}

function renderizarGrupoPendiente(grupo, escaparHTML, traducir, locale) {
  if (grupo.length === 1) return renderizarPendiente(grupo[0], escaparHTML, traducir, locale);
  const t = (clave, variables) => escaparHTML(traducir(clave, variables));
  const muestra = grupo[0];
  const fecha = diaYMes(muestra.plazo_ultimo_dia ?? "", locale);
  const tono = muestra.plazo_estado === "vencido" ? "vencido"
    : (muestra.plazo_estado === "vence_hoy" ? "hoy" : (fecha ? "" : "sin-fecha"));
  const motivos = [
    faseConOrden(muestra.fase_clave, traducir),
    muestra.plazo_estado === "vencido" ? traducir("inicio_rrhh_grupo_vencidas", { fecha: muestra.plazo ?? "" }) : "",
    muestra.plazo_estado === "vence_hoy" ? traducir("inicio_rrhh_plazo_hoy") : "",
    muestra.estado_clave === "incidencia" ? traducir("tramite_estado_incidencia") : "",
  ].filter(Boolean);
  const numeros = grupo.map((e) => e.numero_visible ?? "").filter(Boolean).join(", ");
  return `<li data-grupo-pendientes="${grupo.length}">
    <span class="fecha-tarea${tono ? ` ${tono}` : ""}" aria-hidden="true">${fecha
    ? `<strong>${escaparHTML(fecha.dia)}</strong>${escaparHTML(fecha.mes)}` : "—"}</span>
    <div>
      <h3>${t("inicio_rrhh_grupo_peticiones", { total: grupo.length })} · ${escaparHTML(muestra.categoria ?? "—")}</h3>
      <p>${escaparHTML(muestra.centro ?? "—")} · ${escaparHTML(motivos.join(" · "))}</p>
      ${numeros ? `<p><small>${t("inicio_rrhh_grupo_numeros", { numeros })}</small></p>` : ""}
    </div>
    <button type="button" class="boton-secundario" ${DESTINO_LISTA}
      aria-label="${t("inicio_rrhh_grupo_ver_aria", { total: grupo.length, categoria: muestra.categoria ?? "", centro: muestra.centro ?? "" })}">${t("inicio_rrhh_grupo_ver")}</button>
  </li>`;
}

function renderizarPendientes(resumen, escaparHTML, traducir, locale) {
  const t = (clave, variables) => escaparHTML(traducir(clave, variables));
  const total = resumen.atencion.length;
  const titulo = total === 0 ? t("inicio_rrhh_pendientes_ninguno")
    : (total === 1 ? t("inicio_rrhh_pendientes_uno") : t("inicio_rrhh_pendientes_varios", { total }));
  return `<section class="panel portal-rrhh-pendientes" aria-labelledby="inicio-rrhh-pendientes-titulo">
    <div class="cabecera-panel"><h3 id="inicio-rrhh-pendientes-titulo">${titulo}</h3>
      <button type="button" class="boton-terciario" ${DESTINO_LISTA} data-ct-exp-lista-mostrar="vencidos">${t("inicio_rrhh_plazos_vencidos", { total: resumen.vencidos })}</button>
      <button type="button" class="boton-terciario" ${DESTINO_LISTA}>${t("inicio_rrhh_ver_peticiones")} →</button></div>
    ${resumen.parcial ? `<p class="portal-rrhh-parcial" role="status">${t("inicio_rrhh_recuento_parcial")}</p>` : ""}
    ${total === 0 ? `<p class="portal-rrhh-resumen-vacio">${t("inicio_rrhh_pendientes_vacio")}</p>`
    : `<ol class="tareas-pendientes">${agruparPendientes(resumen.atencion).map((g) => renderizarGrupoPendiente(g, escaparHTML, traducir, locale)).join("")}</ol>`}
  </section>`;
}

// Indicador que lleva a una lista: icono, valor y rótulo. Sin dato: «—».
function renderizarIndicador({ clave, iconoNombre, tono, valor, etiqueta, destino, escaparHTML, traducir }) {
  const contenido = `<span class="icono-kpi">${icono(iconoNombre)}</span>
    <div><strong class="valor-kpi">${valor === null ? "—" : escaparHTML(valor)}</strong>
    <span class="etiqueta-kpi">${escaparHTML(etiqueta)}</span>
    ${destino && valor !== null ? `<span class="metrica-enlace" aria-hidden="true">${escaparHTML(traducir("inicio_rrhh_kpi_ver"))} →</span>` : ""}</div>`;
  const clase = `tarjeta-kpi${tono ? ` kpi--${tono}` : ""}`;
  return destino && valor !== null
    ? `<button type="button" class="${clase}" data-metrica="${clave}" ${destino}>${contenido}</button>`
    : `<div class="${clase}" data-metrica="${clave}">${contenido}</div>`;
}

function renderizarPorFase(resumen, escaparHTML, traducir, numero) {
  const t = (clave, variables) => escaparHTML(traducir(clave, variables));
  const filas = FASES_RRHH.map((fase, indice) => {
    const nombre = traducir(`tramite_fase_${fase}`);
    const total = resumen.porFase[fase] ?? 0;
    return `<tr>
      <th scope="row"><button type="button" class="enlace-tabla" ${DESTINO_LISTA} data-ct-exp-lista-fase="${fase}"
        aria-label="${t("inicio_rrhh_por_fase_aria", { total, fase: nombre })}">${indice + 1}. ${escaparHTML(nombre)}</button></th>
      <td class="numero">${escaparHTML(numero(total))}</td>
    </tr>`;
  }).join("");
  return `<section class="panel" aria-labelledby="inicio-rrhh-por-fase">
    <div class="cabecera-panel"><h3 id="inicio-rrhh-por-fase">${t("inicio_rrhh_por_fase")}</h3></div>
    <div class="portal-rrhh-tabla"><table class="tabla-datos">
      <thead><tr><th scope="col">${t("inicio_rrhh_col_fase")}</th><th scope="col" class="numero">${t("inicio_rrhh_col_peticiones")}</th></tr></thead>
      <tbody>${filas}</tbody>
    </table></div>
  </section>`;
}

const MAXIMO_BOLSAS_INICIO = 8;

function vigenciaBolsaEn(bolsa, generadoEn) {
  const instante = Date.parse(generadoEn);
  const desde = Date.parse(bolsa?.vigente_desde);
  const hasta = bolsa?.vigente_hasta === null ? Infinity : Date.parse(bolsa?.vigente_hasta);
  if (!Number.isFinite(instante) || !Number.isFinite(desde)
    || !(Number.isFinite(hasta) || hasta === Infinity)) return null;
  return instante >= desde && instante < hasta;
}

export function resumirBolsasInicio(lectura) {
  if (lectura?.carga !== "listo") return { estado: lectura?.carga || "cargando", bolsas: null };
  const bolsas = lectura.datos?.bolsas;
  if (!Array.isArray(bolsas)) return { estado: "error", bolsas: null };
  const generadoEn = lectura.datos?.generado_en;
  const vigencias = bolsas.map((bolsa) => vigenciaBolsaEn(bolsa, generadoEn));
  return Object.freeze({
    estado: "listo",
    bolsas,
    total: bolsas.length,
    generadoEn,
    vigentes: vigencias.every((vigente) => vigente !== null)
      ? vigencias.filter(Boolean).length : null,
    llamamientos: bolsas.reduce((total, bolsa) => total + bolsa.llamamientos_en_curso, 0),
  });
}

function renderizarBolsasInicio(resumen, acceso, escaparHTML, traducir, numero, locale) {
  const t = (clave) => escaparHTML(traducir(clave));
  let cuerpo;
  if (acceso?.disponible !== true || resumen.estado === "denegado") {
    cuerpo = acceso?.estado === "cargando"
      ? `<p role="status" class="portal-rrhh-resumen-vacio">${t("txt_cargando_bolsas_de_trabajo")}</p>`
      : acceso?.estado === "error"
        ? `<p role="alert" class="portal-rrhh-resumen-vacio">${t("txt_no_se_pudieron_cargar_las_bolsas_de_trabajo")}</p>`
        : `<p role="status" class="portal-rrhh-resumen-vacio">${t("txt_la_sesion_actual_no_dispone_de_permisos_suficien")}</p>`;
  } else if (resumen.estado === "cargando") {
    cuerpo = `<p role="status" class="portal-rrhh-resumen-vacio">${t("txt_cargando_bolsas_de_trabajo")}</p>`;
  } else if (resumen.estado !== "listo") {
    cuerpo = `<p role="alert" class="portal-rrhh-resumen-vacio">${t("txt_no_se_pudieron_cargar_las_bolsas_de_trabajo")}</p>`;
  } else if (resumen.bolsas.length === 0) {
    cuerpo = `<p class="portal-rrhh-resumen-vacio">${t("txt_el_servicio_no_ha_devuelto_bolsas_de_trabajo_reg")}</p>`;
  } else {
    const filas = resumen.bolsas.slice(0, MAXIMO_BOLSAS_INICIO).map((bolsa) => {
      const disponibles = bolsa?.por_estado?.disponible;
      return `<tr>
        <th scope="row"><button type="button" class="enlace-tabla" data-vista="resumen">${escaparHTML(bolsa.categoria)}</button></th>
        <td class="numero">${Number.isSafeInteger(disponibles) ? escaparHTML(numero(disponibles)) : "—"}</td>
        <td class="numero">${Number.isSafeInteger(bolsa.llamamientos_en_curso) ? escaparHTML(numero(bolsa.llamamientos_en_curso)) : "—"}</td>
        <td>${escaparHTML(finVigenciaBolsaPortal(bolsa.vigente_hasta, traducir))}</td>
      </tr>`;
    }).join("");
    cuerpo = `<div class="portal-rrhh-tabla"><table class="tabla-datos">
      <thead><tr><th scope="col">${t("inicio_rrhh_col_bolsa")}</th><th scope="col" class="numero">${t("inicio_rrhh_col_disponibles")}</th>
        <th scope="col" class="numero">${t("inicio_rrhh_col_llamamiento")}</th><th scope="col">${t("inicio_rrhh_col_vigente_hasta")}</th></tr></thead>
      <tbody>${filas}</tbody></table></div>
      <div class="pie-panel"><button type="button" class="boton-terciario" data-vista="resumen">${t("inicio_rrhh_ver_bolsas")} →</button></div>`;
  }
  return `<section class="panel" aria-labelledby="inicio-rrhh-bolsas">
    <div class="cabecera-panel"><h3 id="inicio-rrhh-bolsas">${t("inicio_rrhh_pestana_bolsas")}</h3></div>
    ${cuerpo}
  </section>`;
}

// Tarjeta ofrecida: módulo disponible o todavía comprobándose.
export function accesoModuloOfrecido(acceso) {
  return acceso?.disponible === true || acceso?.estado === "cargando";
}

function renderizarModulosOfrecidos(catalogo, resolverAcceso, escaparHTML, traducir) {
  return catalogo.map((modulo) => [modulo, resolverAcceso(modulo.clave)])
    .filter(([, acceso]) => accesoModuloOfrecido(acceso))
    .map(([modulo, acceso]) => renderizarModulo(modulo, acceso, escaparHTML, traducir))
    .join("");
}

function renderizarErrorCatalogo(escaparHTML, traducir) {
  return `<section class="panel" role="alert" aria-labelledby="error-catalogo-modulos-titulo">
    <div class="cabecera-panel"><h3 id="error-catalogo-modulos-titulo">${escaparHTML(traducir("titulo_error_catalogo_modulos"))}</h3></div>
    <div class="cuerpo-panel"><p>${escaparHTML(traducir("error_catalogo_modulos"))}</p>
      <div class="acciones-vista"><button type="button" class="boton-primario" data-accion="recargar-fuente">${escaparHTML(traducir("accion_reintentar"))}</button></div>
    </div>
  </section>`;
}

export function crearVistaInicioPortal({
  encabezadoVista,
  escaparHTML,
  obtenerCatalogo,
  resolverAcceso,
  traducir = traducirPortal,
  esPerfilRRHH = () => false,
  numero = (v) => String(v ?? 0),
  obtenerCuadroInicio = () => null,
  cuadroInicioPendiente = () => false,
  obtenerBolsasInicio = () => null,
  obtenerAccesosEmpleado = () => ({}),
  locale = "es-ES",
  idioma = IDIOMA_ACTUAL,
  ahora = () => new Date(),
  catalogoFallido = () => false,
  inicioPendiente = () => false,
}) {
  if (typeof encabezadoVista !== "function" || typeof escaparHTML !== "function"
    || typeof obtenerCatalogo !== "function" || typeof resolverAcceso !== "function"
    || typeof traducir !== "function" || typeof catalogoFallido !== "function"
    || typeof inicioPendiente !== "function") {
    throw new TypeError("la vista inicial requiere sus dependencias");
  }

  // Portada de RRHH: lo pendiente primero, después indicadores y reparto.
  function renderizarInicioRRHH(avisoCatalogo) {
    const t = (clave, variables) => escaparHTML(traducir(clave, variables));
    const accesoCT = resolverAcceso("contratacion_temporal");
    const disponibleCT = accesoCT?.disponible === true;
    const cuadro = disponibleCT ? obtenerCuadroInicio?.() ?? null : null;
    const resumen = cuadro ? resumirPeticiones(cuadro) : null;
    const accesoBolsa = resolverAcceso("bolsa");
    const bolsas = accesoBolsa?.disponible === true
      ? resumirBolsasInicio(obtenerBolsasInicio?.())
      : { estado: accesoBolsa?.estado || "denegado", bolsas: null };
    const disponiblesBolsa = bolsas.estado === "listo" && bolsas.bolsas.every((b) => Number.isSafeInteger(b?.por_estado?.disponible))
      ? bolsas.bolsas.reduce((suma, b) => suma + b.por_estado.disponible, 0) : null;
    const estadoCT = accesoCT?.estado === "denegado"
      ? `<p role="status" class="portal-rrhh-resumen-vacio">${t("permiso_perfil_denegado")}</p>`
      : accesoCT?.estado === "cargando"
        ? `<p role="status" class="portal-rrhh-resumen-vacio">${t("inicio_comprobando_accesos")}</p>`
        : disponibleCT && !resumen && cuadroInicioPendiente()
          ? `<p role="status" class="portal-rrhh-resumen-vacio">${t("inicio_rrhh_cuadro_cargando")}</p>`
          : (disponibleCT && !resumen ? `<p role="alert" class="portal-rrhh-resumen-vacio">${t("inicio_rrhh_cuadro_no_disponible")}</p>` : "");
    const fecha = new Intl.DateTimeFormat(locale, { dateStyle: "full", timeZone: "Europe/Madrid" }).format(ahora());
    const indicadores = [
      { clave: "en_tramite", iconoNombre: "expediente", tono: "", valor: resumen ? numero(resumen.enTramite) : null,
        etiqueta: traducir("inicio_rrhh_kpi_en_tramite"), destino: disponibleCT ? `${DESTINO_LISTA} data-ct-exp-lista-mostrar="en_tramite"` : "" },
      { clave: "vencen_semana", iconoNombre: "reloj", tono: "advertencia", valor: resumen?.vencenSemana == null ? null : numero(resumen.vencenSemana),
        etiqueta: traducir("inicio_rrhh_kpi_vencen_semana"), destino: disponibleCT ? `${DESTINO_LISTA} data-ct-exp-lista-mostrar="vencen_semana"` : "" },
      { clave: "disponibles", iconoNombre: "personas", tono: "exito", valor: disponiblesBolsa === null ? null : numero(disponiblesBolsa),
        etiqueta: traducir("inicio_rrhh_kpi_disponibles", { total: bolsas.bolsas?.length ?? 0 }), destino: disponiblesBolsa === null ? "" : 'data-vista="resumen"' },
    ].map((indicador) => renderizarIndicador({ ...indicador, escaparHTML, traducir })).join("");
    // Ofertas al SAE: sin fuente todavía; se dice en llano y sin cifra.
    const sae = `<button type="button" class="tarjeta-kpi portal-rrhh-sae" data-metrica="sae" data-vista="ofertas-sae">
      <span class="icono-kpi">${icono("contrato")}</span>
      <div><strong class="valor-kpi portal-rrhh-sae-valor">${t("inicio_rrhh_kpi_sae_valor")}</strong>
      <span class="etiqueta-kpi">${t("inicio_rrhh_kpi_sae")}</span></div></button>`;
    return `
      ${avisoCatalogo}
      <section class="portal-rrhh-inicio" aria-label="${t("menu_inicio")}">
        <header class="cabeza-pagina">
          <div><h2 class="portal-rrhh-fecha">${escaparHTML(fecha.charAt(0).toLocaleUpperCase(locale) + fecha.slice(1))}</h2></div>
          <div class="fila-acciones"><a class="boton-secundario" href="/portal-empleado/categorias-rpt/?lang=${encodeURIComponent(idioma)}">${t("inicio_rrhh_categorias_rpt")}</a>
          ${disponibleCT ? `<button type="button" class="boton-primario" data-vista="contratacion-temporal" data-ct-exp-vista="alta">${t("inicio_rrhh_nueva_peticion")}</button>` : ""}</div>
        </header>
        ${estadoCT}
        ${renderizarAccesosEmpleado({ accesos: obtenerAccesosEmpleado(), escaparHTML })}
        ${resumen ? renderizarPendientes(resumen, escaparHTML, traducir, locale) : ""}
        <div class="rejilla-kpi cuatro">${indicadores}${sae}</div>
        <div class="rejilla-dos">
          ${resumen ? renderizarPorFase(resumen, escaparHTML, traducir, numero) : ""}
          ${renderizarBolsasInicio(bolsas, accesoBolsa, escaparHTML, traducir, numero, locale)}
        </div>
      </section>`;
  }

  return function renderizarInicioPortal() {
    // Aún no se sabe si el perfil es RRHH: Inicio neutro, sin el bloque de
    // RRHH ni la nota del empleado, con todas las tarjetas «Comprobando».
    if (inicioPendiente()) {
      const catalogo = obtenerCatalogo();
      if (!Array.isArray(catalogo)) throw new TypeError("catálogo de módulos no válido");
      const comprobando = Object.freeze({ disponible: false, vista: "", estado: "cargando" });
      return `
        ${encabezadoVista("", traducir("inicio_titulo_neutro"), "")}
        <p class="portal-inicio-comprobando" role="status" data-inicio-pendiente>${escaparHTML(traducir("inicio_comprobando_accesos"))}</p>
        <div class="rejilla-modulos" aria-label="${escaparHTML(traducir("inicio_modulos_etiqueta"))}">
          ${catalogo.map((modulo) => renderizarModulo(modulo, comprobando, escaparHTML, traducir)).join("")}
        </div>`;
    }
    const avisoCatalogo = catalogoFallido() ? renderizarErrorCatalogo(escaparHTML, traducir) : "";
    if (typeof esPerfilRRHH === "function" && esPerfilRRHH()) {
      return renderizarInicioRRHH(avisoCatalogo);
    }

    const catalogo = obtenerCatalogo();
    if (!Array.isArray(catalogo)) throw new TypeError("catálogo de módulos no válido");
    const modulos = renderizarModulosOfrecidos(catalogo, resolverAcceso, escaparHTML, traducir);
    const accesosPropios = renderizarAccesosEmpleado({ accesos: obtenerAccesosEmpleado(), escaparHTML });
    // Sin ningún módulo que ofrecer (y sin fallo del catálogo, que ya tiene su
    // aviso), Inicio dice que no hay módulos en lugar de quedar en blanco.
    const contenido = modulos === "" && avisoCatalogo === "" && accesosPropios === ""
      ? `<section class="panel portal-inicio-empleado-vacio"><div class="cuerpo-panel vacio-controlado" role="status" data-inicio-sin-modulos>
          <p>${escaparHTML(traducir("inicio_empleado_sin_modulos"))}</p>
        </div></section>`
      : `<div class="rejilla-modulos portal-inicio-empleado" aria-label="${escaparHTML(traducir("txt_modulos_del_portal_del_empleado"))}">
        ${modulos}
      </div>`;
    return `
      ${encabezadoVista("", traducir("txt_portal_del_empleado"), "")}
      ${avisoCatalogo}
      ${accesosPropios}
      ${contenido}`;
  };
}

function renderizarModulo(modulo, acceso, escaparHTML, traducir) {
  const habilitado = acceso?.disponible === true && typeof acceso?.vista === "string";
  const fase = habilitado ? "disponible" : acceso?.estado;
  const etiquetaAcceso = typeof acceso?.etiqueta === "string" && acceso.etiqueta.trim() !== ""
    ? acceso.etiqueta
    : "";
  // La composición es la única que conoce si una ruta corresponde a un
  // recorrido visual o a un adaptador compuesto. La tarjeta lo deja visible,
  // sin deducirlo de un menú ni convertir una pantalla en una conexión real.
  const presentacion = habilitado && acceso?.presentacion === true;
  const comprobando = fase === "cargando";
  // Mientras su módulo carga, la tarjeta dice «Comprobando», no «no habilitado».
  const estado = etiquetaAcceso || (habilitado
    ? traducir("estado_modulo_disponible_perfil")
    : traducir(comprobando ? "estado_modulo_comprobando" : "estado_modulo_no_habilitado"));
  const reintentar = fase === "error" && acceso?.reintentar === true;
  const etiquetaAccion = typeof acceso?.accion_etiqueta === "string" && acceso.accion_etiqueta.trim() !== ""
    ? acceso.accion_etiqueta
    : traducir("accion_entrar");
  const etiquetaBoton = comprobando
    ? traducir("estado_modulo_comprobando")
    : (fase === "denegado"
      ? traducir("estado_modulo_sin_permiso")
      : traducir("estado_modulo_no_disponible"));
  return `
    <article class="tarjeta-modulo ${habilitado ? "tarjeta-modulo-habilitada" : "tarjeta-modulo-bloqueada"}${presentacion ? " tarjeta-modulo-presentacion" : ""}" data-modulo-catalogo="${escaparHTML(modulo.clave)}" tabindex="-1"${comprobando ? ' aria-busy="true"' : ""} data-estado-conexion="${presentacion ? "recorrido-visual" : (habilitado ? "conectado" : "no-conectado")}">
      <span class="icono-modulo" aria-hidden="true">${escaparHTML(modulo.sigla)}</span>
      <h3>${escaparHTML(modulo.titulo)}</h3>
      <p>${escaparHTML(modulo.texto)}</p>
      <div class="pie-tarjeta">
        <span class="${presentacion ? "estado-presentacion" : (habilitado ? "estado-disponible" : "estado-proximamente")}">${escaparHTML(estado)}</span>
        ${habilitado
          ? `<button type="button" class="${presentacion ? "boton-secundario" : "boton-primario"}" data-vista="${escaparHTML(acceso.vista)}">${escaparHTML(etiquetaAccion)}</button>`
          : (reintentar
            ? `<button type="button" class="boton-secundario" data-accion="reintentar-borradores">${escaparHTML(traducir("accion_reintentar"))}</button>`
            : `<button type="button" class="boton-secundario" disabled>${escaparHTML(etiquetaBoton)}</button>`)}
      </div>
    </article>`;
}
