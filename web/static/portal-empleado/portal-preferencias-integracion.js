import { crearSuperficiePreferenciasPortal } from "./portal-preferencias.js?v=20261001-ct-a-i18n-v1";
import { crearClientePreferencias } from "./portal-preferencias-api.js?v=20260930-codexf-temas-v2";
import { cargarTextosCorreos, crearClienteCorreos, crearSuperficieCorreos } from "../comun/correos-propios.js?v=20260929-correos-508b-v1";
import { crearAvatarCabecera, crearClienteImagen, crearSuperficieImagen, peticionesEnSerie } from "../comun/imagen-propia.js?v=20260929-imagen-508c-v2";
import { aplicarPreferenciasVisuales } from "../comun/tema-vec.js?v=20260930-codexf-temas-v2";
import { IDIOMAS_DISPONIBLES, resolverIdiomaNavegacion } from "../comun/idioma.js";

const textosCorreos = await cargarTextosCorreos();

/** Adapta la autoridad de Usuarios al shell RRHH sin replicar su tema ni guardar datos locales. */
export function crearIntegracionPreferenciasPortal({ documento, ventana, porId, estado, renderizar,
  aplicarFilas, navegar, vistaBolsaDisponible, altaCTDisponible, anunciar, traducir }) {
  let controladorVisual = null;
  function aplicarVisual(valores) {
    const visuales = { tema: valores.tema, alto_contraste: valores.alto_contraste, tamano_texto: valores.tamano_texto };
    if (controladorVisual) controladorVisual.aplicarPreferenciasServidor(visuales);
    else controladorVisual = aplicarPreferenciasVisuales(visuales, { documento, ventana });
    delete documento.body.dataset.textoGrande;
    delete documento.documentElement.dataset.textoGrande;
    porId("boton-texto")?.setAttribute("aria-pressed", String(valores.tamano_texto !== "normal"));
    porId("boton-contraste")?.setAttribute("aria-pressed", String(valores.alto_contraste));
  }
  function alternarVisualVolatil(nombre) {
    const actual = controladorVisual?.leerEstado().preferencias_servidor
      || { tema: "sistema", alto_contraste: false, tamano_texto: "normal" };
    const nuevos = nombre === "texto"
      ? { ...actual, tamano_texto: actual.tamano_texto === "normal" ? "grande" : "normal" }
      : { ...actual, alto_contraste: !actual.alto_contraste };
    aplicarVisual(nuevos);
    return nombre === "texto" ? nuevos.tamano_texto !== "normal" : nuevos.alto_contraste;
  }
  function aplicarIdioma(valores) {
    const url = new URL(ventana.location.href);
    const solicitado = url.searchParams.get("lang");
    const sinIdioma = new URL(url.href);
    sinIdioma.searchParams.delete("lang");
    const idioma = resolverIdiomaNavegacion({ ubicacion: sinIdioma,
      idiomaPreferido: valores.idioma, navegador: ventana.navigator });
    if (IDIOMAS_DISPONIBLES.some(({ codigo }) => codigo === solicitado)) {
      // La URL se usa en esta carga; si coincide con la preferencia resuelta,
      // retirarla evita que bloquee un cambio guardado en una carga futura.
      if (solicitado === idioma) ventana.history.replaceState(null, "", sinIdioma.href);
      return;
    }
    if (idioma === documento.documentElement.lang) return;
    url.searchParams.set("lang", idioma);
    ventana.location.replace(url.href);
  }
  // Preferencias, imagen y correos comparten una cola: la identidad de
  // desarrollo no admite dos altas de sesión simultáneas de la misma cuenta.
  const enSerie = peticionesEnSerie(globalThis.fetch.bind(globalThis));
  const marco = { panel: "panel pref-panel", cabecera: "div", claseCabecera: "cabecera-panel", cuerpo: "cuerpo-panel" };
  const correos = crearSuperficieCorreos({ cliente: crearClienteCorreos({ ruta: "/api/vec/usuarios/mis-correos", fetchImpl: enSerie }),
    textos: textosCorreos, marco });
  const avatar = crearAvatarCabecera(porId("sesion-visible")?.querySelector(".avatar"));
  let iniciales = "";
  const imagen = crearSuperficieImagen({ cliente: crearClienteImagen({ ruta: "/api/vec/usuarios/mi-imagen", fetchImpl: enSerie }),
    textos: textosCorreos, marco, alCambiar: (vista) => avatar.fijarImagen(vista), iniciales: () => iniciales });
  const superficie = crearSuperficiePreferenciasPortal({
    cliente: crearClientePreferencias({ fetchImpl: enSerie }),
    correos, imagen,
    actualizar: () => { if (estado.vista === "mis-preferencias") renderizar(); },
    alCargar: ({ estado: actual }) => {
      aplicarVisual(actual.valores);
      aplicarFilas(actual.valores.filas);
      aplicarIdioma(actual.valores);
    },
    alGuardar: (recibo) => {
      aplicarVisual(recibo.valores);
      aplicarFilas(recibo.valores.filas);
    },
  });
  function instalarMenu() {
    const boton = porId("boton-identidad");
    const menu = porId("menu-identidad");
    const opcion = porId("abrir-mis-preferencias");
    if (!boton || !menu || !opcion) return;
    const cerrar = (restaurar = false) => {
      menu.hidden = true;
      boton.setAttribute("aria-expanded", "false");
      if (restaurar) boton.focus({ preventScroll: true });
    };
    boton.addEventListener("click", () => {
      if (!menu.hidden) { cerrar(true); return; }
      menu.hidden = false;
      boton.setAttribute("aria-expanded", "true");
      opcion.focus({ preventScroll: true });
    });
    opcion.addEventListener("click", () => { cerrar(); navegar("mis-preferencias"); });
    documento.addEventListener("keydown", (evento) => {
      if (evento.key === "Escape" && !menu.hidden) { evento.preventDefault(); cerrar(true); }
    });
    documento.addEventListener("click", (evento) => {
      if (!menu.hidden && !menu.contains(evento.target) && !boton.contains(evento.target)) cerrar();
    });
    documento.addEventListener("focusin", (evento) => {
      if (!menu.hidden && !menu.contains(evento.target) && evento.target !== boton) cerrar();
    });
  }
  function aplicarInicio() {
    const inicio = superficie.leer()?.estado.valores.inicio;
    if (!inicio || inicio === "cuadro" || estado.vista !== "portal") return;
    if (inicio === "bolsas" && vistaBolsaDisponible()) {
      navegar("resumen", { enfocar: false });
      return;
    }
    if (inicio === "peticiones" && altaCTDisponible()) {
      navegar("contratacion-temporal", { subvista: "alta", enfocar: false });
      return;
    }
    const aviso = porId("aviso-inicio-preferido");
    aviso.textContent = traducir("preferencias_destino_no_disponible");
    aviso.hidden = false;
    anunciar(aviso.textContent);
  }
  function fijarIniciales(texto) { iniciales = String(texto ?? ""); avatar.fijarIniciales(iniciales); }
  return Object.freeze({ superficie, instalarMenu, aplicarInicio, alternarVisualVolatil, fijarIniciales });
}
