/**
 * Catálogo del Portal del Empleado y portada de RRHH.
 *
 * Esta vista no conoce adaptadores ni datos de negocio. La composición decide
 * qué módulos están disponibles para el ContextoActor activo. Solo se ofrecen
 * los disponibles y los que aún se comprueban: un módulo sin acceso para este
 * perfil, o sin servicio, no aparece en lugar de mostrar una tarjeta vacía.
 *
 * La portada de RRHH es un cuadro de mandos: primero cuántas peticiones piden
 * atención (plazo de fase vencido o que vence hoy, o incidencia), después los
 * indicadores y el reparto por fase. Los recuentos los calcula el servidor
 * sobre todo el cuadro con los criterios de la lista (recuentos-peticiones.js)
 * y la misma autorización; la portada no descarga filas ni deduce tareas.
 */
import { finVigenciaBolsaPortal, traducirPortal } from "./portal-i18n.js?v=20261007-pantallas-textos-final-v1";
import { faseRRHH, FASES_RRHH } from "./modulos/contratacion-temporal/fases-rrhh-datos.js?v=20261007-pantallas-textos-final-v1";
import { icono } from "../comun/iconos-vec.js?v=20260925-aspecto-v1";
import { IDIOMA_ACTUAL } from "../comun/idioma.js";
import { renderizarAccesosEmpleado } from "./portal-accesos-empleado.js?v=20261001-g364-reconciliar-v2";
import { rutaCandidatosBolsaCompartible } from "./portal-bolsas-ruta-filtros.js?v=20261010-ct-bolsa-cohorte-v10";

const DESTINO_LISTA = 'data-vista="contratacion-temporal" data-ct-exp-vista="cuadro"';

// Recuentos del servidor (todo el cuadro, misma autorización que la lista)
// en la forma de la portada: el reparto pasa de la fase del servidor a las
// ocho fases del procedimiento de RRHH.
export function resumenPortadaDesdeServidor(resumen) {
  if (!resumen || typeof resumen !== "object") return null;
  const porFase = {};
  for (const [faseServidor, total] of Object.entries(resumen.por_fase ?? {})) {
    const fase = faseRRHH(faseServidor)?.clave;
    if (fase) porFase[fase] = (porFase[fase] ?? 0) + total;
  }
  return Object.freeze({
    enTramite: resumen.en_tramite, vencidos: resumen.vencidos, vencenHoy: resumen.vencen_hoy,
    conIncidencia: resumen.con_incidencia, vencenSemana: resumen.vencen_semana,
    sinCalcular: resumen.sin_calcular, porFase: Object.freeze(porFase),
  });
}

// Los recuentos pueden solaparse. El cuadro V1 solo puede consultar todas las
// páginas de «incidencia» con el mismo predicado del resumen; los plazos son
// informativos hasta disponer de un filtro de servidor equivalente.
function renderizarPendientes(resumen, escaparHTML, traducir, numero) {
  const t = (clave, variables) => escaparHTML(traducir(clave, variables));
  const total = resumen.vencidos + resumen.vencenHoy + resumen.conIncidencia;
  const contadores = [
    ["vencidos", "reloj", "peligro", resumen.vencidos, "vencidos", "inicio_rrhh_pendientes_vencidos"],
    ["vencen_hoy", "reloj", "advertencia", resumen.vencenHoy, "vence_hoy", "inicio_rrhh_pendientes_hoy"],
    ["incidencias", "alerta", "peligro", resumen.conIncidencia, "incidencia", "inicio_rrhh_pendientes_incidencia"],
  ].map(([clave, iconoNombre, tono, valor, mostrar, rotulo]) => renderizarIndicador({
    clave, iconoNombre, tono, valor: numero(valor), etiqueta: traducir(rotulo),
    ariaEtiqueta: traducir(`${rotulo}_aria`, { total: numero(valor) }),
    destino: mostrar === "incidencia" ? `${DESTINO_LISTA} data-ct-exp-lista-mostrar="incidencia"` : "",
    escaparHTML, traducir,
  })).join("");
  const sinCalcular = resumen.sinCalcular > 0
    ? `<p class="portal-rrhh-parcial" role="status">${resumen.sinCalcular === 1 ? t("inicio_rrhh_sin_calcular_uno")
      : t("inicio_rrhh_sin_calcular_varias", { total: numero(resumen.sinCalcular) })}</p>` : "";
  return `<section class="panel portal-rrhh-pendientes" aria-labelledby="inicio-rrhh-pendientes-titulo">
    <div class="cabecera-panel"><h3 id="inicio-rrhh-pendientes-titulo">${t("inicio_rrhh_pendientes_titulo")}</h3>
      <button type="button" class="boton-terciario" ${DESTINO_LISTA}>${t("inicio_rrhh_ver_peticiones")} →</button></div>
    ${total === 0 && resumen.sinCalcular === 0 ? `<p class="portal-rrhh-resumen-vacio">${t("inicio_rrhh_pendientes_vacio")}</p>`
    : `<div class="rejilla-kpi">${contadores}</div>`}
    ${sinCalcular}
  </section>`;
}

// Indicador que lleva a una lista: icono, valor y rótulo. Sin dato: «—».
function renderizarIndicador({ clave, iconoNombre, tono, valor, etiqueta, destino, ariaEtiqueta = "", escaparHTML, traducir }) {
  const contenido = `<span class="icono-kpi">${icono(iconoNombre)}</span>
    <div><strong class="valor-kpi">${valor === null ? "—" : escaparHTML(valor)}</strong>
    <span class="etiqueta-kpi">${escaparHTML(etiqueta)}</span>
    ${destino && valor !== null ? `<span class="metrica-enlace" aria-hidden="true">${escaparHTML(traducir("inicio_rrhh_kpi_ver"))} →</span>` : ""}</div>`;
  const clase = `tarjeta-kpi${tono ? ` kpi--${tono}` : ""}`;
  return destino && valor !== null
    ? `<button type="button" class="${clase}" data-metrica="${clave}" ${destino}${ariaEtiqueta ? ` aria-label="${escaparHTML(ariaEtiqueta)}"` : ""}>${contenido}</button>`
    : `<div class="${clase}" data-metrica="${clave}">${contenido}</div>`;
}

function renderizarPorFase(resumen, escaparHTML, traducir, numero) {
  const t = (clave, variables) => escaparHTML(traducir(clave, variables));
  const filas = FASES_RRHH.map((fase, indice) => {
    const nombre = traducir(`tramite_fase_${fase}`);
    const total = resumen.porFase[fase] ?? 0;
    return `<tr>
      <th scope="row">${indice + 1}. ${escaparHTML(nombre)}</th>
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
      : `<p role="status" class="portal-rrhh-resumen-vacio">${t("txt_la_sesion_actual_no_dispone_de_permisos_suficien")}</p>`;
  } else if (resumen.estado === "cargando") {
    cuerpo = `<p role="status" class="portal-rrhh-resumen-vacio">${t("txt_cargando_bolsas_de_trabajo")}</p>`;
  } else if (resumen.estado !== "listo") {
    cuerpo = `<p role="alert" class="portal-rrhh-resumen-vacio">${t("txt_no_se_pudieron_cargar_las_bolsas_de_trabajo")}</p>
      <button type="button" class="boton-secundario" data-bolsa-inicio-reintentar>${t("accion_reintentar")}</button>`;
  } else if (resumen.bolsas.length === 0) {
    cuerpo = `<p class="portal-rrhh-resumen-vacio">${t("txt_el_servicio_no_ha_devuelto_bolsas_de_trabajo_reg")}</p>`;
  } else {
    const filas = resumen.bolsas.slice(0, MAXIMO_BOLSAS_INICIO).map((bolsa) => {
      const disponibles = bolsa?.por_estado?.disponible;
      let enlace = null;
      let enlaceDisponibles = null;
      try {
        enlace = rutaCandidatosBolsaCompartible(globalThis.location?.search ?? "", bolsa.bolsa_ref);
        if (Number.isSafeInteger(disponibles) && disponibles >= 0) {
          enlaceDisponibles = rutaCandidatosBolsaCompartible(globalThis.location?.search ?? "", bolsa.bolsa_ref, "disponible");
        }
      }
      catch { /* Una referencia no válida se muestra sin enlace. */ }
      const cifraDisponibles = enlaceDisponibles
        ? `<a class="enlace-tabla" href="${escaparHTML(enlaceDisponibles)}" data-accion="ver-bolsa" data-bolsa-ref="${escaparHTML(bolsa.bolsa_ref)}" data-estado="disponible" aria-label="${escaparHTML(traducir("txt_aria_ver_candidatos_estado", { total: numero(disponibles), estado: traducir("inicio_rrhh_col_disponibles").toLocaleLowerCase(locale), categoria: bolsa.categoria }))}">${escaparHTML(numero(disponibles))}</a>`
        : Number.isSafeInteger(disponibles) && disponibles >= 0 ? escaparHTML(numero(disponibles)) : "—";
      return `<tr>
        <th scope="row">${enlace
          ? `<a class="enlace-tabla" href="${escaparHTML(enlace)}" data-accion="ver-bolsa" data-bolsa-ref="${escaparHTML(bolsa.bolsa_ref)}">${escaparHTML(bolsa.categoria)}</a>`
          : escaparHTML(bolsa.categoria)}</th>
        <td class="numero">${cifraDisponibles}</td>
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
    const resumen = resumenPortadaDesdeServidor(cuadro?.resumen);
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
        : accesoCT?.estado === "no_disponible" && obtenerCatalogo().some((modulo) => modulo.clave === "contratacion_temporal")
          ? `<section class="panel" role="status" aria-labelledby="inicio-ct-no-disponible">
              <div class="cabecera-panel"><h3 id="inicio-ct-no-disponible">${t("contratacion_temporal_titulo")}</h3></div>
              <div class="cuerpo-panel"><p>${t("contratacion_temporal_aviso_no_disponible")}</p>
                <div class="acciones-vista"><button type="button" class="boton-secundario" data-accion="recargar-fuente">${t("txt_reintentar")}</button></div>
              </div></section>`
        : (disponibleCT && !resumen ? `<p role="alert" class="portal-rrhh-resumen-vacio">${t("inicio_rrhh_cuadro_no_disponible")}</p>` : "");
    const fecha = new Intl.DateTimeFormat(locale, { dateStyle: "full", timeZone: "Europe/Madrid" }).format(ahora());
    const indicadores = [
      { clave: "en_tramite", iconoNombre: "expediente", tono: "", valor: resumen ? numero(resumen.enTramite) : null,
        etiqueta: traducir("inicio_rrhh_kpi_en_tramite"), destino: "" },
      { clave: "vencen_semana", iconoNombre: "reloj", tono: "advertencia", valor: resumen?.vencenSemana == null ? null : numero(resumen.vencenSemana),
        etiqueta: traducir("inicio_rrhh_kpi_vencen_semana"), destino: "" },
      { clave: "disponibles", iconoNombre: "personas", tono: "exito", valor: disponiblesBolsa === null ? null : numero(disponiblesBolsa),
        etiqueta: traducir("inicio_rrhh_kpi_disponibles", { total: bolsas.bolsas?.length ?? 0 }), destino: disponiblesBolsa === null ? "" : 'data-vista="resumen"' },
    ].map((indicador) => renderizarIndicador({ ...indicador, escaparHTML, traducir })).join("");
    return `
      ${avisoCatalogo}
      <section class="portal-rrhh-inicio" aria-label="${t("menu_inicio")}">
        <header class="cabeza-pagina">
          <div><h2 class="portal-rrhh-fecha">${escaparHTML(fecha.charAt(0).toLocaleUpperCase(locale) + fecha.slice(1))}</h2></div>
          <div class="fila-acciones"><button type="button" class="boton-secundario" data-vista="categorias-rpt">${t("inicio_rrhh_categorias_rpt")}</button>
          ${disponibleCT ? `<button type="button" class="boton-primario" data-vista="contratacion-temporal" data-ct-exp-vista="alta">${t("inicio_rrhh_nueva_peticion")}</button>` : ""}</div>
        </header>
        ${estadoCT}
        ${renderizarAccesosEmpleado({ accesos: obtenerAccesosEmpleado(), escaparHTML })}
        ${resumen ? renderizarPendientes(resumen, escaparHTML, traducir, numero) : ""}
        <div class="rejilla-kpi rejilla-kpi--compacta">${indicadores}</div>
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
  const fase = acceso?.estado === "cargando" ? "cargando" : (habilitado ? "disponible" : acceso?.estado);
  const etiquetaAcceso = typeof acceso?.etiqueta === "string" && acceso.etiqueta.trim() !== ""
    ? acceso.etiqueta
    : "";
  // La composición es la única que conoce si una ruta corresponde a un
  // recorrido visual o a un adaptador compuesto. La tarjeta lo deja visible,
  // sin deducirlo de un menú ni convertir una pantalla en una conexión real.
  const presentacion = habilitado && acceso?.presentacion === true;
  const comprobando = fase === "cargando";
  // Mientras su módulo carga, la tarjeta dice «Comprobando», no «no habilitado».
  const estado = comprobando && habilitado ? traducir("estado_modulo_comprobando") : etiquetaAcceso || (habilitado
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
    <article class="tarjeta-modulo ${habilitado ? "tarjeta-modulo-habilitada" : "tarjeta-modulo-bloqueada"}${presentacion ? " tarjeta-modulo-presentacion" : ""}" data-modulo-catalogo="${escaparHTML(modulo.clave)}" tabindex="-1"${comprobando ? ' aria-busy="true"' : ""} data-estado-conexion="${comprobando ? "comprobando" : (presentacion ? "recorrido-visual" : (habilitado ? "conectado" : "no-conectado"))}">
      <span class="icono-modulo" aria-hidden="true">${escaparHTML(modulo.sigla)}</span>
      <h3>${escaparHTML(modulo.titulo)}</h3>
      <p>${escaparHTML(modulo.texto)}</p>
      <div class="pie-tarjeta">
        <span class="${comprobando ? "estado-proximamente" : (presentacion ? "estado-presentacion" : (habilitado ? "estado-disponible" : "estado-proximamente"))}">${escaparHTML(estado)}</span>
        ${habilitado
          ? `<button type="button" class="${presentacion ? "boton-secundario" : "boton-primario"}" data-vista="${escaparHTML(acceso.vista)}">${escaparHTML(etiquetaAccion)}</button>`
          : (reintentar
            ? `<button type="button" class="boton-secundario" data-accion="reintentar-borradores">${escaparHTML(traducir("accion_reintentar"))}</button>`
            : `<button type="button" class="boton-secundario" disabled>${escaparHTML(etiquetaBoton)}</button>`)}
      </div>
    </article>`;
}
