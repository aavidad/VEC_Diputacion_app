# Preferencias de Mi Bolsa interna

El código preparado monta `GET/PUT /api/vec/bolsa/mi-bolsa/preferencias` para la sesión
corporativa de Antonio. El formulario web ya usa estos dos métodos. `GET`
devuelve el orden y el último recibo; `PUT` conserva el orden con versión
esperada e idempotencia y devuelve un recibo. La ruta queda en 503 si se pide
la capacidad sin completar su material o sus funciones. Sin el selector, el
servidor arranca y la ruta no se monta.

## Contrato de SQL con C

La instalación causal en una copia nueva es **AUT96 → AD273 consolidada →
B116 → AD275**. No se reaplica una migración ya instalada. AUT96 coteja la
asignación interna vigente. B116 crea el grupo
`vec_bolsa_llamamientos_preferencias_interna` y tres fachadas Bolsa con sufijo
`_interna_v1`. B116 amplía la restricción de clave de B104 y guarda actor y
perfil en las versiones internas nuevas. Almacena la clave interna con un
prefijo que B104 no admite, de modo que cada superficie solo recupera su
propio recibo. AD275 crea las dos fachadas nominales de lectura y de intento
denegado; solo admite la sesión interna.

AD273, propiedad de C, debe aportar dos piezas antes de B116/AD275:

1. La función
   `vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_interna_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)`.
   Debe aceptar únicamente el LOGIN interno nominal, la audiencia
   `vec.bolsa.preferencias-bolsas.ordenar.interna.v1`, la acción
   `bolsa.preferencias_bolsas.ordenar_propias`, el recurso propio
   `mi-bolsa:can_*`, la finalidad
   `gestion_preferencias_bolsas_propias` y el vínculo
   `interna_corporativa`. Conserva el consumo V3, la auditoría y la
   recuperación del recibo original en una transacción.
2. La restricción validada
   `vec_autorizacion_atestada_v3.auditoria_consumo_v3.auditoria_lectura_preferencias_bolsas_formato_v2`.
   Debe conservar todas las condiciones de la restricción v1 de AD246 y
   admitir el canal `interna_corporativa` para el tipo
   `lectura_preferencias_bolsas`, junto al canal exterior que ya existía.
   AD275 exige el nombre y el canal internos antes de instalarse. La
   modificación de esa tabla pertenece a C.

AUT96/AD275 cotejan una concesión publicada por Administración para el perfil
propio de Antonio. La lectura requiere la acción
`bolsa.preferencias_bolsas.consultar_propias`, la finalidad anterior y los
ocho campos de `CamposLecturaPreferenciasBolsas()`. La escritura requiere
`bolsa.preferencias_bolsas.ordenar_propias`, la misma finalidad y ningún
campo. Ambas conservan el ámbito `candidato_ref` único. El arranque no
publica concesiones ni asignaciones.

## Material y activación

El DBA crea fuera de Git el LOGIN
`vec_interno_bolsa_preferencias_desarrollo` con membresía directa y exclusiva
en `vec_bolsa_llamamientos_preferencias_interna`, `INHERIT TRUE`, `SET FALSE`
y `ADMIN FALSE`. B116 concede al grupo solo `USAGE` del esquema Bolsa y
`EXECUTE` de las tres fachadas internas. El LOGIN no comparte las credenciales
del portal externo ni las de Usuarios.

El material privado del proceso interno incluye
`identidad/bolsa-preferencias-interna.json` con `version: 1`,
`superficie: "interna_corporativa"`, `dsn_bolsa` del LOGIN nominal y las
referencias acreditadas `certificado_sha256`, `sujeto`, `cuenta_ref`,
`perfil_ref`, `persona_ref` y `candidato_ref`, con TLS `verify-full`.
La cuenta y el perfil internos son distintos de los del aspirante externo.
Se activa con `VEC_BOLSA_PREFERENCIAS_INTERNA_ENABLED=true`.
También requiere `VEC_USUARIOS_PREFERENCIAS_ENABLED=true` y su material
interno acreditado. El montaje reutiliza su infraestructura de sesión y
resuelve la cuenta y el perfil propios de Bolsa desde el fichero anterior.
Exige el certificado, sujeto, persona y candidatura de Antonio antes de cada
petición. La cuenta y el perfil internos deben tener un contexto propio
publicado por los circuitos de Identidad y ContextoActor.
La clave de capacidad de la audiencia interna se deriva del material V3 ya
preparado; el operador publica esa clave, la raíz y su configuración por el
gobierno de confianza antes de activar la ruta. La app no publica claves ni
permisos al arrancar.

Tras instalar y publicar, comprobar en la copia: arranque con el selector
apagado y encendido; `GET` propio con 200 y apunte de lectura; `PUT` propio
con 201 y recibo; repetición idéntica con 200 y el mismo recibo; consulta
después del reinicio con la misma versión, orden, referencia y fecha.
Una candidatura ajena o una sesión exterior deben ser rechazadas y dejar su
auditoría. `POST` responde 405; el cliente web usa `PUT`.

Este archivo describe una activación pendiente. Ninguna de estas migraciones
ni sus actos están instalados o ensayados por este corte de código.
