# Preparación de Carrera

H08 conserva un servicio de consulta nominal preparado y un consumidor de interfaz.
Es WIP para composición: todavía no hay lector operativo de Personal, ruta HTTP,
perfil de Carrera provisionado ni acto nominal instalado por esta entrega.

La raíz debe entregar una consulta `ConsultaAntecedentesCarreraV1` de Personal,
con actor, empleado objetivo, organismo y corte resueltos por el servidor, junto
a una orden de concesión y su confirmación V3 comunes. El servicio comprueba el
actor, el recurso completo, la acción, la finalidad, los campos y la ventana.
Personal conserva su autorización y su transacción de lectura y auditoría propias.
No se emplea un booleano para conceder acceso ni se crea un emisor nuevo.

La interfaz recibe una función `consultar` autorizada y el traductor común.
Muestra antecedentes, cobertura, certeza, recibo y pendientes; cancela respuestas
antiguas y permite volver a consultar. No tiene transporte propio ni carga un
escenario sintético cuando falta la fuente. La ausencia de dependencias devuelve
indisponibilidad sin datos. Solo una denegación central expresa registrada se
presenta como acceso denegado.

La preparación mantiene pendientes grupo/subgrupo, grado y política competente.
No infiere grado desde nivel, suma servicios, transforma periodos de Personal,
registra solicitudes ni reconoce derechos. No instala SQL, permisos, relaciones,
ocupaciones o actos de toma de posesión. H07 y H11 permanecen intactos.

Pendiente para componer el recorrido nominal:

- B confirma e implementa el lector `LectorAntecedentesCarreraV1`, su finalidad,
  campos, resolución de objetivo/organismo y recibo transaccional.
- La raíz acuerda y gobierna el perfil fijo de Carrera; esta declaración no
  provisiona permisos ni permite pedirlos desde el navegador.
- Dirección compone la fuente y el servicio reales, acuerda la frontera HTTP y
  monta la interfaz. Después se prueba el recorrido autorizado y sus denegaciones.

La evidencia local cubre pruebas unitarias de Go con dobles solo de prueba,
Go/race/vet/gosec, Semgrep, i18n y 23 pruebas Node. Backend y usabilidad recibieron
revisiones independientes sobre `06d09a450457012677b73c7c2a1127a1a91ad395`.
No acredita navegador, datos reales, autorización instalada ni persistencia nominal.
