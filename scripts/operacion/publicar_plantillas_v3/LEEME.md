# Publicar las plantillas CT v3

Dirección ejecuta este guion en cidonia como `openclaw`, después de desplegar
#1065. Prepara las diez plantillas y sus etiquetas para los PDF y Word de la
ficha CT. La comprobación no cambia datos ni archivos.

Se usa la alternativa SQL de datos ficticios autorizada el 12/10. La composición
actual de plantillas solo monta un actor RRHH y la CLI de identidad no concede
su perfil. Este guion no crea una segunda identidad, certificados ni permisos,
ni presenta la operación como una publicación aprobada por dos personas RRHH.
Conserva en la base una referencia explícita de la operación de datos, su
instalador SQL, fecha, recibo, auditoría de provisión y evento, en una transacción.
No crea un consumo V3 ni una firma. Claude revisa el código antes de ejecutarlo.

## Preparación

Compilar en el equipo local desde la rama de esta PR (Go 1.26.9), y copiar el
binario y esta carpeta al paquete de operación de cidonia. El catálogo v3 es
el archivo de #1065; no hay otra copia de los textos en este guion.

```sh
go build -p 6 -o /var/tmp/publicar-plantillas-v3 ./scripts/operacion/publicar_plantillas_v3
```

Instalar el binario fuera de Git y dar a `openclaw` permiso de ejecución:

```sh
install -m 0700 /ruta/de/entrega/publicar-plantillas-v3 /ruta/privada/publicar-plantillas-v3
export VEC_PLANTILLAS_V3_BIN=/ruta/privada/publicar-plantillas-v3
export VEC_PLANTILLAS_V3_CATALOGO=/ruta/del/paquete/data/catalogos/contratacion-temporal/plantillas_documentos.v3.json
```

Usar la conexión privada de instalación de CT con acceso al propietario,
mediante `PGHOST`, `PGPORT`, `PGUSER`, `PGDATABASE` y `PGPASSFILE`, o
`VEC_PLANTILLAS_V3_DATABASE_URL`. No usar la cuenta de ejecución de la app ni
poner contraseñas en argumentos, Git o el canal. `--comprobar` valida el
catálogo exacto y la v1 conservada; ante otra publicación o un borrador ajeno
se detiene indicando los valores que no coinciden.

## Aplicación y recuperación

```sh
scripts/operacion/publicar_plantillas_v3/publicar.sh --comprobar
scripts/operacion/publicar_plantillas_v3/publicar.sh --aplicar PUBLICA-PLANTILLAS-V3
scripts/operacion/publicar_plantillas_v3/publicar.sh --comprobar
```

La primera ejecución devuelve `registrado`; las siguientes devuelven `replay`
con el mismo recibo y auditoría. Si se corta la conexión, repetir la misma
orden. Nunca borrar historia, hacer DOWN o reprovisionar la v1. No se fabrica
una v2 intermedia: v3 conserva la referencia anterior del catálogo de #1065.

CONFIG NUEVA de la app: **ninguna**. Estas dos líneas de `arrancar_app.sh` se
conservan exactamente; dirección no debe sustituir la fuente por v3:

```sh
export VEC_CT_PLANTILLAS_SOURCE_PATH=/vec-incorporacion/ct_plantillas_documentos.ejemplo.demo.json
export VEC_CT_PLANTILLAS_DOCUMENTAL_ENABLED=true
```

No activar `VEC_CT_PLANTILLAS_GOBIERNO_ENABLED` para esta operación. La fuente
v1 es la preimagen del arranque; el lector documental instalado toma la última
publicación de PostgreSQL en cada descarga. Mantenerla evita romper el recibo
provisionado en cidonia (`recibo:ed1e5032…`).

## Comprobación desde la ficha

Abrir un expediente en nombramiento (por ejemplo `2026/CT-000132`), actualizar
la ficha y descargar **Resolución → PDF**. La petición existente es:

```text
POST /api/vec/contratacion-temporal/expedientes/borradores
{"expediente_ref":"expediente:referencia-de-la-ficha","version_observada":7,"tipo":"resolucion","formato":"pdf"}
```

Usar la referencia y versión actuales de la ficha. La respuesta debe ser 200,
con `X-VEC-Catalogo-Ref` terminado en `:3`, y el PDF debe carecer de las marcas
«borrador de desarrollo», «Datos sintéticos» y «modelo de ejemplo». El estado
de firma del documento sigue siendo el que conste en el expediente. Los PDF
ya emitidos y custodiados conservan sus bytes, recibos y plantilla original;
esta operación afecta a las descargas nuevas con el catálogo publicado.
