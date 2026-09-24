/** Consulta de relaciones autorizadas y pasos pendientes de formalización. */
import { traducirContratos } from "./portal-i18n-contratos.js?v=20260924-f2-web2";

export function crearVistasOperaciones({ escaparHTML: e, fecha, chip, tabla, encabezadoVista }) {
  function renderizarContratos(datos) {
    const t = traducirContratos;
    // Solo este campo explícito admite lecturas autorizadas; `datos.contratos`
    // pertenece al antiguo panel de presentación y nunca alimenta esta vista.
    const fuente = datos?.contratos_fuente;
    const estados = ["cargando", "disponible", "vacio", "no_configurado", "denegado", "error"];
    let estado = estados.includes(fuente?.estado) ? fuente.estado : "no_configurado";
    const registroValido = (item) => item && typeof item === "object"
      && ["expediente", "acto", "bolsa", "estado"].every((clave) => typeof item[clave] === "string");
    if (estado === "disponible" && (!Array.isArray(fuente.registros)
      || !fuente.registros.every(registroValido))) estado = "error";
    const contratos = estado === "disponible" ? fuente.registros : [];
    if (estado === "disponible" && contratos.length === 0) estado = "vacio";
    const clasesEstado = { cargando: "info", disponible: "exito", vacio: "neutro", no_configurado: "aviso", denegado: "peligro", error: "peligro" };
    const filas = contratos.map((item) => [
      `<strong class="contratos-referencia">${e(item.expediente)}</strong>`,
      e(item.acto), e(item.bolsa),
      `<span class="contratos-fecha">${e(fecha(item.inicio))}</span><span class="contratos-fecha contratos-fecha-fin">${e(fecha(item.fin))}</span>`,
      chip(item.estado),
    ]);
    const pasos = [
      ["paso_bolsa", "paso_bolsa_descripcion"],
      ["paso_formalizacion", "paso_formalizacion_descripcion"],
      ["paso_personal", "paso_personal_descripcion"],
      ["paso_ginpix", "paso_ginpix_descripcion"],
      ["paso_reincorporacion", "paso_reincorporacion_descripcion"],
    ];
    const pasoHTML = pasos.map(([titulo, descripcion], indice) => `
      <li class="contratos-paso"><span class="contratos-paso-numero" aria-hidden="true">${indice + 1}</span>
        <div><strong>${e(t(titulo))}</strong><small>${e(t(descripcion))}</small></div></li>`).join("");
    const accionPendiente = (etiqueta, motivo, atributoOperacion) => `
      <div class="contratos-accion"><button class="boton-secundario" type="button" ${atributoOperacion} disabled aria-disabled="true" title="${e(t(motivo))}">${e(t(etiqueta))}</button>
        <small>${e(t(motivo))}</small></div>`;
    return `<div class="contratos-vista">
      ${encabezadoVista(t("sobrelinea"), t("titulo"), t("descripcion"))}
      <section class="nota-pendiente" role="status" aria-live="polite">${e(t(`detalle_${estado}`))}</section>
      <section class="panel contratos-panel-circuito" aria-labelledby="contratos-circuito-titulo">
        <div class="cabecera-panel"><div><h3 id="contratos-circuito-titulo">${e(t("circuito_titulo"))}</h3><p>${e(t("circuito_subtitulo"))}</p></div></div>
        <ol class="contratos-pasos">${pasoHTML}</ol>
      </section>
      <section class="contratos-secciones" aria-label="${e(t("navegacion_titulo"))}">
        <details class="panel contratos-seccion contratos-panel-registros" name="contratos-recorrido" open>
          <summary><strong>${e(t("seccion_contratos"))}</strong><span class="estado-chip ${clasesEstado[estado]}">${e(t(`estado_${estado}`))}</span></summary>
          <p class="contratos-seccion-descripcion">${e(t("registros_subtitulo"))}</p>
          ${tabla({ titulo: t("tabla_titulo"), cabeceras: [t("columna_expediente"), t("columna_acto"), t("columna_bolsa"), t("columna_fechas"), t("columna_estado")], clavesColumnas: ["referencia", "acto", "bolsa", "fechas", "estado"], prioridadColumnas: "estado", filas, vacio: t("vacio") })}
          ${accionPendiente("accion_contrato", "motivo_contrato", 'data-operacion="registrar-contrato"')}
        </details>
        <details class="panel contratos-seccion" name="contratos-recorrido">
          <summary><strong>${e(t("seccion_ceses"))}</strong><span class="estado-chip aviso">${e(t("accion_pendiente"))}</span></summary>
          <p class="contratos-seccion-descripcion">${e(t("descripcion_ceses"))}</p>
          ${accionPendiente("accion_cese", "motivo_cese", 'data-operacion="registrar-cese"')}
        </details>
        <details class="panel contratos-seccion" name="contratos-recorrido">
          <summary><strong>${e(t("seccion_reincorporacion"))}</strong><span class="estado-chip aviso">${e(t("accion_pendiente"))}</span></summary>
          <p class="contratos-seccion-descripcion">${e(t("descripcion_reincorporacion"))}</p>
          ${accionPendiente("accion_reincorporar", "motivo_reincorporar", 'data-operacion="reincorporar-bolsa"')}
        </details>
      </section>
      <details class="contratos-ayuda"><summary>${e(t("ayuda"))}</summary><p>${e(t("ayuda_contenido"))}</p></details>
    </div>`;
  }

  return Object.freeze({ renderizarContratos });
}
