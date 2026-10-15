// Este apartado edita únicamente las condiciones que ya publicó la bolsa.
const escapar = (valor) => String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
  .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");

export function renderizarCriteriosOfrecimiento(criterios, textos, deshabilitado = false) {
  if (!criterios) return "";
  const tx = (clave, variables = {}) => {
    let valor = textos[clave] ?? "";
    for (const [nombre, reemplazo] of Object.entries(variables)) valor = valor.replaceAll(`{${nombre}}`, String(reemplazo));
    return escapar(valor);
  };
  const disabled = deshabilitado ? " disabled" : "";
  const modalidades = [...new Set([...(criterios.mejora?.modalidades_destino ?? []),
    ...(criterios.mejora?.condiciones_actuales ?? []).map((c) => c.modalidad)])];
  const causas = [...new Set((criterios.mejora?.condiciones_actuales ?? []).map((c) => c.causa).filter(Boolean))];
  const rotulo = (tipo, clave) => tx(`${tipo}_${clave}`) || tx(`${tipo}_publicada`, {
    numero: (tipo === "modalidad" ? modalidades : causas).indexOf(clave) + 1,
  });
  const destinos = criterios.mejora?.modalidades_destino.map((modalidad) => rotulo("modalidad", modalidad)).join(", ") ?? "";
  const causasDestino = criterios.mejora?.causas_destino?.map((causa) => rotulo("causa", causa)).join(", ") ?? "";
  const limite = (condicion, indice, nombre) => {
    const valor = condicion[nombre];
    if (!valor) return "";
    return `<label class="campo"><span>${tx(nombre)}</span><input type="number" name="criterio_${nombre}_${indice}" min="1" step="1" required value="${escapar(valor.meses)}"${disabled}></label>` +
      `<label class="campo"><span>${tx("incluye_limite")}</span><input type="checkbox" name="criterio_${nombre}_incluye_${indice}"${valor.incluye_limite ? " checked" : ""}${disabled}></label>`;
  };
  const mejora = criterios.mejora ? `<div class="campo campo--ancho"><label><input type="checkbox" name="criterio_mejora_habilitada"${criterios.mejora.habilitada === true ? " checked" : ""}${disabled}> ${tx("mejora_habilitada")}</label></div>` +
    `<div class="campo campo--ancho"><span>${tx("destinos")}</span><p class="rrhh-plazos__valor">${destinos}</p></div>` +
    `<div class="campo campo--ancho"><span>${tx("causas_destino")}</span><p class="rrhh-plazos__valor">${causasDestino}</p></div>` +
    `<label class="campo"><span>${tx("maximo_vacante_meses")}</span><input type="number" name="criterio_maximo_vacante_meses" min="1" max="1200" step="1" required value="${escapar(criterios.mejora.maximo_vacante_meses ?? "")}"${disabled}></label>` +
    `<div class="campo campo--ancho"><span>${tx("reduccion_por_modalidades_previas")}</span><p class="rrhh-plazos__valor">${(criterios.mejora.reduccion_por_modalidades_previas ?? []).map((m) => rotulo("modalidad", m)).join(", ")}</p></div>` +
    criterios.mejora.condiciones_actuales.map((condicion, indice) => `<fieldset class="campo campo--ancho"><legend>${tx("condicion", { numero: indice + 1 })}: ${rotulo("modalidad", condicion.modalidad)}${condicion.causa ? ` · ${rotulo("causa", condicion.causa)}` : ""}</legend><div class="rrhh-plazos__campos">` +
      `<div class="campo"><span>${tx("baja_comprobada")}</span><p class="rrhh-plazos__valor">${tx(condicion.requiere_baja_comprobada ? "si" : "no")}</p></div>` +
      limite(condicion, indice, "meses_transcurridos") + limite(condicion, indice, "meses_duracion_inicial") +
      `</div></fieldset>`).join("") : "";
  return `<section class="panel"><div class="cabecera-panel"><h3>${tx("titulo")}</h3></div><div class="cuerpo-panel rrhh-plazos__campos">` +
    `<label class="campo campo--ancho"><span>${tx("fuente_normativa")}</span><input name="criterio_fuente_normativa" type="text" required maxlength="500" value="${escapar(criterios.fuente_normativa)}"${disabled}></label>` +
    `${mejora}</div></section>`;
}

export function cambiarCriteriosOfrecimiento(criterios, control) {
  if (!criterios || !control?.name?.startsWith("criterio_")) return false;
  if (control.name === "criterio_fuente_normativa") criterios.fuente_normativa = control.value.trim();
  else if (control.name === "criterio_maximo_vacante_meses" && criterios.mejora)
    criterios.mejora.maximo_vacante_meses = control.value === "" ? 0 : Number(control.value);
  else if (control.name === "criterio_mejora_habilitada" && criterios.mejora) criterios.mejora.habilitada = control.checked;
  else {
    const coincidencia = /^criterio_(meses_transcurridos|meses_duracion_inicial)(_incluye)?_(\d+)$/u.exec(control.name);
    if (!coincidencia) return false;
    const condicion = criterios.mejora?.condiciones_actuales[Number(coincidencia[3])];
    const limite = condicion?.[coincidencia[1]];
    if (!limite) return false;
    if (coincidencia[2]) limite.incluye_limite = control.checked;
    else limite.meses = control.value === "" ? 0 : Number(control.value);
  }
  return true;
}
