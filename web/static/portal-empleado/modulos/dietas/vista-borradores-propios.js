import { crearTraductorDietas, MENSAJES_DIETAS_ES } from "./i18n.js";
import { obtenerCatalogoRutasProvincial } from "./catalogo-rutas-provincial.js";

// El catálogo público incluye núcleos NGMEP aún pendientes de importación.
// El alta calculada sólo ofrece municipios INE presentes en ambos contratos.
const PUNTOS_RUTA = obtenerCatalogoRutasProvincial().puntos.filter((punto) => /^\d{5}$/u.test(punto.codigo));
const NOMBRES_RUTA = new Map(PUNTOS_RUTA.map((punto) => [punto.codigo, punto.nombre]));

function nodo(documento, etiqueta, texto = "") {
  const resultado = documento.createElement(etiqueta);
  if (texto !== "") resultado.textContent = texto;
  return resultado;
}
function montada(raiz, contenedor) {
  return (
    raiz.querySelector?.("[data-dietas-borradores-propios]") === contenedor
  );
}
function retirar(raiz, contenedor) {
  if (!montada(raiz, contenedor)) return;
  if (typeof contenedor.remove === "function") contenedor.remove();
  else raiz.removeChild?.(contenedor);
}
function claveContenido(solicitud) {
  return JSON.stringify([
    solicitud.fecha_inicio,
    solicitud.fecha_fin,
    solicitud.hora_inicio,
    solicitud.hora_fin,
    solicitud.motivo,
    solicitud.codigos_ruta,
    solicitud.relacion_ref,
  ]);
}
function euros(centimos) { return new Intl.NumberFormat("es-ES", { style: "currency", currency: "EUR" }).format(centimos / 100); }
function referenciasRelacionAutorizadas(valores) {
  if (!Array.isArray(valores))
    throw new TypeError("relaciones autorizadas de Dietas no válidas");
  const referencias = valores.map((valor) => {
    if (
      typeof valor !== "string" ||
      !/^rel_[A-Za-z0-9_-]{22,128}$/u.test(valor)
    )
      throw new TypeError("relaciones autorizadas de Dietas no válidas");
    return valor;
  });
  if (new Set(referencias).size !== referencias.length)
    throw new TypeError("relaciones autorizadas de Dietas no válidas");
  return Object.freeze(referencias);
}
function fechaLegible(valor, conHora = false) {
  const fecha = new Date(conHora ? valor : `${valor}T00:00:00Z`);
  return Number.isFinite(fecha.getTime())
    ? new Intl.DateTimeFormat(
        "es-ES",
        conHora
          ? {
              dateStyle: "medium",
              timeStyle: "medium",
              timeZone: "Europe/Madrid",
            }
          : { dateStyle: "medium", timeZone: "UTC" },
      ).format(fecha)
    : "—";
}
function errorClave(error) {
  if (error?.codigo === "acceso_denegado")
    return "borradores_propios_error_acceso";
  if (
    [
      "relacion_ambigua",
      "relacion_no_disponible",
      "relacion_no_valida",
    ].includes(error?.codigo)
  )
    return "borradores_propios_error_relacion";
  if (error?.codigo === "conflicto_idempotencia")
    return "borradores_propios_error_conflicto";
  return error?.resultadoIndeterminado
    ? "borradores_propios_error_incierto"
    : "borradores_propios_error";
}

/** Monta la única superficie de creación y consulta de borradores propios de Dietas. */
export function montarVistaBorradoresPropios(
  contenedor,
  {
    cliente,
    traducir = crearTraductorDietas(MENSAJES_DIETAS_ES),
    anunciar = () => {},
    generarClaveIdempotencia = () => globalThis.crypto?.randomUUID?.(),
    registrarDesmontar,
    // Sólo la composición que haya consultado una fuente autorizada puede
    // aportar estas referencias opacas. Esta vista no deduce ni fabrica una.
    relacionesAutorizadas = [],
  } = {},
) {
  if (
    !contenedor?.append ||
    !contenedor?.querySelector ||
    !contenedor.ownerDocument ||
    (cliente !== undefined &&
      (typeof cliente?.crear !== "function" ||
        typeof cliente?.listar !== "function" ||
        typeof cliente?.obtener !== "function")) ||
    typeof traducir !== "function" ||
    typeof anunciar !== "function" ||
    typeof generarClaveIdempotencia !== "function" ||
    (registrarDesmontar !== undefined &&
      typeof registrarDesmontar !== "function")
  ) {
    throw new TypeError("vista de borradores propios de Dietas no disponible");
  }
  const documento = contenedor.ownerDocument;
  const relaciones = referenciasRelacionAutorizadas(relacionesAutorizadas);
  const raiz = nodo(documento, "section");
  raiz.className = "modulo-dietas";
  raiz.dataset.dietasBorradoresPropios = "";
  contenedor.append(raiz);
  const conectada = cliente !== undefined;
  let activa = true;
  let controlador = null;
  let pendiente = null;
  let seleccionada = null;
  let relacionSeleccionada =
    relaciones.length === 1 ? relaciones[0] : undefined;
  let estado = {
    carga: conectada,
    items: [],
    mensaje: conectada
      ? "borradores_propios_cargando"
      : "borradores_propios_pendiente_conexion",
    tono: "informacion",
    detalle: null,
  };
  const activaAhora = () => activa && montada(contenedor, raiz);
  const desmontar = () => {
    if (!activa) return;
    activa = false;
    controlador?.abort();
    raiz.removeEventListener("submit", enviar);
    raiz.removeEventListener("click", clic);
    raiz.removeEventListener("change", cambiarRelacion);
    retirar(contenedor, raiz);
  };
  registrarDesmontar?.(desmontar);

  function mensaje(clave, tono = "informacion") {
    estado = { ...estado, mensaje: clave, tono };
    try {
      anunciar(traducir(clave), tono);
    } catch {}
  }
  function formulario() {
    const form = nodo(documento, "form");
    form.dataset.dietasBorradorForm = "";
    form.className = "panel dietas-comision-formulario";
    const campo = (nombre, etiqueta, tipo = "text", obligatorio = true) => {
      const label = nodo(documento, "label", traducir(etiqueta));
      const input = nodo(documento, "input");
      input.name = nombre;
      input.type = tipo;
      input.required = obligatorio;
      label.append(input);
      return label;
    };
    const motivo = campo("motivo", "borradores_propios_motivo");
    const puntoRuta = (nombre, clave) => {
      const etiqueta = nodo(documento, "label", traducir(clave));
      const selector = nodo(documento, "select"); selector.name = nombre; selector.required = true;
      const inicial = nodo(documento, "option", traducir("borradores_propios_elegir_localidad")); inicial.value = ""; selector.append(inicial);
      PUNTOS_RUTA.forEach((punto) => { const opcion=nodo(documento,"option",punto.nombre); opcion.value=punto.codigo; selector.append(opcion); });
      etiqueta.append(selector); return etiqueta;
    };
    const boton = nodo(
      documento,
      "button",
      traducir("borradores_propios_guardar"),
    );
    boton.type = "submit";
    boton.className = "boton-primario";
    boton.disabled = controlador !== null;
    const selectorRelacion = (() => {
      if (!relaciones.length) return null;
      const etiqueta = nodo(
        documento,
        "label",
        traducir("borradores_propios_referencia"),
      );
      const selector = nodo(documento, relaciones.length === 1 ? "input" : "select");
      selector.name = "relacion_ref";
      if (relaciones.length === 1) {
        selector.type = "hidden";
        selector.value = relaciones[0];
      } else {
        selector.required = true;
        const inicial = nodo(documento, "option", "—");
        inicial.value = "";
        selector.append(inicial);
        relaciones.forEach((referencia) => {
          const opcion = nodo(documento, "option", referencia);
          opcion.value = referencia;
          selector.append(opcion);
        });
        selector.value = relacionSeleccionada || "";
      }
      etiqueta.append(selector);
      return etiqueta;
    })();
    form.append(
      nodo(documento,"h3",traducir("borradores_propios_nuevo")),
      campo("fecha_inicio", "borradores_propios_fecha_inicio", "date"),
      campo("hora_inicio", "borradores_propios_hora_inicio", "time"),
      campo("fecha_fin", "borradores_propios_fecha_fin", "date"),
      campo("hora_fin", "borradores_propios_hora_fin", "time"),
      motivo,
      puntoRuta("origen_codigo","borradores_propios_origen"),
      puntoRuta("destino_codigo","borradores_propios_destino"),
      nodo(documento,"p",traducir("borradores_propios_ruta_ayuda")),
      ...(selectorRelacion ? [selectorRelacion] : []),
      boton,
    );
    if (!conectada) {
      [...form.querySelectorAll("input"), ...form.querySelectorAll("select"), boton].forEach((control) => {
        control.disabled = true;
      });
    }
    return form;
  }
  function recibo(item) {
    const seccion = nodo(documento, "section");
    seccion.className = "dietas-recibo";
    seccion.dataset.dietasBorradorRecibo = "";
    seccion.setAttribute("role", "status");
    seccion.setAttribute("aria-live", "polite");
    seccion.append(
      nodo(documento, "strong", traducir("borradores_propios_recibo_titulo")),
    );
    const datos = nodo(documento, "dl");
    [
      ["borradores_propios_recibo_referencia", item.recibo.referencia],
      ["borradores_propios_recibo_version", String(item.recibo.version)],
      [
        "borradores_propios_recibo_fecha",
        fechaLegible(item.recibo.registrado_en, true),
      ],
    ].forEach(([etiqueta, valor]) => {
      const fila = nodo(documento, "div");
      fila.append(
        nodo(documento, "dt", traducir(etiqueta)),
        nodo(documento, "dd", valor),
      );
      datos.append(fila);
    });
    seccion.append(datos);
    if (item.recibo.repeticion)
      seccion.append(
        nodo(documento, "p", traducir("borradores_propios_repeticion")),
      );
    return seccion;
  }
  function listado() {
    const seccion = nodo(documento, "section");
    seccion.className = "panel dietas-listado";
    seccion.append(
      nodo(documento, "h3", traducir("borradores_propios_listado")),
    );
    if (!conectada) {
      seccion.append(
        nodo(documento, "p", traducir("borradores_propios_pendiente_conexion")),
      );
      return seccion;
    }
    if (estado.carga) {
      seccion.append(
        nodo(documento, "p", traducir("borradores_propios_cargando")),
      );
      return seccion;
    }
    if (!estado.items.length) {
      const vacio = nodo(documento, "p", traducir("borradores_propios_vacio"));
      vacio.dataset.dietasBorradoresVacio = "";
      seccion.append(vacio);
      return seccion;
    }
    const ul = nodo(documento, "ul");
    estado.items.forEach((item) => {
      const li = nodo(documento, "li");
      const boton = nodo(
        documento,
        "button",
        `${fechaLegible(item.comision.fecha_inicio)} · ${item.comision.motivo}`,
      );
      boton.type = "button";
      boton.className = "enlace-tabla";
      boton.dataset.dietasBorradorDetalle = item.comision.referencia;
      boton.dataset.dietasBorradorRelacion = item.comision.relacion_ref;
      boton.setAttribute(
        "aria-label",
        `${traducir("borradores_propios_seleccionar")}: ${item.comision.referencia}`,
      );
      li.append(
        boton,
        nodo(
          documento,
          "span",
          ` · ${traducir("borradores_propios_estado_borrador")}`,
        ),
      );
      ul.append(li);
    });
    seccion.append(ul);
    return seccion;
  }
  function detalle() {
    const seccion = nodo(documento, "section");
    seccion.className = "panel dietas-detalle";
    seccion.append(
      nodo(documento, "h3", traducir("borradores_propios_detalle")),
    );
    const item = estado.detalle;
    if (!item) {
      seccion.append(
        nodo(documento, "p", traducir("borradores_propios_sin_detalle")),
      );
      return seccion;
    }
    const datos = nodo(documento, "dl");
    [
      ["borradores_propios_referencia", item.comision.referencia],
      [
        "borradores_propios_fecha_inicio",
        fechaLegible(item.comision.fecha_inicio),
      ],
      ["borradores_propios_fecha_fin", fechaLegible(item.comision.fecha_fin)],
      ["borradores_propios_motivo", item.comision.motivo],
      ["borradores_propios_ruta", item.comision.codigos_ruta.join(", ") || "—"],
    ].forEach(([etiqueta, valor]) => {
      const fila = nodo(documento, "div");
      fila.append(
        nodo(documento, "dt", traducir(etiqueta)),
        nodo(documento, "dd", valor),
      );
      datos.append(fila);
    });
    seccion.append(datos);
    if (item.comision.calculo) seccion.append(resumenCalculo(item.comision.calculo));
    return seccion;
  }
  function resumenCalculo(calculo) {
    const resumen=nodo(documento,"section"); resumen.className="dietas-comision-calculo";
    resumen.append(nodo(documento,"h4",traducir("borradores_propios_calculo")));
    const cifras=nodo(documento,"div"); cifras.className="dietas-comision-cifras";
    [[traducir("borradores_propios_km"),`${Number(calculo.kilometros).toLocaleString("es-ES",{maximumFractionDigits:1})} km`],
      [traducir("borradores_propios_importe_km"),euros(calculo.importe_kilometraje_centimos)]].forEach(([titulo,valor])=>{
        const tarjeta=nodo(documento,"article"); tarjeta.className="tarjeta-kpi";
        tarjeta.append(nodo(documento,"small",titulo),nodo(documento,"strong",valor)); cifras.append(tarjeta);
      });
    resumen.append(cifras,nodo(documento,"p",`${calculo.rotulo} · ${calculo.version_tarifa} · ${calculo.version_grafo}`));
    const ruta=nodo(documento,"ol"); ruta.className="dietas-comision-tramos-ruta";
    calculo.tramos_ruta.forEach((tramo)=>ruta.append(nodo(documento,"li",`${NOMBRES_RUTA.get(tramo.origen_codigo)||tramo.origen_codigo} → ${NOMBRES_RUTA.get(tramo.destino_codigo)||tramo.destino_codigo} · ${Number(tramo.kilometros).toLocaleString("es-ES",{maximumFractionDigits:1})} km`)));
    resumen.append(nodo(documento,"h5",traducir("borradores_propios_tramos_ruta")),ruta);
    const aviso=nodo(documento,"p",traducir("borradores_propios_grupo_pendiente")); aviso.className="estado-chip aviso"; resumen.append(aviso);
    calculo.opciones_dieta.forEach((opcion)=>{
      const bloque=nodo(documento,"section"); bloque.className="dietas-comision-grupo";
      bloque.append(nodo(documento,"h5",`${traducir("borradores_propios_grupo")} ${opcion.grupo} · ${euros(opcion.calculo.total_maximo_orientativo_centimos)}`));
      const lista=nodo(documento,"ul");
      opcion.calculo.tramos.forEach((tramo)=>lista.append(nodo(documento,"li",`${tramo.fecha} · ${tramo.tipo==="manutencion"?traducir("borradores_propios_manutencion"):traducir("borradores_propios_alojamiento_tope")} ${tramo.porcentaje}% · ${euros(tramo.importe_centimos)}`)));
      bloque.append(lista); resumen.append(bloque);
    });
    return resumen;
  }
  function pintar() {
    if (!activaAhora()) return;
    raiz.replaceChildren(
      nodo(documento, "h2", traducir("borradores_propios_titulo")),
    );
    const aviso = nodo(documento, "p", traducir(estado.mensaje));
    aviso.dataset.dietasBorradoresEstado = "";
    aviso.setAttribute("role", estado.tono === "error" ? "alert" : "status");
    aviso.setAttribute("aria-live", "polite");
    raiz.append(aviso, formulario(), listado(), detalle());
    if (seleccionada) raiz.append(recibo(seleccionada));
  }
  async function cargar() {
    if (!conectada) {
      pintar();
      return;
    }
    if (relaciones.length > 1 && !relacionSeleccionada) {
      estado = { ...estado, carga: false, items: [], detalle: null };
      mensaje("borradores_propios_error_relacion", "aviso");
      pintar();
      return;
    }
    controlador?.abort();
    controlador = new AbortController();
    estado = { ...estado, carga: true };
    pintar();
    const signal = controlador.signal;
    try {
      const pagina = await cliente.listar(
        {
          limit: 50,
          ...(relacionSeleccionada
            ? { relacion_ref: relacionSeleccionada }
            : {}),
        },
        { signal },
      );
      if (!activaAhora() || signal.aborted) return;
      estado = {
        ...estado,
        carga: false,
        items: pagina.items,
        mensaje: "borradores_propios_listado",
      };
    } catch (error) {
      if (!activaAhora() || signal.aborted) return;
      estado = { ...estado, carga: false };
      mensaje(errorClave(error), "error");
    } finally {
      if (controlador?.signal === signal) controlador = null;
      if (activaAhora()) pintar();
    }
  }
  async function enviar(evento) {
    const form = evento.target?.closest?.("[data-dietas-borrador-form]");
    if (!form || !activaAhora()) return;
    evento.preventDefault();
    if (!form.checkValidity?.()) {
      form.reportValidity?.();
      return;
    }
    const datos = new FormData(form);
    const base = {
      fecha_inicio: datos.get("fecha_inicio"),
      fecha_fin: datos.get("fecha_fin"),
      motivo: String(datos.get("motivo") || "").trim(),
      ...(datos.get("relacion_ref")
        ? { relacion_ref: String(datos.get("relacion_ref")) }
        : {}),
      hora_inicio: String(datos.get("hora_inicio")||""),
      hora_fin: String(datos.get("hora_fin")||""),
      codigos_ruta: [String(datos.get("origen_codigo")||""),String(datos.get("destino_codigo")||"")],
    };
    if (base.codigos_ruta[0] === base.codigos_ruta[1]) { mensaje("borradores_propios_ruta_distinta","aviso"); pintar(); return; }
    const contenido = claveContenido(base);
    if (pendiente && pendiente.contenido !== contenido) {
      pendiente = null;
      mensaje("borradores_propios_cambio_intencion", "aviso");
      pintar();
      return;
    }
    if (controlador) return;
    const clave = pendiente?.clave || generarClaveIdempotencia();
    if (typeof clave !== "string" || !clave) {
      mensaje("borradores_propios_error", "error");
      pintar();
      return;
    }
    pendiente = { clave, contenido };
    const solicitud = { clave_idempotencia: clave, ...base };
    controlador = new AbortController();
    const signal = controlador.signal;
    mensaje("borradores_propios_enviando");
    pintar();
    try {
      const item = await cliente.crear(solicitud, { signal });
      if (!activaAhora() || signal.aborted) return;
      pendiente = null;
      seleccionada = item;
      estado = {
        ...estado,
        detalle: item,
        items: [
          item,
          ...estado.items.filter(
            (fila) => fila.comision.referencia !== item.comision.referencia,
          ),
        ],
      };
      mensaje(
        item.recibo.repeticion
          ? "borradores_propios_repeticion"
          : "borradores_propios_creado",
        "exito",
      );
    } catch (error) {
      if (!activaAhora() || signal.aborted) return;
      if (!error?.resultadoIndeterminado) pendiente = null;
      mensaje(errorClave(error), "error");
    } finally {
      if (controlador?.signal === signal) controlador = null;
      if (activaAhora()) pintar();
    }
  }
  async function clic(evento) {
    const boton = evento.target?.closest?.("[data-dietas-borrador-detalle]");
    if (!boton || controlador || !activaAhora()) return;
    const referencia = boton.dataset.dietasBorradorDetalle;
    controlador = new AbortController();
    const signal = controlador.signal;
    mensaje("borradores_propios_cargando");
    pintar();
    try {
      const item = await cliente.obtener(referencia, {
        signal,
        relacion_ref: boton.dataset.dietasBorradorRelacion || undefined,
      });
      if (!activaAhora() || signal.aborted) return;
      estado = { ...estado, detalle: item };
      mensaje("borradores_propios_detalle");
    } catch (error) {
      if (!activaAhora() || signal.aborted) return;
      mensaje(errorClave(error), "error");
    } finally {
      if (controlador?.signal === signal) controlador = null;
      if (activaAhora()) pintar();
    }
  }
  function cambiarRelacion(evento) {
    const selector = evento.target?.closest?.('[name="relacion_ref"]');
    if (!selector || relaciones.length < 2 || !activaAhora()) return;
    relacionSeleccionada = relaciones.includes(selector.value)
      ? selector.value
      : undefined;
    seleccionada = null;
    estado = { ...estado, items: [], detalle: null };
    cargar();
  }
  raiz.addEventListener("submit", enviar);
  raiz.addEventListener("click", clic);
  raiz.addEventListener("change", cambiarRelacion);
  pintar();
  cargar();
  return Object.freeze({ desmontar, recargar: cargar });
}
