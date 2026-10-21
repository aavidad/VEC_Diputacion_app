# F1-122: comprobar la vuelta a Bolsa tras el cese

El paso 20 de `docs/producto/FLUJO_APP.md` usa una vía nativa ya montada.
Falta comprobarla con el cese actual del caso que recorre P y repetir las
consultas tras reiniciar. Este documento prepara esa comprobación; no acredita
un retorno ejecutado.

CONFIG NUEVA: ninguna. SQL a instalar: ninguno adicional a la lista de main.
No crear otra copia para este tramo ni modificar la copia de P sin su cesión.

## Vía que se reutiliza

1. `internal/app/bootstrap/cese_personal_b2_desarrollo.go` envuelve el cese de
   Contratación. Tras obtener su recibo, consulta el origen de incorporación.
   Solo el origen `personal_b2_v1` continúa por Personal.
2. `internal/app/incorporacionejercicio/cese_personal_b2.go` cierra la relación
   y la ocupación originales. Repetir el mismo cese recupera ambos recibos
   mediante sus claves de idempotencia, también si se perdió una respuesta.
3. `internal/app/bootstrap/alta_cese_bolsa_nativa_b115.go`, `confirmarCese`,
   relee el anclaje de la aceptación y entrega a Bolsa el selector original,
   el recibo del cese y los dos recibos de Personal.
4. `internal/modules/bolsa/application/cese_nativo_ct.go` usa la autoridad
   existente de cambios de situación. Su repositorio llama a
   `vec_bolsa_llamamientos.aplicar_cese_nativo_ct_v1` (B115). Esta función
   comprueba el cese publicado mediante CT213, ambos cierres de Personal y el
   alta nativa previa; guarda contrato de cese, restricción, auditoría y recibo.

La situación visible se proyecta con la política de Bolsa y la fecha de
consulta. El registro histórico «trabajando» no se borra. Un cese futuro no
adelanta el retorno y una pausa u otra restricción posterior se conserva.
La ausencia de un vínculo antiguo en CT201 no demuestra un fallo de esta vía.
El relevo histórico de ceses es otro camino; no debe sustituir el cese nativo.

Si falla el tramo posterior al cese de Contratación, la operación queda
pendiente con respuesta 503. Se recupera con la misma clave y el mismo
contenido del primer intento; no se registra otro cese para continuar.

## Comprobación con P

Continuar el caso único desde sus pasos 17 → 18 → 16a → 16b → 19, con la
configuración de Personal ya preparada por la fábrica y la regla
`personal_b2.cese_fecha_efecto` vigente. No reaplicar CT213/B115 ni preparar
otra autoridad. Los permisos siguen siendo los de las operaciones existentes.

1. Antes del cese, conservar fuera de Git expediente, participación y bolsa,
   recibo de aceptación, alta nativa y recibos de relación y ocupación.
   Consultar la misma participación desde Bolsas de RRHH y Mi Bolsa de su
   titular. Guardar situación, fecha y orden, cuando esté disponible.
2. Desde la ficha del expediente, registrar el cese con su causa, fecha y
   justificante. Conservar el cuerpo y la clave del primer intento, y su
   recibo. Si queda pendiente, recuperar esa operación antes de continuar.
3. Comprobar en Personal los cierres de la relación y la ocupación del alta.
   En Bolsa, comprobar un único cese nativo ligado a esos recibos y la versión
   de política que determina `disponible_desde`.
4. Volver a consultar la misma participación en ambas pantallas. Deben
   mostrar Disponible o Disponible desde según la política y el corte temporal,
   sin duplicar la participación ni perder su posición de acta. No cambiar la
   política para obtener una etiqueta concreta.
5. P reinicia aplicación y PostgreSQL con la fábrica de su copia. Repetir las
   consultas y comparar recibos, fechas, participación e historia. Si se
   recupera el cese, usar la operación original: los recibos deben coincidir
   y no debe aparecer otro contrato o restricción para ese evento.

Las capturas y los recibos del caso se conservan en el material privado de P.
La entrega en Git solo recogerá el resultado, la revisión ensayada y sus
límites, sin credenciales ni datos personales.

## Pruebas focales existentes

```sh
go test -p 6 ./internal/app/incorporacionejercicio -run TestCesePersonalB2 -count=1
go test -p 6 ./internal/app/bootstrap -run 'Test(EfectoBolsaNativoTrasPersonal|CesePersonalB2|IncorporacionB2GETConfirmadaTrasCese)' -count=1
go test -p 6 ./internal/modules/bolsa/application -run 'TestProyeccion(EstadoCese|Cese)' -count=1
```

Las tres pruebas focales pasaron el 21/10/2026 sobre `origin/main@c47f8df16`,
con cambios solo de documentación. Cubren recuperación de los cierres, entrega del mismo material
a Bolsa y proyección temporal. No sustituyen el cese y las consultas en Chrome
del caso único. La prueba SQL existente
`deploy/postgresql/bolsa_llamamientos/pruebas_sql/b115_contrato_cese_b2.sql`
comprueba el contrato instalado; tampoco acredita por sí sola un retorno.
