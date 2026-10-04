# Personal33 — una unidad inicial desde una fuente sintética

Personal33 publica una única unidad de desarrollo. Requiere un plan y un
documento de fuente coincidentes, una aprobación externa fijada por el DBA,
un LOGIN técnico exclusivo y una preimagen vacía exacta. No publica Personas,
cuentas, perfiles, permisos, RPT ni actos jurídicos. No activa OH11.

La nueva función de entrada es:

```text
vec_personal.inicializar_unidad_sintetica_admin_v1(
  plan_canonico text, sha_aprobado text, fuente_canonica text
) -> jsonb
```

La cuenta sólo ejecuta esa fachada. Carece de acceso a las tablas, funciones
privadas y configuración. La CLI prepara y aplica por el mismo contrato; no
concede permisos ni registra su propia aprobación. El DBA conserva fuera de
Git la configuración positiva de LOGIN, SHA del plan, fuente y preimagen,
aprobación y ventana. No hay una configuración favorable sembrada en SQL.

## Plan y documento de fuente

Orden cerrado del plan V1, compatible con `encoding/json` de Go:

```text
version = 1
operacion_ref = pui_<22..124 caracteres opacos>
preparado_en, caduca_en = instantes UTC con segundos
entorno = desarrollo
alcance_fuente = sintetico_declarado
unidad {
  nodo_ref, organizacion_ref, unidad_ref, clase, denominacion,
  catalogo_ref, catalogo_version, catalogo_revision, catalogo_entrada_clave,
  revision_esperada, vigente_desde, vigente_hasta
}
acto_tecnico_ref
fuente {referencia, version = 1, huella_sha256}
```

Orden cerrado de la fuente V1, en otro documento:

```text
version = 1
referencia
entorno = desarrollo
alcance_fuente = sintetico_declarado
unidad = mismo objeto del plan
acto_tecnico_ref = misma referencia técnica del plan
```

La fuente no incluye su propio SHA. `fuente.huella_sha256` en el plan compromete
los bytes UTF8 exactos del canon de la fuente. El canon respeta el orden de los
DTO, rechaza campos ajenos y conserva los textos de la fuente; no los recorta ni
normaliza. SHA256 usa funciones nativas de PostgreSQL18, sin ampliar las ACL
de `public` ni recibir HMAC, claves o certificados.

`nodo_ref` es un UUID canónico en minúsculas. Organización y unidad respetan
`^[a-z][a-z0-9_:-]{2,127}$`. La referencia de fuente conserva ese mismo máximo
128 para concordar con AD176. Entrada de catálogo conserva la cota ASCII160
de Personal10. El acto técnico usa `acto_tecnico:[a-z0-9_:-]{1,147}`: máximo160;
el prefijo distingue una referencia técnica y no acredita una firma, delegación
ni aprobación jurídica. La denominación tiene 1..300 caracteres Unicode, sin
controles ni blancos al principio/final. Esos blancos se rechazan, no se borran.

El catálogo conserva `estructura-organizativa-dipgra` y las clases instaladas:
`delegacion`, `centro`, `puesto_responsabilidad`. Versión y revisión iniciales
de catálogo son uno; `revision_esperada` es cero y el nodo publicado tiene
revisión uno, padre nulo y `retirado=false`. Las fechas son días civiles:
desde no supera la preparación, hasta es exclusiva y cubre la caducidad del
plan. `conocido_desde` lo fija el publicador con su reloj real.

## Integridad y auditoría

La preimagen privada `preimagen_unidad_inicial_admin_v1(jsonb)` no concede nada.
Retiene la fila de generación de Personal31 y exige ausencia total por nodo,
unidad y clave de catálogo, incluso ante historia retirada. Su SHA se obtiene
de `convert_to(preimagen::text,'UTF8')`; el DBA fija ese compromiso antes del
intento de publicación.

El publicador hace un INSERT normal en `org_nodo_historia`. El trigger instalado
de Personal31 avanza la generación en la misma transacción. No se desactiva,
recrea ni sustituye. La confirmación privada es un singleton global: ni otro
LOGIN con otra configuración aprobada puede utilizar esta función para una
segunda alta. Sólo la misma operación, plan, fuente, LOGIN y aprobación
recuperan el recibo original, sin una fila nueva.

Los recibos y la configuración son de solo adición, con RLS forzada y permisos
exclusivos del propietario. Personal no consulta tablas CA, AUT o de otros
módulos. La organización del documento es una referencia externa aprobada;
AUT38 comprobará después CA y la unidad mediante sus fachadas propietarias.

AD176 registra la confirmación de la acción propia
`inicializar_unidad_sintetica_admin_v1`, módulo `personal`, finalidad
`inicializar_unidad_sintetica_admin` y proceso observado `postgresql`.
`recurso_ref=unidad:<UUID real publicado>`. Distingue el SHA de la fuente del
SHA del recibo base: este último se calcula antes de añadir las coordenadas de
auditoría, sin ciclo de huellas. El alcance sintético permanece explícito.

Cada invocación gestionada añade además un intento común: éxito, replay,
denegación o error. Sólo conserva LOGIN, acción, recurso opaco, finalidad,
resultado, proceso/canal, fecha y SHA de la entrada; no los documentos recibidos.
El efecto va en una subtransacción. Si falla, revierte sus filas, generación y
confirmación y se conserva únicamente el intento. Si falla el append común,
se aborta todo. Ningún error expone referencias revertidas o texto del motor.

El sobre tiene cinco claves: `estado`, `codigo`, `recibo`, `replay` y
`auditoria_intento`. El consumidor confiable hace COMMIT antes de comunicarlo.
Esta cobertura no acredita fallos de conexión/ACL previos, cancelaciones o un
ROLLBACK directo del cliente. El recibo de replay es histórico; no revive una
unidad retirada después. Personal31 comprobará la revisión actual al consumirla.

## Validación y siguiente paso

Orden causal: Personal10 y Personal31 instaladas; AD176 tras AD174; después
Personal33. La lista de esta rama sólo contiene Personal33. No se reaplican
las SQL previas ni se ejecuta DOWN sobre su historia.

Pruebas preparadas, siempre en el clon y con ROLLBACK:

- `unidad_inicial_admin_000033_preimagen.sql`: canon, fuente, preparación sin
  efectos, RLS, tablas privadas y ACL de la fachada.
- `unidad_inicial_admin_000033_operador.sql`: conexión real del operador aprobado,
  alta, recibo, replay sin otra confirmación, intento nuevo por llamada y rechazos
  de aprobación, fuente cambiada, producción, campo ajeno y denominación con blancos.

Dirección carga por parámetros enlazados los GUC de sesión
`vec.ensayo.plan_unidad_inicial` y `vec.ensayo.fuente_unidad_inicial`. Los
scripts no siembran aprobaciones ni permisos. El segundo requiere el LOGIN real
configurado; SET ROLE o un superusuario no sustituyen esa comprobación.

Quedan para el clon coordinado dos revisiones exactas, positivo durable,
recuperación tras reinicio, rechazo de una segunda operación/LOGIN, ausencia
de rescate de historia retirada y fallo del append sin efectos. La mera
presencia de esta migración no acredita la unidad ni el arranque 2+1. Tras
confirmar esta fuente, se vuelve al bootstrap AUT36/37/38 con un plan nuevo y
sus huellas/aprobación propias; no se elimina la guarda de Personal31.
