package mibolsa

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

type lectorModalidadesAplicacionPrueba struct {
	llamadas int
	captura  bolsa.CapturaLecturaPreferencias
}

func (l *lectorModalidadesAplicacionPrueba) LeerPreferenciasModalidades(_ context.Context, captura bolsa.CapturaLecturaPreferencias) (bolsa.EstadoPreferenciasModalidades, error) {
	l.llamadas++
	l.captura = captura
	return bolsa.EstadoPreferenciasModalidades{}, nil
}

func TestLecturaModalidadesProyeccionCerradaYConcesionComun(t *testing.T) {
	e := nuevoEntorno(t)
	i := &e.fuente.instantanea
	i.VersionRol.RolID = "candidato_bolsa_portal_historial_propio_desarrollo"
	i.AsignacionPerfil.VersionRolRef = i.VersionRol.Referencia()
	i.ControlVigenciaVersionRol.VersionRolRef = i.VersionRol.Referencia()
	c := &i.VersionRol.Concesiones[0]
	c.Accion, c.ModuloID, c.TipoRecurso = bolsa.AccionConsultarPreferenciasBolsas, bolsa.ModuloMiBolsa, bolsa.TipoRecursoMiBolsa
	c.Finalidades, c.GarantiaMinima = []string{bolsa.FinalidadPreferenciasBolsas}, core.AuthAssuranceHigh
	c.CamposPermitidos = bolsa.CamposConcesionPreferenciasPropias()
	lector, auditor := new(lectorModalidadesAplicacionPrueba), new(auditorPreferenciasAplicacionPrueba)
	catalogo := &catalogoModalidadesPrueba{disponibles: []bolsa.ModalidadPreferencia{{Clave: "vacante"}}}
	reloj := &relojAutorizacionServicioPrueba{ahora: e.ahora}
	servicio, err := NuevaLecturaPreferenciasModalidades(lector, auditor, e.fuente, catalogo, reloj)
	if err != nil {
		t.Fatal(err)
	}
	estado, err := servicio.ConsultarPreferenciasModalidades(t.Context(), e.orden)
	var captura struct {
		Campos []string `json:"campos_permitidos"`
		Motivo string   `json:"motivo_ref"`
	}
	if err != nil || json.Unmarshal(lector.captura.OrdenJSON(), &captura) != nil ||
		!slices.Equal(captura.Campos, bolsa.CamposLecturaPreferenciasModalidades()) ||
		captura.Motivo != "lectura_preferencias_modalidades_correcta" ||
		estado.ModalidadesGuardadas != nil || len(estado.ModalidadesDisponibles) != 1 {
		t.Fatalf("lectura=%+v err=%v campos=%v motivo=%s", estado, err, captura.Campos, captura.Motivo)
	}
	// La misma concesión ampliada conserva el acceso al orden de bolsas.
	bolsas, err := NuevaLecturaPreferenciasBolsas(new(lectorPreferenciasAplicacionPrueba), auditor, e.fuente, reloj)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = bolsas.ConsultarPreferencias(t.Context(), e.orden); err != nil {
		t.Fatal(err)
	}
	for _, campos := range [][]string{bolsa.CamposLecturaPreferenciasBolsas(), append(bolsa.CamposConcesionPreferenciasPropias(), "campo_ajeno")} {
		c.CamposPermitidos = campos
		if _, err = servicio.ConsultarPreferenciasModalidades(t.Context(), e.orden); !errors.Is(err, core.ErrAutorizacionDenegada) || lector.llamadas != 1 || catalogo.llamadas != 1 {
			t.Fatalf("concesión inválida permitió lectura: %v", err)
		}
	}
	c.CamposPermitidos = bolsa.CamposConcesionPreferenciasPropias()
	catalogo.err = bolsa.ErrPreferenciasNoDisponibles
	if _, err = servicio.ConsultarPreferenciasModalidades(t.Context(), e.orden); !errors.Is(err, bolsa.ErrPreferenciasNoDisponibles) || lector.llamadas != 1 || auditor.llamadas != 3 {
		t.Fatalf("fallo de catálogo sin intento: %v", err)
	}
	interno, err := NuevaLecturaPreferenciasModalidadesInterna(lector, auditor, e.fuente, catalogo, reloj)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = interno.ConsultarPreferenciasModalidades(t.Context(), e.orden); !errors.Is(err, core.ErrAutorizacionDenegada) || lector.llamadas != 1 {
		t.Fatalf("sesión externa cruzó superficie: %v", err)
	}
}

// AD300 exige que la fila de auditoría diga qué proyección se leyó.
func TestMotivoLecturaPreferenciasSegunProyeccion(t *testing.T) {
	if m := motivoLecturaPreferencias(bolsa.CamposLecturaPreferenciasBolsas()); m != "lectura_preferencias_bolsas_correcta" {
		t.Fatalf("motivo de bolsas: %s", m)
	}
	if m := motivoLecturaPreferencias(bolsa.CamposLecturaPreferenciasModalidades()); m != "lectura_preferencias_modalidades_correcta" {
		t.Fatalf("motivo de modalidades: %s", m)
	}
}
