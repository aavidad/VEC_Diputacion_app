\set ON_ERROR_STOP on
BEGIN;
SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
DO $ensayo$
DECLARE n integer; primera record; segunda record; rechazado boolean;
BEGIN
 SELECT count(*) INTO n FROM public.b56_consultar('participacion:uno',100);
 IF n<>7 THEN RAISE EXCEPTION 'B56: filas inesperadas: %',n; END IF;
 SELECT count(*) INTO n FROM public.b56_consultar('participacion:uno',100) h
  WHERE (h.recibo_ref='recibo:s1' AND h.accion='pausar' AND h.campo='situacion')
     OR (h.recibo_ref='recibo:s2' AND h.accion='reactivar' AND h.campo='situacion');
 IF n<>2 THEN RAISE EXCEPTION 'B56: operaciones B8 no están en las filas de cambio'; END IF;
 SELECT count(*) INTO n FROM public.b56_consultar('participacion:uno',100) h
  WHERE h.id LIKE 'cambio:%' AND (
   h.motivo IS DISTINCT FROM 'Motivo reservado en Bolsa'
   OR h.recibo_ref IS NULL OR h.valor_nuevo IS NULL
   OR h.motivo LIKE '%privado%' OR h.motivo LIKE '%personal%');
 IF n<>0 THEN RAISE EXCEPTION 'B56: cambio incompleto o motivo libre expuesto'; END IF;
 SELECT count(*) INTO n FROM public.b56_consultar('participacion:uno',100) h
  WHERE (h.recibo_ref='recibo:s2' AND h.motivo='Motivo reservado en Bolsa')
     OR (h.recibo_ref='recibo:situacion:constitucion:participacion:uno'
         AND h.motivo='Constitución de bolsa');
 IF n<>2 THEN RAISE EXCEPTION 'B56: motivo libre imita constitucion o falta origen estructural'; END IF;
 SELECT count(*) INTO n FROM public.b56_consultar('participacion:uno',100) h
  WHERE h.id LIKE 'cambio:%' AND h.recibo_ref NOT IN ('recibo:s1','recibo:s2','recibo:c1','recibo:c2');
 IF n<>0 THEN RAISE EXCEPTION 'B56: traza huérfana o ajena expuesta'; END IF;
 SELECT * INTO primera FROM public.b56_consultar('participacion:uno',1) LIMIT 1;
 SELECT * INTO segunda FROM public.b56_consultar('participacion:uno',1,primera.ocurrido_en,primera.id) LIMIT 1;
 IF primera.id=segunda.id OR primera.recibo_ref IS DISTINCT FROM 'recibo:c2'
    OR segunda.recibo_ref IS DISTINCT FROM 'recibo:c2'
    OR primera.motivo IS DISTINCT FROM 'Motivo reservado en Bolsa'
    OR segunda.motivo IS DISTINCT FROM 'Motivo reservado en Bolsa'
    OR primera.valor_nuevo IS DISTINCT FROM 'version:2'
    OR segunda.valor_nuevo IS DISTINCT FROM 'version:2' THEN
  RAISE EXCEPTION 'B56: cursor límite 1 pierde datos del cambio';
 END IF;
 SELECT count(*) INTO n FROM public.b56_consultar('participacion:ajena',100) h
  WHERE h.expediente_ref<>'participacion:ajena';
 IF n<>0 THEN RAISE EXCEPTION 'B56: participación ajena mezclada'; END IF;
 rechazado:=false;
 BEGIN
  PERFORM * FROM public.b56_consultar('participacion:uno',1,NULL,NULL,'bolsa.situacion_participacion.cambiar');
 EXCEPTION WHEN insufficient_privilege THEN rechazado:=true; END;
 IF NOT rechazado THEN RAISE EXCEPTION 'B56: acción de mutación aceptada'; END IF;
END $ensayo$;
COMMIT;
