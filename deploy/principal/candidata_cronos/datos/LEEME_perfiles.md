# Personas ficticias para el recorrido de Cronos

La copia de ensayo de Cronos contiene tres personas existentes y distintas. Sus perfiles nuevos permiten recorrer C7 y preparar C5a. La semilla se aplicó una sola vez, en una transacción, sobre la copia local del 14/10/2026. No es una asignación aprobada por Administración ni acredita una relación de empleo real.

| Papel en el recorrido | Certificado de desarrollo reutilizado | Competencia en Cronos |
| --- | --- | --- |
| Empleado | `centro-solicitante` | Consulta y actos propios. Su asignación tiene un único ámbito positivo: su empleado. |
| Jefatura | `centro-ratificador` | Consulta y resuelve el paso `responsable` de ese empleado. La asignación liga su persona y ese paso. |
| RRHH | `cliente` | Consulta y resuelve el paso `administracion` del mismo empleado. La asignación liga su persona y ese paso. |

Los tres certificados proceden de la autoridad de desarrollo ya instalada y superaban 60 días de vigencia al preparar la copia. La identidad de cada certificado, la cuenta, el perfil y la persona están ligados por la huella y el registro de sesiones. Se añadieron dos cuentas ordinarias con alias HMAC; la tercera ya existía. Los tres perfiles y sus vínculos de contexto son nuevos. Personal conserva una proyección previa y recibió dos proyecciones nuevas, además de tres relaciones y tres ocupaciones ficticias con plaza y puesto vigentes de la misma unidad. Esas filas llevan fuente y huella propias, vigencia del 14/10/2026 al 30/04/2027, `firma_oficial=false` y `eficacia_administrativa=false`.

Cronos recibió dos asignaciones positivas en `permiso_resolutor`: una para jefatura y otra para RRHH, ambas limitadas al empleado del recorrido. El empleado carece de competencia para resolver su propia solicitud. Las concesiones de la jefatura y de RRHH se limitan a consultar la bandeja y resolver en su paso. El perfil del empleado incluye la consulta del recibo de corrección y sus avisos. Los perfiles anteriores y su historia siguen disponibles.

## Activación en esta copia

1. Conservar el material privado de la copia y el fichero `identidad/cronos-empleado.json` de esta preparación. Contiene tres cuentas. La semilla SQL, su fuente y la selección de referencias se custodian fuera de Git; su SHA-256 es `e75b848c56c2472cb9d057c4547097c493fbe55bbeb9e59d68a4c3a6490071de`. No volver a aplicarla a la misma base.
2. Instalar las migraciones pendientes en el orden de la candidata. El catálogo técnico de motivos de Cronos debe estar publicado antes del arranque. El fichero privado usa la versión 1 de ese catálogo para lecturas, actuaciones y denegaciones con entradas distintas; ninguna equivale a aprobación normativa.
3. Partir del entorno privado de arranque de la copia y añadir los tres selectores: `VEC_CRONOS_EMPLEADO_ENABLED=true`, `VEC_CRONOS_RESOLUCION_ENABLED=true` y `VEC_CRONOS_NOTIFICACIONES_ENABLED=false`. El directorio indicado por `VEC_DEVELOPMENT_MATERIAL_DIR` debe ser el que conserva esas identidades y la configuración de Cronos. Añadir `cronos` al valor vigente de `VEC_PORTAL_MODULOS_VISIBLES`.
4. Dirección comprueba arranque, Chrome y autorización con los tres certificados. La solicitud y su recibo nacen del POST real; la recuperación C5a consulta ese mismo recibo. No se insertan solicitudes, decisiones, avisos ni recibos en la semilla.

La comprobación de datos dio, para cada papel, una cuenta, un alias, un vínculo de contexto, una proyección de empleado, una relación, una ocupación y una asignación de perfil vigentes. Hay una competencia vigente por cada paso y ninguna para el empleado sobre sí mismo. Falta acreditar con la aplicación el recorrido, las denegaciones por datos ajenos y la recuperación tras reiniciar. Personal 000026 y el consumidor de C5a seguían pendientes cuando se preparó esta semilla.
