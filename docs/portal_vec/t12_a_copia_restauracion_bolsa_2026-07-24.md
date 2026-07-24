# T12-A — copia y restauración de baremación de Bolsa

**Fecha:** 24 de julio de 2026  
**Estado:** candidata implementada y probada; T12 permanece abierto  
**Datos admitidos:** exclusivamente sintéticos

## Resultado

La vertical PostgreSQL V3 de baremación dispone de una prueba reproducible de
recuperación ante pérdida total de la instancia. El corredor crea historia y
manifiestos probatorios, genera los artefactos, elimina el servidor origen y
restaura en otro PostgreSQL 18.4 sin compartir el volumen de datos.

No se añade un repositorio paralelo ni se cambia el dominio. La restauración
actúa sobre el adaptador PostgreSQL existente y conserva la separación
hexagonal: el procedimiento de continuidad no se filtra al núcleo.

## Flujo probado

1. Arranca un PostgreSQL aislado desde una imagen fijada por digest.
2. Instala autorización, baremación, outbox y manifiesto probatorio V3.
3. Confirma fixtures sintéticos y calcula una huella SHA-256 de todas las
   tablas funcionales, con filas JSON ordenadas canónicamente.
4. Exporta:
   - roles globales con `pg_dumpall --roles-only --no-role-passwords`;
   - base completa en formato personalizado con `pg_dump --create`.
5. Rechaza el artefacto de roles si contiene `PASSWORD`, SCRAM o hash `md5`.
6. Protege los tres ficheros con modo `0600` y un inventario `SHA256SUMS`.
7. Destruye el contenedor origen.
8. Arranca una instancia limpia, verifica los artefactos y restaura primero
   los roles y después la base.
9. Reinicia PostgreSQL y exige igualdad exacta de la huella funcional.
10. Repite los inventarios de rol, membresía, ACL, RLS, fachadas ejecutables y
    recuperación de evidencias.

El volumen Docker temporal se elimina incluso ante error o interrupción. No se
escriben copias en el árbol del repositorio ni se imprime la clave aleatoria
de inicialización.

## Evidencias

- `deploy/postgresql/bolsa_baremacion/probar_copia_restauracion_v3.sh`
- `pruebas_sql/huella_estado_restaurable_v3.sql`
- `pruebas_sql/inventario_global_restaurado_v3.sql`
- `pruebas_sql/acl_inventario_v3.sql`
- `pruebas_sql/recuperacion_reinicio_v3.sql`
- `pruebas_sql/lectura_exacta_replay_v3.sql`
- `pruebas_sql/lectura_evidencia_replay_v3.sql`
- trabajo `puerta-postgresql-restauracion-bolsa-v3` de la integración continua

Ejecución:

```bash
./deploy/postgresql/bolsa_baremacion/probar_copia_restauracion_v3.sh
```

## Garantías y límites

T12-A acredita, para esta vertical y este corte:

- restaurabilidad lógica tras perder el servidor;
- conservación exacta de historia y material probatorio;
- restauración de propietarios, roles `NOLOGIN` y dos membresías mínimas;
- ausencia de credenciales dentro del artefacto de roles;
- conservación de RLS, ACL y superficies cerradas;
- reproducibilidad automática en integración continua.

No acredita todavía:

- copia productiva cifrada ni custodia de sus claves;
- archivado continuo de WAL o recuperación a un instante concreto;
- réplica, conmutación, segundo emplazamiento o copia desconectada;
- anclaje monotónico externo capaz de detectar una copia antigua íntegra;
- copia y recuperación coordinadas del almacén S3/Ceph;
- RPO/RTO aprobados, alertas, capacidad o simulacro organizativo;
- restauración de todas las verticales de Bolsa;
- conformidad formal ENS, RGPD/LOPDGDD, ENI, DPD o Sistemas.

La suma SHA-256 contigua al volcado detecta corrupción accidental, pero no
protege frente a quien pueda sustituir simultáneamente copia e inventario. El
anclaje externo y la separación administrativa siguen siendo obligatorios.

## Siguiente incremento

Sistemas debe aprobar uno de los conectores de copia continua —pgBackRest o
WAL-G, sin acoplar el núcleo a la herramienta— y proporcionar repositorio
cifrado, claves externas, retención, red y supervisión. Después se automatizará
una recuperación PITR aislada y coordinada con objetos.

En aplicación continúan T12-B y T13: registro durable y consultable de todos
los accesos/efectos con finalidad, checkpoints de firma y anclaje externo. No
se habilitará un piloto con datos personales hasta cerrar estas condiciones y
obtener las validaciones formales.
