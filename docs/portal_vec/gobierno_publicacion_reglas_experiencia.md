# Gobierno y publicación de reglas de experiencia

## Estado de esta decisión

| Campo | Valor |
| --- | --- |
| Fecha de corte | 17 de julio de 2026. |
| Estado | Diseño aprobado para el siguiente corte vertical; implantación incompleta. |
| Alcance | Edición, simulación, aprobación, publicación, activación y retirada de las reglas de experiencia de una convocatoria. |
| Dependencia crítica | Motor exacto de experiencia V1 y composición productiva PostgreSQL. |
| Política | Denegación por defecto, instantáneas inmutables, autorización ligada a la solicitud y trazabilidad transaccional. |

Este documento distingue una configuración guardada de una regla jurídicamente
aplicable. Una pantalla administrativa, un borrador válido o una simulación
correcta no autorizan por sí solos a calcular una bolsa ni a mostrar el baremo
como vigente.

## Secuencia administrativa cerrada

La primera versión productiva seguirá esta secuencia:

```text
borrador completo
  -> validación y simulación reproducible
  -> aprobación firmada
  -> publicación de la versión de reglas
  -> publicación de la convocatoria que referencia esa versión exacta
  -> activación de las reglas para esa convocatoria exacta
  -> proyección pública conjunta
```

Una versión de reglas en estado `publicada` todavía no es visible al ciudadano.
Solo se proyecta cuando está `activa` y coincide por referencia, versión y
huella con la convocatoria publicada. La retirada oculta la proyección en la
misma transacción que cambia el estado; no depende de que un consumidor de
eventos funcione posteriormente.

Modificar una configuración crea otra instantánea completa. El objeto
gobernado no se convierte en un formulario mutable. Si RRHH necesita
autoguardado parcial, se usará un agregado no normativo
`ProyectoConfiguracionReglas`, separado de la versión que puede aprobarse.

## Casos de uso

El orden funcional previsto es:

1. consultar el índice autorizado de reglas de una convocatoria;
2. consultar una versión exacta;
3. validar una configuración tipada;
4. crear un borrador completo;
5. simular el borrador con casos patrón;
6. preparar y obtener la aprobación firmada;
7. publicar la versión de reglas;
8. publicar la convocatoria que la referencia;
9. activar la relación exacta convocatoria-reglas;
10. consultar públicamente el baremo;
11. sustituir, retirar o descartar versiones cuando corresponda.

El editor utilizará campos tipados, selectores de catálogos, unidades,
políticas de jornada, restos, redondeos y topes. No aceptará SQL, expresiones
libres ni código ejecutable. Una opción modelada pero no ejecutable por el
motor V1 puede conservarse como borrador, pero no publicarse.

## Identidad, autorización y separación de funciones

Cada lectura y cada escritura requieren decisiones de autorización distintas.
Una decisión concedida para consultar una fuente no sirve para registrar una
simulación ni para publicar. El servidor construye el actor, recurso, acción,
referencias, revisión, huellas y hora; no confía en roles ni actores declarados
por el cliente.

Acciones mínimas:

```text
bolsa.reglas_experiencia.indice.consultar
bolsa.reglas_experiencia.version.consultar
bolsa.reglas_experiencia.simulacion.fuente.consultar
bolsa.reglas_experiencia.borrador.crear
bolsa.reglas_experiencia.simulacion.registrar
bolsa.reglas_experiencia.aprobacion.preparar
bolsa.reglas_experiencia.version.publicar
bolsa.reglas_experiencia.version.activar
bolsa.reglas_experiencia.version.sustituir
bolsa.reglas_experiencia.version.retirar
bolsa.reglas_experiencia.version.descartar
```

La aprobación firmada acredita una decisión, pero no demuestra por sí sola la
separación de funciones. La política impedirá, cuando sea exigible, que quien
configuró sea también la única persona que apruebe y publique.

## Persistencia y transacción

La primera implantación usará PostgreSQL mediante puertos sustituibles. Como
mínimo separará:

- `vec_bolsa_reglas_baremo`, para gobierno, simulaciones, evidencias y estado
  interno;
- `vec_bolsa_publico`, para la proyección pública minimizada.

Las instantáneas canónicas se almacenarán como `bytea`, con huella verificada,
límites previos a la materialización, UTC a microsegundos, privilegios
revocados a `PUBLIC` y funciones nominales estrechas. Lectura pública, lectura
interna, escritura y migraciones tendrán roles técnicos diferentes.

Una escritura se ejecutará con aislamiento `SERIALIZABLE` y confirmará o
revertirá conjuntamente:

1. revalidación de la autorización ligada a la solicitud;
2. intención idempotente y actor;
3. aprobación y dependencias exactas vigentes;
4. comparación optimista de revisión y huella;
5. consumo único de las evidencias y de la decisión;
6. nueva instantánea y puntero monotónico;
7. relación activa con la convocatoria, cuando proceda;
8. proyección pública o su retirada inmediata;
9. auditoría encadenada;
10. evento en la bandeja transaccional;
11. recibo exacto de la operación.

La bandeja de eventos tendrá arrendamiento, reintentos y deduplicación. No es
la autoridad para decidir qué versión está públicamente vigente.

## Idempotencia y concurrencia

La clave enviada por el cliente no se conservará en claro. Se derivará mediante
HMAC incluyendo generación, principal y acción. La intención semántica fijará
operación, actor, referencias, huellas, estado esperado, motivo catalogado,
prueba de transición e instantes administrativos congelados.

- misma clave y misma intención: devuelve el recibo original;
- misma clave con otro contenido, actor o acción: conflicto;
- pérdida de respuesta tras `COMMIT`: repetición exacta sin duplicar auditoría
  ni eventos;
- dos publicaciones concurrentes: solo vence la que conserva la revisión y
  huella esperadas.

La revisión del dominio se complementa con comparación y sustitución atómicas
en PostgreSQL.

## Superficies previstas

El portal interno se sirve en composición y listener separados, sin cookies de
sesión, con identidad corporativa reforzada:

```text
GET  /api/interno/v1/bolsa/reglas-experiencia/indice
GET  /api/interno/v1/bolsa/reglas-experiencia/version
POST /api/interno/v1/bolsa/reglas-experiencia/validaciones
POST /api/interno/v1/bolsa/reglas-experiencia/borradores
POST /api/interno/v1/bolsa/reglas-experiencia/simulaciones
POST /api/interno/v1/bolsa/reglas-experiencia/aprobaciones
GET  /api/interno/v1/bolsa/reglas-experiencia/aprobaciones/{referencia}
POST /api/interno/v1/bolsa/reglas-experiencia/publicaciones
POST /api/interno/v1/bolsa/reglas-experiencia/activaciones
```

La consulta ciudadana será de solo lectura:

```text
GET|HEAD /api/publico/bolsa/convocatorias/{id}/baremo/experiencia
```

Antes de la activación y después de la retirada responde como no disponible.
La salida explica reglas, unidades, jornadas, redondeos y límites, pero no
expone actores, firmas, expediente, atestaciones, referencias internas ni
datos personales.

## Condiciones de aceptación

El recorrido no se considerará terminado hasta demostrar:

- borrador, simulación, firma, publicación, activación y consulta pública tras
  reiniciar procesos y PostgreSQL;
- repetición después de perder la respuesta sin efectos duplicados;
- resolución segura de publicaciones y activaciones concurrentes;
- rechazo de una decisión de lectura usada para escribir;
- rollback ante autorización usada, expirada o revocada;
- rechazo de firma incorrecta, revocada o vinculada a otra huella;
- separación de funciones efectiva;
- imposibilidad de publicar una configuración no soportada por el motor;
- atomicidad ante fallos inyectados en cada paso;
- invisibilidad pública antes de activar e inmediata al retirar;
- incapacidad del rol público para leer tablas internas;
- salida sin datos personales y comportamiento correcto de `HEAD`, métodos,
  rutas y límites.

## Limitación conocida de la primera iteración

Con el contrato actual, sustituir una regla activa y activar su sucesora no es
todavía una operación multiagregado atómica. La solución inicial segura es
ocultar o sustituir la anterior y activar después la nueva, aceptando un breve
intervalo sin baremo público. No se prometerá intercambio sin interrupción
hasta disponer del contrato transaccional que lo demuestre.
