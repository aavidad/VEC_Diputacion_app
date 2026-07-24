package domain

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

type resolutorContextoRegistradoV2Prueba struct {
	resultado    ResultadoContextoActorRegistradoV2
	err          error
	invocaciones int
	despues      func()
}

type revalidadorAutenticacionV2Prueba struct {
	resultado    AutenticacionRevalidadaV1
	err          error
	invocaciones int
	despues      func()
}

func (r *revalidadorAutenticacionV2Prueba) RevalidarAutenticacionActorV1(
	_ context.Context,
	_ SolicitudRevalidacionAutenticacionActorV1,
) (AutenticacionRevalidadaV1, error) {
	r.invocaciones++
	if r.despues != nil {
		r.despues()
	}
	return r.resultado, r.err
}

func (r *resolutorContextoRegistradoV2Prueba) ResolverContextoActorRegistradoV2(
	_ context.Context,
	_ SolicitudContextoActor,
) (ResultadoContextoActorRegistradoV2, error) {
	r.invocaciones++
	if r.despues != nil {
		r.despues()
	}
	return r.resultado, r.err
}

type relojVinculoV2Prueba struct{ ahora time.Time }

func (r *relojVinculoV2Prueba) Ahora() time.Time { return r.ahora }

func TestVinculoAutenticacionActorV2FabricaSoloDesdeDosAutoridades(t *testing.T) {
	ahora := instanteVinculoAutenticacionActorV2Prueba()
	resultado := resultadoContextoActorRegistradoV2Prueba(t, ahora)
	autenticacion := autenticacionRevalidadaVinculoPrueba(ahora)
	revalidador := &revalidadorAutenticacionV2Prueba{resultado: autenticacion}
	resolutor := &resolutorContextoRegistradoV2Prueba{resultado: resultado}
	vinculo, err := CrearVinculoAutenticacionActorV2(
		context.Background(), revalidador, solicitudRevalidacionVinculoPrueba(autenticacion),
		resolutor, solicitudContextoVinculoV2Prueba(resultado), &relojVinculoV2Prueba{ahora: ahora},
	)
	if err != nil || vinculo.ValidarPara(resultado) != nil || !vinculo.VigenteEn(ahora, resultado) {
		t.Fatalf("vinculo V2 valido rechazado: %#v, %v", vinculo, err)
	}
	if revalidador.invocaciones != 1 || resolutor.invocaciones != 1 {
		t.Fatalf("autoridades no invocadas exactamente una vez: auth=%d contexto=%d",
			revalidador.invocaciones, resolutor.invocaciones)
	}
	datos, err := vinculo.Datos()
	if err != nil || datos.Esquema != EsquemaVinculoAutenticacionActorV2 ||
		datos.BloqueVersion != VersionVinculoAutenticacionActorV2 ||
		datos.RegistroContextoRef != resultado.RegistroContextoRef ||
		datos.ContextoActorEsquema != EsquemaRepresentacionContextoActorV2 ||
		datos.ContextoActorCuentaVersion != resultado.Contexto.Instantanea.CuentaVersion ||
		datos.ContextoActorHuellaSHA256 != resultado.HuellaSHA256 ||
		datos.ManifiestoProcedenciaHuellaSHA256 != resultado.ManifiestoProcedenciaHuellaSHA256 ||
		datos.AutoridadEfectiva != AutoridadProcedenciaContextoActorMaestraAcreditadaV1 {
		t.Fatalf("evidencia V2 incompleta: %+v, %v", datos, err)
	}

	tipo := reflect.TypeOf(datos)
	for i := 0; i < tipo.NumField(); i++ {
		nombre := tipo.Field(i).Name
		if nombre == "OperacionRef" || nombre == "DNI" || nombre == "Nombre" ||
			nombre == "Email" || nombre == "Roles" || nombre == "Permissions" || nombre == "Attributes" {
			t.Fatalf("campo prohibido expuesto: %s", nombre)
		}
	}
	contenido, err := json.Marshal(vinculo)
	if err != nil || bytes.Contains(contenido, []byte("operacion_ref")) ||
		!bytes.Contains(contenido, []byte(`"registro_contexto_ref"`)) {
		t.Fatalf("serializacion insegura: %s, %v", contenido, err)
	}
	var reconstruido VinculoAutenticacionActorV2
	if err := json.Unmarshal(contenido, &reconstruido); !errors.Is(err, ErrReconstruccionVinculoAutenticacionActorProhibida) ||
		reconstruido.Validar() == nil {
		t.Fatalf("codec de entrada reconstruyo capacidad: %v", err)
	}
	if (VinculoAutenticacionActorV2{}).Validar() == nil ||
		(VinculoAutenticacionActorV2{}).CoincideExactamenteCon(VinculoAutenticacionActorV2{}) {
		t.Fatal("valor cero participo como capacidad valida")
	}
}

func TestVinculoAutenticacionActorV2CierraCodecsAlternativosSinMutarCapacidad(t *testing.T) {
	vinculo, _, _ := vinculoAutenticacionActorV2Prueba(
		t, instanteVinculoAutenticacionActorV2Prueba(),
	)
	original, err := vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}

	codificadores := map[string]func() error{
		"texto":   func() error { _, err := vinculo.MarshalText(); return err },
		"binario": func() error { _, err := vinculo.MarshalBinary(); return err },
		"gob":     func() error { _, err := vinculo.GobEncode(); return err },
		"cbor":    func() error { _, err := vinculo.MarshalCBOR(); return err },
		"yaml":    func() error { _, err := vinculo.MarshalYAML(); return err },
		"xml": func() error {
			return vinculo.MarshalXML(xml.NewEncoder(&bytes.Buffer{}), xml.StartElement{})
		},
	}
	for nombre, codificar := range codificadores {
		t.Run("codificar "+nombre, func(t *testing.T) {
			if err := codificar(); !errors.Is(err, ErrSerializacionAlternativaVinculoAutenticacionActorV2Prohibida) {
				t.Fatalf("codec alternativo aceptado: %v", err)
			}
		})
	}

	decodificadores := map[string]func() error{
		"json":    func() error { return vinculo.UnmarshalJSON([]byte(`{}`)) },
		"texto":   func() error { return vinculo.UnmarshalText([]byte("x")) },
		"binario": func() error { return vinculo.UnmarshalBinary([]byte("x")) },
		"gob":     func() error { return vinculo.GobDecode([]byte("x")) },
		"cbor":    func() error { return vinculo.UnmarshalCBOR([]byte("x")) },
		"yaml":    func() error { return vinculo.UnmarshalYAML(func(any) error { return nil }) },
		"xml": func() error {
			return vinculo.UnmarshalXML(xml.NewDecoder(bytes.NewBufferString("<x/>")), xml.StartElement{})
		},
	}
	for nombre, decodificar := range decodificadores {
		t.Run("decodificar "+nombre, func(t *testing.T) {
			if err := decodificar(); !errors.Is(err, ErrReconstruccionVinculoAutenticacionActorProhibida) {
				t.Fatalf("codec de entrada aceptado: %v", err)
			}
			despues, err := vinculo.Datos()
			if err != nil || despues != original {
				t.Fatalf("codec de entrada altero la capacidad: %+v, %v", despues, err)
			}
		})
	}
}

func TestDatosVinculoAutenticacionActorV2RechazaCadaCampoCritico(t *testing.T) {
	vinculo, _, _ := vinculoAutenticacionActorV2Prueba(t, instanteVinculoAutenticacionActorV2Prueba())
	base, err := vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre string
		mutar  func(*DatosVinculoAutenticacionActorV2)
	}{
		{"esquema", func(d *DatosVinculoAutenticacionActorV2) { d.Esquema = "" }},
		{"bloque", func(d *DatosVinculoAutenticacionActorV2) { d.BloqueVersion = 1 }},
		{"autenticacion", func(d *DatosVinculoAutenticacionActorV2) { d.AutenticacionRef = "" }},
		{"huella autenticacion", func(d *DatosVinculoAutenticacionActorV2) { d.AutenticacionHuellaSHA256 = "" }},
		{"asercion", func(d *DatosVinculoAutenticacionActorV2) { d.AsercionRef = "" }},
		{"sesion", func(d *DatosVinculoAutenticacionActorV2) { d.SesionRef = "" }},
		{"control", func(d *DatosVinculoAutenticacionActorV2) { d.ControlSesionRef = "" }},
		{"revision control", func(d *DatosVinculoAutenticacionActorV2) { d.ControlSesionRevision = 0 }},
		{"huella control", func(d *DatosVinculoAutenticacionActorV2) { d.ControlSesionHuellaSHA256 = "" }},
		{"cuenta", func(d *DatosVinculoAutenticacionActorV2) { d.CuentaRef = "" }},
		{"cuenta ordinaria", func(d *DatosVinculoAutenticacionActorV2) { d.CuentaOrdinariaRef = "" }},
		{"principal", func(d *DatosVinculoAutenticacionActorV2) { d.PrincipalID = "" }},
		{"perfil", func(d *DatosVinculoAutenticacionActorV2) { d.PerfilActivoRef = "" }},
		{"privilegio incoherente", func(d *DatosVinculoAutenticacionActorV2) { d.CuentaPrivilegiada = true }},
		{"superficie", func(d *DatosVinculoAutenticacionActorV2) { d.Superficie = "" }},
		{"metodo", func(d *DatosVinculoAutenticacionActorV2) { d.MetodoObservado = "" }},
		{"garantia", func(d *DatosVinculoAutenticacionActorV2) { d.GarantiaObservada = "" }},
		{"politica", func(d *DatosVinculoAutenticacionActorV2) { d.PoliticaGarantiaRef = "" }},
		{"huella politica", func(d *DatosVinculoAutenticacionActorV2) { d.PoliticaGarantiaHuellaSHA256 = "" }},
		{"verificada", func(d *DatosVinculoAutenticacionActorV2) { d.AutenticacionVerificadaEn = time.Time{} }},
		{"emitida", func(d *DatosVinculoAutenticacionActorV2) { d.SesionEmitidaEn = time.Time{} }},
		{"valida hasta", func(d *DatosVinculoAutenticacionActorV2) { d.SesionValidaHasta = time.Time{} }},
		{"revalidada", func(d *DatosVinculoAutenticacionActorV2) { d.SesionRevalidadaEn = time.Time{} }},
		{"recibo", func(d *DatosVinculoAutenticacionActorV2) { d.RegistroContextoRef = "" }},
		{"esquema contexto", func(d *DatosVinculoAutenticacionActorV2) { d.ContextoActorEsquema = esquemaHuellaContextoActorV1 }},
		{"contexto", func(d *DatosVinculoAutenticacionActorV2) { d.ContextoActorRef = "" }},
		{"version contexto", func(d *DatosVinculoAutenticacionActorV2) { d.ContextoActorVersion = 0 }},
		{"version cuenta", func(d *DatosVinculoAutenticacionActorV2) { d.ContextoActorCuentaVersion = 0 }},
		{"huella contexto", func(d *DatosVinculoAutenticacionActorV2) { d.ContextoActorHuellaSHA256 = "" }},
		{"huella manifiesto", func(d *DatosVinculoAutenticacionActorV2) { d.ManifiestoProcedenciaHuellaSHA256 = "" }},
		{"autoridad", func(d *DatosVinculoAutenticacionActorV2) {
			d.AutoridadEfectiva = AutoridadProcedenciaContextoActorNoAutoritativaV1
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			candidata := base
			caso.mutar(&candidata)
			if candidata.Validar() == nil {
				t.Fatal("campo critico adulterado aceptado")
			}
		})
	}
}

func TestResultadoContextoActorRegistradoV2LigaCanonHuellasYAutoridad(t *testing.T) {
	base := resultadoContextoActorRegistradoV2Prueba(t, instanteVinculoAutenticacionActorV2Prueba())
	casos := []struct {
		nombre string
		mutar  func(*ResultadoContextoActorRegistradoV2)
	}{
		{"rca", func(r *ResultadoContextoActorRegistradoV2) { r.RegistroContextoRef = "rca_corto" }},
		{"cuenta version cero", func(r *ResultadoContextoActorRegistradoV2) { r.Contexto.Instantanea.CuentaVersion = 0 }},
		{"canon", func(r *ResultadoContextoActorRegistradoV2) { r.RepresentacionCanonica[0] ^= 1 }},
		{"huella V1", func(r *ResultadoContextoActorRegistradoV2) { r.HuellaSHA256 = strings.Repeat("0", 64) }},
		{"manifiesto", func(r *ResultadoContextoActorRegistradoV2) { r.ManifiestoProcedenciaCanonico[0] ^= 1 }},
		{"huella manifiesto", func(r *ResultadoContextoActorRegistradoV2) {
			r.ManifiestoProcedenciaHuellaSHA256 = strings.Repeat("0", 64)
		}},
		{"autoridad", func(r *ResultadoContextoActorRegistradoV2) {
			r.AutoridadEfectiva = AutoridadProcedenciaContextoActorNoAutoritativaV1
		}},
		{"instante", func(r *ResultadoContextoActorRegistradoV2) {
			r.ResueltoEnAutoritativo = r.ResueltoEnAutoritativo.Add(time.Microsecond)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			candidata, err := base.Clonar()
			if err != nil {
				t.Fatal(err)
			}
			caso.mutar(&candidata)
			if candidata.Validar() == nil {
				t.Fatal("recibo adulterado aceptado")
			}
		})
	}

	downgrade, err := base.Contexto.RepresentacionCanonicaVinculadaV1()
	if err == nil || len(downgrade) != 0 {
		t.Fatal("contexto V2 produjo preimagen V1")
	}
}

func TestResultadoContextoActorRegistradoV2ClonaDefensivamente(t *testing.T) {
	original := resultadoContextoActorRegistradoV2Prueba(t, instanteVinculoAutenticacionActorV2Prueba())
	clon, err := original.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	original.RepresentacionCanonica[0] ^= 1
	original.ManifiestoProcedenciaCanonico[0] ^= 1
	original.Contexto.Instantanea.CuentaRef = "cta_otra234567890abcdefghijkl"
	if clon.Validar() != nil {
		t.Fatal("mutacion del original alcanzo el clon")
	}
	datosCanon := append([]byte(nil), clon.RepresentacionCanonica...)
	clon.RepresentacionCanonica[0] ^= 1
	if bytes.Equal(clon.RepresentacionCanonica, datosCanon) {
		t.Fatal("fixture no mutado")
	}
	if original.Validar() == nil {
		t.Fatal("original adulterado siguio valido")
	}
}

func TestRegistroContextoRefV2ExigeLongitudDurable(t *testing.T) {
	resultado := resultadoContextoActorRegistradoV2Prueba(
		t, instanteVinculoAutenticacionActorV2Prueba(),
	)
	resultado.RegistroContextoRef = "rca_" + strings.Repeat("r", 24)
	if resultado.Validar() != nil {
		t.Fatal("rca_ con sufijo durable de 24 fue rechazado")
	}
	corto := resultado
	corto.RegistroContextoRef = "rca_" + strings.Repeat("r", 23)
	if corto.Validar() == nil {
		t.Fatal("rca_ con sufijo de 23 fue aceptado")
	}
	maximo := resultado
	maximo.RegistroContextoRef = "rca_" + strings.Repeat("r", 128)
	if maximo.Validar() != nil {
		t.Fatal("rca_ con sufijo maximo de 128 fue rechazado")
	}
	excesivo := resultado
	excesivo.RegistroContextoRef = "rca_" + strings.Repeat("r", 129)
	if excesivo.Validar() == nil {
		t.Fatal("rca_ con sufijo de 129 fue aceptado")
	}

	vinculo, _, _ := vinculoAutenticacionActorV2Prueba(
		t, instanteVinculoAutenticacionActorV2Prueba(),
	)
	datos, err := vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	datos.RegistroContextoRef = "rca_" + strings.Repeat("r", 24)
	if datos.Validar() != nil {
		t.Fatal("datos V2 rechazaron rca_ durable de 24")
	}
	datos.RegistroContextoRef = "rca_" + strings.Repeat("r", 23)
	if datos.Validar() == nil {
		t.Fatal("datos V2 aceptaron rca_ de 23")
	}
}
