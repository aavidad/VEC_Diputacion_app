import { cargarTextos, reintentarTextos } from "../comun/textos.js";
import { idiomaActivoAreaPersonal, traducir } from "./i18n.js";
import { crearClienteCartera, ErrorCartera } from "./meritos-reales-cliente.js?v=20261015-o-justificante-bolsa-v1";
import { renderizarMeritosPropios } from "./meritos-reales-vista.js?v=20261015-o-justificante-bolsa-v1";

const FILTROS = new Set(["todos", "disponibles", "retirados"]);

function estadoInicial(ubicacion) {
  const filtro = new URLSearchParams(ubicacion?.search || "").get("filtro");
  return { estado: "cargando", filtro: FILTROS.has(filtro) ? filtro : "todos", hechos: [],
    total: 0, cursor_siguiente: "", motivos: [], seleccion: null, confirmacion: null,
    pendiente: null, guardando: false, alta: null, altaPendiente: null, confirmacionAlta: null, errorAlta: "",
    estadoArchivo: "", estadoCarga: null, comprobante: null, errorDescarga: false };
}

// La vista reemplaza el HTML al cambiar de estado. Se identifica el control
// activo por su acción, nunca por una referencia interpolada en un selector.
function restauradorFoco(contenedor, activo) {
  if (!activo || !contenedor.contains(activo)) return null;
  if (activo.dataset?.meritosFiltro !== undefined) {
    const filtro = activo.dataset.meritosFiltro;
    return () => [...contenedor.querySelectorAll("[data-meritos-filtro]")]
      .find((nodo) => nodo.dataset.meritosFiltro === filtro);
  }
  if (activo.dataset?.meritosRetirar !== undefined) {
    const hecho = activo.dataset.meritosRetirar;
    return () => [...contenedor.querySelectorAll("[data-meritos-retirar]")]
      .find((nodo) => nodo.dataset.meritosRetirar === hecho);
  }
  if (activo.dataset?.meritosDescargar !== undefined) {
    const ref = activo.dataset.meritosDescargar;
    return () => [...contenedor.querySelectorAll("[data-meritos-descargar]")]
      .find((nodo) => nodo.dataset.meritosDescargar === ref);
  }
  if (activo.id === "meritos-retirada-motivo") return () => contenedor.querySelector("#meritos-retirada-motivo");
  for (const marca of ["meritosConsultar", "meritosMas", "meritosCancelar", "meritosTextosReintentar"]) {
    if (activo.dataset?.[marca] !== undefined) {
      const atributo = marca.replace(/[A-Z]/gu, (letra) => `-${letra.toLowerCase()}`);
      return () => contenedor.querySelector(`[data-${atributo}]`);
    }
  }
  return null;
}

/** Carga catálogo y datos solo al abrir la cartera; mantiene la clave incierta en memoria. */
export function montarMeritosReales({ contenedor, fetchImpl = globalThis.fetch,
  idioma = idiomaActivoAreaPersonal(), ubicacion = globalThis.location,
  historial = globalThis.history, pendiente = null } = {}) {
  if (!contenedor || typeof contenedor.replaceChildren !== "function") {
    throw new TypeError("meritos.cartera.contenedor_invalido");
  }
  const cliente = crearClienteCartera({ fetchImpl });
  const doc = contenedor.ownerDocument || document;
  const estilo = doc.createElement("link");
  estilo.rel = "stylesheet";
  estilo.href = "/area-personal/meritos-reales-estilos.css?v=20261014-aspirante-pdf-local-v1";
  doc.head.append(estilo);
  const modelo = estadoInicial(ubicacion);
  modelo.pendiente = pendiente?.operacion === "declaracion" ? null : pendiente;
  if (pendiente?.operacion === "declaracion") {
    modelo.altaPendiente = pendiente.orden; modelo.alta = pendiente.alta;
    modelo.comprobante = pendiente.comprobante || null;
    modelo.estadoCarga = pendiente.estadoCarga || null;
    modelo.estadoArchivo = modelo.estadoCarga?.estado || "pendiente";
    modelo.errorAlta = modelo.alta?.archivo || modelo.estadoCarga ? "" : "alta_error_incierto";
  }
  let activo = true;
  let textos = null;
  let generacion = 0;
  let lectura = null;
  let descarga = null;
  let operacion = null;

  const pintar = () => {
    if (activo && textos && contenedor.isConnected) {
      const restaurar = restauradorFoco(contenedor, doc.activeElement);
      contenedor.innerHTML = renderizarMeritosPropios(modelo, textos);
      if (restaurar) {
        const destino = restaurar() ?? contenedor.querySelector(".meritos-reales-pendiente")
          ?? contenedor.querySelector(".meritos-reales-confirmacion")
          ?? contenedor.querySelector(".meritos-reales-aviso")
          ?? contenedor.querySelector("#meritos-cartera-titulo");
        destino?.focus?.({ preventScroll: true });
      }
    }
  };

  async function consultar({ siguiente = false } = {}) {
    const version = ++generacion;
    lectura?.abort();
    lectura = new AbortController();
    if (!siguiente) {
      modelo.estado = "cargando";
      modelo.hechos = [];
      modelo.cursor_siguiente = "";
    }
    pintar();
    try {
      const pagina = await cliente.consultar({ filtro: modelo.filtro, idioma: textos.idioma, limite: 25,
        cursor: siguiente ? modelo.cursor_siguiente : "", signal: lectura.signal });
      if (!activo || version !== generacion) return;
      modelo.hechos = siguiente ? [...modelo.hechos, ...pagina.hechos] : pagina.hechos;
      modelo.total = pagina.total;
      modelo.declaradaEn = pagina.recibo_acceso?.consultada_en;
      modelo.cursor_siguiente = pagina.cursor_siguiente;
      modelo.motivos = Array.isArray(pagina.motivos_retirada) ? pagina.motivos_retirada : [];
      modelo.estado = modelo.hechos.length ? "listo" : "vacio";
      if (modelo.pendiente) {
        const actual = modelo.hechos.find((h) => h.hecho_ref === modelo.pendiente.hecho_ref);
        if (actual?.disponibilidad?.estado === "retirado" && actual.retirada?.recibo_ref) {
          modelo.confirmacion = { repetida: true, recibo: { referencia: actual.retirada.recibo_ref } };
          modelo.pendiente = null;
          modelo.seleccion = null;
        }
      }
      pintar();
    } catch (error) {
      if (!activo || version !== generacion || error?.name === "AbortError") return;
      modelo.hechos = [];
      modelo.motivos = [];
      modelo.seleccion = null;
      if (error instanceof ErrorCartera && [401, 403].includes(error.estado)) {
        modelo.confirmacion = null;
        modelo.estado = "denegado";
      } else modelo.estado = error instanceof ErrorCartera && error.estado === 409 ? "conflicto" : "error";
      pintar();
    }
  }

  function cambiarFiltro(filtro) {
    if (!FILTROS.has(filtro)) return;
    modelo.filtro = filtro;
    modelo.seleccion = null;
    const url = new URL(ubicacion.href);
    url.searchParams.set("filtro", filtro);
    url.searchParams.delete("cursor");
    historial?.replaceState?.(null, "", url.pathname + url.search + url.hash);
    void consultar();
  }

  async function retirar(formulario) {
    if (modelo.guardando || !modelo.seleccion) return;
    const selectorMotivo = formulario.elements.namedItem("motivo");
    if (!selectorMotivo?.value) {
      selectorMotivo?.reportValidity?.();
      return;
    }
    const indice = Number(selectorMotivo.value);
    const motivo = modelo.motivos[indice]?.motivo;
    const hecho = modelo.hechos.find((h) => h.hecho_ref === modelo.seleccion.hecho_ref);
    if (!motivo || !hecho || hecho.disponibilidad?.estado !== "disponible") return;
    const motivoClave = JSON.stringify(motivo);
    if (modelo.pendiente && (modelo.pendiente.hecho_ref !== hecho.hecho_ref ||
      modelo.pendiente.motivoClave !== motivoClave)) {
      modelo.estado = "resultado_indeterminado";
      pintar();
      return;
    }
    if (!modelo.pendiente && typeof globalThis.crypto?.randomUUID !== "function") {
      modelo.estado = "error";
      pintar();
      return;
    }
    modelo.pendiente ??= { hecho_ref: hecho.hecho_ref, motivoClave,
      orden: { hecho_ref: hecho.hecho_ref, hecho_version_esperada: hecho.version_hecho,
        disponibilidad_version_esperada: hecho.disponibilidad.version, motivo,
        clave_idempotencia: `retiro:${globalThis.crypto.randomUUID()}` } };
    const orden = modelo.pendiente.orden;
    modelo.confirmacion = null;
    modelo.guardando = true;
    pintar();
    try {
      const confirmacion = await cliente.retirar(orden);
      if (!activo) return;
      modelo.confirmacion = confirmacion;
      modelo.pendiente = null;
      modelo.seleccion = null;
      await consultar();
    } catch (error) {
      if (!activo) return;
      modelo.hechos = [];
      modelo.motivos = [];
      modelo.seleccion = null;
      if (error instanceof ErrorCartera && [400, 409, 422].includes(error.estado)) {
        // El servidor rechazó esta orden definitivamente. La siguiente
        // selección tomará versión y clave nuevas después de consultar.
        modelo.pendiente = null;
        modelo.estado = "conflicto";
      } else if (error instanceof ErrorCartera && [401, 403].includes(error.estado)) {
        modelo.pendiente = null;
        modelo.confirmacion = null;
        modelo.estado = "denegado";
      } else if (error instanceof ErrorCartera && error.estado === 404) {
        modelo.pendiente = null;
        modelo.estado = "error";
      } else modelo.estado = "resultado_indeterminado";
    } finally {
      modelo.guardando = false;
      pintar();
      const destino = modelo.confirmacion?.recibo?.referencia ? ".meritos-reales-confirmacion"
        : modelo.pendiente ? ".meritos-reales-pendiente" : ".meritos-reales-aviso";
      contenedor.querySelector(destino)?.focus?.({ preventScroll: true });
    }
  }

  async function declarar(formulario) {
    if (modelo.guardando || !modelo.alta) return;
    if (modelo.estadoArchivo === "rechazado") return;
    if (!modelo.altaPendiente && !formulario.reportValidity()) return;
    const consultarPrimero = !!modelo.altaPendiente
      && ["pendiente", "reservada", "recibido", "analizando"].includes(modelo.estadoArchivo);
    if (!modelo.altaPendiente) {
      const valores = modelo.alta;
      modelo.altaPendiente = { tipo: valores.tipo, denominacion: valores.denominacion.trim(),
        fecha_desde: valores.fecha_desde, ...(valores.fecha_hasta ? { fecha_hasta: valores.fecha_hasta } : {}),
        ...(["curso_asistencia", "curso_superacion"].includes(valores.tipo) && valores.horas !== "" ? { horas: Number(valores.horas) } : {}),
        declarada_en: modelo.declaradaEn, clave_idempotencia: `declaracion:${globalThis.crypto.randomUUID()}` };
    }
    modelo.guardando = true; modelo.errorAlta = ""; modelo.confirmacionAlta = null;
    modelo.estadoArchivo = consultarPrimero ? "consultando"
      : modelo.alta.archivo && !modelo.comprobante ? "enviando" : ""; pintar();
    operacion = new AbortController();
    try {
      if (modelo.alta.archivo && !modelo.comprobante) {
        let estado = null;
        if (consultarPrimero) {
          try {
            estado = await cliente.consultarComprobante(modelo.altaPendiente.clave_idempotencia,
              { signal: operacion.signal });
          } catch (error) {
            if (!(error instanceof ErrorCartera && error.estado === 404
              && error.codigo === "meritos.error.comprobante_no_encontrado")) throw error;
          }
        }
        if (!estado || ["reservada", "recibido", "analizando"].includes(estado.estado)) {
          estado = await cliente.subirComprobante(modelo.altaPendiente.clave_idempotencia,
            modelo.alta.archivo, { signal: operacion.signal });
        }
        if (!activo) return;
        modelo.estadoCarga = estado;
        modelo.estadoArchivo = estado.estado;
        if (estado.estado === "rechazado") {
          modelo.altaPendiente = null; modelo.comprobante = null;
          pintar(); return;
        }
        if (estado.estado !== "admitido") { pintar(); return; }
        modelo.comprobante = estado;
        pintar();
      }
      const orden = modelo.comprobante
        ? { ...modelo.altaPendiente, comprobante_ref: modelo.comprobante.comprobante_ref }
        : modelo.altaPendiente;
      const recibo = await cliente.declarar(orden, { signal: operacion.signal });
      if (!activo) return;
      modelo.confirmacionAlta = recibo; modelo.alta = null; modelo.altaPendiente = null;
      modelo.comprobante = null; modelo.estadoCarga = null; modelo.estadoArchivo = "";
      await consultar();
    } catch (error) {
      if (!activo) return;
      if (error instanceof ErrorCartera && [401, 403].includes(error.estado)) {
        modelo.altaPendiente = null; modelo.alta = null; modelo.comprobante = null; modelo.estadoCarga = null;
        modelo.hechos = []; modelo.estado = "denegado";
      } else if (error instanceof ErrorCartera && [400, 409, 422, 404].includes(error.estado)) {
        modelo.altaPendiente = null; modelo.comprobante = null; modelo.estadoCarga = null; modelo.estadoArchivo = "error";
        modelo.errorAlta = "alta_error_datos";
      } else {
        modelo.estadoArchivo = modelo.alta?.archivo && !modelo.comprobante ? "pendiente" : modelo.estadoArchivo;
        modelo.errorAlta = consultarPrimero ? "comprobante_consulta_error"
          : modelo.alta?.archivo && !modelo.comprobante ? "" : "alta_error_incierto";
      }
    } finally {
      operacion = null; modelo.guardando = false; pintar();
      if (activo) (contenedor.querySelector(modelo.confirmacionAlta ? ".meritos-reales-confirmacion"
        : modelo.altaPendiente ? "#meritos-alta-accion" : "#meritos-alta-titulo")
        ?? contenedor.querySelector("#meritos-cartera-titulo"))?.focus?.({ preventScroll: true });
    }
  }

  async function descargar(ref) {
    descarga?.abort();
    descarga = new AbortController();
    modelo.errorDescarga = false;
    pintar();
    try {
      const archivo = await cliente.descargarComprobante(ref, { signal: descarga.signal });
      if (!activo) return;
      const url = URL.createObjectURL(archivo);
      const enlace = doc.createElement("a");
      enlace.href = url;
      enlace.download = textos.traducir("propio.comprobante_nombre_archivo");
      enlace.rel = "noopener";
      enlace.click();
      setTimeout(() => URL.revokeObjectURL(url), 60_000);
    } catch (error) {
      if (!activo || error?.name === "AbortError") return;
      modelo.errorDescarga = error instanceof ErrorCartera
        && error.codigo === "meritos.error.comprobante_no_disponible" ? "pdf_no_disponible" : "descarga_error";
      pintar();
    }
  }

  const alClic = (evento) => {
    const boton = evento.target?.closest?.("button");
    if (!boton || !contenedor.contains(boton)) return;
    if (boton.hasAttribute("data-meritos-alta")) {
      modelo.alta = { tipo: "titulacion", denominacion: "", fecha_desde: "", fecha_hasta: "", horas: "", archivo: null };
      modelo.errorAlta = ""; pintar(); contenedor.querySelector('[name="denominacion"]')?.focus();
    } else if (boton.hasAttribute("data-meritos-alta-cancelar") && !modelo.guardando && !modelo.altaPendiente) {
      modelo.alta = null; modelo.comprobante = null; modelo.estadoCarga = null; modelo.estadoArchivo = "";
      modelo.errorAlta = ""; pintar(); contenedor.querySelector("[data-meritos-alta]")?.focus();
    } else if (boton.hasAttribute("data-meritos-archivo-quitar") && modelo.alta && !modelo.altaPendiente && !modelo.guardando) {
      modelo.alta.archivo = null; modelo.comprobante = null; modelo.estadoCarga = null; modelo.estadoArchivo = "";
      pintar(); contenedor.querySelector('[name="documento"]')?.focus();
    } else if (boton.hasAttribute("data-meritos-textos-reintentar")) void prepararTextos();
    else if (boton.dataset.meritosFiltro) cambiarFiltro(boton.dataset.meritosFiltro);
    else if (boton.hasAttribute("data-meritos-consultar")) void consultar();
    else if (boton.hasAttribute("data-meritos-mas") && modelo.cursor_siguiente) void consultar({ siguiente: true });
    else if (boton.dataset.meritosDescargar) void descargar(boton.dataset.meritosDescargar);
    else if (boton.dataset.meritosRetirar) {
      modelo.seleccion = { hecho_ref: boton.dataset.meritosRetirar };
      pintar();
      contenedor.querySelector("#meritos-retirada-motivo")?.focus();
    } else if (boton.hasAttribute("data-meritos-cancelar")) {
      const hecho = modelo.seleccion?.hecho_ref;
      modelo.seleccion = null;
      pintar();
      [...contenedor.querySelectorAll("[data-meritos-retirar]")]
        .find((nodo) => nodo.dataset.meritosRetirar === hecho)?.focus?.({ preventScroll: true });
    }
  };
  const alEnviar = (evento) => {
    if (evento.target?.id === "meritos-reales-alta-formulario") {
      evento.preventDefault(); void declarar(evento.target); return;
    }
    if (evento.target?.id !== "meritos-reales-retirada-formulario") return;
    evento.preventDefault();
    void retirar(evento.target);
  };
  contenedor.addEventListener("click", alClic);
  contenedor.addEventListener("submit", alEnviar);
  const alEditarAlta = (evento) => {
    const control = evento.target;
    if (!modelo.alta || modelo.altaPendiente || !control?.closest?.("#meritos-reales-alta-formulario")
      || (control.name !== "documento" && !Object.hasOwn(modelo.alta, control.name))) return;
    if (control.name === "documento") {
      if (evento.type !== "change") return;
      const archivo = control.files?.[0] || null;
      if (archivo && (archivo.type !== "application/pdf" && !/\.pdf$/iu.test(archivo.name))) {
        modelo.errorAlta = "comprobante_formato"; control.value = ""; pintar(); return;
      }
      modelo.alta.archivo = archivo; modelo.comprobante = null; modelo.estadoCarga = null;
      modelo.estadoArchivo = archivo ? "seleccionado" : ""; modelo.errorAlta = "";
      pintar(); contenedor.querySelector(archivo ? "[data-meritos-archivo-quitar]" : '[name="documento"]')?.focus(); return;
    }
    if (evento.type === "change") return;
    modelo.alta[control.name] = control.value;
    if (control.name === "tipo") { pintar(); contenedor.querySelector('[name="tipo"]')?.focus(); }
  };
  contenedor.addEventListener("input", alEditarAlta);
  contenedor.addEventListener("change", alEditarAlta);
  async function prepararTextos() {
    try {
      const cargados = await cargarTextos("meritos-reales", { idioma })
        .catch(() => reintentarTextos("meritos-reales", { idioma }));
      if (!activo) return;
      textos = cargados;
      pintar();
      await consultar();
    } catch {
      if (!activo || !contenedor.isConnected) return;
      const aviso = doc.createElement("p");
      aviso.setAttribute("role", "alert");
      aviso.textContent = traducir("areaPersonal.estado.error.detalle");
      const boton = doc.createElement("button");
      boton.type = "button";
      boton.className = "boton-secundario";
      boton.dataset.meritosTextosReintentar = "";
      boton.textContent = traducir("areaPersonal.estado.error.reintentar");
      contenedor.replaceChildren(aviso, boton);
    }
  }
  const preparada = prepararTextos();
  return Object.freeze({ preparada,
    desmontar() { lectura?.abort(); descarga?.abort(); operacion?.abort(); generacion++; contenedor.removeEventListener("click", alClic); contenedor.removeEventListener("submit", alEnviar);
      contenedor.removeEventListener("input", alEditarAlta); contenedor.removeEventListener("change", alEditarAlta);
      contenedor.replaceChildren(); estilo.remove(); activo = false; },
    pendiente() { return modelo.altaPendiente ? { operacion: "declaracion", orden: modelo.altaPendiente,
      alta: modelo.alta, comprobante: modelo.comprobante, estadoCarga: modelo.estadoCarga } : modelo.pendiente; },
  });
}
