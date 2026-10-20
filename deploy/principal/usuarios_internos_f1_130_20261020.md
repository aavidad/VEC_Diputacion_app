# Preferencias e imagen de las identidades internas

F1-130 continúa la preparación de Usuarios de F1-113. El consumidor es
«Mis preferencias» y «Mi imagen» del portal interno, también para Jefatura.
La cuenta y el perfil de cada persona tienen que estar publicados antes de
enlazar su certificado. El enlace de configuración no concede permisos.

## Fuente y gobierno

`fuentes/usuarios_preferencias_imagen_propias_v1.json` contiene solo:

- `vec.preferencias.consultar`: catálogo, valores y versión propios;
- `vec.preferencias.actualizar`: valores y versión propios;
- `vec.imagen.consultar` y `vec.imagen.actualizar`: foto, icono, modo, paleta
  y versión propios.

Las cuatro entradas exigen garantía alta y el ámbito `persona_ref`. La
asignación debe limitar ese ámbito a la persona acreditada por la cuenta.
El servidor obtiene esa persona de la sesión certificada; la petición no
puede elegir otra. No se incluyen las seis acciones de correo del rol antiguo.
La vigencia de esta fuente de desarrollo es de 90 días.

La receta admite la fuente con el productor de catálogo de acciones existente
(AUT58), su plan y su paquete aprobados. Después publica la versión de rol y
su asignación con los actos de los dos administradores de la copia. Se
conservan claves, huellas, recibos y referencias para recuperar esos mismos
actos. No se añaden asignaciones mediante SQL ni se reutiliza la fuente externa.

## Enlace con la orden de identidad

La etapa de identidad de F1-131 consumirá esta función de
`copia_local_usuarios.py`, bajo el candado de la instancia:

```python
vincular_cuenta_interna(material, identidad, certificado, cuenta_ref, perfil_ref)
```

`identidad` y `certificado` son rutas relativas de un solo fichero bajo
`identidad/` y `mtls/`. Los dos últimos argumentos proceden de la publicación
confirmada, no del perfil de firma elegido por su nombre. La función coteja
el certificado con el sujeto del fichero de identidad y añade los cuatro
campos de la cuenta al JSON interno ya preparado por F1-113.

La llamada devuelve `True` al añadir el enlace y `False` ante una repetición
exacta, sin reescribir el fichero. Una referencia divergente, una colisión
con la superficie externa o una ruta ajena detiene el enlace y conserva el
original. Las conexiones, los motivos y el JSON externo se conservan.
La etapa reinicia después de forma vigilada y comprueba su consulta; no se
cambia este fichero mientras lo está leyendo una aplicación activa.

## Comprobación y alcance actual

El corte `6173508c6` pasó 14 pruebas Python y la prueba Go de canon, campos,
acciones y ámbito de la fuente. Las pruebas del enlace incluyen recuperación
exacta, certificado divergente, cuentas externas y conservación del original.

Para dar por terminado el recorrido falta integrar el consumidor de F1-131,
admitir la fuente y publicar las versiones y asignaciones en una copia nueva.
Como Jefatura, comprobar en Chrome «Mis preferencias» y «Mi imagen» con 200,
sin cookies. Repetir después del reinicio, conservando las mismas referencias.
Comprobar también que una identidad sin asignación positiva recibe denegación.
El servidor debe arrancar antes de esta activación y después de ella.

No hay variables de entorno ni migraciones nuevas. El fichero privado que
cambia es `identidad/usuarios-preferencias-interna.json`, ya existente. Hasta
que se publique el perfil y se integre la etapa, este documento no acredita
la consulta de Jefatura ni la preparación completa de una copia.
