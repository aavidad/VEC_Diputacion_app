# Preparar el registro de notas de un ejercicio

Esta herramienta prepara una revisión completa de las notas de **una fase de prueba**. Conserva la convocatoria, la versión y huella de las bases, las reglas de fases aportadas, el ejercicio, la revisión, las notas y la referencia de cada corrección. Ordena las notas por referencia y calcula la huella del material para poder cotejarlo después. No escribe en la base de datos.

Desde la raíz del repositorio:

```sh
GOCACHE="$HOME/.cache/go-build" go run -p 6 ./cmd/vec-seleccion-calificaciones \
  < cmd/vec-seleccion-calificaciones/testdata/material.json
```

El ejemplo usa referencias y huellas ficticias. La salida es JSON por stdout; los errores salen por stderr con un código estable y código de proceso 1. Una nota sin puntos queda `pendiente`. Una nota con puntos exige `fuente_ref` y queda `aportada_para_revision`. Los puntos son enteros en micropuntos entre cero y el máximo de la fase. No se aceptan nombres ni documentos.

`configuracion` conserva mínimos, máximos, pesos y desempates de la versión aportada. Su contenido forma parte de la huella del material. La herramienta comprueba que `fase_ref` sea una fase de prueba de esa configuración y que las notas no se repitan ni excedan su máximo. El cálculo de mínimos, ponderaciones y empates corresponde al evaluador compartido de Selección, después de cotejar las fuentes y completar las fases. Esta preparación no calcula aprobados.

La primera revisión lleva `revision: 1` sin antecedente. Las siguientes requieren `antecedente_sha256`; el registro institucional tendrá que comprobar que esa huella pertenece a la revisión anterior vigente. La huella de salida identifica el material normalizado, no una firma o un asiento. Las referencias de solicitud, ejercicio, fuente, convocatoria y bases son aportadas por quien ejecuta la herramienta; aquí no se consultan sus registros.

Para usar este contrato en el circuito institucional faltan el cotejo de las bases y de su versión, la admisión y el anonimato que corresponda, la fuente de las correcciones y el acta del tribunal. También faltan autoría autorizada, decisión competente, reclamaciones y publicación. Una revisión debe conservar y enlazar la anterior; no debe reemplazarla. La salida declara `aprobada: false` y `publicada: false`.

La entrada está limitada a 1 MiB, 128 notas y 16 fases configuradas; se rechazan claves JSON repetidas y campos desconocidos. Estos límites son técnicos. El proceso no abre rutas ni servicios. Para verificar el corte:

```sh
GOCACHE="$HOME/.cache/go-build" go test -p 6 ./internal/modules/seleccion/domain ./internal/modules/seleccion/application ./cmd/vec-seleccion-calificaciones
```
