# Motivos de retirada de la cartera

`motivos_cartera_ejemplo_v1.json` ofrece dos motivos para que RRHH revise el
catálogo antes de publicarlo. Es un borrador: la migración CC13 no publica sus
entradas ni les da vigencia.

La publicación requiere dos pasos de gobierno. Primero se publica en
Autorización el catálogo `motivos_meritos_cartera` con las mismas claves.
Después se publica `meritos.cartera.motivos` con sus etiquetas en castellano e
inglés, estado `publicado` y `vinculo_autorizacion` que incluya la versión y
huella del primer catálogo. La etiqueta principal de cada entrada debe
coincidir con `etiquetas.es`. Ambas publicaciones conservan su historia y sus
aprobaciones; no se preparan mediante una migración.

El motivo técnico de la retirada V3 es distinto: su referencia publicada en
`motivos_autorizacion_meritos_cartera` se guarda en el fichero privado
`identidad/meritos-cartera-externa.json` con permisos `0600`:

```json
{
  "version": 1,
  "motivo_retirada": {
    "catalogo_id": "motivos_autorizacion_meritos_cartera",
    "catalogo_version": 1,
    "catalogo_huella_sha256": "<huella real publicada>",
    "entrada_clave": "<clave real publicada>"
  }
}
```

La aplicación muestra la cartera como no disponible hasta que existan las
publicaciones, los cinco accesos PostgreSQL y el material V3 propios de M4.
