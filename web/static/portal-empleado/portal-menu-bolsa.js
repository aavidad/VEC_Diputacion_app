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

export function categoriaDeVistaBolsa(vista) {
  if (typeof vista !== "string") return "";
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
