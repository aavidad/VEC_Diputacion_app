/** Borrador local: no activa reglas ni conserva datos en el navegador. */
const copia = (valor) => structuredClone(valor);
export const MAXIMO_ARCHIVO = 2 * 1024 * 1024;
export function leerReglas(texto) {
  if (typeof texto !== "string" || new TextEncoder().encode(texto).length > MAXIMO_ARCHIVO) throw new Error("archivo_invalido");
  const reglas = JSON.parse(texto);
  if (!reglas || typeof reglas.fecha_corte_inclusiva !== "string" || !Array.isArray(reglas.secciones)
    || reglas.secciones.length > 100 || !["vec.bolsa.conjunto_reglas_baremo.v1", "vec.bolsa.reglas_meritos.v1"].includes(reglas.esquema)
    || !Array.isArray(reglas.reglas_experiencia ?? reglas.reglas) || (reglas.reglas_experiencia ?? reglas.reglas).length > 500) throw new Error("archivo_invalido");
  const puntos = (valor) => typeof valor === "string" && /^(0|[1-9][0-9]{0,18})$/u.test(valor);
  if (!/^\d{4}-\d{2}-\d{2}$/u.test(reglas.fecha_corte_inclusiva)
    || reglas.secciones.some((s) => !s || typeof s.clave !== "string" || !puntos(s.puntos_maximos ?? s.maximo_puntos))) throw new Error("archivo_invalido");
  for (const regla of reglas.reglas_experiencia ?? reglas.reglas) {
    if (!regla || typeof regla.seccion_clave !== "string" || !puntos(regla.puntos_por_unidad)
      || !["dia", "mes", "ano", "hora", "titulo", "unidad"].includes(regla.unidad_temporal?.unidad_puntuable ?? regla.unidad)
      || !(puntos(regla.maximo_puntos) || regla.maximo_puntos?.modo === "sin_limite" || regla.maximo_puntos?.modo === "limitado" && puntos(regla.maximo_puntos.valor))) throw new Error("archivo_invalido");
  }
  if (Object.hasOwn(reglas, "maximo_total") && !puntos(reglas.maximo_total)) throw new Error("archivo_invalido");
  return reglas;
}
/** Conversión de representación, no fórmula de baremación. */
export function aMicropuntos(valor) {
  if (!/^(0|[1-9][0-9]{0,12})(?:[.,][0-9]{1,6})?$/u.test(valor)) throw new Error("puntos_invalidos");
  const [entero, decimal = ""] = valor.replace(",", ".").split(".");
  return String(BigInt(entero) * 1000000n + BigInt(decimal.padEnd(6, "0")));
}
export function aDecimal(valor) {
  if (!/^(0|[1-9][0-9]*)$/u.test(valor)) throw new Error("puntos_invalidos");
  const puntos = BigInt(valor);
  const fraccion = String(puntos % 1000000n).padStart(6, "0").replace(/0+$/u, "");
  return `${puntos / 1000000n}${fraccion ? `.${fraccion}` : ""}`;
}
export function crearEditorBaremo({ cliente, alCambiar = () => {} }) {
  let solicitud = null;
  let generacion = 0;
  const estado = { ejemplo: null, original: null, borrador: null, cambiado: false, trabajando: false, comparacion: null, error: "", invalidos: {} };
  function invalidar() {
    generacion++; solicitud?.abort(); solicitud = null;
    estado.comparacion = null; estado.error = ""; estado.trabajando = false;
  }
  function cargar(ejemplo, reglas = ejemplo.reglas) {
    invalidar();
    const cargadas = leerReglas(JSON.stringify(reglas));
    estado.ejemplo = copia(ejemplo); estado.original = copia(ejemplo.reglas); estado.borrador = cargadas;
    estado.invalidos = {};
    estado.cambiado = JSON.stringify(cargadas) !== JSON.stringify(estado.original); alCambiar();
  }
  function editar(ruta, valor) {
    if (!estado.borrador) return;
    // La ruta solo procede de controles propios; nunca se aceptan claves heredadas.
    if (!Array.isArray(ruta) || ruta.some((parte) => ["__proto__", "constructor", "prototype"].includes(String(parte)))) throw new Error("campo_invalido");
    let destino = estado.borrador;
    for (const parte of ruta.slice(0, -1)) {
      if (!Object.hasOwn(destino, parte)) throw new Error("campo_invalido");
      destino = destino[parte];
    }
    const clave = ruta.at(-1);
    if (!Object.hasOwn(destino, clave)) throw new Error("campo_invalido");
    invalidar(); delete estado.invalidos[JSON.stringify(ruta)]; destino[clave] = valor; estado.cambiado = true;
  }
  async function comparar() {
    if (!estado.ejemplo || estado.trabajando || Object.keys(estado.invalidos).length) return;
    invalidar(); const turno = generacion; solicitud = new AbortController();
    const signal = solicitud.signal; estado.trabajando = true; alCambiar();
    const contexto = { modo: estado.ejemplo.modo, ejemplo_ref: estado.ejemplo.referencia };
    try {
      const antes = await cliente.simular({ ...contexto, reglas: copia(estado.original) }, { signal });
      if (turno !== generacion) return;
      const despues = await cliente.simular({ ...contexto, reglas: copia(estado.borrador) }, { signal });
      if (turno !== generacion) return;
      estado.comparacion = { antes, despues };
    } catch (error) {
      if (turno !== generacion || error.name === "AbortError") return;
      estado.error = error.codigo ?? "simulacion_fallida";
    } finally {
      if (turno === generacion) { estado.trabajando = false; solicitud = null; alCambiar(); }
    }
  }
  return Object.freeze({ cargar, editar, comparar, invalidar,
    registrarInvalido(ruta, valor) { invalidar(); estado.cambiado = true; estado.invalidos[JSON.stringify(ruta)] = valor; }, estado: () => copia(estado),
    exportar: () => { if (Object.keys(estado.invalidos).length) throw new Error("campo_invalido"); return JSON.stringify(estado.borrador); },
    desmontar() { invalidar(); estado.ejemplo = null; } });
}
