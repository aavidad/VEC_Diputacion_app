# Impeccable en VEC — orden de prioridad

Impeccable (Apache 2.0, pbakaus/impeccable v4.4.0, 28/09/2026) se usa como criterio de
diseño y para `audit`, `critique`, `polish`, `clarify`, `harden` y `layout`.

En VEC mandan, por este orden, sobre lo que proponga Impeccable:
1. `usabilidad-vec`: normativa (RD 1112/2018, UNE-EN 301 549 / WCAG 2.2 AA), lenguaje claro,
   cuadro de mandos, fase y siguiente paso visibles, ayuda solo tras «?».
2. `aspecto-vec` y `disenar-sistema-visual-vec`: maqueta de RRHH (`fotos/image004.png`),
   tokens del tema común, sin CSS estructural propio por módulo.
3. Impeccable, para todo lo demás (jerarquía, tipografía, espaciado, estados, detalles).

No usar `overdrive`, `bolder` ni efectos llamativos: es una aplicación de gestión de una
administración pública, sobria y legible. No añadir fuentes externas: solo las del tema.
El motor se descarga bajo demanda desde las versiones publicadas del proyecto (con suma
SHA-256); no se incluyen binarios en el repositorio.


## Humanizer
Para repasar textos (manuales, ayuda «?», mensajes); ver `../humanizer/VEC-USO.md`.
