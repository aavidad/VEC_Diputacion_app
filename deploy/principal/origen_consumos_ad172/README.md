# Filas de origen AD172 para vec-server

## Para qué sirve

Desde que se instaló AD172 en la principal, el núcleo de autorización solo
acepta un uso nuevo de una autorización si encuentra su fila en
`vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1`. La fila es la
terna LOGIN, audiencia y operación, más el canal y el nombre del proceso. Si no
la encuentra, PostgreSQL responde `42501 origen de consumo no acreditado` y la
API lo devuelve como `403`, sin más pista.

El 6 de octubre de 2026 la tabla de la principal solo tenía las dos filas de la
administración. Por eso casi todo lo que hace RRHH en vec-server daba 403 aunque
la persona tuviera permiso. El registro de PostgreSQL de esa mañana tiene 97
rechazos con ese mensaje: «Mis preferencias», «Mi imagen», «Mis correos», el
contacto de la participación en Bolsa, las peticiones de los centros, Cronos y
la política de ofertas. El último uso correcto anterior es del 3 de octubre,
antes de instalar AD172.

Este paquete añade las filas que faltan. Las filas no conceden acciones,
perfiles ni membresías. El núcleo sigue exigiendo la decisión firmada, el
grupo ejecutor de cada perfil y que audiencia, operación y canal coincidan.
Las filas solo dicen qué proceso técnico firma el asiento de auditoría.

## La lista

`ternas.tsv` es la única fuente: la leen el guion, el inventario, el cotejo
posterior y la prueba. Tiene una fila por terna con su bloque, el LOGIN, el
grupo ejecutor, la audiencia, la operación, el canal, el proceso y el perfil
del núcleo.

| Bloque | LOGIN | Ternas | Proceso | Se instala |
| --- | --- | --- | --- | --- |
| `usuarios` | `vec_pref508a_i_ue` (interna) y `vec_pref508a_e_ue` (externa) | 21 | `vec-usuarios` | por defecto |
| `contratacion` | `vec_ct_o207_runtime` | 50 | `vec-contratacion-temporal` | por defecto |
| `bolsa` | `vec_bolsa_llamamientos_desarrollo` | 21 | `vec-bolsa` | por defecto |
| `documentos` | `vec_documentos_rrhh_ejecutor_desarrollo` | 8 | `vec-documentos` | por defecto |
| `incorporacion` | `vec_inc_v2_registro_ct_20260910`, `vec_inc_v2_alta_personal_20260910`, `vec_inc_v2_lector_personal_20260910` | 3 | `vec-incorporacion` | por defecto |
| `cronos` | `vec_cronos_emp_ejecutor_desarrollo` | 8 | `vec-cronos` | solo si se pide |

De dónde sale cada dato, cotejado en la principal el 6 de octubre, solo con
lecturas:

- **Perfil, audiencia y operación:** del texto vivo de
  `consumir_decision_mutacion_v3_interna`. Para cada perfil, la parte que fija
  sus audiencias y operaciones y la parte que fija el grupo ejecutor que debe
  tener el LOGIN. Los perfiles de Contratación son los que caen en su rama
  general, que exige `vec_contratacion_temporal_ejecutor`.
- **LOGIN:** el miembro de ese grupo que usa vec-server. Contratación y Bolsa
  salen de las DSN del arranque (`VEC_CT_DATABASE_URL` y
  `VEC_BOLSA_LLAMAMIENTOS_DATABASE_URL`). Documentos y Usuarios tienen un solo
  miembro con conexión abierta desde vec-server. Incorporación sale de los
  grupos de `internal/app/composicion/interna/pools_seguimiento.go`: registro
  CT, alta en Personal y lectura de Personal. Esa composición está apagada hoy
  en la principal (no hay `VEC_CT_INCORPORACION_V2_FILE`), pero sus LOGIN
  existen y sus tres filas quedan listas para cuando se encienda.
- **Canal:** `interna_corporativa` para todo lo de RRHH, como en el historial de
  usos. Usuarios lleva además la superficie externa.

El emparejamiento de perfil, audiencia y operación, y el grupo que el núcleo
exige a cada perfil, se cotejaron a mano sobre el texto exacto del núcleo, y
una segunda revisión independiente lo repitió. El guion no repite ese cotejo,
pero sí se asegura de que el núcleo es el mismo: su huella SHA256 tiene que
estar en `nucleos_cotejados.txt`. Ahí están la de la principal del 6 de
octubre (postimagen de AD193) y la que deja AD208 (#809), que solo cambia el
SQLSTATE del rechazo. Con cualquier otra huella el guion se para. Si el núcleo
cambia, hay que volver a cotejar la lista y añadir la huella nueva en la misma
revisión.

Unas 24 de las ternas por defecto tienen hoy audiencias sin clave de capacidad
publicada en la principal: los avisos de llamamiento de Usuarios; en
Contratación, las firmas, las plantillas, los ajustes de reglas, el circuito,
el correo del llamamiento y la reincorporación del titular; y en Bolsa, la
auditoría, la aceptación en Contratación, la política de cese, la
reincorporación del titular y la resolución de solicitudes documentales.
Mientras no haya clave, esas filas no hacen nada. Quedan listas para cuando se
publique, así que esas funciones no vuelvan a dar 403. Lo mismo pasa con las
tres de incorporación hasta que se encienda su composición. Si al encenderla
los LOGIN privados no fueran estos, las filas quedarían sin uso, sin riesgo.

Fuera de la lista, a propósito:

- **Dietas.** Sus LOGIN están en `NOLOGIN` desde la retirada P6, y F4b, que
  cambiaría esos LOGIN, no está aplicado. Una fila hoy no serviría. Cuando se
  reactive, va con sus LOGIN reales.
- **Personal B2, categorías RPT, vínculo de categoría y plan de incorporación a
  Personal.** Solo los usa la composición B2, que está apagada en la principal
  y no tiene sus LOGIN. Cuando se componga, va en su propio corte.
- **Portal del candidato, «Mi bolsa» y Aspirantes.** Son de la superficie
  externa, y sus grupos ejecutores no tienen miembros en la principal.
- **Administración.** Es otro proceso. Ya tiene sus filas o las pone su propio
  paquete.
- **Organización histórica de Personal.** Ningún LOGIN de vec-server la usa.

El cuadro, el detalle y la auditoría de Contratación no necesitan fila: pasan
por el núcleo de consulta RRHH, que no lleva AD172.

## Qué comprueba antes de escribir

- PostgreSQL 18, DBA por el socket local y base `postgres`.
- Que cada fila de la lista tiene ocho campos válidos y que no hay claves
  repetidas.
- Que AD172 está instalada: tabla con RLS forzada, su propietario, sus dos
  disparadores de inmutabilidad y ningún permiso ajeno al propietario.
- Que el núcleo vivo es el cotejado (`nucleos_cotejados.txt`), que llama al
  resolutor y que nombra cada perfil, audiencia y operación, y que cada
  audiencia está admitida para las claves de capacidad.
- Que cada LOGIN cumple lo que pide el resolutor y que su única membresía es el
  grupo que el núcleo exige a su perfil: heredada, sin `SET` ni `ADMIN`.
- Que ninguna terna existe ya con otro proceso o canal. Si existe, se para:
  la tabla no admite cambios y eso lo decide el DBA.

Inserta como `vec_autorizacion_atestada_v3_propietario`, que es lo que exige la
política de la tabla. Las ternas que ya existen idénticas se saltan, así que
repetirlo no tiene efecto.

## Cómo se usa en la principal

Como `openclaw` en cidonia, desde una copia del repositorio con esta rama:

```bash
export VEC_ORIGEN_PG_CONTENEDOR=vec-postgresql-20260906
bash deploy/principal/origen_consumos_ad172/ejecutar.sh --inventario
bash deploy/principal/origen_consumos_ad172/ejecutar.sh --ensayo
VEC_ORIGEN_AD172_APLICAR=SI-REVISADO \
  bash deploy/principal/origen_consumos_ad172/ejecutar.sh --aplicar
```

Sin `VEC_ORIGEN_BLOQUES` se instalan los cinco bloques de RRHH, 103 ternas.
Cronos se instala aparte cuando se quiera, con `VEC_ORIGEN_BLOQUES=cronos`.

`--ensayo` ejecuta todo y termina en `ROLLBACK`. Debe imprimir
`ternas_nuevas=103` y `verificado: ROLLBACK`, con el inventario sin cambios.
`--aplicar` termina en `COMMIT` y comprueba que estén todas las ternas, que no
haya desaparecido ninguna fila previa y que no haya filas nuevas fuera de la
lista. Los inventarios quedan en un directorio temporal privado cuya ruta se
imprime. Hay que copiarlo a la bitácora privada, fuera de Git, porque `/tmp`
puede vaciarse.

No hace falta reiniciar la aplicación: el núcleo lee la tabla en cada uso.
Después, «Mis preferencias» con un certificado de RRHH debe dar 200, igual que
la bandeja de peticiones del centro y el contacto de la participación.

**Antes de aplicar hay que decidir los nombres de proceso.** La tabla no se
puede corregir después. Aquí va uno por componente (`vec-usuarios`,
`vec-contratacion-temporal`, `vec-bolsa`, `vec-documentos`,
`vec-incorporacion`, `vec-cronos`). Para cambiarlos basta con editar la séptima
columna de `ternas.tsv`.

## Ensayo

`bash deploy/principal/origen_consumos_ad172/probar_pg18.sh`

Usa PostgreSQL 18.4 desechable, sin red. Los objetos de AD172 se copian tal cual
de su migración, y los roles y el texto del núcleo se generan desde la misma
`ternas.tsv`. Prueba lo siguiente:

- El ensayo con `ROLLBACK`, la aplicación con y sin confirmación y la
  repetición sin efecto.
- El resolutor con el LOGIN real de cada bloque. Acepta su terna y rechaza el
  canal cruzado, el LOGIN cruzado y otro LOGIN del mismo grupo.
- Que Cronos solo se instala si se pide.
- Los rechazos por terna en conflicto, membresía extra, grupo equivocado,
  permisos de más en la tabla, núcleo sin AD172, núcleo distinto del cotejado,
  bloque desconocido y LOGIN sin permiso de conexión.

El núcleo de la prueba es un sustituto generado desde la misma lista. Prueba el
guion, no la lista: los errores de la lista solo los detecta el cotejo con el
núcleo real descrito arriba.

En la principal se cotejó además, en una transacción de solo lectura, que las
111 ternas las reconoce el núcleo vivo y el catálogo de audiencias, que cada
LOGIN tiene la única membresía que exige su perfil y que no hay ninguna en
conflicto.
