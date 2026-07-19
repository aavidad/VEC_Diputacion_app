const urlTesela = "http://127.0.0.1:8080/styles/osm-granada/256/8/125/99.png";

try {
  const respuesta = await fetch(urlTesela, { signal: AbortSignal.timeout(10_000) });
  const tipo = respuesta.headers.get("content-type") ?? "";
  const cuerpo = await respuesta.arrayBuffer();

  if (respuesta.status !== 200 || !tipo.startsWith("image/png") || cuerpo.byteLength < 100) {
    process.exitCode = 1;
  }
} catch {
  process.exitCode = 1;
}
