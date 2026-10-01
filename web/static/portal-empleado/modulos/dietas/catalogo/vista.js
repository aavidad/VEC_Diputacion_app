/** Consulta de tarifas y preparación local. La composición aporta fuente y textos. */
let numeroInstancias = 0;
function nodo(documento, etiqueta, texto) {
  const elemento = documento.createElement(etiqueta);
  if (texto !== undefined) elemento.textContent = String(texto);
  return elemento;
}

function fechaValida(valor) {
  if (typeof valor !== "string" || !/^\d{4}-\d{2}-\d{2}$/u.test(valor)) return false;
  const fecha = new Date(`${valor}T00:00:00Z`);
  return Number.isFinite(fecha.getTime()) && fecha.toISOString().slice(0, 10) === valor;
}

function textoValido(valor, maximo = 300) {
  return typeof valor === "string" && valor.length > 0 && valor.length <= maximo &&
    valor.trim() === valor && !/[\x00-\x1f\x7f]/u.test(valor);
}

function idiomaMotivoValido(valor) {
  if (typeof valor !== "string" || !/^[a-z]{2,3}(?:-[A-Z]{2})?$/u.test(valor)) return false;
  try { return Intl.getCanonicalLocales(valor).length === 1; }
  catch { return false; }
}

function fuenteHTTPS(valor) {
  if (!textoValido(valor, 500)) return false;
  try {
    const url = new URL(valor);
    return url.protocol === "https:" && Boolean(url.hostname) && !url.username && !url.password;
  } catch { return false; }
}

export function catalogoTarifasDietasValido(catalogo) {
  if (!catalogo || typeof catalogo !== "object" || Array.isArray(catalogo) ||
      !textoValido(catalogo.version, 120) || catalogo.estado !== "ejemplo" ||
      !textoValido(catalogo.referencia_catalogo, 120) ||
      !textoValido(catalogo.version_tarifa_ref, 120) ||
      !fechaValida(catalogo.vigente_desde) ||
      (catalogo.vigente_hasta !== null && (!fechaValida(catalogo.vigente_hasta) || catalogo.vigente_hasta <= catalogo.vigente_desde)) ||
      !Array.isArray(catalogo.fuentes) || catalogo.fuentes.length === 0 || catalogo.fuentes.length > 10 ||
      catalogo.fuentes.some((fuente) => !fuenteHTTPS(fuente)) ||
      !Array.isArray(catalogo.tarifas) || catalogo.tarifas.length > 100 ||
      !Array.isArray(catalogo.historia) || catalogo.historia.length > 100) return false;
  const ids = new Set();
  for (const tarifa of catalogo.tarifas) {
    if (!textoValido(tarifa?.id, 120) || ids.has(tarifa.id) || !textoValido(tarifa.concepto, 80) ||
        !(tarifa.grupo === null || (Number.isSafeInteger(tarifa.grupo) && tarifa.grupo > 0)) ||
        !/^[A-Z]{2}$/u.test(tarifa.pais || "") ||
        !/^[A-Z]{3}$/u.test(tarifa.moneda || "") ||
        !Number.isSafeInteger(tarifa.importe_centimos) || tarifa.importe_centimos < 0 ||
        !["importe", "kilometro"].includes(tarifa.unidad)) return false;
    ids.add(tarifa.id);
  }
  return catalogo.historia.every((entrada) =>
    entrada?.estado === "ejemplo" && textoValido(entrada.actor, 100) &&
    textoValido(entrada.accion, 80) &&
    idiomaMotivoValido(entrada.idioma_motivo) &&
    textoValido(entrada.motivo, 500) && fechaValida(entrada.fecha) &&
    textoValido(entrada.version, 120));
}

/** Convierte un decimal escrito según la localización, sin pasar por punto flotante. */
export function importeACentimos(valor, localizacion) {
  if (typeof valor !== "string" || typeof localizacion !== "string") return null;
  const separador = new Intl.NumberFormat(localizacion).formatToParts(1.1).find((p) => p.type === "decimal")?.value;
  if (!separador || valor !== valor.trim()) return null;
  const partes = valor.split(separador);
  if (partes.length > 2 || !/^\d{1,9}$/u.test(partes[0]) ||
      (partes.length === 2 && !/^\d{1,2}$/u.test(partes[1]))) return null;
  const entero = Number(partes[0]);
  const fraccion = Number((partes[1] || "").padEnd(2, "0"));
  const centimos = entero * 100 + fraccion;
  return Number.isSafeInteger(centimos) ? centimos : null;
}

/** Presenta todos los céntimos seguros sin redondearlos al convertir a Number decimal. */
export function formatearImporteCentimos(centimos, localizacion, codigo) {
  if (!Number.isSafeInteger(centimos) || centimos < 0) throw new TypeError("importe no valido");
  const exacto = BigInt(centimos);
  const entero = exacto / 100n;
  const resto = Number(exacto % 100n);
  const partes = new Intl.NumberFormat(localizacion, {
    style: "currency", currency: codigo, currencyDisplay: "code",
    minimumFractionDigits: 0, maximumFractionDigits: 0,
  }).formatToParts(entero);
  const separador = new Intl.NumberFormat(localizacion).formatToParts(1.1)
    .find((parte) => parte.type === "decimal")?.value;
  if (!separador) throw new TypeError("localizacion no valida");
  const fraccion = new Intl.NumberFormat(localizacion, {
    useGrouping: false, minimumIntegerDigits: 2,
  }).format(resto);
  const ultimaParteEntera = partes.map((parte) => parte.type).lastIndexOf("integer");
  if (ultimaParteEntera < 0) throw new TypeError("formato de moneda no valido");
  partes.splice(ultimaParteEntera + 1, 0,
    { type: "decimal", value: separador }, { type: "fraction", value: fraccion });
  return partes.map((parte) => parte.value).join("");
}

/** Monta una lectura inyectada; no consulta ni publica tarifas por su cuenta. */
export function montarCatalogoTarifasDietas(contenedor, {
  fuente, traducir, localizacion, anunciar = () => {}, registrarDesmontar,
} = {}) {
  if (!contenedor?.ownerDocument?.createElement || typeof contenedor.append !== "function" ||
      typeof fuente !== "function" || typeof traducir !== "function" ||
      typeof localizacion !== "string" || typeof anunciar !== "function" ||
      (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function"))
    throw new TypeError("catalogo de tarifas de Dietas no disponible");

  const d = contenedor.ownerDocument;
  const idLimite = `dietas-catalogo-limite-${++numeroInstancias}`;
  const t = (clave, variables) => traducir(`catalogo.${clave}`, variables);
  const referenciaFuente = (url) => {
    const parsed = new URL(url);
    const id = parsed.searchParams.get("id");
    return id && /^BOE-[A-Z]-\d{4}-\d+$/u.test(id) ? id : parsed.hostname;
  };
  const moneda = (centimos, codigo) => formatearImporteCentimos(centimos, localizacion, codigo);
  const importeTarifa = (centimos, tarifa) =>
    `${moneda(centimos, tarifa.moneda)}${tarifa.unidad === "kilometro" ? t("por_km") : ""}`;
  const fecha = (valor) => new Intl.DateTimeFormat(localizacion, { dateStyle: "medium", timeZone: "UTC" }).format(new Date(`${valor}T00:00:00Z`));
  const raiz = nodo(d, "section"); raiz.className = "dietas-catalogo panel"; raiz.dataset.dietasCatalogo = "";
  const cabecera = nodo(d, "div"); cabecera.className = "cabecera-panel";
  cabecera.append(nodo(d, "h2", t("titulo")));
  const ayuda = nodo(d, "details");
  const botonAyuda = nodo(d, "summary", "?"); botonAyuda.setAttribute("aria-label", t("ayuda_etiqueta"));
  ayuda.append(botonAyuda, nodo(d, "p", t("ayuda"))); cabecera.append(ayuda);
  const estado = nodo(d, "p"); estado.setAttribute("role", "status"); estado.setAttribute("aria-live", "polite"); estado.tabIndex = -1;
  const cuerpo = nodo(d, "div"); cuerpo.className = "cuerpo-panel modulo-dietas";
  raiz.append(cabecera, estado, cuerpo); contenedor.append(raiz);
  let vivo = true, secuencia = 0, controlador, catalogo;
  const propuestas = new Map();
  let tarifaEditada;
  const claveDe = (tarifa) => `${catalogo.version}\u0000${tarifa.id}`;
  const nombreTarifa = (tarifa) => t("tarifa_resumen", {
    concepto: t(`concepto_${tarifa.concepto}`),
    grupo: tarifa.grupo === null ? t("sin_grupo") : String(tarifa.grupo), pais: tarifa.pais,
  });
  const ponerEstado = (clave) => { estado.textContent = t(clave); anunciar(estado.textContent); };

  function textoCortable(valor) {
    const contenedorTexto = nodo(d, "span");
    for (const parte of String(valor).split(/(?<=[:/._-])/u)) {
      contenedorTexto.append(d.createTextNode(parte), nodo(d, "wbr"));
    }
    return contenedorTexto;
  }

  function par(lista, clave, valor) {
    const fila = nodo(d, "div"), definicion = nodo(d, "dd");
    if (valor?.nodeType) definicion.append(valor); else definicion.textContent = String(valor);
    fila.append(nodo(d, "dt", t(clave)), definicion); lista.append(fila);
    return definicion;
  }

  function pintar() {
    cuerpo.replaceChildren();
    const contexto = nodo(d, "dl");
    par(contexto, "version", textoCortable(catalogo.version));
    par(contexto, "version_tarifa", textoCortable(catalogo.version_tarifa_ref));
    par(contexto, "estado_ejemplo", t("ejemplo"));
    par(contexto, "aprobacion", t("aprobacion_pendiente"));
    par(contexto, "vigencia", `${fecha(catalogo.vigente_desde)} — ${catalogo.vigente_hasta ? fecha(catalogo.vigente_hasta) : t("sin_fin")}`);
    const panelContexto = nodo(d, "section"); panelContexto.className = "panel";
    const cabContexto = nodo(d, "div"); cabContexto.className = "cabecera-panel";
    cabContexto.append(nodo(d, "h3", t("version")));
    const cuerpoContexto = nodo(d, "div"); cuerpoContexto.className = "cuerpo-panel"; cuerpoContexto.append(contexto);
    panelContexto.append(cabContexto, cuerpoContexto); cuerpo.append(panelContexto);
    const fuentes = nodo(d, "section"); fuentes.className = "panel";
    const cabFuentes = nodo(d, "div"); cabFuentes.className = "cabecera-panel";
    cabFuentes.append(nodo(d, "h3", t("fuentes")));
    const cuerpoFuentes = nodo(d, "div"); cuerpoFuentes.className = "cuerpo-panel";
    const listaFuentes = nodo(d, "ul");
    for (const url of catalogo.fuentes) {
      const li = nodo(d, "li"), enlace = nodo(d, "a", t("consultar_fuente", { referencia: referenciaFuente(url) }));
      enlace.href = url; enlace.rel = "noopener noreferrer"; enlace.target = "_blank"; li.append(enlace); listaFuentes.append(li);
    }
    cuerpoFuentes.append(listaFuentes); fuentes.append(cabFuentes, cuerpoFuentes); cuerpo.append(fuentes);

    if (!catalogo.tarifas.length) cuerpo.append(nodo(d, "p", t("vacio")));
    else {
      const envoltura = nodo(d, "div"); envoltura.className = "tabla-contenedor";
      envoltura.tabIndex = 0; envoltura.setAttribute("role", "region");
      envoltura.setAttribute("aria-label", t("tabla_desplazable"));
      const tabla = nodo(d, "table"); tabla.className = "tabla-datos"; tabla.setAttribute("aria-label", t("tabla_etiqueta"));
      const cab = nodo(d, "thead"), filaCab = nodo(d, "tr");
      for (const clave of ["concepto", "grupo", "pais", "importe", "accion"]) filaCab.append(nodo(d, "th", t(clave)));
      cab.append(filaCab); tabla.append(cab);
      const tbody = nodo(d, "tbody");
      for (const tarifa of catalogo.tarifas) {
        const tr = nodo(d, "tr"), accion = nodo(d, "td");
        const boton = nodo(d, "button", t("preparar")); boton.type = "button"; boton.className = "boton-secundario";
        boton.addEventListener("click", () => { pintarEditor(tarifa, true); pintarPropuestas(); }); accion.append(boton);
        tr.append(nodo(d, "td", t(`concepto_${tarifa.concepto}`)),
          nodo(d, "td", tarifa.grupo === null ? t("sin_grupo") : String(tarifa.grupo)),
          nodo(d, "td", tarifa.pais),
          nodo(d, "td", importeTarifa(tarifa.importe_centimos, tarifa)), accion);
        tbody.append(tr);
      }
      tabla.append(tbody); envoltura.append(tabla); cuerpo.append(envoltura);
      const resumen = nodo(d, "section"); resumen.className = "panel";
      resumen.dataset.dietasCatalogoPropuestas = ""; cuerpo.append(resumen);
      pintarPropuestas();
      const editor = nodo(d, "section"); editor.className = "panel"; editor.dataset.dietasCatalogoEditor = "";
      editor.hidden = true; cuerpo.append(editor);
    }
    const historia = nodo(d, "section"); historia.className = "panel";
    const cabHistoria = nodo(d, "div"); cabHistoria.className = "cabecera-panel"; cabHistoria.append(nodo(d, "h3", t("historia")));
    const listaHistoria = nodo(d, "div"); listaHistoria.className = "cuerpo-panel";
    if (!catalogo.historia.length) listaHistoria.append(nodo(d, "p", t("historia_vacia")));
    for (const evento of catalogo.historia) {
      const dl = nodo(d, "dl");
      par(dl, "historia_fecha", fecha(evento.fecha)); par(dl, "historia_actor", evento.actor);
      par(dl, "historia_accion", t(`accion_${evento.accion}`));
      par(dl, "historia_motivo", evento.motivo).setAttribute("lang", evento.idioma_motivo);
      par(dl, "version", textoCortable(evento.version));
      par(dl, "estado_ejemplo", t("ejemplo")); listaHistoria.append(dl);
    }
    historia.append(cabHistoria, listaHistoria); cuerpo.append(historia);
  }

  function pintarPropuestas(focoTrasRetirada) {
    const resumen = cuerpo.querySelector("[data-dietas-catalogo-propuestas]");
    if (!resumen) return;
    resumen.replaceChildren();
    const cab = nodo(d, "div"); cab.className = "cabecera-panel";
    const titulo = nodo(d, "h3", t("propuestas_titulo")); titulo.tabIndex = -1; cab.append(titulo);
    const zona = nodo(d, "div"); zona.className = "cuerpo-panel";
    zona.append(nodo(d, "p", t("propuestas_limite")));
    const revisadas = catalogo.tarifas.filter((tarifa) => Number.isSafeInteger(propuestas.get(claveDe(tarifa))?.importe_centimos));
    const botones = [];
    if (!revisadas.length) zona.append(nodo(d, "p", t("propuestas_vacias")));
    else {
      const envoltura = nodo(d, "div"); envoltura.className = "tabla-contenedor";
      envoltura.tabIndex = 0; envoltura.setAttribute("role", "region");
      envoltura.setAttribute("aria-label", t("propuestas_desplazable"));
      const tabla = nodo(d, "table"); tabla.className = "tabla-datos";
      tabla.setAttribute("aria-label", t("propuestas_titulo"));
      const cabTabla = nodo(d, "thead"), filaCab = nodo(d, "tr");
      for (const clave of ["tarifa_acciones", "importe_actual", "propuesto", "diferencia", "historia_motivo", "fecha_propuesta", "fuentes"]) {
        const th = nodo(d, "th", t(clave)); th.setAttribute("scope", "col"); filaCab.append(th);
      }
      cabTabla.append(filaCab); tabla.append(cabTabla);
      const filas = nodo(d, "tbody");
      for (const tarifa of revisadas) {
        const propuesta = propuestas.get(claveDe(tarifa));
        const fila = nodo(d, "tr");
        const nombre = nodo(d, "th", nombreTarifa(tarifa)); nombre.setAttribute("scope", "row");
        const diferencia = propuesta.importe_centimos - tarifa.importe_centimos;
        const delta = diferencia === 0 ? t("sin_cambio") : t(diferencia > 0 ? "aumento" : "reduccion", {
          importe: importeTarifa(Math.abs(diferencia), tarifa),
        });
        const motivo = nodo(d, "td", propuesta.motivo); motivo.setAttribute("lang", propuesta.idioma_motivo);
        const referencia = nodo(d, "td"), enlace = nodo(d, "a", t("consultar_fuente", { referencia: referenciaFuente(propuesta.fuente) }));
        enlace.href = propuesta.fuente; enlace.rel = "noopener noreferrer"; enlace.target = "_blank"; referencia.append(enlace);
        const grupoAcciones = nodo(d, "div"); grupoAcciones.className = "acciones-vista";
        const elegir = nodo(d, "button", t("corregir_propuesta")); elegir.type = "button"; elegir.className = "boton-secundario";
        elegir.setAttribute("aria-label", t("corregir_tarifa", { tarifa: nombreTarifa(tarifa) }));
        if (tarifaEditada === tarifa.id) elegir.setAttribute("aria-current", "true");
        elegir.addEventListener("click", () => { pintarEditor(tarifa, true); pintarPropuestas(); }); botones.push(elegir);
        const retirar = nodo(d, "button", t("retirar_propuesta")); retirar.type = "button"; retirar.className = "boton-secundario";
        retirar.setAttribute("aria-label", t("retirar_tarifa", { tarifa: nombreTarifa(tarifa) }));
        retirar.addEventListener("click", () => {
          const indice = revisadas.indexOf(tarifa);
          propuestas.delete(claveDe(tarifa));
          if (tarifaEditada === tarifa.id) {
            const editor = cuerpo.querySelector("[data-dietas-catalogo-editor]");
            editor.replaceChildren(); editor.hidden = true; tarifaEditada = undefined;
          }
          pintarPropuestas(indice);
          anunciar(t("propuesta_retirada", { tarifa: nombreTarifa(tarifa) }));
        });
        grupoAcciones.append(elegir, retirar); nombre.append(grupoAcciones);
        fila.append(nombre, nodo(d, "td", importeTarifa(tarifa.importe_centimos, tarifa)),
          nodo(d, "td", importeTarifa(propuesta.importe_centimos, tarifa)), nodo(d, "td", delta),
          motivo, nodo(d, "td", fecha(propuesta.vigencia)), referencia); filas.append(fila);
      }
      tabla.append(filas); envoltura.append(tabla); zona.append(envoltura);
    }
    resumen.append(cab, zona);
    if (focoTrasRetirada !== undefined) {
      (botones[Math.min(focoTrasRetirada, botones.length - 1)] || titulo).focus({ preventScroll: true });
    }
  }

  function pintarEditor(tarifa, moverFoco = false) {
    const editor = cuerpo.querySelector("[data-dietas-catalogo-editor]");
    if (!editor) return;
    editor.replaceChildren(); editor.hidden = false; tarifaEditada = tarifa.id;
    const cab = nodo(d, "div"); cab.className = "cabecera-panel";
    const tituloEditor = nodo(d, "h3", t("propuesta_titulo")); tituloEditor.tabIndex = -1; cab.append(tituloEditor);
    const zona = nodo(d, "div"); zona.className = "cuerpo-panel";
    const grupo = tarifa.grupo === null ? t("sin_grupo") : String(tarifa.grupo);
    const referencia = nodo(d, "p", `${t(`concepto_${tarifa.concepto}`)} · ${t("grupo")} ${grupo} · ${tarifa.pais} · ${importeTarifa(tarifa.importe_centimos, tarifa)}`);
    const clavePropuesta = claveDe(tarifa);
    const previo = propuestas.get(clavePropuesta);
    const form = nodo(d, "form"); form.noValidate = true;
    const importeLabel = nodo(d, "label", t("importe_propuesto"));
    const importe = nodo(d, "input"); importe.name = "importe_propuesto"; importe.inputMode = "decimal"; importe.required = true;
    importe.value = previo?.texto || ""; importeLabel.append(nodo(d, "br"), importe);
    const motivoLabel = nodo(d, "label", t("motivo_obligatorio"));
    const motivo = nodo(d, "textarea"); motivo.name = "motivo"; motivo.required = true; motivo.maxLength = 500;
    motivo.value = previo?.motivo || ""; motivoLabel.append(nodo(d, "br"), motivo);
    const vigenciaLabel = nodo(d, "label", t("vigencia_propuesta"));
    const vigencia = nodo(d, "input"); vigencia.type = "date"; vigencia.name = "vigencia_propuesta";
    vigencia.required = true; vigencia.value = previo?.vigencia || catalogo.vigente_desde;
    vigenciaLabel.append(nodo(d, "br"), vigencia);
    const fuenteLabel = nodo(d, "label", t("fuente_propuesta"));
    const fuente = nodo(d, "select"); fuente.name = "fuente_propuesta"; fuente.required = true;
    for (const [indice, url] of catalogo.fuentes.entries()) {
      const opcion = nodo(d, "option", t("fuente_numero", { numero: indice + 1 }));
      opcion.value = url; fuente.append(opcion);
    }
    fuente.value = previo?.fuente || catalogo.fuentes[0]; fuenteLabel.append(nodo(d, "br"), fuente);
    const validacion = nodo(d, "small"); validacion.setAttribute("role", "alert");
    const revisar = nodo(d, "button", t("revisar")); revisar.type = "submit"; revisar.className = "boton-secundario";
    const resultado = nodo(d, "div"); resultado.setAttribute("aria-live", "polite");
    const publicar = nodo(d, "button", t("publicar")); publicar.type = "button"; publicar.disabled = true;
    publicar.title = t("publicacion_pendiente"); publicar.setAttribute("aria-describedby", idLimite);
    const limite = nodo(d, "p", t("publicacion_pendiente")); limite.id = idLimite;
    const conservar = () => { propuestas.set(clavePropuesta, { texto: importe.value, motivo: motivo.value, vigencia: vigencia.value, fuente: fuente.value }); resultado.replaceChildren(); pintarPropuestas(); };
    for (const control of [importe, motivo, vigencia, fuente]) control.addEventListener("input", conservar);
    form.addEventListener("submit", (evento) => {
      evento.preventDefault(); validacion.textContent = ""; resultado.replaceChildren();
      const centimos = importeACentimos(importe.value, localizacion);
      if (centimos === null) { validacion.textContent = t("importe_invalido"); importe.focus(); return; }
      const razon = motivo.value.trim();
      if (razon.length < 3 || razon.length > 500 || /[\x00-\x1f\x7f]/u.test(razon)) {
        validacion.textContent = t("motivo_invalido"); motivo.focus(); return;
      }
      if (!fechaValida(vigencia.value)) { validacion.textContent = t("vigencia_invalida"); vigencia.focus(); return; }
      if (!catalogo.fuentes.includes(fuente.value)) { validacion.textContent = t("fuente_invalida"); fuente.focus(); return; }
      propuestas.set(clavePropuesta, { texto: importe.value, motivo: razon, vigencia: vigencia.value, fuente: fuente.value, importe_centimos: centimos, idioma_motivo: localizacion });
      pintarPropuestas();
      const dl = nodo(d, "dl");
      par(dl, "importe_actual", importeTarifa(tarifa.importe_centimos, tarifa));
      par(dl, "importe_propuesto", importeTarifa(centimos, tarifa));
      par(dl, "historia_motivo", razon); par(dl, "version", textoCortable(catalogo.version));
      par(dl, "vigencia_propuesta", fecha(vigencia.value));
      const filaFuente = nodo(d, "div"); filaFuente.append(nodo(d, "dt", t("fuente_propuesta")));
      const valorFuente = nodo(d, "dd"), enlaceFuente = nodo(d, "a", t("consultar_fuente", { referencia: referenciaFuente(fuente.value) }));
      enlaceFuente.href = fuente.value; enlaceFuente.rel = "noopener noreferrer"; enlaceFuente.target = "_blank";
      valorFuente.append(enlaceFuente); filaFuente.append(valorFuente); dl.append(filaFuente);
      par(dl, "aprobacion", t("aprobacion_pendiente"));
      resultado.append(nodo(d, "h4", t("revision_local")), dl, nodo(d, "p", t("revision_limite")));
      anunciar(t("revision_local"));
    });
    for (const etiqueta of [importeLabel, motivoLabel, vigenciaLabel, fuenteLabel]) {
      const campo = nodo(d, "p"); campo.append(etiqueta); form.append(campo);
    }
    form.append(validacion, nodo(d, "br"), revisar); zona.append(referencia, form, resultado, publicar, limite);
    editor.append(cab, zona);
    if (moverFoco) {
      tituloEditor.focus({ preventScroll: true });
      editor.scrollIntoView?.({ block: "nearest", inline: "nearest" });
    }
  }

  async function cargar(origenFoco) {
    const turno = ++secuencia;
    const recuperarFoco = origenFoco && d.activeElement === origenFoco;
    const focoLibre = () => !d.activeElement || d.activeElement === d.body || d.activeElement === origenFoco;
    controlador?.abort(); controlador = new AbortController();
    ponerEstado("cargando"); cuerpo.replaceChildren();
    try {
      const dato = await fuente({ signal: controlador.signal });
      if (!vivo || turno !== secuencia) return;
      if (!catalogoTarifasDietasValido(dato)) throw new TypeError("catalogo invalido");
      catalogo = structuredClone(dato); ponerEstado("ejemplo"); pintar();
      if (recuperarFoco && focoLibre()) {
        estado.focus({ preventScroll: true });
        estado.scrollIntoView?.({ block: "nearest", inline: "nearest" });
      }
    } catch (error) {
      if (!vivo || turno !== secuencia || controlador.signal.aborted) return;
      ponerEstado(error?.codigo === "acceso_denegado" ? "denegado" : "error");
      const reintentar = nodo(d, "button", t("reintentar")); reintentar.type = "button";
      reintentar.className = "boton-secundario";
      reintentar.addEventListener("click", () => { void cargar(reintentar); }); cuerpo.append(reintentar);
      if (recuperarFoco && focoLibre()) reintentar.focus({ preventScroll: true });
    }
  }
  const desmontar = () => { vivo = false; ++secuencia; controlador?.abort(); propuestas.clear(); raiz.remove(); };
  registrarDesmontar?.(desmontar); void cargar();
  return desmontar;
}
