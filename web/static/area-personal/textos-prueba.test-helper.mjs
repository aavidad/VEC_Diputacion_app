import { readFile } from "node:fs/promises";

/**
 * Lector de catálogos para pruebas del área personal. Devuelve los catálogos
 * reales de `textos/` y, en `area-personal.json`, aplica `cambios` con claves
 * planas `areaPersonal.<sección>.<clave>` (como las usa la interfaz). Anota en
 * `leer.pedidas` cada catálogo solicitado como `<idioma>/<módulo>.json`.
 */
export function lectorCatalogos(cambios = {}) {
  const pedidas = [];
  async function leer(url) {
    pedidas.push(url.pathname.split("/textos/").at(-1));
    const datos = JSON.parse(await readFile(url, "utf8"));
    if (url.pathname.endsWith("/area-personal.json")) aplicarCambios(datos, cambios);
    return datos;
  }
  leer.pedidas = pedidas;
  return leer;
}

function aplicarCambios(datos, cambios) {
  for (const [clave, texto] of Object.entries(cambios)) {
    const partes = clave.replace(/^areaPersonal\./u, "").split(".");
    let nodo = datos;
    for (const parte of partes.slice(0, -1)) {
      if (typeof nodo[parte] === "string") nodo[parte] = { _: nodo[parte] };
      nodo[parte] ??= {};
      nodo = nodo[parte];
    }
    const ultima = partes.at(-1);
    if (nodo[ultima] && typeof nodo[ultima] === "object") nodo[ultima]._ = texto;
    else nodo[ultima] = texto;
  }
}

/** Catálogo plano real (`areaPersonal.…` → texto) de un idioma, para comparar. */
export async function catalogoPlano(idioma) {
  const raiz = new URL("../textos/", import.meta.url);
  const salida = {};
  const aplanar = (seccion, prefijo) => {
    for (const [clave, valor] of Object.entries(seccion)) {
      const ruta = clave === "_" ? prefijo : `${prefijo}.${clave}`;
      if (typeof valor === "string") salida[ruta] = valor;
      else if (valor && typeof valor === "object" && typeof valor.other !== "string") aplanar(valor, ruta);
    }
  };
  const preferencias = JSON.parse(await readFile(new URL(`${idioma}/preferencias.json`, raiz), "utf8"));
  aplanar(preferencias.areaPersonal, "areaPersonal");
  aplanar(JSON.parse(await readFile(new URL(`${idioma}/area-personal.json`, raiz), "utf8")), "areaPersonal");
  return salida;
}
