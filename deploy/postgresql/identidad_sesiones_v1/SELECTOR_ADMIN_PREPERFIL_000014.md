# Selector ADMIN sin perfil activo — IS14 y CA31

Este corte prepara el listado y la elección explícita de perfiles propios de
Administración de la aplicación. La cuenta se obtiene de IS9 mediante certificado
mTLS verificado, revocación vigente, host y política exactos. El listado no exige
perfil activo ni una decisión V3. La categoría procede del catálogo positivo AUT33;
los roles antiguos sin esa metadata quedan fuera. Sistemas continúa pendiente.

## Contrato para el adaptador

Las dos fachadas del esquema `vec_identidad_sesiones_v1` devuelven una sola fila
`resultado jsonb, auditoria_comun_ref text`:

- `listar_perfiles_admin_auditado_v1`: nueve argumentos de observación mTLS del
  contrato anterior, más `evento_ref` y `correlacion_ref`.
- `seleccionar_perfil_admin_auditado_v1`: esos nueve, `perfil_ref`,
  `revision_esperada`, `evento_ref` y `correlacion_ref`.

El listado contiene `revision`, `perfil_activo_ref` y `perfiles`. Cada perfil
contiene `perfil_ref`, `rol_version_ref`, `clave_i18n` y `categoria_admin`. La clave
`administracion.perfiles.rol` ya pertenece al catálogo; no se devuelve un nombre
visible escrito dentro de SQL. La selección contiene `perfil_ref`,
`seleccion_revision` y `seleccionada_en`.

La petición genera `evento_<32 hex>` nuevo en el adaptador confiable; la correlación
es `correlacion_<32 hex>` del contexto de petición. Ninguna cabecera libre concede
estos datos. Cada lectura y cada recuperación de selección dejan su evento común.
Una repetición semántica conserva revisión y fecha de elección; revalida identidad,
asignación, categoría y vigencia. Una revisión anterior a otro cambio se deniega,
incluso si el perfil actual vuelve a tener la misma referencia.

Las fachadas requieren `SERIALIZABLE READ WRITE` con `TimeZone=UTC`. El adaptador
valida el acuse y confirma la transacción antes de entregar datos o comunicar la
denegación. Una denegación posterior a resolver la identidad devuelve
`{"estado":"denegado","motivo_ref":"..."}` y su auditoría común. Un fallo de
identidad sin actor acreditado no fabrica un evento preperfil: su auditoría
corresponde a la frontera de autenticación. Un fallo técnico de AD171 o del COMMIT
impide devolver éxito. La aplicación debe tratar los códigos SQL de conflicto
sin atribuir persistencia a una transacción fallida.

## Dependencias y acreditación técnica

Orden causal: AD171, CA31 e IS14. La lista está en
`deploy/principal/lista_sql_trabajo_codexk_identidad_preperfil_20261003.txt`.
AD171 pertenece a la rama del escritor de auditoría y debe estar en el árbol
integrado para ensayar esa lista. También se requieren IS9, CA20, AUT24 y AUT33
ya instaladas; no reaplicarlas. No se importan IS11, CA22, CA23 ni IS12.

El rol técnico nuevo es `vec_identidad_sesiones_v1_admin_preperfil`. Sólo tiene
CONNECT, USAGE en identidad y EXECUTE sobre las dos fachadas y el preflight
`acreditar_runtime_preperfil_admin_v1()`. Un LOGIN privado se incorpora a ese
único grupo con `INHERIT TRUE, SET FALSE, ADMIN FALSE`, sin privilegios directos.
No tiene acceso a tablas, a los escritores propietarios ni al append de AD171.

El canal privado aprovisiona una fila en
`vec_identidad_sesiones_v1.config_runtime_admin_preperfil_v1`: `identidad_login`,
`proceso`, `entorno`, `host_admin`, `audiencia`, `vigente_hasta`. `proceso` admite
`^[a-z][a-z0-9._-]{1,79}$`. No hay LOGIN ni configuración sembrados. El preflight
acredita la topología, las ACL exactas y la vigencia; devuelve `acreditada` y
`proceso`. Para sustituir un consumidor se usa otro LOGIN y otra fila; la fila
anterior permanece inmutable.

IS conserva la observación original en una tabla de sólo adición y construye el
JSON interno de AD171. La fuente es `observacion_admin_preperfil:<32 hex>` y su
SHA256 corresponde al documento de la observación. Nunca se copian certificados,
secretos ni cuerpos HTTP a la cadena. CA almacena la referencia común que recibe
IS, junto con la revisión y fecha; no fabrica un hash local como auditoría.

La selección CA31 aún no compone una sesión posterior con perfil. Esa composición
necesita su tarea propia; este corte no acredita ADMIN completo, una familia de
Sistemas, autenticación externa publicada ni un recorrido de navegador.

## Comprobaciones preparadas y pendientes

`pruebas_sql/000014_selector_admin_acl.sql` crea un LOGIN sintético dentro de
ROLLBACK. Comprueba preflight mínimo, ausencia de acceso a evidencia/tablas,
denegación por host ajeno o identidad ausente y rechazo al ampliar la ACL.
Debe ejecutarse en el clon con PostgreSQL real tras instalar el árbol causal.

Quedan por ejecutar el ensayo de migraciones en el clon, esa prueba SQL y los
casos positivos con una asignación sintética actual del catálogo AUT33: listado
sin selección, elección CAS, replay con nueva auditoría, revisión obsoleta,
revocación, rollback del CAS y recuperación tras reiniciar. Las dos revisiones
sensibles deben usar el hash final. El productor no ha conectado PostgreSQL ni
Go; Dirección coordina el ensayo y monta el adaptador.
