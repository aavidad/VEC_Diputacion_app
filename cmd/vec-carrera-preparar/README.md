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
