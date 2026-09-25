/**
 * Navegación compacta de Bolsa basada en las diez áreas funcionales de RRHH.
 *
 * Las categorías solo ordenan enlaces del router existente. No deciden
 * permisos, no cargan datos y no conservan estado en el navegador.
 */
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

export function resumenAccesosModulos(accesos, comprobandoBolsas) {
  if (comprobandoBolsas || accesos.some((acceso) => acceso.estado === "cargando")) {
    return "Fase inicial: comprobando módulos";
  }
  const disponibles = accesos.filter((acceso) => acceso.disponible === true).length;
  return disponibles > 0
    ? `${disponibles} ${disponibles === 1 ? "módulo habilitado" : "módulos habilitados"} en fase inicial`
    : "Módulos pendientes de sesión autorizada";
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
