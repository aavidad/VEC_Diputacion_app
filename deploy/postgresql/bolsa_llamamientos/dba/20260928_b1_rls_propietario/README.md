# Cierre DBA de las políticas RLS de Bolsa B1 — 28/09/2026

`01_cerrar_politicas.sql` es un ajuste aditivo para bases que tengan B1. Se
invoca **antes** del selector de conexiones corporativas. Una comprobación
previa al primer bloqueo distingue tres estados:

- **0/14 tablas y políticas B1:** imprime `NO_APLICA` y no altera nada. Esto
  no acredita que la seguridad esté cerrada; el selector debe comprobar su
  propia preimagen.
- **1–13/14 o deriva estructural:** falla cerrado con nombre de tabla/política.
- **14/14 exactas:** entra en la transacción protegida y cambia solo el destino
  `TO` de las catorce políticas.

En el caso 14/14, las políticas `solo_propietario` pasan de `PUBLIC` a
`vec_bolsa_llamamientos_propietario`. No modifica B1 histórica, tablas, datos,
predicados, ACL, roles, funciones ni políticas de otras migraciones.

Cuando B1 está presente, la transacción exige catorce nombres exactos, dueño
`NOLOGIN` sin `BYPASSRLS`, RLS habilitada y forzada, una única política por
tabla con `polroles={0}`, comando `ALL`, modo permisivo, predicados idénticos
a B1 y ausencia de permisos directos de tabla o columna para otros roles.
Toma bloqueos antes de comprobar y devuelve error `55000` ante deriva. La
postimagen exige los mismos OID, predicados y ACL de tabla y columna, con
solo el propietario en `polroles`.

No hay `DOWN`: devolver `TO PUBLIC` reabriría el defecto. Si la preimagen no
coincide, detener la instalación y revisar la base; no relajar guardas ni
reaplicar migraciones con historia. La aplicación de este fichero no acredita
por sí sola el selector ni un despliegue.

Prueba aislada sobre PostgreSQL 18.4 real, con la cadena original hasta B1 y
datos sintéticos:

```bash
deploy/postgresql/bolsa_llamamientos/dba/20260928_b1_rls_propietario/probar_pg18.sh
```

El runner usa un contenedor efímero sin red. Prueba `NO_APLICA` sin esquema y
con esquema Bolsa, los trece estados parciales y seis derivas negativas:
predicado, rol, política adicional, ACL de tabla, ACL de columna y `FORCE RLS`.
Comprueba ausencia de efectos parciales, historia/ACL conservadas y rechazo de
reaplicación. No instala SQL en bases compartidas.
