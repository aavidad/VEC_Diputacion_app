# Regla común de funciones PostgreSQL

Toda función o procedimiento `SECURITY DEFINER` nuevo o reconstruido debe declarar
`SET search_path = pg_catalog, pg_temp`, con `pg_temp` explícito al final. La
puerta `scripts/verificar_search_path_definer.py` comprueba el SQL cambiado
respecto al punto de bifurcación de la rama base. Las funciones históricas
sin cambios quedan fuera de este corte; una reconstrucción dinámica que no
permita verificar la definición final detiene la puerta y exige revisión SQL.
