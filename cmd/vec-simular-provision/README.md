# Simular un concurso interno de provisión

El comando prepara una convocatoria borrador, comprueba la solicitud de un
empleado sintético y valora cada puesto solicitado con el motor común de
Provisión. Devuelve el desglose por orden de preferencia. Todo el resultado es
una simulación: no presenta solicitudes, registra expedientes, firma documentos
ni emite recibos duraderos.

```sh
go run ./cmd/vec-simular-provision -entrada cmd/vec-simular-provision/testdata/proceso.sintetico.json
# La misma entrada por stdin:
go run ./cmd/vec-simular-provision < cmd/vec-simular-provision/testdata/proceso.sintetico.json
```

Use exclusivamente datos sintéticos. El comando no consulta Personal ni RPT.
Las fechas, reglas y coeficientes del ejemplo proceden del ejemplo sintético del
motor ya existente; no representan bases aprobadas. No hay valores de baremo
predeterminados en el caso de uso.

## Entrada JSON

La raíz es `ports.PeticionProceso` y exige dos objetos: `proceso` y `solicitud`.
El [ejemplo completo](testdata/proceso.sintetico.json) contiene dos puestos y
preferencias en orden distinto del catálogo.

| Objeto | Campos obligatorios |
| --- | --- |
| `proceso` | `schema_version` = `provision.proceso.v1`, `referencia`, `version`, `estado` = `borrador`, `configuracion`, `puestos` |
| Cada `puesto` | `referencia`, `rpt_ref`, `rpt_version`, `nivel`, `requisitos` |
| Cada requisito del puesto | `referencia`, `version` |
| `solicitud` | `referencia`, `proceso_ref`, `proceso_version`, `empleado_ref`, `version_reglas`, `instantanea`, `preferencias`, `valoraciones` |
| `instantanea` | `referencia`, `empleado_ref`, `version`, `fuente_ref`, `condicion_interna` = `cumple` |
| Cada preferencia | `orden`, `puesto_ref` |
| Cada entrada de valoración | `puesto_ref`, `requisitos`, `entrada` |
| Cada comprobación de requisito | `requisito_ref`, `requisito_version`, `estado`, `motivo_codigo`, `fuente_ref` |

`configuracion` y `entrada` usan los tipos `domain.Configuracion` y
`domain.Entrada` del motor existente, definidos en
`internal/modules/provision/domain/concursos_contrato.go`. Los puntos se
transportan como cadenas de enteros en micropuntos; las fracciones, como
cadenas exactas `numerador/denominador`. Ningún canal calcula con `float64`.

Las referencias del proceso son tokens opacos de hasta 160 bytes, con letras
ASCII, dígitos y los signos `:`, `-`, `_`, `.`, `/`, `#`. El contrato no contiene
campos de nombre ni DNI.

La convocatoria de la configuración debe coincidir con la referencia del
proceso. La solicitud debe conservar exactamente las versiones del proceso y
de las reglas. La instantánea debe pertenecer a la misma referencia de empleado.
Cada entrada conserva la misma instantánea, referencia de puesto y nivel que el
puesto ofertado. Las referencias y versiones de RPT se declaran en el ejercicio;
no acreditan existencia, publicación ni vacancia oficial.

Las preferencias forman una lista ordenada desde 1, sin saltos ni repeticiones,
y sólo pueden contener puestos ofertados. Se pueden seleccionar menos puestos
que los ofertados. `valoraciones` debe contener exactamente una entrada por
preferencia; no acepta puestos adicionales. El proceso admite hasta 100 puestos
y 100 requisitos por puesto.

Cada requisito exige su referencia y versión exactas, un motivo de catálogo y
procedencia. Su estado es `cumple`, `no_cumple` o `pendiente`. Deben estar presentes
todos los requisitos del puesto; una ausencia de información se expresa como
`pendiente`, con su motivo y fuente. Las listas de requisitos vacías son `[]`,
nunca `null`.

La condición interna sólo es un dato declarado del ejercicio. Las entradas que
la marcan `no_cumple` o `pendiente` no preparan una solicitud de empleado interno.
Marcarla `cumple` no acredita empleo, identidad ni autorización institucional.

El lector reutiliza `simulacion.Decodificar`: limita el documento a 2 MiB y
rechaza campos desconocidos, claves repetidas, cambios de mayúsculas, omisiones
de campos obligatorios, escalares nulos, tipos incorrectos, profundidad excesiva
y documentos concatenados. Las formas numéricas las validan los tipos exactos
del motor. Los archivos deben ser regulares; no se leen directorios ni tuberías
con `-entrada`. Stdin sigue disponible.

## Resultado

Stdout contiene `domain.ResultadoProceso`:

- `schema_version`: `provision.proceso.v1`.
- `alcance`: `simulacion`; `estado`: `borrador`.
- `proceso` y `solicitud`: definición e instantánea suministradas.
- `valoraciones`: puesto, orden de preferencia, `requisitos_estado`,
  comprobaciones y `resultado` del motor común.
- `huella_simulacion`: SHA256 del resultado JSON tipado completo, con ese propio
  campo vacío al calcularlo. Vincula RPT, reglas, instantánea, requisitos y
  preferencias. No es una firma ni un recibo.

La huella no depende del orden de las claves de un objeto JSON; conserva el
orden de las colecciones de entrada. El motor mantiene además sus huellas de
reglas, entrada y resultado por puesto. Repetir la misma entrada reproduce el
resultado y sus huellas; cambiar preferencias o versiones de RPT cambia la
huella global.

Un origen de méritos no disponible deja el desglose `pendiente_dato` y el total
`null`. No lo representa como cero calculado. `disponibles` distingue esa
situación de una fuente consultada sin méritos computables, que puede dar cero.
Un requisito incumplido se muestra separado de los puntos: ni una puntuación
positiva implica admisión ni la simulación dicta una exclusión administrativa.

Si falla la entrada o el cálculo, no se devuelve un resultado parcial. El comando
termina con código 1 y escribe en stderr `{"error":{"codigo":"…","campo":"…"}}`.
Los errores usan claves de catálogo y no incluyen el contenido de la entrada ni
la ruta del archivo. La ejecución correcta termina con código 0.

## Puertos e integración pendiente

`application.PrepararProceso` valida la preparación y
`application.SimularProceso` es el consumidor del motor por cada preferencia.
`simulacion.EjemploProceso` devuelve una copia del ejemplo sintético para el
servidor local. En HTTP, el servidor debe fijar hechos y condición interna; el
cliente puede seleccionar las preferencias y reglas admitidas para el ensayo.
Si selecciona un subconjunto de puestos, el servidor conserva únicamente las
entradas sintéticas correspondientes antes de invocar el caso de uso.

`ports/proceso.go` reserva interfaces neutrales para consultar la condición de
empleado en Personal, obtener una versión exacta del puesto de RPT, conectar la
autorización central y conservar borradores. No hay implementaciones de estos
puertos en el simulador. Una caída de RPT deberá devolver error; no permitirá
presentar un puesto vacío como puesto oficial.

La escritura futura lleva concesión central, versión esperada, clave de
idempotencia y correlación. Su adaptador debe revalidar y consumir la concesión
en la misma transacción que estado, versión, historia, auditoría y outbox. Este
corte no incluye SQL, conexiones institucionales, publicación, firma, admisión,
adjudicación ni toma de posesión. Tampoco afirma persistencia ni recuperación
tras reinicio.
