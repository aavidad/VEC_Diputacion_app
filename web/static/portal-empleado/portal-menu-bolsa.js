/**
 * Navegación compacta de Bolsa basada en las diez áreas funcionales de RRHH.
 *
 * Las categorías solo ordenan enlaces del router existente. No deciden
 * permisos, no cargan datos y no conservan estado en el navegador.
 */
import { traducirPortal } from "./portal-i18n.js?v=20260925-cronos-notif-e10-v1";
const VISTAS_POR_CATEGORIA = Object.freeze({
  "bolsas-candidatos": Object.freeze([
    "elaboracion", "convocatorias", "solicitudes", "meritos", "alegaciones", "importacion",
  ]),
  llamamientos: Object.freeze(["llamamientos"]),
  contratos: Object.freeze(["contratos"]),
  reglas: Object.freeze(["reglas", "baremacion"]),
  consulta: Object.freeze(["consulta"]),
  resumen: Object.freeze(["resumen"]),
  estadisticas: Object.freeze(["estadisticas"]),
  documentos: Object.freeze(["documentos"]),
  comunicaciones: Object.freeze(["comunicaciones"]),
  auditoria: Object.freeze(["auditoria", "configuracion"]),
});

const CATEGORIAS_EXPANDIBLES = Object.freeze([
  "bolsas-candidatos", "reglas", "auditoria",
]);

// La ficha B5 no es un segundo enlace del menú: se alcanza desde el cuadro de
// bolsas al elegir una bolsa concreta. Sí forma parte de la navegación interna
// y debe conservar categoría activa cuando el usuario llega por ese recorrido.
export const VISTA_CANDIDATOS_BOLSA = "bolsa-candidatos";
export const VISTAS_INTERNAS_BOLSA = Object.freeze([
  ...Object.values(VISTAS_POR_CATEGORIA).flat(),
  VISTA_CANDIDATOS_BOLSA,
]);

// Se mantienen las rutas para la composición futura, pero estos puntos no
// tienen todavía un servicio autorizado que pueda ejecutar su operación.
export const VISTAS_BOLSA_PENDIENTES_NO_COMPUESTAS = Object.freeze([
  "llamamientos", "contratos", "documentos", "comunicaciones",
]);

export function vistaBolsaPendienteNoCompuesta(vista) {
  return VISTAS_BOLSA_PENDIENTES_NO_COMPUESTAS.includes(vista);
}

// El módulo Bolsa está disponible si lo está la elaboración de borradores o,
// en su defecto, si la API del cuadro de bolsas ha respondido con su contrato:
// entonces la entrada del menú abre el cuadro en vez de marcarse «no
// disponible». Un cuadro sin bolsas sigue siendo un módulo compuesto y
// autorizado; lo que no cuenta es un fallo, una denegación o una consulta
// pendiente.
export function accesoBolsaEfectivo(accesoBorradores, datosBolsas) {
  if (accesoBorradores && accesoBorradores.disponible === true) return accesoBorradores;
  const bolsas = datosBolsas?.carga === "listo" ? datosBolsas.datos?.bolsas : null;
  if (!Array.isArray(bolsas)) return accesoBorradores;
  return Object.freeze({ disponible: true, vista: "resumen", estado: "disponible", etiqueta: "Cuadro de bolsas" });
}

export function resumenAccesosModulos(accesos, comprobandoBolsas, traducir = traducirPortal) {
  if (comprobandoBolsas || accesos.some((acceso) => acceso.estado === "cargando")) {
    return traducir("resumen_modulos_comprobando");
  }
  const disponibles = accesos.filter((acceso) => acceso.disponible === true).length;
  if (disponibles === 0) return traducir("resumen_modulos_ninguno");
  const cantidad = new Intl.NumberFormat("es-ES").format(disponibles);
  return traducir(new Intl.PluralRules("es-ES").select(disponibles) === "one"
    ? "resumen_modulos_uno" : "resumen_modulos_varios", { cantidad });
}

/**
 * Decide si una entrada del menú de Bolsa se ofrece según la capacidad real
 * que la sirve: el cuadro, los candidatos, las estadísticas y el llamamiento
 * desde cada bolsa usan la API del cuadro (ya comprobada para abrir Bolsa);
 * Elaboración, su API de borradores; «Documentos y firma», la vista de
 * Contratación temporal; y el resto, el panel interno agregado. Sin esa
 * capacidad la entrada no se ofrece, en lugar de abrir una pantalla vacía.
 *
 * La API de borradores se comprueba al cargar el portal, en paralelo: mientras
 * no conste que falta (`borradores` distinto de `false`) Elaboración se ofrece,
 * y al abrirla muestra el resultado de esa misma comprobación.
 */
export function vistaBolsaOfrecida(vista, capacidades = {}) {
  switch (vista) {
    case "resumen":
    case "estadisticas":
    case "llamamientos":
    case VISTA_CANDIDATOS_BOLSA:
      return true;
    case "elaboracion":
      return capacidades?.borradores !== false;
    case "contratacion-temporal":
      return capacidades?.contratacionTemporal === true;
    default:
      return VISTAS_INTERNAS_BOLSA.includes(vista) && capacidades?.panelInterno === true;
  }
}

/**
 * Navegación directa (enlace o historial) a una vista de Bolsa: se permite lo
 * mismo que el menú ofrece.
 */
export function vistaBolsaNavegable(vista, capacidades = {}) {
  return vistaBolsaOfrecida(vista, capacidades);
}

// Vista que ofrece una entrada de primer nivel. «Correo y mensajería» abre hoy
// el llamamiento, pero su capacidad propia es la de su categoría; «Documentos y
// firma» lleva a Contratación temporal.
function vistaDeControl(control) {
  const vista = control?.getAttribute?.("data-vista") || "";
  const categoria = control?.dataset?.categoriaBolsa || "";
  const categoriaVista = categoriaDeVistaBolsa(vista);
  if (categoria && categoriaVista && categoriaVista !== categoria) {
    return VISTAS_POR_CATEGORIA[categoria]?.[0] || "";
  }
  return vista;
}

/**
 * Oculta del menú de Bolsa las entradas sin capacidad y los grupos que quedan
 * vacíos. Devuelve los indicadores numéricos de las categorías visibles, en
 * orden, para que el shell los renumere seguidos.
 */
export function aplicarDisponibilidadMenuBolsa(raiz, capacidades = {}) {
  if (!raiz?.querySelectorAll) return [];
  raiz.querySelectorAll(".submenu-bolsa [data-vista]").forEach((control) => {
    control.hidden = !vistaBolsaOfrecida(control.getAttribute?.("data-vista"), capacidades);
  });
  raiz.querySelectorAll(".grupo-menu-bolsa").forEach((grupo) => {
    grupo.hidden = !Array.from(grupo.querySelectorAll(".submenu-bolsa [data-vista]"))
      .some((control) => !control.hidden);
  });
  const indicadores = [];
  raiz.querySelectorAll(".categoria-menu-bolsa").forEach((control) => {
    const grupo = control.dataset.grupoBolsa ? control.closest?.(".grupo-menu-bolsa") : null;
    const visible = grupo ? !grupo.hidden : vistaBolsaOfrecida(vistaDeControl(control), capacidades);
    if (!grupo) control.hidden = !visible;
    const indicador = control.querySelector?.(".numero-menu");
    if (visible && indicador) indicadores.push(indicador);
  });
  return indicadores;
}

export function categoriaDeVistaBolsa(vista) {
  if (typeof vista !== "string") return "";
  if (vista === VISTA_CANDIDATOS_BOLSA) return "bolsas-candidatos";
  return Object.entries(VISTAS_POR_CATEGORIA)
    .find(([, vistas]) => vistas.includes(vista))?.[0] || "";
}

function obtenerSubmenu(raiz, boton) {
  const identificador = boton?.getAttribute?.("aria-controls") || "";
  if (!/^[a-z][a-z0-9-]{0,63}$/.test(identificador)) return null;
  return raiz?.querySelector?.(`[id="${identificador}"]`) || null;
}

export function establecerExpansionGrupoBolsa(raiz, boton, expandido) {
  const submenu = obtenerSubmenu(raiz, boton);
  if (!submenu) return false;
  const abierto = expandido === true;
  boton.setAttribute("aria-expanded", String(abierto));
  submenu.hidden = !abierto;
  return true;
}

export function alternarGrupoBolsa(raiz, boton) {
  const expandido = boton?.getAttribute?.("aria-expanded") !== "true";
  return establecerExpansionGrupoBolsa(raiz, boton, expandido);
}

export function sincronizarMenuBolsa(raiz, vista) {
  if (!raiz?.querySelectorAll) return;
  const categoriaActiva = categoriaDeVistaBolsa(vista);

  raiz.querySelectorAll("[data-categoria-bolsa]").forEach((control) => {
    if (control.dataset.categoriaBolsa === categoriaActiva) {
      control.dataset.categoriaActiva = "true";
    } else {
      delete control.dataset.categoriaActiva;
    }
  });

  raiz.querySelectorAll("[data-grupo-bolsa]").forEach((boton) => {
    const categoria = boton.dataset.grupoBolsa || "";
    establecerExpansionGrupoBolsa(
      raiz,
      boton,
      CATEGORIAS_EXPANDIBLES.includes(categoria) && categoria === categoriaActiva,
    );
  });

  // Solo las entradas navegables reciben aria-current. B5 no se repite en el
  // menú porque necesita antes una bolsa elegida; su grupo ya queda abierto.
  raiz.querySelectorAll("[data-vista]").forEach((control) => {
    if (control.getAttribute?.("data-vista") === vista) control.setAttribute("aria-current", "page");
    else control.removeAttribute?.("aria-current");
  });
}

export function instalarMenuBolsa(raiz) {
  if (!raiz?.addEventListener || !raiz?.contains) return () => {};
  const manejarClick = (evento) => {
    const boton = evento.target?.closest?.("[data-grupo-bolsa]");
    if (!boton || !raiz.contains(boton)) return;
    evento.preventDefault();
    alternarGrupoBolsa(raiz, boton);
  };
  raiz.addEventListener("click", manejarClick);
  return () => raiz.removeEventListener?.("click", manejarClick);
}

export const CATEGORIAS_MENU_BOLSA = VISTAS_POR_CATEGORIA;
