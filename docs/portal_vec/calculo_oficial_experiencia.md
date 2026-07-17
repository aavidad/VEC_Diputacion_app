# Cálculo oficial de experiencia

## Estado y alcance

| Campo | Valor |
| --- | --- |
| Fecha de corte | 17 de julio de 2026. |
| Estado | Contrato de integración en implementación. |
| Alcance | Convertir el resultado puro del motor de experiencia en un efecto administrativo durable, autorizado, reproducible y auditable. |
| No incluido | Revisión humana de documentos, composición completa del autobaremo y publicación de listas. |
| Política | Denegación por defecto, dos autorizaciones independientes y ausencia de resultados parciales. |

El motor puro calcula y explica. Este caso de uso añade las garantías que
permiten conservar ese cálculo como resultado oficial. No interpreta las bases,
no modifica reglas y no decide si una evidencia documental debe admitirse.

## Secuencia obligatoria

```text
sesión confiable
  -> autorización V2 de lectura
  -> fuente exacta y consumida
  -> reglas activas vinculadas a la convocatoria
  -> compilación y cálculo puro
  -> bytes y huella canónicos
  -> clave e intención semánticas
  -> autorización V2 nueva de escritura
  -> confirmación PostgreSQL serializable
  -> recibo cotejado
```

La autorización de lectura se obtiene antes de revelar reglas o tramos. La de
escritura se solicita después de conocer la huella del resultado. Deben usar
decisiones y correlaciones diferentes. Ninguna puede sustituir a la otra ni
reutilizarse entre intentos.

Se ofrecerán composiciones distintas para el área personal externa y el portal
interno de RRHH. La primera exige el perfil ordinario sobre superficie externa;
la segunda, garantía alta sobre superficie corporativa o de administración.
El cliente no elige el perfil.

## Fuente exacta y cálculo

La fuente liga por referencia, versión y huella:

- estado gobernado de las reglas;
- convocatoria;
- sujeto pseudonimizado;
- instantánea de entrada;
- contenido de la entrada;
- evidencia, verificador y consumos durables de lectura.

El caso oficial solo admite `EstadoReglasBaremoActiva` y exige que
`ConvocatoriaActivacion()` coincida exactamente con el selector. La compilación
permanece separada porque una simulación sí puede utilizar un borrador. Tanto
la simulación como el cálculo oficial invocan el mismo
`CalcularExperienciaV1`; el modo de uso no altera los bytes del resultado.

Un resultado bloqueado por negocio se puede conservar como resultado oficial
explicable. Un fallo técnico devuelve el valor cero y nunca llega al
confirmador.

## Idempotencia semántica

Se separan dos materiales canónicos:

1. `ClaveEfectoV1`, que identifica qué cálculo se pidió.
2. `IntencionResultadoV1`, que liga esa clave al resultado obtenido.

La clave incluye sujeto pseudonimizado, convocatoria, estado exacto de reglas,
instantánea y huella de entrada, contrato y versión del motor, huella del plan,
causa gobernada y, solo al rectificar, el resultado predecesor. Excluye actor,
sesión, autorizaciones, correlaciones, reloj, auditoría y referencias de
transacción.

Se conserva SHA-256 del material canónico y se deriva el índice interno con:

```text
HMAC-SHA-256(clave del servidor, ClaveEfectoV1 canónica)
```

La clave HMAC nunca sale del servidor ni se almacena junto al código. El mismo
efecto con el mismo resultado devuelve el recibo original; el mismo efecto con
otra huella de resultado se rechaza como no determinismo. La comparación de la
huella pública impide aceptar una colisión simulada del índice HMAC.

## Confirmación durable

La primera implementación exige que fuente, autorización y resultados puedan
revalidarse dentro de la misma instancia PostgreSQL. Una transacción
`SERIALIZABLE`, sin llamadas remotas, debe:

1. revalidar y bloquear la autorización V2 de escritura;
2. revalidar la fuente, la autorización de lectura y sus consumos;
3. comprobar que las reglas continúan activas para la convocatoria exacta;
4. restaurar los bytes canónicos y comprobar su huella;
5. recalcular el índice HMAC y buscar el efecto existente;
6. insertar, en un efecto nuevo, resultado, vínculos, intención y recibo;
7. consumir la autorización de escritura;
8. registrar auditoría encadenable e intento nominal;
9. insertar un único evento mínimo en el outbox;
10. confirmar conjuntamente todos los efectos.

Una repetición idéntica consume y audita la nueva autorización de escritura,
pero no duplica resultado ni evento. Si se pierde la respuesta después del
`COMMIT`, la referencia nominal del intento permite recuperar el recibo tras
reiniciar, sin ejecutar de nuevo el efecto.

Los bytes canónicos se guardan inicialmente en `bytea` para no romper la
atomicidad. El almacenamiento de objetos podrá recibir una réplica posterior
mediante outbox; no participa en la decisión transaccional.

## Privacidad y trazabilidad

Resultado, intención, auditoría, outbox, errores y logs no contendrán DNI,
nombre, correo, dirección, causas médicas, texto laboral libre ni atributos
completos del tramo. Las trazas técnicas se limitan a referencias opacas,
huellas, códigos, contadores y duración.

El outbox conserva solamente referencias, huellas, estado y fase. Leer la
explicación completa requiere otro caso de uso autorizado y queda registrado.
Los roles de aplicación no pueden actualizar ni borrar resultados, auditoría o
eventos históricos.

## Pruebas de aceptación

Antes del uso real deben acreditarse, como mínimo:

- dos decisiones V2 distintas y denegación al reutilizar una;
- rechazo de reglas no activas o vinculadas a otra convocatoria;
- identidad exacta entre resultado recalculado, bytes, huella e intención;
- una sola fila y un solo evento ante peticiones concurrentes;
- rollback completo al fallar cualquiera de los pasos;
- recuperación tras respuesta perdida y reinicio;
- rechazo de colisión HMAC y de resultado distinto para el mismo efecto;
- RLS, privilegios mínimos e inmutabilidad comprobados en PostgreSQL;
- pruebas de ausencia de datos personales en bytes auxiliares y logs;
- separación real entre composición externa ordinaria e interna de garantía
  alta.

## Trabajo pendiente

El motor puro ya está implementado. Para completar este corte faltan integrar
el dominio de identidad semántica, el servicio de aplicación, el adaptador
PostgreSQL, la fuente durable de reglas e instantáneas, la composición y las
pruebas de extremo a extremo. Ninguna pantalla se declarará operativa mientras
alguno de esos elementos siga sustituido por datos sintéticos.
