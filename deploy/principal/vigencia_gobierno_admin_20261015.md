# Vigencia del gobierno de Administración

Encargo de dirección del 15 de octubre de 2026, a las 19:36. Es un ajuste
técnico para preparar claves y confianza nuevas. No cambia las competencias
de RRHH ni los permisos de las personas.

## Preparación

1. Restaurar la copia de ensayo propia y comprobar que están instaladas las
   funciones de gobierno de usuarios y del conjunto de capacidades. Instalar
   AD282 una sola vez, según
   `lista_sql_codexh_vigencia_gobierno_20261015.txt`. Si su preimagen no
   coincide, detener la operación y revisar la base de partida. Si la
   posimagen ya es la esperada, no ejecutar otra vez la migración.
2. Compilar `cmd/vec-gobierno-usuarios-admin`. Conservar el material anterior
   para la vuelta atrás y usar una carpeta privada nueva, vacía y con permisos
   `0700`. Los JSON privados conservan permisos `0600`.
3. Preparar el JSON habitual del comando, con las conexiones de lectura y del
   operador nominal, el material y la semilla de la raíz de esa copia. No
   copiar recibos de otra instalación.

## Selección del plazo

Sin valor nuevo, el comando usa `horas_validez_claves` del JSON. Su ausencia o
el valor cero seleccionan 24 horas. La variable
`VEC_GOBIERNO_ADMIN_VALIDEZ_HORAS` tiene prioridad; debe ser un entero positivo.

| Entorno | Plazo admitido |
| --- | --- |
| Producción, también cuando falta el perfil | De 1 a 24 horas |
| Desarrollo con las tres llaves existentes | De 1 a 9600 horas, hasta 400 días |
| Perfil desconocido | Operación rechazada |

Para una copia de desarrollo con vigencias de 90 días, que deja margen sobre
el mínimo de 60 días restantes:

```sh
export VEC_EXECUTION_PROFILE=desarrollo
export VEC_AUTH_MODE=desarrollo
export VEC_DEVELOPMENT_GUARD=ACEPTO_CREDENCIALES_NO_AUTORITATIVAS_SOLO_DESARROLLO
export VEC_GOBIERNO_ADMIN_VALIDEZ_HORAS=2160
```

cidonia usa el perfil de desarrollo, pero solo dirección puede instalar allí
una candidata ensayada. Este guion se ejecuta primero en una copia local.

La publicación de Administración usa la confianza común de CT. El
verificador solo admite más de 24 horas si están las tres llaves de desarrollo
y todas las raíces tienen la audiencia exacta de CT de desarrollo. No basta
un nombre de revisión o un prefijo. Al recargar, CT conserva la publicación
vigente, sin acortarla ni sustituirla por otra diaria.

Las claves y la confianza no pueden durar más que la raíz existente. Si la
raíz no cubre el plazo pedido, preparar se rechaza; no se prolonga la raíz.
El plan para aplicar sigue caducando en dos horas como máximo. La ventana
técnica de admisión de catálogos también conserva su límite anterior. Las
capacidades de cada acto siguen siendo breves y se consumen una sola vez;
alargar la confianza no prolonga una decisión de autorización.

## Publicación y comprobación

1. Ejecutar `preparar` con el JSON privado y el catálogo de textos del idioma
   activo. Comprobar las fechas de `configuracion-material.json` y `plan.json`
   sin imprimir sus secretos.
2. Fijar en la base la aprobación técnica con las huellas que devuelve la
   preparación, mediante el operador y el procedimiento ya existentes.
3. Ejecutar `aplicar` dentro de la ventana del plan y guardar el acuse.
   Repetir con otro nombre de acuse: debe conservar el mismo recibo y la
   publicación, sin añadir otra configuración ni otras claves.
4. Ejecutar `verificar`. Incorporar el material nuevo a la configuración
   privada del proceso de Administración mediante su procedimiento habitual.
   Reiniciar vigilado y comprobar en Chrome la lista y la ficha de usuarios.
5. Comprobar el arranque sin la nueva variable y con ella. El portal no usa
   esta variable para conceder permisos: solo la herramienta de gobierno
   decide el plazo del material que prepara o aplica.
6. En producción, pedir más de 24 horas debe rechazar la operación, incluso
   si el material se preparó antes en desarrollo. No debe reservar un acuse
   ni publicar nada.

Un plan diario anterior todavía sin aplicar se rechaza si sus claves rebasan
su confianza. Se prepara otro en una carpeta nueva. Las publicaciones y sus
recibos ya existentes no se reescriben.

## Vuelta atrás y límites

Retirar la variable devuelve el plazo predeterminado para la siguiente
preparación; no cambia la duración de lo ya publicado. No ejecutar DOWN ni
reaplicar una migración. Si una publicación nueva impide el arranque, restaurar
la configuración privada anterior y seguir la renovación gobernada, sin
alterar claves ni firmas históricas.

La lectura de usuarios de Administración es el consumidor web. Este corte no
añade pantallas ni acredita una activación en cidonia. Los resultados del
ensayo, del reinicio y de la revisión se adjuntan a la PR de esta rama.
