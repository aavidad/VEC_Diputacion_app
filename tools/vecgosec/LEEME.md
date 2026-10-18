# GoSec local con paquetes completos

Tarea F0-29. Este ejecutable usa el adaptador oficial `goanalysis` de GoSec
2.25.0. Conserva sus reglas y exige construir la representación SSA antes de
analizar. No modifica GoSec ni carga proveedores de correcciones remotas.

El CLI instalado pasaba a su cargador los archivos del paquete por separado.
En el montaje de VEC eso producía 37 errores de tipos y metadatos, aunque el
paquete compilaba, y dejaba sin ejecutar los analizadores SSA. La carga por
nombre de paquete y el adaptador oficial completaron el mismo paquete sin
esos errores. La preparación y las credenciales de la aplicación no intervienen.

Desde la raíz de la fuente que se va a revisar:

```sh
python3 scripts/gosec_local.py ./internal/app/bootstrap
python3 scripts/gosec_local.py --informe "$VEC_GOSEC_INFORME" ./internal/app/bootstrap
```

Indique directorios de los paquetes cambiados. Se admiten varios y el sufijo
`/...`; un archivo `.go` suelto se rechaza. La herramienta selecciona el Go del
proyecto, compila el adaptador en una carpeta temporal y lo elimina al terminar.
Acepta el ejecutable de `PATH` si tiene la versión exacta declarada; si no,
usa la caché local existente. Así funciona también con `setup-go` en un
ejecutor limpio. Cada patrón debe corresponder al menos a un paquete.
El primer uso necesita descargar las dependencias fijadas en `go.mod` y `go.sum`.

El resultado JSON va a la salida estándar o a un archivo nuevo con permisos
privados. El estado de salida es `0` sin hallazgos, `1` con hallazgos y `2` si
no pudo completar la carga o el análisis. El lanzador corrige expresamente el
estado cero que devuelve el modo JSON estándar incluso cuando hay diagnósticos.
Incluye los archivos generados; los archivos de pruebas quedan fuera, como en
el uso predeterminado del CLI anterior. No filtra reglas ni descarta avisos
heredados: la revisión debe distinguirlos de las líneas del cambio.

`bash scripts/probar_gosec_local.sh` comprueba una fuente correcta, la detección
de TLS sin verificar incluso en un archivo generado, el rechazo de un error
de tipos, de archivos sueltos y de patrones sin paquetes. Comprueba también
la selección del Go exacto en `PATH` con caché de módulos vacía. No envía
peticiones de red desde las fuentes de control.

Fuente: [adaptador oficial de GoSec](https://github.com/securego/gosec/tree/v2.25.0/goanalysis).
