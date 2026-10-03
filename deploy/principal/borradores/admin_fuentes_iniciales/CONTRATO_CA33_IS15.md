# Borrador de contrato CA33 / IS15 — fuentes iniciales ADMIN

La rama implementa CA33 e IS15 sobre los números reservados por Dirección.
Incluye vectores SQL para el clon sintético; la instalación, el ensayo conjunto
y las dos revisiones corresponden a Dirección. Ninguna migración se instala en
la principal por esta entrega. La aplicación y la auditoría común se cierran
junto con AUT39 y AD174, antes de presentar la pieza como usable.

## Alcance acordado

Crear fuentes iniciales sintéticas gobernadas en desarrollo mediante un operador
técnico exclusivo. AUT39 cotejará la configuración privada del operador, el plan,
la preimagen y sus SHA aprobados; la palabra `desarrollo` del JSON no concede
nada. Se rechaza producción; cidonia sintética queda para otro corte. CA conserva
Persona, Organización, procedencias y proyección/titularidad de cuentas sin
perfil. IS conserva cuentas, política y titularidad propia, reutilizando
`provisionar_cuenta_v1`. No se crean roles, asignaciones ni perfiles y no cambia
`control_continuidad_admin`. Los nombres civiles y CA32 quedan fuera.

Las cuentas se devuelven con referencias `cta_` reales generadas por IS. No se
fuerzan las referencias propuestas en el mapa. El recibo de fuentes permitirá
preparar después el plan de bootstrap con esas referencias. AUT38 será la
sucesora del bootstrap; AUT39 orquestará las fuentes iniciales en una sola
transacción, con auditoría común AD174 y cierre/recibo propios.

## Plan separado del bootstrap

K2 publicó los tipos en `b940094e0`,
`internal/vec/domain/plan_fuentes_iniciales_admin.go`. Orden y nombres acordados:

```text
version = 1
operacion_ref
preparado_en
caduca_en
entorno = desarrollo
alcance_fuente = sintetico_declarado
procedencia {referencia, version, huella_sha256}
organizacion {organizacion_ref, version_esperada, vigente_hasta}
personas[2] {
  persona_ref, version_esperada, vigente_hasta,
  operacion_cuenta_ordinaria_ref,
  operacion_cuenta_privilegiada_ref,
  fuente_titularidad {referencia, version, huella_sha256}
}
fuente_hmac {referencia, version, huella_sha256}
politica_admin {
  politica_ref, host_admin, ca_sha256, huella_aprobacion_sha256,
  maxima_edad_revocacion_segundos, vigente_hasta
}
```

Los instantes del canon del plan son UTC con precisión de segundos. Las
preimágenes iniciales de Persona y Organización esperan versión cero; las
fuentes nuevas declaran versión uno. No contiene material HMAC, referencias de
cuenta elegidas por el cliente, nombres civiles, actor, perfil ni rol.
`politica_admin.huella_aprobacion_sha256` es la aprobación de la política,
distinta de la aprobación externa del plan; no se introduce un ciclo de huellas.
La aprobación del plan y la preimagen aprobada se fijan en la configuración
privada de AUT39.

## ABI pactada con K1, K2 y K3

Funciones privadas, con `search_path` fijo, PUBLIC revocado y ejecución acotada:

```text
CA.preimagen_fuentes_iniciales_admin_v1(jsonb) -> jsonb
CA.confirmar_fuentes_iniciales_admin_v1(jsonb,jsonb,text,text,text,text) -> jsonb
  [plan, reciboIS, preSHA_CA, operacion_ref, plan_sha256, aprobacion_ref]

IS.preimagen_fuentes_iniciales_admin_v1(jsonb,text) -> jsonb
  [plan, material_hmac_canonico_privado]
IS.aplicar_fuentes_iniciales_admin_v1(jsonb,text,text,text,text,text) -> jsonb
  [plan, material_hmac_canonico_privado, preSHA_IS,
   operacion_ref, plan_sha256, aprobacion_ref]
IS.cotejar_recibo_fuentes_iniciales_admin_v1(text,text,text) -> jsonb
  [operacion_ref, plan_sha256, aprobacion_ref]
```

CA es `vec_contexto_actor_v1`; IS es `vec_identidad_sesiones_v1`. Preimagen y
publicación sólo conceden EXECUTE al propietario AUT. El cotejo del recibo IS
se concede también al propietario CA para verificar la evidencia real, en vez
de aceptar un recibo JSON del solicitante como prueba.

AUT39 prepara y bloquea ambas preimágenes antes de los efectos. IS aplica
primero, obtiene cuatro cuentas reales, registra política y su titularidad sin
perfil. CA coteja el recibo mediante la fachada IS y confirma Persona,
Organización, procedencias, proyecciones y titularidad sin perfil. Antes de
escribir CA comprueba la ausencia o correspondencia exacta de las referencias
reales recibidas. Una colisión o divergencia revierte íntegramente AUT39.

Los recibos tienen diez claves:

```text
esquema, version, recibo_ref, operacion_ref, plan_sha256,
aprobacion_ref, alcance_fuente, registrada_en, datos, huella_sha256
```

`version=1`, `alcance_fuente=sintetico_declarado` y `registrada_en` es el instante
real en UTC con microsegundos. La huella corresponde al documento JSONB privado
sin el campo `huella_sha256`. IS entrega referencias reales de cuentas y
política; CA entrega referencias reales de organización, Personas, proyecciones
y titularidad. No incluyen nombres civiles, bytes HMAC, claves ni evidencia
cifrada nueva. K3 agrega ambos recibos reales para AD174; el operador es
`session_user`, sin Persona, perfil o decisión V3 inventados.

## Fuentes existentes inspeccionadas

CA1 posee `procedencias`, `persona_versiones`, `persona_actual` y las
proyecciones de cuenta; CA3 posee `organizacion_versiones` y
`organizacion_actual`. La generación de punteros actual existente protege la
lectura MVCC y debe reutilizarse con su protocolo. Insertar el enum
`autoridad_maestra_acreditada` no demuestra una fuente institucional: el
registro nuevo debe conservar el alcance sintético y el acto inicial acreditado.

IS1/IS2 ofrecen:

```text
provisionar_cuenta_v1(
  operacion_ref, esquema_hmac, dominio_hmac_ref, clave_hmac_id,
  clave_hmac_version, cuenta_id_hmac bytea, sujeto_id_hmac bytea,
  cuenta_privilegiada boolean, cuenta_ordinaria_id_hmac bytea
) -> cuenta_ref
```

El productor genera la referencia real, exige coordenadas válidas y HMAC de
32 bytes y enlaza la privilegiada a una ordinaria activa con el mismo sujeto.
El material lo calcula el proveedor real con keybundle/KMS privado existente;
SQL no recibe una clave ni calcula un HMAC. La implementación existente de
aprovisionamiento usa metadatos privados de token PKCS#11 y su PIN, fuera de
Git. Ningún valor aleatorio sustituye esa fuente. El material privado tiene este formato cerrado:

```text
version = 1
fuente_ref, fuente_version = 1
esquema_hmac, dominio_hmac_ref, clave_hmac_id, clave_hmac_version
personas[2] {
  persona_ref,
  cuenta_ordinaria_id_hmac_hex,
  cuenta_privilegiada_id_hmac_hex,
  sujeto_id_hmac_hex
}
```

`fuente_ref` y `fuente_version` coinciden con el plan. La huella de `fuente_hmac`
es SHA256 de los bytes UTF8 exactos del documento canónico privado; el material
no incluye su propia huella. Los tres HMAC por Persona son 32 bytes en hex
minúsculo. Las seis coordenadas son distintas. Sólo el proveedor privado calcula
esos HMAC con su clave; SQL recibe el resultado. Ninguna clave llega a SQL.
La configuración privada de AUT39 coteja el material, la aprobación y el operador.
El material no se añade a recibos ni a la auditoría común.

IS9 posee `politica_certificado_admin_v1`; su límite de edad de revocación sólo
exige un intervalo positivo. No establece un máximo de 600 segundos. El contrato nuevo admite entre 1 y 2147483647 segundos como límite
técnico de representación; no es una regla legal ni un máximo de antigüedad
fijado por Sistemas. El valor concreto proviene del plan aprobado.

CA20 `crear_perfil_vinculo_admin_v1` exige hoy un vínculo contextual previo que
incluye perfil. CA33 introduce la titularidad sin perfil y adapta esa guarda conservando
el resto de precondiciones y la ABI de CA20. Antes exige la huella exacta del
cuerpo instalado. Nunca se siembra un perfil ficticio para satisfacerla. Las migraciones
instaladas se conservan.

## Validación y orden causal

Orden: IS15, CA33 y después AUT39 con AD174. IS15 publica un validador privado
del plan para CA/AUT y el cotejo de recibo para CA; no permite leer tablas ajenas.
Las fachadas requieren una transacción SERIALIZABLE de escritura y retienen las
barreras de continuidad/fuentes. CA conserva el protocolo instalado de punteros
y su generación. Las tablas nuevas tienen RLS forzada, propietario exacto e
historia inmutable; ninguna concesión nueva permite runtime, PUBLIC o Sistemas.

El plan usa `pfi_` con sufijo opaco de 22 a 124 caracteres; la longitud total
máxima es 128. Las operaciones de provisión de cuentas usan `opr_`, según IS2.
Las Personas, procedencias y política usan `per_`, `prc_` y `pga_` con sufijo
opaco de 22 a 128 caracteres. Organización conserva `org_[a-z0-9]{16,80}`.
El host usa etiquetas DNS minúsculas válidas, con punto y longitud total 4..253.
Las vigencias cubren la caducidad del plan; Persona no supera Organización.

Las preimágenes son JSONB sin huella propia. El CAS usa SHA256 de
`convert_to(preimagen::text,'UTF8')`. AUT39 comprueba por separado el SHA del
canon Go del plan y la aprobación externa. Las fachadas propietarias reciben
ese compromiso; el JSON no concede permisos. El replay conserva el recibo
original y coteja otra vez el recibo real IS, sin crear efectos nuevos.

Vectores preparados:

- IS15: positivo, cuatro cuentas/titularidades reales, replay, CAS divergente,
  producción, material cambiado, permiso enviado por el solicitante y recibo
  sin coordenadas privadas.
- CA33: fallo de la segunda Persona y rollback integral de ambos módulos,
  recibo IS inventado, positivo, replay, huella, ausencia de perfiles nuevos y
  comprobación inmediata de claves foráneas.

Dirección carga mediante parámetros enlazados los GUC de sesión privados
`vec.ensayo.plan_fuentes` y `vec.ensayo.material_fuentes`; los vectores no los
imprimen. Cada script termina con ROLLBACK. El SHA local del plan en esos vectores
sólo identifica la fixture de fachada; AUT39 ensaya el canon público real.
El recorrido durable, replay tras reinicio, auditoría AD174 y bootstrap AUT38
se validan en la transacción y el clon coordinados por Dirección. No hay ensayo
ni recuperación acreditados por la mera presencia de estos archivos.

CA mantiene el enum instalado `autoridad_maestra_acreditada` porque lo exigen
sus consumidores. La operación y la titularidad conservan el alcance explícito
`sintetico_declarado` y la aprobación externa. Ese enum técnico no transforma
la fuente de desarrollo en una fuente institucional ni acredita producción.
