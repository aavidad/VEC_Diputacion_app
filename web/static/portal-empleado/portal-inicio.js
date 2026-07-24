/**
 * Catálogo del Portal del Empleado.
 *
 * Esta vista no conoce adaptadores ni datos de negocio. La composición decide
 * qué módulos están disponibles para el ContextoActor activo; los restantes
 * permanecen visibles para conservar la navegación estable y fallan cerrados.
 */
import { traducirPortal } from "./portal-i18n.js?v=20260721-acceso-real-v2";

export function crearVistaInicioPortal({
  encabezadoVista, escaparHTML, obtenerCatalogo, resolverAcceso, traducir = traducirPortal,
}) {
  if (typeof encabezadoVista !== "function" || typeof escaparHTML !== "function"
    || typeof obtenerCatalogo !== "function" || typeof resolverAcceso !== "function"
    || typeof traducir !== "function") {
    throw new TypeError("la vista inicial requiere sus dependencias");
  }

  return function renderizarInicioPortal() {
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
