# Ensayo sintético de gobierno RPT con V3/COSE

Este directorio contiene una puerta de fuente de solo lectura. Ejecución desde
la raíz del worktree:

```bash
python3 scripts/ensayos/rpt_gobierno_v3/preflight.py
```

La salida `bloqueado` y el código 2 son el resultado esperado mientras falte
la composición ADMIN #211. El guion fija la candidata, la fuente de las 41 SQL
previas y Cat4/AD134; coteja los bytes de estas dos migraciones y busca un
ensamblaje real de las rutas. No instala SQL, no toca un clon ni emite una
concesión. El plan H6 citado solo acredita el inventario previo: omite Cat4 y
AD134. Cada instalación posterior requiere un clon H1 nuevo, su historial
transaccional exacto y los dos artefactos SQL revisados.

El ensayo completo de servicio/PostgreSQL aún requiere tres identidades de
prueba con perfil HIGH y capacidades V3 frescas emitidas por la autoridad
central: editor, segundo aprobador y confirmador ajeno al editor. Debe probar
propuesta, dos aprobaciones distintas, confirmación, denegaciones de actor,
huella y capacidad revocada, auditoría central y recuperación tras reinicio
con igual recibo, fecha y estado. Esas credenciales no pueden deducirse del
cuerpo HTTP ni de un certificado sintético por sí solo. #211 debe aportar
listener ADMIN mTLS, fuente de credenciales y política nominal; hasta entonces
no hay recorrido HTTP autoritativo. Ningún resultado aquí acredita firma legal.
