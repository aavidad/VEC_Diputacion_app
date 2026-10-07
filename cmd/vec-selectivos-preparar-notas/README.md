# Preparar una revisión de notas de ejemplo

Compara dos cálculos de Selección con el baremador común y la misma configuración.
La propuesta cambia sólo notas de pruebas sintéticas: conserva los requisitos,
los méritos y las demás notas del antecedente. Una nota pendiente sigue sin
calificación.

Desde la raíz del repositorio:

```sh
GOCACHE="$HOME/.cache/go-build" go run -p 8 ./cmd/vec-selectivos-preparar-notas \
  < cmd/vec-selectivos-preparar-notas/testdata/entrada.json > revision-notas.json
```

El ejemplo modifica una nota anterior y propone dejar otra pendiente. La entrada
JSON admite `ejemplo_ref`, `configuracion`, `notas_antecedente` y
`notas_propuestas`. Las dos listas son opcionales. Cada nota contiene
`solicitud_ref`, `fase_ref` y `puntos_micropuntos`: un entero entre cero y el máximo
de la prueba, o `null`. Las referencias pertenecen al catálogo de ejemplos de
Selección. No se admiten nombres ni hechos aportados por el operador.

Primero se calcula el antecedente a partir del ejemplo y sus cambios. Después se
aplica la propuesta sobre esas mismas notas, conservando los pares que no cambia,
y se calcula de nuevo. La salida conserva ambos resultados, las diferencias de
nota y las actuaciones pendientes: revisión competente, aprobación, firma y
publicación. No contiene una resolución de reclamación ni una lista oficial.

`huella_antecedente_sha256` es el SHA256 del JSON Go del resultado anterior
recalculado. No identifica los bytes de entrada, no coteja una salida anterior
independiente ni acredita su origen, firma o aprobación. Repetir la misma entrada
produce la misma preparación.

La herramienta lee hasta 64 KiB por la entrada estándar y escribe el JSON por la
salida estándar. Ante una entrada inválida devuelve un código nominal por stderr
y deja stdout vacío. No abre rutas de archivos, conexiones, perfiles ni servicios
institucionales.
