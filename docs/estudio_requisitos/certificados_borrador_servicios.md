# CER-001: preparar un borrador de servicios

Este corte permite revisar un PDF de servicios y su JSON con una muestra
sintética. Usa una plantilla versionada y el generador PDF común de VEC.
El encabezado y el primer bloque indican que es un borrador de ensayo.

CER-001 incluye datos de oficio, revisión, firma o sello, registro, CSV,
entrega y conservación. Esta pieza prepara material para revisar; las demás
operaciones siguen pendientes. No está montada en el portal ni conecta con
Personal, la base de datos o los proveedores de firma.

## Probar la pieza

Desde la raíz de este worktree, elija una carpeta de salida que aún no exista:

```sh
TMPDIR=/tmp go run ./cmd/vec-certificados-borrador \
  -ensayo-sintetico \
  -fuente internal/modules/certificados/adapters/fichero/testdata/servicios.ensayo.json \
  -salida /tmp/vec-certificado-servicios
```

La carpeta contiene `borrador.pdf` y `borrador.json`, con permisos privados.
Si la carpeta ya existe, el comando termina sin sobrescribirla. El JSON
conserva el contenido mostrado en el PDF, los periodos, su estado y las
huellas SHA-256 de la muestra, la plantilla y el catálogo de textos.
Las mismas entradas generan el mismo PDF.

El idioma de respaldo y los idiomas admitidos salen de
`web/static/textos/idiomas.json`. Se puede seleccionar uno con `-idioma`.
Los mensajes y formatos de fecha viven en
`web/static/textos/<idioma>/certificados.json`; el código no enumera idiomas.
`-h` muestra las opciones. `-indice-idiomas` y `-textos` permiten usar otro
índice y catálogo locales durante el ensayo.

La versión sale del fichero de plantilla. Para exigir una versión concreta,
indíquela con `-version`; una discrepancia impide preparar el borrador.
`-plantilla` permite seleccionar otro fichero versionado del mismo tipo.
La plantilla contiene claves de texto ordenadas, sin expresiones ejecutables.
Debe conservar los bloques de límites, persona, corte, fuente, versión y
criterio. La plantilla incluida está en estado de ensayo; su contenido
requiere la aprobación de RRHH antes de una emisión real.

## Contrato y límites

El caso de uso `application.Preparador.PrepararEnsayo` recibe una orden con
identificador y versión de plantilla e idioma. Coordina tres puertos:
`FuenteServicios`, `CatalogoPlantillas` y `Renderizador`.
El adaptador de fichero aporta una instantánea sintética con un corte de
vigencia y otro de información conocida. El renderizador delega en
`internal/vec/adapters/documentos/pdf`.

La muestra usa a Elena Martín Robles, un nombre ficticio. La entrada exige
el esquema de ensayo, `sintetica: true`, una referencia de ensayo y el
argumento `-ensayo-sintetico`. Estas marcas expresan la confirmación del
operador; no detectan ni anonimizan datos reales. Este consumidor local no
autoriza su uso. No deben introducirse datos reales.

Los servicios declarados, comprobados y reconocidos se muestran por
separado. Los días se copian de la muestra; no se suman ni recalculan.
Los solapamientos se conservan para su revisión. Una muestra vacía dice
que no contiene servicios, sin afirmar que la persona no los haya prestado.
Se rechazan fechas inválidas, periodos posteriores al corte, estados
incompatibles, caracteres de control y entradas superiores a 200 servicios.
Los JSON tienen un límite técnico de 256 KiB y rechazan campos desconocidos,
claves duplicadas y contenido adicional.

No se crea un certificado emitido, recibo de autorización, acto de
reconocimiento, firma, sello, CSV, asiento de registro ni entrega. Para el
siguiente corte, Personal debe aportar su proyección autorizada con
procedencia, actos, periodos, jornada e interrupciones. La emisión deberá
usar la autorización, el archivo documental y la firma ya existentes,
sin crear autoridades paralelas.

El generador PDF común fija actualmente el idioma de sus metadatos en
castellano. El contenido puede estar traducido, pero el PDF en inglés no
acredita accesibilidad lingüística completa. Este corte tampoco acredita
PDF etiquetado ni PDF/UA. Su corrección pertenece al generador común.

## Fuente con la forma del contrato V1 de Personal (CER-002)

Personal publica el contrato `LectorServiciosParaCertificadosV1`
(`internal/modules/personal/ports/servicios_para_certificados.go`). Todavía
no tiene implementación ni montaje. El adaptador
`internal/modules/certificados/adapters/personalv1` traduce su respuesta a la
fuente del borrador, sin reglas de cómputo:

- el periodo de Personal es semiabierto, [desde, hasta): el último día del
  servicio es el anterior a «hasta». Un «hasta» vacío es un periodo abierto y
  el borrador lo presenta así, sin inventar una fecha de fin. V1 no recorta los
  periodos al corte: si el fin previsto es posterior a la fecha de referencia
  (un temporal en activo), el servicio se presenta en curso a esa fecha;
- V1 no trae días. El borrador no los muestra ni los calcula y cambia el
  bloque de criterio por uno que lo dice;
- se conservan la cobertura (completa, parcial o no acreditada), la certeza de
  la procedencia, el acto y la versión de la clase de cada servicio;
- solo un servicio reconocido con procedencia acreditada se marca como capaz
  de sustentar un certificado. Un declarado o comprobado no se convierte en
  reconocido;
- V1 no trae el nombre: llega aparte, desde la autoridad de identidad.

La forma V1 solo entra por este traductor: el adaptador de fichero de ensayo
rechaza una fuente que ya venga con el esquema V1 escrito a mano.

Con la muestra sintética
`adapters/personalv1/testdata/servicios-personal-v1.ensayo.json` el CLI
prepara el mismo tipo de borrador. Una fuente no sintética se sigue
rechazando: este consumidor no tiene autorización para datos reales.

Queda para el paso siguiente: que Personal implemente el lector con su
autorización y auditoría, y montar en el servidor la consulta propia del
empleado con identidad y permiso. Para RRHH conviene además que el contrato
V1 aporte los días reconocidos, que Personal ya guarda; es una decisión de su
dueño. Antes de emitir, el borrador debe conservar también la versión de la
respuesta, las referencias de empleado y organismo y la fuente y versión de
cada procedencia, para trazar el certificado a su origen; la traducción de
ensayo aún no las guarda.

## Verificación focal

```sh
TMPDIR=$HOME/.cache/vec-claude-t go test -race -p 4 \
  ./cmd/vec-certificados-borrador ./internal/modules/certificados/...
TMPDIR=$HOME/.cache/vec-claude-t go vet -p 4 \
  ./cmd/vec-certificados-borrador ./internal/modules/certificados/...
~/go/bin/gosec -quiet -fmt text \
  ./cmd/vec-certificados-borrador/... ./internal/modules/certificados/...
semgrep --metrics=off --config p/golang --error \
  cmd/vec-certificados-borrador internal/modules/certificados
```

Las pruebas cubren PDF real en los dos catálogos incluidos, determinismo,
permisos privados, carpeta existente, confirmación de ensayo, un idioma
adicional aportado solo como datos, versión exacta, cancelación y rechazo
antes del renderizado. También cubren la conservación de periodos solapados
y la copia defensiva de los servicios. La revisión independiente y la CI
corresponden a dirección antes de integrar.

## Fuentes del alcance

- [Catálogo funcional: CER-001](catalogo_funcional_rrhh_y_hoja_ruta.md).
- [Análisis integral: servicios y documentos](analisis_integral_rrhh.md).
- [Petición de RRHH](peticion_rrhh_transcripcion_y_lectura.md).
- [Matriz normativa](matriz_normativa_rrhh_2026.md).

Estas fuentes asignan servicios a Personal y requieren revisión y firma.
El ensayo no introduce reglas de cómputo ni una plantilla administrativa
aprobada.
