package mibolsa

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

type catalogoModalidadesPrueba struct {
	disponibles []bolsa.ModalidadPreferencia
	err         error
	llamadas    int
}

func (c *catalogoModalidadesPrueba) ConsultarModalidadesDisponibles(context.Context) ([]bolsa.ModalidadPreferencia, error) {
	c.llamadas++
	return slices.Clone(c.disponibles), c.err
}

type escrituraModalidadesPrueba struct {
	solicitud    bolsa.GuardarPreferenciasModalidades
	llamadas     int
	registradoEn time.Time
	reutilizado  bool
	despues      func()
}

func (r *escrituraModalidadesPrueba) GuardarPreferenciasModalidades(_ context.Context,
	s bolsa.GuardarPreferenciasModalidades,
) (bolsa.ReciboPreferenciasModalidades, error) {
	r.llamadas++
	r.solicitud = s
	fecha := r.registradoEn
	if fecha.IsZero() {
		fecha = s.ConsultadaEn
	}
	if r.despues != nil {
		r.despues()
	}
	return bolsa.ReciboPreferenciasModalidades{ReciboRef: "recibo:modalidades:prueba",
		Version: s.VersionEsperada + 1, Modalidades: slices.Clone(s.Modalidades), RegistradoEn: fecha,
		Reutilizado: r.reutilizado}, nil
}

func modalidadPrueba(clave string, version int) bolsa.ModalidadPreferencia {
	return bolsa.ModalidadPreferencia{Clave: clave, Fuente: domain.ReferenciaEntradaCatalogo{
		CatalogoID: reglas.CatalogoContratacionTemporal, CatalogoVersion: version,
		CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: reglas.CTPrefijoModalidad + clave,
	}}
}

func entornoModalidadesPrueba(t *testing.T) (*entornoMiBolsa, *ServicioPreferenciasModalidades,
	*catalogoModalidadesPrueba, *escrituraModalidadesPrueba,
) {
	t.Helper()
	candidato := "can_" + strings.Repeat("c", 43)
	e := nuevoEntorno(t, func(i *domain.InstantaneaContextoActor) {
		i.Vinculos[0].Referencia = candidato
	})
	e.fuente.instantanea.AsignacionPerfil.Ambitos = []domain.AmbitoPerfil{{Clave: "candidato_ref", Valores: []string{candidato}}}
	concesion := &e.fuente.instantanea.VersionRol.Concesiones[0]
	concesion.Accion = bolsa.AccionOrdenarPreferenciasBolsas
	concesion.Finalidades = []string{bolsa.FinalidadPreferenciasBolsas}
	concesion.CamposPermitidos = nil
	reloj := &relojAutorizacionServicioPrueba{ahora: e.ahora}
	clave, err := confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3(
		"clave:capacidad:modalidades", 1, bytes.Repeat([]byte{0x73}, 32),
		"emisor:prueba:modalidades", bolsa.AudienciaOrdenarPreferenciasBolsas,
		confianza.EstadoClaveHMACCapacidadAtestacionV3Emision,
		e.ahora.Add(-time.Hour), e.ahora.Add(time.Hour), time.Time{}, 1, strings.Repeat("9", 64))
	exigir(t, err)
	e.emisor.capacidades, err = confianza.NuevoEmisorCapacidadesAtestacionAutorizacionV3(clave, reloj)
	exigir(t, err)
	catalogo := &catalogoModalidadesPrueba{disponibles: []bolsa.ModalidadPreferencia{
		modalidadPrueba("vacante", 1), modalidadPrueba("sustitucion", 1),
	}}
	repo := new(escrituraModalidadesPrueba)
	s, err := NuevoServicioPreferenciasModalidades(repo, catalogo, e.autorizador,
		proveedorPreferenciasPrueba{e.emisor}, reloj)
	exigir(t, err)
	return e, s, catalogo, repo
}

func TestModalidadesDistinguenTodasYNingunaEnReciboYMaterial(t *testing.T) {
	huellas := make([]string, 0, 2)
	for _, caso := range []struct {
		nombre    string
		seleccion []string
	}{
		{nombre: "todas", seleccion: nil},
		{nombre: "ninguna", seleccion: []string{}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e, s, catalogo, repo := entornoModalidadesPrueba(t)
			recibo, err := s.GuardarModalidades(t.Context(), e.orden, caso.seleccion, 0, "clave-modalidad-1")
			if err != nil || repo.llamadas != 1 || catalogo.llamadas != 1 || recibo.Version != 1 ||
				(recibo.Modalidades == nil) != (caso.seleccion == nil) ||
				(repo.solicitud.Modalidades == nil) != (caso.seleccion == nil) {
				t.Fatalf("selección no preservada: %+v %v", recibo, err)
			}
			datos, err := e.concesiones.orden.Datos()
			exigir(t, err)
			nominal, err := datos.Solicitud.Datos()
			exigir(t, err)
			a := nominal.Recurso.Atributos
			if nominal.Accion != bolsa.AccionOrdenarPreferenciasBolsas ||
				nominal.Finalidad != bolsa.FinalidadPreferenciasBolsas ||
				a["propiedad"] != "candidato" || a["tipo_preferencia"] != "modalidades" ||
				len(a["material_sha256"]) != 64 || repo.solicitud.Material.ValidarEstructura() != nil {
				t.Fatal("el material no quedó ligado a la autoridad propia")
			}
			huellas = append(huellas, a["material_sha256"])
		})
	}
	if len(huellas) != 2 || huellas[0] == huellas[1] {
		t.Fatal("nil y lista vacía comparten huella")
	}
}

func TestModalidadesCatalogoFallaAntesDeAutorizarYSeleccionAjenaNoEscribe(t *testing.T) {
	e, s, catalogo, repo := entornoModalidadesPrueba(t)
	catalogo.err = errors.New("catálogo no disponible")
	_, err := s.GuardarModalidades(t.Context(), e.orden, nil, 0, "clave-modalidad-2")
	if !errors.Is(err, bolsa.ErrPreferenciasNoDisponibles) || e.emisor.llamadas != 0 || repo.llamadas != 0 {
		t.Fatalf("fallo del catálogo alcanzó autoridad/escritura: %v", err)
	}
	catalogo.err = nil
	_, err = s.GuardarModalidades(t.Context(), e.orden, []string{"ajena"}, 0, "clave-modalidad-2")
	if !errors.Is(err, bolsa.ErrOrdenPreferenciasInvalido) || e.emisor.llamadas != 0 || repo.llamadas != 0 {
		t.Fatalf("clave ajena alcanzó autoridad/escritura: %v", err)
	}
}

func TestModalidadesHuellaLigaSeleccionYFuente(t *testing.T) {
	var huellas []string
	for _, caso := range []struct {
		seleccion []string
		version   int
	}{
		{[]string{"vacante"}, 1},
		{[]string{"sustitucion"}, 1},
		{[]string{"vacante"}, 2},
	} {
		e, s, catalogo, _ := entornoModalidadesPrueba(t)
		catalogo.disponibles[0] = modalidadPrueba("vacante", caso.version)
		_, err := s.GuardarModalidades(t.Context(), e.orden, caso.seleccion, 0, "clave-modalidad-3")
		exigir(t, err)
		d, err := e.concesiones.orden.Datos()
		exigir(t, err)
		n, err := d.Solicitud.Datos()
		exigir(t, err)
		huellas = append(huellas, n.Recurso.Atributos["material_sha256"])
	}
	if huellas[0] == huellas[1] || huellas[0] == huellas[2] {
		t.Fatal("la huella no cambia con la selección o la fuente")
	}
}

func TestModalidadesRechazanReciboFuturo(t *testing.T) {
	e, s, _, repo := entornoModalidadesPrueba(t)
	reloj := s.autoridad.reloj.(*relojAutorizacionServicioPrueba)
	repo.registradoEn = e.ahora.Add(3 * time.Second)
	repo.despues = func() { reloj.ahora = e.ahora.Add(2 * time.Second) }
	_, err := s.GuardarModalidades(t.Context(), e.orden, nil, 0, "clave-modalidad-4")
	if !errors.Is(err, bolsa.ErrPreferenciasNoDisponibles) || repo.llamadas != 1 {
		t.Fatalf("se aceptó recibo futuro: %v", err)
	}
}

func TestModalidadesConservanReciboEnReplay(t *testing.T) {
	e, s, _, repo := entornoModalidadesPrueba(t)
	seleccion := []string{"vacante"}
	primero, err := s.GuardarModalidades(t.Context(), e.orden, seleccion, 2, "clave-modalidad-replay")
	if err != nil || primero.Reutilizado || repo.llamadas != 1 {
		t.Fatalf("primera escritura rechazada: %v", err)
	}
	repo.reutilizado = true
	repetido, err := s.GuardarModalidades(t.Context(), e.orden, seleccion, 2, "clave-modalidad-replay")
	if err != nil || !repetido.Reutilizado || repo.llamadas != 2 ||
		repetido.ReciboRef != primero.ReciboRef || repetido.Version != primero.Version ||
		!repetido.RegistradoEn.Equal(primero.RegistradoEn) ||
		!slices.Equal(repetido.Modalidades, primero.Modalidades) {
		t.Fatalf("replay no conservó recibo: %v", err)
	}
}

func TestModalidadesInternasRechazanSesionExterna(t *testing.T) {
	e, externo, catalogo, repo := entornoModalidadesPrueba(t)
	s, err := NuevoServicioPreferenciasModalidadesInterna(repo, catalogo, e.autorizador,
		proveedorPreferenciasPrueba{e.emisor}, externo.autoridad.reloj)
	exigir(t, err)
	_, err = s.GuardarModalidades(t.Context(), e.orden, nil, 0, "clave-modalidad-5")
	if !errors.Is(err, domain.ErrAutorizacionDenegada) || catalogo.llamadas != 0 || repo.llamadas != 0 {
		t.Fatalf("sesión externa alcanzó catálogo o escritura interna: %v", err)
	}
}
