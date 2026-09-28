/**
 * Catálogo del Portal del Empleado.
 *
 * Esta vista no conoce adaptadores ni datos de negocio. La composición decide
 * qué módulos están disponibles para el ContextoActor activo. Solo se ofrecen
 * los disponibles y los que aún se comprueban: un módulo sin acceso para este
 * perfil, o sin servicio, no aparece en lugar de mostrar una tarjeta vacía.
 */
import { textoPortal, traducirPortal } from "./portal-i18n.js?v=20260928-rrhh-i18n-unificada-v1";

export function calcularMetricasCuadro(cuadro) {
	const totales = cuadro?.totales;
	if (totales && Number.isSafeInteger(totales.en_tramitacion) &&
		Number.isSafeInteger(totales.con_incidencia) && Number.isSafeInteger(totales.en_llamamiento)) {
		return totales;
	}
  // Una página parcial no permite contar: hasta que el cuadro traiga totales
  // del servidor, el inicio no muestra cifras que serían falsas.
  if (cuadro?.hay_mas === true || (typeof cuadro?.paginacion?.cursor_siguiente === "string"
    && cuadro.paginacion.cursor_siguiente !== "")) return null;
  const expedientes = Array.isArray(cuadro?.expedientes) ? cuadro.expedientes : [];
  let enTramitacion = 0;
  let conIncidencia = 0;
  let enLlamamiento = 0;

  for (const exp of expedientes) {
    const estadoClave = String(exp.estado_clave || "").toLowerCase();
    const estado = String(exp.estado || "").toLowerCase();
    const faseClave = String(exp.fase_clave || "").toLowerCase();
    const faseActual = String(exp.fase_actual || "").toLowerCase();

    if (estadoClave === "en_curso" || estado.includes("tramitación") || estado.includes("en curso")) {
      enTramitacion++;
    }
    if (estadoClave === "incidencia" || estado.includes("incidencia")) {
      conIncidencia++;
    }
    if (faseClave.includes("llamamiento") || faseActual.includes("llamamiento")) {
      enLlamamiento++;
    }
  }

  return {
    en_tramitacion: enTramitacion,
    con_incidencia: conIncidencia,
    en_llamamiento: enLlamamiento,
  };
}

// Trámites que RRHH debe ver nada más entrar: primero los que tienen
// incidencia, después los más recientes; cada fila abre su expediente.
const MAXIMO_TRAMITES_INICIO = 8;

export function tramitesParaInicio(cuadro, maximo = MAXIMO_TRAMITES_INICIO) {
  const expedientes = Array.isArray(cuadro?.expedientes) ? cuadro.expedientes : [];
  const orden = (e) => (String(e.estado_clave || "") === "incidencia" ? 0 : 1);
  return [...expedientes].sort((a, b) => orden(a) - orden(b)).slice(0, maximo);
}

function renderizarTramitesInicio(tramites, escaparHTML, traducir) {
  const t = (clave) => escaparHTML(traducir(clave));
  if (!Array.isArray(tramites) || tramites.length === 0) {
    return `<p class="portal-rrhh-resumen-vacio">${t("txt_no_hay_tramites_que_mostrar")}</p>`;
  }
  return `<div class="tabla-contenedor" tabindex="0" role="region" aria-label="${t("txt_tramites_recientes")}">
      <table class="tabla-datos portal-rrhh-tramites">
        <thead><tr><th scope="col">${t("txt_expediente")}</th><th scope="col">${t("txt_centro")}</th><th scope="col">${t("txt_categoria")}</th><th scope="col">${t("txt_fase")}</th><th scope="col">${t("txt_estado")}</th></tr></thead>
        <tbody>${tramites.map((e) => `<tr>
          <th scope="row"><button type="button" class="boton-terciario portal-rrhh-abrir" data-vista="contratacion-temporal" data-ct-exp-abrir-inicio="${escaparHTML(e.expediente_ref ?? "")}">${escaparHTML(e.numero_visible ?? "")}</button></th>
          <td>${escaparHTML(e.centro ?? "—")}</td>
          <td>${escaparHTML(e.categoria ?? "—")}</td>
          <td>${escaparHTML(e.fase_actual ?? "—")}</td>
          <td><span class="ct-exp-chip ct-fase-${escaparHTML(e.estado_clave ?? "pendiente")}">${escaparHTML(e.estado ?? "—")}</span></td>
        </tr>`).join("")}</tbody>
      </table>
    </div>`;
}

export function resumirBolsasInicio(lectura) {
  if (lectura?.carga !== "listo") return { estado: lectura?.carga || "cargando", bolsas: null };
  const bolsas = lectura.datos?.bolsas;
  if (!Array.isArray(bolsas)) return { estado: "error", bolsas: null };
  const vigentes = bolsas.filter((bolsa) => bolsa.vigente_hasta === null);
  return Object.freeze({
    estado: "listo",
    bolsas,
    total: bolsas.length,
    vigentes: vigentes.length,
    llamamientos: bolsas.reduce((total, bolsa) => total + bolsa.llamamientos_en_curso, 0),
  });
}

function renderizarTarjetaInicio({ clave, etiqueta, valor, destino, accion, escaparHTML, traducir }) {
  const t = (id) => escaparHTML(traducir(id));
  const contenido = `<span class="metrica-etiqueta">${t(etiqueta)}</span>
    <strong class="metrica-valor">${valor === null ? "—" : escaparHTML(valor)}</strong>
    ${valor === null ? `<span class="metrica-enlace">${t("inicio_rrhh_recuento_no_disponible")}</span>`
      : (destino ? `<span class="metrica-enlace">${t(accion)}</span>` : "")}`;
  return destino && valor !== null
    ? `<button type="button" class="tarjeta-metrica-rrhh" data-metrica="${clave}" ${destino}>${contenido}</button>`
    : `<div class="tarjeta-metrica-rrhh" data-metrica="${clave}">${contenido}</div>`;
}

function renderizarBolsasInicio(resumen, acceso, escaparHTML, traducir, numero) {
  const t = (clave) => escaparHTML(traducir(clave));
  if (resumen.estado === "cargando") return `<p role="status" class="portal-rrhh-resumen-vacio">${t("txt_cargando_bolsas_de_trabajo")}</p>`;
  if (resumen.estado === "denegado") return `<p role="status" class="portal-rrhh-resumen-vacio">${t("txt_la_sesion_actual_no_dispone_de_permisos_suficien")}</p>`;
  if (resumen.estado !== "listo") return `<p role="alert" class="portal-rrhh-resumen-vacio">${t("txt_no_se_pudieron_cargar_las_bolsas_de_trabajo")}</p>`;
  // La portada de Bolsa puede ofrecer Elaboración por su capacidad propia;
  // el cuadro se abre si además llegó la lista autorizada que vemos aquí.
  const destino = acceso?.disponible === true ? 'data-vista="resumen"' : "";
  const tarjetas = [
    ["bolsas", "inicio_rrhh_tramites_bolsa", resumen.total],
    ["vigentes", "txt_vigentes", resumen.vigentes],
    ["llamamientos", "txt_llamamientos_en_curso", resumen.llamamientos],
  ].map(([clave, etiqueta, valor]) => renderizarTarjetaInicio({
    clave, etiqueta, valor: numero(valor), destino, accion: "inicio_rrhh_ver_bolsas", escaparHTML, traducir,
  })).join("");
  const filas = resumen.bolsas.slice(0, MAXIMO_TRAMITES_INICIO).map((bolsa) => `<tr>
    <th scope="row">${escaparHTML(bolsa.categoria)}</th>
    <td>${t("inicio_rrhh_sin_fase_bolsa")}</td>
    <td><span class="ct-exp-chip ${bolsa.vigente_hasta === null ? "ct-fase-en_curso" : "ct-fase-completado"}">${t(bolsa.vigente_hasta === null ? "txt_vigente" : "txt_sustituida")}</span></td>
  </tr>`).join("");
  return `<div class="rejilla-metricas-rrhh">${tarjetas}</div>
    <section class="portal-rrhh-tramites-seccion" aria-label="${t("inicio_rrhh_pestana_bolsas")}">
      <div class="cabecera-panel"><h3>${t("inicio_rrhh_tramites_bolsa")}</h3>
        ${destino ? `<button type="button" class="boton-terciario" ${destino}>${t("txt_ver_todos")}</button>` : ""}</div>
      ${filas ? `<div class="tabla-contenedor" tabindex="0" role="region" aria-label="${t("inicio_rrhh_tramites_bolsa")}">
        <table class="tabla-datos portal-rrhh-tramites"><thead><tr><th scope="col">${t("txt_categoria")}</th><th scope="col">${t("txt_fase")}</th><th scope="col">${t("txt_estado")}</th></tr></thead><tbody>${filas}</tbody></table></div>`
        : `<p class="portal-rrhh-resumen-vacio">${t("txt_el_servicio_no_ha_devuelto_bolsas_de_trabajo_reg")}</p>`}
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
  obtenerMetricasCuadro = () => null,
  numero = (v) => String(v ?? 0),
  obtenerTramitesInicio = () => null,
  obtenerBolsasInicio = () => null,
  catalogoFallido = () => false,
  inicioPendiente = () => false,
}) {
  if (typeof encabezadoVista !== "function" || typeof escaparHTML !== "function"
    || typeof obtenerCatalogo !== "function" || typeof resolverAcceso !== "function"
    || typeof traducir !== "function" || typeof catalogoFallido !== "function"
    || typeof inicioPendiente !== "function") {
    throw new TypeError("la vista inicial requiere sus dependencias");
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
      const catalogo = obtenerCatalogo();
      if (!Array.isArray(catalogo)) throw new TypeError("catálogo de módulos no válido");
      const t = (clave) => escaparHTML(traducir(clave));
      const accesoCT = resolverAcceso("contratacion_temporal");
      const metricas = accesoCT?.disponible === true ? obtenerMetricasCuadro?.() || null : null;
      const tramites = accesoCT?.disponible === true ? obtenerTramitesInicio?.() : null;
      const destinoCT = accesoCT?.disponible === true ? 'data-vista="contratacion-temporal" data-ct-exp-vista="cuadro"' : "";
      const tarjetasCT = [
        ["en_tramitacion", "txt_en_tramitacion", 'data-ct-exp-filtro-estado="en_curso"'],
        ["con_incidencia", "txt_con_incidencia", 'data-ct-exp-filtro-estado="incidencia"'],
        ["en_llamamiento", "txt_en_llamamiento", 'data-ct-exp-filtro-fase="llamamiento"'],
      ].map(([clave, etiqueta, filtro]) => renderizarTarjetaInicio({
        clave, etiqueta, valor: metricas ? numero(metricas[clave]) : null,
        destino: destinoCT ? `${destinoCT} ${filtro}` : "", accion: "txt_ver_tramites", escaparHTML, traducir,
      })).join("");
      const resumenBolsas = resumirBolsasInicio(obtenerBolsasInicio?.());
      const accesoBolsa = resolverAcceso("bolsa");
      const estadoCT = accesoCT?.estado === "denegado"
        ? `<p role="status" class="portal-rrhh-resumen-vacio">${t("permiso_perfil_denegado")}</p>`
        : (!metricas && tramites === null
          ? `<p role="alert" class="portal-rrhh-resumen-vacio">${t("inicio_rrhh_cuadro_no_disponible")}</p>`
          : (!metricas ? `<p role="status" class="portal-rrhh-resumen-vacio">${t("txt_los_totales_se_consultan_en_el_cuadro_de_mando")}</p>` : ""));
      return `
        ${encabezadoVista("", traducir("txt_inicio_del_portal"), "")}
        ${avisoCatalogo}
        <section class="portal-rrhh-inicio" aria-label="${t("txt_resumen_del_cuadro_de_mando")}">
          <div class="portal-rrhh-accesos" aria-label="${t("txt_accesos_directos")}">
            ${destinoCT ? `<button type="button" class="boton-primario" ${destinoCT}>${t("txt_cuadro_de_mando")}</button>
              <button type="button" class="boton-secundario" data-vista="contratacion-temporal" data-ct-exp-vista="alta">${t("txt_nueva_peticion")}</button>` : ""}
            <button type="button" class="boton-secundario portal-rrhh-ayuda" data-accion="ayuda" aria-label="${t("txt_ayuda")}">?</button>
          </div>
          <section class="portal-rrhh-cuadro panel" aria-label="${t("txt_resumen_del_cuadro_de_mando")}">
            <div class="cabecera-panel"><h3>${t("txt_resumen_del_cuadro_de_mando")}</h3></div>
            <div class="portal-rrhh-cuadro-cuerpo">
              <input class="portal-rrhh-tab-radio" type="radio" name="portal-rrhh-tab" id="portal-rrhh-tab-expedientes" checked>
              <label class="portal-rrhh-tab" for="portal-rrhh-tab-expedientes">${t("inicio_rrhh_pestana_expedientes")}</label>
              <input class="portal-rrhh-tab-radio" type="radio" name="portal-rrhh-tab" id="portal-rrhh-tab-bolsas">
              <label class="portal-rrhh-tab" for="portal-rrhh-tab-bolsas">${t("inicio_rrhh_pestana_bolsas")}</label>
              <input class="portal-rrhh-tab-radio" type="radio" name="portal-rrhh-tab" id="portal-rrhh-tab-sae">
              <label class="portal-rrhh-tab" for="portal-rrhh-tab-sae">${t("inicio_rrhh_pestana_sae")}</label>
              <section class="portal-rrhh-panel portal-rrhh-panel-expedientes" aria-label="${t("inicio_rrhh_pestana_expedientes")}">
                ${estadoCT}
                <div class="rejilla-metricas-rrhh">${tarjetasCT}</div>
                <section class="portal-rrhh-tramites-seccion" aria-label="${t("txt_tramites_recientes")}">
                  <div class="cabecera-panel"><h3>${t("txt_tramites_recientes")}</h3>
                    ${destinoCT ? `<button type="button" class="boton-terciario" ${destinoCT}>${t("txt_ver_todos")}</button>` : ""}</div>
                  ${accesoCT?.disponible === true ? renderizarTramitesInicio(tramites, escaparHTML, traducir) : ""}
                </section>
              </section>
              <section class="portal-rrhh-panel portal-rrhh-panel-bolsas" aria-label="${t("inicio_rrhh_pestana_bolsas")}">
                ${renderizarBolsasInicio(resumenBolsas, accesoBolsa, escaparHTML, traducir, numero)}
              </section>
              <section class="portal-rrhh-panel portal-rrhh-panel-sae" aria-label="${t("inicio_rrhh_pestana_sae")}">
                <p role="status">${t("inicio_rrhh_sae_pendiente")}</p>
                <button type="button" class="boton-secundario" disabled>${t("txt_ver_tramites")}</button>
              </section>
            </div>
          </section>
        </section>
        <details class="portal-rrhh-todos-modulos panel">
          <summary class="cabecera-panel" id="portal-rrhh-todos-modulos-titulo">${t("txt_todos_los_modulos_de_recursos_humanos")}</summary>
          <div class="rejilla-modulos" aria-label="${t("txt_todos_los_modulos_de_recursos_humanos")}">
            ${renderizarModulosOfrecidos(catalogo, resolverAcceso, escaparHTML, traducir)}
          </div>
        </details>`;
    }

    const catalogo = obtenerCatalogo();
    if (!Array.isArray(catalogo)) throw new TypeError("catálogo de módulos no válido");
    const modulos = renderizarModulosOfrecidos(catalogo, resolverAcceso, escaparHTML, traducir);
    // Sin ningún módulo que ofrecer (y sin fallo del catálogo, que ya tiene su
    // aviso), Inicio dice que no hay módulos en lugar de quedar en blanco.
    const contenido = modulos === "" && avisoCatalogo === ""
      ? `<section class="panel portal-inicio-empleado-vacio"><div class="cuerpo-panel vacio-controlado" role="status" data-inicio-sin-modulos>
          <p>${escaparHTML(traducir("inicio_empleado_sin_modulos"))}</p>
        </div></section>`
      : `<div class="rejilla-modulos portal-inicio-empleado" aria-label="${textoPortal("txt_modulos_del_portal_del_empleado")}">
        ${modulos}
      </div>`;
    return `
      ${encabezadoVista("", traducirPortal("txt_portal_del_empleado"), "")}
      ${avisoCatalogo}
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
