package bootstrap

import (
	"context"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
)

func TestNuevaPublicacionCTUsaVigenciaConfiguradaSinCambiarRaiz(t *testing.T) {
	ahora := time.Date(2026, 10, 16, 13, 25, 0, 0, time.UTC)
	derivador := nuevoDerivadorIdempotenciaPrueba(t, 2, 1)
	t.Setenv(config.EnvExecutionProfile, config.ExecutionProfileProduction)
	t.Setenv(config.EnvAuthMode, config.AuthModeDisabled)
	t.Setenv(config.EnvDevelopmentGuard, "")
	t.Setenv(config.EnvGobiernoAdminValidezHoras, "")
	comprobar := func(nombre, override string, horas int, aceptada bool) {
		t.Helper()
		t.Run(nombre, func(t *testing.T) {
			t.Setenv(config.EnvGobiernoAdminValidezHoras, override)
			m, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(derivador, ahora)
			if !aceptada {
				if err == nil {
					m.borrarCopiasEfimeras()
					t.Fatal("se publicó con una vigencia inválida")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			defer m.borrarCopiasEfimeras()
			if got := m.expiraEn.Sub(m.publicadaEn); got != time.Duration(horas)*time.Hour {
				t.Fatalf("vigencia=%v, esperada=%d horas", got, horas)
			}
			if horas < 24 && !m.publicadaEn.Equal(ahora) {
				t.Fatal("el plazo breve nació antes de la publicación")
			}
			if m.validaDesde != time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) ||
				m.validaHasta != time.Date(2036, 1, 1, 0, 0, 0, 0, time.UTC) ||
				!m.expiraEn.Before(m.validaHasta) {
				t.Fatal("la publicación alteró o rebasó la raíz")
			}
		})
	}
	comprobar("predeterminada", "", 24, true)
	comprobar("produccion_seis_horas", "6", 6, true)
	comprobar("produccion_25_horas", "25", 0, false)
	t.Setenv(config.EnvExecutionProfile, config.ExecutionProfileDevelopment)
	t.Setenv(config.EnvAuthMode, config.AuthModeDevelopment)
	t.Setenv(config.EnvDevelopmentGuard, config.DevelopmentGuardAcknowledgement)
	comprobar("desarrollo_90_dias", "2160", 2160, true)
	comprobar("desarrollo_90_dias_y_una_hora", "2161", 0, false)
	t.Setenv(config.EnvDevelopmentGuard, "")
	comprobar("desarrollo_incompleto", "2160", 0, false)
	t.Setenv(config.EnvExecutionProfile, config.ExecutionProfileProduction)
	comprobar("produccion_con_modo_desarrollo", "2160", 0, false)
}

func TestPublicacionExternaNuevaConservaClavesYUsaVigenciaComun(t *testing.T) {
	t.Setenv(config.EnvExecutionProfile, config.ExecutionProfileDevelopment)
	t.Setenv(config.EnvAuthMode, config.AuthModeDevelopment)
	t.Setenv(config.EnvDevelopmentGuard, config.DevelopmentGuardAcknowledgement)
	t.Setenv(config.EnvGobiernoAdminValidezHoras, "2160")
	materiales := materialesPublicadosPrueba(t, consumidoresPortalExternoV3...)
	defer func() {
		for _, lista := range materiales {
			borrarMaterialesV3PortalExterno(lista)
		}
	}()
	lista := ordenarMaterialExternoV3(materiales)
	if len(lista) != 23 {
		t.Fatalf("material externo incompleto: %d claves", len(lista))
	}
	coordenadas := make([]coordenadaClaveExternaV3, len(lista))
	for i, m := range lista {
		coordenadas[i] = coordenadaClaveExternaV3{m.audienciaConsumo, uint64(i + 1), uint64(i + 1), uint64(i + 1)}
	}
	publicada := time.Date(2026, 10, 16, 13, 25, 0, 0, time.UTC)
	expira, err := caducidadNuevaPublicacionConfianza(publicada, lista[0].validaHasta)
	if err != nil {
		t.Fatal(err)
	}
	if err := coordinarMaterialExternoV3(lista, "confianza:atestacion:externo:31", 31,
		publicada, expira, lista[0].validaDesde, lista[0].validaHasta, coordenadas); err != nil {
		t.Fatal(err)
	}
	for i, m := range lista {
		if m.expiraEn.Sub(m.publicadaEn) != 90*24*time.Hour ||
			m.validaHasta != time.Date(2036, 1, 1, 0, 0, 0, 0, time.UTC) ||
			m.claveHMACVersion != coordenadas[i].Version {
			t.Fatalf("publicación externa alteró vigencia, raíz o clave %d", i)
		}
	}
	if _, err := caducidadNuevaPublicacionConfianza(lista[0].validaHasta.Add(-time.Hour), lista[0].validaHasta); err == nil {
		t.Fatal("publicación nueva rebasó la raíz")
	}
}

func TestPublicacionTecnicaCTSeAdoptaSinOverrideYRechazaOrigenAjeno(t *testing.T) {
	t.Setenv(config.EnvExecutionProfile, config.ExecutionProfileDevelopment)
	t.Setenv(config.EnvAuthMode, config.AuthModeDevelopment)
	t.Setenv(config.EnvDevelopmentGuard, config.DevelopmentGuardAcknowledgement)
	t.Setenv(config.EnvGobiernoAdminValidezHoras, "1440")
	ahora := time.Date(2026, 10, 16, 13, 25, 0, 0, time.UTC)
	m := materialRenovableCTPrueba(t, ahora)
	if m.expiraEn.Sub(m.publicadaEn) != 60*24*time.Hour {
		t.Fatal("la publicación nueva no tomó las 1440 horas")
	}
	t.Setenv(config.EnvGobiernoAdminValidezHoras, "")
	if !publicacionTecnicaCTDesarrollo(m.publicadaEn, m.expiraEn, m.configuracionRef, m.configuracionOrden) ||
		!vigenciaPublicadaCTDesarrolloValida(m.publicadaEn, m.expiraEn, m.validaHasta, ahora) {
		t.Fatal("el reinicio sin override rechazó la publicación vigente")
	}
	r := &relojRenovableCTPrueba{}
	r.fijar(ahora)
	f := fuenteRenovableCTPrueba(t, m, r)
	f.leer = func(context.Context, materialAtestacionContratacionTemporalDesarrollo, time.Time) (materialAtestacionContratacionTemporalDesarrollo, error) {
		return m, nil
	}
	f.renovar = func(context.Context, materialAtestacionContratacionTemporalDesarrollo, time.Time) (materialAtestacionContratacionTemporalDesarrollo, error) {
		t.Fatal("se intentó renovar una publicación vigente")
		return materialAtestacionContratacionTemporalDesarrollo{}, nil
	}
	if _, err := f.instantanea(t.Context()); err != nil || f.material.expiraEn != m.expiraEn {
		t.Fatalf("la adopción acortó la publicación: %v", err)
	}
	base := m.configuracionRef
	for _, tc := range []struct {
		nombre   string
		publica  time.Time
		expira   time.Time
		revision string
	}{
		{"fecha_ajena", m.publicadaEn.Add(time.Hour), m.expiraEn, base},
		{"revision_ajena", m.publicadaEn, m.expiraEn, "confianza:atestacion:otra:31"},
		{"exceso", m.publicadaEn, m.publicadaEn.Add(90*24*time.Hour + time.Hour), base},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			if publicacionTecnicaCTDesarrollo(tc.publica, tc.expira, tc.revision, m.configuracionOrden) {
				t.Fatal("publicación ajena reconocida como técnica")
			}
		})
	}
	t.Setenv(config.EnvExecutionProfile, config.ExecutionProfileProduction)
	if publicacionTecnicaCTDesarrollo(m.publicadaEn, m.expiraEn, m.configuracionRef, m.configuracionOrden) {
		t.Fatal("producción reconoció la publicación técnica larga")
	}
	t.Setenv(config.EnvGobiernoAdminValidezHoras, "6")
	breve := m.publicadaEn.Add(6 * time.Hour)
	if !publicacionTecnicaCTDesarrollo(m.publicadaEn, breve, m.configuracionRef, m.configuracionOrden) {
		t.Fatal("producción rechazó seis horas propias")
	}
	if !publicacionTecnicaCTDesarrollo(ahora, ahora.Add(6*time.Hour), m.configuracionRef, m.configuracionOrden) ||
		publicacionTecnicaCTDesarrollo(ahora, ahora.Add(24*time.Hour), m.configuracionRef, m.configuracionOrden) {
		t.Fatal("el reconocimiento subdiario no respeta su fecha de publicación")
	}
	if _, err := confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(
		m.configuracionRef, m.configuracionOrden, m.publicadaEn, m.expiraEn, m.raiz); err == nil {
		t.Fatal("el verificador común aceptó 60 días en producción")
	}
}

func TestPublicacionTecnicaCTDeSeisHorasRenuevaElMismoDia(t *testing.T) {
	t.Setenv(config.EnvExecutionProfile, config.ExecutionProfileDevelopment)
	t.Setenv(config.EnvAuthMode, config.AuthModeDevelopment)
	t.Setenv(config.EnvDevelopmentGuard, config.DevelopmentGuardAcknowledgement)
	t.Setenv(config.EnvGobiernoAdminValidezHoras, "6")
	ahora := time.Date(2026, 10, 16, 13, 25, 0, 0, time.UTC)
	previa := materialRenovableCTPrueba(t, ahora)
	if !previa.publicadaEn.Equal(ahora) || !previa.expiraEn.Equal(ahora.Add(6*time.Hour)) ||
		!vigenciaPublicadaCTDesarrolloValida(previa.publicadaEn, previa.expiraEn, previa.validaHasta, ahora) {
		t.Fatal("la publicación de seis horas nacida por la tarde no está vigente")
	}
	siguiente := previa
	var err error
	siguiente.publicadaEn, siguiente.expiraEn, err = fechasNuevaPublicacionTecnicaCT(previa.expiraEn, previa.validaHasta)
	if err != nil || !siguiente.publicadaEn.Equal(previa.expiraEn) ||
		!siguiente.expiraEn.Equal(previa.expiraEn.Add(6*time.Hour)) {
		t.Fatalf("la renovación del mismo día no avanza seis horas: %v", err)
	}
	siguiente.configuracionOrden++
	siguiente.configuracionRef = previa.configuracionRef + ":r" + numeroDecimal64(siguiente.configuracionOrden)
	siguiente.configuracion, err = confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(
		siguiente.configuracionRef, siguiente.configuracionOrden, siguiente.publicadaEn, siguiente.expiraEn, siguiente.raiz)
	if err != nil {
		t.Fatal(err)
	}
	siguiente.configuracionHuella, err = siguiente.configuracion.HuellaSHA256ParaGobierno()
	if err != nil {
		t.Fatal(err)
	}
	if !publicacionTecnicaCTDesarrollo(siguiente.publicadaEn, siguiente.expiraEn, siguiente.configuracionRef, siguiente.configuracionOrden) {
		t.Fatal("la segunda publicación del día no conserva el contrato técnico")
	}
	r := &relojRenovableCTPrueba{}
	r.fijar(previa.expiraEn)
	f := fuenteRenovableCTPrueba(t, previa, r)
	renovada := false
	f.renovar = func(context.Context, materialAtestacionContratacionTemporalDesarrollo, time.Time) (materialAtestacionContratacionTemporalDesarrollo, error) {
		renovada = true
		return siguiente, nil
	}
	f.leer = func(context.Context, materialAtestacionContratacionTemporalDesarrollo, time.Time) (materialAtestacionContratacionTemporalDesarrollo, error) {
		if renovada {
			return siguiente, nil
		}
		return previa, nil
	}
	if _, err := f.instantanea(t.Context()); err != nil || !renovada ||
		f.material.configuracionRef != siguiente.configuracionRef ||
		previa.configuracionRef == siguiente.configuracionRef ||
		!previa.expiraEn.Equal(ahora.Add(6*time.Hour)) {
		t.Fatalf("renovación sustituyó la referencia previa: %v", err)
	}
}
