import { crearClienteRecuperacionFirmasV2 } from "../cliente-http-recuperacion-firmas-v2.js?v=20261004-r5-recuperacion-v1";
import { montarVistaRecuperacionFirmasV2 } from "../vista-recuperacion-firmas-v2.js?v=20261004-r5-recuperacion-v1";
import { crearTraductorRecuperacionFirmasV2 } from "../i18n-recuperacion-firmas-v2.js?v=20261004-r5-recuperacion-v1";
import { cargarTextos } from "../../../../comun/textos.js";
import { ZONA_HORARIA_PORTAL } from "../../../portal-i18n.js";

const solicitud = Object.freeze(await (await fetch(
  "./recuperacion-firmas-v2-selector-go.json?v=20261004-r5-recuperacion-v1", {
    cache: "no-store", credentials: "omit", redirect: "error",
  },
)).json());

let modo = "normal";
const dato = (await (await fetch("./recuperacion-firmas-v2-go.json?v=20261004-r5-recuperacion-v1", {
  cache: "no-store", credentials: "omit", redirect: "error",
})).json()).data;
const cliente = crearClienteRecuperacionFirmasV2({
  ejecutar: async () => {
    if (modo === "denegado") throw Object.assign(new Error("denegado"), { estado: 401 });
    if (modo === "espera") await new Promise((resolver) => { setTimeout(resolver, 250); });
    return structuredClone(dato);
  },
  validarOpciones: (opciones) => opciones ?? {},
});
const textos = await cargarTextos("contratacion-temporal-recuperacion-firmas-v2");
document.documentElement.lang = textos.idioma;
const mensajes = textos.seccion("general");
const t = crearTraductorRecuperacionFirmasV2(mensajes);
document.title = t("demo_titulo");
document.getElementById("aviso-demo").textContent = t("demo_aviso");
const vista = montarVistaRecuperacionFirmasV2({
  raiz: document.getElementById("recuperacion-demo"),
  cliente, obtenerSolicitud: () => solicitud, locale: textos.localizacion,
  zonaHoraria: ZONA_HORARIA_PORTAL, mensajes,
});
window.demoRecuperacion = Object.freeze({
  vista,
  modo(nuevo) { modo = nuevo; },
  solicitud,
});
