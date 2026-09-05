import { escaparHTML as e } from "./componentes-expedientes.js";
import { CAMPOS_SELECCION, CAMPOS_COMUNICACION } from "./contrato-llamamiento.js";

export function renderizarLlamamiento(estado, t, fecha) {
  function campo(operacion, nombre, valor, bloqueado) {
    const id = `ct-llamamiento-${operacion}-${nombre}`;
    const numero = nombre === "version_esperada";
    return `<div class="ct-campo"><label for="${id}">${e(t("llamamiento_" + nombre))} *</label>
      <input id="${id}" name="${nombre}" value="${e(valor)}"
        ${numero ? 'type="number" min="1" max="9007199254740990" step="1"' : 'type="text" maxlength="160"'}
        required autocomplete="off" spellcheck="false"${bloqueado ? " readonly" : ""}
        aria-describedby="ct-llamamiento-${operacion}-ayuda">
      </div>`;
  }
  function recibo(operacion, datos) {
    if (!datos) return `<p class="ct-ayuda">${e(t("llamamiento_sin_recibo"))}</p>`;
    const campos = operacion === "seleccion"
      ? ["recibo_ref", "confirmada_en", "organizacion_ref", "llamamiento_ref", "version_llamamiento"]
      : ["comunicacion_ref", "recibo_ref", "auditoria_ref",
        "version_resultante", "estado_local", ...(Object.hasOwn(datos, "registrada_en")
          ? ["registrada_en", "intencion_envio_ref"] : ["respuesta_hasta"])];
    return `<section class="ct-recibo" data-ct-llamamiento-recibo="${operacion}"
      aria-labelledby="ct-llamamiento-recibo-${operacion}" tabindex="-1">
      <h4 id="ct-llamamiento-recibo-${operacion}">${e(t("llamamiento_recibo"))}</h4>
      <p>${e(t("llamamiento_recibo_" + operacion + "_ayuda"))}</p>
      <dl>${campos.map((nombre) => {
        let valor = datos[nombre];
        if (["confirmada_en", "respuesta_hasta", "registrada_en"].includes(nombre)) {
          valor = fecha.format(new Date(valor));
        } else if (nombre === "estado_local") valor = t("llamamiento_" + valor);
        return `<div><dt>${e(t("llamamiento_" + nombre))}</dt><dd>${e(valor)}</dd></div>`;
      }).join("")}</dl>
    </section>`;
  }
  function formulario(operacion, campos) {
    const paso = estado[operacion];
    const titulo = t("llamamiento_" + operacion);
    return `<section class="ct-llamamiento-paso" aria-labelledby="ct-llamamiento-${operacion}-titulo">
      <h3 id="ct-llamamiento-${operacion}-titulo">${e(titulo)}</h3>
      <form data-ct-llamamiento-form="${operacion}" novalidate aria-busy="${paso.ocupado}">
        <p id="ct-llamamiento-${operacion}-ayuda">${e(t(operacion === "seleccion"
          ? "llamamiento_clave_ayuda" : "llamamiento_prueba_ayuda"))}</p>
        ${operacion === "comunicacion" ? `<p>${e(t("llamamiento_clave_ayuda"))}</p>` : ""}
        <fieldset${paso.ocupado ? " disabled" : ""}>
          <legend>${e(t("llamamiento_contexto"))}</legend>
          <div class="ct-campos">${campos.map((nombre) => campo(
            operacion, nombre, paso.valores[nombre] ?? "", paso.solicitud !== null
              || (operacion === "comunicacion" && nombre !== "clave_idempotencia"),
          )).join("")}</div>
        </fieldset>
        <div class="ct-acciones">
        ${paso.solicitud === null ? `<button class="boton-secundario" type="button"
          data-ct-llamamiento-clave="${operacion}">${e(t("llamamiento_crear_clave"))}</button>` : ""}
        ${!paso.recibo && !paso.bloqueado ? `<button class="boton-primario" type="submit"
          ${paso.ocupado ? "disabled" : ""}>${e(t(paso.solicitud !== null
            ? "llamamiento_recuperar"
            : operacion === "seleccion" ? "llamamiento_seleccionar" : "llamamiento_registrar"))}</button>` : ""}
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
  </section>`;
}
