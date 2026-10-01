# Simular baremos de Bolsa y Concursos

`vec-baremador` calcula experiencia o méritos a partir de una versión
exacta de reglas y una entrada sintética. El modo predeterminado sigue siendo
`experiencia`; añada `--modo meritos` para formación, titulaciones y otros
méritos configurados por Bolsa. Exige los JSON canónicos y las
huellas SHA256 de ambos archivos. Usa el caso de uso de Bolsa y su motor de
aritmética exacta; cada punto equivale a un millón de micropuntos.

La salida tiene el alcance `simulacion`. Permite reproducir el cálculo con
borradores, pero no aprueba reglas, registra una solicitud ni acredita una
puntuación oficial. Este corte no guarda auditoría institucional ni conecta
con Personal, PostgreSQL o el portal. Use solo datos sintéticos.

## Comparar dos versiones de reglas de méritos

La comparación muestra los coeficientes, topes, fecha de corte, políticas y
referencias que han cambiado. Identifica las reglas y secciones por su clave;
una clave renombrada aparece como baja y alta, sin presumir equivalencia.
No requiere una entrada de méritos ni calcula puntos.

```sh
GOCACHE=/dev/shm/go-build go run -p 8 ./cmd/vec-baremador \
  --modo meritos --comparar-reglas \
  --reglas cmd/vec-baremador/testdata/comparacion_meritos_v1.json \
  --reglas-sha256 c20c7a88aa153e4970240a3841226ff7a0274fa297a2e9f27b769c4cab1f70be \
  --reglas-nuevas cmd/vec-baremador/testdata/comparacion_meritos_v2.json \
  --reglas-nuevas-sha256 b72b1a4e7d1e5fbfb7177fd38d6e73c7040fa33bc3578cf1740425f034cc8743
```

Este ejemplo sintético devuelve siete cambios: fecha de corte, tope total,
coeficiente y tope de la regla de cursos, tope de formación y las dos
referencias de definición correspondientes. Cada cambio conserva los valores
anterior y nuevo; la cabecera identifica ambas versiones y sus huellas.
Los puntos se expresan en micropuntos y las fracciones mantienen su forma exacta.

Ambas versiones deben pertenecer al mismo conjunto, convocatoria y expediente.
La versión nueva debe ser mayor, aunque no necesariamente consecutiva.
Comparar una versión consigo misma devuelve una lista vacía. La misma versión
con contenido distinto, las huellas contradictorias y el orden inverso se
rechazan. Este informe no publica reglas ni acredita aprobación, firma o
sucesión administrativa.

## Comparar reglas de experiencia

Use `--modo experiencia` con las mismas opciones de comparación. La salida
tiene el esquema `vec.bolsa.diferencia_reglas_experiencia.v1` y valores tipados:
puntos, fechas, referencias, criterios y políticas. Cada valor activa una sola
variante; el cero se conserva y una ausencia se representa como `null`.

```sh
GOCACHE=/dev/shm/go-build go run -p 8 ./cmd/vec-baremador \
  --modo experiencia --comparar-reglas \
  --reglas cmd/vec-baremador/testdata/comparacion_experiencia_v1.json \
  --reglas-sha256 8fdcca0bce51606a680364997d1d232de22d7ba5957cfc5bcf8a4538e5d9bc1d \
  --reglas-nuevas cmd/vec-baremador/testdata/comparacion_experiencia_v2.json \
  --reglas-nuevas-sha256 5ef04e7a8beac1563e85905da2ccebc9fe42e8a314bbf45fbbd246288ea99f09
```

El ejemplo devuelve diez cambios sintéticos. Incluye coeficiente, topes,
fecha de corte, jornada con umbral `1/2`, conversión `365/12` y solape.
El comparador también conserva orden, prioridad, reparto del exceso, desempate,
restos y redondeo. Distingue un límite concreto de `sin_limite`.

Los criterios se muestran completos, con claves, valores y referencias de
catálogo. Su orden canónico evita diferencias por una reordenación equivalente;
todavía no hay detalle por criterio. Cambiar la clave de una sección, grupo o
regla produce baja y alta. Cada archivo de experiencia queda limitado a 4 MiB.
Se aplican las mismas restricciones de identidad, versiones y huellas de la
comparación de méritos. No se calcula puntuación ni se aprueba una regla.

## Probar los ejemplos

Desde la raíz del repositorio:

```sh
datos_baremo=internal/modules/bolsa/application/simulacionbaremo/testdata
GOCACHE=/dev/shm/go-build go run ./cmd/vec-baremador \
  --reglas "$datos_baremo/reglas_a.json" \
  --reglas-sha256 8fdcca0bce51606a680364997d1d232de22d7ba5957cfc5bcf8a4538e5d9bc1d \
  --entrada "$datos_baremo/entrada.json" \
  --entrada-sha256 82fb38d543d82c73c36bd47cda906aefe079b73d50a8115e22efb10de27158b8
```

Para la segunda convocatoria, use la misma entrada y cambie las reglas:

```sh
GOCACHE=/dev/shm/go-build go run ./cmd/vec-baremador \
  --reglas "$datos_baremo/reglas_b.json" \
  --reglas-sha256 225218047153f9cc5692586642421fdefcd064c338ce83b6ebe361e7b9064a7b \
  --entrada "$datos_baremo/entrada.json" \
  --entrada-sha256 82fb38d543d82c73c36bd47cda906aefe079b73d50a8115e22efb10de27158b8
```

La entrada representa 61 días, del 1 de enero al 2 de marzo de 2026, ambos
incluidos, con jornada de `1/2`. Las reglas y sus dependencias son sintéticas.
Los coeficientes siguientes sirven para probar la configuración; cada
convocatoria real deberá fijar los de sus bases.

| Configuración | A | B |
| --- | --- | --- |
| Versión del conjunto | 1 | 3 |
| Jornada | Proporcional | Íntegra desde el umbral `1/2` |
| Puntos por mes de 30 días | 0,1 | 0,2 |
| Tope de unidades | 12 meses | 12 meses |
| Tope de regla y sección | 1 punto | 0,25 puntos |
| Restos | Conservar exactos | Conservar exactos |
| Redondeo | Mitad hacia arriba por regla, a micropuntos | Igual |
| Resultado | 101667 micropuntos (0,101667 puntos) | 250000 micropuntos (0,250000 puntos) |

Las referencias opacas de convocatoria y conjunto son distintas. En B, el
bruto redondeado es 406667 micropuntos y el tope lo reduce a 250000.
Ambas configuraciones rechazan solapes y coincidencias entre reglas.

Para comprobar un bloqueo de negocio, cambie la entrada por
`entrada_bloqueada.json` y su huella por
`7106e263ef5796dd0d9d76b0ce872e7ca6ff8cb6e996ae292dd718a0d0377b4e`.
Esa entrada usa una versión incompatible del catálogo: devuelve
`resultado.estado: bloqueado`, con explicación y sin campo `total`.

## Contrato de salida y errores

La salida estándar contiene un único JSON canónico, sin salto de línea final.
Su esquema es `vec.bolsa.simulacion_experiencia.v1`. Incluye `alcance`,
`convocatoria_ref`, `huella_resultado_sha256` y `resultado`. La huella corresponde
a los bytes canónicos del objeto `resultado`, que conserva las referencias,
versiones y huellas exactas del conjunto, la entrada y el motor, junto con el
desglose de jornada, unidades, redondeos y topes.

El código de salida es 0 cuando se obtiene un resultado, completado o
bloqueado. Los consumidores deben comprobar `resultado.estado` antes de usar
un total. Un fallo de entrada o cálculo devuelve código 2, deja vacía la salida
estándar y escribe un diagnóstico JSON con `codigo` y, cuando procede, `fase`,
en la salida de errores. Si falla la escritura de salida, también devuelve 2,
pero puede haber escrito parte del JSON: descarte esa salida. Un cero válido
se representa como un resultado completado con
total `"0"`. Un fallo técnico o un bloqueo nunca se convierte en ese cero.

Los archivos deben ser regulares y contener entre 1 byte y 16 MiB. El parámetro
`--limite-bytes` permite reducir ese máximo. La CLI rechaza directorios y FIFO;
no espera a que un proceso escriba en una tubería. Trabaja con las rutas que
elige el operador y sus permisos locales. No imprime rutas ni contenido de
entrada en los diagnósticos. Cambiar el formato, añadir un salto de línea o
presentar una huella distinta impide la simulación.

## Comprobación focal

```sh
GOCACHE=/dev/shm/go-build go test -p 32 \
  ./internal/modules/bolsa/application/simulacionbaremo ./cmd/vec-baremador
```

Las pruebas comprueban los dos resultados, las versiones y convocatorias
distintas, la reproducción de bytes y huellas, el bloqueo sin total, el rechazo
de bytes no canónicos y huellas incorrectas, y los límites de lectura.

## Formación, titulaciones y otros méritos

Los ejemplos `meritos_reglas_a.json` y `meritos_reglas_b.json` usan la misma
entrada `meritos_entrada.json`. Sus huellas están en los archivos `.sha256`
correspondientes. Para recorrer ambos desde la raíz:

```sh
datos_baremo=internal/modules/bolsa/application/simulacionbaremo/testdata
for configuracion in a b; do
  GOCACHE=/dev/shm/go-build go run ./cmd/vec-baremador --modo meritos \
    --reglas "$datos_baremo/meritos_reglas_$configuracion.json" \
    --reglas-sha256 "$(cat "$datos_baremo/meritos_reglas_$configuracion.sha256")" \
    --entrada "$datos_baremo/meritos_entrada.json" \
    --entrada-sha256 "$(cat "$datos_baremo/meritos_entrada.sha256")"
done
```

A devuelve `4000000` micropuntos (4 puntos); B, `6000000` (6 puntos).
En A, 80 horas generan 1,6 puntos, el tope de regla los reduce a 1,5 y el de
sección a 1,2. La titulación adicional aporta 2 y otros méritos 0,9. La suma
es 4,1; el tope total deja 4. B cambia coeficientes, cantidad de títulos y
topes mediante una versión distinta de reglas.

Las dos configuraciones excluyen un título usado como requisito, un curso de
19 horas, otro obtenido después del corte y otro caducado. La fecha de corte
y la vigencia son inclusivas. Los coeficientes son inventados para el ejemplo;
no representan bases oficiales de Bolsa ni de Concursos.

Para comprobar la falta de una regla, sustituya la entrada y su huella por
`meritos_bloqueada.json` y `meritos_bloqueada.sha256`. La CLI devuelve código
0 y estado `bloqueado`, con la incidencia `regla_ausente` y sin total. Una
huella incorrecta devuelve código 2 sin resultado.

El contrato de reglas es `vec.bolsa.reglas_meritos.v1`; la entrada,
`vec.bolsa.entrada_meritos.v1`. Las cantidades son cadenas racionales, por
ejemplo `"3/2"`; los puntos son cadenas enteras de micropuntos. Cada regla
fija familia, clase y catálogo exacto, mínimo por mérito, máximo de unidades
acumuladas, máximo de elementos, coeficiente, redondeo y tope. Se admiten las
familias `formacion`/`hora`, `titulacion`/`titulo` y `otros`/`unidad`.
Un hecho de titulación representa exactamente un título.

V1 exige las políticas `duplicados: bloquear`, `momento_redondeo: por_regla`
y `seleccion_elementos: mayor_unidad`. Cuando se limita el número de méritos,
se eligen primero los de mayor cantidad; una igualdad se ordena por referencia
solo para reproducir el desglose. El motor suma cantidades exactas, aplica el
tope de unidades, redondea una vez por regla y aplica los topes de regla,
sección y total. No reparte el recorte de sección o total entre méritos.

Una evidencia o hecho repetido bloquea el cálculo. Una regla ausente, un
catálogo incompatible o datos pendientes también lo bloquean, sin publicar
puntos parciales. Un mérito rechazado o no aplicable queda excluido. Un
requisito no recibe puntos; este simulador no evalúa la admisión.

La salida `vec.bolsa.simulacion_meritos.v1` conserva alcance `simulacion`,
convocatoria y huella del resultado. Este incluye referencias, versiones y
huellas de reglas y entrada, la versión del motor, incidencias y desgloses de
méritos, reglas y secciones. El consumidor debe comprobar `estado` antes de
usar `total`.

Los adaptadores pueden llamar a `ServicioMeritos.SimularMeritos(Solicitud)`.
`DatosEjemploMeritos()` y `DatosEjemploExperiencia()` devuelven copias de los
JSON embebidos, sin leer archivos del operador. `calculomeritos.Conjunto`
valida y genera la representación canónica con `RepresentacionCanonica()`;
las restauraciones exigen bytes y huellas exactos. Para editar reglas de
experiencia, `reglasbaremo.CanonicalizarConjuntoReglasBaremoJSON()` acepta
espacios u orden de campos diferentes y reconstruye el contrato existente.
Rechaza claves desconocidas, duplicadas o con mayúsculas, datos sobrantes y
anidamiento superior a 32. No cambia la restauración histórica ni activa reglas.

Este modo no suma experiencia con otros méritos ni incluye grado, antigüedad
o permanencia de Provisión. No guarda reglas ni puntuaciones oficiales.

Para comprobar el corte:

```sh
GOCACHE=/dev/shm/go-build go test -race -p 32 \
  ./internal/modules/bolsa/domain/calculomeritos \
  ./internal/modules/bolsa/domain/reglasbaremo \
  ./internal/modules/bolsa/application/simulacionbaremo ./cmd/vec-baremador
```


## Concursos de provisión de puestos

El modo `concursos` valora grado personal, trabajo desarrollado por nivel,
antigüedad, permanencia, cursos y titulaciones. Las reglas pertenecen a
Provisión y reutilizan la aritmética exacta común. No suman puntuaciones de
Bolsa ni consultan su información.

Puede abrir el ejemplo sintético embebido sin preparar archivos:

```sh
GOCACHE=/dev/shm/go-build go run ./cmd/vec-baremador --modo concursos --listar-ejemplos
GOCACHE=/dev/shm/go-build go run ./cmd/vec-baremador --modo concursos \
  --ejemplo ejemplo:concursos:v1
```

El ejemplo se encuentra en
`internal/modules/provision/adapters/simulacion/ejemplos/concursos-v1.sintetico.json`.
Sus valores permiten ensayar reglas; no son bases aprobadas de Diputación.
La configuración fija convocatoria, versión, bases, fecha de corte, ventana,
coeficientes, tablas, topes, jornada, conversión y redondeo.

Para una configuración propia, entregue los objetos `configuracion` y
`entrada` en archivos separados con `--reglas`, `--entrada` y sus respectivas
huellas SHA256. Cada JSON de Concursos admite hasta 2 MiB; `--limite-bytes` puede
reducir el límite de lectura. Los archivos sólo pueden contener datos sintéticos. No combine
estos parámetros con `--ejemplo` o `--listar-ejemplos`.

Los intervalos incluyen su comienzo y excluyen su final. `fecha_corte` es el
primer día que queda fuera del cálculo. Una regla mensual con `por_periodo`
descarta los días restantes de cada hecho. V1 rechaza los métodos mensuales y
anuales con `por_tramo`; este último agrupa días exactos ponderados por jornada
antes de convertirlos. No presupone que un mes tenga treinta días.

El resultado `provision.simulacion.v1` conserva las versiones del motor y las
reglas, las huellas y el desglose de unidades, bruto y topes. Compruebe
`resultado.completo` y `resultado.total`: si falta una fuente necesaria,
el total queda sin calcular y aparece una incidencia. Un cero exige datos
conocidos. La simulación no admite, adjudica, firma ni publica una valoración.

CLI y web usan `application.Simular`. Este corte no guarda borradores ni
puntuaciones institucionales; la activación durable continúa pendiente de su
cadena de autorización y auditoría.

```sh
GOCACHE=/dev/shm/go-build go test -p 32 ./internal/modules/provision/... ./cmd/vec-baremador
```
