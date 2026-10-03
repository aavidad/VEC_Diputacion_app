# Comprobar una propuesta de perfil

La herramienta compara una propuesta con un catálogo de acciones y perfiles.
Lee archivos locales y emite el resultado en JSON. No publica perfiles, concede
acceso, inicia servicios ni conecta con una base de datos.

```sh
go run ./cmd/vec-admin-comprobar-perfil \
  web/static/textos/es/admin-comprobar-perfil.json \
  /ruta/privada/catalogo.json /ruta/privada/propuesta.json \
  2026-10-03T12:00:00Z
```

Para obtener el resultado en inglés, use el catálogo de textos de `en`.
La fecha debe usar UTC y una precisión máxima de microsegundos. Los archivos
de datos admiten hasta 4 MiB; el de textos, hasta 64 KiB. Los nombres de campos
son los definidos en los contratos Go, en minúsculas. Campos desconocidos o
repetidos, listas duplicadas, comodines, UTF-8 no válido y estructuras excesivas
se rechazan.

El catálogo usa `CatalogoAccionesAdministracionV1`. Cada entrada conserva la
concesión completa de `ConcesionRol`, las dimensiones de ámbito, la clase de
control, la vigencia y la referencia, versión y huella de su fuente. Los perfiles
publicados incluyen su `VersionRol`, el control de vigencia y un `tipo_perfil`
obligatorio: `fijo_sistema` o `administrable`.

La propuesta usa `PropuestaPerfilAdministracionV1`: referencia, versión y huella
del catálogo, base exacta del perfil anterior, rol propuesto y selección de
entradas con referencia, versión y huella. Las selecciones siguen el mismo orden
que las concesiones del rol. La huella de una entrada se obtiene con su método
`HuellaSHA256`; la del catálogo y la del rol usan sus métodos correspondientes.

Un perfil nuevo exige versión 1 y ninguna base anterior. Un perfil administrable
existente exige la siguiente versión y la referencia y huella exactas de su base
habilitada. Un perfil fijo solo se puede consultar; ninguna propuesta con su
identificador puede modificarlo. Cada concesión propuesta debe coincidir por
completo con su entrada vigente, incluidos campos, finalidades, garantía y
obligaciones. Esta versión del comprobador no admite restringir esas listas.

Código de salida: `0` si la propuesta coincide, `1` si se rechaza y `2` si no se
pueden leer o inicializar los textos, los argumentos son incompletos o falla la salida.
`publicado` siempre es `false`. Un rechazo no devuelve un dictamen positivo.

La comprobación acredita coherencia entre los archivos aportados. No acredita
que procedan de una fuente publicada ni que reflejen el estado actual del
sistema. La integración futura debe resolver un catálogo completo en la fuente
administrativa confiable y conservar las referencias elegidas para aplicar las
dimensiones de ámbito en las asignaciones. Publicar una nueva versión requiere
la autorización central, los controles administrativos y el registro duradero
de solo adición existentes; esta CLI no implementa ese efecto.

Para preparar un plan de creación, nueva versión o retirada de una versión,
añada `--preparar-plan` como quinto argumento:

```sh
go run ./cmd/vec-admin-comprobar-perfil \
  web/static/textos/es/admin-comprobar-perfil.json \
  /ruta/privada/catalogo.json /ruta/privada/intencion.json \
  2026-10-03T12:00:00Z --preparar-plan
```

`intencion.json` usa `SolicitudPlanGobiernoPerfil`:

- `operacion`: `crear`, `versionar` o `deshabilitar`.
- `publicacion`: la propuesta anterior, solo para crear o versionar.
- `deshabilitacion`: selección exacta de catálogo, versión del rol y control
  de vigencia, solo para deshabilitar. Incluye sus referencias, huellas y la
  revisión esperada del control.
- `motivo`: referencia estructurada al catálogo de motivos.
- `referencia_acto`: referencia administrativa opcional.

La salida contiene `plan` y `plan_huella_sha256`. La definición nueva del plan
incluye nombre y concesiones, pero excluye autor, fecha y estado de publicación
declarados en el archivo. La autoridad central debe producir esos metadatos
reales. Una retirada selecciona una versión exacta y exige su control vigente;
no declara retiradas otras versiones del mismo rol. Los perfiles fijos siguen
siendo de solo lectura. Preparar el plan conserva los roles y controles previos
y no cambia ni migra asignaciones.

El plan usa la preparación común de aplicación. `publicado` sigue siendo
`false`. Sus huellas permiten cotejar el material; no acreditan identidad,
permiso, procedencia ni un acto ejecutado. Publicar o retirar requiere la fuente
central completa, dos personas administradoras distintas y una transacción que
una autorización, CAS, historia, auditoría y recibo. El puerto nominal está
preparado; este corte no incluye su proveedor ni una publicación real.

Pruebas focales del comprobador y del contrato:

```sh
go test -p 8 ./internal/vec/domain ./internal/vec/application ./internal/vec/ports ./cmd/vec-admin-comprobar-perfil \
  -run 'CatalogoAccionesAdministracion|ComprobacionPerfilAdministracion|GobiernoPerfil|CLI|LecturaJSON' -count=1
go test -race -p 8 ./internal/vec/domain ./internal/vec/application ./internal/vec/ports ./cmd/vec-admin-comprobar-perfil \
  -run 'CatalogoAccionesAdministracion|ComprobacionPerfilAdministracion|GobiernoPerfil|CLI|LecturaJSON' -count=1
go vet -p 8 ./internal/vec/domain ./internal/vec/application ./internal/vec/ports ./cmd/vec-admin-comprobar-perfil
```

Estas pruebas no ejecutan SQL, servidores ni proveedores externos. En `ports`
comprueban la compilación de la interfaz; el filtro no selecciona pruebas allí.
