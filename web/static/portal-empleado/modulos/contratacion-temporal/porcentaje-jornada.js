export function formatearPorcentajeJornada(valor, locale = "es-ES") {
  if (typeof valor === "number") {
    if (!Number.isInteger(valor) || valor < 1 || valor > 10000) return "";
  } else if (typeof valor === "string") {
    if (!/^[1-9][0-9]*$/.test(valor)) return "";
    const n = Number(valor);
    if (!Number.isInteger(n) || n < 1 || n > 10000) return "";
  } else {
    return "";
  }
  const n = typeof valor === "number" ? valor : Number(valor);
  const fmt = new Intl.NumberFormat(locale, { style: "percent", maximumFractionDigits: 2 });
  return fmt.format(n / 10000);
}
