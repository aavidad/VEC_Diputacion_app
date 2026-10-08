# Borrador de servicios para ensayo

Desde la raíz del repositorio, prepara un PDF y un JSON con la muestra
sintética incluida. Conserva la versión y las huellas de la fuente, la
plantilla y los textos. No emite un certificado ni registra una solicitud.

El directorio de salida debe ser nuevo. La herramienta no sobrescribe uno
existente y crea los archivos con permisos privados.

```sh
mkdir -m 700 /var/tmp/vec-cer-ensayo
TMPDIR=/var/tmp/vec-cer-ensayo go run -p 4 \
  ./cmd/vec-certificados-borrador \
  -ensayo-sintetico \
  -fuente internal/modules/certificados/adapters/fichero/testdata/servicios.ensayo.json \
  -salida /var/tmp/vec-cer-ensayo/salida
```

Revise `salida/borrador.pdf` y `salida/borrador.json`. Los catálogos de idiomas
aportan los mensajes y los formatos de fecha; `-h` muestra las opciones.
La marca sintética confirma el uso de ensayo: no detecta ni anonimiza datos
personales. Use únicamente muestras ficticias.

```sh
TMPDIR=/var/tmp/vec-cer-ensayo go test -race -p 4 \
  ./cmd/vec-certificados-borrador ./internal/modules/certificados/...
TMPDIR=/var/tmp/vec-cer-ensayo go vet -p 4 \
  ./cmd/vec-certificados-borrador ./internal/modules/certificados/...
```

`-fuente` admite también una muestra con la forma de la respuesta del contrato
`LectorServiciosParaCertificadosV1` de Personal; el CLI la reconoce por su
esquema y la traduce con el mismo adaptador que usará la respuesta real:

```sh
TMPDIR=/var/tmp/vec-cer-ensayo go run -p 4 \
  ./cmd/vec-certificados-borrador \
  -ensayo-sintetico \
  -fuente internal/modules/certificados/adapters/personalv1/testdata/servicios-personal-v1.ensayo.json \
  -salida /var/tmp/vec-cer-ensayo/salida-v1
```

La pieza no conecta con Personal, HTTP, PostgreSQL ni proveedores de firma.
El generador PDF común fija sus metadatos en castellano; el ensayo traducido
no acredita accesibilidad lingüística completa ni PDF/UA.

Los contratos, límites y pasos de emisión pendientes se describen en
[CER-001](../../docs/estudio_requisitos/certificados_borrador_servicios.md).
Retire la carpeta de ensayo al terminar la revisión.
