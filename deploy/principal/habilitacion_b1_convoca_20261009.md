# Encender la carga CONVOCA de Bolsa

Este procedimiento corresponde a la carga desde RRHH de una bolsa ya aprobada. Dirección ejecuta la instalación y los actos en cidonia. Los ensayos de esta rama usan una copia PostgreSQL 18 con identidades sintéticas; ninguna prueba local instala SQL en cidonia.

## Orden de instalación

1. Comparar con la historia de cidonia las migraciones HZ10, AD219/AUT58 y BIC7/B79/B80/B95/B93. Instalar solo las ausentes, una vez, en el orden del paquete B1 conservado por dirección. Los archivos de #886 se instalan después: AUT59, AD220, AUT60 y AUT61. No ejecutar `DOWN` ni repetir un `UP` instalado.
2. Instalar la lista `deploy/principal/lista_sql_b1_versionar_20261009.txt`: AUT62, AD227, AUT63 y AUT64, en ese orden. AD228, si se despliega la inscripción de aspirantes, va después de AD227 y conserva su preimagen exacta.
3. Confirmar que el ejecutor B1 es un LOGIN exclusivo del grupo `vec_admin_version_rol_bolsa_ejecutor`. Solo puede ejecutar las dos fachadas B1 y `resolver_rol_administrable_v1`. El ejecutor de #886 conserva otro LOGIN y su ACL exacta.

Antes de ejecutar el mantenimiento AUT59, contar las concesiones del rol ADMIN v7 y sus descriptores nominales. En la copia HZ10 del ensayo había 20 y 13: AUT59 rechazó correctamente esa diferencia. La rama dependiente de ratificación instala AD230 y AUT65, y solo un plan externo aprobado puede añadir desde ese acto los siete descriptores ausentes. Su ejecución no cambia el rol ni las asignaciones. Después de comprobar 20/20 se prepara un plan AUT59 nuevo con la preimagen actual; el plan anterior ya no sirve. Si la base ya tiene cobertura completa, no se ejecuta la ratificación.

## Autoridad que debe aprobarse

El mantenimiento AUT64 publica `administracion_perfiles` v9 con dos acciones nuevas: `administracion.perfiles.version_bolsa.proponer` y `.aprobar`. Requiere plan versión 5, aprobación externa ligada a la huella del plan, configuración DBA vigente y el LOGIN técnico exclusivo de `vec_admin_mantenimiento_version_bolsa_ejecutor`. `vec-mantener-admin-fijo` ejecuta ese plan con la variante 5 y conserva las dos `AsignacionID` ADMIN. Un recibo incierto se reconcilia con la misma operación y sus bytes originales antes de otro intento.

Después del mantenimiento se prepara el catálogo AUT58/AUT61 con el censo completo actualizado a ADMIN v9. La fuente versionada de Bolsa es `deploy/principal/fuentes/bolsa_carga_convoca_b1_v1.json`, con SHA de entradas canónicas `a81a99cd4be193f720222ef8d659ce199727790db793bcb7365f8a7b47bae089`. La entrada está vigente del 1 de octubre de 2026 al 1 de octubre de 2027; el catálogo permite cambiar esa ventana mediante una nueva versión aprobada. Contiene una sola acción nueva: `bolsa.carga_convoca.confirmar`, módulo `bolsa`, recurso `carga_convoca`, finalidad `carga_bolsa_convoca`, garantía `alto`, sin campos ni obligaciones. El paquete conserva todos los roles del censo y clasifica el RolID RRHH vigente como `administrable`, sin modificar sus concesiones. La aprobación externa del paquete y la configuración DBA ligan las huellas exactas del plan, fuente, catálogo y censo. `vec-catalogo-acciones-admitir` aplica el plan aprobado y permite consultar el mismo catálogo ante resultado incierto.

Dos personas ADMIN de Aplicación distintas abren sesiones con certificado vigente. La primera presenta `/api/admin/perfiles/v1/gobierno-version-bolsa/propuestas` con el catálogo admitido, la base máxima del RolID RRHH y las asignaciones seleccionadas como preimágenes CAS. Los documentos de asignación enviados son expectativas: AUT63 coteja íntegramente documento y SHA con su tabla, bajo bloqueo. La segunda aprueba por `/api/admin/perfiles/v1/gobierno-version-bolsa/cierres` con la referencia y huella de la propuesta. El cierre publica una versión nueva del mismo RolID y avanza solo las asignaciones aprobadas, conservando cada `AsignacionID`, perfil, persona, ámbitos y concesiones anteriores. La auditoría común, historia, outbox y recibo se confirman en la misma transacción.

CONFIG NUEVA: `VEC_ADMIN_VERSION_BOLSA_CONFIG_FILE=/etc/vec/administracion/version-bolsa.json`. El fichero privado, modo `0600`, contiene los dos pools segregados, dos materiales V3 y los motivos publicados para las audiencias `vec_autorizacion.versionar_rol_bolsa.propuesta.v1` y `.cierre.v1`. La ausencia de este fichero mantiene las rutas cerradas.

## Comprobación tras el encendido

Reiniciar la aplicación después del acto de cierre. En RRHH abrir `/portal-empleado/modulos/bolsa/carga-convoca/`, elegir un resumen CONVOCA de personas sintéticas y una categoría RPT, revisar la vista previa y confirmar una sola vez. Recuperar con la misma referencia, sin crear otra carga. Comprobar el mismo recibo, fecha, acta y participación tras reiniciar aplicación y PostgreSQL. Una fila aceptada se enlaza con Mi Bolsa solo si coincide con la identidad acreditada; las filas ambiguas quedan pendientes de revisión.

Antes del despliegue faltan las aprobaciones reales de mantenimiento, catálogo y dos ADMIN, y la comprobación de la historia exacta de cidonia. Los resultados de ensayo local se incorporan a esta página al cerrar la rama.
