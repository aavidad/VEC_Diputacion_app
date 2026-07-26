# O4-05: migración de la capacidad de consulta RRHH a VEC-AD-3

## Propósito

Este corte sustituye la autorización nominal antigua de las proyecciones de
Contratación temporal por una concesión breve VEC-AD-3. La decisión, su
confirmación durable y las diez piezas probatorias originales permanecen
ligadas hasta el futuro consumo autoritativo en PostgreSQL.

El cambio afecta al cuadro operativo de RRHH y al detalle de un expediente.
No crea todavía el consumidor PostgreSQL, la ruta HTTP ni la identidad
productiva; por ello no eleva por sí solo el porcentaje oficial de O4-05.

## Frontera nominal cerrada

Cuadro y detalle comparten la infraestructura criptográfica, pero no autoridad:

| Consulta | Acción | Audiencia | Recurso | Ligadura funcional |
|---|---|---|---|---|
| Cuadro RRHH | `contratacion_temporal.cuadro.consultar` | `vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1` | `cuadro_rrhh_contratacion_temporal` | ámbito, filtros, límite y cursor |
| Detalle | `contratacion_temporal.expediente.consultar` | `vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1` | `expediente_contratacion_temporal` | expediente y versión observada |

No se admiten comodines ni conversiones entre ambas concesiones. El recurso
del cuadro se refiere al ámbito autorizado; el recurso del detalle se refiere
al expediente concreto.

## Cadena de confianza

`ContextoConsultaRRHH` solo se construye desde
`ContextoAutorizacionAltaV3`. Su inicio efectivo es el máximo entre la
resolución autoritativa, la autenticación verificada, la emisión y
revalidación de sesión y el comienzo de vigencia de los vínculos. Su final es
el menor de las expiraciones correspondientes.

`MaterialAutorizacionConsultaRRHH` coteja:

1. solicitud, decisión concedida y confirmación durable;
2. autenticación, sesión, actor, perfil y registro de contexto;
3. acción, finalidad, audiencia y tipo de recurso nominales;
4. decisión, motivo, efecto y contexto contra el resumen tipado;
5. los cánones originales contra las diez piezas exportadas;
6. la ventana temporal máxima de cinco segundos.

El resumen Go es defensivo y no autoritativo. PostgreSQL deberá verificar y
consumir las piezas originales en la misma transacción que la lectura y el
registro de acceso.

## Orden antes que material

Una incidencia de revisión detectó que una primera versión dejaba exportar el
material SQL antes de ligarlo a la consulta funcional exacta. Esa superficie
se rechazó.

La exportación para PostgreSQL solo está disponible desde
`OrdenConsultaCuadroRRHH` y `OrdenConsultaDetalleRRHH`. Cada orden recalcula la
huella canónica y revalida contexto, capacidad, acción, finalidad, consulta y
vigencia antes de entregar una copia defensiva. Ni el material ni la capacidad
aislados exponen la exportación SQL.

## Minimización y trazabilidad

- Las solicitudes, capacidades, órdenes, material y recibos sensibles bloquean
  serialización y representaciones accidentales.
- Los DTO públicos conservan únicamente las referencias opacas y datos
  operativos imprescindibles.
- El recibo de lectura liga decisión, capacidad, conjunto probatorio, consulta,
  correlación, sesión, organización, ámbito, acción, finalidad y versión.
- Los resultados de cuadro y detalle se vuelven a validar contra la orden
  exacta antes de abandonar la aplicación.
- Denegado, ausente y ajeno no forman un oráculo de existencia.

## Pruebas exigidas

El candidato solo puede integrarse cuando superen:

- contratos y aplicación de Contratación temporal;
- ejecución con detector de carreras;
- análisis estático y formato;
- cruces hostiles de audiencia, recurso, acción, finalidad, ámbito, consulta,
  expediente, versión y vigencia;
- prueba de superficie que impida exportar material SQL sin una orden;
- serialización, registro y formato redactados;
- revisión independiente sin editar el producto.

## Próximo corte

El siguiente trabajo es un adaptador PostgreSQL nominal para cuadro y detalle.
Debe usar funciones SQL separadas y roles de mínimo privilegio, verificar las
diez piezas VEC-AD-3 y realizar en una sola transacción:

1. consumo único de la capacidad, con repetición rechazada;
2. lectura de la proyección autorizada;
3. registro durable del acceso;
4. emisión de un recibo ligado a la misma consulta.

La función existente para confirmar altas está cerrada a su propia operación y
audiencia. No se reutiliza ni se generaliza para estas consultas.
