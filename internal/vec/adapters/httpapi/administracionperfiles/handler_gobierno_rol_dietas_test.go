package administracionperfiles

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type servicioGobiernoRolDietasPrueba struct{ llamadas int }

func (s *servicioGobiernoRolDietasPrueba) ProponerGobiernoRolNuevo(context.Context,
	domain.SolicitudPropuestaGobiernoPerfil) (ports.ResultadoPropuestaGobiernoRolNuevo, error) {
	s.llamadas++
	return ports.ResultadoPropuestaGobiernoRolNuevo{}, domain.ErrPlanGobiernoPerfilInvalido
}

func (s *servicioGobiernoRolDietasPrueba) CerrarGobiernoRolPorReferencia(context.Context,
	domain.SolicitudCierreGobiernoRolPorReferencia) (domain.CierreGobiernoPerfil, error) {
	s.llamadas++
	return domain.CierreGobiernoPerfil{}, domain.ErrPlanGobiernoPerfilInvalido
}

type fuenteGobiernoRolDietasPrueba struct{ llamadas int }

func (f *fuenteGobiernoRolDietasPrueba) ObtenerCatalogoAccionesAdministracionV1(context.Context,
	string, int, string) (domain.CatalogoAccionesAdministracionV1, error) {
	f.llamadas++
	return domain.CatalogoAccionesAdministracionV1{}, domain.ErrCatalogoAccionesAdministracionInvalido
}

func TestGobiernoRolDietasHTTPAdmiteSoloCardinalidadNominal(t *testing.T) {
	casos := []struct {
		rolID        string
		numero       int
		aceptada     bool
		entradaUnica bool
	}{
		{domain.RolIDPersonaDietasComisiones, 3, true, false},
		{domain.RolIDPersonaDietasComisiones, 2, false, false},
		{domain.RolIDPersonaDietasRutas, 2, true, false},
		{domain.RolIDPersonaDietasRutas, 3, false, false},
		{"otro_rol", 3, false, false},
		{"otro_rol", 2, false, false},
		{"otro_rol", 1, true, true},
	}
	for _, caso := range casos {
		t.Run(caso.rolID+"_"+strings.Repeat("x", caso.numero), func(t *testing.T) {
			sesion := &sesionPrueba{resultado: sesionAplicacionNominalPrueba(t)}
			auditor := &auditorPrueba{}
			h, err := NuevoHandlerUsuariosMetadatos("https://admin.example.test", sesion, &lecturasPrueba{}, auditor)
			if err != nil {
				t.Fatal(err)
			}
			servicio := &servicioGobiernoRolDietasPrueba{}
			fuente := &fuenteGobiernoRolDietasPrueba{}
			if err := h.ConGobiernoRolNuevo(servicio, fuente, relojFocal{}); err != nil {
				t.Fatal(err)
			}
			selecciones := make([]seleccionGobiernoRolDTO, caso.numero)
			for i := range selecciones {
				selecciones[i] = seleccionGobiernoRolDTO{
					EntradaRef: "entrada:dietas:" + string(rune('a'+i)), EntradaVersion: 1,
					EntradaHuellaSHA256: strings.Repeat("b", 64)}
			}
			dto := solicitudGobiernoRolProponerDTO{
				OperacionRef: "propuesta_admin:" + strings.Repeat("a", 32),
				CatalogoRef:  "catalogo:dietas", CatalogoVersion: 1,
				CatalogoHuellaSHA256: strings.Repeat("c", 64), RolID: caso.rolID,
				Nombre: "Dietas", Selecciones: selecciones,
			}
			if caso.entradaUnica {
				dto.Selecciones = nil
				dto.EntradaRef = selecciones[0].EntradaRef
				dto.EntradaVersion = 1
				dto.EntradaHuellaSHA256 = selecciones[0].EntradaHuellaSHA256
			}
			cuerpo, err := json.Marshal(dto)
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticionADMIN(http.MethodPost, RutaGobiernoRolProponer, string(cuerpo)))
			if caso.aceptada {
				if fuente.llamadas != 1 || w.Code != http.StatusServiceUnavailable || servicio.llamadas != 0 {
					t.Fatalf("propuesta nominal no llegó a la fuente: estado=%d fuente=%d servicio=%d", w.Code, fuente.llamadas, servicio.llamadas)
				}
			} else if fuente.llamadas != 0 || servicio.llamadas != 0 || w.Code != http.StatusBadRequest {
				t.Fatalf("propuesta ajena llegó a la fuente: estado=%d fuente=%d servicio=%d", w.Code, fuente.llamadas, servicio.llamadas)
			}
		})
	}
}
