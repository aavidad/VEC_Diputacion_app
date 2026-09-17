/**
 * Catálogo del Portal del Empleado.
 *
 * Esta vista no conoce adaptadores ni datos de negocio. La composición decide
 * qué módulos están disponibles para el ContextoActor activo; los restantes
 * permanecen visibles para conservar la navegación estable y fallan cerrados.
 */
import { traducirPortal } from "./portal-i18n.js?v=20260721-acceso-real-v2";

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
        <thead><tr><th scope="col">Expediente</th><th scope="col">Centro</th><th scope="col">Categoría</th><th scope="col">Fase</th><th scope="col">Estado</th><th scope="col"><span class="visualmente-oculto">Acción</span></th></tr></thead>
        <tbody>${tramites.map((e) => `<tr>
          <th scope="row">${escaparHTML(e.numero_visible ?? "")}</th>
          <td>${escaparHTML(e.centro ?? "—")}</td>
          <td>${escaparHTML(e.categoria ?? "—")}</td>
          <td>${escaparHTML(e.fase_actual ?? "—")}</td>
          <td><span class="ct-fase-${escaparHTML(e.estado_clave ?? "pendiente")}">${escaparHTML(e.estado ?? "—")}</span></td>
          <td><button type="button" class="boton-secundario" data-vista="contratacion-temporal" data-ct-exp-abrir-inicio="${escaparHTML(e.expediente_ref ?? "")}">Abrir</button></td>
        </tr>`).join("")}</tbody>
      </table>
    </div>`;
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
}) {
  if (typeof encabezadoVista !== "function" || typeof escaparHTML !== "function"
    || typeof obtenerCatalogo !== "function" || typeof resolverAcceso !== "function"
    || typeof traducir !== "function") {
    throw new TypeError("la vista inicial requiere sus dependencias");
  }

  return function renderizarInicioPortal() {
    if (typeof esPerfilRRHH === "function" && esPerfilRRHH()) {
      const metricas = obtenerMetricasCuadro?.() || null;
      const resumen = metricas
        ? `<div class="rejilla-metricas-rrhh">
              <article class="tarjeta-metrica-rrhh" data-metrica="en_tramitacion">
                <span class="metrica-etiqueta">En tramitación</span>
                <strong class="metrica-valor">${escaparHTML(numero(metricas.en_tramitacion))}</strong>
              </article>
              <article class="tarjeta-metrica-rrhh" data-metrica="con_incidencia">
                <span class="metrica-etiqueta">Con incidencia</span>
                <strong class="metrica-valor">${escaparHTML(numero(metricas.con_incidencia))}</strong>
              </article>
              <article class="tarjeta-metrica-rrhh" data-metrica="en_llamamiento">
                <span class="metrica-etiqueta">En llamamiento</span>
                <strong class="metrica-valor">${escaparHTML(numero(metricas.en_llamamiento))}</strong>
              </article>
            </div>`
        : `<p class="portal-rrhh-resumen-vacio">Los totales se consultan en el cuadro de mando.</p>`;
      return `
        ${encabezadoVista(
          "Gestión de personal",
          "Inicio del portal",
          "Accesos directos y estado general de la contratación temporal para Recursos Humanos.",
        )}
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
        </section>`;
    }

    const catalogo = obtenerCatalogo();
    if (!Array.isArray(catalogo)) throw new TypeError("catálogo de módulos no válido");
    return `
      ${encabezadoVista("Acceso unificado", "Portal del Empleado", "Identidad, datos, documentos y trazabilidad se comparten mediante contratos comunes. La disponibilidad depende del perfil y de los adaptadores compuestos.")}
      <section class="nota-seguridad" aria-label="Separación de acceso">
        Este portal representa el acceso interno. La zona externa de aspirantes usa otra sesión, permisos y proyección de datos; nunca muestra expedientes de terceras personas.
      </section>
      <div class="rejilla-modulos" aria-label="Módulos del Portal del Empleado">
        ${catalogo.map((modulo) => renderizarModulo(
          modulo, resolverAcceso(modulo.clave), escaparHTML, traducir,
        )).join("")}
      </div>`;
  };
}

function renderizarModulo(modulo, acceso, escaparHTML, traducir) {
  const habilitado = acceso?.disponible === true && typeof acceso?.vista === "string";
  const fase = habilitado ? "disponible" : acceso?.estado;
  const estado = habilitado
    ? traducir("estado_modulo_disponible_perfil")
    : (acceso?.etiqueta || traducir("estado_modulo_no_habilitado"));
  const comprobando = fase === "cargando";
  const reintentar = fase === "error" && acceso?.reintentar === true;
  const etiquetaBoton = comprobando
    ? traducir("estado_modulo_comprobando")
    : (fase === "denegado"
      ? traducir("estado_modulo_sin_permiso")
      : traducir("estado_modulo_no_disponible"));
  return `
    <article class="tarjeta-modulo ${habilitado ? "tarjeta-modulo-habilitada" : "tarjeta-modulo-bloqueada"}" data-modulo-catalogo="${escaparHTML(modulo.clave)}" tabindex="-1"${comprobando ? ' aria-busy="true"' : ""}>
      <span class="icono-modulo" aria-hidden="true">${escaparHTML(modulo.sigla)}</span>
      <h3>${escaparHTML(modulo.titulo)}</h3>
      <p>${escaparHTML(modulo.texto)}</p>
      <div class="pie-tarjeta">
        <span class="${habilitado ? "estado-disponible" : "estado-proximamente"}" role="status" aria-live="polite">${escaparHTML(estado)}</span>
        ${habilitado
          ? `<button type="button" class="boton-primario" data-vista="${escaparHTML(acceso.vista)}">${escaparHTML(traducir("accion_entrar"))}</button>`
          : (reintentar
            ? `<button type="button" class="boton-secundario" data-accion="reintentar-borradores">${escaparHTML(traducir("accion_reintentar"))}</button>`
            : `<button type="button" class="boton-secundario" disabled>${escaparHTML(etiquetaBoton)}</button>`)}
      </div>
    </article>`;
}
