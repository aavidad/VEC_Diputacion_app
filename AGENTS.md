# Instrucciones del repositorio para agentes

## Prioridad vigente

La prioridad es el módulo `contrataciontemporal`, basado en el procedimiento
remitido por RRHH. Bolsa no se borra: mantiene convocatorias, candidaturas,
posiciones, reglas y llamamientos.

Si una tarea de contratación temporal necesita una capacidad común de VEC,
Bolsa, Personal, documentos o firma:

1. se define una tarea dependiente y acotada;
2. se implementa en el módulo que posee esa autoridad;
3. se prueba e integra;
4. se vuelve inmediatamente al camino crítico de contratación temporal.

No se amplía otro módulo por conveniencia ni se cambia la prioridad sin
instrucción de dirección.

## Lectura obligatoria

Antes de editar:

1. instrucciones de mayor prioridad del entorno;
2. `AGENTS.md`;
3. `docs/portal_vec/relevo_sesion_2026-07-29_inicio_ct47.md`;
4. `docs/portal_vec/mapa_objetivos_tareas_y_paralelizacion_2026-07-23.md`;
5. `docs/portal_vec/tablero_tareas_contratacion_temporal_2026-07-23.md`;
6. `docs/portal_vec/relevo_contratacion_temporal_2026-07-23.md`;
7. la especificación y matriz normativa enlazadas desde esos documentos.

El relevo fechado identifica la rama integradora y los worktrees vigentes. El
directorio raíz conserva una rama histórica y no es el lugar de desarrollo.
El relevo anterior del 28 de julio es histórico y no sustituye al corte
vigente del 29 de julio.

Un agente recibe un identificador de tarea. No toma otra por iniciativa propia.

## Trabajo paralelo

- Rama y worktree exclusivos fuera de `/tmp`.
- Write-set declarado y disjunto.
- Un agente productor no integra ni da el visto bueno a su propio trabajo.
- El revisor reproduce pruebas y emite `GO` o `NO-GO`.
- Solo dirección modifica los documentos transversales de estado durante la
  integración.
- No fusionar, rebasar ni limpiar ramas ajenas.
- Commits pequeños con el identificador funcional, pruebas y documentación.

## Granularidad obligatoria

Todo VEP se desarrolla mediante minitareas, también fuera de contratación
temporal:

- una minitarea tiene una sola responsabilidad observable y un único criterio
  de cierre;
- un agente recibe una sola minitarea y no amplía su alcance por iniciativa
  propia;
- código y pruebas forman un commit local autónomo; la evidencia de revisión y
  el estado transversal se confirman inmediatamente después por integración;
- como señal de alarma, una tarea que modifica más de tres ficheros de
  producción, añade más de unas doscientas líneas productivas o necesita dos
  motivos distintos en el mensaje se divide o justifica antes de programarse;
- cada commit debe compilar y superar sus pruebas focales; no se admiten cortes
  intermedios que dejen una API pública incompleta, una migración sin consumidor
  coherente o una ruta registrada sin autoridad;
- una capacidad grande se expresa como un grafo de minitareas con dependencias,
  no como un identificador que acumula contratos, adaptadores, composición,
  interfaz y E2E;
- las piezas independientes se asignan a subagentes con write-sets disjuntos;
  otro agente revisa el resultado antes de que dirección lo integre;
- si dos minitareas necesitan el mismo fichero, se ejecutan en secuencia o se
  separa primero una abstracción propietaria; nunca se resuelve permitiendo
  edición concurrente del mismo archivo.

Dividir no significa producir commits rotos o cambios cosméticos sin valor. La
unidad mínima es una capacidad compilable, protegida y verificable de extremo a
extremo dentro de su alcance.

## Arquitectura no negociable

- Hexagonal estricta.
- `domain` no importa aplicación, puertos, adaptadores, HTTP, SQL ni
  proveedores.
- `application` coordina dominio y puertos; no conoce HTTP, PostgreSQL, Docker
  ni SDK concretos.
- `ports` contiene contratos mínimos y neutrales; no implementaciones.
- `adapters` traduce proveedores y transportes sin redefinir reglas de negocio.
- Ningún módulo lee o escribe tablas de otro módulo.
- Los intercambios usan referencias opacas, comandos, eventos e
  outbox/inbox.
- El dominio y los casos de uso son neutrales al cliente. Web, escritorio,
  CLI y MCP consumen las mismas capacidades de aplicación mediante
  adaptadores distintos; ninguna regla funcional vive en HTTP, DOM o una
  sesión de navegador.
- Base de datos, almacenamiento, firma, antivirus, identidad, comunicaciones,
  documentos, calendarios y sistemas externos son adaptadores intercambiables.
- Fases, opciones, plantillas, formatos y reglas funcionales se gobiernan por
  catálogos versionados. Solo las invariantes técnicas quedan compiladas.
- No crear una segunda autoridad de identidad, roles, permisos, auditoría,
  i18n o temas.

## Seguridad y protección de datos

- Denegación predeterminada y privilegio mínimo.
- La identidad y el perfil proceden de una frontera confiable; nunca de JSON,
  cookies, parámetros, cabeceras libres o datos del navegador.
- Cookies, `localStorage` y `sessionStorage` no son autoridades ni mecanismos
  de sesión. La composición real no depende de ellos. Los clientes usan
  credenciales breves, ligadas al emisor y verificadas por la API; escritorio
  puede emplear certificado/mTLS y Kerberos corporativo mediante conectores.
- La API rechaza por defecto `Cookie`, credenciales persistidas por el
  navegador y cualquier cabecera de identidad que no esté atestada por la
  frontera confiable. Ninguna respuesta real emite `Set-Cookie`.
- Operaciones internas sensibles exigen garantía alta y superficie interna o
  administrativa coherente.
- Una decisión positiva queda ligada a acción, recurso, finalidad, ámbito,
  motivo, actor, perfil, correlación y vigencia exactos.
- Todo efecto consume la autorización dentro de la misma transacción que
  escribe estado, auditoría y outbox.
- Idempotencia semántica, control optimista de versión e historia de solo
  adición.
- Ningún secreto, certificado, clave, token, DSN, dato personal real o ruta
  privada entra en Git, fixtures, errores o logs.
- Datos minimizados y referencias opacas en listados, auditoría y eventos.
- Cifrado en tránsito y reposo; claves fuera del proceso y rotables.
- Límites de tamaño, tiempo, profundidad y cardinalidad antes de reservar
  memoria o llamar conectores.
- La indisponibilidad nunca se interpreta como autorización, validación,
  firma, entrega o éxito.
- No usar datos reales hasta cerrar EIPD, categorización ENS, análisis de
  riesgos, registro de actividades y autorizaciones formales.
- IA aplicada a selección, empleo, baremación, evaluación o asignación queda
  cerrada hasta completar la clasificación y obligaciones del Reglamento de
  IA. El bot público solo accede a corpus público gobernado.

La matriz vigente del módulo es
`docs/portal_vec/matriz_normativa_contratacion_temporal_2026-07-23.md`.

## i18n, idioma y presentación

- Código de dominio en castellano coherente, salvo términos técnicos
  universalmente adoptados y convenciones de Go.
- No mezclar castellano e inglés en el mismo vocabulario.
- Todo texto visible, mensaje de validación, estado, ayuda, documento y
  notificación usa claves i18n.
- Fechas, números, moneda, zonas horarias y plurales se formatean por
  localización.
- El tema común es la autoridad visual; ningún módulo duplica CSS estructural.
- Accesibilidad desde diseño: teclado, foco, contraste, zoom, lector de
  pantalla, estados alternativos y documentos descargables accesibles.

## Calidad

- Código documentado cuando la intención o el contrato no sean obvios.
- Sin adaptadores ficticios en la composición real.
- No declarar E2E, producción o cumplimiento por tener una pantalla o una
  prueba aislada.
- Objetivo de 500 líneas y tope duro de 800 por fichero conforme a DEC-051.
- Sin funciones monolíticas, estados globales mutables ni errores que filtren
  detalles internos.
- Contextos y cancelación en todas las fronteras lentas.
- Copias defensivas al cruzar límites mutables.
- Dependencias externas solo si están mantenidas, son compatibles en licencia
  y reducen riesgo real.

Puertas mínimas:

```text
gofmt
go test del paquete afectado
go test -race del paquete afectado
go vet del paquete afectado
git diff --check
```

Antes de integrar se añaden `go test ./...`, `go vet ./...` y
`scripts/verificar_calidad.sh` cuando el alcance lo permita. PostgreSQL se
prueba en instancia efímera real con roles, ACL, concurrencia, reintento y
reversión protegida. HTTP/web exige contrato, seguridad, accesibilidad y
revisión visual.

## Entrega

```text
Tarea:
Estado: GO / NO-GO / bloqueada
Commit(s):
Archivos modificados:
Resultado:
Pruebas ejecutadas:
Pruebas omitidas y motivo:
Seguridad, privacidad, i18n y accesibilidad:
Limitaciones:
Riesgos:
Siguiente tarea desbloqueada:
Revisión independiente:
```

La documentación, pruebas y código se entregan juntos. El agente no cambia una
tarea a cerrada ni actualiza el porcentaje; lo hace dirección tras verificar e
integrar el commit.
