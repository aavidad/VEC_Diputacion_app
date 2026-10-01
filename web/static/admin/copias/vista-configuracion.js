import { traducirCopias as t } from "./i18n.js?v=20261001-cs09-copias-ux-v2";

const DIAS = ["domingo", "lunes", "martes", "miercoles", "jueves", "viernes", "sabado"];
export function pintarConfiguracion({ cuerpo, estado, cliente, s, operar, recibido, pintar, cargarConfiguracion }) {
  const { crear, boton, panel, campo, dato, numero } = s;
  const { p, h, c } = panel("calendario"); cuerpo.append(p); h.append(boton("actualizar", () => {
    estado.configuracion = null; estado.borradorConfiguracion = null;
    pintar(); cargarConfiguracion();
  }, { disabled: estado.busy }));
  if (!estado.configuracion) { c.append(crear("p", t("fuente_pendiente"))); return; }
  const config = estado.configuracion, politica = config.politica;
  if (!estado.borradorConfiguracion) estado.borradorConfiguracion = structuredClone(politica);
  const draft = estado.borradorConfiguracion;
  if (estado.revisionConfiguracion) {
    const revision = estado.revisionConfiguracion, dl = crear("dl", "", "copias-datos");
    const titulo = crear("h4", t("confirmar_config")); titulo.tabIndex = -1; c.append(titulo);
    if (revision.ambito === "calendario") {
      dato(dl, "fecha_inicial", revision.politica.fecha_inicial); dato(dl, "cada_dias", numero(revision.politica.cada_dias));
      dato(dl, "zona_horaria", revision.politica.zona_horaria); dato(dl, "ventana_inicio", revision.politica.ventana.inicio); dato(dl, "ventana_fin", revision.politica.ventana.fin);
      dato(dl, "dias_semana", revision.politica.dias_semana.map(d => t(DIAS[d])).join(", ") || t("todas"));
    } else {
      dato(dl, "conservar_minimo", numero(revision.politica.retencion.conservar_minimo)); dato(dl, "edad_maxima_dias", numero(revision.politica.retencion.edad_maxima_dias));
      dato(dl, "doble_control", t(revision.politica.retencion.doble_control ? "si" : "no"));
      dato(dl, "borrado_permitido", t(revision.politica.retencion.borrado_permitido ? "si" : "no"));
      c.append(crear("p", t("no_borrado")));
    }
    c.append(dl, crear("p", t("sin_cambios"), "copias-nota"), boton("confirmar_guardado", () => operar(signal => cliente.guardarConfiguracion(revision.ambito, {
      operacion_ref: revision.operacion_ref, version_esperada: config.version, politica: revision.politica,
    }, { signal }), recibo => { recibido(recibo); estado.revisionConfiguracion = null; estado.borradorConfiguracion = null; estado.configuracion = null;
      cargarConfiguracion();
    }), { clase: "boton-primario", disabled: estado.busy }), boton("cambiar", () => { estado.revisionConfiguracion = null; pintar(); }, { disabled: estado.busy }));
    return;
  }
  const form = crear("form", "", "copias-formulario"), errores = crear("div", "", "copias-errores"); errores.setAttribute("role", "alert");
  const campos = crear("div", "", "copias-campos"), inputs = {};
  function entrada(clave, tipo, valor, opciones = {}) {
    const { label, input } = campo(clave, tipo, valor, opciones); input.id = "copias-config-" + clave; inputs[clave] = input;
    input.disabled = estado.busy || !estado.capacidades.configurar_calendario;
    input.addEventListener("input", () => { draft[clave] = tipo === "number" ? Number(input.value) : input.value; });
    input.addEventListener("blur", () => { validar("calendario", false); }); campos.append(label); return input;
  }
  entrada("fecha_inicial", "date", draft.fecha_inicial);
  entrada("cada_dias", "number", draft.cada_dias, { min: 1, max: 366 });
  const inicio = entrada("ventana_inicio", "time", draft.ventana.inicio), fin = entrada("ventana_fin", "time", draft.ventana.fin);
  inicio.addEventListener("input", () => { draft.ventana.inicio = inicio.value; }); fin.addEventListener("input", () => { draft.ventana.fin = fin.value; });
  const zona = crear("dl", "", "copias-datos"); dato(zona, "zona_horaria", politica.zona_horaria);
  const semana = crear("fieldset", "", "copias-checks"); semana.append(crear("legend", t("dias_semana")));
  DIAS.forEach((clave, indice) => {
    const label = crear("label", t(clave), "copias-check"), input = crear("input"); input.type = "checkbox"; input.name = "dia_" + indice; input.checked = draft.dias_semana.includes(indice);
    input.disabled = estado.busy || !estado.capacidades.configurar_calendario;
    input.addEventListener("change", () => { draft.dias_semana = input.checked ? [...draft.dias_semana, indice].sort() : draft.dias_semana.filter(d => d !== indice); });
    label.prepend(input); semana.append(label);
  });
  form.append(errores, zona, campos, semana, crear("p", t("dias_ayuda"), "copias-nota"));
  const retencion = crear("fieldset", "", "copias-formulario"); retencion.append(crear("legend", t("guardar_retencion")));
  const retenCampos = crear("div", "", "copias-campos");
  for (const clave of ["conservar_minimo", "edad_maxima_dias"]) {
    const { label, input } = campo(clave, "number", draft.retencion[clave], { min: 1 }); input.id = "copias-config-" + clave; inputs[clave] = input;
    input.disabled = estado.busy || !estado.capacidades.configurar_retencion;
    input.addEventListener("input", () => { draft.retencion[clave] = Number(input.value); }); input.addEventListener("blur", () => validar("retencion", false)); retenCampos.append(label);
  }
  retencion.append(retenCampos);
  for (const clave of ["borrado_permitido", "doble_control"]) {
    const label = crear("label", t(clave), "copias-check"), input = crear("input"); input.type = "checkbox"; input.name = clave; input.checked = draft.retencion[clave];
    input.disabled = estado.busy || !estado.capacidades.configurar_retencion;
    input.addEventListener("change", () => { draft.retencion[clave] = input.checked; }); label.prepend(input); retencion.append(label);
  }
  const protegidas = crear("dl", "", "copias-datos"); dato(protegidas, "protegidas", numero(politica.retencion.protegidas.length));
  retencion.append(crear("p", t("doble_control_ayuda"), "copias-nota"), protegidas);
  form.append(retencion);
  function validar(ambito, mostrar = true) {
    const problemas = [];
    if (ambito === "calendario") {
      if (!inputs.fecha_inicial.validity.valid) problemas.push(["fecha_inicial", "error_fecha"]);
      if (!Number.isInteger(draft.cada_dias) || draft.cada_dias < 1 || draft.cada_dias > 366) problemas.push(["cada_dias", "error_intervalo"]);
      if (!draft.ventana.inicio || !draft.ventana.fin || draft.ventana.inicio >= draft.ventana.fin) problemas.push(["ventana_fin", "error_ventana"]);
    } else for (const k of ["conservar_minimo", "edad_maxima_dias"]) if (!Number.isSafeInteger(draft.retencion[k]) || draft.retencion[k] < 1) problemas.push([k, "error_numero"]);
    for (const [k, input] of Object.entries(inputs)) input.setAttribute("aria-invalid", String(problemas.some(p => p[0] === k)));
    if (mostrar) {
      errores.replaceChildren(); if (problemas.length) errores.append(crear("p", t("errores")));
      for (const [k, mensaje] of problemas) {
        const a = crear("a", t(mensaje)); a.href = "#" + inputs[k].id; a.addEventListener("click", e => { e.preventDefault(); inputs[k].focus(); }); errores.append(a);
      }
      if (problemas.length) inputs[problemas[0][0]].focus();
    }
    return problemas.length === 0;
  }
  function revisar(ambito) {
    if (!validar(ambito)) return;
    const nueva = structuredClone(politica);
    if (ambito === "calendario") Object.assign(nueva, { fecha_inicial: draft.fecha_inicial, cada_dias: draft.cada_dias, dias_semana: draft.dias_semana, ventana: draft.ventana });
    else nueva.retencion = draft.retencion;
    estado.revisionConfiguracion = { ambito, politica: nueva, operacion_ref: "operacion:" + globalThis.crypto.randomUUID() }; pintar();
    cuerpo.querySelector("h4")?.focus();
  }
  form.addEventListener("submit", e => e.preventDefault());
  const acciones = crear("div", "", "copias-acciones");
  for (const ambito of ["calendario", "retencion"]) acciones.append(boton("guardar_" + ambito, () => revisar(ambito), { clase: "boton-primario", disabled: estado.busy || !estado.capacidades["configurar_" + ambito] }));
  if (estado.capacidades.configurar_calendario === false || estado.capacidades.configurar_retencion === false) form.append(crear("p", t("sin_permiso"), "copias-nota"));
  form.append(acciones, crear("p", t("sin_cambios"), "copias-nota")); c.append(form);
}
