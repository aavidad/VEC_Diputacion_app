/**
 * Ficha del expediente según la dirección de diseño del 29/09/2026. Responde,
 * en este orden, a tres preguntas: qué toca ahora, en qué fase está y qué hay
 * (documentos, historial y datos). Lo técnico (referencias, versión, flujo y
 * la tabla de actuaciones) queda plegado tras «Ver detalle técnico».
 *
 * HTML puro a partir del detalle ya autorizado: no ejecuta actuaciones ni
 * deduce responsables o plazos que el servidor no haya dado.
 */
import { faseRRHH } from "./fases-rrhh-datos.js?v=20261007-pantallas-textos-final-v1";

function escapar(valor) {
  return String(valor ?? "")
    .replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;").replaceAll("'", "&#039;");
}

// Valores que son referencias internas («unidad:rrhh-seleccion»): solo en el detalle técnico.
const PATRON_REFERENCIA = /^[a-z_]+:[^\s]+$/iu;
// Datos que responden a «qué se pidió», en el orden en que RRHH los lee.
const DATOS_CLAVE = Object.freeze([
  "centro", "categoria", "grupo_subgrupo", "motivo", "periodo", "jornada", "coste_estimado",
  "via_cobertura", "bolsa_cobertura",
]);
const MAXIMO_DATOS_CLAVE = 8;

function campoCabecera(expediente, clave) {
  return expediente.cabecera.find((campo) => campo.clave === clave);
}

function resumenDelCuadro(estado, expediente) {
  return estado.cuadro?.expedientes?.find(({ expediente_ref: ref }) => ref === expediente.expediente_ref) ?? null;
}

/** Cabecera: categoría y centro, «Fase N de 8: …» y un único estado; «Volver a la lista». */
export function renderizarCabeceraFicha(expediente, estado, t) {
  const categoria = campoCabecera(expediente, "categoria")?.valor;
  const centro = campoCabecera(expediente, "centro")?.valor;
  const fase = campoCabecera(expediente, "fase")?.valor;
  const resumen = resumenDelCuadro(estado, expediente);
  const estadoTexto = campoCabecera(expediente, "estado")?.valor ?? resumen?.estado;
  const titulo = [categoria, centro].filter(Boolean).join(" · ");
  return `<header class="cabeza-pagina ct-exp-ficha-cabecera">
    <div>
      <h3 class="ct-exp-ficha-titulo">${escapar(titulo || t("expediente_etiqueta"))}</h3>
      <p class="estado-linea">${fase ? `<span>${escapar(fase)}</span>` : ""}${estadoTexto
    ? `<span class="ct-exp-chip${resumen ? ` ct-fase-${escapar(resumen.estado_clave)}` : ""}">${escapar(estadoTexto)}</span>` : ""}</p>
    </div>
    <button type="button" class="boton-secundario" data-ct-exp-vista="cuadro">${escapar(t("ficha_volver_lista"))}</button>
  </header>`;
}

/**
 * Siguiente paso: qué, quién y hasta cuándo, con lo que dice el detalle. El
 * plazo es el de la fase del expediente (el que calcula el servidor) y se
 * nombra así; no se atribuye a una persona.
 */
export function renderizarSiguientePasoFicha(expediente, estado, t) {
  const tarea = expediente.tareas.find(({ estado_clave: e }) => e === "en_curso")
    ?? expediente.tareas.find(({ estado_clave: e }) => e === "espera")
    ?? expediente.tareas.find(({ estado_clave: e }) => e === "pendiente");
  const bloqueado = estado.carga !== "listo" || estado.ocupado || estado.actualizacion_pendiente
    || estado.resultado_indeterminado;
  const mensajeBloqueo = estado.resultado_indeterminado ? "estado_resultado_indeterminado"
    : (estado.ocupado ? "estado_registrando_actuacion"
      : (estado.actualizacion_pendiente ? "estado_actualizacion_pendiente"
        : (estado.carga === "denegado" ? "estado_denegado_expediente"
          : (estado.carga === "error" ? "estado_error_expediente"
            : (estado.carga === "cargando" ? "estado_cargando_expediente" : "estado_actualizacion_pendiente")))));
  const accion = bloqueado ? null
    : tarea?.acciones?.find(({ tipo, disponible }) => tipo === "efecto" && disponible === true);
  const actorLegible = (valor) => typeof valor === "string" && valor.trim() !== ""
    && !/^(—|-|pendiente\b|por asignar\b|por definir\b|sin asignar\b|sin determinar\b|no consta\b)/iu.test(valor.trim())
    && !PATRON_REFERENCIA.test(valor.trim());
  const actor = actorLegible(tarea?.responsable) ? tarea.responsable
    : (actorLegible(tarea?.unidad) ? tarea.unidad : "");
  const resumen = resumenDelCuadro(estado, expediente);
  const faseActual = faseRRHH(resumen?.fase_clave);
  const nombreFase = faseActual ? t(`etiqueta_fase_${faseActual.clave}`) : "";
  const terminado = ["completado", "cancelado"].includes(resumen?.estado_clave);
  const espera = resumen?.estado_clave === "espera";
  const incidencia = resumen?.estado_clave === "incidencia";
  const titulo = terminado ? t("ficha_siguiente_paso_terminado")
    : (espera ? t("ficha_siguiente_paso_espera")
      : (nombreFase ? t("ficha_siguiente_paso_fase", { fase: nombreFase }) : t("siguiente_paso_titulo")));
  const que = accion?.etiqueta ?? (tarea ? t("siguiente_paso_espera", { tarea: tarea.etiqueta }) : t("siguiente_paso_sin_tarea"));
  const conPlazo = resumen?.plazo_estado && resumen.plazo_estado !== "no_calculado";
  const plazo = conPlazo ? t("ficha_plazo_fase_estado", {
    fecha: resumen.plazo, estado: t(`plazo_fase_${resumen.plazo_estado}`),
  }) : t("siguiente_paso_plazo_desconocido");
  const clase = terminado || espera ? " en-espera" : (incidencia ? " con-incidencia" : "");
  return `<section class="siguiente-paso ct-exp-siguiente-paso${clase}" aria-labelledby="ct-exp-siguiente-paso-titulo">
    <div>
      <h3 id="ct-exp-siguiente-paso-titulo">${escapar(titulo)}</h3>
      <dl>
        <div><dt>${escapar(t("siguiente_paso_que"))}</dt><dd>${tarea || accion || terminado || espera ? escapar(que)
    : `<span class="ct-exp-que-con-tramite">${escapar(t("siguiente_paso_tramite_abajo"))}</span><span class="ct-exp-que-sin-tramite">${escapar(que)}</span>`}</dd></div>
        ${tarea ? `<div><dt>${escapar(t("siguiente_paso_quien"))}</dt><dd>${escapar(actor || t("siguiente_paso_quien_desconocido"))}</dd></div>` : ""}
        <div><dt>${escapar(t("siguiente_paso_hasta"))}</dt><dd>${escapar(plazo)}</dd></div>
      </dl>
      <p id="ct-exp-siguiente-paso-estado" role="status" aria-live="polite" aria-atomic="true">${bloqueado ? escapar(t(mensajeBloqueo)) : ""}</p>
    </div>
    ${terminado || espera ? "" : `<button type="button" class="boton-primario" data-ct-exp-accion="ir-tramite"${bloqueado ? ' disabled aria-disabled="true" aria-describedby="ct-exp-siguiente-paso-estado"' : ""}>${escapar(t("ficha_ir_tramite"))}</button>`}
  </section>`;
}

const PASO_LINEA = Object.freeze({
  completado: "hecho", en_curso: "ahora", espera: "ahora", incidencia: "con-incidencia",
});

/** Línea de las ocho fases: símbolo y palabra (Hecho / Ahora / Falta). */
export function renderizarLineaFases(expediente, t) {
  if (expediente.fases.length === 0) return "";
  const pasos = expediente.fases.map((fase) => ({ fase, paso: PASO_LINEA[fase.estado_clave] ?? "falta" }));
  const cuenta = (paso) => pasos.filter((item) => item.paso === paso).length;
  const texto = { hecho: "linea_fase_hecho", ahora: "linea_fase_ahora", "con-incidencia": "linea_fase_incidencia", falta: "linea_fase_falta" };
  return `<nav class="panel ct-exp-fases" aria-labelledby="ct-exp-fases-titulo">
    <div class="cabecera-panel"><h3 id="ct-exp-fases-titulo">${escapar(t("ficha_fases_titulo"))}</h3>
      <p class="texto-secundario">${escapar(t("ficha_fases_resumen", {
    hechas: cuenta("hecho"), ahora: cuenta("ahora") + cuenta("con-incidencia"), faltan: cuenta("falta"),
  }))}</p></div>
    <ol class="linea-fases" data-ct-exp-rail>${pasos.map(({ fase, paso }) => {
    const clave = String(fase.fase_ref ?? "").split(":").at(-1) ?? "";
    const actual = paso === "ahora" || paso === "con-incidencia";
    return `<li class="${paso}" data-ct-exp-orden="${fase.orden}"${actual ? ' aria-current="step"' : ""}>
      <button type="button" data-ct-exp-fase-ver="${escapar(clave)}" aria-pressed="false"
        aria-label="${escapar(`${t("fase_ver_pantalla", { fase: fase.etiqueta })}: ${t(texto[paso])}`)}">
        <span class="marca" aria-hidden="true">${paso === "hecho" ? "✓" : escapar(fase.orden)}</span>
        <span class="nombre">${escapar(fase.etiqueta)}</span>
        <small>${escapar(t(texto[paso]))}</small>
      </button>
    </li>`;
  }).join("")}</ol>
  </nav>`;
}

/**
 * Documentos: si el índice del expediente ya está consultado, como lista de
 * comprobación; si no, el acceso a su pantalla. La firma se ancla debajo.
 */
export function renderizarDocumentosFicha(estado, t) {
  const expediente = estado.expediente;
  const indice = estado.documentos?.expediente_ref === expediente.expediente_ref ? estado.documentos : null;
  const sinPantallaPropia = estado.navegacion?.documentos === false;
  const verDocumentos = sinPantallaPropia ? ""
    : `<button type="button" class="boton-secundario" data-ct-exp-vista="documentos">${escapar(t("ficha_documentos_ver"))}</button>`;
  // Sin índice propio, el montaje coloca aquí la lista común de documentos
  // del expediente (con su botón «Descargar») si el portal la ofrece.
  const cuerpo = indice && indice.documentos.length
    ? renderizarListaDocumentos(indice.documentos, t)
    : `<div class="cuerpo-panel" data-ct-exp-documentos-comun><p>${escapar(t(sinPantallaPropia
      ? "ficha_documentos_no_disponibles" : "ficha_documentos_aparte"))}</p>${verDocumentos}</div>`;
  return `<section class="panel ct-exp-ficha-documentos" aria-labelledby="ct-exp-ficha-documentos-titulo">
    <div class="cabecera-panel"><h3 id="ct-exp-ficha-documentos-titulo">${escapar(t("ficha_documentos_titulo"))}</h3>
      ${indice ? `<p class="texto-secundario">${escapar(t("ficha_documentos_recuento", { total: indice.documentos.length }))}</p>` : ""}</div>
    ${cuerpo}
  </section>
  <div data-ct-exp-ancla-firma hidden></div>`;
}

/** Lista de comprobación de documentos; el símbolo acompaña a la palabra. */
export function renderizarListaDocumentos(documentos, t) {
  return `<ul class="lista-documentos" aria-label="${escapar(t("documentos_tabla"))}">${documentos.map((documento) => {
    const enFirma = /firma|sign/iu.test(String(documento.firma ?? "")) && !/firmad|signed/iu.test(String(documento.firma ?? ""));
    const esta = documento.descarga_disponible === true;
    const clase = enFirma ? "firma" : (esta ? "esta" : "falta");
    const simbolo = { firma: "…", esta: "✓", falta: "—" }[clase];
    const palabra = t({ firma: "ficha_documento_firma", esta: "ficha_documento_esta", falta: "ficha_documento_falta" }[clase]);
    return `<li class="${clase}">
      <span class="simbolo" aria-hidden="true">${simbolo}</span>
      <div><strong>${escapar(documento.titulo)}</strong>
        <span class="texto-secundario">${escapar([palabra, documento.estado, documento.firma, documento.fecha].filter(Boolean).join(" · "))}</span></div>
      <span class="texto-secundario">${escapar(documento.tipo)}</span>
    </li>`;
  }).join("")}</ul>`;
}

/** Historial en frases (lo más reciente arriba) y, plegado, el detalle técnico. */
export function renderizarHistorialFicha(expediente, t, faseDeCampo = () => "general") {
  const historial = Array.isArray(expediente.historial) ? expediente.historial : [];
  const frases = historial.length === 0
    ? `<p class="cuerpo-panel">${escapar(t("ficha_historial_vacio"))}</p>`
    : `<ol class="historial-frases">${[...historial].reverse().map((hito) => `<li>
        <div><time>${escapar(hito.fecha)}</time>${escapar(hito.accion)}
          <span class="texto-secundario"> · ${escapar(t("ficha_historial_fase", { fase: hito.fase }))}</span></div>
      </li>`).join("")}</ol>`;
  // La referencia opaca de la bolsa no se muestra nunca como texto.
  const tecnicos = expediente.cabecera.filter((campo) => campo.clave !== "bolsa_cobertura"
    && PATRON_REFERENCIA.test(String(campo.valor ?? "").trim()));
  return `<section class="panel ct-exp-ficha-historial" aria-labelledby="ct-exp-ficha-historial-titulo">
    <div class="cabecera-panel"><h3 id="ct-exp-ficha-historial-titulo">${escapar(t("ficha_historial_titulo"))}</h3></div>
    ${frases}
    <details class="detalle-tecnico-plegado ct-exp-historial">
      <summary>${escapar(t("ficha_detalle_tecnico"))}</summary>
      <dl class="datos-clave">
        <div><dt>${escapar(t("ficha_detalle_version"))}</dt><dd>${escapar(expediente.version)}</dd></div>
        ${expediente.flujo_ref ? `<div><dt>${escapar(t("ficha_detalle_flujo"))}</dt><dd>${escapar(expediente.flujo_ref)} · v${escapar(expediente.flujo_version)}</dd></div>` : ""}
        ${tecnicos.map((campo) => `<div data-ct-exp-campo-fase="${escapar(faseDeCampo(campo.clave))}"><dt>${escapar(campo.etiqueta)}</dt><dd>${escapar(campo.valor)}</dd></div>`).join("")}
      </dl>
      ${historial.length ? `<div class="tabla-contenedor" tabindex="0" role="region" aria-label="${escapar(t("historial_hitos_titulo"))}">
        <table class="tabla-datos ct-exp-tabla-panel">
          <caption>${escapar(t("historial_hitos_titulo"))}</caption>
          <thead><tr><th scope="col">${escapar(t("historial_hito_secuencia"))}</th>
            <th scope="col">${escapar(t("historial_hito_fecha"))}</th>
            <th scope="col">${escapar(t("historial_hito_accion"))}</th>
            <th scope="col">${escapar(t("historial_hito_fase"))}</th>
            <th scope="col">${escapar(t("historial_hito_estado"))}</th></tr></thead>
          <tbody>${historial.map((hito) => `<tr data-ct-exp-hito-fase="${escapar(hito.fase)}" data-ct-exp-hito-accion="${escapar(hito.accion_clave ?? "")}">
            <td>${hito.secuencia}</td><td>${escapar(hito.fecha)}</td>
            <td>${escapar(hito.accion)}</td><td>${escapar(hito.fase)}</td>
            <td>${escapar(hito.estado)}</td></tr>`).join("")}</tbody>
        </table>
      </div>` : ""}
    </details>
  </section>`;
}

/**
 * Datos de la petición: hasta ocho datos clave y, plegados, todos los demás.
 * `valorCampo` pinta el valor (enlace de bolsa incluido) o null si no procede;
 * `faseDeCampo` marca cada dato con su fase para la pantalla de cada fase.
 */
export function renderizarDatosPeticion(expediente, t, { valorCampo, faseDeCampo }) {
  const visibles = expediente.cabecera
    .filter((campo) => !["fase", "estado"].includes(campo.clave))
    .filter((campo) => !PATRON_REFERENCIA.test(String(campo.valor ?? "").trim()) || campo.clave === "bolsa_cobertura")
    .map((campo) => ({ campo, valor: valorCampo(campo) }))
    .filter(({ valor }) => valor !== null);
  const fila = ({ campo, valor }) => `<div data-ct-exp-campo-fase="${escapar(faseDeCampo(campo.clave))}">
      <dt>${escapar(campo.etiqueta)}</dt><dd class="ct-tono-${escapar(campo.tono)}">${valor}</dd></div>`;
  const clave = DATOS_CLAVE.map((c) => visibles.find(({ campo }) => campo.clave === c)).filter(Boolean)
    .slice(0, MAXIMO_DATOS_CLAVE);
  const resto = visibles.filter((item) => !clave.includes(item));
  return `<section class="panel ct-exp-ficha-datos" aria-labelledby="ct-exp-ficha-datos-titulo">
    <div class="cabecera-panel"><h3 id="ct-exp-ficha-datos-titulo">${escapar(t("ficha_datos_titulo"))}</h3></div>
    <dl class="datos-clave">${clave.map(fila).join("")}</dl>
    ${resto.length ? `<details class="detalle-tecnico-plegado">
      <summary>${escapar(t("ficha_datos_todos"))}</summary>
      <dl class="datos-clave">${resto.map(fila).join("")}</dl>
    </details>` : ""}
  </section>`;
}
