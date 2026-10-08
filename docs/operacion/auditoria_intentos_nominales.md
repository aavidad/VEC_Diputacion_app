# Auditoría nominal de denegaciones y errores

K y E pueden registrar el resultado observado de una operación fallida después de
que su repositorio haya terminado la transacción de negocio. El registrador abre
una transacción independiente y devuelve el acuse cuando confirma COMMIT.
El registro queda en `vec_autorizacion_atestada_v3.auditoria_consumo_v3`, con su
cadena común. No crea una decisión ni consume una autorización de negocio.

El puerto es `ports.RegistradorIntentosAuditoria.AppendIntentoAuditoria(ctx,
orden) (ports.AcuseIntentoAuditoria, error)`. La orden se construye mediante
`ports.NuevaOrdenIntentoAuditoria(intentoRef, resultadoContexto, vinculo, datos)`.
`resultadoContexto` es el contexto registrado V2 y `vinculo` procede de la
frontera autenticada del servidor. Referencias, perfiles y canal no se toman
libremente del navegador. `AppendAudit` genérico permanece cerrado.

## Uso desde un consumidor

1. Crear una referencia opaca de intento en el servidor con
   `ports.NuevaReferenciaIntentoAuditoria()` y conservarla para los reintentos.
2. Esperar al retorno del repositorio: la transacción original debe estar cerrada.
3. Construir la orden con identidad y perfil históricos, acción, módulo, recurso
   opaco, finalidad, motivo de catálogo, resultado `denegado` o `error`, proceso,
   canal y correlación común de la petición.
4. Invocar el registrador con su pool dedicado. Comprobar su error y conservar el
   acuse; una llamada iniciada o una fila leída antes de COMMIT no es un acuse.

Misma referencia y mismo material recuperan el recibo original. Una referencia
repetida con otro material se rechaza. La correlación puede enlazar varios hechos
y no sirve por sí sola como clave de idempotencia. El acuse identifica el registro,
su secuencia, huella, correlación y fecha conservadas.

Una cancelación HTTP no cancela la auditoría: el adaptador usa un plazo propio
configurado. Si falló el COMMIT original sin poder conocer su resultado, la orden
registra el error observado; no afirma que el efecto de negocio no exista.
Si falla el registro, el consumidor no comunica que el intento quedó auditado.

## Identidad histórica y acceso técnico

Las fachadas propietarias CA26 e IS13 cotejan la historia exacta de contexto y
autenticación. Una revocación posterior no borra la identidad del intento y el
cotejo no renueva permisos. No se fabrican asignaciones o versiones de rol que
no hayan sido seleccionadas. El arranque anterior al primer perfil necesita su
propia evidencia auténtica y no queda cubierto por esta orden posperfil.

El LOGIN técnico pertenece únicamente al rol registrador y tiene una configuración
DBA explícita de proceso y canal. Sin ella se deniega. El LOGIN no recibe lectura
directa de tablas, permisos de negocio ni facultad de cambiar esa configuración.
El constructor del adaptador recibe proceso, canal y plazo desde configuración
privada. Su preflight debe pasar antes de conectar un consumidor.

## Instalación y comprobación

Orden causal: roles técnicos, CA26, IS13 y AD169, según la lista SQL del corte.
La provisión del LOGIN y de proceso/canal es externa a Git. No reaplicar SQL
instalado ni ejecutar DOWN sobre historia conservada.

Ensayo final en copia fría sintética PostgreSQL 18.4: denegado y error después de
rollback, mismo acuse en replay, rechazo de material distinto y de LOGIN sin
configuración, y negativa de fuentes históricas que no coincidieron en el tiempo.
Las 6.240 filas anteriores conservaron sus huellas. La llamada Go→PostgreSQL y
su replay después de reiniciar PostgreSQL mantuvieron recibo, fecha y una sola
fila; la cadena final tuvo 6.243 registros. Se recomputaron los tres materiales
y eslabones nuevos sin divergencias. El clon y sus datos se retiraron.

La prueba `TestIntegracionIntentoAuditoriaPostgreSQL` usa las variables
`VEC_INTENTOS_AUDITORIA_FIXTURE` y `VEC_INTENTOS_AUDITORIA_DSN` para el material
histórico sintético privado y el LOGIN dedicado. `VEC_INTENTOS_AUDITORIA_REF`
permite repetir la misma orden tras reinicio. Sin esas variables se omite: las
pruebas unitarias no sustituyen este ensayo. El fixture se extrae de la pareja
conservada `atestacion_decision_v3.contexto_actor_canonico` y
`decision_canonica.vinculo_autenticacion_actor`, cotejada con CA e IS; no debe
contener claves, HMAC ni datos reales, y no se guarda en Git.

El motivo es una referencia de catálogo resuelta por el consumidor confiable.
SQL conserva esa referencia como metadato del intento; no publica su catálogo
ni concede permisos por ella. Cada consumidor debe usar su catálogo gobernado.

Las revisiones exactas se anotan en la PR y el canal de coordinación.
Este puerto permite a los consumidores ampliar su cobertura; no significa que
todos los módulos ya lo llamen. El sello TSA y la exportación judicial pertenecen
a los cortes siguientes de auditoría. El verificador anterior de consumos debe
ampliarse para este nuevo tipo de registro antes de verificar un rango mixto.
