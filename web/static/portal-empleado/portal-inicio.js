/**
 * Catálogo del Portal del Empleado.
 *
 * Esta vista no conoce adaptadores ni datos de negocio. La composición decide
 * qué módulos están disponibles para el ContextoActor activo. Solo se ofrecen
 * los disponibles y los que aún se comprueban: un módulo sin acceso para este
 * perfil, o sin servicio, no aparece en lugar de mostrar una tarjeta vacía.
 */
import { traducirPortal } from "./portal-i18n.js?v=20260925-portal-integrado-v1";

export function calcularMetricasCuadro(cuadro) {
	const totales = cuadro?.totales;
	if (totales && Number.isSafeInteger(totales.en_tramitacion) &&
		Number.isSafeInteger(totales.con_incidencia) && Number.isSafeInteger(totales.en_llamamiento)) {
		return totales;
	}
  // Una página parcial no permite contar: hasta que el cuadro traiga totales
  // del servidor, el inicio no muestra cifras que serían falsas.
  if (cuadro?.hay_mas === true) return null;
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

function renderizarTramitesInicio(tramites, escaparHTML) {
  if (!Array.isArray(tramites) || tramites.length === 0) {
    return `<p class="portal-rrhh-resumen-vacio">No hay trámites que mostrar.</p>`;
  }
  return `<div class="tabla-contenedor" tabindex="0" role="region" aria-label="Trámites recientes">
      <table class="tabla-datos portal-rrhh-tramites">
        <thead><tr><th scope="col">Expediente</th><th scope="col">Centro</th><th scope="col">Categoría</th><th scope="col">Fase</th><th scope="col">Estado</th></tr></thead>
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
      const metricas = obtenerMetricasCuadro?.() || null;
      const resumen = metricas
        ? `<div class="rejilla-metricas-rrhh">
              <button type="button" class="tarjeta-metrica-rrhh" data-metrica="en_tramitacion" data-vista="contratacion-temporal" data-ct-exp-vista="cuadro" data-ct-exp-filtro-estado="en_curso">
                <span class="metrica-etiqueta">En tramitación</span>
                <strong class="metrica-valor">${escaparHTML(numero(metricas.en_tramitacion))}</strong>
                <span class="metrica-enlace">Ver trámites</span>
              </button>
              <button type="button" class="tarjeta-metrica-rrhh" data-metrica="con_incidencia" data-vista="contratacion-temporal" data-ct-exp-vista="cuadro" data-ct-exp-filtro-estado="incidencia">
                <span class="metrica-etiqueta">Con incidencia</span>
                <strong class="metrica-valor">${escaparHTML(numero(metricas.con_incidencia))}</strong>
                <span class="metrica-enlace">Ver trámites</span>
              </button>
              <button type="button" class="tarjeta-metrica-rrhh" data-metrica="en_llamamiento" data-vista="contratacion-temporal" data-ct-exp-vista="cuadro" data-ct-exp-filtro-fase="llamamiento">
                <span class="metrica-etiqueta">En llamamiento</span>
                <strong class="metrica-valor">${escaparHTML(numero(metricas.en_llamamiento))}</strong>
                <span class="metrica-enlace">Ver trámites</span>
              </button>
            </div>`
        : `<p class="portal-rrhh-resumen-vacio">Los totales se consultan en el cuadro de mando.</p>`;
      return `
        ${encabezadoVista("", "Inicio del portal", "")}
        ${avisoCatalogo}
        <section class="portal-rrhh-inicio" aria-label="Inicio de Contratación Temporal">
          <div class="portal-rrhh-accesos" aria-label="Accesos directos">
            <button type="button" class="boton-primario" data-vista="contratacion-temporal" data-ct-exp-vista="cuadro">Cuadro de mando</button>
            <button type="button" class="boton-secundario" data-vista="contratacion-temporal" data-ct-exp-vista="alta">Nueva petición</button>
            <button type="button" class="boton-secundario" data-accion="ayuda">Ayuda</button>
          </div>
          <section class="portal-rrhh-resumen" aria-label="Resumen de expedientes">
            <div class="cabecera-panel">
              <h3>Resumen del cuadro de mando</h3>
            </div>
            ${resumen}
          </section>
          <section class="portal-rrhh-tramites-seccion" aria-label="Trámites recientes">
            <div class="cabecera-panel">
              <h3>Trámites recientes</h3>
              <button type="button" class="boton-terciario" data-vista="contratacion-temporal" data-ct-exp-vista="cuadro">Ver todos</button>
            </div>
            ${renderizarTramitesInicio(obtenerTramitesInicio?.(), escaparHTML)}
          </section>
        </section>
        <section class="portal-rrhh-todos-modulos" aria-labelledby="portal-rrhh-todos-modulos-titulo">
          <div class="cabecera-panel">
            <h3 id="portal-rrhh-todos-modulos-titulo">Todos los módulos de Recursos Humanos</h3>
          </div>
          <div class="rejilla-modulos" aria-label="Todos los módulos de Recursos Humanos">
            ${renderizarModulosOfrecidos(catalogo, resolverAcceso, escaparHTML, traducir)}
          </div>
        </section>`;
    }

    const catalogo = obtenerCatalogo();
    if (!Array.isArray(catalogo)) throw new TypeError("catálogo de módulos no válido");
    return `
      ${encabezadoVista("", "Portal del Empleado", "")}
      ${avisoCatalogo}
      <div class="rejilla-modulos portal-inicio-empleado" aria-label="Módulos del Portal del Empleado">
        ${renderizarModulosOfrecidos(catalogo, resolverAcceso, escaparHTML, traducir)}
      </div>`;
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
