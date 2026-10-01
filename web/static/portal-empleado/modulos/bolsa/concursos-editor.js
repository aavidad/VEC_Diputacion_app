/** Estado del borrador de Provisión. El navegador no calcula puntuaciones. */
const copia = (v) => structuredClone(v);
export const MAXIMO_CONFIGURACION = 256 * 1024;
const fallo = (codigo) => Object.assign(new Error(codigo), { codigo });

export function leerConfiguracion(texto) {
  if (typeof texto !== "string" || new TextEncoder().encode(texto).length > MAXIMO_CONFIGURACION) throw fallo("archivo_invalido");
  let datos;
  try { datos = JSON.parse(texto); } catch { throw fallo("archivo_invalido"); }
  if (!datos || datos.schema_version !== "provision.v1" || !Array.isArray(datos.reglas) || !datos.reglas.length || datos.reglas.length > 100) throw fallo("archivo_invalido");
  return datos;
}

export function crearEditorConcursos({ cliente, alCambiar = () => {} }) {
  let generacion = 0, peticion = null;
  const estado = { ejemplo: null, original: null, borrador: null, cambiado: false, trabajando: false, comparacion: null, error: "", invalidos: {} };
  function cancelar({ resultado = false } = {}) {
    generacion++; peticion?.abort(); peticion = null; estado.trabajando = false;
    if (resultado) { estado.comparacion = null; estado.error = ""; }
  }
  function cargar(ejemplo) {
    cancelar({ resultado: true });
    const validado = leerConfiguracion(JSON.stringify(ejemplo.configuracion));
    estado.ejemplo = copia(ejemplo); estado.original = copia(validado); estado.borrador = validado;
    estado.invalidos = {}; estado.cambiado = false; alCambiar();
  }
  function editar(ruta, valor, { invalido = false } = {}) {
    if (!estado.borrador || !Array.isArray(ruta) || !ruta.length || ruta.some((p) => ["__proto__", "prototype", "constructor"].includes(String(p)))) throw fallo("campo_invalido");
    let destino = estado.borrador;
    for (const parte of ruta.slice(0, -1)) {
      if (!destino || !Object.hasOwn(destino, parte)) throw fallo("campo_invalido");
      destino = destino[parte];
    }
    if (!destino || !Object.hasOwn(destino, ruta.at(-1))) throw fallo("campo_invalido");
    cancelar({ resultado: true }); estado.cambiado = true;
    if (invalido) estado.invalidos[JSON.stringify(ruta)] = valor;
    else { delete estado.invalidos[JSON.stringify(ruta)]; destino[ruta.at(-1)] = valor; }
  }
  async function comparar() {
    if (!estado.ejemplo || estado.trabajando || Object.keys(estado.invalidos).length) return;
    cancelar({ resultado: true }); const turno = generacion;
    peticion = new AbortController(); const signal = peticion.signal;
    estado.trabajando = true; alCambiar();
    try {
      const solicitud = { ejemplo_ref: estado.ejemplo.referencia };
      const antes = await cliente.simular({ ...solicitud, configuracion: copia(estado.original) }, { signal });
      if (turno !== generacion) return;
      const despues = await cliente.simular({ ...solicitud, configuracion: copia(estado.borrador) }, { signal });
      if (turno !== generacion) return;
      estado.comparacion = { antes, despues };
    } catch (error) {
      if (turno !== generacion || error.name === "AbortError") return;
      estado.error = error.codigo ?? "simulacion_fallida";
    } finally {
      if (turno === generacion) { peticion = null; estado.trabajando = false; alCambiar(); }
    }
  }
  // Primero valida en Go; un archivo rechazado nunca sustituye el borrador.
  async function importar(texto) {
    const configuracion = leerConfiguracion(texto);
    if (!estado.ejemplo) throw fallo("archivo_invalido");
    cancelar(); const turno = generacion; peticion = new AbortController();
    const signal = peticion.signal; estado.trabajando = true; alCambiar();
    try {
      await cliente.simular({ ejemplo_ref: estado.ejemplo.referencia, configuracion }, { signal });
      if (turno !== generacion) return false;
      estado.borrador = copia(configuracion); estado.invalidos = {}; estado.comparacion = null; estado.error = "";
      estado.cambiado = JSON.stringify(estado.borrador) !== JSON.stringify(estado.original);
      return true;
    } catch (error) {
      if (turno !== generacion || error.name === "AbortError") return false;
      throw fallo("archivo_invalido");
    } finally {
      if (turno === generacion) { peticion = null; estado.trabajando = false; alCambiar(); }
    }
  }
  return Object.freeze({ cargar, editar, comparar, importar, cancelar,
    estado: () => copia(estado),
    exportar() { if (!estado.borrador || Object.keys(estado.invalidos).length) throw fallo("campo_invalido"); return JSON.stringify(estado.borrador, null, 2); },
    desmontar() { cancelar({ resultado: true }); estado.ejemplo = null; },
  });
}
