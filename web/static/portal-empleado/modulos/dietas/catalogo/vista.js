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
  const moneda = (centimos, codigo) => new Intl.NumberFormat(localizacion, {
    style: "currency", currency: codigo, currencyDisplay: "code",
  }).format(centimos / 100);
  const fecha = (valor) => new Intl.DateTimeFormat(localizacion, { dateStyle: "medium", timeZone: "UTC" }).format(new Date(`${valor}T00:00:00Z`));
  const raiz = nodo(d, "section"); raiz.className = "dietas-catalogo panel"; raiz.dataset.dietasCatalogo = "";
  const cabecera = nodo(d, "div"); cabecera.className = "cabecera-panel";
  cabecera.append(nodo(d, "h2", t("titulo")));
  const ayuda = nodo(d, "details");
  const botonAyuda = nodo(d, "summary", "?"); botonAyuda.setAttribute("aria-label", t("ayuda_etiqueta"));
  ayuda.append(botonAyuda, nodo(d, "p", t("ayuda"))); cabecera.append(ayuda);
  const estado = nodo(d, "p"); estado.setAttribute("role", "status"); estado.setAttribute("aria-live", "polite");
  const cuerpo = nodo(d, "div"); cuerpo.className = "cuerpo-panel";
  raiz.append(cabecera, estado, cuerpo); contenedor.append(raiz);
  let vivo = true, secuencia = 0, controlador, catalogo;
  const propuestas = new Map();
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
  }

  function pintar() {
    cuerpo.replaceChildren();
    if (!catalogo.tarifas.length) { cuerpo.append(nodo(d, "p", t("vacio"))); return; }
    const contexto = nodo(d, "dl");
    par(contexto, "version", textoCortable(catalogo.version));
    par(contexto, "version_tarifa", textoCortable(catalogo.version_tarifa_ref));
    par(contexto, "estado_ejemplo", t("ejemplo"));
    par(contexto, "aprobacion", t("aprobacion_pendiente"));
    par(contexto, "vigencia", `${fecha(catalogo.vigente_desde)} — ${catalogo.vigente_hasta ? fecha(catalogo.vigente_hasta) : t("sin_fin")}`);
    cuerpo.append(contexto);
    const fuentes = nodo(d, "section"); fuentes.append(nodo(d, "h3", t("fuentes")));
    const listaFuentes = nodo(d, "ul");
    for (const [indice, url] of catalogo.fuentes.entries()) {
      const li = nodo(d, "li"), enlace = nodo(d, "a", t("consultar_fuente", { referencia: referenciaFuente(url) }));
      enlace.href = url; enlace.rel = "noopener noreferrer"; enlace.target = "_blank"; li.append(enlace); listaFuentes.append(li);
    }
    fuentes.append(listaFuentes); cuerpo.append(fuentes);

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
      boton.addEventListener("click", () => pintarEditor(tarifa)); accion.append(boton);
      tr.append(nodo(d, "td", t(`concepto_${tarifa.concepto}`)),
        nodo(d, "td", tarifa.grupo === null ? t("sin_grupo") : String(tarifa.grupo)),
        nodo(d, "td", tarifa.pais),
        nodo(d, "td", `${moneda(tarifa.importe_centimos, tarifa.moneda)}${tarifa.unidad === "kilometro" ? t("por_km") : ""}`), accion);
      tbody.append(tr);
    }
    tabla.append(tbody); envoltura.append(tabla); cuerpo.append(envoltura);
    const editor = nodo(d, "section"); editor.className = "panel"; editor.dataset.dietasCatalogoEditor = ""; cuerpo.append(editor);
    const historia = nodo(d, "section"); historia.className = "panel";
    const cabHistoria = nodo(d, "div"); cabHistoria.className = "cabecera-panel"; cabHistoria.append(nodo(d, "h3", t("historia")));
    const listaHistoria = nodo(d, "div"); listaHistoria.className = "cuerpo-panel";
    if (!catalogo.historia.length) listaHistoria.append(nodo(d, "p", t("historia_vacia")));
    for (const evento of catalogo.historia) {
      const dl = nodo(d, "dl");
      par(dl, "historia_fecha", fecha(evento.fecha)); par(dl, "historia_actor", evento.actor);
      par(dl, "historia_accion", t(`accion_${evento.accion}`));
      par(dl, "historia_motivo", evento.motivo); par(dl, "version", textoCortable(evento.version));
      par(dl, "estado_ejemplo", t("ejemplo")); listaHistoria.append(dl);
    }
    historia.append(cabHistoria, listaHistoria); cuerpo.append(historia);
    pintarEditor(catalogo.tarifas[0]);
  }

  function pintarEditor(tarifa) {
    const editor = cuerpo.querySelector("[data-dietas-catalogo-editor]");
    if (!editor) return;
    editor.replaceChildren();
    const cab = nodo(d, "div"); cab.className = "cabecera-panel"; cab.append(nodo(d, "h3", t("propuesta_titulo")));
    const zona = nodo(d, "div"); zona.className = "cuerpo-panel";
    const referencia = nodo(d, "p", `${t(`concepto_${tarifa.concepto}`)} · ${moneda(tarifa.importe_centimos, tarifa.moneda)}`);
    const clavePropuesta = `${catalogo.version}\u0000${tarifa.id}`;
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
    const conservar = () => { propuestas.set(clavePropuesta, { texto: importe.value, motivo: motivo.value, vigencia: vigencia.value, fuente: fuente.value }); resultado.replaceChildren(); };
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
      propuestas.set(clavePropuesta, { texto: importe.value, motivo: razon, vigencia: vigencia.value, fuente: fuente.value, importe_centimos: centimos });
      const dl = nodo(d, "dl");
      par(dl, "importe_actual", moneda(tarifa.importe_centimos, tarifa.moneda));
      par(dl, "importe_propuesto", moneda(centimos, tarifa.moneda));
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
  }

  async function cargar() {
    const turno = ++secuencia;
    controlador?.abort(); controlador = new AbortController();
    ponerEstado("cargando"); cuerpo.replaceChildren();
    try {
      const dato = await fuente({ signal: controlador.signal });
      if (!vivo || turno !== secuencia) return;
      if (!catalogoTarifasDietasValido(dato)) throw new TypeError("catalogo invalido");
      catalogo = dato; ponerEstado("ejemplo"); pintar();
    } catch (error) {
      if (!vivo || turno !== secuencia || controlador.signal.aborted) return;
      ponerEstado(error?.codigo === "acceso_denegado" ? "denegado" : "error");
      const reintentar = nodo(d, "button", t("reintentar")); reintentar.type = "button";
      reintentar.className = "boton-secundario"; reintentar.addEventListener("click", cargar); cuerpo.append(reintentar);
    }
  }
  const desmontar = () => { vivo = false; ++secuencia; controlador?.abort(); propuestas.clear(); raiz.remove(); };
  registrarDesmontar?.(desmontar); void cargar();
  return desmontar;
}
