# Categorías RPT

`categorias_rpt_v1.sql` publica la versión 1 del catálogo `categorias_rpt` del módulo `personal`. Es el catálogo que lee la incorporación de Contratación temporal a Personal (B2), y con él el vínculo de un expediente con su categoría RPT tiene categorías publicadas donde elegir.

Trae 16 categorías de la RPT de la Diputación de Granada publicada en el portal de Transparencia (revisión del 7 de mayo de 2026, copiada en `data/catalogos/rpt/v1.rpt-2026.json`). Cada categoría lleva la etiqueta en castellano, `etiqueta_en` en inglés, la denominación original de la RPT, los grupos y las escalas.

La publicación la hace el superusuario de la base con la función `publicar`. Las dos referencias de aprobación que pide la función (`catalogos:categorias_rpt:v1:carga-inicial-fuente-rpt` y `catalogos:categorias_rpt:v1:carga-inicial-superusuario`) solo identifican esta carga inicial; detrás no hay una resolución de RRHH.

## Quién lo ejecuta y cómo

Lo ejecuta el superusuario de la base (`postgres`). La función `publicar` solo la puede llamar su propietario, y el script hace `SET ROLE vec_autorizacion_atestada_v3_propietario`. Además lee las tablas del esquema antes y después de publicar, y eso un rol sin superusuario no puede hacerlo. Si no eres superusuario, el script se para al principio. La aplicación no lo ejecuta.

```sh
psql -X -d <base> -f deploy/postgresql/catalogos_configurables/datos/categorias_rpt_v1.sql
```

Todo va en una sola transacción. Si algo falla, no queda nada a medias. Repetirlo no duplica nada: la función devuelve el mismo recibo. Si ya existe una versión 1 con otro contenido, el script se para sin tocarla.

## Cambiar o retirar categorías

Este fichero no se edita. Para cambiar la lista se publica una versión 2 con la misma función, con las preimágenes de las categorías que ya existen. Para retirar una categoría se deshabilita con `cambiar_proyeccion`. La historia de la versión 1 se conserva.

El documento está en los bytes que produce `json.Marshal` del tipo Go `CatalogoConfigurable`, para que la huella que guarda la base coincida con la que calcula la aplicación. Una versión nueva se genera igual.

## Límites conocidos

- El motivo `motivos_catalogos:1:carga_inicial` solo cumple el formato que pide la función. No apunta a ningún catálogo de motivos, igual que en `pruebas_sql/cc11_publicacion_admin_real.sql`.
- Las claves son `categoria:rpt:<clave>` porque el vínculo de Contratación temporal exige que coincidan letra por letra con la categoría del análisis y de Bolsa. El vínculo, la incorporación B2 y el lector de catálogos de Go aceptan los dos puntos.
- La clave `ctpd-auxilar-de-enfermaria` conserva la errata de la RPT original porque Bolsa usa esa misma referencia.
