# Sembrado de ejemplo de Peticiones de personal temporal

`sembrar_ejemplo_ct.py` crea o completa los expedientes de `casos_ejemplo_ct.json` usando solo la
API de VEC con los certificados mTLS de RRHH e Intervención. Exige la CA local, comprueba el
nombre del servidor y compara la huella SHA256 del certificado en cada conexión. No escribe
en tablas.
Es idempotente: claves derivadas de `espacio_claves` y del código de cada caso, fechas fijas, y
reanudación a partir de los hitos del expediente.

Alberto lo lanza desde el kit del hito 6. El kit verifica el nombre y el ID del contenedor,
obtiene del fichero preparado por Alberto la huella correspondiente a `clon` o `principal`,
introduce el fichero de casos en el contenedor y ejecuta el guion allí. La API se consulta en
`https://localhost` dentro de ese contenedor; el puerto interno lo pasa el kit. El guion no
admite una URL. El material mTLS se lee de `/vec-material`.

El kit pasa `--destino=clon|principal`, `--puerto-interno`, `--huella-servidor-sha256` y
`--casos` con la ruta del fichero dentro del contenedor. Sin `--ejecutar`, muestra el plan
sin escribir. Con `--ejecutar`, exige una terminal y pide teclear exactamente el destino.
Antes de escribir comprueba que todos los centros, categorías, contactos, grupos y modalidades
coinciden con los catálogos. Para `principal`, el kit exige antes `CLON-OK` del mismo paquete.

La opción `--destino` y la huella no identifican por sí solas el contenedor ni la base: esa
comprobación pertenece al kit. No ejecute el guion directamente ni mediante un túnel.

Límites del circuito actual: el análisis no mueve el expediente a «Análisis RRHH»; el llamamiento
no lo mueve a «Obtención del candidato»; la aceptación en Bolsa solo admite la categoría de
desarrollo C2, ausente del catálogo RPT, así que nombramiento, incorporación y seguimiento no se
alcanzan. Los plazos se cuentan desde la entrada en la fase: lo sembrado hoy queda en plazo.
