# F1-30: cese recibido antes de vincular al candidato

Comprobación del 20 de octubre de 2026 sobre `main@64a546ed2`, en una
copia local propia de PostgreSQL 18.4. No se ha cambiado el código de producto
ni se ha instalado SQL nueva. Se reutiliza la solución de #774: CT197, B81,
B90 y la reconciliación de ceses de `internal/app/bootstrap/`.

## Qué se ha comprobado

| Paso | Resultado observado |
| --- | --- |
| Recibir un cese de una participación constituida que aún no tiene candidato | La función nominal conserva una proyección con motivo `vinculo_candidato_pendiente`. El cursor alcanza la posición 15320 y la referencia del cese. |
| Crear después el vínculo con el aspirante de la copia | La consulta autenticada de «Mi Bolsa» responde 200. Sus dos participaciones figuran como `no_disponible`, con `fecha_disponible=null`. |
| Abrir «Mi Bolsa» en Chrome | «No disponible», fecha de recepción del cese y sin fecha de fin registrada. Capturas a 1440 y 390 px; el ancho de la página coincide con el de la ventana. |
| Reiniciar la aplicación y PostgreSQL con el relevo de ceses apagado | Se conservan exactamente las dos participaciones, sus estados y fechas. |
| Encender el relevo existente y reiniciar | El trabajador recupera el pendiente anterior al cursor y registra una restricción. La proyección original permanece. |
| Reiniciar de nuevo con el relevo encendido | Una proyección y una restricción; mismo recibo, fechas y contenido de ambas. La consulta de «Mi Bolsa» sigue respondiendo 200. |

El estado pendiente nace de la recepción del cese en Bolsa, no del momento en
que se añade el vínculo. La fecha observada antes y después del primer
reinicio es `2026-10-20T06:47:47.161498Z`.

La consulta se ha hecho con el certificado del aspirante y la autorización
publicada por el preparador existente de la copia. No se han introducido
permisos por SQL para leer «Mi Bolsa». Chrome no usa cookies ni almacenamiento
local o de sesión como credencial; las tres colecciones estaban vacías.

## Preparación y repetición

1. Crear una copia con módulos y RRHH original mediante
   `deploy/principal/copia_local.py`, siguiendo `LEEME_copia_local.md` y la
   lista única `deploy/principal/lista_sql_main_tras_hz37.txt`. Usar puertos y
   material propios. La creación de esta copia tardó 97,65 segundos.
2. Ejecutar las pruebas ya existentes, antes de preparar el caso persistente:

   ```sh
   docker exec -i -u postgres "$CONTENEDOR_PROPIO" \
     psql -X -v ON_ERROR_STOP=1 -U postgres -d postgres \
     < deploy/postgresql/bolsa_llamamientos/pruebas_sql/b81_cese_sin_candidato.sql
   docker exec -i -u postgres "$CONTENEDOR_PROPIO" \
     psql -X -v ON_ERROR_STOP=1 -U postgres -d postgres \
     < deploy/postgresql/bolsa_llamamientos/pruebas_sql/b90_cese_pendiente_lecturas.sql
   ```

   Ambas terminan en `ROLLBACK`. Comprueban también la pertenencia al acta,
   la huella divergente, el cursor parcial, la repetición, la denegación a
   otro rol y las lecturas con corte histórico.
3. Para observar la pausa en pantalla, reutilizar en un guion privado los
   datos de B81: incorporación histórica, cese, recepción en Bolsa y
   constitución, inicialmente sin `vinculo_candidato`. Confirmar el cese con
   la función nominal `confirmar_cese_sin_candidato_bolsa_v1` y conservar los
   datos en esta copia desechable. Mantener apagado el relevo durante la
   observación del pendiente.
4. Añadir el vínculo de esa participación con la candidatura de Antonio que
   prepara la copia. El acta y la instantánea deben corresponder a la
   participación. Abrir `/area-personal/?vista=llamamientos&lang=es` con su
   certificado. Cotejar también `GET /api/vec/bolsa/mi-bolsa` y guardar las
   participaciones de la respuesta, sin comparar `consultada_en`.
5. Reiniciar con `copia_local.py reiniciar`, repetir la consulta y comparar
   estados y fechas. No emitir un segundo cese.
6. Preparar un LOGIN propio y exclusivo para el grupo
   `vec_bolsa_llamamientos_relevo_cese`, con TLS verificado y vigencia de
   90 días. Configurar `VEC_BOLSA_CESE_CT_DATABASE_URL` y
   `VEC_BOLSA_CESE_CT_ENABLED=true` conforme a `03_entorno.md`, actualizar
   la huella del entorno privado y reiniciar. No reutilizar otra conexión.
7. Comprobar que la proyección conserva su recibo y que aparece una sola
   restricción; reiniciar y comparar los registros completos. El trabajador
   usa `listar_ceses_sin_candidato_pendientes_v1`, por lo que recupera el
   vínculo aunque la publicación ya haya quedado detrás del cursor.

Los guiones y capturas de esta ejecución quedan fuera de Git, en el material
privado del equipo. No contienen cambios de producto para fabricar datos.

## Alcance y límites

La preparación persistente reutiliza el contrato histórico de la prueba B81:
incorporación de 2027 y cese de 2030. Conserva ese periodo para respetar sus
comprobaciones de coherencia. Se adapta únicamente la categoría y el vínculo
al aspirante de la copia. También se reutiliza el ajuste de política de la
prueba B81 para su modalidad histórica, exclusivamente como dato de la copia.
La restricción calculada vence el 1 de junio de 2030. Tras resolver el
pendiente, «Mi Bolsa» muestra «Disponible» en el corte actual, anterior a ese
cese; no se presenta como una vuelta tras un cese ya efectivo.

Este ensayo cubre la recuperación de #774 para su puente histórico y la
lectura nominal del pendiente en pantalla. No acredita los pasos 19 a 21 del
caso único 4.0 con un cese actual, ni la vuelta de un nombramiento creado
completamente desde pantalla. Tampoco atribuye al usuario el cese preparado
por la prueba. La continuidad tras un fallo de otra fila se comprueba con
las focales existentes de Go, no con dos ceses nuevos desde Chrome.

La copia tiene apagadas las rutas opcionales de preferencias y de imagen del
área personal: se observaron respuestas 404. Chrome registró además un error
TLS al intentar cargar el trabajador de caché. La consulta y la presentación
del estado de Bolsa funcionaron; no se afirma una consola sin errores ni se
valida en este corte la pantalla completa del aspirante.

## Validación y activación

- Pruebas SQL B81 y B90: correctas sobre la copia actual.
- `go test ./internal/app/bootstrap -run 'TestReconciliacionB81|Test.*B81' -count=1`,
  con `GOFLAGS=-p=6`: correcto. Incluye continuación después de un fallo y
  recuperación de pendientes anteriores al cursor.
- Chrome del sistema, con Playwright: consulta 200, escritorio y móvil,
  recuperación del pendiente y de la restricción tras reiniciar.
- CONFIG NUEVA: ninguna. Para el ensayo se enciende el relevo existente;
  apagado y encendido, la aplicación arranca y consulta Bolsa y Contratación.
- SQL nueva a instalar: ninguna. No reaplicar CT197, B81 ni B90.

La decisión de cerrar F1-30 y actualizar el tablero corresponde a dirección.
