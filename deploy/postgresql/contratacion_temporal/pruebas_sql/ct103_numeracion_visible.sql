BEGIN;
DO $prueba$
DECLARE primero text; segundo text; otro_anio text;
BEGIN
  SELECT vec_contratacion_temporal.siguiente_numero_visible_v1(2026) INTO primero;
  SELECT vec_contratacion_temporal.siguiente_numero_visible_v1(2026) INTO segundo;
  SELECT vec_contratacion_temporal.siguiente_numero_visible_v1(2027) INTO otro_anio;
  IF primero <> '2026/CT-000001' OR segundo <> '2026/CT-000002'
     OR otro_anio <> '2027/CT-000001' THEN
    RAISE EXCEPTION 'numeración CT103 inválida: %, %, %', primero, segundo, otro_anio;
  END IF;
END $prueba$;
ROLLBACK;
