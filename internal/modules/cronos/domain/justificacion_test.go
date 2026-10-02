package domain

import (
	"errors"
	"strings"
	"testing"
)

func ejemploJustificacion() (SolicitudJustificable, PoliticaJustificacion, VinculoJustificacion) {
	p := PoliticaJustificacion{Referencia: "politica:justificacion:v1", Version: 1, SHA256: strings.Repeat("a", 64), CatalogoVersionRef: "catalogo:permiso:v1", PermisoRef: "permiso:neutral", TipoDocumentalRef: "ref:" + strings.Repeat("b", 64), CustodioID: "custodia.interna", MotivosRef: []string{"motivo:documentacion:conforme", "motivo:documentacion:incompleta"}}
	s := SolicitudJustificable{SolicitudRef: "permiso:cronos:solicitud:ensayo001", EmpleadoRef: "emp_AAAAAAAAAAAAAAAAAAAAAA", CatalogoVersionRef: p.CatalogoVersionRef, PermisoRef: p.PermisoRef, ExpedienteDocumentalRef: "ref:" + strings.Repeat("c", 64), Version: 3, Estado: EstadoPermisoConcedido, JustificanteExigido: true}
	v := VinculoJustificacion{SolicitudRef: s.SolicitudRef, EmpleadoRef: s.EmpleadoRef, CatalogoVersionRef: s.CatalogoVersionRef, PermisoRef: s.PermisoRef, ExpedienteDocumentalRef: s.ExpedienteDocumentalRef, Documento: DocumentoJustificacion{ID: "ref:" + strings.Repeat("d", 64), Version: 1, SHA256: strings.Repeat("e", 64), CustodioID: p.CustodioID, CustodiaRef: "original:ensayo001"}}
	return s, p, v
}
func TestJustificacionNoCambiaConcesionNiSaldo(t *testing.T) {
	for _, decision := range []EstadoJustificacion{JustificacionAceptada, JustificacionRechazada} {
		s, p, v := ejemploJustificacion()
		antes := s
		j, e := PrepararAnexoJustificacion(s, p, nil, v, 0)
		if e != nil {
			t.Fatal(e)
		}
		original := j
		r, e := PrepararRevisionJustificacion(s, p, j, v, 1, decision, p.MotivosRef[0])
		if e != nil || r.Version != 2 || r.Estado != decision || s != antes || j != original {
			t.Fatal("revision altera antecedente", e)
		}
		if _, e = PrepararRevisionJustificacion(s, p, r, v, 2, decision, p.MotivosRef[0]); !errors.Is(e, ErrJustificacionConflicto) {
			t.Fatal("segunda revision", e)
		}
	}
}
func TestJustificacionEnlaceExactoYCAS(t *testing.T) {
	s, p, v := ejemploJustificacion()
	j, _ := PrepararAnexoJustificacion(s, p, nil, v, 0)
	for nombre, mutar := range map[string]func(*VinculoJustificacion){
		"solicitud": func(v *VinculoJustificacion) { v.SolicitudRef += "2" }, "empleado": func(v *VinculoJustificacion) { v.EmpleadoRef = "emp_BBBBBBBBBBBBBBBBBBBBBB" }, "catalogo": func(v *VinculoJustificacion) { v.CatalogoVersionRef += "2" }, "permiso": func(v *VinculoJustificacion) { v.PermisoRef += "2" }, "expediente": func(v *VinculoJustificacion) { v.ExpedienteDocumentalRef = "ref:" + strings.Repeat("f", 64) }, "id": func(v *VinculoJustificacion) { v.Documento.ID = "ref:" + strings.Repeat("f", 64) }, "version": func(v *VinculoJustificacion) { v.Documento.Version++ }, "sha": func(v *VinculoJustificacion) { v.Documento.SHA256 = strings.Repeat("f", 64) }, "custodio": func(v *VinculoJustificacion) { v.Documento.CustodioID = "otro.interno" }, "referencia": func(v *VinculoJustificacion) { v.Documento.CustodiaRef += "2" },
	} {
		t.Run(nombre, func(t *testing.T) {
			c := v
			mutar(&c)
			if _, e := PrepararRevisionJustificacion(s, p, j, c, 1, JustificacionAceptada, p.MotivosRef[0]); e == nil {
				t.Fatal("acepta vinculacion sustituida")
			}
		})
	}
	if _, e := PrepararAnexoJustificacion(s, p, nil, v, 1); !errors.Is(e, ErrJustificacionConflicto) {
		t.Fatal(e)
	}
	if _, e := PrepararRevisionJustificacion(s, p, j, v, 2, JustificacionAceptada, p.MotivosRef[0]); !errors.Is(e, ErrJustificacionConflicto) {
		t.Fatal(e)
	}
	if _, e := PrepararRevisionJustificacion(s, p, j, v, 1, JustificacionAceptada, "diagnostico:libre"); !errors.Is(e, ErrJustificacionInvalida) {
		t.Fatal(e)
	}
	nuevo := v
	nuevo.Documento.ID = "ref:" + strings.Repeat("f", 64)
	n, e := PrepararAnexoJustificacion(s, p, &j, nuevo, 1)
	if e != nil || n.Version != 2 || j.Vinculo != v {
		t.Fatal("no conserva anterior", e)
	}
}
func TestJustificacionSoloPosteriorYExigida(t *testing.T) {
	s, p, v := ejemploJustificacion()
	for _, estado := range []EstadoSolicitudPermiso{EstadoPermisoSolicitado, EstadoPermisoDenegado, EstadoPermisoCancelado, EstadoPermisoPendienteAdministracion} {
		s.Estado = estado
		if _, e := PrepararAnexoJustificacion(s, p, nil, v, 0); e == nil {
			t.Fatal("estado admitido", estado)
		}
	}
	s.Estado = EstadoPermisoConcedido
	s.JustificanteExigido = false
	if _, e := PrepararAnexoJustificacion(s, p, nil, v, 0); e == nil {
		t.Fatal("justificante no exigido")
	}
}
func TestJustificacionCustodiaOpacaSinRutas(t *testing.T) {
	_, _, v := ejemploJustificacion()
	for _, r := range []string{"../original", "https://externo", "/ruta", "archivo/uno", "documento?secret", "diagnostico texto"} {
		v.Documento.CustodiaRef = r
		if v.Documento.Validar() == nil {
			t.Fatal("referencia aceptada", r)
		}
	}
}

func TestMaterialJustificacionRechazaVinculoIncompleto(t *testing.T) {
	s, p, v := ejemploJustificacion()
	m := MaterialJustificacion{
		ActorRef: "per_AAAAAAAAAAAAAAAAAAAAAA", PerfilRef: "prf_AAAAAAAAAAAAAAAAAAAAAA",
		ClaveOperacion: "ref:" + strings.Repeat("a", 64), Accion: AccionAnexarJustificacion,
		SolicitudVersion: s.Version, VersionEsperada: 0, PoliticaRef: p.Referencia,
		PoliticaVersion: p.Version, PoliticaSHA256: p.SHA256, Vinculo: v,
	}
	if _, err := m.Canonico(); err != nil {
		t.Fatal("material válido rechazado", err)
	}
	for nombre, mutar := range map[string]func(*VinculoJustificacion){
		"solicitud":  func(v *VinculoJustificacion) { v.SolicitudRef = "" },
		"empleado":   func(v *VinculoJustificacion) { v.EmpleadoRef = "" },
		"catalogo":   func(v *VinculoJustificacion) { v.CatalogoVersionRef = "" },
		"permiso":    func(v *VinculoJustificacion) { v.PermisoRef = "" },
		"expediente": func(v *VinculoJustificacion) { v.ExpedienteDocumentalRef = "" },
	} {
		t.Run(nombre, func(t *testing.T) {
			c := m
			mutar(&c.Vinculo)
			if _, err := c.Canonico(); !errors.Is(err, ErrJustificacionInvalida) {
				t.Fatal("se aceptó material sin vínculo completo", err)
			}
		})
	}
}
