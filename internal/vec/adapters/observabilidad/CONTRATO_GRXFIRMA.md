# Resultado técnico del validador GrxFirma

El componente cerrado es `domain.ComponenteIncidenciaGrxFirma` (`grxfirma`).
Identifica el validador remoto configurado, sin atribuirle una identidad de usuario.
La etapa de la llamada es `domain.EtapaIncidenciaPeticion` (`peticion`).

El cliente de E recibe por constructor el puerto común ya existente:

```go
ports.EmisorResultadosTecnicosConContexto
// EmitirResultadoConContexto(context.Context, domain.SolicitudResultadoTecnico)
```

Tras conocer la respuesta real, E puede aportar esta solicitud:

```go
domain.SolicitudResultadoTecnico{
    Resultado:  domain.ResultadoTecnicoNoDisponible,
    Componente: domain.ComponenteIncidenciaGrxFirma,
    Etapa:      domain.EtapaIncidenciaPeticion,
}
```

`no_disponible` se declara solo tras un HTTP 5xx del origen autorizado. Un
dictamen positivo confirmado puede declarar `ResultadoTecnicoCorrecto` cuando
el contexto de la llamada sigue vivo. E mantiene estas condiciones en su cliente;
el puerto común valida los códigos y conserva la correlación, sin interpretar HTTP.

Cancelación, error TLS, credenciales rechazadas, JSON inválido, fallo de red y
timeout quedan fuera de esta declaración. No se incluyen URL, cuerpo, error de
biblioteca, documento, actor ni referencia de expediente. El emisor extrae del
contexto la correlación interna existente; si falta, cuenta la pérdida y no crea
otra que aparente pertenecer a la llamada. El fallo del destino técnico no cambia
el dictamen ni provoca un reintento de negocio.

El catálogo de incidencias pasa a revisión 2 al ampliar un componente, como exige
su contrato vigente. Conserva los 13 códigos y las 13 plantillas de cada idioma.
Solo cambia `version_catalogo` a la cadena canónica `"2"` en los archivos de textos. Emisor y recolector deben
usar esa misma revisión; un catálogo de otra revisión se rechaza antes de escribir.

La revisión del catálogo no forma parte de los campos JSONL. El recolector nuevo
acepta las líneas históricas con los códigos, componentes y textos anteriores,
sin atribuirles una revisión. Las líneas emitidas con el formato canónico anterior
conservan los mismos bytes; una presentación JSON distinta se reconstruye con la
lista blanca existente. Si alguien añade a una línea un campo `version_catalogo`,
se rechaza como campo ajeno. Un archivo de catálogo de revisión 1 se rechaza; no
se convierte implícitamente a revisión 2. La metadata externa también debe usar
una cadena canónica (`"1"` o `"2"` según la revisión); un número JSON se rechaza.
Este formato nuevo todavía no está instalado.

Los esquemas JSONL siguen siendo `vec.incidencia_tecnica.v1` y
`vec.resultado_tecnico.v1`, con los mismos campos. La incidencia genérica
`HTTP_INTERNO_FALLIDO` admite también `grxfirma/peticion`; no es necesaria para
emitir el resultado observado. Alias como `GrxFirma`, `validador_firma` o `remoto`
siguen fuera del catálogo. Este corte no monta el cliente de E, ni acredita una
llamada HTTP, una firma o un registro nominal de auditoría.
