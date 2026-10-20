# F1-132: ficha de firma sin consultas de Personal

La ficha del perfil firmante conserva el detalle y el panel de firma en las
versiones 7 y 8 del expediente. El plan de alta en Personal y el vínculo con
la categoría de la relación de puestos se cargan desde la vista de RRHH.

## Activación

Requiere F1-129. Instalar aplicación y recursos web de la misma revisión y
reiniciar por el procedimiento habitual. Se reutilizan la sesión atestada,
el montaje y la marca común `VEC_WEB_VERSION`.
No hay SQL, configuración ni permisos nuevos.

## Comprobación en la copia local

Usar la copia nominal preparada para F1-129 y sus certificados. No repetir
actos administrativos ni publicar otro plan para comprobar este ajuste.

1. Abrir como Jefatura las fichas observadas en versiones 7 y 8. Confirmar
   sesión 200 con el único rol `ct_cargo_jefatura_firma_dev`, detalle y panel
   de firma. Comprobar escritorio y móvil, teclado y ausencia de desbordamiento.
2. Registrar cero peticiones a las rutas GET bajo `/api/vec/`:
   `contratacion-temporal/incorporacion-personal-b2/plan/v1` y
   `contratacion-temporal/incorporacion-personal-b2/vinculo-categoria-rpt/v1`.
3. Con el perfil técnico de RRHH, comprobar que el montaje conserva su cliente
   de Personal y las consultas que requiere su ficha.
4. Tras reiniciar por el procedimiento de la copia, recuperar las fichas y
   repetir únicamente las lecturas afectadas.

El positivo en Chrome se comprueba junto con F1-129. Los resultados de las
pruebas focales y de CI se registran aparte; no sustituyen ese recorrido.
