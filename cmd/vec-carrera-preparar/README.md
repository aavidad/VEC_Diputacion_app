# Preparar un expediente de Carrera

La CLI coteja la integridad de datos sintéticos para las vías de grado,
progresión laboral y promoción interna. Mantiene el nivel del puesto separado
del grado personal. Señala fechas inválidas, periodos solapados, fuentes o
versiones incompletas y compatibilidades declaradas en la política aportada.
No suma periodos ni aplica umbrales de antigüedad.

```sh
go run ./cmd/vec-carrera-preparar < cmd/vec-carrera-preparar/testdata/entrada.json
```

La entrada admite un documento JSON de hasta 1 MiB, 64 casos, 32 fuentes por
caso y 128 periodos o evidencias. Las claves usan minúsculas. Rechaza campos desconocidos, miembros
duplicados y documentos concatenados. Los errores devuelven una clave JSON
por stderr y código de salida 1, sin reproducir los datos de entrada.

El ejemplo incluye nombres y referencias ficticios. El campo `alcance` debe
ser `preparacion_sintetica`. La fuente, su versión, la política, sus regímenes
y las referencias de aprobación proceden del JSON. Una referencia disponible
solo acredita que el dato está aportado; esta herramienta no autentica fuentes
ni aprobaciones.

La salida determinista contiene `casos`, `comprobaciones` y `pendientes`.
Cada caso incluye `antecedentes` con los grupos declarados, periodos y sus
evidencias, política y referencias de aprobación, convenio, curso, prueba y
convocatoria. Conserva las referencias, fuentes y versiones aportadas para
revisarlas al descargar el JSON. La salida mantiene una copia independiente
de la entrada.
Cada caso permanece en `estado_global: pendiente`. La progresión requiere
referencias de convenio y versión, la declaración `convenio_consolidado`, curso,
prueba y sus aprobaciones. La promoción
solo conserva la referencia de convocatoria del módulo de procesos selectivos.

El reconocimiento y la inscripción en Personal quedan pendientes en todos los
casos. La herramienta no cambia grado, categoría, retribución o relación de
servicio, no crea resoluciones ni asientos, y no conecta red, identidad,
autorización o persistencia. Su resultado prepara la revisión documental.

Para revisar el borrador en Chrome desde la raíz del repositorio:

```sh
python3 scripts/servir_preparacion_rrhh.py --modulo carrera
```

El comando muestra una dirección local. El visor permite filtrar, revisar pendientes
y descargar el borrador; no se publica en los portales ni modifica expedientes.

Para reunir la instantánea de antecedentes y el catálogo de política en una revisión:

```sh
go run ./cmd/vec-carrera-preparar --expediente-sintetico \
  data/catalogos/carrera/politica_grado_ejemplo.json \
  cmd/vec-carrera-preparar/testdata/antecedentes_expediente.json \
  < cmd/vec-carrera-preparar/testdata/entrada_expediente.json
```

Los tres archivos son configurables y sintéticos. La instantánea contiene referencias,
fecha de corte, cobertura, ocupaciones, grado y servicios con sus evidencias y versiones.
Su `caso_ref` y `version` deben corresponder al escenario. Cada archivo admite hasta
1 MiB; los antecedentes admiten hasta 64 instantáneas, sin referencias repetidas.

La salida reúne `preparacion`, `politica_grado_sintetica` y `revision_expediente`.
En la revisión, `Declaracion` conserva los datos iniciales y `Antecedentes` conserva
la instantánea y sus `Faltantes`. `Contradicciones` indica qué campos difieren cuando
ambas fuentes aportan un valor: régimen, grupo, nivel del puesto o grado personal.
Una ausencia conserva su faltante y nunca se presenta como contradicción comprobada.
Las comprobaciones de la preparación explican fechas, procedencia y solapes mediante
las claves del catálogo existente.

El ejemplo declara nivel 22 y aporta nivel 24 en la ocupación; mantiene el grado 20
separado. Contiene dos periodos solapados y un servicio declarado: conserva los dos
periodos sin sumarlos y deja el servicio declarado fuera de los periodos reconocidos.
Los datos etiquetados como reconocidos pertenecen al ensayo; no acreditan actos reales.

La política conserva la referencia de aprobación aportada en su revisión, pero la
preparación no la usa como aprobación competente. Todos los expedientes siguen
pendientes. El contrato nominal, el lector autorizado de Personal y la autorización
de Carrera también permanecen pendientes. El visor actual consume la preparación
simple; este informe conjunto se consulta y conserva desde la CLI.
