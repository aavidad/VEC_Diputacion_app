package gobiernoconvocatorias

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type confirmadorConVinculoPrueba struct {
	*confirmadorPrueba
	vinculo VinculoVerificadorReciboBorrador
}

func (c *confirmadorConVinculoPrueba) VinculoVerificadorReciboBorrador() VinculoVerificadorReciboBorrador {
	if c == nil {
		return VinculoVerificadorReciboBorrador{}
	}
	return c.vinculo
}

type verificadorConVinculoPrueba struct {
	*verificadorReciboPrueba
	identidad IdentidadAutoridadBorrador
	vinculo   VinculoVerificadorReciboBorrador
}

func (v *verificadorConVinculoPrueba) IdentidadAutoridadBorrador() IdentidadAutoridadBorrador {
	if v == nil {
		return IdentidadAutoridadBorrador{}
	}
	return v.identidad
}

func (v *verificadorConVinculoPrueba) VinculoVerificadorReciboBorrador() VinculoVerificadorReciboBorrador {
	if v == nil {
		return VinculoVerificadorReciboBorrador{}
	}
	return v.vinculo
}

func identidadVinculoVerificadorPrueba(sufijo string) IdentidadAutoridadBorrador {
	identidad, _ := NuevaIdentidadAutoridadBorrador(
		"proveedor-"+sufijo, "instancia-"+sufijo,
		"credencial-"+sufijo, "rol-"+sufijo,
	)
	return identidad
}

func vinculoVerificadorPrueba(
	t *testing.T,
	persistencia, criptografia IdentidadAutoridadBorrador,
) VinculoVerificadorReciboBorrador {
	t.Helper()
	vinculo, err := NuevoVinculoVerificadorReciboBorrador(persistencia, criptografia)
	if err != nil {
		t.Fatal(err)
	}
	return vinculo
}

func reconstruirServicioConVerificacionPrueba(
	t *testing.T,
	confirmador ConfirmadorAtomicoBorrador,
	verificador VerificadorReciboBorrador,
) error {
	t.Helper()
	escenario := nuevoEscenario(t, confirmarBien, 1)
	s := escenario.servicio
	_, err := NuevoServicioBorradores(
		s.reloj, s.preparadorAlta, s.motivos, s.lector, s.comprometedor,
		s.derivador, s.autorizador, s.diario, s.sellador, s.politicasCifrado,
		s.perfilesCifrado, s.cifrador, confirmador, verificador, s.procedencia,
	)
	return err
}

func TestServicioBorradoresExigeElMismoVinculoIntegralDeVerificacion(t *testing.T) {
	confirmadorBase := &confirmadorPrueba{}
	persistenciaA := (&verificadorReciboPrueba{}).IdentidadAutoridadBorrador()
	criptografiaA := identidadVinculoVerificadorPrueba("cripto-a")
	vinculoA := vinculoVerificadorPrueba(t, persistenciaA, criptografiaA)
	confirmadorA := &confirmadorConVinculoPrueba{
		confirmadorPrueba: confirmadorBase, vinculo: vinculoA,
	}
	verificadorA := &verificadorConVinculoPrueba{
		verificadorReciboPrueba: &verificadorReciboPrueba{},
		identidad:               persistenciaA, vinculo: vinculoA,
	}
	if err := reconstruirServicioConVerificacionPrueba(t, confirmadorA, verificadorA); err != nil {
		t.Fatalf("composicion A/A valida rechazada: %v", err)
	}

	persistenciaB := identidadVinculoVerificadorPrueba("db-b")
	criptografiaB := identidadVinculoVerificadorPrueba("cripto-b")
	vinculoB := vinculoVerificadorPrueba(t, persistenciaB, criptografiaB)
	verificadorB := &verificadorConVinculoPrueba{
		verificadorReciboPrueba: &verificadorReciboPrueba{},
		identidad:               persistenciaB, vinculo: vinculoB,
	}
	if err := reconstruirServicioConVerificacionPrueba(t, confirmadorA, verificadorB); !errors.Is(err, ErrServicioBorradoresInvalido) {
		t.Fatalf("confirmador A cruzado con verificador B aceptado: %v", err)
	}

	vinculoMismaDBOtroCripto := vinculoVerificadorPrueba(t, persistenciaA, criptografiaB)
	verificadorMismaDBOtroCripto := &verificadorConVinculoPrueba{
		verificadorReciboPrueba: &verificadorReciboPrueba{},
		identidad:               persistenciaA, vinculo: vinculoMismaDBOtroCripto,
	}
	if err := reconstruirServicioConVerificacionPrueba(
		t, confirmadorA, verificadorMismaDBOtroCripto,
	); !errors.Is(err, ErrServicioBorradoresInvalido) {
		t.Fatalf("cambio oculto de autoridad criptografica aceptado: %v", err)
	}

	verificadorIdentidadAjena := &verificadorConVinculoPrueba{
		verificadorReciboPrueba: &verificadorReciboPrueba{},
		identidad:               persistenciaB, vinculo: vinculoA,
	}
	if err := reconstruirServicioConVerificacionPrueba(
		t, confirmadorA, verificadorIdentidadAjena,
	); !errors.Is(err, ErrServicioBorradoresInvalido) {
		t.Fatalf("identidad DB ajena al vinculo aceptada: %v", err)
	}

	vinculoCriptoIgualConfirmador := vinculoVerificadorPrueba(
		t, persistenciaA, confirmadorBase.IdentidadAutoridadBorrador(),
	)
	confirmadorSinSeparacion := &confirmadorConVinculoPrueba{
		confirmadorPrueba: confirmadorBase, vinculo: vinculoCriptoIgualConfirmador,
	}
	verificadorSinSeparacion := &verificadorConVinculoPrueba{
		verificadorReciboPrueba: &verificadorReciboPrueba{},
		identidad:               persistenciaA, vinculo: vinculoCriptoIgualConfirmador,
	}
	if err := reconstruirServicioConVerificacionPrueba(
		t, confirmadorSinSeparacion, verificadorSinSeparacion,
	); !errors.Is(err, ErrServicioBorradoresInvalido) {
		t.Fatalf("confirmador y autoridad criptografica no separados aceptados: %v", err)
	}
}

func TestServicioBorradoresRechazaFronterasDeVerificacionNulasTipadas(t *testing.T) {
	var confirmadorNulo *confirmadorPrueba
	if err := reconstruirServicioConVerificacionPrueba(
		t, confirmadorNulo, &verificadorReciboPrueba{},
	); !errors.Is(err, ErrServicioBorradoresInvalido) {
		t.Fatalf("confirmador nulo tipado aceptado: %v", err)
	}

	var verificadorNulo *verificadorReciboPrueba
	if err := reconstruirServicioConVerificacionPrueba(
		t, &confirmadorPrueba{}, verificadorNulo,
	); !errors.Is(err, ErrServicioBorradoresInvalido) {
		t.Fatalf("verificador nulo tipado aceptado: %v", err)
	}
}

func TestVinculoVerificadorComprometeCadaCoordenadaSinExponerla(t *testing.T) {
	persistencia := identidadVinculoVerificadorPrueba("db-referencia")
	criptografia := identidadVinculoVerificadorPrueba("cripto-referencia")
	vinculo := vinculoVerificadorPrueba(t, persistencia, criptografia)
	referencia, err := vinculo.ReferenciaParaAcreditacion()
	if err != nil || len(referencia) != len(prefijoVinculoVerificadorReciboBorradorV1)+64 ||
		!strings.HasPrefix(referencia, prefijoVinculoVerificadorReciboBorradorV1) {
		t.Fatalf("referencia opaca invalida: %q err=%v", referencia, err)
	}
	for _, coordenada := range []string{
		persistencia.ProveedorRef, persistencia.InstanciaRef,
		persistencia.CredencialRef, persistencia.RolRef,
		criptografia.ProveedorRef, criptografia.InstanciaRef,
		criptografia.CredencialRef, criptografia.RolRef,
	} {
		if strings.Contains(referencia, coordenada) {
			t.Fatalf("la referencia expone una coordenada de autoridad: %q", coordenada)
		}
	}

	mutaciones := []struct {
		nombre string
		mutar  func(*IdentidadAutoridadBorrador, *IdentidadAutoridadBorrador)
	}{
		{"db proveedor", func(db, _ *IdentidadAutoridadBorrador) { db.ProveedorRef += "-otro" }},
		{"db instancia", func(db, _ *IdentidadAutoridadBorrador) { db.InstanciaRef += "-otra" }},
		{"db credencial", func(db, _ *IdentidadAutoridadBorrador) { db.CredencialRef += "-otra" }},
		{"db rol", func(db, _ *IdentidadAutoridadBorrador) { db.RolRef += "-otro" }},
		{"cripto proveedor", func(_, cripto *IdentidadAutoridadBorrador) { cripto.ProveedorRef += "-otro" }},
		{"cripto instancia", func(_, cripto *IdentidadAutoridadBorrador) { cripto.InstanciaRef += "-otra" }},
		{"cripto credencial", func(_, cripto *IdentidadAutoridadBorrador) { cripto.CredencialRef += "-otra" }},
		{"cripto rol", func(_, cripto *IdentidadAutoridadBorrador) { cripto.RolRef += "-otro" }},
	}
	for _, mutacion := range mutaciones {
		t.Run(mutacion.nombre, func(t *testing.T) {
			dbMutada, criptoMutada := persistencia, criptografia
			mutacion.mutar(&dbMutada, &criptoMutada)
			otro := vinculoVerificadorPrueba(t, dbMutada, criptoMutada)
			referenciaOtra, err := otro.ReferenciaParaAcreditacion()
			if err != nil || referenciaOtra == referencia {
				t.Fatalf("coordenada no comprometida: referencia=%q err=%v", referenciaOtra, err)
			}
		})
	}

	if _, err := json.Marshal(vinculo); !errors.Is(err, ErrSerializacionDiarioProhibida) {
		t.Fatalf("vinculo interno serializable por accidente: %v", err)
	}
	if _, err := (VinculoVerificadorReciboBorrador{}).ReferenciaParaAcreditacion(); !errors.Is(err, ErrServicioBorradoresInvalido) {
		t.Fatalf("vinculo cero aceptado: %v", err)
	}
}
