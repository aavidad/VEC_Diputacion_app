import { cargarTextosNombramientos } from "./i18n-nombramientos-derivados.js?v=20261018-f119-derivacion-v2";

const referencia = (valor) => typeof valor === "string" && /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u.test(valor);
const entero = (valor) => Number.isSafeInteger(valor) && valor >= 1;

export function plazasDerivables(oferta, recibo) {
  if (oferta?.oferta_ref !== recibo?.oferta_ref || oferta?.bolsa_ref !== recibo?.bolsa_ref
    || oferta?.numero_plazas !== recibo?.numero_plazas || oferta?.confirmacion_adjudicacion !== "aceptacion_previa"
    || !Array.isArray(oferta.plazas) || oferta.plazas.length !== recibo.numero_plazas) throw new TypeError("oferta_incompleta");
  return oferta.plazas.filter((plaza) => {
    const acto = plaza.historial?.at(-1);
    if (plaza.estado === "cubierta" && (!entero(plaza.numero_de_plaza)
      || !entero(plaza.secuencia) || !referencia(plaza.participacion_ref)
      || acto?.tipo !== "adjudicada" || !referencia(acto.recibo_ref)
      || acto.secuencia !== plaza.secuencia)) throw new TypeError("plaza_incompleta");
    return plaza.estado === "cubierta" && entero(plaza.numero_de_plaza)
      && entero(plaza.secuencia) && referencia(plaza.participacion_ref)
      && acto?.tipo === "adjudicada" && referencia(acto.recibo_ref)
      && acto.secuencia === plaza.secuencia;
  }).map((plaza) => ({ numero_plaza: plaza.numero_de_plaza, secuencia_plaza: plaza.secuencia,
    participacion_ref: plaza.participacion_ref, recibo_adjudicacion_ref: plaza.historial.at(-1).recibo_ref }));
}

export async function consultarHijos(clienteCT, numeroVisible, signal) {
  const textoBusqueda = `${numeroVisible}-P`;
  const hijos = new Map();
  let cursor = "";
  const vistos = new Set();
  do {
    const pagina = await clienteCT.consultarCuadroRRHH({ filtros: { texto: textoBusqueda, estado_clave: "", fase_clave: "" },
      paginacion: { limite: 100, cursor } }, { signal });
    if (!Array.isArray(pagina?.expedientes) || typeof pagina.hay_mas !== "boolean") throw new TypeError("pagina_incompleta");
    for (const expediente of pagina.expedientes) {
      if (!expediente.numero_visible?.startsWith(textoBusqueda)) continue;
      const numero = Number(expediente.numero_visible.slice(textoBusqueda.length));
      if (entero(numero) && referencia(expediente.expediente_ref) && entero(expediente.version)) {
        if (hijos.has(numero)) throw new TypeError("hijo_duplicado");
        hijos.set(numero, expediente);
      }
    }
    if (!pagina.hay_mas) break;
    cursor = pagina.cursor_siguiente;
    if (!cursor || vistos.has(cursor)) throw new TypeError("paginacion_incompleta");
    vistos.add(cursor);
  } while (true);
  return hijos;
}

export function montarNombramientosDerivados({ raiz, expediente, resumen, clienteCT, clienteOfertas, abrirExpediente,
  confirmarOperacion = () => false, generarClave = () => crypto.randomUUID(),
  locale, traducirComun, cargarCatalogo = cargarTextosNombramientos }) {
  const esPadre = expediente?.version >= 6 && expediente?.numero_personas >= 2
    && resumen?.version === expediente.version && resumen?.fase_clave === "fiscalizacion"
    && resumen?.estado_clave === "en_curso" && expediente?.demostracion === false;
  if (!esPadre || !raiz?.ownerDocument) return () => {};
  let mensajes = null;
  const texto = (clave, datos = {}) => String(mensajes[clave])
    .replace(/\{(\w+)\}/gu, (_, nombre) => String(datos[nombre] ?? ""));
  const doc = raiz.ownerDocument;
  const panel = doc.createElement("section");
  panel.className = "panel panel-separado";
  panel.setAttribute("aria-label", traducirComun("fase_nombramiento"));
  panel.setAttribute("data-ct-nombramientos", "");
  const cuerpo = doc.createElement("div"); cuerpo.className = "cuerpo-panel";
  const titulo = doc.createElement("h3"); titulo.textContent = traducirComun("fase_nombramiento");
  const ayuda = doc.createElement("details");
  ayuda.hidden = true;
  const resumenAyuda = doc.createElement("summary");
  const ayudaTexto = doc.createElement("p"); ayuda.append(resumenAyuda, ayudaTexto);
  const estado = doc.createElement("div"); estado.setAttribute("role", "status"); estado.setAttribute("aria-live", "polite");
  cuerpo.append(titulo, ayuda, estado); panel.append(cuerpo);
  const ancla = raiz.querySelector("[data-ct-exp-ancla-firma]");
  if (ancla) ancla.before(panel); else raiz.append(panel);
  let controlador = null; let viva = true; let plazas = []; let hijos = new Map(); let seleccionRef = "";
  const intentos = new Map();
  const boton = (etiqueta, accion) => { const b = doc.createElement("button"); b.type = "button";
    b.className = "boton-secundario"; b.textContent = etiqueta; b.addEventListener("click", accion); return b; };
  const falloCatalogo = () => { estado.replaceChildren();
    const p = doc.createElement("p"); p.textContent = traducirComun("txt_el_servicio_no_esta_disponible_ahora_puede_reint");
    estado.append(p, boton(traducirComun("accion_reintentar"), () => void preparar(true))); };
  const cargandoCatalogo = () => { estado.replaceChildren();
    const p = doc.createElement("p"); p.textContent = traducirComun("estado_modulo_comprobando"); estado.append(p); };
  const aviso = (clave, reintento = false) => { estado.replaceChildren();
    const p = doc.createElement("p"); p.textContent = texto(clave); estado.append(p);
    if (reintento) estado.append(boton(texto("reintentar"), cargar)); };
  const pintar = () => {
    estado.replaceChildren();
    if (!plazas.length) { aviso("sin_aceptadas"); return; }
    const lista = doc.createElement("ul"); lista.className = "lista-acciones";
    for (const plaza of plazas) {
      const fila = doc.createElement("li"); fila.className = "panel panel-separado";
      const nombre = doc.createElement("strong"); nombre.textContent = texto("plaza", { numero: plaza.numero_plaza }); fila.append(nombre);
      const hijo = hijos.get(plaza.numero_plaza);
      const situacion = doc.createElement("p"); situacion.textContent = texto(hijo ? "creada" : "pendiente"); fila.append(situacion);
      if (hijo) fila.append(boton(texto("abrir", { numero: hijo.numero_visible }), () => abrirExpediente?.(hijo.expediente_ref, hijo.version)));
      else fila.append(boton(texto("crear"), () => void derivar(plaza)));
      lista.append(fila);
    }
    estado.append(lista);
  };
  async function cargar() {
    controlador?.abort(); controlador = new AbortController(); const { signal } = controlador;
    aviso("cargando");
    if (typeof clienteCT?.consultarEstadoSeleccionOferta !== "function" || typeof clienteCT?.consultarCuadroRRHH !== "function"
      || typeof clienteCT?.derivarNombramiento !== "function" || typeof clienteOfertas?.consultar !== "function") { aviso("sin_montaje"); return; }
    try {
      const seleccion = await clienteCT.consultarEstadoSeleccionOferta(expediente.expediente_ref, { signal });
      if (seleccion?.estado !== "confirmada") { if (!signal.aborted) aviso("sin_oferta"); return; }
      const recibo = seleccion.oferta;
      seleccionRef = recibo.oferta_ref;
      const resultado = await clienteOfertas.consultar(recibo.bolsa_ref, { signal });
      if (!resultado?.ok || !Array.isArray(resultado.datos?.ofertas)) throw new TypeError("consulta_oferta_fallida");
      const oferta = resultado.datos.ofertas.find((item) => item.oferta_ref === recibo.oferta_ref);
      if (!oferta) { if (!signal.aborted) aviso("error_oferta_pagina", true); return; }
      const siguientes = plazasDerivables(oferta, recibo);
      const existentes = await consultarHijos(clienteCT, expediente.numero_visible, signal);
      if (signal.aborted || !viva) return;
      plazas = siguientes; hijos = existentes; pintar();
    } catch (error) { if (!signal.aborted && viva) aviso(error?.estado === 403 ? "error_403" : "error_consulta", true); }
  }
  async function derivar(plaza) {
    if (hijos.has(plaza.numero_plaza)) return;
    const numero = plaza.numero_plaza;
    let entrada = intentos.get(numero);
    if (!entrada) {
      if (confirmarOperacion({ titulo: texto("confirmar_titulo", { numero }),
        advertencia: texto("confirmar_advertencia"), referencia: expediente.expediente_ref }) !== true) return;
      entrada = Object.freeze({ padre_ref: expediente.expediente_ref, version_padre: expediente.version,
        oferta_ref: seleccionRef, ...plaza, clave_idempotencia: generarClave() }); intentos.set(numero, entrada);
    }
    aviso("cargando");
    try {
      const recibo = await clienteCT.derivarNombramiento(entrada, { signal: controlador?.signal });
      if (!viva || controlador?.signal.aborted) return;
      hijos.set(numero, { expediente_ref: recibo.hijo_ref, numero_visible: recibo.numero_visible, version: recibo.version });
      intentos.delete(numero); pintar();
      const p = doc.createElement("p"); p.textContent = texto("confirmado");
      p.setAttribute("data-ct-recibo-ref", recibo.recibo_ref); estado.prepend(p);
    } catch (error) {
      if (!viva || controlador?.signal.aborted) return;
      aviso(error?.estado === 403 ? "error_403" : error?.estado === 409 ? "error_409"
        : error?.estado === 503 ? "error_503" : "incierto");
      estado.append(boton(texto("recuperar"), () => void recuperar(numero)));
    }
  }
  async function recuperar(numero) {
    aviso("cargando");
    try {
      const actuales = await consultarHijos(clienteCT, expediente.numero_visible, controlador?.signal);
      if (!viva || controlador?.signal.aborted) return;
      hijos = actuales;
      if (hijos.has(numero)) { intentos.delete(numero); pintar(); return; }
      aviso("incierto");
      const plaza = plazas.find((item) => item.numero_plaza === numero);
      if (plaza && intentos.has(numero)) estado.append(boton(texto("recuperar"), () => void derivar(plaza)));
    } catch (error) { if (viva && !controlador?.signal.aborted) aviso(error?.estado === 403 ? "error_403" : "error_consulta", true); }
  }
  async function preparar(reintentar = false) {
    mensajes = null; ayuda.hidden = true; cargandoCatalogo();
    try {
      const nuevos = await cargarCatalogo(locale, reintentar);
      if (!viva) return;
      mensajes = nuevos;
      titulo.textContent = texto("titulo"); panel.setAttribute("aria-label", texto("titulo"));
      resumenAyuda.textContent = texto("simbolo_ayuda"); resumenAyuda.setAttribute("aria-label", texto("abrir_ayuda"));
      ayudaTexto.textContent = texto("ayuda");
      ayuda.hidden = false;
      await cargar();
    } catch { if (viva) falloCatalogo(); }
  }
  void preparar();
  return () => { viva = false; controlador?.abort(); panel.remove(); };
}
