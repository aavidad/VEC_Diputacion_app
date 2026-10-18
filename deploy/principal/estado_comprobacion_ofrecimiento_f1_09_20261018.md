# Estado de una comprobación del ofrecimiento

F1-09. Amplía la lectura de origen CT ya autorizada y auditada. La selección
consume el resultado en su respuesta409 existente; la pantalla de F1-57 podrá
usarlo para registrar o rectificar los hechos. No añade rutas, acciones ni
lectores independientes y conserva el orden y la elegibilidad actuales.

## Activación

1. Instalar el primer corte F1-09 (CT233, AD290 y B133) y la bitácora B134.
2. Respetar la cadena de dirección: AD288 antes de B137→AD293; después irá
   F0-18/AD291. No repetir migraciones instaladas ni ejecutar DOWN.
3. B137 reemplaza la firma de tres argumentos de la lectura contextual por
   la misma función con un cuarto contexto opcional. Retira la firma anterior
   mediante DROP RESTRICT; las llamadas de tres argumentos conservan su salida.
4. Instalar el binario con la configuración existente de CT y Bolsa. La orden
   interna añade nueve datos del expediente fiscalizado; SQL los revalida con
   CT233 antes de consultar el estado. No hay configuración nueva de arranque.
5. Reiniciar con vigilancia y ejecutar un ofrecimiento pendiente, una lectura
   sin política de criterios y la recuperación de una reserva anterior.

Lista propia: `lista_sql_codexr_estado_comprobacion_f1_09_20261018.txt`.
Añadir sus dos filas a la lista única de main al llegar su turno en la cadena.
Las preimágenes proceden de los cuerpos instalados por B118, CT233 y AD276.
Las concesiones, el apunte de auditoría y los permisos de ejecución se conservan.
El ejecutor no obtiene lectura directa de los hechos. La reserva histórica
mantiene su instantánea; no se completa con el estado de hoy.

## Contrato de la respuesta

El409 conserva `comprobacion_ofrecimiento.contexto` (16 campos). Añade `estado`
a partir de la misma entrada del origen:

- Campo ausente: la lectura no aporta el estado; no equivale a ausencia de acto.
- `{"version_esperada":0}`: ausencia comprobada para organización, expediente,
  necesidad, bolsa y participación exactos.
- Versión positiva: última comprobación, resultado, motivo, versión de política,
  fecha y recibo, junto a `vigente`. El estado conserva su versión cuando
  `vigente` es falso; una rectificación no vuelve a empezar desde cero.

La vigencia compara el contexto completo, incluidas versiones de expediente,
bolsa, instantánea y política, huella, modalidad, causa, ámbito y unidad.
No se entregan hechos, actor, contactos ni documentos. La pantalla no debe
interpretar un acto desactualizado como autorización para continuar. Esa
comprobación corresponde a F1-10 y este corte no cambia la selección.

Las definiciones explícitas, permisos y huellas SQL se comprobaron en una
transacción con ROLLBACK sobre la copia propia de las 77 migraciones. Después
se confirmó que seguía la firma antigua y no quedó instalada la ampliación.
Esto comprueba el DDL y sus guardas; no acredita una lectura positiva de actos.

Las focales Go cubren compatibilidad histórica, ausencia y acto desactualizado,
coherencia del contexto y minimización. El ensayo SQL sobre la cabeza final,
la configuración conjunta y el recorrido en Chrome siguen pendientes del turno
de la cadena; la preparación de fuente no acredita su instalación.

Vuelta atrás: recuperar el binario anterior y su configuración. Mantener las
migraciones y la historia; no borrar actos ni recibos.
