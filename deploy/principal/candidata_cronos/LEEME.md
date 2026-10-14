# Candidata de Cronos

Encargo del 14 de octubre de 2026. Se prepara sobre una copia desechable de
HZ37, con los datos ficticios del respaldo. Dirección revisa, integra y decide
su incorporación a una candidata completa. Esta carpeta no autoriza una
instalación en cidonia.

## Código que se reutiliza

- Los servicios de Cronos y sus adaptadores de identidad, autorización y
  PostgreSQL, montados en `internal/app/bootstrap/cronos_empleado.go`.
- Las vistas conectadas del portal: jornada, saldo, movimientos, calendario,
  solicitud de olvido, permisos, bandejas, avisos y comunicaciones a RRHH.
- La proyección histórica de Personal26, por su puerto; Cronos no consulta
  tablas de Personal.
- CRN15, ya integrada mediante las PR #831 y #877. No se vuelve a implementar
  la optimización del libro de saldo.

El inventario se hizo sobre `origin/main@a958227f2`. Los borradores de
correcciones, justificación y saldo con permisos se conservan como referencias;
no se presentan como SQL instalable. AD181 (#578) cerró sin fusionarse.

## Activación y ensayo

1. Restaurar el respaldo en una base y red propias con `restaurar_copia.sh`.
   Consultar `inventario_instalacion.md` antes de elegir cualquier SQL.
2. Fijar el commit candidato y las huellas de sus archivos. Completar
   `lista_sql.txt` con las migraciones pendientes, en orden causal, y ensayarlas
   una sola vez. No ejecutar DOWN ni reaplicar una migración instalada.
3. Preparar los ficheros privados de identidad, conexiones y motivos con los
   LOGIN nominales de la copia. Conservar la identidad de cada persona y la
   confianza gobernada. Ninguna clave, certificado o conexión entra en Git.
4. Publicar los perfiles y competencias mediante el circuito de administración
   existente. Empleado, jefatura y RRHH son personas distintas. Sus vigencias
   ficticias abarcan al menos sesenta días.
5. Añadir `cronos` al valor existente de `VEC_PORTAL_MODULOS_VISIBLES`. Activar
   `VEC_CRONOS_EMPLEADO_ENABLED`; resolución y comunicaciones se activan con
   sus selectores propios, cuando sus dependencias estén instaladas.
6. Comprobar el arranque con la configuración completa, con Cronos apagado y
   con su fichero privado ausente o inválido. En los dos últimos casos el
   resto de la aplicación debe seguir respondiendo y Cronos debe cerrar sus
   rutas sin acceder a datos.
7. Recorrer Chrome con las tres personas, comprobar denegaciones y conservar
   los recibos. Reiniciar la aplicación y PostgreSQL propios y recuperar las
   mismas operaciones, sin repetir los actos para fabricar un resultado.

Los ajustes de datos ficticios se hacen en esta copia. Los catálogos conservan
versión, fuente y vigencia; no se añaden reglas laborales como constantes.

## Estado de la entrega

El recorrido completo sigue pendiente. `resultado_recorrido.md` recoge solo
las comprobaciones ejecutadas. Las capacidades que faltan son la aprobación y
aplicación de correcciones, la justificación documental, el saldo con permisos
y Calendarios, las consultas de equipo y las exportaciones autorizadas.
