# Datos de organización de Personal

`organizacion_rpt_2026_v1.sql` carga la organización 2026 en las tablas de historia de Personal (migración 000010). Sale de la RPT publicada de la Diputación de Granada, revisión del 7 de mayo de 2026, que está en `data/catalogos/rpt/v1.rpt-2026.json`.

Qué carga:

- la versión de la RPT, publicada, con la huella SHA-256 de ese fichero;
- la plantilla 2026, publicada;
- 32 puestos tipo, dos por cada categoría del catálogo `categorias_rpt` v1 (administrativo, auxiliar administrativo, analista programador, etc.), con su denominación, centro, nivel y dotación tal como figuran en la RPT;
- un puesto y una plaza por puesto tipo, unidos por un vínculo confirmado;
- la cobertura de ocupaciones de la plantilla, que consta completa porque al cargarla no hay ninguna plaza ocupada.

Con esto, la incorporación a Personal de Contratación temporal (B2) encuentra vacantes y puede resolver la plaza y el puesto elegidos.

Los códigos de plaza (`2026/0001` a `2026/0032`) y el número de puesto (`<código RPT>/01`) no salen de la RPT, que no publica la plantilla. El resto sí.

## Cómo se ejecuta

Necesita antes la migración Personal 000040. Lo lanza el superusuario de la base, con el `organismo_ref` que tiene `personal-b2/servidor.json`:

```sh
psql -X -d <base> -v organismo=<organismo_ref> -f deploy/postgresql/personal/datos/organizacion_rpt_2026_v1.sql
```

Va en una sola transacción. Si se repite, no cambia nada. Si el organismo ya tiene otra RPT o plantilla, o si con las mismas referencias hay otro contenido, se para sin tocar nada.

## Cambiar los datos

Este fichero no se edita. Las tablas solo admiten añadir, así que un cambio es una revisión nueva (por ejemplo, una plaza amortizada o un puesto suprimido) o una versión nueva de RPT o plantilla que sustituye a la anterior. Lo anterior queda en la historia.

## Comprobación

`pruebas_sql/personal40_cobertura_fuente_plantilla.sql` comprueba, sin dejar cambios, que las 32 plazas se pueden seleccionar para el plan B2, que la cobertura es única y que una cobertura completa de otra fuente se rechaza.
