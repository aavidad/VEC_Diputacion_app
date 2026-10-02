import { cargarTextos } from "../../../comun/textos.js";

const TEXTOS = await cargarTextos("meritos-consulta");

/** Usa el traductor, idioma, números y fechas comunes, sin diccionario local. */
export function crearTextosConsultaMerito(textos = TEXTOS) {
  if (typeof textos?.traducir !== "function" || typeof textos?.fecha !== "function" || typeof textos?.numero !== "function" || !textos.idioma) throw new TypeError("meritos.consulta.textos_invalidos");
  return Object.freeze({
    idioma: textos.idioma,
    t: (clave, variables) => textos.traducir(`consulta.${clave}`, variables),
    numero: (valor) => textos.numero(valor),
    fecha: (valor) => textos.fecha(`${valor}T12:00:00Z`, { dateStyle: "medium", timeZone: "Europe/Madrid" }),
    instante: (valor) => textos.fecha(valor, { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }),
  });
}
