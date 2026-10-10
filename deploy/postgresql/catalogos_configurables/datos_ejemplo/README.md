# Categorías RPT de ejemplo

`categorias_rpt_ejemplo_v1.sql` publica la versión 1 del catálogo `categorias_rpt` del módulo `personal`, que es el que lee la incorporación de Contratación temporal a Personal (B2). Así el vínculo de un expediente con su categoría RPT tiene categorías publicadas donde elegir.

Trae 16 categorías de la RPT pública de la Diputación de Granada (revisión del 7 de mayo de 2026, copiada en `data/catalogos/rpt/v1.rpt-2026.json`). Es un paquete de ejemplo que se puede retirar. No es la publicación jurídica de la RPT y no tiene aprobación de RRHH: lo dicen la fuente, las dos referencias de aprobación y el motivo del documento. Cada categoría lleva el atributo `paquete = ejemplo`, la etiqueta en castellano, `etiqueta_en` en inglés y la denominación original de la RPT.

## Quién lo ejecuta y cómo

Lo ejecuta el superusuario de la base (`postgres`). La función `publicar` solo la puede llamar su propietario, y el script hace `SET ROLE vec_autorizacion_atestada_v3_propietario`. Además lee las tablas del esquema antes y después de publicar, y eso un rol sin superusuario no puede hacerlo. Si no eres superusuario, el script se para al principio. La aplicación no lo ejecuta.

```sh
psql -X -d <base> -f deploy/postgresql/catalogos_configurables/datos_ejemplo/categorias_rpt_ejemplo_v1.sql
```

Todo va en una sola transacción. Si algo falla, no queda nada a medias. Repetirlo no duplica nada: la función devuelve el mismo recibo. Si ya existe una versión 1 con otro contenido, el script se para sin tocarla.

## Cambiar o retirar las categorías

Este fichero no se edita. Para cambiar la lista se publica una versión 2 con la misma función, con las preimágenes de las categorías que ya existen. Para retirar una categoría se deshabilita con `cambiar_proyeccion`. La historia de la versión 1 se conserva.

El documento está en los bytes que produce `json.Marshal` del tipo Go `CatalogoConfigurable`, para que la huella que guarda la base coincida con la que calcula la aplicación. Una versión nueva se genera igual.

## Límites conocidos

- El motivo `motivos_catalogos:1:paquete_ejemplo` solo cumple el formato que pide la función. No apunta a ningún catálogo de motivos, igual que en `pruebas_sql/cc11_publicacion_admin_real.sql`.
- Las claves son `categoria:rpt:<clave>` porque el vínculo de Contratación temporal exige que coincidan letra por letra con la categoría del análisis y de Bolsa. El vínculo y la incorporación B2 aceptan los dos puntos. El lector estricto de catálogos de Go (`ListarCategoriasHabilitadasRPT`) no los acepta, y hoy nadie lo usa fuera de las pruebas. Antes de montar una pantalla que liste con ese lector hay que alinear la regla de claves del dominio con la de la base.
- La clave `ctpd-auxilar-de-enfermaria` conserva la errata de la RPT original porque Bolsa usa esa misma referencia.
