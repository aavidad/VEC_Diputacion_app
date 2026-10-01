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
    if (reglas.reglas_experiencia && (!regla.jornada || typeof regla.jornada !== "object" || Array.isArray(regla.jornada)
      || typeof regla.jornada.modo !== "string" || !/^[a-z_]+$/u.test(regla.jornada.modo))) throw new Error("archivo_invalido");
    if (Object.hasOwn(regla, "criterios") && (!Array.isArray(regla.criterios)
      || regla.criterios.some((criterio) => !criterio || !Array.isArray(criterio.valores)
        || criterio.valores.some((valor) => typeof valor !== "string")))) throw new Error("archivo_invalido");
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
/** Normaliza una fracción escrita; no aplica ninguna política ni calcula puntos. */
export function normalizarFraccionJornada(valor) {
  if (typeof valor !== "string" || !/^[1-9][0-9]{0,18}\/[1-9][0-9]{0,18}$/u.test(valor)) throw new Error("umbral_invalido");
  const [numerador, denominador] = valor.split("/").map(BigInt);
  if (numerador > denominador) throw new Error("umbral_invalido");
  let a = numerador, b = denominador;
  while (b) [a, b] = [b, a % b];
  const n = numerador / a, d = denominador / a;
  // Límite de los componentes canónicos de baremacion.Racional V1.
  if (n > 1000000000n || d > 1000000000n) throw new Error("umbral_invalido");
  return `${n}/${d}`;
}
/** Catálogo de la herramienta local; no es aprobación de las bases. */
export function comprobarCatalogoJornada(datos) {
  if (!datos || datos.esquema !== "vec.bolsa.catalogo_jornada.v1" || datos.version !== 1
    || datos.contrato_reglas !== "vec.bolsa.conjunto_reglas_baremo.v1" || datos.motor !== "vec.bolsa.motor_experiencia.v1"
    || !Array.isArray(datos.opciones) || datos.opciones.length !== 5) throw new Error("catalogo_jornada_no_disponible");
  const modos = new Set();
  for (const opcion of datos.opciones) {
    if (!opcion || typeof opcion.modo !== "string" || !/^[a-z_]+$/u.test(opcion.modo)
      || modos.has(opcion.modo) || opcion.etiqueta !== `jornada_${opcion.modo}`
      || typeof opcion.requiere_umbral !== "boolean" || typeof opcion.disponible !== "boolean"
      || !opcion.disponible && !/^jornada_[a-z_]+$/u.test(opcion.motivo ?? "")) throw new Error("catalogo_jornada_no_disponible");
    modos.add(opcion.modo);
  }
  if (datos.opciones.filter((o) => o.disponible).length !== 4 || datos.opciones.filter((o) => o.requiere_umbral).length !== 1) throw new Error("catalogo_jornada_no_disponible");
  return copia(datos);
}
export function crearEditorBaremo({ cliente, catalogoJornada = null, alCambiar = () => {} }) {
  let solicitud = null;
  let generacion = 0;
  let catalogo = null;
  try { catalogo = comprobarCatalogoJornada(catalogoJornada); } catch { /* Edición de jornada cerrada si no hay catálogo compatible. */ }
  const estado = { catalogoJornada: catalogo, ejemplo: null, original: null, borrador: null, cambiado: false, trabajando: false, comparacion: null, error: "", invalidos: {} };
  function invalidar() {
    generacion++; solicitud?.abort(); solicitud = null;
    estado.comparacion = null; estado.error = ""; estado.trabajando = false;
  }
  function cargar(ejemplo, reglas = ejemplo.reglas) {
    const cargadas = leerReglas(JSON.stringify(reglas));
    if (estado.catalogoJornada) for (const regla of cargadas.reglas_experiencia ?? []) {
      const opcion = estado.catalogoJornada.opciones.find((o) => o.modo === regla.jornada.modo);
      if (!opcion?.disponible) continue;
      try {
        if (opcion.requiere_umbral ? normalizarFraccionJornada(regla.jornada.umbral) !== regla.jornada.umbral : Object.hasOwn(regla.jornada, "umbral")) throw new Error();
      } catch { throw new Error("archivo_invalido"); }
    }
    const caso = copia(ejemplo); const original = copia(ejemplo.reglas);
    const cambiado = JSON.stringify(cargadas) !== JSON.stringify(original);
    invalidar();
    estado.ejemplo = caso; estado.original = original; estado.borrador = cargadas;
    estado.invalidos = {};
    estado.cambiado = cambiado; alCambiar();
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
  function reglaJornada(indice) {
    if (!Number.isInteger(indice) || indice < 0 || !estado.borrador?.reglas_experiencia?.[indice]) throw new Error("campo_invalido");
    return estado.borrador.reglas_experiencia[indice];
  }
  function editarPoliticaJornada(indice, modo) {
    const opcion = estado.catalogoJornada?.opciones.find((o) => o.modo === modo);
    if (!opcion?.disponible) throw new Error("jornada_no_disponible");
    const regla = reglaJornada(indice);
    if (regla.jornada?.modo === modo) return;
    const ruta = ["reglas_experiencia", indice, "jornada"], umbral = [...ruta, "umbral"];
    editar(ruta, opcion.requiere_umbral ? { modo, umbral: "" } : { modo });
    delete estado.invalidos[JSON.stringify(umbral)];
    if (opcion.requiere_umbral) estado.invalidos[JSON.stringify(umbral)] = "";
    alCambiar();
  }
  function editarUmbralJornada(indice, valor) {
    const regla = reglaJornada(indice);
    const opcion = estado.catalogoJornada?.opciones.find((o) => o.modo === regla.jornada?.modo);
    if (!opcion?.disponible || !opcion.requiere_umbral) throw new Error("campo_invalido");
    const ruta = ["reglas_experiencia", indice, "jornada", "umbral"];
    try { editar(ruta, normalizarFraccionJornada(valor)); }
    catch (error) { invalidar(); estado.cambiado = true; estado.invalidos[JSON.stringify(ruta)] = valor; throw error; }
  }
  async function comparar() {
    if (!estado.ejemplo || estado.trabajando || Object.keys(estado.invalidos).length) return;
    if (estado.catalogoJornada && (estado.borrador.reglas_experiencia ?? []).some((r) => !estado.catalogoJornada.opciones.find((o) => o.modo === r.jornada?.modo)?.disponible)) {
      invalidar(); estado.error = "jornada_no_disponible"; alCambiar(); return;
    }
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
  return Object.freeze({ cargar, editar, editarPoliticaJornada, editarUmbralJornada, comparar, invalidar,
    cancelarSimulacion() {
      // Cambiar de panel no equivale a descartar el borrador ni su último error.
      generacion++; solicitud?.abort(); solicitud = null; estado.trabajando = false;
    },
    registrarInvalido(ruta, valor) { invalidar(); estado.cambiado = true; estado.invalidos[JSON.stringify(ruta)] = valor; }, estado: () => copia(estado),
    exportar: () => { if (Object.keys(estado.invalidos).length) throw new Error("campo_invalido"); return JSON.stringify(estado.borrador); },
    desmontar() { invalidar(); estado.ejemplo = null; } });
}
