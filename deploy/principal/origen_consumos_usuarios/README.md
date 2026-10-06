# Origen AD172 de Usuarios: preferencias, imagen, correos y avisos

## Para qué sirve

Desde que se instaló AD172 en la principal, el núcleo de autorización solo
acepta un consumo nuevo si encuentra su fila en
`vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1`. Si no la
encuentra, PostgreSQL responde `42501 origen de consumo no acreditado` y la
API lo devuelve como `403 prohibido`.

El 6 de octubre de 2026 la tabla de la principal solo tenía las dos filas de
la administración. «Mis preferencias», «Mi imagen» y «Mis correos» daban 403
a cualquier persona, aunque la decisión de autorización era positiva. El
registro de PostgreSQL lo confirma: entre las 09:10 y las 10:57 (UTC) hubo 40
rechazos en preferencias, 26 en imagen y 18 en correos, todos en esa línea del
núcleo y con el LOGIN `vec_pref508a_i_ue`. El último consumo correcto de
preferencias es del 3 de octubre a las 21:14 (UTC), antes de instalar AD172.

Este paquete añade 21 ternas de Usuarios. Veinte son las diez operaciones de
esas tres pantallas (consultar y actualizar preferencias, consultar y
actualizar imagen, y las seis de correos) en las dos superficies:

| LOGIN | Grupo ejecutor | Canal | Audiencia | Operación |
| --- | --- | --- | --- | --- |
| `vec_pref508a_i_ue` | `vec_usuarios_ejecutor_interno` | `interna_corporativa` | `vec_usuarios.<familia>.<acción>.interna_corporativa.v1` | `vec.<familia>.<acción>` |
| `vec_pref508a_e_ue` | `vec_usuarios_ejecutor_externo` | `externa_personal` | `vec_usuarios.<familia>.<acción>.externa_personal.v1` | `vec.<familia>.<acción>` |

La terna 21 es la de AD109: al emitir un llamamiento, Bolsa pide a Usuarios
el correo activo de cada candidata. Esa lectura la ejecuta `vec_pref508a_i_ue`
con la audiencia `vec_usuarios.correos.avisos_llamamiento.interna_corporativa.v1`,
la operación `llamamiento.emitir.v1` y el canal `interna_corporativa`. Sin ella,
el aviso saldría siempre al correo del alta en la bolsa.

El proceso es `vec-usuarios` en todas; el canal ya distingue las superficies
en la auditoría. Como la tabla no admite cambios, el nombre del proceso se
decide antes de aplicar. Las filas no conceden acciones, perfiles
ni membresías. Solo dicen qué proceso técnico firma el asiento de auditoría.
El núcleo sigue comprobando la decisión, el LOGIN, su grupo y el canal.

## Qué comprueba antes de escribir

- PostgreSQL 18, DBA por el socket local y base `postgres`.
- AD172 instalada: tabla con RLS forzada, propietario correcto, sus dos
  disparadores de inmutabilidad y ningún permiso ajeno al propietario.
- El núcleo vivo llama al resolutor y nombra cada audiencia y operación, y
  cada audiencia está admitida para las claves de capacidad.
- Cada LOGIN cumple lo que pide el resolutor y es el único miembro de su grupo
  ejecutor, con una sola membresía heredada y sin `SET` ni `ADMIN`.
- Si una terna ya existe con otro proceso o canal, se para. La tabla es
  inmutable y eso lo decide el DBA.

Inserta como `vec_autorizacion_atestada_v3_propietario`, que es lo que exige la
política de la tabla. Las ternas que ya existen idénticas se saltan, así que se
puede repetir sin efecto.

## Cómo se usa en la principal

Como `openclaw` en cidonia, desde una copia del repositorio con esta rama:

```bash
export VEC_ORIGEN_PG_CONTENEDOR=vec-postgresql-20260906
bash deploy/principal/origen_consumos_usuarios/ejecutar.sh --inventario
bash deploy/principal/origen_consumos_usuarios/ejecutar.sh --ensayo
VEC_ORIGEN_USUARIOS_APLICAR=SI-REVISADO \
  bash deploy/principal/origen_consumos_usuarios/ejecutar.sh --aplicar
```

`--ensayo` ejecuta todo y termina en `ROLLBACK`. Debe imprimir
`ternas_nuevas=21` y `verificado: ROLLBACK`, con el inventario sin cambios.
`--aplicar` termina en `COMMIT` y comprueba que estén las 21 ternas, que no
haya desaparecido ninguna fila previa y que no haya filas nuevas ajenas. Los
inventarios quedan en un directorio temporal privado cuya ruta se imprime;
copiarlo a la bitácora privada, fuera de Git, porque `/tmp` puede vaciarse.

No hace falta reiniciar la aplicación: el núcleo lee la tabla en cada consumo.
Después, abrir «Mis preferencias» con un certificado de RRHH debe dar 200.

## Ensayo

`bash deploy/principal/origen_consumos_usuarios/probar_pg18.sh`

Usa PostgreSQL 18.4 desechable sin red, con los objetos de AD172 copiados
literalmente de su migración y un fixture sintético. Prueba el ensayo con
`ROLLBACK`, la aplicación, la repetición sin efecto y el resolutor con el LOGIN
real: acepta su terna y rechaza el canal, el LOGIN o la operación cruzados.
También prueba los rechazos por terna en conflicto, membresía extra, grupo
compartido, permisos de más en la tabla y núcleo sin AD172.
