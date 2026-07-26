package registroaccesos

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type seudonimizadorAccesosPrueba struct {
	seudonimo string
	sujeto    string
	ambito    string
}

func (s *seudonimizadorAccesosPrueba) SeudonimizarSujetoAlmacen(
	_ context.Context,
	solicitud vecports.SolicitudSeudonimizarSujetoAlmacen,
) (string, error) {
	s.sujeto, s.ambito, _ = solicitud.RevelarParaSellado()
	return s.seudonimo, nil
}

func TestEvidenciaActorSoloNaceDeFronteraHMACYLigaPrincipal(t *testing.T) {
	t.Parallel()
	sellador := &seudonimizadorAccesosPrueba{
		seudonimo: "hmac-sha256:bolsa_accesos_v1:" +
			strings.Repeat("a", 64),
	}
	evidencia, err := NuevaEvidenciaActorConsultaAccesos(
		context.Background(), "principal_interno_123456", sellador,
	)
	if err != nil || sellador.sujeto != "principal_interno_123456" ||
		sellador.ambito != ambitoSeudonimizacionAccesos {
		t.Fatalf("frontera HMAC no ligada: evidencia=%v error=%v", evidencia, err)
	}
	if evidencia.validarPara(
		"otro_principal", sellador.seudonimo,
	) == nil {
		t.Fatal("la evidencia se reutilizo con otro principal")
	}
	if (EvidenciaActorConsultaAccesos{}).validarPara(
		"principal_interno_123456", sellador.seudonimo,
	) == nil {
		t.Fatal("el valor cero actuo como evidencia HMAC")
	}
}

func TestFiltroConsultaAdministrativaAccesosDenyByDefault(t *testing.T) {
	t.Parallel()
	base := filtroValidoPrueba()
	casos := map[string]func(*FiltroConsultaAdministrativaAccesos){
		"version cero": func(f *FiltroConsultaAdministrativaAccesos) {
			f.Version = 0
		},
		"sin ancla": func(f *FiltroConsultaAdministrativaAccesos) {
			f.ActorSeudonimizado = ""
		},
		"limite excesivo": func(f *FiltroConsultaAdministrativaAccesos) {
			f.Limite = 101
		},
		"intervalo excesivo": func(f *FiltroConsultaAdministrativaAccesos) {
			f.HastaExclusive = f.DesdeInclusive.Add(32 * 24 * time.Hour)
		},
		"comodin": func(f *FiltroConsultaAdministrativaAccesos) {
			f.ModuloID = "*"
		},
		"sin finalidad": func(f *FiltroConsultaAdministrativaAccesos) {
			f.FinalidadDeLaConsulta = ""
		},
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			t.Parallel()
			filtro := base
			mutar(&filtro)
			if !errors.Is(
				filtro.Validar(),
				ErrConsultaAdministrativaAccesosDenegada,
			) {
				t.Fatal("la consulta insegura no fue denegada")
			}
		})
	}
	if err := base.Validar(); err != nil {
		t.Fatalf("filtro valido rechazado: %v", err)
	}
}

func TestCursorEsOpacoYNoSeFiltraAlFormatear(t *testing.T) {
	t.Parallel()
	valor := "cursor:v1:" + strings.Repeat("a", 64)
	cursor, err := NuevoCursorConsultaAdministrativaAccesos(valor)
	if err != nil || cursor.Valor() != valor {
		t.Fatalf("cursor valido rechazado: %v", err)
	}
	if strings.Contains(cursor.String(), valor) {
		t.Fatal("String filtro el cursor")
	}
	if _, err := NuevoCursorConsultaAdministrativaAccesos(
		"cursor:v1:1",
	); err == nil {
		t.Fatal("cursor malformado aceptado")
	}
	if _, err := NuevoCursorConsultaAdministrativaAccesos(
		"cursor:v1:" + strings.Repeat("A", 64),
	); err == nil {
		t.Fatal("cursor hexadecimal mayusculo aceptado")
	}
}

func TestRecursoAutorizableLigaFiltroMinimoYCompleto(t *testing.T) {
	t.Parallel()
	seudonimo := "hmac-sha256:bolsa_accesos_v1:" + strings.Repeat("a", 64)
	actor, err := NuevaEvidenciaActorConsultaAccesos(
		context.Background(), "principal_interno_123456",
		&seudonimizadorAccesosPrueba{seudonimo: seudonimo},
	)
	if err != nil {
		t.Fatal(err)
	}
	minimo := filtroValidoPrueba()
	completo := minimo
	completo.ModuloID = "vec.module.bolsa"
	completo.Accion = "expediente.leer"
	completo.FinalidadAcceso = "tramitacion"
	completo.RecursoRef = "expediente:sha256:" + strings.Repeat("b", 64)
	completo.ExpedienteRef = "expediente:sha256:" + strings.Repeat("c", 64)
	completo.Resultado = "permitido"
	completo.VersionObjeto = 7
	completo.Cursor, err = NuevoCursorConsultaAdministrativaAccesos(
		"cursor:v1:" + strings.Repeat("d", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	for nombre, filtro := range map[string]FiltroConsultaAdministrativaAccesos{
		"minimo": minimo, "completo": completo,
	} {
		t.Run(nombre, func(t *testing.T) {
			recurso, err := RecursoAutorizableConsultaAdministrativaAccesos(
				filtro, actor,
			)
			if err != nil || recurso.Validar() != nil ||
				len(recurso.Atributos) != 16 {
				t.Fatalf("recurso exacto invalido: %+v / %v", recurso, err)
			}
			for clave, valor := range recurso.Atributos {
				if valor == "" {
					t.Fatalf("atributo vacio %q", clave)
				}
			}
		})
	}
}

func TestRespuestaDebeCumplirCadaFiltroYElIntervalo(t *testing.T) {
	t.Parallel()
	filtro := filtroValidoPrueba()
	filtro.ModuloID, filtro.Accion = "vec.module.bolsa", "expediente.leer"
	filtro.FinalidadAcceso, filtro.Resultado = "tramitacion", "permitido"
	filtro.RecursoRef = "expediente:sha256:" + strings.Repeat("b", 64)
	filtro.ExpedienteRef = "expediente:sha256:" + strings.Repeat("c", 64)
	filtro.VersionObjeto = 7
	base := ResumenAccesoAdministrativo{
		RegistroRef:        "acc_" + strings.Repeat("d", 40),
		ActorSeudonimizado: filtro.ActorSeudonimizado,
		ModuloID:           filtro.ModuloID, Accion: filtro.Accion,
		Finalidad: filtro.FinalidadAcceso, RecursoRef: filtro.RecursoRef,
		ExpedienteRef: filtro.ExpedienteRef, Resultado: filtro.Resultado,
		VersionObjeto:  filtro.VersionObjeto,
		OcurridoEn:     filtro.DesdeInclusive.Add(time.Minute),
		VersionEsquema: filtro.Version,
	}
	if !registroCumpleFiltroConsultaAccesos(base, filtro) {
		t.Fatal("fila valida rechazada")
	}
	casos := map[string]func(*ResumenAccesoAdministrativo){
		"actor": func(r *ResumenAccesoAdministrativo) {
			r.ActorSeudonimizado = strings.Replace(r.ActorSeudonimizado, "a", "b", 1)
		},
		"modulo":     func(r *ResumenAccesoAdministrativo) { r.ModuloID = "otro" },
		"accion":     func(r *ResumenAccesoAdministrativo) { r.Accion = "otra" },
		"finalidad":  func(r *ResumenAccesoAdministrativo) { r.Finalidad = "otra" },
		"recurso":    func(r *ResumenAccesoAdministrativo) { r.RecursoRef = "otro" },
		"expediente": func(r *ResumenAccesoAdministrativo) { r.ExpedienteRef = "otro" },
		"resultado":  func(r *ResumenAccesoAdministrativo) { r.Resultado = "denegado" },
		"version":    func(r *ResumenAccesoAdministrativo) { r.VersionObjeto++ },
		"esquema":    func(r *ResumenAccesoAdministrativo) { r.VersionEsquema++ },
		"desde":      func(r *ResumenAccesoAdministrativo) { r.OcurridoEn = filtro.DesdeInclusive.Add(-time.Microsecond) },
		"hasta":      func(r *ResumenAccesoAdministrativo) { r.OcurridoEn = filtro.HastaExclusive },
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			fila := base
			mutar(&fila)
			if registroCumpleFiltroConsultaAccesos(fila, filtro) {
				t.Fatal("fila fuera del filtro aceptada")
			}
		})
	}
}

func TestSolicitudNoUsaAuditEntryComoAutoridad(t *testing.T) {
	t.Parallel()
	filtro := filtroValidoPrueba()
	huella, err := HuellaFiltroConsultaAdministrativaAccesos(filtro)
	if err != nil {
		t.Fatal(err)
	}
	entrada := entradaValidaPrueba()
	entrada.ModuleID = ModuloAuditoriaConsultaAccesos
	entrada.Action = AccionAuditoriaConsultaAccesos
	entrada.Purpose = filtro.FinalidadDeLaConsulta
	entrada.SubjectRef = "consulta-accesos:sha256:" + huella
	entrada.ObjectVersion = 1
	if _, err := NuevaSolicitudConsultaAdministrativaAccesos(
		filtro,
		entrada,
		EvidenciaActorConsultaAccesos{},
		vecports.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{},
	); !errors.Is(err, ErrConsultaAdministrativaAccesosDenegada) {
		t.Fatal("un AuditEntry fabricable concedio la consulta")
	}
}

func TestEntradaExigeActorSeudonimizadoYRolesCanonicos(t *testing.T) {
	t.Parallel()
	entrada := entradaValidaPrueba()
	entrada.ActorRoles = []string{"auditor", "supervisor"}
	if err := ValidarEntradaRegistroAcceso(entrada); err != nil {
		t.Fatalf("entrada valida rechazada: %v", err)
	}
	entrada.ActorRoles = []string{"supervisor", "auditor"}
	if !errors.Is(
		ValidarEntradaRegistroAcceso(entrada),
		ErrRegistroAccesosInvalido,
	) {
		t.Fatal("se aceptaron roles no canonicos")
	}
	entrada = entradaValidaPrueba()
	entrada.ActorID = "DNI-real"
	if !errors.Is(
		ValidarEntradaRegistroAcceso(entrada),
		ErrRegistroAccesosInvalido,
	) {
		t.Fatal("se acepto identidad no seudonimizada")
	}
	entrada = entradaValidaPrueba()
	entrada.Signature = strings.Repeat("a", 64)
	if !errors.Is(
		ValidarEntradaRegistroAcceso(entrada),
		ErrRegistroAccesosInvalido,
	) {
		t.Fatal("se acepto una firma aportada por cliente")
	}
	if ActorSeudonimizadoValido(
		"hmac-sha256:bolsa_accesos_v1:" + strings.Repeat("A", 64),
	) {
		t.Fatal("seudonimo hexadecimal mayusculo aceptado")
	}
}

func filtroValidoPrueba() FiltroConsultaAdministrativaAccesos {
	desde := time.Date(2026, 7, 25, 10, 0, 0, 0, time.UTC)
	return FiltroConsultaAdministrativaAccesos{
		Version: 1,
		ActorSeudonimizado: "hmac-sha256:bolsa_accesos_v1:" +
			strings.Repeat("a", 64),
		DesdeInclusive:        desde,
		HastaExclusive:        desde.Add(time.Hour),
		Limite:                25,
		FinalidadDeLaConsulta: "control-interno",
	}
}

func entradaValidaPrueba() vecdomain.AuditEntry {
	return vecdomain.AuditEntry{
		ActorID:    "hmac-sha256:bolsa_accesos_v1:" + strings.Repeat("a", 64),
		Purpose:    "tramitacion",
		Action:     "expediente.leer",
		ModuleID:   "vec.module.bolsa",
		SubjectRef: "expediente:sha256:" + strings.Repeat("b", 64),
		Result:     "permitido",
		CorrelationRef: "correlacion:sha256:" +
			strings.Repeat("c", 64),
		OccurredAt: time.Date(2026, 7, 25, 10, 0, 0, 0, time.UTC),
	}
}
