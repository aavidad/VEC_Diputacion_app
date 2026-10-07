package main

import (
	"encoding/json"
	"io"
	"time"

	"vec-diputacion-granada/internal/shared/i18n"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
)

func ejecutarPlanGobiernoPerfil(ruta string, catalogo domain.CatalogoAccionesAdministracionV1, instante time.Time, salida io.Writer, traductor *i18n.Catalog, idioma string) int {
	var intencion domain.SolicitudPlanGobiernoPerfil
	if leerJSON(ruta, &intencion, 4<<20) != nil {
		return emitirPlanGobiernoPerfil(salida, traductor, idioma, nil, "", errEntrada)
	}
	plan, err := application.PrepararPlanGobiernoPerfilOffline(catalogo, intencion, instante)
	if err != nil {
		return emitirPlanGobiernoPerfil(salida, traductor, idioma, nil, "", err)
	}
	h, err := plan.HuellaSHA256()
	if err != nil {
		return emitirPlanGobiernoPerfil(salida, traductor, idioma, nil, "", err)
	}
	return emitirPlanGobiernoPerfil(salida, traductor, idioma, &plan, h, nil)
}

func emitirPlanGobiernoPerfil(salida io.Writer, traductor *i18n.Catalog, idioma string, plan *domain.PlanGobiernoPerfil, huella string, fallo error) int {
	codigo, exitCode := "admin_gobierno_plan_preparado", 0
	if fallo != nil {
		codigo, exitCode = fallo.Error(), 1
	}
	resultado := salidaComprobador{Comprobado: fallo == nil, Codigo: codigo, Mensaje: traductor.T(idioma, codigo),
		Limite: traductor.T(idioma, "admin_gobierno_plan_limite"), Plan: plan, PlanHuellaSHA256: huella}
	if json.NewEncoder(salida).Encode(resultado) != nil {
		return 2
	}
	return exitCode
}
