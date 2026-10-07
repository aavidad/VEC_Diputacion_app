# Inspección del binario antes de preparar una copia

Esta CLI lee un binario Go local y emite la parte observada del descriptor: SHA256,
tamaño, versión Go, plataforma y metadatos Git incrustados. No ejecuta el binario.

```sh
GOCACHE=/dev/shm/go-build go run -p 8 ./cmd/vec-copias-inspeccionar-binario \
  -raiz /directorio/local/autorizado \
  -binario bin/vec-server \
  -max-bytes 134217728
```

El límite del ejemplo es 128 MiB. Sistemas debe elegirlo explícitamente para el
binario que inspecciona; no hay un valor operativo por defecto. El techo técnico
del lector es 256 MiB para limitar el buffer en memoria. La ruta del binario debe
ser relativa a la raíz autorizada. Solo se leen archivos regulares.

SHA256 y metadatos se obtienen de los mismos bytes capturados. La salida omite rutas,
dependencias, parámetros del enlazador y demás configuración de compilación. Conserva
únicamente `GoVersion`, `GOOS`, `GOARCH`, `vcs`, `vcs.revision`, `vcs.time` y
`vcs.modified`. Los campos ajenos se ignoran para admitir versiones futuras de Go.

Una revisión Git ausente, un campo relevante repetido o inválido, fuentes modificadas,
un archivo truncado o un límite superado producen `no_comprobable`. Las razones indican
la clave y los valores comparados; para resolver metadatos incompletos debe obtenerse
un descriptor autenticado o una release aprobada.

`datos_observados` significa que se pudieron leer los bytes y esos campos. El commit
sigue siendo `commit_declarado_en_binario`; la correspondencia con una release requiere
la autoridad externa pendiente de Sistemas. `autenticidad` siempre es `no_comprobada`;
`habilita_copia` y `habilita_restauracion` siempre son `false`. No consulta PostgreSQL
ni verifica el resto del conjunto instalado. Una copia deberá superar su restauración
aislada por separado.

La salida es JSON: código 0 para datos completos, 1 para `no_comprobable`, 2 para
argumentos incorrectos y 4 si no se puede escribir la salida. Las pruebas construyen binarios Go reales
con un repositorio Git sintético temporal; no ejecutan esos binarios.
