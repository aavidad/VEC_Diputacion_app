# AUT42 — mantenimiento aprobado del perfil fijo

La migración instala configuración y helpers privados; no publica Rol5. La
operación técnica `mantener_version_perfil_fijo_admin_v1(text,text)` compara el
SHA de los bytes del plan con la aprobación externa del LOGIN exclusivo. El
DBA prepara esa configuración fuera de Git: plan/preimagen/catálogo/SHA y
aprobación, entorno desarrollo y ventana. La CLI no concede permisos ni crea
esa aprobación.

Plan V1, diez campos cerrados:

```text
version = 1
operacion_ref = pmf_<22..124 caracteres opacos>
preparado_en, caduca_en = UTC con segundos
rol_origen_sha256
control_revision_esperada, control_huella_sha256
catalogo_sha256
rol_destino_doc
asignaciones[2] {
 perfil_ref, asignacion_origen_ref, asignacion_origen_sha256,
 persona_ref, cuenta_ref, vinculo_ref,
 cuenta_version, persona_version, perfil_version, vinculo_version,
 ambitos_fuente
}
```

`rol_destino_doc` debe ser copia exacta de Rol4, salvo versión5, publicador
`mantenimiento_operador:<LOGIN real>`, fecha igual a preparación y las dos
concesiones cerradas del catálogo de denominación. No se acepta un documento
arbitrario aunque su SHA esté aprobado. Los campos/vigencias/ámbitos originales
se conservan en ambas asignaciones v2; no se crea Persona, perfil o vínculo CA.

La preimagen revalida Rol4/control/categoría/gobierno y catálogo actuales, los
dos perfiles de Aplicación activos y su fuente CA. Organización y unidad se
cotejan por los puertos existentes de CA/Personal31; AUT no consulta sus tablas.
Las asignaciones no pueden tener historia revocada ni se amplía su vigencia.

El efecto une publicación Rol5/control/categoría/catálogo, versiones v2 de las
dos asignaciones, sellos de AUT24 y punteros CAS con AD183 en una transacción.
Sistemas y el bootstrap original permanecen en su historia. Los helpers
legados tienen dispatcher exacto4/5: la rama4 conserva el cuerpo literal; la
rama5 sólo admite catálogo5 heredado y controles propios, sin `version>=`.
No se modifica el núcleo AD ni se utiliza una tabla de auditoría propia.

La recuperación usa los sellos24 y el evento determinista AD183 para conservar
el acuse original. No repone un puntero revocado o cambiado después. Cada
invocación, incluido replay/rechazo/error gestionados, añade intento común.
Un fallo de append aborta todo; un fallo de efecto revierte el subbloque y deja
sólo el intento. La CLI comunica resultado únicamente después de COMMIT.

`validar_administrador_denominacion_persona_v1(d,m)` es privado para AD184:
rol5 fijo de Aplicación, actor/asignación/control vigentes y huellas exactas;
acción publicar/leer, campos `[denominacion]`/`[nombre_mostrar]`, obligación
`[auditar]`, módulo `vec`, tipo `persona_denominacion`, finalidad
`presentacion_persona`. Exige organización y unidad asignadas y el contexto
hash ligado a los bytes canónicos de M/procedencia. Los codecs puros de AD184
se resuelven al llamar; si faltan, el gate deniega. No hay ciclo de instalación:
42 crea el gate; 184 exige ese gate y aporta sus codecs. Ninguno concede
permisos por los datos JSON de la petición.

Orden causal: AUT33/34/37 y AD183 previas; después AUT42 y AD184/CA32 por sus
propietarios. La lista de esta rama sólo contiene la migración42. No se
reaplican SQL instaladas ni se ejecuta DOWN. Las actas SQL reales y dos revisiones
exactas las coordina dirección. La vigencia antigua del ensayo no se amplía:
el positivo se realiza en otro objetivo recuperado y aprobado si ha caducado.
