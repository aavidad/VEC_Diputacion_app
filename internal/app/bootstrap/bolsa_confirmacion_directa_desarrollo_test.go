package bootstrap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestConfirmacionDirectaAdoptaVersionPublicadaSinPublicarSemilla(t *testing.T) {
	p, autoridad, datos := politicaProvisionBolsaPrueba(t, 0)
	i, err := nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(
		datos.PrincipalID, datos.PerfilActivoRef, p.soporte.unidadRef, p.soporte.ambitoRef, p.reloj.Ahora(), 5)
	if err != nil {
		t.Fatal(err)
	}
	i.VersionRol.Version = 24
	i.VersionRol.PublicadaPor = "seguridad:acto-gobernado"
	i.AsignacionPerfil.EmitidaPor = "identidad:acto-gobernado"
	i.VersionRol.Concesiones = append(i.VersionRol.Concesiones, dominiovec.ConcesionRol{
		Accion: puertosbolsa.AccionConfirmarAceptacionDirecta, ModuloID: "bolsa",
		TipoRecurso:    puertosbolsa.TipoRecursoConfirmarAceptacionDirecta,
		Finalidades:    []string{puertosbolsa.FinalidadConfirmarAceptacionDirecta},
		GarantiaMinima: dominiovec.AuthAssuranceHigh}, concesionCargaConvocaPrueba())
	i.AsignacionPerfil.VersionRolRef = i.VersionRol.Referencia()
	i.ControlVigenciaVersionRol.VersionRolRef = i.VersionRol.Referencia()
	i.ControlVigenciaVersionRol.ActualizadoPor = i.VersionRol.PublicadaPor
	if err := i.Validar(); err != nil {
		t.Fatal(err)
	}
	autoridad.leida, autoridad.encontrada = instantaneaPublicadaDesarrollo{instantanea: i}, true
	if err := p.PublicarInicial(context.Background()); err != nil || autoridad.publicadas != 0 ||
		autoridad.preparadas != 0 || p.instantanea.VersionRol.Version != 24 {
		t.Fatalf("se reconstruyó o rechazó la versión publicada: %v", err)
	}
}

func TestConfirmacionDirectaAdoptaSoloExtrasExactosIncluidoB1(t *testing.T) {
	confirmacion := dominiovec.ConcesionRol{Accion: puertosbolsa.AccionConfirmarAceptacionDirecta,
		ModuloID: "bolsa", TipoRecurso: puertosbolsa.TipoRecursoConfirmarAceptacionDirecta,
		Finalidades:    []string{puertosbolsa.FinalidadConfirmarAceptacionDirecta},
		GarantiaMinima: dominiovec.AuthAssuranceHigh}
	b1 := dominiovec.ConcesionRol{Accion: puertosbolsa.AccionConfirmarCargaConvoca,
		ModuloID: puertosbolsa.ModuloCargaConvoca, TipoRecurso: puertosbolsa.TipoRecursoCargaConvoca,
		Finalidades:    []string{puertosbolsa.FinalidadConfirmarCargaConvoca},
		GarantiaMinima: dominiovec.AuthAssuranceHigh}
	if !concesionesConfirmacionDirectaExactas([]dominiovec.ConcesionRol{confirmacion, b1}) ||
		!contieneAccionConfirmacionDirecta([]dominiovec.ConcesionRol{confirmacion}) {
		t.Fatal("la concesión nominal junto a B1 exacta fue rechazada")
	}
	consultaS6 := dominiovec.ConcesionRol{Accion: puertosbolsa.AccionConsultarTransicionesSituacion,
		ModuloID: "bolsa", TipoRecurso: puertosbolsa.TipoRecursoGobiernoTransicionesSituacion,
		Finalidades:      []string{puertosbolsa.FinalidadConsultaTransicionesSituacion},
		CamposPermitidos: []string{puertosbolsa.CampoConsultaTransicionesSituacion},
		GarantiaMinima:   dominiovec.AuthAssuranceHigh}
	if !concesionesConfirmacionDirectaExactas([]dominiovec.ConcesionRol{confirmacion, b1, consultaS6}) {
		t.Fatal("la consulta S6 exacta no convivió con el acto")
	}
	alterada := b1
	alterada.TipoRecurso = "otro_recurso"
	consultaAlterada := consultaS6
	consultaAlterada.TipoRecurso = "otro_recurso"
	for _, extras := range [][]dominiovec.ConcesionRol{
		{confirmacion, alterada}, {confirmacion, b1, b1}, {confirmacion, consultaAlterada},
		{confirmacion, {Accion: "bolsa.otra_accion", GarantiaMinima: dominiovec.AuthAssuranceHigh}},
		{confirmacion, b1, {Accion: "bolsa.otra_accion", GarantiaMinima: dominiovec.AuthAssuranceHigh}},
	} {
		if concesionesConfirmacionDirectaExactas(extras) {
			t.Fatalf("se admitió concesión ajena o B1 alterada: %+v", extras)
		}
	}
}

func TestConfirmacionDirectaMuxLlegaAlHandlerNominal(t *testing.T) {
	ruta := bolsahttp.RutaBolsasGestion + "/bolsa:01/candidatos/participacion:01/respuestas/llamamiento:01/confirmacion"
	llamadas := 0
	m := &manejadorParticipacionBolsaDesarrollo{confirmacionDirecta: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		llamadas++
		w.WriteHeader(http.StatusConflict)
	})}
	w := httptest.NewRecorder()
	m.ServeHTTP(w, httptest.NewRequest(http.MethodPost, ruta, nil))
	if w.Code != http.StatusConflict || llamadas != 1 {
		t.Fatalf("el mux no entregó el acto: estado=%d llamadas=%d", w.Code, llamadas)
	}
}
