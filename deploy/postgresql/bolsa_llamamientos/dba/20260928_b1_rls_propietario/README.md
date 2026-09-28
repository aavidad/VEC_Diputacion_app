# Cierre DBA de las políticas RLS de Bolsa B1 — 28/09/2026

`01_cerrar_politicas.sql` es un ajuste aditivo de una sola ejecución. Debe
aplicarse **después** de `bolsa_llamamientos` B1 y **antes** del selector de
conexiones corporativas. Solo cambia el destino `TO` de las catorce políticas
`solo_propietario` creadas en B1: de `PUBLIC` a
`vec_bolsa_llamamientos_propietario`. No modifica B1 histórica, tablas, datos,
predicados, ACL, roles, funciones ni políticas de otras migraciones.

La transacción exige la preimagen cerrada: catorce nombres exactos, dueño
`NOLOGIN` sin `BYPASSRLS`, RLS habilitada y forzada, una única política por
tabla con `polroles={0}`, comando `ALL`, modo permisivo, predicados idénticos
a los de B1 y ausencia de permisos directos para otros roles. Toma bloqueos de
tabla antes de comprobar y devuelve error `55000` ante deriva. La postimagen
comprueba que los mismos OID, predicados y ACL persisten y solo el propietario
figura en `polroles`.

No hay `DOWN`: devolver `TO PUBLIC` reabriría el defecto. Si la preimagen no
coincide, detener la instalación y revisar la base; no relajar guardas ni
reaplicar migraciones con historia. La aplicación de este fichero no acredita
por sí sola el selector ni un despliegue.

Prueba aislada sobre PostgreSQL 18.4 real, con la cadena original hasta B1 y
datos sintéticos:

```bash
deploy/postgresql/bolsa_llamamientos/dba/20260928_b1_rls_propietario/probar_pg18.sh
```

El runner usa un contenedor efímero sin red, verifica cinco derivas negativas
(predicado, rol, política adicional, ACL y `FORCE RLS`), ausencia de efectos
parciales, historia/ACL conservadas y rechazo de reaplicación. No instala SQL
en bases compartidas.
