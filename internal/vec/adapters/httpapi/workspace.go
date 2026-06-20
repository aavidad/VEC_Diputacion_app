package httpapi

import (
	"context"
	"net/http"
	"time"

	bolsamodule "vec-diputacion-granada/internal/modules/bolsa"
	cronosmodule "vec-diputacion-granada/internal/modules/cronos"
	cronosmemory "vec-diputacion-granada/internal/modules/cronos/adapters/memory"
	cronosapp "vec-diputacion-granada/internal/modules/cronos/application"
	cronosdomain "vec-diputacion-granada/internal/modules/cronos/domain"
	dietasmodule "vec-diputacion-granada/internal/modules/dietas"
	personalmodule "vec-diputacion-granada/internal/modules/personal"
	"vec-diputacion-granada/internal/vec/domain"
)

func (h *Handler) handleWorkspace(w http.ResponseWriter, r *http.Request) {
	if !h.requireMethod(w, r, http.MethodGet) {
		return
	}
	modules, err := h.service.Modules(r.Context())
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, workspaceSnapshotWithCronos(r.Context(), modules, h.cronos))
}

func workspaceSnapshot(modules []domain.ModuleManifest) map[string]any {
	cronosService, err := newWorkspaceCronosService()
	if err != nil {
		return workspaceSnapshotWithCronos(context.Background(), modules, nil)
	}
	return workspaceSnapshotWithCronos(context.Background(), modules, cronosService)
}

func workspaceSnapshotWithCronos(ctx context.Context, modules []domain.ModuleManifest, cronosService *cronosapp.Service) map[string]any {
	cronosData := workspaceCronosData(ctx, cronosService)
	return map[string]any{
		"generated_at": time.Now().UTC(),
		"modules":      modules,
		"kpis": []map[string]any{
			{"id": "plantilla", "label": "Personas activas", "value": 1248, "note": "Puestos, situaciones y nomina"},
			{"id": "horarios", "label": "Perfiles horarios", "value": 12, "note": "8 flexibles - 4 con cobertura fija"},
			{"id": "dietas", "label": "Dietas pendientes", "value": 11, "note": "4 completas - 7 medias dietas"},
			{"id": "rutas", "label": "Km provincia", "value": "642 km", "note": "Rutas estimadas para liquidacion"},
		},
		"screen_catalog": workspaceScreenCatalog(),
		"payroll_run":    workspacePayrollRun(),
		"expense_policy": workspaceExpensePolicy(),
		"action_catalog": workspaceActionCatalog(),
		"flow_states":    workspaceFlowStates(),
		"access_roles":   workspaceAccessRoles(),
		"role_assignments": []map[string]any{
			{"user": "demo.admin", "display_name": "Administrador VEC", "roles": []string{"administrador"}, "units": []string{"Diputacion"}, "state": "Activo", "mfa": "Certificado/Clave"},
			{"user": "demo.rrhh.tecnico", "display_name": "Tecnico RRHH", "roles": []string{"tecnico_rrhh"}, "units": []string{"Recursos Humanos"}, "state": "Activo", "mfa": "Certificado"},
			{"user": "demo.servicio", "display_name": "Jefatura de Servicio", "roles": []string{"jefe_servicio"}, "units": []string{"Asistencia a Municipios"}, "state": "Activo", "mfa": "Clave"},
			{"user": "demo.seccion", "display_name": "Jefatura de Seccion", "roles": []string{"jefe_seccion"}, "units": []string{"Centro servicios sociales"}, "state": "Suplencia vigente", "mfa": "Clave"},
			{"user": "demo.administrativo", "display_name": "Administrativo unidad", "roles": []string{"administrativo"}, "units": []string{"Administracion General"}, "state": "Activo", "mfa": "Clave"},
		},
		"rpt_catalog":             workspaceRPTCatalog(),
		"rpt_contract_types":      workspaceRPTContractTypes(),
		"rpt_position_samples":    workspaceRPTPositionSamples(),
		"professional_categories": workspaceProfessionalCategories(),
		"professional_category_aliases": []map[string]any{
			{"alias": "cocinero-00001", "category_slug": "cocinero", "source": "USO/OPES seed"},
			{"alias": "tecnico-superior-archivo-00001", "category_slug": "tecnico-superior-archivo", "source": "USO/OPES seed"},
			{"alias": "tecnico-de-administracion-general-00001", "category_slug": "tecnico-de-administracion-general", "source": "USO/OPES seed"},
		},
		"bolsa_category_rules": []map[string]any{
			{"id": "experiencia_misma_categoria", "label": "Experiencia en misma categoria", "points_per_month": 0.2, "section": "experiencia", "source": "Bolsa baremo rules"},
			{"id": "experiencia_otra_categoria", "label": "Experiencia en otra categoria", "points_per_month": 0.1, "section": "experiencia", "source": "Bolsa baremo rules"},
		},
		"operational_records": []map[string]any{
			{
				"id":        "PERSONAL-2026-0001",
				"module_id": personalmodule.ModuleID,
				"module":    "Personal",
				"scope":     "Expediente de empleado",
				"title":     "Variacion de puesto y unidad organica",
				"employee":  "EMP-0042 - Tecnico/a provincia",
				"state":     "Pendiente validar RRHH",
				"deadline":  "Hoy 13:00",
				"metric":    "Puesto 24-AG",
				"document":  "Resolucion interna",
				"action":    "Validar puesto",
			},
			{
				"id":        "NOMINA-2026-0042",
				"module_id": personalmodule.ModuleID,
				"module":    "Personal",
				"scope":     "Nomina e incidencias",
				"title":     "Incidencia retributiva vinculada a reduccion 63/64",
				"employee":  "EMP-0064 - Ordenanza",
				"state":     "Cruce Cronos pendiente",
				"deadline":  "Cierre nomina 25/06",
				"metric":    "2 conceptos",
				"document":  "Precalculo nomina",
				"action":    "Revisar nomina",
			},
			{
				"id":        "PERSONAL-2026-0109",
				"module_id": personalmodule.ModuleID,
				"module":    "Personal",
				"scope":     "Situacion administrativa",
				"title":     "Comision de servicio enlazada con Dietas",
				"employee":  "EMP-0054 - Servicio Provincial",
				"state":     "Activa",
				"deadline":  "Sin vencimiento critico",
				"metric":    "15 dias",
				"document":  "Nombramiento temporal",
				"action":    "Abrir expediente",
			},
			{
				"id":        "ANTIG-2026-0006",
				"module_id": personalmodule.ModuleID,
				"module":    "Personal",
				"scope":     "Antiguedad y trienios",
				"title":     "Calculo automatico de antiguedad",
				"employee":  "EMP-0088 - Secretaria Intervencion",
				"state":     "Calculado",
				"deadline":  "Cierre nomina 25/06",
				"metric":    "6 trienios",
				"document":  "Base para permisos y nomina",
				"action":    "Recalcular",
			},
			{
				"id":        "SERV-2026-0032",
				"module_id": personalmodule.ModuleID,
				"module":    "Personal",
				"scope":     "Servicios prestados",
				"title":     "Servicios prestados en Diputacion",
				"employee":  "EMP-0031 - Area Obras",
				"state":     "Disponible para certificado",
				"deadline":  "Sin vencimiento critico",
				"metric":    "14a 3m 12d",
				"document":  "Historial normalizado",
				"action":    "Generar certificado",
			},
			{
				"id":        "CERT-2026-0014",
				"module_id": personalmodule.ModuleID,
				"module":    "Personal",
				"scope":     "Certificados",
				"title":     "Certificado automatico para meritos Bolsa",
				"employee":  "EMP-0031 - Area Obras",
				"state":     "Pendiente firma",
				"deadline":  "Vence en 72 h",
				"metric":    "PDF/CSV",
				"document":  "Servicios prestados",
				"action":    "Emitir certificado",
			},
			{
				"id":        "HORARIO-2026-0002",
				"module_id": cronosmodule.ModuleID,
				"module":    "Cronos",
				"scope":     "Horarios del personal",
				"title":     "Perfil flexible administrativo",
				"employee":  "Unidad Administracion General",
				"state":     "Flexible con tramo obligatorio",
				"deadline":  "Vigente desde 01/01/2026",
				"metric":    "37h 30m",
				"document":  "Entrada 07:30-09:30",
				"action":    "Editar perfil",
			},
			{
				"id":        "HORARIO-2026-0008",
				"module_id": cronosmodule.ModuleID,
				"module":    "Cronos",
				"scope":     "Horarios del personal",
				"title":     "Atencion directa a personas mayores",
				"employee":  "Centro servicios sociales",
				"state":     "Sin flexibilidad",
				"deadline":  "Cobertura obligatoria 08:00-15:00",
				"metric":    "Turno fijo",
				"document":  "Servicio presencial",
				"action":    "Revisar cobertura",
			},
			{
				"id":        "REDUCCION-2026-0063",
				"module_id": cronosmodule.ModuleID,
				"module":    "Cronos",
				"scope":     "Prejubilacion 63/64",
				"title":     "Reduccion diaria por 63 anos",
				"employee":  "EMP-0063 - Administrativo/a",
				"state":     "1 hora menos diaria",
				"deadline":  "Validar antiguedad",
				"metric":    "6h 30m objetivo",
				"document":  "Permiso RRHH",
				"action":    "Validar reduccion",
			},
			{
				"id":        "REDUCCION-2026-0064",
				"module_id": cronosmodule.ModuleID,
				"module":    "Cronos",
				"scope":     "Prejubilacion 63/64",
				"title":     "Reduccion diaria por 64 anos",
				"employee":  "EMP-0064 - Ordenanza",
				"state":     "2 horas menos diarias",
				"deadline":  "Aplicar al cuadrante",
				"metric":    "5h 30m objetivo",
				"document":  "Permiso RRHH",
				"action":    "Aplicar reduccion",
			},
			{
				"id":        "CRONOS-2026-0007",
				"module_id": cronosmodule.ModuleID,
				"module":    "Cronos",
				"scope":     "Fichajes e incidencias",
				"title":     "Fichaje sin salida - Asistencia a Municipios",
				"employee":  "EMP-0042 - Tecnico/a provincia",
				"state":     "Incidencia pendiente",
				"deadline":  "Hoy 14:00",
				"metric":    "7h 12m",
				"document":  "Justificante requerido",
				"action":    "Justificar",
			},
			{
				"id":        "CRONOS-2026-0013",
				"module_id": cronosmodule.ModuleID,
				"module":    "Cronos",
				"scope":     "Permisos y vacaciones",
				"title":     "Asuntos propios solicitados",
				"employee":  "EMP-0088 - Secretaria Intervencion",
				"state":     "Pendiente aprobacion",
				"deadline":  "Vence en 72 h",
				"metric":    "2 dias",
				"document":  "Saldo 4 dias",
				"action":    "Aprobar",
			},
			{
				"id":        "CRONOS-2026-0021",
				"module_id": cronosmodule.ModuleID,
				"module":    "Cronos",
				"scope":     "Vacaciones",
				"title":     "Solicitud vacaciones agosto",
				"employee":  "EMP-0026 - Oficina Comarcal",
				"state":     "Solape detectado",
				"deadline":  "Seguimiento ordinario",
				"metric":    "10 dias",
				"document":  "Calendario unidad",
				"action":    "Revisar",
			},
			{
				"id":        "DIETAS-2026-0044",
				"module_id": dietasmodule.ModuleID,
				"module":    "Dietas",
				"scope":     "Comision de servicio",
				"title":     "Granada - Motril - asistencia tecnica",
				"employee":  "EMP-0031 - Area Obras",
				"state":     "Ruta pendiente validar",
				"deadline":  "Hoy 15:00",
				"metric":    "140,8 km",
				"document":  "Media dieta",
				"action":    "Validar km",
			},
			{
				"id":        "DIETAS-2026-0052",
				"module_id": dietasmodule.ModuleID,
				"module":    "Dietas",
				"scope":     "Dieta completa",
				"title":     "Granada - Baza - jornada completa",
				"employee":  "EMP-0054 - Servicio Provincial",
				"state":     "Justificante completo",
				"deadline":  "Sin vencimiento critico",
				"metric":    "214,4 km",
				"document":  "Dieta completa",
				"action":    "Liquidar",
			},
			{
				"id":        "DIETAS-2026-0061",
				"module_id": dietasmodule.ModuleID,
				"module":    "Dietas",
				"scope":     "Mapa provincial",
				"title":     "Ruta con parada: Granada - Guadix - Baza",
				"employee":  "EMP-0019 - Inspeccion",
				"state":     "Politica excedida",
				"deadline":  "Plazo vencido",
				"metric":    "168,6 km",
				"document":  "Motivo requerido",
				"action":    "Resolver",
			},
			{
				"id":        "BOLSA-2026-0142",
				"module_id": bolsamodule.ModuleID,
				"module":    "Bolsa",
				"scope":     "Seleccion y bolsas",
				"title":     "Solicitud con subsanacion",
				"employee":  "CAND-0007 - DNI ***4567*",
				"state":     "Subsanacion requerida",
				"deadline":  "Vence en 72 h",
				"metric":    "61,4 pt",
				"document":  "CSV pendiente",
				"action":    "Revisar",
			},
		},
		"province_routes": []map[string]any{
			{"id": "R-GR-MOTRIL", "from": "Granada", "to": "Motril", "km_one_way": 70.4, "estimated_minutes": 55, "allowance": "media dieta"},
			{"id": "R-GR-LOJA", "from": "Granada", "to": "Loja", "km_one_way": 54.0, "estimated_minutes": 45, "allowance": "sin dieta por defecto"},
			{"id": "R-GR-GUADIX", "from": "Granada", "to": "Guadix", "km_one_way": 60.5, "estimated_minutes": 50, "allowance": "media dieta si hay manutencion"},
			{"id": "R-GR-BAZA", "from": "Granada", "to": "Baza", "km_one_way": 107.2, "estimated_minutes": 80, "allowance": "dieta completa segun horario"},
			{"id": "R-GR-ALMUNECAR", "from": "Granada", "to": "Almunecar", "km_one_way": 80.5, "estimated_minutes": 65, "allowance": "media dieta"},
		},
		"schedule_profiles": cronosData["schedule_profiles"],
		"cronos_sections": []string{
			"Saldo Horario",
			"Movimientos",
			"Permisos y Licencias",
			"Notificaciones",
			"Configuracion",
			"Mensajes",
			"Incidencias / Acumulados",
		},
		"cronos_daily_summary":       cronosData["daily_summary"],
		"cronos_timecards":           cronosData["timecards"],
		"cronos_reductions":          cronosData["reduction_cases"],
		"cronos_leave_policies":      cronosData["leave_policies"],
		"cronos_leave_requests":      cronosData["leave_requests"],
		"cronos_permission_balances": cronosData["permission_balances"],
	}
}

var workspaceCronosDate = time.Date(2026, time.June, 19, 0, 0, 0, 0, time.UTC)

func newWorkspaceCronosService() (*cronosapp.Service, error) {
	service, err := cronosapp.NewService(cronosmemory.NewStore())
	if err != nil {
		return nil, err
	}
	if err := seedWorkspaceCronos(context.Background(), service); err != nil {
		return nil, err
	}
	return service, nil
}

func seedWorkspaceCronos(ctx context.Context, service *cronosapp.Service) error {
	profiles := workspaceCronosProfiles()
	for _, profile := range profiles {
		if err := service.SaveProfile(ctx, profile); err != nil {
			return err
		}
	}
	for _, workday := range workspaceCronosWorkdays(workspaceCronosDate, profiles[0], profiles[1]) {
		if err := service.RegisterWorkday(ctx, workday); err != nil {
			return err
		}
	}
	for _, policy := range cronosdomain.DefaultLeavePolicies() {
		if err := service.SaveLeavePolicy(ctx, policy); err != nil {
			return err
		}
	}
	for _, balance := range workspaceCronosLeaveBalances() {
		if err := service.SaveLeaveBalance(ctx, balance); err != nil {
			return err
		}
	}
	for _, request := range workspaceCronosLeaveRequests() {
		if _, err := service.RequestLeave(ctx, request); err != nil {
			return err
		}
	}
	return nil
}

func workspaceCronosProfiles() []cronosdomain.ScheduleProfile {
	return []cronosdomain.ScheduleProfile{
		{
			ID:               "H-FLEX-ADM",
			Name:             "Flexible administrativo",
			Unit:             "Administracion General",
			Flexible:         true,
			AllowsTelework:   true,
			RequiresCoverage: false,
			DailyTarget:      cronosdomain.Minutes(7, 30),
			WeeklyTarget:     cronosdomain.Minutes(37, 30),
			EntryWindowStart: cronosdomain.Minutes(7, 30),
			EntryWindowEnd:   cronosdomain.Minutes(9, 30),
			CoreStart:        cronosdomain.Minutes(9, 30),
			CoreEnd:          cronosdomain.Minutes(14, 0),
		},
		{
			ID:               "H-FIJO-MAYORES",
			Name:             "Atencion personas mayores",
			Unit:             "Centro servicios sociales",
			Flexible:         false,
			AllowsTelework:   false,
			RequiresCoverage: true,
			DailyTarget:      cronosdomain.Minutes(7, 30),
			WeeklyTarget:     cronosdomain.Minutes(37, 30),
			EntryWindowStart: cronosdomain.Minutes(8, 0),
			EntryWindowEnd:   cronosdomain.Minutes(15, 30),
		},
	}
}

func workspaceCronosWorkdays(day time.Time, flexProfile, fixedProfile cronosdomain.ScheduleProfile) []cronosdomain.Workday {
	return []cronosdomain.Workday{
		{
			EmployeeID:       "EMP-0031",
			Date:             day,
			Age:              41,
			Profile:          flexProfile,
			TeleworkDuration: cronosdomain.Minutes(7, 30),
		},
		{
			EmployeeID: "EMP-0063",
			Date:       day,
			Age:        63,
			Profile:    flexProfile,
			Punches: []cronosdomain.Punch{
				{At: atDemo(day, 8, 0), Kind: cronosdomain.PunchEntry, Channel: "web", Mode: cronosdomain.WorkModeOnSite},
				{At: atDemo(day, 14, 30), Kind: cronosdomain.PunchExit, Channel: "web", Mode: cronosdomain.WorkModeOnSite},
			},
		},
		{
			EmployeeID: "EMP-0064",
			Date:       day,
			Age:        64,
			Profile:    flexProfile,
			Punches: []cronosdomain.Punch{
				{At: atDemo(day, 8, 0), Kind: cronosdomain.PunchEntry, Channel: "terminal", Mode: cronosdomain.WorkModeOnSite},
				{At: atDemo(day, 13, 30), Kind: cronosdomain.PunchExit, Channel: "terminal", Mode: cronosdomain.WorkModeOnSite},
			},
		},
		{
			EmployeeID: "EMP-0042",
			Date:       day,
			Age:        39,
			Profile:    flexProfile,
			Punches: []cronosdomain.Punch{
				{At: atDemo(day, 7, 55), Kind: cronosdomain.PunchEntry, Channel: "web", Mode: cronosdomain.WorkModeOnSite},
			},
		},
		{
			EmployeeID: "EMP-0100",
			Date:       day,
			Age:        45,
			Profile:    fixedProfile,
			Punches: []cronosdomain.Punch{
				{At: atDemo(day, 9, 0), Kind: cronosdomain.PunchEntry, Channel: "terminal", Mode: cronosdomain.WorkModeOnSite},
				{At: atDemo(day, 15, 30), Kind: cronosdomain.PunchExit, Channel: "terminal", Mode: cronosdomain.WorkModeOnSite},
			},
		},
	}
}

func workspaceCronosLeaveBalances() []cronosdomain.LeaveBalance {
	year := workspaceCronosDate.Year()
	return []cronosdomain.LeaveBalance{
		{EmployeeID: "EMP-0031", Year: year, PolicyID: "asuntos_propios", Granted: 6, Consumed: 2},
		{EmployeeID: "EMP-0031", Year: year, PolicyID: "vacaciones", Granted: 22, Approved: 20, Consumed: 2},
		{EmployeeID: "EMP-0031", Year: year, PolicyID: "bolsa_conciliacion", Granted: 30 * 60, Consumed: 16*60 + 52},
		{EmployeeID: "EMP-0031", Year: year, PolicyID: "compensacion_horaria", Granted: 19},
		{EmployeeID: "EMP-0031", Year: year, PolicyID: "horas_sindicales", Granted: 60 * 60, Consumed: 58*60 + 30},
		{EmployeeID: "EMP-0031", Year: year, PolicyID: "medico", Granted: 3 * 60},
		{EmployeeID: "EMP-0031", Year: year, PolicyID: "enfermedad_sin_baja", Granted: 4, Consumed: 1},
		{EmployeeID: "EMP-0031", Year: year, PolicyID: "gestion_servicio", Granted: 999 * 60},
		{EmployeeID: "EMP-0088", Year: year, PolicyID: "asuntos_propios", Granted: 6, Consumed: 1},
		{EmployeeID: "EMP-0088", Year: year, PolicyID: "vacaciones", Granted: 22, Approved: 8},
	}
}

func workspaceCronosLeaveRequests() []cronosdomain.LeaveRequest {
	day := workspaceCronosDate
	return []cronosdomain.LeaveRequest{
		{
			ID:         "CRONOS-ABS-2026-0001",
			EmployeeID: "EMP-0031",
			PolicyID:   "asuntos_propios",
			From:       day.AddDate(0, 0, 4),
			To:         day.AddDate(0, 0, 4),
			Amount:     1,
			Unit:       cronosdomain.LeaveUnitDay,
			Reason:     "Asunto propio",
			State:      cronosdomain.LeaveStateReview,
			CreatedAt:  day.Add(-48 * time.Hour),
		},
		{
			ID:          "CRONOS-ABS-2026-0002",
			EmployeeID:  "EMP-0031",
			PolicyID:    "medico",
			From:        time.Date(2026, time.June, 24, 10, 0, 0, 0, time.UTC),
			To:          time.Date(2026, time.June, 24, 11, 0, 0, 0, time.UTC),
			Amount:      60,
			Unit:        cronosdomain.LeaveUnitHour,
			Reason:      "Consulta medica",
			DocumentRef: "DOC-MED-2026-0002",
			State:       cronosdomain.LeaveStateReview,
			CreatedAt:   day.Add(-24 * time.Hour),
		},
	}
}

func workspaceCronosData(ctx context.Context, service *cronosapp.Service) map[string]any {
	if service == nil {
		var err error
		service, err = newWorkspaceCronosService()
		if err != nil {
			return emptyCronosData()
		}
	}
	snapshot, err := service.Snapshot(ctx, workspaceCronosDate)
	if err != nil {
		return emptyCronosData()
	}
	summary := firstCronosResult(snapshot.Results, "EMP-0031")
	profiles := make([]map[string]any, 0, len(snapshot.Profiles)+2)
	for _, profile := range snapshot.Profiles {
		profiles = append(profiles, scheduleProfileView(profile, 0))
	}
	reductionBase := firstFlexibleProfile(snapshot.Profiles)
	if reductionBase.ID != "" {
		profiles = append(profiles,
			reductionProfileView("H-RED-63", "Prejubilacion 63 anos", reductionBase, 63),
			reductionProfileView("H-RED-64", "Prejubilacion 64 anos", reductionBase, 64),
		)
	}
	timecards := make([]map[string]any, 0, len(snapshot.Results))
	reductionCases := make([]map[string]any, 0, 2)
	for _, result := range snapshot.Results {
		timecards = append(timecards, timecardView(
			timecardID(result.EmployeeID),
			timecardTitle(result),
			result,
			timecardState(result),
		))
		if result.Reduction > 0 {
			reductionCases = append(reductionCases, reductionCaseView(result, reductionAction(result)))
		}
	}
	return map[string]any{
		"schedule_profiles": profiles,
		"leave_policies":    leavePolicyViews(snapshot.LeavePolicies),
		"leave_requests":    leaveRequestViews(snapshot.LeaveRequests, snapshot.LeavePolicies),
		"permission_balances": leaveBalanceViews(
			snapshot.LeaveBalances,
			snapshot.LeavePolicies,
			"EMP-0031",
		),
		"daily_summary": map[string]any{
			"date":             workspaceCronosDate.Format("2006-01-02"),
			"theoretical":      cronosdomain.FormatHHMM(summary.Theoretical),
			"worked":           cronosdomain.FormatHHMM(summary.OnSiteWorked),
			"telework":         cronosdomain.FormatHHMM(summary.Telework),
			"daily_balance":    cronosdomain.FormatHHMM(summary.Balance),
			"period_from":      "2026-06-01",
			"period_to":        "2026-06-18",
			"previous_balance": "00:00",
			"period_expected":  "74:30",
			"period_worked":    "62:26",
			"period_telework":  "07:30",
			"period_balance":   "-04:34",
			"engine":           "cronos.application.Service.Snapshot",
			"store":            "cronos.adapters.memory.Store",
		},
		"timecards":       timecards,
		"reduction_cases": reductionCases,
	}
}

func emptyCronosData() map[string]any {
	return map[string]any{
		"schedule_profiles": []map[string]any{},
		"daily_summary":     map[string]any{"engine": "cronos.application.Service.Snapshot", "store": "unavailable"},
		"timecards":         []map[string]any{},
		"reduction_cases":   []map[string]any{},
	}
}

func firstCronosResult(results []cronosdomain.DayResult, employeeID string) cronosdomain.DayResult {
	for _, result := range results {
		if result.EmployeeID == employeeID {
			return result
		}
	}
	if len(results) > 0 {
		return results[0]
	}
	return cronosdomain.DayResult{Date: workspaceCronosDate}
}

func firstFlexibleProfile(profiles []cronosdomain.ScheduleProfile) cronosdomain.ScheduleProfile {
	for _, profile := range profiles {
		if profile.Flexible {
			return profile
		}
	}
	if len(profiles) > 0 {
		return profiles[0]
	}
	return cronosdomain.ScheduleProfile{}
}

func timecardID(employeeID string) string {
	if len(employeeID) > 4 && employeeID[:4] == "EMP-" {
		return "CRONOS-TC-" + employeeID[4:]
	}
	return "CRONOS-TC-" + employeeID
}

func timecardTitle(result cronosdomain.DayResult) string {
	switch {
	case result.OpenEntry:
		return "Fichaje sin salida"
	case result.FixedCoverageBreach:
		return "Cobertura fija incumplida"
	case result.Reduction >= 2*time.Hour:
		return "Reduccion 64 aplicada"
	case result.Reduction >= time.Hour:
		return "Reduccion 63 aplicada"
	case result.Telework > 0:
		return "Teletrabajo validado"
	default:
		return "Jornada evaluada"
	}
}

func timecardState(result cronosdomain.DayResult) string {
	switch {
	case result.OpenEntry:
		return "Incidencia pendiente"
	case result.FixedCoverageBreach || hasBlockingIncident(result.Incidents):
		return "Bloqueante"
	case result.Reduction > 0:
		return "Cuadrado"
	default:
		return "Al dia"
	}
}

func hasBlockingIncident(incidents []cronosdomain.Incident) bool {
	for _, incident := range incidents {
		if incident.Severity == cronosdomain.IncidentBlocking {
			return true
		}
	}
	return false
}

func reductionAction(result cronosdomain.DayResult) string {
	if result.Age >= 64 {
		return "Aplicar reduccion 64"
	}
	return "Validar reduccion 63"
}

func leavePolicyViews(policies []cronosdomain.LeavePolicy) []map[string]any {
	views := make([]map[string]any, 0, len(policies))
	for _, policy := range policies {
		views = append(views, map[string]any{
			"id":                policy.ID,
			"name":              policy.Name,
			"unit":              policy.Unit,
			"annual_allowance":  cronosdomain.FormatLeaveAmount(policy.AnnualAllowance, policy.Unit),
			"request":           policy.Requestable,
			"requires_document": policy.RequiresDocument,
			"requires_approval": policy.RequiresApproval,
			"payroll_impact":    policy.PayrollImpact,
			"min_request":       cronosdomain.FormatLeaveAmount(policy.MinRequest, policy.Unit),
			"max_request":       cronosdomain.FormatLeaveAmount(policy.MaxRequest, policy.Unit),
		})
	}
	return views
}

func leaveBalanceViews(balances []cronosdomain.LeaveBalance, policies []cronosdomain.LeavePolicy, employeeID string) []map[string]any {
	policyByID := leavePoliciesByID(policies)
	views := make([]map[string]any, 0, len(balances))
	for _, balance := range balances {
		if employeeID != "" && balance.EmployeeID != employeeID {
			continue
		}
		policy, ok := policyByID[balance.PolicyID]
		if !ok {
			continue
		}
		views = append(views, map[string]any{
			"id":          balance.PolicyID,
			"employee":    balance.EmployeeID,
			"name":        policy.Name,
			"unit":        policy.Unit,
			"request":     policy.Requestable,
			"max":         cronosdomain.FormatLeaveAmount(balance.Granted, policy.Unit),
			"min":         cronosdomain.FormatLeaveAmount(policy.MinRequest, policy.Unit),
			"requested":   cronosdomain.FormatLeaveAmount(balance.Requested, policy.Unit),
			"approved":    cronosdomain.FormatLeaveAmount(balance.Approved, policy.Unit),
			"consumed":    cronosdomain.FormatLeaveAmount(balance.Consumed, policy.Unit),
			"remaining":   cronosdomain.FormatLeaveAmount(balance.Remaining(), policy.Unit),
			"document":    policy.RequiresDocument,
			"payroll":     policy.PayrollImpact,
			"next_action": "Solicitar",
		})
	}
	return views
}

func leaveRequestViews(requests []cronosdomain.LeaveRequest, policies []cronosdomain.LeavePolicy) []map[string]any {
	policyByID := leavePoliciesByID(policies)
	views := make([]map[string]any, 0, len(requests))
	for _, request := range requests {
		policy := policyByID[request.PolicyID]
		views = append(views, map[string]any{
			"id":           request.ID,
			"employee":     request.EmployeeID,
			"policy_id":    request.PolicyID,
			"name":         policy.Name,
			"from":         request.From.Format("2006-01-02"),
			"to":           request.To.Format("2006-01-02"),
			"amount":       cronosdomain.FormatLeaveAmount(request.Amount, request.Unit),
			"unit":         request.Unit,
			"state":        request.State,
			"reason":       request.Reason,
			"document_ref": request.DocumentRef,
			"created_at":   request.CreatedAt.Format(time.RFC3339),
		})
	}
	return views
}

func leavePoliciesByID(policies []cronosdomain.LeavePolicy) map[string]cronosdomain.LeavePolicy {
	byID := make(map[string]cronosdomain.LeavePolicy, len(policies))
	for _, policy := range policies {
		byID[policy.ID] = policy
	}
	return byID
}

func scheduleProfileView(profile cronosdomain.ScheduleProfile, age int) map[string]any {
	return map[string]any{
		"id":            profile.ID,
		"name":          profile.Name,
		"unit":          profile.Unit,
		"flexible":      profile.Flexible,
		"telework":      profile.AllowsTelework,
		"coverage":      profile.RequiresCoverage,
		"entry_window":  timeRange(profile.EntryWindowStart, profile.EntryWindowEnd),
		"core_time":     timeRange(profile.CoreStart, profile.CoreEnd),
		"daily_target":  cronosdomain.FormatHHMM(profile.DailyTarget - cronosdomain.DailyReductionForAge(age)),
		"weekly_hours":  cronosdomain.FormatHHMM(profile.WeeklyTarget - 5*cronosdomain.DailyReductionForAge(age)),
		"reduction_age": age,
	}
}

func reductionProfileView(id, name string, base cronosdomain.ScheduleProfile, age int) map[string]any {
	view := scheduleProfileView(base, age)
	view["id"] = id
	view["name"] = name
	view["daily_reduction"] = cronosdomain.FormatHHMM(cronosdomain.DailyReductionForAge(age))
	return view
}

func timecardView(id, title string, result cronosdomain.DayResult, state string) map[string]any {
	return map[string]any{
		"id":          id,
		"title":       title,
		"employee":    result.EmployeeID,
		"date":        result.Date.Format("2006-01-02"),
		"profile":     result.ProfileID,
		"age":         result.Age,
		"theoretical": cronosdomain.FormatHHMM(result.Theoretical),
		"worked":      cronosdomain.FormatHHMM(result.Worked),
		"onsite":      cronosdomain.FormatHHMM(result.OnSiteWorked),
		"telework":    cronosdomain.FormatHHMM(result.Telework),
		"balance":     cronosdomain.FormatHHMM(result.Balance),
		"reduction":   cronosdomain.FormatHHMM(result.Reduction),
		"state":       state,
		"open_entry":  result.OpenEntry,
		"incidents":   incidentViews(result.Incidents),
	}
}

func reductionCaseView(result cronosdomain.DayResult, action string) map[string]any {
	return map[string]any{
		"employee":        result.EmployeeID,
		"age":             result.Age,
		"date":            result.Date.Format("2006-01-02"),
		"daily_reduction": cronosdomain.FormatHHMM(result.Reduction),
		"target":          cronosdomain.FormatHHMM(result.Theoretical),
		"worked":          cronosdomain.FormatHHMM(result.Worked),
		"balance":         cronosdomain.FormatHHMM(result.Balance),
		"state":           "Vigente",
		"payroll_effect":  "Pendiente nomina",
		"action":          action,
	}
}

func incidentViews(incidents []cronosdomain.Incident) []map[string]any {
	views := make([]map[string]any, 0, len(incidents))
	for _, incident := range incidents {
		views = append(views, map[string]any{
			"code":     incident.Code,
			"label":    incident.Label,
			"severity": incident.Severity,
		})
	}
	return views
}

func timeRange(start, end time.Duration) string {
	if start == 0 && end == 0 {
		return "-"
	}
	if end == 0 || start == end {
		return cronosdomain.FormatHHMM(start)
	}
	return cronosdomain.FormatHHMM(start) + "-" + cronosdomain.FormatHHMM(end)
}

func atDemo(day time.Time, hour, minute int) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, day.Location())
}

type workspaceScreenSpec struct {
	ID           string
	ModuleKey    string
	MenuID       string
	Title        string
	Description  string
	Fields       []string
	Actions      []string
	States       []string
	Integrations []string
	Validations  []string
	DoneCriteria string
}

func workspaceScreenCatalog() []map[string]any {
	specs := []workspaceScreenSpec{
		screenSpec("personal.dashboard", "personal", "Dashboard de Personal", "Prioriza altas, bajas, cambios de puesto, certificados e incidencias por unidad.", []string{"Unidad", "Expedientes", "Cambios puesto", "Certificados", "Incidencias"}, []string{"Alta empleado", "Abrir expediente", "Asignar responsable"}),
		screenSpec("personal.expedientes", "personal", "Expedientes de personal", "Ficha 360 del empleado publico con regimen, puesto, unidad, situacion e historial.", []string{"Empleado", "DNI", "Regimen", "Puesto", "Unidad", "Situacion", "Fecha efecto"}, []string{"Editar expediente", "Anexar resolucion", "Emitir cambio"}),
		screenSpec("personal.puestos", "personal", "Puestos y RPT", "Gestiona codigo RPT, tipo de puesto, adscripcion, forma de provision, grupo, nivel, complemento, teletrabajo y cobertura.", []string{"Codigo RPT", "Puesto", "Tp", "Ad", "Fp", "Grupo", "CD", "CE", "TA", "Cobertura"}, []string{"Importar RPT", "Versionar RPT", "Asociar empleado"}),
		screenSpec("personal.situaciones", "personal", "Situaciones administrativas", "Registra activo, baja, excedencia, servicios especiales, comision e interinidad.", []string{"Empleado", "Situacion", "Fecha inicio", "Fecha fin", "Resolucion", "Efecto nomina"}, []string{"Registrar situacion", "Corregir fecha", "Cerrar situacion"}),
		screenSpec("personal.antiguedad", "personal", "Antiguedad y trienios", "Calcula periodos reconocidos, trienios, proximo vencimiento e impacto retributivo.", []string{"Empleado", "Periodos", "Trienios", "Proximo trienio", "Fuente", "Impacto"}, []string{"Recalcular", "Revisar periodo", "Enviar a nomina"}),
		screenSpec("personal.servicios", "personal", "Servicios prestados", "Consolida periodos computables para certificados, meritos y trienios.", []string{"Empleado", "Periodo", "Categoria", "Unidad", "Jornada", "Computable Bolsa", "Computable trienio"}, []string{"Consolidar", "Corregir solape", "Certificar"}),
		screenSpec("personal.certificados", "personal", "Certificados", "Genera certificados reutilizables por empleado, modulo destino y version.", []string{"Tipo", "Empleado", "Datos incluidos", "Version", "Firma/CSV", "Destino", "Validez"}, []string{"Generar", "Firmar", "Revocar", "Enviar"}),

		screenSpec("nominas.cierre", "nominas", "Cierre de nomina", "Controla periodo, precalculo, incidencias, recibos, atrasos y cierre.", []string{"Periodo", "Empleados", "Bruto", "Liquido", "Incidencias", "Recibos", "Estado"}, []string{"Abrir periodo", "Precalcular", "Cerrar", "Publicar recibos"}),
		screenSpec("nominas.retribuciones", "nominas", "Retribuciones", "Mantiene conceptos economicos, vigencias y simulacion de impacto.", []string{"Concepto", "Tipo", "Importe", "Cotiza", "IRPF", "Vigencia", "Estado"}, []string{"Crear concepto", "Versionar tabla", "Simular impacto"}),
		screenSpec("nominas.incidencias", "nominas", "Incidencias retributivas", "Resuelve variaciones de Cronos, Dietas, puestos, atrasos, bajas y errores maestros.", []string{"Incidencia", "Empleado", "Origen", "Concepto", "Impacto", "Responsable", "Severidad"}, []string{"Resolver", "Devolver a origen", "Recalcular empleado"}),
		screenSpec("nominas.integraciones", "nominas", "Integraciones de nomina", "Vigila entradas/salidas de Cronos, Dietas, Personal, banco, contabilidad y recibos.", []string{"Sistema", "Lote", "Payload", "Totales", "Errores", "Reintentos", "SLA"}, []string{"Reprocesar", "Conciliar", "Bloquear cierre"}),

		screenSpec("cronos.dashboard", "cronos", "Dashboard Cronos", "Situacion diaria: saldo, fichajes, permisos, vacaciones, teletrabajo e incidencias.", []string{"Dia", "Saldo mes", "Fichajes abiertos", "Permisos", "Vacaciones", "Incidencias"}, []string{"Fichar", "Justificar", "Aprobar"}),
		screenSpec("cronos.fichajes", "cronos", "Fichajes", "Registra entrada, salida, pausas, canal, teletrabajo y correcciones.", []string{"Empleado", "Fecha", "Entrada", "Salida", "Pausas", "Canal", "Justificante"}, []string{"Fichar", "Alta manual", "Corregir", "Justificar"}),
		screenSpec("cronos.incidencias", "cronos", "Incidencias de jornada", "Resuelve sin salida, defecto/exceso, fuera de perfil, solapes y ausencias.", []string{"Empleado", "Fecha", "Tipo", "Severidad", "Impacto", "Documento", "Responsable"}, []string{"Justificar", "Aprobar", "Rechazar", "Enviar a nomina"}),
		screenSpec("horarios.perfiles", "horarios", "Perfiles horarios", "Define tramos, flexibilidad, cobertura obligatoria, vigencia y excepciones.", []string{"Perfil", "Unidad", "Flexible", "Entrada", "Tramo obligatorio", "Horas", "Vigencia"}, []string{"Crear perfil", "Asignar puesto", "Cerrar vigencia"}),
		screenSpec("horarios.reducciones", "horarios", "Reducciones 63/64", "Aplica una hora menos a 63 anos y dos horas menos a 64 anos.", []string{"Empleado", "Edad", "Fecha efecto", "Reduccion", "Perfil resultante", "Resolucion", "Impacto nomina"}, []string{"Validar RRHH", "Aplicar cuadrante", "Enviar a nomina"}),
		screenSpec("permisos.solicitudes", "permisos", "Permisos", "Gestiona asuntos propios, permisos, compensaciones y saldo restante.", []string{"Empleado", "Tipo", "Desde", "Hasta", "Solicitado", "Saldo", "Responsable"}, []string{"Solicitar", "Aprobar", "Denegar", "Adjuntar"}),
		screenSpec("permisos.vacaciones", "permisos", "Vacaciones", "Calendario con saldo anual, solapes de unidad y bloqueos.", []string{"Empleado", "Periodo", "Dias", "Saldo anual", "Solape", "Unidad", "Estado"}, []string{"Solicitar", "Aprobar", "Mover periodo", "Bloquear fechas"}),
		screenSpec("permisos.aprobaciones", "permisos", "Aprobaciones Cronos", "Bandeja de responsables para permisos, incidencias, vacaciones y cambios horarios.", []string{"Tipo", "Solicitante", "Plazo", "Impacto", "Documento", "Responsable", "Prioridad"}, []string{"Aprobar lote", "Rechazar", "Pedir subsanacion"}),
		screenSpec("permisos.saldos", "permisos", "Saldos", "Explica saldos diarios, mensuales y anuales de jornada y permisos.", []string{"Empleado", "Teoricas", "Trabajadas", "Permisos", "Exceso/defecto", "Saldo anterior", "Cierre"}, []string{"Recalcular", "Exportar", "Cerrar mes", "Reabrir"}),

		screenSpec("dietas.dashboard", "dietas", "Dashboard Dietas", "Prioriza comisiones, gastos fuera de politica, km a validar, liquidaciones e importes.", []string{"Unidad", "Pendientes", "Fuera politica", "Km validar", "Importe", "Liquidaciones"}, []string{"Nueva comision", "Aprobar", "Liquidar", "Exportar"}),
		screenSpec("dietas.comisiones", "dietas", "Comisiones de servicio", "Autoriza desplazamientos con motivo, trayecto, horario, centro coste y estado.", []string{"Empleado", "Motivo", "Origen", "Destino", "Inicio", "Fin", "Centro coste"}, []string{"Crear comision", "Aprobar", "Cancelar", "Vincular justificante"}),
		screenSpec("dietas.dietas", "dietas", "Dietas y per diem", "Calcula media dieta, dieta completa, alojamiento y manutencion segun politica.", []string{"Comision", "Salida", "Llegada", "Tipo dieta", "Importe", "Regla", "Justificante"}, []string{"Calcular dieta", "Aplicar excepcion", "Pedir justificante"}),
		screenSpec("dietas.justificantes", "dietas", "Justificantes", "Captura tickets, facturas, autorizaciones, asistencia, hash/CSV y OCR.", []string{"Documento", "Tipo", "Importe", "Fecha", "Hash", "OCR", "Estado"}, []string{"Subir", "Validar", "Rechazar", "Solicitar subsanacion"}),
		screenSpec("dietas.aprobaciones", "dietas", "Aprobaciones de dietas", "Decide gastos segun politica, importe, km, justificantes e historial.", []string{"Comision", "Politica", "Importe", "Km", "Justificantes", "Responsable", "Plazo"}, []string{"Aprobar", "Rechazar", "Escalar", "Devolver"}),
		screenSpec("dietas.liquidaciones", "dietas", "Liquidaciones", "Prepara pago/reembolso y conciliacion con nomina o contabilidad.", []string{"Lote", "Empleado", "Importe", "Conceptos", "Pago", "Nomina", "Errores"}, []string{"Liquidar", "Enviar a nomina", "Conciliar", "Reabrir"}),
		screenSpec("rutas.kilometraje", "rutas", "Kilometraje", "Calcula y justifica kilometraje reembolsable con ruta, tarifa y motivo.", []string{"Fecha", "Origen", "Destino", "Paradas", "Km calculados", "Km ajustados", "Tarifa"}, []string{"Calcular ruta", "Ajustar km", "Validar", "Rechazar"}),
		screenSpec("rutas.mapa_provincia", "rutas", "Mapa provincia", "Mantiene referencia de municipios, distancias, tiempos y dieta orientativa.", []string{"Origen", "Destino", "Km", "Minutos", "Ruta preferente", "Dieta sugerida", "Vigencia"}, []string{"Seleccionar ruta", "Anadir parada", "Actualizar tabla"}),

		screenSpec("bolsa.dashboard", "bolsa", "Dashboard Bolsa", "Prioriza convocatorias, solicitudes, subsanaciones, certificados, listados y alegaciones.", []string{"Convocatoria", "Solicitudes", "Subsanaciones", "Certificados", "Listados", "Vencimientos"}, []string{"Abrir expediente", "Publicar listado", "Asignar revisores"}),
		screenSpec("bolsa.convocatorias", "bolsa", "Convocatorias", "Configura bases, categorias, requisitos, fechas, baremo y responsables.", []string{"Convocatoria", "Bases", "Categoria", "Fechas", "Baremo", "Version", "Estado"}, []string{"Crear", "Versionar", "Publicar", "Cerrar"}),
		screenSpec("bolsa.solicitudes", "bolsa", "Solicitudes", "Revisa admision, expediente documental, certificados, meritos y requerimientos.", []string{"Solicitud", "Candidato", "Documentos", "Certificado", "Meritos", "Plazo", "Estado"}, []string{"Revisar", "Requerir", "Admitir", "Excluir"}),
		screenSpec("bolsa.meritos", "bolsa", "Meritos", "Valida fuente, evidencia, categoria, puntos declarados, aplicados y topes.", []string{"Merito", "Fuente", "Categoria", "Evidencia", "Puntos", "Tope", "Resultado"}, []string{"Validar", "Rechazar", "Recalcular", "Pedir evidencia"}),
		screenSpec("bolsa.autobaremo", "bolsa", "Autobaremacion", "Simula y fija baremacion segun reglas, topes, avisos y version.", []string{"Regla", "Declarado", "Aplicado", "Tope", "Aviso", "Categoria", "Recibo"}, []string{"Simular", "Recalcular", "Cerrar baremo"}),
		screenSpec("bolsa.alegaciones", "bolsa", "Alegaciones", "Resuelve impugnaciones, subsanaciones, informes y notificaciones.", []string{"Alegacion", "Item", "Plazo", "Documento", "Propuesta", "Tecnico", "Estado"}, []string{"Admitir tramite", "Resolver", "Pedir informe", "Notificar"}),
		screenSpec("bolsa.listados", "bolsa", "Listados", "Genera, firma y publica listados provisionales/definitivos.", []string{"Tipo", "Ranking", "Puntos", "Exclusion", "Firma", "CSV", "Version"}, []string{"Generar", "Validar", "Firmar", "Publicar"}),

		screenSpec("documentos.repositorio", "documentos", "Repositorio documental", "Gobierna evidencias por expediente, empleado, modulo, version y permisos.", []string{"Documento", "Tipo", "Modulo", "Expediente", "Version", "Hash", "CSV"}, []string{"Subir", "Vincular", "Reemplazar version", "Archivar"}),
		screenSpec("documentos.firma_csv", "documentos", "Firma y CSV", "Controla firma, CSV, sello temporal, errores y verificabilidad.", []string{"Documento", "Firmante", "Certificado", "CSV", "Sello", "Estado", "Errores"}, []string{"Enviar a firma", "Reintentar", "Validar CSV", "Revocar"}),
		screenSpec("documentos.plantillas", "documentos", "Plantillas", "Mantiene plantillas oficiales para certificados, resoluciones y listados.", []string{"Plantilla", "Modulo", "Version", "Campos", "Firmantes", "Vigencia", "Estado"}, []string{"Crear version", "Probar mezcla", "Publicar", "Retirar"}),
		screenSpec("aprobaciones.bandeja", "aprobaciones", "Bandeja de aprobaciones", "Centraliza decisiones por rol sin sustituir la pantalla origen.", []string{"Modulo", "Tipo", "Solicitante", "Importe/dias", "Plazo", "Evidencias", "Responsable"}, []string{"Aprobar", "Rechazar", "Subsanacion", "Reasignar"}),
		screenSpec("aprobaciones.reglas", "aprobaciones", "Reglas de aprobacion", "Configura aprobadores, umbrales, suplencias, vigencias y excepciones.", []string{"Regla", "Modulo", "Unidad", "Umbral", "Rol", "Suplente", "Vigencia"}, []string{"Crear regla", "Probar ruta", "Publicar", "Desactivar"}),
		screenSpec("auditoria.eventos", "auditoria", "Eventos de auditoria", "Consulta eventos por actor, expediente, modulo, fecha y resultado.", []string{"Evento", "Actor", "Rol", "Fecha", "Modulo", "Estado anterior", "Estado nuevo"}, []string{"Filtrar", "Abrir entidad", "Exportar", "Comparar payload"}),
		screenSpec("auditoria.trazabilidad", "auditoria", "Trazabilidad", "Reconstruye timeline de expediente, documento, decisiones e integraciones.", []string{"Entidad", "Timeline", "Decision", "Documento", "Integracion", "Recibo", "Responsable"}, []string{"Ver detalle", "Descargar paquete", "Comparar versiones"}),
		screenSpec("admin.usuarios_roles", "administracion", "Usuarios y roles", "Administra perfiles por rol, unidad, modulo, aprobacion, suplencia y auditoria.", []string{"Rol", "Ambito", "Usuarios", "Modulos", "Permisos clave", "Estado"}, []string{"Alta usuario", "Asignar rol", "Suspender", "Exportar matriz"}),
		screenSpec("admin.catalogos", "administracion", "Catalogos", "Mantiene regimenes de personal, tipos/codigos RPT, formas de provision, vigencias, versiones y dependencias.", []string{"Catalogo", "Codigo", "Descripcion", "Fuente", "Modulo", "Estado"}, []string{"Crear entrada", "Versionar", "Desactivar", "Validar impacto"}),
		screenSpec("admin.integraciones", "administracion", "Integraciones", "Configura conectores externos, lotes, errores, reintentos y SLA.", []string{"Sistema", "Endpoint", "Lote", "Ultimo envio", "Errores", "Reintentos", "SLA"}, []string{"Probar conexion", "Pausar", "Reintentar lote", "Rotar credencial"}),
		screenSpec("admin.monitorizacion", "administracion", "Monitorizacion", "Vigila salud funcional, trabajos en segundo plano, colas y alertas.", []string{"Job", "Cola", "Ultimo exito", "Tiempo", "Errores", "Modulo", "Capacidad"}, []string{"Reintentar job", "Pausar cola", "Abrir incidencia"}),
	}
	catalog := make([]map[string]any, 0, len(specs))
	for _, spec := range specs {
		catalog = append(catalog, screenCatalogEntry(spec))
	}
	return catalog
}

func screenSpec(id, moduleKey, title, description string, fields, actions []string) workspaceScreenSpec {
	return workspaceScreenSpec{
		ID:           id,
		ModuleKey:    moduleKey,
		MenuID:       id,
		Title:        title,
		Description:  description,
		Fields:       fields,
		Actions:      actions,
		States:       defaultScreenStates(moduleKey),
		Integrations: defaultScreenIntegrations(moduleKey),
		Validations:  defaultScreenValidations(moduleKey),
		DoneCriteria: "Estado persistido con referencia funcional, recibo y auditoria visible.",
	}
}

func screenCatalogEntry(spec workspaceScreenSpec) map[string]any {
	return map[string]any{
		"id":            spec.ID,
		"module_key":    spec.ModuleKey,
		"module_id":     screenModuleID(spec.ModuleKey),
		"menu_id":       spec.MenuID,
		"path":          screenPath(spec.ModuleKey, spec.ID),
		"title":         spec.Title,
		"description":   spec.Description,
		"fields":        screenFields(spec.Fields),
		"actions":       spec.Actions,
		"states":        spec.States,
		"integrations":  spec.Integrations,
		"validations":   spec.Validations,
		"done_criteria": spec.DoneCriteria,
	}
}

func screenModuleID(moduleKey string) string {
	switch moduleKey {
	case "personal", "nominas":
		return personalmodule.ModuleID
	case "cronos", "horarios", "permisos":
		return cronosmodule.ModuleID
	case "dietas", "rutas":
		return dietasmodule.ModuleID
	case "bolsa":
		return bolsamodule.ModuleID
	default:
		return "vec.module." + moduleKey
	}
}

func screenPath(moduleKey, id string) string {
	switch moduleKey {
	case "personal", "nominas":
		return "/modules/personal/" + id
	case "cronos", "horarios", "permisos":
		return "/modules/cronos/" + id
	case "dietas", "rutas":
		return "/modules/dietas/" + id
	case "bolsa":
		return "/modules/bolsa/" + id
	default:
		return "/modules/" + moduleKey + "/" + id
	}
}

func screenFields(labels []string) []map[string]any {
	fields := make([]map[string]any, 0, len(labels))
	for _, label := range labels {
		fields = append(fields, screenField(screenFieldKey(label), label, "text", true))
	}
	return fields
}

func screenFieldKey(label string) string {
	replacer := map[rune]rune{
		' ': '_', '/': '_', '-': '_', '.': '_',
	}
	out := make([]rune, 0, len(label))
	for _, r := range label {
		if replacement, ok := replacer[r]; ok {
			out = append(out, replacement)
			continue
		}
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return "campo"
	}
	return string(out)
}

func defaultScreenStates(moduleKey string) []string {
	switch moduleKey {
	case "nominas":
		return []string{"Preparacion", "Calculando", "Con errores", "Validado", "Cerrado"}
	case "cronos", "horarios", "permisos":
		return []string{"Borrador", "Pendiente aprobacion", "Aprobado", "Subsanacion", "Cerrado"}
	case "dietas", "rutas":
		return []string{"Borrador", "Solicitada", "Fuera politica", "Aprobada", "Liquidada"}
	case "bolsa":
		return []string{"Borrador", "Presentada", "En revision", "Subsanacion", "Publicada"}
	case "documentos":
		return []string{"Borrador", "Recibido", "Validado", "Firmado", "Archivado"}
	case "auditoria":
		return []string{"Registrado", "Investigado", "Exportado", "Retenido"}
	default:
		return []string{"Borrador", "Pendiente", "Validado", "Bloqueado", "Cerrado"}
	}
}

func defaultScreenIntegrations(moduleKey string) []string {
	switch moduleKey {
	case "personal":
		return []string{"Nominas", "Cronos", "Bolsa", "Documentos", "Auditoria"}
	case "nominas":
		return []string{"Personal", "Cronos", "Dietas", "Banco", "Contabilidad", "Auditoria"}
	case "cronos", "horarios", "permisos":
		return []string{"Personal", "Nominas", "Documentos", "Auditoria"}
	case "dietas", "rutas":
		return []string{"Personal", "Nominas", "Contabilidad", "Documentos", "Auditoria"}
	case "bolsa":
		return []string{"Personal", "Documentos", "Notificaciones", "Auditoria"}
	default:
		return []string{"Auditoria", "Documentos"}
	}
}

func defaultScreenValidations(moduleKey string) []string {
	switch moduleKey {
	case "nominas":
		return []string{"Empleado activo", "Conceptos vigentes", "Incidencias bloqueantes", "Totales de control"}
	case "cronos", "horarios", "permisos":
		return []string{"Perfil vigente", "Saldo suficiente", "Solapes", "Responsable competente"}
	case "dietas", "rutas":
		return []string{"Ruta valida", "Politica aplicable", "Justificante", "Centro coste"}
	case "bolsa":
		return []string{"Plazo", "Identidad", "Documentos obligatorios", "Reglas de baremo"}
	case "documentos":
		return []string{"Formato", "Hash duplicado", "CSV unico", "Permisos"}
	default:
		return []string{"Permiso de acceso", "Datos obligatorios", "Auditoria"}
	}
}

func screenField(key, label, kind string, required bool) map[string]any {
	return map[string]any{
		"key":      key,
		"label":    label,
		"type":     kind,
		"required": required,
	}
}

func workspaceAccessRoles() []map[string]any {
	return []map[string]any{
		{
			"id":              "administrador",
			"label":           "Administrador",
			"scope":           "Global VEC",
			"users_count":     2,
			"modules":         []string{"Dashboard", "Personal", "Nominas", "Cronos", "Dietas", "Bolsa", "Documentos", "Administracion", "Auditoria"},
			"key_permissions": []string{"Gestion usuarios/roles", "Catalogos maestros", "Integraciones", "Auditoria completa"},
			"state":           "Activo",
			"risk":            "Privilegiado; requiere doble factor y trazabilidad completa.",
		},
		{
			"id":              "administrativo",
			"label":           "Administrativo",
			"scope":           "Unidad o delegacion asignada",
			"users_count":     34,
			"modules":         []string{"Personal", "Cronos", "Dietas", "Bolsa", "Documentos"},
			"key_permissions": []string{"Alta de borradores", "Registro documental", "Subsanaciones", "Consulta de expedientes de su unidad"},
			"state":           "Activo",
			"risk":            "No aprueba permisos, dietas ni cierre de nomina.",
		},
		{
			"id":              "jefe_seccion",
			"label":           "Jefe/a de seccion",
			"scope":           "Seccion y personal adscrito",
			"users_count":     18,
			"modules":         []string{"Cronos", "Horarios", "Permisos", "Dietas", "Personal"},
			"key_permissions": []string{"Aprobar permisos ordinarios", "Validar incidencias de jornada", "Autorizar comisiones de su seccion"},
			"state":           "Activo",
			"risk":            "Sin acceso a cierre de nomina ni permisos globales.",
		},
		{
			"id":              "jefe_servicio",
			"label":           "Jefe/a de servicio",
			"scope":           "Servicio, centros y secciones dependientes",
			"users_count":     11,
			"modules":         []string{"Cronos", "Horarios", "Permisos", "Dietas", "Personal", "Aprobaciones"},
			"key_permissions": []string{"Aprobacion de segundo nivel", "Cuadrantes de servicio", "Escalado de excepciones", "Lectura agregada de personal"},
			"state":           "Activo",
			"risk":            "Puede aprobar fuera de politica solo con regla publicada.",
		},
		{
			"id":              "tecnico_rrhh",
			"label":           "Tecnico/a RRHH-Personal",
			"scope":           "Recursos Humanos",
			"users_count":     9,
			"modules":         []string{"Personal", "Nominas", "Cronos", "Horarios", "Bolsa", "Documentos", "Auditoria"},
			"key_permissions": []string{"Expedientes de personal", "RPT y puestos", "Situaciones administrativas", "Trienios", "Certificados", "Incidencias de nomina"},
			"state":           "Activo",
			"risk":            "Gestiona datos sensibles; requiere auditoria por expediente.",
		},
		{
			"id":              "jefatura_rrhh",
			"label":           "Jefatura RRHH-Personal",
			"scope":           "Recursos Humanos provincial",
			"users_count":     3,
			"modules":         []string{"Personal", "Nominas", "Cronos", "Horarios", "Dietas", "Bolsa", "Documentos", "Administracion", "Auditoria"},
			"key_permissions": []string{"Publicar catalogos", "Cerrar periodos", "Resolver conflictos de competencia", "Aprobar cambios maestros"},
			"state":           "Activo",
			"risk":            "Rol de gobierno funcional, separado del administrador tecnico.",
		},
	}
}

func workspaceRPTCatalog() map[string]any {
	return map[string]any{
		"source_summary": "Catalogo inicial construido con la pagina oficial de transparencia/RPT de dipgra.es, procesos RRHH publicados y extractor local de /home/alberto/Trabajo/nominas.",
		"official_links": []map[string]any{
			{"id": "dipgra_rpt", "label": "Relacion de Puestos de Trabajo", "url": "https://www.dipgra.es/servicios/areas/transparencia/portal-de-transparencia/a-informacion-institucional-y-organizativa/a3-personal/relacion-de-Puestos-de-trabajo/"},
			{"id": "dipgra_rrhh", "label": "Procesos de seleccion y provision RRHH", "url": "https://www.dipgra.es/diputacion/delegaciones/transparencia-recursos-humanos-y-administracion-electronica/recursos-humanos/"},
		},
		"local_sources": []map[string]any{
			{"id": "nominas_rpt_extractor", "path": "/home/alberto/Trabajo/nominas/internal/rpt/extractor.go", "usage": "Parser pdftotext para importar RPT completa."},
			{"id": "nominas_rpt_model", "path": "/home/alberto/Trabajo/nominas/internal/domain/types.go", "usage": "Modelo RPTFullPosition con campos RPT completos."},
			{"id": "uso_afiliados", "path": "/home/alberto/Trabajo/USO/app_afiliados/main.go", "usage": "Catalogo simple de situacion: funcionario, interino, laboral, otro."},
			{"id": "opes_bolsa_categories", "path": "/home/alberto/Trabajo/USO/web/internal/adapters/secondary/postgres/migrations/0002_seed_categories_from_opes.sql", "usage": "Categorias profesionales de Diputacion/OPES para Bolsa, Personal y RPT."},
		},
		"rpt_fields": []map[string]any{
			{"code": "codigo", "label": "Codigo de puesto", "required": true},
			{"code": "denominacion", "label": "Denominacion / datos de reserva", "required": true},
			{"code": "dot", "label": "Dotacion", "required": true},
			{"code": "tp", "label": "Tipo de puesto RPT", "required": true},
			{"code": "ad", "label": "Adscripcion / administracion RPT", "required": true},
			{"code": "fp", "label": "Forma de provision", "required": true},
			{"code": "grupo", "label": "Grupo/subgrupo", "required": true},
			{"code": "am", "label": "Ambito/area", "required": false},
			{"code": "escala", "label": "Escala", "required": false},
			{"code": "categoria", "label": "Categoria", "required": false},
			{"code": "cd", "label": "Complemento de destino", "required": false},
			{"code": "ce", "label": "Complemento especifico e importe anual", "required": false},
			{"code": "gcp_ct", "label": "GCP/CT", "required": false},
			{"code": "dg", "label": "Dispersion geografica", "required": false},
			{"code": "ta", "label": "Teletrabajo", "required": false},
			{"code": "fe", "label": "Fecha/efecto", "required": false},
			{"code": "observaciones", "label": "Observaciones", "required": false},
		},
	}
}

func workspaceRPTContractTypes() []map[string]any {
	return []map[string]any{
		{"catalog": "regimen_personal", "code": "funcionario_carrera", "label": "Funcionario/a de carrera", "source": "Catalogo VEC + USO", "module_key": "personal", "state": "Vigente", "usage": "Expediente, RPT, trienios, nomina y certificados."},
		{"catalog": "regimen_personal", "code": "funcionario_interino", "label": "Funcionario/a interino/a", "source": "Catalogo VEC + USO", "module_key": "personal", "state": "Vigente", "usage": "Nombramientos temporales, bolsas y servicios prestados."},
		{"catalog": "regimen_personal", "code": "laboral_fijo", "label": "Personal laboral fijo", "source": "dipgra.es procesos RRHH", "module_key": "personal", "state": "Vigente", "usage": "Provision por concurso general laboral y nomina."},
		{"catalog": "regimen_personal", "code": "laboral_temporal", "label": "Personal laboral temporal", "source": "Catalogo VEC", "module_key": "personal", "state": "Vigente", "usage": "Contratos temporales, llamamientos y certificados."},
		{"catalog": "regimen_personal", "code": "eventual", "label": "Personal eventual / confianza", "source": "RPT local nominas, codigo Tp/Ad E a validar", "module_key": "personal", "state": "Pendiente leyenda RPT", "usage": "Puestos de secretaria/asesoria detectados en RPT."},
		{"catalog": "regimen_personal", "code": "directivo", "label": "Personal directivo / jefaturas publicadas", "source": "dipgra.es RRHH", "module_key": "personal", "state": "Vigente", "usage": "Titulares de jefaturas de servicio y puestos directivos."},
		{"catalog": "rpt_tp", "code": "N", "label": "Puesto ordinario/no singularizado", "source": "PDF RPT local nominas", "module_key": "personal", "state": "Pendiente leyenda RPT", "usage": "Clasificacion del puesto en RPT."},
		{"catalog": "rpt_tp", "code": "S", "label": "Puesto singularizado", "source": "PDF RPT local nominas + RRHH dipgra", "module_key": "personal", "state": "Vigente", "usage": "Puestos singularizados y convocatorias especificas."},
		{"catalog": "rpt_tp", "code": "E", "label": "Puesto eventual/especial", "source": "PDF RPT local nominas", "module_key": "personal", "state": "Pendiente leyenda RPT", "usage": "Mantener codigo literal hasta importar leyenda oficial."},
		{"catalog": "rpt_ad", "code": "F", "label": "Adscripcion funcionarial", "source": "PDF RPT local nominas", "module_key": "personal", "state": "Pendiente leyenda RPT", "usage": "Cruce con regimen del expediente."},
		{"catalog": "rpt_ad", "code": "E", "label": "Adscripcion eventual/especial", "source": "PDF RPT local nominas", "module_key": "personal", "state": "Pendiente leyenda RPT", "usage": "Cruce con puestos de confianza o especiales."},
		{"catalog": "forma_provision", "code": "C", "label": "Concurso general", "source": "dipgra.es RRHH + RPT Fp C", "module_key": "personal", "state": "Vigente", "usage": "Provision ordinaria de puestos."},
		{"catalog": "forma_provision", "code": "L", "label": "Libre designacion", "source": "dipgra.es RRHH + RPT Fp L", "module_key": "personal", "state": "Vigente", "usage": "Puestos directivos, jefaturas o singularizados segun convocatoria."},
		{"catalog": "forma_provision", "code": "I", "label": "Codigo RPT I", "source": "PDF RPT local nominas", "module_key": "personal", "state": "Pendiente leyenda RPT", "usage": "Importar como codigo literal hasta confirmar significado oficial."},
		{"catalog": "forma_provision", "code": "2A", "label": "Codigo RPT 2A", "source": "PDF RPT local nominas", "module_key": "personal", "state": "Pendiente leyenda RPT", "usage": "Importar como codigo literal hasta confirmar significado oficial."},
		{"catalog": "procedimiento_rrhh", "code": "movilidad_voluntaria_provisional", "label": "Movilidad voluntaria provisional", "source": "dipgra.es RRHH", "module_key": "personal", "state": "Vigente", "usage": "Coberturas temporales y cambios provisionales."},
		{"catalog": "procedimiento_rrhh", "code": "cobertura_temporal", "label": "Cobertura temporal", "source": "dipgra.es RRHH", "module_key": "personal", "state": "Vigente", "usage": "Provision temporal y sustituciones."},
	}
}

func workspaceRPTPositionSamples() []map[string]any {
	return []map[string]any{
		{"code": "118", "name": "Delegado/a de Proteccion de Datos", "tp": "S", "ad": "F", "fp": "L", "group": "A1/A2", "area": "A7", "scale": "AGAEHN", "category": "TM1/TM4", "cd": 26, "ce": "CE anual", "dg": "NO", "ta": "Segun RPT", "coverage": "Libre designacion", "state": "Importado demo"},
		{"code": "344", "name": "Licenciado/a Comunicaciones Audiovisuales", "tp": "S", "ad": "F", "fp": "C", "group": "A1", "area": "A7", "scale": "AE", "category": "TS1", "cd": 24, "ce": "CE anual", "dg": "NO", "ta": "Segun RPT", "coverage": "Concurso", "state": "Importado demo"},
		{"code": "669", "name": "Secretaria de Presidencia", "tp": "E", "ad": "E", "fp": "I", "group": "C1", "area": "DG", "scale": "AG", "category": "AM", "cd": 22, "ce": "CE anual", "dg": "NO", "ta": "Segun RPT", "coverage": "Codigo RPT I", "state": "Pendiente leyenda"},
		{"code": "175", "name": "Encargado/a conductor", "tp": "S", "ad": "F", "fp": "C", "group": "C2", "area": "A7", "scale": "AE", "category": "EE4", "cd": 18, "ce": "CE anual", "dg": "NO", "ta": "Segun RPT", "coverage": "Concurso", "state": "Importado demo"},
		{"code": "8", "name": "Administrativo/a", "tp": "N", "ad": "F", "fp": "C", "group": "C1", "area": "A7", "scale": "AG", "category": "AM", "cd": 18, "ce": "CE anual", "dg": "NO", "ta": "Segun RPT", "coverage": "Concurso", "state": "Importado demo"},
		{"code": "359", "name": "Oficial de servicios multiples", "tp": "N", "ad": "F", "fp": "C", "group": "C2", "area": "A7", "scale": "AE", "category": "EE4", "cd": 16, "ce": "CE anual", "dg": "NO", "ta": "Segun RPT", "coverage": "Concurso", "state": "Importado demo"},
	}
}

func workspaceProfessionalCategories() []map[string]any {
	rows := []struct {
		slug, name, area, sourcePath string
	}{
		{"administrativo", "Administrativo", "administracion_general", "OPES/administracion-general/Administrativo"},
		{"auxiliar-administrativo", "Auxiliar Administrativo", "administracion_general", "OPES/administracion-general/Auxiliar-Administrativo"},
		{"subalterno", "Subalterno", "administracion_general", "OPES/administracion-general/Subalterno"},
		{"tecnico-de-administracion-general", "Tecnico de Administracion General", "administracion_general", "OPES/administracion-general/Tecnico-de-Administracion-General-00001"},
		{"tecnico-de-gestion", "Tecnico de Gestion", "administracion_general", "OPES/administracion-general/Tecnico-de-gestion"},
		{"analista", "Analista", "administracion_especial", "OPES/administracion-especial/Analista"},
		{"analista-programador", "Analista Programador", "administracion_especial", "OPES/administracion-especial/Analista-Programador"},
		{"analista-de-laboratorio", "Analista de Laboratorio", "administracion_especial", "OPES/administracion-especial/Analista-de-Laboratorio"},
		{"arquitecto", "Arquitecto", "administracion_especial", "OPES/administracion-especial/Arquitecto"},
		{"arquitecto-tecnico", "Arquitecto Tecnico", "administracion_especial", "OPES/administracion-especial/Arquitecto-Tecnico"},
		{"auxiliar-deportivo", "Auxiliar Deportivo", "administracion_especial", "OPES/administracion-especial/Auxiliar-Deportivo"},
		{"auxiliar-tecnico-superior-centros-sociales", "Auxiliar Tecnico Superior de Centros Sociales", "administracion_especial", "OPES/administracion-especial/Auxiliar-Tecnico-Superior-de-Centros-Sociales"},
		{"auxiliar-tecnico-superior-informatica", "Auxiliar Tecnico Superior de Informatica", "administracion_especial", "OPES/administracion-especial/Auxiliar-Tecnico-Superior-de-Informatica"},
		{"auxiliar-tecnico-superior-integracion-social", "Auxiliar Tecnico Superior de Integracion Social", "administracion_especial", "OPES/administracion-especial/Auxiliar-Tecnico-Superior-de-Integracion-Social"},
		{"auxiliar-tecnico-superior-obras", "Auxiliar Tecnico Superior de Obras", "administracion_especial", "OPES/administracion-especial/Auxiliar-Tecnico-Superior-de-Obras"},
		{"auxiliar-tecnico-superior-salud-ambiental", "Auxiliar Tecnico Superior de Salud Ambiental", "administracion_especial", "OPES/administracion-especial/Auxiliar-Tecnico-Superior-Salud-Ambiental-00001"},
		{"auxiliar-tecnico-preimpresion", "Auxiliar Tecnico a Preimpresion", "administracion_especial", "OPES/administracion-especial/Auxiliar-Tecnico-a-Preimpresion"},
		{"auxiliar-tecnico-superior-gestion-agua", "Auxiliar Tecnico Superior de Gestion de Agua", "administracion_especial", "OPES/administracion-especial/Auxiliar-Tecnico-a-Superior-de-Gestion-Agua"},
		{"auxiliar-tecnico-archivo", "Auxiliar Tecnico de Archivo", "administracion_especial", "OPES/administracion-especial/Auxiliar-Tecnico-de-Archivo"},
		{"auxiliar-enfermeria", "Auxiliar de Enfermeria", "administracion_especial", "OPES/administracion-especial/Auxiliar-de-Enfermeria"},
		{"auxiliar-informatica", "Auxiliar de Informatica", "administracion_especial", "OPES/administracion-especial/Auxiliar-de-Informatica"},
		{"auxiliar-servicios-generales", "Auxiliar de Servicios Generales", "administracion_especial", "OPES/administracion-especial/Auxiliar-de-Servicios-Generales"},
		{"ayudante-tecnico-sanitario-due", "Ayudante Tecnico Sanitario DUE", "administracion_especial", "OPES/administracion-especial/Ayudante-Tecnico-Sanitario-DUE"},
		{"cocinero", "Cocinero", "administracion_especial", "OPES/administracion-especial/Cocinero"},
		{"conductor", "Conductor/a", "administracion_especial", "OPES/administracion-especial/Conductor-a"},
		{"cuidador-tecnico-personas-dependientes", "Cuidador Tecnico de Personas Dependientes", "administracion_especial", "OPES/administracion-especial/Cuidador-Tecnico-de-Personas-Dependientes"},
		{"economista", "Economista", "administracion_especial", "OPES/administracion-especial/Economista"},
		{"educador", "Educador", "administracion_especial", "OPES/administracion-especial/Educador"},
		{"educador-social", "Educador Social", "administracion_especial", "OPES/administracion-especial/Educador-Social"},
		{"encargado", "Encargado", "administracion_especial", "OPES/administracion-especial/Encargado"},
		{"fisioterapeuta", "Fisioterapeuta", "administracion_especial", "OPES/administracion-especial/Fisioterapeuta"},
		{"ingenieria-tecnica-industrial", "Ingenieria Tecnica Industrial", "administracion_especial", "OPES/administracion-especial/Ingenieria-Tecnica-Industrial"},
		{"ingenieria-tecnica-obras-publicas", "Ingenieria Tecnica de Obras Publicas", "administracion_especial", "OPES/administracion-especial/Ingenieria-Tecnica-de-Obras-Publicas"},
		{"ingeniero-industrial", "Ingeniero Industrial", "administracion_especial", "OPES/administracion-especial/Ingeniero-Industrial"},
		{"ingeniero-tecnico-topografo", "Ingeniero Tecnico Topografo", "administracion_especial", "OPES/administracion-especial/Ingeniero-Tecnico-Topografo"},
		{"ingeniero-tecnico-agricola", "Ingeniero/a Tecnico/a Agricola", "administracion_especial", "OPES/administracion-especial/Ingeniero-a-Tecnico-a-Agricola"},
		{"ingeniero-telecomunicaciones", "Ingeniero/a Telecomunicaciones", "administracion_especial", "OPES/administracion-especial/Ingeniero-a-Telecomunicaciones"},
		{"ingeniero-caminos-canales-puertos", "Ingeniero de Caminos, Canales y Puertos", "administracion_especial", "OPES/administracion-especial/Ingeniero-de-Caminos-Canales-y-Puertos"},
		{"medico", "Medico", "administracion_especial", "OPES/administracion-especial/Medico"},
		{"medico-psiquiatra", "Medico Psiquiatra", "administracion_especial", "OPES/administracion-especial/Medico-Psiquiatra"},
		{"medico-medicina-trabajo", "Medico/a especialista en Medicina del Trabajo", "administracion_especial", "OPES/administracion-especial/Medico-a-especialista-en-Medicina-del-Trabajo"},
		{"oficial-fontaneria", "Oficial 1 Fontaneria", "administracion_especial", "OPES/administracion-especial/Oficial-1-Fontaneria"},
		{"oficial-letrado", "Oficial Letrado", "administracion_especial", "OPES/administracion-especial/Oficial-Letrado"},
		{"oficial-servicios-multiples", "Oficial de Servicios Multiples", "administracion_especial", "OPES/administracion-especial/Oficial-de-Servicios-Multiples"},
		{"operario", "Operario", "administracion_especial", "OPES/administracion-especial/Operario"},
		{"operario-tractorista-grupo-5", "Operario Tractorista Grupo 5", "administracion_especial", "OPES/administracion-especial/Operario-Tractorista-Grupo-5"},
		{"psicologo", "Psicologo", "administracion_especial", "OPES/administracion-especial/Psicologo"},
		{"tecnico-medio-desarrollo", "Tecnico Medio de Desarrollo", "administracion_especial", "OPES/administracion-especial/Tecnico-Medio-de-Desarrollo"},
		{"tecnico-medio-medio-ambiente", "Tecnico Medio de Medio Ambiente", "administracion_especial", "OPES/administracion-especial/Tecnico-Medio-de-Medio-Ambiente"},
		{"tecnico-superior-archivo", "Tecnico Superior Archivo", "administracion_especial", "OPES/administracion-especial/Tecnico-Superior-Archivo"},
		{"tecnico-superior-ade", "Tecnico Superior de Administracion y Direccion de Empresas", "administracion_especial", "OPES/administracion-especial/Tecnico-Superior-de-Administracion-y-Direccion-de-Empresas"},
		{"tecnico-superior-desarrollo", "Tecnico Superior de Desarrollo", "administracion_especial", "OPES/administracion-especial/Tecnico-Superior-de-Desarrollo"},
		{"tecnico-superior-gestion-presupuestaria", "Tecnico Superior de Gestion Presupuestaria", "administracion_especial", "OPES/administracion-especial/Tecnico-Superior-de-Gestion-Presupuestaria"},
		{"tecnico-superior-servicios-culturales", "Tecnico Superior de Servicios Culturales", "administracion_especial", "OPES/administracion-especial/Tecnico-Superior-de-Servicios-Culturales"},
		{"tecnico-medio-archivo-biblioteca", "Tecnico/a Medio Archivo Biblioteca", "administracion_especial", "OPES/administracion-especial/Tecnico-a-Medio-Archivo-Biblioteca"},
		{"tecnico-superior-deportes", "Tecnico/a Superior de Deportes", "administracion_especial", "OPES/administracion-especial/Tecnico-a-Superior-de-Deportes"},
		{"terapeuta-ocupacional", "Terapeuta Ocupacional", "administracion_especial", "OPES/administracion-especial/Terapeuta-Ocupacional"},
		{"trabajador-social", "Trabajador Social", "administracion_especial", "OPES/administracion-especial/Trabajador-Social"},
	}
	categories := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		categories = append(categories, map[string]any{
			"catalog":     "categoria_profesional",
			"slug":        row.slug,
			"name":        row.name,
			"area":        row.area,
			"source":      "Bolsa/OPES",
			"source_path": row.sourcePath,
			"module_key":  "bolsa",
			"state":       "Vigente catalogo Bolsa",
			"usage":       "Convocatorias, baremo por misma/otra categoria, Personal/RPT y certificados de servicios prestados.",
		})
	}
	return categories
}

func workspacePayrollRun() map[string]any {
	return map[string]any{
		"id":          "PAYRUN-2026-06-ORD",
		"module_key":  "nominas",
		"module_id":   personalmodule.ModuleID,
		"period":      "2026-06",
		"description": "Nomina ordinaria junio 2026 - datos demo para validacion funcional",
		"state":       "precalculo_en_validacion",
		"demo_notice": "Datos precalculados de demostracion; no conectan con bancos, contabilidad ni sistemas reales.",
		"calendar": map[string]any{
			"cutoff_at":       "2026-06-25T14:00:00+02:00",
			"approval_due_at": "2026-06-27T12:00:00+02:00",
			"pay_date":        "2026-06-30",
		},
		"population": map[string]any{
			"employees_in_scope":       1248,
			"with_cronos_variation":    37,
			"with_expense_liquidation": 11,
			"excluded_demo_records":    3,
		},
		"totals": map[string]any{
			"currency":             "EUR",
			"gross_amount":         3128450.72,
			"deduction_amount":     823140.18,
			"net_amount":           2305310.54,
			"expense_allowances":   1842.30,
			"variable_concepts":    74,
			"manual_review_amount": 3268.41,
		},
		"concepts": []map[string]any{
			{"code": "BASE", "label": "Retribuciones basicas", "records": 1248, "amount": 1680440.22},
			{"code": "COMP", "label": "Complementos destino/especifico", "records": 1216, "amount": 1032815.74},
			{"code": "TRIEN", "label": "Trienios", "records": 804, "amount": 206144.90},
			{"code": "DIETA", "label": "Dietas y kilometraje liquidable", "records": 11, "amount": 1842.30},
			{"code": "REG", "label": "Regularizaciones demo", "records": 9, "amount": 7207.56},
		},
		"dependencies": []map[string]any{
			{"module_key": "personal", "label": "Puestos, situaciones y antiguedad", "state": "actualizado", "records": 1248},
			{"module_key": "cronos", "label": "Fichajes, reducciones 63/64 e incidencias", "state": "pendiente_revision", "records": 37},
			{"module_key": "permisos", "label": "Permisos con impacto retributivo", "state": "pendiente_responsable", "records": 5},
			{"module_key": "dietas", "label": "Liquidaciones aprobadas para abono", "state": "lista_para_nomina", "records": 11},
		},
		"validation_checks": []map[string]any{
			{"id": "check-cronos-open", "label": "Incidencias Cronos abiertas", "state": "warning", "count": 7, "blocking": false},
			{"id": "check-bank-demo", "label": "Cuenta bancaria demo informada", "state": "ok", "count": 1248, "blocking": false},
			{"id": "check-irpf", "label": "Retenciones informadas", "state": "ok", "count": 1248, "blocking": false},
			{"id": "check-expenses", "label": "Dietas con justificante validado", "state": "review", "count": 2, "blocking": false},
		},
		"incidents": []map[string]any{
			{"id": "NOM-INC-2026-063", "employee": "EMP-0063", "source_module": "cronos", "summary": "Reduccion 63 anos pendiente de consolidar", "estimated_amount_delta": -64.20, "flow_state": "validacion_rrhh", "next_action": "Validar reduccion"},
			{"id": "NOM-INC-2026-064", "employee": "EMP-0064", "source_module": "cronos", "summary": "Reduccion 64 anos con cruce horario pendiente", "estimated_amount_delta": -128.40, "flow_state": "pendiente_responsable", "next_action": "Cruzar cuadrante"},
			{"id": "NOM-INC-2026-044", "employee": "EMP-0031", "source_module": "dietas", "summary": "Kilometraje Granada-Motril a validar antes de abono", "estimated_amount_delta": 36.61, "flow_state": "subsanacion_requerida", "next_action": "Confirmar km"},
		},
		"audit": map[string]any{
			"last_recalculated_at": "2026-06-19T09:30:00+02:00",
			"actor":                "demo.rrhh",
			"receipt":              "DEMO-PAYRUN-2026-06-0001",
		},
	}
}

func workspaceExpensePolicy() map[string]any {
	return map[string]any{
		"id":             "EXP-POL-DPGR-2026-DEMO",
		"module_key":     "dietas",
		"module_id":      dietasmodule.ModuleID,
		"effective_from": "2026-01-01",
		"state":          "demo_publicado",
		"demo_notice":    "Politica de ejemplo para validar pantallas; los importes no tienen valor normativo.",
		"currency":       "EUR",
		"mileage": map[string]any{
			"rate_eur_km":       0.26,
			"rounding":          "2 decimales",
			"route_source":      "province_routes demo",
			"requires_route_ok": true,
		},
		"allowance_types": []map[string]any{
			{"id": "no_dieta", "label": "Sin dieta", "amount": 0.00, "requires_meal_proof": false, "rule": "Desplazamiento sin derecho por horario o distancia."},
			{"id": "media_dieta", "label": "Media dieta", "amount": 26.67, "requires_meal_proof": true, "rule": "Jornada con manutencion justificada y retorno en el dia."},
			{"id": "dieta_completa", "label": "Dieta completa", "amount": 53.34, "requires_meal_proof": true, "rule": "Jornada completa con franja de comida/cena segun autorizacion."},
			{"id": "pernocta_demo", "label": "Pernocta demo", "amount": 65.97, "requires_meal_proof": true, "rule": "Solo para probar el flujo; requiere autorizacion previa."},
		},
		"required_documents": []map[string]any{
			{"id": "authorization", "label": "Autorizacion de comision", "required_for": []string{"media_dieta", "dieta_completa", "pernocta_demo"}},
			{"id": "attendance", "label": "Certificado de asistencia", "required_for": []string{"media_dieta", "dieta_completa"}},
			{"id": "receipt", "label": "Justificante de gasto", "required_for": []string{"media_dieta", "dieta_completa", "pernocta_demo"}},
			{"id": "route_reason", "label": "Motivo de ruta no estandar", "required_for": []string{"ruta_excedida"}},
		},
		"approval_chain": []map[string]any{
			{"step": 1, "role": "solicitante", "state": "borrador", "action": "Registrar comision"},
			{"step": 2, "role": "responsable_unidad", "state": "pendiente_responsable", "action": "Autorizar desplazamiento"},
			{"step": 3, "role": "intervencion_demo", "state": "validacion_gasto", "action": "Validar politica y justificantes"},
			{"step": 4, "role": "nominas", "state": "lista_para_nomina", "action": "Incluir en payroll_run demo"},
		},
		"thresholds": map[string]any{
			"route_over_km_requires_reason": 15.0,
			"same_day_max_hours":            12,
			"manual_review_amount":          150.0,
			"late_submission_days":          10,
		},
		"exceptions": []map[string]any{
			{"id": "route-exceeded", "label": "Ruta excedida", "state": "subsanacion_requerida", "next_action": "Aportar motivo y validar responsable"},
			{"id": "missing-receipt", "label": "Justificante pendiente", "state": "bloqueado", "next_action": "Subir justificante"},
			{"id": "policy-ok", "label": "Politica cumplida", "state": "lista_para_nomina", "next_action": "Liquidar"},
		},
		"payroll_link": map[string]any{
			"enabled_demo":       true,
			"payroll_period_ref": "PAYRUN-2026-06-ORD",
			"concept_code":       "DIETA",
			"handoff_state":      "lista_para_nomina",
		},
	}
}

func workspaceActionCatalog() map[string][]map[string]any {
	return map[string][]map[string]any{
		"personal": {
			{"id": "personal.empleado.update", "label": "Actualizar expediente", "screen_id": "personal.expedientes", "required_permissions": []string{personalmodule.PermissionEmployeeManage}, "flow_transition": "borrador -> validacion_rrhh", "demo_effect": "Actualiza puesto, unidad o situacion en el workspace demo."},
			{"id": personalmodule.ActionIssueServiceCertificate, "label": "Emitir certificado", "screen_id": "personal.expedientes", "required_permissions": []string{personalmodule.PermissionCertificateManage}, "flow_transition": "validacion_rrhh -> firmado_demo", "demo_effect": "Genera recibo demo para servicios prestados o meritos Bolsa."},
		},
		"nominas": {
			{"id": personalmodule.ActionReviewPayrollIncident, "label": "Revisar incidencia de nomina", "screen_id": "nominas.incidencias", "required_permissions": []string{personalmodule.PermissionPayrollManage}, "flow_transition": "pendiente_responsable -> validacion_rrhh", "demo_effect": "Marca la incidencia para recalculo del payroll_run demo."},
			{"id": "personal.nomina.precalculo.rebuild", "label": "Recalcular precalculo", "screen_id": "nominas.cierre", "required_permissions": []string{personalmodule.PermissionPayrollManage}, "flow_transition": "precalculo_en_validacion -> precalculo_actualizado", "demo_effect": "Refresca totales demo sin publicar pagos reales."},
		},
		"cronos": {
			{"id": cronosmodule.ActionReviewJustification, "label": "Revisar justificante", "screen_id": "cronos.fichajes", "required_permissions": []string{cronosmodule.PermissionTimeManage}, "flow_transition": "subsanacion_requerida -> validacion_cronos", "demo_effect": "Cierra una incidencia de fichaje en datos demo."},
			{"id": "cronos.jornada.telework.confirm", "label": "Confirmar teletrabajo", "screen_id": "cronos.fichajes", "required_permissions": []string{cronosmodule.PermissionTimeManage}, "flow_transition": "borrador -> aprobado", "demo_effect": "Incluye la jornada en el resumen Cronos demo."},
		},
		"horarios": {
			{"id": "cronos.horario.profile.update", "label": "Editar perfil horario", "screen_id": "horarios.perfiles", "required_permissions": []string{cronosmodule.PermissionScheduleManage}, "flow_transition": "borrador -> vigente", "demo_effect": "Actualiza un perfil horario demo."},
			{"id": "cronos.horario.reduction.apply", "label": "Aplicar reduccion 63/64", "screen_id": "horarios.perfiles", "required_permissions": []string{cronosmodule.PermissionScheduleManage}, "flow_transition": "validacion_rrhh -> vigente", "demo_effect": "Propaga una reduccion demo a Cronos y nominas."},
		},
		"permisos": {
			{"id": cronosmodule.ActionReviewLeaveAndHoliday, "label": "Resolver permiso/vacacion", "screen_id": "permisos.solicitudes", "required_permissions": []string{cronosmodule.PermissionApprovalManage}, "flow_transition": "pendiente_responsable -> aprobado", "demo_effect": "Actualiza saldo y cuadrante demo."},
			{"id": "cronos.permiso.balance.adjust", "label": "Ajustar saldo", "screen_id": "permisos.solicitudes", "required_permissions": []string{cronosmodule.PermissionLeaveManage}, "flow_transition": "validacion_cronos -> saldo_actualizado", "demo_effect": "Registra un ajuste auditado de saldo demo."},
		},
		"dietas": {
			{"id": dietasmodule.ActionReviewTravelExpense, "label": "Revisar comision", "screen_id": "dietas.comisiones", "required_permissions": []string{dietasmodule.PermissionExpenseManage}, "flow_transition": "pendiente_responsable -> validacion_gasto", "demo_effect": "Valida justificantes y politica demo."},
			{"id": "dietas.liquidacion.send_to_payroll", "label": "Enviar a nomina demo", "screen_id": "dietas.comisiones", "required_permissions": []string{dietasmodule.PermissionExpenseManage}, "flow_transition": "validacion_gasto -> lista_para_nomina", "demo_effect": "Anade importe al payroll_run demo como concepto DIETA."},
		},
		"rutas": {
			{"id": dietasmodule.ActionReviewRouteKM, "label": "Validar kilometraje", "screen_id": "rutas.mapa_provincia", "required_permissions": []string{dietasmodule.PermissionRouteManage}, "flow_transition": "subsanacion_requerida -> ruta_validada", "demo_effect": "Confirma distancia de province_routes demo."},
			{"id": "dietas.ruta.exception.resolve", "label": "Resolver excepcion de ruta", "screen_id": "rutas.mapa_provincia", "required_permissions": []string{dietasmodule.PermissionRouteManage}, "flow_transition": "bloqueado -> validacion_gasto", "demo_effect": "Registra motivo de ruta no estandar."},
		},
		"bolsa": {
			{"id": bolsamodule.ActionDemoIntegration, "label": "Sincronizar expediente demo", "screen_id": "bolsa.solicitudes", "required_permissions": []string{bolsamodule.PermissionDemoAction}, "flow_transition": "registrado -> en_revision", "demo_effect": "Conecta certificados demo de personal con solicitud Bolsa."},
			{"id": "bolsa.autobaremo.recalculate", "label": "Recalcular autobaremo", "screen_id": "bolsa.autobaremo", "required_permissions": []string{bolsamodule.PermissionManage}, "flow_transition": "en_revision -> baremo_provisional", "demo_effect": "Actualiza puntuacion provisional sin valor juridico."},
		},
		"administracion": {
			{"id": "vec.roles.assign", "label": "Asignar rol", "screen_id": "admin.usuarios_roles", "required_permissions": []string{"vec.roles.manage"}, "flow_transition": "pendiente -> activo", "demo_effect": "Registra una asignacion de rol con ambito y suplencia demo."},
			{"id": "vec.catalog.rpt.publish", "label": "Publicar catalogo RPT", "screen_id": "admin.catalogos", "required_permissions": []string{"vec.catalogs.manage", personalmodule.PermissionPositionManage}, "flow_transition": "borrador -> vigente", "demo_effect": "Versiona catalogos RPT demo y deja traza de fuente."},
		},
	}
}

func workspaceFlowStates() []map[string]any {
	return []map[string]any{
		{"id": "borrador", "label": "Borrador", "category": "draft", "modules": []string{"personal", "nominas", "cronos", "horarios", "permisos", "dietas", "rutas", "bolsa"}, "sla": "sin vencimiento", "terminal": false, "next_actions": []string{"guardar", "enviar_a_validacion"}},
		{"id": "pendiente_responsable", "label": "Pendiente responsable", "category": "review", "modules": []string{"permisos", "dietas", "nominas", "cronos"}, "sla": "72 h", "terminal": false, "next_actions": []string{"aprobar", "devolver", "escalar"}},
		{"id": "validacion_rrhh", "label": "Validacion RRHH", "category": "review", "modules": []string{"personal", "nominas", "horarios", "bolsa"}, "sla": "48 h", "terminal": false, "next_actions": []string{"validar", "solicitar_subsanacion", "rechazar"}},
		{"id": "validacion_cronos", "label": "Validacion Cronos", "category": "review", "modules": []string{"cronos", "horarios", "permisos"}, "sla": "24 h", "terminal": false, "next_actions": []string{"cuadrar_jornada", "aprobar", "devolver"}},
		{"id": "validacion_gasto", "label": "Validacion gasto", "category": "review", "modules": []string{"dietas", "rutas"}, "sla": "5 dias", "terminal": false, "next_actions": []string{"validar_politica", "pedir_justificante", "enviar_a_nomina"}},
		{"id": "subsanacion_requerida", "label": "Subsanacion requerida", "category": "blocked", "modules": []string{"cronos", "dietas", "rutas", "bolsa", "nominas"}, "sla": "72 h", "terminal": false, "next_actions": []string{"aportar_documento", "justificar", "devolver_a_revision"}},
		{"id": "bloqueado", "label": "Bloqueado", "category": "blocked", "modules": []string{"nominas", "dietas", "rutas", "bolsa"}, "sla": "requiere accion manual", "terminal": false, "next_actions": []string{"resolver_bloqueo", "escalar"}},
		{"id": "aprobado", "label": "Aprobado", "category": "accepted", "modules": []string{"permisos", "dietas", "cronos", "personal"}, "sla": "seguimiento ordinario", "terminal": false, "next_actions": []string{"registrar", "notificar", "liquidar_si_aplica"}},
		{"id": "lista_para_nomina", "label": "Lista para nomina", "category": "handoff", "modules": []string{"dietas", "nominas", "cronos"}, "sla": "antes de corte nomina", "terminal": false, "next_actions": []string{"incluir_payroll_run", "recalcular"}},
		{"id": "baremo_provisional", "label": "Baremo provisional", "category": "provisional", "modules": []string{"bolsa"}, "sla": "segun convocatoria demo", "terminal": false, "next_actions": []string{"publicar_provisional", "abrir_alegaciones"}},
		{"id": "firmado_demo", "label": "Firmado demo", "category": "receipt", "modules": []string{"personal", "bolsa"}, "sla": "sin vencimiento", "terminal": false, "next_actions": []string{"descargar_recibo", "vincular_expediente"}},
		{"id": "cerrado", "label": "Cerrado", "category": "closed", "modules": []string{"personal", "nominas", "cronos", "horarios", "permisos", "dietas", "rutas", "bolsa"}, "sla": "finalizado", "terminal": true, "next_actions": []string{"consultar_auditoria", "exportar_demo"}},
	}
}
