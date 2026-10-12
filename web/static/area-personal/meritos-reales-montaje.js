import { cargarTextos, reintentarTextos } from "../comun/textos.js";
import { idiomaActivoAreaPersonal, traducir } from "./i18n.js";
import { crearClienteCartera, ErrorCartera } from "./meritos-reales-cliente.js";
import { renderizarMeritosPropios } from "./meritos-reales-vista.js";

const FILTROS = new Set(["todos", "disponibles", "retirados"]);

function estadoInicial(ubicacion) {
  const filtro = new URLSearchParams(ubicacion?.search || "").get("filtro");
  return { estado: "cargando", filtro: FILTROS.has(filtro) ? filtro : "todos", hechos: [],
    total: 0, cursor_siguiente: "", motivos: [], seleccion: null, confirmacion: null,
    pendiente: null, guardando: false };
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
  estilo.href = "/area-personal/meritos-reales-estilos.css";
  doc.head.append(estilo);
  const modelo = estadoInicial(ubicacion);
  modelo.pendiente = pendiente;
  let activo = true;
  let textos = null;
  let generacion = 0;
  let lectura = null;

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

  const alClic = (evento) => {
    const boton = evento.target?.closest?.("button");
    if (!boton || !contenedor.contains(boton)) return;
    if (boton.hasAttribute("data-meritos-textos-reintentar")) void prepararTextos();
    else if (boton.dataset.meritosFiltro) cambiarFiltro(boton.dataset.meritosFiltro);
    else if (boton.hasAttribute("data-meritos-consultar")) void consultar();
    else if (boton.hasAttribute("data-meritos-mas") && modelo.cursor_siguiente) void consultar({ siguiente: true });
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
    if (evento.target?.id !== "meritos-reales-retirada-formulario") return;
    evento.preventDefault();
    void retirar(evento.target);
  };
  contenedor.addEventListener("click", alClic);
  contenedor.addEventListener("submit", alEnviar);
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
    desmontar() { lectura?.abort(); generacion++; contenedor.removeEventListener("click", alClic); contenedor.removeEventListener("submit", alEnviar); contenedor.replaceChildren(); estilo.remove(); activo = false; },
    pendiente() { return modelo.pendiente; },
  });
}
