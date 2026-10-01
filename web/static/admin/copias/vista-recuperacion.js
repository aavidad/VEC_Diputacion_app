import { TEXTOS_COPIAS, traducirCopias as t, traducirCodigo as tc } from "./i18n.js?v=20261001-cs09-copias-v1";

export function etiquetaOpcion(opciones, grupo, ref) {
  const opcion = opciones?.[grupo]?.find(o => o.ref === ref);
  return opcion && Object.hasOwn(TEXTOS_COPIAS.seccion("opciones"), opcion.clave_i18n)
    ? TEXTOS_COPIAS.seccion("opciones")[opcion.clave_i18n] : null;
}
export function resumenRevisable(propuesta, opciones) {
  return propuesta?.doble_control === true && propuesta?.copia_previa_requerida === true
    && !!propuesta.preimagen_sha256 && !!propuesta.conjunto_huella_sha256
    && Date.parse(propuesta.ventana_fin) > Date.parse(propuesta.ventana_inicio) && Date.parse(propuesta.caduca_en) > Date.now()
    && !!etiquetaOpcion(opciones, "destinos", propuesta.destino_ref)
    && !!etiquetaOpcion(opciones, "motivos", propuesta.motivo_ref)
    && !!etiquetaOpcion(opciones, "ventanas", propuesta.ventana_ref);
}
export function pintarRecuperacion({ cuerpo, estado, cliente, s, operar, recibido, pintar, cargarRestauracion }) {
  const { crear, boton, panel, campo, dato, chip, fecha } = s;
  const { p, h, c } = panel("restauracion"); cuerpo.append(p);
  h.append(boton("actualizar", cargarRestauracion, { disabled: estado.busy }));
  const copia = estado.seleccion, opciones = estado.opciones;
  const disponibles = opciones && ["destinos", "motivos", "ventanas"].every(k => opciones[k].some(o => etiquetaOpcion(opciones, k, o.ref)));
  if (estado.propuestaNueva) {
    const nueva = estado.propuestaNueva, dl = crear("dl", "", "copias-datos");
    const titulo = crear("h4", t("revisar_propuesta")); titulo.tabIndex = -1; c.append(titulo);
    dato(dl, "fecha", fecha(copia.iniciada_en));
    for (const [k, grupo] of [["destino", "destinos"], ["motivo", "motivos"], ["ventana", "ventanas"]]) dato(dl, k, etiquetaOpcion(opciones, grupo, nueva[k + "_ref"]));
    for (const k of ["ventana_inicio", "ventana_fin", "caduca_en"]) dato(dl, k === "caduca_en" ? "caduca" : k, fecha(nueva[k]));
    c.append(dl, crear("p", t("perdidas_texto")), crear("p", t("segunda_texto")), crear("p", t("previa_texto")),
      boton("confirmar_propuesta", () => operar(signal => cliente.proponer(nueva, { signal }), propuesta => {
        estado.propuestaNueva = null; estado.borradorPropuesta = null; estado.propuestaSeleccionada = propuesta;
        estado.propuestas = [...(estado.propuestas ?? []).filter(p => p.propuesta_ref !== propuesta.propuesta_ref), propuesta];
      }), { clase: "boton-peligro", disabled: estado.busy || !estado.capacidades.proponer }),
      boton("cambiar", () => { estado.propuestaNueva = null; pintar(); }, { disabled: estado.busy }));
    return;
  }
  if (copia?.estado === "valida" && copia.compatibilidad.estado === "compatible" && disponibles && estado.capacidades.proponer) {
    const form = crear("form", "", "copias-formulario"), campos = crear("div", "", "copias-campos"), entradas = {};
    form.noValidate = true;
    const errores = crear("div", "", "copias-errores"); errores.setAttribute("role", "alert");
    estado.borradorPropuesta ??= { conjunto_ref: copia.copia_ref };
    if (estado.borradorPropuesta.conjunto_ref !== copia.copia_ref) estado.borradorPropuesta = { conjunto_ref: copia.copia_ref };
    const draft = estado.borradorPropuesta;
    for (const [k, grupo] of [["destino", "destinos"], ["motivo", "motivos"], ["ventana", "ventanas"]]) {
      const label = crear("label", t(k), "copias-campo"), select = crear("select"); select.name = k + "_ref"; select.setAttribute("aria-label", t(k)); select.required = true; select.disabled = estado.busy;
      select.append(crear("option", t("elegir")));
      for (const opcion of opciones[grupo]) {
        const etiqueta = etiquetaOpcion(opciones, grupo, opcion.ref); if (!etiqueta) continue;
        const o = crear("option", etiqueta); o.value = opcion.ref; select.append(o);
      }
      select.firstElementChild.value = ""; select.value = draft[k + "_ref"] ?? "";
      select.addEventListener("change", () => { draft[k + "_ref"] = select.value; }); label.append(select); campos.append(label); entradas[k + "_ref"] = select;
    }
    for (const clave of ["ventana_inicio", "ventana_fin", "caduca_en"]) {
      const { label, input } = campo(clave + "_utc", "datetime-local", draft[clave]); input.disabled = estado.busy;
      input.addEventListener("input", () => { draft[clave] = input.value; }); campos.append(label); entradas[clave] = input;
    }
    form.append(crear("p", fecha(copia.iniciada_en)), errores, campos, crear("p", t("perdidas_texto")), crear("p", t("segunda_texto")), crear("p", t("previa_texto")));
    form.addEventListener("submit", e => {
      e.preventDefault(); errores.replaceChildren();
      const invalidos = Object.entries(entradas).filter(([, input]) => !input.validity.valid);
      for (const [k, input] of Object.entries(entradas)) input.setAttribute("aria-invalid", String(invalidos.some(([clave]) => clave === k)));
      if (invalidos.length) { errores.append(crear("p", t("errores"))); invalidos[0][1].focus(); return; }
      const nueva = { operacion_ref: "operacion:" + globalThis.crypto.randomUUID(), conjunto_ref: copia.copia_ref,
        destino_ref: draft.destino_ref, motivo_ref: draft.motivo_ref, ventana_ref: draft.ventana_ref };
      for (const clave of ["ventana_inicio", "ventana_fin", "caduca_en"]) nueva[clave] = new Date(draft[clave] + "Z").toISOString();
      if (Date.parse(nueva.ventana_inicio) >= Date.parse(nueva.ventana_fin) || Date.parse(nueva.caduca_en) <= Date.now()) { errores.append(crear("p", t("error_ventana"))); entradas.ventana_fin.focus(); return; }
      estado.propuestaNueva = nueva; pintar(); cuerpo.querySelector("h4")?.focus();
    });
    const enviar = crear("button", t("revisar_propuesta"), "boton-peligro"); enviar.type = "submit"; enviar.disabled = estado.busy; form.append(enviar); c.append(form);
  } else c.append(crear("p", t(estado.capacidades.proponer === false ? "sin_permiso" : !copia ? "seleccionar" : "propuesta_bloqueada"), "copias-nota"));
  const lista = crear("div", "", "copias-propuestas"); c.append(crear("h4", t("propuestas")), lista);
  if (!estado.propuestas?.length) lista.append(crear("p", t("sin_propuestas")));
  for (const propuesta of estado.propuestas ?? []) {
    const bloque = crear("details"), cabecera = crear("summary"); cabecera.append(crear("span", fecha(propuesta.ventana_inicio)), chip(propuesta.estado));
    bloque.open = estado.propuestaSeleccionada?.propuesta_ref === propuesta.propuesta_ref;
    const dl = crear("dl", "", "copias-datos");
    for (const [k, grupo] of [["destino", "destinos"], ["motivo", "motivos"], ["ventana", "ventanas"]]) dato(dl, k, etiquetaOpcion(opciones, grupo, propuesta[k + "_ref"]));
    dato(dl, "caduca", fecha(propuesta.caduca_en));
    if (propuesta.perdida_desde) dato(dl, "perdida_desde", fecha(propuesta.perdida_desde));
    if (propuesta.perdida_hasta) dato(dl, "perdida_hasta", fecha(propuesta.perdida_hasta));
    const tecnico = crear("details"), refs = crear("dl", "", "copias-datos");
    dato(refs, "preimagen", propuesta.preimagen_sha256); dato(refs, "referencia", propuesta.conjunto_ref); dato(refs, "huella", propuesta.huella_sha256);
    tecnico.append(crear("summary", t("tecnico")), refs);
    bloque.append(cabecera, dl, crear("h4", t("perdidas")), crear("p", t("perdidas_texto")), tecnico, crear("p", t("segunda_texto")), crear("p", t("previa_texto")));
    const revisable = resumenRevisable(propuesta, opciones);
    if (!revisable) bloque.append(crear("p", t("resumen_incompleto"), "copias-nota"));
    const confirmar = crear("label", t("confirmar_revision"), "copias-check"), check = crear("input"); check.type = "checkbox"; confirmar.prepend(check);
    check.disabled = estado.busy || !revisable;
    const control = accion => {
      const key = propuesta.propuesta_ref + ":" + accion;
      estado.clavesControl ??= new Map();
      if (!estado.clavesControl.has(key)) estado.clavesControl.set(key, "operacion:" + globalThis.crypto.randomUUID());
      const solicitud = { operacion_ref: estado.clavesControl.get(key), destino_ref: propuesta.destino_ref, propuesta_huella_sha256: propuesta.huella_sha256, version_esperada: propuesta.version };
      estado.propuestaSeleccionada = propuesta;
      operar(signal => cliente[accion](propuesta.propuesta_ref, solicitud, { signal }), respuesta => {
        if (accion === "ejecutar") recibido(respuesta);
        else { estado.propuestaSeleccionada = respuesta; estado.propuestas = estado.propuestas.map(p => p.propuesta_ref === respuesta.propuesta_ref ? respuesta : p); }
      });
    };
    const aprobar = boton("aprobar", () => { if (check.checked) control("revisar"); }, { clase: "boton-peligro", disabled: true });
    const ejecutar = boton("ejecutar", () => { if (check.checked) control("ejecutar"); }, { clase: "boton-peligro", disabled: true });
    check.addEventListener("change", () => { aprobar.disabled = !check.checked || !revisable || estado.busy || !estado.capacidades.revisar || ["revisada", "aprobada"].includes(propuesta.estado);
      ejecutar.disabled = !check.checked || !revisable || estado.busy || !estado.capacidades.ejecutar || propuesta.estado !== "aprobada"; });
    bloque.append(confirmar, aprobar, ejecutar);
    if (!estado.capacidades.ejecutar) bloque.append(crear("p", t("ejecutar_bloqueado"), "copias-nota"));
    lista.append(bloque);
  }
}
