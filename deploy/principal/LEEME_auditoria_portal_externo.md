# Material de auditoría del portal externo

Tarea F0-28. El registro de intentos del aspirante ya leía
`auditoria-intentos-externa.json`, pero la separación de portales clasificaba
ese nombre como interno. El cambio lo reconoce como externo por su nombre
exacto. Los demás archivos conservan su clasificación.

En el material privado del portal externo, el JSON tiene esta estructura:

```json
{
  "esquema": "vec.auditoria.intentos.servidor.v1",
  "dsn_file": "externo/auditoria-intentos.dsn",
  "proceso": "vec-portal-externo",
  "canal": "externa_personal",
  "limite_segundos": 10
}
```

El proceso debe coincidir con la configuración del LOGIN nominal en la copia.
La conexión va en `externo/auditoria-intentos.dsn`, con permisos `0600` dentro
del material `0700`, separado del material interno. La cuenta y su admisión se
preparan con el productor correspondiente; este cambio no las crea.

El JSON interno, otros JSON o conexiones en la raíz y las claves privadas de
cliente o de la autoridad certificadora siguen rechazados en el portal
externo. El portal interno rechaza el JSON y la conexión externos.

Para activar, el productor de la copia coloca esos dos archivos antes del
reinicio vigilado de su aplicación. El lector conserva sus comprobaciones de
esquema, ruta local, canal, proceso, TLS y LOGIN exclusivo. El archivo ausente
no habilita el registro de auditoría. No hay migraciones ni cambios de permisos.

Prueba focal: `go test -p 6 ./internal/app/separacionportales`. El recorrido
completo del aspirante requiere además identidad, contexto, confianza y los
datos de inscripción de su copia; esta corrección cubre la clasificación.
