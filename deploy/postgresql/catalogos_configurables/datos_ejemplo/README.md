# Categorías RPT de ejemplo

`categorias_rpt_ejemplo_v1.sql` publica la versión 1 del catálogo `categorias_rpt` del módulo `personal`, que es el que lee la incorporación de Contratación temporal a Personal (B2). Así el vínculo de un expediente con su categoría RPT tiene categorías publicadas donde elegir.

Trae 16 categorías de la RPT pública de la Diputación de Granada (revisión del 7 de mayo de 2026, copiada en `data/catalogos/rpt/v1.rpt-2026.json`). Es un paquete de ejemplo que se puede retirar. No es la publicación jurídica de la RPT y no tiene aprobación de RRHH: lo dicen la fuente, las dos referencias de aprobación y el motivo del documento. Cada categoría lleva el atributo `paquete = ejemplo`, la etiqueta en castellano, `etiqueta_en` en inglés y la denominación original de la RPT.

## Quién lo ejecuta y cómo

Lo ejecuta una persona con rol de administrador de la base (DBA), porque la función `publicar` solo la puede llamar su propietario y el script hace `SET ROLE vec_autorizacion_atestada_v3_propietario`. La aplicación no lo ejecuta.

```sh
psql -X -d <base> -f deploy/postgresql/catalogos_configurables/datos_ejemplo/categorias_rpt_ejemplo_v1.sql
```

Todo va en una sola transacción. Si algo falla, no queda nada a medias. Repetirlo no duplica nada: la función devuelve el mismo recibo. Si ya existe una versión 1 con otro contenido, el script se para sin tocarla.

## Cambiar o retirar las categorías

Este fichero no se edita. Para cambiar la lista se publica una versión 2 con la misma función, con las preimágenes de las categorías que ya existen. Para retirar una categoría se deshabilita con `cambiar_proyeccion`. La historia de la versión 1 se conserva.

El documento está en los bytes que produce `json.Marshal` del tipo Go `CatalogoConfigurable`, para que la huella que guarda la base coincida con la que calcula la aplicación. Una versión nueva se genera igual.
