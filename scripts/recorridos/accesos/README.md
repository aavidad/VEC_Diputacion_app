# Recorrido de accesos por perfil

Este guion comprueba, con Chrome y certificados distintos, una autorización positiva y una denegación por cada perfil: RRHH, centro solicitante, ratificador, Intervención y candidato/Área personal. Se detiene en el primer fallo. Solo admite lecturas `GET` y la consulta `POST` sin escritura del cuadro de Contratación.

## Condición para ejecutarlo

Preparar fuera de Git un clon local **H3–H5** con datos sintéticos, el binario correspondiente arrancado y un origen HTTPS loopback que sirva la interfaz y la API. La declaración de «clon H3–H5» corresponde al responsable del entorno; el guion verifica la huella SHA256 del binario y que el PID lo ejecuta, pero no instala ni certifica las migraciones. Se necesitan cinco identidades sintéticas con concesiones positivas exactas, cinco certificados y claves privados, y la CA del servidor confiada por Chrome. Cada comprobación positiva y negativa debe corresponder a una concesión publicada del clon y a un recurso sintético del ámbito del perfil. No usar el servidor de presentación ni la base principal.

El plan JSON se guarda con modo `0600` **fuera del repositorio**. El guion rechaza rutas privadas situadas dentro de Git. No imprime certificados, claves, cuerpos de respuesta ni datos de la persona. No desactiva la verificación TLS. El navegador solo solicita recursos del origen local declarado y corta cualquier respuesta de redirección antes de seguirla. Los mensajes de terminal están en `mensajes.es.json` y `mensajes.en.json`. Ejecución:

```bash
python3 scripts/recorridos/accesos/recorrer.py --plan /ruta/privada/accesos.json --ejecutar
```

Sin `--ejecutar`, o si falta cualquier condición, devuelve `NO EJECUTADO` y código 2. Un fallo real devuelve `PRIMER CORTE` y código 1. El código 0 confirma solo las sondas configuradas: no acredita un recorrido completo de RRHH, una firma, un efecto de escritura, persistencia ni publicación.

## Plan privado

Campos de nivel superior: `origen`, `evidencia`, `perfiles`. El origen debe ser `https://127.0.0.1:PUERTO`, `https://localhost:PUERTO` o `https://[::1]:PUERTO`. `evidencia` requiere `clon_h3_h5: true`, `binario_en_uso: true`, `binario` (ruta absoluta externa), `sha256_binario` (hexadecimal en minúsculas) y `pid` del proceso local. `perfiles` contiene exactamente `rrhh`, `centro`, `ratificador`, `intervencion` y `candidato_area`.

Cada perfil declara `certificado`, `clave` y `pruebas`. La clave debe tener modo `0600` o más estricto. Cada lista de pruebas incluye al menos un `permiso` de estado 200 y una `denegacion` de estado 401 o 403. Una prueba tiene `clase`, `metodo`, `ruta` y `estado`; la positiva añade `comprobacion` con un campo de `data` y `igual` o `tipo` (`objeto`, `lista`, `cadena`, `booleano`, `numero`). La negativa exige que la respuesta JSON no contenga datos en `data`. Un `POST` se admite únicamente para `/api/vec/contratacion-temporal/cuadro/consultas`, con este cuerpo exacto:

```json
{"filtros":{"texto":""},"paginacion":{"limite":10}}
```

La secuencia mínima a preparar en el clon es:

| Perfil | Permiso positivo a comprobar | Denegación a comprobar |
| --- | --- | --- |
| RRHH | Bandeja RRHH de peticiones ratificadas (`GET /api/vec/contratacion-temporal/peticiones-centro/rrhh`) | Contexto de centro, si la concesión RRHH no lo incluye |
| Centro | Contexto de petición (`GET /api/vec/contratacion-temporal/peticiones-centro/contexto`), con `data.actor.puede_presentar = true` | Bandeja RRHH de peticiones |
| Ratificador | Mismo contexto, con `data.actor.puede_ratificar = true` | Bandeja RRHH de peticiones |
| Intervención | Consulta propia de fiscalización que el clon publique y conceda expresamente | Bandeja RRHH de peticiones |
| Candidato/Área | Área propia (`GET /api/vec/bolsa/mi-bolsa`) | Bandeja RRHH de peticiones |

La tabla es una guía para configurar expectativas **después de comprobar las concesiones del clon**. No atribuye hoy permisos ni rutas de fiscalización a un entorno en ejecución. Para Intervención falta fijar la consulta positiva exacta del clon; si no existe, el plan permanece incompleto y el resultado es `NO EJECUTADO`. La mera carga de una página estática o de `/api/vec/session` no sustituye una comprobación funcional positiva.

## Prueba focal sintética

```bash
python3 -m unittest discover -s scripts/recorridos/accesos -p 'test_*.py'
```

Usa respuestas y ficheros temporales sintéticos. Comprueba la puerta sin ejecución, la matriz completa y la parada en el primer fallo. Una prueba adicional abre Chrome contra dos servidores loopback desechables: el primero devuelve `302` al segundo y verifica que el segundo no recibe ninguna solicitud. No contacta con VEC.
