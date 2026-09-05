import { escaparHTML as e } from "./componentes-expedientes.js";
import { CAMPOS_SELECCION, CAMPOS_COMUNICACION,
  CAMPOS_RESPUESTA_RECIBIDA, CAMPOS_RESPUESTA_EDITABLES } from "./contrato-llamamiento.js";

export function renderizarLlamamiento(estado, t, fecha) {
  function campo(operacion, nombre, valor, bloqueado) {
    const id = `ct-llamamiento-${operacion}-${nombre}`;
    const numero = nombre === "version_esperada" || nombre === "version_comunicacion_esperada";
    if (nombre === "respuesta") return `<div class="ct-campo">
      <label for="${id}">${e(t("llamamiento_respuesta_declarada"))} *</label>
      <select id="${id}" name="respuesta" required${bloqueado ? " disabled" : ""}
        aria-describedby="ct-llamamiento-respuesta-ayuda">
        <option value="">${e(t("seleccionar"))}</option>
        ${["aceptacion", "renuncia"].map((opcion) => `<option value="${opcion}"
          ${valor === opcion ? "selected" : ""}>${e(t("llamamiento_respuesta_" + opcion))}</option>`).join("")}
      </select></div>`;
    const recepcion = nombre === "recibida_en";
    const tipo = numero ? 'type="number" min="1" max="9007199254740990" step="1"'
      : recepcion ? 'type="datetime-local" step="0.000001"'
      : `type="text" maxlength="${nombre === "correo_sha256" ? 64 : 160}"`;
    return `<div class="ct-campo"><label for="${id}">${e(t("llamamiento_" + nombre))} *</label>
      <input id="${id}" name="${nombre}" value="${e(recepcion ? String(valor).replace(/Z$/u, "") : valor)}"
        ${tipo}
        required autocomplete="off" spellcheck="false"${bloqueado ? " readonly" : ""}
        aria-describedby="ct-llamamiento-${operacion}-ayuda">
      </div>`;
  }
  function recibo(operacion, datos) {
    if (!datos) return `<p class="ct-ayuda">${e(t("llamamiento_sin_recibo"))}</p>`;
    const campos = operacion === "respuesta"
      ? [...CAMPOS_RESPUESTA_RECIBIDA, "justificante_ref", "recibo_ref", "auditoria_ref", "registrada_en", "estado"]
      : operacion === "seleccion"
      ? ["recibo_ref", "confirmada_en", "organizacion_ref", "llamamiento_ref", "version_llamamiento"]
      : ["comunicacion_ref", "recibo_ref", "auditoria_ref",
        "version_resultante", "estado_local", ...(Object.hasOwn(datos, "registrada_en")
          ? ["registrada_en", "intencion_envio_ref"] : ["respuesta_hasta"])];
    return `<section class="ct-recibo" data-ct-llamamiento-recibo="${operacion}"
      aria-labelledby="ct-llamamiento-recibo-${operacion}" tabindex="-1">
      <h4 id="ct-llamamiento-recibo-${operacion}">${e(t(operacion === "respuesta"
        ? "llamamiento_respuesta_recibo" : "llamamiento_recibo"))}</h4>
      <p>${e(t("llamamiento_recibo_" + operacion + "_ayuda"))}</p>
      <dl>${campos.map((nombre) => {
        let valor = datos[nombre];
        if (["confirmada_en", "respuesta_hasta", "registrada_en", "recibida_en"].includes(nombre)) {
          valor = operacion === "respuesta" ? datos[nombre] : fecha.format(new Date(valor));
        } else if (nombre === "estado_local" || nombre === "estado") valor = t("llamamiento_" + valor);
        else if (nombre === "respuesta") valor = t("llamamiento_respuesta_" + valor);
        return `<div><dt>${e(t(nombre === "respuesta" ? "llamamiento_respuesta_declarada"
          : "llamamiento_" + nombre))}</dt><dd>${e(valor)}</dd></div>`;
      }).join("")}</dl>
    </section>`;
  }
  function formulario(operacion, campos) {
    const paso = estado[operacion];
    const titulo = t("llamamiento_" + operacion);
    return `<section class="ct-llamamiento-paso" aria-labelledby="ct-llamamiento-${operacion}-titulo">
      <h3 id="ct-llamamiento-${operacion}-titulo">${e(titulo)}</h3>
      <form data-ct-llamamiento-form="${operacion}" novalidate aria-busy="${paso.ocupado || paso.calculando}">
        <p id="ct-llamamiento-${operacion}-ayuda">${e(t(operacion === "seleccion"
          ? "llamamiento_clave_ayuda" : operacion === "respuesta"
            ? "llamamiento_respuesta_ayuda" : "llamamiento_prueba_ayuda"))}</p>
        ${operacion !== "seleccion" ? `<p>${e(t("llamamiento_clave_ayuda"))}</p>` : ""}
        <fieldset${paso.ocupado || paso.calculando ? " disabled" : ""}>
          <legend>${e(t("llamamiento_contexto"))}</legend>
          <div class="ct-campos">${campos.map((nombre) => campo(
            operacion, nombre, paso.valores[nombre] ?? "", paso.solicitud !== null
              || (operacion === "comunicacion" && nombre !== "clave_idempotencia")
              || (operacion === "respuesta" && !CAMPOS_RESPUESTA_EDITABLES.includes(nombre)),
          )).join("")}</div>
        ${operacion === "respuesta" ? `<div class="ct-campo">
          <label for="ct-llamamiento-correo">${e(t("llamamiento_correo_archivo"))} *</label>
          <input id="ct-llamamiento-correo" type="file" accept=".eml" data-ct-llamamiento-correo
            ${paso.solicitud !== null ? "disabled" : ""} aria-describedby="ct-llamamiento-correo-ayuda">
          <p id="ct-llamamiento-correo-ayuda" class="ct-ayuda">${e(t("llamamiento_correo_ayuda"))}</p>
          ${paso.valores.correo_sha256 ? `<p class="ct-ayuda" data-ct-llamamiento-huella-calculada>
            ${e(t("llamamiento_correo_huella_conservada"))}</p>` : ""}
        </div>` : ""}
        </fieldset>
        <div class="ct-acciones">
        ${paso.solicitud === null ? `<button class="boton-secundario" type="button"
          data-ct-llamamiento-clave="${operacion}"${paso.calculando ? " disabled" : ""}>${e(t("llamamiento_crear_clave"))}</button>` : ""}
        ${!paso.recibo && !paso.bloqueado ? `<button class="boton-primario" type="submit"
          ${paso.ocupado || paso.calculando ? "disabled" : ""}>${e(t(paso.solicitud !== null
            ? "llamamiento_recuperar"
            : operacion === "seleccion" ? "llamamiento_seleccionar"
              : operacion === "respuesta" ? "llamamiento_registrar_respuesta" : "llamamiento_registrar"))}</button>` : ""}
        </div>
      </form>
      <div class="ct-estado ct-estado-${paso.tono}" data-ct-llamamiento-estado="${operacion}"
        role="${paso.tono === "error" ? "alert" : "status"}"
        aria-live="polite" aria-atomic="true" tabindex="-1">${e(t(paso.mensaje))}</div>
      ${recibo(operacion, paso.recibo)}
    </section>`;
  }
  return `<section class="ct-alta ct-llamamiento" data-ct-llamamiento
    aria-labelledby="ct-llamamiento-titulo">
    <header class="ct-cabecera"><div>
      <p class="sobrelinea">${e(t("llamamiento_sobrelinea"))}</p>
      <h2 id="ct-llamamiento-titulo">${e(t("llamamiento_titulo"))}</h2>
      <p>${e(t("llamamiento_descripcion"))}</p>
    </div><aside class="ct-alcance">${e(t("llamamiento_alcance"))}</aside></header>
    <p class="ct-exp-mensaje ct-tono-informacion" data-ct-llamamiento-contexto>${e(t(
      estado.enlazado ? "llamamiento_contexto_enlazado" : "llamamiento_contexto_manual",
    ))}</p>
    ${formulario("seleccion", CAMPOS_SELECCION)}
    <details data-ct-llamamiento-comunicacion${estado.comunicacionAbierta ? " open" : ""}>
      <summary>${e(t("llamamiento_comunicacion"))}</summary>
      ${estado.seleccion.recibo ? formulario("comunicacion", CAMPOS_COMUNICACION)
        : `<p role="status">${e(t("llamamiento_espera_seleccion"))}</p>`}
    </details>
    ${estado.comunicacion.recibo?.version_resultante === 2
      ? formulario("respuesta", CAMPOS_RESPUESTA_RECIBIDA) : ""}
  </section>`;
}
