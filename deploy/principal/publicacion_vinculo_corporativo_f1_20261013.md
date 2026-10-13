# Publicar el vínculo corporativo de C3

La lectura de contacto de Bolsa conserva el cotejo CA40. El publicador registra
el vínculo que falta desde una fuente de desarrollo aprobada: cuenta, persona,
perfil RRHH, contexto y organización corporativa exactos. No concede permisos
de CT ni de Bolsa.

## Orden de instalación

Primero, instalar una vez las cinco SQL de la PR de Auditoría RRHH: AUT82 →
CT208 → B109 → P47 → AD251. Después, aplicar
`lista_sql_vinculo_corporativo_f1.txt`: AD258 → CA42.

AD258 conserva el asiento común del acto. CA42 añade la fuente aprobada, la
prueba y el buzón transaccional, y expone las dos funciones nominales para
comprobar y publicar. No reconstruye CA9 ni CA40. Conservar copia fría de la
base, binario y configuración antes de instalar; la recuperación repone ese
conjunto. No ejecutar DOWN ni reaplicar las migraciones instaladas.

## Provisión de Sistemas y aprobación de la fuente

CONFIG NUEVA: `VEC_VINCULO_CORPORATIVO_DSN` en el entorno privado de la
herramienta, con el LOGIN exclusivo de publicación; no se añade a vec-server.

Sistemas crea el LOGIN fuera de Git y conserva su credencial privada. Debe ser
LOGIN, INHERIT, sin superusuario, creación de roles o bases, replicación,
BYPASSRLS, ajustes ni caducidad. Su única membresía es
`vec_contexto_actor_corporativo_rrhh_publicador`, otorgada por el propietario
de la base con INHERIT TRUE, SET FALSE y ADMIN FALSE. No recibe acceso directo
a tablas ni permiso de cambiar de rol. No se reutiliza ningún LOGIN de la app.

Dirección aprueba el fichero de fuente con sus datos inventados. El DBA
registra esa aprobación en `vec_contexto_actor_v1.fuente_aprobada_vinculo_f1`:
la herramienta nunca crea ni corrige esa fila. La fuente contiene:

- `operacion_ref` opaca `pubv_…` y `login_nombre` exacto.
- `cuenta_ref`, `persona_ref`, `perfil_ref` y `vinculo_contexto_ref` actuales de
  F1, correspondientes al perfil que usa C3.
- `vinculo_corporativo_ref` opaca nueva `vcr_…` y `organizacion_ref` corporativa
  `org_…`, existente y vigente. No usar aquí el ámbito de negocio de CT.
- Procedencia ya acreditada: referencia, versión, huella y autoridad.
- Huella del fichero de fuente, referencia y huella de aprobación.
- Ventana finita `[vigente_desde, vigente_hasta)`, dentro de las vigencias de
  cuenta, persona, perfil, contexto y organización.

Las referencias se obtienen de F1 y del contexto nominal que usa C3. No se
deducen del nombre de Carmen ni del nombre del rol. El DBA aprueba la fuente;
los vínculos de negocio se escriben exclusivamente mediante el publicador.

## Comprobar y aplicar

Construir `./cmd/vec-publicar-vinculo-corporativo` con la toolchain del
repositorio. Con el DSN privado en el entorno, usar las mismas coordenadas en
ambas llamadas:

```sh
vec-publicar-vinculo-corporativo --comprobar \
  --operacion "$f1_operacion" --cuenta "$f1_cuenta" \
  --persona "$f1_persona" --perfil "$f1_perfil" \
  --contexto "$f1_contexto" --vinculo "$f1_vinculo" \
  --organizacion "$f1_organizacion"
```

La comprobación confirma un asiento común de lectura y devuelve su referencia;
no modifica el vínculo. La salida JSON contiene también el estado y la huella
del plan. Revisarla y conservarla con la aprobación. Aplicar exige esa huella;
relee y bloquea las coordenadas actuales en una transacción SERIALIZABLE:

```sh
vec-publicar-vinculo-corporativo --aplicar \
  --operacion "$f1_operacion" --cuenta "$f1_cuenta" \
  --persona "$f1_persona" --perfil "$f1_perfil" \
  --contexto "$f1_contexto" --vinculo "$f1_vinculo" \
  --organizacion "$f1_organizacion" --huella "$f1_plan_sha256"
```

Conservar el recibo confirmado. Si se pierde la conexión al confirmar, repetir
la misma operación y huella: un replay exacto recupera el recibo sin otra alta.
Una huella distinta, un vínculo revocado o caducado, otra cuenta o un perfil
distinto se rechazan. La herramienta no sustituye una fuente ni un vínculo
existente por comodidad.

Los rechazos controlados devuelven `estado=denegado`, una referencia de auditoría
común y código de salida 1. El subbloque de efecto se revierte antes de registrar
ese intento; no quedan un vínculo o un buzón parciales.

## Comprobación de dirección

Después de la publicación, repetir en Chrome la lectura enmascarada de contacto
que devolvía 403 y conservar el recibo de consulta. Este corte no declara que
ese recorrido haya pasado en cidonia. La prueba local y sus tiempos se anotan
en la PR sobre el cuerpo final ensayado.

Este es el recorte de desarrollo aprobado por dirección el 13/10: fuente
aprobada por DBA, LOGIN segregado y publicación exacta. No declara terminado
C2.3 ni introduce capacidades de fuente F0, HSM o transporte productivo.

El contrato actual admite un perfil corporativo por cuenta/superficie/uso.
El publicador rechaza atómicamente sustituirlo por otro perfil. Dirección
encargó como corte siguiente el vínculo por perfil, con selección explícita
y sin sumar permisos, para que C3 y Auditoría puedan coexistir en la cuenta.

## Ensayo local del cuerpo final

AD258 y CA42 se instalaron una sola vez sobre la copia de principal posterior
a AD251, en PostgreSQL 18, en 0,180 s. Comprobación nominal: 40,6 ms;
publicación: 41,9 ms; replay: 33,1 ms, mismo recibo y evento. Se conservan una
prueba de publicación y un buzón. Las cuatro negativas (otra cuenta, revocado,
caducado y otro perfil aprobado) registran su intento común y no publican otro
vínculo. CA40 acepta el contexto construido de las filas actuales tras publicar
y deniega las variantes de otra cuenta, revocación y caducidad.

Estos tiempos incluyen conexión y CLI; no son tiempos de Chrome ni un GET de
C3. En la copia se repusieron sólo TEMP de PUBLIC y el latido del sellador,
caducado por estar detenida. No se cambió ningún cotejo ni guardia de código.
Los estados de revocación y caducidad son datos de ensayo fuera de Git.

La guarda de SQL pidió expresar las políticas y los disparadores como DDL
literal. Se sustituyó sólo su bucle de construcción: las tres políticas, los
seis disparadores y RLS/ACL coinciden con los objetos ensayados. El DDL literal
se ejecutó y comparó en una transacción de prueba terminada en ROLLBACK; no se
reaplicó la migración instalada ni cambió ninguna función del publicador.

Las familias nuevas participan en la cadena SQL V5. El verificador offline
actual no las representa; no declarar verificación offline completa ni omitir
sus asientos. Dirección conserva el recorrido de C3 en Chrome tras instalar.
