# Simular la experiencia de una bolsa

`vec-baremador` calcula un desglose de experiencia a partir de una versión
exacta de reglas y una entrada de servicios. Exige los JSON canónicos y las
huellas SHA256 de ambos archivos. Usa el caso de uso de Bolsa y su motor de
aritmética exacta; cada punto equivale a un millón de micropuntos.

La salida tiene el alcance `simulacion`. Permite reproducir el cálculo con
borradores, pero no aprueba reglas, registra una solicitud ni acredita una
puntuación oficial. Este corte no guarda auditoría institucional ni conecta
con Personal, PostgreSQL o el portal. Use solo datos sintéticos.

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
