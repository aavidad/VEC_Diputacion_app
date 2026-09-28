# Delta DBA de tipos fila CT/AD3 previo al selector

Este delta cierra `USAGE PUBLIC` de los 28 tipos fila creados por
Contratación temporal 000001, 000002, 000003, 000004, 000006, 000007,
000008 y 000014 y los 14 creados por Autorización atestada V3 000001.
La migración AD3 000002 no crea tipos fila adicionales. La lista explícita
y los propietarios están en
`acl_tipos_ct_ad3_preselector_v1.up.sql`.

## Orden causal

En la cadena nueva PG18 UTF8: instalar las migraciones CT 000001–000016 y
AD3 000001–000002 una sola vez; instalar Bolsa B1 y su delta DBA RLS;
instalar Contexto 000003 y ejecutar el delta C3 existente; aplicar este
delta CT/AD3 **antes** de
`roles_contexto_corporativo_rrhh_selector_v1_up.sql`. El selector inspecciona
`PUBLIC` en todos los tipos de los esquemas no internos y rechaza la cadena
si los 42 siguen abiertos. Ejecutar el delta bajo el DBA superusuario en
una base local aislada o en el canal de instalación autorizado. No ejecutarlo
contra una base principal por simple arranque de aplicación.

El SQL toma un bloqueo transaccional del catálogo de tipos, coteja que existan
exactamente los 42 tipos fila declarados, que cada tipo sea el de su tabla y
tenga el propietario y ACL esperados, y revoca únicamente `USAGE` a
`PUBLIC`. Si los 42 ya están cerrados, emite `NO_APLICA` sin mutación. Un
conjunto parcial, owner distinto, ACL extra o selector ya instalado con
tipos abiertos se rechaza con SQLSTATE `55000`. Un estado ya cerrado
también devuelve `NO_APLICA` con selector instalado. No hay `DOWN`: volver a conceder
`USAGE PUBLIC` reabriría la frontera. Tras cualquier DDL posterior,
repetir la puerta global del selector antes de continuar.

## Ensayo focal reproducible

```sh
bash deploy/postgresql/contexto_actor_v1/pruebas_sql/probar_acl_tipos_ct_ad3_preselector_v1_pg18.sh
```

La prueba arranca un contenedor PostgreSQL 18.4 nuevo, sin red ni puertos y
con el repositorio montado solo lectura. Carga la cadena sintética indicada
en `fixture_acl_ct_ad3_preselector_v1.txt`, verifica 42 ACL abiertas,
ROLLBACK, estado parcial, concesión ajena, cierre, ausencia de cambios en
ACL ajenas, `NO_APLICA` repetido antes y después del selector, recuperación
tras reinicio y alta del selector. El contenedor se borra al finalizar. Esta prueba no instala nada
en la base principal ni acredita los pasos posteriores de la cadena.
