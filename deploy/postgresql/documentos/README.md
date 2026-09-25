# Documentos comunes B5: persistencia

Este módulo conserva metadatos, numeración interna, referencias a originales
inmutables, instantánea de conservación, preparaciones de notificación,
auditoría y outbox. Los bytes originales permanecen en `AlmacenObjetos` y el
adaptador comprueba su recibo antes de confirmar el documento. El número
`VEC-AAAA-N` no es un asiento de registro general.

## Orden de instalación

En una base preparada y con la cadena VEC-AD-3 hasta 000052 instalada:

1. `roles_up.sql` como DBA, una sola vez.
2. `../autorizacion_atestada_v3/migraciones/000060_documentos_comunes.up.sql` por su migrador.
3. `migraciones/000001_documentos_comunes.up.sql` por el migrador documental.
4. Crear fuera de Git el LOGIN de aplicación con **solo** la membresía
   `vec_documentos_ejecutor`; no conceder propiedad, migración ni acceso a tablas.

No reaplicar ni ejecutar `DOWN` sobre bases con historia. Cada fachada exige
transacción `SERIALIZABLE`, material V3 nominal y consumo nuevo en el mismo
`COMMIT` que la actuación y auditoría. La consulta registra el acceso antes de
resolver el objeto por su referencia y versión inmutables.

`probar_integracion_pg18.sh` crea y destruye su propio contenedor PostgreSQL
18.4. Comprueba `ROLLBACK`/`COMMIT`, RLS forzada, ACL y recuperación tras reinicio
sobre una **preimagen sintética** AD3-50 seguida de las migraciones reales
AD3-51/52/60. Esa preimagen no acredita las decisiones COSE, las políticas ni
el consumo V3 de una cadena instalada de verdad. El ensayo completo en una
base desechable con la cadena real sigue siendo condición de integración.

`estado_firma` permanece `pendiente_proveedor` e inmutable. La integración con
AutoFirmaV2 requiere recibo verificable del verificador y otro consumidor V3
nominal; la referencia del objeto firmado o un resultado booleano no habilitan
la transición. La preparación de notificación no declara envío ni entrega.
