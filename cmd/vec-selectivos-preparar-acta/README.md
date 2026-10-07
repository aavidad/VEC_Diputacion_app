# Preparar un borrador para una sesión propuesta

La herramienta produce material JSON revisable con el orden del día y propuestas
de acuerdo. Conserva los textos aportados y muestra lo que falta. No afirma que
la sesión se haya celebrado ni que el tribunal haya adoptado acuerdos.

Desde la raíz del repositorio:

```sh
go run ./cmd/vec-selectivos-preparar-acta \
  -catalogos-dir web/static/textos -idioma es \
  < cmd/vec-selectivos-preparar-acta/testdata/material.json
```

Use `-idioma en` para obtener los avisos en inglés. Los textos de las propuestas
se conservan en el idioma aportado; no se traducen ni se redactan automáticamente.
El borrador sale por stdout; los errores, por stderr. Ambos usan JSON.
Una preparación válida devuelve código 0 y estado `borrador_propuesto`.
Una entrada inválida devuelve código 1 sin borrador por stdout.
Si no puede escribir el diagnóstico por stderr, devuelve código 2.
La herramienta no escribe archivos ni llama a servicios externos.

Para cotejar el borrador con una propuesta de tribunal concreta, guarde primero
la salida del preparador de tribunal y pásela al comando existente:

```sh
go run ./cmd/vec-selectivos-preparar-tribunal \
  -catalogos-dir web/static/textos -idioma es \
  < cmd/vec-selectivos-preparar-tribunal/testdata/material.json > /var/tmp/tribunal-sintetico.json
go run ./cmd/vec-selectivos-preparar-acta \
  -catalogos-dir web/static/textos -idioma es \
  -tribunal-salida /var/tmp/tribunal-sintetico.json \
  < cmd/vec-selectivos-preparar-acta/testdata/material-cotejo.json
```

El material de cotejo deja `huella_aportada_sha256` vacía. El comando calcula el SHA256 de los
bytes exactos del archivo de salida del tribunal, incluida su presentación JSON,
y lo incorpora al borrador. Si ya se aportó una huella, exige que coincida. También
coteja la identidad, la versión y la fase propuesta, y rechaza una salida de
tribunal alterada que no coincida con la preparación recalculada. `cotejo_local`
identifica esta comprobación. Cambiar espacios o mensajes del archivo cambia su
huella, por lo que debe conservarse el archivo exacto que se haya cotejado.

Sin `-tribunal-salida`, el comando mantiene el contrato anterior: acepta la
huella aportada si tiene formato válido y deja el antecedente y la fase pendientes
de cotejo. Con el archivo, el cotejo sigue siendo local: cualquier persona puede
preparar ese JSON y no se verifica su procedencia institucional.

## Referencias y propuestas

`antecedente_tribunal` identifica una preparación local de S5 por identidad,
versión y `huella_aportada_sha256`. La huella procede de quien prepara el archivo:
el preparador de tribunal no la emite. El cotejo opcional descrito arriba enlaza
el borrador con su salida guardada.
Una referencia con formato válido no acredita la existencia o vigencia del tribunal.

S5 conserva las referencias a la preparación de bases de S2 y al baremo existente.
Este borrador no las duplica ni las verifica. `fase_propuesta` se aporta de forma
explícita; su pertenencia al antecedente queda pendiente de comprobación.

`sesion_ref` y `fecha_propuesta` pueden faltar mientras se prepara el material.
La fecha, si se aporta, usa RFC3339 con zona horaria. Se conserva como propuesta:
no hay fecha actual por defecto ni se infiere cuándo se celebró una sesión.

Cada punto del `orden_dia_propuesto` tiene una referencia única y un texto.
Cada elemento de `acuerdos_propuestos` tiene una referencia única, enlaza un
punto del orden del día y conserva un texto propuesto. Varias propuestas pueden
referirse al mismo punto. Los textos vacíos y las listas vacías quedan pendientes.
No se generan asistentes, cargos, deliberaciones, votos ni acuerdos adoptados.
Los campos de asistencia, votación, aprobación o firma no forman parte de la
entrada y se rechazan como campos desconocidos.

La entrada admite como máximo 1 MiB; cada catálogo, 64 KiB. Se permiten hasta
100 puntos, 100 propuestas de acuerdo y 4096 bytes por texto. Son límites técnicos
del borrador. Se rechazan referencias repetidas, acuerdos con puntos ajenos,
texto con controles o Unicode inválido, claves JSON duplicadas y campos desconocidos.
Las referencias, textos y huella del ejemplo son sintéticos.

El parser estricto permanece local en este CLI, como en la preparación de S5.
Su extracción a una pieza común necesita un propietario y un corte separado;
este incremento no introduce un nuevo marco de serialización.

## Lo que queda pendiente

La comprobación del antecedente y la fase precede al circuito real de designación
y habilitación. La sesión, la asistencia y las deliberaciones necesitan sus hechos
y evidencias. Los acuerdos adoptados, la aprobación y la firma del acta requieren
la actuación competente; este borrador no la sustituye.

Esta separación sigue el [artículo 18 de la Ley 40/2015](https://www.boe.es/buscar/act.php?id=BOE-A-2015-10566#a18),
que regula el contenido del acta de una sesión celebrada y su aprobación.
Las abstenciones y recusaciones siguen el circuito de los
[artículos 23 y 24](https://www.boe.es/buscar/act.php?id=BOE-A-2015-10566#a23);
preparar una propuesta no las resuelve. No se aplica una regla universal de quórum.

Este corte no acredita persistencia, permisos, custodia, firma ni publicación.
Su uso se limita a material sintético. Repetir la entrada produce el mismo borrador;
no constituye recuperación de un acta institucional.

Pruebas focales ejecutadas con resultado correcto:

```sh
go test -p 8 ./internal/modules/seleccion/domain \
  ./internal/modules/seleccion/application ./cmd/vec-selectivos-preparar-acta
```
