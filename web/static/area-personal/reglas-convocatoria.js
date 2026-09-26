/**
 * Lectura de las reglas publicadas de una convocatoria (Selección): si el
 * servidor la declara abierta, turnos, requisitos y baremo de méritos. El
 * navegador no decide plazos ni requisitos; solo muestra lo publicado. El
 * autobaremo que se calcula aquí es orientativo mientras se escribe: la
 * puntuación que cuenta la devuelve el servidor al guardar y al presentar.
 */

const PATRON_CLAVE = /^[A-Za-z0-9][A-Za-z0-9._:-]{0,79}$/u;
const PATRON_DECIMAL = /^\d{1,9}(?:\.\d{1,6})?$/u;
const texto = (valor, maximo = 300) => typeof valor === "string" ? valor.trim().slice(0, maximo) : "";
/** Las cifras del contrato llegan como cadena decimal; ausente → sin máximo. */
const decimal = (valor) => typeof valor === "string" && PATRON_DECIMAL.test(valor) ? Number(valor) : null;

function leerTurnos(turnos) {
  if (!Array.isArray(turnos)) return [];
  return turnos.filter((turno) => PATRON_CLAVE.test(turno?.clave || "") && texto(turno?.etiqueta))
    .slice(0, 20).map((turno) => Object.freeze({ clave: turno.clave, etiqueta: texto(turno.etiqueta, 120) }));
}

function leerBaremo(baremo) {
  if (!baremo || typeof baremo !== "object" || !Array.isArray(baremo.grupos)) return null;
  const grupos = baremo.grupos.filter((grupo) => PATRON_CLAVE.test(grupo?.clave || "") && texto(grupo?.titulo) && Array.isArray(grupo.meritos))
    .slice(0, 30).map((grupo) => Object.freeze({
      clave: grupo.clave, titulo: texto(grupo.titulo), maximo: decimal(grupo.maximo),
      meritos: Object.freeze(grupo.meritos
        .filter((merito) => PATRON_CLAVE.test(merito?.clave || "") && texto(merito?.titulo) && decimal(merito?.puntos_por_unidad) !== null)
        .slice(0, 60).map((merito) => Object.freeze({
          clave: merito.clave, titulo: texto(merito.titulo), unidad: texto(merito.unidad, 60),
          puntos_por_unidad: decimal(merito.puntos_por_unidad), maximo: decimal(merito.maximo),
        }))),
    })).filter((grupo) => grupo.meritos.length > 0);
  return grupos.length ? Object.freeze({ maximo: decimal(baremo.maximo), grupos: Object.freeze(grupos) }) : null;
}

export function leerReglasConvocatoria(dato) {
  const requisitos = (Array.isArray(dato?.requisitos) ? dato.requisitos : [])
    .filter((requisito) => PATRON_CLAVE.test(requisito?.clave || "") && texto(requisito?.titulo))
    .slice(0, 60).map((requisito) => Object.freeze({
      clave: requisito.clave, titulo: texto(requisito.titulo), descripcion: texto(requisito.descripcion, 1200),
      obligatorio: requisito.obligatorio === true,
    }));
  return Object.freeze({
    convocatoriaRef: texto(dato?.convocatoria_ref, 160),
    abierta: dato?.abierta === true,
    titulo: texto(dato?.titulo),
    abreEn: texto(dato?.abre_en, 40),
    cierraEn: texto(dato?.cierra_en, 40),
    ejemplo: dato?.marca_ejemplo === true,
    requisitos: Object.freeze(requisitos),
    turnos: Object.freeze(leerTurnos(dato?.turnos)),
    baremo: leerBaremo(dato?.baremo),
  });
}

const redondear = (valor) => Math.round(valor * 1000) / 1000;

/**
 * Autobaremo orientativo con el baremo publicado: cantidad por puntos de
 * unidad, con los máximos publicados de cada mérito, grupo y total.
 * `cantidades` es un Map «grupo|mérito» → número.
 */
export function calcularAutobaremoOrientativo(baremo, cantidades) {
  const porMerito = new Map();
  let total = 0;
  for (const grupo of baremo?.grupos || []) {
    let sumaGrupo = 0;
    for (const merito of grupo.meritos) {
      const cantidad = Number(cantidades?.get?.(`${grupo.clave}|${merito.clave}`) || 0);
      const bruto = Number.isFinite(cantidad) && cantidad > 0 ? cantidad * merito.puntos_por_unidad : 0;
      const puntos = redondear(merito.maximo === null ? bruto : Math.min(bruto, merito.maximo));
      porMerito.set(`${grupo.clave}|${merito.clave}`, puntos);
      sumaGrupo += puntos;
    }
    total += grupo.maximo === null ? sumaGrupo : Math.min(sumaGrupo, grupo.maximo);
  }
  if (typeof baremo?.maximo === "number") total = Math.min(total, baremo.maximo);
  return Object.freeze({ porMerito, total: redondear(total) });
}
